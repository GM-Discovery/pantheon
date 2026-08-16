package pantheon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildPrimaryWeaveIndexExcludesContextualAndExcluded(t *testing.T) {
	v := t.TempDir()
	for p, s := range map[string]string{"20 Projects/a.md": "# A\nprimary", "Nexus/a.md": "# N\ncontext", "00 Pantheon Core/Weave/a.md": "# W\nexcluded"} {
		f := filepath.Join(v, p)
		if err := os.MkdirAll(filepath.Dir(f), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	idx, err := buildPrimaryWeaveIndex(v)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Passages) != 1 || idx.Passages[0].Path != "20 Projects/a.md" {
		t.Fatalf("%#v", idx.Passages)
	}
}
