package worker

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

// Message is a single outbound email.
type Message struct {
	From     string
	FromName string
	To       string
	Subject  string
	HTML     string
	Text     string
}

// Build renders RFC5322 multipart/alternative bytes.
func (m Message) Build() string {
	var b strings.Builder
	from := m.From
	if m.FromName != "" {
		from = fmt.Sprintf("%s <%s>", m.FromName, m.From)
	}
	boundary := "phishforge-boundary-42"
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + m.To + "\r\n")
	b.WriteString("Subject: " + m.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
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
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write([]byte(m.Build())); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}
	return c.Quit()
}
