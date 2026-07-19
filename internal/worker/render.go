package worker

import (
	"bytes"
	"html/template"
	"net/url"
	"strings"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

// TemplateData is exposed to email templates via merge-tags, e.g. {{.FirstName}}.
type TemplateData struct {
	FirstName  string
	LastName   string
	Email      string
	Position   string
	TrackURL   string // click-through landing URL
	TrackPixel string // open tracking pixel URL
	ReportURL  string // "report phishing" URL
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

// InjectTracking appends an open-tracking pixel and ensures a click link exists.
// If the template contains {{.TrackURL}} the operator placed the link; otherwise
// we do not rewrite arbitrary anchors (kept explicit for auditability).
func InjectTracking(html string, data TemplateData) string {
	pixel := `<img src="` + template.HTMLEscapeString(data.TrackPixel) + `" width="1" height="1" alt="" style="display:none">`
	if strings.Contains(strings.ToLower(html), "</body>") {
		return strings.Replace(html, "</body>", pixel+"</body>", 1)
	}
	return html + pixel
}

func buildData(t models.Target, phishBase, rid string) TemplateData {
	return TemplateData{
		FirstName:  t.FirstName,
		LastName:   t.LastName,
		Email:      t.Email,
		Position:   t.Position,
		TrackURL:   phishBase + "/l/" + url.PathEscape(rid),
		TrackPixel: phishBase + "/t/" + url.PathEscape(rid),
		ReportURL:  phishBase + "/r/" + url.PathEscape(rid),
	}
}
