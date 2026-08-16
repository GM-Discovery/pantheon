# Pantheon Configuration

This paper describes the configuration architecture established by Kernel 8 and extended by Kernels 9–10. It documents implemented behavior and distinguishes it from configuration reserved for later Kernels.

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
| `models.archimedes.model` | string | empty | reserved/inactive | Archimedes | no current behavior |

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

Legacy flat configuration remains readable and receives these Kernel 9 defaults in memory. The canonical file is not silently rewritten; use the explicit migration command when you choose to persist the current schema.

## Kernel 10 active fields

Kernel 10 activates `paths.raw` with the default `<paths.vault>/80 Pantheon Raw`. It also activates local processor configuration under `ingestion.whisper` and `ingestion.media.converter`. Missing optional processors make only the corresponding source type unavailable; they do not invalidate the entire configuration. No cloud or frontier processor is used.

## Reserved namespaces

The following namespaces have a documented place but are not active merely because they appear in the schema model or this table:

`meta`, `time`, `retention`, `ingestion`, `present`, `past`, `weave`, `future`, `mercury`, `prometheus`, `notifications`, `zeus`, `archimedes`, `panopticon`, `setup`, and `organs`.

Future examples include Intake paths under `paths`, raw retention under `retention`, work windows under `time`, attention defaults under `notifications`, Mercury rounds under `mercury`, provider settings under `archimedes`, and display preferences under `panopticon`. Kernel 8 does not activate those behaviors, scheduling, purge, frontier calls, or an installer.

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
