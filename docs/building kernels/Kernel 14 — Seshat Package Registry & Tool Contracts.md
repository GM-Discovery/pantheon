# Kernel 14 — Seshat Package Registry & Tool Contracts

## Objective

Implement **Seshat**, Pantheon’s deterministic archivist for tool and plugin package contracts.

Seshat maintains the inspectable knowledge required to translate Pantheon’s small internal order vocabulary into the structured packages expected by tools and plugins.

Seshat exists so that reasoning deities do not need to know external API encodings, plugin-specific request structures, or implementation details.

The intended separation is:

```text
Apollo
  writes meaning
      ↓
short Pantheon subject/order code
      ↓
Seshat
  knows the form
  validates the missive
  constructs the package
      ↓
Future
  determines whether existing authority permits it
      ↓
seal
      ↓
Mercury
  carries the package
      ↓
adapter / plugin / tool
```

Seshat knows **how a request must be packaged**.

Seshat does not determine **whether the request may happen**.

Kernel 14 must also establish a documented extension mechanism allowing the person to teach Seshat about future plugins without modifying Seshat’s core Go implementation.

---

## A. Actual predecessor

Kernel 14 must begin from repository truth.

At implementation start record:

- current branch;
- exact predecessor commit;
- `git status --short`;
- Kernel 13 implementation/reportback status;
- installed runtime identity.

Kernel 13 reports repository and live backend PASS on August 16, 2026.

Do not assume its predecessor or reportback commit remains the current repository tip.

---

## B. Constitutional boundary

Kernel 14 preserves the existing principles:

- the person remains sovereign;
- intelligence is not authority;
- deities possess no operational authority;
- configured/installed capability is not permission;
- authority belongs to deterministic Pantheon machinery;
- Prometheus remains above operational control;
- Mercury transports authorized work rather than deciding what deserves authorization.

Seshat introduces another important separation:

> **Knowledge of how to invoke a capability does not constitute authority to invoke it.**

Installing a plugin, registering a package, validating a missive, or successfully constructing a tool request grants **zero operational authority**.

Kernel 13 remains the source of authority truth.

---

## C. Seshat is a daemon, not a deity

Seshat is deterministic.

She does not require a reasoning model.

She may:

- maintain package definitions;
- maintain the available package/code index;
- validate package definitions;
- validate incoming missives against deterministic contracts;
- transform Pantheon missives into tool packages;
- expose plugin/package documentation;
- report installed capabilities;
- report package versions;
- identify malformed or unsupported packages;
- expose appropriate permission shapes associated with a package class.

She may not:

- decide whether an action is desirable;
- grant herself authority;
- grant a plugin authority;
- approve an operation;
- expand a grant;
- seal an order merely because she can package it;
- execute a tool;
- interpret ambiguous prose using model reasoning;
- infer missing consequential fields.

---

## D. Apollo’s vocabulary

Apollo should not learn every plugin API.

Apollo receives a small, inspectable Pantheon vocabulary describing the classes of requests Pantheon currently knows how to package.

Examples may eventually resemble:

```text
EMAIL:DRAFT
EMAIL:SEND
CALENDAR:READ
CALENDAR:CREATE
CALENDAR:MAINTAIN
NOTIFICATION
SEARCH:WEB
FILE:READ
```

These examples are not all required Kernel 14 integrations.

The vocabulary should remain short enough that Apollo can reliably use it as part of its local reasoning context.

For each available code, Apollo needs only enough information to construct the meaningful missive.

For example:

```text
TYPE: EMAIL:DRAFT

TO: Person#17
SUBJECT: Meeting follow-up

Thank you for meeting with me today...
```

Apollo should not need to know:

- Gmail API JSON;
- OAuth implementation;
- HTTP routes;
- MIME construction;
- plugin-specific attachment encoding;
- database IDs used only internally;
- adapter invocation syntax.

That belongs below the reasoning boundary.

---

## E. Missives

A missive is the human-auditable semantic material produced for a proposed operation.

Seshat must support a deterministic boundary between:

```text
Pantheon routing/control headers
```

and:

```text
substantive payload
```

For example:

```text
TYPE: EMAIL:DRAFT
TO: Person#17
SUBJECT: Updated schedule

<body begins>

Hello...
```

Seshat may strip Pantheon-only headers when constructing the external tool package.

The original missive must remain durably traceable for audit/provenance where the surrounding workflow persists it.

Seshat must not silently rewrite substantive prose merely to satisfy an adapter.

If a required field is absent, Seshat should return a deterministic validation failure rather than inventing content.

---

## F. Package registry

Seshat maintains an inspectable registry of package contracts.

A package definition should contain at minimum:

```text
package_id
subject_code
version
description
capability
adapter
required_fields
optional_fields
payload_rules
return_fields
permission_class
permission_shapes
enabled
```

Where appropriate it may additionally contain:

```text
aliases
constraints
examples
schema_version
adapter_version
plugin_name
documentation
```

Avoid fields that have no demonstrated use.

The registry should be human-readable and machine-validatable.

---

## G. Package contract

A package contract describes how Pantheon translates a recognized missive into a bounded adapter request.

For example:

```text
subject_code: EMAIL:DRAFT

capability:
  email.draft

required_fields:
  - to
  - subject
  - body

optional_fields:
  - cc
  - bcc
  - attachments

adapter:
  gmail

returns:
  - draft_id
  - status
```

Seshat performs deterministic mapping.

Given the same valid missive, package definition, and relevant deterministic context, Seshat should construct the same logical package.

No model inference is required.

---

## H. Permission classes

Seshat may describe the **shape of authority that makes sense for a package class**.

Seshat does not grant that authority.

For example:

```text
EMAIL:DRAFT

permission_class:
  nonpublic_composed_output

permission_shapes:
  - once
  - project
  - bounded_uses
  - until_revoked
```

Another package might declare:

```text
EMAIL:SEND

permission_class:
  external_communication

permission_shapes:
  - once
  - bounded_uses
  - temporary
```

And a recurring self-directed briefing might eventually declare:

```text
BRIEF:DAILY

permission_class:
  recurring_self_delivery

permission_shapes:
  - once
  - until_revoked
```

These definitions tell Future which approval options are sensible to present or construct.

They do not constitute approval.

Kernel 13’s actual grant system remains authoritative.

---

## I. Context-sensitive permission shapes

Permission choices should not be reduced to one universal menu.

The package contract may identify reasonable grant shapes based on the class of operation.

A random consequential one-off command might naturally support:

```text
once
this_project
```

A recurring briefing to the person might naturally support:

```text
once
until_revoked
```

A bounded research capability might support:

```text
once
N_uses
until_timestamp
```

Seshat provides deterministic metadata describing these options.

Future later determines which options are legally available under the actual authority hierarchy.

No package definition may create an authority shape that Kernel 13 cannot represent.

---

## J. Plugin registration contract

A plugin/tool integration should be able to teach Seshat its package contracts without modifying Seshat’s core implementation.

The extension mechanism should use a documented, human-readable manifest format.

Prefer a common configuration format already appropriate to the repository unless there is a demonstrated reason to introduce another.

A plugin manifest should be able to declare:

- plugin identity;
- plugin/package version;
- one or more subject/order codes;
- capabilities exposed;
- required fields;
- optional fields;
- deterministic mapping rules;
- adapter target;
- expected return fields;
- permission class;
- permitted/sensible permission shapes;
- examples;
- validation constraints;
- compatibility/schema version.

Registration must not grant operational authority.

---

## K. Human extension workflow

Kernel 14 must make it obvious how the person can add another plugin later.

Provide documented commands or equivalent surfaces resembling:

```text
pantheon plugin help
pantheon plugin schema
pantheon plugin validate <path>
pantheon plugin register <path>

pantheon show plugins
pantheon show packages
pantheon show package <subject-code>
```

Exact command names may follow repository conventions.

The person should be able to ask Pantheon:

> How do I teach Seshat about another plugin?

and receive enough local documentation to do so without searching the source code.

At minimum, the installed system/repository should provide:

1. a plugin/package manifest template;
2. one minimal working example;
3. one more structured example;
4. field documentation;
5. validation instructions;
6. registration instructions;
7. inspection instructions;
8. removal/disable instructions if supported;
9. explanation that registration grants no authority;
10. explanation of where adapter implementation fits.

---

## L. Plugin manifest template

Provide an inspectable starter template resembling:

```text
plugin:
  name: example
  version: 1

packages:
  - subject_code: EXAMPLE:ACTION
    description: Example bounded action

    capability: example.action
    adapter: example

    required_fields:
      - target
      - body

    optional_fields:
      - note

    returns:
      - result_id
      - status

    permission_class: bounded_external_action

    permission_shapes:
      - once
      - bounded_uses
```

The final syntax should reflect the actual chosen format.

The template must be usable rather than pseudodocumentation disconnected from the implementation.

---

## M. Validation

Before registration, Seshat must deterministically validate a plugin/package manifest.

Validation should reject at minimum:

- missing subject code;
- duplicate/conflicting subject code;
- missing capability;
- missing adapter;
- malformed field definitions;
- unsupported manifest/schema version;
- invalid permission shape;
- malformed deterministic mapping;
- package definitions that attempt to declare themselves authorized.

Validation should clearly distinguish:

```text
invalid package definition
```

from:

```text
valid package, no authority
```

A package can be perfectly valid and completely unauthorized.

That is normal.

---

## N. Registration

Registration makes Seshat aware of a package.

It does not:

- create a Kernel 13 grant;
- modify Prometheus authority;
- authorize Future;
- authorize Apollo;
- authorize Mercury;
- invoke the plugin;
- establish recurring work.

After registration, Pantheon should be able to say:

```text
Package available: EMAIL:DRAFT
Capability: email.draft
Adapter: gmail
Authority: none
```

Availability and authority remain visibly separate.

---

## O. Package discovery

Seshat should expose the currently available package vocabulary in two forms.

### Human inspection

Enough detail for the person to understand what Pantheon knows how to package.

### Apollo-safe index

A compact representation suitable for Apollo’s reasoning context.

For example:

```text
EMAIL:DRAFT
  Draft an email.
  Fields: to, subject, body; optional attachments.

CALENDAR:CREATE
  Propose creation of a calendar event.
  Fields: calendar, title, start, end; optional attendees, location, notes.
```

Do not dump full adapter implementation details into Apollo context.

The Apollo index should explain **how to write the missive**, not **how the tool works internally**.

---

## P. Versioning

Package definitions must be versioned.

A plugin/tool contract can change independently of Apollo.

Seshat should make that manageable without requiring reasoning-layer changes.

At minimum, inspection should reveal:

```text
subject_code
package version
adapter
adapter/plugin version if known
```

If a package contract becomes incompatible with a previously constructed package, Pantheon should fail visibly rather than silently reinterpret consequential fields.

Do not build a generalized dependency/package manager in Kernel 14.

Simple explicit compatibility is sufficient.

---

## Q. Adapter boundary

Seshat knows **how to construct the package**.

Seshat does not necessarily implement the tool adapter itself.

Conceptually:

```text
Apollo missive
      ↓
Seshat package contract
      ↓
normalized Pantheon package
      ↓
Future authority
      ↓
seal
      ↓
Mercury
      ↓
adapter
      ↓
external/local tool
```

The adapter may be built into Pantheon or supplied by a plugin.

Seshat’s registry must identify the adapter required.

If the package is known but the adapter is unavailable:

```text
known package
+ unavailable adapter
= unavailable operation
```

It must not fall back to arbitrary shell execution.

---

## R. Return contracts

Seshat should also know what a package is expected to bring back.

Examples:

```text
EMAIL:DRAFT
returns:
  draft_id
  status
```

```text
CALENDAR:READ
returns:
  events
  source_calendar
  retrieved_at
```

This gives Mercury a deterministic expectation for the returned package.

Seshat does not reason about the significance of returned information.

Returned material requiring interpretation can later re-enter the normal Pantheon cognition path.

---

## S. No temporal semantics

Kernel 14 must not make Seshat responsible for temporal execution policy.

Seshat may describe that a package contains fields such as:

```text
scheduled_for
```

if a particular tool contract requires such a field.

But Seshat does not decide:

- when an operation becomes executable;
- whether a missed action should execute late;
- whether something should backfill;
- whether stale work should expire;
- whether expired work should be reconsidered;
- what happens after a long Prometheus pause.

Father Time remains the source of chronology.

Temporal threshold/state policy belongs to a separate deterministic daemon.

Current architectural candidate: **Janus**.

---

## T. Janus boundary reserved

Kernel 14 should avoid implementing behavior that would make later extraction of Janus difficult.

The intended future distinction is:

```text
Father Time
  tells when

Janus
  applies deterministic temporal transition rules

Seshat
  knows package form

Future
  administers authority

Mercury
  transports authorized work
```

An expired operation may eventually be:

- terminally expired;
- backfilled;
- recomputed;
- returned for human review;
- resurfaced into Intake for Present/Past/Apollo reconsideration.

Seshat does not choose among these.

Future does not need to own those semantics merely because Janus is not implemented yet.

---

## U. Resurfacing principle

Preserve room for a later temporal daemon to return materially changed or expired work to the cognition loop.

The intended pattern may become:

```text
old operation crosses threshold
        ↓
      Janus
        ↓
old operation marked expired
        +
new occurrence created
        ↓
      Intake
        ↓
      Present
        ↓
       Past
        ↓
      Apollo
```

Apollo can then determine whether the underlying matter remains important.

The deterministic temporal daemon should not pretend to understand semantic importance.

Kernel 14 does not implement this loop.

It merely must not foreclose it.

---

## V. Built-in proof packages

Kernel 14 should prove the package abstraction with a **small number** of materially different package types.

Do not integrate every available tool.

Prefer two examples that demonstrate different structures.

For example:

### Example 1 — notification

Simple bounded output:

```text
NOTIFICATION

required:
  message

optional:
  title
```

### Example 2 — structured draft or calendar operation

For example:

```text
EMAIL:DRAFT

required:
  to
  subject
  body

optional:
  attachments
```

or an existing harmless calendar/read operation where technically easier.

The exact examples should follow capabilities already available in the repository/runtime.

Do not add consequential external writes merely to make the demonstration impressive.

---

## W. Extensibility proof

**Kernel 14 does not PASS unless at least one test package is added through the documented extension mechanism without modifying Seshat’s core implementation.**

The test should demonstrate:

```text
write manifest
    ↓
validate manifest
    ↓
register manifest
    ↓
Seshat discovers subject code
    ↓
Apollo-safe index exposes it
    ↓
valid missive packages successfully
    ↓
inspection shows package
```

No operational authority need be granted to the test package.

Indeed, verification should demonstrate that the newly registered package remains unauthorized absent a Kernel 13 grant.

This proves Seshat is genuinely extensible rather than merely architected for hypothetical future extensibility.

---

## X. Package construction proof

Automated verification should demonstrate that Seshat can:

1. identify a registered subject code;
2. reject an unknown subject code;
3. validate required fields;
4. preserve optional fields;
5. reject missing consequential required fields;
6. strip Pantheon-only headers where appropriate;
7. preserve substantive payload;
8. deterministically construct the normalized package;
9. identify the required capability;
10. identify the required adapter;
11. expose expected return fields;
12. expose permission-class metadata;
13. expose appropriate permission-shape metadata;
14. do all of the above without invoking a reasoning model.

---

## Y. Security and containment verification

Tests must demonstrate:

1. registering a package creates no authority;
2. a package manifest cannot grant itself authority;
3. a plugin cannot identify itself as an authority holder;
4. Apollo cannot create authority by selecting a subject code;
5. a valid package with no grant remains unauthorized;
6. malformed package metadata cannot produce an executable fallback;
7. unknown adapters do not fall back to shell execution;
8. duplicate/conflicting subject codes fail visibly;
9. disabled packages cannot be used for new package construction;
10. existing Kernel 13 authority behavior remains intact.

---

## Z. Inspection surfaces

Provide read-only inspection consistent with existing Pantheon conventions.

At minimum, the person should be able to inspect:

- registered plugins;
- registered packages;
- subject codes;
- package versions;
- capabilities;
- adapters;
- required/optional fields;
- return contracts;
- permission classes;
- permission shapes;
- enabled/disabled state;
- package origin/manifest;
- whether the required adapter is presently available.

The goal is to answer:

> **What does Seshat know how to package right now?**

and:

> **How do I teach her another one?**

without opening the source code.

---

## AA. Human documentation requirement

Kernel 14 must leave behind concise human documentation specifically for future plugin addition.

This documentation should be written for a person who remembers Pantheon’s architecture but does **not** remember the implementation details of Kernel 14.

It should answer:

### What is a Seshat plugin/package?

### Where do manifests live?

### How do I copy the template?

### How do I define a subject code?

### How do I declare fields?

### How do I associate a capability?

### How do I identify the Mercury adapter?

### How do I define expected return fields?

### How do I describe sensible permission shapes?

### How do I validate it?

### How do I register it?

### How do I inspect it afterward?

### How do I disable/remove it?

### How do I know whether it has authority?

### What requires Go code versus only a manifest?

That final distinction is especially important.

A manifest can describe a package contract.

A genuinely new protocol/tool may still require an adapter implementation.

The documentation must say so clearly.

---

## AB. Suggested local documentation surface

Prefer making the extension instructions available both in repository documentation and through the CLI.

A command resembling:

```text
pantheon plugin help
```

should summarize the extension workflow and point to the installed template/documentation location.

A command resembling:

```text
pantheon plugin schema
```

should expose the currently supported manifest schema or its local path.

Do not require Internet access to learn how to extend Seshat.

---

## AC. Persistence

Use the smallest durable representation appropriate to the existing Pantheon runtime.

Package definitions should survive restart.

Do not store package knowledge only inside model prompts.

Do not require Apollo to remember installed plugins.

The registry is Pantheon state.

Human-readable manifests should remain the inspectable source of package definitions where practical.

A compiled/cache/index representation may exist for efficient deterministic use, but it should be reproducible from the inspectable definitions.

---

## AD. Reload behavior

Kernel 14 must explicitly document whether package registration:

- takes effect immediately;
- requires Seshat reload;
- requires Pantheon daemon restart.

Prefer the least disruptive deterministic mechanism consistent with the existing runtime.

Do not build hot-reload complexity solely for elegance.

Whatever behavior is implemented must be predictable and documented.

---

## AE. Failure behavior

Seshat failures should be boring.

Examples:

```text
unknown subject code
missing required field
invalid manifest
adapter unavailable
unsupported package version
package disabled
ambiguous duplicate code
```

These should produce explicit deterministic errors.

They must not cause:

- model fallback;
- guessed fields;
- arbitrary command execution;
- implicit authority escalation;
- silent package substitution.

---

## AF. Existing architecture reconciliation

Kernel 14 should update active architectural documentation where necessary so the information flow no longer implies that Apollo must know exact external tool encodings.

The intended separation is:

```text
Apollo
  semantic reasoning + missive

Seshat
  package knowledge + deterministic formatting

Future
  authority administration

Mercury
  transport/execution

Plugin/tool
  bounded operation
```

Kernel 13’s authority architecture remains unchanged: capability availability and authorization are separate. Kernel 13 already establishes lifecycle states, bounded grants, delegation containment, Prometheus escalation, and deity/model exclusion from authority.

Historical reportbacks should remain historical rather than being rewritten to appear prescient.

---

## AG. Automated verification

Before repository PASS:

```text
gofmt -w $(rg --files -g '*.go')
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel14 .
```

Focused Kernel 14 tests should cover:

1. package registry initialization;
2. manifest validation;
3. manifest registration;
4. persistence across restart/reload boundary;
5. package discovery;
6. Apollo-safe vocabulary/index;
7. deterministic missive parsing;
8. deterministic package construction;
9. required/optional field handling;
10. payload preservation;
11. adapter lookup;
12. return contract;
13. permission-class metadata;
14. permission-shape metadata;
15. no-authority-on-registration;
16. deity/model authority containment remains intact;
17. duplicate/conflicting package rejection;
18. unavailable adapter handling;
19. disabled package handling;
20. unsupported manifest version handling;
21. extension without core Seshat modification;
22. existing Kernel 11–13 regression suite remains green.

---

## AH. Live verification

Before claiming live PASS:

1. build the exact Kernel 14 candidate;
2. record candidate SHA-256;
3. install the exact candidate to the runtime;
4. restart only necessary Pantheon service(s);
5. confirm runtime health;
6. inspect Seshat/package registry;
7. validate a manifest using the live CLI;
8. register or load a harmless test package through the documented extension mechanism;
9. confirm it appears in package inspection;
10. confirm it appears in the Apollo-safe package index;
11. construct a valid normalized package from a harmless missive;
12. prove the package has no operational authority merely because it was registered;
13. restart/reload according to the documented lifecycle and confirm package persistence;
14. verify existing authority inspection still works;
15. record exact live evidence.

Do not create consequential external state merely to prove Seshat works.

---

## AI. Non-goals

Kernel 14 does not:

- implement Future;
- implement Janus;
- make Seshat an authority holder merely by virtue of registry ownership;
- give Apollo direct plugin knowledge beyond the safe package vocabulary;
- teach Apollo external API encoding;
- implement every available plugin;
- create a general package manager;
- download arbitrary code from the Internet;
- grant authority when a plugin is installed;
- execute arbitrary shell commands;
- create recurring autonomous work;
- determine temporal backfill/expiry behavior;
- implement sophisticated scheduling;
- enable Archimedes;
- build the GUI;
- replace Kernel 13 grants;
- replace Mercury;
- create a new reasoning agent.

---

## AJ. Reportback requirements

The Kernel 14 reportback must state:

- exact predecessor commit;
- starting repository state;
- active documentation reconciled;
- final Seshat package architecture;
- manifest format selected and why;
- registry persistence mechanism;
- package fields actually implemented;
- subject-code behavior;
- Apollo-safe index behavior;
- deterministic missive parsing behavior;
- deterministic package construction behavior;
- adapter discovery behavior;
- return-contract behavior;
- permission-class and permission-shape behavior;
- human plugin-extension workflow;
- exact manifest template location;
- exact extension documentation location;
- CLI inspection/help surfaces;
- extension proof performed without modifying Seshat core;
- proof that registration created no authority;
- automated verification results;
- live verification results;
- candidate and installed binary SHA-256;
- files changed;
- known limitations;
- report expiry conditions.

Do not claim generic plugin compatibility merely because one manifest parses.

State exactly what the extension contract supports.

---

## AK. Success condition

Kernel 14 PASS means:

> **Pantheon possesses a deterministic, inspectable, extensible package language between reasoning and authority.**

After PASS:

- Apollo can discover the small vocabulary of package types Pantheon understands.
- Apollo can produce human-auditable missives without knowing plugin encoding.
- Seshat can deterministically recognize and package those missives.
- Package definitions identify capabilities and adapters.
- Package definitions can describe sensible permission shapes without granting permission.
- New package contracts can be added through documented manifests without modifying Seshat core.
- The person can discover how to add another plugin locally.
- Plugin registration grants no authority.
- Future can subsequently consume Seshat packages and Kernel 13 authority without becoming responsible for plugin-specific formatting.

---

## AL. Next safe successor

If Kernel 14 passes, the next safe successor is:

**Kernel 15 — Future Authority Administration**

Future shall consume Seshat-normalized proposed operations, resolve them against Kernel 13 authority, seal operations already covered by valid authority, and refer insufficient authority toward Prometheus or the person.

Future shall not reason about the substantive wisdom of Apollo’s proposal.

Future shall not encode plugin-specific requests.

Future shall not manufacture permission.

### Reserved successor

A subsequent kernel should implement the deterministic temporal threshold role currently called:

**Janus**

Janus will operate alongside Father Time to govern transitions such as:

```text
scheduled
→ executable

scheduled
→ missed
→ expired

missed
→ backfill

missed
→ recompute

expired
→ resurface through Intake
```

Janus will not determine semantic importance. Where expired or changed work requires renewed understanding, Janus may create a new occurrence that re-enters Intake → Present → Past → Apollo.

Do not implement Janus inside Seshat merely to accelerate the roadmap.