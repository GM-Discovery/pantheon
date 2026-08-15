# Pantheon Roadmap

**Status:** Draft 0.1  
**Governing authority:** Constitution of Pantheon v0.2  
**Supporting contract:** Archimedes Frontier Contract v0.2  
**Construction horizon:** Kernels 8–22

---

# 1. Purpose

This roadmap describes the intended construction sequence from Pantheon's existing local daemon, capability, Apollo, and Weave foundations toward a mature executive harness.

It is not a declaration that future functionality already exists.

Each Kernel is a bounded implementation increment.

Each completed Kernel must produce evidence and a human-readable reportback before its successor is treated as standing on implemented behavior.

The roadmap may change.

The Constitution governs changes to the roadmap.

---

# 2. Mature Pantheon

Pantheon is mature when the following loop can operate proactively without requiring the person to continually manage the machinery:

```text
                         PERSON
                           ↕
                       PANTHEON
                 intake / return / harness
                           │
                      PRESENT
                           │
                     factual missive
                           │
                       MERCURY
                 lawful processing order
                           │
                         APOLLO
                    ↙             ↘
                  PAST           WEAVE
                    ↘             ↙
                   local understanding
                           │
                    Apollo report
                           │
                        FUTURE
                 state + explicit orders
                           │
                       MERCURY
            ┌──────────────┴──────────────┐
            │                             │
     presently authorized          needs ruling
            │                             │
           seal                         PERSON
            │
            ↓
       PROMETHEUS
      controlled slack
            │
            ↓
       agent / tool
            │
            ↓
         result
            │
         MERCURY
            │
         PANTHEON
            │
 BRIEF / QUEUE / NOTIFY / ALARM
```

Father Time provides chronology, identity, expiry, recurrence, and retention throughout the loop.

Archimedes extends reasoning beyond the local boundary only through its separate privacy contract.

Zeus cleans up after everyone else.

---

# 3. Construction Rules

Every Kernel from Kernel 8 onward must answer:

## 3.1 Configuration

What configuration does this capability introduce?

Future Kernels shall extend the shared configuration architecture established in Kernel 8 rather than creating unrelated configuration mechanisms.

## 3.2 Prometheus

How can the capability be stopped, restrained, inspected, or recovered?

A capability that cannot yet receive its mature individual Prometheus chain must at minimum remain subordinate to Protocol Zero.

## 3.3 Evidence

What evidence proves the capability did what it claims?

## 3.4 Failure

What happens when the capability fails?

Failure shall prefer preservation and human inspection over destructive guessing.

## 3.5 Inspection

What status must eventually appear through Panopticon.tower?

Every organ must expose meaningful state.

## 3.6 Reportback

Every Kernel ends in a reportback distinguishing:

- implemented;
- verified;
- configured;
- designed;
- intended;
- unresolved.

These terms are not interchangeable.

---

# 4. Durable-State Doctrine

Pantheon shall preferentially preserve:

**meaningful source → useful result → relationship between them**

It need not permanently preserve every intermediate transformation.

Examples:

- human-readable transcript: durable;
- cited conclusion: durable where useful;
- Weave relationships: durable derived continuity;
- temporary transcription intermediates: temporary;
- frontier scrub packets: temporary;
- frontier identity translation maps: ephemeral.

The result should remain capable of leading back to meaningful source material without requiring an archaeological dump of every computation that occurred between them.

---

# Phase I — Establish the Rails

---

# Kernel 8 — Configuration Foundation and Prometheus Protocol Zero

## Objective

Create the configuration and human-control architecture that every later Kernel must inherit.

## What becomes true

Pantheon has:

- an explicit configuration schema version;
- coherent configuration namespaces;
- defaults;
- validation;
- explicit migration;
- resolved-configuration inspection;
- configuration documentation;
- a clean distinction between configuration and authority;
- and Prometheus Protocol Zero.

The canonical emergency command becomes:

`pantheon prometheus protocol --pause`

It requires no reasoning model, Mercury order, or normal capability grant.

## Configuration shape reserved

Kernel 8 establishes room for at least:

- runtime;
- paths;
- models;
- Father Time;
- Intake;
- Present;
- Past;
- Weave;
- Future;
- Mercury;
- Prometheus;
- notifications;
- retention;
- work windows;
- Zeus;
- Archimedes;
- Panopticon.tower;
- setup/installation;
- and future organs.

Reserved configuration must be explicitly distinguished from active configuration.

## Primary user surfaces

```text
pantheon show config
pantheon config validate
pantheon config path
pantheon config migrate
pantheon prometheus protocol --status
pantheon prometheus protocol --pause
pantheon prometheus protocol --resume
pantheon prometheus protocol --hard-stop
```

Pantheon entry teaches the emergency mnemonic.

## Explicitly not yet

No Intake processing.

No Father Time chronicle.

No individual Prometheus chains.

No new autonomy.

---

# Kernel 9 — Father Time and Intake Ledger

## Objective

Give Pantheon deterministic answers to:

> What entered?  
> When?  
> Is this the same object?  
> Where is it?  
> How long should the raw source remain?

## What becomes true

Pantheon Intake receives deterministic local inventory.

Each accepted object receives:

- object ID;
- SHA-256 fingerprint;
- first-seen time;
- source path;
- type;
- size where useful;
- retention state;
- retention deadline;
- processing state;
- yearly chronology reference.

Father Time begins the active yearly chronicle.

Prior yearly chronicles are structurally supported.

## Storage

Configuration activates:

- Intake location;
- processed Markdown location;
- raw-retention location;
- default raw retention period.

Default raw retention:

**30 days**

Retention storage may live on another mounted drive.

Missing/full/unmounted retention storage fails safe.

## Behavior

Scanning is initially explicit and testable.

Repeated scans of unchanged objects are idempotent.

No automatic purge occurs.

---

# Kernel 10 — Present and Normalization

## Objective

Turn Intake objects and direct human instructions into factual Present missives.

## What becomes true

Present can:

- observe newly ingested material;
- attach provenance;
- attach Father Time identity;
- normalize supported material;
- create human-readable Markdown;
- preserve uncertainty;
- accept direct orders from the person.

A command may take the conceptual form:

```text
pantheon present "Give me a brief summary and action items from the meeting recording I just uploaded."
```

Present may resolve recent Intake references where deterministic.

Ambiguous references remain unresolved rather than guessed.

## Audio

Audio enters the same Intake architecture.

Local transcription integration begins here or through a narrowly bounded normalizer owned by this Kernel.

Raw audio remains eligible for the default 30-day retention.

The useful human-readable transcript becomes durable source material.

## Failure

Unprocessed input remains recoverable.

Ordinary processing failures default toward BRIEF once the attention system exists.

---

# Phase II — Build the Executive Loop

---

# Kernel 11 — Mercury Orders, Motes, and Seals

## Objective

Give Pantheon a durable operational language for work.

## What becomes true

A real **order** exists.

An order can identify:

- purpose;
- source missive/report;
- requested capability;
- intended recipient;
- scope;
- constraints;
- current authority;
- mote of Chron;
- status;
- result;
- evidence.

Mercury becomes the carrier of orders.

## The seal

Mercury evaluates whether an order is presently authorized.

If authorized:

**seal and carry**

If not presently authorized:

**surface for human ruling**

Existing capability grants become an ancestor/input to this broader authority model rather than an unrelated permission system.

## Initial transport

Mercury can carry bounded local processing orders among Pantheon components.

Full recurring rounds come later.

---

# Kernel 12 — Apollo Deep Local Processing

## Objective

Turn Apollo from a callable local model into Pantheon's principal local reasoning position.

## What becomes true

Mercury can deliver a Present missive to Apollo.

Apollo can aggressively consult authorized local continuity, including:

- Past;
- Weave;
- Vault material;
- relevant local records;
- and other authorized local corpora.

Apollo returns a cited report.

## Reasoning discipline

Apollo is explicitly taught:

1. **how to learn inside Pantheon**, and
2. **how to recognize the useful end of its own reasoning capability**.

Apollo should retrieve rather than bluff.

Apollo should surface uncertainty rather than conceal it.

Apollo may recommend frontier escalation when the available local reasoning engine cannot responsibly finish the problem.

## Boundary

Apollo remains locally confined.

No outside-world access is granted merely because Apollo wants more information.

---

# Kernel 13 — Future and Active State

## Objective

Convert Apollo's understanding into organized work.

## What becomes true

Future can receive Apollo's report and maintain useful active state concerning:

- unresolved obligations;
- waiting conditions;
- dependencies;
- prepared work;
- expected outcomes;
- and actionable next steps.

Future does not continuously regenerate projections unnecessarily.

Apollo remains responsible for substantive prioritization.

Future transforms that reasoning into explicit orders.

Those orders go to Mercury.

---

# Kernel 14 — Brief, Needs-Ruling Queue, and Attention Ladder

## Objective

Give Pantheon a disciplined means of returning useful work to the person.

## What becomes true

Pantheon has:

- a morning/situation brief;
- one unified queue of matters requiring human ruling;
- structured decision items;
- notification escalation;
- dismissal semantics.

The attention ladder becomes:

**SILENT ← BRIEF → QUEUE → NOTIFY → ALARM**

Default:

**BRIEF**

## Human rulings

CLI is sufficient initially.

The person can:

- approve;
- reject;
- modify;
- defer;
- create suitable standing authority.

## Long memory

Dismissal lowers attention.

It does not erase consequential decisions.

Relevant dismissed matters may resurface when circumstances change or a later review becomes meaningful.

Default long-horizon resurfacing may be approximately five years where suitable.

---

# Kernel 15 — Mercury's Rounds, Work Windows, and Retention Lifecycle

## Objective

Make Pantheon genuinely proactive without making every daemon continuously active.

## What becomes true

Mercury makes recurring rounds.

Stops may include:

- pending orders;
- completed agents;
- failed work;
- Needs-Ruling items;
- notification delivery;
- retention work;
- scheduled jobs;
- future Archimedes exchanges.

## Work windows

Pantheon supports configured:

- foreground periods;
- heavy-work periods;
- quiet periods.

Heavy Apollo work, transcription, Weave maintenance, and similar local compute should prefer periods when the machine is not needed by the person.

Notifications remain capable of escalation.

## Retention

Father Time can generate retention-due state.

Future can generate lawful maintenance orders.

Mercury can carry those orders.

Destructive execution still waits for Zeus.

---

# Phase III — Constrain, Clean, and Reach Outward

---

# Kernel 16 — Prometheus, Full Jailer

## Objective

Replace broad trust in running components with inspectable individual chains.

## What becomes true

Prometheus can govern individual agents/processes through controls such as:

- pause;
- resume;
- restart;
- quarantine;
- revoke;
- filesystem boundaries;
- network boundaries;
- tool boundaries;
- resource limits;
- execution ceilings.

Protocol Zero remains directly available to the person.

## Vulture Feed

Prometheus receives machine telemetry such as:

- heartbeats;
- process exits;
- policy refusals;
- capability use;
- unexpected network attempts;
- resource anomalies;
- crashes;
- restart events;
- and audit-integrity problems.

Prometheus watches machinery, not private human prose.

## Chains

Chains may record:

- why they exist;
- what risk they control;
- what evidence could justify loosening them.

---

# Kernel 17 — Zeus, Sanitation

## Objective

Give Pantheon a dedicated executor for lawful deterministic cleanup.

## What becomes true

Zeus can perform sealed sanitation orders involving:

- expired raw audio;
- expired frontier intermediate records;
- temporary conversion files;
- caches;
- logs;
- orphaned work directories;
- abandoned intermediate artifacts;
- other configured safe machine garbage.

**Zeus may dispose of trash. Zeus may never decide what is trash.**

Every consequential deletion returns a receipt.

## Default retention examples

Raw recordings:

**30 days**

Temporary frontier transformation/debug records:

**30 days**

Ephemeral frontier identity maps are not Zeus's 30-day problem; they are destroyed at exchange completion.

---

# Kernel 18 — Archimedes Frontier Exchange

## Objective

Implement the Archimedes Frontier Contract.

## What becomes true

Apollo can determine that local reasoning is insufficient and prepare a frontier missive.

### Apollo

Apollo:

- minimizes context;
- semantically removes identifying information;
- preserves relevant relationships and roles;
- creates random exchange-scoped frontier identities;
- creates an ephemeral local translation map.

### Privacy gate

A deterministic second pass checks obvious leaks.

**Zero findings:** pass.

**One finding:** replace offending span with `<REDACTED>`, report the privacy incident to Mercury, escalate to the person, and permit the sanitized packet to proceed if otherwise safe.

**Two or more findings:** do not send; return to Apollo for correction; inspect again.

### Archimedes

Archimedes drives one complete frontier exchange.

Temporary IDs remain stable across clarification turns within that exchange only.

Archimedes receives no identity map and no direct Vault access.

### Mercury

Mercury checks Archimedes during normal rounds.

Mercury carries:

- outbound frontier orders;
- clarification material;
- completed results;
- failures;
- privacy incidents.

### Return

The frontier response is rehydrated locally.

The ephemeral identity map is then destroyed.

### Retention

Durable:

- meaningful source;
- human-readable transcripts;
- useful rehydrated results;
- cited relationships/results.

Thirty days:

- sanitized outbound missives;
- sanitized raw frontier responses;
- debugging/transformation evidence.

Ephemeral:

- translation map.

---

# Kernel 19 — Self.Change

## Objective

Make Pantheon capable of participating in its own improvement without authorizing its own sovereignty.

## What becomes true

The mature chain becomes executable:

```text
1. Pantheon detects a miss.
2. Past consults Weave.
3. Present forms the missive.
4. Mercury carries the reasoning order.
5. Apollo investigates.
6. Future prepares proposed change orders.
7. Mercury surfaces the ruling.

8. THE PERSON RULES.

9. Mercury seals the resulting authorized order.
10. Prometheus grants the selected builder controlled slack.
11. Builder changes code/configuration.
12. Tests run.
13. Diff/evidence/rollback material is produced.
14. Prometheus tightens the chains.
15. Result returns through Pantheon.
```

The central placement of the person is deliberate.

Detection is automated.

Investigation is automated.

Proposal may be automated.

**The call belongs to the person.**

Constitutional amendment remains outside automatic self-ratification.

---

# Kernel 20 — Executive Reference Organs

## Objective

Prove that the entire Pantheon loop works on ordinary real-world executive work.

## Reference domains

At minimum:

- email;
- calendar;
- notifications.

Existing Gmail work should be reconciled into the mature Mercury/order model rather than maintained as a parallel architecture.

## Example mature behavior

Pantheon observes:

- a meeting tomorrow;
- a related unanswered email;
- relevant prior continuity.

Apollo understands:

- who the participants are;
- what occurred previously;
- what remains unresolved.

Future prepares:

- meeting context;
- action items;
- an email draft;
- any appropriate follow-up orders.

Mercury:

- seals routine work covered by standing authority;
- carries it;
- puts consequential outward actions lacking authority into Needs-Ruling.

Morning brief says, in substance:

> Here is who you are meeting.  
> Here is what matters.  
> The response is drafted.  
> Review/send when ready.

## Success condition

The person receives meaningful executive benefit without needing to manually orchestrate the gods.

---

# Phase IV — Look Down, Then Give It Away

---

# Kernel 21 — Panopticon.tower

## Objective

Give the person a coherent terminal-first view over Pantheon.

## What becomes true

The person can inspect:

```text
pantheon show
pantheon show config
pantheon show gods
pantheon show orders
pantheon show mercury
pantheon show prometheus
pantheon show father-time
pantheon show intake
pantheon show brief
pantheon show grants
pantheon show audit
```

The exact command grammar may differ, but the concepts must remain accessible.

## Terminal surface

Plain `pantheon` may eventually present a compact terminal view of:

- organ status;
- current work;
- current order;
- runtime duration;
- resource use;
- chain state;
- queue state;
- Intake state;
- next Mercury round;
- next heavy-work window;
- failures requiring attention.

## Prometheus controls

The person may invoke individual operational controls through Panopticon.tower.

Harmless diagnostics may support metaphorical commands such as a **buzz**.

A **zap** must map to deterministic operational behavior rather than random punishment.

The gods are observable from above.

---

# Kernel 22 — Progressive Installation, Packaging, Setup, and Doctor

## Objective

Turn the working Pantheon runtime into something another person can safely install and progressively bring to life on a new supported Linux system.

## One-line beginning

A future release should permit a new supported Linux machine to begin with one installation command.

Immediately after installation:

**Prometheus Protocol Zero becomes available.**

The person's emergency control mechanism is taught before sophisticated automation.

## Progressive discovery

Pantheon inspects the machine for available resources such as:

- systemd/user-service support;
- Ollama or compatible local model runtimes;
- installed models;
- compatible local agents;
- CPU/GPU/RAM;
- storage;
- mounted additional drives;
- Whisper/transcription capability;
- Git;
- Obsidian Vault candidates;
- notification facilities;
- supported frontier-provider configuration.

Discovery is read-only.

Discovery does not equal attachment.

## Guided setup

Setup proceeds as understandable steps:

```text
Step 1/?
Checking this system...
```

Pantheon explains requirements in terms of roles.

Example:

```text
Apollo requires a strong local reasoning engine.

Found:
  1. Model A
  2. Model B

Recommended:
  Model A

Use it for Apollo? [Y/n]
```

Or:

```text
Mercury benefits from a smaller, fast local model.

No separate candidate was found.

Pantheon can use Apollo's model temporarily,
but this means Mercury and Apollo are not
operationally independent.

Proceed? [Y/n]
```

## Structural weaknesses

Pantheon may run imperfectly when safe.

It must tell the person what is weak rather than pretending the configuration is ideal.

A single model may temporarily inhabit:

- Mercury;
- Apollo;
- Archimedes;

if technically possible.

Pantheon should surface this as structural coupling.

## Frontier setup

Archimedes provider selection is terminal-based.

No frontier provider is a valid configuration.

Pantheon remains operational with frontier escalation unavailable.

## Partial installations

Pantheon should begin providing useful functionality as soon as a safe viable subset exists.

A setup may finish temporarily with:

```text
Available:
✓ Configuration
✓ Prometheus
✓ Intake
✓ Past
✓ Apollo

Unavailable:
! Dedicated Mercury reasoning resource
! Archimedes

Run:
pantheon setup continue
```

Missing capabilities remain visible.

Setup progress survives reboot and may continue later.

## Packaging

Kernel 22 must establish:

- installation layout;
- per-user configuration layout;
- state layout;
- systemd user service;
- upgrades;
- configuration migrations;
- backup/restore;
- uninstall;
- version reporting;
- dependency handling;
- preservation of Vault/durable continuity.

Removing Pantheon must not silently remove the person's Vault or durable human work.

## Doctor

`pantheon doctor` should become a high-quality diagnostic surface reporting:

- daemon/service health;
- configuration health;
- storage;
- models;
- organs;
- chains;
- Intake;
- Weave;
- optional missing components;
- structural weaknesses;
- actionable repair paths.

---

# 5. Kernel Reportback Discipline

Every completed Kernel shall produce a reportback containing:

1. constitutional status;
2. predecessor actually used;
3. inherited claims retained/revised/superseded/expired;
4. implemented surface;
5. configuration changes;
6. authority and Prometheus boundary;
7. state/evidence produced;
8. verification commands and actual results;
9. files changed;
10. risks and unresolved work;
11. expiry register;
12. next smallest safe successor.

Reportbacks shall not treat roadmap intent as evidence.

---

# 6. Configuration Accretion Rule

Kernel 8 creates the configuration architecture.

Every later Kernel activates only its own necessary configuration.

Examples:

**Kernel 9**
- Intake paths
- retention path
- raw retention period
- Father Time chronology

**Kernel 10**
- normalization/transcription behavior

**Kernel 11**
- Mercury/order policy

**Kernel 14**
- attention defaults
- resurfacing periods

**Kernel 15**
- rounds
- heavy-work windows
- quiet periods

**Kernel 16**
- Prometheus chains/resource boundaries

**Kernel 17**
- sanitation policies

**Kernel 18**
- frontier provider/Archimedes settings
- temporary frontier retention

**Kernel 21**
- Panopticon display behavior

**Kernel 22**
- setup/discovery/package state

A Kernel shall not silently activate previously reserved configuration.

---

# 7. Prometheus Accretion Rule

Protocol Zero exists before growing autonomy.

Every later Kernel must explicitly state:

- whether it performs work while Pantheon is paused;
- how global pause affects it;
- what individual chain it will eventually receive;
- what state remains inspectable when stopped;
- how incomplete work is recovered.

By Kernel 16, these scattered requirements consolidate into the full Prometheus jailer.

---

# 8. Definition of the First Mature Pantheon

After Kernel 22, Pantheon should be able to satisfy this description:

> A person can install Pantheon on a supported Linux machine, progressively attach suitable local and optional frontier intelligence, place ordinary life/work material into Intake, and largely stop managing the AI itself.
>
> Pantheon observes through Present, remembers through Father Time and Weave, learns through Past, reasons deeply through Apollo, organizes work through Future, carries lawful orders through Mercury, constrains execution through Prometheus, reaches frontier reasoning safely through Archimedes, disposes of lawful garbage through Zeus, and returns useful work through briefs, rulings, notifications, and Panopticon.tower.
>
> The person's meaningful sources and useful results endure.
>
> Replaceable machinery may come and go.
>
> The person remains at the center.

Everything after this point is principally:

- improving intelligence;
- loosening justified chains;
- adding organs;
- adding integrations;
- improving installation/support;
- and reducing the amount of attention Pantheon itself requires.