# Future Authority Administration

Kernel 16 gives Future one narrow responsibility: administer bounded proposals
against existing authority. Future is deterministic. It does not use a model,
plan, interpret Apollo's conclusions, issue Go, or invoke an adapter.

The normal boundary is:

```text
Present -> Past / Weave -> Future context contract -> Mercury -> Apollo
                                                       |
                                      structured proposal -> Future
                                                       |
                        existing grant + Janus eligibility -> sealed Mercury order
```

Until the later Future state engine exists, Apollo's structured result is only
evidence for Future. It is not authorization. The current proposal contract
records the Seshat package code, capability, scope, adapter, payload,
constraints, source report or missive, and optional Janus subject. Future
accepts only package identity that Seshat already registered.

## Authority outcomes

- An active Future-held Kernel 13 grant that contains the requested capability
  and scope permits one exact immutable Mercury order.
- A matching delegable Prometheus grant creates a durable pending child grant
  for Future and a visible Prometheus referral. The pending child has no
  authority until Prometheus approves it.
- Without a delegable Prometheus parent, Future records a durable request for
  the person. A human ruling remains unexecuted until a normal authority grant
  exists.
- A missing, unavailable, or mismatched Seshat package is visible as a bounded
  proposal failure. Seshat manifests cannot declare grants or holders.
- A temporal package requires a Janus subject in an eligible `ready` or
  `active` state. Scheduled, missed, expired, and review-required subjects are
  blocked; Future invents no temporal policy.

Prometheus pause defers Future sealing and Mercury dispatch. Resuming does not
replay a blocked round automatically. A later explicit round evaluates the
durable proposal once, and sealed proposals are excluded from later rounds.

## Inspection

```text
pantheon future round
pantheon show future
pantheon show future proposals
pantheon show future proposal <proposal-id>
```

The corresponding local API is control-token protected for round and proposal
creation, while inspection is read-only:

- `POST /v1/future/round`
- `POST /v1/future/proposals`
- `GET /v1/future`
- `GET /v1/future/proposals`
- `GET /v1/future/proposals/{id}`

Future persists proposals and authority requests so missing authority,
Prometheus referral, human referral, temporal blocking, order identity, and
errors remain inspectable. The durable records contain no hidden chain of
thought and do not imply permission to execute recommendations.
