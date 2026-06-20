package application

import (
	"context"
	"fmt"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

type TokenGenerator interface {
	Generate(claims domain.Claims) (string, error)
	Validate(tokenString string) (*domain.Claims, error)
}

type AuthService struct {
	userRepo         domain.UserRepository
	tokenGen         TokenGenerator
	bcryptCompare    func(hashedPassword, password []byte) error
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

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

type LoginResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Login authenticates a user with email and password.
// Returns ErrUnauthorized for both wrong email and wrong password to prevent enumeration.
func (s *AuthService) Login(email, password string) (*LoginResult, error) {
	user, err := s.userRepo.FindByEmail(context.Background(), email)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if err := s.bcryptCompare([]byte(user.Password), []byte(password)); err != nil {
		return nil, domain.ErrUnauthorized
	}

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

type RefreshResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Refresh validates a refresh token and issues new access + refresh tokens.
// Token rotation: a new refresh token is issued on each refresh.
func (s *AuthService) Refresh(refreshToken string) (*RefreshResult, error) {
	claims, err := s.tokenGen.Validate(refreshToken)
	if err != nil {
		return nil, err
	}

	if claims.Type != "refresh" {
		return nil, fmt.Errorf("%w: expected refresh token, got %s token", domain.ErrTokenInvalid, claims.Type)
	}

	// Verify the user still exists — prevents token use after account deletion.
	user, err := s.userRepo.FindByID(context.Background(), claims.Subject)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

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