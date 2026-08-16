# Kernel 13 — Authority, Grants, and Delegation

## Objective

Implement Pantheon’s deterministic authority model.

Kernel 13 must allow Pantheon to answer:

> **May this component cause this action, against this scope, right now—and under whose authority?**

Authority must be inspectable, bounded, revocable, and traceable to a valid authority chain. Intelligence is never authority.

Kernel 13 establishes the authority machinery that Future and other deterministic Pantheon components will consume. It does **not** implement Future’s complete daemon behavior.

---

## A. Constitutional invariants

### 1. The person remains sovereign

Pantheon’s authority ultimately derives from the person.

A component cannot create authority merely because an action appears useful, safe, intelligent, or consistent with prior behavior.

Existing Pantheon principles remain controlling:

- A person is not a user account.
- Durable state belongs to Pantheon.
- Intelligence is not authority.
- No component inherits authority merely because another component possesses it.
- Human inspection, correction, revocation, and exit remain available.

### 2. Deities possess no operational authority

A deity may:

- reason;
- interpret;
- investigate;
- classify;
- retrieve through capabilities provided to its deterministic boundary;
- compose a missive;
- recommend an action;
- propose structured work.

A deity may **not**:

- grant permission;
- expand permission;
- delegate authority;
- seal an order;
- issue a Go Command;
- invoke an operational capability merely because it recommends doing so;
- treat its own output as authorization.

No model output constitutes executable authority.

Apollo may conclude that an email should be sent and may compose the complete email. Apollo cannot authorize its transmission.

The proposed action must cross the appropriate deterministic Pantheon authority boundary.

### 3. Daemon and deity are distinct

The reasoning deity associated with a function is not interchangeable with the deterministic daemon boundary surrounding that function.

Operational authority may be entrusted only to deterministic Pantheon machinery.

If an unauthorized operational effect is traceable through a deity-facing daemon, Pantheon must treat this as a containment failure requiring Prometheus attention. Possible causes include jailbreak behavior, compromised or defective daemon enforcement, malformed authority validation, or another control failure.

The existence of model nondeterminism is not an excuse for unauthorized execution.

---

## B. Authority model

Authority shall be represented through explicit grants.

At minimum, an authority grant records:

```text
grant_id
holder
capability
scope
granted_by
parent_grant_id
created_at
valid_from
valid_until
remaining_uses
status
purpose
delegation_constraints
```

Implementation may retain or extend Kernel 11’s existing `capability_grants` machinery where appropriate. Do not create a parallel authority system merely to obtain cleaner names.

Kernel 11 already recognizes `person_direct`, `standing_authority`, and `parent_order` authority and performs exact capability/scope checks. Kernel 13 shall reconcile and generalize those primitives rather than discard them.

---

## C. Authority holders

Operational authority may be held by deterministic Pantheon components.

Reasoning deities and external/frontier models are never authority holders.

Tools do not gain independent authority merely because Pantheon is capable of invoking them. They receive bounded authorized requests through Pantheon.

The authority record must identify the actual deterministic holder rather than attributing authority to the intelligence that may have proposed the work.

---

## D. Capabilities

A capability identifies **what may be done**.

Capabilities should remain concrete and inspectable.

Examples include:

```text
vault.read
calendar.read
calendar.maintain
email.read
email.send
notify.send
internet.search
present.process
```

Configuration or technical availability does not imply authority.

A configured capability answers:

> Pantheon can potentially do this.

An active grant answers:

> This holder is presently authorized to cause this bounded use of it.

These must remain separate concepts.

---

## E. Scope

Every operational grant must have a scope.

Scope answers **what the capability may act upon or affect**.

Examples may include:

```text
calendar:personal
calendar:event:<id>
email:inbox
email:message:<id>
occurrence:<id>
vault:00-Pantheon-Core
internet:domain:<domain>
```

Kernel 13 should prefer deterministic scope comparison.

Do not introduce semantic model judgment into authority containment.

Where Pantheon cannot deterministically prove that a requested child scope is contained within its parent authority, it must not assume containment.

---

## F. Grant duration and limits

Pantheon must support bounded authority.

A grant may be:

- valid for one task;
- valid until a timestamp;
- limited to a specified number of uses;
- standing until revoked;
- bounded by more than one of these conditions.

Examples:

```text
internet.search
scope: domain:example.org
remaining_uses: 1
```

```text
calendar.maintain
scope: calendar:personal
valid_until: <timestamp>
```

```text
notify.send
scope: situation-brief
status: active
valid_until: none
```

Expiration and use exhaustion must be deterministic.

An expired or exhausted grant cannot authorize new work.

---

## G. Grant states

Kernel 13 begins with the following authority states:

```text
pending
active
paused
expired
revoked
```

### `pending`

The proposed authority exists as durable state but is not currently usable.

### `active`

The grant may authorize work within its capability, scope, limits, and delegation constraints.

### `paused`

The authority remains recorded but cannot presently authorize execution.

Pause does not silently become revocation.

### `expired`

The grant has passed a deterministic validity boundary.

Expiration must not erase its history.

### `revoked`

An authorized control action has withdrawn the grant.

Revocation must preserve the historical authority record and the evidence of its revocation.

Additional states require demonstrated need rather than speculative completeness.

---

## H. Delegation

Authority may be delegated only where the grant explicitly permits delegation.

The fundamental invariant is:

> **A component may preserve or narrow authority downstream. It may not enlarge authority unless it separately possesses explicit authority to perform that enlargement.**

For example:

```text
calendar.maintain
    ↓
calendar.read
    ↓
calendar:event:123
```

may be a lawful narrowing where the capability relationships and scopes explicitly establish that containment.

This is not lawful:

```text
email.read
    ↓
email.send
```

unless the delegating component independently possesses authority allowing that expansion.

A child authority must remain provably inside the authority from which it derives.

The authority chain must remain inspectable back to its valid source.

---

## I. Provenance

Pantheon must be able to explain why an operation was authorized.

For consequential authorized work, inspection should be able to establish a chain resembling:

```text
Grant
  → authority delegated to Prometheus
      → bounded authority administered downstream
          → specific order sealed
              → Go issued
                  → Mercury delivered
                      → capability invoked
```

The exact chain will vary.

The invariant does not:

> Every operational action must have a traceable authority basis.

A deity's recommendation may appear in the evidentiary history of an action, but it must never occupy an authority position in that chain.

---

## J. Prometheus boundary

Prometheus is the control boundary for changes in Pantheon authority.

Kernel 13 must establish the mechanism by which a requested authority change can be referred to Prometheus.

Prometheus may approve changes that fall within authority already delegated to Prometheus.

Prometheus may not bootstrap authority beyond that delegation.

When a requested permission change exceeds Prometheus’s existing authority, the request must be escalated to the person.

Kernel 13 does **not** attempt to comprehensively define what constitutes a “meaningful permission change.”

It establishes the machinery required for Prometheus to:

```text
approve within existing authority
deny
refer/escalate to the person
```

The richer policy governing when Prometheus should exercise or escalate its discretion may be developed separately.

---

## K. Future boundary

Kernel 13 establishes infrastructure Future will later administer.

Future is not a reasoning deity.

Future does not determine whether Apollo’s recommendation is intelligent, wise, or strategically desirable.

Apollo may provide reasoning, conclusions, proposed operations, prepared missives, requested capabilities, intended scopes, and other required metadata.

Future’s later responsibility is deterministic:

> **Does sufficient existing authority already cover the proposed operation?**

Where sufficient authority exists, Future may exercise whatever sealing authority has been explicitly granted to Future.

Where it does not, Future must not manufacture authority.

The request proceeds toward the appropriate authority boundary, including Prometheus or the person as required.

Kernel 13 should not implement speculative Future planning or trajectory reasoning.

---

## L. Seals and Mercury

Do not replace Kernel 11’s seal and Go machinery without demonstrated need.

Kernel 11 already establishes:

- immutable order sealing using SHA-256;
- Father Time binding Go Commands to the exact order seal;
- bounded Go validity;
- one-Go/one-attempt behavior;
- consumption;
- invalidation under Prometheus pause;
- durable audit evidence.

Kernel 13 must make authority provenance available to sealing components without turning Mercury into the authority adjudicator.

Mercury carries authorized work.

Mercury does not decide that the underlying proposal was wise.

Mercury does not grant permission.

Mercury does not expand scope.

Mercury receives valid sealed work and the Go necessary to move it through the authorized capability path.

The component authorized to seal a particular document/order is responsible for doing so.

Avoid additional cryptographic infrastructure unless the actual local threat model demonstrates a need for it.

---

## M. Apollo boundary

Kernel 12 establishes Apollo’s deep local reasoning capability.

Kernel 13 must preserve a hard separation between that intelligence and operational authority.

A typical future flow may resemble:

```text
Present
    ↓
Past
    ↓
Apollo
    │
    ├── reasoning
    ├── evidence
    ├── conclusion
    ├── proposed operation metadata
    └── prepared missive/payload where appropriate
              ↓
           Future
              ↓
       authority resolution
          /          \
 already authorized   insufficient authority
       ↓                    ↓
      seal             Prometheus / person
       ↓
    Mercury
       ↓
      tool
```

Apollo's ability to produce a perfectly structured request must not allow Apollo to bypass Future or another required deterministic authority boundary.

Well-formed model output is still model output.

---

## N. Containment failures

Pantheon must distinguish:

```text
bad recommendation
```

from:

```text
unauthorized operational effect
```

A deity producing an inappropriate, malicious, confused, or jailbreak-influenced recommendation is a reasoning failure.

If deterministic controls contain it, the authority system has worked.

If that deity's output nevertheless causes an operational effect without valid authority, Pantheon has experienced a control/containment failure.

Such an event must be:

- durably recorded;
- attributable as far as evidence permits;
- surfaced to Prometheus;
- prevented from silently establishing precedent or new authority.

No component may infer future permission merely because an unauthorized operation previously succeeded.

---

## O. Inspection

Kernel 13 must provide human-inspectable authority state.

The exact CLI names may follow existing Pantheon conventions, but the person must be able to inspect at least:

- current active grants;
- pending grants;
- paused grants;
- expired grants;
- revoked grants;
- the authority held by a particular deterministic component;
- the capability and scope of a grant;
- grant limits and remaining uses;
- grant provenance;
- parent/child delegation relationships;
- why a particular operation was or was not authorized.

Prefer extending existing `pantheon show ...` surfaces rather than creating an unrelated administration interface.

---

## P. Persistence and migration

Preserve existing durable authority and audit state.

Kernel 13 is **not a database-migration project**.

If implementation requires a SQLite schema migration because existing persisted structures cannot represent the required authority model, perform the smallest safe migration necessary and preserve existing records.

Do not redesign storage merely for architectural neatness.

The architecture is the authority model; SQLite is an implementation detail.

---

## Q. Verification

Automated verification should demonstrate at minimum:

1. A deity/model cannot create an executable grant.
2. A deity/model cannot seal its own proposed order.
3. An active exact grant authorizes an in-scope operation.
4. A paused grant does not authorize execution.
5. An expired grant does not authorize execution.
6. A revoked grant does not authorize execution.
7. A one-use grant cannot authorize a second use.
8. A time-limited grant deterministically expires.
9. A child delegation may preserve or narrow valid authority.
10. A child delegation cannot enlarge capability or scope.
11. A component cannot delegate authority it does not possess.
12. Prometheus may approve a change only within authority delegated to Prometheus.
13. Authority outside Prometheus’s delegation remains pending/escalated for the person.
14. Configuration/capability availability alone does not constitute permission.
15. Every successful authorization exposes a traceable authority basis.
16. Historical grants remain inspectable after expiry or revocation.
17. Existing Kernel 11 seal/Go invariants remain intact.
18. Prometheus pause continues to prevent execution and invalidate outstanding Go Commands as previously established.

Tests should prefer deterministic components and mocked/local boundaries. Do not require frontier inference to prove authority behavior.

---

## R. Non-goals

Kernel 13 does not:

- give deities authority;
- implement a generalized role/RBAC framework without demonstrated need;
- define every future Pantheon capability;
- comprehensively define “meaningful permission change”;
- make Future a planner;
- give Apollo direct tool authority;
- make Mercury an authority adjudicator;
- add unnecessary cryptographic infrastructure;
- introduce recurring autonomous scheduling;
- add Archimedes/frontier execution;
- build the GUI;
- silently convert existing configuration into standing permission.

---

## S. Reportback requirements

The implementation reportback must state:

- actual predecessor commit and starting repository state;
- whether constitutional documentation required correction;
- existing Kernel 11 authority machinery reused;
- schema/storage changes, if any;
- final grant representation;
- grant states implemented;
- duration/use limits implemented;
- deterministic scope-containment rules;
- delegation rules;
- Prometheus escalation path;
- model/deity containment enforcement;
- inspection surfaces;
- tests performed and results;
- live verification performed;
- known limitations;
- files changed;
- exact conditions under which the report expires.

Do not claim behavior that was only designed but not implemented or tested.

---

## T. Next safe successor

If Kernel 13 proves deterministic authority, delegation, provenance, and escalation without weakening Kernel 11’s execution controls, the next safe successor is:

**Kernel 14 — Future Authority Administration**

Future shall consume structured proposed work, resolve it against existing Pantheon authority, seal work that is already authorized within Future’s delegated authority, and refer insufficient authority toward Prometheus or the person.

Future shall not become a reasoning authority and shall not manufacture permission merely because a proposed action appears desirable.