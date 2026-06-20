// Fleet store — persisted to localStorage. History capped at 50 points per vehicle.
// Defensive: guards against concurrent fetches; shows cached data while refreshing.

import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { Vehicle, SensorData } from "@/types/domain";
import { getVehicles, getVehicleHistory, getAlerts } from "@/api/endpoints";
import type { Alert } from "@/types/domain";

const MAX_HISTORY_PER_VEHICLE = 50;
const STALE_THRESHOLD_MS = 5 * 60 * 1000; // 5 minutes

// Migration: clear corrupt localStorage data from old formats
function migrateFleetCache() {
  try {
    const raw = localStorage.getItem("fleet-cache");
    if (!raw) return;
    const parsed = JSON.parse(raw);
    if (!parsed?.state?.vehicles || !Array.isArray(parsed.state.vehicles)) {
      console.warn("[fleetStore] Clearing corrupt localStorage cache");
      localStorage.removeItem("fleet-cache");
    }
  } catch {
    localStorage.removeItem("fleet-cache");
  }
}
migrateFleetCache();

export interface GPSPosition {
  lat: number;
  lng: number;
  timestamp: string;
}

export interface FleetState {
  vehicles: Vehicle[];
  history: Record<string, SensorData[]>;
  selectedVehicleId: string | null;
  isLoading: boolean;
  error: string | null;
  lastFetched: number | null;
  isFetching: boolean;

  liveGPS: Record<string, GPSPosition>;

  alerts: Alert[];
  alertLoading: boolean;
  alertError: string | null;

  clearError: () => void;
  fetchAlerts: (force?: boolean) => Promise<void>;
  clearAlertError: () => void;
  setVehicles: (vehicles: Vehicle[]) => void;
  fetchVehicles: (force?: boolean) => Promise<void>;
  fetchHistory: (vehicleId: string) => Promise<void>;
  cacheHistory: (vehicleId: string, points: SensorData[]) => void;
  selectVehicle: (id: string | null) => void;
  getVehicleById: (id: string) => Vehicle | undefined;
  updateVehicleFromSensor: (data: { vehicleId: string; fuelLevel?: number; temperature?: number; latitude?: number; longitude?: number }) => void;
  setLiveGPS: (vehicleId: string, position: GPSPosition) => void;
}

export const useFleetStore = create<FleetState>()(
  persist(
    (set, get) => ({
      vehicles: [],
      history: {},
      selectedVehicleId: null,
      isLoading: false,
      error: null,
      lastFetched: null,
      isFetching: false,
      liveGPS: {},
      alerts: [],
      alertLoading: false,
      alertError: null,

      setVehicles: (vehicles: Vehicle[]) => {
        set({ vehicles });
      },

      fetchVehicles: async (force = false) => {
        // Guard: prevent concurrent fetches
        if (get().isFetching) return;

        // Guard: don't refetch if data is fresh (unless forced)
        const lastFetched = get().lastFetched;
        const hasVehicles = get().vehicles.length > 0;
        if (!force && hasVehicles && lastFetched && Date.now() - lastFetched < STALE_THRESHOLD_MS) {
          return;
        }

        set({ isLoading: !hasVehicles, isFetching: true, error: null });
        try {
          const response = await getVehicles();
          set({
            vehicles: response.vehicles,
            isLoading: false,
            isFetching: false,
            lastFetched: Date.now(),
          });
        } catch (err) {
          const message = err instanceof Error ? err.message : "Failed to fetch vehicles";
          set({ error: message, isLoading: false, isFetching: false });
        }
      },

      fetchHistory: async (vehicleId: string) => {
        // Guard: already have history for this vehicle
        if (get().history[vehicleId]) return;

        try {
          const response = await getVehicleHistory(vehicleId);
          set((state) => ({
            history: {
              ...state.history,
              [vehicleId]: response.history,
            },
          }));
        } catch (err) {
          const message = err instanceof Error ? err.message : "Failed to fetch history";
          set({ error: message });
        }
      },

      cacheHistory: (vehicleId: string, points: SensorData[]) => {
        set((state) => ({
          history: {
            ...state.history,
            [vehicleId]: points,
          },
        }));
      },

      selectVehicle: (id: string | null) => {
        set({ selectedVehicleId: id });
      },

      getVehicleById: (id: string) => {
        return get().vehicles.find((v) => v.id === id);
      },

      updateVehicleFromSensor: (data) => {
        const vehicles = get().vehicles.map((v) => {
          if (v.id !== data.vehicleId) return v;
          return {
            ...v,
            ...(data.fuelLevel !== undefined ? { fuel_level: data.fuelLevel } : {}),
            ...(data.temperature !== undefined ? { temperature: data.temperature } : {}),
            ...(data.latitude !== undefined ? { latitude: data.latitude } : {}),
            ...(data.longitude !== undefined ? { longitude: data.longitude } : {}),
          };
        });
        set({ vehicles });
      },

      setLiveGPS: (vehicleId: string, position: GPSPosition) => {
        set((state) => ({
          liveGPS: {
            ...state.liveGPS,
            [vehicleId]: position,
          },
        }));
      },

      clearError: () => set({ error: null }),

      fetchAlerts: async (force = false) => {
        // Guard: don't refetch if already loaded (unless forced)
        if (!force && get().alerts.length > 0) return;

        set({ alertLoading: get().alerts.length === 0, alertError: null });
        try {
          const response = await getAlerts();
          set({ alerts: response.alerts, alertLoading: false });
        } catch (err) {
          const message = err instanceof Error ? err.message : "Failed to fetch alerts";
          set({ alertError: message, alertLoading: false });
        }
      },

      clearAlertError: () => set({ alertError: null }),
    }),
    {
      name: "fleet-cache",
      partialize: (state) => ({
        vehicles: state.vehicles,
        history: Object.fromEntries(
          Object.entries(state.history)
            .filter(([, points]) => Array.isArray(points))
            .map(([id, points]) => [
              id,
              points.slice(-MAX_HISTORY_PER_VEHICLE),
            ])
        ),
        lastFetched: state.lastFetched,
      }),
      onRehydrateStorage: () => (state) => {
        // Guard: if rehydrated state is corrupt, reset it
        if (state && !Array.isArray(state.vehicles)) {
          state.vehicles = [];
        }
        if (state && typeof state.history !== "object") {
          state.history = {};
        }
      },
    }
  )
);
