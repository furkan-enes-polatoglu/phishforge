package api

import (
	"net/http"
	"strings"

	"github.com/furkan-enes-polatoglu/phishforge/internal/deliverability"
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
