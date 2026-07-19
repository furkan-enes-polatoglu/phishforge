package worker

import "testing"

func TestIsMailgunSMTP(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"smtp.mailgun.org", true},
		{"smtp.eu.mailgun.org", true},
		{"SMTP.MAILGUN.ORG", true},
		{"smtp.gmail.com", false},
		{"mail.acme-corp.com", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isMailgunSMTP(c.host); got != c.want {
			t.Errorf("isMailgunSMTP(%q) = %v, want %v", c.host, got, c.want)
		}
	}
}
