<!-- path: studies/critiques/2026-05-30-amendment-adr-0015-single-user-codeowners-critique-1.md -->

# Critique — `studies/decisions/2026-05-30-amendment-adr-0015-single-user-codeowners.md` — round 1

## Blocking findings

**None.** R6 path header present. AC-1…AC-4, AC-7 satisfied:
seven required sections in order, three real options (Option D
correctly noted as technical-impossibility not a counted
alternative), recommendation grounded in ADR-0015 + CONTRIBUTING.md
Flow 5 + ADR-0049 §(a) + `adr-writing` A4. R5 commodity
exemptions correctly applied (GitHub is environment; the operator's
actual GitHub handle is the repo's deployment-substrate owner,
not external prior-art). R8 forward-only (no backlinks to
`studies/`). The standalone-amendment classification is surfaced
cleanly at the top of Metadata + restated in Consequence #1. The
second-order consequences (a)…(e) are surfaced explicitly as
instructed — not buried.

## Important findings

- **[important] AC-10 / AC-2: "Consequences → item 9" — AC-10
  deviation needs explicit framing in this study.** AC-10 reads
  *"The matching row in `studies/foundation/06-decision-log.md`
  is updated to `resolved-study` and links to the file."*
  Consequence #9 commits *"No B-row is opened … the
  amendment-promotion session adds an 'Earlier update' entry to
  the decision log."* The amendment shape genuinely has no
  B-row to update (there is no originating B-row, unlike B2-36
  → ADR-0054 where the existing B2 row was the anchor), but
  AC-10's literal pattern isn't explicitly addressed — a
  reviewer applying AC-10 mechanically would flag the study as
  incomplete. Add a one-line note to Consequence #9 (or as a
  new Consequence #11): *"AC-10 from `acceptance-criteria.md`
  is satisfied vacuously — there is no B-row to update because
  this amendment was initiated by operator session prompt
  without an originating B-row (unlike B2-36 → ADR-0054 where
  the originating B-row provided the anchor). The decision-log
  'Earlier update' entry lands at promotion-session close per
  the precedent shape used by the Wave-S full-gate declaration
  (PR #116) for governance-event records without a B-row."*
  Avoids review-time confusion.

- **[important] AC-2 / R3: "Consequences → item 6" — Flow 5
  refresh trigger is procedurally underspecified.** Consequence
  #6 says the amendment triggers *"a one-line refresh of Flow 5
  making explicit that under the single-user model the `[H]`
  step is operator-as-person discipline. This refresh is a
  Flow-6-shape doc edit that lands with the amendment ADR's
  implementation slice (or in a same-PR R4 collapse if the
  operator authorizes)."* Three concrete questions are
  unaddressed: (a) is `CONTRIBUTING.md` Flow 5
  authoritative-via-ADR-0051 such that editing it requires an
  ADR amendment to ADR-0051 (or to ADR-0009), not Flow 6
  housekeeping?; (b) does the refresh ship in the same PR as
  the amendment-promotion ADR (R4 collapse, precedent
  ADR-0054/0055/0056) or as a separate Flow 6 PR?; (c) is the
  refresh text scoped to a clause-block or sentence? Either
  commit one path explicitly in Consequence #6 or surface as a
  new OQ-5 so the promotion session knows which discipline the
  refresh follows. The current "(or in a same-PR R4 collapse
  if the operator authorizes)" parenthetical leaves the
  procedural shape ambiguous.

- **[important] AC-6 / P5: "Open Questions → OQ-1" — the
  operator's instruction was *"the reading I lean toward …
  but the study defends it, I don't"*; the study's `**My
  recommendation: STAYS.**` framing reads as a settled
  disposition inside the study rather than a surfaced position
  for ratification.** The substance of the recommendation is
  sound and well-defended; the framing label undersells how
  operator-ratifiable the disposition is. Reframe as
  `**Author's recommendation (NOT pre-decided; surfaced for
  operator ratification per CONTRIBUTING.md Flow 5): STAYS.**`
  to match the operator's stated discipline (*"the
  recommendation is mine to write; the ratification is the
  operator's"* already captures this in the line below — fold
  the qualifier into the recommendation header so the
  disposition shape is visible at a glance).

## Minor findings

- **[minor] AC-6: "Open Questions → OQ-1" — non-standard
  out-of-scope marker.** OQ-2/3/4 carry the standard
  *"Out-of-scope for current cycle resolution"* marker per
  AC-6's wording. OQ-1 carries *"to be contested by /critique"*
  + *"operator ratification"* instead — semantically equivalent
  (the disposition is clear: not deferred, surfaced for
  operator), but a reviewer running AC-6 against the marker
  text mechanically would flag OQ-1 as missing the marker.
  Either align with the standard wording or add a one-line
  explicit-deviation note.

- **[minor] AC-2: "Recommendation → §3" — *"the SAME human
  reviews both halves of every merge by construction"* is
  imprecise.** A single-author repo has no review at all
  (author cannot self-approve in default GitHub
  branch-protection); the asymmetric-review clause from
  ADR-0015 §Consequence 1 isn't *"satisfied"* — it's **moot**
  (no second author exists for asymmetry to operate over).
  Restate as *"the enforcement-via-CODEOWNERS-group framing in
  ADR-0015 §Consequence 1 becomes moot under the single-user
  model — no second author exists for asymmetric review to
  operate over"*.

- **[minor] AC-2: "Promotion target" — doesn't explicitly note
  this study PR does NOT touch `docs/adr/0015-codeowners.md`.**
  The paragraph reads as if the Status-line update happens
  somewhere undefined; add *"This study PR does NOT edit
  `docs/adr/0015-codeowners.md`; the Status-line update +
  amendment-marker addition land at promotion time per
  `adr-writing` A4 idiom (same shape as ADR-0010's Status line
  was updated when ADR-0017 amended it, and ADR-0042's Status
  line was updated when ADR-0054 amended it)."*

- **[minor] AC-2: "Context → load-bearing clauses" — five
  separate file artifacts get block-quoted, totalling ~25 lines
  of verbatim quotes.** Polish — could be trimmed to
  single-sentence essentials each.

- **[minor] AC-2: "Consequences → item 5 (branch-protection
  precondition)" — important enough to be forward-pointed to
  the eventual amendment ADR's Notes block.** Add half-sentence
  *"This deployment-precondition fact MUST be recorded in the
  promotion ADR's Notes block per `adr-writing` A7."*

## Disposition reminder

`/critique` runs in a session whose author also wrote the study
(author-equals-reviewer circularity per ADR-0051 §Consequence 7
— and this amendment makes that circularity the **only** review
discipline under single-user; OQ-1 surfaces a related concern).
This critique does not ratify the deny-block question; OQ-1 is
operator-side per CONTRIBUTING.md Flow 5 §"Operator-side
responsibilities" and will be settled at merge act unless the
operator explicit-ratifies mid-PR (precedent B3-4 / B3-5 /
Wave-S declaration). The critique's job is to make the surface
defensible against a reviewer who hasn't seen the session —
under single-user that reviewer is the operator at a later
moment, which raises the bar on the artifact's standalone
clarity.
