package worker

import (
	"bytes"
	"html/template"
	"net/url"
	"regexp"
	"strings"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

// TemplateData is exposed to email templates via merge-tags, e.g. {{.FirstName}}.
type TemplateData struct {
	FirstName     string
	LastName      string
	Email         string
	Position      string
	Department    string
	TrackURL      string // click-through landing URL
	TrackPixel    string // open tracking pixel URL
	ReportURL     string // "report phishing" URL
	QRCodeURL     string // PNG image of a QR code encoding a scan-tracked link (quishing simulation)
	AttachmentURL string // simulated malicious-attachment link (records an "attachment_open" event)
}

// Render substitutes merge-tags in an email body. Uses html/template so target
// attributes are HTML-escaped (defense against broken markup / injection).
func Render(body string, data TemplateData) (string, error) {
	tpl, err := template.New("email").Parse(body)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// InjectTracking appends an open-tracking pixel before </body> (or at the end).
func InjectTracking(html string, data TemplateData) string {
	pixel := `<img src="` + template.HTMLEscapeString(data.TrackPixel) + `" width="1" height="1" alt="" style="display:none">`
	if strings.Contains(strings.ToLower(html), "</body>") {
		return strings.Replace(html, "</body>", pixel+"</body>", 1)
	}
	return html + pixel
}

var reAnchorHref = regexp.MustCompile(`(?i)(<a\b[^>]*\shref=)("|')(.*?)("|')`)

// RewriteLinks replaces the href of every anchor with the tracked click URL so
// clicks are recorded regardless of which link the target follows. The original
// href is dropped (the tracked URL leads to the simulation landing page).
func RewriteLinks(html, trackURL string) string {
	repl := `${1}"` + template.HTMLEscapeString(trackURL) + `"`
	return reAnchorHref.ReplaceAllString(html, repl)
}

// effectiveLandingBase picks the base URL used to build tracking/landing links
// for a send, in priority order:
//  1. an explicit per-campaign override (like GoPhish's per-launch "URL" field)
//  2. the sending profile's own domain (the per-client-domain workflow — a
//     fresh domain bought per engagement, set once and reused by every
//     campaign that uses that profile)
//  3. the instance-wide default from config
func effectiveLandingBase(campaignOverride string, profile *models.SendingProfile, globalDefault string) string {
	if campaignOverride != "" {
		return strings.TrimRight(campaignOverride, "/")
	}
	if profile != nil && profile.LandingBaseURL != "" {
		return strings.TrimRight(profile.LandingBaseURL, "/")
	}
	return globalDefault
}

func buildData(t models.Target, phishBase, rid string) TemplateData {
	escRID := url.PathEscape(rid)
	return TemplateData{
		FirstName:     t.FirstName,
		LastName:      t.LastName,
		Email:         t.Email,
		Position:      t.Position,
		Department:    t.Department,
		TrackURL:      phishBase + "/l/" + escRID,
		TrackPixel:    phishBase + "/t/" + escRID,
		ReportURL:     phishBase + "/r/" + escRID,
		QRCodeURL:     phishBase + "/qr/" + escRID,
		AttachmentURL: phishBase + "/a/" + escRID,
	}
}
