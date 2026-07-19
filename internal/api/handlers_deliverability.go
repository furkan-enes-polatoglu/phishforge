package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/deliverability"
	"github.com/furkan-enes-polatoglu/phishforge/internal/gateway"
	"github.com/furkan-enes-polatoglu/phishforge/internal/seedtest"
)

type deliverabilityReq struct {
	Domain       string `json:"domain"`
	DKIMSelector string `json:"dkim_selector"`
	SenderIP     string `json:"sender_ip"`
	Subject      string `json:"subject"`
	HTML         string `json:"html"`
	RawMessage   string `json:"raw_message"` // optional, for SpamAssassin scoring
}

// handleDeliverabilityCheck runs legitimate pre-send email health checks —
// SPF/DKIM/DMARC with alignment analysis, PTR/FCrDNS, MTA-STS/TLS-RPT,
// parallel blocklist lookups, and content/spam-trigger analysis — aggregated
// into one delivery confidence score. It does not perform, and must not be used
// for, spam-filter evasion — the intent is to verify correct email
// authentication and coordinate allowlisting with the client.
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
	res.MTASTS = ptr(deliverability.CheckMTASTS(r.Context(), req.Domain))
	res.TLSRPT = deliverability.CheckTLSRPT(r.Context(), req.Domain)
	if req.SenderIP != "" {
		res.RBL = deliverability.CheckIPReputation(r.Context(), req.SenderIP)
		p := deliverability.CheckPTR(r.Context(), req.SenderIP)
		res.PTR = &p
	}
	if req.HTML != "" {
		res.HTMLLint = deliverability.LintHTML(req.HTML)
	}
	if req.HTML != "" || req.Subject != "" {
		c := deliverability.AnalyzeContent(req.Subject, req.HTML)
		res.Content = &c
	}
	if req.RawMessage != "" {
		if score, err := deliverability.SpamScore(r.Context(), s.cfg.SpamdAddr, req.RawMessage); err == nil {
			res.SpamScore = score
		}
	}
	res.Score = res.ComputeScore()
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "deliverability.check", "domain", req.Domain,
		map[string]any{"score": res.Score.Score, "grade": res.Score.Grade})
	writeJSON(w, http.StatusOK, res)
}

func ptr[T any](v T) *T { return &v }

type gatewayCheckReq struct {
	TargetDomain  string    `json:"target_domain"`
	ClientName    string    `json:"client_name"`
	SendingDomain string    `json:"sending_domain"`
	SendingIP     string    `json:"sending_ip"`
	DKIMDomain    string    `json:"dkim_domain"`
	DKIMSelector  string    `json:"dkim_selector"`
	StartsAt      time.Time `json:"starts_at"`
	EndsAt        time.Time `json:"ends_at"`
}

// handleGatewayCheck is the headline deliverability feature: it fingerprints the
// TARGET organization's email security gateway from its MX records and returns
// a tailored, step-by-step allowlisting playbook plus a ready-to-send request
// email the operator can forward to the client's IT/security team. This turns
// "coordinate an allowlist with the client" from vague advice into an automated,
// provider-specific deliverable.
func (s *Server) handleGatewayCheck(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	var req gatewayCheckReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if strings.TrimSpace(req.TargetDomain) == "" {
		writeError(w, http.StatusBadRequest, "target_domain zorunlu")
		return
	}
	provider, hosts, err := gateway.DetectProvider(r.Context(), req.TargetDomain)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if req.StartsAt.IsZero() {
		req.StartsAt = time.Now()
	}
	if req.EndsAt.IsZero() {
		req.EndsAt = req.StartsAt.AddDate(0, 0, 30)
	}
	coverEmail := gateway.BuildCoverEmail(provider, gateway.CoverEmailRequest{
		ClientName: req.ClientName, SendingDomain: req.SendingDomain, SendingIP: req.SendingIP,
		DKIMDomain: req.DKIMDomain, DKIMSelector: req.DKIMSelector, StartsAt: req.StartsAt, EndsAt: req.EndsAt,
	})
	resp := map[string]any{"mx_hosts": hosts, "cover_email": coverEmail, "provider": nil}
	if provider != nil {
		resp["provider"] = provider
	}
	_ = s.st.Audit(r.Context(), p.OrgID, &p.UserID, "deliverability.gateway_check", "domain", req.TargetDomain,
		map[string]any{"provider": provider})
	writeJSON(w, http.StatusOK, resp)
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
