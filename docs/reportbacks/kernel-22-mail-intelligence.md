---
kernel: 22
status: REPOSITORY VERIFIED — live Gmail mail round pending operator-run verification August 17, 2026
predecessor: 280a02327e6214c8e7e29d83c3504fbc298db6e4
---

# Kernel 22 Reportback — Mail Intelligence and Governed Action Pipeline

Kernel 22 extends the existing Kernel 19 Bifrost/Himalaya read path into a
manual, durable mail-understanding workflow. It preserves the read-only
`gmail-mail` capability and gives mailbox mutations a separate configured
capability identity, `gmail-mail-actions`. No credentials, mail contents, or
live Gmail result are claimed here.

## Implemented

- Canonical mail configuration declares the existing `gmail` account, `INBOX`,
  Himalaya adapter, newest-to-oldest bounded backlog, batch size,
  standing-authority-only mark-read policy, approval-required attachments, and
  the separate action capability identity.
- Bifrost supports explicit semantic mail search requests while retaining the
  existing source-preserving artifact and processed-message ledger. Explicit
  backlog searches do not advance the new-mail success watermark.
- Durable mail tables record runs, message identity and state, Apollo
  understanding, citations and provenance, deterministic rejection rules and
  events, backlog cursor state, and mailbox action receipts.
- New messages default to `unclassified`; deterministic rejection rules are
  inspectable, provenance-bearing, disableable, and retain rejected messages
  in Pantheon accounting.
- Mail understanding enters the existing Present → Past/Weave → local Apollo
  path. Apollo receives structured mail-analysis fields for classification,
  priority, people, reply-worthiness, requested action, commitments,
  deadlines, meetings, attachments, suspicion, uncertainty, citations, and
  merely suggested mailbox work. Suggestions do not authorize execution.
- Lawful mailbox work is represented as Seshat packages and Future proposals,
  then remains subject to authority, Janus where relevant, Mercury sealing and
  Go, and Bifrost capability approval. Mark-read, safe labels, move, and
  non-sent draft are bounded operations; send, delete, archive, forwarding,
  filter creation, and attachment download are not autonomous actions.
- The Panopticon now has a human-readable Mail surface and Home summary with
  explicit **Check mail now**, message understanding, deterministic filters,
  action receipts, backlog state, and unavailable/unknown handling. No Zodiac
  scheduler was added.
- Kernel 21 behavior remains intact, including Radio Prometheus, the Fates
  grouping, Current Work, Rumor, Configuration, navigation, and local serving.

## Hermes boundary

The existing Hermes gateway and its overlapping mail jobs were inspected but
not stopped, deleted, or rewritten. Kernel 22 has no scheduler and does not
claim a Hermes cutover. Any retirement requires a later human-reviewed proof
that the Pantheon path is live and sufficient.

## Focused verification

`internal/pantheon/kernel22_test.go` covers canonical mail defaults and
invalid-adapter rejection; deterministic rejection provenance, retained
accounting, and rule disablement; idempotent mail-ledger synchronization that
preserves Apollo understanding; separation of read-only capability metadata
from write authority and refusal of send/delete; Prometheus pause refusal with
no durable run and no replay on resume; and bounded, secret-free Mail API and
Panopticon delivery.

The focused suite passed:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./internal/pantheon -run Kernel22 -count=1
```

Tests that create loopback `httptest.Server` listeners require the host test
environment's network-listener permission; a restricted sandbox may reject
those listeners before application code runs.

## Live status

The live Gmail → Bifrost → Intake → Present → Apollo round has not been run
for this reportback. Therefore Kernel 22 is repository-verified, not live
PASS. After the backend is installed, the explicit bounded path is:

```text
pantheon mail status
pantheon mail check
pantheon mail messages
pantheon show future
pantheon show mercury
```

The action-capability record must be separately registered, approved, and
enabled before any suggested mailbox mutation can lawfully execute. A mail
check alone never sends or deletes mail.

Kernel 23 / Zodiac is not started by this work.
