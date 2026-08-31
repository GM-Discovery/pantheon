package pantheon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func kernel28GUISessionCookie(t *testing.T, d *daemon) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/gui/session", nil)
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GUI session status=%d body=%s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != guiSessionCookie {
		t.Fatalf("GUI session cookies=%v", cookies)
	}
	return cookies[0]
}

func TestKernel28PanopticonCanEvaluateZodiacWithGUISession(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL28GUI", "expire"))
	defer d.Close()
	cookie := kernel28GUISessionCookie(t, d)
	req := httptest.NewRequest(http.MethodPost, "/v1/gui/zodiac/round", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GUI Zodiac round status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "CLI control boundary") {
		t.Fatalf("GUI Zodiac round still exposed CLI-only boundary: %s", rec.Body.String())
	}
}

func TestKernel28LibraReportDefinitionCreatesTypedZodiacSchedule(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL28LIBRA", "expire"))
	defer d.Close()
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	d.zodiacClock = func() time.Time { return time.Date(2026, time.August, 23, 6, 0, 0, 0, location) }
	cookie := kernel28GUISessionCookie(t, d)
	req := httptest.NewRequest(http.MethodPost, "/v1/gui/zodiac/report-definitions", strings.NewReader(`{"constellation_id":"libra","name":"Daily evidence recap","cadence":"daily","local_time":"07:00","timezone":"America/Los_Angeles","window_policy":"previous_day","destination":"situation_briefs","notify_on_success":true,"catch_up_policy":"run_once"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Libra report status=%d body=%s", rec.Code, rec.Body.String())
	}
	var report libraReportDefinition
	if err := json.NewDecoder(rec.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if report.ConstellationID != "libra" || report.Cadence != "daily" || report.Destination != "situation_briefs" || !report.Enabled {
		t.Fatalf("report=%+v", report)
	}
	schedule, err := d.loadZodiacSchedule(report.ScheduleID)
	if err != nil {
		t.Fatal(err)
	}
	if schedule.Source != "panopticon" || schedule.SourceID != report.ID || schedule.Action != zodiacActionBriefGenerate || schedule.Recurrence != "RRULE:FREQ=DAILY;INTERVAL=1" {
		t.Fatalf("typed schedule=%+v", schedule)
	}
	var payload map[string]any
	if err := json.Unmarshal(schedule.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["report_definition_id"] != report.ID || payload["window_policy"] != "previous_day" {
		t.Fatalf("schedule payload=%v", payload)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/gui/zodiac/report-definitions/"+report.ID+"/disable", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable report status=%d body=%s", rec.Code, rec.Body.String())
	}
	var enabled int
	if err := d.db.QueryRow(`SELECT enabled FROM zodiac_schedules WHERE id = ?`, report.ScheduleID).Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 0 {
		t.Fatalf("linked Zodiac schedule remained enabled=%d", enabled)
	}
}

func TestKernel28LibraRecipeTriggersOnIngestionAndWritesMultipleOutputs(t *testing.T) {
	t.Setenv(externalIntakeSecretEnv, "kernel28-recipe-secret")
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL28RECIPE", "expire"))
	defer d.Close()
	d.cfg.VaultPath = filepath.Join(t.TempDir(), "vault")
	d.cfg.RawPath = filepath.Join(d.cfg.VaultPath, "80 Pantheon Raw")
	d.cfg.LocalModel = "apollo-recipe-test"
	d.cfg.ApolloRequireCitations = false
	d.cfg.ApolloMaxReasoningDurationSeconds = 5
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"content":"{\"answer\":\"The uploaded event was converted and recorded.\",\"status\":\"complete\",\"summary\":\"Event evidence reviewed.\"}"}}`))
	}))
	defer model.Close()
	d.cfg.OllamaURL = model.URL

	recipe, err := d.createLibraReportDefinition(libraReportDefinitionRequest{
		ConstellationID: "libra", Name: "Event packet", Trigger: libraTriggerOnIngestion,
		Timezone: "America/Los_Angeles", WindowPolicy: "previous_day", Destination: libraOutputIntegrated,
		SourceFamilies: []string{"transcripts"}, Sections: []string{"summary", "requests"},
		Instructions: "Name explicit requests plainly.", Outputs: []string{libraOutputIntegrated, libraOutputSituation},
	})
	if err != nil {
		t.Fatal(err)
	}
	if recipe.Cadence != libraCadenceEvent || recipe.ScheduleID != "" || recipe.Recipe.Trigger != libraTriggerOnIngestion || len(recipe.Recipe.Outputs) != 2 {
		t.Fatalf("recipe=%+v", recipe)
	}

	envelope := externalIntakeEnvelope{
		SourceKind: "omi_memory", SourceID: "recipe-memory-001", SourceEventID: "recipe-event-001",
		OccurredAt: "2026-08-23T10:15:00Z", Payload: json.RawMessage(`{"title":"Recipe test","labels":["personal"]}`),
		Transcript: "Grant: Please record this event.\nTrinh: Agreed.", Labels: []string{"personal"},
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/intake/external", strings.NewReader(string(body)))
	req.Header.Set("X-Pantheon-Intake-Secret", "kernel28-recipe-secret")
	rec := httptest.NewRecorder()
	d.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("intake status=%d body=%s", rec.Code, rec.Body.String())
	}
	var receipt externalIntakeReceipt
	if err := json.Unmarshal(rec.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	var queued int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM libra_ingestion_jobs WHERE report_definition_id = ? AND state = 'queued'`, recipe.ID).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("queued ingestion jobs=%d", queued)
	}
	if err := d.processLibraIngestionQueue(context.Background(), 4); err != nil {
		t.Fatal(err)
	}
	var runID string
	if err := d.db.QueryRow(`SELECT run_id FROM libra_report_runs WHERE report_definition_id = ?`, recipe.ID).Scan(&runID); err != nil {
		t.Fatal(err)
	}
	run, err := d.loadLibraReportRunByID(runID)
	if err != nil {
		t.Fatal(err)
	}
	if run.State != "completed" || len(run.Outputs) != 2 {
		t.Fatalf("run=%+v", run)
	}
	for _, output := range run.Outputs {
		if output.State != "completed" || output.Path == "" {
			t.Fatalf("output=%+v", output)
		}
		if _, err := os.Stat(output.Path); err != nil {
			t.Fatalf("output missing=%+v err=%v", output, err)
		}
	}
	var state string
	if err := d.db.QueryRow(`SELECT state FROM libra_ingestion_jobs WHERE report_definition_id = ? AND source_id = ?`, recipe.ID, receipt.SourceID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "completed" {
		t.Fatalf("ingestion job state=%s", state)
	}
}

func TestKernel28LibraDailyAndWeeklyReportsCompleteEndToEnd(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL28E2E", "expire"))
	defer d.Close()
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	vault := filepath.Join(t.TempDir(), "vault")
	eventPath := filepath.Join(vault, "70 Pantheon Reports", "Integrated Events", "2026-08-22-event.md")
	if err := os.MkdirAll(filepath.Dir(eventPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eventPath, []byte("# Event\n\nGrant requested a bounded Libra report proof.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d.cfg.VaultPath = vault
	d.cfg.ApolloReportsPath = ""
	d.cfg.LocalModel = "apollo-test"
	d.cfg.ApolloRequireCitations = true
	d.cfg.ApolloMaxReasoningDurationSeconds = 5
	d.zodiacClock = func() time.Time { return time.Date(2026, time.August, 23, 4, 0, 0, 0, location) }
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"content":"{\"answer\":\"A concrete follow-up is recorded [[70 Pantheon Reports/Integrated Events/2026-08-22-event.md]].\",\"status\":\"complete\",\"summary\":\"Evidence reviewed.\"}"}}`))
	}))
	defer model.Close()
	d.cfg.OllamaURL = model.URL
	notificationPath := filepath.Join(t.TempDir(), "notification.txt")
	notificationHelper := filepath.Join(t.TempDir(), "notify-helper")
	if err := os.WriteFile(notificationHelper, []byte("#!/bin/sh\nprintf '%s|%s|%s' \"$1\" \"$2\" \"$3\" > \""+notificationPath+"\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	oldNotify := notifyCommand
	notifyCommand = notificationHelper
	defer func() { notifyCommand = oldNotify }()

	daily, err := d.createLibraReportDefinition(libraReportDefinitionRequest{ConstellationID: "libra", Name: "Daily continuity", Cadence: "daily", LocalTime: "05:00", Timezone: "America/Los_Angeles", WindowPolicy: "previous_day", Destination: "situation_briefs", NotifyOnSuccess: true, CatchUpPolicy: zodiacCatchUpRunOnce})
	if err != nil {
		t.Fatal(err)
	}
	weekly, err := d.createLibraReportDefinition(libraReportDefinitionRequest{ConstellationID: "libra", Name: "Weekly continuity", Cadence: "weekly", LocalTime: "05:00", WeeklyDay: "SU", Timezone: "America/Los_Angeles", WindowPolicy: "rolling_7_days", Destination: "pantheon_reports_weekly", NotifyOnSuccess: false, CatchUpPolicy: zodiacCatchUpRunOnce})
	if err != nil {
		t.Fatal(err)
	}
	d.zodiacClock = func() time.Time { return time.Date(2026, time.August, 23, 6, 0, 0, 0, location) }
	first, err := d.zodiacRound(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Due != 2 {
		t.Fatalf("first Libra round=%+v", first)
	}
	for _, report := range []libraReportDefinition{daily, weekly} {
		var run libraReportRun
		if err := d.db.QueryRow(`SELECT run_id FROM libra_report_runs WHERE report_definition_id = ?`, report.ID).Scan(&run.RunID); err != nil {
			t.Fatal(err)
		}
		run, err = d.loadLibraReportRunByID(run.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if run.State != "completed" || run.AssemblyID == "" || run.ApolloReportID == "" || run.FinalReportPath == "" {
			t.Fatalf("incomplete Libra run=%+v", run)
		}
		if _, err := os.Stat(run.FinalReportPath); err != nil {
			t.Fatalf("final report missing: %v", err)
		}
	}
	notification, err := os.ReadFile(notificationPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(notification), "Daily continuity") {
		t.Fatalf("success notification=%q", notification)
	}
	second, err := d.zodiacRound(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if second.Due != 0 {
		t.Fatalf("second Libra round replayed work=%+v", second)
	}
	var runs int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM libra_report_runs WHERE report_definition_id IN (?, ?)`, daily.ID, weekly.ID).Scan(&runs); err != nil {
		t.Fatal(err)
	}
	if runs != 2 {
		t.Fatalf("Libra run count=%d", runs)
	}
}
