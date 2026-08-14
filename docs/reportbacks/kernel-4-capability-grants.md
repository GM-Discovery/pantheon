# Kernel 4 Reportback — Capability Grants and Mercury Email Boundary

**Status:** active

**Supersedes:** none. This report follows the active Kernel 3 reportback and retains its sections unless revised here.

## 1. Identity and mandate — retained

Pantheon remains the executive locus. Apollo reasons locally; Mercury carries only bounded work; Pantheon authorizes. A capability name does not itself grant authority.

## 2. Current kernel surface — revised

Implemented in this increment:

- Persistent SQLite `capability_grants` records.
- `GET /v1/grants` lists current grant records.
- `POST /v1/grants` creates an inspectable active grant only for a capability that is present in the configured registry.
- A `self.change` grant is rejected unless it includes a readable intent summary, non-empty diff, and `human_approved: true`.
- `pantheon grants` is a read-only CLI view of registered grants.
- The registry now declares the narrow Mercury email capabilities: `email.read`, `email.tag`, `email.archive`, and `email.draft`.
- No `email.send` capability exists.

A declaration or grant does not yet invoke Himalaya or Hermes. No mailbox was listed, read, changed, moved, drafted, or sent as part of this milestone.

## 3. Evidence and reportable state — revised

Every created grant receives a stable ID, capability, scope, intent summary, optional diff, human approval flag, status, and creation timestamp. Grant creation also receives a normal audit event.

The grant contract makes it possible to inspect why an operation was allowed before executing a future Mercury action.

## 4. Information boundaries — retained

Archimedes remains outside the filesystem/Vault boundary. The Vault permission manifest remains declarative. Apollo's full authorized local context and the private identity map remain local-only. No external provider configuration or packet transport was added.

## 5. Risks and unresolved work — revised

- Grants are recorded but not yet consulted at a real Mercury execution boundary; no email action implementation exists yet.
- Grant revocation, expiration, schedules, one-time versus standing grants, and binding a grant to an exact message selection remain unimplemented.
- `self.change` validates the presence of an intent/diff/approval record, but does not apply a change. Applying a self-change requires a separate reviewed proposal-and-apply flow.
- The configured Himalaya binary was verified present, but its account configuration was intentionally not inspected and no mailbox call was made.
- A user-approved scope is required before creating the first real email grant. Pantheon must not infer a mailbox, sender set, folder, label, or message-selection policy.

## 6. Expiry register — retained

This report is accurate only while `capability_grants` reflects the daemon's grant model. A successor must state whether this contract remains active, was revised, or was superseded by executable policy evaluation.

## 7. Next reportback instructions — revised

The next safe increment is a read-only, bounded Mercury email inventory behind an explicit `email.read` grant. It must establish the mailbox account, folder, strict message/time cap, and resulting evidence format before listing any mail. It should not add tagging, archiving, or drafting until the read-only evidence packet is reviewed.
