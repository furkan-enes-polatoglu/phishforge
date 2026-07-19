package phishing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyMailgunSignature(t *testing.T) {
	key := "test-signing-key"
	timestamp, token := "1234567890", "abcstaticnonce"
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(timestamp + token))
	valid := hex.EncodeToString(mac.Sum(nil))

	if !verifyMailgunSignature(key, timestamp, token, valid) {
		t.Error("expected a correctly computed signature to verify")
	}
	if verifyMailgunSignature(key, timestamp, token, "deadbeef") {
		t.Error("expected a tampered signature to fail verification")
	}
	if verifyMailgunSignature(key, timestamp, "different-token", valid) {
		t.Error("expected signature computed for a different token to fail")
	}
}

func TestVerifyMailgunSignatureNoKeyConfigured(t *testing.T) {
	if !verifyMailgunSignature("", "any", "thing", "whatever") {
		t.Error("expected verification to be skipped (return true) when no signing key is configured")
	}
}
