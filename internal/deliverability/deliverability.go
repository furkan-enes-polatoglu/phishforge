// Package deliverability provides legitimate pre-send email health checks:
// SPF/DKIM/DMARC record validation, RBL/blocklist lookups, an optional
// SpamAssassin score, and HTML lint hints. The goal is to make authorized test
// mail *reach the inbox* through correct email infrastructure and coordinated
// allowlisting — NOT to evade or deceive spam filters.
package deliverability

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CheckResult is the outcome of a domain deliverability check.
type CheckResult struct {
	Domain    string        `json:"domain"`
	SPF       RecordCheck   `json:"spf"`
	DMARC     RecordCheck   `json:"dmarc"`
	DKIM      RecordCheck   `json:"dkim"`
	RBL       []RBLResult   `json:"rbl"`
	SpamScore *float64      `json:"spam_score,omitempty"`
	HTMLLint  []string      `json:"html_lint,omitempty"`
	Advice    []string      `json:"advice"`
}

type RecordCheck struct {
	Found  bool   `json:"found"`
	Value  string `json:"value,omitempty"`
	Status string `json:"status"` // ok | warn | missing
	Detail string `json:"detail,omitempty"`
}

type RBLResult struct {
	List   string `json:"list"`
	Listed bool   `json:"listed"`
}

var resolver = &net.Resolver{}

func lookupTXT(ctx context.Context, name string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return resolver.LookupTXT(ctx, name)
}

// CheckDomain runs SPF/DMARC (and a best-effort DKIM selector probe) checks.
func CheckDomain(ctx context.Context, domain, dkimSelector string) CheckResult {
	domain = strings.ToLower(strings.TrimSpace(domain))
	res := CheckResult{Domain: domain}

	// SPF
	if txts, err := lookupTXT(ctx, domain); err == nil {
		for _, t := range txts {
			if strings.HasPrefix(strings.ToLower(t), "v=spf1") {
				res.SPF = RecordCheck{Found: true, Value: t, Status: "ok"}
				if strings.Contains(t, "+all") {
					res.SPF.Status = "warn"
					res.SPF.Detail = "+all is dangerously permissive"
				}
				break
			}
		}
	}
	if !res.SPF.Found {
		res.SPF.Status = "missing"
		res.Advice = append(res.Advice, "No SPF record: add v=spf1 including your sending host, then -all.")
	}

	// DMARC
	if txts, err := lookupTXT(ctx, "_dmarc."+domain); err == nil {
		for _, t := range txts {
			if strings.HasPrefix(strings.ToLower(t), "v=dmarc1") {
				res.DMARC = RecordCheck{Found: true, Value: t, Status: "ok"}
				break
			}
		}
	}
	if !res.DMARC.Found {
		res.DMARC.Status = "missing"
		res.Advice = append(res.Advice, "No DMARC record: publish _dmarc TXT (start with p=none for monitoring).")
	}

	// DKIM (probe a selector if provided)
	if dkimSelector != "" {
		name := dkimSelector + "._domainkey." + domain
		if txts, err := lookupTXT(ctx, name); err == nil && len(txts) > 0 {
			joined := strings.Join(txts, "")
			if strings.Contains(strings.ToLower(joined), "p=") {
				res.DKIM = RecordCheck{Found: true, Value: joined, Status: "ok"}
			}
		}
		if !res.DKIM.Found {
			res.DKIM.Status = "missing"
			res.Advice = append(res.Advice, fmt.Sprintf("DKIM selector %q not found; publish the public key TXT.", dkimSelector))
		}
	} else {
		res.DKIM.Status = "warn"
		res.DKIM.Detail = "no selector provided to probe"
	}

	if len(res.Advice) == 0 {
		res.Advice = append(res.Advice, "Core authentication records look present. Coordinate an allowlist with the client's mail gateway for the engagement.")
	}
	return res
}

// Common DNSBLs. RBL checks reverse the sender IP and query each zone.
var defaultRBLs = []string{
	"zen.spamhaus.org",
	"bl.spamcop.net",
	"b.barracudacentral.org",
}

// CheckIPReputation looks up an IPv4 address against common DNSBLs.
func CheckIPReputation(ctx context.Context, ip string) []RBLResult {
	parsed := net.ParseIP(ip).To4()
	out := []RBLResult{}
	if parsed == nil {
		return out
	}
	rev := fmt.Sprintf("%d.%d.%d.%d", parsed[3], parsed[2], parsed[1], parsed[0])
	for _, zone := range defaultRBLs {
		q := rev + "." + zone
		c, cancel := context.WithTimeout(ctx, 4*time.Second)
		addrs, _ := resolver.LookupHost(c, q)
		cancel()
		out = append(out, RBLResult{List: zone, Listed: len(addrs) > 0})
	}
	return out
}

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
// deliverability (broken images, external stylesheets, missing text part hint).
func LintHTML(html string) []string {
	var hints []string
	if strings.TrimSpace(html) == "" {
		return []string{"empty HTML body"}
	}
	if imgMissingAlt(html) {
		hints = append(hints, "some <img> tags lack alt text (accessibility + spam heuristics)")
	}
	if reInlineStyle.MatchString(html) {
		hints = append(hints, "external stylesheet <link> found; many clients strip it — prefer inline styles")
	}
	if !strings.Contains(strings.ToLower(html), "unsubscribe") {
		hints = append(hints, "no unsubscribe/footer text; consider a plausible footer for realism and policy")
	}
	if strings.Count(html, "!") > 8 {
		hints = append(hints, "many exclamation marks; spammy-tone heuristics may trigger")
	}
	return hints
}

// SpamScore contacts a SpamAssassin spamd instance (CHECK) and returns the score.
// If spamdAddr is empty or unreachable, returns (nil, error) and callers proceed.
func SpamScore(ctx context.Context, spamdAddr, rawMessage string) (*float64, error) {
	if spamdAddr == "" {
		return nil, fmt.Errorf("spamd disabled")
	}
	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", spamdAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	body := rawMessage
	req := fmt.Sprintf("CHECK SPAMC/1.2\r\nContent-length: %d\r\n\r\n%s", len(body), body)
	if _, err := conn.Write([]byte(req)); err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(conn)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "Spam:") {
			// e.g. "Spam: False ; 2.3 / 5.0"
			parts := strings.Split(line, ";")
			if len(parts) == 2 {
				scoreStr := strings.TrimSpace(strings.Split(parts[1], "/")[0])
				if v, err := strconv.ParseFloat(scoreStr, 64); err == nil {
					return &v, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("no score in spamd response")
}
