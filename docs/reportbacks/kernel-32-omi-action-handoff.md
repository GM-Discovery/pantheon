# Kernel 32 Reportback — Omi Action Handoff

Status: PASS — repository implementation and focused tests complete

Kernel 32 closes the missing seam between Apollo's structured understanding
and Pantheon's existing authority machinery.

The new durable `action_drafts` table records typed, reviewable proposals with
their Apollo report/source references, payload, citations, state, and any
Future proposal ID. The handoff is replay-safe: the same report, action kind,
and payload resolve to the same action ID.

Completed Omi webhook deliveries and Rumor handoffs now enqueue a durable
action job. The local watcher runs Present and Apollo for that job without
requiring a manual Present/Apollo command. The watcher stops at Apollo's
report/action-draft boundary; it does not accept drafts or run Mercury.

Apollo now routes:

- complete meeting proposals to `calendar_event` drafts;
- complete email recommendations to `email_draft` drafts; and
- incomplete or untyped requests to `needs_clarification` drafts.

The operator can inspect these at `GET /v1/actions` or with
`pantheon show actions`. `pantheon actions accept <id>` requires the local
control token, validates the Seshat package, and creates a Future proposal.
It deliberately does not execute Mercury.

The existing Gmail mail adapter now recognizes the provider-neutral
`email.draft` capability as a non-sent Drafts operation. It remains behind the
existing Bifrost capability and approval checks and is reachable only through
a sealed Future/Mercury order. No send operation was added.

Calendar remains a local proposal boundary. The repository has no approved
calendar adapter installed, so accepting a calendar draft records that provider
availability is missing rather than installing or invoking one.

Focused verification:

```text
go test ./internal/pantheon -run Kernel32 -count=1
```

The full repository test, vet, build, deployment, and live service checks are
the final acceptance steps for this increment.
