package application

import (
	"context"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

// VehicleService handles vehicle listing and sensor history retrieval.
// Non-admin users see masked device IDs; admins see raw IDs.
type VehicleService struct {
	vehicleRepo domain.VehicleRepository
	sensorRepo domain.SensorDataRepository
	maskFunc func(string) string
}

func NewVehicleService(
	vehicleRepo domain.VehicleRepository,
	sensorRepo domain.SensorDataRepository,
	maskFunc func(string) string,
) *VehicleService {
	return &VehicleService{
		vehicleRepo: vehicleRepo,
		sensorRepo:  sensorRepo,
		maskFunc:    maskFunc,
	}
}

// ListVehicles returns all vehicles, masking device IDs for non-admin users.
func (s *VehicleService) ListVehicles(ctx context.Context, userRole string) ([]domain.Vehicle, error) {
	vehicles, err := s.vehicleRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	if userRole != "admin" {
		for i := range vehicles {
			vehicles[i].DeviceID = s.maskFunc(vehicles[i].DeviceID)
		}
	}

	return vehicles, nil
}

// GetVehicleHistory retrieves sensor data history and vehicle details for a specific vehicle.
func (s *VehicleService) GetVehicleHistory(
	ctx context.Context,
	vehicleID string,
	from, to time.Time,
	sensorType string,
	userRole string,
) ([]domain.SensorData, *domain.Vehicle, error) {
	vehicle, err := s.vehicleRepo.FindByID(ctx, vehicleID)
	if err != nil {
		return nil, nil, err
	}

	sensorData, err := s.sensorRepo.FindByVehicleID(ctx, vehicleID, from, to, sensorType, 0)
	if err != nil {
		return nil, nil, err
	}

	if userRole != "admin" {
		vehicle.DeviceID = s.maskFunc(vehicle.DeviceID)
	}

	return sensorData, vehicle, nil
}