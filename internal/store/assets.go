package store

import (
	"context"
	"errors"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ---- Sending profiles ----

const sendingProfileCols = `id,org_id,name,smtp_host,smtp_port,username,from_address,from_name,use_tls,dkim_domain,dkim_selector,sign_dkim,x_mailer,landing_base_url,created_at`

func scanSendingProfile(row interface{ Scan(...any) error }, p *models.SendingProfile) error {
	return row.Scan(&p.ID, &p.OrgID, &p.Name, &p.SMTPHost, &p.SMTPPort, &p.Username, &p.FromAddress, &p.FromName,
		&p.UseTLS, &p.DKIMDomain, &p.DKIMSelector, &p.SignDKIM, &p.XMailer, &p.LandingBaseURL, &p.CreatedAt)
}

func (s *Store) CreateSendingProfile(ctx context.Context, p *models.SendingProfile) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO sending_profiles(org_id,name,smtp_host,smtp_port,username,password,from_address,from_name,use_tls,dkim_domain,dkim_selector,dkim_private_key,sign_dkim,x_mailer,landing_base_url)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id, created_at`,
		p.OrgID, p.Name, p.SMTPHost, p.SMTPPort, p.Username, p.Password, p.FromAddress, p.FromName, p.UseTLS,
		p.DKIMDomain, p.DKIMSelector, p.DKIMPrivateKey, p.SignDKIM, p.XMailer, p.LandingBaseURL,
	).Scan(&p.ID, &p.CreatedAt)
}

func (s *Store) ListSendingProfiles(ctx context.Context, orgID uuid.UUID) ([]models.SendingProfile, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+sendingProfileCols+` FROM sending_profiles WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.SendingProfile{}
	for rows.Next() {
		var p models.SendingProfile
		if err := scanSendingProfile(rows, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetSendingProfileFull includes secrets (password + DKIM key); worker/internal use only.
func (s *Store) GetSendingProfileFull(ctx context.Context, orgID, id uuid.UUID) (*models.SendingProfile, error) {
	var p models.SendingProfile
	err := s.pool.QueryRow(ctx,
		`SELECT id,org_id,name,smtp_host,smtp_port,username,password,from_address,from_name,use_tls,dkim_domain,dkim_selector,dkim_private_key,sign_dkim,x_mailer,landing_base_url,created_at
		 FROM sending_profiles WHERE org_id=$1 AND id=$2`, orgID, id,
	).Scan(&p.ID, &p.OrgID, &p.Name, &p.SMTPHost, &p.SMTPPort, &p.Username, &p.Password, &p.FromAddress, &p.FromName, &p.UseTLS,
		&p.DKIMDomain, &p.DKIMSelector, &p.DKIMPrivateKey, &p.SignDKIM, &p.XMailer, &p.LandingBaseURL, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) UpdateSendingProfile(ctx context.Context, orgID uuid.UUID, p *models.SendingProfile) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE sending_profiles SET name=$1,smtp_host=$2,smtp_port=$3,username=$4,password=$5,from_address=$6,from_name=$7,use_tls=$8,dkim_domain=$9,dkim_selector=$10,dkim_private_key=$11,sign_dkim=$12,x_mailer=$13,landing_base_url=$14
		 WHERE org_id=$15 AND id=$16`,
		p.Name, p.SMTPHost, p.SMTPPort, p.Username, p.Password, p.FromAddress, p.FromName, p.UseTLS,
		p.DKIMDomain, p.DKIMSelector, p.DKIMPrivateKey, p.SignDKIM, p.XMailer, p.LandingBaseURL, orgID, p.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteSendingProfile(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sending_profiles WHERE org_id=$1 AND id=$2`, orgID, id)
	return err
}

// ---- Targets ----

func (s *Store) CreateTarget(ctx context.Context, t *models.Target) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO targets(engagement_id,email,first_name,last_name,position,department,is_vip,timezone)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (engagement_id,email) DO UPDATE SET first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name,
		   position=EXCLUDED.position, department=EXCLUDED.department, is_vip=EXCLUDED.is_vip
		 RETURNING id, created_at`,
		t.EngagementID, t.Email, t.FirstName, t.LastName, t.Position, t.Department, t.IsVIP, t.Timezone,
	).Scan(&t.ID, &t.CreatedAt)
}

func (s *Store) ListTargets(ctx context.Context, engagementID uuid.UUID) ([]models.Target, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,engagement_id,email,first_name,last_name,position,department,is_vip,timezone,created_at
		 FROM targets WHERE engagement_id=$1 ORDER BY created_at`, engagementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Target{}
	for rows.Next() {
		var t models.Target
		if err := rows.Scan(&t.ID, &t.EngagementID, &t.Email, &t.FirstName, &t.LastName, &t.Position, &t.Department, &t.IsVIP, &t.Timezone, &t.CreatedAt); err != nil {
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

func (s *Store) UpdateEmailTemplate(ctx context.Context, orgID uuid.UUID, t *models.EmailTemplate) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE email_templates SET name=$1,subject=$2,html=$3,text=$4,version=version+1 WHERE org_id=$5 AND id=$6`,
		t.Name, t.Subject, t.HTML, t.Text, orgID, t.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteEmailTemplate(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM email_templates WHERE org_id=$1 AND id=$2`, orgID, id)
	return err
}

// ---- Landing pages ----

const landingCols = `id,org_id,name,html,capture_meta,capture_submitted_data,capture_passwords,redirect_url,created_at`

func scanLanding(row interface{ Scan(...any) error }, l *models.LandingPage) error {
	return row.Scan(&l.ID, &l.OrgID, &l.Name, &l.HTML, &l.CaptureMeta,
		&l.CaptureSubmittedData, &l.CapturePasswords, &l.RedirectURL, &l.CreatedAt)
}

func (s *Store) CreateLandingPage(ctx context.Context, l *models.LandingPage) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO landing_pages(org_id,name,html,capture_meta,capture_submitted_data,capture_passwords,redirect_url)
		 VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at`,
		l.OrgID, l.Name, l.HTML, l.CaptureMeta, l.CaptureSubmittedData, l.CapturePasswords, l.RedirectURL,
	).Scan(&l.ID, &l.CreatedAt)
}

func (s *Store) ListLandingPages(ctx context.Context, orgID uuid.UUID) ([]models.LandingPage, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+landingCols+` FROM landing_pages WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.LandingPage{}
	for rows.Next() {
		var l models.LandingPage
		if err := scanLanding(rows, &l); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) GetLandingPage(ctx context.Context, orgID, id uuid.UUID) (*models.LandingPage, error) {
	var l models.LandingPage
	err := scanLanding(s.pool.QueryRow(ctx,
		`SELECT `+landingCols+` FROM landing_pages WHERE org_id=$1 AND id=$2`, orgID, id), &l)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (s *Store) UpdateLandingPage(ctx context.Context, orgID uuid.UUID, l *models.LandingPage) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE landing_pages SET name=$1,html=$2,capture_meta=$3,capture_submitted_data=$4,capture_passwords=$5,redirect_url=$6 WHERE org_id=$7 AND id=$8`,
		l.Name, l.HTML, l.CaptureMeta, l.CaptureSubmittedData, l.CapturePasswords, l.RedirectURL, orgID, l.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteLandingPage(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM landing_pages WHERE org_id=$1 AND id=$2`, orgID, id)
	return err
}
