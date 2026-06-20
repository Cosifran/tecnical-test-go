package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

type SensorDataRepo struct {
	db *sql.DB
}

func NewSensorDataRepo(db *sql.DB) *SensorDataRepo {
	return &SensorDataRepo{db: db}
}

// FindByVehicleID retrieves sensor data for a vehicle with optional time range, type, and limit filters.
func (r *SensorDataRepo) FindByVehicleID(ctx context.Context, vehicleID string, from, to time.Time, sensorType string, limit int) ([]domain.SensorData, error) {
	query := "SELECT id, vehicle_id, type, value, timestamp, created_at FROM sensor_data WHERE vehicle_id = ?"
	args := []interface{}{vehicleID}

	if !from.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, formatTime(from))
	}
	if !to.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, formatTime(to))
	}
	if sensorType != "" {
		query += " AND type = ?"
		args = append(args, sensorType)
	}

	query += " ORDER BY timestamp DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.SensorData
	for rows.Next() {
		var sd domain.SensorData
		var valueStr string
		var timestampStr string
		var createdAtStr string

		if err := rows.Scan(&sd.ID, &sd.VehicleID, &sd.Type, &valueStr, &timestampStr, &createdAtStr); err != nil {
			return nil, err
		}

		sd.Value = json.RawMessage(valueStr)
		sd.Timestamp, err = parseTime(timestampStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}
		sd.CreatedAt, err = parseTime(createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}

		results = append(results, sd)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// BulkInsert persists multiple sensor data points in a single transaction.
// If any point fails, the entire batch rolls back.
func (r *SensorDataRepo) BulkInsert(ctx context.Context, data []domain.SensorData) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	const insertSQL = "INSERT INTO sensor_data (id, vehicle_id, type, value, timestamp, created_at) VALUES (?, ?, ?, ?, ?, ?)"
	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, d := range data {
		_, err := stmt.ExecContext(ctx,
			d.ID,
			d.VehicleID,
			d.Type,
			string(d.Value),
			formatTime(d.Timestamp),
			formatTime(d.CreatedAt),
		)
		if err != nil {
			return fmt.Errorf("failed to insert sensor data point %s: %w", d.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tx = nil

	return nil
}