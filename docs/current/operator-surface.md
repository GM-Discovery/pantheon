---
type: canonical-operator-inventory
status: active
updated: 2026-08-17
---

# Operator Surface Inventory

This is the current CLI inventory. `internal/pantheon/help.go` is the source
for discoverable command metadata; this document supplies the human and GUI
classification.

## Informational commands

| Command | Purpose | GUI equivalent |
| --- | --- | --- |
| `pantheon status` | daemon, model, pause, capabilities, and Rumor summary | Home/System |
| `pantheon system inventory` | canonical secret-free Kernel 20 inventory | Home/System/Pantheon |
| `pantheon show intake` | latest scan and recent occurrences | Intake |
| `pantheon show present` | normalization state | Intake |
| `pantheon show processors` | local processor availability | Rumor/Intake |
| `pantheon show rumor` / `queue` / `source <id>` | audio sources, due work, attempts, handoff | Rumor |
| `pantheon show apollo` / `apollo report <id>` | local reasoning state and cited reports | Models/Apollo |
| `pantheon show janus` / `transitions` / `transition <id>` | temporal subjects and transitions | Work/Temporal |
| `pantheon show future` / `future proposals` / `future proposal <id>` | authority state and proposals | Prometheus/Work |
| `pantheon show mercury` / `orders` / `order <id>` | seals, Go, and execution evidence | Work/System |
| `pantheon show organism` | whole-animal trace | Home/Organism |
| `pantheon grants` | persisted grants | Prometheus |
| `pantheon authority check <capability> <scope>` | read-only effective authority check | Prometheus |
| `pantheon show plugins` / `packages` / `package <code>` | Seshat vocabulary and packages | Bifrost/Packages |
| `pantheon bifrost status` / `list` / `show` / `approvals` / `invocations` | external capability registry and receipts | Bifrost |
| `pantheon bifrost mail status` | bounded mail crossing state | Bifrost |
| `pantheon help`, `help --commands`, `help --codes` | discoverability and package vocabulary | Help |

## Routine or significant mutating commands

| Command | Authority class | Required boundary | GUI equivalent |
| --- | --- | --- | --- |
| `pantheon config validate` | informational | none | Configuration diagnostics |
| `pantheon intake scan` | routine deterministic | local daemon | Intake scan |
| `pantheon present --new` / `--occurrence <id>` | routine source processing | local daemon and Prometheus | Present/process |
| `pantheon rumor scan` / `transcribe <path>` | routine local processing | local control token and Prometheus | Rumor controls |
| `pantheon rumor retry` / `ignore` | significant state change | local control token | Rumor source actions |
| `pantheon plugin validate <path>` | informational | none | Package validation |
| `pantheon plugin register <path>` | significant registry mutation | local control token | Package registration |
| `pantheon bifrost register ...` | significant registry mutation | local control token; registration approval is separate metadata | Capability registration |
| `pantheon bifrost enable/disable <id>` | restricted capability state | local control token; all required Bifrost approvals for enable | Capability controls |
| `pantheon bifrost mail intake` | restricted external read | local control token, Bifrost use approval, Prometheus | Mail intake |
| `pantheon order ...` | significant durable order | local control token | Work/order composer |
| `pantheon janus round` | routine deterministic transition | local control token and Prometheus | Temporal round |
| `pantheon future round` | restricted authority administration | local control token and Prometheus | Authority review |
| `pantheon mercury round` | restricted execution | local control token, valid seal, Prometheus | Execute approved work |
| `pantheon organism run ...` | restricted composed work | local control token, all component boundaries | Organism run |
| `pantheon notify` | restricted external delivery | daemon policy, Prometheus, local notification adapter | Notifications |

## Emergency surface

The canonical emergency command remains:

```text
pantheon prometheus protocol --pause
```

Also available:

```text
pantheon prometheus protocol --status
pantheon prometheus protocol --resume
pantheon prometheus protocol --hard-stop
```

Pause is an Emergency action. Resume is Significant because it changes the
control state, but it does not replay skipped work. Hard-stop is an Emergency
service-manager action. The compatibility aliases `pantheon pause` and
`pantheon resume` remain temporarily supported and are not the canonical GUI
labels.

## Hidden/developer-only surfaces

The local control token, direct HTTP control header, SQLite schema, test
helpers, and optional external helper commands are implementation surfaces,
not operator controls. The GUI must not expose them as ordinary settings.

