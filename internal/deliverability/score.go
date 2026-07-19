package deliverability

// DeliveryScore aggregates every signal this package can gather into a single
// 0-100 confidence score with a letter grade, so an operator (or the client)
// can see at a glance how ready a sending setup is — instead of having to
// mentally weigh a dozen separate checks.
type DeliveryScore struct {
	Score     int            `json:"score"`
	Grade     string         `json:"grade"` // A/B/C/D/F
	Breakdown map[string]int `json:"breakdown"`
}

// Compute derives a DeliveryScore from a populated CheckResult. Call it after
// filling in SPF/DMARC/DKIM/PTR/MTA-STS/RBL/Content — any left nil/zero simply
// don't contribute points (treated as "unknown", not "bad").
func (r CheckResult) ComputeScore() DeliveryScore {
	b := map[string]int{}
	total := 0

	// Authentication — the core deliverability trio (60 points).
	if r.SPF.Status == "ok" {
		b["spf"] = 20
	} else if r.SPF.Status == "warn" {
		b["spf"] = 10
	}
	if r.DKIM.Status == "ok" {
		b["dkim"] = 20
	}
	switch r.DMARC.Status {
	case "ok":
		b["dmarc"] = 20
	case "warn":
		b["dmarc"] = 12 // e.g. present but p=none
	}

	// Sending infrastructure hygiene (25 points).
	if r.PTR != nil {
		switch r.PTR.Status {
		case "ok":
			b["ptr"] = 15
		case "warn":
			b["ptr"] = 7
		}
	}
	if r.MTASTS != nil && r.MTASTS.Found {
		b["mta_sts"] = 5
	}
	if r.TLSRPT {
		b["tls_rpt"] = 5
	}

	// Reputation (15 points) — deduct for every blocklist hit.
	rblPoints := 15
	for _, rb := range r.RBL {
		if rb.Listed {
			rblPoints -= 8
		}
	}
	if rblPoints < 0 {
		rblPoints = 0
	}
	if len(r.RBL) > 0 {
		b["reputation"] = rblPoints
	}

	// Content hygiene (bonus up to 10, penalized by heuristic hits).
	contentPoints := 10
	if r.Content != nil {
		contentPoints -= r.Content.HeuristicPenalty / 10
		if contentPoints < 0 {
			contentPoints = 0
		}
		b["content"] = contentPoints
	}

	for _, v := range b {
		total += v
	}
	if total > 100 {
		total = 100
	}

	grade := "F"
	switch {
	case total >= 90:
		grade = "A"
	case total >= 75:
		grade = "B"
	case total >= 60:
		grade = "C"
	case total >= 40:
		grade = "D"
	}
	return DeliveryScore{Score: total, Grade: grade, Breakdown: b}
}
