# Software

## Kernel

The Pantheon kernel is the small, deterministic executive layer. It loads configuration, discovers available capabilities, applies policy, routes work, records audit state, and returns results. It is responsible for authorization, not for pretending to be the most intelligent component.

## Configuration

Configuration is local per user at `~/pantheon-runtime/config/pantheon.json`. It may declare an Ollama endpoint, local model, optional frontier model, Vault location, and governing context. Empty fields are valid; naming a model does not create credentials or grant it authority.

## Information flow

```text
speaker -> Present -> Past -> Future -> Pantheon
                                      |
                         policy + bounded capability
                                      |
                            Apollo / Hermes / tool
```

Apollo is the local reasoning deity. It can be given a full-Vault capability when the task warrants it and the local boundary is intact.

Hermes moves approved messages and actions between nodes. Hermes does not establish policy.

The frontier model is an optional reasoning escalation. It receives an intentionally curated brief, never an automatic Vault dump.

## Past retrieval contract

Past searches only the configured Vault and never writes it. Its normal response contains two to five jumps, unless fewer credible jumps exist. Each jump includes source, location, excerpt, match type, confidence, age, and correction status. Older evidence is not discarded merely because it is old; recency adjusts ranking, while explicit correction and supersession govern validity.
