package main

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"
)

const weaveContextRadius = 2

func expandWeaveContext(vaultPath string, passage weavePassage) (string, error) {
	relative := filepath.Clean(passage.Path)
	if filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid Vault-relative passage path")
	}
	file, err := openVaultMarkdown(filepath.Join(vaultPath, relative))
	if err != nil {
		return "", err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if passage.Line < 1 || passage.Line > len(lines) {
		return "", fmt.Errorf("passage line is outside source")
	}
	start := max(0, passage.Line-1-weaveContextRadius)
	end := min(len(lines), passage.Line+weaveContextRadius)
	return strings.Join(lines[start:end], "\n"), nil
}
