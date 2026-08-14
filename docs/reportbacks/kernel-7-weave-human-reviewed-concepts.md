# Kernel 7 — Weave: human-reviewed local concept substrate

## Status
Completion candidate, pending final source-policy review, verification, and your explicit commit approval. Kernel 7 now has bounded read-only retrieval, source-policy dry runs, incremental primary/Nexus derived-state refresh, and reproducible local subject reviews. No nightly refresh or semantic graphing is enabled.

## What was built
- A local passage index and source-aware term inventory.
- A local Nexus authorship inventory that separates explicit `nexus_user` contributions from `nexus_agent` context.
- Obsidian-facing review notes under `00 Pantheon Core/Weave/`.
- A first human-reviewed concept card: `Concepts/Bread Standard.md`.
- Human-reviewed signal rules at `config/weave-signal-rules.yaml`.

## Human decisions incorporated
- Nexus user contributions are a separate “Grant conversation contributions” category.
- Assistant text is contextual-only and cannot define personal-pattern ranking.
- Socio-: Stories of Us is a human-promoted major-work concept with a reviewed source order: official rules, drafts, project reference, then bounded adjacent-Vault context.
- Future Past retrieval is intended to stop below 90% confidence once scored retrieval exists; Apollo may explore contextual material to 30% when requested, but remains propositional and cited.
- Structured `approval_required: true` is the future review marker; it never authorizes action.

## Boundaries preserved
- Original Vault and Nexus source notes remain intact.
- No automatic note links, tags, semantic claims, or external calls were made.
- Nexus excerpts on a concept card remain speaker-attributed, contextual-only, and do not affect concept rank or meaning.
- No nightly job has been enabled.

## Executable retrieval and incremental slices
- Added a tested local retrieval ranker for Weave passages. It uses transparent query-term coverage as a provisional score and preserves Socio tie ordering: official rules → drafts → reference → adjacent material.
- Added `POST /v1/weave/retrieve`: bounded to 12 results; returns source path, heading, line, excerpt, five-line local context window, and provisional confidence. Default Past excludes Nexus and generated Weave material.
- Added `POST /v1/weave/dry-run`: a read-only, audited fingerprint comparison reporting new, changed, unchanged, excluded, and contextual source files.
- Human-approved primary baseline: 19 notes, currently 0 new / 0 changed / 19 unchanged. Generated source state is local-only under `state/weave/`.
- Added and tested parser support for explicit `nexus_user` and `nexus_agent` callouts. `pantheon weave-refresh` is CLI-control-token protected, fingerprints Nexus source files, retains unchanged chunks, selectively replaces new/changed chunks, removes stale chunks for deleted sources, atomically writes local derived state, and audits the result. The verified second consecutive refresh reported 0 new / 0 changed / 0 removed. The raw Nexus source notes remain untouched.
- Added reproducible local subject-review configuration (`config/weave-subject-reviews.json`) and derived reviews for Bread Standard and Socio. Each is relevance-only, capped to 12 cited passages, and currently about 100 lines. `pantheon weave-review <subject>` prints the same Markdown directly for an on-demand inspection.
- The confidence values are lexical coverage, not epistemic certainty or semantic authority.

## Next work
1. Conduct final source-policy/diff review and verification; commit only after explicit human approval.
2. A future bounded concept review workflow may add promote/suppress/historical classifications, but it is not required for this deterministic retrieval substrate.
3. Design a thin Obsidian integration layer only after the human-reviewed concept model proves useful.
4. Consider nightly incremental indexing only after outputs are stable and explicitly approved.
