package sqlite

import (
	"context"
	"testing"

	"github.com/francisco/fleet-monitor/internal/domain"
)

func TestVehicleRepo_CreateAndFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleRepo(db)
	ctx := context.Background()

	vehicle := &domain.Vehicle{
		DeviceID: "DEV-12345678-ABCD",
		Name:     "Truck 01",
	}

	if err := repo.Create(ctx, vehicle); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if vehicle.ID == "" {
		t.Fatal("expected ID to be set after Create, got empty string")
	}
	if vehicle.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set after Create, got zero value")
	}

	found, err := repo.FindByID(ctx, vehicle.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if found.DeviceID != vehicle.DeviceID {
		t.Errorf("expected device_id %s, got %s", vehicle.DeviceID, found.DeviceID)
	}
	if found.Name != vehicle.Name {
		t.Errorf("expected name %s, got %s", vehicle.Name, found.Name)
	}
	if found.ID != vehicle.ID {
		t.Errorf("expected ID %s, got %s", vehicle.ID, found.ID)
	}
}

func TestVehicleRepo_FindByDeviceID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleRepo(db)
	ctx := context.Background()

	vehicle := &domain.Vehicle{
		DeviceID: "DEV-98765432-EFGH",
		Name:     "Van 02",
	}

	if err := repo.Create(ctx, vehicle); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := repo.FindByDeviceID(ctx, "DEV-98765432-EFGH")
	if err != nil {
		t.Fatalf("FindByDeviceID failed: %v", err)
	}

	if found.ID != vehicle.ID {
		t.Errorf("expected ID %s, got %s", vehicle.ID, found.ID)
	}
	if found.Name != vehicle.Name {
		t.Errorf("expected name %s, got %s", vehicle.Name, found.Name)
	}
}

func TestVehicleRepo_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleRepo(db)
	ctx := context.Background()

	vehicles := []struct {
		deviceID, name string
	}{
		{"DEV-11111111-AAAA", "Truck 01"},
		{"DEV-22222222-BBBB", "Van 02"},
		{"DEV-33333333-CCCC", "Car 03"},
	}

	for _, v := range vehicles {
		if err := repo.Create(ctx, &domain.Vehicle{DeviceID: v.deviceID, Name: v.name}); err != nil {
			t.Fatalf("Create failed for %s: %v", v.name, err)
		}
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("expected 3 vehicles, got %d", len(all))
	}

	// Verify ordering: newest first (created_at DESC)
	for i := 1; i < len(all); i++ {
		if all[i].CreatedAt.After(all[i-1].CreatedAt) {
			t.Errorf("vehicles not ordered by created_at DESC: %v should come after %v", all[i].CreatedAt, all[i-1].CreatedAt)
		}
	}
}

func TestVehicleRepo_DuplicateDeviceID_ErrConflict(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleRepo(db)
	ctx := context.Background()

	v1 := &domain.Vehicle{DeviceID: "DEV-DUP-1234", Name: "Vehicle 1"}
	v2 := &domain.Vehicle{DeviceID: "DEV-DUP-1234", Name: "Vehicle 2"}

	if err := repo.Create(ctx, v1); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	err := repo.Create(ctx, v2)
	if err == nil {
		t.Fatal("expected ErrConflict for duplicate device_id, got nil")
	}
	if err != domain.ErrConflict {
		t.Errorf("expected domain.ErrConflict, got %v", err)
	}
}

func TestVehicleRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleRepo(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "nonexistent-id")
	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestVehicleRepo_FindByDeviceID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVehicleRepo(db)
	ctx := context.Background()

	_, err := repo.FindByDeviceID(ctx, "DEV-NONEXISTENT")
	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}