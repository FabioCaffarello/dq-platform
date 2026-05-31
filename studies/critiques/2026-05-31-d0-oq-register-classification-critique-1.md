<!-- path: studies/critiques/2026-05-31-d0-oq-register-classification-critique-1.md -->

# Critique — D0 OQ-register classification (round 1)

- Target: [`studies/decisions/2026-05-31-d0-oq-register-classification.md`](../decisions/2026-05-31-d0-oq-register-classification.md)
- Round: 1
- Date: 2026-05-31
- Result: **0 blocking / 7 important / 8 minor**
- Disposition (filled at apply-time): see Operator Response trailer

---

## Frame

Coverage: R1, R5, R6, P1 explicitly considered; AC-1 through
AC-10 systematic walk. Path-header present (R6); no production
code (R1 — Waves 1/2 closed in any case); no external-vendor /
sibling-team / prior-art naming (R5); declarative principles
uninvolved (P1); the seven required sections appear in order;
the two options each defended on their own terms; the Open
Questions section's four entries are all explicitly marked
`out-of-scope for current cycle` (AC-6); the Recommendation
cites real grounding (AC-4); the R5 new-contribution markers are
loud and named (AC-4).

---

## Findings

### Blocking

**No `blocking` findings.**

The recommendation (Option A — Flow 6) holds against the
critique: no finding below moves the lane choice. The
substantive design tensions (labeled vs unlabeled deferrals;
uniform vs heterogeneous vocabulary) are correctly deferred to
the register's §Open Questions rather than absorbed silently.

### Important

- **[important] AC-7: "Promotion target" — the section commits
  a dual branch ("If Option A is ratified: no ADR. If Option B
  is ratified: `docs/adr/0060-oq-index-tooling.md`"). Under
  Option A (the recommended branch), no concrete
  `docs/adr/<NNNN>-<slug>.md` exists, which AC-7's strict
  reading expects.** This is a structural mismatch between a
  classification D0 and the AC-7 wording authored for
  single-path B-row studies. Either reframe AC-7's
  applicability to this artifact shape explicitly in the §"Why
  this is a D0" preamble, or surface the mismatch in the Open
  Questions section so the operator's ratification covers the
  AC-7 reading.

- **[important] AC-10: "Status / decision-log row" — the
  artifact has no row in `studies/foundation/06-decision-log.md`
  today; AC-10 expects "the matching row" to move to
  `resolved-study`.** The D0 implicitly assumes that
  classification D0s do not need a row (Option A) or open one at
  ratification (Option B). Make this implicit assumption
  explicit, ideally by adding a one-line note in the §"Why this
  is a D0" preamble or by registering a row-creation step in the
  per-branch §Consequences.

- **[important] R2: "Context" — the parenthetical claim that
  ADRs 0040 / 0042 / 0043 / 0044 carry a §"Open Questions"
  subsection is factually wrong.** In all four, the labeled
  OQ-N items live inside §Consequences (Consequence #10 in
  0040 / 0042 / 0043; Consequence #11 in 0044), not in a
  separate §"Open Questions" section. The list of ADRs with a
  discrete §"Open Questions" subsection narrows to ADRs 0032,
  0051, 0052, 0053, plus the §"Open Questions"-style notes
  embedded inside §Notes for 0058 / 0059. Rewrite the
  parenthetical to match the actual surface structure, or drop
  the subsection claim altogether — "registering items
  explicitly deferred" carries the load without it.

- **[important] R3: "Decision Drivers DD-2" — the phrase
  "tooling that carries new contract belongs in B3-N"
  recharacterizes [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
  §(a) Condition 4.** The committed wording is "materially
  novel rather than incremental". Replace the D0's gloss with
  the ADR's verbatim phrasing (or a tight paraphrase that
  preserves the threshold framing). The current wording reads
  as if "new contract" is the gate, when ADR-0049's gate is
  "materially novel".

- **[important] AC-2: "Recommendation — Status vocabulary
  marker" — the register's status terms (`open` /
  `consumed-by:<ADR>` / `superseded-by:<ADR>`) diverge from the
  decision-log's existing §"Status Vocabulary" (`open` /
  `in-progress` / `resolved-study` / `resolved-adr` /
  `rejected`).** The new-contribution marker calls this out,
  but the divergence has design weight: a future reader
  scanning the decision-log will see two vocabularies in the
  same file. Either (a) reuse the existing vocabulary (e.g.,
  `open` and `resolved-adr` with the consuming ADR linked), or
  (b) commit to a separate "Open Questions Register status
  vocabulary" subsection that names the divergence explicitly.
  The current draft leaves the gap silent.

- **[important] R2: "Appendix A — column sourcing rule
  unstated" — the "One-line description" column for ADR-0058
  OQ-1…OQ-5 and ADR-0059 OQ-1…OQ-6 is derived from the source
  studies, not from ADR §Notes (which only state "OQ-1 through
  OQ-5" / "OQ-1 through OQ-6" without restating each).** The
  register's per-OQ description sourcing rule is silent. Commit
  a rule in §"The substance of the proposal" (e.g.,
  "description = ADR §Notes restatement if present, else
  source-study OQ heading"), and note that the rule means the
  decision-log's register may carry study-derived text under R8
  (the decision-log itself is in studies/, so this is
  consistent; but the sourcing path matters for amendment
  maintenance).

- **[important] AC-5 / R2: "Appendix B — count column
  unreliable" — multiple rows show count-vs-examples
  mismatch.** ADR-0014: column says 6, the slash-separated
  examples list 7 items (manifestz / error-code / v2 path-bump /
  self-link / authentication / rate-limit / gRPC). ADR-0025:
  column says 4, examples list 6. ADR-0030: includes
  "quantitative ceilings (N/A)" in the deferral count, but
  ADR-0030 §Notes explicitly says those ceilings "do not apply
  here" — that's a non-applicability, not a deferral. ADR-0047:
  includes "mutating endpoints (closed-off)", which §Notes
  explicitly closes off — not a deferral either. The "~140
  across 45+ ADRs" aggregate inherits these errors. Either (a)
  tighten the count by re-walking each row's §Notes against a
  deferral-vs-other heuristic, or (b) drop the count column
  entirely and keep only the examples list. The current claim
  risks the operator over-weighting the unlabeled-deferral
  surface relative to its actual size.

### Minor

- **[minor] AC-1 / R6: "Appendix B anchor" — link
  `#b-unlabeled-prose-deferrals--not-in-v0` likely does not
  match the GitHub-rendered slug** (typically
  `appendix-b--unlabeled-prose-deferrals--not-in-v0` for an H2
  starting with "Appendix B —"). Test the anchor in the
  rendered PR preview and adjust, or use the in-document
  section number directly rather than the anchor.

- **[minor] AC-2: "Recommendation — pacing wording" — uses
  "demand-driven posture" once and "demand-driven pacing"
  elsewhere; ADR-0049 §"Premises" labels P-B3.3 as
  "Demand-driven pacing".** Normalize on "pacing" for
  consistency with the source.

- **[minor] AC-3: "Sub-option not considered separately"
  disclaimer is honest but reads as if AC-3 might fail without
  it.** AC-3 is satisfied by Options A and B; the null-option
  closure is supplementary. Either keep the disclaimer as
  supplementary, or move it into Option B §"Cons" as item (l)
  extending — current placement reads slightly defensively.

- **[minor] AC-2: "Consequences (Option A) item 1" says
  "new H2 section adjacent to §Wave Gates" without specifying
  before/after.** Pick one (e.g., "immediately after §Wave
  Gates, before §'Recommended Next Sequence'") so the operator
  does not have to make the call at PR-edit time.

- **[minor] AC-2: "New contribution markers (R5)" lists three
  items but the third ("deliberate scoping of v0 to *labeled*
  OQs only") is also captured in §Open Questions OQ-1.** The
  two surfaces register the same point twice; consolidate or
  cross-reference so a future reader updating the scope only
  has to touch one place.

- **[minor] AC-2: "Open Questions OQ-2" says "one ADR uses
  `OQ-G3.1`"; only ADR-0051 carries that shape today, but the
  D0 should also note that the labeling convention is informal
  across all of ADR-0001 through ADR-0035 (most use unlabeled
  prose).** The OQ-2 framing currently implies the labels are
  uniformly OQ-N with one outlier, when the actual surface is
  more heterogeneous (most pre-Wave-3 ADRs use no labels at
  all). Appendix B observation (n) already names this; tighten
  OQ-2 to reference observation (n) directly.

- **[minor] AC-2: "Recommendation" — commits a specific
  "B3-8" working slug.** Per `CONTRIBUTING.md` Flow 5
  §"Operator-side responsibilities" and
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clause 7, ADR-number reservation is operator-side; B-row
  numbering may be analogous. The slug is marked "working"
  which softens the claim, but flag the reservation step as
  operator-side when the branch is exercised, to mirror the
  ADR-number reservation rule.

- **[minor] AC-2: "Appendix B intro" says "Counts are
  conservative" but the count errors flagged in the important
  finding above suggest some counts are *over* the true
  deferral count (closed-off and N/A items included).**
  "Conservative" usually means "lower bound" — the
  ADR-0014 / ADR-0025 / ADR-0030 / ADR-0047 examples show
  that's not always true. Rephrase to "Counts are approximate —
  re-walk for exact figures".

---

## Summary

**0 blocking / 7 important / 8 minor.** The D0 frames the lane
decision cleanly, the recommendation is well-grounded, and the
appendix data is genuinely useful for the operator's call. The
important findings cluster around (a) factual errors in the
§Context ADR-list (I3) and the Appendix B counts (I7), (b)
shape mismatch with AC-7 / AC-10 for the classification D0
artifact-type (I1 / I2), (c) the status-vocabulary gap (I5),
and (d) one citation imprecision against ADR-0049 (I4). All
seven are fixable without rethinking the recommendation; round
2 lands them.

---

## Operator Response

- **Round-2 disposition:** all seven important findings applied
  in-place in the D0; the eight minor findings applied
  alongside per operator authorization to bundle round 2 with
  round 1. No second critique round produced — the round-1
  capture + the applied-edits diff is the durable record per
  [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)
  §"applied as recommended" disposition.
- **Specific dispositions:**
  - I1 (AC-7 dual-branch) — applied as recommended: surfaced
    in §"Why this is a D0" preamble; D0 declares the AC-7
    structural mismatch openly.
  - I2 (AC-10 row gap) — applied as recommended: surfaced in
    §"Why this is a D0" preamble alongside I1.
  - I3 (§Context factual error) — applied as recommended: the
    list of ADRs with a discrete §"Open Questions" subsection
    narrowed to 0032, 0051, 0052, 0053; ADRs 0040 / 0042 /
    0043 / 0044 acknowledged as OQ-labeled inside §Consequences.
  - I4 (DD-2 recharacterization) — applied as recommended:
    "materially novel rather than incremental" used verbatim.
  - I5 (Status vocabulary gap) — applied with variation:
    chose option (a) — reuse existing decision-log vocabulary
    (`open` / `resolved-adr` with consuming-ADR link) rather
    than mint a new subsection. Status terms in Appendix A
    updated to match.
  - I6 (Description sourcing rule) — applied as recommended:
    rule committed in §"The substance of the proposal".
  - I7 (Appendix B count column) — applied with variation:
    chose option (b) — drop the count column entirely; kept
    examples list only. Aggregate-total prose also updated
    away from "~140".
