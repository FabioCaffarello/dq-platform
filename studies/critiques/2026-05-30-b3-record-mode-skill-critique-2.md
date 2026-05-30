<!-- path: studies/critiques/2026-05-30-b3-record-mode-skill-critique-2.md -->

# B3-3 — Critique Round 2

## Metadata

- **Target study:**
  [`studies/decisions/2026-05-30-b3-record-mode-skill.md`](../decisions/2026-05-30-b3-record-mode-skill.md)
  (post-round-1-revision state).
- **Round:** 2 (final per
  [`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  step 7 — two-round cap).
- **Date:** 2026-05-30.
- **Preservation status:** preserved (this file is the round-2
  capture; round 2 is the final round under the two-round cap).
- **Closing commit hash:** filled in at the close-PR step of the
  post-Wave-3 session loop.

## Critique Output

```text
Adversarial review of the round-1-revised study.
Grounding: CLAUDE.md §3 / §4, acceptance-criteria.md,
feedback-protocol.md, the round-1 critique capture's
Operator Response trailer to confirm the three important
findings landed (F1 Track C reasoning; F4 S3 ToCheckSpecs
citation; F5 S6 sync.Mutex sentinel citation) and the
two applied-minor findings landed (F2 franz-go scope
note; F7 OQ-4 re-label).

[no-blocking] No R1, R2, R3, R4, R5, R6, R7, R8
violation found in the revised draft. The round-1
important findings landed cleanly:
 - F1: Context "Why now" + P-B3RM.5 now lead with
   today's discoverability gap; Track C is named as a
   bonus, not the load-bearing rationale.
 - F4: S3 cites dsl/spec.go:114-127 (ToCheckSpecs) and
   spells out that buildRecordRunners calls it.
 - F5: S6 cites record_runner.go:399-403 (the sync-import
   sentinel) with an explanation of the explicit-import
   marker discipline.

[important] R5: "Recommendation §Scope note on library
naming" — the new scope note resolves the franz-go
descriptive-vs-prescriptive question, but the framing
goes one step further and asserts that franz-go "sits
inside the environment-exempt scope" as a
"substrate-adjacent client library". This is a new R5
reading: CLAUDE.md R5's enumerated list ("BigQuery,
Kafka, GCS, Pub/Sub, OIDC, Prometheus, OpenTelemetry,
Kubernetes, Go, Docker, slog, JSON Schema, and
equivalents") explicitly includes substrates and
runtime tools; whether a third-party Go client library
(franz-go) counts as "and equivalents" is interpretive.
The two readings: (a) tight — franz-go is a concrete
library reference, not a pattern source, so R5 is
satisfied because the naming is descriptive; (b) loose
— franz-go is substrate-adjacent "and equivalents",
covered by R5 exemption. Both readings reach the same
conclusion (the naming is fine), but the scope note
pre-commits the loose reading rather than letting both
work. Suggest: re-phrase so the scope note explains the
descriptive intent without asserting a new R5 reading.
Example: "the naming is descriptive of what the code
uses, not prescriptive of a pattern source — R5
('we use X is fine; we are doing Y because X does Y is
not') is satisfied because S1 / S2 describe what the
code does, not why a particular pattern was chosen."
This keeps both R5 readings open and lets the promotion
session land a tighter or looser reading if needed.

[minor] AC-2: "Recommendation §The seven conventions —
S5 ADR-0024 reference citations format" — S5's
"the runner code at engine/internal/runner/record_runner.go:209,
:273, :292-293, :348" mixes single-line and range
citations. Pure style. Could be tightened to a
brace-form or a per-line list. No structural issue.

[minor] AC-2: "Recommendation §The seven conventions —
S2 and S5 link discipline" — S2 cites
kafka_consumer.go:62-67 (DisableAutoCommit block) and
:94-102 (commit-after-fetch call). S5 cites several
record_runner.go ranges. Both are correctly cited, but
neither cites the ADR-0024 file path directly the way
S5's "ADR backing" sub-bullet does. This is a minor
inconsistency in citation density across conventions;
accept-as-is since the implementation slice will
normalize when the SKILL.md is written.

[minor] AC-2: "Consequences §1-9 — count and density" —
the consequence list runs nine items. Each item traces
to a specific decision (artifact split, PR
discoverability, Track-C pre-positioning, no-code-
change, ADR-0051 precedent reuse, amendment path, CI
gates, citation drift, frontmatter shape). Could be
consolidated but doing so loses precision; the current
shape mirrors B3-2's consequence density. No change
needed.

Acceptance criteria sweep (post-round-1-revision):
- AC-1 (path header): pass.
- AC-2 (required sections in order): pass.
- AC-3 (≥2 options): pass — four options A/B/C/D.
- AC-4 (Recommendation grounded): pass.
- AC-5 (no external naming as justification): pass with
  [important] note F8 above (scope-note framing).
- AC-6 (Open Questions marked): pass — OQ-1..OQ-4
  carry out-of-scope markers.
- AC-7 (Promotion target concrete): pass.
- AC-8 (≥1 critique round): pass — round 1 preserved,
  round 2 will be on commit.
- AC-9 (blocking findings addressed): N/A — no blocking
  in either round.
- AC-10 (decision-log row updated): pending step 9 of
  the loop; not yet a violation.

Summary: 0 blocking / 1 important / 3 minor. The single
important finding (scope-note framing pre-commits a new
R5 reading) is the one material gap surfaced by the
round-1 revision. No round 3 under the two-round cap —
the finding lands in this revision; the study moves to
resolved-study after operator dispositions the round-2
trailer.
```

## Operator Response

- **[important] R5 scope-note framing** —
  *applied as recommended*: the scope note will be
  rephrased to explain the descriptive-vs-prescriptive
  distinction without asserting that franz-go falls
  inside R5's "and equivalents" exemption. Both R5
  readings remain open; the promotion session can land
  whichever the reviewer prefers.

- **[minor] AC-2 S5 ADR-0024 citation format style** —
  *accepted as-is*: the implementation slice will
  normalize citation format when the SKILL.md is
  written; pure style note.

- **[minor] AC-2 S2 / S5 citation density consistency** —
  *accepted as-is*: the implementation slice normalizes
  citation density when the SKILL.md is written.

- **[minor] AC-2 Consequences §1-9 density** —
  *accepted as-is*: each item traces to a specific
  decision; consolidation would lose precision. Mirrors
  B3-2's consequence density.

Two-round cap reached. Round-2 dispositions land in the
next commit on this branch; the study moves to
`resolved-study` after the decision-log row update.
