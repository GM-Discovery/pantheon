# Kernel 6 Reportback — Apollo Vault Connection Spike

**Status:** active, one-shot read-only test completed in this increment.

## 1. Purpose

Test the first working connective path across the existing Vault:

```text
Vault text → deterministic local evidence extraction → Pantheon policy gate
→ local Apollo proposal → separate runtime report
```

This is retrieval and connection analysis, not model training or fine-tuning.

## 2. Implemented surface

- `POST /v1/vault/connections` is a localhost-only Pantheon endpoint.
- `POST /v1/grants` requires the local CLI control token stored at `state/control.token` (mode `0600`). Raw HTTP callers cannot mint grants. The current CLI command is `pantheon grant <capability> <scope> <intent-summary>`; a GUI approval surface can replace it later.
- Apollo accepts only approved local HTTP hosts: loopback, `localhost`, and the explicit local Docker-host bridge `172.17.0.1`.
- It reads Markdown notes only, skips `.obsidian` plugin/configuration data, and skips all symlinks rather than following them outside the Vault.
- It deterministically counts lowercase words of five or more characters, excluding a small list of common connective words.
- It retains up to two cited source lines for each of the top 12 terms.
- Apollo (`mistral-small3.1:latest` through local Ollama) receives only the bounded cited evidence and returns an explicitly tentative proposal.
- The endpoint writes a report beneath Pantheon runtime state, not into the Obsidian Vault:
  `state/reports/vault-connections-<timestamp>.md`.

## 3. Authority and boundaries

Allowed:

```text
read authorized local Vault Markdown
call local Apollo/Ollama
write separate runtime report and audit record
```

Prohibited:

```text
Vault edits, tags, links, graph mutations, email operations, external-model
calls, Smart Connections changes, training/fine-tuning, scheduling, or
background crawling
```

Past's existing cited Vault search remains separate and unchanged. Future and Present remain conceptual; this spike does not impersonate either role.

## 4. Smart Connections observation

Smart Connections v4.5.3 is installed. Its configured data file contains only installation/version metadata, so no plugin-created semantic index was found in the inspected plugin directory. This spike neither reads nor changes the plugin's private data.

## 5. Interpretation limits

Term frequency is evidence about repeated text, not proof of importance, intent, sentiment, or a meaningful relationship. Apollo output is a proposal. Every deterministic item in the report carries a path and line number for inspection.

## 6. Next safe successor

Review the generated report. If it is useful, design a persistent incremental index with note fingerprints, passage-level nodes, explicit edge types, correction/strike support, and a human-readable graph export. Do not schedule crawls or write links/tags until that design is separately reviewed and approved.
