package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

// ---- Email templates ----

type emailTemplateReq struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
	Text    string `json:"text"`
}

func (s *Server) handleCreateEmailTemplate(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	var req emailTemplateReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Subject) == "" {
		writeError(w, http.StatusBadRequest, "name and subject required")
		return
	}
	t := &models.EmailTemplate{
		OrgID: p.OrgID, Name: req.Name, Subject: req.Subject,
		HTML: req.HTML, Text: req.Text, Version: 1,
	}
	if err := s.st.CreateEmailTemplate(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleListEmailTemplates(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	list, err := s.st.ListEmailTemplates(r.Context(), p.OrgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// ---- Landing pages ----

type landingReq struct {
	Name        string `json:"name"`
	HTML        string `json:"html"`
	CaptureMeta bool   `json:"capture_meta"`
	RedirectURL string `json:"redirect_url"`
}

func (s *Server) handleCreateLandingPage(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	var req landingReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	l := &models.LandingPage{
		OrgID: p.OrgID, Name: req.Name, HTML: req.HTML,
		CaptureMeta: req.CaptureMeta, RedirectURL: req.RedirectURL,
	}
	if err := s.st.CreateLandingPage(r.Context(), l); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

func (s *Server) handleListLandingPages(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	list, err := s.st.ListLandingPages(r.Context(), p.OrgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

type importReq struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// handleImportLandingPage fetches a URL's HTML as a starting point for a landing
// page. Operators are expected to use this only against pages they are authorized
// to clone for the engagement.
func (s *Server) handleImportLandingPage(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	var req importReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		writeError(w, http.StatusBadRequest, "url must be http(s)")
		return
	}
	resp, err := http.Get(req.URL)
	if err != nil {
		writeError(w, http.StatusBadGateway, "fetch failed: "+err.Error())
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2 MiB cap
	if err != nil {
		writeError(w, http.StatusBadGateway, "read failed")
		return
	}
	name := req.Name
	if name == "" {
		name = "Imported page"
	}
	l := &models.LandingPage{OrgID: p.OrgID, Name: name, HTML: string(body)}
	if err := s.st.CreateLandingPage(r.Context(), l); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "landing.import", "landing_page", l.ID.String(),
		map[string]any{"source_url": req.URL})
	writeJSON(w, http.StatusCreated, l)
}

// ---- Sending profiles ----

type sendingProfileReq struct {
	Name        string `json:"name"`
	SMTPHost    string `json:"smtp_host"`
	SMTPPort    int    `json:"smtp_port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	FromAddress string `json:"from_address"`
	FromName    string `json:"from_name"`
	UseTLS      bool   `json:"use_tls"`
}

func (s *Server) handleCreateSendingProfile(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	var req sendingProfileReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.SMTPHost == "" || req.FromAddress == "" {
		writeError(w, http.StatusBadRequest, "smtp_host and from_address required")
		return
	}
	if req.SMTPPort == 0 {
		req.SMTPPort = 587
	}
	prof := &models.SendingProfile{
		OrgID: p.OrgID, Name: req.Name, SMTPHost: req.SMTPHost, SMTPPort: req.SMTPPort,
		Username: req.Username, Password: req.Password, FromAddress: req.FromAddress,
		FromName: req.FromName, UseTLS: req.UseTLS,
	}
	if err := s.st.CreateSendingProfile(r.Context(), prof); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, prof)
}

func (s *Server) handleListSendingProfiles(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	list, err := s.st.ListSendingProfiles(r.Context(), p.OrgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}
