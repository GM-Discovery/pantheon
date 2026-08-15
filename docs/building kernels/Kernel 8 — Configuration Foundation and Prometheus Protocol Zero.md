# Kernel 8 — Configuration Foundation and Prometheus Protocol Zero

**Status:** Proposed implementation order  
**Constitutional authority:** Constitution of Pantheon v0.2  
**Roadmap position:** Kernel 8 of the current construction roadmap  
**Expected successor:** Kernel 9 — Father Time and Intake Ledger

---

# 1. Purpose

Kernel 8 establishes two foundations before Pantheon gains additional autonomous organs:

1. a coherent, versioned configuration architecture that future Kernels extend rather than bypass; and
2. Prometheus Protocol Zero, the permanent direct human control path over Pantheon.

The result of this Kernel should be:

> Pantheon knows how it is configured, can explain that configuration, can safely migrate older configuration, and can always be stopped directly by the person without consulting a reasoning model or another Pantheon organ.

Kernel 8 does **not** add new reasoning capability or proactive processing.

Its purpose is to ensure that all such later capability is constructed on stable rails.

---

# 2. Why This Kernel Comes Now

Pantheon already has multiple implemented concerns with configuration requirements:

- local Apollo model and endpoint;
- optional frontier model name;
- Vault path;
- roadmap path;
- configured capability registry;
- daemon state;
- grants;
- Weave behavior;
- email integration;
- local control token.

The current Go configuration structure remains small and flat:

```text
model
ollama_url
local_model
frontier_model
vault_path
roadmap_path
```

The daemon currently loads:

`~/pantheon-runtime/config/pantheon.json`

and separately loads:

`~/pantheon-runtime/config/capabilities.json`

The current loader also enforces the important constraint that Apollo's Ollama endpoint must remain on an approved local HTTP host.

Future Kernels will add:

- Intake paths;
- storage;
- Father Time;
- retention;
- Present;
- Mercury;
- notifications;
- work windows;
- Prometheus;
- Zeus;
- Archimedes;
- Panopticon.tower;
- setup and installation state.

Those Kernels should not independently invent how configuration works.

Prometheus Protocol Zero also belongs here because further autonomous functionality should not be added before the person's permanent direct control path exists.

---

# 3. Starting State

Codex shall inspect the actual current repository before modifying it.

Do not infer repository state solely from this Kernel order.

At minimum, reconcile the implementation against the latest committed code and reportbacks.

Known existing behavior includes:

## 3.1 Runtime configuration

Pantheon currently loads per-user runtime configuration from:

`~/pantheon-runtime/config/pantheon.json`

Current configuration supports local model information, Ollama endpoint, Vault path, roadmap path, and an optional frontier model.

## 3.2 Capabilities

Configured capabilities exist separately from the runtime configuration.

Do not collapse capability declarations or grants into general configuration merely for convenience.

## 3.3 Pause

Pantheon already maintains persistent daemon pause state.

Current pause semantics block at least some external/background behavior and intentionally do not replay skipped work automatically after resume.

## 3.4 Hard stop

Pantheon can currently be stopped beneath the daemon through the user's systemd service:

```text
systemctl --user stop pantheon
systemctl --user start pantheon
systemctl --user restart pantheon
```



## 3.5 Local control token

The daemon already creates and uses a local control token stored under Pantheon state with restrictive permissions.

Do not weaken that mechanism.

## 3.6 Kernel 7

The supplied Kernel 7 report describes itself as a completion candidate pending final source-policy review, verification, and explicit commit approval.

Codex shall identify the actual committed predecessor.

Do not report Kernel 7 as completed merely because this Kernel expects to follow it.

---

# 4. Configuration Principles

Kernel 8 shall encode the following principles in implementation and documentation.

## 4.1 Configuration is human-readable

Canonical user configuration remains inspectable outside Pantheon.

Do not move critical human configuration exclusively into SQLite.

## 4.2 Configuration is not authority

A configuration value may make a capability available.

It does not authorize use of that capability.

For example:

```text
email:
  enabled: true
```

must not mean:

> Pantheon may perform arbitrary email operations.

Existing grants and future Mercury seals remain separate from configuration.

## 4.3 Secrets are not ordinary configuration

Secrets shall not appear in resolved configuration output.

The configuration architecture may reference secret locations or mechanisms.

Do not normalize credentials, API keys, control tokens, or equivalent secrets into an ordinary configuration dump.

## 4.4 Missing optional organs are valid

Pantheon configuration shall not assume every installation has:

- Archimedes;
- a dedicated Mercury model;
- transcription;
- additional storage;
- email;
- calendar;
- or any future optional organ.

A valid partial Pantheon is allowed.

## 4.5 Defaults should be useful

The user should not need to author a giant configuration document before Pantheon can function.

## 4.6 Configuration changes are explicit

Pantheon startup shall not silently rewrite the user's configuration into a newer format.

Migration is a deliberate operation.

---

# 5. Configuration Version

Introduce an explicit schema version.

Conceptually:

```json
{
  "schema_version": 1
}
```

The exact field naming may differ if the current repository has a compelling convention, but the result must be obvious and stable.

The loader must distinguish:

- supported current schema;
- recognized legacy flat configuration;
- unsupported future schema;
- malformed configuration.

Do not silently attempt to interpret an unknown future schema.

---

# 6. Configuration Structure

Establish a nested configuration structure appropriate for future extension.

The exact Go type decomposition may follow repository conventions, but the human configuration must have coherent namespaces.

Reserve room for at least:

```text
meta
runtime
paths
models
time
retention
ingestion
present
past
weave
future
mercury
prometheus
notifications
zeus
archimedes
panopticon
setup
organs
```

Not all namespaces are active in Kernel 8.

Reserved configuration must never be presented as implemented functionality.

---

# 7. Active Kernel 8 Configuration

Kernel 8 should migrate existing active configuration concepts into the new structure.

At minimum:

```text
schema_version

runtime.listen

paths.vault
paths.roadmap

models.apollo.model
models.apollo.endpoint

models.archimedes.model
```

If `runtime.listen` is configurable, its default must preserve the current localhost-only boundary.

Do not accidentally create a public listener.

If making the listener configurable would increase risk or scope without benefit, retaining it as a validated fixed default is acceptable.

Document the decision.

---

# 8. Apollo Locality

The existing approved local-host validation for Apollo shall remain.

Apollo's endpoint must not become externally configurable merely because the new configuration architecture is more flexible.

The current implementation recognizes approved local transports including loopback/localhost and the explicit Docker-host bridge already used by Pantheon.

Kernel 8 should preserve equivalent behavior.

A configuration migration must not broaden this boundary.

---

# 9. Reserved Future Configuration

Kernel 8 should document, but not necessarily activate, configuration concepts required by the roadmap.

At minimum reserve:

## 9.1 Intake and paths

```text
paths.intake
paths.processed_markdown
paths.raw_retention
```

## 9.2 Retention

```text
retention.raw_input_days
```

Document future default:

`30`

Do not implement purge behavior.

## 9.3 Time

```text
time.timezone
time.heavy_work_windows
time.quiet_windows
```

Do not implement scheduling.

## 9.4 Attention

```text
notifications.default_level
notifications.dismissed_resurface_years
```

Expected future defaults:

```text
brief
5
```

Do not implement the attention ladder in Kernel 8.

## 9.5 Mercury

Reserve configuration space for:

```text
mercury.rounds
```

Do not implement rounds.

## 9.6 Prometheus

Reserve configuration space for future agent-specific constraints.

Protocol Zero itself must not depend on an ordinary configuration flag being enabled.

The user must not be able to accidentally disable the emergency control path by setting:

```text
prometheus.enabled = false
```

or equivalent.

## 9.7 Archimedes

Reserve:

- provider identity;
- model selection;
- temporary transformation retention;
- provider-specific non-secret settings.

Do not implement frontier calls.

The Archimedes Frontier Contract remains future governing context.

## 9.8 Panopticon

Reserve control/display preferences.

Do not build Panopticon.tower.

## 9.9 Setup

Reserve room for progressive-installation state/configuration distinctions.

Do not implement the installer.

---

# 10. Active, Reserved, and Deprecated Fields

Pantheon should be capable of documenting configuration fields as:

**active**  
Used by current implemented behavior.

**reserved**  
Defined for a known future architectural purpose but not yet active.

**deprecated**  
Supported only for migration/backward compatibility.

Add a human-readable configuration reference under the project documentation.

For each field or logical group, document:

- purpose;
- type;
- default;
- required/optional;
- active/reserved/deprecated status;
- owner/organ;
- whether restart is required.

Do not generate hundreds of useless placeholder values in the user's actual config file.

Reserved schema space belongs primarily in the documented schema/types, not necessarily in serialized user configuration.

---

# 11. Legacy Configuration Compatibility

Existing flat configuration must continue to load.

A user upgrading Pantheon shall not need to manually reconstruct their configuration before the daemon can be inspected.

Legacy examples may resemble:

```json
{
  "model": "mistral-small3.1:latest",
  "local_model": "mistral-small3.1:latest",
  "frontier_model": "",
  "ollama_url": "http://172.17.0.1:11434/api/chat",
  "vault_path": "...",
  "roadmap_path": "..."
}
```

The loader may normalize these values internally.

It must not silently overwrite the file.

---

# 12. Configuration Migration

Provide an explicit migration interface.

Preferred conceptual commands:

```text
pantheon config migrate
pantheon config migrate --apply
```

## 12.1 Dry run

Without `--apply`:

- read the current config;
- determine whether migration is necessary;
- validate the proposed new configuration;
- show a human-readable proposed result or diff;
- write nothing.

## 12.2 Apply

With `--apply`:

1. read original config;
2. validate migration;
3. create timestamped backup;
4. write destination to a temporary file;
5. validate the temporary file;
6. atomically replace the canonical configuration;
7. preserve restrictive permissions appropriate to local configuration;
8. report the backup location.

Migration must be idempotent.

Running migration against current-schema configuration should clearly report that no migration is required.

## 12.3 Failure

If migration fails:

- the original configuration remains intact;
- the backup remains intact if already created;
- no partial configuration becomes canonical.

---

# 13. Configuration Validation

Implement:

```text
pantheon config validate
```

Validation should detect at minimum:

- malformed JSON;
- unsupported schema version;
- missing required active fields;
- invalid field types;
- invalid Apollo endpoint;
- invalid path shape where deterministically checkable;
- contradictory configuration that would prevent Pantheon startup.

Validation should distinguish:

> configuration is structurally valid

from:

> every optional resource currently exists.

For example, a configured future external storage path may be temporarily unmounted without making the configuration schema itself malformed.

Runtime health belongs to later inspection/doctor behavior.

---

# 14. Configuration Inspection

Implement:

```text
pantheon show config
```

This displays the **resolved effective configuration**.

It should show:

- schema version;
- active values;
- defaults applied;
- optional missing values where meaningful;
- status of relevant reserved areas where useful.

Where reasonably practical, distinguish:

- explicit user value;
- default value.

Never display:

- control token;
- secret provider key;
- passwords;
- other secret values.

If the current config is legacy, inspection may report that fact and suggest:

```text
pantheon config migrate
```

without forcing migration.

---

# 15. Configuration Path

Implement:

```text
pantheon config path
```

It should identify the canonical configuration file currently in use.

If a future or existing CLI override allows a different config path, this command should report the actual resolved path.

Avoid adding environment-variable precedence complexity unless the current repository already depends upon it.

---

# 16. Prometheus Protocol Zero

Kernel 8 shall establish the permanent direct human control surface.

Canonical commands:

```text
pantheon prometheus protocol --status
pantheon prometheus protocol --pause
pantheon prometheus protocol --resume
pantheon prometheus protocol --hard-stop
```

These commands are not reasoning requests.

They shall not invoke Apollo.

They shall not invoke Archimedes.

They shall not require Future.

They shall not require Mercury.

They shall not require an ordinary capability grant.

They exist so the person can control Pantheon when Pantheon itself is the problem.

---

# 17. Protocol Status

Implement:

```text
pantheon prometheus protocol --status
```

At minimum report:

- whether the Pantheon systemd user service appears running;
- whether the daemon API responds;
- persisted pause state if obtainable;
- whether control state is inconsistent.

The command must remain useful when the daemon is unavailable.

Example conceptual output:

```text
Prometheus Protocol

service: running
daemon: reachable
paused: false
control: healthy
```

or:

```text
Prometheus Protocol

service: stopped
daemon: unreachable
paused: unknown
control: hard-stopped
```

Do not involve a model in determining this state.

---

# 18. Protocol Pause

Implement:

```text
pantheon prometheus protocol --pause
```

Semantics:

> Stop new autonomous or consequential Pantheon execution while preserving the system for inspection and recovery.

Use the existing persistent pause state where appropriate.

Kernel 8 should centralize pause enforcement enough that new successor work has one obvious gate to consult.

Do not attempt a massive refactor of all existing code merely to achieve architectural purity.

Existing execution paths touched by this Kernel should use the common pause check.

---

# 19. Allowed Behavior While Paused

The following should remain available as appropriate:

- Prometheus Protocol commands;
- configuration inspection;
- configuration validation;
- status;
- read-only health/administrative inspection;
- other explicitly safe local control-plane operations.

The following should be blocked while paused:

- new model reasoning;
- new external tool execution;
- external mutations;
- scheduled/background work;
- destructive maintenance;
- future Mercury execution.

Existing Kernel functionality that cannot safely be classified should default toward blocked if it performs consequential work.

No skipped work automatically replays merely because the system resumes.

---

# 20. Protocol Resume

Implement:

```text
pantheon prometheus protocol --resume
```

If the daemon is running and merely paused:

- clear the pause state safely.

If the service is stopped:

- start the Pantheon user service;
- wait only as necessary to verify startup;
- restore normal unpaused control state where appropriate.

Do not require a functioning Pantheon daemon to begin the resume operation.

The OS service manager is below the daemon and may be used directly.

Resume does not replay skipped autonomous work.

---

# 21. Protocol Hard Stop

Implement:

```text
pantheon prometheus protocol --hard-stop
```

Semantics:

> Stop the Pantheon user service beneath normal daemon cooperation.

Use the local user service manager.

The command should remain meaningful when:

- the daemon is hung;
- the daemon API is unreachable;
- a model call is stuck;
- ordinary Pantheon processing cannot respond.

Hard stop must not delete:

- Vault data;
- Weave;
- configuration;
- audit history;
- pending human source material.

It stops execution.

It does not destroy continuity.

---

# 22. Existing Pause/Resume Compatibility

Existing:

```text
pantheon pause
pantheon resume
```

should remain usable for now.

They may:

- call the same underlying implementation;
- become compatibility aliases;
- or be marked as future-deprecated shortcuts.

Do not remove them in Kernel 8.

The Prometheus Protocol becomes the canonical taught interface.

---

# 23. First-Run / Entry Reminder

When the person enters Pantheon without a specific command, prominently display:

```text
Welcome to Pantheon.

Remember:
pantheon prometheus protocol --pause
```

Then display ordinary help/status information as appropriate.

This is intentionally analogous to learning a computer's emergency control sequence.

Do not turn it into an elaborate splash screen.

Do not require network access.

Do not depend on a model.

The reminder should remain easy to find even after the person becomes familiar with Pantheon.

---

# 24. Audit

Prometheus control events should remain auditable when the audit store is available.

Record, as appropriate:

- pause;
- resume;
- hard-stop request;
- recovery/start request;
- configuration migration;
- configuration validation failure where consequential.

However:

> inability to write an audit event must not prevent the person from invoking a hard stop.

Emergency human control outranks successful telemetry.

When the daemon is unavailable, OS-level hard-stop evidence may necessarily exist outside the normal SQLite audit path.

Document this limitation rather than weakening the emergency control.

---

# 25. Configuration Files and Documentation

Create or update human-readable project documentation for configuration.

At minimum document:

- canonical path;
- schema version;
- legacy compatibility;
- migration;
- active/reserved/deprecated distinction;
- configuration versus authority;
- secret handling;
- Prometheus Protocol Zero;
- how future Kernels add configuration.

Update the top-level documentation index to link the configuration paper.

Existing documentation already treats human-readable papers as governing construction references; retain that pattern.

---

# 26. Failure Behavior

## 26.1 Invalid configuration at daemon startup

Fail clearly.

Do not guess.

Report:

- field/path involved;
- invalid value category;
- suggested corrective action where known.

## 26.2 Unknown future schema

Refuse normal startup rather than interpreting the configuration incorrectly.

Prometheus Protocol and configuration inspection/recovery commands should remain as functional as possible.

## 26.3 Migration failure

Preserve original.

Do not leave partial canonical state.

## 26.4 Daemon unavailable

Prometheus Protocol status, hard-stop, and resume must remain meaningful through direct OS/service inspection.

## 26.5 systemd unavailable

Report the unsupported control condition explicitly.

Do not simulate success.

Packaging support for other init systems is outside Kernel 8 unless already present.

## 26.6 Audit unavailable

Normal consequential work may continue to respect existing audit-required behavior.

Emergency `--hard-stop` must still work.

---

# 27. Security Boundaries

Kernel 8 must not weaken:

- localhost daemon listener;
- approved Apollo local HTTP endpoint policy;
- control-token permissions;
- current Vault symlink protections;
- capability-grant distinction;
- pause no-replay rule.

Configuration flexibility is not justification for broadening trust boundaries.

---

# 28. Explicit Non-Goals

Do **not** implement:

- Father Time chronicle;
- Intake watcher;
- Intake processing;
- transcription;
- Present;
- Mercury orders;
- Mercury rounds;
- Future;
- BRIEF/QUEUE/NOTIFY/ALARM;
- individual Prometheus chains;
- Vulture Feed;
- Zeus;
- Archimedes calls;
- frontier privacy workflow;
- Apollo Internet research;
- scheduler/work windows;
- automatic retention purge;
- self.change execution;
- Panopticon.tower;
- graphical interface;
- progressive installer;
- package repository;
- generalized plugin marketplace.

Do not refactor Weave simply because a cleaner architecture occurs to the implementing agent.

Do not turn Kernel 8 into a full configuration-management framework.

---

# 29. Tests

Add focused automated tests.

At minimum verify:

## Configuration

1. legacy flat config loads;
2. current-schema config loads;
3. unsupported schema fails;
4. malformed config fails clearly;
5. required active field failure is specific;
6. Apollo endpoint outside approved local hosts fails;
7. valid local Apollo endpoint passes;
8. defaults resolve predictably;
9. resolved config does not expose control-token data;
10. migration dry run writes nothing;
11. migration apply creates backup;
12. migration apply produces valid current-schema config;
13. migration is idempotent;
14. migration failure preserves original.

## Prometheus

15. protocol status requires no model;
16. protocol pause requires no model;
17. protocol pause persists state;
18. paused state blocks at least the existing tested reasoning/execution path;
19. read-only control/config inspection remains available while paused;
20. resume clears pause;
21. skipped work is not automatically replayed;
22. hard-stop invokes the user-service stop path;
23. resume can invoke the user-service start path when daemon is stopped;
24. Prometheus control does not require an ordinary capability grant.

Where direct systemd integration is unsuitable for ordinary unit tests, isolate command execution behind a small testable boundary and test the intended commands without stopping the development environment.

---

# 30. Verification

Before reporting completion, run at minimum:

```text
gofmt
go test ./...
go build ./...
```

Also manually demonstrate the relevant CLI behavior in the actual runtime environment.

Capture actual output for:

```text
pantheon show config
pantheon config validate
pantheon config path
pantheon config migrate
pantheon prometheus protocol --status
pantheon prometheus protocol --pause
pantheon prometheus protocol --status
pantheon prometheus protocol --resume
```

Test `--hard-stop` and recovery carefully using the actual user service and report exactly what occurred.

Do not claim behavior was tested if it was only unit-tested through a mocked command runner.

Distinguish unit verification from live service verification.

---

# 31. Acceptance Criteria

Kernel 8 is a PASS only when all of the following are true:

1. Pantheon has an explicit current configuration schema version.
2. Legacy flat configuration remains readable.
3. Migration is explicit rather than automatic.
4. Migration dry run is non-destructive.
5. Applied migration is backed up and atomic.
6. Future configuration namespaces have an established place to live.
7. Reserved configuration is not represented as implemented behavior.
8. Configuration remains separate from authority.
9. Secrets remain outside normal resolved configuration output.
10. Apollo remains restricted to approved local transport.
11. Effective configuration is inspectable from the CLI.
12. Configuration can be validated without invoking a model.
13. Prometheus Protocol Zero exists.
14. The person can pause Pantheon without a reasoning model.
15. The person can hard-stop Pantheon beneath the daemon.
16. The person can recover/start Pantheon without first requiring the daemon to cooperate.
17. Prometheus Protocol does not require a normal capability grant.
18. Appropriate control-plane inspection remains available while paused.
19. Skipped autonomous work does not replay merely because Pantheon resumes.
20. The Prometheus pause mnemonic is prominently taught at Pantheon entry.
21. Existing unrelated tested functionality remains functional unless a deliberate documented migration was required.
22. Documentation accurately distinguishes what Kernel 8 implemented from what it merely reserves.

---

# 32. Required Kernel 8 Reportback

After implementation, produce:

# Kernel 8 Reportback — Configuration Foundation and Prometheus Protocol Zero

The reportback does not need to slavishly follow a universal template.

It must, however, answer the following.

## A. Predecessor

State:

- exact branch/commit implemented against;
- actual Kernel/reportback status inherited;
- whether Kernel 7 was committed/approved at implementation time.

Do not rewrite history.

## B. Constitutional status

State whether Kernel 8 required any constitutional change.

Expected:

`No constitutional amendment.`

If implementation reveals a real constitutional conflict, stop and report it rather than silently resolving it in code.

## C. Implemented configuration

Show:

- schema version;
- actual active namespaces/fields;
- actual reserved namespaces;
- actual deprecated/legacy fields;
- default behavior.

Include a concise example of the new canonical configuration.

## D. Legacy migration

Show:

- legacy input example;
- proposed migrated shape;
- dry-run behavior;
- backup naming/location;
- atomic replacement mechanism;
- idempotency evidence.

## E. Prometheus Protocol

Report actual behavior for:

```text
--status
--pause
--resume
--hard-stop
```

Explain:

- which paths use daemon API;
- which paths use direct systemd control;
- what still works when the daemon is unavailable;
- what works while paused.

## F. Pause enforcement

List the existing execution paths verified to respect pause.

Do not say:

> all future work is blocked

unless there is actually a central enforcement mechanism that guarantees that claim.

Instead state exactly what current code does.

## G. Security

Confirm whether Kernel 8 changed:

- daemon listener;
- Apollo endpoint restrictions;
- control-token handling;
- capability-grant boundaries;
- filesystem permissions.

Any change must be explicit.

## H. Tests

Provide:

- exact automated test commands;
- test counts/results;
- exact live verification commands;
- live outputs or concise observed results.

Clearly distinguish mocked systemd testing from actual user-service testing.

## I. Files changed

List each meaningful changed/added file with one-line purpose.

## J. Risks and unresolved work

Include anything such as:

- incomplete configuration migration edge cases;
- hard-stop platform limitations;
- config values reserved but inactive;
- pause paths still individually enforced;
- future need for Panopticon visibility.

## K. Expiry conditions

State which reportback claims become invalid if:

- configuration schema changes;
- Prometheus protocol semantics change;
- service manager changes;
- pause enforcement changes;
- control-token model changes.

## L. Next safe successor

Expected:

**Kernel 9 — Father Time and Intake Ledger**

Kernel 9 may assume only the configuration/control behavior actually proven by the Kernel 8 reportback.

It must not infer that reserved Intake, retention, or Father Time configuration became active merely because Kernel 8 documented those namespaces.

---

# 33. Stop Conditions for the Implementing Agent

The implementing agent should stop and report rather than improvise if it discovers:

- the actual repository materially contradicts the stated starting architecture;
- Kernel 7 has unresolved changes that would be overwritten;
- current configuration contains secrets that migration would expose;
- the current service is not actually a systemd user service;
- the existing pause mechanism has behavior materially different from the reportbacks;
- migration would destroy or silently reinterpret unknown user configuration;
- implementation requires weakening Apollo locality;
- implementation requires granting ordinary Pantheon code authority over Prometheus Protocol Zero.

Do not solve constitutional problems through convenience code.

---

# 34. Successor Boundary

Kernel 8 creates the rails.

Kernel 9 puts the first new cargo on them.

Kernel 9 shall activate:

- Intake path;
- raw-retention path;
- processed-output path;
- default 30-day raw retention;
- Father Time identity;
- hashing;
- yearly chronology;
- Intake ledger.

Kernel 9 shall **not** redesign configuration or emergency human control.

If Kernel 9 needs either of those redesigned, Kernel 8 was not complete.