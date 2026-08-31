# Pantheon

> **A person is not a user account.**

```text
V(person) ≈ (A × R × C) / P

A = Agency
R = Relational continuity
C = Capacity for correction
P = Replaceability

As P → 0, value → ∞.
```

Pantheon is a **local-first, model-agnostic personal intelligence runtime**.

It is not a chatbot.

It is not an agent wearing a clever system prompt.

It is an attempt to build something more durable: a personal computing system in which models, agents, services, and tools are replaceable organs operating inside a persistent architecture of memory, permissions, values, workflows, and human control.

The model is not Pantheon.

The agent is not Pantheon.

**Pantheon is the system.**

## License

Pantheon's own source code is licensed under the GNU Affero General Public
License, version 3 (AGPLv3). See [LICENSE](LICENSE) and [NOTICE](NOTICE).
Third-party dependencies, model weights, and system tools retain their own
licenses; see [the licensing inventory](docs/licensing.md).

For implemented current state, use the [active architecture brief](docs/pantheon-architecture-brief.md),
the [current system inventory](docs/current/system-inventory.md), and the
[GUI readiness contract](docs/gui/GUI-CONTRACT.md). This README remains the
project overview; kernel reportbacks and older examples are historical context
when they differ from those canonical documents.

---

## Why Pantheon?

Most AI assistants put the intelligence, identity, memory, and authority in roughly the same place: somebody else's model running on somebody else's infrastructure.

That creates a fragile relationship.

Change the model and the assistant changes.

Lose the service and the assistant disappears.

Give the agent tools and the same entity doing the reasoning may also hold the keys.

Pantheon takes the opposite approach.

```text
Human
  │
  ▼
Pantheon Runtime
  │
  ├── Memory
  ├── Values
  ├── Current state
  ├── Policy
  ├── Capability routing
  └── Audit
          │
          ▼
     Policy Harness
          │
     ┌────┼─────────┐
     ▼    ▼         ▼
  Models Agents   Tools
     │    │         │
     └────┴────┬────┘
               ▼
             Result
```

Models reason.

Agents act.

Tools perform defined operations.

**Pantheon decides what belongs where and what each component is allowed to do.**

But also, it's great metaphor. Assemble your own Pantheon of AIs, very cyberchic, very digital savvy.

---

## The Operating Principle

> **Agents propose. Pantheon authorizes. The harness enforces.**

A model may decide that checking a calendar would help.

That does not mean the model receives permission to modify the calendar.

An agent may be extraordinarily capable.

That does not make the agent sovereign.

Pantheon is designed around explicit capabilities such as:

```text
calendar.read
calendar.write

email.read
email.draft
email.send

vault.read
vault.write

notify.send

transcribe

shell.safe
shell.admin
```

A task receives the capabilities it requires—not whatever powers happen to exist on the machine.

That distinction becomes increasingly important as AI agents become more capable.

---

## Local First, Not Local Only

Pantheon prefers local execution when local execution is sufficient.

A local model can classify a transcript, search personal context, extract tasks, summarize documents, or perform routine reasoning without shipping a person's life to a remote inference provider.

But Pantheon does not confuse locality with intelligence.

If a task exceeds the capability of the available local model, the architecture can route it to a stronger model.

The goal is not:

> Make one model do everything.

The goal is:

> **Use the smallest sufficient intelligence for each task.**

Today that may mean a local 24B model for routine cognition and a frontier model for exceptional reasoning.

Tomorrow those models can change.

Pantheon should not have to.

---

## Current Stack

Pantheon is under active development.

### Runtime

**Go**

The Pantheon Runtime is the executive locus: the place where initial requests enter, capabilities are discovered, policy is applied, and work is routed.

### Local reasoning

**Ollama**

Current local reasoning engine:

```text
mistral-small3.1:latest
24B
Q4_K_M
```

Models are explicitly treated as replaceable components.

### Memory

**Obsidian / Markdown**

Pantheon's durable knowledge, procedures, project context, and governing documents remain human-readable.

The machine should not be the only thing capable of understanding its own memory.

### Execution

**Hermes Agent**

Hermes serves as a cross-domain operator capable of using tools, skills, schedules, email, calendars, and other services.

Hermes is part of Pantheon.

Hermes is not Pantheon.

### Hearing

**whisper.cpp + Whisper large-v3**

Local multilingual speech transcription, intended to receive audio from devices such as Omi and turn ambient speech into structured information Pantheon can process.

### Notifications

**ntfy**

Pantheon can send authenticated push notifications from the local machine to a phone.

The bell works.

---

## What It Can Do Today

The Runtime already supports several real paths.

### Local reasoning

```bash
pantheon "What requires my attention?"
```

Pantheon loads its governing context and sends the problem to the configured local reasoning engine.

### Runtime status

```bash
pantheon status
```

Runtime facts come from deterministic configuration rather than asking an LLM to guess what system it is running inside.

### Explicit delegation

```bash
pantheon hermes "Perform this delegated task."
```

Pantheon can invoke Hermes and receive the result.

### Notifications

```bash
pantheon notify "Your Situation Brief is ready."
```

Pantheon can send an authenticated push notification to a phone.

### Capability routing

```bash
pantheon route "Check my email and calendar for tomorrow."
```

Example result:

```text
Capability: hermes
Confidence: 1
Reason: user requested checking of their emails and calendar.
```

### Policy-gated execution

```bash
pantheon do "Dinner is ready."
```

Pantheon can autonomously route and execute low-risk capabilities while refusing automatic execution of capabilities that have not yet been authorized.

Example:

```text
Pantheon routed to: hermes
Hermes selected, but automatic Hermes execution is not yet authorized.
```

This refusal is a feature.

---

## The Pantheon

The names are deliberate.

Pantheon is not intended to become one enormous artificial personality.

Different components have different domains.

### Hermes

Movement between systems.

Tools.

Scheduling.

Delivery.

Execution.

The messenger can be extraordinarily capable without becoming king.

### Whisper

Hearing.

Audio becomes text.

No authority is implied by perception.

### Models

Reasoning engines.

Mistral today.

Something else tomorrow.

A useful mind inside Pantheon is still an organ—not the organism.

More domains will emerge as the architecture requires them.

---

## Ambient Computing

One long-term path looks like this:

```text
Omi / microphone
      │
      ▼
  whisper.cpp
      │
      ▼
 raw transcript
      │
      ▼
   Pantheon
      │
      ├── preserve
      ├── normalize names
      ├── identify people
      ├── extract commitments
      ├── detect tasks
      ├── recognize projects
      └── decide what matters
               │
       ┌───────┼─────────┐
       ▼       ▼         ▼
    Obsidian  Hermes    ntfy
              │
         ┌────┴─────┐
         ▼          ▼
       email     calendar
```

The objective is not an AI that constantly talks.

The objective is a system that quietly converts information into useful state and returns human attention only when human attention is actually valuable.

---

## Memory Without Surrender

Pantheon separates several things that are often collapsed into one opaque AI service:

**Memory** belongs in durable data.

**Identity** belongs in continuity and governing rules.

**Authority** belongs in explicit policy.

**Reasoning** belongs in replaceable models.

**Execution** belongs in constrained tools and agents.

This makes it possible to replace a model without losing Pantheon.

Replace an agent without losing Pantheon.

Change a cloud provider without losing Pantheon.

Run offline without losing Pantheon.

The continuity survives because the continuity was never entrusted to the model in the first place.

---

## Human Sovereignty

Pantheon is built around a simple asymmetry:

**Software is replaceable. People are not.**

That affects architecture.

A human should be able to:

* inspect what Pantheon knows;
* correct what Pantheon believes;
* override Pantheon's decisions;
* revoke permissions;
* replace models;
* replace agents;
* inspect consequential actions;
* preserve personal continuity across technical migrations;
* leave.

An assistant that maximizes its own persistence has misunderstood the assignment.

---

## Governing Objective

Pantheon's objective is intentionally larger than automation:

> **Deliver as much genuinely free human time as possible without violating the system's governing values.**

Automation is useful only when it returns agency.

The desired outcome is not more time spent operating Pantheon.

It is less time spent operating everything else.

---

## Architecture Direction

Pantheon is moving toward:

```text
Input
  │
  ▼
Pantheon Runtime
  │
  ├── deterministic command?
  │       └── execute known capability
  │
  └── ambiguous / compound?
          │
          ▼
      reasoning model
          │
          ▼
    proposed route/actions
          │
          ▼
     policy harness
          │
     allowed?
      │     │
     yes    no
      │     │
      ▼     ▼
   execute  ask/refuse
      │
      ▼
 audit + state + result
```

The reasoning layer can remain probabilistic.

The authority layer does not have to be.

---

## Repository Status

Pantheon is **experimental and incomplete**.

The Runtime works.

The architecture is becoming real.

The safety harness is still being built.

Do not mistake a working prototype for a finished autonomous system.

Current development priorities include:

1. Capability-scoped agent execution
2. Persistent runtime state and audit history
3. Local knowledge retrieval
4. Whisper ingestion
5. Omi audio integration
6. Model routing and escalation
7. Email/calendar capability separation
8. Background and event-driven processing
9. A human-facing Pantheon interface
10. Making the system increasingly independent of any single AI provider

---

## Philosophy

Pantheon does not assume that powerful institutions are trustworthy.

It also does not assume that open-source software is virtuous merely because its source is visible.

The design goal is therefore not to identify a perfectly trustworthy model vendor, cloud provider, agent developer, or government.

The design goal is to **require less trust**.

Run locally where practical.

Expose the minimum necessary data.

Grant the minimum necessary capability.

Keep humans capable of correction.

Preserve an audit trail.

Make important components replaceable.

And never confuse intelligence with authority.

---

## The North Star

Before the first `package`.

Before the first import.

Before the machinery:

```text
V(person) ≈ (A × R × C) / P

A = Agency
R = Relational continuity
C = Capacity for correction
P = Replaceability

As P → 0, value → ∞.

A person is not a user account.
```

Everything after that is implementation.
