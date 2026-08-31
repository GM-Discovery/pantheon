package pantheon

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	actionKindCalendar         = "calendar_event"
	actionKindEmail            = "email_draft"
	actionKindClarification    = "needs_clarification"
	actionStateProposed        = "proposed"
	actionStateClarification   = "needs_clarification"
	actionStateAwaitingAuth    = "awaiting_authority"
	actionStateAwaitingMercury = "awaiting_mercury"
	actionStateProvider        = "provider_unavailable"
)

// actionDraft is the deterministic seam between Apollo's interpretation and
// Future's authority process. Its payload is a proposal, never an executable
// order. The original Apollo report remains the source of truth for why it
// exists.
type actionDraft struct {
	ActionID           string          `json:"action_id"`
	Kind               string          `json:"kind"`
	State              string          `json:"state"`
	SourceReportID     string          `json:"source_report_id"`
	SourceReportPath   string          `json:"source_report_path,omitempty"`
	SourceOccurrenceID string          `json:"source_occurrence_id,omitempty"`
	Summary            string          `json:"summary"`
	Payload            json.RawMessage `json:"payload,omitempty"`
	SourceCitations    []string        `json:"source_citations,omitempty"`
	FutureProposalID   string          `json:"future_proposal_id,omitempty"`
	LastError          string          `json:"last_error,omitempty"`
	CreatedAt          string          `json:"created_at"`
	UpdatedAt          string          `json:"updated_at"`
}

func initializeActionDraftSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS action_drafts (
		action_id TEXT PRIMARY KEY, kind TEXT NOT NULL, state TEXT NOT NULL,
		source_report_id TEXT NOT NULL, source_report_path TEXT, source_occurrence_id TEXT,
		summary TEXT NOT NULL, payload_json TEXT NOT NULL, source_citations_json TEXT NOT NULL,
		future_proposal_id TEXT, last_error TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
		UNIQUE(source_report_id, kind, payload_json)
	)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS action_drafts_state_idx ON action_drafts(state, updated_at DESC)`)
	return err
}

func actionDraftID(reportID, kind string, payload json.RawMessage) string {
	digest := sha256.Sum256(append([]byte(reportID+"\x00"+kind+"\x00"), payload...))
	return "action-" + hex.EncodeToString(digest[:])[:24]
}

func actionMapValue(payload map[string]any, key string) (any, bool) {
	if value, ok := payload[key]; ok {
		return value, true
	}
	for candidate, value := range payload {
		if strings.EqualFold(strings.ReplaceAll(candidate, "-", "_"), key) {
			return value, true
		}
	}
	return nil, false
}

func actionValueText(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), true
	case []string:
		return strings.TrimSpace(strings.Join(typed, ", ")), true
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return "", false
			}
			values = append(values, strings.TrimSpace(text))
		}
		return strings.TrimSpace(strings.Join(values, ", ")), true
	default:
		return "", false
	}
}

func actionPayloadFields(payload json.RawMessage) (map[string]string, error) {
	var values map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, fmt.Errorf("action payload must be an object: %w", err)
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		text, ok := actionValueText(value)
		if !ok {
			return nil, fmt.Errorf("action field %q must be text or a list of text", key)
		}
		if text != "" {
			result[strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))] = text
		}
	}
	return result, nil
}

func actionMissingFields(fields map[string]string, required []string) []string {
	missing := make([]string, 0)
	for _, field := range required {
		if strings.TrimSpace(fields[field]) == "" {
			missing = append(missing, field)
		}
	}
	return missing
}

func actionDraftSummary(kind string, payload json.RawMessage, fallback string) string {
	var values map[string]any
	_ = json.Unmarshal(payload, &values)
	keys := []string{"title", "subject", "summary", "body", "requested_action"}
	for _, key := range keys {
		if value, ok := actionMapValue(values, key); ok {
			if text, ok := actionValueText(value); ok && text != "" {
				if len(text) > 240 {
					text = text[:240] + "…"
				}
				return text
			}
		}
	}
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	switch kind {
	case actionKindCalendar:
		return "Calendar event proposal from Apollo"
	case actionKindEmail:
		return "Email draft proposal from Apollo"
	default:
		return "Apollo action requires clarification"
	}
}

func actionDraftState(kind string, payload json.RawMessage) (string, string) {
	fields, err := actionPayloadFields(payload)
	if err != nil {
		return actionStateClarification, err.Error()
	}
	required := []string{}
	switch kind {
	case actionKindCalendar:
		required = []string{"title", "start", "end"}
	case actionKindEmail:
		required = []string{"to", "subject", "body"}
	default:
		return actionStateClarification, "Apollo did not provide a supported action type"
	}
	missing := actionMissingFields(fields, required)
	if len(missing) > 0 {
		return actionStateClarification, "missing required fields: " + strings.Join(missing, ", ")
	}
	return actionStateProposed, ""
}

func (d *daemon) createActionDraft(draft actionDraft) (actionDraft, error) {
	if err := initializeActionDraftSchema(d.db); err != nil {
		return actionDraft{}, err
	}
	draft.SourceReportID = strings.TrimSpace(draft.SourceReportID)
	draft.Kind = strings.TrimSpace(draft.Kind)
	draft.Summary = strings.TrimSpace(draft.Summary)
	if draft.SourceReportID == "" || draft.Kind == "" || len(draft.Payload) == 0 || !json.Valid(draft.Payload) {
		return actionDraft{}, errors.New("action draft requires a source report, kind, and valid payload")
	}
	if draft.ActionID == "" {
		draft.ActionID = actionDraftID(draft.SourceReportID, draft.Kind, draft.Payload)
	}
	if draft.Summary == "" {
		draft.Summary = actionDraftSummary(draft.Kind, draft.Payload, "")
	}
	if draft.State == "" {
		draft.State, draft.LastError = actionDraftState(draft.Kind, draft.Payload)
	}
	if draft.SourceCitations == nil {
		draft.SourceCitations = []string{}
	}
	if draft.CreatedAt == "" {
		draft.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	draft.UpdatedAt = draft.CreatedAt
	citations, err := json.Marshal(draft.SourceCitations)
	if err != nil {
		return actionDraft{}, err
	}
	_, err = d.db.Exec(`INSERT INTO action_drafts (action_id, kind, state, source_report_id, source_report_path, source_occurrence_id, summary, payload_json, source_citations_json, future_proposal_id, last_error, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(action_id) DO UPDATE SET state = CASE WHEN COALESCE(action_drafts.future_proposal_id, '') <> '' THEN action_drafts.state ELSE excluded.state END, source_report_path = excluded.source_report_path, source_occurrence_id = excluded.source_occurrence_id, summary = excluded.summary, source_citations_json = excluded.source_citations_json, last_error = CASE WHEN COALESCE(action_drafts.future_proposal_id, '') <> '' THEN action_drafts.last_error ELSE excluded.last_error END, updated_at = excluded.updated_at`, draft.ActionID, draft.Kind, draft.State, draft.SourceReportID, nullableString(draft.SourceReportPath), nullableString(draft.SourceOccurrenceID), draft.Summary, string(draft.Payload), string(citations), nullableString(draft.FutureProposalID), nullableString(draft.LastError), draft.CreatedAt, draft.UpdatedAt)
	if err != nil {
		return actionDraft{}, err
	}
	return d.loadActionDraft(draft.ActionID)
}

func (d *daemon) routeApolloActions(reportID, reportPath, occurrenceID string, result futureReadyResult) ([]actionDraft, error) {
	if result.Status != "complete" {
		return nil, nil
	}
	var drafts []actionDraft
	add := func(kind string, payload any, fallback string) error {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		state, reason := actionDraftState(kind, data)
		draft, err := d.createActionDraft(actionDraft{Kind: kind, State: state, SourceReportID: reportID, SourceReportPath: reportPath, SourceOccurrenceID: occurrenceID, Summary: actionDraftSummary(kind, data, fallback), Payload: data, SourceCitations: append([]string(nil), result.SourceCitations...), LastError: reason})
		if err != nil {
			return err
		}
		drafts = append(drafts, draft)
		return nil
	}
	if len(result.MeetingProposal) > 0 {
		if err := add(actionKindCalendar, result.MeetingProposal, ""); err != nil {
			return drafts, err
		}
	}
	if len(result.DraftRecommendation) > 0 {
		if err := add(actionKindEmail, result.DraftRecommendation, ""); err != nil {
			return drafts, err
		}
	}
	if len(drafts) == 0 && strings.TrimSpace(result.RequestedAction) != "" {
		if err := add(actionKindClarification, map[string]string{"requested_action": result.RequestedAction}, result.RequestedAction); err != nil {
			return drafts, err
		}
	}
	return drafts, nil
}

func (d *daemon) loadActionDraft(id string) (actionDraft, error) {
	if err := initializeActionDraftSchema(d.db); err != nil {
		return actionDraft{}, err
	}
	var draft actionDraft
	var reportPath, occurrenceID, payload, citations, proposalID, lastError sql.NullString
	err := d.db.QueryRow(`SELECT action_id, kind, state, source_report_id, COALESCE(source_report_path, ''), COALESCE(source_occurrence_id, ''), summary, payload_json, source_citations_json, COALESCE(future_proposal_id, ''), COALESCE(last_error, ''), created_at, updated_at FROM action_drafts WHERE action_id = ?`, id).Scan(&draft.ActionID, &draft.Kind, &draft.State, &draft.SourceReportID, &reportPath, &occurrenceID, &draft.Summary, &payload, &citations, &proposalID, &lastError, &draft.CreatedAt, &draft.UpdatedAt)
	if err != nil {
		return actionDraft{}, err
	}
	draft.SourceReportPath, draft.SourceOccurrenceID, draft.FutureProposalID, draft.LastError = reportPath.String, occurrenceID.String, proposalID.String, lastError.String
	draft.Payload = json.RawMessage(payload.String)
	if citations.Valid {
		_ = json.Unmarshal([]byte(citations.String), &draft.SourceCitations)
	}
	return draft, nil
}

func (d *daemon) listActionDrafts(limit int) ([]actionDraft, error) {
	if err := initializeActionDraftSchema(d.db); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, err := d.db.Query(`SELECT action_id FROM action_drafts ORDER BY updated_at DESC, action_id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []actionDraft
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		draft, err := d.loadActionDraft(id)
		if err != nil {
			return nil, err
		}
		result = append(result, draft)
	}
	return result, rows.Err()
}

func (d *daemon) actionDraftInspection() (map[string]any, error) {
	drafts, err := d.listActionDrafts(100)
	if err != nil {
		return nil, err
	}
	jobs, err := d.actionJobInspection()
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, draft := range drafts {
		counts[draft.State]++
	}
	return map[string]any{
		"state_counts": counts,
		"drafts":       drafts,
		"jobs":         jobs,
		"contract": map[string]any{
			"source":            "Apollo future-ready result",
			"authority":         "Future",
			"execution":         "Mercury",
			"calendar_provider": "not installed by this boundary",
			"email_send":        "not implemented",
		},
	}, nil
}

func actionDraftMissive(pkg seshatPackage, fields map[string]string, summary string) string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := []string{"TYPE: " + pkg.SubjectCode}
	body := ""
	for _, key := range keys {
		source := pkg.PayloadRules[key]
		if strings.EqualFold(source, "body") || key == "body" {
			body = fields[key]
			continue
		}
		if source == "" {
			source = strings.ToUpper(strings.ReplaceAll(key, "_", "-"))
		}
		lines = append(lines, source+": "+fields[key])
	}
	if body == "" {
		body = summary
	}
	return strings.Join(lines, "\n") + "\n\n" + body
}

func (r *seshatRegistry) packageForCapability(capability string) (seshatPackage, error) {
	capability = strings.TrimSpace(capability)
	var found *seshatPackage
	for _, pkg := range r.packages {
		if !packageEnabled(pkg) || pkg.Capability != capability {
			continue
		}
		candidate := pkg
		if found != nil && found.SubjectCode != candidate.SubjectCode {
			return seshatPackage{}, fmt.Errorf("multiple enabled Seshat packages provide %q", capability)
		}
		found = &candidate
	}
	if found == nil {
		return seshatPackage{}, fmt.Errorf("no enabled Seshat package provides %q", capability)
	}
	return *found, nil
}

func actionStateFromFuture(proposal futureProposal) (string, string) {
	switch proposal.State {
	case futureStateSealed:
		return actionStateAwaitingMercury, "Future sealed a proposal; Mercury execution remains a separate explicit round"
	case futureStateNeedsAuthority, futureStateReferredPrometheus:
		return actionStateAwaitingAuth, proposal.LastError
	case futureStateAdapterUnavailable:
		return actionStateProvider, proposal.LastError
	case futureStateNeedsPerson:
		return actionStateAwaitingAuth, proposal.LastError
	case futureStateMalformed, futureStateDenied, futureStateTemporallyBlocked:
		return actionStateClarification, proposal.LastError
	default:
		return actionStateAwaitingAuth, proposal.LastError
	}
}

func (d *daemon) acceptActionDraft(ctx context.Context, id string) (actionDraft, error) {
	draft, err := d.loadActionDraft(id)
	if err != nil {
		return actionDraft{}, err
	}
	if draft.FutureProposalID != "" {
		return draft, nil
	}
	if draft.State != actionStateProposed {
		return draft, fmt.Errorf("action draft is %s: %s", draft.State, draft.LastError)
	}
	fields, err := actionPayloadFields(draft.Payload)
	if err != nil {
		return draft, err
	}
	capability := ""
	switch draft.Kind {
	case actionKindEmail:
		capability = "email.draft"
	case actionKindCalendar:
		capability = "calendar.create"
	default:
		return draft, errors.New("only calendar and email action drafts can be accepted")
	}
	pkg, err := d.seshat.packageForCapability(capability)
	if err != nil {
		state := actionStateProvider
		_, _ = d.db.Exec(`UPDATE action_drafts SET state = ?, last_error = ?, updated_at = ? WHERE action_id = ?`, state, err.Error(), time.Now().UTC().Format(time.RFC3339Nano), draft.ActionID)
		return d.loadActionDraft(draft.ActionID)
	}
	if missing := actionMissingFields(fields, pkg.RequiredFields); len(missing) > 0 {
		return draft, fmt.Errorf("action draft is missing package fields: %s", strings.Join(missing, ", "))
	}
	payload, err := json.Marshal(fields)
	if err != nil {
		return draft, err
	}
	missive := actionDraftMissive(pkg, fields, draft.Summary)
	proposal, err := d.createFutureProposal(futureProposalRequest{
		ProposalID:     "future-" + draft.ActionID,
		SourceReportID: draft.SourceReportID, SourceMissive: missive,
		PackageCode: pkg.SubjectCode, Capability: pkg.Capability,
		Scope: draft.Kind + ":" + draft.ActionID, Adapter: pkg.Adapter,
		Payload: payload, Purpose: "Accepted Apollo action draft",
	})
	if err != nil {
		return draft, err
	}
	if err := d.processFutureProposal(ctx, proposal, false, &futureRoundResponse{}); err != nil {
		return draft, err
	}
	proposal, err = d.loadFutureProposal(proposal.ProposalID)
	if err != nil {
		return draft, err
	}
	state, reason := actionStateFromFuture(proposal)
	_, err = d.db.Exec(`UPDATE action_drafts SET state = ?, future_proposal_id = ?, last_error = ?, updated_at = ? WHERE action_id = ?`, state, proposal.ProposalID, nullableString(reason), time.Now().UTC().Format(time.RFC3339Nano), draft.ActionID)
	if err != nil {
		return draft, err
	}
	return d.loadActionDraft(draft.ActionID)
}

func (d *daemon) handleShowActionDrafts(w http.ResponseWriter, r *http.Request) {
	inspection, err := d.actionDraftInspection()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, inspection)
}

func (d *daemon) handleShowActionDraft(w http.ResponseWriter, r *http.Request) {
	draft, err := d.loadActionDraft(r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("action draft not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

func (d *daemon) handleAcceptActionDraft(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("accepting an action draft requires the local control boundary"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	draft, err := d.acceptActionDraft(ctx, r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("action draft not found"))
		return
	}
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": err.Error(), "action": draft})
		return
	}
	writeJSON(w, http.StatusOK, draft)
}
