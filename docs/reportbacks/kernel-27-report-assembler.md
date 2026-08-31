---
kernel: 27
status: PASS — deterministic event and daily evidence assembly
predecessor: Kernel 26 — Omi Intake Bridge
verified: 2026-08-23
---

# Kernel 27 Reportback — Report Assembler and Daily Continuity

## A. Verdict

Kernel 27 passes its bounded first increment.

The implementation now provides a controlled, replay-safe report assembly
boundary. It selects verified Vault Markdown sources, validates that they stay
inside the configured Vault and outside symlink paths, records each exact
Apollo citation and SHA-256, and writes a durable manifest to the appropriate
human-facing destination.

It does not yet schedule a daily Apollo order or compose the final Situation
Brief prose. Those remain separate follow-on work, as required by the
authority boundary.

## B. Runtime contract

```text
POST /v1/reports/assemble
GET  /v1/reports/assemblies
GET  /v1/reports/assemblies/{assembly-id}
```

The POST route requires the local control token and is blocked while Prometheus
is paused. The CLI equivalents are:

```text
pantheon report assemble --date YYYY-MM-DD
pantheon report assemble --kind event --date YYYY-MM-DD --source <Vault-path>
pantheon report assemblies
```

Daily assembly auto-collects same-day verified Present artifacts, same-day
Apollo reports, and date-prefixed integrated event reports when no explicit
source set is supplied. Event assembly requires explicit sources.

## C. Live acceptance

The deployed Kernel 27 service was active with `ExecMainStatus=0` and MainPID
`3716454` after restart.

The live daily assembly produced:

```text
assembly: assembly-a1dde4c8f66bee7b90bfc4b3d1029ba7
manifest: 10 Operations/Situation Briefs/2026-08-23 Situation Brief — Evidence-fdf4451d4c09.md
sources: 2
body: sha256:29c3b485eedba79589115b8e8343c1ba3880f71ac7e5c32d343af76c49948c42
status: ready_for_apollo
```

The two selected sources were the two same-day Apollo reports already in the
configured Vault. The generated manifest includes their full citation tokens,
source hashes, byte counts, and modification times.

The identical daily request was replayed successfully. It returned the same
assembly ID with `duplicate: true`, and the manifest was not overwritten.

The live event assembly produced:

```text
assembly: assembly-41c6972ae6505ef82dbd9fcb7c45cc65
manifest: 70 Pantheon Reports/Integrated Events/2026-08-23-assembled-event-b6f88fb1029c.md
sources: 3
body: sha256:441f84681f67905d7dde4e6bff566ff86fb69704ac88cd3fe5fd132424939619
status: ready_for_apollo
```

Its source set was the proven Kernel 26 transcript, Present artifact, and
corrected Apollo report:

```text
80 Pantheon Raw/Audio Transcripts/omi-dbus-kernel26-live2--330ce56ccfae235e.md
80 Pantheon Raw/Text/2026/08/Rumor/omi-dbus-kernel26-live2--330ce56ccfae235e.md
70 Pantheon Reports/Apollo/report-fc390110f2bcb7f62919a50bc39d08b9.md
```

## D. Verification

Focused Kernel 27 tests cover:

- creation of a cited manifest in the configured Vault;
- exact source-set and body hash recording;
- duplicate replay without overwrite; and
- rejection of a source reference escaping the Vault.

The complete `go test ./internal/pantheon` suite passed. `git diff --check`
passed. The runtime binary was rebuilt, deployed, and restarted before the
live route checks.

## E. Explicit non-claims

Kernel 27 does not claim that:

- Apollo prose was automatically generated from the daily manifest;
- Zodiac now composes or delivers the Situation Brief;
- a calendar event, email, or notification was created;
- raw Intake material was pruned;
- source importance or identity was inferred by the assembler; or
- Zeus or Archimedes was implemented.

The next safe operating step is to use the manifests against a real
multi-source day, then add a separately authorized Zodiac-to-Apollo daily order
only if that workflow proves useful.
