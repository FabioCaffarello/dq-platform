<!-- path: studies/critiques/2026-05-30-b3-session-reading-router-critique-2.md -->

# B3-2 — Critique Round 2

## Metadata

- **Target study:**
  [`studies/decisions/2026-05-30-b3-session-reading-router.md`](../decisions/2026-05-30-b3-session-reading-router.md)
  (post-round-1-revision state).
- **Round:** 2 (final per
  [`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  step 7 — "maximum two critique-revise rounds").
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
Operator Response trailer to confirm dispositions landed.

[no-blocking] No R1, R2, R3, R4, R5, R6, R7, R8 violation
found in the revised draft. The five round-1 important
findings landed as committed in the Operator Response
trailer (§Consequences §1 cost claim qualified + OQ-5;
P-B3RR.4 boundary cases enumerated; Flow 6 row PR-flow
inheritance clarified; sixth implementation-slice row
added with AC-W3-3 / AC-W3-7 load-bearing; D0 split into
formal §(a) outcome vs. practical re-routing). The five
minor findings landed cleanly.

[important] P5: "Recommendation §The router — row
boundary between B2 follow-up and Implementation slice" —
the sixth row (implementation slice) and the first row (B2
follow-up) overlap when a B2 study closes with a follow-on
implementation PR. A B2 follow-up that ships an
implementation slice originates from a B2 B-row *and*
lands an implementation under a closed B-row — both row
descriptions apply. The router needs a tie-breaker.
Recommended shape: the tie-breaker is the session's
output artifact. B2 / B3 rows apply when the session's
output is a *study* (and AC-1..AC-10 is the gate); the
implementation-slice row applies when the session's output
is *code or scaffold* (and AC-W3-3 / AC-W3-7 are the
gates). A B2 follow-up that produces both a study and an
implementation slice runs as two separate sessions (the
R4 one-topic discipline already requires this). Add a
one-line clarification to the router preamble or to the
implementation-slice row's Trigger column.

[minor] P5: "Decision Drivers D0 — formal §(a) outcome
wording" — the split between formal §(a) outcome and
practical re-routing reads cleanly, but the sentence "the
§(a) outcome is `rejected` only if no other lane fits"
could be tighter. ADR-0049 §(a) defines `rejected` as
"iff the proposal falls outside the three in-scope
families *and* outside any active wave's gate". A
Condition-4 materiality failure that re-routes to Flow 6
is not `rejected` under §(a) — it is "out-of-scope for
B3 but in-scope for Flow 6", which is a separate
outcome from §(a)'s four-option taxonomy. Tighten:
explicitly distinguish "ineligible for B3 but valid
elsewhere" from `rejected` per §(a).

[minor] AC-2: "Recommendation §The router —
Implementation slice row, R1 awareness" — R1 (no
production code during Waves 1 and 2) is historical now
(Wave 3 closed; CLAUDE.md §2.1 says so). Implementation-
slice sessions land production code. The always-on floor
already includes CLAUDE.md §2.1 + §3 R1 (with its
historical status), so the row does not need to repeat
the status. No change required; raised so the row is not
later read as if R1 still forbids the implementation slice.

[minor] AC-6: "Open Questions OQ-5 — measurement
timeline" — OQ-5 commits to revisit the cost claim after
the first 2-3 post-router sessions, but does not commit
when the measurement decision lands. Recommended: add
"the measurement-decision PR opens by the third
post-router session at the latest, regardless of whether
2-3 sessions have surfaced enough signal". This gives
the measurement a forcing function rather than
indefinite deferral.

[minor] AC-2: "Historical-skim set note paragraph" — the
explanatory note about wave-3-acceptance-criteria.md
being moved out of the historical-skim set is helpful
but slightly duplicative of the implementation-slice
row's load-bearing declaration. Could be tightened by
referencing the row directly ("see the Implementation
slice row above") instead of restating the AC-W3
applicability. No change required if the duplication
helps reviewers track the round-1 revision; raised as a
style observation.

Acceptance criteria sweep (post-round-1-revision):
- AC-1 (path header): pass.
- AC-2 (required sections in order): pass.
- AC-3 (≥2 options): pass — four options A/B/C/D.
- AC-4 (Recommendation grounded): pass.
- AC-5 (no external naming): pass.
- AC-6 (Open Questions marked): pass — OQ-1..OQ-5 each
  carry out-of-scope-for-current-cycle markers.
- AC-7 (Promotion target concrete): pass.
- AC-8 (≥1 critique round): pass — round 1 preserved,
  this round 2 will be preserved on commit.
- AC-9 (blocking findings addressed): N/A — no blocking
  findings round 2.
- AC-10 (decision-log row updated): pending step 9 of
  the loop; not yet a violation.

Summary: 0 blocking / 1 important / 4 minor. The single
important finding (row boundary tie-breaker between
B2 follow-up and Implementation slice) is the one
material gap surfaced by the round-1 revision. No round 3
under the two-round cap — the finding lands in this
revision and the study moves to resolved-study after
operator dispositions the round-2 trailer.
```

## Operator Response

- **[important] P5 row boundary tie-breaker** — *applied
  as recommended*: the router preamble will spell out the
  output-artifact tie-breaker — B2 / B3 rows apply when
  the session's output is a study (AC-1..AC-10 gate);
  the implementation-slice row applies when the session's
  output is code or scaffold (AC-W3-3 / AC-W3-7 gates).
  A B2 follow-up producing both a study and an
  implementation slice runs as two separate sessions per
  R4.

- **[minor] P5 D0 formal §(a) outcome wording** —
  *applied as recommended*: D0 will tighten to
  distinguish "ineligible for B3 but valid elsewhere"
  from §(a) `rejected`.

- **[minor] AC-2 implementation-slice R1 awareness** —
  *accepted as-is*: no change required; raised as a
  reading note only. The always-on floor already
  carries R1's historical status.

- **[minor] AC-6 OQ-5 measurement forcing function** —
  *applied as recommended*: OQ-5 will commit "the
  measurement-decision PR opens by the third post-router
  session at the latest".

- **[minor] AC-2 historical-skim duplication note** —
  *accepted as-is*: no change required; the duplication
  helps reviewers track the round-1 revision.

Two-round cap reached. Round-2 dispositions land in the
next commit on this branch; the study moves to
`resolved-study` after the decision-log row update.
