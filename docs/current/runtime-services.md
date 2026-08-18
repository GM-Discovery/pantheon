---
type: canonical-runtime-inventory
status: active
updated: 2026-08-17
---

# Runtime and Service Inventory

This inventory records what starts automatically and who owns it. The service
manager files were inspected on 2026-08-17; service activation state could not
be queried from the restricted audit environment, so “configured” and “live”
are kept distinct.

## Services and timers

| Unit | Type | Starts | Owner | Restart/failure behavior | Current classification |
| --- | --- | --- | --- | --- | --- |
| `pantheon.service` | long-running user service | user systemd default target | Pantheon runtime | `Restart=on-failure`, 3-second delay; service is the daemon owner | configured; live state must come from systemd or `/v1/status` |
| `pantheon-weave-refresh.timer` | user timer | systemd timers target; calendar `03:15 America/Los_Angeles` | Weave maintenance service | persistent timer invokes `pantheon-weave-refresh.service` | intentionally retained scheduled maintenance |
| `pantheon-weave-refresh.service` | oneshot | timer | Pantheon runtime | runs `pantheon weave-refresh`; ordered after `pantheon.service` | intentionally retained; Prometheus blocks the endpoint while paused |
| `pantheon-vault-backup.timer` | user timer | systemd timers target; boot + one hour cadence | vault backup helper | persistent timer invokes `pantheon-vault-backup.service` | external Git backup; outside Go runtime |
| `pantheon-vault-backup.service` | oneshot | timer | user shell helper | commits and pushes Vault changes; failure is service-manager-visible | retained; not a Pantheon reasoning or authority loop |
| `hermes-gateway.service` | long-running user service | user systemd default target | Hermes installation | `Restart=always`, 5-second delay | retained rollback/external gateway; not retired by Kernel 19 |

No Pantheon-owned cron job was found in the repository. The
`mercury.round_on_cron_wake` and `bifrost.mail.schedule=hourly` values are
configuration contracts; neither creates a scheduler. The Rumor watcher is
started by the Pantheon daemon and is an in-process polling loop, not a
separate service.

## Ownership rules

- Systemd starts/stops `pantheon.service`, its timers, and the Hermes gateway.
- Pantheon owns its SQLite state, internal Rumor watcher, Prometheus pause
  state, and all route-level enforcement.
- The Weave timer owns when scheduled local maintenance is requested; the
  daemon owns whether the operation is allowed and records the audit.
- The backup helper owns its own Git commit/push behavior. It does not grant
  Pantheon access to GitHub and is not a Bifrost adapter.
- Hermes owns its gateway process and any external jobs still configured in
  its installation. Bifrost does not silently disable or replace those jobs.

## Reboot and failure behavior

After reboot, enabled user units are the source of truth. Pantheon should
restart under systemd according to its unit; the daemon reopens durable SQLite
state and retains Prometheus pause markers. A failed Pantheon process is
restarted by systemd, while a failed Rumor item remains in its durable retry or
ignored state. A failed timer run is visible to systemd and does not become an
automatic replay burst. Resume after pause is explicit and non-replaying.

The Kernel 22 inventory endpoint reports live daemon state and durable work
counts, but does not pretend to know the systemd activation state when the
service manager is unavailable.

## Hermes residual audit

| Residual | Classification | Reason |
| --- | --- | --- |
| Mercury labels `hermes` / `Terra` | intentionally retained compatibility vocabulary | deterministic fallback is current execution proof; external helper remains optional |
| `hermes-gateway.service` | retained rollback/external integration | no Kernel 19 cutover proof retired it |
| legacy `hermes` CLI route | compatibility/deferred | current CLI reports it as direct and not daemon-routed |
| old capability declarations in `config/capabilities.json` | compatibility and operator vocabulary | Bifrost is the governed Kernel 19 path; declarations do not authorize use |
| Hermes routine mail jobs | not proven superseded | Kernel 22 adds a manual Pantheon mail path, but no live cutover proof was performed; existing jobs remain intact |
| Terra as an architecture role | superseded | Terra is only an external helper label; Mercury is the canonical runtime role |

Kernel 22 does not uninstall Hermes, stop its service, or retire its jobs. It
also does not add a scheduler; mail processing is explicit and manual until
Zodiac is implemented.
