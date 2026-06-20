import { apiClient } from "./apiClient";
import type {
  LoginRequest,
  LoginResponse,
  VehiclesResponse,
  HistoryResponse,
  AlertsResponse,
} from "@/types/api";

export async function login(credentials: LoginRequest): Promise<LoginResponse> {
  return apiClient<LoginResponse>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify(credentials),
    headers: { "Content-Type": "application/json" },
  });
}

export async function refreshToken(token: string): Promise<LoginResponse> {
  return apiClient<LoginResponse>("/api/v1/auth/refresh", {
    method: "POST",
    body: JSON.stringify({ refresh_token: token }),
    headers: { "Content-Type": "application/json" },
  });
}

export async function getVehicles(): Promise<VehiclesResponse> {
  return apiClient<VehiclesResponse>("/api/v1/vehicles");
}

export async function getVehicleHistory(
  vehicleId: string,
  params?: { from?: string; to?: string }
): Promise<HistoryResponse> {
  const searchParams = params
    ? "?" + new URLSearchParams(
        Object.entries(params).filter(([, v]) => v !== undefined) as [string, string][]
      ).toString()
    : "";
  return apiClient<HistoryResponse>(`/api/v1/vehicles/${vehicleId}/history${searchParams}`);
}

export async function getAlerts(): Promise<AlertsResponse> {
  return apiClient<AlertsResponse>("/api/v1/alerts");
}

// Admin/testing endpoint
export async function ingestSensorData(data: Record<string, unknown>): Promise<unknown> {
  return apiClient("/api/v1/sensors/data", {
    method: "POST",
    body: JSON.stringify(data),
    headers: { "Content-Type": "application/json" },
  });
}