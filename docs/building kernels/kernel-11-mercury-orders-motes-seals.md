# Kernel 11 — Mercury Orders, Motes, and Seals

**Status:** Proposed implementation order  
**Constitutional authority:** Constitution of Pantheon v0.2  
**Roadmap position:** Kernel 11  
**Committed predecessor:** `f15c5b2` — `reconcile kernels 8 through 10`  
**Expected successor:** Kernel 12 — Apollo Deep Local Processing

---

# 1. Purpose

Kernel 11 gives Pantheon a durable, inspectable order system.

After Kernel 11, Pantheon shall be able to answer:

> What was ordered?

> Who or what requested it?

> What authority permits it?

> What exact scope was authorized?

> Has Mercury sealed it?

> Has Father Time given it the Go Command?

> Was it dispatched, completed, interrupted, expired, cancelled, or left needing a ruling?

Mercury is the courier and operational vessel for Pantheon orders.

For the present implementation, **Hermes powered by Terra may embody Mercury's flexible agent behavior**.

Pantheon itself remains the authority boundary.

Hermes/Terra may interpret, route, and carry an order. It may not manufacture authority.

A deterministic Mercury fallback shall remain able to perform the core order lifecycle when Hermes/Terra is unavailable or unnecessary.

---

# 2. Why This Kernel Comes Now

Kernel 4 established persistent capability grants and the rule that a capability name does not itself grant authority.

Kernel 10 established verified Present missives and an evidence chain:

```text
Intake occurrence
    ↓
content identity
    ↓
verified Raw Markdown
    ↓
Present missive
```

Kernel 11 now adds:

```text
Present / Person / Agent
    ↓
ORDER
    ↓
authority evaluation
    ↓
SEAL
    ↓
Mercury round
    ↓
GO COMMAND
    ↓
dispatch
    ↓
result / evidence
```

Kernel 12 may then give Apollo work through this order system rather than inventing a separate execution path.

---

# 3. Constitutional Rule

Intelligence is not authority.

Possessing a model, tool, credential, capability, filesystem path, configuration entry, grant name, or proposed action does not itself authorize action.

Pantheon authorizes.

Mercury carries.

Prometheus may stop.

The person rules.

---

# 4. Current Mercury Implementation

Mercury is a role, not a permanently fixed implementation.

For Kernel 11:

```text
Mercury role
├── flexible carrier: Hermes / Terra
└── deterministic fallback: Pantheon kernel
```

Hermes/Terra may remain the normal flexible Mercury implementation if configured and available.

Do not create a second competing Mercury agent merely to satisfy the metaphor.

Do not rename Hermes or Terra.

The software should describe the role accurately:

> Hermes/Terra currently embodies Mercury's flexible courier behavior.

Later implementations may replace or supplement it without replacing the order contract.

---

# 5. Deterministic Mercury Fallback

Mercury's core functions shall not disappear merely because Hermes/Terra is unavailable.

The deterministic fallback must be able to:

1. read pending structured orders;
2. identify trusted person-origin orders;
3. consult active grants;
4. validate deterministic child-order scope inheritance;
5. determine `needs_ruling` when authority is absent;
6. create immutable seals for presently authorized orders;
7. issue Go Commands when execution is presently lawful;
8. dispatch orders to exact deterministic Pantheon handlers when a handler is known;
9. record results and evidence;
10. preserve pending work for a later Mercury round.

The deterministic fallback shall not semantically reinterpret ambiguous prose, infer a new capability, expand scope, invent child work, decide vague scopes are equivalent, bypass missing authority, or impersonate Apollo.

If flexible interpretation is required and Hermes/Terra is unavailable, the order may remain:

```text
needs_interpretation
```

This is distinct from:

```text
needs_ruling
```

`needs_interpretation` means Pantheon cannot deterministically structure the request yet.

`needs_ruling` means Pantheon understands the requested action but lacks present authority.

---

# 6. Orders

An order is a durable local record of requested work, not the execution itself.

At minimum an order contains:

```text
order_id
created_at
origin
requested_by
request_text
recipient
capability
scope
parameters / payload references
constraints
authority_basis
authority_ref, if any
parent_order_id, if any
status
expires_at, if any
sealed_at, if any
seal_digest, if any
supersedes_order_id, if any
```

Exact schema naming may follow repository conventions.

Avoid embedding large source bodies in the order table.

Prefer references to Present missives, occurrence IDs, Raw artifact paths, grant IDs, and other stable local evidence.

---

# 7. Order Origin

The system must distinguish order origin.

Useful origins include:

```text
person_direct
standing_authority
parent_order
present
apollo
future
system_maintenance
```

Only origins actually implemented should be emitted.

Do not mark an order as `person_direct` merely because a caller supplied that string.

Person origin must be established through Pantheon's trusted local human control boundary.

---

# 8. Trusted Human Ingress

A direct order from the person is sufficient authorization.

The person does not need to approve the same action twice.

Therefore:

```text
pantheon order "..."
```

when invoked through Pantheon's trusted local CLI/control-token boundary, creates a person-origin order.

Raw HTTP callers must not be able to forge:

```text
requested_by: person
authority_basis: person_direct
```

Reuse the established local control-token boundary or an equally strong existing trusted-local mechanism.

Do not add a second unnecessary password prompt.

---

# 9. Direct Human Authority

A direct instruction from the person is sufficient authority by default.

Mercury shall not respond to an ordinary direct order with:

> Please approve this order.

unless a future explicit policy identifies a class of actions requiring double confirmation.

No such double-confirmation class is introduced by Kernel 11.

A direct order may still be technically unavailable, blocked by Prometheus, outside an installed capability, malformed, or impossible to dispatch.

Those are execution conditions, not a lack of human authority.

---

# 10. Authority Hierarchy

Mercury evaluates authority in this order:

## 10.1 Person direct

A trusted direct instruction from the person.

This is sufficient authority for the requested scope.

## 10.2 Existing standing authority / grant

An active grant whose capability and scope authorize the proposed order.

## 10.3 Sealed parent-order inheritance

A child order may inherit authority from an authorized parent only where the child remains demonstrably within the parent's sealed scope.

## 10.4 Otherwise

```text
needs_ruling
```

No model may move an order from `needs_ruling` to authorized merely by arguing that the action is sensible.

---

# 11. Capability Is Not Authority

Preserve the Kernel 4 rule:

> A capability declaration does not grant authority.

Capability answers:

> Can this organ perform this class of action?

Authority answers:

> Is this action presently permitted?

Mercury requires both before dispatch.

---

# 12. Existing Capability Grants

Kernel 11 shall consume the existing `capability_grants` contract rather than create a parallel authorization registry.

The order ledger should reference the exact grant used as authority:

```text
authority_basis: grant
authority_ref: grant-...
```

If the existing grant schema requires a minimal extension for revocation, expiration, standing versus one-use semantics, or remaining uses, implement only what Kernel 11 actually requires.

Do not silently broaden old grants.

Do not infer new scopes from grant intent prose.

Where legacy grant semantics are ambiguous, preserve safety and report the ambiguity.

---

# 13. Scope

Every actionable order shall have a scope Pantheon can inspect.

Scope may use existing capability conventions, deterministic structured fields, or a documented canonical representation.

Examples:

```text
occurrence:intake-abc123
mailbox:work/folder:INBOX/max:20
vault:read
notification:local
```

Do not require one universal scope grammar if existing capabilities already have safe scope conventions.

The authority evaluator must not rely on vague natural-language similarity to prove scope inclusion.

---

# 14. Child Orders

A sealed parent order may create child orders.

A child order must contain:

```text
parent_order_id
parent_seal_digest
recipient
capability
scope
purpose
```

The child's authority may be inherited only when it is demonstrably inside the parent's authority.

Example parent:

> Prepare tomorrow's meeting brief.

Potentially in-scope children:

```text
read the named calendar event
retrieve related authorized email
retrieve relevant local Vault notes
```

Not automatically in scope:

```text
send email
delete a file
grant Internet access
publish a document
change Pantheon configuration
```

If deterministic scope inclusion cannot be proven:

```text
needs_ruling
```

Do not let an agent broaden a parent's authority by creating children.

---

# 15. Limited Inheritance

Child orders may narrow authority.

They may not widen it.

Conceptually:

```text
child capability ⊆ parent permitted capability set
child scope      ⊆ parent permitted scope
child constraints ≥ parent constraints
```

Where subset comparison is not mechanically meaningful, inheritance fails closed into:

```text
needs_ruling
```

Hermes/Terra may propose the relationship.

Pantheon decides whether the authority relation is mechanically valid.

---

# 16. Immutable Sealed Orders

Before sealing, an order may be structured or corrected.

After sealing, its actionable content is immutable.

The sealed portion includes at minimum:

```text
order_id
origin
request_text or canonical request reference
recipient
capability
scope
parameters / payload references
constraints
authority basis/reference
parent relation
expiry, if any
```

Create a canonical serialization and compute a SHA-256 seal digest.

Do not include mutable execution status in the seal digest.

---

# 17. Seal Meaning

A seal means:

> **This exact immutable order is authorized within this exact scope.**

It does not mean:

> Execute right now.

That distinction belongs to the Go Command.

---

# 18. Amendment and Supersession

A sealed order is never silently edited.

If the person or Pantheon needs to change it:

1. retain the original sealed order;
2. create a new successor order;
3. reference the original with `supersedes_order_id` or equivalent;
4. record the relationship in audit/order history.

The earlier order may become `superseded` as projected state, but its sealed record remains intact.

---

# 19. Cancellation

The person must be able to cancel a specific pending order without editing its sealed contents.

Provide a trusted local control such as:

```text
pantheon order cancel <order-id>
```

Cancellation writes a separate event/state transition.

It does not mutate the sealed order body.

A cancelled order receives no future Go Command.

If already executing, cancellation requests best-effort interruption where the target supports it and records the result.

Prometheus remains the universal emergency stop.

---

# 20. The Mote of Chron

The mote of Chron is **the Go Command**.

Do not turn it into a decorative bag of metadata.

Its semantic meaning is:

> **GO.**

It carries only the bound information needed to establish what is being told to proceed.

At minimum:

```text
go_id
order_id
order_seal_digest
issued_at
authority_basis/reference
parent_go_id, if applicable
valid_until, if explicitly required
```

The Go Command is not a secret credential.

It is an auditable, Father-Time-stamped authorization to dispatch a particular sealed order.

---

# 21. Father Time and the Go Command

All Go Commands are issued with Father Time.

Do not allow agents to invent their own authorization timestamps.

The sequence is:

```text
sealed order
    ↓
Mercury round
    ↓
present authority / validity check
    ↓
Father Time
    ↓
GO
```

A Go Command must bind to the exact current seal digest.

If the seal does not match:

```text
DO NOT DISPATCH
```

---

# 22. One Go, One Dispatch Attempt

By default, a Go Command authorizes one dispatch attempt.

It is not a reusable standing credential.

If an order requires a later legitimate dispatch attempt, Mercury may issue a fresh Go only after reevaluating authority, Prometheus state, expiry, cancellation, parent validity, and recipient/capability availability.

Standing authority belongs in grants, not reusable Go tokens.

---

# 23. Order Status

Use explicit lifecycle states.

Recommended projected states:

```text
pending
needs_interpretation
needs_ruling
sealed
waiting_paused
ready
dispatched
running
completed
failed
interrupted
expired
cancelled
superseded
blocked_unavailable
```

The exact set may be simplified if semantics remain clear.

Do not overload `failed` to mean missing authority, paused, missing implementation, or expired.

---

# 24. `needs_ruling`

When Mercury understands the proposed action but lacks current authority:

```text
needs_ruling
```

Expose:

```text
order_id
requested action
requester/origin
recipient
capability
scope
missing authority
why it was requested
parent order, if any
```

Kernel 11 does not need the mature Kernel 14 ruling queue.

For now, read-only order/Mercury inspection must make these visible.

---

# 25. `needs_interpretation`

When deterministic fallback cannot safely translate free-form text into a structured actionable order and Hermes/Terra is unavailable:

```text
needs_interpretation
```

This is not a ruling request.

The human already authorized the intent if it was person-direct.

Pantheon simply lacks safe executable structure.

When flexible Mercury becomes available, it may interpret the preserved request without requiring the person to repeat it.

---

# 26. Hermes/Terra Interpretation

When configured, Hermes/Terra may transform a free-form order into a structured execution proposal.

Example:

```text
"Process the latest recording."
```

may become:

```text
recipient: present
capability: present.process
scope: occurrence:intake-...
```

The original person request remains durable.

The structured proposal does not replace it.

Pantheon applies deterministic authority and scope checks before sealing.

Hermes/Terra does not stamp its own Go.

---

# 27. Privacy of Mercury Payloads

Mercury should normally carry references rather than unnecessary source bodies.

Prefer:

```text
occurrence_id
Present missive ID
Raw artifact path
grant ID
order ID
```

over copying complete transcripts, documents, or Vault contents into the order ledger or flexible courier context.

If Hermes/Terra requires request text to interpret an order, provide only the bounded order/request context needed.

Do not automatically dump the Vault into Mercury.

---

# 28. Order Creation CLI

Add:

```text
pantheon order "request"
```

It shall:

1. establish trusted person origin;
2. preserve the request verbatim;
3. create a durable order;
4. attempt safe structuring if available;
5. seal immediately when enough structured information and authority exist;
6. otherwise leave it pending for Mercury;
7. never require a second confirmation merely because the order came from the person.

Return the order ID.

---

# 29. Structured Order CLI

Provide an exact structured path for deterministic operation and scripting.

Conceptually:

```text
pantheon order   --recipient present   --capability present.process   --scope occurrence:intake-...   "Process this occurrence"
```

Exact syntax may follow current CLI style.

This path is important because deterministic fallback must work without a language model.

---

# 30. Present Missive Orders

Kernel 10 preserves original user requests in Present missives.

Kernel 11 may create or accept an order referencing a Present missive.

For a request such as:

```text
"I need a brief summary and all action items from the recording I just uploaded."
```

Kernel 10 may already have produced a verified transcript and Present missive with reasoning awaiting Apollo.

Kernel 11 can preserve that work as an order targeted to Apollo even though Kernel 12 has not yet made Apollo an order consumer.

Such an order may remain accurately blocked/pending.

Do not pretend Apollo execution exists before Kernel 12.

---

# 31. Minimal Mercury Round

Kernel 11 introduces:

```text
pantheon mercury round
```

A Mercury round shall:

1. inspect Prometheus state;
2. inspect pending orders;
3. apply expiration/cancellation/supersession state;
4. attempt interpretation through configured Hermes/Terra where needed and allowed;
5. fall back deterministically where possible;
6. evaluate authority;
7. mark missing authority `needs_ruling`;
8. seal authorized orders;
9. evaluate present executability;
10. issue Go Commands for ready orders;
11. dispatch exact supported recipients;
12. record outcome/evidence;
13. leave unresolved orders durable for a later round.

The round must be idempotent.

Running it twice must not dispatch a completed one-shot order twice.

---

# 32. Cron Wake Contract

Mercury makes the rounds when the configured cron/scheduled agent wakes.

Kernel 11 must implement the round primitive so Hermes cron can call it safely.

Preferred integration:

> At an existing Hermes/Terra scheduled wake under Pantheon control, invoke one Mercury round before or as part of the scheduled work cycle.

Do not create an aggressive new high-frequency polling service merely to satisfy this Kernel.

If there is no safe centralized Hermes cron-wake hook, provide `pantheon mercury round` and document the exact integration point required.

Kernel 15 will later expand work windows, richer recurring rounds, retention maintenance, and proactive checking.

Kernel 11 establishes the executable round contract.

---

# 33. Pause Semantics — Pending Orders

Prometheus pause stops execution.

It does not erase human intent.

A trusted person may submit an order while Pantheon is paused.

That order may be recorded and, if its authority is deterministic, may be sealed.

It receives **no Go Command while paused**.

Its status should indicate:

```text
waiting_paused
```

or equivalent.

After resume, Mercury does not immediately detonate a backlog.

The next explicit/cron Mercury round reevaluates the order and may issue a fresh Go if it remains valid.

---

# 34. Pending Is Not Replay

Preserve:

```text
pending unattempted order ≠ blocked execution replay
```

An order submitted while paused and never dispatched may be picked up later.

An execution attempt already blocked/interrupted by pause is not silently replayed after resume.

A fresh Go or successor/retry action is required according to order state.

---

# 35. Pause Semantics — Existing Go Commands

When Prometheus becomes paused, an outstanding Go Command not yet consumed must not remain valid for new dispatch.

On resume:

- retain the historical Go record;
- do not reuse it;
- reevaluate the sealed order;
- issue a new Go if the order remains valid.

The seal may remain valid.

The prior temporal Go does not.

This is why seals and Go Commands are distinct.

---

# 36. Pause During Execution

If an order is already executing when Prometheus pauses:

- stop/cancel where the target safely supports cancellation;
- otherwise prevent further Pantheon-controlled follow-on actions;
- mark the attempt `interrupted` or equally precise;
- record what did and did not complete;
- do not silently replay it after resume.

Later retry must be explicit through a fresh authorized order/attempt.

---

# 37. Missing Capability

Authority and capability availability are separate.

If the person directly orders something that is authorized but no current organ can perform:

```text
blocked_unavailable
```

Do not turn this into `needs_ruling`.

The person has already ruled.

Example:

> "Use OCR on this scanned PDF."

If no local OCR processor exists:

```text
authority: sufficient
capability: unavailable
status: blocked_unavailable
```

The machine should not ask the person to approve OCR twice.

---

# 38. Expiration

Orders may optionally contain:

```text
expires_at
```

If absent, do not invent a short arbitrary order expiry.

A pending human order may remain pending until completed, cancelled, superseded, or explicitly expired by its own terms.

Go Commands remain one-dispatch temporal artifacts even when the order has no expiry.

---

# 39. Recurrence

Kernel 11 may reserve recurrence fields but shall not implement a general recurrence engine.

Do not confuse:

```text
standing authority
```

with:

```text
standing order
```

A standing grant permits a class of future orders.

It does not itself require those orders to occur.

---

# 40. Dispatch Targets

Dispatch only to recipients with explicit local execution adapters.

Initial useful targets may include existing deterministic Pantheon surfaces such as Present, local notification, or bounded existing Mercury/email pilot operations if already safe.

Do not create speculative adapters just to make the list long.

Apollo may accept orders in Kernel 12.

---

# 41. Dispatch Contract

A recipient receives:

```text
sealed order reference
Go Command
bounded parameters/references
```

The recipient must be able to verify:

```text
order_id
seal digest
Go validity
```

before consequential execution.

Where legacy handlers cannot yet verify the full contract, route through a single Pantheon dispatcher that performs verification before invoking them.

Do not duplicate authority logic inside every organ.

---

# 42. Result Contract

Each dispatch attempt records:

```text
attempt_id
order_id
go_id
recipient
started_at
ended_at
status
result summary
evidence references
error, if any
```

Avoid storing huge result bodies in the order ledger.

Reference durable artifacts/reports where practical.

---

# 43. Audit

At minimum record:

```text
order created
order interpreted
authority evaluated
needs ruling
order sealed
Go issued
dispatch started
dispatch completed
dispatch failed
dispatch interrupted
order cancelled
order expired
order superseded
```

Audit records should reference stable IDs rather than duplicating full source content.

---

# 44. Order Ledger

Use local durable machine state, preferably existing SQLite.

Recommended conceptual tables:

```text
orders
order_events
order_go_commands
order_attempts
```

Equivalent schema is acceptable.

Avoid one giant mutable row if it makes sealed immutability difficult to prove.

The original sealed order must remain recoverable after later status changes.

---

# 45. Current State as Projection

Durable history is append-oriented.

Current order status may be stored/indexed for efficiency, but consequential transitions should remain reconstructable from order/event records.

Do not rewrite history to make an order appear as though it was always cancelled, failed, or superseded.

---

# 46. Read-Only Inspection

Add:

```text
pantheon show orders
pantheon show order <order-id>
pantheon show mercury
```

`show orders` should expose recent/current orders with:

```text
order ID
origin
short request
recipient
capability
status
authority basis
parent
created
sealed?
Go?
```

`show order` exposes the lifecycle without dumping private source contents unnecessarily.

`show mercury` exposes:

```text
Mercury flexible implementation: Hermes/Terra ready/unavailable
deterministic fallback: ready
last round
pending
needs interpretation
needs ruling
sealed waiting
ready
running
blocked
```

Do not build Panopticon.tower yet.

---

# 47. Hermes/Terra Readiness

Expose enough configuration/inspection to answer:

```text
Hermes binary found?
Terra agent configured?
Mercury flexible routing available?
deterministic fallback available?
```

Do not treat Hermes unavailability as Pantheon failure.

This state is degraded but functional:

```text
mercury:
  flexible: unavailable
  deterministic_fallback: ready
```

---

# 48. Configuration

Activate only configuration required for Mercury.

Conceptually:

```json
{
  "mercury": {
    "mode": "hermes",
    "agent": "Terra",
    "binary": "",
    "deterministic_fallback": true,
    "round_on_cron_wake": true
  }
}
```

Exact structure may follow Kernel 8 conventions.

Do not hard-code the development user's home path.

Do not silently rewrite legacy configuration.

Do not make Hermes availability mandatory for config validity when deterministic fallback is enabled.

---

# 49. Flexible Mercury Failure

If Hermes/Terra is absent, exits nonzero, times out, returns malformed structured output, proposes unknown capabilities, or proposes scope expansion:

1. retain the original order;
2. record flexible-courier failure;
3. attempt deterministic handling if possible;
4. otherwise use `needs_interpretation`, `needs_ruling`, or `blocked_unavailable` accurately.

Do not convert a model failure into authority.

---

# 50. No Hidden Model Dependency

The following must work without Hermes/Terra:

```text
create exact structured person order
inspect orders
evaluate direct-person authority
evaluate exact grant authority
validate deterministic child scope
seal order
issue Go
dispatch known deterministic recipient
record result
cancel pending order
run Mercury round
```

Natural-language interpretation may degrade when flexible Mercury is absent.

Authority and safety may not.

---

# 51. Existing Direct Commands

Kernel 11 shall not break:

```text
pantheon present ...
pantheon intake scan
pantheon prometheus protocol --pause
```

Do not require every legacy command to be converted into an order in this Kernel.

Later kernels may consolidate direct surfaces behind Mercury once the order system proves reliable.

---

# 52. Prometheus Authority

Prometheus remains above ordinary dispatch.

No Go Command overrides:

```text
Prometheus paused
Prometheus hard-stopped
Prometheus revoked capability boundary
```

Mercury may retain orders while paused.

Mercury may not argue with Prometheus.

The person retains direct Prometheus control independent of Mercury, Hermes, Terra, Apollo, or reasoning.

---

# 53. No Second Approval

Acceptance tests must explicitly verify:

1. person submits a valid structured order;
2. authority basis becomes `person_direct`;
3. Mercury seals it without asking for another approval;
4. when unpaused and executable, Mercury issues Go;
5. the order dispatches.

No hidden confirmation round is permitted.

---

# 54. Tests — Order Identity and Immutability

Verify:

1. order IDs are stable and unique;
2. person request text is preserved verbatim;
3. canonical seal serialization is deterministic;
4. seal digest changes if any sealed actionable field changes;
5. status changes do not change seal digest;
6. sealed actionable fields cannot be silently mutated;
7. supersession preserves old order;
8. cancellation preserves sealed body.

---

# 55. Tests — Human Authority

Verify:

9. trusted local CLI can create `person_direct`;
10. raw unauthenticated HTTP cannot forge person origin;
11. direct person order does not become `needs_ruling`;
12. no second approval is required;
13. missing capability becomes `blocked_unavailable`, not `needs_ruling`;
14. person order submitted while paused is retained.

---

# 56. Tests — Grants

Verify:

15. capability existence alone is not sufficient authority;
16. matching active grant authorizes matching scope;
17. mismatched scope does not authorize;
18. revoked/expired grant does not authorize where those semantics exist;
19. grant ID used for authority is recorded;
20. legacy ambiguous grant behavior is safe and documented.

---

# 57. Tests — Child Orders

Verify:

21. narrowed child scope can inherit;
22. broader child scope cannot inherit;
23. new capability outside parent scope becomes `needs_ruling`;
24. child preserves parent order/seal reference;
25. cancelled/expired parent prevents a pending child from receiving a new Go unless another authority basis exists.

---

# 58. Tests — Go Commands

Verify:

26. only sealed authorized orders receive Go;
27. Go binds to exact seal digest;
28. Go uses Father Time timestamp;
29. one Go authorizes one dispatch attempt;
30. consumed Go is not reused;
31. mismatched seal/Go refuses dispatch;
32. cancelled/expired order receives no new Go.

---

# 59. Tests — Pause

Verify:

33. paused Pantheon issues no Go;
34. person may submit a pending order while paused;
35. pending unattempted order survives pause;
36. resume alone does not dispatch backlog;
37. next Mercury round reevaluates pending order;
38. pre-pause unused Go cannot be reused after pause;
39. interrupted attempt does not auto-replay;
40. read-only order/Mercury inspection remains available while paused.

---

# 60. Tests — Deterministic Fallback

Verify:

41. exact structured order can complete with Hermes unavailable;
42. direct-person authority evaluation works without a model;
43. exact grant evaluation works without a model;
44. deterministic child-scope validation works without a model;
45. ambiguous prose becomes `needs_interpretation`, not guessed execution;
46. malformed Hermes/Terra output cannot expand scope;
47. flexible Mercury failure leaves order durable.

---

# 61. Tests — Round Idempotency

Verify:

48. repeated Mercury rounds do not redispatch completed one-shot orders;
49. a sealed waiting order does not receive duplicate simultaneous Go Commands;
50. completed attempt remains completed;
51. `needs_ruling` remains visible and is not recreated as duplicate orders;
52. `needs_interpretation` remains durable for later flexible handling.

---

# 62. Tests — Dispatch

Verify at least one harmless deterministic end-to-end order.

Preferred initial live-safe target:

```text
Present processing of an exact harmless existing occurrence
```

or another existing local non-destructive target.

Prove:

```text
person order
→ structured order
→ person_direct authority
→ seal
→ Go
→ dispatch
→ result
```

Do not use email send, deletion, or external effects as the first live test.

---

# 63. Build Verification

Before reportback:

```text
gofmt
go test ./...
go vet ./...
go build -o /tmp/pantheon-kernel11 .
```

Continue using explicit build output if repository layout prevents default output.

Do not conflate source build success with installed runtime deployment.

---

# 64. Live Verification

After repository review, deploy only with human authorization.

## A. Inspect Mercury

```text
pantheon show mercury
```

Confirm Hermes/Terra readiness and deterministic fallback readiness.

## B. Create harmless structured direct order

Confirm:

```text
origin: person_direct
authority: sufficient
sealed: yes
needs_ruling: no
```

## C. Run Mercury round

```text
pantheon mercury round
```

Confirm Go issuance and one dispatch.

## D. Run again

Confirm completed one-shot order is not repeated.

## E. Pause queue test

```text
pantheon prometheus protocol --pause
```

Submit another harmless order.

Confirm it is recorded but gets no Go.

Resume:

```text
pantheon prometheus protocol --resume
```

Confirm nothing executes merely because resume occurred.

Then:

```text
pantheon mercury round
```

Confirm Mercury reevaluates and dispatches once.

---

# 65. Kernel 10 Verification Debt

Do not silently mark Kernel 10's Whisper path live-verified during Kernel 11 unless that smoke test is actually performed.

Kernel 11 may use verified repository contracts and harmless deterministic local material.

Keep outstanding live Whisper verification explicit.

Do not make Mercury depend on Whisper merely to prove Kernel 11.

---

# 66. Acceptance Criteria

Kernel 11 is a PASS only when:

1. durable order identity exists;
2. trusted direct human origin cannot be forged through ordinary raw HTTP;
3. direct person instruction is sufficient authority;
4. no second approval is required;
5. capability remains distinct from authority;
6. existing grants are consulted at execution boundary;
7. child authority can narrow but not widen parent scope;
8. absent authority becomes `needs_ruling`;
9. absent safe interpretation becomes `needs_interpretation`;
10. sealed actionable order fields are immutable;
11. supersession preserves original sealed order;
12. cancellation preserves original sealed order;
13. seal digest is deterministic;
14. mote of Chron is implemented as the Go Command;
15. Go binds to exact sealed order;
16. Go is issued by Mercury/Father Time after current validity evaluation;
17. Go is one-dispatch by default;
18. Prometheus pause prevents Go issuance/dispatch;
19. pending unattempted orders may survive pause;
20. resume does not itself replay/dispatch them;
21. a later Mercury round may pick them up;
22. interrupted/blocked execution does not silently replay;
23. Hermes/Terra can embody flexible Mercury;
24. deterministic fallback performs core lifecycle without Hermes/Terra;
25. model failure cannot become authority;
26. minimal `pantheon mercury round` exists;
27. rounds are idempotent;
28. order/Mercury inspection exists;
29. at least one harmless order completes end-to-end through seal and Go;
30. repository predecessor remains `f15c5b2` unless explicitly committed during this Kernel.

---

# 67. Required Kernel 11 Reportback

Produce:

# Kernel 11 Reportback — Mercury Orders, Motes, and Seals

## A. Actual predecessor

Report branch, exact predecessor commit, clean/dirty starting state, and inherited Kernel 10 status.

## B. Constitutional status

Expected:

```text
No constitutional amendment.
```

If implementation requires changing who authorizes action, stop and report.

## C. Mercury implementation

Describe:

```text
role
Hermes/Terra flexible implementation
deterministic fallback
readiness detection
```

State exactly which behavior requires flexible interpretation and which does not.

## D. Order schema

Show actual durable fields and lifecycle.

## E. Authority

Show implementation of:

```text
person_direct
grant
parent inheritance
needs_ruling
needs_interpretation
blocked_unavailable
```

Explicitly prove direct person orders are not double-confirmed.

## F. Seals

Show canonical sealed fields, digest method, immutability enforcement, and supersession/cancellation handling.

## G. Go Commands

Show the actual mote-of-Chron record.

Explain:

```text
seal = authorized immutable order
Go   = execute this sealed order now
```

Show Father Time binding and one-dispatch behavior.

## H. Pause

Show order submission while paused, no Go while paused, resume without automatic dispatch, later round pickup, and interrupted/attempted no-replay behavior.

## I. Child orders

Show one in-scope child and one refused broader child.

## J. Grants

State how Kernel 4 grants are now consulted at real execution boundary and any schema changes required.

## K. Mercury rounds

Document:

```text
pantheon mercury round
```

and any Hermes cron-wake integration actually implemented.

Do not claim recurring scheduling that was not installed.

## L. Deterministic fallback

Demonstrate at least one core lifecycle with Hermes/Terra unavailable or mocked unavailable.

## M. Inspection

Show actual output shapes for:

```text
pantheon show orders
pantheon show order <id>
pantheon show mercury
```

## N. Tests

List exact automated commands and results.

Separate source tests from live runtime verification.

## O. Files changed

List meaningful files and purpose.

## P. Risks

Include free-form interpretation limits, scope-comparison limitations, legacy grant semantics, Hermes/Terra availability, cron-wake integration limits, recipient adapter maturity, and interruption/cancellation limitations.

## Q. Expiry conditions

Identify future changes that invalidate this report.

## R. Next safe successor

**Kernel 12 — Apollo Deep Local Processing.**

Kernel 12 shall consume the Mercury order system rather than inventing a new authority path.

---

# 68. Stop Conditions

Codex shall stop and report rather than improvise if:

- direct human orders cannot be distinguished from forged caller claims;
- implementing Mercury requires granting Hermes/Terra authority independent of Pantheon;
- a model must decide whether authority exists;
- child scope can only be validated by vague semantic similarity;
- sealed orders must be mutated to make execution work;
- Prometheus pause would need to be weakened;
- an outstanding Go could remain executable across pause without reevaluation;
- existing grants must be silently broadened;
- deterministic fallback would require unrestricted shell execution;
- cron integration would require installing an aggressive new background loop;
- a live test requires email sending, deletion, external publication, or another unnecessary consequential side effect.

Do not solve these by weakening the order contract.

---

# 69. Successor Boundary

Kernel 10 established:

> **This is a faithful durable representation of what entered Pantheon.**

Kernel 11 establishes:

> **This is the order, this is its authority, this is its immutable seal, and this is the Go Command that permits execution now.**

Kernel 12 will establish:

> **Apollo can receive a sealed reasoning order, search deeply through authorized local knowledge, know when it lacks sufficient information, and return a cited report without becoming the authority that acts on its own conclusions.**

Mercury carries the order.

Father Time gives the Go.

Pantheon remains the place where authority lives.
