package pantheon

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultRawInputDays       = 30
	retentionAwaitingMarkdown = "awaiting_verified_markdown"
	occurrencePresent         = "present"
	occurrenceDeparted        = "departed"
)

type intakeScanResult struct {
	ScanID         string `json:"scan_id"`
	IntakePath     string `json:"intake_path"`
	New            int    `json:"new"`
	Unchanged      int    `json:"unchanged"`
	Departed       int    `json:"departed"`
	Deferred       int    `json:"deferred"`
	Skipped        int    `json:"skipped"`
	Failed         int    `json:"failed"`
	FatherTimeYear int    `json:"father_time_year"`
	ChroniclePath  string `json:"chronicle_path,omitempty"`
	Chronology     string `json:"chronology"`
	AuditID        string `json:"audit_id,omitempty"`
}

type intakeOccurrence struct {
	OccurrenceID        string `json:"occurrence_id"`
	ContentID           string `json:"content_id"`
	RelativePath        string `json:"relative_path"`
	OriginalFilename    string `json:"original_filename"`
	Size                int64  `json:"size"`
	Type                string `json:"type,omitempty"`
	Extension           string `json:"extension,omitempty"`
	FirstSeenAt         string `json:"first_seen_at"`
	LastSeenAt          string `json:"last_seen_at"`
	State               string `json:"state"`
	FatherTimeYear      int    `json:"father_time_year"`
	ProcessingState     string `json:"processing_state"`
	RetentionDays       int    `json:"retention_policy_days"`
	RetentionState      string `json:"retention_state"`
	RetentionDueAt      string `json:"retention_due_at,omitempty"`
	ChronicleState      string `json:"chronicle_state"`
	ProcessingStartedAt string `json:"processing_started_at,omitempty"`
	NormalizedAt        string `json:"normalized_at,omitempty"`
	VerifiedAt          string `json:"verified_at,omitempty"`
	NormalizedPath      string `json:"normalized_path,omitempty"`
	ArtifactSHA256      string `json:"artifact_sha256,omitempty"`
	ArtifactBodySHA256  string `json:"artifact_body_sha256,omitempty"`
	Processor           string `json:"processor,omitempty"`
	ProcessorVersion    string `json:"processor_version,omitempty"`
	ProcessingError     string `json:"processing_error,omitempty"`
	Warnings            string `json:"warnings,omitempty"`
}

func initializeIntakeSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS intake_contents (
			content_id TEXT PRIMARY KEY, size INTEGER NOT NULL, type TEXT NOT NULL,
			extension TEXT NOT NULL, first_seen_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS intake_occurrences (
			occurrence_id TEXT PRIMARY KEY, content_id TEXT NOT NULL, relative_path TEXT NOT NULL,
			original_filename TEXT NOT NULL, first_seen_at TEXT NOT NULL, last_seen_at TEXT NOT NULL,
			state TEXT NOT NULL, father_time_year INTEGER NOT NULL, processing_state TEXT NOT NULL,
			retention_policy_days INTEGER NOT NULL, retention_state TEXT NOT NULL,
			retention_eligible_at TEXT, retention_due_at TEXT, chronicle_state TEXT NOT NULL,
			processing_started_at TEXT, normalized_at TEXT, verified_at TEXT,
			normalized_path TEXT, artifact_sha256 TEXT, artifact_body_sha256 TEXT,
			processor TEXT, processor_version TEXT, processing_error TEXT, warnings TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS intake_occurrences_path_idx ON intake_occurrences(relative_path, first_seen_at DESC)`,
		`CREATE INDEX IF NOT EXISTS intake_occurrences_state_idx ON intake_occurrences(state, first_seen_at DESC)`,
		`CREATE TABLE IF NOT EXISTS intake_chronicle_events (
			occurrence_id TEXT PRIMARY KEY, event_type TEXT NOT NULL, event_at TEXT NOT NULL,
			father_time_year INTEGER NOT NULL, state TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS intake_scans (
			scan_id TEXT PRIMARY KEY, started_at TEXT NOT NULL, ended_at TEXT,
			intake_path TEXT NOT NULL, new_count INTEGER NOT NULL DEFAULT 0,
			unchanged_count INTEGER NOT NULL DEFAULT 0, departed_count INTEGER NOT NULL DEFAULT 0,
			deferred_count INTEGER NOT NULL DEFAULT 0, skipped_count INTEGER NOT NULL DEFAULT 0,
			failed_count INTEGER NOT NULL DEFAULT 0, father_time_year INTEGER NOT NULL DEFAULT 0,
			chronology TEXT NOT NULL DEFAULT 'pending',
			outcome TEXT NOT NULL DEFAULT 'running', error TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS present_missives (
			missive_id TEXT PRIMARY KEY, created_at TEXT NOT NULL, original_request TEXT NOT NULL,
			occurrence_id TEXT NOT NULL, content_id TEXT NOT NULL, source_path TEXT NOT NULL,
			source_type TEXT NOT NULL, normalized_path TEXT, processor TEXT NOT NULL,
			processor_version TEXT, warnings TEXT, verification_state TEXT NOT NULL,
			verified_at TEXT, retention_state TEXT NOT NULL, retention_due_at TEXT,
			reasoning_state TEXT NOT NULL
		)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	for column, definition := range map[string]string{
		"processing_started_at": "TEXT", "normalized_at": "TEXT", "verified_at": "TEXT",
		"normalized_path": "TEXT", "artifact_sha256": "TEXT", "artifact_body_sha256": "TEXT",
		"processor": "TEXT", "processor_version": "TEXT", "processing_error": "TEXT", "warnings": "TEXT",
	} {
		if err := ensureIntakeColumn(db, "intake_occurrences", column, definition); err != nil {
			return err
		}
	}
	if err := ensureIntakeColumn(db, "intake_scans", "father_time_year", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	return nil
}

func effectiveFatherTimePath(cfg config) string {
	if cfg.FatherTimeChroniclesPath != "" {
		return cfg.FatherTimeChroniclesPath
	}
	if cfg.VaultPath == "" {
		return ""
	}
	return filepath.Join(cfg.VaultPath, "00 Pantheon Core", "Chronicles", "Father Time")
}

func fatherTimeLocation(cfg config) (*time.Location, error) {
	if strings.TrimSpace(cfg.Timezone) == "" {
		return time.Local, nil
	}
	return time.LoadLocation(strings.TrimSpace(cfg.Timezone))
}

func newLocalID(prefix string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b[:]), nil
}

func intakeScanLockPath(databasePath string) string {
	return filepath.Join(filepath.Dir(databasePath), "intake.scan.lock")
}

func acquireIntakeScanLock(databasePath string) (func(), error) {
	path := intakeScanLockPath(databasePath)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, errors.New("Intake scan already running")
		}
		return nil, err
	}
	_, _ = fmt.Fprintf(file, "pid=%d\nstarted_at=%s\n", os.Getpid(), time.Now().Format(time.RFC3339Nano))
	_ = file.Close()
	return func() { _ = os.Remove(path) }, nil
}

type hashedIntakeFile struct {
	ContentID string
	Size      int64
	Type      string
	Extension string
}

var intakeHashFile = hashStableIntakeFile

func hashStableIntakeFile(path string) (hashedIntakeFile, error) {
	before, err := os.Stat(path)
	if err != nil {
		return hashedIntakeFile{}, err
	}
	if !before.Mode().IsRegular() {
		return hashedIntakeFile{}, errors.New("object is no longer a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return hashedIntakeFile{}, err
	}
	hasher := sha256.New()
	var sample []byte
	buffer := make([]byte, 128*1024)
	for {
		n, readErr := file.Read(buffer)
		if n > 0 {
			_, _ = hasher.Write(buffer[:n])
			if len(sample) < 512 {
				need := 512 - len(sample)
				if need > n {
					need = n
				}
				sample = append(sample, buffer[:need]...)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = file.Close()
			return hashedIntakeFile{}, readErr
		}
	}
	if err := file.Close(); err != nil {
		return hashedIntakeFile{}, err
	}
	after, err := os.Stat(path)
	if err != nil {
		return hashedIntakeFile{}, err
	}
	if before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || before.Mode() != after.Mode() {
		return hashedIntakeFile{}, fmt.Errorf("file changed during inspection")
	}
	typeName := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if typeName == "" && len(sample) > 0 {
		typeName = http.DetectContentType(sample)
	}
	if typeName == "" {
		typeName = "unknown"
	}
	return hashedIntakeFile{
		ContentID: "sha256:" + hex.EncodeToString(hasher.Sum(nil)),
		Size:      before.Size(), Type: typeName, Extension: strings.ToLower(filepath.Ext(path)),
	}, nil
}

func hiddenOrSystemName(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch strings.ToLower(name) {
	case "thumbs.db", "desktop.ini":
		return true
	}
	return strings.HasPrefix(name, "._")
}

func (d *daemon) runIntakeScan() (intakeScanResult, error) {
	unlock, err := acquireIntakeScanLock(d.databasePath)
	if err != nil {
		return intakeScanResult{}, err
	}
	defer unlock()
	if err := initializeIntakeSchema(d.db); err != nil {
		return intakeScanResult{}, err
	}
	loc, err := fatherTimeLocation(d.cfg)
	if err != nil {
		return intakeScanResult{}, err
	}
	now := time.Now().In(loc)
	scanID, err := newLocalID("scan-")
	if err != nil {
		return intakeScanResult{}, err
	}
	result := intakeScanResult{ScanID: scanID, IntakePath: d.cfg.IntakePath, FatherTimeYear: now.Year(), ChroniclePath: filepath.Join(effectiveFatherTimePath(d.cfg), fmt.Sprintf("%d.md", now.Year())), Chronology: "pending"}
	started := now.Format(time.RFC3339Nano)
	if _, err := d.db.Exec(`INSERT INTO intake_scans (scan_id, started_at, intake_path, father_time_year, chronology) VALUES (?, ?, ?, ?, ?)`, scanID, started, d.cfg.IntakePath, now.Year(), result.Chronology); err != nil {
		return result, err
	}
	complete := true
	seenPaths := make(map[string]bool)
	root := d.cfg.IntakePath
	if _, statErr := os.Stat(root); errors.Is(statErr, os.ErrNotExist) {
		if d.cfg.IntakeConfigured {
			return d.finishIntakeScan(result, started, false, fmt.Errorf("configured Intake path does not exist: %s", root))
		}
		if err := os.MkdirAll(root, 0o700); err != nil {
			return d.finishIntakeScan(result, started, false, fmt.Errorf("create default Intake path: %w", err))
		}
	} else if statErr != nil {
		return d.finishIntakeScan(result, started, false, fmt.Errorf("inspect Intake path: %w", statErr))
	}
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			complete = false
			result.Failed++
			return nil
		}
		if path == root {
			return nil
		}
		if hiddenOrSystemName(entry.Name()) {
			result.Skipped++
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			result.Skipped++
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			result.Failed++
			complete = false
			return nil
		}
		if !info.Mode().IsRegular() {
			result.Skipped++
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			result.Failed++
			return nil
		}
		relative = filepath.ToSlash(relative)
		seenPaths[relative] = true
		hashed, hashErr := intakeHashFile(path)
		if hashErr != nil {
			if strings.Contains(hashErr.Error(), "changed during inspection") {
				result.Deferred++
			} else {
				result.Failed++
			}
			return nil
		}
		isNew, previousDeparted, reconcileErr := d.reconcileIntakeOccurrence(hashed, relative, entry.Name(), now)
		if reconcileErr != nil {
			result.Failed++
			complete = false
			return nil
		}
		if isNew {
			result.New++
		}
		if previousDeparted {
			result.Departed++
		}
		if !isNew {
			result.Unchanged++
		}
		return nil
	})
	if walkErr != nil {
		complete = false
		result.Failed++
	}
	if complete {
		rows, queryErr := d.db.Query(`SELECT occurrence_id, relative_path FROM intake_occurrences WHERE state = ?`, occurrencePresent)
		if queryErr != nil {
			return d.finishIntakeScan(result, started, complete, queryErr)
		}
		var presentOccurrences [][2]string
		for rows.Next() {
			var occurrenceID, relative string
			if err := rows.Scan(&occurrenceID, &relative); err != nil {
				rows.Close()
				return d.finishIntakeScan(result, started, complete, err)
			}
			presentOccurrences = append(presentOccurrences, [2]string{occurrenceID, relative})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return d.finishIntakeScan(result, started, complete, err)
		}
		rows.Close()
		for _, occurrence := range presentOccurrences {
			occurrenceID, relative := occurrence[0], occurrence[1]
			if !seenPaths[relative] {
				if _, err := d.db.Exec(`UPDATE intake_occurrences SET state = ?, last_seen_at = ? WHERE occurrence_id = ?`, occurrenceDeparted, now.Format(time.RFC3339Nano), occurrenceID); err != nil {
					return d.finishIntakeScan(result, started, complete, err)
				}
				result.Departed++
			}
		}
	}
	chronicleErr := d.repairPendingChronicles()
	if chronicleErr != nil {
		result.Chronology = "pending"
	} else {
		result.Chronology = "complete"
	}
	return d.finishIntakeScan(result, started, complete, chronicleErr)
}

func (d *daemon) reconcileIntakeOccurrence(file hashedIntakeFile, relativePath, filename string, now time.Time) (bool, bool, error) {
	var oldID, oldContent, oldState string
	err := d.db.QueryRow(`SELECT occurrence_id, content_id, state FROM intake_occurrences WHERE relative_path = ? ORDER BY first_seen_at DESC LIMIT 1`, relativePath).Scan(&oldID, &oldContent, &oldState)
	if err == nil && oldState == occurrencePresent && oldContent == file.ContentID {
		_, err = d.db.Exec(`UPDATE intake_occurrences SET last_seen_at = ? WHERE occurrence_id = ?`, now.Format(time.RFC3339Nano), oldID)
		return false, false, err
	}
	previousDeparted := false
	if err == nil && oldState == occurrencePresent {
		_, err = d.db.Exec(`UPDATE intake_occurrences SET state = ?, last_seen_at = ? WHERE occurrence_id = ?`, occurrenceDeparted, now.Format(time.RFC3339Nano), oldID)
		if err != nil {
			return false, false, err
		}
		previousDeparted = true
	}
	if !errors.Is(err, sql.ErrNoRows) && err != nil {
		return false, false, err
	}
	if _, err := d.db.Exec(`INSERT OR IGNORE INTO intake_contents (content_id, size, type, extension, first_seen_at) VALUES (?, ?, ?, ?, ?)`, file.ContentID, file.Size, file.Type, file.Extension, now.Format(time.RFC3339Nano)); err != nil {
		return false, false, err
	}
	occurredID, err := newLocalID("intake-")
	if err != nil {
		return false, false, err
	}
	_, err = d.db.Exec(`INSERT INTO intake_occurrences (occurrence_id, content_id, relative_path, original_filename, first_seen_at, last_seen_at, state, father_time_year, processing_state, retention_policy_days, retention_state, chronicle_state) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, occurredID, file.ContentID, relativePath, filename, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), occurrencePresent, now.Year(), "unprocessed", d.cfg.RawInputDays, retentionAwaitingMarkdown, "pending")
	if err != nil {
		return false, false, err
	}
	_, err = d.db.Exec(`INSERT INTO intake_chronicle_events (occurrence_id, event_type, event_at, father_time_year, state) VALUES (?, ?, ?, ?, ?)`, occurredID, "observed", now.Format(time.RFC3339Nano), now.Year(), "pending")
	return true, previousDeparted, err
}

func (d *daemon) repairPendingChronicles() error {
	rows, err := d.db.Query(`SELECT o.occurrence_id, o.content_id, o.relative_path, o.original_filename, o.first_seen_at, o.father_time_year, c.size, c.type, e.event_at FROM intake_occurrences o JOIN intake_chronicle_events e ON e.occurrence_id = o.occurrence_id JOIN intake_contents c ON c.content_id = o.content_id WHERE e.state = 'pending' ORDER BY e.event_at, o.occurrence_id`)
	if err != nil {
		return err
	}
	type pendingEvent struct {
		occurrenceID, contentID, relativePath, filename, firstSeen, typeName, eventAt string
		year                                                                          int
		size                                                                          int64
	}
	var firstErr error
	var pending []pendingEvent
	for rows.Next() {
		var occurrenceID, contentID, relativePath, filename, firstSeen, eventAt, typeName string
		var year int
		var size int64
		if err := rows.Scan(&occurrenceID, &contentID, &relativePath, &filename, &firstSeen, &year, &size, &typeName, &eventAt); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		pending = append(pending, pendingEvent{occurrenceID, contentID, relativePath, filename, firstSeen, typeName, eventAt, year, size})
	}
	if err := rows.Err(); err != nil && firstErr == nil {
		firstErr = err
	}
	rows.Close()
	for _, item := range pending {
		if err := appendFatherTimeEvent(effectiveFatherTimePath(d.cfg), item.year, intakeOccurrence{OccurrenceID: item.occurrenceID, ContentID: item.contentID, RelativePath: item.relativePath, OriginalFilename: item.filename, Size: item.size, Type: item.typeName, FirstSeenAt: item.firstSeen}, item.eventAt); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if _, err := d.db.Exec(`UPDATE intake_chronicle_events SET state = 'written' WHERE occurrence_id = ?`, item.occurrenceID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if _, err := d.db.Exec(`UPDATE intake_occurrences SET chronicle_state = 'written' WHERE occurrence_id = ?`, item.occurrenceID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func ensureNoSymlinkComponents(path string) error {
	clean := filepath.Clean(path)
	volume := filepath.VolumeName(clean)
	rest := strings.TrimPrefix(clean, volume)
	current := volume
	if filepath.IsAbs(clean) {
		current += string(filepath.Separator)
		rest = strings.TrimPrefix(rest, string(filepath.Separator))
	}
	for _, part := range strings.Split(rest, string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("chronicle path contains symbolic link: %s", current)
		}
	}
	return nil
}

func appendFatherTimeEvent(root string, year int, occurrence intakeOccurrence, eventAt string) error {
	if root == "" {
		return errors.New("Father Time chronicle path is unavailable; configure paths.vault or paths.father_time_chronicles")
	}
	if err := ensureNoSymlinkComponents(root); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	path := filepath.Join(root, fmt.Sprintf("%d.md", year))
	if err := ensureNoSymlinkComponents(path); err != nil {
		return err
	}
	marker := "<!-- pantheon-event: " + occurrence.OccurrenceID + " -->"
	if data, err := os.ReadFile(path); err == nil {
		if strings.Contains(string(data), marker) {
			return nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if info, err := file.Stat(); err == nil && info.Size() == 0 {
		if _, err := io.WriteString(file, fmt.Sprintf("# Father Time — %d\n\n", year)); err != nil {
			return err
		}
	}
	when, err := time.Parse(time.RFC3339Nano, eventAt)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(file, "\n## %s\n\n%s\n- %s — Intake occurrence `%s`\n  - Source: `%s`\n  - Content: `%s`\n  - Size: %d bytes\n  - Type: %s\n  - State: observed\n  - Retention: awaiting verified Markdown\n", when.Format("2006-01-02"), marker, when.Format("15:04:05 -07:00"), occurrence.OccurrenceID, occurrence.RelativePath, occurrence.ContentID, occurrence.Size, occurrence.Type)
	if err != nil {
		return err
	}
	return file.Sync()
}

func (d *daemon) finishIntakeScan(result intakeScanResult, started string, complete bool, scanErr error) (intakeScanResult, error) {
	ended := time.Now().Format(time.RFC3339Nano)
	outcome := "completed"
	if scanErr != nil || !complete {
		outcome = "completed_with_errors"
	}
	_, updateErr := d.db.Exec(`UPDATE intake_scans SET ended_at = ?, new_count = ?, unchanged_count = ?, departed_count = ?, deferred_count = ?, skipped_count = ?, failed_count = ?, chronology = ?, outcome = ?, error = ? WHERE scan_id = ?`, ended, result.New, result.Unchanged, result.Departed, result.Deferred, result.Skipped, result.Failed, result.Chronology, outcome, errorText(scanErr), result.ScanID)
	if updateErr != nil && scanErr == nil {
		scanErr = updateErr
	}
	return result, scanErr
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (d *daemon) intakeInspection() (map[string]any, error) {
	return d.intakeInspectionWithQuery(10, "", "", "")
}

func (d *daemon) intakeInspectionWithQuery(limit int, relativePath, extension, typeName string) (map[string]any, error) {
	if err := initializeIntakeSchema(d.db); err != nil {
		return nil, err
	}
	var latest intakeScanResult
	var started, ended, chronology string
	err := d.db.QueryRow(`SELECT scan_id, started_at, COALESCE(ended_at, ''), intake_path, new_count, unchanged_count, departed_count, deferred_count, skipped_count, failed_count, father_time_year, chronology FROM intake_scans ORDER BY started_at DESC LIMIT 1`).Scan(&latest.ScanID, &started, &ended, &latest.IntakePath, &latest.New, &latest.Unchanged, &latest.Departed, &latest.Deferred, &latest.Skipped, &latest.Failed, &latest.FatherTimeYear, &chronology)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	} else if err != nil {
		return nil, err
	}
	latest.Chronology = chronology
	if latest.FatherTimeYear == 0 {
		latest.FatherTimeYear = time.Now().Year()
	}
	if latest.FatherTimeYear != 0 {
		latest.ChroniclePath = filepath.Join(effectiveFatherTimePath(d.cfg), fmt.Sprintf("%d.md", latest.FatherTimeYear))
	}
	var present, pending, failed int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences WHERE state = 'present'`).Scan(&present); err != nil {
		return nil, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences WHERE chronicle_state = 'pending'`).Scan(&pending); err != nil {
		return nil, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_scans WHERE outcome = 'completed_with_errors'`).Scan(&failed); err != nil {
		return nil, err
	}
	recent, err := d.queryIntakeOccurrences(limit, relativePath, extension, typeName)
	if err != nil {
		return nil, err
	}
	return map[string]any{"intake_path": d.cfg.IntakePath, "latest_scan": latest, "latest_scan_started_at": started, "latest_scan_ended_at": ended, "present": present, "pending_chronicle_writes": pending, "failed_scans": failed, "recent_occurrences": recent}, nil
}

func (d *daemon) recentIntakeOccurrences(limit int) ([]intakeOccurrence, error) {
	return d.queryIntakeOccurrences(limit, "", "", "")
}

// queryIntakeOccurrences provides deterministic local retrieval for later
// kernels. Empty filters are ignored; non-empty filters use exact matches.
func (d *daemon) queryIntakeOccurrences(limit int, relativePath, extension, typeName string) ([]intakeOccurrence, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	query := `SELECT o.occurrence_id, o.content_id, o.relative_path, o.original_filename, c.size, c.type, c.extension, o.first_seen_at, o.last_seen_at, o.state, o.father_time_year, o.processing_state, o.retention_policy_days, o.retention_state, COALESCE(o.retention_due_at, ''), o.chronicle_state FROM intake_occurrences o JOIN intake_contents c ON c.content_id = o.content_id`
	var clauses []string
	args := make([]any, 0, 4)
	if relativePath != "" {
		clauses = append(clauses, "o.relative_path = ?")
		args = append(args, relativePath)
	}
	if extension != "" {
		clauses = append(clauses, "c.extension = ?")
		args = append(args, strings.ToLower(extension))
	}
	if typeName != "" {
		clauses = append(clauses, "c.type = ?")
		args = append(args, typeName)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY o.first_seen_at DESC, o.occurrence_id DESC LIMIT ?"
	args = append(args, limit)
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []intakeOccurrence
	for rows.Next() {
		var item intakeOccurrence
		if err := rows.Scan(&item.OccurrenceID, &item.ContentID, &item.RelativePath, &item.OriginalFilename, &item.Size, &item.Type, &item.Extension, &item.FirstSeenAt, &item.LastSeenAt, &item.State, &item.FatherTimeYear, &item.ProcessingState, &item.RetentionDays, &item.RetentionState, &item.RetentionDueAt, &item.ChronicleState); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *daemon) fatherTimeInspection() (map[string]any, error) {
	if err := initializeIntakeSchema(d.db); err != nil {
		return nil, err
	}
	loc, err := fatherTimeLocation(d.cfg)
	if err != nil {
		return nil, err
	}
	now := time.Now().In(loc)
	path := effectiveFatherTimePath(d.cfg)
	rows, err := d.db.Query(`SELECT o.occurrence_id, o.relative_path, e.event_at, e.father_time_year, e.state FROM intake_chronicle_events e JOIN intake_occurrences o ON o.occurrence_id = e.occurrence_id ORDER BY e.event_at DESC LIMIT 10`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type event struct {
		OccurrenceID string `json:"occurrence_id"`
		RelativePath string `json:"relative_path"`
		EventAt      string `json:"event_at"`
		Year         int    `json:"year"`
		State        string `json:"state"`
	}
	var events []event
	for rows.Next() {
		var item event
		if err := rows.Scan(&item.OccurrenceID, &item.RelativePath, &item.EventAt, &item.Year, &item.State); err != nil {
			return nil, err
		}
		events = append(events, item)
	}
	var pending int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_chronicle_events WHERE state = 'pending'`).Scan(&pending); err != nil {
		return nil, err
	}
	return map[string]any{"timezone": loc.String(), "year": now.Year(), "chronicle_path": filepath.Join(path, fmt.Sprintf("%d.md", now.Year())), "pending_chronicle_writes": pending, "recent_events": events}, rows.Err()
}
