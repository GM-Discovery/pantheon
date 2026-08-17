# Kernel 16 Reportback — Future Authority Administration

## Status

**PASS — repository and live backend verified August 16, 2026.**

Kernel 16 adds Future as a deterministic authority-administration daemon. The
implementation was verified with the repository suite, installed as the exact
candidate below, and exercised through the local backend. Kernel 17 was not
started.

## Starting point and reconciliation

- Branch: `trunk`
- Exact predecessor: `6f3a9d8` (`feat: implement Kernel 15 Janus temporal transitions`)
- Kernel 15 reportback status: PASS, with Janus live verification on August 16,
  2026; its installed candidate was
  `c8b81291af5aaead0066dc84c75fb6a6237868fc3ac0b98632fd4afebab9242a`.
- Starting working tree also contained user-supplied untracked roadmap and
  Kernel 17 documents. Those files were preserved and are not part of this
  Kernel 16 commit.
- Kernel 15's Janus state and Seshat temporal contracts remain the source of
  temporal eligibility. Future does not replace or rewrite them.

## Responsibility boundary

Future consumes a bounded proposal carrying Seshat package identity,
capability, scope, adapter, payload, constraints, Apollo provenance, and an
optional Janus subject. It validates those references and resolves authority
using the existing Kernel 13 grant schema and Mercury sealing path.

Future does not:

- run a model or perform planning/reasoning;
- interpret Apollo's recommendations into tool calls;
- create a second grant registry or cryptographic authority system;
- issue Go, invoke tools, or execute its own recommendations;
- invent temporal policy or bypass Janus;
- treat Apollo text, a pending child grant, or a permission option as
  authorization.

The corrected orchestration contract is preserved: Present establishes the
source, Past and Weave provide context, and a future-supplied context contract
may eventually frame the missive before Mercury carries it to Apollo. Apollo's
structured result is evidence addressed to Future. Kernel 16 persists and
returns that evidence boundary but does not implement Future's later active
state engine.

## Durable state and outcomes

Future persists `future_proposals` and `future_authority_requests`. Proposal
records preserve package identity, requested operation, source report/missive,
Janus subject, authority request/grant, sealed order, state, and error.

- Active Future authority that contains the exact operation produces one
  immutable sealed Mercury order with `origin: future`, `requested_by: future`,
  standing-grant provenance, and the original payload and constraints.
- A matching delegable Prometheus grant produces one pending Future child grant
  and one durable `referred_prometheus` request. The child cannot seal an order
  until Prometheus approves it.
- No delegable parent produces one durable `needs_person` request. The human
  ruling path remains unexecuted.
- Seshat mismatch/unavailability is visible and bounded.
- A temporal package is blocked unless Janus reports `ready` or `active` for
  the matching subject. Future consumes Janus state and does not determine the
  policy.
- Pause defers Future sealing. Resume alone does not replay the proposal; a
  later explicit Future round is required. Sealed proposals are excluded from
  subsequent rounds.

## Surfaces

```text
pantheon future round
pantheon show future
pantheon show future proposals
pantheon show future proposal <proposal-id>
```

These correspond to the local Future round, proposal creation, inspection, and
proposal-detail endpoints. Round and proposal creation require the existing
local control token. Inspection is read-only.

## Automated verification

Focused Kernel 16 tests cover:

- exact Future-held grant matching and one sealed order with Future provenance;
- repeated-round idempotency and exclusion of sealed proposals;
- pause deferral and the absence of automatic replay on resume;
- Prometheus referral, pending-child non-authority, approval, and later seal;
- durable person referral when no delegable Prometheus authority exists;
- Janus temporal blocking before eligibility;
- existing Mercury sealing and Present dispatch boundaries through the
  repository suite.

Successful verification commands:

```text
gofmt -w $(rg --files -g '*.go')
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel16 .
```

## Live verification

The exact Kernel 16 candidate was built with SHA-256:

```text
79f07d16f14c4f998f5dd70352d462fc7dec094fb2f3fa6ae4657ba59df0a45d
```

The candidate was installed at `/home/brick/pantheon-runtime/bin/pantheon`,
with the pre-Kernel-16 binary preserved in `/tmp`. Only `pantheon.service` was
restarted. Live proof covered a harmless Future-authorized proposal, one sealed
order and Mercury result; insufficient authority with a delegable Prometheus
parent; person referral outside that parent; Janus temporal blocking; and
pause/resume without automatic replay. Exact IDs and command results will be
recorded below.

The final candidate was installed and restarted at 20:27:16 PDT. The user
service reported `active`, `ExecMainStatus=0`, PID `1736547`, and the running
`/proc/1736547/exe` hash matched the candidate above. Final status showed
`paused=false`, localhost listen address, and the expected local configuration.
The temporary pre-Kernel-16 runtime binary remains recoverable at
`/tmp/pantheon-before-kernel16-20260816`.

Live evidence collected August 16, 2026:

- A Future-held one-use grant for `notify.send` and the existing Seshat
  `NOTIFICATION`/`local-notify` package sealed proposal
  `proposal-kernel16-live-notification` exactly once. Future returned
  `sealed: 1`; order `order-b7c3d88f3659d2bfaf14dcbd188247d8` carried
  `origin: future`, `requested_by: future`, standing-grant provenance, and
  completed through Mercury with `dispatched: 1`, `completed: 1`.
- A second Future round examined zero work and a second Mercury round examined
  zero orders for that completed proposal, proving no replay. The final
  candidate repeated the same proof with proposal
  `proposal-kernel16-live-final`, order
  `order-f22286279b1deed291e1416499150505`, seal digest
  `sha256:92eb821417308543e07114f6a7b8992db50663101c59448121cff5c52f146a52`,
  and one dispatch/completion; the following Mercury round dispatched and
  completed zero orders.
- A nested scope under a temporary delegable Prometheus grant produced
  `referred_prometheus` for proposal
  `proposal-kernel16-live-referral-nested` and pending child grant
  `grant-future-1786935924274272636`. The child remained non-authorizing and
  was revoked after evidence collection. A sibling request outside the parent
  scope produced `needs_person` for `proposal-kernel16-live-person`; no
  notification was delivered.
- A future-dated Janus subject `janus-kernel16-live-blocked` bound proposal
  `proposal-kernel16-live-temporal`; Future returned
  `temporally_blocked: 1` while Janus remained `scheduled`. The temporary
  temporal plugin was moved to `/tmp/pantheon-kernel16-live-temporal-20260816.json`
  and is not registered in the final runtime.
- With the canonical pause command
  `pantheon prometheus protocol --pause`, a bounded proposal was deferred with
  `sealed: 0`. Resume alone did not dispatch it; one explicit Future round then
  sealed and one Mercury round completed order
  `order-a6d7b894ceba2521336b71b181ffd85c`. The temporary authority fixtures
  were revoked after proof, while proposal/order/report evidence remains
  durable.
- Final inspection showed only the persistent `core` Seshat plugin and the
  original `NOTIFICATION` and `EMAIL:DRAFT` packages. Prometheus ended resumed.

The live authority fixtures were deliberately temporary and recoverable proof
state; all were revoked after their bounded local tests. No email, internet,
or other consequential external operation was performed.

## Known boundary

Kernel 16 does not implement Future's active forward-state engine, Apollo
reasoning, tool-call proposal generation, recurring scheduling, or Kernel 17
whole-animal integration. Those remain later work. A structured Apollo result
is evidence, not an execution authorization.
