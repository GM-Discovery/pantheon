# Kernel 31 Reportback — Switchyard Staging Install

## Result

**PASS — pinned native Switchyard server installed and locally validated through
the Archimedes application boundary; no runtime activation.**

Kernel 31 now has the optional transport artifact and the first internal
application boundary Archimedes can use in a future, separately authorized
deployment. The artifact is staged under `third_party/switchyard/0.2.0`;
Pantheon does not depend on it and the live service did not change.

## What was installed

- Switchyard v0.2.0 from upstream commit
  `1fc9ab887d1c663b0048ae24d5f473d15ed8daaa`.
- Native `switchyard-server`, compiled with Rust 1.96.1 and
  `cargo install --locked`.
- Isolated Python 3.12 wheel environment for package inspection; its legacy
  `switchyard serve` command remains unregistered and unused.
- A local no-op TOML profile with no provider, endpoint, target, credential, or
  fallback.

## License and dependency audit

Switchyard and its native crates are Apache-2.0. The upstream Python NOTICE
records the direct SDK/server-extra licenses, including Apache-2.0, MIT, and
BSD-3-Clause components. Cargo metadata for the pinned lockfile covered 319
packages, 311 of them external, and every package declared an SPDX license
expression. The Python environment is staging-only and is not a distributable
dependency bundle.

## Verification

```text
native/bin/switchyard-server --version
switchyard-server 0.2.0

native/bin/switchyard-server \
  --config routes.local-noop.toml \
  --host 127.0.0.1 --port 48731 --dry-run
server OK: pantheon/kernel31-noop
```

No server process was left running. No route was added to the live Bifrost
configuration. No provider credential was read, no network route was
configured, and no personal material was sent anywhere during the install
proof.

The local Kernel 31 integration test then started the native binary with a
temporary TOML containing one `switchyard/frontier` passthrough route to a fake
localhost upstream. It proved that:

- Archimedes sent the sanitized alias-bearing missive, not the raw identity;
- the native Switchyard route returned a typed response;
- Archimedes translated and rehydrated the response into a proposed Apollo
  calendar draft; and
- a failed upstream produced a readable failed exchange and Bifrost receipt,
  with no fallback and no action draft.

Configuration is persisted under the Bifrost capability surface. Panopticon
does not add a Switchyard navigation page; it renders the endpoint, route,
target label, credential environment name, and optional config-path reference
inside the relevant Bifrost capability card.

```text
GOCACHE=/tmp/pantheon-go-cache go test ./internal/pantheon \
  -run 'Kernel30|Kernel31|Kernel32|Kernel33' -count=1
```

## What remains

Kernel 31 is complete as a staged, locally proven application boundary, not as
a live provider integration. The next step is operational choice: if a real
provider is ever selected, it must be separately configured and approved in
Bifrost, with credentials kept outside Pantheon configuration and no automatic
fallback or authority granted by the provider response.
