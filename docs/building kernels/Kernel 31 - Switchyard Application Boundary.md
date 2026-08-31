# Kernel 31 — Switchyard Application Boundary

## Status

Staged and verified — native process boundary and local application proof;
optional application layer, not a Pantheon kernel dependency and not enabled in
the live runtime.

This paper prepares a later integration of
[NVIDIA-NeMo/Switchyard](https://github.com/NVIDIA-NeMo/Switchyard) as a
replaceable transport application for Archimedes. The upstream project
describes Switchyard as a Rust proxy/library for routing and translating LLM
traffic, and currently labels it pre-alpha and experimental. Its repository
identifies the project as Apache-2.0 licensed. Those facts must be rechecked at
the time of adoption, alongside a dependency and NOTICE audit.

On August 24, 2026, the v0.2.0 native server was installed locally for this
bounded proof. The upstream tag resolves to commit
`1fc9ab887d1c663b0048ae24d5f473d15ed8daaa`. The native binary is staged at
`third_party/switchyard/0.2.0/native/bin/switchyard-server` and was built with
Rust 1.96.1 using `cargo install --locked`; its SHA-256 is
`28a967534e128ec235d6131b355708847607197a4790febe94c69e1e9d0f49e9`.

The Python wheel is also present in the isolated
`third_party/switchyard/0.2.0/.venv` for package inspection only. It exposes
the upstream transition-only `switchyard serve` path; the native binary is the
selected Kernel 31 artifact.

The install audit found Apache-2.0 for Switchyard and its native crates. The
locked Cargo metadata covered 319 packages, including 311 external packages;
all declared SPDX license expressions. The Python package's NOTICE identifies
the direct SDK and server-extra licenses (Apache-2.0, MIT, and BSD-3-Clause).
The installed Python environment is not a release bundle and must not be
copied into a distributable build without a separately generated attribution
bundle.

The local validation profile is
`third_party/switchyard/0.2.0/routes.local-noop.toml`. It declares no client,
endpoint, target, credential, or provider and exists only for native config
parsing.

## Placement

Switchyard belongs after the local privacy and authority boundaries:

```text
Apollo referral
    ↓
semantic minimization
    ↓
deterministic privacy gate
    ↓
Future exact order
    ↓
Mercury seal and Go
    ↓
Bifrost capability approval
    ↓
Switchyard transport application
    ↓
Archimedes response translator and rehydrator
    ↓
Apollo typed action drafts
    ↓
Future acceptance → Mercury/Bifrost adapter
    ↓
explicit provider target
```

Switchyard is a router and protocol translator. It is not an escalation
decider, privacy gate, authority plane, tool host, Vault reader, or provider
marketplace.

Archimedes owns the return boundary. It translates the untrusted response into
either plain text or a typed suggestion, rehydrates approved aliases only
inside Archimedes, and hands calendar/email suggestions to Apollo's existing
action-draft path. That is an Apollo handoff, not a second authority grant or a
direct calendar/email execution.

## First safe Switchyard increment

The implemented application proof starts the staged native
`switchyard-server` as a temporary supervised localhost process with one
explicit `passthrough` route to a local fake upstream. It proves:

- the sanitized request enters Switchyard only after the Archimedes gate;
- the configured target is the target authorized by the Bifrost capability;
- the response returns through the Archimedes envelope unchanged in meaning;
- typed response suggestions are translated, rehydrated, and returned as
  Apollo action drafts still awaiting Future acceptance;
- the route, target, latency, status, and body hashes are recorded locally;
- no raw local context is added by the proxy; and
- a failed or unavailable target remains a bounded exchange failure with a
  Bifrost invocation receipt.

Do not begin with classifier routing, automatic escalation, stronger-model
fallback, A/B traffic splitting, or multiple live providers. Any of those can
be useful transport experiments later, but they make the first safety proof
harder to inspect.

## Bifrost capability shape

Switchyard is represented as a specific Bifrost application capability, not as
a generic permission to use whatever the proxy can reach. Bifrost owns the
endpoint, route, target label, credential environment name, and optional
configuration-path reference. Panopticon exposes those controls inside the
Bifrost capability surface; Switchyard has no separate navigation page.

```text
capability_id: archimedes-switchyard-local
component: Archimedes
operation: frontier.exchange
resource: switchyard://route/<explicit-route-id>
network_access: true when a provider target is configured
sends_data_off_machine: true for a remote provider; false for a local fake
authentication_required: provider-specific
prohibited_operations:
  - arbitrary_route_creation
  - tool_invocation
  - filesystem_read
  - vault_query
  - identity_lookup
  - context_expansion
```

The route ID, target ID, provider identity, and network-egress fact belong in
the capability and invocation receipt. A route configuration must not silently
expand the order's scope.

## Process boundary

The live-adjacent design keeps Switchyard outside the Pantheon daemon as a
separately supervised localhost process. The Kernel 31 test harness exercises
that process boundary temporarily; it does not install a persistent service.
The default is:

- bind to `127.0.0.1` only;
- use a dedicated temporary configuration containing only approved routes;
- pass requests through an authenticated local boundary;
- keep provider credentials outside receipts and request logs;
- give the process no Vault mount, identity-map path, or general shell access;
- stop the application when Prometheus is paused or the capability is disabled;
- reject any route or target not named in the sealed order; and
- treat the process as unavailable rather than silently routing elsewhere.

Pantheon does not install, start, or register Switchyard automatically. The
staged build is an operator-side artifact; the test registers an ephemeral
capability in a temporary database only. Normal Bifrost registration does not
download, enable, or approve it.

## Request and response contract

Pantheon-to-Switchyard requests should contain only:

```json
{
  "exchange_id": "exchange-local-test",
  "order_id": "order-frontier-test",
  "order_seal_digest": "sha256:...",
  "capability_id": "archimedes-switchyard-local",
  "route_id": "passthrough-local-fake",
  "turn": 1,
  "expires_at": "...",
  "missive": {
    "question": "...",
    "facts": ["..."],
    "constraints": ["..."],
    "aliases": ["Person#..."],
    "uncertainties": ["..."]
  },
  "missive_sha256": "sha256:..."
}
```

The actual implementation may use a different wire format, but the boundary
must preserve these facts. In particular, `missive` is the already-sanitized
product of Archimedes; it is not a reference that Switchyard may dereference
into the Vault.

The local result must identify:

```text
exchange_id
turn
route_id
target_id
provider_id and model, when available
provider_status
response body hash
started_at / completed_at
network activity
failure kind, if any
```

The provider response remains untrusted reasoning. Switchyard cannot create an
order, grant, calendar event, email action, notification, tool call, or new
provider route. Archimedes may convert an explicitly typed response into a
Future-held Apollo action draft; it never skips that approval path.

## Route rules

The first route file should contain one explicit route and no fallback:

```text
passthrough-local-fake → fake-provider
```

Later routes may be introduced only as separately reviewed capability data.
Switchyard itself may optimize among already-authorized targets, but it may
not:

- choose a stronger model because the question looks difficult;
- add another provider when the selected one fails;
- retry with a wider missive;
- ask for raw context;
- call tools or browse;
- create a Future proposal or Mercury order; or
- conceal the target from Panopticon receipts.

## Acceptance proof

The Switchyard application proof passes only when tests demonstrate:

1. A request without a valid sealed order is rejected.
2. A request with a valid order but unapproved capability is rejected.
3. A route outside the order's explicit target scope is rejected.
4. The native proxy receives the sanitized missive and no local path or
   identity map.
5. A local fake upstream completes a request through the native binary and
   produces a Bifrost receipt plus a typed Apollo action draft.
6. A provider failure remains an exchange failure with a readable failed
   Bifrost receipt and no fallback or action draft.
7. Provider and route metadata are inspectable without secrets.
8. Request and response hashes permit local correlation without preserving
   credentials or unnecessary raw packets indefinitely.
9. The same exchange can handle a bounded clarification only after Apollo
    rebuilds and the privacy gate rechecks the packet.

The implemented test command is:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./internal/pantheon \
  -run 'Kernel30|Kernel31|Kernel32|Kernel33' -count=1
```

It uses the installed v0.2.0 native binary and a local fake upstream. It does
not require an Internet provider account or send real personal material.

## Distribution and licensing gate

Before Switchyard is included in a distributable Pantheon package:

- pin a reviewed upstream revision or release;
- record the upstream license and required NOTICE text;
- audit transitive Rust dependencies and bundled assets;
- document whether the standalone binary or library path is used;
- record the network and credential behavior of the selected route; and
- keep the Switchyard application optional so Pantheon remains usable without
  it.

The current proof satisfies the pin, license, NOTICE, binary-path,
dependency-inventory, temporary process, Bifrost configuration-ownership, and
local failure-trace portions of this gate. It does not satisfy provider
selection or live transport approval.

Pantheon’s own license decision remains independent of Switchyard’s license.
Using an Apache-2.0 application does not grant it authority over Pantheon’s
privacy, policy, or execution contracts.

## Non-goals

This increment does not:

- start a persistent Switchyard service; the test harness only starts and
  stops a temporary local process;
- select a frontier provider;
- create a real provider capability or approval;
- enable external data egress;
- add automatic model routing or fallback;
- expose direct frontier chat;
- replace Bifrost, Future, Mercury, Apollo, or the privacy gate; or
- make Archimedes live.
