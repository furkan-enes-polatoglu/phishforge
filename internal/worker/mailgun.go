package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// MailgunClient sends via Mailgun's HTTP API — the integration Mailgun itself
// recommends over raw SMTP AUTH relay: no SMTP-timeout risk, per-message
// delivery status via the API response, and (paired with the webhook receiver)
// a real delivered/bounced/complained feedback loop.
type MailgunClient struct {
	APIKey string
	Domain string
	client *http.Client
}

func NewMailgunClient(apiKey, domain string) *MailgunClient {
	return &MailgunClient{APIKey: apiKey, Domain: domain, client: &http.Client{Timeout: 20 * time.Second}}
}

// MailgunSendParams mirrors the fields needed to send one message.
type MailgunSendParams struct {
	From             string // technical/authenticated sender (envelope + DKIM domain Mailgun signs with)
	FromName         string
	HeaderFrom       string // optional pretext override of the visible From: address
	HeaderFromName   string
	To               string
	Subject          string
	HTML             string
	Text             string
	ReplyTo          string
	CampaignTargetID string // correlation key, sent as Mailgun custom variable v:cid so webhooks can be matched back
}

// Send posts the message to Mailgun's /messages endpoint and returns Mailgun's
// message id (useful for support/debugging, not required for webhook
// correlation — that uses the custom variable instead, which survives
// Mailgun's own message-id rewriting).
func (c *MailgunClient) Send(ctx context.Context, p MailgunSendParams) (messageID string, err error) {
	displayAddr, displayName := p.From, p.FromName
	if p.HeaderFrom != "" {
		displayAddr, displayName = p.HeaderFrom, p.HeaderFromName
	}
	from := displayAddr
	if displayName != "" {
		from = fmt.Sprintf("%s <%s>", displayName, displayAddr)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fields := map[string]string{
		"from":       from,
		"to":         p.To,
		"subject":    p.Subject,
		"html":       p.HTML,
		"text":       p.Text,
		"h:Reply-To": p.ReplyTo,
		// We do our own open/click tracking (signed RIDs); Mailgun's own
		// tracking would rewrite links and inject its own pixel into the body
		// AFTER any point we might sign it, and would double up with ours.
		"o:tracking": "no",
		"o:tag":      "phishforge",
		"v:cid":      p.CampaignTargetID,
	}
	for k, v := range fields {
		if v == "" {
			continue
		}
		if err := w.WriteField(k, v); err != nil {
			return "", err
		}
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.mailgun.net/v3/%s/messages", c.Domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth("api", c.APIKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("mailgun request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("mailgun returned %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(body, &out)
	return out.ID, nil
}
