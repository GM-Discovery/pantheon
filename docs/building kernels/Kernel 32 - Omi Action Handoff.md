---
kernel: 32
status: PASS — Apollo action drafts and explicit Future handoff implemented
predecessor: Kernel 26 — Omi Intake Bridge; Kernel 28 — Libra Reporting
---

# Kernel 32 — Omi Action Handoff

## Purpose

Connect the verified Omi/Apollo path to reviewable work without treating
Apollo's interpretation as authority.

The intended path is:

```text
Omi whisper
  → Intake / Present
  → Apollo report
  → typed action draft
  → person accepts
  → Future proposal
  → authority / Mercury / provider boundary
```

## Implemented increment

Apollo's existing structured fields now produce durable `action_drafts`:

- `meeting_proposal` becomes a `calendar_event` draft;
- `draft_recommendation` becomes an `email_draft` draft;
- missing required fields become `needs_clarification`; and
- a plain `requested_action` without a typed payload is retained as a
  clarification draft rather than guessed into an external operation.

Completed Omi webhook deliveries and native Rumor-to-Intake handoffs also
enqueue a durable local Apollo job. The action watcher processes that queue
automatically when Prometheus is not paused and the configured local Apollo
endpoint is available. A failed local run is retried up to three times and is
then retained as failed work for inspection.

Each draft preserves its Apollo report ID/path, source occurrence when known,
source citations, payload, state, and any Future proposal reference. The action
ID is deterministic for the same report, kind, and payload, so an Apollo replay
does not create duplicate drafts.

## Operator surface

```text
GET  /v1/actions
GET  /v1/actions/{id}
POST /v1/actions/{id}/accept

pantheon show actions
pantheon actions show <id>
pantheon actions accept <id>
```

Listing and inspection are read-only. Accepting requires the local control
boundary. Acceptance validates the matching Seshat package and creates a
Future proposal; it does not start a Mercury round.

## Safety boundary

Apollo does not send email, write a calendar, install a provider, create a
grant, seal an order, or issue Go. A calendar draft remains provider-neutral
until an approved calendar package and adapter exist. A generic email draft
uses the existing `EMAIL:DRAFT` package and `email.draft` capability, but still
requires the normal Future authority, Bifrost approval, and explicit Mercury
round before a non-sent draft can be written to a mailbox. Email sending is not
implemented.

The generic email adapter creates a non-sent Drafts message only when the
complete authority chain is later exercised. It never sends the message.

## Verification

Focused Kernel 32 tests cover:

- typed calendar and email routing from one Apollo result;
- deterministic replay without duplicate action drafts;
- incomplete calendar proposals becoming clarification work;
- accepting a complete email draft into Future and stopping at missing
  authority; and
- the action inspection and control-token routes.

The live deployment must continue to report action drafts as proposals and
must not claim that a calendar provider is installed merely because a calendar
draft exists.
