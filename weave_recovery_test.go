package main

import "testing"

func TestFullRebuildDiffMarksEverySourceNew(t *testing.T) {
	d := fullRebuildDiff(weaveFingerprintState{Files: map[string]string{"b.md": "2", "a.md": "1"}})
	if len(d.New) != 2 || d.New[0] != "a.md" || d.New[1] != "b.md" {
		t.Fatalf("%#v", d)
	}
}
