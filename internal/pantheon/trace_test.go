package pantheon

import (
	"context"
	"testing"
	"time"
)

// A correlation id is only worth having if a request can be read back as an
// ordered sequence with the breaking stage identified. These tests pin that.
func TestTraceForRequestOrdersStagesAndNamesTheBreak(t *testing.T) {
	d := doTestDaemon(t)
	ctx := withRequestID(context.Background())
	id := requestIDFrom(ctx)

	// The real shape of a refused `do`: authority recorded, then refusal.
	if _, err := d.auditRequiredCtx(ctx, "POST /v1/do", "in", "hermes", "m", "started", "", "routing", time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond) // distinct occurred_at for ordering
	d.auditCtx(ctx, "POST /v1/do", "in", "hermes", "m", "refused", "", "automatic execution is not authorized", 2*time.Millisecond)

	trace, err := d.traceForRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.Stages) != 2 {
		t.Fatalf("stages = %d, want 2", len(trace.Stages))
	}
	if trace.Stages[0].Outcome != "started" || trace.Stages[1].Outcome != "refused" {
		t.Fatalf("stages out of order: %s then %s", trace.Stages[0].Outcome, trace.Stages[1].Outcome)
	}
	if trace.Outcome != "refused" {
		t.Fatalf("terminal outcome = %q, want refused", trace.Outcome)
	}
	if trace.Broke == "" {
		t.Fatal("a refused request must name where it broke -- that is the whole point")
	}
	if trace.Stages[1].Error == "" {
		t.Fatal("the breaking stage must carry its error")
	}
}

// Tracing must not leak the payloads audit_events stores. Pantheon's stages
// carry email bodies and vault text; "where did this break" does not need
// them, and a routine debugging command must not surface them.
func TestTraceOmitsPayloads(t *testing.T) {
	d := doTestDaemon(t)
	ctx := withRequestID(context.Background())
	secret := "private vault content that must not appear in a trace"

	d.auditCtx(ctx, "POST /v1/do", secret, "reason", "m", "completed", secret, "", 0)

	trace, err := d.traceForRequest(requestIDFrom(ctx))
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.Stages) != 1 {
		t.Fatalf("stages = %d, want 1", len(trace.Stages))
	}
	stage := trace.Stages[0]
	for field, value := range map[string]string{
		"endpoint": stage.Endpoint, "capability": stage.Capability,
		"model": stage.Model, "outcome": stage.Outcome, "error": stage.Error,
	} {
		if value == secret {
			t.Fatalf("payload leaked into trace field %q", field)
		}
	}
}

func TestTraceRejectsEmptyAndUnknownRequests(t *testing.T) {
	d := doTestDaemon(t)
	if _, err := d.traceForRequest("  "); err == nil {
		t.Fatal("an empty request id must be an error, not an empty trace")
	}
	trace, err := d.traceForRequest("req-does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.Stages) != 0 || trace.Outcome != "unknown" {
		t.Fatalf("unknown request should yield no stages and outcome unknown, got %#v", trace)
	}
}

// You almost never know the request id at the moment something breaks, so
// the failure list is what makes tracing reachable at all.
func TestListRecentFailuresSurfacesIDsNewestFirst(t *testing.T) {
	d := doTestDaemon(t)

	// Written first because pre-correlation events are, by definition, the
	// older ones -- this also keeps the newest-first assertion meaningful.
	if _, err := d.writeAudit("POST /v1/legacy", "in", "cap", "m", "failed", "", "old failure", 0); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)

	healthy := withRequestID(context.Background())
	d.auditCtx(healthy, "POST /v1/do", "in", "reason", "m", "completed", "fine", "", 0)

	time.Sleep(2 * time.Millisecond)
	broken := withRequestID(context.Background())
	d.auditCtx(broken, "POST /v1/do", "in", "hermes", "m", "refused", "", "not authorized", 0)

	time.Sleep(2 * time.Millisecond)
	blocked := withRequestID(context.Background())
	d.auditCtx(blocked, "POST /v1/notify", "in", "notify", "m", "blocked", "", "Pantheon is paused", 0)

	failures, err := d.listRecentFailures(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(failures) != 3 {
		t.Fatalf("failures = %d, want 3 (refused, blocked, legacy failed -- never the completed one)", len(failures))
	}
	for _, f := range failures {
		if f.Outcome == "completed" {
			t.Fatal("a completed event is not a failure")
		}
	}
	// Newest first.
	if failures[0].Outcome != "blocked" {
		t.Fatalf("newest failure = %q, want blocked", failures[0].Outcome)
	}

	var untraceable int
	for _, f := range failures {
		if !f.Traceable {
			untraceable++
			if f.RequestID != "" {
				t.Fatal("an untraceable failure must not claim a request id")
			}
		}
	}
	if untraceable != 1 {
		t.Fatalf("untraceable failures = %d, want 1 -- old events must be listed, not hidden", untraceable)
	}
}
