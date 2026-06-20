package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

type VehicleRepo struct {
	db *sql.DB
}

func NewVehicleRepo(db *sql.DB) *VehicleRepo {
	return &VehicleRepo{db: db}
}

func (r *VehicleRepo) FindByID(ctx context.Context, id string) (*domain.Vehicle, error) {
	const query = "SELECT id, device_id, name, created_at FROM vehicles WHERE id = ?"

	var v domain.Vehicle
	var createdAt string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&v.ID, &v.DeviceID, &v.Name, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	v.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func (r *VehicleRepo) FindByDeviceID(ctx context.Context, deviceID string) (*domain.Vehicle, error) {
	const query = "SELECT id, device_id, name, created_at FROM vehicles WHERE device_id = ?"

	var v domain.Vehicle
	var createdAt string
	err := r.db.QueryRowContext(ctx, query, deviceID).Scan(&v.ID, &v.DeviceID, &v.Name, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	v.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// FindAll returns all vehicles ordered by created_at descending.
func (r *VehicleRepo) FindAll(ctx context.Context) ([]domain.Vehicle, error) {
	const query = "SELECT id, device_id, name, created_at FROM vehicles ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicles []domain.Vehicle
	for rows.Next() {
		var v domain.Vehicle
		var createdAt string
		if err := rows.Scan(&v.ID, &v.DeviceID, &v.Name, &createdAt); err != nil {
			return nil, err
		}
		v.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		vehicles = append(vehicles, v)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return vehicles, nil
}

func (r *VehicleRepo) Create(ctx context.Context, vehicle *domain.Vehicle) error {
	id := generateUUID()
	now := time.Now()

	const query = "INSERT INTO vehicles (id, device_id, name, created_at) VALUES (?, ?, ?, ?)"
	_, err := r.db.ExecContext(ctx, query, id, vehicle.DeviceID, vehicle.Name, formatTime(now))
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrConflict
		}
		return err
	}

	vehicle.ID = id
	vehicle.CreatedAt = now
	return nil
}