package pantheon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func futureTestPackage(code, capability, adapter string, temporal *seshatTemporal) seshatPackageSpec {
	return seshatPackageSpec{
		SubjectCode: code, Version: "1.0.0", Description: "Kernel 16 Future test package",
		Capability: capability, Adapter: adapter, RequiredFields: []string{"body"},
		PayloadRules: map[string]string{"body": "body"}, Returns: []string{"status"},
		PermissionClass: "bounded_test", PermissionShapes: []string{"once", "bounded_uses", "until_revoked"}, Temporal: temporal,
	}
}

func createFutureTestProposal(t *testing.T, d *daemon, code, scope, janusSubjectID string) futureProposal {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"occurrence_id": strings.TrimPrefix(scope, "occurrence:")})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := d.createFutureProposal(futureProposalRequest{
		ProposalID:     "proposal-" + strings.ToLower(strings.ReplaceAll(scope, ":", "-")),
		SourceReportID: "apollo-report-test", SourceMissive: "Future proposal from a cited report",
		PackageCode: code, Capability: "present.process", Scope: scope, Adapter: "present",
		Payload: payload, Purpose: "bounded Kernel 16 test", JanusSubjectID: janusSubjectID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return proposal
}

func TestKernel16FutureSealsExactlyOnceAndDefersPausedWork(t *testing.T) {
	d := kernel15Daemon(t, futureTestPackage("FUTURE:PRESENT", "present.process", "present", nil))
	defer d.Close()
	d.capabilities = capabilityConfig{Capabilities: []capability{{Name: "present.process", Type: "deterministic", Risk: "low"}}}
	kernel13Grant(t, d, capabilityGrant{ID: "future-authority", Holder: "future", Capability: "present.process", Scope: "occurrence:allowed", IntentSummary: "bounded Future test authority", HumanApproved: true})
	allowed := createFutureTestProposal(t, d, "FUTURE:PRESENT", "occurrence:allowed", "")
	first, err := d.futureRound(context.Background())
	if err != nil || first.Sealed != 1 || first.Failed != 0 {
		t.Fatalf("first Future round=%#v err=%v", first, err)
	}
	sealed, err := d.loadFutureProposal(allowed.ProposalID)
	if err != nil || sealed.State != futureStateSealed || sealed.OrderID == "" || sealed.AuthorityGrantID != "future-authority" {
		t.Fatalf("sealed Future proposal=%#v err=%v", sealed, err)
	}
	order, err := d.loadOrder(sealed.OrderID)
	if err != nil || order.Status != orderStatusSealed || order.Origin != orderOriginFuture || order.AuthorityRef != "future-authority" {
		t.Fatalf("Future order=%#v err=%v", order, err)
	}
	second, err := d.futureRound(context.Background())
	if err != nil || second.Sealed != 0 || second.Examined != 0 {
		t.Fatalf("sealed proposal replayed: %#v err=%v", second, err)
	}

	kernel13Grant(t, d, capabilityGrant{ID: "future-paused-authority", Holder: "future", Capability: "present.process", Scope: "occurrence:paused", IntentSummary: "paused Future test authority", HumanApproved: true})
	pausedProposal := createFutureTestProposal(t, d, "FUTURE:PRESENT", "occurrence:paused", "")
	if err := d.setPaused(true); err != nil {
		t.Fatal(err)
	}
	pausedRound, err := d.futureRound(context.Background())
	if err != nil || pausedRound.Deferred != 1 || pausedRound.Sealed != 0 {
		t.Fatalf("paused Future round=%#v err=%v", pausedRound, err)
	}
	if err := d.setPaused(false); err != nil {
		t.Fatal(err)
	}
	resumed, err := d.loadFutureProposal(pausedProposal.ProposalID)
	if err != nil || resumed.State != futureStateAwaiting || resumed.OrderID != "" {
		t.Fatalf("resume replayed Future work: %#v err=%v", resumed, err)
	}
	afterResume, err := d.futureRound(context.Background())
	if err != nil || afterResume.Sealed != 1 {
		t.Fatalf("Future did not seal after explicit post-resume round=%#v err=%v", afterResume, err)
	}
	if _, err := d.mercuryRound(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestKernel16FutureAuthorizedLocalNotificationReachesMercury(t *testing.T) {
	d := kernel15Daemon(t, seshatPackageSpec{
		SubjectCode: "FUTURE:NOTICE", Version: "1.0.0", Description: "Kernel 16 local notification proof",
		Capability: "notify.send", Adapter: "local-notify", RequiredFields: []string{"message"},
		PayloadRules: map[string]string{"message": "body"}, Returns: []string{"status"},
		PermissionClass: "bounded_local_output", PermissionShapes: []string{"once"},
	})
	defer d.Close()
	d.capabilities = capabilityConfig{Capabilities: []capability{{Name: "notify.send", Type: "deterministic", Risk: "low"}}}
	helper := filepath.Join(t.TempDir(), "notify-helper")
	output := filepath.Join(t.TempDir(), "notification.txt")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nprintf '%s|%s|%s' \"$1\" \"$2\" \"$3\" > \""+output+"\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	oldNotify := notifyCommand
	notifyCommand = helper
	defer func() { notifyCommand = oldNotify }()
	kernel13Grant(t, d, capabilityGrant{ID: "future-notify-authority", Holder: "future", Capability: "notify.send", Scope: "notification:kernel16", IntentSummary: "bounded Future notification proof", HumanApproved: true})
	payload := json.RawMessage(`{"message":"Kernel 16 local proof","title":"Pantheon test"}`)
	proposal, err := d.createFutureProposal(futureProposalRequest{ProposalID: "proposal-notification", SourceReportID: "apollo-report-notification", SourceMissive: "Deliver this bounded local proof notification.", PackageCode: "FUTURE:NOTICE", Capability: "notify.send", Scope: "notification:kernel16", Adapter: "local-notify", Payload: payload, Purpose: "bounded local proof"})
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := d.futureRound(context.Background())
	if err != nil || sealed.Sealed != 1 {
		t.Fatalf("Future notification round=%#v err=%v", sealed, err)
	}
	result, err := d.mercuryRound(context.Background())
	if err != nil || result.Dispatched != 1 || result.Completed != 1 {
		t.Fatalf("Mercury notification round=%#v err=%v", result, err)
	}
	loaded, err := d.loadFutureProposal(proposal.ProposalID)
	if err != nil {
		t.Fatal(err)
	}
	order, err := d.loadOrder(loaded.OrderID)
	if err != nil || order.Status != orderStatusCompleted {
		t.Fatalf("completed Future notification order=%#v err=%v", order, err)
	}
	contents, err := os.ReadFile(output)
	if err != nil || string(contents) != "pantheon-alerts|Pantheon test|Kernel 16 local proof" {
		t.Fatalf("notification output=%q err=%v", contents, err)
	}
}

func TestKernel16FutureRefersPrometheusAndDoesNotTreatPendingGrantAsAuthority(t *testing.T) {
	d := kernel15Daemon(t, futureTestPackage("FUTURE:REFERRAL", "present.process", "present", nil))
	defer d.Close()
	d.capabilities = capabilityConfig{Capabilities: []capability{{Name: "present.process", Type: "deterministic", Risk: "low"}}}
	constraints := json.RawMessage(`{"allow_delegation":true}`)
	kernel13Grant(t, d, capabilityGrant{ID: "prometheus-parent", Holder: holderPrometheus, Capability: "present.process", Scope: "occurrence:referred", IntentSummary: "delegable Future test authority", HumanApproved: true, DelegationConstraints: constraints})
	proposal := createFutureTestProposal(t, d, "FUTURE:REFERRAL", "occurrence:referred", "")
	first, err := d.futureRound(context.Background())
	if err != nil || first.Referred != 1 || first.Sealed != 0 {
		t.Fatalf("referral Future round=%#v err=%v", first, err)
	}
	pending, err := d.loadFutureProposal(proposal.ProposalID)
	if err != nil || pending.State != futureStateReferredPrometheus || pending.AuthorityRequestID == "" {
		t.Fatalf("referral proposal=%#v err=%v", pending, err)
	}
	request, err := d.loadFutureAuthorityRequest(pending.AuthorityRequestID)
	if err != nil || request.PrometheusGrantID == "" {
		t.Fatalf("authority request=%#v err=%v", request, err)
	}
	child, err := d.loadCapabilityGrant(request.PrometheusGrantID)
	if err != nil || child.Status != grantStatusPending || child.Holder != "future" {
		t.Fatalf("pending Future child=%#v err=%v", child, err)
	}
	pendingOrder, err := d.createOrderWithOrigin(orderRequest{RequestText: "pending child must not seal", Recipient: "present", Capability: "present.process", Scope: "occurrence:referred", AuthorityGrantID: child.ID}, false, orderOriginFuture, "future")
	if err != nil || pendingOrder.Status == orderStatusSealed || pendingOrder.SealDigest != "" {
		t.Fatalf("pending Future grant authorized an order: %#v err=%v", pendingOrder, err)
	}
	if err := d.approveGrantAsPrometheus(child.ID); err != nil {
		t.Fatal(err)
	}
	second, err := d.futureRound(context.Background())
	if err != nil || second.Sealed != 1 {
		t.Fatalf("approved referral Future round=%#v err=%v", second, err)
	}
}

func TestKernel16FutureNeedsPersonAndHonorsJanusEligibility(t *testing.T) {
	noAuthority := kernel15Daemon(t, futureTestPackage("FUTURE:PERSON", "present.process", "present", nil))
	defer noAuthority.Close()
	noAuthority.capabilities = capabilityConfig{Capabilities: []capability{{Name: "present.process", Type: "deterministic", Risk: "low"}}}
	proposal := createFutureTestProposal(t, noAuthority, "FUTURE:PERSON", "occurrence:person", "")
	result, err := noAuthority.futureRound(context.Background())
	if err != nil || result.NeedsPerson != 1 || result.Sealed != 0 {
		t.Fatalf("needs-person Future round=%#v err=%v", result, err)
	}
	request, err := noAuthority.loadFutureProposal(proposal.ProposalID)
	if err != nil || request.State != futureStateNeedsPerson {
		t.Fatalf("needs-person proposal=%#v err=%v", request, err)
	}

	temporal := futureTestPackage("FUTURE:TEMPORAL", "present.process", "present", &seshatTemporal{Mode: "scheduled", MissedPolicy: "expire"})
	d := kernel15Daemon(t, temporal)
	defer d.Close()
	d.capabilities = capabilityConfig{Capabilities: []capability{{Name: "present.process", Type: "deterministic", Risk: "low"}}}
	kernel13Grant(t, d, capabilityGrant{ID: "future-temporal-authority", Holder: "future", Capability: "present.process", Scope: "occurrence:temporal", IntentSummary: "temporal Future test authority", HumanApproved: true})
	subject, err := d.createJanusSubject(janusSubjectRequest{SubjectID: "future-temporal-subject", PackageCode: "FUTURE:TEMPORAL", Threshold: time.Now().Add(time.Hour).Format(time.RFC3339Nano), ThresholdKind: janusThresholdNotBefore, RequestText: "Future temporal eligibility test"})
	if err != nil {
		t.Fatal(err)
	}
	temporalProposal := createFutureTestProposal(t, d, "FUTURE:TEMPORAL", "occurrence:temporal", subject.SubjectID)
	blocked, err := d.futureRound(context.Background())
	if err != nil || blocked.TemporallyBlocked != 1 || blocked.Sealed != 0 {
		t.Fatalf("temporally blocked Future round=%#v err=%v", blocked, err)
	}
	loaded, err := d.loadFutureProposal(temporalProposal.ProposalID)
	if err != nil || loaded.State != futureStateTemporallyBlocked {
		t.Fatalf("temporal proposal=%#v err=%v", loaded, err)
	}
}
