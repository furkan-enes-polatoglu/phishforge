package store

import (
	"context"
	"errors"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateCampaign(ctx context.Context, c *models.Campaign) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO campaigns(engagement_id,name,email_template_id,landing_page_id,sending_profile_id,status,launch_at,rate_per_minute)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at`,
		c.EngagementID, c.Name, c.EmailTemplateID, c.LandingPageID, c.SendingProfileID, string(c.Status), c.LaunchAt, c.RatePerMinute,
	).Scan(&c.ID, &c.CreatedAt)
}

func (s *Store) ListCampaigns(ctx context.Context, engagementID uuid.UUID) ([]models.Campaign, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,engagement_id,name,email_template_id,landing_page_id,sending_profile_id,status,launch_at,rate_per_minute,created_at
		 FROM campaigns WHERE engagement_id=$1 ORDER BY created_at DESC`, engagementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Campaign{}
	for rows.Next() {
		var c models.Campaign
		if err := rows.Scan(&c.ID, &c.EngagementID, &c.Name, &c.EmailTemplateID, &c.LandingPageID, &c.SendingProfileID, &c.Status, &c.LaunchAt, &c.RatePerMinute, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCampaign fetches a campaign scoped to an org (via its engagement).
func (s *Store) GetCampaign(ctx context.Context, orgID, id uuid.UUID) (*models.Campaign, error) {
	var c models.Campaign
	err := s.pool.QueryRow(ctx,
		`SELECT c.id,c.engagement_id,c.name,c.email_template_id,c.landing_page_id,c.sending_profile_id,c.status,c.launch_at,c.rate_per_minute,c.created_at
		 FROM campaigns c JOIN engagements e ON e.id=c.engagement_id
		 WHERE e.org_id=$1 AND c.id=$2`, orgID, id,
	).Scan(&c.ID, &c.EngagementID, &c.Name, &c.EmailTemplateID, &c.LandingPageID, &c.SendingProfileID, &c.Status, &c.LaunchAt, &c.RatePerMinute, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) SetCampaignStatus(ctx context.Context, id uuid.UUID, status models.CampaignStatus) error {
	_, err := s.pool.Exec(ctx, `UPDATE campaigns SET status=$1 WHERE id=$2`, string(status), id)
	return err
}

// CreateCampaignTarget inserts a campaign_target using a caller-provided id and
// signed RID (the RID is derived from the id, so it must be set before insert).
func (s *Store) CreateCampaignTarget(ctx context.Context, ct *models.CampaignTarget) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO campaign_targets(id,campaign_id,target_id,rid,status)
		 VALUES($1,$2,$3,$4,'pending')`,
		ct.ID, ct.CampaignID, ct.TargetID, ct.RID,
	)
	return err
}

func (s *Store) SetCampaignTargetRID(ctx context.Context, id uuid.UUID, rid string) error {
	_, err := s.pool.Exec(ctx, `UPDATE campaign_targets SET rid=$1 WHERE id=$2`, rid, id)
	return err
}

// PendingCampaignTargets returns campaign_targets for a campaign that still need sending.
func (s *Store) PendingCampaignTargets(ctx context.Context, campaignID uuid.UUID) ([]models.CampaignTarget, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,campaign_id,target_id,rid,status FROM campaign_targets
		 WHERE campaign_id=$1 AND status='pending' ORDER BY id`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.CampaignTarget{}
	for rows.Next() {
		var ct models.CampaignTarget
		if err := rows.Scan(&ct.ID, &ct.CampaignID, &ct.TargetID, &ct.RID, &ct.Status); err != nil {
			return nil, err
		}
		out = append(out, ct)
	}
	return out, rows.Err()
}

func (s *Store) SetCampaignTargetStatus(ctx context.Context, id uuid.UUID, status, errMsg string) error {
	_, err := s.pool.Exec(ctx, `UPDATE campaign_targets SET status=$1, error=$2 WHERE id=$3`, status, errMsg, id)
	return err
}

// CampaignTargetByRID resolves a tracking id to its campaign_target + related ids.
func (s *Store) CampaignTargetByRID(ctx context.Context, rid string) (*models.CampaignTarget, *models.Campaign, *models.Target, error) {
	var ct models.CampaignTarget
	var c models.Campaign
	var t models.Target
	err := s.pool.QueryRow(ctx,
		`SELECT ct.id,ct.campaign_id,ct.target_id,ct.rid,ct.status,
		        c.id,c.engagement_id,c.landing_page_id,
		        t.id,t.email,t.first_name,t.last_name
		 FROM campaign_targets ct
		 JOIN campaigns c ON c.id=ct.campaign_id
		 JOIN targets t ON t.id=ct.target_id
		 WHERE ct.rid=$1`, rid,
	).Scan(&ct.ID, &ct.CampaignID, &ct.TargetID, &ct.RID, &ct.Status,
		&c.ID, &c.EngagementID, &c.LandingPageID,
		&t.ID, &t.Email, &t.FirstName, &t.LastName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, nil, err
	}
	return &ct, &c, &t, nil
}

// ---- Events ----

func (s *Store) RecordEvent(ctx context.Context, e *models.Event) error {
	meta := e.Meta
	if meta == "" {
		meta = "{}"
	}
	return s.pool.QueryRow(ctx,
		`INSERT INTO events(campaign_target_id,type,ip,user_agent,meta)
		 VALUES($1,$2,$3,$4,$5::jsonb) RETURNING id, created_at`,
		e.CampaignTargetID, string(e.Type), e.IP, e.UserAgent, meta,
	).Scan(&e.ID, &e.CreatedAt)
}

// FunnelCounts returns distinct-target counts per event type for a campaign.
func (s *Store) FunnelCounts(ctx context.Context, campaignID uuid.UUID) (map[string]int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT ev.type, count(DISTINCT ct.id)
		 FROM campaign_targets ct
		 JOIN events ev ON ev.campaign_target_id=ct.id
		 WHERE ct.campaign_id=$1
		 GROUP BY ev.type`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{"sent": 0, "open": 0, "click": 0, "submit": 0, "report": 0}
	for rows.Next() {
		var typ string
		var n int
		if err := rows.Scan(&typ, &n); err != nil {
			return nil, err
		}
		out[typ] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// total targets
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM campaign_targets WHERE campaign_id=$1`, campaignID).Scan(&total); err != nil {
		return nil, err
	}
	out["targets"] = total
	return out, nil
}

// Timeline returns recent events for a campaign with the target email attached.
func (s *Store) Timeline(ctx context.Context, campaignID uuid.UUID, limit int) ([]map[string]any, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx,
		`SELECT t.email, ev.type, ev.ip, ev.user_agent, ev.created_at
		 FROM events ev
		 JOIN campaign_targets ct ON ct.id=ev.campaign_target_id
		 JOIN targets t ON t.id=ct.target_id
		 WHERE ct.campaign_id=$1
		 ORDER BY ev.created_at DESC LIMIT $2`, campaignID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var email, typ, ip, ua string
		var ts any
		if err := rows.Scan(&email, &typ, &ip, &ua, &ts); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"email": email, "type": typ, "ip": ip, "user_agent": ua, "created_at": ts})
	}
	return out, rows.Err()
}
