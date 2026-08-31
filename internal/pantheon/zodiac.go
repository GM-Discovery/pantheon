package pantheon

// Zodiac is Pantheon's deterministic schedule keeper. It stores explicit
// temporal declarations and due instances; it does not interpret language or
// create authority. Consequential actions become Future proposals and follow
// the existing Future -> Mercury -> adapter path.

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	zodiacScheduleActive   = "active"
	zodiacScheduleDisabled = "disabled"

	zodiacInstancePending           = "pending"
	zodiacInstanceDue               = "due"
	zodiacInstanceWaiting           = "waiting"
	zodiacInstanceDispatched        = "dispatched"
	zodiacInstanceCompleted         = "completed"
	zodiacInstanceFailed            = "failed"
	zodiacInstanceSkipped           = "skipped"
	zodiacInstanceBlockedPrometheus = "blocked_prometheus"
	zodiacInstanceNeedsYou          = "needs_you"
	zodiacInstanceAwaitingAuthority = "awaiting_authority"
	zodiacInstanceAwaitingPipeline  = "awaiting_pipeline"

	zodiacCatchUpSkip      = "skip"
	zodiacCatchUpRunOnce   = "run_once"
	zodiacCatchUpReconcile = "reconcile"
	zodiacCatchUpNeedsYou  = "needs_you"

	zodiacPrecisionExact   = "exact"
	zodiacPrecisionDate    = "date"
	zodiacPrecisionDaypart = "daypart"

	zodiacProjectionNone    = "none"
	zodiacProjectionApple   = "apple"
	zodiacProjectionGoogle  = "google"
	zodiacProjectionBoth    = "both"
	zodiacProjectionUnknown = "unknown"

	zodiacActionMailRound        = "mail.round"
	zodiacActionBriefGenerate    = "brief.generate"
	zodiacActionRumorProcess     = "rumor.process"
	zodiacActionNotifyEvaluate   = "notification.evaluate"
	zodiacActionCalendarReminder = "calendar.reminder"
)

type zodiacSchedule struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Kind                 string          `json:"kind"`
	Owner                string          `json:"owner"`
	Source               string          `json:"source"`
	SourceID             string          `json:"source_id,omitempty"`
	Enabled              bool            `json:"enabled"`
	Status               string          `json:"status"`
	Timezone             string          `json:"timezone"`
	StartAt              string          `json:"start_at"`
	EndAt                string          `json:"end_at,omitempty"`
	Recurrence           string          `json:"recurrence,omitempty"`
	Deadline             string          `json:"deadline,omitempty"`
	Precision            string          `json:"precision"`
	OriginalWording      string          `json:"original_wording,omitempty"`
	Daypart              string          `json:"daypart,omitempty"`
	CatchUpPolicy        string          `json:"catch_up_policy"`
	ProjectionPolicy     string          `json:"projection_policy"`
	Priority             string          `json:"priority"`
	AuthorityRequirement string          `json:"authority_requirement,omitempty"`
	Dependency           json.RawMessage `json:"dependency,omitempty"`
	Action               string          `json:"action"`
	Payload              json.RawMessage `json:"payload,omitempty"`
	AttestationID        string          `json:"attestation_id,omitempty"`
	NextDueAt            string          `json:"next_due_at,omitempty"`
	LastDueAt            string          `json:"last_due_at,omitempty"`
	CreatedAt            string          `json:"created_at"`
	UpdatedAt            string          `json:"updated_at"`
}

type zodiacScheduleRequest struct {
	ID                   string          `json:"id,omitempty"`
	Name                 string          `json:"name"`
	Kind                 string          `json:"kind"`
	Owner                string          `json:"owner,omitempty"`
	Source               string          `json:"source,omitempty"`
	SourceID             string          `json:"source_id,omitempty"`
	Enabled              *bool           `json:"enabled,omitempty"`
	Timezone             string          `json:"timezone,omitempty"`
	StartAt              string          `json:"start_at"`
	EndAt                string          `json:"end_at,omitempty"`
	Recurrence           string          `json:"recurrence,omitempty"`
	Deadline             string          `json:"deadline,omitempty"`
	Precision            string          `json:"precision,omitempty"`
	OriginalWording      string          `json:"original_wording,omitempty"`
	Daypart              string          `json:"daypart,omitempty"`
	CatchUpPolicy        string          `json:"catch_up_policy,omitempty"`
	ProjectionPolicy     string          `json:"projection_policy,omitempty"`
	Priority             string          `json:"priority,omitempty"`
	AuthorityRequirement string          `json:"authority_requirement,omitempty"`
	Dependency           json.RawMessage `json:"dependency,omitempty"`
	Action               string          `json:"action"`
	Payload              json.RawMessage `json:"payload,omitempty"`
	AttestationID        string          `json:"attestation_id,omitempty"`
}

type zodiacInstance struct {
	ID               string `json:"instance_id"`
	ScheduleID       string `json:"schedule_id"`
	OccurrenceKey    string `json:"occurrence_key"`
	DueAt            string `json:"due_at"`
	Precision        string `json:"precision"`
	State            string `json:"state"`
	CatchUpDecision  string `json:"catch_up_decision,omitempty"`
	FutureProposalID string `json:"future_proposal_id,omitempty"`
	AuthorityState   string `json:"authority_state,omitempty"`
	ProjectionState  string `json:"projection_state,omitempty"`
	LastError        string `json:"last_error,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	CompletedAt      string `json:"completed_at,omitempty"`
}

type zodiacProjection struct {
	ID                       string `json:"projection_id"`
	ScheduleID               string `json:"schedule_id"`
	InstanceID               string `json:"instance_id,omitempty"`
	Provider                 string `json:"provider"`
	Account                  string `json:"account"`
	Calendar                 string `json:"calendar"`
	ExternalEventID          string `json:"external_event_id,omitempty"`
	ProjectionRole           string `json:"projection_role"`
	CreatedAt                string `json:"created_at"`
	LastSyncedAt             string `json:"last_synced_at,omitempty"`
	LastSeenExternalRevision string `json:"last_seen_external_revision,omitempty"`
	Status                   string `json:"status"`
	Correction               string `json:"correction,omitempty"`
}

type zodiacAttestation struct {
	ID                string          `json:"attestation_id"`
	SourceID          string          `json:"source_id"`
	SourcePath        string          `json:"source_path,omitempty"`
	SourceFingerprint string          `json:"source_fingerprint,omitempty"`
	Speaker           string          `json:"speaker,omitempty"`
	CitationStart     string          `json:"citation_start"`
	CitationEnd       string          `json:"citation_end"`
	QuotedSpan        string          `json:"quoted_span"`
	Normalized        json.RawMessage `json:"normalized"`
	Confidence        float64         `json:"confidence"`
	Result            string          `json:"result"`
	Reason            string          `json:"reason,omitempty"`
	CreatedAt         string          `json:"created_at"`
}

type zodiacProviderHealth struct {
	Provider     string   `json:"provider"`
	State        string   `json:"state"`
	Reason       string   `json:"reason,omitempty"`
	CapabilityID string   `json:"capability_id"`
	Operations   []string `json:"operations"`
	LastSyncedAt string   `json:"last_synced_at,omitempty"`
	Credentials  string   `json:"credentials"`
}

type zodiacProjectionReconcileRequest struct {
	ExternalEventID  string `json:"external_event_id"`
	ExternalRevision string `json:"external_revision,omitempty"`
	StartAt          string `json:"start_at,omitempty"`
	EndAt            string `json:"end_at,omitempty"`
	Title            string `json:"title,omitempty"`
	Deleted          bool   `json:"deleted,omitempty"`
	HumanModified    bool   `json:"human_modified"`
	Correction       string `json:"correction,omitempty"`
}

type zodiacInspection struct {
	Enabled     bool                   `json:"enabled"`
	Now         string                 `json:"now"`
	Timezone    string                 `json:"timezone"`
	Paused      bool                   `json:"paused"`
	Schedules   []zodiacSchedule       `json:"schedules"`
	Instances   []zodiacInstance       `json:"instances"`
	Projections []zodiacProjection     `json:"projections"`
	NeedsYou    []zodiacInstance       `json:"needs_you"`
	Providers   []zodiacProviderHealth `json:"providers"`
	Recurring   []zodiacSchedule       `json:"recurring"`
	Contract    map[string]any         `json:"contract"`
}

type zodiacRoundResponse struct {
	StartedAt  string   `json:"started_at"`
	ObservedAt string   `json:"observed_at"`
	Paused     bool     `json:"paused"`
	Examined   int      `json:"examined"`
	Due        int      `json:"due"`
	Created    int      `json:"created"`
	Skipped    int      `json:"skipped"`
	Blocked    int      `json:"blocked"`
	Proposals  int      `json:"proposals"`
	Failed     int      `json:"failed"`
	Errors     []string `json:"errors,omitempty"`
	Result     string   `json:"result"`
}

type zodiacRecurrence struct {
	Frequency string
	Interval  int
	ByDay     map[time.Weekday]bool
	Count     int
	Until     time.Time
}

func initializeZodiacSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS zodiac_schedules (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, kind TEXT NOT NULL, owner TEXT NOT NULL,
			source TEXT NOT NULL, source_id TEXT, enabled INTEGER NOT NULL, status TEXT NOT NULL,
			timezone TEXT NOT NULL, start_at TEXT NOT NULL, end_at TEXT, recurrence TEXT,
			deadline TEXT, precision TEXT NOT NULL, original_wording TEXT, daypart TEXT,
			catch_up_policy TEXT NOT NULL, projection_policy TEXT NOT NULL, priority TEXT NOT NULL,
			authority_requirement TEXT, dependency_json TEXT, action TEXT NOT NULL, payload_json TEXT,
			attestation_id TEXT, next_due_at TEXT, last_due_at TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS zodiac_schedules_due_idx ON zodiac_schedules(enabled, next_due_at, status)`,
		`CREATE TABLE IF NOT EXISTS zodiac_instances (
			instance_id TEXT PRIMARY KEY, schedule_id TEXT NOT NULL, occurrence_key TEXT NOT NULL,
			due_at TEXT NOT NULL, precision TEXT NOT NULL, state TEXT NOT NULL, catch_up_decision TEXT,
			future_proposal_id TEXT, authority_state TEXT, projection_state TEXT, last_error TEXT,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL, completed_at TEXT,
			UNIQUE(schedule_id, occurrence_key)
		)`,
		`CREATE INDEX IF NOT EXISTS zodiac_instances_state_idx ON zodiac_instances(state, due_at)`,
		`CREATE TABLE IF NOT EXISTS zodiac_projections (
			projection_id TEXT PRIMARY KEY, schedule_id TEXT NOT NULL, instance_id TEXT,
			provider TEXT NOT NULL, account TEXT NOT NULL, calendar TEXT NOT NULL, external_event_id TEXT,
			projection_role TEXT NOT NULL, created_at TEXT NOT NULL, last_synced_at TEXT,
			last_seen_external_revision TEXT, status TEXT NOT NULL, correction TEXT
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS zodiac_projection_external_idx ON zodiac_projections(provider, external_event_id) WHERE external_event_id <> ''`,
		`CREATE TABLE IF NOT EXISTS zodiac_revisions (
			revision_id TEXT PRIMARY KEY, schedule_id TEXT NOT NULL, previous_json TEXT NOT NULL,
			new_json TEXT NOT NULL, source TEXT NOT NULL, reason TEXT, occurred_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS zodiac_attestations (
			attestation_id TEXT PRIMARY KEY, source_id TEXT NOT NULL, source_path TEXT,
			source_fingerprint TEXT, speaker TEXT, citation_start TEXT NOT NULL, citation_end TEXT NOT NULL,
			quoted_span TEXT NOT NULL, normalized_json TEXT NOT NULL, confidence REAL NOT NULL,
			result TEXT NOT NULL, reason TEXT, created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS zodiac_rounds (
			round_id TEXT PRIMARY KEY, started_at TEXT NOT NULL, observed_at TEXT NOT NULL,
			paused INTEGER NOT NULL, result_json TEXT NOT NULL, error TEXT
		)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func (d *daemon) zodiacLocation() *time.Location {
	zone := strings.TrimSpace(d.cfg.ZodiacDefaultTimezone)
	if zone == "" {
		zone = "America/Los_Angeles"
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return time.FixedZone("zodiac-fallback", 0)
	}
	return location
}

func (d *daemon) zodiacNow() time.Time {
	if d.zodiacClock != nil {
		return d.zodiacClock().In(d.zodiacLocation())
	}
	return time.Now().In(d.zodiacLocation())
}

func zoneOffset(value time.Time) string {
	_, seconds := value.Zone()
	sign := "+"
	if seconds < 0 {
		sign, seconds = "-", -seconds
	}
	return fmt.Sprintf("%s%02d:%02d", sign, seconds/3600, (seconds%3600)/60)
}

func parseZodiacTime(value, timezone string) (time.Time, string, error) {
	value = strings.TrimSpace(value)
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, "", err
	}
	if len(value) == len("2006-01-02") {
		parsed, err := time.ParseInLocation("2006-01-02", value, location)
		return parsed, zodiacPrecisionDate, err
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.In(location), zodiacPrecisionExact, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02T15:04", value, location)
	return parsed, zodiacPrecisionExact, err
}

func zodiacWeekday(value string) (time.Weekday, bool) {
	weekdays := map[string]time.Weekday{"SU": time.Sunday, "MO": time.Monday, "TU": time.Tuesday, "WE": time.Wednesday, "TH": time.Thursday, "FR": time.Friday, "SA": time.Saturday}
	day, ok := weekdays[value]
	return day, ok
}

func parseZodiacRecurrence(value string) (zodiacRecurrence, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return zodiacRecurrence{}, nil
	}
	if !strings.HasPrefix(strings.ToUpper(value), "RRULE:") {
		switch strings.ToLower(value) {
		case "hourly":
			value = "RRULE:FREQ=HOURLY"
		case "daily":
			value = "RRULE:FREQ=DAILY"
		case "weekdays":
			value = "RRULE:FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"
		case "weekly":
			value = "RRULE:FREQ=WEEKLY"
		default:
			return zodiacRecurrence{}, fmt.Errorf("recurrence must be an iCalendar RRULE or supported shorthand")
		}
	}
	rule := zodiacRecurrence{Interval: 1}
	for _, part := range strings.Split(strings.TrimPrefix(strings.ToUpper(value), "RRULE:"), ";") {
		pieces := strings.SplitN(part, "=", 2)
		if len(pieces) != 2 {
			return zodiacRecurrence{}, fmt.Errorf("malformed recurrence component %q", part)
		}
		switch pieces[0] {
		case "FREQ":
			rule.Frequency = pieces[1]
		case "INTERVAL":
			parsed, err := strconv.Atoi(pieces[1])
			if err != nil || parsed < 1 {
				return zodiacRecurrence{}, errors.New("recurrence interval must be positive")
			}
			rule.Interval = parsed
		case "COUNT":
			parsed, err := strconv.Atoi(pieces[1])
			if err != nil || parsed < 1 {
				return zodiacRecurrence{}, errors.New("recurrence count must be positive")
			}
			rule.Count = parsed
		case "UNTIL":
			parsed, err := time.Parse(time.RFC3339, pieces[1])
			if err != nil {
				return zodiacRecurrence{}, fmt.Errorf("recurrence UNTIL must be RFC3339: %w", err)
			}
			rule.Until = parsed
		case "BYDAY":
			rule.ByDay = map[time.Weekday]bool{}
			for _, day := range strings.Split(pieces[1], ",") {
				weekday, ok := zodiacWeekday(day)
				if !ok {
					return zodiacRecurrence{}, fmt.Errorf("unsupported BYDAY %q", day)
				}
				rule.ByDay[weekday] = true
			}
		default:
			return zodiacRecurrence{}, fmt.Errorf("unsupported recurrence component %q", pieces[0])
		}
	}
	if rule.Frequency != "HOURLY" && rule.Frequency != "DAILY" && rule.Frequency != "WEEKLY" && rule.Frequency != "MONTHLY" && rule.Frequency != "YEARLY" {
		return zodiacRecurrence{}, fmt.Errorf("unsupported recurrence frequency %q", rule.Frequency)
	}
	return rule, nil
}

func validateZodiacScheduleRequest(d *daemon, request *zodiacScheduleRequest) (time.Time, error) {
	request.Name, request.Kind, request.Action = strings.TrimSpace(request.Name), strings.TrimSpace(request.Kind), strings.TrimSpace(request.Action)
	if request.Name == "" || request.Kind == "" || request.Action == "" || strings.TrimSpace(request.StartAt) == "" {
		return time.Time{}, errors.New("name, kind, start_at, and action are required")
	}
	validKinds := map[string]bool{"one_time_event": true, "one_time_reminder": true, "deadline": true, "recurring_event": true, "recurring_system_job": true, "conditional_reminder": true, "follow_up": true, "dependency": true, "date_only_obligation": true, "daypart_obligation": true}
	if !validKinds[request.Kind] {
		return time.Time{}, fmt.Errorf("unsupported schedule kind %q", request.Kind)
	}
	if request.Timezone == "" {
		request.Timezone = d.cfg.ZodiacDefaultTimezone
	}
	if _, err := time.LoadLocation(request.Timezone); err != nil {
		return time.Time{}, fmt.Errorf("timezone: %w", err)
	}
	start, precision, err := parseZodiacTime(request.StartAt, request.Timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("start_at: %w", err)
	}
	if request.Precision == "" {
		request.Precision = precision
	}
	if request.Precision != zodiacPrecisionExact && request.Precision != zodiacPrecisionDate && request.Precision != zodiacPrecisionDaypart {
		return time.Time{}, errors.New("precision must be exact, date, or daypart")
	}
	if request.CatchUpPolicy == "" {
		request.CatchUpPolicy = zodiacCatchUpSkip
	}
	if request.CatchUpPolicy != zodiacCatchUpSkip && request.CatchUpPolicy != zodiacCatchUpRunOnce && request.CatchUpPolicy != zodiacCatchUpReconcile && request.CatchUpPolicy != zodiacCatchUpNeedsYou {
		return time.Time{}, errors.New("unsupported catch_up_policy")
	}
	if request.ProjectionPolicy == "" {
		request.ProjectionPolicy = zodiacProjectionNone
	}
	if request.ProjectionPolicy != zodiacProjectionNone && request.ProjectionPolicy != zodiacProjectionApple && request.ProjectionPolicy != zodiacProjectionGoogle && request.ProjectionPolicy != zodiacProjectionBoth && request.ProjectionPolicy != zodiacProjectionUnknown {
		return time.Time{}, errors.New("unsupported projection_policy")
	}
	if request.Priority == "" {
		request.Priority = "normal"
	}
	if request.Owner == "" {
		request.Owner = "zodiac"
	}
	if request.Source == "" {
		request.Source = "human_gui"
	}
	if request.Enabled == nil {
		enabled := true
		request.Enabled = &enabled
	}
	if request.Recurrence != "" {
		if _, err := parseZodiacRecurrence(request.Recurrence); err != nil {
			return time.Time{}, err
		}
	}
	if request.Dependency != nil && !json.Valid(request.Dependency) {
		return time.Time{}, errors.New("dependency must be valid JSON")
	}
	if request.Payload != nil && !json.Valid(request.Payload) {
		return time.Time{}, errors.New("payload must be valid JSON")
	}
	validActions := map[string]bool{zodiacActionMailRound: true, zodiacActionBriefGenerate: true, zodiacActionRumorProcess: true, zodiacActionNotifyEvaluate: true, zodiacActionCalendarReminder: true}
	if !validActions[request.Action] {
		return time.Time{}, fmt.Errorf("unsupported bounded schedule action %q", request.Action)
	}
	if request.EndAt != "" {
		if _, _, err := parseZodiacTime(request.EndAt, request.Timezone); err != nil {
			return time.Time{}, fmt.Errorf("end_at: %w", err)
		}
	}
	return start, nil
}

func (d *daemon) ensureZodiacDefaults() error {
	if err := initializeZodiacSchema(d.db); err != nil {
		return err
	}
	now := d.zodiacNow()
	briefStart := now.Format("2006-01-02") + "T" + d.cfg.ZodiacBriefTime + ":00" + zoneOffset(now)
	defaults := []zodiacScheduleRequest{
		{ID: "zodiac-mail-hourly", Name: "Mail Intake", Kind: "recurring_system_job", Owner: "zodiac", Source: "configuration", StartAt: now.Format(time.RFC3339), Recurrence: "RRULE:FREQ=HOURLY;INTERVAL=1", Precision: zodiacPrecisionExact, CatchUpPolicy: zodiacCatchUpReconcile, ProjectionPolicy: zodiacProjectionNone, Priority: "normal", Action: zodiacActionMailRound},
		{ID: "zodiac-situation-brief", Name: "Grant's Situation Brief", Kind: "recurring_system_job", Owner: "zodiac", Source: "configuration", StartAt: briefStart, Recurrence: "RRULE:FREQ=DAILY;INTERVAL=1", Precision: zodiacPrecisionExact, CatchUpPolicy: d.cfg.ZodiacBriefCatchUpPolicy, ProjectionPolicy: zodiacProjectionNone, Priority: "person_requested", Action: zodiacActionBriefGenerate},
	}
	for _, request := range defaults {
		var existing string
		err := d.db.QueryRow(`SELECT id FROM zodiac_schedules WHERE id = ?`, request.ID).Scan(&existing)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err := d.createZodiacSchedule(request); err != nil {
			return err
		}
	}
	return nil
}

func (d *daemon) createZodiacSchedule(request zodiacScheduleRequest) (zodiacSchedule, error) {
	start, err := validateZodiacScheduleRequest(d, &request)
	if err != nil {
		return zodiacSchedule{}, err
	}
	id := strings.TrimSpace(request.ID)
	if id == "" {
		id, err = newLocalID("zodiac-")
		if err != nil {
			return zodiacSchedule{}, err
		}
	}
	now := d.zodiacNow().Format(time.RFC3339Nano)
	status := zodiacScheduleDisabled
	if request.Enabled == nil || *request.Enabled {
		status = zodiacScheduleActive
	}
	schedule := zodiacSchedule{ID: id, Name: request.Name, Kind: request.Kind, Owner: request.Owner, Source: request.Source, SourceID: request.SourceID, Enabled: status == zodiacScheduleActive, Status: status, Timezone: request.Timezone, StartAt: request.StartAt, EndAt: request.EndAt, Recurrence: request.Recurrence, Deadline: request.Deadline, Precision: request.Precision, OriginalWording: request.OriginalWording, Daypart: request.Daypart, CatchUpPolicy: request.CatchUpPolicy, ProjectionPolicy: request.ProjectionPolicy, Priority: request.Priority, AuthorityRequirement: request.AuthorityRequirement, Dependency: request.Dependency, Action: request.Action, Payload: request.Payload, AttestationID: request.AttestationID, NextDueAt: start.Format(time.RFC3339Nano), CreatedAt: now, UpdatedAt: now}
	_, err = d.db.Exec(`INSERT INTO zodiac_schedules (id, name, kind, owner, source, source_id, enabled, status, timezone, start_at, end_at, recurrence, deadline, precision, original_wording, daypart, catch_up_policy, projection_policy, priority, authority_requirement, dependency_json, action, payload_json, attestation_id, next_due_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, schedule.ID, schedule.Name, schedule.Kind, schedule.Owner, schedule.Source, nullableString(schedule.SourceID), schedule.Enabled, schedule.Status, schedule.Timezone, schedule.StartAt, nullableString(schedule.EndAt), nullableString(schedule.Recurrence), nullableString(schedule.Deadline), schedule.Precision, nullableString(schedule.OriginalWording), nullableString(schedule.Daypart), schedule.CatchUpPolicy, schedule.ProjectionPolicy, schedule.Priority, nullableString(schedule.AuthorityRequirement), nullableJSON(schedule.Dependency), schedule.Action, nullableJSON(schedule.Payload), nullableString(schedule.AttestationID), schedule.NextDueAt, schedule.CreatedAt, schedule.UpdatedAt)
	if err != nil {
		return zodiacSchedule{}, err
	}
	if err := d.zodiacCreateProjectionRows(schedule); err != nil {
		return zodiacSchedule{}, err
	}
	return schedule, nil
}

func (d *daemon) loadZodiacSchedule(id string) (zodiacSchedule, error) {
	var item zodiacSchedule
	var enabled int
	var sourceID, endAt, recurrence, deadline, wording, daypart, authority, dependency, payload, attestation, nextDue, lastDue sql.NullString
	err := d.db.QueryRow(`SELECT id, name, kind, owner, source, COALESCE(source_id, ''), enabled, status, timezone, start_at, COALESCE(end_at, ''), COALESCE(recurrence, ''), COALESCE(deadline, ''), precision, COALESCE(original_wording, ''), COALESCE(daypart, ''), catch_up_policy, projection_policy, priority, COALESCE(authority_requirement, ''), COALESCE(dependency_json, ''), action, COALESCE(payload_json, ''), COALESCE(attestation_id, ''), COALESCE(next_due_at, ''), COALESCE(last_due_at, ''), created_at, updated_at FROM zodiac_schedules WHERE id = ?`, id).Scan(&item.ID, &item.Name, &item.Kind, &item.Owner, &item.Source, &sourceID, &enabled, &item.Status, &item.Timezone, &item.StartAt, &endAt, &recurrence, &deadline, &item.Precision, &wording, &daypart, &item.CatchUpPolicy, &item.ProjectionPolicy, &item.Priority, &authority, &dependency, &item.Action, &payload, &attestation, &nextDue, &lastDue, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return zodiacSchedule{}, err
	}
	item.Enabled = enabled != 0
	item.SourceID, item.EndAt, item.Recurrence, item.Deadline, item.OriginalWording, item.Daypart, item.AuthorityRequirement, item.NextDueAt, item.LastDueAt = sourceID.String, endAt.String, recurrence.String, deadline.String, wording.String, daypart.String, authority.String, nextDue.String, lastDue.String
	if dependency.Valid && dependency.String != "" {
		item.Dependency = json.RawMessage(dependency.String)
	}
	if payload.Valid && payload.String != "" {
		item.Payload = json.RawMessage(payload.String)
	}
	if attestation.Valid {
		item.AttestationID = attestation.String
	}
	return item, nil
}

func (d *daemon) listZodiacSchedules() ([]zodiacSchedule, error) {
	rows, err := d.db.Query(`SELECT id FROM zodiac_schedules ORDER BY enabled DESC, next_due_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []zodiacSchedule
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		item, err := d.loadZodiacSchedule(id)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *daemon) loadZodiacInstance(id string) (zodiacInstance, error) {
	var item zodiacInstance
	var catchup, proposal, authority, projection, lastError, completed sql.NullString
	err := d.db.QueryRow(`SELECT instance_id, schedule_id, occurrence_key, due_at, precision, state, COALESCE(catch_up_decision, ''), COALESCE(future_proposal_id, ''), COALESCE(authority_state, ''), COALESCE(projection_state, ''), COALESCE(last_error, ''), created_at, updated_at, COALESCE(completed_at, '') FROM zodiac_instances WHERE instance_id = ?`, id).Scan(&item.ID, &item.ScheduleID, &item.OccurrenceKey, &item.DueAt, &item.Precision, &item.State, &catchup, &proposal, &authority, &projection, &lastError, &item.CreatedAt, &item.UpdatedAt, &completed)
	item.CatchUpDecision, item.FutureProposalID, item.AuthorityState, item.ProjectionState, item.LastError, item.CompletedAt = catchup.String, proposal.String, authority.String, projection.String, lastError.String, completed.String
	return item, err
}

func (d *daemon) listZodiacInstances(limit int, where string, args ...any) ([]zodiacInstance, error) {
	if limit < 1 || limit > 500 {
		limit = 200
	}
	query := `SELECT instance_id FROM zodiac_instances ` + where + ` ORDER BY due_at, instance_id LIMIT ?`
	args = append(args, limit)
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []zodiacInstance
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		item, err := d.loadZodiacInstance(id)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *daemon) zodiacCreateInstance(schedule zodiacSchedule, due time.Time, state, decision, lastError string) (zodiacInstance, bool, error) {
	id, err := newLocalID("zodiac-instance-")
	if err != nil {
		return zodiacInstance{}, false, err
	}
	now := d.zodiacNow().Format(time.RFC3339Nano)
	key := due.Format(time.RFC3339Nano)
	if schedule.Precision == zodiacPrecisionDate {
		key = due.Format("2006-01-02")
	}
	result, err := d.db.Exec(`INSERT OR IGNORE INTO zodiac_instances (instance_id, schedule_id, occurrence_key, due_at, precision, state, catch_up_decision, last_error, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, schedule.ID, key, due.Format(time.RFC3339Nano), schedule.Precision, state, nullableString(decision), nullableString(lastError), now, now)
	if err != nil {
		return zodiacInstance{}, false, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected == 0 {
		return zodiacInstance{}, false, err
	}
	item, err := d.loadZodiacInstance(id)
	return item, true, err
}

func (d *daemon) zodiacUpdateScheduleDue(schedule zodiacSchedule, next, last string) error {
	updated := d.zodiacNow().Format(time.RFC3339Nano)
	_, err := d.db.Exec(`UPDATE zodiac_schedules SET next_due_at = ?, last_due_at = ?, updated_at = ? WHERE id = ?`, nullableString(next), nullableString(last), updated, schedule.ID)
	return err
}

func zodiacOccurrences(schedule zodiacSchedule, now time.Time) ([]time.Time, error) {
	start, precision, err := parseZodiacTime(schedule.StartAt, schedule.Timezone)
	if err != nil {
		return nil, err
	}
	if schedule.Precision == zodiacPrecisionDate || precision == zodiacPrecisionDate {
		start = start.In(now.Location())
	}
	if schedule.Recurrence == "" {
		if !start.After(now) {
			return []time.Time{start}, nil
		}
		return nil, nil
	}
	rule, err := parseZodiacRecurrence(schedule.Recurrence)
	if err != nil {
		return nil, err
	}
	result := make([]time.Time, 0, 32)
	current := start
	count := 0
	for count < 20000 && !current.After(now) {
		if rule.Until.IsZero() || !current.After(rule.Until) {
			include := len(rule.ByDay) == 0 || rule.ByDay[current.Weekday()]
			if include {
				result = append(result, current)
				count++
			}
		}
		if rule.Count > 0 && count >= rule.Count {
			break
		}
		switch rule.Frequency {
		case "HOURLY":
			current = current.Add(time.Duration(rule.Interval) * time.Hour)
		case "DAILY":
			current = current.AddDate(0, 0, rule.Interval)
		case "WEEKLY":
			current = current.AddDate(0, 0, 1)
			if len(rule.ByDay) == 0 && current.Weekday() == start.Weekday() {
				current = current.AddDate(0, 0, 7*(rule.Interval-1))
			}
		case "MONTHLY":
			current = current.AddDate(0, rule.Interval, 0)
		case "YEARLY":
			current = current.AddDate(rule.Interval, 0, 0)
		}
	}
	return result, nil
}

func zodiacLastAndNext(schedule zodiacSchedule, now time.Time) (string, string, []time.Time, error) {
	occurrences, err := zodiacOccurrences(schedule, now)
	if err != nil {
		return "", "", nil, err
	}
	if len(occurrences) == 0 {
		start, _, parseErr := parseZodiacTime(schedule.StartAt, schedule.Timezone)
		if parseErr != nil {
			return "", "", nil, parseErr
		}
		return "", start.Format(time.RFC3339Nano), nil, nil
	}
	last := occurrences[len(occurrences)-1]
	next := last
	if schedule.Recurrence != "" {
		rule, _ := parseZodiacRecurrence(schedule.Recurrence)
		switch rule.Frequency {
		case "HOURLY":
			next = next.Add(time.Duration(rule.Interval) * time.Hour)
		case "DAILY":
			next = next.AddDate(0, 0, rule.Interval)
		case "WEEKLY":
			next = next.AddDate(0, 0, 7*rule.Interval)
		case "MONTHLY":
			next = next.AddDate(0, rule.Interval, 0)
		case "YEARLY":
			next = next.AddDate(rule.Interval, 0, 0)
		}
	} else {
		next = time.Time{}
	}
	if next.IsZero() {
		return last.Format(time.RFC3339Nano), "", occurrences, nil
	}
	return last.Format(time.RFC3339Nano), next.Format(time.RFC3339Nano), occurrences, nil
}

func (d *daemon) zodiacDependencyState(schedule zodiacSchedule, now time.Time) (string, error) {
	if len(schedule.Dependency) == 0 {
		return "ready", nil
	}
	var dependency struct {
		Kind       string `json:"kind"`
		InstanceID string `json:"instance_id"`
		At         string `json:"at"`
	}
	if err := json.Unmarshal(schedule.Dependency, &dependency); err != nil {
		return "needs_you", err
	}
	switch dependency.Kind {
	case "after_completion":
		item, err := d.loadZodiacInstance(dependency.InstanceID)
		if errors.Is(err, sql.ErrNoRows) {
			return "needs_you", errors.New("dependency instance is unavailable")
		}
		if err != nil {
			return "needs_you", err
		}
		if item.State == zodiacInstanceCompleted {
			return "ready", nil
		}
		return "waiting", nil
	case "not_before":
		at, err := time.Parse(time.RFC3339Nano, dependency.At)
		if err != nil {
			return "needs_you", err
		}
		if now.Before(at) {
			return "waiting", nil
		}
		return "ready", nil
	case "no_reply":
		return "needs_you", errors.New("reply state is not available to deterministic Zodiac")
	default:
		return "needs_you", fmt.Errorf("unsupported dependency kind %q", dependency.Kind)
	}
}

func (d *daemon) zodiacProposalForInstance(schedule zodiacSchedule, instance zodiacInstance) (string, error) {
	if schedule.Action != zodiacActionMailRound {
		return "", nil
	}
	proposalID := "zodiac-proposal-" + instance.ID
	var existing string
	if err := d.db.QueryRow(`SELECT proposal_id FROM future_proposals WHERE proposal_id = ?`, proposalID).Scan(&existing); err == nil {
		return existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	payload, _ := json.Marshal(map[string]any{"zodiac_instance_id": instance.ID, "zodiac_schedule_id": schedule.ID, "action": schedule.Action})
	proposal, err := d.createFutureProposal(futureProposalRequest{ProposalID: proposalID, SourceMissive: "Zodiac due instance " + instance.ID + " requests bounded " + schedule.Action, PackageCode: "ZODIAC:MAIL_ROUND", Capability: "mail.round", Scope: "zodiac:instance:" + instance.ID, Adapter: "pantheon", Payload: payload, Purpose: schedule.Name, RequestedExecutionBound: instance.DueAt})
	if err != nil {
		return "", err
	}
	return proposal.ProposalID, nil
}

func (d *daemon) zodiacPrepareInstance(schedule zodiacSchedule, instance zodiacInstance, paused bool) (zodiacInstance, bool, error) {
	if paused {
		_, err := d.db.Exec(`UPDATE zodiac_instances SET state = ?, authority_state = ?, last_error = ?, updated_at = ? WHERE instance_id = ? AND state IN (?, ?, ?)`, zodiacInstanceBlockedPrometheus, "blocked_by_prometheus", "Prometheus is paused; this due instance was not dispatched", d.zodiacNow().Format(time.RFC3339Nano), instance.ID, zodiacInstanceDue, zodiacInstancePending, zodiacInstanceAwaitingAuthority)
		if err != nil {
			return instance, false, err
		}
		updated, err := d.loadZodiacInstance(instance.ID)
		return updated, false, err
	}
	dependencyState, dependencyErr := d.zodiacDependencyState(schedule, d.zodiacNow())
	if dependencyErr != nil || dependencyState == "needs_you" {
		message := "dependency requires human review"
		if dependencyErr != nil {
			message = dependencyErr.Error()
		}
		_, err := d.db.Exec(`UPDATE zodiac_instances SET state = ?, last_error = ?, updated_at = ? WHERE instance_id = ?`, zodiacInstanceNeedsYou, message, d.zodiacNow().Format(time.RFC3339Nano), instance.ID)
		if err != nil {
			return instance, false, err
		}
		updated, err := d.loadZodiacInstance(instance.ID)
		return updated, false, err
	}
	if dependencyState == "waiting" {
		_, err := d.db.Exec(`UPDATE zodiac_instances SET state = ?, last_error = ?, updated_at = ? WHERE instance_id = ?`, zodiacInstanceWaiting, "waiting on declared dependency", d.zodiacNow().Format(time.RFC3339Nano), instance.ID)
		updated, loadErr := d.loadZodiacInstance(instance.ID)
		return updated, false, errOr(err, loadErr)
	}
	if schedule.Owner == "libra" && schedule.Action == zodiacActionBriefGenerate {
		run, reportErr := d.processLibraReportInstance(context.Background(), schedule, instance)
		if reportErr != nil {
			state, authority, message := zodiacInstanceFailed, "report_pipeline_failed", reportErr.Error()
			if errors.Is(reportErr, libraNeedsEvidence) || strings.Contains(strings.ToLower(reportErr.Error()), "apollo did not complete") {
				state, authority = zodiacInstanceNeedsYou, "report_needs_decision"
			}
			_, updateErr := d.db.Exec(`UPDATE zodiac_instances SET state = ?, authority_state = ?, last_error = ?, updated_at = ? WHERE instance_id = ?`, state, authority, message, d.zodiacNow().Format(time.RFC3339Nano), instance.ID)
			updated, loadErr := d.loadZodiacInstance(instance.ID)
			if updateErr != nil {
				return instance, false, updateErr
			}
			if loadErr != nil {
				return instance, false, loadErr
			}
			_ = run
			return updated, false, nil
		}
		completedAt := d.zodiacNow().Format(time.RFC3339Nano)
		lastError := run.Error
		_, updateErr := d.db.Exec(`UPDATE zodiac_instances SET state = ?, authority_state = ?, last_error = ?, completed_at = ?, updated_at = ? WHERE instance_id = ?`, zodiacInstanceCompleted, "report_complete", nullableString(lastError), completedAt, completedAt, instance.ID)
		updated, loadErr := d.loadZodiacInstance(instance.ID)
		if updateErr != nil {
			return instance, false, updateErr
		}
		return updated, false, loadErr
	}
	proposalID, err := d.zodiacProposalForInstance(schedule, instance)
	if err != nil {
		return instance, false, err
	}
	if proposalID == "" {
		state, message := zodiacInstanceAwaitingPipeline, "the bounded action pipeline is not implemented in this kernel"
		if strings.HasPrefix(schedule.Action, "calendar.") {
			state, message = zodiacInstanceNeedsYou, "calendar projection requires an approved provider adapter"
		}
		_, err := d.db.Exec(`UPDATE zodiac_instances SET state = ?, authority_state = ?, projection_state = ?, last_error = ?, updated_at = ? WHERE instance_id = ?`, state, "not_requested", "not_requested", message, d.zodiacNow().Format(time.RFC3339Nano), instance.ID)
		updated, loadErr := d.loadZodiacInstance(instance.ID)
		return updated, false, errOr(err, loadErr)
	}
	_, err = d.db.Exec(`UPDATE zodiac_instances SET state = ?, authority_state = ?, future_proposal_id = ?, updated_at = ? WHERE instance_id = ? AND future_proposal_id IS NULL`, zodiacInstanceAwaitingAuthority, "awaiting_future", proposalID, d.zodiacNow().Format(time.RFC3339Nano), instance.ID)
	if err != nil {
		return instance, false, err
	}
	updated, err := d.loadZodiacInstance(instance.ID)
	return updated, true, err
}

func (d *daemon) zodiacRound(ctx context.Context) (zodiacRoundResponse, error) {
	d.zodiacMu.Lock()
	defer d.zodiacMu.Unlock()
	if err := initializeZodiacSchema(d.db); err != nil {
		return zodiacRoundResponse{}, err
	}
	started := d.zodiacNow()
	paused, err := d.paused()
	if err != nil {
		return zodiacRoundResponse{}, err
	}
	response := zodiacRoundResponse{StartedAt: started.Format(time.RFC3339Nano), ObservedAt: started.Format(time.RFC3339Nano), Paused: paused}
	schedules, err := d.listZodiacSchedules()
	if err != nil {
		return response, err
	}
	for _, schedule := range schedules {
		if !schedule.Enabled {
			continue
		}
		response.Examined++
		last, next, occurrences, evalErr := zodiacLastAndNext(schedule, started)
		if evalErr != nil {
			response.Failed++
			response.Errors = append(response.Errors, schedule.ID+": "+evalErr.Error())
			continue
		}
		if err := d.zodiacUpdateScheduleDue(schedule, next, last); err != nil {
			response.Failed++
			response.Errors = append(response.Errors, schedule.ID+": "+err.Error())
			continue
		}
		if len(occurrences) == 0 {
			continue
		}
		seen := map[string]bool{}
		rows, queryErr := d.db.Query(`SELECT occurrence_key FROM zodiac_instances WHERE schedule_id = ?`, schedule.ID)
		if queryErr == nil {
			for rows.Next() {
				var key string
				if rows.Scan(&key) == nil {
					seen[key] = true
				}
			}
			rows.Close()
		}
		selected := occurrences
		if schedule.Recurrence != "" && len(occurrences) > 1 {
			switch schedule.CatchUpPolicy {
			case zodiacCatchUpSkip:
				for _, occurrence := range occurrences[:len(occurrences)-1] {
					key := occurrence.Format(time.RFC3339Nano)
					if seen[key] {
						continue
					}
					if _, created, createErr := d.zodiacCreateInstance(schedule, occurrence, zodiacInstanceSkipped, zodiacCatchUpSkip, "missed occurrence skipped by schedule policy"); createErr != nil {
						response.Failed++
						response.Errors = append(response.Errors, createErr.Error())
					} else if created {
						response.Skipped++
					}
				}
				selected = occurrences[len(occurrences)-1:]
			case zodiacCatchUpRunOnce, zodiacCatchUpReconcile, zodiacCatchUpNeedsYou:
				selected = occurrences[len(occurrences)-1:]
			}
		}
		for _, occurrence := range selected {
			key := occurrence.Format(time.RFC3339Nano)
			if schedule.Precision == zodiacPrecisionDate {
				key = occurrence.Format("2006-01-02")
			}
			if seen[key] {
				continue
			}
			state, decision := zodiacInstanceDue, "on_time"
			if paused {
				state, decision = zodiacInstanceBlockedPrometheus, "blocked_by_prometheus"
			}
			if schedule.Recurrence != "" && !occurrence.Equal(started) {
				decision = schedule.CatchUpPolicy
			}
			if schedule.CatchUpPolicy == zodiacCatchUpNeedsYou {
				state, decision = zodiacInstanceNeedsYou, "needs_you"
			}
			instance, created, createErr := d.zodiacCreateInstance(schedule, occurrence, state, decision, "")
			if createErr != nil {
				response.Failed++
				response.Errors = append(response.Errors, schedule.ID+": "+createErr.Error())
				continue
			}
			if !created {
				continue
			}
			response.Due++
			if paused {
				response.Blocked++
				continue
			}
			updated, proposal, prepareErr := d.zodiacPrepareInstance(schedule, instance, paused)
			if prepareErr != nil {
				response.Failed++
				response.Errors = append(response.Errors, schedule.ID+": "+prepareErr.Error())
				continue
			}
			if proposal {
				response.Proposals++
			}
			if updated.State == zodiacInstanceNeedsYou {
				response.Blocked++
			}
		}
	}
	if err := d.zodiacSyncProposalStates(); err != nil {
		response.Failed++
		response.Errors = append(response.Errors, err.Error())
	}
	response.Result = fmt.Sprintf("Zodiac examined %d schedule(s), found %d due instance(s), created %d Future proposal(s), skipped %d, blocked %d, failed %d.", response.Examined, response.Due, response.Proposals, response.Skipped, response.Blocked, response.Failed)
	roundID, _ := newLocalID("zodiac-round-")
	data, _ := json.Marshal(response)
	_, _ = d.db.Exec(`INSERT INTO zodiac_rounds (round_id, started_at, observed_at, paused, result_json) VALUES (?, ?, ?, ?, ?)`, roundID, response.StartedAt, response.ObservedAt, response.Paused, string(data))
	d.audit("POST /v1/zodiac/round", "", "zodiac.round", "", "completed", response.Result, "", 0)
	_ = ctx
	return response, nil
}

func (d *daemon) zodiacSyncProposalStates() error {
	rows, err := d.db.Query(`SELECT instance_id, future_proposal_id FROM zodiac_instances WHERE future_proposal_id IS NOT NULL AND state NOT IN (?, ?, ?)`, zodiacInstanceCompleted, zodiacInstanceFailed, zodiacInstanceSkipped)
	if err != nil {
		return err
	}
	type pair struct{ instance, proposal string }
	var pairs []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.instance, &p.proposal); err != nil {
			rows.Close()
			return err
		}
		pairs = append(pairs, p)
	}
	rows.Close()
	for _, p := range pairs {
		proposal, err := d.loadFutureProposal(p.proposal)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		state, message := zodiacInstanceAwaitingAuthority, proposal.State
		if proposal.State == futureStateNeedsAuthority || proposal.State == futureStateNeedsPerson || proposal.State == futureStateReferredPrometheus {
			state, message = zodiacInstanceNeedsYou, "Future requires authority before this due work can proceed"
		}
		if proposal.State == futureStateAdapterUnavailable || proposal.State == futureStateMalformed {
			state, message = zodiacInstanceFailed, proposal.LastError
		}
		if proposal.OrderID != "" {
			order, orderErr := d.loadOrder(proposal.OrderID)
			if orderErr == nil {
				switch order.Status {
				case orderStatusCompleted:
					state, message = zodiacInstanceCompleted, ""
				case orderStatusFailed, orderStatusUnavailable, orderStatusExpired:
					state, message = zodiacInstanceFailed, order.LastError
				case orderStatusSealed, orderStatusReady, orderStatusRunning:
					state, message = zodiacInstanceDispatched, order.Status
				}
			}
		}
		completed := ""
		if state == zodiacInstanceCompleted {
			completed = d.zodiacNow().Format(time.RFC3339Nano)
		}
		_, err = d.db.Exec(`UPDATE zodiac_instances SET state = ?, authority_state = ?, last_error = ?, completed_at = CASE WHEN ? <> '' THEN ? ELSE completed_at END, updated_at = ? WHERE instance_id = ?`, state, message, nullableString(message), completed, completed, d.zodiacNow().Format(time.RFC3339Nano), p.instance)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *daemon) zodiacCreateProjectionRows(schedule zodiacSchedule) error {
	providers := zodiacProjectionProviders(schedule.ProjectionPolicy)
	if len(providers) == 0 {
		return nil
	}
	now := d.zodiacNow().Format(time.RFC3339Nano)
	for index, provider := range providers {
		role := "primary"
		if index > 0 {
			role = "secondary"
		}
		id, err := newLocalID("zodiac-projection-")
		if err != nil {
			return err
		}
		_, err = d.db.Exec(`INSERT OR IGNORE INTO zodiac_projections (projection_id, schedule_id, provider, account, calendar, projection_role, created_at, status, correction) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, schedule.ID, provider, zodiacProviderAccount(provider), zodiacProviderCalendar(provider), role, now, "awaiting_adapter", "")
		if err != nil {
			return err
		}
	}
	return nil
}

func zodiacProjectionProviders(policy string) []string {
	switch policy {
	case zodiacProjectionApple:
		return []string{"apple"}
	case zodiacProjectionGoogle:
		return []string{"google"}
	case zodiacProjectionBoth:
		return []string{"google", "apple"}
	default:
		return nil
	}
}

func zodiacProviderAccount(provider string) string {
	if provider == "apple" {
		return "icloud"
	}
	return "work"
}
func zodiacProviderCalendar(provider string) string {
	if provider == "apple" {
		return "family"
	}
	return "work"
}

func (d *daemon) zodiacProviderHealth(provider string) zodiacProviderHealth {
	capabilityID := "calendar." + provider
	result := zodiacProviderHealth{Provider: provider, CapabilityID: capabilityID, Operations: []string{"calendar.list", "calendar.read", "calendar.create", "calendar.update", "calendar.cancel", "calendar.reconcile", "calendar.availability"}, Credentials: "omitted"}
	capability, err := d.loadBifrostCapability(capabilityID)
	if errors.Is(err, sql.ErrNoRows) {
		result.State, result.Reason = "unavailable", "No approved Bifrost calendar adapter is registered."
		return result
	}
	if err != nil {
		result.State, result.Reason = "unknown", "Bifrost capability state could not be read."
		return result
	}
	if !capability.Enabled || !capability.UseApproved {
		result.State, result.Reason = "approval_required", "Capability is present but is not enabled and use-approved."
		return result
	}
	result.State, result.Reason = "ready_contract_only", "Capability is approved; provider invocation still requires its registered adapter contract."
	return result
}

func (d *daemon) loadZodiacProjection(id string) (zodiacProjection, error) {
	var item zodiacProjection
	var instance, external, synced, revision, correction sql.NullString
	err := d.db.QueryRow(`SELECT projection_id, schedule_id, COALESCE(instance_id, ''), provider, account, calendar, COALESCE(external_event_id, ''), projection_role, created_at, COALESCE(last_synced_at, ''), COALESCE(last_seen_external_revision, ''), status, COALESCE(correction, '') FROM zodiac_projections WHERE projection_id = ?`, id).Scan(&item.ID, &item.ScheduleID, &instance, &item.Provider, &item.Account, &item.Calendar, &external, &item.ProjectionRole, &item.CreatedAt, &synced, &revision, &item.Status, &correction)
	item.InstanceID, item.ExternalEventID, item.LastSyncedAt, item.LastSeenExternalRevision, item.Correction = instance.String, external.String, synced.String, revision.String, correction.String
	return item, err
}

func (d *daemon) zodiacInspection() (zodiacInspection, error) {
	if err := initializeZodiacSchema(d.db); err != nil {
		return zodiacInspection{}, err
	}
	schedules, err := d.listZodiacSchedules()
	if err != nil {
		return zodiacInspection{}, err
	}
	instances, err := d.listZodiacInstances(200, "WHERE state NOT IN (?, ?)", zodiacInstanceCompleted, zodiacInstanceSkipped)
	if err != nil {
		return zodiacInspection{}, err
	}
	needs, err := d.listZodiacInstances(200, "WHERE state IN (?, ?, ?, ?)", zodiacInstanceNeedsYou, zodiacInstanceBlockedPrometheus, zodiacInstanceAwaitingPipeline, zodiacInstanceFailed)
	if err != nil {
		return zodiacInspection{}, err
	}
	rows, err := d.db.Query(`SELECT projection_id FROM zodiac_projections ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		return zodiacInspection{}, err
	}
	var projections []zodiacProjection
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return zodiacInspection{}, err
		}
		projection, err := d.loadZodiacProjection(id)
		if err != nil {
			rows.Close()
			return zodiacInspection{}, err
		}
		projections = append(projections, projection)
	}
	rows.Close()
	paused, err := d.paused()
	if err != nil {
		return zodiacInspection{}, err
	}
	var recurring []zodiacSchedule
	for _, schedule := range schedules {
		if schedule.Recurrence != "" {
			recurring = append(recurring, schedule)
		}
	}
	return zodiacInspection{Enabled: d.cfg.ZodiacEnabled, Now: d.zodiacNow().Format(time.RFC3339Nano), Timezone: d.cfg.ZodiacDefaultTimezone, Paused: paused, Schedules: schedules, Instances: instances, Projections: projections, NeedsYou: needs, Providers: []zodiacProviderHealth{d.zodiacProviderHealth("apple"), d.zodiacProviderHealth("google")}, Recurring: recurring, Contract: map[string]any{"intelligence": "upstream only", "authority": "Future", "execution": "Mercury", "external_calendars": "Bifrost projections; presence is not permission", "notifications": "existing Pantheon notification system"}}, nil
}

func (d *daemon) zodiacToday() (map[string]any, error) {
	now := d.zodiacNow()
	start := now.Format("2006-01-02") + "T00:00:00" + zoneOffset(now)
	end := now.AddDate(0, 0, 1).Format("2006-01-02") + "T00:00:00" + zoneOffset(now)
	items, err := d.listZodiacInstances(200, "WHERE due_at >= ? AND due_at < ?", start, end)
	if err != nil {
		return nil, err
	}
	return map[string]any{"date": now.Format("2006-01-02"), "timezone": d.cfg.ZodiacDefaultTimezone, "items": items, "note": "Today shows declarations Zodiac has made durable; calendar details remain behind approved Bifrost projections."}, nil
}

func (d *daemon) zodiacUpcoming() (map[string]any, error) {
	items, err := d.listZodiacInstances(100, "WHERE due_at >= ? AND state NOT IN (?, ?, ?)", d.zodiacNow().Format(time.RFC3339Nano), zodiacInstanceCompleted, zodiacInstanceSkipped, zodiacInstanceFailed)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": items, "timezone": d.cfg.ZodiacDefaultTimezone, "note": "Upcoming work is bounded and sourced from persisted due instances."}, nil
}

func (d *daemon) recordZodiacAttestation(request zodiacAttestation) (zodiacAttestation, error) {
	request.SourceID, request.CitationStart, request.CitationEnd, request.QuotedSpan = strings.TrimSpace(request.SourceID), strings.TrimSpace(request.CitationStart), strings.TrimSpace(request.CitationEnd), strings.TrimSpace(request.QuotedSpan)
	if request.SourceID == "" || request.CitationStart == "" || request.CitationEnd == "" || request.QuotedSpan == "" || len(request.Normalized) == 0 || !json.Valid(request.Normalized) {
		return zodiacAttestation{}, errors.New("source, citation bounds, quoted span, and valid normalized action are required")
	}
	if request.Confidence < 0 || request.Confidence > 1 {
		return zodiacAttestation{}, errors.New("confidence must be between 0 and 1")
	}
	result, reason := "unverified_source", "source text was not supplied to the deterministic checker"
	if request.SourcePath != "" {
		data, err := os.ReadFile(request.SourcePath)
		if err != nil {
			return zodiacAttestation{}, fmt.Errorf("source cannot be read: %w", err)
		}
		digest := sha256.Sum256(data)
		fingerprint := hex.EncodeToString(digest[:])
		if request.SourceFingerprint != "" && request.SourceFingerprint != fingerprint {
			return zodiacAttestation{}, errors.New("source fingerprint does not match")
		}
		if !strings.Contains(string(data), request.QuotedSpan) {
			return zodiacAttestation{}, errors.New("quoted span is not present in source")
		}
		request.SourceFingerprint, result, reason = fingerprint, "verified", "source fingerprint and quoted span verified"
	}
	if request.ID == "" {
		var err error
		request.ID, err = newLocalID("zodiac-attestation-")
		if err != nil {
			return zodiacAttestation{}, err
		}
	}
	request.Result, request.Reason, request.CreatedAt = result, reason, d.zodiacNow().Format(time.RFC3339Nano)
	_, err := d.db.Exec(`INSERT INTO zodiac_attestations (attestation_id, source_id, source_path, source_fingerprint, speaker, citation_start, citation_end, quoted_span, normalized_json, confidence, result, reason, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, request.ID, request.SourceID, nullableString(request.SourcePath), nullableString(request.SourceFingerprint), nullableString(request.Speaker), request.CitationStart, request.CitationEnd, request.QuotedSpan, string(request.Normalized), request.Confidence, request.Result, request.Reason, request.CreatedAt)
	return request, err
}

func (d *daemon) reconcileZodiacProjection(id string, request zodiacProjectionReconcileRequest) (zodiacProjection, error) {
	projection, err := d.loadZodiacProjection(id)
	if err != nil {
		return zodiacProjection{}, err
	}
	if strings.TrimSpace(request.ExternalEventID) == "" {
		return zodiacProjection{}, errors.New("external_event_id is required")
	}
	now := d.zodiacNow().Format(time.RFC3339Nano)
	status, correction := "reconciled", request.Correction
	if request.Deleted {
		status, correction = "human_deleted", "Human deletion is preserved; Zodiac will not recreate this projection automatically."
	} else if request.HumanModified {
		status = "human_corrected"
		if correction == "" {
			correction = "Human calendar edit takes precedence over the prior Pantheon projection."
		}
	}
	snapshot, _ := json.Marshal(request)
	_, err = d.db.Exec(`UPDATE zodiac_projections SET external_event_id = ?, last_synced_at = ?, last_seen_external_revision = ?, status = ?, correction = ? WHERE projection_id = ?`, request.ExternalEventID, now, nullableString(request.ExternalRevision), status, string(snapshot)+"; "+correction, id)
	if err != nil {
		return zodiacProjection{}, err
	}
	projection, err = d.loadZodiacProjection(id)
	if err != nil {
		return zodiacProjection{}, err
	}
	if request.HumanModified || request.Deleted {
		var siblingIDs []string
		rows, queryErr := d.db.Query(`SELECT projection_id FROM zodiac_projections WHERE schedule_id = ? AND projection_id <> ?`, projection.ScheduleID, projection.ID)
		if queryErr == nil {
			for rows.Next() {
				var sibling string
				if rows.Scan(&sibling) == nil {
					siblingIDs = append(siblingIDs, sibling)
				}
			}
			rows.Close()
		}
		for _, sibling := range siblingIDs {
			_, _ = d.db.Exec(`UPDATE zodiac_projections SET status = CASE WHEN status = 'human_corrected' OR status = 'human_deleted' THEN status ELSE 'needs_you' END, correction = CASE WHEN status = 'human_corrected' OR status = 'human_deleted' THEN correction ELSE ? END WHERE projection_id = ?`, "Another provider projection received a human correction; compare both revisions before choosing a policy.", sibling)
		}
	}
	return projection, nil
}

func (d *daemon) startZodiacWatcher() func() {
	if !d.cfg.ZodiacEnabled {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	interval := time.Duration(d.cfg.ZodiacWakeSeconds) * time.Second
	if interval < 5*time.Second {
		interval = 30 * time.Second
	}
	go func() {
		if err := d.processLibraIngestionQueue(ctx, 4); err != nil {
			d.audit("libra.ingestion.watcher", "", "libra.report", "", "failed", "", err.Error(), 0)
		}
		if _, err := d.zodiacRound(ctx); err != nil {
			d.audit("zodiac.watcher", "", "zodiac.round", "", "failed", "", err.Error(), 0)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := d.processLibraIngestionQueue(ctx, 4); err != nil {
					d.audit("libra.ingestion.watcher", "", "libra.report", "", "failed", "", err.Error(), 0)
				}
				if _, err := d.zodiacRound(ctx); err != nil {
					d.audit("zodiac.watcher", "", "zodiac.round", "", "failed", "", err.Error(), 0)
				}
			}
		}
	}()
	return cancel
}

func (d *daemon) handleShowZodiac(w http.ResponseWriter, r *http.Request) {
	inspection, err := d.zodiacInspection()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, inspection)
}
func (d *daemon) handleZodiacToday(w http.ResponseWriter, r *http.Request) {
	result, err := d.zodiacToday()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (d *daemon) handleZodiacUpcoming(w http.ResponseWriter, r *http.Request) {
	result, err := d.zodiacUpcoming()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (d *daemon) handleZodiacNeedsYou(w http.ResponseWriter, r *http.Request) {
	inspection, err := d.zodiacInspection()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": inspection.NeedsYou, "timezone": inspection.Timezone, "note": "Needs You is a first-class exception queue; Zodiac does not guess through ambiguity or missing authority."})
}
func (d *daemon) handleZodiacRound(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("Zodiac round requires the local CLI control boundary"))
		return
	}
	result, err := d.zodiacRound(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (d *daemon) handleCreateZodiacSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("schedule creation requires the local CLI control boundary"))
		return
	}
	var request zodiacScheduleRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	schedule, err := d.createZodiacSchedule(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, schedule)
}

func (d *daemon) handleZodiacAttestation(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("attestation recording requires the local CLI control boundary"))
		return
	}
	var request zodiacAttestation
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	attestation, err := d.recordZodiacAttestation(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, attestation)
}

func (d *daemon) handleZodiacProjectionReconcile(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("calendar reconciliation requires the local CLI control boundary"))
		return
	}
	var request zodiacProjectionReconcileRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	projection, err := d.reconcileZodiacProjection(r.PathValue("id"), request)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("Zodiac projection not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, projection)
}
func (d *daemon) handleShowZodiacSchedule(w http.ResponseWriter, r *http.Request) {
	schedule, err := d.loadZodiacSchedule(r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("Zodiac schedule not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, schedule)
}
func (d *daemon) handleDisableZodiacSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("schedule disable requires the local CLI control boundary"))
		return
	}
	id := r.PathValue("id")
	schedule, err := d.loadZodiacSchedule(id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("Zodiac schedule not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	previous, _ := json.Marshal(schedule)
	now := d.zodiacNow().Format(time.RFC3339Nano)
	if _, err := d.db.Exec(`UPDATE zodiac_schedules SET enabled = 0, status = ?, updated_at = ? WHERE id = ?`, zodiacScheduleDisabled, now, id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	schedule.Enabled, schedule.Status, schedule.UpdatedAt = false, zodiacScheduleDisabled, now
	updated, _ := json.Marshal(schedule)
	revisionID, _ := newLocalID("zodiac-revision-")
	_, _ = d.db.Exec(`INSERT INTO zodiac_revisions (revision_id, schedule_id, previous_json, new_json, source, reason, occurred_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, revisionID, id, string(previous), string(updated), "human_gui", "schedule disabled", now)
	writeJSON(w, http.StatusOK, schedule)
}
