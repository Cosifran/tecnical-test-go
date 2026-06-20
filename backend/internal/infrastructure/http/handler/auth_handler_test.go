package handler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"github.com/francisco/fleet-monitor/internal/application"
	"github.com/francisco/fleet-monitor/internal/domain"
	fleethttp "github.com/francisco/fleet-monitor/internal/infrastructure/http"
	"github.com/francisco/fleet-monitor/internal/infrastructure/http/handler"
	"github.com/francisco/fleet-monitor/internal/infrastructure/jwt"
	"github.com/francisco/fleet-monitor/internal/infrastructure/persistence"
	sqlite "github.com/francisco/fleet-monitor/internal/infrastructure/persistence/sqlite"
)

// ============================================================
// Integration test setup
// ============================================================

const testJWTSecret = "integration-test-secret-must-be-at-least-32-bytes"

// testEnv holds all dependencies for an integration test.
type testEnv struct {
	db           *sql.DB
	authService  *application.AuthService
	tokenService *jwt.TokenService
	authHandler  *handler.AuthHandler
	server       *httptest.Server
}

// setupTestEnv creates a full integration test environment with:
//   - In-memory SQLite database with migrations
//   - Real JWT token service with known secret
//   - Real auth service with real bcrypt comparison
//   - httptest.Server wired to the auth handler
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Open in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Run migrations using the shared RunMigrations function so the
	// test schema matches production — including migration tracking.
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	migrationsDir := filepath.Join(dir, "..", "..", "..", "..", "migrations")
	if err := persistence.RunMigrations(db, migrationsDir); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Create repos
	userRepo := sqlite.NewUserRepo(db)

	// Create token service
	tokenService, err := jwt.NewTokenService(
		testJWTSecret,
		15*time.Minute,
		7*24*time.Hour,
		nil, // uses time.Now
	)
	if err != nil {
		t.Fatalf("failed to create token service: %v", err)
	}

	// Create auth service with real bcrypt comparison
	authService := application.NewAuthService(
		userRepo,
		tokenService,
		bcrypt.CompareHashAndPassword,
		15*time.Minute,
		7*24*time.Hour,
	)

	// Create auth handler
	authHandler := handler.NewAuthHandler(authService)

	// Create test router
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)

	// Wrap with logging middleware
	h := fleethttp.LoggingMiddleware(mux)

	server := httptest.NewServer(h)
	t.Cleanup(server.Close)

	return &testEnv{
		db:           db,
		authService:  authService,
		tokenService: tokenService,
		authHandler:  authHandler,
		server:       server,
	}
}

// seedTestUser creates a test user in the database with a known password.
func seedTestUser(t *testing.T, db *sql.DB, email, password, role string) *domain.User {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := &domain.User{
		Email:    email,
		Password: string(hashedPassword),
		Role:     role,
	}

	userRepo := sqlite.NewUserRepo(db)
	if err := userRepo.Create(t.Context(), user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

// ============================================================
// Login integration tests
// ============================================================

func TestLogin_Integration(t *testing.T) {
	env := setupTestEnv(t)
	_ = seedTestUser(t, env.db, "admin@example.com", "correctpassword", "admin")

	tests := []struct {
		name       string
		email      string
		password   string
		wantStatus int
		wantError  string // expected error code, empty if success
	}{
		{
			name:       "login with correct credentials",
			email:      "admin@example.com",
			password:   "correctpassword",
			wantStatus: http.StatusOK,
		},
		{
			name:       "login with wrong password",
			email:      "admin@example.com",
			password:   "wrongpassword",
			wantStatus: http.StatusUnauthorized,
			wantError:  "unauthorized",
		},
		{
			name:       "login with nonexistent email",
			email:      "nonexistent@example.com",
			password:   "password",
			wantStatus: http.StatusUnauthorized,
			wantError:  "unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{
				"email":    tt.email,
				"password": tt.password,
			})

			resp, err := http.Post(
				env.server.URL+"/api/v1/auth/login",
				"application/json",
				bytes.NewReader(body),
			)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status: got %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if tt.wantError != "" {
				if result["error"] != tt.wantError {
					t.Errorf("error code: got %v, want %v", result["error"], tt.wantError)
				}
			} else {
				// Successful login should return tokens
				if result["access_token"] == nil || result["access_token"] == "" {
					t.Error("expected non-empty access_token")
				}
				if result["refresh_token"] == nil || result["refresh_token"] == "" {
					t.Error("expected non-empty refresh_token")
				}
				if result["token_type"] != "Bearer" {
					t.Errorf("token_type: got %v, want Bearer", result["token_type"])
				}
			}
		})
	}
}

func TestLogin_MalformedRequest(t *testing.T) {
	env := setupTestEnv(t)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			body:       "{broken",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(
				env.server.URL+"/api/v1/auth/login",
				"application/json",
				bytes.NewReader([]byte(tt.body)),
			)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status: got %d, want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestLogin_ValidTokenWorks(t *testing.T) {
	// Full integration test: login → get token → validate token
	env := setupTestEnv(t)
	user := seedTestUser(t, env.db, "user@example.com", "mypassword", "user")
	_ = user // used via login

	// Step 1: Login to get tokens
	body, _ := json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": "mypassword",
	})

	resp, err := http.Post(
		env.server.URL+"/api/v1/auth/login",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed with status %d", resp.StatusCode)
	}

	var loginResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&loginResult); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	accessToken, ok := loginResult["access_token"].(string)
	if !ok || accessToken == "" {
		t.Fatal("expected non-empty access_token in login response")
	}

	// Step 2: Validate the access token with the token service
	claims, err := env.tokenService.Validate(accessToken)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}

	if claims.Subject == "" {
		t.Error("expected non-empty subject in token claims")
	}
	if claims.Role != "user" {
		t.Errorf("role in claims: got %q, want %q", claims.Role, "user")
	}
	if claims.Type != "access" {
		t.Errorf("type in claims: got %q, want %q", claims.Type, "access")
	}
}

// ============================================================
// Refresh integration tests
// ============================================================

func TestRefresh_Integration(t *testing.T) {
	env := setupTestEnv(t)
	_ = seedTestUser(t, env.db, "admin@example.com", "password123", "admin")

	// First, login to get a valid refresh token
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "password123",
	})

	loginResp, err := http.Post(
		env.server.URL+"/api/v1/auth/login",
		"application/json",
		bytes.NewReader(loginBody),
	)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer loginResp.Body.Close()

	var loginResult map[string]interface{}
	if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	refreshToken, _ := loginResult["refresh_token"].(string)
	if refreshToken == "" {
		t.Fatal("expected non-empty refresh_token from login")
	}

	// Now test the refresh endpoint
	t.Run("valid refresh token returns new tokens", func(t *testing.T) {
		refreshBody, _ := json.Marshal(map[string]string{
			"refresh_token": refreshToken,
		})

		resp, err := http.Post(
			env.server.URL+"/api/v1/auth/refresh",
			"application/json",
			bytes.NewReader(refreshBody),
		)
		if err != nil {
			t.Fatalf("refresh request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errBody map[string]string
			json.NewDecoder(resp.Body).Decode(&errBody)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, errBody)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to parse refresh response: %v", err)
		}

		if result["access_token"] == nil || result["access_token"] == "" {
			t.Error("expected non-empty access_token from refresh")
		}
		if result["refresh_token"] == nil || result["refresh_token"] == "" {
			t.Error("expected non-empty refresh_token from refresh (rotation)")
		}
		if result["token_type"] != "Bearer" {
			t.Errorf("token_type: got %v, want Bearer", result["token_type"])
		}
	})

	t.Run("invalid refresh token returns 401", func(t *testing.T) {
		refreshBody, _ := json.Marshal(map[string]string{
			"refresh_token": "invalid.token.here",
		})

		resp, err := http.Post(
			env.server.URL+"/api/v1/auth/refresh",
			"application/json",
			bytes.NewReader(refreshBody),
		)
		if err != nil {
			t.Fatalf("refresh request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("malformed request returns 400", func(t *testing.T) {
		resp, err := http.Post(
			env.server.URL+"/api/v1/auth/refresh",
			"application/json",
			bytes.NewReader([]byte("{invalid")),
		)
		if err != nil {
			t.Fatalf("refresh request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for malformed request, got %d", resp.StatusCode)
		}
	})
}

// ============================================================
// Full flow integration test
// ============================================================

func TestAuthHandler_Integration_FullFlow(t *testing.T) {
	// This test exercises the full auth flow:
	// 1. Register a user (directly in DB)
	// 2. Login → get tokens
	// 3. Validate access token
	// 4. Validate refresh token type
	// 5. Use refresh token to get new tokens
	// 6. Verify new tokens are valid

	env := setupTestEnv(t)
	user := seedTestUser(t, env.db, fmt.Sprintf("flow-%d@example.com", time.Now().UnixNano()), "testpassword", "admin")

	t.Logf("Seeded user: ID=%s, Email=%s, Role=%s", user.ID, user.Email, user.Role)

	// Step 2: Login
	loginBody, _ := json.Marshal(map[string]string{
		"email":    user.Email,
		"password": "testpassword",
	})

	loginResp, err := http.Post(
		env.server.URL+"/api/v1/auth/login",
		"application/json",
		bytes.NewReader(loginBody),
	)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login failed with status %d", loginResp.StatusCode)
	}

	var loginResult application.LoginResult
	if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	// Step 3: Validate access token
	accessClaims, err := env.tokenService.Validate(loginResult.AccessToken)
	if err != nil {
		t.Fatalf("access token validation failed: %v", err)
	}
	if accessClaims.Type != "access" {
		t.Errorf("access token type: got %q, want %q", accessClaims.Type, "access")
	}
	if accessClaims.Role != "admin" {
		t.Errorf("access token role: got %q, want %q", accessClaims.Role, "admin")
	}

	// Step 4: Validate refresh token
	refreshClaims, err := env.tokenService.Validate(loginResult.RefreshToken)
	if err != nil {
		t.Fatalf("refresh token validation failed: %v", err)
	}
	if refreshClaims.Type != "refresh" {
		t.Errorf("refresh token type: got %q, want %q", refreshClaims.Type, "refresh")
	}

	// Step 5: Refresh tokens
	refreshBody, _ := json.Marshal(map[string]string{
		"refresh_token": loginResult.RefreshToken,
	})

	refreshResp, err := http.Post(
		env.server.URL+"/api/v1/auth/refresh",
		"application/json",
		bytes.NewReader(refreshBody),
	)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	defer refreshResp.Body.Close()

	if refreshResp.StatusCode != http.StatusOK {
		t.Fatalf("refresh failed with status %d", refreshResp.StatusCode)
	}

	var refreshResult application.RefreshResult
	if err := json.NewDecoder(refreshResp.Body).Decode(&refreshResult); err != nil {
		t.Fatalf("failed to parse refresh response: %v", err)
	}

	// Step 6: Verify new tokens are valid
	newAccessClaims, err := env.tokenService.Validate(refreshResult.AccessToken)
	if err != nil {
		t.Fatalf("new access token validation failed: %v", err)
	}
	if newAccessClaims.Type != "access" {
		t.Errorf("new access token type: got %q, want %q", newAccessClaims.Type, "access")
	}

	t.Logf("Full auth flow completed: user=%s, role=%s", newAccessClaims.Subject, newAccessClaims.Role)
}