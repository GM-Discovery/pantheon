package pantheon

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	grantStatusPending = "pending"
	grantStatusActive  = "active"
	grantStatusPaused  = "paused"
	grantStatusExpired = "expired"
	grantStatusRevoked = "revoked"

	holderPerson     = "person"
	holderPrometheus = "prometheus"
)

var deityHolders = map[string]bool{
	"apollo": true, "archimedes": true, "deity": true, "model": true,
}

var deterministicAuthorityHolders = map[string]bool{
	"person": true, "pantheon": true, "prometheus": true, "future": true,
	"mercury": true, "present": true, "past": true, "weave": true,
}

type delegationConstraints struct {
	AllowDelegation bool `json:"allow_delegation"`
}

func initializeAuthoritySchema(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS capability_grants (
		id TEXT PRIMARY KEY, capability TEXT NOT NULL, scope TEXT NOT NULL,
		intent_summary TEXT NOT NULL, diff TEXT, human_approved INTEGER NOT NULL,
		status TEXT NOT NULL, created_at TEXT NOT NULL,
		holder TEXT NOT NULL DEFAULT 'prometheus', granted_by TEXT NOT NULL DEFAULT 'person',
		parent_grant_id TEXT, valid_from TEXT, valid_until TEXT, remaining_uses INTEGER,
		delegation_constraints TEXT, revoked_at TEXT, updated_at TEXT
	)`); err != nil {
		return err
	}
	for column, definition := range map[string]string{
		"holder": "TEXT NOT NULL DEFAULT 'prometheus'", "granted_by": "TEXT NOT NULL DEFAULT 'person'",
		"parent_grant_id": "TEXT", "valid_from": "TEXT", "valid_until": "TEXT", "remaining_uses": "INTEGER",
		"delegation_constraints": "TEXT", "revoked_at": "TEXT", "updated_at": "TEXT",
	} {
		if err := ensureIntakeColumn(db, "capability_grants", column, definition); err != nil {
			return err
		}
	}
	return nil
}

func loadCapabilityGrantRow(row *sql.Row) (capabilityGrant, error) {
	var grant capabilityGrant
	var approved, uses sql.NullInt64
	var diff, holder, grantedBy, parentID, validFrom, validUntil, constraints, revokedAt, updatedAt sql.NullString
	if err := row.Scan(&grant.ID, &holder, &grant.Capability, &grant.Scope, &grant.IntentSummary, &diff, &approved, &grantedBy, &parentID, &grant.CreatedAt, &validFrom, &validUntil, &uses, &grant.Status, &constraints, &revokedAt, &updatedAt); err != nil {
		return capabilityGrant{}, err
	}
	grant.Holder, grant.GrantedBy = holder.String, grantedBy.String
	grant.Diff, grant.ParentGrantID = diff.String, parentID.String
	grant.ValidFrom, grant.ValidUntil = validFrom.String, validUntil.String
	grant.RevokedAt, grant.UpdatedAt = revokedAt.String, updatedAt.String
	grant.HumanApproved = approved.Int64 == 1
	if uses.Valid {
		remaining := int(uses.Int64)
		grant.RemainingUses = &remaining
	}
	if constraints.Valid && strings.TrimSpace(constraints.String) != "" {
		grant.DelegationConstraints = json.RawMessage(constraints.String)
	}
	return grant, nil
}

func (d *daemon) loadCapabilityGrant(id string) (capabilityGrant, error) {
	if err := initializeAuthoritySchema(d.db); err != nil {
		return capabilityGrant{}, err
	}
	return loadCapabilityGrantRow(d.db.QueryRow(`SELECT id, holder, capability, scope, intent_summary, diff, human_approved, granted_by, parent_grant_id, created_at, valid_from, valid_until, remaining_uses, status, delegation_constraints, revoked_at, updated_at FROM capability_grants WHERE id = ?`, id))
}

func (d *daemon) refreshGrantState(grant capabilityGrant, now time.Time) capabilityGrant {
	if grant.Status != grantStatusActive {
		return grant
	}
	if grant.RemainingUses != nil && *grant.RemainingUses <= 0 {
		return grant
	}
	if grant.ValidUntil != "" {
		if until, err := time.Parse(time.RFC3339Nano, grant.ValidUntil); err == nil && !now.Before(until) {
			_, _ = d.db.Exec(`UPDATE capability_grants SET status = ?, updated_at = ? WHERE id = ? AND status = ?`, grantStatusExpired, now.Format(time.RFC3339Nano), grant.ID, grantStatusActive)
			grant.Status = grantStatusExpired
		}
	}
	return grant
}

func grantTimeValid(grant capabilityGrant, now time.Time) bool {
	if grant.Status != grantStatusActive {
		return false
	}
	if grant.RemainingUses != nil && *grant.RemainingUses <= 0 {
		return false
	}
	if grant.ValidFrom != "" {
		from, err := time.Parse(time.RFC3339Nano, grant.ValidFrom)
		if err != nil || now.Before(from) {
			return false
		}
	}
	if grant.ValidUntil != "" {
		until, err := time.Parse(time.RFC3339Nano, grant.ValidUntil)
		if err != nil || !now.Before(until) {
			return false
		}
	}
	return true
}

func capabilityWithin(parent, child string) bool {
	parent, child = strings.TrimSpace(parent), strings.TrimSpace(child)
	if parent == child {
		return true
	}
	switch parent {
	case "calendar.maintain":
		return child == "calendar.read"
	default:
		return false
	}
}

func (d *daemon) matchingGrant(capability, scope string) (capabilityGrant, error) {
	if err := initializeAuthoritySchema(d.db); err != nil {
		return capabilityGrant{}, err
	}
	rows, err := d.db.Query(`SELECT id, holder, capability, scope, intent_summary, diff, human_approved, granted_by, parent_grant_id, created_at, valid_from, valid_until, remaining_uses, status, delegation_constraints, revoked_at, updated_at FROM capability_grants WHERE status = ? ORDER BY created_at DESC, id DESC`, grantStatusActive)
	if err != nil {
		return capabilityGrant{}, err
	}
	var grants []capabilityGrant
	for rows.Next() {
		var grant capabilityGrant
		var approved, uses sql.NullInt64
		var diff, holder, grantedBy, parentID, validFrom, validUntil, constraints, revokedAt, updatedAt sql.NullString
		if err := rows.Scan(&grant.ID, &holder, &grant.Capability, &grant.Scope, &grant.IntentSummary, &diff, &approved, &grantedBy, &parentID, &grant.CreatedAt, &validFrom, &validUntil, &uses, &grant.Status, &constraints, &revokedAt, &updatedAt); err != nil {
			return capabilityGrant{}, err
		}
		grant.Holder, grant.GrantedBy = holder.String, grantedBy.String
		grant.Diff, grant.ParentGrantID = diff.String, parentID.String
		grant.ValidFrom, grant.ValidUntil = validFrom.String, validUntil.String
		grant.RevokedAt, grant.UpdatedAt = revokedAt.String, updatedAt.String
		grant.HumanApproved = approved.Int64 == 1
		if uses.Valid {
			remaining := int(uses.Int64)
			grant.RemainingUses = &remaining
		}
		if constraints.Valid {
			grant.DelegationConstraints = json.RawMessage(constraints.String)
		}
		grants = append(grants, grant)
	}
	if err := rows.Err(); err != nil {
		return capabilityGrant{}, err
	}
	if err := rows.Close(); err != nil {
		return capabilityGrant{}, err
	}
	now := time.Now()
	for _, grant := range grants {
		grant = d.refreshGrantState(grant, now)
		if grant.Status == grantStatusActive && grantTimeValid(grant, now) && capabilityWithin(grant.Capability, capability) && scopeWithin(grant.Scope, scope) {
			return grant, nil
		}
	}
	return capabilityGrant{}, sql.ErrNoRows
}

func (d *daemon) hasActiveGrant(capability, scope string) bool {
	_, err := d.matchingGrant(capability, scope)
	return err == nil
}

func (d *daemon) consumeGrant(id string) error {
	grant, err := d.loadCapabilityGrant(id)
	if err != nil {
		return err
	}
	grant = d.refreshGrantState(grant, time.Now())
	if !grantTimeValid(grant, time.Now()) {
		return errors.New("grant is not active, valid, and unused")
	}
	if grant.RemainingUses == nil {
		return nil
	}
	result, err := d.db.Exec(`UPDATE capability_grants SET remaining_uses = remaining_uses - 1, updated_at = ? WHERE id = ? AND status = ? AND remaining_uses > 0`, time.Now().Format(time.RFC3339Nano), id, grantStatusActive)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return errors.New("grant use was already consumed")
	}
	return nil
}

func delegationAllowed(grant capabilityGrant) bool {
	if len(grant.DelegationConstraints) == 0 {
		return false
	}
	var constraints delegationConstraints
	return json.Unmarshal(grant.DelegationConstraints, &constraints) == nil && constraints.AllowDelegation
}

func deterministicHolder(holder string) bool {
	holder = strings.ToLower(strings.TrimSpace(holder))
	return deterministicAuthorityHolders[holder] && !deityHolders[holder]
}

func validateGrantWindow(request grantRequest) error {
	if request.ValidFrom != "" {
		if _, err := time.Parse(time.RFC3339Nano, request.ValidFrom); err != nil {
			return fmt.Errorf("valid_from is not RFC3339: %w", err)
		}
	}
	if request.ValidUntil != "" {
		until, err := time.Parse(time.RFC3339Nano, request.ValidUntil)
		if err != nil {
			return fmt.Errorf("valid_until is not RFC3339: %w", err)
		}
		if request.ValidFrom != "" {
			from, _ := time.Parse(time.RFC3339Nano, request.ValidFrom)
			if !until.After(from) {
				return errors.New("valid_until must be after valid_from")
			}
		}
	}
	if request.RemainingUses != nil && *request.RemainingUses < 1 {
		return errors.New("remaining_uses must be positive")
	}
	if len(request.DelegationConstraints) > 0 && !json.Valid(request.DelegationConstraints) {
		return errors.New("delegation_constraints must be valid JSON")
	}
	return nil
}

func (d *daemon) validateDelegatedGrant(request grantRequest) (capabilityGrant, error) {
	if request.ParentGrantID == "" {
		return capabilityGrant{}, nil
	}
	parent, err := d.loadCapabilityGrant(request.ParentGrantID)
	if err != nil {
		return capabilityGrant{}, fmt.Errorf("parent grant unavailable: %w", err)
	}
	parent = d.refreshGrantState(parent, time.Now())
	if !grantTimeValid(parent, time.Now()) || !delegationAllowed(parent) {
		return capabilityGrant{}, errors.New("parent grant is not active or does not permit delegation")
	}
	if !capabilityWithin(parent.Capability, request.Capability) || !scopeWithin(parent.Scope, request.Scope) {
		return capabilityGrant{}, errors.New("delegated grant widens parent capability or scope")
	}
	if request.ValidFrom != "" && parent.ValidFrom != "" {
		childFrom, childErr := time.Parse(time.RFC3339Nano, request.ValidFrom)
		parentFrom, parentErr := time.Parse(time.RFC3339Nano, parent.ValidFrom)
		if childErr != nil || parentErr != nil || childFrom.Before(parentFrom) {
			return capabilityGrant{}, errors.New("delegated valid_from precedes parent grant")
		}
	}
	if request.ValidUntil != "" && parent.ValidUntil != "" {
		childUntil, childErr := time.Parse(time.RFC3339Nano, request.ValidUntil)
		parentUntil, parentErr := time.Parse(time.RFC3339Nano, parent.ValidUntil)
		if childErr != nil || parentErr != nil || childUntil.After(parentUntil) {
			return capabilityGrant{}, errors.New("delegated valid_until exceeds parent grant")
		}
	}
	if parent.RemainingUses != nil && request.RemainingUses != nil && *request.RemainingUses > *parent.RemainingUses {
		return capabilityGrant{}, errors.New("delegated remaining_uses exceeds parent grant")
	}
	if parent.RemainingUses != nil && request.RemainingUses == nil {
		return capabilityGrant{}, errors.New("delegated unlimited uses exceed parent grant limit")
	}
	if parent.ValidUntil != "" && request.ValidUntil == "" {
		return capabilityGrant{}, errors.New("delegated unlimited duration exceeds parent grant limit")
	}
	if parent.ValidFrom != "" && request.ValidFrom == "" {
		return capabilityGrant{}, errors.New("delegated validity begins before parent grant")
	}
	return parent, nil
}

func (d *daemon) authorityChain(grantID string) ([]capabilityGrant, error) {
	var chain []capabilityGrant
	seen := map[string]bool{}
	for grantID != "" {
		if seen[grantID] {
			return nil, errors.New("grant provenance contains a cycle")
		}
		seen[grantID] = true
		grant, err := d.loadCapabilityGrant(grantID)
		if err != nil {
			return nil, err
		}
		chain = append(chain, grant)
		grantID = grant.ParentGrantID
	}
	return chain, nil
}

func (d *daemon) approveGrantAsPrometheus(id string) error {
	grant, err := d.loadCapabilityGrant(id)
	if err != nil {
		return err
	}
	if grant.Status != grantStatusPending || grant.ParentGrantID == "" {
		return errors.New("grant is not a pending delegated request")
	}
	if !deterministicHolder(grant.Holder) {
		return errors.New("delegated grant holder is not deterministic")
	}
	parent, err := d.validateDelegatedGrant(grantRequest{Capability: grant.Capability, Scope: grant.Scope, ParentGrantID: grant.ParentGrantID, ValidFrom: grant.ValidFrom, ValidUntil: grant.ValidUntil, RemainingUses: grant.RemainingUses})
	if err != nil || parent.Holder != holderPrometheus {
		if err != nil {
			return err
		}
		return errors.New("Prometheus does not hold the parent authority")
	}
	now := time.Now().Format(time.RFC3339Nano)
	_, err = d.db.Exec(`UPDATE capability_grants SET status = ?, granted_by = ?, updated_at = ? WHERE id = ? AND status = ?`, grantStatusActive, holderPrometheus, now, id, grantStatusPending)
	return err
}
