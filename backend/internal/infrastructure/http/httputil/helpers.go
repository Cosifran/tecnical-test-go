package httputil

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/francisco/fleet-monitor/internal/domain"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyEmail  contextKey = "user_email"
	ContextKeyRole   contextKey = "user_role"
)

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyUserID).(string)
	return v
}

func EmailFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyEmail).(string)
	return v
}

func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ContextKeyRole).(string)
	return v
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, &errorResponse{
		Error:   code,
		Message: message,
	})
}

func DecodeJSON(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("%w: request body is empty", domain.ErrValidation)
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}
	return nil
}

// ReadParam checks path parameters first (Go 1.22+ {name} patterns), then query parameters.
func ReadParam(r *http.Request, name string) string {
	if val := r.PathValue(name); val != "" {
		return val
	}
	return r.URL.Query().Get(name)
}