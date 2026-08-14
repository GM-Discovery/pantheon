package main

import "testing"

func TestApplyPrimaryIndexDiffReplacesOnlyAffectedPaths(t *testing.T) {
	old := weaveIndex{Passages: []weavePassage{{Path: "same", Terms: []string{"same"}}, {Path: "changed", Terms: []string{"old"}}, {Path: "removed", Terms: []string{"gone"}}}}
	rebuilt := weaveIndex{Passages: []weavePassage{{Path: "same", Terms: []string{"same"}}, {Path: "changed", Terms: []string{"new"}}, {Path: "new", Terms: []string{"add"}}}}
	got := applyPrimaryIndexDiff(old, rebuilt, weaveChangeReport{New: []string{"new"}, Changed: []string{"changed"}, Removed: []string{"removed"}})
	if len(got.Passages) != 3 || got.PrimaryTermCounts["same"] != 1 || got.PrimaryTermCounts["new"] != 1 || got.PrimaryTermCounts["add"] != 1 {
		t.Fatalf("%#v", got)
	}
}
