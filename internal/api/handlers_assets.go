package api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	Name                 string `json:"name"`
	HTML                 string `json:"html"`
	CaptureMeta          bool   `json:"capture_meta"`
	CaptureSubmittedData bool   `json:"capture_submitted_data"`
	CapturePasswords     bool   `json:"capture_passwords"`
	RedirectURL          string `json:"redirect_url"`
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
		CaptureMeta: req.CaptureMeta, CaptureSubmittedData: req.CaptureSubmittedData,
		CapturePasswords: req.CapturePasswords, RedirectURL: req.RedirectURL,
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

var importClient = &http.Client{
	Timeout: 20 * time.Second,
	CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

// handleImportLandingPage fetches a URL's HTML as a starting point for a landing
// page and injects a <base href> so the cloned page's relative CSS/images resolve.
// Operators must only use this against pages they are authorized to clone.
func (s *Server) handleImportLandingPage(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	var req importReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	target := strings.TrimSpace(req.URL)
	if target == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target // be forgiving: default to https
	}
	parsed, err := url.Parse(target)
	if err != nil || parsed.Host == "" {
		writeError(w, http.StatusBadRequest, "invalid URL")
		return
	}

	httpReq, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	// A realistic UA + Accept helps many sites return their normal HTML.
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	httpReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := importClient.Do(httpReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not fetch the page: "+err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("the site returned HTTP %d — it may block automated fetches; paste the HTML manually instead", resp.StatusCode))
		return
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "html") && !strings.Contains(ct, "text") {
		writeError(w, http.StatusBadGateway, "the URL did not return an HTML page (content-type: "+ct+")")
		return
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20)) // 4 MiB cap
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to read the page body")
		return
	}
	html := injectBaseHref(string(body), parsed.Scheme+"://"+parsed.Host)

	name := req.Name
	if name == "" {
		name = "Imported: " + parsed.Host
	}
	l := &models.LandingPage{OrgID: p.OrgID, Name: name, HTML: html}
	if err := s.st.CreateLandingPage(r.Context(), l); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "landing.import", "landing_page", l.ID.String(),
		map[string]any{"source_url": target})
	writeJSON(w, http.StatusCreated, l)
}

// injectBaseHref inserts a <base href> right after <head> so relative asset URLs
// in the cloned page resolve against the original origin.
func injectBaseHref(html, origin string) string {
	base := `<base href="` + origin + `/">`
	lower := strings.ToLower(html)
	if i := strings.Index(lower, "<head>"); i >= 0 {
		return html[:i+6] + base + html[i+6:]
	}
	if i := strings.Index(lower, "<head "); i >= 0 {
		if j := strings.Index(lower[i:], ">"); j >= 0 {
			pos := i + j + 1
			return html[:pos] + base + html[pos:]
		}
	}
	return base + html
}

// ---- Sending profiles ----

type sendingProfileReq struct {
	Name         string `json:"name"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	FromAddress  string `json:"from_address"`
	FromName     string `json:"from_name"`
	UseTLS       bool   `json:"use_tls"`
	DKIMDomain   string `json:"dkim_domain"`
	DKIMSelector string `json:"dkim_selector"`
	SignDKIM     bool   `json:"sign_dkim"`
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
		DKIMDomain: req.DKIMDomain, DKIMSelector: req.DKIMSelector, SignDKIM: req.SignDKIM,
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
