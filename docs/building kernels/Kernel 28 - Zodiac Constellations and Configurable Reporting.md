# Kernel 28 — Zodiac Constellations and Configurable Reporting

## Implementation status — 2026-08-23

The completed Kernel 28 increment is now present in Pantheon. Panopticon can
list the reusable Libra constellation and create, pause, or enable report
recipes. A recipe can run on a daily or weekly schedule, or after a verified
external Intake conversion; select source families and Vault path prefixes;
require a minimum evidence count; choose report sections and report-only
instructions; and write to multiple durable Vault outputs. Scheduled recipes
create typed Zodiac schedules; ingestion recipes use a durable bounded queue.
The GUI can evaluate Zodiac through its local session; it no longer
mistakenly requires the CLI control token.

A due Libra instance now selects verified Vault sources, creates an idempotent
assembly manifest, sends an isolated Apollo order through Mercury, verifies a
final Vault Markdown report, optionally delivers a local notification, and
records the full receipt chain. Missing evidence and incomplete Apollo work
remain visible as Needs You states rather than false successes.

## Mission

Expand Zodiac from a schedule keeper into a configurable constellation keeper
without making it a reasoning organ or a personal hardcoded cron table.

Kernel 28 introduces a reusable reporting architecture:

```text
Panopticon report definition
          ↓
      Libra constellation
          ↓
       Zodiac schedule
          ↓
      due instance
          ↓
 report assembly → Apollo → Vault report → optional local notification
```

Panopticon is where a person configures what reports exist. Zodiac evaluates
when they are due. The report assembler selects and proves evidence. Apollo
composes cited prose. Future and Mercury remain the authority/execution path
for consequential operations.

## Why constellations

A single schedule is too small to express a coherent operating rhythm. A
constellation groups related schedules, report definitions, windows, source
policies, and delivery rules while keeping each due instance inspectable.

The constellation is declarative metadata, not executable code. It does not
contain shell commands, arbitrary prompts with authority, or hidden model
behavior.

### Libra

Libra is the first constellation archetype. Its governing idea is balance:

- balance across the configured reporting window;
- balance between source families and explicit gaps;
- balance between primary evidence and derived interpretation; and
- balance between what is complete, unresolved, waiting, and proposed.

Balance does not mean pretending every source deserves equal weight. Libra
should surface asymmetry and missing expected inputs rather than silently
correcting them.

Libra is a reusable template/constellation, not an automatically active
personal schedule. A person can create, rename, configure, pause, or remove
Libra report definitions through Panopticon.

## Data model

### Constellation

```text
id
name
archetype                  e.g. libra
description
owner
timezone
enabled
policy_json                typed source/window/balance policy
created_at
updated_at
```

### Report definition

```text
id
constellation_id
name
enabled
report_kind                 daily | weekly | custom
window_policy               previous_day | previous_week | rolling | explicit
source_policy_json          allowed source families and required evidence
destination_policy_json     Vault destination and filename template
apollo_policy_json          bounded instructions and citation requirements
notification_policy_json    success/failure/needs-you behavior
catch_up_policy
recipe_json                 trigger, source selectors, sections, outputs
created_at
updated_at
```

### Zodiac schedule

Existing Zodiac schedules remain the temporal declarations. Kernel 28 adds a
typed link to a report definition/constellation rather than encoding report
semantics only in a free-form payload. The first increment carries that link
in the schedule source/source-id pair and a validated payload; a later schema
increment can promote the link to dedicated columns without deleting existing
declarations. Existing schedules remain inspectable as legacy declarations
until a person maps or retires them in Panopticon.

Each due instance must retain:

- constellation and report-definition IDs;
- the exact local time and reporting window;
- catch-up decision;
- assembly receipt ID;
- Apollo order/report IDs when created;
- notification receipt when requested; and
- final state or Needs You reason.

## Panopticon surface

Panopticon exposes the Libra configuration area inside Zodiac, separate from
low-level configuration fields. The report-run receipt chain is visible next
to each definition and in the dedicated receipt list.

### Configure

The person can:

- create or list a constellation;
- select the Libra archetype;
- create, pause, or enable report definitions;
- choose a scheduled or verified-intake trigger;
- choose daily or weekly cadence;
- choose timezone and local delivery time;
- choose the reporting window;
- choose source families, optional Vault-relative path prefixes, and a
  minimum verified source count;
- choose report sections and report-only instructions;
- choose one or more bounded Vault outputs;
- inspect assembly, Apollo, final Vault, and notification receipts; and
- choose whether a success notification is requested; and
- choose catch-up behavior.

Every submitted field is typed and validated. The current UI shows the
operational consequence of each choice; a richer dry-run/source preview remains
a future refinement.

### Inspect

The person can inspect:

- active constellations and report definitions;
- next due times and reporting windows;
- configured source selectors and missing expected inputs;
- assembly, Apollo, Vault, and notification receipts;
- duplicate and catch-up decisions; and
- stale or blocked instances.

Panopticon must not display a report as successful merely because Zodiac made a
due instance. Success requires the downstream receipt chain.

## Default report examples

These are templates, not forced personal defaults:

### Daily recap

```text
cadence: daily
window: previous local operating day
destination: 10 Operations/Situation Briefs/
assembly: daily evidence manifest
reasoning: cited Apollo report
notification: local success ding after durable report write
```

### Weekly overall recap

```text
cadence: weekly
window: previous Monday–Sunday (configurable)
destination: 70 Pantheon Reports/Weekly/
assembly: weekly evidence manifest over configured daily/event reports
reasoning: cited Apollo report
notification: configurable local success/failure behavior
```

The user may choose different times, windows, destinations, source policies,
and notification behavior. The architecture must work for another person or
installation without changing Go constants or renaming the report in source.

## Due-time pipeline

```text
Zodiac evaluates due constellation schedule
          ↓
creates/reconciles one due instance
          ↓
report pipeline validates definition and window
          ↓
report assembler creates/reuses exact evidence manifest
          ↓
Future/Mercury boundary is applied where required
          ↓
Apollo produces a cited report
          ↓
Vault write is verified
          ↓
optional local notification is sent
```

Verified external Intake conversion may also enter the same bounded pipeline:

```text
external Intake receipt completed
          ↓
enabled on-ingestion Libra recipes receive one durable job
          ↓
event assembly → Apollo → every configured Vault output
          ↓
optional local notification
```

Local report generation is not permission to perform unrelated external
actions. Calendar, email, external notifications, and data deletion remain
separate capabilities and must not be smuggled into a report definition.

## Catch-up and stale instances

The current daily Situation Brief declaration has historical due instances in
`awaiting_pipeline`. Kernel 28 must not blindly emit a burst of old reports.
Panopticon should show the backlog and apply the configured policy:

- `skip`: record why the instance was skipped;
- `run_once`: create one current catch-up report for the configured window;
- `reconcile`: assemble missing evidence and reconcile one durable report per
  unique reporting window; or
- `needs_you`: wait for a person to decide.

Identical windows and source sets must deduplicate. A changed source set must
produce a visible revision rather than silently overwrite a report.

## Acceptance criteria and implementation boundary

Kernel 28 passes its bounded implementation contract when:

1. A person can create and configure a Libra constellation and report
   definitions through Panopticon without editing JSON or Go source.
2. Zodiac schedules those definitions through typed links and preserves the
   existing schedule/instance authority boundary.
3. A configured daily report completes assembly → Apollo → Vault verification
   → optional notification with a durable receipt chain.
4. A configured weekly report aggregates its declared window and completes the
   same receipt chain.
5. Catch-up, duplicate, missing-source, paused, and failed-Apollo states are
   visible and deterministic.
6. A recipe can select source families, sections, report-only instructions,
   and multiple Vault outputs without editing Go or JSON.
7. A verified external Intake conversion can create one replay-safe event run
   for each enabled on-ingestion recipe.
8. No report definition can execute arbitrary shell commands or create a
   calendar/email action implicitly.

9. The existing hardcoded Situation Brief default is explicitly
   marked legacy; it is not silently duplicated.

Future refinements may add custom filename templates, a richer dry-run
preview, source labels/people selectors, and non-local delivery adapters. They
are not required to cross the Kernel 28 completion boundary.

## Non-goals

Kernel 28 does not implement Archimedes, Zeus sanitation, arbitrary user
scripting, provider installation, or a universal workflow engine. Constellations
organize declared Pantheon work; they do not become a second Bifrost, Future,
or Apollo. An on-ingestion recipe is limited to the verified Intake boundary;
it is not a general event bus.
