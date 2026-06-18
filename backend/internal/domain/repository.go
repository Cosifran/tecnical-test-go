// Package domain defines the repository interfaces (contracts) that the
// application layer depends on. The infrastructure layer will provide
// concrete implementations (SQLite repos).
//
// WHY interfaces instead of direct struct calls: This is the Dependency
// Inversion Principle (the "D" in SOLID). The application layer defines
// WHAT it needs (contracts), and the infrastructure layer provides HOW
// to fulfill those needs. This means:
//   1. We can test services with mock implementations (no database needed)
//   2. We can swap SQLite for PostgreSQL by writing a new repo implementation
//   3. The business rules don't care about storage details
//
// Go interfaces are satisfied implicitly — if a struct has all the methods,
// it automatically "implements" the interface. No "implements" keyword needed.
package domain

import (
	"context"
	"time"
)

// UserRepository defines the contract for user persistence.
// The application layer uses this to look up users for authentication.
type UserRepository interface {
	// FindByID retrieves a user by their unique ID.
	// Returns ErrNotFound if no user exists with the given ID.
	FindByID(ctx context.Context, id string) (*User, error)

	// FindByEmail retrieves a user by their email address.
	// Used during login to look up credentials.
	// Returns ErrNotFound if no user exists with the given email.
	FindByEmail(ctx context.Context, email string) (*User, error)

	// Create inserts a new user into the database.
	// Returns ErrConflict if a user with the same email already exists.
	Create(ctx context.Context, user *User) error
}

// VehicleRepository defines the contract for vehicle persistence.
type VehicleRepository interface {
	// FindByID retrieves a vehicle by its unique ID.
	// Returns ErrNotFound if no vehicle exists.
	FindByID(ctx context.Context, id string) (*Vehicle, error)

	// FindByDeviceID retrieves a vehicle by its device ID.
	// Used during sensor data ingestion to resolve device_id → vehicle_id.
	// Returns ErrNotFound if no vehicle exists with the given device_id.
	FindByDeviceID(ctx context.Context, deviceID string) (*Vehicle, error)

	// FindAll returns all vehicles in the system.
	// For a fleet monitoring dashboard, we typically show all vehicles.
	FindAll(ctx context.Context) ([]Vehicle, error)

	// Create inserts a new vehicle into the database.
	// Returns ErrConflict if a vehicle with the same device_id exists.
	Create(ctx context.Context, vehicle *Vehicle) error
}

// SensorDataRepository defines the contract for sensor data persistence.
// The BulkInsert method is transactional: all points succeed or all fail.
type SensorDataRepository interface {
	// FindByVehicleID retrieves sensor data for a vehicle, with optional
	// time range and type filters. This supports the history endpoint
	// and the fuel calculation (which fetches last N fuel readings).
	//
	// Parameters:
	//   - from, to: optional time range bounds (zero value = no filter)
	//   - sensorType: optional type filter (empty string = all types)
	FindByVehicleID(ctx context.Context, vehicleID string, from, to time.Time, sensorType string) ([]SensorData, error)

	// BulkInsert persists multiple sensor data points in a single transaction.
	// If any point fails validation at the DB level, the entire batch rolls back.
	// This implements the spec requirement: "reject the entire batch if any
	// data point fails."
	BulkInsert(ctx context.Context, data []SensorData) error
}

// AlertRepository defines the contract for alert persistence.
type AlertRepository interface {
	// Create inserts a new alert into the database.
	// Called by the fuel service when autonomy drops below 1 hour.
	Create(ctx context.Context, alert *Alert) error

	// FindAll retrieves all alerts, ordered by created_at descending
	// (newest first). Admin users see all alerts; regular users
	// see an empty list (filtered at the handler level, not here).
	FindAll(ctx context.Context) ([]Alert, error)

	// FindByVehicleID retrieves alerts for a specific vehicle.
	// Useful for viewing a vehicle's alert history.
	FindByVehicleID(ctx context.Context, vehicleID string) ([]Alert, error)
}