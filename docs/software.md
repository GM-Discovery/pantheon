# Software

## Kernel

The Pantheon kernel is the small, deterministic executive layer. It loads configuration, discovers available capabilities, applies policy, routes work, records audit state, and returns results. It is responsible for authorization, not for pretending to be the most intelligent component.

## Configuration

Configuration is local per user at `~/pantheon-runtime/config/pantheon.json`. It may declare an Ollama endpoint, local model, optional frontier model, Vault location, and governing context. Empty fields are valid; naming a model does not create credentials or grant it authority.

## Information flow

```text
speaker -> Present -> Past / Weave -> Future context contract -> Mercury -> Apollo
                                                                    |
                                              Future authority administration
                                                                    |
                                                           sealed order -> tool
```

Apollo writes a human-auditable missive in Pantheon’s small subject vocabulary. Seshat owns the deterministic package contract and adapter-facing formatting, so Apollo does not need to know external API encodings. A registered package describes form, not authority; Kernel 13 grants remain authoritative.

Hermes moves approved messages and actions between nodes. Hermes does not establish policy. Mercury carries the normalized package after authority has been resolved.

Father Time owns chronology. Janus applies only the temporal threshold rules declared by Seshat; Janus does not schedule recurring work, reason about importance, grant authority, issue Go, or invoke tools. Orion remains the reserved owner of recurring schedules.

Future is currently a deterministic authority-administration boundary, not a planning or reasoning engine. Kernel 12 Apollo results and later proposals may be supplied to Future with Seshat package identity, capability, scope, adapter, payload, provenance, and Janus subject references. Future consumes existing Kernel 13 grants, creates no new grant registry or cryptography, refers insufficient authority to Prometheus or the person, and seals an exact order only when an active Future grant and temporal eligibility are present. Mercury transports and records that order; Future does not issue Go, invoke tools, interpret Apollo, or turn recommendations into authorization. A later Future implementation may consume the structured Apollo result and determine proposed child work, but that is outside Kernel 16.

The frontier model is an optional reasoning escalation. It receives an intentionally curated brief, never an automatic Vault dump.

## Past retrieval contract

Past searches only the configured Vault and never writes it. Its normal response contains two to five jumps, unless fewer credible jumps exist. Each jump includes source, location, excerpt, match type, confidence, age, and correction status. Older evidence is not discarded merely because it is old; recency adjusts ranking, while explicit correction and supersession govern validity.
