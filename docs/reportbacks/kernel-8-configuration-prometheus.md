# Kernel 8 Reportback — Configuration Foundation and Prometheus Protocol Zero

## Status

Implementation and live verification are complete. This reportback is being reconciled into the accumulated Kernel 8–10 commit stack. Kernel 8 adds the versioned configuration rail, explicit migration and inspection commands, and Prometheus Protocol Zero. It does not add Intake, Father Time, scheduling, new autonomy, frontier calls, or later Kernel organs.

## A. Predecessor

- Branch: `trunk`
- Implementation commit: `7373adb` (`feat: schedule nightly Weave maintenance`)
- Inherited Kernel 7 reportback: `docs/reportbacks/kernel-7-weave-human-reviewed-concepts.md`
- Kernel 7 status at implementation time: its own reportback remains a completion candidate pending final source-policy review, verification, and explicit human commit approval. Kernel 8 does not rewrite that status.

The current HEAD is the actual predecessor used. The newly supplied Kernel 8 missive, roadmap, Constitution, and Frontier Contract were read as governing working documents and left otherwise untouched.

The older Kernel 7 reportback statement that no nightly refresh was enabled is historical. Later predecessor commit `7373adb` added scheduled/nightly Weave maintenance; Kernel 7 history is not rewritten.

## B. Constitutional status

No constitutional amendment was required. Kernel 8 implements the existing human-control, local-first, correction, and authority boundaries.

## C. Implemented configuration

Current schema version: `1`.

Active runtime fields:

```text
schema_version
runtime.listen                  fixed at 127.0.0.1:8765
paths.vault                     optional absolute path
paths.roadmap                   optional absolute path
models.apollo.model             required
models.apollo.endpoint          required, approved local HTTP only
models.archimedes.model         readable/resolved but inactive
```

The configuration type establishes reserved namespace space for `meta`, `time`, `retention`, `ingestion`, `present`, `past`, `weave`, `future`, `mercury`, `prometheus`, `notifications`, `zeus`, `archimedes`, `panopticon`, `setup`, and `organs`. These namespaces are documented but are not represented as implemented behavior merely because the space exists.

The actual installed `config/pantheon.json` remains in the legacy flat format. This was deliberate: startup does not silently rewrite it. `pantheon show config`, `pantheon config validate`, and dry-run migration inspect it successfully and identify it as legacy.

## D. Legacy migration

Recognized legacy fields are `model`, `local_model`, `ollama_url`, `frontier_model`, `vault_path`, and `roadmap_path`. They normalize to the current namespaces without changing the existing Apollo locality rule.

```text
pantheon config migrate
pantheon config migrate --apply
```

The first command reads and validates the source, prints the proposed current-schema document, and writes nothing. The second creates a timestamped `pantheon.json.*.bak` backup with mode `0600`, validates a same-directory temporary file, and atomically replaces the canonical file. Unknown legacy fields stop migration rather than being discarded or guessed. Current-schema migration is idempotent and reports that no migration is required.

Automated tests verify dry-run non-mutation, backup creation, current-schema output, idempotency, and refusal of unknown legacy fields.

## E. Prometheus Protocol

Implemented commands:

```text
pantheon prometheus protocol --status
pantheon prometheus protocol --pause
pantheon prometheus protocol --resume
pantheon prometheus protocol --hard-stop
```

The canonical human kill switch remains:

```text
pantheon prometheus protocol --pause
```

Behavior:

- `--status` directly checks `systemctl --user is-active pantheon`, probes the daemon status endpoint, and reports persisted pause state when the daemon or local marker makes it available. It does not use a model.
- `--pause` writes a `0600` local pause marker and synchronizes the daemon's existing SQLite pause state when reachable. It remains useful when the daemon is unavailable.
- `--resume` uses the daemon when reachable. If the daemon is unavailable, it clears persisted pause state directly, removes the marker, and starts the user service when it is not active. A failed service start restores the pause marker and persisted pause state.
- `--hard-stop` invokes `systemctl --user stop pantheon` directly and does not require daemon cooperation, a model, or a capability grant.

When reachable, the CLI makes a short best-effort audit request before the direct stop. Failure to write or reach that audit path does not delay or prevent the service-manager stop.
- Existing `pantheon pause` and `pantheon resume` remain available as compatibility commands.

The direct hard-stop path was exercised against the actual user service in the approved live verification environment. It returned success; the following status reported `service: inactive`, `daemon: unreachable`, and `control: hard-stopped`. The subsequent `--resume` returned success and restored `service: active`, `daemon: reachable`, `paused: false`, and `control: healthy`. A more restricted shell earlier denied the systemd bus, so live service verification requires the service-manager access described above.

## F. Pause enforcement

The common daemon pause gate now blocks:

- routed model input and explicit route requests;
- capability grant creation;
- notifications;
- Vault connection work;
- email inventory;
- Weave refresh/maintenance.

Read-only status and configuration inspection remain available. Existing read-only Weave retrieval, review, and dry-run paths remain inspectable. No skipped work is replayed on resume. This report does not claim that every future path is blocked; successor Kernels must use the established pause boundary.

## G. Security

Kernel 8 did not change:

- the daemon's fixed `127.0.0.1:8765` listener;
- approved local Apollo endpoint hosts or HTTP-only transport;
- control-token creation, lookup, or `0600` permissions;
- capability/grant separation;
- Vault symlink protections;
- pause no-replay semantics.

Configuration endpoints containing URL credentials are rejected. Resolved configuration output omits secret values and does not print the control token.

## H. Tests and verification

Automated verification:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./...
```

Result: `ok   pantheon  0.297s` after the sandbox restriction was lifted for the existing IPv6 `httptest` listeners.

Focused Kernel 8 tests also pass:

```text
GOCACHE=/tmp/pantheon-go-cache go test -run 'TestKernel8|TestLoadConfigRejectsNonLocalApolloEndpoint|TestDaemonPausePersistsAndBlocksExternalNotification' ./...
```

Static/build verification:

```text
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel8 .
```

Both pass. The exact requested `go build ./...` cannot write its default `pantheon` binary because this repository already contains a directory named `pantheon`; the explicit-output build succeeds without changing that existing layout.

Manual CLI verification against the installed legacy configuration:

```text
pantheon config path
pantheon config validate
pantheon show config
pantheon config migrate
pantheon prometheus protocol --status
```

These reported the canonical path, structurally valid legacy configuration, resolved values/defaults without secrets, a non-writing migration proposal, and daemon/systemd availability without invoking a model. Live Prometheus pause/resume and hard-stop/recovery were also verified against the actual user service; the final state was active, reachable, unpaused, and healthy.

The terminal welcome keeps the Prometheus stop graphic, inspection safety language, ANSI/`NO_COLOR` behavior, and a useful command list. The current list includes `pantheon present "request"`, `pantheon intake scan`, `pantheon show intake`, `pantheon show present`, `pantheon show processors`, `pantheon show father-time`, and the canonical pause command.

## I. Files changed

- `configuration.go`: schema parsing, legacy normalization, validation, resolved inspection, and atomic migration.
- `prometheus.go`: direct control protocol, systemd boundary, pause marker, and daemon-unavailable recovery.
- `main.go`: configuration and Prometheus CLI surfaces plus first-run pause mnemonic.
- `daemon.go`: pause-marker synchronization and common pause enforcement on consequential paths.
- `kernel8_test.go`: focused configuration, migration, Prometheus, and pause-gate tests.
- `config/pantheon.example.json`: current-schema example.
- `config/README.md`: current configuration usage.
- `docs/configuration.md`: human-readable configuration reference.
- `docs/README.md`: configuration paper index link.

The supplied untracked governing documents were not edited. The actual installed legacy configuration was not migrated automatically.

## J. Risks and unresolved work

- Systemd control is intentionally limited to the user's systemd service. Other init systems are unsupported.
- Live systemd verification requires access to the user's systemd bus; the approved live verification environment provided that access and the sequence completed successfully.
- Pause enforcement is explicit over the current consequential paths listed above, not a claim about future unimplemented paths.
- `models.archimedes.model` and other reserved fields are inactive.
- The installed configuration remains legacy until a human explicitly applies migration.
- `go build ./...` retains the repository's existing binary-directory naming collision; explicit output verification passes.

## K. Expiry conditions

This reportback's claims become stale if:

- configuration schema version or namespace mapping changes;
- Prometheus command semantics or the user service manager changes;
- pause enforcement moves to different execution paths;
- control-token storage or permissions change;
- Apollo endpoint locality policy changes;
- the installed configuration is explicitly migrated or replaced.

## L. Next safe successor

**Kernel 9 — Father Time and Intake Ledger**

Kernel 9 may rely only on the configuration and control behavior proven here. It must activate its own Intake, retention, and chronology fields without treating the reserved Kernel 8 namespaces as already implemented behavior.
