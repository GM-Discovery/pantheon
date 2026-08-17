# Janus Temporal Boundaries

Janus is Pantheon’s deterministic threshold daemon. Father Time provides the
configured chronology and time zone; Janus applies the temporal rule declared
by a Seshat package to a known bounded subject. Janus does not schedule
recurring work, decide importance, administer authority, seal orders, issue Go,
invoke tools, or replay old work. Orion remains the reserved owner of recurring
schedules. Future remains the later authority administrator.

## Temporal package metadata

A Seshat manifest may add:

```json
"temporal": {
  "mode": "scheduled",
  "missed_policy": "require_review",
  "pause_policy": "missed"
}
```

`mode` is `scheduled`, `reviewable`, or `active`. Supported missed policies
are `expire`, `resurface`, `backfill`, `recompute`, and `require_review`.
`pause_policy` may be `hold`, `missed`, or `require_review`.

The manifest defines the package-class rule. Each bounded instance supplies its
own absolute threshold and threshold kind: `not_before`, `not_after`,
`review_until`, or `expires_at`.

## What a Janus round does

Run:

```text
pantheon janus round
pantheon show janus
pantheon show transitions
pantheon show transition <transition-id>
```

Janus records every transition with the subject, package, source references,
threshold, Father Time reference, policy, and reason. Repeating a round is
durably idempotent.

For a crossed `not_before` threshold while Pantheon is operating, a subject
becomes `ready`. If the threshold was missed, Janus first records `missed` and
then applies the package policy:

- `expire` → `expired`, preserving history;
- `resurface` → `resurfaced`, with exactly one new Markdown occurrence in
  Intake;
- `backfill` → `backfill_pending`, leaving catch-up composition to later work;
- `recompute` → `recompute_pending`, requesting current-state evaluation rather
  than replaying historical instances;
- `require_review` → `review_required`, never late-executable automatically.

The original subject remains historical. Resurfacing creates a new Intake
occurrence; it never resurrects the old order.

## Pause and resume

Father Time continues while Prometheus is paused. Janus may record safe
internal threshold facts and apply the declared missed-work policy, but it does
not cause external effects. Mercury remains responsible for Go and existing
pause invalidation rules. Resume does not replay missed work or create a burst
of stale effects.

For a bounded local verification subject, the control-protected API accepts:

```text
POST /v1/janus/subjects
{
  "package_code": "EXAMPLE:REVIEW",
  "threshold": "2026-08-16T09:00:00-07:00",
  "threshold_kind": "not_before",
  "request_text": "harmless bounded test"
}
```

Subject creation is only a durable temporal fixture/proposal boundary. It does
not create a grant, seal an order, issue Go, or invoke an adapter.

The example manifests in `config/plugins/examples/janus-temporal.json` show
expire, resurface, and require-review package contracts. They are examples and
are not loaded into the registered runtime automatically.
