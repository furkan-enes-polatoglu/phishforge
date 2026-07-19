package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/furkan-enes-polatoglu/phishforge/internal/auth"
	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/google/uuid"
)

type ctxKey string

const claimsKey ctxKey = "claims"

// principal is the authenticated identity extracted from the access token.
type principal struct {
	UserID uuid.UUID
	OrgID  uuid.UUID
	Role   models.Role
}

func (s *Server) authRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API-key auth (automation) takes precedence when the header is present.
		if key := r.Header.Get("X-API-Key"); key != "" {
			if p, ok := s.authAPIKey(r, key); ok {
				ctx := context.WithValue(r.Context(), claimsKey, p)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			writeError(w, http.StatusUnauthorized, "invalid API key")
			return
		}
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing bearer token or API key")
			return
		}
		claims, err := s.tokens.Parse(strings.TrimPrefix(h, "Bearer "))
		if err != nil || claims.Kind != "access" {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		uid, err := uuid.Parse(claims.Subject)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid subject")
			return
		}
		p := principal{UserID: uid, OrgID: claims.OrgID, Role: models.Role(claims.Role)}
		ctx := context.WithValue(r.Context(), claimsKey, p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func mustPrincipal(r *http.Request) principal {
	p, _ := r.Context().Value(claimsKey).(principal)
	return p
}

// authAPIKey validates an X-API-Key header and returns the corresponding principal.
func (s *Server) authAPIKey(r *http.Request, key string) (principal, bool) {
	prefix := auth.APIKeyPrefix(key)
	if prefix == "" {
		return principal{}, false
	}
	orgID, role, hash, err := s.st.APIKeyByPrefix(r.Context(), prefix)
	if err != nil || !auth.VerifyAPIKey(key, hash) {
		return principal{}, false
	}
	s.st.TouchAPIKey(r.Context(), prefix)
	return principal{UserID: uuid.Nil, OrgID: orgID, Role: models.Role(role)}, true
}

// requireRole wraps a handler, enforcing a minimum role.
func (s *Server) requireRole(min models.Role, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := mustPrincipal(r)
		if !p.Role.AtLeast(min) {
			writeError(w, http.StatusForbidden, "insufficient role")
			return
		}
		h(w, r)
	}
}
