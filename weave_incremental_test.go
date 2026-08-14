package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildWeaveChangeReportIdentifiesNewAndChangedMarkdown(t *testing.T) {
	vault := t.TempDir()
	state := filepath.Join(t.TempDir(), "weave-state.json")
	first := filepath.Join(vault, "20 Projects", "one.md")
	if err := os.MkdirAll(filepath.Dir(first), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := buildWeaveChangeReport(vault, state)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.New) != 1 || report.New[0] != "20 Projects/one.md" {
		t.Fatalf("new=%#v", report.New)
	}
	if err := writeWeaveFingerprintState(state, report.Current); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err = buildWeaveChangeReport(vault, state)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Changed) != 1 || report.Changed[0] != "20 Projects/one.md" {
		t.Fatalf("changed=%#v", report.Changed)
	}
}
