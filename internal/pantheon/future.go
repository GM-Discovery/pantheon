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

const (
	futureStateAwaiting           = "awaiting"
	futureStateSealed             = "sealed"
	futureStateNeedsAuthority     = "needs_authority"
	futureStateReferredPrometheus = "referred_prometheus"
	futureStateNeedsPerson        = "needs_person"
	futureStateDenied             = "denied"
	futureStateTemporallyBlocked  = "temporally_blocked"
	futureStateMalformed          = "malformed"
	futureStateAdapterUnavailable = "adapter_unavailable"
)

type futureProposal struct {
	ProposalID              string          `json:"proposal_id"`
	CreatedAt               string          `json:"created_at"`
	UpdatedAt               string          `json:"updated_at"`
	State                   string          `json:"state"`
	SourceReportID          string          `json:"source_report_id,omitempty"`
	SourceMissive           string          `json:"source_missive,omitempty"`
	PackageCode             string          `json:"package_code"`
	Capability              string          `json:"capability"`
	Scope                   string          `json:"scope"`
	Adapter                 string          `json:"adapter"`
	Payload                 json.RawMessage `json:"payload,omitempty"`
	Purpose                 string          `json:"purpose,omitempty"`
	Constraints             json.RawMessage `json:"constraints,omitempty"`
	JanusSubjectID          string          `json:"janus_subject_id,omitempty"`
	RequestedExecutionBound string          `json:"requested_execution_bound,omitempty"`
	AuthorityRequestID      string          `json:"authority_request_id,omitempty"`
	AuthorityGrantID        string          `json:"authority_grant_id,omitempty"`
	OrderID                 string          `json:"order_id,omitempty"`
	LastError               string          `json:"last_error,omitempty"`
}

type futureProposalRequest struct {
	ProposalID              string          `json:"proposal_id,omitempty"`
	SourceReportID          string          `json:"source_report_id,omitempty"`
	SourceMissive           string          `json:"source_missive,omitempty"`
	PackageCode             string          `json:"package_code"`
	Capability              string          `json:"capability"`
	Scope                   string          `json:"scope"`
	Adapter                 string          `json:"adapter"`
	Payload                 json.RawMessage `json:"payload,omitempty"`
	Purpose                 string          `json:"purpose,omitempty"`
	Constraints             json.RawMessage `json:"constraints,omitempty"`
	JanusSubjectID          string          `json:"janus_subject_id,omitempty"`
	RequestedExecutionBound string          `json:"requested_execution_bound,omitempty"`
}

type futureAuthorityRequest struct {
	RequestID         string   `json:"request_id"`
	ProposalID        string   `json:"proposal_id"`
	Status            string   `json:"status"`
	PackageCode       string   `json:"package_code"`
	Capability        string   `json:"capability"`
	Scope             string   `json:"scope"`
	Purpose           string   `json:"purpose,omitempty"`
	SourceReportID    string   `json:"source_report_id,omitempty"`
	SourceMissive     string   `json:"source_missive,omitempty"`
	CurrentAuthority  string   `json:"current_authority"`
	AuthorityGap      string   `json:"authority_gap"`
	PermissionOptions []string `json:"permission_options"`
	PrometheusGrantID string   `json:"prometheus_grant_id,omitempty"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

type futureRoundResponse struct {
	StartedAt          string   `json:"started_at"`
	Paused             bool     `json:"paused"`
	Examined           int      `json:"examined"`
	Sealed             int      `json:"sealed"`
	NeedsAuthority     int      `json:"needs_authority"`
	Referred           int      `json:"referred_prometheus"`
	NeedsPerson        int      `json:"needs_person"`
	TemporallyBlocked  int      `json:"temporally_blocked"`
	Malformed          int      `json:"malformed"`
	AdapterUnavailable int      `json:"adapter_unavailable"`
	Deferred           int      `json:"deferred"`
	Failed             int      `json:"failed"`
	Errors             []string `json:"errors,omitempty"`
	Result             string   `json:"result"`
}

func initializeFutureSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS future_proposals (
			proposal_id TEXT PRIMARY KEY, created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
			state TEXT NOT NULL, source_report_id TEXT, source_missive TEXT,
			package_code TEXT NOT NULL, capability TEXT NOT NULL, scope TEXT NOT NULL,
			adapter TEXT NOT NULL, payload_json TEXT, purpose TEXT, constraints_json TEXT,
			janus_subject_id TEXT, requested_execution_bound TEXT, authority_request_id TEXT,
			authority_grant_id TEXT, order_id TEXT, last_error TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS future_proposals_state_idx ON future_proposals(state, created_at)`,
		`CREATE TABLE IF NOT EXISTS future_authority_requests (
			request_id TEXT PRIMARY KEY, proposal_id TEXT NOT NULL UNIQUE, status TEXT NOT NULL,
			package_code TEXT NOT NULL, capability TEXT NOT NULL, scope TEXT NOT NULL,
			purpose TEXT, source_report_id TEXT, source_missive TEXT,
			current_authority TEXT NOT NULL, authority_gap TEXT NOT NULL,
			permission_options_json TEXT NOT NULL, prometheus_grant_id TEXT,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS future_authority_requests_status_idx ON future_authority_requests(status, created_at)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func (d *daemon) createFutureProposal(request futureProposalRequest) (futureProposal, error) {
	if err := initializeFutureSchema(d.db); err != nil {
		return futureProposal{}, err
	}
	request.PackageCode = strings.ToUpper(strings.TrimSpace(request.PackageCode))
	request.Capability = strings.TrimSpace(request.Capability)
	request.Scope = strings.TrimSpace(request.Scope)
	request.Adapter = strings.TrimSpace(request.Adapter)
	if request.PackageCode == "" || request.Capability == "" || request.Scope == "" || request.Adapter == "" {
		return futureProposal{}, errors.New("package_code, capability, scope, and adapter are required")
	}
	if len(request.Payload) > 0 && !json.Valid(request.Payload) {
		return futureProposal{}, errors.New("payload must be valid JSON")
	}
	if len(request.Constraints) > 0 && !json.Valid(request.Constraints) {
		return futureProposal{}, errors.New("constraints must be valid JSON")
	}
	id := strings.TrimSpace(request.ProposalID)
	if id == "" {
		var err error
		id, err = newLocalID("proposal-")
		if err != nil {
			return futureProposal{}, err
		}
	}
	now, err := fatherTimeNow(d.cfg)
	if err != nil {
		return futureProposal{}, err
	}
	proposal := futureProposal{ProposalID: id, CreatedAt: now, UpdatedAt: now, State: futureStateAwaiting, SourceReportID: strings.TrimSpace(request.SourceReportID), SourceMissive: request.SourceMissive, PackageCode: request.PackageCode, Capability: request.Capability, Scope: request.Scope, Adapter: request.Adapter, Payload: request.Payload, Purpose: strings.TrimSpace(request.Purpose), Constraints: request.Constraints, JanusSubjectID: strings.TrimSpace(request.JanusSubjectID), RequestedExecutionBound: strings.TrimSpace(request.RequestedExecutionBound)}
	if pkg, packageErr := d.seshat.packageFor(proposal.PackageCode); packageErr != nil {
		proposal.State, proposal.LastError = futureStateMalformed, packageErr.Error()
	} else if pkg.Capability != proposal.Capability || pkg.Adapter != proposal.Adapter {
		proposal.State, proposal.LastError = futureStateMalformed, "proposal capability or adapter does not match Seshat package identity"
	} else if !pkg.AdapterAvailable {
		proposal.State, proposal.LastError = futureStateAdapterUnavailable, "Seshat package adapter is unavailable"
	}
	if _, err := d.db.Exec(`INSERT INTO future_proposals (proposal_id, created_at, updated_at, state, source_report_id, source_missive, package_code, capability, scope, adapter, payload_json, purpose, constraints_json, janus_subject_id, requested_execution_bound, last_error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, proposal.ProposalID, proposal.CreatedAt, proposal.UpdatedAt, proposal.State, nullableString(proposal.SourceReportID), nullableString(proposal.SourceMissive), proposal.PackageCode, proposal.Capability, proposal.Scope, proposal.Adapter, nullableJSON(proposal.Payload), nullableString(proposal.Purpose), nullableJSON(proposal.Constraints), nullableString(proposal.JanusSubjectID), nullableString(proposal.RequestedExecutionBound), nullableString(proposal.LastError)); err != nil {
		return futureProposal{}, err
	}
	return proposal, nil
}

func (d *daemon) loadFutureProposal(id string) (futureProposal, error) {
	var proposal futureProposal
	var sourceReport, sourceMissive, payload, purpose, constraints, janusID, bound, authorityRequest, authorityGrant, orderID, lastError sql.NullString
	err := d.db.QueryRow(`SELECT proposal_id, created_at, updated_at, state, source_report_id, source_missive, package_code, capability, scope, adapter, payload_json, purpose, constraints_json, janus_subject_id, requested_execution_bound, authority_request_id, authority_grant_id, order_id, last_error FROM future_proposals WHERE proposal_id = ?`, id).Scan(&proposal.ProposalID, &proposal.CreatedAt, &proposal.UpdatedAt, &proposal.State, &sourceReport, &sourceMissive, &proposal.PackageCode, &proposal.Capability, &proposal.Scope, &proposal.Adapter, &payload, &purpose, &constraints, &janusID, &bound, &authorityRequest, &authorityGrant, &orderID, &lastError)
	if err != nil {
		return futureProposal{}, err
	}
	proposal.SourceReportID, proposal.SourceMissive, proposal.Purpose, proposal.JanusSubjectID, proposal.RequestedExecutionBound = sourceReport.String, sourceMissive.String, purpose.String, janusID.String, bound.String
	proposal.AuthorityRequestID, proposal.AuthorityGrantID, proposal.OrderID, proposal.LastError = authorityRequest.String, authorityGrant.String, orderID.String, lastError.String
	if payload.Valid {
		proposal.Payload = json.RawMessage(payload.String)
	}
	if constraints.Valid {
		proposal.Constraints = json.RawMessage(constraints.String)
	}
	return proposal, nil
}

func (d *daemon) updateFutureProposal(proposalID, state, authorityRequestID, authorityGrantID, orderID, lastError string) error {
	updated, err := fatherTimeNow(d.cfg)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`UPDATE future_proposals SET state = ?, updated_at = ?, authority_request_id = CASE WHEN ? <> '' THEN ? ELSE authority_request_id END, authority_grant_id = CASE WHEN ? <> '' THEN ? ELSE authority_grant_id END, order_id = CASE WHEN ? <> '' THEN ? ELSE order_id END, last_error = ? WHERE proposal_id = ?`, state, updated, authorityRequestID, authorityRequestID, authorityGrantID, authorityGrantID, orderID, orderID, nullableString(lastError), proposalID)
	return err
}

func (d *daemon) futureActiveGrant(capability, scope string) (capabilityGrant, error) {
	rows, err := d.db.Query(`SELECT id FROM capability_grants WHERE status = ? AND holder = ? ORDER BY created_at DESC, id DESC`, grantStatusActive, "future")
	if err != nil {
		return capabilityGrant{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return capabilityGrant{}, err
		}
		grant, err := d.loadCapabilityGrant(id)
		if err != nil {
			continue
		}
		grant = d.refreshGrantState(grant, time.Now())
		if grantTimeValid(grant, time.Now()) && capabilityWithin(grant.Capability, capability) && scopeWithin(grant.Scope, scope) {
			return grant, nil
		}
	}
	return capabilityGrant{}, sql.ErrNoRows
}

func futurePermissionOptions(pkg seshatPackage) []string {
	allowed := map[string]bool{"once": true, "bounded_uses": true, "until_timestamp": true, "until_revoked": true}
	seen := map[string]bool{}
	var options []string
	for _, shape := range pkg.PermissionShapes {
		if allowed[shape] && !seen[shape] {
			seen[shape] = true
			options = append(options, shape)
		}
	}
	return options
}

func (d *daemon) futurePrometheusParent(capability, scope string) (capabilityGrant, error) {
	rows, err := d.db.Query(`SELECT id FROM capability_grants WHERE status = ? AND holder = ? ORDER BY created_at DESC, id DESC`, grantStatusActive, holderPrometheus)
	if err != nil {
		return capabilityGrant{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return capabilityGrant{}, err
		}
		grant, err := d.loadCapabilityGrant(id)
		if err != nil {
			continue
		}
		grant = d.refreshGrantState(grant, time.Now())
		if grantTimeValid(grant, time.Now()) && delegationAllowed(grant) && capabilityWithin(grant.Capability, capability) && scopeWithin(grant.Scope, scope) {
			return grant, nil
		}
	}
	return capabilityGrant{}, sql.ErrNoRows
}

func (d *daemon) createFuturePendingGrant(proposal futureProposal, parent capabilityGrant) (capabilityGrant, error) {
	if _, err := d.validateDelegatedGrant(grantRequest{Capability: proposal.Capability, Scope: proposal.Scope, ParentGrantID: parent.ID, RemainingUses: intPointer(1), ValidFrom: parent.ValidFrom, ValidUntil: parent.ValidUntil}); err != nil {
		return capabilityGrant{}, err
	}
	now, err := fatherTimeNow(d.cfg)
	if err != nil {
		return capabilityGrant{}, err
	}
	id := fmt.Sprintf("grant-future-%d", time.Now().UnixNano())
	grant := capabilityGrant{ID: id, Holder: "future", Capability: proposal.Capability, Scope: proposal.Scope, IntentSummary: "Future authority request for proposal " + proposal.ProposalID, HumanApproved: false, GrantedBy: holderPrometheus, ParentGrantID: parent.ID, ValidFrom: parent.ValidFrom, ValidUntil: parent.ValidUntil, RemainingUses: intPointer(1), Status: grantStatusPending, CreatedAt: now, UpdatedAt: now}
	_, err = d.db.Exec(`INSERT INTO capability_grants (id, holder, capability, scope, intent_summary, human_approved, granted_by, parent_grant_id, created_at, valid_from, valid_until, remaining_uses, status, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, grant.ID, grant.Holder, grant.Capability, grant.Scope, grant.IntentSummary, 0, grant.GrantedBy, grant.ParentGrantID, grant.CreatedAt, nullableString(grant.ValidFrom), nullableString(grant.ValidUntil), grant.RemainingUses, grant.Status, grant.UpdatedAt)
	return grant, err
}

func intPointer(value int) *int { return &value }

func (d *daemon) createFutureAuthorityRequest(proposal futureProposal, pkg seshatPackage) (futureAuthorityRequest, error) {
	var existingID string
	if err := d.db.QueryRow(`SELECT request_id FROM future_authority_requests WHERE proposal_id = ?`, proposal.ProposalID).Scan(&existingID); err == nil {
		return d.loadFutureAuthorityRequest(existingID)
	}
	parent, parentErr := d.futurePrometheusParent(proposal.Capability, proposal.Scope)
	if parentErr != nil && !errors.Is(parentErr, sql.ErrNoRows) {
		return futureAuthorityRequest{}, parentErr
	}
	status := futureStateNeedsPerson
	prometheusGrantID := ""
	if parentErr == nil {
		grant, err := d.createFuturePendingGrant(proposal, parent)
		if err != nil {
			return futureAuthorityRequest{}, err
		}
		status, prometheusGrantID = futureStateReferredPrometheus, grant.ID
	}
	requestID, err := newLocalID("authority-request-")
	if err != nil {
		return futureAuthorityRequest{}, err
	}
	now, err := fatherTimeNow(d.cfg)
	if err != nil {
		return futureAuthorityRequest{}, err
	}
	options, err := json.Marshal(futurePermissionOptions(pkg))
	if err != nil {
		return futureAuthorityRequest{}, err
	}
	request := futureAuthorityRequest{RequestID: requestID, ProposalID: proposal.ProposalID, Status: status, PackageCode: proposal.PackageCode, Capability: proposal.Capability, Scope: proposal.Scope, Purpose: proposal.Purpose, SourceReportID: proposal.SourceReportID, SourceMissive: proposal.SourceMissive, CurrentAuthority: "none", AuthorityGap: "no active Future grant covers this capability and scope", PermissionOptions: futurePermissionOptions(pkg), PrometheusGrantID: prometheusGrantID, CreatedAt: now, UpdatedAt: now}
	_, err = d.db.Exec(`INSERT INTO future_authority_requests (request_id, proposal_id, status, package_code, capability, scope, purpose, source_report_id, source_missive, current_authority, authority_gap, permission_options_json, prometheus_grant_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, request.RequestID, request.ProposalID, request.Status, request.PackageCode, request.Capability, request.Scope, nullableString(request.Purpose), nullableString(request.SourceReportID), nullableString(request.SourceMissive), request.CurrentAuthority, request.AuthorityGap, string(options), nullableString(request.PrometheusGrantID), request.CreatedAt, request.UpdatedAt)
	if err != nil {
		return futureAuthorityRequest{}, err
	}
	if err := d.updateFutureProposal(proposal.ProposalID, status, request.RequestID, "", "", "authority is not currently granted to Future"); err != nil {
		return futureAuthorityRequest{}, err
	}
	return request, nil
}

func (d *daemon) loadFutureAuthorityRequest(id string) (futureAuthorityRequest, error) {
	var request futureAuthorityRequest
	var purpose, sourceReport, sourceMissive, options, parent sql.NullString
	err := d.db.QueryRow(`SELECT request_id, proposal_id, status, package_code, capability, scope, purpose, source_report_id, source_missive, current_authority, authority_gap, permission_options_json, prometheus_grant_id, created_at, updated_at FROM future_authority_requests WHERE request_id = ?`, id).Scan(&request.RequestID, &request.ProposalID, &request.Status, &request.PackageCode, &request.Capability, &request.Scope, &purpose, &sourceReport, &sourceMissive, &request.CurrentAuthority, &request.AuthorityGap, &options, &parent, &request.CreatedAt, &request.UpdatedAt)
	if err != nil {
		return futureAuthorityRequest{}, err
	}
	request.Purpose, request.SourceReportID, request.SourceMissive, request.PrometheusGrantID = purpose.String, sourceReport.String, sourceMissive.String, parent.String
	if options.Valid {
		_ = json.Unmarshal([]byte(options.String), &request.PermissionOptions)
	}
	return request, nil
}

func (d *daemon) processFutureProposal(ctx context.Context, proposal futureProposal, paused bool, response *futureRoundResponse) error {
	_ = ctx
	pkg, err := d.seshat.packageFor(proposal.PackageCode)
	if err != nil {
		_ = d.updateFutureProposal(proposal.ProposalID, futureStateMalformed, "", "", "", err.Error())
		response.Malformed++
		return nil
	}
	if pkg.Capability != proposal.Capability || pkg.Adapter != proposal.Adapter {
		_ = d.updateFutureProposal(proposal.ProposalID, futureStateMalformed, "", "", "", "Seshat package identity no longer matches proposal")
		response.Malformed++
		return nil
	}
	if !pkg.AdapterAvailable {
		_ = d.updateFutureProposal(proposal.ProposalID, futureStateAdapterUnavailable, "", "", "", "Seshat adapter is unavailable")
		response.AdapterUnavailable++
		return nil
	}
	if pkg.Temporal != nil {
		if proposal.JanusSubjectID == "" {
			_ = d.updateFutureProposal(proposal.ProposalID, futureStateTemporallyBlocked, "", "", "", "temporally governed proposal has no Janus subject")
			response.TemporallyBlocked++
			return nil
		}
		subject, err := d.loadJanusSubject(proposal.JanusSubjectID)
		if err != nil || subject.PackageCode != proposal.PackageCode || (subject.State != janusStateReady && subject.State != janusStateActive) {
			message := "Janus has not established temporal eligibility"
			if err != nil {
				message = "Janus subject unavailable: " + err.Error()
			}
			_ = d.updateFutureProposal(proposal.ProposalID, futureStateTemporallyBlocked, "", "", "", message)
			response.TemporallyBlocked++
			return nil
		}
	}
	if paused {
		response.Deferred++
		return nil
	}
	grant, err := d.futureActiveGrant(proposal.Capability, proposal.Scope)
	if err != nil {
		request, requestErr := d.createFutureAuthorityRequest(proposal, pkg)
		if requestErr != nil {
			return requestErr
		}
		if request.Status == futureStateReferredPrometheus {
			response.Referred++
		} else {
			response.NeedsPerson++
		}
		return nil
	}
	order, err := d.createOrderWithOrigin(orderRequest{RequestText: proposal.SourceMissive, Recipient: proposal.Adapter, Capability: proposal.Capability, Scope: proposal.Scope, Payload: proposal.Payload, Constraints: proposal.Constraints, Purpose: proposal.Purpose, ExpiresAt: proposal.RequestedExecutionBound, AuthorityGrantID: grant.ID}, false, orderOriginFuture, "future")
	if err != nil {
		return err
	}
	if order.Status == orderStatusUnavailable {
		_ = d.updateFutureProposal(proposal.ProposalID, futureStateAdapterUnavailable, "", grant.ID, "", order.LastError)
		response.AdapterUnavailable++
		return nil
	}
	if order.Status != orderStatusSealed {
		return fmt.Errorf("Future authority resolution did not produce a sealed order: %s", order.Status)
	}
	if err := d.updateFutureProposal(proposal.ProposalID, futureStateSealed, "", grant.ID, order.OrderID, ""); err != nil {
		return err
	}
	d.recordOrderEvent(order.OrderID, "future_sealed", "proposal "+proposal.ProposalID+"; package "+proposal.PackageCode)
	response.Sealed++
	return nil
}

func (d *daemon) futureRound(ctx context.Context) (futureRoundResponse, error) {
	if err := initializeFutureSchema(d.db); err != nil {
		return futureRoundResponse{}, err
	}
	started, err := fatherTimeNow(d.cfg)
	if err != nil {
		return futureRoundResponse{}, err
	}
	paused, err := d.paused()
	if err != nil {
		return futureRoundResponse{}, err
	}
	response := futureRoundResponse{StartedAt: started, Paused: paused}
	rows, err := d.db.Query(`SELECT proposal_id FROM future_proposals WHERE state NOT IN (?, ?, ?, ?) ORDER BY created_at, proposal_id`, futureStateSealed, futureStateDenied, futureStateMalformed, futureStateAdapterUnavailable)
	if err != nil {
		return response, err
	}
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
	if err := rows.Close(); err != nil {
		return response, err
	}
	for _, id := range ids {
		response.Examined++
		proposal, err := d.loadFutureProposal(id)
		if err != nil {
			return response, err
		}
		if err := d.processFutureProposal(ctx, proposal, paused, &response); err != nil {
			response.Failed++
			response.Errors = append(response.Errors, proposal.ProposalID+": "+err.Error())
			_ = d.updateFutureProposal(proposal.ProposalID, proposal.State, "", "", "", err.Error())
		}
	}
	response.Result = fmt.Sprintf("Future examined %d proposal(s), sealed %d, referred %d, needs person %d, temporally blocked %d, adapter unavailable %d, deferred %d, failed %d.", response.Examined, response.Sealed, response.Referred, response.NeedsPerson, response.TemporallyBlocked, response.AdapterUnavailable, response.Deferred, response.Failed)
	return response, nil
}

func (d *daemon) futureInspection() (map[string]any, error) {
	if err := initializeFutureSchema(d.db); err != nil {
		return nil, err
	}
	counts := map[string]int{}
	rows, err := d.db.Query(`SELECT state, COUNT(*) FROM future_proposals GROUP BY state`)
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
	proposals, err := d.futureProposals(100)
	if err != nil {
		return nil, err
	}
	requests, err := d.futureAuthorityRequests(100)
	if err != nil {
		return nil, err
	}
	paused, err := d.paused()
	if err != nil {
		return nil, err
	}
	return map[string]any{"ready": true, "paused": paused, "state_counts": counts, "proposals": proposals, "authority_requests": requests}, nil
}

func (d *daemon) futureProposals(limit int) ([]futureProposal, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	rows, err := d.db.Query(`SELECT proposal_id FROM future_proposals ORDER BY created_at DESC, proposal_id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []futureProposal
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		proposal, err := d.loadFutureProposal(id)
		if err != nil {
			return nil, err
		}
		result = append(result, proposal)
	}
	return result, rows.Err()
}

func (d *daemon) futureAuthorityRequests(limit int) ([]futureAuthorityRequest, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	rows, err := d.db.Query(`SELECT request_id FROM future_authority_requests ORDER BY created_at DESC, request_id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []futureAuthorityRequest
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		request, err := d.loadFutureAuthorityRequest(id)
		if err != nil {
			return nil, err
		}
		result = append(result, request)
	}
	return result, rows.Err()
}

func (d *daemon) handleFutureRound(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("Future round requires the local CLI control boundary"))
		return
	}
	result, err := d.futureRound(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (d *daemon) handleCreateFutureProposal(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("Future proposal creation requires the local CLI control boundary"))
		return
	}
	var request futureProposalRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	proposal, err := d.createFutureProposal(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, proposal)
}

func (d *daemon) handleShowFuture(w http.ResponseWriter, r *http.Request) {
	inspection, err := d.futureInspection()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, inspection)
}

func (d *daemon) handleShowFutureProposals(w http.ResponseWriter, r *http.Request) {
	proposals, err := d.futureProposals(100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, proposals)
}

func (d *daemon) handleShowFutureProposal(w http.ResponseWriter, r *http.Request) {
	proposal, err := d.loadFutureProposal(r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("Future proposal not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, proposal)
}
