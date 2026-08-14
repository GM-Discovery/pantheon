package main

import (
	"strings"
	"testing"
)

func TestRenderWeavePrimaryReviewProducesCitedMarkdown(t *testing.T) {
	out := renderWeavePrimaryReview(weaveIndex{Passages: []weavePassage{{Path: "20 Projects/a.md", Heading: "A", Line: 3, Text: "Evidence"}}})
	for _, want := range []string{"# Weave Primary Review", "[[20 Projects/a.md|20 Projects/a.md]]", "Primary source notes: 1", "Passage units: 1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}
}
