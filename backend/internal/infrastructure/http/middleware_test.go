package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
	"github.com/francisco/fleet-monitor/internal/infrastructure/http/httputil"
	"github.com/francisco/fleet-monitor/internal/infrastructure/jwt"
)

// ============================================================
// Test helpers
// ============================================================

// newTestTokenService creates a TokenService with a known secret for testing.
func newTestTokenService() *jwt.TokenService {
	ts, err := jwt.NewTokenService(
		"test-secret-key-that-is-at-least-32-bytes-long!",
		15*time.Minute,
		7*24*time.Hour,
		nil, // uses time.Now
	)
	if err != nil {
		panic("failed to create test token service: " + err.Error())
	}
	return ts
}

// newFutureTokenService creates a TokenService whose clock is 24h in the future.
// When used for validation, tokens that expire within 24h appear expired.
// This lets us test the ErrTokenExpired path without waiting for real time to pass.
func newFutureTokenService() *jwt.TokenService {
	ts, err := jwt.NewTokenService(
		"test-secret-key-that-is-at-least-32-bytes-long!",
		15*time.Minute,
		7*24*time.Hour,
		func() time.Time { return time.Now().Add(24 * time.Hour) }, // clock 24h ahead
	)
	if err != nil {
		panic("failed to create future token service: " + err.Error())
	}
	return ts
}

// generateTestToken creates a valid JWT token for testing.
func generateTestToken(ts *jwt.TokenService, sub, email, role, tokenType string) string {
	claims := domain.Claims{
		Subject: sub,
		Email:   email,
		Role:    role,
		Type:    tokenType,
	}
	token, err := ts.Generate(claims)
	if err != nil {
		panic("failed to generate test token: " + err.Error())
	}
	return token
}

// okHandler is a test handler that writes a 200 OK response with context values.
var okHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
	userID := httputil.UserIDFromContext(r.Context())
	email := httputil.EmailFromContext(r.Context())
	role := httputil.RoleFromContext(r.Context())

	resp := map[string]string{
		"status":  "ok",
		"user_id": userID,
		"email":   email,
		"role":    role,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ============================================================
// AuthMiddleware tests
// ============================================================

func TestAuthMiddleware(t *testing.T) {
	ts := newTestTokenService()
	validToken := generateTestToken(ts, "user-1", "admin@example.com", "admin", "access")

	// To test expired tokens: generate a token with the normal service (exp = now+15min),
	// then validate it with a service whose clock is 24h in the future (so now > exp).
	// This makes the token appear expired during validation.
	futureTS := newFutureTokenService()
	expiredToken := generateTestToken(ts, "user-2", "user@example.com", "user", "access")

	tests := []struct {
		name         string
		token        string // "Bearer <token>" header value, or empty for no header
		queryToken   string // ?token= query param
		tokenService *jwt.TokenService // which TokenService to use for validation (defaults to ts)
		wantStatus   int
		wantError    string // expected error code in response body
		wantUserID   string // expected user_id in context (if request reaches handler)
		wantRole     string // expected role in context
	}{
		{
			name:       "valid Bearer token",
			token:      "Bearer " + validToken,
			wantStatus: http.StatusOK,
			wantUserID: "user-1",
			wantRole:   "admin",
		},
		{
			name:       "valid token via query param",
			queryToken: validToken,
			wantStatus: http.StatusOK,
			wantUserID: "user-1",
			wantRole:   "admin",
		},
		{
			name:       "missing token",
			token:      "",
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid_token",
		},
		{
			name:       "invalid token",
			token:      "Bearer this.is.not.a.valid.token",
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid_token",
		},
		{
			name:         "expired token",
			token:        "Bearer " + expiredToken,
			tokenService: futureTS, // validate with future clock → token appears expired
			wantStatus:   http.StatusUnauthorized,
			wantError:    "token_expired",
		},
		{
			name:       "malformed Authorization header (no Bearer prefix)",
			token:      validToken, // missing "Bearer " prefix
			wantStatus: http.StatusUnauthorized,
			wantError:  "invalid_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the test-specific token service if provided, otherwise default to ts
			validator := ts
			if tt.tokenService != nil {
				validator = tt.tokenService
			}
			handler := AuthMiddleware(validator)(okHandler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}
			if tt.queryToken != "" {
				req.URL.RawQuery = "token=" + tt.queryToken
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantError != "" {
				var errResp map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("failed to parse response body: %v", err)
				}
				if errResp["error"] != tt.wantError {
					t.Errorf("error code: got %v, want %v", errResp["error"], tt.wantError)
				}
			}

			if tt.wantUserID != "" {
				var resp map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response body: %v", err)
				}
				if resp["user_id"] != tt.wantUserID {
					t.Errorf("user_id in context: got %q, want %q", resp["user_id"], tt.wantUserID)
				}
				if resp["role"] != tt.wantRole {
					t.Errorf("role in context: got %q, want %q", resp["role"], tt.wantRole)
				}
			}
		})
	}
}

// ============================================================
// RBACMiddleware tests
// ============================================================

func TestRBACMiddleware(t *testing.T) {
	tests := []struct {
		name         string
		role         string // role injected into context
		allowedRoles []string
		wantStatus   int
		wantError    string // expected error code, empty if request should succeed
	}{
		{
			name:         "admin accessing admin-only endpoint",
			role:         "admin",
			allowedRoles: []string{"admin"},
			wantStatus:   http.StatusOK,
		},
		{
			name:         "user accessing admin-only endpoint → 403",
			role:         "user",
			allowedRoles: []string{"admin"},
			wantStatus:   http.StatusForbidden,
			wantError:    "forbidden",
		},
		{
			name:         "admin accessing admin+user endpoint",
			role:         "admin",
			allowedRoles: []string{"admin", "user"},
			wantStatus:   http.StatusOK,
		},
		{
			name:         "user accessing admin+user endpoint",
			role:         "user",
			allowedRoles: []string{"admin", "user"},
			wantStatus:   http.StatusOK,
		},
		{
			name:         "unknown role accessing admin-only endpoint → 403",
			role:         "guest",
			allowedRoles: []string{"admin"},
			wantStatus:   http.StatusForbidden,
			wantError:    "forbidden",
		},
		{
			name:         "empty role accessing admin+user endpoint → 403",
			role:         "",
			allowedRoles: []string{"admin", "user"},
			wantStatus:   http.StatusForbidden,
			wantError:    "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RBACMiddleware(tt.allowedRoles...)(okHandler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			ctx := context.WithValue(req.Context(), httputil.ContextKeyRole, tt.role)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantError != "" {
				var errResp map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("failed to parse response body: %v", err)
				}
				if errResp["error"] != tt.wantError {
					t.Errorf("error code: got %v, want %v", errResp["error"], tt.wantError)
				}
			}
		})
	}
}

// ============================================================
// LoggingMiddleware tests
// ============================================================

func TestLoggingMiddleware(t *testing.T) {
	handler := LoggingMiddleware(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("response body: got %v, want status=ok", resp)
	}
}

// ============================================================
// Token extraction tests
// ============================================================

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		queryParam string
		wantToken  string
	}{
		{
			name:       "Bearer token in Authorization header",
			authHeader: "Bearer my-token-123",
			wantToken:  "my-token-123",
		},
		{
			name:       "Token in query parameter",
			queryParam: "my-token-456",
			wantToken:  "my-token-456",
		},
		{
			name:       "Authorization header takes precedence over query param",
			authHeader: "Bearer header-token",
			queryParam: "query-token",
			wantToken:  "header-token",
		},
		{
			name:       "No token anywhere",
			authHeader: "",
			queryParam: "",
			wantToken:  "",
		},
		{
			name:       "Malformed Authorization header (no Bearer prefix)",
			authHeader: "Basic dXNlcjpwYXNz",
			wantToken:  "",
		},
		{
			name:       "Empty Bearer token",
			authHeader: "Bearer ",
			wantToken:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.queryParam != "" {
				q := req.URL.Query()
				q.Set("token", tt.queryParam)
				req.URL.RawQuery = q.Encode()
			}

			got := extractToken(req)
			if got != tt.wantToken {
				t.Errorf("extractToken(): got %q, want %q", got, tt.wantToken)
			}
		})
	}
}

// ============================================================
// DecodeJSON tests
// ============================================================

func TestDecodeJSON(t *testing.T) {
	type testPayload struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	tests := []struct {
		name    string
		body    string
		wantErr bool
		want    testPayload
	}{
		{
			name:    "valid JSON",
			body:    `{"name":"Alice","email":"alice@example.com"}`,
			wantErr: false,
			want:    testPayload{Name: "Alice", Email: "alice@example.com"},
		},
		{
			name:    "malformed JSON",
			body:    `{broken`,
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			var result testPayload
			err := httputil.DecodeJSON(req, &result)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != tt.want {
					t.Errorf("decoded result: got %+v, want %+v", result, tt.want)
				}
			}
		})
	}
}

// ============================================================
// ReadParam tests
// ============================================================

func TestReadParam(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		query string
		param string
		want  string
	}{
		{
			name:  "query parameter present",
			path:  "/vehicles",
			query: "from=2024-01-01&to=2024-12-31",
			param: "from",
			want:  "2024-01-01",
		},
		{
			name:  "missing parameter returns empty string",
			path:  "/vehicles",
			query: "",
			param: "nonexistent",
			want:  "",
		},
		{
			name:  "type filter query parameter",
			path:  "/vehicles/123/history",
			query: "type=fuel",
			param: "type",
			want:  "fuel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path+"?"+tt.query, nil)
			got := httputil.ReadParam(req, tt.param)
			if got != tt.want {
				t.Errorf("ReadParam(): got %q, want %q", got, tt.want)
			}
		})
	}
}

// ============================================================
// WriteJSON / WriteError tests
// ============================================================

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	data := map[string]string{"hello": "world"}
	httputil.WriteJSON(rec, http.StatusOK, data)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json")
	}

	var result map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON body: %v", err)
	}
	if result["hello"] != "world" {
		t.Errorf("body: got %v, want hello=world", result)
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()

	httputil.WriteError(rec, http.StatusNotFound, "not_found", "resource not found")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusNotFound)
	}

	var result map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON body: %v", err)
	}
	if result["error"] != "not_found" {
		t.Errorf("error code: got %q, want %q", result["error"], "not_found")
	}
	if result["message"] != "resource not found" {
		t.Errorf("message: got %q, want %q", result["message"], "resource not found")
	}
}

// ============================================================
// CORSMiddleware tests
// ============================================================

func TestCORSMiddleware_SetsHeaders(t *testing.T) {
	// Verify that CORS headers are set on a normal GET request.
	handler := CORSMiddleware(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check CORS headers
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin: got %q, want %q", got, "*")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods: got %q, want %q", got, "GET, POST, OPTIONS")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization" {
		t.Errorf("Access-Control-Allow-Headers: got %q, want %q", got, "Content-Type, Authorization")
	}

	// The response should still reach the underlying handler.
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCORSMiddleware_PreflightOptions(t *testing.T) {
	// Verify that OPTIONS preflight requests return 204 with CORS headers.
	handler := CORSMiddleware(okHandler)

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// OPTIONS should return 204 No Content
	if rec.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusNoContent)
	}

	// CORS headers must still be present
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin: got %q, want %q", got, "*")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods: got %q, want %q", got, "GET, POST, OPTIONS")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization" {
		t.Errorf("Access-Control-Allow-Headers: got %q, want %q", got, "Content-Type, Authorization")
	}
}

func TestCORSMiddleware_PostRequest(t *testing.T) {
	// Verify that POST requests also get CORS headers and reach the handler.
	handler := CORSMiddleware(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin: got %q, want %q", got, "*")
	}
}