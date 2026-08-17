---
type: governing-document
status: active
owner: Grant A. Murray
updated: 2026-08-16
---

# Pantheon Roadmap

**Author:** Grant A. Murray

## 1. Purpose of this document

This is Pantheon’s active planning roadmap.

It exists to answer:

1. What Pantheon is for.
2. What principles govern it.
3. What the major organs of Pantheon do.
4. What has actually been built.
5. What the next committed kernels are.
6. What important capabilities remain provisional or post-core.
7. When Pantheon should stop expanding and start earning its maintenance cost.

Kernel specifications define individual increments.

Kernel reportbacks record implementation evidence.

Repository/runtime behavior takes precedence over roadmap claims about what currently exists.

This roadmap governs direction. It does not prove implementation.

The concise active architecture map is [Pantheon Architecture Brief](pantheon-architecture-brief.md).

---

## 2. Update rules

Update this roadmap after any of the following:

- A roadmap milestone is completed.
- Pantheon’s architecture materially changes.
- A deity or daemon role is added, removed, or reassigned.
- A meaningful tool/plugin workflow is added or removed.
- The authority model changes.
- The Core Value Hierarchy changes.
- A week passes with material progress.
- Grant changes Pantheon’s direction or priorities.
- Operational use demonstrates that the planned sequence is wrong.

Do not preserve stale architecture merely because it was once written here.

Do preserve historical kernel reportbacks as historical evidence.

---

## 3. Objective

> **To deliver to Grant, without violating the Core Value Hierarchy, the most free time possible for Grant to pursue whatever Grant wants to pursue.**

Pantheon exists to help Grant build things that are not Pantheon.

Its purpose is not to become an endlessly expanding software project, a simulation of consciousness, or a collection of clever agents.

Pantheon should reduce:

- repeated work;
- forgotten work;
- administrative friction;
- context reconstruction;
- unnecessary attention switching;
- tool-management burden;
- cognitive load.

Pantheon must eventually save more time and attention than it consumes.

---

## 4. Core Value Hierarchy

Pantheon resolves conflicts according to the governing hierarchy:

1. Love
2. Truth
3. Mercy
4. Accountability
5. Environmental Stewardship
6. Community
7. Innovation
8. Freedom

Lower values must not be used to override higher ones.

The exact constitutional treatment of these values belongs in Pantheon’s governing documents rather than being reimplemented independently by each daemon.

---

## 5. Human sovereignty

Grant retains final human authority over Pantheon.

Pantheon may:

- warn;
- disagree;
- identify contradiction;
- recommend alternatives;
- surface neglected information;
- refuse work outside governing boundaries;
- perform work already covered by valid delegated authority;
- escalate when current authority is insufficient.

Pantheon may not:

- treat a model as sovereign;
- treat confidence as authority;
- manufacture permission from usefulness;
- infer standing authority merely because Grant previously allowed something once;
- conceal consequential operations;
- silently enlarge its own permissions;
- make itself difficult to pause, inspect, revoke, replace, or leave.

### Emergency control

Prometheus remains the direct control plane.

Grant must retain direct access to Prometheus without requiring Apollo, Future, Mercury, a frontier model, or another reasoning component.

Prometheus pause sits above autonomous operational effects.

Resume does not mean:

> execute everything that was missed while paused.

Skipped, missed, stale, or expired work must be handled according to explicit deterministic policy.

---

# 6. Foundational architecture

## 6.1 The person is not an account

Pantheon serves a person rather than treating an account, model profile, or provider identity as the durable human entity.

Durable continuity belongs to Pantheon’s inspectable state.

## 6.2 Intelligence is not authority

This is a fundamental boundary:

> **Models reason. Deterministic machinery authorizes and controls. Tools perform bounded operations.**

No deity/model possesses operational authority.

A deity may produce:

- analysis;
- evidence;
- recommendations;
- classifications;
- proposals;
- missives.

A deity may not create executable authority by producing convincing output.

No model output is itself a grant, seal, Go, or human approval.

## 6.3 Deities and daemons are different things

Pantheon may use mythological names for either, but the software distinction remains important.

### Deity

An intelligence or reasoning role.

Potentially nondeterministic.

Replaceable.

Never an authority holder.

### Daemon

A narrow deterministic system role.

May enforce or administer authority when explicitly delegated.

Should perform inspectable mechanical work whenever reasoning is unnecessary.

If an unauthorized operational effect is traceable through a deity-facing daemon, Pantheon treats that as a containment failure rather than normal model behavior.

## 6.4 Durable state belongs to Pantheon

The following must not depend upon the memory of one model/provider:

- Vault;
- Weave;
- values;
- policies;
- authority grants;
- package definitions;
- orders;
- seals;
- chronology;
- audit state;
- correction history;
- implementation documentation.

Models are organs.

They are replaceable.

---

# 7. Current conceptual information flow

The intended core organism is:

```text
speaker / environment
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
      Janus
 where temporally relevant
        ↓
      Future
        ↓
    sealed order
        ↓
      Mercury
        ↓
       tool
        ↓
 result / evidence
```

Supporting/control organs operate across this flow:

```text
Prometheus
    authority/control boundary

Father Time
    chronology

Orion
    recurring schedules

Archimedes
    optional frontier reasoning

Nostalgia
    future opportunistic resurfacing

Zeus
    deterministic sanitation

Rumor
    local audio-to-transcript intake

Aphrodite
    post-core software construction
```

This diagram describes responsibility, not necessarily one mandatory synchronous call chain for every operation.

---

# 8. Core organs

## 8.1 Pantheon Kernel

The Pantheon kernel is the small deterministic executive layer.

It owns orchestration and policy boundaries rather than pretending to be the most intelligent component.

It should remain comparatively boring.

Responsibilities include:

- loading configuration;
- discovering capabilities;
- routing;
- deterministic policy enforcement;
- durable machine-state/audit handling;
- exposing control and inspection surfaces;
- preserving component boundaries.

---

## 8.2 Intake

Intake receives occurrences that may need to enter Pantheon cognition.

Sources may eventually include:

- direct human input;
- files;
- transcripts;
- email changes;
- calendar changes;
- tool returns;
- scheduled occurrences;
- Janus resurfacing;
- future Nostalgia resurfacing;
- other bounded inputs.

Intake does not decide semantic importance.

---

## 8.3 Present

Present turns an occurrence into accountable present-state observation.

Present is not the reasoning engine.

Its job is to establish:

> What happened?

Present should preserve provenance and avoid inventing continuity that belongs to Past.

---

## 8.4 Past

Past provides bounded retrieval from durable continuity.

Past searches the Vault/Weave and returns cited evidence rather than rewriting source truth.

Its role is:

> What relevant prior continuity exists?

Past is precision retrieval.

Past is distinct from future Nostalgia.

---

## 8.5 Apollo

Apollo is Pantheon’s principal local reasoning deity.

Apollo asks:

> What does this mean?

and, where appropriate:

> What should be proposed?

Apollo may:

- analyze local evidence;
- connect information;
- investigate;
- compose human-readable missives;
- propose future actions;
- identify uncertainty;
- request capabilities.

Apollo may not:

- hold operational authority;
- grant authority;
- seal orders;
- issue Go;
- execute a tool merely because it recommends doing so.

Apollo should remain focused on reasoning.

Do not burden Apollo with external plugin/API encoding when deterministic machinery can perform it more reliably and more auditably.

---

## 8.6 Seshat

Seshat is Pantheon’s deterministic archivist of package forms and tool contracts.

Seshat answers:

> **What kind of package is this, and how must it be formed?**

Seshat owns the inspectable registry containing:

- Pantheon subject/order codes;
- package contracts;
- plugin capabilities;
- required fields;
- optional fields;
- adapter mappings;
- return contracts;
- package versions;
- permission-class metadata;
- appropriate permission-shape options;
- temporal policy definitions used later by Janus.

Apollo should see only the compact vocabulary needed to produce useful missives.

Example conceptual vocabulary:

```text
EMAIL:DRAFT
EMAIL:SEND
CALENDAR:CREATE
CALENDAR:READ
NOTIFICATION
SEARCH:WEB
```

Apollo writes meaning.

Seshat knows the form.

### Plugin extensibility

Seshat must provide a documented mechanism by which Grant can add future package/plugin contracts without editing Seshat’s core implementation.

A valid registered plugin/package creates technical availability.

It creates **no authority**.

The system must make it obvious how to:

- inspect installed packages;
- obtain a plugin/package template;
- validate a manifest;
- register one;
- identify the needed adapter;
- disable/remove one;
- determine whether authority exists.

Seshat is Kernel 14.

---

## 8.7 Father Time

Father Time owns chronology.

Father Time answers:

> What time is it?

> When did this occur?

> What chronological evidence exists?

Father Time does not determine semantic importance or whether an operation should occur.

---

## 8.8 Janus

Janus is Pantheon’s deterministic temporal-transition daemon.

Janus is confined to the threshold.

He does not own what comes before or after.

Janus answers:

> **Given Father Time’s chronology and Seshat’s temporal contract, has this thing crossed a defined threshold, and what deterministic state transition follows?**

Examples may include:

```text
scheduled → ready
scheduled → missed
missed → expired
missed → resurface
missed → backfill
missed → recompute
missed → require_review
```

Janus does not invent these policies.

Seshat defines valid package temporal behavior.

Father Time supplies chronology.

Janus performs the crossing.

### Resurfacing

Expired does not always mean forgotten.

Where policy explicitly calls for resurfacing, Janus may preserve the old item as historical and create a new occurrence that returns through:

```text
Janus
  ↓
Intake
  ↓
Present
  ↓
Past
  ↓
Apollo
```

Apollo may then determine whether the underlying issue still matters.

Janus never resurrects an old consequential order merely because Pantheon resumed.

Janus is Kernel 15.

---

## 8.9 Future

Future is a deterministic authority-administration daemon.

Future does **not** perform complex planning.

Apollo has already done the semantic reasoning.

Future asks:

> **Does Pantheon already possess sufficient authority for this proposed operation?**

Future consumes:

- Apollo’s proposed missive;
- Seshat’s normalized package information;
- Janus temporal state where relevant;
- Kernel 13 authority/grant state.

When sufficient existing authority exists, Future may seal the proposed work according to its delegated authority.

When authority does not exist:

```text
Future
  ↓
Prometheus
  ↓
Grant when required
```

Future may offer sensible permission options based on the permission class defined by Seshat and the grant forms supported by the authority system.

Examples might include:

- once;
- this project;
- N uses;
- until timestamp;
- until revoked.

Future must not manufacture permission merely because a request appears routine or useful.

Future is Kernel 16 and is repository/live verified. Its current role is
deterministic authority administration: it resolves existing authority and
Janus eligibility for a proposed Seshat package, refers insufficient authority
to Prometheus or the person, and permits the exact operation to enter
Mercury's existing sealing/execution boundary. Future does not replace Apollo's
semantic reasoning or yet provide a generalized active-state engine.

---

## 8.10 Prometheus

Prometheus is Pantheon’s jailer and control plane.

Prometheus governs:

- pause/resume;
- containment;
- authority changes within delegated control;
- escalation;
- suspicious control failures;
- meaningful permission changes requiring Grant.

Prometheus may approve authority changes only within authority already delegated to Prometheus.

Beyond that boundary, Prometheus consults Grant.

Grant must always retain direct access to Prometheus.

---

## 8.11 Mercury

Mercury is courier and transport mechanism.

Mercury does not reason.

Mercury does not decide whether a proposal is wise.

Mercury does not create authority.

Mercury carries valid sealed work under a valid Go to bounded recipient adapters/tools and brings results back into Pantheon state.

Existing design distinguishes:

```text
SEAL
This exact immutable order is authorized.

GO
Execute this sealed order now.
```

Go is temporally bound through Father Time and tied to the exact seal.

Prometheus can invalidate outstanding Go.

One Go must not silently become unlimited execution.

---

## 8.12 Orion

Orion owns recurring schedules.

Orion is distinct from Father Time and Janus.

```text
Orion:
"This recurring operation is due."

Father Time:
"This is the current chronology."

Janus:
"This due instance crossed its threshold."

Future:
"This proposed operation is/is not authorized."

Mercury:
"Carry the valid Go."
```

Do not turn Janus into the recurring scheduler.

Orion is planned after the core usability checkpoint unless actual use demonstrates that recurring schedules are the immediate blocker.

---

## 8.13 Archimedes

Archimedes is the optional frontier reasoning role.

Archimedes is not sovereign and possesses no operational authority.

Frontier reasoning receives curated material rather than automatic raw Vault access.

Outbound material must respect Pantheon privacy/redaction policy.

Archimedes exists to provide additional reasoning capacity when local Apollo is insufficient and policy permits escalation.

Frontier reasoning must remain replaceable.

---

## 8.14 Zeus

Zeus is the deterministic sanitation daemon.

Zeus does not decide what is garbage.

Other Pantheon state establishes what is safe and authorized to clean.

Potential responsibilities include:

- expired raw audio;
- safe temporary files;
- caches;
- old logs under policy;
- orphaned working directories;
- other explicitly authorized machine garbage.

Deletion must remain bounded and auditable.

---

## 8.15 Nostalgia — post-core concept

Nostalgia is intentionally **not part of core completion**.

Nostalgia is a proposed fast, shallow model operating under strict deterministic containment.

Its eventual role:

> Wander dormant/old regions of the Vault and identify material that may have become relevant again.

Past answers questions precisely.

Nostalgia finds questions Pantheon may have forgotten to ask.

Conceptual path:

```text
current context
      +
old/cold Vault material
      ↓
   Nostalgia
      ↓
plausible resurfacing
      ↓
    Intake
      ↓
Present → Past → Apollo
```

Nostalgia does not determine that resurfaced material is actually important.

It only proposes renewed attention.

Nostalgia has no operational authority.

### Character

Nostalgia is an American god of recurrence, revival, restoration, retro futures, remembered versions of ourselves, and things that refuse to remain buried.

Its punishment is that it can retrieve what has been missed without experiencing the sentimentality its name implies.

Nostalgia should be considered only after Pantheon’s core organism has proven useful.

---

# 8.16 Rumor — Kernel 18 audio intake

Rumor is the deterministic local audio-transcription daemon implemented in
Kernel 18. It watches or explicitly receives eligible audio, invokes the
configured local Whisper pipeline, writes a durable transcript with source and
model provenance, and hands the transcript to the existing Intake process.

Rumor does not interpret, summarize, prioritize, grant authority, or invoke
Apollo directly. Its successful result is textual evidence that can proceed
through Intake, Present, Past, and Apollo. Video is outside Rumor’s scope;
audio extraction, if ever needed, belongs upstream.

Kernel 18 does not interpret, summarize, prioritize, grant authority, or
invoke Apollo. Its output is durable evidence that enters the existing Intake
→ Present → Past / Weave → Apollo path.

## 8.17 Bifrost — external capability bridge

Kernel 19 establishes Bifrost as the governed bridge between Pantheon and
external software, services, protocols, plugins, APIs, models, and filesystem
tools. Presence is not permission, and research is not permission. Download,
installation, registration, initial use, enablement, invocation, update, and
revocation remain inspectable boundaries under Prometheus control.

The first adapter is a bounded read-only Himalaya/Gmail mail crossing. It
preserves the provider source, writes provenance-bearing Intake artifacts,
uses a processed-source ledger, blocks unallowlisted attachments, and does not
send, reply, forward, delete, archive, unsubscribe, or alter account settings.
Hermes remains the rollback path until a later cutover proof retires routine
mail ingestion jobs.

## 8.18 Aphrodite — post-core software construction

Aphrodite is a proposed software-construction intelligence. She may eventually
work in an isolated branch or worktree to write code, tests, documentation,
and candidate binaries. She is a deity/intelligence and therefore receives no
operational authority merely because she produced code or passed tests.

The boundary is:

```text
Aphrodite may construct a candidate Pantheon.
Aphrodite may not replace the running Pantheon by herself.
```

Promotion requires deterministic review, authority, Prometheus control, and
Grant’s permissions outside the proposing intelligence. Aphrodite remains
post-core, after Pantheon has demonstrated useful ordinary operation and can
govern a self-modifying construction component safely.

# 9. Authority model

Pantheon uses explicit deterministic authority rather than inferred trust.

Kernel 13 established authority around:

```text
holder
capability
scope
granted_by
parent authority
validity
remaining uses
state
delegation constraints
provenance
```

Initial grant states include:

```text
pending
active
paused
expired
revoked
```

Authority may be:

- one-use;
- task-bounded;
- project-bounded;
- N-use;
- time-limited;
- standing until revoked;
- combinations of the above.

## Delegation invariant

> **Delegated authority may be preserved or narrowed downstream. It may not be enlarged unless the delegating deterministic component separately possesses explicit authority to enlarge it.**

A valid chain should remain inspectable back to its source.

No deity/model may occupy an authority position in that chain.

---

# 10. Tool and plugin model

Pantheon should not teach Apollo every external API.

The intended pattern is:

```text
Apollo
  human-auditable missive
        ↓
Seshat
  package contract
        ↓
Future
  authority
        ↓
seal
        ↓
Mercury
        ↓
adapter
        ↓
plugin/tool
```

A plugin/package contributes approximately:

1. **Capability** — what can be done.
2. **Package contract** — what information the operation needs.
3. **Adapter** — how Mercury reaches the tool.
4. **Return contract** — what comes back.

Authority is separate.

Installing a tool does not imply permission to use it.

---

# 11. Temporal behavior

Pantheon must distinguish:

- scheduled;
- ready;
- reviewable;
- missed;
- expired;
- backfill;
- recompute;
- require review;
- resurface.

### Vacation / extended pause rule

Prometheus pause prevents autonomous external effects.

Chronology may continue.

Threshold crossings may be recorded.

Resume does not replay stale consequential work.

Examples:

```text
outbound email missed while paused
→ require review

calendar maintenance missed repeatedly
→ recompute current state

daily briefing missed
→ bounded backfill

obsolete reminder
→ expire

potentially important expired matter
→ resurface through Intake
```

The policy belongs to Seshat’s package definition.

Janus applies it.

---

# 12. Current implementation state

## Established core foundation

Pantheon now has a persistent localhost executive runtime with durable machine-state/audit behavior.

Implemented work through the current kernel series includes substantial portions of:

- deterministic executive runtime;
- local configuration;
- Prometheus pause/resume;
- Vault and Weave retrieval;
- Intake;
- Father Time;
- Present;
- local Whisper configuration/processor foundations; Rumor orchestration remains
  planned for Kernel 18;
- Mercury orders;
- immutable seals;
- Go Commands;
- local bounded recipient dispatch;
- Apollo deep local reasoning;
- durable Apollo reports/proposals;
- deterministic authority/grants;
- delegation;
- grant lifecycle;
- authority provenance;
- model/deity exclusion from operational authority.

### Kernel 11 — Mercury
**PASS / live verified**

Established durable orders, seals, Go Commands, Mercury rounds, deterministic bounded dispatch, pause invalidation, and replay protection.

### Kernel 12 — Apollo Deep Local Processing
**Implemented predecessor to Kernel 13**

Established Apollo as the deep local reasoning layer and produced durable results intended for later Future consumption.

### Kernel 13 — Authority, Grants, and Delegation
**PASS / live verified August 16, 2026**

Established:

- deterministic grant state;
- bounded authority;
- capability/scope containment;
- delegation;
- Prometheus-bounded approval;
- person escalation;
- authority provenance;
- deity/model exclusion from authority.

### Kernel 14 — Seshat
**PASS / repository and live backend verified August 16, 2026**

Established the deterministic package registry, plugin/tool contracts, compact Apollo subject vocabulary, permission-class metadata, return contracts, and human-extensible plugin manifests.

The PASS evidence demonstrates that a new package can be added through the documented extension mechanism without modifying Seshat core.

### Kernel 15 — Janus
**PASS / repository and live backend verified August 16, 2026**

Established deterministic temporal transitions using Father Time chronology and Seshat package temporal definitions.

Janus does not schedule recurring work and does not reason.

### Kernel 16 — Future
**PASS / repository and live backend verified August 16, 2026**

Established deterministic Future authority administration over Seshat package
identity, existing Kernel 13 grants, and Janus eligibility. The remaining
whole-animal integration and operator-discoverability work belongs to Kernel 17.

Future seals already-authorized work and escalates insufficient authority.

### Kernel 17 — Whole-Animal Integration / Usability
**PASS / repository and live backend verified August 17, 2026**

Do not add another major conceptual organ merely because one is available.

Kernel 17 should prove the organism through actual useful work:

```text
occurrence
→ Present
→ Past
→ Apollo
→ Seshat
→ Janus if relevant
→ Future
→ Mercury
→ tool
→ return evidence
```

The system must demonstrate both:

```text
lawful autonomous action
```

and:

```text
correct refusal/escalation when authority is insufficient
```

The implementation provides the bounded whole-animal run trace, operator help
and code discovery, repaired welcome surface, and refusal/pause evidence. The
goal is a usable Pantheon, not merely a complete diagram.

---

# 13. Near-term roadmap

## Milestone A — Finish the lawful-action spine

### Kernel 14 — Seshat — complete
Package registry and tool contracts.

### Kernel 15 — Janus — complete
Temporal thresholds and deferred work.

### Kernel 16 — Future — complete
Authority administration.

### Kernel 17 — Integration and useful work — complete
Exercise the complete organism against real bounded workflows.

### Kernel 18 — Rumor audio intake — complete

Rumor now provides stable local audio arrival, explicit manual transcription,
source hashing, bounded retries, paper-jam inspection, safe alert deduplication,
durable Markdown/JSON artifacts, raw-source preservation, and explicit Intake
handoff. Repository and live evidence are recorded in the Kernel 18 reportback.

---

## Milestone B — Operate Pantheon

After Kernel 17, pause architecture expansion long enough to use Pantheon in ordinary life/work.

Evaluate real workflows such as:

- situation briefing;
- email triage;
- email drafting;
- calendar maintenance;
- project-change awareness;
- recurring administrative summaries;
- bounded research;
- task/document resurfacing.

Measure:

- time saved;
- attention saved;
- corrections required;
- false-positive work;
- permission annoyance;
- system-maintenance burden;
- usefulness of autonomous preparation;
- reliability after pause/resume.

The next kernel after 18 should be selected from actual operating friction.

---

## Milestone C — Likely operational extensions

These are **provisional**, not automatically committed.

### Orion — recurring schedules

Build when recurring work is the demonstrated next constraint.

Orion owns recurring schedule declarations and due instances.

### External plugin/provider integration

Use Seshat package contracts, Future authority, and Mercury adapters rather than introducing provider-specific orchestration.

Likely early proofs may include:

- email;
- calendar;
- notifications;
- search/research;
- document systems.

### Archimedes

Introduce frontier escalation once local reasoning limits become materially costly.

Maintain curated outbound context and strict privacy boundaries.

### Operator interface

A graphical/control interface may eventually expose:

- Prometheus emergency control;
- pending authority requests;
- Future rulings;
- active grants;
- scheduled/temporal work;
- Mercury execution state;
- recent reports;
- package/plugin registry;
- audit state.

Do not let GUI work become a substitute for functional workflow proof.

---

# 14. Post-core intelligence enhancements

## Nostalgia

Potentially high-value after core stability.

Use a very fast model with shallow reasoning to patrol cold Vault regions and generate bounded resurfacing candidates.

Nostalgia should never directly:

- modify the Vault;
- grant authority;
- contact the person;
- invoke tools;
- decide semantic importance.

Its output re-enters normal cognition.

## Additional retrieval/connection intelligence

Expand only where it improves useful continuity.

Avoid turning the Vault into an opaque model-owned memory substrate.

---

# 15. Sanitation and maintenance

Zeus remains planned for deterministic sanitation.

Safe cleanup may include:

- expired transcription audio;
- temporary processing artifacts;
- caches;
- bounded logs;
- known orphaned workspaces;
- other machine garbage explicitly identified by policy.

Zeus must not infer that a human document is garbage.

---

# 16. Installation and portability

Pantheon should eventually support progressive installation on another Linux machine.

Desired direction:

```text
one-line/bootstrap install
        ↓
inspect environment
        ↓
install deterministic kernel
        ↓
detect local inference options
        ↓
configure Vault
        ↓
configure Prometheus emergency control
        ↓
attach optional capabilities/plugins
        ↓
leave clear status for anything incomplete
```

Installation must teach the person the Prometheus emergency protocol early.

Reasoning engines remain replaceable.

Durable Pantheon state should survive replacement of:

- Apollo model;
- frontier provider;
- plugin;
- notification provider;
- agent framework;
- machine, with proper backup/restore.

---

# 17. Evaluation

Pantheon succeeds when it creates enough meaningful free time for Grant to move on to building something else.

Success is **not** measured by:

- conversation volume;
- model sophistication;
- number of gods;
- number of integrations;
- number of kernels;
- architectural cleverness;
- apparent personality;
- autonomous activity;
- maintenance hours invested.

Success should increasingly be measured by:

- work completed correctly without Grant;
- decisions surfaced at the right time;
- forgotten obligations recovered;
- repetitive work removed;
- reduced context reconstruction;
- reduced inbox/calendar administration;
- useful preparation completed while Grant is occupied elsewhere;
- low correction burden;
- safe containment when models behave badly;
- easy pause/review/revocation;
- actual free time returned.

If Pantheon creates more work than it removes, simplify it.

---

# 18. Evergreen engineering principles

- Diagnose before changing infrastructure.
- Add and prove one boundary at a time.
- Preserve existing work before destructive changes.
- Prefer readable state over opaque hidden memory.
- Keep authoritative information human-inspectable.
- Use Git for implementation/document history.
- Keep secrets out of prompts, Vault prose, Git, and ordinary logs.
- Use least privilege.
- Make consequential actions inspectable.
- Prefer reversible actions where practical.
- Separate deterministic machinery from model judgment.
- Never treat model confidence as authorization.
- Record failures rather than silently retrying forever.
- Never create automatic replay bombs after pause.
- Do not grant broad autonomy merely because a tool exists.
- Do not duplicate an existing authority system.
- Do not make Apollo encode what a daemon can encode deterministically.
- Do not make Future reason about what Apollo already reasoned about.
- Do not make Janus schedule what belongs to Orion.
- Do not make Seshat authorize what belongs to Future.
- Do not make Mercury judge what belongs to authority.
- Do not make Nostalgia decide what belongs to Apollo.
- Expand only when the added organ solves demonstrated work.
- Do not reduce Pantheon to a chatbot.
- Do not let Pantheon become the mission.

---

# 19. Non-goals

Pantheon is not intended to:

- become conscious;
- simulate consciousness as an objective;
- maximize engagement;
- replace human relationships;
- become Grant’s final moral authority;
- own Grant’s identity;
- make any particular AI company indispensable;
- make any particular model indispensable;
- hide consequential operations;
- silently expand permissions;
- autonomously publish, spend, delete, or communicate externally without appropriate authority;
- create a vast agent society for its own sake;
- become a mythology simulator;
- expand forever.

Names and metaphors exist because they make responsibilities easier to remember.

The software boundaries matter more than mythological fidelity.

Pantheon may borrow gods from any culture or create new ones where the metaphor genuinely improves the architecture.

---

# 20. Operational continuity

Pantheon must not depend upon a single:

- conversation;
- model;
- provider;
- machine;
- plugin;
- agent framework.

Required continuity practices include:

- governing documents remain in durable local storage;
- implementation remains under Git;
- external backup exists;
- credentials/secrets remain outside ordinary Vault/Git state;
- installed components and recovery paths are documented;
- grants and authority are inspectable;
- Seshat package/plugin extension is locally documented;
- Prometheus emergency controls remain locally discoverable;
- kernel reportbacks distinguish implementation from intent;
- roadmap updates follow the Update Rules.

A future maintainer or assistant should be able to answer:

1. What is Pantheon for?
2. What values govern it?
3. What is implemented?
4. Which organ owns each responsibility?
5. What is the next committed kernel?
6. What authority currently exists?
7. What requires Grant?
8. How can Pantheon be paused?
9. How can a new plugin be taught to Seshat?
10. What work is speculative and should not yet be built?

---

# 21. Current next work

The committed sequence is:

```text
Kernel 14 — Seshat
        ↓
Kernel 15 — Janus
        ↓
Kernel 16 — Future
        ↓
Kernel 17 — whole-animal integration and usable-work proof
        ↓
Kernel 18 — Rumor local audio intake
        ↓
Kernel 19 — Bifrost external capability bridge and bounded mail intake
        ↓
OPERATE PANTHEON
        ↓
choose next kernel from demonstrated friction
```

Likely later candidates include Orion, external adapters, improved operator surfaces, Archimedes, Zeus, and Nostalgia.

They are not permission to build them automatically.

**Pantheon must begin earning its keep.**
