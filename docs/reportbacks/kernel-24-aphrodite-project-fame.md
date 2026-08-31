---
kernel: 24
status: REPOSITORY IMPLEMENTED — bounded Aphrodite and Project Fame workshop
predecessor: Kernel 23 — Zodiac Deterministic Schedule Keeper
---

# Kernel 24 Reportback — Aphrodite and Project Fame Workshop

Kernel 24 establishes Aphrodite as a governed project workshop and creates the
first Project Fame corpus and workspace. It does not promote candidate code,
install capabilities, grant authority, or replace the running Pantheon daemon.

## A. Purpose and boundary

Aphrodite is Pantheon's design and development intelligence. In this kernel,
her first project is Project Fame: research and validation of the human-context
layer that will eventually sit between recorded speech and later continuity
consumers.

The implemented boundary is deliberately narrower than a self-modifying
developer. Aphrodite can record project state, preserve candidate artifacts,
copy bounded evidence fixtures, and formulate research requests. Those records
are inputs for human review; they are not integration authority.

## B. Durable project state

The daemon creates a bounded workspace below its state directory:

```text
aphrodite/
└── projects/
    └── project-fame/
        ├── README.md
        ├── inputs/fame/
        ├── research/
        └── artifacts/
```

SQLite records maintain the project, artifacts, research requests, and copied
Fame fixtures. Workspace files are written with restrictive local permissions
and are recorded with SHA-256 digests.

The default Project Fame record starts in the `observation` phase and remains
`active`. Supported lifecycle phases are:

```text
observation → concept → research → design → prototype → validation
             → implementation_proposal → candidate_diff → authorized_integration
```

The lifecycle is descriptive and reviewable. Moving a project to a later phase
does not itself authorize production integration.

## C. Candidate artifacts and research

Aphrodite can persist bounded artifacts classified as observations, research
notes, experiments, test fixtures, designs, proposals, candidates, or rejected
approaches. It can also write an Apollo research request with an optional
Situation Brief flag.

Each research request states that it is not an order to install software,
access a new source, or change production. Bifrost capability visibility may be
reported to Aphrodite, but the workshop has no download, installation,
activation, or authorization path.

## D. Project Fame corpus

The seeded corpus covers representative interpretation boundaries:

- explicit request;
- joking request;
- uncertain commitment;
- quoted speech; and
- ordinary conversation with nothing actionable.

The fixture endpoint can also copy a preserved Rumor transcript into the
Project Fame workspace. The copy carries source identity, source path,
optional citation bounds, and a digest. The Rumor original is not edited or
relabeled.

This makes it possible to test false positives as well as false negatives. A
system that calls every conversational possibility a commitment has failed the
Project Fame boundary even if it finds many genuine requests.

## E. Inspection and operator surface

The implementation exposes the Aphrodite snapshot through the local API and
Panopticon, including project phase, workspace, artifacts, research requests,
Fame fixtures, approved inspection routes, and capability inventory.

Relevant commands include:

```text
pantheon show aphrodite
pantheon aphrodite project <id>
pantheon aphrodite phase <id> --phase <phase>
pantheon aphrodite artifact <project-id> ...
pantheon aphrodite research <project-id> ...
pantheon aphrodite fame-fixture <project-id> ...
```

The web surface describes the workspace as a candidate workshop and makes the
promotion boundary visible.

## F. Verification

Kernel 24 tests cover:

- creation of the bounded Project Fame workspace and seeded corpus;
- lifecycle and artifact persistence;
- workspace path containment;
- copying a Rumor transcript without modifying the source;
- source hash/provenance retention; and
- Panopticon/API visibility and the absence of a promotion path.

## G. Deliberate non-claims

Kernel 24 does not claim:

- that Aphrodite can modify production code;
- that Project Fame is the final runtime architecture;
- that speech has become authorization;
- that an MCP or n8n capability has been installed;
- that a transcript has been resolved to a person;
- that Apollo has produced an integrated report; or
- that a calendar event or notification has been created.

The next runtime boundary is Kernel 25: use Echo's reviewed speaker identity
evidence to create a bounded Fame conversation and persist cited interaction
observations.
