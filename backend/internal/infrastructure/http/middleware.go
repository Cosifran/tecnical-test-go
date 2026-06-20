package http

import (
	"bufio"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
	"github.com/francisco/fleet-monitor/internal/infrastructure/http/httputil"
	"github.com/francisco/fleet-monitor/internal/infrastructure/jwt"
)

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func AuthMiddleware(tokenService *jwt.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractToken(r)
			if tokenString == "" {
				httputil.WriteError(w, http.StatusUnauthorized, "invalid_token", "missing or malformed token")
				return
			}

			claims, err := tokenService.Validate(tokenString)
			if err != nil {
				if errors.Is(err, domain.ErrTokenExpired) {
					httputil.WriteError(w, http.StatusUnauthorized, "token_expired", "token has expired")
					return
				}
				httputil.WriteError(w, http.StatusUnauthorized, "invalid_token", "invalid or tampered token")
				return
			}

			ctx := context.WithValue(r.Context(), httputil.ContextKeyUserID, claims.Subject)
			ctx = context.WithValue(ctx, httputil.ContextKeyEmail, claims.Email)
			ctx = context.WithValue(ctx, httputil.ContextKeyRole, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			return authHeader[7:]
		}
	}
	return r.URL.Query().Get("token")
}

func RBACMiddleware(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := httputil.RoleFromContext(r.Context())
			if role == "" {
				httputil.WriteError(w, http.StatusForbidden, "forbidden", "no role found in context")
				return
			}

			for _, allowed := range allowedRoles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}

			httputil.WriteError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
		})
	}
}

// responseRecorder wraps ResponseWriter to capture the status code for logging.
// Must implement http.Hijacker for WebSocket upgrade to work.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, errors.New("response writer does not support hijacking")
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(recorder, r)

		duration := time.Since(start)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.statusCode,
			"duration", duration,
		)
	})
}