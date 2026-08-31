# Pantheon licensing

## Project license

Pantheon's own source code is licensed under the GNU Affero General Public
License, version 3 (AGPLv3), SPDX identifier `AGPL-3.0-only`. See the root
[`LICENSE`](../LICENSE) and [`NOTICE`](../NOTICE) files.

AGPLv3 permits commercial use, commercial distribution, paid installation,
paid support, and paid hosting. It also preserves the corresponding-source
obligations for modified versions made available to users over a network.
Those obligations apply to the covered Pantheon source, not automatically to
independent tools or assets that Pantheon invokes.

The AGPL license does not grant rights to use the Pantheon name or branding as
an indication of official support. A future trademark policy should define
what counts as an official Pantheon distribution.

## Dependency boundary

Pantheon is not a relicensing mechanism for its dependencies. Every release
must keep the applicable notices and license terms for the exact software,
models, and media assets it distributes or instructs an operator to obtain.

| Component | Role | Current licensing boundary |
| --- | --- | --- |
| Go module graph | Runtime and storage libraries | Individual module licenses; current graph is primarily BSD/MIT-style, with SQLite's separate public-domain dedication. |
| `whisper.cpp` | Local speech recognition | MIT; preserve its upstream notice when bundled. |
| Whisper model assets | Speech-recognition weights | Follow the license and notice shipped with the selected asset. |
| Silero VAD | Voice-activity detection | MIT; preserve its upstream notice when bundled. |
| libopus | Native Opus packet decoding for the Linux BLE receiver | BSD; external shared library used through the system ABI. |
| `pyannote.audio` and related packages | Speaker diarization | MIT for the packages currently audited. |
| `pyannote/speaker-diarization-community-1` | Diarization model | CC BY 4.0 and model-card terms; attribution and gated-access conditions apply. |
| FFmpeg | Audio/video normalization | External dependency. Effective terms depend on the build configuration; GPL-enabled builds must not be silently represented as LGPL-only. |
| Ollama | Local model runtime | External dependency; retain Ollama's own license and notices. |
| BlueZ / `gatttool` | Linux BLE transport for the Omi receiver | External GPL-licensed system tooling invoked as a process; not bundled by Pantheon. |
| Language-model weights | Local reasoning and extraction | Each selected model has its own license; verify before redistribution. |

## Distribution rule

The safest initial distribution is:

1. distribute Pantheon's source under AGPLv3;
2. document external dependencies as installation prerequisites;
3. avoid bundling model weights and FFmpeg until the exact redistribution
   terms have been checked;
4. publish a versioned dependency notice for any installer or binary bundle;
5. keep Pantheon-specific branding separate from the freedom to fork the code.

This document is an engineering inventory, not legal advice. Recheck the
licenses of every pinned version and asset before a public release or a
commercial customer delivery.
