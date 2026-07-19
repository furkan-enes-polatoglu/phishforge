package deliverability

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PTRResult is the outcome of a forward-confirmed reverse DNS (FCrDNS) check —
// one of the highest-impact, most commonly overlooked deliverability factors.
// Many corporate gateways (Cisco, Barracuda, Proofpoint among them) silently
// drop or heavily penalize mail from sending IPs with no PTR record, or whose
// PTR hostname doesn't resolve back to the same IP.
type PTRResult struct {
	IP      string `json:"ip"`
	PTRHost string `json:"ptr_host,omitempty"`
	Forward bool   `json:"forward_confirmed"`
	Status  string `json:"status"` // ok | warn | missing
	Detail  string `json:"detail,omitempty"`
}

// CheckPTR performs a reverse lookup on ip, then forward-resolves the returned
// hostname to confirm it points back to the same IP (FCrDNS).
func CheckPTR(ctx context.Context, ip string) PTRResult {
	res := PTRResult{IP: ip}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	names, err := resolver.LookupAddr(ctx, ip)
	if err != nil || len(names) == 0 {
		res.Status = "missing"
		res.Detail = "Bu IP için PTR (ters DNS) kaydı bulunamadı. Birçok kurumsal e-posta ağ geçidi PTR kaydı olmayan gönderenleri reddeder veya ağır biçimde puanlar — sunucu sağlayıcınızdan bu IP için bir PTR kaydı isteyin."
		return res
	}
	host := strings.TrimSuffix(names[0], ".")
	res.PTRHost = host

	fctx, fcancel := context.WithTimeout(ctx, 5*time.Second)
	defer fcancel()
	addrs, err := resolver.LookupHost(fctx, host)
	if err == nil {
		for _, a := range addrs {
			if a == ip {
				res.Forward = true
				break
			}
		}
	}
	if res.Forward {
		res.Status = "ok"
	} else {
		res.Status = "warn"
		res.Detail = fmt.Sprintf("PTR kaydı %q bulundu ancak ileri DNS bu ad için %s IP'sine dönmüyor (FCrDNS uyuşmazlığı).", host, ip)
	}
	return res
}
