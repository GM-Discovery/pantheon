# Kernel 28 Reportback — Zodiac, Libra, and Configurable Reporting

Date: 2026-08-23
Status: PASS — bounded end-to-end implementation deployed

## Delivered

Panopticon now owns reusable Libra report recipes. A recipe selects either a
daily/weekly schedule or the verified external-Intake conversion event, its
timezone and evidence window, source families and optional Vault-relative path
prefixes, a minimum evidence threshold, required report sections, report-only
instructions, one or more durable Vault outputs, and an optional local success
notification. Scheduled recipes are materialized as linked Zodiac schedules;
intake-triggered recipes use a durable Libra ingestion job queue.

When a Libra schedule becomes due—or a completed external Intake conversion
queues an intake-triggered recipe—Pantheon now:

1. selects verified Vault sources in the configured window;
2. creates or reuses a deterministic report-assembly manifest;
3. creates an isolated Apollo order containing only the authorized source set;
4. dispatches Apollo through the existing Mercury seal/Go path;
5. requires Apollo to complete with valid citations;
6. writes and re-reads every selected Markdown output in the configured Vault;
7. records one output receipt per destination in `libra_report_outputs`;
8. optionally delivers a local notification; and
9. records the assembly, Apollo, Vault, notification, error, and replay state in
   `libra_report_runs`.

Missing evidence and incomplete Apollo responses become Needs You states. A
completed window is not replayed on the next Zodiac round.

## Interface

The Zodiac view in Panopticon now includes:

- a plain-language schedule explanation;
- Libra scheduled or after-Intake report recipe configuration;
- source-family, path-prefix, evidence-threshold, section, instruction, and
  multiple-output controls;
- pause/enable controls;
- last-run state and final report path; and
- a receipt-chain list showing assembly, Apollo, each Vault output, and
  notification state.

The GUI evaluate action uses the local Panopticon session boundary. It no
longer incorrectly requires the CLI control token.

Relevant routes:

```text
GET  /v1/gui/zodiac/constellations
GET  /v1/gui/zodiac/report-definitions
POST /v1/gui/zodiac/report-definitions
GET  /v1/gui/zodiac/report-runs
POST /v1/gui/zodiac/report-definitions/{id}/enable|disable
POST /v1/gui/zodiac/round
```

## Verification

- `go test ./...` — PASS (`pantheon/internal/pantheon`, 34.821s)
- `go vet ./...` — PASS
- Panopticon JavaScript syntax check — PASS
- focused Kernel 27/28 tests — PASS
- intake-triggered recipe with multiple Vault outputs — PASS
- daily and weekly fake-Apollo end-to-end proof — PASS
- notification helper receipt — PASS
- second Zodiac round produced zero duplicate Libra runs — PASS
- live Libra constellation, definitions, and run endpoints — PASS
- deployed binary and running service SHA-256: `44b2b62a3eb81b2640928ebbb412e16901e4c443a9e88cfc51f063875da21166`

The live installation currently has the reusable Libra constellation and the
existing `Daily Brief` and `Weekly Recap` scheduled recipes. The deployment did
not create another personal recipe or trigger a report; the live run endpoint
was empty at smoke-test time. The previous binary is recoverable at
`/tmp/pantheon-before-libra-recipes-20260823`.

## Boundaries retained

Libra does not execute calendar or email operations, install providers, run
arbitrary shell commands, or treat Apollo prose as authorization. The local
notification is an explicitly configured bounded copy after the verified
Vault write.
