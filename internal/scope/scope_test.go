package scope

import (
	"testing"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

func TestAllowed(t *testing.T) {
	rules := []models.ScopeRule{
		{Kind: "domain", Pattern: "acme.com"},
		{Kind: "email", Pattern: "vip-*@partner.io"},
	}
	cases := []struct {
		email string
		want  bool
	}{
		{"bob@acme.com", true},
		{"bob@mail.acme.com", true},     // subdomain allowed
		{"bob@notacme.com", false},      // suffix trick must not match
		{"vip-1@partner.io", true},      // glob match
		{"user@partner.io", false},      // glob miss
		{"", false},                     // empty
		{"bob@evil.com", false},         // unrelated
	}
	for _, c := range cases {
		if got := Allowed(c.email, rules); got != c.want {
			t.Errorf("Allowed(%q) = %v, want %v", c.email, got, c.want)
		}
	}
}

func TestEmptyRulesDenyAll(t *testing.T) {
	if Allowed("anyone@acme.com", nil) {
		t.Error("empty rule set must deny all (fail closed)")
	}
}
