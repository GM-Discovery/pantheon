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
