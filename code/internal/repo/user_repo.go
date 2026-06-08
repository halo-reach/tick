package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tickplatform/tick/internal/domain"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, username, password_hash, display_name, email, status, failed_attempts, locked_until, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		u.ID, u.Username, u.PasswordHash, u.DisplayName, u.Email, u.Status, u.FailedAttempts, u.LockedUntil, u.CreatedAt)
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	u := &domain.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, display_name, email, status, failed_attempts, locked_until, created_at
		 FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Email, &u.Status, &u.FailedAttempts, &u.LockedUntil, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	u := &domain.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, display_name, email, status, failed_attempts, locked_until, created_at
		 FROM users WHERE username = $1`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Email, &u.Status, &u.FailedAttempts, &u.LockedUntil, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) UpdateFailedAttempts(ctx context.Context, id string, attempts int, lockedUntil *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET failed_attempts = $1, locked_until = $2 WHERE id = $3`,
		attempts, lockedUntil, id)
	return err
}

func (r *UserRepo) ResetFailedAttempts(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE id = $1`, id)
	return err
}

type UserSearchResult struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

func (r *UserRepo) SearchExcludingTenant(ctx context.Context, query string, tenantID string, limit int) ([]*UserSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, username, display_name FROM users
		 WHERE username ILIKE '%' || $1 || '%' AND status = 'active'
		 AND id NOT IN (SELECT user_id FROM tenant_members WHERE tenant_id = $2)
		 ORDER BY username LIMIT $3`, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*UserSearchResult
	for rows.Next() {
		u := &UserSearchResult{}
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName); err != nil {
			return nil, err
		}
		results = append(results, u)
	}
	return results, nil
}
