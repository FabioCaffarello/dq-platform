<!-- path: .claude/playbooks/feedback-protocol.md -->

# Feedback Protocol

How to give feedback on a Wave-1 study without going personal.

---

## Frame

Feedback is to the **artifact**, not to the agent or contributor.
The artifact is wrong or incomplete; people aren't.

## Cite a label

Every piece of feedback names one of:

- A **hard rule** — R1, R2, R3, R4, R5, R6, R7, R8
  (see [`CLAUDE.md`](../../CLAUDE.md) §3).
- A **platform principle** — P1, P2, P3, P4, P5, P6
  (see [`CLAUDE.md`](../../CLAUDE.md) §4).
- An **acceptance criterion** — AC-1…AC-10
  (see [`acceptance-criteria.md`](./acceptance-criteria.md)).

## Template

    <R/P/AC label>: <section name> — <what to change>.

One sentence. Specific section. Concrete change.

## Examples

- `R1: "Proposed Implementation" — replace the SQL block with a
  prose description; we're in Wave 1.`
- `R5: "Considered Options" — option 3 cites a vendor by name as
  justification. Rewrite in our own terms.`
- `R6: "header" — file is missing its path-header HTML comment;
  add one.`
- `P1: "Recommendation" — the proposed escape hatch breaks
  declarative-only. Remove it or move to Open Questions.`
- `P3: "Alert Routing" — no owner mapping in the example. Add one
  or note why it's deferred.`
- `AC-6: "Open Questions" — three items are unresolved with no
  out-of-scope marker. Either resolve or defer explicitly.`
- `AC-9: "Recommendation" — critique finding #2 (blocking) is not
  addressed and not deferred.`

## Anti-patterns (do not use)

- "This is bad / unclear / weird."
- "I don't like X."
- "The previous version was better."
- "Just rewrite this section."

These are not feedback — they are reactions. They give the next
contributor (human or agent) nothing to act on.

## Why labels

Labels make feedback **auditable across sessions**. Six months from
now, anyone reading the conversation history can tell whether a
review was rule-bound or vibes-bound. They also prevent the loop
from drifting: if no R/P/AC fits, the feedback may be off-scope for
Wave 1.
