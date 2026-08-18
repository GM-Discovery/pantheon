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
	"strings"
	"testing"
	"time"
)

func zodiacTestPackage() seshatPackage {
	return seshatPackage{PluginName: "test", PluginVersion: "1", AdapterAvailable: true, seshatPackageSpec: seshatPackageSpec{
		SubjectCode: "ZODIAC:MAIL_ROUND", Version: "1", Description: "test mail round", Capability: "mail.round", Adapter: "pantheon", RequiredFields: []string{"zodiac_instance_id", "zodiac_schedule_id", "action"}, Returns: []string{"status"}, PermissionClass: "test", PermissionShapes: []string{"once"},
	}}
}

func disableDefaultZodiacSchedules(t *testing.T, d *daemon) {
	t.Helper()
	if _, err := d.db.Exec(`UPDATE zodiac_schedules SET enabled = 0, status = ?`, zodiacScheduleDisabled); err != nil {
		t.Fatal(err)
	}
}

func TestKernel23ZodiacConfigurationAndProviderBoundary(t *testing.T) {
	resolution, err := parseConfigBytes("kernel23", []byte(`{"schema_version":1,"models":{"apollo":{"model":"local","endpoint":"http://127.0.0.1:11434/api/chat"}},"zodiac":{"enabled":true,"default_timezone":"America/Los_Angeles","personal_projection":"apple","work_projection":"google","brief_time":"07:00","brief_catch_up_policy":"run_once","conflict_policy":"needs_you","wake_seconds":30}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !resolution.Config.ZodiacEnabled || resolution.Config.ZodiacDefaultTimezone != "America/Los_Angeles" || resolution.Config.ZodiacPersonalProjection != "apple" || resolution.Config.ZodiacWorkProjection != "google" {
		t.Fatalf("zodiac configuration=%+v", resolution.Config)
	}
	if _, err := parseConfigBytes("kernel23-invalid", []byte(`{"schema_version":1,"models":{"apollo":{"model":"local","endpoint":"http://127.0.0.1:11434/api/chat"}},"zodiac":{"default_timezone":"Not/AZone"}}`)); err == nil || !strings.Contains(err.Error(), "zodiac.default_timezone") {
		t.Fatalf("invalid timezone error=%v", err)
	}
}

func TestKernel23RecurrencePreservesLocalTimeAcrossDST(t *testing.T) {
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.March, 10, 8, 0, 0, 0, location)
	occurrences, err := zodiacOccurrences(zodiacSchedule{StartAt: "2026-03-06T07:00:00-08:00", Timezone: "America/Los_Angeles", Precision: zodiacPrecisionExact, Recurrence: "RRULE:FREQ=DAILY"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(occurrences) != 5 {
		t.Fatalf("occurrences=%d, want 5", len(occurrences))
	}
	for _, occurrence := range occurrences {
		if occurrence.Hour() != 7 || occurrence.Location().String() != "America/Los_Angeles" {
			t.Fatalf("local recurrence=%s", occurrence)
		}
	}
	if _, offset := occurrences[len(occurrences)-1].Zone(); offset != -7*60*60 {
		t.Fatalf("post-DST offset=%d", offset)
	}
}

func TestKernel23ReconcileCatchUpIsBoundedAndIdempotent(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL23RECONCILE", "expire"))
	defer d.Close()
	d.seshat.packages["ZODIAC:MAIL_ROUND"] = zodiacTestPackage()
	d.zodiacClock = func() time.Time {
		return time.Date(2026, time.August, 18, 12, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	}
	disableDefaultZodiacSchedules(t, d)
	schedule, err := d.createZodiacSchedule(zodiacScheduleRequest{ID: "reconcile-mail", Name: "Reconcile mail", Kind: "recurring_system_job", StartAt: "2026-08-18T08:00:00-07:00", Timezone: "America/Los_Angeles", Recurrence: "RRULE:FREQ=HOURLY", CatchUpPolicy: zodiacCatchUpReconcile, Action: zodiacActionMailRound})
	if err != nil {
		t.Fatal(err)
	}
	first, err := d.zodiacRound(nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Proposals != 1 {
		t.Fatalf("first round=%+v", first)
	}
	var instances, proposals int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM zodiac_instances WHERE schedule_id = ?`, schedule.ID).Scan(&instances); err != nil {
		t.Fatal(err)
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM future_proposals WHERE scope LIKE ?`, "zodiac:instance:%").Scan(&proposals); err != nil {
		t.Fatal(err)
	}
	if instances != 1 || proposals != 1 {
		t.Fatalf("bounded catch-up instances=%d proposals=%d", instances, proposals)
	}
	second, err := d.zodiacRound(nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.Proposals != 0 {
		t.Fatalf("idempotent second round=%+v", second)
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM zodiac_instances WHERE schedule_id = ?`, schedule.ID).Scan(&instances); err != nil {
		t.Fatal(err)
	}
	if instances != 1 {
		t.Fatalf("second round created duplicate instances=%d", instances)
	}
}

func TestKernel23PrometheusPauseBlocksAndResumeDoesNotReplaySameInstance(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL23PAUSE", "expire"))
	defer d.Close()
	d.seshat.packages["ZODIAC:MAIL_ROUND"] = zodiacTestPackage()
	d.zodiacClock = func() time.Time {
		return time.Date(2026, time.August, 18, 12, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	}
	disableDefaultZodiacSchedules(t, d)
	schedule, err := d.createZodiacSchedule(zodiacScheduleRequest{ID: "paused-mail", Name: "Paused mail", Kind: "recurring_system_job", StartAt: "2026-08-18T11:00:00-07:00", Timezone: "America/Los_Angeles", Recurrence: "RRULE:FREQ=HOURLY", CatchUpPolicy: zodiacCatchUpReconcile, Action: zodiacActionMailRound})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.setPaused(true); err != nil {
		t.Fatal(err)
	}
	if _, err := d.zodiacRound(nil); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := d.db.QueryRow(`SELECT state FROM zodiac_instances WHERE schedule_id = ?`, schedule.ID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != zodiacInstanceBlockedPrometheus {
		t.Fatalf("paused state=%q", state)
	}
	var proposals int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM future_proposals WHERE scope LIKE ?`, "zodiac:instance:%").Scan(&proposals); err != nil {
		t.Fatal(err)
	}
	if proposals != 0 {
		t.Fatalf("paused Zodiac created proposals=%d", proposals)
	}
	if err := d.setPaused(false); err != nil {
		t.Fatal(err)
	}
	if _, err := d.zodiacRound(nil); err != nil {
		t.Fatal(err)
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM zodiac_instances WHERE schedule_id = ?`, schedule.ID).Scan(&proposals); err != nil {
		t.Fatal(err)
	}
	if proposals != 1 {
		t.Fatalf("resume replayed blocked instance count=%d", proposals)
	}
}

func TestKernel23AttestationVerifiesSourceFingerprintAndQuotedSpan(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL23ATTEST", "expire"))
	defer d.Close()
	sourcePath := t.TempDir() + "/transcript.md"
	contents := []byte("[00:42:17] Grant: Put a test appointment on my calendar Thursday at two.")
	if err := os.WriteFile(sourcePath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(contents)
	normalized := json.RawMessage(`{"action":"calendar.create","start":"2026-08-20T14:00:00-07:00"}`)
	attestation, err := d.recordZodiacAttestation(zodiacAttestation{SourceID: "transcript-1", SourcePath: sourcePath, SourceFingerprint: hex.EncodeToString(digest[:]), CitationStart: "00:42:17", CitationEnd: "00:42:29", QuotedSpan: "Put a test appointment on my calendar Thursday at two.", Normalized: normalized, Confidence: .97})
	if err != nil {
		t.Fatal(err)
	}
	if attestation.Result != "verified" {
		t.Fatalf("attestation=%+v", attestation)
	}
	_, err = d.recordZodiacAttestation(zodiacAttestation{SourceID: "transcript-2", SourcePath: sourcePath, SourceFingerprint: "wrong", CitationStart: "1", CitationEnd: "2", QuotedSpan: "Put a test appointment", Normalized: normalized, Confidence: .9})
	if err == nil || !strings.Contains(err.Error(), "fingerprint") {
		t.Fatalf("wrong fingerprint error=%v", err)
	}
}

func TestKernel23PanopticonShowsZodiacAndNeedsYouWithoutCalendarAdapter(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL23GUI", "expire"))
	defer d.Close()
	page := d.Handler()
	req := httptest.NewRequest("GET", "/v1/zodiac", nil)
	rec := httptest.NewRecorder()
	page.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("zodiac status=%d body=%s", rec.Code, rec.Body.String())
	}
	var inspection zodiacInspection
	if err := json.NewDecoder(rec.Body).Decode(&inspection); err != nil {
		t.Fatal(err)
	}
	if len(inspection.Recurring) < 2 {
		t.Fatalf("recurring=%d", len(inspection.Recurring))
	}
	if len(inspection.Providers) != 2 || inspection.Providers[0].State != "unavailable" && inspection.Providers[1].State != "unavailable" {
		t.Fatalf("providers=%+v", inspection.Providers)
	}
	if _, err := d.loadZodiacSchedule("missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing schedule error=%v", err)
	}
}

func TestKernel23HumanCalendarCorrectionIsPreservedAndConflictSurfaces(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL23CORRECTION", "expire"))
	defer d.Close()
	schedule, err := d.createZodiacSchedule(zodiacScheduleRequest{ID: "cross-domain", Name: "Cross-domain test", Kind: "one_time_event", StartAt: "2026-08-20T14:00:00-07:00", Timezone: "America/Los_Angeles", ProjectionPolicy: zodiacProjectionBoth, Action: zodiacActionCalendarReminder})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := d.db.Query(`SELECT projection_id FROM zodiac_projections WHERE schedule_id = ? ORDER BY provider`, schedule.ID)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) != 2 {
		t.Fatalf("projection count=%d", len(ids))
	}
	corrected, err := d.reconcileZodiacProjection(ids[0], zodiacProjectionReconcileRequest{ExternalEventID: "external-1", ExternalRevision: "rev-1", StartAt: "2026-08-20T15:00:00-07:00", HumanModified: true})
	if err != nil {
		t.Fatal(err)
	}
	if corrected.Status != "human_corrected" || !strings.Contains(corrected.Correction, "Human calendar edit") && !strings.Contains(corrected.Correction, "external_revision") {
		t.Fatalf("corrected projection=%+v", corrected)
	}
	other, err := d.loadZodiacProjection(ids[1])
	if err != nil {
		t.Fatal(err)
	}
	if other.Status != "needs_you" {
		t.Fatalf("conflicting sibling status=%q", other.Status)
	}
	deleted, err := d.reconcileZodiacProjection(ids[0], zodiacProjectionReconcileRequest{ExternalEventID: "external-1", ExternalRevision: "rev-2", Deleted: true})
	if err != nil {
		t.Fatal(err)
	}
	if deleted.Status != "human_deleted" || !strings.Contains(deleted.Correction, "will not recreate") {
		t.Fatalf("deleted projection=%+v", deleted)
	}
}
