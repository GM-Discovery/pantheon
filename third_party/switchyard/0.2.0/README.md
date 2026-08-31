# Switchyard v0.2.0 staging record

This directory is an operator-side Kernel 31 staging area. It is not started
by Pantheon, added to `PATH`, registered with Bifrost, or connected to a
provider.

## Installed artifacts

- Native server: `native/bin/switchyard-server`
- Native version: `0.2.0`
- Upstream revision: `1fc9ab887d1c663b0048ae24d5f473d15ed8daaa`
- Rust toolchain used: `1.96.1`
- Native SHA-256: `28a967534e128ec235d6131b355708847607197a4790febe94c69e1e9d0f49e9`
- Python inspection environment: `.venv`
- No-op validation profile: `routes.local-noop.toml`

## Validation

```text
native/bin/switchyard-server --version
switchyard-server 0.2.0

native/bin/switchyard-server \
  --config routes.local-noop.toml \
  --host 127.0.0.1 --port 48731 --dry-run
server OK: pantheon/kernel31-noop
```

The no-op profile contains no LLM client, endpoint, target, route fallback,
credential name, or provider. `--dry-run` validates configuration without
binding a socket.

## Licensing

Switchyard v0.2.0 is Apache-2.0 licensed. The Python wheel includes the
upstream `LICENSE` and `NOTICE` files under its `.dist-info/licenses` directory.
The native build used the pinned Cargo lockfile with `--locked`; the Kernel 31
documentation records the resulting dependency/license inventory.

Do not ship the Python virtual environment or local Rust toolchain as part of a
Pantheon release without refreshing the attribution and reproducibility audit.
