package pantheon

import (
	"strings"
	"testing"
)

func TestRenderWeaveQueryReviewShowsOnlyRankedCitations(t *testing.T) {
	out := renderWeaveQueryReview("bread", []weaveRetrievalResult{{Passage: weavePassage{Path: "20 Projects/a.md", Heading: "A", Line: 3, Text: "Bread evidence"}, Confidence: 1}})
	for _, want := range []string{"# Weave Subject Review: bread", "[[20 Projects/a.md|20 Projects/a.md]]", "Bread evidence", "confidence: 1.00"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q", want)
		}
	}
}
