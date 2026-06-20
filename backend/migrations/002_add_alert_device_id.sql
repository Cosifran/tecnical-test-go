-- migrations/002_add_alert_device_id.sql
-- Adds device_id column to the alerts table so alerts store/expose the IoT
-- device identifier alongside vehicle_id. Nullable initially to avoid
-- backfilling existing rows; new alerts will populate it going forward.
ALTER TABLE alerts ADD COLUMN device_id TEXT;

-- Index on alerts by device_id — used when fetching alerts for a device.
CREATE INDEX IF NOT EXISTS idx_alerts_device
    ON alerts(device_id);