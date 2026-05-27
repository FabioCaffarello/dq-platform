<!-- path: .claude/skills/critique-anti-patterns/SKILL.md -->
---
name: critique-anti-patterns
description: Use when reviewing a decision study, ADR draft, or any Wave-1 artifact under studies/decisions/ or docs/adr/. Encodes the critique finding template ([severity] R/P/AC label: section — concrete change), the three severity levels (blocking / important / minor) from .claude/commands/critique.md, six verbatim labeled exemplars from feedback-protocol.md, four reaction-style anti-patterns to never emit ("this is bad" / "I don't like X" / "the previous was better" / "just rewrite this"), and seven substantive anti-patterns (prior-art-as-justification, hidden commitments in Open Questions, citation drift, "additive" mislabel for breaking changes, vocabulary drift, strawman options, unaddressed blocking findings). Most substantive anti-patterns are preventive — incident history is not preserved in artifacts.
---

# `critique-anti-patterns`

How to give a critique that is auditable, actionable, and grounded in
the project's existing rules. Built on top of
`.claude/playbooks/feedback-protocol.md` (the source of truth for the
template), `.claude/commands/critique.md` (the severity ladder), and
`.claude/playbooks/acceptance-criteria.md` (the AC-1..AC-10 sweep).

> Reference files:
> - `reference/finding-template.md` — the literal template, six
>   labeled-exemplar findings, and the four reaction-style anti-patterns
>   (all verbatim from `feedback-protocol.md`).
> - `reference/anti-patterns-catalog.md` — the seven substantive
>   anti-patterns with origin (documented vs preventive) marked.

---

## The shape of a good finding

Every critique finding is **one sentence**, in this template:

```
[severity] <R/P/AC label>: <section name> — <what to change>.
```

Source: `.claude/playbooks/feedback-protocol.md:27` and
`.claude/commands/critique.md:32`.

Four ingredients, all required:

1. **Severity** — one of `[blocking]`, `[important]`, `[minor]`. See
   §"Severity ladder" below.
2. **Label** — one of `R1..R8` (CLAUDE.md §3 hard rules), `P1..P6`
   (CLAUDE.md §4 principles), or `AC-1..AC-10` (acceptance criteria).
3. **Section name** — the literal heading the finding targets.
4. **Concrete change** — what to do, not what is wrong. "Rewrite in
   our own terms" is concrete; "this is unclear" is not.

## Severity ladder (from `.claude/commands/critique.md:22-29`)

- **blocking** — must be fixed before the study can move to
  `resolved-study`. Typically a violation of R1, R2, R5, R6, or an
  unresolved Open Question with no out-of-scope marker.
- **important** — should be fixed; weakens the study but does not
  invalidate it.
- **minor** — wording, ordering, or polish.

If a critique pass finds zero blocking, say so explicitly — the
operator needs that signal to advance the loop
(`.claude/commands/critique.md:42-43`).

## The four anti-patterns of bad feedback (verbatim)

Source: `.claude/playbooks/feedback-protocol.md:48-56`. Never emit any
of these — they are reactions, not feedback, and they give the next
contributor nothing to act on:

- "This is bad / unclear / weird."
- "I don't like X."
- "The previous version was better."
- "Just rewrite this section."

If no R/P/AC label fits a concern, the concern may be off-scope for
Wave 1 (`feedback-protocol.md:62-64`).

---

## The seven substantive anti-patterns to flag in critique

Each one is named with the label that catches it.

1. **B1 — Reaction-style feedback in critique output.** Verbatim list
   above; catch by inspecting your own output before posting.

2. **B2 — Prior-art-as-justification (R5).** An option or rationale
   cites an external product, vendor, sibling team's internal
   project, or third-party design as the reason for a choice.
   Catch: AC-5 scan. Allowed: environment commodities (BigQuery,
   Kafka, Pub/Sub, OIDC, Prometheus, …).

3. **B3 — Hidden commitments in Open Questions (AC-6).** An OQ
   section item that quietly commits a position rather than asking a
   question. Catch: every OQ item must be (a) resolved, (b) marked
   `out-of-scope for current cycle` with a one-line reason, or
   (c) deferred with a forward-pointer. See
   `acceptance-criteria.md:21`.

4. **B4 — Citation drift (R5 / AC-4).** A claim cites a foundation
   doc or prior ADR but the cited text does not actually support the
   claim. Catch: open the citation and read the surrounding text.

5. **B5 — "Additive" mislabel for breaking changes (P5).** A change
   labeled `additive` or `non-breaking` that removes a field, changes
   a type, makes a constraint stricter, removes an enum value, or
   changes a default in a way that alters runtime behavior. Catch:
   match the change against the enumerated list in
   `studies/foundation/03-boundary-contract.md` "How it is versioned"
   (Surface 1, lines 98-113).

6. **B6 — Strawman options (AC-3).** An Options-Considered section
   where the non-preferred options are weakened so the recommendation
   looks obvious. Catch: every rejected option must list both pros
   and cons; rejection reasons must be architectural, not dismissive.

7. **B7 — Vocabulary drift cross-section (P5 / AC-2).** The same
   concept named differently across sections of one document (e.g.,
   `kind` vs `check type` vs `rule type`). Catch: scan the document
   for synonyms of the same noun; pick one and use it everywhere.

8. **B8 — Unaddressed blocking findings (AC-9).** A Recommendation
   that ignores a blocking finding from a prior critique round without
   either resolving it in the study or explicitly deferring it in Open
   Questions with a rationale (`acceptance-criteria.md:24`).

---

## Honest framing: most substantive anti-patterns are preventive

Critique findings are **not preserved in repository artifacts**. Only
one-line status records survive on resolved studies (e.g., "round 1
cleared with no blocking findings"). This means:

- The four reaction-style anti-patterns (above) are **documented by
  protocol** — the playbook lists them verbatim.
- The eight substantive anti-patterns B1–B8 are **preventive checks**
  — the project's discipline keeps incident history from
  accumulating. No before/after quotes are extractable from history.

The single fully-documented multi-round critique is commit `926e3e5`
(PR #10, ADR-0014 HTTP trigger handler): "Critique round 1: 1
blocking + 1 important + 5 minor findings; all addressed in the same
revision pass." That commit is the canonical reference for what a
clean critique-and-revision cycle looks like.

---

## The AC-1..AC-10 sweep (from `acceptance-criteria.md`)

A `/critique` pass must cover at minimum R1, R5, R6, P1, and every
acceptance criterion (`.claude/commands/critique.md:39-40`). The
ten ACs are reproduced in `reference/finding-template.md` §"Sweep
checklist" for fast reference.
