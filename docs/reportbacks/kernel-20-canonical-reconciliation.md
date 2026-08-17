---
kernel: 20
status: PASS — repository reconciliation verified August 17, 2026
predecessor: 659c764d2d91ed47499aea0eeaebe7842b90a812
---

# Kernel 20 Reportback — Canonical Reconciliation and GUI Preparation

Kernel 20 is complete as a repository reconciliation pass. It did not build
the GUI, install a plugin, add a major integration, restart the runtime, or
begin Kernel 21. The clean predecessor was Kernel 19 commit
`659c764d2d91ed47499aea0eeaebe7842b90a812`.

## Result

The repository now has one explicit current-state layer for Kernel 21 to use:

- [Current system inventory](../current/system-inventory.md)
- [Configuration inventory](../current/configuration-inventory.md)
- [Operator surface inventory](../current/operator-surface.md)
- [Runtime and service inventory](../current/runtime-services.md)
- [Permission and approval inventory](../current/permissions-and-approvals.md)
- [GUI readiness contract](../gui/GUI-CONTRACT.md)

Historical kernel specifications and reportbacks remain under `docs/building
kernels/` and `docs/reportbacks/`. They were not rewritten to pretend that
later architecture existed earlier.

## Implementation

Added a deterministic, read-only GUI/system inventory:

```text
GET /v1/system/inventory
pantheon system inventory
```

The versioned response (`contract_version: 1`) reports component states,
model configuration, bounded work counts, Prometheus state, failure counts,
and links to deeper inspections. It omits the control token, credentials, raw
mail, transcript bodies, and full Apollo reports. The route does not create an
audit event because it is read-only inspection.

The surface records implemented/deferred states for Pantheon, Intake, Present,
Past, Weave, Apollo, Seshat, Father Time, Janus, Future, Mercury, Prometheus,
Rumor, Bifrost, Notifications, Archimedes, Orion, and Zeus.

## Audit findings reconciled

- The canonical flow is Intake → Present → Past/Weave → Apollo → Seshat →
  Janus when temporally relevant → Future → sealed Mercury order → Bifrost or
  local adapter.
- Future is deterministic authority administration and exact sealing; it is
  not an active-state planning engine.
- Rumor is local Whisper-backed audio evidence production. Live transcription
  and Project Fame behavior remain deferred.
- Bifrost is a governed bridge with separate download, installation,
  registration, use, update, enablement, invocation, and receipt concepts. Its
  first adapter remains bounded read-only Himalaya/Gmail intake.
- The current Bifrost adapter does not prove a live Hermes cutover. Hermes is
  retained as an external gateway/rollback path, and its scheduled jobs were
  not disabled or uninstalled.
- `mercury.round_on_cron_wake` and `bifrost.mail.schedule` are contracts, not
  schedulers. Actual automatic ownership is documented from the user-systemd
  unit files: Pantheon service, Weave refresh timer, Vault backup timer, and
  Hermes gateway.
- The old flat capability declaration file remains compatibility vocabulary;
  it does not mean the old email tag/archive actions are active Bifrost
  operations.
- The repository has no generic Bifrost download/install engine, no active
  Archimedes path, no Orion scheduler, and no Zeus sanitation component.

## Configuration and secrets

The existing typed schema/version 1 configuration remains canonical. The audit
documents meaningful values, defaults, validation, restart behavior, GUI
exposure, and authority class without exposing implementation debris as
operator settings. `pantheon show config` continues to omit secrets.

No credentials were found in the tracked canonical configuration, status
output, Bifrost descriptions, or reportback files inspected for this kernel.
The local control token remains in ignored runtime state and is not surfaced.
No credentials were rotated.

## Repository organization

Files added:

- `internal/pantheon/system_inventory.go`
- `internal/pantheon/kernel20_test.go`
- `docs/current/system-inventory.md`
- `docs/current/configuration-inventory.md`
- `docs/current/operator-surface.md`
- `docs/current/runtime-services.md`
- `docs/current/permissions-and-approvals.md`
- `docs/gui/GUI-CONTRACT.md`
- this reportback

Files materially updated:

- `internal/pantheon/daemon.go`, `cli.go`, and `help.go` for the inventory
  route and command.
- `docs/README.md`, `README.md`, `docs/pantheon-architecture-brief.md`,
  `docs/configuration.md`, `docs/operator-guide.md`, and the active roadmap.

Files moved: none.

Files removed: none. Existing binaries, runtime artifacts, fixtures, and
untracked user documents were not deleted or staged. No safe proof established
that the tracked compatibility declarations or deployment artifacts could be
removed without changing an existing workflow.

## Verification

Focused Kernel 20 tests passed:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./internal/pantheon -run Kernel20 -count=1
```

The full repository test, vet, and build pass are the final completion checks
for this kernel. No live GUI verification was performed because Kernel 21 has
not begun. Systemd activation state could not be queried from the restricted
audit environment; the service ownership and restart contracts are documented
from the unit files, and live daemon truth remains available from `/v1/status`
and `/v1/system/inventory` when the service is reachable.

## Next clean predecessor

After commit, this reportback and the inventory surface become the clean
predecessor for Kernel 21. Kernel 21 may begin the GUI against the documented
contract. Plugin/MCP discussion and external capability changes remain outside
this pass.

