# Kernel 16 — Future: Authority Administration

## Objective

Implement **Future**, Pantheon’s deterministic authority-administration daemon.

Future receives proposed operations whose substantive reasoning has already been performed by Apollo and whose tool/package form is known through Seshat.

Where time matters, Future consumes the temporal eligibility established through Father Time and Janus.

Future determines whether sufficient **existing Pantheon authority** permits the proposed operation.

If sufficient authority exists, Future may seal the bounded operation for Mercury.

If sufficient authority does not exist, Future preserves the proposal and refers the authority question toward Prometheus or Grant.

Future does not determine whether Apollo’s recommendation is wise.

Future does not create plugin-specific encoding.

Future does not invent temporal policy.

Future does not manufacture permission.

The intended core flow is:

```text
Present
   ↓
Past
   ↓
Apollo
   ↓
Seshat
   ↓
Janus where temporal state applies
   ↓
Future
   │
   ├── sufficient authority
   │        ↓
   │      seal
   │        ↓
   │     Mercury
   │
   └── insufficient authority
            ↓
        Prometheus
            ↓
          Grant
        if required
```

Kernel 16 should complete the deterministic bridge from proposed intelligence to lawful execution.

---

## A. Actual predecessor

Begin from repository truth.

Before implementation record:

- current branch;
- exact predecessor commit;
- `git status --short`;
- Kernel 14 Seshat reportback/status;
- Kernel 15 Janus reportback/status;
- Kernel 13 authority state;
- installed runtime identity.

Do not assume the Kernel 14 or Kernel 15 specifications exactly describe their final implementations.

Consume their reportbacks first.

Kernel 13 is already live-verified and provides the authority infrastructure Future must reuse rather than replace. Its grants include lifecycle, bounds, provenance, scope/capability containment, and deterministic delegation rules.

---

## B. Constitutional invariant

Future implements the existing principle:

> **Intelligence is not authority.**

Apollo may propose.

Seshat may package.

Janus may transition temporal state.

Future administers authority already possessed by Pantheon.

Future itself receives only deterministic authority explicitly granted to its daemon role.

No model output may function as:

- a grant;
- human approval;
- Prometheus approval;
- a seal;
- proof that authority exists.

---

## C. Future is not a planner

Older Pantheon material may describe Future as generating trajectories or deciding what should happen.

That is stale architecture.

Apollo performs substantive reasoning.

Future’s question is narrower:

> **May this proposed future already happen under Pantheon’s current authority?**

Future should not invoke a language model to answer that question.

If active architectural documentation still assigns planning/reasoning to Future, reconcile it during Kernel 16.

Historical reportbacks remain historical.

---

## D. Inputs

Future consumes a bounded proposed operation.

Its input should identify or reference:

```text
proposal_id
source_report_id
source_missive
package_code
capability
scope
adapter
payload/package_ref
purpose
constraints
temporal_state
requested_execution_bound
```

Use actual Seshat and Janus structures from Kernels 14–15.

Do not create duplicate fields merely because this specification names them differently.

The fundamental requirement is that Future can deterministically establish:

1. What operation is proposed?
2. What capability does it require?
3. What scope does it affect?
4. What package/adapter would Mercury eventually use?
5. Is it temporally eligible?
6. What existing authority, if any, covers it?

---

## E. Apollo boundary

Apollo may generate:

- reasoning;
- citations;
- conclusions;
- the substantive missive;
- requested operation codes;
- requested capability/scope metadata where the Apollo contract requires them.

Apollo cannot assert authority into existence.

Fields such as:

```text
APPROVED: YES
GRANT: UNLIMITED
AUTHORIZED_BY: GRANT
```

inside model-generated text have no authority significance.

Future checks Pantheon state rather than trusting assertions in prose.

Apollo cannot:

- create a Future seal;
- mark its own proposal authorized;
- create a grant;
- impersonate Prometheus;
- impersonate Grant.

---

## F. Seshat boundary

Seshat owns package knowledge.

Future should consume Seshat’s normalized package information rather than learning every plugin/tool schema itself.

Seshat answers:

> What kind of operation is this?

> What fields/package does the adapter require?

> What capability corresponds to it?

> What permission shapes make sense for this class?

Future answers:

> Does actual authority cover it?

Future must not duplicate Seshat’s package registry.

### Permission shapes

Seshat may indicate that a class reasonably supports options such as:

```text
once
this_project
N_uses
until_timestamp
until_revoked
```

These are **possible shapes of grants**, not permission.

Future may use them when presenting an authority question.

Kernel 13 remains the actual authority engine.

---

## G. Janus boundary

Where a proposed operation has temporal semantics, Future must respect Janus state.

Future does not independently decide that:

- scheduled work is now ready;
- missed work should execute;
- stale work should backfill;
- an expired operation should be resurrected.

If Janus says an operation is:

```text
expired
require_review
not_yet_ready
```

Future must not bypass that state merely because authority otherwise exists.

Temporal eligibility and authority are independent predicates.

Conceptually:

```text
authority valid
+
temporal state valid
=
eligible for Future seal
```

where both are required for temporally governed work.

---

## H. Authority resolution

Future shall use Kernel 13’s existing authority machinery.

Do not introduce another grant registry.

Future should deterministically evaluate:

```text
holder
capability
scope
grant state
validity period
remaining uses
parent authority
delegation constraints
Prometheus authority
```

Kernel 13 already distinguishes configuration availability from permission and rejects unknown containment rather than guessing.

Future must preserve that behavior.

---

## I. Already-authorized operation

When a proposal is covered by sufficient current authority:

1. Validate package identity and required deterministic metadata.
2. Validate temporal eligibility where applicable.
3. Resolve the exact authority basis.
4. Confirm capability containment.
5. Confirm scope containment.
6. Confirm grant lifecycle and limits.
7. Confirm Future possesses authority to administer/seal this class of work.
8. Construct or bind the final immutable order using existing order machinery.
9. Record Apollo/Seshat/Janus provenance.
10. Seal the order.
11. Make the sealed order available to Mercury.

Future’s seal means:

> **This exact proposed operation fits existing Pantheon authority.**

It does not mean:

> **Future agrees with Apollo.**

---

## J. Existing authority only

Future normally approves only **already-approved classes of work**.

Examples may eventually include:

```text
daily briefing to Grant
routine calendar maintenance
approved email drafting
bounded notification delivery
pre-approved research classes
```

Future should not ask Grant to reapprove every individual instance of a standing authority.

The purpose of standing authority is to remove repetitive approval burden while retaining inspectable boundaries.

---

## K. Initial authority

Future may begin with little or no standing operational authority beyond what is required for its deterministic administrative function.

Package availability must not create implicit grants.

A new Pantheon installation may therefore know:

```text
EMAIL:DRAFT exists
CALENDAR:MAINTAIN exists
NOTIFICATION exists
```

while Future is authorized to approve none of them yet.

Authority accumulates through explicit rulings rather than learned assumption.

---

## L. Insufficient authority

If the proposed operation is well-formed but no sufficient grant exists:

Future must not execute it.

Future creates or preserves a durable authority request containing enough information for a decision:

```text
proposal identity
package class
capability
scope
purpose
substantive missive/payload reference
current authority
specific authority gap
suggested permission shapes
source Apollo report
```

The request proceeds toward Prometheus.

---

## M. Prometheus path

Prometheus remains the control boundary for authority changes.

Future may ask Prometheus to resolve a requested authority change.

Prometheus may:

```text
approve within delegated authority
deny
escalate to Grant
```

Kernel 13 already requires a Prometheus approval to remain within active authority explicitly delegated to Prometheus; out-of-delegation requests remain pending/escalated.

Kernel 16 must use that mechanism.

Future must not broaden Prometheus authority.

---

## N. Grant path

Where Prometheus cannot resolve the request within its delegated authority, Future should surface a human-ruling request.

Grant should be able to inspect:

- what Apollo proposed;
- the actual missive or meaningful payload;
- package type;
- capability;
- scope;
- whether this operation has temporal constraints;
- current authority;
- requested authority;
- why current authority is insufficient;
- sensible grant choices.

The request should avoid opaque command strings where the structured class of authority can be described clearly.

---

## O. Context-sensitive approval options

Future should not present the same generic grant menu for every request.

Seshat’s permission class/package metadata should inform reasonable options.

### Example: one-off unusual operation

Possible choices:

```text
deny
allow once
allow for this project
```

### Example: recurring self-directed briefing

Possible choices:

```text
deny
allow once
allow until revoked
```

### Example: bounded research

Possible choices:

```text
deny
allow once
allow N uses
allow until timestamp
```

### Example: outbound external communication

Possible choices should normally be more conservative than nonpublic drafting.

Future supplies only grant shapes Kernel 13 can actually represent.

---

## P. Human approval is structured authority

When Grant approves an option, Pantheon should create the corresponding Kernel 13 authority state.

Do not merely mark a proposal:

```text
user said yes
```

The resulting durable grant should establish:

- holder;
- capability;
- scope;
- duration/use bounds;
- granting authority;
- delegation constraints;
- provenance.

This allows later matching proposals to be approved deterministically.

---

## Q. Once means once

If Grant approves an operation once, that must not silently establish standing permission.

Kernel 13 already supports finite-use authority and atomically consumes finite grants when authorized Go is issued.

Future must preserve this distinction.

A successful one-time operation is not precedent for future authority.

---

## R. Project/context-bounded authority

Where Seshat and the authority model support a project/context scope, Future may offer a bounded class such as:

```text
this operation once
this operation type for Project X
```

Do not invent semantic project containment in Future.

A project scope must be represented deterministically enough for Kernel 13 containment to enforce it.

Unknown containment must escalate rather than guess.

---

## S. Sealing

Future is the normal sealing boundary for operations proposed by Apollo and resolved through this pathway.

The seal must bind the immutable operation using existing Mercury/order semantics rather than introducing Future-specific cryptography.

Existing Kernel 11 behavior already distinguishes authorization seal from execution Go and binds Go to the exact sealed order.

Reuse it.

Future should not issue Go.

---

## T. Mercury handoff

Mercury receives sealed work.

Mercury then performs its existing role:

- verify valid sealed order;
- obtain/issue Go under existing rules;
- dispatch through the exact bounded adapter;
- record attempt;
- consume Go;
- preserve result/evidence.

Future does not directly call the target tool as an optimization.

No “simple operation” bypass exists.

---

## U. Tool/adapter availability

A proposal may be:

```text
authorized
```

but still:

```text
adapter unavailable
```

These are different conditions.

Future should preserve that distinction.

Do not convert lack of an adapter into:

- arbitrary shell execution;
- Apollo tool access;
- a different plugin with vaguely similar behavior.

Seshat should identify the expected adapter.

Mercury should dispatch only through an explicit available adapter.

---

## V. Draft versus send

Kernel 16 should preserve capability distinctions such as:

```text
EMAIL:DRAFT
```

versus:

```text
EMAIL:SEND
```

Authority to create nonpublic drafts must not imply authority to send external communications.

Likewise:

```text
CALENDAR:READ
```

must not silently imply:

```text
CALENDAR:MODIFY
```

Capability containment remains explicit under Kernel 13.

---

## W. Pause

Prometheus pause remains above Future execution flow.

While paused:

- proposals may remain durable;
- authority questions may remain visible;
- Future inspection remains available;
- safe human rulings may be recorded if existing Prometheus/control policy permits;
- no proposal proceeds to autonomous external effect;
- no stale operation becomes executable merely because authority existed before pause.

Resume does not automatically dispatch queued operations.

Existing Mercury semantics already preserve this no-replay principle.

Future must not undermine it.

---

## X. Future round

Implement one explicit deterministic processing primitive:

```text
pantheon future round
```

or the repository-consistent equivalent.

A Future round should:

1. locate eligible unprocessed proposals/packages;
2. validate Seshat package state;
3. inspect Janus state where applicable;
4. check Kernel 13 authority;
5. seal sufficiently authorized work;
6. create/refine authority requests for insufficiently authorized work;
7. refer requests toward Prometheus where appropriate;
8. preserve human-ruling requirements;
9. record decisions/provenance;
10. preserve idempotency.

A Future round must not invoke a reasoning model.

---

## Y. Future decision states

Use the smallest durable state set consistent with actual implementation.

The system must at least distinguish the equivalent of:

```text
awaiting
sealed
needs_authority
referred_prometheus
needs_person
denied/refused
temporally_blocked
malformed
adapter_unavailable
```

Do not add states merely for conceptual elegance.

Prefer reuse of existing order/grant/package states where clean.

---

## Z. Idempotency

One proposal must not produce duplicate sealed orders across repeated Future rounds.

Authority requests should also avoid duplication.

Repeated processing should converge on durable state rather than repeatedly asking Grant the same question.

A proposal already linked to a sealed or terminal order should not generate another unless a fresh successor proposal explicitly exists.

---

## AA. Provenance

The person must be able to trace an operation:

```text
Apollo report
   ↓
missive
   ↓
Seshat package
   ↓
Janus transition, if any
   ↓
Future authority decision
   ↓
grant / authority chain
   ↓
sealed order
   ↓
Go
   ↓
Mercury attempt
   ↓
result
```

Not every operation requires every step.

Where a step exists, preserve the linkage.

---

## AB. Inspection

Provide human-inspectable surfaces consistent with Pantheon conventions.

At minimum, the person should be able to answer:

> What is Future waiting on?

> What has Future sealed?

> Which proposals need Prometheus?

> Which require me?

> What authority covered this operation?

> Why was this one denied or blocked?

Suggested surfaces:

```text
pantheon show future
pantheon show future proposals
pantheon show future proposal <id>
```

Exact CLI naming may follow existing conventions.

Expose:

- proposal identity;
- package;
- capability;
- scope;
- source Apollo report/missive;
- temporal state where relevant;
- authority decision;
- matching grant;
- requested grant options;
- Prometheus disposition;
- resulting order;
- eventual Mercury status.

---

## AC. First authorized proof

Kernel 16 must prove a harmless already-authorized path.

Prefer an existing bounded local capability.

Example shape:

```text
Apollo/Seshat proposal
        ↓
active matching grant
        ↓
Future seal
        ↓
Mercury Go
        ↓
harmless adapter
        ↓
durable result
```

Do not add broad external write access solely to make the demonstration impressive.

---

## AD. Prometheus-bounded proof

Kernel 16 must prove a second path:

```text
proposal
  ↓
insufficient Future-standing authority
  ↓
Prometheus possesses appropriate delegated authority
  ↓
Prometheus approves bounded grant/change
  ↓
Future later resolves authorization
  ↓
seal
```

The exact operation can remain local/synthetic if necessary.

The point is to prove hierarchy.

---

## AE. Human-ruling proof

Kernel 16 must prove a third path:

```text
proposal
  ↓
no sufficient authority
  ↓
Prometheus cannot approve within delegation
  ↓
needs_person
  ↓
NO EXECUTION
```

Do not mutate consequential live authority merely for smoke testing.

Repository tests may exercise the complete approval path.

Live testing may demonstrate the unresolved human boundary without granting it.

---

## AF. No model in authority path

Automated tests must prove that Future authority resolution functions with Apollo unavailable.

A previously persisted proposal/package must be resolvable by Future without loading a reasoning model.

This is a central Kernel 16 boundary.

---

## AG. No packaging in Future

Tests should demonstrate Future consumes Seshat output rather than reimplementing plugin-specific parsing.

Future may validate references/invariants needed for safety.

It should not contain hard-coded Gmail/Calendar/etc. serialization merely to make initial cases pass.

---

## AH. Temporal safety

For Janus-managed work, tests must establish:

- not-yet-ready work is not sealed for execution;
- expired work is not revived;
- `require_review` remains blocked;
- missed work is handled through Janus policy rather than Future improvisation;
- pause/resume does not create stale replay.

Future must not reinterpret Janus transition policy.

---

## AI. Permission learning without behavioral inference

Future may gradually accumulate useful standing authority because Grant explicitly chooses broader grant shapes.

That is acceptable.

Future must not learn authority merely by observing repeated approvals.

For example:

```text
Grant approves EMAIL:DRAFT once
```

does not mean:

```text
Future notices Grant usually approves drafts
→ silently creates until_revoked
```

Explicit grant state, not inferred habit, governs.

---

## AJ. Automated verification

Before repository PASS:

```text
gofmt -w $(rg --files -g '*.go')
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel16 .
```

Focused tests should demonstrate at minimum:

1. Future processes persisted proposal/package state without invoking a model.
2. Apollo cannot self-authorize.
3. Seshat package identity/capability is respected.
4. Janus temporal blocking is respected.
5. Matching active authority allows Future sealing.
6. Paused/expired/revoked/exhausted grants do not authorize.
7. One-use grants remain one-use.
8. Capability widening is rejected.
9. Scope widening is rejected.
10. Already-authorized work does not require redundant human approval.
11. Insufficient authority creates a durable authority request.
12. Seshat permission shapes inform available grant options.
13. Future does not invent unsupported grant shapes.
14. Prometheus approves only within delegated authority.
15. Out-of-Prometheus authority reaches `needs_person`.
16. No execution occurs while human ruling is required.
17. Future cannot issue Go.
18. Future cannot directly invoke adapters/tools.
19. Future seal links to exact authority basis and source proposal.
20. Repeated Future rounds do not duplicate orders.
21. Repeated rounds do not duplicate authority requests.
22. Adapter-unavailable remains distinct from unauthorized.
23. Prometheus pause blocks autonomous effect.
24. Resume does not replay stale work automatically.
25. Existing Kernel 11–15 tests remain green.

---

## AK. Live verification

Before live PASS:

1. Build exact candidate.
2. Record SHA-256.
3. Install exact candidate.
4. Restart only necessary Pantheon service(s).
5. Verify backend health.
6. Verify Future inspection.
7. Exercise one harmless already-authorized proposal.
8. Prove Future sealed exactly one order.
9. Prove Mercury produced/consumed the expected Go and result.
10. Repeat Future/Mercury rounds and prove no duplicate effect.
11. Exercise an insufficient-authority proposal.
12. Prove it remains unexecuted and is referred correctly.
13. Prove one out-of-Prometheus request reaches person-required state.
14. Verify Janus blocking remains honored.
15. Verify Prometheus pause/no-replay remains intact.
16. Record exact evidence.

Do not send real consequential external communication merely to satisfy smoke criteria.

---

## AL. Human usability criterion

Future should reduce permission friction, not create a permission bureaucracy.

Grant should not routinely need to inspect:

- raw grant IDs;
- command strings;
- adapter internals;
- cryptographic digests.

Those remain available for deeper inspection.

The ordinary authority question should read conceptually like:

```text
Apollo prepared an email draft.

Requested:
Draft email to Heather

Authority:
No standing EMAIL:DRAFT permission currently covers this request.

Options:
- Deny
- Allow this draft once
- Allow email drafts for this project
- Allow email drafts until revoked
```

Actual options depend on Seshat’s permission class and Kernel 13’s enforceable scopes.

The interface may initially remain CLI/textual.

The underlying structured decision matters more than polish.

---

## AM. Non-goals

Kernel 16 does not:

- implement Orion;
- implement recurring schedule ownership;
- implement Nostalgia;
- implement Aphrodite;
- make Future a planner;
- invoke Apollo for authority judgment;
- teach Future plugin APIs;
- replace Seshat;
- replace Janus;
- replace Father Time;
- replace Kernel 13 authority;
- replace Prometheus;
- replace Mercury;
- automatically infer standing permission from behavior;
- create arbitrary shell execution;
- enable unrestricted Internet access;
- implement broad email-send autonomy;
- build the GUI;
- create another cryptographic layer.

---

## AN. Reportback requirements

The reportback must state:

- exact predecessor;
- starting repository state;
- actual Kernel 14 Seshat contract consumed;
- actual Kernel 15 Janus state consumed;
- active documentation corrections;
- Future persistence/state model;
- authority-resolution implementation;
- already-authorized behavior;
- permission-option generation;
- Prometheus referral behavior;
- human-ruling behavior;
- sealing implementation;
- Mercury handoff;
- temporal blocking;
- pause/no-replay behavior;
- inspection surfaces;
- automated test results;
- live authorized proof;
- live insufficient-authority proof;
- live person-required proof;
- binary SHA-256;
- files changed;
- limitations;
- inherited verification debt;
- report expiry conditions.

Do not claim Future performs reasoning if it does not.

Do not claim whole-animal usability merely because Future itself passes.

---

## AO. Success condition

Kernel 16 PASS means:

> **Pantheon can deterministically turn a well-formed proposed action into either lawful sealed work or an exact authority question without allowing the reasoning model to decide its own permission.**

Pantheon should then possess the major pieces required for:

```text
observe
→ remember
→ reason
→ compose
→ package
→ understand temporal eligibility
→ resolve authority
→ seal
→ transport
→ execute
→ record
```

---

## AP. Next safe successor

If Kernel 16 passes, the next committed kernel is:

**Kernel 17 — Whole-Animal Integration and Usability Proof**

Kernel 17 should not introduce another major deity merely to expand Pantheon.

It should connect and operate what already exists.

The point will be to prove useful real workflows end to end and identify the first actual operational friction that deserves the next kernel.

Orion, Archimedes, Zeus, Nostalgia, Aphrodite, GUI work, and additional plugin adapters remain candidates after that checkpoint rather than automatically sequenced construction.