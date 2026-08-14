package main

import "testing"

func TestWeaveCorpusDispositionSeparatesPrimaryContextAndExcludedSources(t *testing.T) {
	cases := map[string]string{
		"20 Projects/Bread Standard/constitution.md": "primary",
		"01 User Identity/UareAIreports/Identity.md": "contextual",
		"Nexus/Conversations/chatgpt/example.md":     "contextual",
		"00 Pantheon Core/Weave/review.md":           "excluded",
		"10 Operations/Situation Briefs/today.md":    "excluded",
		"10 Operations/Email Triage/today.md":        "excluded",
		"20 Projects/Writings Inbox/queue.md":        "excluded",
	}
	for path, want := range cases {
		if got := weaveCorpusDisposition(path); got != want {
			t.Errorf("%s: got %s want %s", path, got, want)
		}
	}
}
