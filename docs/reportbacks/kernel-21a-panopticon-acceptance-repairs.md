---
kernel: 21A
status: REPAIRED — repository verified; operator visual walkthrough pending August 17, 2026
parent: 21
---

# Mini-Kernel 21A Reportback — Panopticon Acceptance Repairs

Kernel 21A closes the small defects found during the first live Panopticon
walkthrough. It does not add a new service, alter the local-serving model, or
begin Kernel 22.

## Repairs

- Bifrost capability rendering now accepts either a capability array or a
  wrapped `{capabilities: [...]}` response and safely handles `null`, malformed
  data, and HTTP failure. The surface renders **Capability information
  unavailable**, marks the state unknown, preserves the rest of the page, and
  provides a retry button. Technical detail is sent to browser diagnostics via
  `console.warn`; raw JavaScript/runtime errors are not shown to the operator.
- Present, Past, and Future now appear as one grouped Fates operational
  surface. Weave is a smaller explicitly labeled tool used by the Fates, not a
  fourth peer occupant.
- Home now makes Current Work the primary operational panel below Radio
  Prometheus. It shows active work, waiting work, Needs Grant, blocked work,
  needs decision, recent completion, and upcoming known work before the lower
  generic cell inventory. Each card is clickable.
- A bounded read-only `GET /v1/gui/work` surface now supplies the underlying
  proposals, orders, and temporal subjects for a selected category. Each item
  explains why it appears, its current state, and the next safe step. The
  former generic Home details link now opens this work context instead of the
  raw system inventory.
- Execution / Capabilities now consumes those same records with explicit
  filters for All current work, Active work, Blocked work, and Needs decision.
  A blocked record visibly says when no safe execution action is available;
  the GUI does not offer a fake retry or invent an option.
- The successful Radio Prometheus, Father Time, Rumor, Configuration,
  navigation, and embedded local-serving behavior was preserved.
- The GUI continues to use read-only Kernel 20 inventory for its polling path;
  it does not poll the legacy status route that writes an audit row on reads.

## Changed files

- `internal/pantheon/web/index.html`
- `internal/pantheon/web/app.css`
- `internal/pantheon/web/app.js`
- `internal/pantheon/kernel21_test.go`
- `internal/pantheon/kernel21a_test.go`
- `internal/pantheon/daemon.go`
- `internal/pantheon/gui.go`
- this reportback

The existing Kernel 21 daemon, GUI, contract, operator-guide, and user-added
untracked documents were preserved. No database files were deleted or reset.

## Verification

Focused acceptance tests:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./internal/pantheon -run Kernel21A -count=1
```

The acceptance tests verify the Fates grouping, Weave's tool treatment,
Current Work's position before the generic inventory, and the null/unavailable
capability fallback contract. Existing Kernel 21 tests continue to cover
embedded serving, GUI sessions, Prometheus pause/resume, configuration
validation/application, and local-origin enforcement.

Repository checks passed:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
node --check internal/pantheon/web/app.js
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel21a .
```

## Live service and visual walkthrough

The Kernel 21A build was installed into the local Pantheon service and the
service was restarted. Live HTTP delivery of the repaired Home/Fates markup
was confirmed. The operator will perform the visual walkthrough personally.

No screenshot is claimed in this report: the available headless Firefox
renderer exited before producing an image in this environment. This is an
environment limitation, not a product result.

## Remaining visual limitations

The current work board is count-based because Kernel 20 inventory deliberately
does not expose full per-item work cards. Detailed order, proposal, and
occurrence evidence remains in the CLI and subsystem routes. The browser
walkthrough is local-only and does not claim mobile, remote, or live
transcription behavior.
