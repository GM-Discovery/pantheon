# Per-user Pantheon configuration

Copy `pantheon.example.json` to `~/pantheon-runtime/config/pantheon.json` and fill in only the fields you use. Empty fields are intentional: Pantheon can be configured incrementally.

- `ollama_url`: local Ollama chat endpoint, normally `http://127.0.0.1:11434/api/chat`.
- `local_model`: the local reasoning model (for example, `mistral-small3.1:latest`).
- `frontier_model`: an optional stronger remote model. Naming it here grants no access or authority.
- `vault_path`: the directory Past may search, read-only.
- `roadmap_path`: optional governing context for the normal reasoning command.
- `policy.yaml`: human-readable authority declaration; it does not silently grant capabilities. Code remains the enforcement point.
- `vault-permissions.yaml`: proposed logical Vault access map. It is not an OS-level permission change.

## Past

`pantheon past "search terms"` searches only readable Markdown, text, and YAML files below `vault_path`. It returns up to twelve matching lines, each with a vault-relative file path and line number. Past does not write to the vault, call models, or invoke other daemons.
