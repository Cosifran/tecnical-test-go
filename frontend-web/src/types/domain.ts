// Domain types mirroring backend models. Uses const + type pattern for single source of truth.

export const VEHICLE_STATUS = {
  ACTIVE: "active",
  INACTIVE: "inactive",
  MAINTENANCE: "maintenance",
} as const;

export type VehicleStatus = (typeof VEHICLE_STATUS)[keyof typeof VEHICLE_STATUS];

// Matches backend domain.Vehicle (json-tagged). Telemetry comes from SensorData.
export interface Vehicle {
  id: string;
  device_id: string;
  name: string;
  created_at: string;
}

// Matches Go domain.SensorData. Backend serializes without json tags → PascalCase keys.
export interface SensorData {
  ID: string;
  VehicleID: string;
  Type: string;
  Value: unknown;
  Timestamp: string;
  CreatedAt: string;
}

/** Computed from SensorData for charts */
export interface SensorHistoryPoint {
  timestamp: string;
  fuel_level: number;
  temperature: number;
}

export interface Alert {
  id: string;
  vehicle_id: string;
  type: string;
  severity: string;
  details: Record<string, unknown>;
  created_at: string;
}