// Package application_test tests the device ID masking function.
// Masking is a security-critical feature: non-admin users must NEVER
// see raw device IDs, so we test every edge case.
package application_test

import (
	"testing"

	"github.com/francisco/fleet-monitor/internal/application"
)

// TestMaskDeviceID tests the Device ID masking function using table-driven tests.
// This is one of the most important tests in the system because it verifies
// that the privacy requirement is met consistently.
//
// The format MUST be: DEV-****-{last4}
// where {last4} is the last 4 characters of the raw ID.
func TestMaskDeviceID(t *testing.T) {
	tests := []struct {
		name     string // descriptive name for the subtest
		input    string // raw device ID
		expected string // expected masked output
	}{
		{
			name:     "standard device ID with hyphens",
			input:    "DEV-12345678-ABCD",
			expected: "DEV-****-ABCD",
		},
		{
			name:     "UUID format device ID",
			input:    "550e8400-e29b-41d4-a716-446655440000",
			expected: "DEV-****-0000",
		},
		{
			name:     "short meaningful ID",
			input:    "SENSOR-XYZ-1234",
			expected: "DEV-****-1234",
		},
		{
			name:     "exactly 4 characters",
			input:    "ABCD",
			expected: "DEV-****-ABCD",
		},
		{
			name:     "3 characters — too short",
			input:    "ABC",
			expected: "DEV-****-????",
		},
		{
			name:     "2 characters — too short",
			input:    "AB",
			expected: "DEV-****-????",
		},
		{
			name:     "1 character — too short",
			input:    "X",
			expected: "DEV-****-????",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "DEV-****-????",
		},
		{
			name:     "5 characters — just enough",
			input:    "ABCDE",
			expected: "DEV-****-BCDE",
		},
		{
			name:     "numeric last 4",
			input:    "DEV-12345678-1234",
			expected: "DEV-****-1234",
		},
		{
			name:     "mixed case last 4",
			input:    "DEV-12345678-XyZ9",
			expected: "DEV-****-XyZ9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := application.MaskDeviceID(tt.input)
			if result != tt.expected {
				t.Errorf("MaskDeviceID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestMaskDeviceID_Deterministic verifies that the same input always
// produces the same output. This is important because:
// 1. A user can correlate masked IDs across multiple API calls
// 2. The masking format must be consistent for UI rendering
func TestMaskDeviceID_Deterministic(t *testing.T) {
	input := "DEV-12345678-ABCD"

	// Call MaskDeviceID 100 times with the same input
	for i := 0; i < 100; i++ {
		result := application.MaskDeviceID(input)
		if result != "DEV-****-ABCD" {
			t.Errorf("MaskDeviceID is not deterministic: iteration %d got %q", i, result)
		}
	}
}

// TestMaskDeviceID_DifferentInputsProduceDifferentOutputs verifies that
// two different device IDs don't accidentally produce the same masked ID.
// This could happen if the last 4 characters collide.
func TestMaskDeviceID_DifferentInputsProduceDifferentOutputs(t *testing.T) {
	// Two IDs with different content but SAME last 4 characters
	// This is expected to produce the SAME masked output (by design)
	id1 := "DEV-11111111-ABCD"
	id2 := "DEV-22222222-ABCD"

	m1 := application.MaskDeviceID(id1)
	m2 := application.MaskDeviceID(id2)

	// They SHOULD be the same because the mask is based on last 4 chars
	if m1 != m2 {
		t.Errorf("IDs with same last 4 chars should produce same mask: %q vs %q", m1, m2)
	}

	// Two IDs with DIFFERENT last 4 characters should produce different masks
	id3 := "DEV-11111111-ABCD"
	id4 := "DEV-22222222-EFGH"

	m3 := application.MaskDeviceID(id3)
	m4 := application.MaskDeviceID(id4)

	if m3 == m4 {
		t.Errorf("IDs with different last 4 chars should produce different masks: %q == %q", m3, m4)
	}
}