// Package worker consumes launch jobs and sends campaign email with rate limiting.
// Scope is re-validated at send time as a defense-in-depth guardrail: even if a
// campaign was somehow assembled with an out-of-scope target, the worker refuses
// to send to it.
package worker

import (
	"context"
	"log"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/config"
	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/furkan-enes-polatoglu/phishforge/internal/queue"
	"github.com/furkan-enes-polatoglu/phishforge/internal/scope"
	"github.com/furkan-enes-polatoglu/phishforge/internal/store"
)

type Worker struct {
	cfg   *config.Config
	st    *store.Store
	q     *queue.Queue
}

func New(cfg *config.Config, st *store.Store, q *queue.Queue) *Worker {
	return &Worker{cfg: cfg, st: st, q: q}
}

// Run blocks, consuming launch jobs until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	log.Printf("worker: started (concurrency=%d)", w.cfg.WorkerConcurrency)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		job, err := w.q.DequeueLaunch(ctx, 5*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("worker: dequeue error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if job == nil {
			continue
		}
		if err := w.processLaunch(ctx, *job); err != nil {
			log.Printf("worker: launch %s failed: %v", job.CampaignID, err)
		}
	}
}

func (w *Worker) processLaunch(ctx context.Context, job queue.LaunchJob) error {
	c, err := w.st.GetCampaign(ctx, job.OrgID, job.CampaignID)
	if err != nil {
		return err
	}
	eng, err := w.st.GetEngagement(ctx, job.OrgID, c.EngagementID)
	if err != nil {
		return err
	}
	// Guardrail: engagement must be active within its window.
	if !eng.Active(time.Now()) {
		_ = w.st.SetCampaignStatus(ctx, c.ID, models.CampaignDraft)
		log.Printf("worker: refusing campaign %s — engagement not active", c.ID)
		return nil
	}
	rules, err := w.st.ListScopeRules(ctx, eng.ID)
	if err != nil {
		return err
	}
	tpl, err := w.st.GetEmailTemplate(ctx, job.OrgID, c.EmailTemplateID)
	if err != nil {
		return err
	}
	profile, err := w.st.GetSendingProfileFull(ctx, job.OrgID, c.SendingProfileID)
	if err != nil {
		return err
	}

	pending, err := w.st.PendingCampaignTargets(ctx, c.ID)
	if err != nil {
		return err
	}
	_ = w.st.SetCampaignStatus(ctx, c.ID, models.CampaignRunning)

	rate := c.RatePerMinute
	if rate <= 0 {
		rate = 30
	}
	interval := time.Minute / time.Duration(rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	targets, err := w.st.ListTargets(ctx, eng.ID)
	if err != nil {
		return err
	}
	byID := map[string]models.Target{}
	for _, t := range targets {
		byID[t.ID.String()] = t
	}

	for _, ct := range pending {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		t, ok := byID[ct.TargetID.String()]
		if !ok {
			_ = w.st.SetCampaignTargetStatus(ctx, ct.ID, "error", "target missing")
			continue
		}
		// Defense-in-depth: re-check scope at send time.
		if !scope.Allowed(t.Email, rules) {
			_ = w.st.SetCampaignTargetStatus(ctx, ct.ID, "error", "out of scope")
			_ = w.st.Audit(ctx, job.OrgID, nil, "send.blocked_out_of_scope", "campaign_target", ct.ID.String(),
				map[string]any{"email": t.Email})
			continue
		}

		data := buildData(t, w.cfg.PhishBaseURL, ct.RID)
		htmlBody, err := Render(tpl.HTML, data)
		if err != nil {
			_ = w.st.SetCampaignTargetStatus(ctx, ct.ID, "error", "render: "+err.Error())
			continue
		}
		htmlBody = InjectTracking(htmlBody, data)
		textBody, _ := Render(tpl.Text, data)
		subject, _ := Render(tpl.Subject, data)

		msg := Message{
			From:     profile.FromAddress,
			FromName: profile.FromName,
			To:       t.Email,
			Subject:  subject,
			HTML:     htmlBody,
			Text:     textBody,
		}
		if err := Send(profile, msg); err != nil {
			_ = w.st.SetCampaignTargetStatus(ctx, ct.ID, "error", err.Error())
			continue
		}
		_ = w.st.SetCampaignTargetStatus(ctx, ct.ID, "sent", "")
		_ = w.st.RecordEvent(ctx, &models.Event{CampaignTargetID: ct.ID, Type: models.EventSent})
	}

	_ = w.st.SetCampaignStatus(ctx, c.ID, models.CampaignDone)
	_ = w.st.Audit(ctx, job.OrgID, nil, "campaign.completed", "campaign", c.ID.String(), nil)
	log.Printf("worker: campaign %s completed (%d targets)", c.ID, len(pending))
	return nil
}
