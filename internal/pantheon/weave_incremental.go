package pantheon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type weaveFingerprintState struct {
	Files map[string]string `json:"files"`
}
type weaveChangeReport struct {
	New        []string              `json:"new"`
	Changed    []string              `json:"changed"`
	Removed    []string              `json:"removed"`
	Unchanged  []string              `json:"unchanged"`
	Excluded   []string              `json:"excluded"`
	Contextual []string              `json:"contextual"`
	Current    weaveFingerprintState `json:"-"`
}

func buildWeaveChangeReport(vaultPath, statePath string) (weaveChangeReport, error) {
	previous := weaveFingerprintState{Files: map[string]string{}}
	if data, err := os.ReadFile(statePath); err == nil {
		if err := json.Unmarshal(data, &previous); err != nil {
			return weaveChangeReport{}, err
		}
	}
	report := weaveChangeReport{Current: weaveFingerprintState{Files: map[string]string{}}}
	err := filepath.WalkDir(vaultPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			report.Excluded = append(report.Excluded, path)
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
		rel, err := filepath.Rel(vaultPath, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		switch weaveCorpusDisposition(rel) {
		case "excluded":
			report.Excluded = append(report.Excluded, rel)
			return nil
		case "contextual":
			report.Contextual = append(report.Contextual, rel)
			return nil
		}
		f, err := openVaultMarkdown(path)
		if err != nil {
			report.Excluded = append(report.Excluded, rel)
			return nil
		}
		h := sha256.New()
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return err
		}
		fp := hex.EncodeToString(h.Sum(nil))
		report.Current.Files[rel] = fp
		if old, ok := previous.Files[rel]; !ok {
			report.New = append(report.New, rel)
		} else if old != fp {
			report.Changed = append(report.Changed, rel)
		} else {
			report.Unchanged = append(report.Unchanged, rel)
		}
		return nil
	})
	for path := range previous.Files {
		if _, present := report.Current.Files[path]; !present {
			report.Removed = append(report.Removed, path)
		}
	}
	for _, list := range [][]string{report.New, report.Changed, report.Removed, report.Unchanged, report.Excluded, report.Contextual} {
		sort.Strings(list)
	}
	return report, err
}
func writeWeaveFingerprintState(path string, state weaveFingerprintState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
