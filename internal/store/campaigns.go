package store

import (
	"context"
	"errors"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Columns qualified with the "c" alias so queries that JOIN engagements (which
// also has id/created_at) are unambiguous. All campaign queries alias campaigns AS c.
const campaignCols = `c.id,c.engagement_id,c.name,c.email_template_id,c.landing_page_id,c.sending_profile_id,c.status,c.launch_at,c.rate_per_minute,c.send_window_start,c.send_window_end,c.business_days_only,c.jitter_seconds,c.warmup_batch,c.rewrite_links,c.spoofed_from_name,c.spoofed_from_address,c.reply_to,c.created_at`

func scanCampaign(row interface {
	Scan(dest ...any) error
}, c *models.Campaign) error {
	return row.Scan(&c.ID, &c.EngagementID, &c.Name, &c.EmailTemplateID, &c.LandingPageID,
		&c.SendingProfileID, &c.Status, &c.LaunchAt, &c.RatePerMinute,
		&c.SendWindowStart, &c.SendWindowEnd, &c.BusinessDaysOnly, &c.JitterSeconds,
		&c.WarmupBatch, &c.RewriteLinks, &c.SpoofedFromName, &c.SpoofedFromAddress, &c.ReplyTo, &c.CreatedAt)
}

func (s *Store) CreateCampaign(ctx context.Context, c *models.Campaign) error {
	return s.pool.QueryRow(ctx,
		`INSERT INTO campaigns(engagement_id,name,email_template_id,landing_page_id,sending_profile_id,status,launch_at,rate_per_minute,send_window_start,send_window_end,business_days_only,jitter_seconds,warmup_batch,rewrite_links,spoofed_from_name,spoofed_from_address,reply_to)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17) RETURNING id, created_at`,
		c.EngagementID, c.Name, c.EmailTemplateID, c.LandingPageID, c.SendingProfileID, string(c.Status), c.LaunchAt, c.RatePerMinute,
		c.SendWindowStart, c.SendWindowEnd, c.BusinessDaysOnly, c.JitterSeconds, c.WarmupBatch, c.RewriteLinks,
		c.SpoofedFromName, c.SpoofedFromAddress, c.ReplyTo,
	).Scan(&c.ID, &c.CreatedAt)
}

func (s *Store) ListCampaigns(ctx context.Context, engagementID uuid.UUID) ([]models.Campaign, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+campaignCols+` FROM campaigns c WHERE c.engagement_id=$1 ORDER BY c.created_at DESC`, engagementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Campaign{}
	for rows.Next() {
		var c models.Campaign
		if err := scanCampaign(rows, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCampaign fetches a campaign scoped to an org (via its engagement).
func (s *Store) GetCampaign(ctx context.Context, orgID, id uuid.UUID) (*models.Campaign, error) {
	var c models.Campaign
	err := scanCampaign(s.pool.QueryRow(ctx,
		`SELECT `+campaignCols+` FROM campaigns c JOIN engagements e ON e.id=c.engagement_id
		 WHERE e.org_id=$1 AND c.id=$2`, orgID, id), &c)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// DueScheduledCampaigns returns scheduled campaigns whose launch_at has passed,
// for the worker's scheduler loop.
func (s *Store) DueScheduledCampaigns(ctx context.Context, now time.Time) ([]struct {
	CampaignID uuid.UUID
	OrgID      uuid.UUID
}, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT c.id, e.org_id FROM campaigns c JOIN engagements e ON e.id=c.engagement_id
		 WHERE c.status='scheduled' AND (c.launch_at IS NULL OR c.launch_at <= $1)`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		CampaignID uuid.UUID
		OrgID      uuid.UUID
	}
	for rows.Next() {
		var r struct {
			CampaignID uuid.UUID
			OrgID      uuid.UUID
		}
		if err := rows.Scan(&r.CampaignID, &r.OrgID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) SetCampaignStatus(ctx context.Context, id uuid.UUID, status models.CampaignStatus) error {
	_, err := s.pool.Exec(ctx, `UPDATE campaigns SET status=$1 WHERE id=$2`, string(status), id)
	return err
}

// StopCampaign halts a scheduled/running campaign. Returns true if it changed.
func (s *Store) StopCampaign(ctx context.Context, id uuid.UUID) (bool, error) {
	ct, err := s.pool.Exec(ctx,
		`UPDATE campaigns SET status='stopped' WHERE id=$1 AND status IN ('scheduled','running')`, id)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() == 1, nil
}

// DeleteCampaign removes a campaign (cascades to campaign_targets and events).
func (s *Store) DeleteCampaign(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM campaigns WHERE id=$1`, id)
	return err
}

// CampaignStatus returns just the status string (used by the worker to detect stops).
func (s *Store) CampaignStatus(ctx context.Context, id uuid.UUID) (string, error) {
	var st string
	err := s.pool.QueryRow(ctx, `SELECT status FROM campaigns WHERE id=$1`, id).Scan(&st)
	return st, err
}

// TryStartCampaign atomically transitions a campaign from 'scheduled' to 'running',
// returning true only for the caller that won the transition. Prevents two workers
// (queue + scheduler) from sending the same campaign concurrently.
func (s *Store) TryStartCampaign(ctx context.Context, id uuid.UUID) (bool, error) {
	ct, err := s.pool.Exec(ctx,
		`UPDATE campaigns SET status='running' WHERE id=$1 AND status='scheduled'`, id)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() == 1, nil
}

// CreateCampaignTarget inserts a campaign_target using a caller-provided id and
// signed RID (the RID is derived from the id, so it must be set before insert).
func (s *Store) CreateCampaignTarget(ctx context.Context, ct *models.CampaignTarget) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO campaign_targets(id,campaign_id,target_id,variant_id,rid,status)
		 VALUES($1,$2,$3,$4,$5,'pending')`,
		ct.ID, ct.CampaignID, ct.TargetID, ct.VariantID, ct.RID,
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
		`SELECT id,campaign_id,target_id,variant_id,rid,status FROM campaign_targets
		 WHERE campaign_id=$1 AND status='pending' ORDER BY id`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.CampaignTarget{}
	for rows.Next() {
		var ct models.CampaignTarget
		if err := rows.Scan(&ct.ID, &ct.CampaignID, &ct.TargetID, &ct.VariantID, &ct.RID, &ct.Status); err != nil {
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
	out := map[string]int{
		"sent": 0, "open": 0, "click": 0, "submit": 0, "report": 0,
		"delivered": 0, "bounced": 0, "complained": 0,
	}
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
		`SELECT t.email, ev.type, ev.ip, ev.user_agent, ev.meta, ev.created_at
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
		var meta map[string]any
		var ts any
		if err := rows.Scan(&email, &typ, &ip, &ua, &meta, &ts); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"email": email, "type": typ, "ip": ip, "user_agent": ua, "meta": meta, "created_at": ts})
	}
	return out, rows.Err()
}
