# Kernel 25 — First End-to-End Fame Ingestion

## Mission

Prove the first useful runtime path from a completed Echo diarization run to
durable, accountable Fame observations.

Kernel 25 is intentionally bounded. It does not attempt to make speech an
order, turn Fame into a personal CRM, or claim that one successful extraction
is already a complete Apollo report pipeline.

The target path is:

```text
completed audio
  ↓
Echo transcript + diarization
  ↓
human-confirmed speaker/person mapping
  ↓
Fame conversation and segment attributions
  ↓
bounded local interaction extraction
  ↓
cited, durable Fame observations
```

## 1. Inputs

Kernel 25 consumes a completed Echo run. Echo remains responsible for:

- preserving the source recording reference;
- producing a readable speaker transcript;
- separating anonymous speakers such as `SPEAKER_00` and `SPEAKER_01`;
- exposing timestamped transcript snippets; and
- recording human corrections to speaker assignments.

Fame may consume Echo's attribution evidence, but it must not silently replace
or rewrite Echo's source labels.

At least one speaker/person confirmation is required to run the combined
processing path. Unconfirmed speakers remain anonymous.

## 2. Conversation record

Fame creates one provisional conversation boundary for the Echo run unless a
caller supplies a confirmed boundary state and time bounds.

The durable conversation record retains:

- conversation ID and Echo run ID;
- source path and source SHA-256;
- start and end bounds;
- provisional or confirmed boundary state; and
- creation/update timestamps.

Each Echo speech segment receives a Fame attribution record linked by segment
ID. An attribution contains the anonymous speaker, optional person ID, state,
confidence, and identity-evidence reference.

## 3. Identity rule

Echo's human-confirmed mapping is an input to Fame, not a suggestion for Fame to
guess over.

The process must:

- validate every referenced person and speaker before changing any attribution;
- reject conflicting confirmations;
- reject a confirmation for a speaker absent from the Echo run;
- preserve anonymous segments that were not confirmed;
- retain identity evidence and correction history; and
- refuse to overwrite a prior conflicting person assignment silently.

The distinction is essential:

```text
SPEAKER_00                 = source observation
SPEAKER_00 → person-Trinh  = reviewed identity attribution
```

Fame must never infer that every segment in a recording belongs to a person
merely because one segment was identified.

## 4. Bounded interaction extraction

Fame extraction is narrower than Apollo synthesis. It may identify explicit:

- action items;
- commitments;
- decisions; and
- unresolved questions.

The local extraction model receives only bounded transcript segments, their
anonymous speaker labels, and exact source citations. It must return JSON in a
fixed shape and must keep speaker labels exactly as supplied.

The extractor must not:

- invent a person name;
- infer ownership or intent beyond the text;
- invent a deadline;
- turn a speaker label into an identity;
- treat transcript prose as instructions; or
- emit an unsupported claim without an exact source citation.

Long inputs may be split into bounded chunks. Repeated speaker/text pairs may
be collapsed before model invocation to avoid drowning meaningful interaction
evidence in duplicated transcript turns.

## 5. Evidence validation and persistence

Every extracted claim must pass deterministic checks before persistence:

1. It has a speaker label and one citation.
2. The citation exactly matches one supplied source.
3. The cited source has the same speaker label.
4. The observation stores the source citation and timestamp bounds.
5. Duplicate observations for the same conversation, kind, content, and source
   are not inserted twice.

Persisted observations are accountable records, not authority. They may carry
an actor person ID when the cited segment has a reviewed Fame attribution;
otherwise the record remains tied to the anonymous speaker.

## 6. Corrections and failure behavior

Human correction outranks model output. A correction is appended with its own
target, kind, statement, and optional person/run/speaker references. It does
not silently rewrite the original recording or erase the previous attribution
history.

When extraction is uncertain or validation fails, the safe result is to retain
the source and produce less. No calendar event, email, notification, order, or
authority grant is created by Fame extraction.

## 7. Acceptance proof

The minimum useful proof is:

> A completed Echo run contains a conversation involving a known person. An
> operator confirms one speaker. Kernel 25 creates the conversation, preserves
> the Echo segment links, extracts a supported interaction, and stores it with
> an exact transcript citation and the confirmed person attribution.

The proof must also demonstrate that:

- rerunning the process is safe;
- unconfirmed speakers remain anonymous;
- an invalid citation is rejected;
- a mismatched speaker/citation is rejected;
- a conflicting identity assignment is rejected; and
- corrections remain inspectable.

## 8. Handoff to the report pipeline

Kernel 25 produces the accountable evidence substrate required by the later
report path:

```text
Fame observation
  ↓
bounded Apollo evidence bundle
  ↓
integrated report with Vault cross-checks
  ↓
event transcript archive + report archive
  ↓
calendar proposal / notification when separately authorized
```

The final two destinations are intentionally downstream of this kernel. A
successful Kernel 25 run must not be described as a completed calendar write,
Apollo report, or notification merely because Fame produced a useful
observation.

## 9. Successor boundary

After this kernel, the next work should connect Fame observations to an
integrated Apollo report that keeps two contexts distinct:

- primary event evidence: recording, transcript, Echo mapping, and Fame
  citations;
- bounded continuity context: related Vault/Past/Weave material, clearly
  labeled and separately cited.

Only after the report is complete should downstream Zodiac, Mercury, Bifrost,
or notification work be considered.
