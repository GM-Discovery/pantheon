package main

import (
	"strings"
)

const nexusChunkMaxChars = 900

type nexusChunk struct {
	Path       string `json:"path"`
	Authorship string `json:"authorship"`
	Text       string `json:"text"`
}

func chunkNexusConversation(path, source string) []nexusChunk {
	var chunks []nexusChunk
	var authorship string
	var lines []string
	emit := func() {
		text := strings.TrimSpace(strings.Join(lines, "\n"))
		if authorship == "" || text == "" {
			return
		}
		for len(text) > nexusChunkMaxChars {
			end := strings.LastIndex(text[:nexusChunkMaxChars], " ")
			if end < 1 {
				end = nexusChunkMaxChars
			}
			chunks = append(chunks, nexusChunk{Path: path, Authorship: authorship, Text: strings.TrimSpace(text[:end])})
			text = strings.TrimSpace(text[end:])
		}
		chunks = append(chunks, nexusChunk{Path: path, Authorship: authorship, Text: text})
	}
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ">[!nexus_user]") || strings.HasPrefix(trimmed, ">[!nexus_agent]") {
			emit()
			lines = nil
			if strings.HasPrefix(trimmed, ">[!nexus_user]") {
				authorship = "nexus_user"
			} else {
				authorship = "nexus_agent"
			}
			continue
		}
		if authorship != "" {
			lines = append(lines, strings.TrimSpace(strings.TrimPrefix(trimmed, ">")))
		}
	}
	emit()
	return chunks
}
