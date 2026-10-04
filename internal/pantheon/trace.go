package pantheon

import (
	"fmt"
	"net/http"
	"strings"
)

// Request tracing: replaying one request through every stage it touched.
//
// The correlation id made the grouping possible; this makes it readable.
// The practical problem with a correlation id is that when something
// breaks you do not know the id -- so listRecentFailures answers "what
// went wrong lately" and hands back the ids to trace.
//
// Reads only. Tracing must never be able to change what it is explaining.

// traceStage is one audited step of a request. Payloads are deliberately
// NOT included: the question this answers is where a request broke, which
// the endpoint, capability, outcome and error already answer, and
// audit_events' input/result hold personal content (email bodies, vault
// text) that should not be surfaced by a routine debugging command.
type traceStage struct {
	EventID    string `json:"event_id"`
	OccurredAt string `json:"occurred_at"`
	Endpoint   string `json:"endpoint"`
	Capability string `json:"capability,omitempty"`
	Model      string `json:"model,omitempty"`
	Outcome    string `json:"outcome"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

type requestTrace struct {
	RequestID string       `json:"request_id"`
	Stages    []traceStage `json:"stages"`
	Outcome   string       `json:"outcome"`
	Broke     string       `json:"broke_at,omitempty"`
}

// failureSummary points at a request worth tracing.
type failureSummary struct {
	RequestID  string `json:"request_id,omitempty"`
	EventID    string `json:"event_id"`
	OccurredAt string `json:"occurred_at"`
	Endpoint   string `json:"endpoint"`
	Capability string `json:"capability,omitempty"`
	Outcome    string `json:"outcome"`
	Error      string `json:"error,omitempty"`
	Traceable  bool   `json:"traceable"`
}

func (d *daemon) traceForRequest(requestID string) (requestTrace, error) {
	trace := requestTrace{RequestID: requestID, Stages: []traceStage{}}
	if strings.TrimSpace(requestID) == "" {
		return trace, fmt.Errorf("request id is required")
	}

	rows, err := d.db.Query(`SELECT id, occurred_at, endpoint, COALESCE(capability, ''),
		COALESCE(model, ''), outcome, COALESCE(error, ''), duration_ms
		FROM audit_events WHERE request_id = ? ORDER BY occurred_at ASC`, requestID)
	if err != nil {
		return trace, err
	}
	defer rows.Close()

	for rows.Next() {
		var stage traceStage
		if err := rows.Scan(&stage.EventID, &stage.OccurredAt, &stage.Endpoint,
			&stage.Capability, &stage.Model, &stage.Outcome, &stage.Error, &stage.DurationMS); err != nil {
			return trace, err
		}
		trace.Stages = append(trace.Stages, stage)
	}
	if err := rows.Err(); err != nil {
		return trace, err
	}

	// The terminal outcome is the request's outcome, and the first stage
	// that did not complete is where it broke. Reported separately because
	// a request can fail at an early stage and still record later ones.
	for _, stage := range trace.Stages {
		if trace.Broke == "" && !isHealthyOutcome(stage.Outcome) {
			trace.Broke = fmt.Sprintf("%s (%s) at %s", stage.Endpoint, stage.Outcome, stage.OccurredAt)
		}
		trace.Outcome = stage.Outcome
	}
	if len(trace.Stages) == 0 {
		trace.Outcome = "unknown"
	}
	return trace, nil
}

func isHealthyOutcome(outcome string) bool {
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case "completed", "started":
		return true
	default:
		return false
	}
}

// listRecentFailures returns the most recent non-healthy events, newest
// first. Events predating the correlation id (request_id NULL) are included
// with Traceable=false rather than hidden -- an old failure is still a
// failure, and silently omitting it would misrepresent the system's health.
func (d *daemon) listRecentFailures(limit int) ([]failureSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := d.db.Query(`SELECT COALESCE(request_id, ''), id, occurred_at, endpoint,
		COALESCE(capability, ''), outcome, COALESCE(error, '')
		FROM audit_events WHERE outcome NOT IN ('completed', 'started')
		ORDER BY occurred_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	failures := []failureSummary{}
	for rows.Next() {
		var f failureSummary
		if err := rows.Scan(&f.RequestID, &f.EventID, &f.OccurredAt, &f.Endpoint,
			&f.Capability, &f.Outcome, &f.Error); err != nil {
			return nil, err
		}
		f.Traceable = f.RequestID != ""
		failures = append(failures, f)
	}
	return failures, rows.Err()
}

func (d *daemon) handleTrace(w http.ResponseWriter, r *http.Request) {
	trace, err := d.traceForRequest(r.PathValue("request_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(trace.Stages) == 0 {
		writeJSON(w, http.StatusNotFound, apiError{fmt.Sprintf("no audited stages for request %q", trace.RequestID)})
		return
	}
	writeJSON(w, http.StatusOK, trace)
}

func (d *daemon) handleFailures(w http.ResponseWriter, r *http.Request) {
	failures, err := d.listRecentFailures(0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"failures": failures})
}
