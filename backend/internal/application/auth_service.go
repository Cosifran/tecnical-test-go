// Package application contains business logic (use cases / services).
//
// WHY a separate application layer: This package orchestrates domain operations
// without knowing about HTTP, databases, or JWT implementation details.
// It depends on domain interfaces, not concrete implementations. This makes
// the business logic testable with mocks and portable across infrastructure.
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

// TokenGenerator is an interface for JWT operations.
// The infrastructure/jwt package implements this interface.
//
// WHY an interface: So the AuthService can be tested without a real JWT
// implementation. In tests, we can pass a mock that generates predictable tokens.
// In production, we pass the real manual JWT implementation.
//
// Go tip: Interfaces should be defined where they're CONSUMED, not where they're
// IMPLEMENTED. This interface lives in the application package because that's
// where it's used. The jwt package doesn't know about this interface — it just
// happens to have the same method signatures (implicit satisfaction).
type TokenGenerator interface {
	// Generate creates a signed JWT token with the given claims.
	Generate(claims domain.Claims) (string, error)

	// Validate checks a JWT token and returns its claims if valid.
	Validate(tokenString string) (*domain.Claims, error)
}

// AuthService handles authentication use cases: login, token generation,
// and token refresh. It delegates JWT operations to a TokenGenerator
// and password verification to bcrypt (via a hash comparator).
type AuthService struct {
	// userRepo looks up users by email (for login) and by ID (for refresh).
	userRepo domain.UserRepository

	// tokenGen creates and validates JWT tokens.
	tokenGen TokenGenerator

	// bcryptCompare is a function that compares a plaintext password
	// with a bcrypt hash. In production, this is bcrypt.CompareHashAndPassword.
	// In tests, we can inject a mock that always returns true or false.
	//
	// WHY injectable: So we don't need a real database with hashed passwords
	// in unit tests. We just mock the comparison function.
	bcryptCompare func(hashedPassword, password []byte) error

	// accessTokenTTL and refreshTokenTTL determine token lifetimes.
	// These are configurable (typically 15 min and 7 days respectively).
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewAuthService creates a new AuthService with the given dependencies.
//
// Parameters:
//   - userRepo: for looking up user credentials
//   - tokenGen: for creating and validating JWT tokens
//   - bcryptCompare: function that verifies a password against a hash
//   - accessTTL: access token lifetime (15 minutes per spec)
//   - refreshTTL: refresh token lifetime (7 days per spec)
func NewAuthService(
	userRepo domain.UserRepository,
	tokenGen TokenGenerator,
	bcryptCompare func(hashedPassword, password []byte) error,
	accessTTL, refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		tokenGen:         tokenGen,
		bcryptCompare:    bcryptCompare,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// LoginResult contains the tokens returned by a successful login.
type LoginResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`  // always "Bearer"
	ExpiresIn    int64  `json:"expires_in"`   // seconds until access token expires
}

// Login authenticates a user with email and password.
//
// Flow:
//  1. Look up the user by email
//  2. Compare the provided password with the stored bcrypt hash
//  3. Generate an access token (15 min) and a refresh token (7 days)
//  4. Return both tokens
//
// Errors:
//   - ErrNotFound: no user with that email
//   - ErrUnauthorized: wrong password
func (s *AuthService) Login(email, password string) (*LoginResult, error) {
	// Step 1: Look up the user by email.
	user, err := s.userRepo.FindByEmail(context.Background(), email)
	if err != nil {
		// Don't reveal whether the email exists — generic error prevents
		// enumeration attacks ("is this email registered?").
		return nil, domain.ErrUnauthorized
	}

	// Step 2: Compare the provided password with the bcrypt hash.
	// bcrypt.CompareHashAndPassword returns nil on match, error on mismatch.
	if err := s.bcryptCompare([]byte(user.Password), []byte(password)); err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Step 3: Generate access token.
	now := time.Now()
	accessClaims := domain.Claims{
		Subject: user.ID,
		Email:   user.Email,
		Role:    user.Role,
		Type:    "access",
		IssuedAt: now.Unix(),
		ExpireAt: now.Add(s.accessTokenTTL).Unix(),
	}

	accessToken, err := s.tokenGen.Generate(accessClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Step 4: Generate refresh token.
	refreshClaims := domain.Claims{
		Subject: user.ID,
		Email:   user.Email,
		Role:    user.Role,
		Type:    "refresh",
		IssuedAt: now.Unix(),
		ExpireAt: now.Add(s.refreshTokenTTL).Unix(),
	}

	refreshToken, err := s.tokenGen.Generate(refreshClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}

// RefreshResult contains the new tokens returned by a successful refresh.
type RefreshResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Refresh validates a refresh token and issues new access + refresh tokens.
//
// Per the spec (with user clarification): Refresh token rotation is implemented.
// A new refresh token is issued on each refresh, and the old token remains valid
// until its natural expiration. This is "rotation" in the sense that the client
// should use the newest token, but we don't maintain a blacklist of old tokens
// (that would require server-side state, complicating the system for a 3-day test).
//
// Flow:
//  1. Validate the refresh token (signature + expiration)
//  2. Verify it's actually a refresh token (not an access token)
//  3. Look up the user to ensure they still exist
//  4. Generate new access + refresh tokens
//
// Errors:
//   - ErrTokenInvalid: malformed/invalid token, or wrong token type
//   - ErrTokenExpired: token past its expiration time
func (s *AuthService) Refresh(refreshToken string) (*RefreshResult, error) {
	// Step 1: Validate the token signature and expiration.
	claims, err := s.tokenGen.Validate(refreshToken)
	if err != nil {
		return nil, err
	}

	// Step 2: Verify it's a refresh token, not an access token.
	// Access tokens should NOT be used to obtain new tokens.
	if claims.Type != "refresh" {
		return nil, fmt.Errorf("%w: expected refresh token, got %s token", domain.ErrTokenInvalid, claims.Type)
	}

	// Step 3: Look up the user to ensure they still exist.
	// A user might have been deleted — we don't want to issue tokens
	// for non-existent users.
	user, err := s.userRepo.FindByID(context.Background(), claims.Subject)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Step 4: Generate new tokens with updated timestamps.
	now := time.Now()
	accessClaims := domain.Claims{
		Subject: user.ID,
		Email:   user.Email,
		Role:    user.Role,
		Type:    "access",
		IssuedAt: now.Unix(),
		ExpireAt: now.Add(s.accessTokenTTL).Unix(),
	}

	accessToken, err := s.tokenGen.Generate(accessClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshClaims := domain.Claims{
		Subject: user.ID,
		Email:   user.Email,
		Role:    user.Role,
		Type:    "refresh",
		IssuedAt: now.Unix(),
		ExpireAt: now.Add(s.refreshTokenTTL).Unix(),
	}

	newRefreshToken, err := s.tokenGen.Generate(refreshClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}