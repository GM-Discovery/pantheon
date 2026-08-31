package pantheon

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestKernel33ExternalTransportSendsOnlyMinimizedMissive(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL33", "expire"))
	defer d.Close()

	var requestBody []byte
	var authorization string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		requestBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read provider request: %v", err)
		}
		authorization = r.Header.Get("Authorization")
		var chatRequest archimedesChatRequest
		if err := json.Unmarshal(requestBody, &chatRequest); err != nil || len(chatRequest.Messages) != 2 {
			t.Errorf("decode provider request: %v", err)
		}
		var minimized archimedesMissive
		if len(chatRequest.Messages) == 2 {
			if err := json.Unmarshal([]byte(chatRequest.Messages[1].Content), &minimized); err != nil {
				t.Errorf("decode minimized missive: %v", err)
			}
		}
		aliases := make([]string, 0, len(minimized.Entities))
		for _, entity := range minimized.Entities {
			aliases = append(aliases, entity.Value)
		}
		answer := "The bounded answer identifies " + strings.Join(aliases, ", ") + "."
		w.Header().Set("Content-Type", "application/json")
		response, _ := json.Marshal(map[string]any{"model": "frontier-test", "choices": []any{map[string]any{"message": map[string]string{"content": answer}}}})
		_, _ = w.Write(response)
	}))
	defer provider.Close()

	const credentialEnv = "PANTHEON_KERNEL33_TEST_KEY"
	if err := os.Setenv(credentialEnv, "test-secret-must-not-be-persisted"); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv(credentialEnv)
	d.cfg.ArchimedesEnabled = true
	d.cfg.ArchimedesProvider = archimedesProviderOpenAICompatible
	d.cfg.ArchimedesEndpoint = provider.URL
	d.cfg.ArchimedesModel = "frontier-test"
	d.cfg.ArchimedesRoute = "test-route"
	d.cfg.ArchimedesCredentialEnv = credentialEnv
	d.cfg.ArchimedesBifrostCapabilityID = "archimedes-provider-test"
	d.cfg.ArchimedesTimeoutSeconds = 5
	d.cfg.ArchimedesMaxResponseBytes = 1 << 20

	capability, err := d.registerBifrostCapability(bifrostRegistrationRequest{
		CapabilityID:           d.cfg.ArchimedesBifrostCapabilityID,
		Name:                   "Archimedes test provider",
		Type:                   bifrostCapabilityTypeModel,
		Source:                 "kernel33-test",
		Version:                "test",
		Description:            "local fake HTTP provider for the Kernel 33 transport proof",
		Operations:             []string{"frontier.exchange"},
		ProhibitedOperations:   []string{"vault.read", "identity.lookup", "tool.invoke", "route.create"},
		PantheonComponents:     []string{"Archimedes"},
		NetworkAccess:          true,
		SendsDataOffMachine:    true,
		AuthenticationRequired: true,
		Installed:              true,
		DownloadApproved:       true,
		InstallationApproved:   true,
		UseApproved:            true,
	}, "kernel33-test")
	if err != nil {
		t.Fatal(err)
	}
	if !capability.Enabled {
		t.Fatalf("test provider capability was not enabled: %+v", capability)
	}

	missive := archimedesMissive{
		Question:       "Should Jane Doe coordinate with Acme about the launch?",
		Facts:          []string{"Jane Doe owns the decision; Acme is the partner organization."},
		DesiredProduct: "comparison",
		Entities:       []archimedesEntity{{Value: "Jane Doe", Kind: "person"}, {Value: "Acme", Kind: "organization"}},
	}
	payload, err := json.Marshal(archimedesExchangeOrderPayload{Provider: archimedesProviderOpenAICompatible, Route: d.cfg.ArchimedesRoute, Missive: missive})
	if err != nil {
		t.Fatal(err)
	}
	order, err := d.createOrder(orderRequest{
		RequestText: missive.Question,
		Recipient:   "archimedes",
		Capability:  archimedesCapability,
		Scope:       "exchange:kernel33-test",
		Payload:     payload,
		Purpose:     "Kernel 33 external transport acceptance test",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.mercuryRound(context.Background()); err != nil {
		t.Fatal(err)
	}
	loadedOrder, err := d.loadOrder(order.OrderID)
	if err != nil {
		t.Fatal(err)
	}
	if loadedOrder.Status != orderStatusCompleted {
		t.Fatalf("order status=%s error=%s", loadedOrder.Status, loadedOrder.LastError)
	}

	if strings.Contains(string(requestBody), "Jane Doe") || strings.Contains(string(requestBody), "Acme") {
		t.Fatalf("provider received raw identity: %s", requestBody)
	}
	if !strings.HasPrefix(authorization, "Bearer ") || strings.Contains(string(requestBody), "test-secret") {
		t.Fatalf("provider request did not use isolated credential header safely: auth=%q body=%s", authorization, requestBody)
	}

	exchange := archimedesExchangeByOrder(t, d, order.OrderID)
	if exchange.Provider != archimedesProviderOpenAICompatible || exchange.ProviderCapabilityID != d.cfg.ArchimedesBifrostCapabilityID || exchange.State != archimedesStateCompleted {
		t.Fatalf("exchange=%+v", exchange)
	}
	result, err := os.ReadFile(exchange.DurableResultPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "Jane Doe") || !strings.Contains(string(result), "Acme") {
		t.Fatalf("local result was not rehydrated: %s", result)
	}

	receipts, err := d.bifrostInvocations(20)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, receipt := range receipts {
		if receipt.CapabilityID == d.cfg.ArchimedesBifrostCapabilityID && receipt.Operation == "frontier.exchange" {
			found = true
			if receipt.Outcome != bifrostOutcomeCompleted || !receipt.NetworkActivity || strings.Contains(receipt.Error, "test-secret") || strings.Contains(receipt.ResultSummary, "test-secret") {
				t.Fatalf("unsafe Bifrost receipt=%+v", receipt)
			}
		}
	}
	if !found {
		t.Fatal("missing Bifrost Archimedes invocation receipt")
	}
}

func TestKernel33ExternalTransportCannotBypassBifrost(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL33GATE", "expire"))
	defer d.Close()
	d.cfg.ArchimedesEnabled = true
	d.cfg.ArchimedesProvider = archimedesProviderOpenAICompatible
	d.cfg.ArchimedesEndpoint = "https://provider.invalid/v1/chat/completions"
	d.cfg.ArchimedesModel = "frontier-test"
	d.cfg.ArchimedesRoute = "test-route"
	d.cfg.ArchimedesCredentialEnv = "PANTHEON_KERNEL33_MISSING_KEY"
	d.cfg.ArchimedesBifrostCapabilityID = "missing-archimedes-capability"

	missive := archimedesMissive{Question: "Compare two generalized options."}
	payload, err := json.Marshal(archimedesExchangeOrderPayload{Provider: archimedesProviderOpenAICompatible, Route: d.cfg.ArchimedesRoute, Missive: missive})
	if err != nil {
		t.Fatal(err)
	}
	order, err := d.createOrder(orderRequest{RequestText: missive.Question, Recipient: "archimedes", Capability: archimedesCapability, Scope: "exchange:kernel33-gate", Payload: payload, Purpose: "Kernel 33 Bifrost gate test"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.mercuryRound(context.Background()); err != nil {
		t.Fatal(err)
	}
	loadedOrder, err := d.loadOrder(order.OrderID)
	if err != nil {
		t.Fatal(err)
	}
	if loadedOrder.Status != orderStatusFailed || !strings.Contains(loadedOrder.LastError, "Bifrost") {
		t.Fatalf("external order bypassed Bifrost or failed for wrong reason: %+v", loadedOrder)
	}
	receipts, err := d.bifrostInvocations(20)
	if err != nil {
		t.Fatal(err)
	}
	for _, receipt := range receipts {
		if receipt.CapabilityID == d.cfg.ArchimedesBifrostCapabilityID && receipt.Operation == "frontier.exchange" && receipt.NetworkActivity {
			t.Fatalf("Bifrost gate recorded network activity without a capability: %+v", receipt)
		}
	}
}

func TestKernel33PrivacyCorpusStaysWithinTheGate(t *testing.T) {
	cases := []struct {
		name          string
		missive       archimedesMissive
		wantFindings  int
		wantGateState string
		absent        []string
	}{
		{
			name: "explicit identities are replaced",
			missive: archimedesMissive{
				Question: "Should Jane Doe coordinate with Acme?",
				Facts:    []string{"Jane Doe owns the decision; Acme is the partner."},
				Entities: []archimedesEntity{{Value: "Jane Doe", Kind: "person"}, {Value: "Acme", Kind: "organization"}},
			},
			wantFindings:  0,
			wantGateState: archimedesGatePassed,
			absent:        []string{"Jane Doe", "Acme"},
		},
		{
			name:          "email is redacted",
			missive:       archimedesMissive{Question: "Send the decision to alice@example.com."},
			wantFindings:  1,
			wantGateState: archimedesGateRedacted,
			absent:        []string{"alice@example.com"},
		},
		{
			name:          "phone is redacted",
			missive:       archimedesMissive{Question: "Call +1 415-555-1212 about the decision."},
			wantFindings:  1,
			wantGateState: archimedesGateRedacted,
			absent:        []string{"+1 415-555-1212"},
		},
		{
			name:          "filesystem path is redacted",
			missive:       archimedesMissive{Question: "Inspect /home/brick/Documents/Obsidian Vault/notes.md."},
			wantFindings:  1,
			wantGateState: archimedesGateRedacted,
			absent:        []string{"/home/brick/Documents/Obsidian Vault/notes.md"},
		},
		{
			name:          "multiple disclosures block",
			missive:       archimedesMissive{Question: "Send alice@example.com a call reminder at +1 415-555-1212."},
			wantFindings:  2,
			wantGateState: archimedesGateBlocked,
			absent:        nil,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			minimized, aliases, err := archimedesMinimize(testCase.missive)
			if err != nil {
				t.Fatal(err)
			}
			checked, findings := archimedesPrivacyGate(minimized, aliases)
			if len(findings) != testCase.wantFindings {
				t.Fatalf("findings=%+v, want %d", findings, testCase.wantFindings)
			}
			gateState := archimedesGatePassed
			if len(findings) == 1 {
				gateState = archimedesGateRedacted
			} else if len(findings) >= 2 {
				gateState = archimedesGateBlocked
			}
			if gateState != testCase.wantGateState {
				t.Fatalf("gate state=%s, want %s", gateState, testCase.wantGateState)
			}
			encoded, err := json.Marshal(checked)
			if err != nil {
				t.Fatal(err)
			}
			for _, value := range testCase.absent {
				if strings.Contains(string(encoded), value) {
					t.Fatalf("sanitized missive retained %q: %s", value, encoded)
				}
			}
		})
	}
}
