package pantheon

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testKernel9Daemon(t *testing.T, cfg config) *daemon {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "pantheon.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := initializeIntakeSchema(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS audit_events (id TEXT PRIMARY KEY, occurred_at TEXT NOT NULL, endpoint TEXT NOT NULL, input TEXT, capability TEXT, model TEXT, outcome TEXT NOT NULL, result TEXT, error TEXT, duration_ms INTEGER NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	// Deliberately created WITHOUT request_id above, then migrated -- the
	// same additive step the daemon runs on an existing database, so every
	// test exercises the real upgrade path rather than a table that was
	// born with the column.
	if err := ensureIntakeColumn(db, "audit_events", "request_id", "TEXT"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS runtime_state (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at TEXT NOT NULL); INSERT OR REPLACE INTO runtime_state (key, value, updated_at) VALUES ('paused', 'false', 'now')`); err != nil {
		t.Fatal(err)
	}
	return &daemon{cfg: cfg, db: db, databasePath: filepath.Join(dir, "pantheon.db")}
}

func testKernel9Config(t *testing.T, root string) config {
	t.Helper()
	return config{
		IntakePath: filepath.Join(root, "Intake"), IntakeConfigured: true,
		VaultPath: filepath.Join(root, "Vault"), RawInputDays: 30,
		Timezone: "America/Los_Angeles",
	}
}

func TestKernel9DefaultIntakePathUsesRuntimeHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"schema_version":1,"models":{"apollo":{"model":"local","endpoint":"http://127.0.0.1:11434/api/chat"}}}`)
	resolution, err := parseConfigBytes("test", data)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "Pantheon Intake")
	if resolution.Config.IntakePath != want {
		t.Fatalf("Intake path = %q, want %q", resolution.Config.IntakePath, want)
	}
	if strings.Contains(resolution.Config.IntakePath, "/home/brick") && home != "/home/brick" {
		t.Fatal("development home leaked into default path")
	}
}

func TestKernel9ConfigTimezoneAndRetentionValidation(t *testing.T) {
	base := `{"schema_version":1,"retention":{"raw_input_days":30},"time":{"timezone":"Mars/Olympus"},"models":{"apollo":{"model":"local","endpoint":"http://127.0.0.1:11434/api/chat"}}}`
	if _, err := parseConfigBytes("test", []byte(base)); err == nil || !strings.Contains(err.Error(), "time.timezone") {
		t.Fatalf("invalid timezone error = %v", err)
	}
	badRetention := strings.Replace(base, "Mars/Olympus", "UTC", 1)
	badRetention = strings.Replace(badRetention, `"raw_input_days":30`, `"raw_input_days":0`, 1)
	if _, err := parseConfigBytes("test", []byte(badRetention)); err == nil || !strings.Contains(err.Error(), "raw_input_days") {
		t.Fatalf("invalid retention error = %v", err)
	}
}

func TestKernel9IntakeTraversalIdentityAndIdempotency(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	d := testKernel9Daemon(t, cfg)
	if err := os.MkdirAll(filepath.Join(cfg.IntakePath, "MFPI"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.IntakePath, "meeting.txt"), []byte("same bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.IntakePath, "copy.txt"), []byte("same bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.IntakePath, "MFPI", "notes.pdf"), []byte("different bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.IntakePath, ".hidden"), []byte("skip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(cfg.IntakePath, ".hidden-dir"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.IntakePath, ".hidden-dir", "secret"), []byte("skip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(cfg.IntakePath, "meeting.txt"), filepath.Join(cfg.IntakePath, "link.txt")); err != nil {
		t.Fatal(err)
	}

	first, err := d.runIntakeScan()
	if err != nil {
		t.Fatal(err)
	}
	if first.New != 3 || first.Unchanged != 0 {
		t.Fatalf("first scan = %+v", first)
	}
	if first.Skipped < 3 {
		t.Fatalf("skipped = %d, want hidden file, hidden directory, and symlink", first.Skipped)
	}
	second, err := d.runIntakeScan()
	if err != nil {
		t.Fatal(err)
	}
	if second.New != 0 || second.Unchanged != 3 {
		t.Fatalf("second scan = %+v", second)
	}

	var occurrences, contents int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences`).Scan(&occurrences); err != nil {
		t.Fatal(err)
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_contents`).Scan(&contents); err != nil {
		t.Fatal(err)
	}
	if occurrences != 3 || contents != 2 {
		t.Fatalf("occurrences=%d contents=%d", occurrences, contents)
	}
	var days int
	if err := d.db.QueryRow(`SELECT retention_policy_days FROM intake_occurrences LIMIT 1`).Scan(&days); err != nil || days != 30 {
		t.Fatalf("retention days=%d err=%v", days, err)
	}
	var state string
	var eligible, due sql.NullString
	if err := d.db.QueryRow(`SELECT retention_state, retention_eligible_at, retention_due_at FROM intake_occurrences LIMIT 1`).Scan(&state, &eligible, &due); err != nil || state != retentionAwaitingMarkdown {
		t.Fatalf("retention state=%q err=%v", state, err)
	}
	if eligible.Valid || due.Valid {
		t.Fatalf("retention clock armed: eligible=%v due=%v", eligible, due)
	}

	data, err := os.ReadFile(filepath.Join(cfg.VaultPath, "00 Pantheon Core", "Chronicles", "Father Time", time.Now().In(time.FixedZone("PDT", -7*60*60)).Format("2006")+".md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), "<!-- pantheon-event:"); got != 3 {
		t.Fatalf("chronicle event markers=%d", got)
	}
}

func TestKernel9ChangedAndDepartedReappearedBecomeNewOccurrences(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	d := testKernel9Daemon(t, cfg)
	path := filepath.Join(cfg.IntakePath, "recording.txt")
	if err := os.MkdirAll(cfg.IntakePath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := d.runIntakeScan()
	if err != nil {
		t.Fatal(err)
	}
	if changed.New != 1 || changed.Departed != 1 {
		t.Fatalf("changed scan=%+v", changed)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	departed, err := d.runIntakeScan()
	if err != nil {
		t.Fatal(err)
	}
	if departed.Departed != 1 {
		t.Fatalf("departure scan=%+v", departed)
	}
	if err := os.WriteFile(path, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	reappeared, err := d.runIntakeScan()
	if err != nil {
		t.Fatal(err)
	}
	if reappeared.New != 1 {
		t.Fatalf("reappearance scan=%+v", reappeared)
	}
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences WHERE relative_path = 'recording.txt'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("occurrence count=%d, want 3", count)
	}
	var distinct int
	if err := d.db.QueryRow(`SELECT COUNT(DISTINCT content_id) FROM intake_occurrences WHERE relative_path = 'recording.txt'`).Scan(&distinct); err != nil {
		t.Fatal(err)
	}
	if distinct != 2 {
		t.Fatalf("content identity count=%d, want 2", distinct)
	}
}

func TestKernel9ChangingFileDeferred(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	d := testKernel9Daemon(t, cfg)
	if err := os.MkdirAll(cfg.IntakePath, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg.IntakePath, "unstable.bin")
	if err := os.WriteFile(path, []byte("bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	original := intakeHashFile
	intakeHashFile = func(string) (hashedIntakeFile, error) {
		return hashedIntakeFile{}, errors.New("file changed during inspection")
	}
	t.Cleanup(func() { intakeHashFile = original })
	result, err := d.runIntakeScan()
	if err != nil {
		t.Fatal(err)
	}
	if result.Deferred != 1 || result.New != 0 {
		t.Fatalf("unstable scan=%+v", result)
	}
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unstable file finalized as occurrence: %d", count)
	}
}

func TestKernel9ChronicleFailureRepairsWithoutDuplicate(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	badChronicle := filepath.Join(root, "chronicle-target")
	if err := os.WriteFile(badChronicle, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.FatherTimeChroniclesPath = badChronicle
	d := testKernel9Daemon(t, cfg)
	if err := os.MkdirAll(cfg.IntakePath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.IntakePath, "note.txt"), []byte("note"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err == nil {
		t.Fatal("expected chronicle failure")
	}
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("ledger count=%d err=%v", count, err)
	}
	if err := os.Remove(badChronicle); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("ledger count after repair=%d err=%v", count, err)
	}
	var state string
	if err := d.db.QueryRow(`SELECT state FROM intake_chronicle_events`).Scan(&state); err != nil || state != "written" {
		t.Fatalf("chronicle state=%q err=%v", state, err)
	}
}

func TestKernel9PausedScanRefusedInspectionAvailable(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	d := testKernel9Daemon(t, cfg)
	if err := d.setPaused(true); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/intake/scan", nil)
	rec := httptest.NewRecorder()
	d.handleIntakeScan(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("paused scan status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := d.intakeInspection(); err != nil {
		t.Fatal(err)
	}
	if _, err := d.fatherTimeInspection(); err != nil {
		t.Fatal(err)
	}
	if err := d.setPaused(false); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_scans`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("paused scan was replayed/started: %d", count)
	}
}

func TestKernel9LargeFileHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.bin")
	data := make([]byte, 2*1024*1024)
	for i := range data {
		data[i] = byte(i % 251)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	hashed, err := hashStableIntakeFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if hashed.ContentID == "" || hashed.Size != int64(len(data)) {
		t.Fatalf("hash result=%+v", hashed)
	}
	expected := sha256.Sum256(data)
	if hashed.ContentID != "sha256:"+hex.EncodeToString(expected[:]) {
		t.Fatalf("content id=%q", hashed.ContentID)
	}
}

func TestKernel9InspectionJSONRemainsMachineReadable(t *testing.T) {
	root := t.TempDir()
	d := testKernel9Daemon(t, testKernel9Config(t, root))
	inspection, err := d.intakeInspection()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(inspection); err != nil {
		t.Fatal(err)
	}
}

func TestKernel9OccurrenceRetrievalSupportsDeterministicMetadataFilters(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	d := testKernel9Daemon(t, cfg)
	if err := os.MkdirAll(filepath.Join(cfg.IntakePath, "meetings"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.IntakePath, "meetings", "recording.m4a"), []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.IntakePath, "notes.txt"), []byte("notes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	filtered, err := d.queryIntakeOccurrences(10, "meetings/recording.m4a", ".m4a", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].RelativePath != "meetings/recording.m4a" || filtered[0].Extension != ".m4a" {
		t.Fatalf("filtered occurrences = %+v", filtered)
	}
}
