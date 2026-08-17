# Kernel 18 — Rumor: Whisper Audio Intake

## Objective

Implement **Rumor**, Pantheon’s deterministic audio-transcription daemon.

Rumor watches for eligible audio files, sends them through the configured local Whisper pipeline, produces durable transcript artifacts with provenance, places those artifacts into the Vault, and triggers Pantheon’s existing Intake process so the new material can flow toward Present, Past, and Apollo.

The intended flow is:

```text
audio arrives
    ↓
  Rumor
    ↓
source identity / queue
    ↓
Whisper
    ↓
durable transcript artifact
    ↓
Vault
    ↓
Intake process
    ↓
Present
    ↓
Past
    ↓
Apollo
```

Rumor does not interpret the recording.

Rumor does not summarize it.

Rumor does not decide whether it matters.

Rumor makes speech available to Pantheon as durable, inspectable textual evidence and ensures that the reasoning pipeline knows new material exists.

---

## A. Actual predecessor

Kernel 18 must begin from repository truth.

At implementation start record:

- current branch;
- exact predecessor commit;
- `git status --short`;
- Kernel 17 implementation/reportback state;
- current Whisper configuration;
- current Intake pipeline behavior;
- current Vault transcript/document conventions;
- installed runtime identity.

Do not assume older Whisper behavior remains current merely because previous kernels mentioned it.

Read the current implementation and reportbacks first.

---

## B. Rumor is a daemon

Rumor is deterministic orchestration around Whisper.

Rumor may:

- watch configured audio-arrival locations;
- accept explicit transcription requests;
- identify supported audio files;
- fingerprint/hash source files;
- queue transcription work;
- invoke the configured local Whisper implementation;
- track transcription attempts;
- bound retries;
- persist success/failure state;
- create transcript artifacts;
- create machine-readable metadata;
- place completed transcript artifacts into the Vault;
- trigger the Intake process;
- alert when transcription becomes stuck.

Rumor may not:

- reason about transcript meaning;
- summarize recordings;
- infer tasks or obligations;
- decide who is important;
- grant authority;
- seal orders;
- invoke unrelated tools;
- silently delete source recordings;
- loop indefinitely on broken files.

---

## C. Intake is a process

Kernel 18 must preserve the architectural distinction that **Intake is a process/pipeline, not a personified actor or daemon**.

Rumor feeds completed transcription material into that process.

The Intake process may involve multiple deterministic and reasoning flows depending on the material.

For audio-derived material, the intended path is broadly:

```text
Rumor completes transcript
        ↓
transcript becomes durable Vault material
        ↓
Intake process is triggered
        ↓
Present establishes what arrived / happened
        ↓
Past retrieves relevant continuity
        ↓
Apollo is informed that new material needs reasoning
```

Do not introduce an `Intake daemon` merely to make the diagram symmetrical.

---

## D. Supported source paths

Rumor must support two ingestion paths.

### 1. Watched arrival path

A configured directory or equivalent deterministic source is monitored for eligible audio files.

When a new supported file arrives:

```text
file detected
    ↓
stable-file check
    ↓
source identity/hash
    ↓
queued
    ↓
Whisper
```

Rumor must avoid beginning transcription while a file is still being copied or written.

Use a deterministic stability check appropriate to the runtime.

### 2. Explicit fallback

Provide an operator command resembling:

```text
pantheon rumor transcribe <file>
```

or the repository-consistent equivalent.

This allows Grant to manually feed a file to Rumor when the watcher fails, when the source lives elsewhere, or when explicit control is preferable.

Both routes should converge on the same underlying transcription pipeline.

---

## E. Audio only

Kernel 18 supports audio files.

Do not add video ingestion.

If video material needs transcription later, audio can be extracted upstream and passed to Rumor as ordinary audio.

Avoid adding FFmpeg/video-container complexity to Kernel 18 merely because Whisper could theoretically process extracted media.

Support only audio formats demonstrated by the configured local pipeline.

Document the actual supported set.

---

## F. Whisper is a major local capability

Whisper becomes a standard operating component of Pantheon rather than a side experiment.

Rumor should use the best configured local Whisper model available under Pantheon’s current hardware/configuration policy.

Accuracy is preferred over speed.

The default should not silently fall back to a materially weaker model merely to finish faster.

If fallback behavior exists, it must be explicit and inspectable.

The runtime should expose:

```text
configured Whisper implementation
configured model
model location
language mode
transcription readiness
```

Rumor must not require a cloud transcription provider for normal operation.

---

## G. Language detection

Whisper should normally operate with automatic language detection.

Kernel 18 must preserve multilingual speech rather than assuming English.

The operating environment specifically requires useful handling of:

- English;
- Spanish;
- Vietnamese.

Do not force English unless explicitly configured for a particular request.

The transcript metadata should record detected language information where the Whisper implementation exposes it.

If a recording contains multiple languages, preserve the transcript Whisper actually returns rather than rewriting it into a single language.

Translation is separate from transcription.

---

## H. Accuracy over cosmetic cleanup

Rumor’s responsibility is faithful transcription.

Do not add model-based transcript beautification inside Rumor.

Rumor should not silently:

- paraphrase;
- summarize;
- correct claims;
- translate;
- restructure substantive meaning;
- remove repetitions because they seem unnecessary;
- infer missing speech.

Where deterministic formatting is useful, such as timestamps or segment boundaries already produced by Whisper, preserve it.

Semantic cleanup can occur later through normal Pantheon reasoning if explicitly required.

---

## I. Source identity

Every source audio file should receive durable identity before transcription.

At minimum preserve:

```text
source_id
source_path
source_filename
source_hash
size
detected_at
queued_at
```

Where readily available, also preserve:

```text
duration
audio format
modified timestamp
```

The source hash should be used to prevent accidental duplicate processing.

Renaming or copying the same unchanged source should not automatically produce a second transcript unless explicitly requested.

---

## J. Stable-file detection

A watched file must not be transcribed while still being written.

Rumor should use a bounded deterministic stability rule.

For example:

```text
detected
→ size/mtime observed
→ unchanged across stability interval
→ eligible
```

The exact mechanism should fit the runtime.

Do not rely solely on filename appearance.

If a file never stabilizes, surface it as a stuck source rather than retrying forever.

---

## K. Transcription lifecycle

Rumor should maintain a small durable lifecycle.

Suggested states:

```text
detected
waiting_stable
queued
transcribing
retry_wait
completed
failed
ignored
```

Use fewer or different states if the implementation already has appropriate equivalents.

The important distinctions are:

- waiting for a file to become safe to read;
- queued for transcription;
- actively transcribing;
- bounded retry;
- completed;
- failed and requiring attention.

---

## L. Bounded retries

Whisper failures must not produce an endless retry loop.

Implement bounded retry behavior.

A reasonable pattern is:

```text
attempt 1
    ↓ failure
attempt 2
    ↓ failure
attempt 3
    ↓ failure
FAILED / PAPER JAM
```

The exact retry count should be configurable or explicitly documented.

Retries should use sensible bounded spacing rather than tight-looping.

The attempt history must remain inspectable.

---

## M. Paper-jam behavior

A transcription source that exceeds the retry limit becomes a visible operator problem.

Think of it like a printer paper jam.

Pantheon should surface something like:

> Rumor could not transcribe `<file>` after 3 attempts. The source remains in place. Fix, replace, remove, or retry it.

The alert should identify:

- source file;
- failure state;
- attempt count;
- most useful local error;
- retry command;
- removal/ignore path where appropriate.

Rumor should stop hammering the same source until a deliberate condition permits retry.

---

## N. Alerts

Use Pantheon’s existing safe notification mechanism for persistent Rumor failure where available.

A failed file should be visible through both:

- Rumor inspection;
- an operator notification/alert.

Do not send repeated alerts every watcher cycle.

One failure event should not become notification spam.

A subsequent state change or explicit retry may generate a new meaningful notification.

---

## O. Manual retry

Provide a deterministic operator path resembling:

```text
pantheon rumor retry <source-id>
```

or equivalent.

The operator should not need to rename or duplicate the file simply to make Rumor reconsider it.

Manual retry should preserve historical attempt records.

---

## P. Ignore/remove handling

Rumor should distinguish:

```text
failed
```

from:

```text
intentionally ignored
```

If Grant decides a broken or irrelevant source should not be processed, provide a deterministic way to suppress further attempts without deleting history.

Rumor itself should not delete the source file unless a separate explicit authorized command exists.

Source-file sanitation belongs elsewhere.

---

## Q. Canonical transcript artifact

A successful transcription must produce a durable human-readable Markdown artifact.

The artifact should contain enough provenance to remain meaningful outside Pantheon.

Suggested structure:

```markdown
---
type: transcript
source_id: ...
source_file: ...
source_hash: ...
transcribed_at: ...
whisper_model: ...
detected_language: ...
duration: ...
---

# Transcript — <source filename>

<transcript text>
```

Use actual Vault/frontmatter conventions where they already exist.

Do not invent a conflicting document format if the repository already has one.

---

## R. Machine-readable transcript state

In addition to Markdown, preserve machine-readable metadata/state.

This may live in:

- SQLite;
- JSON sidecar;
- existing Intake/source state;
- another current Pantheon machine-state mechanism.

It should support deterministic answers to:

```text
Has this source been transcribed?

Which transcript belongs to it?

Which model was used?

How many attempts occurred?

What language was detected?

Has the transcript entered Intake?

Did Intake processing succeed?
```

Do not depend on parsing Markdown frontmatter as the only machine-state store if existing Pantheon state is better suited.

---

## S. Transcript location

Completed transcripts should become durable Vault material.

Use a clearly defined transcript location compatible with the existing Vault architecture.

The path should be deterministic and collision-safe.

Filename collisions must not overwrite unrelated transcripts.

Prefer retaining a readable connection to the original source filename while using source identity/hash where needed to guarantee uniqueness.

---

## T. Source audio remains separate

Successful transcription does not authorize deletion of source audio.

Rumor may mark:

```text
transcription complete
```

but the source remains subject to separate retention policy.

Future cleanup may involve:

```text
Father Time
→ retention state
→ lawful sanitation order
→ Zeus
```

Kernel 18 does not implement that cleanup path.

Do not silently delete source audio after successful transcription.

---

## U. Intake handoff

After transcript creation succeeds, Rumor must trigger the existing Intake process.

The preferred pattern is:

```text
transcript artifact created
        ↓
machine state recorded
        ↓
explicit Intake handoff
        ↓
existing processing flow
```

Do not depend on “copy Markdown into a folder and hope another watcher notices it” if Pantheon already exposes a direct deterministic Intake primitive.

If the canonical Intake architecture is folder-based, use it deliberately and document that fact.

The handoff must be observable.

---

## V. What Intake receives

The Intake process should receive enough information to understand that:

- a new transcript artifact exists;
- which source audio produced it;
- where the transcript lives;
- when it was transcribed;
- which Whisper model produced it;
- which language was detected.

Do not reduce the handoff to an anonymous wall of transcript text.

The durable transcript should remain the source artifact.

---

## W. Apollo notification / reasoning trigger

A completed transcript should eventually inform Apollo that new material requires reasoning through the existing Intake → Present → Past → Apollo architecture.

Rumor should not directly invoke Apollo with an ad hoc prompt if the normal Intake flow can carry the event.

The desired effect is:

```text
new transcript
→ normal Pantheon observation/retrieval path
→ Apollo sees new evidence in context
```

not:

```text
Rumor
→ special private Apollo transcription workflow
```

Avoid creating a second reasoning pipeline solely for audio.

---

## X. No automatic importance decision

Rumor does not decide whether:

- the meeting mattered;
- someone made a commitment;
- a task should be created;
- Grant should be notified;
- the transcript should become a project note;
- a calendar event should change.

Those are downstream reasoning/authority concerns.

Rumor’s success ends when the transcript is durable and the Intake process has been correctly triggered.

---

## Y. Explicit transcription command

The manual fallback command should support a simple operator workflow.

For example:

```text
pantheon rumor transcribe ~/Downloads/meeting.m4a
```

Expected behavior:

```text
source registered
→ duplicate check
→ queued
→ transcribed
→ transcript written
→ Intake triggered
```

If the source was already successfully transcribed, the command should report that rather than silently duplicating work.

Provide an explicit force/reprocess option only if demonstrated necessary.

---

## Z. Rumor watcher

Rumor’s watched-directory behavior should be part of the persistent Pantheon runtime.

Do not require a separate manually launched shell watcher after installation.

The watcher should:

- start with the relevant Pantheon daemon/service;
- inspect only configured source locations;
- ignore unsupported files;
- avoid symlink/path escape outside configured boundaries;
- avoid duplicate work;
- survive restart through durable state.

Do not grant it broad filesystem surveillance merely for convenience.

---

## AA. Configuration

Expose clear configuration resembling:

```text
rumor:
  enabled: true
  watch_paths:
    - <configured audio inbox>
  whisper_model: <best configured local model>
  language: auto
  retry_limit: 3
  transcript_destination: <Vault transcript path>
  on_complete: intake
```

Use the repository’s actual configuration format.

Document defaults.

An empty or disabled watcher configuration should fail safely rather than scanning arbitrary directories.

---

## AB. Whisper readiness

Pantheon status/inspection should make Whisper readiness visible.

Rumor should distinguish:

```text
Whisper configured and ready
Whisper configured but unavailable
Whisper model missing
Whisper executable missing
Rumor disabled
```

Do not discover these conditions only after the first audio file fails.

---

## AC. Rumor inspection

Provide operator surfaces resembling:

```text
pantheon show rumor
pantheon show rumor queue
pantheon show rumor source <id>
```

or repository-consistent equivalents.

Grant should be able to answer:

> Is Rumor running?

> What is it transcribing?

> What failed?

> Where did this transcript go?

> Did it enter Intake?

> Which model is being used?

---

## AD. Help integration

Kernel 18 should integrate Rumor into the command-discovery work established by Kernel 17.

`pantheon help --commands` or the final equivalent should list Rumor commands.

Rumor help should show:

- manual transcribe;
- retry;
- inspect queue;
- inspect failed sources;
- supported formats;
- configured watch path;
- transcript location.

Do not make Grant remember new hidden commands immediately after Kernel 17 makes command discovery a usability requirement.

---

## AE. Supported formats

Document and test the actual supported audio formats.

Do not claim “any audio file” unless the configured Whisper pipeline truly supports arbitrary formats.

A practical set may include whichever formats the existing local toolchain demonstrably handles, such as:

```text
wav
mp3
m4a
flac
ogg
```

The implementation should derive from actual runtime support.

Unsupported formats should be ignored or rejected visibly, not handed repeatedly to Whisper.

---

## AF. Multilingual verification

Kernel 18 must include multilingual verification.

At minimum, automated or fixture-based testing should establish that Rumor does not force English and preserves language metadata.

Where suitable local fixture material exists, include:

- English;
- Spanish;
- Vietnamese.

Live verification need not record new private speech solely for testing if safe fixtures can prove the pipeline.

The reportback must state exactly what multilingual behavior was actually verified.

---

## AG. Source and transcript idempotency

Kernel 18 must guarantee that one source file does not repeatedly generate duplicate transcript artifacts.

Use durable source identity/hash.

Repeated watcher scans should converge on:

```text
already completed
```

rather than:

```text
new transcript
new transcript
new transcript
```

If a file changes contents while retaining its filename, treat the new content as a distinct source identity.

---

## AH. Intake idempotency

A successful transcript should enter the Intake process exactly once under normal operation.

Repeated Rumor rounds/restarts must not duplicate the same Intake occurrence.

If the transcript succeeds but Intake handoff fails, Rumor should retain a recoverable state such as:

```text
transcription_completed
intake_pending
```

and retry the **handoff**, not rerun Whisper unnecessarily.

---

## AI. Separate transcription failure from Intake failure

Rumor must distinguish:

```text
Whisper failed
```

from:

```text
transcript succeeded but Intake handoff failed
```

The latter should not waste compute by retranscribing the audio.

Likewise:

```text
transcript written
```

should be distinguishable from:

```text
transcript successfully incorporated into downstream flow
```

This should be inspectable.

---

## AJ. Resource control

Whisper is computationally expensive relative to most Pantheon daemons.

Rumor should avoid uncontrolled concurrent transcription.

Use a bounded worker model.

For the current local system, a single active large-model transcription at a time is acceptable unless repository evidence demonstrates safe benefit from more concurrency.

Additional files may queue.

Do not overload local reasoning availability merely to improve transcription throughput.

---

## AK. Apollo coexistence

Kernel 18 should avoid unnecessarily starving Apollo.

If local Whisper and Apollo compete heavily for memory/CPU, document the behavior and choose a conservative queue policy.

Correct transcription completed later is preferable to unstable simultaneous execution.

Do not silently downgrade Whisper accuracy solely to allow more concurrency.

---

## AL. Prometheus pause

Kernel 18 must define how Rumor behaves during Prometheus pause.

Conservative default:

```text
Prometheus paused
→ do not start new transcription jobs
→ preserve detected/queued files
→ allow safe inspection
→ do not trigger new downstream autonomous reasoning/execution
```

An already-running transcription may either:

- finish safely and remain pending downstream handoff; or
- be interrupted according to existing process-control policy.

Choose one deterministic behavior and document it.

Resume must not duplicate queued files or completed transcripts.

---

## AM. Privacy boundary

Raw audio and transcripts are sensitive local material.

Rumor should remain local-first.

Kernel 18 must not:

- upload recordings to a cloud transcription provider;
- send transcript text to Archimedes/frontier models automatically;
- expose watched directories publicly;
- place recordings in ordinary logs;
- place transcript contents in machine audit logs unnecessarily.

Audit state should record identifiers and operational evidence without duplicating entire recordings/transcripts where not required.

---

## AN. Errors

Rumor errors should be understandable.

Examples:

```text
unsupported audio format
source never stabilized
Whisper executable unavailable
Whisper model unavailable
Whisper process failed
transcript artifact write failed
Vault destination unavailable
Intake handoff failed
duplicate source
```

Each should produce a deterministic state and meaningful operator guidance.

Do not collapse every failure into:

```text
transcription failed
```

---

## AO. Built-in proof path

Kernel 18 must prove the complete audio-ingestion chain with at least one harmless test audio file:

```text
audio file arrives
→ watcher detects it
→ file stabilizes
→ source identity recorded
→ Whisper runs
→ Markdown transcript created
→ metadata persisted
→ transcript written into Vault
→ Intake triggered
→ downstream Intake state confirms receipt
```

This should be performed through the same code path intended for real use.

---

## AP. Manual fallback proof

Separately prove:

```text
pantheon rumor transcribe <audio>
```

using the explicit operator path.

The manual path must use the same source identity/idempotency machinery.

A source already successfully processed by the watcher should not duplicate merely because it is later manually submitted.

---

## AQ. Failure proof

Kernel 18 must prove paper-jam behavior.

Using a safe invalid/broken fixture:

```text
source detected
→ bounded attempts
→ failed
→ alert
→ no further automatic retry storm
```

Then prove manual retry or ignore handling works.

Do not intentionally corrupt live user audio for testing.

---

## AR. Restart proof

Rumor must survive Pantheon restart without losing queue truth.

Test:

```text
source registered / queued
→ service restart
→ source remains known
→ processing resumes or remains safely queued
```

and:

```text
completed source
→ restart
→ watcher sees same file
→ no duplicate transcript
```

---

## AS. Automated verification

Before repository PASS:

```text
gofmt -w $(rg --files -g '*.go')
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel18 .
```

Focused Kernel 18 tests should cover at minimum:

1. watched-directory detection;
2. explicit manual transcription;
3. stable-file detection;
4. supported-format filtering;
5. unsupported-format behavior;
6. source hash/idempotency;
7. changed-content/new-hash behavior;
8. bounded worker concurrency;
9. Whisper invocation;
10. best configured model selection;
11. automatic language mode;
12. multilingual metadata preservation;
13. Markdown transcript creation;
14. machine-readable metadata persistence;
15. collision-safe transcript naming;
16. Vault placement;
17. Intake handoff;
18. Intake idempotency;
19. transcription-success / Intake-failure recovery;
20. bounded retry;
21. paper-jam failure state;
22. alert deduplication;
23. manual retry;
24. ignore handling;
25. source preservation after success;
26. pause behavior;
27. restart recovery;
28. help/command discovery;
29. no cloud transcription path;
30. existing Kernel 10–17 regressions remain green.

---

## AT. Live verification

Before claiming live PASS:

1. build exact Kernel 18 candidate;
2. record SHA-256;
3. install exact candidate;
4. restart only necessary service(s);
5. verify Pantheon health;
6. verify Rumor readiness;
7. verify configured Whisper model;
8. verify watch location;
9. place a harmless supported audio file into the watched location;
10. confirm watcher detection;
11. confirm one Whisper transcription;
12. confirm Markdown transcript artifact;
13. confirm machine metadata;
14. confirm Vault placement;
15. confirm Intake handoff;
16. prove watcher rescanning does not duplicate;
17. exercise manual fallback with a separate harmless file;
18. prove one bounded failure/paper-jam path using a safe fixture;
19. verify alert behavior;
20. verify manual retry/ignore path;
21. verify Prometheus pause behavior;
22. verify command/help surfaces;
23. record exact evidence.

Where appropriate, also verify Spanish and Vietnamese auto-detection with safe local fixtures.

Do not claim multilingual live verification that was not actually performed.

---

## AU. Documentation

Kernel 18 must update active documentation so Whisper/Rumor is treated as a standard Pantheon operating capability.

Document:

- Rumor’s role;
- watched directory;
- manual fallback;
- supported formats;
- configured model;
- language auto-detection;
- transcript destination;
- Intake handoff;
- retry limit;
- paper-jam behavior;
- inspection commands;
- pause behavior;
- source-retention boundary;
- privacy boundary.

Update `pantheon help --commands` or the final Kernel 17 equivalent.

Do not leave Whisper documented as experimental after Kernel 18 PASS.

---

## AV. Existing Whisper debt

Earlier Pantheon reportbacks explicitly retained live Whisper verification debt.

Kernel 18 should reconcile that debt rather than continuing to carry it ambiguously.

The Kernel 18 reportback must state:

- which older Whisper debt is now closed;
- which historical claims remain unchanged;
- whether the current live Whisper path was actually proven end to end;
- any remaining audio/transcription limitations.

Kernel 11 specifically recorded that Kernel 10’s live Whisper verification remained open at that time.

---

## AW. Non-goals

Kernel 18 does not:

- process video;
- summarize transcripts;
- translate transcripts;
- identify semantic importance;
- create tasks from speech;
- send email based directly on speech;
- change calendars based directly on speech;
- delete raw audio;
- implement Zeus;
- implement Nostalgia;
- implement Orion;
- implement Archimedes;
- use cloud transcription by default;
- create a second reasoning pipeline;
- make Intake into a new daemon;
- build a general media server;
- optimize for maximum transcription throughput.

---

## AX. Reportback requirements

The Kernel 18 reportback must state:

- exact predecessor commit;
- starting repository state;
- Kernel 17 state consumed;
- previous Whisper implementation found;
- final Rumor architecture;
- watch mechanism;
- manual fallback command;
- supported formats;
- source identity/hash behavior;
- stable-file behavior;
- configured Whisper model;
- language-detection behavior;
- multilingual verification actually performed;
- retry limit;
- paper-jam behavior;
- alert mechanism;
- transcript Markdown format;
- machine-state representation;
- Vault transcript location;
- Intake handoff mechanism;
- Intake idempotency;
- pause behavior;
- restart behavior;
- source-retention behavior;
- command/help surfaces;
- old Whisper verification debt closed;
- automated test results;
- live end-to-end proof;
- candidate/installed SHA-256;
- files changed;
- known limitations;
- report expiry conditions.

Do not call Rumor PASS merely because Whisper can transcribe a manually supplied file.

The watcher → transcript → Vault → Intake path is the kernel.

---

## AY. Success condition

Kernel 18 PASS means:

> **Speech has become a standard local Pantheon input modality.**

After PASS:

- an eligible audio file may arrive without Grant manually managing the transcription pipeline;
- Rumor notices it;
- the best configured local Whisper model transcribes it;
- multilingual speech is auto-detected rather than assumed English;
- the transcript becomes durable human-readable Vault material;
- machine provenance remains inspectable;
- the Intake process receives the new material exactly once;
- Apollo is eventually informed through the ordinary Pantheon cognition flow;
- broken files become visible bounded paper jams rather than silent retry storms;
- source recordings remain preserved pending separate retention policy;
- manual transcription remains available as a fallback.

The desired experience is:

```text
recording appears
    ↓
Pantheon eventually knows what was said
```

without requiring Grant to babysit Whisper.