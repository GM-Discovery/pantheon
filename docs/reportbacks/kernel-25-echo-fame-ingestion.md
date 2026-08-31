---
kernel: 25
status: REPOSITORY IMPLEMENTED — bounded Echo-to-Fame ingestion path
predecessor: Kernel 24 / 24A — Aphrodite and Fame human-context design
---

# Kernel 25 Reportback — Echo to Fame Ingestion

Kernel 25 implements the first bounded runtime path from a completed Echo run
to durable Fame observations. It uses reviewed speaker identity as input,
preserves anonymous and corrected evidence, and validates every extracted
interaction against an exact transcript citation.

## A. Runtime path

```text
Echo completed run
  → Fame conversation record
  → Fame segment attributions
  → reviewed speaker/person confirmation
  → bounded local extraction model
  → deterministic citation/speaker validation
  → durable Fame observation
```

Fame is not a replacement for Echo. Echo remains the source of diarization and
speaker-review evidence. Fame joins that evidence to a durable person record
only where the mapping has been supplied and validated.

## B. Durable records

Kernel 25 persists:

- people and their status, including provisional people;
- identity evidence with source reference, detail, confidence, and correction
  link;
- conversations with Echo run, source path, source hash, and boundary state;
- per-segment anonymous/person attribution;
- interaction observations with kind, content, actor/subject person IDs,
  citation, timestamps, confidence, and status; and
- human corrections with target and statement.

The default local self record is `person-self` / `Grant`. New people default
to `provisional`; that status is not silently upgraded by extraction.

## C. Process contract

The combined process requires at least one confirmation of the form:

```text
SPEAKER_00 → <known Fame person ID>
```

It preflights all confirmations before changing state. It rejects unknown
people, unknown speakers, conflicting confirmations, and replacement of an
existing conflicting assignment.

The process creates or updates the conversation as `provisional` unless a
caller explicitly provides a confirmed boundary. It initializes attributions
for every Echo segment and only attaches a person ID to reviewed mappings.

## D. Extraction contract

The extraction model uses the configured local Apollo extraction model slot,
with local-only endpoint validation and a bounded request. Its output is
limited to action items, commitments, decisions, unresolved questions, and
uncertainties.

The persistence layer rejects an extracted claim when:

- the speaker is missing;
- the evidence citation is missing;
- the citation was not supplied to the model;
- the cited segment has a different speaker; or
- the claim cannot be represented as an accountable observation.

Stored observations use the exact source citation and, where available, the
source segment's start/end timestamps. Repeated identical observations are
deduplicated by conversation, kind, content, and source.

## E. Operator and API surfaces

The local operator surface includes:

```text
pantheon show fame
pantheon fame person add <name>
pantheon fame conversation <echo-run-id>
pantheon fame process <echo-run-id> <speaker> <person-id>
pantheon fame extract <conversation-id>
pantheon fame correct <target-type> <kind> <statement>
```

The corresponding local API exposes Fame inspection, people, evidence,
conversations, processing, extraction, observations, and corrections. The
Panopticon view separates people, conversations/attributions, observations,
and corrections so that identity and interpretation remain inspectable.

## F. Verification

Focused Kernel 25 coverage verifies that:

- a confirmed speaker is attached to the correct Fame person;
- extraction produces a cited observation through the local model boundary;
- identity evidence and corrections remain durable;
- anonymous segments are not silently relabeled;
- the HTTP and slash-command surfaces expose the same Fame state; and
- source/citation and speaker consistency checks protect persistence.

Repository-wide verification remains the handoff requirement before calling
the kernel complete:

```text
GOCACHE=/tmp/pantheon-go-cache go test ./...
GOCACHE=/tmp/pantheon-go-cache go vet ./...
git diff --check
```

## G. What this kernel does not yet claim

Kernel 25 does not yet provide the full end-to-end report outcome requested by
the broader ingestion design. Specifically, it does not by itself:

- write an event-specific Vault transcript archive;
- compose an integrated Apollo report from Fame plus related Vault context;
- write a calendar event;
- send email;
- send a completion notification; or
- create an execution order from speech.

Those are downstream consumers. The current kernel makes their evidence
boundary possible without allowing a local extraction result to masquerade as
their completion.

## H. Next implementation boundary

The next safe increment is a report assembler that receives a Fame conversation
and creates two explicit durable outputs:

1. the event transcript/identity archive; and
2. the cited Apollo report with a separate, bounded Vault/Past/Weave context
   section.

Zodiac/calendar projection and notification should run only after that report
passes validation and each downstream capability is available and authorized.
