package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const (
	orderOriginPersonDirect = "person_direct"
	orderOriginParent       = "parent_order"
	orderOriginStanding     = "standing_authority"
	orderOriginPresent      = "present"
	orderStatusPending      = "pending"
	orderStatusInterpret    = "needs_interpretation"
	orderStatusRuling       = "needs_ruling"
	orderStatusSealed       = "sealed"
	orderStatusWaiting      = "waiting_paused"
	orderStatusReady        = "ready"
	orderStatusDispatched   = "dispatched"
	orderStatusRunning      = "running"
	orderStatusCompleted    = "completed"
	orderStatusFailed       = "failed"
	orderStatusInterrupted  = "interrupted"
	orderStatusExpired      = "expired"
	orderStatusCancelled    = "cancelled"
	orderStatusSuperseded   = "superseded"
	orderStatusUnavailable  = "blocked_unavailable"
	goStatusIssued          = "issued"
	goStatusConsumed        = "consumed"
	goStatusInvalidated     = "invalidated"
)

type orderRequest struct {
	RequestText      string          `json:"request_text,omitempty"`
	Recipient        string          `json:"recipient,omitempty"`
	Capability       string          `json:"capability,omitempty"`
	Scope            string          `json:"scope,omitempty"`
	Payload          json.RawMessage `json:"payload,omitempty"`
	Constraints      json.RawMessage `json:"constraints,omitempty"`
	ParentOrderID    string          `json:"parent_order_id,omitempty"`
	ParentSealDigest string          `json:"parent_seal_digest,omitempty"`
	Purpose          string          `json:"purpose,omitempty"`
	ExpiresAt        string          `json:"expires_at,omitempty"`
	SupersedesID     string          `json:"supersedes_order_id,omitempty"`
}

type mercuryOrder struct {
	OrderID          string          `json:"order_id"`
	CreatedAt        string          `json:"created_at"`
	Origin           string          `json:"origin"`
	RequestedBy      string          `json:"requested_by,omitempty"`
	RequestText      string          `json:"request_text"`
	Recipient        string          `json:"recipient,omitempty"`
	Capability       string          `json:"capability,omitempty"`
	Scope            string          `json:"scope,omitempty"`
	Payload          json.RawMessage `json:"payload,omitempty"`
	Constraints      json.RawMessage `json:"constraints,omitempty"`
	AuthorityBasis   string          `json:"authority_basis,omitempty"`
	AuthorityRef     string          `json:"authority_ref,omitempty"`
	ParentOrderID    string          `json:"parent_order_id,omitempty"`
	ParentSealDigest string          `json:"parent_seal_digest,omitempty"`
	Purpose          string          `json:"purpose,omitempty"`
	Status           string          `json:"status"`
	ExpiresAt        string          `json:"expires_at,omitempty"`
	SealedAt         string          `json:"sealed_at,omitempty"`
	SealDigest       string          `json:"seal_digest,omitempty"`
	SupersedesID     string          `json:"supersedes_order_id,omitempty"`
	LastError        string          `json:"last_error,omitempty"`
}

type sealedOrderFields struct {
	OrderID          string          `json:"order_id"`
	CreatedAt        string          `json:"created_at"`
	Origin           string          `json:"origin"`
	RequestedBy      string          `json:"requested_by,omitempty"`
	RequestText      string          `json:"request_text"`
	Recipient        string          `json:"recipient,omitempty"`
	Capability       string          `json:"capability,omitempty"`
	Scope            string          `json:"scope,omitempty"`
	Payload          json.RawMessage `json:"payload,omitempty"`
	Constraints      json.RawMessage `json:"constraints,omitempty"`
	AuthorityBasis   string          `json:"authority_basis,omitempty"`
	AuthorityRef     string          `json:"authority_ref,omitempty"`
	ParentOrderID    string          `json:"parent_order_id,omitempty"`
	ParentSealDigest string          `json:"parent_seal_digest,omitempty"`
	Purpose          string          `json:"purpose,omitempty"`
	ExpiresAt        string          `json:"expires_at,omitempty"`
	SupersedesID     string          `json:"supersedes_order_id,omitempty"`
}

type mercuryGo struct {
	GoID            string `json:"go_id"`
	OrderID         string `json:"order_id"`
	OrderSealDigest string `json:"order_seal_digest"`
	IssuedAt        string `json:"issued_at"`
	AuthorityBasis  string `json:"authority_basis"`
	AuthorityRef    string `json:"authority_ref"`
	ParentGoID      string `json:"parent_go_id,omitempty"`
	ValidUntil      string `json:"valid_until,omitempty"`
}

type mercuryRoundResponse struct {
	StartedAt  string `json:"started_at"`
	Paused     bool   `json:"paused"`
	Examined   int    `json:"examined"`
	Dispatched int    `json:"dispatched"`
	Completed  int    `json:"completed"`
	Deferred   int    `json:"deferred"`
	Result     string `json:"result"`
}

func initializeMercurySchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS orders (
			order_id TEXT PRIMARY KEY, created_at TEXT NOT NULL, origin TEXT NOT NULL,
			requested_by TEXT, request_text TEXT NOT NULL, recipient TEXT, capability TEXT,
			scope TEXT, payload_json TEXT, constraints_json TEXT, authority_basis TEXT,
			authority_ref TEXT, parent_order_id TEXT, parent_seal_digest TEXT, purpose TEXT,
			status TEXT NOT NULL, expires_at TEXT, sealed_at TEXT, seal_digest TEXT,
			supersedes_order_id TEXT, last_error TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS orders_status_idx ON orders(status, created_at)`,
		`CREATE TABLE IF NOT EXISTS order_events (
			event_id TEXT PRIMARY KEY, order_id TEXT NOT NULL, occurred_at TEXT NOT NULL,
			event_type TEXT NOT NULL, detail TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS order_go_commands (
			go_id TEXT PRIMARY KEY, order_id TEXT NOT NULL, order_seal_digest TEXT NOT NULL,
			issued_at TEXT NOT NULL, authority_basis TEXT NOT NULL, authority_ref TEXT,
			parent_go_id TEXT, valid_until TEXT, consumed_at TEXT, invalidated_at TEXT,
			status TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS order_go_order_idx ON order_go_commands(order_id, issued_at)`,
		`CREATE TABLE IF NOT EXISTS order_attempts (
			attempt_id TEXT PRIMARY KEY, order_id TEXT NOT NULL, go_id TEXT NOT NULL,
			recipient TEXT NOT NULL, started_at TEXT NOT NULL, ended_at TEXT,
			status TEXT NOT NULL, result_summary TEXT, evidence_refs TEXT, error TEXT
		)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func (d *daemon) ensureMercury() error { return initializeMercurySchema(d.db) }

func validJSONOrEmpty(value json.RawMessage) (string, error) {
	if len(value) == 0 {
		return "", nil
	}
	if !json.Valid(value) {
		return "", errors.New("payload and constraints must be valid JSON")
	}
	return string(value), nil
}

func (d *daemon) createOrder(input orderRequest, trustedPerson bool) (mercuryOrder, error) {
	if err := d.ensureMercury(); err != nil {
		return mercuryOrder{}, err
	}
	requestText := input.RequestText
	if strings.TrimSpace(requestText) == "" {
		return mercuryOrder{}, errors.New("request_text is required and is preserved verbatim")
	}
	input.Recipient, input.Capability, input.Scope = strings.TrimSpace(input.Recipient), strings.TrimSpace(input.Capability), strings.TrimSpace(input.Scope)
	if (input.Recipient == "") != (input.Capability == "") || (input.Capability == "") != (input.Scope == "") {
		return mercuryOrder{}, errors.New("recipient, capability, and scope must be supplied together")
	}
	payload, err := validJSONOrEmpty(input.Payload)
	if err != nil {
		return mercuryOrder{}, err
	}
	constraints, err := validJSONOrEmpty(input.Constraints)
	if err != nil {
		return mercuryOrder{}, err
	}
	if payload == "" && strings.HasPrefix(input.Scope, "occurrence:") {
		payloadBytes, _ := json.Marshal(map[string]string{"occurrence_id": strings.TrimPrefix(input.Scope, "occurrence:")})
		payload = string(payloadBytes)
	}
	orderID, err := newLocalID("order-")
	if err != nil {
		return mercuryOrder{}, err
	}
	createdAt, err := fatherTimeNow(d.cfg)
	if err != nil {
		return mercuryOrder{}, err
	}
	order := mercuryOrder{OrderID: orderID, CreatedAt: createdAt, Origin: "system", RequestText: requestText, Recipient: input.Recipient, Capability: input.Capability, Scope: input.Scope, Payload: json.RawMessage(payload), Constraints: json.RawMessage(constraints), ParentOrderID: strings.TrimSpace(input.ParentOrderID), ParentSealDigest: strings.TrimSpace(input.ParentSealDigest), Purpose: strings.TrimSpace(input.Purpose), ExpiresAt: strings.TrimSpace(input.ExpiresAt), SupersedesID: strings.TrimSpace(input.SupersedesID), Status: orderStatusPending}
	if trustedPerson {
		order.Origin, order.RequestedBy = orderOriginPersonDirect, "local-person"
	}
	if order.Recipient != "" {
		order.Status = orderStatusReady
		basis, ref, authorityErr := d.authorityForOrder(order, trustedPerson)
		if authorityErr != nil {
			order.Status = orderStatusRuling
			order.LastError = authorityErr.Error()
		} else {
			order.AuthorityBasis, order.AuthorityRef = basis, ref
			if !d.mercuryCapabilityAvailable(order.Recipient, order.Capability) {
				order.Status, order.LastError = orderStatusUnavailable, "no explicit local adapter for recipient/capability"
			} else if err := d.validateChildOrder(order); err != nil {
				order.Status, order.LastError = orderStatusRuling, err.Error()
			} else if err := d.sealOrder(&order); err != nil {
				return mercuryOrder{}, err
			}
		}
	} else {
		order.Status = orderStatusInterpret
		order.LastError = "flexible Mercury interpretation is unavailable; structured recipient, capability, and scope are required"
	}
	paused, err := d.paused()
	if err != nil {
		return mercuryOrder{}, err
	}
	if paused && (order.Status == orderStatusReady || order.Status == orderStatusSealed) {
		order.Status = orderStatusWaiting
	}
	if err := d.persistOrder(order); err != nil {
		return mercuryOrder{}, err
	}
	if order.SupersedesID != "" {
		if _, err := d.db.Exec(`UPDATE orders SET status = ?, last_error = ? WHERE order_id = ? AND status NOT IN (?, ?, ?)`, orderStatusSuperseded, "superseded by "+order.OrderID, order.SupersedesID, orderStatusCompleted, orderStatusCancelled, orderStatusSuperseded); err != nil {
			return mercuryOrder{}, err
		}
		d.recordOrderEvent(order.SupersedesID, "superseded", order.OrderID)
	}
	d.recordOrderEvent(order.OrderID, "created", order.Status)
	return order, nil
}

func fatherTimeNow(cfg config) (string, error) {
	loc, err := fatherTimeLocation(cfg)
	if err != nil {
		return "", err
	}
	return time.Now().In(loc).Format(time.RFC3339Nano), nil
}

func (d *daemon) persistOrder(order mercuryOrder) error {
	_, err := d.db.Exec(`INSERT INTO orders (order_id, created_at, origin, requested_by, request_text, recipient, capability, scope, payload_json, constraints_json, authority_basis, authority_ref, parent_order_id, parent_seal_digest, purpose, status, expires_at, sealed_at, seal_digest, supersedes_order_id, last_error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, order.OrderID, order.CreatedAt, order.Origin, order.RequestedBy, order.RequestText, order.Recipient, order.Capability, order.Scope, nullableJSON(order.Payload), nullableJSON(order.Constraints), order.AuthorityBasis, order.AuthorityRef, order.ParentOrderID, order.ParentSealDigest, order.Purpose, order.Status, order.ExpiresAt, order.SealedAt, order.SealDigest, order.SupersedesID, order.LastError)
	return err
}

func nullableJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return string(value)
}

func (d *daemon) authorityForOrder(order mercuryOrder, trustedPerson bool) (string, string, error) {
	if trustedPerson {
		return orderOriginPersonDirect, order.RequestedBy, nil
	}
	if d.hasActiveGrant(order.Capability, order.Scope) {
		var grantID string
		if err := d.db.QueryRow(`SELECT id FROM capability_grants WHERE capability = ? AND scope = ? AND status = 'active' ORDER BY created_at DESC LIMIT 1`, order.Capability, order.Scope).Scan(&grantID); err == nil {
			return orderOriginStanding, grantID, nil
		}
	}
	if order.ParentOrderID != "" {
		return orderOriginParent, order.ParentOrderID, nil
	}
	return "", "", errors.New("current authority is absent")
}

func (d *daemon) authorityStillValid(order mercuryOrder) error {
	switch order.AuthorityBasis {
	case orderOriginPersonDirect:
		return nil
	case orderOriginStanding:
		if !d.hasActiveGrant(order.Capability, order.Scope) {
			return errors.New("active grant is no longer present")
		}
		return nil
	case orderOriginParent:
		return d.validateChildOrder(order)
	default:
		return errors.New("order has no recognized authority basis")
	}
}

func (d *daemon) validateChildOrder(order mercuryOrder) error {
	if order.ParentOrderID == "" {
		return nil
	}
	var parent mercuryOrder
	if err := d.scanOrder(d.db.QueryRow(`SELECT order_id, created_at, origin, requested_by, request_text, recipient, capability, scope, payload_json, constraints_json, authority_basis, authority_ref, parent_order_id, parent_seal_digest, purpose, status, expires_at, sealed_at, seal_digest, supersedes_order_id, last_error FROM orders WHERE order_id = ?`, order.ParentOrderID), &parent); err != nil {
		return fmt.Errorf("parent order unavailable: %w", err)
	}
	if parent.Status == orderStatusCancelled || parent.Status == orderStatusExpired || parent.Status == orderStatusSuperseded || parent.SealDigest == "" {
		return errors.New("parent order is not a valid sealed authority")
	}
	if order.ParentSealDigest != parent.SealDigest {
		return errors.New("parent seal digest does not match")
	}
	if parent.Recipient != order.Recipient || parent.Capability != order.Capability || !scopeWithin(parent.Scope, order.Scope) {
		return errors.New("child order widens parent recipient, capability, or scope")
	}
	return nil
}

func scopeWithin(parent, child string) bool {
	parent, child = strings.TrimSpace(parent), strings.TrimSpace(child)
	if parent == child {
		return true
	}
	return strings.HasSuffix(parent, "/") && strings.HasPrefix(child, parent)
}

func (d *daemon) sealOrder(order *mercuryOrder) error {
	fields := sealedOrderFields{OrderID: order.OrderID, CreatedAt: order.CreatedAt, Origin: order.Origin, RequestedBy: order.RequestedBy, RequestText: order.RequestText, Recipient: order.Recipient, Capability: order.Capability, Scope: order.Scope, Payload: order.Payload, Constraints: order.Constraints, AuthorityBasis: order.AuthorityBasis, AuthorityRef: order.AuthorityRef, ParentOrderID: order.ParentOrderID, ParentSealDigest: order.ParentSealDigest, Purpose: order.Purpose, ExpiresAt: order.ExpiresAt, SupersedesID: order.SupersedesID}
	data, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	order.SealDigest = "sha256:" + hex.EncodeToString(digest[:])
	order.SealedAt, err = fatherTimeNow(d.cfg)
	if err != nil {
		return err
	}
	if order.Status == orderStatusReady {
		order.Status = orderStatusSealed
	}
	return nil
}

func (d *daemon) scanOrder(row *sql.Row, order *mercuryOrder) error {
	var payload, constraints sql.NullString
	return row.Scan(&order.OrderID, &order.CreatedAt, &order.Origin, &order.RequestedBy, &order.RequestText, &order.Recipient, &order.Capability, &order.Scope, &payload, &constraints, &order.AuthorityBasis, &order.AuthorityRef, &order.ParentOrderID, &order.ParentSealDigest, &order.Purpose, &order.Status, &order.ExpiresAt, &order.SealedAt, &order.SealDigest, &order.SupersedesID, &order.LastError)
}

func (d *daemon) loadOrder(id string) (mercuryOrder, error) {
	var order mercuryOrder
	var payload, constraints sql.NullString
	err := d.db.QueryRow(`SELECT order_id, created_at, origin, requested_by, request_text, recipient, capability, scope, payload_json, constraints_json, authority_basis, authority_ref, parent_order_id, parent_seal_digest, purpose, status, expires_at, sealed_at, seal_digest, supersedes_order_id, last_error FROM orders WHERE order_id = ?`, id).Scan(&order.OrderID, &order.CreatedAt, &order.Origin, &order.RequestedBy, &order.RequestText, &order.Recipient, &order.Capability, &order.Scope, &payload, &constraints, &order.AuthorityBasis, &order.AuthorityRef, &order.ParentOrderID, &order.ParentSealDigest, &order.Purpose, &order.Status, &order.ExpiresAt, &order.SealedAt, &order.SealDigest, &order.SupersedesID, &order.LastError)
	if err != nil {
		return mercuryOrder{}, err
	}
	if payload.Valid {
		order.Payload = json.RawMessage(payload.String)
	}
	if constraints.Valid {
		order.Constraints = json.RawMessage(constraints.String)
	}
	return order, nil
}

func (d *daemon) orderSealValid(order mercuryOrder) bool {
	copy := order
	copy.SealDigest = ""
	copy.SealedAt = ""
	if d.sealOrder(&copy) != nil {
		return false
	}
	return copy.SealDigest == order.SealDigest
}

func (d *daemon) mercuryCapabilityAvailable(recipient, capability string) bool {
	return recipient == "present" && capability == "present.process"
}

func (d *daemon) recordOrderEvent(orderID, eventType, detail string) {
	eventID, err := newLocalID("order-event-")
	if err != nil {
		return
	}
	_, _ = d.db.Exec(`INSERT INTO order_events (event_id, order_id, occurred_at, event_type, detail) VALUES (?, ?, ?, ?, ?)`, eventID, orderID, time.Now().Format(time.RFC3339Nano), eventType, detail)
}

func (d *daemon) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) == "" || r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("order creation requires the local CLI control boundary"))
		return
	}
	var request orderRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	order, err := d.createOrder(request, true)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	d.audit("POST /v1/orders", order.RequestText, order.Capability, "", "completed", "order created: "+order.OrderID, "", 0)
	writeJSON(w, http.StatusCreated, order)
}

func (d *daemon) handleShowOrders(w http.ResponseWriter, r *http.Request) {
	if err := d.ensureMercury(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	rows, err := d.db.Query(`SELECT order_id FROM orders ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	orders := make([]mercuryOrder, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		order, err := d.loadOrder(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		orders = append(orders, order)
	}
	writeJSON(w, http.StatusOK, orders)
}

func (d *daemon) handleShowOrder(w http.ResponseWriter, r *http.Request) {
	if err := d.ensureMercury(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	order, err := d.loadOrder(r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("order not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (d *daemon) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) == "" || r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("order cancellation requires the local CLI control boundary"))
		return
	}
	if err := d.ensureMercury(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	id := r.PathValue("id")
	order, err := d.loadOrder(id)
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("order not found"))
		return
	}
	if order.Status == orderStatusCompleted || order.Status == orderStatusCancelled || order.Status == orderStatusSuperseded {
		writeError(w, http.StatusConflict, errors.New("order is already terminal"))
		return
	}
	if _, err := d.db.Exec(`UPDATE orders SET status = ?, last_error = ? WHERE order_id = ?`, orderStatusCancelled, "cancelled by person", id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	d.recordOrderEvent(id, "cancelled", "cancelled by person; sealed fields unchanged")
	order.Status, order.LastError = orderStatusCancelled, "cancelled by person"
	writeJSON(w, http.StatusOK, order)
}

func (d *daemon) handleMercuryRound(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) == "" || r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("Mercury round requires the local CLI control boundary"))
		return
	}
	result, err := d.mercuryRound(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (d *daemon) mercuryRound(ctx context.Context) (mercuryRoundResponse, error) {
	if err := d.ensureMercury(); err != nil {
		return mercuryRoundResponse{}, err
	}
	started := time.Now().Format(time.RFC3339Nano)
	paused, err := d.paused()
	if err != nil {
		return mercuryRoundResponse{}, err
	}
	response := mercuryRoundResponse{StartedAt: started, Paused: paused}
	rows, err := d.db.Query(`SELECT order_id FROM orders WHERE status IN (?, ?, ?, ?, ?, ?) ORDER BY created_at`, orderStatusPending, orderStatusInterpret, orderStatusRuling, orderStatusSealed, orderStatusReady, orderStatusWaiting)
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
	for _, id := range ids {
		response.Examined++
		order, err := d.loadOrder(id)
		if err != nil {
			return response, err
		}
		if paused {
			if order.Status == orderStatusReady || order.Status == orderStatusSealed {
				_, _ = d.db.Exec(`UPDATE orders SET status = ? WHERE order_id = ?`, orderStatusWaiting, id)
				d.recordOrderEvent(id, "waiting_paused", "Prometheus pause prevents Go issuance")
			}
			response.Deferred++
			continue
		}
		if order.Status == orderStatusInterpret || order.Status == orderStatusRuling || order.Status == orderStatusPending {
			response.Deferred++
			continue
		}
		if order.ExpiresAt != "" {
			expires, parseErr := time.Parse(time.RFC3339Nano, order.ExpiresAt)
			if parseErr == nil && time.Now().After(expires) {
				_, _ = d.db.Exec(`UPDATE orders SET status = ? WHERE order_id = ?`, orderStatusExpired, id)
				response.Deferred++
				continue
			}
		}
		if !d.orderSealValid(order) {
			_, _ = d.db.Exec(`UPDATE orders SET status = ?, last_error = ? WHERE order_id = ?`, orderStatusFailed, "sealed fields no longer match seal digest", id)
			response.Deferred++
			continue
		}
		if err := d.authorityStillValid(order); err != nil {
			_, _ = d.db.Exec(`UPDATE orders SET status = ?, last_error = ? WHERE order_id = ?`, orderStatusRuling, err.Error(), id)
			response.Deferred++
			continue
		}
		if !d.mercuryCapabilityAvailable(order.Recipient, order.Capability) {
			_, _ = d.db.Exec(`UPDATE orders SET status = ?, last_error = ? WHERE order_id = ?`, orderStatusUnavailable, "no explicit local adapter for recipient/capability", id)
			response.Deferred++
			continue
		}
		goCommand, err := d.issueGo(order)
		if err != nil {
			return response, err
		}
		response.Dispatched++
		if paused, _ = d.paused(); paused {
			_, _ = d.db.Exec(`UPDATE order_go_commands SET status = ?, invalidated_at = ? WHERE go_id = ?`, goStatusInvalidated, time.Now().Format(time.RFC3339Nano), goCommand.GoID)
			_, _ = d.db.Exec(`UPDATE orders SET status = ?, last_error = ? WHERE order_id = ?`, orderStatusInterrupted, "Prometheus paused before dispatch", order.OrderID)
			continue
		}
		if err := d.dispatchOrder(ctx, order, goCommand); err != nil {
			response.Deferred++
			continue
		}
		response.Completed++
	}
	response.Result = fmt.Sprintf("Mercury examined %d order(s), dispatched %d, completed %d, deferred %d.", response.Examined, response.Dispatched, response.Completed, response.Deferred)
	d.audit("POST /v1/mercury/round", "", "mercury.round", "", "completed", response.Result, "", 0)
	return response, nil
}

func (d *daemon) issueGo(order mercuryOrder) (mercuryGo, error) {
	id, err := newLocalID("go-")
	if err != nil {
		return mercuryGo{}, err
	}
	issued, err := fatherTimeNow(d.cfg)
	if err != nil {
		return mercuryGo{}, err
	}
	command := mercuryGo{GoID: id, OrderID: order.OrderID, OrderSealDigest: order.SealDigest, IssuedAt: issued, AuthorityBasis: order.AuthorityBasis, AuthorityRef: order.AuthorityRef, ParentGoID: ""}
	if _, err := d.db.Exec(`INSERT INTO order_go_commands (go_id, order_id, order_seal_digest, issued_at, authority_basis, authority_ref, parent_go_id, valid_until, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, command.GoID, command.OrderID, command.OrderSealDigest, command.IssuedAt, command.AuthorityBasis, command.AuthorityRef, command.ParentGoID, command.ValidUntil, goStatusIssued); err != nil {
		return mercuryGo{}, err
	}
	if _, err := d.db.Exec(`UPDATE orders SET status = ? WHERE order_id = ?`, orderStatusRunning, order.OrderID); err != nil {
		return mercuryGo{}, err
	}
	d.recordOrderEvent(order.OrderID, "go_issued", command.GoID)
	return command, nil
}

func (d *daemon) dispatchOrder(ctx context.Context, order mercuryOrder, command mercuryGo) error {
	started := time.Now().Format(time.RFC3339Nano)
	attemptID, err := newLocalID("attempt-")
	if err != nil {
		return err
	}
	_, _ = d.db.Exec(`INSERT INTO order_attempts (attempt_id, order_id, go_id, recipient, started_at, status) VALUES (?, ?, ?, ?, ?, ?)`, attemptID, order.OrderID, command.GoID, order.Recipient, started, orderStatusRunning)
	var resultSummary, evidence, dispatchErr string
	if order.Recipient == "present" && order.Capability == "present.process" {
		var payload struct {
			OccurrenceID string `json:"occurrence_id"`
		}
		if err := json.Unmarshal(order.Payload, &payload); err != nil || strings.TrimSpace(payload.OccurrenceID) == "" {
			dispatchErr = "present.process requires payload occurrence_id"
		} else {
			result, processErr := d.processOccurrence(ctx, payload.OccurrenceID, order.RequestText)
			if processErr != nil {
				dispatchErr = processErr.Error()
			} else if result.Error != "" {
				dispatchErr, resultSummary = result.Error, presentResultText(result, order.RequestText)
			} else {
				resultSummary, evidence = presentResultText(result, order.RequestText), result.NormalizedPath
			}
		}
	} else {
		dispatchErr = "no exact dispatcher for order"
	}
	ended := time.Now().Format(time.RFC3339Nano)
	attemptStatus := orderStatusCompleted
	orderStatusValue := orderStatusCompleted
	if dispatchErr != "" {
		attemptStatus, orderStatusValue = orderStatusFailed, orderStatusFailed
	}
	_, _ = d.db.Exec(`UPDATE order_attempts SET ended_at = ?, status = ?, result_summary = ?, evidence_refs = ?, error = ? WHERE attempt_id = ?`, ended, attemptStatus, resultSummary, evidence, dispatchErr, attemptID)
	_, _ = d.db.Exec(`UPDATE order_go_commands SET consumed_at = ?, status = ? WHERE go_id = ? AND status = ?`, ended, goStatusConsumed, command.GoID, goStatusIssued)
	lastError := dispatchErr
	_, _ = d.db.Exec(`UPDATE orders SET status = ?, last_error = ? WHERE order_id = ?`, orderStatusValue, lastError, order.OrderID)
	d.recordOrderEvent(order.OrderID, "dispatch_"+attemptStatus, resultSummary)
	if dispatchErr != "" {
		return errors.New(dispatchErr)
	}
	return nil
}

func (d *daemon) invalidateMercuryGos() {
	_, _ = d.db.Exec(`UPDATE order_go_commands SET status = ?, invalidated_at = ? WHERE status = ?`, goStatusInvalidated, time.Now().Format(time.RFC3339Nano), goStatusIssued)
	_, _ = d.db.Exec(`UPDATE orders SET status = ?, last_error = ? WHERE status IN (?, ?) AND order_id IN (SELECT order_id FROM order_go_commands WHERE status = ?)`, orderStatusInterrupted, "Prometheus paused while Go was outstanding", orderStatusRunning, orderStatusDispatched, goStatusInvalidated)
}

func (d *daemon) mercuryInspection() (map[string]any, error) {
	if err := d.ensureMercury(); err != nil {
		return nil, err
	}
	var pending, sealed, waiting, completed, failed, ruling, interpretation int
	rows, err := d.db.Query(`SELECT status, COUNT(*) FROM orders GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		switch status {
		case orderStatusPending:
			pending += count
		case orderStatusSealed, orderStatusReady:
			sealed += count
		case orderStatusWaiting:
			waiting += count
		case orderStatusCompleted:
			completed += count
		case orderStatusFailed, orderStatusUnavailable:
			failed += count
		case orderStatusRuling:
			ruling += count
		case orderStatusInterpret:
			interpretation += count
		}
	}
	hermes := map[string]any{"mode": d.cfg.MercuryMode, "agent": d.cfg.MercuryAgent, "state": "unavailable"}
	if d.cfg.MercuryBinary != "" {
		if path, err := exec.LookPath(d.cfg.MercuryBinary); err == nil {
			hermes["state"], hermes["binary"] = "ready", path
		}
	}
	return map[string]any{"flexible": hermes, "deterministic_fallback": d.cfg.MercuryDeterministicFallback, "round_on_cron_wake": d.cfg.MercuryRoundOnCronWake, "orders": map[string]int{"pending": pending, "sealed": sealed, "waiting_paused": waiting, "needs_ruling": ruling, "needs_interpretation": interpretation, "completed": completed, "failed_or_unavailable": failed}}, nil
}
