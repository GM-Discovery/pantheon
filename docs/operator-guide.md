# Pantheon Operator Guide

Pantheon is a local-first control system. The CLI exposes deterministic inspection and control locally; Apollo uses the configured local model, and no package registration grants execution authority.

## Panopticon

When the user service is running, open the embedded human control surface at
`http://127.0.0.1:8765/`. Panopticon is local-only and served by the Pantheon
Go daemon; it does not start a second server. Home shows live work, cell
health, the Father Time watch, and the red Radio Prometheus pause control.

Use the GUI for readable state, typed configuration editing, notifications,
Rumor attention, and governed Bifrost enablement. Use the CLI for complete
forensic output and the terminal-only hard-stop path. The GUI never exposes
the control token, raw audit tables, secrets, hidden chain of thought, or a
marketplace/download engine.

## Start and check

The user service is the normal runtime:

```text
systemctl --user status pantheon.service
pantheon status
pantheon system inventory
pantheon help
pantheon help --commands
```

With no daemon, `pantheon help`, `pantheon help --commands`, and `pantheon help --codes` still work because they read local metadata and do not require a model.

## Stop safely

Use the actual Prometheus Protocol command shown on the welcome screen:

```text
pantheon prometheus protocol --pause
pantheon prometheus protocol --status
pantheon prometheus protocol --resume
```

Pause blocks new Intake, Present, authority, Mercury, and Bifrost activity. Resume does not replay a blocked round; run the desired round explicitly after inspection.

## Bifrost external capabilities

Bifrost is the governed bridge between Pantheon and external software or
services. Presence is not permission, and research is not permission. Register
metadata, download, installation, and use are separate approval facts; a
registered capability is not invokable until it is installed, fully approved,
and explicitly enabled.

The first crossing is read-only Gmail mail intake through the existing local
Himalaya configuration:

```text
pantheon bifrost register --id gmail-mail --name "Gmail read-only mail" \
  --source himalaya --version local-configured --type command \
  --description "Bounded read-only mail intake" \
  --operation mail.read --resource gmail:INBOX --component Bifrost \
  --network --authentication-required \
  --prohibited-operation send --prohibited-operation reply \
  --prohibited-operation delete --prohibited-operation archive \
  --download-approved --installation-approved --use-approved
pantheon bifrost enable gmail-mail
pantheon bifrost mail intake
```

The first ordinary intake establishes a baseline and excludes historical mail.
An explicit `--bootstrap` may process one bounded batch of at most the
configured limit. The adapter reads messages, writes provenance-bearing
Markdown under Intake, preserves the provider source, and never sends,
replies, forwards, deletes, archives, or changes account settings. Allowed
attachments are copied under the bounded Intake artifact; other attachments
remain metadata-only and blocked. Hermes remains available until a separate
cutover proof retires its routine mail jobs.

## Mail intelligence

Kernel 22 adds an explicit, bounded mail-understanding round. It considers new
messages before the durable newest-to-oldest backlog and sends eligible
messages through Intake, Present, Past/Weave, and local Apollo. It does not
create a scheduler.

```text
pantheon mail status
pantheon mail check
pantheon mail messages
pantheon mail rules
pantheon mail actions
```

Apollo suggestions are evidence for Future, not permission. Mark-read, safe
labels, moves, and non-sent drafts require the separate `gmail-mail-actions`
Bifrost capability plus exact authority and Mercury sealing. Send, delete,
archive, filter creation, and attachment download are unavailable.

## Zodiac schedules

Kernel 23 adds the deterministic schedule keeper. Zodiac remembers explicit
obligations and recurring system work; it does not interpret natural language,
create grants, or silently install calendar connectors.

```text
pantheon show zodiac
pantheon zodiac today
pantheon zodiac upcoming
pantheon zodiac needs-you
pantheon zodiac round
```

Declare an explicit schedule with `pantheon zodiac schedule --name ...
--kind ... --start ... --action ...`. Due consequential work enters Future and
then Mercury. Apple/Google projection state is visible through Zodiac, but
actual provider writes require an approved Bifrost adapter. Human edits are
preserved and incompatible provider corrections become Needs You.

## Inspect the organism

```text
pantheon show intake
pantheon show present
pantheon show apollo
pantheon show future
pantheon show mercury
pantheon show janus
pantheon show transitions
pantheon show organism
pantheon show rumor
pantheon show rumor queue
pantheon grants
pantheon help --codes
pantheon show plugins
pantheon show packages
pantheon bifrost status
pantheon bifrost list
pantheon bifrost approvals
pantheon bifrost mail status
```

Apollo reports are durable Markdown files with citations. Future proposals show whether work is awaiting a person, referred to Prometheus, temporally blocked, sealed, or complete. Mercury orders and attempts provide the execution evidence.

## Normal integration path

For a bounded human-directed operation, use the whole-animal surface after Present has an occurrence:

```text
pantheon organism run \
  --occurrence <occurrence-id> \
  --request "Describe the bounded work" \
  --package NOTIFICATION \
  --scope notification:<local-scope> \
  --missive $'TYPE: NOTIFICATION\nTITLE: Pantheon\n\nA bounded local notification.'
```

The integration records the path Intake/Present → Past/Weave → Apollo → Seshat → Janus when relevant → Future → sealed Mercury order → adapter result. A Future proposal that lacks authority is retained for inspection and does not execute.

## Rumor audio intake

Rumor is the deterministic local audio boundary. It watches only the
configured `rumor.watch_paths` and can also receive an explicit local file:

```text
pantheon rumor scan
pantheon rumor transcribe <audio-path>
pantheon show rumor source <source-id>
pantheon rumor retry <source-id>
pantheon rumor ignore <source-id>
```

The installed local Whisper processor currently accepts FLAC, MP3, OGG, and
WAV. Rumor waits for a stable file, hashes it, bounds retries, preserves the
raw audio, writes a Vault Markdown transcript plus machine metadata, and
hands the transcript to the existing Intake process. Identical bytes do not
create a second transcript. Rumor does not summarize, prioritize, authorize,
or execute anything; its output remains evidence for Present and Apollo.

Prometheus pause prevents new Rumor scans and transcription. In-flight local
decoding is allowed to finish, but a paused completion does not hand off to
Intake. Resume does not replay a completed or ignored source.

## Omi Intake Bridge

The proven Linux live-audio route uses native BlueZ D-Bus and system `libopus`;
it does not require n8n or a Python package install:

```text
PYTHONDONTWRITEBYTECODE=1 python3 scripts/omi_dbus_capture.py \
  --device <omi-mac-address> \
  --duration 30 \
  --output "/home/<user>/Pantheon Audio/Omi BLE/omi-live.wav"
```

Place the completed non-empty WAV under a configured Rumor watch path, or use
the output directory directly. The native receiver connects through BlueZ,
synchronizes the Omi clock, subscribes to live Opus notifications, decodes the
audio, and refuses to publish an empty capture. The resulting path is:

```text
native Omi → WAV → Rumor → transcript → Intake → Present → Apollo
```

The webhook lane is separate:

```text
POST /v1/intake/omi
POST /v1/intake/external
```

Both lanes preserve source material and hashes. Omi labels and speaker claims
remain source evidence; they do not silently become Fame identity or authority.
Inspect the full acceptance evidence in
`docs/reportbacks/kernel-26-omi-intake-bridge.md`.

## Echo and Fame review

Echo and Fame are separate from Rumor's transcription boundary. Echo imports a
completed local diarization run and keeps speaker labels anonymous until a
human reviews them. Fame then records the conversation, attaches only reviewed
speaker/person mappings, and can run a bounded extraction of cited action
items, commitments, decisions, and unresolved questions.

```text
pantheon echo import <audio-path> <diarization.tsv>
pantheon show echo
pantheon fame person add <name>
pantheon fame conversation <echo-run-id>
pantheon fame process <echo-run-id> <speaker> <person-id>
pantheon fame extract <conversation-id>
pantheon show fame
```

Fame observations are evidence for later Apollo work. They are not calendar
events, notifications, email orders, or authority. Exact transcript citations,
anonymous labels, identity evidence, and human corrections remain inspectable.

## Report assembly and daily continuity

Kernel 27 adds a deterministic source-selection boundary for event and daily
reports:

```text
pantheon report assemble --date 2026-08-23
pantheon report assemble --kind event --date 2026-08-23 \
  --source "80 Pantheon Raw/Audio Transcripts/2026/08/example.md"
pantheon report assemblies
```

The daily form collects same-day verified Vault artifacts, Apollo reports, and
integrated event reports when no explicit source is supplied. It writes a
hash-checked evidence manifest under `10 Operations/Situation Briefs/`; event
assemblies go under `70 Pantheon Reports/Integrated Events/`. These manifests
record the exact source set for Apollo but do not compose prose or execute
calendar, email, notification, or retention actions. An identical replay
returns the existing receipt instead of overwriting it.

## Add a package

```text
pantheon plugin schema
pantheon plugin validate config/plugins/examples/notification.json
pantheon plugin register <manifest-path>
pantheon help --codes
pantheon show package <subject-code>
```

Registration changes vocabulary and packaging only. A package still needs an exact local adapter and an authority grant before Future can seal work.

## When work is blocked

Pause first if the state is unclear. Then inspect the relevant report, proposal, Janus transition, grant, and Mercury order. `needs_person`, `referred_prometheus`, `temporally_blocked`, `adapter_unavailable`, and `awaiting_authority` are safe states, not failures to hide. Resolve the boundary explicitly and run the relevant round again.
