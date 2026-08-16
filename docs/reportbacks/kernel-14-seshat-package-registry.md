# Kernel 14 Reportback — Seshat Package Registry & Tool Contracts

## Status

**PASS — repository verified August 16, 2026; live Kernel 14 deployment not performed.**

This report records the repository implementation and verification boundary.
The runtime service was not restarted or changed in this pass. Kernel 13 remains
the installed live runtime until a later, explicitly requested deployment and
live smoke verification.

## Starting point

- Branch: `trunk`
- Exact predecessor commit: `07039bf` (`docs: record Kernel 13 backend deployment`)
- Starting branch state: `trunk...origin/trunk [ahead 10]`
- Starting working tree: clean except for the newly supplied, untracked Kernel
  14 specification, `docs/building kernels/Kernel 14 — Seshat Package Registry & Tool Contracts.md`.
- Kernel 13 implementation/reportback: repository and live backend PASS on
  August 16, 2026.
- Installed runtime at start: Kernel 13 binary
  `/home/brick/pantheon-runtime/bin/pantheon`, SHA-256
  `6e98e3d534cb11028405a5b1883432e541b5d4d279395f4da73386f9898fecf0`.

## Architecture implemented

Seshat is a deterministic registry and package constructor between semantic
missives and adapter-facing packages:

```text
Apollo missive
    ↓
Seshat subject-code contract and normalized package
    ↓
Future / Kernel 13 authority resolution
    ↓
sealed Mercury order and bounded adapter
```

Seshat knows package form, fields, adapter identity, return expectations, and
sensible permission shapes. It does not reason, execute tools, grant authority,
seal orders, interpret ambiguous prose, or fall back to shell execution.
Registration and package construction report no authority. Kernel 13 remains
the authority source.

## Manifest and persistence

The manifest format is JSON because Pantheon already uses JSON for its
machine-readable configuration and has no YAML dependency. A manifest contains
`schema_version`, plugin name/version, and one or more package contracts.
Implemented package fields are:

- `subject_code`, `version`, `description`;
- `capability`, `adapter`;
- `required_fields`, `optional_fields`;
- `payload_rules` mapping fields to missive headers or the literal `body`;
- `returns`;
- `permission_class`, `permission_shapes`;
- optional `aliases`, `constraints`, `examples`, and `enabled`.

Registered manifests persist as human-readable JSON in
`config/plugins/registered/` and are reloaded at daemon startup. Registration
copies a validated manifest into that directory and takes effect immediately;
there is no separate cache or model prompt state. Editing a registered manifest
to set `enabled: false` and restarting disables it; removing the registered
manifest removes it after normal conflict checks.

The built-in proof manifest is
`config/plugins/registered/core.json`. It contains `NOTIFICATION` with the
existing harmless local-notify adapter and `EMAIL:DRAFT` with the Gmail adapter
identity, which is currently unavailable to Seshat’s adapter availability
index. Additional examples are in `config/plugins/examples/`.

## Subject codes, missives, and construction

Subject codes are uppercase colon-delimited identifiers such as
`EMAIL:DRAFT`. A missive has a `TYPE: SUBJECT:CODE` header, optional field
headers, a blank line, and substantive body text. Parsing is deterministic:
duplicate/malformed headers and missing `TYPE` fail; no model is called.

Seshat validates the code, maps declared fields from declared headers or the
body, rejects missing required fields, preserves optional fields and body text,
and emits a normalized package containing:

- subject and package/plugin versions;
- capability and adapter;
- adapter availability;
- normalized fields;
- expected return fields;
- permission-class and permission-shape metadata;
- SHA-256 identity of the source missive.

Only declared fields cross the boundary, so Pantheon-only control headers are
not emitted as adapter fields. Substantive prose is not rewritten. Unknown
subject codes, disabled packages, missing fields, and unavailable adapters
return explicit deterministic errors; unavailable adapters never trigger shell
execution or guessed substitution.

The Apollo-safe index exposes only subject code, description, fields, and
package version. It does not expose adapter implementation details, API
encodings, OAuth, HTTP, MIME, or invocation syntax.

## Extension workflow and inspection

Human extension documentation is at
`docs/seshat-plugin-extension.md`. The usable starter template is
`config/plugins/manifest-template.json`. The documentation explains field
definitions, deterministic mappings, capabilities, adapters, return contracts,
permission shapes, validation, registration, inspection, disable/removal, the
no-authority boundary, and when a genuinely new adapter requires Go code.

CLI surfaces are:

```text
pantheon plugin help
pantheon plugin schema
pantheon plugin validate <path>
pantheon plugin register <path>
pantheon show plugins
pantheon show packages
pantheon show package <subject-code>
```

The matching read-only/local API surfaces are under `/v1/seshat/`. Registration
requires the existing local control token. Validation, inspection, indexing,
and package construction do not create grants or mutate Kernel 13 authority.

## Extension proof

The test suite creates a new `TEST:ACTION` manifest outside Seshat, validates
it, registers it through the registry mechanism, discovers it in the
Apollo-safe index, constructs a valid normalized package, and reopens the
registry to prove persistence. The package uses the existing deterministic
`noop` proof adapter; no Seshat core change is made for the test package.
The result carries `authority: none` at registration, and package construction
does not add an authority field or grant.

## Automated verification

Focused Kernel 14 coverage includes:

- registry initialization and manifest reload persistence;
- unsupported schema, malformed field/mapping, invalid permission shape, and
  self-authority rejection;
- manifest registration and duplicate/conflicting subject-code rejection;
- Apollo-safe discovery with adapter details excluded;
- deterministic missive parsing, required/optional fields, payload/body
  preservation, source identity, capability, adapter, return, and permission
  metadata;
- unknown, disabled, and unavailable package behavior;
- inspection/help/schema endpoints and control-token protection;
- extension without modifying Seshat core;
- existing Kernel 11–13 regression behavior through the complete repository
  suite.

Commands completed successfully:

```text
gofmt -w $(rg --files -g '*.go')
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel14 .
```

Candidate binary SHA-256:

```text
2d7be478ca4e805ca74dcaca5ad77ab2a77e6ad6353816698ef3049d1ee796c4
```

## Live verification boundary

No Kernel 14 binary was installed, no Pantheon service was restarted, and no
live manifest was registered. Therefore this report does not claim live
Seshat health, live package discovery, live package construction, or live
post-restart persistence. The installed identity remains the Kernel 13 hash
recorded above. A later deployment pass must build/install the exact candidate,
inspect the live registry and Apollo-safe index, validate/register a harmless
test manifest, construct a harmless package, prove authority remains absent,
verify persistence, and record the installed SHA-256.

## Files changed

- `internal/pantheon/seshat.go`
- `internal/pantheon/kernel14_test.go`
- `internal/pantheon/daemon.go`
- `internal/pantheon/cli.go`
- `internal/pantheon/welcome.go`
- `config/plugins/registered/core.json`
- `config/plugins/manifest-template.json`
- `config/plugins/examples/notification.json`
- `config/plugins/examples/calendar.json`
- `docs/seshat-plugin-extension.md`
- `docs/software.md`
- `docs/README.md`
- `docs/building kernels/Kernel 14 — Seshat Package Registry & Tool Contracts.md`

## Known limitations and expiry conditions

This kernel does not implement Future authority administration, Janus temporal
transitions, new external adapters, a general package manager, arbitrary code
download, plugin execution, GUI surfaces, recurring work, Archimedes, or
adapter-specific external writes. The `noop` adapter is only a deterministic
construction proof boundary. The report’s repository PASS expires if the
manifest schema, registry path, package mapping, or installed binary changes,
or once a live deployment is attempted without recording the exact candidate
and runtime identity.

## Safe successor

The next safe successor is **Kernel 15 — Future Authority Administration**.
Future will consume Seshat-normalized proposals and resolve them against
Kernel 13 grants. It must not encode plugin requests, manufacture permission,
or move Seshat’s package knowledge into the reasoning layer.
