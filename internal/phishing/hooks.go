package phishing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/notify"
	"github.com/google/uuid"
)

func randToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// notifyEvent fires webhooks/Slack subscribed to an event, in the background so
// it never blocks the target-facing response.
func (s *Server) notifyEvent(campaignID uuid.UUID, event string, targetEmail string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		orgID, err := s.st.OrgIDForCampaign(ctx, campaignID)
		if err != nil {
			return
		}
		hooks, err := s.st.WebhooksForEvent(ctx, orgID, event)
		if err != nil {
			return
		}
		for _, h := range hooks {
			if isChatWebhook(h.URL) {
				_ = notify.Slack(ctx, h.URL, fmt.Sprintf("🎣 PhishForge: *%s* by `%s`", event, targetEmail))
			} else {
				_ = notify.Webhook(ctx, h.URL, h.Secret, map[string]any{
					"event": event, "target": targetEmail, "campaign_id": campaignID, "ts": time.Now().UTC(),
				})
			}
		}
	}()
}

func isChatWebhook(u string) bool {
	l := strings.ToLower(u)
	return strings.Contains(l, "slack.com") || strings.Contains(l, "webhook.office.com") || strings.Contains(l, "office365")
}

// Reputation-safety thresholds. Below minSampleForPause, a single bounce/
// complaint can swing the rate wildly (e.g. 1 bounce out of 3 sent = 33%) so we
// wait for a meaningful sample before acting.
const (
	minSampleForPause = 20
	maxBounceRate     = 0.05  // 5% hard/soft bounce
	maxComplaintRate  = 0.003 // 0.3% spam complaints — the industry red line
)

// maybeAutoPause checks the campaign a bounce/complaint event just arrived for
// and stops it if the bounce or complaint rate has crossed a reputation-risk
// threshold. This protects the sending domain (and every future engagement
// that uses it) from a burnt-out campaign silently damaging deliverability
// while nobody is watching the dashboard.
func (s *Server) maybeAutoPause(ctx context.Context, campaignTargetID uuid.UUID) {
	campaignID, err := s.st.CampaignIDForCampaignTarget(ctx, campaignTargetID)
	if err != nil {
		return
	}
	sent, bounced, complained := 0, 0, 0
	sent, bounced, complained, err = s.st.BounceComplaintStats(ctx, campaignID)
	if err != nil || sent < minSampleForPause {
		return
	}
	bounceRate := float64(bounced) / float64(sent)
	complaintRate := float64(complained) / float64(sent)
	if bounceRate < maxBounceRate && complaintRate < maxComplaintRate {
		return
	}
	stopped, err := s.st.StopCampaign(ctx, campaignID)
	if err != nil || !stopped {
		return // already stopped/not running — nothing to do
	}
	orgID, _ := s.st.OrgIDForCampaign(ctx, campaignID)
	_ = s.st.Audit(ctx, orgID, nil, "campaign.auto_paused_reputation_risk", "campaign", campaignID.String(), map[string]any{
		"sent": sent, "bounced": bounced, "complained": complained,
		"bounce_rate": bounceRate, "complaint_rate": complaintRate,
	})
	s.notifyEvent(campaignID, "campaign_auto_paused_reputation_risk", "-")
}

// autoAssignTraining assigns the org's first training module to a target after a
// risky action, returning the completion URL for redirect (empty if none exists).
func (s *Server) autoAssignTraining(ctx context.Context, campaignID, targetID uuid.UUID) string {
	orgID, err := s.st.OrgIDForCampaign(ctx, campaignID)
	if err != nil {
		return ""
	}
	modules, err := s.st.ListTrainingModules(ctx, orgID)
	if err != nil || len(modules) == 0 {
		return ""
	}
	cid := campaignID
	token, err := s.st.AssignTraining(ctx, targetID, modules[0].ID, &cid, randToken(16))
	if err != nil {
		return ""
	}
	return s.cfg.PhishBaseURL + "/training/" + token
}
