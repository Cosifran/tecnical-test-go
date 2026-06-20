package sqlite

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate random ID: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// isUniqueConstraintError checks whether a database error resulted from a UNIQUE constraint violation.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// parseTime parses a timestamp string. Tries RFC3339 first (Go-issued), then SQLite's datetime format.
func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05", s)
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

var (
	_ domain.UserRepository        = (*UserRepo)(nil)
	_ domain.VehicleRepository    = (*VehicleRepo)(nil)
	_ domain.SensorDataRepository = (*SensorDataRepo)(nil)
	_ domain.AlertRepository       = (*AlertRepo)(nil)
)