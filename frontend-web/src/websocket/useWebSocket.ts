import { useEffect } from "react";
import { WebSocketManager } from "@/websocket/WebSocketManager";
import { useWStore } from "@/stores/wsStore";
import { useAuthStore } from "@/stores/authStore";
import { useFleetStore } from "@/stores/fleetStore";
import { useToastStore } from "@/stores/toastStore";
import type { SensorUpdateMessage, LowFuelMessage } from "@/types/ws";

const WS_URL = import.meta.env.VITE_WS_URL || "/api/v1/ws";

interface GPSValue {
  lat: number;
  lng: number;
}

function isGPSValue(v: unknown): v is GPSValue {
  return (
    typeof v === "object" &&
    v !== null &&
    "lat" in v &&
    "lng" in v &&
    typeof (v as Record<string, unknown>).lat === "number" &&
    typeof (v as Record<string, unknown>).lng === "number"
  );
}

export function useWebSocket() {
  const connected = useWStore((s) => s.connected);
  const lastMessage = useWStore((s) => s.lastMessage);

  useEffect(() => {
    const accessToken = useAuthStore.getState().accessToken;
    if (!accessToken) return;

    const manager = WebSocketManager.getInstance();
    manager.connect(WS_URL, accessToken);

    // sensor_update → update fleetStore liveGPS in real-time
    const unsubSensor = manager.subscribe("sensor_update", (msg) => {
      const sensorMsg = msg as SensorUpdateMessage;
      
      for (const reading of sensorMsg.data) {
        if (reading.Type === "gps" && isGPSValue(reading.Value)) {
          useFleetStore.getState().setLiveGPS(reading.VehicleID, {
            lat: reading.Value.lat,
            lng: reading.Value.lng,
            timestamp: reading.Timestamp,
          });
        }
      }
    });

    // Subscribe to low_fuel → show toast (admin only)
    const unsubLowFuel = manager.subscribe("low_fuel", (msg) => {
      const fuelMsg = msg as LowFuelMessage;
      const role = useAuthStore.getState().role;
      if (role === "admin") {
        useToastStore.getState().addToast({
          type: "low_fuel",
          message: `Low fuel alert for vehicle ${fuelMsg.vehicle_id}`,
          vehicleId: fuelMsg.vehicle_id,
        });
      }
    });

    return () => {
      unsubSensor();
      unsubLowFuel();
      manager.disconnect();
    };
  }, []);

  return { connected, lastMessage };
}
