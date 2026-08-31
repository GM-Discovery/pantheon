# Kernel 30 — Archimedes: Governed Frontier Reasoning Exchange

## Status

PASS — local-only fake-provider exchange implemented and verified — 2026-08-24

This kernel turns the existing [Archimedes Frontier Contract](../Archimedes%20Frontier%20Contract%20%E2%80%94%20Draft%200.2.md) into an implementation boundary. It does not activate a provider, add a privacy-off mode, or grant a frontier model access to the Vault.

The implemented increment records referrals, validates exact Archimedes orders
through Mercury, minimizes explicit entities into exchange-scoped aliases,
runs the deterministic privacy gate, completes a local fake-provider exchange,
rehydrates the result locally, and emits a Zeus retention receipt. Real
frontier transport remains disabled.

## Purpose

Apollo is Pantheon's local reasoning organ. Archimedes is an optional referral
boundary for questions that local reasoning cannot responsibly or economically
solve.

The useful promise is:

> Send a frontier model the problem it needs to solve, not the life of the
> person whose problem it is.

Archimedes must increase reasoning leverage without making private continuity
the price of that leverage.

## Why this follows Zeus

Zeus is not the privacy gate. It cannot decide whether a name, relationship, or
detail should cross the frontier. Apollo and the deterministic Archimedes
privacy gate own that decision.

Zeus comes first because a real frontier exchange creates temporary machinery:

- sanitized missives;
- exchange-scoped alias metadata;
- provider request and response envelopes;
- privacy-gate receipts;
- clarification packets;
- transport diagnostics; and
- temporary return/rehydration material.

Kernel 29 gives those machine-owned artifacts an inventory, retention policy,
dry-run, quarantine, and receipt boundary. Kernel 30 may define the exchange
while provider use remains disabled until that lifecycle is proven.

This is a dependency on operational containment, not a dependency of
redaction on a file cleaner.

## Governing principles

1. Local reasoning is preferred where sufficient.
2. Apollo may recommend escalation; a model never authorizes frontier use.
3. Future creates or approves the exact frontier order.
4. Mercury carries only a valid sealed order.
5. Archimedes receives only an intentionally constructed missive.
6. No frontier provider receives raw Vault access, Weave access, local secrets,
   identity records, or a re-identification tool.
7. Semantic minimization and deterministic privacy checking are separate steps.
8. The translation map is local, exchange-scoped, temporary, and destroyed
   after successful rehydration.
9. Useful local results may endure; transformation machinery does not endure by
   default.
10. A provider response is intelligence, not authority.

## Role separation

| Role | Responsibility | Explicit non-responsibility |
| --- | --- | --- |
| Apollo | detect local reasoning limits; prepare a minimized missive; interpret returned reasoning | frontier transport, authority, final privacy approval by itself |
| Archimedes | conduct the bounded frontier exchange through a replaceable provider | local file access, authority, identity resolution |
| Privacy gate | catch deterministic disclosure failures after semantic minimization | deciding the meaning of the question or inventing aliases |
| Future | administer the exact frontier capability/order decision | semantic redaction, provider reasoning |
| Mercury | seal/Go validation, transport, polling, return receipt | enriching a packet with private context |
| Bifrost/provider adapter | bounded external crossing, if explicitly enabled | generic installation, hidden tool use, raw Vault retrieval |
| Zeus | lifecycle of closed temporary exchange artifacts | deciding what may be disclosed |
| Panopticon | inspect referrals, privacy incidents, exchange state, and results | a frontier “chat” shortcut or privacy bypass |

## Exchange flow

```text
Apollo local reasoning
        ↓
referral: needs_frontier_reasoning
        ↓
Future validates exact frontier capability and scope
        ↓
sealed Archimedes order
        ↓
Apollo builds local semantic-minimization manifest
        ↓
exchange-scoped aliases and translation map stay local
        ↓
deterministic privacy gate
   ┌────┴────────────┐
   │ pass            │ fail
   ↓                 ↓
Mercury/Bifrost   redact or return to Apollo
   ↓                 ↑
frontier provider ───┘
        ↓
Mercury collects response or clarification
        ↓
local rehydration using the temporary map
        ↓
Apollo/Pantheon durable result and citations
        ↓
exchange closes; Zeus receives retention metadata
```

The exchange remains one bounded unit across clarification turns until the
provider returns a sufficient result, the order is cancelled, the exchange
fails, or Pantheon closes it.

## Referral contract

Apollo should refer only when the local boundary is materially insufficient,
not merely because a frontier provider is configured. The referral must state:

```text
referral_type: needs_frontier_reasoning
reason: why local Apollo is insufficient
question: the bounded problem to solve
desired_product: answer | comparison | critique | synthesis | plan | other
local_evidence_refs: exact local citations used by Apollo
constraints: required limits and assumptions
uncertainties: what remains unknown
authority_effect: none
```

The referral is a proposal. It is not an order and it does not contain a raw
Vault bundle automatically.

Apollo must not refer a question outward when the real state is:

- missing local evidence that Past or Weave can retrieve;
- a person clarification that Pantheon can ask locally;
- an authority decision for Prometheus or Future;
- a provider capability that has not been approved; or
- a task that should be deterministic instead of reasoned.

## Frontier missive

The outbound missive is an explicit local artifact assembled for one exchange.
It may contain:

- the bounded question;
- relevant facts and exact local citations;
- necessary roles and relationships;
- chronology relevant to the problem;
- constraints and competing interpretations;
- uncertainty and evidence gaps; and
- the requested reasoning product.

It must not contain unrelated continuity merely because Pantheon has it.

The missive should include a manifest with:

```text
exchange_id
order_id
question_hash
source_manifest_hash
alias_manifest_hash
privacy_policy_version
provider_capability_id
created_at
expires_at
```

The manifest records provenance and policy versions. It must not become a
second copy of the private identity map.

## Semantic minimization

Apollo performs the semantic pass because deterministic pattern matching cannot
reliably recognize a distinctive biography, relationship, quoted phrase, or
combination of ordinary facts that identifies someone.

Apollo should remove or generalize:

- names and handles;
- email addresses, phone numbers, and account identifiers;
- exact addresses and unnecessary precise locations;
- employers, organizations, domains, and unique project names when not
  necessary;
- filenames and local filesystem paths;
- distinctive biographical combinations;
- quoted language that is not needed for the reasoning product; and
- metadata that reveals identity without improving the answer.

It should preserve what the frontier model actually needs:

- roles and relationships when material;
- relative chronology or generalized dates;
- constraints and dependencies;
- distinctions between parties;
- evidence quality and uncertainty; and
- stable exchange-scoped aliases.

The target transformation is:

```text
identity removed; reasoning context preserved
```

## Exchange-scoped aliases

Real entities are replaced with random, exchange-scoped identifiers such as:

```text
Person#518204
Organization#440193
Project#318775
Location#640188
```

Alias requirements:

- generated from secure local randomness or an equivalent non-semantic source;
- stable for the active exchange, including clarification turns;
- not derived directly from the real value;
- regenerated for later exchanges;
- never a permanent cross-exchange pseudonym; and
- never resolvable by the provider.

The local translation map may contain the real mapping while rehydration is
pending. It is not sent to Archimedes, Mercury's provider adapter, or any
frontier tool.

## Deterministic privacy gate

The gate follows Apollo's semantic pass and has a deliberately narrow role:

> Detect obvious disclosure failures and prevent or redact them.

It may check configured forbidden patterns, including:

- values in the local translation map;
- names and known identity values;
- email addresses and phone numbers;
- filesystem paths and home-directory fragments;
- account, order, or external service identifiers;
- identifying URLs and domains; and
- provider-specific forbidden metadata.

The gate must not use a model to decide whether it passed, silently repair the
meaning of the packet, or invent replacement identities.

### Strike policy

| Gate result | Action |
| --- | --- |
| zero findings | pass if order, provider, and scope checks also pass |
| one finding | replace the affected span with `<REDACTED>`, record an incident, notify Mercury/Panopticon, and continue only if the result remains useful and authorized |
| two or more findings | block transmission, return to Apollo for correction, and run the gate again |
| uncertain gate outcome | block transmission and mark `needs_you` |

A caught privacy miss is consequential even when redaction allows the packet to
continue. Repeated strikes should surface as a system problem, not disappear
into ordinary logs.

## Transport and authority

The first implementation must use a fake/local provider adapter in tests. A
real provider remains disabled until a person chooses a provider, reviews its
capability contract, and supplies any required Bifrost approval.

The live transport boundary is:

```text
Future exact order
        ↓
Mercury seal and Go validation
        ↓
Bifrost capability adapter, if approved
        ↓
provider request containing only the passed missive
```

The provider may not receive a general-purpose tool, filesystem mount, Vault
path, Weave query, identity lookup, or hidden continuation channel.

Provider/model metadata must be recorded locally. It does not itself authorize
the next exchange.

## Switchyard placement

If “Switchyard” means [NVIDIA-NeMo/Switchyard](https://github.com/NVIDIA-NeMo/Switchyard), it belongs here as an optional Archimedes transport application—not as a Pantheon kernel, deity, authority plane, or replacement for Bifrost.

Switchyard's relevant job is provider-neutral LLM traffic routing: protocol
translation, configured target selection, multi-backend routing, and
operational metrics. Its standalone server and embeddable routing library are
useful precisely because Archimedes should not hard-code one frontier provider.
The project currently describes itself as pre-alpha and not for production use,
so it should begin as an isolated experiment behind a fake provider and remain
disabled on the live path until independently reviewed. Its repository is
Apache-2.0 licensed; any adoption still requires a dependency and NOTICE audit
before distribution.

The intended placement is:

```text
Apollo referral
      ↓
local semantic minimization
      ↓
deterministic privacy gate
      ↓
Future exact order → Mercury → approved Bifrost capability
                                      ↓
                              local Switchyard proxy
                                      ↓
                         explicitly configured provider target
```

Switchyard must see only the already-passed sanitized missive. It must not sit
in front of the privacy gate, receive raw Vault context, choose whether
escalation is warranted, create a Future order, or invoke arbitrary tools.

For the first integration, use a `passthrough` route or an explicitly selected
target. Do not let a classifier or fallback policy silently choose a stronger
frontier model, expand the evidence bundle, or cross a provider boundary that
Future and Bifrost did not approve. Route choice may optimize transport after
authority and privacy are settled; it may not create either one.

Switchyard is therefore a likely Kernel 31/application-layer candidate:

- Kernel 30 defines and tests the provider-neutral Archimedes exchange;
- a later Switchyard app proves local proxying, route receipts, provider
  translation, failure behavior, and metrics; and
- Bifrost continues to govern the capability and external crossing.

The other planned Bifrost apps—such as n8n or MCP-based adapters—remain
separate capabilities. They should not be smuggled into Switchyard merely
because both happen to connect software.

## Clarification turns

When the provider needs clarification:

1. Mercury records the clarification request;
2. Apollo evaluates whether the request is relevant and safe;
3. Pantheon retrieves additional local evidence only through normal bounded
   paths;
4. the same exchange-scoped aliases remain in force;
5. Apollo builds a new minimized packet;
6. the deterministic privacy gate runs again; and
7. Mercury returns the packet to the same active exchange.

A clarification request never becomes permission for direct local access.

## Return and rehydration

The provider response returns through Mercury as an untrusted reasoning result.
Pantheon must validate the response envelope, associate it with the active
exchange, and preserve the raw frontier response only under the temporary
retention policy.

Rehydration occurs locally:

```text
Person#991034 should speak with Person#518204.
                         ↓
Jane should speak with Grant.
```

If the translation map is unavailable, Pantheon must not guess. It may preserve
the sanitized result temporarily for diagnosis, mark rehydration failed, and
ask for human review.

After the exchange is closed, the useful local result may be:

- returned to Apollo for synthesis;
- shown in Panopticon;
- included in a Libra report;
- preserved as a cited local conclusion; or
- turned into a proposal for Future.

The result remains intelligence. It does not become a grant, seal, Go command,
calendar event, email, or notification without a separate explicit path.

## Retention and Zeus handoff

Archimedes emits typed retention metadata when an exchange closes:

```text
exchange_id
artifact_category
artifact_paths_or_ids
exchange_closed_at
rehydration_state
durable_result_path
retention_policy_id
```

This metadata is not a delete order. Zeus inventories the exact temporary
artifacts later, checks Kernel 29 policy, and waits for Future/Mercury if
sanitation is approved.

The translation map has a stricter lifecycle than ordinary temporary traces:

1. exchange closes;
2. required response is returned;
3. local rehydration succeeds;
4. useful local result is committed; and
5. the translation map is destroyed promptly.

The audit may retain counts, hashes, timestamps, and a destroyed flag. It must
not retain the real alias mappings after destruction.

Sanitized missives, provider responses, gate reports, and diagnostics may use a
short configured retention period, with a default of 30 days from the existing
Frontier Contract. Zeus owns their later inventory and sanitation.

## Durable data model

The first schema should keep the exchange, privacy, and return boundaries
inspectable:

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

The translation map itself is local ephemeral state. It must not be exposed by
these read routes or serialized into a durable report after destruction.

## Operator surface

Panopticon should show:

- why Apollo referred the question;
- current exchange state and bounded turn count;
- provider capability and approval state without secrets;
- source-count and privacy-policy version, not raw private packets by default;
- privacy incidents and whether the exchange was blocked, redacted, or sent;
- returned result and rehydration state;
- translation-map destruction receipt; and
- temporary-artifact retention state delegated to Zeus.

The operator surface must not offer:

- “send all context”;
- “disable redaction”;
- direct frontier chat using the Vault;
- a provider-generated tool call;
- alias lookup from the frontier; or
- automatic conversion of a frontier recommendation into work.

## Failure rules

- Local evidence insufficiency remains local `needs_evidence`, not an automatic
  frontier send.
- Apollo cannot complete semantic minimization: block and mark `needs_you`.
- One privacy finding: redact, record, and continue only under the strike rule.
- Two or more findings: do not send; return to Apollo.
- Gate uncertainty: do not send.
- Provider unavailable: preserve local order/referral state; do not broaden
  disclosure or silently substitute an unapproved provider.
- Transport timeout: keep the exchange bounded and retry only through explicit
  Mercury policy.
- Provider asks for unrestricted context: deny and convert to a clarification
  decision requiring the same minimization gate.
- Translation map lost: do not guess; mark rehydration failed.
- Prometheus pause: stop new frontier transport and do not replay skipped work
  automatically.
- Exchange closed: reject late provider responses unless a new explicit order
  opens a new exchange.

## Acceptance criteria

Kernel 30 passes its first implementation boundary when:

1. Apollo can produce an explicit `needs_frontier_reasoning` referral without
   sending anything.
2. Future/Mercury can represent and validate an exact Archimedes order with no
   implicit authority.
3. A fake provider can complete a bounded exchange through the same transport
   and receipt path intended for a real provider.
4. No raw Vault, Weave, identity map, secret, or unrestricted filesystem access
   reaches the provider.
5. Semantic minimization produces exchange-scoped aliases and preserves the
   relationships necessary for the test question.
6. The deterministic privacy gate blocks two or more findings, redacts one
   finding with an incident, and passes a clean packet.
7. Clarification turns reuse the same exchange aliases and pass the gate again.
8. A returned response can be rehydrated locally without exposing the mapping
   to the provider.
9. Losing the translation map never causes guessed identity restoration.
10. The map destruction receipt is durable while the mappings are not.
11. A completed exchange emits Zeus retention metadata but no delete order.
12. A frontier result remains a cited proposal or conclusion, never a grant,
    seal, Go command, calendar action, or email action.
13. Provider use is disabled by default and requires an explicit reviewed
    capability boundary.

## Non-goals

Kernel 30 does not implement:

- unrestricted frontier chat;
- a privacy-off setting;
- direct Vault or Weave access at the frontier;
- permanent provider-facing identities;
- identity resolution by Archimedes or provider tools;
- a generic web-research browser;
- provider installation or automatic model selection;
- autonomous frontier escalation without Future/Mercury;
- calendar, email, notification, or other consequential execution; or
- semantic sanitation by Zeus.

## Governing reference

This implementation kernel is subordinate to
`docs/Archimedes Frontier Contract — Draft 0.2.md`. If this kernel and that
contract conflict, the stricter privacy and authority boundary applies until
the contract is deliberately revised.
