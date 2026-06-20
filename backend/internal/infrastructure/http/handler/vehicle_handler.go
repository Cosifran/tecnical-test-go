package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/francisco/fleet-monitor/internal/application"
	"github.com/francisco/fleet-monitor/internal/domain"
	"github.com/francisco/fleet-monitor/internal/infrastructure/http/httputil"
)

type VehicleHandler struct {
	vehicleService *application.VehicleService
}

func NewVehicleHandler(vehicleService *application.VehicleService) *VehicleHandler {
	return &VehicleHandler{vehicleService: vehicleService}
}

// List handles GET /vehicles. Returns vehicles with masked device IDs for non-admin users.
func (h *VehicleHandler) List(w http.ResponseWriter, r *http.Request) {
	role := httputil.RoleFromContext(r.Context())

	vehicles, err := h.vehicleService.ListVehicles(r.Context(), role)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list vehicles")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"vehicles": vehicles,
	})
}

// History handles GET /vehicles/{id}/history with optional from, to, and type query params.
func (h *VehicleHandler) History(w http.ResponseWriter, r *http.Request) {
	role := httputil.RoleFromContext(r.Context())
	vehicleID := httputil.ReadParam(r, "id")
	if vehicleID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "validation_error", "vehicle id is required")
		return
	}

	var from, to time.Time
	if fromStr := httputil.ReadParam(r, "from"); fromStr != "" {
		parsed, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "validation_error",
				"invalid 'from' timestamp: expected RFC3339 format")
			return
		}
		from = parsed
	}

	if toStr := httputil.ReadParam(r, "to"); toStr != "" {
		parsed, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			httputil.WriteError(w, http.StatusBadRequest, "validation_error",
				"invalid 'to' timestamp: expected RFC3339 format")
			return
		}
		to = parsed
	}

	sensorType := httputil.ReadParam(r, "type")

	sensorData, vehicle, err := h.vehicleService.GetVehicleHistory(
		r.Context(), vehicleID, from, to, sensorType, role,
	)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "vehicle not found")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve vehicle history")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"vehicle": vehicle,
		"history": sensorData,
	})
}