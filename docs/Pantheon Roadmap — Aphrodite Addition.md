## Aphrodite — post-core construction capability

Aphrodite is Pantheon’s proposed software-construction intelligence.

Aphrodite eventually replaces the temporary external development scaffold currently used to construct Pantheon.

Conceptually:

```text
current scaffold:

Grant
  ↓
external architectural reasoning
  ↓
coding agent / Codex CLI
  ↓
Git + tests + smoke verification
  ↓
Pantheon Runtime
```

The intended later state is:

```text
Grant
  ↓
Pantheon
  ↓
Aphrodite
  ↓
isolated branch / worktree
  ↓
code + tests + candidate build
  ↓
review and authority boundary
  ↓
possible Pantheon successor
```

### Role

Aphrodite asks:

> **What new form should this software take, and can I construct a candidate version of it?**

Aphrodite may eventually be allowed to:

- read the Pantheon repository;
- inspect architecture and reportbacks;
- create isolated Git branches/worktrees;
- write code;
- write and update tests;
- run formatters, compilers, static analysis, and test suites;
- inspect failures;
- iteratively repair candidate changes;
- prepare documentation/reportbacks;
- construct candidate binaries;
- present diffs and evidence for review.

Aphrodite is a deity/intelligence and therefore possesses **no operational authority merely because she wrote the code**.

### Critical boundary

Pantheon must preserve a hard distinction:

> **Aphrodite may construct Pantheon.**

from:

> **Aphrodite may replace the running Pantheon.**

The first may eventually receive substantial bounded autonomy.

The second remains behind deterministic authority, Prometheus control, and Grant’s permissions.

A candidate binary must never become the running control plane merely because:

- Aphrodite recommends it;
- tests pass;
- it compiles;
- it appears to fix the problem;
- Aphrodite produced the reportback herself.

Self-modifying code must be governed by machinery outside the intelligence proposing the modification.

### Workshop boundary

Prefer giving Aphrodite an isolated workshop such as a Git worktree or equivalent constrained development environment.

Aphrodite may be permitted to make extensive changes there without those changes automatically affecting:

- the active Pantheon binary;
- active Prometheus controls;
- authority/grant state;
- Vault state;
- production configuration;
- secrets;
- live external integrations.

Promotion from workshop to active runtime requires an explicit deterministic path.

### Bootstrapping requirement

Do not implement Aphrodite until Pantheon’s existing authority and containment system has demonstrated enough maturity to govern a component capable of changing Pantheon itself.

Aphrodite should therefore remain post-core, after the whole-animal usability checkpoint and a period of real operation.

Pantheon should first prove that it can safely govern ordinary tools before being asked to govern its own software constructor.

### Relationship to Apollo

Apollo and Aphrodite remain distinct.

```text
Apollo:
  understand / investigate / reason / propose

Aphrodite:
  construct candidate software form
```

Apollo may diagnose an architectural problem.

Aphrodite may turn an approved construction request into code.

Neither gains deployment authority from intelligence alone.

### Roadmap placement

Add Aphrodite to the post-core horizon alongside Orion, Archimedes, Zeus, Nostalgia, and later operator improvements.

Likely later candidates are selected from demonstrated operational need rather than mythology or numerical sequence.

Aphrodite should be considered when the continuing external coding scaffold itself becomes a meaningful limitation on Pantheon’s independence, maintainability, or usefulness.