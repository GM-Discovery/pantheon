package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildNexusChunkInventoryUsesExplicitAuthorship(t *testing.T) {
	vault := t.TempDir()
	p := filepath.Join(vault, "Nexus", "chat.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(">[!nexus_user]\n> Grant words\n>[!nexus_agent]\n> Assistant words\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := buildNexusChunkInventory(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Chunks) != 2 || got.UserChunkCount != 1 || got.AgentChunkCount != 1 {
		t.Fatalf("%#v", got)
	}
}
