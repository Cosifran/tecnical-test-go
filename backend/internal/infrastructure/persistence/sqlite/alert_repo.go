package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

type AlertRepo struct {
	db *sql.DB
}

func NewAlertRepo(db *sql.DB) *AlertRepo {
	return &AlertRepo{db: db}
}

// Create inserts a new alert. Sets the generated ID and CreatedAt on the alert struct.
func (r *AlertRepo) Create(ctx context.Context, alert *domain.Alert) error {
	id := generateUUID()
	now := time.Now()

	// Handle nil Details: json.RawMessage(nil) marshals to "" which isn't valid JSON.
	details := string(alert.Details)
	if len(alert.Details) == 0 {
		details = "null"
	}

	const query = "INSERT INTO alerts (id, vehicle_id, device_id, type, severity, details, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)"
	_, err := r.db.ExecContext(ctx, query, id, alert.VehicleID, alert.DeviceID, alert.Type, alert.Severity, details, formatTime(now))
	if err != nil {
		return err
	}

	alert.ID = id
	alert.CreatedAt = now
	return nil
}

// FindAll retrieves all alerts ordered by created_at descending.
func (r *AlertRepo) FindAll(ctx context.Context) ([]domain.Alert, error) {
	const query = "SELECT id, vehicle_id, device_id, type, severity, details, created_at FROM alerts ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []domain.Alert
	for rows.Next() {
		var a domain.Alert
		var detailsStr sql.NullString
		var deviceIDStr sql.NullString
		var createdAtStr string

		if err := rows.Scan(&a.ID, &a.VehicleID, &deviceIDStr, &a.Type, &a.Severity, &detailsStr, &createdAtStr); err != nil {
			return nil, err
		}

		if deviceIDStr.Valid {
			a.DeviceID = deviceIDStr.String
		}

		if detailsStr.Valid {
			a.Details = json.RawMessage(detailsStr.String)
		} else {
			a.Details = json.RawMessage("null")
		}

		a.CreatedAt, err = parseTime(createdAtStr)
		if err != nil {
			return nil, err
		}

		alerts = append(alerts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return alerts, nil
}

func (r *AlertRepo) FindByVehicleID(ctx context.Context, vehicleID string) ([]domain.Alert, error) {
	const query = "SELECT id, vehicle_id, device_id, type, severity, details, created_at FROM alerts WHERE vehicle_id = ? ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []domain.Alert
	for rows.Next() {
		var a domain.Alert
		var detailsStr sql.NullString
		var deviceIDStr sql.NullString
		var createdAtStr string

		if err := rows.Scan(&a.ID, &a.VehicleID, &deviceIDStr, &a.Type, &a.Severity, &detailsStr, &createdAtStr); err != nil {
			return nil, err
		}

		if deviceIDStr.Valid {
			a.DeviceID = deviceIDStr.String
		}

		if detailsStr.Valid {
			a.Details = json.RawMessage(detailsStr.String)
		} else {
			a.Details = json.RawMessage("null")
		}

		a.CreatedAt, err = parseTime(createdAtStr)
		if err != nil {
			return nil, err
		}

		alerts = append(alerts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return alerts, nil
}