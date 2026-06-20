import { describe, it, expect } from "vitest";
import type { Vehicle, VehicleStatus, VEHICLE_STATUS, SensorData, Alert } from "../types/domain";
import type {
  LoginRequest,
  LoginResponse,
  RefreshRequest,
  VehiclesResponse,
  HistoryResponse,
  AlertsResponse,
  ApiError,
} from "../types/api";

describe("domain types", () => {
  it("VEHICLE_STATUS has required values", async () => {
    const { VEHICLE_STATUS } = await import("../types/domain");
    expect(VEHICLE_STATUS.ACTIVE).toBe("active");
    expect(VEHICLE_STATUS.INACTIVE).toBe("inactive");
    expect(VEHICLE_STATUS.MAINTENANCE).toBe("maintenance");
  });

  it("Vehicle interface compatibility — creates a valid vehicle object", async () => {
    const vehicle: Vehicle = {
      id: "v1",
      device_id: "DEV-1234-ABCD",
      name: "Truck 1",
      
      created_at: "2024-01-01T00:00:00Z",
    };

    expect(vehicle.id).toBe("v1");
    expect(vehicle.device_id).toBe("DEV-1234-ABCD");
    expect(vehicle.name).toBe("Truck 1");
  });

  it("VehicleStatus type accepts only valid values", async () => {
    const { VEHICLE_STATUS } = await import("../types/domain");
    const validStatuses: VehicleStatus[] = [
      VEHICLE_STATUS.ACTIVE,
      VEHICLE_STATUS.INACTIVE,
      VEHICLE_STATUS.MAINTENANCE,
    ];

    expect(validStatuses).toHaveLength(3);
    expect(validStatuses).toContain("active");
    expect(validStatuses).toContain("inactive");
    expect(validStatuses).toContain("maintenance");
  });

  it("SensorData interface works", async () => {
    const point: SensorData = {
      ID: "s1",
      VehicleID: "v1",
      Type: "fuel",
      Value: { level: 80, unit: "liters" },
      Timestamp: "2024-01-01T00:00:00Z",
      CreatedAt: "2024-01-01T00:00:00Z",
    };
    expect(point.Type).toBe("fuel");
    expect(point.VehicleID).toBe("v1");
  });

  it("Alert interface works", () => {
    const alert: Alert = {
      id: "a1",
      vehicle_id: "v1",
      type: "low_fuel",
      severity: "critical",
      details: { estimated_minutes: 45 },
      created_at: "2024-01-01T00:00:00Z",
    };
    expect(alert.type).toBe("low_fuel");
    expect(alert.severity).toBe("critical");
  });
});

describe("api types", () => {
  it("LoginRequest type contract", () => {
    const req: LoginRequest = { email: "admin@test.com", password: "secret123" };
    expect(req.email).toBe("admin@test.com");
    expect(req.password).toBe("secret123");
  });

  it("LoginResponse type contract", () => {
    const res: LoginResponse = {
      access_token: "eyJhbGciOiJIUzI1NiJ9.eyJyb2xlIjoiYWRtaW4ifQ.xxx",
      refresh_token: "rt-abc123",
      token_type: "Bearer",
      expires_in: 3600,
    };
    expect(res.token_type).toBe("Bearer");
    expect(res.expires_in).toBe(3600);
  });

  it("VehiclesResponse type contract — wraps Vehicle array", () => {
    const res: VehiclesResponse = {
      vehicles: [
        {
          id: "v1",
          device_id: "DEV-1234-ABCD",
          name: "Truck 1",
          
          created_at: "2024-01-01T00:00:00Z",
        },
      ],
    };
    expect(res.vehicles).toHaveLength(1);
    expect(res.vehicles[0]!.id).toBe("v1");
  });

  it("ApiError type contract", () => {
    const err: ApiError = { error: "unauthorized", message: "Invalid token" };
    expect(err.error).toBe("unauthorized");
    expect(err.message).toBe("Invalid token");
  });
});
