// Package jwt implements manual JWT generation and validation using only Go standard library crypto.
// No external JWT libraries — per the technical test requirement.
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

var jwtHeader = map[string]string{
	"alg": "HS256",
	"typ": "JWT",
}

type TokenService struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	now             func() time.Time // injectable clock for testing
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// NewTokenService creates a JWT service. Pass nil for nowFunc to use time.Now in production.
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
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		now:             now,
	}, nil
}

func (ts *TokenService) Generate(claims domain.Claims) (string, error) {
	now := ts.now()
	claims.IssuedAt = now.Unix()
	claims.ExpireAt = now.Add(ts.accessTokenTTL).Unix()

	if claims.Type == "refresh" {
		claims.ExpireAt = now.Add(ts.refreshTokenTTL).Unix()
	}

	headerJSON, err := json.Marshal(jwtHeader)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}
	headerEncoded := base64urlEncode(headerJSON)

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}
	payloadEncoded := base64urlEncode(payloadJSON)

	signingInput := headerEncoded + "." + payloadEncoded
	signature := computeHMAC(signingInput, ts.secret)
	signatureEncoded := base64urlEncode(signature)

	return headerEncoded + "." + payloadEncoded + "." + signatureEncoded, nil
}

func (ts *TokenService) Validate(tokenString string) (*domain.Claims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: expected 3 segments, got %d", domain.ErrTokenInvalid, len(parts))
	}

	headerEncoded := parts[0]
	payloadEncoded := parts[1]
	signatureEncoded := parts[2]

	signingInput := headerEncoded + "." + payloadEncoded
	expectedSignature := computeHMAC(signingInput, ts.secret)

	actualSignature, err := base64urlDecode(signatureEncoded)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode signature: %v", domain.ErrTokenInvalid, err)
	}

	// Constant-time comparison to prevent timing attacks.
	if !hmac.Equal(expectedSignature, actualSignature) {
		return nil, fmt.Errorf("%w: signature mismatch", domain.ErrTokenInvalid)
	}

	payloadJSON, err := base64urlDecode(payloadEncoded)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode payload: %v", domain.ErrTokenInvalid, err)
	}

	var claims domain.Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal claims: %v", domain.ErrTokenInvalid, err)
	}

	now := ts.now()
	if claims.ExpireAt < now.Unix() {
		return nil, fmt.Errorf("%w: token expired at %d, current time %d", domain.ErrTokenExpired, claims.ExpireAt, now.Unix())
	}

	return &claims, nil
}

func (ts *TokenService) ParseClaims(tokenString string) (*domain.Claims, error) {
	return ts.Validate(tokenString)
}

// base64urlEncode uses base64url without padding per JWT spec (RFC 7519).
func base64urlEncode(data []byte) string {
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

// base64urlDecode handles both padded and unpadded base64url input.
func base64urlDecode(s string) ([]byte, error) {
	s = strings.TrimRight(s, "=")
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	case 0:
		// already a multiple of 4
	case 1:
		return nil, fmt.Errorf("invalid base64url string: length %d", len(s))
	}
	return base64.URLEncoding.DecodeString(s)
}

func computeHMAC(message string, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return mac.Sum(nil)
}