package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/auth"
	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/furkan-enes-polatoglu/phishforge/internal/queue"
	"github.com/furkan-enes-polatoglu/phishforge/internal/scope"
	"github.com/google/uuid"
)

type createCampaignReq struct {
	Name             string     `json:"name"`
	EmailTemplateID  string     `json:"email_template_id"`
	LandingPageID    string     `json:"landing_page_id"`
	SendingProfileID string     `json:"sending_profile_id"`
	RatePerMinute    int        `json:"rate_per_minute"`
	LaunchAt         *time.Time `json:"launch_at"`
	SendWindowStart  int        `json:"send_window_start"`
	SendWindowEnd    int        `json:"send_window_end"`
	BusinessDaysOnly bool       `json:"business_days_only"`
	JitterSeconds    int        `json:"jitter_seconds"`
	WarmupBatch      int        `json:"warmup_batch"`
	RewriteLinks     bool       `json:"rewrite_links"`
	// Pretext realism (see models.Campaign doc comment): the visible From can
	// show an exact real address, decoupled from the sending profile's own
	// technically-authenticated domain used for SPF/DKIM.
	SpoofedFromName    string `json:"spoofed_from_name"`
	SpoofedFromAddress string `json:"spoofed_from_address"`
	ReplyTo            string `json:"reply_to"`
}

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	engID, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	eng, err := s.st.GetEngagement(r.Context(), p.OrgID, engID)
	if err != nil {
		writeError(w, http.StatusNotFound, "engagement not found")
		return
	}
	var req createCampaignReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	tplID := uuidMust(req.EmailTemplateID)
	lpID := uuidMust(req.LandingPageID)
	spID := uuidMust(req.SendingProfileID)
	if _, err := s.st.GetEmailTemplate(r.Context(), p.OrgID, tplID); err != nil {
		writeError(w, http.StatusBadRequest, "email_template_id invalid")
		return
	}
	if _, err := s.st.GetLandingPage(r.Context(), p.OrgID, lpID); err != nil {
		writeError(w, http.StatusBadRequest, "landing_page_id invalid")
		return
	}
	if _, err := s.st.GetSendingProfileFull(r.Context(), p.OrgID, spID); err != nil {
		writeError(w, http.StatusBadRequest, "sending_profile_id invalid")
		return
	}
	rate := req.RatePerMinute
	if rate <= 0 {
		rate = 30
	}
	windowEnd := req.SendWindowEnd
	if windowEnd == 0 {
		windowEnd = 24 // default: no window restriction
	}
	c := &models.Campaign{
		EngagementID: engID, Name: req.Name, EmailTemplateID: tplID,
		LandingPageID: lpID, SendingProfileID: spID, Status: models.CampaignDraft,
		RatePerMinute: rate, LaunchAt: req.LaunchAt,
		SendWindowStart: req.SendWindowStart, SendWindowEnd: windowEnd,
		BusinessDaysOnly: req.BusinessDaysOnly, JitterSeconds: req.JitterSeconds,
		WarmupBatch: req.WarmupBatch, RewriteLinks: req.RewriteLinks,
		SpoofedFromName: req.SpoofedFromName, SpoofedFromAddress: req.SpoofedFromAddress, ReplyTo: req.ReplyTo,
	}
	if err := s.st.CreateCampaign(r.Context(), c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Materialize campaign_targets for every in-scope target, each with a signed RID.
	rules, _ := s.st.ListScopeRules(r.Context(), engID)
	targets, _ := s.st.ListTargets(r.Context(), engID)
	added, skipped := 0, 0
	for _, t := range targets {
		if !scope.Allowed(t.Email, rules) {
			skipped++
			continue
		}
		// Generate the id up front so the RID (derived from it) can be signed and
		// inserted atomically.
		ct := &models.CampaignTarget{ID: uuid.New(), CampaignID: c.ID, TargetID: t.ID}
		ct.RID = auth.SignRID(s.cfg.RIDSecret, ct.ID)
		if err := s.st.CreateCampaignTarget(r.Context(), ct); err != nil {
			skipped++
			continue
		}
		added++
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "campaign.create", "campaign", c.ID.String(),
		map[string]any{"engagement": eng.ClientName, "targets": added, "skipped": skipped})
	writeJSON(w, http.StatusCreated, map[string]any{"campaign": c, "targets_added": added, "skipped": skipped})
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	engID, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.st.GetEngagement(r.Context(), p.OrgID, engID); err != nil {
		writeError(w, http.StatusNotFound, "engagement not found")
		return
	}
	list, err := s.st.ListCampaigns(r.Context(), engID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleLaunchCampaign(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	c, err := s.st.GetCampaign(r.Context(), p.OrgID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "campaign not found")
		return
	}
	eng, err := s.st.GetEngagement(r.Context(), p.OrgID, c.EngagementID)
	if err != nil {
		writeError(w, http.StatusNotFound, "engagement not found")
		return
	}
	// Primary guardrail: only launch inside an active, in-window engagement.
	if !eng.Active(time.Now()) {
		writeError(w, http.StatusForbidden, "engagement is not active or outside its authorized date window")
		return
	}
	if c.Status == models.CampaignRunning || c.Status == models.CampaignDone {
		writeError(w, http.StatusConflict, "campaign already launched")
		return
	}
	if err := s.st.SetCampaignStatus(r.Context(), c.ID, models.CampaignScheduled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// If scheduled for the future, the worker's scheduler loop will pick it up when
	// due; otherwise enqueue immediately for prompt sending.
	future := c.LaunchAt != nil && c.LaunchAt.After(time.Now())
	if !future {
		if err := s.q.EnqueueLaunch(r.Context(), queue.LaunchJob{CampaignID: c.ID, OrgID: p.OrgID}); err != nil {
			writeError(w, http.StatusInternalServerError, "enqueue failed: "+err.Error())
			return
		}
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "campaign.launch", "campaign", c.ID.String(),
		map[string]any{"client": eng.ClientName, "authz_ref": eng.AuthzRef, "scheduled_future": future})
	msg := "launching"
	if future {
		msg = "scheduled"
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": msg})
}

func (s *Server) handleStopCampaign(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	c, err := s.st.GetCampaign(r.Context(), p.OrgID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "campaign not found")
		return
	}
	stopped, err := s.st.StopCampaign(r.Context(), c.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !stopped {
		writeError(w, http.StatusConflict, "campaign is not running or scheduled")
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "campaign.stop", "campaign", c.ID.String(), nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) handleDeleteCampaign(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	c, err := s.st.GetCampaign(r.Context(), p.OrgID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "campaign not found")
		return
	}
	if err := s.st.DeleteCampaign(r.Context(), c.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "campaign.delete", "campaign", c.ID.String(),
		map[string]any{"name": c.Name})
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleCampaignReport(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	c, err := s.st.GetCampaign(r.Context(), p.OrgID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "campaign not found")
		return
	}
	funnel, err := s.st.FunnelCounts(r.Context(), c.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	variants, _ := s.st.VariantFunnel(r.Context(), c.ID)
	writeJSON(w, http.StatusOK, map[string]any{"campaign": c, "funnel": funnel, "variants": variants})
}

func (s *Server) handleCampaignTimeline(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	c, err := s.st.GetCampaign(r.Context(), p.OrgID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "campaign not found")
		return
	}
	tl, err := s.st.Timeline(r.Context(), c.ID, 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tl)
}
