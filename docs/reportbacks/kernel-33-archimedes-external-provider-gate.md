# Kernel 33 Reportback — Archimedes External Provider Gate

## Result

**PASS — provider-neutral external transport implemented and live transport
left disabled.**

Kernel 33 extends the completed Kernel 30 local fake exchange with an
OpenAI-compatible HTTP adapter. The adapter is opt-in, Bifrost-gated, bounded,
and tested only against a local fake HTTP endpoint.

## What changed

- Added explicit `archimedes` configuration for provider, endpoint, model,
  route, credential environment name, timeout, response bound, and Bifrost
  capability ID.
- Kept `enabled: false` and `local-fake` as the live defaults.
- Added HTTPS and credential-free URL validation; loopback HTTP is allowed only
  for local tests.
- Added an OpenAI-compatible chat request/response adapter.
- Read credentials only from the named environment variable at invocation
  time; no secret is written to configuration, SQLite, artifacts, audit
  records, or logs.
- Required Bifrost authorization immediately before network access, including
  explicit `network_access` and `sends_data_off_machine` declarations.
- Added bounded timeout and response size checks with no unapproved fallback.
- Extended Archimedes inspection, system inventory, and Panopticon to show the
  transport state without exposing secrets.

## Verification

Focused Kernel 30 and Kernel 33 tests passed:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./internal/pantheon -run 'Kernel(30|33)' -count=1
```

The test provider received only minimized entity aliases. The local result
rehydrated the supplied names, and the Bifrost receipt recorded network
activity without containing the test credential. A second test proved that an
unregistered Bifrost capability fails before network access.

No real provider was contacted. No credential was added to the machine.

## Explicit user boundary

Clearing ChatGPT history is not the credential boundary for Pantheon. A real
provider credential should never be pasted into a chat or placed in a Vault
note. It belongs in a local secret mechanism attached to `pantheon.service`,
and Pantheon should send only the minimized packet produced by its own privacy
gate.

## Remaining work

The live system still needs a human-selected provider, terms/retention review,
exact Bifrost capability registration, and one deliberate sanitized live
exchange. Switchyard is not installed or required for this increment.
