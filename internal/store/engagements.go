package store

import (
	"context"
	"errors"
	"strings"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateEngagement(ctx context.Context, e *models.Engagement) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO engagements(org_id, client_name, authz_ref, starts_at, ends_at, status)
		 VALUES($1,$2,$3,$4,$5,$6) RETURNING id, created_at`,
		e.OrgID, e.ClientName, e.AuthzRef, e.StartsAt, e.EndsAt, string(e.Status),
	).Scan(&e.ID, &e.CreatedAt)
}

func (s *Store) ListEngagements(ctx context.Context, orgID uuid.UUID) ([]models.Engagement, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, org_id, client_name, authz_ref, starts_at, ends_at, status, created_at
		 FROM engagements WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Engagement{}
	for rows.Next() {
		var e models.Engagement
		if err := rows.Scan(&e.ID, &e.OrgID, &e.ClientName, &e.AuthzRef, &e.StartsAt, &e.EndsAt, &e.Status, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) GetEngagement(ctx context.Context, orgID, id uuid.UUID) (*models.Engagement, error) {
	var e models.Engagement
	err := s.pool.QueryRow(ctx,
		`SELECT id, org_id, client_name, authz_ref, starts_at, ends_at, status, created_at
		 FROM engagements WHERE org_id=$1 AND id=$2`, orgID, id,
	).Scan(&e.ID, &e.OrgID, &e.ClientName, &e.AuthzRef, &e.StartsAt, &e.EndsAt, &e.Status, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// GetEngagementByID fetches an engagement without tenant scoping. Used by the
// target-facing phishing server, which authenticates via the signed RID rather
// than an org-scoped principal (there is no logged-in user on that side).
func (s *Store) GetEngagementByID(ctx context.Context, id uuid.UUID) (*models.Engagement, error) {
	var e models.Engagement
	err := s.pool.QueryRow(ctx,
		`SELECT id, org_id, client_name, authz_ref, starts_at, ends_at, status, created_at
		 FROM engagements WHERE id=$1`, id,
	).Scan(&e.ID, &e.OrgID, &e.ClientName, &e.AuthzRef, &e.StartsAt, &e.EndsAt, &e.Status, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) SetEngagementStatus(ctx context.Context, orgID, id uuid.UUID, status models.EngagementStatus) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE engagements SET status=$1 WHERE org_id=$2 AND id=$3`, string(status), orgID, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateEngagement(ctx context.Context, e *models.Engagement) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE engagements SET client_name=$1,authz_ref=$2,starts_at=$3,ends_at=$4 WHERE org_id=$5 AND id=$6`,
		e.ClientName, e.AuthzRef, e.StartsAt, e.EndsAt, e.OrgID, e.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteEngagement(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM engagements WHERE org_id=$1 AND id=$2`, orgID, id)
	return err
}

func (s *Store) AddScopeRule(ctx context.Context, r *models.ScopeRule) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO scope_rules(engagement_id, kind, pattern) VALUES($1,$2,$3)
		 RETURNING id, created_at`,
		r.EngagementID, r.Kind, strings.ToLower(strings.TrimSpace(r.Pattern)),
	).Scan(&r.ID, &r.CreatedAt)
}

func (s *Store) ListScopeRules(ctx context.Context, engagementID uuid.UUID) ([]models.ScopeRule, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, engagement_id, kind, pattern, created_at FROM scope_rules
		 WHERE engagement_id=$1 ORDER BY created_at`, engagementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ScopeRule{}
	for rows.Next() {
		var r models.ScopeRule
		if err := rows.Scan(&r.ID, &r.EngagementID, &r.Kind, &r.Pattern, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DeleteScopeRule(ctx context.Context, engagementID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM scope_rules WHERE engagement_id=$1 AND id=$2`, engagementID, id)
	return err
}
