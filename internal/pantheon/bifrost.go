package pantheon

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	bifrostCapabilityTypeMCP         = "mcp"
	bifrostCapabilityTypeCommand     = "command"
	bifrostCapabilityTypeApplication = "application"
	bifrostCapabilityTypeHTTP        = "http"
	bifrostCapabilityTypeAPI         = "api"
	bifrostCapabilityTypePlugin      = "plugin"
	bifrostCapabilityTypeModel       = "model"
	bifrostCapabilityTypeFilesystem  = "filesystem"
	bifrostCapabilityTypeImported    = "imported"
	bifrostCapabilityTypeProtocol    = "protocol"

	bifrostOutcomeCompleted = "completed"
	bifrostOutcomeBlocked   = "blocked"
	bifrostOutcomeFailed    = "failed"

	bifrostFailureApproval       = "approval_required"
	bifrostFailureUnavailable    = "unavailable"
	bifrostFailureAuthentication = "authentication_expired"
	bifrostFailurePolicy         = "blocked_by_policy"
	bifrostFailurePrometheus     = "blocked_by_prometheus"
	bifrostFailureExternal       = "external_failure"
	bifrostFailureMalformed      = "malformed_response"
	bifrostFailureUnsupported    = "unsupported_operation"
	bifrostFailureLocal          = "local_adapter_failure"
	bifrostFailureArtifact       = "artifact_modified"

	bifrostMailMaxDefault  = 50
	bifrostMailReadLimit   = 16 << 20
	bifrostAttachmentLimit = 16 << 20
)

var bifrostCapabilityTypes = map[string]bool{
	bifrostCapabilityTypeMCP: true, bifrostCapabilityTypeCommand: true,
	bifrostCapabilityTypeApplication: true, bifrostCapabilityTypeHTTP: true,
	bifrostCapabilityTypeAPI: true, bifrostCapabilityTypePlugin: true,
	bifrostCapabilityTypeModel: true, bifrostCapabilityTypeFilesystem: true,
	bifrostCapabilityTypeImported: true, bifrostCapabilityTypeProtocol: true,
}

type bifrostCapability struct {
	CapabilityID           string                           `json:"capability_id"`
	Name                   string                           `json:"name"`
	Type                   string                           `json:"type"`
	Source                 string                           `json:"source"`
	Version                string                           `json:"version"`
	Description            string                           `json:"description"`
	Invocation             string                           `json:"invocation,omitempty"`
	Resources              []string                         `json:"resources,omitempty"`
	Operations             []string                         `json:"operations,omitempty"`
	ProhibitedOperations   []string                         `json:"prohibited_operations,omitempty"`
	PantheonComponents     []string                         `json:"pantheon_components,omitempty"`
	NetworkAccess          bool                             `json:"network_access"`
	SendsDataOffMachine    bool                             `json:"sends_data_off_machine"`
	AuthenticationRequired bool                             `json:"authentication_required"`
	Installed              bool                             `json:"installed"`
	Enabled                bool                             `json:"enabled"`
	DownloadApproved       bool                             `json:"download_approved"`
	InstallationApproved   bool                             `json:"installation_approved"`
	RegistrationApproved   bool                             `json:"registration_approved"`
	UseApproved            bool                             `json:"use_approved"`
	AutoUpdateApproved     bool                             `json:"auto_update_approved"`
	RegisteredAt           string                           `json:"registered_at"`
	LastInvokedAt          string                           `json:"last_invoked_at,omitempty"`
	LastChangedAt          string                           `json:"last_changed_at,omitempty"`
	DisabledAt             string                           `json:"disabled_at,omitempty"`
	ApplicationConfig      *bifrostApplicationConfiguration `json:"application_config,omitempty"`
}

type bifrostRegistrationRequest struct {
	CapabilityID           string   `json:"capability_id"`
	Name                   string   `json:"name"`
	Type                   string   `json:"type"`
	Source                 string   `json:"source"`
	Version                string   `json:"version"`
	Description            string   `json:"description"`
	Invocation             string   `json:"invocation,omitempty"`
	Resources              []string `json:"resources,omitempty"`
	Operations             []string `json:"operations,omitempty"`
	ProhibitedOperations   []string `json:"prohibited_operations,omitempty"`
	PantheonComponents     []string `json:"pantheon_components,omitempty"`
	NetworkAccess          bool     `json:"network_access"`
	SendsDataOffMachine    bool     `json:"sends_data_off_machine"`
	AuthenticationRequired bool     `json:"authentication_required"`
	Installed              bool     `json:"installed"`
	DownloadApproved       bool     `json:"download_approved"`
	InstallationApproved   bool     `json:"installation_approved"`
	UseApproved            bool     `json:"use_approved"`
	AutoUpdateApproved     bool     `json:"auto_update_approved"`
}

type bifrostApprovalEvent struct {
	EventID      string `json:"event_id"`
	CapabilityID string `json:"capability_id"`
	EventType    string `json:"event_type"`
	Approved     bool   `json:"approved"`
	ApprovedBy   string `json:"approved_by"`
	Scope        string `json:"scope,omitempty"`
	OccurredAt   string `json:"occurred_at"`
	AuditID      string `json:"audit_id,omitempty"`
}

type bifrostInvocation struct {
	ReceiptID           string `json:"receipt_id"`
	CapabilityID        string `json:"capability_id"`
	InvokedAt           string `json:"invoked_at"`
	RequestingComponent string `json:"requesting_component"`
	AuthorizationRef    string `json:"authorization_ref,omitempty"`
	Operation           string `json:"operation"`
	Outcome             string `json:"outcome"`
	FailureKind         string `json:"failure_kind,omitempty"`
	NetworkActivity     bool   `json:"network_activity"`
	ResultSummary       string `json:"result_summary,omitempty"`
	Error               string `json:"error,omitempty"`
	SourceID            string `json:"source_id,omitempty"`
}

type bifrostMailIntakeRequest struct {
	CapabilityID string `json:"capability_id,omitempty"`
	Account      string `json:"account,omitempty"`
	Mailbox      string `json:"mailbox,omitempty"`
	MaxMessages  int    `json:"max_messages,omitempty"`
	Bootstrap    bool   `json:"bootstrap,omitempty"`
	Query        string `json:"query,omitempty"`
}

type bifrostMailIntakeResult struct {
	RunID               string            `json:"run_id"`
	CapabilityID        string            `json:"capability_id"`
	Account             string            `json:"account"`
	Mailbox             string            `json:"mailbox"`
	BaselineEstablished bool              `json:"baseline_established"`
	Bootstrap           bool              `json:"bootstrap"`
	Examined            int               `json:"examined"`
	New                 int               `json:"new"`
	Unchanged           int               `json:"unchanged"`
	Blocked             int               `json:"blocked"`
	Failed              int               `json:"failed"`
	Truncated           bool              `json:"truncated"`
	Warnings            []string          `json:"warnings,omitempty"`
	OldestReceivedAt    string            `json:"oldest_received_at,omitempty"`
	NewestReceivedAt    string            `json:"newest_received_at,omitempty"`
	IntakeScan          *intakeScanResult `json:"intake_scan,omitempty"`
	ReceiptID           string            `json:"receipt_id,omitempty"`
}

type bifrostMailEnvelope struct {
	ID            string          `json:"id"`
	MessageID     string          `json:"message-id"`
	Date          string          `json:"date"`
	From          json.RawMessage `json:"from"`
	To            json.RawMessage `json:"to"`
	Subject       string          `json:"subject"`
	HasAttachment bool            `json:"has-attachment"`
}

type bifrostMailState struct {
	Account           string
	Mailbox           string
	InitialBaselineAt string
	LastSuccessAt     string
}

type bifrostAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size,omitempty"`
	Allowed     bool   `json:"allowed"`
	Path        string `json:"path,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Data        []byte `json:"-"`
}

type bifrostMailDocument struct {
	Body        string
	From        string
	To          string
	Subject     string
	Date        string
	Attachments []bifrostAttachment
}

func initializeBifrostSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS bifrost_capabilities (
			capability_id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL,
			source TEXT NOT NULL, version TEXT NOT NULL, description TEXT NOT NULL,
			invocation TEXT, resources_json TEXT NOT NULL, operations_json TEXT NOT NULL,
			prohibited_operations_json TEXT NOT NULL, pantheon_components_json TEXT NOT NULL,
			network_access INTEGER NOT NULL, sends_data_off_machine INTEGER NOT NULL,
			authentication_required INTEGER NOT NULL, installed INTEGER NOT NULL,
			enabled INTEGER NOT NULL, download_approved INTEGER NOT NULL,
			installation_approved INTEGER NOT NULL, registration_approved INTEGER NOT NULL,
			use_approved INTEGER NOT NULL, auto_update_approved INTEGER NOT NULL,
			registered_at TEXT NOT NULL, last_invoked_at TEXT, last_changed_at TEXT,
			disabled_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS bifrost_approval_events (
			event_id TEXT PRIMARY KEY, capability_id TEXT NOT NULL, event_type TEXT NOT NULL,
			approved INTEGER NOT NULL, approved_by TEXT NOT NULL, scope TEXT,
			occurred_at TEXT NOT NULL, audit_id TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS bifrost_invocations (
			receipt_id TEXT PRIMARY KEY, capability_id TEXT NOT NULL, invoked_at TEXT NOT NULL,
			requesting_component TEXT NOT NULL, authorization_ref TEXT, operation TEXT NOT NULL,
			outcome TEXT NOT NULL, failure_kind TEXT, network_activity INTEGER NOT NULL,
			result_summary TEXT, error TEXT, source_id TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS bifrost_application_configurations (
			capability_id TEXT PRIMARY KEY, application TEXT NOT NULL, endpoint TEXT NOT NULL,
			route TEXT NOT NULL, model TEXT NOT NULL, credential_env TEXT, config_path TEXT,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS bifrost_mail_state (
			account TEXT NOT NULL, mailbox TEXT NOT NULL, initial_baseline_at TEXT NOT NULL,
			last_success_at TEXT, PRIMARY KEY (account, mailbox)
		)`,
		`CREATE TABLE IF NOT EXISTS bifrost_mail_messages (
			source_id TEXT PRIMARY KEY, account TEXT NOT NULL, mailbox TEXT NOT NULL,
			message_id TEXT, received_at TEXT, subject TEXT, status TEXT NOT NULL,
			artifact_path TEXT, artifact_sha256 TEXT, intake_occurrence_id TEXT,
			attachments_json TEXT NOT NULL, warnings TEXT, processed_at TEXT, last_error TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS bifrost_invocations_time_idx ON bifrost_invocations(invoked_at DESC)`,
		`CREATE INDEX IF NOT EXISTS bifrost_mail_messages_status_idx ON bifrost_mail_messages(status, processed_at DESC)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func bifrostJSONArray(values []string) string {
	if values == nil {
		values = []string{}
	}
	data, _ := json.Marshal(values)
	return string(data)
}

func bifrostStringArray(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var result []string
	if json.Unmarshal([]byte(value), &result) != nil {
		return nil
	}
	return result
}

func bifrostBool(value int64) bool { return value == 1 }

func bifrostCapabilityFromRequest(request bifrostRegistrationRequest, now string) bifrostCapability {
	id := strings.TrimSpace(request.CapabilityID)
	if id == "" {
		id = bifrostSlug(request.Name)
	}
	return bifrostCapability{
		CapabilityID: id, Name: strings.TrimSpace(request.Name), Type: strings.TrimSpace(request.Type),
		Source: strings.TrimSpace(request.Source), Version: strings.TrimSpace(request.Version),
		Description: strings.TrimSpace(request.Description), Invocation: strings.TrimSpace(request.Invocation),
		Resources: append([]string(nil), request.Resources...), Operations: append([]string(nil), request.Operations...),
		ProhibitedOperations: append([]string(nil), request.ProhibitedOperations...),
		PantheonComponents:   append([]string(nil), request.PantheonComponents...),
		NetworkAccess:        request.NetworkAccess, SendsDataOffMachine: request.SendsDataOffMachine,
		AuthenticationRequired: request.AuthenticationRequired, Installed: request.Installed,
		DownloadApproved: request.DownloadApproved, InstallationApproved: request.InstallationApproved,
		RegistrationApproved: true, UseApproved: request.UseApproved,
		AutoUpdateApproved: request.AutoUpdateApproved, RegisteredAt: now,
		LastChangedAt: now,
	}
}

func bifrostSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == ':' || r == '_' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func validateBifrostRegistration(request bifrostRegistrationRequest) error {
	if strings.TrimSpace(request.Name) == "" || strings.TrimSpace(request.Source) == "" || strings.TrimSpace(request.Version) == "" {
		return errors.New("name, source, and version are required")
	}
	if strings.TrimSpace(request.Type) == "" {
		request.Type = bifrostCapabilityTypeCommand
	}
	if !bifrostCapabilityTypes[strings.TrimSpace(request.Type)] {
		return fmt.Errorf("unsupported Bifrost capability type %q", request.Type)
	}
	if bifrostSlug(request.Name) == "" && strings.TrimSpace(request.CapabilityID) == "" {
		return errors.New("capability_id is required when name has no safe identifier")
	}
	for field, values := range map[string][]string{"resources": request.Resources, "operations": request.Operations, "prohibited_operations": request.ProhibitedOperations, "pantheon_components": request.PantheonComponents} {
		for _, value := range values {
			if strings.TrimSpace(value) == "" || strings.IndexByte(value, 0) >= 0 {
				return fmt.Errorf("%s contains an invalid value", field)
			}
		}
	}
	for field, value := range map[string]string{"capability_id": request.CapabilityID, "name": request.Name, "source": request.Source, "version": request.Version, "description": request.Description, "invocation": request.Invocation} {
		if strings.IndexByte(value, 0) >= 0 {
			return fmt.Errorf("%s contains a NUL byte", field)
		}
	}
	return nil
}

func (d *daemon) registerBifrostCapability(request bifrostRegistrationRequest, auditID string) (bifrostCapability, error) {
	if err := initializeBifrostSchema(d.db); err != nil {
		return bifrostCapability{}, err
	}
	if strings.TrimSpace(request.Type) == "" {
		request.Type = bifrostCapabilityTypeCommand
	}
	if err := validateBifrostRegistration(request); err != nil {
		return bifrostCapability{}, err
	}
	now := time.Now().Format(time.RFC3339Nano)
	capability := bifrostCapabilityFromRequest(request, now)
	var existing bifrostCapability
	var resources, operations, prohibited, components string
	var network, offMachine, auth, installed, enabled, download, installation, registration, use, automatic int64
	err := d.db.QueryRow(`SELECT capability_id, name, type, source, version, description, COALESCE(invocation, ''), resources_json, operations_json, prohibited_operations_json, pantheon_components_json, network_access, sends_data_off_machine, authentication_required, installed, enabled, download_approved, installation_approved, registration_approved, use_approved, auto_update_approved, registered_at, COALESCE(last_invoked_at, ''), COALESCE(last_changed_at, ''), COALESCE(disabled_at, '') FROM bifrost_capabilities WHERE capability_id = ?`, capability.CapabilityID).Scan(&existing.CapabilityID, &existing.Name, &existing.Type, &existing.Source, &existing.Version, &existing.Description, &existing.Invocation, &resources, &operations, &prohibited, &components, &network, &offMachine, &auth, &installed, &enabled, &download, &installation, &registration, &use, &automatic, &existing.RegisteredAt, &existing.LastInvokedAt, &existing.LastChangedAt, &existing.DisabledAt)
	if err == nil {
		// Registration is idempotent for metadata, but approval can only move
		// forward through this explicit local-control operation.
		var approvals struct{ Download, Installation, Use, AutoUpdate bool }
		approvals.Download, approvals.Installation, approvals.Use, approvals.AutoUpdate = bifrostBool(download) || request.DownloadApproved, bifrostBool(installation) || request.InstallationApproved, bifrostBool(use) || request.UseApproved, bifrostBool(automatic) || request.AutoUpdateApproved
		capability.DownloadApproved, capability.InstallationApproved, capability.UseApproved, capability.AutoUpdateApproved = approvals.Download, approvals.Installation, approvals.Use, approvals.AutoUpdate
		existing.Resources, existing.Operations, existing.ProhibitedOperations, existing.PantheonComponents = bifrostStringArray(resources), bifrostStringArray(operations), bifrostStringArray(prohibited), bifrostStringArray(components)
		existing.Installed, existing.Enabled = bifrostBool(installed), bifrostBool(enabled)
		capability.Installed = existing.Installed || request.Installed
		capability.RegistrationApproved = true
		capability.Enabled = existing.Enabled && capability.Installed && capability.DownloadApproved && capability.InstallationApproved && capability.UseApproved
		capability.RegisteredAt = existing.RegisteredAt
		capability.LastInvokedAt, capability.DisabledAt = existing.LastInvokedAt, existing.DisabledAt
		_, err = d.db.Exec(`UPDATE bifrost_capabilities SET name = ?, type = ?, source = ?, version = ?, description = ?, invocation = ?, resources_json = ?, operations_json = ?, prohibited_operations_json = ?, pantheon_components_json = ?, network_access = ?, sends_data_off_machine = ?, authentication_required = ?, installed = ?, enabled = ?, download_approved = ?, installation_approved = ?, registration_approved = 1, use_approved = ?, auto_update_approved = ?, last_changed_at = ?, disabled_at = CASE WHEN ? THEN NULL ELSE disabled_at END WHERE capability_id = ?`, capability.Name, capability.Type, capability.Source, capability.Version, capability.Description, capability.Invocation, bifrostJSONArray(capability.Resources), bifrostJSONArray(capability.Operations), bifrostJSONArray(capability.ProhibitedOperations), bifrostJSONArray(capability.PantheonComponents), capability.NetworkAccess, capability.SendsDataOffMachine, capability.AuthenticationRequired, capability.Installed, capability.Enabled, capability.DownloadApproved, capability.InstallationApproved, capability.UseApproved, capability.AutoUpdateApproved, now, capability.Enabled, capability.CapabilityID)
		if err != nil {
			return bifrostCapability{}, err
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		capability.Enabled = capability.Installed && capability.DownloadApproved && capability.InstallationApproved && capability.UseApproved
		_, err = d.db.Exec(`INSERT INTO bifrost_capabilities (capability_id, name, type, source, version, description, invocation, resources_json, operations_json, prohibited_operations_json, pantheon_components_json, network_access, sends_data_off_machine, authentication_required, installed, enabled, download_approved, installation_approved, registration_approved, use_approved, auto_update_approved, registered_at, last_changed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?)`, capability.CapabilityID, capability.Name, capability.Type, capability.Source, capability.Version, capability.Description, capability.Invocation, bifrostJSONArray(capability.Resources), bifrostJSONArray(capability.Operations), bifrostJSONArray(capability.ProhibitedOperations), bifrostJSONArray(capability.PantheonComponents), capability.NetworkAccess, capability.SendsDataOffMachine, capability.AuthenticationRequired, capability.Installed, capability.Enabled, capability.DownloadApproved, capability.InstallationApproved, capability.UseApproved, capability.AutoUpdateApproved, capability.RegisteredAt, capability.LastChangedAt)
		if err != nil {
			return bifrostCapability{}, err
		}
	} else {
		return bifrostCapability{}, err
	}
	for eventType, approved := range map[string]bool{"download": request.DownloadApproved, "installation": request.InstallationApproved, "use": request.UseApproved, "automatic_update": request.AutoUpdateApproved} {
		if approved {
			if _, err := d.recordBifrostApproval(capability.CapabilityID, eventType, true, "local-control", "", auditID); err != nil {
				return bifrostCapability{}, err
			}
		}
	}
	if _, err := d.recordBifrostApproval(capability.CapabilityID, "registration", true, "local-control", "registry metadata only", auditID); err != nil {
		return bifrostCapability{}, err
	}
	return d.loadBifrostCapability(capability.CapabilityID)
}

func (d *daemon) loadBifrostCapability(id string) (bifrostCapability, error) {
	var capability bifrostCapability
	var resources, operations, prohibited, components string
	var network, offMachine, auth, installed, enabled, download, installation, registration, use, automatic int64
	err := d.db.QueryRow(`SELECT capability_id, name, type, source, version, description, COALESCE(invocation, ''), resources_json, operations_json, prohibited_operations_json, pantheon_components_json, network_access, sends_data_off_machine, authentication_required, installed, enabled, download_approved, installation_approved, registration_approved, use_approved, auto_update_approved, registered_at, COALESCE(last_invoked_at, ''), COALESCE(last_changed_at, ''), COALESCE(disabled_at, '') FROM bifrost_capabilities WHERE capability_id = ?`, id).Scan(&capability.CapabilityID, &capability.Name, &capability.Type, &capability.Source, &capability.Version, &capability.Description, &capability.Invocation, &resources, &operations, &prohibited, &components, &network, &offMachine, &auth, &installed, &enabled, &download, &installation, &registration, &use, &automatic, &capability.RegisteredAt, &capability.LastInvokedAt, &capability.LastChangedAt, &capability.DisabledAt)
	if err != nil {
		return bifrostCapability{}, err
	}
	capability.Resources, capability.Operations, capability.ProhibitedOperations, capability.PantheonComponents = bifrostStringArray(resources), bifrostStringArray(operations), bifrostStringArray(prohibited), bifrostStringArray(components)
	capability.NetworkAccess, capability.SendsDataOffMachine, capability.AuthenticationRequired = bifrostBool(network), bifrostBool(offMachine), bifrostBool(auth)
	capability.Installed, capability.Enabled, capability.DownloadApproved, capability.InstallationApproved = bifrostBool(installed), bifrostBool(enabled), bifrostBool(download), bifrostBool(installation)
	capability.RegistrationApproved, capability.UseApproved, capability.AutoUpdateApproved = bifrostBool(registration), bifrostBool(use), bifrostBool(automatic)
	applicationConfig, err := d.loadBifrostApplicationConfiguration(id)
	if err != nil {
		return bifrostCapability{}, err
	}
	capability.ApplicationConfig = applicationConfig
	return capability, nil
}

func (d *daemon) listBifrostCapabilities() ([]bifrostCapability, error) {
	rows, err := d.db.Query(`SELECT capability_id FROM bifrost_capabilities ORDER BY name, capability_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []bifrostCapability
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		capability, err := d.loadBifrostCapability(id)
		if err != nil {
			return nil, err
		}
		result = append(result, capability)
	}
	return result, rows.Err()
}

func (d *daemon) recordBifrostApproval(capabilityID, eventType string, approved bool, approvedBy, scope, auditID string) (bifrostApprovalEvent, error) {
	eventID, err := newLocalID("bifrost-approval-")
	if err != nil {
		return bifrostApprovalEvent{}, err
	}
	event := bifrostApprovalEvent{EventID: eventID, CapabilityID: capabilityID, EventType: eventType, Approved: approved, ApprovedBy: approvedBy, Scope: scope, OccurredAt: time.Now().Format(time.RFC3339Nano), AuditID: auditID}
	_, err = d.db.Exec(`INSERT INTO bifrost_approval_events (event_id, capability_id, event_type, approved, approved_by, scope, occurred_at, audit_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, event.EventID, event.CapabilityID, event.EventType, event.Approved, event.ApprovedBy, nullableString(event.Scope), event.OccurredAt, nullableString(event.AuditID))
	return event, err
}

func (d *daemon) bifrostApprovals(limit int) ([]bifrostApprovalEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := d.db.Query(`SELECT event_id, capability_id, event_type, approved, approved_by, COALESCE(scope, ''), occurred_at, COALESCE(audit_id, '') FROM bifrost_approval_events ORDER BY occurred_at DESC, event_id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []bifrostApprovalEvent
	for rows.Next() {
		var event bifrostApprovalEvent
		var approved int64
		if err := rows.Scan(&event.EventID, &event.CapabilityID, &event.EventType, &approved, &event.ApprovedBy, &event.Scope, &event.OccurredAt, &event.AuditID); err != nil {
			return nil, err
		}
		event.Approved = approved == 1
		result = append(result, event)
	}
	return result, rows.Err()
}

func (d *daemon) recordBifrostInvocation(capabilityID, requester, authorizationRef, operation, outcome, failureKind, summary, errText, sourceID string, network bool) (bifrostInvocation, error) {
	receiptID, err := newLocalID("bifrost-receipt-")
	if err != nil {
		return bifrostInvocation{}, err
	}
	now := time.Now().Format(time.RFC3339Nano)
	receipt := bifrostInvocation{ReceiptID: receiptID, CapabilityID: capabilityID, InvokedAt: now, RequestingComponent: requester, AuthorizationRef: authorizationRef, Operation: operation, Outcome: outcome, FailureKind: failureKind, NetworkActivity: network, ResultSummary: summary, Error: bifrostSafeError(errText), SourceID: sourceID}
	_, err = d.db.Exec(`INSERT INTO bifrost_invocations (receipt_id, capability_id, invoked_at, requesting_component, authorization_ref, operation, outcome, failure_kind, network_activity, result_summary, error, source_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, receipt.ReceiptID, receipt.CapabilityID, receipt.InvokedAt, receipt.RequestingComponent, nullableString(receipt.AuthorizationRef), receipt.Operation, receipt.Outcome, nullableString(receipt.FailureKind), receipt.NetworkActivity, nullableString(receipt.ResultSummary), nullableString(receipt.Error), nullableString(receipt.SourceID))
	return receipt, err
}

func (d *daemon) bifrostInvocations(limit int) ([]bifrostInvocation, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := d.db.Query(`SELECT receipt_id, capability_id, invoked_at, requesting_component, COALESCE(authorization_ref, ''), operation, outcome, COALESCE(failure_kind, ''), network_activity, COALESCE(result_summary, ''), COALESCE(error, ''), COALESCE(source_id, '') FROM bifrost_invocations ORDER BY invoked_at DESC, receipt_id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []bifrostInvocation
	for rows.Next() {
		var receipt bifrostInvocation
		var network int64
		if err := rows.Scan(&receipt.ReceiptID, &receipt.CapabilityID, &receipt.InvokedAt, &receipt.RequestingComponent, &receipt.AuthorizationRef, &receipt.Operation, &receipt.Outcome, &receipt.FailureKind, &network, &receipt.ResultSummary, &receipt.Error, &receipt.SourceID); err != nil {
			return nil, err
		}
		receipt.NetworkActivity = network == 1
		result = append(result, receipt)
	}
	return result, rows.Err()
}

func (d *daemon) setBifrostEnabled(id string, enabled bool) (bifrostCapability, error) {
	capability, err := d.loadBifrostCapability(id)
	if err != nil {
		return bifrostCapability{}, err
	}
	if enabled && !(capability.Installed && capability.DownloadApproved && capability.InstallationApproved && capability.RegistrationApproved && capability.UseApproved) {
		return bifrostCapability{}, bifrostGateError{Kind: bifrostFailureApproval, Message: "capability requires recorded download, installation, registration, and use approval before enablement"}
	}
	now := time.Now().Format(time.RFC3339Nano)
	disabledAt := ""
	if !enabled {
		disabledAt = now
	}
	_, err = d.db.Exec(`UPDATE bifrost_capabilities SET enabled = ?, disabled_at = ?, last_changed_at = ? WHERE capability_id = ?`, enabled, nullableString(disabledAt), now, id)
	if err != nil {
		return bifrostCapability{}, err
	}
	return d.loadBifrostCapability(id)
}

type bifrostGateError struct {
	Kind    string
	Message string
}

func (e bifrostGateError) Error() string { return e.Message }

func (d *daemon) authorizeBifrostInvocation(id string) (bifrostCapability, error) {
	if !d.cfg.BifrostEnabled {
		return bifrostCapability{}, bifrostGateError{Kind: bifrostFailurePolicy, Message: "Bifrost is disabled by configuration"}
	}
	capability, err := d.loadBifrostCapability(id)
	if errors.Is(err, sql.ErrNoRows) {
		return bifrostCapability{}, bifrostGateError{Kind: bifrostFailureApproval, Message: "capability is not registered with Bifrost"}
	}
	if err != nil {
		return bifrostCapability{}, err
	}
	if !capability.DownloadApproved || !capability.InstallationApproved || !capability.RegistrationApproved || !capability.UseApproved || !capability.Installed || !capability.Enabled {
		return bifrostCapability{}, bifrostGateError{Kind: bifrostFailureApproval, Message: "capability is present but not fully approved and enabled"}
	}
	paused, err := d.paused()
	if err != nil {
		return bifrostCapability{}, err
	}
	if paused {
		return bifrostCapability{}, bifrostGateError{Kind: bifrostFailurePrometheus, Message: "Prometheus is paused; new Bifrost activity is blocked"}
	}
	return capability, nil
}

func bifrostSafeError(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, marker := range []string{"password", "passwd", "token", "secret", "api_key", "authorization", "cookie"} {
		lower := strings.ToLower(value)
		if index := strings.Index(lower, marker); index >= 0 {
			value = value[:index] + "[redacted]"
			break
		}
	}
	if len(value) > 1000 {
		value = value[:1000] + "…"
	}
	return value
}

func bifrostFailureKind(err error, output string) string {
	var gate bifrostGateError
	if errors.As(err, &gate) {
		return gate.Kind
	}
	lower := strings.ToLower(err.Error() + " " + output)
	for _, marker := range []string{"authentication", "credential", "login", "oauth", "password"} {
		if strings.Contains(lower, marker) {
			return bifrostFailureAuthentication
		}
	}
	for _, marker := range []string{"not found", "no such file", "unavailable", "connection refused", "timed out"} {
		if strings.Contains(lower, marker) {
			return bifrostFailureUnavailable
		}
	}
	return bifrostFailureExternal
}

func (d *daemon) bifrostStatus() (map[string]any, error) {
	if err := initializeBifrostSchema(d.db); err != nil {
		return nil, err
	}
	capabilities, err := d.listBifrostCapabilities()
	if err != nil {
		return nil, err
	}
	var invocations, approvals int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM bifrost_invocations`).Scan(&invocations); err != nil {
		return nil, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM bifrost_approval_events`).Scan(&approvals); err != nil {
		return nil, err
	}
	paused, err := d.paused()
	if err != nil {
		return nil, err
	}
	return map[string]any{"paused": paused, "capabilities": capabilities, "approval_events": approvals, "invocations": invocations, "presence_is_not_permission": true, "research_is_not_permission": true, "mail": map[string]any{"account": d.cfg.BifrostMailAccount, "mailbox": d.cfg.BifrostMailMailbox, "capability_id": d.cfg.BifrostMailCapabilityID, "schedule": d.cfg.BifrostMailSchedule, "max_messages": d.cfg.BifrostMailMaxMessages, "hermes_migration": "replacement must pass cutover proof before old jobs are retired"}}, nil
}

func bifrostSafePathComponent(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." || strings.IndexByte(value, 0) >= 0 || strings.ContainsAny(value, `/\\`) {
		return "", errors.New("unsafe path component")
	}
	return bifrostSlug(value), nil
}

func bifrostHashBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func bifrostHashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return bifrostHashBytes(data), nil
}

func (d *daemon) bifrostMailState(account, mailbox string) (bifrostMailState, error) {
	var state bifrostMailState
	err := d.db.QueryRow(`SELECT account, mailbox, initial_baseline_at, COALESCE(last_success_at, '') FROM bifrost_mail_state WHERE account = ? AND mailbox = ?`, account, mailbox).Scan(&state.Account, &state.Mailbox, &state.InitialBaselineAt, &state.LastSuccessAt)
	return state, err
}

func (d *daemon) runHimalaya(ctx context.Context, args ...string) ([]byte, string, error) {
	command := exec.CommandContext(ctx, himalayaCommand, args...)
	output, err := command.Output()
	if err == nil {
		return output, "", nil
	}
	errText := err.Error()
	if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
		errText += ": " + string(exitErr.Stderr)
	}
	return output, bifrostSafeError(errText), err
}

func (d *daemon) searchBifrostMail(ctx context.Context, account, mailbox string, max int, since string, bootstrap bool, requestedQuery string) ([]bifrostMailEnvelope, bool, string, error) {
	args := []string{"--account", account, "envelope", "search", "--mailbox", mailbox, "--page", "1", "--page-size", fmt.Sprint(max), "--json"}
	if strings.TrimSpace(requestedQuery) != "" {
		output, errText, err := d.runHimalaya(ctx, append(args, requestedQuery)...)
		if err != nil {
			return nil, false, errText, err
		}
		var document struct {
			Envelopes []bifrostMailEnvelope `json:"envelopes"`
		}
		if err := json.Unmarshal(output, &document); err != nil {
			return nil, false, bifrostSafeError(err.Error()), bifrostGateError{Kind: bifrostFailureMalformed, Message: "Himalaya returned invalid envelope JSON"}
		}
		return document.Envelopes, len(document.Envelopes) >= max, "", nil
	}
	query := "order by date asc"
	if !bootstrap && since != "" {
		parsed, err := time.Parse(time.RFC3339Nano, since)
		if err != nil {
			return nil, false, "", err
		}
		query = "after " + parsed.UTC().Format("2006-01-02") + " order by date asc"
	}
	args = append(args, query)
	output, errText, err := d.runHimalaya(ctx, args...)
	if err != nil {
		return nil, false, errText, err
	}
	var document struct {
		Envelopes []bifrostMailEnvelope `json:"envelopes"`
	}
	if err := json.Unmarshal(output, &document); err != nil {
		return nil, false, bifrostSafeError(err.Error()), bifrostGateError{Kind: bifrostFailureMalformed, Message: "Himalaya returned invalid envelope JSON"}
	}
	return document.Envelopes, len(document.Envelopes) >= max, "", nil
}

func bifrostRawHeader(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var addresses []map[string]any
	if json.Unmarshal(raw, &addresses) == nil {
		var parts []string
		for _, address := range addresses {
			name, _ := address["name"].(string)
			email, _ := address["email"].(string)
			if email == "" {
				email, _ = address["address"].(string)
			}
			if name != "" && email != "" {
				parts = append(parts, name+" <"+email+">")
			} else if email != "" {
				parts = append(parts, email)
			}
		}
		return strings.Join(parts, ", ")
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return value
	}
	return strings.TrimSpace(string(raw))
}

func bifrostEnvelopeSourceID(envelope bifrostMailEnvelope) string {
	if value := strings.TrimSpace(envelope.MessageID); value != "" {
		return value
	}
	return strings.TrimSpace(envelope.ID)
}

func bifrostAllowedAttachment(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch contentType {
	case "text/plain", "text/markdown", "application/pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return true
	default:
		return false
	}
}

func normalizeBifrostBody(body []byte) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(string(body), "\r\n", "\n"), "\r", "\n"))
}

func parseBifrostMail(raw []byte, envelope bifrostMailEnvelope) (bifrostMailDocument, error) {
	document := bifrostMailDocument{From: bifrostRawHeader(envelope.From), To: bifrostRawHeader(envelope.To), Subject: envelope.Subject, Date: envelope.Date}
	message, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		document.Body = normalizeBifrostBody(raw)
		return document, nil
	}
	if value := message.Header.Get("From"); value != "" {
		document.From = value
	}
	if value := message.Header.Get("To"); value != "" {
		document.To = value
	}
	if value := message.Header.Get("Subject"); value != "" {
		document.Subject = value
	}
	if value := message.Header.Get("Date"); value != "" {
		document.Date = value
	}
	contentType, params, contentErr := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if contentErr != nil || !strings.HasPrefix(strings.ToLower(contentType), "multipart/") {
		body, readErr := io.ReadAll(io.LimitReader(message.Body, bifrostMailReadLimit+1))
		if readErr != nil {
			return bifrostMailDocument{}, readErr
		}
		if len(body) > bifrostMailReadLimit {
			return bifrostMailDocument{}, bifrostGateError{Kind: bifrostFailurePolicy, Message: "message body exceeds the bounded Bifrost intake limit"}
		}
		document.Body = normalizeBifrostBody(body)
		return document, nil
	}
	reader := multipart.NewReader(message.Body, params["boundary"])
	var bodyParts []string
	for {
		part, nextErr := reader.NextPart()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return bifrostMailDocument{}, nextErr
		}
		filename := strings.TrimSpace(part.FileName())
		partType := strings.ToLower(strings.Split(part.Header.Get("Content-Type"), ";")[0])
		if filename != "" || strings.HasPrefix(strings.ToLower(part.Header.Get("Content-Disposition")), "attachment") {
			attachment := bifrostAttachment{Filename: filename, ContentType: partType}
			if filename == "" {
				filename = "unnamed-attachment"
				attachment.Filename = filename
			}
			if !bifrostAllowedAttachment(partType) {
				attachment.Reason = "content type is not in the explicit safe allowlist"
				document.Attachments = append(document.Attachments, attachment)
				continue
			}
			data, readErr := io.ReadAll(io.LimitReader(part, bifrostAttachmentLimit+1))
			if readErr != nil {
				return bifrostMailDocument{}, readErr
			}
			if len(data) > bifrostAttachmentLimit {
				attachment.Reason = "attachment exceeds the bounded size limit"
				document.Attachments = append(document.Attachments, attachment)
				continue
			}
			attachment.Allowed, attachment.Size, attachment.Data = true, len(data), data
			document.Attachments = append(document.Attachments, attachment)
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(part, bifrostMailReadLimit+1))
		if readErr != nil {
			return bifrostMailDocument{}, readErr
		}
		bodyParts = append(bodyParts, string(data))
	}
	document.Body = normalizeBifrostBody([]byte(strings.Join(bodyParts, "\n\n")))
	return document, nil
}

func parseBifrostMailDate(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}
	return mail.ParseDate(value)
}

func (d *daemon) writeBifrostMailArtifact(envelope bifrostMailEnvelope, document bifrostMailDocument, account, mailbox string) (string, string, error) {
	accountPath, err := bifrostSafePathComponent(account)
	if err != nil {
		return "", "", err
	}
	if d.cfg.IntakePath == "" {
		return "", "", errors.New("Intake path is not configured")
	}
	root := filepath.Join(d.cfg.IntakePath, "Bifrost Mail", accountPath)
	if err := ensureNoSymlinkComponents(root); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", "", err
	}
	sourceID := bifrostEnvelopeSourceID(envelope)
	if sourceID == "" {
		return "", "", errors.New("message has no stable source identifier")
	}
	prefix := "message-" + hex.EncodeToString(sha256.New().Sum(nil))
	digest := sha256.Sum256([]byte(sourceID))
	prefix = "message-" + hex.EncodeToString(digest[:])
	artifactPath := filepath.Join(root, prefix+".md")
	var attachmentLines []string
	for index := range document.Attachments {
		attachment := &document.Attachments[index]
		if !attachment.Allowed {
			attachmentLines = append(attachmentLines, fmt.Sprintf("- `%s` (%s) — blocked: %s", strings.ReplaceAll(attachment.Filename, "`", ""), attachment.ContentType, attachment.Reason))
			continue
		}
		name := filepath.Base(attachment.Filename)
		name = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(name, "\x00", ""), "..", ""))
		if name == "" || name == "." {
			name = fmt.Sprintf("attachment-%d", index+1)
		}
		attachmentPath := filepath.Join(root, prefix+"-"+bifrostSlug(name))
		if existing, readErr := os.ReadFile(attachmentPath); readErr == nil {
			if bifrostHashBytes(existing) != bifrostHashBytes(attachment.Data) {
				return "", "", bifrostGateError{Kind: bifrostFailureArtifact, Message: "existing allowed attachment was modified; refusing overwrite"}
			}
		} else if errors.Is(readErr, os.ErrNotExist) {
			if err := os.WriteFile(attachmentPath, attachment.Data, 0o600); err != nil {
				return "", "", err
			}
		} else {
			return "", "", readErr
		}
		attachment.Path = attachmentPath
		attachmentLines = append(attachmentLines, fmt.Sprintf("- `%s` (%s) — ingested at `%s`", strings.ReplaceAll(attachment.Filename, "`", ""), attachment.ContentType, attachmentPath))
	}
	if len(attachmentLines) == 0 {
		attachmentLines = []string{"- None"}
	}
	content := fmt.Sprintf("---\nsource: %s\nsource_id: %s\nmessage_id: %s\naccount: %s\nmailbox: %s\nreceived_at: %s\nprovenance: Bifrost read-only Himalaya adapter\n---\n\n# %s\n\n- From: %s\n- To: %s\n- Subject: %s\n- Date: %s\n\n## Body\n\n%s\n\n## Attachments\n\n%s\n", "bifrost://"+account+"/"+mailbox+"/"+sourceID, sourceID, envelope.MessageID, account, mailbox, document.Date, strings.TrimSpace(document.Subject), document.From, document.To, document.Subject, document.Date, document.Body, strings.Join(attachmentLines, "\n"))
	data := []byte(normalizeBifrostBody([]byte(content)) + "\n")
	hash := bifrostHashBytes(data)
	if existing, readErr := os.ReadFile(artifactPath); readErr == nil {
		if bifrostHashBytes(existing) != hash {
			return "", "", bifrostGateError{Kind: bifrostFailureArtifact, Message: "existing mail Intake artifact was modified; refusing overwrite"}
		}
	} else if errors.Is(readErr, os.ErrNotExist) {
		if err := os.WriteFile(artifactPath, data, 0o600); err != nil {
			return "", "", err
		}
	} else {
		return "", "", readErr
	}
	return artifactPath, hash, nil
}

func (d *daemon) persistBifrostMailMessage(sourceID, account, mailbox string, envelope bifrostMailEnvelope, status, artifactPath, artifactHash, occurrenceID, attachments, warnings, lastError string) error {
	_, err := d.db.Exec(`INSERT INTO bifrost_mail_messages (source_id, account, mailbox, message_id, received_at, subject, status, artifact_path, artifact_sha256, intake_occurrence_id, attachments_json, warnings, processed_at, last_error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(source_id) DO UPDATE SET status = excluded.status, artifact_path = excluded.artifact_path, artifact_sha256 = excluded.artifact_sha256, intake_occurrence_id = excluded.intake_occurrence_id, attachments_json = excluded.attachments_json, warnings = excluded.warnings, processed_at = excluded.processed_at, last_error = excluded.last_error`, sourceID, account, mailbox, envelope.MessageID, envelope.Date, envelope.Subject, status, nullableString(artifactPath), nullableString(artifactHash), nullableString(occurrenceID), attachments, nullableString(warnings), nullableString(time.Now().Format(time.RFC3339Nano)), nullableString(lastError))
	return err
}

func (d *daemon) bifrostMailOccurrence(relative string) (string, error) {
	var occurrence string
	err := d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences WHERE relative_path = ? ORDER BY first_seen_at DESC LIMIT 1`, filepath.ToSlash(relative)).Scan(&occurrence)
	return occurrence, err
}

func (d *daemon) bifrostMailIntake(ctx context.Context, request bifrostMailIntakeRequest) (bifrostMailIntakeResult, error) {
	if err := initializeBifrostSchema(d.db); err != nil {
		return bifrostMailIntakeResult{}, err
	}
	capabilityID := strings.TrimSpace(request.CapabilityID)
	if capabilityID == "" {
		capabilityID = d.cfg.BifrostMailCapabilityID
	}
	account, mailbox := strings.TrimSpace(request.Account), strings.TrimSpace(request.Mailbox)
	if account == "" {
		account = d.cfg.BifrostMailAccount
	}
	if mailbox == "" {
		mailbox = d.cfg.BifrostMailMailbox
	}
	max := request.MaxMessages
	if max <= 0 {
		max = d.cfg.BifrostMailMaxMessages
	}
	if max <= 0 {
		max = bifrostMailMaxDefault
	}
	if max > 250 {
		max = 250
	}
	result := bifrostMailIntakeResult{RunID: "", CapabilityID: capabilityID, Account: account, Mailbox: mailbox, Bootstrap: request.Bootstrap}
	result.RunID, _ = newLocalID("bifrost-mail-")
	if account == "" || mailbox == "" {
		return result, errors.New("Bifrost mail account and mailbox are required")
	}
	if strings.IndexByte(account, 0) >= 0 || strings.IndexByte(mailbox, 0) >= 0 {
		return result, errors.New("Bifrost mail account and mailbox may not contain NUL bytes")
	}
	capability, err := d.authorizeBifrostInvocation(capabilityID)
	if err != nil {
		kind := bifrostFailureKind(err, "")
		receipt, receiptErr := d.recordBifrostInvocation(capabilityID, "bifrost.mail.intake", capabilityID, "mail.search", bifrostOutcomeBlocked, kind, "mail intake did not start", err.Error(), "", true)
		if receiptErr == nil {
			result.ReceiptID = receipt.ReceiptID
		}
		return result, err
	}
	state, stateErr := d.bifrostMailState(account, mailbox)
	if errors.Is(stateErr, sql.ErrNoRows) {
		baseline := time.Now().UTC().Format(time.RFC3339Nano)
		if _, err := d.db.Exec(`INSERT INTO bifrost_mail_state (account, mailbox, initial_baseline_at) VALUES (?, ?, ?)`, account, mailbox, baseline); err != nil {
			return result, err
		}
		state = bifrostMailState{Account: account, Mailbox: mailbox, InitialBaselineAt: baseline}
		if !request.Bootstrap {
			result.BaselineEstablished = true
			receipt, receiptErr := d.recordBifrostInvocation(capability.CapabilityID, "bifrost.mail.intake", capability.CapabilityID, "mail.search", bifrostOutcomeCompleted, "", "initial bounded baseline established; historical mail excluded", "", "", true)
			if receiptErr == nil {
				result.ReceiptID = receipt.ReceiptID
			}
			return result, nil
		}
	}
	if stateErr != nil && !errors.Is(stateErr, sql.ErrNoRows) {
		return result, stateErr
	}
	since := state.LastSuccessAt
	envelopes, truncated, searchError, err := d.searchBifrostMail(ctx, account, mailbox, max, since, request.Bootstrap, request.Query)
	if err != nil {
		kind := bifrostFailureKind(err, searchError)
		receipt, receiptErr := d.recordBifrostInvocation(capability.CapabilityID, "bifrost.mail.intake", capability.CapabilityID, "mail.search", bifrostOutcomeFailed, kind, "bounded mailbox search failed", searchError+" "+err.Error(), "", true)
		if receiptErr == nil {
			result.ReceiptID = receipt.ReceiptID
		}
		return result, err
	}
	searchReceipt, receiptErr := d.recordBifrostInvocation(capability.CapabilityID, "bifrost.mail.intake", capability.CapabilityID, "mail.search", bifrostOutcomeCompleted, "", fmt.Sprintf("bounded search returned %d envelope(s)", len(envelopes)), "", "", true)
	if receiptErr == nil {
		result.ReceiptID = searchReceipt.ReceiptID
	}
	result.Truncated = truncated
	if truncated {
		result.Warnings = append(result.Warnings, fmt.Sprintf("mailbox result reached the bounded %d-message limit; no additional IDs were discovered", max))
	}
	newest := state.LastSuccessAt
	allSucceeded := true
	for _, envelope := range envelopes {
		result.Examined++
		sourceID := bifrostEnvelopeSourceID(envelope)
		if sourceID == "" {
			result.Blocked++
			allSucceeded = false
			result.Warnings = append(result.Warnings, "mail envelope without a stable source identifier was blocked")
			continue
		}
		received, dateErr := parseBifrostMailDate(envelope.Date)
		if dateErr == nil {
			receivedText := received.UTC().Format(time.RFC3339Nano)
			if result.NewestReceivedAt == "" || receivedText > result.NewestReceivedAt {
				result.NewestReceivedAt = receivedText
			}
			if result.OldestReceivedAt == "" || receivedText < result.OldestReceivedAt {
				result.OldestReceivedAt = receivedText
			}
		}
		var knownStatus, knownArtifact, knownHash string
		if err := d.db.QueryRow(`SELECT status, COALESCE(artifact_path, ''), COALESCE(artifact_sha256, '') FROM bifrost_mail_messages WHERE source_id = ?`, sourceID).Scan(&knownStatus, &knownArtifact, &knownHash); err == nil {
			if knownStatus == "artifact_modified" {
				result.Blocked++
				allSucceeded = false
				continue
			}
			if knownStatus == "completed" || knownStatus == "blocked_attachment" {
				if knownStatus != "artifact_modified" && knownArtifact != "" && knownHash != "" {
					actualHash, hashErr := bifrostHashFile(knownArtifact)
					if hashErr != nil || actualHash != knownHash {
						message := "existing mail Intake artifact was modified; refusing overwrite"
						if hashErr != nil {
							message = "existing mail Intake artifact is unavailable; refusing replay"
						}
						result.Blocked++
						allSucceeded = false
						_ = d.persistBifrostMailMessage(sourceID, account, mailbox, envelope, "artifact_modified", knownArtifact, knownHash, "", "[]", "artifact protection", message)
						_, _ = d.recordBifrostInvocation(capability.CapabilityID, "bifrost.mail.intake", capability.CapabilityID, "mail.persist", bifrostOutcomeBlocked, bifrostFailureArtifact, "mail Intake artifact was protected from overwrite", message, sourceID, false)
						continue
					}
				}
				result.Unchanged++
				continue
			}
		}
		if dateErr != nil {
			result.Blocked++
			allSucceeded = false
			_ = d.persistBifrostMailMessage(sourceID, account, mailbox, envelope, "blocked", "", "", "", "[]", "missing or invalid received timestamp", dateErr.Error())
			_, _ = d.recordBifrostInvocation(capability.CapabilityID, "bifrost.mail.intake", capability.CapabilityID, "mail.message", bifrostOutcomeBlocked, bifrostFailureMalformed, "message lacked a trustworthy received timestamp", dateErr.Error(), sourceID, true)
			continue
		}
		if newest == "" || received.After(func() time.Time { parsed, _ := parseBifrostMailDate(newest); return parsed }()) {
			newest = received.UTC().Format(time.RFC3339Nano)
		}
		// Himalaya envelope search returns both a provider-native ID and the
		// RFC 5322 Message-ID header. The latter is our durable source key, but
		// message read requires the provider-native ID (for Gmail, typically a
		// numeric IMAP UID or backend ID). Keep the stable source key for
		// deduplication and receipts, while using the provider ID for retrieval.
		readID := strings.TrimSpace(envelope.ID)
		if readID == "" {
			readID = sourceID
		}
		messageRaw, readErrorText, readErr := d.runHimalaya(ctx, "--account", account, "message", "read", "--mailbox", mailbox, readID)
		if readErr != nil {
			result.Failed++
			allSucceeded = false
			kind := bifrostFailureKind(readErr, readErrorText)
			_ = d.persistBifrostMailMessage(sourceID, account, mailbox, envelope, "failed", "", "", "", "[]", "mail read failed", readErrorText)
			_, _ = d.recordBifrostInvocation(capability.CapabilityID, "bifrost.mail.intake", capability.CapabilityID, "mail.message", bifrostOutcomeFailed, kind, "message read failed; source remains in provider", readErrorText, sourceID, true)
			continue
		}
		messageReceipt, _ := d.recordBifrostInvocation(capability.CapabilityID, "bifrost.mail.intake", capability.CapabilityID, "mail.message", bifrostOutcomeCompleted, "", "message read completed; provider source unchanged", "", sourceID, true)
		document, parseErr := parseBifrostMail(messageRaw, envelope)
		if parseErr != nil {
			result.Failed++
			allSucceeded = false
			_ = d.persistBifrostMailMessage(sourceID, account, mailbox, envelope, "failed", "", "", "", "[]", "message normalization failed", parseErr.Error())
			_, _ = d.recordBifrostInvocation(capability.CapabilityID, "bifrost.mail.intake", capability.CapabilityID, "mail.normalize", bifrostOutcomeFailed, bifrostFailureLocal, "message normalization failed", parseErr.Error(), sourceID, false)
			continue
		}
		artifactPath, artifactHash, writeErr := d.writeBifrostMailArtifact(envelope, document, account, mailbox)
		attachmentsJSON, _ := json.Marshal(document.Attachments)
		if writeErr != nil {
			result.Blocked++
			allSucceeded = false
			kind := bifrostFailureKind(writeErr, "")
			if gate, ok := writeErr.(bifrostGateError); ok {
				kind = gate.Kind
			}
			_ = d.persistBifrostMailMessage(sourceID, account, mailbox, envelope, "artifact_modified", "", "", "", string(attachmentsJSON), "artifact was not overwritten", writeErr.Error())
			_, _ = d.recordBifrostInvocation(capability.CapabilityID, "bifrost.mail.intake", capability.CapabilityID, "mail.persist", bifrostOutcomeBlocked, kind, "mail Intake artifact was protected from overwrite", writeErr.Error(), sourceID, false)
			continue
		}
		if messageReceipt.ReceiptID == "" {
			messageReceipt.ReceiptID = result.ReceiptID
		}
		scanResult, scanErr := d.runIntakeScan()
		result.IntakeScan = &scanResult
		if scanErr != nil {
			result.Failed++
			allSucceeded = false
			_ = d.persistBifrostMailMessage(sourceID, account, mailbox, envelope, "failed", artifactPath, artifactHash, "", string(attachmentsJSON), "Intake scan failed", scanErr.Error())
			_, _ = d.recordBifrostInvocation(capability.CapabilityID, "bifrost.mail.intake", capability.CapabilityID, "intake.scan", bifrostOutcomeFailed, bifrostFailureLocal, "mail artifact was preserved but Intake scan failed", scanErr.Error(), sourceID, false)
			continue
		}
		relative, _ := filepath.Rel(d.cfg.IntakePath, artifactPath)
		occurrenceID, occurrenceErr := d.bifrostMailOccurrence(relative)
		if occurrenceErr != nil {
			result.Failed++
			allSucceeded = false
			_ = d.persistBifrostMailMessage(sourceID, account, mailbox, envelope, "failed", artifactPath, artifactHash, "", string(attachmentsJSON), "Intake occurrence was not found", occurrenceErr.Error())
			continue
		}
		warnings := ""
		status := "completed"
		for _, attachment := range document.Attachments {
			if !attachment.Allowed {
				status = "blocked_attachment"
				if warnings != "" {
					warnings += "; "
				}
				warnings += attachment.Filename + ": " + attachment.Reason
			}
		}
		if status == "blocked_attachment" {
			result.Blocked++
			result.Warnings = append(result.Warnings, warnings)
		}
		result.New++
		if err := d.persistBifrostMailMessage(sourceID, account, mailbox, envelope, status, artifactPath, artifactHash, occurrenceID, string(attachmentsJSON), warnings, ""); err != nil {
			return result, err
		}
	}
	if strings.TrimSpace(request.Query) == "" && allSucceeded && result.Failed == 0 && result.Blocked == 0 {
		if _, err := d.db.Exec(`UPDATE bifrost_mail_state SET last_success_at = ? WHERE account = ? AND mailbox = ?`, newest, account, mailbox); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (d *daemon) bifrostMailStatus() (map[string]any, error) {
	if err := initializeBifrostSchema(d.db); err != nil {
		return nil, err
	}
	var processed, blocked, failed int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM bifrost_mail_messages WHERE status IN ('completed', 'blocked_attachment')`).Scan(&processed); err != nil {
		return nil, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM bifrost_mail_messages WHERE status = 'blocked_attachment'`).Scan(&blocked); err != nil {
		return nil, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM bifrost_mail_messages WHERE status = 'failed'`).Scan(&failed); err != nil {
		return nil, err
	}
	state, err := d.bifrostMailState(d.cfg.BifrostMailAccount, d.cfg.BifrostMailMailbox)
	if errors.Is(err, sql.ErrNoRows) {
		state = bifrostMailState{Account: d.cfg.BifrostMailAccount, Mailbox: d.cfg.BifrostMailMailbox}
		err = nil
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{"account": d.cfg.BifrostMailAccount, "mailbox": d.cfg.BifrostMailMailbox, "capability_id": d.cfg.BifrostMailCapabilityID, "initial_baseline_at": state.InitialBaselineAt, "last_success_at": state.LastSuccessAt, "processed": processed, "blocked_attachments": blocked, "failed": failed, "source_is_preserved": true, "mutations": []string{"read-only message retrieval", "normalized Intake write"}}, nil
}

func (d *daemon) handleBifrostStatus(w http.ResponseWriter, r *http.Request) {
	inspection, err := d.bifrostStatus()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, inspection)
}

func (d *daemon) handleBifrostCapabilities(w http.ResponseWriter, r *http.Request) {
	capabilities, err := d.listBifrostCapabilities()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, capabilities)
}

func (d *daemon) handleBifrostCapability(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	capability, err := d.loadBifrostCapability(id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("Bifrost capability not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, capability)
}

func (d *daemon) handleBifrostApprovals(w http.ResponseWriter, r *http.Request) {
	approvals, err := d.bifrostApprovals(100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, approvals)
}

func (d *daemon) handleBifrostInvocations(w http.ResponseWriter, r *http.Request) {
	receipts, err := d.bifrostInvocations(100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, receipts)
}

func (d *daemon) bifrostControlRequired(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get(controlTokenHeader) == d.controlToken {
		return true
	}
	writeError(w, http.StatusForbidden, errors.New("local CLI control token required for Bifrost administration"))
	return false
}

func (d *daemon) handleBifrostRegister(w http.ResponseWriter, r *http.Request) {
	if !d.bifrostControlRequired(w, r) {
		return
	}
	var request bifrostRegistrationRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	auditID := d.audit("POST /v1/bifrost/capabilities", request.Name, "bifrost.register", "", "requested", "capability metadata registration", "", 0)
	capability, err := d.registerBifrostCapability(request, auditID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, capability)
}

func (d *daemon) handleBifrostEnable(w http.ResponseWriter, r *http.Request, enabled bool) {
	if !d.bifrostControlRequired(w, r) {
		return
	}
	capability, err := d.setBifrostEnabled(r.PathValue("id"), enabled)
	if err != nil {
		var gate bifrostGateError
		if errors.As(err, &gate) {
			_, _ = d.recordBifrostInvocation(r.PathValue("id"), "local-control", "local-control", "capability.enable", bifrostOutcomeBlocked, gate.Kind, "capability state change refused", err.Error(), "", false)
			writeJSON(w, http.StatusForbidden, apiError{err.Error()})
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("Bifrost capability not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, capability)
}

func (d *daemon) handleBifrostUnsupportedLifecycle(w http.ResponseWriter, r *http.Request, operation string) {
	if !d.bifrostControlRequired(w, r) {
		return
	}
	id := r.PathValue("id")
	if _, err := d.loadBifrostCapability(id); errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("Bifrost capability not found"))
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	_, _ = d.recordBifrostInvocation(id, "local-control", "local-control", operation, bifrostOutcomeBlocked, bifrostFailureUnsupported, "Bifrost has no autonomous download or installation engine", "download and installation remain explicit external human operations", "", false)
	writeJSON(w, http.StatusNotImplemented, apiError{"Bifrost does not download or install software; register existing software only after separate human approval"})
}

func (d *daemon) handleBifrostMailStatus(w http.ResponseWriter, r *http.Request) {
	status, err := d.bifrostMailStatus()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (d *daemon) handleBifrostMailIntake(w http.ResponseWriter, r *http.Request) {
	if !d.bifrostControlRequired(w, r) {
		return
	}
	var request bifrostMailIntakeRequest
	if r.Body != nil {
		if err := decodeJSON(r, &request); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	result, err := d.bifrostMailIntake(r.Context(), request)
	if err != nil {
		status := http.StatusBadGateway
		var gate bifrostGateError
		if errors.As(err, &gate) {
			if gate.Kind == bifrostFailureApproval || gate.Kind == bifrostFailurePrometheus || gate.Kind == bifrostFailurePolicy {
				status = http.StatusServiceUnavailable
			}
		}
		writeJSON(w, status, map[string]any{"error": bifrostSafeError(err.Error()), "result": result})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (d *daemon) handleBifrostMailInvocationStatus(w http.ResponseWriter, r *http.Request) {
	status, err := d.bifrostMailStatus()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}
