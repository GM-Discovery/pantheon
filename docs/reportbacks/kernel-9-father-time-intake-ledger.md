# Kernel 9 Reportback — Father Time and Intake Ledger

## Status

**PASS — repository and live runtime verified August 15, 2026.**

## A. Actual predecessor

- Branch: `trunk`, with pre-existing review-candidate changes preserved.
- Exact committed predecessor: `7373adb` (`feat: schedule nightly Weave maintenance`).
- Kernel 8 reportback: `docs/reportbacks/kernel-8-configuration-prometheus.md`.
- Kernel 8 cleanup performed: the single `welcome.go` implementation now retains the Prometheus safety graphic and adds a concise current command list. The stale Kernel 7 statement about no nightly refresh was explicitly marked historical; commit `7373adb` is identified as the later change that added scheduled Weave maintenance. No Weave behavior was refactored.
- Live runtime deployment succeeded. The first live Intake scan succeeded, Father Time wrote the 2026 Markdown chronicle, and the scan completed with zero failed scans and zero pending chronology writes.

## B. Constitutional status

No constitutional amendment.

## C. Configuration

Kernel 9 activates:

```text
paths.intake
paths.father_time_chronicles
retention.raw_input_days
time.timezone
```

Defaults are `~/Pantheon Intake`, `<vault>/00 Pantheon Core/Chronicles/Father Time/`, `30`, and the machine-local timezone. An absent default Intake directory is created only by an explicit scan. A configured missing custom Intake path is reported and not created. Explicit timezones use IANA names and invalid names fail validation. Retention days must be between 1 and 36500.

Legacy Kernel 8 configuration remains readable and is not rewritten automatically. Kernel 9 defaults are applied in memory. The current installed legacy configuration resolves to:

```text
paths.intake: /home/brick/Pantheon Intake (default)
paths.father_time_chronicles: /home/brick/Documents/Obsidian Vault/00 Pantheon Core/Chronicles/Father Time (default beneath paths.vault)
retention.raw_input_days: 30 (default)
time.timezone: Local (machine local)
```

The live scan verified recursive Intake traversal. Hidden and symlink test material did not become occurrences. Two files containing identical bytes received distinct occurrence IDs while sharing one SHA-256 content identity. The second scan returned `new: 0` and `unchanged: 4`.

## D. Intake ledger

SQLite tables in the existing Pantheon machine-state database separate `intake_contents` from `intake_occurrences`. Content identity is `sha256:<64 lowercase hex characters>`. Occurrences receive independent random local IDs such as `intake-...`; identical bytes uploaded at separate occurrences therefore share content identity without sharing occurrence identity.

The scanner preserves slash-normalized relative paths, tracks `present` and `departed`, updates `last_seen_at` for unchanged files, creates a new occurrence for changed content at a path, and creates another occurrence when a departed path reappears. Recent occurrence inspection is deterministic and chronological. The read-only `GET /v1/intake` surface accepts bounded `limit` plus exact `path`, `extension`, and `type` filters for Kernel 10 retrieval.

## E. Father Time

Observation timestamps are RFC3339 with an offset. The configured or machine-local timezone selects the calendar year and yearly Markdown chronicle. The first event for a year creates `<chronicle-root>/<year>.md`; prior years are untouched.

The chronicle is ordinary Obsidian-readable Markdown. A representative entry is:

```markdown
## 2026-08-15

<!-- pantheon-event: intake-... -->
- 09:44:12 -07:00 — Intake occurrence `intake-...`
  - Source: `MFPI/meeting-recording.m4a`
  - Content: `sha256:...`
  - Size: 123456 bytes
  - Type: audio/mp4
  - State: observed
  - Retention: awaiting verified Markdown
```

Event markers make repair idempotent. A failed chronicle write leaves the occurrence and pending event in SQLite; a later scan repairs it without creating another occurrence or duplicate entry.

## F. Retention

Raw retention defaults to 30 days. New occurrences begin as `awaiting_verified_markdown` with null `retention_eligible_at` and `retention_due_at`. The 30-day clock has not started. Kernel 9 performs no move, archive, or deletion.

## G. Prometheus

`POST /v1/intake/scan`, exposed as `pantheon intake scan`, checks the common pause gate before doing any work. While paused it returns a refusal and writes an audit event; it does not create a scan or queue replay. `pantheon show intake` and `pantheon show father-time` remain read-only and available while paused.

Live Prometheus verification caused `pantheon intake scan` to refuse with HTTP 503. After resume, the blocked scan did not replay and the latest scan ID remained unchanged. Raw inputs remained `awaiting_verified_markdown`; Kernel 9 armed no retention clock.

No model, network call, Apollo request, Mercury order, or frontier provider is involved.

## H. Filesystem safety

Scanning is recursive and accepts only ordinary regular files. Hidden files/directories, symlinks, and special objects are skipped without following or ingesting them. Candidate files are statted, streamed through SHA-256, and statted again; changes during inspection are deferred. Unreadable files are failed and left untouched. Large files are streamed in 128 KiB chunks.

The Father Time path rejects symbolic-link components before creating directories or files. Source Intake files are never renamed, copied, moved, normalized, interpreted, or deleted.

## I. Kernel 8 cleanup

The source-built bare command was verified to show both the Prometheus kill-switch language and useful command help. `rg` finds exactly one `printWelcome()` implementation, in `welcome.go`.

## J. Tests

Passed:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel9 .
```

The focused `TestKernel9...` suite covers default-path portability, configuration validation, recursive traversal, hidden and symlink skipping, SHA-256, large-file streaming behavior, content/occurrence separation, idempotent rescans, changed/departed/reappeared paths, unstable-file deferral, chronicle repair, retention non-arming, pause refusal, read-only inspection, deterministic metadata filtering, and no replay.

Source-built and live read-only checks verified `pantheon show config`, the bare welcome presentation, the live scan, the 2026 chronicle, pause refusal, resume no-replay behavior, zero failed scans, and zero pending chronology writes.

## K. Files changed

- `intake.go`: SQLite ledger schema, safe recursive scan, streaming hash, occurrence reconciliation, Father Time writing/repair, and inspection data.
- `configuration.go`: active Kernel 9 fields, defaults, validation, migration rendering, and resolved inspection.
- `daemon.go`: daemon-routed scan and read-only inspection endpoints, including bounded occurrence filters.
- `main.go`: `intake scan`, `show intake`, and `show father-time` commands.
- `welcome.go`: useful command list beneath the Prometheus safety presentation.
- `kernel9_test.go`: focused Kernel 9 automated tests.
- `config/pantheon.example.json`, `config/README.md`, `docs/configuration.md`: configuration reference and examples.
- `docs/reportbacks/kernel-7-weave-human-reviewed-concepts.md`: explicit historical reconciliation.

## L. Risks and unresolved work

- A stale scan lock left by a crashed process requires operator inspection; it is intentionally not auto-removed by assuming a lock is stale.
- External drives remain local filesystem dependencies; Kernel 9 records availability failures and never destroys source material.
- The legacy configuration cannot persist custom Kernel 9 fields until a human explicitly applies schema migration.
- Chronicle reconciliation is marker-based and intentionally does not rewrite unrelated human edits.
- Kernel 10 must consume this occurrence identity system, produce and verify durable Markdown, and only then arm retention eligibility.

## M. Expiry conditions

These claims expire if the ledger schema, occurrence identity rules, chronicle format, Intake movement policy, retention activation, timezone resolution, Father Time storage, or pause enforcement changes.

## N. Next safe successor

**Kernel 10 — Present and Normalization.** Kernel 10 may assume Father Time can identify and chronicle Intake occurrences. It owns local interpretation, transcription/extraction, durable Markdown production and verification, Present missives, and the later transition from `awaiting_verified_markdown` to lawful retention eligibility.
