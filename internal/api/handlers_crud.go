package api

import (
	"net/http"

	"github.com/furkan-enes-polatoglu/phishforge/internal/dkim"
	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

// ---- Engagements ----

func (s *Server) handleUpdateEngagement(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req createEngagementReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	e := &models.Engagement{ID: id, OrgID: p.OrgID, ClientName: req.ClientName, AuthzRef: req.AuthzRef, StartsAt: req.StartsAt, EndsAt: req.EndsAt}
	if err := s.st.UpdateEngagement(r.Context(), e); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "engagement.update", "engagement", id.String(), nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteEngagement(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if err := s.st.DeleteEngagement(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "engagement.delete", "engagement", id.String(), nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ---- Email templates ----

func (s *Server) handleUpdateEmailTemplate(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req emailTemplateReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	t := &models.EmailTemplate{ID: id, Name: req.Name, Subject: req.Subject, HTML: req.HTML, Text: req.Text}
	if err := s.st.UpdateEmailTemplate(r.Context(), p.OrgID, t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteEmailTemplate(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if err := s.st.DeleteEmailTemplate(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleDuplicateEmailTemplate(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	src, err := s.st.GetEmailTemplate(r.Context(), p.OrgID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	cp := &models.EmailTemplate{OrgID: p.OrgID, Name: src.Name + " (copy)", Subject: src.Subject, HTML: src.HTML, Text: src.Text, Version: 1}
	if err := s.st.CreateEmailTemplate(r.Context(), cp); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cp)
}

// ---- Landing pages ----

func (s *Server) handleUpdateLandingPage(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req landingReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	l := &models.LandingPage{ID: id, Name: req.Name, HTML: req.HTML, CaptureMeta: req.CaptureMeta, CaptureSubmittedData: req.CaptureSubmittedData, CapturePasswords: req.CapturePasswords, RedirectURL: req.RedirectURL}
	if err := s.st.UpdateLandingPage(r.Context(), p.OrgID, l); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteLandingPage(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if err := s.st.DeleteLandingPage(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleDuplicateLandingPage(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	src, err := s.st.GetLandingPage(r.Context(), p.OrgID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	cp := &models.LandingPage{OrgID: p.OrgID, Name: src.Name + " (copy)", HTML: src.HTML, CaptureMeta: src.CaptureMeta, CaptureSubmittedData: src.CaptureSubmittedData, CapturePasswords: src.CapturePasswords, RedirectURL: src.RedirectURL}
	if err := s.st.CreateLandingPage(r.Context(), cp); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cp)
}

// ---- Sending profiles ----

func (s *Server) handleUpdateSendingProfile(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req sendingProfileReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	existing, err := s.st.GetSendingProfileFull(r.Context(), p.OrgID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if req.SMTPPort == 0 {
		req.SMTPPort = 587
	}
	prof := &models.SendingProfile{
		ID: id, OrgID: p.OrgID, Name: req.Name, SMTPHost: req.SMTPHost, SMTPPort: req.SMTPPort,
		Username: req.Username, Password: req.Password, FromAddress: req.FromAddress, FromName: req.FromName,
		UseTLS: req.UseTLS, DKIMDomain: req.DKIMDomain, DKIMSelector: req.DKIMSelector,
		DKIMPrivateKey: existing.DKIMPrivateKey, SignDKIM: req.SignDKIM,
	}
	// Keep the existing password/DKIM key if the update leaves them blank.
	if req.Password == "" {
		prof.Password = existing.Password
	}
	if err := s.st.UpdateSendingProfile(r.Context(), p.OrgID, prof); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteSendingProfile(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if err := s.st.DeleteSendingProfile(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleDuplicateSendingProfile(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	src, err := s.st.GetSendingProfileFull(r.Context(), p.OrgID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	cp := *src
	cp.ID = [16]byte{}
	cp.Name = src.Name + " (copy)"
	if err := s.st.CreateSendingProfile(r.Context(), &cp); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cp)
}

// handleGenerateDKIM generates a DKIM keypair for a sending profile, stores the
// private key, and returns the DNS TXT record to publish.
func (s *Server) handleGenerateDKIM(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	prof, err := s.st.GetSendingProfileFull(r.Context(), p.OrgID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Domain   string `json:"domain"`
		Selector string `json:"selector"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Domain == "" {
		body.Domain = domainOfEmail(prof.FromAddress)
	}
	if body.Selector == "" {
		body.Selector = "phishforge"
	}
	privPEM, dnsName, dnsValue, err := dkim.GenerateKey(body.Selector, body.Domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	prof.DKIMDomain = body.Domain
	prof.DKIMSelector = body.Selector
	prof.DKIMPrivateKey = privPEM
	prof.SignDKIM = true
	if err := s.st.UpdateSendingProfile(r.Context(), p.OrgID, prof); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "sending_profile.dkim_generate", "sending_profile", id.String(),
		map[string]any{"domain": body.Domain, "selector": body.Selector})
	writeJSON(w, http.StatusOK, map[string]string{
		"dns_record_name":  dnsName,
		"dns_record_type":  "TXT",
		"dns_record_value": dnsValue,
	})
}

func domainOfEmail(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == '@' {
			return addr[i+1:]
		}
	}
	return ""
}

// ---- Training modules ----

func (s *Server) handleUpdateTraining(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	var req trainingModuleReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	m := &models.TrainingModule{ID: id, Name: req.Name, HTML: req.HTML}
	if err := s.st.UpdateTrainingModule(r.Context(), p.OrgID, m); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteTraining(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	id, ok := urlUUID(w, r, "id")
	if !ok {
		return
	}
	if err := s.st.DeleteTrainingModule(r.Context(), p.OrgID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
