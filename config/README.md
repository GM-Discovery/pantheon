# Per-user Pantheon configuration

Copy `pantheon.example.json` to `~/pantheon-runtime/config/pantheon.json` and fill in the required Apollo model. The canonical file uses `schema_version: 1` and nested namespaces. A partial installation is valid when optional organs are absent.

- `schema_version`: current configuration schema, currently `1`.
- `runtime.listen`: fixed approved daemon listener, `127.0.0.1:8765`.
- `paths.vault`: the directory Past and Weave may search, read-only.
- `paths.roadmap`: governing context for the normal reasoning command.
- `paths.intake`: configurable Intake tree; absent means the per-user default `~/Pantheon Intake`.
- `paths.father_time_chronicles`: yearly Father Time Markdown directory; absent means beneath `paths.vault/00 Pantheon Core/Chronicles/Father Time/`.
- `paths.raw`: Kernel 10 durable source-faithful Markdown root; absent means `paths.vault/80 Pantheon Raw`.
- `retention.raw_input_days`: raw-input policy value, default `30`; Kernel 9 does not arm this clock or delete files.
- `time.timezone`: optional IANA timezone for Father Time; absent means the machine's local timezone.
- `ingestion.whisper`: optional local-only Whisper settings. Missing binaries do not invalidate the rest of Pantheon.
- `ingestion.media.converter`: optional local media converter, normally FFmpeg, used for video audio extraction.
- `models.apollo.model`: the local reasoning model (for example, `mistral-small3.1:latest`).
- `models.apollo.endpoint`: local Ollama chat endpoint, normally `http://127.0.0.1:11434/api/chat`.
- `models.archimedes.model`: reserved and inactive in Kernel 8; naming a model grants no access or authority.
- `policy.yaml`: human-readable authority declaration; it does not silently grant capabilities. Code remains the enforcement point.
- `vault-permissions.yaml`: proposed logical Vault access map. It is not an OS-level permission change.

Use `pantheon config validate`, `pantheon show config`, and `pantheon config path` for deterministic inspection. Use `pantheon config migrate` to preview conversion from the deprecated flat configuration; migration is explicit and never happens at startup. See [`docs/configuration.md`](../docs/configuration.md) for the full field reference.

## Past

`pantheon past "search terms"` searches only readable Markdown, text, and YAML files below `vault_path`. It returns up to twelve matching lines, each with a vault-relative file path and line number. Past does not write to the vault, call models, or invoke other daemons.
