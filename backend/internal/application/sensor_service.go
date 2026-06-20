// Package application contains business logic services (use cases).
package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

type FuelChecker interface {
	CheckAutonomy(ctx context.Context, vehicleID string, deviceID string, readingsCount int) (*domain.Alert, error)
}

type Broadcaster interface {
	Broadcast(msg []byte)
}

type SensorService struct {
	sensorRepo   domain.SensorDataRepository
	vehicleRepo  domain.VehicleRepository
	fuelService  FuelChecker
	broadcaster  Broadcaster // nil-safe: if nil, broadcasts are skipped
}

func NewSensorService(
	sensorRepo domain.SensorDataRepository,
	vehicleRepo domain.VehicleRepository,
	fuelService FuelChecker,
) *SensorService {
	return &SensorService{
		sensorRepo:  sensorRepo,
		vehicleRepo: vehicleRepo,
		fuelService: fuelService,
	}
}

// WithBroadcaster sets the broadcaster for WebSocket push notifications.
func (s *SensorService) WithBroadcaster(b Broadcaster) *SensorService {
	s.broadcaster = b
	return s
}

// IngestBatch processes a batch of sensor data points from IoT devices.
// Rejects the entire batch if any device_id is unknown.
// Fuel autonomy check errors are swallowed — alerts are best-effort.
func (s *SensorService) IngestBatch(ctx context.Context, inputs []domain.SensorInput) error {
	if len(inputs) == 0 {
		return fmt.Errorf("%w: empty batch", domain.ErrValidation)
	}

	if len(inputs) > 100 {
		return fmt.Errorf("%w: batch size %d exceeds maximum of 100", domain.ErrValidation, len(inputs))
	}

	for _, input := range inputs {
		if err := s.validateInput(input); err != nil {
			return err
		}
	}

	// Resolve device_id → vehicle_id with caching to avoid N+1 queries.
	deviceToVehicle := make(map[string]string)
	for _, input := range inputs {
		if _, exists := deviceToVehicle[input.DeviceID]; exists {
			continue
		}

		vehicle, err := s.vehicleRepo.FindByDeviceID(ctx, input.DeviceID)
		if err != nil {
			// Reject entire batch if any device_id is unknown.
			return fmt.Errorf("%w: device_id %s not found", domain.ErrValidation, input.DeviceID)
		}
		deviceToVehicle[input.DeviceID] = vehicle.ID
	}

	dataPoints := make([]domain.SensorData, 0, len(inputs))
	now := time.Now()

	for _, input := range inputs {
		timestamp, _ := time.Parse(time.RFC3339, input.Timestamp)

		dataPoints = append(dataPoints, domain.SensorData{
			ID:        generateID(),
			VehicleID: deviceToVehicle[input.DeviceID],
			Type:      input.Type,
			Value:     input.Value,
			Timestamp: timestamp,
			CreatedAt: now,
		})
	}

	// BulkInsert is atomic: all succeed or all fail.
	if err := s.sensorRepo.BulkInsert(ctx, dataPoints); err != nil {
		return err
	}

	// Fuel autonomy alerts are best-effort — errors are swallowed.
	for _, input := range inputs {
		if input.Type == "fuel" {
			alert, _ := s.fuelService.CheckAutonomy(ctx, deviceToVehicle[input.DeviceID], input.DeviceID, 10)
			if alert != nil && s.broadcaster != nil {
				payload, err := json.Marshal(map[string]interface{}{
					"type":       "low_fuel",
					"vehicle_id": alert.VehicleID,
					"device_id":  alert.DeviceID,
				})
				if err == nil {
					s.broadcaster.Broadcast(payload)
				}
			}
		}
	}

	// Broadcast after persistence so clients only see committed data.
	if s.broadcaster != nil {
		msg := struct {
			Type string               `json:"type"`
			Data []domain.SensorData `json:"data"`
		}{
			Type: "sensor_update",
			Data: dataPoints,
		}
		if payload, err := json.Marshal(msg); err == nil {
			s.broadcaster.Broadcast(payload)
		}
	}

	return nil
}

func (s *SensorService) validateInput(input domain.SensorInput) error {
	if input.DeviceID == "" {
		return fmt.Errorf("%w: device_id is required", domain.ErrValidation)
	}

	ts, err := time.Parse(time.RFC3339, input.Timestamp)
	if err != nil {
		return fmt.Errorf("%w: invalid timestamp format: expected RFC3339", domain.ErrValidation)
	}

	// Allow up to 30 seconds of clock drift from IoT devices.
	if ts.After(time.Now().Add(30 * time.Second)) {
		return fmt.Errorf("%w: timestamp is too far in the future", domain.ErrValidation)
	}

	switch input.Type {
	case "gps", "fuel", "temperature":
	default:
		return fmt.Errorf("%w: invalid sensor type %q, must be gps, fuel, or temperature", domain.ErrValidation, input.Type)
	}

	switch input.Type {
	case "gps":
		var coords domain.GPSCoordinates
		if err := json.Unmarshal(input.Value, &coords); err != nil {
			return fmt.Errorf("%w: invalid GPS coordinates: %v", domain.ErrValidation, err)
		}
		// float64 can't distinguish "missing key" from "key present with value 0".
		// Rejecting 0 catches missing-field bugs; see Null Island trade-off.
		if coords.Lat == 0 {
			return fmt.Errorf("%w: GPS coordinates must include lat", domain.ErrValidation)
		}
		if coords.Lng == 0 {
			return fmt.Errorf("%w: GPS coordinates must include lng", domain.ErrValidation)
		}

	case "fuel":
		var fuel domain.FuelReading
		if err := json.Unmarshal(input.Value, &fuel); err != nil {
			return fmt.Errorf("%w: invalid fuel reading: %v", domain.ErrValidation, err)
		}
		// float64 can't distinguish "missing" from "0". Rejecting 0 is pragmatic.
		if fuel.Level == 0 {
			return fmt.Errorf("%w: fuel reading must include level", domain.ErrValidation)
		}

	case "temperature":
		var temp domain.TemperatureReading
		if err := json.Unmarshal(input.Value, &temp); err != nil {
			return fmt.Errorf("%w: invalid temperature reading: %v", domain.ErrValidation, err)
		}
		// 0°C is valid but float64 can't distinguish "missing" from "0".
		if temp.Celsius == 0 {
			return fmt.Errorf("%w: temperature reading must include celsius", domain.ErrValidation)
		}
	}

	return nil
}

// generateID creates a UUID v4 using crypto/rand for unpredictable IDs.
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate random ID: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2

	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}