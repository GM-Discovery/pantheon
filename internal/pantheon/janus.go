package pantheon

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	janusStateScheduled        = "scheduled"
	janusStateReviewable       = "reviewable"
	janusStateActive           = "active"
	janusStateReady            = "ready"
	janusStateMissed           = "missed"
	janusStateExpired          = "expired"
	janusStateResurfaced       = "resurfaced"
	janusStateBackfillPending  = "backfill_pending"
	janusStateRecomputePending = "recompute_pending"
	janusStateReviewRequired   = "review_required"
	janusStateBlocked          = "blocked_error"

	janusThresholdNotBefore   = "not_before"
	janusThresholdNotAfter    = "not_after"
	janusThresholdReviewUntil = "review_until"
	janusThresholdExpiresAt   = "expires_at"
)

type janusSubject struct {
	SubjectID              string `json:"subject_id"`
	PackageCode            string `json:"package_code"`
	State                  string `json:"state"`
	Threshold              string `json:"threshold"`
	ThresholdKind          string `json:"threshold_kind"`
	CreatedAt              string `json:"created_at"`
	UpdatedAt              string `json:"updated_at"`
	SourceOrderID          string `json:"source_order_id,omitempty"`
	SourceProposalID       string `json:"source_proposal_id,omitempty"`
	SourceGrantID          string `json:"source_grant_id,omitempty"`
	RequestText            string `json:"request_text,omitempty"`
	LastError              string `json:"last_error,omitempty"`
	ResurfacedOccurrenceID string `json:"resurfaced_occurrence_id,omitempty"`
}

type janusTransition struct {
	TransitionID           string `json:"transition_id"`
	SubjectID              string `json:"subject_id"`
	PackageCode            string `json:"package_code"`
	FromState              string `json:"from_state"`
	ToState                string `json:"to_state"`
	Threshold              string `json:"threshold"`
	ThresholdAt            string `json:"threshold_at"`
	ObservedAt             string `json:"observed_at"`
	Policy                 string `json:"policy"`
	FatherTimeReference    string `json:"father_time_reference"`
	Reason                 string `json:"reason"`
	SourceOrderID          string `json:"source_order_id,omitempty"`
	SourceProposalID       string `json:"source_proposal_id,omitempty"`
	SourceGrantID          string `json:"source_grant_id,omitempty"`
	ResurfacedOccurrenceID string `json:"resurfaced_occurrence_id,omitempty"`
}

type janusSubjectRequest struct {
	SubjectID        string `json:"subject_id,omitempty"`
	PackageCode      string `json:"package_code"`
	Threshold        string `json:"threshold"`
	ThresholdKind    string `json:"threshold_kind"`
	State            string `json:"state,omitempty"`
	SourceOrderID    string `json:"source_order_id,omitempty"`
	SourceProposalID string `json:"source_proposal_id,omitempty"`
	SourceGrantID    string `json:"source_grant_id,omitempty"`
	RequestText      string `json:"request_text,omitempty"`
}

type janusRoundResponse struct {
	StartedAt           string   `json:"started_at"`
	ObservedAt          string   `json:"observed_at"`
	FatherTimeReference string   `json:"father_time_reference"`
	Paused              bool     `json:"paused"`
	Examined            int      `json:"examined"`
	Transitioned        int      `json:"transitioned"`
	Expired             int      `json:"expired"`
	Resurfaced          int      `json:"resurfaced"`
	Backfill            int      `json:"backfill"`
	Recompute           int      `json:"recompute"`
	ReviewRequired      int      `json:"review_required"`
	Deferred            int      `json:"deferred"`
	Failed              int      `json:"failed"`
	Errors              []string `json:"errors,omitempty"`
	Result              string   `json:"result"`
}

func initializeJanusSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS janus_subjects (
			subject_id TEXT PRIMARY KEY, package_code TEXT NOT NULL, state TEXT NOT NULL,
			threshold TEXT NOT NULL, threshold_kind TEXT NOT NULL, created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL, source_order_id TEXT, source_proposal_id TEXT,
			source_grant_id TEXT, request_text TEXT, last_error TEXT,
			resurfaced_occurrence_id TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS janus_subjects_state_idx ON janus_subjects(state, threshold)`,
		`CREATE TABLE IF NOT EXISTS janus_transitions (
			transition_id TEXT PRIMARY KEY, subject_id TEXT NOT NULL, package_code TEXT NOT NULL,
			from_state TEXT NOT NULL, to_state TEXT NOT NULL, threshold TEXT NOT NULL,
			threshold_at TEXT NOT NULL, observed_at TEXT NOT NULL, policy TEXT NOT NULL,
			father_time_reference TEXT NOT NULL, reason TEXT NOT NULL, source_order_id TEXT,
			source_proposal_id TEXT, source_grant_id TEXT, resurfaced_occurrence_id TEXT
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS janus_transition_identity_idx ON janus_transitions(subject_id, threshold_at, from_state, to_state)`,
		`CREATE INDEX IF NOT EXISTS janus_transitions_observed_idx ON janus_transitions(observed_at DESC)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func validateJanusThreshold(kind, value string) (time.Time, error) {
	switch kind {
	case janusThresholdNotBefore, janusThresholdNotAfter, janusThresholdReviewUntil, janusThresholdExpiresAt:
	default:
		return time.Time{}, fmt.Errorf("unsupported threshold_kind %q", kind)
	}
	threshold, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("threshold must be RFC3339: %w", err)
	}
	return threshold, nil
}

func (d *daemon) createJanusSubject(request janusSubjectRequest) (janusSubject, error) {
	if err := initializeJanusSchema(d.db); err != nil {
		return janusSubject{}, err
	}
	request.PackageCode = strings.ToUpper(strings.TrimSpace(request.PackageCode))
	pkg, err := d.seshat.packageFor(request.PackageCode)
	if err != nil {
		return janusSubject{}, err
	}
	if pkg.Temporal == nil {
		return janusSubject{}, fmt.Errorf("package %q has no temporal contract", request.PackageCode)
	}
	threshold, err := validateJanusThreshold(request.ThresholdKind, request.Threshold)
	if err != nil {
		return janusSubject{}, err
	}
	state := strings.TrimSpace(request.State)
	if state == "" {
		state = janusStateScheduled
	}
	if state != janusStateScheduled && state != janusStateReviewable && state != janusStateActive {
		return janusSubject{}, fmt.Errorf("unsupported initial Janus state %q", state)
	}
	subjectID := strings.TrimSpace(request.SubjectID)
	if subjectID == "" {
		subjectID, err = newLocalID("janus-")
		if err != nil {
			return janusSubject{}, err
		}
	}
	now, err := fatherTimeNow(d.cfg)
	if err != nil {
		return janusSubject{}, err
	}
	requestText := strings.TrimSpace(request.RequestText)
	if requestText == "" {
		requestText = "Janus temporal subject " + subjectID
	}
	subject := janusSubject{SubjectID: subjectID, PackageCode: request.PackageCode, State: state, Threshold: threshold.Format(time.RFC3339Nano), ThresholdKind: request.ThresholdKind, CreatedAt: now, UpdatedAt: now, SourceOrderID: strings.TrimSpace(request.SourceOrderID), SourceProposalID: strings.TrimSpace(request.SourceProposalID), SourceGrantID: strings.TrimSpace(request.SourceGrantID), RequestText: requestText}
	_, err = d.db.Exec(`INSERT INTO janus_subjects (subject_id, package_code, state, threshold, threshold_kind, created_at, updated_at, source_order_id, source_proposal_id, source_grant_id, request_text) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, subject.SubjectID, subject.PackageCode, subject.State, subject.Threshold, subject.ThresholdKind, subject.CreatedAt, subject.UpdatedAt, subject.SourceOrderID, subject.SourceProposalID, subject.SourceGrantID, subject.RequestText)
	if err != nil {
		return janusSubject{}, err
	}
	return subject, nil
}

func (d *daemon) loadJanusSubject(id string) (janusSubject, error) {
	var subject janusSubject
	err := d.db.QueryRow(`SELECT subject_id, package_code, state, threshold, threshold_kind, created_at, updated_at, COALESCE(source_order_id, ''), COALESCE(source_proposal_id, ''), COALESCE(source_grant_id, ''), COALESCE(request_text, ''), COALESCE(last_error, ''), COALESCE(resurfaced_occurrence_id, '') FROM janus_subjects WHERE subject_id = ?`, id).Scan(&subject.SubjectID, &subject.PackageCode, &subject.State, &subject.Threshold, &subject.ThresholdKind, &subject.CreatedAt, &subject.UpdatedAt, &subject.SourceOrderID, &subject.SourceProposalID, &subject.SourceGrantID, &subject.RequestText, &subject.LastError, &subject.ResurfacedOccurrenceID)
	return subject, err
}

func (d *daemon) recordJanusTransition(subject janusSubject, toState, policy, observedAt, fatherReference, reason, resurfacedID string) (bool, error) {
	transitionID, err := newLocalID("janus-transition-")
	if err != nil {
		return false, err
	}
	result, err := d.db.Exec(`INSERT OR IGNORE INTO janus_transitions (transition_id, subject_id, package_code, from_state, to_state, threshold, threshold_at, observed_at, policy, father_time_reference, reason, source_order_id, source_proposal_id, source_grant_id, resurfaced_occurrence_id) SELECT ?, subject_id, package_code, state, ?, threshold, threshold, ?, ?, ?, ?, source_order_id, source_proposal_id, source_grant_id, ? FROM janus_subjects WHERE subject_id = ?`, transitionID, toState, observedAt, policy, fatherReference, reason, resurfacedID, subject.SubjectID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if rows == 0 {
		return false, nil
	}
	if _, err := d.db.Exec(`UPDATE janus_subjects SET state = ?, updated_at = ?, last_error = '', resurfaced_occurrence_id = CASE WHEN ? <> '' THEN ? ELSE resurfaced_occurrence_id END WHERE subject_id = ?`, toState, observedAt, resurfacedID, resurfacedID, subject.SubjectID); err != nil {
		return false, err
	}
	return true, nil
}

func (d *daemon) markJanusBlocked(subject janusSubject, observedAt, fatherReference string, err error) error {
	_, recordErr := d.recordJanusTransition(subject, janusStateBlocked, "error", observedAt, fatherReference, err.Error(), "")
	if recordErr != nil {
		return recordErr
	}
	_, updateErr := d.db.Exec(`UPDATE janus_subjects SET last_error = ?, updated_at = ? WHERE subject_id = ?`, err.Error(), observedAt, subject.SubjectID)
	return updateErr
}

func (d *daemon) janusResurfacedOccurrence(subject janusSubject, observedAt string) (string, error) {
	if strings.TrimSpace(d.cfg.IntakePath) == "" {
		return "", errors.New("resurfacing requires a configured Intake path")
	}
	directory := filepath.Join(d.cfg.IntakePath, "Janus")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	filename := "janus-" + strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(subject.SubjectID) + ".md"
	path := filepath.Join(directory, filename)
	content := fmt.Sprintf("# Janus temporal transition\n\nThis known bounded subject crossed a deterministic temporal threshold and was resurfaced for human/Pantheon review.\n\n- Subject: `%s`\n- Package: `%s`\n- Threshold: `%s`\n- Observed: `%s`\n- Source order: `%s`\n\n%s\n", subject.SubjectID, subject.PackageCode, subject.Threshold, observedAt, subject.SourceOrderID, subject.RequestText)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	if _, err := d.runIntakeScan(); err != nil {
		return "", fmt.Errorf("scan resurfaced Intake occurrence: %w", err)
	}
	relative, err := filepath.Rel(d.cfg.IntakePath, path)
	if err != nil {
		return "", err
	}
	var occurrenceID string
	err = d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences WHERE relative_path = ? AND state = ? ORDER BY first_seen_at DESC LIMIT 1`, filepath.ToSlash(relative), occurrencePresent).Scan(&occurrenceID)
	if err != nil {
		return "", fmt.Errorf("find resurfaced Intake occurrence: %w", err)
	}
	return occurrenceID, nil
}

func (d *daemon) applyJanusPolicy(subject janusSubject, pkg seshatPackage, observedAt, fatherReference, policy string, response *janusRoundResponse) error {
	var finalState string
	switch policy {
	case "expire":
		finalState = janusStateExpired
	case "resurface":
		occurrenceID, err := d.janusResurfacedOccurrence(subject, observedAt)
		if err != nil {
			return err
		}
		finalState = janusStateResurfaced
		changed, err := d.recordJanusTransition(subject, finalState, policy, observedAt, fatherReference, "missed work resurfaced as a new Intake occurrence", occurrenceID)
		if err != nil {
			return err
		}
		if changed {
			response.Transitioned++
			response.Resurfaced++
		}
		return nil
	case "backfill":
		finalState = janusStateBackfillPending
	case "recompute":
		finalState = janusStateRecomputePending
	case "require_review":
		finalState = janusStateReviewRequired
	default:
		return fmt.Errorf("unknown temporal policy %q", policy)
	}
	changed, err := d.recordJanusTransition(subject, finalState, policy, observedAt, fatherReference, "missed work transitioned by Seshat temporal policy", "")
	if err != nil {
		return err
	}
	if changed {
		response.Transitioned++
		switch finalState {
		case janusStateExpired:
			response.Expired++
		case janusStateBackfillPending:
			response.Backfill++
		case janusStateRecomputePending:
			response.Recompute++
		case janusStateReviewRequired:
			response.ReviewRequired++
		}
	}
	return nil
}

func (d *daemon) processJanusSubject(ctx context.Context, subject janusSubject, now time.Time, paused bool, response *janusRoundResponse) error {
	_ = ctx
	pkg, err := d.seshat.packageFor(subject.PackageCode)
	if err != nil {
		return err
	}
	if pkg.Temporal == nil {
		return errors.New("package has no temporal contract")
	}
	threshold, err := validateJanusThreshold(subject.ThresholdKind, subject.Threshold)
	if err != nil {
		return err
	}
	if now.Before(threshold) {
		return nil
	}
	observedAt := now.Format(time.RFC3339Nano)
	fatherReference := "father-time:" + observedAt
	policy := pkg.Temporal.MissedPolicy
	if paused && pkg.Temporal.PausePolicy == "hold" {
		response.Deferred++
		return nil
	}
	if paused && pkg.Temporal.PausePolicy == "require_review" {
		policy = "require_review"
	}
	if subject.State == janusStateMissed {
		return d.applyJanusPolicy(subject, pkg, observedAt, fatherReference, policy, response)
	}
	if subject.ThresholdKind == janusThresholdNotBefore && !paused {
		changed, err := d.recordJanusTransition(subject, janusStateReady, "not_before", observedAt, fatherReference, "not_before threshold crossed; work is temporally eligible", "")
		if err != nil {
			return err
		}
		if changed {
			response.Transitioned++
		}
		return nil
	}
	changed, err := d.recordJanusTransition(subject, janusStateMissed, policy, observedAt, fatherReference, "temporal threshold crossed without execution", "")
	if err != nil {
		return err
	}
	if changed {
		response.Transitioned++
	}
	return d.applyJanusPolicy(janusSubject{SubjectID: subject.SubjectID, PackageCode: subject.PackageCode, State: janusStateMissed, Threshold: subject.Threshold, ThresholdKind: subject.ThresholdKind, SourceOrderID: subject.SourceOrderID, SourceProposalID: subject.SourceProposalID, SourceGrantID: subject.SourceGrantID, RequestText: subject.RequestText}, pkg, observedAt, fatherReference, policy, response)
}

func (d *daemon) janusRound(ctx context.Context) (janusRoundResponse, error) {
	if err := initializeJanusSchema(d.db); err != nil {
		return janusRoundResponse{}, err
	}
	loc, err := fatherTimeLocation(d.cfg)
	if err != nil {
		return janusRoundResponse{}, err
	}
	now := time.Now().In(loc)
	paused, err := d.paused()
	if err != nil {
		return janusRoundResponse{}, err
	}
	response := janusRoundResponse{StartedAt: now.Format(time.RFC3339Nano), ObservedAt: now.Format(time.RFC3339Nano), FatherTimeReference: "father-time:" + now.Format(time.RFC3339Nano), Paused: paused}
	rows, err := d.db.Query(`SELECT subject_id FROM janus_subjects WHERE state IN (?, ?, ?, ?) ORDER BY threshold, subject_id`, janusStateScheduled, janusStateReviewable, janusStateActive, janusStateMissed)
	if err != nil {
		return response, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return response, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return response, err
	}
	for _, id := range ids {
		response.Examined++
		subject, err := d.loadJanusSubject(id)
		if err != nil {
			return response, err
		}
		if err := d.processJanusSubject(ctx, subject, now, paused, &response); err != nil {
			response.Failed++
			response.Errors = append(response.Errors, subject.SubjectID+": "+err.Error())
			if blockErr := d.markJanusBlocked(subject, response.ObservedAt, response.FatherTimeReference, err); blockErr != nil {
				return response, blockErr
			}
		}
	}
	response.Result = fmt.Sprintf("Janus examined %d subject(s), transitioned %d, expired %d, resurfaced %d, backfill %d, recompute %d, review %d, deferred %d, failed %d.", response.Examined, response.Transitioned, response.Expired, response.Resurfaced, response.Backfill, response.Recompute, response.ReviewRequired, response.Deferred, response.Failed)
	return response, nil
}

func (d *daemon) janusInspection() (map[string]any, error) {
	if err := initializeJanusSchema(d.db); err != nil {
		return nil, err
	}
	loc, err := fatherTimeLocation(d.cfg)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	rows, err := d.db.Query(`SELECT state, COUNT(*) FROM janus_subjects GROUP BY state`)
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
	rows.Close()
	transitions, err := d.janusTransitions(20)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ready": true, "timezone": loc.String(), "father_time_reference": "father-time:" + time.Now().In(loc).Format(time.RFC3339Nano), "state_counts": counts, "recent_transitions": transitions}, nil
}

func (d *daemon) janusTransitions(limit int) ([]janusTransition, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := d.db.Query(`SELECT transition_id, subject_id, package_code, from_state, to_state, threshold, threshold_at, observed_at, policy, father_time_reference, reason, COALESCE(source_order_id, ''), COALESCE(source_proposal_id, ''), COALESCE(source_grant_id, ''), COALESCE(resurfaced_occurrence_id, '') FROM janus_transitions ORDER BY observed_at DESC, transition_id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []janusTransition
	for rows.Next() {
		var item janusTransition
		if err := rows.Scan(&item.TransitionID, &item.SubjectID, &item.PackageCode, &item.FromState, &item.ToState, &item.Threshold, &item.ThresholdAt, &item.ObservedAt, &item.Policy, &item.FatherTimeReference, &item.Reason, &item.SourceOrderID, &item.SourceProposalID, &item.SourceGrantID, &item.ResurfacedOccurrenceID); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *daemon) janusTransition(id string) (janusTransition, error) {
	var item janusTransition
	err := d.db.QueryRow(`SELECT transition_id, subject_id, package_code, from_state, to_state, threshold, threshold_at, observed_at, policy, father_time_reference, reason, COALESCE(source_order_id, ''), COALESCE(source_proposal_id, ''), COALESCE(source_grant_id, ''), COALESCE(resurfaced_occurrence_id, '') FROM janus_transitions WHERE transition_id = ?`, id).Scan(&item.TransitionID, &item.SubjectID, &item.PackageCode, &item.FromState, &item.ToState, &item.Threshold, &item.ThresholdAt, &item.ObservedAt, &item.Policy, &item.FatherTimeReference, &item.Reason, &item.SourceOrderID, &item.SourceProposalID, &item.SourceGrantID, &item.ResurfacedOccurrenceID)
	return item, err
}

func (d *daemon) handleJanusRound(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("Janus round requires the local CLI control boundary"))
		return
	}
	result, err := d.janusRound(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (d *daemon) handleCreateJanusSubject(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("Janus subject creation requires the local CLI control boundary"))
		return
	}
	var request janusSubjectRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	subject, err := d.createJanusSubject(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, subject)
}

func (d *daemon) handleShowJanus(w http.ResponseWriter, r *http.Request) {
	inspection, err := d.janusInspection()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, inspection)
}

func (d *daemon) handleShowJanusTransitions(w http.ResponseWriter, r *http.Request) {
	transitions, err := d.janusTransitions(100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, transitions)
}

func (d *daemon) handleShowJanusTransition(w http.ResponseWriter, r *http.Request) {
	transition, err := d.janusTransition(r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("Janus transition not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, transition)
}
