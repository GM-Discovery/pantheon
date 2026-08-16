package pantheon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildNexusFingerprintStateIncludesOnlyNexusMarkdown(t *testing.T) {
	vault := t.TempDir()
	p := filepath.Join(vault, "Nexus", "chat.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := buildNexusFingerprintState(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Files) != 1 || got.Files["Nexus/chat.md"] == "" {
		t.Fatalf("%#v", got)
	}
}
