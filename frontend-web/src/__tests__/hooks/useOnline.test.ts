/**
 * Tests for useOnline hook.
 *
 * Verifies:
 * - Returns true when navigator.onLine is true
 * - Returns false when navigator.onLine is false
 * - Updates on online/offline events
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useOnline } from "@/hooks/useOnline";

describe("useOnline", () => {
  const originalOnLine = navigator.onLine;

  beforeEach(() => {
    // Default to online
    vi.stubGlobal("navigator", { onLine: true });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("returns true when navigator.onLine is true", () => {
    const { result } = renderHook(() => useOnline());
    expect(result.current).toBe(true);
  });

  it("returns false when navigator.onLine is false", () => {
    vi.stubGlobal("navigator", { onLine: false });
    const { result } = renderHook(() => useOnline());
    expect(result.current).toBe(false);
  });

  it("updates to false when offline event fires", () => {
    const { result } = renderHook(() => useOnline());
    expect(result.current).toBe(true);

    act(() => {
      vi.stubGlobal("navigator", { onLine: false });
      window.dispatchEvent(new Event("offline"));
    });

    expect(result.current).toBe(false);
  });

  it("updates to true when online event fires after offline", () => {
    vi.stubGlobal("navigator", { onLine: false });
    const { result } = renderHook(() => useOnline());
    expect(result.current).toBe(false);

    act(() => {
      vi.stubGlobal("navigator", { onLine: true });
      window.dispatchEvent(new Event("online"));
    });

    expect(result.current).toBe(true);
  });
});