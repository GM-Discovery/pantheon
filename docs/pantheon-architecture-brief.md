---
type: active-architecture-brief
status: active
updated: 2026-08-17
---

# Pantheon Architecture Brief

This is the concise active brief for Pantheon’s current organism. Kernel
reportbacks remain historical implementation evidence; the roadmap governs
direction; this brief is the quickest way to understand who does what.

## Purpose

Pantheon exists to return meaningful free time and attention to Grant while
preserving truth, human sovereignty, accountability, inspectability, and a
direct ability to stop or revoke the system.

The governing distinction is:

> Models reason. Deterministic machinery packages, authorizes, seals, and
> controls. Tools perform bounded operations.

No persuasive model output is a grant, seal, Go command, or human approval.

## Canonical organism

```text
audio arrival
        ↓
      Rumor
        ↓
transcript occurrence / direct request
        ↓
      Intake
        ↓
      Present
        ↓
       Past
        ↓
      Apollo
        ↓
      Seshat
        ↓
 Janus when time matters
        ↓
      Future
        ↓
 authority resolution + deterministic seal
        ↓
      Mercury
        ↓
     Bifrost
        ↓
   external capability / tool / plugin / organ
        ↓
 result, evidence, and durable state
```

Intake is the ingestion process that supplies occurrences to Present; it is
not a competing reasoning deity. Past supplies bounded continuity to Apollo.
Apollo supplies semantic understanding and proposed work. Seshat turns that
meaning into a declared package shape. Janus supplies temporal eligibility
when a package has a temporal contract. Future is the deterministic authority
sign-off: it resolves existing authority and permits the exact operation to
be sealed. Mercury transports the sealed order, issues Go only at the lawful
execution boundary, invokes the bounded adapter, and records the result.
Bifrost governs crossings to external software and services; it does not
decide that work should occur.

Not every harmless operation needs every stage. A proof may legitimately skip
Past, Apollo, or Janus when the operation has no continuity, reasoning, or
temporal condition. The architecture must never skip a stage merely to hide an
authority or provenance decision.

## Responsibility map

| Component | Type | May do | May not do |
| --- | --- | --- | --- |
| Pantheon kernel | deterministic daemon | configuration, orchestration, policy, audit, control surfaces | replace human authority or become a reasoning engine |
| Intake | deterministic process | receive and identity-track occurrences | decide semantic importance |
| Rumor | deterministic local audio process | stable-file detection, source hashing, local Whisper transcription, durable transcript/metadata, explicit Intake handoff | interpret, summarize, prioritize, authorize, delete raw audio, or call Apollo |
| Present | deterministic daemon | faithful source normalization, provenance, retention boundaries | invent continuity or decide what matters |
| Past | bounded retrieval role | return cited local continuity from Vault/Weave | rewrite sources or authorize work |
| Apollo | local reasoning deity | analyze, connect evidence, explain, recommend, cite | grant authority, seal, issue Go, invoke tools |
| Seshat | deterministic registry/packager | subject codes, package contracts, fields, adapter identity, temporal metadata | reason, grant, seal, or execute |
| Father Time | chronology service | time references, chronicles, temporal evidence | decide importance or authority |
| Janus | deterministic temporal daemon | apply Seshat-declared threshold policies and durable transitions | invent policy, schedule recurring work, grant, seal, or execute |
| Future | deterministic authority daemon | consume proposed package work, check grants and Janus eligibility, refer insufficient authority, authorize exact sealing | perform semantic planning, manufacture authority, issue Go, or invoke tools |
| Prometheus | control/authority boundary | pause, resume, inspect, escalate, approve or revoke within granted bounds | reason about source meaning or silently replay skipped work |
| Mercury | transport/execution daemon | validate seals and authority at execution, issue Go, invoke bounded adapters, record results | invent permission or decide semantic importance |
| Bifrost | governed external-capability bridge | register, describe, approve, enable, invoke, inspect, disable, and audit bounded external adapters | invent work, bypass Prometheus, treat installation or presence as permission, or become a universal compatibility layer |
| Mail workflow | deterministic bounded integration | account for new Gmail mail, advance a manual backlog, preserve Intake provenance, expose Apollo understanding, and create governed action proposals | send, delete, archive, create filters, download attachments, or turn Apollo suggestions into authority |

## Control and safety

Prometheus remains directly reachable without Apollo, Future, Mercury, or a
frontier model. The canonical emergency command is:

```text
pantheon prometheus protocol --pause
```

Pause blocks autonomous effects and invalidates outstanding execution where
required. Resume does not mean “execute everything missed”; deterministic
state and temporal policy decide what can be reconsidered. Every consequential
operation must remain inspectable through authority, order, seal, Go, attempt,
and result evidence.

## Current implementation boundary

Kernels 13–16 established the authority, Seshat, Janus, and Future foundations
individually. Kernel 17 integrated the existing stages and usability surface.
Kernel 18 adds Rumor as a bounded local audio-to-evidence boundary. Kernel 19
adds Bifrost as the governed crossing for external capabilities and its first
read-only Himalaya/Gmail mail-intake adapter. Kernel 22 adds manual mail
understanding and governed action proposals while preserving a separate
read-only capability identity. It does not add autonomous software
acquisition or outbound mail authority.

Future currently administers authority deterministically. A later active-state
engine may organize longer-lived obligations and dependencies, but that does
not turn Future into a second Apollo.

## Concept horizon

- **Rumor** — Kernel 18 deterministic local Whisper orchestration. Rumor makes
  audio into durable, provenance-bearing transcript evidence and hands it to
  Intake. Rumor does not interpret, summarize, prioritize, or authorize.
- **Zodiac** — later recurring-schedule owner. Father Time answers what time it
  is; Janus applies thresholds; Zodiac owns recurring declarations and due
  instances.
- **Archimedes** — later frontier-reasoning boundary with curated,
  privacy-minimized outbound context. It is not Apollo and receives no direct
  authority or unrestricted Vault access.
- **Zeus** — later deterministic sanitation owner for explicitly identified
  machine garbage. Zeus does not decide that human material is garbage.
- **Nostalgia** — later shallow resurfacing candidate. Its suggestions return
  through normal Intake/Present/Past/Apollo evaluation and never become
  importance decisions or authority.
- **Aphrodite** — post-core software-construction intelligence. Aphrodite may
  eventually construct isolated candidate changes, tests, and binaries, but
  cannot replace the running Pantheon merely because she wrote code or passed
  tests. Promotion remains outside the proposing intelligence.
- **Bifrost** — the governed bridge for external software, services, protocols,
  plugins, APIs, and models. Presence is not permission; research is not
  permission; download, installation, registration, and use remain distinct
  inspectable events.

These concepts are named responsibility boundaries, not permission to begin
their implementation automatically. Kernel 22 adds no scheduler; Zodiac, the
recurring-schedule owner, remains deferred. Aphrodite, Archimedes, and Zeus
remain deferred as well.

## Documentation entry points

- [Pantheon Roadmap](Pantheon%20Roadmap.md) — active direction and sequencing.
- [Software](software.md) — software boundaries and information flow.
- [Configuration](configuration.md) — runtime configuration and Prometheus.
- [Future authority](future-authority.md) — current Future behavior.
- [Seshat plugin extension](seshat-plugin-extension.md) — package contracts.
- [Janus temporal behavior](janus-temporal.md) — temporal transitions.
- [Kernel 19 reportback](reportbacks/kernel-19-bifrost-capability-bridge.md) — Bifrost registry and bounded mail crossing.
- [Kernel 22 reportback](reportbacks/kernel-22-mail-intelligence.md) — manual mail intelligence and governed action proposals.
- [Current system inventory](current/system-inventory.md) — implemented components and truth locations.
- [GUI readiness contract](gui/GUI-CONTRACT.md) — stable Kernel 21 read/control contract.
- [Reportbacks](reportbacks/) — historical implementation and verification.
