<!-- path: studies/critiques/2026-05-30-b3-session-reading-router-critique-1.md -->

# B3-2 — Critique Round 1

## Metadata

- **Target study:**
  [`studies/decisions/2026-05-30-b3-session-reading-router.md`](../decisions/2026-05-30-b3-session-reading-router.md).
- **Round:** 1.
- **Date:** 2026-05-30.
- **Preservation status:** preserved (this file is the round-1
  capture; the round-2 capture will be a sibling
  `-critique-2.md` if a second round runs).
- **Closing commit hash:** filled in at the close-PR step of the
  post-Wave-3 session loop.

## Critique Output

```text
Adversarial review of studies/decisions/2026-05-30-b3-session-reading-router.md.

Grounding read: CLAUDE.md §3 (R1-R8), §4 (P1-P6),
acceptance-criteria.md (AC-1..AC-10), feedback-protocol.md.

Findings follow.

[no-blocking] No R1, R2, R3, R4, R5, R6, R7, R8 violation found.
R1 historical (Wave 3 closed). R5 hygiene clean — no vendor /
sibling / prior-art naming. R6 path header present at line 1.
The two borderline conditions are marked
"new-contribution-requires-review" per R5. The five session
types are the exhaustive set declared by P-B3RR.4; R4 scope
(one topic) is respected — only CLAUDE.md §6, the decision-log
row, and the eventual ADR-0052 are touched.

[important] P4: "Consequences §1" — cost-savings claim
(~600 lines / ~85% for Flow 6; ~280 lines minimum for B3) is
asserted by line count, not measured against attention cost.
~700 lines of structured prose is not equivalent to ~700 lines
of dense code; the actual session-load drop is plausibly
smaller. Either qualify the claim ("by line count, not
benchmarked against attention cost") or commit a follow-up
OQ to measure on the first 2-3 post-router sessions.

[important] AC-2: "Locked premises P-B3RR.4" — claim that
"the five session types … are the exhaustive set under the
current operating shape" is asserted without testing boundary
cases. What about: a B-row triage session (the operator
deciding whether a newly surfaced item is B2 / B3 / amendment
/ rejected per ADR-0049 §(a))? A study revival session
(re-opening a previously deferred study)? An ADR
supersession session (the ADR-0017 pattern)? Either
explicitly enumerate why these collapse into the five named
types, or add them, or defer them with a one-line
out-of-scope marker.

[important] P5: "Recommendation §The router — Flow 6 row" —
the row says minimal playbook reading is
"post-wave3-session-loop.md step 10 only (PR-flow close —
Flow 6 inherits Flow 5 PR-flow)". CONTRIBUTING.md Flow 6
itself currently says "Same as Flow 5: dedicated branch …
applicable local gates …". The two statements are reconcilable
(step 10 *is* the PR-flow content in the playbook) but the
row reads as if Flow 6 sessions skip the rest of Flow 5's
PR-flow inheritance. Tighten: spell out that "step 10 only"
means the load-bearing playbook content for Flow 6 PR-flow is
exactly the step-10 block, not that Flow 6 skips other Flow 5
elements.

[important] P5: "Recommendation §Historical-skim set —
wave-3-acceptance-criteria.md" — AC-W3-7 ("local build, lint,
and test gates that exist for this surface") and AC-W3-3
("load-bearing implementation cites the B0/W2 commitment")
are gate semantics that arguably still apply to post-Wave-3
*implementation slices* landing under a closed B-row (e.g.,
an ADR-0051 follow-on slice shipping the four artifacts).
The router declares five session types, none of which is
"implementation slice landing under a closed B-row". Either
add a sixth row for implementation slices (with AC-W3 in the
load-bearing set) or argue why implementation slices are
covered by the existing five (e.g., they ride "B2 follow-up"
when the closed row was B2, or "B3 entry's follow-on" when
the closed row was B3). The omission is a structural gap, not
a wording fix.

[important] P5: "Decision Drivers D0 — re-routing language" —
D0 mixes two distinct outcomes. ADR-0049 §(a) "rejected"
applies when the proposal is "outside the three in-scope
families *and* outside any active wave's gate". A
Condition-4 materiality failure does NOT make the proposal
"outside the family" — it makes it Flow-6-territory. The
re-routing language ("through Flow 6 if Condition-4 fails
alone, or shelved pending a different framing if Condition 1
fails") is correct in spirit but mixes the §(a) rejected
outcome with re-routing, which is a different mechanism.
Separate them: ADR-0049 §(a) rejected is the formal outcome;
re-routing to Flow 6 is the practical follow-up. The decision
log row records the §(a) outcome; the operator opens a new
Flow 6 PR for the substance.

[minor] AC-2: "Recommendation §The router — Always-on floor"
— the bullet "CLAUDE.md §1-8 (…) required reading in §6 — the
floor includes itself by reference" is recursive. Either drop
the parenthetical reference to §6, or rewrite to make the
self-reference an intentional reminder ("§6 itself, including
the router, is part of the floor").

[minor] P5: "Consequences §9 — /sync-agents coverage" — the
study notes that /sync-agents currently does NOT detect drift
in playbook listings (ADR-0051 OQ-2 deferral), and that the
router's structural change to §6 may pressure OQ-2 earlier
than otherwise. Surface this dependency more explicitly: add a
one-line "see ADR-0051 OQ-2" pointer in §Consequences §9 so
the next OQ-2 ruling has a concrete forcing-function cite.

[minor] P5: "Consequences §7 — under-reading drift mitigation"
— the mitigation ("the trigger column has a concrete test")
relies on the contributor having calibration to apply the
test. A new contributor may misclassify. Consider adding to
the router preamble: "If unsure between two session types,
default to the next-larger reading set." This is a one-line
safety belt that does not erode the per-type narrowing
benefit.

[minor] AC-6: "Open Questions OQ-3 — artifact-type axis" —
documented adequately but the trade-off math is thin. The OQ
treats session-type axis vs. artifact-type axis as exclusive,
but they may compose (route on session type first, then route
on artifact type within each row). Surface this composition
possibility so the future B3-N entry that revisits OQ-3 has a
starting point instead of re-deriving from scratch.

[minor] P5: "Recommendation §The router — Flow 6 row,
feedback-protocol.md framed as optional" — the framing is
reasonable (Flow 6 edits often skip /critique), but if /critique
DOES run, feedback-protocol.md becomes load-bearing as a
dependency of the /critique command itself
(.claude/commands/critique.md grounds on it). Add the
conditional dependency to the row: "optional; load-bearing if
/critique is run".

Acceptance criteria sweep:
- AC-1 (path header): pass.
- AC-2 (required sections present and in order): pass.
- AC-3 (≥2 options): pass — four options A/B/C/D.
- AC-4 (Recommendation grounded): pass — cites ADR-0049,
  -0050, -0051, -0048, CONTRIBUTING.md, CLAUDE.md; D0
  marked new-contribution-requires-review.
- AC-5 (no external naming as justification): pass.
- AC-6 (Open Questions marked): pass; minor sharpening
  noted above for OQ-3.
- AC-7 (Promotion target concrete): pass —
  docs/adr/0052-session-reading-router.md.
- AC-8 (≥1 critique round): pass after this file commits.
- AC-9 (blocking findings addressed): N/A — no blocking
  findings in round 1.
- AC-10 (decision-log row updated): pending step 9 of the
  loop; not yet a violation.

Summary: 0 blocking / 5 important / 5 minor. The five
important findings concentrate on §Recommendation and on the
P-B3RR.4 exhaustiveness premise. Round 2 expected after the
operator dispositions the round-1 findings.
```

## Operator Response

- **[important] P4 cost-savings claim** — *applied as
  recommended*: §Consequences §1 will qualify the claim as
  "by line count, not benchmarked against attention cost"
  and add a follow-up OQ to measure on the first 2–3
  post-router sessions.

- **[important] AC-2 P-B3RR.4 exhaustiveness** — *applied as
  recommended*: P-B3RR.4 will explicitly address B-row
  triage, study revival, and ADR supersession sessions —
  arguing why each collapses into the five named types or
  deferring with an explicit out-of-scope marker.

- **[important] P5 Flow 6 row PR-flow inheritance** —
  *applied as recommended*: the Flow 6 row will be tightened
  to spell out that "step 10 only" names the load-bearing
  playbook content, not a skip of Flow 5 inheritance.

- **[important] P5 wave-3-acceptance-criteria.md
  implementation-slice gap** — *applied with variation*: a
  sixth row will be added for implementation-slice sessions
  landing under a closed B-row, with AC-W3-3 and AC-W3-7 in
  the load-bearing set. This expands the router to six rows
  and amends P-B3RR.4 to reflect the new exhaustive set.
  The variation: the new row covers "implementation slice
  landing under a closed B-row" rather than splitting by
  originating wave, because the AC-W3 gate semantics apply
  uniformly to any post-Wave-3 implementation slice.

- **[important] P5 D0 re-routing language** — *applied as
  recommended*: D0 will separate the formal ADR-0049 §(a)
  outcome (rejected) from the practical re-routing
  follow-up (Flow 6 PR for the substance).

- **[minor] AC-2 self-referential floor bullet** — *applied
  as recommended*: the §6 self-reference will be rewritten
  as an intentional reminder.

- **[minor] P5 /sync-agents OQ-2 dependency** —
  *applied as recommended*: §Consequences §9 will add a
  pointer to ADR-0051 OQ-2 as a concrete forcing function.

- **[minor] P5 under-reading drift safety belt** — *applied
  as recommended*: the router preamble will add the
  "default to the next-larger reading set if unsure" rule.

- **[minor] AC-6 OQ-3 composition** — *applied as
  recommended*: OQ-3 will note that session-type and
  artifact-type axes may compose, so the future B3-N entry
  has a starting point.

- **[minor] P5 Flow 6 row feedback-protocol conditional** —
  *applied as recommended*: the row will spell out
  "optional; load-bearing if /critique is run".

All findings dispositioned. Revision follows in the next
commit on this branch.
