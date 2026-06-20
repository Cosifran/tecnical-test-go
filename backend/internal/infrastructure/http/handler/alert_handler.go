package handler

import (
	"net/http"

	"github.com/francisco/fleet-monitor/internal/domain"
	"github.com/francisco/fleet-monitor/internal/infrastructure/http/httputil"
)

type AlertHandler struct {
	alertRepo domain.AlertRepository
}

func NewAlertHandler(alertRepo domain.AlertRepository) *AlertHandler {
	return &AlertHandler{alertRepo: alertRepo}
}

// List handles GET /alerts. Only accessible by admin users (enforced by RBACMiddleware).
func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.alertRepo.FindAll(r.Context())
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve alerts")
		return
	}

	// Return empty array instead of null for JSON clients.
	if alerts == nil {
		alerts = []domain.Alert{}
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"alerts": alerts,
	})
}