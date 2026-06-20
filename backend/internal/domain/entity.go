package domain

import (
	"encoding/json"
	"time"
)

type User struct {
	ID        string
	Email     string
	Password  string    // bcrypt hash
	Role      string    // "admin" or "user"
	CreatedAt time.Time
}

// Vehicle represents a physical vehicle with an attached IoT device.
// DeviceID is masked for non-admin users: "DEV-12345678-ABCD" → "DEV-****-ABCD".
type Vehicle struct {
	ID        string    `json:"id"`
	DeviceID  string    `json:"device_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type SensorData struct {
	ID         string
	VehicleID  string
	Type       string           // "gps", "fuel", or "temperature"
	Value      json.RawMessage  // type-specific JSON payload
	Timestamp  time.Time
	CreatedAt  time.Time
}

type SensorInput struct {
	DeviceID  string           `json:"device_id"`
	Timestamp string           `json:"timestamp"` // RFC3339
	Type      string           `json:"type"`
	Value     json.RawMessage  `json:"value"`
}

// Alert represents a predictive alert (e.g., low fuel autonomy). Stored separately from sensor data.
type Alert struct {
	ID        string          `json:"id"`
	VehicleID string          `json:"vehicle_id"`
	DeviceID  string          `json:"device_id"`
	Type      string          `json:"type"`
	Severity  string          `json:"severity"`
	Details   json.RawMessage `json:"details"`
	CreatedAt time.Time       `json:"created_at"`
}

type GPSCoordinates struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type FuelReading struct {
	Level float64 `json:"level"`
	Unit  string  `json:"unit"`
}

type TemperatureReading struct {
	Celsius float64 `json:"celsius"`
}

type FuelReadingWithTime struct {
	Level     float64
	Timestamp time.Time
}

// Claims represents the JWT payload. Type distinguishes access tokens from refresh tokens.
type Claims struct {
	Subject  string `json:"sub"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IssuedAt int64  `json:"iat"`
	ExpireAt int64  `json:"exp"`
	Type     string `json:"type"` // "access" or "refresh"
}