# Kernel 10 Reportback — Present and Normalization

## Status

Repository implementation and automated verification are complete at the reconciled Kernel 8–10 commit. The installed runtime is not being claimed as Kernel 10-live-verified in this report; the local Whisper path is implemented but not live-verified because no Whisper binary is currently available on PATH.

## A. Actual predecessor

- Branch: `trunk`.
- Exact reconciled predecessor: `f15c5b2` (`reconcile kernels 8 through 10`).
- Kernel 9 is a completed live PASS separately recorded in `docs/reportbacks/kernel-9-father-time-intake-ledger.md`; Kernel 10 must not inherit that live deployment claim.
- Kernel 10 cleanup corrected persisted `intake_scans.father_time_year` and populated nested `show intake` scan fields.

## B. Constitutional status

No constitutional amendment.

## C. Raw Vault

`paths.raw` is active. Its default is `<paths.vault>/80 Pantheon Raw`. Output is organized as `Text`, `Documents`, or `Transcripts`, followed by observation year/month and the Intake-relative path. Source extensions become `.md`; human filenames remain readable. Existing destination collisions receive an observation timestamp and, if necessary, an occurrence fragment. Raw artifacts are written with symlink-component checks and never overwrite an existing file.

## D. Processors

- Markdown and plain text: built-in, ready, body-preserving.
- PDF: local `pdftotext` when available; image-only/OCR-required PDFs remain blocked.
- DOCX: built-in ZIP/XML extraction for paragraphs, headings, lists, tables, and relationships-backed hyperlinks.
- Whisper: local configured binary only; currently blocked pending a local binary in this environment.
- Media conversion: local configured converter or local `ffmpeg`; currently available at `/usr/bin/ffmpeg` in this environment.

No processor sends source content to the network.

## E. Provenance and verification

Every artifact has Pantheon YAML frontmatter containing `occurrence_id`, `content_id`, source path, source type, observation time, normalization time, processor, processor version, and normalized-body SHA-256. Read-back verification checks the frontmatter identity, body hash, artifact readability, and generated artifact hash before the occurrence is marked `verified`.

The evidence chain is:

```text
Intake occurrence → content SHA-256 → Raw Markdown provenance → verified artifact → Present missive
```

## F. Retention

Successful verification transitions:

```text
awaiting_verified_markdown → retention_armed
```

`verified_at` becomes the eligibility time and `retention_due_at` is `verified_at + retention.raw_input_days`. The original Intake source remains in place. Kernel 10 performs no move, rename, archive, or deletion.

## G. Present

Supported interfaces are:

```text
pantheon present "request"
pantheon present --occurrence intake-...
pantheon present --new
pantheon show present
pantheon show processors
```

Recent deterministic references such as “latest file,” “latest recording,” “latest PDF,” and “latest document” use chronology and file metadata only. Exact occurrence processing bypasses reference resolution. Original requests are stored verbatim in missives; reasoning is reported as awaiting Apollo rather than answered by Present.

## H. Local-only boundary

Present invokes only local filesystem processors and local subprocesses. There is no cloud transcription, frontier processing, OCR fallback, image interpretation, or network upload.

## I. Whisper

The resolved configuration is `ingestion.whisper.enabled`, `binary`, `model`, and `language`; `language: auto` omits an English-only flag. Audio/video dispatch and local media conversion are implemented. Live Whisper verification remains blocked until a local Whisper executable and model are configured.

## J. Prometheus

Present, exact occurrence processing, and `--new` pass through the common pause gate. Read-only Present and processor inspection remain available while paused. Blocked work is not queued or replayed.

## K. Tests

Passed in the repository:

```text
GOCACHE=/tmp/pantheon-go-cache go test -run '^TestKernel10' ./...
GOCACHE=/tmp/pantheon-go-cache go test -run 'TestKernel10|TestKernel9|TestDaemonStatusReturnsAuthoritativeConfigAndAuditsRequest|TestDaemonNotifyUsesConfiguredHelper' ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel10 .
```

Focused tests cover faithful text body preservation, provenance, verification, retention arming and due-date calculation, idempotency, unsupported images, request preservation, pause refusal, configuration resolution, processor inspection, DOCX structure, conditional local PDF extraction, Raw-path traversal and symlink escape rejection, stale source identity rejection, filename collision protection, human-modified artifact protection, local Whisper dispatch, and missing-Whisper blocking. Live installed-runtime and Whisper smoke tests remain pending.

Coverage boundaries are explicit. The suite has focused coverage for PDF extraction when `pdftotext` is available, DOCX structure, path traversal prevention, symlink escape prevention, changed/stale source identity, exact occurrence processing, filename collisions, idempotent processing, human-modified Raw artifact protection, mocked/local Whisper dispatch, and missing Whisper processor behavior. It does not currently have focused coverage for `present --new` end-to-end, video audio-only handling, deterministic recent-source resolution, or a live Whisper installation. The exact occurrence primitive is exercised directly through occurrence IDs, and Present no-replay behavior is covered by the pause test.

## L. Files changed

- `present.go`: Raw Vault organization, local processors, verification, retention transition, Present missives, inspection, and pause-routed handlers.
- `configuration.go`: `paths.raw` and local ingestion processor configuration.
- `intake.go`: persisted scan presentation reconciliation and Kernel 10 occurrence metadata/schema migration.
- `daemon.go`, `main.go`, `welcome.go`: Present routes, CLI commands, inspection commands, and discovery help.
- `kernel10_test.go`: focused Kernel 10 tests.
- `config/pantheon.example.json`, `config/README.md`, `docs/configuration.md`: configuration documentation.

## M. Risks

PDF extraction fidelity depends on `pdftotext`; scanned PDFs remain blocked. DOCX extraction intentionally does not reproduce complex Word layout or tracked changes. Whisper requires explicit local setup and can be resource-intensive. Human edits to verified Raw artifacts are preserved and cause later verification refusal rather than silent overwrite.

## N. Expiry conditions

These claims expire if Raw path enforcement, processor invocation, artifact metadata, occurrence identity, retention transition, pause enforcement, or missive schema changes.

## O. Next safe successor

**Kernel 11 — Mercury Orders, Motes, and Seals.** Kernel 11 may consume verified Present missives and exact occurrence/artifact provenance. It must not replace the Kernel 9 identity chain or treat Present’s factual output as semantic reasoning.
