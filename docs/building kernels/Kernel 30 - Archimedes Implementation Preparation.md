# Kernel 30 — Archimedes Implementation Preparation

## Status

Implementation complete for the local-only increment — frontier provider disabled.

This paper turns the Kernel 30 exchange contract into an implementation
sequence. It is deliberately smaller than a provider integration: the first
working increment proves the privacy, authority, exchange, return, and
retention boundaries with a local fake provider.

The governing contract remains [Archimedes Frontier Contract — Draft
0.2](../Archimedes%20Frontier%20Contract%20%E2%80%94%20Draft%200.2.md), and the
full exchange design remains in [Kernel 30 — Governed Frontier Reasoning
Exchange](Kernel%2030%20-%20Archimedes%20Frontier%20Exchange.md).

## What is ready

The repository already has the surrounding boundaries Archimedes must use:

- Apollo is local reasoning and proposal-making.
- Future administers exact authority and creates the order.
- Mercury seals and carries the order.
- Bifrost governs the external capability and records the invocation.
- Zeus owns later inventory of temporary exchange artifacts.
- Panopticon is the inspection surface.

Archimedes must not create a parallel authority path. Its first implementation
is a bounded exchange service inside the existing daemon, with a provider
interface that is fake in tests and unavailable in the live configuration.

## First implementation increment

The first increment is complete when Pantheon can run this entirely locally:

```text
Apollo referral
    ↓
Future-ready exact exchange proposal
    ↓
exchange-scoped semantic minimization
    ↓
deterministic privacy gate
    ↓
fake provider adapter
    ↓
untrusted response validation
    ↓
local rehydration and durable result
    ↓
Zeus retention metadata
```

The fake provider must not be a shortcut around the real boundary. It receives
the same sanitized missive structure, returns the same response envelope, and
produces the same receipts that a later Bifrost adapter will use.

No real provider, network transport, Switchyard process, provider credential,
or public listener is part of this increment.

## Suggested implementation seams

Keep the first code pass in small deterministic units so every boundary can be
tested without a model or network:

| Seam | Responsibility | Must not do |
| --- | --- | --- |
| `archimedesReferral` | validate and persist a bounded local referral | authorize or send |
| `archimedesExchange` | open one exchange and enforce turn/order state | widen scope or invent authority |
| `archimedesMinimize` | replace unnecessary identity detail with exchange aliases | decide whether escalation is wise |
| `archimedesPrivacyGate` | detect configured disclosure findings and apply strike policy | use a model or repair meaning silently |
| `archimedesProvider` | fake-provider interface and response envelope | read files, call tools, or choose a provider |
| `archimedesReturn` | validate responses and associate them with the active exchange | trust provider instructions as authority |
| `archimedesRehydrate` | restore local aliases while the map exists | guess when the map is missing |
| `archimedesRetention` | emit typed Zeus metadata and destroy the map after closure | issue deletion or quarantine orders |

These names are implementation suggestions, not a new public API. The durable
truth remains the exchange and receipt records, not an in-memory status flag.

## Local data contract

The exchange records should follow the Kernel 30 model:

```text
archimedes_exchanges
  exchange_id, order_id, provider_capability_id, state, referral_type,
  question_hash, privacy_policy_version, opened_at, closed_at, error

archimedes_missives
  missive_id, exchange_id, turn, direction, manifest_path, body_sha256,
  source_manifest_hash, gate_state, gate_findings_count, sent_at, expires_at

archimedes_responses
  response_id, exchange_id, turn, provider_status, body_path, body_sha256,
  rehydration_state, received_at, expires_at

archimedes_privacy_incidents
  incident_id, exchange_id, missive_id, category, finding_count, action,
  state, created_at

archimedes_exchange_receipts
  exchange_id, translation_entry_count, map_destroyed, result_path,
  retention_metadata_path, destroyed_at
```

The translation map is intentionally absent from this schema. During an open
exchange it may exist only in local temporary state or a tightly scoped
protected file. It must never be sent to a provider, exposed through an API,
or copied into the durable result after destruction.

## Test-first proof

The focused Kernel 30 test should prove, without an external model:

1. A referral is a proposal and does not send anything.
2. An exchange requires the exact order, capability, and scope it records.
3. Clean text passes the privacy gate.
4. One finding is redacted, recorded, and allowed only if the result remains
   useful under the configured strike policy.
5. Two findings block transmission.
6. Aliases remain stable through a clarification turn and change on a new
   exchange.
7. The provider receives no raw name, identity map, Vault path, secret, or
   unrelated source material.
8. A provider response is treated as untrusted intelligence.
9. Rehydration succeeds only while the local map exists.
10. A missing map produces `rehydration_failed`; it never guesses.
11. A closed exchange emits Zeus retention metadata but no delete order.
12. Prometheus pause blocks new transport and does not replay it on resume.
13. A late response for a closed exchange is rejected.

The fake provider should return a deterministic answer containing at least one
exchange alias. That makes the return and rehydration proof visible rather than
testing only a one-way request.

## Configuration posture

The existing `models.archimedes.model` field remains informational until this
increment is proven. Add no provider secret and do not enable data egress as a
side effect of implementing the local exchange.

The safe initial state is:

```text
archimedes implementation: local fake provider only
archimedes live transport: disabled
provider capability: absent or unapproved
Switchyard: absent and disabled
```

If a later provider is selected, it must first be registered as a Bifrost
capability with an exact operation, target scope, network/data-egress facts,
approval events, and invocation receipt behavior. A configured model name is
not an approval.

## Handoff to Switchyard

Switchyard preparation begins only after the fake-provider exchange passes.
Switchyard may then replace the fake provider transport behind the same
`archimedesProvider` interface. It must not replace Apollo, the privacy gate,
Future, Mercury, Bifrost, or Zeus.

The handoff packet is exactly:

```text
approved capability ID
sealed order ID and digest
exchange ID and turn
sanitized missive body
missive body hash
explicit route/target ID
expiry and timeout bounds
```

It does not include the local Vault path, source files, identity map,
translation map, control token, or a general-purpose tool description.

## Exit criteria for Kernel 30 implementation

These exit criteria are now satisfied. The fake-provider acceptance tests pass
and the live inventory reports:

```text
Archimedes: deferred or local-test-only
frontier model: disabled
provider capability: not enabled by default
Switchyard: not installed as a runtime dependency
```

Only then should a separate Switchyard application proof begin.
