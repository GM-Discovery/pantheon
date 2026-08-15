# Archimedes Frontier Contract

**Status:** Draft 0.2  
**Scope:** All frontier reasoning performed on behalf of Pantheon  
**Constitutional basis:** Local First; Intelligence and Authority; Discipline of Intelligence; Archimedes; Durable Continuity

---

## 1. Purpose

Pantheon provides the stable place from which Archimedes may apply greater external reasoning power.

Archimedes exists so that local intelligence may recognize the useful boundary of its own reasoning and refer a problem outward without surrendering the private continuity of the person Pantheon serves.

**Pantheon is the stable place to stand.  
Archimedes is the lever that must push.**

This contract permits strong frontier reasoning while preventing frontier providers from receiving persistent identifying information about:

- the person Pantheon serves;
- family;
- friends;
- coworkers;
- acquaintances;
- clients;
- people encountered incidentally;
- organizations where identity would be revealing;
- or other identifiable persons represented within Pantheon.

---

# 2. Fundamental Boundary

Archimedes shall never receive unrestricted access to:

- the Vault;
- Weave;
- local identity records;
- local filesystems;
- Pantheon secrets;
- raw personal corpora;
- or any mechanism capable of resolving frontier aliases back to real identities.

All frontier reasoning occurs through deliberately constructed missives.

The frontier model sees only what Pantheon intentionally sends.

There is no ordinary Pantheon setting that disables this privacy boundary.

A person wishing to interact directly with a frontier provider may do so outside Pantheon and assumes responsibility for that disclosure.

Pantheon shall not provide a convenience mechanism for bypassing the Archimedes privacy contract.

---

# 3. When Archimedes Is Used

Local reasoning is preferred where sufficient.

Apollo is expected to distinguish among:

- a problem it can solve locally;
- a problem requiring additional local evidence;
- a problem requiring further retrieval through Past or Weave;
- and a problem exceeding the useful limits of available local reasoning.

When frontier reasoning is warranted, Apollo may recommend escalation.

Future may convert that recommendation into an order.

Mercury carries the order according to Pantheon's normal authority process.

A configured frontier provider does not give Apollo direct frontier access.

Frontier escalation remains an explicit Pantheon operation.

---

# 4. The Frontier Exchange

A **frontier exchange** is the complete frontier reasoning episode associated with one sealed Pantheon order.

An exchange may contain multiple turns, including:

- initial question;
- frontier response;
- clarification request;
- additional sanitized context;
- revised question;
- and final response.

The exchange remains one bounded unit until:

- Archimedes returns a sufficient result;
- the order is cancelled;
- the exchange fails;
- or Pantheon otherwise closes the order.

Temporary frontier identities remain stable throughout that exchange.

They do not survive into the next exchange.

---

# 5. The Frontier Missive

A frontier missive is a purpose-built reasoning packet.

It shall contain only information reasonably necessary for Archimedes to address the question.

A missive may include:

- the specific question;
- relevant facts;
- necessary relationships;
- necessary constraints;
- relevant chronology;
- evidence-derived conclusions;
- uncertainty;
- competing interpretations;
- and instructions describing the desired reasoning product.

A missive shall not contain unrelated continuity merely because that information is locally available.

---

# 6. Semantic Deidentification by Apollo

Apollo is responsible for constructing the deidentified frontier missive.

This is a reasoning task.

It shall not be reduced to deterministic pattern matching alone.

Apollo shall consider whether information could identify or materially assist in identifying a person through:

- names;
- usernames or handles;
- email addresses;
- telephone numbers;
- account identifiers;
- physical addresses;
- precise locations;
- employers;
- organizations;
- unique project names;
- domains;
- family relationships;
- distinctive combinations of biography;
- quoted language;
- filenames;
- filesystem paths;
- metadata;
- or combinations of otherwise ordinary details that become identifying together.

Apollo should preserve useful meaning while removing unnecessary identity.

The desired transformation is:

> **Identity removed. Reasoning context preserved.**

---

# 7. Relationships and Roles

Deidentification does not require removing relationships or roles that materially affect the reasoning problem.

Archimedes may need to know, for example:

```text
Person#518204 supervises Person#991034.

Person#721003 is the child of Person#518204.

Person#991034 works for Organization#440193.

Person#721003 participates in Project#318775.
```

These relationships may be essential.

The real identities usually are not.

Apollo may consult the Vault, Past, and Weave locally to determine which relationships and roles illuminate the problem.

Those relationships may be represented in the missive without exposing the identities behind them.

---

# 8. Ephemeral Frontier Identities

Real identities shall be replaced with randomly generated, exchange-scoped identifiers.

Examples:

```text
Person#518204
Person#991034
Organization#440193
Project#318775
Location#640188
```

These identifiers shall:

- remain stable during one frontier exchange;
- have no intended meaning outside that exchange;
- not be derived directly from real names or identifying values;
- not provide reusable cross-exchange identity;
- and be regenerated for later exchanges.

For example:

```text
Exchange A:
Jane Smith → Person#991034

Exchange B:
Jane Smith → Person#287410
```

The frontier provider therefore receives no intentional persistent pseudonym through which separate exchanges can be trivially joined.

---

# 9. The Ephemeral Translation Map

For each frontier exchange, Apollo produces a temporary local translation map sufficient to restore frontier identifiers to locally meaningful identities.

Conceptually:

```text
Jane Smith                  Person#991034
Moving Forward Institute    Organization#440193
Victory Theater             Project#318775
```

The translation map is:

- local-only;
- exchange-scoped;
- unavailable to Archimedes;
- unavailable to the frontier provider;
- unavailable through frontier tools;
- and temporary.

The translation map is not:

- part of Weave;
- a permanent pseudonym registry;
- durable identity infrastructure;
- or a frontier-accessible data source.

It exists solely to support one exchange and its return path.

---

# 10. Deterministic Privacy Gate

Apollo's semantic deidentification shall be followed by a deterministic privacy safety pass.

The privacy gate does not replace Apollo's judgment.

It does not reinterpret the missive.

Its authority is intentionally narrow:

> **Detect obvious privacy failures and redact them.**

The gate may inspect for mechanically recognizable information including:

- real names present in the temporary translation map;
- email addresses;
- telephone numbers;
- filesystem paths;
- account identifiers;
- identifying URLs;
- or other configured forbidden patterns.

When the gate detects an offending span, it may replace only that affected material with:

`<REDACTED>`

It shall not invent a replacement identity or reinterpret the surrounding meaning.

---

# 11. Privacy Strike Rules

## 11.1 No detected failures

If the deterministic gate finds no privacy failure:

**PASS**

The missive may proceed if otherwise authorized.

## 11.2 One detected failure

If the gate finds one privacy failure:

1. replace the affected area with `<REDACTED>`;
2. record a privacy incident;
3. report the incident to Mercury;
4. escalate the incident to the person;
5. allow the now-sanitized packet to continue if no other rule blocks it.

A caught privacy miss is considered consequential even when the fallback protection succeeds.

## 11.3 Two or more detected failures

If the gate detects two or more privacy failures in the outbound packet:

**DO NOT SEND.**

The packet shall:

1. be returned to Apollo;
2. identify the locations/categories of the detected failures without requiring unnecessary exposure;
3. require Apollo to rebuild or correct the missive;
4. return to the deterministic privacy gate;
5. and await another passing inspection before Mercury may carry it across the frontier.

The deterministic gate shall not attempt semantic repair beyond redaction.

Apollo remains responsible for semantic correction.

---

# 12. Privacy Incidents

A detected outbound privacy failure is a first-class Pantheon event.

Mercury shall carry notice of such failures through the appropriate escalation path.

The event should make it possible to determine:

- which frontier exchange was affected;
- which privacy rule triggered;
- whether material was redacted;
- whether transmission proceeded or was blocked;
- whether Apollo was required to correct the missive;
- and whether repeated failures indicate a broader system problem.

Pantheon shall not silently normalize repeated privacy misses as ordinary behavior.

---

# 13. No Identity Resolution at the Frontier

Archimedes shall never receive:

- the translation map;
- a query interface to the translation map;
- a persistent identity registry;
- a tool capable of resolving temporary frontier aliases;
- or indirect access to another Pantheon component capable of performing re-identification.

The frontier boundary shall be designed so that a frontier model cannot ask:

> Who is Person#991034?

and receive the answer from Pantheon.

---

# 14. Mercury and the Mote of Chron

Mercury carries the sealed frontier order.

The order carries its mote of Chron, establishing sufficient identity and authority to know:

- which order this is;
- which exchange belongs to it;
- when it became valid;
- what authority sealed it;
- what constraints apply;
- and whether the exchange remains active.

Mercury carries the sanitized missive to Archimedes.

Mercury does not independently enrich the frontier packet with private continuity.

Authority metadata needed internally by Pantheon does not automatically become frontier context.

---

# 15. Archimedes

Archimedes is the engine that drives the frontier model through the exchange.

Archimedes may be inhabited by different frontier models or providers.

The provider is replaceable.

The role is stable.

Archimedes shall:

- receive the sanitized missive;
- conduct the frontier exchange;
- maintain the temporary aliases supplied for that exchange;
- submit clarification requests where needed;
- receive additional sanitized context where authorized;
- obtain the requested frontier reasoning product;
- and make the response available for collection.

Archimedes does not receive unrestricted local access while conducting the exchange.

---

# 16. Mercury's Rounds

Mercury does not wait idly for frontier responses.

Mercury makes rounds.

One of Mercury's regular stops is Archimedes.

Conceptually:

```text
Mercury arrives at Archimedes
        ↓
Has the active exchange returned something?
        ├── No → continue the round
        └── Yes
             ↓
          collect response
             ↓
          carry it locally
             ↓
          continue the round
```

Mercury's rounds may therefore carry:

- new frontier orders;
- clarification responses;
- completed frontier results;
- failed exchange reports;
- and privacy incidents.

Polling frequency is a later operational/configuration decision.

The architecture does not require a separate daemon whose sole purpose is to wait for Archimedes.

---

# 17. Clarification Turns

If Archimedes cannot responsibly answer without clarification, it may request more information.

The request returns through Mercury.

Any additional information prepared locally shall pass through the same process as the original missive:

1. local reasoning;
2. semantic deidentification;
3. use of the existing exchange-scoped aliases;
4. deterministic privacy gate;
5. Mercury transport;
6. return to the same active Archimedes exchange.

A clarification request does not create permission for direct Vault access.

The same temporary translation map remains valid until the exchange closes.

---

# 18. Return and Local Rehydration

When Mercury collects a frontier response, that response returns to the local Pantheon boundary.

Temporary frontier identifiers are translated back into locally meaningful identities using the ephemeral translation map.

Conceptually:

```text
Frontier response:

Person#991034 should speak with Person#518204 before proceeding.

Local result:

Jane should speak with Grant before proceeding.
```

Rehydration occurs locally.

The rehydrated result may then be:

- returned to Apollo;
- processed by Future;
- shown to the person;
- incorporated into a brief;
- preserved as useful reasoning;
- or otherwise processed by Pantheon.

Archimedes never receives the rehydrated form.

---

# 19. Destruction of the Translation Map

Once:

1. the frontier exchange is closed;
2. the required frontier response has returned;
3. temporary identifiers have been successfully rehydrated;
4. the locally useful result has been produced;

the temporary translation map shall be destroyed.

Destruction should be auditable without preserving the mappings themselves.

Pantheon may retain records such as:

```text
exchange_id: exchange-8217
translation_entries: 7
translation_map_destroyed: true
destroyed_at: <timestamp>
```

It shall not retain:

```text
Person#991034 = Jane Smith
```

after the map is no longer operationally required.

---

# 20. Retention Model

Pantheon shall distinguish durable human continuity from temporary transformation machinery.

## 20.1 Durable source material

Human-readable source material may be preserved indefinitely according to ordinary Vault and retention policy.

Examples include:

- transcripts;
- meaningful Markdown;
- source notes;
- documents retained as human evidence.

## 20.2 Durable useful results

Useful resulting understanding may be preserved indefinitely where appropriate.

Examples include:

- rehydrated Archimedes conclusions;
- Apollo conclusions;
- useful factual claims;
- relationships promoted into Weave;
- decisions;
- resulting orders;
- human-readable reports.

These durable results should hyperlink or otherwise cite meaningful source material wherever practical.

## 20.3 Temporary frontier transformation records

Intermediate Archimedes transformation/debug material has a default retention period of:

**30 days**

This may include:

- sanitized outbound missives;
- sanitized raw frontier responses;
- privacy-gate reports;
- exchange traces;
- provider/model metadata;
- and other temporary debugging evidence.

The purpose of this retention is operational diagnosis.

It is not archival accumulation.

These records exist long enough to investigate:

- privacy leakage;
- broken transformations;
- confusing responses;
- provider failures;
- or other frontier problems.

They should then expire according to lawful retention policy.

## 20.4 Ephemeral material

The translation map is not subject to the 30-day period.

It is destroyed promptly after successful exchange closure and rehydration.

---

# 21. Weave and Transformation History

Pantheon does not need to preserve every intermediate reasoning transformation indefinitely.

Its durable model is:

> **Preserve meaningful source. Preserve useful result. Preserve the relationships between them.**

Weave exists to preserve and navigate those relationships.

Intermediate machinery should not become permanent merely because it once participated in producing a result.

A useful frontier result should therefore be able to lead the person back toward meaningful source material without requiring preservation of every frontier packet and transformation step forever.

---

# 22. Residual Re-identification Risk

Pantheon shall not pretend that replacing names guarantees mathematical anonymity.

Distinctive combinations of contextual facts may themselves reveal identity.

Apollo's semantic responsibility therefore includes minimization:

> Preserve what Archimedes needs to reason. Remove what Archimedes does not need to know.

The privacy objective is not merely pseudonymization.

It is bounded disclosure.

---

# 23. Failure Rules

## 23.1 Apollo cannot confidently sanitize

Do not send the packet until the privacy process can complete satisfactorily.

## 23.2 Privacy gate finds one failure

Redact the affected span, escalate the incident, and continue only if the resulting packet otherwise passes.

## 23.3 Privacy gate finds two or more failures

Do not send.

Return to Apollo for correction and inspect again.

## 23.4 Translation map is lost before rehydration

Do not guess identities.

Preserve the sanitized response temporarily if useful for diagnosis.

Report the failed rehydration.

## 23.5 Frontier transport fails

Preserve local source state.

Do not broaden disclosure merely to obtain a successful response.

Mercury may retry according to later retry policy.

## 23.6 Frontier model requests unrestricted local access

Deny the request.

A request for more information becomes a clarification turn subject to this same contract.

---

# 24. Audit Requirements

A consequential frontier exchange should make it possible to determine:

- why escalation occurred;
- which order authorized it;
- which frontier provider and model were used;
- when each outbound packet crossed the frontier;
- whether Apollo's semantic scrub occurred;
- whether the deterministic privacy gate passed;
- whether any privacy incident occurred;
- whether the packet was redacted or returned for correction;
- what sanitized request and response were exchanged during the temporary audit period;
- whether rehydration succeeded;
- when the temporary translation map was destroyed;
- and which durable local result, if any, was produced.

Audit records shall not retain destroyed identity mappings.

Temporary transformation evidence shall expire according to the 30-day default unless a specific lawful diagnostic need temporarily extends it.

---

# 25. Non-Goals

This contract does not provide:

- direct frontier chat through Pantheon;
- a privacy-off mode;
- unrestricted Vault forwarding;
- permanent frontier pseudonyms;
- cross-exchange frontier identity continuity;
- frontier access to the identity map;
- arbitrary frontier filesystem access;
- indefinite retention of transformation traces;
- or permission for frontier providers to demand arbitrary additional local context.

Direct intentional disclosure through a provider's own interface occurs outside Pantheon.

---

# 26. Governing Principle

Archimedes exists to increase reasoning leverage without making private lives the price of that leverage.

Pantheon supplies the stable place to stand.

Apollo determines what must be understood and removes what need not be known.

The privacy gate checks the boundary.

Mercury carries the sealed force and returns on his rounds.

Archimedes drives the frontier exchange.

Weave preserves the meaningful relationship between enduring source and useful result.

The temporary machinery disappears when its work is finished.

**The frontier model receives the problem it needs to solve—not the life of the person whose problem it is.**