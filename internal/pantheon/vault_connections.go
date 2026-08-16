package pantheon

import (
	"bufio"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const (
	connectionTopTerms     = 12
	connectionEvidence     = 2
	connectionExcerptLimit = 180
)

type termEvidence struct {
	Path string
	Line int
	Text string
}

type vaultConnectionIndex struct {
	NoteCount int
	Terms     map[string]int
	Evidence  map[string][]termEvidence
}

func buildVaultConnectionIndex(vaultPath string) (vaultConnectionIndex, error) {
	index := vaultConnectionIndex{Terms: map[string]int{}, Evidence: map[string][]termEvidence{}}
	err := filepath.WalkDir(vaultPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".obsidian" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}
		file, err := openVaultMarkdown(path)
		if err != nil {
			return err
		}
		defer file.Close()
		relative, err := filepath.Rel(vaultPath, path)
		if err != nil {
			return err
		}
		index.NoteCount++
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for line := 1; scanner.Scan(); line++ {
			text := strings.TrimSpace(scanner.Text())
			for _, term := range connectionTerms(text) {
				index.Terms[term]++
				if len(index.Evidence[term]) < connectionEvidence {
					index.Evidence[term] = append(index.Evidence[term], termEvidence{Path: relative, Line: line, Text: connectionExcerpt(text)})
				}
			}
		}
		return scanner.Err()
	})
	return index, err
}

func connectionExcerpt(text string) string {
	runes := []rune(text)
	if len(runes) <= connectionExcerptLimit {
		return text
	}
	return string(runes[:connectionExcerptLimit]) + "…"
}

func connectionTerms(text string) []string {
	var terms []string
	var word strings.Builder
	flush := func() {
		term := strings.ToLower(word.String())
		word.Reset()
		if len([]rune(term)) >= 5 && !connectionStopWords[term] {
			terms = append(terms, term)
		}
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			word.WriteRune(r)
		} else if word.Len() > 0 {
			flush()
		}
	}
	if word.Len() > 0 {
		flush()
	}
	return terms
}

var connectionStopWords = map[string]bool{
	"about": true, "after": true, "again": true, "also": true, "because": true, "before": true, "between": true, "could": true, "every": true, "first": true, "from": true, "have": true, "into": true, "just": true, "more": true, "most": true, "other": true, "over": true, "should": true, "their": true, "there": true, "these": true, "they": true, "this": true, "through": true, "under": true, "using": true, "what": true, "when": true, "where": true, "which": true, "while": true, "with": true, "would": true, "your": true,
}

func connectionTop(index vaultConnectionIndex) []string {
	terms := make([]string, 0, len(index.Terms))
	for term := range index.Terms {
		terms = append(terms, term)
	}
	sort.Slice(terms, func(i, j int) bool {
		if index.Terms[terms[i]] == index.Terms[terms[j]] {
			return terms[i] < terms[j]
		}
		return index.Terms[terms[i]] > index.Terms[terms[j]]
	})
	if len(terms) > connectionTopTerms {
		terms = terms[:connectionTopTerms]
	}
	return terms
}

func connectionEvidenceReport(index vaultConnectionIndex) string {
	var report strings.Builder
	for _, term := range connectionTop(index) {
		fmt.Fprintf(&report, "## %s — %d occurrence(s)\n", term, index.Terms[term])
		for _, evidence := range index.Evidence[term] {
			fmt.Fprintf(&report, "- %s:%d — %s\n", evidence.Path, evidence.Line, evidence.Text)
		}
		report.WriteString("\n")
	}
	return report.String()
}
