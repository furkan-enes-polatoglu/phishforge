package gateway

import (
	"strings"
	"testing"
	"time"
)

func TestBuildCoverEmailWithProvider(t *testing.T) {
	p := &Provider{ID: "m365", Name: "Microsoft 365 / Exchange Online Protection", FeatureName: "Advanced Delivery", Steps: []string{"Step one", "Step two"}}
	email := BuildCoverEmail(p, CoverEmailRequest{
		ClientName: "Acme Corp", SendingDomain: "sim.acme-test.com", SendingIP: "203.0.113.10",
		DKIMDomain: "sim.acme-test.com", DKIMSelector: "pf1",
		StartsAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
	})
	for _, want := range []string{"Acme Corp", "Microsoft 365", "sim.acme-test.com", "203.0.113.10", "pf1", "Step one", "Step two", "01.08.2026", "31.08.2026"} {
		if !strings.Contains(email, want) {
			t.Errorf("cover email missing %q:\n%s", want, email)
		}
	}
}

func TestBuildCoverEmailUnknownProvider(t *testing.T) {
	email := BuildCoverEmail(nil, CoverEmailRequest{
		ClientName: "Acme", SendingDomain: "sim.acme.com", SendingIP: "1.2.3.4",
		StartsAt: time.Now(), EndsAt: time.Now(),
	})
	if !strings.Contains(email, "tespit edilemedi") {
		t.Errorf("expected unknown-provider fallback text, got:\n%s", email)
	}
}

func TestKnownGatewayFingerprints(t *testing.T) {
	cases := []struct {
		host   string
		wantID string
	}{
		{"acme-com.mail.protection.outlook.com", "m365"},
		{"outlook-com.olc.protection.outlook.com", "m365"}, // real-world consumer routing variant
		{"aspmx.l.google.com", "google"},
		{"mx0a-00123456.pphosted.com", "proofpoint"},
		{"eu-smtp-inbound-1.mimecast.com", "mimecast"},
		{"acme.mail.barracudanetworks.com", "barracuda"},
		{"acme.iphmx.com", "cisco"},
	}
	for _, c := range cases {
		matched := false
		for _, fp := range knownGateways {
			for _, m := range fp.match {
				if strings.Contains(c.host, m) {
					if fp.provider.ID != c.wantID {
						t.Errorf("host %q matched provider %q, want %q", c.host, fp.provider.ID, c.wantID)
					}
					matched = true
				}
			}
		}
		if !matched {
			t.Errorf("host %q matched no known gateway (want %q)", c.host, c.wantID)
		}
	}
}
