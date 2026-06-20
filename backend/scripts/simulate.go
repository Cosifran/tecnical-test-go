// Package main provides a simulation script for the fleet monitoring system.
//
// It authenticates as admin, then sends batches of sensor data to the API
// in a loop. This is useful for testing the full pipeline: ingestion,
// fuel autonomy calculation, alert generation, and WebSocket broadcasts.
//
// Usage:
//
//	go run scripts/simulate.go                         # default settings
//	go run scripts/simulate.go -server http://localhost:8080 -iterations 50 -alert-mode
//
// Run with -alert-mode to send fuel readings that drop below the threshold,
// triggering low_fuel alerts.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"time"
)

// Config holds the simulation configuration from command-line flags.
type Config struct {
	Server     string
	Iterations int
	AlertMode  bool
}

// loginRequest is the JSON body for POST /auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginResponse is the JSON response from the login endpoint.
type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.Server, "server", "http://localhost:8080", "server URL")
	flag.IntVar(&cfg.Iterations, "iterations", 20, "number of iterations")
	flag.BoolVar(&cfg.AlertMode, "alert-mode", false, "send fuel readings that drop below threshold to trigger low_fuel alerts")
	flag.Parse()

	log.Printf("Simulator starting: server=%s iterations=%d alert-mode=%v", cfg.Server, cfg.Iterations, cfg.AlertMode)

	// Step 1: Authenticate as admin.
	token, err := login(cfg.Server)
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	log.Printf("Authenticated successfully, token: %s...", token[:20])

	// Step 2: Send sensor data in a loop.
	// Use a seeded random for reproducibility in alert-mode testing.
	rng := rand.New(rand.NewPCG(42, 100))

	// Vehicle device IDs matching the seeded data.
	deviceIDs := []string{"DEV-11111111-AAAA", "DEV-22222222-BBBB", "DEV-33333333-CCCC"}

	// Track fuel levels for gradual decrease (especially in alert mode).
	fuelLevels := []float64{80.0, 65.0, 50.0}

	for i := 0; i < cfg.Iterations; i++ {
		// Build a batch of sensor data — one GPS, one fuel, one temperature
		// per vehicle.
		var batch []map[string]interface{}

		for v, deviceID := range deviceIDs {
			// GPS reading — random coordinates near Bogotá, Colombia.
			// Center: lat 4.6097, lng -74.0817
			batch = append(batch, map[string]interface{}{
				"device_id": deviceID,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"type":      "gps",
				"value": map[string]interface{}{
					"lat": 4.6097 + rng.Float64()*0.08,
					"lng": -74.0817 + rng.Float64()*0.08,
				},
			})

			// Fuel reading — gradually decrease level in alert mode.
			fuelLevel := fuelLevels[v]
			if cfg.AlertMode {
				// Drop faster to trigger low_fuel alerts.
				fuelLevels[v] -= rng.Float64() * 5.0
				if fuelLevels[v] < 1.0 {
					fuelLevels[v] = 1.0
				}
				fuelLevel = fuelLevels[v]
			} else {
				// Normal mode: slight random variation.
				fuelLevels[v] = fuelLevels[v] - rng.Float64()*2.0
				if fuelLevels[v] < 5.0 {
					fuelLevels[v] = 5.0
				}
				fuelLevel = fuelLevels[v]
			}

			batch = append(batch, map[string]interface{}{
				"device_id": deviceID,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"type":      "fuel",
				"value": map[string]interface{}{
					"level": math.Round(fuelLevel*100) / 100,
					"unit":  "liters",
				},
			})

			// Temperature reading — random between 70-95°C (engine temps).
			batch = append(batch, map[string]interface{}{
				"device_id": deviceID,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"type":      "temperature",
				"value": map[string]interface{}{
					"celsius": math.Round((70+rng.Float64()*25)*100) / 100,
				},
			})
		}

		// Send the batch.
		body, err := json.Marshal(batch)
		if err != nil {
			log.Printf("Iteration %d: failed to marshal batch: %v", i+1, err)
			continue
		}

		req, err := http.NewRequest("POST", cfg.Server+"/api/v1/sensors/data", bytes.NewReader(body))
		if err != nil {
			log.Printf("Iteration %d: failed to create request: %v", i+1, err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("Iteration %d: request failed: %v", i+1, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusCreated {
			log.Printf("Iteration %d: OK — sent %d sensor readings", i+1, len(batch))
		} else {
			log.Printf("Iteration %d: server returned status %d", i+1, resp.StatusCode)
		}

		// Wait 5 seconds between iterations.
		if i < cfg.Iterations-1 {
			time.Sleep(5 * time.Second)
		}
	}

	log.Printf("Simulation complete: %d iterations", cfg.Iterations)
}

// login authenticates as admin and returns the access token.
func login(server string) (string, error) {
	body, err := json.Marshal(loginRequest{
		Email:    "admin@example.com",
		Password: "admin123",
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal login request: %w", err)
	}

	resp, err := http.Post(server+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login returned status %d", resp.StatusCode)
	}

	var result loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode login response: %w", err)
	}

	return result.AccessToken, nil
}
