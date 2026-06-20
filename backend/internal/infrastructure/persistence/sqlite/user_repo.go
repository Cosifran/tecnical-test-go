package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/francisco/fleet-monitor/internal/domain"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	const query = "SELECT id, email, password, role, created_at FROM users WHERE id = ?"

	var u domain.User
	var createdAt string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	u.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = "SELECT id, email, password, role, created_at FROM users WHERE email = ?"

	var u domain.User
	var createdAt string
	err := r.db.QueryRowContext(ctx, query, email).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	u.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	id := generateUUID()
	now := time.Now()

	const query = "INSERT INTO users (id, email, password, role, created_at) VALUES (?, ?, ?, ?, ?)"
	_, err := r.db.ExecContext(ctx, query, id, user.Email, user.Password, user.Role, formatTime(now))
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrConflict
		}
		return err
	}

	user.ID = id
	user.CreatedAt = now
	return nil
}