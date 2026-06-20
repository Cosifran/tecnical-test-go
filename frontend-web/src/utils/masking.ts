/**
 * Client-side device ID masking utility.
 *
 * Mirrors the backend MaskDeviceID function EXACTLY.
 * Format: DEV-****-{last4} where {last4} is the last 4 characters of the raw ID.
 *
 * WHY client-side masking: The backend always sends raw device IDs in the API.
 * Frontend applies masking based on the user's role — non-admin users see
 * masked IDs. This keeps the masking logic consistent with the backend.
 *
 * Edge case: IDs shorter than 4 characters produce "DEV-****-????".
 */

export function maskDeviceId(raw: string): string {
  if (raw.length < 4) {
    return "DEV-****-????";
  }
  const last4 = raw.slice(-4);
  return `DEV-****-${last4}`;
}