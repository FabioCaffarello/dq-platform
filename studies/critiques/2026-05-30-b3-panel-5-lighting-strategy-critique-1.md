<!-- path: studies/critiques/2026-05-30-b3-panel-5-lighting-strategy-critique-1.md -->

# Critique — `studies/decisions/2026-05-30-b3-panel-5-lighting-strategy.md` — round 1

## Blocking findings

**None.** R6 path header present. AC-1…AC-7 all satisfied: seven
required sections in order, three options (≥2 per AC-3),
recommendation grounded in prior ADRs (AC-4), no sibling-team /
external prior-art names as justification (AC-5), every OQ
marked out-of-scope with a one-line reason (AC-6), promotion
target points to a concrete `docs/adr/0056-…` filename (AC-7).
R5 commodity exemptions (Prometheus, Kubernetes-via-ADR-0010)
correctly applied. The two-D0 framing on Conditions 1 + 3 is
surfaced honestly and not absorbed into the recommendation —
author-equals-reviewer circularity respected.

## Important findings

- **[important] AC-2 / ADR-0049 §(a): "Considered Options →
  Option B" — the classification claim "Not Rejected per
  ADR-0049 §(a) 'rejected' branch (the work is in-scope for
  some future wave)" contradicts the ADR-0049 §(a) Rejected
  text.** The §(a) Rejected definition is *iff the proposal
  falls outside the three in-scope families **and** outside any
  active wave's gate*. Option B fails Condition 2 (in-scope
  family) AND is outside any active wave's gate (no
  scheduler-binary wave exists). Both clauses hold → Option B
  IS Rejected per §(a), not a fourth category. Two ways to fix:
  (a) reclassify Option B as Rejected per §(a) and note that
  the §(a) Rejected branch explicitly allows the one-line
  decision-log entry pointing at a rationale (i.e., this study
  itself), or (b) defend the "in-scope for some future wave"
  carve-out as a new contribution proposed here per R5 and
  surface it as a fourth D0. (a) is the smaller surface; (b)
  opens a substantive §(a) precedent question that probably
  doesn't need opening for this study's purposes.

- **[important] ADR-0049 §(a) / AC-2: "Considered Options →
  Option A" — sub-paths A.x (drop
  `dq_scheduler_triggers_managed` from engine emission) and A.y
  (emit constant zero) are not classified separately under
  §(a), but they may have different amendment-vs-extension
  implications.** Under the strong reading of ADR-0039, both
  are amendment-shaped (removing a metric from the inventory
  vs. preserving it but flattening the contract's intended
  signal). Under the weak reading, A.x (dropping the metric)
  may itself be amendment (the metric name is committed in
  ADR-0039's inventory; engine declining to emit it = the
  inventory no longer matches what the engine ships), while A.y
  (constant zero) is more plausibly extension (the metric is in
  the inventory and emits a value; the value's information
  content is just minimal). The §Eligibility table treats
  Option A as a single unit; surface A.x vs A.y as a
  sub-classification so the operator's ratification has the
  granularity to disambiguate.

- **[important] P5 / AC-2: "Recommendation → §1 weak-reading
  branch" — the phrase "rename the metric, or qualify the
  label-source" collapses two mechanisms with different §(a)
  classifications.** Renaming `dq_queue_depth` to (e.g.)
  `dq_engine_inflight_runs` is amendment-shaped at any reading:
  the metric name itself is part of ADR-0039's committed
  contract surface (operators key on `dq_runs_total` etc.
  exactly because the name is the contract). Adding a label
  (e.g., `source="engine"` alongside the existing `state`
  label) is extension-shaped (additive within an
  engine-major-version per ADR-0039 §"Evolution rules").
  Sharpen the recommendation by separating these — additive
  label is the only weak-reading-and-extension-compatible
  option; renaming routes back through amendment regardless of
  the Condition 1 reading.

- **[important] AC-2 / ADR-0050: "Consequences → item 4" —
  "small ADR-0055 §Notes append" is underspecified.** ADR-0055
  was merged 2026-05-30; touching it post-merge is amendment
  territory governed by ADR-0050 §Consequence 4 (in-place
  Amendment-log subsection) or by a follow-up amendment ADR
  (per the ADR-0017 / ADR-0054 standalone-amendment pattern).
  The study's Consequence #4 names neither and leaves the
  mechanism ambiguous. Either (a) commit the correction as an
  Amendment-log subsection on ADR-0055 in the follow-on
  session's PR (light touch, per ADR-0050 §Consequence 4), or
  (b) defer the correction to a separate housekeeping session
  and record it as OQ-6 here, or (c) carry the correction
  inline in this study's Context (where it already lives in
  narrative form) and let the inheritance pattern from the
  decision-log B3-5 row's "Earlier update" entry surface it to
  future readers without touching ADR-0055 directly.

## Minor findings

- **[minor] AC-2: "Recommendation" — framing as "does not
  pre-decide" plus two branched recommendations (one per
  ratification outcome) is a *contingent* recommendation, not
  the absence of one.** Recast the lead sentence to "The
  recommendation is conditional on the D0 ratification — two
  coupled outcomes follow" so reviewers don't have to reconcile
  "no recommendation" with the substantive bullets below it.

- **[minor] AC-6: "Open Questions → OQ-1" — this study's OQ-1
  name ("Operator interpretation of ADR-0039's 'scheduler
  currently tracks' phrasing") collides nominally with B3-4
  OQ-1 ("Panel 5 lighting"), which is the parent.** Future
  cross-references will be ambiguous. Rename to (e.g.) "OQ-1:
  ADR-0039 Meaning column interpretation (D0 ratification)" so
  it disambiguates from the parent.

- **[minor] AC-2: "Eligibility under ADR-0049 §(a)" — the table
  doesn't carry a short legend mapping the weak/strong reading
  to its §(a) outcome.** Add a single line above the table:
  *"Weak reading → Path A is extension (B3-eligible); strong
  reading → Path A is amendment (out of B3 scope; routes
  through amendment process)."* The table currently embeds this
  mapping inline in Conditions 1 + 3, but a one-line legend
  makes the binary visible at a glance.

- **[minor] AC-2: "Decision Drivers → DD-4" — DD-4
  ("Reading-of-record on ADR-0039's Meaning column is not
  re-settled by this study") is descriptive of the D0's status,
  not a driver of option selection.** Drivers should answer
  "why is this option ranked here"; DD-4 answers "why is the
  ratification needed". Move to §Context as a contextual note
  or fold into the §Eligibility table's intro paragraph.

- **[minor] AC-2: "Eligibility under ADR-0049 §(a) → Summary" —
  "Same precedent shape as B3-2 (two D0s on Conditions 1 + 4,
  operator-ratified at merge)" is imprecise.** B3-5's D0s are
  Conditions 1 + 3, not 1 + 4. The structural similarity to
  B3-2 is "two coupled D0s, mid-PR or at-merge ratification per
  CONTRIBUTING.md Flow 5" — but the specific conditions differ.
  Either drop the "Same precedent shape as B3-2" line (the
  structural-shape claim is already made earlier in the
  paragraph) or qualify as "structural-shape precedent only —
  B3-2's D0s were on Conditions 1 + 4".

## Disposition reminder

`/critique` runs in a session whose author also wrote the study
(author-equals-reviewer circularity per ADR-0051 §Consequence
7). The two D0s on Conditions 1 + 3 are operator-side per
CONTRIBUTING.md Flow 5; this critique does not ratify them,
surface a ratification reading, or absorb the choice into its
findings. The operator picks the reading after PR open; the
critique's job is to make the surface defensible against a
reviewer who hasn't seen the session.
