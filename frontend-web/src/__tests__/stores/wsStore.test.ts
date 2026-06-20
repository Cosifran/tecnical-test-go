/**
 * Tests for wsStore — ephemeral Zustand slice.
 *
 * Verifies initial state, state setters, and reset behavior.
 * wsStore is NOT persisted — it starts fresh on every page load.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { useWStore } from "@/stores/wsStore";
import type { WsMessage } from "@/types/ws";

describe("wsStore", () => {
  beforeEach(() => {
    // Reset to initial state before each test
    useWStore.getState().reset();
  });

  describe("initial state", () => {
    it("starts disconnected with zero attempts and no last message", () => {
      const state = useWStore.getState();
      expect(state.connected).toBe(false);
      expect(state.reconnectAttempt).toBe(0);
      expect(state.lastMessage).toBeNull();
    });
  });

  describe("setConnected", () => {
    it("sets connected to true", () => {
      useWStore.getState().setConnected(true);
      expect(useWStore.getState().connected).toBe(true);
    });

    it("sets connected to false", () => {
      useWStore.getState().setConnected(true);
      expect(useWStore.getState().connected).toBe(true);

      useWStore.getState().setConnected(false);
      expect(useWStore.getState().connected).toBe(false);
    });
  });

  describe("setReconnectAttempt", () => {
    it("sets reconnect attempt to a specific number", () => {
      useWStore.getState().setReconnectAttempt(3);
      expect(useWStore.getState().reconnectAttempt).toBe(3);
    });

    it("resets reconnect attempt to zero", () => {
      useWStore.getState().setReconnectAttempt(5);
      expect(useWStore.getState().reconnectAttempt).toBe(5);

      useWStore.getState().setReconnectAttempt(0);
      expect(useWStore.getState().reconnectAttempt).toBe(0);
    });
  });

  describe("setLastMessage", () => {
    it("stores a sensor_update message", () => {
      const msg: WsMessage = {
        type: "sensor_update",
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
      expect(useWStore.getState().lastMessage).toEqual(msg);
    });

    it("stores a low_fuel message", () => {
      const msg: WsMessage = {
        type: "low_fuel",
        vehicle_id: "v-002",
      };

      useWStore.getState().setLastMessage(msg);
      expect(useWStore.getState().lastMessage).toEqual(msg);
    });

    it("replaces the last message with a new one", () => {
      const msg1: WsMessage = {
        type: "sensor_update",
        data: [
          {
            ID: "s1",
            VehicleID: "v-001",
            Type: "fuel",
            Value: { level: 80, unit: "liters" },
            Timestamp: "2026-01-01T00:00:00Z",
            CreatedAt: "2026-01-01T00:00:00Z",
          },
        ],
      };
      const msg2: WsMessage = {
        type: "low_fuel",
        vehicle_id: "v-003",
      };

      useWStore.getState().setLastMessage(msg1);
      expect(useWStore.getState().lastMessage).toEqual(msg1);

      useWStore.getState().setLastMessage(msg2);
      expect(useWStore.getState().lastMessage).toEqual(msg2);
    });
  });

  describe("reset", () => {
    it("restores all state to initial values", () => {
      // Set some state
      useWStore.getState().setConnected(true);
      useWStore.getState().setReconnectAttempt(4);
      useWStore.getState().setLastMessage({
        type: "low_fuel",
        vehicle_id: "v-999",
      });

      // Verify state was set
      expect(useWStore.getState().connected).toBe(true);
      expect(useWStore.getState().reconnectAttempt).toBe(4);
      expect(useWStore.getState().lastMessage).not.toBeNull();

      // Reset
      useWStore.getState().reset();

      // Verify reset
      expect(useWStore.getState().connected).toBe(false);
      expect(useWStore.getState().reconnectAttempt).toBe(0);
      expect(useWStore.getState().lastMessage).toBeNull();
    });
  });
});
