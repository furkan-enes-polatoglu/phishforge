// Package gateway detects which email security gateway sits in front of a
// target organization (via MX record fingerprinting) and generates a tailored,
// step-by-step allowlisting playbook — including a ready-to-send request the
// operator can forward to the client's IT/security team.
//
// This is the single highest-leverage action for guaranteeing inbox delivery in
// an authorized engagement: getting the simulation's sending infrastructure
// explicitly allowlisted in the exact product standing between the sender and
// the target mailbox. Every major gateway calls this something different and
// buries it in a different part of its admin console — this package encodes
// that knowledge so operators don't have to look it up under time pressure.
package gateway

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// Provider describes a known email security gateway product and exactly how to
// allowlist a sender in it.
type Provider struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	FeatureName string   `json:"feature_name"`
	Steps       []string `json:"steps"`
}

type fingerprint struct {
	match    []string // substrings matched against lowercased MX hostnames
	provider Provider
}

// knownGateways encodes publicly documented allowlisting mechanisms for the
// gateway products most commonly seen protecting corporate mailboxes. MX
// hostname patterns are stable, well-known signatures for each provider.
var knownGateways = []fingerprint{
	{
		// EOP-protected tenants route through various subdomains of
		// protection.outlook.com (commonly "*.mail.protection.outlook.com" for
		// business tenants, "*.olc.protection.outlook.com" for outlook.com/live.com
		// consumer routing) — match the stable common suffix, not one specific form.
		match: []string{"protection.outlook.com"},
		provider: Provider{
			ID: "m365", Name: "Microsoft 365 / Exchange Online Protection",
			FeatureName: "Gelişmiş Teslimat (Advanced Delivery) → Kimlik Avı Simülasyonu",
			Steps: []string{
				"Microsoft 365 Defender portalına gidin: security.microsoft.com",
				"E-posta ve işbirliği → İlkeler ve kurallar → Tehdit ilkeleri → Gelişmiş Teslimat",
				`"Kimlik avı simülasyonu" (Phishing simulation) sekmesine gönderen alan adını, gönderen IP'yi ve gönderim URL desenini ekleyin — bu özellik Microsoft tarafından özellikle 3. parti oltalama simülasyon araçları için tasarlanmıştır`,
				"Ek olarak: Exchange Yönetim Merkezi → Mail akışı → Koruma → Bağlantı filtresi → IP İzin Listesi'ne gönderen IP'yi ekleyin",
				"Test bağlantılarının bozulmaması için aynı ilkede Safe Links / Safe Attachments istisnası tanımlayın",
			},
		},
	},
	{
		match: []string{"google.com"}, // aspmx.l.google.com, alt1-4.aspmx.l.google.com
		provider: Provider{
			ID: "google", Name: "Google Workspace (Gmail)",
			FeatureName: "E-posta Beyaz Listesi (Email allowlist)",
			Steps: []string{
				"Google Yönetici Konsolu: admin.google.com",
				"Uygulamalar → Google Workspace → Gmail → Spam, Kimlik Avı ve Kötü Amaçlı Yazılım ayarları",
				`"E-posta beyaz listesi" (Email allowlist) alanına gönderen sunucunun IP adresini ekleyin`,
				"Not: Bu ayar listelenen kaynak için spam/kimlik avı/kötü amaçlı yazılım taramasını tamamen atlar — yalnızca test süresince aktif tutulması önerilir",
			},
		},
	},
	{
		match: []string{"pphosted.com"},
		provider: Provider{
			ID: "proofpoint", Name: "Proofpoint Email Protection",
			FeatureName: "Allowed Senders List / Bypass Kuralı",
			Steps: []string{
				"Proofpoint yönetim konsoluna (Protection Server) giriş yapın",
				"Email Firewall / Classifier kuralları altında gönderen IP/alan adı için bir 'Bypass' (izin ver) kuralı oluşturun",
				"Kurumsal 'Allowed Senders List' kaydına gönderen alan adını ve IP'yi ekleyin",
			},
		},
	},
	{
		match: []string{"mimecast.com"},
		provider: Provider{
			ID: "mimecast", Name: "Mimecast",
			FeatureName: "Permitted Senders Policy",
			Steps: []string{
				"Mimecast Administration Console'a giriş yapın",
				"Gateway → Policies → Permitted Senders",
				"Gönderen alan adı/IP için yeni bir 'Permit' politikası oluşturun (spam taramasını atlar)",
			},
		},
	},
	{
		match: []string{"barracudanetworks.com"},
		provider: Provider{
			ID: "barracuda", Name: "Barracuda Email Security Gateway",
			FeatureName: "IP/Sender Exemptions",
			Steps: []string{
				"Barracuda yönetim arayüzüne giriş yapın",
				"Inbound Settings → Sender Authentication → IP Address / Sender Exemptions",
				"Gönderen IP'yi ve/veya alan adını 'exempt' (muaf) olarak ekleyin",
			},
		},
	},
	{
		match: []string{"iphmx.com"},
		provider: Provider{
			ID: "cisco", Name: "Cisco Secure Email (IronPort) Cloud Gateway",
			FeatureName: "HAT İzinli Gönderen Grubu (Sender Group)",
			Steps: []string{
				"Cisco Secure Email yönetim arayüzüne giriş yapın",
				"Mail Policies → HAT Overview → ilgili listener",
				"Gönderen IP'yi 'ALLOWED_LIST' (veya eşdeğer izinli) sender group'a ekleyin",
			},
		},
	},
}

// DetectProvider looks up MX records for domain and matches them against known
// gateway fingerprints. Returns nil provider (not an error) when the MX records
// resolve but don't match any known signature — that's still useful information
// (the target uses a custom/unrecognized gateway).
func DetectProvider(ctx context.Context, domain string) (provider *Provider, mxHosts []string, err error) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return nil, nil, fmt.Errorf("alan adı boş olamaz")
	}
	resolver := net.Resolver{}
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	mxs, err := resolver.LookupMX(ctx, domain)
	if err != nil || len(mxs) == 0 {
		return nil, nil, fmt.Errorf("bu alan adı için MX kaydı bulunamadı: %w", err)
	}
	hosts := make([]string, 0, len(mxs))
	for _, mx := range mxs {
		hosts = append(hosts, strings.ToLower(strings.TrimSuffix(mx.Host, ".")))
	}
	for _, fp := range knownGateways {
		for _, host := range hosts {
			for _, m := range fp.match {
				if strings.Contains(host, m) {
					p := fp.provider
					return &p, hosts, nil
				}
			}
		}
	}
	return nil, hosts, nil
}

// CoverEmailRequest carries the engagement-specific values used to fill in the
// generated allowlist request.
type CoverEmailRequest struct {
	ClientName    string
	SendingDomain string
	SendingIP     string
	DKIMDomain    string
	DKIMSelector  string
	StartsAt      time.Time
	EndsAt        time.Time
}

// BuildCoverEmail renders a ready-to-send (Turkish) allowlist request the
// operator can forward to the client's IT/security team, filled in with the
// detected provider's exact steps and this engagement's sending details.
func BuildCoverEmail(p *Provider, req CoverEmailRequest) string {
	providerName := "tespit edilemedi (özel/bilinmeyen bir ağ geçidi kullanıyor olabilirler)"
	stepsText := "IT ekibinizden kullandıkları e-posta güvenlik ağ geçidini (Proofpoint, Mimecast, Microsoft 365, Google Workspace, Barracuda, Cisco vb.) öğrenip aşağıdaki gönderim bilgilerini o ürünün 'izinli gönderen' listesine eklemelerini isteyin."
	if p != nil {
		providerName = fmt.Sprintf("%s (%s)", p.Name, p.FeatureName)
		var sb strings.Builder
		for i, s := range p.Steps {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, s))
		}
		stepsText = sb.String()
	}
	starts, ends := req.StartsAt.Format("02.01.2006"), req.EndsAt.Format("02.01.2006")
	return fmt.Sprintf(`Konu: [Yetkili Güvenlik Testi] E-posta Gönderim Altyapısının Beyaz Listeye Alınması Talebi

Merhaba,

%s için planlanan yetkili güvenlik farkındalık / oltalama simülasyonu kapsamında
(%s – %s tarihleri arasında), test e-postalarının filtrelerinize takılmadan
hedef kutulara ulaşması gerekmektedir. Bu, testin gerçekçi ve anlamlı sonuçlar
üretebilmesi için teknik bir zorunluluktur.

Kurumunuzun e-posta ağ geçidi: %s

Lütfen aşağıdaki gönderim bilgilerini ilgili beyaz listeye ekleyin:
  Gönderen alan adı : %s
  Gönderen IP       : %s
  DKIM alan adı     : %s (seçici: %s)

Önerilen adımlar:
%s
Bu izin yalnızca test süresince (%s – %s) tanımlanabilir ve sonrasında güvenle
kaldırılabilir.

Sorularınız için bize ulaşabilirsiniz.
`,
		req.ClientName, starts, ends, providerName,
		req.SendingDomain, req.SendingIP, req.DKIMDomain, req.DKIMSelector,
		stepsText, starts, ends)
}
