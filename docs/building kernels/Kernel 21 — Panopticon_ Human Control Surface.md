# Kernel 21 — Panopticon: Human Control Surface

## Mission

Build **Panopticon**, Pantheon’s primary human-facing control surface.

Panopticon is where the human operator sees the organism, understands what it is doing, receives its notifications, inspects its occupants, changes ordinary configuration, gives approvals, and reaches emergency controls.

It is not merely a graphical version of the CLI.

The CLI remains the serious audit/debugging surface.

Panopticon should answer, immediately and in human terms:

> **What is happening?**

> **Is everything healthy?**

> **What needs me?**

> **Is anything dangerous?**

The human is the operator.

The named intelligences and daemons are the subjects being watched.

Prometheus is the guard.

Father Time is something stranger.

---

# 1. Build Against Kernel 20 Truth

Kernel 20 created the canonical current-state layer for this work.

Treat these as authoritative:

- `docs/current/system-inventory.md`
- `docs/current/configuration-inventory.md`
- `docs/current/operator-surface.md`
- `docs/current/runtime-services.md`
- `docs/current/permissions-and-approvals.md`
- `docs/gui/GUI-CONTRACT.md`

Also preserve the current architecture described by:

- `docs/pantheon-architecture-brief.md`
- `docs/configuration.md`
- `docs/operator-guide.md`

Do not rediscover Pantheon by scraping arbitrary implementation files when Kernel 20 already established a canonical contract.

If implementation and `GUI-CONTRACT.md` disagree, stop that surface from becoming a destructive GUI control until the discrepancy is understood.

Do not invent a second authority system inside the frontend.

---

# 2. Delivery Model

Panopticon is a **local browser application served directly by the Pantheon daemon**.

Initial operator experience should be approximately:

```text
systemd starts Pantheon
        ↓
Pantheon Go daemon
        ↓
local API + embedded Panopticon frontend
        ↓
http://127.0.0.1:8765/
```

Requirements:

- no Docker requirement for the GUI
- no separate web server
- no separate frontend daemon the operator must babysit
- frontend assets should be bundled/embedded with Pantheon for normal installation
- use the existing localhost boundary
- do not open network listening beyond the existing approved local boundary merely to support the GUI

Where practical, provide a Linux `.desktop` launcher so Grant can click **Pantheon** or **Panopticon** from the application launcher and have the local GUI opened in the default browser.

Do not build Electron/Tauri/native packaging in this kernel.

The browser implementation should remain structurally suitable for a future native wrapper if desired.

---

# 3. Product Principle: Show the Organism Before the Settings

Do not open Panopticon onto configuration forms.

The default view is the living system.

The information hierarchy is:

1. **urgent notification / attention**
2. **Radio Prometheus**
3. **current work / today**
4. **organism health**
5. **cells / systems**
6. **toolkits and libraries**
7. **configuration**

The GUI should feel like entering a control room, not opening an admin panel.

Never begin the human experience with a wall of fields such as:

```text
INGEST_MODE
OLLAMA_URL
poll_seconds
runtime.listen
```

Technical identifiers remain available where useful, but behind human-readable surfaces.

---

# 4. Visual Language: The Panopticon

The visual metaphor is a **Panopticon control center**.

The interface should communicate that the operator is observing and controlling multiple powerful systems from one central point.

This is not a generic SaaS dashboard with deity names pasted onto cards.

Use architectural/control-room cues where they improve comprehension:

- cells/panes
- observation windows
- status lights
- control surfaces
- restrained industrial structure
- clear occupancy labels
- visible operational state
- strong hierarchy around authority controls

Avoid turning it into:

- fantasy temple art
- marble-column Greek cosplay
- cyberpunk clutter
- hacker-terminal cosplay
- a novelty prison game

The metaphor serves operation.

---

# 5. Home Screen

The Home screen is the primary Panopticon room.

It should contain, in roughly this hierarchy:

## A. Notification Banner

A persistent top-area notification/attention region.

It should clearly surface:

- important notifications
- failures needing attention
- pending approvals
- blocked work
- authentication problems
- significant Bifrost warnings
- other events already routed through Pantheon’s notification system

Do not create a second notification engine.

Panopticon is the **canonical readable notification history**; external channels such as desktop/ntfy are delivery mechanisms.

If there is nothing requiring attention, the banner should become quiet rather than continually announcing health.

---

## B. Radio Prometheus

Prometheus is **not a prisoner**.

Do not present a normal cell labelled `Prometheus`.

The central control should be called:

# **RADIO PROMETHEUS**

It should be visually unmistakable.

The Home screen must include a large, deliberately dramatic red emergency control.

It should produce the feeling:

> “Why is there such a serious emergency switch here?”

Because Pantheon contains powerful reasoning and execution systems, and the operator retains direct authority over them.

The control must map to the actual Prometheus protocol.

Support at minimum:

- status
- pause
- resume
- hard stop where the current operator contract safely exposes it

The emergency interaction must explain consequences in plain language before/while acting.

Example conceptual language:

> **PAUSE PANTHEON**
>
> Stops new autonomous activity so you can inspect the system. Work skipped during the pause is not automatically replayed.

Do not bury this control behind menus.

Do not require Apollo, Mercury, Future, Bifrost, or any model to operate it.

Do not fake the switch with client-only state.

The visual design may be dramatic. The behavior must remain boringly deterministic.

---

# 6. Return-to-Home Safety Cue

Prometheus does not need to remain permanently visible on every page.

However, when the operator leaves Home, the navigation affordance back to Home should make the emergency location obvious.

Something conceptually like:

```text
← Back to Panopticon
  Emergency control on Home
```

Do not clutter every page with a giant red button.

The operator should nevertheless never wonder how to get back to the emergency control.

---

# 7. Father Time

Father Time does **not** receive an ordinary cell.

He is neither presented as an ordinary prisoner nor as Radio Prometheus.

Father Time appears through a **watch-face interface built into Panopticon**.

The watch should be visually present enough to feel architectural rather than decorative.

It should communicate the joke/truth:

> Everyone here is a prisoner of Father Time.

The watch may expose useful temporal information such as:

- current local time
- configured timezone
- current chronology status
- next known scheduled operation where the backend can truthfully provide it
- temporally blocked work
- Father Time status

Clicking or opening the watch may expose deeper temporal information if current APIs support it.

Do not invent scheduler behavior Father Time does not own.

Do not present Father Time as a cell occupant.

---

# 8. Human-Task Navigation

Primary navigation should be organized by **what the human is trying to understand or do**, not by requiring knowledge of Pantheon mythology.

Use a hybrid model where deity/daemon identities remain visible inside relevant domains.

Likely major navigation domains include:

- Home / Current Work
- Ingestion / Understanding
- Execution / Capabilities
- Approvals
- Notifications
- Intake
- Rumor
- Bifrost
- Models / Reasoning
- Configuration
- System

Exact grouping may adapt to Kernel 20’s audited surfaces.

Do not create navigation categories simply to give every named component equal screen time.

---

# 9. Cells and Occupants

Named Pantheon components should appear as occupants/groups where useful.

A cell/pane should normally communicate:

- occupant/group name
- role in human language
- health
- current operation
- waiting/idle/busy/blocked state
- last meaningful activity
- relevant warning
- controls appropriate to that component
- deeper inspection link

Do not require one cell per daemon.

Some components make more sense as groups.

The interface must support:

```text
domain
  → cell/group
      → occupants
      → tools
      → current work
```

rather than hardcoding:

```text
one deity = one top-level card
```

---

# 10. The Fates

Treat the Fates as one grouped operational surface.

The Fates are daemon roles, not three independent deity products needing separate dashboard real estate.

Their surface may contain:

- Present
- Past
- Future

and should expose the Fates’ relevant operational state collectively and individually where useful.

**Weave is a tool used by the Fates. It is not an occupant.**

Past’s work should be accessible through the Fates surface rather than demanding a separate top-level Past card.

Place the Fates where the human workflow makes sense.

Current design intent places them naturally near Apollo within the broader ingestion/understanding flow.

Do not let mythology override the canonical architecture. The GUI may group a process for comprehension without falsifying its actual responsibility.

---

# 11. Apollo

Apollo should have a strong visible presence because he is Pantheon’s reasoning intelligence.

His surface should show, when available:

- health/model availability
- current task
- queued/waiting work
- most recent completed meaningful work
- reports/findings
- source/citation state
- relevant errors or blocked conditions

Apollo should write durable material according to the existing architecture; Panopticon does not become a replacement storage layer.

## Streaming Output

Provide an optional **streaming-response view** when supported by the underlying model/runtime.

This means ordinary token/text streaming of the response as it is generated.

It does **not** mean exposing hidden chain-of-thought.

Default:

**Streaming display OFF**

When enabled, the operator may watch generated output arrive progressively in a small live window rather than waiting for the full response.

If the backend/runtime does not currently support streamed output cleanly, expose the setting as unavailable/deferred rather than implementing an unsafe fake.

The off state should remain computationally and visually quiet.

---

# 12. Current Work — Main Home Surface

Most of the Home screen should be dedicated to **current work**.

Include meaningful items such as:

- actively running work
- queued work
- work waiting on Grant
- pending approvals
- blocked work
- recently completed meaningful work
- scheduled work coming soon when truthfully known
- escalations
- recent automatic ingestion such as transcript/email intake
- relevant failures

Do not flood this surface with low-level audit receipts.

An item should answer:

- what is happening?
- who/what owns it?
- what state is it in?
- does Grant need to do anything?
- when did it begin/change?
- where can Grant inspect it?

The goal is to see the shape of the organism’s day.

---

# 13. Live Status

Panopticon should monitor active state with **seconds-scale freshness**, not minutes.

Use efficient polling, server push, streaming, or another appropriate mechanism based on the existing backend.

Do not create unnecessary high-frequency load.

The requirement is perceptual:

> If something meaningful changes, the operator should normally see that change within seconds.

Examples:

- Rumor begins/finishes transcription
- Apollo starts/finishes work
- Bifrost changes health
- an approval becomes required
- Intake receives new material
- Prometheus enters pause
- a component becomes degraded

Use the Kernel 20 system inventory and deeper read surfaces as the source of truth.

Never manufacture “activity” animations disconnected from real state.

---

# 14. Human First, Technical Detail on Demand

Every operator-facing control or field should first communicate:

1. what it means
2. what changing it will do
3. what consequence the operator should expect

Technical data may be visible through:

- expandable details
- secondary text
- tooltips where inexpensive
- advanced sections
- copied identifiers
- links to CLI documentation

Do not require elaborate hover behavior if it substantially complicates the frontend.

Prefer visible concise explanations over hidden mystery.

Example:

Instead of:

> `rumor.poll_seconds`

prefer:

> **How often Rumor checks for new recordings**
>
> Currently: every 5 seconds
>
> Advanced: `rumor.poll_seconds`

Do not become precious with metaphorical labels when ordinary English is clearer.

---

# 15. Edit → Apply Configuration Model

Configuration changes should use:

**Edit → review state → Apply**

Do not instantly commit every toggle as soon as it is touched.

The operator should be able to change one or several related fields and then explicitly Apply.

Requirements:

- show current value
- show edited value
- validate before persistence
- explain restart requirement
- explain meaningful consequences
- preserve the current typed configuration contract
- keep secrets out of ordinary forms
- do not confuse configuration availability with execution authority

If a setting cannot yet safely be changed through an existing backend contract, render it read-only rather than writing directly to arbitrary files from JavaScript.

Kernel 21 may add small, deterministic configuration endpoints needed to support audited GUI-safe settings.

Do not make the frontend edit `pantheon.json` directly.

---

# 16. Consequence-Based Confirmation

For significant controls, confirmations should explain **what will happen**, not merely ask:

> Are you sure?

Examples:

> **Disable Gmail capability?**
>
> Pantheon will stop receiving new Gmail through Bifrost until this capability is re-enabled. Existing Intake records will remain.

or:

> **Pause Rumor?**
>
> Rumor will stop accepting new transcription work. Existing raw recordings will not be deleted.

Use the authority classification from Kernel 20.

Routine reversible edits should not be burdened with theatrical confirmation.

Restricted/emergency actions must be unmistakable.

---

# 17. Approvals

Create a centralized **Approvals** surface.

An approval may also appear contextually in the component that requested it.

Examples:

- Bifrost capability approval
- capability update approval
- significant configuration approval
- work awaiting Grant
- future external/restricted operations

Each approval should explain:

- who/what is requesting it
- what is requested
- why
- what capability/resource is involved
- what happens if approved
- what happens if denied/left pending
- whether permission persists
- how to revoke persistent permission

Do not collapse separate Bifrost states such as download, installation, registration, use, and updates into one vague “Allow plugin” checkbox.

Presence is not permission.

Research is not permission.

---

# 18. Notifications

Panopticon becomes the canonical human-readable notification history.

Provide a surface for:

- unread/current notifications
- acknowledged/recent notifications
- severity/attention state
- source component
- timestamp
- related work/control link

Do not reproduce raw audit logs.

Where the backend distinguishes notification types or escalation levels, represent them meaningfully.

The Home banner should use this same source.

---

# 19. Intake

Keep Intake simple.

Grant expects to manually place many items into Intake himself, while recurring paths such as email and transcripts may add material automatically.

Panopticon should show enough to answer:

- what came in?
- when?
- from where?
- what type?
- current ingestion state?
- warning/blocked attachment?
- has it advanced?

Do not build a sophisticated mail-client/file-manager/inbox-management product.

Useful views may include:

- recent Intake items
- basic state
- source
- timestamp
- warnings
- open/inspect when safe

Leave deep inspection/audit to existing deterministic/CLI surfaces when appropriate.

---

# 20. Rumor

Rumor receives a focused human-facing surface.

Provide, where the current backend supports it:

- microphone/input status
- recording status
- start/stop capture if Kernel 21 can safely connect the already-proven local audio mechanism
- import audio
- transcription queue
- recent transcripts
- source status
- retry/ignore actions
- storage/retention summary
- current Whisper model name/path as secondary detail

Do not build a Whisper configuration laboratory inside Panopticon.

If Grant needs to develop/debug Whisper itself, that remains work in the Whisper repository/tooling.

Do not add Project Fame.

Do not add transcript interpretation.

Do not imply live transcription exists if it remains deferred.

---

# 21. Bifrost

Represent Bifrost primarily as **capability cards**, not a registry table.

Each capability card should communicate:

- name
- human-readable purpose
- type
- source
- version
- health
- enabled/disabled
- network use
- authentication requirement
- approved operations
- prohibited operations
- approval state
- update policy
- recent invocation status
- warning/failure state

The detail view may expose technical registry fields.

Support existing safe controls such as enable/disable and approvals according to the current API/authority contract.

Do not implement a plugin marketplace.

Do not implement autonomous software discovery/download.

Do not pretend Kernel 19 built a generic installer if it did not.

The current Gmail/Himalaya crossing should be visibly distinguishable from hypothetical future capabilities.

---

# 22. Hermes Visibility

Hermes currently remains part of the rollback/external gateway path.

Do not present Hermes/Terra as a core Pantheon deity/daemon simply because compatibility remains.

If useful, surface Hermes under:

- external services
- Bifrost/external dependency status
- runtime compatibility/rollback

Clearly distinguish:

**core Pantheon**
from
**retained external tooling**

Do not remove Hermes in Kernel 21 unless an already-approved cutover process has independently proven it safe.

---

# 23. Models / Reasoning

Provide a human-readable reasoning/model area.

At minimum show current truth about:

- Apollo model
- provider/runtime endpoint in advanced detail
- local/remote status
- availability
- current configured state
- future Archimedes state as unavailable/deferred if appropriate

Do not expose an empty Archimedes configuration as though frontier reasoning is active.

The operator should understand **where reasoning occurs** before seeing model IDs.

---

# 24. System Cells and Deferred Occupants

Kernel 20 inventories implemented and deferred components.

Panopticon must visually distinguish:

- active
- inactive
- degraded
- configured but unavailable
- deferred/not built

Do not show Orion, Zeus, Archimedes, Aphrodite, Nostalgia, or another horizon concept as if it is already operating.

It is acceptable to omit distant future systems from the normal Home view entirely.

If a deferred component appears for architectural orientation, label it unmistakably as future/unavailable.

---

# 25. Libraries and Toolkits

Secondary navigation may expose lower-priority tools/libraries along the sides or bottom of the Home experience.

These should not compete with Current Work.

Possible examples, depending on audited implementation:

- Bifrost capabilities
- package/plugin vocabulary
- model settings
- Vault/Weave information
- configuration
- documentation/help
- diagnostics

Use the actual Kernel 20 inventories rather than creating decorative empty libraries.

---

# 26. Recent Activity, Not Audit Theater

Each major cell/system may show a small recent-activity list.

Example:

```text
14:03  4 emails entered Intake
13:58  Gmail capability healthy
13:21  Rumor transcript completed
12:58  Apollo report completed
12:55  New instruction entered Present
```

Keep this bounded.

Do not attempt to reproduce every audit receipt in Panopticon.

If someone needs forensic inspection, the CLI and canonical records remain available.

---

# 27. No Raw Chain-of-Thought Surface

Do not build a feature that exposes hidden model chain-of-thought.

When showing live Apollo behavior, use one of:

- streamed final-response generation
- explicit tool/activity events
- status messages
- source retrieval events
- bounded progress data already intentionally emitted by the system

Do not fabricate internal reasoning traces.

---

# 28. Visual State Language

Define a consistent state vocabulary across the GUI.

At minimum distinguish concepts such as:

- Healthy
- Working
- Waiting
- Needs You
- Paused
- Blocked
- Degraded
- Offline
- Deferred

Use iconography/text in addition to color.

Do not rely on red/green alone.

Danger red should be used sparingly enough that **Radio Prometheus remains visually exceptional**.

---

# 29. Accessibility and Basic Responsiveness

Build reasonable baseline accessibility:

- keyboard navigability
- visible focus
- semantic buttons/forms
- useful labels
- adequate contrast
- status not communicated by color alone
- reduced-motion respect where animation exists

Desktop Linux is the target for Kernel 21.

Basic responsive behavior is welcome where inexpensive, but do not turn this into the future mobile-control project.

No mobile biometrics.

No native-phone approval flows.

No remote-access architecture.

Future Grant can deal with that.

---

# 30. No Remote Exposure Yet

Panopticon remains localhost/local-machine-first.

Do not:

- bind Pantheon publicly
- add cloud relay
- add remote accounts
- add OAuth login solely for Panopticon
- punch firewall holes
- expose the control API to the LAN
- create an Internet-accessible Prometheus switch

Remote operation may be valuable later.

It is not necessary for the nightly-sync/daily-brief operating model and should not delay the first useful GUI.

---

# 31. Frontend Implementation

Use the simplest frontend architecture consistent with maintainability.

Prefer:

- frontend assets built as part of the repository
- static compiled output embedded/served by Go
- small dependency surface
- clear API client boundary
- no requirement for Node at runtime

If the repository already has an established suitable frontend convention, use it.

If not, avoid introducing a massive framework solely for this GUI.

Build tooling may use standard frontend tooling if needed, but the installed operator experience should not require running:

```text
npm run dev
```

for normal use.

Production Pantheon should simply serve Panopticon.

---

# 32. API Discipline

The browser must talk to stable Pantheon HTTP surfaces.

Use:

`GET /v1/system/inventory`

as the primary organism summary contract.

Use deeper existing routes defined by `docs/gui/GUI-CONTRACT.md` where appropriate.

Kernel 21 may add small GUI-oriented API endpoints when a real audited surface currently exists only through CLI/local deterministic logic.

Requirements for new endpoints:

- deterministic
- narrow
- versioned consistently
- authority-preserving
- secret-safe
- tested
- no model required merely to configure/inspect the GUI

Do not shell out from browser code to the Pantheon CLI.

Do not duplicate domain logic in JavaScript.

---

# 33. Prometheus Control Must Be Proven Independently

The GUI emergency control is security-sensitive.

Prove:

1. Panopticon displays real Prometheus state.
2. Pause from the GUI causes the actual daemon pause state.
3. New consequential activity is blocked according to the existing Prometheus contract.
4. Browser reload still shows paused state.
5. Resume from GUI works.
6. Resume does not automatically replay skipped work.
7. If hard-stop is exposed, it invokes the actual approved service-manager path and clearly warns what it will do.
8. Failure to contact Pantheon must never produce a fake “safe” status.

If the daemon is unreachable, say so.

Do not show green simply because the frontend cannot fetch state.

---

# 34. Father Time Watch Verification

Prove the watch:

- uses real configured/local temporal information
- does not claim Father Time schedules work he does not schedule
- displays the correct configured timezone
- survives refresh
- communicates temporal blocked state if such information is available

The watch is allowed to be visually distinctive.

It is not allowed to lie.

---

# 35. Edit → Apply Verification

For at least several representative GUI-safe configuration fields:

1. load current effective value
2. enter edit mode
3. change value without immediately persisting it
4. cancel and prove nothing changed
5. edit again
6. validate
7. Apply
8. prove the persisted/configured result
9. show restart requirement where applicable
10. reject invalid input safely

Do not use secrets for this proof.

---

# 36. Live-State Verification

Prove that Panopticon visibly reacts within seconds to several real backend changes.

Use representative operations such as:

- Prometheus pause/resume
- Rumor queue/status transition
- Bifrost enabled/disabled state
- new Intake occurrence
- pending approval
- Apollo active/completed state if practical

Do not fake state changes with frontend test fixtures for final live acceptance.

Fixtures are fine for unit/frontend tests.

---

# 37. Required Visual Proof

Kernel 21 is a GUI kernel.

Repository tests are not sufficient.

Capture visual evidence or conduct an explicit live operator walkthrough showing at minimum:

1. Panopticon Home
2. notification banner
3. Radio Prometheus and emergency control
4. Father Time watch face
5. current-work surface
6. representative cells/groups
7. Fates grouped surface
8. Apollo surface
9. Rumor surface
10. Bifrost capability cards
11. Approvals surface
12. configuration Edit → Apply interaction
13. paused Pantheon state
14. degraded/offline state representation
15. deferred/unbuilt component treatment if visible

The final product should be opened through the same path Grant will actually use.

---

# 38. Linux Launcher Proof

If practical, install or provide the `.desktop` launcher using the existing installation/deployment conventions.

Prove:

- application appears in the Linux application launcher
- clicking it opens the local Panopticon URL
- no terminal is required
- it does not launch a duplicate Pantheon daemon
- failure state is understandable if Pantheon is not running

Do not make launcher polish block the entire kernel if desktop-environment integration is unusually hostile, but report any limitation clearly.

---

# 39. Advertising/Personality Note

The intentionally alarming Prometheus control is part of Pantheon’s identity.

It is acceptable for future promotional material to play with the idea:

> “Yes, the big red AI emergency switch is there for a reason.”

Do not add legal/disclaimer advertising copy into the operational GUI unless it improves the operator experience.

Kernel 21 builds the authentic control surface first.

Marketing can quote the reality later.

---

# 40. Do Not Build

Do not begin:

- Aphrodite
- Project Fame
- Project Purity
- Archimedes
- Orion
- Zeus
- Nostalgia
- live transcript interpretation
- automatic voice-command execution
- remote access
- mobile approval application
- biometric approvals
- plugin marketplace
- autonomous plugin downloads
- a replacement for CLI forensic/audit tooling
- a new scheduler
- a Whisper development console
- raw hidden chain-of-thought viewing
- native Electron/Tauri packaging

Do not allow attractive GUI possibilities to consume the next five kernels.

This kernel makes the current organism operable and visible.

---

# 41. Tests

Add appropriate tests for:

- embedded/static Panopticon assets
- root/local GUI route
- system inventory rendering contract
- Prometheus UI control endpoints
- configuration read/validate/apply endpoints
- authority enforcement
- secret exclusion
- Bifrost enable/disable state
- approvals retrieval/action where implemented
- notification retrieval
- Rumor inspection/actions
- unavailable/degraded state behavior
- frontend API error handling
- localhost boundary
- no accidental public binding
- path traversal/static asset safety

Use frontend tests where useful, but do not make a giant frontend test framework a prerequisite for simple behavior.

---

# 42. Documentation

Update documentation to establish:

- Panopticon as Pantheon’s human-facing operator surface
- local URL
- launcher behavior
- relationship to CLI
- Radio Prometheus
- Father Time watch
- major navigation domains
- Edit → Apply configuration model
- notification history
- approvals
- cell/group metaphor
- live-state behavior
- current limitations
- remote/mobile deferral

Update `docs/gui/GUI-CONTRACT.md` only where Kernel 21 legitimately extends the stable contract.

Do not rewrite Kernel 20’s audit history.

---

# 43. Reportback

Report:

- PASS / PARTIAL / FAIL
- predecessor commit
- frontend architecture chosen
- build/runtime dependencies introduced
- how assets are served/embedded
- local URL
- launcher status
- pages/surfaces created
- APIs added
- APIs reused
- configuration controls implemented
- controls deliberately left read-only
- Radio Prometheus behavior
- Father Time watch behavior
- live-state refresh mechanism and observed latency
- Apollo streaming support status
- Intake behavior
- Rumor behavior
- Bifrost behavior
- Approvals behavior
- Notifications behavior
- Hermes representation/remaining dependency
- tests
- visual/browser proof
- screenshots or equivalent evidence
- accessibility checks
- security findings
- known limitations
- deferred surfaces
- files changed
- clean successor commit/hash

---

# 44. PASS Standard

Kernel 21 passes only if Grant can open Panopticon and usefully operate Pantheon without needing to translate the organism back into terminal jargon.

A PASS requires:

1. Panopticon is served locally by Pantheon without Docker or a second runtime server.
2. Home immediately communicates health, current work, attention needs, and safety state.
3. Radio Prometheus is prominent and actually controls Prometheus.
4. Father Time has a functional watch-face interface rather than a prisoner cell.
5. meaningful named occupants/groups are visible.
6. the Fates are represented coherently as a group.
7. current work updates within seconds where backend state permits.
8. notifications are visible through a canonical GUI surface.
9. approvals are visible and understandable.
10. Bifrost uses capability cards and preserves distinct approval/permission concepts.
11. Rumor has a simple useful operator surface without becoming a Whisper IDE.
12. Apollo has useful visible state; streaming generated output is optional/configurable and off by default.
13. representative GUI-safe configuration uses Edit → Apply.
14. dangerous actions explain consequences.
15. secrets remain hidden.
16. unavailable backend state is never rendered as false health.
17. CLI remains available and is not duplicated into a giant forensic GUI.
18. visual/live browser proof exists.
19. the GUI is understandable primarily through human language.
20. Kernel 21 does not quietly implement future daemons/projects.

The desired emotional result is:

> Grant opens the room and immediately understands that **Pantheon is here, it is doing things, he can see those things, and he can stop them.**

The desired engineering result is much less dramatic:

> A local, deterministic, inspectable frontend over the existing governed Pantheon contracts.

Both must be true.

**Do not begin Aphrodite.**