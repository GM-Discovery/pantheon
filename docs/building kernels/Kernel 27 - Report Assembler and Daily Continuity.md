# Kernel 27 — Report Assembler and Daily Continuity

## Mission

Turn verified Pantheon evidence into a durable, inspectable source set for
Apollo. Kernel 27 establishes the missing continuity seam after Kernel 26:

```text
verified event evidence
        ↓
  event assembly receipt
        ↓
  daily continuity assembly
        ↓
  Apollo source-isolated reasoning
        ↓
  cited report in 70 Pantheon Reports
```

The assembler is deterministic. It selects, validates, hashes, and records
Vault Markdown sources. It does not summarize, rank human importance, infer
identity, create authority, or execute calendar, email, or notification work.
Apollo remains the reasoning organ; Future and Mercury remain the authority
and execution boundaries.

## Why this kernel exists

Kernel 26 proved that Omi material can reach Intake, Present, and Apollo, but
the next report still depended on manually choosing a source set. That makes
daily continuity fragile and makes it easy for a report to mention an
unresolved request without naming it. Kernel 27 gives Apollo an explicit,
hash-checked source set and a durable handoff receipt.

## Two assembly modes

### Event assembly

Event assembly creates an evidence archive under:

```text
70 Pantheon Reports/Integrated Events/
```

It accepts one or more explicit Vault-relative Markdown sources, normally a
transcript or Present artifact plus an existing identity/event report. The
result records the full source paths, hashes, timestamps, and exact Apollo
citation tokens. It does not overwrite an existing assembly. A changed source
set creates a new revision identified by its source-set hash.

### Daily continuity assembly

Daily assembly creates a source manifest under:

```text
10 Operations/Situation Briefs/
```

When no explicit sources are supplied, it deterministically gathers verified
same-day Present artifacts, same-day Apollo reports, and same-day integrated
event reports. Explicit sources may be supplied when the operator has a
narrower bounded brief in mind. An empty source set is refused rather than
presented as a meaningful brief.

The output is an evidence manifest, not the final prose Situation Brief. A
future Zodiac-owned daily job may request the assembly and then create a
separate Apollo order. Zodiac still schedules; it does not compose prose.

## Durable contract

```text
POST /v1/reports/assemble
GET  /v1/reports/assemblies
GET  /v1/reports/assemblies/{assembly-id}
```

Example request:

```json
{
  "kind": "daily",
  "period_date": "2026-08-23",
  "source_refs": [
    "80 Pantheon Raw/Audio Transcripts/2026/08/example.md",
    "70 Pantheon Reports/Integrated Events/2026-08-23-event.md"
  ]
}
```

The write route requires the local Pantheon control token and is blocked by
Prometheus pause. The receipt contains:

- a stable assembly ID;
- event or daily kind and local operating date;
- the manifest path;
- a body SHA-256;
- every selected source path and exact `[[Vault/path.md]]` citation;
- every selected source SHA-256, byte count, and modification time;
- `ready_for_apollo` status; and
- duplicate/replay state.

Source references must remain beneath the configured Vault, contain regular
Markdown files, and pass the existing no-symlink safety checks. The manifest
never replaces the original source material.

## Idempotency and replay

The assembly identity is the SHA-256 of the mode, operating date, and sorted
source set. Repeating an identical request returns the existing receipt and
does not overwrite the manifest. Adding or removing a source produces a new
source-set revision. This makes a late-arriving Omi transcript visible without
silently mutating an earlier report input.

## Apollo handoff

The manifest is the deterministic handoff record. A separate Apollo order may
use the exact source references returned in the receipt. Apollo must still:

- name every concrete request, action, commitment, decision, unresolved item,
  and proposal in plain language;
- preserve full citation tokens;
- distinguish primary event evidence from continuity context; and
- stop or refer when the source set is insufficient.

Creating an assembly does not create an Apollo order, grant authority, issue a
Mercury Go, send a notification, write a calendar event, or send email. Those
remain separately inspectable boundaries.

## Operator surface

```text
pantheon report assemble --date 2026-08-23
pantheon report assemble --kind event --date 2026-08-23 \
  --source "80 Pantheon Raw/Audio Transcripts/2026/08/example.md"
pantheon report assemblies
```

The first command auto-collects same-day durable sources. The event form is
explicit by design. The report-assembly receipt and manifest are the proof of
selection; the eventual Apollo report is the proof of reasoning.

## Non-goals

Kernel 27 does not:

- install n8n or another gateway;
- replace Echo/Fame identity review;
- decide that a source is important because it is recent;
- create a calendar event or modify email;
- deliver a notification;
- prune Intake or delete raw material; or
- implement Archimedes frontier escalation.

Zeus and Archimedes remain independent future boundaries. Zeus begins with
retention inventory and dry-run sanitation; Archimedes begins with a redaction
manifest and curated outbound bundle.
