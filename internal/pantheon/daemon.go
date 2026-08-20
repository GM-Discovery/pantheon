package pantheon

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	configPath   string
	controlToken string
	seshat       *seshatRegistry
	mu           sync.Mutex
	configMu     sync.Mutex
	rumorMu      sync.Mutex
	guiMu        sync.Mutex
	zodiacMu     sync.Mutex
	zodiacClock  func() time.Time
	guiSessions  map[string]time.Time
}

type statusResponse struct {
	LocalModel    string         `json:"local_model"`
	FrontierModel string         `json:"frontier_model,omitempty"`
	OllamaURL     string         `json:"ollama_url"`
	VaultPath     string         `json:"vault_path,omitempty"`
	RoadmapPath   string         `json:"roadmap_path,omitempty"`
	Capabilities  []capability   `json:"capabilities"`
	ListenAddress string         `json:"listen_address"`
	AuditRequests int            `json:"audit_requests"`
	Paused        bool           `json:"paused"`
	Rumor         map[string]any `json:"rumor"`
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
	Capability            string          `json:"capability"`
	Scope                 string          `json:"scope"`
	IntentSummary         string          `json:"intent_summary"`
	Diff                  string          `json:"diff,omitempty"`
	HumanApproved         bool            `json:"human_approved"`
	Holder                string          `json:"holder,omitempty"`
	GrantedBy             string          `json:"granted_by,omitempty"`
	ParentGrantID         string          `json:"parent_grant_id,omitempty"`
	ValidFrom             string          `json:"valid_from,omitempty"`
	ValidUntil            string          `json:"valid_until,omitempty"`
	RemainingUses         *int            `json:"remaining_uses,omitempty"`
	DelegationConstraints json.RawMessage `json:"delegation_constraints,omitempty"`
}

type capabilityGrant struct {
	ID                    string          `json:"id"`
	Holder                string          `json:"holder"`
	Capability            string          `json:"capability"`
	Scope                 string          `json:"scope"`
	IntentSummary         string          `json:"intent_summary"`
	Diff                  string          `json:"diff,omitempty"`
	HumanApproved         bool            `json:"human_approved"`
	GrantedBy             string          `json:"granted_by"`
	ParentGrantID         string          `json:"parent_grant_id,omitempty"`
	CreatedAt             string          `json:"created_at"`
	ValidFrom             string          `json:"valid_from,omitempty"`
	ValidUntil            string          `json:"valid_until,omitempty"`
	RemainingUses         *int            `json:"remaining_uses,omitempty"`
	Status                string          `json:"status"`
	DelegationConstraints json.RawMessage `json:"delegation_constraints,omitempty"`
	RevokedAt             string          `json:"revoked_at,omitempty"`
	UpdatedAt             string          `json:"updated_at,omitempty"`
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
	if pauseMarkerExists(pauseMarkerPathForDatabase(opts.DatabasePath)) {
		if _, err := db.Exec(`UPDATE runtime_state SET value = 'true', updated_at = ? WHERE key = 'paused'`, time.Now().Format(time.RFC3339Nano)); err != nil {
			db.Close()
			return nil, err
		}
	}
	if err := initializeAuthoritySchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := initializeIntakeSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := initializeMercurySchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := initializeJanusSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := initializeFutureSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := initializeOrganismSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := initializeRumorSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := initializeBifrostSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := initializeMailSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := initializeZodiacSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := initializeEchoSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	controlToken, err := loadOrCreateControlToken(filepath.Join(filepath.Dir(opts.DatabasePath), "control.token"))
	if err != nil {
		db.Close()
		return nil, err
	}
	seshat, err := newSeshatRegistry(filepath.Join(filepath.Dir(opts.ConfigPath), "plugins", "registered"))
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("load Seshat registry: %w", err)
	}
	d := &daemon{cfg: cfg, capabilities: caps, db: db, databasePath: opts.DatabasePath, configPath: opts.ConfigPath, controlToken: controlToken, seshat: seshat, guiSessions: map[string]time.Time{}}
	if err := d.ensureZodiacDefaults(); err != nil {
		db.Close()
		return nil, err
	}
	if err := d.ensureAphroditeDefaults(); err != nil {
		db.Close()
		return nil, err
	}
	if err := d.ensureEchoDefaults(); err != nil {
		db.Close()
		return nil, err
	}
	if err := d.ensureFameDefaults(); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
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
	mux.HandleFunc("GET /", d.handlePanopticon)
	mux.HandleFunc("GET /assets/{name}", d.handlePanopticonAsset)
	mux.HandleFunc("POST /v1/gui/session", d.handleGUISession)
	mux.HandleFunc("GET /v1/gui/work", d.handleGUIWork)
	mux.HandleFunc("GET /v1/gui/configuration", d.handleGUIConfiguration)
	mux.HandleFunc("POST /v1/gui/configuration/validate", d.handleGUIConfigurationValidate)
	mux.HandleFunc("POST /v1/gui/configuration/apply", d.handleGUIConfigurationApply)
	mux.HandleFunc("GET /v1/gui/notifications", d.handleGUINotifications)
	mux.HandleFunc("POST /v1/gui/prometheus/pause", d.handleGUIPrometheusPause)
	mux.HandleFunc("POST /v1/gui/prometheus/resume", d.handleGUIPrometheusResume)
	mux.HandleFunc("POST /v1/gui/bifrost/capabilities/{id}/enable", func(w http.ResponseWriter, r *http.Request) { d.handleGUIBifrostEnable(w, r, true) })
	mux.HandleFunc("POST /v1/gui/bifrost/capabilities/{id}/disable", func(w http.ResponseWriter, r *http.Request) { d.handleGUIBifrostEnable(w, r, false) })
	mux.HandleFunc("POST /v1/gui/rumor/scan", d.handleGUIRumorScan)
	mux.HandleFunc("POST /v1/gui/rumor/sources/{id}/retry", func(w http.ResponseWriter, r *http.Request) { d.handleGUIRumorSource(w, r, true) })
	mux.HandleFunc("POST /v1/gui/rumor/sources/{id}/ignore", func(w http.ResponseWriter, r *http.Request) { d.handleGUIRumorSource(w, r, false) })
	mux.HandleFunc("GET /v1/gui/mail", d.handleShowMail)
	mux.HandleFunc("GET /v1/gui/mail/messages", d.handleShowMailMessages)
	mux.HandleFunc("GET /v1/gui/mail/messages/{id}", d.handleShowMailMessage)
	mux.HandleFunc("GET /v1/gui/mail/rules", d.handleShowMailRules)
	mux.HandleFunc("GET /v1/gui/mail/actions", d.handleMailActionReceipts)
	mux.HandleFunc("POST /v1/gui/mail/round", d.handleGUIMailRound)
	mux.HandleFunc("POST /v1/gui/mail/rules/{id}/disable", d.handleGUIDisableMailRule)
	mux.HandleFunc("GET /v1/gui/zodiac", d.handleShowZodiac)
	mux.HandleFunc("GET /v1/gui/zodiac/today", d.handleZodiacToday)
	mux.HandleFunc("GET /v1/gui/zodiac/upcoming", d.handleZodiacUpcoming)
	mux.HandleFunc("GET /v1/gui/zodiac/needs-you", d.handleZodiacNeedsYou)
	mux.HandleFunc("POST /v1/gui/zodiac/round", d.handleZodiacRound)
	mux.HandleFunc("POST /v1/gui/zodiac/schedules", d.handleCreateZodiacSchedule)
	mux.HandleFunc("POST /v1/gui/zodiac/schedules/{id}/disable", d.handleDisableZodiacSchedule)
	mux.HandleFunc("POST /v1/gui/zodiac/attestations", d.handleZodiacAttestation)
	mux.HandleFunc("POST /v1/gui/zodiac/projections/{id}/reconcile", d.handleZodiacProjectionReconcile)
	mux.HandleFunc("GET /v1/status", d.handleStatus)
	mux.HandleFunc("GET /v1/system/inventory", d.handleSystemInventory)
	mux.HandleFunc("GET /v1/aphrodite", d.handleAphroditeSnapshot)
	mux.HandleFunc("GET /v1/gui/aphrodite", d.handleAphroditeSnapshot)
	mux.HandleFunc("GET /v1/echo", d.handleEchoSnapshot)
	mux.HandleFunc("GET /v1/gui/echo", d.handleEchoSnapshot)
	mux.HandleFunc("GET /v1/fame", d.handleFameSnapshot)
	mux.HandleFunc("GET /v1/gui/fame", d.handleFameSnapshot)
	mux.HandleFunc("GET /v1/fame/people", d.handleFamePeople)
	mux.HandleFunc("POST /v1/fame/people", d.handleFamePeople)
	mux.HandleFunc("POST /v1/fame/people/{id}/evidence", d.handleFamePersonEvidence)
	mux.HandleFunc("POST /v1/fame/conversations", d.handleFameConversation)
	mux.HandleFunc("POST /v1/fame/observations", d.handleFameObservation)
	mux.HandleFunc("POST /v1/fame/corrections", d.handleFameCorrection)
	mux.HandleFunc("POST /v1/fame/segments/{id}/clear", d.handleFameSegmentClear)
	mux.HandleFunc("POST /v1/fame/command", d.handleFameCommand)
	mux.HandleFunc("GET /v1/echo/voices", d.handleEchoVoices)
	mux.HandleFunc("POST /v1/echo/voices", d.handleEchoVoices)
	mux.HandleFunc("POST /v1/echo/voices/{id}/samples", d.handleEchoEnroll)
	mux.HandleFunc("POST /v1/echo/voices/{id}/confirm", d.handleEchoConfirm)
	mux.HandleFunc("POST /v1/echo/voices/{id}/rename", d.handleEchoVoiceRename)
	mux.HandleFunc("POST /v1/echo/segments/{id}/confirm", d.handleEchoSegmentConfirm)
	mux.HandleFunc("POST /v1/echo/segments/{id}/clear", d.handleEchoSegmentClear)
	mux.HandleFunc("POST /v1/echo/runs/import", d.handleEchoImport)
	mux.HandleFunc("POST /v1/echo/command", d.handleEchoCommand)
	mux.HandleFunc("POST /v1/aphrodite/projects", d.handleAphroditeProjectCreate)
	mux.HandleFunc("GET /v1/aphrodite/projects/{id}", d.handleAphroditeProject)
	mux.HandleFunc("POST /v1/aphrodite/projects/{id}/phase", d.handleAphroditePhase)
	mux.HandleFunc("GET /v1/aphrodite/projects/{id}/artifacts", d.handleAphroditeArtifacts)
	mux.HandleFunc("POST /v1/aphrodite/projects/{id}/artifacts", d.handleAphroditeArtifacts)
	mux.HandleFunc("GET /v1/aphrodite/projects/{id}/research-requests", d.handleAphroditeResearch)
	mux.HandleFunc("POST /v1/aphrodite/projects/{id}/research-requests", d.handleAphroditeResearch)
	mux.HandleFunc("GET /v1/aphrodite/projects/{id}/fame/fixtures", d.handleAphroditeFixtures)
	mux.HandleFunc("POST /v1/aphrodite/projects/{id}/fame/fixtures", d.handleAphroditeFixtures)
	mux.HandleFunc("POST /v1/intake/scan", d.handleIntakeScan)
	mux.HandleFunc("GET /v1/intake", d.handleShowIntake)
	mux.HandleFunc("GET /v1/father-time", d.handleShowFatherTime)
	mux.HandleFunc("POST /v1/present", d.handlePresent)
	mux.HandleFunc("GET /v1/present", d.handleShowPresent)
	mux.HandleFunc("GET /v1/processors", d.handleShowProcessors)
	mux.HandleFunc("GET /v1/rumor", d.handleShowRumor)
	mux.HandleFunc("GET /v1/rumor/queue", d.handleShowRumorQueue)
	mux.HandleFunc("GET /v1/rumor/sources/{id}", d.handleShowRumorSource)
	mux.HandleFunc("GET /v1/rumor/sources/{id}/attempts", d.handleRumorAttempts)
	mux.HandleFunc("POST /v1/rumor/scan", d.handleRumorScan)
	mux.HandleFunc("POST /v1/rumor/transcribe", d.handleRumorTranscribe)
	mux.HandleFunc("POST /v1/rumor/sources/{id}/retry", d.handleRumorRetry)
	mux.HandleFunc("POST /v1/rumor/sources/{id}/ignore", d.handleRumorIgnore)
	mux.HandleFunc("POST /v1/orders", d.handleCreateOrder)
	mux.HandleFunc("GET /v1/orders", d.handleShowOrders)
	mux.HandleFunc("GET /v1/orders/{id}", d.handleShowOrder)
	mux.HandleFunc("POST /v1/orders/{id}/cancel", d.handleCancelOrder)
	mux.HandleFunc("POST /v1/mercury/round", d.handleMercuryRound)
	mux.HandleFunc("GET /v1/mercury", d.handleShowMercury)
	mux.HandleFunc("POST /v1/janus/round", d.handleJanusRound)
	mux.HandleFunc("POST /v1/janus/subjects", d.handleCreateJanusSubject)
	mux.HandleFunc("GET /v1/janus", d.handleShowJanus)
	mux.HandleFunc("GET /v1/janus/transitions", d.handleShowJanusTransitions)
	mux.HandleFunc("GET /v1/janus/transitions/{id}", d.handleShowJanusTransition)
	mux.HandleFunc("POST /v1/future/round", d.handleFutureRound)
	mux.HandleFunc("POST /v1/future/proposals", d.handleCreateFutureProposal)
	mux.HandleFunc("GET /v1/future", d.handleShowFuture)
	mux.HandleFunc("GET /v1/future/proposals", d.handleShowFutureProposals)
	mux.HandleFunc("GET /v1/future/proposals/{id}", d.handleShowFutureProposal)
	mux.HandleFunc("POST /v1/organism/run", d.handleOrganismRun)
	mux.HandleFunc("GET /v1/organism", d.handleShowOrganism)
	mux.HandleFunc("GET /v1/apollo", d.handleShowApollo)
	mux.HandleFunc("GET /v1/apollo/reports", d.handleShowApolloReports)
	mux.HandleFunc("GET /v1/apollo/reports/{id}", d.handleShowApolloReport)
	mux.HandleFunc("POST /v1/input", d.handleInput)
	mux.HandleFunc("POST /v1/route", d.handleRoute)
	mux.HandleFunc("POST /v1/notify", d.handleNotify)
	mux.HandleFunc("POST /v1/pause", d.handlePause)
	mux.HandleFunc("POST /v1/resume", d.handleResume)
	mux.HandleFunc("POST /v1/prometheus/hard-stop", d.handlePrometheusHardStopAudit)
	mux.HandleFunc("GET /v1/grants", d.handleGrants)
	mux.HandleFunc("GET /v1/grants/{id}", d.handleGrant)
	mux.HandleFunc("POST /v1/grants", d.handleCreateGrant)
	mux.HandleFunc("POST /v1/grants/{id}/approve", d.handleApproveGrant)
	mux.HandleFunc("POST /v1/grants/{id}/revoke", d.handleRevokeGrant)
	mux.HandleFunc("POST /v1/grants/{id}/pause", d.handlePauseGrant)
	mux.HandleFunc("POST /v1/grants/{id}/resume", d.handleResumeGrant)
	mux.HandleFunc("POST /v1/prometheus/grants/{id}/approve", d.handlePrometheusApproveGrant)
	mux.HandleFunc("GET /v1/authority/check", d.handleAuthorityCheck)
	mux.HandleFunc("GET /v1/seshat/plugins", d.handleSeshatPlugins)
	mux.HandleFunc("GET /v1/seshat/packages", d.handleSeshatPackages)
	mux.HandleFunc("GET /v1/seshat/packages/{subject}", d.handleSeshatPackage)
	mux.HandleFunc("GET /v1/seshat/index", d.handleSeshatIndex)
	mux.HandleFunc("GET /v1/seshat/help", d.handleSeshatHelp)
	mux.HandleFunc("GET /v1/seshat/schema", d.handleSeshatSchema)
	mux.HandleFunc("POST /v1/seshat/plugins/validate", d.handleSeshatValidate)
	mux.HandleFunc("POST /v1/seshat/plugins/register", d.handleSeshatRegister)
	mux.HandleFunc("POST /v1/seshat/package", d.handleSeshatPackageMissive)
	mux.HandleFunc("POST /v1/email/inventory", d.handleEmailInventory)
	mux.HandleFunc("GET /v1/bifrost", d.handleBifrostStatus)
	mux.HandleFunc("GET /v1/bifrost/status", d.handleBifrostStatus)
	mux.HandleFunc("GET /v1/bifrost/capabilities", d.handleBifrostCapabilities)
	mux.HandleFunc("GET /v1/bifrost/capabilities/{id}", d.handleBifrostCapability)
	mux.HandleFunc("POST /v1/bifrost/capabilities", d.handleBifrostRegister)
	mux.HandleFunc("POST /v1/bifrost/capabilities/{id}/download", func(w http.ResponseWriter, r *http.Request) {
		d.handleBifrostUnsupportedLifecycle(w, r, "capability.download")
	})
	mux.HandleFunc("POST /v1/bifrost/capabilities/{id}/install", func(w http.ResponseWriter, r *http.Request) {
		d.handleBifrostUnsupportedLifecycle(w, r, "capability.install")
	})
	mux.HandleFunc("POST /v1/bifrost/capabilities/{id}/enable", func(w http.ResponseWriter, r *http.Request) { d.handleBifrostEnable(w, r, true) })
	mux.HandleFunc("POST /v1/bifrost/capabilities/{id}/disable", func(w http.ResponseWriter, r *http.Request) { d.handleBifrostEnable(w, r, false) })
	mux.HandleFunc("GET /v1/bifrost/approvals", d.handleBifrostApprovals)
	mux.HandleFunc("GET /v1/bifrost/invocations", d.handleBifrostInvocations)
	mux.HandleFunc("GET /v1/bifrost/mail/status", d.handleBifrostMailStatus)
	mux.HandleFunc("POST /v1/bifrost/mail/intake", d.handleBifrostMailIntake)
	mux.HandleFunc("POST /v1/mail/round", d.handleMailRound)
	mux.HandleFunc("GET /v1/mail", d.handleShowMail)
	mux.HandleFunc("GET /v1/mail/messages", d.handleShowMailMessages)
	mux.HandleFunc("GET /v1/mail/messages/{id}", d.handleShowMailMessage)
	mux.HandleFunc("GET /v1/mail/rules", d.handleShowMailRules)
	mux.HandleFunc("POST /v1/mail/rules/{id}/disable", d.handleDisableMailRule)
	mux.HandleFunc("GET /v1/zodiac", d.handleShowZodiac)
	mux.HandleFunc("GET /v1/zodiac/today", d.handleZodiacToday)
	mux.HandleFunc("GET /v1/zodiac/upcoming", d.handleZodiacUpcoming)
	mux.HandleFunc("GET /v1/zodiac/needs-you", d.handleZodiacNeedsYou)
	mux.HandleFunc("POST /v1/zodiac/round", d.handleZodiacRound)
	mux.HandleFunc("POST /v1/zodiac/schedules", d.handleCreateZodiacSchedule)
	mux.HandleFunc("GET /v1/zodiac/schedules/{id}", d.handleShowZodiacSchedule)
	mux.HandleFunc("POST /v1/zodiac/schedules/{id}/disable", d.handleDisableZodiacSchedule)
	mux.HandleFunc("POST /v1/zodiac/attestations", d.handleZodiacAttestation)
	mux.HandleFunc("POST /v1/zodiac/projections/{id}/reconcile", d.handleZodiacProjectionReconcile)
	mux.HandleFunc("GET /v1/mail/actions", d.handleMailActionReceipts)
	mux.HandleFunc("POST /v1/vault/connections", d.handleVaultConnections)
	mux.HandleFunc("POST /v1/weave/retrieve", d.handleWeaveRetrieve)
	mux.HandleFunc("POST /v1/weave/review", d.handleWeaveReview)
	mux.HandleFunc("POST /v1/weave/dry-run", d.handleWeaveDryRun)
	mux.HandleFunc("POST /v1/weave/refresh", d.handleWeaveRefresh)
	return mux
}

func (d *daemon) handleIntakeScan(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if d.blockIfPaused(w, "POST /v1/intake/scan", d.cfg.IntakePath, "intake.scan") {
		return
	}
	result, err := d.runIntakeScan()
	if err != nil {
		id := d.audit("POST /v1/intake/scan", d.cfg.IntakePath, "intake.scan", "", "failed", fmt.Sprintf("new=%d unchanged=%d departed=%d deferred=%d skipped=%d failed=%d", result.New, result.Unchanged, result.Departed, result.Deferred, result.Skipped, result.Failed), err.Error(), time.Since(started))
		result.AuditID = id
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "scan": result})
		return
	}
	result.AuditID = d.audit("POST /v1/intake/scan", d.cfg.IntakePath, "intake.scan", "", "completed", fmt.Sprintf("new=%d unchanged=%d departed=%d deferred=%d skipped=%d failed=%d chronology=%s", result.New, result.Unchanged, result.Departed, result.Deferred, result.Skipped, result.Failed, result.Chronology), "", time.Since(started))
	writeJSON(w, http.StatusOK, result)
}

func (d *daemon) handleShowIntake(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, errors.New("limit must be an integer between 1 and 100"))
			return
		}
		limit = parsed
	}
	inspection, err := d.intakeInspectionWithQuery(limit, r.URL.Query().Get("path"), r.URL.Query().Get("extension"), r.URL.Query().Get("type"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, inspection)
}

func (d *daemon) handleShowFatherTime(w http.ResponseWriter, r *http.Request) {
	inspection, err := d.fatherTimeInspection()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, inspection)
}

type weaveRetrieveRequest struct {
	Query string `json:"query"`
	Mode  string `json:"mode"`
}

type weaveCitedResult struct {
	Path       string  `json:"path"`
	Heading    string  `json:"heading"`
	Line       int     `json:"line"`
	Excerpt    string  `json:"excerpt"`
	Context    string  `json:"context"`
	Confidence float64 `json:"confidence"`
}

type weaveRetrieveResponse struct {
	Mode    string             `json:"mode"`
	Results []weaveCitedResult `json:"results"`
	AuditID string             `json:"audit_id"`
}

func (d *daemon) handleWeaveRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("local CLI control token required"))
		return
	}
	started := time.Now()
	paused, err := d.paused()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if paused {
		id := d.audit("POST /v1/weave/refresh", d.cfg.VaultPath, "vault.connect", "", "blocked", "", "Pantheon is paused; scheduled maintenance is blocked", 0)
		writeJSON(w, http.StatusServiceUnavailable, apiError{fmt.Sprintf("Pantheon is paused; refresh not run (audit %s)", id)})
		return
	}
	if _, err := d.auditRequired("POST /v1/weave/refresh", d.cfg.VaultPath, "vault.connect", "", "started", "local derived Nexus refresh authorized", "", 0); err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	stateDir := filepath.Join(filepath.Dir(d.databasePath), "weave")
	inventoryPath := filepath.Join(stateDir, "nexus-chunks.json")
	fingerprintPath := filepath.Join(stateDir, "nexus-fingerprints.json")
	current, err := buildNexusFingerprintState(d.cfg.VaultPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	previous := weaveFingerprintState{Files: map[string]string{}}
	if loaded, err := readFingerprintState(fingerprintPath); err == nil {
		previous = loaded
	} else if !os.IsNotExist(err) {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	diff := diffFingerprintStates(previous, current)
	rebuilt, err := buildNexusChunkInventory(d.cfg.VaultPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	existing := nexusChunkInventory{}
	if loaded, err := readNexusChunkInventory(inventoryPath); err == nil {
		existing = loaded
	} else if os.IsNotExist(err) {
		// Inventory and fingerprints are a consistency pair. Rebuild fully if either is absent.
		previous = weaveFingerprintState{Files: map[string]string{}}
		diff = diffFingerprintStates(previous, current)
	} else {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	inventory := applyNexusChunkDiff(existing, rebuilt, diff)
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	tmp := inventoryPath + ".tmp"
	if err := writeNexusChunkInventory(tmp, inventory); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := os.Rename(tmp, inventoryPath); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := writeWeaveFingerprintState(fingerprintPath, current); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	primaryFingerprintPath := filepath.Join(stateDir, "fingerprints.json")
	primaryDiff, err := buildWeaveChangeReport(d.cfg.VaultPath, primaryFingerprintPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	rebuiltPrimary, err := buildPrimaryWeaveIndex(d.cfg.VaultPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	primaryPath := filepath.Join(stateDir, "primary-index.json")
	existingPrimary := weaveIndex{PrimaryTermCounts: map[string]int{}}
	if loaded, err := readWeaveIndex(primaryPath); err == nil {
		existingPrimary = loaded
	} else if os.IsNotExist(err) {
		// Index and fingerprints are a consistency pair. Rebuild fully if either is absent.
		primaryDiff = fullRebuildDiff(primaryDiff.Current)
	} else {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	primary := applyPrimaryIndexDiff(existingPrimary, rebuiltPrimary, primaryDiff)
	if err := writeWeaveIndex(primaryPath+".tmp", primary); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := os.Rename(primaryPath+".tmp", primaryPath); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := writeWeaveFingerprintState(primaryFingerprintPath, primaryDiff.Current); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	reviewPath := filepath.Join(stateDir, "primary-review.md")
	if err := os.WriteFile(reviewPath+".tmp", []byte(renderWeavePrimaryReview(primary)), 0o600); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := os.Rename(reviewPath+".tmp", reviewPath); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	configPath := filepath.Join(filepath.Dir(filepath.Dir(d.databasePath)), "config", "weave-subject-reviews.json")
	if err := writeConfiguredSubjectReviews(configPath, stateDir, primary); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	id := d.audit("POST /v1/weave/refresh", d.cfg.VaultPath, "vault.connect", "", "completed", fmt.Sprintf("new=%d changed=%d removed=%d; Nexus user=%d agent=%d", len(diff.New), len(diff.Changed), len(diff.Removed), inventory.UserChunkCount, inventory.AgentChunkCount), "", time.Since(started))
	writeJSON(w, http.StatusOK, map[string]any{"audit_id": id, "new": len(diff.New), "changed": len(diff.Changed), "removed": len(diff.Removed), "nexus_user_chunks": inventory.UserChunkCount, "nexus_agent_chunks": inventory.AgentChunkCount})
}

func (d *daemon) handleWeaveDryRun(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if _, err := d.auditRequired("POST /v1/weave/dry-run", d.cfg.VaultPath, "vault.connect", "", "started", "read-only incremental weave inspection authorized", "", 0); err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	report, err := buildWeaveChangeReport(d.cfg.VaultPath, filepath.Join(filepath.Dir(d.databasePath), "weave", "fingerprints.json"))
	if err != nil {
		d.audit("POST /v1/weave/dry-run", d.cfg.VaultPath, "vault.connect", "", "failed", "", err.Error(), time.Since(started))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	id := d.audit("POST /v1/weave/dry-run", d.cfg.VaultPath, "vault.connect", "", "completed", fmt.Sprintf("new=%d changed=%d unchanged=%d excluded=%d", len(report.New), len(report.Changed), len(report.Unchanged), len(report.Excluded)), "", time.Since(started))
	writeJSON(w, http.StatusOK, struct {
		weaveChangeReport
		AuditID string `json:"audit_id"`
	}{report, id})
}

func (d *daemon) handleWeaveReview(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	var request weaveRetrieveRequest
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Query) == "" {
		if err == nil {
			err = errors.New("query is required")
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := d.auditRequired("POST /v1/weave/review", request.Query, "vault.connect", "", "started", "read-only subject review authorized", "", time.Since(started)); err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	index, err := buildPrimaryWeaveIndex(d.cfg.VaultPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	results := rankWeavePassages(request.Query, index.Passages, 0.30)
	if len(results) > 12 {
		results = results[:12]
	}
	id := d.audit("POST /v1/weave/review", request.Query, "vault.connect", "", "completed", fmt.Sprintf("%d cited passages", len(results)), "", time.Since(started))
	writeJSON(w, http.StatusOK, map[string]string{"audit_id": id, "markdown": renderWeaveQueryReview(request.Query, results)})
}

func (d *daemon) handleWeaveRetrieve(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	var request weaveRetrieveRequest
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Query) == "" {
		if err == nil {
			err = errors.New("query is required")
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	floor := 0.90
	mode := "past"
	if request.Mode == "apollo_context" {
		floor, mode = 0.30, request.Mode
	}
	if request.Mode != "" && request.Mode != mode {
		writeError(w, http.StatusBadRequest, errors.New("mode must be past or apollo_context"))
		return
	}
	if _, err := d.auditRequired("POST /v1/weave/retrieve", request.Query, "vault.connect", "", "started", "read-only weave retrieval authorized", "", time.Since(started)); err != nil {
		writeError(w, http.StatusServiceUnavailable, err)
		return
	}
	index, err := buildPrimaryWeaveIndex(d.cfg.VaultPath)
	if err != nil {
		d.audit("POST /v1/weave/retrieve", request.Query, "vault.connect", "", "failed", "", err.Error(), time.Since(started))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	ranked := rankWeavePassages(request.Query, index.Passages, floor)
	results := make([]weaveCitedResult, 0, min(12, len(ranked)))
	for _, result := range ranked {
		if strings.HasPrefix(result.Passage.Path, "Nexus/") || strings.HasPrefix(result.Passage.Path, "00 Pantheon Core/Weave/") {
			continue
		}
		context, err := expandWeaveContext(d.cfg.VaultPath, result.Passage)
		if err != nil {
			d.audit("POST /v1/weave/retrieve", request.Query, "vault.connect", "", "failed", "", err.Error(), time.Since(started))
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		results = append(results, weaveCitedResult{Path: result.Passage.Path, Heading: result.Passage.Heading, Line: result.Passage.Line, Excerpt: result.Passage.Text, Context: context, Confidence: result.Confidence})
		if len(results) == 12 {
			break
		}
	}
	id := d.audit("POST /v1/weave/retrieve", request.Query, "vault.connect", "", "completed", fmt.Sprintf("%d cited passages", len(results)), "", time.Since(started))
	writeJSON(w, http.StatusOK, weaveRetrieveResponse{Mode: mode, Results: results, AuditID: id})
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
	writeJSON(w, http.StatusOK, statusResponse{LocalModel: d.cfg.LocalModel, FrontierModel: d.cfg.FrontierModel, OllamaURL: d.cfg.OllamaURL, VaultPath: d.cfg.VaultPath, RoadmapPath: d.cfg.RoadmapPath, Capabilities: d.capabilities.Capabilities, ListenAddress: "127.0.0.1", AuditRequests: requests, Paused: paused, Rumor: d.rumorStatusSummary()})
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
	if d.blockIfPaused(w, "POST /v1/input", request.Input, "reason") {
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
	if d.blockIfPaused(w, "POST /v1/route", request.Input, "route") {
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
	grant, grantErr := d.matchingGrant("email.read", scope)
	if grantErr != nil {
		id := d.audit("POST /v1/email/inventory", scope, "email.read", "", "refused", "", "no matching active grant", time.Since(started))
		writeJSON(w, http.StatusForbidden, apiError{fmt.Sprintf("email.read requires an active matching grant (audit %s)", id)})
		return
	}
	if err := d.consumeGrant(grant.ID); err != nil {
		id := d.audit("POST /v1/email/inventory", scope, "email.read", "", "refused", "", err.Error(), time.Since(started))
		writeJSON(w, http.StatusForbidden, apiError{fmt.Sprintf("email.read grant could not be consumed (audit %s)", id)})
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

func (d *daemon) handleGrants(w http.ResponseWriter, r *http.Request) {
	if err := initializeAuthoritySchema(d.db); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	rows, err := d.db.Query(`SELECT id, holder, capability, scope, intent_summary, diff, human_approved, granted_by, parent_grant_id, created_at, valid_from, valid_until, remaining_uses, status, delegation_constraints, revoked_at, updated_at FROM capability_grants ORDER BY created_at DESC, id DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	var grants []capabilityGrant
	for rows.Next() {
		var grant capabilityGrant
		var approved, uses sql.NullInt64
		var diff, holder, grantedBy, parentID, validFrom, validUntil, constraints, revokedAt, updatedAt sql.NullString
		if err := rows.Scan(&grant.ID, &holder, &grant.Capability, &grant.Scope, &grant.IntentSummary, &diff, &approved, &grantedBy, &parentID, &grant.CreatedAt, &validFrom, &validUntil, &uses, &grant.Status, &constraints, &revokedAt, &updatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
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
		if constraints.Valid && constraints.String != "" {
			grant.DelegationConstraints = json.RawMessage(constraints.String)
		}
		grant = d.refreshGrantState(grant, time.Now())
		grants = append(grants, grant)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, grants)
}

func (d *daemon) handleGrant(w http.ResponseWriter, r *http.Request) {
	grant, err := d.loadCapabilityGrant(r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("grant not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	grant = d.refreshGrantState(grant, time.Now())
	chain, chainErr := d.authorityChain(grant.ID)
	if chainErr != nil {
		writeError(w, http.StatusInternalServerError, chainErr)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"grant": grant, "authority_chain": chain})
}

func (d *daemon) requireGrantControl(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get(controlTokenHeader) == d.controlToken {
		return true
	}
	id := d.audit("POST "+r.URL.Path, "", "grant", "", "refused", "", "local CLI control token required", 0)
	writeJSON(w, http.StatusForbidden, apiError{fmt.Sprintf("grant control requires local CLI approval (audit %s)", id)})
	return false
}

func (d *daemon) setGrantStatus(id, status string) error {
	grant, err := d.loadCapabilityGrant(id)
	if err != nil {
		return err
	}
	grant = d.refreshGrantState(grant, time.Now())
	if grant.Status == grantStatusRevoked || grant.Status == grantStatusExpired {
		return errors.New("terminal grant cannot change status")
	}
	now := time.Now().Format(time.RFC3339Nano)
	var query string
	switch status {
	case grantStatusRevoked:
		query = `UPDATE capability_grants SET status = ?, revoked_at = ?, updated_at = ? WHERE id = ?`
		_, err = d.db.Exec(query, status, now, now, id)
	case grantStatusPaused, grantStatusActive:
		query = `UPDATE capability_grants SET status = ?, updated_at = ? WHERE id = ?`
		_, err = d.db.Exec(query, status, now, id)
	default:
		return errors.New("unsupported grant status transition")
	}
	return err
}

func (d *daemon) handleApproveGrant(w http.ResponseWriter, r *http.Request) {
	if !d.requireGrantControl(w, r) {
		return
	}
	grant, err := d.loadCapabilityGrant(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, errors.New("grant not found"))
		return
	}
	if grant.Status != grantStatusPending {
		writeError(w, http.StatusConflict, errors.New("only pending grants may be approved"))
		return
	}
	if grant.ParentGrantID != "" {
		request := grantRequest{Capability: grant.Capability, Scope: grant.Scope, ParentGrantID: grant.ParentGrantID, ValidFrom: grant.ValidFrom, ValidUntil: grant.ValidUntil, RemainingUses: grant.RemainingUses}
		if _, err := d.validateDelegatedGrant(request); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
	}
	if err := d.setGrantStatus(grant.ID, grantStatusActive); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	grant, _ = d.loadCapabilityGrant(grant.ID)
	d.audit("POST "+r.URL.Path, grant.IntentSummary, grant.Capability, "", "completed", "grant approved by person: "+grant.ID, "", 0)
	writeJSON(w, http.StatusOK, grant)
}

func (d *daemon) handlePrometheusApproveGrant(w http.ResponseWriter, r *http.Request) {
	if !d.requireGrantControl(w, r) {
		return
	}
	if err := d.approveGrantAsPrometheus(r.PathValue("id")); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	grant, err := d.loadCapabilityGrant(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	d.audit("POST "+r.URL.Path, grant.IntentSummary, grant.Capability, "", "completed", "grant approved within Prometheus delegation: "+grant.ID, "", 0)
	writeJSON(w, http.StatusOK, grant)
}

func (d *daemon) handleRevokeGrant(w http.ResponseWriter, r *http.Request) {
	if !d.requireGrantControl(w, r) {
		return
	}
	if err := d.setGrantStatus(r.PathValue("id"), grantStatusRevoked); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	grant, err := d.loadCapabilityGrant(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	d.audit("POST "+r.URL.Path, grant.IntentSummary, grant.Capability, "", "completed", "grant revoked: "+grant.ID, "", 0)
	writeJSON(w, http.StatusOK, grant)
}

func (d *daemon) handlePauseGrant(w http.ResponseWriter, r *http.Request) {
	if !d.requireGrantControl(w, r) {
		return
	}
	if err := d.setGrantStatus(r.PathValue("id"), grantStatusPaused); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	grant, err := d.loadCapabilityGrant(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, grant)
}

func (d *daemon) handleResumeGrant(w http.ResponseWriter, r *http.Request) {
	if !d.requireGrantControl(w, r) {
		return
	}
	if err := d.setGrantStatus(r.PathValue("id"), grantStatusActive); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	grant, err := d.loadCapabilityGrant(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, grant)
}

func (d *daemon) handleAuthorityCheck(w http.ResponseWriter, r *http.Request) {
	capability, scope := strings.TrimSpace(r.URL.Query().Get("capability")), strings.TrimSpace(r.URL.Query().Get("scope"))
	if capability == "" || scope == "" {
		writeError(w, http.StatusBadRequest, errors.New("capability and scope query parameters are required"))
		return
	}
	decision := map[string]any{"capability": capability, "scope": scope, "capability_configured": d.hasCapability(capability), "allowed": false}
	grant, err := d.matchingGrant(capability, scope)
	if err == nil {
		chain, chainErr := d.authorityChain(grant.ID)
		if chainErr != nil {
			writeError(w, http.StatusInternalServerError, chainErr)
			return
		}
		decision["allowed"] = true
		decision["reason"] = "active grant contains requested capability and scope"
		decision["grant"] = grant
		decision["authority_chain"] = chain
	} else {
		decision["reason"] = "no active exact or containing grant; request remains unauthorized or requires escalation"
	}
	writeJSON(w, http.StatusOK, decision)
}

func (d *daemon) handleCreateGrant(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		id := d.audit("POST /v1/grants", "", "", "", "refused", "", "local CLI control token required", 0)
		writeJSON(w, http.StatusForbidden, apiError{fmt.Sprintf("grant creation requires local CLI approval (audit %s)", id)})
		return
	}
	if d.blockIfPaused(w, "POST /v1/grants", "", "grant") {
		return
	}
	if err := initializeAuthoritySchema(d.db); err != nil {
		writeError(w, http.StatusInternalServerError, err)
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
	if err := validateGrantWindow(request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	holder := strings.TrimSpace(request.Holder)
	if holder == "" {
		holder = holderPrometheus
	}
	if !deterministicHolder(holder) {
		writeError(w, http.StatusBadRequest, errors.New("deities and models cannot hold operational authority"))
		return
	}
	grantedBy := strings.TrimSpace(request.GrantedBy)
	if grantedBy == "" {
		grantedBy = holderPerson
	}
	if deityHolders[strings.ToLower(grantedBy)] {
		writeError(w, http.StatusBadRequest, errors.New("deities and models cannot grant operational authority"))
		return
	}
	status := grantStatusActive
	if request.ParentGrantID != "" && !request.HumanApproved {
		status = grantStatusPending
	}
	if request.ParentGrantID != "" {
		if _, err := d.validateDelegatedGrant(request); err != nil {
			if request.HumanApproved {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			status = grantStatusPending
		}
	}
	if _, err := d.auditRequired("POST /v1/grants", request.IntentSummary, request.Capability, "", "started", "grant creation authorized", "", 0); err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("grant creation blocked: audit unavailable: %w", err))
		return
	}
	created := time.Now().Format(time.RFC3339Nano)
	grant := capabilityGrant{ID: fmt.Sprintf("grant-%d", time.Now().UnixNano()), Holder: holder, Capability: request.Capability, Scope: request.Scope, IntentSummary: request.IntentSummary, Diff: request.Diff, HumanApproved: request.HumanApproved, GrantedBy: grantedBy, ParentGrantID: request.ParentGrantID, ValidFrom: request.ValidFrom, ValidUntil: request.ValidUntil, RemainingUses: request.RemainingUses, DelegationConstraints: request.DelegationConstraints, Status: status, CreatedAt: created, UpdatedAt: created}
	approved := 0
	if grant.HumanApproved {
		approved = 1
	}
	if _, err := d.db.Exec(`INSERT INTO capability_grants (id, holder, capability, scope, intent_summary, diff, human_approved, granted_by, parent_grant_id, created_at, valid_from, valid_until, remaining_uses, status, delegation_constraints, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, grant.ID, grant.Holder, grant.Capability, grant.Scope, grant.IntentSummary, grant.Diff, approved, grant.GrantedBy, nullableString(grant.ParentGrantID), grant.CreatedAt, nullableString(grant.ValidFrom), nullableString(grant.ValidUntil), grant.RemainingUses, grant.Status, nullableJSON(grant.DelegationConstraints), grant.UpdatedAt); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	outcome := "completed"
	if grant.Status == grantStatusPending {
		outcome = "referred"
	}
	d.audit("POST /v1/grants", grant.IntentSummary, grant.Capability, "", outcome, "grant created: "+grant.ID, "", 0)
	statusCode := http.StatusCreated
	if grant.Status == grantStatusPending {
		statusCode = http.StatusAccepted
	}
	writeJSON(w, statusCode, grant)
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func (d *daemon) hasCapability(name string) bool {
	for _, capability := range d.capabilities.Capabilities {
		if capability.Name == name {
			return true
		}
	}
	// A registered, enabled Seshat package with a locally available adapter is
	// also a known capability boundary. This makes it possible for a person to
	// grant that exact package capability without treating registration itself
	// as authority. Future still requires the resulting grant before sealing.
	if d.seshat != nil {
		for _, pkg := range d.seshat.packagesInspection() {
			if packageEnabled(pkg) && pkg.AdapterAvailable && pkg.Capability == name {
				return true
			}
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

func (d *daemon) handlePrometheusHardStopAudit(w http.ResponseWriter, r *http.Request) {
	id := d.audit("POST /v1/prometheus/hard-stop", "", "", "", "requested", "direct user-service hard-stop requested", "", 0)
	writeJSON(w, http.StatusOK, map[string]any{"audit_id": id, "result": "Direct user-service hard-stop requested."})
}

func (d *daemon) paused() (bool, error) {
	if pauseMarkerExists(pauseMarkerPathForDatabase(d.databasePath)) {
		return true, nil
	}
	var value string
	if err := d.db.QueryRow(`SELECT value FROM runtime_state WHERE key = 'paused'`).Scan(&value); err != nil {
		return false, err
	}
	return value == "true", nil
}

func (d *daemon) setPaused(value bool) error {
	markerPath := pauseMarkerPathForDatabase(d.databasePath)
	if value {
		if err := writePauseMarker(markerPath); err != nil {
			return err
		}
	}
	text := "false"
	if value {
		text = "true"
	}
	if _, err := d.db.Exec(`UPDATE runtime_state SET value = ?, updated_at = ? WHERE key = 'paused'`, text, time.Now().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if !value {
		return removePauseMarker(markerPath)
	}
	d.invalidateMercuryGos()
	return nil
}

func (d *daemon) handleShowMercury(w http.ResponseWriter, r *http.Request) {
	inspection, err := d.mercuryInspection()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, inspection)
}

func (d *daemon) blockIfPaused(w http.ResponseWriter, endpoint, input, capability string) bool {
	paused, err := d.paused()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("pause state unavailable; consequential work is blocked: %w", err))
		return true
	}
	if !paused {
		return false
	}
	id := d.audit(endpoint, input, capability, "", "blocked", "", "Pantheon is paused; consequential work is blocked", 0)
	writeJSON(w, http.StatusServiceUnavailable, apiError{fmt.Sprintf("Pantheon is paused; request not run (audit %s)", id)})
	return true
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
