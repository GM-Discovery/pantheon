package pantheon

import "testing"

func TestApplyNexusChunkDiffReplacesOnlyChangedPaths(t *testing.T) {
	old := nexusChunkInventory{Chunks: []nexusChunk{{Path: "same", Authorship: "nexus_user", Text: "old"}, {Path: "changed", Authorship: "nexus_user", Text: "old"}, {Path: "removed", Authorship: "nexus_agent", Text: "old"}}}
	rebuilt := nexusChunkInventory{Chunks: []nexusChunk{{Path: "changed", Authorship: "nexus_user", Text: "new"}, {Path: "new", Authorship: "nexus_agent", Text: "new"}}}
	got := applyNexusChunkDiff(old, rebuilt, fingerprintDiff{New: []string{"new"}, Changed: []string{"changed"}, Removed: []string{"removed"}})
	if len(got.Chunks) != 3 || got.Chunks[0].Path != "changed" || got.Chunks[1].Path != "new" || got.Chunks[2].Path != "same" {
		t.Fatalf("%#v", got.Chunks)
	}
}
