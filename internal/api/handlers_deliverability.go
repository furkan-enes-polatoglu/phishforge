package api

import (
	"net/http"
	"strings"

	"github.com/furkan-enes-polatoglu/phishforge/internal/deliverability"
	"github.com/furkan-enes-polatoglu/phishforge/internal/seedtest"
)

type deliverabilityReq struct {
	Domain       string `json:"domain"`
	DKIMSelector string `json:"dkim_selector"`
	SenderIP     string `json:"sender_ip"`
	HTML         string `json:"html"`
	RawMessage   string `json:"raw_message"` // optional, for SpamAssassin scoring
}

// handleDeliverabilityCheck runs legitimate pre-send email health checks. It does
// not perform, and must not be used for, spam-filter evasion — the intent is to
// verify correct email authentication and coordinate allowlisting with the client.
func (s *Server) handleDeliverabilityCheck(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	var req deliverabilityReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if strings.TrimSpace(req.Domain) == "" {
		writeError(w, http.StatusBadRequest, "domain required")
		return
	}
	res := deliverability.CheckDomain(r.Context(), req.Domain, req.DKIMSelector)
	if req.SenderIP != "" {
		res.RBL = deliverability.CheckIPReputation(r.Context(), req.SenderIP)
	}
	if req.HTML != "" {
		res.HTMLLint = deliverability.LintHTML(req.HTML)
	}
	if req.RawMessage != "" {
		if score, err := deliverability.SpamScore(r.Context(), s.cfg.SpamdAddr, req.RawMessage); err == nil {
			res.SpamScore = score
		}
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "deliverability.check", "domain", req.Domain, nil)
	writeJSON(w, http.StatusOK, res)
}

type seedCheckReq struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	UseTLS        bool   `json:"use_tls"`
	SubjectMarker string `json:"subject_marker"`
}

// handleSeedCheck performs a real inbox-placement check: it logs into a seed
// mailbox over IMAP and reports whether a marked test message landed in the
// inbox or a spam/junk folder. Requires a test email to have already been sent
// to the seed address with subject_marker somewhere in its subject line.
func (s *Server) handleSeedCheck(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	var req seedCheckReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if strings.TrimSpace(req.Host) == "" || strings.TrimSpace(req.SubjectMarker) == "" {
		writeError(w, http.StatusBadRequest, "host ve subject_marker zorunlu")
		return
	}
	if req.Port == 0 {
		req.Port = 993
	}
	folder, found, err := seedtest.CheckPlacement(seedtest.Config{
		Host: req.Host, Port: req.Port, Username: req.Username, Password: req.Password, UseTLS: req.UseTLS,
	}, req.SubjectMarker)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "deliverability.seed_check", "seed_mailbox", req.Host,
		map[string]any{"found": found, "folder": folder})
	writeJSON(w, http.StatusOK, map[string]any{"found": found, "folder": folder})
}
