package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/francisco/fleet-monitor/internal/application"
	"github.com/francisco/fleet-monitor/internal/domain"
)

// ============================================================
// Mocks for VehicleService tests
// ============================================================

// mockListVehicleRepo implements domain.VehicleRepository for tests.
type mockListVehicleRepo struct {
	vehiclesByDeviceID map[string]*domain.Vehicle
	vehiclesByID       map[string]*domain.Vehicle
	allVehicles        []domain.Vehicle
	findByDeviceIDErr error
	findByIDErr        error
	findAllErr         error
}

func (m *mockListVehicleRepo) FindByDeviceID(ctx context.Context, deviceID string) (*domain.Vehicle, error) {
	if m.findByDeviceIDErr != nil {
		return nil, m.findByDeviceIDErr
	}
	v, ok := m.vehiclesByDeviceID[deviceID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return v, nil
}

func (m *mockListVehicleRepo) FindByID(ctx context.Context, id string) (*domain.Vehicle, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	v, ok := m.vehiclesByID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return v, nil
}

func (m *mockListVehicleRepo) FindAll(ctx context.Context) ([]domain.Vehicle, error) {
	if m.findAllErr != nil {
		return nil, m.findAllErr
	}
	return m.allVehicles, nil
}

func (m *mockListVehicleRepo) Create(ctx context.Context, vehicle *domain.Vehicle) error {
	return nil
}

// mockListSensorRepo implements domain.SensorDataRepository for tests.
type mockListSensorRepo struct {
	data []domain.SensorData
	err  error
}

func (m *mockListSensorRepo) FindByVehicleID(ctx context.Context, vehicleID string, from, to time.Time, sensorType string, limit int) ([]domain.SensorData, error) {
	return m.data, m.err
}

func (m *mockListSensorRepo) BulkInsert(ctx context.Context, data []domain.SensorData) error {
	return nil
}

// ============================================================
// Test helpers
// ============================================================

// testMaskFunc is a predictable masking function for tests.
// It prefixes "MASKED-" to the input, making assertions easy.
func testMaskFunc(raw string) string {
	return "MASKED-" + raw
}

// ============================================================
// Tests for VehicleService.ListVehicles
// ============================================================

func TestListVehicles_AsAdmin_ReturnsRawDeviceIDs(t *testing.T) {
	// Arrange: vehicle repo returns two vehicles with raw device IDs
	vehicles := []domain.Vehicle{
		{ID: "v1", DeviceID: "DEV-12345678-ABCD", Name: "Truck 01", CreatedAt: time.Now()},
		{ID: "v2", DeviceID: "DEV-87654321-EFGH", Name: "Truck 02", CreatedAt: time.Now()},
	}
	vehicleRepo := &mockListVehicleRepo{allVehicles: vehicles}
	sensorRepo := &mockListSensorRepo{}

	service := application.NewVehicleService(vehicleRepo, sensorRepo, application.MaskDeviceID)

	// Act: list vehicles as admin
	result, err := service.ListVehicles(context.Background(), "admin")

	// Assert: admin should see RAW device IDs, not masked
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 vehicles, got %d", len(result))
	}
	if result[0].DeviceID != "DEV-12345678-ABCD" {
		t.Errorf("admin should see raw device ID, got %q", result[0].DeviceID)
	}
	if result[1].DeviceID != "DEV-87654321-EFGH" {
		t.Errorf("admin should see raw device ID, got %q", result[1].DeviceID)
	}
}

func TestListVehicles_AsUser_ReturnsMaskedDeviceIDs(t *testing.T) {
	// Arrange: vehicle repo returns two vehicles with raw device IDs
	vehicles := []domain.Vehicle{
		{ID: "v1", DeviceID: "DEV-12345678-ABCD", Name: "Truck 01", CreatedAt: time.Now()},
		{ID: "v2", DeviceID: "DEV-87654321-EFGH", Name: "Truck 02", CreatedAt: time.Now()},
	}
	vehicleRepo := &mockListVehicleRepo{allVehicles: vehicles}
	sensorRepo := &mockListSensorRepo{}

	service := application.NewVehicleService(vehicleRepo, sensorRepo, application.MaskDeviceID)

	// Act: list vehicles as regular user (role "user")
	result, err := service.ListVehicles(context.Background(), "user")

	// Assert: regular user should see MASKED device IDs
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 vehicles, got %d", len(result))
	}
	// MaskDeviceID("DEV-12345678-ABCD") → "DEV-****-ABCD"
	if result[0].DeviceID != "DEV-****-ABCD" {
		t.Errorf("user should see masked device ID, got %q", result[0].DeviceID)
	}
	if result[1].DeviceID != "DEV-****-EFGH" {
		t.Errorf("user should see masked device ID, got %q", result[1].DeviceID)
	}
}

func TestListVehicles_AsUser_UsesCustomMaskFunc(t *testing.T) {
	// Arrange: use the test mask function instead of MaskDeviceID
	vehicles := []domain.Vehicle{
		{ID: "v1", DeviceID: "DEV-1234", Name: "Truck 01", CreatedAt: time.Now()},
	}
	vehicleRepo := &mockListVehicleRepo{allVehicles: vehicles}
	sensorRepo := &mockListSensorRepo{}

	// Using the test mask function (prefixes "MASKED-")
	service := application.NewVehicleService(vehicleRepo, sensorRepo, testMaskFunc)

	// Act
	result, err := service.ListVehicles(context.Background(), "user")

	// Assert: the custom mask function should be applied
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result[0].DeviceID != "MASKED-DEV-1234" {
		t.Errorf("expected custom-masked device ID, got %q", result[0].DeviceID)
	}
}

// ============================================================
// Tests for VehicleService.GetVehicleHistory
// ============================================================

func TestGetVehicleHistory_ReturnsDataAndVehicle(t *testing.T) {
	// Arrange
	now := time.Now()
	vehicle := &domain.Vehicle{
		ID:        "vehicle-1",
		DeviceID:  "DEV-12345678-ABCD",
		Name:      "Truck 01",
		CreatedAt: now,
	}
	sensorData := []domain.SensorData{
		{ID: "sd-1", VehicleID: "vehicle-1", Type: "gps", Value: []byte(`{"lat":40.7,"lng":-74.0}`), Timestamp: now, CreatedAt: now},
		{ID: "sd-2", VehicleID: "vehicle-1", Type: "fuel", Value: []byte(`{"level":75.5,"unit":"liters"}`), Timestamp: now, CreatedAt: now},
	}

	vehicleRepo := &mockListVehicleRepo{
		vehiclesByID: map[string]*domain.Vehicle{"vehicle-1": vehicle},
	}
	sensorRepo := &mockListSensorRepo{data: sensorData}

	service := application.NewVehicleService(vehicleRepo, sensorRepo, application.MaskDeviceID)

	// Act: get history as admin
	from := time.Time{}
	to := time.Time{}
	data, veh, err := service.GetVehicleHistory(context.Background(), "vehicle-1", from, to, "", "admin")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(data) != 2 {
		t.Errorf("expected 2 sensor data points, got %d", len(data))
	}
	if veh.ID != "vehicle-1" {
		t.Errorf("expected vehicle ID 'vehicle-1', got %q", veh.ID)
	}
	// Admin should see the raw device ID
	if veh.DeviceID != "DEV-12345678-ABCD" {
		t.Errorf("admin should see raw device ID, got %q", veh.DeviceID)
	}
}

func TestGetVehicleHistory_NonAdmin_MasksDeviceID(t *testing.T) {
	// Arrange
	now := time.Now()
	vehicle := &domain.Vehicle{
		ID:        "vehicle-1",
		DeviceID:  "DEV-12345678-ABCD",
		Name:      "Truck 01",
		CreatedAt: now,
	}
	sensorData := []domain.SensorData{
		{ID: "sd-1", VehicleID: "vehicle-1", Type: "gps", Value: []byte(`{"lat":40.7,"lng":-74.0}`), Timestamp: now, CreatedAt: now},
	}

	vehicleRepo := &mockListVehicleRepo{
		vehiclesByID: map[string]*domain.Vehicle{"vehicle-1": vehicle},
	}
	sensorRepo := &mockListSensorRepo{data: sensorData}

	service := application.NewVehicleService(vehicleRepo, sensorRepo, application.MaskDeviceID)

	// Act: get history as regular user (role "user")
	from := time.Time{}
	to := time.Time{}
	data, veh, err := service.GetVehicleHistory(context.Background(), "vehicle-1", from, to, "", "user")

	// Assert: vehicle's device ID should be masked
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(data) != 1 {
		t.Errorf("expected 1 sensor data point, got %d", len(data))
	}
	if veh.DeviceID != "DEV-****-ABCD" {
		t.Errorf("non-admin should see masked device ID, got %q", veh.DeviceID)
	}
}

func TestGetVehicleHistory_VehicleNotFound_ErrNotFound(t *testing.T) {
	// Arrange: empty vehicle repo
	vehicleRepo := &mockListVehicleRepo{
		vehiclesByID: map[string]*domain.Vehicle{},
	}
	sensorRepo := &mockListSensorRepo{}

	service := application.NewVehicleService(vehicleRepo, sensorRepo, application.MaskDeviceID)

	// Act: try to get history for a non-existent vehicle
	from := time.Time{}
	to := time.Time{}
	data, veh, err := service.GetVehicleHistory(context.Background(), "nonexistent", from, to, "", "admin")

	// Assert: should return ErrNotFound
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data, got %v", data)
	}
	if veh != nil {
		t.Errorf("expected nil vehicle, got %v", veh)
	}
}

func TestGetVehicleHistory_SensorRepoError_ReturnsError(t *testing.T) {
	// Arrange: vehicle exists but sensor repo fails
	now := time.Now()
	vehicle := &domain.Vehicle{
		ID:        "vehicle-1",
		DeviceID:  "DEV-12345678-ABCD",
		Name:      "Truck 01",
		CreatedAt: now,
	}
	repoErr := errors.New("database unavailable")

	vehicleRepo := &mockListVehicleRepo{
		vehiclesByID: map[string]*domain.Vehicle{"vehicle-1": vehicle},
	}
	sensorRepo := &mockListSensorRepo{err: repoErr}

	service := application.NewVehicleService(vehicleRepo, sensorRepo, application.MaskDeviceID)

	// Act
	from := time.Time{}
	to := time.Time{}
	data, veh, err := service.GetVehicleHistory(context.Background(), "vehicle-1", from, to, "", "admin")

	// Assert: should propagate the sensor repo error
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repository error, got: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data, got %v", data)
	}
	if veh != nil {
		t.Errorf("expected nil vehicle, got %v", veh)
	}
}