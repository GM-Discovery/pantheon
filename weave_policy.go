package main

import "strings"

func weaveCorpusDisposition(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if strings.HasPrefix(path, "Nexus/") || strings.HasPrefix(path, "01 User Identity/UareAIreports/") {
		return "contextual"
	}
	for _, excluded := range []string{
		"00 Pantheon Core/Weave/",
		"10 Operations/Situation Briefs/",
		"10 Operations/Email Triage/",
		"20 Projects/Writings Inbox/",
	} {
		if strings.HasPrefix(path, excluded) {
			return "excluded"
		}
	}
	return "primary"
}
