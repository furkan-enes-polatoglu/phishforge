package store

import (
	"context"
	"errors"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("not found")

func (s *Store) CreateOrg(ctx context.Context, name string) (*models.Organization, error) {
	var o models.Organization
	err := s.pool.QueryRow(ctx,
		`INSERT INTO organizations(name) VALUES($1) RETURNING id, name, created_at`, name,
	).Scan(&o.ID, &o.Name, &o.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) CreateUser(ctx context.Context, orgID uuid.UUID, username, passwordHash string, role models.Role) (*models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users(org_id, username, password_hash, role)
		 VALUES($1,$2,$3,$4) RETURNING id, org_id, username, password_hash, role, created_at`,
		orgID, username, passwordHash, string(role),
	).Scan(&u.ID, &u.OrgID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UserByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, org_id, username, password_hash, role, created_at FROM users WHERE username=$1`, username,
	).Scan(&u.ID, &u.OrgID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var u models.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, org_id, username, password_hash, role, created_at FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.OrgID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) ListUsers(ctx context.Context, orgID uuid.UUID) ([]models.User, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, org_id, username, password_hash, role, created_at FROM users WHERE org_id=$1 ORDER BY created_at`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.User{}
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.OrgID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
