package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"github.com/francisco/fleet-monitor/internal/application"
	"github.com/francisco/fleet-monitor/internal/config"
	"github.com/francisco/fleet-monitor/internal/domain"
	htthandler "github.com/francisco/fleet-monitor/internal/infrastructure/http"
	"github.com/francisco/fleet-monitor/internal/infrastructure/jwt"
	"github.com/francisco/fleet-monitor/internal/infrastructure/persistence"
	"github.com/francisco/fleet-monitor/internal/infrastructure/persistence/sqlite"
	"github.com/francisco/fleet-monitor/internal/infrastructure/websocket"
)

func main() {
	_ = godotenv.Load()

	seedFlag := flag.Bool("seed", false, "seed the database with sample data and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	db, err := persistence.Open(cfg.DSN())
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := persistence.RunMigrations(db, "./migrations"); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	userRepo := sqlite.NewUserRepo(db)
	vehicleRepo := sqlite.NewVehicleRepo(db)
	sensorRepo := sqlite.NewSensorDataRepo(db)
	alertRepo := sqlite.NewAlertRepo(db)

	if *seedFlag {
		if err := seedDatabase(userRepo, vehicleRepo, cfg.BcryptCost); err != nil {
			slog.Error("failed to seed database", "error", err)
			os.Exit(1)
		}
		slog.Info("database seeded successfully")
		return
	}

	tokenService, err := jwt.NewTokenService(
		cfg.JWTSecret,
		time.Duration(cfg.AccessTokenTTLMinutes)*time.Minute,
		time.Duration(cfg.RefreshTokenTTLDays)*24*time.Hour,
		nil,
	)
	if err != nil {
		slog.Error("failed to create token service", "error", err)
		os.Exit(1)
	}

	authService := application.NewAuthService(
		userRepo,
		tokenService,
		bcrypt.CompareHashAndPassword,
		time.Duration(cfg.AccessTokenTTLMinutes)*time.Minute,
		time.Duration(cfg.RefreshTokenTTLDays)*24*time.Hour,
	)

	fuelService := application.NewFuelService(sensorRepo, alertRepo)

	hub := websocket.NewHub()
	go hub.Run()

	sensorService := application.NewSensorService(
		sensorRepo,
		vehicleRepo,
		fuelService,
	).WithBroadcaster(hub)

	vehicleService := application.NewVehicleService(
		vehicleRepo,
		sensorRepo,
		application.MaskDeviceID,
	)

	deps := htthandler.Dependencies{
		AuthService:    authService,
		VehicleService: vehicleService,
		SensorService:  sensorService,
		TokenService:   tokenService,
		AlertRepo:      alertRepo,
		Hub:            hub,
	}

	router := htthandler.NewRouter(cfg, deps)

	server := &http.Server{
		Addr:         cfg.ListenAddr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "addr", cfg.ListenAddr())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received, shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server shut down gracefully")
}

func seedDatabase(userRepo domain.UserRepository, vehicleRepo domain.VehicleRepository, bcryptCost int) error {
	adminPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	admin := &domain.User{
		Email:    "admin@example.com",
		Password: string(adminPassword),
		Role:     "admin",
	}
	if err := userRepo.Create(context.Background(), admin); err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}
	slog.Info("seeded admin user", "id", admin.ID, "email", admin.Email)

	userPassword, err := bcrypt.GenerateFromPassword([]byte("user123"), bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash user password: %w", err)
	}

	user := &domain.User{
		Email:    "user@example.com",
		Password: string(userPassword),
		Role:     "user",
	}
	if err := userRepo.Create(context.Background(), user); err != nil {
		return fmt.Errorf("failed to create regular user: %w", err)
	}
	slog.Info("seeded regular user", "id", user.ID, "email", user.Email)

	vehicles := []*domain.Vehicle{
		{DeviceID: "DEV-11111111-AAAA", Name: "Truck 01"},
		{DeviceID: "DEV-22222222-BBBB", Name: "Truck 02"},
		{DeviceID: "DEV-33333333-CCCC", Name: "Truck 03"},
	}

	for _, v := range vehicles {
		if err := vehicleRepo.Create(context.Background(), v); err != nil {
			return fmt.Errorf("failed to create vehicle %s: %w", v.DeviceID, err)
		}
		slog.Info("seeded vehicle", "id", v.ID, "device_id", v.DeviceID, "name", v.Name)
	}

	return nil
}