# Pantheon Weave Schema

The Weave is Pantheon's inspectable continuity layer.

It records observations, follows threads through time, forms trajectories, and preserves correction history.

The Weave is not a model's memory.
The Weave belongs to Pantheon.

## Capabilities

- `weave.read` — inspect Weave records and their cited source material.
- `weave.write` — create or revise Weave records according to this schema.

Neither capability grants authority to modify the source material from which a record was derived.

---

# Observation

Created primarily by Present.

```yaml
---
weave_type: observation
id: obs-YYYYMMDD-HHMMSS-slug
created: YYYY-MM-DDTHH:MM:SS-07:00
fate: present

status: active
# active | struck | superseded | archived

subject:
  type: person | project | relationship | commitment | habit | idea | system | other
  name: ""

thread_refs: []

source:
  type: interaction | transcript | vault | email | calendar | system
  ref: ""
  captured_at: ""

observation: ""

evidence:
  direct: true
  strength: high
  # low | moderate | high
  excerpt: ""

values_implicated: []

confidence: 1.0

correction:
  corrected: false
  corrected_at: null
  reason: null
  superseded_by: null
---
