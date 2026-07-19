package worker

import (
	"testing"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

func TestEffectiveLandingBasePrefersCampaignOverride(t *testing.T) {
	p := &models.SendingProfile{LandingBaseURL: "https://portal.profile-default.com"}
	got := effectiveLandingBase("https://one-off-override.test/", p, "https://global-default.test")
	if got != "https://one-off-override.test" {
		t.Errorf("expected campaign override to win (trailing slash trimmed), got %q", got)
	}
}

func TestEffectiveLandingBaseFallsBackToProfile(t *testing.T) {
	p := &models.SendingProfile{LandingBaseURL: "https://portal.client-a.com/"}
	got := effectiveLandingBase("", p, "https://global-default.test")
	if got != "https://portal.client-a.com" {
		t.Errorf("expected profile default (trailing slash trimmed), got %q", got)
	}
}

func TestEffectiveLandingBaseFallsBackToGlobalDefault(t *testing.T) {
	p := &models.SendingProfile{LandingBaseURL: ""}
	got := effectiveLandingBase("", p, "https://global-default.test")
	if got != "https://global-default.test" {
		t.Errorf("expected fallback to global default, got %q", got)
	}
}
