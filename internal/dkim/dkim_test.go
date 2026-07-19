package dkim

import (
	"strings"
	"testing"
)

func TestGenerateAndSign(t *testing.T) {
	priv, dnsName, dnsValue, err := GenerateKey("pf1", "acme.com")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if dnsName != "pf1._domainkey.acme.com" {
		t.Errorf("unexpected dns name: %s", dnsName)
	}
	if !strings.HasPrefix(dnsValue, "v=DKIM1; k=rsa; p=") {
		t.Errorf("unexpected dns value: %s", dnsValue)
	}

	raw := "From: it@acme.com\r\nTo: bob@corp.com\r\nSubject: Hello\r\nDate: Mon, 01 Jan 2026 00:00:00 +0000\r\nMIME-Version: 1.0\r\nContent-Type: text/plain\r\n\r\nbody\r\n"
	signed, err := Sign(raw, "acme.com", "pf1", priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if !strings.Contains(signed, "DKIM-Signature:") {
		t.Fatalf("signed message missing DKIM-Signature header:\n%s", signed)
	}
	if !strings.Contains(signed, "d=acme.com") || !strings.Contains(signed, "s=pf1") {
		t.Errorf("DKIM-Signature missing domain/selector tags")
	}
}
