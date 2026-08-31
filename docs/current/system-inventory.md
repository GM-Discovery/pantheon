---
type: canonical-current-inventory
status: active
updated: 2026-08-23
---

# Current System Inventory

This is the canonical current-state inventory for Kernel 27 and later work.
It describes the repository and runtime as implemented, not the full design
horizon. Kernel specifications and reportbacks remain historical evidence.

## The organism now

```text
Omi BLE / external webhook / local files / bounded mail
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
| Omi Intake Bridge | deterministic boundary | proven / optional gateway not installed | lossless Omi webhook intake, native Linux BLE capture, hashes, replay receipts, and handoff to Rumor or Present | no n8n dependency, identity inference, calendar/email action, or Zeus deletion |
| Present | deterministic | ready | faithful Markdown normalization, provenance, verification | no interpretation or authorization |
| Past | bounded retrieval | configured when a Vault path exists | cited local continuity | no source rewrite or authority |
| Weave | deterministic derived state | configured when a Vault path exists | local relationship/index maintenance and retrieval | no source ownership or authority |
| Apollo | local reasoning | configured when a local model and endpoint exist | cited analysis, recommendations, Future-ready evidence | no grant, seal, Go, or tool execution |
| Action drafts | deterministic handoff | ready | replay-safe typed calendar/email proposals and clarification work from Apollo | no implicit authority, provider installation, calendar write, email send, or Mercury execution |
| Seshat | deterministic registry | ready | package codes, fields, adapters, temporal contracts | no grants or execution |
| Father Time | deterministic chronology | ready | time references and yearly chronology | no importance or permission |
| Janus | deterministic temporal daemon | ready | threshold transitions and missed-work policy | no recurring scheduler, grant, seal, or execution |
| Future | deterministic authority daemon | ready | consume proposals, check grants/temporal state, seal exact orders | no semantic planning or tool calls |
| Mercury | deterministic execution daemon | ready through fallback | validate seals, issue Go, invoke bounded adapters, record results | no invented permission |
| Prometheus | control plane | running/paused | emergency pause, resume, approval, revocation | no reasoning or replay of skipped work |
| Rumor | deterministic local audio boundary | enabled when configured | stable-file detection, local Whisper transcription, timestamped artifacts, and Intake handoff | no automatic Echo handoff, identity, interpretation, or authority |
| Echo | local speaker-review boundary | available when configured | completed diarization import, transcript snippets, speaker-scoped review, and human-confirmed voice attribution evidence | no person assignment without review, no cloud upload, no authority |
| Fame | human continuity layer | available through local API/CLI | durable people, conversations, identity evidence, segment attributions, cited observations, and corrections | no automatic authority, full personal CRM, calendar write, or final Apollo report |
| Report assembler | deterministic continuity boundary | ready / Kernel 27 | replay-safe event and daily Vault evidence manifests with exact citations, hashes, and receipts | no prose composition, authority, calendar/email/notification action, or raw-material deletion |
| Bifrost | governed integration bridge | enabled when configured | capability registry, approvals, enablement, receipts, bounded adapters | no generic installer or autonomous capability acquisition |
| Mail | bounded workflow | enabled when configured | explicit Gmail intake, durable newest-to-oldest backlog, local Apollo understanding, deterministic filters, and Future/Mercury/Bifrost action proposals | no send, delete, archive, filter creation, or implicit authority |
| Notifications | local adapter | available if helper exists | fixed notification delivery boundary | no arbitrary command execution |
| Archimedes | reasoning boundary | local-only by default; external route opt-in | referrals, exact sealed-order exchange validation, exchange-scoped aliases, deterministic privacy gate, local fake-provider result, optional Bifrost-gated OpenAI-compatible transport, rehydration, and Zeus retention receipts | no raw Vault access, identity-map exposure, provider credential persistence, unapproved fallback, or operational authority |
| Zodiac / Libra | deterministic scheduler and configurable report constellation | enabled when configured | report recipes with scheduled or verified-intake triggers, source-family/path selectors, evidence thresholds, sections, multiple Vault outputs, recurrence, due instances, assembly/Apollo/Vault/notification receipts, catch-up policy, Needs You, and provider-neutral calendar projections | no language interpretation, arbitrary shell execution, authority creation, provider install, or automatic calendar/email write without an approved Bifrost adapter |
| Zeus | deterministic sanitation boundary | ready when enabled | inventory and dry-run sanitation of explicitly configured machine-owned cache roots, with durable candidate/run receipts | no deletion, content judgment, or human-material classification |

Zeus is enabled only by explicit `zeus.enabled` configuration and configured
absolute roots. Its first implementation records inventory and dry-run
receipts; it does not delete anything. No current component is authorized to
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
| Intake | `GET /v1/intake`, authenticated external receipts at `GET /v1/intake/external`, generic handoff at `POST /v1/intake/external`, and native Omi webhook at `POST /v1/intake/omi` |
| Present/processors | `GET /v1/present`, `GET /v1/processors` |
| Rumor | `GET /v1/rumor`, `GET /v1/rumor/queue`, `GET /v1/rumor/sources/{id}` |
| Echo | `GET /v1/echo`, `GET /v1/echo/voices`; controlled import/review routes |
| Fame | `GET /v1/fame`; people, evidence, conversation, extraction, observation, and correction routes |
| Apollo | `GET /v1/apollo`, `GET /v1/apollo/reports` |
| Action drafts | `GET /v1/actions`, `GET /v1/actions/{id}`, controlled `POST /v1/actions/{id}/accept` |
| Report assembly | `GET /v1/reports/assemblies`, controlled `POST /v1/reports/assemble`, and `GET /v1/reports/assemblies/{id}` |
| Seshat | `GET /v1/seshat/index`, `GET /v1/seshat/packages` |
| Janus | `GET /v1/janus`, `GET /v1/janus/transitions` |
| Future | `GET /v1/future`, `GET /v1/future/proposals` |
| Mercury | `GET /v1/mercury`, `GET /v1/orders` |
| Bifrost | `GET /v1/bifrost`, `GET /v1/bifrost/approvals`, `GET /v1/bifrost/invocations` |
| Archimedes | `GET /v1/archimedes`, `/referrals`, `/exchanges`, and `/exchanges/{id}`; controlled `POST /v1/archimedes/referrals` |
| Mail | `GET /v1/mail`, `GET /v1/mail/messages`, `GET /v1/mail/rules`, `GET /v1/mail/actions` |
| Zodiac / Libra | `GET /v1/zodiac`, `/today`, `/upcoming`, `/needs-you`, `/constellations`, `/report-definitions`, `/report-runs`; controlled `POST /v1/zodiac/round` and Panopticon report-definition routes |
| Zeus | `GET /v1/zeus`; `POST /v1/zeus/inventory` or `/v1/zeus/dry-run` |
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
- Kernel 26 acceptance evidence: `docs/reportbacks/kernel-26-omi-intake-bridge.md`.
- Kernel 27 contract: `docs/building kernels/Kernel 27 - Report Assembler and Daily Continuity.md`.
- Kernel 27 acceptance evidence: `docs/reportbacks/kernel-27-report-assembler.md`.
