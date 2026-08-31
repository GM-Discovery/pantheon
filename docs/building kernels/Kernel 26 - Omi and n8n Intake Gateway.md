# Kernel 26 — Omi and n8n Intake Gateway

## Mission

Establish a governed external-intake bridge that can receive Omi memories and,
when useful, other source types through a compatible outer gateway, preserve
the complete source material, prove its conversion into the Vault, and hand the
resulting evidence to the existing Pantheon processing path.

Kernel 26 is not an Omi summarizer and does not require n8n. An n8n-style
gateway remains an optional outer transport for later webhook/form/API sources;
the current Omi BLE bridge is native Linux software using the system BlueZ,
D-Bus/GLib, and libopus components. Pantheon remains responsible for source
identity, evidence preservation, Vault conversion, Echo/Fame identity, Apollo
reasoning, authority, and downstream execution boundaries.

The target path is:

```text
Omi or another approved source
          ↓
  native bridge or optional gateway
          ↓
  Pantheon Intake envelope
          ↓
 raw payload + transcript + receipt
          ↓
       Present / Vault
          ↓
      Echo / Fame when audio identity is relevant
          ↓
       Apollo integrated report
          ↓
  70 Pantheon Reports / downstream proposals
```

There are two materially different Omi input lanes:

| Lane | What arrives | What Pantheon can prove |
|---|---|---|
| Completed-memory webhook | Omi JSON with memory ID, transcript segments, labels, and structured fields | Original Omi payload plus normalized transcript and Intake/Vault receipt |
| Linux BLE receiver | Live Opus audio packets from the device | Device audio bytes, local WAV provenance, Rumor transcript, and later Echo/Fame evidence |

The BLE lane must not be represented as if it were a completed Omi memory. It
does not contain Omi's cloud transcript, labels, summaries, or participant
claims. It reaches the same durable processing boundary indirectly through
Rumor and local Whisper. A BLE capture is not successful merely because GATT
connected; it must contain audio frames before it is published to the Rumor
watch tree.

## 1. Optional gateway boundary

n8n could receive different webhook, form, API, and file-based inputs without
teaching Pantheon every external transport, but it is not part of the current
installation and is not required for the Omi BLE path. The gateway may
normalize transport details and forward a common Intake envelope.

n8n must not become the source of truth or the reasoning brain. It must not:

- decide what a conversation means;
- discard source fields because Pantheon does not currently index them;
- turn speech into authority;
- write directly into the Vault's canonical derived state;
- bypass Pantheon's Intake identity and audit path;
- invoke calendar, email, or notification actions merely because a source
  contains an action item; or
- silently add a new source type to Pantheon without a registered contract.

For a future webhook source, the preferred boundary is:

```text
external source → optional gateway → authenticated local Pantheon intake endpoint
```

n8n may stage a file for Intake when that is required by the transport, but the
Pantheon handoff must still create the canonical occurrence and receipt.

## 2. Omi as the first source

Kernel 26 begins with Omi completed-memory delivery. The first implementation
should accept a completed memory and preserve, as available:

- the complete original Omi JSON payload;
- Omi memory/conversation identifiers;
- receipt time and source timestamps;
- transcript text and segment metadata;
- Omi summaries and action items;
- labels, tags, and other user-facing metadata;
- participant and speaker metadata as untrusted source claims;
- source URLs or references when supplied; and
- the original payload content hash.

Pantheon should normalize only the fields needed for routing, deduplication,
provenance, and later retrieval. It must not throw away fields simply because
they do not yet have a Pantheon schema. Personal labels and other useful Omi
metadata remain available in the preserved source payload for future indexing.

This is a lossless source-preservation rule, not a requirement that every Omi
field become a first-class Pantheon fact.

## 3. Canonical Intake envelope

Every n8n handoff must carry a small deterministic envelope around the original
source object:

```json
{
  "source_kind": "omi_memory",
  "source_id": "<stable Omi memory ID>",
  "source_event_id": "<provider event ID when supplied>",
  "received_at": "<RFC3339 timestamp>",
  "occurred_at": "<source timestamp when known>",
  "content_sha256": "sha256:<digest>",
  "payload_format": "application/json",
  "payload_path": "<preserved Intake path>",
  "transcript_path": "<preserved transcript path when available>",
  "metadata": {
    "provider": "omi",
    "labels": []
  }
}
```

The original provider object is stored beside or inside the Intake occurrence
without semantic rewriting. Envelope fields are routing/provenance data; they
are not a replacement for the source.

## 4. Idempotency and delivery behavior

The bridge must be safe under webhook retries, n8n retries, process restarts,
and duplicate file delivery.

The canonical duplicate key is the provider source identity when trustworthy,
such as the Omi memory ID. The content hash is a secondary identity check. A
repeated delivery must produce a receipt showing that the source was already
known rather than a second transcript, report, or event.

The bridge must:

1. authenticate the inbound request or its n8n handoff;
2. validate the minimum envelope fields;
3. persist the original payload before acknowledging durable acceptance;
4. create or find the Intake occurrence;
5. record the payload and transcript hashes;
6. return a stable receipt for new, duplicate, rejected, or quarantined input;
7. avoid holding the external webhook open while Apollo reasons; and
8. preserve enough state to resume processing after a restart.

Malformed, unauthenticated, or unverifiable input is rejected or quarantined.
It is never silently converted into a plausible transcript or report.

## 5. Raw material and transcript locations

Raw Omi payloads belong in the configured Intake area with other incoming
material. The bridge must not invent a second permanent raw-data silo merely
because n8n is involved.

Transcripts belong with the other preserved transcripts entering through
Intake. Human-readable Markdown filenames should include a stable source/date
identity where available; provider IDs and hashes remain in frontmatter and
machine metadata. Timestamped filenames are acceptable as part of the stable
machine identity, but the visible title should make the event understandable.

The report destination is the configured Vault report root:

```text
70 Pantheon Reports/Apollo
```

The report must link back to the Intake payload, transcript, and conversion
receipt. A report without recoverable source evidence is not a successful
conversion.

## 6. Proof of ingestion and Vault conversion

Kernel 26 is successful only when it can prove both transport acceptance and
human-readable Vault conversion.

The durable proof should include:

```text
source identity
  → n8n delivery receipt
  → Pantheon Intake occurrence
  → raw payload path + SHA-256
  → transcript path + SHA-256
  → Vault conversion/read-back result
  → Apollo report path + source citations
```

The proof must be inspectable without trusting an n8n success message alone.
At minimum, `pantheon show intake`, the relevant source artifact, and the
Apollo report must agree on source ID and content identity.

The preferred report/frontmatter references are:

- source kind and stable source ID;
- n8n receipt or workflow execution reference;
- Intake occurrence ID;
- raw payload path and hash;
- transcript path and hash;
- Vault read-back status; and
- related report ID/path.

## 7. Identity and personal metadata

Omi participant names, labels, and speaker fields are source metadata. They may
help routing and retrieval, but they do not override Echo/Fame identity state.

Where a recording is also processed through Echo and Fame:

```text
Omi speaker claim          = untrusted source metadata
Echo diarization           = anonymous voice segmentation
Fame confirmation          = reviewed Pantheon identity evidence
Apollo person statement    = cited derived interpretation
```

The bridge should preserve all Omi labels for personal use while exposing only
reviewed identity as a Pantheon person fact. Labels may later become a
first-class personal-memory capability without forcing Kernel 26 to guess
their meaning.

## 8. Processing and report handoff

After Intake accepts the source, processing follows the existing boundaries:

```text
Intake → Present → Past / Weave context → Apollo
```

For audio involving human identity, Echo/Fame evidence may be added before
Apollo. The report must keep two contexts visibly separate:

1. primary event evidence: Omi payload, transcript, timestamps, Echo/Fame
   mappings, and exact source citations;
2. bounded continuity context: related Vault/Past/Weave notes, separately
   labeled and cited.

Apollo may produce a report only from evidence it can recover and cite. Omi's
summary or action-item fields are useful source observations, not unquestioned
facts.

## 9. Retention and pruning boundary

Raw material should remain in Intake long enough for the configured retention
policy and downstream conversion proof. Pruning must be deterministic,
auditable, and recoverable according to the declared policy.

The intended future shape is:

```text
retention policy → Zodiac due state → lawful order → Zeus sanitation → receipt
```

At the time of this kernel design, Zeus is a deferred component and the live
Zodiac-to-Zeus pruning path must not be claimed as implemented. Kernel 26 may
define the retention metadata and handoff contract, but it must not delete raw
Omi material or pretend a Zeus order exists until that capability is actually
implemented and authorized.

## 10. Security and authority

The bridge must keep external transport authority separate from Pantheon work
authority.

Required controls include:

- n8n credentials and Omi secrets outside the repository and report body;
- authenticated local handoff to Pantheon;
- source-size and processing bounds;
- replay/deduplication protection;
- explicit quarantine for malformed or suspicious payloads;
- Prometheus pause preventing new downstream processing;
- no direct n8n access to Pantheon control tokens; and
- no calendar, email, notification, or external MCP action from intake alone.

Registering n8n or an Omi credential is not permission to process every source
or invoke every downstream capability. The source contract and permitted
workflow must remain inspectable.

## 10A. Native Omi webhook adapter

The bridge also accepts Omi's completed-memory webhook shape directly:

```text
POST /v1/intake/omi?pantheon_intake_secret=<bridge-secret>
```

Omi's memory trigger sends a JSON memory object containing an `id`, timestamps,
transcript segments, and structured memory fields. The adapter preserves that
entire request body as the raw payload, uses the Omi memory `id` as the stable
source identity, carries Omi's `uid` query value into the envelope metadata,
and converts transcript segments into timestamped Markdown without discarding
speaker labels or personal folder/category labels.

For an Omi webhook URL, put the bridge secret in the
`pantheon_intake_secret` query parameter. A reverse proxy or another controlled
edge may instead translate its authenticated request into the preferred
`X-Pantheon-Intake-Secret` header before forwarding it. The secret must never
be committed to this repository, a report, or a receipt.

This adapter handles completed memory delivery. Omi's real-time transcript and
raw audio streaming triggers are separate future adapters; they must not be
pretended to be completed memories. See Omi's [integration app
documentation](https://docs.omi.me/doc/developer/apps/Integrations) for the
provider-side webhook contract.

## 10B. Native Linux BLE live-audio path

Pantheon also includes a dependency-light Linux receiver for Omi's live BLE
audio stream:

```bash
python3 scripts/omi_ble_capture.py \
  --device AA:BB:CC:DD:EE:FF \
  --output-dir "/home/brick/Pantheon Audio/Omi BLE"
```

The receiver discovers the Omi audio and codec characteristics, subscribes to
the audio notification characteristic, removes the documented three-byte
packet header, decodes Opus through the system `libopus`, and atomically
renames the completed WAV file into the configured Rumor watch tree. Rumor can
then perform the existing local Whisper → Intake handoff.

The legacy receiver explicitly exchanges a larger ATT MTU after connecting.
The current production test path uses the native BlueZ D-Bus GATT interface,
which avoids the unreliable legacy `gatttool` notification path while keeping
the same audio-file handoff contract:

```bash
PYTHONDONTWRITEBYTECODE=1 python3 scripts/omi_dbus_capture.py \
  --device FC:92:2A:DB:64:79 \
  --duration 30 \
  --output "/home/brick/Pantheon Audio/Omi BLE/omi-dbus-live.wav"
```

The native receiver connects through `org.bluez`, synchronizes the Omi clock,
subscribes to the official audio characteristic, removes the documented
three-byte packet header, decodes Opus through system `libopus`, and refuses to
publish an empty WAV. It needs the system Python `dbus` and `gi` bindings,
BlueZ, and `libopus`; it adds no Python package or proprietary runtime.

If the device uses a random BLE address, the native receiver discovers the
paired BlueZ object by address. The Omi mobile app must not hold the device
connection while Linux is connecting. The legacy `omi_ble_capture.py` remains
available as a diagnostic fallback, but a successful `gatttool` connection
alone is not evidence that live notifications work.

This path captures live stream packets only. A connected CV1 may instead expose
already-recorded audio through its ring-buffer storage service. Kernel 26 also
includes the documented reader for that lane:

```bash
python3 scripts/omi_ble_storage_capture.py \
  --device AA:BB:CC:DD:EE:FF \
  --address-type random \
  --packet-count 36 \
  --output-dir "/home/brick/Pantheon Audio/Omi BLE"
```

The storage reader exchanges the MTU, requests ring metadata, reads a bounded
number of records, reassembles unaligned data notifications into 444-byte
records, extracts the packed Opus frames, and writes a WAV plus raw-ring and
metadata sidecars. It sends neither `CMD_RING_ADVANCE` nor `CMD_RING_CLEAR`
explicitly. However, CV1 firmware may checkpoint a successful read as part of
its normal sync behavior, so a completed storage read must be treated as a
device-sync operation and its raw sidecar retained as provenance. The current
operator session can see the Linux Bluetooth controller, but host-level HCI
permissions and a nearby device are required for a real scan or connection.

## 11. Acceptance proof

The minimum end-to-end acceptance case is one completed Omi memory containing a
real conversation:

1. n8n receives the source and produces a durable delivery receipt;
2. Pantheon accepts one Intake occurrence;
3. the complete raw Omi payload is preserved;
4. the transcript is converted into the normal Intake/Vault transcript form;
5. a duplicate delivery creates no second occurrence or transcript;
6. Echo/Fame identity is preserved or clearly remains provisional/anonymous;
7. Apollo cross-checks bounded Vault context and writes a cited report under
   `70 Pantheon Reports/Apollo`;
8. the report links to the raw payload, transcript, Intake occurrence, and
   hashes; and
9. only after those artifacts exist may Zodiac create a calendar proposal or
   the notification path issue a completion notice.

The success signal must be tied to the durable artifacts, not merely to an
HTTP 200 from Omi or a green n8n workflow node.

## 12. Implemented local contract

The first local bridge is available at:

```text
POST /v1/intake/omi
POST /v1/intake/external
GET  /v1/intake/external
GET  /v1/intake/external/{source_id}?source_kind=omi_memory
```

The endpoint requires `PANTHEON_INTAKE_BRIDGE_SECRET` in the daemon
environment and accepts either `X-Pantheon-Intake-Secret` or a Bearer token.
The secret is deliberately not part of the repository configuration or any
receipt.

The request envelope carries `payload` as the complete original JSON object
and may carry `transcript`, `labels`, and provider metadata. The bridge writes
the payload under `Intake/Omi/YYYY/MM/*.json`, writes a recoverable envelope
sidecar beside it for labels and routing metadata, writes a Markdown transcript
under `Intake/Transcripts/YYYY/MM/*.md`, runs the existing Intake scan, and
converts the transcript through Present into the configured raw Vault tree.
The JSON payload is written byte-for-byte and its `sha256:` digest is checked
before acceptance.

Each accepted source receives a durable receipt in the
`external_intake_receipts` table. It records source identity, both hashes,
Intake payload/envelope/transcript paths and occurrence IDs, conversion status,
and the verified normalized path when conversion completes. Replaying the same source and
content returns `duplicate`; replaying a source ID with different content
returns `409 Conflict`.

The native Omi endpoint wraps the provider payload into this same envelope
internally, so it has the same receipt and replay guarantees as the generic
endpoint.

The Linux BLE receiver reaches the same durable Intake path indirectly through
the existing Rumor watcher: completed WAV → local Whisper transcript → Intake
occurrence → Present/Vault conversion. No cloud webhook or tunnel is involved
in this transport mode.

This increment deliberately stops at verified Vault conversion for the generic
bridge. The local BLE proof below also demonstrates a separately submitted
Apollo order, but it does not claim that a gateway account is connected or that
a calendar, email, or notification order has been executed. Those remain
downstream, separately authorized stages.

## 12A. Verified Linux/Omi proof

On 2026-08-23, Linux connected to Omi `FC:92:2A:DB:64:79` through native
BlueZ D-Bus. A 30-second capture received 1,480 Opus notifications and wrote a
29.6-second, 16 kHz mono WAV. The file entered the configured Rumor watch path:

```text
/home/brick/Pantheon Audio/Omi BLE/omi-dbus-kernel26-live2.wav
```

The durable chain then completed as follows:

```text
native BlueZ GATT
  → WAV
  → Rumor source rumor-98502234c512f4c58bf32cfde1d915ce
  → Whisper transcript in 80 Pantheon Raw/Audio Transcripts
  → Intake occurrence intake-1bc9e606f8803c5ed3db28886b667bf0
  → Present verified artifact in 80 Pantheon Raw/Text
  → Apollo report report-371f67e13905ab18200da73610425575
```

The first Apollo dispatch correctly stopped before reasoning because the
Rumor-to-Intake handoff had not yet run Present. Running Present explicitly
changed the occurrence to verified, after which the corrected Apollo order
completed and wrote a cited report. This is an important acceptance boundary:
successful audio capture and transcript creation do not by themselves prove
Vault conversion or authorize downstream actions.

Apollo report quality also has a plain-language requirement: every concrete
request, action item, commitment, decision, or proposal must be named in the
answer. A citation is attached evidence, not a substitute for the statement,
and citations must use the full supplied Vault-relative token rather than an
invented short number.

## 13. Successor boundary

Kernel 26 should leave behind a general intake contract that later sources can
use without changing Apollo or teaching each daemon a new provider API.

The next likely increments are:

- Omi audio/Echo reconciliation when a local audio source is available;
- richer personal-label and relationship indexing;
- report-driven Zodiac calendar proposals;
- governed notification fan-out; and
- deterministic retention execution once Zeus exists.
