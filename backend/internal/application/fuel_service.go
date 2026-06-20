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

// CheckAutonomy calculates fuel consumption rate from historical sensor data
// and returns an alert if the vehicle has less than 1 hour of autonomy remaining.
func (fs *FuelService) CheckAutonomy(ctx context.Context, vehicleID string, deviceID string, readingsCount int) (*domain.Alert, error) {
	data, err := fs.sensorRepo.FindByVehicleID(ctx, vehicleID, time.Time{}, time.Time{}, "fuel", readingsCount)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

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

	// At least 3 readings are needed for a meaningful consumption rate.
	if len(readings) < 3 {
		return nil, nil
	}

	oldest := readings[0]
	newest := readings[len(readings)-1]

	deltaLiters := oldest.Level - newest.Level
	deltaTime := newest.Timestamp.Sub(oldest.Timestamp)
	deltaHours := deltaTime.Hours()

	rate := deltaLiters / deltaHours

	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return nil, nil
	}

	autonomy := newest.Level / rate
	if math.IsNaN(autonomy) || math.IsInf(autonomy, 0) {
		return nil, nil
	}

	if autonomy >= 1.0 {
		return nil, nil
	}

	detailsMap := map[string]interface{}{
		"autonomy":      autonomy,
		"rate":          rate,
		"current_level": newest.Level,
	}

	detailsJSON, err := json.Marshal(detailsMap)
	if err != nil {
		return nil, err
	}

	alert := &domain.Alert{
		VehicleID: vehicleID,
		DeviceID:  deviceID,
		Type:      "low_fuel",
		Severity:  "critical",
		Details:   detailsJSON,
	}

	if err := fs.alertRepo.Create(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}