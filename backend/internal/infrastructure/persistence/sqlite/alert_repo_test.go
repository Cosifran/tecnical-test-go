package sqlite

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/francisco/fleet-monitor/internal/domain"
)

func TestAlertRepo_CreateAndFindAll(t *testing.T) {
	db := setupTestDB(t)
	alertRepo := NewAlertRepo(db)
	vehicleRepo := NewVehicleRepo(db)
	ctx := context.Background()

	// Create a vehicle first (alerts have FK constraint on vehicle_id)
	vehicle := &domain.Vehicle{
		DeviceID: "DEV-ALERT-TEST",
		Name:     "Alert Test Vehicle",
	}
	if err := vehicleRepo.Create(ctx, vehicle); err != nil {
		t.Fatalf("vehicle Create failed: %v", err)
	}

	alert := &domain.Alert{
		VehicleID: vehicle.ID,
		DeviceID:  vehicle.DeviceID,
		Type:      "low_fuel",
		Severity:  "critical",
		Details:   json.RawMessage(`{"autonomy_minutes": 45, "consumption_rate": 2.5}`),
	}

	if err := alertRepo.Create(ctx, alert); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if alert.ID == "" {
		t.Fatal("expected ID to be set after Create, got empty string")
	}
	if alert.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set after Create, got zero value")
	}

	alerts, err := alertRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}

	if alerts[0].ID != alert.ID {
		t.Errorf("expected ID %s, got %s", alert.ID, alerts[0].ID)
	}
	if alerts[0].VehicleID != alert.VehicleID {
		t.Errorf("expected VehicleID %s, got %s", alert.VehicleID, alerts[0].VehicleID)
	}
	if alerts[0].DeviceID != alert.DeviceID {
		t.Errorf("expected DeviceID %s, got %s", alert.DeviceID, alerts[0].DeviceID)
	}
	if alerts[0].Type != "low_fuel" {
		t.Errorf("expected type low_fuel, got %s", alerts[0].Type)
	}
	if alerts[0].Severity != "critical" {
		t.Errorf("expected severity critical, got %s", alerts[0].Severity)
	}
}

func TestAlertRepo_FindByVehicleID(t *testing.T) {
	db := setupTestDB(t)
	alertRepo := NewAlertRepo(db)
	vehicleRepo := NewVehicleRepo(db)
	ctx := context.Background()

	// Create two vehicles (need separate device IDs)
	vehicle1 := &domain.Vehicle{DeviceID: "DEV-ALERT-V1", Name: "Vehicle 1"}
	vehicle2 := &domain.Vehicle{DeviceID: "DEV-ALERT-V2", Name: "Vehicle 2"}
	if err := vehicleRepo.Create(ctx, vehicle1); err != nil {
		t.Fatalf("vehicle1 Create failed: %v", err)
	}
	if err := vehicleRepo.Create(ctx, vehicle2); err != nil {
		t.Fatalf("vehicle2 Create failed: %v", err)
	}

	alert1 := &domain.Alert{
		VehicleID: vehicle1.ID,
		DeviceID:  vehicle1.DeviceID,
		Type:      "low_fuel",
		Severity:  "critical",
		Details:   json.RawMessage(`{"autonomy_minutes": 30}`),
	}
	alert2 := &domain.Alert{
		VehicleID: vehicle1.ID,
		DeviceID:  vehicle1.DeviceID,
		Type:      "low_fuel",
		Severity:  "warning",
		Details:   json.RawMessage(`{"autonomy_minutes": 50}`),
	}
	alert3 := &domain.Alert{
		VehicleID: vehicle2.ID,
		DeviceID:  vehicle2.DeviceID,
		Type:      "low_fuel",
		Severity:  "critical",
		Details:   json.RawMessage(`{"autonomy_minutes": 20}`),
	}

	if err := alertRepo.Create(ctx, alert1); err != nil {
		t.Fatalf("alert1 Create failed: %v", err)
	}
	if err := alertRepo.Create(ctx, alert2); err != nil {
		t.Fatalf("alert2 Create failed: %v", err)
	}
	if err := alertRepo.Create(ctx, alert3); err != nil {
		t.Fatalf("alert3 Create failed: %v", err)
	}

	// Find alerts for vehicle1 only
	alerts, err := alertRepo.FindByVehicleID(ctx, vehicle1.ID)
	if err != nil {
		t.Fatalf("FindByVehicleID failed: %v", err)
	}

	if len(alerts) != 2 {
		t.Errorf("expected 2 alerts for vehicle1, got %d", len(alerts))
	}

	// Verify all alerts belong to vehicle1 and carry the right device_id
	for _, a := range alerts {
		if a.VehicleID != vehicle1.ID {
			t.Errorf("expected VehicleID %s, got %s", vehicle1.ID, a.VehicleID)
		}
		if a.DeviceID != vehicle1.DeviceID {
			t.Errorf("expected DeviceID %s, got %s", vehicle1.DeviceID, a.DeviceID)
		}
	}
}

func TestAlertRepo_FindAll_Empty(t *testing.T) {
	db := setupTestDB(t)
	alertRepo := NewAlertRepo(db)
	ctx := context.Background()

	alerts, err := alertRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts for empty table, got %d", len(alerts))
	}
}

func TestAlertRepo_FindByVehicleID_Empty(t *testing.T) {
	db := setupTestDB(t)
	alertRepo := NewAlertRepo(db)
	vehicleRepo := NewVehicleRepo(db)
	ctx := context.Background()

	vehicle := &domain.Vehicle{DeviceID: "DEV-ALERT-EMPTY", Name: "Empty Vehicle"}
	if err := vehicleRepo.Create(ctx, vehicle); err != nil {
		t.Fatalf("vehicle Create failed: %v", err)
	}

	alerts, err := alertRepo.FindByVehicleID(ctx, vehicle.ID)
	if err != nil {
		t.Fatalf("FindByVehicleID failed: %v", err)
	}

	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts for vehicle with no alerts, got %d", len(alerts))
	}
}