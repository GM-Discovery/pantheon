# Kernel 29 — Zeus: Retention Inventory and Deterministic Sanitation

## Status

PASS for the bounded Kernel 29 increment — implemented and live-verified
2026-08-24. Destructive quarantine/removal remains intentionally deferred.

Zeus is the next implementation kernel after Libra. This document defines the
first safe increment: deterministic retention inventory, dry-run sanitation,
and an explicit receipt boundary for machine-owned cleanup. It does not grant
Zeus the ability to decide that human material is garbage.

## The short reason Zeus comes first

Archimedes is a privacy boundary for optional frontier reasoning. It will
create temporary sanitized missives, provider responses, privacy-gate reports,
exchange traces, and other transformation artifacts. Those artifacts need a
known owner, retention rule, inventory, and disposal receipt before frontier
traffic is enabled.

Zeus does not make Archimedes' redaction safe. Apollo and the deterministic
Archimedes privacy gate remain responsible for what may cross the frontier.
Zeus gives the temporary material a bounded lifecycle after it exists.

The ordering is therefore operational, not mythological:

```text
typed retention metadata and machine-artifact inventory
                         ↓
              Zeus dry-run and receipts
                         ↓
       Archimedes temporary exchange artifacts
                         ↓
        Zeus retention/quarantine after closure
```

Archimedes can be designed in parallel. Its implementation should not enable
frontier transmission until the required temporary-artifact retention path is
proven.

## Mission

Give Pantheon a deterministic sanitation owner that can answer:

1. what machine-owned material exists;
2. who or which subsystem owns it;
3. what retention policy applies;
4. when it becomes eligible for sanitation;
5. what durable replacement or completion receipt must already exist; and
6. what exact bounded action would occur.

The first Zeus surface should make cleanup understandable before it makes
cleanup destructive.

## Governing boundary

> Zeus may dispose of explicitly identified machine garbage. Zeus may never
> decide that human material is garbage.

Zeus is deterministic. It does not use Apollo, Archimedes, an embedding model,
filename semantics, topic classification, or “old-looking” content to decide
eligibility.

| Zeus may do | Zeus may not do |
| --- | --- |
| inventory configured machine-owned roots | classify a human Vault document as trash |
| evaluate typed retention metadata | infer importance from content or age alone |
| produce a dry-run sanitation report | delete because a schedule became due |
| revalidate an exact sealed sanitation manifest | create authority or a Future ruling |
| quarantine or remove an explicitly permitted machine artifact | follow symlinks or leave configured roots |
| record hashes, paths, states, and receipts | rewrite, summarize, or interpret source material |

No command, schedule, notification, model response, or configuration toggle
may silently turn on destructive cleanup.

## Existing contracts Zeus consumes

Zeus is a downstream sanitation owner. It should consume existing durable
state rather than inventing a second retention system:

- Father Time supplies received, completed, expired, and eligible timestamps;
- Intake, Rumor, Present, Apollo, Libra, and future Archimedes stages supply
  artifact ownership and processing state;
- Present and report assembly supply proof that a durable Markdown result was
  verified where a policy requires one;
- Future supplies the explicit decision and exact sanitation order;
- Mercury supplies the seal, Go boundary, and execution receipt; and
- Prometheus can pause the operation before any new sanitation work begins.

If the upstream state is missing, contradictory, or merely implied by a path,
Zeus marks the candidate `needs_you` or `protected`; it does not guess.

## Machine-artifact categories

Kernel 29 should begin with a small allowlist. Every category must have a typed
owner and a configured root. Unknown paths remain protected.

Initial candidate categories:

- temporary Present or processor files whose verified handoff is complete;
- expired Rumor audio when the retention policy explicitly permits removal;
- temporary conversion and extraction workspaces;
- completed report-assembly scratch material, never the durable manifest;
- bounded logs whose retention period has elapsed;
- model/cache artifacts under a designated machine-owned cache root;
- orphaned machine workspaces with no active database reference; and
- later, closed Archimedes exchange artifacts after the exchange and local
  rehydration are complete.

The following are protected by default:

- the configured Vault;
- Intake raw payloads and transcripts unless a separate typed policy marks the
  exact machine-owned artifact eligible;
- verified normalized Markdown and report manifests;
- identity, Fame, Echo, and correction records;
- active or failed work still required for recovery;
- anything referenced by an open order, exchange, occurrence, or report run;
- symlinks, paths outside an allowed root, and unknown ownership; and
- any path without a deterministic retention reason.

Raw Omi material is not machine garbage merely because Present has converted a
transcript. Its retention remains a separately declared policy decision.

## Retention declaration

The implementation should expose typed policy rather than scattered constants.
A retention rule needs at least:

```text
policy_id
artifact_category
owner
allowed_root
minimum_age
eligibility_event
required_state
replacement_requirement
action                quarantine | remove
enabled
created_at
updated_at
```

`eligibility_event` must name a deterministic event such as
`conversion_verified`, `exchange_closed`, `log_completed`, or
`workspace_orphaned`. “Not used recently” is not sufficient for human-facing
material.

`replacement_requirement` may require a verified normalized file, report
manifest, completion receipt, or other exact durable reference before the
candidate becomes eligible.

## Candidate and run records

The durable schema should make a sanitation decision inspectable without
opening or preserving unnecessary sensitive content.

### Sanitation candidate

```text
candidate_id
policy_id
category
owner
root
relative_path
absolute_path                 local inspection only; omit from remote output
bytes
modified_at
content_sha256
source_receipt_id
replacement_path
eligible_at
state                        protected | eligible | needs_you | queued |
                             quarantined | removed | failed | skipped
reason_code
reason_detail
discovered_at
updated_at
```

The candidate record must not claim eligibility merely because the file exists
or has an old modification time.

### Sanitation run

```text
run_id
mode                         inventory | dry_run | execute
policy_snapshot_sha256
candidate_count
eligible_count
protected_count
needs_you_count
action_count
state
created_at
completed_at
error
```

### Action receipt

```text
action_id
run_id
candidate_id
sealed_order_id
action                       quarantine | remove
path_before
path_after                   quarantine only
bytes
content_sha256
state
verified_at
error
```

A receipt records what happened to the exact candidate. It does not authorize
future candidates with similar names.

## Inventory semantics

An inventory round shall:

1. load the typed retention policy snapshot;
2. enumerate only the configured roots without following symlinks;
3. identify artifacts using deterministic ownership metadata, database
   references, and exact category rules;
4. calculate size, modification time, and a content hash where safe and
   practical;
5. verify active references, open orders, unfinished processing, and current
   Prometheus state;
6. evaluate the retention rule and replacement requirement;
7. write candidates and a run receipt; and
8. report protected and ambiguous material alongside eligible material.

The inventory must not read human document contents to classify them. A path
that cannot be proven machine-owned is protected.

## Dry-run report

The dry-run is the primary Kernel 29 deliverable. It should be available from
Panopticon and a discoverable CLI/API inspection route.

It must answer, for every candidate:

- what path or machine-owned object was found;
- which policy selected it;
- which deterministic event made it eligible;
- which durable replacement or completion receipt was verified;
- what action would occur;
- what would be retained;
- and why anything was protected or deferred.

The report must distinguish:

```text
eligible now
eligible after a known time
protected by policy
needs a person
unknown / not classified as machine garbage
```

Dry-run mode has no deletion, quarantine, rename, overwrite, or order-creation
side effect.

## Execution boundary

The first destructive proof should be one narrow, recoverable machine-owned
category. The path is:

```text
Panopticon or CLI requests dry-run
              ↓
person reviews exact candidates
              ↓
Future creates an exact sanitation proposal/order
              ↓
Mercury seals and issues Go
              ↓
Zeus reloads the policy and exact candidate manifest
              ↓
Zeus rechecks root, hash, state, and eligibility
              ↓
quarantine or remove exact machine artifacts
              ↓
Zeus verifies the result and records receipts
```

Zeus must refuse execution when:

- the candidate hash changed;
- the path moved outside its allowed root;
- the candidate is now referenced by active work;
- the retention policy changed incompatibly;
- Prometheus is paused;
- the order does not name the exact candidate set; or
- the candidate is human, unknown, or not explicitly machine-owned.

Quarantine is preferred to irreversible removal for the first execution proof.
If removal is later allowed, it must be a separate typed action and receipt,
not an implicit fallback from failed quarantine.

## Operator surface

Panopticon should expose:

- current retention policies without secrets;
- last inventory and dry-run status;
- counts of eligible, protected, needs-you, and failed candidates;
- candidate evidence and exact reason codes;
- a human review boundary for the proposed candidate set; and
- receipts for any approved quarantine/removal.

The operator surface must not offer “clean everything,” semantic cleanup, or a
single button that combines inventory, approval, and deletion.

## Failure and recovery rules

- Failed inventory preserves all source material and records the failure.
- A partial scan does not make unscanned files eligible.
- A failed replacement verification protects the candidate.
- A changed file hash invalidates the action and requires a new dry-run.
- A paused Prometheus state stops new execution; it does not replay skipped
  sanitation automatically.
- A missing database reference does not prove orphan status by itself.
- A failed quarantine leaves the original artifact untouched.
- A completed quarantine can be restored only through a separate explicit
  recovery operation with its own receipt.
- Zeus never retries a destructive action from memory; it reloads the exact
  order and candidate state.

## Acceptance criteria

Kernel 29 passes its first implementation boundary when:

1. A person can inspect typed retention policies and run an inventory without a
   model.
2. A dry-run lists eligible, protected, needs-you, and unknown material with a
   policy and reason for each.
3. The configured Vault and unknown paths are protected by default.
4. No dry-run creates an order or changes a file.
5. A narrow machine-owned category can be proven eligible only after its exact
   replacement/completion condition is met.
6. An explicit Future/Mercury order can quarantine that exact candidate set,
   with hash and path revalidation at execution time.
7. Every action has a durable receipt and read-back verification.
8. Pause, policy change, hash change, and active-reference change prevent
   execution safely.
9. A second scan is idempotent and does not recreate completed candidates.
10. The system never invokes Apollo or Archimedes to decide whether a file is
    garbage.

## Non-goals

Kernel 29 does not implement:

- semantic importance or topic classification;
- human Vault cleanup;
- automatic deletion of raw Omi material;
- a general-purpose file cleaner;
- arbitrary shell commands;
- frontier privacy redaction;
- Archimedes transport or provider installation;
- calendar, email, or notification actions; or
- a hidden retention timer that bypasses Future, Mercury, or Prometheus.

## Relationship to Archimedes

Zeus owns the lifecycle of temporary machine artifacts. Archimedes owns the
meaning and privacy of the outbound frontier exchange. The handoff is narrow:

```text
Archimedes closes an exchange and proves local rehydration
                         ↓
Archimedes emits retention metadata, not a delete order
                         ↓
Zeus inventories the exact temporary artifacts
                         ↓
Future/Mercury decide and carry any later sanitation action
```

Archimedes must never ask Zeus to decide whether content is safe to disclose.
Zeus must never inspect frontier content to decide whether it is safe to send.
