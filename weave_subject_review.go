package main

import (
	"fmt"
	"strings"
)

func renderWeaveQueryReview(query string, results []weaveRetrievalResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Weave Subject Review: %s\n\n> Relevance-ranked, cited primary-source excerpts. Confidence is lexical coverage, not factual certainty.\n\n", query)
	for _, r := range results {
		p := r.Passage
		fmt.Fprintf(&b, "## [[%s|%s]] — %s\n\n- line %d · confidence: %.2f\n\n%s\n\n", p.Path, p.Path, p.Heading, p.Line, r.Confidence, p.Text)
	}
	return b.String()
}
