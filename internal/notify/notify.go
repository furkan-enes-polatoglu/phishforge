// Package notify delivers signed webhook and Slack/Teams notifications for
// campaign events.
package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var client = &http.Client{Timeout: 8 * time.Second}

// Webhook posts a JSON payload with an HMAC signature header (X-PhishForge-Signature).
func Webhook(ctx context.Context, url, secret string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		req.Header.Set("X-PhishForge-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

// Slack posts a simple text message to a Slack/Teams-compatible incoming webhook.
func Slack(ctx context.Context, webhookURL, text string) error {
	return Webhook(ctx, webhookURL, "", map[string]string{"text": text})
}
