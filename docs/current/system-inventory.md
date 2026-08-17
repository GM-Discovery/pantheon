---
type: canonical-current-inventory
status: active
updated: 2026-08-17
---

# Current System Inventory

This is the canonical current-state inventory for Kernel 20 and later work.
It describes the repository and runtime as implemented, not the full design
horizon. Kernel specifications and reportbacks remain historical evidence.

## The organism now

```text
audio / local files / bounded mail
              ↓
           Intake
              ↓
           Present
              ↓
     Past and Weave retrieval
              ↓
            Apollo
              ↓
   Seshat → Janus when needed
              ↓
           Future
              ↓
      sealed Mercury order
              ↓
     Bifrost or local adapter
              ↓
      result, evidence, audit
```

The normal local path is Intake → Present → Past/Weave → Apollo. Apollo
reasons and cites; it does not authorize or execute. Seshat supplies package
contracts, Janus applies declared temporal rules, Future administers existing
authority and seals exact work, and Mercury executes only a valid sealed order.
Bifrost governs crossings to external software and services.

## Implemented components

| Component | Kind | Current state | Responsibility | Deferred boundary |
| --- | --- | --- | --- | --- |
| Pantheon kernel | daemon | running when the service is active | configuration, orchestration, policy, audit, control | no sovereignty or hidden model authority |
| Intake | deterministic | ready | identity-track incoming files and occurrences | no semantic importance decision |
| Present | deterministic | ready | faithful Markdown normalization, provenance, verification | no interpretation or authorization |
| Past | bounded retrieval | configured when a Vault path exists | cited local continuity | no source rewrite or authority |
| Weave | deterministic derived state | configured when a Vault path exists | local relationship/index maintenance and retrieval | no source ownership or authority |
| Apollo | local reasoning | configured when a local model and endpoint exist | cited analysis, recommendations, Future-ready evidence | no grant, seal, Go, or tool execution |
| Seshat | deterministic registry | ready | package codes, fields, adapters, temporal contracts | no grants or execution |
| Father Time | deterministic chronology | ready | time references and yearly chronology | no importance or permission |
| Janus | deterministic temporal daemon | ready | threshold transitions and missed-work policy | no recurring scheduler, grant, seal, or execution |
| Future | deterministic authority daemon | ready | consume proposals, check grants/temporal state, seal exact orders | no semantic planning or tool calls |
| Mercury | deterministic execution daemon | ready through fallback | validate seals, issue Go, invoke bounded adapters, record results | no invented permission |
| Prometheus | control plane | running/paused | emergency pause, resume, approval, revocation | no reasoning or replay of skipped work |
| Rumor | deterministic local audio boundary | enabled when configured | stable-file detection, local Whisper transcription, Intake handoff | no live transcription, interpretation, or Project Fame |
| Bifrost | governed integration bridge | enabled when configured | capability registry, approvals, enablement, receipts, bounded adapters | no generic installer or autonomous capability acquisition |
| Notifications | local adapter | available if helper exists | fixed notification delivery boundary | no arbitrary command execution |
| Archimedes | future reasoning boundary | deferred | privacy-minimized frontier escalation | not implemented |
| Orion | future deterministic daemon | deferred | recurring schedules | not implemented |
| Zeus | future deterministic daemon | deferred | explicitly identified machine-garbage sanitation | not implemented |

Zeus is not currently implemented. No current component is authorized to
decide that human material is garbage.

## Deterministic inspection

The Kernel 20 inventory is available without a model at:

```text
GET /v1/system/inventory
pantheon system inventory
```

The response has `contract_version: 1` and contains component state, model
configuration without secrets, bounded work counts, Prometheus state, and
links to subsystem inspections. It intentionally omits the control token,
credentials, raw mail, transcript bodies, and full Apollo reports.

Other subsystem truth remains at its durable inspection routes:

| Domain | Route |
| --- | --- |
| runtime and pause | `GET /v1/status` |
| Intake | `GET /v1/intake` |
| Present/processors | `GET /v1/present`, `GET /v1/processors` |
| Rumor | `GET /v1/rumor`, `GET /v1/rumor/queue`, `GET /v1/rumor/sources/{id}` |
| Apollo | `GET /v1/apollo`, `GET /v1/apollo/reports` |
| Seshat | `GET /v1/seshat/index`, `GET /v1/seshat/packages` |
| Janus | `GET /v1/janus`, `GET /v1/janus/transitions` |
| Future | `GET /v1/future`, `GET /v1/future/proposals` |
| Mercury | `GET /v1/mercury`, `GET /v1/orders` |
| Bifrost | `GET /v1/bifrost`, `GET /v1/bifrost/approvals`, `GET /v1/bifrost/invocations` |
| organism trace | `GET /v1/organism` |
| authority | `GET /v1/grants`, `GET /v1/authority/check` |

## Truth locations

- Effective configuration: `config/pantheon.json`, resolved by the Go
  configuration loader.
- Capability declarations: `config/capabilities.json`.
- Package vocabulary: `config/plugins/registered/*.json`.
- Human policy declarations: `config/policy.yaml` and
  `config/vault-permissions.yaml`; Go remains the enforcement point.
- Durable runtime state and audit: the per-user SQLite database under
  `state/pantheon.db`.
- Human source and reports: the configured Vault and Intake roots.
- Current architecture and contracts: `docs/pantheon-architecture-brief.md`,
  this directory, and `docs/gui/GUI-CONTRACT.md`.
- Historical implementation evidence: `docs/reportbacks/` and
  `docs/building kernels/`.

