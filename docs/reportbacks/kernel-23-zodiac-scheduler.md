---
kernel: 23
status: REPOSITORY VERIFIED — live Apple/Google, Situation Brief, and Hermes cutover acceptance pending
predecessor: 7b7eeb9019e648791bf7375ec234db7d05e82999
---

# Kernel 23 Reportback — Zodiac Deterministic Schedule Keeper

Kernel 23 adds Zodiac as Pantheon's persisted application scheduler. Zodiac
stores what is supposed to happen when; it does not interpret language, decide
importance, create authority, replace Father Time or Janus, install adapters,
or overwrite human calendar corrections.

## A. Predecessor and boundary

The clean Kernel 22 predecessor is:

```text
7b7eeb9019e648791bf7375ec234db7d05e82999
record Kernel 22 verification results
```

Kernel 22's mail understanding path is preserved. Existing Hermes services and
jobs were not stopped, deleted, or rewritten.

## B. Zodiac registry and persistence

SQLite now persists:

- `zodiac_schedules`: normalized declarations, provenance, timezone, precision,
  recurrence, catch-up policy, dependency, action, projection policy, priority,
  attestation reference, and next/last due state;
- `zodiac_instances`: stable occurrence identity, due state, catch-up decision,
  Future proposal, authority/projection state, and error/result history;
- `zodiac_projections`: provider/account/calendar, external event ID, role,
  revision, sync state, and human correction state;
- `zodiac_revisions`, `zodiac_attestations`, and `zodiac_rounds` for correction,
  evidence, and evaluation history.

The daemon seeds two explicit declarations when Zodiac is enabled:

```text
Mail Intake              every hour       catch_up=reconcile
Grant's Situation Brief  07:00 daily      configured catch-up policy
```

The declarations survive restart. A unique `(schedule_id, occurrence_key)`
identity prevents duplicate instances after repeated evaluation.

## C. Wake engine and catch-up

Zodiac runs an in-process bounded wake loop with a configurable interval (30
seconds by default). It uses Father Time's local operating timezone and
timezone-aware `time.AddDate` recurrence arithmetic, including DST behavior.
Supported recurrence forms include hourly, daily, weekdays, weekly, monthly,
yearly, `INTERVAL`, `COUNT`, `UNTIL`, and `BYDAY` through an iCalendar-style
RRULE contract.

Each schedule declares `skip`, `run_once`, `reconcile`, or `needs_you` catch-up.
Reconcile and run-once policies select one bounded current instance rather than
replaying every missed recurrence. A paused round records a
`blocked_prometheus` instance and creates no Future proposal; resuming does not
replay that same instance.

## D. Dependencies and attestation

The deterministic dependency seam supports `after_completion` and
`not_before`. Unsupported or semantically unavailable conditions, including
reply-state evaluation without a deterministic source, become Needs You rather
than being guessed through.

`POST /v1/zodiac/attestations` accepts a source ID, source path/fingerprint,
citation bounds, quoted span, normalized action, speaker, and confidence. When
the source path is available, Zodiac verifies the SHA-256 fingerprint and that
the quoted span exists before recording a verified attestation. This checker
does not decide whether the person truly intended the action.

## E. Future and Mercury handoff

An hourly mail due instance creates one exact `ZODIAC:MAIL_ROUND` Future
proposal with scope bound to that instance. Future remains the authority
consumer; Mercury remains the only execution path. The new `pantheon` adapter
contract dispatches `mail.round` through the existing bounded mail operation.
No Zodiac due state is treated as a grant, seal, Go, or completed result.

The Situation Brief declaration is durable and due-state aware. Its independent
composition/delivery pipeline is not fabricated by this kernel, so its due
instance remains `awaiting_pipeline` until that bounded operation exists.

## F. Apple, Google, and projection policy

Zodiac exposes provider-neutral operations:

```text
calendar.list
calendar.read
calendar.create
calendar.update
calendar.cancel
calendar.reconcile
calendar.availability
```

Initial configurable policy is personal/family → Apple and work → Google;
`both`, `none`, and provider swaps are supported as policy values. The
projection registry stores one Zodiac identity with separate provider IDs.

No Apple/iCloud or Google Calendar connector was installed or downloaded.
The runtime reports Apple/Google as unavailable unless a corresponding approved
Bifrost capability is already registered and enabled. No live provider write,
external event ID, or credential is claimed.

## G. Human corrections and conflicts

The projection reconciliation route records human edits as
`human_corrected`, preserves the external revision, and does not overwrite the
correction. Human deletion becomes `human_deleted` and is not automatically
recreated. A second projection of the same Zodiac schedule is marked
`needs_you` when a human correction requires comparison. Provider identifiers
are never inferred from title/time matching once an external ID exists.

## H. Panopticon and operator surface

Panopticon now includes Zodiac with Today, recurring declarations, Needs You,
provider health, correction state, and an explicit **Evaluate now** control.
Current Work includes due Zodiac records with human explanations and next
actions. New read/control routes are documented in the GUI contract. CLI entry
points include:

```text
pantheon show zodiac
pantheon zodiac today
pantheon zodiac upcoming
pantheon zodiac needs-you
pantheon zodiac round
pantheon zodiac schedule ...
```

## I. Verification

Focused Kernel 23 tests cover:

- typed Zodiac configuration and invalid timezone rejection;
- local-time recurrence across the America/Los_Angeles DST transition;
- bounded reconcile catch-up and repeated-round idempotence;
- Prometheus pause blocking and no replay of the same instance on resume;
- deterministic source fingerprint/quoted-span attestation;
- provider-unavailable Panopticon state;
- human correction preservation, deletion non-recreation, and conflicting
  projection Needs You state.

Repository verification passed:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
node --check internal/pantheon/web/app.js
python3 -m json.tool config/pantheon.json
python3 -m json.tool config/plugins/registered/core.json
git diff --check
```

The live calendar acceptance, controlled transcript-to-calendar proof, real
Situation Brief delivery, 1–2 day Hermes overlap, and final Hermes pause remain
unperformed. This reportback does not claim them.

## J. Known limitations and next boundary

After-hours work policy fields are configurable, but live provider conflict and
dual-calendar attention proof awaits an approved adapter. Calendar availability,
external ingestion, and provider writes remain Bifrost adapter work. The
Situation Brief schedule exists, but composition and delivery are separate
capabilities. No Kernel 24 or Aphrodite implementation was started.

The next smallest safe step is operator-run live acceptance against already
approved calendar capabilities, followed by a measured Hermes overlap/cutover
decision. It is not permission to begin another architecture kernel.
