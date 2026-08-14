package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDaemonStatusReturnsAuthoritativeConfigAndAuditsRequest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "pantheon.json")
	capsPath := filepath.Join(dir, "capabilities.json")
	if err := os.WriteFile(cfgPath, []byte(`{"local_model":"apollo-test","ollama_url":"http://127.0.0.1:11434/api/chat","roadmap_path":"/tmp/roadmap"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capsPath, []byte(`{"capabilities":[{"name":"reason","type":"model","risk":"low"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	d, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: filepath.Join(dir, "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	d.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, body=%s", rr.Code, rr.Body.String())
	}

	var status statusResponse
	if err := json.NewDecoder(rr.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status.LocalModel != "apollo-test" {
		t.Fatalf("local model = %q", status.LocalModel)
	}
	if status.ListenAddress != "127.0.0.1" {
		t.Fatalf("listen address = %q", status.ListenAddress)
	}
	if status.AuditRequests != 1 {
		t.Fatalf("audit requests = %d, want 1", status.AuditRequests)
	}
}

func TestDaemonRouteValidatesTheConfiguredCapability(t *testing.T) {
	t.Parallel()
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{\"message\":{\"role\":\"assistant\",\"content\":\"{\\\"capability\\\":\\\"reason\\\",\\\"reason\\\":\\\"local reasoning\\\",\\\"confidence\\\":1}\"}}"))
	}))
	defer ollama.Close()
	dir := t.TempDir()
	cfgPath, capsPath := filepath.Join(dir, "pantheon.json"), filepath.Join(dir, "capabilities.json")
	if err := os.WriteFile(cfgPath, []byte(`{"local_model":"apollo-test","ollama_url":"`+ollama.URL+`","roadmap_path":"/tmp/roadmap"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capsPath, []byte(`{"capabilities":[{"name":"reason","type":"model","risk":"low"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: filepath.Join(dir, "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/route", strings.NewReader(`{"input":"What is Pantheon's objective?"}`))
	d.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("route code = %d, body=%s", rr.Code, rr.Body.String())
	}
	var result routeDecision
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Capability != "reason" || result.Confidence != 1 {
		t.Fatalf("unexpected route: %#v", result)
	}
}

func TestDaemonNotifyUsesConfiguredHelper(t *testing.T) {
	dir := t.TempDir()
	cfgPath, capsPath := filepath.Join(dir, "pantheon.json"), filepath.Join(dir, "capabilities.json")
	if err := os.WriteFile(cfgPath, []byte(`{"local_model":"apollo-test","ollama_url":"http://127.0.0.1:1","roadmap_path":"/tmp/roadmap"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capsPath, []byte(`{"capabilities":[{"name":"notify","type":"deterministic","risk":"low"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	helperPath := filepath.Join(dir, "notify-helper")
	outputPath := filepath.Join(dir, "notification.txt")
	script := "#!/bin/sh\nprintf '%s|%s|%s' \"$1\" \"$2\" \"$3\" > \"" + outputPath + "\"\n"
	if err := os.WriteFile(helperPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	old := notifyCommand
	notifyCommand = helperPath
	defer func() { notifyCommand = old }()
	d, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: filepath.Join(dir, "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/notify", strings.NewReader(`{"input":"Dinner is ready."}`))
	d.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("notify code = %d, body=%s", rr.Code, rr.Body.String())
	}
	contents, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "pantheon-alerts|Pantheon|Dinner is ready." {
		t.Fatalf("helper arguments = %q", contents)
	}
}

func TestApolloClientRejectsRedirectsAwayFromApprovedLocalHost(t *testing.T) {
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.com/steal", http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	cfg := config{LocalModel: "apollo-test", OllamaURL: redirect.URL, RoadmapPath: filepath.Join(t.TempDir(), "roadmap.md")}
	if err := os.WriteFile(cfg.RoadmapPath, []byte("roadmap"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &daemon{cfg: cfg}
	if _, err := d.reason("test"); err == nil {
		t.Fatal("Apollo client followed a redirect")
	}
}

func TestOpenVaultMarkdownRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside.md")
	link := filepath.Join(dir, "link.md")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if _, err := openVaultMarkdown(link); err == nil {
		t.Fatal("symlink was opened")
	}
}

func TestDaemonRejectsRawGrantCreationWithoutLocalControlToken(t *testing.T) {
	dir := t.TempDir()
	cfgPath, capsPath := filepath.Join(dir, "pantheon.json"), filepath.Join(dir, "capabilities.json")
	if err := os.WriteFile(cfgPath, []byte(`{"local_model":"apollo-test","ollama_url":"http://127.0.0.1:1","roadmap_path":"/tmp/roadmap"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capsPath, []byte(`{"capabilities":[{"name":"email.read","type":"delegated","risk":"low"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: filepath.Join(dir, "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	denied := httptest.NewRecorder()
	d.Handler().ServeHTTP(denied, newJSONRequest(t, http.MethodPost, "/v1/grants", grantRequest{Capability: "email.read", Scope: "bounded", IntentSummary: "test"}))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("raw grant = %d, body=%s", denied.Code, denied.Body.String())
	}
	allowed := httptest.NewRecorder()
	req := newJSONRequest(t, http.MethodPost, "/v1/grants", grantRequest{Capability: "email.read", Scope: "bounded", IntentSummary: "test"})
	req.Header.Set(controlTokenHeader, d.controlToken)
	d.Handler().ServeHTTP(allowed, req)
	if allowed.Code != http.StatusCreated {
		t.Fatalf("controlled grant = %d, body=%s", allowed.Code, allowed.Body.String())
	}
}

func TestDaemonBlocksGrantWhenAuditIsUnavailable(t *testing.T) {
	dir := t.TempDir()
	cfgPath, capsPath := filepath.Join(dir, "pantheon.json"), filepath.Join(dir, "capabilities.json")
	if err := os.WriteFile(cfgPath, []byte(`{"local_model":"apollo-test","ollama_url":"http://127.0.0.1:1","roadmap_path":"/tmp/roadmap"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capsPath, []byte(`{"capabilities":[{"name":"email.read","type":"delegated","risk":"low"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: filepath.Join(dir, "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.db.Close(); err != nil {
		t.Fatal(err)
	}
	req := newJSONRequest(t, http.MethodPost, "/v1/grants", grantRequest{Capability: "email.read", Scope: "bounded", IntentSummary: "test"})
	req.Header.Set(controlTokenHeader, d.controlToken)
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, req)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("grant with audit unavailable = %d, body=%s", response.Code, response.Body.String())
	}
}

func TestVaultConnectionIndexSkipsSymlinksOutsideVault(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	outside := filepath.Join(dir, "outside.md")
	if err := os.MkdirAll(vault, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vault, "inside.md"), []byte("# Inside\nFlourishing inside vault.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("SecretExternalTerm must never enter the index.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(vault, "escape.md")); err != nil {
		t.Fatal(err)
	}
	index, err := buildVaultConnectionIndex(vault)
	if err != nil {
		t.Fatal(err)
	}
	if index.NoteCount != 1 || index.Terms["secretexternalterm"] != 0 {
		t.Fatalf("symlink escaped vault: %#v", index)
	}
}

func TestLoadConfigRejectsNonLocalApolloEndpoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pantheon.json")
	if err := os.WriteFile(path, []byte(`{"local_model":"apollo-test","ollama_url":"http://8.8.8.8/api/chat","roadmap_path":"/tmp/roadmap"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfigPath(path); err == nil {
		t.Fatal("remote Apollo endpoint was accepted")
	}
}

func TestDaemonVaultConnectionSpikeBuildsCitedReportWithoutChangingVault(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	if err := os.MkdirAll(vault, 0o700); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(vault, "first.md")
	second := filepath.Join(vault, "second.md")
	if err := os.WriteFile(first, []byte("# Flourishing\nFlourishing grows through patient work and trust.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("# Trust\nTrust and flourishing connect people across time.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"message":{"content":"## Apollo proposal\n- Flourishing is frequently paired with trust."}}`))
	}))
	defer ollama.Close()
	cfgPath, capsPath := filepath.Join(dir, "pantheon.json"), filepath.Join(dir, "capabilities.json")
	if err := os.WriteFile(cfgPath, []byte(`{"local_model":"apollo-test","ollama_url":"`+ollama.URL+`","vault_path":"`+vault+`","roadmap_path":"`+first+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capsPath, []byte(`{"capabilities":[{"name":"vault.connect","type":"model","risk":"low"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: filepath.Join(dir, "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	rr := httptest.NewRecorder()
	d.Handler().ServeHTTP(rr, newJSONRequest(t, http.MethodPost, "/v1/vault/connections", vaultConnectionRequest{}))
	if rr.Code != http.StatusOK {
		t.Fatalf("connection spike = %d, body=%s", rr.Code, rr.Body.String())
	}
	var response vaultConnectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.NoteCount != 2 || response.ReportPath == "" || response.AuditID == "" {
		t.Fatalf("unexpected response: %#v", response)
	}
	report, err := os.ReadFile(response.ReportPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(report), "first.md:2") || !strings.Contains(string(report), "Apollo proposal") {
		t.Fatalf("missing cited evidence or proposal: %s", report)
	}
	if contents, err := os.ReadFile(first); err != nil || !strings.Contains(string(contents), "patient work") {
		t.Fatal("vault was changed")
	}
}

func TestDaemonEmailInventoryRequiresMatchingGrantAndEnforcesFiftyEnvelopeCap(t *testing.T) {
	dir := t.TempDir()
	cfgPath, capsPath := filepath.Join(dir, "pantheon.json"), filepath.Join(dir, "capabilities.json")
	if err := os.WriteFile(cfgPath, []byte(`{"local_model":"apollo-test","ollama_url":"http://127.0.0.1:1","roadmap_path":"/tmp/roadmap"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capsPath, []byte(`{"capabilities":[{"name":"email.read","type":"delegated","risk":"low"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(dir, "himalaya")
	argsPath := filepath.Join(dir, "args.txt")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \""+argsPath+"\"\nprintf '%s' '{\"envelopes\":[{\"id\":\"1\",\"subject\":\"Test\"}]}'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	old := himalayaCommand
	himalayaCommand = helper
	defer func() { himalayaCommand = old }()
	d, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: filepath.Join(dir, "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	blocked := httptest.NewRecorder()
	d.Handler().ServeHTTP(blocked, newJSONRequest(t, http.MethodPost, "/v1/email/inventory", emailInventoryRequest{Account: "gmail", Mailbox: "inbox"}))
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("ungranted inventory = %d, body=%s", blocked.Code, blocked.Body.String())
	}
	grant := httptest.NewRecorder()
	grantRequest := newJSONRequest(t, http.MethodPost, "/v1/grants", grantRequest{Capability: "email.read", Scope: "account=gmail; mailbox=inbox; page=1; page_size=50; envelopes_only", IntentSummary: "Read-only email pilot"})
	grantRequest.Header.Set(controlTokenHeader, d.controlToken)
	d.Handler().ServeHTTP(grant, grantRequest)
	if grant.Code != http.StatusCreated {
		t.Fatalf("grant = %d, body=%s", grant.Code, grant.Body.String())
	}
	ok := httptest.NewRecorder()
	d.Handler().ServeHTTP(ok, newJSONRequest(t, http.MethodPost, "/v1/email/inventory", emailInventoryRequest{Account: "gmail", Mailbox: "inbox", PageSize: 1000}))
	if ok.Code != http.StatusOK {
		t.Fatalf("inventory = %d, body=%s", ok.Code, ok.Body.String())
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "--page-size\n50\n") || !strings.Contains(string(args), "--account\ngmail\n") {
		t.Fatalf("unbounded or wrong account command: %q", args)
	}
}

func TestDaemonGrantRequiresApprovalForSelfChange(t *testing.T) {
	dir := t.TempDir()
	cfgPath, capsPath := filepath.Join(dir, "pantheon.json"), filepath.Join(dir, "capabilities.json")
	if err := os.WriteFile(cfgPath, []byte(`{"local_model":"apollo-test","ollama_url":"http://127.0.0.1:1","roadmap_path":"/tmp/roadmap"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capsPath, []byte(`{"capabilities":[{"name":"email.tag","type":"deterministic","risk":"low"},{"name":"self.change","type":"deterministic","risk":"high"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: filepath.Join(dir, "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	denied := httptest.NewRecorder()
	deniedRequest := newJSONRequest(t, http.MethodPost, "/v1/grants", grantRequest{Capability: "self.change", Scope: "config/policy.yaml", IntentSummary: "Change policy", Diff: "- old\n+ new"})
	deniedRequest.Header.Set(controlTokenHeader, d.controlToken)
	d.Handler().ServeHTTP(denied, deniedRequest)
	if denied.Code != http.StatusBadRequest {
		t.Fatalf("unapproved self change = %d, body=%s", denied.Code, denied.Body.String())
	}

	approved := httptest.NewRecorder()
	approvedRequest := newJSONRequest(t, http.MethodPost, "/v1/grants", grantRequest{Capability: "self.change", Scope: "config/policy.yaml", IntentSummary: "Change policy", Diff: "- old\n+ new", HumanApproved: true})
	approvedRequest.Header.Set(controlTokenHeader, d.controlToken)
	d.Handler().ServeHTTP(approved, approvedRequest)
	if approved.Code != http.StatusCreated {
		t.Fatalf("approved self change = %d, body=%s", approved.Code, approved.Body.String())
	}
	var grant capabilityGrant
	if err := json.NewDecoder(approved.Body).Decode(&grant); err != nil {
		t.Fatal(err)
	}
	if grant.ID == "" || grant.Status != "active" || !grant.HumanApproved {
		t.Fatalf("unexpected grant: %#v", grant)
	}
}

func newJSONRequest(t *testing.T, method, path string, value any) *http.Request {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(method, path, strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestDaemonPausePersistsAndBlocksExternalNotification(t *testing.T) {
	dir := t.TempDir()
	cfgPath, capsPath := filepath.Join(dir, "pantheon.json"), filepath.Join(dir, "capabilities.json")
	if err := os.WriteFile(cfgPath, []byte(`{"local_model":"apollo-test","ollama_url":"http://127.0.0.1:1","roadmap_path":"/tmp/roadmap"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capsPath, []byte(`{"capabilities":[{"name":"notify","type":"deterministic","risk":"low"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "state.db")
	d, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	pause := httptest.NewRecorder()
	d.Handler().ServeHTTP(pause, httptest.NewRequest(http.MethodPost, "/v1/pause", nil))
	if pause.Code != http.StatusOK {
		t.Fatalf("pause code = %d, body=%s", pause.Code, pause.Body.String())
	}
	blocked := httptest.NewRecorder()
	d.Handler().ServeHTTP(blocked, httptest.NewRequest(http.MethodPost, "/v1/notify", strings.NewReader(`{"input":"Do not deliver this."}`)))
	if blocked.Code != http.StatusServiceUnavailable {
		t.Fatalf("notify while paused = %d, body=%s", blocked.Code, blocked.Body.String())
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	status := httptest.NewRecorder()
	reopened.Handler().ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/v1/status", nil))
	var result statusResponse
	if err := json.NewDecoder(status.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if !result.Paused {
		t.Fatal("pause state was not persisted")
	}
}

func TestDaemonInputUsesConfiguredLocalApolloAndAuditsResult(t *testing.T) {
	t.Parallel()
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var request ollamaRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "apollo-test" {
			t.Fatalf("model = %q", request.Model)
		}
		if request.Stream {
			t.Fatal("daemon input must request one complete response")
		}
		if len(request.Messages) == 2 && strings.Contains(request.Messages[0].Content, "routing engine") {
			_, _ = w.Write([]byte("{\"message\":{\"role\":\"assistant\",\"content\":\"{\\\"capability\\\":\\\"reason\\\",\\\"reason\\\":\\\"local reasoning\\\",\\\"confidence\\\":1}\"}}"))
			return
		}
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"Pantheon delivers free human time."},"done":true}`))
	}))
	defer ollama.Close()

	dir := t.TempDir()
	roadmapPath := filepath.Join(dir, "roadmap.md")
	if err := os.WriteFile(roadmapPath, []byte("Objective: free human time."), 0o600); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "pantheon.json")
	capsPath := filepath.Join(dir, "capabilities.json")
	if err := os.WriteFile(cfgPath, []byte(`{"local_model":"apollo-test","ollama_url":"`+ollama.URL+`","roadmap_path":"`+roadmapPath+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capsPath, []byte(`{"capabilities":[{"name":"reason","type":"model","risk":"low"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	d, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: filepath.Join(dir, "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/input", strings.NewReader(`{"input":"What is Pantheon's objective?"}`))
	req.Header.Set("Content-Type", "application/json")
	d.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("input code = %d, body=%s", rr.Code, rr.Body.String())
	}
	var result inputResponse
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Capability != "reason" || result.Model != "apollo-test" || result.Result != "Pantheon delivers free human time." {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.AuditID == "" {
		t.Fatal("input response did not identify its audit record")
	}
}
