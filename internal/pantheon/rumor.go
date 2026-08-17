package pantheon

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	rumorStateDetected     = "detected"
	rumorStateWaiting      = "waiting_stable"
	rumorStateQueued       = "queued"
	rumorStateTranscribing = "transcribing"
	rumorStateRetryWait    = "retry_wait"
	rumorStateCompleted    = "completed"
	rumorStateFailed       = "failed"
	rumorStateIgnored      = "ignored"
	rumorStateDuplicate    = "duplicate"
)

var rumorAudioExtensions = map[string]bool{
	".flac": true,
	".mp3":  true,
	".ogg":  true,
	".wav":  true,
}

type rumorSource struct {
	SourceID         string `json:"source_id"`
	SourcePath       string `json:"source_path"`
	SourceFilename   string `json:"source_filename"`
	SourceHash       string `json:"source_hash"`
	Size             int64  `json:"size"`
	ModifiedAt       string `json:"modified_at"`
	DetectedAt       string `json:"detected_at"`
	QueuedAt         string `json:"queued_at,omitempty"`
	CompletedAt      string `json:"completed_at,omitempty"`
	State            string `json:"state"`
	Attempts         int    `json:"attempts"`
	NextAttemptAt    string `json:"next_attempt_at,omitempty"`
	LastError        string `json:"last_error,omitempty"`
	FailureKind      string `json:"failure_kind,omitempty"`
	TranscriptPath   string `json:"transcript_path,omitempty"`
	MetadataPath     string `json:"metadata_path,omitempty"`
	DetectedLanguage string `json:"detected_language,omitempty"`
	WhisperBinary    string `json:"whisper_binary,omitempty"`
	WhisperModel     string `json:"whisper_model,omitempty"`
	IntakeOccurrence string `json:"intake_occurrence_id,omitempty"`
	IntakeState      string `json:"intake_state,omitempty"`
	IntakeError      string `json:"intake_error,omitempty"`
	AlertSent        bool   `json:"alert_sent"`
	DuplicateOf      string `json:"duplicate_of,omitempty"`
	IgnoredAt        string `json:"ignored_at,omitempty"`
	UpdatedAt        string `json:"updated_at"`
}

type rumorAttempt struct {
	AttemptID   string `json:"attempt_id"`
	SourceID    string `json:"source_id"`
	Attempt     int    `json:"attempt"`
	StartedAt   string `json:"started_at"`
	EndedAt     string `json:"ended_at,omitempty"`
	State       string `json:"state"`
	FailureKind string `json:"failure_kind,omitempty"`
	Error       string `json:"error,omitempty"`
}

type rumorTranscribeRequest struct {
	Path string `json:"path"`
}

type rumorCycleResult struct {
	Scanned          int  `json:"scanned"`
	Queued           int  `json:"queued"`
	Transcribed      int  `json:"transcribed"`
	Completed        int  `json:"completed"`
	Failed           int  `json:"failed"`
	HandoffCompleted int  `json:"handoff_completed"`
	HandoffPending   int  `json:"handoff_pending"`
	Paused           bool `json:"paused"`
}

func initializeRumorSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS rumor_sources (
			source_id TEXT PRIMARY KEY,
			source_path TEXT NOT NULL,
			source_filename TEXT NOT NULL,
			source_hash TEXT NOT NULL,
			size INTEGER NOT NULL,
			modified_at TEXT NOT NULL,
			detected_at TEXT NOT NULL,
			queued_at TEXT,
			completed_at TEXT,
			state TEXT NOT NULL,
			attempts INTEGER NOT NULL DEFAULT 0,
			next_attempt_at TEXT,
			last_error TEXT,
			failure_kind TEXT,
			transcript_path TEXT,
			metadata_path TEXT,
			detected_language TEXT,
			whisper_binary TEXT,
			whisper_model TEXT,
			intake_occurrence_id TEXT,
			intake_state TEXT,
			intake_error TEXT,
			alert_sent INTEGER NOT NULL DEFAULT 0,
			duplicate_of TEXT,
			ignored_at TEXT,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS rumor_sources_state_idx ON rumor_sources(state, next_attempt_at, detected_at)`,
		`CREATE INDEX IF NOT EXISTS rumor_sources_path_idx ON rumor_sources(source_path, detected_at DESC)`,
		`CREATE INDEX IF NOT EXISTS rumor_sources_hash_idx ON rumor_sources(source_hash, state)`,
		`CREATE TABLE IF NOT EXISTS rumor_attempts (
			attempt_id TEXT PRIMARY KEY,
			source_id TEXT NOT NULL,
			attempt INTEGER NOT NULL,
			started_at TEXT NOT NULL,
			ended_at TEXT,
			state TEXT NOT NULL,
			failure_kind TEXT,
			error TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS rumor_attempts_source_idx ON rumor_attempts(source_id, attempt)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	for column, definition := range map[string]string{
		"failure_kind": "TEXT", "intake_error": "TEXT", "completed_at": "TEXT",
		"duplicate_of": "TEXT", "ignored_at": "TEXT",
	} {
		if err := ensureRumorColumn(db, "rumor_sources", column, definition); err != nil {
			return err
		}
	}
	return nil
}

func ensureRumorColumn(db *sql.DB, table, column, definition string) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return nil
	}
	_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

func rumorSourceScanColumns() string {
	return `source_id, source_path, source_filename, source_hash, size, modified_at, detected_at,
		COALESCE(queued_at, ''), COALESCE(completed_at, ''), state, attempts,
		COALESCE(next_attempt_at, ''), COALESCE(last_error, ''), COALESCE(failure_kind, ''),
		COALESCE(transcript_path, ''), COALESCE(metadata_path, ''), COALESCE(detected_language, ''),
		COALESCE(whisper_binary, ''), COALESCE(whisper_model, ''), COALESCE(intake_occurrence_id, ''),
		COALESCE(intake_state, ''), COALESCE(intake_error, ''), alert_sent,
		COALESCE(duplicate_of, ''), COALESCE(ignored_at, ''), updated_at`
}

func scanRumorSource(scanner interface{ Scan(...any) error }) (rumorSource, error) {
	var source rumorSource
	var alert int
	err := scanner.Scan(&source.SourceID, &source.SourcePath, &source.SourceFilename, &source.SourceHash, &source.Size, &source.ModifiedAt, &source.DetectedAt, &source.QueuedAt, &source.CompletedAt, &source.State, &source.Attempts, &source.NextAttemptAt, &source.LastError, &source.FailureKind, &source.TranscriptPath, &source.MetadataPath, &source.DetectedLanguage, &source.WhisperBinary, &source.WhisperModel, &source.IntakeOccurrence, &source.IntakeState, &source.IntakeError, &alert, &source.DuplicateOf, &source.IgnoredAt, &source.UpdatedAt)
	source.AlertSent = alert == 1
	return source, err
}

func (d *daemon) loadRumorSource(id string) (rumorSource, error) {
	if err := initializeRumorSchema(d.db); err != nil {
		return rumorSource{}, err
	}
	return scanRumorSource(d.db.QueryRow(`SELECT `+rumorSourceScanColumns()+` FROM rumor_sources WHERE source_id = ?`, id))
}

func (d *daemon) rumorSources(limit int, state string) ([]rumorSource, error) {
	if err := initializeRumorSchema(d.db); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := `SELECT ` + rumorSourceScanColumns() + ` FROM rumor_sources`
	args := []any{}
	if state != "" {
		query += ` WHERE state = ?`
		args = append(args, state)
	}
	query += ` ORDER BY detected_at DESC, source_id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]rumorSource, 0)
	for rows.Next() {
		item, scanErr := scanRumorSource(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func rumorSupportedAudio(path string) bool {
	return rumorAudioExtensions[strings.ToLower(filepath.Ext(path))]
}

func rumorSourcePathAllowed(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("Rumor refuses symbolic-link audio sources")
	}
	if !info.Mode().IsRegular() {
		return errors.New("Rumor source is not a regular file")
	}
	return ensureNoSymlinkComponents(path)
}

func (d *daemon) rumorPathInWatchRoots(path string) bool {
	clean := filepath.Clean(path)
	for _, root := range d.cfg.RumorWatchPaths {
		rel, err := filepath.Rel(filepath.Clean(root), clean)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
}

func (d *daemon) rumorRegisterPath(path string, explicit bool) (rumorSource, error) {
	path, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return rumorSource{}, err
	}
	if err := rumorSourcePathAllowed(path); err != nil {
		return rumorSource{}, err
	}
	if !rumorSupportedAudio(path) {
		return rumorSource{}, fmt.Errorf("unsupported Rumor audio format %q; whisper.cpp supports flac, mp3, ogg, and wav", filepath.Ext(path))
	}
	hashed, err := hashStableIntakeFile(path)
	if err != nil {
		return rumorSource{}, fmt.Errorf("inspect stable audio source: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return rumorSource{}, err
	}
	now := time.Now().UTC()
	modified := info.ModTime().UTC().Format(time.RFC3339Nano)
	var previous rumorSource
	previous, err = d.loadLatestRumorPath(path)
	if err == nil {
		if previous.SourceHash == hashed.ContentID && previous.Size == info.Size() && previous.ModifiedAt == modified {
			switch previous.State {
			case rumorStateWaiting:
				stable := explicit || d.cfg.RumorStabilitySeconds == 0
				if !stable {
					detected, parseErr := time.Parse(time.RFC3339Nano, previous.DetectedAt)
					stable = parseErr == nil && now.Sub(detected) >= time.Duration(d.cfg.RumorStabilitySeconds)*time.Second
				}
				if stable {
					return d.queueRumorSource(previous.SourceID, now)
				}
			case rumorStateQueued, rumorStateTranscribing, rumorStateRetryWait, rumorStateCompleted, rumorStateFailed, rumorStateIgnored, rumorStateDuplicate:
				return previous, nil
			}
		}
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return rumorSource{}, err
	}
	if !explicit && !d.rumorPathInWatchRoots(path) {
		return rumorSource{}, errors.New("audio source is outside configured Rumor watch paths")
	}
	var duplicate rumorSource
	if err := d.db.QueryRow(`SELECT `+rumorSourceScanColumns()+` FROM rumor_sources WHERE source_hash = ? AND state IN (?, ?) ORDER BY completed_at, source_id LIMIT 1`, hashed.ContentID, rumorStateCompleted, rumorStateDuplicate).Scan(
		&duplicate.SourceID, &duplicate.SourcePath, &duplicate.SourceFilename, &duplicate.SourceHash, &duplicate.Size, &duplicate.ModifiedAt, &duplicate.DetectedAt, &duplicate.QueuedAt, &duplicate.CompletedAt, &duplicate.State, &duplicate.Attempts, &duplicate.NextAttemptAt, &duplicate.LastError, &duplicate.FailureKind, &duplicate.TranscriptPath, &duplicate.MetadataPath, &duplicate.DetectedLanguage, &duplicate.WhisperBinary, &duplicate.WhisperModel, &duplicate.IntakeOccurrence, &duplicate.IntakeState, &duplicate.IntakeError, new(int), &duplicate.DuplicateOf, &duplicate.IgnoredAt, &duplicate.UpdatedAt); err == nil {
		id, idErr := newLocalID("rumor-")
		if idErr != nil {
			return rumorSource{}, idErr
		}
		_, err = d.db.Exec(`INSERT INTO rumor_sources (source_id, source_path, source_filename, source_hash, size, modified_at, detected_at, completed_at, state, attempts, transcript_path, metadata_path, detected_language, whisper_binary, whisper_model, intake_occurrence_id, intake_state, duplicate_of, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, path, filepath.Base(path), hashed.ContentID, hashed.Size, modified, now.Format(time.RFC3339Nano), duplicate.CompletedAt, rumorStateDuplicate, 0, duplicate.TranscriptPath, duplicate.MetadataPath, duplicate.DetectedLanguage, duplicate.WhisperBinary, duplicate.WhisperModel, duplicate.IntakeOccurrence, duplicate.IntakeState, duplicate.SourceID, now.Format(time.RFC3339Nano))
		if err != nil {
			return rumorSource{}, err
		}
		return d.loadRumorSource(id)
	}
	id, err := newLocalID("rumor-")
	if err != nil {
		return rumorSource{}, err
	}
	state := rumorStateWaiting
	queuedAt := ""
	if explicit || d.cfg.RumorStabilitySeconds == 0 {
		state = rumorStateQueued
		queuedAt = now.Format(time.RFC3339Nano)
	}
	_, err = d.db.Exec(`INSERT INTO rumor_sources (source_id, source_path, source_filename, source_hash, size, modified_at, detected_at, queued_at, state, attempts, intake_state, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`, id, path, filepath.Base(path), hashed.ContentID, hashed.Size, modified, now.Format(time.RFC3339Nano), queuedAt, state, "pending", now.Format(time.RFC3339Nano))
	if err != nil {
		return rumorSource{}, err
	}
	return d.loadRumorSource(id)
}

func (d *daemon) loadLatestRumorPath(path string) (rumorSource, error) {
	return scanRumorSource(d.db.QueryRow(`SELECT `+rumorSourceScanColumns()+` FROM rumor_sources WHERE source_path = ? ORDER BY detected_at DESC, source_id DESC LIMIT 1`, path))
}

func (d *daemon) queueRumorSource(id string, now time.Time) (rumorSource, error) {
	stamp := now.Format(time.RFC3339Nano)
	if _, err := d.db.Exec(`UPDATE rumor_sources SET state = ?, queued_at = COALESCE(NULLIF(queued_at, ''), ?), next_attempt_at = '', last_error = '', failure_kind = '', intake_error = '', updated_at = ? WHERE source_id = ? AND state = ?`, rumorStateQueued, stamp, stamp, id, rumorStateWaiting); err != nil {
		return rumorSource{}, err
	}
	return d.loadRumorSource(id)
}

func (d *daemon) rumorScanWatchPaths() (int, error) {
	if !d.cfg.RumorEnabled || len(d.cfg.RumorWatchPaths) == 0 {
		return 0, nil
	}
	count := 0
	var firstErr error
	for _, configuredRoot := range d.cfg.RumorWatchPaths {
		root := filepath.Clean(configuredRoot)
		if err := ensureNoSymlinkComponents(root); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if firstErr == nil {
					firstErr = walkErr
				}
				return nil
			}
			if path == root {
				return nil
			}
			if hiddenOrSystemName(entry.Name()) {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() || !rumorSupportedAudio(path) {
				return nil
			}
			if _, err := d.rumorRegisterPath(path, false); err != nil {
				if firstErr == nil && !strings.Contains(err.Error(), "unsupported") {
					firstErr = err
				}
				return nil
			}
			count++
			return nil
		})
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return count, firstErr
}

func (d *daemon) rumorDueSources() ([]rumorSource, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rows, err := d.db.Query(`SELECT `+rumorSourceScanColumns()+` FROM rumor_sources WHERE state = ? OR (state = ? AND (next_attempt_at = '' OR next_attempt_at <= ?)) ORDER BY detected_at, source_id`, rumorStateQueued, rumorStateRetryWait, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]rumorSource, 0)
	for rows.Next() {
		item, scanErr := scanRumorSource(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *daemon) rumorProcessCycle(ctx context.Context) (rumorCycleResult, error) {
	d.rumorMu.Lock()
	defer d.rumorMu.Unlock()
	if err := initializeRumorSchema(d.db); err != nil {
		return rumorCycleResult{}, err
	}
	paused, err := d.paused()
	if err != nil {
		return rumorCycleResult{}, err
	}
	result := rumorCycleResult{Paused: paused}
	if paused {
		return result, nil
	}
	result.Scanned, err = d.rumorScanWatchPaths()
	if err != nil {
		return result, err
	}
	if d.cfg.RumorOnComplete == "intake" {
		completed, pending, handoffErr := d.rumorProcessPendingHandoffs()
		result.HandoffCompleted, result.HandoffPending = completed, pending
		if handoffErr != nil {
			return result, handoffErr
		}
	}
	due, err := d.rumorDueSources()
	if err != nil {
		return result, err
	}
	result.Queued = len(due)
	for _, source := range due {
		if paused, pauseErr := d.paused(); pauseErr != nil {
			return result, pauseErr
		} else if paused {
			result.Paused = true
			break
		}
		if processErr := d.processRumorSource(ctx, source); processErr != nil {
			result.Failed++
			continue
		}
		result.Transcribed++
		result.Completed++
	}
	return result, nil
}

func (d *daemon) processRumorSource(ctx context.Context, source rumorSource) error {
	started := time.Now().UTC()
	attempt := source.Attempts + 1
	result, err := d.db.Exec(`UPDATE rumor_sources SET state = ?, attempts = ?, updated_at = ? WHERE source_id = ? AND state IN (?, ?)`, rumorStateTranscribing, attempt, started.Format(time.RFC3339Nano), source.SourceID, rumorStateQueued, rumorStateRetryWait)
	if err != nil {
		return err
	}
	if changed, err := result.RowsAffected(); err != nil {
		return err
	} else if changed != 1 {
		return errors.New("Rumor source is no longer queued")
	}
	attemptID, err := newLocalID("rumor-attempt-")
	if err != nil {
		return err
	}
	if _, err := d.db.Exec(`INSERT INTO rumor_attempts (attempt_id, source_id, attempt, started_at, state) VALUES (?, ?, ?, ?, 'running')`, attemptID, source.SourceID, attempt, started.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	transcript, err := d.runRumorWhisper(ctx, source)
	if err == nil {
		err = d.persistRumorArtifacts(source, transcript)
	}
	if err != nil {
		return d.finishRumorFailure(source, attemptID, attempt, started, err)
	}
	completedAt := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := d.db.Exec(`UPDATE rumor_sources SET state = ?, completed_at = ?, next_attempt_at = '', last_error = '', failure_kind = '', updated_at = ? WHERE source_id = ?`, rumorStateCompleted, completedAt, completedAt, source.SourceID); err != nil {
		return err
	}
	if _, err := d.db.Exec(`UPDATE rumor_attempts SET ended_at = ?, state = 'completed' WHERE attempt_id = ?`, completedAt, attemptID); err != nil {
		return err
	}
	if d.cfg.RumorOnComplete == "intake" {
		if paused, pauseErr := d.paused(); pauseErr != nil {
			return pauseErr
		} else if !paused {
			if err := d.handoffRumorSource(source.SourceID); err != nil {
				_ = d.markRumorHandoffFailure(source.SourceID, err)
				return nil
			}
		}
	}
	return nil
}

type rumorTranscription struct {
	Text             string
	DetectedLanguage string
	Binary           string
	Model            string
}

func (d *daemon) rumorWhisperConfiguration() (string, string, string, error) {
	if !d.cfg.WhisperEnabled {
		return "", "", "", errors.New("local Whisper is disabled")
	}
	binary := strings.TrimSpace(d.cfg.WhisperBinary)
	model := strings.TrimSpace(d.cfg.RumorWhisperModel)
	if model == "" {
		model = strings.TrimSpace(d.cfg.WhisperModel)
	}
	for _, capability := range d.capabilities.Capabilities {
		if capability.Name != "transcribe" {
			continue
		}
		if binary == "" {
			binary = strings.TrimSpace(capability.Command)
		}
		if model == "" {
			model = strings.TrimSpace(capability.Model)
		}
	}
	if binary == "" {
		var err error
		binary, err = configuredCommand("", "whisper-cli")
		if err != nil {
			binary, err = configuredCommand("", "whisper")
			if err != nil {
				return "", "", "", errors.New("local Whisper command is unavailable")
			}
		}
	} else {
		resolved, err := configuredCommand(binary, "whisper-cli")
		if err != nil {
			return "", "", "", err
		}
		binary = resolved
	}
	if model == "" {
		return "", "", "", errors.New("local Whisper model is unavailable")
	}
	if info, err := os.Stat(model); err != nil || !info.Mode().IsRegular() {
		return "", "", "", errors.New("configured local Whisper model is unavailable")
	}
	language := strings.TrimSpace(d.cfg.RumorLanguage)
	if language == "" {
		language = strings.TrimSpace(d.cfg.WhisperLanguage)
	}
	if language == "" {
		language = "auto"
	}
	return binary, model, language, nil
}

func (d *daemon) runRumorWhisper(ctx context.Context, source rumorSource) (rumorTranscription, error) {
	binary, model, language, err := d.rumorWhisperConfiguration()
	if err != nil {
		return rumorTranscription{}, err
	}
	workDir, err := os.MkdirTemp("", "pantheon-rumor-")
	if err != nil {
		return rumorTranscription{}, err
	}
	defer os.RemoveAll(workDir)
	outputBase := filepath.Join(workDir, "transcript")
	args := []string{"-m", model, "-f", source.SourcePath, "--output-txt", "--output-json-full", "--output-file", outputBase, "--no-prints", "--no-timestamps", "-l", language}
	cmd := exec.CommandContext(ctx, binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	outputPath := outputBase + ".txt"
	transcript, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		transcript = stdout.Bytes()
	}
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return rumorTranscription{}, fmt.Errorf("Whisper transcription failed: %s", message)
	}
	transcript = bytes.TrimSpace(transcript)
	if len(transcript) == 0 {
		return rumorTranscription{}, errors.New("Whisper produced no transcript")
	}
	detectedLanguage := rumorDetectedLanguage(stderr.String() + "\n" + stdout.String() + "\n" + string(transcript))
	if metadata, readErr := os.ReadFile(outputBase + ".json"); readErr == nil {
		detectedLanguage = firstNonEmpty(rumorJSONLanguage(metadata), detectedLanguage)
	}
	return rumorTranscription{Text: string(transcript), DetectedLanguage: detectedLanguage, Binary: binary, Model: model}, nil
}

func rumorJSONLanguage(data []byte) string {
	var value any
	if json.Unmarshal(data, &value) != nil {
		return ""
	}
	var visit func(any) string
	visit = func(candidate any) string {
		switch item := candidate.(type) {
		case map[string]any:
			for key, child := range item {
				if strings.EqualFold(key, "language") {
					if text, ok := child.(string); ok {
						return strings.TrimSpace(text)
					}
				}
				if found := visit(child); found != "" {
					return found
				}
			}
		case []any:
			for _, child := range item {
				if found := visit(child); found != "" {
					return found
				}
			}
		}
		return ""
	}
	return visit(value)
}

func rumorDetectedLanguage(output string) string {
	for _, line := range strings.Split(output, "\n") {
		lower := strings.ToLower(strings.TrimSpace(line))
		for _, marker := range []string{"auto-detected language:", "detected language:", "language:"} {
			if index := strings.Index(lower, marker); index >= 0 {
				value := strings.TrimSpace(line[index+len(marker):])
				value = strings.Trim(value, "[](){} \t\r")
				if value != "" && len(value) <= 16 && !strings.ContainsAny(value, " \t") {
					return strings.ToLower(value)
				}
			}
		}
	}
	return ""
}

func rumorSafeName(name string) string {
	name = strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "audio"
	}
	return b.String()
}

func (d *daemon) persistRumorArtifacts(source rumorSource, transcription rumorTranscription) error {
	destination := strings.TrimSpace(d.cfg.RumorTranscriptDestination)
	if destination == "" {
		return errors.New("Rumor transcript destination is not configured")
	}
	if d.cfg.VaultPath != "" {
		rel, err := filepath.Rel(filepath.Clean(d.cfg.VaultPath), filepath.Clean(destination))
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return errors.New("Rumor transcript destination must remain beneath the configured Vault")
		}
	}
	if err := ensureNoSymlinkComponents(destination); err != nil {
		return err
	}
	digest := strings.TrimPrefix(source.SourceHash, "sha256:")
	if len(digest) > 16 {
		digest = digest[:16]
	}
	base := rumorSafeName(source.SourceFilename) + "--" + digest
	transcriptPath := filepath.Join(destination, base+".md")
	metadataPath := filepath.Join(destination, base+".json")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var body strings.Builder
	fmt.Fprintf(&body, "---\npantheon:\n  kind: rumor-transcript\n  source_id: %s\n  source_path: %s\n  source_filename: %s\n  source_hash: %s\n  transcribed_at: %s\n  whisper_binary: %s\n  whisper_model: %s\n  language: %s\n---\n\n# Transcript — %s\n\n%s\n", yamlQuote(source.SourceID), yamlQuote(source.SourcePath), yamlQuote(source.SourceFilename), yamlQuote(source.SourceHash), yamlQuote(now), yamlQuote(transcription.Binary), yamlQuote(transcription.Model), yamlQuote(firstNonEmpty(transcription.DetectedLanguage, d.cfg.RumorLanguage)), yamlQuote(source.SourceFilename), transcription.Text)
	if err := writeRumorArtifact(transcriptPath, []byte(body.String())); err != nil {
		return err
	}
	metadata := map[string]any{"source_id": source.SourceID, "source_path": source.SourcePath, "source_filename": source.SourceFilename, "source_hash": source.SourceHash, "size": source.Size, "modified_at": source.ModifiedAt, "transcribed_at": now, "whisper_binary": transcription.Binary, "whisper_model": transcription.Model, "language": firstNonEmpty(transcription.DetectedLanguage, d.cfg.RumorLanguage), "transcript_path": transcriptPath}
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	metadataBytes = append(metadataBytes, '\n')
	if err := writeRumorArtifact(metadataPath, metadataBytes); err != nil {
		return err
	}
	_, err = d.db.Exec(`UPDATE rumor_sources SET transcript_path = ?, metadata_path = ?, detected_language = ?, whisper_binary = ?, whisper_model = ?, updated_at = ? WHERE source_id = ?`, transcriptPath, metadataPath, firstNonEmpty(transcription.DetectedLanguage, d.cfg.RumorLanguage), transcription.Binary, transcription.Model, now, source.SourceID)
	return err
}

func writeRumorArtifact(path string, data []byte) error {
	if err := writeArtifact(path, data); err == nil {
		return nil
	} else if !strings.Contains(err.Error(), "already exists") {
		return err
	}
	existing, readErr := os.ReadFile(path)
	if readErr != nil {
		return readErr
	}
	if !bytes.Equal(existing, data) {
		return errors.New("Rumor artifact collision would overwrite a different artifact")
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "unknown"
}

func (d *daemon) finishRumorFailure(source rumorSource, attemptID string, attempt int, started time.Time, failure error) error {
	now := time.Now().UTC()
	kind := "transcription"
	if strings.Contains(failure.Error(), "artifact") || strings.Contains(failure.Error(), "destination") || strings.Contains(failure.Error(), "collision") {
		kind = "artifact"
	}
	state := rumorStateRetryWait
	next := now.Add(time.Duration(attempt) * time.Second).Format(time.RFC3339Nano)
	if attempt >= d.cfg.RumorRetryLimit {
		state = rumorStateFailed
		next = ""
	}
	message := failure.Error()
	_, dbErr := d.db.Exec(`UPDATE rumor_sources SET state = ?, next_attempt_at = ?, last_error = ?, failure_kind = ?, updated_at = ? WHERE source_id = ?`, state, next, message, kind, now.Format(time.RFC3339Nano), source.SourceID)
	if dbErr != nil {
		return dbErr
	}
	if _, dbErr := d.db.Exec(`UPDATE rumor_attempts SET ended_at = ?, state = 'failed', failure_kind = ?, error = ? WHERE attempt_id = ?`, now.Format(time.RFC3339Nano), kind, message, attemptID); dbErr != nil {
		return dbErr
	}
	if state == rumorStateFailed {
		if alertErr := d.alertRumorFailure(source.SourceID, message); alertErr != nil {
			return alertErr
		}
	}
	_ = started
	return failure
}

func (d *daemon) alertRumorFailure(sourceID, message string) error {
	source, err := d.loadRumorSource(sourceID)
	if err != nil {
		return err
	}
	if source.AlertSent {
		return nil
	}
	output, commandErr := exec.Command(notifyCommand, "pantheon-alerts", "Rumor paper jam", fmt.Sprintf("Rumor source %s failed after %d attempt(s): %s", sourceID, source.Attempts, message)).CombinedOutput()
	if commandErr != nil {
		d.audit("rumor.alert", sourceID, "notify", "", "failed", "", strings.TrimSpace(string(output)), 0)
	} else {
		d.audit("rumor.alert", sourceID, "notify", "", "completed", "paper-jam alert sent", "", 0)
	}
	_, err = d.db.Exec(`UPDATE rumor_sources SET alert_sent = 1, updated_at = ? WHERE source_id = ?`, time.Now().UTC().Format(time.RFC3339Nano), sourceID)
	return err
}

func (d *daemon) handoffRumorSource(sourceID string) error {
	source, err := d.loadRumorSource(sourceID)
	if err != nil {
		return err
	}
	if source.TranscriptPath == "" {
		return errors.New("Rumor transcript artifact is unavailable for Intake handoff")
	}
	if source.IntakeState == "received" {
		return nil
	}
	if d.cfg.IntakePath == "" {
		return errors.New("Intake path is not configured")
	}
	data, err := os.ReadFile(source.TranscriptPath)
	if err != nil {
		return err
	}
	intakeDir := filepath.Join(d.cfg.IntakePath, "Rumor")
	intakePath := filepath.Join(intakeDir, filepath.Base(source.TranscriptPath))
	if err := ensureNoSymlinkComponents(intakePath); err != nil {
		return err
	}
	if err := writeRumorArtifact(intakePath, data); err != nil {
		return err
	}
	if _, err := d.runIntakeScan(); err != nil {
		_, _ = d.db.Exec(`UPDATE rumor_sources SET intake_state = 'pending', intake_error = ?, updated_at = ? WHERE source_id = ?`, err.Error(), time.Now().UTC().Format(time.RFC3339Nano), sourceID)
		return err
	}
	hashed, err := hashStableIntakeFile(intakePath)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(d.cfg.IntakePath, intakePath)
	if err != nil {
		return err
	}
	var occurrenceID string
	if err := d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences WHERE relative_path = ? AND content_id = ? AND state = ? ORDER BY first_seen_at DESC LIMIT 1`, filepath.ToSlash(relative), hashed.ContentID, occurrencePresent).Scan(&occurrenceID); err != nil {
		return err
	}
	_, err = d.db.Exec(`UPDATE rumor_sources SET intake_occurrence_id = ?, intake_state = 'received', intake_error = '', updated_at = ? WHERE source_id = ?`, occurrenceID, time.Now().UTC().Format(time.RFC3339Nano), sourceID)
	return err
}

func (d *daemon) markRumorHandoffFailure(sourceID string, handoffErr error) error {
	_, err := d.db.Exec(`UPDATE rumor_sources SET intake_state = 'pending', intake_error = ?, updated_at = ? WHERE source_id = ?`, handoffErr.Error(), time.Now().UTC().Format(time.RFC3339Nano), sourceID)
	return err
}

func (d *daemon) rumorProcessPendingHandoffs() (int, int, error) {
	rows, err := d.db.Query(`SELECT source_id FROM rumor_sources WHERE state = ? AND intake_state = 'pending' ORDER BY completed_at, source_id`, rumorStateCompleted)
	if err != nil {
		return 0, 0, err
	}
	var sourceIDs []string
	for rows.Next() {
		var sourceID string
		if err := rows.Scan(&sourceID); err != nil {
			rows.Close()
			return 0, 0, err
		}
		sourceIDs = append(sourceIDs, sourceID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, 0, err
	}
	rows.Close()
	completed, pending := 0, 0
	for _, sourceID := range sourceIDs {
		if err := d.handoffRumorSource(sourceID); err != nil {
			_ = d.markRumorHandoffFailure(sourceID, err)
			pending++
			continue
		}
		completed++
	}
	return completed, pending, rows.Err()
}

func (d *daemon) rumorInspection() (map[string]any, error) {
	if err := initializeRumorSchema(d.db); err != nil {
		return nil, err
	}
	sources, err := d.rumorSources(50, "")
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, source := range sources {
		counts[source.State]++
	}
	return map[string]any{"enabled": d.cfg.RumorEnabled, "watch_paths": d.cfg.RumorWatchPaths, "transcript_destination": d.cfg.RumorTranscriptDestination, "whisper_model": d.cfg.RumorWhisperModel, "language": d.cfg.RumorLanguage, "retry_limit": d.cfg.RumorRetryLimit, "stability_seconds": d.cfg.RumorStabilitySeconds, "poll_seconds": d.cfg.RumorPollSeconds, "counts": counts, "recent_sources": sources}, nil
}

func (d *daemon) rumorQueueInspection() (map[string]any, error) {
	sources, err := d.rumorDueSources()
	if err != nil {
		return nil, err
	}
	return map[string]any{"queued": sources}, nil
}

func (d *daemon) rumorAttempts(sourceID string) ([]rumorAttempt, error) {
	rows, err := d.db.Query(`SELECT attempt_id, source_id, attempt, started_at, COALESCE(ended_at, ''), state, COALESCE(failure_kind, ''), COALESCE(error, '') FROM rumor_attempts WHERE source_id = ? ORDER BY attempt`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]rumorAttempt, 0)
	for rows.Next() {
		var item rumorAttempt
		if err := rows.Scan(&item.AttemptID, &item.SourceID, &item.Attempt, &item.StartedAt, &item.EndedAt, &item.State, &item.FailureKind, &item.Error); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *daemon) rumorTranscribePath(ctx context.Context, path string) (rumorSource, error) {
	d.rumorMu.Lock()
	defer d.rumorMu.Unlock()
	source, err := d.rumorRegisterPath(path, true)
	if err != nil {
		return rumorSource{}, err
	}
	if source.State == rumorStateFailed || source.State == rumorStateIgnored {
		if _, err := d.db.Exec(`UPDATE rumor_sources SET state = ?, attempts = 0, next_attempt_at = '', last_error = '', failure_kind = '', alert_sent = 0, ignored_at = '', intake_state = 'pending', updated_at = ? WHERE source_id = ?`, rumorStateQueued, time.Now().UTC().Format(time.RFC3339Nano), source.SourceID); err != nil {
			return rumorSource{}, err
		}
		source, err = d.loadRumorSource(source.SourceID)
		if err != nil {
			return rumorSource{}, err
		}
	}
	if source.State == rumorStateCompleted || source.State == rumorStateDuplicate {
		return source, nil
	}
	if err := d.processRumorSource(ctx, source); err != nil {
		loaded, loadErr := d.loadRumorSource(source.SourceID)
		if loadErr != nil {
			return rumorSource{}, loadErr
		}
		return loaded, err
	}
	return d.loadRumorSource(source.SourceID)
}

func (d *daemon) retryRumorSource(id string) (rumorSource, error) {
	d.rumorMu.Lock()
	defer d.rumorMu.Unlock()
	source, err := d.loadRumorSource(id)
	if err != nil {
		return rumorSource{}, err
	}
	if source.State != rumorStateFailed && source.State != rumorStateIgnored && source.State != rumorStateRetryWait {
		return rumorSource{}, errors.New("only failed, ignored, or waiting Rumor sources can be retried")
	}
	if _, err := d.db.Exec(`UPDATE rumor_sources SET state = ?, attempts = 0, next_attempt_at = '', last_error = '', failure_kind = '', alert_sent = 0, ignored_at = '', intake_state = 'pending', updated_at = ? WHERE source_id = ?`, rumorStateQueued, time.Now().UTC().Format(time.RFC3339Nano), id); err != nil {
		return rumorSource{}, err
	}
	return d.loadRumorSource(id)
}

func (d *daemon) ignoreRumorSource(id string) (rumorSource, error) {
	d.rumorMu.Lock()
	defer d.rumorMu.Unlock()
	source, err := d.loadRumorSource(id)
	if err != nil {
		return rumorSource{}, err
	}
	if source.State == rumorStateCompleted || source.State == rumorStateDuplicate {
		return rumorSource{}, errors.New("completed Rumor sources cannot be ignored")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := d.db.Exec(`UPDATE rumor_sources SET state = ?, ignored_at = ?, next_attempt_at = '', updated_at = ? WHERE source_id = ?`, rumorStateIgnored, now, now, id); err != nil {
		return rumorSource{}, err
	}
	return d.loadRumorSource(id)
}

func (d *daemon) startRumorWatcher() func() {
	if !d.cfg.RumorEnabled || len(d.cfg.RumorWatchPaths) == 0 {
		return func() {}
	}
	interval := time.Duration(d.cfg.RumorPollSeconds) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cycle := func() {
			if _, err := d.rumorProcessCycle(ctx); err != nil {
				d.audit("rumor.watcher", strings.Join(d.cfg.RumorWatchPaths, ","), "transcribe", "", "failed", "", err.Error(), 0)
			}
		}
		cycle()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cycle()
			}
		}
	}()
	return func() {
		cancel()
		wg.Wait()
	}
}

func (d *daemon) requireRumorControl(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get(controlTokenHeader) == d.controlToken {
		return true
	}
	writeError(w, http.StatusForbidden, errors.New("Rumor mutation requires the local CLI control boundary"))
	return false
}

func (d *daemon) handleShowRumor(w http.ResponseWriter, r *http.Request) {
	inspection, err := d.rumorInspection()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, inspection)
}

func (d *daemon) handleShowRumorQueue(w http.ResponseWriter, r *http.Request) {
	inspection, err := d.rumorQueueInspection()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, inspection)
}

func (d *daemon) handleShowRumorSource(w http.ResponseWriter, r *http.Request) {
	source, err := d.loadRumorSource(r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("Rumor source not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	attempts, err := d.rumorAttempts(source.SourceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"source": source, "attempts": attempts})
}

func (d *daemon) handleRumorScan(w http.ResponseWriter, r *http.Request) {
	if !d.requireRumorControl(w, r) {
		return
	}
	result, err := d.rumorProcessCycle(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (d *daemon) handleRumorTranscribe(w http.ResponseWriter, r *http.Request) {
	if !d.requireRumorControl(w, r) {
		return
	}
	paused, err := d.paused()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if paused {
		writeError(w, http.StatusServiceUnavailable, errors.New("Pantheon is paused; Rumor transcription is blocked"))
		return
	}
	var request rumorTranscribeRequest
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Path) == "" {
		if err == nil {
			err = errors.New("path is required")
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	source, err := d.rumorTranscribePath(r.Context(), request.Path)
	if err != nil && source.SourceID == "" {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (d *daemon) handleRumorRetry(w http.ResponseWriter, r *http.Request) {
	if !d.requireRumorControl(w, r) {
		return
	}
	source, err := d.retryRumorSource(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (d *daemon) handleRumorIgnore(w http.ResponseWriter, r *http.Request) {
	if !d.requireRumorControl(w, r) {
		return
	}
	source, err := d.ignoreRumorSource(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (d *daemon) handleRumorAttempts(w http.ResponseWriter, r *http.Request) {
	attempts, err := d.rumorAttempts(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, attempts)
}

func (d *daemon) rumorStatusSummary() map[string]any {
	return map[string]any{"enabled": d.cfg.RumorEnabled, "watch_paths": append([]string(nil), d.cfg.RumorWatchPaths...), "transcript_destination": d.cfg.RumorTranscriptDestination}
}
