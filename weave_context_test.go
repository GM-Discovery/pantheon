package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandWeaveContextReturnsBoundedSourceWindow(t *testing.T) {
	vault := t.TempDir()
	path := filepath.Join(vault, "note.md")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\nfour\nfive\nsix\nseven\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := expandWeaveContext(vault, weavePassage{Path: "note.md", Line: 4})
	if err != nil {
		t.Fatal(err)
	}
	if got != "two\nthree\nfour\nfive\nsix" {
		t.Fatalf("context=%q", got)
	}
}
