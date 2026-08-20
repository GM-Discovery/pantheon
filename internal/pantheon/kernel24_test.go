package pantheon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKernel24CreatesBoundedFameWorkspaceAndCorpus(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL24", "expire"))
	defer d.Close()

	snapshot, err := d.aphroditeSnapshot(aphroditeProjectFameID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ActiveProject == nil || snapshot.ActiveProject.Name != "Project Fame" || snapshot.ActiveProject.Phase != aphroditePhaseObservation {
		t.Fatalf("active project=%+v", snapshot.ActiveProject)
	}
	if len(snapshot.FameFixtures) != 5 {
		t.Fatalf("initial Fame corpus=%d, want 5", len(snapshot.FameFixtures))
	}
	for _, fixture := range snapshot.FameFixtures {
		if !strings.HasPrefix(fixture.TranscriptPath, snapshot.ActiveProject.WorkspacePath) {
			t.Fatalf("fixture escaped workspace: %q", fixture.TranscriptPath)
		}
		if _, err := os.Stat(fixture.TranscriptPath); err != nil {
			t.Fatalf("fixture %s: %v", fixture.Title, err)
		}
	}
	if _, err := os.Stat(filepath.Join(snapshot.ActiveProject.WorkspacePath, "README.md")); err != nil {
		t.Fatal(err)
	}
}

func TestKernel24ProjectArtifactsAndLifecycleStayInWorkshop(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL24", "expire"))
	defer d.Close()

	artifact, err := d.createAphroditeArtifact(aphroditeProjectFameID, aphroditeArtifactRequest{Kind: "proposal", Title: "Fame concept pitch", Content: "Investigate how requests differ from jokes, quotations, and commitments."})
	if err != nil {
		t.Fatal(err)
	}
	project, err := d.loadAphroditeProject(aphroditeProjectFameID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(artifact.Path, project.WorkspacePath) {
		t.Fatalf("artifact escaped workspace: %q", artifact.Path)
	}
	if _, err := os.Stat(artifact.Path); err != nil {
		t.Fatal(err)
	}
	updated, err := d.db.Exec(`UPDATE aphrodite_projects SET phase = ?, state = ? WHERE project_id = ?`, aphroditePhaseConcept, aphroditeProjectActive, aphroditeProjectFameID)
	if err != nil {
		t.Fatal(err)
	}
	if count, _ := updated.RowsAffected(); count != 1 {
		t.Fatalf("updated project rows=%d", count)
	}
	if _, err := d.db.Exec(`SELECT 1 FROM aphrodite_projects WHERE project_id = ?`, aphroditeProjectFameID); err != nil {
		t.Fatal(err)
	}
	request, err := d.createAphroditeResearchRequest(aphroditeProjectFameID, aphroditeResearchRequestRequest{Question: "Which local diarization approaches preserve provenance?", SurfaceBrief: true})
	if err != nil {
		t.Fatal(err)
	}
	if request.Status != "proposed" || !request.SurfacedBrief || !strings.HasPrefix(request.ArtifactPath, project.WorkspacePath) {
		t.Fatalf("research request=%+v", request)
	}
}

func TestKernel24RumorFixtureCopiesEvidenceWithoutChangingOriginal(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL24", "expire"))
	defer d.Close()

	transcript := filepath.Join(t.TempDir(), "rumor-transcript.md")
	original := []byte("# Transcript\n\nGrant: Please remind me Thursday.\n")
	if err := os.WriteFile(transcript, original, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := d.db.Exec(`INSERT INTO rumor_sources (source_id, source_path, source_filename, source_hash, size, modified_at, detected_at, state, attempts, transcript_path, intake_state, alert_sent, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, "rumor-k24", "/audio/k24.wav", "k24.wav", "audio-hash", len(original), "2026-08-18T00:00:00Z", "2026-08-18T00:00:00Z", rumorStateCompleted, 1, transcript, "received", 0, "2026-08-18T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := d.createAphroditeFixture(aphroditeProjectFameID, aphroditeFixtureRequest{Title: "copied Rumor evidence", RumorSourceID: "rumor-k24", CitationStart: "00:00:00", CitationEnd: "00:00:08"})
	if err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(fixture.TranscriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(copied) != string(original) {
		t.Fatalf("copied transcript=%q want=%q", copied, original)
	}
	untouched, err := os.ReadFile(transcript)
	if err != nil {
		t.Fatal(err)
	}
	if string(untouched) != string(original) {
		t.Fatalf("Rumor original changed: %q", untouched)
	}
	if fixture.SourceID != "rumor-k24" || fixture.SourcePath != transcript || fixture.CitationStart != "00:00:00" {
		t.Fatalf("fixture provenance=%+v", fixture)
	}
}

func TestKernel24PanopticonSurfaceAndPromotionBoundary(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL24", "expire"))
	defer d.Close()

	request := httptest.NewRequest(http.MethodGet, "/v1/gui/aphrodite", nil)
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var snapshot aphroditeSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.ActiveProject == nil || len(snapshot.ApprovedInspectionRoutes) == 0 || len(snapshot.Boundaries) == 0 {
		t.Fatalf("Panopticon snapshot=%+v", snapshot)
	}
	promotion := httptest.NewRequest(http.MethodPost, "/v1/aphrodite/projects/project-fame/promote", strings.NewReader(`{}`))
	promotionResponse := httptest.NewRecorder()
	d.Handler().ServeHTTP(promotionResponse, promotion)
	if promotionResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("promotion route status=%d body=%s", promotionResponse.Code, promotionResponse.Body.String())
	}
}
