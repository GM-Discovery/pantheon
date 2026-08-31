# Per-user Pantheon configuration

Copy `pantheon.example.json` to `~/pantheon-runtime/config/pantheon.json` and fill in the required Apollo model. The canonical file uses `schema_version: 1` and nested namespaces. A partial installation is valid when optional organs are absent.

- `schema_version`: current configuration schema, currently `1`.
- `runtime.listen`: fixed approved daemon listener, `127.0.0.1:8765`.
- `paths.vault`: the directory Past and Weave may search, read-only.
- `paths.roadmap`: governing context for the normal reasoning command.
- `paths.intake`: configurable Intake tree; absent means the per-user default `~/Pantheon Intake`.
- `paths.father_time_chronicles`: yearly Father Time Markdown directory; absent means beneath `paths.vault/00 Pantheon Core/Chronicles/Father Time/`.
- `paths.raw`: Kernel 10 durable source-faithful Markdown root; absent means `paths.vault/80 Pantheon Raw`.
- `paths.apollo_reports`: Kernel 12 cited Apollo report root; absent means `paths.vault/70 Pantheon Reports/Apollo`.
- `retention.raw_input_days`: raw-input policy value, default `30`; Kernel 9 does not arm this clock or delete files.
- `zeus.enabled`: explicit opt-in for Zeus inventory and dry-run sanitation; default `false`.
- `zeus.roots`: absolute, explicitly machine-owned roots. The first implementation recognizes Go build-cache roots and Python `__pycache__` roots; unknown roots remain protected.
- `zeus.quarantine_path`: reserved destination for a later recoverable quarantine order. Kernel 29 does not create it or move files.
- `time.timezone`: optional IANA timezone for Father Time; absent means the machine's local timezone.
- `ingestion.whisper`: optional local-only Whisper settings. Missing binaries do not invalidate the rest of Pantheon.
- `ingestion.media.converter`: optional local media converter, normally FFmpeg, used for video audio extraction.
- `mercury.mode` and `mercury.agent`: flexible Mercury labels (`hermes` / `Terra` by default); absence of the optional binary does not block deterministic orders.
- `mercury.binary`: optional local Hermes/Terra executable. Kernel 11 does not install or start it.
- `mercury.deterministic_fallback`: enables the durable structured-order fallback, default `true`.
- `mercury.round_on_cron_wake`: records the safe scheduled-wake integration point; Kernel 11 provides `pantheon mercury round` but does not install a new scheduler.
- `apollo`: bounded local reasoning budgets and citation policy. `external_research.enabled` defaults to `false`; Kernel 12 does not grant Apollo Internet access by configuration alone.
- `models.apollo.model`: the local reasoning model (for example, `mistral-small3.1:latest`).
- `models.apollo.extraction_model`: the bounded transcript-extraction model (for example, `granite3.3:8b`); if omitted, it falls back to `models.apollo.model`.
- `apollo.context_tokens`: maximum Ollama prompt-plus-output context for Apollo and Fame (default `8192`).
- `apollo.max_output_tokens`: maximum Apollo report output tokens (default `2048`); Fame extraction remains bounded to `768`.
- `omi.enabled`: enables the local automatic Omi CV1 storage-sync watcher. It drains bounded stored-record batches into the Rumor audio watch tree; it does not send data to a provider.
- `omi.device`, `omi.adapter`, and `omi.address_type`: the already-paired local Bluetooth device identity used for storage sync.
- `omi.sync_interval_seconds`, `omi.packet_count`, and `omi.max_batches`: explicit bounds for retry cadence and one drain attempt.
- `omi.output_dir`: local folder for WAV, raw-ring, metadata, and sync-receipt artifacts; it should also be beneath a Rumor watch path.
- `models.apollo.endpoint`: local Ollama chat endpoint, normally `http://127.0.0.1:11434/api/chat`.
- `models.archimedes.model`: compatibility model identifier for Archimedes; it grants no access or authority.
- `archimedes.enabled`: external transport opt-in; default `false`.
- `archimedes.provider`: `local-fake` by default, or the explicit `openai-compatible` provider when a real route is deliberately prepared.
- `archimedes.endpoint`: credential-free HTTPS provider endpoint; loopback HTTP is allowed only for local tests.
- `archimedes.model`, `archimedes.route`: provider model and exact configured route.
- `archimedes.credential_env`: environment-variable name only; the credential itself must never be placed in JSON, Vault, SQLite, or chat.
- `archimedes.capability`: exact Bifrost capability required before any external request.
- `archimedes.timeout_seconds`, `archimedes.max_response_bytes`: bounded provider request limits.
- `policy.yaml`: human-readable authority declaration; it does not silently grant capabilities. Code remains the enforcement point.
- `vault-permissions.yaml`: proposed logical Vault access map. It is not an OS-level permission change.

Use `pantheon config validate`, `pantheon show config`, and `pantheon config path` for deterministic inspection. Use `pantheon config migrate` to preview conversion from the deprecated flat configuration; migration is explicit and never happens at startup. See [`docs/configuration.md`](../docs/configuration.md) for the full field reference.

## Past

`pantheon past "search terms"` searches only readable Markdown, text, and YAML files below `vault_path`. It returns up to twelve matching lines, each with a vault-relative file path and line number. Past does not write to the vault, call models, or invoke other daemons.
