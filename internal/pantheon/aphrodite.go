package pantheon

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Aphrodite is deliberately a project/workshop boundary in Kernel 24. These
// records can describe and construct candidates, but none of these interfaces
// can promote a candidate, alter production code, install a capability, or
// replace the running daemon.
const (
	aphroditeProjectFameID  = "project-fame"
	aphroditeProjectActive  = "active"
	aphroditeProjectPaused  = "paused"
	aphroditeProjectStopped = "stopped"

	aphroditePhaseObservation            = "observation"
	aphroditePhaseConcept                = "concept"
	aphroditePhaseResearch               = "research"
	aphroditePhaseDesign                 = "design"
	aphroditePhasePrototype              = "prototype"
	aphroditePhaseValidation             = "validation"
	aphroditePhaseImplementationProposal = "implementation_proposal"
	aphroditePhaseCandidateDiff          = "candidate_diff"
	aphroditePhaseAuthorizedIntegration  = "authorized_integration"
)

var aphroditePhases = map[string]bool{
	aphroditePhaseObservation: true, aphroditePhaseConcept: true,
	aphroditePhaseResearch: true, aphroditePhaseDesign: true,
	aphroditePhasePrototype: true, aphroditePhaseValidation: true,
	aphroditePhaseImplementationProposal: true, aphroditePhaseCandidateDiff: true,
	aphroditePhaseAuthorizedIntegration: true,
}

var aphroditeProjectStates = map[string]bool{
	aphroditeProjectActive: true, aphroditeProjectPaused: true, aphroditeProjectStopped: true,
}

var aphroditeArtifactKinds = map[string]bool{
	"observation": true, "research_note": true, "experiment": true,
	"test_fixture": true, "design": true, "proposal": true,
	"candidate": true, "rejected": true,
}

type aphroditeProject struct {
	ProjectID        string   `json:"project_id"`
	Name             string   `json:"name"`
	Purpose          string   `json:"purpose"`
	Phase            string   `json:"phase"`
	State            string   `json:"state"`
	WorkspacePath    string   `json:"workspace_path"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
	LastResult       string   `json:"last_result,omitempty"`
	PendingQuestions []string `json:"pending_questions,omitempty"`
	PendingApprovals []string `json:"pending_approvals,omitempty"`
}

type aphroditeArtifact struct {
	ArtifactID string `json:"artifact_id"`
	ProjectID  string `json:"project_id"`
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

type aphroditeResearchRequest struct {
	RequestID     string `json:"request_id"`
	ProjectID     string `json:"project_id"`
	Question      string `json:"question"`
	Rationale     string `json:"rationale,omitempty"`
	Status        string `json:"status"`
	SurfacedBrief bool   `json:"surfaced_in_brief"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	ArtifactPath  string `json:"artifact_path,omitempty"`
}

type aphroditeFameFixture struct {
	FixtureID      string `json:"fixture_id"`
	ProjectID      string `json:"project_id"`
	Title          string `json:"title"`
	SourceKind     string `json:"source_kind"`
	SourceID       string `json:"source_id,omitempty"`
	SourcePath     string `json:"source_path,omitempty"`
	TranscriptPath string `json:"transcript_path"`
	CitationStart  string `json:"citation_start,omitempty"`
	CitationEnd    string `json:"citation_end,omitempty"`
	SHA256         string `json:"sha256"`
	CreatedAt      string `json:"created_at"`
}

type aphroditeSnapshot struct {
	GeneratedAt              string                     `json:"generated_at"`
	WorkspaceRoot            string                     `json:"workspace_root"`
	Projects                 []aphroditeProject         `json:"projects"`
	ActiveProject            *aphroditeProject          `json:"active_project,omitempty"`
	Artifacts                []aphroditeArtifact        `json:"artifacts"`
	ResearchRequests         []aphroditeResearchRequest `json:"research_requests"`
	FameFixtures             []aphroditeFameFixture     `json:"fame_fixtures"`
	ApprovedInspectionRoutes []string                   `json:"approved_inspection_routes"`
	CapabilityInventory      []bifrostCapability        `json:"capability_inventory"`
	Boundaries               []string                   `json:"boundaries"`
}

type aphroditeProjectRequest struct {
	ProjectID string `json:"project_id,omitempty"`
	Name      string `json:"name"`
	Purpose   string `json:"purpose"`
	Phase     string `json:"phase,omitempty"`
}

type aphroditePhaseRequest struct {
	Phase            string   `json:"phase"`
	State            string   `json:"state,omitempty"`
	LastResult       string   `json:"last_result,omitempty"`
	PendingQuestions []string `json:"pending_questions,omitempty"`
	PendingApprovals []string `json:"pending_approvals,omitempty"`
}

type aphroditeArtifactRequest struct {
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Status  string `json:"status,omitempty"`
}

type aphroditeResearchRequestRequest struct {
	Question     string `json:"question"`
	Rationale    string `json:"rationale,omitempty"`
	SurfaceBrief bool   `json:"surface_in_brief"`
}

type aphroditeFixtureRequest struct {
	Title         string `json:"title"`
	Transcript    string `json:"transcript,omitempty"`
	RumorSourceID string `json:"rumor_source_id,omitempty"`
	CitationStart string `json:"citation_start,omitempty"`
	CitationEnd   string `json:"citation_end,omitempty"`
}

func initializeAphroditeSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS aphrodite_projects (
			project_id TEXT PRIMARY KEY, name TEXT NOT NULL, purpose TEXT NOT NULL,
			phase TEXT NOT NULL, state TEXT NOT NULL, workspace_path TEXT NOT NULL,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL, last_result TEXT,
			pending_questions TEXT NOT NULL DEFAULT '[]', pending_approvals TEXT NOT NULL DEFAULT '[]'
		)`,
		`CREATE TABLE IF NOT EXISTS aphrodite_artifacts (
			artifact_id TEXT PRIMARY KEY, project_id TEXT NOT NULL, kind TEXT NOT NULL,
			title TEXT NOT NULL, path TEXT NOT NULL, sha256 TEXT NOT NULL,
			status TEXT NOT NULL, created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS aphrodite_artifacts_project_idx ON aphrodite_artifacts(project_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS aphrodite_research_requests (
			request_id TEXT PRIMARY KEY, project_id TEXT NOT NULL, question TEXT NOT NULL,
			rationale TEXT, status TEXT NOT NULL, surfaced_in_brief INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL, artifact_path TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS aphrodite_research_project_idx ON aphrodite_research_requests(project_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS aphrodite_fame_fixtures (
			fixture_id TEXT PRIMARY KEY, project_id TEXT NOT NULL, title TEXT NOT NULL,
			source_kind TEXT NOT NULL, source_id TEXT, source_path TEXT,
			transcript_path TEXT NOT NULL, citation_start TEXT, citation_end TEXT,
			sha256 TEXT NOT NULL, created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS aphrodite_fame_project_idx ON aphrodite_fame_fixtures(project_id, created_at DESC)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func aphroditeWorkspaceRoot(databasePath string) string {
	return filepath.Join(filepath.Dir(databasePath), "aphrodite")
}

func aphroditeProjectWorkspace(databasePath, projectID string) string {
	return filepath.Join(aphroditeWorkspaceRoot(databasePath), "projects", projectID)
}

func (d *daemon) ensureAphroditeDefaults() error {
	if err := initializeAphroditeSchema(d.db); err != nil {
		return err
	}
	root := aphroditeWorkspaceRoot(d.databasePath)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	projectPath := aphroditeProjectWorkspace(d.databasePath, aphroditeProjectFameID)
	for _, path := range []string{projectPath, filepath.Join(projectPath, "inputs", "fame"), filepath.Join(projectPath, "research"), filepath.Join(projectPath, "artifacts")} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := d.db.Exec(`INSERT OR IGNORE INTO aphrodite_projects (project_id, name, purpose, phase, state, workspace_path, created_at, updated_at, pending_questions, pending_approvals) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '[]', '[]')`, aphroditeProjectFameID, "Project Fame", "Discover and validate Pantheon's human-context interpretation layer from Rumor evidence.", aphroditePhaseObservation, aphroditeProjectActive, projectPath, now, now)
	if err != nil {
		return err
	}
	readme := filepath.Join(projectPath, "README.md")
	if _, statErr := os.Stat(readme); errors.Is(statErr, os.ErrNotExist) {
		content := "# Aphrodite workspace — Project Fame\n\nThis is a bounded candidate workshop. Rumor evidence is copied into `inputs/fame`; originals are never edited.\n\nAphrodite may produce observations, research, experiments, proposals, and candidate artifacts here. Nothing in this workspace promotes code, grants authority, installs a capability, or replaces the running Pantheon daemon.\n"
		if err := os.WriteFile(readme, []byte(content), 0o600); err != nil {
			return err
		}
	}
	var fixtureCount int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM aphrodite_fame_fixtures WHERE project_id = ?`, aphroditeProjectFameID).Scan(&fixtureCount); err != nil {
		return err
	}
	if fixtureCount == 0 {
		fixtures := []struct{ title, text string }{
			{"explicit request", "[00:00:01] Grant: Please remind me to call the clinic Thursday at three."},
			{"joking request", "[00:00:04] Friend: You should put that on my calendar.\n[00:00:06] Grant: Ha, only if you can make time travel work."},
			{"uncertain commitment", "[00:00:09] Grant: I can probably get that done sometime next week."},
			{"quoted speech", "[00:00:12] Friend: She said, 'Please send the report by Friday,' but I do not know whether she meant it."},
			{"nothing actionable", "[00:00:15] Grant: The rain sounds nice today.\n[00:00:18] Friend: It does."},
		}
		for _, fixture := range fixtures {
			if _, err := d.createAphroditeFixture(aphroditeProjectFameID, aphroditeFixtureRequest{Title: fixture.title, Transcript: fixture.text}); err != nil {
				return err
			}
		}
	}
	return nil
}

func decodeStringList(raw string) []string {
	var result []string
	if json.Unmarshal([]byte(raw), &result) != nil {
		return []string{}
	}
	return result
}

func scanAphroditeProject(scanner interface{ Scan(...any) error }) (aphroditeProject, error) {
	var project aphroditeProject
	var questions, approvals string
	err := scanner.Scan(&project.ProjectID, &project.Name, &project.Purpose, &project.Phase, &project.State, &project.WorkspacePath, &project.CreatedAt, &project.UpdatedAt, &project.LastResult, &questions, &approvals)
	project.PendingQuestions = decodeStringList(questions)
	project.PendingApprovals = decodeStringList(approvals)
	return project, err
}

func (d *daemon) loadAphroditeProject(id string) (aphroditeProject, error) {
	return scanAphroditeProject(d.db.QueryRow(`SELECT project_id, name, purpose, phase, state, workspace_path, created_at, updated_at, COALESCE(last_result, ''), pending_questions, pending_approvals FROM aphrodite_projects WHERE project_id = ?`, id))
}

func (d *daemon) listAphroditeProjects() ([]aphroditeProject, error) {
	rows, err := d.db.Query(`SELECT project_id, name, purpose, phase, state, workspace_path, created_at, updated_at, COALESCE(last_result, ''), pending_questions, pending_approvals FROM aphrodite_projects ORDER BY updated_at DESC, project_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	projects := []aphroditeProject{}
	for rows.Next() {
		project, scanErr := scanAphroditeProject(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

func aphroditeWorkspaceFile(project aphroditeProject, relative string) (string, error) {
	relative = filepath.Clean(relative)
	if relative == "." || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("Aphrodite artifact path escapes the project workspace")
	}
	return filepath.Join(project.WorkspacePath, relative), nil
}

func aphroditeSafeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "artifact"
	}
	if len(value) > 60 {
		value = value[:60]
	}
	return value
}

func writeAphroditeWorkspaceFile(path string, content []byte) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	digest := sha256.Sum256(content)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest[:]), nil
}

func (d *daemon) createAphroditeArtifact(projectID string, request aphroditeArtifactRequest) (aphroditeArtifact, error) {
	project, err := d.loadAphroditeProject(projectID)
	if err != nil {
		return aphroditeArtifact{}, err
	}
	request.Kind = strings.TrimSpace(request.Kind)
	request.Title = strings.TrimSpace(request.Title)
	if !aphroditeArtifactKinds[request.Kind] {
		return aphroditeArtifact{}, fmt.Errorf("unsupported Aphrodite artifact kind %q", request.Kind)
	}
	if request.Title == "" || strings.TrimSpace(request.Content) == "" {
		return aphroditeArtifact{}, errors.New("artifact title and content are required")
	}
	if request.Status == "" {
		request.Status = "candidate"
	}
	id, err := newLocalID("aph-artifact-")
	if err != nil {
		return aphroditeArtifact{}, err
	}
	path, err := aphroditeWorkspaceFile(project, filepath.Join("artifacts", id+"-"+aphroditeSafeName(request.Title)+".md"))
	if err != nil {
		return aphroditeArtifact{}, err
	}
	digest, err := writeAphroditeWorkspaceFile(path, []byte(request.Content+"\n"))
	if err != nil {
		return aphroditeArtifact{}, err
	}
	created := time.Now().UTC().Format(time.RFC3339Nano)
	artifact := aphroditeArtifact{ArtifactID: id, ProjectID: projectID, Kind: request.Kind, Title: request.Title, Path: path, SHA256: digest, Status: request.Status, CreatedAt: created}
	_, err = d.db.Exec(`INSERT INTO aphrodite_artifacts (artifact_id, project_id, kind, title, path, sha256, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, artifact.ArtifactID, artifact.ProjectID, artifact.Kind, artifact.Title, artifact.Path, artifact.SHA256, artifact.Status, artifact.CreatedAt)
	return artifact, err
}

func (d *daemon) createAphroditeResearchRequest(projectID string, request aphroditeResearchRequestRequest) (aphroditeResearchRequest, error) {
	if _, err := d.loadAphroditeProject(projectID); err != nil {
		return aphroditeResearchRequest{}, err
	}
	request.Question = strings.TrimSpace(request.Question)
	if request.Question == "" {
		return aphroditeResearchRequest{}, errors.New("research question is required")
	}
	id, err := newLocalID("aph-research-")
	if err != nil {
		return aphroditeResearchRequest{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	path := filepath.Join(aphroditeProjectWorkspace(d.databasePath, projectID), "research", id+".md")
	content := fmt.Sprintf("# Apollo research request\n\n- Request ID: `%s`\n- Project: `%s`\n- Status: proposed\n- Surface in Situation Brief: %t\n\n## Question\n\n%s\n\n## Rationale\n\n%s\n\n## Boundary\n\nThis is a research request, not an order to install software, access a new source, or change production.\n", id, projectID, request.SurfaceBrief, request.Question, strings.TrimSpace(request.Rationale))
	if _, err := writeAphroditeWorkspaceFile(path, []byte(content)); err != nil {
		return aphroditeResearchRequest{}, err
	}
	result := aphroditeResearchRequest{RequestID: id, ProjectID: projectID, Question: request.Question, Rationale: strings.TrimSpace(request.Rationale), Status: "proposed", SurfacedBrief: request.SurfaceBrief, CreatedAt: now, UpdatedAt: now, ArtifactPath: path}
	_, err = d.db.Exec(`INSERT INTO aphrodite_research_requests (request_id, project_id, question, rationale, status, surfaced_in_brief, created_at, updated_at, artifact_path) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, result.RequestID, result.ProjectID, result.Question, result.Rationale, result.Status, boolInt(result.SurfacedBrief), result.CreatedAt, result.UpdatedAt, result.ArtifactPath)
	return result, err
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (d *daemon) createAphroditeFixture(projectID string, request aphroditeFixtureRequest) (aphroditeFameFixture, error) {
	project, err := d.loadAphroditeProject(projectID)
	if err != nil {
		return aphroditeFameFixture{}, err
	}
	request.Title = strings.TrimSpace(request.Title)
	if request.Title == "" {
		return aphroditeFameFixture{}, errors.New("Fame fixture title is required")
	}
	sourceKind, sourceID, sourcePath, transcript := "corpus", "", "", strings.TrimSpace(request.Transcript)
	if strings.TrimSpace(request.RumorSourceID) != "" {
		source, loadErr := d.loadRumorSource(strings.TrimSpace(request.RumorSourceID))
		if loadErr != nil {
			return aphroditeFameFixture{}, fmt.Errorf("load Rumor source: %w", loadErr)
		}
		if source.TranscriptPath == "" {
			return aphroditeFameFixture{}, errors.New("Rumor source has no preserved transcript")
		}
		data, readErr := os.ReadFile(source.TranscriptPath)
		if readErr != nil {
			return aphroditeFameFixture{}, fmt.Errorf("read Rumor transcript: %w", readErr)
		}
		sourceKind, sourceID, sourcePath, transcript = "rumor_transcript", source.SourceID, source.TranscriptPath, string(data)
	}
	if transcript == "" {
		return aphroditeFameFixture{}, errors.New("Fame fixture transcript or Rumor source ID is required")
	}
	id, err := newLocalID("aph-fame-")
	if err != nil {
		return aphroditeFameFixture{}, err
	}
	path, err := aphroditeWorkspaceFile(project, filepath.Join("inputs", "fame", id+"-"+aphroditeSafeName(request.Title)+".md"))
	if err != nil {
		return aphroditeFameFixture{}, err
	}
	data := []byte(transcript)
	digest, err := writeAphroditeWorkspaceFile(path, data)
	if err != nil {
		return aphroditeFameFixture{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	fixture := aphroditeFameFixture{FixtureID: id, ProjectID: projectID, Title: request.Title, SourceKind: sourceKind, SourceID: sourceID, SourcePath: sourcePath, TranscriptPath: path, CitationStart: strings.TrimSpace(request.CitationStart), CitationEnd: strings.TrimSpace(request.CitationEnd), SHA256: digest, CreatedAt: now}
	_, err = d.db.Exec(`INSERT INTO aphrodite_fame_fixtures (fixture_id, project_id, title, source_kind, source_id, source_path, transcript_path, citation_start, citation_end, sha256, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, fixture.FixtureID, fixture.ProjectID, fixture.Title, fixture.SourceKind, nullableString(fixture.SourceID), nullableString(fixture.SourcePath), fixture.TranscriptPath, nullableString(fixture.CitationStart), nullableString(fixture.CitationEnd), fixture.SHA256, fixture.CreatedAt)
	return fixture, err
}

func (d *daemon) aphroditeSnapshot(projectID string) (aphroditeSnapshot, error) {
	projects, err := d.listAphroditeProjects()
	if err != nil {
		return aphroditeSnapshot{}, err
	}
	snapshot := aphroditeSnapshot{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), WorkspaceRoot: aphroditeWorkspaceRoot(d.databasePath),
		Projects: projects, Artifacts: []aphroditeArtifact{}, ResearchRequests: []aphroditeResearchRequest{}, FameFixtures: []aphroditeFameFixture{},
		ApprovedInspectionRoutes: []string{"/v1/status", "/v1/system/inventory", "/v1/rumor", "/v1/rumor/sources/{id}", "/v1/bifrost/capabilities", "/v1/weave/retrieve"},
		Boundaries:               []string{"production source and running binary are read-only from Aphrodite surfaces", "Rumor transcripts are copied, never edited", "Bifrost inventory is read-only; download, install, enable, and authorization remain outside Aphrodite", "candidate artifacts do not become production without an explicit future integration path"},
	}
	if projectID == "" && len(projects) > 0 {
		projectID = projects[0].ProjectID
	}
	for i := range projects {
		if projects[i].ProjectID == projectID {
			project := projects[i]
			snapshot.ActiveProject = &project
			break
		}
	}
	if projectID != "" {
		rows, queryErr := d.db.Query(`SELECT artifact_id, project_id, kind, title, path, sha256, status, created_at FROM aphrodite_artifacts WHERE project_id = ? ORDER BY created_at DESC LIMIT 100`, projectID)
		if queryErr != nil {
			return aphroditeSnapshot{}, queryErr
		}
		for rows.Next() {
			var item aphroditeArtifact
			if scanErr := rows.Scan(&item.ArtifactID, &item.ProjectID, &item.Kind, &item.Title, &item.Path, &item.SHA256, &item.Status, &item.CreatedAt); scanErr != nil {
				rows.Close()
				return aphroditeSnapshot{}, scanErr
			}
			snapshot.Artifacts = append(snapshot.Artifacts, item)
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			rows.Close()
			return aphroditeSnapshot{}, rowsErr
		}
		rows.Close()
		research, researchErr := d.listAphroditeResearch(projectID)
		if researchErr != nil {
			return aphroditeSnapshot{}, researchErr
		}
		snapshot.ResearchRequests = research
		fixtures, fixtureErr := d.listAphroditeFixtures(projectID)
		if fixtureErr != nil {
			return aphroditeSnapshot{}, fixtureErr
		}
		snapshot.FameFixtures = fixtures
	}
	snapshot.CapabilityInventory, err = d.listBifrostCapabilities()
	if err != nil {
		return aphroditeSnapshot{}, err
	}
	return snapshot, nil
}

func (d *daemon) listAphroditeResearch(projectID string) ([]aphroditeResearchRequest, error) {
	rows, err := d.db.Query(`SELECT request_id, project_id, question, COALESCE(rationale, ''), status, surfaced_in_brief, created_at, updated_at, COALESCE(artifact_path, '') FROM aphrodite_research_requests WHERE project_id = ? ORDER BY created_at DESC LIMIT 100`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []aphroditeResearchRequest{}
	for rows.Next() {
		var item aphroditeResearchRequest
		var surfaced int
		if err := rows.Scan(&item.RequestID, &item.ProjectID, &item.Question, &item.Rationale, &item.Status, &surfaced, &item.CreatedAt, &item.UpdatedAt, &item.ArtifactPath); err != nil {
			return nil, err
		}
		item.SurfacedBrief = surfaced == 1
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *daemon) listAphroditeFixtures(projectID string) ([]aphroditeFameFixture, error) {
	rows, err := d.db.Query(`SELECT fixture_id, project_id, title, source_kind, COALESCE(source_id, ''), COALESCE(source_path, ''), transcript_path, COALESCE(citation_start, ''), COALESCE(citation_end, ''), sha256, created_at FROM aphrodite_fame_fixtures WHERE project_id = ? ORDER BY created_at DESC LIMIT 100`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []aphroditeFameFixture{}
	for rows.Next() {
		var item aphroditeFameFixture
		if err := rows.Scan(&item.FixtureID, &item.ProjectID, &item.Title, &item.SourceKind, &item.SourceID, &item.SourcePath, &item.TranscriptPath, &item.CitationStart, &item.CitationEnd, &item.SHA256, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *daemon) systemAphroditeSummary() (map[string]any, error) {
	projects, err := d.listAphroditeProjects()
	if err != nil {
		return nil, err
	}
	var artifacts, research, fixtures int
	for _, query := range []struct {
		table string
		out   *int
	}{
		{"aphrodite_artifacts", &artifacts},
		{"aphrodite_research_requests", &research},
		{"aphrodite_fame_fixtures", &fixtures},
	} {
		if err := d.db.QueryRow("SELECT COUNT(*) FROM " + query.table).Scan(query.out); err != nil {
			return nil, err
		}
	}
	active := 0
	for _, project := range projects {
		if project.State == aphroditeProjectActive {
			active++
		}
	}
	return map[string]any{"workspace_root": aphroditeWorkspaceRoot(d.databasePath), "projects": len(projects), "active_projects": active, "artifacts": artifacts, "research_requests": research, "fame_fixtures": fixtures, "production_promotion": "unavailable"}, nil
}

func (d *daemon) handleAphroditeSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := d.aphroditeSnapshot(strings.TrimSpace(r.URL.Query().Get("project")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (d *daemon) handleAphroditeProjectCreate(w http.ResponseWriter, r *http.Request) {
	var request aphroditeProjectRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	request.Name, request.Purpose = strings.TrimSpace(request.Name), strings.TrimSpace(request.Purpose)
	if request.Name == "" || request.Purpose == "" {
		writeError(w, http.StatusBadRequest, errors.New("project name and purpose are required"))
		return
	}
	phase := request.Phase
	if phase == "" {
		phase = aphroditePhaseObservation
	}
	if !aphroditePhases[phase] {
		writeError(w, http.StatusBadRequest, errors.New("invalid Aphrodite project phase"))
		return
	}
	id := strings.TrimSpace(request.ProjectID)
	if id == "" {
		id = "project-" + aphroditeSafeName(request.Name)
	}
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`).MatchString(id) {
		writeError(w, http.StatusBadRequest, errors.New("project_id must contain only lowercase letters, numbers, and hyphens"))
		return
	}
	path := aphroditeProjectWorkspace(d.databasePath, id)
	if err := os.MkdirAll(filepath.Join(path, "inputs"), 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := d.db.Exec(`INSERT INTO aphrodite_projects (project_id, name, purpose, phase, state, workspace_path, created_at, updated_at, pending_questions, pending_approvals) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '[]', '[]')`, id, request.Name, request.Purpose, phase, aphroditeProjectActive, path, now, now)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	project, err := d.loadAphroditeProject(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (d *daemon) handleAphroditeProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	project, err := d.loadAphroditeProject(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("Aphrodite project not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (d *daemon) handleAphroditePhase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var request aphroditePhaseRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !aphroditePhases[request.Phase] {
		writeError(w, http.StatusBadRequest, errors.New("invalid Aphrodite project phase"))
		return
	}
	if request.State == "" {
		request.State = aphroditeProjectActive
	}
	if !aphroditeProjectStates[request.State] {
		writeError(w, http.StatusBadRequest, errors.New("invalid Aphrodite project state"))
		return
	}
	questions, _ := json.Marshal(request.PendingQuestions)
	approvals, _ := json.Marshal(request.PendingApprovals)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := d.db.Exec(`UPDATE aphrodite_projects SET phase = ?, state = ?, last_result = ?, pending_questions = ?, pending_approvals = ?, updated_at = ? WHERE project_id = ?`, request.Phase, request.State, strings.TrimSpace(request.LastResult), string(questions), string(approvals), now, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		writeError(w, http.StatusNotFound, errors.New("Aphrodite project not found"))
		return
	}
	project, err := d.loadAphroditeProject(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (d *daemon) handleAphroditeArtifacts(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if r.Method == http.MethodPost {
		var request aphroditeArtifactRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		artifact, err := d.createAphroditeArtifact(projectID, request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, artifact)
		return
	}
	if _, err := d.loadAphroditeProject(projectID); err != nil {
		writeError(w, http.StatusNotFound, errors.New("Aphrodite project not found"))
		return
	}
	rows, err := d.db.Query(`SELECT artifact_id, project_id, kind, title, path, sha256, status, created_at FROM aphrodite_artifacts WHERE project_id = ? ORDER BY created_at DESC LIMIT 100`, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	items := []aphroditeArtifact{}
	for rows.Next() {
		var item aphroditeArtifact
		if err := rows.Scan(&item.ArtifactID, &item.ProjectID, &item.Kind, &item.Title, &item.Path, &item.SHA256, &item.Status, &item.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (d *daemon) handleAphroditeResearch(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if r.Method == http.MethodPost {
		var request aphroditeResearchRequestRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item, err := d.createAphroditeResearchRequest(projectID, request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusAccepted, item)
		return
	}
	items, err := d.listAphroditeResearch(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (d *daemon) handleAphroditeFixtures(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if r.Method == http.MethodPost {
		var request aphroditeFixtureRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item, err := d.createAphroditeFixture(projectID, request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
		return
	}
	items, err := d.listAphroditeFixtures(projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
