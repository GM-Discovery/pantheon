# Kernel 15 — Janus: Temporal Thresholds & Deferred Work

## Objective

Implement **Janus**, Pantheon’s deterministic temporal-transition daemon.

Janus exists at the threshold between states.

He does not reason about the past. He does not predict the future. He does not schedule recurring work. He does not decide whether an action is wise or authorized.

Janus observes that a deterministic temporal boundary has been crossed and performs the transition prescribed for that boundary.

Conceptually:

```text
Father Time
    │
    │ chronology
    ▼
  Janus
    │
    │ deterministic transition
    ▼
new state
```

Janus is intentionally narrow.

> **Janus never owns the thing before the threshold or after the threshold. His domain is the crossing.**

Kernel 15 must establish temporal state transitions without turning Pantheon into an autonomous scheduler or creating surprise replay after pause.

---

## A. Actual predecessor

Kernel 15 must begin from repository truth.

At implementation start record:

- current branch;
- exact predecessor commit;
- `git status --short`;
- Kernel 14 implementation/reportback state;
- installed runtime identity;
- current Father Time behavior relevant to temporal state.

Do not assume Kernel 14’s specification equals its final implementation.

Read the Kernel 14 reportback before implementing Janus.

---

## B. Constitutional boundary

Kernel 15 preserves:

- the person remains sovereign;
- intelligence is not authority;
- deities possess no operational authority;
- Prometheus pause remains above autonomous effects;
- Kernel 13 remains the authority source;
- Seshat owns package definitions;
- Mercury carries valid executable work;
- Father Time owns chronology.

Janus introduces the additional invariant:

> **The passage of time may change the state of an operation without granting new authority to perform it.**

A temporal transition is not authorization.

---

## C. Janus is deterministic

Janus is a daemon.

No language model is required.

Janus may:

- inspect eligible temporally bounded state;
- consult Father Time;
- inspect temporal rules associated with a Seshat package;
- detect crossed thresholds;
- perform prescribed state transitions;
- record transition evidence;
- mark work ready, missed, expired, or otherwise temporally transitioned;
- create bounded resurfacing occurrences where the applicable policy explicitly requires it.

Janus may not:

- decide whether old information remains important;
- reason over Vault prose;
- invent a schedule;
- grant authority;
- expand authority;
- seal an order because time has passed;
- issue Go;
- invoke an external tool;
- rewrite substantive missives;
- infer a temporal policy that Seshat did not define.

---

## D. Father Time boundary

Father Time answers:

> **What time is it, and what chronology has occurred?**

Janus answers:

> **Given that time, has this defined threshold been crossed?**

Do not duplicate Father Time.

Janus should use the configured Father Time source/location and preserve the chronology/evidence needed to explain a transition.

The conceptual distinction is:

```text
Father Time:
    09:00 has arrived.

Janus:
    This item's 09:00 threshold has therefore been crossed.
```

Father Time does not need to know what that crossing means.

Janus applies the deterministic transition contract.

---

## E. Seshat owns temporal definitions

Janus does not invent temporal behavior for package classes.

Seshat’s package contract is the appropriate source for package-specific temporal rules.

A package may eventually define metadata resembling:

```text
temporal:
  mode: scheduled
  missed_policy: require_review
```

or:

```text
temporal:
  mode: recurring_output
  missed_policy: backfill
```

or:

```text
temporal:
  mode: state_maintenance
  missed_policy: recompute
```

The exact schema should follow Kernel 14’s actual package manifest architecture.

Kernel 15 may extend Seshat’s manifest schema minimally where temporal metadata is required.

Do not create a second package-policy registry inside Janus.

---

## F. Temporal contracts

A Seshat package may describe deterministic temporal behavior including, where applicable:

```text
not_before
not_after
review_until
expires_at
missed_policy
pause_policy
```

Not every package needs every field.

Prefer the smallest set that supports demonstrated behavior.

Temporal metadata should distinguish between:

### Absolute instance data

For example:

```text
send_after: 2026-08-20T09:00
```

### Package-class policy

For example:

```text
missed_policy: require_review
```

Seshat defines the valid contract.

Janus applies it.

---

## G. Core transitions

Kernel 15 should implement only transitions supported by real Pantheon use cases.

Likely examples include:

```text
scheduled → ready

reviewable → ready

scheduled → missed

reviewable → missed

missed → expired

missed → resurface

missed → backfill

missed → recompute

missed → require_review

active → expired
```

The implementation need not use these exact state names if existing durable state provides a cleaner representation.

Do not build a giant state machine for hypothetical future package types.

Every implemented transition should have an identifiable current or near-term use.

---

## H. The transition record

Every Janus transition must be inspectable.

At minimum preserve:

```text
transition_id
subject_id
package_code
from_state
to_state
threshold
threshold_at
observed_at
policy
father_time_reference
reason
```

Where relevant also preserve:

```text
source_order_id
source_proposal_id
source_grant_id
resurfaced_occurrence_id
```

The record should answer:

> What changed?

> When did it cross?

> Which rule caused that transition?

> What original thing did this transition belong to?

Janus does not need to explain whether the rule was philosophically wise.

---

## I. No resurrection of old orders

A terminally expired operation remains part of history.

Janus must not revive the same old operation and execute it late merely because Pantheon resumed.

For example:

```text
EMAIL:SEND
scheduled Tuesday
Pantheon paused Tuesday
resume Friday
```

must not become:

```text
send Friday
```

unless a fresh lawful process explicitly creates a new operation.

The original Tuesday operation remains missed/expired according to its temporal contract.

---

## J. Resurfacing

Some expired or missed matters may still be semantically important.

Janus must not determine that importance.

Where Seshat’s temporal contract specifies `resurface`, Janus may create a **new occurrence** describing the temporal fact.

For example:

```text
Scheduled EMAIL:SEND order <id>
crossed its execution threshold while Pantheon was paused
and expired without execution.
```

That new occurrence enters the normal cognition path:

```text
Janus
  ↓
new occurrence
  ↓
Intake
  ↓
Present
  ↓
Past
  ↓
Apollo
```

Apollo may then determine that:

- the matter is obsolete;
- the person should be notified;
- a new message should be proposed;
- circumstances have changed;
- no further action is needed.

The old order itself remains historical.

---

## K. Resurfacing is not Nostalgia

Janus resurfacing is triggered by a **specific deterministic temporal transition**.

It is not general archival rediscovery.

Janus does not crawl old Vault material looking for forgotten relevance.

That possible future role belongs elsewhere.

Current architectural candidate:

**Nostalgia** — a fast, shallow model operating under deterministic containment that may later patrol dormant Vault material and surface plausible forgotten connections.

Nostalgia is explicitly out of scope for Kernel 15.

The distinction is:

```text
Janus:
"This known thing crossed a known time boundary."

Nostalgia:
"This forgotten old thing might matter again."
```

---

## L. Backfill

A package may define missed work as eligible for `backfill`.

Backfill does not mean blindly replaying every missed execution.

Janus should create or mark the appropriate bounded backfill state according to Seshat’s contract.

Example:

```text
DAILY_BRIEF
Monday missed
Tuesday missed
Wednesday resume
```

A sensible package contract might allow:

```text
one catch-up briefing
```

rather than:

```text
three historical brief executions
```

Janus applies the deterministic package policy.

Janus does not compose the catch-up briefing.

That remains reasoning/work for the normal Pantheon path.

---

## M. Recompute

Some missed work should evaluate **current state** instead of replaying historical operations.

For example:

```text
CALENDAR:MAINTAIN
```

may run periodically.

If Pantheon is paused for seven days, resume should not cause hundreds of missed historical maintenance operations.

A `recompute` policy means the historical scheduled instances are not replayed.

Instead, the appropriate current-state work is created or marked for fresh evaluation.

Conceptually:

```text
168 missed maintenance opportunities
        ↓
      Janus
        ↓
recompute current calendar state once
```

Janus does not perform the calendar reasoning or maintenance itself.

---

## N. Require review

Consequential external effects may use:

```text
missed_policy: require_review
```

If the intended execution moment passes without execution, the operation cannot become executable merely because Pantheon later resumes.

It enters a review-required state.

A fresh human or lawful Pantheon process must resolve it.

This is particularly appropriate for future capabilities such as:

- outbound email;
- external publishing;
- consequential calendar invitations;
- other time-sensitive communications.

Kernel 15 need not enable those capabilities to establish the rule.

---

## O. Expire

Some work simply stops mattering after its threshold.

For:

```text
missed_policy: expire
```

Janus records the transition and performs no further action.

Expiration:

- preserves history;
- does not delete the original item;
- does not create authority;
- does not automatically resurface;
- does not notify the person unless another explicit policy requires it.

---

## P. Orion boundary

**Orion owns recurring schedules.**

Janus does not.

Orion may eventually maintain declarations such as:

```text
DAILY_BRIEF
every day
07:00
```

or:

```text
CALENDAR:MAINTAIN
every hour
```

Orion determines that a recurring scheduled occurrence is due.

Father Time provides chronology.

Janus governs the temporal transition associated with that due instance.

Mercury later carries valid Gos associated with executable work.

Conceptually:

```text
      Orion
 recurring schedule
        │
        ▼
   due instance
        │
Father Time ──→ Janus
                 │
              transition
                 │
                 ▼
              workflow
```

Kernel 15 must not implement a generalized Orion scheduler merely because Janus needs test fixtures.

Use bounded deterministic test instances.

---

## Q. Mercury boundary

Janus never carries work to tools.

Janus never issues Go.

Janus may transition an authorized/scheduled item into a state from which another deterministic component may later request execution.

Mercury remains the courier.

Existing Go semantics remain controlling:

- exact seal binding;
- bounded validity;
- one attempt;
- consumption;
- invalidation under Prometheus pause;
- no automatic replay.

---

## R. Future boundary

Future is not yet implemented as the full authority-administration daemon.

Kernel 15 must prepare temporal state so Future can later answer:

> Is this operation both temporally eligible and currently authorized?

Janus answers only the temporal half.

Conceptually:

```text
Seshat:
What kind of thing is this?
What temporal rules apply?

Janus:
What has time done to it?

Future:
May it happen?

Mercury:
Carry it.
```

Do not implement Future inside Janus.

---

## S. Prometheus pause

Prometheus pause remains above autonomous operational effects.

While paused:

- Father Time continues chronology;
- Janus may continue recording deterministic threshold crossings where safe;
- no external operational effect occurs merely because Janus transitioned something;
- Mercury issues/carries no executable Go contrary to existing pause rules;
- queued consequential work must not silently accumulate into a replay bomb.

Resume means:

> Pantheon may operate again.

Resume does **not** mean:

> Perform everything that would have happened while paused.

Janus must apply the relevant missed-work policies instead.

---

## T. Vacation behavior

Kernel 15 should explicitly prove the safety property needed for extended pauses such as vacation.

Example:

```text
Day 1:
Pantheon paused.

Day 2:
EMAIL:SEND reaches intended execution time.

Day 3:
DAILY_BRIEF becomes due.

Day 4:
CALENDAR:MAINTAIN would ordinarily run.

Day 8:
Pantheon resumes.
```

Expected behavior may be:

```text
EMAIL:SEND
→ require_review

DAILY_BRIEF
→ backfill

CALENDAR:MAINTAIN
→ recompute
```

There must be no automatic burst of stale external effects on resume.

This behavior should be demonstrable without actually sending external email.

---

## U. Grants and temporal expiry

Kernel 13 already supports grant validity windows and deterministic expiration.

Do not duplicate grant authority semantics in Janus.

Where grant expiration already occurs correctly through Kernel 13’s authority machinery, preserve that implementation.

Janus may record or surface a temporal transition when useful, but Kernel 15 should not create competing truth about whether a grant is authorized.

Authority remains Kernel 13’s domain.

Time may invalidate authority.

Janus does not grant it.

---

## V. Review cushions

Kernel 15 should support the concept of a review window without hard-coding a universal review duration.

A package may eventually define or accept:

```text
review_until
```

or an equivalent deterministic boundary.

For a consequential scheduled operation:

```text
prepared
   ↓
reviewable
   ↓
review threshold crossed
   ↓
temporally eligible
```

The package contract, user-granted authority, or later scheduling machinery may determine the actual cushion.

Janus merely performs the defined transition.

Do not assume every scheduled operation needs review.

---

## W. Temporal policy vocabulary

Use a small explicit policy vocabulary.

At minimum, Kernel 15 should consider:

```text
expire
resurface
backfill
recompute
require_review
```

Only implement policies needed by the actual Kernel 15 proof.

Unknown policy values must fail visibly.

Janus must never guess the closest policy.

Seshat validation should reject unsupported temporal policy definitions where possible.

---

## X. Janus round

Implement one explicit deterministic primitive resembling:

```text
pantheon janus round
```

A Janus round should:

1. obtain current Father Time chronology;
2. inspect eligible temporal subjects;
3. identify crossed thresholds;
4. load applicable Seshat temporal contracts;
5. apply deterministic transitions;
6. create resurfacing occurrences where explicitly required;
7. preserve idempotency;
8. record transition evidence;
9. report what changed.

A Janus round must not invoke a reasoning model.

Repeated rounds at the same effective time must not duplicate transitions or resurfaced occurrences.

---

## Y. Inspection

Provide read-only surfaces consistent with existing Pantheon conventions.

At minimum, the person should be able to inspect:

```text
pantheon show janus
pantheon show transitions
pantheon show transition <id>
```

Equivalent repository-consistent names are acceptable.

Inspection should expose:

- Janus readiness;
- current Father Time reference;
- pending temporal subjects;
- recent transitions;
- transition policy;
- from/to state;
- threshold time;
- observation time;
- source package;
- source order/proposal where applicable;
- resurfaced occurrence where applicable;
- blocked/error states.

The person should be able to answer:

> **What crossed a threshold, and what did Janus do about it?**

---

## Z. Persistence

Temporal transitions must be durable.

Use existing state where practical.

Add only the smallest persistence needed to guarantee:

- transition idempotency;
- provenance;
- threshold history;
- linkage to source objects;
- linkage to resurfaced occurrences;
- inspection.

Do not create a parallel scheduler database.

Do not duplicate Father Time chronology.

Do not duplicate Seshat package definitions.

---

## AA. Transition idempotency

A particular subject crossing a particular threshold must transition once.

Repeated Janus rounds must not produce:

- duplicate expiry;
- duplicate resurfacing occurrences;
- duplicate backfill requests;
- duplicate recompute requests;
- duplicate review-required transitions.

Idempotency should be based on durable transition identity, not merely process memory.

---

## AB. Failure behavior

Janus failures should be deterministic and inspectable.

Examples:

```text
unknown temporal policy
missing package definition
invalid threshold
Father Time unavailable
malformed temporal metadata
source subject missing
```

Failures must not trigger:

- model reasoning;
- guessed timestamps;
- guessed policies;
- automatic execution;
- arbitrary shell fallback.

A failed transition remains visible for inspection.

---

## AC. Seshat extension

If Kernel 15 requires extending Seshat manifests with temporal metadata, do so through the package extension architecture Kernel 14 established.

The person’s plugin documentation must be updated accordingly.

A future plugin author should be able to learn:

- whether their package has temporal behavior;
- which temporal policies are supported;
- how to declare them;
- what Janus will do when a threshold is crossed.

Do not require editing Janus core merely to assign an existing supported temporal policy to a new package.

---

## AD. Built-in proof cases

Kernel 15 should prove at least three materially different temporal outcomes.

Prefer harmless synthetic/local cases.

### Case 1 — Expire

```text
REMINDER:test
threshold passes
→ expired
```

### Case 2 — Resurface

```text
TEST:RESURFACE
threshold passes
→ old item remains historical
→ exactly one new Intake occurrence created
```

### Case 3 — Missed consequential work

```text
TEST:REVIEW
scheduled while Prometheus paused
threshold passes
resume
→ require_review
→ no automatic execution
```

A fourth `recompute` or `backfill` proof is desirable if it can be implemented without broadening the kernel substantially.

---

## AE. Whole-system regression

Kernel 15 must preserve:

- Intake;
- Father Time;
- Present;
- Past/Weave;
- Apollo;
- Kernel 13 authority;
- Seshat package registry;
- Mercury;
- Prometheus pause.

Existing Kernel 11–14 tests must remain green.

Janus must not change the meaning of an existing Go, seal, grant, or package merely because it adds temporal state.

---

## AF. Automated verification

Before repository PASS:

```text
gofmt -w $(rg --files -g '*.go')
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel15 .
```

Focused Kernel 15 tests should demonstrate:

1. Janus uses Father Time rather than an independent semantic clock source.
2. Janus invokes no language model.
3. Seshat owns package temporal definitions.
4. A threshold transition occurs exactly once.
5. Repeated Janus rounds are idempotent.
6. `expire` preserves history and causes no execution.
7. `resurface` creates exactly one new Intake occurrence.
8. Resurfacing does not revive the old order.
9. `require_review` prevents late automatic execution.
10. Prometheus pause prevents autonomous effects.
11. Resume does not automatically replay missed work.
12. `backfill`, if implemented, creates bounded catch-up work rather than replaying every missed instance.
13. `recompute`, if implemented, requests current-state evaluation rather than historical replay.
14. Unknown temporal policies fail visibly.
15. Missing/malformed temporal metadata fails safely.
16. Temporal transitions do not create authority.
17. Janus cannot issue Go.
18. Janus cannot invoke a tool.
19. Existing Kernel 13 grant expiration remains authoritative.
20. Existing Seshat plugin extension remains functional.
21. Existing Kernel 11–14 regression tests remain green.

---

## AG. Live verification

Before claiming live PASS:

1. build the exact Kernel 15 candidate;
2. record candidate SHA-256;
3. install the exact candidate to the runtime;
4. restart only necessary Pantheon services;
5. verify service health;
6. inspect Janus;
7. create/use a harmless bounded temporal test subject;
8. cross or simulate its threshold using the repository’s safe testing mechanism;
9. run Janus;
10. prove exactly one transition;
11. run Janus again and prove idempotency;
12. prove one resurfacing path reaches Intake without reviving the source;
13. prove one paused/missed consequential test path does not execute on resume;
14. verify Seshat inspection remains healthy;
15. verify authority inspection remains healthy;
16. record exact live evidence.

Do not alter the system clock merely to make the test convenient if a safer injected/test threshold mechanism exists.

Do not send consequential external communications during verification.

---

## AH. Human documentation

Document Janus for future operators and plugin authors.

The documentation should explain:

- what Janus owns;
- what Father Time owns;
- what Orion will own;
- what Seshat owns;
- what Future will own;
- what Mercury owns;
- supported temporal policies;
- how package manifests declare temporal behavior;
- how pause affects thresholds;
- what resume does;
- why missed work is not blindly replayed;
- how resurfacing differs from resurrection;
- how to inspect transitions.

Keep the explanation concrete.

---

## AI. Non-goals

Kernel 15 does not:

- implement Orion;
- own recurring schedules;
- implement Future;
- administer authority;
- make authorization decisions;
- seal proposed work;
- issue Go;
- invoke tools;
- reason about semantic importance;
- crawl the Vault;
- implement Nostalgia;
- compose backfill content;
- perform calendar maintenance;
- send email;
- create a generalized workflow engine;
- replace Father Time;
- replace Seshat;
- replace Kernel 13 authority;
- replace Mercury;
- automatically replay work after pause;
- build the GUI.

---

## AJ. Reportback requirements

The Kernel 15 reportback must state:

- exact predecessor commit;
- starting repository state;
- Kernel 14 reportback state consumed;
- active documentation reconciled;
- final Janus responsibility boundary;
- Father Time integration;
- Seshat temporal-schema changes;
- supported temporal policies;
- transition persistence;
- transition identity/idempotency;
- resurfacing behavior;
- backfill behavior, if implemented;
- recompute behavior, if implemented;
- require-review behavior;
- Prometheus pause/resume behavior;
- Orion boundary preserved;
- Future boundary preserved;
- inspection surfaces;
- automated verification results;
- live transition proof;
- live resurfacing proof;
- live pause/no-replay proof;
- candidate and installed SHA-256;
- files changed;
- known limitations;
- inherited verification debt;
- report expiry conditions.

Do not claim a policy was implemented merely because its name exists in a schema.

---

## AK. Success condition

Kernel 15 PASS means:

> **Pantheon has a deterministic understanding of what the passage of time does to bounded work without confusing time, authority, scheduling, or reasoning.**

After PASS:

- Father Time can establish chronology.
- Seshat can describe temporal behavior for package classes.
- Janus can detect and record threshold crossings.
- Janus can deterministically transition work.
- missed work does not blindly replay;
- expired work remains historical;
- specifically configured expired/missed matters can resurface through Intake;
- Prometheus pause remains safe across long absences;
- resume does not unleash stale external effects;
- Future can later consume temporally valid work without having to invent temporal semantics.

---

## AL. Next safe successor

If Kernel 15 passes, the next safe successor is:

**Kernel 16 — Future Authority Administration**

Future shall consume Apollo’s proposed missives as normalized through Seshat, consider their Janus-managed temporal state where applicable, resolve them against Kernel 13 authority, and seal work already covered by valid authority.

Future shall refer insufficient authority toward Prometheus or the person.

Future shall not:

- reason about whether Apollo’s proposal is wise;
- encode plugin-specific packages;
- determine temporal transition policy;
- manufacture authority.

The resulting conceptual path will be:

```text
Present
   ↓
Past
   ↓
Apollo
   ↓
Seshat
   ↓
Janus, where temporally relevant
   ↓
Future
   ↓
seal
   ↓
Mercury
   ↓
tool
```

Orion remains reserved for recurring schedules.

Nostalgia remains a post-core enhancement for opportunistic resurfacing of dormant Vault material.

---

## AM. Post-Kernel-16 horizon

After Future is implemented, the next kernel should integrate and prove the complete organism rather than immediately adding Orion, Nostalgia, or another deity.

The intended checkpoint remains:

> **Can Pantheon now take useful work from observation through reasoning, packaging, temporal validity, authority, execution, and reportback while preserving the person’s agency?**

If not, fix the organism.

If yes, use it.

Let actual operational friction determine what deserves to exist next.