/**
 * Tests for client-side device ID masking.
 *
 * This mirrors the backend MaskDeviceID function EXACTLY.
 * The format must be consistent: DEV-****-{last4}
 *
 * These tests verify the same cases as the backend masking tests
 * to ensure frontend and backend masking produce identical results.
 */

import { describe, it, expect } from "vitest";
import { maskDeviceId } from "../utils/masking";

describe("maskDeviceId", () => {
  it("masks a standard device ID with hyphens", () => {
    expect(maskDeviceId("DEV-12345678-ABCD")).toBe("DEV-****-ABCD");
  });

  it("masks a UUID format device ID", () => {
    expect(maskDeviceId("550e8400-e29b-41d4-a716-446655440000")).toBe("DEV-****-0000");
  });

  it("masks a short meaningful ID", () => {
    expect(maskDeviceId("SENSOR-XYZ-1234")).toBe("DEV-****-1234");
  });

  it("handles exactly 4 characters", () => {
    expect(maskDeviceId("ABCD")).toBe("DEV-****-ABCD");
  });

  it("returns fallback for 3 characters (too short)", () => {
    expect(maskDeviceId("ABC")).toBe("DEV-****-????");
  });

  it("returns fallback for 2 characters", () => {
    expect(maskDeviceId("AB")).toBe("DEV-****-????");
  });

  it("returns fallback for 1 character", () => {
    expect(maskDeviceId("X")).toBe("DEV-****-????");
  });

  it("returns fallback for empty string", () => {
    expect(maskDeviceId("")).toBe("DEV-****-????");
  });

  it("handles 5 characters (just enough)", () => {
    expect(maskDeviceId("ABCDE")).toBe("DEV-****-BCDE");
  });

  it("handles numeric last 4", () => {
    expect(maskDeviceId("DEV-12345678-1234")).toBe("DEV-****-1234");
  });

  it("handles mixed case last 4", () => {
    expect(maskDeviceId("DEV-12345678-XyZ9")).toBe("DEV-****-XyZ9");
  });

  it("is deterministic — same input always produces same output", () => {
    const input = "DEV-12345678-ABCD";
    const results = new Array(100).fill(null).map(() => maskDeviceId(input));
    const unique = new Set(results);
    expect(unique.size).toBe(1);
    expect(results[0]).toBe("DEV-****-ABCD");
  });

  it("different IDs with different last 4 produce different masks", () => {
    const mask1 = maskDeviceId("DEV-11111111-ABCD");
    const mask2 = maskDeviceId("DEV-22222222-EFGH");
    expect(mask1).not.toBe(mask2);
  });
});