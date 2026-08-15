# Kernel 9 — Father Time and Intake Ledger

**Status:** Proposed implementation order  
**Constitutional authority:** Constitution of Pantheon v0.2  
**Roadmap position:** Kernel 9  
**Predecessor:** Actual committed/runtime state following Kernel 8  
**Expected successor:** Kernel 10 — Present and Normalization

---

# 1. Purpose

Kernel 9 gives Pantheon deterministic knowledge of material intentionally placed into its Intake.

After Kernel 9, Pantheon shall be able to answer:

> What entered Pantheon?

> When did it enter?

> Where did it enter?

> Is this content identical to something already seen?

> Is this the same Intake occurrence or a new occurrence?

> What retention policy will eventually apply?

> Which yearly Father Time chronicle records the event?

Kernel 9 does **not** interpret, transcribe, normalize, summarize, move, archive, or delete Intake material.

Its job is to establish identity, chronology, provenance, and safe ledger state.

---

# 2. Why This Kernel Comes Now

Kernel 8 established:

- versioned configuration;
- configuration inspection and migration;
- reserved namespaces for Intake, retention, and time;
- Prometheus Protocol Zero;
- current pause enforcement;
- the requirement that successor Kernels explicitly activate their own configuration.

Kernel 10 will need a trustworthy answer to:

> Which file is “the recording I just uploaded”?

Before Present can reason about or normalize anything, Pantheon needs deterministic Intake identity.

Father Time comes first.

---

# 3. Kernel 8 Cleanup Sweep

Before beginning substantive Kernel 9 work, perform a narrow reconciliation of Kernel 8's live CLI presentation.

This is not a redesign of Kernel 8.

## 3.1 Bare Pantheon entry

The current terminal welcome should preserve the new Prometheus kill-switch graphic **and** restore a concise useful list of available commands beneath it.

The bare command:

```text
pantheon
```

should not leave a first-time user with only the emergency control graphic and no indication of what Pantheon can otherwise do.

Keep:

- the terminal-friendly kill-switch presentation;
- the explanation that the command pauses Pantheon and autonomous activity for inspection;
- the statement that the person does not need to run it now;
- ANSI color when supported;
- `NO_COLOR` compatibility if already implemented.

Restore a concise current-command/help surface below it.

Do not turn this into Panopticon.tower.

## 3.2 No duplicate welcome implementation

Ensure there is only one `printWelcome()` implementation.

Prefer a small dedicated presentation file if the current repository already moved the function there.

Do not leave duplicate or dead versions in `main.go`.

## 3.3 Runtime reconciliation

After Kernel 9 implementation, ensure the installed runtime binary actually reflects the reviewed source.

The final reportback must distinguish:

- repository build success;
- installed runtime deployment;
- live service verification.

## 3.4 Kernel 7 / Kernel 8 history

Inspect the actual predecessor history.

If the Kernel 8 predecessor commit contains nightly Weave maintenance while the inherited Kernel 7 reportback still says no nightly refresh is enabled, explicitly reconcile that stale historical claim in the appropriate reportback/documentation.

Do not rewrite history.

Do not make unrelated Weave changes.

---

# 4. Configuration Activated by Kernel 9

Kernel 9 shall extend the configuration architecture established in Kernel 8.

Activate at minimum:

```text
paths.intake
paths.father_time_chronicles

retention.raw_input_days

time.timezone
```

The repository may choose equivalent nested Go field names if they remain consistent with the established human configuration.

## 4.1 Intake path

Default:

```text
~/Pantheon Intake
```

The path must be configurable.

Pantheon may create the default Intake directory if absent when explicitly initializing/scanning Intake, provided this is local and deterministic.

Do not silently create arbitrary custom configured paths if doing so could conceal a typo.

## 4.2 Father Time chronicle path

Default should resolve beneath the configured Vault to a human-readable Pantheon location such as:

```text
00 Pantheon Core/Chronicles/Father Time/
```

Each year receives its own Markdown chronicle:

```text
2026.md
2027.md
2028.md
```

The exact safe path may follow the existing Vault structure if the repository already has an established convention.

## 4.3 Raw retention policy

Default:

```text
30 days
```

This is a **policy value only** in Kernel 9.

It does not authorize deletion.

It does not begin a deletion countdown merely because an Intake occurrence exists.

## 4.4 Timezone

If explicitly configured, use a valid IANA timezone.

If unset, use the machine's local timezone.

Father Time stores full offset-aware RFC3339 timestamps.

The configured/local timezone determines the calendar year of the active Father Time chronicle.

Invalid explicitly configured timezone shall fail validation clearly.

---

# 5. Intake Scope

Kernel 9 shall recursively scan the configured Intake tree.

Subdirectories are allowed.

Relative paths beneath Intake shall be preserved in the ledger.

Example:

```text
~/Pantheon Intake/
    meeting.m4a
    MFPI/
        notes.pdf
    Victory Theater/
        sketch.png
```

All three ordinary files are eligible Intake objects.

Kernel 9 does not need to understand their semantic type.

---

# 6. Eligible Files

Kernel 9 shall be deliberately format-agnostic.

Accept ordinary regular files.

The ledger may record best-effort metadata such as:

- filename;
- extension;
- size;
- detected MIME type where deterministic and inexpensive.

Understanding the file's content belongs to Kernel 10.

## Skip

Do not ingest:

- symbolic links;
- sockets;
- device files;
- named pipes;
- other non-regular filesystem objects;
- hidden files;
- hidden directories;
- obvious operating-system metadata files where already recognizable.

Do not follow symlinks out of Intake.

A skipped object should be reportable without becoming an Intake occurrence.

---

# 7. File Stability During Scan

Pantheon must avoid recording partially copied files as completed Intake occurrences.

For each candidate file:

1. stat the file;
2. hash/read it;
3. stat it again;
4. confirm relevant size/modification state did not change during the read.

If the file changed during inspection:

- do not create a completed occurrence;
- report it as unstable/deferred;
- leave it untouched;
- allow a later scan to retry.

Do not add arbitrary sleeps merely to guess that copying has finished.

---

# 8. Content Identity

Father Time shall use SHA-256 as the initial deterministic content fingerprint.

Represent content identity unambiguously, for example:

```text
sha256:<64-hex-character-digest>
```

The SHA-256 identifies content.

It does not by itself identify the historical occurrence.

Two files with identical bytes share the same content identity.

---

# 9. Occurrence Identity

Pantheon must distinguish:

**content**

from:

**an occurrence of that content entering Intake**

Example:

The same PDF is uploaded in August 2026 and again in February 2027.

The SHA-256 is identical.

These are two Intake occurrences.

Each occurrence shall receive its own stable local occurrence ID.

The exact identifier format may follow repository conventions, but it must not rely solely upon the SHA-256.

---

# 10. Idempotent Re-Scanning

Repeated scans of the same unchanged file still present at the same Intake relative path shall not create duplicate occurrences.

Pantheon shall maintain enough machine state to distinguish:

```text
same file still sitting here
```

from:

```text
this content entered Intake again later
```

Recommended operational behavior:

- current known occurrence remains `present`;
- each scan updates `last_seen_at`;
- an item absent from a complete later scan may be marked `departed`;
- if the same content later reappears after departure, it may become a new occurrence.

A changed file at the same path is new content and therefore a new occurrence.

Do not silently overwrite historical occurrence state.

---

# 11. Intake Ledger

Create a deterministic local operational ledger.

SQLite within Pantheon's existing machine-state database is acceptable and preferred unless repository structure strongly indicates otherwise.

The ledger should separate content identity from occurrence identity.

A reasonable model includes:

## Content

```text
content_id / sha256
size
best-effort type metadata
first_seen_at
```

## Occurrence

```text
occurrence_id
content_id
relative_path
original_filename
first_seen_at
last_seen_at
present/departed state
Father Time year
processing state
retention state
```

Do not over-normalize merely for database elegance.

The implementation should remain understandable.

---

# 12. Retention State

Kernel 9 shall encode the distinction between:

> a policy exists

and:

> deletion is now lawful.

For a newly observed raw Intake occurrence:

```text
retention_policy_days: 30
retention_state: awaiting_verified_markdown
retention_eligible_at: null
retention_due_at: null
```

Equivalent naming is acceptable.

The critical rule is:

> **The 30-day purge clock does not begin when the raw file enters Intake.**

A later Kernel must first produce and verify the durable Markdown artifact in the Vault.

Only after that verification may the raw material become eligible for timed retention.

---

# 13. Future Zeus Rule Reserved Here

Kernel 9 does not implement Zeus.

It shall, however, preserve state capable of supporting the later deterministic lifecycle:

```text
raw Intake/source material
        ↓
durable Markdown produced
        ↓
Markdown verified in Vault
        ↓
retention becomes eligible
        ↓
Father Time establishes due date
        ↓
Future produces maintenance order
        ↓
Mercury seals lawful purge order
        ↓
Zeus receives explicit command
        ↓
Zeus deletes only the named authorized target
```

Zeus never decides that the Markdown is good enough.

Zeus never decides that an object is trash.

Zeus deletes only after receiving a lawful explicit command.

No portion of that deletion path is implemented in Kernel 9.

---

# 14. Father Time Machine State

Father Time shall maintain efficient deterministic operational state.

This may include:

- content hashes;
- occurrence IDs;
- scan timestamps;
- first/last seen times;
- local year;
- presence state;
- retention state.

This is working machine state.

It does not replace the human-readable chronology.

---

# 15. Father Time Human Chronicle

Father Time shall also produce a human-readable yearly Markdown chronicle in the Vault.

Example structure:

```markdown
# Father Time — 2026

## 2026-08-15

- 09:44:12 -07:00 — Intake occurrence `intake-...`
  - Source: `MFPI/meeting-recording.m4a`
  - Content: `sha256:...`
  - Size: ...
  - State: observed
  - Retention: awaiting verified Markdown
```

Exact formatting may be improved for readability.

The chronicle shall be:

- ordinary Markdown;
- inspectable in Obsidian;
- append-oriented;
- deterministic enough to avoid duplicate entries;
- human-readable without requiring Pantheon.

Do not dump unnecessary machine telemetry into the chronicle.

The chronicle records meaningful temporal events.

SQLite retains operational detail.

---

# 16. Annual Regeneration

Father Time conceptually regenerates each calendar year.

Kernel 9 shall implement this simply.

No theatrical daemon restart is required.

The first qualifying event in a new configured/local calendar year writes to:

```text
<chronicle-root>/<new-year>.md
```

If absent, create the new yearly chronicle safely.

Prior yearly chronicles remain untouched.

The annual boundary is therefore visible and durable without requiring special January 1 execution.

Future kernels may add ceremonial or scheduled behavior if useful.

---

# 17. Chronicle Write Safety

Writing the Father Time chronicle is a local durable-state operation.

Use safe file handling.

If the chronicle cannot be written:

- do not delete or move source material;
- do not pretend the occurrence is fully chronicled;
- preserve the machine-ledger occurrence;
- mark the chronology write as pending/failed;
- report the problem;
- allow a later scan/repair to retry without creating a duplicate occurrence.

Chronicle repair must be idempotent.

---

# 18. Intake Scan

Implement a deterministic explicit command.

Preferred conceptual interface:

```text
pantheon intake scan
```

The command should run through the daemon/control architecture rather than create an unrelated duplicate runtime where practical.

It shall:

1. respect Prometheus pause;
2. load resolved Kernel 9 configuration;
3. recursively enumerate Intake;
4. skip ineligible objects;
5. verify candidate stability;
6. hash new/changed content;
7. reconcile existing occurrence state;
8. create new content/occurrence records when appropriate;
9. update Father Time chronology;
10. audit the scan;
11. return a concise result.

Example:

```text
Pantheon Intake

new:       3
unchanged: 8
departed:  0
deferred:  1
skipped:   2

Father Time: 2026
Chronicle: .../Father Time/2026.md
```

No model is invoked.

---

# 19. Intake Inspection

Provide simple pre-Panopticon inspection.

Preferred conceptual commands:

```text
pantheon show intake
pantheon show father-time
```

Equivalent command grammar is acceptable if current CLI conventions strongly favor another shape.

## `pantheon show intake`

Should show useful current operational state such as:

- Intake path;
- latest scan;
- number currently present;
- recent occurrences;
- deferred/failed items;
- retention state.

## `pantheon show father-time`

Should show:

- configured timezone;
- current local Father Time year;
- current chronicle path;
- most recent chronological events;
- pending chronicle-write failures if any.

Do not build a full dashboard.

Kernel 21 remains Panopticon.tower.

---

# 20. Recent Occurrence Retrieval

Kernel 9 shall provide deterministic chronological retrieval sufficient for Kernel 10 to resolve references such as:

> the file I just uploaded

> the meeting recording I just uploaded

Kernel 9 does not interpret those natural-language phrases.

It provides the data needed to do so.

At minimum, internal/API retrieval should support:

- newest occurrence;
- recent N occurrences;
- newest occurrence matching basic deterministic metadata such as extension/type/path where available.

Sort primarily by Father Time observation time.

Do not involve Apollo.

---

# 21. Audit

Record scan-level audit evidence.

At minimum:

- scan start/end;
- configured Intake path;
- number new;
- number unchanged;
- number departed;
- number deferred;
- number skipped;
- number failed;
- chronology result.

Avoid duplicating entire source content or giant file metadata into the audit table.

The Intake ledger and Father Time chronicle are the detailed state.

---

# 22. Prometheus Boundary

Kernel 9 Intake scanning is consequential local processing and shall respect the common pause gate established in Kernel 8.

While Pantheon is paused:

```text
pantheon intake scan
```

must refuse to begin new processing.

Read-only inspection may remain available:

```text
pantheon show intake
pantheon show father-time
```

Do not queue the blocked scan for automatic execution after resume.

The person may invoke it again explicitly.

---

# 23. No Model Requirement

Kernel 9 must require:

- no Apollo;
- no Mercury reasoning agent;
- no frontier provider;
- no transcription model.

It is deterministic infrastructure.

Pantheon should still be able to inventory Intake even if every reasoning model is offline.

---

# 24. Source Preservation

Kernel 9 shall not:

- rename Intake files;
- move Intake files;
- copy them into raw retention;
- normalize them;
- convert them;
- transcribe them;
- delete them.

Kernel 9 observes.

Kernel 10 processes.

Later retention machinery moves and eventually deletes raw sources according to lawful rules.

---

# 25. Ambiguous or Unsupported Material

Because Kernel 9 accepts ordinary regular files regardless of semantic format, “unsupported” should normally mean filesystem/object safety rather than content format.

If a regular file cannot be read:

- leave it untouched;
- report failure;
- do not create a completed occurrence unless enough identity was safely established;
- allow retry.

If a file's content type is unknown:

- that is acceptable;
- record it as unknown/binary where appropriate;
- Kernel 10 will decide what can be normalized.

---

# 26. Configuration Validation

Extend Kernel 8 configuration validation.

Validate:

- Intake path form;
- Father Time chronicle path resolution;
- explicit timezone;
- retention days greater than zero and within a sane numeric range.

Do not require configured external retention storage to be mounted merely to declare the configuration structurally valid.

Runtime availability and schema validity remain distinct.

If the installation still uses legacy Kernel 8 configuration:

- preserve legacy compatibility;
- apply Kernel 9 defaults internally where appropriate;
- do not silently migrate the file.

Customization of new Kernel 9 fields may require explicit migration to the current configuration schema.

Document this clearly.

---

# 27. Default Paths

Use sane per-user defaults rather than hard-coded `/home/brick` paths.

Resolve home directory at runtime.

Expected conceptual defaults:

```text
paths.intake:
~/Pantheon Intake

paths.father_time_chronicles:
<configured Vault>/00 Pantheon Core/Chronicles/Father Time

retention.raw_input_days:
30

time.timezone:
machine local timezone
```

Do not embed the development user's home directory into portable source.

---

# 28. Concurrency

Prevent two Intake scans from racing against one another.

A second scan should either:

- receive a clear “scan already running” response;
- or serialize behind the first through a simple deterministic lock.

Do not allow concurrent scans to create duplicate occurrences or duplicate chronicle entries.

Avoid building a generalized distributed locking system.

Pantheon is local.

---

# 29. Large Files

Hash files as streams.

Do not load entire recordings or large files into memory merely to compute SHA-256.

The scan should remain capable of inventorying large audio/video files that later Kernels may process overnight.

No artificial file-size rejection is necessary unless required for system safety.

If one is introduced, it must be configurable and documented rather than silently skipping large files.

---

# 30. Privacy and Network Boundary

Intake inspection is local-only.

Kernel 9 shall make no network call because an Intake file was discovered.

Filename, path, hash, type, and chronology remain local.

No Intake metadata is sent to Apollo or Archimedes in this Kernel.

---

# 31. Explicit Non-Goals

Do **not** implement:

- Present reasoning;
- natural-language reference resolution;
- transcription;
- PDF extraction;
- image interpretation;
- Markdown normalization;
- Vault fact creation from Intake content;
- automatic Intake watcher;
- scheduled scans;
- Mercury orders;
- Mercury rounds;
- Future;
- automatic Brief entries;
- notification escalation;
- raw-file movement;
- raw-retention countdown activation;
- Zeus;
- deletion;
- Archimedes;
- Internet research;
- self.change;
- Panopticon.tower.

Do not redesign Kernel 8 configuration.

Do not refactor Weave.

---

# 32. Tests

Add focused automated tests.

At minimum verify:

## Intake traversal

1. default Intake path resolution does not hard-code the development user;
2. recursive regular files are discovered;
3. relative subdirectory paths are preserved;
4. hidden files are skipped;
5. hidden directories are skipped;
6. symlinks are skipped and not followed;
7. special/non-regular objects are skipped where testable.

## Hashing and identity

8. SHA-256 is correct;
9. large-file hashing streams rather than requiring whole-file buffering;
10. identical bytes share content identity;
11. separate occurrences may point to the same content identity;
12. unchanged repeated scan does not create duplicate occurrence;
13. changed content at same path creates a new occurrence;
14. departed then reappearing content can become a new occurrence.

## Stability

15. file changed during scan is deferred rather than finalized.

## Father Time

16. timestamps are RFC3339 with offset;
17. configured timezone is respected;
18. invalid timezone fails validation;
19. machine timezone is used when no override exists;
20. correct yearly chronicle is chosen;
21. first event of a new year creates the new chronicle without altering the prior year;
22. chronicle re-run does not duplicate an existing event.

## Retention

23. new occurrence records 30-day policy;
24. retention state begins as awaiting verified Markdown;
25. retention due date is **not** armed at Intake;
26. no deletion occurs.

## Prometheus

27. paused Pantheon blocks Intake scan;
28. read-only Intake/Father Time inspection remains available;
29. blocked scan does not replay on resume.

## Failure

30. unreadable file remains untouched;
31. failed chronicle write preserves ledger/source state;
32. later repair does not create duplicate occurrence.

---

# 33. Verification

Before reporting completion, run at minimum:

```text
gofmt
go test ./...
go vet ./...
go build -o /tmp/pantheon-kernel9 .
```

Use an explicit output path if the repository's existing `pantheon/` directory still prevents ordinary `go build ./...`.

Also perform live runtime verification.

Create a harmless temporary Intake test set containing:

- ordinary text file;
- copy of identical text under another filename;
- file in a subdirectory;
- hidden file;
- symlink if safe.

Run:

```text
pantheon intake scan
pantheon show intake
pantheon show father-time
```

Run the scan twice and verify the second does not duplicate unchanged occurrences.

Verify the Father Time Markdown chronicle in the actual Vault.

Verify pause:

```text
pantheon prometheus protocol --pause
pantheon intake scan
pantheon prometheus protocol --resume
```

The scan while paused must refuse and must not replay automatically.

Do not use irreplaceable personal input as the initial test corpus.

---

# 34. Live Runtime Refresh

Kernel 9 is not considered fully live merely because the repository builds.

After review-safe implementation:

1. build the runtime binary to the actual installed Pantheon runtime location;
2. restart the Pantheon user service;
3. verify the live commands;
4. confirm bare `pantheon` shows:
   - kill-switch graphic;
   - explanatory safety language;
   - current useful command list.

The reportback shall distinguish repository implementation from installed runtime verification.

---

# 35. Acceptance Criteria

Kernel 9 is a PASS only when:

1. Intake path is configurable with a sane default.
2. Intake scanning is recursive.
3. Only safe regular files become candidate occurrences.
4. Symlinks and hidden/system paths are not ingested.
5. Files receive deterministic SHA-256 content identity.
6. Content identity is distinct from occurrence identity.
7. Repeated scans of an unchanged present file are idempotent.
8. A repeated upload later can create a new occurrence for existing content.
9. Partial/changing files are not finalized.
10. Father Time records offset-aware chronology.
11. Configured timezone works.
12. Machine-local timezone is the default.
13. Yearly human-readable Markdown chronicles are produced.
14. A new year naturally creates a new chronicle.
15. The operational Intake ledger remains local and inspectable.
16. Recent occurrences can be retrieved deterministically for future Present use.
17. Raw retention policy defaults to 30 days.
18. **The retention clock is not armed until durable Markdown is later verified in the Vault.**
19. Kernel 9 performs no deletion.
20. Kernel 9 performs no move/archive of Intake source material.
21. Kernel 9 requires no model.
22. Prometheus pause blocks scanning.
23. Read-only inspection remains available while paused.
24. Failed processing never destroys source material.
25. Chronicle failure is recoverable without duplicate occurrence creation.
26. The Kernel 8 welcome graphic retains a useful command list.
27. The installed runtime reflects the reviewed implementation.
28. Documentation clearly distinguishes Kernel 9 behavior from future Present/Zeus behavior.

---

# 36. Required Kernel 9 Reportback

Produce:

# Kernel 9 Reportback — Father Time and Intake Ledger

The reportback should answer the following rather than blindly follow a rigid universal template.

## A. Actual predecessor

State:

- branch;
- exact predecessor commit;
- Kernel 8 commit/status;
- any small Kernel 8 cleanup performed.

If historical reportback claims were found stale, identify them precisely.

## B. Constitutional status

Expected:

`No constitutional amendment.`

If implementation reveals a constitutional conflict, stop and report rather than resolving it through code.

## C. Configuration

List:

- newly active fields;
- defaults;
- legacy-config behavior;
- timezone behavior;
- validation behavior.

Show a concise resolved example.

## D. Intake ledger

Document actual schema/state representation.

Explain:

- content identity;
- occurrence identity;
- presence/departure;
- idempotency;
- recent-occurrence retrieval.

## E. Father Time

Show:

- timestamp format;
- timezone;
- yearly rollover behavior;
- chronicle path;
- actual human-readable chronicle example.

## F. Retention

State explicitly:

- raw retention default;
- initial retention state;
- whether any due date exists;
- confirmation that deletion is not implemented.

The expected answer is:

> Raw material remains `awaiting_verified_markdown`; the 30-day clock has not started.

## G. Prometheus

Report:

- scan behavior while active;
- scan behavior while paused;
- inspection available while paused;
- no-replay result.

## H. Filesystem safety

Report:

- recursion;
- symlink behavior;
- hidden-file behavior;
- changing-file behavior;
- unreadable-file behavior.

## I. Kernel 8 cleanup

Show the live bare:

```text
pantheon
```

surface and confirm it contains both:

- Prometheus kill-switch presentation;
- useful current command help.

## J. Tests

Provide:

- exact automated commands;
- actual results;
- focused Kernel 9 tests;
- actual live scan demonstration.

Distinguish mocked filesystem tests from real runtime verification.

## K. Files changed

List meaningful files with one-line purpose.

## L. Risks and unresolved work

Include:

- filesystem edge cases;
- external-drive considerations;
- legacy config limitations;
- chronicle reconciliation limitations;
- anything Kernel 10 must know.

## M. Expiry conditions

Identify claims invalidated by future changes to:

- ledger schema;
- chronology format;
- Intake movement;
- retention activation;
- timezone behavior;
- Father Time storage;
- pause enforcement.

## N. Next safe successor

**Kernel 10 — Present and Normalization**

Kernel 10 may assume that Father Time can identify and chronicle Intake occurrences.

Kernel 10 shall be responsible for:

- interpreting supported source type;
- local transcription/extraction;
- producing durable human-readable Markdown;
- verifying that Markdown in the Vault;
- creating Present missives;
- and only then arming raw-retention eligibility.

Kernel 10 must not invent a different Intake identity system.

---

# 37. Stop Conditions for the Implementing Agent

Stop and report rather than improvise if:

- actual Kernel 8 configuration does not support safe extension;
- Intake scanning would require following symlinks;
- the configured Vault cannot safely contain the Father Time chronicle;
- existing Vault policy forbids the intended Chronicle location;
- current machine state cannot distinguish re-scan from re-upload without destructive assumptions;
- implementation would require deleting or moving source material;
- implementation would require a reasoning model;
- a new retention system would conflict with the established future Zeus authority boundary.

Do not solve these by silently weakening the contract.

---

# 38. Successor Boundary

Kernel 9 establishes:

**this object existed, here, at this time.**

Kernel 10 establishes:

**this is what the object says or contains, and here is its durable human-readable representation.**

Only after Kernel 10 verifies that durable representation in the Vault may raw-material retention begin moving toward eventual lawful Zeus disposal.

Father Time remembers the arrival.

Present will describe what arrived.

Apollo will eventually decide what it means.