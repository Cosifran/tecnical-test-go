-- migrations/001_init.sql
-- Initial schema for the IoT Fleet Monitoring system.
-- Uses SQLite — that's why we store UUIDs and timestamps as TEXT.
-- SQLite has no native UUID or datetime type, but TEXT works well:
-- ISO 8601 strings sort lexicographically (= chronologically),
-- and UUIDs are just strings with well-known format.

-- Users table: stores account information for authentication.
-- The 'role' column uses a CHECK constraint to enforce only 'admin' or 'user'.
-- This is simpler than a separate roles table for a 3-day test.
CREATE TABLE IF NOT EXISTS users (
    id         TEXT PRIMARY KEY,          -- UUID as text (generated in Go code)
    email      TEXT NOT NULL UNIQUE,      -- unique login identifier
    password   TEXT NOT NULL,             -- bcrypt hash, never stored plaintext
    role       TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))  -- ISO 8601 string
);

-- Vehicles table: maps physical IoT devices to fleet vehicles.
-- device_id is the unique hardware identifier on the IoT device.
-- This is what gets MASKED for non-admin users (spec requirement).
CREATE TABLE IF NOT EXISTS vehicles (
    id         TEXT PRIMARY KEY,
    device_id  TEXT NOT NULL UNIQUE,      -- e.g., "DEV-12345678-ABCD"
    name       TEXT NOT NULL,             -- human-readable name like "Truck 01"
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Sensor data table: stores GPS, fuel, and temperature readings.
-- The 'value' column stores JSON as text — SQLite supports json_extract()
-- for querying, but we validate shapes in Go code, not in SQL.
-- The foreign key on vehicle_id ensures data integrity.
CREATE TABLE IF NOT EXISTS sensor_data (
    id         TEXT PRIMARY KEY,
    vehicle_id TEXT NOT NULL REFERENCES vehicles(id),
    type       TEXT NOT NULL CHECK (type IN ('gps', 'fuel', 'temperature')),
    value      TEXT NOT NULL,             -- JSON payload, validated in Go
    timestamp  TEXT NOT NULL,             -- ISO 8601 from the device (not server time)
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Alerts table: stores predictive alerts (currently low_fuel, extensible).
-- The 'severity' column defaults to 'critical' since all current alerts are critical.
-- The 'details' column stores JSON with calculation results (autonomy, rate, etc.).
CREATE TABLE IF NOT EXISTS alerts (
    id         TEXT PRIMARY KEY,
    vehicle_id TEXT NOT NULL REFERENCES vehicles(id),
    type       TEXT NOT NULL,             -- e.g., 'low_fuel'
    severity   TEXT NOT NULL DEFAULT 'critical',
    details    TEXT,                      -- JSON with calculation details
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Performance indexes:
-- WHY: These indexes match the most common query patterns in the spec.
-- sensor_data queries always filter by vehicle and often by type + time range.
-- alerts queries filter by vehicle or by type.

-- This index speeds up: "get recent sensor data for vehicle X"
-- which is used by the fuel calculation service (fetching last N readings).
CREATE INDEX IF NOT EXISTS idx_sensor_data_vehicle_timestamp
    ON sensor_data(vehicle_id, timestamp DESC);

-- This index speeds up: "get fuel readings for vehicle X in time range"
-- which combines vehicle, type, and time — exactly what the fuel service needs.
CREATE INDEX IF NOT EXISTS idx_sensor_data_vehicle_type_timestamp
    ON sensor_data(vehicle_id, type, timestamp DESC);

-- Index on alerts by vehicle — used when fetching alerts for a specific vehicle.
CREATE INDEX IF NOT EXISTS idx_alerts_vehicle
    ON alerts(vehicle_id);

-- Index on alerts by type — used when filtering alerts (e.g., low_fuel only).
CREATE INDEX IF NOT EXISTS idx_alerts_type
    ON alerts(type);