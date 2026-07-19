package store

import (
	"context"
	"errors"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// LandingPageByCampaign fetches the landing page attached to a campaign directly,
// resolving the org via the campaign's engagement.
func (s *Store) LandingPageByCampaign(ctx context.Context, campaignID uuid.UUID) (*models.LandingPage, error) {
	var l models.LandingPage
	err := s.pool.QueryRow(ctx,
		`SELECT lp.id, lp.org_id, lp.name, lp.html, lp.capture_meta, lp.redirect_url, lp.created_at
		 FROM campaigns c
		 JOIN landing_pages lp ON lp.id = c.landing_page_id
		 WHERE c.id=$1`, campaignID,
	).Scan(&l.ID, &l.OrgID, &l.Name, &l.HTML, &l.CaptureMeta, &l.RedirectURL, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}
