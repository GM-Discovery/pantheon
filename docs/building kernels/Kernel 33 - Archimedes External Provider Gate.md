# Kernel 33 — Archimedes External Provider Gate

## Status

Implemented and locally verified. External transport remains disabled in the
live configuration.

Kernel 30 proved the privacy gate, temporary aliases, local rehydration, and
Future/Mercury exchange path with a fake provider. Kernel 33 adds the missing
provider seam without selecting or contacting a real provider.

## Contract

```text
Apollo referral
    ↓
curated missive
    ↓
Archimedes alias minimization and deterministic privacy gate
    ↓
Future exact order
    ↓
Mercury seal and Go
    ↓
Bifrost capability authorization
    ↓
OpenAI-compatible provider endpoint (optional)
```

The external adapter receives only the minimized missive. It does not receive
Vault paths, source references, the translation map, raw transcripts,
credentials, tools, or permission to widen the request.

The provider response is untrusted intelligence. It cannot create a Future
proposal, Mercury order, calendar event, email action, notification, tool call,
or new route.

## Configuration

The canonical configuration contains an opt-in block:

```json
"archimedes": {
  "enabled": false,
  "provider": "local-fake",
  "endpoint": "",
  "model": "",
  "route": "passthrough-local-fake",
  "credential_env": "PANTHEON_ARCHIMEDES_API_KEY",
  "capability": "archimedes-provider",
  "timeout_seconds": 60,
  "max_response_bytes": 1048576
}
```

To prepare a real provider later, change the provider to
`openai-compatible`, select an HTTPS endpoint and model, and set
`enabled: true`. Plain HTTP is accepted only for an explicit loopback test
endpoint. URLs may not contain userinfo, query parameters, or fragments.

The API credential is never a configuration value. `credential_env` names the
environment variable that the Pantheon user service may read at invocation
time. Store the secret in a user-owned, mode-0600 environment file or another
approved secret mechanism, and attach it to `pantheon.service` through a
systemd user drop-in. Do not paste credentials into ChatGPT, Apollo, Vault
documents, source code, SQLite, receipts, or provider request logs.

## Bifrost requirement

The provider must be registered as a specific Bifrost capability. Registration
does not download or install software. The capability must explicitly declare:

```text
capability_id: archimedes-provider
component: Archimedes
operation: frontier.exchange
network_access: true
sends_data_off_machine: true
authentication_required: true
prohibited_operations:
  - vault.read
  - identity.lookup
  - tool.invoke
  - route.create
```

The capability must be installed, registered, approved for use, and enabled.
If it is absent, disabled, paused by Prometheus, or missing either explicit
egress declaration, no network request occurs.

## Receipts and failure behavior

Every external attempt records a Bifrost invocation receipt containing the
capability, operation, authorization reference, network fact, outcome, and
exchange ID. It never records the API key or authorization header.

The provider client uses a bounded timeout and response size. Non-success HTTP
responses, malformed responses, missing credentials, timeouts, and provider
errors fail the exchange; they do not trigger fallback to another provider or
send a wider missive.

Archimedes still writes its normal exchange manifest, provider-safe response
artifact, rehydrated local result, and retention receipt. The temporary alias
map is destroyed at closure and is never sent to the provider.

## Operator inspection

```text
pantheon show archimedes
pantheon bifrost list
pantheon bifrost show archimedes-provider
pantheon bifrost approvals
pantheon bifrost invocations
```

Panopticon System reports whether the transport is disabled, configured, or
blocked, along with the provider, endpoint host, route, Bifrost capability,
egress state, and exchange receipts. It does not display a credential value.

## Acceptance proof

Kernel 33 tests prove that:

1. the local HTTP provider receives aliases instead of supplied identities;
2. the credential is carried only in the request authorization header;
3. local rehydration restores the meaningful result;
4. Bifrost records the completed egress receipt without the credential; and
5. an unregistered Bifrost capability fails before network activity.

No real provider, Switchyard process, or user credential is required for the
proof.

## Remaining decision

The implementation is ready for a deliberate provider choice. It is not yet a
live external integration. The next operator action, if desired, is to select
the provider's retention and training terms, register the exact Bifrost
capability, configure the service secret outside Pantheon, and perform one
sanitized test exchange. Switchyard remains optional and separate.
