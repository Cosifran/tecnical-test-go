/**
 * Tests for useWebSocket hook.
 *
 * Verifies:
 * - Connects with token from authStore when authenticated
 * - Does not connect when not authenticated
 * - Disconnects on unmount
 * - Returns connection status and last message from wsStore
 * - Subscribes to sensor_update and low_fuel message types
 * - Updates liveGPS in fleetStore when sensor_update arrives
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { useWebSocket } from "@/websocket/useWebSocket";
import { useWStore } from "@/stores/wsStore";
import { useAuthStore } from "@/stores/authStore";
import { useFleetStore } from "@/stores/fleetStore";
import { useToastStore } from "@/stores/toastStore";

// Track calls to WebSocketManager methods
const mockConnectFn = vi.fn();
const mockDisconnectFn = vi.fn();
const mockSubscribeFn = vi.fn(() => vi.fn());

// Mock the WebSocketManager module
vi.mock("@/websocket/WebSocketManager", () => ({
  WebSocketManager: {
    getInstance: () => ({
      connect: mockConnectFn,
      disconnect: mockDisconnectFn,
      subscribe: mockSubscribeFn,
    }),
  },
}));

describe("useWebSocket", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWStore.getState().reset();
    useToastStore.getState().clearToasts();

    // Set up auth store with a token (admin role)
    const payload = {
      sub: "user-1",
      email: "admin@fleet.com",
      role: "admin",
      exp: 1893456000,
    };
    const payloadB64 = btoa(JSON.stringify(payload))
      .replace(/\+/g, "-")
      .replace(/\//g, "_")
      .replace(/=+$/, "");
    const fakeToken = `header.${payloadB64}.sig`;

    useAuthStore.setState({
      accessToken: fakeToken,
      refreshToken: "refresh-123",
      role: "admin",
      email: "admin@fleet.com",
      userId: "user-1",
      isAuthenticated: true,
    });
  });

  describe("connection", () => {
    it("connects to WebSocket on mount when authenticated", () => {
      renderHook(() => useWebSocket());

      expect(mockConnectFn).toHaveBeenCalledWith(
        expect.any(String),
        expect.any(String)
      );
      // Token should be the access token from auth store
      expect(mockConnectFn.mock.calls[0]![1]).toBeTruthy();
    });

    it("does not connect when not authenticated", () => {
      useAuthStore.setState({
        accessToken: null,
        isAuthenticated: false,
      });

      renderHook(() => useWebSocket());

      expect(mockConnectFn).not.toHaveBeenCalled();
    });

    it("disconnects on unmount", () => {
      const { unmount } = renderHook(() => useWebSocket());

      unmount();

      expect(mockDisconnectFn).toHaveBeenCalled();
    });
  });

  describe("state bridge", () => {
    it("returns connected status from wsStore", () => {
      useWStore.getState().setConnected(true);

      const { result } = renderHook(() => useWebSocket());

      expect(result.current.connected).toBe(true);
    });

    it("returns last message from wsStore", () => {
      const msg = {
        type: "sensor_update" as const,
        data: [
          {
            ID: "s1",
            VehicleID: "v-001",
            Type: "gps",
            Value: { lat: 4.6, lng: -74.0 },
            Timestamp: "2026-01-01T00:00:00Z",
            CreatedAt: "2026-01-01T00:00:00Z",
          },
        ],
      };
      useWStore.getState().setLastMessage(msg);

      const { result } = renderHook(() => useWebSocket());

      expect(result.current.lastMessage).toEqual(msg);
    });
  });

  describe("subscription forwarding", () => {
    it("subscribes to sensor_update messages", () => {
      renderHook(() => useWebSocket());

      const subscribedTypes = mockSubscribeFn.mock.calls.map(
        (call: unknown[]) => call[0] as string
      );
      expect(subscribedTypes).toContain("sensor_update");
    });

    it("subscribes to low_fuel messages", () => {
      renderHook(() => useWebSocket());

      const subscribedTypes = mockSubscribeFn.mock.calls.map(
        (call: unknown[]) => call[0] as string
      );
      expect(subscribedTypes).toContain("low_fuel");
    });

    it("sensor_update handler updates liveGPS in fleetStore", () => {
      renderHook(() => useWebSocket());

      // Find the sensor_update subscription handler
      const sensorCall = mockSubscribeFn.mock.calls.find(
        (call: unknown[]) => call[0] === "sensor_update"
      );
      expect(sensorCall).toBeDefined();
      // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
      const handler = (sensorCall as unknown as [string, (msg: unknown) => void])[1];

      // Simulate a sensor_update message with GPS data
      handler({
        type: "sensor_update",
        data: [
          {
            ID: "s1",
            VehicleID: "v-001",
            Type: "gps",
            Value: { lat: 4.65, lng: -74.05 },
            Timestamp: "2026-01-01T00:00:00Z",
            CreatedAt: "2026-01-01T00:00:00Z",
          },
        ],
      });

      const livePos = useFleetStore.getState().liveGPS["v-001"];
      expect(livePos).toBeDefined();
      expect(livePos!.lat).toBe(4.65);
      expect(livePos!.lng).toBe(-74.05);
    });

    it("low_fuel handler adds toast for admin users", () => {
      // Auth store is set to admin in beforeEach
      renderHook(() => useWebSocket());

      // Find the low_fuel subscription handler
      const fuelCall = mockSubscribeFn.mock.calls.find(
        (call: unknown[]) => call[0] === "low_fuel"
      );
      expect(fuelCall).toBeDefined();
      const handler = (fuelCall as unknown as [string, (msg: unknown) => void])[1];

      // Simulate a low_fuel message
      handler({
        type: "low_fuel",
        vehicle_id: "v-002",
      });

      const toasts = useToastStore.getState().toasts;
      expect(toasts).toHaveLength(1);
      expect(toasts[0]!.type).toBe("low_fuel");
      expect(toasts[0]!.vehicleId).toBe("v-002");
    });

    it("low_fuel handler does NOT add toast for non-admin users", () => {
      // Set role to user (not admin)
      useAuthStore.setState({ role: "user" });

      renderHook(() => useWebSocket());

      const fuelCall = mockSubscribeFn.mock.calls.find(
        (call: unknown[]) => call[0] === "low_fuel"
      );
      expect(fuelCall).toBeDefined();
      const handler2 = (fuelCall as unknown as [string, (msg: unknown) => void])[1];

      handler2({
        type: "low_fuel",
        vehicle_id: "v-002",
      });

      expect(useToastStore.getState().toasts).toHaveLength(0);
    });
  });
});
