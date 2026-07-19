package phishing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/google/uuid"
)

// mailgunWebhookPayload is the subset of Mailgun's webhook JSON we need.
// Reference: https://documentation.mailgun.com/en/latest/user_manual.html#webhooks
type mailgunWebhookPayload struct {
	Signature struct {
		Timestamp string `json:"timestamp"`
		Token     string `json:"token"`
		Signature string `json:"signature"`
	} `json:"signature"`
	EventData struct {
		Event         string            `json:"event"` // "delivered" | "failed" | "complained" | ...
		Severity      string            `json:"severity"`
		Recipient     string            `json:"recipient"`
		UserVariables map[string]string `json:"user-variables"`
	} `json:"event-data"`
}

// verifyMailgunSignature checks Mailgun's HMAC-SHA256(timestamp+token) against
// the account's webhook signing key. If no key is configured, verification is
// skipped (development convenience) — production deployments should always set
// MAILGUN_WEBHOOK_SIGNING_KEY.
func verifyMailgunSignature(signingKey, timestamp, token, signature string) bool {
	if signingKey == "" {
		return true
	}
	mac := hmac.New(sha256.New, []byte(signingKey))
	mac.Write([]byte(timestamp + token))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// handleMailgunWebhook ingests Mailgun's delivered/failed(bounce)/complained
// events — the delivery feedback loop that raw SMTP sending can never provide.
// Correlation back to our internal campaign_target uses the "cid" custom
// variable we attach at send time (MailgunSendParams.CampaignTargetID), which
// survives Mailgun's own message-id rewriting.
func (s *Server) handleMailgunWebhook(w http.ResponseWriter, r *http.Request) {
	var payload mailgunWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !verifyMailgunSignature(s.cfg.MailgunWebhookSigningKey, payload.Signature.Timestamp, payload.Signature.Token, payload.Signature.Signature) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var evType models.EventType
	switch payload.EventData.Event {
	case "delivered":
		evType = models.EventDelivered
	case "failed":
		evType = models.EventBounced
	case "complained":
		evType = models.EventComplained
	default:
		// Accept and ignore events we don't track (opened/clicked/unsubscribed —
		// we already do our own open/click tracking via signed RIDs).
		w.WriteHeader(http.StatusOK)
		return
	}

	ctID, err := uuid.Parse(payload.EventData.UserVariables["cid"])
	if err != nil {
		w.WriteHeader(http.StatusOK) // ack — nothing to correlate, not Mailgun's fault
		return
	}
	_ = s.st.RecordEvent(r.Context(), &models.Event{CampaignTargetID: ctID, Type: evType})

	if evType == models.EventBounced || evType == models.EventComplained {
		s.maybeAutoPause(r.Context(), ctID)
	}
	w.WriteHeader(http.StatusOK)
}
