package store

import (
	"context"
	"errors"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ---- A/B variants ----

func (s *Store) CreateVariant(ctx context.Context, v *models.CampaignVariant) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO campaign_variants(campaign_id,name,email_template_id,weight)
		 VALUES($1,$2,$3,$4) RETURNING id, created_at`,
		v.CampaignID, v.Name, v.EmailTemplateID, v.Weight,
	).Scan(&v.ID, &v.CreatedAt)
}

func (s *Store) SetCampaignTargetVariant(ctx context.Context, ctID, variantID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE campaign_targets SET variant_id=$1 WHERE id=$2`, variantID, ctID)
	return err
}

func (s *Store) ListVariants(ctx context.Context, campaignID uuid.UUID) ([]models.CampaignVariant, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,campaign_id,name,email_template_id,weight,created_at
		 FROM campaign_variants WHERE campaign_id=$1 ORDER BY created_at`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.CampaignVariant{}
	for rows.Next() {
		var v models.CampaignVariant
		if err := rows.Scan(&v.ID, &v.CampaignID, &v.Name, &v.EmailTemplateID, &v.Weight, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// TemplateIDForCampaignTarget returns the email template a campaign_target should
// use: its variant's template if assigned, else the campaign's default template.
func (s *Store) TemplateIDForCampaignTarget(ctx context.Context, ct models.CampaignTarget, campaignDefault uuid.UUID) uuid.UUID {
	if ct.VariantID == nil {
		return campaignDefault
	}
	var tplID uuid.UUID
	if err := s.pool.QueryRow(ctx,
		`SELECT email_template_id FROM campaign_variants WHERE id=$1`, *ct.VariantID,
	).Scan(&tplID); err != nil {
		return campaignDefault
	}
	return tplID
}

// VariantFunnel returns per-variant distinct-target event counts for a campaign.
func (s *Store) VariantFunnel(ctx context.Context, campaignID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT COALESCE(cv.name,'(default)') AS variant,
		        count(DISTINCT ct.id) AS targets,
		        count(DISTINCT ct.id) FILTER (WHERE ev.type='open')   AS opened,
		        count(DISTINCT ct.id) FILTER (WHERE ev.type='click')  AS clicked,
		        count(DISTINCT ct.id) FILTER (WHERE ev.type='submit') AS submitted
		 FROM campaign_targets ct
		 LEFT JOIN campaign_variants cv ON cv.id=ct.variant_id
		 LEFT JOIN events ev ON ev.campaign_target_id=ct.id
		 WHERE ct.campaign_id=$1
		 GROUP BY variant ORDER BY variant`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var name string
		var targets, opened, clicked, submitted int
		if err := rows.Scan(&name, &targets, &opened, &clicked, &submitted); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"variant": name, "targets": targets, "opened": opened,
			"clicked": clicked, "submitted": submitted,
		})
	}
	return out, rows.Err()
}

// ---- Training modules ----

func (s *Store) CreateTrainingModule(ctx context.Context, m *models.TrainingModule) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO training_modules(org_id,name,html) VALUES($1,$2,$3) RETURNING id, created_at`,
		m.OrgID, m.Name, m.HTML,
	).Scan(&m.ID, &m.CreatedAt)
}

func (s *Store) ListTrainingModules(ctx context.Context, orgID uuid.UUID) ([]models.TrainingModule, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,org_id,name,html,created_at FROM training_modules WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.TrainingModule{}
	for rows.Next() {
		var m models.TrainingModule
		if err := rows.Scan(&m.ID, &m.OrgID, &m.Name, &m.HTML, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetTrainingModule(ctx context.Context, orgID, id uuid.UUID) (*models.TrainingModule, error) {
	var m models.TrainingModule
	err := s.pool.QueryRow(ctx,
		`SELECT id,org_id,name,html,created_at FROM training_modules WHERE org_id=$1 AND id=$2`, orgID, id,
	).Scan(&m.ID, &m.OrgID, &m.Name, &m.HTML, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &m, err
}

func (s *Store) UpdateTrainingModule(ctx context.Context, orgID uuid.UUID, m *models.TrainingModule) error {
	ct, err := s.pool.Exec(ctx,
		`UPDATE training_modules SET name=$1,html=$2 WHERE org_id=$3 AND id=$4`, m.Name, m.HTML, orgID, m.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteTrainingModule(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM training_modules WHERE org_id=$1 AND id=$2`, orgID, id)
	return err
}

// AssignTraining creates (idempotently) a training assignment for a target and
// returns the assignment token used in the completion link.
func (s *Store) AssignTraining(ctx context.Context, targetID, moduleID uuid.UUID, campaignID *uuid.UUID, token string) (string, error) {
	var existing string
	err := s.pool.QueryRow(ctx,
		`SELECT token FROM training_assignments WHERE target_id=$1 AND module_id=$2 AND campaign_id IS NOT DISTINCT FROM $3`,
		targetID, moduleID, campaignID,
	).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO training_assignments(target_id,module_id,campaign_id,token) VALUES($1,$2,$3,$4)`,
		targetID, moduleID, campaignID, token)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) TrainingModuleByToken(ctx context.Context, token string) (*models.TrainingModule, error) {
	var m models.TrainingModule
	err := s.pool.QueryRow(ctx,
		`SELECT tm.id,tm.org_id,tm.name,tm.html,tm.created_at
		 FROM training_assignments ta JOIN training_modules tm ON tm.id=ta.module_id
		 WHERE ta.token=$1`, token,
	).Scan(&m.ID, &m.OrgID, &m.Name, &m.HTML, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &m, err
}

func (s *Store) CompleteTraining(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE training_assignments SET status='completed', completed_at=now()
		 WHERE token=$1 AND status<>'completed'`, token)
	return err
}

func (s *Store) ListTrainingAssignments(ctx context.Context, orgID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT t.email, tm.name, ta.status, ta.assigned_at, ta.completed_at
		 FROM training_assignments ta
		 JOIN targets t ON t.id=ta.target_id
		 JOIN training_modules tm ON tm.id=ta.module_id
		 WHERE tm.org_id=$1 ORDER BY ta.assigned_at DESC LIMIT 500`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var email, module, status string
		var assigned time.Time
		var completed *time.Time
		if err := rows.Scan(&email, &module, &status, &assigned, &completed); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"email": email, "module": module, "status": status,
			"assigned_at": assigned, "completed_at": completed,
		})
	}
	return out, rows.Err()
}

// ---- API keys ----

func (s *Store) CreateAPIKey(ctx context.Context, k *models.APIKey, keyHash string) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO api_keys(org_id,name,prefix,key_hash,role) VALUES($1,$2,$3,$4,$5)
		 RETURNING id, created_at`,
		k.OrgID, k.Name, k.Prefix, keyHash, string(k.Role),
	).Scan(&k.ID, &k.CreatedAt)
}

func (s *Store) ListAPIKeys(ctx context.Context, orgID uuid.UUID) ([]models.APIKey, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,org_id,name,prefix,role,created_at,last_used_at,revoked
		 FROM api_keys WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.APIKey{}
	for rows.Next() {
		var k models.APIKey
		if err := rows.Scan(&k.ID, &k.OrgID, &k.Name, &k.Prefix, &k.Role, &k.CreatedAt, &k.LastUsedAt, &k.Revoked); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) RevokeAPIKey(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE api_keys SET revoked=TRUE WHERE org_id=$1 AND id=$2`, orgID, id)
	return err
}

// APIKeyByPrefix returns the stored hash + identity for a key prefix (for auth).
func (s *Store) APIKeyByPrefix(ctx context.Context, prefix string) (orgID uuid.UUID, role string, hash string, err error) {
	err = s.pool.QueryRow(ctx,
		`SELECT org_id, role, key_hash FROM api_keys WHERE prefix=$1 AND revoked=FALSE`, prefix,
	).Scan(&orgID, &role, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return
}

func (s *Store) TouchAPIKey(ctx context.Context, prefix string) {
	_, _ = s.pool.Exec(ctx, `UPDATE api_keys SET last_used_at=now() WHERE prefix=$1`, prefix)
}

// ---- Webhooks ----

func (s *Store) CreateWebhook(ctx context.Context, w *models.Webhook) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO webhooks(org_id,url,secret,events) VALUES($1,$2,$3,$4) RETURNING id, created_at`,
		w.OrgID, w.URL, w.Secret, w.Events,
	).Scan(&w.ID, &w.CreatedAt)
}

func (s *Store) ListWebhooks(ctx context.Context, orgID uuid.UUID) ([]models.Webhook, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,org_id,url,secret,events,created_at FROM webhooks WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Webhook{}
	for rows.Next() {
		var w models.Webhook
		if err := rows.Scan(&w.ID, &w.OrgID, &w.URL, &w.Secret, &w.Events, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) DeleteWebhook(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM webhooks WHERE org_id=$1 AND id=$2`, orgID, id)
	return err
}

// WebhooksForEvent returns webhooks subscribed to a given event type (or all).
func (s *Store) WebhooksForEvent(ctx context.Context, orgID uuid.UUID, event string) ([]models.Webhook, error) {
	all, err := s.ListWebhooks(ctx, orgID)
	if err != nil {
		return nil, err
	}
	var out []models.Webhook
	for _, w := range all {
		if len(w.Events) == 0 {
			out = append(out, w)
			continue
		}
		for _, e := range w.Events {
			if e == event || e == "*" {
				out = append(out, w)
				break
			}
		}
	}
	return out, nil
}

// CampaignIDForCampaignTarget resolves the campaign a campaign_target belongs
// to — used to correlate an inbound ESP webhook event (keyed by
// campaign_target_id) back to its campaign for reputation-safety checks.
func (s *Store) CampaignIDForCampaignTarget(ctx context.Context, campaignTargetID uuid.UUID) (uuid.UUID, error) {
	var campaignID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT campaign_id FROM campaign_targets WHERE id=$1`, campaignTargetID,
	).Scan(&campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return campaignID, err
}

// BounceComplaintStats returns the sent/bounced/complained counts for a
// campaign, used to decide whether reputation is at risk and the campaign
// should be auto-paused.
func (s *Store) BounceComplaintStats(ctx context.Context, campaignID uuid.UUID) (sent, bounced, complained int, err error) {
	err = s.pool.QueryRow(ctx,
		`SELECT
		   count(DISTINCT ct.id) FILTER (WHERE ev.type='sent'),
		   count(DISTINCT ct.id) FILTER (WHERE ev.type='bounced'),
		   count(DISTINCT ct.id) FILTER (WHERE ev.type='complained')
		 FROM campaign_targets ct
		 LEFT JOIN events ev ON ev.campaign_target_id=ct.id
		 WHERE ct.campaign_id=$1`, campaignID,
	).Scan(&sent, &bounced, &complained)
	return
}

// OrgIDForCampaign resolves the owning org of a campaign (used by the phishing server).
func (s *Store) OrgIDForCampaign(ctx context.Context, campaignID uuid.UUID) (uuid.UUID, error) {
	var orgID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT e.org_id FROM campaigns c JOIN engagements e ON e.id=c.engagement_id WHERE c.id=$1`,
		campaignID,
	).Scan(&orgID)
	return orgID, err
}

// ---- Risk scoring ----

// RiskScores aggregates per-target behavior across an engagement. A simple model:
// score = clicks*3 + submits*5 - reports*2, clamped at 0. Higher = riskier.
func (s *Store) RiskScores(ctx context.Context, engagementID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT t.email, t.position, t.department, t.is_vip,
		        count(*) FILTER (WHERE ev.type='open')   AS opens,
		        count(*) FILTER (WHERE ev.type='click')  AS clicks,
		        count(*) FILTER (WHERE ev.type='submit') AS submits,
		        count(*) FILTER (WHERE ev.type='report') AS reports
		 FROM targets t
		 LEFT JOIN campaign_targets ct ON ct.target_id=t.id
		 LEFT JOIN events ev ON ev.campaign_target_id=ct.id
		 WHERE t.engagement_id=$1
		 GROUP BY t.email, t.position, t.department, t.is_vip
		 ORDER BY (count(*) FILTER (WHERE ev.type='click'))*3
		        + (count(*) FILTER (WHERE ev.type='submit'))*5 DESC, t.email`, engagementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var email, position, department string
		var isVIP bool
		var opens, clicks, submits, reports int
		if err := rows.Scan(&email, &position, &department, &isVIP, &opens, &clicks, &submits, &reports); err != nil {
			return nil, err
		}
		score := clicks*3 + submits*5 - reports*2
		if score < 0 {
			score = 0
		}
		level := "low"
		if score >= 8 {
			level = "high"
		} else if score >= 3 {
			level = "medium"
		}
		out = append(out, map[string]any{
			"email": email, "position": position, "department": department, "is_vip": isVIP,
			"opens": opens, "clicks": clicks, "submits": submits, "reports": reports,
			"score": score, "level": level,
		})
	}
	return out, rows.Err()
}

// OrgFunnel aggregates the funnel across all campaigns in an org (dashboard).
func (s *Store) OrgFunnel(ctx context.Context, orgID uuid.UUID) (map[string]int, error) {
	var targets, sent, open, click, submit, report int
	err := s.pool.QueryRow(ctx,
		`SELECT
		   count(DISTINCT ct.id),
		   count(DISTINCT ct.id) FILTER (WHERE ev.type='sent'),
		   count(DISTINCT ct.id) FILTER (WHERE ev.type='open'),
		   count(DISTINCT ct.id) FILTER (WHERE ev.type='click'),
		   count(DISTINCT ct.id) FILTER (WHERE ev.type='submit'),
		   count(DISTINCT ct.id) FILTER (WHERE ev.type='report')
		 FROM campaign_targets ct
		 JOIN campaigns c ON c.id=ct.campaign_id
		 JOIN engagements e ON e.id=c.engagement_id
		 LEFT JOIN events ev ON ev.campaign_target_id=ct.id
		 WHERE e.org_id=$1`, orgID,
	).Scan(&targets, &sent, &open, &click, &submit, &report)
	return map[string]int{
		"targets": targets, "sent": sent, "open": open,
		"click": click, "submit": submit, "report": report,
	}, err
}
