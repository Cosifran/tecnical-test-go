package application

import "fmt"

// MaskDeviceID converts a raw device ID into its masked form: DEV-****-{last4}.
func MaskDeviceID(raw string) string {
	if len(raw) < 4 {
		return "DEV-****-????"
	}
	last4 := raw[len(raw)-4:]
	return fmt.Sprintf("DEV-****-%s", last4)
}