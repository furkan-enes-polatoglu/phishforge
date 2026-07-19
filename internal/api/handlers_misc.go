package api

import (
	"net/http"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	engagements, _ := s.st.ListEngagements(r.Context(), p.OrgID)
	active := 0
	for _, e := range engagements {
		if e.Status == "active" {
			active++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"org_id":              p.OrgID,
		"engagements_total":   len(engagements),
		"engagements_active":  active,
		"role":                p.Role,
	})
}

func (s *Server) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	entries, err := s.st.AuditList(r.Context(), p.OrgID, 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entries)
}
