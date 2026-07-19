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
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/auth"
	"github.com/furkan-enes-polatoglu/phishforge/internal/config"
	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/furkan-enes-polatoglu/phishforge/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/skip2/go-qrcode"
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
	r.Get("/q/{rid}", s.handleScan)       // QR code scanned (quishing simulation)
	r.Get("/qr/{rid}", s.handleQRImage)   // PNG QR code embeddable in email
	r.Get("/a/{rid}", s.handleAttachment) // simulated malicious attachment opened
	r.Get("/training/{token}", s.handleTraining)
	r.Post("/webhooks/mailgun", s.handleMailgunWebhook)
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

// resolve validates the RID signature, loads the campaign target, and checks
// that the campaign/engagement is still allowed to serve tracking content —
// stopping a campaign or closing/expiring its engagement immediately cuts off
// access to an already-sent link, it doesn't keep working forever.
func (s *Server) resolve(ctx context.Context, rid string) (*models.CampaignTarget, *models.Campaign, *models.Target, bool) {
	if _, err := auth.VerifyRID(s.cfg.RIDSecret, rid); err != nil {
		return nil, nil, nil, false
	}
	ct, c, t, err := s.st.CampaignTargetByRID(ctx, rid)
	if err != nil {
		return nil, nil, nil, false
	}
	eng, err := s.st.GetEngagementByID(ctx, c.EngagementID)
	if err != nil {
		return nil, nil, nil, false
	}
	if !campaignServable(*eng, *c, time.Now()) {
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
	s.notifyEvent(c.ID, "click", t.Email)
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
	ct, c, t, ok := s.resolve(r.Context(), rid)
	if !ok {
		http.NotFound(w, r)
		return
	}
	lp, _ := s.st.LandingPageByCampaign(r.Context(), c.ID)

	// Capture behavior is controlled per landing page (GoPhish-parity, opt-in):
	//   - default: store only the fact of submission (no field data)
	//   - capture_meta: also store which field NAMES were filled (no values)
	//   - capture_submitted_data: store field values (password-like fields redacted
	//       unless capture_passwords is also on)
	//   - capture_passwords: also store password-like values (sensitive!)
	meta := "{}"
	if lp != nil && (lp.CaptureMeta || lp.CaptureSubmittedData) {
		_ = r.ParseForm()
		out := map[string]any{}
		if lp.CaptureMeta {
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
			out["fields_filled"] = names
		}
		if lp.CaptureSubmittedData {
			submitted := map[string]string{}
			for k, vs := range r.PostForm {
				if len(vs) == 0 {
					continue
				}
				val := vs[0]
				if isPasswordField(k) && !lp.CapturePasswords {
					val = "[redacted]"
				}
				submitted[k] = val
			}
			out["submitted"] = submitted
			out["captured_passwords"] = lp.CapturePasswords
		}
		b, _ := json.Marshal(out)
		meta = string(b)
	}
	_ = s.st.RecordEvent(r.Context(), &models.Event{
		CampaignTargetID: ct.ID, Type: models.EventSubmit,
		IP: clientIP(r), UserAgent: r.UserAgent(), Meta: meta,
	})
	s.notifyEvent(c.ID, "submit", t.Email)

	// Explicit landing redirect wins; otherwise auto-assign a training module and
	// redirect there so the awareness loop closes.
	if lp != nil && lp.RedirectURL != "" {
		http.Redirect(w, r, lp.RedirectURL, http.StatusFound)
		return
	}
	if trainingURL := s.autoAssignTraining(r.Context(), c.ID, t.ID); trainingURL != "" {
		http.Redirect(w, r, trainingURL, http.StatusFound)
		return
	}
	writeAwareness(w)
}

// handleScan records a QR-code scan (quishing simulation) then redirects to the
// normal landing/click flow, so the funnel distinguishes "scanned a QR" from
// "clicked a link" while still measuring the full click-through afterwards.
func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	rid := chi.URLParam(r, "rid")
	ct, c, t, ok := s.resolve(r.Context(), rid)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_ = s.st.RecordEvent(r.Context(), &models.Event{
		CampaignTargetID: ct.ID, Type: models.EventScan,
		IP: clientIP(r), UserAgent: r.UserAgent(),
	})
	s.notifyEvent(c.ID, "scan", t.Email)
	http.Redirect(w, r, "/l/"+rid, http.StatusFound)
}

// handleQRImage renders a PNG QR code encoding the scan-tracked URL, meant to be
// embedded in an email via <img src="{{.QRCodeURL}}">.
func (s *Server) handleQRImage(w http.ResponseWriter, r *http.Request) {
	rid := chi.URLParam(r, "rid")
	if _, _, _, ok := s.resolve(r.Context(), rid); !ok {
		http.NotFound(w, r)
		return
	}
	png, err := qrcode.Encode(s.cfg.PhishBaseURL+"/q/"+rid, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "qr generation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

// handleAttachment simulates a malicious-attachment open (e.g. a macro-enabled
// document or a booby-trapped PDF in a real attack). It only records the event
// and hands the target to awareness training — no file is ever executed or
// delivered.
func (s *Server) handleAttachment(w http.ResponseWriter, r *http.Request) {
	rid := chi.URLParam(r, "rid")
	ct, c, t, ok := s.resolve(r.Context(), rid)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_ = s.st.RecordEvent(r.Context(), &models.Event{
		CampaignTargetID: ct.ID, Type: models.EventAttachmentOpen,
		IP: clientIP(r), UserAgent: r.UserAgent(),
	})
	s.notifyEvent(c.ID, "attachment_open", t.Email)
	if trainingURL := s.autoAssignTraining(r.Context(), c.ID, t.ID); trainingURL != "" {
		http.Redirect(w, r, trainingURL, http.StatusFound)
		return
	}
	writeAwareness(w)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	rid := chi.URLParam(r, "rid")
	ct, c, t, ok := s.resolve(r.Context(), rid)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_ = s.st.RecordEvent(r.Context(), &models.Event{
		CampaignTargetID: ct.ID, Type: models.EventReport,
		IP: clientIP(r), UserAgent: r.UserAgent(),
	})
	s.notifyEvent(c.ID, "report", t.Email)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8"><title>Thank you</title>
<div style="font-family:sans-serif;max-width:520px;margin:80px auto;text-align:center">
<h2>&#9989; Thanks for reporting</h2>
<p>You correctly identified and reported a simulated phishing email. Great instinct!</p>
</div>`))
}
