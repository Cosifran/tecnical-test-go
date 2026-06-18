package application

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

type FuelService struct {
	sensorRepo domain.SensorDataRepository
	alertRepo  domain.AlertRepository
}

func NewFuelService(sensorRepo domain.SensorDataRepository, alertRepo domain.AlertRepository) *FuelService {
	return &FuelService{
		sensorRepo: sensorRepo,
		alertRepo:  alertRepo,
	}
}

func (fs *FuelService) CheckAutonomy(ctx context.Context, vehicleID string, readingsCount int) error {
	// IMPLEMENTAR:
	// 1. Llamar a sensorRepo.FindByVehicleID para obtener lecturas de tipo "fuel"
	to := time.Time{}
	from := time.Time{}

	data, err := fs.sensorRepo.FindByVehicleID(ctx, vehicleID, from, to, "fuel")

	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	// 2. Filtrar/ordenar por timestamp
	sort.Slice(data, func(i, j int) bool {
		// Esta función anonima retorna true si data[i] debe ir ANTES que data[j]
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

	// 3. Extraer los valores de combustible usando json.Unmarshal en FuelReading
	readings := make([]domain.FuelReadingWithTime, 0, len(data))

	for _, point := range data {
		var fuel domain.FuelReading
		if err := json.Unmarshal(point.Value, &fuel); err != nil {
			continue
		}

		readings = append(readings, domain.FuelReadingWithTime{
			Level:     fuel.Level,
			Timestamp: point.Timestamp,
		})
	}

	if len(readings) < 3 {
		return nil
	}

	// 4. Calcular la tasa y autonomía
	oldest := readings[0]
	newest := readings[len(readings)-1]

	deltaLiters := oldest.Level - newest.Level

	deltaTime := newest.Timestamp.Sub(oldest.Timestamp)
	deltaHours := deltaTime.Hours()

	rate := deltaLiters / deltaHours

	// 5. Si autonomía < 1 hora, crear alerta con alertRepo.Create

	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return nil
	}

	autonomy := newest.Level / rate

	if math.IsNaN(autonomy) || math.IsInf(autonomy, 0) {
		return nil
	}

	// 6. Si autonomía < 1 hora, crear alerta con alertRepo.Create
	if autonomy >= 1.0 {
		return nil
	}

	detailsMap := map[string]interface{}{
		"autonomy":      autonomy,
		"rate":          rate,
		"current_level": newest.Level,
	}

	detailsJSON, err := json.Marshal(detailsMap)

	if err != nil {
		return err
	}

	alert := &domain.Alert{
		VehicleID: vehicleID,
		Type:      "low_fuel",
		Severity:  "critical",
		Details:   detailsJSON,
	}

	return fs.alertRepo.Create(ctx, alert)
}
