# Kernel 8 — Local Intake and Whisper Preparation Implementation Plan

> **For Hermes:** Use fresh task-scoped agents and independent review; do not use one long-lived coding context as the source of authority.

**Goal:** Turn `/home/brick/Pantheon Intake` into a safe, local-only, human-controlled intake boundary for raw files, beginning with a manually invoked Whisper transcription pilot.

**Architecture:** The intake directory remains an owner-only source staging area. Pantheon records a manifest/fingerprint and writes derived outputs outside the raw intake directory. Local Whisper receives only explicitly selected supported audio files. A transcript is a cited artifact, not a task list, diagnosis, personal conclusion, or execution instruction. No watcher, overnight intake processing, external upload, source deletion, or source mutation is in scope for Kernel 8.

**Tech stack:** Go runtime, SQLite audit/state, local filesystem, whisper.cpp (`/home/brick/whisper.cpp/build/bin/whisper-cli`), Whisper large-v3 (`/home/brick/whisper.cpp/models/ggml-large-v3.bin`), systemd user service only for the existing daemon.

---

## Non-negotiable boundaries

- `Pantheon Intake` is raw source material. Never alter, move, rename, archive, or delete its files automatically.
- No recursive automatic ingest and no filesystem watcher in Kernel 8.
- No external model, email, notification, Mercury action, or Apollo interpretation receives raw intake/transcript material.
- Only a local CLI-control-token protected command may write derived intake state.
- Audio selection must be explicit by filename or a bounded manifest selection; no shell command may come from model output.
- Pause blocks any intake/transcription action; resume never replays skipped work.
- Derived transcript output retains source path, source SHA-256, creation time, Whisper model/binary version/path, language/configuration, and a direct raw-source provenance reference.
- Present records factual processing state only. It must not infer urgency, beliefs, tasks, diagnosis, or personal meaning.

## Kernel 8 acceptance criteria

1. `pantheon intake status` reports only deterministic local facts: supported/discovered files, fingerprints, already-derived outputs, and whether Pantheon is paused.
2. `pantheon intake transcribe <relative-audio-path>` is CLI-token protected, rejects traversal/symlinks/unsupported types, and runs a fixed local whisper.cpp argument list.
3. It writes a local derived transcript and machine-readable manifest under `state/intake/`, never under the raw intake directory.
4. It audits `started`, `completed`, `failed`, and `blocked` outcomes before/after the bounded action.
5. A same-file/same-fingerprint request is idempotent: it reports an existing derived result rather than silently re-running, unless an explicit future `--force` policy is separately approved.
6. Pause, traversal rejection, symlink rejection, unsupported type, nonzero Whisper exit, and missing binary/model are all regression-tested.
7. One manually supplied short audio fixture or user-selected real non-sensitive sample is transcribed locally as an end-to-end proof.

---

## Task 1: Write the Kernel 8 contract and intake configuration

**Objective:** Make the raw/derived boundary and allowed local transcription parameters reviewable before code exists.

**Files:**
- Create: `config/intake-policy.yaml`
- Create: `docs/reportbacks/kernel-8-local-intake-whisper.md`
- Modify: `README.md` only if current capability wording needs correction.

**Policy contents:**
```yaml
raw_intake_root: /home/brick/Pantheon Intake
allowed_extensions: [.wav, .mp3, .m4a, .flac, .ogg, .mp4]
derived_state_root: state/intake
watcher_enabled: false
automatic_transcription_enabled: false
external_upload_enabled: false
source_mutation_enabled: false
whisper:
  binary: /home/brick/whisper.cpp/build/bin/whisper-cli
  model: /home/brick/whisper.cpp/models/ggml-large-v3.bin
```

**Verification:** Configuration parses in a Go unit test. The reportback begins with `In progress` and explicitly names the deferred watcher/overnight-ingestion work.

## Task 2: Create a safe intake-file enumerator

**Objective:** Discover raw files without reading content or escaping intake-root bounds.

**Files:**
- Create: `intake.go`
- Create: `intake_test.go`

**Tests first:**
- Include regular supported audio files only.
- Ignore `README.md`, unsupported extensions, directories, and symlinks.
- Reject a caller-supplied `../` or absolute relative path.
- Fingerprint contents with SHA-256 through a no-follow regular-file boundary.

**Implementation:** Reuse/improve the existing Vault safe-open pattern where appropriate, but do not assume the intake root is a Vault directory. Return relative path, byte length, mtime, extension, and SHA-256. Do not derive semantic metadata.

**Verification:** Focused RED/GREEN test cycle, then `go test -count=1 ./...`.

## Task 3: Add read-only intake status

**Objective:** Allow inspection before processing.

**Files:**
- Modify: `daemon.go`
- Modify: `main.go`
- Create: `intake_status_test.go`

**Contract:**
```text
GET /v1/intake/status
pantheon intake-status
```

Status returns a capped list of safe metadata plus derived-state presence. It must never return raw audio bytes, transcript bodies, or invoke Whisper.

**Verification:** Temporary intake root fixture proves symlink exclusion, cap behavior, and no write to raw directory.

## Task 4: Create local Whisper adapter with fixed arguments

**Objective:** Isolate process invocation behind an injectable deterministic adapter.

**Files:**
- Create: `whisper_local.go`
- Create: `whisper_local_test.go`

**Tests first:**
- Missing executable/model fails before process start.
- Supported source path yields an exact argument list assembled by Go, not model output.
- A fake local binary can emit a transcript fixture.
- Nonzero exit returns a bounded error and writes no completed manifest.

**Implementation:** Invoke only the configured absolute Whisper binary and model path. Use a controlled derived temporary directory. Do not use `sh -c`, shell interpolation, or external network access.

## Task 5: Add protected single-file transcription

**Objective:** Create the first bounded write action over explicit raw audio.

**Files:**
- Modify: `daemon.go`
- Modify: `main.go`
- Create: `intake_transcribe_test.go`

**Contract:**
```text
POST /v1/intake/transcribe
pantheon intake-transcribe <relative-audio-path>
```

**Required order:**
1. Validate CLI control token.
2. Check persisted pause state; audit/refuse if paused.
3. Validate intake-relative path, no symlink, extension, file-size cap, and regular file.
4. Write required `started` audit record.
5. Fingerprint source and check existing derived manifest.
6. Run local Whisper through fixed adapter.
7. Atomically write transcript Markdown/plaintext plus JSON manifest under `state/intake/transcripts/` with mode `0600`.
8. Record completed/failed audit result.

**Output:** Include source-relative path and source SHA-256. Do not create an Obsidian note yet.

**Verification:** Prove raw file hash and timestamp are unchanged after successful and failed runs.

## Task 6: End-to-end manual proof and reportback

**Objective:** Verify the actual installed local Whisper path, not just fake adapters.

**Procedure:**
1. User places a short, non-sensitive audio file in `/home/brick/Pantheon Intake`.
2. Confirm intake status sees it as supported metadata.
3. Invoke one explicit `pantheon intake-transcribe <file>` command.
4. Inspect audit record, transcript, manifest, source fingerprint, and raw-file metadata.
5. Run again to prove idempotent no-op behavior.
6. Pause Pantheon and prove a transcription request is blocked; resume without replay.

**Commit gate:** `gofmt`, full tests, `go vet`, build, service restart/status/socket verification, static scan, independent review, readable reportback, explicit commit approval.

---

# Software Roadmap After Kernel 8

## Kernel 9 — Present: bounded factual intake records

Turn successful derived transcripts/doc extractions into local factual records with provenance and correction status. No task extraction, behavioral conclusion, relation inference, or automatic Vault mutation. Introduce human review of a record before it can be promoted into other contexts.

## Kernel 10 — Situation Brief comparison path

Build a local-first proposed daily brief from approved Present records, bounded Gmail envelope metadata, and Weave retrieval. Keep the current Hermes Situation Brief as fallback. Compare outputs manually before replacing any existing scheduled brief. No email sending.

## Kernel 11 — Thin Obsidian integration

Expose existing local derived reviews/status in Obsidian without moving authority into a plugin. The plugin/UI is read-only/review-oriented: source links, provenance, corrections, and explicit promote/suppress controls. It must not become a policy or identity authority.

## Kernel 12 — Mercury bounded execution proposals

Represent local reasoning output as reviewable proposed actions with exact scope, grant requirement, pre-action audit, and a human-readable diff. Begin with non-send email drafts/tags only if separately granted. Never permit generic shell execution from model output.

## Kernel 13 — Archimedes de-identified escalation packets

Implement a local packet builder that strips raw identity, Vault paths, transcript bodies, and direct email content. Frontier reasoning receives only minimum-necessary de-identified material and returns recommendations. Pantheon remains the authority.

## Deferred: automatic overnight intake

Do not schedule a watcher or bulk Whisper processing merely because a raw intake directory exists. Consider it only after Kernel 8 proves single-file processing, manifest recovery, pause semantics, resource limits, storage retention, and user review behavior. A future schedule must include: bounded batch count/CPU budget, backlog visibility, no replay after pause, and explicit user approval.

---

# Coding Method: Durable Context Without Long-Session Dependence

## Keep the authority in versioned artifacts

Use the repository—not an accumulated chat—as the primary coordination surface:

- `docs/architecture/` for invariants and trust boundaries.
- `docs/kernels/` or reportbacks for completed capability evidence and explicitly deferred work.
- `config/` for reviewable declarative policy, never hidden model memory.
- `.hermes/plans/` for a one-kernel implementation plan.
- tests for non-negotiable behavior.

A new conversation should need only: the current kernel plan, `README.md`, the latest reportback, `git log -1`, `git status`, and the relevant tests/config—not five compactions of chat history.

## Use short-lived coding cells

For each task: fresh implementation context → focused test → independent review context → commit. Do not let the implementation agent also be the final authority on its correctness. The committed reportback is the handoff.

## Maintain a small human-owned Kernel Brief

Before each kernel, write one page containing: goal, non-goals, boundaries, acceptance tests, exact human decisions required, and completion gate. It should be enough for a fresh agent to resume safely.

## Treat persistent memory as preferences, not project state

Keep only durable personal preferences and safety boundaries in Hermes memory. Put changing architecture, task progress, paths, counts, and implementation choices in the repo. This reduces context bleed and makes model/provider changes survivable.

## Use compaction as a handoff test

If a compaction makes the next agent unable to proceed safely, the missing fact belongs in a plan, config, reportback, test, or architecture document—not in more chat history.
