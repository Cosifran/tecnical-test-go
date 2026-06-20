/**
 * JWT payload decoding utility.
 *
 * IMPORTANT: This implements JWT decoding WITHOUT external libraries,
 * as required by the project specification. Only the payload is decoded;
 * signature verification happens on the backend. The frontend extracts
 * claims (role, email, exp) for UI decisions only.
 *
 * Uses standard browser/base64 utilities — no jwt-decode, no jsonwebtoken.
 */

export interface JWTPayload {
  sub?: string;
  email?: string;
  role?: string;
  exp?: number;
  iat?: number;
  type?: string;
  [key: string]: unknown;
}

/**
 * Decode the payload segment of a JWT token.
 * Does NOT validate the signature — that's the backend's job.
 * Returns null for malformed tokens.
 */
export function decodeJWT(token: string): JWTPayload | null {
  if (!token || typeof token !== "string") {
    return null;
  }

  const parts = token.split(".");
  if (parts.length !== 3) {
    return null;
  }

  const payloadB64 = parts[1];
  if (!payloadB64) {
    return null;
  }

  try {
    // Convert base64url to base64
    let base64 = payloadB64.replace(/-/g, "+").replace(/_/g, "/");
    // Add padding
    const padding = base64.length % 4;
    if (padding === 2) {
      base64 += "==";
    } else if (padding === 3) {
      base64 += "=";
    }

    const json = atob(base64);
    const payload: JWTPayload = JSON.parse(json);
    return payload;
  } catch {
    return null;
  }
}

/**
 * Check if a JWT token is expired.
 * Returns true if the token has no exp claim, is malformed,
 * or if the current time is past the exp claim.
 */
export function isTokenExpired(token: string): boolean {
  const payload = decodeJWT(token);
  if (!payload || payload.exp === undefined) {
    return true;
  }
  return Date.now() / 1000 >= payload.exp;
}

/**
 * Extract the user role from a JWT token.
 * Returns null if the token is malformed or has no role claim.
 */
export function extractRole(token: string): string | null {
  const payload = decodeJWT(token);
  if (!payload || payload.role === undefined) {
    return null;
  }
  return payload.role;
}

/**
 * Extract the email from a JWT token.
 * Returns null if the token is malformed or has no email claim.
 */
export function extractEmail(token: string): string | null {
  const payload = decodeJWT(token);
  if (!payload || payload.email === undefined) {
    return null;
  }
  return payload.email as string;
}