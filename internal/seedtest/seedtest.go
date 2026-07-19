// Package seedtest performs an automated inbox-placement check: it connects to
// a seed mailbox via IMAP and reports which folder a marked test message landed
// in (Inbox vs. Spam/Junk). This closes the loop on the deliverability
// promise — rather than just checking DNS records, it observes the real
// outcome at a real mailbox for an authorized test send.
package seedtest

import (
	"fmt"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	UseTLS   bool
}

// candidateFolders covers the common Inbox/Spam naming across major providers.
// Folders that don't exist for a given provider are skipped silently.
var candidateFolders = []string{
	"INBOX", "Spam", "Junk", "Junk E-mail", "[Gmail]/Spam", "Bulk Mail", "Bulk", "Gereksiz", "Önemsiz",
}

// CheckPlacement logs into the seed mailbox and searches each candidate folder
// for a message whose Subject contains marker. It returns the first folder the
// message is found in (INBOX first), or found=false if it isn't in any of them
// yet (e.g. still in transit, or filtered somewhere this tool doesn't check).
func CheckPlacement(cfg Config, marker string) (folder string, found bool, err error) {
	if marker == "" {
		return "", false, fmt.Errorf("subject_marker boş olamaz")
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	var c *client.Client
	if cfg.UseTLS {
		c, err = client.DialTLS(addr, nil)
	} else {
		c, err = client.Dial(addr)
	}
	if err != nil {
		return "", false, fmt.Errorf("IMAP bağlantısı kurulamadı: %w", err)
	}
	defer c.Logout()

	if err := c.Login(cfg.Username, cfg.Password); err != nil {
		return "", false, fmt.Errorf("IMAP girişi başarısız: %w", err)
	}

	for _, name := range candidateFolders {
		if _, err := c.Select(name, true); err != nil {
			continue // folder doesn't exist for this provider — try the next
		}
		criteria := imap.NewSearchCriteria()
		criteria.Header.Add("Subject", marker)
		ids, err := c.Search(criteria)
		if err != nil {
			continue
		}
		if len(ids) > 0 {
			return name, true, nil
		}
	}
	return "", false, nil
}
