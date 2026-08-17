package pantheon

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type organismRunRequest struct {
	OccurrenceID string `json:"occurrence_id,omitempty"`
	RequestText  string `json:"request_text"`
	PackageCode  string `json:"package_code"`
	Scope        string `json:"scope"`
	Missive      string `json:"missive"`
	JanusSubject string `json:"janus_subject_id,omitempty"`
	Purpose      string `json:"purpose,omitempty"`
}

type organismTrace struct {
	Input          string `json:"input"`
	Present        string `json:"present"`
	Past           string `json:"past"`
	ApolloOrderID  string `json:"apollo_order_id,omitempty"`
	ApolloReportID string `json:"apollo_report_id,omitempty"`
	SeshatCode     string `json:"seshat_code,omitempty"`
	SeshatDigest   string `json:"seshat_missive_sha256,omitempty"`
	Janus          string `json:"janus,omitempty"`
	FutureProposal string `json:"future_proposal_id,omitempty"`
	FutureState    string `json:"future_state,omitempty"`
	MercuryOrderID string `json:"mercury_order_id,omitempty"`
	MercuryStatus  string `json:"mercury_status,omitempty"`
	Evidence       string `json:"evidence,omitempty"`
}

type organismRun struct {
	RunID     string        `json:"run_id"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
	State     string        `json:"state"`
	Trace     organismTrace `json:"trace"`
	LastError string        `json:"last_error,omitempty"`
}

func initializeOrganismSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS organism_runs (
		run_id TEXT PRIMARY KEY, created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
		state TEXT NOT NULL, trace_json TEXT NOT NULL, last_error TEXT
	)`)
	return err
}

func (d *daemon) persistOrganismRun(run organismRun) error {
	trace, err := json.Marshal(run.Trace)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`INSERT INTO organism_runs (run_id, created_at, updated_at, state, trace_json, last_error) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(run_id) DO UPDATE SET updated_at=excluded.updated_at, state=excluded.state, trace_json=excluded.trace_json, last_error=excluded.last_error`, run.RunID, run.CreatedAt, run.UpdatedAt, run.State, string(trace), nullableString(run.LastError))
	return err
}

func (d *daemon) runOrganism(ctx context.Context, request organismRunRequest) (organismRun, error) {
	if err := initializeOrganismSchema(d.db); err != nil {
		return organismRun{}, err
	}
	request.RequestText = strings.TrimSpace(request.RequestText)
	request.PackageCode = strings.ToUpper(strings.TrimSpace(request.PackageCode))
	request.Scope = strings.TrimSpace(request.Scope)
	request.Missive = strings.TrimSpace(request.Missive)
	if request.RequestText == "" || request.PackageCode == "" || request.Scope == "" || request.Missive == "" {
		return organismRun{}, errors.New("request_text, package_code, scope, and missive are required")
	}
	if _, err := parseSeshatMissive(request.Missive); err != nil {
		return organismRun{}, fmt.Errorf("Seshat missive: %w", err)
	}
	runID, err := newLocalID("organism-")
	if err != nil {
		return organismRun{}, err
	}
	now, err := fatherTimeNow(d.cfg)
	if err != nil {
		return organismRun{}, err
	}
	run := organismRun{RunID: runID, CreatedAt: now, UpdatedAt: now, State: "running", Trace: organismTrace{Input: "direct request"}}
	if request.OccurrenceID != "" {
		run.Trace.Input = "Intake occurrence"
	}
	if err := d.persistOrganismRun(run); err != nil {
		return organismRun{}, err
	}
	fail := func(runErr error) (organismRun, error) {
		run.State, run.LastError = "failed", runErr.Error()
		run.UpdatedAt = time.Now().Format(time.RFC3339Nano)
		_ = d.persistOrganismRun(run)
		return run, runErr
	}
	if paused, pauseErr := d.paused(); pauseErr != nil {
		return fail(pauseErr)
	} else if paused {
		return fail(errors.New("Prometheus is paused; organism run did not start"))
	}

	var occurrenceID string
	if request.OccurrenceID != "" {
		if _, err := d.runIntakeScan(); err != nil {
			return fail(fmt.Errorf("Intake scan: %w", err))
		}
		occurrenceID = request.OccurrenceID
		if _, err := d.loadIntakeOccurrence(occurrenceID); err != nil {
			return fail(err)
		}
		present, err := d.processOccurrence(ctx, occurrenceID, request.RequestText)
		if err != nil {
			return fail(fmt.Errorf("Present: %w", err))
		}
		if present.Error != "" {
			return fail(errors.New("Present: " + present.Error))
		}
		run.Trace.Present = present.NormalizedPath
		run.Trace.Past = "Past/Weave prepared by Apollo order"
	} else {
		run.Trace.Present = "skipped: direct human request has no Intake occurrence"
		run.Trace.Past = "skipped: no occurrence-local source was supplied"
	}
	if err := d.persistOrganismRun(run); err != nil {
		return fail(err)
	}

	var apolloPayload json.RawMessage
	if occurrenceID != "" {
		apolloPayload, _ = json.Marshal(map[string]string{"occurrence_id": occurrenceID})
	} else {
		apolloPayload, _ = json.Marshal(apolloOrderPayload{PreparedContext: request.RequestText})
	}
	apolloOrder, err := d.createOrder(orderRequest{RequestText: request.RequestText, Recipient: "apollo", Capability: apolloCapability, Scope: "organism:" + run.RunID, Payload: apolloPayload, Purpose: "Kernel 17 whole-animal preparation"}, true)
	if err != nil {
		return fail(fmt.Errorf("Apollo order: %w", err))
	}
	run.Trace.ApolloOrderID = apolloOrder.OrderID
	if _, err := d.mercuryRound(ctx); err != nil {
		return fail(fmt.Errorf("Apollo Mercury round: %w", err))
	}
	apolloOrder, err = d.loadOrder(apolloOrder.OrderID)
	if err != nil || apolloOrder.Status != orderStatusCompleted {
		if err == nil {
			err = fmt.Errorf("Apollo order ended in %s: %s", apolloOrder.Status, apolloOrder.LastError)
		}
		return fail(err)
	}
	if err := d.db.QueryRow(`SELECT report_id FROM apollo_reports WHERE order_id = ? ORDER BY created_at DESC LIMIT 1`, apolloOrder.OrderID).Scan(&run.Trace.ApolloReportID); err != nil {
		return fail(fmt.Errorf("Apollo report: %w", err))
	}

	pkg, err := d.seshat.packageFor(request.PackageCode)
	if err != nil {
		return fail(fmt.Errorf("Seshat: %w", err))
	}
	normalized, err := constructSeshatPackage(pkg, request.Missive)
	if err != nil {
		return fail(fmt.Errorf("Seshat: %w", err))
	}
	run.Trace.SeshatCode, run.Trace.SeshatDigest = normalized.SubjectCode, normalized.SourceMissiveHash
	if pkg.Temporal == nil {
		run.Trace.Janus = "not required: package has no temporal contract"
	} else {
		if request.JanusSubject == "" {
			return fail(errors.New("Janus: temporal package requires janus_subject_id"))
		}
		subject, subjectErr := d.loadJanusSubject(request.JanusSubject)
		if subjectErr != nil {
			return fail(fmt.Errorf("Janus: %w", subjectErr))
		}
		if subject.PackageCode != pkg.SubjectCode {
			return fail(errors.New("Janus: subject package does not match Seshat package"))
		}
		run.Trace.Janus = subject.State
	}
	payload, err := json.Marshal(normalized.Fields)
	if err != nil {
		return fail(err)
	}
	proposal, err := d.createFutureProposal(futureProposalRequest{SourceReportID: run.Trace.ApolloReportID, SourceMissive: request.Missive, PackageCode: normalized.SubjectCode, Capability: normalized.Capability, Scope: request.Scope, Adapter: normalized.Adapter, Payload: payload, Purpose: request.Purpose, JanusSubjectID: request.JanusSubject})
	if err != nil {
		return fail(fmt.Errorf("Future proposal: %w", err))
	}
	run.Trace.FutureProposal = proposal.ProposalID
	if _, err := d.futureRound(ctx); err != nil {
		return fail(fmt.Errorf("Future round: %w", err))
	}
	proposal, err = d.loadFutureProposal(proposal.ProposalID)
	if err != nil {
		return fail(err)
	}
	run.Trace.FutureState = proposal.State
	if proposal.State != futureStateSealed {
		run.State = "awaiting_authority"
		run.UpdatedAt = time.Now().Format(time.RFC3339Nano)
		if err := d.persistOrganismRun(run); err != nil {
			return run, err
		}
		return run, nil
	}
	if proposal.OrderID == "" {
		return fail(errors.New("Future sealed without a Mercury order"))
	}
	run.Trace.MercuryOrderID = proposal.OrderID
	if _, err := d.mercuryRound(ctx); err != nil {
		return fail(fmt.Errorf("final Mercury round: %w", err))
	}
	order, err := d.loadOrder(proposal.OrderID)
	if err != nil {
		return fail(err)
	}
	run.Trace.MercuryStatus = order.Status
	run.Trace.Evidence = "order " + order.OrderID + "; Future proposal " + proposal.ProposalID + "; Apollo report " + run.Trace.ApolloReportID
	run.State = "completed"
	run.UpdatedAt = time.Now().Format(time.RFC3339Nano)
	if err := d.persistOrganismRun(run); err != nil {
		return run, err
	}
	if order.Status != orderStatusCompleted {
		return run, fmt.Errorf("Mercury order ended in %s: %s", order.Status, order.LastError)
	}
	return run, nil
}

func (d *daemon) handleOrganismRun(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("whole-animal run requires the local CLI control boundary"))
		return
	}
	var request organismRunRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	run, err := d.runOrganism(ctx, request)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, run)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (d *daemon) handleShowOrganism(w http.ResponseWriter, r *http.Request) {
	if err := initializeOrganismSchema(d.db); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	rows, err := d.db.Query(`SELECT run_id, created_at, updated_at, state, trace_json, COALESCE(last_error, '') FROM organism_runs ORDER BY created_at DESC LIMIT 20`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	result := []organismRun{}
	for rows.Next() {
		var run organismRun
		var trace string
		if err := rows.Scan(&run.RunID, &run.CreatedAt, &run.UpdatedAt, &run.State, &trace, &run.LastError); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := json.Unmarshal([]byte(trace), &run.Trace); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		result = append(result, run)
	}
	writeJSON(w, http.StatusOK, result)
}
