package pantheon

import (
	"net/http"
	"os"
	"os/exec"
	"time"
)

const systemInventoryContractVersion = 1

type systemComponent struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	Kind            string   `json:"kind"`
	State           string   `json:"state"`
	Role            string   `json:"role"`
	InspectionRoute string   `json:"inspection_route,omitempty"`
	Controls        []string `json:"controls,omitempty"`
}

type systemWorkSummary struct {
	Intake    map[string]any `json:"intake"`
	Rumor     map[string]any `json:"rumor"`
	Apollo    map[string]any `json:"apollo"`
	Future    map[string]any `json:"future"`
	Janus     map[string]any `json:"janus"`
	Mercury   map[string]any `json:"mercury"`
	Bifrost   map[string]any `json:"bifrost"`
	Mail      map[string]any `json:"mail"`
	Zodiac    map[string]any `json:"zodiac"`
	Aphrodite map[string]any `json:"aphrodite"`
	Failures  map[string]any `json:"failures"`
}

type systemInventoryResponse struct {
	ContractVersion int               `json:"contract_version"`
	GeneratedAt     string            `json:"generated_at"`
	Runtime         map[string]any    `json:"runtime"`
	Models          map[string]any    `json:"models"`
	Components      []systemComponent `json:"components"`
	Work            systemWorkSummary `json:"work"`
	Security        map[string]any    `json:"security"`
	Sources         map[string]string `json:"sources"`
}

func (d *daemon) systemInventory() (systemInventoryResponse, error) {
	paused, err := d.paused()
	if err != nil {
		return systemInventoryResponse{}, err
	}
	intake, err := d.systemIntakeSummary()
	if err != nil {
		return systemInventoryResponse{}, err
	}
	rumor, err := d.systemRumorSummary()
	if err != nil {
		return systemInventoryResponse{}, err
	}
	apollo, err := d.systemApolloSummary()
	if err != nil {
		return systemInventoryResponse{}, err
	}
	future, err := d.systemStateCounts("future_proposals", "state")
	if err != nil {
		return systemInventoryResponse{}, err
	}
	janus, err := d.systemStateCounts("janus_subjects", "state")
	if err != nil {
		return systemInventoryResponse{}, err
	}
	mercury, err := d.systemStateCounts("orders", "status")
	if err != nil {
		return systemInventoryResponse{}, err
	}
	bifrost, err := d.systemBifrostSummary()
	if err != nil {
		return systemInventoryResponse{}, err
	}
	mailSummary, err := d.systemMailSummary()
	if err != nil {
		return systemInventoryResponse{}, err
	}
	zodiac, err := d.zodiacInventorySummary()
	if err != nil {
		return systemInventoryResponse{}, err
	}
	aphrodite, err := d.systemAphroditeSummary()
	if err != nil {
		return systemInventoryResponse{}, err
	}
	failures, err := d.systemFailureSummary()
	if err != nil {
		return systemInventoryResponse{}, err
	}

	localModelState := "unconfigured"
	if d.cfg.LocalModel != "" && d.cfg.OllamaURL != "" {
		localModelState = "configured"
	}
	mercuryState := "unavailable"
	if d.cfg.MercuryDeterministicFallback {
		mercuryState = "ready"
	} else if d.cfg.MercuryBinary != "" {
		if _, lookErr := exec.LookPath(d.cfg.MercuryBinary); lookErr == nil {
			mercuryState = "ready"
		}
	}
	notifyState := "unavailable"
	if _, statErr := os.Stat(notifyCommand); statErr == nil {
		notifyState = "ready"
	}
	vaultState := "unconfigured"
	if d.cfg.VaultPath != "" {
		vaultState = "configured"
	}

	components := []systemComponent{
		{ID: "pantheon", Label: "Pantheon kernel", Kind: "daemon", State: "running", Role: "configuration, orchestration, policy, audit, and control", InspectionRoute: "/v1/status"},
		{ID: "intake", Label: "Intake", Kind: "deterministic", State: "ready", Role: "identity-track incoming occurrences", InspectionRoute: "/v1/intake"},
		{ID: "present", Label: "Present", Kind: "deterministic", State: "ready", Role: "faithful source normalization and verification", InspectionRoute: "/v1/present", Controls: []string{"POST /v1/present"}},
		{ID: "past", Label: "Past", Kind: "retrieval", State: vaultState, Role: "bounded cited local continuity", InspectionRoute: "/v1/weave/retrieve"},
		{ID: "weave", Label: "Weave", Kind: "deterministic", State: vaultState, Role: "derived local relationship and retrieval index", InspectionRoute: "/v1/weave/retrieve", Controls: []string{"POST /v1/weave/refresh"}},
		{ID: "apollo", Label: "Apollo", Kind: "reasoning", State: localModelState, Role: "local cited reasoning and proposals", InspectionRoute: "/v1/apollo"},
		{ID: "seshat", Label: "Seshat", Kind: "deterministic", State: "ready", Role: "package registry and missive contracts", InspectionRoute: "/v1/seshat/index"},
		{ID: "father_time", Label: "Father Time", Kind: "deterministic", State: "ready", Role: "chronology and time references", InspectionRoute: "/v1/father-time"},
		{ID: "janus", Label: "Janus", Kind: "deterministic", State: "ready", Role: "declared temporal thresholds and transitions", InspectionRoute: "/v1/janus", Controls: []string{"POST /v1/janus/round"}},
		{ID: "future", Label: "Future", Kind: "deterministic", State: "ready", Role: "authority administration and exact sealing", InspectionRoute: "/v1/future", Controls: []string{"POST /v1/future/round"}},
		{ID: "mercury", Label: "Mercury", Kind: "deterministic", State: mercuryState, Role: "sealed-order transport, Go, and bounded execution", InspectionRoute: "/v1/mercury", Controls: []string{"POST /v1/mercury/round"}},
		{ID: "prometheus", Label: "Prometheus", Kind: "control", State: map[bool]string{true: "paused", false: "running"}[paused], Role: "emergency pause, resume, approval, and revocation", InspectionRoute: "/v1/status", Controls: []string{"pantheon prometheus protocol --pause", "pantheon prometheus protocol --resume", "pantheon prometheus protocol --hard-stop"}},
		{ID: "rumor", Label: "Rumor", Kind: "deterministic", State: map[bool]string{true: "enabled", false: "disabled"}[d.cfg.RumorEnabled], Role: "local audio stability, transcription, and Intake handoff", InspectionRoute: "/v1/rumor", Controls: []string{"POST /v1/rumor/scan", "POST /v1/rumor/transcribe"}},
		{ID: "bifrost", Label: "Bifrost", Kind: "integration", State: map[bool]string{true: "enabled", false: "disabled"}[d.cfg.BifrostEnabled], Role: "governed external-capability registry and adapters", InspectionRoute: "/v1/bifrost"},
		{ID: "mail", Label: "Mail", Kind: "workflow", State: map[bool]string{true: "enabled", false: "disabled"}[d.cfg.MailEnabled], Role: "manual bounded mail intake, understanding, backlog, and governed actions", InspectionRoute: "/v1/mail", Controls: []string{"POST /v1/mail/round"}},
		{ID: "notifications", Label: "Notifications", Kind: "integration", State: notifyState, Role: "fixed local notification adapter", InspectionRoute: "/v1/notify", Controls: []string{"POST /v1/notify"}},
		{ID: "archimedes", Label: "Archimedes", Kind: "reasoning", State: "deferred", Role: "future frontier reasoning boundary"},
		{ID: "zodiac", Label: "Zodiac", Kind: "deterministic", State: map[bool]string{true: "enabled", false: "disabled"}[d.cfg.ZodiacEnabled], Role: "explicit schedules, recurrence, due instances, and calendar projections", InspectionRoute: "/v1/zodiac", Controls: []string{"POST /v1/zodiac/round"}},
		{ID: "aphrodite", Label: "Aphrodite", Kind: "development intelligence", State: "bounded workshop", Role: "project lifecycle, research, experiments, and candidate construction outside production", InspectionRoute: "/v1/aphrodite"},
		{ID: "zeus", Label: "Zeus", Kind: "deterministic", State: "deferred", Role: "future explicitly identified machine-garbage sanitation"},
	}

	return systemInventoryResponse{
		ContractVersion: systemInventoryContractVersion,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		Runtime: map[string]any{
			"listen_address": defaultListenAddress,
			"database_path":  d.databasePath,
			"paused":         paused,
			"audit_events":   failures["audit_events"],
		},
		Models: map[string]any{
			"apollo":     map[string]any{"model": d.cfg.LocalModel, "extraction_model": d.cfg.ApolloExtractionModel, "endpoint": d.cfg.OllamaURL, "state": localModelState, "local_only_by_default": true},
			"archimedes": map[string]any{"model": d.cfg.FrontierModel, "state": "deferred", "enabled": false, "data_egress": false},
		},
		Components: components,
		Work:       systemWorkSummary{Intake: intake, Rumor: rumor, Apollo: apollo, Future: map[string]any{"state_counts": future}, Janus: map[string]any{"state_counts": janus}, Mercury: map[string]any{"state_counts": mercury}, Bifrost: bifrost, Mail: mailSummary, Zodiac: zodiac, Aphrodite: aphrodite, Failures: failures},
		Security: map[string]any{
			"control_token":          "omitted",
			"secrets":                "omitted",
			"prometheus_pause":       paused,
			"resume_replays_skipped": false,
			"external_model_access":  "disabled unless separately implemented and authorized",
		},
		Sources: map[string]string{
			"configuration": "resolved config/pantheon.json schema version 1",
			"runtime":       "live daemon process and local SQLite state",
			"components":    "Kernel 20 canonical inventory",
		},
	}, nil
}

func (d *daemon) systemIntakeSummary() (map[string]any, error) {
	var present, pendingChronicle, failedScans int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences WHERE state = 'present'`).Scan(&present); err != nil {
		return nil, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences WHERE chronicle_state = 'pending'`).Scan(&pendingChronicle); err != nil {
		return nil, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_scans WHERE outcome = 'completed_with_errors'`).Scan(&failedScans); err != nil {
		return nil, err
	}
	var latestScanAt string
	_ = d.db.QueryRow(`SELECT COALESCE(ended_at, started_at) FROM intake_scans ORDER BY started_at DESC LIMIT 1`).Scan(&latestScanAt)
	return map[string]any{"present_occurrences": present, "pending_chronicle_writes": pendingChronicle, "failed_scans": failedScans, "latest_activity_at": latestScanAt}, nil
}

func (d *daemon) systemRumorSummary() (map[string]any, error) {
	counts, err := d.systemStateCounts("rumor_sources", "state")
	if err != nil {
		return nil, err
	}
	var pendingHandoffs int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM rumor_sources WHERE state = 'completed' AND intake_state = 'pending'`).Scan(&pendingHandoffs); err != nil {
		return nil, err
	}
	return map[string]any{"enabled": d.cfg.RumorEnabled, "state_counts": counts, "pending_intake_handoffs": pendingHandoffs, "watch_path_count": len(d.cfg.RumorWatchPaths), "live_transcription": false}, nil
}

func (d *daemon) systemApolloSummary() (map[string]any, error) {
	var pendingOrders, completedReports int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM orders WHERE recipient = 'apollo' AND status IN (?, ?, ?, ?, ?)`, orderStatusPending, orderStatusSealed, orderStatusReady, orderStatusWaiting, orderStatusRunning).Scan(&pendingOrders); err != nil {
		return nil, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM apollo_reports WHERE status = 'completed'`).Scan(&completedReports); err != nil {
		return nil, err
	}
	return map[string]any{"pending_orders": pendingOrders, "completed_reports": completedReports, "citation_required": d.cfg.ApolloRequireCitations, "external_research_enabled": d.cfg.ApolloExternalResearchEnabled}, nil
}

func (d *daemon) systemStateCounts(table, column string) (map[string]int, error) {
	// table and column are internal constants selected only by this file.
	rows, err := d.db.Query("SELECT " + column + ", COUNT(*) FROM " + table + " GROUP BY " + column)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var state string
		var count int
		if err := rows.Scan(&state, &count); err != nil {
			return nil, err
		}
		counts[state] = count
	}
	return counts, rows.Err()
}

func (d *daemon) systemBifrostSummary() (map[string]any, error) {
	capabilities, err := d.listBifrostCapabilities()
	if err != nil {
		return nil, err
	}
	counts := map[string]int{"registered": len(capabilities)}
	for _, capability := range capabilities {
		state := "disabled"
		if capability.Enabled {
			state = "enabled"
		}
		counts[state]++
	}
	return map[string]any{"enabled": d.cfg.BifrostEnabled, "capability_counts": counts, "mail_account": d.cfg.BifrostMailAccount, "mailbox": d.cfg.BifrostMailMailbox, "mail_schedule": d.cfg.BifrostMailSchedule, "mail_max_messages": d.cfg.BifrostMailMaxMessages, "download_install_engine": false}, nil
}

func (d *daemon) systemMailSummary() (map[string]any, error) {
	if err := initializeMailSchema(d.db); err != nil {
		return nil, err
	}
	counts := map[string]int{}
	rows, err := d.db.Query(`SELECT state, COUNT(*) FROM mail_messages GROUP BY state`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var state string
		var count int
		if err := rows.Scan(&state, &count); err != nil {
			rows.Close()
			return nil, err
		}
		counts[state] = count
	}
	rowErr := rows.Err()
	rows.Close()
	var cursor string
	_ = d.db.QueryRow(`SELECT COALESCE(cursor_received_at, '') FROM mail_backlog_state WHERE account = ? AND inbox = ?`, d.cfg.MailAccount, d.cfg.MailInbox).Scan(&cursor)
	return map[string]any{"enabled": d.cfg.MailEnabled, "adapter": d.cfg.MailAdapter, "account": d.cfg.MailAccount, "inbox": d.cfg.MailInbox, "state_counts": counts, "backlog_enabled": d.cfg.MailBacklogEnabled, "backlog_batch_size": d.cfg.MailBatchSize, "backlog_cursor": cursor, "mark_read_policy": d.cfg.MailMarkReadPolicy, "attachment_policy": d.cfg.MailAttachmentPolicy}, rowErr
}

func (d *daemon) zodiacInventorySummary() (map[string]any, error) {
	if err := initializeZodiacSchema(d.db); err != nil {
		return nil, err
	}
	counts := map[string]int{}
	rows, err := d.db.Query(`SELECT state, COUNT(*) FROM zodiac_instances GROUP BY state`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var state string
		var count int
		if err := rows.Scan(&state, &count); err != nil {
			rows.Close()
			return nil, err
		}
		counts[state] = count
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	var schedules, projections, needsYou int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM zodiac_schedules WHERE enabled = 1`).Scan(&schedules); err != nil {
		return nil, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM zodiac_projections`).Scan(&projections); err != nil {
		return nil, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM zodiac_instances WHERE state = ?`, zodiacInstanceNeedsYou).Scan(&needsYou); err != nil {
		return nil, err
	}
	return map[string]any{"enabled": d.cfg.ZodiacEnabled, "timezone": d.cfg.ZodiacDefaultTimezone, "active_schedules": schedules, "instances_by_state": counts, "needs_you": needsYou, "projection_count": projections, "wake_seconds": d.cfg.ZodiacWakeSeconds}, nil
}

func (d *daemon) systemFailureSummary() (map[string]any, error) {
	var failures, auditEvents int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE outcome IN ('failed', 'refused', 'blocked')`).Scan(&failures); err != nil {
		return nil, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM audit_events`).Scan(&auditEvents); err != nil {
		return nil, err
	}
	var latestFailureAt string
	_ = d.db.QueryRow(`SELECT occurred_at FROM audit_events WHERE outcome IN ('failed', 'refused', 'blocked') ORDER BY occurred_at DESC LIMIT 1`).Scan(&latestFailureAt)
	return map[string]any{"audit_events": auditEvents, "failed_or_blocked": failures, "latest_failure_at": latestFailureAt}, nil
}

func (d *daemon) handleSystemInventory(w http.ResponseWriter, r *http.Request) {
	inspection, err := d.systemInventory()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, inspection)
}
