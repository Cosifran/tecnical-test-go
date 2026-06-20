package http

import (
	"net/http"

	"github.com/francisco/fleet-monitor/internal/application"
	"github.com/francisco/fleet-monitor/internal/config"
	"github.com/francisco/fleet-monitor/internal/domain"
	"github.com/francisco/fleet-monitor/internal/infrastructure/http/handler"
	"github.com/francisco/fleet-monitor/internal/infrastructure/jwt"
	"github.com/francisco/fleet-monitor/internal/infrastructure/websocket"
)

type Dependencies struct {
	AuthService    *application.AuthService
	VehicleService *application.VehicleService
	SensorService  *application.SensorService
	TokenService   *jwt.TokenService
	AlertRepo      domain.AlertRepository
	Hub            *websocket.Hub
}

func NewRouter(cfg *config.Config, deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	authHandler := handler.NewAuthHandler(deps.AuthService)
	vehicleHandler := handler.NewVehicleHandler(deps.VehicleService)
	sensorHandler := handler.NewSensorHandler(deps.SensorService)
	alertHandler := handler.NewAlertHandler(deps.AlertRepo)

	// Public routes
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)

	// Protected routes
	mux.Handle("GET /api/v1/vehicles",
		AuthMiddleware(deps.TokenService)(
			RBACMiddleware("admin", "user")(
				http.HandlerFunc(vehicleHandler.List),
			),
		),
	)

	mux.Handle("GET /api/v1/vehicles/{id}/history",
		AuthMiddleware(deps.TokenService)(
			RBACMiddleware("admin", "user")(
				http.HandlerFunc(vehicleHandler.History),
			),
		),
	)

	mux.Handle("POST /api/v1/sensors/data",
		AuthMiddleware(deps.TokenService)(
			RBACMiddleware("admin")(
				http.HandlerFunc(sensorHandler.Ingest),
			),
		),
	)

	mux.Handle("GET /api/v1/alerts",
		AuthMiddleware(deps.TokenService)(
			RBACMiddleware("admin")(
				http.HandlerFunc(alertHandler.List),
			),
		),
	)

	// WebSocket — auth via ?token= query parameter
	mux.HandleFunc("GET /api/v1/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWS(deps.Hub, w, r, deps.TokenService)
	})

	return LoggingMiddleware(CORSMiddleware(mux))
}