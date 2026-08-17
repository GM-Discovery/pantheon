# Software

## Kernel

The Pantheon kernel is the small, deterministic executive layer. It loads configuration, discovers available capabilities, applies policy, routes work, records audit state, and returns results. It is responsible for authorization, not for pretending to be the most intelligent component.

## Configuration

Configuration is local per user at `~/pantheon-runtime/config/pantheon.json`. It may declare an Ollama endpoint, local model, optional frontier model, Vault location, and governing context. Empty fields are valid; naming a model does not create credentials or grant it authority.

## Information flow

```text
Intake -> Present -> Past / Weave -> Apollo -> Seshat -> Janus when relevant
                                                       -> Future -> seal -> Mercury -> tool/plugin -> result/evidence
```

Present establishes the current source. Past and Weave provide cited continuity before Apollo reasons locally. Apollo writes a durable cited report and a Future-ready structured result. Seshat owns the deterministic package contract and adapter-facing formatting, so Apollo does not need to know external API encodings. A registered package describes form, not authority; Kernel 13 grants remain authoritative.

Hermes moves approved messages and actions between nodes. Hermes does not establish policy. Mercury carries the normalized package after authority has been resolved.

Father Time owns chronology. Janus applies only the temporal threshold rules declared by Seshat; Janus does not schedule recurring work, reason about importance, grant authority, issue Go, or invoke tools. Orion remains the reserved owner of recurring schedules.

Future is currently a deterministic authority-signing boundary, not a planning or reasoning engine. The Kernel 17 integration handoff supplies it with Seshat package identity, capability, scope, adapter, payload, provenance, and any Janus subject reference. Future consumes existing Kernel 13 grants, creates no new grant registry or cryptography, refers insufficient authority to Prometheus or the person, and seals an exact order only when an active Future grant and temporal eligibility are present. Mercury transports and records that order; Future does not issue Go, invoke tools, or interpret Apollo’s conclusions. Future’s later active-state consumer remains a subsequent kernel boundary.

The frontier model is an optional reasoning escalation. It receives an intentionally curated brief, never an automatic Vault dump.

## Past retrieval contract

Past searches only the configured Vault and never writes it. Its normal response contains two to five jumps, unless fewer credible jumps exist. Each jump includes source, location, excerpt, match type, confidence, age, and correction status. Older evidence is not discarded merely because it is old; recency adjusts ranking, while explicit correction and supersession govern validity.
