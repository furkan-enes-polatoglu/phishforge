// Package phishing is the target-facing tracking server. It records funnel events
// (open/click/submit/report) and serves landing pages. Per the product's data
// minimization guardrail, submitted form VALUES are never stored — only the fact
// of submission and (optionally) the set of field NAMES that were filled.
package phishing

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/furkan-enes-polatoglu/phishforge/internal/auth"
	"github.com/furkan-enes-polatoglu/phishforge/internal/config"
	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/furkan-enes-polatoglu/phishforge/internal/store"
	"github.com/go-chi/chi/v5"
)

// 1x1 transparent GIF.
var pixelGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00,
	0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02,
	0x44, 0x01, 0x00, 0x3b,
}

type Server struct {
	cfg *config.Config
	st  *store.Store
}

func New(cfg *config.Config, st *store.Store) *Server {
	return &Server{cfg: cfg, st: st}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	r.Get("/t/{rid}", s.handleOpen)
	r.Get("/l/{rid}", s.handleClick)
	r.Post("/l/{rid}", s.handleSubmit)
	r.Get("/r/{rid}", s.handleReport)
	return r
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// resolve validates the RID signature then loads the campaign target.
func (s *Server) resolve(ctx context.Context, rid string) (*models.CampaignTarget, *models.Campaign, *models.Target, bool) {
	if _, err := auth.VerifyRID(s.cfg.RIDSecret, rid); err != nil {
		return nil, nil, nil, false
	}
	ct, c, t, err := s.st.CampaignTargetByRID(ctx, rid)
	if err != nil {
		return nil, nil, nil, false
	}
	return ct, c, t, true
}

func (s *Server) handleOpen(w http.ResponseWriter, r *http.Request) {
	rid := chi.URLParam(r, "rid")
	if ct, _, _, ok := s.resolve(r.Context(), rid); ok {
		_ = s.st.RecordEvent(r.Context(), &models.Event{
			CampaignTargetID: ct.ID, Type: models.EventOpen,
			IP: clientIP(r), UserAgent: r.UserAgent(),
		})
	}
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	_, _ = w.Write(pixelGIF)
}

func (s *Server) handleClick(w http.ResponseWriter, r *http.Request) {
	rid := chi.URLParam(r, "rid")
	ct, c, t, ok := s.resolve(r.Context(), rid)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_ = s.st.RecordEvent(r.Context(), &models.Event{
		CampaignTargetID: ct.ID, Type: models.EventClick,
		IP: clientIP(r), UserAgent: r.UserAgent(),
	})
	lp, err := s.st.LandingPageByCampaign(r.Context(), c.ID)
	if err != nil {
		writeAwareness(w)
		return
	}
	html := renderLanding(lp.HTML, *t, rid, s.cfg.PhishBaseURL)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	rid := chi.URLParam(r, "rid")
	ct, c, _, ok := s.resolve(r.Context(), rid)
	if !ok {
		http.NotFound(w, r)
		return
	}
	lp, _ := s.st.LandingPageByCampaign(r.Context(), c.ID)

	// DATA MINIMIZATION: never store submitted values. Optionally capture the set
	// of field NAMES that were provided (excluding any value).
	meta := "{}"
	if lp != nil && lp.CaptureMeta {
		_ = r.ParseForm()
		var names []string
		for k, vs := range r.PostForm {
			for _, v := range vs {
				if strings.TrimSpace(v) != "" {
					names = append(names, k)
					break
				}
			}
		}
		sort.Strings(names)
		b, _ := json.Marshal(map[string]any{"fields_filled": names})
		meta = string(b)
	}
	_ = s.st.RecordEvent(r.Context(), &models.Event{
		CampaignTargetID: ct.ID, Type: models.EventSubmit,
		IP: clientIP(r), UserAgent: r.UserAgent(), Meta: meta,
	})

	if lp != nil && lp.RedirectURL != "" {
		http.Redirect(w, r, lp.RedirectURL, http.StatusFound)
		return
	}
	writeAwareness(w)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	rid := chi.URLParam(r, "rid")
	if ct, _, _, ok := s.resolve(r.Context(), rid); ok {
		_ = s.st.RecordEvent(r.Context(), &models.Event{
			CampaignTargetID: ct.ID, Type: models.EventReport,
			IP: clientIP(r), UserAgent: r.UserAgent(),
		})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8"><title>Thank you</title>
<div style="font-family:sans-serif;max-width:520px;margin:80px auto;text-align:center">
<h2>&#9989; Thanks for reporting</h2>
<p>You correctly identified and reported a simulated phishing email. Great instinct!</p>
</div>`))
}
