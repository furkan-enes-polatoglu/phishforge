package phishing

import (
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleTraining renders an assigned awareness-training module and marks the
// assignment completed (viewing the module counts as completion in this MVP).
func (s *Server) handleTraining(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	m, err := s.st.TrainingModuleByToken(r.Context(), token)
	if err != nil {
		writeAwareness(w)
		return
	}
	_ = s.st.CompleteTraining(r.Context(), token)

	body := m.HTML
	if body == "" {
		body = defaultTraining
	}
	page := `<!doctype html><html><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + template.HTMLEscapeString(m.Name) + `</title>
<style>body{font-family:system-ui,-apple-system,Segoe UI,Roboto,sans-serif;background:#f5f7fb;color:#1e293b;margin:0}
.wrap{max-width:720px;margin:0 auto;padding:40px 24px}.card{background:#fff;border:1px solid #e2e8f0;border-radius:14px;padding:32px;box-shadow:0 1px 3px rgba(15,23,42,.06)}
h1{font-size:22px}.done{display:inline-block;margin-top:20px;background:#dcfce7;color:#166534;padding:6px 12px;border-radius:999px;font-size:13px;font-weight:600}</style>
</head><body><div class="wrap"><div class="card">` + body + `
<div class="done">✓ Training completed</div></div></div></body></html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page))
}

const defaultTraining = `<h1>You clicked a simulated phishing link</h1>
<p>This was part of an <strong>authorized security awareness exercise</strong>. No
credentials were captured. Here is how to stay safe:</p>
<ul>
<li><strong>Check the sender</strong> — hover the address; look for lookalike domains.</li>
<li><strong>Hover links</strong> before clicking; the visible text can lie.</li>
<li><strong>Distrust urgency</strong> — "act now or else" is a classic pressure tactic.</li>
<li><strong>Never enter credentials</strong> from an emailed link. Navigate to the site directly.</li>
<li><strong>Report it</strong> — forward suspicious mail to your security team.</li>
</ul>
<p>Thanks for taking a moment to learn. Your awareness protects the whole organization.</p>`
