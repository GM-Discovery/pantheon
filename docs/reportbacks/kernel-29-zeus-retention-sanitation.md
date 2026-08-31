---
kernel: 29
status: PASS — deterministic inventory and dry-run sanitation
predecessor: Kernel 28 — Zodiac / Libra configurable reporting
verified: 2026-08-24
---

# Kernel 29 Reportback — Zeus Retention Inventory and Sanitation

## A. Verdict

The bounded Zeus increment passes. Zeus is now a deterministic daemon surface
that inventories only explicitly configured machine-owned roots, hashes regular
files, records durable candidate/run receipts, and reports what a later
recoverable quarantine would do.

No file was quarantined or deleted.

## B. Live proof

The deployed Pantheon binary was verified by service-manager inspection:

```text
service: active
MainPID: 3783635
sha256: 413be875e08a383a7ca888d3bf1de7dff3d031f711a36be44da6b2163d983104
```

The final live dry-run receipt was:

```text
run_id: zeus-run-1787554961787247409
mode: dry_run
policy_snapshot_sha256: sha256:881952a930e4db4461d5a6711959792d828e9f8030cd6e5be28a331dc5b668d3
candidate_count: 6
eligible_count: 6
protected_count: 0
state: completed
```

The configured roots were evaluated independently:

| Root | Result |
| --- | --- |
| `/home/brick/pantheon-runtime/scripts/__pycache__` | six reproducible Python bytecode files, 83,844 bytes, eligible for later quarantine |
| `/tmp/pantheon-go-cache` | absent in the live service namespace; recorded as missing, not guessed or invented |

The six Python cache files remained present after the dry-run. The configured
`/tmp/pantheon-zeus-quarantine` path was not created.

## C. Interfaces

The operator and Panopticon surfaces are:

```text
GET  /v1/zeus
POST /v1/zeus/inventory
POST /v1/zeus/dry-run

pantheon show zeus
pantheon zeus inventory
pantheon zeus dry-run
```

The canonical system inventory now reports Zeus as `ready` when explicitly
enabled and includes its latest run summary. Panopticon’s System view shows
the same summary and states that exact candidate details remain on the local
Zeus inspection route.

## D. Implementation boundary

Zeus recognizes only the first explicit cache policies:

- Go build-cache roots named `pantheon-go-cache` or containing `go-cache`;
- Python roots named `__pycache__`.

Unknown configured roots remain protected. Symbolic links, non-regular files,
paths outside their configured root, source material, Vault, Intake, reports,
Echo/Fame records, and active work are not sanitation candidates.

Every candidate has a policy ID, owner, exact relative and local absolute path,
size, modification time, content hash, state, reason, intended action, and
run/receipt identity. The database stores the complete candidate set even
though API responses preview at most 100 candidates.

## E. Verification

- focused Kernel 29 tests: PASS;
- `go test ./...`: PASS;
- `go vet ./...`: PASS;
- Panopticon JavaScript syntax check: PASS;
- deployed service restart and binary hash verification: PASS;
- live Zeus dry-run with durable receipt and no file movement: PASS.

## F. Remaining boundary

Kernel 29 does not create authority, infer garbage from age or content, follow
symlinks, quarantine automatically, remove files, or schedule itself. A later
recoverable quarantine requires an exact Future/Mercury order naming the sealed
candidate identities and paths. Raw Omi material and human Vault material
remain outside Zeus’s diet.
