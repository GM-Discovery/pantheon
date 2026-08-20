# Teaching Seshat a Plugin

Seshat is Pantheon’s deterministic package registry. A plugin manifest teaches
Pantheon how to turn a short subject code and an auditable missive into a
normalized adapter package. It does not grant permission. Kernel 13 grants are
the only authority source.

## Manifest workflow

Copy [the starter manifest](../config/plugins/manifest-template.json) to a
separate local file, change the plugin identity and package fields, then run:

```text
pantheon plugin schema
pantheon plugin validate /path/to/my-plugin.json
pantheon plugin register /path/to/my-plugin.json
pantheon show plugins
pantheon show packages
pantheon show package MY:CODE
```

Registration validates first, copies the manifest into
`config/plugins/registered/`, and takes effect immediately in the running
daemon. The copied manifest is loaded again at the next daemon start, so the
package registry survives restart. To disable a package, edit its registered
manifest to set `enabled` to `false` and restart the daemon. Remove the
registered manifest to remove the package, after checking that no other
manifest uses its subject code.

## Fields

Each package declares an uppercase `subject_code`, its contract `version`, a
human description, a Pantheon `capability`, and the adapter identity. Required
and optional field names are lowercase identifiers. `payload_rules` maps a
field to an uppercase missive header or the literal `body`; this is deterministic
mapping, not model interpretation. `returns` lists the fields the adapter is
expected to return. `permission_class` and `permission_shapes` describe
sensible authority options for Future; they never create a grant.

If a package has temporal behavior, add the optional `temporal` block. Its
`mode` is `scheduled`, `reviewable`, or `active`; `missed_policy` must be one
of `expire`, `resurface`, `backfill`, `recompute`, or `require_review`; and
`pause_policy` may be `hold`, `missed`, or `require_review`. Seshat validates
these values and Janus applies them when a bounded subject crosses its
instance threshold. The manifest declares the rule; it does not schedule an
instance or grant authority.

Apollo needs only the safe index exposed by `pantheon show packages`: subject
code, description, and fields. It does not need Gmail JSON, OAuth details,
HTTP routes, MIME rules, or adapter invocation syntax. A missive begins with a
`TYPE: SUBJECT:CODE` header, optional field headers, a blank line, and the
substantive body. Pantheon-only headers are not passed into normalized fields.

## Validation and adapters

Validation rejects unsupported schema versions, duplicate codes, malformed
fields or mappings, invalid permission shapes, and manifest keys that attempt
to declare authority. A valid package has `authority: none` until Future and
the Kernel 13 authority machinery resolve an actual operation.

Seshat knows the adapter identity and whether that identity is presently
available. It never executes the adapter and never falls back to shell
execution. A new package contract can be manifest-only when an existing
adapter already understands it. A genuinely new protocol or external tool
still requires a bounded adapter implementation in Go (or a separately
installed, explicitly integrated plugin); the manifest alone cannot create
that implementation.

The currently included proof packages are in
`config/plugins/registered/core.json`. Additional structured examples are in
`config/plugins/examples/`, including `janus-temporal.json`.
`pantheon plugin help` and `pantheon plugin schema`
provide the same local workflow without requiring Internet access.
