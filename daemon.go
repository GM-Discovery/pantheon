package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const defaultListenAddress = "127.0.0.1:8765"
const controlTokenHeader = "X-Pantheon-Control-Token"

var localApolloHosts = map[string]bool{"127.0.0.1": true, "::1": true, "localhost": true, "172.17.0.1": true}

var notifyCommand = "/home/brick/.local/bin/pantheon-notify"
var himalayaCommand = "/home/brick/.local/bin/himalaya"

type daemonOptions struct {
	ConfigPath       string
	CapabilitiesPath string
	DatabasePath     string
}

type daemon struct {
	cfg          config
	capabilities capabilityConfig
	db           *sql.DB
	databasePath string
	controlToken string
	mu           sync.Mutex
}

type statusResponse struct {
	LocalModel    string       `json:"local_model"`
	FrontierModel string       `json:"frontier_model,omitempty"`
	OllamaURL     string       `json:"ollama_url"`
	VaultPath     string       `json:"vault_path,omitempty"`
	RoadmapPath   string       `json:"roadmap_path,omitempty"`
	Capabilities  []capability `json:"capabilities"`
	ListenAddress string       `json:"listen_address"`
	AuditRequests int          `json:"audit_requests"`
	Paused        bool         `json:"paused"`
}

type inputRequest struct {
	Input string `json:"input"`
}

type inputResponse struct {
	AuditID    string `json:"audit_id"`
	Capability string `json:"capability"`
	Model      string `json:"model"`
	Result     string `json:"result"`
}

type apiError struct {
	Error string `json:"error"`
}

type vaultConnectionRequest struct{}

type vaultConnectionResponse struct {
	ReportPath string `json:"report_path"`
	NoteCount  int    `json:"note_count"`
	AuditID    string `json:"audit_id"`
}

type emailInventoryRequest struct {
	Account  string `json:"account"`
	Mailbox  string `json:"mailbox"`
	PageSize int    `json:"page_size,omitempty"`
}

type emailInventoryResponse struct {
	Account   string          `json:"account"`
	Mailbox   string          `json:"mailbox"`
	PageSize  int             `json:"page_size"`
	Envelopes json.RawMessage `json:"envelopes"`
	AuditID   string          `json:"audit_id"`
}
type grantRequest struct {
	Capability    string `json:"capability"`
	Scope         string `json:"scope"`
	IntentSummary string `json:"intent_summary"`
	Diff          string `json:"diff,omitempty"`
	HumanApproved bool   `json:"human_approved"`
}

type capabilityGrant struct {
	ID            string `json:"id"`
	Capability    string `json:"capability"`
	Scope         string `json:"scope"`
	IntentSummary string `json:"intent_summary"`
	Diff          string `json:"diff,omitempty"`
	HumanApproved bool   `json:"human_approved"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

func defaultDaemonOptions() (daemonOptions, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return daemonOptions{}, err
	}
	root := filepath.Join(home, "pantheon-runtime")
	return daemonOptions{
		ConfigPath:       filepath.Join(root, "config", "pantheon.json"),
		CapabilitiesPath: filepath.Join(root, "config", "capabilities.json"),
		DatabasePath:     filepath.Join(root, "state", "pantheon.db"),
	}, nil
}

func loadConfigPath(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, err
	}
	if cfg.LocalModel == "" {
		cfg.LocalModel = cfg.Model
	}
	if cfg.Model == "" {
		cfg.Model = cfg.LocalModel
	}
	if cfg.LocalModel == "" || cfg.OllamaURL == "" {
		return config{}, errors.New("local_model/model and ollama_url must be configured")
	}
	endpoint, err := url.Parse(cfg.OllamaURL)
	if err != nil || endpoint.Scheme != "http" || !localApolloHosts[strings.ToLower(endpoint.Hostname())] {
		return config{}, errors.New("ollama_url must use approved local HTTP transport")
	}
	return cfg, nil
}

func loadCapabilitiesPath(path string) (capabilityConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return capabilityConfig{}, err
	}
	var caps capabilityConfig
	if err := json.Unmarshal(data, &caps); err != nil {
		return capabilityConfig{}, err
	}
	return caps, nil
}

func newDaemon(opts daemonOptions) (*daemon, error) {
	cfg, err := loadConfigPath(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	caps, err := loadCapabilitiesPath(opts.CapabilitiesPath)
	if err != nil {
		return nil, fmt.Errorf("load capabilities: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(opts.DatabasePath), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", opts.DatabasePath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS audit_events (
		id TEXT PRIMARY KEY, occurred_at TEXT NOT NULL, endpoint TEXT NOT NULL,
		input TEXT, capability TEXT, model TEXT, outcome TEXT NOT NULL,
		result TEXT, error TEXT, duration_ms INTEGER NOT NULL
	)`); err != nil {
		db.Close()
		return nil, err
	}
	if err := os.Chmod(opts.DatabasePath, 0o600); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS runtime_state (
		key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at TEXT NOT NULL
	)`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO runtime_state (key, value, updated_at) VALUES ('paused', 'false', ?)`, time.Now().Format(time.RFC3339Nano)); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS capability_grants (
		id TEXT PRIMARY KEY, capability TEXT NOT NULL, scope TEXT NOT NULL,
		intent_summary TEXT NOT NULL, diff TEXT, human_approved INTEGER NOT NULL,
		status TEXT NOT NULL, created_at TEXT NOT NULL
	)`); err != nil {
		db.Close()
		return nil, err
	}
	controlToken, err := loadOrCreateControlToken(filepath.Join(filepath.Dir(opts.DatabasePath), "control.token"))
	if err != nil {
		db.Close()
		return nil, err
	}
	return &daemon{cfg: cfg, capabilities: caps, db: db, databasePath: opts.DatabasePath, controlToken: controlToken}, nil
}

func loadOrCreateControlToken(path string) (string, error) {
	if data, err := os.ReadFile(path); err == nil {
		token := strings.TrimSpace(string(data))
		if token == "" {
			return "", errors.New("Pantheon control token is empty")
		}
		if err := os.Chmod(path, 0o600); err != nil {
			return "", err
		}
		return token, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return "", err
	}
	return token, nil
}

func (d *daemon) Close() error { return d.db.Close() }

func (d *daemon) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/status", d.handleStatus)
	mux.HandleFunc("POST /v1/input", d.handleInput)
	mux.HandleFunc("POST /v1/route", d.handleRoute)
	mux.HandleFunc("POST /v1/notify", d.handleNotify)
	mux.HandleFunc("POST /v1/pause", d.handlePause)
	mux.HandleFunc("POST /v1/resume", d.handleResume)
	mux.HandleFunc("GET /v1/grants", d.handleGrants)
	mux.HandleFunc("POST /v1/grants", d.handleCreateGrant)
	mux.HandleFunc("POST /v1/email/inventory", d.handleEmailInventory)
	mux.HandleFunc("POST /v1/vault/connections", d.handleVaultConnections)
	return mux
}

func (d *daemon) handleStatus(w http.ResponseWriter, r *http.Request) {
	count := d.audit(r.Method+" "+r.URL.Path, "", "", "", "completed", "", "", 0)
	_ = count
	var requests int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM audit_events`).Scan(&requests); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	paused, err := d.paused()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{d.cfg.LocalModel, d.cfg.FrontierModel, d.cfg.OllamaURL, d.cfg.VaultPath, d.cfg.RoadmapPath, d.capabilities.Capabilities, "127.0.0.1", requests, paused})
}

func (d *daemon) handleInput(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	var request inputRequest
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Input) == "" {
		if err == nil {
			err = errors.New("input is required")
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	decision, err := routeRequest(d.cfg, d.capabilities, request.Input)
	if err != nil {
		id := d.audit("POST /v1/input", request.Input, "", d.cfg.LocalModel, "failed", "", err.Error(), time.Since(started))
		writeJSON(w, http.StatusBadGateway, apiError{fmt.Sprintf("routing failed (audit %s): %v", id, err)})
		return
	}
	if decision.Capability != "reason" {
		id := d.audit("POST /v1/input", request.Input, decision.Capability, d.cfg.LocalModel, "refused", "", "automatic execution is not authorized", time.Since(started))
		writeJSON(w, http.StatusConflict, apiError{fmt.Sprintf("capability %q requires explicit authorization (audit %s)", decision.Capability, id)})
		return
	}
	result, err := d.reason(request.Input)
	if err != nil {
		id := d.audit("POST /v1/input", request.Input, decision.Capability, d.cfg.LocalModel, "failed", "", err.Error(), time.Since(started))
		writeJSON(w, http.StatusBadGateway, apiError{fmt.Sprintf("Apollo request failed (audit %s): %v", id, err)})
		return
	}
	id := d.audit("POST /v1/input", request.Input, decision.Capability, d.cfg.LocalModel, "completed", result, "", time.Since(started))
	writeJSON(w, http.StatusOK, inputResponse{AuditID: id, Capability: decision.Capability, Model: d.cfg.LocalModel, Result: result})
}

func (d *daemon) handleRoute(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	var request inputRequest
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Input) == "" {
		if err == nil {
			err = errors.New("input is required")
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	decision, err := routeRequest(d.cfg, d.capabilities, request.Input)
	if err != nil {
		d.audit("POST /v1/route", request.Input, "", d.cfg.LocalModel, "failed", "", err.Error(), time.Since(started))
		writeError(w, http.StatusBadGateway, err)
		return
	}
	d.audit("POST /v1/route", request.Input, decision.Capability, d.cfg.LocalModel, "completed", decision.Reason, "", time.Since(started))
	writeJSON(w, http.StatusOK, decision)
}
func (d *daemon) handleNotify(w http.ResponseWriter, r *http.Request) {
	paused, err := d.paused()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if paused {
		id := d.audit("POST /v1/notify", "", "notify", "", "blocked", "", "Pantheon is paused; external work is blocked", 0)
		writeJSON(w, http.StatusServiceUnavailable, apiError{fmt.Sprintf("Pantheon is paused; notification not delivered (audit %s)", id)})
		return
	}
	started := time.Now()
	var request inputRequest
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Input) == "" {
		if err == nil {
			err = errors.New("input is required")
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := d.auditRequired("POST /v1/notify", request.Input, "notify", "", "started", "notification delivery authorized", "", time.Since(started)); err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("notification blocked: audit unavailable: %w", err))
		return
	}
	output, err := exec.Command(notifyCommand, "pantheon-alerts", "Pantheon", request.Input).CombinedOutput()
	if err != nil {
		id := d.audit("POST /v1/notify", request.Input, "notify", "", "failed", "", fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(output))), time.Since(started))
		writeJSON(w, http.StatusBadGateway, apiError{fmt.Sprintf("notification failed (audit %s): %v", id, err)})
		return
	}
	id := d.audit("POST /v1/notify", request.Input, "notify", "", "completed", "notification sent", "", time.Since(started))
	writeJSON(w, http.StatusOK, map[string]string{"audit_id": id, "result": "Notification sent."})
}

func (d *daemon) handleVaultConnections(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	paused, err := d.paused()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if paused {
		writeError(w, http.StatusServiceUnavailable, errors.New("Pantheon is paused; Vault connection work is blocked"))
		return
	}
	if !d.hasCapability("vault.connect") {
		writeError(w, http.StatusConflict, errors.New("vault.connect is not configured"))
		return
	}
	if strings.TrimSpace(d.cfg.VaultPath) == "" {
		writeError(w, http.StatusBadRequest, errors.New("vault_path is not configured"))
		return
	}
	if _, err := d.auditRequired("POST /v1/vault/connections", d.cfg.VaultPath, "vault.connect", d.cfg.LocalModel, "started", "read-only Vault connection analysis authorized", "", time.Since(started)); err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("Vault connection analysis blocked: audit unavailable: %w", err))
		return
	}
	index, err := buildVaultConnectionIndex(d.cfg.VaultPath)
	if err != nil {
		id := d.audit("POST /v1/vault/connections", d.cfg.VaultPath, "vault.connect", d.cfg.LocalModel, "failed", "", err.Error(), time.Since(started))
		writeJSON(w, http.StatusBadGateway, apiError{fmt.Sprintf("Vault indexing failed (audit %s): %v", id, err)})
		return
	}
	evidence := connectionEvidenceReport(index)
	proposal, err := d.reasonWithContext("You are Apollo. Create a concise, explicitly tentative interpretation of the recurring terms below. Do not invent facts. Refer only to the cited evidence; do not recommend actions or modify anything. Label your output 'Apollo proposal'.\n\nVAULT CONNECTION EVIDENCE:\n" + evidence)
	if err != nil {
		id := d.audit("POST /v1/vault/connections", d.cfg.VaultPath, "vault.connect", d.cfg.LocalModel, "failed", "", err.Error(), time.Since(started))
		writeJSON(w, http.StatusBadGateway, apiError{fmt.Sprintf("Apollo connection proposal failed (audit %s): %v", id, err)})
		return
	}
	reportPath := filepath.Join(filepath.Dir(d.databasePath), "reports", fmt.Sprintf("vault-connections-%s.md", time.Now().Format("20060102-150405")))
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	report := "# Apollo Vault Connection Spike\n\n> Read-only local analysis. This report is a proposal, not a fact record or a Vault edit.\n\n" + fmt.Sprintf("- Notes scanned: %d\n- Term extraction: lowercase words of five or more characters; common connective words excluded.\n- Source evidence: paths and line numbers below.\n\n", index.NoteCount) + "# Deterministic evidence\n\n" + evidence + "# Apollo proposal\n\n" + proposal + "\n"
	if err := os.WriteFile(reportPath, []byte(report), 0o600); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	id := d.audit("POST /v1/vault/connections", d.cfg.VaultPath, "vault.connect", d.cfg.LocalModel, "completed", "read-only connection report: "+reportPath, "", time.Since(started))
	writeJSON(w, http.StatusOK, vaultConnectionResponse{ReportPath: reportPath, NoteCount: index.NoteCount, AuditID: id})
}

func (d *daemon) dbStatsPath() string {
	var path string
	_ = d.db.QueryRow(`PRAGMA database_list`).Scan(new(int), new(string), &path)
	return path
}

func (d *daemon) handleEmailInventory(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	paused, err := d.paused()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if paused {
		writeError(w, http.StatusServiceUnavailable, errors.New("Pantheon is paused; email inventory is blocked"))
		return
	}
	var request emailInventoryRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	request.Account, request.Mailbox = strings.TrimSpace(request.Account), strings.TrimSpace(request.Mailbox)
	if request.Account == "" || request.Mailbox == "" {
		writeError(w, http.StatusBadRequest, errors.New("account and mailbox are required"))
		return
	}
	const pageSize = 50
	scope := fmt.Sprintf("account=%s; mailbox=%s; page=1; page_size=50; envelopes_only", request.Account, request.Mailbox)
	if !d.hasActiveGrant("email.read", scope) {
		id := d.audit("POST /v1/email/inventory", scope, "email.read", "", "refused", "", "no matching active grant", time.Since(started))
		writeJSON(w, http.StatusForbidden, apiError{fmt.Sprintf("email.read requires an active matching grant (audit %s)", id)})
		return
	}
	if _, err := d.auditRequired("POST /v1/email/inventory", scope, "email.read", "", "started", "read-only envelope inventory authorized", "", time.Since(started)); err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("email inventory blocked: audit unavailable: %w", err))
		return
	}
	output, err := exec.Command(himalayaCommand, "envelope", "list", "--account", request.Account, "--mailbox", request.Mailbox, "--page", "1", "--page-size", "50", "--json").CombinedOutput()
	if err != nil {
		id := d.audit("POST /v1/email/inventory", scope, "email.read", "", "failed", "", strings.TrimSpace(string(output)), time.Since(started))
		writeJSON(w, http.StatusBadGateway, apiError{fmt.Sprintf("Himalaya inventory failed (audit %s): %v", id, err)})
		return
	}
	var envelopeDocument struct {
		Envelopes json.RawMessage `json:"envelopes"`
	}
	if err := json.Unmarshal(output, &envelopeDocument); err != nil || !json.Valid(envelopeDocument.Envelopes) {
		writeError(w, http.StatusBadGateway, errors.New("Himalaya returned invalid envelope JSON"))
		return
	}
	id := d.audit("POST /v1/email/inventory", scope, "email.read", "", "completed", "read-only envelope inventory", "", time.Since(started))
	writeJSON(w, http.StatusOK, emailInventoryResponse{Account: request.Account, Mailbox: request.Mailbox, PageSize: pageSize, Envelopes: envelopeDocument.Envelopes, AuditID: id})
}

func (d *daemon) hasActiveGrant(capability, scope string) bool {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM capability_grants WHERE capability = ? AND scope = ? AND status = 'active'`, capability, scope).Scan(&count)
	return err == nil && count > 0
}

func (d *daemon) handleGrants(w http.ResponseWriter, r *http.Request) {
	rows, err := d.db.Query(`SELECT id, capability, scope, intent_summary, diff, human_approved, status, created_at FROM capability_grants ORDER BY created_at DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	var grants []capabilityGrant
	for rows.Next() {
		var grant capabilityGrant
		var approved int
		if err := rows.Scan(&grant.ID, &grant.Capability, &grant.Scope, &grant.IntentSummary, &grant.Diff, &approved, &grant.Status, &grant.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		grant.HumanApproved = approved == 1
		grants = append(grants, grant)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, grants)
}

func (d *daemon) handleCreateGrant(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		id := d.audit("POST /v1/grants", "", "", "", "refused", "", "local CLI control token required", 0)
		writeJSON(w, http.StatusForbidden, apiError{fmt.Sprintf("grant creation requires local CLI approval (audit %s)", id)})
		return
	}
	var request grantRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	request.Capability, request.Scope, request.IntentSummary = strings.TrimSpace(request.Capability), strings.TrimSpace(request.Scope), strings.TrimSpace(request.IntentSummary)
	if request.Capability == "" || request.Scope == "" || request.IntentSummary == "" {
		writeError(w, http.StatusBadRequest, errors.New("capability, scope, and intent_summary are required"))
		return
	}
	if !d.hasCapability(request.Capability) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unknown configured capability %q", request.Capability))
		return
	}
	if request.Capability == "self.change" && (!request.HumanApproved || strings.TrimSpace(request.Diff) == "") {
		writeError(w, http.StatusBadRequest, errors.New("self.change requires human_approved=true and a non-empty diff"))
		return
	}
	if _, err := d.auditRequired("POST /v1/grants", request.IntentSummary, request.Capability, "", "started", "grant creation authorized", "", 0); err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("grant creation blocked: audit unavailable: %w", err))
		return
	}
	created := time.Now().Format(time.RFC3339Nano)
	grant := capabilityGrant{ID: fmt.Sprintf("grant-%d", time.Now().UnixNano()), Capability: request.Capability, Scope: request.Scope, IntentSummary: request.IntentSummary, Diff: request.Diff, HumanApproved: request.HumanApproved, Status: "active", CreatedAt: created}
	approved := 0
	if grant.HumanApproved {
		approved = 1
	}
	if _, err := d.db.Exec(`INSERT INTO capability_grants (id, capability, scope, intent_summary, diff, human_approved, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, grant.ID, grant.Capability, grant.Scope, grant.IntentSummary, grant.Diff, approved, grant.Status, grant.CreatedAt); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	d.audit("POST /v1/grants", grant.IntentSummary, grant.Capability, "", "completed", "grant created: "+grant.ID, "", 0)
	writeJSON(w, http.StatusCreated, grant)
}

func (d *daemon) hasCapability(name string) bool {
	for _, capability := range d.capabilities.Capabilities {
		if capability.Name == name {
			return true
		}
	}
	return false
}

func (d *daemon) handlePause(w http.ResponseWriter, r *http.Request) {
	if err := d.setPaused(true); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	id := d.audit("POST /v1/pause", "", "", "", "completed", "background and external work paused", "", 0)
	writeJSON(w, http.StatusOK, map[string]any{"audit_id": id, "paused": true, "result": "Pantheon paused. No queued work will replay automatically."})
}

func (d *daemon) handleResume(w http.ResponseWriter, r *http.Request) {
	if err := d.setPaused(false); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	id := d.audit("POST /v1/resume", "", "", "", "completed", "background and external work resumed", "", 0)
	writeJSON(w, http.StatusOK, map[string]any{"audit_id": id, "paused": false, "result": "Pantheon resumed. Skipped work remains skipped."})
}

func (d *daemon) paused() (bool, error) {
	var value string
	if err := d.db.QueryRow(`SELECT value FROM runtime_state WHERE key = 'paused'`).Scan(&value); err != nil {
		return false, err
	}
	return value == "true", nil
}

func (d *daemon) setPaused(value bool) error {
	text := "false"
	if value {
		text = "true"
	}
	_, err := d.db.Exec(`UPDATE runtime_state SET value = ?, updated_at = ? WHERE key = 'paused'`, text, time.Now().Format(time.RFC3339Nano))
	return err
}

var apolloHTTPClient = &http.Client{
	Timeout: 10 * time.Minute,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return errors.New("Apollo redirects are not permitted")
	},
}

func (d *daemon) reason(input string) (string, error) {
	return d.reasonWithContext(input)
}

func (d *daemon) reasonWithContext(input string) (string, error) {
	roadmap, err := os.ReadFile(d.cfg.RoadmapPath)
	if err != nil {
		return "", fmt.Errorf("read roadmap: %w", err)
	}
	system := fmt.Sprintf("You are Apollo, the local reasoning organ serving Pantheon. Pantheon authorizes; you do not. Runtime facts are authoritative: local model=%s; Ollama endpoint=%s; roadmap=%s. Use the supplied roadmap as governing context. Be concise and useful.", d.cfg.LocalModel, d.cfg.OllamaURL, d.cfg.RoadmapPath)
	payload, err := json.Marshal(ollamaRequest{Model: d.cfg.LocalModel, Messages: []message{{Role: "system", Content: system + "\n\nPANTHEON ROADMAP:\n" + string(roadmap)}, {Role: "user", Content: input}}, Stream: false})
	if err != nil {
		return "", err
	}
	resp, err := apolloHTTPClient.Post(d.cfg.OllamaURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned %s: %s", resp.Status, body)
	}
	var response ollamaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}
	return strings.TrimSpace(response.Message.Content), nil
}

func (d *daemon) auditRequired(endpoint, input, capability, model, outcome, result, errText string, duration time.Duration) (string, error) {
	return d.writeAudit(endpoint, input, capability, model, outcome, result, errText, duration)
}

func (d *daemon) audit(endpoint, input, capability, model, outcome, result, errText string, duration time.Duration) string {
	id, err := d.writeAudit(endpoint, input, capability, model, outcome, result, errText, duration)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pantheon audit write failed for %s: %v\n", endpoint, err)
		return ""
	}
	return id
}

func (d *daemon) writeAudit(endpoint, input, capability, model, outcome, result, errText string, duration time.Duration) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := fmt.Sprintf("evt-%d", time.Now().UnixNano())
	if _, err := d.db.Exec(`INSERT INTO audit_events (id, occurred_at, endpoint, input, capability, model, outcome, result, error, duration_ms) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, time.Now().Format(time.RFC3339Nano), endpoint, input, capability, model, outcome, result, errText, duration.Milliseconds()); err != nil {
		return "", err
	}
	return id, nil
}

func decodeJSON(r *http.Request, value any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(value)
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, apiError{err.Error()})
}
