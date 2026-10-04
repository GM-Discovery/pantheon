package pantheon

import (
	"context"
	"testing"
	"time"
)

// Every audit row a single request produces must carry the same id, so one
// request can be replayed end to end. Before this, writeAudit minted an
// id per EVENT and audit_events had no column linking them -- a refused
// `do` wrote a started row and a refused row with unrelated ids and no way
// to group them.
func TestAuditRowsShareOneRequestID(t *testing.T) {
	d := doTestDaemon(t)
	ctx := withRequestID(context.Background())

	requestID := requestIDFrom(ctx)
	if requestID == "" {
		t.Fatal("withRequestID must stamp an id")
	}

	// The shape handleDo produces: authority recorded, then the outcome.
	if _, err := d.auditRequiredCtx(ctx, "POST /v1/do", "in", "notify", "m", "started", "", "authorized", time.Millisecond); err != nil {
		t.Fatal(err)
	}
	d.auditCtx(ctx, "POST /v1/do", "in", "notify", "m", "completed", "sent", "", time.Millisecond)

	var rows int
	if err := d.db.QueryRow(`SELECT count(*) FROM audit_events WHERE request_id = ?`, requestID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 2 {
		t.Fatalf("rows under request id = %d, want 2 -- the whole point is grouping", rows)
	}

	// Distinct event ids: grouping must not come at the cost of per-event
	// identity, since an audit id is what error messages hand the user.
	var distinct int
	if err := d.db.QueryRow(`SELECT count(DISTINCT id) FROM audit_events WHERE request_id = ?`, requestID).Scan(&distinct); err != nil {
		t.Fatal(err)
	}
	if distinct != 2 {
		t.Fatalf("distinct event ids = %d, want 2", distinct)
	}
}

// Two requests must not blend. A correlation id that collides is worse than
// none, because it invents a causal link that did not exist.
func TestRequestIDsDoNotCollideAcrossRequests(t *testing.T) {
	d := doTestDaemon(t)

	first := withRequestID(context.Background())
	second := withRequestID(context.Background())
	if requestIDFrom(first) == requestIDFrom(second) {
		t.Fatal("two requests must get different ids")
	}

	d.auditCtx(first, "POST /v1/do", "a", "notify", "m", "completed", "", "", 0)
	d.auditCtx(second, "POST /v1/do", "b", "notify", "m", "completed", "", "", 0)

	for _, ctx := range []context.Context{first, second} {
		var rows int
		if err := d.db.QueryRow(`SELECT count(*) FROM audit_events WHERE request_id = ?`, requestIDFrom(ctx)).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		if rows != 1 {
			t.Fatalf("request %s has %d rows, want 1", requestIDFrom(ctx), rows)
		}
	}
}

// An unstamped context must still produce an audit row. Losing the audit
// entirely would be a far worse failure than losing the correlation, and
// the 80-odd existing d.audit call sites are all in this position until
// they are converted.
func TestAuditWithoutCorrelationStillRecords(t *testing.T) {
	d := doTestDaemon(t)

	if id := d.auditCtx(context.Background(), "POST /v1/do", "in", "notify", "m", "completed", "", "", 0); id == "" {
		t.Fatal("an unstamped context must still write an audit row")
	}
	if requestIDFrom(context.Background()) != "" {
		t.Fatal("an unstamped context must report no id, not a fabricated one")
	}
	if requestIDFrom(nil) != "" {
		t.Fatal("a nil context must not panic")
	}

	// Legacy path keeps working and leaves request_id NULL rather than "".
	if _, err := d.writeAudit("POST /v1/legacy", "in", "cap", "m", "completed", "", "", 0); err != nil {
		t.Fatal(err)
	}
	var nulls int
	if err := d.db.QueryRow(`SELECT count(*) FROM audit_events WHERE request_id IS NULL`).Scan(&nulls); err != nil {
		t.Fatal(err)
	}
	if nulls < 1 {
		t.Fatal("uncorrelated rows should store NULL, so they are distinguishable from a correlated-but-empty id")
	}
}
