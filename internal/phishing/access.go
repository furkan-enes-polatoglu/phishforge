package phishing

import (
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

// campaignServable reports whether tracking/landing endpoints should still
// respond for a campaign. Stopping a campaign (manually, or via the
// reputation-safety auto-pause) immediately cuts off access to its landing
// pages and tracking pixels — an already-sent link simply stops resolving.
// Access is also cut off once the owning engagement is no longer active
// (closed, or outside its authorized date window), so simulation
// infrastructure doesn't stay live past the end of an engagement.
func campaignServable(eng models.Engagement, c models.Campaign, now time.Time) bool {
	return eng.Active(now) && c.Status != models.CampaignStopped
}
