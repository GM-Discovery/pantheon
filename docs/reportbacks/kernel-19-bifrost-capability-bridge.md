# Kernel 19 Reportback — Bifrost External Capability Bridge

## Status

**PASS — repository and controlled adapter integration verified August 17, 2026.**

No live Gmail mailbox cutover was performed in this pass; that would be an
external side-effectful operation requiring a separate smoke/cutover decision.
Hermes routine mail jobs remain available and have not been retired.

## What was built

Kernel 19 adds Bifrost as a governed external-capability boundary without
making it a new reasoning or authority deity.

- A durable capability registry records source, type, version, description,
  invocation, resources, network/data-egress facts, authentication need,
  Pantheon consumers, operations, prohibited operations, lifecycle state, and
  last invocation/change times.
- Download approval, installation approval, Bifrost registration, initial use
  approval, and automatic-update approval are recorded as separate durable
  events. Registration describes existing software; it does not download or
  install software.
- Enablement fails closed unless the capability is installed and has all
  required approval facts. Disablement leaves the record and receipts intact.
- Bifrost invocations produce non-secret receipts with the capability,
  requesting component, authorization reference, operation, outcome, failure
  kind, network-activity fact, result summary, and source ID where applicable.
- Prometheus pause blocks new Bifrost invocations. Resume does not replay a
  blocked invocation.
- The CLI exposes `pantheon bifrost status`, `list`, `show`, `approvals`,
  `invocations`, `register`, `enable`, `disable`, `mail status`, and
  `mail intake`.

## First adapter: read-only mail intake

The first crossing uses the existing local Himalaya/Gmail configuration as
implementation evidence, replacing neither Hermes wholesale nor the mailbox
itself.

- Account and mailbox are explicit and bounded; the default is Gmail/INBOX
  with a maximum of 50 messages per run.
- The first ordinary run establishes a timestamp baseline and excludes
  historical mail. `--bootstrap` is an explicit bounded batch mode.
- Processed source IDs form a durable deduplication ledger. Repeating a run
  does not reread or rewrite a completed source.
- Each selected message is read without mutation and written as a normalized,
  provenance-bearing Markdown artifact under Intake. The provider remains the
  authoritative mailbox.
- Source identity, message identity, sender, recipients, subject, date,
  body, account, mailbox, adapter provenance, attachment metadata, and
  ingestion evidence are preserved.
- The explicit attachment allowlist currently permits plain text, Markdown,
  PDF, and DOCX. HTML and other unlisted types remain metadata-only and are
  not opened, rendered, or executed. The email itself still enters Intake.
- Source read failures, authentication failures, unavailable connectors,
  malformed responses, policy blocks, pause blocks, and protected/human-
  modified artifacts have distinct implementation failure kinds.
- The adapter performs no send, reply, forward, delete, archive, unsubscribe,
  or account-configuration operation.

## Hermes migration boundary

The existing Hermes scripts remain preserved as evidence and rollback
reference:

- `/home/brick/.hermes/scripts/new_mail_triage_manifest.py`
- `/home/brick/.hermes/scripts/overnight_backlog_batch.py`

No Hermes job was deleted or disabled. A future cutover must first prove that
Bifrost and Hermes cannot concurrently mutate the same mailbox state, then
pause the corresponding Hermes jobs with a rollback record. The Kernel 19
implementation deliberately keeps the first Bifrost adapter read-only, making
the migration reversible.

## Verification performed

Focused Kernel 19 tests prove:

- capability presence does not imply approval, and unapproved invocation fails
  closed;
- download, installation, registration, and use approval history remains
  inspectable separately;
- enablement and disablement are explicit and deterministic;
- bounded mail search and message reads create normalized Intake artifacts;
- allowed text attachments are preserved while HTML attachments are blocked
  without blocking the email;
- source/provider data is not mutated;
- repeated intake is idempotent and does not reread a completed message;
- Bifrost’s artifact writer protects a human-modified Intake artifact from
  overwrite;
- Prometheus pause blocks Bifrost and does not replay after resume;
- invocation receipts record outcomes without persisting connector output or
  credentials.

Repository verification:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./...
```

passed. The final repository pass also includes `go vet ./...` and a
production build before commit.

## Known limitations

- No live Gmail smoke or cutover was performed; the adapter has only controlled
  local connector evidence in this kernel.
- No download or installation engine is implemented. Bifrost records those
  approval boundaries and refuses use until they are explicitly represented.
- Scheduling is exposed as an hourly configuration contract and manual CLI
  operation; Pantheon’s final scheduler remains future work.
- Email interpretation remains downstream work for Present/Past/Apollo.
- Hermes remains the rollback path until a later, explicit migration proof.
- Bifrost is not a plugin marketplace, universal protocol adapter, Aphrodite,
  Project Fame, Project Purity, or Kernel 20 GUI.

## Recommended clean predecessor for Kernel 20

Kernel 19 leaves a governed, inspectable Bifrost boundary and a reversible
read-only mail migration candidate. The clean predecessor for Kernel 20 is
this repository-verified state with Hermes retained, no live mailbox mutation,
and no unbounded external capability acquisition.
