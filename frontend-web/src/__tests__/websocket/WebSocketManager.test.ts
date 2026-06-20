/**
 * Tests for WebSocketManager — singleton class.
 *
 * Mocks WebSocket to verify:
 * - Connect with token in URL
 * - Message dispatch by type
 * - Exponential backoff reconnect (1s → 2s → 4s → ... → max 30s)
 * - Reconnect attempt counter resets on successful connection
 * - No reconnect on graceful disconnect
 * - Subscribe/unsubscribe pattern
 * - Disconnect on logout
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { WebSocketManager } from "@/websocket/WebSocketManager";
import { useWStore } from "@/stores/wsStore";

// --- MockWebSocket implementation ---
class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static LAST(): MockWebSocket {
    return MockWebSocket.instances[MockWebSocket.instances.length - 1]!;
  }

  url: string;
  readyState: number = WebSocket.CONNECTING;
  onopen: ((ev: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  send = vi.fn();
  close = vi.fn((code?: number) => {
    this.readyState = WebSocket.CLOSED;
  });

  // Test helpers: simulate server events
  simulateOpen() {
    this.readyState = WebSocket.OPEN;
    this.onopen?.(new Event("open"));
  }

  simulateMessage(data: unknown) {
    this.onmessage?.(new MessageEvent("message", { data: JSON.stringify(data) }));
  }

  simulateClose(code = 1006, reason = "") {
    this.readyState = WebSocket.CLOSED;
    this.onclose?.(new CloseEvent("close", { code, reason }));
  }

  simulateError() {
    this.onerror?.(new Event("error"));
  }
}

// Mock global WebSocket
const originalWebSocket = globalThis.WebSocket;

beforeEach(() => {
  MockWebSocket.instances = [];
  vi.useFakeTimers();
  useWStore.getState().reset();
  globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket;
});

afterEach(() => {
  vi.useRealTimers();
  globalThis.WebSocket = originalWebSocket;
  // Disconnect the manager to clear any pending timers
  WebSocketManager.getInstance().disconnect();
});

describe("WebSocketManager", () => {
  describe("connect", () => {
    it("connects with access token in the URL", () => {
      const manager = WebSocketManager.getInstance();
      manager.connect("ws://localhost:8080/api/v1/ws", "test-token-123");

      expect(MockWebSocket.LAST().url).toBe(
        "ws://localhost:8080/api/v1/ws?token=test-token-123"
      );
    });

    it("sets wsStore connected to true on open", () => {
      const manager = WebSocketManager.getInstance();
      manager.connect("ws://localhost:8080/api/v1/ws", "my-token");

      MockWebSocket.LAST().simulateOpen();

      expect(useWStore.getState().connected).toBe(true);
    });

    it("resets reconnect attempt to 0 on successful connection", () => {
      const manager = WebSocketManager.getInstance();
      useWStore.getState().setReconnectAttempt(5);

      manager.connect("ws://localhost:8080/api/v1/ws", "token");
      MockWebSocket.LAST().simulateOpen();

      expect(useWStore.getState().reconnectAttempt).toBe(0);
    });

    it("stores last message in wsStore", () => {
      const manager = WebSocketManager.getInstance();
      manager.connect("ws://localhost:8080/api/v1/ws", "token");
      MockWebSocket.LAST().simulateOpen();

      const sensorMsg = {
        type: "sensor_update",
        vehicle_id: "v-001",
        fuel_level: 75,
        temperature: 22,
        latitude: 40.7,
        longitude: -74.0,
      };
      MockWebSocket.LAST().simulateMessage(sensorMsg);

      expect(useWStore.getState().lastMessage).toEqual(sensorMsg);
    });
  });

  describe("message dispatch", () => {
    it("dispatches sensor_update messages to subscribers", () => {
      const manager = WebSocketManager.getInstance();
      const callback = vi.fn();
      manager.subscribe("sensor_update", callback);

      manager.connect("ws://localhost:8080/api/v1/ws", "token");
      MockWebSocket.LAST().simulateOpen();

      const msg = {
        type: "sensor_update",
        vehicle_id: "v-001",
        fuel_level: 80,
        temperature: 20,
        latitude: 40.0,
        longitude: -74.0,
      };
      MockWebSocket.LAST().simulateMessage(msg);

      expect(callback).toHaveBeenCalledWith(msg);
    });

    it("dispatches low_fuel messages to subscribers", () => {
      const manager = WebSocketManager.getInstance();
      const callback = vi.fn();
      manager.subscribe("low_fuel", callback);

      manager.connect("ws://localhost:8080/api/v1/ws", "token");
      MockWebSocket.LAST().simulateOpen();

      const msg = {
        type: "low_fuel",
        vehicle_id: "v-002",
        estimated_minutes: 45,
      };
      MockWebSocket.LAST().simulateMessage(msg);

      expect(callback).toHaveBeenCalledWith(msg);
    });

    it("does not dispatch to subscribers of other types", () => {
      const manager = WebSocketManager.getInstance();
      const sensorCallback = vi.fn();
      const fuelCallback = vi.fn();

      manager.subscribe("sensor_update", sensorCallback);
      manager.subscribe("low_fuel", fuelCallback);

      manager.connect("ws://localhost:8080/api/v1/ws", "token");
      MockWebSocket.LAST().simulateOpen();

      MockWebSocket.LAST().simulateMessage({
        type: "low_fuel",
        vehicle_id: "v-002",
        estimated_minutes: 30,
      });

      expect(sensorCallback).not.toHaveBeenCalled();
      expect(fuelCallback).toHaveBeenCalledTimes(1);
    });
  });

  describe("reconnection with exponential backoff", () => {
    it("reconnects after unexpected close with 1s initial delay", () => {
      const manager = WebSocketManager.getInstance();
      manager.connect("ws://localhost:8080/api/v1/ws", "token");
      MockWebSocket.LAST().simulateOpen();

      // Simulate unexpected close
      MockWebSocket.LAST().simulateClose(1006);

      expect(useWStore.getState().connected).toBe(false);

      // Advance by 1 second — should trigger reconnect
      vi.advanceTimersByTime(1000);

      // A new WebSocket should have been created
      expect(MockWebSocket.instances.length).toBe(2);
      expect(useWStore.getState().reconnectAttempt).toBe(1);
    });

    it("increases backoff exponentially: 1s → 2s → 4s → 8s → 16s", () => {
      const manager = WebSocketManager.getInstance();
      manager.connect("ws://localhost:8080/api/v1/ws", "token");
      MockWebSocket.LAST().simulateOpen();

      // First disconnect: backoff = 1s
      MockWebSocket.LAST().simulateClose(1006);
      vi.advanceTimersByTime(1000);
      // Second disconnect: backoff = 2s
      MockWebSocket.LAST().simulateClose(1006);
      vi.advanceTimersByTime(2000);
      // Third disconnect: backoff = 4s
      MockWebSocket.LAST().simulateClose(1006);
      vi.advanceTimersByTime(4000);
      // Fourth disconnect: backoff = 8s
      MockWebSocket.LAST().simulateClose(1006);
      vi.advanceTimersByTime(8000);
      // Fifth disconnect: backoff = 16s
      MockWebSocket.LAST().simulateClose(1006);
      vi.advanceTimersByTime(16000);

      // 5 reconnects total + the initial connection = 6 instances
      expect(MockWebSocket.instances.length).toBe(6);
      expect(useWStore.getState().reconnectAttempt).toBe(5);
    });

    it("caps backoff at 30 seconds", () => {
      const manager = WebSocketManager.getInstance();
      manager.connect("ws://localhost:8080/api/v1/ws", "token");
      MockWebSocket.LAST().simulateOpen();

      // Simulate 10 disconnects to reach cap
      for (let i = 0; i < 10; i++) {
        MockWebSocket.LAST().simulateClose(1006);
        // The max delay should be 30000ms, so advancing by 30s should be enough
        vi.advanceTimersByTime(30000);
      }

      // All connections should have been created
      expect(MockWebSocket.instances.length).toBe(11);
      expect(useWStore.getState().reconnectAttempt).toBe(10);
    });

    it("resets reconnect attempt on successful reconnection", () => {
      const manager = WebSocketManager.getInstance();
      manager.connect("ws://localhost:8080/api/v1/ws", "token");
      MockWebSocket.LAST().simulateOpen();

      // Disconnect and reconnect once
      MockWebSocket.LAST().simulateClose(1006);
      vi.advanceTimersByTime(1000);
      expect(useWStore.getState().reconnectAttempt).toBe(1);

      // New connection succeeds — should reset attempt counter
      MockWebSocket.LAST().simulateOpen();
      expect(useWStore.getState().reconnectAttempt).toBe(0);
    });
  });

  describe("graceful disconnect", () => {
    it("does not reconnect on graceful disconnect (code 1000)", () => {
      const manager = WebSocketManager.getInstance();
      manager.connect("ws://localhost:8080/api/v1/ws", "token");
      MockWebSocket.LAST().simulateOpen();

      // Graceful disconnect
      MockWebSocket.LAST().simulateClose(1000);

      // Advance time significantly — no reconnect should happen
      vi.advanceTimersByTime(60000);
      expect(MockWebSocket.instances.length).toBe(1);
    });

    it("does not reconnect after explicit disconnect()", () => {
      const manager = WebSocketManager.getInstance();
      manager.connect("ws://localhost:8080/api/v1/ws", "token");
      MockWebSocket.LAST().simulateOpen();

      manager.disconnect();

      // Advance time significantly — no reconnect should happen
      vi.advanceTimersByTime(60000);
      expect(MockWebSocket.instances.length).toBe(1);
    });
  });

  describe("subscribe / unsubscribe", () => {
    it("stops receiving messages after unsubscribe", () => {
      const manager = WebSocketManager.getInstance();
      const callback = vi.fn();
      const unsub = manager.subscribe("sensor_update", callback);

      manager.connect("ws://localhost:8080/api/v1/ws", "token");
      MockWebSocket.LAST().simulateOpen();

      MockWebSocket.LAST().simulateMessage({
        type: "sensor_update",
        vehicle_id: "v-001",
        fuel_level: 50,
        temperature: 25,
        latitude: 40.0,
        longitude: -74.0,
      });
      expect(callback).toHaveBeenCalledTimes(1);

      // Unsubscribe
      unsub();

      MockWebSocket.LAST().simulateMessage({
        type: "sensor_update",
        vehicle_id: "v-002",
        fuel_level: 60,
        temperature: 30,
        latitude: 41.0,
        longitude: -73.0,
      });
      expect(callback).toHaveBeenCalledTimes(1); // Still 1, not 2
    });
  });

  describe("singleton", () => {
    it("returns the same instance", () => {
      const a = WebSocketManager.getInstance();
      const b = WebSocketManager.getInstance();
      expect(a).toBe(b);
    });
  });
});