package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const pastResultLimit = 12

type pastMatch struct {
	path    string
	line    int
	excerpt string
}

// searchPast is Past's entire operational surface: bounded, read-only vault
// search. Its output cites the source file and line for human inspection.
func searchPast(vaultPath, query string) (string, error) {
	if strings.TrimSpace(vaultPath) == "" {
		return "", fmt.Errorf("vault_path is not configured; add it to ~/pantheon-runtime/config/pantheon.json")
	}

	info, err := os.Stat(vaultPath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("vault_path must be a directory")
	}

	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return "", fmt.Errorf("search query is empty")
	}

	var matches []pastMatch
	err = filepath.WalkDir(vaultPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || len(matches) >= pastResultLimit {
			return walkErr
		}
		if entry.Type()&fs.ModeSymlink != 0 || entry.IsDir() || !isVaultText(path) {
			return nil
		}

		file, err := openVaultMarkdown(path)
		if err != nil {
			return nil // An unreadable note is not a reason to stop searching Past.
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for line := 1; scanner.Scan() && len(matches) < pastResultLimit; line++ {
			text := scanner.Text()
			lower := strings.ToLower(text)
			if containsAll(lower, terms) {
				relative, err := filepath.Rel(vaultPath, path)
				if err != nil {
					relative = path
				}
				matches = append(matches, pastMatch{relative, line, strings.TrimSpace(text)})
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].path == matches[j].path {
			return matches[i].line < matches[j].line
		}
		return matches[i].path < matches[j].path
	})

	if len(matches) == 0 {
		return fmt.Sprintf("Past found no evidence in the vault for %q.\n", query), nil
	}

	var output strings.Builder
	fmt.Fprintf(&output, "Past found %d cited observation(s) for %q:\n", len(matches), query)
	for _, match := range matches {
		fmt.Fprintf(&output, "- %s:%d — %s\n", match.path, match.line, match.excerpt)
	}
	return output.String(), nil
}

func containsAll(text string, terms []string) bool {
	for _, term := range terms {
		if !strings.Contains(text, term) {
			return false
		}
	}
	return true
}

func isVaultText(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt", ".yaml", ".yml":
		return true
	default:
		return false
	}
}
