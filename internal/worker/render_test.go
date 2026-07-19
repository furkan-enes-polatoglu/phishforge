package worker

import (
	"strings"
	"testing"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
)

func TestRewriteLinks(t *testing.T) {
	in := `<p><a href="https://real.example/login">click</a> and <a href='http://x.io'>x</a></p>`
	out := RewriteLinks(in, "https://phish.test/l/RID")
	if strings.Contains(out, "real.example") || strings.Contains(out, "x.io") {
		t.Fatalf("original hrefs should be replaced: %s", out)
	}
	if strings.Count(out, "https://phish.test/l/RID") != 2 {
		t.Fatalf("expected 2 rewritten links, got: %s", out)
	}
}

func TestInSendWindow(t *testing.T) {
	c := models.Campaign{SendWindowStart: 9, SendWindowEnd: 17, BusinessDaysOnly: true}
	// Wednesday 10:00 — inside window
	wed10 := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	if !c.InSendWindow(wed10) {
		t.Error("Wed 10:00 should be inside window")
	}
	// Wednesday 18:00 — after window
	if c.InSendWindow(time.Date(2026, 7, 15, 18, 0, 0, 0, time.UTC)) {
		t.Error("Wed 18:00 should be outside window")
	}
	// Saturday 10:00 — weekend excluded
	if c.InSendWindow(time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)) {
		t.Error("Saturday should be excluded (business days only)")
	}
}
