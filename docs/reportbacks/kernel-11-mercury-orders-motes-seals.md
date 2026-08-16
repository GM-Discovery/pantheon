# Kernel 11 Reportback — Mercury Orders, Motes, and Seals

## Status

Repository implementation and automated verification are complete. **Live deployment was not performed in this pass.** Kernel 10’s live Whisper verification debt remains open.

## A. Actual predecessor

- Branch: `trunk`.
- Exact predecessor commit: `f15c5b2` (`reconcile kernels 8 through 10`).
- Starting state was dirty because the Kernel 11 specification had been added and the tracked tests were temporarily present under an untracked `testing/` directory; the test files were restored to their tracked root layout without content changes.
- Inherited Kernel 10 status: repository-verified; live Present/Whisper deployment verification remains pending.

## B. Constitutional status

No constitutional amendment.

Pantheon remains the authority. Mercury carries orders, Father Time timestamps Go Commands, and Prometheus pause remains above execution.

## C. Mercury implementation

Mercury is implemented as a durable daemon role. Hermes/Terra are optional flexible labels and readiness is reported from the configured local binary; Kernel 11 does not install or start a flexible agent. The deterministic fallback is the active implementation for exact structured orders.

Flexible interpretation is required for free-form requests that do not identify a recipient, capability, and inspectable scope. Those requests remain `needs_interpretation`. Exact structured orders, authority checks, child scope checks, sealing, Go issuance, pause handling, Present dispatch, and result recording do not require a model.

## D. Order schema and lifecycle

SQLite tables are `orders`, `order_events`, `order_go_commands`, and `order_attempts`. Orders retain request text, origin, requester, recipient, capability, scope, payload/constraint references, authority basis/reference, parent relation, expiry, seal, and supersession fields. Status is projected separately from sealed fields.

The implemented lifecycle is:

```text
person ingress → pending/structured evaluation → sealed or needs_* / blocked_unavailable
→ waiting_paused or ready → Go issued → one dispatch attempt → completed/failed/interrupted
```

`pantheon order cancel <id>` records a cancellation event and changes only projected status. Sealed data is not rewritten. A successor may use `supersedes_order_id`; the old order remains sealed and is projected `superseded`.

## E. Authority

- `person_direct`: only the local CLI control-token boundary can create this origin; requested-by is assigned by Pantheon rather than accepted from the HTTP body. A direct person order is sufficient and is not double-confirmed.
- `standing_authority`: the existing `capability_grants` table is consulted for an exact active capability/scope match at order evaluation.
- `parent_order`: a child must name the parent and exact parent seal digest; the child may only retain or narrow the recipient, capability, and scope.
- `needs_ruling`: structured work without current authority, or a child that cannot be proven inside its parent, remains visible and undispatched.
- `needs_interpretation`: incomplete/free-form order text remains durable without semantic guessing.
- `blocked_unavailable`: known authority exists but no explicit local recipient adapter exists.

The first adapter is exact `present` / `present.process` for `occurrence:<id>` scopes. No unrestricted shell execution was added.

## F. Seals

The canonical sealed serialization is a fixed-order JSON struct containing the immutable order identity, original request, origin/requester, recipient, capability, scope, payload and constraints, authority basis/reference, parent relation, purpose, expiry, and supersession reference. Its SHA-256 digest is stored as `sha256:<hex>` in `seal_digest`. Status, `sealed_at`, events, Go Commands, and attempts are not part of the digest, so lifecycle projection does not mutate the seal.

Cancellation and supersession append events and update projected status only.

## G. Go Commands

`order_go_commands` records:

```text
go_id, order_id, order_seal_digest, issued_at, authority_basis,
authority_ref, parent_go_id, valid_until, consumed_at, invalidated_at, status
```

`seal` means “this immutable order was authorized.” `Go` means “execute this sealed order now.” Go issuance uses the configured Father Time location and binds the exact seal digest. One issued Go creates one attempt and is consumed once; completed orders are skipped on later rounds.

## H. Pause

Person order submission remains available while paused. Structured orders can be sealed but are projected `waiting_paused`; Mercury issues no Go. Prometheus pause invalidates outstanding issued Go Commands. Resume only clears the pause; it does not dispatch. A later explicit `pantheon mercury round` reevaluates and may issue a fresh Go. An in-flight/outstanding Go is recorded as invalidated/interrupted rather than replayed automatically.

## I. Child orders

The automated suite demonstrates an equal exact occurrence scope as an in-scope child with `authority_basis=parent_order`. A child with `occurrence:x/..` under parent `occurrence:x` is refused as `needs_ruling`. Scope comparison is deliberately conservative and does not use vague semantic similarity.

## J. Grants

Kernel 11 uses the existing `capability_grants` schema and exact active capability/scope lookup; no parallel grant registry or schema extension was needed. Grants are consulted before sealing and again at the executable structured-order boundary for non-person-origin orders. Configuration/capability presence alone is not authority.

## K. Mercury rounds

```text
pantheon mercury round
```

is a control-token-protected one-round primitive. It inspects durable pending work, preserves interpretation/ruling states, enforces expiry and pause, validates seals and adapter availability, issues Go, dispatches exact Present occurrences, and records attempts/evidence. No recurring scheduler was installed. `mercury.round_on_cron_wake` documents the safe hook for an existing controlled wake; the exact integration point is to invoke this command once from that wake.

## L. Deterministic fallback

The Kernel 11 e2e test runs with no Hermes/Terra binary. A trusted structured person order for an exact Intake occurrence is sealed, receives a Father Time Go, invokes local Present, records one attempt and completion, and is not dispatched on a second round. A free-form request remains `needs_interpretation` instead of being guessed.

## M. Inspection

Implemented read-only surfaces are:

```text
pantheon show orders
pantheon show order <id>
pantheon show mercury
```

The JSON shapes expose order identity/status/authority/seal, and Mercury readiness/backlog including flexible state, deterministic fallback, paused backlog, ruling, interpretation, completed, and failed/unavailable counts.

## N. Tests

Repository verification for this report:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel11 .
```

Kernel 11 tests cover direct ingress anti-forgery, no second approval, immutable seal projection, one-Go/one-dispatch/idempotent rounds, pause queue and no automatic resume replay, `needs_interpretation`, `needs_ruling`, `blocked_unavailable`, exact grant authority, parent-child scope inheritance/refusal, and harmless Present end-to-end dispatch. Live runtime verification is intentionally not claimed.

Kernel 10 coverage remains accurately bounded: PDF extraction, DOCX structure, traversal and symlink escape, stale source identity, exact occurrence processing, filename collisions, idempotency, human-modified artifacts, mocked/local Whisper dispatch, and missing processor behavior are tested. Kernel 10 does not currently have focused tests for `present --new` end-to-end, video audio-only handling, or deterministic recent-source resolution; no claim is made for those gaps. Kernel 10’s pause/no-replay test covers Present, while Kernel 11 separately covers no replay after Mercury resume.

## O. Files changed

- `mercury.go`: durable schema, authority, sealing, Go Commands, rounds, dispatch, cancellation, inspection, and pause invalidation.
- `daemon.go`, `main.go`, `welcome.go`: routes, trusted CLI commands, inspection, and command discovery.
- `configuration.go`, `config/pantheon.example.json`, `config/README.md`, `docs/configuration.md`: Mercury configuration and safe wake documentation.
- `kernel11_test.go`: focused lifecycle and safety tests.
- `docs/reportbacks/kernel-11-mercury-orders-motes-seals.md`: this reportback.

## P. Risks

Free-form interpretation is intentionally limited without Hermes/Terra. Scope comparison is conservative rather than semantically rich. Existing grants have exact-match semantics and no expiry/remaining-use extension. Hermes/Terra readiness is observational only. Cron integration is a documented command hook, not an installed scheduler. Recipient adapters are currently narrow. Cancellation and interruption are durable best-effort state transitions; they do not promise process-level termination of an already-running local operation.

## Q. Expiry conditions

This report expires if order authority, seal serialization, Go validation, pause invalidation, recipient adapters, grant semantics, or the live deployment state changes. It also expires if Kernel 10’s outstanding Whisper verification is completed or materially changed; that result must be recorded separately.

## R. Next safe successor

**Kernel 12 — Apollo Deep Local Processing.** Kernel 12 shall consume sealed Mercury orders and return cited local reports without becoming the authority that acts on its own conclusions.
