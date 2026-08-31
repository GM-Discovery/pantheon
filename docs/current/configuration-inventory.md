---
type: canonical-configuration-inventory
status: active
updated: 2026-08-23
---

# Configuration Inventory

The canonical operator configuration is the versioned JSON file
`~/pantheon-runtime/config/pantheon.json`, currently schema version `1`.
`pantheon show config` prints the resolved values and labels defaults versus
user values. Secrets and the control token are omitted. Configuration is not
authority: grants, Prometheus, Future, Mercury, and Bifrost still enforce
their own boundaries.

## Active surfaces

Current values below are the repository's configured values on 2026-08-17;
paths are installation-specific and are included so the GUI can distinguish a
configured path from a default.

| Technical setting | Human label and purpose | Current value | Default / valid values | Restart | GUI / authority |
| --- | --- | --- | --- | --- | --- |
| `schema_version` | Configuration format | `1` | `1` only | yes | informational; no approval |
| `runtime.listen` | Local listener | `127.0.0.1:8765` | fixed approved localhost listener | yes | system information; restricted to local |
| `paths.vault` | Vault location | `/home/brick/Documents/Obsidian Vault` | empty/unconfigured | yes | safe display; path access remains component policy |
| `paths.roadmap` | Governing roadmap | configured Vault roadmap | empty/unconfigured | yes | safe display |
| `paths.intake` | Incoming source folder | `/home/brick/Pantheon Intake` | `~/Pantheon Intake` | next scan/reload | safe display; writes are bounded deterministic work |
| `paths.father_time_chronicles` | Chronology destination | default beneath Vault | Vault `00 Pantheon Core/Chronicles/Father Time` | yes | safe display |
| `paths.raw` | Source-faithful Raw root | `/home/brick/Documents/Obsidian Vault/80 Pantheon Raw` | beneath Vault | yes | safe display |
| `paths.apollo_reports` | Cited Apollo reports | default beneath Vault | Vault `70 Pantheon Reports/Apollo` | yes | safe display |
| `time.timezone` | Human time zone | machine local | empty or valid IANA zone | no | safe display |
| `retention.raw_input_days` | Raw retention policy | `30` | positive bounded integer | no; policy is not armed until verified Markdown | significant; never silently deletes |
| `models.apollo.model` | Local reasoning model | `mistral-small3.1:latest` | non-empty local model identifier | yes | significant; no approval by itself |
| `models.apollo.endpoint` | Local model endpoint | Docker-host Ollama endpoint | approved local host only; credentials rejected | yes | significant; network/privacy impact shown |
| `models.archimedes.model` | Frontier model compatibility identifier | empty | model name only; no authority | no external transport by itself | informational |
| `archimedes.enabled` | External transport opt-in | `false` | must be explicitly true; live route remains Bifrost-gated | restart | significant; possible data egress |
| `archimedes.provider` | Provider protocol | `local-fake` | `local-fake` or `openai-compatible` | restart | significant |
| `archimedes.endpoint` | Provider endpoint | empty | credential-free HTTPS; loopback HTTP only for tests | restart | significant; network/privacy impact |
| `archimedes.credential_env` | Credential variable name | `PANTHEON_ARCHIMEDES_API_KEY` | variable name only; secret never stored by Pantheon | service restart | secret reference only |
| `archimedes.capability` | Bifrost egress capability | `archimedes-provider` | exact registered, approved, enabled capability | restart | significant; explicit egress approval |
| `apollo.*` | Apollo budgets and citation policy | rounds `12`, sources `64`, context `200000`, duration `900`, citations `true` | positive bounded values; citations boolean | yes | significant; no authority |
| `apollo.external_research.enabled` | Public research request boundary | `false` | boolean; does not itself execute research | yes | restricted when enabled; Archimedes remains separate |
| `ingestion.whisper.enabled` | Local Whisper availability | `true` | boolean | yes | safe/routine |
| `ingestion.whisper.binary` | Whisper executable | configured local path | empty or local executable | yes | informational; local-only |
| `ingestion.whisper.model` | Whisper model | configured local large-v3 path | empty or local model path | yes | informational; local-only |
| `ingestion.whisper.language` | Whisper language mode | `auto` | `auto` or processor-supported language | yes | routine |
| `ingestion.media.converter` | Video/audio converter | empty | empty or explicit local converter | yes | significant; video handling remains deferred without it |
| `rumor.*` | Audio intake | enabled; one audio watch path; retry `3`; stability `1`; poll `2`; handoff `intake` | bounded values; `on_complete=intake` | watcher observes after reload/restart | routine except changing paths or retention |
| `mercury.mode` / `agent` | Flexible transport labels | `hermes` / `Terra` | labels only | yes | informational; no authority |
| `mercury.binary` | Optional external Mercury helper | empty | empty or local executable | yes | significant; separate adapter approvals apply |
| `mercury.deterministic_fallback` | Exact local order fallback | `true` | boolean | yes | significant; no external egress |
| `mercury.round_on_cron_wake` | Existing controlled wake hook | `true` | boolean; no scheduler is created here | yes | informational until a scheduler calls it |
| `bifrost.enabled` | Governed external-capability bridge | `true` | boolean | yes | significant; capability use still needs approval |
| `bifrost.mail.*` | Bounded Gmail read crossing | `gmail` / `INBOX` / `gmail-mail` / `hourly` / `50` | non-empty account/mailbox/id; schedule is a contract; max `1..250` | yes | restricted when invoked; read-only adapter |
| `mail.enabled` / `adapter` | Explicit mail intelligence path | `true` / `himalaya` | `himalaya`, `gmail`, or reserved `mcp` | no | significant; still subject to Prometheus and Bifrost |
| `mail.account` / `inbox` | Existing personal mail source | `gmail` / `INBOX` | non-empty configured identity and mailbox | no | informational source boundary |
| `mail.backlog_enabled` / `batch_size` / `direction` | Manual backlog policy | `true` / `100` / `newest_to_oldest` | batch `1..100`; direction fixed | no | routine bounded processing |
| `mail.mark_read_policy` / `attachment_policy` | Mail safety policy | `standing_authority_only` / `approval_required` | mark-read `standing_authority_only` or `never`; attachments approval-required | no | restricted; understanding never implies authority |
| `mail.action_capability` | Separate Bifrost write-capability identity | `gmail-mail-actions` | non-empty capability ID; separately registered and approved | no | restricted; presence is not permission |
| `zodiac.enabled` | Run the deterministic schedule keeper | `true` | boolean; no language reasoning or authority creation | yes | significant; Prometheus and Future still apply |
| `zodiac.default_timezone` | Human schedule timezone | `America/Los_Angeles` | valid IANA timezone | no | significant |
| `zodiac.personal_projection` | Personal/family calendar policy | `apple` | `apple`, `google`, `both`, or `none` | no | restricted policy; adapter approval remains separate |
| `zodiac.work_projection` | Work calendar policy | `google` | `apple`, `google`, `both`, or `none` | no | restricted policy; adapter approval remains separate |
| `zodiac.work_hours_start` / `work_hours_end` | Ordinary work-hour boundary | `09:00` / `17:00` | local `HH:MM` | no | routine policy |
| `zodiac.brief_time` | Situation Brief due time | `07:00` | local `HH:MM`; Zodiac does not compose prose | no | significant schedule policy |
| `zodiac.brief_catch_up_policy` | Missed daily brief behavior | `run_once` | `skip`, `run_once`, `reconcile`, or `needs_you` | no | significant |
| `zodiac.conflict_policy` | Calendar conflict behavior | `needs_you` | fixed until an explicit deterministic rule exists | no | restricted safety boundary |
| `zodiac.wake_seconds` | Internal schedule evaluation interval | `30` | bounded `5..86400` seconds | yes | routine |
| `zeus.enabled` | Enable machine-root inventory and dry-run sanitation | `true` | boolean; disabled by default in examples | yes | significant; no deletion in Kernel 29 |
| `zeus.roots` | Explicit machine-owned roots | Go cache and `scripts/__pycache__` | absolute paths; unknown roots remain protected | yes | significant; source/Vault roots are prohibited by policy |
| `zeus.quarantine_path` | Later recoverable quarantine destination | `/tmp/pantheon-zeus-quarantine` | absolute path; not created by Kernel 29 | yes | informational until an exact Future/Mercury order exists |

## Reserved or informational namespaces

The typed schema preserves `meta`, `present`, `past`, `weave`, `future`,
`prometheus`, `notifications`, `panopticon`, `setup`, and `organs` so
future kernels have stable names. Their presence does not activate behavior.
There is no active operator setting for live transcription or plugins as an
autonomous installer. Kernel 30's local fake exchange is available only
through an exact Archimedes order. Kernel 33 provides a disabled-by-default
external transport; it can proceed only through an exact configured Bifrost
capability after local minimization and the privacy gate. Zeus settings activate
only deterministic inventory and dry-run receipts; Zodiac settings activate
deterministic scheduling only. Neither installs connectors or grants external
write authority.

Kernel 32 action drafts are not a configuration switch. A completed Apollo
report may create a typed local draft; accepting it is a separate control-bound
operation. Calendar provider presence and email-draft use remain governed by
Seshat, Future, Bifrost, and Mercury.

## Safe configuration contract

- Validate with `pantheon config validate`.
- Inspect with `pantheon show config`.
- Migrate legacy flat configuration only with the explicit migration command.
- Endpoint credentials are rejected by validation; credentials belong in a
  separate mechanism and are never included in resolved output.
- Configuration changes do not create grants. Actions with external effects
  still pass Prometheus, authority, Future, Mercury, and/or Bifrost.
