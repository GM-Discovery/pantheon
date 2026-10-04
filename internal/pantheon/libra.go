package pantheon

// Libra is Zodiac's reusable reporting constellation. It stores report
// definitions in Panopticon and materializes each definition as a typed,
// inspectable Zodiac schedule. Libra describes balance and evidence policy;
// Apollo/report assembly remain separate bounded stages.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const libraArchetype = "libra"

const (
	libraTriggerSchedule    = "schedule"
	libraTriggerOnIngestion = "on_ingestion"
	libraCadenceEvent       = "event"
	libraOutputSituation    = "situation_briefs"
	libraOutputWeekly       = "pantheon_reports_weekly"
	libraOutputIntegrated   = "integrated_events"
)

var libraDefaultSections = []string{
	"summary",
	"requests",
	"commitments",
	"decisions",
	"questions",
	"uncertainties",
}

var libraAllowedSections = map[string]string{
	"summary":       "Concise summary",
	"requests":      "Concrete requests and action items",
	"commitments":   "Commitments and deadlines",
	"decisions":     "Decisions",
	"questions":     "Unresolved questions",
	"uncertainties": "Uncertainties and gaps",
	"people":        "People and identity context",
	"coverage":      "Source coverage and evidence gaps",
}

var libraAllowedSourceFamilies = map[string]bool{
	"all":               true,
	"transcripts":       true,
	"apollo_reports":    true,
	"integrated_events": true,
	"weekly_reports":    true,
	"situation_briefs":  true,
	"mail":              true,
	"vault":             true,
}

var libraAllowedOutputs = map[string]bool{
	libraOutputSituation:  true,
	libraOutputWeekly:     true,
	libraOutputIntegrated: true,
}

type libraReportRecipe struct {
	Trigger        string   `json:"trigger,omitempty"`
	SourceFamilies []string `json:"source_families,omitempty"`
	IncludePaths   []string `json:"include_paths,omitempty"`
	MinimumSources int      `json:"minimum_sources,omitempty"`
	Sections       []string `json:"sections,omitempty"`
	Instructions   string   `json:"instructions,omitempty"`
	Outputs        []string `json:"outputs,omitempty"`
}

type zodiacConstellation struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Archetype   string          `json:"archetype"`
	Description string          `json:"description"`
	Owner       string          `json:"owner"`
	Timezone    string          `json:"timezone"`
	Enabled     bool            `json:"enabled"`
	Policy      json.RawMessage `json:"policy,omitempty"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

type zodiacConstellationRequest struct {
	ID          string          `json:"id,omitempty"`
	Name        string          `json:"name"`
	Archetype   string          `json:"archetype,omitempty"`
	Description string          `json:"description,omitempty"`
	Owner       string          `json:"owner,omitempty"`
	Timezone    string          `json:"timezone,omitempty"`
	Enabled     *bool           `json:"enabled,omitempty"`
	Policy      json.RawMessage `json:"policy,omitempty"`
}

type libraReportDefinition struct {
	ID              string            `json:"id"`
	ConstellationID string            `json:"constellation_id"`
	Name            string            `json:"name"`
	Cadence         string            `json:"cadence"`
	LocalTime       string            `json:"local_time"`
	Timezone        string            `json:"timezone"`
	WeeklyDay       string            `json:"weekly_day,omitempty"`
	WindowPolicy    string            `json:"window_policy"`
	Destination     string            `json:"destination"`
	NotifyOnSuccess bool              `json:"notify_on_success"`
	Recipe          libraReportRecipe `json:"recipe"`
	Enabled         bool              `json:"enabled"`
	ScheduleID      string            `json:"schedule_id"`
	CreatedAt       string            `json:"created_at"`
	UpdatedAt       string            `json:"updated_at"`
	LastRunID       string            `json:"last_run_id,omitempty"`
	LastRunState    string            `json:"last_run_state,omitempty"`
	LastReportPath  string            `json:"last_report_path,omitempty"`
	LastRunError    string            `json:"last_run_error,omitempty"`
}

type libraReportDefinitionRequest struct {
	ConstellationID string   `json:"constellation_id"`
	Name            string   `json:"name"`
	Cadence         string   `json:"cadence"`
	LocalTime       string   `json:"local_time"`
	Timezone        string   `json:"timezone,omitempty"`
	WeeklyDay       string   `json:"weekly_day,omitempty"`
	WindowPolicy    string   `json:"window_policy"`
	Destination     string   `json:"destination"`
	NotifyOnSuccess bool     `json:"notify_on_success"`
	Trigger         string   `json:"trigger,omitempty"`
	SourceFamilies  []string `json:"source_families,omitempty"`
	IncludePaths    []string `json:"include_paths,omitempty"`
	MinimumSources  int      `json:"minimum_sources,omitempty"`
	Sections        []string `json:"sections,omitempty"`
	Instructions    string   `json:"instructions,omitempty"`
	Outputs         []string `json:"outputs,omitempty"`
	CatchUpPolicy   string   `json:"catch_up_policy,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
}

type libraBackfillRequest struct {
	ReportDefinitionID string `json:"report_definition_id,omitempty"`
	StartDate          string `json:"start_date"`
	EndDate            string `json:"end_date"`
	BatchBytes         int    `json:"batch_bytes,omitempty"`
}

type libraBackfillResult struct {
	ReportDefinitionID string           `json:"report_definition_id"`
	StartDate          string           `json:"start_date"`
	EndDate            string           `json:"end_date"`
	SourceCount        int              `json:"source_count"`
	BatchCount         int              `json:"batch_count"`
	BatchRuns          []libraReportRun `json:"batch_runs"`
	FinalRun           libraReportRun   `json:"final_run"`
}

type libraReportRun struct {
	RunID               string              `json:"run_id"`
	ReportDefinitionID  string              `json:"report_definition_id"`
	ScheduleID          string              `json:"schedule_id"`
	InstanceID          string              `json:"instance_id"`
	WindowKey           string              `json:"window_key"`
	WindowStart         string              `json:"window_start"`
	WindowEnd           string              `json:"window_end"`
	State               string              `json:"state"`
	AssemblyID          string              `json:"assembly_id,omitempty"`
	ApolloOrderID       string              `json:"apollo_order_id,omitempty"`
	ApolloReportID      string              `json:"apollo_report_id,omitempty"`
	ApolloReportPath    string              `json:"apollo_report_path,omitempty"`
	FinalReportPath     string              `json:"final_report_path,omitempty"`
	NotificationOrderID string              `json:"notification_order_id,omitempty"`
	NotificationState   string              `json:"notification_state,omitempty"`
	Error               string              `json:"error,omitempty"`
	Outputs             []libraReportOutput `json:"outputs,omitempty"`
	CreatedAt           string              `json:"created_at"`
	UpdatedAt           string              `json:"updated_at"`
}

type libraReportOutput struct {
	OutputID    string `json:"output_id"`
	RunID       string `json:"run_id"`
	Destination string `json:"destination"`
	Path        string `json:"path,omitempty"`
	BodySHA256  string `json:"body_sha256,omitempty"`
	State       string `json:"state"`
	Error       string `json:"error,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type libraReportWindow struct {
	Key       string
	StartDate string
	EndDate   string
	Start     time.Time
	End       time.Time
}

var libraNeedsEvidence = errors.New("Libra found no verified source material in the configured reporting window")

func initializeLibraSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS zodiac_constellations (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, archetype TEXT NOT NULL,
			description TEXT NOT NULL, owner TEXT NOT NULL, timezone TEXT NOT NULL,
			enabled INTEGER NOT NULL, policy_json TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS zodiac_constellations_enabled_idx ON zodiac_constellations(enabled, id)`,
		`CREATE TABLE IF NOT EXISTS zodiac_report_definitions (
			id TEXT PRIMARY KEY, constellation_id TEXT NOT NULL, name TEXT NOT NULL,
			cadence TEXT NOT NULL, local_time TEXT NOT NULL, timezone TEXT NOT NULL,
			weekly_day TEXT, window_policy TEXT NOT NULL, destination TEXT NOT NULL,
			notify_on_success INTEGER NOT NULL, enabled INTEGER NOT NULL,
			schedule_id TEXT NOT NULL, recipe_json TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS zodiac_report_definitions_constellation_idx ON zodiac_report_definitions(constellation_id, enabled, id)`,
		`CREATE TABLE IF NOT EXISTS libra_report_runs (
			run_id TEXT PRIMARY KEY, report_definition_id TEXT NOT NULL, schedule_id TEXT NOT NULL,
			instance_id TEXT NOT NULL, window_key TEXT NOT NULL, window_start TEXT NOT NULL,
			window_end TEXT NOT NULL, state TEXT NOT NULL, assembly_id TEXT, apollo_order_id TEXT,
			apollo_report_id TEXT, apollo_report_path TEXT, final_report_path TEXT,
			notification_order_id TEXT, notification_state TEXT, error TEXT,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
			UNIQUE(report_definition_id, window_key)
		)`,
		`CREATE INDEX IF NOT EXISTS libra_report_runs_state_idx ON libra_report_runs(state, updated_at DESC)`,
		`CREATE TABLE IF NOT EXISTS libra_report_outputs (
			output_id TEXT PRIMARY KEY, run_id TEXT NOT NULL, destination TEXT NOT NULL,
			path TEXT, body_sha256 TEXT, state TEXT NOT NULL, error TEXT,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
			UNIQUE(run_id, destination)
		)`,
		`CREATE INDEX IF NOT EXISTS libra_report_outputs_run_idx ON libra_report_outputs(run_id, destination)`,
		`CREATE TABLE IF NOT EXISTS libra_ingestion_jobs (
			job_id TEXT PRIMARY KEY, report_definition_id TEXT NOT NULL, source_kind TEXT NOT NULL,
			source_id TEXT NOT NULL, receipt_id TEXT NOT NULL, state TEXT NOT NULL,
			error TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
			UNIQUE(report_definition_id, source_kind, source_id)
		)`,
		`CREATE INDEX IF NOT EXISTS libra_ingestion_jobs_state_idx ON libra_ingestion_jobs(state, updated_at)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	// Kernel 28 created this table before recipes existed. Keep existing
	// installations readable while adding the recipe column in place.
	if _, err := db.Exec(`ALTER TABLE zodiac_report_definitions ADD COLUMN recipe_json TEXT NOT NULL DEFAULT '{}'`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return err
	}
	return nil
}

func (d *daemon) ensureLibraDefaults() error {
	if err := initializeLibraSchema(d.db); err != nil {
		return err
	}
	var existing string
	err := d.db.QueryRow(`SELECT id FROM zodiac_constellations WHERE id = ?`, "libra").Scan(&existing)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	now := d.zodiacNow().Format(time.RFC3339Nano)
	policy, _ := json.Marshal(map[string]any{
		"balance":    []string{"time windows", "source coverage", "evidence and interpretation", "unresolved and proposed states"},
		"activation": "report definitions are created explicitly in Panopticon",
	})
	_, err = d.db.Exec(`INSERT INTO zodiac_constellations (id, name, archetype, description, owner, timezone, enabled, policy_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"libra", "Libra — balanced reports", libraArchetype,
		"A reusable reporting pattern that makes time windows, source coverage, evidence, and uncertainty visible.",
		"panopticon", d.cfg.ZodiacDefaultTimezone, 1, string(policy), now, now)
	return err
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}

func normalizeLibraRecipe(request libraReportDefinitionRequest) (libraReportRecipe, error) {
	trigger := strings.ToLower(strings.TrimSpace(request.Trigger))
	if trigger == "" {
		trigger = libraTriggerSchedule
	}
	if trigger != libraTriggerSchedule && trigger != libraTriggerOnIngestion {
		return libraReportRecipe{}, errors.New("trigger must be schedule or on_ingestion")
	}
	recipe := libraReportRecipe{Trigger: trigger, MinimumSources: request.MinimumSources}
	if recipe.MinimumSources < 1 {
		recipe.MinimumSources = 1
	}
	if recipe.MinimumSources > 10000 {
		return libraReportRecipe{}, errors.New("minimum_sources must be between 1 and 10000")
	}
	for _, family := range request.SourceFamilies {
		family = strings.ToLower(strings.TrimSpace(family))
		if family == "" {
			continue
		}
		if !libraAllowedSourceFamilies[family] {
			return libraReportRecipe{}, fmt.Errorf("unsupported source family %q", family)
		}
		if !containsLibraString(recipe.SourceFamilies, family) {
			recipe.SourceFamilies = append(recipe.SourceFamilies, family)
		}
	}
	if len(recipe.SourceFamilies) == 0 || containsLibraString(recipe.SourceFamilies, "all") {
		recipe.SourceFamilies = []string{"all"}
	}
	for _, rawPath := range request.IncludePaths {
		path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(rawPath))))
		if path == "." || path == "" {
			continue
		}
		if filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, "../") {
			return libraReportRecipe{}, fmt.Errorf("include_paths must be Vault-relative: %q", rawPath)
		}
		if !containsLibraString(recipe.IncludePaths, path) {
			recipe.IncludePaths = append(recipe.IncludePaths, path)
		}
	}
	for _, section := range request.Sections {
		section = strings.ToLower(strings.TrimSpace(section))
		if section == "" {
			continue
		}
		if _, ok := libraAllowedSections[section]; !ok {
			return libraReportRecipe{}, fmt.Errorf("unsupported report section %q", section)
		}
		if !containsLibraString(recipe.Sections, section) {
			recipe.Sections = append(recipe.Sections, section)
		}
	}
	if len(recipe.Sections) == 0 {
		recipe.Sections = cloneStrings(libraDefaultSections)
	}
	recipe.Instructions = strings.TrimSpace(request.Instructions)
	if len(recipe.Instructions) > 4000 {
		return libraReportRecipe{}, errors.New("instructions cannot exceed 4000 characters")
	}
	for _, output := range request.Outputs {
		output = strings.ToLower(strings.TrimSpace(output))
		if output == "" {
			continue
		}
		if !libraAllowedOutputs[output] {
			return libraReportRecipe{}, fmt.Errorf("unsupported report output %q", output)
		}
		if !containsLibraString(recipe.Outputs, output) {
			recipe.Outputs = append(recipe.Outputs, output)
		}
	}
	if len(recipe.Outputs) == 0 {
		recipe.Outputs = []string{request.Destination}
	}
	if trigger == libraTriggerOnIngestion {
		if len(recipe.Outputs) == 0 {
			return libraReportRecipe{}, errors.New("on_ingestion reports require at least one output")
		}
	}
	return recipe, nil
}

func parseLibraRecipe(raw string, destination, cadence string) libraReportRecipe {
	var recipe libraReportRecipe
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &recipe)
	}
	if recipe.Trigger == "" {
		recipe.Trigger = libraTriggerSchedule
	}
	if len(recipe.SourceFamilies) == 0 {
		recipe.SourceFamilies = []string{"all"}
	}
	if len(recipe.Sections) == 0 {
		recipe.Sections = cloneStrings(libraDefaultSections)
	}
	if recipe.MinimumSources < 1 {
		recipe.MinimumSources = 1
	}
	if len(recipe.Outputs) == 0 && destination != "" {
		recipe.Outputs = []string{destination}
	}
	if recipe.Trigger != libraTriggerOnIngestion && cadence == libraCadenceEvent {
		recipe.Trigger = libraTriggerOnIngestion
	}
	return recipe
}

func containsLibraString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func (d *daemon) createZodiacConstellation(request zodiacConstellationRequest) (zodiacConstellation, error) {
	request.Name = strings.TrimSpace(request.Name)
	request.Archetype = strings.TrimSpace(request.Archetype)
	request.Description = strings.TrimSpace(request.Description)
	request.Owner = strings.TrimSpace(request.Owner)
	request.Timezone = strings.TrimSpace(request.Timezone)
	if request.Name == "" {
		return zodiacConstellation{}, errors.New("constellation name is required")
	}
	if request.Archetype == "" {
		request.Archetype = libraArchetype
	}
	if request.Archetype != libraArchetype {
		return zodiacConstellation{}, fmt.Errorf("unsupported constellation archetype %q", request.Archetype)
	}
	if request.Owner == "" {
		request.Owner = "panopticon"
	}
	if request.Timezone == "" {
		request.Timezone = d.cfg.ZodiacDefaultTimezone
	}
	if _, err := time.LoadLocation(request.Timezone); err != nil {
		return zodiacConstellation{}, fmt.Errorf("timezone: %w", err)
	}
	if request.Policy != nil && !json.Valid(request.Policy) {
		return zodiacConstellation{}, errors.New("policy must be valid JSON")
	}
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	id := strings.TrimSpace(request.ID)
	if id == "" {
		var err error
		id, err = newLocalID("constellation-")
		if err != nil {
			return zodiacConstellation{}, err
		}
	}
	now := d.zodiacNow().Format(time.RFC3339Nano)
	item := zodiacConstellation{ID: id, Name: request.Name, Archetype: request.Archetype, Description: request.Description, Owner: request.Owner, Timezone: request.Timezone, Enabled: enabled, Policy: request.Policy, CreatedAt: now, UpdatedAt: now}
	_, err := d.db.Exec(`INSERT INTO zodiac_constellations (id, name, archetype, description, owner, timezone, enabled, policy_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, item.Name, item.Archetype, item.Description, item.Owner, item.Timezone, item.Enabled, nullableJSON(item.Policy), item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return zodiacConstellation{}, err
	}
	return item, nil
}

func (d *daemon) listZodiacConstellations() ([]zodiacConstellation, error) {
	rows, err := d.db.Query(`SELECT id, name, archetype, description, owner, timezone, enabled, COALESCE(policy_json, ''), created_at, updated_at FROM zodiac_constellations ORDER BY enabled DESC, name, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []zodiacConstellation{}
	for rows.Next() {
		var item zodiacConstellation
		var enabled int
		var policy string
		if err := rows.Scan(&item.ID, &item.Name, &item.Archetype, &item.Description, &item.Owner, &item.Timezone, &enabled, &policy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Enabled = enabled != 0
		if policy != "" {
			item.Policy = json.RawMessage(policy)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *daemon) loadZodiacConstellation(id string) (zodiacConstellation, error) {
	var item zodiacConstellation
	var enabled int
	var policy string
	err := d.db.QueryRow(`SELECT id, name, archetype, description, owner, timezone, enabled, COALESCE(policy_json, ''), created_at, updated_at FROM zodiac_constellations WHERE id = ?`, id).Scan(&item.ID, &item.Name, &item.Archetype, &item.Description, &item.Owner, &item.Timezone, &enabled, &policy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return zodiacConstellation{}, err
	}
	item.Enabled = enabled != 0
	if policy != "" {
		item.Policy = json.RawMessage(policy)
	}
	return item, nil
}

func validateLibraReportRequest(d *daemon, request *libraReportDefinitionRequest) (zodiacConstellation, time.Time, error) {
	request.ConstellationID = strings.TrimSpace(request.ConstellationID)
	request.Name = strings.TrimSpace(request.Name)
	request.Cadence = strings.ToLower(strings.TrimSpace(request.Cadence))
	request.LocalTime = strings.TrimSpace(request.LocalTime)
	request.Timezone = strings.TrimSpace(request.Timezone)
	request.WeeklyDay = strings.ToUpper(strings.TrimSpace(request.WeeklyDay))
	request.WindowPolicy = strings.ToLower(strings.TrimSpace(request.WindowPolicy))
	request.Destination = strings.ToLower(strings.TrimSpace(request.Destination))
	request.Trigger = strings.ToLower(strings.TrimSpace(request.Trigger))
	if request.Trigger == "" {
		request.Trigger = libraTriggerSchedule
	}
	if request.Trigger == libraTriggerOnIngestion && request.WindowPolicy == "" {
		request.WindowPolicy = "previous_day"
	}
	if request.Destination == "" && len(request.Outputs) > 0 {
		request.Destination = strings.ToLower(strings.TrimSpace(request.Outputs[0]))
	}
	if request.ConstellationID == "" || request.Name == "" || request.WindowPolicy == "" || request.Destination == "" {
		return zodiacConstellation{}, time.Time{}, errors.New("constellation_id, name, cadence, local_time, window_policy, and destination are required")
	}
	if request.Trigger != libraTriggerSchedule && request.Trigger != libraTriggerOnIngestion {
		return zodiacConstellation{}, time.Time{}, errors.New("trigger must be schedule or on_ingestion")
	}
	constellation, err := d.loadZodiacConstellation(request.ConstellationID)
	if err != nil {
		return zodiacConstellation{}, time.Time{}, err
	}
	if constellation.Archetype != libraArchetype {
		return zodiacConstellation{}, time.Time{}, errors.New("report definitions require a Libra constellation")
	}
	if request.Timezone == "" {
		request.Timezone = constellation.Timezone
	}
	location, err := time.LoadLocation(request.Timezone)
	if err != nil {
		return zodiacConstellation{}, time.Time{}, fmt.Errorf("timezone: %w", err)
	}
	var clock time.Time
	if request.Trigger == libraTriggerOnIngestion {
		request.Cadence = libraCadenceEvent
		request.LocalTime = ""
		request.WeeklyDay = ""
	} else {
		if request.Cadence == "" || request.LocalTime == "" {
			return zodiacConstellation{}, time.Time{}, errors.New("scheduled reports require cadence and local_time")
		}
		clock, err = time.ParseInLocation("15:04", request.LocalTime, location)
		if err != nil {
			return zodiacConstellation{}, time.Time{}, errors.New("local_time must use HH:MM")
		}
		if request.Cadence != "daily" && request.Cadence != "weekly" {
			return zodiacConstellation{}, time.Time{}, errors.New("cadence must be daily or weekly")
		}
		if request.Cadence == "weekly" {
			if request.WeeklyDay == "" {
				request.WeeklyDay = "MO"
			}
			if _, ok := zodiacWeekday(request.WeeklyDay); !ok {
				return zodiacConstellation{}, time.Time{}, errors.New("weekly_day must be SU, MO, TU, WE, TH, FR, or SA")
			}
		} else {
			request.WeeklyDay = ""
		}
	}
	validWindows := map[string]bool{"previous_day": true, "previous_week": true, "rolling_7_days": true}
	if !validWindows[request.WindowPolicy] {
		return zodiacConstellation{}, time.Time{}, errors.New("window_policy must be previous_day, previous_week, or rolling_7_days")
	}
	validDestinations := map[string]bool{libraOutputSituation: true, libraOutputWeekly: true, libraOutputIntegrated: true}
	if !validDestinations[request.Destination] {
		return zodiacConstellation{}, time.Time{}, fmt.Errorf("unsupported report destination %q", request.Destination)
	}
	if request.CatchUpPolicy == "" {
		request.CatchUpPolicy = zodiacCatchUpRunOnce
	}
	if request.CatchUpPolicy != zodiacCatchUpSkip && request.CatchUpPolicy != zodiacCatchUpRunOnce && request.CatchUpPolicy != zodiacCatchUpReconcile && request.CatchUpPolicy != zodiacCatchUpNeedsYou {
		return zodiacConstellation{}, time.Time{}, errors.New("unsupported catch_up_policy")
	}
	if request.Trigger == libraTriggerOnIngestion {
		return constellation, time.Time{}, nil
	}
	now := d.zodiacNow().In(location)
	start := time.Date(now.Year(), now.Month(), now.Day(), clock.Hour(), clock.Minute(), 0, 0, location)
	if request.Cadence == "daily" {
		if !start.After(now) {
			start = start.AddDate(0, 0, 1)
		}
	} else {
		weekday, _ := zodiacWeekday(request.WeeklyDay)
		for start.Weekday() != weekday || !start.After(now) {
			start = start.AddDate(0, 0, 1)
		}
	}
	return constellation, start, nil
}

func (d *daemon) createLibraReportDefinition(request libraReportDefinitionRequest) (libraReportDefinition, error) {
	constellation, start, err := validateLibraReportRequest(d, &request)
	if err != nil {
		return libraReportDefinition{}, err
	}
	recipe, err := normalizeLibraRecipe(request)
	if err != nil {
		return libraReportDefinition{}, err
	}
	id, err := newLocalID("libra-report-")
	if err != nil {
		return libraReportDefinition{}, err
	}
	scheduleID := ""
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	payload, _ := json.Marshal(map[string]any{
		"constellation_id":     request.ConstellationID,
		"report_definition_id": id,
		"cadence":              request.Cadence,
		"window_policy":        request.WindowPolicy,
		"destination":          request.Destination,
		"trigger":              recipe.Trigger,
		"outputs":              recipe.Outputs,
		"notification":         map[string]any{"on_success": request.NotifyOnSuccess},
	})
	if recipe.Trigger == libraTriggerSchedule {
		scheduleID = "zodiac-" + id
		recurrence := "RRULE:FREQ=DAILY;INTERVAL=1"
		if request.Cadence == "weekly" {
			recurrence = "RRULE:FREQ=WEEKLY;INTERVAL=1;BYDAY=" + request.WeeklyDay
		}
		schedule, scheduleErr := d.createZodiacSchedule(zodiacScheduleRequest{
			ID: scheduleID, Name: request.Name, Kind: "recurring_system_job", Owner: "libra", Source: "panopticon", SourceID: id,
			Enabled: &enabled, Timezone: request.Timezone, StartAt: start.Format("2006-01-02T15:04:05") + zoneOffset(start), Recurrence: recurrence,
			Precision: zodiacPrecisionExact, CatchUpPolicy: request.CatchUpPolicy, ProjectionPolicy: zodiacProjectionNone, Priority: "person_requested",
			Action: zodiacActionBriefGenerate, Payload: payload,
		})
		if scheduleErr != nil {
			return libraReportDefinition{}, scheduleErr
		}
		scheduleID = schedule.ID
	}
	now := d.zodiacNow().Format(time.RFC3339Nano)
	recipeJSON, _ := json.Marshal(recipe)
	item := libraReportDefinition{ID: id, ConstellationID: constellation.ID, Name: request.Name, Cadence: request.Cadence, LocalTime: request.LocalTime, Timezone: request.Timezone, WeeklyDay: request.WeeklyDay, WindowPolicy: request.WindowPolicy, Destination: request.Destination, NotifyOnSuccess: request.NotifyOnSuccess, Recipe: recipe, Enabled: enabled, ScheduleID: scheduleID, CreatedAt: now, UpdatedAt: now}
	_, err = d.db.Exec(`INSERT INTO zodiac_report_definitions (id, constellation_id, name, cadence, local_time, timezone, weekly_day, window_policy, destination, notify_on_success, enabled, schedule_id, recipe_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, item.ConstellationID, item.Name, item.Cadence, item.LocalTime, item.Timezone, nullableString(item.WeeklyDay), item.WindowPolicy, item.Destination, item.NotifyOnSuccess, item.Enabled, item.ScheduleID, string(recipeJSON), item.CreatedAt, item.UpdatedAt)
	if err != nil {
		if scheduleID != "" {
			_, _ = d.db.Exec(`DELETE FROM zodiac_schedules WHERE id = ?`, scheduleID)
		}
		return libraReportDefinition{}, err
	}
	return item, nil
}

func (d *daemon) loadLibraReportRunByID(id string) (libraReportRun, error) {
	var run libraReportRun
	var assemblyID, apolloOrderID, apolloReportID, apolloReportPath, finalReportPath, notificationOrderID, notificationState, lastError sql.NullString
	err := d.db.QueryRow(`SELECT run_id, report_definition_id, schedule_id, instance_id, window_key, window_start, window_end, state, COALESCE(assembly_id, ''), COALESCE(apollo_order_id, ''), COALESCE(apollo_report_id, ''), COALESCE(apollo_report_path, ''), COALESCE(final_report_path, ''), COALESCE(notification_order_id, ''), COALESCE(notification_state, ''), COALESCE(error, ''), created_at, updated_at FROM libra_report_runs WHERE run_id = ?`, id).Scan(&run.RunID, &run.ReportDefinitionID, &run.ScheduleID, &run.InstanceID, &run.WindowKey, &run.WindowStart, &run.WindowEnd, &run.State, &assemblyID, &apolloOrderID, &apolloReportID, &apolloReportPath, &finalReportPath, &notificationOrderID, &notificationState, &lastError, &run.CreatedAt, &run.UpdatedAt)
	if err != nil {
		return libraReportRun{}, err
	}
	run.AssemblyID, run.ApolloOrderID, run.ApolloReportID, run.ApolloReportPath = assemblyID.String, apolloOrderID.String, apolloReportID.String, apolloReportPath.String
	run.FinalReportPath, run.NotificationOrderID, run.NotificationState, run.Error = finalReportPath.String, notificationOrderID.String, notificationState.String, lastError.String
	run.Outputs, err = d.listLibraReportOutputs(run.RunID)
	if err != nil {
		return libraReportRun{}, err
	}
	return run, nil
}

func (d *daemon) listLibraReportOutputs(runID string) ([]libraReportOutput, error) {
	rows, err := d.db.Query(`SELECT output_id, run_id, destination, COALESCE(path, ''), COALESCE(body_sha256, ''), state, COALESCE(error, ''), created_at, updated_at FROM libra_report_outputs WHERE run_id = ? ORDER BY destination, output_id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []libraReportOutput{}
	for rows.Next() {
		var output libraReportOutput
		if err := rows.Scan(&output.OutputID, &output.RunID, &output.Destination, &output.Path, &output.BodySHA256, &output.State, &output.Error, &output.CreatedAt, &output.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, output)
	}
	return result, rows.Err()
}

func (d *daemon) latestLibraReportRun(reportDefinitionID string) (libraReportRun, error) {
	var id string
	if err := d.db.QueryRow(`SELECT run_id FROM libra_report_runs WHERE report_definition_id = ? ORDER BY updated_at DESC, run_id DESC LIMIT 1`, reportDefinitionID).Scan(&id); err != nil {
		return libraReportRun{}, err
	}
	return d.loadLibraReportRunByID(id)
}

func (d *daemon) listLibraReportRuns(limit int) ([]libraReportRun, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := d.db.Query(`SELECT run_id FROM libra_report_runs ORDER BY updated_at DESC, run_id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []libraReportRun{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		run, err := d.loadLibraReportRunByID(id)
		if err != nil {
			return nil, err
		}
		result = append(result, run)
	}
	return result, rows.Err()
}

func (d *daemon) listLibraReportDefinitions() ([]libraReportDefinition, error) {
	rows, err := d.db.Query(`SELECT id, constellation_id, name, cadence, local_time, timezone, COALESCE(weekly_day, ''), window_policy, destination, notify_on_success, enabled, schedule_id, COALESCE(recipe_json, '{}'), created_at, updated_at FROM zodiac_report_definitions ORDER BY enabled DESC, name, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []libraReportDefinition{}
	for rows.Next() {
		var item libraReportDefinition
		var notify, enabled int
		var recipeJSON string
		if err := rows.Scan(&item.ID, &item.ConstellationID, &item.Name, &item.Cadence, &item.LocalTime, &item.Timezone, &item.WeeklyDay, &item.WindowPolicy, &item.Destination, &notify, &enabled, &item.ScheduleID, &recipeJSON, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.NotifyOnSuccess, item.Enabled = notify != 0, enabled != 0
		item.Recipe = parseLibraRecipe(recipeJSON, item.Destination, item.Cadence)
		if run, runErr := d.latestLibraReportRun(item.ID); runErr == nil {
			item.LastRunID, item.LastRunState, item.LastReportPath, item.LastRunError = run.RunID, run.State, run.FinalReportPath, run.Error
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *daemon) setLibraReportEnabled(id string, enabled bool) (libraReportDefinition, error) {
	var scheduleID string
	if err := d.db.QueryRow(`SELECT schedule_id FROM zodiac_report_definitions WHERE id = ?`, id).Scan(&scheduleID); err != nil {
		return libraReportDefinition{}, err
	}
	now := d.zodiacNow().Format(time.RFC3339Nano)
	status := zodiacScheduleDisabled
	if enabled {
		status = zodiacScheduleActive
	}
	if _, err := d.db.Exec(`UPDATE zodiac_report_definitions SET enabled = ?, updated_at = ? WHERE id = ?`, enabled, now, id); err != nil {
		return libraReportDefinition{}, err
	}
	if scheduleID != "" {
		if _, err := d.db.Exec(`UPDATE zodiac_schedules SET enabled = ?, status = ?, updated_at = ? WHERE id = ?`, enabled, status, now, scheduleID); err != nil {
			return libraReportDefinition{}, err
		}
	}
	items, err := d.listLibraReportDefinitions()
	if err != nil {
		return libraReportDefinition{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return libraReportDefinition{}, sql.ErrNoRows
}

func (d *daemon) loadLibraReportDefinition(id string) (libraReportDefinition, error) {
	var item libraReportDefinition
	var notify, enabled int
	var recipeJSON string
	err := d.db.QueryRow(`SELECT id, constellation_id, name, cadence, local_time, timezone, COALESCE(weekly_day, ''), window_policy, destination, notify_on_success, enabled, schedule_id, COALESCE(recipe_json, '{}'), created_at, updated_at FROM zodiac_report_definitions WHERE id = ?`, id).Scan(&item.ID, &item.ConstellationID, &item.Name, &item.Cadence, &item.LocalTime, &item.Timezone, &item.WeeklyDay, &item.WindowPolicy, &item.Destination, &notify, &enabled, &item.ScheduleID, &recipeJSON, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return libraReportDefinition{}, err
	}
	item.NotifyOnSuccess, item.Enabled = notify != 0, enabled != 0
	item.Recipe = parseLibraRecipe(recipeJSON, item.Destination, item.Cadence)
	return item, nil
}

func (d *daemon) loadLibraReportRunByWindow(reportDefinitionID, windowKey string) (libraReportRun, error) {
	var run libraReportRun
	var assemblyID, apolloOrderID, apolloReportID, apolloReportPath, finalReportPath, notificationOrderID, notificationState, lastError sql.NullString
	err := d.db.QueryRow(`SELECT run_id, report_definition_id, schedule_id, instance_id, window_key, window_start, window_end, state, COALESCE(assembly_id, ''), COALESCE(apollo_order_id, ''), COALESCE(apollo_report_id, ''), COALESCE(apollo_report_path, ''), COALESCE(final_report_path, ''), COALESCE(notification_order_id, ''), COALESCE(notification_state, ''), COALESCE(error, ''), created_at, updated_at FROM libra_report_runs WHERE report_definition_id = ? AND window_key = ?`, reportDefinitionID, windowKey).Scan(&run.RunID, &run.ReportDefinitionID, &run.ScheduleID, &run.InstanceID, &run.WindowKey, &run.WindowStart, &run.WindowEnd, &run.State, &assemblyID, &apolloOrderID, &apolloReportID, &apolloReportPath, &finalReportPath, &notificationOrderID, &notificationState, &lastError, &run.CreatedAt, &run.UpdatedAt)
	if err != nil {
		return libraReportRun{}, err
	}
	run.AssemblyID, run.ApolloOrderID, run.ApolloReportID, run.ApolloReportPath = assemblyID.String, apolloOrderID.String, apolloReportID.String, apolloReportPath.String
	run.FinalReportPath, run.NotificationOrderID, run.NotificationState, run.Error = finalReportPath.String, notificationOrderID.String, notificationState.String, lastError.String
	run.Outputs, err = d.listLibraReportOutputs(run.RunID)
	if err != nil {
		return libraReportRun{}, err
	}
	return run, nil
}

func (d *daemon) saveLibraReportRun(run libraReportRun) error {
	_, err := d.db.Exec(`UPDATE libra_report_runs SET state = ?, assembly_id = ?, apollo_order_id = ?, apollo_report_id = ?, apollo_report_path = ?, final_report_path = ?, notification_order_id = ?, notification_state = ?, error = ?, updated_at = ? WHERE run_id = ?`, run.State, nullableString(run.AssemblyID), nullableString(run.ApolloOrderID), nullableString(run.ApolloReportID), nullableString(run.ApolloReportPath), nullableString(run.FinalReportPath), nullableString(run.NotificationOrderID), nullableString(run.NotificationState), nullableString(run.Error), run.UpdatedAt, run.RunID)
	return err
}

func (d *daemon) createLibraReportRun(report libraReportDefinition, schedule zodiacSchedule, instance zodiacInstance, window libraReportWindow) (libraReportRun, error) {
	if existing, err := d.loadLibraReportRunByWindow(report.ID, window.Key); err == nil {
		if existing.State == "completed" || existing.State == "completed_notification_failed" {
			return existing, nil
		}
		if existing.State == "running" {
			return existing, errors.New("Libra report run is already in progress")
		}
		existing.InstanceID, existing.State, existing.Error, existing.UpdatedAt = instance.ID, "running", "", d.zodiacNow().Format(time.RFC3339Nano)
		if err := d.saveLibraReportRun(existing); err != nil {
			return libraReportRun{}, err
		}
		return existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return libraReportRun{}, err
	}
	runID, err := newLocalID("libra-run-")
	if err != nil {
		return libraReportRun{}, err
	}
	now := d.zodiacNow().Format(time.RFC3339Nano)
	run := libraReportRun{RunID: runID, ReportDefinitionID: report.ID, ScheduleID: schedule.ID, InstanceID: instance.ID, WindowKey: window.Key, WindowStart: window.StartDate, WindowEnd: window.EndDate, State: "running", CreatedAt: now, UpdatedAt: now}
	_, err = d.db.Exec(`INSERT INTO libra_report_runs (run_id, report_definition_id, schedule_id, instance_id, window_key, window_start, window_end, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, run.RunID, run.ReportDefinitionID, run.ScheduleID, run.InstanceID, run.WindowKey, run.WindowStart, run.WindowEnd, run.State, run.CreatedAt, run.UpdatedAt)
	if err != nil {
		return libraReportRun{}, err
	}
	return run, nil
}

func libraReportWindowFor(report libraReportDefinition, dueAt string) (libraReportWindow, error) {
	location, err := time.LoadLocation(report.Timezone)
	if err != nil {
		return libraReportWindow{}, err
	}
	due, err := time.Parse(time.RFC3339Nano, dueAt)
	if err != nil {
		return libraReportWindow{}, err
	}
	due = due.In(location)
	day := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, location)
	start, end := day.AddDate(0, 0, -1), day.AddDate(0, 0, -1)
	switch report.WindowPolicy {
	case "previous_day":
	case "rolling_7_days":
		start = end.AddDate(0, 0, -6)
	case "previous_week":
		mondayOffset := (int(day.Weekday()) + 6) % 7
		currentMonday := day.AddDate(0, 0, -mondayOffset)
		end = currentMonday.AddDate(0, 0, -1)
		start = end.AddDate(0, 0, -6)
	default:
		return libraReportWindow{}, fmt.Errorf("unsupported Libra window policy %q", report.WindowPolicy)
	}
	return libraReportWindow{Key: start.Format("2006-01-02") + ".." + end.Format("2006-01-02"), StartDate: start.Format("2006-01-02"), EndDate: end.Format("2006-01-02"), Start: start, End: end}, nil
}

func libraReportSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			lastDash = false
		} else if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "report"
	}
	if len(result) > 48 {
		result = result[:48]
	}
	return strings.Trim(result, "-")
}

func libraReportFinalPath(vault string, report libraReportDefinition, window libraReportWindow, destination string, runID string) string {
	slug := libraReportSlug(report.Name)
	switch destination {
	case libraOutputWeekly:
		return filepath.Join(vault, "70 Pantheon Reports", "Weekly", window.StartDate+" Weekly Report — "+slug+".md")
	case libraOutputIntegrated:
		suffix := strings.TrimPrefix(runID, "libra-run-")
		if len(suffix) > 12 {
			suffix = suffix[:12]
		}
		return filepath.Join(vault, "70 Pantheon Reports", "Integrated Events", window.EndDate+" "+slug+" — "+suffix+".md")
	default:
		return filepath.Join(vault, "10 Operations", "Situation Briefs", window.EndDate+" Situation Brief — "+slug+".md")
	}
}

func libraCosmicForecast(date string) string {
	forecasts := []string{
		"The stars advise turning one good idea into a next step before collecting seven more.",
		"Mercury is in retrograde, but your inbox is merely being dramatic.",
		"Protect your attention from tasks that arrived wearing a suspiciously urgent little hat.",
		"A useful pattern is approaching; please label it before it becomes a mystery six weeks from now.",
		"The cosmos favors small, reversible actions and aggressively labeled drafts.",
		"Someone will say ‘quick question.’ The stars recommend a timer and a chair with good lumbar support.",
		"Your future contains a cited report, which is already more evidence than most horoscopes can provide.",
	}
	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return forecasts[0]
	}
	index := (day.YearDay() + day.Year()) % len(forecasts)
	return forecasts[index]
}

func humanApolloReportBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if strings.HasPrefix(text, "---") {
		if end := strings.Index(text[3:], "\n---"); end >= 0 {
			text = strings.TrimSpace(text[end+3+len("\n---"):])
		}
	}
	lines := strings.Split(text, "\n")
	start := -1
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "## Human-readable result" || trimmed == "## Concise summary" {
			start = index + 1
			break
		}
	}
	if start < 0 {
		return strings.TrimSpace(text)
	}
	kept := make([]string, 0, len(lines)-start)
	for _, line := range lines[start:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "## Evidence" || trimmed == "## Future-ready result" || trimmed == "## Handoff" {
			break
		}
		if trimmed == "```" || strings.HasPrefix(trimmed, "```json") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

func formatLibraSources(assembly reportAssemblyReceipt) string {
	if len(assembly.Sources) == 0 {
		return "No verified source links were recorded."
	}
	lines := make([]string, 0, len(assembly.Sources))
	for _, source := range assembly.Sources {
		label := source.Kind
		if label == "" {
			label = "source"
		}
		lines = append(lines, "- "+label+": "+source.Citation)
	}
	return strings.Join(lines, "\n")
}

func renderLibraReport(run libraReportRun, report libraReportDefinition, assembly reportAssemblyReceipt, apolloBody []byte) ([]byte, string) {
	outputs := report.Recipe.Outputs
	if len(outputs) == 0 {
		outputs = []string{report.Destination}
	}
	humanBody := humanApolloReportBody(apolloBody)
	if humanBody == "" {
		humanBody = "No human-readable findings were produced."
	}
	body := fmt.Sprintf("# %s\n\n> Cosmic forecast: %s\n\n## Reporting window\n\n%s through %s\n\n## Human-readable findings\n\n%s\n\n## Evidence coverage\n\n- Verified source count: %d\n- Configured outputs: %s\n\n### Source links\n\n%s\n\n### Assembly receipt\n\n- Assembly: `%s`\n- Manifest: `%s`\n", report.Name, libraCosmicForecast(run.WindowEnd), run.WindowStart, run.WindowEnd, humanBody, assembly.SourceCount, strings.Join(outputs, ", "), formatLibraSources(assembly), assembly.AssemblyID, filepath.ToSlash(assembly.ManifestPath))
	bodyHash := sha256Bytes([]byte(body))
	frontmatter := fmt.Sprintf("---\npantheon:\n  kind: libra-report\n  report_definition_id: %s\n  run_id: %s\n  window_start: %s\n  window_end: %s\n  assembly_id: %s\n  body_sha256: %s\n---\n\n", yamlQuote(report.ID), yamlQuote(run.RunID), yamlQuote(run.WindowStart), yamlQuote(run.WindowEnd), yamlQuote(assembly.AssemblyID), yamlQuote(bodyHash))
	return []byte(frontmatter + body), bodyHash
}

func (d *daemon) writeLibraReport(run libraReportRun, report libraReportDefinition, assembly reportAssemblyReceipt, apolloPath string) (string, error) {
	if d.cfg.VaultPath == "" {
		return "", errors.New("Libra report requires a configured Vault")
	}
	if !pathWithin(d.cfg.VaultPath, apolloPath) {
		return "", errors.New("Apollo report is outside the configured Vault")
	}
	apolloBody, err := os.ReadFile(apolloPath)
	if err != nil {
		return "", err
	}
	if len(apolloBody) == 0 {
		return "", errors.New("Apollo report is empty")
	}
	outputs := report.Recipe.Outputs
	if len(outputs) == 0 {
		outputs = []string{report.Destination}
	}
	data, bodyHash := renderLibraReport(run, report, assembly, apolloBody)
	window := libraReportWindow{StartDate: run.WindowStart, EndDate: run.WindowEnd}
	var primaryPath string
	for _, destination := range outputs {
		if !libraAllowedOutputs[destination] {
			return "", fmt.Errorf("unsupported Libra output %q", destination)
		}
		path := libraReportFinalPath(d.cfg.VaultPath, report, window, destination, run.RunID)
		if primaryPath == "" {
			primaryPath = path
		}
		if err := ensureNoSymlinkComponents(filepath.Dir(path)); err != nil {
			return "", err
		}
		if existing, readErr := os.ReadFile(path); readErr == nil {
			if sha256Bytes(existing) != sha256Bytes(data) {
				return "", fmt.Errorf("Libra output already exists with different content: %s", path)
			}
		} else if errors.Is(readErr, os.ErrNotExist) {
			if err := writeReportAssemblyFile(path, data); err != nil {
				return "", err
			}
		} else {
			return "", readErr
		}
		verified, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		if len(verified) != len(data) || sha256Bytes(verified) != sha256Bytes(data) {
			return "", errors.New("Libra report verification failed after Vault write")
		}
		now := d.zodiacNow().Format(time.RFC3339Nano)
		outputID := "libra-output-" + sha256Bytes([]byte(run.RunID + "\x00" + destination))[:24]
		_, err = d.db.Exec(`INSERT INTO libra_report_outputs (output_id, run_id, destination, path, body_sha256, state, error, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(run_id, destination) DO UPDATE SET path = excluded.path, body_sha256 = excluded.body_sha256, state = excluded.state, error = excluded.error, updated_at = excluded.updated_at`, outputID, run.RunID, destination, path, bodyHash, "completed", nil, now, now)
		if err != nil {
			return "", err
		}
	}
	return primaryPath, nil
}

func (d *daemon) dispatchLibraOrder(ctx context.Context, order mercuryOrder) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if order.Status != orderStatusSealed {
		return fmt.Errorf("Libra order is not sealed: %s", order.Status)
	}
	if paused, err := d.paused(); err != nil {
		return err
	} else if paused {
		return errors.New("Prometheus is paused")
	}
	command, err := d.issueGo(order)
	if err != nil {
		return err
	}
	return d.dispatchOrder(ctx, order, command)
}

func (d *daemon) loadApolloRunResult(orderID string) (apolloRunResult, error) {
	var result apolloRunResult
	var futureJSON string
	if err := d.db.QueryRow(`SELECT report_id, report_path, future_result_json FROM apollo_reports WHERE order_id = ? ORDER BY created_at DESC LIMIT 1`, orderID).Scan(&result.ReportID, &result.ReportPath, &futureJSON); err != nil {
		return apolloRunResult{}, err
	}
	if err := json.Unmarshal([]byte(futureJSON), &result.FutureResult); err != nil {
		return apolloRunResult{}, err
	}
	result.Status = result.FutureResult.Status
	return result, nil
}

func (d *daemon) deliverLibraNotification(ctx context.Context, report libraReportDefinition, run libraReportRun) (string, error) {
	payload, _ := json.Marshal(map[string]string{"title": "Libra report ready", "message": report.Name + " is ready for " + run.WindowStart + " through " + run.WindowEnd + ". Vault report: " + run.FinalReportPath})
	order, err := d.createOrder(orderRequest{RequestText: "Deliver the configured local success notification for Libra report " + report.Name + ".", Recipient: "local-notify", Capability: "notify.send", Scope: "notification:libra:" + run.RunID, Payload: payload, Purpose: "Libra report success notification"}, true)
	if err != nil {
		return "", err
	}
	if err := d.dispatchLibraOrder(ctx, order); err != nil {
		return order.OrderID, err
	}
	return order.OrderID, nil
}

func libraReportPrompt(report libraReportDefinition, window libraReportWindow) string {
	sections := report.Recipe.Sections
	if len(sections) == 0 {
		sections = libraDefaultSections
	}
	labels := make([]string, 0, len(sections))
	for _, section := range sections {
		if label, ok := libraAllowedSections[section]; ok {
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		labels = []string{"Concise summary"}
	}
	prompt := fmt.Sprintf("Prepare the %s Libra report named %q for the local reporting window %s through %s. Use only the explicitly supplied sources. Organize the report into: %s. Name every concrete item plainly and cite each material claim with the exact supplied [[citation]]. State source coverage and evidence gaps when configured. Do not invent actions, meetings, deadlines, or people. This is a report, not an authorization.", report.Cadence, report.Name, window.StartDate, window.EndDate, strings.Join(labels, "; "))
	if strings.TrimSpace(report.Recipe.Instructions) != "" {
		prompt += " Additional report instructions from the person: " + strings.TrimSpace(report.Recipe.Instructions)
	}
	return prompt
}

func (d *daemon) processLibraReportInstance(ctx context.Context, schedule zodiacSchedule, instance zodiacInstance) (libraReportRun, error) {
	definitionID := strings.TrimSpace(schedule.SourceID)
	if definitionID == "" && len(schedule.Payload) > 0 {
		var payload struct {
			ReportDefinitionID string `json:"report_definition_id"`
		}
		_ = json.Unmarshal(schedule.Payload, &payload)
		definitionID = strings.TrimSpace(payload.ReportDefinitionID)
	}
	if definitionID == "" {
		return libraReportRun{}, errors.New("Libra schedule has no report definition link")
	}
	report, err := d.loadLibraReportDefinition(definitionID)
	if err != nil {
		return libraReportRun{}, err
	}
	window, err := libraReportWindowFor(report, instance.DueAt)
	if err != nil {
		return libraReportRun{}, err
	}
	return d.processLibraReportWithSources(ctx, report, schedule, instance, window, nil)
}

func (d *daemon) processLibraReportWithSources(ctx context.Context, report libraReportDefinition, schedule zodiacSchedule, instance zodiacInstance, window libraReportWindow, explicitRefs []string) (libraReportRun, error) {
	run, err := d.createLibraReportRun(report, schedule, instance, window)
	if err != nil {
		return run, err
	}
	if run.State == "completed" {
		return run, nil
	}
	if run.State == "completed_notification_failed" {
		if report.NotifyOnSuccess && run.FinalReportPath != "" {
			notificationID, notificationErr := d.deliverLibraNotification(ctx, report, run)
			run.NotificationOrderID, run.NotificationState, run.UpdatedAt = notificationID, "delivered", d.zodiacNow().Format(time.RFC3339Nano)
			if notificationErr != nil {
				run.NotificationState, run.Error = "failed", notificationErr.Error()
				d.audit("libra.report.notification", report.Name, "notify", "", "failed", run.RunID, notificationErr.Error(), 0)
			} else {
				run.State, run.Error = "completed", ""
				d.audit("libra.report.notification", report.Name, "notify", "", "completed", run.RunID, "", 0)
			}
			if err := d.saveLibraReportRun(run); err != nil {
				return run, err
			}
		}
		return run, nil
	}
	fail := func(state string, failure error) (libraReportRun, error) {
		run.State, run.Error, run.UpdatedAt = state, failure.Error(), d.zodiacNow().Format(time.RFC3339Nano)
		_ = d.saveLibraReportRun(run)
		return run, failure
	}
	refs := explicitRefs
	if refs == nil {
		refs, err = d.autoReportRefsBetweenWithRecipe(window.Start, window.End, report.Recipe)
		if err != nil {
			return fail("failed", err)
		}
	}
	minimumSources := report.Recipe.MinimumSources
	if minimumSources < 1 {
		minimumSources = 1
	}
	if len(refs) < minimumSources {
		return fail("needs_you", libraNeedsEvidence)
	}
	kind := reportAssemblyDaily
	period := window.EndDate
	if report.Cadence == "weekly" {
		kind, period = reportAssemblyWeekly, window.StartDate
	} else if report.Cadence == libraCadenceEvent {
		kind = reportAssemblyEvent
	}
	assembly, err := d.assembleReport(reportAssemblyRequest{Kind: kind, PeriodDate: period, SourceRefs: refs})
	if err != nil {
		return fail("failed", err)
	}
	run.AssemblyID, run.UpdatedAt = assembly.AssemblyID, d.zodiacNow().Format(time.RFC3339Nano)
	if err := d.saveLibraReportRun(run); err != nil {
		return run, err
	}
	absoluteRefs := []string{assembly.ManifestPath}
	for _, ref := range refs {
		absolute, _, pathErr := d.reportAssemblySourcePath(ref)
		if pathErr != nil {
			return fail("failed", pathErr)
		}
		absoluteRefs = append(absoluteRefs, absolute)
	}
	requestText := libraReportPrompt(report, window)
	payload, _ := json.Marshal(apolloOrderPayload{SourceRefs: absoluteRefs})
	order, err := d.createOrder(orderRequest{RequestText: requestText, Recipient: "apollo", Capability: apolloCapability, Scope: "report:libra:" + run.RunID, Payload: payload, Purpose: "Libra report reasoning"}, true)
	if err != nil {
		return fail("failed", err)
	}
	run.ApolloOrderID, run.UpdatedAt = order.OrderID, d.zodiacNow().Format(time.RFC3339Nano)
	if err := d.saveLibraReportRun(run); err != nil {
		return run, err
	}
	if err := d.dispatchLibraOrder(ctx, order); err != nil {
		return fail("failed", err)
	}
	apolloResult, err := d.loadApolloRunResult(order.OrderID)
	if err != nil {
		return fail("failed", err)
	}
	run.ApolloReportID, run.ApolloReportPath = apolloResult.ReportID, apolloResult.ReportPath
	run.UpdatedAt = d.zodiacNow().Format(time.RFC3339Nano)
	if err := d.saveLibraReportRun(run); err != nil {
		return run, err
	}
	if apolloResult.Status != "complete" {
		return fail("needs_you", fmt.Errorf("Apollo did not complete this report: %s", apolloResult.Status))
	}
	finalPath, err := d.writeLibraReport(run, report, assembly, apolloResult.ReportPath)
	if err != nil {
		return fail("failed", err)
	}
	run.Outputs, err = d.listLibraReportOutputs(run.RunID)
	if err != nil {
		return fail("failed", err)
	}
	run.FinalReportPath, run.State, run.UpdatedAt = finalPath, "completed", d.zodiacNow().Format(time.RFC3339Nano)
	if report.NotifyOnSuccess {
		notificationID, notificationErr := d.deliverLibraNotification(ctx, report, run)
		run.NotificationOrderID, run.NotificationState = notificationID, "delivered"
		if notificationErr != nil {
			run.State, run.Error, run.NotificationState = "completed_notification_failed", notificationErr.Error(), "failed"
			d.audit("libra.report.notification", report.Name, "notify", "", "failed", run.RunID, notificationErr.Error(), 0)
		} else {
			d.audit("libra.report.notification", report.Name, "notify", "", "completed", run.RunID, "", 0)
		}
	} else {
		run.NotificationState = "not_requested"
	}
	if err := d.saveLibraReportRun(run); err != nil {
		return run, err
	}
	return run, nil
}

func (d *daemon) loadBackfillReportDefinition(id string) (libraReportDefinition, error) {
	if strings.TrimSpace(id) != "" {
		return d.loadLibraReportDefinition(strings.TrimSpace(id))
	}
	rows, err := d.db.Query(`SELECT id FROM zodiac_report_definitions WHERE enabled = 1 AND cadence = 'daily' ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return libraReportDefinition{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return libraReportDefinition{}, err
		}
		return libraReportDefinition{}, errors.New("historical backfill requires an enabled daily Libra report definition")
	}
	var definitionID string
	if err := rows.Scan(&definitionID); err != nil {
		return libraReportDefinition{}, err
	}
	return d.loadLibraReportDefinition(definitionID)
}

func libraBackfillDateWindow(report libraReportDefinition, startDate, endDate string) (libraReportWindow, error) {
	location, err := time.LoadLocation(report.Timezone)
	if err != nil {
		return libraReportWindow{}, err
	}
	start, err := time.ParseInLocation("2006-01-02", startDate, location)
	if err != nil {
		return libraReportWindow{}, errors.New("start_date must use YYYY-MM-DD")
	}
	end, err := time.ParseInLocation("2006-01-02", endDate, location)
	if err != nil {
		return libraReportWindow{}, errors.New("end_date must use YYYY-MM-DD")
	}
	if end.Before(start) {
		return libraReportWindow{}, errors.New("end_date must not precede start_date")
	}
	return libraReportWindow{Key: "historical:" + startDate + ".." + endDate, StartDate: startDate, EndDate: endDate, Start: start, End: end}, nil
}

func (d *daemon) splitLibraBackfillRefs(refs []string, maxBytes int) ([][]string, error) {
	if maxBytes < 8000 {
		maxBytes = 24000
	}
	var batches [][]string
	var current []string
	currentBytes := 0
	for _, ref := range refs {
		absolute, _, err := d.reportAssemblySourcePath(ref)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(absolute)
		if err != nil {
			return nil, err
		}
		size := int(info.Size()) + 160
		if len(current) > 0 && currentBytes+size > maxBytes {
			batches = append(batches, current)
			current = nil
			currentBytes = 0
		}
		current = append(current, ref)
		currentBytes += size
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}
	return batches, nil
}

func (d *daemon) processLibraHistoricalBackfill(ctx context.Context, request libraBackfillRequest) (libraBackfillResult, error) {
	report, err := d.loadBackfillReportDefinition(request.ReportDefinitionID)
	if err != nil {
		return libraBackfillResult{}, err
	}
	window, err := libraBackfillDateWindow(report, strings.TrimSpace(request.StartDate), strings.TrimSpace(request.EndDate))
	if err != nil {
		return libraBackfillResult{}, err
	}
	refs, err := d.autoReportRefsBetweenWithRecipe(window.Start, window.End, report.Recipe)
	if err != nil {
		return libraBackfillResult{}, err
	}
	if len(refs) == 0 {
		return libraBackfillResult{}, libraNeedsEvidence
	}
	batches, err := d.splitLibraBackfillRefs(refs, request.BatchBytes)
	if err != nil {
		return libraBackfillResult{}, err
	}
	result := libraBackfillResult{ReportDefinitionID: report.ID, StartDate: window.StartDate, EndDate: window.EndDate, SourceCount: len(refs), BatchCount: len(batches), BatchRuns: []libraReportRun{}}
	batchRefs := make([]string, 0, len(batches))
	for index, batch := range batches {
		batchReport := report
		batchReport.Name = fmt.Sprintf("Historical evidence batch %d of %d", index+1, len(batches))
		batchReport.NotifyOnSuccess = false
		batchReport.Recipe.Outputs = []string{libraOutputIntegrated}
		batchReport.Recipe.Instructions = strings.TrimSpace(strings.Join([]string{
			"This is an intermediate evidence digest for a historical backfill. Extract named people only when the sources name them; distinguish named people from anonymous speakers and people merely mentioned. Capture plain-language facts, requests, commitments, decisions, relationships, and uncertainties with citations. Do not produce code, JSON, or raw transcript dumps.",
			report.Recipe.Instructions,
		}, " "))
		batchWindow := window
		batchWindow.Key = fmt.Sprintf("historical:%s..%s:batch-%03d", window.StartDate, window.EndDate, index+1)
		schedule := zodiacSchedule{ID: "libra-backfill-" + sha256Bytes([]byte(report.ID + "\x00" + batchWindow.Key))[:24], Owner: "libra", SourceID: report.ID, Action: zodiacActionBriefGenerate}
		instance := zodiacInstance{ID: "libra-backfill-" + sha256Bytes([]byte(schedule.ID + "\x00instance"))[:24], DueAt: window.End.Format(time.RFC3339Nano)}
		run, runErr := d.processLibraReportWithSources(ctx, batchReport, schedule, instance, batchWindow, batch)
		if runErr != nil {
			return result, runErr
		}
		result.BatchRuns = append(result.BatchRuns, run)
		if run.State != "completed" || strings.TrimSpace(run.ApolloReportPath) == "" {
			return result, fmt.Errorf("historical evidence batch %d did not complete: %s", index+1, run.State)
		}
		_, relative, pathErr := d.reportAssemblySourcePath(run.ApolloReportPath)
		if pathErr != nil {
			return result, pathErr
		}
		batchRefs = append(batchRefs, relative)
	}

	finalReport := report
	finalReport.Name = "Historical Conversation Backfill"
	finalReport.NotifyOnSuccess = false
	finalReport.Recipe.Outputs = []string{libraOutputSituation}
	finalReport.Recipe.Instructions = strings.TrimSpace(strings.Join([]string{
		"This is the final human-readable historical baseline assembled from intermediate evidence digests. Consolidate duplicate people carefully. Include a clear People and identity context section, explicitly distinguishing named people, people mentioned, anonymous speakers, and uncertain identity. Then summarize facts, themes, requests, commitments, decisions, unresolved questions, and evidence gaps. Use plain prose and exact source citations. Do not output JSON, code, or a transcript dump.",
		report.Recipe.Instructions,
	}, " "))
	finalWindow := window
	finalWindow.Key = "historical:" + window.StartDate + ".." + window.EndDate + ":final"
	finalSchedule := zodiacSchedule{ID: "libra-backfill-final-" + sha256Bytes([]byte(report.ID + "\x00" + finalWindow.Key))[:24], Owner: "libra", SourceID: report.ID, Action: zodiacActionBriefGenerate}
	finalInstance := zodiacInstance{ID: "libra-backfill-final-" + sha256Bytes([]byte(finalSchedule.ID + "\x00instance"))[:24], DueAt: window.End.Format(time.RFC3339Nano)}
	finalRun, err := d.processLibraReportWithSources(ctx, finalReport, finalSchedule, finalInstance, finalWindow, batchRefs)
	if err != nil {
		return result, err
	}
	result.FinalRun = finalRun
	return result, nil
}

func libraEventWindow(report libraReportDefinition, occurredAt, eventKey string) (libraReportWindow, error) {
	location, err := time.LoadLocation(report.Timezone)
	if err != nil {
		return libraReportWindow{}, err
	}
	when, err := time.Parse(time.RFC3339Nano, occurredAt)
	if err != nil {
		return libraReportWindow{}, err
	}
	when = when.In(location)
	day := time.Date(when.Year(), when.Month(), when.Day(), 0, 0, 0, 0, location)
	date := day.Format("2006-01-02")
	return libraReportWindow{Key: "event:" + sha256Bytes([]byte(report.ID + "\x00" + eventKey + "\x00" + occurredAt))[:24], StartDate: date, EndDate: date, Start: day, End: day}, nil
}

func (d *daemon) enqueueLibraIngestionJobs(receipt externalIntakeReceipt) error {
	if receipt.Status != "completed" || strings.TrimSpace(receipt.NormalizedPath) == "" {
		return nil
	}
	rows, err := d.db.Query(`SELECT id FROM zodiac_report_definitions WHERE enabled = 1 AND COALESCE(recipe_json, '{}') LIKE '%on_ingestion%'`)
	if err != nil {
		return err
	}
	definitionIDs := []string{}
	for rows.Next() {
		var definitionID string
		if err := rows.Scan(&definitionID); err != nil {
			rows.Close()
			return err
		}
		definitionIDs = append(definitionIDs, definitionID)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	now := d.zodiacNow().Format(time.RFC3339Nano)
	for _, definitionID := range definitionIDs {
		jobID := "libra-job-" + sha256Bytes([]byte(definitionID + "\x00" + receipt.SourceKind + "\x00" + receipt.SourceID))[:24]
		if _, err := d.db.Exec(`INSERT INTO libra_ingestion_jobs (job_id, report_definition_id, source_kind, source_id, receipt_id, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'queued', ?, ?) ON CONFLICT(report_definition_id, source_kind, source_id) DO NOTHING`, jobID, definitionID, receipt.SourceKind, receipt.SourceID, receipt.ReceiptID, now, now); err != nil {
			return err
		}
	}
	return nil
}

func (d *daemon) processLibraIngestionQueue(ctx context.Context, limit int) error {
	if limit < 1 {
		limit = 4
	}
	if paused, err := d.paused(); err != nil {
		return err
	} else if paused {
		return nil
	}
	rows, err := d.db.Query(`SELECT job_id, report_definition_id, source_kind, source_id, receipt_id FROM libra_ingestion_jobs WHERE state = 'queued' ORDER BY created_at LIMIT ?`, limit)
	if err != nil {
		return err
	}
	type job struct{ ID, DefinitionID, SourceKind, SourceID, ReceiptID string }
	jobs := []job{}
	for rows.Next() {
		var item job
		if err := rows.Scan(&item.ID, &item.DefinitionID, &item.SourceKind, &item.SourceID, &item.ReceiptID); err != nil {
			rows.Close()
			return err
		}
		jobs = append(jobs, item)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, item := range jobs {
		mark := func(state string, jobErr error) error {
			message := ""
			if jobErr != nil {
				message = jobErr.Error()
			}
			_, err := d.db.Exec(`UPDATE libra_ingestion_jobs SET state = ?, error = ?, updated_at = ? WHERE job_id = ?`, state, nullableString(message), d.zodiacNow().Format(time.RFC3339Nano), item.ID)
			return err
		}
		report, reportErr := d.loadLibraReportDefinition(item.DefinitionID)
		if reportErr != nil {
			if err := mark("failed", reportErr); err != nil {
				return err
			}
			continue
		}
		if !report.Enabled || report.Recipe.Trigger != libraTriggerOnIngestion {
			if err := mark("skipped", nil); err != nil {
				return err
			}
			continue
		}
		receipt, receiptErr := d.loadExternalIntakeReceipt(item.SourceKind, item.SourceID)
		if receiptErr != nil {
			if err := mark("failed", receiptErr); err != nil {
				return err
			}
			continue
		}
		_, relative, pathErr := d.reportAssemblySourcePath(receipt.NormalizedPath)
		if pathErr != nil || !libraRecipeAllowsPath(report.Recipe, relative) {
			if pathErr == nil {
				pathErr = errors.New("ingested source is excluded by the Libra source recipe")
			}
			if err := mark("skipped", pathErr); err != nil {
				return err
			}
			continue
		}
		window, windowErr := libraEventWindow(report, receipt.OccurredAt, item.SourceKind+"\x00"+item.SourceID)
		if windowErr != nil {
			if err := mark("failed", windowErr); err != nil {
				return err
			}
			continue
		}
		schedule := zodiacSchedule{ID: "libra-ingestion-" + report.ID, Owner: "libra", SourceID: report.ID, Action: zodiacActionBriefGenerate}
		instance := zodiacInstance{ID: "libra-ingestion-" + sha256Bytes([]byte(report.ID + "\x00" + item.SourceKind + "\x00" + item.SourceID))[:24], DueAt: receipt.OccurredAt}
		_, processErr := d.processLibraReportWithSources(ctx, report, schedule, instance, window, []string{relative})
		if processErr != nil {
			if err := mark("failed", processErr); err != nil {
				return err
			}
			continue
		}
		if err := mark("completed", nil); err != nil {
			return err
		}
	}
	return nil
}

func (d *daemon) handleListZodiacConstellations(w http.ResponseWriter, r *http.Request) {
	items, err := d.listZodiacConstellations()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"constellations": items, "note": "Constellations describe reusable policies; report definitions decide which schedules are active."})
}

func (d *daemon) handleCreateZodiacConstellation(w http.ResponseWriter, r *http.Request) {
	if !d.guiControlRequired(w, r) {
		return
	}
	var request zodiacConstellationRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := d.createZodiacConstellation(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (d *daemon) handleListLibraReportDefinitions(w http.ResponseWriter, r *http.Request) {
	items, err := d.listLibraReportDefinitions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": items, "note": "Libra definitions configure bounded report runs. Zodiac records due instances; each completed run exposes assembly, Apollo, Vault, and optional notification receipts."})
}

func (d *daemon) handleListLibraReportRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := d.listLibraReportRuns(50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs, "note": "A completed run proves assembly, Apollo, Vault, and requested notification stages recorded their receipts."})
}

func (d *daemon) handleLibraHistoricalBackfill(w http.ResponseWriter, r *http.Request) {
	if !d.guiControlRequired(w, r) {
		return
	}
	var request libraBackfillRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := d.processLibraHistoricalBackfill(r.Context(), request)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, libraNeedsEvidence) || strings.Contains(err.Error(), "blocked_model_unavailable") {
			status = http.StatusBadGateway
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (d *daemon) handleCreateLibraReportDefinition(w http.ResponseWriter, r *http.Request) {
	if !d.guiControlRequired(w, r) {
		return
	}
	var request libraReportDefinitionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := d.createLibraReportDefinition(request)
	if err != nil {
		status := http.StatusBadRequest
		if !errors.Is(err, sql.ErrNoRows) {
			writeError(w, status, err)
		} else {
			writeError(w, http.StatusNotFound, errors.New("Libra constellation not found"))
		}
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (d *daemon) handleSetLibraReportEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	if !d.guiControlRequired(w, r) {
		return
	}
	item, err := d.setLibraReportEnabled(r.PathValue("id"), enabled)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("Libra report definition not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
