package sqlite

import (
	"context"
	"testing"

	"github.com/francisco/fleet-monitor/internal/domain"
)

func TestUserRepo_CreateAndFindByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	user := &domain.User{
		Email:    "test@example.com",
		Password: "hashedpassword123",
		Role:     "admin",
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if user.ID == "" {
		t.Fatal("expected ID to be set after Create, got empty string")
	}
	if user.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set after Create, got zero value")
	}

	found, err := repo.FindByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}

	if found.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, found.Email)
	}
	if found.Password != user.Password {
		t.Errorf("expected password %s, got %s", user.Password, found.Password)
	}
	if found.Role != user.Role {
		t.Errorf("expected role %s, got %s", user.Role, found.Role)
	}
	if found.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, found.ID)
	}
}

func TestUserRepo_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	user := &domain.User{
		Email:    "findbyid@example.com",
		Password: "hashedpass",
		Role:     "user",
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if found.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, found.ID)
	}
	if found.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, found.Email)
	}
}

func TestUserRepo_DuplicateEmail_ErrConflict(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	user1 := &domain.User{
		Email:    "duplicate@example.com",
		Password: "pass1",
		Role:     "user",
	}
	user2 := &domain.User{
		Email:    "duplicate@example.com",
		Password: "pass2",
		Role:     "admin",
	}

	if err := repo.Create(ctx, user1); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	err := repo.Create(ctx, user2)
	if err == nil {
		t.Fatal("expected ErrConflict for duplicate email, got nil")
	}
	if err != domain.ErrConflict {
		t.Errorf("expected domain.ErrConflict, got %v", err)
	}
}

func TestUserRepo_FindByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	_, err := repo.FindByEmail(ctx, "nonexistent@example.com")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}

func TestUserRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected ErrNotFound, got nil")
	}
	if err != domain.ErrNotFound {
		t.Errorf("expected domain.ErrNotFound, got %v", err)
	}
}