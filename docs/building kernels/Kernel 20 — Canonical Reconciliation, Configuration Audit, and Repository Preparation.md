# Kernel 20 — Canonical Reconciliation, Configuration Audit, and Repository Preparation

## Mission

Prepare Pantheon for its first serious GUI by making the repository, documentation, configuration surface, and operator-facing system coherent enough that Kernel 21 can build against a trustworthy contract.

Kernel 20 is not a GUI kernel.

It is the **reconciliation and preparation kernel** that answers:

> What does Pantheon actually contain now, what can Grant configure or inspect, where is that truth recorded, and what must be cleaned up before a GUI makes those structures permanent?

The repository has accumulated architecture, daemons, commands, configuration, procedures, tests, compatibility paths, and documentation over many kernels. This kernel should bring them back into alignment.

Do not add major new Pantheon capabilities merely because cleanup exposes opportunities.

The goal is to make the organism **legible, canonical, and ready to surface.**

---

# 1. Establish the Canonical Current System

Audit the repository as implemented through Kernel 19.

Do not rely solely on roadmap prose, old kernel documents, comments, or assumptions.

Determine the current implemented state of:

- Pantheon core
- daemons
- reasoning components
- deterministic components
- Intake
- Weave
- Present
- Past
- Future
- Mercury
- Prometheus
- Father Time
- Rumor
- Bifrost
- Zeus, if currently implemented
- model/provider selection
- notification system
- scheduler/timers/cron surfaces
- CLI
- configuration files
- permissions
- approvals
- external integrations
- Vault integration
- logging
- receipts/audit state
- runtime/service management
- emergency controls

Produce one canonical system inventory from **actual repository evidence**.

Where implementation and documentation disagree, record the disagreement before fixing it.

---

# 2. Configuration Surface Audit

Identify every meaningful operator-facing configuration surface currently available or implied by the codebase.

Include at minimum:

- local reasoning model
- frontier/provider settings
- model endpoints
- daemon enable/disable state
- schedules
- notification level
- notification destinations
- Prometheus controls
- Bifrost capabilities
- plugin permissions
- update permissions
- Intake sources
- audio/Rumor settings
- Whisper model/path/settings
- retention settings
- Father Time/chronology settings
- Vault paths
- email/account integrations
- privacy/redaction settings
- Archimedes-related placeholders if already present
- logging/debug levels
- network access
- any developer-only toggles that accidentally became operational dependencies

For each setting determine:

- canonical name
- purpose
- current value
- default
- valid values/range
- persistence location
- whether restart is required
- whether changing it is safe at runtime
- whether Grant should ever need to see it
- whether it belongs in the future GUI
- whether it is informational only
- whether changing it requires approval
- whether it is obsolete or duplicated

Do not expose every internal constant simply because it exists.

The future GUI should expose **meaningful control**, not implementation debris.

---

# 3. Operator Surface Audit

Inventory everything Grant can currently do through Pantheon.

Include:

- CLI commands
- status commands
- show commands
- approvals
- pause/resume
- Intake inspection
- notification controls
- Bifrost controls
- Rumor controls
- model/provider controls
- any scheduler commands
- emergency commands
- daemon-specific commands

For each command record:

- command
- purpose
- authority level
- required permissions
- whether it mutates state
- whether it should eventually have a GUI equivalent
- whether it is redundant with another command
- whether it is internal/developer-only and should remain hidden

Preserve:

`pantheon prometheus protocol --pause`

as a first-class emergency operator surface.

---

# 4. Status and Observability Audit

The GUI will need trustworthy state.

Determine what Pantheon can currently report about:

- daemon health
- service/runtime health
- active models
- pending work
- Intake backlog
- current/last Rumor recording or transcription
- Bifrost capabilities
- plugin health
- authentication failures
- notification health
- scheduled work
- recent failures
- Prometheus state
- last successful activity per subsystem
- storage/retention pressure where relevant

Identify missing status signals that Kernel 21 would otherwise be forced to infer indirectly.

Add **small deterministic status surfaces** where necessary.

Do not build dashboards yet.

---

# 5. Documentation Reconciliation

Audit the documentation tree.

Identify:

- canonical architecture documents
- roadmap
- Constitution/value documents
- kernel records
- operator procedures
- installation/onboarding documentation
- daemon/component documentation
- security/privacy documentation
- integration documentation
- obsolete drafts
- duplicate explanations
- files whose names no longer match their contents
- documents that contradict current implementation

Reconcile documentation to actual current state.

Do not erase useful historical kernel records merely because the architecture evolved.

Separate:

- **current canonical documentation**
- **historical implementation records**
- **procedures**
- **design/roadmap**
- **generated reports**
- **deprecated material**

A reader should be able to tell which documents describe **Pantheon now** versus how Pantheon got here.

---

# 6. Repository Organization

Inspect the repository structure and reorganize where beneficial.

The goal is not aesthetic churn.

Move files only when doing so makes ownership and architecture clearer.

Establish or improve sensible homes for areas such as:

- commands/CLI
- core contracts/types
- daemons
- reasoning components
- deterministic services
- integrations/adapters
- Bifrost
- configuration
- schemas
- migrations
- Intake
- Weave
- notifications
- scheduler
- security/Prometheus
- tests
- docs
- scripts/tools
- generated/runtime artifacts

Do not create a directory hierarchy so deep that ordinary work becomes annoying.

Remove or quarantine:

- abandoned experiments
- dead compatibility shims
- duplicate implementations
- obsolete generated artifacts committed by mistake
- stale scripts no longer used
- unused configuration templates

Only remove code after proving it is unused or superseded.

---

# 7. Naming Reconciliation

Audit names used for the same concept across:

- code
- CLI
- config
- docs
- logs
- schemas
- database fields
- tests

Resolve needless inconsistencies.

Examples of problems to detect:

- one daemon called three different names
- `intake_path` in one location and `inbox_dir` elsewhere for the same thing
- obsolete Ra-era naming where Apollo is now canonical
- Hermes/Terra terminology surviving where Bifrost/Pantheon has replaced the role
- stale provider/model names
- old command aliases that no longer represent the architecture

Do not rename stable public interfaces casually.

Where compatibility aliases remain necessary, make the canonical form explicit.

---

# 8. Configuration Canonicalization

Pantheon should not require the future GUI to edit arbitrary source files.

Determine the canonical configuration mechanism going forward.

Where configuration is currently scattered, centralize or provide a deterministic abstraction over it.

Requirements:

- typed configuration
- validation
- explicit defaults
- safe handling of missing values
- readable current-state inspection
- no silent fallback to dangerous behavior
- secrets kept separate from ordinary configuration
- configuration changes produce useful receipts where appropriate

Avoid building an elaborate configuration framework if a simple existing structure is sufficient.

The GUI should later interact with a stable configuration API/contract rather than knowing filesystem implementation details.

---

# 9. Secrets Audit

Locate how secrets are currently handled.

Inspect for accidental storage of:

- passwords
- API keys
- OAuth refresh/access tokens
- auth headers
- mail credentials
- provider credentials

Confirm that secrets are not stored in:

- normal config dumps
- logs
- kernel reports
- Vault notes
- status output
- Bifrost registry descriptions

If historical leakage exists in repository history or working files, report it clearly.

Do not rotate credentials without Grant’s authorization unless an immediate existing security contract already mandates doing so.

---

# 10. Runtime and Service Inventory

Determine what actually runs automatically.

Inventory:

- systemd user services
- timers
- cron jobs
- Pantheon internal scheduling
- Hermes scheduling still remaining
- background processes
- local servers
- model servers
- Whisper/Rumor workers
- Bifrost adapters
- notification services

Document ownership:

> Which component starts this?
> Who stops it?
> What happens after reboot?
> What happens after failure?
> Who reports failure?

Do not leave Kernel 21 guessing which process owns which state.

---

# 11. Hermes Residual Audit

After Kernel 19, inspect what Hermes still does.

Classify every remaining Hermes dependency as:

- still intentionally used through Bifrost
- temporarily retained for rollback
- superseded
- safe to disable
- candidate for later removal

Do not uninstall Hermes automatically.

If Pantheon has proven a replacement for a scheduled Hermes function, pause or retire the old automation according to the approved migration contract.

Document what remains and why.

Terra should not remain architecturally necessary merely because old automation still references it.

---

# 12. Bifrost Reconciliation

Inspect Kernel 19’s implementation against its contract.

Ensure the repository clearly exposes:

- capability registry
- capability status
- permissions
- approval state
- update policy
- Prometheus relationship
- Mercury invocation path
- failure states
- receipts
- email integration
- Hermes migration state

Kernel 20 may fix inconsistencies, documentation gaps, and small status/config issues.

Do not expand Bifrost into additional major integrations during this kernel.

---

# 13. Rumor Reconciliation

Ensure Rumor’s current contract is accurately documented after Kernel 18.

Capture:

- supported inputs
- microphone capture
- imported audio
- transcription path
- Whisper model
- storage locations
- segmentation behavior
- silence handling if implemented
- processing state
- failure behavior
- retention behavior
- current live-transcription status

Explicitly document whether live transcription is implemented, experimental, or deferred.

Do not add Project Fame behavior to Rumor.

Rumor records/transcribes. Human-context interpretation belongs to later work.

---

# 14. Model and Provider Inventory

Identify every reasoning/inference engine Pantheon currently knows about.

For each record:

- role
- provider
- model
- endpoint
- local/remote
- privacy implications
- fallback behavior
- where configured
- current availability
- whether operator-selectable

Keep the architectural distinction between:

- Mercury as deterministic mechanism
- Apollo as reasoning intelligence
- future Archimedes as frontier model path

Do not resurrect Terra as an architectural requirement.

---

# 15. Approval and Permission Inventory

Audit all actions currently requiring explicit Grant approval.

Create a canonical permission/approval inventory.

Include:

- Bifrost download
- Bifrost install
- Bifrost registration
- Bifrost first use
- automatic updates
- email destructive actions
- external/network access where governed
- future frontier-model disclosure boundaries
- configuration changes carrying meaningful risk

Identify cases where:

- approval exists only in prose
- code does not enforce the documented rule
- duplicate approval systems exist
- approval state cannot currently be inspected

Repair small enforcement gaps where clearly within established policy.

Escalate architectural contradictions rather than inventing policy.

---

# 16. GUI Readiness Contract

Produce a durable artifact describing what Kernel 21 may safely build against.

Call it something clear, for example:

`docs/gui/GUI-CONTRACT.md`

or the repository’s equivalent canonical documentation location.

It should identify the main GUI domains, likely including:

### Home / Organism
- overall status
- health
- pending work
- important failures
- current model state

### Pantheon
- gods/daemons
- roles
- running/stopped/degraded
- recent activity

### Prometheus
- emergency pause
- authority state
- blocked operations
- approvals

### Intake
- incoming items
- source
- state
- warnings

### Rumor
- mic/recording
- transcription queue/status
- imported audio
- retention

### Bifrost
- capabilities/plugins
- permissions
- approvals
- update policy
- health

### Models
- Apollo/local reasoning
- future Archimedes/frontier state
- provider configuration

### Notifications
- level
- destinations
- recent alerts
- persistent permission reminders

### Configuration
- safe operator-configurable settings

### System
- services
- logs
- diagnostics
- version/build information

This is an audit output, **not a mandatory visual layout**.

Kernel 21 should remain free to design the actual experience.

---

# 17. Human-Language Labels

The GUI is for Grant, not merely developers.

For every GUI-worthy setting/status item, provide:

- technical identifier
- concise human-readable label
- short explanation
- possible values/actions
- danger level if applicable

Example:

Technical:

`bifrost.capabilities.gmail.update_policy`

Human:

**Automatic updates — Gmail connector**

> Allows this approved connector to download and install its own approved updates without asking each time. Pantheon will periodically remind you this permission remains active.

Do not force Kernel 21 to invent user-facing semantics from field names.

---

# 18. Danger and Authority Classification

Assign meaningful operator actions an approximate safety class.

For example:

### Informational
Read-only.

### Routine
Low-risk reversible configuration.

### Significant
Meaningful system behavior change.

### Restricted
Requires explicit Grant authorization.

### Emergency
Prometheus-level control.

Do not overengineer this taxonomy if an equivalent already exists.

The goal is for Kernel 21 to know which controls can be ordinary toggles and which require unmistakable confirmation/approval flows.

---

# 19. Remove False Complexity

Look for abstractions that no longer earn their keep.

Especially inspect:

- legacy Hermes glue
- old agent/persona pathways
- duplicated model wrappers
- duplicated configuration readers
- duplicated status representations
- multiple notification paths
- scripts replaced by daemon functionality
- temporary kernel scaffolding that became permanent by accident

Prefer one clear implementation over three compatibility layers where safe.

Do not perform broad rewrites simply because a newer pattern looks prettier.

---

# 20. Preserve History

Kernel 20 is allowed to reorganize documentation and repository structure, but must preserve historical traceability.

Kernel reports and major architectural decisions should remain discoverable.

When relocating historical files:

- preserve git history naturally where possible
- use clear archival directories
- do not rewrite historical documents to pretend they always described the current architecture

Current truth and historical truth are different artifacts.

Preserve both.

---

# 21. Roadmap Reconciliation

Update the roadmap to reflect the actual current sequence:

- Kernel 17 — Whole Organism
- Kernel 18 — Rumor
- Kernel 19 — Bifrost / plugins
- Kernel 20 — Canonical Reconciliation, Configuration Audit, and Repository Preparation
- Kernel 21 — GUI

Then preserve the next horizon:

- Aphrodite
  - first major project: **Project Fame**
  - likely later project: **Project Purity**
- Archimedes
- post-build organism reconciliation / missing-capability pass
- feature-development era
- increasing Aphrodite-generated improvement proposals, always subject to Grant’s authority

Do not prematurely assign exact kernel numbers to later work unless the existing roadmap already does.

---

# 22. Aphrodite Preparation — Documentation Only

Do not build Aphrodite.

However, ensure the current architecture produces enough inspectable state that she will eventually be able to take system readings.

Future Aphrodite should be able to inspect relevant:

- component health
- configuration
- Bifrost capability inventory
- failures
- performance metrics
- receipts
- system changes
- known limitations

Her eventual role is project-oriented.

Project Fame will investigate/design human-context interpretation of transcript material and may produce a new daemon/deity.

Project Purity is expected to investigate how Pantheon monitors itself and proposes improvements to Grant in progressively concrete human terms:

**concept pitch → design/architecture pitch → implementation pitch → code/diff**

Kernel 20 should not implement those systems.

It should merely avoid hiding the state they will eventually need.

---

# 23. Archimedes Preparation — Documentation Only

Do not build Archimedes.

Ensure current documentation still preserves the frontier-model privacy contract and distinguishes future Archimedes from Apollo.

The future GUI will eventually need a clear way to show:

- whether frontier access is enabled
- what model/provider is being used
- what data may leave the machine
- sanitization/privacy status
- failures/escalations

Do not implement the frontier path here unless already required by a prior kernel.

---

# 24. Tests

Add or update tests for reconciliation work where behavior changes.

Especially protect:

- config parsing/validation
- canonical defaults
- status serialization
- permissions
- Prometheus pause
- Bifrost capability state
- daemon inventory/state
- secrets not leaking into status
- deprecated aliases if retained
- repository migration assumptions where relevant

Do not manufacture tests solely to increase counts.

---

# 25. Verification

Before PASS, perform an explicit repository-wide verification.

Prove:

### Documentation
- Canonical current architecture exists.
- Roadmap reflects actual current state.
- Historical docs remain discoverable.
- Contradictory canonical docs have been resolved or explicitly flagged.

### Configuration
- Every operator-worthy setting discovered during the audit appears in the configuration inventory.
- Duplicate/obsolete settings are identified.
- Secrets are not exposed through normal config inspection.

### CLI
- Current commands are inventoried.
- GUI-worthy commands are identified.
- emergency Prometheus control remains functional.

### Runtime
- Automatically running services/jobs are inventoried.
- ownership and restart behavior are documented.

### Components
- each implemented daemon/component has a current role/status description.
- stale names and obsolete architectural descriptions are reconciled.

### Bifrost
- Kernel 19 surfaces remain coherent and documented.

### Rumor
- current actual behavior is documented, including what remains deferred.

### Hermes
- remaining dependencies are explicitly classified.

### GUI readiness
- a stable GUI contract exists.
- Kernel 21 will not need to discover the basic configuration model by scraping arbitrary source code.

---

# 26. Deliverables

At minimum produce or update canonical artifacts equivalent to:

- Current Architecture
- Configuration Inventory
- Operator/CLI Inventory
- Runtime/Service Inventory
- Permission & Approval Inventory
- GUI Readiness Contract
- Updated Roadmap
- Reconciled repository documentation/index
- Kernel 20 reportback

Use names and locations appropriate to the existing repository rather than blindly creating duplicates.

---

# 27. Reportback

Report:

- files added
- files moved
- files removed
- files materially rewritten
- repository structure changes
- canonical docs chosen
- configuration surfaces discovered
- duplicate/obsolete settings found
- operator commands inventoried
- runtime jobs/services inventoried
- Hermes dependencies remaining
- Bifrost status
- Rumor status
- secrets/security findings
- inconsistencies discovered
- inconsistencies corrected
- inconsistencies deliberately deferred
- GUI domains identified
- tests run
- live verification performed
- known limitations
- clean predecessor commit/hash for Kernel 21

If substantial unresolved architectural contradictions remain, do **not** report PASS merely because tests are green.

---

# 28. Completion Standard

Kernel 20 is complete when Pantheon’s repository can answer, from canonical current documentation and deterministic inspection:

> What is Pantheon?

> What is running?

> What can it do?

> What can Grant configure?

> What requires Grant’s approval?

> What external capabilities are connected?

> What is healthy or broken?

> Where does the truth live?

> What should the GUI expose?

A developer beginning Kernel 21 should not need to archaeology-dive through twenty kernels to discover those answers.

The repository should be clean enough that the GUI can make Pantheon visible **without accidentally fossilizing old mistakes.**

**Do not begin Kernel 21.**