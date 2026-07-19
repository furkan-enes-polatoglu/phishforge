package phishing

import (
	"testing"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

func TestCampaignServable(t *testing.T) {
	now := time.Now()
	activeEng := models.Engagement{
		Status: models.EngagementActive, StartsAt: now.Add(-24 * time.Hour), EndsAt: now.Add(24 * time.Hour),
	}
	closedEng := activeEng
	closedEng.Status = models.EngagementClosed
	expiredEng := activeEng
	expiredEng.EndsAt = now.Add(-time.Hour)

	draft := models.Campaign{Status: models.CampaignDraft}
	scheduled := models.Campaign{Status: models.CampaignScheduled}
	running := models.Campaign{Status: models.CampaignRunning}
	completed := models.Campaign{Status: models.CampaignDone}
	stopped := models.Campaign{Status: models.CampaignStopped}

	cases := []struct {
		name string
		eng  models.Engagement
		c    models.Campaign
		want bool
	}{
		{"active engagement, draft campaign — not launched yet, not servable", activeEng, draft, false},
		{"active engagement, scheduled campaign — not started sending yet, not servable", activeEng, scheduled, false},
		{"active engagement, running campaign — servable", activeEng, running, true},
		{"active engagement, completed campaign — late clicks still tracked", activeEng, completed, true},
		{"active engagement, stopped campaign — access cut off", activeEng, stopped, false},
		{"closed engagement, running campaign — access cut off", closedEng, running, false},
		{"expired engagement window, running campaign — access cut off", expiredEng, running, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := campaignServable(tc.eng, tc.c, now); got != tc.want {
				t.Errorf("campaignServable() = %v, want %v", got, tc.want)
			}
		})
	}
}
