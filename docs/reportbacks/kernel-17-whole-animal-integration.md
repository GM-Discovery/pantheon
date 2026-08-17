# Kernel 17 Reportback — Whole-Animal Integration, Documentation, and Usability

**Status: PASS — repository and live runtime verified August 17, 2026.**

## Documentation first

Kernel 17 began by reconciling the active documentation rather than treating
the kernel reports as the architecture. Reviewed material included the
roadmap, Foundations, Construction, Software, configuration and Prometheus
guidance, the daemon milestone, Seshat extension documentation, Father Time
and Janus behavior, Future authority behavior, Mercury execution behavior, and
the current command surface.

Changed:

- `docs/pantheon-architecture-brief.md` and `docs/Pantheon Roadmap.md` remain
  the concise active architecture and direction documents established before
  implementation.
- `docs/software.md` now records the canonical
  `Intake → Present → Past → Apollo → Seshat → Janus → Future → seal → Mercury`
  flow and Future’s current deterministic authority-signing boundary.
- `docs/construction.md` now assigns Intake, Present, Apollo, Seshat, Janus,
  Future, and Mercury their current responsibilities.
- `docs/daemon-milestone.md` is explicitly historical so its old out-of-scope
  statements are not mistaken for the current architecture.
- `docs/operator-guide.md` provides the working command, pause, inspection,
  package, and blocked-work guidance.
- `docs/README.md` links the active brief, roadmap, operator guide, and this
  reportback.
- Seshat’s runtime help points to `pantheon help --codes`.

No historical kernel reportback was rewritten or deleted. No active document
was retired; the historical daemon milestone is marked as historical and
points to current sources. The active architecture map is
`docs/pantheon-architecture-brief.md`; the operator guide is
`docs/operator-guide.md`; package extension guidance remains
`docs/seshat-plugin-extension.md`.

The major consideration exposed by reconciliation was a real integration
boundary: Kernels 13–16 had durable authority, package, temporal, and Future
ledgers, but required a human/operator to manually hand Apollo’s result to
Seshat and then Future. Kernel 17 adds a bounded integration trace and run
surface to make that handoff explicit and inspectable. It is deterministic
orchestration, not a new reasoning organ, authority registry, or active-state
engine. A package still supplies its own missive, Seshat still supplies its
capability/adapter identity, Future still decides whether authority is
sufficient, and Mercury still executes only the sealed order.

## Runtime usability

The final local metadata surfaces are:

```text
pantheon help
pantheon help --commands
pantheon help --codes
pantheon show codes
```

Help and code discovery do not require the daemon or a model. The command
catalog groups current commands and states whether they mutate state and need
the local control boundary. Code discovery shows subject code, meaning,
required/optional fields, capability, and package readiness without exposing
adapter internals in the normal index.

The welcome renderer now uses a fixed visible-width box helper. ANSI escape
sequences are excluded from width calculations, long model/Vault/runtime
strings are truncated cleanly, Unicode is measured consistently as runes, and
every boxed line has the same visible width. The Prometheus Protocol remains
dominant and is adjacent to the actual command:

```text
pantheon prometheus protocol --pause
```

The discovery instruction is immediately adjacent:

```text
pantheon help --commands
```

The welcome also reports whether the daemon is alive, whether it is paused,
and how to begin inspection. If the daemon is unavailable, local help remains
usable and the welcome reports the degraded state.

## Integration implementation

`POST /v1/organism/run`, exposed as `pantheon organism run`, records a bounded
run trace in the `organism_runs` ledger. It accepts a human request, an
optional Intake occurrence, a Seshat subject code, a scope, and a Seshat
missive. The path is:

```text
Intake scan / occurrence
→ Present verification
→ Apollo sealed person-direct order
→ Past / Weave retrieval during Apollo preparation
→ Mercury Apollo dispatch
→ Seshat normalization
→ Janus validation when the package is temporal
→ Future authority resolution and seal
→ Mercury adapter order
→ result and evidence references
```

The operation is control-boundary protected. The integration surface does not
allow a model or package manifest to create authority. Direct Apollo
reasoning is person-authorized; downstream operational work is Future-
authorized or safely retained as awaiting authority.

Focused Kernel 17 tests prove:

- an authorized local notification traversed Present, Apollo, Seshat, Future,
  and Mercury and produced exactly one adapter attempt and one Go;
- the same handoff without an active Future grant ended in `awaiting_authority`
  / `needs_person`, produced no local-notify order, and performed no adapter
  execution;
- a temporal package still requires a Janus subject and the inherited Janus
  tests prove threshold transition/idempotency behavior;
- registered package discovery remains available and package registration
  remains authority-free through the inherited Seshat tests;
- Future idempotency, Mercury replay protection, and Prometheus pause/no-replay
  remain green through the prior-kernel regression suite.

For a non-temporal local notification, Janus is legitimately recorded as
`not required`: the package has no temporal contract. Temporal package work
cannot use this shortcut; Future requires an eligible Janus subject.

## Verification

Automated verification completed:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./...       PASS
GOCACHE=/tmp/pantheon-go-cache go vet ./...        PASS
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel17 .  PASS
```

The focused Kernel 17 tests and all prior kernel tests pass. The candidate
SHA-256 and installed runtime SHA-256 are recorded in the live verification
section below.

Live verification completed against the exact candidate:

- candidate build: `/tmp/pantheon-kernel17`;
- installed executable: `/home/brick/pantheon-runtime/bin/pantheon`;
- candidate and installed SHA-256: `754e13d63b8829f4585ed97577f91dda5265aeaa8e64d5fbfbeea195ed12229b`;
- `pantheon.service` restarted at `2026-08-17 01:34:20 PDT`, reported active,
  and returned `ExecMainStatus=0` with MainPID `1767587`; the running `/proc`
  executable had the same SHA-256;
- welcome display showed aligned borders, Prometheus Protocol, and command
  discovery;
- `pantheon help`, command discovery, Seshat code discovery, authority,
  Future, Janus, Mercury, and whole-animal inspection commands returned;
- authorized and insufficient-authority integration paths were exercised with
  bounded local evidence;
- pause blocked the integration surface and resume did not replay it;
- active documentation paths were present.

The live authorized direct-request trace was
`organism-6e4b3e3d57ba53d90d716ab6ed9c1f41`: Apollo report
`report-b3bc387c0d5cc7d907cefae50e197389`, Seshat `NOTIFICATION`, Future
proposal `proposal-e3b74c2f0bda944d534ecba168e2ef73`, and completed Mercury
order `order-5337e9b700a638a2cfe64a811c83c7ba`. Its order was sealed under the
one-use grant `grant-1786954520474326507`, and the grant was consumed. Because
the live existing occurrence had already departed from Intake, this live
proof intentionally used the supported direct-request form; the repository
focused proof exercised the full occurrence → Present path. The live trace
records the legitimate skips as `Present: direct human request` and
`Past: no occurrence-local source`.

The live refusal trace was
`organism-bfd9dccace5f437745c7bf47d55c46fd`, ending
`awaiting_authority` with Future `needs_person`, Apollo report
`report-1ef03361fe66e955bef9f4570119bb9c`, and no Mercury order. The paused
request trace was `organism-06eee9b57279a9680b3e0796eb664a67`; it ended before
Apollo with `Prometheus is paused; organism run did not start`. Resume reported
that skipped work would not replay, and the subsequent status was
`paused: false` / `control: healthy`.

## Files changed

Runtime:

- `internal/pantheon/help.go`;
- `internal/pantheon/organism.go`;
- `internal/pantheon/kernel17_test.go`;
- `internal/pantheon/cli.go`;
- `internal/pantheon/daemon.go`;
- `internal/pantheon/seshat.go`;
- `internal/pantheon/welcome.go`.

Documentation:

- `docs/README.md`;
- `docs/construction.md`;
- `docs/daemon-milestone.md`;
- `docs/operator-guide.md`;
- `docs/software.md`;
- this reportback;
- the Kernel 17 specification.

## Known limitations and expiry conditions

The integration surface is intentionally bounded. Apollo’s structured result
is evidence for Future, not authorization. Future still administers supplied
proposals; it does not yet own the active-state engine or determine arbitrary
child work. A temporal integrated run requires the caller to provide the
appropriate Janus subject. Only explicitly implemented Mercury adapters can
execute a sealed package.

This PASS expires if the installed binary differs from the recorded hash, if
the service is replaced without rerunning the live checks, if Seshat package
identity or adapter availability changes, if Future/Janus authority behavior
changes, or if Prometheus pause/resume semantics change. Kernel 18 remains a
separate Rumor specification and was not started by this pass.
