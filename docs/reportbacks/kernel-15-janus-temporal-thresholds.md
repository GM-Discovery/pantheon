# Kernel 15 Reportback — Janus Temporal Thresholds & Deferred Work

## Status

**PASS — repository and live backend verified August 16, 2026.**

Kernel 15 implements Janus as a deterministic temporal-transition daemon. The
exact candidate was rebuilt and installed after repository verification; live
transition evidence is recorded below. No consequential external communication
or tool operation was created during verification.

## Starting point

- Branch: `trunk`
- Exact predecessor commit: `a8bf3ac` (`feat: implement Kernel 14 Seshat package registry`)
- Starting branch state: `trunk...origin/trunk [ahead 11]`
- Starting working tree: clean except for the supplied, untracked Kernel 15
  specification, `docs/building kernels/Kernel 15 — Janus_ Temporal Thresholds & Deferred Work.md`.
- Kernel 14 reportback consumed: repository PASS August 16, 2026; live Kernel
  14 deployment intentionally not performed.
- Installed runtime at start: Kernel 13 binary
  `/home/brick/pantheon-runtime/bin/pantheon`, SHA-256
  `6e98e3d534cb11028405a5b1883432e541b5d4d279395f4da73386f9898fecf0`.
- Father Time behavior consumed: configured timezone/chronicle location,
  Intake chronology, and `fatherTimeNow`/`fatherTimeLocation`; Janus adds no
  independent chronology store.

## Responsibility boundary

```text
Father Time  → configured chronology and time reference
Seshat       → package temporal contract
Janus        → threshold crossing and deterministic state transition
Future       → later authority administration
Mercury      → seal/Go transport and execution boundary
Orion        → reserved recurring schedules
```

Janus does not reason, schedule recurring work, decide semantic importance,
grant authority, seal an order, issue Go, invoke tools, rewrite missives, or
replay stale work. A temporal transition never creates authority.

## Seshat temporal schema

Seshat package manifests now optionally support:

```json
"temporal": {
  "mode": "scheduled|reviewable|active",
  "missed_policy": "expire|resurface|backfill|recompute|require_review",
  "pause_policy": "hold|missed|require_review"
}
```

Seshat validates these values. Janus does not maintain a second package-policy
registry. Instance data remains separate: a bounded Janus subject supplies an
absolute RFC3339 threshold and one of `not_before`, `not_after`,
`review_until`, or `expires_at`.

The human extension guide, schema surface, starter template, and examples were
updated. The temporal example manifest is
`config/plugins/examples/janus-temporal.json`; it is not automatically loaded
as a registered runtime plugin.

## Supported transitions and persistence

SQLite state adds `janus_subjects` and `janus_transitions` beside the existing
Pantheon state. A subject can transition through:

- `scheduled`/`reviewable` + crossed `not_before` → `ready` when operating;
- crossed missed threshold → `missed`, then one policy result:
  `expired`, `resurfaced`, `backfill_pending`, `recompute_pending`, or
  `review_required`;
- malformed or unavailable temporal state → `blocked_error` with a visible
  error.

Each transition records its identity, subject, package, from/to states,
threshold, observed time, policy, Father Time reference, reason, and source
order/proposal/grant references where present. A unique durable identity on
subject, threshold, and from/to state makes repeated rounds idempotent.

`backfill_pending` and `recompute_pending` are bounded pending states only.
Janus does not compose catch-up content or perform current-state maintenance.

## Resurfacing and pause behavior

`resurface` writes one deterministic Markdown fact into the configured Intake
path, runs the existing Intake scan, and links the resulting new occurrence to
the Janus transition. The old subject remains `resurfaced`; it is never revived
or re-executed. Repeated rounds cannot create a second occurrence.

While Prometheus is paused, Janus may record safe internal transitions and
resurfacing, but no Mercury Go is issued and no adapter is invoked. The
`require_review` pause policy moves missed work to `review_required`; resume
does not replay it. Existing Kernel 13 grant expiration remains authoritative
and is not duplicated by Janus.

## Inspection and control surfaces

```text
pantheon janus round
pantheon show janus
pantheon show transitions
pantheon show transition <id>
```

The corresponding API surfaces are:

- `POST /v1/janus/round` — local control-token protected round;
- `GET /v1/janus` — readiness, Father Time reference, state counts, recent transitions;
- `GET /v1/janus/transitions` and `/v1/janus/transitions/{id}` — read-only evidence;
- `POST /v1/janus/subjects` — control-token protected bounded subject/proposal fixture.

## Automated verification

Focused Kernel 15 tests cover:

- Seshat temporal metadata round-trip and unsupported-policy rejection;
- Father Time-configured threshold parsing and deterministic round behavior;
- exact-once expiry and durable repeated-round idempotency;
- one-occurrence resurfacing without source resurrection;
- paused require-review behavior and no replay after resume;
- bounded backfill and recompute pending states;
- missing package/temporal state becoming inspectable `blocked_error`;
- inspection endpoints and control-token protection;
- no Go commands or tool execution from Janus;
- existing Kernel 11–14 behavior through the full repository suite.

Commands completed successfully:

```text
gofmt -w $(rg --files -g '*.go')
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel15 .
```

## Live verification

The exact Kernel 15 candidate was built with SHA-256:

```text
c8b81291af5aaead0066dc84c75fb6a6237868fc3ac0b98632fd4afebab9242a
```

The candidate was installed at `/home/brick/pantheon-runtime/bin/pantheon`,
and only `pantheon.service` was restarted. The installed SHA-256 matched the
candidate. The pre-Kernel-15 installed binary was preserved at
`/tmp/pantheon-before-kernel15-20260816`.

Live evidence on August 16, 2026:

- candidate installed and service active with `ExecMainStatus=0`; final clean
  restart PID `1705389`, `ActiveEnterTimestamp=Sun 2026-08-16 17:10:31 PDT`;
- `GET /v1/status` returned `paused=false`, localhost listen address, and the
  expected local model/configuration;
- the harmless manifest was registered at 17:00 through
  `pantheon plugin register`, returned `authority: none`, and exposed
  `EXAMPLE:REMINDER`, `EXAMPLE:RESURFACE`, and `EXAMPLE:REVIEW`;
- `pantheon show janus` reported `ready: true` and a Father Time reference;
- three bounded subjects were created at 17:00:19 with thresholds already
  crossed;
- while Prometheus was paused, `pantheon janus round` examined 3 subjects,
  performed 6 transitions, expired 1, resurfaced 1, and moved 1 to
  `review_required`, with zero failures;
- the live transition evidence was:
  `live-expire-20260816` → `expired`,
  `live-resurface-20260816` → `resurfaced`, and
  `live-review-20260816` → `review_required`;
- the resurfacing transition linked exactly one new Intake occurrence,
  `intake-3093ba862c6e47140f4a9a4bcbf423c4`;
- a second paused Janus round examined 0 subjects; after resume, another
  Janus round also examined 0 subjects, proving durable idempotency and no
  replay;
- Mercury inspection showed no failed/unavailable/ruling backlog, and Janus
  itself has no Go/tool issuance path; the live subjects created no new order
  or Go command;
- read-only authority inspection for
  `present.process` / `occurrence:kernel15-live-check` returned
  `allowed:false`, `capability_configured:false`;
- after evidence collection, the generated harmless Intake Markdown artifact
  was moved to `/tmp/pantheon-kernel15-live-resurface.md`, the registered test
  manifest was moved to `/tmp/pantheon-kernel15-live-test-manifest.json`, and
  the service was restarted again. Final live Seshat inspection showed only
  the persistent `core` plugin with 2 enabled packages; Janus transition
  history remained durable.

## Files changed

- `internal/pantheon/janus.go`
- `internal/pantheon/kernel15_test.go`
- `internal/pantheon/seshat.go`
- `internal/pantheon/daemon.go`
- `internal/pantheon/cli.go`
- `internal/pantheon/welcome.go`
- `config/plugins/manifest-template.json`
- `config/plugins/examples/janus-temporal.json`
- `docs/janus-temporal.md`
- `docs/seshat-plugin-extension.md`
- `docs/janus-temporal.md`
- `docs/software.md`
- `docs/README.md`
- this Kernel 15 specification

## Known limitations and inherited verification debt

Janus does not implement Orion, Future, Nostalgia, a generalized workflow
engine, recurring schedules, external adapters, semantic reasoning, authority
administration, tool execution, GUI surfaces, or automatic backfill/recompute
work. Live verification uses harmless local bounded subjects only. The Kernel
14 live deployment debt remains historical: Kernel 14 itself was repository
verified, while this candidate is the first later live backend identity.

## Report expiry and successor

This report expires if the installed binary, temporal manifest schema, Janus
SQLite state, pause behavior, or transition implementation changes without a
new verification record. The next safe successor is **Kernel 16 — Future
Authority Administration**, which must consume Janus state and Seshat packages
without moving temporal policy or adapter encoding into Future.
