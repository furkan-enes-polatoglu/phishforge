package worker

import (
	"testing"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

func TestEffectiveLandingBasePrefersProfileOverride(t *testing.T) {
	p := &models.SendingProfile{LandingBaseURL: "https://portal.client-a.com/"}
	if got := effectiveLandingBase(p, "https://global-default.test"); got != "https://portal.client-a.com" {
		t.Errorf("expected profile override (trailing slash trimmed), got %q", got)
	}
}

func TestEffectiveLandingBaseFallsBackToGlobalDefault(t *testing.T) {
	p := &models.SendingProfile{LandingBaseURL: ""}
	if got := effectiveLandingBase(p, "https://global-default.test"); got != "https://global-default.test" {
		t.Errorf("expected fallback to global default, got %q", got)
	}
}
