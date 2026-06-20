package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

// createTestVehicleForSensor creates a vehicle via VehicleRepo and returns it.
// This is needed because sensor_data has a FK constraint on vehicles(id).
func createTestVehicleForSensor(t *testing.T, db *sql.DB) *domain.Vehicle {
	t.Helper()
	repo := NewVehicleRepo(db)
	ctx := context.Background()
	vehicle := &domain.Vehicle{
		DeviceID: "DEV-SENSOR-TEST",
		Name:     "Sensor Test Vehicle",
	}
	if err := repo.Create(ctx, vehicle); err != nil {
		t.Fatalf("failed to create test vehicle: %v", err)
	}
	return vehicle
}

func TestSensorDataRepo_BulkInsert(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSensorDataRepo(db)
	ctx := context.Background()

	vehicle := createTestVehicleForSensor(t, db)

	data := []domain.SensorData{
		{
			ID:        "sensor-1",
			VehicleID: vehicle.ID,
			Type:      "fuel",
			Value:     json.RawMessage(`{"level": 45.0, "unit": "liters"}`),
			Timestamp: time.Now().Add(-1 * time.Hour),
			CreatedAt: time.Now(),
		},
		{
			ID:        "sensor-2",
			VehicleID: vehicle.ID,
			Type:      "gps",
			Value:     json.RawMessage(`{"lat": -34.6, "lng": -58.4}`),
			Timestamp: time.Now().Add(-30 * time.Minute),
			CreatedAt: time.Now(),
		},
	}

	if err := repo.BulkInsert(ctx, data); err != nil {
		t.Fatalf("BulkInsert failed: %v", err)
	}

	// Verify we can read the data back
	results, err := repo.FindByVehicleID(ctx, vehicle.ID, time.Time{}, time.Time{}, "", 0)
	if err != nil {
		t.Fatalf("FindByVehicleID failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 sensor data points, got %d", len(results))
	}
}

func TestSensorDataRepo_FindByVehicleID_NoFilters(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSensorDataRepo(db)
	ctx := context.Background()

	vehicle := createTestVehicleForSensor(t, db)

	data := []domain.SensorData{
		{
			ID:        "fuel-old",
			VehicleID: vehicle.ID,
			Type:      "fuel",
			Value:     json.RawMessage(`{"level": 50.0, "unit": "liters"}`),
			Timestamp: time.Now().Add(-2 * time.Hour),
			CreatedAt: time.Now(),
		},
		{
			ID:        "gps-mid",
			VehicleID: vehicle.ID,
			Type:      "gps",
			Value:     json.RawMessage(`{"lat": -34.6, "lng": -58.4}`),
			Timestamp: time.Now().Add(-1 * time.Hour),
			CreatedAt: time.Now(),
		},
		{
			ID:        "fuel-new",
			VehicleID: vehicle.ID,
			Type:      "fuel",
			Value:     json.RawMessage(`{"level": 40.0, "unit": "liters"}`),
			Timestamp: time.Now().Add(-10 * time.Minute),
			CreatedAt: time.Now(),
		},
	}

	if err := repo.BulkInsert(ctx, data); err != nil {
		t.Fatalf("BulkInsert failed: %v", err)
	}

	// No filters — should return all 3 points for this vehicle
	results, err := repo.FindByVehicleID(ctx, vehicle.ID, time.Time{}, time.Time{}, "", 0)
	if err != nil {
		t.Fatalf("FindByVehicleID failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestSensorDataRepo_FindByVehicleID_FilterByType(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSensorDataRepo(db)
	ctx := context.Background()

	vehicle := createTestVehicleForSensor(t, db)

	data := []domain.SensorData{
		{
			ID:        "fuel-1",
			VehicleID: vehicle.ID,
			Type:      "fuel",
			Value:     json.RawMessage(`{"level": 50.0, "unit": "liters"}`),
			Timestamp: time.Now().Add(-2 * time.Hour),
			CreatedAt: time.Now(),
		},
		{
			ID:        "gps-1",
			VehicleID: vehicle.ID,
			Type:      "gps",
			Value:     json.RawMessage(`{"lat": -34.6, "lng": -58.4}`),
			Timestamp: time.Now().Add(-1 * time.Hour),
			CreatedAt: time.Now(),
		},
		{
			ID:        "fuel-2",
			VehicleID: vehicle.ID,
			Type:      "fuel",
			Value:     json.RawMessage(`{"level": 40.0, "unit": "liters"}`),
			Timestamp: time.Now().Add(-30 * time.Minute),
			CreatedAt: time.Now(),
		},
	}

	if err := repo.BulkInsert(ctx, data); err != nil {
		t.Fatalf("BulkInsert failed: %v", err)
	}

	// Filter by type = "fuel"
	results, err := repo.FindByVehicleID(ctx, vehicle.ID, time.Time{}, time.Time{}, "fuel", 0)
	if err != nil {
		t.Fatalf("FindByVehicleID with type filter failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 fuel results, got %d", len(results))
	}

	for _, r := range results {
		if r.Type != "fuel" {
			t.Errorf("expected type fuel, got %s", r.Type)
		}
	}
}

func TestSensorDataRepo_FindByVehicleID_FilterByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSensorDataRepo(db)
	ctx := context.Background()

	vehicle := createTestVehicleForSensor(t, db)

	now := time.Now()
	data := []domain.SensorData{
		{
			ID:        "old",
			VehicleID: vehicle.ID,
			Type:      "fuel",
			Value:     json.RawMessage(`{"level": 50.0, "unit": "liters"}`),
			Timestamp: now.Add(-2 * time.Hour), // 2 hours ago
			CreatedAt: now,
		},
		{
			ID:        "mid",
			VehicleID: vehicle.ID,
			Type:      "gps",
			Value:     json.RawMessage(`{"lat": -34.6, "lng": -58.4}`),
			Timestamp: now.Add(-1 * time.Hour), // 1 hour ago
			CreatedAt: now,
		},
		{
			ID:        "recent",
			VehicleID: vehicle.ID,
			Type:      "fuel",
			Value:     json.RawMessage(`{"level": 40.0, "unit": "liters"}`),
			Timestamp: now.Add(-5 * time.Minute), // 5 minutes ago
			CreatedAt: now,
		},
	}

	if err := repo.BulkInsert(ctx, data); err != nil {
		t.Fatalf("BulkInsert failed: %v", err)
	}

	// Filter: from = 90 minutes ago (should exclude the 2-hour-old point)
	from := now.Add(-90 * time.Minute)
	results, err := repo.FindByVehicleID(ctx, vehicle.ID, from, time.Time{}, "", 0)
	if err != nil {
		t.Fatalf("FindByVehicleID with time filter failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results in time range, got %d", len(results))
	}
}

func TestSensorDataRepo_FindByVehicleID_Limit(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSensorDataRepo(db)
	ctx := context.Background()

	vehicle := createTestVehicleForSensor(t, db)

	now := time.Now()
	data := make([]domain.SensorData, 5)
	for i := 0; i < 5; i++ {
		data[i] = domain.SensorData{
			ID:        fmt.Sprintf("sensor-%d", i),
			VehicleID: vehicle.ID,
			Type:      "fuel",
			Value:     json.RawMessage(`{"level": 50.0, "unit": "liters"}`),
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			CreatedAt: now,
		}
	}

	if err := repo.BulkInsert(ctx, data); err != nil {
		t.Fatalf("BulkInsert failed: %v", err)
	}

	// Limit to 3 results
	results, err := repo.FindByVehicleID(ctx, vehicle.ID, time.Time{}, time.Time{}, "", 3)
	if err != nil {
		t.Fatalf("FindByVehicleID with limit failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results with limit, got %d", len(results))
	}
}

func TestSensorDataRepo_FindByVehicleID_EmptyResult(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSensorDataRepo(db)
	ctx := context.Background()

	vehicle := createTestVehicleForSensor(t, db)

	// No sensor data inserted — should return empty slice
	results, err := repo.FindByVehicleID(ctx, vehicle.ID, time.Time{}, time.Time{}, "", 0)
	if err != nil {
		t.Fatalf("FindByVehicleID failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty vehicle, got %d", len(results))
	}
}