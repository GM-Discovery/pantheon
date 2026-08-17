package pantheon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestKernel17HelpCodeDiscoveryAndWelcomeRendering(t *testing.T) {
	commands := commandDiscoveryText()
	for _, expected := range []string{"pantheon prometheus protocol --pause", "pantheon help --codes", "pantheon organism run", "pantheon show janus"} {
		if !strings.Contains(commands, expected) {
			t.Fatalf("command discovery omitted %q", expected)
		}
	}
	if strings.Contains(commands, "pantheon command-that-does-not-exist") {
		t.Fatal("command discovery advertised a nonexistent command")
	}
	codes := seshatCodeHelp([]seshatPackage{{seshatPackageSpec: seshatPackageSpec{
		SubjectCode: "NOTIFICATION", Description: "Prepare a bounded local notification.", RequiredFields: []string{"message"}, OptionalFields: []string{"title"}, Capability: "notify.send",
	}, AdapterAvailable: true}})
	if !strings.Contains(codes, "NOTIFICATION") || !strings.Contains(codes, "Requires: message") || !strings.Contains(codes, "Capability: notify.send") || !strings.Contains(codes, "Status: ready") {
		t.Fatalf("code discovery=%s", codes)
	}
	output := renderWelcome(welcomeState{Runtime: "daemon alive at 127.0.0.1:8765", Model: strings.Repeat("long-model-", 12), Vault: strings.Repeat("/a/very/long/vault/path/", 8), Paused: true})
	for _, line := range strings.Split(output, "\n") {
		if visibleWidth(line) != welcomeWidth {
			t.Fatalf("welcome line width=%d line=%q", visibleWidth(line), line)
		}
	}
	if !strings.Contains(output, "pantheon prometheus protocol --pause") || !strings.Contains(output, "pantheon help --commands") || !strings.Contains(output, "PAUSED") {
		t.Fatalf("welcome omitted operator controls: %s", output)
	}
}

func TestKernel17ActiveDocumentationReferencesCurrentSurfaces(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate repository source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	for _, path := range []string{"docs/README.md", "docs/pantheon-architecture-brief.md", "docs/operator-guide.md", "docs/future-authority.md", "docs/seshat-plugin-extension.md", "docs/janus-temporal.md"} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("documented path %s is missing: %v", path, err)
		}
	}
	for path, required := range map[string][]string{
		"docs/operator-guide.md":              {"pantheon help --commands", "pantheon help --codes", "pantheon prometheus protocol --pause", "pantheon organism run"},
		"docs/pantheon-architecture-brief.md": {"Intake", "Present", "Past", "Apollo", "Seshat", "Janus", "Future", "Mercury"},
		"docs/software.md":                    {"Intake -> Present -> Past / Weave -> Apollo -> Seshat", "Future is currently a deterministic authority-signing boundary"},
	} {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		for _, text := range required {
			if !strings.Contains(string(data), text) {
				t.Fatalf("%s omitted active reference %q", path, text)
			}
		}
	}
}

func kernel17ApolloServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ollamaResponse{Message: message{Role: "assistant", Content: `{"answer":"The bounded source supports the requested local proof.","status":"complete","action_items":[],"commitments":[],"decisions":[],"unresolved_threads":[],"dependencies":[],"waiting_conditions":[],"time_constraints":[],"facts":[],"relationships":[],"interpretations":[],"recommendations":[],"candidate_followups":[],"uncertainties":[]}`}})
	}))
}

func kernel17NotificationDaemon(t *testing.T, grant bool) (*daemon, string) {
	t.Helper()
	root := t.TempDir()
	d := kernel15Daemon(t, seshatPackageSpec{
		SubjectCode: "NOTIFICATION", Version: "1.0.0", Description: "Kernel 17 local notification proof", Capability: "notify.send", Adapter: "local-notify",
		RequiredFields: []string{"message"}, OptionalFields: []string{"title"}, PayloadRules: map[string]string{"message": "body", "title": "TITLE"}, Returns: []string{"status"},
		PermissionClass: "bounded_local_output", PermissionShapes: []string{"once"},
	})
	server := kernel17ApolloServer(t)
	t.Cleanup(server.Close)
	d.cfg.OllamaURL, d.cfg.LocalModel = server.URL, "apollo-kernel17"
	d.cfg.VaultPath = filepath.Join(root, "Vault")
	d.cfg.RawPath = filepath.Join(d.cfg.VaultPath, "80 Pantheon Raw")
	d.cfg.ApolloReportsPath = filepath.Join(d.cfg.VaultPath, "70 Pantheon Reports", "Apollo")
	d.capabilities = capabilityConfig{Capabilities: []capability{{Name: "notify.send", Type: "deterministic", Risk: "low"}}}
	if grant {
		kernel13Grant(t, d, capabilityGrant{ID: "kernel17-notify", Holder: "future", Capability: "notify.send", Scope: "notification:kernel17", IntentSummary: "bounded Kernel 17 local proof", HumanApproved: true})
	}
	return d, root
}

func TestKernel17AuthorizedWholeAnimalPathExecutesOnce(t *testing.T) {
	d, _ := kernel17NotificationDaemon(t, true)
	defer d.Close()
	output := filepath.Join(t.TempDir(), "notification.txt")
	helper := filepath.Join(t.TempDir(), "notify-helper")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nprintf '%s|%s|%s' \"$1\" \"$2\" \"$3\" > \""+output+"\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	oldNotify := notifyCommand
	notifyCommand = helper
	defer func() { notifyCommand = oldNotify }()
	source := filepath.Join(d.cfg.IntakePath, "kernel17.md")
	if err := os.WriteFile(source, []byte("# Kernel 17\n\nA harmless whole-animal proof source.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	var occurrenceID string
	if err := d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences LIMIT 1`).Scan(&occurrenceID); err != nil {
		t.Fatal(err)
	}
	run, err := d.runOrganism(context.Background(), organismRunRequest{
		OccurrenceID: occurrenceID, RequestText: "Prepare the bounded local proof.", PackageCode: "NOTIFICATION", Scope: "notification:kernel17",
		Missive: "TYPE: NOTIFICATION\nTITLE: Kernel 17\n\nThe bounded local proof completed.",
	})
	if err != nil || run.State != "completed" || run.Trace.FutureState != futureStateSealed || run.Trace.MercuryStatus != orderStatusCompleted {
		t.Fatalf("whole-animal run=%#v err=%v", run, err)
	}
	contents, err := os.ReadFile(output)
	if err != nil || string(contents) != "pantheon-alerts|Kernel 17|The bounded local proof completed." {
		t.Fatalf("notification=%q err=%v", contents, err)
	}
	var attempts, gos int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM order_attempts WHERE order_id = ?`, run.Trace.MercuryOrderID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM order_go_commands WHERE order_id = ?`, run.Trace.MercuryOrderID).Scan(&gos); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 || gos != 1 {
		t.Fatalf("execution was not exactly once: attempts=%d gos=%d", attempts, gos)
	}
}

func TestKernel17InsufficientAuthorityDoesNotExecute(t *testing.T) {
	d, _ := kernel17NotificationDaemon(t, false)
	defer d.Close()
	output := filepath.Join(t.TempDir(), "notification.txt")
	helper := filepath.Join(t.TempDir(), "notify-helper")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nprintf delivered > \""+output+"\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	oldNotify := notifyCommand
	notifyCommand = helper
	defer func() { notifyCommand = oldNotify }()
	run, err := d.runOrganism(context.Background(), organismRunRequest{
		RequestText: "Prepare a bounded notification without authority.", PackageCode: "NOTIFICATION", Scope: "notification:kernel17-no-authority",
		Missive: "TYPE: NOTIFICATION\n\nThis must not be delivered.",
	})
	if err != nil || run.State != "awaiting_authority" || run.Trace.FutureState != futureStateNeedsPerson {
		t.Fatalf("insufficient-authority run=%#v err=%v", run, err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("insufficient authority executed adapter: stat err=%v", err)
	}
	var orders int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM orders WHERE recipient = 'local-notify'`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if orders != 0 {
		t.Fatalf("Future created an execution order without authority: %d", orders)
	}
}
