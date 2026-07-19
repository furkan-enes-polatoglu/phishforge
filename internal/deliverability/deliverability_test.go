package deliverability

import "testing"

func TestImgMissingAlt(t *testing.T) {
	if !imgMissingAlt(`<img src="x.png">`) {
		t.Error("expected missing alt to be detected")
	}
	if imgMissingAlt(`<img src="x.png" alt="ok">`) {
		t.Error("alt present should not be flagged")
	}
	if imgMissingAlt(`<p>no images</p>`) {
		t.Error("no img tags should not be flagged")
	}
}

func TestLintHTMLEmpty(t *testing.T) {
	if got := LintHTML(""); len(got) == 0 {
		t.Error("empty HTML should return a hint")
	}
}
