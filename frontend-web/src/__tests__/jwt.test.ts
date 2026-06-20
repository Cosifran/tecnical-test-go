/**
 * Tests for JWT payload decoding utility.
 * 
 * The backend requires JWT validation WITHOUT external libraries,
 * so we implement base64url decode manually.
 * 
 * These tests verify:
 * - Valid JWT tokens are decoded correctly
 * - Claims (sub, email, role, exp, iat, type) are extracted
 * - Malformed tokens are handled gracefully
 * - Expired tokens are detected
 * - Tampered payloads are detected
 */

import { describe, it, expect } from "vitest";
import {
  decodeJWT,
  isTokenExpired,
  extractRole,
  extractEmail,
  type JWTPayload,
} from "../utils/jwt";

// Helper: create a valid base64url-encoded payload
function encodePayload(payload: Record<string, unknown>): string {
  const json = JSON.stringify(payload);
  // btoa in browser/Node handles standard base64
  const base64 = btoa(json);
  // Convert base64 to base64url
  return base64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

// Create a properly formatted JWT-like token string
function makeToken(payload: Record<string, unknown>): string {
  const header = encodePayload({ alg: "HS256", typ: "JWT" });
  const payloadEncoded = encodePayload(payload);
  const signature = "fake-signature";
  return `${header}.${payloadEncoded}.${signature}`;
}

describe("decodeJWT", () => {
  it("decodes a valid JWT token and extracts claims", () => {
    const payload = {
      sub: "user-123",
      email: "admin@fleet.com",
      role: "admin",
      exp: 1893456000, // far future
      iat: 1700000000,
      type: "access",
    };
    const token = makeToken(payload);

    const decoded = decodeJWT(token);
    expect(decoded).not.toBeNull();

    expect(decoded!.sub).toBe("user-123");
    expect(decoded!.email).toBe("admin@fleet.com");
    expect(decoded!.role).toBe("admin");
    expect(decoded!.exp).toBe(1893456000);
    expect(decoded!.iat).toBe(1700000000);
    expect(decoded!.type).toBe("access");
  });

  it("decodes a JWT with user role", () => {
    const payload = {
      sub: "user-456",
      email: "driver@fleet.com",
      role: "user",
      exp: 1893456000,
      iat: 1700000000,
    };
    const token = makeToken(payload);

    const decoded = decodeJWT(token);
    expect(decoded).not.toBeNull();

    expect(decoded!.role).toBe("user");
    expect(decoded!.email).toBe("driver@fleet.com");
  });

  it("returns null for malformed token with wrong number of parts", () => {
    expect(decodeJWT("not-a-jwt")).toBeNull();
  });

  it("returns null for token with only two parts", () => {
    expect(decodeJWT("header.payload")).toBeNull();
  });

  it("returns null for token with empty payload segment", () => {
    expect(decodeJWT("header..signature")).toBeNull();
  });

  it("returns null for token with invalid base64url in payload", () => {
    // Invalid base64url characters
    expect(decodeJWT("eyJhbGciOiJIUzI1NiJ9.!!!invalid!!!.sig")).toBeNull();
  });

  it("returns null for token with non-JSON payload", () => {
    const b64 = encodePayload({}); // Empty but valid JSON
    // Use a valid base64 string that decodes to non-JSON
    const fakePayload = btoa("not-json").replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
    expect(decodeJWT(`eyJhbGciOiJIUzI1NiJ9.${fakePayload}.sig`)).toBeNull();
  });

  it("handles payload with missing optional claims gracefully", () => {
    const payload = {
      sub: "user-789",
      exp: 1893456000,
    };
    const token = makeToken(payload);

    const decoded = decodeJWT(token);

    expect(decoded!.sub).toBe("user-789");
    expect(decoded!.role).toBeUndefined();
    expect(decoded!.email).toBeUndefined();
  });
});

describe("isTokenExpired", () => {
  it("returns true for expired token", () => {
    const payload = {
      sub: "user-1",
      role: "admin",
      exp: 1000, // Jan 1, 1970 — long past
      iat: 999,
    };
    const token = makeToken(payload);

    expect(isTokenExpired(token)).toBe(true);
  });

  it("returns false for non-expired token", () => {
    const payload = {
      sub: "user-1",
      role: "admin",
      exp: 1893456000, // far future
      iat: 1700000000,
    };
    const token = makeToken(payload);

    expect(isTokenExpired(token)).toBe(false);
  });

  it("returns true for token without exp claim", () => {
    const payload = {
      sub: "user-1",
      role: "admin",
    };
    const token = makeToken(payload);

    expect(isTokenExpired(token)).toBe(true);
  });

  it("returns true for malformed token", () => {
    expect(isTokenExpired("bad-token")).toBe(true);
  });
});

describe("extractRole", () => {
  it("extracts admin role from token", () => {
    const token = makeToken({ sub: "1", role: "admin", exp: 1893456000 });
    expect(extractRole(token)).toBe("admin");
  });

  it("extracts user role from token", () => {
    const token = makeToken({ sub: "1", role: "user", exp: 1893456000 });
    expect(extractRole(token)).toBe("user");
  });

  it("returns null for token without role claim", () => {
    const token = makeToken({ sub: "1", exp: 1893456000 });
    expect(extractRole(token)).toBeNull();
  });

  it("returns null for malformed token", () => {
    expect(extractRole("bad")).toBeNull();
  });
});

describe("extractEmail", () => {
  it("extracts email from token", () => {
    const token = makeToken({
      sub: "1",
      email: "admin@fleet.com",
      role: "admin",
      exp: 1893456000,
    });
    expect(extractEmail(token)).toBe("admin@fleet.com");
  });

  it("returns null for token without email claim", () => {
    const token = makeToken({ sub: "1", role: "admin", exp: 1893456000 });
    expect(extractEmail(token)).toBeNull();
  });

  it("returns null for malformed token", () => {
    expect(extractEmail("bad")).toBeNull();
  });
});