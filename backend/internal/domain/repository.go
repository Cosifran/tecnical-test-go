package domain

import (
	"context"
	"time"
)

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
}

type VehicleRepository interface {
	FindByID(ctx context.Context, id string) (*Vehicle, error)
	FindByDeviceID(ctx context.Context, deviceID string) (*Vehicle, error)
	FindAll(ctx context.Context) ([]Vehicle, error)
	Create(ctx context.Context, vehicle *Vehicle) error
}

type SensorDataRepository interface {
	FindByVehicleID(ctx context.Context, vehicleID string, from, to time.Time, sensorType string, limit int) ([]SensorData, error)
	// BulkInsert is transactional: all points succeed or all fail.
	BulkInsert(ctx context.Context, data []SensorData) error
}

type AlertRepository interface {
	Create(ctx context.Context, alert *Alert) error
	FindAll(ctx context.Context) ([]Alert, error)
	FindByVehicleID(ctx context.Context, vehicleID string) ([]Alert, error)
}