# Kernel 18 Reportback — Rumor / Whisper Audio Intake

**Status: PASS — repository and live runtime verified August 17, 2026.**

Kernel 18 implements Rumor as a deterministic local audio-to-evidence
boundary. Rumor watches explicitly configured paths or accepts an explicit
local audio path, waits for stable content, records source identity, invokes
the configured local Whisper processor, writes durable transcript artifacts,
and hands the transcript to the existing Intake process.

Rumor does not summarize, interpret, prioritize, decide importance, create
authority, invoke Apollo, call the cloud, or delete raw audio. Its output is
evidence for the existing Intake → Present → Past / Weave → Apollo path.

## Actual predecessor

- Branch: `trunk`.
- Exact predecessor: `53d22eb6034fc0f7260ecd58637f7a6b8d657351`, Kernel 17 whole-animal integration.
- The user’s untracked Aphrodite addition, Kernel 18 specification, and later
  Kernel 19 specification were preserved. Kernel 19 was not opened, changed,
  staged, or implemented.

## Implementation

### Configuration

Canonical configuration now supports:

```json
"rumor": {
  "enabled": true,
  "watch_paths": ["/home/example/Pantheon Audio"],
  "whisper_model": "/home/example/whisper.cpp/models/ggml-large-v3.bin",
  "language": "auto",
  "retry_limit": 3,
  "transcript_destination": "/home/example/Vault/80 Pantheon Raw/Audio Transcripts",
  "on_complete": "intake",
  "stability_seconds": 2,
  "poll_seconds": 5
}
```

An empty or disabled watcher is inert. Binary resolution uses the explicit
`ingestion.whisper.binary`, then the declared local `transcribe` capability,
then a local `whisper-cli`/`whisper` lookup. Model resolution prefers
`rumor.whisper_model`, then `ingestion.whisper.model`, then the declared local
capability model. No network transcription path exists.

The installed `whisper.cpp` binary supports FLAC, MP3, OGG, and WAV. Rumor
uses its local `-m`, `-f`, `--output-txt`, `--output-json-full`, `--no-prints`,
`--no-timestamps`, and `-l` contract. `auto` is passed as auto-detection; the
runtime records a detected JSON language when Whisper supplies one and does
not infer a language when it does not.

### Durable lifecycle and safety

SQLite tables `rumor_sources` and `rumor_attempts` persist:

- source path, filename, size, modification time, and SHA-256 identity;
- `waiting_stable`, `queued`, `transcribing`, `retry_wait`, `completed`,
  `failed`, `ignored`, and `duplicate` lifecycle states;
- bounded attempts, next retry time, failure kind, and last error;
- Whisper binary/model/language and transcript/metadata paths;
- Intake occurrence and handoff state;
- duplicate source identity and one-shot paper-jam alert state.

The worker is single-active and protected against concurrent duplicate manual
dispatch. Renamed/copied identical bytes become a duplicate source record and
do not create a second transcript. Changed bytes at the same path receive a
new source identity and transcript suffix. Filename/path construction is
sanitized and hash-qualified; transcript destinations must remain beneath the
configured Vault. Hidden entries, symlinks, and symlink path components are
not followed.

Raw audio remains in place. A successful transcript writes Markdown and JSON
metadata under Vault, then writes a deterministic handoff copy under
`Intake/Rumor/` and invokes the existing Intake scan. A transcription failure
and an Intake handoff failure are separate: handoff failure leaves the
completed transcript durable with `intake_state: pending` and retries the
handoff without retranscribing.

Paper-jammed sources stop after the configured retry limit. Alerts use the
existing local notification command and are deduplicated. Manual `retry`
resets only the current attempt cycle; `ignore` preserves the source while
preventing further automatic work.

The watcher starts with the persistent daemon, scans only configured paths,
and rehydrates durable queue state after restart. Prometheus pause prevents
new scans and transcription. An already-running local decode may finish, but
a paused completion does not hand off to Intake. Resume does not replay a
completed or ignored source.

Inspection and control surfaces are:

```text
pantheon show rumor
pantheon show rumor queue
pantheon show rumor source <source-id>
pantheon rumor scan
pantheon rumor transcribe <audio-path>
pantheon rumor retry <source-id>
pantheon rumor ignore <source-id>
```

Mutation commands require the local control token. Help and command discovery
advertise the same surfaces.

## Focused automated coverage

New Kernel 18 tests explicitly cover:

- watched arrival, recursive stable-file gating, hidden-file exclusion, and
  symlink exclusion;
- explicit manual transcription and local Whisper argument construction;
- FLAC/MP3/OGG/WAV acceptance and unsupported-format refusal;
- source SHA-256 identity, renamed/copied identical bytes, changed content,
  hash-qualified output names, and idempotency;
- concurrent manual requests producing one source and one Whisper invocation;
- local model resolution, missing-model refusal, mocked local Whisper
  dispatch, diagnostic stderr not becoming transcript text, and language
  metadata parsing;
- faithful Markdown and machine metadata, Vault containment, path traversal
  refusal, and raw-source preservation;
- Intake handoff receipt, Intake idempotency, handoff failure recovery without
  retranscription, and restart recovery;
- bounded retry, transcription-vs-handoff failure distinction, paper-jam
  alert deduplication, manual retry, ignore, and source preservation;
- Prometheus pause/resume behavior, no duplicate after resume, command help,
  HTTP source inspection, and regression coverage through the full repository
  suite.

The suite does not claim a Rumor video test: video is intentionally outside
Rumor’s contract. It also does not claim a cloud fallback test because no
cloud implementation exists; the local subprocess boundary and local-only
configuration are the tested contract. Multilingual behavior is represented
by mocked/local language metadata handling; the live sample was recorded as
`auto` because the installed Whisper JSON did not expose a detected language.

The earlier Kernel 10 reportback’s live-Whisper limitation is retained as
historical Kernel 10 status and reconciled with the later local installation
and live evidence here.

## Repository verification

The final source passed:

```text
gofmt -w .                                  PASS
GOCACHE=/tmp/pantheon-go-cache go test ./... PASS
GOCACHE=/tmp/pantheon-go-cache go vet ./...  PASS
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel18 . PASS
```

The focused Kernel 18 run also passed:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./internal/pantheon -run 'TestKernel18' -count=1 PASS
```

## Live verification

The final candidate and installed/running executable all have SHA-256:

```text
43c4861526c6f28e1d9309d00db8be055396c545c4d6cd7979cc9a3813d1c294
```

The final runtime was installed at `/home/brick/pantheon-runtime/bin/pantheon`
and `pantheon.service` was restarted only. It was active with `ExecMainStatus=0`
and MainPID `1831770` at `2026-08-17 11:11:56 PDT`. The running `/proc` executable
matched the candidate hash.

The live configuration used the dedicated user path
`/home/brick/Pantheon Audio`, local
`/home/brick/whisper.cpp/build/bin/whisper-cli`, and
`/home/brick/whisper.cpp/models/ggml-large-v3.bin`. The prior legacy config was
preserved by the explicit migration backup:
`config/pantheon.json.20260817T152436.386291218Z.bak`.

### Successful watcher proof

The final installed runtime successfully processed a re-encoded WAV after
stable-file detection:

- source: `rumor-ef437e6d0a2e4ea4cec8cf4abd44e68e`;
- source hash: `sha256:9b2355139f1739437dcc19e49a30abbebdb436ab65fdec275a2b336dda6d7eac`;
- attempts: `1`;
- state: `completed`;
- transcript: `80 Pantheon Raw/Audio Transcripts/kernel18-final-runtime-watcher--9b2355139f173943.md`;
- metadata: matching `.json` sidecar;
- Intake occurrence: `intake-da43d8d8eb972dafb7f9f0567c0ac904`;
- Intake state: `received`;
- raw WAV remained present.

The read-back transcript contained the expected JFK sample sentence. The
corresponding Intake scan recorded `new: 1`, `failed: 0`, and
`chronology: complete`; the due Rumor queue was empty after completion.

### Successful manual proof and idempotency

The final installed runtime explicitly processed a separate MP3 outside the
watch path:

- source: `rumor-576a594ccbde1a0a17dda2d21040e30f`;
- source hash: `sha256:8ec030673a2594676c3b91da06fb57226f63a8f64d855b57243d1f1b969f2bc6`;
- attempts: `1`;
- state: `completed`;
- Intake occurrence: `intake-59a0d1735012c3d330ec4f4d18b5ab09`;
- Intake state: `received`.

Repeating the earlier manual source command returned its existing source ID,
artifact paths, and `attempts: 1`; it did not create a second transcript.

### Paper jam, alert, retry, and ignore

The corrected runtime processed invalid WAV source
`rumor-f0f8439371de46972e89aeecdf579cdf` through exactly three attempts. It
ended with `state: failed`, `failure_kind: transcription`,
`last_error: Whisper produced no transcript`, and `alert_sent: true`. No
transcript or Intake occurrence was created for the corrected paper-jam
fixture. Manual retry returned it to `queued`; the watcher retried it, and
manual ignore ended it in `ignored` while preserving the raw file.

### Pause and resume

Prometheus pause returned the canonical pause message. A new
`kernel18-paused.wav` arrival was not registered during the four-second pause
window. Resume returned “Skipped work will not replay automatically”; the
fixture was then handled by the normal bounded lifecycle and ended failed
after three attempts with one deduplicated alert. Its raw file remained
present.

### Corrective verification note

Before the final stdout/stderr safety fix, an invalid fixture was mistakenly
persisted as a transcript containing Whisper diagnostic stderr because an
exit-zero/no-output case was treated as combined output. That artifact and
source were preserved rather than deleted. The implementation now separates
stdout from stderr, rejects an empty transcript, adds a regression test, and
the fresh final paper-jam proof above verifies the corrected behavior. The
earlier artifact is not counted as successful evidence.

## Files and boundary

Kernel 18 added Rumor runtime/configuration, CLI/API inspection and control,
focused tests, active documentation, and this reportback. It did not add
Future, Apollo semantics, video handling, cloud fallback, new authority,
Kernel 19, or any later architecture.

The next kernel remains unstarted and should be chosen from actual operating
friction after this runtime is used.
