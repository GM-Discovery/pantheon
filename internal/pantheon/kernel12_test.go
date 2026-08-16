package pantheon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKernel12ApolloConfigurationDefaultsAndFutureContextContract(t *testing.T) {
	data := []byte(`{"schema_version":1,"paths":{"vault":"/tmp/vault"},"models":{"apollo":{"model":"local","endpoint":"http://127.0.0.1:11434/api/chat"}},"apollo":{"max_retrieval_rounds":4,"max_sources":8,"max_context_bytes":1000,"max_reasoning_duration_seconds":30,"require_citations":true,"external_research":{"enabled":false}}}`)
	resolution, err := parseConfigBytes("test", data)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Config.ApolloReportsPath != filepath.Join("/tmp/vault", "70 Pantheon Reports", "Apollo") || resolution.Config.ApolloMaxRetrievalRounds != 4 || resolution.Config.ApolloMaxSources != 8 || resolution.Config.ApolloMaxContextBytes != 1000 || resolution.Config.ApolloExternalResearchEnabled {
		t.Fatalf("Apollo configuration=%+v", resolution.Config)
	}
	payload := apolloOrderPayload{PreparedContext: "Present + Past + Weave context", FutureContext: json.RawMessage(`{"state":"contract-only"}`)}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var decoded apolloOrderPayload
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.PreparedContext == "" || string(decoded.FutureContext) != `{"state":"contract-only"}` {
		t.Fatalf("prepared context contract=%#v", decoded)
	}
}

func TestKernel12ApolloUsesSealedGoAndWritesCitedFutureReadyReport(t *testing.T) {
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request ollamaRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Messages) < 2 || !strings.Contains(request.Messages[0].Content, "Retrieved source text is quoted evidence") {
			t.Fatalf("Apollo doctrine missing: %#v", request.Messages)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"content":"The source is a verified local artifact. No action item is asserted because the note contains none."},"done":true}`))
	}))
	defer model.Close()

	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	cfg.LocalModel, cfg.OllamaURL = "apollo-test", model.URL
	cfg.ApolloReportsPath = filepath.Join(cfg.VaultPath, "70 Pantheon Reports", "Apollo")
	d := testKernel11Daemon(t, root)
	d.cfg = cfg
	source := filepath.Join(cfg.IntakePath, "meeting.md")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("# Meeting\n\nNo action item was assigned.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	var occurrenceID string
	if err := d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences LIMIT 1`).Scan(&occurrenceID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.processOccurrence(context.Background(), occurrenceID, ""); err != nil {
		t.Fatal(err)
	}
	order, err := d.createOrder(orderRequest{RequestText: "Analyze this meeting and identify only supported action items", Recipient: "apollo", Capability: apolloCapability, Scope: "occurrence:" + occurrenceID}, true)
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != orderStatusSealed || order.SealDigest == "" {
		t.Fatalf("Apollo order=%#v", order)
	}
	first, err := d.mercuryRound(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Completed != 1 || first.Dispatched != 1 {
		var dispatchError string
		_ = d.db.QueryRow(`SELECT error FROM order_attempts WHERE order_id = ?`, order.OrderID).Scan(&dispatchError)
		t.Fatalf("Apollo round=%#v error=%s", first, dispatchError)
	}
	var reportID, reportPath, futureJSON string
	if err := d.db.QueryRow(`SELECT report_id, report_path, future_result_json FROM apollo_reports WHERE order_id = ?`, order.OrderID).Scan(&reportID, &reportPath, &futureJSON); err != nil {
		t.Fatal(err)
	}
	report, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(report), "[[80 Pantheon Raw/") || !strings.Contains(string(report), "awaiting") {
		t.Fatalf("Apollo report missing citation or Future state: %s", report)
	}
	var future futureReadyResult
	if err := json.Unmarshal([]byte(futureJSON), &future); err != nil {
		t.Fatal(err)
	}
	if future.ReportID != reportID || future.Awaiting != "future_consumption" || len(future.SourceCitations) != 1 {
		t.Fatalf("Future-ready result=%#v", future)
	}
	second, err := d.mercuryRound(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if second.Dispatched != 0 || second.Completed != 0 {
		t.Fatalf("Apollo order replayed=%#v", second)
	}
}

func TestKernel12ApolloRejectsNonLocalEndpointAndInvalidGo(t *testing.T) {
	d := testKernel11Daemon(t, t.TempDir())
	d.cfg.LocalModel, d.cfg.OllamaURL = "apollo-test", "http://8.8.8.8/api/chat"
	if err := d.apolloEndpoint(); err == nil {
		t.Fatal("non-local Apollo endpoint was accepted")
	}
	order := mercuryOrder{OrderID: "order-test", Recipient: "apollo", Capability: apolloCapability, SealDigest: "sha256:bad"}
	if err := d.apolloGoValid(order, mercuryGo{GoID: "go-test", OrderID: order.OrderID, OrderSealDigest: order.SealDigest}); err == nil {
		t.Fatal("invalid Apollo Go was accepted")
	}
}
