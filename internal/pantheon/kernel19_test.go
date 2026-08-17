package pantheon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func kernel19Capability(t *testing.T, d *daemon, approved bool) bifrostCapability {
	t.Helper()
	d.cfg.BifrostEnabled = true
	d.cfg.BifrostMailAccount = "gmail"
	d.cfg.BifrostMailMailbox = "INBOX"
	d.cfg.BifrostMailCapabilityID = "gmail-mail"
	request := bifrostRegistrationRequest{
		CapabilityID: "gmail-mail", Name: "Gmail read-only mail", Type: bifrostCapabilityTypeCommand,
		Source: "himalaya", Version: "local-configured", Description: "Read-only bounded mail intake.",
		Invocation: "himalaya --account gmail envelope search/message read", Operations: []string{"mail.read"},
		ProhibitedOperations: []string{"send", "reply", "forward", "delete", "archive", "account configuration"},
		PantheonComponents:   []string{"Bifrost", "Intake"}, Resources: []string{"gmail:INBOX"},
		NetworkAccess: true, AuthenticationRequired: true, Installed: true,
	}
	if approved {
		request.DownloadApproved, request.InstallationApproved, request.UseApproved = true, true, true
	}
	capability, err := d.registerBifrostCapability(request, "test-audit")
	if err != nil {
		t.Fatal(err)
	}
	if approved {
		capability, err = d.setBifrostEnabled(capability.CapabilityID, true)
		if err != nil {
			t.Fatal(err)
		}
	}
	return capability
}

func TestKernel19BifrostSeparatesPresenceApprovalAndEnablement(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:BIFROST", "expire"))
	defer d.Close()
	capability := kernel19Capability(t, d, false)
	if capability.Installed != true || capability.RegistrationApproved != true || capability.UseApproved || capability.Enabled {
		t.Fatalf("unapproved registered capability = %#v", capability)
	}
	if _, err := d.setBifrostEnabled(capability.CapabilityID, true); err == nil {
		t.Fatal("unapproved capability was enabled")
	}
	_, err := d.bifrostMailIntake(context.Background(), bifrostMailIntakeRequest{CapabilityID: capability.CapabilityID, Bootstrap: true})
	var gate bifrostGateError
	if !errors.As(err, &gate) || gate.Kind != bifrostFailureApproval {
		t.Fatalf("unapproved invocation error=%v", err)
	}
	var invocations int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM bifrost_invocations WHERE failure_kind = ?`, bifrostFailureApproval).Scan(&invocations); err != nil {
		t.Fatal(err)
	}
	if invocations != 1 {
		t.Fatalf("approval refusal receipt count=%d", invocations)
	}
	approved := kernel19Capability(t, d, true)
	if !approved.Enabled || !approved.DownloadApproved || !approved.InstallationApproved || !approved.UseApproved {
		t.Fatalf("approved capability was not enabled: %#v", approved)
	}
	if _, err := d.setBifrostEnabled(approved.CapabilityID, false); err != nil {
		t.Fatal(err)
	}
	if disabled, err := d.loadBifrostCapability(approved.CapabilityID); err != nil || disabled.Enabled {
		t.Fatalf("disabled capability=%#v err=%v", disabled, err)
	}
	approvals, err := d.bifrostApprovals(20)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, event := range approvals {
		seen[event.EventType] = true
	}
	for _, eventType := range []string{"registration", "download", "installation", "use"} {
		if !seen[eventType] {
			t.Fatalf("approval history omitted %q: %#v", eventType, approvals)
		}
	}
}

func kernel19MailFixture(t *testing.T) (string, string) {
	t.Helper()
	var raw bytes.Buffer
	writer := multipart.NewWriter(&raw)
	bodyHeader := make(textproto.MIMEHeader)
	bodyHeader.Set("Content-Type", "text/plain; charset=utf-8")
	bodyPart, err := writer.CreatePart(bodyHeader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bodyPart.Write([]byte("A bounded message body.\n")); err != nil {
		t.Fatal(err)
	}
	allowedHeader := make(textproto.MIMEHeader)
	allowedHeader.Set("Content-Type", "text/plain")
	allowedHeader.Set("Content-Disposition", `attachment; filename="notes.txt"`)
	allowedPart, err := writer.CreatePart(allowedHeader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := allowedPart.Write([]byte("allowed attachment\n")); err != nil {
		t.Fatal(err)
	}
	blockedHeader := make(textproto.MIMEHeader)
	blockedHeader.Set("Content-Type", "text/html")
	blockedHeader.Set("Content-Disposition", `attachment; filename="unsafe.html"`)
	blockedPart, err := writer.CreatePart(blockedHeader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := blockedPart.Write([]byte("<script>do not render</script>")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return raw.String(), writer.FormDataContentType()
}

func TestKernel19BifrostMailIntakeIsBoundedNormalizedAndIdempotent(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:BIFROST", "expire"))
	defer d.Close()
	kernel19Capability(t, d, true)
	rawBody, contentType := kernel19MailFixture(t)
	rawPath := filepath.Join(t.TempDir(), "message.eml")
	if err := os.WriteFile(rawPath, []byte("From: sender@example.com\nTo: grant@example.com\nSubject: Bifrost proof\nDate: "+time.Now().UTC().Format(time.RFC1123Z)+"\nContent-Type: "+contentType+"\nMIME-Version: 1.0\n\n"+rawBody), 0o600); err != nil {
		t.Fatal(err)
	}
	argsPath := filepath.Join(t.TempDir(), "himalaya.args")
	envelopeDate := time.Now().UTC().Add(time.Minute).Format(time.RFC3339Nano)
	helper := filepath.Join(t.TempDir(), "himalaya")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\ncase \"$*\" in\n  *'envelope search'*) printf '%%s' %q ;;\n  *) cat %q ;;\nesac\n", argsPath, fmt.Sprintf(`{"envelopes":[{"id":"42","message-id":"<kernel19@example>","date":"%s","from":[{"name":"Sender","email":"sender@example.com"}],"to":[{"email":"grant@example.com"}],"subject":"Bifrost proof","has-attachment":true}]}`, envelopeDate), rawPath)
	if err := os.WriteFile(helper, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	oldHimalaya := himalayaCommand
	himalayaCommand = helper
	t.Cleanup(func() { himalayaCommand = oldHimalaya })

	first, err := d.bifrostMailIntake(context.Background(), bifrostMailIntakeRequest{CapabilityID: "gmail-mail", Bootstrap: true, MaxMessages: 10})
	if err != nil {
		t.Fatalf("first mail intake: %#v: %v", first, err)
	}
	if first.Examined != 1 || first.New != 1 || first.Blocked != 1 || first.Failed != 0 || first.IntakeScan == nil {
		t.Fatalf("first mail intake result=%#v", first)
	}
	var status, artifactPath, occurrenceID string
	if err := d.db.QueryRow(`SELECT status, artifact_path, intake_occurrence_id FROM bifrost_mail_messages WHERE source_id = '<kernel19@example>'`).Scan(&status, &artifactPath, &occurrenceID); err != nil {
		t.Fatal(err)
	}
	if status != "blocked_attachment" || artifactPath == "" || occurrenceID == "" {
		t.Fatalf("mail ledger status=%q artifact=%q occurrence=%q", status, artifactPath, occurrenceID)
	}
	artifact, err := os.ReadFile(artifactPath)
	if err != nil || !strings.Contains(string(artifact), "A bounded message body.") || !strings.Contains(string(artifact), "unsafe.html") || !strings.Contains(string(artifact), "blocked") {
		t.Fatalf("normalized artifact=%q err=%v", artifact, err)
	}
	entries, err := os.ReadDir(filepath.Dir(artifactPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected main artifact plus allowed attachment, entries=%d", len(entries))
	}
	second, err := d.bifrostMailIntake(context.Background(), bifrostMailIntakeRequest{CapabilityID: "gmail-mail", MaxMessages: 10})
	if err != nil {
		t.Fatalf("second mail intake: %#v: %v", second, err)
	}
	if second.New != 0 || second.Unchanged != 1 || second.Failed != 0 {
		t.Fatalf("second mail intake result=%#v", second)
	}
	var reads int
	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.Contains(line, "message read") {
			reads++
		}
	}
	if reads != 1 {
		t.Fatalf("message read invocations=%d, args=%s", reads, data)
	}
	var occurrences int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences WHERE relative_path LIKE 'Bifrost Mail/%'`).Scan(&occurrences); err != nil {
		t.Fatal(err)
	}
	if occurrences != 2 {
		t.Fatalf("Bifrost Intake occurrence count=%d", occurrences)
	}
}

func TestKernel19BifrostPauseAndArtifactProtectionFailClosed(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:BIFROST", "expire"))
	defer d.Close()
	envelope := bifrostMailEnvelope{ID: "artifact-proof", MessageID: "<artifact-proof@example>"}
	artifactPath, _, err := d.writeBifrostMailArtifact(envelope, bifrostMailDocument{Body: "original"}, "gmail", "INBOX")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte("human edit"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := d.writeBifrostMailArtifact(envelope, bifrostMailDocument{Body: "original"}, "gmail", "INBOX"); err == nil {
		t.Fatal("human-modified artifact was overwritten")
	} else {
		var gate bifrostGateError
		if !errors.As(err, &gate) || gate.Kind != bifrostFailureArtifact {
			t.Fatalf("artifact protection error=%v", err)
		}
	}
	kernel19Capability(t, d, true)
	oldHimalaya := himalayaCommand
	himalayaCommand = filepath.Join(t.TempDir(), "does-not-run")
	t.Cleanup(func() { himalayaCommand = oldHimalaya })
	if err := d.setPaused(true); err != nil {
		t.Fatal(err)
	}
	_, err = d.bifrostMailIntake(context.Background(), bifrostMailIntakeRequest{CapabilityID: "gmail-mail", Bootstrap: true})
	var gate bifrostGateError
	if !errors.As(err, &gate) || gate.Kind != bifrostFailurePrometheus {
		t.Fatalf("paused mail intake error=%v", err)
	}
	if err := d.setPaused(false); err != nil {
		t.Fatal(err)
	}
	if _, err := d.bifrostStatus(); err != nil {
		t.Fatal(err)
	}
}
