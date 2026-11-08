# Evidence-backed Obsidian links — implementation reportback

## Current chunk-crawler status

The original note-level linker described below is historical and is not the
current graph crawl. The current crawler operates on the user-authored Nexus
chunk inventory, excludes agent-authored chunks, and writes a separate
derived edge graph without modifying Vault notes.

- User-authored chunks crawled: 14,745
- Agent-authored chunks excluded: 33,460
- Chunk-to-chunk discovery edges: 34,766
- Durable state: `state/weave-links/chunk-crawl.json`
- Graph: `state/weave-links/user-chunk-graph.json`

These are lexical discovery candidates with evidence, not approved facts.

Status: bounded preview complete; all 18 reviewed proposals applied
successfully. The archive-scale workflow is batch-based.

## Implemented

The `weave-links` CLI now provides:

- `pantheon weave-links run --batch-size 5`
- `pantheon weave-links preview --limit 20`
- `pantheon weave-links apply --manifest PATH --select ID1,ID2`
- `pantheon weave-links rollback --journal PATH`

`run` is the normal archive-scale workflow. It snapshots the eligible
conversation sources, processes five at a time by default, applies the
evidence-backed links immediately, and writes durable state to
`state/weave-links/active-run.json`. If the machine reboots, rerunning the
same command resumes the pending batch or the next source cursor instead of
starting the archive over. Each batch has its own journal and can be rolled
back independently.

The workflow reads complete eligible Markdown notes, records SHA-256 content
hashes and heading/line evidence, selects a reproducible varied conversation
sample, ranks deterministic candidates without all-pairs model calls, and
excludes import reports, generated reports, raw transcripts, configured Weave
review material, and the linker's own output from semantic candidate content.

Apply writes only a delimited generated Related notes block. It validates
source and target hashes, resolves targets from real Vault files, rejects
traversal and symlink escapes, writes per-file backups outside the Vault,
records a durable journal, supports deterministic repeat application, and
rolls back only files that still have the tool-written after-hash.

## Real preview

Vault: `/home/brick/Documents/Obsidian Vault`

- Imported conversation notes: 598
- Reviewed sources: 20
- Eligible candidate targets: 21
- Proposals: 18
- Scoped Markdown hashes: 635
- Before/after scoped hashes: identical
- Failures: 1 local-model availability check (no proposal failures)
- Assessment: lexical retrieval only; semantic validation was unavailable
- Preview elapsed time: 293 ms

Artifacts:

- `state/weave-links/preview-20261105T123148.986113919Z.json`
- `state/weave-links/preview-20261105T123148.986113919Z.md`

The configured local model is `ministral-3:14b-instruct-2512-q4_K_M` at
`http://172.17.0.1:11434/api/chat`. The endpoint was unreachable during the
bounded availability check, so these proposals must not be described as
semantic model validation. No model-generated relationship was accepted.

## Verification

- `GOCACHE=/tmp/pantheon-go-cache go test ./... -run 'WeaveLink' -count=1` — passed
- unrestricted `GOCACHE=/tmp/pantheon-go-cache go test ./...` — passed (`internal/pantheon` 58.376s)
- `GOCACHE=/tmp/pantheon-go-cache go vet ./...` — passed
- `git diff --check` — passed
- Real preview — passed; scoped Vault hashes unchanged

The restricted sandbox could not bind the existing localhost `httptest` case;
the unrestricted full-suite run passed, and that existing test was not changed.

## Apply receipt

- Journal: `state/weave-links/journal-20261105T123853.848849652Z.json`
- Source notes changed: 6
- Proposals verified in generated blocks: 18 of 18
- Actions: 6 writes, 0 refusals, 0 failures
- Backups: 6, stored outside the Vault under `state/weave-links/backups/`

Rollback remains available with:

`go run . weave-links rollback --journal /home/brick/pantheon-runtime/state/weave-links/journal-20261105T123853.848849652Z.json`
