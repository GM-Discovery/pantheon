# Kernel 10 — Present and Normalization

**Status:** Proposed implementation order  
**Constitutional authority:** Constitution of Pantheon v0.2  
**Roadmap position:** Kernel 10  
**Predecessor:** Kernel 9 — Father Time and Intake Ledger, live PASS  
**Expected successor:** Kernel 11 — Mercury Orders, Motes, and Seals

---

## 1. Purpose

Kernel 10 gives Pantheon its first deterministic understanding of **what an Intake object contains**.

After Kernel 10, Pantheon shall be able to take an occurrence established by Father Time and produce a durable, source-faithful Markdown representation inside the Vault.

The fundamental chain becomes:

```text
SOURCE
  ↓
Father Time identifies it
  ↓
Present observes its type
  ↓
local processor extracts/transcribes
  ↓
source-faithful Markdown
  ↓
Pantheon reads it back and verifies provenance
  ↓
Present missive
  ↓
raw retention clock may now be armed
```

Present does not decide what the source means.

Present establishes:

> This is the source Pantheon received.
>
> This is the durable representation Pantheon produced from it.
>
> This representation corresponds to this Father Time occurrence.

---

## 2. Kernel 9 Cleanup Sweep

Before substantive Kernel 10 implementation, fix the small inspection inconsistency observed during live Kernel 9 testing.

`pantheon show intake` currently reports valid outer state while these nested fields may display:

```text
latest_scan.intake_path: ""
latest_scan.father_time_year: 0
```

Populate those fields correctly from persisted scan state.

This is presentation/state reconciliation only.

Do not redesign the Kernel 9 ledger.

Also update the final Kernel 9 reportback to record that live verification completed successfully on August 15, 2026, including:

- live daemon deployment;
- first Intake scan;
- identical-content/different-occurrence verification;
- recursive Intake verification;
- second-scan idempotency;
- Prometheus pause refusal;
- no replay after resume;
- zero failed scans;
- zero pending chronology writes.

Kernel 9 is a PASS.

---

## 3. Durable Raw Vault

Activate a configurable raw-material Markdown root.

Default:

```text
<configured Vault>/80 Pantheon Raw
```

Conceptual configuration:

```text
paths.raw
```

or a clearly equivalent field consistent with Kernel 8 configuration conventions.

Do not hard-code `/home/brick`.

---

## 4. Raw Organization

Use deterministic source-type organization beneath `80 Pantheon Raw`.

Initial structure:

```text
80 Pantheon Raw/
├── Text/
│   └── YYYY/
│       └── MM/
├── Documents/
│   └── YYYY/
│       └── MM/
└── Transcripts/
    └── YYYY/
        └── MM/
```

Classification is mechanical:

### Text

```text
.md
.txt
```

### Documents

```text
.pdf
.docx
```

### Transcripts

Audio:

```text
.wav
.mp3
.m4a
.flac
.ogg
```

Video whose audio is being transcribed:

```text
.mp4
.mov
```

Preserve useful Intake relative-path organization beneath that hierarchy when practical.

Example:

```text
Pantheon Intake/
└── MFPI/
    └── Meetings/
        └── Monday.m4a
```

may become:

```text
80 Pantheon Raw/
└── Transcripts/
    └── 2026/
        └── 08/
            └── MFPI/
                └── Meetings/
                    └── Monday.md
```

This is deterministic organization, not semantic filing.

Apollo may establish conceptual relationships later.

---

## 5. Filename Policy

Preserve the human-readable source filename wherever possible.

Examples:

```text
notes.txt
→ notes.md
```

```text
meeting.m4a
→ meeting.md
```

```text
project-proposal.pdf
→ project-proposal.md
```

Do not expose opaque occurrence IDs in ordinary filenames unless required to prevent ambiguity.

If the destination filename already belongs to another occurrence, use a readable deterministic suffix such as:

```text
meeting--2026-08-15-1101.md
```

If further disambiguation is required, append a short occurrence fragment.

Never overwrite another occurrence's raw artifact merely because filenames collide.

---

## 6. Source Faithfulness

Normalization means:

> preserve the information and recoverable structure of the source in human-readable Markdown.

It does **not** mean:

- summarize;
- improve prose;
- rewrite grammar;
- remove repetition;
- infer missing content;
- identify action items;
- decide importance;
- create facts;
- interpret intent;
- add advice;
- fill gaps;
- invent speaker identities.

Present is not Apollo.

---

## 7. Markdown Source Metadata

Every normalized artifact shall contain inspectable provenance.

Use ordinary YAML frontmatter unless the repository has an established safer equivalent.

Example:

```markdown
---
pantheon:
  kind: raw-source
  occurrence_id: intake-40a9a0ddc586dabcc3daf0df6baa7950
  content_id: sha256:fa7195...
  source_path: "Kernel 9 Test/original.md"
  original_filename: "original.md"
  source_type: "text/markdown"
  observed_at: "2026-08-15T11:01:22-07:00"
  normalized_at: "2026-08-15T11:40:00-07:00"
  processor: "pantheon-text"
  processor_version: "..."
  body_sha256: "sha256:..."
---

# Kernel 9 Intake Test

...
```

Do not attempt to place a hash of the entire file inside the file itself.

Instead:

- store a hash of the normalized **body** in metadata;
- store the complete generated artifact hash in machine state if useful.

This avoids self-referential hashing.

---

## 8. Markdown Verification

Verification is mechanical.

It means Pantheon has established that:

1. the Markdown artifact was successfully written;
2. the artifact can be opened and read back;
3. required Pantheon provenance metadata is present;
4. occurrence ID matches the intended Kernel 9 occurrence;
5. source content ID matches the intended Kernel 9 content;
6. normalized body hash matches the generated body;
7. processor completed without a fatal error;
8. the artifact resides beneath the configured Vault/raw boundary.

It does **not** mean:

> Every word extracted or transcribed is guaranteed correct.

Extraction/transcription warnings survive verification.

---

## 9. Processing State

Extend Intake occurrence processing state.

Useful conceptual states:

```text
unprocessed
processing
verified
blocked_unsupported
blocked_missing_processor
failed
```

Equivalent concise names are acceptable.

Do not mark an occurrence `verified` merely because processing began.

If Pantheon crashes during processing, later inspection must not pretend processing succeeded.

---

## 10. Retention Transition

Kernel 9 established new occurrences as:

```text
retention_state: awaiting_verified_markdown
retention_due_at: null
```

Only successful Kernel 10 Markdown verification may transition this state.

After verification:

```text
retention_state: retention_armed
verified_at: <Father Time timestamp>
retention_due_at: verified_at + retention.raw_input_days
```

With the current default:

```text
verified_at + 30 days
```

The source file remains exactly where it is.

Kernel 10 shall not:

- delete it;
- move it;
- archive it;
- rename it.

The due date records future eligibility.

It grants nobody permission to delete.

Eventually:

```text
Father Time
    ↓
Future
    ↓
Mercury
    ↓
lawful explicit purge order
    ↓
Zeus
```

will perform that work.

---

## 11. Text Processor

For:

```text
.md
.txt
```

processing should remain simple.

### Markdown

Preserve the body faithfully.

Add Pantheon provenance frontmatter without rewriting existing content.

If the source already contains YAML frontmatter, do not destroy or ambiguously merge the user's metadata.

Use a safe strategy such as retaining source frontmatter as part of the source body while keeping Pantheon provenance structurally distinct.

### Plain text

Preserve text faithfully as Markdown-compatible body content.

Do not infer headings merely because a line looks important.

---

## 12. PDF Processor

Initial PDF support is for PDFs containing extractable text.

Normalize recoverable:

- paragraphs;
- headings where reliably exposed;
- lists;
- page boundaries where useful;
- tables where reasonably recoverable without fabrication.

Preserve page provenance where practical, for example:

```markdown
<!-- source-page: 4 -->
```

or another unobtrusive deterministic marker.

This will later make citations back to source regions much easier.

### Scanned/image-only PDFs

Do not silently OCR them in Kernel 10.

If usable text cannot be extracted:

```text
blocked_unsupported
```

or a more precise:

```text
blocked_needs_ocr
```

may be used.

The original source remains untouched and retention remains unarmed.

---

## 13. DOCX Processor

Extract source-faithful human-readable structure where recoverable:

- headings;
- paragraphs;
- ordered lists;
- unordered lists;
- simple tables;
- hyperlinks.

Do not attempt pixel-perfect Word reproduction.

Do not interpret comments, tracked changes, or complex layout unless safely supported.

If material cannot be represented faithfully enough, include explicit extraction warnings rather than silently discarding important structure.

---

## 14. Local Whisper Processor

Audio transcription shall be **local only**.

No cloud fallback exists.

No frontier fallback exists.

No upload to a transcription API exists.

Pantheon shall provide configuration for a local Whisper processor rather than hard-code the development machine.

Conceptual configuration may include:

```text
ingestion.whisper.enabled
ingestion.whisper.binary
ingestion.whisper.model
ingestion.whisper.language
```

Exact naming may follow the existing configuration shape.

`language` should permit automatic language detection.

This is important because the local transcription path must not assume English-only material.

---

## 15. Whisper Ownership

Pantheon owns the transcription workflow.

The person should not eventually need to manually run Whisper and then feed its output back into Pantheon.

The intended user experience is:

```text
audio enters Intake
        ↓
Father Time identifies it
        ↓
Present invokes configured local Whisper
        ↓
transcript returned locally
        ↓
Markdown created
        ↓
Markdown verified
```

Kernel 10 shall implement that callable pipeline now.

It does **not** need to continuously watch Intake yet.

Automatic recurring detection belongs to later proactive rounds.

For now, explicit Present processing invokes the pipeline.

---

## 16. Whisper Installation Boundary

Kernel 10 shall detect and use a configured/available local Whisper installation.

It shall **not silently install system software**.

If Whisper is unavailable:

```text
blocked_missing_processor
```

and report what requirement is missing.

Example:

```text
Audio source identified.

Local Whisper processor is not configured.
Source remains untouched.
Retention has not been armed.
```

Full guided installation, dependency discovery, and environment setup remains Kernel 22.

However, Kernel 10 must expose enough diagnostics that we can configure the existing local Whisper installation and prove the integration works live.

---

## 17. Audio Conversion

Some supported audio/video containers may require a local conversion step before Whisper can process them.

Pantheon may use a configured/local deterministic media tool such as FFmpeg when required.

No network conversion service is permitted.

Temporary conversion artifacts:

- must live outside the Vault raw-source tree;
- must never replace the original;
- must not be confused with durable source material;
- must be clearly process-owned.

Orphaned transformation temp files may later be eligible for Zeus sanitation.

Do not let a conversion failure arm retention.

---

## 18. Transcript Format

A normalized transcript should remain readable without Pantheon.

Example:

```markdown
---
pantheon:
  kind: raw-source
  source_type: audio/m4a
  ...
---

# Meeting Recording

**Source:** `MFPI/meeting.m4a`

## Transcript

[00:00:00]

Good morning. I wanted to start by...

[00:02:31]

The next issue is...
```

Timestamps should be retained at useful intervals if the processor provides reliable timing.

Do not generate a timestamp for every word.

Do not invent speaker identity.

---

## 19. Speaker Handling

If the installed transcription stack genuinely provides reliable diarization later, Pantheon may preserve those processor outputs.

Kernel 10 shall not fabricate:

```text
Speaker 1
Speaker 2
```

merely because multiple voices seem likely.

A transcript with no speaker labels is preferable to false attribution.

---

## 20. Video

For supported video containers, Kernel 10 handles **audio transcription only**.

Example:

```text
meeting.mp4
→ extract local audio
→ Whisper
→ meeting.md transcript
```

Kernel 10 does not inspect visual frames.

It does not claim to represent visible information from the video.

The Markdown should state that only the audio channel was processed.

---

## 21. Images

Standalone images are recognized but unsupported by Kernel 10.

Examples:

```text
.png
.jpg
.jpeg
.webp
```

They remain in Intake with a clear blocked/unsupported state.

Retention remains unarmed.

No cloud vision fallback.

No OCR fallback.

---

## 22. Present Missive

After successful normalization and verification, Present creates a factual local missive.

The missive should contain at least:

```text
missive_id
created_at
original user request, if any
occurrence_id
content_id
source path
source type
normalized Markdown path
processor
processor version if available
warnings
verification state
verified_at
retention state
retention_due_at
```

The missive is operational state for later Mercury/Apollo handling.

It may reside in SQLite.

The durable source representation remains the Vault Markdown.

---

## 23. Present's Factual Result

Present may produce statements such as:

> Audio transcription completed.
>
> Source duration: 47 minutes 12 seconds.
>
> Markdown transcript written and verified.
>
> Processor reported three low-confidence spans.

Present shall not produce:

> The meeting was mainly about funding problems.

That is interpretation.

Present shall not produce:

> Your action items are...

That requires reasoning.

---

## 24. Preserve the Person's Request

This command must be useful:

```text
pantheon present "I need a brief summary and all action items from the meeting recording I just uploaded."
```

Present should **not throw away the requested work merely because Apollo does not exist yet**.

It shall:

1. record the person's request verbatim;
2. scan/reconcile Intake if necessary;
3. deterministically identify the most plausible referenced occurrence where possible;
4. normalize and verify that occurrence;
5. create the Present missive;
6. preserve the original requested work for later reasoning.

Current response might say:

```text
Present prepared the source.

Source:
  meeting.m4a

Normalized:
  80 Pantheon Raw/Transcripts/2026/08/meeting.md

Verification:
  complete

Requested reasoning:
  "I need a brief summary and all action items..."

Reasoning is awaiting Apollo.
```

No bluffing.

---

## 25. Deterministic Recent-Source Resolution

Kernel 9 already provides chronological metadata filters.

Kernel 10 may deterministically resolve narrow references such as:

```text
the file I just uploaded
the latest file
the recording I just uploaded
latest recording
that PDF
latest PDF
latest document
```

Resolution may use:

- chronology;
- extension;
- MIME/type;
- current presence state.

It must not use semantic content reasoning.

---

## 26. Ambiguous Reference

If two occurrences are genuinely equally plausible, Present shall not silently guess.

Return a compact ambiguity result showing the small number of candidates.

Example:

```text
I found two recent recordings:

1. staff-meeting.m4a — 11:14
2. client-note.m4a — 11:16

Use the filename or occurrence ID.
```

The CLI may support an exact occurrence selector.

Do not invoke Apollo merely to resolve deterministic file identity.

---

## 27. Exact Occurrence Processing

Provide an explicit interface for scripts and debugging.

Conceptually:

```text
pantheon present --occurrence intake-...
```

This bypasses natural reference resolution and processes the specified occurrence.

This will also be useful later when Mercury carries a sealed order containing an exact occurrence ID.

---

## 28. Process-New Primitive

Implement a reusable deterministic primitive for processing currently unprocessed supported Intake occurrences.

Conceptually:

```text
pantheon present --new
```

It may first reconcile Intake through the Kernel 9 scanner.

It then processes eligible unprocessed occurrences.

This is **not** a background watcher.

It is an explicit primitive that later Mercury rounds can invoke.

That gives us the eventual behavior:

> audio arrives → Pantheon notices → Whisper

without prematurely building Kernel 15 scheduling.

---

## 29. Idempotent Processing

Running Present twice against the same already-verified occurrence must not create duplicate raw Markdown artifacts.

Expected behavior:

```text
already verified
```

or equivalent.

Present should verify the existing artifact still satisfies its recorded provenance before treating it as complete.

Do not duplicate:

```text
meeting.md
meeting--2.md
meeting--3.md
```

merely because the same occurrence was processed repeatedly.

---

## 30. Failed Processing

A failed extraction/transcription shall:

- leave original Intake material untouched;
- leave retention unarmed;
- record failure status;
- preserve useful processor error information;
- permit explicit retry.

Do not create a misleading verified Markdown artifact containing only an error message.

---

## 31. Partial Processing

If processor output is incomplete but recoverable, distinguish:

```text
warning
```

from:

```text
fatal failure
```

A verified artifact may contain explicit warnings if source representation is still mechanically complete according to the processor.

Examples:

```text
Warning: text extraction reports malformed embedded font.
```

or:

```text
Warning: transcription processor reported uncertain spans.
```

Do not hide processor uncertainty.

---

## 32. Unsupported Processing

Unsupported inputs shall remain ordinary Intake occurrences.

Example inspection:

```text
photo.png
processing_state: blocked_unsupported
retention_state: awaiting_verified_markdown
```

A blocked item is not a failed system.

It means Pantheon currently lacks the authorized processor.

---

## 33. Prometheus Boundary

Present processing is consequential local work.

It respects the common pause gate.

While paused:

```text
pantheon present ...
pantheon present --new
pantheon present --occurrence ...
```

must refuse.

Read-only inspection remains available.

A blocked Present operation shall not automatically replay after resume.

---

## 34. Local-Only Boundary

Kernel 10 processing makes **no Internet call**.

This includes:

- PDFs;
- DOCX;
- audio;
- video;
- unsupported images.

A missing processor remains missing.

Pantheon does not solve the problem by sending the person's material somewhere else.

---

## 35. Present Inspection

Add a small read-only inspection surface.

Conceptually:

```text
pantheon show present
```

It should show:

- most recent Present missives;
- processing successes;
- blocked inputs;
- failures;
- pending/unprocessed occurrences;
- normalized Markdown paths;
- retention state.

Do not build Panopticon.tower.

---

## 36. Processor Inspection

Provide enough processor visibility for setup and debugging.

Conceptually:

```text
pantheon show processors
```

or fold the same information sensibly into `show present`.

At minimum expose:

```text
text: ready
pdf: ready / missing dependency
docx: ready
whisper: ready / missing / misconfigured
media conversion: ready / missing
```

For Whisper, include resolved local configuration without exposing secrets.

There should be no secrets involved in ordinary local Whisper configuration.

---

## 37. Audit

Record meaningful processing events.

At minimum:

```text
processing started
processing completed
processing blocked
processing failed
Markdown verified
retention armed
```

Audit entries should reference:

- occurrence ID;
- missive ID where applicable;
- artifact path;
- processor.

Do not duplicate entire source/transcript text into machine audit logs.

---

## 38. Father Time

All relevant transition times come from Father Time.

At minimum:

```text
processing_started_at
normalized_at
verified_at
retention_due_at
missive_created_at
```

Do not scatter ad hoc local timestamp generation across individual processors.

The same configured timezone policy established in Kernel 9 continues.

---

## 39. Evidence Chain

Kernel 10 should make this chain inspectable:

```text
Intake occurrence
    ↓ occurrence_id
source content
    ↓ content_id / SHA-256
normalized Markdown
    ↓ provenance metadata
verified raw artifact
    ↓ artifact/body hash
Present missive
```

This becomes the foundation for later:

```text
Past citation
Weave relation
Apollo conclusion
```

---

## 40. Source Mutation

If the original source changes before processing begins, Kernel 9 must establish the new occurrence/content identity first.

Present shall not process bytes and then attach them to a stale occurrence ID.

Before processing, verify that source identity still matches the ledger occurrence.

If not:

- stop;
- reconcile Intake;
- use the correct new occurrence.

---

## 41. Artifact Mutation

Kernel 10 must never silently overwrite a normalized artifact that no longer matches the artifact Pantheon originally wrote.

If an existing artifact appears to have been human-modified:

- preserve it;
- report the condition;
- do not silently restore generated text over the person's changes.

The exact human-correction model can mature later.

Human edits are not an excuse to destroy material.

---

## 42. Markdown Already in Intake

The Kernel 9 test material gives us an immediate deterministic live test.

For:

```text
Kernel 9 Test/original.md
```

Kernel 10 should create an artifact under:

```text
80 Pantheon Raw/Text/2026/08/Kernel 9 Test/original.md
```

with Pantheon provenance plus the exact original body.

After read-back verification:

```text
processing_state: verified
retention_state: retention_armed
retention_due_at: <verified_at + 30 days>
```

The original remains in:

```text
~/Pantheon Intake/Kernel 9 Test/original.md
```

---

## 43. No Automatic Bulk Processing at Daemon Startup

Do not process every historic Intake file merely because the Kernel 10 daemon starts.

Processing is triggered explicitly through Present in this Kernel.

Later Mercury rounds may make that proactive according to policy.

This avoids an upgrade unexpectedly launching transcription jobs.

---

## 44. Resource Discipline

Audio/video transcription can be expensive in CPU, RAM, and time.

Kernel 10 needs basic cancellation/error discipline but does not need the full Prometheus resource-control system of Kernel 16.

Do not run multiple heavyweight Whisper transcriptions concurrently by default.

A single local processing lane is acceptable.

Text/document work may remain lightweight.

---

## 45. Configuration

Activate or add the minimum configuration required.

Conceptually:

```json
{
  "paths": {
    "raw": "<vault>/80 Pantheon Raw"
  },
  "ingestion": {
    "whisper": {
      "enabled": true,
      "binary": "",
      "model": "",
      "language": "auto"
    },
    "media": {
      "converter": ""
    }
  }
}
```

Do not require exact schema if a more coherent representation fits the existing Kernel 8 code.

Rules:

- sane defaults where possible;
- no development-user hardcoding;
- missing optional processors do not invalidate Pantheon globally;
- explicitly configured nonexistent binaries should produce clear validation/readiness errors;
- legacy configuration remains readable;
- explicit migration remains required to persist new fields.

---

## 46. Bare Pantheon Welcome

Update the concise useful command list beneath the kill-switch graphic to include Present without turning it into a wall of help.

At minimum, a first-time person should now discover:

```text
pantheon present "request"
pantheon intake scan
pantheon show intake
pantheon show present
pantheon prometheus protocol --pause
```

Keep the existing safety presentation.

---

## 47. Explicit Non-Goals

Do **not** implement:

- semantic summaries;
- action-item extraction;
- facts;
- relational interpretation;
- Weave updates from source meaning;
- Apollo reasoning;
- Internet research;
- frontier processing;
- cloud transcription;
- OCR;
- image vision;
- visual video understanding;
- automatic background Intake watching;
- recurring rounds;
- notifications;
- Mercury orders;
- Future;
- Zeus deletion;
- automatic source movement;
- directory auto-classification by semantic topic;
- self.change.

Kernel 10 represents.

Kernel 12 reasons.

---

## 48. Tests — Text

Verify:

1. Markdown body is preserved exactly.
2. Plain text content is preserved.
3. provenance is added.
4. existing user YAML frontmatter is not destroyed.
5. artifact can be read back.
6. occurrence/content identity matches.
7. body hash verifies.
8. same occurrence reprocessing is idempotent.
9. filename collision does not overwrite another occurrence.
10. relative Intake organization is preserved.

---

## 49. Tests — Retention

Verify:

11. unprocessed source begins `awaiting_verified_markdown`.
12. processing start does not arm retention.
13. extraction success without verification does not arm retention.
14. verified Markdown arms retention.
15. due date equals verification time plus configured retention days.
16. source remains in Intake.
17. no deletion occurs.
18. failed processing leaves retention unarmed.
19. unsupported processing leaves retention unarmed.

---

## 50. Tests — Documents

Verify:

20. text PDF produces source-faithful Markdown.
21. PDF page provenance is retained where supported.
22. image-only PDF does not pretend extraction succeeded.
23. DOCX headings/paragraphs/lists are preserved where recoverable.
24. basic DOCX hyperlinks/tables survive where implemented.
25. extraction warning remains visible.

---

## 51. Tests — Whisper

Use a fake/local processor harness for deterministic automated tests.

Verify:

26. audio dispatches to the configured Whisper processor.
27. processor output becomes Markdown transcript.
28. timestamps survive if supplied.
29. missing Whisper becomes `blocked_missing_processor`.
30. no cloud fallback occurs.
31. transcription failure leaves source intact.
32. video passes only its audio path to transcription.
33. visual video content is never claimed to have been processed.
34. speaker names are not invented.

---

## 52. Tests — Present

Verify:

35. exact occurrence selection works.
36. “latest file” resolves chronologically.
37. “latest recording” restricts to audio/video recording types.
38. “latest PDF” restricts appropriately.
39. genuine ambiguity produces candidate choices rather than guess.
40. original person's request is retained verbatim.
41. reasoning request is marked pending rather than answered.
42. `--new` processes eligible unprocessed sources.
43. `--new` does not duplicate previously verified artifacts.

---

## 53. Tests — Prometheus

Verify:

44. pause blocks Present processing.
45. pause blocks `--new`.
46. pause blocks exact occurrence processing.
47. read-only Present/processor inspection remains available.
48. blocked work does not replay after resume.

---

## 54. Tests — Failure and Safety

Verify:

49. changed source is not attached to stale occurrence identity.
50. unreadable source remains untouched.
51. failed artifact write leaves retention unarmed.
52. failed read-back verification leaves retention unarmed.
53. human-modified existing artifact is not silently overwritten.
54. raw path cannot escape configured Vault through traversal.
55. raw path symbolic-link escape is rejected.
56. processor cannot cause output to be written outside the allowed raw root merely by controlling a filename.

---

## 55. Build Verification

Before reportback:

```text
gofmt
go test ./...
go vet ./...
go build -o /tmp/pantheon-kernel10 .
```

Continue using explicit build output if the repository's `pantheon/` directory still causes the known Go output-name collision.

---

## 56. Live Verification — Markdown First

Deploy the reviewed binary and restart the user service.

Use our existing harmless Kernel 9 occurrence.

Process:

```text
Kernel 9 Test/original.md
```

through Present.

Verify:

1. artifact appears beneath `80 Pantheon Raw/Text/...`;
2. source body matches exactly;
3. provenance identifies the correct occurrence;
4. artifact reads back successfully;
5. Intake occurrence becomes `verified`;
6. retention becomes armed;
7. due date is 30 days after verification;
8. original Intake file still exists.

Then process it again and prove no duplicate artifact appears.

---

## 57. Live Verification — Whisper

After deterministic text processing passes, inspect:

```text
pantheon show processors
```

Configure the person's existing local Whisper installation if necessary.

Do not install a cloud service.

Then use a harmless short local audio recording.

Suggested spoken test:

> Pantheon transcription test. Father Time saw this recording. Present should preserve these words without interpreting them.

Run it through Present.

Verify:

- Whisper invocation is local;
- transcript Markdown exists;
- recognizable spoken content is present;
- source path and occurrence match;
- transcript is verified;
- retention becomes armed;
- original recording remains untouched.

If Whisper is not yet correctly installed/configured, Kernel 10 may remain repository-PASS/live-text-PASS with Whisper integration marked **blocked pending local processor configuration**, but do not call the full audio path verified until this smoke test succeeds.

---

## 58. Acceptance Criteria

Kernel 10 is a PASS only when:

1. supported source type is detected deterministically;
2. Markdown is normalized faithfully;
3. durable output goes beneath `80 Pantheon Raw`;
4. organization is deterministic;
5. human filenames are preserved where possible;
6. provenance connects artifact to occurrence/content identity;
7. artifact is mechanically read-back verified;
8. successful verification arms retention;
9. retention begins at verification, not Intake;
10. source remains untouched;
11. failed/blocked sources remain protected;
12. `.md` works;
13. `.txt` works;
14. text PDF works;
15. DOCX works;
16. configured local audio transcription path exists;
17. no external transcription fallback exists;
18. video processing is audio-only;
19. images remain unsupported;
20. Present creates factual missives;
21. original requested work is preserved;
22. Present does not answer reasoning requests;
23. recent deterministic reference resolution works;
24. exact occurrence selection works;
25. repeated processing is idempotent;
26. Prometheus pause blocks processing;
27. no blocked work replays;
28. processor readiness is inspectable;
29. Kernel 9 inspection fields are corrected;
30. installed runtime reflects reviewed source.

---

## 59. Required Kernel 10 Reportback

Produce:

# Kernel 10 Reportback — Present and Normalization

### A. Actual predecessor

- branch;
- exact predecessor;
- Kernel 9 final status;
- live verification status;
- Kernel 9 cleanup performed.

### B. Constitutional status

Expected:

```text
No constitutional amendment.
```

### C. Raw Vault

Show:

- configured root;
- actual resolved root;
- organization hierarchy;
- filename collision strategy.

### D. Processors

For each:

```text
Markdown
text
PDF
DOCX
Whisper
media conversion
```

report:

- implementation;
- dependency;
- current readiness;
- known limitations.

### E. Provenance

Show one real generated frontmatter example and explain:

```text
occurrence → content → Markdown → verification
```

### F. Verification

Describe the exact mechanical checks required before `verified`.

### G. Retention

Show an actual occurrence transition:

```text
awaiting_verified_markdown
→ retention_armed
```

with:

```text
verified_at
retention_due_at
```

Explicitly confirm the source still exists.

### H. Present

Show:

- direct occurrence processing;
- recent-source resolution;
- preserved user request;
- factual missive;
- deferred reasoning.

### I. Local-only boundary

Confirm:

- no content network transmission;
- no cloud transcription;
- no frontier processing.

### J. Whisper

Report actual resolved Whisper configuration.

Distinguish:

```text
implementation exists
```

from:

```text
live local Whisper path verified
```

Do not claim the latter without running it.

### K. Prometheus

Show pause refusal and no replay.

### L. Tests

List automated and live results separately.

### M. Files changed

One-line purpose per meaningful file.

### N. Risks

Include:

- extraction fidelity;
- PDF limitations;
- DOCX limitations;
- Whisper configuration;
- large media resource load;
- human modification of Raw artifacts;
- future OCR/vision needs.

### O. Expiry conditions

Identify changes that invalidate reportback claims.

### P. Next safe successor

**Kernel 11 — Mercury Orders, Motes, and Seals.**

---

## 60. Stop Conditions

Codex shall stop and report rather than improvise if:

- implementing a source type requires sending source content externally;
- safe Vault path enforcement cannot be guaranteed;
- verification cannot reliably bind output to the Kernel 9 occurrence;
- processing requires deleting/moving the source;
- the existing Whisper setup cannot be invoked without unclear environmental changes;
- retention would need to arm before verification;
- a processor would require Present to make semantic judgments;
- PDF support would require unapproved OCR;
- image/video visual interpretation becomes necessary;
- Kernel 9 identity must be replaced rather than consumed.

---

## 61. Successor Boundary

Kernel 9 established:

> **This object existed here at this time.**

Kernel 10 establishes:

> **This is a faithful durable representation of that object.**

Kernel 11 will establish:

> **This is an order, this is who carries it, and this is the authority under which it may proceed.**

Kernel 12 will finally allow Apollo to say:

> **This is what the source means in the context of everything else Pantheon knows.**

That separation is important. Present should be trustworthy precisely because it is **boring**. It records what came in, preserves it, verifies the chain, and hands reasoning forward.
