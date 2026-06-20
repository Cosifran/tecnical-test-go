package handler

import (
	"errors"
	"net/http"

	"github.com/francisco/fleet-monitor/internal/application"
	"github.com/francisco/fleet-monitor/internal/domain"
	"github.com/francisco/fleet-monitor/internal/infrastructure/http/httputil"
)

type SensorHandler struct {
	sensorService *application.SensorService
}

func NewSensorHandler(sensorService *application.SensorService) *SensorHandler {
	return &SensorHandler{sensorService: sensorService}
}

// Ingest handles POST /sensors/data. Decodes a JSON array and passes it to SensorService.IngestBatch.
// Rejects batches > 100 items at handler level as defense in depth (service also validates).
func (h *SensorHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	var inputs []domain.SensorInput
	if err := httputil.DecodeJSON(r, &inputs); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	if len(inputs) > 100 {
		httputil.WriteError(w, http.StatusBadRequest, "validation_error",
			"batch size exceeds maximum of 100 items")
		return
	}

	if len(inputs) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "validation_error",
			"batch cannot be empty")
		return
	}

	if err := h.sensorService.IngestBatch(r.Context(), inputs); err != nil {
		if errors.Is(err, domain.ErrValidation) {
			httputil.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to ingest sensor data")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"inserted": len(inputs),
	})
}