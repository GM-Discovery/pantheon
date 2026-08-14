# Kernel 5 Reportback — Bounded Gmail Envelope Pilot

**Status:** active

**Supersedes:** none. This follows Kernel 4 and retains its authority boundaries.

## 1. Identity and mandate — retained

Pantheon is the executive locus. Mercury is the bounded cross-domain execution role; the current transport implementation uses Himalaya for this read-only email path. No mailbox capability inherits authority from the presence of the Himalaya binary.

## 2. Current kernel surface — revised

Implemented in this increment:

- `POST /v1/email/inventory` performs a read-only envelope inventory only after checking a matching active `email.read` grant.
- The endpoint requires an explicit account and mailbox.
- The endpoint always executes Himalaya with `--page 1 --page-size 50 --json`, regardless of a larger supplied request value.
- The inventory is envelope metadata only. Pantheon does not call `message read`, `message export`, `message move`, flag/tag commands, draft commands, or any send command.
- The selected first account is explicitly `gmail`; a later account addition will require its own explicit grant and request.

## 3. Evidence and reportable state — revised

The inventory audit records the exact bounded scope and outcome but does not duplicate the envelope data into SQLite. The API returns the envelope data to the caller for review. No Vault file is written in this increment.

Exact pilot scope:

```text
account=gmail
mailbox=inbox
page=1
page_size=50
envelopes_only
```

## 4. Information boundaries — retained

The Gmail data remains local to the machine and localhost Pantheon API. It is not sent to Apollo, Hermes, Sol/Archimedes, or any external provider in this pilot. No email body is fetched by this endpoint.

## 5. Risks and unresolved work — revised

- Envelope metadata is still personal data. This pilot must be reviewed before a later message-body or Apollo-context step.
- Mailbox alias behavior is delegated to the existing Himalaya account configuration. Pantheon supplies the canonical `inbox` mailbox and records the exact arguments it requests.
- Tagging, archiving, drafting, and any email-body retrieval remain unimplemented.
- The first real inventory requires a matching active grant, created as an explicit audit event before invocation.
- No sender/folder filtering exists in this narrow first pilot. The hard cap is 50 most-recent Inbox envelopes.

## 6. Expiry register — retained

This report expires if the pilot reads message bodies, writes an evidence file, invokes Apollo, changes the cap, or permits any mail mutation. Such behavior requires a successor reportback and a new grant scope.

## 7. Next reportback instructions — revised

Review the initial 50-envelope output. The next safe decision is whether a selected subset may be read as full bodies and given to Apollo locally, with a written selection rule and an explicit evidence location. Do not add mutation capabilities before that review.
