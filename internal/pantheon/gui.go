package pantheon

import (
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Panopticon is intentionally embedded in the daemon. The operator does not
// need a second web server, a Node runtime, or a separate frontend process.
//
//go:embed web/*
var panopticonAssets embed.FS

const guiSessionCookie = "pantheon_gui_session"

type guiConfigField struct {
	Key             string   `json:"key"`
	Label           string   `json:"label"`
	Description     string   `json:"description"`
	Value           any      `json:"value"`
	Default         any      `json:"default,omitempty"`
	Options         []string `json:"options,omitempty"`
	Min             *int     `json:"min,omitempty"`
	Max             *int     `json:"max,omitempty"`
	Editable        bool     `json:"editable"`
	RestartRequired bool     `json:"restart_required"`
	Safety          string   `json:"safety"`
}

type guiConfigurationResponse struct {
	SchemaVersion int              `json:"schema_version"`
	Path          string           `json:"path"`
	Secrets       string           `json:"secrets"`
	Fields        []guiConfigField `json:"fields"`
}

type guiConfigurationRequest struct {
	Changes map[string]json.RawMessage `json:"changes"`
}

type guiConfigurationApplyResponse struct {
	Valid   bool                     `json:"valid"`
	Applied bool                     `json:"applied"`
	Backup  string                   `json:"backup,omitempty"`
	Changed []string                 `json:"changed,omitempty"`
	Config  guiConfigurationResponse `json:"config"`
	AuditID string                   `json:"audit_id,omitempty"`
	Restart bool                     `json:"restart_required"`
	Message string                   `json:"message,omitempty"`
}

type guiNotification struct {
	ID       string `json:"id"`
	Occurred string `json:"occurred_at"`
	Message  string `json:"message"`
	Outcome  string `json:"outcome"`
	Result   string `json:"result,omitempty"`
	Error    string `json:"error,omitempty"`
}

type guiWorkItem struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	State       string `json:"state"`
	Explanation string `json:"explanation"`
	NextAction  string `json:"next_action"`
	View        string `json:"view"`
}

type guiWorkResponse struct {
	GeneratedAt string         `json:"generated_at"`
	Counts      map[string]int `json:"counts"`
	Items       []guiWorkItem  `json:"items"`
	Note        string         `json:"note"`
}

func (d *daemon) handleGUIWork(w http.ResponseWriter, r *http.Request) {
	work, err := d.guiWorkSnapshot()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	if category != "" && category != "all" {
		filtered := make([]guiWorkItem, 0, len(work.Items))
		for _, item := range work.Items {
			if item.Category == category {
				filtered = append(filtered, item)
			}
		}
		work.Items = filtered
	}
	writeJSON(w, http.StatusOK, work)
}

func (d *daemon) guiWorkSnapshot() (guiWorkResponse, error) {
	counts := map[string]int{"active": 0, "waiting": 0, "needs_grant": 0, "needs_attention": 0, "blocked": 0, "upcoming": 0}
	items := make([]guiWorkItem, 0, 30)
	appendItem := func(item guiWorkItem) {
		if _, ok := counts[item.Category]; !ok {
			counts[item.Category] = 0
		}
		counts[item.Category]++
		if len(items) < 30 {
			items = append(items, item)
		}
	}

	futureRows, err := d.db.Query(`SELECT proposal_id, state, package_code, capability, scope, COALESCE(purpose, ''), COALESCE(last_error, '') FROM future_proposals WHERE state NOT IN (?, ?) ORDER BY updated_at DESC, proposal_id DESC LIMIT 30`, futureStateSealed, futureStateDenied)
	if err != nil {
		return guiWorkResponse{}, err
	}
	for futureRows.Next() {
		var id, state, packageCode, capability, scope, purpose, lastError string
		if err := futureRows.Scan(&id, &state, &packageCode, &capability, &scope, &purpose, &lastError); err != nil {
			futureRows.Close()
			return guiWorkResponse{}, err
		}
		category, explanation, nextAction, view := guiFutureWorkMeaning(state, lastError)
		title := strings.TrimSpace(packageCode + " · " + capability)
		if title == "·" || title == "" {
			title = "Future proposal " + id
		}
		if purpose != "" {
			title += " — " + guiBoundedText(purpose, 90)
		}
		appendItem(guiWorkItem{ID: id, Category: category, Title: guiBoundedText(title, 180), State: state, Explanation: guiBoundedText(explanation+" Scope: "+scope, 300), NextAction: nextAction, View: view})
	}
	if err := futureRows.Err(); err != nil {
		futureRows.Close()
		return guiWorkResponse{}, err
	}
	futureRows.Close()

	orderRows, err := d.db.Query(`SELECT order_id, status, COALESCE(recipient, ''), COALESCE(capability, ''), request_text, COALESCE(last_error, '') FROM orders WHERE status NOT IN (?, ?, ?, ?) ORDER BY created_at DESC, order_id DESC LIMIT 30`, orderStatusCompleted, orderStatusCancelled, orderStatusSuperseded, orderStatusExpired)
	if err != nil {
		return guiWorkResponse{}, err
	}
	for orderRows.Next() {
		var id, state, recipient, capability, requestText, lastError string
		if err := orderRows.Scan(&id, &state, &recipient, &capability, &requestText, &lastError); err != nil {
			orderRows.Close()
			return guiWorkResponse{}, err
		}
		category, explanation, nextAction, view := guiMercuryWorkMeaning(state, lastError)
		title := guiBoundedText(strings.TrimSpace(requestText), 150)
		if title == "" {
			title = strings.TrimSpace(recipient + " · " + capability)
		}
		appendItem(guiWorkItem{ID: id, Category: category, Title: title, State: state, Explanation: guiBoundedText(explanation, 300), NextAction: nextAction, View: view})
	}
	if err := orderRows.Err(); err != nil {
		orderRows.Close()
		return guiWorkResponse{}, err
	}
	orderRows.Close()

	janusRows, err := d.db.Query(`SELECT subject_id, package_code, state, COALESCE(request_text, ''), COALESCE(last_error, '') FROM janus_subjects WHERE state NOT IN (?, ?) ORDER BY updated_at DESC, subject_id DESC LIMIT 30`, janusStateExpired, janusStateResurfaced)
	if err != nil {
		return guiWorkResponse{}, err
	}
	for janusRows.Next() {
		var id, packageCode, state, requestText, lastError string
		if err := janusRows.Scan(&id, &packageCode, &state, &requestText, &lastError); err != nil {
			janusRows.Close()
			return guiWorkResponse{}, err
		}
		category, explanation, nextAction, view := guiJanusWorkMeaning(state, lastError)
		title := guiBoundedText(strings.TrimSpace(requestText), 150)
		if title == "" {
			title = "Janus · " + packageCode
		}
		appendItem(guiWorkItem{ID: id, Category: category, Title: title, State: state, Explanation: guiBoundedText(explanation+" Package: "+packageCode, 300), NextAction: nextAction, View: view})
	}
	if err := janusRows.Err(); err != nil {
		janusRows.Close()
		return guiWorkResponse{}, err
	}
	janusRows.Close()

	mailRows, err := d.db.Query(`SELECT source_id, state, COALESCE(sender, ''), COALESCE(subject, ''), COALESCE(last_error, '') FROM mail_messages WHERE state IN (?, ?, ?, ?) ORDER BY received_at DESC, source_id DESC LIMIT 30`, mailStateUnclassified, mailStateAwaitingApollo, mailStateBlocked, mailStateFailed)
	if err != nil {
		return guiWorkResponse{}, err
	}
	for mailRows.Next() {
		var id, state, sender, subject, lastError string
		if err := mailRows.Scan(&id, &state, &sender, &subject, &lastError); err != nil {
			mailRows.Close()
			return guiWorkResponse{}, err
		}
		category, explanation, nextAction := "waiting", "Mail is recorded but is waiting for its explicit understanding boundary.", "Open Mail and run Check mail now when you want this message considered."
		if state == mailStateAwaitingApollo {
			explanation, nextAction = "Present has verified this message; local Apollo processing is pending.", "Open Mail to inspect the message and its cited Apollo result."
		}
		if state == mailStateBlocked || state == mailStateFailed {
			category, explanation, nextAction = "blocked", "Mail processing retained a safety or adapter boundary: "+lastError, "Inspect Mail and the Bifrost boundary before deciding whether to retry."
		}
		title := strings.TrimSpace(subject)
		if title == "" {
			title = "Mail message " + id
		}
		if sender != "" {
			title += " · from " + sender
		}
		appendItem(guiWorkItem{ID: id, Category: category, Title: guiBoundedText(title, 180), State: state, Explanation: guiBoundedText(explanation, 300), NextAction: nextAction, View: "mail"})
	}
	if err := mailRows.Err(); err != nil {
		mailRows.Close()
		return guiWorkResponse{}, err
	}
	mailRows.Close()

	zodiacRows, err := d.db.Query(`SELECT i.instance_id, i.state, s.name, i.due_at, COALESCE(i.last_error, '') FROM zodiac_instances i JOIN zodiac_schedules s ON s.id = i.schedule_id WHERE i.state NOT IN (?, ?) ORDER BY i.due_at, i.instance_id LIMIT 30`, zodiacInstanceCompleted, zodiacInstanceSkipped)
	if err != nil {
		return guiWorkResponse{}, err
	}
	for zodiacRows.Next() {
		var id, state, name, dueAt, lastError string
		if err := zodiacRows.Scan(&id, &state, &name, &dueAt, &lastError); err != nil {
			zodiacRows.Close()
			return guiWorkResponse{}, err
		}
		category, explanation, nextAction := "upcoming", "Zodiac has a durable schedule declaration for this time.", "Open Zodiac to inspect the declaration and due-instance policy."
		switch state {
		case zodiacInstanceNeedsYou:
			category, explanation, nextAction = "needs_attention", "Zodiac will not guess through an ambiguous schedule, missing authority, or provider boundary.", "Open Zodiac Needs You and resolve the stated exception."
		case zodiacInstanceBlockedPrometheus, zodiacInstanceFailed:
			category, explanation, nextAction = "blocked", "Zodiac retained a temporal or execution boundary: "+lastError, "Open Zodiac and inspect the boundary before deciding whether to run another round."
		case zodiacInstanceWaiting:
			category, explanation, nextAction = "waiting", "This schedule is waiting on its declared dependency.", "Open Zodiac to inspect the dependency state."
		case zodiacInstanceAwaitingAuthority:
			category, explanation, nextAction = "needs_grant", "This due schedule is waiting for Future authority; due state is not permission.", "Open Future/Approvals and review the exact authority request."
		case zodiacInstanceDispatched:
			category, explanation, nextAction = "active", "Mercury has a bounded order associated with this due instance.", "Inspect the exact order and its result."
		}
		appendItem(guiWorkItem{ID: id, Category: category, Title: guiBoundedText(name+" · due "+dueAt, 180), State: state, Explanation: guiBoundedText(explanation, 300), NextAction: nextAction, View: "zodiac"})
	}
	if err := zodiacRows.Err(); err != nil {
		zodiacRows.Close()
		return guiWorkResponse{}, err
	}
	zodiacRows.Close()

	return guiWorkResponse{GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Counts: counts, Items: items, Note: "These are bounded live work records. A count is only shown when Pantheon can identify the underlying proposal, order, or temporal subject."}, nil
}

func guiFutureWorkMeaning(state, lastError string) (string, string, string, string) {
	switch state {
	case futureStateNeedsAuthority, futureStateReferredPrometheus:
		return "needs_grant", "Future cannot seal this proposal because the required authority is not present.", "Review the authority request; a person must approve, amend, or decline it.", "approvals"
	case futureStateTemporallyBlocked:
		return "waiting", "Janus is holding this proposal behind a declared temporal condition.", "Review the temporal subject and threshold before asking Future to reconsider.", "ingestion"
	case futureStateAdapterUnavailable, futureStateMalformed:
		if lastError != "" {
			return "blocked", "Future cannot safely prepare this proposal: " + lastError, "Inspect the package or adapter boundary; do not retry blindly.", "execution"
		}
		return "blocked", "Future cannot safely prepare this proposal because its package or adapter is unavailable.", "Inspect the package or adapter boundary; do not retry blindly.", "execution"
	case futureStateAwaiting:
		return "upcoming", "This proposal is recorded and awaiting Future's next deterministic authority evaluation.", "Review the proposal if it is expected to move now; otherwise leave it pending.", "approvals"
	default:
		return "active", "Future is evaluating or holding this proposal in its current state.", "Review the proposal before requesting another round.", "approvals"
	}
}

func guiMercuryWorkMeaning(state, lastError string) (string, string, string, string) {
	switch state {
	case orderStatusWaiting:
		return "waiting", "Mercury is waiting because Prometheus pause prevents Go issuance.", "Resume only when ready, then run the intended Mercury round explicitly; resume does not replay.", "execution"
	case orderStatusUnavailable:
		return "blocked", "Mercury has no safe local adapter for this order.", "Inspect the recipient/capability boundary; do not broaden the order to make it run.", "execution"
	case orderStatusRuling, orderStatusInterpret:
		return "needs_attention", "Mercury needs a bounded ruling or interpretation before it can issue Go.", "Review the order's exact request and resolve the stated human decision.", "approvals"
	default:
		if lastError != "" {
			return "blocked", "Mercury has retained an error for this order: " + lastError, "Inspect the order and its last error before retrying.", "execution"
		}
		return "active", "Mercury has retained this order for bounded processing.", "Inspect the sealed fields and current execution state.", "execution"
	}
}

func guiJanusWorkMeaning(state, lastError string) (string, string, string, string) {
	switch state {
	case janusStateBlocked:
		if lastError != "" {
			return "blocked", "Janus recorded a temporal evaluation error: " + lastError, "Inspect the temporal subject and correct the declared boundary before rerunning.", "ingestion"
		}
		return "blocked", "Janus recorded a temporal evaluation error.", "Inspect the temporal subject before rerunning.", "ingestion"
	case janusStateScheduled, janusStateReviewable:
		return "upcoming", "Janus is holding a declared threshold for future evaluation.", "Review the threshold and request context if the timing is no longer right.", "ingestion"
	case janusStateReady, janusStateActive:
		return "active", "Janus says this temporal subject is eligible or active.", "Inspect the linked order or proposal before proceeding.", "execution"
	default:
		return "needs_attention", "Janus retained this temporal subject for human inspection.", "Review the transition history and declared threshold.", "ingestion"
	}
}

func (d *daemon) handlePanopticon(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	d.servePanopticonAsset(w, "web/index.html", "text/html; charset=utf-8")
}

func (d *daemon) handlePanopticonAsset(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.PathValue("name"))
	asset := "web/" + name
	contentType := "application/octet-stream"
	switch filepath.Ext(name) {
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".js":
		contentType = "text/javascript; charset=utf-8"
	case ".html":
		contentType = "text/html; charset=utf-8"
	}
	d.servePanopticonAsset(w, asset, contentType)
}

func (d *daemon) servePanopticonAsset(w http.ResponseWriter, asset, contentType string) {
	data, err := panopticonAssets.ReadFile(asset)
	if err != nil {
		http.Error(w, "Panopticon asset not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	if asset == "web/index.html" {
		w.Header().Set("Cache-Control", "no-store")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=300")
	}
	_, _ = w.Write(data)
}

func guiOriginAllowed(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme != "http" {
		return false
	}
	return parsed.Host == r.Host && (parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "localhost" || parsed.Hostname() == "::1")
}

func (d *daemon) handleGUISession(w http.ResponseWriter, r *http.Request) {
	if !guiOriginAllowed(r) {
		writeError(w, http.StatusForbidden, errors.New("Panopticon session requires the local Pantheon origin"))
		return
	}
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	session := hex.EncodeToString(bytes)
	d.guiMu.Lock()
	if d.guiSessions == nil {
		d.guiSessions = map[string]time.Time{}
	}
	d.guiSessions[session] = time.Now().Add(8 * time.Hour)
	for token, expires := range d.guiSessions {
		if time.Now().After(expires) {
			delete(d.guiSessions, token)
		}
	}
	d.guiMu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: guiSessionCookie, Value: session, Path: "/", MaxAge: 8 * 60 * 60, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	writeJSON(w, http.StatusOK, map[string]any{"ready": true, "control": "local Panopticon session", "expires_in_seconds": 8 * 60 * 60})
}

func (d *daemon) guiControlRequired(w http.ResponseWriter, r *http.Request) bool {
	if !guiOriginAllowed(r) {
		writeError(w, http.StatusForbidden, errors.New("Panopticon mutation requires the local Pantheon origin"))
		return false
	}
	if r.Header.Get(controlTokenHeader) == d.controlToken {
		return true
	}
	cookie, err := r.Cookie(guiSessionCookie)
	if err == nil {
		d.guiMu.Lock()
		expires, ok := d.guiSessions[cookie.Value]
		if ok && time.Now().Before(expires) {
			d.guiMu.Unlock()
			return true
		}
		if ok {
			delete(d.guiSessions, cookie.Value)
		}
		d.guiMu.Unlock()
	}
	writeError(w, http.StatusForbidden, errors.New("Panopticon control requires a local GUI session"))
	return false
}

func (d *daemon) handleGUIConfiguration(w http.ResponseWriter, r *http.Request) {
	d.configMu.Lock()
	response := guiConfigurationFor(d.cfg, d.configPath)
	d.configMu.Unlock()
	writeJSON(w, http.StatusOK, response)
}

func guiConfigurationFor(cfg config, path string) guiConfigurationResponse {
	minZero, minOne, maxOneDay, maxRetention := 0, 1, 86400, 36500
	maxMailBatch := 100
	return guiConfigurationResponse{
		SchemaVersion: cfg.SchemaVersion,
		Path:          path,
		Secrets:       "omitted",
		Fields: []guiConfigField{
			{Key: "zodiac.enabled", Label: "Zodiac schedule keeper", Description: "Run the persisted deterministic schedule keeper. Zodiac evaluates explicit time and recurrence only; it does not reason or create authority.", Value: cfg.ZodiacEnabled, Default: true, Editable: true, RestartRequired: true, Safety: "significant"},
			{Key: "zodiac.default_timezone", Label: "Zodiac timezone", Description: "Canonical timezone for human schedules and recurrence evaluation.", Value: cfg.ZodiacDefaultTimezone, Default: "America/Los_Angeles", Editable: true, RestartRequired: false, Safety: "significant"},
			{Key: "zodiac.personal_projection", Label: "Personal calendar projection", Description: "Provider-neutral policy for personal/family obligations. Presence is not permission.", Value: cfg.ZodiacPersonalProjection, Default: "apple", Options: []string{"apple", "google", "both", "none"}, Editable: true, RestartRequired: false, Safety: "restricted"},
			{Key: "zodiac.work_projection", Label: "Work calendar projection", Description: "Provider-neutral policy for work obligations. Actual writes require an approved Bifrost adapter.", Value: cfg.ZodiacWorkProjection, Default: "google", Options: []string{"apple", "google", "both", "none"}, Editable: true, RestartRequired: false, Safety: "restricted"},
			{Key: "zodiac.work_hours_start", Label: "Work hours start", Description: "Configurable local boundary used to surface after-hours work impact.", Value: cfg.ZodiacWorkHoursStart, Default: "09:00", Editable: true, RestartRequired: false, Safety: "routine"},
			{Key: "zodiac.work_hours_end", Label: "Work hours end", Description: "Configurable local boundary used to surface after-hours work impact.", Value: cfg.ZodiacWorkHoursEnd, Default: "17:00", Editable: true, RestartRequired: false, Safety: "routine"},
			{Key: "zodiac.brief_time", Label: "Situation Brief time", Description: "The local daily time at which Zodiac creates the Situation Brief due instance; Zodiac does not compose it.", Value: cfg.ZodiacBriefTime, Default: "07:00", Editable: true, RestartRequired: false, Safety: "significant"},
			{Key: "zodiac.brief_catch_up_policy", Label: "Brief catch-up policy", Description: "Explicit recovery policy for a missed daily brief.", Value: cfg.ZodiacBriefCatchUpPolicy, Default: "run_once", Options: []string{"skip", "run_once", "reconcile", "needs_you"}, Editable: true, RestartRequired: false, Safety: "significant"},
			{Key: "zodiac.wake_seconds", Label: "Zodiac wake interval", Description: "How often Zodiac checks persisted declarations; it never wakes every millisecond.", Value: cfg.ZodiacWakeSeconds, Default: 30, Min: &minOne, Max: &maxOneDay, Editable: true, RestartRequired: true, Safety: "routine"},
			{Key: "rumor.enabled", Label: "Rumor audio intake", Description: "Allow the local audio boundary to watch configured folders and process due recordings.", Value: cfg.RumorEnabled, Default: false, Editable: true, RestartRequired: true, Safety: "routine"},
			{Key: "rumor.poll_seconds", Label: "Rumor check interval", Description: "How often Rumor checks configured audio locations.", Value: cfg.RumorPollSeconds, Default: 5, Min: &minOne, Max: &maxOneDay, Editable: true, RestartRequired: true, Safety: "routine"},
			{Key: "rumor.stability_seconds", Label: "Recording stability wait", Description: "How long a file must remain unchanged before Rumor processes it.", Value: cfg.RumorStabilitySeconds, Default: 2, Min: &minZero, Max: &maxOneDay, Editable: true, RestartRequired: false, Safety: "routine"},
			{Key: "rumor.language", Label: "Whisper language mode", Description: "Language hint for the local Whisper processor.", Value: cfg.RumorLanguage, Default: "auto", Editable: true, RestartRequired: true, Safety: "routine"},
			{Key: "apollo.require_citations", Label: "Require Apollo citations", Description: "Keep material Apollo claims tied to cited evidence.", Value: cfg.ApolloRequireCitations, Default: true, Editable: true, RestartRequired: false, Safety: "significant"},
			{Key: "mercury.deterministic_fallback", Label: "Local execution fallback", Description: "Allow exact local order execution when no optional external Mercury helper is available.", Value: cfg.MercuryDeterministicFallback, Default: true, Editable: true, RestartRequired: false, Safety: "significant"},
			{Key: "bifrost.enabled", Label: "Bifrost capability bridge", Description: "Enable governed external-capability surfaces; individual capabilities still require their own approvals.", Value: cfg.BifrostEnabled, Default: true, Editable: true, RestartRequired: true, Safety: "restricted"},
			{Key: "mail.enabled", Label: "Mail intelligence pipeline", Description: "Enable explicit local Gmail intake, bounded Apollo understanding, and governed proposals. No send or delete operation is implemented.", Value: cfg.MailEnabled, Default: true, Editable: true, RestartRequired: false, Safety: "significant"},
			{Key: "mail.backlog_enabled", Label: "Mail backlog review", Description: "Allow manual Check mail now rounds to advance the durable newest-to-oldest backlog cursor.", Value: cfg.MailBacklogEnabled, Default: true, Editable: true, RestartRequired: false, Safety: "routine"},
			{Key: "mail.batch_size", Label: "Mail backlog batch size", Description: "Maximum older messages considered per explicit round; bounded to 100.", Value: cfg.MailBatchSize, Default: 100, Min: &minOne, Max: &maxMailBatch, Editable: true, RestartRequired: false, Safety: "routine"},
			{Key: "mail.mark_read_policy", Label: "Mail mark-read policy", Description: "Mark-read remains governed and is never implied by understanding alone.", Value: cfg.MailMarkReadPolicy, Default: "standing_authority_only", Options: []string{"standing_authority_only", "never"}, Editable: true, RestartRequired: false, Safety: "restricted"},
			{Key: "mail.adapter", Label: "Mail adapter", Description: "Existing local Himalaya/Gmail boundary. Adapter presence does not grant permission.", Value: cfg.MailAdapter, Editable: false, RestartRequired: false, Safety: "informational"},
			{Key: "mail.account", Label: "Mail account", Description: "Configured personal account name; credentials are omitted.", Value: cfg.MailAccount, Editable: false, RestartRequired: false, Safety: "informational"},
			{Key: "mail.inbox", Label: "Mail inbox", Description: "Configured source mailbox for bounded intake.", Value: cfg.MailInbox, Editable: false, RestartRequired: false, Safety: "informational"},
			{Key: "mail.action_capability", Label: "Mail action capability", Description: "Separate Bifrost capability identity for governed mark-read, label, move, and draft operations; registration and use approval remain distinct.", Value: cfg.MailActionCapabilityID, Editable: false, RestartRequired: false, Safety: "restricted"},
			{Key: "retention.raw_input_days", Label: "Raw input retention policy", Description: "Policy duration after verified Markdown; this setting does not silently delete source material.", Value: cfg.RawInputDays, Default: 30, Min: &minOne, Max: &maxRetention, Editable: true, RestartRequired: false, Safety: "significant"},
			{Key: "paths.vault", Label: "Vault location", Description: "Configured local continuity root used by Past and Weave.", Value: cfg.VaultPath, Editable: false, RestartRequired: true, Safety: "informational"},
			{Key: "paths.intake", Label: "Intake location", Description: "Folder Pantheon identity-tracks for incoming occurrences.", Value: cfg.IntakePath, Editable: false, RestartRequired: true, Safety: "informational"},
			{Key: "models.apollo.model", Label: "Apollo local model", Description: "Model used for local cited reasoning.", Value: cfg.LocalModel, Editable: false, RestartRequired: true, Safety: "informational"},
			{Key: "models.archimedes.model", Label: "Archimedes frontier model", Description: "Reserved future frontier path; currently inactive.", Value: cfg.FrontierModel, Editable: false, RestartRequired: false, Safety: "deferred"},
		},
	}
}

func (d *daemon) guiConfigurationCandidate(changes map[string]json.RawMessage) (configResolution, []string, []byte, error) {
	if len(changes) == 0 {
		return configResolution{}, nil, nil, errors.New("changes are required")
	}
	if strings.TrimSpace(d.configPath) == "" {
		return configResolution{}, nil, nil, errors.New("configuration path is unavailable")
	}
	data, err := os.ReadFile(d.configPath)
	if err != nil {
		return configResolution{}, nil, nil, err
	}
	var canonical canonicalConfig
	if err := json.Unmarshal(data, &canonical); err != nil || canonical.SchemaVersion != currentConfigSchemaVersion {
		return configResolution{}, nil, nil, errors.New("GUI configuration apply requires the current schema")
	}
	changed := make([]string, 0, len(changes))
	for key, raw := range changes {
		if err := applyGUIConfigurationChange(&canonical, key, raw); err != nil {
			return configResolution{}, nil, nil, err
		}
		changed = append(changed, key)
	}
	sort.Strings(changed)
	proposed, err := json.MarshalIndent(canonical, "", "  ")
	if err != nil {
		return configResolution{}, nil, nil, err
	}
	proposed = append(proposed, '\n')
	resolution, err := parseConfigBytes(d.configPath, proposed)
	if err != nil {
		return configResolution{}, nil, nil, err
	}
	return resolution, changed, proposed, nil
}

func applyGUIConfigurationChange(canonical *canonicalConfig, key string, raw json.RawMessage) error {
	decodeBool := func() (bool, error) {
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return false, fmt.Errorf("%s must be boolean", key)
		}
		return value, nil
	}
	decodeInt := func() (int, error) {
		var value int
		if err := json.Unmarshal(raw, &value); err != nil {
			return 0, fmt.Errorf("%s must be an integer", key)
		}
		return value, nil
	}
	decodeString := func() (string, error) {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", fmt.Errorf("%s must be a string", key)
		}
		return strings.TrimSpace(value), nil
	}
	switch key {
	case "zodiac.enabled":
		value, err := decodeBool()
		if err != nil {
			return err
		}
		if canonical.Zodiac == nil {
			canonical.Zodiac = &canonicalZodiac{}
		}
		canonical.Zodiac.Enabled = &value
	case "zodiac.default_timezone":
		value, err := decodeString()
		if err != nil {
			return err
		}
		if canonical.Zodiac == nil {
			canonical.Zodiac = &canonicalZodiac{}
		}
		canonical.Zodiac.DefaultTimezone = &value
	case "zodiac.personal_projection":
		value, err := decodeString()
		if err != nil {
			return err
		}
		if canonical.Zodiac == nil {
			canonical.Zodiac = &canonicalZodiac{}
		}
		canonical.Zodiac.PersonalProjection = &value
	case "zodiac.work_projection":
		value, err := decodeString()
		if err != nil {
			return err
		}
		if canonical.Zodiac == nil {
			canonical.Zodiac = &canonicalZodiac{}
		}
		canonical.Zodiac.WorkProjection = &value
	case "zodiac.work_hours_start":
		value, err := decodeString()
		if err != nil {
			return err
		}
		if canonical.Zodiac == nil {
			canonical.Zodiac = &canonicalZodiac{}
		}
		canonical.Zodiac.WorkHoursStart = &value
	case "zodiac.work_hours_end":
		value, err := decodeString()
		if err != nil {
			return err
		}
		if canonical.Zodiac == nil {
			canonical.Zodiac = &canonicalZodiac{}
		}
		canonical.Zodiac.WorkHoursEnd = &value
	case "zodiac.brief_time":
		value, err := decodeString()
		if err != nil {
			return err
		}
		if canonical.Zodiac == nil {
			canonical.Zodiac = &canonicalZodiac{}
		}
		canonical.Zodiac.BriefTime = &value
	case "zodiac.brief_catch_up_policy":
		value, err := decodeString()
		if err != nil {
			return err
		}
		if canonical.Zodiac == nil {
			canonical.Zodiac = &canonicalZodiac{}
		}
		canonical.Zodiac.BriefCatchUpPolicy = &value
	case "zodiac.wake_seconds":
		value, err := decodeInt()
		if err != nil {
			return err
		}
		if canonical.Zodiac == nil {
			canonical.Zodiac = &canonicalZodiac{}
		}
		canonical.Zodiac.WakeSeconds = &value
	case "rumor.enabled":
		value, err := decodeBool()
		if err != nil {
			return err
		}
		if canonical.Rumor == nil {
			canonical.Rumor = &canonicalRumor{}
		}
		canonical.Rumor.Enabled = &value
	case "rumor.poll_seconds":
		value, err := decodeInt()
		if err != nil {
			return err
		}
		if canonical.Rumor == nil {
			canonical.Rumor = &canonicalRumor{}
		}
		canonical.Rumor.PollSeconds = &value
	case "rumor.stability_seconds":
		value, err := decodeInt()
		if err != nil {
			return err
		}
		if canonical.Rumor == nil {
			canonical.Rumor = &canonicalRumor{}
		}
		canonical.Rumor.StabilitySeconds = &value
	case "rumor.language":
		value, err := decodeString()
		if err != nil {
			return err
		}
		if canonical.Rumor == nil {
			canonical.Rumor = &canonicalRumor{}
		}
		canonical.Rumor.Language = &value
	case "apollo.require_citations":
		value, err := decodeBool()
		if err != nil {
			return err
		}
		if canonical.Apollo == nil {
			canonical.Apollo = &canonicalApolloSettings{}
		}
		canonical.Apollo.RequireCitations = &value
	case "mercury.deterministic_fallback":
		value, err := decodeBool()
		if err != nil {
			return err
		}
		if canonical.Mercury == nil {
			canonical.Mercury = &canonicalMercury{}
		}
		canonical.Mercury.DeterministicFallback = &value
	case "bifrost.enabled":
		value, err := decodeBool()
		if err != nil {
			return err
		}
		if canonical.Bifrost == nil {
			canonical.Bifrost = &canonicalBifrost{}
		}
		canonical.Bifrost.Enabled = &value
	case "mail.enabled":
		value, err := decodeBool()
		if err != nil {
			return err
		}
		if canonical.Mail == nil {
			canonical.Mail = &canonicalMail{}
		}
		canonical.Mail.Enabled = &value
	case "mail.backlog_enabled":
		value, err := decodeBool()
		if err != nil {
			return err
		}
		if canonical.Mail == nil {
			canonical.Mail = &canonicalMail{}
		}
		canonical.Mail.BacklogEnabled = &value
	case "mail.batch_size":
		value, err := decodeInt()
		if err != nil {
			return err
		}
		if canonical.Mail == nil {
			canonical.Mail = &canonicalMail{}
		}
		canonical.Mail.BatchSize = &value
	case "mail.mark_read_policy":
		value, err := decodeString()
		if err != nil {
			return err
		}
		if canonical.Mail == nil {
			canonical.Mail = &canonicalMail{}
		}
		canonical.Mail.MarkReadPolicy = &value
	case "retention.raw_input_days":
		value, err := decodeInt()
		if err != nil {
			return err
		}
		if canonical.Retention == nil {
			canonical.Retention = &canonicalRetention{}
		}
		canonical.Retention.RawInputDays = &value
	default:
		return fmt.Errorf("%s is not GUI-editable", key)
	}
	return nil
}

func (d *daemon) handleGUIConfigurationValidate(w http.ResponseWriter, r *http.Request) {
	var request guiConfigurationRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	d.configMu.Lock()
	resolution, changed, _, err := d.guiConfigurationCandidate(request.Changes)
	if err == nil {
		write := guiConfigurationApplyResponse{Valid: true, Applied: false, Changed: changed, Config: guiConfigurationFor(resolution.Config, d.configPath), Restart: guiChangesRequireRestart(changed), Message: "Configuration is valid and has not been written."}
		d.configMu.Unlock()
		writeJSON(w, http.StatusOK, write)
		return
	}
	d.configMu.Unlock()
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"valid": false, "error": err.Error()})
}

func (d *daemon) handleGUIConfigurationApply(w http.ResponseWriter, r *http.Request) {
	if !d.guiControlRequired(w, r) {
		return
	}
	var request guiConfigurationRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	d.configMu.Lock()
	resolution, changed, proposed, err := d.guiConfigurationCandidate(request.Changes)
	if err != nil {
		d.configMu.Unlock()
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"valid": false, "error": err.Error()})
		return
	}
	backup := ""
	if err == nil {
		backup, err = backupConfig(d.configPath)
	}
	if err == nil {
		err = writeValidatedConfig(d.configPath, proposed)
	}
	if err == nil {
		d.cfg = resolution.Config
	}
	d.configMu.Unlock()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"valid": true, "applied": false, "error": err.Error()})
		return
	}
	auditID := d.audit("POST /v1/gui/configuration/apply", strings.Join(changed, ","), "configuration.apply", "", "completed", "validated GUI configuration applied", "", 0)
	writeJSON(w, http.StatusOK, guiConfigurationApplyResponse{Valid: true, Applied: true, Backup: backup, Changed: changed, Config: guiConfigurationFor(resolution.Config, d.configPath), Restart: guiChangesRequireRestart(changed), AuditID: auditID, Message: "Configuration applied. Restart Pantheon where indicated."})
}

func guiChangesRequireRestart(changed []string) bool {
	for _, key := range changed {
		if key == "rumor.enabled" || key == "rumor.poll_seconds" || key == "rumor.language" || key == "bifrost.enabled" || key == "zodiac.enabled" || key == "zodiac.wake_seconds" {
			return true
		}
	}
	return false
}

func (d *daemon) handleGUIPrometheusPause(w http.ResponseWriter, r *http.Request) {
	if !d.guiControlRequired(w, r) {
		return
	}
	if err := d.setPaused(true); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	id := d.audit("POST /v1/gui/prometheus/pause", "", "prometheus.pause", "", "completed", "Pantheon paused from Panopticon", "", 0)
	writeJSON(w, http.StatusOK, map[string]any{"paused": true, "audit_id": id, "result": "Pantheon paused. Skipped work will not replay automatically."})
}

func (d *daemon) handleGUIPrometheusResume(w http.ResponseWriter, r *http.Request) {
	if !d.guiControlRequired(w, r) {
		return
	}
	if err := d.setPaused(false); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	id := d.audit("POST /v1/gui/prometheus/resume", "", "prometheus.resume", "", "completed", "Pantheon resumed from Panopticon", "", 0)
	writeJSON(w, http.StatusOK, map[string]any{"paused": false, "audit_id": id, "result": "Pantheon resumed. Skipped work remains skipped."})
}

func (d *daemon) handleGUIBifrostEnable(w http.ResponseWriter, r *http.Request, enabled bool) {
	if !d.guiControlRequired(w, r) {
		return
	}
	capability, err := d.setBifrostEnabled(r.PathValue("id"), enabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusForbidden, map[string]any{"error": err.Error(), "enabled": false})
		return
	}
	writeJSON(w, http.StatusOK, capability)
}

func (d *daemon) handleGUIRumorScan(w http.ResponseWriter, r *http.Request) {
	if !d.guiControlRequired(w, r) {
		return
	}
	result, err := d.rumorProcessCycle(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error(), "result": result})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (d *daemon) handleGUIRumorSource(w http.ResponseWriter, r *http.Request, retry bool) {
	if !d.guiControlRequired(w, r) {
		return
	}
	id := r.PathValue("id")
	var source rumorSource
	var err error
	if retry {
		source, err = d.retryRumorSource(id)
	} else {
		source, err = d.ignoreRumorSource(id)
	}
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (d *daemon) handleGUINotifications(w http.ResponseWriter, r *http.Request) {
	limit := 30
	if value, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && value > 0 && value <= 100 {
		limit = value
	}
	rows, err := d.db.Query(`SELECT id, occurred_at, COALESCE(input, ''), outcome, COALESCE(result, ''), COALESCE(error, '') FROM audit_events WHERE endpoint = 'POST /v1/notify' OR endpoint LIKE 'POST /v1/gui/notifications%' ORDER BY occurred_at DESC LIMIT ?`, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	notifications := make([]guiNotification, 0)
	for rows.Next() {
		var item guiNotification
		if err := rows.Scan(&item.ID, &item.Occurred, &item.Message, &item.Outcome, &item.Result, &item.Error); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		item.Message = guiBoundedText(item.Message, 2000)
		item.Result = guiBoundedText(item.Result, 1000)
		item.Error = guiBoundedText(item.Error, 1000)
		notifications = append(notifications, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notifications": notifications, "source": "Pantheon notification audit", "raw_audit": false})
}

func guiBoundedText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}
