package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/furkan-enes-polatoglu/phishforge/internal/scope"
)

type createEngagementReq struct {
	ClientName string    `json:"client_name"`
	AuthzRef   string    `json:"authz_ref"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
}

func (s *Server) handleCreateEngagement(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	var req createEngagementReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	// Guardrail: an engagement without an authorization reference or a valid window
	// cannot be created. This is the authorization record for the whole campaign.
	if strings.TrimSpace(req.ClientName) == "" || strings.TrimSpace(req.AuthzRef) == "" {
		writeError(w, http.StatusBadRequest, "client_name and authz_ref are required")
		return
	}
	if req.EndsAt.Before(req.StartsAt) || req.EndsAt.IsZero() {
		writeError(w, http.StatusBadRequest, "ends_at must be after starts_at")
		return
	}
	e := &models.Engagement{
		OrgID: p.OrgID, ClientName: req.ClientName, AuthzRef: req.AuthzRef,
		StartsAt: req.StartsAt, EndsAt: req.EndsAt, Status: models.EngagementDraft,
	}
	if err := s.st.CreateEngagement(r.Context(), e); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "engagement.create", "engagement", e.ID.String(),
		map[string]any{"client": e.ClientName, "authz_ref": e.AuthzRef})
	writeJSON(w, http.StatusCreated, e)
}

func (s *Server) handleListEngagements(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	list, err := s.st.ListEngagements(r.Context(), p.OrgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleGetEngagement(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	e, err := s.st.GetEngagement(r.Context(), p.OrgID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "engagement not found")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

type setStatusReq struct {
	Status string `json:"status"`
}

func (s *Server) handleSetEngagementStatus(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req setStatusReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	st := models.EngagementStatus(req.Status)
	if st != models.EngagementDraft && st != models.EngagementActive && st != models.EngagementClosed {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	// Guardrail: activating requires at least one scope rule.
	if st == models.EngagementActive {
		rules, _ := s.st.ListScopeRules(r.Context(), id)
		if len(rules) == 0 {
			writeError(w, http.StatusBadRequest, "cannot activate: define at least one scope (allowlist) rule first")
			return
		}
	}
	if err := s.st.SetEngagementStatus(r.Context(), p.OrgID, id, st); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "engagement.status", "engagement", id.String(),
		map[string]any{"status": req.Status})
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

// ---- scope ----

type addScopeReq struct {
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
}

func (s *Server) handleListScope(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.st.GetEngagement(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusNotFound, "engagement not found")
		return
	}
	rules, err := s.st.ListScopeRules(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (s *Server) handleAddScope(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.st.GetEngagement(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusNotFound, "engagement not found")
		return
	}
	var req addScopeReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Kind != "domain" && req.Kind != "email" {
		writeError(w, http.StatusBadRequest, "kind must be domain or email")
		return
	}
	if strings.TrimSpace(req.Pattern) == "" {
		writeError(w, http.StatusBadRequest, "pattern required")
		return
	}
	rule := &models.ScopeRule{EngagementID: id, Kind: req.Kind, Pattern: req.Pattern}
	if err := s.st.AddScopeRule(r.Context(), rule); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "scope.add", "engagement", id.String(),
		map[string]any{"kind": req.Kind, "pattern": rule.Pattern})
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleDeleteScope(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	ruleID, ok := urlUUID(w, r, "ruleID")
	if !ok {
		return
	}
	if _, err := s.st.GetEngagement(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusNotFound, "engagement not found")
		return
	}
	if err := s.st.DeleteScopeRule(r.Context(), id, ruleID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "scope.delete", "engagement", id.String(), nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ---- targets ----

type createTargetsReq struct {
	Targets []struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Position  string `json:"position"`
		Timezone  string `json:"timezone"`
	} `json:"targets"`
}

func (s *Server) handleCreateTargets(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.st.GetEngagement(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusNotFound, "engagement not found")
		return
	}
	var req createTargetsReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	rules, _ := s.st.ListScopeRules(r.Context(), id)

	created := []models.Target{}
	rejected := []string{}
	for _, in := range req.Targets {
		email := strings.ToLower(strings.TrimSpace(in.Email))
		if email == "" {
			continue
		}
		// Guardrail: reject targets outside the engagement allowlist at import time.
		if !scope.Allowed(email, rules) {
			rejected = append(rejected, email)
			continue
		}
		tz := in.Timezone
		if tz == "" {
			tz = "UTC"
		}
		t := &models.Target{
			EngagementID: id, Email: email, FirstName: in.FirstName,
			LastName: in.LastName, Position: in.Position, Timezone: tz,
		}
		if err := s.st.CreateTarget(r.Context(), t); err != nil {
			rejected = append(rejected, email)
			continue
		}
		created = append(created, *t)
	}
	if len(rejected) > 0 {
		_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "targets.rejected_out_of_scope", "engagement", id.String(),
			map[string]any{"count": len(rejected), "emails": rejected})
	}
	writeJSON(w, http.StatusOK, map[string]any{"created": created, "rejected_out_of_scope": rejected})
}

func (s *Server) handleListTargets(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.st.GetEngagement(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusNotFound, "engagement not found")
		return
	}
	list, err := s.st.ListTargets(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}
