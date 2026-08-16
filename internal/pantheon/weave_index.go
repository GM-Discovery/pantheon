package pantheon

import (
	"bufio"
	"io/fs"
	"path/filepath"
	"strings"
)

type weavePassage struct {
	Path    string   `json:"path"`
	Domain  string   `json:"domain"`
	Heading string   `json:"heading"`
	Line    int      `json:"line"`
	Text    string   `json:"text"`
	Terms   []string `json:"terms"`
}

type weaveIndex struct {
	Passages          []weavePassage `json:"passages"`
	PrimaryTermCounts map[string]int `json:"primary_term_counts"`
}

var weavePrimaryDomains = map[string]bool{"00 Pantheon Core": true, "10 Operations": true, "20 Projects": true, "30 Procedures": true, "90 Archive": true}

func buildWeaveIndex(vaultPath string) (weaveIndex, error) {
	index := weaveIndex{PrimaryTermCounts: map[string]int{}}
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
		parts := strings.Split(relative, string(filepath.Separator))
		domain := parts[0]
		heading := strings.TrimSuffix(filepath.Base(relative), filepath.Ext(relative))
		start := 1
		var block []string
		emit := func() {
			text := strings.TrimSpace(strings.Join(block, "\n"))
			if text == "" {
				return
			}
			terms := connectionTerms(text)
			p := weavePassage{Path: relative, Domain: domain, Heading: heading, Line: start, Text: connectionExcerpt(text), Terms: terms}
			index.Passages = append(index.Passages, p)
			if weavePrimaryDomains[domain] {
				seen := map[string]bool{}
				for _, term := range terms {
					if !seen[term] {
						index.PrimaryTermCounts[term]++
						seen[term] = true
					}
				}
			}
		}
		s := bufio.NewScanner(file)
		s.Buffer(make([]byte, 64*1024), 1024*1024)
		for line := 1; s.Scan(); line++ {
			text := strings.TrimSpace(s.Text())
			if strings.HasPrefix(text, "#") {
				emit()
				heading = strings.TrimSpace(strings.TrimLeft(text, "#"))
				start = line
				block = nil
				continue
			}
			block = append(block, text)
		}
		emit()
		return s.Err()
	})
	return index, err
}
