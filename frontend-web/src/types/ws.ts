// WebSocket message types matching backend broadcast format

import type { SensorData } from "./domain";

export interface SensorUpdateMessage {
  type: "sensor_update";
  data: SensorData[];
}

export interface LowFuelMessage {
  type: "low_fuel";
  vehicle_id: string;
}

export type WsMessage = SensorUpdateMessage | LowFuelMessage;
