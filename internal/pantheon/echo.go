package pantheon

import (
	"bufio"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Echo owns speaker separation and the human-confirmed voice directory. It
// never silently turns an anonymous diarization label into a person.
const (
	echoVoiceAwaitingSample = "awaiting_sample"
	echoVoiceEnrolled       = "enrolled"
	echoVoiceDisabled       = "disabled"
	echoMappingAnonymous    = "anonymous"
	echoMappingConfirmed    = "confirmed"
)

type echoVoiceProfile struct {
	VoiceID     string   `json:"voice_id"`
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases,omitempty"`
	State       string   `json:"state"`
	SampleCount int      `json:"sample_count"`
	LocalOnly   bool     `json:"local_only"`
	Notes       string   `json:"notes,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type echoVoiceSample struct {
	SampleID     string  `json:"sample_id"`
	VoiceID      string  `json:"voice_id"`
	SourcePath   string  `json:"source_path"`
	StartSeconds float64 `json:"start_seconds,omitempty"`
	EndSeconds   float64 `json:"end_seconds,omitempty"`
	SHA256       string  `json:"sha256"`
	CreatedAt    string  `json:"created_at"`
}

type echoRun struct {
	RunID           string `json:"run_id"`
	SourcePath      string `json:"source_path"`
	SourceSHA256    string `json:"source_sha256,omitempty"`
	SpeakerCount    int    `json:"speaker_count"`
	Status          string `json:"status"`
	DiarizationPath string `json:"diarization_path"`
	TranscriptPath  string `json:"transcript_path,omitempty"`
	SegmentCount    int    `json:"segment_count"`
	CreatedAt       string `json:"created_at"`
	CompletedAt     string `json:"completed_at,omitempty"`
	LastError       string `json:"last_error,omitempty"`
}

type echoSpeakerSegment struct {
	SegmentID         string  `json:"segment_id"`
	RunID             string  `json:"run_id"`
	StartSeconds      float64 `json:"start_seconds"`
	EndSeconds        float64 `json:"end_seconds"`
	AnonymousSpeaker  string  `json:"anonymous_speaker"`
	VoiceID           string  `json:"voice_id,omitempty"`
	MappingState      string  `json:"mapping_state"`
	TranscriptSnippet string  `json:"transcript_snippet,omitempty"`
}

type echoSnapshot struct {
	GeneratedAt       string               `json:"generated_at"`
	Enabled           bool                 `json:"enabled"`
	LocalOnly         bool                 `json:"local_only"`
	Model             string               `json:"model"`
	Profiles          []echoVoiceProfile   `json:"profiles"`
	SelectedRunID     string               `json:"selected_run_id,omitempty"`
	LatestRun         *echoRun             `json:"latest_run,omitempty"`
	RecentRuns        []echoRun            `json:"recent_runs"`
	AnonymousSpeakers []string             `json:"anonymous_speakers"`
	Segments          []echoSpeakerSegment `json:"segments,omitempty"`
	Boundaries        []string             `json:"boundaries"`
}

type echoVoiceRequest struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases,omitempty"`
	Notes   string   `json:"notes,omitempty"`
}

type echoEnrollRequest struct {
	SourcePath   string  `json:"source_path"`
	StartSeconds float64 `json:"start_seconds,omitempty"`
	EndSeconds   float64 `json:"end_seconds,omitempty"`
}

type echoImportRequest struct {
	RunID           string `json:"run_id,omitempty"`
	SourcePath      string `json:"source_path"`
	DiarizationPath string `json:"diarization_path"`
	SpeakerCount    int    `json:"speaker_count,omitempty"`
}

type echoConfirmRequest struct {
	RunID   string `json:"run_id"`
	Speaker string `json:"speaker"`
}

type echoSegmentConfirmRequest struct {
	VoiceID string `json:"voice_id"`
}

type echoVoiceRenameRequest struct {
	Name string `json:"name"`
}

type echoCommandRequest struct {
	Input string `json:"input"`
}

func initializeEchoSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS echo_voice_profiles (
			voice_id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, aliases_json TEXT NOT NULL DEFAULT '[]',
			state TEXT NOT NULL, sample_count INTEGER NOT NULL DEFAULT 0, local_only INTEGER NOT NULL DEFAULT 1,
			notes TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS echo_voice_samples (
			sample_id TEXT PRIMARY KEY, voice_id TEXT NOT NULL, source_path TEXT NOT NULL,
			start_seconds REAL NOT NULL DEFAULT 0, end_seconds REAL NOT NULL DEFAULT 0,
			sha256 TEXT NOT NULL, created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS echo_voice_samples_voice_idx ON echo_voice_samples(voice_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS echo_runs (
			run_id TEXT PRIMARY KEY, source_path TEXT NOT NULL, source_sha256 TEXT, speaker_count INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL, diarization_path TEXT NOT NULL, segment_count INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL, completed_at TEXT, last_error TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS echo_runs_created_idx ON echo_runs(created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS echo_speaker_segments (
			segment_id TEXT PRIMARY KEY, run_id TEXT NOT NULL, start_seconds REAL NOT NULL, end_seconds REAL NOT NULL,
			anonymous_speaker TEXT NOT NULL, voice_id TEXT, mapping_state TEXT NOT NULL DEFAULT 'anonymous'
		)`,
		`CREATE INDEX IF NOT EXISTS echo_segments_run_idx ON echo_speaker_segments(run_id, start_seconds)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func (d *daemon) ensureEchoDefaults() error {
	if err := initializeEchoSchema(d.db); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := d.db.Exec(`INSERT OR IGNORE INTO echo_voice_profiles (voice_id, name, aliases_json, state, sample_count, local_only, created_at, updated_at) VALUES ('me', 'me', '[]', ?, 0, 1, ?, ?)`, echoVoiceAwaitingSample, now, now)
	return err
}

func readEchoProfiles(db *sql.DB) ([]echoVoiceProfile, error) {
	rows, err := db.Query(`SELECT voice_id, name, aliases_json, state, sample_count, local_only, COALESCE(notes, ''), created_at, updated_at FROM echo_voice_profiles WHERE state != ? ORDER BY name`, echoVoiceDisabled)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	profiles := []echoVoiceProfile{}
	for rows.Next() {
		var p echoVoiceProfile
		var aliases string
		var local int
		if err := rows.Scan(&p.VoiceID, &p.Name, &aliases, &p.State, &p.SampleCount, &local, &p.Notes, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(aliases), &p.Aliases)
		p.LocalOnly = local != 0
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (d *daemon) echoSnapshot() (echoSnapshot, error) {
	return d.echoSnapshotForRun("")
}

func (d *daemon) echoSnapshotForRun(selectedRunID string) (echoSnapshot, error) {
	if err := d.ensureEchoDefaults(); err != nil {
		return echoSnapshot{}, err
	}
	profiles, err := readEchoProfiles(d.db)
	if err != nil {
		return echoSnapshot{}, err
	}
	s := echoSnapshot{GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Enabled: true, LocalOnly: true, Model: "pyannote/speaker-diarization-community-1", Profiles: profiles, RecentRuns: []echoRun{}, AnonymousSpeakers: []string{}, Boundaries: []string{
		"Voice profiles and diarization metadata stay on this machine.",
		"Whisper transcribes; Echo separates speakers. Neither model assigns a person without confirmation.",
		"Raw recordings are referenced, not copied, uploaded, or deleted by Echo.",
	}}
	rows, err := d.db.Query(`SELECT run_id, source_path, COALESCE(source_sha256,''), speaker_count, status, diarization_path, segment_count, created_at, COALESCE(completed_at,''), COALESCE(last_error,'') FROM echo_runs ORDER BY created_at DESC LIMIT 10`)
	if err != nil {
		return echoSnapshot{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var r echoRun
		if err := rows.Scan(&r.RunID, &r.SourcePath, &r.SourceSHA256, &r.SpeakerCount, &r.Status, &r.DiarizationPath, &r.SegmentCount, &r.CreatedAt, &r.CompletedAt, &r.LastError); err != nil {
			return echoSnapshot{}, err
		}
		s.RecentRuns = append(s.RecentRuns, r)
	}
	if err := rows.Err(); err != nil {
		return echoSnapshot{}, err
	}
	if len(s.RecentRuns) > 0 {
		s.LatestRun = &s.RecentRuns[0]
		if strings.TrimSpace(selectedRunID) != "" {
			found := false
			for i := range s.RecentRuns {
				if s.RecentRuns[i].RunID == strings.TrimSpace(selectedRunID) {
					s.LatestRun = &s.RecentRuns[i]
					found = true
					break
				}
			}
			if !found {
				return echoSnapshot{}, fmt.Errorf("unknown Echo run %q", selectedRunID)
			}
		}
		s.SelectedRunID = s.LatestRun.RunID
	}
	if s.LatestRun != nil {
		s.LatestRun.TranscriptPath = d.echoTranscriptPath(s.LatestRun.SourcePath)
		transcript, _ := readEchoTranscript(s.LatestRun.TranscriptPath)
		rows, err := d.db.Query(`SELECT DISTINCT anonymous_speaker FROM echo_speaker_segments WHERE run_id = ? ORDER BY anonymous_speaker`, s.LatestRun.RunID)
		if err != nil {
			return echoSnapshot{}, err
		}
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return echoSnapshot{}, err
			}
			s.AnonymousSpeakers = append(s.AnonymousSpeakers, name)
		}
		rows.Close()
		segRows, err := d.db.Query(`SELECT segment_id, run_id, start_seconds, end_seconds, anonymous_speaker, COALESCE(voice_id,''), mapping_state FROM echo_speaker_segments WHERE run_id = ? ORDER BY start_seconds LIMIT 50`, s.LatestRun.RunID)
		if err != nil {
			return echoSnapshot{}, err
		}
		defer segRows.Close()
		for segRows.Next() {
			var x echoSpeakerSegment
			if err := segRows.Scan(&x.SegmentID, &x.RunID, &x.StartSeconds, &x.EndSeconds, &x.AnonymousSpeaker, &x.VoiceID, &x.MappingState); err != nil {
				return echoSnapshot{}, err
			}
			x.TranscriptSnippet = echoTranscriptSnippetForSegment(transcript, x)
			s.Segments = append(s.Segments, x)
		}
	}
	return s, nil
}

type echoTranscriptSnippet struct {
	StartSeconds     float64
	EndSeconds       float64
	AnonymousSpeaker string
	Text             string
}

var echoTranscriptLinePattern = regexp.MustCompile(`^\[([^\]]+)[–-]([^\]]+)\]\s+\*\*([^:*]+):\*\*\s*(.*)$`)

func parseEchoTimestamp(value string) (float64, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 3 {
		return 0, errors.New("invalid timestamp")
	}
	hours, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}
	minutes, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, err
	}
	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, err
	}
	return hours*3600 + minutes*60 + seconds, nil
}

func (d *daemon) echoTranscriptPath(sourcePath string) string {
	if strings.TrimSpace(d.cfg.VaultPath) == "" {
		return ""
	}
	base := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	pattern := filepath.Join(d.cfg.VaultPath, "80 Pantheon Raw", "Audio", "*", "*", "*", base+".kernel24-safe-*.speaker-transcript.md")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}

func readEchoTranscript(path string) ([]echoTranscriptSnippet, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var snippets []echoTranscriptSnippet
	for scanner.Scan() {
		match := echoTranscriptLinePattern.FindStringSubmatch(strings.TrimSpace(scanner.Text()))
		if len(match) != 5 {
			continue
		}
		start, startErr := parseEchoTimestamp(match[1])
		end, endErr := parseEchoTimestamp(match[2])
		if startErr != nil || endErr != nil || end < start || strings.TrimSpace(match[4]) == "" {
			continue
		}
		snippets = append(snippets, echoTranscriptSnippet{StartSeconds: start, EndSeconds: end, AnonymousSpeaker: strings.TrimSpace(match[3]), Text: strings.TrimSpace(match[4])})
	}
	return snippets, scanner.Err()
}

func echoTranscriptSnippetForSegment(snippets []echoTranscriptSnippet, segment echoSpeakerSegment) string {
	bestOverlap := 0.0
	bestText := ""
	for _, snippet := range snippets {
		if snippet.AnonymousSpeaker != segment.AnonymousSpeaker {
			continue
		}
		overlap := minFloat(snippet.EndSeconds, segment.EndSeconds) - maxFloat(snippet.StartSeconds, segment.StartSeconds)
		if overlap > bestOverlap {
			bestOverlap, bestText = overlap, snippet.Text
		}
	}
	return bestText
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func echoFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func validateEchoLocalFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("a local file path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("path is not a regular file")
	}
	return abs, nil
}

func (d *daemon) createEchoVoice(req echoVoiceRequest) (echoVoiceProfile, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 80 {
		return echoVoiceProfile{}, errors.New("voice name must be 1-80 characters")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	aliases, _ := json.Marshal(req.Aliases)
	_, err := d.db.Exec(`INSERT INTO echo_voice_profiles (voice_id, name, aliases_json, state, sample_count, local_only, notes, created_at, updated_at) VALUES (?, ?, ?, ?, 0, 1, ?, ?, ?)`, strings.ToLower(strings.ReplaceAll(name, " ", "-")), name, string(aliases), echoVoiceAwaitingSample, req.Notes, now, now)
	if err != nil {
		return echoVoiceProfile{}, fmt.Errorf("create voice profile: %w", err)
	}
	profiles, err := readEchoProfiles(d.db)
	if err != nil {
		return echoVoiceProfile{}, err
	}
	for _, p := range profiles {
		if p.Name == name {
			return p, nil
		}
	}
	return echoVoiceProfile{}, errors.New("created profile could not be read")
}

func (d *daemon) enrollEchoVoice(voiceID string, req echoEnrollRequest) (echoVoiceSample, error) {
	path, err := validateEchoLocalFile(req.SourcePath)
	if err != nil {
		return echoVoiceSample{}, err
	}
	if req.StartSeconds < 0 || req.EndSeconds < 0 || (req.EndSeconds > 0 && req.EndSeconds <= req.StartSeconds) {
		return echoVoiceSample{}, errors.New("sample time bounds are invalid")
	}
	var exists int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM echo_voice_profiles WHERE voice_id = ?`, voiceID).Scan(&exists); err != nil {
		return echoVoiceSample{}, err
	}
	if exists == 0 {
		return echoVoiceSample{}, errors.New("unknown voice profile")
	}
	hash, err := echoFileHash(path)
	if err != nil {
		return echoVoiceSample{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	id := fmt.Sprintf("sample-%s-%.3f-%.3f", hash[:16], req.StartSeconds, req.EndSeconds)
	_, err = d.db.Exec(`INSERT OR REPLACE INTO echo_voice_samples (sample_id, voice_id, source_path, start_seconds, end_seconds, sha256, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, id, voiceID, path, req.StartSeconds, req.EndSeconds, hash, now)
	if err != nil {
		return echoVoiceSample{}, err
	}
	_, err = d.db.Exec(`UPDATE echo_voice_profiles SET state = ?, sample_count = (SELECT COUNT(*) FROM echo_voice_samples WHERE voice_id = ?), updated_at = ? WHERE voice_id = ?`, echoVoiceEnrolled, voiceID, now, voiceID)
	if err != nil {
		return echoVoiceSample{}, err
	}
	return echoVoiceSample{SampleID: id, VoiceID: voiceID, SourcePath: path, StartSeconds: req.StartSeconds, EndSeconds: req.EndSeconds, SHA256: hash, CreatedAt: now}, nil
}

func (d *daemon) importEchoRun(req echoImportRequest) (echoRun, error) {
	source, err := validateEchoLocalFile(req.SourcePath)
	if err != nil {
		return echoRun{}, err
	}
	diarization, err := validateEchoLocalFile(req.DiarizationPath)
	if err != nil {
		return echoRun{}, err
	}
	f, err := os.Open(diarization)
	if err != nil {
		return echoRun{}, err
	}
	defer f.Close()
	reader := csv.NewReader(bufio.NewReader(f))
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	if _, err := reader.Read(); err != nil {
		return echoRun{}, err
	}
	type row struct {
		start, end float64
		speaker    string
	}
	var rows []row
	speakers := map[string]bool{}
	for {
		fields, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return echoRun{}, readErr
		}
		if len(fields) < 3 {
			continue
		}
		start, e1 := strconv.ParseFloat(fields[0], 64)
		end, e2 := strconv.ParseFloat(fields[1], 64)
		if e1 != nil || e2 != nil || end < start || strings.TrimSpace(fields[2]) == "" {
			continue
		}
		speaker := strings.TrimSpace(fields[2])
		rows = append(rows, row{start, end, speaker})
		speakers[speaker] = true
	}
	if len(rows) == 0 {
		return echoRun{}, errors.New("diarization file contains no valid segments")
	}
	hash, err := echoFileHash(source)
	if err != nil {
		return echoRun{}, err
	}
	runID := req.RunID
	if runID == "" {
		runID = "echo-" + hash[:16]
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	count := req.SpeakerCount
	if count == 0 {
		count = len(speakers)
	}
	tx, err := d.db.Begin()
	if err != nil {
		return echoRun{}, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT OR REPLACE INTO echo_runs (run_id, source_path, source_sha256, speaker_count, status, diarization_path, segment_count, created_at, completed_at) VALUES (?, ?, ?, ?, 'completed', ?, ?, ?, ?)`, runID, source, hash, count, diarization, len(rows), now, now)
	if err != nil {
		return echoRun{}, err
	}
	if _, err = tx.Exec(`DELETE FROM echo_speaker_segments WHERE run_id = ?`, runID); err != nil {
		return echoRun{}, err
	}
	stmt, err := tx.Prepare(`INSERT INTO echo_speaker_segments (segment_id, run_id, start_seconds, end_seconds, anonymous_speaker, mapping_state) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return echoRun{}, err
	}
	defer stmt.Close()
	for i, item := range rows {
		if _, err = stmt.Exec(fmt.Sprintf("%s-%06d", runID, i), runID, item.start, item.end, item.speaker, echoMappingAnonymous); err != nil {
			return echoRun{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return echoRun{}, err
	}
	return echoRun{RunID: runID, SourcePath: source, SourceSHA256: hash, SpeakerCount: count, Status: "completed", DiarizationPath: diarization, SegmentCount: len(rows), CreatedAt: now, CompletedAt: now}, nil
}

func (d *daemon) handleEchoSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := d.echoSnapshotForRun(r.URL.Query().Get("run_id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, apiError{err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}
func (d *daemon) handleEchoVoices(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		profiles, err := readEchoProfiles(d.db)
		if err != nil {
			writeJSON(w, 500, apiError{err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"profiles": profiles, "note": "Profiles remain anonymous until a human confirms a speaker."})
		return
	}
	var req echoVoiceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, 400, apiError{err.Error()})
		return
	}
	profile, err := d.createEchoVoice(req)
	if err != nil {
		writeJSON(w, 409, apiError{err.Error()})
		return
	}
	writeJSON(w, 201, profile)
}
func (d *daemon) handleEchoEnroll(w http.ResponseWriter, r *http.Request) {
	var req echoEnrollRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, 400, apiError{err.Error()})
		return
	}
	sample, err := d.enrollEchoVoice(r.PathValue("id"), req)
	if err != nil {
		writeJSON(w, 400, apiError{err.Error()})
		return
	}
	writeJSON(w, 201, sample)
}
func (d *daemon) handleEchoImport(w http.ResponseWriter, r *http.Request) {
	var req echoImportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, 400, apiError{err.Error()})
		return
	}
	run, err := d.importEchoRun(req)
	if err != nil {
		writeJSON(w, 400, apiError{err.Error()})
		return
	}
	writeJSON(w, 201, run)
}
func (d *daemon) handleEchoConfirm(w http.ResponseWriter, r *http.Request) {
	var req echoConfirmRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, 400, apiError{err.Error()})
		return
	}
	speaker := strings.TrimSpace(req.Speaker)
	if req.RunID == "" || speaker == "" {
		writeJSON(w, 400, apiError{"run_id and speaker are required"})
		return
	}
	var voiceID string
	if err := d.db.QueryRow(`SELECT voice_id FROM echo_voice_profiles WHERE name = ? OR voice_id = ?`, r.PathValue("id"), r.PathValue("id")).Scan(&voiceID); err != nil {
		writeJSON(w, 404, apiError{"unknown voice profile"})
		return
	}
	result, err := d.db.Exec(`UPDATE echo_speaker_segments SET voice_id = ?, mapping_state = 'confirmed' WHERE run_id = ? AND anonymous_speaker = ?`, voiceID, req.RunID, speaker)
	if err != nil {
		writeJSON(w, 500, apiError{err.Error()})
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		writeJSON(w, 404, apiError{"no matching anonymous speaker in that run"})
		return
	}
	writeJSON(w, 200, map[string]any{"confirmed": n, "run_id": req.RunID, "speaker": speaker, "voice_id": voiceID})
}

func (d *daemon) handleEchoSegmentConfirm(w http.ResponseWriter, r *http.Request) {
	var request echoSegmentConfirmRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	voiceID := strings.TrimSpace(request.VoiceID)
	if voiceID == "" {
		writeError(w, http.StatusBadRequest, errors.New("voice_id is required"))
		return
	}
	var exists int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM echo_voice_profiles WHERE voice_id = ?`, voiceID).Scan(&exists); err != nil || exists == 0 {
		writeError(w, http.StatusNotFound, errors.New("unknown voice profile"))
		return
	}
	result, err := d.db.Exec(`UPDATE echo_speaker_segments SET voice_id = ?, mapping_state = ? WHERE segment_id = ?`, voiceID, echoMappingConfirmed, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		writeError(w, http.StatusNotFound, errors.New("unknown Echo segment"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"confirmed": true, "segment_id": r.PathValue("id"), "voice_id": voiceID})
}

func (d *daemon) handleEchoSegmentClear(w http.ResponseWriter, r *http.Request) {
	result, err := d.db.Exec(`UPDATE echo_speaker_segments SET voice_id = NULL, mapping_state = ? WHERE segment_id = ?`, echoMappingAnonymous, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		writeError(w, http.StatusNotFound, errors.New("unknown Echo segment"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cleared": true, "segment_id": r.PathValue("id")})
}

func (d *daemon) handleEchoVoiceRename(w http.ResponseWriter, r *http.Request) {
	var request echoVoiceRenameRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	name := strings.TrimSpace(request.Name)
	if name == "" || len(name) > 80 {
		writeError(w, http.StatusBadRequest, errors.New("voice name must be 1-80 characters"))
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := d.db.Exec(`UPDATE echo_voice_profiles SET name = ?, updated_at = ? WHERE voice_id = ?`, name, now, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusConflict, errors.New("voice profile name is already in use or invalid"))
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		writeError(w, http.StatusNotFound, errors.New("unknown voice profile"))
		return
	}
	profiles, err := readEchoProfiles(d.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	for _, profile := range profiles {
		if profile.VoiceID == r.PathValue("id") {
			writeJSON(w, http.StatusOK, profile)
			return
		}
	}
	writeError(w, http.StatusInternalServerError, errors.New("renamed voice profile could not be read"))
}

func (d *daemon) handleEchoVoiceArchive(w http.ResponseWriter, r *http.Request) {
	voiceID := r.PathValue("id")
	if voiceID == "" || voiceID == "me" {
		writeError(w, http.StatusBadRequest, errors.New("the default self profile cannot be removed"))
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := d.db.Exec(`UPDATE echo_voice_profiles SET state = ?, updated_at = ? WHERE voice_id = ? AND state != ?`, echoVoiceDisabled, now, voiceID, echoVoiceDisabled)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		writeError(w, http.StatusNotFound, errors.New("unknown or already removed voice profile"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"archived": true, "voice_id": voiceID, "note": "The profile is hidden from the active list; historical samples and assignments remain inspectable."})
}
func (d *daemon) handleEchoCommand(w http.ResponseWriter, r *http.Request) {
	var req echoCommandRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, 400, apiError{err.Error()})
		return
	}
	input := strings.TrimSpace(req.Input)
	if !strings.HasPrefix(input, "/echo") {
		writeJSON(w, 400, apiError{"command must start with /echo"})
		return
	}
	fields := strings.Fields(input)
	if len(fields) == 1 || (len(fields) == 2 && fields[1] == "voices") {
		snapshot, err := d.echoSnapshot()
		if err != nil {
			writeJSON(w, 500, apiError{err.Error()})
			return
		}
		writeJSON(w, 200, snapshot)
		return
	}
	if len(fields) >= 4 && fields[1] == "voice" && fields[2] == "add" {
		profile, err := d.createEchoVoice(echoVoiceRequest{Name: strings.Join(fields[3:], " ")})
		if err != nil {
			writeJSON(w, 409, apiError{err.Error()})
			return
		}
		writeJSON(w, 201, map[string]any{"profile": profile, "message": "Voice profile created; it is awaiting a local sample."})
		return
	}
	if len(fields) >= 5 && fields[1] == "voice" && fields[2] == "enroll" {
		req := echoEnrollRequest{SourcePath: fields[4]}
		for i := 5; i+1 < len(fields); i += 2 {
			value, err := strconv.ParseFloat(fields[i+1], 64)
			if err != nil {
				writeJSON(w, 400, apiError{"sample bounds must be seconds"})
				return
			}
			switch fields[i] {
			case "--start":
				req.StartSeconds = value
			case "--end":
				req.EndSeconds = value
			default:
				writeJSON(w, 400, apiError{"supported options: --start seconds --end seconds"})
				return
			}
		}
		sample, err := d.enrollEchoVoice(fields[3], req)
		if err != nil {
			writeJSON(w, 400, apiError{err.Error()})
			return
		}
		writeJSON(w, 201, map[string]any{"sample": sample, "message": "Local sample reference recorded; the source audio was not copied."})
		return
	}
	writeJSON(w, 400, apiError{"supported commands: /echo voices | /echo voice add <name> | /echo voice enroll <voice-id> <audio-path>"})
}
