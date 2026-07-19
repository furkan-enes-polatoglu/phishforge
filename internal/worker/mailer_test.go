package worker

import (
	"strings"
	"testing"
)

func TestBuildUsesHeaderFromOverrideForPretextRealism(t *testing.T) {
	m := Message{
		From: "sim@sending-domain.test", FromName: "Sim Sender",
		HeaderFrom: "ceo@realtarget.com", HeaderFromName: "Jane CEO",
		To: "bob@realtarget.com", Subject: "Hi", HTML: "<p>hi</p>",
	}
	raw := m.Build()
	if !strings.Contains(raw, "From: Jane CEO <ceo@realtarget.com>") {
		t.Errorf("expected visible From to use the spoofed header override, got:\n%s", raw)
	}
	// The technical/envelope identity must still be reflected in Message-ID's
	// domain — it must NOT leak the target's real domain there.
	if !strings.Contains(raw, "@sending-domain.test>") {
		t.Errorf("expected Message-ID to use the technical sending domain, got:\n%s", raw)
	}
}

func TestBuildFallsBackToTechnicalFromWithoutOverride(t *testing.T) {
	m := Message{From: "it@acme-test.com", FromName: "IT", To: "bob@corp.com", Subject: "Hi", HTML: "<p>hi</p>"}
	raw := m.Build()
	if !strings.Contains(raw, "From: IT <it@acme-test.com>") {
		t.Errorf("expected From to fall back to the technical sender, got:\n%s", raw)
	}
}

func TestBuildIncludesReplyTo(t *testing.T) {
	m := Message{From: "it@acme-test.com", To: "bob@corp.com", Subject: "Hi", HTML: "x", ReplyTo: "watch@ourfirm.test"}
	raw := m.Build()
	if !strings.Contains(raw, "Reply-To: watch@ourfirm.test\r\n") {
		t.Errorf("expected Reply-To header, got:\n%s", raw)
	}
}

func TestBuildOmitsReplyToWhenEmpty(t *testing.T) {
	m := Message{From: "it@acme-test.com", To: "bob@corp.com", Subject: "Hi", HTML: "x"}
	raw := m.Build()
	if strings.Contains(raw, "Reply-To:") {
		t.Errorf("expected no Reply-To header when unset, got:\n%s", raw)
	}
}

func TestBuildIncludesMailgunVariablesForWebhookCorrelation(t *testing.T) {
	m := Message{
		From: "it@acme-test.com", To: "bob@corp.com", Subject: "Hi", HTML: "x",
		Variables: map[string]string{"cid": "abc-123"},
	}
	raw := m.Build()
	if !strings.Contains(raw, `X-Mailgun-Variables: {"cid":"abc-123"}`) {
		t.Errorf("expected X-Mailgun-Variables header for webhook correlation, got:\n%s", raw)
	}
}

func TestBuildOmitsMailgunVariablesWhenEmpty(t *testing.T) {
	m := Message{From: "it@acme-test.com", To: "bob@corp.com", Subject: "Hi", HTML: "x"}
	raw := m.Build()
	if strings.Contains(raw, "X-Mailgun-Variables:") {
		t.Errorf("expected no X-Mailgun-Variables header when unset, got:\n%s", raw)
	}
}
