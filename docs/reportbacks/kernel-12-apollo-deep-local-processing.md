# Kernel 12 Reportback — Apollo Deep Local Processing

## Status

Kernel 12 implementation is in progress. This is not a PASS report and does not claim live deployment or live Apollo verification. The runtime was not restarted for this work.

## Orchestration boundary

The governing flow is:

```text
Present + Past/Weave prepared context + original request
        ↓
Future context contract when Kernel 13 exists
        ↓
Mercury sealed apollo.reason order + Go
        ↓
Apollo local reasoning
        ↓
durable cited report + Future-ready structured result
        ↓
awaiting Future consumption
```

Kernel 12 does not implement Future’s active state, tool-call selection, or child-order generation. Apollo’s structured result is evidence for Future, not authorization. Mercury transports and records the result; it does not interpret Apollo conclusions into future work.

## Implemented increment

- Go package layout now has root `main.go` as the executable entrypoint and implementation/tests under `internal/pantheon/`; `testing/` remains as an explicit organization note for future black-box tests.
- Apollo is an explicit Mercury recipient/capability: `recipient=apollo`, `capability=apollo.reason`.
- Apollo dispatch requires the existing sealed order, matching seal digest, current unconsumed Go, unpaused Prometheus state, and an approved local endpoint/model.
- Apollo order payloads accept `prepared_context`, exact verified Present occurrence references, source references within the configured Vault, and optional `future_context` as a contract input only.
- Verified Present artifacts are bounded into quoted local evidence with human-readable citations. Source text is treated as untrusted evidence, not authority.
- A local Apollo model call produces a Markdown report beneath `70 Pantheon Reports/Apollo`, with order/seal/Go/attempt/model provenance and body hash.
- A durable `future_ready_result` is persisted with citations and `awaiting: future_consumption`; no Future state is mutated.
- Read-only inspection exists through `pantheon show apollo`, `pantheon show apollo reports`, and `pantheon show apollo report <id>`.
- Apollo external research remains disabled by default and is not implemented as a network adapter.

## Automated verification

The refactored repository and first Apollo increment pass the focused/full tests with local loopback access:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./...
```

Coverage currently includes local endpoint rejection, invalid Go rejection, sealed Apollo Present dispatch, cited report creation, Future-ready result persistence, and one-shot Mercury idempotency. The full Kernel 12 retrieval-loop, citation-correction, prompt-injection, mid-run pause, referral, and human citation review work remains pending.

## Explicit non-claims

- No live Apollo deployment or runtime restart was performed.
- No Future active-state engine was implemented.
- No Internet/public-research adapter was implemented or enabled.
- No Archimedes/frontier reasoning was added.
- Kernel 10 Whisper live verification remains separate and open.

## Next safe increment

Add bounded iterative Past/Weave retrieval and deterministic citation validation/correction around the same Mercury-only Apollo adapter, without creating Future state or executing Apollo recommendations.
