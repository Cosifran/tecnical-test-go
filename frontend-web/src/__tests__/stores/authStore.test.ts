/**
 * Tests for authStore — Zustand slice with persist.
 * 
 * Verifies login/logout state transitions, token persistence,
 * role derivation, and token refresh behavior.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { useAuthStore } from "@/stores/authStore";

// Mock the api endpoints module
vi.mock("@/api/endpoints", () => ({
  login: vi.fn(),
  refreshToken: vi.fn(),
}));

import { login as loginEndpoint, refreshToken as refreshEndpoint } from "@/api/endpoints";

describe("authStore", () => {
  beforeEach(() => {
    // Reset store state before each test
    useAuthStore.setState({
      accessToken: null,
      refreshToken: null,
      role: null,
      email: null,
      userId: null,
      isAuthenticated: false,
    });
    // Clear localStorage
    localStorage.clear();
    vi.clearAllMocks();
  });

  describe("initial state", () => {
    it("starts with null tokens and unauthenticated", () => {
      const state = useAuthStore.getState();
      expect(state.accessToken).toBeNull();
      expect(state.refreshToken).toBeNull();
      expect(state.role).toBeNull();
      expect(state.email).toBeNull();
      expect(state.userId).toBeNull();
      expect(state.isAuthenticated).toBe(false);
    });
  });

  describe("login", () => {
    it("stores tokens and extracts role/email from JWT on successful login", async () => {
      // Create a valid JWT-like payload
      const payload = {
        sub: "user-123",
        email: "admin@fleet.com",
        role: "admin",
        exp: 1893456000,
        iat: 1700000000,
      };
      const payloadB64 = btoa(JSON.stringify(payload))
        .replace(/\+/g, "-")
        .replace(/\//g, "_")
        .replace(/=+$/, "");
      const fakeAccessToken = `eyJhbGciOiJIUzI1NiJ9.${payloadB64}.fake-sig`;

      (loginEndpoint as ReturnType<typeof vi.fn>).mockResolvedValue({
        access_token: fakeAccessToken,
        refresh_token: "refresh-abc",
        token_type: "Bearer",
        expires_in: 3600,
      });

      await useAuthStore.getState().login("admin@fleet.com", "password123");

      const state = useAuthStore.getState();
      expect(state.accessToken).toBe(fakeAccessToken);
      expect(state.refreshToken).toBe("refresh-abc");
      expect(state.role).toBe("admin");
      expect(state.email).toBe("admin@fleet.com");
      expect(state.userId).toBe("user-123");
      expect(state.isAuthenticated).toBe(true);
    });

    it("clears state on login failure", async () => {
      (loginEndpoint as ReturnType<typeof vi.fn>).mockRejectedValue(
        new Error("Invalid credentials")
      );

      await expect(
        useAuthStore.getState().login("bad@fleet.com", "wrong")
      ).rejects.toThrow("Invalid credentials");

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.accessToken).toBeNull();
    });
  });

  describe("logout", () => {
    it("clears all auth state", async () => {
      // First set up an authenticated state
      const payload = {
        sub: "user-456",
        email: "user@fleet.com",
        role: "user",
        exp: 1893456000,
      };
      const payloadB64 = btoa(JSON.stringify(payload))
        .replace(/\+/g, "-")
        .replace(/\//g, "_")
        .replace(/=+$/, "");
      const fakeToken = `eyJhbGciOiJIUzI1NiJ9.${payloadB64}.fake-sig`;

      (loginEndpoint as ReturnType<typeof vi.fn>).mockResolvedValue({
        access_token: fakeToken,
        refresh_token: "refresh-xyz",
        token_type: "Bearer",
        expires_in: 3600,
      });

      await useAuthStore.getState().login("user@fleet.com", "pass");
      expect(useAuthStore.getState().isAuthenticated).toBe(true);

      // Now logout
      useAuthStore.getState().logout();

      const state = useAuthStore.getState();
      expect(state.accessToken).toBeNull();
      expect(state.refreshToken).toBeNull();
      expect(state.role).toBeNull();
      expect(state.email).toBeNull();
      expect(state.userId).toBeNull();
      expect(state.isAuthenticated).toBe(false);
    });
  });

  describe("refresh", () => {
    it("updates tokens on successful refresh", async () => {
      // Set up initial auth state
      const initialPayload = {
        sub: "user-1",
        email: "admin@fleet.com",
        role: "admin",
        exp: 1, // expired
      };
      const initialB64 = btoa(JSON.stringify(initialPayload))
        .replace(/\+/g, "-")
        .replace(/\//g, "_")
        .replace(/=+$/, "");

      useAuthStore.setState({
        accessToken: `header.${initialB64}.sig`,
        refreshToken: "old-refresh",
        isAuthenticated: true,
        role: "admin",
        email: "admin@fleet.com",
        userId: "user-1",
      });

      // New token with fresh exp
      const newPayload = {
        sub: "user-1",
        email: "admin@fleet.com",
        role: "admin",
        exp: 1893456000,
      };
      const newB64 = btoa(JSON.stringify(newPayload))
        .replace(/\+/g, "-")
        .replace(/\//g, "_")
        .replace(/=+$/, "");
      const newAccessToken = `header.${newB64}.new-sig`;

      (refreshEndpoint as ReturnType<typeof vi.fn>).mockResolvedValue({
        access_token: newAccessToken,
        refresh_token: "new-refresh",
        token_type: "Bearer",
        expires_in: 3600,
      });

      const result = await useAuthStore.getState().refresh();

      expect(result).toBe(true);
      expect(useAuthStore.getState().accessToken).toBe(newAccessToken);
      expect(useAuthStore.getState().refreshToken).toBe("new-refresh");
    });

    it("returns false on refresh failure", async () => {
      useAuthStore.setState({
        refreshToken: "bad-refresh-token",
        isAuthenticated: true,
      });

      (refreshEndpoint as ReturnType<typeof vi.fn>).mockRejectedValue(
        new Error("Refresh token expired")
      );

      const result = await useAuthStore.getState().refresh();
      expect(result).toBe(false);
    });
  });

  describe("setTokens", () => {
    it("updates tokens and derivess role from access token", () => {
      const payload = {
        sub: "user-789",
        email: "driver@fleet.com",
        role: "user",
        exp: 1893456000,
      };
      const payloadB64 = btoa(JSON.stringify(payload))
        .replace(/\+/g, "-")
        .replace(/\//g, "_")
        .replace(/=+$/, "");
      const fakeToken = `eyJhbGciOiJIUzI1NiJ9.${payloadB64}.fake-sig`;

      useAuthStore.getState().setTokens(fakeToken, "new-refresh");

      const state = useAuthStore.getState();
      expect(state.accessToken).toBe(fakeToken);
      expect(state.refreshToken).toBe("new-refresh");
      expect(state.role).toBe("user");
      expect(state.email).toBe("driver@fleet.com");
      expect(state.userId).toBe("user-789");
      expect(state.isAuthenticated).toBe(true);
    });
  });
});