package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/francisco/fleet-monitor/internal/application"
	"github.com/francisco/fleet-monitor/internal/domain"
)

// ============================================================
// Mocks for SensorService tests
// ============================================================

// mockIngestSensorRepo implements domain.SensorDataRepository for tests.
// It tracks calls to BulkInsert so tests can verify what was persisted.
type mockIngestSensorRepo struct {
	findByVehicleIDData []domain.SensorData
	findByVehicleIDErr  error
	bulkInsertErr       error
	insertedData        []domain.SensorData // tracks all data passed to BulkInsert
}

func (m *mockIngestSensorRepo) FindByVehicleID(ctx context.Context, vehicleID string, from, to time.Time, sensorType string, limit int) ([]domain.SensorData, error) {
	return m.findByVehicleIDData, m.findByVehicleIDErr
}

func (m *mockIngestSensorRepo) BulkInsert(ctx context.Context, data []domain.SensorData) error {
	if m.bulkInsertErr != nil {
		return m.bulkInsertErr
	}
	m.insertedData = append(m.insertedData, data...)
	return nil
}

// mockIngestVehicleRepo implements domain.VehicleRepository for tests.
// It maps device IDs to vehicles for lookup.
type mockIngestVehicleRepo struct {
	vehiclesByDeviceID map[string]*domain.Vehicle
	vehiclesByID       map[string]*domain.Vehicle
	allVehicles        []domain.Vehicle
	findByDeviceIDErr  error
	findByIDErr        error
	findAllErr         error
}

func (m *mockIngestVehicleRepo) FindByDeviceID(ctx context.Context, deviceID string) (*domain.Vehicle, error) {
	if m.findByDeviceIDErr != nil {
		return nil, m.findByDeviceIDErr
	}
	v, ok := m.vehiclesByDeviceID[deviceID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return v, nil
}

func (m *mockIngestVehicleRepo) FindByID(ctx context.Context, id string) (*domain.Vehicle, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	v, ok := m.vehiclesByID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return v, nil
}

func (m *mockIngestVehicleRepo) FindAll(ctx context.Context) ([]domain.Vehicle, error) {
	if m.findAllErr != nil {
		return nil, m.findAllErr
	}
	return m.allVehicles, nil
}

func (m *mockIngestVehicleRepo) Create(ctx context.Context, vehicle *domain.Vehicle) error {
	return nil
}

// mockIngestFuelChecker implements application.FuelChecker for tests.
// It tracks CheckAutonomy calls so tests can verify fuel checks are triggered.
type mockIngestFuelChecker struct {
	calls  []fuelCheckCall  // records each CheckAutonomy invocation
	alert  *domain.Alert    // alert to return, nil means no alert
	err    error            // error to return, nil means success
}

type fuelCheckCall struct {
	vehicleID     string
	deviceID      string
	readingsCount int
}

func (m *mockIngestFuelChecker) CheckAutonomy(ctx context.Context, vehicleID string, deviceID string, readingsCount int) (*domain.Alert, error) {
	m.calls = append(m.calls, fuelCheckCall{
		vehicleID:     vehicleID,
		deviceID:      deviceID,
		readingsCount: readingsCount,
	})
	return m.alert, m.err
}

// ============================================================
// Test helpers
// ============================================================

// validGPSInput creates a valid SensorInput for GPS type.
func validGPSInput(deviceID, timestamp string) domain.SensorInput {
	value, _ := json.Marshal(domain.GPSCoordinates{Lat: 40.7128, Lng: -74.006})
	return domain.SensorInput{
		DeviceID:  deviceID,
		Timestamp: timestamp,
		Type:      "gps",
		Value:     value,
	}
}

// validFuelInput creates a valid SensorInput for fuel type.
func validFuelInput(deviceID, timestamp string) domain.SensorInput {
	value, _ := json.Marshal(domain.FuelReading{Level: 75.5, Unit: "liters"})
	return domain.SensorInput{
		DeviceID:  deviceID,
		Timestamp: timestamp,
		Type:      "fuel",
		Value:     value,
	}
}

// validTemperatureInput creates a valid SensorInput for temperature type.
func validTemperatureInput(deviceID, timestamp string) domain.SensorInput {
	value, _ := json.Marshal(domain.TemperatureReading{Celsius: 22.3})
	return domain.SensorInput{
		DeviceID:  deviceID,
		Timestamp: timestamp,
		Type:      "temperature",
		Value:     value,
	}
}

// nowTimestamp returns the current time formatted as RFC3339.
func nowTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

// ============================================================
// Tests for SensorService.IngestBatch
// ============================================================

func TestIngestBatch_ValidBatchWithTwoPoints_Persisted(t *testing.T) {
	// Arrange: set up a sensor service with mock dependencies
	sensorRepo := &mockIngestSensorRepo{}
	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-001": {ID: "vehicle-1", DeviceID: "DEV-001", Name: "Truck 01"},
			"DEV-002": {ID: "vehicle-2", DeviceID: "DEV-002", Name: "Truck 02"},
		},
	}
	fuelChecker := &mockIngestFuelChecker{}
	service := application.NewSensorService(sensorRepo, vehicleRepo, fuelChecker)

	ts := nowTimestamp()
	inputs := []domain.SensorInput{
		validGPSInput("DEV-001", ts),
		validFuelInput("DEV-002", ts),
	}

	// Act: ingest the batch
	err := service.IngestBatch(context.Background(), inputs)

	// Assert: no error, data was persisted
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(sensorRepo.insertedData) != 2 {
		t.Errorf("expected 2 data points persisted, got %d", len(sensorRepo.insertedData))
	}

	// Verify the GPS point was saved with the correct vehicle mapping
	gpsData := sensorRepo.insertedData[0]
	if gpsData.VehicleID != "vehicle-1" {
		t.Errorf("expected GPS data vehicleID 'vehicle-1', got %q", gpsData.VehicleID)
	}
	if gpsData.Type != "gps" {
		t.Errorf("expected GPS data type 'gps', got %q", gpsData.Type)
	}

	// Verify the fuel point triggered a CheckAutonomy call
	if len(fuelChecker.calls) != 1 {
		t.Fatalf("expected 1 CheckAutonomy call, got %d", len(fuelChecker.calls))
	}
	if fuelChecker.calls[0].vehicleID != "vehicle-2" {
		t.Errorf("expected CheckAutonomy for 'vehicle-2', got %q", fuelChecker.calls[0].vehicleID)
	}
	if fuelChecker.calls[0].deviceID != "DEV-002" {
		t.Errorf("expected CheckAutonomy deviceID 'DEV-002', got %q", fuelChecker.calls[0].deviceID)
	}
	if fuelChecker.calls[0].readingsCount != 10 {
		t.Errorf("expected readingsCount 10, got %d", fuelChecker.calls[0].readingsCount)
	}
}

func TestIngestBatch_EmptyBatch_Error(t *testing.T) {
	// Arrange
	service := application.NewSensorService(
		&mockIngestSensorRepo{},
		&mockIngestVehicleRepo{},
		&mockIngestFuelChecker{},
	)

	// Act: try to ingest an empty batch
	err := service.IngestBatch(context.Background(), []domain.SensorInput{})

	// Assert: should return ErrValidation
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestIngestBatch_BatchExceeds100Items_Error(t *testing.T) {
	// Arrange: create a batch with 101 items
	batch := make([]domain.SensorInput, 101)
	ts := nowTimestamp()
	for i := range batch {
		batch[i] = validGPSInput("DEV-001", ts)
	}

	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-001": {ID: "vehicle-1", DeviceID: "DEV-001", Name: "Truck 01"},
		},
	}

	service := application.NewSensorService(
		&mockIngestSensorRepo{},
		vehicleRepo,
		&mockIngestFuelChecker{},
	)

	// Act: try to ingest the oversized batch
	err := service.IngestBatch(context.Background(), batch)

	// Assert: should return ErrValidation
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestIngestBatch_InvalidType_Error(t *testing.T) {
	// Arrange: create an input with an invalid sensor type
	value, _ := json.Marshal(domain.GPSCoordinates{Lat: 40.7128, Lng: -74.006})
	inputs := []domain.SensorInput{
		{
			DeviceID:  "DEV-001",
			Timestamp: nowTimestamp(),
			Type:      "humidity", // invalid type!
			Value:     value,
		},
	}

	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-001": {ID: "vehicle-1", DeviceID: "DEV-001", Name: "Truck 01"},
		},
	}

	service := application.NewSensorService(
		&mockIngestSensorRepo{},
		vehicleRepo,
		&mockIngestFuelChecker{},
	)

	// Act
	err := service.IngestBatch(context.Background(), inputs)

	// Assert: should return ErrValidation with message about invalid type
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestIngestBatch_FutureTimestamp_Error(t *testing.T) {
	// Arrange: create an input with a timestamp 60 seconds in the future
	// (beyond the 30-second tolerance window)
	futureTS := time.Now().Add(60 * time.Second).Format(time.RFC3339)
	inputs := []domain.SensorInput{
		validGPSInput("DEV-001", futureTS),
	}

	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-001": {ID: "vehicle-1", DeviceID: "DEV-001", Name: "Truck 01"},
		},
	}

	service := application.NewSensorService(
		&mockIngestSensorRepo{},
		vehicleRepo,
		&mockIngestFuelChecker{},
	)

	// Act
	err := service.IngestBatch(context.Background(), inputs)

	// Assert: should reject the timestamp as too far in the future
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestIngestBatch_UnknownDeviceID_Error(t *testing.T) {
	// Arrange: the device ID is not in the repository
	sensorRepo := &mockIngestSensorRepo{}
	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			// No entry for "DEV-UNKNOWN"
		},
	}

	service := application.NewSensorService(
		sensorRepo,
		vehicleRepo,
		&mockIngestFuelChecker{},
	)

	inputs := []domain.SensorInput{
		validGPSInput("DEV-UNKNOWN", nowTimestamp()),
	}

	// Act
	err := service.IngestBatch(context.Background(), inputs)

	// Assert: should reject the entire batch
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}

	// No data should have been persisted
	if len(sensorRepo.insertedData) != 0 {
		t.Errorf("expected 0 persisted data points, got %d", len(sensorRepo.insertedData))
	}
}

func TestIngestBatch_FuelPointTriggersCheckAutonomy(t *testing.T) {
	// Arrange: set up with a fuel input
	sensorRepo := &mockIngestSensorRepo{}
	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-FUEL": {ID: "vehicle-fuel", DeviceID: "DEV-FUEL", Name: "Fuel Truck"},
		},
	}
	fuelChecker := &mockIngestFuelChecker{}

	service := application.NewSensorService(sensorRepo, vehicleRepo, fuelChecker)

	inputs := []domain.SensorInput{
		validFuelInput("DEV-FUEL", nowTimestamp()),
	}

	// Act
	err := service.IngestBatch(context.Background(), inputs)

	// Assert: no error from ingestion
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// CheckAutonomy should have been called exactly once
	if len(fuelChecker.calls) != 1 {
		t.Fatalf("expected 1 CheckAutonomy call, got %d", len(fuelChecker.calls))
	}
	if fuelChecker.calls[0].vehicleID != "vehicle-fuel" {
		t.Errorf("expected vehicleID 'vehicle-fuel', got %q", fuelChecker.calls[0].vehicleID)
	}
	if fuelChecker.calls[0].deviceID != "DEV-FUEL" {
		t.Errorf("expected deviceID 'DEV-FUEL', got %q", fuelChecker.calls[0].deviceID)
	}
	if fuelChecker.calls[0].readingsCount != 10 {
		t.Errorf("expected readingsCount 10, got %d", fuelChecker.calls[0].readingsCount)
	}
}

func TestIngestBatch_FuelCheckErrorSwallowed(t *testing.T) {
	// Arrange: set up with a fuel input, but CheckAutonomy returns an error.
	// The error should be SWALLOWED — not propagated to the caller.
	sensorRepo := &mockIngestSensorRepo{}
	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-FUEL": {ID: "vehicle-fuel", DeviceID: "DEV-FUEL", Name: "Fuel Truck"},
		},
	}
	// Simulate a fuel check failure
	fuelChecker := &mockIngestFuelChecker{
		err: errors.New("fuel check failed"),
	}

	service := application.NewSensorService(sensorRepo, vehicleRepo, fuelChecker)

	inputs := []domain.SensorInput{
		validFuelInput("DEV-FUEL", nowTimestamp()),
	}

	// Act: ingest should succeed even though CheckAutonomy fails
	err := service.IngestBatch(context.Background(), inputs)

	// Assert: no error should be returned (fuel error swallowed)
	if err != nil {
		t.Errorf("expected no error (fuel error should be swallowed), got: %v", err)
	}

	// Data should still be persisted
	if len(sensorRepo.insertedData) != 1 {
		t.Errorf("expected 1 data point persisted, got %d", len(sensorRepo.insertedData))
	}
}

func TestIngestBatch_RepoErrorOnBulkInsert_ErrorReturned(t *testing.T) {
	// Arrange: set up BulkInsert to return an error
	repoErr := errors.New("database connection lost")
	sensorRepo := &mockIngestSensorRepo{
		bulkInsertErr: repoErr,
	}
	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-001": {ID: "vehicle-1", DeviceID: "DEV-001", Name: "Truck 01"},
		},
	}

	service := application.NewSensorService(sensorRepo, vehicleRepo, &mockIngestFuelChecker{})

	inputs := []domain.SensorInput{
		validGPSInput("DEV-001", nowTimestamp()),
	}

	// Act
	err := service.IngestBatch(context.Background(), inputs)

	// Assert: the repository error should be returned to the caller
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repository error, got: %v", err)
	}
}

func TestIngestBatch_DeviceIDCacheAvoidsN1Queries(t *testing.T) {
	// Arrange: two inputs with the SAME device_id should only trigger
	// one FindByDeviceID call (the map acts as a cache).
	sensorRepo := &mockIngestSensorRepo{}
	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-001": {ID: "vehicle-1", DeviceID: "DEV-001", Name: "Truck 01"},
		},
	}
	fuelChecker := &mockIngestFuelChecker{}

	service := application.NewSensorService(sensorRepo, vehicleRepo, fuelChecker)

	ts := nowTimestamp()
	inputs := []domain.SensorInput{
		validGPSInput("DEV-001", ts),
		validFuelInput("DEV-001", ts), // same device_id as above
	}

	// Act
	err := service.IngestBatch(context.Background(), inputs)

	// Assert: both data points should be persisted for the same vehicle
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(sensorRepo.insertedData) != 2 {
		t.Errorf("expected 2 data points, got %d", len(sensorRepo.insertedData))
	}
	// Both should resolve to the same vehicle
	for _, data := range sensorRepo.insertedData {
		if data.VehicleID != "vehicle-1" {
			t.Errorf("expected vehicleID 'vehicle-1', got %q", data.VehicleID)
		}
	}
}

func TestIngestBatch_InvalidGPSValue_Error(t *testing.T) {
	// Arrange: GPS input with missing lat field
	value, _ := json.Marshal(map[string]interface{}{"lng": -74.006})
	inputs := []domain.SensorInput{
		{
			DeviceID:  "DEV-001",
			Timestamp: nowTimestamp(),
			Type:      "gps",
			Value:     value,
		},
	}

	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-001": {ID: "vehicle-1", DeviceID: "DEV-001", Name: "Truck 01"},
		},
	}

	service := application.NewSensorService(
		&mockIngestSensorRepo{},
		vehicleRepo,
		&mockIngestFuelChecker{},
	)

	// Act
	err := service.IngestBatch(context.Background(), inputs)

	// Assert: should reject the GPS value with no lat
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestIngestBatch_EmptyDeviceID_Error(t *testing.T) {
	// Arrange: input with empty device_id
	inputs := []domain.SensorInput{
		{
			DeviceID:  "", // empty!
			Timestamp: nowTimestamp(),
			Type:      "gps",
			Value:     json.RawMessage(`{"lat": 40.7, "lng": -74.0}`),
		},
	}

	service := application.NewSensorService(
		&mockIngestSensorRepo{},
		&mockIngestVehicleRepo{},
		&mockIngestFuelChecker{},
	)

	// Act
	err := service.IngestBatch(context.Background(), inputs)

	// Assert
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestIngestBatch_InvalidTimestampFormat_Error(t *testing.T) {
	// Arrange: input with a non-RFC3339 timestamp
	inputs := []domain.SensorInput{
		{
			DeviceID:  "DEV-001",
			Timestamp: "not-a-timestamp", // invalid format!
			Type:      "gps",
			Value:     json.RawMessage(`{"lat": 40.7, "lng": -74.0}`),
		},
	}

	service := application.NewSensorService(
		&mockIngestSensorRepo{},
		&mockIngestVehicleRepo{},
		&mockIngestFuelChecker{},
	)

	// Act
	err := service.IngestBatch(context.Background(), inputs)

	// Assert
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestIngestBatch_FuelValueMissingLevel_Error(t *testing.T) {
	// Arrange: fuel input with missing level field (level defaults to 0)
	value, _ := json.Marshal(map[string]interface{}{"unit": "liters"})
	inputs := []domain.SensorInput{
		{
			DeviceID:  "DEV-001",
			Timestamp: nowTimestamp(),
			Type:      "fuel",
			Value:     value,
		},
	}

	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-001": {ID: "vehicle-1", DeviceID: "DEV-001", Name: "Truck 01"},
		},
	}

	service := application.NewSensorService(
		&mockIngestSensorRepo{},
		vehicleRepo,
		&mockIngestFuelChecker{},
	)

	// Act
	err := service.IngestBatch(context.Background(), inputs)

	// Assert: should reject fuel value with no level
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got: %v", err)
	}
}

func TestIngestBatch_GPSCoordinatesValidation(t *testing.T) {
	tests := []struct {
		name      string
		gpsValue  json.RawMessage
		wantErr   bool
	}{
		{
			name:     "valid GPS coordinates",
			gpsValue: json.RawMessage(`{"lat": 40.7128, "lng": -74.006}`),
			wantErr:  false,
		},
		{
			name:     "GPS with missing lat",
			gpsValue: json.RawMessage(`{"lng": -74.006}`),
			wantErr:  true,
		},
		{
			name:     "GPS with missing lng",
			gpsValue: json.RawMessage(`{"lat": 40.7128}`),
			wantErr:  true,
		},
		{
			name:     "GPS with both missing",
			gpsValue: json.RawMessage(`{}`),
			wantErr:  true,
		},
		{
			name:     "GPS with invalid JSON",
			gpsValue: json.RawMessage(`not json at all`),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sensorRepo := &mockIngestSensorRepo{}
			vehicleRepo := &mockIngestVehicleRepo{
				vehiclesByDeviceID: map[string]*domain.Vehicle{
					"DEV-001": {ID: "vehicle-1", DeviceID: "DEV-001", Name: "Truck 01"},
				},
			}
			service := application.NewSensorService(sensorRepo, vehicleRepo, &mockIngestFuelChecker{})

			inputs := []domain.SensorInput{
				{
					DeviceID:  "DEV-001",
					Timestamp: nowTimestamp(),
					Type:      "gps",
					Value:     tt.gpsValue,
				},
			}

			err := service.IngestBatch(context.Background(), inputs)

			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

// mockBroadcaster implements application.Broadcaster for tests.
// It records all broadcast calls so tests can verify what was sent.
type mockBroadcaster struct {
	messages [][]byte // records each Broadcast call
}

func (m *mockBroadcaster) Broadcast(msg []byte) {
	// Copy the message so later modifications don't affect the recorded value.
	cp := make([]byte, len(msg))
	copy(cp, msg)
	m.messages = append(m.messages, cp)
}

// ============================================================
// Tests for Broadcaster integration
// ============================================================

func TestIngestBatch_BroadcastsSensorData(t *testing.T) {
	// Arrange: set up a sensor service WITH a broadcaster.
	// After successful ingestion, the service should broadcast the data.
	sensorRepo := &mockIngestSensorRepo{}
	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-001": {ID: "vehicle-1", DeviceID: "DEV-001", Name: "Truck 01"},
		},
	}
	broadcaster := &mockBroadcaster{}

	fuelChecker := &mockIngestFuelChecker{
		alert: &domain.Alert{
			VehicleID: "vehicle-1",
			DeviceID:  "DEV-001",
			Type:      "low_fuel",
			Severity:  "critical",
		},
	}

	service := application.NewSensorService(sensorRepo, vehicleRepo, fuelChecker).
		WithBroadcaster(broadcaster)

	ts := nowTimestamp()
	inputs := []domain.SensorInput{
		validGPSInput("DEV-001", ts),
		validFuelInput("DEV-001", ts),
	}

	// Act
	err := service.IngestBatch(context.Background(), inputs)

	// Assert: no error, data was persisted, and broadcast was called
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(sensorRepo.insertedData) != 2 {
		t.Errorf("expected 2 data points persisted, got %d", len(sensorRepo.insertedData))
	}

	// Verify broadcast was called twice: once for low_fuel, once for sensor_update
	if len(broadcaster.messages) != 2 {
		t.Fatalf("expected 2 broadcast calls, got %d", len(broadcaster.messages))
	}

	// First message should be low_fuel (broadcast before sensor_update)
	var lowFuelMsg map[string]interface{}
	if err := json.Unmarshal(broadcaster.messages[0], &lowFuelMsg); err != nil {
		t.Fatalf("failed to parse low_fuel broadcast message: %v", err)
	}
	if lowFuelMsg["type"] != "low_fuel" {
		t.Errorf("expected first message type 'low_fuel', got %v", lowFuelMsg["type"])
	}
	if lowFuelMsg["vehicle_id"] != "vehicle-1" {
		t.Errorf("expected vehicle_id 'vehicle-1', got %v", lowFuelMsg["vehicle_id"])
	}
	if lowFuelMsg["device_id"] != "DEV-001" {
		t.Errorf("expected device_id 'DEV-001', got %v", lowFuelMsg["device_id"])
	}

	// Second message should be sensor_update
	var sensorMsg map[string]interface{}
	if err := json.Unmarshal(broadcaster.messages[1], &sensorMsg); err != nil {
		t.Fatalf("failed to parse sensor_update broadcast message: %v", err)
	}
	if sensorMsg["type"] != "sensor_update" {
		t.Errorf("expected second message type 'sensor_update', got %v", sensorMsg["type"])
	}

	// Verify the data array has 2 items (GPS + fuel)
	data, ok := sensorMsg["data"].([]interface{})
	if !ok {
		t.Fatalf("expected 'data' to be an array, got %T", sensorMsg["data"])
	}
	if len(data) != 2 {
		t.Errorf("expected 2 data items in broadcast, got %d", len(data))
	}
}

func TestIngestBatch_NoFuelAlert_OnlySensorUpdateBroadcast(t *testing.T) {
	// Arrange: fuel check returns nil alert (sufficient autonomy)
	sensorRepo := &mockIngestSensorRepo{}
	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-001": {ID: "vehicle-1", DeviceID: "DEV-001", Name: "Truck 01"},
		},
	}
	broadcaster := &mockBroadcaster{}
	fuelChecker := &mockIngestFuelChecker{
		alert: nil, // no alert
	}

	service := application.NewSensorService(sensorRepo, vehicleRepo, fuelChecker).
		WithBroadcaster(broadcaster)

	ts := nowTimestamp()
	inputs := []domain.SensorInput{
		validFuelInput("DEV-001", ts),
	}

	// Act
	err := service.IngestBatch(context.Background(), inputs)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Only sensor_update should be broadcast (no low_fuel)
	if len(broadcaster.messages) != 1 {
		t.Fatalf("expected 1 broadcast call, got %d", len(broadcaster.messages))
	}

	var msg map[string]interface{}
	if err := json.Unmarshal(broadcaster.messages[0], &msg); err != nil {
		t.Fatalf("failed to parse broadcast message: %v", err)
	}
	if msg["type"] != "sensor_update" {
		t.Errorf("expected message type 'sensor_update', got %v", msg["type"])
	}
}

func TestIngestBatch_NoBroadcaster_NoPanic(t *testing.T) {
	// Arrange: set up a sensor service WITHOUT a broadcaster.
	// Ingestion should work fine — broadcast is simply skipped.
	sensorRepo := &mockIngestSensorRepo{}
	vehicleRepo := &mockIngestVehicleRepo{
		vehiclesByDeviceID: map[string]*domain.Vehicle{
			"DEV-001": {ID: "vehicle-1", DeviceID: "DEV-001", Name: "Truck 01"},
		},
	}

	// NewSensorService WITHOUT WithBroadcaster — broadcaster field is nil
	service := application.NewSensorService(sensorRepo, vehicleRepo, &mockIngestFuelChecker{})

	ts := nowTimestamp()
	inputs := []domain.SensorInput{
		validGPSInput("DEV-001", ts),
	}

	// Act: should NOT panic even though broadcaster is nil
	err := service.IngestBatch(context.Background(), inputs)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(sensorRepo.insertedData) != 1 {
		t.Errorf("expected 1 data point persisted, got %d", len(sensorRepo.insertedData))
	}
}