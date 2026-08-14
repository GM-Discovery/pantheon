package main

import "testing"

func TestRankWeavePassagesPrefersSocioOfficialRulesAndAppliesConfidenceFloor(t *testing.T) {
	passages := []weavePassage{
		{Path: "20 Projects/Socio/Writing/rules.md", Text: "Bread Standard governs the common good."},
		{Path: "20 Projects/Socio/Drafts/draft.md", Text: "Bread Standard governs an encounter."},
		{Path: "10 Operations/notes.md", Text: "Bread makes excellent toast."},
	}

	results := rankWeavePassages("bread standard", passages, 0.90)
	if len(results) != 2 {
		t.Fatalf("expected the two exact, high-confidence results; got %#v", results)
	}
	if results[0].Passage.Path != "20 Projects/Socio/Writing/rules.md" {
		t.Fatalf("official rules should rank first, got %q", results[0].Passage.Path)
	}
	if results[0].Confidence != 1 {
		t.Fatalf("expected exact-term confidence 1, got %v", results[0].Confidence)
	}
}
