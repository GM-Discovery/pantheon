package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteConfiguredSubjectReviewsWritesBoundedReview(t *testing.T) {
	d := t.TempDir()
	config := filepath.Join(d, "subjects.json")
	if err := os.WriteFile(config, []byte(`{"subjects":[{"slug":"topic","query":"topic"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	idx := weaveIndex{Passages: []weavePassage{{Path: "a.md", Heading: "A", Line: 1, Text: "topic text", Terms: []string{"topic"}}}}
	if err := writeConfiguredSubjectReviews(config, d, idx); err != nil {
		t.Fatal(err)
	}
}

func TestWriteConfiguredSubjectReviewsRejectsTraversalSlug(t *testing.T) {
	d := t.TempDir()
	config := filepath.Join(d, "subjects.json")
	if err := os.WriteFile(config, []byte(`{"subjects":[{"slug":"../escape","query":"topic"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeConfiguredSubjectReviews(config, d, weaveIndex{}); err == nil {
		t.Fatal("expected traversal slug rejection")
	}
}
