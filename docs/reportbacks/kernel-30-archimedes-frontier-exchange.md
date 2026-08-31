# Kernel 30 Reportback — Archimedes Governed Frontier Exchange

## Status

**PASS — local-only implementation verified August 24, 2026.**

Kernel 30 now has a working bounded exchange boundary. It does not enable a
real frontier provider, send data off-machine, install Switchyard, or grant a
provider access to the Vault.

## What was built

- `needs_frontier_reasoning` referrals are durable proposals with
  `authority_effect: none`.
- Archimedes orders use the existing Future/Mercury order path with the exact
  capability `archimedes.exchange`.
- Mercury validates the sealed order, authority, Prometheus state, and exact
  local dispatcher before issuing Go.
- The local semantic pass replaces explicitly supplied entities with random,
  exchange-scoped aliases such as `Person#…` and `Organization#…`.
- The deterministic privacy gate detects configured email, phone, filesystem,
  and unreplaced identity values.
- One finding is redacted and recorded; two or more findings block the
  exchange.
- The fake provider receives only the minimized missive and returns an
  untrusted alias-bearing response.
- Rehydration happens locally while the temporary alias map exists. A durable
  result contains the locally meaningful form; the provider response artifact
  does not contain the original identities.
- Temporary missive/response artifacts receive hashes and retention metadata.
- The exchange receipt records the alias-entry count and map destruction
  without persisting the translation map.
- Panopticon System shows local-only Archimedes state, exchange counts, privacy
  boundary, data-egress state, and map-destruction counts.

## Operator surface

```text
pantheon show archimedes
pantheon archimedes referrals
pantheon archimedes exchanges
pantheon archimedes refer --reason <text> --question <text>
pantheon order --recipient archimedes --capability archimedes.exchange --scope <scope> "question"
pantheon mercury round
```

The generic order command remains the authority-bearing step. Archimedes does
not create an order merely because a referral exists.

## Durable inspection

```text
GET /v1/archimedes
GET /v1/archimedes/referrals
POST /v1/archimedes/referrals
GET /v1/archimedes/exchanges
GET /v1/archimedes/exchanges/{id}
```

Exchange detail routes expose hashes, paths, gate states, incident metadata,
provider status, rehydration state, and receipts. They do not expose the
temporary translation map.

## Verification

Focused Kernel 30 tests prove:

- a fake exchange completes through Mercury and rehydrates locally;
- the raw provider response contains aliases rather than original identities;
- the local result restores the original identities;
- the retention artifact records map destruction;
- one privacy finding is redacted and recorded;
- two findings block transport and fail the order;
- a referral remains a proposal and leaves live transport disabled; and
- system inventory reports Archimedes as `local-only` with `data_egress: false`.

The focused command was:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./internal/pantheon -run Kernel30 -count=1
```

The final repository checks also passed:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel30-final .
```

## Explicit remaining boundary

Kernel 30 is complete for the local fake-provider increment. Real provider
transport still requires a later Bifrost capability, a reviewed provider
contract, explicit data-egress approval, and the separate Switchyard
application proof. No live frontier model or Switchyard process is enabled by
this implementation.
