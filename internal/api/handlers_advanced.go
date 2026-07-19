package api

import (
	"net/http"
	"strings"

	"github.com/furkan-enes-polatoglu/phishforge/internal/auth"
	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

// ---- A/B variants ----

type variantReq struct {
	Name            string `json:"name"`
	EmailTemplateID string `json:"email_template_id"`
	Weight          int    `json:"weight"`
}

func (s *Server) handleAddVariant(w http.ResponseWriter, r *http.Request) {
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
	var req variantReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	tplID := uuidMust(req.EmailTemplateID)
	if _, err := s.st.GetEmailTemplate(r.Context(), p.OrgID, tplID); err != nil {
		writeError(w, http.StatusBadRequest, "email_template_id invalid")
		return
	}
	weight := req.Weight
	if weight <= 0 {
		weight = 1
	}
	v := &models.CampaignVariant{CampaignID: c.ID, Name: req.Name, EmailTemplateID: tplID, Weight: weight}
	if err := s.st.CreateVariant(r.Context(), v); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) handleListVariants(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.st.GetCampaign(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusNotFound, "campaign not found")
		return
	}
	list, err := s.st.ListVariants(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// ---- Training modules ----

type trainingModuleReq struct {
	Name string `json:"name"`
	HTML string `json:"html"`
}

func (s *Server) handleCreateTraining(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	var req trainingModuleReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	m := &models.TrainingModule{OrgID: p.OrgID, Name: req.Name, HTML: req.HTML}
	if err := s.st.CreateTrainingModule(r.Context(), m); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) handleListTraining(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	list, err := s.st.ListTrainingModules(r.Context(), p.OrgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleTrainingAssignments(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	list, err := s.st.ListTrainingAssignments(r.Context(), p.OrgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// ---- Risk scoring ----

func (s *Server) handleRiskScores(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if _, err := s.st.GetEngagement(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusNotFound, "engagement not found")
		return
	}
	scores, err := s.st.RiskScores(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, scores)
}

// ---- API keys ----

type apiKeyReq struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	var req apiKeyReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	role := models.Role(req.Role)
	if !role.Valid() {
		role = models.RoleOperator
	}
	fullKey, prefix, hash := auth.GenerateAPIKey()
	k := &models.APIKey{OrgID: p.OrgID, Name: req.Name, Prefix: prefix, Role: role}
	if err := s.st.CreateAPIKey(r.Context(), k, hash); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "apikey.create", "api_key", k.ID.String(),
		map[string]any{"name": k.Name, "role": string(role)})
	// The full key is returned exactly once.
	writeJSON(w, http.StatusCreated, map[string]any{"api_key": k, "key": fullKey})
}

func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	list, err := s.st.ListAPIKeys(r.Context(), p.OrgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if err := s.st.RevokeAPIKey(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// ---- Webhooks ----

type webhookReq struct {
	URL    string   `json:"url"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	var req webhookReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if !strings.HasPrefix(req.URL, "http") {
		writeError(w, http.StatusBadRequest, "url must be http(s)")
		return
	}
	if req.Events == nil {
		req.Events = []string{}
	}
	wh := &models.Webhook{OrgID: p.OrgID, URL: req.URL, Secret: req.Secret, Events: req.Events}
	if err := s.st.CreateWebhook(r.Context(), wh); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, wh)
}

func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	list, err := s.st.ListWebhooks(r.Context(), p.OrgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Do not leak secrets in the list response.
	for i := range list {
		list[i].Secret = ""
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if err := s.st.DeleteWebhook(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
