// Package application provides business logic utilities.
// This file implements device ID masking per the spec requirement:
// non-admin users must see masked device IDs, not the raw values.
//
// WHY handler-level masking: The domain and persistence layers always store
// and work with raw device IDs. Masking is a PRESENTATION concern — it
// changes based on WHO is looking at the data, not what the data IS.
// By keeping this function in the application layer, we keep it testable
// and decoupled from HTTP response serialization.
package application

import "fmt"

// MaskDeviceID converts a raw device ID into its masked form.
// The format is: DEV-****-{last4}
// where {last4} is the last 4 characters of the raw ID.
//
// Examples:
//   - "DEV-12345678-ABCD" → "DEV-****-ABCD"
//   - "SENSOR-XYZ-1234"   → "DEV-****-1234"
//   - "AB"                → "DEV-****-????" (too short to extract last 4)
//
// WHY deterministic: The same raw ID always produces the same masked ID.
// This ensures consistency across multiple API calls — a user can still
// correlate observations without seeing the full ID.
//
// Edge case: If the raw ID has fewer than 4 characters, we use "????"
// as the suffix. This is unlikely in production (device IDs are typically
// UUIDs or formatted identifiers) but makes the function safe for any input.
func MaskDeviceID(raw string) string {
	if len(raw) < 4 {
		return "DEV-****-????"
	}
	last4 := raw[len(raw)-4:]
	return fmt.Sprintf("DEV-****-%s", last4)
}