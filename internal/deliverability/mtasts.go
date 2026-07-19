package deliverability

import (
	"context"
	"strings"
)

// MTASTSResult reports whether a sending domain publishes an MTA-STS policy —
// increasingly expected of mature, trustworthy senders and a positive signal to
// receiving gateways that this domain takes transport security seriously.
type MTASTSResult struct {
	Found  bool   `json:"found"`
	Status string `json:"status"` // ok | warn
	Detail string `json:"detail,omitempty"`
}

// CheckMTASTS looks for a well-formed "_mta-sts.<domain>" TXT record
// (v=STSv1; id=...). It does not fetch the HTTPS policy file itself — DNS
// presence is enough to advise the operator either way.
func CheckMTASTS(ctx context.Context, domain string) MTASTSResult {
	txts, err := lookupTXT(ctx, "_mta-sts."+domain)
	if err == nil {
		for _, t := range txts {
			if strings.HasPrefix(strings.ToLower(t), "v=stsv1") {
				return MTASTSResult{Found: true, Status: "ok"}
			}
		}
	}
	return MTASTSResult{
		Found: false, Status: "warn",
		Detail: "MTA-STS kaydı yok. Zorunlu değildir ama gönderen alan adının TLS zorunluluğu tanımlaması, bazı alıcı ağ geçitlerinde itibarı güçlendirir.",
	}
}

// CheckTLSRPT reports whether the domain publishes a TLS-RPT reporting address
// ("_smtp._tls.<domain>" TXT, v=TLSRPTv1). Usually configured alongside MTA-STS.
func CheckTLSRPT(ctx context.Context, domain string) bool {
	txts, err := lookupTXT(ctx, "_smtp._tls."+domain)
	if err != nil {
		return false
	}
	for _, t := range txts {
		if strings.HasPrefix(strings.ToLower(t), "v=tlsrptv1") {
			return true
		}
	}
	return false
}
