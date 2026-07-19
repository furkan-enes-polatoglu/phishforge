package phishing

import (
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

// campaignServable reports whether tracking/landing endpoints should respond
// for a campaign right now. Access is only granted once a campaign has
// actually started sending (status "running") or finished ("completed") —
// a campaign that's still a draft, or scheduled for a future launch time,
// is not live yet, even if its campaign_targets (and their signed rids)
// already exist in the database. Stopping a campaign (manually, or via the
// reputation-safety auto-pause) immediately cuts access back off — an
// already-sent link simply stops resolving. Access is also cut off once the
// owning engagement is no longer active (closed, or outside its authorized
// date window), so simulation infrastructure doesn't stay live past the end
// of an engagement.
func campaignServable(eng models.Engagement, c models.Campaign, now time.Time) bool {
	if !eng.Active(now) {
		return false
	}
	switch c.Status {
	case models.CampaignRunning, models.CampaignDone:
		return true
	default: // draft, scheduled, stopped
		return false
	}
}
