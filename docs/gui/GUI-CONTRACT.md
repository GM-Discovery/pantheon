---
type: gui-readiness-contract
status: active
updated: 2026-08-17
---

# Pantheon GUI Contract

This is the stable contract for Kernel 21. It is an audit output, not a
required visual layout. Kernel 21 may choose its own navigation and visual
language while preserving these domains, labels, authority classes, and
truth sources.

## Primary read surface

Use `GET /v1/system/inventory` as the initial read. It is local, read-only,
secret-free, and versioned with `contract_version: 1`. It supplies component
state, model configuration, bounded work counts, Prometheus state, and routes
for deeper inspection. The GUI must not scrape Go source, SQLite tables, or
arbitrary configuration fields.

The GUI may call the subsystem routes listed in
`docs/current/system-inventory.md` for details. A read route may contain
human-source paths or report metadata; the GUI must not treat that as an
authorization to mutate anything.

## Domains and labels

| Domain | Human-facing items | Technical source | Allowed actions |
| --- | --- | --- | --- |
| Home / Organism | Organism health, pending work, important failures, current model | system inventory, `/v1/organism`, `/v1/status` | inspect; open evidence |
| Pantheon | component roles and running/configured/deferred state | system inventory | inspect; open subsystem |
| Prometheus | **Pause Pantheon**, Resume, Hard stop, blocked activity, approvals | `/v1/status`, grants, Future, audit | Emergency or restricted controls with unmistakable confirmation |
| Intake | Incoming items, source, identity, processing state, warnings | `/v1/intake`, `/v1/present` | scan/process; never overwrite protected artifacts |
| Rumor | Audio watch, queue, transcription attempts, failures, raw retention | `/v1/rumor*`, processors | scan, transcribe, retry, ignore; local-only |
| Bifrost | capabilities, installation/presence, approvals, enablement, receipts, mail crossing | `/v1/bifrost*` | inspect/register/enable/disable; external use is Restricted |
| Models | Apollo local model, endpoint locality, citation policy, future Archimedes state | inventory, `/v1/apollo`, config | inspect/change config; frontier access stays disabled |
| Notifications | adapter health, destination label, recent failures | inventory, audit | send only through the fixed audited adapter |
| Work | proposals, temporal subjects, seals, Go, results | Future, Janus, Mercury, orders | inspect; execute only exact authorized work |
| Configuration | meaningful paths, models, budgets, Rumor, Mercury, Bifrost, retention | `pantheon show config`, config inventory | edit only typed validated settings; risk class visible |
| System | service/runtime health, logs/audit, version/build, diagnostics | systemd + inventory + audit | inspect and diagnostics; no hidden service mutation |

## Human-language labels

The GUI should use labels like these rather than exposing raw field names:

| Technical identifier | Human label | Explanation | Class |
| --- | --- | --- | --- |
| `prometheus.paused` | **Pause Pantheon** | Stop new consequential activity for inspection. Skipped work will not replay automatically. | Emergency |
| `models.apollo.model` | **Apollo local model** | The local model used for cited reasoning. | Significant |
| `apollo.external_research.enabled` | **Public research requests** | Allow Apollo to request bounded public research; this does not enable Archimedes or authorize execution. | Restricted |
| `rumor.watch_paths` | **Audio watch locations** | Local folders Rumor may inspect for stable audio files. | Significant |
| `rumor.whisper_model` | **Whisper model** | Local model used to transcribe audio; audio is not sent to a cloud service by this setting. | Routine |
| `bifrost.capabilities.gmail-mail.use_approved` | **Use approval — Gmail connector** | Allows the approved read-only connector to cross the external boundary. | Restricted |
| `bifrost.capabilities.gmail-mail.update_approved` | **Automatic updates — Gmail connector** | Records a separate future permission; Kernel 19 has no update engine. | Restricted |
| `retention.raw_input_days` | **Raw input retention policy** | Policy value only; retention is armed after verified Markdown and never silently deletes source material. | Significant |
| `mercury.deterministic_fallback` | **Local execution fallback** | Permit the deterministic local adapter path when no optional external Mercury helper is available. | Significant |

## Authority rules the GUI must preserve

- Models reason; deterministic machinery authorizes and controls; tools perform
  bounded operations.
- Presence, registration, installation, and capability metadata are not use
  permission.
- A Future proposal is evidence, not authorization. A sealed order is exact;
  Mercury cannot broaden it.
- Every restricted action should show the capability, scope, purpose, current
  approval, Prometheus state, and resulting receipt.
- Pause and revoke controls must not require Apollo, Future, Mercury, a
  frontier model, or the GUI's own reasoning layer.
- No GUI view should display secrets, control tokens, raw credentials, or
  hidden chain-of-thought.

## Deferred GUI discoveries

Kernel 21 may design views for notification destinations, service-manager
health, storage pressure, and plugin health. The current inventory identifies
what is known and what is not. It must show “unknown/unavailable/deferred”
rather than infer health from a missing route or a model answer.

