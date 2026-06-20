package handler

import (
	"errors"
	"net/http"

	"github.com/francisco/fleet-monitor/internal/application"
	"github.com/francisco/fleet-monitor/internal/domain"
	"github.com/francisco/fleet-monitor/internal/infrastructure/http/httputil"
)

type AuthHandler struct {
	authService *application.AuthService
}

func NewAuthHandler(authService *application.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles POST /auth/login. Returns ErrUnauthorized for both wrong email and wrong password to prevent enumeration.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	result, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid email or password")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, result)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh handles POST /auth/refresh. Validates a refresh token and returns new tokens.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	result, err := h.authService.Refresh(req.RefreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrTokenExpired) {
			httputil.WriteError(w, http.StatusUnauthorized, "token_expired", "refresh token has expired")
			return
		}
		if errors.Is(err, domain.ErrTokenInvalid) {
			httputil.WriteError(w, http.StatusUnauthorized, "invalid_token", "invalid refresh token")
			return
		}
		if errors.Is(err, domain.ErrUnauthorized) {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid refresh token")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, result)
}