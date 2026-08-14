# Kernel 1 Reportback

**Status:** active

**Scope:** the first documented Pantheon kernel increment.

**Successor rule:** Every new Kernel reportback starts from the immediately preceding active reportback. It retains sections that remain useful, updates claims with evidence, adds new sections only when needed, and explicitly marks removed sections as expired. A later reportback must link to the reportback it supersedes.

## 1. Identity and mandate

Pantheon is the local-first executive system. It protects durable continuity and decides what component may act. It is not a model or an agent.

## 2. Current kernel surface

- Per-user runtime configuration is loaded from `~/pantheon-runtime/config/pantheon.json`.
- Local and frontier model names may be declared independently.
- Capability configuration describes available organs and their risk levels.
- The `past` command performs bounded, read-only Vault search.
- Normal local reasoning and explicit Hermes delegation remain separate paths.

## 3. Evidence and reportable state

Past currently returns cited text matches. The intended next retrieval contract is two to five ranked jumps with match confidence, source, age, and correction status. This is an implementation gap, not an assertion that the capability already exists.

## 4. Information boundaries

Local reasoning may be granted full-Vault access through an explicit local capability. Frontier reasoning must receive only a curated brief and question. No model receives authority merely by receiving context.

## 5. Risks and unresolved work

- Past is command-invoked rather than a resident event-driven service.
- Retrieval ranking, correction-aware filtering, and confidence calibration are not yet implemented.
- Frontier routing and curated-brief construction are not yet implemented.
- The policy harness is still incomplete.

## 6. Expiry register

Sections may expire when their statement is no longer an accurate description, their responsibility has moved, or a successor has consolidated their content.

To expire a section, replace its status with `expired`, record the date and reason, and link the successor section or reportback. Do not silently delete an expired consequential claim.

## 7. Next reportback instructions

The Kernel 2 reportback must use this reportback as its template and state whether each numbered section is retained, revised, superseded, or expired. It must distinguish implemented behavior from design intent, cite relevant source files or audit evidence, and name the next smallest safe kernel increment.
