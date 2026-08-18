package pantheon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestKernel22MailConfigurationDefaultsAndBoundaries(t *testing.T) {
	resolution, err := parseConfigBytes("kernel22", []byte(`{"schema_version":1,"local_model":"local","ollama_url":"http://127.0.0.1:11434/api/chat","roadmap_path":"/tmp/roadmap","models":{"apollo":{"model":"local","endpoint":"http://127.0.0.1:11434/api/chat"}},"mail":{"batch_size":100,"direction":"newest_to_oldest","mark_read_policy":"standing_authority_only","attachment_policy":"approval_required","action_capability":"gmail-mail-actions"}}`))
	if err != nil {
		t.Fatal(err)
	}
	cfg := resolution.Config
	if !cfg.MailEnabled || cfg.MailAdapter != "himalaya" || cfg.MailAccount != "gmail" || cfg.MailInbox != "INBOX" || cfg.MailBatchSize != 100 || cfg.MailActionCapabilityID != "gmail-mail-actions" {
		t.Fatalf("mail configuration=%+v", cfg)
	}
	invalid := []byte(`{"schema_version":1,"models":{"apollo":{"model":"local","endpoint":"http://127.0.0.1:11434/api/chat"}},"mail":{"adapter":"unknown"}}`)
	if _, err := parseConfigBytes("kernel22-invalid", invalid); err == nil || !strings.Contains(err.Error(), "mail.adapter") {
		t.Fatalf("invalid adapter error=%v", err)
	}
}

func TestKernel22MailRejectionRegistryRetainsProvenanceAndCanBeDisabled(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL22", "expire"))
	defer d.Close()
	if err := initializeMailSchema(d.db); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := d.db.Exec(`INSERT INTO mail_messages (source_id, account, inbox, sender, subject, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, "message-one", "gmail", "INBOX", "sender@example.com", "Routine update", mailStateUnclassified, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := d.db.Exec(`INSERT INTO mail_rejection_rules (rule_id, rule_type, match_value, created_at, behavior, state) VALUES (?, ?, ?, ?, ?, ?)`, "rule-one", "sender", "sender@example.com", now, "bypass_apollo", mailRuleEnabled); err != nil {
		t.Fatal(err)
	}
	item, err := d.loadMailMessage("message-one")
	if err != nil {
		t.Fatal(err)
	}
	rule, matched, err := d.applyMailRejectionRules(item)
	if err != nil || !matched || rule.RuleID != "rule-one" {
		t.Fatalf("rule match=%#v matched=%v err=%v", rule, matched, err)
	}
	if _, err := d.db.Exec(`UPDATE mail_messages SET state = ? WHERE source_id = ?`, mailStateDeterministicallyReject, item.SourceID); err != nil {
		t.Fatal(err)
	}
	var events int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM mail_rejection_events WHERE source_id = ?`, item.SourceID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 1 {
		t.Fatalf("rejection events=%d", events)
	}
	if err := d.disableMailRule("rule-one"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.db.Exec(`INSERT INTO mail_messages (source_id, account, inbox, sender, subject, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, "message-two", "gmail", "INBOX", "sender@example.com", "Routine update", mailStateUnclassified, now, now); err != nil {
		t.Fatal(err)
	}
	second, err := d.loadMailMessage("message-two")
	if err != nil {
		t.Fatal(err)
	}
	if _, matched, err := d.applyMailRejectionRules(second); err != nil || matched {
		t.Fatalf("disabled rule matched=%v err=%v", matched, err)
	}
}

func TestKernel22MailLedgerSyncIsIdempotentAndPreservesUnderstanding(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL22LEDGER", "expire"))
	defer d.Close()
	if err := initializeBifrostSchema(d.db); err != nil {
		t.Fatal(err)
	}
	if err := initializeMailSchema(d.db); err != nil {
		t.Fatal(err)
	}
	if _, err := d.db.Exec(`INSERT INTO bifrost_mail_messages (source_id, account, mailbox, message_id, received_at, subject, status, artifact_path, artifact_sha256, intake_occurrence_id, attachments_json, warnings, processed_at, last_error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, "message-ledger", "gmail", "INBOX", "<ledger@example>", time.Now().UTC().Format(time.RFC3339Nano), "Ledger proof", "completed", "", "", "occ-ledger", "[]", "", time.Now().UTC().Format(time.RFC3339Nano), ""); err != nil {
		t.Fatal(err)
	}
	if err := d.syncMailLedger(); err != nil {
		t.Fatal(err)
	}
	if _, err := d.db.Exec(`UPDATE mail_messages SET state = ?, summary = ? WHERE source_id = ?`, mailStateUnderstood, "already understood", "message-ledger"); err != nil {
		t.Fatal(err)
	}
	if err := d.syncMailLedger(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM mail_messages WHERE source_id = ?`, "message-ledger").Scan(&count); err != nil {
		t.Fatal(err)
	}
	item, err := d.loadMailMessage("message-ledger")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || item.State != mailStateUnderstood || item.Summary != "already understood" {
		t.Fatalf("idempotent ledger count=%d item=%#v", count, item)
	}
}

func TestKernel22MailActionsRemainSeparateFromReadCapabilityAndSend(t *testing.T) {
	readOnly := bifrostCapability{Operations: []string{"mail.read"}}
	if mailOperationAllowed(readOnly, "mail.mark_read") || mailOperationAllowed(readOnly, "mail.draft") {
		t.Fatal("read-only capability acquired write operations")
	}
	write := bifrostCapability{Operations: []string{"mail.mark_read", "mail.draft"}}
	if !mailOperationAllowed(write, "mail.mark_read") || !mailOperationAllowed(write, "mail.draft") {
		t.Fatal("declared mail operations were not recognized")
	}
	if mailOperationAllowed(write, "mail.send") || mailOperationAllowed(write, "mail.delete") {
		t.Fatal("unsupported send/delete operation was allowed")
	}
}

func TestKernel22MailPauseDoesNotCreateOrReplayAWorkRound(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL22PAUSE", "expire"))
	defer d.Close()
	d.cfg.MailEnabled = true
	if err := d.setPaused(true); err != nil {
		t.Fatal(err)
	}
	result, err := d.mailRound(context.Background(), mailRunRequest{})
	if err == nil || !result.Paused || !strings.Contains(err.Error(), "paused") {
		t.Fatalf("paused mail round result=%#v err=%v", result, err)
	}
	var runs int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM mail_runs`).Scan(&runs); err != nil {
		t.Fatal(err)
	}
	if runs != 0 {
		t.Fatalf("paused mail round created %d durable runs", runs)
	}
	if err := d.setPaused(false); err != nil {
		t.Fatal(err)
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM mail_runs`).Scan(&runs); err != nil {
		t.Fatal(err)
	}
	if runs != 0 {
		t.Fatalf("resume replayed a paused mail round: %d", runs)
	}
}

func TestKernel22MailPanopticonSurfaceIsBoundedAndReadable(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL22GUI", "expire"))
	defer d.Close()
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/gui/mail", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "state_counts") || !strings.Contains(response.Body.String(), "action_capabilities") {
		t.Fatalf("mail GUI status=%d body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["secrets"] != "omitted" || strings.Contains(response.Body.String(), d.controlToken) {
		t.Fatalf("mail GUI exposed sensitive state: %s", response.Body.String())
	}
	page := httptest.NewRecorder()
	d.Handler().ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Check mail now") || !strings.Contains(page.Body.String(), "Mail intelligence") {
		t.Fatalf("Panopticon mail surface missing: %s", page.Body.String())
	}
}
