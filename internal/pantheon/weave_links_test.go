package pantheon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWeaveLinkPreviewApplyRepeatAndRollback(t *testing.T) {
	vault := t.TempDir()
	writeFixtureNote(t, vault, "Nexus/Conversations/chatgpt/2026/01/Character Creation Guide.md", "# Character Creation\n\nThe Bread Standard federation uses trust points and transparent governance.\n\nLate passage: the validator council applies the same trust points to character responsibilities.\n")
	target := "20 Projects/Bread Standard/Constitution.md"
	writeFixtureNote(t, vault, target, "# Bread Standard\n\nThe Bread Standard uses transparent governance and federated trust points to protect human dignity.\n")
	state := t.TempDir()
	before, err := os.ReadFile(filepath.Join(vault, filepath.FromSlash("Nexus/Conversations/chatgpt/2026/01/Character Creation Guide.md")))
	if err != nil {
		t.Fatal(err)
	}
	result, err := generateWeaveLinkPreview(weaveLinkOptions{Limit: 1, OutputDir: state, VaultPath: vault})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Manifest.Proposals) == 0 {
		t.Fatalf("expected a two-sided evidence proposal")
	}
	if !mapsEqual(result.Manifest.ScopeHashes, result.Manifest.ScopeHashesAfter) {
		t.Fatal("preview changed scoped hashes")
	}

	journalPath, err := applyWeaveLinkManifest(result.ManifestPath, state, []string{result.Manifest.Proposals[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(vault, filepath.FromSlash(result.Manifest.Proposals[0].SourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), weaveLinksBegin) || !strings.Contains(string(after), "[[20 Projects/Bread Standard/Constitution.md|Constitution]]") {
		t.Fatalf("generated link block missing: %s", after)
	}
	if !strings.Contains(string(after), "Late passage") {
		t.Fatal("late source passage was not preserved")
	}

	if _, err := applyWeaveLinkManifest(result.ManifestPath, state, []string{result.Manifest.Proposals[0].ID}); err != nil {
		t.Fatalf("repeat apply should be a no-op: %v", err)
	}
	if err := rollbackWeaveLinkJournal(journalPath); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(filepath.Join(vault, filepath.FromSlash(result.Manifest.Proposals[0].SourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != string(before) {
		t.Fatal("rollback did not restore the original bytes")
	}
}

func TestWeaveLinkApplyRefusesStaleTargetAndHumanEdit(t *testing.T) {
	vault := t.TempDir()
	source := "Nexus/Conversations/chatgpt/2026/01/Source.md"
	target := "20 Projects/Federated Governance/Topic.md"
	writeFixtureNote(t, vault, source, "# Source\n\nFederated governance and validator trust points matter.\n")
	writeFixtureNote(t, vault, target, "# Topic\n\nFederated governance uses validator trust points.\n")
	state := t.TempDir()
	result, err := generateWeaveLinkPreview(weaveLinkOptions{Limit: 1, OutputDir: state, VaultPath: vault})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Manifest.Proposals) == 0 {
		t.Fatal("expected proposal")
	}
	if err := os.WriteFile(filepath.Join(vault, filepath.FromSlash(target)), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := applyWeaveLinkManifest(result.ManifestPath, state, []string{result.Manifest.Proposals[0].ID}); err == nil || !strings.Contains(err.Error(), "target hash changed") {
		t.Fatalf("expected stale target refusal, got %v", err)
	}
	writeFixtureNote(t, vault, target, "# Topic\n\nFederated governance uses validator trust points.\n")

	result, err = generateWeaveLinkPreview(weaveLinkOptions{Limit: 1, OutputDir: state, VaultPath: vault})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Manifest.Proposals) == 0 {
		t.Fatal("expected regenerated proposal")
	}
	path := filepath.Join(vault, filepath.FromSlash(source))
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("\nHuman edit.\n")
	_ = f.Close()
	if _, err := applyWeaveLinkManifest(result.ManifestPath, state, []string{result.Manifest.Proposals[0].ID}); err == nil {
		t.Fatal("expected stale source refusal")
	}
}

func TestWeaveLinkManifestRoundTrip(t *testing.T) {
	manifest := weaveLinkManifest{Version: weaveLinksVersion, ScopeHashes: map[string]string{"a.md": "1"}, ScopeHashesAfter: map[string]string{"a.md": "1"}}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var decoded weaveLinkManifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Version != weaveLinksVersion || !mapsEqual(decoded.ScopeHashes, decoded.ScopeHashesAfter) {
		t.Fatalf("manifest round trip failed: %#v", decoded)
	}
}

func TestSegmentNexusConversationUsesUserTurns(t *testing.T) {
	content := `---
nexus: nexus-ai-chat-importer
---
# Title: Bread Conversation

>[!nexus_user] **User** - 06/01/2026 at 9:00 AM
> I want the Bread Standard to connect trust points, dignity, and participatory governance.
<!-- UID: user-1 -->
>[!nexus_agent] **Assistant** - 06/01/2026 at 9:00 AM
> Here is a generated assistant framework about completely unrelated machinery and telemetry.
<!-- UID: assistant-1 -->
>[!nexus_user] **User** - 06/01/2026 at 9:05 AM
> Keep the Socio setting connected as a practical expression of that governance.
<!-- UID: user-2 -->
`
	passages := segmentWeaveLinkMarkdown(content)
	if len(passages) != 2 {
		t.Fatalf("expected one passage per user turn, got %d: %#v", len(passages), passages)
	}
	if !strings.Contains(passages[0].Text, "Bread Standard") || !strings.Contains(passages[1].Text, "Socio setting") {
		t.Fatalf("user turn text was not preserved: %#v", passages)
	}
	for _, passage := range passages {
		if strings.Contains(passage.Text, "machinery") || passage.Line == 0 {
			t.Fatalf("assistant text leaked or line citation missing: %#v", passage)
		}
	}
}

func TestWeaveLinkRunProcessesDurableBatches(t *testing.T) {
	vault := t.TempDir()
	for i := 0; i < 5; i++ {
		name := filepath.Join("Nexus/Conversations/chatgpt/2026/01", "Source "+string(rune('A'+i))+".md")
		writeFixtureNote(t, vault, name, "# Source\n\nFederated governance and validator trust points matter.\n")
	}
	writeFixtureNote(t, vault, "20 Projects/Federated Governance/Topic.md", "# Topic\n\nFederated governance uses validator trust points.\n")
	stateDir := t.TempDir()
	statePath := filepath.Join(stateDir, "run.json")
	if err := runWeaveLinkBatches(statePath, stateDir, vault, "", "", 2, 1, 5, false); err != nil {
		t.Fatal(err)
	}
	state, err := loadWeaveLinkRunState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != "completed" || state.NextOffset != 5 || state.CompletedBatches != 3 {
		t.Fatalf("unexpected completed run state: %#v", state)
	}
	if state.CurrentManifest != "" {
		t.Fatalf("completed run retained pending manifest: %#v", state)
	}
}

func writeFixtureNote(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
