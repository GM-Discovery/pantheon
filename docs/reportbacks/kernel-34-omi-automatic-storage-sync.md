# Operational Increment — Automatic Omi Storage Sync and Libra Forecast

Date: 2026-08-25
Status: IMPLEMENTED — live Omi transport remains device-dependent

## Delivered

Pantheon now has a configurable local Omi storage-sync watcher owned by the
existing `pantheon.service`. When enabled, it periodically asks BlueZ to
reconnect the device and drains bounded groups of stored Omi CV1 ring records
into the configured local audio folder.
Rumor then observes the WAVs and continues its existing local path:

```text
Omi ring → native BlueZ D-Bus storage reader → WAV/raw/metadata
  → Rumor → local Whisper → Intake → Present/Vault → Libra/Apollo report
```

The native D-Bus reader is preferred because it matches the proven Kernel 26
live-audio transport. It uses the ring control INFO notification to discover
sequence bounds because some Omi firmware does not answer a direct status
read. The legacy `gatttool` storage reader remains a fallback only when native
D-Bus bindings are unavailable.

Each sync attempt writes an `omi-sync-*.json` receipt beside the audio. Each
batch retains the WAV, raw ring bytes, and metadata sidecars. Empty storage is
a successful no-op; a connection or transport failure is recorded as a failed
sync and does not claim ingestion.

## Configuration

The current machine configuration enables:

```json
"omi": {
  "enabled": true,
  "device": "FC:92:2A:DB:64:79",
  "adapter": "hci0",
  "address_type": "random",
  "sync_interval_seconds": 900,
  "packet_count": 36,
  "max_batches": 128,
  "output_dir": "/home/brick/Pantheon Audio/Omi BLE"
}
```

The same fields are visible through Panopticon configuration. The device
address and output folder are informational; enablement and bounds are
editable and require a Pantheon restart.

## Libra presentation

Regular Libra reports retain their existing names, destinations, evidence
sections, Apollo citations, and notification behavior. They now include one
deterministic playful `Cosmic forecast` blockquote immediately below the
title. It is presentation-only and does not influence report reasoning,
authority, or actions.

## Verification

- Python Omi ring/parser and sync tests: PASS.
- Repository Go tests: PASS in the approved live-test environment.
- Omi device was already paired and was marked trusted for unattended local
  reconnection.
- A first live legacy-reader attempt timed out during GATT discovery.
- Native D-Bus retries reached the device bus but ended with a D-Bus no-reply
  timeout before INFO; no recording was consumed and each failure receipt was
  written. A direct Bluetooth reconnect succeeded once, but the device is not
  currently staying connectable, so the latest attempt remains transport
  blocked.

The automatic watcher is installed and retries on its configured 15-minute
interval. A completed sync receipt and subsequent Rumor/Intake receipt are
required before treating today's recordings as report evidence.

The runtime applies a bounded SQLite busy timeout so short-lived local overlap
has a chance to clear before a Rumor, Libra, Zodiac, or audit write fails.

## Privacy and authority boundary

This increment performs local device synchronization only. It does not use
n8n, MCP, a frontier provider, calendar, email, or external network egress.
The Omi audio remains under the configured local Rumor/Intake/Vault chain.
