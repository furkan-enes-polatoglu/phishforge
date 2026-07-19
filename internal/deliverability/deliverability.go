// Package deliverability provides legitimate pre-send email health checks:
// SPF/DKIM/DMARC record validation and alignment analysis, PTR/FCrDNS,
// MTA-STS/TLS-RPT, blocklist lookups, an optional SpamAssassin score, and
// content/spam-trigger analysis — all aggregated into one delivery confidence
// score. The goal is to make authorized test mail *reach the inbox* through
// correct email infrastructure and coordinated allowlisting — NOT to evade or
// deceive spam filters.
package deliverability

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// CheckResult is the outcome of a full domain + sending-IP deliverability check.
type CheckResult struct {
	Domain    string       `json:"domain"`
	SPF       RecordCheck  `json:"spf"`
	DMARC     RecordCheck  `json:"dmarc"`
	DMARCPolicy *DMARCPolicy `json:"dmarc_policy,omitempty"`
	DKIM      RecordCheck  `json:"dkim"`
	PTR       *PTRResult   `json:"ptr,omitempty"`
	MTASTS    *MTASTSResult `json:"mta_sts,omitempty"`
	TLSRPT    bool         `json:"tls_rpt"`
	RBL       []RBLResult  `json:"rbl"`
	SpamScore *float64     `json:"spam_score,omitempty"`
	HTMLLint  []string     `json:"html_lint,omitempty"`
	Content   *ContentAnalysis `json:"content,omitempty"`
	Advice    []string     `json:"advice"`
	Score     DeliveryScore `json:"score"`
}

type RecordCheck struct {
	Found  bool   `json:"found"`
	Value  string `json:"value,omitempty"`
	Status string `json:"status"` // ok | warn | missing
	Detail string `json:"detail,omitempty"`
}

// DMARCPolicy is the parsed set of DMARC tags relevant to deliverability and
// alignment (which determine whether SPF/DKIM passes actually count under DMARC).
type DMARCPolicy struct {
	Policy      string `json:"policy,omitempty"`     // p=
	SubPolicy   string `json:"sub_policy,omitempty"`  // sp=
	AlignSPF    string `json:"align_spf,omitempty"`   // aspf= (r=relaxed default, s=strict)
	AlignDKIM   string `json:"align_dkim,omitempty"`  // adkim=
	Percent     string `json:"percent,omitempty"`     // pct=
	RUA         bool   `json:"has_rua"`               // aggregate reports configured
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

// parseDMARCTags extracts DMARC tags (p=, sp=, aspf=, adkim=, pct=, rua=) from a
// raw "v=DMARC1; p=reject; ..." TXT value.
func parseDMARCTags(record string) DMARCPolicy {
	pol := DMARCPolicy{AlignSPF: "r", AlignDKIM: "r"} // relaxed is the DMARC default
	for _, part := range strings.Split(record, ";") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := strings.ToLower(strings.TrimSpace(kv[0])), strings.TrimSpace(kv[1])
		switch key {
		case "p":
			pol.Policy = val
		case "sp":
			pol.SubPolicy = val
		case "aspf":
			pol.AlignSPF = val
		case "adkim":
			pol.AlignDKIM = val
		case "pct":
			pol.Percent = val
		case "rua":
			pol.RUA = val != ""
		}
	}
	return pol
}

// CheckDomain runs SPF/DMARC (with alignment-mode parsing) and a best-effort
// DKIM selector probe for a sending domain.
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
					res.SPF.Detail = "+all aşırı derecede izin verici — spoofing'e açık kapı bırakır"
				} else if !strings.Contains(t, "-all") && !strings.Contains(t, "~all") {
					res.SPF.Status = "warn"
					res.SPF.Detail = "kayıt bir 'all' mekanizmasıyla bitmiyor; -all (fail) önerilir"
				}
				break
			}
		}
	}
	if !res.SPF.Found {
		res.SPF.Status = "missing"
		res.Advice = append(res.Advice, "SPF kaydı yok: gönderim sunucunuzu içeren v=spf1 ... -all TXT kaydı ekleyin.")
	}

	// DMARC
	if txts, err := lookupTXT(ctx, "_dmarc."+domain); err == nil {
		for _, t := range txts {
			if strings.HasPrefix(strings.ToLower(t), "v=dmarc1") {
				res.DMARC = RecordCheck{Found: true, Value: t, Status: "ok"}
				pol := parseDMARCTags(t)
				res.DMARCPolicy = &pol
				switch pol.Policy {
				case "", "none":
					res.DMARC.Status = "warn"
					res.DMARC.Detail = "p=none: DMARC yalnızca izleme modunda, sahte gönderimler reddedilmiyor"
				case "quarantine":
					res.DMARC.Detail = "p=quarantine: hizalanmayan mesajlar spam'e düşürülür"
				case "reject":
					res.DMARC.Detail = "p=reject: hizalanmayan mesajlar tamamen reddedilir — SPF/DKIM hizalamasının doğru olduğundan emin olun"
				}
				if pol.AlignSPF == "s" || pol.AlignDKIM == "s" {
					res.Advice = append(res.Advice, "DMARC katı (strict) hizalama kullanıyor: MAIL FROM / DKIM d= alan adının gönderen alan adıyla TAM olarak eşleşmesi gerekir.")
				}
				if !pol.RUA {
					res.Advice = append(res.Advice, "DMARC'ta rua= (toplu rapor adresi) tanımlı değil; raporlama olmadan sahtecilik girişimlerini göremezsiniz.")
				}
				break
			}
		}
	}
	if !res.DMARC.Found {
		res.DMARC.Status = "missing"
		res.Advice = append(res.Advice, "DMARC kaydı yok: _dmarc TXT kaydı yayınlayın (izleme için p=none ile başlayabilirsiniz).")
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
			res.Advice = append(res.Advice, fmt.Sprintf("DKIM seçici %q bulunamadı; genel anahtar TXT kaydını yayınlayın.", dkimSelector))
		}
	} else {
		res.DKIM.Status = "warn"
		res.DKIM.Detail = "sorgulanacak bir seçici (selector) verilmedi"
	}

	if len(res.Advice) == 0 {
		res.Advice = append(res.Advice, "Temel kimlik doğrulama kayıtları mevcut görünüyor. Angajman için müşterinin mail gateway'inde bir izin listesi koordine edin.")
	}
	return res
}

// defaultRBLs are well-established, widely-consulted DNSBLs. RBL checks reverse
// the sending IP and query each zone concurrently (each with its own timeout so
// one slow/unreachable list never blocks the others).
var defaultRBLs = []string{
	"zen.spamhaus.org",
	"bl.spamcop.net",
	"b.barracudacentral.org",
	"dnsbl.sorbs.net",
	"psbl.surriel.com",
}

// CheckIPReputation looks up an IPv4 address against common DNSBLs in parallel.
func CheckIPReputation(ctx context.Context, ip string) []RBLResult {
	parsed := net.ParseIP(ip).To4()
	if parsed == nil {
		return nil
	}
	rev := fmt.Sprintf("%d.%d.%d.%d", parsed[3], parsed[2], parsed[1], parsed[0])

	type slot struct {
		i   int
		res RBLResult
	}
	ch := make(chan slot, len(defaultRBLs))
	for i, zone := range defaultRBLs {
		go func(i int, zone string) {
			c, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()
			addrs, _ := resolver.LookupHost(c, rev+"."+zone)
			ch <- slot{i: i, res: RBLResult{List: zone, Listed: len(addrs) > 0}}
		}(i, zone)
	}
	out := make([]RBLResult, len(defaultRBLs))
	for range defaultRBLs {
		s := <-ch
		out[s.i] = s.res
	}
	return out
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
