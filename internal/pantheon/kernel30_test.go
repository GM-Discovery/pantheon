package pantheon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func createKernel30ArchimedesOrder(t *testing.T, d *daemon, missive archimedesMissive) mercuryOrder {
	t.Helper()
	payload, err := json.Marshal(archimedesExchangeOrderPayload{Provider: archimedesProviderFake, Route: "passthrough-local-fake", Missive: missive})
	if err != nil {
		t.Fatal(err)
	}
	order, err := d.createOrder(orderRequest{
		RequestText: missive.Question,
		Recipient:   "archimedes",
		Capability:  archimedesCapability,
		Scope:       "exchange:kernel30-test",
		Payload:     payload,
		Purpose:     "Kernel 30 acceptance test",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != orderStatusSealed {
		t.Fatalf("Archimedes order status=%s error=%s", order.Status, order.LastError)
	}
	return order
}

func archimedesExchangeByOrder(t *testing.T, d *daemon, orderID string) archimedesExchange {
	t.Helper()
	var exchangeID string
	if err := d.db.QueryRow(`SELECT exchange_id FROM archimedes_exchanges WHERE order_id = ?`, orderID).Scan(&exchangeID); err != nil {
		t.Fatal(err)
	}
	exchange, err := d.loadArchimedesExchange(exchangeID)
	if err != nil {
		t.Fatal(err)
	}
	return exchange
}

func TestKernel30FakeExchangeRehydratesAndDestroysMap(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL30", "expire"))
	defer d.Close()
	missive := archimedesMissive{
		Question:       "Should Jane Doe coordinate with Acme about the launch?",
		Facts:          []string{"Jane Doe owns the decision; Acme is the partner organization."},
		Constraints:    []string{"Return a concise comparison."},
		DesiredProduct: "comparison",
		Entities:       []archimedesEntity{{Value: "Jane Doe", Kind: "person"}, {Value: "Acme", Kind: "organization"}},
	}
	order := createKernel30ArchimedesOrder(t, d, missive)
	round, err := d.mercuryRound(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if round.Completed != 1 || round.Dispatched != 1 {
		t.Fatalf("Mercury round=%+v", round)
	}
	loadedOrder, err := d.loadOrder(order.OrderID)
	if err != nil {
		t.Fatal(err)
	}
	if loadedOrder.Status != orderStatusCompleted {
		t.Fatalf("order status=%s error=%s", loadedOrder.Status, loadedOrder.LastError)
	}
	exchange := archimedesExchangeByOrder(t, d, order.OrderID)
	if exchange.State != archimedesStateCompleted || exchange.RehydrationState != "completed" || exchange.DurableResultPath == "" || exchange.RetentionMetadataPath == "" {
		t.Fatalf("exchange=%+v", exchange)
	}
	details, err := d.archimedesExchangeDetails(exchange.ExchangeID)
	if err != nil {
		t.Fatal(err)
	}
	receipt, ok := details["receipt"].(archimedesExchangeReceipt)
	if !ok || !receipt.MapDestroyed || receipt.TranslationEntryCount != 2 {
		t.Fatalf("receipt=%#v details=%#v", details["receipt"], details)
	}
	responseRows, ok := details["responses"].([]archimedesResponseRecord)
	if !ok || len(responseRows) != 1 {
		t.Fatalf("responses=%#v", details["responses"])
	}
	rawResponse, err := os.ReadFile(responseRows[0].BodyPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawResponse), "Jane Doe") || strings.Contains(string(rawResponse), "Acme") {
		t.Fatalf("fake provider received or persisted raw identity: %s", rawResponse)
	}
	result, err := os.ReadFile(exchange.DurableResultPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result), "Jane Doe") || !strings.Contains(string(result), "Acme") {
		t.Fatalf("local result was not rehydrated: %s", result)
	}
	retention, err := os.ReadFile(exchange.RetentionMetadataPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(retention), `"translation_map_destroyed": true`) {
		t.Fatalf("retention metadata omitted map destruction: %s", retention)
	}
}

func TestKernel30PrivacyGateRedactsOneAndBlocksTwo(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL30PRIVACY", "expire"))
	defer d.Close()
	redactedOrder := createKernel30ArchimedesOrder(t, d, archimedesMissive{
		Question:       "Ask alice@example.com for the decision.",
		DesiredProduct: "answer",
	})
	if _, err := d.mercuryRound(context.Background()); err != nil {
		t.Fatal(err)
	}
	redactedExchange := archimedesExchangeByOrder(t, d, redactedOrder.OrderID)
	if redactedExchange.State != archimedesStateCompleted {
		t.Fatalf("one-finding exchange=%+v", redactedExchange)
	}
	var action, gateState string
	if err := d.db.QueryRow(`SELECT action, state FROM archimedes_privacy_incidents WHERE exchange_id = ?`, redactedExchange.ExchangeID).Scan(&action, &gateState); err != nil {
		t.Fatal(err)
	}
	if action != "redacted" || gateState != "recorded" {
		t.Fatalf("privacy incident action=%s state=%s", action, gateState)
	}
	result, err := os.ReadFile(redactedExchange.DurableResultPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(result), "alice@example.com") || !strings.Contains(string(result), "<REDACTED>") {
		t.Fatalf("single privacy finding was not redacted: %s", result)
	}

	blockedOrder := createKernel30ArchimedesOrder(t, d, archimedesMissive{
		Question:       "Contact alice@example.com at +1 415-555-1212.",
		DesiredProduct: "answer",
	})
	if _, err := d.mercuryRound(context.Background()); err != nil {
		t.Fatal(err)
	}
	blockedExchange := archimedesExchangeByOrder(t, d, blockedOrder.OrderID)
	if blockedExchange.State != archimedesStateBlocked || blockedExchange.DurableResultPath != "" {
		t.Fatalf("two-finding exchange=%+v", blockedExchange)
	}
	loadedOrder, err := d.loadOrder(blockedOrder.OrderID)
	if err != nil {
		t.Fatal(err)
	}
	if loadedOrder.Status != orderStatusFailed || !strings.Contains(loadedOrder.LastError, "privacy gate") {
		t.Fatalf("blocked order=%+v", loadedOrder)
	}
}

func TestKernel30ReferralAndInspectionNeverAuthorizeTransport(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL30REFERRAL", "expire"))
	defer d.Close()
	request := httptest.NewRequest(http.MethodPost, "/v1/archimedes/referrals", strings.NewReader(`{"reason":"Apollo local boundary is insufficient","question":"Compare two generalized options","desired_product":"comparison","local_evidence_refs":["70 Pantheon Reports/example.md"]}`))
	request.Header.Set(controlTokenHeader, d.controlToken)
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("referral status=%d body=%s", response.Code, response.Body.String())
	}
	var referral archimedesReferral
	if err := json.Unmarshal(response.Body.Bytes(), &referral); err != nil {
		t.Fatal(err)
	}
	if referral.ReferralType != "needs_frontier_reasoning" || referral.AuthorityEffect != "none" || referral.State != archimedesStateReferred {
		t.Fatalf("referral=%+v", referral)
	}
	inspection, err := d.archimedesInspection()
	if err != nil {
		t.Fatal(err)
	}
	if inspection["live_transport"] != "disabled" || inspection["data_egress"] != false || inspection["provider"] != archimedesProviderFake {
		t.Fatalf("inspection=%#v", inspection)
	}
	page := httptest.NewRecorder()
	d.Handler().ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/v1/archimedes", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), referral.ReferralID) || !strings.Contains(page.Body.String(), "local_only_test_boundary") {
		t.Fatalf("inspection route=%d body=%s", page.Code, page.Body.String())
	}
}

func TestKernel30SystemInventoryShowsLocalOnlyArchimedes(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL30SYSTEM", "expire"))
	defer d.Close()
	inventory, err := d.systemInventory()
	if err != nil {
		t.Fatal(err)
	}
	componentFound := false
	for _, component := range inventory.Components {
		if component.ID == "archimedes" {
			componentFound = true
			if component.State != "local-only" || component.InspectionRoute != "/v1/archimedes" {
				t.Fatalf("Archimedes component=%+v", component)
			}
		}
	}
	if !componentFound {
		t.Fatal("system inventory omitted Archimedes")
	}
	if inventory.Work.Archimedes["data_egress"] != false {
		t.Fatalf("inventory Archimedes work=%#v", inventory.Work.Archimedes)
	}
}
