<!-- path: .claude/skills/critique-anti-patterns/reference/finding-template.md -->

# Finding template — verbatim, with annotations

Source: `.claude/playbooks/feedback-protocol.md:25-46`. The six
exemplars are verbatim. Annotations after each show which label fires
and why.

## Template

```
<R/P/AC label>: <section name> — <what to change>.
```

(One sentence. Specific section. Concrete change. Source line 27.)

When run as `/critique`, prefix with severity:

```
[severity] <R/P/AC label>: <section name> — <what to change>.
```

(Source: `.claude/commands/critique.md:32`. Severity is one of
`[blocking]`, `[important]`, `[minor]`.)

---

## Six exemplar findings (verbatim, with annotations)

```
R1: "Proposed Implementation" — replace the SQL block with a
prose description; we're in Wave 1.
```
*R1: no production code during waves 1 and 2 — a literal SQL block
in a study draft is exactly what R1 forbids.* (Source line 33-34.)

```
R5: "Considered Options" — option 3 cites a vendor by name as
justification. Rewrite in our own terms.
```
*R5: no prior-art naming as justification — vendor name as the
reason a design is right is the canonical R5 violation.* (Source
line 35-36.)

```
R6: "header" — file is missing its path-header HTML comment;
add one.
```
*R6: every produced markdown file must start with an HTML path
comment.* (Source line 37-38.)

```
P1: "Recommendation" — the proposed escape hatch breaks
declarative-only. Remove it or move to Open Questions.
```
*P1: rules must remain declarative — proposing an escape hatch in a
Recommendation directly contradicts a non-negotiable principle.*
(Source line 39-40.)

```
P3: "Alert Routing" — no owner mapping in the example. Add one
or note why it's deferred.
```
*P3: ownership must be explicit everywhere — an alert example
without an owner mapping fails the principle.* (Source line 41-42.)

```
AC-6: "Open Questions" — three items are unresolved with no
out-of-scope marker. Either resolve or defer explicitly.
```
*AC-6: every OQ item must be resolved or explicitly marked
out-of-scope with a one-line reason.* (Source line 43-44.)

```
AC-9: "Recommendation" — critique finding #2 (blocking) is not
addressed and not deferred.
```
*AC-9: blocking findings from a prior round must be resolved in the
study or explicitly deferred in OQ with rationale.* (Source line
45-46.)

---

## The four reaction-style anti-patterns (never use)

Source: `.claude/playbooks/feedback-protocol.md:48-56`. These are
reactions, not feedback. They give the next contributor nothing to
act on:

- `"This is bad / unclear / weird."`
- `"I don't like X."`
- `"The previous version was better."`
- `"Just rewrite this section."`

If a concern cannot be expressed as `R/P/AC: section — change`, it
may be off-scope for Wave 1 (`feedback-protocol.md:62-64`).

---

## Why labels (source: `feedback-protocol.md:58-64`)

Labels make feedback auditable across sessions. Six months from now,
anyone reading the conversation history can tell whether a review was
rule-bound or vibes-bound. They also prevent the loop from drifting:
if no R/P/AC fits, the feedback may be off-scope for Wave 1.

---

## Sweep checklist — AC-1..AC-10 (from `acceptance-criteria.md`)

A `/critique` pass must cover at minimum R1, R5, R6, P1, and every
AC (`.claude/commands/critique.md:39-40`):

| # | Criterion | Quick check |
|---|---|---|
| AC-1 | File starts with HTML path-header comment (R6) | `head -1 <file>` is `<!-- path: ... -->` |
| AC-2 | Required sections present in order: Context, Decision Drivers, Considered Options, Recommendation, Consequences, Open Questions, Promotion target | Visual scan |
| AC-3 | At least two options considered | Count subsections under "Considered Options" |
| AC-4 | Recommendation grounded in (a) foundation doc, (b) prior decision, or (c) marked "new contribution proposed here, requires review" (R5) | Inline citation or marker present |
| AC-5 | No external vendor / sibling project / prior-art name as justification (R5). Environment commodities allowed (BigQuery, Kafka, Pub/Sub, OIDC, …) | Grep / manual scan |
| AC-6 | Open Questions empty, or each item marked "out-of-scope for current cycle" with one-line reason | Visual scan |
| AC-7 | Promotion target points to concrete `docs/adr/<NNNN>-<slug>.md` | Grep for `docs/adr/` |
| AC-8 | Survived ≥1 `/critique` round | Session history / commit messages |
| AC-9 | All blocking findings from latest critique resolved or explicitly deferred | Cross-reference latest critique vs. study |
| AC-10 | Matching row in `studies/foundation/06-decision-log.md` updated to `resolved-study` and links to file | Open log, check |

Source verbatim: `.claude/playbooks/acceptance-criteria.md:14-26`.

Note on AC-9 (`acceptance-criteria.md:31-34`): a blocking finding the
author *disagrees with* is not automatically deferred — the author
must rebut it in the study itself (typically in Decision Drivers or
Recommendation) and let the next critique surface whether the
rebuttal holds.
