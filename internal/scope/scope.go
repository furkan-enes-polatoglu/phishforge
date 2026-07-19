// Package scope enforces the engagement allowlist: a target may only be contacted
// if its email matches at least one scope rule. This is a primary safety guardrail
// — campaigns cannot send outside an authorized scope.
package scope

import (
	"path"
	"strings"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

// EmailDomain returns the lowercased domain part of an email, or "".
func EmailDomain(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return ""
	}
	return email[at+1:]
}

// Allowed reports whether email satisfies at least one rule.
//
//   - kind "domain": matches if the email domain equals the pattern, or is a
//     subdomain of it (e.g. rule "acme.com" allows "bob@mail.acme.com").
//   - kind "email": glob match against the full email (e.g. "*@acme.com",
//     "finance-*@acme.com").
//
// An empty rule set denies everything (fail closed).
func Allowed(email string, rules []models.ScopeRule) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(rules) == 0 {
		return false
	}
	dom := EmailDomain(email)
	for _, r := range rules {
		pat := strings.ToLower(strings.TrimSpace(r.Pattern))
		switch r.Kind {
		case "domain":
			if dom == pat || strings.HasSuffix(dom, "."+pat) {
				return true
			}
		case "email":
			if ok, _ := path.Match(pat, email); ok {
				return true
			}
		}
	}
	return false
}

// Partition splits a list of emails into allowed and rejected sets.
func Partition(emails []string, rules []models.ScopeRule) (allowed, rejected []string) {
	for _, e := range emails {
		if Allowed(e, rules) {
			allowed = append(allowed, e)
		} else {
			rejected = append(rejected, e)
		}
	}
	return
}
