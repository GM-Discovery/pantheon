---
kernel: 26
status: PASS — bounded native Omi bridge and external-intake contract
predecessor: Kernel 25 — Echo to Fame Ingestion
verified: 2026-08-23
---

# Kernel 26 Reportback — Omi Intake Bridge

## A. Verdict

Kernel 26 passes its bounded acceptance proof.

The verified local path is:

```text
Omi native BlueZ transport
  → decoded WAV
  → Rumor stable-file watcher
  → local Whisper transcript
  → Intake occurrence
  → Present/Vault verification
  → cited Apollo report
  → local completion notification
```

The optional n8n-style outer gateway is documented as a future transport
boundary but was not installed or required for this proof. The bridge remains
freeware-only on the Linux side: system BlueZ, D-Bus/GLib bindings, libopus,
and existing Pantheon components.

## B. Durable external-intake contract

Kernel 26 adds and tests:

```text
POST /v1/intake/omi
POST /v1/intake/external
GET  /v1/intake/external
GET  /v1/intake/external/{source_id}?source_kind=omi_memory
```

The contract:

- preserves the original Omi JSON payload byte-for-byte;
- preserves labels, provider metadata, transcript segments, and source IDs;
- writes a deterministic envelope and content hashes;
- creates a durable receipt and Intake occurrence IDs;
- converts available transcript text into the normal Vault path;
- rejects unauthenticated or malformed delivery;
- returns `duplicate` for an identical replay; and
- returns conflict for the same source identity with changed content.

Focused Kernel 26 tests cover preservation, conversion, authorization,
duplicate delivery, conflicting replay, query-secret webhook delivery, and
Omi timestamped transcript conversion.

## C. Native Omi BLE proof

The legacy `gatttool` path connected but emitted no live notifications while
the device was recording. The native BlueZ D-Bus path succeeded against Omi
`FC:92:2A:DB:64:79`.

The successful capture:

- received 1,480 Opus notifications;
- decoded 473,600 PCM samples;
- wrote a valid 29.6-second, 16 kHz, mono WAV; and
- refused to publish empty captures by contract.

The capture was placed at:

```text
/home/brick/Pantheon Audio/Omi BLE/omi-dbus-kernel26-live2.wav
```

The implementation is [scripts/omi_dbus_capture.py](../../scripts/omi_dbus_capture.py).
The legacy diagnostic remains in [scripts/omi_ble_capture.py](../../scripts/omi_ble_capture.py),
but native D-Bus is now the proven live-stream route.

## D. End-to-end evidence

The successful run produced these durable records:

| Boundary | Evidence |
|---|---|
| Rumor source | `rumor-98502234c512f4c58bf32cfde1d915ce` |
| Transcript | `80 Pantheon Raw/Audio Transcripts/omi-dbus-kernel26-live2--330ce56ccfae235e.md` |
| Intake occurrence | `intake-1bc9e606f8803c5ed3db28886b667bf0` |
| Present artifact | `80 Pantheon Raw/Text/2026/08/Rumor/omi-dbus-kernel26-live2--330ce56ccfae235e.md` |
| Apollo report | `70 Pantheon Reports/Apollo/report-fc390110f2bcb7f62919a50bc39d08b9.md` |
| Notification | local notification audit, outcome `completed` |

The first Apollo attempt correctly stopped because Rumor had handed off the
Intake occurrence before Present verification. Running Present explicitly
completed the missing conversion boundary. A focused Apollo rerun then named
the three proposals spoken in the transcript:

1. individual items and alerts for each transcript event;
2. daily aggregation of those items and alerts; and
3. optional Apollo passes per event and over accumulated daily material.

Those are recorded as proposed requirements, not as implemented capabilities.

## E. Report-quality repair

The first report said that explicit requests existed but did not name them.
That was a real report-quality failure. The Apollo protocol is now stricter:

- every concrete request, action item, commitment, decision, or proposal must
  be stated in plain language;
- citations are evidence attached to a statement, not a substitute for it; and
- the full supplied citation token must be copied rather than shortened into an
  invented number.

The daemon was rebuilt, deployed, restarted, and verified after this repair.

## F. What this kernel does not claim

Kernel 26 does not claim that:

- n8n is installed or connected;
- Omi cloud memories, labels, or speaker identities are available through BLE;
- the current capture automatically runs Echo/Fame speaker reconciliation;
- daily item/alert aggregation or two-pass Apollo scheduling is implemented;
- a calendar event, email, or external notification action was executed; or
- Zeus retention pruning or raw-material deletion exists.

The raw WAV, transcript, Intake records, and reports remain inspectable. No
mechanical device modification was required.

## G. Next boundary

The most direct successor is a report assembler that consumes verified Rumor,
Echo, Fame, and Vault evidence and writes two explicit outputs:

1. an event transcript/identity archive; and
2. a cited event report plus a separately identified daily aggregation path.

Archimedes and Zeus are independent next-work tracks. Archimedes should begin
with privacy/redaction and escalation-contract implementation. Zeus should begin
with retention-policy inventory, dry-run sanitation, and auditable receipts;
neither should be allowed to silently expand the report or authority path.
