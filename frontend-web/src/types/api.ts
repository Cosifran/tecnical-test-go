// API request/response types matching backend endpoints

import type { Vehicle, SensorData, Alert } from "./domain";

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  token_type: "Bearer";
  expires_in: number;
}

export interface RefreshRequest {
  refresh_token: string;
}

export interface VehiclesResponse {
  vehicles: Vehicle[];
}

export interface HistoryResponse {
  vehicle: Vehicle;
  history: SensorData[];
}

export interface AlertsResponse {
  alerts: Alert[];
}

export interface ApiError {
  error: string;
  message: string;
}