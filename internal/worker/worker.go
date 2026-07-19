// Package worker consumes launch jobs and sends campaign email with advanced
// controls: timezone-aware send windows, business-day gating, warm-up batching,
// per-send jitter, A/B variant selection, and automatic link rewriting.
//
// Scope is re-validated at send time as a defense-in-depth guardrail: even if a
// campaign was somehow assembled with an out-of-scope target, the worker refuses
// to send to it.
package worker

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/config"
	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/furkan-enes-polatoglu/phishforge/internal/queue"
	"github.com/furkan-enes-polatoglu/phishforge/internal/scope"
	"github.com/furkan-enes-polatoglu/phishforge/internal/store"
	"github.com/google/uuid"
)

type Worker struct {
	cfg *config.Config
	st  *store.Store
	q   *queue.Queue
}

func New(cfg *config.Config, st *store.Store, q *queue.Queue) *Worker {
	return &Worker{cfg: cfg, st: st, q: q}
}

// Run starts the scheduler loop and the queue consumer.
func (w *Worker) Run(ctx context.Context) error {
	log.Printf("worker: started (concurrency=%d)", w.cfg.WorkerConcurrency)
	go w.scheduler(ctx)
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

// scheduler periodically enqueues scheduled campaigns whose launch time has
// arrived (and re-enqueues campaigns that still have pending, out-of-window
// targets). This is how time-based scheduling and send windows are honored.
func (w *Worker) scheduler(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		due, err := w.st.DueScheduledCampaigns(ctx, time.Now())
		if err != nil {
			log.Printf("scheduler: %v", err)
			continue
		}
		for _, d := range due {
			_ = w.q.EnqueueLaunch(ctx, queue.LaunchJob{CampaignID: d.CampaignID, OrgID: d.OrgID})
		}
	}
}

func (w *Worker) processLaunch(ctx context.Context, job queue.LaunchJob) error {
	// Win the scheduled→running transition; skip if another worker already has it.
	won, err := w.st.TryStartCampaign(ctx, job.CampaignID)
	if err != nil {
		return err
	}
	if !won {
		return nil
	}

	c, err := w.st.GetCampaign(ctx, job.OrgID, job.CampaignID)
	if err != nil {
		return err
	}
	eng, err := w.st.GetEngagement(ctx, job.OrgID, c.EngagementID)
	if err != nil {
		return err
	}
	if !eng.Active(time.Now()) {
		_ = w.st.SetCampaignStatus(ctx, c.ID, models.CampaignDraft)
		log.Printf("worker: refusing campaign %s — engagement not active", c.ID)
		return nil
	}
	rules, err := w.st.ListScopeRules(ctx, eng.ID)
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
	targets, err := w.st.ListTargets(ctx, eng.ID)
	if err != nil {
		return err
	}

	// A/B: if the campaign has variants, assign one (weighted round-robin) to any
	// pending target that doesn't have one yet, so sends split across variants.
	variants, _ := w.st.ListVariants(ctx, c.ID)
	if len(variants) > 0 {
		var pool []uuid.UUID
		for _, v := range variants {
			weight := v.Weight
			if weight < 1 {
				weight = 1
			}
			for i := 0; i < weight; i++ {
				pool = append(pool, v.ID)
			}
		}
		idx := 0
		for i := range pending {
			if pending[i].VariantID == nil {
				vid := pool[idx%len(pool)]
				idx++
				if err := w.st.SetCampaignTargetVariant(ctx, pending[i].ID, vid); err == nil {
					pending[i].VariantID = &vid
				}
			}
		}
	}
	byID := map[string]models.Target{}
	for _, t := range targets {
		byID[t.ID.String()] = t
	}

	rate := c.RatePerMinute
	if rate <= 0 {
		rate = 30
	}
	interval := time.Minute / time.Duration(rate)
	sentThisCycle := 0
	remaining := 0

	for _, ct := range pending {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// Honor a stop requested mid-run: leave remaining targets pending.
		if st, err := w.st.CampaignStatus(ctx, c.ID); err == nil && st == string(models.CampaignStopped) {
			log.Printf("worker: campaign %s stopped by operator", c.ID)
			return nil
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
		// Send window: evaluate in the target's local timezone.
		if !c.InSendWindow(localNow(t.Timezone)) {
			remaining++
			continue
		}
		// Warm-up: cap sends per scheduler cycle.
		if c.WarmupBatch > 0 && sentThisCycle >= c.WarmupBatch {
			remaining++
			continue
		}

		tplID := w.st.TemplateIDForCampaignTarget(ctx, ct, c.EmailTemplateID)
		tpl, err := w.st.GetEmailTemplate(ctx, job.OrgID, tplID)
		if err != nil {
			_ = w.st.SetCampaignTargetStatus(ctx, ct.ID, "error", "template: "+err.Error())
			continue
		}

		data := buildData(t, w.cfg.PhishBaseURL, ct.RID)
		htmlBody, err := Render(tpl.HTML, data)
		if err != nil {
			_ = w.st.SetCampaignTargetStatus(ctx, ct.ID, "error", "render: "+err.Error())
			continue
		}
		if c.RewriteLinks {
			htmlBody = RewriteLinks(htmlBody, data.TrackURL)
		}
		htmlBody = InjectTracking(htmlBody, data)
		textBody, _ := Render(tpl.Text, data)
		subject, _ := Render(tpl.Subject, data)

		// Pretext realism: the visible From can show an exact real address
		// (SpoofedFromAddress) while the envelope/DKIM identity always stays
		// the sending profile's own authenticated address — see Message's
		// doc comment for why this split exists and what it does to alignment.
		msg := Message{
			From: profile.FromAddress, FromName: profile.FromName,
			HeaderFrom: c.SpoofedFromAddress, HeaderFromName: c.SpoofedFromName,
			To: t.Email, Subject: subject, HTML: htmlBody, Text: textBody,
			Unsubscribe: data.ReportURL, XMailer: profile.XMailer, ReplyTo: c.ReplyTo,
			Variables: map[string]string{"cid": ct.ID.String()},
		}
		if err := Send(profile, msg); err != nil {
			_ = w.st.SetCampaignTargetStatus(ctx, ct.ID, "error", err.Error())
			continue
		}
		_ = w.st.SetCampaignTargetStatus(ctx, ct.ID, "sent", "")
		_ = w.st.RecordEvent(ctx, &models.Event{CampaignTargetID: ct.ID, Type: models.EventSent})
		sentThisCycle++

		// Rate limit + optional jitter between sends.
		delay := interval
		if c.JitterSeconds > 0 {
			delay += time.Duration(rand.Intn(c.JitterSeconds+1)) * time.Second
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	if remaining > 0 {
		// Leave the campaign scheduled so the scheduler retries the rest when the
		// send window opens or the next warm-up cycle is due.
		_ = w.st.SetCampaignStatus(ctx, c.ID, models.CampaignScheduled)
		log.Printf("worker: campaign %s sent %d this cycle, %d pending (window/warm-up)", c.ID, sentThisCycle, remaining)
		return nil
	}
	_ = w.st.SetCampaignStatus(ctx, c.ID, models.CampaignDone)
	_ = w.st.Audit(ctx, job.OrgID, nil, "campaign.completed", "campaign", c.ID.String(), nil)
	log.Printf("worker: campaign %s completed", c.ID)
	return nil
}

// localNow returns the current time in the given IANA timezone (falls back to UTC).
func localNow(tz string) time.Time {
	if tz == "" {
		return time.Now().UTC()
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Now().UTC()
	}
	return time.Now().In(loc)
}
