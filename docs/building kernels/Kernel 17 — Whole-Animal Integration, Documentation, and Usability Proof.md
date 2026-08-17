# Kernel 17 — Whole-Animal Integration, Documentation, and Usability Proof

## Objective

Kernel 17 integrates the current Pantheon organism into a usable whole.

It must reconcile Pantheon’s active documentation **before finalizing code**, then make the runtime conform to that documentation.

Kernel 17 should proceed under the working assumption that the current architecture succeeds.

Do not write documentation as though Pantheon is merely an experiment that will probably fail.

Instead document:

> **This is how Pantheon works when the architecture succeeds.**

Where implementation reveals a major consideration necessary to reach that successful state, append or revise the documentation accordingly.

Kernel 17 is not primarily a new-organ kernel.

Its job is to make the existing Pantheon understandable, inspectable, operable, and useful.

---

## A. Documentation comes first

Before completing Kernel 17 code, reconcile Pantheon’s active documents.

The coding agent must read the current repository documentation and identify:

- stale architecture;
- conflicting daemon/deity definitions;
- superseded information flow;
- outdated kernel sequencing;
- obsolete Hermes/Terra assumptions;
- duplicated responsibility;
- missing current commands;
- missing authority behavior;
- missing package/plugin behavior;
- missing temporal behavior;
- missing operator guidance.

Do not preserve contradictions merely because they exist in tracked files.

Historical reportbacks remain historical.

Active governing and architecture documents should describe the system Pantheon is now becoming.

---

## B. Documentation success posture

Write active documentation from the perspective that the architecture works.

For example, prefer:

> Future resolves proposed work against existing authority.

rather than:

> Future may someday potentially resolve proposed work if this architecture proves viable.

The roadmap may distinguish implemented, committed, and post-core work, but architectural documents should describe intended settled responsibility clearly.

If implementation exposes a major requirement necessary for that architecture to succeed, document it.

Do not bury major architectural discoveries only inside code comments or test files.

---

## C. Required document reconciliation

At minimum review and reconcile:

- Pantheon Roadmap;
- Foundations;
- Construction;
- Software;
- configuration documentation;
- command/operator documentation;
- Prometheus emergency-control documentation;
- active daemon/deity descriptions;
- package/plugin extension documentation;
- Father Time / Janus temporal behavior;
- Future authority behavior;
- Mercury execution behavior;
- current CLI commands;
- welcome/startup guidance.

Where documents conflict, repository/runtime truth and later accepted architecture should guide the correction.

Do not rewrite historical kernel reportbacks to match newer architecture.

---

## D. Canonical architecture

The active documentation should converge on the current core flow:

```text
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
seal
  ↓
Mercury
  ↓
tool / plugin
  ↓
result / evidence
```

Supporting control roles include:

```text
Prometheus
  control / pause / authority escalation

Father Time
  chronology

Orion
  recurring schedules — later

Archimedes
  frontier reasoning — later

Zeus
  sanitation — later

Nostalgia
  old-material resurfacing — later

Aphrodite
  software construction — later
```

The documentation should clearly distinguish:

- deity reasoning;
- deterministic daemon behavior;
- authority;
- packaging;
- temporal transition;
- scheduling;
- transport;
- execution.

---

## E. Whole-animal integration

Kernel 17 should prove the current architecture as a connected organism.

The expected integrated path is:

```text
occurrence / direct request
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
      Janus if relevant
        ↓
      Future
        ↓
authority sufficient?
   ┌───────────┐
   │           │
  yes          no
   │           │
 seal      Prometheus / Grant
   │
Mercury
   │
adapter/tool
   │
result
```

Kernel 17 PASS should require at least one meaningful harmless path through as much of this chain as the actual implementation supports.

---

## F. Usability is part of PASS

Kernel 17 should not be considered complete merely because:

- tests pass;
- all daemons start;
- database rows exist;
- endpoints return JSON;
- the architecture can be traced manually.

Grant should be able to operate Pantheon without remembering obscure internal command strings.

The runtime should surface its capabilities.

---

## G. Pantheon help

Implement a useful help surface.

At minimum:

```text
pantheon help
```

should provide concise operator guidance.

A command such as:

```text
pantheon help --commands
```

or an equally clear repository-consistent alternative must list available commands.

The exact syntax may be changed if a cleaner CLI convention already exists, but Pantheon must provide an obvious way to discover its commands locally.

The operator should not need to:

- open source code;
- inspect route tables;
- remember prior chat instructions;
- search old kernel documents.

---

## H. Command discovery

The command-discovery surface should expose current usable commands grouped sensibly.

Suggested groups:

```text
STATUS / INSPECTION

CONTROL

INTAKE / PRESENT / PAST

APOLLO

SESHAT / PLUGINS

JANUS

FUTURE

MERCURY

AUTHORITY / GRANTS

HELP
```

Do not list commands that do not actually exist.

Do not hide implemented commands merely because they were added in an older kernel.

Each command should provide:

- command form;
- one-line purpose;
- required arguments;
- whether it mutates state;
- whether it requires the trusted control boundary where relevant.

Keep normal help concise.

Detailed help may be available through subcommands.

---

## I. Codes and subject vocabulary

Pantheon help must also provide a discoverable way to surface Seshat’s current subject/order codes.

For example:

```text
pantheon help --codes
```

or:

```text
pantheon show codes
```

or another clear equivalent.

The user should be able to see:

- available subject codes;
- short meaning;
- required fields;
- optional fields where useful;
- associated capability;
- package status.

Example:

```text
EMAIL:DRAFT
  Draft an email.
  Requires: to, subject, body.

NOTIFICATION
  Prepare a local notification.
  Requires: message.
```

Do not expose unnecessary adapter internals in the normal code index.

The goal is:

> **What words can Apollo/Pantheon use to describe work?**

---

## J. Prometheus Protocol placement

The welcome screen must prominently preserve the Prometheus emergency protocol.

Immediately beneath or adjacent to the Prometheus Protocol instructions, add command-discovery guidance.

Conceptually:

```text
IN CASE OF EMERGENCY

pantheon prometheus protocol --pause

Pauses Pantheon and autonomous activity for inspection.

Need commands?
pantheon help --commands
```

If another final help syntax is selected, use the actual implemented command.

Prometheus emergency control remains visually dominant.

Help discovery must be visible enough that Grant does not need to remember Pantheon command syntax.

---

## K. Welcome screen repair

The welcome screen currently has a damaged/misaligned right-side border.

Kernel 17 must fix it.

Do not merely change one known line if the layout mechanism itself is fragile.

The implementation should ensure that:

- left and right borders align;
- all lines fit the declared width;
- ANSI escape codes do not count incorrectly toward visible width;
- long model/path/status strings do not break the box;
- Unicode or ASCII width assumptions are handled consistently;
- terminal width constraints degrade cleanly where practical.

If the welcome screen uses manual padding, replace or centralize it with a reliable visible-width helper where appropriate.

---

## L. Welcome screen content

The welcome screen should be useful rather than decorative.

It should surface at least:

- Pantheon identity/runtime state;
- Prometheus Protocol;
- pause command;
- command-discovery command;
- concise status information;
- major degraded/unavailable state where relevant.

Avoid turning startup into a full dashboard.

The welcome screen should answer:

> Is Pantheon alive?

> How do I stop it?

> How do I find out what it can do?

---

## M. Operator documentation

Provide or reconcile a concise operator guide.

It should explain:

### Starting/checking Pantheon

### Prometheus pause/resume

### Finding commands

### Finding Seshat codes

### Inspecting authority

### Inspecting Future

### Inspecting Mercury

### Inspecting Janus transitions

### Inspecting plugins/packages

### Adding another Seshat package/plugin

### Reading recent results/reportbacks

### What to do when something is blocked

The guide should favor commands that actually work over architectural prose.

---

## N. Documentation architecture map

Provide one concise canonical architecture map in active documentation.

Avoid forcing future maintainers to reconstruct Pantheon by reading fifteen kernel reports.

The map should identify for each major component:

```text
NAME
TYPE
RESPONSIBILITY
MAY DO
MAY NOT DO
INPUT
OUTPUT
AUTHORITY STATUS
```

This may be tabular or section-based.

It should include at least:

- Pantheon kernel;
- Intake;
- Present;
- Past;
- Apollo;
- Seshat;
- Father Time;
- Janus;
- Future;
- Prometheus;
- Mercury.

Later/reserved components may be listed separately.

---

## O. Documentation index

If the repository does not already have a clear documentation entry point, create one.

A future coding agent or human should be able to start from one obvious document and learn where to find:

- governing principles;
- architecture;
- roadmap;
- configuration;
- operations;
- plugin extension;
- kernel reportbacks.

Do not require filesystem archaeology.

---

## P. Whole-animal proof — authorized path

Kernel 17 should prove one harmless end-to-end operation that is already authorized.

The ideal path demonstrates:

```text
input / occurrence
→ Present
→ Past where relevant
→ Apollo
→ Seshat
→ Janus if relevant
→ Future
→ seal
→ Mercury
→ adapter/tool
→ result
```

If one component is legitimately unnecessary for that operation, document why it was skipped.

Do not artificially route through a component solely to make the diagram look complete.

---

## Q. Whole-animal proof — insufficient authority

Prove a second path where:

```text
Apollo proposes work
→ Seshat packages it
→ Future checks authority
→ authority insufficient
→ Prometheus / person boundary
→ NO EXECUTION
```

This is equally important evidence of a functioning Pantheon.

The system succeeds both when it acts lawfully and when it refuses to act unlawfully.

---

## R. Whole-animal proof — time boundary

Where Kernel 15 supports it, prove one harmless temporal case:

```text
package with temporal rule
→ Father Time chronology
→ Janus transition
→ Future respects resulting state
```

Prefer a no-external-effect test such as:

- expired;
- require review;
- resurfaced local occurrence.

Do not add external writes merely for proof.

---

## S. Whole-animal proof — plugin/package extensibility

Confirm Seshat’s plugin/package extension remains usable after integration.

At minimum verify:

```text
manifest
→ validate
→ register
→ help/code discovery
→ normalized package
→ no implicit authority
```

The package should appear through the same help/inspection mechanisms Grant will actually use.

---

## T. No hidden scaffolding

Kernel 17 should identify temporary development scaffolding that Pantheon currently depends upon operationally.

Do not remove useful development tools merely because they are external.

But document what is:

- core Pantheon;
- development scaffold;
- external adapter;
- local model dependency;
- optional service.

This is particularly important because future Aphrodite work may eventually replace part of the current external software-construction scaffold.

---

## U. Meaningful documentation over exhaustive documentation

Kernel 17 should prefer a small set of good documents over a large number of repetitive documents.

Delete or consolidate active documentation only where safe and appropriate.

Historical evidence should remain preserved.

A good active documentation set should tell a new helper:

1. What Pantheon is.
2. Why it exists.
3. Who does what.
4. What authority means.
5. What is currently implemented.
6. How to operate it.
7. How to stop it.
8. How to discover commands.
9. How to add package/plugin knowledge.
10. What comes next.

---

## V. Speculate success, document friction

When reconciling active documentation, assume the architecture reaches the intended successful state.

Do not litter documents with unnecessary uncertainty.

However, if implementing Kernel 17 reveals a **major consideration required to achieve that success**, append it to the appropriate active documentation and reportback.

Examples:

- a missing provenance link;
- an unusable CLI boundary;
- a dangerous pause interaction;
- a package ambiguity;
- an authority gap;
- a required adapter abstraction;
- an operator discoverability problem.

Do not conceal these because the documentation is written from a success posture.

The distinction is:

> **Assume the architecture works. Document what it requires to work.**

---

## W. Help verification

Automated tests should verify help surfaces.

At minimum:

1. `pantheon help` succeeds.
2. command discovery succeeds.
3. implemented commands appear.
4. nonexistent commands are not advertised.
5. Seshat code discovery succeeds.
6. registered test package codes appear.
7. help works while Pantheon daemon is unavailable where practical for local CLI metadata.
8. Prometheus protocol command shown in help matches actual implementation.
9. help output does not require a model.
10. help does not mutate state.

---

## X. Welcome-screen verification

Automated tests should validate welcome rendering.

At minimum:

1. right border aligns on every boxed line;
2. ANSI formatting does not corrupt width;
3. typical long model names do not break layout;
4. long Vault/runtime paths do not break layout;
5. Prometheus Protocol appears;
6. help command appears immediately beneath/near it;
7. displayed pause command matches actual command;
8. displayed help command matches actual command.

Where exact terminal-width testing is impractical, test the rendering helper separately.

---

## Y. Documentation verification

Kernel 17 should include a documentation consistency check where practical.

At minimum ensure:

- documented Prometheus command exists;
- documented help command exists;
- documented inspection commands exist;
- documented current kernel sequence matches the roadmap;
- active architecture docs no longer assign stale planning behavior to Future;
- active architecture docs identify Seshat and Janus correctly;
- plugin-extension docs point to real files/commands.

Do not attempt to build a generalized natural-language documentation linter.

Use targeted deterministic checks for high-risk drift.

---

## Z. Automated verification

Before repository PASS:

```text
gofmt -w $(rg --files -g '*.go')
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel17 .
```

Kernel 17 focused tests should cover:

1. command-discovery help;
2. Seshat code discovery;
3. Prometheus Protocol help accuracy;
4. welcome-screen border rendering;
5. welcome-screen command visibility;
6. authorized whole-animal path;
7. insufficient-authority path;
8. temporal path where supported;
9. Seshat plugin extension remains functional;
10. no implicit authority from package registration;
11. Future idempotency remains intact;
12. Mercury Go/idempotency remains intact;
13. Prometheus pause/no-replay remains intact;
14. active documentation command references match actual CLI where deterministically checkable;
15. all prior kernel regression tests remain green.

---

## AA. Live verification

Before claiming live PASS:

1. build exact Kernel 17 candidate;
2. record SHA-256;
3. install exact candidate;
4. restart only necessary services;
5. verify Pantheon service health;
6. display the welcome screen;
7. visually verify border repair;
8. verify Prometheus Protocol display;
9. verify command-discovery display;
10. run `pantheon help`;
11. run command discovery;
12. run Seshat code discovery;
13. inspect authority;
14. inspect Future;
15. inspect Janus;
16. inspect Mercury;
17. exercise one harmless authorized whole-animal path;
18. prove exactly one resulting execution;
19. exercise one insufficient-authority path;
20. prove no execution;
21. verify pause/no-replay;
22. confirm active documentation paths exist;
23. record exact evidence.

---

## AB. Reportback requirements

The Kernel 17 reportback must begin with documentation work rather than burying it after implementation.

It must state:

### Documentation

- documents reviewed;
- documents changed;
- stale architecture corrected;
- documents consolidated or retired;
- canonical architecture location;
- operator guide location;
- plugin-extension documentation location;
- command/help documentation location;
- major considerations discovered while reconciling documentation.

### Runtime usability

- final help syntax;
- final command-discovery syntax;
- final Seshat code-discovery syntax;
- welcome-screen changes;
- border-rendering fix;
- Prometheus Protocol placement.

### Integration

- authorized end-to-end proof;
- insufficient-authority proof;
- temporal proof where applicable;
- package/plugin proof;
- exact components exercised;
- any component legitimately skipped and why.

### Verification

- automated test results;
- vet/build results;
- live verification;
- candidate and installed SHA-256;
- files changed;
- known limitations;
- inherited verification debt;
- report expiry conditions.

Do not call Kernel 17 PASS if the code works but active documentation remains materially contradictory.

---

## AC. PASS criteria

Kernel 17 PASS requires all of the following:

### Documentation

Pantheon has meaningful active documentation describing the current organism.

### Discoverability

Grant can discover commands without remembering them.

### Codes

Grant can inspect Seshat’s current package/order vocabulary.

### Emergency control

Prometheus Protocol is prominent and accurate.

### Welcome screen

The startup display is structurally correct and no longer has the broken right-side border.

### Integration

Pantheon demonstrates a lawful bounded action across the existing organism.

### Refusal

Pantheon demonstrates correct non-execution when authority is insufficient.

### Safety

Pause/no-replay remains intact.

### Extensibility

Seshat’s documented package/plugin extension still works.

### Evidence

The reportback clearly states what was proven rather than inferring whole-system success from isolated tests.

---

## AD. Post-Kernel-17 rule

After Kernel 17, operate Pantheon.

Do not automatically start Kernel 18 because the number exists.

Use the system for real work.

The next kernel should address the first meaningful friction encountered during operation.

Likely candidates remain:

- Orion — recurring schedules;
- additional bounded adapters/plugins;
- operator GUI;
- Archimedes — frontier reasoning;
- Zeus — sanitation;
- Nostalgia — opportunistic resurfacing;
- Aphrodite — bounded software construction.

None is automatically next.

---

## AE. Success posture after Kernel 17

Assuming Kernel 17 succeeds, Pantheon should be documented as a functioning local-first personal AI control system capable of:

```text
observe
remember
reason
compose
package
track time
resolve authority
seal
transport
execute
record
escalate
stop safely
```

The next question is no longer:

> **Can we construct Pantheon?**

It becomes:

> **What work should Pantheon now remove from Grant’s life?**