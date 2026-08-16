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

func kernel13Grant(t *testing.T, d *daemon, grant capabilityGrant) {
	t.Helper()
	if err := initializeAuthoritySchema(d.db); err != nil {
		t.Fatal(err)
	}
	if grant.Holder == "" {
		grant.Holder = holderPrometheus
	}
	if grant.GrantedBy == "" {
		grant.GrantedBy = holderPerson
	}
	if grant.CreatedAt == "" {
		grant.CreatedAt = time.Now().Format(time.RFC3339Nano)
	}
	if grant.Status == "" {
		grant.Status = grantStatusActive
	}
	approved := 0
	if grant.HumanApproved {
		approved = 1
	}
	_, err := d.db.Exec(`INSERT INTO capability_grants (id, holder, capability, scope, intent_summary, diff, human_approved, granted_by, parent_grant_id, created_at, valid_from, valid_until, remaining_uses, status, delegation_constraints, revoked_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, grant.ID, grant.Holder, grant.Capability, grant.Scope, grant.IntentSummary, grant.Diff, approved, grant.GrantedBy, nullableString(grant.ParentGrantID), grant.CreatedAt, nullableString(grant.ValidFrom), nullableString(grant.ValidUntil), grant.RemainingUses, grant.Status, nullableJSON(grant.DelegationConstraints), nullableString(grant.RevokedAt), nullableString(grant.UpdatedAt))
	if err != nil {
		t.Fatal(err)
	}
}

func TestKernel13GrantLifecycleAndOneUseAuthorization(t *testing.T) {
	d := testKernel11Daemon(t, t.TempDir())
	d.capabilities = capabilityConfig{Capabilities: []capability{{Name: "present.process", Type: "deterministic", Risk: "low"}}}
	remaining := 1
	kernel13Grant(t, d, capabilityGrant{ID: "one-use", Capability: "present.process", Scope: "occurrence:one", IntentSummary: "one use", RemainingUses: &remaining})
	first, err := d.createOrder(orderRequest{RequestText: "first authorized operation", Recipient: "present", Capability: "present.process", Scope: "occurrence:one"}, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := d.createOrder(orderRequest{RequestText: "second authorized operation", Recipient: "present", Capability: "present.process", Scope: "occurrence:one"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if first.AuthorityBasis != orderOriginStanding || second.AuthorityBasis != orderOriginStanding {
		t.Fatalf("orders lost grant provenance: first=%#v second=%#v", first, second)
	}
	round, err := d.mercuryRound(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if round.Dispatched != 1 {
		t.Fatalf("one-use round=%#v", round)
	}
	finishedSecond, err := d.loadOrder(second.OrderID)
	if err != nil {
		t.Fatal(err)
	}
	if finishedSecond.Status != orderStatusRuling {
		t.Fatalf("second use status=%s error=%s", finishedSecond.Status, finishedSecond.LastError)
	}
	var uses int
	if err := d.db.QueryRow(`SELECT remaining_uses FROM capability_grants WHERE id = 'one-use'`).Scan(&uses); err != nil || uses != 0 {
		t.Fatalf("remaining uses=%d err=%v", uses, err)
	}
}

func TestKernel13PausedExpiredAndRevokedGrantsDoNotAuthorize(t *testing.T) {
	d := testKernel11Daemon(t, t.TempDir())
	d.capabilities = capabilityConfig{Capabilities: []capability{{Name: "present.process", Type: "deterministic", Risk: "low"}}}
	now := time.Now()
	kernel13Grant(t, d, capabilityGrant{ID: "paused", Capability: "present.process", Scope: "occurrence:paused", IntentSummary: "paused"})
	kernel13Grant(t, d, capabilityGrant{ID: "expired", Capability: "present.process", Scope: "occurrence:expired", IntentSummary: "expired", ValidUntil: now.Add(-time.Minute).Format(time.RFC3339Nano)})
	kernel13Grant(t, d, capabilityGrant{ID: "revoked", Capability: "present.process", Scope: "occurrence:revoked", IntentSummary: "revoked", Status: grantStatusRevoked})
	if err := d.setGrantStatus("paused", grantStatusPaused); err != nil {
		t.Fatal(err)
	}
	for _, scope := range []string{"occurrence:paused", "occurrence:expired", "occurrence:revoked"} {
		order, err := d.createOrder(orderRequest{RequestText: "must not authorize", Recipient: "present", Capability: "present.process", Scope: scope}, false)
		if err != nil {
			t.Fatal(err)
		}
		if order.Status != orderStatusRuling || order.AuthorityBasis != "" {
			t.Fatalf("scope=%s order=%#v", scope, order)
		}
	}
	var status string
	if err := d.db.QueryRow(`SELECT status FROM capability_grants WHERE id = 'expired'`).Scan(&status); err != nil || status != grantStatusExpired {
		t.Fatalf("expired grant status=%s err=%v", status, err)
	}
}

func TestKernel13DeterministicContainmentAndDelegation(t *testing.T) {
	if !capabilityWithin("calendar.maintain", "calendar.read") || capabilityWithin("email.read", "email.send") {
		t.Fatal("capability containment is not deterministic")
	}
	if !scopeWithin("calendar:personal", "calendar:event:123") || scopeWithin("calendar:event:123", "calendar:personal") || !scopeWithin("internet:domain:example.org", "internet:domain:sub.example.org") {
		t.Fatal("scope containment is not deterministic")
	}

	d := testKernel11Daemon(t, t.TempDir())
	d.capabilities = capabilityConfig{Capabilities: []capability{{Name: "calendar.read", Type: "deterministic", Risk: "low"}}}
	constraints, _ := json.Marshal(delegationConstraints{AllowDelegation: true})
	kernel13Grant(t, d, capabilityGrant{ID: "prometheus-parent", Holder: holderPrometheus, Capability: "calendar.maintain", Scope: "calendar:personal", IntentSummary: "bounded calendar authority", DelegationConstraints: constraints})
	childRequest := grantRequest{Capability: "calendar.read", Scope: "calendar:event:123", IntentSummary: "Future read", Holder: "future", ParentGrantID: "prometheus-parent"}
	req := newJSONRequest(t, http.MethodPost, "/v1/grants", childRequest)
	req.Header.Set(controlTokenHeader, d.controlToken)
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, req)
	if response.Code != http.StatusAccepted {
		t.Fatalf("delegated request code=%d body=%s", response.Code, response.Body.String())
	}
	var child capabilityGrant
	if err := json.NewDecoder(response.Body).Decode(&child); err != nil {
		t.Fatal(err)
	}
	if child.Status != grantStatusPending || child.ParentGrantID != "prometheus-parent" {
		t.Fatalf("child grant=%#v", child)
	}
	if err := d.approveGrantAsPrometheus(child.ID); err != nil {
		t.Fatal(err)
	}
	child, err := d.loadCapabilityGrant(child.ID)
	if err != nil || child.Status != grantStatusActive || child.GrantedBy != holderPrometheus {
		t.Fatalf("approved child=%#v err=%v", child, err)
	}
	chain, err := d.authorityChain(child.ID)
	if err != nil || len(chain) != 2 || chain[1].ID != "prometheus-parent" {
		t.Fatalf("authority chain=%#v err=%v", chain, err)
	}

	broaden := childRequest
	broaden.Scope = "calendar:event:999"
	broaden.Capability = "calendar.maintain"
	broaden.HumanApproved = true
	broadenReq := newJSONRequest(t, http.MethodPost, "/v1/grants", broaden)
	broadenReq.Header.Set(controlTokenHeader, d.controlToken)
	broadenResponse := httptest.NewRecorder()
	d.Handler().ServeHTTP(broadenResponse, broadenReq)
	if broadenResponse.Code != http.StatusBadRequest {
		t.Fatalf("widened delegated grant code=%d body=%s", broadenResponse.Code, broadenResponse.Body.String())
	}
	escalated := childRequest
	escalated.Scope = "calendar:other"
	escalated.HumanApproved = false
	escalatedReq := newJSONRequest(t, http.MethodPost, "/v1/grants", escalated)
	escalatedReq.Header.Set(controlTokenHeader, d.controlToken)
	escalatedResponse := httptest.NewRecorder()
	d.Handler().ServeHTTP(escalatedResponse, escalatedReq)
	if escalatedResponse.Code != http.StatusAccepted || !strings.Contains(escalatedResponse.Body.String(), `"status":"pending"`) {
		t.Fatalf("out-of-delegation request code=%d body=%s", escalatedResponse.Code, escalatedResponse.Body.String())
	}
	var pending capabilityGrant
	if err := json.NewDecoder(escalatedResponse.Body).Decode(&pending); err != nil {
		t.Fatal(err)
	}
	if err := d.approveGrantAsPrometheus(pending.ID); err == nil {
		t.Fatal("Prometheus approved authority outside its delegation")
	}
}

func TestKernel13DeitiesCannotGrantOrSealAndInspectionExposesBasis(t *testing.T) {
	d := testKernel11Daemon(t, t.TempDir())
	d.capabilities = capabilityConfig{Capabilities: []capability{{Name: "present.process", Type: "deterministic", Risk: "low"}}}
	deityRequest := newJSONRequest(t, http.MethodPost, "/v1/grants", grantRequest{Capability: "present.process", Scope: "occurrence:model", IntentSummary: "model grant", Holder: "apollo"})
	deityRequest.Header.Set(controlTokenHeader, d.controlToken)
	deityResponse := httptest.NewRecorder()
	d.Handler().ServeHTTP(deityResponse, deityRequest)
	if deityResponse.Code != http.StatusBadRequest {
		t.Fatalf("deity grant code=%d body=%s", deityResponse.Code, deityResponse.Body.String())
	}
	order, err := d.createOrder(orderRequest{RequestText: "model proposed operation", Recipient: "present", Capability: "present.process", Scope: "occurrence:model"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != orderStatusRuling || order.SealDigest != "" {
		t.Fatalf("model sealed unauthorized order=%#v", order)
	}

	kernel13Grant(t, d, capabilityGrant{ID: "inspectable", Capability: "present.process", Scope: "occurrence:inspect", IntentSummary: "inspect", DelegationConstraints: json.RawMessage(`{"allow_delegation":true}`)})
	check := httptest.NewRecorder()
	d.Handler().ServeHTTP(check, httptest.NewRequest(http.MethodGet, "/v1/authority/check?capability=present.process&scope=occurrence:inspect", nil))
	if check.Code != http.StatusOK || !strings.Contains(check.Body.String(), "inspectable") || !strings.Contains(check.Body.String(), "authority_chain") {
		t.Fatalf("authority inspection code=%d body=%s", check.Code, check.Body.String())
	}
	grantView := httptest.NewRecorder()
	d.Handler().ServeHTTP(grantView, httptest.NewRequest(http.MethodGet, "/v1/grants/inspectable", nil))
	if grantView.Code != http.StatusOK || !strings.Contains(grantView.Body.String(), "allow_delegation") {
		t.Fatalf("grant inspection code=%d body=%s", grantView.Code, grantView.Body.String())
	}
}

func TestKernel13LegacyGrantSchemaIsExtendedWithoutLosingRecords(t *testing.T) {
	d := testKernel11Daemon(t, t.TempDir())
	if _, err := d.db.Exec(`INSERT INTO capability_grants (id, capability, scope, intent_summary, human_approved, status, created_at) VALUES ('legacy', 'present.process', 'occurrence:legacy', 'legacy record', 1, 'active', '2026-08-15T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	grant, err := d.loadCapabilityGrant("legacy")
	if err != nil || grant.ID != "legacy" || grant.Holder != holderPrometheus || grant.GrantedBy != holderPerson {
		t.Fatalf("migrated legacy grant=%#v err=%v", grant, err)
	}
	if _, err := d.db.Exec(`SELECT holder, granted_by, parent_grant_id, remaining_uses FROM capability_grants WHERE id = 'legacy'`); err != nil {
		t.Fatal(err)
	}
}
