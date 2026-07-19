package worker

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/dkim"
	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

func messageID(domain string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	if domain == "" {
		domain = "phishforge.local"
	}
	return "<" + hex.EncodeToString(b) + "@" + domain + ">"
}

// Message is a single outbound email. From/FromName are the technically
// authenticated sender (used for the envelope MAIL FROM, DKIM signing domain,
// and Message-ID host) — these must stay the sending profile's own address for
// SPF/DKIM to have any chance of aligning. HeaderFrom/HeaderFromName optionally
// override what's DISPLAYED in the visible "From:" line, decoupled from the
// technical sender — this is how pretext realism (appearing as an exact real
// address) is supported without lying about the technical sending identity.
// When the two differ, DMARC alignment will fail at the target unless the
// sending infrastructure is allowlisted (see the gateway-detection feature).
type Message struct {
	From           string
	FromName       string
	HeaderFrom     string
	HeaderFromName string
	To             string
	Subject        string
	HTML           string
	Text           string
	Unsubscribe    string // optional List-Unsubscribe URL
	XMailer        string // optional realistic mail-client header (deliverability)
	ReplyTo        string // optional Reply-To, independent of the display From
}

func domainOf(addr string) string {
	if i := strings.LastIndex(addr, "@"); i >= 0 {
		return addr[i+1:]
	}
	return ""
}

// Build renders a well-formed RFC5322 multipart/alternative message with the
// headers that legitimate mail carries (Message-ID, Date, MIME-Version, and an
// optional List-Unsubscribe) — these improve deliverability for authorized senders.
func (m Message) Build() string {
	var b strings.Builder
	displayAddr, displayName := m.From, m.FromName
	if m.HeaderFrom != "" {
		displayAddr, displayName = m.HeaderFrom, m.HeaderFromName
	}
	from := displayAddr
	if displayName != "" {
		from = fmt.Sprintf("%s <%s>", displayName, displayAddr)
	}
	boundary := "phishforge-boundary-42"
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + m.To + "\r\n")
	b.WriteString("Subject: " + m.Subject + "\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("Message-ID: " + messageID(domainOf(m.From)) + "\r\n")
	if m.ReplyTo != "" {
		b.WriteString("Reply-To: " + m.ReplyTo + "\r\n")
	}
	if m.XMailer != "" {
		b.WriteString("X-Mailer: " + m.XMailer + "\r\n")
	}
	if m.Unsubscribe != "" {
		b.WriteString("List-Unsubscribe: <" + m.Unsubscribe + ">\r\n")
		b.WriteString("List-Unsubscribe-Post: List-Unsubscribe=One-Click\r\n")
	}
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n")
	if m.Text != "" {
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(m.Text + "\r\n")
	}
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	b.WriteString(m.HTML + "\r\n")
	b.WriteString("--" + boundary + "--\r\n")
	return b.String()
}

// Send delivers a message via the sending profile's SMTP server, using STARTTLS
// when available.
func Send(p *models.SendingProfile, m Message) error {
	addr := net.JoinHostPort(p.SMTPHost, strconv.Itoa(p.SMTPPort))
	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	c, err := smtp.NewClient(conn, p.SMTPHost)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if p.UseTLS {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: p.SMTPHost, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("starttls: %w", err)
			}
		}
	}
	if p.Username != "" {
		auth := smtp.PlainAuth("", p.Username, p.Password, p.SMTPHost)
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(auth); err != nil {
				return fmt.Errorf("auth: %w", err)
			}
		}
	}
	if err := c.Mail(p.FromAddress); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := c.Rcpt(m.To); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}
	raw := m.Build()
	// DKIM-sign when the profile is configured for it (legitimate deliverability).
	if p.SignDKIM && p.DKIMPrivateKey != "" && p.DKIMDomain != "" && p.DKIMSelector != "" {
		if signed, err := dkim.Sign(raw, p.DKIMDomain, p.DKIMSelector, p.DKIMPrivateKey); err == nil {
			raw = signed
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write([]byte(raw)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}
	return c.Quit()
}
