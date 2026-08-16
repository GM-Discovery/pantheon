# Kernel 13 Reportback — Authority, Grants, and Delegation

## Status

**PASS — repository verified August 16, 2026.**

Kernel 13 is complete at the repository boundary. No Kernel 13 live deployment or runtime restart was performed; the backend remains on the previously deployed Kernel 12 binary.

## Predecessor and starting state

- Predecessor implementation commit: `42e76a5 feat: complete Kernel 12 Apollo processing`.
- The tracked repository was clean at start; the user-provided Kernel 13 specification was newly present as an untracked documentation file.
- No constitutional documentation correction was required. Kernel 13 preserves the person-sovereign, intelligence-is-not-authority, Prometheus pause, and Mercury sealed-order principles already recorded by earlier kernels.

## Existing machinery reused

Kernel 13 extends Kernel 11’s `capability_grants`, `person_direct`, `standing_authority`, `parent_order`, order seals, Go Commands, authority checks, pause invalidation, and durable audit records. Mercury remains a transport/execution boundary; it does not grant permission or adjudicate whether a proposal is wise.

## Implemented authority model

- Grants now represent holder, capability, scope, granting authority, parent grant, creation/validity bounds, remaining uses, lifecycle state, delegation constraints, and revocation/update provenance.
- Supported states are `pending`, `active`, `paused`, `expired`, and `revoked`. Expiration is deterministic and preserves the historical row.
- `remaining_uses` is nullable for bounded or standing authority. A finite grant is atomically consumed when Pantheon issues the authorized Go, including through a parent-order authority chain.
- Configured capability availability remains separate from authorization. A configured capability with no matching active grant remains unauthorized.
- Deterministic scope containment supports exact scopes plus explicit bounded families for hierarchical Vault paths, personal calendar/event scopes, email inbox/message scopes, and Internet domains. Unknown containment is rejected.
- Deterministic capability containment is explicit. `calendar.maintain` may narrow to `calendar.read`; unrelated capability expansion such as `email.read` to `email.send` is rejected.
- Delegation requires an explicit `allow_delegation` constraint and preserves/narrows parent capability, scope, validity window, and use limit. Unlimited child duration/uses cannot be derived from bounded parent authority.
- Authority holders are restricted to known deterministic Pantheon components/person; Apollo, Archimedes, generic models, and deity identities cannot hold or grant operational authority.
- A proposed child grant that exceeds Prometheus’s parent authority remains `pending` for escalation. Prometheus approval is accepted only when the parent grant is active, held by Prometheus, explicitly delegable, and contains the child request. Person approval remains available through the local control boundary.
- Authority inspection exposes active and historical grants, holder, capability, scope, limits, lifecycle state, parent chain, and why a requested operation is or is not authorized.

## Persistence and migration

The existing `capability_grants` table is extended additively in place. Legacy rows are preserved and receive deterministic defaults (`holder=prometheus`, `granted_by=person`) when the new columns are added. No parallel authority database or broad migration was introduced.

## Inspection surfaces

- `GET /v1/grants` and `pantheon grants` list current and historical grant state.
- `GET /v1/grants/<id>` and `pantheon show grant <id>` show the grant and authority chain.
- `GET /v1/authority/check?capability=...&scope=...` and `pantheon authority check <capability> <scope>` explain authorization, matching grant, and provenance.
- Local-control grant actions support pending approval, Prometheus-bounded approval, pause, resume, and revocation.

## Model/deity containment

Grant creation remains behind the local control boundary. Deity/model holders and grantors are rejected. Untrusted structured orders without person, standing, or valid parent authority remain ruling/unauthorized and cannot be sealed. Apollo output remains a proposal/evidence source; it cannot create a grant, seal an order, issue Go, or invoke an operational capability by itself.

## Automated verification

The Kernel 13 tests cover:

1. deity/model grant and order containment;
2. active exact grant authorization;
3. paused, expired, and revoked denial;
4. one-use grant exhaustion across two orders;
5. deterministic time-window expiration;
6. capability and scope narrowing/expansion rules;
7. explicit delegation and parent/child provenance;
8. Prometheus approval within delegated authority;
9. out-of-delegation pending/escalation and Prometheus denial;
10. authority inspection and historical grant representation;
11. additive extension of legacy Kernel 11 grant rows;
12. preservation of Kernel 11 seal, Go, pause, invalidation, and Mercury idempotency behavior.

Verification completed successfully:

```text
gofmt -w $(rg --files -g '*.go')
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
GOCACHE=/tmp/pantheon-go-cache go build -o /tmp/pantheon-kernel13 .
```

Build artifact: `/tmp/pantheon-kernel13`.

## Known limitations and non-goals

- Future’s active-state administration and planning consumer remain Kernel 14 work; Kernel 13 provides the authority infrastructure only.
- No generalized RBAC model, GUI, recurring scheduler, Archimedes execution, or new cryptographic layer was added.
- The Prometheus policy for deciding which permission changes are “meaningful” remains intentionally narrow: the deterministic machinery supports approve, deny, and pending/escalate outcomes, but does not claim to encode comprehensive policy discretion.
- No Kernel 13 live deployment, restart, or live operational smoke test was performed.

## Files changed

- `internal/pantheon/authority.go`
- `internal/pantheon/daemon.go`
- `internal/pantheon/mercury.go`
- `internal/pantheon/cli.go`
- `internal/pantheon/kernel13_test.go`
- `docs/building kernels/Kernel 13 — Authority, Grants, and Delegation.md` (user-provided specification retained)
- `docs/reportbacks/kernel-13-authority-grants-delegation.md`

## Report expiry conditions

This report expires if grant lifecycle/state semantics, deterministic scope or capability containment, delegation validation, authority provenance, Prometheus escalation controls, grant-use consumption, Kernel 11 seal/Go behavior, persisted grant schema, or the live deployment state changes. It also expires when Kernel 14 begins consuming this infrastructure in a way that changes the authority boundary.
