package store

import (
	"context"
	"errors"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ---- Sending profiles ----

func (s *Store) CreateSendingProfile(ctx context.Context, p *models.SendingProfile) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO sending_profiles(org_id,name,smtp_host,smtp_port,username,password,from_address,from_name,use_tls)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at`,
		p.OrgID, p.Name, p.SMTPHost, p.SMTPPort, p.Username, p.Password, p.FromAddress, p.FromName, p.UseTLS,
	).Scan(&p.ID, &p.CreatedAt)
}

func (s *Store) ListSendingProfiles(ctx context.Context, orgID uuid.UUID) ([]models.SendingProfile, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,org_id,name,smtp_host,smtp_port,username,from_address,from_name,use_tls,created_at
		 FROM sending_profiles WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.SendingProfile{}
	for rows.Next() {
		var p models.SendingProfile
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.SMTPHost, &p.SMTPPort, &p.Username, &p.FromAddress, &p.FromName, &p.UseTLS, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetSendingProfileFull includes the password (used by the worker only).
func (s *Store) GetSendingProfileFull(ctx context.Context, orgID, id uuid.UUID) (*models.SendingProfile, error) {
	var p models.SendingProfile
	err := s.pool.QueryRow(ctx,
		`SELECT id,org_id,name,smtp_host,smtp_port,username,password,from_address,from_name,use_tls,created_at
		 FROM sending_profiles WHERE org_id=$1 AND id=$2`, orgID, id,
	).Scan(&p.ID, &p.OrgID, &p.Name, &p.SMTPHost, &p.SMTPPort, &p.Username, &p.Password, &p.FromAddress, &p.FromName, &p.UseTLS, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ---- Targets ----

func (s *Store) CreateTarget(ctx context.Context, t *models.Target) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO targets(engagement_id,email,first_name,last_name,position,timezone)
		 VALUES($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (engagement_id,email) DO UPDATE SET first_name=EXCLUDED.first_name
		 RETURNING id, created_at`,
		t.EngagementID, t.Email, t.FirstName, t.LastName, t.Position, t.Timezone,
	).Scan(&t.ID, &t.CreatedAt)
}

func (s *Store) ListTargets(ctx context.Context, engagementID uuid.UUID) ([]models.Target, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,engagement_id,email,first_name,last_name,position,timezone,created_at
		 FROM targets WHERE engagement_id=$1 ORDER BY created_at`, engagementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Target{}
	for rows.Next() {
		var t models.Target
		if err := rows.Scan(&t.ID, &t.EngagementID, &t.Email, &t.FirstName, &t.LastName, &t.Position, &t.Timezone, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ---- Email templates ----

func (s *Store) CreateEmailTemplate(ctx context.Context, t *models.EmailTemplate) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO email_templates(org_id,name,subject,html,text,version)
		 VALUES($1,$2,$3,$4,$5,$6) RETURNING id, created_at`,
		t.OrgID, t.Name, t.Subject, t.HTML, t.Text, t.Version,
	).Scan(&t.ID, &t.CreatedAt)
}

func (s *Store) ListEmailTemplates(ctx context.Context, orgID uuid.UUID) ([]models.EmailTemplate, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,org_id,name,subject,html,text,version,created_at
		 FROM email_templates WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.EmailTemplate{}
	for rows.Next() {
		var t models.EmailTemplate
		if err := rows.Scan(&t.ID, &t.OrgID, &t.Name, &t.Subject, &t.HTML, &t.Text, &t.Version, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetEmailTemplate(ctx context.Context, orgID, id uuid.UUID) (*models.EmailTemplate, error) {
	var t models.EmailTemplate
	err := s.pool.QueryRow(ctx,
		`SELECT id,org_id,name,subject,html,text,version,created_at
		 FROM email_templates WHERE org_id=$1 AND id=$2`, orgID, id,
	).Scan(&t.ID, &t.OrgID, &t.Name, &t.Subject, &t.HTML, &t.Text, &t.Version, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ---- Landing pages ----

func (s *Store) CreateLandingPage(ctx context.Context, l *models.LandingPage) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO landing_pages(org_id,name,html,capture_meta,redirect_url)
		 VALUES($1,$2,$3,$4,$5) RETURNING id, created_at`,
		l.OrgID, l.Name, l.HTML, l.CaptureMeta, l.RedirectURL,
	).Scan(&l.ID, &l.CreatedAt)
}

func (s *Store) ListLandingPages(ctx context.Context, orgID uuid.UUID) ([]models.LandingPage, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,org_id,name,html,capture_meta,redirect_url,created_at
		 FROM landing_pages WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.LandingPage{}
	for rows.Next() {
		var l models.LandingPage
		if err := rows.Scan(&l.ID, &l.OrgID, &l.Name, &l.HTML, &l.CaptureMeta, &l.RedirectURL, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) GetLandingPage(ctx context.Context, orgID, id uuid.UUID) (*models.LandingPage, error) {
	var l models.LandingPage
	err := s.pool.QueryRow(ctx,
		`SELECT id,org_id,name,html,capture_meta,redirect_url,created_at
		 FROM landing_pages WHERE org_id=$1 AND id=$2`, orgID, id,
	).Scan(&l.ID, &l.OrgID, &l.Name, &l.HTML, &l.CaptureMeta, &l.RedirectURL, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}
