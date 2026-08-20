package pantheon

// Fame is the human-recognition and relational-continuity layer between Echo
// and later continuity consumers. Echo may say "SPEAKER_00"; Fame records
// what evidence, if any, supports saying who that speaker is.

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	fameStatusConfirmed      = "confirmed"
	fameStatusHighConfidence = "high-confidence"
	fameStatusProbable       = "probable"
	fameStatusUncertain      = "uncertain"
	fameStatusUnknown        = "unknown"
	fameStatusConflicted     = "conflicted"
	fameStatusProvisional    = "provisional"

	fameBoundaryProvisional  = "provisional"
	fameBoundaryConfirmed    = "confirmed"
	fameAttributionAnonymous = "anonymous"
	fameAttributionInferred  = "inferred"
	fameAttributionCorrected = "corrected"
)

type famePerson struct {
	PersonID      string   `json:"person_id"`
	PreferredName string   `json:"preferred_name"`
	Aliases       []string `json:"aliases,omitempty"`
	Status        string   `json:"status"`
	IsSelf        bool     `json:"is_self"`
	Notes         string   `json:"notes,omitempty"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type fameIdentityEvidence struct {
	EvidenceID   string  `json:"evidence_id"`
	PersonID     string  `json:"person_id"`
	Kind         string  `json:"kind"`
	SourceRef    string  `json:"source_ref,omitempty"`
	Detail       string  `json:"detail,omitempty"`
	Confidence   float64 `json:"confidence"`
	CreatedAt    string  `json:"created_at"`
	CorrectionID string  `json:"correction_id,omitempty"`
}

type fameConversation struct {
	ConversationID string  `json:"conversation_id"`
	RunID          string  `json:"run_id,omitempty"`
	SourcePath     string  `json:"source_path"`
	SourceSHA256   string  `json:"source_sha256,omitempty"`
	StartSeconds   float64 `json:"start_seconds"`
	EndSeconds     float64 `json:"end_seconds"`
	BoundaryState  string  `json:"boundary_state"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type fameSegmentAttribution struct {
	AttributionID    string  `json:"attribution_id"`
	ConversationID   string  `json:"conversation_id"`
	SegmentID        string  `json:"segment_id"`
	AnonymousSpeaker string  `json:"anonymous_speaker"`
	PersonID         string  `json:"person_id,omitempty"`
	State            string  `json:"state"`
	Confidence       float64 `json:"confidence"`
	EvidenceID       string  `json:"evidence_id,omitempty"`
	UpdatedAt        string  `json:"updated_at"`
}

type fameObservation struct {
	ObservationID   string  `json:"observation_id"`
	ConversationID  string  `json:"conversation_id"`
	Kind            string  `json:"kind"`
	Content         string  `json:"content"`
	ActorPersonID   string  `json:"actor_person_id,omitempty"`
	SubjectPersonID string  `json:"subject_person_id,omitempty"`
	SourceRef       string  `json:"source_ref"`
	StartSeconds    float64 `json:"start_seconds,omitempty"`
	EndSeconds      float64 `json:"end_seconds,omitempty"`
	Confidence      float64 `json:"confidence"`
	Status          string  `json:"status"`
	CreatedAt       string  `json:"created_at"`
}

type fameCorrection struct {
	CorrectionID string `json:"correction_id"`
	TargetType   string `json:"target_type"`
	TargetID     string `json:"target_id"`
	Kind         string `json:"kind"`
	Statement    string `json:"statement"`
	PersonID     string `json:"person_id,omitempty"`
	RunID        string `json:"run_id,omitempty"`
	Speaker      string `json:"speaker,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type fameSnapshot struct {
	GeneratedAt    string                   `json:"generated_at"`
	People         []famePerson             `json:"people"`
	Conversations  []fameConversation       `json:"conversations"`
	Attributions   []fameSegmentAttribution `json:"attributions"`
	Observations   []fameObservation        `json:"observations"`
	Corrections    []fameCorrection         `json:"corrections"`
	IdentityStates []string                 `json:"identity_states"`
	Boundaries     []string                 `json:"boundaries"`
}

type famePersonRequest struct {
	PreferredName string   `json:"preferred_name"`
	Aliases       []string `json:"aliases,omitempty"`
	Status        string   `json:"status,omitempty"`
	IsSelf        bool     `json:"is_self,omitempty"`
	Notes         string   `json:"notes,omitempty"`
}

type fameEvidenceRequest struct {
	Kind       string  `json:"kind"`
	SourceRef  string  `json:"source_ref,omitempty"`
	Detail     string  `json:"detail,omitempty"`
	Confidence float64 `json:"confidence"`
}

type fameConversationRequest struct {
	RunID         string  `json:"run_id"`
	StartSeconds  float64 `json:"start_seconds,omitempty"`
	EndSeconds    float64 `json:"end_seconds,omitempty"`
	BoundaryState string  `json:"boundary_state,omitempty"`
}

type fameObservationRequest struct {
	ConversationID  string  `json:"conversation_id"`
	Kind            string  `json:"kind"`
	Content         string  `json:"content"`
	ActorPersonID   string  `json:"actor_person_id,omitempty"`
	SubjectPersonID string  `json:"subject_person_id,omitempty"`
	SourceRef       string  `json:"source_ref"`
	StartSeconds    float64 `json:"start_seconds,omitempty"`
	EndSeconds      float64 `json:"end_seconds,omitempty"`
	Confidence      float64 `json:"confidence"`
}

type fameCorrectionRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id,omitempty"`
	Kind       string `json:"kind"`
	Statement  string `json:"statement"`
	PersonID   string `json:"person_id,omitempty"`
	RunID      string `json:"run_id,omitempty"`
	Speaker    string `json:"speaker,omitempty"`
}

func initializeFameSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS fame_persons (
			person_id TEXT PRIMARY KEY, preferred_name TEXT NOT NULL, aliases_json TEXT NOT NULL DEFAULT '[]',
			status TEXT NOT NULL, is_self INTEGER NOT NULL DEFAULT 0, notes TEXT,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS fame_identity_evidence (
			evidence_id TEXT PRIMARY KEY, person_id TEXT NOT NULL, kind TEXT NOT NULL,
			source_ref TEXT, detail TEXT, confidence REAL NOT NULL DEFAULT 0,
			correction_id TEXT, created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS fame_evidence_person_idx ON fame_identity_evidence(person_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS fame_conversations (
			conversation_id TEXT PRIMARY KEY, run_id TEXT, source_path TEXT NOT NULL, source_sha256 TEXT,
			start_seconds REAL NOT NULL DEFAULT 0, end_seconds REAL NOT NULL DEFAULT 0,
			boundary_state TEXT NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS fame_conversations_created_idx ON fame_conversations(created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS fame_segment_attributions (
			attribution_id TEXT PRIMARY KEY, conversation_id TEXT NOT NULL, segment_id TEXT NOT NULL UNIQUE,
			anonymous_speaker TEXT NOT NULL, person_id TEXT, state TEXT NOT NULL,
			confidence REAL NOT NULL DEFAULT 0, evidence_id TEXT, updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS fame_attributions_conversation_idx ON fame_segment_attributions(conversation_id, anonymous_speaker)`,
		`CREATE TABLE IF NOT EXISTS fame_observations (
			observation_id TEXT PRIMARY KEY, conversation_id TEXT NOT NULL, kind TEXT NOT NULL,
			content TEXT NOT NULL, actor_person_id TEXT, subject_person_id TEXT, source_ref TEXT NOT NULL,
			start_seconds REAL NOT NULL DEFAULT 0, end_seconds REAL NOT NULL DEFAULT 0,
			confidence REAL NOT NULL DEFAULT 0, status TEXT NOT NULL, created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS fame_observations_conversation_idx ON fame_observations(conversation_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS fame_corrections (
			correction_id TEXT PRIMARY KEY, target_type TEXT NOT NULL, target_id TEXT,
			kind TEXT NOT NULL, statement TEXT NOT NULL, person_id TEXT, run_id TEXT, speaker TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS fame_corrections_created_idx ON fame_corrections(created_at DESC)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func (d *daemon) ensureFameDefaults() error {
	if err := initializeFameSchema(d.db); err != nil {
		return err
	}
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM fame_persons WHERE person_id = 'person-self'`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		if _, err := d.db.Exec(`INSERT INTO fame_persons (person_id, preferred_name, aliases_json, status, is_self, notes, created_at, updated_at) VALUES ('person-self', 'Grant', '[]', ?, 1, 'Local owner identity; identity evidence remains inspectable.', ?, ?)`, fameStatusConfirmed, now, now); err != nil {
			return err
		}
	}
	return nil
}

func validFameStatus(status string) bool {
	switch status {
	case fameStatusConfirmed, fameStatusHighConfidence, fameStatusProbable, fameStatusUncertain, fameStatusUnknown, fameStatusConflicted, fameStatusProvisional:
		return true
	default:
		return false
	}
}

func scanFamePerson(scanner interface{ Scan(...any) error }) (famePerson, error) {
	var person famePerson
	var aliases string
	var self int
	if err := scanner.Scan(&person.PersonID, &person.PreferredName, &aliases, &person.Status, &self, &person.Notes, &person.CreatedAt, &person.UpdatedAt); err != nil {
		return famePerson{}, err
	}
	_ = json.Unmarshal([]byte(aliases), &person.Aliases)
	person.IsSelf = self != 0
	return person, nil
}

func (d *daemon) createFamePerson(request famePersonRequest) (famePerson, error) {
	request.PreferredName = strings.TrimSpace(request.PreferredName)
	if request.PreferredName == "" || len(request.PreferredName) > 120 {
		return famePerson{}, errors.New("preferred_name must be 1-120 characters")
	}
	if request.Status == "" {
		request.Status = fameStatusProvisional
	}
	if !validFameStatus(request.Status) {
		return famePerson{}, errors.New("invalid Fame person status")
	}
	id, err := newLocalID("person-")
	if err != nil {
		return famePerson{}, err
	}
	aliases, _ := json.Marshal(request.Aliases)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = d.db.Exec(`INSERT INTO fame_persons (person_id, preferred_name, aliases_json, status, is_self, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, id, request.PreferredName, string(aliases), request.Status, boolInt(request.IsSelf), strings.TrimSpace(request.Notes), now, now)
	if err != nil {
		return famePerson{}, err
	}
	return scanFamePerson(d.db.QueryRow(`SELECT person_id, preferred_name, aliases_json, status, is_self, COALESCE(notes,''), created_at, updated_at FROM fame_persons WHERE person_id = ?`, id))
}

func (d *daemon) addFameEvidence(personID string, request fameEvidenceRequest, correctionID string) (fameIdentityEvidence, error) {
	if strings.TrimSpace(request.Kind) == "" {
		return fameIdentityEvidence{}, errors.New("evidence kind is required")
	}
	var exists int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM fame_persons WHERE person_id = ?`, personID).Scan(&exists); err != nil {
		return fameIdentityEvidence{}, err
	}
	if exists == 0 {
		return fameIdentityEvidence{}, errors.New("unknown Fame person")
	}
	if request.Confidence < 0 || request.Confidence > 1 {
		return fameIdentityEvidence{}, errors.New("confidence must be between 0 and 1")
	}
	id, err := newLocalID("evidence-")
	if err != nil {
		return fameIdentityEvidence{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = d.db.Exec(`INSERT INTO fame_identity_evidence (evidence_id, person_id, kind, source_ref, detail, confidence, correction_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, id, personID, strings.TrimSpace(request.Kind), strings.TrimSpace(request.SourceRef), strings.TrimSpace(request.Detail), request.Confidence, nullableString(correctionID), now)
	if err != nil {
		return fameIdentityEvidence{}, err
	}
	return fameIdentityEvidence{EvidenceID: id, PersonID: personID, Kind: strings.TrimSpace(request.Kind), SourceRef: strings.TrimSpace(request.SourceRef), Detail: strings.TrimSpace(request.Detail), Confidence: request.Confidence, CorrectionID: correctionID, CreatedAt: now}, nil
}

func (d *daemon) createFameConversation(request fameConversationRequest) (fameConversation, error) {
	var run echoRun
	if err := d.db.QueryRow(`SELECT run_id, source_path, COALESCE(source_sha256,''), speaker_count, status, diarization_path, segment_count, created_at, COALESCE(completed_at,''), COALESCE(last_error,'') FROM echo_runs WHERE run_id = ?`, strings.TrimSpace(request.RunID)).Scan(&run.RunID, &run.SourcePath, &run.SourceSHA256, &run.SpeakerCount, &run.Status, &run.DiarizationPath, &run.SegmentCount, &run.CreatedAt, &run.CompletedAt, &run.LastError); err != nil {
		return fameConversation{}, errors.New("unknown Echo run")
	}
	if request.EndSeconds > 0 && request.EndSeconds <= request.StartSeconds {
		return fameConversation{}, errors.New("conversation time bounds are invalid")
	}
	boundary := request.BoundaryState
	if boundary == "" {
		boundary = fameBoundaryProvisional
	}
	if boundary != fameBoundaryProvisional && boundary != fameBoundaryConfirmed {
		return fameConversation{}, errors.New("boundary_state must be provisional or confirmed")
	}
	conversationID := "conversation-" + run.RunID
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := d.db.Exec(`INSERT INTO fame_conversations (conversation_id, run_id, source_path, source_sha256, start_seconds, end_seconds, boundary_state, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(conversation_id) DO UPDATE SET boundary_state=excluded.boundary_state, start_seconds=excluded.start_seconds, end_seconds=excluded.end_seconds, updated_at=excluded.updated_at`, conversationID, run.RunID, run.SourcePath, run.SourceSHA256, request.StartSeconds, request.EndSeconds, boundary, "imported", now, now)
	if err != nil {
		return fameConversation{}, err
	}
	rows, err := d.db.Query(`SELECT segment_id, anonymous_speaker FROM echo_speaker_segments WHERE run_id = ? ORDER BY start_seconds`, run.RunID)
	if err != nil {
		return fameConversation{}, err
	}
	var segments []struct{ id, speaker string }
	for rows.Next() {
		var segmentID, speaker string
		if err := rows.Scan(&segmentID, &speaker); err != nil {
			rows.Close()
			return fameConversation{}, err
		}
		segments = append(segments, struct{ id, speaker string }{segmentID, speaker})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fameConversation{}, err
	}
	rows.Close()
	for _, segment := range segments {
		if _, err = d.db.Exec(`INSERT OR IGNORE INTO fame_segment_attributions (attribution_id, conversation_id, segment_id, anonymous_speaker, state, confidence, updated_at) VALUES (?, ?, ?, ?, ?, 0, ?)`, "attribution-"+segment.id, conversationID, segment.id, segment.speaker, fameAttributionAnonymous, now); err != nil {
			return fameConversation{}, err
		}
	}
	return fameConversation{ConversationID: conversationID, RunID: run.RunID, SourcePath: run.SourcePath, SourceSHA256: run.SourceSHA256, StartSeconds: request.StartSeconds, EndSeconds: request.EndSeconds, BoundaryState: boundary, Status: "imported", CreatedAt: now, UpdatedAt: now}, nil
}

func (d *daemon) createFameObservation(request fameObservationRequest) (fameObservation, error) {
	request.Kind, request.Content, request.SourceRef = strings.TrimSpace(request.Kind), strings.TrimSpace(request.Content), strings.TrimSpace(request.SourceRef)
	if request.Kind == "" || request.Content == "" || request.SourceRef == "" {
		return fameObservation{}, errors.New("observation kind, content, and source_ref are required")
	}
	if request.Confidence < 0 || request.Confidence > 1 {
		return fameObservation{}, errors.New("confidence must be between 0 and 1")
	}
	var exists int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM fame_conversations WHERE conversation_id = ?`, request.ConversationID).Scan(&exists); err != nil {
		return fameObservation{}, err
	}
	if exists == 0 {
		return fameObservation{}, errors.New("unknown Fame conversation")
	}
	id, err := newLocalID("observation-")
	if err != nil {
		return fameObservation{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = d.db.Exec(`INSERT INTO fame_observations (observation_id, conversation_id, kind, content, actor_person_id, subject_person_id, source_ref, start_seconds, end_seconds, confidence, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'accountable_observation', ?)`, id, request.ConversationID, request.Kind, request.Content, nullableString(request.ActorPersonID), nullableString(request.SubjectPersonID), request.SourceRef, request.StartSeconds, request.EndSeconds, request.Confidence, now)
	if err != nil {
		return fameObservation{}, err
	}
	return fameObservation{ObservationID: id, ConversationID: request.ConversationID, Kind: request.Kind, Content: request.Content, ActorPersonID: request.ActorPersonID, SubjectPersonID: request.SubjectPersonID, SourceRef: request.SourceRef, StartSeconds: request.StartSeconds, EndSeconds: request.EndSeconds, Confidence: request.Confidence, Status: "accountable_observation", CreatedAt: now}, nil
}

func (d *daemon) createFameCorrection(request fameCorrectionRequest) (fameCorrection, error) {
	request.TargetType, request.TargetID, request.Kind, request.Statement = strings.TrimSpace(request.TargetType), strings.TrimSpace(request.TargetID), strings.TrimSpace(request.Kind), strings.TrimSpace(request.Statement)
	if request.TargetType == "" || request.Kind == "" || request.Statement == "" {
		return fameCorrection{}, errors.New("target_type, kind, and statement are required")
	}
	id, err := newLocalID("correction-")
	if err != nil {
		return fameCorrection{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = d.db.Exec(`INSERT INTO fame_corrections (correction_id, target_type, target_id, kind, statement, person_id, run_id, speaker, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, request.TargetType, nullableString(request.TargetID), request.Kind, request.Statement, nullableString(request.PersonID), nullableString(request.RunID), nullableString(request.Speaker), now)
	if err != nil {
		return fameCorrection{}, err
	}
	if request.PersonID != "" && request.TargetType == "segment" && request.TargetID != "" {
		if _, err := d.db.Exec(`UPDATE fame_segment_attributions SET person_id = ?, state = ?, confidence = 1, evidence_id = ?, updated_at = ? WHERE segment_id = ?`, request.PersonID, fameAttributionCorrected, id, now, request.TargetID); err != nil {
			return fameCorrection{}, err
		}
	}
	// A correction that explicitly names a run, anonymous speaker, and person
	// is allowed to change attribution. A prose correction alone is retained
	// as evidence and does not silently relabel a whole recording.
	if request.RunID != "" && request.Speaker != "" && request.PersonID != "" {
		if _, err := d.db.Exec(`UPDATE fame_segment_attributions SET person_id = ?, state = ?, confidence = 1, evidence_id = ?, updated_at = ? WHERE segment_id IN (SELECT segment_id FROM echo_speaker_segments WHERE run_id = ? AND anonymous_speaker = ?)`, request.PersonID, fameAttributionCorrected, id, now, request.RunID, request.Speaker); err != nil {
			return fameCorrection{}, err
		}
	}
	return fameCorrection{CorrectionID: id, TargetType: request.TargetType, TargetID: request.TargetID, Kind: request.Kind, Statement: request.Statement, PersonID: request.PersonID, RunID: request.RunID, Speaker: request.Speaker, CreatedAt: now}, nil
}

func (d *daemon) clearFameSegment(segmentID string) (fameCorrection, error) {
	var exists int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM fame_segment_attributions WHERE segment_id = ?`, segmentID).Scan(&exists); err != nil {
		return fameCorrection{}, err
	}
	if exists == 0 {
		return fameCorrection{}, errors.New("unknown Fame segment attribution")
	}
	correction, err := d.createFameCorrection(fameCorrectionRequest{TargetType: "segment", TargetID: segmentID, Kind: "identity_cleared", Statement: "Human cleared this segment's identity assignment for review."})
	if err != nil {
		return fameCorrection{}, err
	}
	if _, err := d.db.Exec(`UPDATE fame_segment_attributions SET person_id = NULL, state = ?, confidence = 0, evidence_id = NULL, updated_at = ? WHERE segment_id = ?`, fameAttributionAnonymous, correction.CreatedAt, segmentID); err != nil {
		return fameCorrection{}, err
	}
	return correction, nil
}

func (d *daemon) fameSnapshot() (fameSnapshot, error) {
	if err := d.ensureFameDefaults(); err != nil {
		return fameSnapshot{}, err
	}
	snapshot := fameSnapshot{GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), People: []famePerson{}, Conversations: []fameConversation{}, Attributions: []fameSegmentAttribution{}, Observations: []fameObservation{}, Corrections: []fameCorrection{}, IdentityStates: []string{fameStatusConfirmed, fameStatusHighConfidence, fameStatusProbable, fameStatusUncertain, fameStatusUnknown, fameStatusConflicted, fameStatusProvisional}, Boundaries: []string{
		"Fame preserves anonymous Echo labels until identity evidence or a human correction supports a person reference.",
		"Voice-derived identity material remains local; voice IDs and person IDs are separate records.",
		"Unknown and provisional people remain durable and may be resolved later without deleting history.",
		"Observations retain a source reference and do not grant executive authority.",
	}}
	rows, err := d.db.Query(`SELECT person_id, preferred_name, aliases_json, status, is_self, COALESCE(notes,''), created_at, updated_at FROM fame_persons ORDER BY is_self DESC, preferred_name`)
	if err != nil {
		return fameSnapshot{}, err
	}
	for rows.Next() {
		person, scanErr := scanFamePerson(rows)
		if scanErr != nil {
			rows.Close()
			return fameSnapshot{}, scanErr
		}
		snapshot.People = append(snapshot.People, person)
	}
	rows.Close()
	rows, err = d.db.Query(`SELECT conversation_id, COALESCE(run_id,''), source_path, COALESCE(source_sha256,''), start_seconds, end_seconds, boundary_state, status, created_at, updated_at FROM fame_conversations ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return fameSnapshot{}, err
	}
	for rows.Next() {
		var item fameConversation
		if err := rows.Scan(&item.ConversationID, &item.RunID, &item.SourcePath, &item.SourceSHA256, &item.StartSeconds, &item.EndSeconds, &item.BoundaryState, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			rows.Close()
			return fameSnapshot{}, err
		}
		snapshot.Conversations = append(snapshot.Conversations, item)
	}
	rows.Close()
	rows, err = d.db.Query(`SELECT attribution_id, conversation_id, segment_id, anonymous_speaker, COALESCE(person_id,''), state, confidence, COALESCE(evidence_id,''), updated_at FROM fame_segment_attributions ORDER BY updated_at DESC LIMIT 200`)
	if err != nil {
		return fameSnapshot{}, err
	}
	for rows.Next() {
		var item fameSegmentAttribution
		if err := rows.Scan(&item.AttributionID, &item.ConversationID, &item.SegmentID, &item.AnonymousSpeaker, &item.PersonID, &item.State, &item.Confidence, &item.EvidenceID, &item.UpdatedAt); err != nil {
			rows.Close()
			return fameSnapshot{}, err
		}
		snapshot.Attributions = append(snapshot.Attributions, item)
	}
	rows.Close()
	rows, err = d.db.Query(`SELECT observation_id, conversation_id, kind, content, COALESCE(actor_person_id,''), COALESCE(subject_person_id,''), source_ref, start_seconds, end_seconds, confidence, status, created_at FROM fame_observations ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return fameSnapshot{}, err
	}
	for rows.Next() {
		var item fameObservation
		if err := rows.Scan(&item.ObservationID, &item.ConversationID, &item.Kind, &item.Content, &item.ActorPersonID, &item.SubjectPersonID, &item.SourceRef, &item.StartSeconds, &item.EndSeconds, &item.Confidence, &item.Status, &item.CreatedAt); err != nil {
			rows.Close()
			return fameSnapshot{}, err
		}
		snapshot.Observations = append(snapshot.Observations, item)
	}
	rows.Close()
	rows, err = d.db.Query(`SELECT correction_id, target_type, COALESCE(target_id,''), kind, statement, COALESCE(person_id,''), COALESCE(run_id,''), COALESCE(speaker,''), created_at FROM fame_corrections ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return fameSnapshot{}, err
	}
	for rows.Next() {
		var item fameCorrection
		if err := rows.Scan(&item.CorrectionID, &item.TargetType, &item.TargetID, &item.Kind, &item.Statement, &item.PersonID, &item.RunID, &item.Speaker, &item.CreatedAt); err != nil {
			rows.Close()
			return fameSnapshot{}, err
		}
		snapshot.Corrections = append(snapshot.Corrections, item)
	}
	rows.Close()
	return snapshot, nil
}

func (d *daemon) handleFameSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := d.fameSnapshot()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (d *daemon) handleFamePeople(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var request famePersonRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		person, err := d.createFamePerson(request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, person)
		return
	}
	snapshot, err := d.fameSnapshot()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot.People)
}

func (d *daemon) handleFamePersonEvidence(w http.ResponseWriter, r *http.Request) {
	var request fameEvidenceRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	evidence, err := d.addFameEvidence(r.PathValue("id"), request, "")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, evidence)
}

func (d *daemon) handleFameConversation(w http.ResponseWriter, r *http.Request) {
	var request fameConversationRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	conversation, err := d.createFameConversation(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, conversation)
}

func (d *daemon) handleFameObservation(w http.ResponseWriter, r *http.Request) {
	var request fameObservationRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	observation, err := d.createFameObservation(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, observation)
}

func (d *daemon) handleFameCorrection(w http.ResponseWriter, r *http.Request) {
	var request fameCorrectionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	correction, err := d.createFameCorrection(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, correction)
}

func (d *daemon) handleFameSegmentClear(w http.ResponseWriter, r *http.Request) {
	correction, err := d.clearFameSegment(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cleared": true, "correction": correction})
}

func (d *daemon) handleFameCommand(w http.ResponseWriter, r *http.Request) {
	var request echoCommandRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	fields := strings.Fields(strings.TrimSpace(request.Input))
	if len(fields) == 0 || fields[0] != "/fame" {
		writeError(w, http.StatusBadRequest, errors.New("command must start with /fame"))
		return
	}
	if len(fields) == 1 || fields[1] == "status" || fields[1] == "people" {
		snapshot, err := d.fameSnapshot()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, snapshot)
		return
	}
	switch fields[1] {
	case "person":
		if len(fields) < 4 || fields[2] != "add" {
			writeError(w, http.StatusBadRequest, errors.New("usage: /fame person add <name>"))
			return
		}
		person, err := d.createFamePerson(famePersonRequest{PreferredName: strings.Join(fields[3:], " ")})
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, person)
	case "conversation":
		if len(fields) != 3 {
			writeError(w, http.StatusBadRequest, errors.New("usage: /fame conversation <echo-run-id>"))
			return
		}
		conversation, err := d.createFameConversation(fameConversationRequest{RunID: fields[2]})
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, conversation)
	case "observation":
		if len(fields) < 6 {
			writeError(w, http.StatusBadRequest, errors.New("usage: /fame observation <conversation-id> <kind> <source-ref> <content>"))
			return
		}
		observation, err := d.createFameObservation(fameObservationRequest{ConversationID: fields[2], Kind: fields[3], SourceRef: fields[4], Content: strings.Join(fields[5:], " ")})
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, observation)
	case "correct":
		if len(fields) < 5 {
			writeError(w, http.StatusBadRequest, errors.New("usage: /fame correct <target-type> <kind> <statement>"))
			return
		}
		correction, err := d.createFameCorrection(fameCorrectionRequest{TargetType: fields[2], Kind: fields[3], Statement: strings.Join(fields[4:], " ")})
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, correction)
	case "evidence":
		if len(fields) < 6 {
			writeError(w, http.StatusBadRequest, errors.New("usage: /fame evidence <person-id> <kind> <confidence> <detail>"))
			return
		}
		confidence, err := strconv.ParseFloat(fields[4], 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, errors.New("confidence must be a number between 0 and 1"))
			return
		}
		evidence, err := d.addFameEvidence(fields[2], fameEvidenceRequest{Kind: fields[3], Confidence: confidence, Detail: strings.Join(fields[5:], " ")}, "")
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, evidence)
	default:
		writeError(w, http.StatusBadRequest, errors.New("supported commands: /fame status | /fame people | /fame person add <name> | /fame conversation <echo-run-id> | /fame observation <conversation-id> <kind> <source-ref> <content> | /fame correct <target-type> <kind> <statement>"))
	}
}
