# Pantheon Configuration

This paper describes the configuration architecture established by Kernel 8 and extended by Kernels 9–10 and 18. It documents implemented behavior and distinguishes it from configuration reserved for later Kernels.

The operator-facing audit of every meaningful setting is maintained in
[`current/configuration-inventory.md`](current/configuration-inventory.md).

## Canonical file

The canonical per-user configuration file is:

```text
~/pantheon-runtime/config/pantheon.json
```

Pantheon also keeps capability declarations separate at `config/capabilities.json`. Configuration may make a capability available, but it does not authorize use. Grants, policy enforcement, and later Mercury seals remain separate.

## Current schema

Kernel 8 uses `schema_version: 1` and nested namespaces. A minimal current configuration is:

```json
{
  "schema_version": 1,
  "runtime": {
    "listen": "127.0.0.1:8765"
  },
  "paths": {
    "vault": "",
    "roadmap": "",
    "intake": "",
    "father_time_chronicles": "",
    "raw": ""
  },
  "models": {
    "apollo": {
      "model": "mistral-small3.1:latest",
      "endpoint": "http://127.0.0.1:11434/api/chat"
    },
    "archimedes": {
      "model": ""
    }
  }
}
```

Active fields in Kernel 8 are:

| Field | Type | Default | Status | Owner | Restart required |
| --- | --- | --- | --- | --- | --- |
| `schema_version` | integer | none | active | Pantheon | yes |
| `runtime.listen` | string | `127.0.0.1:8765` | active, fixed localhost boundary | daemon | yes |
| `paths.vault` | absolute path | empty | active/optional | Past and Weave paths | yes |
| `paths.roadmap` | absolute path | empty | active/optional | Apollo context | yes |
| `models.apollo.model` | string | none | active/required | Apollo | yes |
| `models.apollo.endpoint` | URL | none | active/required | Apollo | yes |
| `models.archimedes.model` | string | empty | compatibility model identifier | Archimedes | no authority |
| `archimedes.enabled` | boolean | `false` | opt-in external transport | Archimedes/Bifrost | yes |
| `archimedes.provider` | string | `local-fake` | `local-fake` or `openai-compatible` | Archimedes | yes |
| `archimedes.endpoint` | URL | empty | credential-free HTTPS; loopback HTTP only for tests | Archimedes | yes |
| `archimedes.credential_env` | environment variable name | `PANTHEON_ARCHIMEDES_API_KEY` | name only; secret is external to config | service boundary | no |
| `archimedes.capability` | string | `archimedes-provider` | exact enabled Bifrost capability | Bifrost | yes |

Apollo remains restricted to HTTP endpoints on approved local hosts: loopback, localhost, and the configured Docker-host bridge. Endpoint credentials are not accepted as ordinary configuration.

## Kernel 9 active fields

Kernel 9 activates the following fields while preserving the same explicit migration boundary:

```json
"paths": {
  "intake": "/home/example/Pantheon Intake",
  "father_time_chronicles": "/home/example/Vault/00 Pantheon Core/Chronicles/Father Time"
},
"retention": { "raw_input_days": 30 },
"time": { "timezone": "America/Los_Angeles" }
```

When `paths.intake` is absent, Pantheon resolves `~/Pantheon Intake` at runtime and may create that default when `pantheon intake scan` is explicitly invoked. A configured but missing custom Intake path is reported rather than created. When `paths.father_time_chronicles` is absent, it resolves beneath `paths.vault` at `00 Pantheon Core/Chronicles/Father Time/`. An empty timezone uses the machine's local timezone. Invalid explicit timezones and non-positive or implausibly large retention values fail validation.

The raw retention value is policy only. New Intake occurrences begin as `awaiting_verified_markdown` with no eligibility or due date; Kernel 9 never moves or deletes source material.

## Kernel 29 Zeus fields

Zeus is an explicit machine-garbage boundary. It is disabled by default and
does nothing unless `zeus.enabled` is true and `zeus.roots` contains absolute,
machine-owned roots:

```json
"zeus": {
  "enabled": true,
  "roots": ["/tmp/pantheon-go-cache", "/home/example/pantheon-runtime/scripts/__pycache__"],
  "quarantine_path": "/tmp/pantheon-zeus-quarantine"
}
```

Kernel 29 recognizes only Go build caches and Python `__pycache__` roots as
recreatable cache candidates. Unknown roots, symlinks, source material,
Vault/Intake files, and non-regular objects remain protected. `pantheon zeus
inventory` and `pantheon zeus dry-run` write durable receipts and hashes but do
not quarantine or delete files. A later quarantine action requires a separate
exact Future/Mercury order.

Legacy flat configuration remains readable and receives these Kernel 9 defaults in memory. The canonical file is not silently rewritten; use the explicit migration command when you choose to persist the current schema.

## Kernel 10 active fields

Kernel 10 activates `paths.raw` with the default `<paths.vault>/80 Pantheon Raw`. It also activates local processor configuration under `ingestion.whisper` and `ingestion.media.converter`. Missing optional processors make only the corresponding source type unavailable; they do not invalidate the entire configuration. No cloud or frontier processor is used.

## Kernel 18 active fields

Rumor is enabled only when a user explicitly configures it and supplies one or
more absolute watch paths. An empty or disabled watcher is inert. The local
Whisper binary/model are inherited from `ingestion.whisper` unless
`rumor.whisper_model` overrides the model.

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

`watch_paths` is inspected recursively without following hidden/system or
symbolic-link entries. The installed local Whisper interface supports FLAC,
MP3, OGG, and WAV. Rumor writes a durable Markdown transcript and JSON
metadata under the configured Vault destination, never deletes the raw audio,
and hands the transcript to the existing folder-based Intake process. Failed
transcription and failed Intake handoff remain distinct durable states.

## Reserved namespaces

The following namespaces have a documented place but are not active merely because they appear in the schema model or this table:

`meta`, `present`, `past`, `weave`, `future`, `prometheus`, `notifications`, `archimedes`, `panopticon`, `setup`, and `organs`.

Future examples include Intake paths under `paths`, raw retention under `retention`, work windows under `time`, attention defaults under `notifications`, Mercury rounds under `mercury`, provider settings under `archimedes`, and display preferences under `panopticon`. Kernel 8 does not activate those behaviors, scheduling, purge, frontier calls, or an installer.

Kernel 33 adds an opt-in OpenAI-compatible Archimedes transport behind the
existing privacy gate and Future/Mercury path. It remains disabled unless the
endpoint is valid, the named credential is present at invocation time, and the
exact Bifrost capability is installed, approved, and enabled. No credential
belongs in this file; use a user-owned secret mechanism attached to
`pantheon.service`.

Kernel 11 activates this small Mercury namespace:

```json
"mercury": {
  "mode": "hermes",
  "agent": "Terra",
  "binary": "",
  "deterministic_fallback": true,
  "round_on_cron_wake": true
}
```

The binary is optional. Kernel 11 does not install Hermes/Terra or create a new recurring background loop; the explicit `pantheon mercury round` command is the safe integration point for an existing controlled wake.

Kernel 12 activates `paths.apollo_reports` and bounded local Apollo settings. The report path defaults beneath the Vault at `70 Pantheon Reports/Apollo`; external research remains disabled:

```json
"paths": { "apollo_reports": "" },
"apollo": {
  "max_retrieval_rounds": 12,
  "max_sources": 64,
  "max_context_bytes": 200000,
  "max_reasoning_duration_seconds": 900,
  "require_citations": true,
  "external_research": { "enabled": false }
}
```

Reserved values belong in documented schema space rather than being emitted as a large collection of false implementation claims in every user's file.

## Legacy compatibility and migration

The prior flat shape remains readable:

```json
{
  "local_model": "mistral-small3.1:latest",
  "ollama_url": "http://127.0.0.1:11434/api/chat",
  "frontier_model": "",
  "vault_path": "/path/to/vault",
  "roadmap_path": "/path/to/roadmap.md"
}
```

Legacy values normalize internally as follows:

| Legacy field | Current field | Status |
| --- | --- | --- |
| `model` or `local_model` | `models.apollo.model` | deprecated/readable |
| `ollama_url` | `models.apollo.endpoint` | deprecated/readable |
| `frontier_model` | `models.archimedes.model` | deprecated/readable; inactive |
| `vault_path` | `paths.vault` | deprecated/readable |
| `roadmap_path` | `paths.roadmap` | deprecated/readable |

Migration is never automatic:

```text
pantheon config migrate
```

This is a non-destructive dry run. It prints the proposed current-schema file and writes nothing. Apply it explicitly:

```text
pantheon config migrate --apply
```

Apply creates a timestamped `pantheon.json.*.bak` backup, validates a temporary file, and atomically replaces the canonical file with restrictive permissions. Migration is idempotent; current-schema configuration reports that no migration is required. Unknown legacy fields cause migration to stop rather than silently reinterpret or discard user configuration.

## Inspection and validation

```text
pantheon config path
pantheon config validate
pantheon show config
```

Validation does not invoke a model. It checks JSON shape, schema version, required Apollo fields, approved endpoint locality, and deterministic path constraints. It does not claim that every optional path or external resource currently exists.

`show config` displays the resolved effective configuration and identifies user values versus defaults. It does not display control tokens, passwords, API keys, or other secret values.

## Bifrost mail defaults

The optional `bifrost` configuration describes the first external crossing; it
does not authorize it. The current shape is:

```json
{
  "bifrost": {
    "enabled": true,
    "mail": {
      "account": "gmail",
      "mailbox": "INBOX",
      "capability": "gmail-mail",
      "schedule": "hourly",
      "max_messages": 50
    }
  }
}
```

The first normal mail intake establishes a safe baseline. `--bootstrap` is an
explicit bounded historical batch, never an unlimited mailbox discovery. The
configured capability still must be registered, fully approved, and enabled
through Bifrost before any connector invocation.

Unknown future schema versions are rejected. Pantheon does not guess how to interpret them.

## Configuration is not authority

Configuration describes available machinery and defaults. It does not authorize arbitrary execution. A configured email capability, model, service, or organ still requires the applicable policy and grant boundary.

## Prometheus Protocol Zero

The direct human control path does not depend on configuration flags, a model, a grant, Mercury, or another Pantheon organ:

```text
pantheon prometheus protocol --status
pantheon prometheus protocol --pause
pantheon prometheus protocol --resume
pantheon prometheus protocol --hard-stop
```

`--pause` uses the persistent daemon pause state and a local control marker so the request remains effective when the daemon is unavailable. It blocks new reasoning, external execution, consequential grants, and maintenance paths currently covered by the daemon. Safe status and configuration inspection remain available. Skipped work is not replayed automatically.

`--hard-stop` calls the user's service manager directly:

```text
systemctl --user stop pantheon
```

It stops execution beneath normal daemon cooperation and does not delete Vault data, Weave, configuration, audit history, or human source material. When the daemon is unavailable, hard-stop telemetry may exist only in the service manager's records.

The existing `pantheon pause` and `pantheon resume` commands remain compatibility aliases for now.
