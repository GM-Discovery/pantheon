# Daemon Milestone

## Status

Implemented as the smallest persistent Pantheon executive service. This record describes implemented behavior, not a claim that the complete policy harness or the Fates are complete.

## Process boundary

```text
systemd --user
  -> pantheon daemon
     -> 127.0.0.1:8765 only
        -> SQLite machine-state audit
        -> Ollama / Apollo for authorized local reasoning
```

The CLI is now a localhost API client for ordinary input, `status`, and `route`. It does not duplicate local reasoning logic.

## Implemented API

- `GET /v1/status` returns deterministic configuration, configured capabilities, listener boundary, and audit-event count.
- `POST /v1/route` invokes the existing routing model, then validates the result against the configured capability registry in Go.
- `POST /v1/input` routes first. Only `reason` is automatically performed in this milestone. Any other selected capability is recorded and refused pending explicit policy authorization.
- `POST /v1/notify` invokes the existing authenticated `pantheon-notify` helper with a fixed topic and title; it does not grant arbitrary command execution.

All endpoints are served only on `127.0.0.1:8765`; no public listener is created.

## Audit persistence

`state/pantheon.db` stores machine-state audit events: event ID, timestamp, endpoint, input, selected capability, model, outcome, result/error, and duration. It holds no secrets and does not replace the Vault or Weave.

## Authority boundary

Apollo is the configured local model running through Ollama. It reasons only after Pantheon has routed and selected the `reason` capability. The model does not grant itself authority. Hermes remains non-automatic. Archimedes, Present, Past trajectories, Future context, Whisper ingestion, Weave writes, scheduled work, and frontend delivery are out of scope for this milestone.

## Operations

The user service is installed at `~/.config/systemd/user/pantheon.service` and has a boot-persistent user manager because user lingering is enabled. The executable is `/home/brick/pantheon-runtime/bin/pantheon`.

Commands:

```text
systemctl --user daemon-reload
systemctl --user enable --now pantheon.service
systemctl --user status pantheon
journalctl --user -u pantheon -f
```

## Next safe increment

Define the policy-harness data model for capability grants and human approval, then move deterministic `notify` through that path with audit evidence. Do not attach Omi/Whisper ingestion until Present's durable observation contract and the audio/transcript retention policy are reviewed.
