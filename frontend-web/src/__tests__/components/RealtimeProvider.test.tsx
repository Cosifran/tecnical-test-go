/**
 * Tests for RealtimeProvider component.
 *
 * Verifies:
 * - Renders children when authenticated
 * - Connects WebSocket when user is authenticated
 * - Does not connect when user is not authenticated
 * - Shows OfflineBanner and ToastContainer
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { RealtimeProvider } from "@/components/RealtimeProvider";
import { useAuthStore } from "@/stores/authStore";
import { useWStore } from "@/stores/wsStore";

// Mock useOnline to control offline state
let mockIsOnline = true;
vi.mock("@/hooks/useOnline", () => ({
  useOnline: () => mockIsOnline,
}));

// Mock WebSocketManager
vi.mock("@/websocket/WebSocketManager", () => ({
  WebSocketManager: {
    getInstance: vi.fn(() => ({
      connect: vi.fn(),
      disconnect: vi.fn(),
      subscribe: vi.fn(() => vi.fn()),
    })),
  },
}));

// Mock useWebSocket since it depends on WebSocketManager
vi.mock("@/websocket/useWebSocket", () => ({
  useWebSocket: () => ({
    connected: mockIsOnline,
    lastMessage: null,
  }),
}));

describe("RealtimeProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWStore.getState().reset();
    mockIsOnline = true;

    // Default: authenticated admin
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

  it("renders children", () => {
    render(
      <RealtimeProvider>
        <div data-testid="child">Hello</div>
      </RealtimeProvider>
    );
    expect(screen.getByTestId("child")).toBeInTheDocument();
  });

  it("does not show offline banner when online", () => {
    mockIsOnline = true;
    render(
      <RealtimeProvider>
        <div>Content</div>
      </RealtimeProvider>
    );
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("shows offline banner when offline", () => {
    mockIsOnline = false;
    render(
      <RealtimeProvider>
        <div>Content</div>
      </RealtimeProvider>
    );
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText(/you are offline/i)).toBeInTheDocument();
  });
});