// Package jwt implements manual JSON Web Token (JWT) generation and validation
// using ONLY Go standard library crypto primitives. No external JWT libraries.
//
// WHY manual JWT: The technical test EXPLICITLY requires this. It demonstrates
// understanding of the JWT specification and HMAC-SHA256 signing. The real
// value is in the test coverage — every edge case (tampering, expiration,
// malformed tokens) is tested with table-driven tests.
//
// HOW JWT WORKS (for someone new to the spec):
//
//	A JWT is three base64url-encoded segments separated by dots:
//	  header.payload.signature
//
//	1. Header: {"alg":"HS256","typ":"JWT"} — tells the verifier which algorithm was used.
//	2. Payload: {"sub":"user-uuid","email":"...","role":"admin","iat":1700000000,"exp":1700000900}
//	   — the actual data (claims) being transmitted.
//	3. Signature: HMAC-SHA256(base64url(header) + "." + base64url(payload), secret)
//	   — proves the token hasn't been tampered with.
//
//	Validation reverses this: recompute the signature, compare with constant-time
//	comparison, then check expiration.
package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

// jwtHeader is the fixed header for all tokens.
// JWT spec requires "alg" and "typ" fields. We always use HS256.
var jwtHeader = map[string]string{
	"alg": "HS256",
	"typ": "JWT",
}

// TokenService provides JWT generation and validation using HMAC-SHA256.
// It uses no external JWT library — just crypto/hmac + encoding/base64.
type TokenService struct {
	// secret is the HMAC-SHA256 signing key. Must be >= 32 bytes (256 bits).
	secret []byte

	// accessTokenTTL is how long access tokens are valid.
	accessTokenTTL time.Duration

	// refreshTokenTTL is how long refresh tokens are valid.
	refreshTokenTTL time.Duration

	// now is a function that returns the current time.
	// WHY injectable: In production, this is time.Now. In tests, this is
	// a fixed time so we can test expiration deterministically without sleeping.
	// This is a common Go testing pattern called "injectable clock."
	now func() time.Time
}

// TokenResponse is the JSON structure returned by the login and refresh endpoints.
// Includes both tokens plus metadata the client needs.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`    // always "Bearer"
	ExpiresIn    int64  `json:"expires_in"`     // seconds until access token expires
}

// NewTokenService creates a TokenService with the given configuration.
//
// Parameters:
//   - secret: HMAC-SHA256 signing key, must be >= 32 bytes
//   - accessTTL: how long access tokens are valid (typically 15 minutes)
//   - refreshTTL: how long refresh tokens are valid (typically 7 days)
//   - nowFunc: injectable clock for testing. Pass nil for production (uses time.Now).
//
// WHY nowFunc as parameter: This makes the TokenService testable. Tests pass
// a fixed time; production passes nil. No global state, no interface, clean.
func NewTokenService(secret string, accessTTL, refreshTTL time.Duration, nowFunc func() time.Time) (*TokenService, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT secret must be at least 32 bytes, got %d", len(secret))
	}

	now := nowFunc
	if now == nil {
		now = time.Now
	}

	return &TokenService{
		secret:          []byte(secret),
		accessTokenTTL: accessTTL,
		refreshTokenTTL: refreshTTL,
		now:             now,
	}, nil
}

// Generate creates a signed JWT token string with the given claims.
//
// The process:
//  1. Encode the header as base64url
//  2. Fill in iat/exp timestamps
//  3. Encode the payload as base64url
//  4. Compute HMAC-SHA256(header.payload, secret)
//  5. Encode the signature as base64url
//  6. Join all three segments with dots
//
// Returns the complete JWT string: "header.payload.signature"
func (ts *TokenService) Generate(claims domain.Claims) (string, error) {
	// Set timestamps based on injectable clock
	now := ts.now()
	claims.IssuedAt = now.Unix()
	claims.ExpireAt = now.Add(ts.accessTokenTTL).Unix()

	// If this is a refresh token, use the longer TTL
	if claims.Type == "refresh" {
		claims.ExpireAt = now.Add(ts.refreshTokenTTL).Unix()
	}

	// Step 1: Encode the header
	headerJSON, err := json.Marshal(jwtHeader)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}
	headerEncoded := base64urlEncode(headerJSON)

	// Step 2: Encode the payload
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}
	payloadEncoded := base64urlEncode(payloadJSON)

	// Step 3: Compute the signature
	// The signing input is "header.payload" (the first two segments with a dot).
	signingInput := headerEncoded + "." + payloadEncoded
	signature := computeHMAC(signingInput, ts.secret)
	signatureEncoded := base64urlEncode(signature)

	// Step 4: Combine segments
	return headerEncoded + "." + payloadEncoded + "." + signatureEncoded, nil
}

// Validate checks a JWT token string and returns its claims if valid.
//
// Validation steps:
//  1. Split the token into 3 segments (header, payload, signature)
//  2. Recompute the HMAC-SHA256 signature from header + payload
//  3. Compare with constant-time comparison (prevents timing attacks)
//  4. Decode the payload into Claims
//  5. Check expiration against the injectable clock
//
// Returns:
//   - Claims: the decoded token data
//   - error: ErrTokenInvalid for tampered/malformed tokens,
//     ErrTokenExpired for expired tokens
func (ts *TokenService) Validate(tokenString string) (*domain.Claims, error) {
	// Step 1: Split into segments. A valid JWT must have exactly 3 parts.
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: expected 3 segments, got %d", domain.ErrTokenInvalid, len(parts))
	}

	headerEncoded := parts[0]
	payloadEncoded := parts[1]
	signatureEncoded := parts[2]

	// Step 2: Recompute the expected signature.
	signingInput := headerEncoded + "." + payloadEncoded
	expectedSignature := computeHMAC(signingInput, ts.secret)

	// Step 3: Decode the provided signature from base64url.
	actualSignature, err := base64urlDecode(signatureEncoded)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode signature: %v", domain.ErrTokenInvalid, err)
	}

	// Step 4: Compare signatures using CONSTANT-TIME comparison.
	// WHY hmac.Equal and not bytes.Equal: Regular comparison leaks timing
	// information — an attacker can figure out how many bytes of the signature
	// are correct by measuring response time. hmac.Equal takes the same time
	// regardless of how many bytes match, preventing this attack.
	if !hmac.Equal(expectedSignature, actualSignature) {
		return nil, fmt.Errorf("%w: signature mismatch", domain.ErrTokenInvalid)
	}

	// Step 5: Decode the payload into Claims.
	payloadJSON, err := base64urlDecode(payloadEncoded)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode payload: %v", domain.ErrTokenInvalid, err)
	}

	var claims domain.Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal claims: %v", domain.ErrTokenInvalid, err)
	}

	// Step 6: Check expiration against the injectable clock.
	// WHY check exp but NOT iat: The spec requires rejecting expired tokens.
	// We don't check iat (issued-at) because a token created "in the past"
	// relative to the server's clock is not a security concern — the exp
	// check ensures the token isn't being used past its lifetime.
	now := ts.now()
	if claims.ExpireAt < now.Unix() {
		return nil, fmt.Errorf("%w: token expired at %d, current time %d", domain.ErrTokenExpired, claims.ExpireAt, now.Unix())
	}

	return &claims, nil
}

// ParseClaims is a convenience method that validates a token and extracts claims.
// It's an alias for Validate — provided for readability at call sites.
func (ts *TokenService) ParseClaims(tokenString string) (*domain.Claims, error) {
	return ts.Validate(tokenString)
}

// base64urlEncode encodes bytes using base64url encoding (RFC 4648) WITHOUT padding.
//
// WHY no padding: The JWT specification (RFC 7519) uses base64url encoding
// without padding. Standard base64 uses '=' for padding, which is not URL-safe.
// base64.URLEncoding.WithPadding(base64.NoPadding) gives us exactly what JWT needs.
func base64urlEncode(data []byte) string {
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

// base64urlDecode decodes a base64url-encoded string.
// Handles both padded and unpadded input for robustness.
// JWT spec (RFC 7519) requires base64url WITHOUT padding, but
// we handle padded input too for compatibility.
func base64urlDecode(s string) ([]byte, error) {
	// Remove any existing padding first (some JWT implementations add it)
	s = strings.TrimRight(s, "=")
	// Add padding to make the length a multiple of 4, as required by base64.
	// base64.URLEncoding.DecodeString requires proper padding.
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	case 0:
		// already a multiple of 4, no padding needed
	case 1:
		// Invalid base64 — can't have 1 remainder byte
		return nil, fmt.Errorf("invalid base64url string: length %d", len(s))
	}
	return base64.URLEncoding.DecodeString(s)
}

// computeHMAC calculates the HMAC-SHA256 of the given message using the given key.
// This is the core signing primitive — the same function is used for both
// generation (signing) and validation (recomputing the expected signature).
func computeHMAC(message string, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return mac.Sum(nil)
}