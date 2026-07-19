package deliverability

import (
	"regexp"
	"strings"
)

var (
	reImgTag      = regexp.MustCompile(`(?i)<img[^>]*>`)
	reHasAlt      = regexp.MustCompile(`(?i)\balt\s*=`)
	reInlineStyle = regexp.MustCompile(`(?i)<link[^>]+stylesheet`)
)

// imgMissingAlt reports whether any <img> tag lacks an alt attribute. Go's RE2
// engine has no lookahead, so we scan matched tags individually.
func imgMissingAlt(html string) bool {
	for _, tag := range reImgTag.FindAllString(html, -1) {
		if !reHasAlt.MatchString(tag) {
			return true
		}
	}
	return false
}

// LintHTML returns non-fatal hints about markup that often hurts rendering or
// deliverability (broken images, external stylesheets, missing unsubscribe text).
func LintHTML(html string) []string {
	var hints []string
	if strings.TrimSpace(html) == "" {
		return []string{"e-posta gövdesi boş"}
	}
	if imgMissingAlt(html) {
		hints = append(hints, "bazı <img> etiketlerinde alt metni yok (erişilebilirlik + spam sezgiselleri)")
	}
	if reInlineStyle.MatchString(html) {
		hints = append(hints, "harici stylesheet <link> bulundu; birçok istemci bunu siler — satır-içi (inline) stil tercih edin")
	}
	if !strings.Contains(strings.ToLower(html), "unsubscribe") && !strings.Contains(strings.ToLower(html), "abonelik") {
		hints = append(hints, "abonelikten çıkma/altbilgi metni yok; gerçekçilik ve politika için makul bir altbilgi düşünün")
	}
	if strings.Count(html, "!") > 8 {
		hints = append(hints, "çok fazla ünlem işareti var; spam-tonu sezgiselleri tetiklenebilir")
	}
	return hints
}
