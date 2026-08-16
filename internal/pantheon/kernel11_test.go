package pantheon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func testKernel11Daemon(t *testing.T, root string) *daemon {
	t.Helper()
	d := testKernel9Daemon(t, testKernel9Config(t, root))
	d.controlToken = "kernel11-test-token"
	if _, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS capability_grants (id TEXT PRIMARY KEY, capability TEXT NOT NULL, scope TEXT NOT NULL, intent_summary TEXT NOT NULL, diff TEXT, human_approved INTEGER NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestKernel11MercuryConfigurationDefaultsAndOverride(t *testing.T) {
	data := []byte(`{"schema_version":1,"models":{"apollo":{"model":"local","endpoint":"http://127.0.0.1:11434/api/chat"}},"mercury":{"mode":"hermes","agent":"Terra","binary":"/tmp/hermes","deterministic_fallback":true,"round_on_cron_wake":false}}`)
	resolution, err := parseConfigBytes("test", data)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Config.MercuryMode != "hermes" || resolution.Config.MercuryAgent != "Terra" || resolution.Config.MercuryBinary != "/tmp/hermes" || !resolution.Config.MercuryDeterministicFallback || resolution.Config.MercuryRoundOnCronWake {
		t.Fatalf("Mercury config=%+v", resolution.Config)
	}
}

func TestKernel11DirectOrderSealsAndDispatchesPresentOnce(t *testing.T) {
	root := t.TempDir()
	d := testKernel11Daemon(t, root)
	source := filepath.Join(d.cfg.IntakePath, "harmless.md")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("# Harmless Kernel 11 order\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	var occurrenceID string
	if err := d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences LIMIT 1`).Scan(&occurrenceID); err != nil {
		t.Fatal(err)
	}
	request := orderRequest{RequestText: "Process this exact occurrence", Recipient: "present", Capability: "present.process", Scope: "occurrence:" + occurrenceID}
	order, err := d.createOrder(request, true)
	if err != nil {
		t.Fatal(err)
	}
	if order.Origin != orderOriginPersonDirect || order.AuthorityBasis != orderOriginPersonDirect || order.Status != orderStatusSealed || order.SealDigest == "" {
		t.Fatalf("direct order = %#v", order)
	}
	digest := order.SealDigest
	first, err := d.mercuryRound(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Completed != 1 || first.Dispatched != 1 {
		t.Fatalf("first Mercury round = %#v", first)
	}
	finished, err := d.loadOrder(order.OrderID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != orderStatusCompleted || finished.SealDigest != digest {
		t.Fatalf("finished order = %#v", finished)
	}
	var attempts, gos int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM order_attempts WHERE order_id = ?`, order.OrderID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM order_go_commands WHERE order_id = ?`, order.OrderID).Scan(&gos); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 || gos != 1 {
		t.Fatalf("attempts=%d gos=%d", attempts, gos)
	}
	second, err := d.mercuryRound(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if second.Dispatched != 0 || second.Completed != 0 {
		t.Fatalf("completed order replayed: %#v", second)
	}
}

func TestKernel11OrderIngressRequiresPersonBoundary(t *testing.T) {
	d := testKernel11Daemon(t, t.TempDir())
	rejected := httptest.NewRecorder()
	d.Handler().ServeHTTP(rejected, newJSONRequest(t, http.MethodPost, "/v1/orders", orderRequest{RequestText: "forged", Recipient: "present", Capability: "present.process", Scope: "occurrence:fake"}))
	if rejected.Code != http.StatusForbidden {
		t.Fatalf("forged order status=%d body=%s", rejected.Code, rejected.Body.String())
	}
	request := newJSONRequest(t, http.MethodPost, "/v1/orders", orderRequest{RequestText: "direct order needs no second approval", Recipient: "present", Capability: "present.process", Scope: "occurrence:fake"})
	request.Header.Set(controlTokenHeader, d.controlToken)
	accepted := httptest.NewRecorder()
	d.Handler().ServeHTTP(accepted, request)
	if accepted.Code != http.StatusCreated {
		t.Fatalf("controlled order status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	var order mercuryOrder
	if err := json.NewDecoder(accepted.Body).Decode(&order); err != nil {
		t.Fatal(err)
	}
	if order.Origin != orderOriginPersonDirect || order.Status != orderStatusSealed {
		t.Fatalf("controlled order = %#v", order)
	}
}

func TestKernel11PauseQueuesOrderAndResumeDoesNotReplayAutomatically(t *testing.T) {
	root := t.TempDir()
	d := testKernel11Daemon(t, root)
	if err := d.setPaused(true); err != nil {
		t.Fatal(err)
	}
	order, err := d.createOrder(orderRequest{RequestText: "queue while paused", Recipient: "present", Capability: "present.process", Scope: "occurrence:missing"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != orderStatusWaiting {
		t.Fatalf("paused order status=%s", order.Status)
	}
	pausedRound, err := d.mercuryRound(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if pausedRound.Dispatched != 0 {
		t.Fatalf("paused round issued Go: %#v", pausedRound)
	}
	if err := d.setPaused(false); err != nil {
		t.Fatal(err)
	}
	var gos int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM order_go_commands WHERE order_id = ?`, order.OrderID).Scan(&gos); err != nil {
		t.Fatal(err)
	}
	if gos != 0 {
		t.Fatalf("resume issued Go count=%d", gos)
	}
	next, err := d.mercuryRound(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if next.Dispatched != 1 {
		// The missing occurrence is still a legitimate deterministic dispatch attempt;
		// Present records the failed result without allowing a replay.
		t.Fatalf("post-resume round = %#v", next)
	}
	var invalidated int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM order_go_commands WHERE status = ?`, goStatusInvalidated).Scan(&invalidated); err != nil {
		t.Fatal(err)
	}
	if invalidated != 0 {
		t.Fatalf("unexpected invalidated Go after resume=%d", invalidated)
	}
}

func TestKernel11AuthorityStatesAndChildScope(t *testing.T) {
	d := testKernel11Daemon(t, t.TempDir())
	interpret, err := d.createOrder(orderRequest{RequestText: "please figure this out"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if interpret.Status != orderStatusInterpret {
		t.Fatalf("interpretation status=%s", interpret.Status)
	}
	ruling, err := d.createOrder(orderRequest{RequestText: "untrusted structured request", Recipient: "present", Capability: "present.process", Scope: "occurrence:x"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if ruling.Status != orderStatusRuling {
		t.Fatalf("ruling status=%s", ruling.Status)
	}
	blocked, err := d.createOrder(orderRequest{RequestText: "unsupported adapter", Recipient: "apollo", Capability: "apollo.query", Scope: "local"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Status != orderStatusUnavailable {
		t.Fatalf("unavailable status=%s", blocked.Status)
	}
	parent, err := d.createOrder(orderRequest{RequestText: "parent", Recipient: "present", Capability: "present.process", Scope: "occurrence:x"}, true)
	if err != nil {
		t.Fatal(err)
	}
	child, err := d.createOrder(orderRequest{RequestText: "narrow child", Recipient: "present", Capability: "present.process", Scope: "occurrence:x", ParentOrderID: parent.OrderID, ParentSealDigest: parent.SealDigest}, false)
	if err != nil {
		t.Fatal(err)
	}
	if child.Status != orderStatusSealed || child.AuthorityBasis != orderOriginParent {
		t.Fatalf("in-scope child=%#v", child)
	}
	broad, err := d.createOrder(orderRequest{RequestText: "broader child", Recipient: "present", Capability: "present.process", Scope: "occurrence:x/..", ParentOrderID: parent.OrderID, ParentSealDigest: parent.SealDigest}, false)
	if err != nil {
		t.Fatal(err)
	}
	if broad.Status != orderStatusRuling {
		t.Fatalf("broader child status=%s", broad.Status)
	}
	if _, err := d.db.Exec(`INSERT INTO capability_grants (id, capability, scope, intent_summary, human_approved, status, created_at) VALUES ('grant-test', 'present.process', 'occurrence:grant', 'test', 1, 'active', 'now')`); err != nil {
		t.Fatal(err)
	}
	grantOrder, err := d.createOrder(orderRequest{RequestText: "grant-backed", Recipient: "present", Capability: "present.process", Scope: "occurrence:grant"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if grantOrder.Status != orderStatusSealed || grantOrder.AuthorityBasis != orderOriginStanding {
		t.Fatalf("grant-backed order=%#v", grantOrder)
	}
	if _, err := d.db.Exec(`UPDATE capability_grants SET status = 'revoked' WHERE id = 'grant-test'`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.mercuryRound(context.Background()); err != nil {
		t.Fatal(err)
	}
	grantAfterRevoke, err := d.loadOrder(grantOrder.OrderID)
	if err != nil {
		t.Fatal(err)
	}
	if grantAfterRevoke.Status != orderStatusRuling {
		t.Fatalf("revoked grant order=%#v", grantAfterRevoke)
	}
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 7 {
		t.Fatalf("orders=%d", count)
	}
}
