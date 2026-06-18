package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

type mockSensorRepo struct {
	data []domain.SensorData
	err  error
}

func (m *mockSensorRepo) FindByVehicleID(ctx context.Context, vehicleID string, from, to time.Time, sensorType string) ([]domain.SensorData, error) {
	return m.data, m.err
}

func (m *mockSensorRepo) BulkInsert(ctx context.Context, data []domain.SensorData) error {
	return nil
}

// Mock del repositorio de alertas
type mockAlertRepo struct {
	created []*domain.Alert
	err     error
}

func (m *mockAlertRepo) Create(ctx context.Context, alert *domain.Alert) error {
	if m.err != nil {
		return m.err
	}
	m.created = append(m.created, alert)
	return nil
}

func (m *mockAlertRepo) FindAll(ctx context.Context) ([]domain.Alert, error) {
	return nil, nil
}

func (m *mockAlertRepo) FindByVehicleID(ctx context.Context, vehicleID string) ([]domain.Alert, error) {
	return nil, nil
}

func makeFuelData(levelStart, consumptionPerHour float64, count int) []domain.SensorData {
	now := time.Now()
	data := make([]domain.SensorData, 0, count)

	for i := 0; i < count; i++ {
		// Cada lectura es 1 hora antes que la anterior (más antigua primero)
		timestamp := now.Add(-time.Duration(count-1-i) * time.Hour)
		level := levelStart - float64(i)*consumptionPerHour

		value, _ := json.Marshal(domain.FuelReading{
			Level: level,
			Unit:  "liters",
		})

		data = append(data, domain.SensorData{
			ID:        "sensor-" + string(rune('A'+i)),
			VehicleID: "vehicle-1",
			Type:      "fuel",
			Value:     value,
			Timestamp: timestamp,
		})
	}
	return data
}

func TestCheckAutonomy_LessThanThreeReadings_NoAlert(t *testing.T) {
	// Arrange: solo 2 lecturas
	sensorRepo := &mockSensorRepo{
		data: makeFuelData(50, 5, 2), // 2 lecturas, 5L/h
	}
	alertRepo := &mockAlertRepo{}
	service := NewFuelService(sensorRepo, alertRepo)

	// Act
	err := service.CheckAutonomy(context.Background(), "vehicle-1", 10)

	// Assert
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(alertRepo.created) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alertRepo.created))
	}
}

func TestCheckAutonomy_HighConsumption_TriggersAlert(t *testing.T) {
	// Arrange: consume 5L/h, nivel actual = 4L → autonomía = 0.8h (< 1)
	sensorRepo := &mockSensorRepo{
		data: makeFuelData(24, 5, 5), // 5 lecturas: 24,19,14,9,4 (consume 5L/h)
	}
	alertRepo := &mockAlertRepo{}
	service := NewFuelService(sensorRepo, alertRepo)

	// Act
	err := service.CheckAutonomy(context.Background(), "vehicle-1", 10)

	// Assert
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(alertRepo.created) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alertRepo.created))
	}

	alert := alertRepo.created[0]
	if alert.Type != "low_fuel" {
		t.Errorf("expected alert type 'low_fuel', got %s", alert.Type)
	}
	if alert.Severity != "critical" {
		t.Errorf("expected severity 'critical', got %s", alert.Severity)
	}
	if alert.VehicleID != "vehicle-1" {
		t.Errorf("expected vehicleID 'vehicle-1', got %s", alert.VehicleID)
	}
}

func TestCheckAutonomy_LowConsumption_NoAlert(t *testing.T) {
	// Arrange: consume 2L/h, nivel actual = 10L → autonomía = 5h (>= 1)
	sensorRepo := &mockSensorRepo{
		data: makeFuelData(18, 2, 5), // 5 lecturas: 18,16,14,12,10 (consume 2L/h)
	}
	alertRepo := &mockAlertRepo{}
	service := NewFuelService(sensorRepo, alertRepo)
	
	// Act
	err := service.CheckAutonomy(context.Background(), "vehicle-1", 10)
	
	// Assert
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(alertRepo.created) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alertRepo.created))
	}
}
func TestCheckAutonomy_ZeroRate_NoAlert(t *testing.T) {
	// Arrange: consume 0L/h (nivel no cambia) → rate = 0
	sensorRepo := &mockSensorRepo{
		data: makeFuelData(50, 0, 5), // 5 lecturas todas de 50L
	}
	alertRepo := &mockAlertRepo{}
	service := NewFuelService(sensorRepo, alertRepo)
	
	// Act
	err := service.CheckAutonomy(context.Background(), "vehicle-1", 10)
	
	// Assert
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(alertRepo.created) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alertRepo.created))
	}
}
func TestCheckAutonomy_NegativeRate_NoAlert(t *testing.T) {
	// Arrange: consume -2L/h (está cargando combustible) → rate < 0
	sensorRepo := &mockSensorRepo{
		data: makeFuelData(10, -2, 5), // 5 lecturas: 10,12,14,16,18 (cargando)
	}
	alertRepo := &mockAlertRepo{}
	service := NewFuelService(sensorRepo, alertRepo)
	
	// Act
	err := service.CheckAutonomy(context.Background(), "vehicle-1", 10)
	
	// Assert
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(alertRepo.created) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alertRepo.created))
	}
}
func TestCheckAutonomy_RepoError_ReturnsError(t *testing.T) {
	// Arrange: el repo retorna error
	sensorRepo := &mockSensorRepo{
		err: domain.ErrNotFound,
	}
	alertRepo := &mockAlertRepo{}
	service := NewFuelService(sensorRepo, alertRepo)
	
	// Act
	err := service.CheckAutonomy(context.Background(), "vehicle-1", 10)
	
	// Assert
	if err == nil {
		t.Error("expected error, got nil")
	}
	if len(alertRepo.created) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alertRepo.created))
	}
}
