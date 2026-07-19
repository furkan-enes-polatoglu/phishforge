package deliverability

import (
	"regexp"
	"strings"
)

// ContentAnalysis scores an email subject+body against well-known Bayesian
// spam-filter heuristics: trigger phrases, ALL-CAPS shouting, link shorteners
// (heavily distrusted by gateways since they hide the real destination), and an
// image-to-text ratio (image-only emails are a classic spam/phishing signature).
type ContentAnalysis struct {
	TriggerWordsFound []string `json:"trigger_words_found,omitempty"`
	ShortenersFound   []string `json:"shorteners_found,omitempty"`
	AllCapsWords      int      `json:"all_caps_words"`
	ImageCount        int      `json:"image_count"`
	TextLength        int      `json:"text_length"`
	ImageOnlyWarning  bool     `json:"image_only_warning"`
	HeuristicPenalty  int      `json:"heuristic_penalty"` // 0-100, higher = more likely flagged
}

// spamTriggerWords are classic Bayesian-filter trigger phrases. Presence isn't
// disqualifying (some are unavoidable in a realistic phishing pretext) but each
// one nudges a real gateway's content score up — worth knowing before you send.
var spamTriggerWords = []string{
	"free", "act now", "click here", "verify your account", "winner", "congratulations",
	"urgent", "guarantee", "no cost", "risk-free", "limited time", "act immediately",
	"suspended", "unusual activity", "confirm your identity", "wire transfer",
	"bedava", "hemen tıklayın", "hesabınızı doğrulayın", "tebrikler", "acil",
	"kazandınız", "son fırsat", "hesabınız askıya alındı", "şifrenizi güncelleyin",
}

// linkShorteners are commonly abused to hide destination URLs; many gateways
// flag or block them outright regardless of the destination's actual reputation.
var linkShorteners = []string{
	"bit.ly", "tinyurl.com", "goo.gl", "t.co", "ow.ly", "is.gd", "buff.ly", "rebrand.ly", "cutt.ly",
}

var reWord = regexp.MustCompile(`[A-Za-zÇĞİÖŞÜçğıöşü]{4,}`)
var reHref = regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)
var reTag = regexp.MustCompile(`<[^>]+>`)

// AnalyzeContent inspects a subject line and HTML body for spam-filter
// heuristics and returns a 0-100 penalty (0 = clean) plus the specific hits so
// operators can fix them before sending.
func AnalyzeContent(subject, html string) ContentAnalysis {
	res := ContentAnalysis{}
	lowerSubject := strings.ToLower(subject)
	lowerHTML := strings.ToLower(html)

	seen := map[string]bool{}
	for _, w := range spamTriggerWords {
		if strings.Contains(lowerSubject, w) || strings.Contains(lowerHTML, w) {
			if !seen[w] {
				res.TriggerWordsFound = append(res.TriggerWordsFound, w)
				seen[w] = true
			}
		}
	}
	res.HeuristicPenalty += len(res.TriggerWordsFound) * 6

	for _, m := range reHref.FindAllStringSubmatch(html, -1) {
		url := strings.ToLower(m[1])
		for _, sh := range linkShorteners {
			if strings.Contains(url, sh) {
				res.ShortenersFound = append(res.ShortenersFound, sh)
			}
		}
	}
	if len(res.ShortenersFound) > 0 {
		res.HeuristicPenalty += 15
	}

	for _, w := range reWord.FindAllString(subject, -1) {
		if w == strings.ToUpper(w) {
			res.AllCapsWords++
		}
	}
	res.HeuristicPenalty += res.AllCapsWords * 8

	res.ImageCount = strings.Count(lowerHTML, "<img")
	visibleText := strings.TrimSpace(reTag.ReplaceAllString(html, " "))
	res.TextLength = len(visibleText)
	if res.ImageCount > 0 && res.TextLength < 40 {
		res.ImageOnlyWarning = true
		res.HeuristicPenalty += 20
	}

	if res.HeuristicPenalty > 100 {
		res.HeuristicPenalty = 100
	}
	return res
}
