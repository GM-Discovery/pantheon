# Kernel 3 Reportback — Kill Switch and Policy Manifest

**Status:** active

**Supersedes:** none. This follows the Kernel 1 reportback template; Kernel 2 was not found in the repository at implementation time.

## 1. Identity and mandate — retained

Pantheon remains the local-first executive system. Models, agents, tools, and providers remain replaceable organs. Human correction and explicit authority remain higher-order constraints.

## 2. Current kernel surface — revised

Implemented in this increment:

- Persistent daemon pause state in local SQLite machine state.
- Localhost API controls: `POST /v1/pause` and `POST /v1/resume`.
- CLI controls: `pantheon pause` and `pantheon resume`.
- `pantheon status` reports the persisted `paused` state.
- While paused, daemon notification delivery is blocked and audited. The same rule is declared for future scheduled work, event-driven work, Mercury execution, and Archimedes calls.
- A human-readable policy manifest declares intended role and capability boundaries.
- A human-readable Vault permission manifest maps the currently discovered top-level Vault directories without changing any filesystem permissions.

The hard stop remains:

```text
systemctl --user stop pantheon
systemctl --user start pantheon
systemctl --user restart pantheon
```

## 3. Evidence and reportable state — revised

Pause, resume, and pause-blocked notification attempts receive SQLite audit events. The daemon intentionally does not replay skipped work after resume. This prevents a recovery action from silently triggering delayed external activity.

The audit database remains machine-state support, not canonical human knowledge. The Vault and Weave remain intended human-readable continuity.

## 4. Information boundaries — revised

The new `config/vault-permissions.yaml` is a declarative policy artifact, not OS enforcement. It records:

- Apollo and Past may receive authorized local identity-map access.
- Present writes fact-only observations and is denied the identity map by default.
- Mercury receives bounded instructions, not unrestricted Vault access.
- Archimedes has no filesystem, Vault, Weave, or identity-map access.
- An external packet may use stable references such as `Adult #<token>`; only Apollo/Past can re-identify them locally.

No directory ownership, ACL, encryption, Vault content, or identity map was changed. The identity-map location and external-packet location remain explicitly unresolved.

## 5. Risks and unresolved work — revised

- The pause gate currently blocks daemon notification delivery. Future scheduled/event-driven/Mercury/Archimedes paths must call the same gate when they exist.
- The policy YAML is not yet parsed and enforced by Go; current enforcement is intentionally limited to the tested pause state and existing capability validation.
- No Present observation writer, Weave lifecycle enforcement, email integration, Mercury implementation, Archimedes transport, Sol provider, Whisper ingestion, or Omi integration was added.
- OS-level directory separation remains a proposed second layer, pending a reviewed directory-by-directory permission plan.
- Full-Vault Apollo access is declared as policy, but no new Vault-reading implementation was added in this increment.

## 6. Expiry register — retained

Claims expire when behavior or ownership changes. A successor reportback must mark each inherited section retained, revised, superseded, or expired and cite evidence.

## 7. Next reportback instructions — revised

The next reportback should begin from this document. The next smallest safe implementation is a capability-grant model that makes the policy manifest executable for a narrow Mercury email operation: read/tag/archive/brief-note/draft only, never send. It should include a human-readable proposed change, matching diff, explicit approval record, and audit evidence before applying any self-change or OS-level Vault permissions.
