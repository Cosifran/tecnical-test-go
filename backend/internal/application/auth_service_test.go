package application_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/francisco/fleet-monitor/internal/application"
	"github.com/francisco/fleet-monitor/internal/domain"
)

// ============================================================
// Mock dependencies
// ============================================================

// mockUserRepository implements domain.UserRepository for testing.
// We define exactly the methods the service needs, and control
// what they return via the test table.
type mockUserRepository struct {
	users         map[string]*domain.User // keyed by email
	usersByID     map[string]*domain.User // keyed by ID
	findByEmailErr error
	findByIDErr    error
}

func newMockUserRepo() *mockUserRepository {
	return &mockUserRepository{
		users:     make(map[string]*domain.User),
		usersByID: make(map[string]*domain.User),
	}
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.findByEmailErr != nil {
		return nil, m.findByEmailErr
	}
	user, ok := m.users[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	user, ok := m.usersByID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	m.users[user.Email] = user
	m.usersByID[user.ID] = user
	return nil
}

// mockTokenGenerator implements application.TokenGenerator for testing.
type mockTokenGenerator struct {
	// tokens maps claims to a predictable token string
	// for validation: validToken → claims
	generateErr  error
	validateErr  error
	validateClaims *domain.Claims
}

func (m *mockTokenGenerator) Generate(claims domain.Claims) (string, error) {
	if m.generateErr != nil {
		return "", m.generateErr
	}
	// Return a predictable token string based on the subject
	return fmt.Sprintf("mock-token-%s-%s", claims.Subject, claims.Type), nil
}

func (m *mockTokenGenerator) Validate(tokenString string) (*domain.Claims, error) {
	if m.validateErr != nil {
		return nil, m.validateErr
	}
	if m.validateClaims != nil {
		return m.validateClaims, nil
	}
	// Default: return a claims based on the token string
	return &domain.Claims{
		Subject: "user-123",
		Email:   "admin@example.com",
		Role:    "admin",
		Type:    "refresh",
	}, nil
}

// mockBcryptCompare simulates bcrypt password comparison.
// In tests, we control whether it returns nil (match) or error (mismatch).
type mockBcryptCompare struct {
	match bool
	err   error
}

func (m *mockBcryptCompare) Compare(hashedPassword, password []byte) error {
	if m.err != nil {
		return m.err
	}
	if m.match {
		return nil
	}
	return fmt.Errorf("password mismatch")
}

// ============================================================
// Helpers
// ============================================================

// createTestUser creates a domain.User for testing.
func createTestUser(id, email, role string) *domain.User {
	return &domain.User{
		ID:        id,
		Email:     email,
		Password:  "$2a$12$hashedpassword",  // simulated bcrypt hash
		Role:      role,
		CreatedAt: time.Now(),
	}
}

// createAuthService creates an AuthService with mock dependencies.
func createAuthService(repo *mockUserRepository, gen *mockTokenGenerator, bcryptResult *mockBcryptCompare) *application.AuthService {
	return application.NewAuthService(
		repo,
		gen,
		bcryptResult.Compare,
		15*time.Minute,
		7*24*time.Hour,
	)
}

// addTestUserToRepo adds a user to the mock repository.
func addTestUserToRepo(repo *mockUserRepository, user *domain.User) {
	repo.users[user.Email] = user
	repo.usersByID[user.ID] = user
}

// ============================================================
// Tests
// ============================================================

// TestLogin_Success tests that a valid email/password combination
// returns both access and refresh tokens.
func TestLogin_Success(t *testing.T) {
	repo := newMockUserRepo()
	user := createTestUser("user-1", "admin@example.com", "admin")
	addTestUserToRepo(repo, user)

	gen := &mockTokenGenerator{}
	bcryptResult := &mockBcryptCompare{match: true}

	service := createAuthService(repo, gen, bcryptResult)

	result, err := service.Login("admin@example.com", "correctpassword")
	if err != nil {
		t.Fatalf("Login() failed: %v", err)
	}

	if result.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if result.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	if result.TokenType != "Bearer" {
		t.Errorf("TokenType: got %q, want %q", result.TokenType, "Bearer")
	}
	if result.ExpiresIn <= 0 {
		t.Error("ExpiresIn should be positive")
	}
}

// TestLogin_InvalidPassword tests that wrong credentials return ErrUnauthorized.
func TestLogin_InvalidPassword(t *testing.T) {
	repo := newMockUserRepo()
	user := createTestUser("user-1", "admin@example.com", "admin")
	addTestUserToRepo(repo, user)

	gen := &mockTokenGenerator{}
	bcryptResult := &mockBcryptCompare{match: false} // password doesn't match

	service := createAuthService(repo, gen, bcryptResult)

	_, err := service.Login("admin@example.com", "wrongpassword")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got: %v", err)
	}
}

// TestLogin_UserNotFound tests that a non-existent email returns ErrUnauthorized.
// WHY ErrUnauthorized and not ErrNotFound: We don't want to reveal whether
// an email is registered — that's an information leak that enables enumeration.
func TestLogin_UserNotFound(t *testing.T) {
	repo := newMockUserRepo() // empty repo — no users

	gen := &mockTokenGenerator{}
	bcryptResult := &mockBcryptCompare{match: true}

	service := createAuthService(repo, gen, bcryptResult)

	_, err := service.Login("nonexistent@example.com", "password")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got: %v", err)
	}
}

// TestLogin_TableDriven tests multiple login scenarios using table-driven tests.
func TestLogin_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		password     string
		userExists   bool
		passwordMatch bool
		wantErr      error
	}{
		{
			name:         "valid credentials",
			email:        "admin@example.com",
			password:     "correctpassword",
			userExists:   true,
			passwordMatch: true,
			wantErr:      nil,
		},
		{
			name:         "wrong password",
			email:        "admin@example.com",
			password:     "wrongpassword",
			userExists:   true,
			passwordMatch: false,
			wantErr:      domain.ErrUnauthorized,
		},
		{
			name:         "nonexistent user",
			email:        "nobody@example.com",
			password:     "password",
			userExists:   false,
			passwordMatch: true, // irrelevant — user doesn't exist
			wantErr:      domain.ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockUserRepo()
			if tt.userExists {
				user := createTestUser("user-1", tt.email, "admin")
				addTestUserToRepo(repo, user)
			}

			gen := &mockTokenGenerator{}
			bcryptResult := &mockBcryptCompare{match: tt.passwordMatch}
			service := createAuthService(repo, gen, bcryptResult)

			result, err := service.Login(tt.email, tt.password)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected %v, got: %v", tt.wantErr, err)
				}
				if result != nil {
					t.Error("expected nil result on error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.AccessToken == "" {
					t.Error("AccessToken should not be empty")
				}
				if result.RefreshToken == "" {
					t.Error("RefreshToken should not be empty")
				}
			}
		})
	}
}

// TestRefresh_Success tests the happy path of token refresh.
func TestRefresh_Success(t *testing.T) {
	repo := newMockUserRepo()
	user := createTestUser("user-123", "admin@example.com", "admin")
	addTestUserToRepo(repo, user)

	// Mock token generator returns valid refresh claims
	gen := &mockTokenGenerator{
		validateClaims: &domain.Claims{
			Subject: "user-123",
			Email:   "admin@example.com",
			Role:    "admin",
			Type:    "refresh",
		},
	}
	bcryptResult := &mockBcryptCompare{match: true}

	service := createAuthService(repo, gen, bcryptResult)

	result, err := service.Refresh("mock-refresh-token")
	if err != nil {
		t.Fatalf("Refresh() failed: %v", err)
	}

	if result.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if result.RefreshToken == "" {
		t.Error("RefreshToken should not be empty — rotation requires new refresh token")
	}
	if result.TokenType != "Bearer" {
		t.Errorf("TokenType: got %q, want %q", result.TokenType, "Bearer")
	}
}

// TestRefresh_WithAccessToken tests that using an access token for refresh
// is rejected with ErrTokenInvalid.
func TestRefresh_WithAccessToken(t *testing.T) {
	repo := newMockUserRepo()

	// Mock returns ACCESS token claims, not refresh
	gen := &mockTokenGenerator{
		validateClaims: &domain.Claims{
			Subject: "user-123",
			Type:    "access", // WRONG — should be "refresh"
		},
	}
	bcryptResult := &mockBcryptCompare{match: true}

	service := createAuthService(repo, gen, bcryptResult)

	_, err := service.Refresh("mock-access-token")
	if !errors.Is(err, domain.ErrTokenInvalid) {
		t.Errorf("expected ErrTokenInvalid for access token refresh, got: %v", err)
	}
}

// TestRefresh_ExpiredToken tests that an expired refresh token is rejected.
func TestRefresh_ExpiredToken(t *testing.T) {
	repo := newMockUserRepo()

	gen := &mockTokenGenerator{
		validateErr: domain.ErrTokenExpired,
	}
	bcryptResult := &mockBcryptCompare{match: true}

	service := createAuthService(repo, gen, bcryptResult)

	_, err := service.Refresh("expired-refresh-token")
	if !errors.Is(err, domain.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got: %v", err)
	}
}

// TestRefresh_InvalidToken tests that an invalid token is rejected.
func TestRefresh_InvalidToken(t *testing.T) {
	repo := newMockUserRepo()

	gen := &mockTokenGenerator{
		validateErr: domain.ErrTokenInvalid,
	}
	bcryptResult := &mockBcryptCompare{match: true}

	service := createAuthService(repo, gen, bcryptResult)

	_, err := service.Refresh("tampered-token")
	if !errors.Is(err, domain.ErrTokenInvalid) {
		t.Errorf("expected ErrTokenInvalid, got: %v", err)
	}
}

// TestRefresh_DeletedUser tests that refreshing a token for a deleted user
// is rejected with ErrUnauthorized.
func TestRefresh_DeletedUser(t *testing.T) {
	repo := newMockUserRepo() // empty — user doesn't exist

	gen := &mockTokenGenerator{
		validateClaims: &domain.Claims{
			Subject: "deleted-user-id",
			Type:    "refresh",
		},
	}
	bcryptResult := &mockBcryptCompare{match: true}

	service := createAuthService(repo, gen, bcryptResult)

	_, err := service.Refresh("valid-refresh-token-for-deleted-user")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for deleted user, got: %v", err)
	}
}