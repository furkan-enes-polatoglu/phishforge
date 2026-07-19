// Package api implements the admin REST API (Chi), including auth, RBAC, and
// tenant-scoped handlers. It also serves the built SPA when present.
package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/auth"
	"github.com/furkan-enes-polatoglu/phishforge/internal/config"
	"github.com/furkan-enes-polatoglu/phishforge/internal/queue"
	"github.com/furkan-enes-polatoglu/phishforge/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	cfg    *config.Config
	st     *store.Store
	q      *queue.Queue
	tokens *auth.TokenManager
}

func NewServer(cfg *config.Config, st *store.Store, q *queue.Queue) *Server {
	return &Server{
		cfg:    cfg,
		st:     st,
		q:      q,
		tokens: auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL),
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(s.cors)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })

	r.Route("/api", func(r chi.Router) {
		// Public
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/refresh", s.handleRefresh)

		// Authenticated
		r.Group(func(r chi.Router) {
			r.Use(s.authRequired)

			r.Get("/auth/me", s.handleMe)
			r.Get("/dashboard", s.handleDashboard)

			r.Get("/engagements", s.handleListEngagements)
			r.Post("/engagements", s.requireRole("operator", s.handleCreateEngagement))
			r.Get("/engagements/{id}", s.handleGetEngagement)
			r.Put("/engagements/{id}", s.requireRole("operator", s.handleUpdateEngagement))
			r.Delete("/engagements/{id}", s.requireRole("operator", s.handleDeleteEngagement))
			r.Post("/engagements/{id}/status", s.requireRole("operator", s.handleSetEngagementStatus))
			r.Get("/engagements/{id}/scope", s.handleListScope)
			r.Post("/engagements/{id}/scope", s.requireRole("operator", s.handleAddScope))
			r.Delete("/engagements/{id}/scope/{ruleID}", s.requireRole("operator", s.handleDeleteScope))
			r.Get("/engagements/{id}/targets", s.handleListTargets)
			r.Post("/engagements/{id}/targets", s.requireRole("operator", s.handleCreateTargets))
			r.Post("/engagements/{id}/targets/import", s.requireRole("operator", s.handleImportTargetsFile))
			r.Get("/engagements/{id}/campaigns", s.handleListCampaigns)
			r.Post("/engagements/{id}/campaigns", s.requireRole("operator", s.handleCreateCampaign))

			r.Get("/email-templates", s.handleListEmailTemplates)
			r.Post("/email-templates", s.requireRole("operator", s.handleCreateEmailTemplate))
			r.Put("/email-templates/{id}", s.requireRole("operator", s.handleUpdateEmailTemplate))
			r.Delete("/email-templates/{id}", s.requireRole("operator", s.handleDeleteEmailTemplate))
			r.Post("/email-templates/{id}/duplicate", s.requireRole("operator", s.handleDuplicateEmailTemplate))

			r.Get("/landing-pages", s.handleListLandingPages)
			r.Post("/landing-pages", s.requireRole("operator", s.handleCreateLandingPage))
			r.Put("/landing-pages/{id}", s.requireRole("operator", s.handleUpdateLandingPage))
			r.Delete("/landing-pages/{id}", s.requireRole("operator", s.handleDeleteLandingPage))
			r.Post("/landing-pages/{id}/duplicate", s.requireRole("operator", s.handleDuplicateLandingPage))
			r.Post("/landing-pages/import", s.requireRole("operator", s.handleImportLandingPage))

			r.Get("/sending-profiles", s.handleListSendingProfiles)
			r.Post("/sending-profiles", s.requireRole("operator", s.handleCreateSendingProfile))
			r.Put("/sending-profiles/{id}", s.requireRole("operator", s.handleUpdateSendingProfile))
			r.Delete("/sending-profiles/{id}", s.requireRole("operator", s.handleDeleteSendingProfile))
			r.Post("/sending-profiles/{id}/duplicate", s.requireRole("operator", s.handleDuplicateSendingProfile))
			r.Post("/sending-profiles/{id}/dkim", s.requireRole("operator", s.handleGenerateDKIM))

			r.Post("/campaigns/{id}/launch", s.requireRole("operator", s.handleLaunchCampaign))
			r.Post("/campaigns/{id}/stop", s.requireRole("operator", s.handleStopCampaign))
			r.Delete("/campaigns/{id}", s.requireRole("operator", s.handleDeleteCampaign))
			r.Get("/campaigns/{id}/report", s.handleCampaignReport)
			r.Get("/campaigns/{id}/timeline", s.handleCampaignTimeline)
			r.Get("/campaigns/{id}/variants", s.handleListVariants)
			r.Post("/campaigns/{id}/variants", s.requireRole("operator", s.handleAddVariant))

			r.Get("/engagements/{id}/risk", s.handleRiskScores)

			r.Get("/training-modules", s.handleListTraining)
			r.Post("/training-modules", s.requireRole("operator", s.handleCreateTraining))
			r.Put("/training-modules/{id}", s.requireRole("operator", s.handleUpdateTraining))
			r.Delete("/training-modules/{id}", s.requireRole("operator", s.handleDeleteTraining))
			r.Get("/training-assignments", s.handleTrainingAssignments)

			r.Get("/api-keys", s.handleListAPIKeys)
			r.Post("/api-keys", s.requireRole("admin", s.handleCreateAPIKey))
			r.Delete("/api-keys/{id}", s.requireRole("admin", s.handleRevokeAPIKey))

			r.Get("/webhooks", s.handleListWebhooks)
			r.Post("/webhooks", s.requireRole("operator", s.handleCreateWebhook))
			r.Delete("/webhooks/{id}", s.requireRole("operator", s.handleDeleteWebhook))

			r.Post("/deliverability/check", s.requireRole("operator", s.handleDeliverabilityCheck))
			r.Post("/deliverability/seed-check", s.requireRole("operator", s.handleSeedCheck))
			r.Post("/deliverability/gateway-check", s.requireRole("operator", s.handleGatewayCheck))

			r.Get("/audit-log", s.handleAuditLog)
		})
	})

	// Serve built SPA if present, else a small placeholder.
	s.mountSPA(r)
	return r
}

func (s *Server) cors(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range s.cfg.CORSOrigins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) mountSPA(r chi.Router) {
	dist := s.cfg.WebDist
	if st, err := os.Stat(filepath.Join(dist, "index.html")); err == nil && !st.IsDir() {
		fs := http.FileServer(http.Dir(dist))
		r.Handle("/assets/*", fs)
		r.NotFound(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, "/api") {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			http.ServeFile(w, req, filepath.Join(dist, "index.html"))
		})
		return
	}
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api") {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(placeholderHTML))
	})
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

const placeholderHTML = `<!doctype html><html><head><meta charset="utf-8">
<title>PhishForge</title><style>body{font-family:system-ui;background:#0b1220;color:#e5e7eb;margin:0}
.wrap{max-width:640px;margin:12vh auto;padding:0 24px}code{background:#1e293b;padding:2px 6px;border-radius:4px}
a{color:#60a5fa}</style></head><body><div class="wrap">
<h1>🎣 PhishForge API</h1>
<p>The admin API is running. The frontend build was not found at <code>WEB_DIST</code>.</p>
<p>Health: <a href="/healthz">/healthz</a> · API base: <code>/api</code></p>
<p>Advanced phishing simulation &amp; security awareness platform.</p>
</div></body></html>`
