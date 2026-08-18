---
kernel: 21
status: READY FOR HUMAN TESTING — repository verified August 17, 2026
predecessor: 0505c688dd5f5fae619f823105b38afeedc719f2
---

# Kernel 21 Reportback — Panopticon Human Control Surface

Kernel 21 adds the first local browser-facing human control surface without
creating a second runtime. The Panopticon is embedded in the Pantheon Go
daemon and is served at `http://127.0.0.1:8765/` when the existing daemon is
running.

## Implemented

- A bundled HTML/CSS/JS Panopticon is served from the daemon; there is no
  Docker, Node, Electron, Tauri, separate frontend daemon, or external CDN.
- Home leads with the notification banner, prominent red Radio Prometheus
  pause/resume control, current work, cell health, and a real local-time
  Father Time watch. Father Time is presented as a watch, not an occupant or
  scheduler.
- Navigation covers Home, Ingestion / Understanding, Execution / Capabilities,
  Approvals, Notifications, Intake, Rumor, Bifrost, Models, Configuration,
  and System.
- Fates are grouped visibly: Present, Past, and Future; Weave is described as
  a retrieval/relationship tool. Apollo state is shown with citation and
  local-model markers. No hidden chain-of-thought is displayed or persisted.
- The GUI uses Kernel 20's versioned `GET /v1/system/inventory` for live state
  and refreshes it every three seconds. It uses bounded GUI routes for session
  establishment, notifications, typed configuration, Prometheus pause/resume,
  Rumor attention, and Bifrost capability enablement.
- The GUI deliberately does not poll the legacy `/v1/status` route because
  that read route also writes an audit event; using the read-only inventory
  contract avoids creating browser-induced SQLite contention.
- Configuration supports Edit → Validate → Apply. The backend validates the
  complete proposed schema, creates a protected backup, writes atomically,
  updates in-memory resolved configuration, marks restart-required fields,
  and omits secrets. Protected paths and model identity fields are read-only.
- Prometheus GUI pause/resume uses the real persisted control state and says
  plainly that skipped work does not replay after resume. Hard stop is not
  exposed in the browser; it remains a terminal/service-manager path.
- Rumor explicitly communicates local-only Whisper boundaries and deferred
  live transcription. Bifrost shows registration, approval, and enablement as
  separate concepts and does not offer a marketplace or autonomous download.
- A practical `.desktop` launcher is included at
  `contrib/pantheon-panopticon.desktop`; it opens the existing local daemon
  and does not start or replace the service.

## Safety and locality

GUI mutation requests require a local-origin check and an HttpOnly,
SameSite-strict GUI session cookie, or the already existing local control
token. The browser surface is local-only, uses a restrictive content security
policy, does not expose credentials/control tokens/raw audit tables, and does
not make remote or mobile claims.

The CLI remains the complete inspection and forensic surface. The GUI is a
human-facing summary and control surface, not a replacement for deterministic
evidence or an authorization to execute Apollo recommendations.

## Focused verification

`internal/pantheon/kernel21_test.go` covers:

- embedded page and JavaScript asset delivery, security headers, and missing
  asset handling;
- local GUI session creation and Prometheus pause/resume with the no-replay
  result;
- typed configuration validation without a write, followed by backup-backed
  application without exposing the control token; and
- rejection of a non-local/HTTPS GUI origin.

The focused Kernel 21 tests pass:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./internal/pantheon -run Kernel21 -count=1
```

No live runtime restart or deployment was performed for this reportback. The
next verification step is human browser testing against the existing local
daemon, followed by any live-service verification the operator chooses.

Kernel 21 does not begin Aphrodite, add plugins/MCPs, or alter the service
manager state.
