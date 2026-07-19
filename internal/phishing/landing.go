package phishing

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

// landingData is exposed to landing-page templates.
type landingData struct {
	FirstName string
	LastName  string
	Email     string
	SubmitURL string // where the form should POST (records a submit event)
}

// renderLanding substitutes merge-tags. The form action must point at SubmitURL
// so submissions are recorded (values are discarded server-side).
func renderLanding(html string, t models.Target, rid, phishBase string) string {
	data := landingData{
		FirstName: t.FirstName,
		LastName:  t.LastName,
		Email:     t.Email,
		SubmitURL: phishBase + "/l/" + url.PathEscape(rid),
	}
	tpl, err := template.New("landing").Parse(html)
	if err != nil {
		return defaultLanding(data)
	}
	var sb strings.Builder
	if err := tpl.Execute(&sb, data); err != nil {
		return defaultLanding(data)
	}
	out := sb.String()
	if strings.TrimSpace(out) == "" {
		return defaultLanding(data)
	}
	return out
}

func defaultLanding(d landingData) string {
	return `<!doctype html><meta charset="utf-8"><title>Sign in</title>
<div style="font-family:sans-serif;max-width:360px;margin:80px auto">
<h3>Sign in to continue</h3>
<form method="post" action="` + template.HTMLEscapeString(d.SubmitURL) + `">
<p><input name="username" placeholder="Email" style="width:100%;padding:8px" value="` + template.HTMLEscapeString(d.Email) + `"></p>
<p><input name="password" type="password" placeholder="Password" style="width:100%;padding:8px"></p>
<p><button type="submit" style="padding:8px 16px">Sign in</button></p>
</form></div>`
}

func writeAwareness(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8"><title>Security Awareness</title>
<div style="font-family:sans-serif;max-width:560px;margin:70px auto;line-height:1.5">
<h2>&#9888;&#65039; This was a simulated phishing test</h2>
<p>This message was part of an <strong>authorized security awareness exercise</strong>.
No credentials were captured. In a real attack, submitting this form could have
compromised your account.</p>
<h3>How to spot phishing next time</h3>
<ul>
<li>Check the sender address and hover links before clicking.</li>
<li>Be wary of urgency and unexpected login prompts.</li>
<li>Report suspicious email to your security team.</li>
</ul>
</div>`))
}
