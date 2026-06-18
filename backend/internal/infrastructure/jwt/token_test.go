// Package jwt_test tests the manual JWT implementation.
//
// These tests are CRITICAL because JWT authentication is a security-critical
// component. We test every edge case with table-driven tests (a Go pattern
// where you define a slice of test cases and loop over them with t.Run).
//
// WHY table-driven tests: They make it easy to add new test cases without
// writing new functions. Each case is a row in the table. When a bug is found,
// add a new case. When refactoring, all cases still pass or you know you broke something.
//
// WHAT we test:
//   - Token generation produces valid tokens that can be validated
//   - Valid tokens return correct claims
//   - Expired tokens return ErrTokenExpired
//   - Tampered tokens (modified payload) return ErrTokenInvalid
//   - Malformed tokens (wrong number of segments) return ErrTokenInvalid
//   - Tokens signed with a different secret return ErrTokenInvalid
//   - Tokens with invalid base64 return ErrTokenInvalid
//   - Refresh tokens have longer TTL than access tokens
package jwt_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
	"github.com/francisco/fleet-monitor/internal/infrastructure/jwt"
)

// fixedTime is a deterministic time value used in tests.
// All tests that care about time will use this instead of time.Now(),
// so results don't depend on when the test runs.
var fixedTime = time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)

// fixedTimePlus15Min is 15 minutes after fixedTime — the exact moment
// when an access token issued at fixedTime would expire.
var fixedTimePlus15Min = fixedTime.Add(15 * time.Minute)

// validSecret is a 32+ byte secret used for most tests.
// It satisfies the minimum security requirement of 256 bits.
const validSecret = "this-is-a-very-secure-secret-32b"

// newTestTokenService creates a TokenService with a fixed clock for testing.
// The injectable clock means we can test expiration without sleeping.
func newTestTokenService(secret string, now func() time.Time) (*jwt.TokenService, error) {
	return jwt.NewTokenService(
		secret,
		15*time.Minute,  // access token TTL
		7*24*time.Hour,  // refresh token TTL (7 days)
		now,
	)
}

// convenience helper that uses the default validSecret and fixedTime
func makeTokenService() *jwt.TokenService {
	ts, err := newTestTokenService(validSecret, func() time.Time { return fixedTime })
	if err != nil {
		panic(fmt.Sprintf("failed to create TokenService: %v", err))
	}
	return ts
}

// helper to generate a valid token with default claims
func makeValidClaims() domain.Claims {
	return domain.Claims{
		Subject: "user-123",
		Email:   "admin@example.com",
		Role:    "admin",
		Type:    "access",
	}
}

// TestGenerateAndValidate tests the happy path: generate a token, then validate it.
// If this fails, the entire JWT system is broken.
func TestGenerateAndValidate(t *testing.T) {
	ts := makeTokenService()
	claims := makeValidClaims()

	token, err := ts.Generate(claims)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if token == "" {
		t.Fatal("Generate() returned empty token")
	}

	// Validate the token we just generated
	validatedClaims, err := ts.Validate(token)
	if err != nil {
		t.Fatalf("Validate() failed for valid token: %v", err)
	}

	// Check that claims match what we put in
	if validatedClaims.Subject != claims.Subject {
		t.Errorf("Subject mismatch: got %q, want %q", validatedClaims.Subject, claims.Subject)
	}
	if validatedClaims.Email != claims.Email {
		t.Errorf("Email mismatch: got %q, want %q", validatedClaims.Email, claims.Email)
	}
	if validatedClaims.Role != claims.Role {
		t.Errorf("Role mismatch: got %q, want %q", validatedClaims.Role, claims.Role)
	}
}

// TestGenerateSetsTimestamps verifies that Generate correctly sets
// iat (issued-at) and exp (expires-at) based on the injectable clock.
func TestGenerateSetsTimestamps(t *testing.T) {
	ts := makeTokenService()
	claims := makeValidClaims()

	token, err := ts.Generate(claims)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	validatedClaims, err := ts.Validate(token)
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// iat should be fixedTime.Unix()
	expectedIAT := fixedTime.Unix()
	if validatedClaims.IssuedAt != expectedIAT {
		t.Errorf("iat mismatch: got %d, want %d", validatedClaims.IssuedAt, expectedIAT)
	}

	// exp should be fixedTime + 15 minutes
	expectedEXP := fixedTime.Add(15 * time.Minute).Unix()
	if validatedClaims.ExpireAt != expectedEXP {
		t.Errorf("exp mismatch: got %d, want %d", validatedClaims.ExpireAt, expectedEXP)
	}
}

// TestRefreshTokenTTL verifies that refresh tokens have a longer TTL than access tokens.
func TestRefreshTokenTTL(t *testing.T) {
	ts := makeTokenService()
	claims := makeValidClaims()
	claims.Type = "refresh"

	token, err := ts.Generate(claims)
	if err != nil {
		t.Fatalf("Generate() failed for refresh token: %v", err)
	}

	validatedClaims, err := ts.Validate(token)
	if err != nil {
		t.Fatalf("Validate() failed for refresh token: %v", err)
	}

	// Refresh token exp should be fixedTime + 7 days
	expectedEXP := fixedTime.Add(7 * 24 * time.Hour).Unix()
	if validatedClaims.ExpireAt != expectedEXP {
		t.Errorf("refresh token exp mismatch: got %d, want %d", validatedClaims.ExpireAt, expectedEXP)
	}
}

// TestTokenValidation tests ALL token validation scenarios using table-driven tests.
// This is the most important test in the entire system — it verifies that
// our JWT implementation correctly rejects invalid tokens and accepts valid ones.
//
// HOW table-driven tests work in Go:
//  1. Define a struct with named fields for inputs and expected outputs
//  2. Create a slice of test cases (the "table")
//  3. Loop over each case with t.Run, which creates a subtest
//  4. Each subtest runs independently — one failure doesn't stop the others
func TestTokenValidation(t *testing.T) {
	// Create a token service with our fixed clock
	ts := makeTokenService()

	// Pre-generate some tokens we'll use in test cases
	validClaims := makeValidClaims()

	// Generate a valid access token for testing
	validToken, err := ts.Generate(validClaims)
	if err != nil {
		t.Fatalf("Failed to generate valid token: %v", err)
	}

	// Generate a valid refresh token
	refreshClaims := makeValidClaims()
	refreshClaims.Type = "refresh"
	validRefreshToken, err := ts.Generate(refreshClaims)
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	// Create a token signed with a DIFFERENT secret (for wrong-key test)
	tsWrongKey, _ := newTestTokenService("another-very-secure-secret-32b-plus",
		func() time.Time { return fixedTime })
	tokenWrongKey, err := tsWrongKey.Generate(validClaims)
	if err != nil {
		t.Fatalf("Failed to generate token with wrong key: %v", err)
	}

	// Create a token and then tamper with the payload
	validTokenParts := splitToken(validToken)
	tamperedPayload := validTokenParts[0] + "." +
		modifyPayload(validTokenParts[1]) + "." +
		validTokenParts[2]

	tests := []struct {
		name      string // descriptive name for the subtest
		token     string // the token string to validate
		wantErr   error  // the expected error type (nil = no error)
		wantSub   string // expected Subject if token is valid
	}{
		{
			name:    "valid access token",
			token:   validToken,
			wantErr: nil,
			wantSub: "user-123",
		},
		{
			name:    "valid refresh token",
			token:   validRefreshToken,
			wantErr: nil,
			wantSub: "user-123",
		},
		{
			name:    "token signed with wrong secret",
			token:   tokenWrongKey,
			wantErr: domain.ErrTokenInvalid,
		},
		{
			name:    "tampered payload (modified claims)",
			token:   tamperedPayload,
			wantErr: domain.ErrTokenInvalid,
		},
		{
			name:    "malformed token with 2 segments",
			token:   "a.b",
			wantErr: domain.ErrTokenInvalid,
		},
		{
			name:    "malformed token with 4 segments",
			token:   "a.b.c.d",
			wantErr: domain.ErrTokenInvalid,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: domain.ErrTokenInvalid,
		},
		{
			name:    "token with invalid base64 in signature",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLTEyMyJ9.!!!invalid!!!",
			wantErr: domain.ErrTokenInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ts.Validate(tt.token)

			if tt.wantErr != nil {
				// We expect an error
				if err == nil {
					t.Fatalf("expected error containing %v, got nil", tt.wantErr)
				}
				// Use errors.Is to check error type (works with wrapped errors)
				if !isError(err, tt.wantErr) {
					t.Errorf("error type mismatch: got %v, want %v", err, tt.wantErr)
				}
			} else {
				// We expect success
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if claims.Subject != tt.wantSub {
					t.Errorf("subject mismatch: got %q, want %q", claims.Subject, tt.wantSub)
				}
			}
		})
	}
}

// TestExpiredToken tests that tokens past their expiration are rejected
// with ErrTokenExpired (not ErrTokenInvalid).
func TestExpiredToken(t *testing.T) {
	// Step 1: Generate a token at fixedTime
	// The token will have exp = fixedTime + 15 min
	genTS, _ := newTestTokenService(validSecret, func() time.Time { return fixedTime })
	validClaims := makeValidClaims()
	token, err := genTS.Generate(validClaims)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Step 2: Validate with a clock that's PAST the expiration time
	// This token should be expired
	valTS, _ := newTestTokenService(validSecret, func() time.Time {
		return fixedTime.Add(16 * time.Minute) // 1 minute past expiration
	})

	_, err = valTS.Validate(token)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}

	if !isError(err, domain.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got: %v", err)
	}
}

// TestTokenNotYetExpired tests that tokens at the EXACT moment of expiration
// are still accepted. This tests the boundary condition.
func TestTokenNotYetExpired(t *testing.T) {
	// Step 1: Generate a token at fixedTime
	genTS, _ := newTestTokenService(validSecret, func() time.Time { return fixedTime })
	validClaims := makeValidClaims()
	token, err := genTS.Generate(validClaims)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Step 2: Validate at the exact expiration time (should still be valid)
	valTS, _ := newTestTokenService(validSecret, func() time.Time {
		return fixedTime.Add(15 * time.Minute) // exactly at expiration
	})

	claims, err := valTS.Validate(token)
	if err != nil {
		t.Fatalf("expected token to be valid at expiration boundary, got error: %v", err)
	}
	if claims.Subject != "user-123" {
		t.Errorf("subject mismatch: got %q, want %q", claims.Subject, "user-123")
	}
}

// TestSecretTooShort verifies that creating a TokenService with a
// short secret fails. This is a security requirement.
func TestSecretTooShort(t *testing.T) {
	_, err := jwt.NewTokenService(
		"short",                          // too short — needs >= 32 bytes
		15*time.Minute,
		7*24*time.Hour,
		nil,
	)
	if err == nil {
		t.Fatal("expected error for short secret, got nil")
	}
}

// TestSecretMinimumLength verifies that a 32-character secret is accepted.
func TestSecretMinimumLength(t *testing.T) {
	_, err := jwt.NewTokenService(
		"12345678901234567890123456789012", // exactly 32 bytes
		15*time.Minute,
		7*24*time.Hour,
		nil,
	)
	if err != nil {
		t.Fatalf("expected 32-byte secret to be accepted, got error: %v", err)
	}
}

// TestRoundTrip tests that generate → validate produces consistent claims
// for both access and refresh token types.
func TestRoundTrip(t *testing.T) {
	ts := makeTokenService()

	tests := []struct {
		name       string
		tokenType  string
		wantTTL    time.Duration
	}{
		{
			name:      "access token round-trip",
			tokenType: "access",
			wantTTL:   15 * time.Minute,
		},
		{
			name:      "refresh token round-trip",
			tokenType: "refresh",
			wantTTL:   7 * 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := domain.Claims{
				Subject: "round-trip-user",
				Email:   "test@example.com",
				Role:    "user",
				Type:    tt.tokenType,
			}

			token, err := ts.Generate(claims)
			if err != nil {
				t.Fatalf("Generate() error: %v", err)
			}

			validated, err := ts.Validate(token)
			if err != nil {
				t.Fatalf("Validate() error: %v", err)
			}

			// Verify all claims survive the round-trip
			if validated.Subject != claims.Subject {
				t.Errorf("sub: got %q, want %q", validated.Subject, claims.Subject)
			}
			if validated.Email != claims.Email {
				t.Errorf("email: got %q, want %q", validated.Email, claims.Email)
			}
			if validated.Role != claims.Role {
				t.Errorf("role: got %q, want %q", validated.Role, claims.Role)
			}
			if validated.Type != claims.Type {
				t.Errorf("type: got %q, want %q", validated.Type, claims.Type)
			}

			// Verify the TTL is approximately correct (within 1 second)
			expectedDur := validated.ExpireAt - validated.IssuedAt
			actualDur := int64(tt.wantTTL.Seconds())
			if expectedDur != actualDur {
				t.Errorf("TTL: got %d seconds, want %d seconds", expectedDur, actualDur)
			}
		})
	}
}

// TestDifferentTokenTypesHaveDifferentClaims verifies that access tokens
// and refresh tokens for the same user have different exp timestamps.
func TestDifferentTokenTypesHaveDifferentClaims(t *testing.T) {
	ts := makeTokenService()

	accessClaims := makeValidClaims()
	accessClaims.Type = "access"
	accessToken, _ := ts.Generate(accessClaims)

	refreshClaims := makeValidClaims()
	refreshClaims.Type = "refresh"
	refreshToken, _ := ts.Generate(refreshClaims)

	accessValidated, _ := ts.Validate(accessToken)
	refreshValidated, _ := ts.Validate(refreshToken)

	// Refresh token should have a MUCH later expiration
	if refreshValidated.ExpireAt <= accessValidated.ExpireAt {
		t.Errorf("refresh token exp (%d) should be after access token exp (%d)",
			refreshValidated.ExpireAt, accessValidated.ExpireAt)
	}
}

// Helper functions

// splitToken splits a JWT string by "." and returns the three segments.
// Returns empty slice if the token format is wrong.
func splitToken(token string) []string {
	return strings.Split(token, ".")
}

// modifyPayload takes a base64url-encoded payload and modifies it
// by changing the subject claim. This simulates a token tampering attack
// where someone modifies the payload without updating the signature.
func modifyPayload(payload string) string {
	// For tampering tests, we simply flip a character in the base64 payload.
	// This changes the decoded JSON, making the signature invalid.
	if len(payload) == 0 {
		return "modified"
	}
	// Change the first character to something else
	runes := []rune(payload)
	runes[0] = runes[0] + 1
	return string(runes)
}

// isError checks if an error wraps the target error.
// Uses errors.Is for proper wrapped error comparison.
func isError(err, target error) bool {
	return errors.Is(err, target)
}