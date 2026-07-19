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
