package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/furkan-enes-polatoglu/phishforge/internal/auth"
	"github.com/furkan-enes-polatoglu/phishforge/internal/store"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Role         string `json:"role"`
	Username     string `json:"username"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	u, err := s.st.UserByUsername(r.Context(), req.Username)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	ok, err := auth.VerifyPassword(req.Password, u.PasswordHash)
	if err != nil || !ok {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	access, _ := s.tokens.Access(u.ID, u.OrgID, string(u.Role))
	refresh, _ := s.tokens.Refresh(u.ID, u.OrgID, string(u.Role))
	_ = s.st.Audit(r.Context(), u.OrgID, &u.ID, "auth.login", "user", u.ID.String(), nil)
	writeJSON(w, http.StatusOK, tokenResp{AccessToken: access, RefreshToken: refresh, Role: string(u.Role), Username: u.Username})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	claims, err := s.tokens.Parse(req.RefreshToken)
	if err != nil || claims.Kind != "refresh" {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	access, _ := s.tokens.Access(uuidMust(claims.Subject), claims.OrgID, claims.Role)
	writeJSON(w, http.StatusOK, map[string]string{"access_token": access})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r)
	u, err := s.st.UserByID(r.Context(), p.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": u.ID, "username": u.Username, "role": u.Role, "org_id": u.OrgID,
	})
}
