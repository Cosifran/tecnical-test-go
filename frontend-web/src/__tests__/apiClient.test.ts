/**
 * Tests for API client — fetch wrapper with Bearer token injection,
 * 401 retry with token refresh, and redirect on failure.
 *
 * Tests mock global fetch to verify behavior without network calls.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { apiClient, API_BASE_URL, setAuthStoreForTests } from "@/api/apiClient";

// Mock auth store reference for tests
const mockAuthStore = {
  accessToken: "test-access-token",
  refreshToken: "test-refresh-token",
  isAuthenticated: true,
  role: "admin" as const,
  email: "admin@fleet.com",
  userId: "user-1",
  login: vi.fn(),
  logout: vi.fn(),
  refresh: vi.fn(),
  setTokens: vi.fn(),
};

describe("apiClient", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    globalThis.fetch = vi.fn();
    setAuthStoreForTests(mockAuthStore);
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it("injects Authorization Bearer header from authStore", async () => {
    const mockResponse = new Response(JSON.stringify({ data: "ok" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse);

    const result = await apiClient<{ data: string }>("/api/v1/vehicles");

    expect(globalThis.fetch).toHaveBeenCalledWith(
      `${API_BASE_URL}/api/v1/vehicles`,
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer test-access-token",
        }),
      })
    );
    expect(result.data).toBe("ok");
  });

  it("makes request without Authorization when not authenticated", async () => {
    setAuthStoreForTests({
      ...mockAuthStore,
      accessToken: null,
      isAuthenticated: false,
    });

    const mockResponse = new Response(JSON.stringify({ result: "public" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse);

const result = await apiClient<{ result: string }>("/api/v1/public");

    const callArgs = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0]!;
    const headers = callArgs[1]!.headers as Record<string, string>;
    expect(headers.Authorization).toBeUndefined();
    expect(result.result).toBe("public");
  });

  it("returns typed response for successful requests", async () => {
    const mockData = { vehicles: [{ id: "v1", name: "Truck 1" }] };
    const mockResponse = new Response(JSON.stringify(mockData), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse);

    const result = await apiClient<{ vehicles: Array<{ id: string; name: string }> }>("/api/v1/vehicles");

    expect(result.vehicles).toHaveLength(1);
    expect(result.vehicles[0]!.id).toBe("v1");
  });

  it("throws with error message on non-401 error response", async () => {
    const errorBody = JSON.stringify({ error: "not_found", message: "Vehicle not found" });
    const mockResponse = new Response(errorBody, {
      status: 404,
      headers: { "Content-Type": "application/json" },
    });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse);

    await expect(apiClient("/api/v1/vehicles/nonexistent")).rejects.toThrow("Vehicle not found");
  });

  it("handles 401 by refreshing token and retrying once", async () => {
    // First call: 401
    const unauthorizedResponse = new Response(JSON.stringify({ error: "unauthorized" }), {
      status: 401,
    });
    // After refresh: success
    const successResponse = new Response(JSON.stringify({ data: "retry-success" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });

    (globalThis.fetch as ReturnType<typeof vi.fn>)
      .mockResolvedValueOnce(unauthorizedResponse)
      .mockResolvedValueOnce(successResponse);

    (mockAuthStore.refresh as ReturnType<typeof vi.fn>).mockResolvedValue(true);

    const result = await apiClient<{ data: string }>("/api/v1/vehicles");

    expect(mockAuthStore.refresh).toHaveBeenCalledOnce();
    expect(result.data).toBe("retry-success");
  });

  it("calls logout and redirects when refresh fails", async () => {
    const unauthorizedResponse = new Response(JSON.stringify({ error: "unauthorized" }), {
      status: 401,
    });

    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(unauthorizedResponse);
    (mockAuthStore.refresh as ReturnType<typeof vi.fn>).mockResolvedValue(false);

    // Mock window.location.href setter
    const originalLocation = window.location;
    const mockLocation = { ...originalLocation, href: "" };
    Object.defineProperty(window, "location", { value: mockLocation, writable: true });

    await expect(apiClient("/api/v1/vehicles")).rejects.toThrow("Session expired");

    expect(mockAuthStore.logout).toHaveBeenCalledOnce();
    expect(mockLocation.href).toBe("/login");

    // Restore
    Object.defineProperty(window, "location", { value: originalLocation, writable: true });
  });

  it("handles non-JSON error responses", async () => {
    const mockResponse = new Response("Internal Server Error", {
      status: 500,
    });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse);

    await expect(apiClient("/api/v1/vehicles")).rejects.toThrow();
  });

  it("allows custom headers in request options", async () => {
    const mockResponse = new Response(JSON.stringify({ data: "ok" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse);

    await apiClient("/api/v1/vehicles", {
      headers: { "X-Custom-Header": "custom-value" },
    });

    const callArgs = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0]!;
    const headers = callArgs[1]!.headers as Record<string, string>;
    expect(headers["X-Custom-Header"]).toBe("custom-value");
    expect(headers.Authorization).toBe("Bearer test-access-token");
  });

  it("uses new access token after refresh for retry", async () => {
    const unauthorizedResponse = new Response(JSON.stringify({ error: "unauthorized" }), {
      status: 401,
    });
    const successResponse = new Response(JSON.stringify({ data: "retried" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });

    (globalThis.fetch as ReturnType<typeof vi.fn>)
      .mockResolvedValueOnce(unauthorizedResponse)
      .mockResolvedValueOnce(successResponse);

    (mockAuthStore.refresh as ReturnType<typeof vi.fn>).mockImplementation(async () => {
      // Simulate token update after refresh
      mockAuthStore.accessToken = "new-access-token";
      return true;
    });

    await apiClient<{ data: string }>("/api/v1/vehicles");

    // The second call (retry) should use the new token
    const secondCallArgs = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[1]!;
    const headers = secondCallArgs[1]!.headers as Record<string, string>;
    expect(headers.Authorization).toBe("Bearer new-access-token");

    // Reset
    mockAuthStore.accessToken = "test-access-token";
  });
});