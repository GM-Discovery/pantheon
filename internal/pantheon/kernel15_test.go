package pantheon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func kernel15Daemon(t *testing.T, packages ...seshatPackageSpec) *daemon {
	t.Helper()
	dir := t.TempDir()
	registryDir := filepath.Join(dir, "plugins", "registered")
	if err := os.MkdirAll(registryDir, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := seshatManifest{SchemaVersion: 1, Plugin: seshatPlugin{Name: "janus-test", Version: "1.0.0"}, Packages: packages}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(registryDir, "janus-test.json"), manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "pantheon.json")
	capsPath := filepath.Join(dir, "capabilities.json")
	roadmap := filepath.Join(dir, "roadmap.md")
	if err := os.WriteFile(roadmap, []byte("roadmap"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"local_model":"apollo-test","ollama_url":"http://127.0.0.1:1/api/chat","roadmap_path":"`+roadmap+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capsPath, []byte(`{"capabilities":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := newDaemon(daemonOptions{ConfigPath: configPath, CapabilitiesPath: capsPath, DatabasePath: filepath.Join(dir, "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	d.cfg.IntakePath = filepath.Join(dir, "intake")
	d.cfg.IntakeConfigured = true
	d.cfg.FatherTimeChroniclesPath = filepath.Join(dir, "chronicles")
	if err := os.MkdirAll(d.cfg.IntakePath, 0o700); err != nil {
		t.Fatal(err)
	}
	return d
}

func janusTestPackage(code, policy string) seshatPackageSpec {
	return seshatPackageSpec{
		SubjectCode: code, Version: "1.0.0", Description: "Janus test package", Capability: "test.temporal", Adapter: "noop",
		RequiredFields: []string{"body"}, PayloadRules: map[string]string{"body": "body"}, Returns: []string{"status"},
		PermissionClass: "bounded_test", PermissionShapes: []string{"once"},
		Temporal: &seshatTemporal{Mode: "scheduled", MissedPolicy: policy},
	}
}

func janusPast() string { return time.Now().Add(-time.Minute).Format(time.RFC3339Nano) }

func janusFuture() string { return time.Now().Add(time.Hour).Format(time.RFC3339Nano) }

func createPastJanusSubject(t *testing.T, d *daemon, code, kind string) janusSubject {
	t.Helper()
	subject, err := d.createJanusSubject(janusSubjectRequest{SubjectID: "subject-" + strings.ToLower(strings.ReplaceAll(code, ":", "-")), PackageCode: code, Threshold: janusPast(), ThresholdKind: kind, RequestText: "bounded Janus test"})
	if err != nil {
		t.Fatal(err)
	}
	return subject
}

func TestKernel15SeshatTemporalSchemaAndSafeCreation(t *testing.T) {
	valid := janusTestPackage("TEST:TEMPORAL", "expire")
	data, err := json.Marshal(seshatManifest{SchemaVersion: 1, Plugin: seshatPlugin{Name: "temporal", Version: "1"}, Packages: []seshatPackageSpec{valid}})
	if err != nil {
		t.Fatal(err)
	}
	var decoded seshatManifest
	if err := json.Unmarshal(data, &decoded); err != nil || decoded.Packages[0].Temporal == nil {
		t.Fatalf("temporal manifest did not round-trip: %v", err)
	}
	invalid := valid
	invalid.Temporal = &seshatTemporal{Mode: "scheduled", MissedPolicy: "guess"}
	invalidData, _ := json.Marshal(seshatManifest{SchemaVersion: 1, Plugin: seshatPlugin{Name: "invalid", Version: "1"}, Packages: []seshatPackageSpec{invalid}})
	if _, err := validateSeshatManifestBytes(invalidData); err == nil || !strings.Contains(err.Error(), "missed_policy") {
		t.Fatalf("invalid temporal policy error = %v", err)
	}
	d := kernel15Daemon(t, valid)
	defer d.Close()
	if _, err := d.createJanusSubject(janusSubjectRequest{PackageCode: "TEST:TEMPORAL", Threshold: janusPast(), ThresholdKind: "bad"}); err == nil {
		t.Fatal("invalid threshold kind was accepted")
	}
	if _, err := d.createJanusSubject(janusSubjectRequest{PackageCode: "TEST:UNKNOWN", Threshold: janusPast(), ThresholdKind: janusThresholdExpiresAt}); err == nil {
		t.Fatal("unknown package was accepted")
	}
}

func TestKernel15ExpireTransitionIsIdempotentAndDoesNotIssueGo(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:EXPIRE", "expire"))
	defer d.Close()
	subject := createPastJanusSubject(t, d, "TEST:EXPIRE", janusThresholdExpiresAt)
	first, err := d.janusRound(nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Expired != 1 || first.Failed != 0 {
		t.Fatalf("first Janus round = %#v", first)
	}
	loaded, err := d.loadJanusSubject(subject.SubjectID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State != janusStateExpired {
		t.Fatalf("subject state = %q", loaded.State)
	}
	second, err := d.janusRound(nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.Examined != 0 || second.Transitioned != 0 {
		t.Fatalf("repeated Janus round was not idempotent: %#v", second)
	}
	var transitions, gos int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM janus_transitions WHERE subject_id = ?`, subject.SubjectID).Scan(&transitions); err != nil {
		t.Fatal(err)
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM order_go_commands`).Scan(&gos); err != nil {
		t.Fatal(err)
	}
	if transitions != 2 || gos != 0 {
		t.Fatalf("transitions=%d gos=%d; Janus must not issue Go", transitions, gos)
	}
}

func TestKernel15ResurfaceCreatesOneNewIntakeOccurrenceWithoutRevivingSource(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:RESURFACE", "resurface"))
	defer d.Close()
	subject := createPastJanusSubject(t, d, "TEST:RESURFACE", janusThresholdExpiresAt)
	first, err := d.janusRound(nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Resurfaced != 1 || first.Failed != 0 {
		t.Fatalf("resurface round = %#v", first)
	}
	loaded, err := d.loadJanusSubject(subject.SubjectID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State != janusStateResurfaced || loaded.ResurfacedOccurrenceID == "" {
		t.Fatalf("resurfaced subject = %#v", loaded)
	}
	second, err := d.janusRound(nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.Examined != 0 {
		t.Fatalf("resurfaced subject was replayed: %#v", second)
	}
	var occurrences int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences WHERE occurrence_id = ?`, loaded.ResurfacedOccurrenceID).Scan(&occurrences); err != nil {
		t.Fatal(err)
	}
	if occurrences != 1 {
		t.Fatalf("resurfaced occurrence count = %d", occurrences)
	}
	var sourceState string
	if err := d.db.QueryRow(`SELECT state FROM janus_subjects WHERE subject_id = ?`, subject.SubjectID).Scan(&sourceState); err != nil {
		t.Fatal(err)
	}
	if sourceState == janusStateReady || sourceState == janusStateScheduled {
		t.Fatalf("source subject was revived: %q", sourceState)
	}
}

func TestKernel15PauseRequireReviewAndResumeDoesNotReplay(t *testing.T) {
	pkg := janusTestPackage("TEST:REVIEW", "require_review")
	pkg.Temporal.PausePolicy = "require_review"
	d := kernel15Daemon(t, pkg)
	defer d.Close()
	subject := createPastJanusSubject(t, d, "TEST:REVIEW", janusThresholdNotBefore)
	if err := d.setPaused(true); err != nil {
		t.Fatal(err)
	}
	pausedRound, err := d.janusRound(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !pausedRound.Paused || pausedRound.ReviewRequired != 1 {
		t.Fatalf("paused review round = %#v", pausedRound)
	}
	if err := d.setPaused(false); err != nil {
		t.Fatal(err)
	}
	resumeRound, err := d.janusRound(nil)
	if err != nil {
		t.Fatal(err)
	}
	if resumeRound.Examined != 0 {
		t.Fatalf("resume replayed review-required work: %#v", resumeRound)
	}
	loaded, err := d.loadJanusSubject(subject.SubjectID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State != janusStateReviewRequired {
		t.Fatalf("review state = %q", loaded.State)
	}
	var gos int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM order_go_commands`).Scan(&gos); err != nil {
		t.Fatal(err)
	}
	if gos != 0 {
		t.Fatalf("Janus produced Go commands = %d", gos)
	}
}

func TestKernel15ReadyBackfillAndRecomputeAreBoundedStates(t *testing.T) {
	backfill := janusTestPackage("TEST:BACKFILL", "backfill")
	recompute := janusTestPackage("TEST:RECOMPUTE", "recompute")
	ready := janusTestPackage("TEST:READY", "expire")
	d := kernel15Daemon(t, backfill, recompute, ready)
	defer d.Close()
	backfillSubject := createPastJanusSubject(t, d, "TEST:BACKFILL", janusThresholdExpiresAt)
	recomputeSubject := createPastJanusSubject(t, d, "TEST:RECOMPUTE", janusThresholdExpiresAt)
	readySubject, err := d.createJanusSubject(janusSubjectRequest{SubjectID: "subject-ready", PackageCode: "TEST:READY", Threshold: janusFuture(), ThresholdKind: janusThresholdNotBefore})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := d.janusRound(nil); err != nil || result.Transitioned != 4 {
		t.Fatalf("initial bounded round = %#v, err=%v", result, err)
	}
	if _, err := d.db.Exec(`UPDATE janus_subjects SET threshold = ? WHERE subject_id = ?`, janusPast(), readySubject.SubjectID); err != nil {
		t.Fatal(err)
	}
	if result, err := d.janusRound(nil); err != nil || result.Transitioned != 1 {
		t.Fatalf("ready round = %#v, err=%v", result, err)
	}
	for id, expected := range map[string]string{backfillSubject.SubjectID: janusStateBackfillPending, recomputeSubject.SubjectID: janusStateRecomputePending, readySubject.SubjectID: janusStateReady} {
		var state string
		if err := d.db.QueryRow(`SELECT state FROM janus_subjects WHERE subject_id = ?`, id).Scan(&state); err != nil {
			t.Fatal(err)
		}
		if state != expected {
			t.Fatalf("subject %s state=%q want=%q", id, state, expected)
		}
	}
}

func TestKernel15InspectionAndControlBoundary(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:INSPECT", "expire"))
	defer d.Close()
	request := newJSONRequest(t, http.MethodPost, "/v1/janus/subjects", janusSubjectRequest{PackageCode: "TEST:INSPECT", Threshold: janusFuture(), ThresholdKind: janusThresholdNotBefore})
	unauthorized := httptest.NewRecorder()
	d.Handler().ServeHTTP(unauthorized, request)
	if unauthorized.Code != http.StatusForbidden {
		t.Fatalf("unauthorized subject creation status=%d", unauthorized.Code)
	}
	request = newJSONRequest(t, http.MethodPost, "/v1/janus/subjects", janusSubjectRequest{PackageCode: "TEST:INSPECT", Threshold: janusFuture(), ThresholdKind: janusThresholdNotBefore})
	request.Header.Set(controlTokenHeader, d.controlToken)
	created := httptest.NewRecorder()
	d.Handler().ServeHTTP(created, request)
	if created.Code != http.StatusCreated {
		t.Fatalf("subject creation status=%d body=%s", created.Code, created.Body.String())
	}
	for _, path := range []string{"/v1/janus", "/v1/janus/transitions"} {
		rr := httptest.NewRecorder()
		d.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("inspection %s status=%d body=%s", path, rr.Code, rr.Body.String())
		}
	}
}

func TestKernel15MissingTemporalSubjectFailsVisibly(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:MISSING", "expire"))
	defer d.Close()
	if _, err := d.db.Exec(`INSERT INTO janus_subjects (subject_id, package_code, state, threshold, threshold_kind, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, "subject-missing", "TEST:UNKNOWN", janusStateScheduled, janusPast(), janusThresholdExpiresAt, janusPast(), janusPast()); err != nil {
		t.Fatal(err)
	}
	result, err := d.janusRound(nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed != 1 || len(result.Errors) != 1 {
		t.Fatalf("missing subject result = %#v", result)
	}
	var state, lastError string
	if err := d.db.QueryRow(`SELECT state, last_error FROM janus_subjects WHERE subject_id = ?`, "subject-missing").Scan(&state, &lastError); err != nil {
		t.Fatal(err)
	}
	if state != janusStateBlocked || lastError == "" {
		t.Fatalf("missing subject state=%q error=%q", state, lastError)
	}
}
