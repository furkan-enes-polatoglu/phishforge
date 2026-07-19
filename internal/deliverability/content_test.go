package deliverability

import "testing"

func TestAnalyzeContentCleanEmail(t *testing.T) {
	res := AnalyzeContent("Aylık rapor hazır", "<p>Merhaba, aylık raporunuz ekte yer almaktadır. Teşekkürler.</p>")
	if res.HeuristicPenalty != 0 {
		t.Errorf("expected 0 penalty for clean content, got %d (%+v)", res.HeuristicPenalty, res)
	}
}

func TestAnalyzeContentFlagsTriggerWordsAndShorteners(t *testing.T) {
	res := AnalyzeContent("URGENT: Congratulations, you are a WINNER!!!", `<p>Click here <a href="https://bit.ly/abc">now</a></p>`)
	if len(res.TriggerWordsFound) == 0 {
		t.Error("expected trigger words to be found")
	}
	if len(res.ShortenersFound) == 0 {
		t.Error("expected bit.ly shortener to be flagged")
	}
	if res.AllCapsWords == 0 {
		t.Error("expected ALL-CAPS words to be counted")
	}
	if res.HeuristicPenalty == 0 {
		t.Error("expected nonzero penalty for spammy content")
	}
}

func TestAnalyzeContentImageOnlyWarning(t *testing.T) {
	res := AnalyzeContent("Invoice", `<img src="x.png">`)
	if !res.ImageOnlyWarning {
		t.Error("expected image-only warning for an all-image body with no text")
	}
}

func TestComputeScoreAllGood(t *testing.T) {
	r := CheckResult{
		SPF:    RecordCheck{Status: "ok"},
		DKIM:   RecordCheck{Status: "ok"},
		DMARC:  RecordCheck{Status: "ok"},
		PTR:    &PTRResult{Status: "ok"},
		MTASTS: &MTASTSResult{Found: true},
		TLSRPT: true,
		RBL:    []RBLResult{{List: "zen.spamhaus.org", Listed: false}},
		Content: &ContentAnalysis{HeuristicPenalty: 0},
	}
	score := r.ComputeScore()
	if score.Score < 90 {
		t.Errorf("expected a high score for a fully clean setup, got %d (%+v)", score.Score, score)
	}
	if score.Grade != "A" {
		t.Errorf("expected grade A, got %s", score.Grade)
	}
}

func TestComputeScorePenalizesRBLListing(t *testing.T) {
	clean := CheckResult{
		SPF: RecordCheck{Status: "ok"}, DKIM: RecordCheck{Status: "ok"}, DMARC: RecordCheck{Status: "ok"},
		RBL: []RBLResult{{List: "zen.spamhaus.org", Listed: false}},
	}
	listed := clean
	listed.RBL = []RBLResult{{List: "zen.spamhaus.org", Listed: true}}

	cleanScore := clean.ComputeScore().Score
	listedScore := listed.ComputeScore().Score
	if listedScore >= cleanScore {
		t.Errorf("expected RBL listing to reduce score: clean=%d listed=%d", cleanScore, listedScore)
	}
}
