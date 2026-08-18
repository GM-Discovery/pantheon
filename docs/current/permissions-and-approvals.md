---
type: canonical-permission-inventory
status: active
updated: 2026-08-17
---

# Permission and Approval Inventory

Pantheon has several intentionally separate boundaries. Configuration and
capability presence do not imply permission. A model recommendation is not a
grant, a grant is not a seal, and a seal is not a completed result.

## Canonical action classes

| Class | Meaning | Examples |
| --- | --- | --- |
| Informational | read-only inspection | status, config validation, reports, authority check |
| Routine | bounded, local, reversible deterministic work | scan, local normalization, a declared Janus round |
| Significant | meaningful state or behavior change | package registration, Rumor ignore/retry, config/model change |
| Restricted | external effect or authority-bearing execution | Bifrost invocation, Future sealing, Mercury round, notification |
| Emergency | immediate control of operational authority | Prometheus pause, resume, hard-stop |

## Enforced approvals

| Action | Stored/enforced by | Current rule | Inspection |
| --- | --- | --- | --- |
| Bifrost download | Bifrost approval event | separate approval; no download engine is implemented | `pantheon bifrost approvals` |
| Bifrost installation | Bifrost approval event | separate approval; no installation engine is implemented | `pantheon bifrost approvals` |
| Bifrost registration | capability record and registration event | metadata may be recorded only through the local control boundary | `pantheon bifrost list`, approvals |
| Bifrost first use | Bifrost use approval and enablement | capability must be installed, approved, enabled, and not paused | `pantheon bifrost show <id>` |
| Bifrost automatic update | Bifrost approval event | separate metadata; no update engine is implemented | `pantheon bifrost approvals` |
| Mercury execution | Future seal + local control + Prometheus | exact order, valid authority and adapter required | `pantheon show order <id>` |
| Future authority | existing grants and Prometheus/person referral | Future cannot create authority from reasoning | `pantheon show future`, grants |
| notifications | local control/policy/audit | pause blocks delivery; fixed helper only | audit and `/v1/notify` |
| email actions | separate Kernel 22 capability identity | mark-read, safe label, move, and non-sent draft require exact Future/Mercury order plus `gmail-mail-actions` Bifrost approval; send/delete/archive/filter creation/download remain unavailable | `pantheon show mail`, Bifrost capabilities/invocations |
| frontier model access | configuration placeholder only | disabled; Archimedes is not implemented | `pantheon show config`, inventory |
| self-change | policy declaration | human-readable intent, diff, and explicit human approval required | `config/policy.yaml` |

## Prometheus relationship

Prometheus is above autonomous operational effects. Pause blocks new Intake,
Present, authority, Mercury, Bifrost, notification, Rumor handoff, and
maintenance paths currently governed by the daemon. Safe inspection remains
available. Resume changes only the pause state; it does not replay missed work.

The direct operator control is:

```text
pantheon prometheus protocol --pause
```

## Findings

- The Bifrost approval model is represented in SQLite and exposed by the CLI;
  it is not duplicated as a second permission system.
- The human-readable policy files are declarations, not silent grants. Go
  handlers enforce control tokens, Prometheus, grants, seals, and adapter
  restrictions.
- The old flat capability file contains historical/compatibility capabilities
  such as `email.archive` and `email.tag`; the Kernel 19 read-only Bifrost adapter does
  not expose those operations. They must not be presented by the GUI as
  currently enabled actions merely because the declarations remain.
- No credentials were found in the tracked canonical configuration, status
  output, Bifrost descriptions, or reportback files inspected for this kernel.
  The local control token is stored under ignored runtime state and is omitted
  from inspection. Kernel 20 does not rotate credentials.
