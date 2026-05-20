<!-- path: CONTRIBUTING.md -->

# Contributing to the DQ Platform

This guide describes how to contribute **during Wave 1**.

The project is in a foundation phase. Contribution right now means
**resolving a blocking decision** listed in
[`studies/foundation/06-decision-log.md`](./studies/foundation/06-decision-log.md)
— not writing production code. Hard rule R1 in
[`CLAUDE.md`](./CLAUDE.md) forbids that until Wave 3.

---

## The loop

Pick a B0 row whose dependencies are already resolved, open a fresh
Claude Code session, and run the 10-step loop documented in
[`.claude/playbooks/wave-1-session-loop.md`](./.claude/playbooks/wave-1-session-loop.md).
A study is "done" when it meets every criterion in
[`.claude/playbooks/acceptance-criteria.md`](./.claude/playbooks/acceptance-criteria.md).

---

## Hard rules (R1–R8)

Canonical text lives in [`CLAUDE.md`](./CLAUDE.md) §3. One-line
summary:

- **R1.** No production code (Go, YAML rules, Dockerfile, CI) during
  Waves 1 and 2.
- **R2.** Do not invent requirements; record gaps as `TBD`.
- **R3.** Settled architectural decisions are final unless you spot
  a genuine inconsistency.
- **R4.** One topic per session.
- **R5.** No external vendor / sibling-team / prior-art names as
  justification. Environment commodities (BigQuery, Kafka, etc.) are
  allowed.
- **R6.** Every markdown file starts with an HTML path-header
  comment.
- **R7.** Technical artifacts in English.
- **R8.** Promoted artifacts (ADRs, published docs) do not link
  backwards into `studies/`.

## Platform principles (P1–P6)

Canonical text lives in [`CLAUDE.md`](./CLAUDE.md) §4.

- **P1.** Rules stay declarative — no SQL, no expressions.
- **P2.** Engine behavior is deterministic.
- **P3.** Ownership is explicit everywhere.
- **P4.** Cost is a first-class constraint.
- **P5.** Evolution is contract-driven.
- **P6.** Borrow patterns, not baggage.

---

## Giving feedback during review

Feedback cites a rule (R1–R8), principle (P1–P6), or acceptance
criterion (AC-1–AC-10) — never personal taste. The full protocol is
[`.claude/playbooks/feedback-protocol.md`](./.claude/playbooks/feedback-protocol.md).

---

## Commit conventions

Mirror [`KICKOFF.md`](./KICKOFF.md):

- `docs(decision): resolve B0-N — <topic>` — closing a B0 study.
- `docs(adr): promote <slug> to ADR` — Wave 3 only.
- `chore: <change>` — repository hygiene.
- `docs: <change>` — documentation that is not a decision.

No `feat(engine):`, `feat(rules):`, or similar until Wave 3.
