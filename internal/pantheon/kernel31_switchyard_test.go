package pantheon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func configureKernel31Switchyard(t *testing.T, d *daemon, endpoint string) {
	t.Helper()
	d.cfg.ArchimedesEnabled = true
	d.cfg.ArchimedesProvider = archimedesProviderSwitchyard
	d.cfg.ArchimedesRoute = "switchyard/frontier"
	d.cfg.ArchimedesModel = "switchyard-fake"
	d.cfg.ArchimedesCredentialEnv = ""
	d.cfg.ArchimedesBifrostCapabilityID = "archimedes-switchyard-local"
	d.cfg.ArchimedesTimeoutSeconds = 5
	d.cfg.ArchimedesMaxResponseBytes = 1 << 20
	capability, err := d.registerBifrostCapability(bifrostRegistrationRequest{
		CapabilityID: "archimedes-switchyard-local", Name: "Switchyard local transport", Type: bifrostCapabilityTypeApplication,
		Source: "third_party/switchyard/0.2.0/native/bin/switchyard-server", Version: "0.2.0",
		Description: "Local Switchyard application boundary for Archimedes tests.",
		Operations:  []string{"frontier.exchange"}, Resources: []string{"switchyard://route/switchyard/frontier"},
		ProhibitedOperations: []string{"vault_query", "identity_lookup", "tool_invocation", "arbitrary_route_creation"}, PantheonComponents: []string{"Archimedes", "Bifrost"},
		NetworkAccess: true, SendsDataOffMachine: false, AuthenticationRequired: false,
		Installed: true, DownloadApproved: true, InstallationApproved: true, UseApproved: true,
	}, "kernel31-test")
	if err != nil {
		t.Fatal(err)
	}
	if !capability.Enabled {
		t.Fatalf("Switchyard capability was not enabled after explicit test approvals: %+v", capability)
	}
	if _, err := d.saveBifrostApplicationConfiguration(capability.CapabilityID, bifrostApplicationConfigurationRequest{
		Application: archimedesProviderSwitchyard, Endpoint: endpoint, Route: d.cfg.ArchimedesRoute, Model: d.cfg.ArchimedesModel,
	}); err != nil {
		t.Fatal(err)
	}
}

func createKernel31SwitchyardOrder(t *testing.T, d *daemon, missive archimedesMissive) mercuryOrder {
	t.Helper()
	payload, err := json.Marshal(archimedesExchangeOrderPayload{Provider: archimedesProviderSwitchyard, Route: "switchyard/frontier", Missive: missive})
	if err != nil {
		t.Fatal(err)
	}
	order, err := d.createOrder(orderRequest{RequestText: missive.Question, Recipient: "archimedes", Capability: archimedesCapability, Scope: "exchange:kernel31", Payload: payload, Purpose: "Kernel 31 Switchyard local boundary"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != orderStatusSealed {
		t.Fatalf("Switchyard order status=%s error=%s", order.Status, order.LastError)
	}
	command, err := d.issueGo(order)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.dispatchOrder(context.Background(), order, command); err != nil {
		t.Fatal(err)
	}
	return order
}

func startKernel31Switchyard(t *testing.T, upstreamBaseURL string) string {
	t.Helper()
	binary := os.Getenv("PANTHEON_SWITCHYARD_SERVER")
	if strings.TrimSpace(binary) == "" {
		binary = "/home/brick/pantheon-runtime/third_party/switchyard/0.2.0/native/bin/switchyard-server"
	}
	if _, err := os.Stat(binary); err != nil {
		t.Skipf("Kernel 31 native Switchyard binary is not staged at %s: %v", binary, err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Kernel 31 could not allocate a local Switchyard port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	configFile, err := os.CreateTemp(t.TempDir(), "switchyard-kernel31-*.toml")
	if err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf(`schema_version = 1

[llm_clients.fake]
format = "openai_chat"
base_url = %q
max_retries = 0

[targets.fake]
id = "switchyard-fake"
llm_client = "fake"

[routes.frontier]
id = "switchyard/frontier"
type = "passthrough"
target = "fake"
`, strings.TrimRight(upstreamBaseURL, "/")+"/v1")
	if _, err := configFile.WriteString(config); err != nil {
		t.Fatal(err)
	}
	if err := configFile.Close(); err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	command := exec.Command(binary, "--config", configFile.Name(), "--host", "127.0.0.1", "--port", fmt.Sprint(port))
	command.Stdout = io.Discard
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatalf("start native Switchyard: %v", err)
	}
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		_ = command.Wait()
	})
	endpoint := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: 150 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		response, requestErr := client.Get(endpoint + "/health")
		if requestErr == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return endpoint
			}
		}
		if command.ProcessState != nil && command.ProcessState.Exited() {
			t.Fatalf("native Switchyard exited before readiness: %s", strings.TrimSpace(stderr.String()))
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("native Switchyard did not become ready: %s", strings.TrimSpace(stderr.String()))
	return ""
}

func TestKernel31SwitchyardRehydratesAndReturnsTypedApolloDraft(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL31", "expire"))
	defer d.Close()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request archimedesChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Messages) != 2 || strings.Contains(request.Messages[1].Content, "Jane Doe") {
			t.Fatalf("Switchyard received unsanitized request: %+v", request)
		}
		var missive archimedesMissive
		if err := json.Unmarshal([]byte(request.Messages[1].Content), &missive); err != nil {
			t.Fatal(err)
		}
		if len(missive.Entities) != 1 {
			t.Fatalf("Switchyard missive entities=%+v", missive.Entities)
		}
		alias := missive.Entities[0].Value
		content, _ := json.Marshal(map[string]any{
			"answer": "The next step is to meet " + alias + ".",
			"actions": []any{map[string]any{
				"kind": "calendar_event", "summary": "Meet " + alias,
				"payload": map[string]any{"title": "Meet " + alias, "start": "2026-08-25T10:00:00-07:00", "end": "2026-08-25T11:00:00-07:00"},
			}},
		})
		response, _ := json.Marshal(map[string]any{"model": "switchyard-fake", "choices": []any{map[string]any{"message": map[string]string{"content": string(content)}}}})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(response)
	}))
	defer provider.Close()
	switchyard := startKernel31Switchyard(t, provider.URL)
	configureKernel31Switchyard(t, d, switchyard+"/v1/chat/completions")
	inspection, err := d.archimedesInspection()
	if err != nil || inspection["configuration_owner"] != "Bifrost" || inspection["application_configured"] != true || inspection["route"] != "switchyard/frontier" {
		t.Fatalf("Switchyard configuration was not exposed through the Bifrost-owned Archimedes surface: inspection=%+v err=%v", inspection, err)
	}

	order := createKernel31SwitchyardOrder(t, d, archimedesMissive{Question: "Arrange time with Jane Doe.", DesiredProduct: "decision", Entities: []archimedesEntity{{Value: "Jane Doe", Kind: "person"}}})
	exchange := archimedesExchangeByOrder(t, d, order.OrderID)
	if exchange.State != archimedesStateCompleted {
		t.Fatalf("Switchyard exchange=%+v", exchange)
	}
	result, err := os.ReadFile(exchange.DurableResultPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "Jane Doe") || !strings.Contains(string(result), "Apollo handoff") || !strings.Contains(string(result), "typed") {
		t.Fatalf("unexpected Archimedes result: %s", result)
	}
	drafts, err := d.listActionDrafts(20)
	if err != nil || len(drafts) != 1 {
		t.Fatalf("Apollo drafts=%+v err=%v", drafts, err)
	}
	if drafts[0].Kind != actionKindCalendar || drafts[0].State != actionStateProposed || !strings.Contains(string(drafts[0].Payload), "Jane Doe") || strings.Contains(string(drafts[0].Payload), "Person#") {
		t.Fatalf("unexpected rehydrated Apollo draft=%+v", drafts[0])
	}
	receipts, err := d.bifrostInvocations(20)
	if err != nil || len(receipts) == 0 || receipts[0].Outcome != bifrostOutcomeCompleted || !receipts[0].NetworkActivity {
		t.Fatalf("Switchyard Bifrost receipt=%+v err=%v", receipts, err)
	}
}

func TestKernel31SwitchyardFailureIsTracedWithoutFallback(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL31FAIL", "expire"))
	defer d.Close()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fake Switchyard upstream failure", http.StatusBadGateway)
	}))
	defer provider.Close()
	configureKernel31Switchyard(t, d, provider.URL)
	payload, _ := json.Marshal(archimedesExchangeOrderPayload{Provider: archimedesProviderSwitchyard, Route: "switchyard/frontier", Missive: archimedesMissive{Question: "Return a bounded answer."}})
	order, err := d.createOrder(orderRequest{RequestText: "Return a bounded answer.", Recipient: "archimedes", Capability: archimedesCapability, Scope: "exchange:kernel31-failure", Payload: payload, Purpose: "Kernel 31 failure proof"}, true)
	if err != nil {
		t.Fatal(err)
	}
	command, err := d.issueGo(order)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.dispatchOrder(context.Background(), order, command); err == nil {
		t.Fatal("Switchyard failure unexpectedly completed")
	}
	exchange := archimedesExchangeByOrder(t, d, order.OrderID)
	if exchange.State != archimedesStateFailed || !strings.Contains(exchange.Error, "Switchyard returned HTTP 502") {
		t.Fatalf("untraced Switchyard failure=%+v", exchange)
	}
	receipts, err := d.bifrostInvocations(20)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, receipt := range receipts {
		if receipt.Operation == "frontier.exchange" && receipt.Outcome == bifrostOutcomeFailed && strings.Contains(receipt.Error, "Switchyard returned HTTP 502") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing traced Switchyard failure receipt: %+v", receipts)
	}
	if drafts, _ := d.listActionDrafts(20); len(drafts) != 0 {
		t.Fatalf("failed Switchyard exchange created Apollo drafts: %+v", drafts)
	}
}

func TestKernel31SwitchyardConfigurationLivesUnderBifrostSurface(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL31CONFIG", "expire"))
	defer d.Close()
	capability, err := d.registerBifrostCapability(bifrostRegistrationRequest{
		CapabilityID: "archimedes-switchyard-config", Name: "Switchyard configuration proof", Type: bifrostCapabilityTypeApplication,
		Source: "third_party/switchyard/0.2.0/native/bin/switchyard-server", Version: "0.2.0", Description: "Switchyard configuration ownership proof.",
		Installed: true, DownloadApproved: true, InstallationApproved: true, UseApproved: true, NetworkAccess: true,
	}, "kernel31-config")
	if err != nil {
		t.Fatal(err)
	}
	session := httptest.NewRecorder()
	d.Handler().ServeHTTP(session, httptest.NewRequest(http.MethodPost, "/v1/gui/session", nil))
	if session.Code != http.StatusOK || len(session.Result().Cookies()) != 1 {
		t.Fatalf("GUI session status=%d cookies=%v", session.Code, session.Result().Cookies())
	}
	payload, _ := json.Marshal(bifrostApplicationConfigurationRequest{Application: "switchyard", Endpoint: "http://127.0.0.1:4000/v1/chat/completions", Route: "pantheon/frontier", Model: "local-fake"})
	request := httptest.NewRequest(http.MethodPost, "/v1/gui/bifrost/capabilities/"+url.PathEscape(capability.CapabilityID)+"/configuration", strings.NewReader(string(payload)))
	request.AddCookie(session.Result().Cookies()[0])
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GUI Bifrost configuration status=%d body=%s", response.Code, response.Body.String())
	}
	configured, err := d.loadBifrostCapability(capability.CapabilityID)
	if err != nil || configured.ApplicationConfig == nil || configured.ApplicationConfig.Route != "pantheon/frontier" {
		t.Fatalf("Bifrost application config=%+v err=%v", configured.ApplicationConfig, err)
	}
}
