<!-- path: studies/decisions/2026-05-31-d0-oq-register-classification.md -->

# D0 — Open Questions register: classification (Flow 6 vs B3-N)

- **Status:** resolved — **Option A (Flow 6) ratified at PR #126 merge on 2026-05-31**; register landed in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md)
  §"Open Questions Register" via Flow 6 direct edit. The D0
  stays in `studies/decisions/` as the reasoning artifact per
  R8; no ADR back-links to it. Step-number drift in this
  document's pointers (the body cites "step 10" for the
  decision-log update; the actual playbook home is step 9) was
  corrected in the Flow 6 landing PR's edits to
  [`06-decision-log.md`](../foundation/06-decision-log.md) and
  [`.claude/playbooks/post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md);
  this D0 is preserved as-written.
- **Date:** 2026-05-31
- **Type:** Classification D0 (Flow 6 direct-edit vs B3-N entry)
- **Critique rounds:** 1 (round 1 — 0 blocking / 7 important / 8 minor; all importants + minors applied in-place; capture at
  [`studies/critiques/2026-05-31-d0-oq-register-classification-critique-1.md`](../critiques/2026-05-31-d0-oq-register-classification-critique-1.md)
  per [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)).
- **Artifact proposed:** a live index of open Open Questions
  (OQs) currently registered as `out-of-scope for current cycle`
  across `docs/adr/*.md` §Notes / §"Open Questions" sections.
- **Why this is a D0:** the operator-facing meta-decision is
  *which lane* lands the artifact. Both lanes are defensible.
  The substance (the OQ inventory itself) is uncontentious; only
  the placement and maintenance posture are. Per
  [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5
  §"Operator-side responsibilities", classification readings of
  this kind are operator-ratified.
- **Artifact-shape disclosure (AC-7 / AC-10 against a
  classification D0):** the acceptance criteria
  [`acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md)
  AC-7 ("Promotion target line points to a concrete
  `docs/adr/<NNNN>-<slug>.md` filename") and AC-10 ("The
  matching row in `06-decision-log.md` is updated to
  `resolved-study`") were authored for single-path B-row
  studies. A classification D0 does not fit either cleanly:
  under the Option A branch (the recommendation) no ADR is
  promoted at all, and no B-row pre-exists to flip. The §Promotion
  target and §Consequences sections below name the dual-branch
  resolution explicitly per branch; the operator's ratification
  of the lane is also the ratification of which AC-7 / AC-10
  reading applies. This shape-mismatch is a **new contribution
  proposed here, requires review** per R5.

---

## Context

Every ADR promoted since [ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md)
carries a §Notes section registering items explicitly deferred
`out-of-scope for current cycle`. The surface where deferred items
live is heterogeneous:

- ADRs [0032](../../docs/adr/0032-baseline-strategy.md),
  [0051](../../docs/adr/0051-claude-tooling-postwave3.md),
  [0052](../../docs/adr/0052-session-reading-router.md), and
  [0053](../../docs/adr/0053-record-mode-skill.md) carry a
  discrete §"Open Questions" subsection (in 0032 it precedes
  §Notes; in 0051 / 0052 / 0053 it is embedded inside §Notes
  as a structured list).
- ADRs [0040](../../docs/adr/0040-entity-onboarding-workflow.md),
  [0042](../../docs/adr/0042-release-engineering-invariants.md),
  [0043](../../docs/adr/0043-logging-contract-specifics.md), and
  [0044](../../docs/adr/0044-external-artifact-references.md)
  embed OQ-labeled items inside §Consequences (Consequence #10
  in 0040 / 0042 / 0043; Consequence #11 in 0044).
- ADRs [0058](../../docs/adr/0058-record-runner-commit-after-dispatch.md)
  and [0059](../../docs/adr/0059-record-runner-commit-retry.md)
  reference OQ-1 through OQ-5 / OQ-6 by identifier inside §Notes
  and carry the full OQ text in their source studies (B3-6 /
  B3-7) per R8.
- Every other accepted ADR registers unlabeled prose deferrals
  inside §Notes.

Many authored items are OQ-N-labeled; many are unlabeled prose
deferrals. The total surface today is large enough that
discovering "what work has been deferred against ADR X" requires
opening ADR X by hand.

This presents two operational pains:

1. **Demand-driven pacing (P-B3.3 in
   [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md))
   has no visible menu.** B3-N entries are *born when concrete
   demand surfaces* against an existing OQ. But operators only
   know an OQ exists by recalling the ADR or grepping for the
   identifier. A reviewer encountering a candidate B3-N proposal
   today has no consolidated surface to check "is this already
   an OQ somewhere?" — and risks either duplicating an OQ as a
   fresh B3-N or missing that an OQ already names this work.

2. **The richest OQ clusters are now post-Wave-3 entries.**
   [ADR-0058](../../docs/adr/0058-record-runner-commit-after-dispatch.md) §Notes
   carries 5 OQs forward from the B3-6 study;
   [ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md) §Notes
   carries 6 OQs forward from B3-7. Both clusters concentrate on
   record-mode runner commit semantics — adjacent OQs whose
   demand-driven realization will arrive together once
   production telemetry lands ([B3-7 OQ-6](#a1--explicitly-labeled-oqs)).
   Discovering this cluster today requires reading both ADRs;
   discovering the related ADR-0055 / ADR-0056 emission-side OQs
   (which feed the same observability gap) requires reading two
   more.

A "menu" — a sorted, source-cited list of open OQs with the
named trigger condition for each — closes both gaps with no
new contract. The substance of an OQ in the register is the
substance the ADR §Notes already commits; the register is a
*derived view* over those §Notes.

The operator-facing question this document opens is **which
governance lane should land the derived view**.

### The substance of the proposal (uncontentious)

A new section, working title **"Open Questions Register"**, in
[`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md):

- one row per OQ explicitly labeled `OQ-N` (or `OQ-G3.1`-style) in
  an accepted ADR's §Notes / §"Open Questions" / §Consequences
  (whichever surface the source ADR uses per the §Context
  list above);
- columns: source ADR, OQ identifier, one-line description, named
  trigger condition (per P-B3.3 — "what concrete operational
  signal surfaces this OQ as a B3-N"), status (using the
  existing decision-log §"Status Vocabulary": `open` /
  `resolved-adr` with the consuming ADR linked; the register
  reuses the established terms rather than minting new ones —
  see also the §Recommendation new-contribution markers);
- **Description-sourcing rule:** the one-line description column
  carries (a) the ADR §Notes / §Consequences restatement of the
  OQ verbatim or lightly summarized when the ADR registers the
  full OQ text in-line (the typical case — ADRs 0032 / 0040 /
  0042–0044 / 0051–0053); (b) the source-study §"Open Questions"
  heading lifted into the register when the ADR references OQs
  by identifier only (ADRs 0058 / 0059, whose source studies
  B3-6 / B3-7 carry the full OQ text). The register lives in
  the decision-log (a `studies/` document); per R8, lifting
  study-derived text into the decision-log is consistent (both
  surfaces are in `studies/`; the R8 boundary at
  ADR→`studies/` is not crossed);
- sorted by source ADR ascending (then OQ identifier);
- unlabeled-prose deferrals are **out-of-scope for v0** — see
  §"Open Questions" of this D0 (the meta-question is whether
  unlabeled deferrals deserve a parallel register surface, or
  whether the lack of an OQ-N label IS the signal the author
  did not want it tracked);
- maintenance: each new ADR's promotion PR adds rows for any new
  labeled OQs; each new B3-N ADR's promotion PR flips the
  consumed OQ's status to `resolved-adr` (with the consuming
  ADR linked in the description column).

The proposed v0 contents are in [Appendix A](#a1--explicitly-labeled-oqs)
below. The unlabeled-prose deferrals deliberately deferred from
v0 are inventoried in [Appendix B](#appendix-b--unlabeled-prose-deferrals--not-in-v0)
so the operator can see what is being held back.

---

## Decision Drivers

- **DD-1 — Lane selection follows the artifact's nature.**
  [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6 frames
  three eligibility shapes for the direct-edit lane: factual
  refreshes, tight clarifications of existing rules, and
  cross-document propagation of an already-promoted ADR's
  contract. The OQ Register is a *derived view of existing
  §Notes content*. If that shape clears the Flow 6 eligibility
  test, the lane is Flow 6; if it does not, the lane is B3-N.
  This driver tests the artifact against the lane definitions
  rather than against the operator's discretion.

- **DD-2 — P-B3.3 demand-driven pacing extends to tooling
  itself.** A `make oq-index` generator that derives the
  register from ADR §Notes at build time is the *tooling-shaped
  realization* of the same artifact (no manual register; the
  register IS the generated output). Per P-B3.3 and
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
  §(a) Condition 4, the B3-N gate is reserved for changes that
  are "materially novel rather than incremental" — and the
  ADR's framing pairs this with the demand-driven pacing of
  P-B3.3. A `make oq-index` generator would cross the
  materially-novel threshold (new scanner, new output target,
  potentially a new CI gate); but the demand for it has not
  surfaced — there is no operator pain "the register I
  hand-maintain has drifted because I forgot to update it on
  PR #X". The first version of the register has to exist
  before tooling against it can be justified.

- **DD-3 — Audit-trail cost matches the meta-decision's
  weight.** Flow 6 records its meta-decision in the PR body
  only (no B-row, no ADR). B3-N records it in a study + ADR
  with critique rounds preserved per ADR-0048. The artifact
  proposed here introduces no contract that future contributors
  must conform to — adding rows for a new ADR's OQs is
  mechanical, not architectural — so the heavier B3-N audit
  trail does not earn its weight. The lighter Flow 6 audit
  trail matches.

- **DD-4 — Precedent for "new derived-state section in the
  decision-log" is Flow 6.** The decision-log §Wave Gates
  section was added via Flow 6 (per memory entry
  `wave-closure-flow6-precedent.md`: "wave-gate closure is a
  Flow 6 dated-closure-note, NOT an ADR ceremony; decision-log
  §Wave Gates entry + Last updated + CLAUDE.md/AGENTS.md flips;
  precedent Wave 1/2/3 + Wave-S full gate"). The OQ Register
  shares the §Wave-Gates shape: derived state, hand-maintained,
  no contract surface, lives in the decision-log file.

- **DD-5 — R3 settled-decisions guard.** The OQ Register does
  not amend any ADR's §Notes content; it carries forward the
  §Notes text verbatim (or one-line summary) under each OQ's
  row. R3 is preserved by construction.

- **DD-6 — R8 published-vs-studies boundary.** ADR §Notes is
  the publication surface; the decision-log is in `studies/`.
  Per R8, ADRs do not link backward into `studies/`, but the
  decision-log (a `studies/` document) freely references ADR
  paths. The Register lives in the decision-log; its links
  point *into* `docs/adr/`, which is the allowed direction.

---

## Considered Options

### Option A — Flow 6: hand-maintained register in the decision-log

A new section "Open Questions Register" lands inside
[`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md)
via a Flow 6 PR per
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6. The PR body
is the only durable record of the meta-decision (audit-trail
clause).

The seed content is [Appendix A](#a1--explicitly-labeled-oqs)
of this D0, copied into the register immediately after the
§Wave Gates section and before §"Recommended Next Sequence".
Future maintenance: each new ADR's promotion PR appends rows
for any newly labeled OQs in the same PR (one extra hunk in
the decision-log edit). Each new B3-N (or amendment) ADR's
promotion PR flips the consumed OQ's status to `resolved-adr`
and adds the consuming ADR's path link to the description
column.

**Why this fits Flow 6.** Three of Flow 6's stated eligibility
shapes apply:

- *Factual refresh* — the register surfaces facts that already
  exist in §Notes; nothing is invented.
- *Pointer to live state* — the register is exactly a "pointer
  to live state" entry of the kind Flow 6 lists.
- *Cross-document propagation* — the register propagates ADR
  §Notes content (the source of truth) into a consolidated
  view in the decision-log (a satellite catching up).

**Maintenance cost.** Adding a row at ADR promotion time is
roughly equivalent in effort to the existing `Last updated`
line maintenance in the decision-log. The maintenance discipline
fits inside the existing post-Wave-3 session loop step 10
(decision-log update) without a new step.

**Drift risk.** The register can drift from §Notes if a future
ADR amendment edits an OQ's wording without touching the
register. The Amendment-log convention from
[ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md)
§Consequence 4 already shapes how such amendments record their
deltas; the register update is one additional hunk in the
amendment PR (same shape, same lane). Drift bounded by
review discipline, not by tooling.

**Pros.**

- (a) Lowest activation energy — the register lands today; no
  tooling to write, no harness verification cycle.
- (b) Matches §Wave-Gates precedent — same lane (Flow 6), same
  file (decision-log), same shape (derived state hand-
  maintained).
- (c) P-B3.3 conformance — tooling waits for demand; the
  register's value is provable before tooling is built.
- (d) No new contract — future ADR authors do not learn a new
  rule beyond "if you defer with an OQ-N label, also add a
  row to the register". This is a one-line extension of the
  existing AC-6 discipline.

**Cons.**

- (e) Manual maintenance can rot — a future ADR's promotion PR
  could land without the register row, and the gap survives
  until a reader notices.
- (f) No CI gate — no automated check that the register's row
  count matches the §Notes-discovered OQ count.

Both cons land in the [Open Questions](#open-questions) of
this D0 as the natural seam where B3-N tooling becomes
justified once Option A reveals concrete pain.

### Option B — B3-N entry: tooling extension (`make oq-index` generator)

A new B3-N row in the decision-log (working slug **B3-8 —
`make oq-index` generator**) opens a study under
[ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §(a)
that proposes a generator under `tools/` (or `scripts/`,
depending on placement decision) that scans `docs/adr/*.md`
for OQ-N labels and emits a Markdown index. Output target
is either:

- (B-i) a generated section in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md)
  delimited by `<!-- BEGIN/END oq-index -->` markers, or
- (B-ii) a separate generated file
  [`studies/foundation/07-open-questions.md`](../foundation/07-open-questions.md)
  the decision-log links to, or
- (B-iii) a standalone CI artifact (no committed file; CI
  publishes the rendered HTML on each main-branch push).

Sub-choice (B-i / B-ii / B-iii) is part of the B3-N study's
scope.

**Eligibility under [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §(a):**

- *Condition 1 (P-B3.1 expands not rewrites)* — the generator
  extends contract coverage of the `tools/` workspace with a
  new failure mode (the OQ-index target missing or stale at
  CI time) the operator must understand. Clears.
- *Condition 2 (one of three in-scope families)* — Tooling
  extensions family per ADR-0049 §"Per-family scope" → "the
  manifest publisher, the dry-run runner, the engine
  dispatcher, and adjacent tooling that extend contract
  coverage without changing the contract shape". The OQ-index
  generator is "adjacent tooling that extends contract
  coverage" by surfacing the deferred-OQ pipeline. Clears
  cleanly per the same precedent ADR-0053 set for
  documentation-extending tooling.
- *Condition 3 (conforms to envelope ADRs)* — does not
  touch [ADR-0020](../../docs/adr/0020-wave-s-launch.md) /
  [ADR-0021](../../docs/adr/0021-mode-as-primitive.md) /
  [ADR-0022](../../docs/adr/0022-kind-catalog.md) /
  [ADR-0023](../../docs/adr/0023-sources-schema.md). Clears.
- *Condition 4 (additive-maintenance threshold)* — the
  generator is materially novel rather than incremental: it
  introduces a new build-time scanner, a new output artifact
  shape, and a new CI gate (if (B-i) or (B-ii) chosen, a
  byte-equality gate on the generated section/file). Clears.

All four conditions clear. The B3-N path is *legally available*
under ADR-0049. The question is whether it is *load-bearing now*.

**Pros.**

- (g) Drift-proof by construction — the register's content is
  always synchronized with §Notes; manual maintenance vanishes.
- (h) CI-gate-able — the byte-equality test of (B-i) / (B-ii)
  catches the missing-row failure mode Option A leaves to
  reviewer discipline.
- (i) Audit-trail-richer — the B3-N study records *why* the
  register exists in a study + ADR pair, not in a PR body
  that scrolls off the GitHub UI after a year.

**Cons.**

- (j) Premature per P-B3.3 — demand for tooling has not
  surfaced. ADR-0049 §(a) Condition 4 framing "the change is
  materially novel rather than incremental" gates B3-N
  entries on demand; the demand for a *manual* register is
  arguable, but demand for a *generated* register requires
  the manual register to first prove its value AND drift.
- (k) Higher activation energy — a study + critique + ADR +
  implementation cycle delays the register's first version
  by at least one session.
- (l) Risk of premature contract — the B3-N ADR commits the
  generator's input grammar (which OQ-label shapes are
  recognized: only `OQ-N`? also `OQ-G3.1`-style? `OQ-W3-N`?).
  Today's grammar is informal; committing it in code before
  it stabilizes risks rework.

### Null option (not enumerated separately)

A null option ("do nothing, leave OQs scattered in §Notes") is
mentioned for completeness but not enumerated as a third option:
§Context above commits the proposition that a register has
positive value, and the operator's lean per the session prompt
is Option A. The ratification gate this D0 surfaces is between
A and B, not against doing nothing. AC-3's two-options-minimum
is satisfied by A and B.

---

## Recommendation

**Adopt Option A — Flow 6 direct edit.** Land the register's
v0 content from [Appendix A](#a1--explicitly-labeled-oqs) in the
decision-log under a new H2 section "Open Questions Register"
placed immediately after §Wave Gates and before §"Recommended
Next Sequence". Defer tooling (Option B) until Option A surfaces
either (or both) of:

- **Trigger T-A.1** — drift incident: a B3-N proposal lands
  citing an OQ that the register did not show as open (because
  a prior PR forgot to add the row, or because an ADR
  amendment moved the OQ without updating the register).
- **Trigger T-A.2** — register volume: the labeled-OQ count
  surfaces sustained drift between PRs (e.g., three
  consecutive ADR promotions each miss the register-row
  update at review time).

Either trigger turns Option B from premature into demand-driven.
Until one fires, manual maintenance is the operationally
correct shape.

**Grounding citations:**

- Flow 6 eligibility test → [`CONTRIBUTING.md`](../../CONTRIBUTING.md)
  Flow 6 §"Scope — what qualifies" (factual refreshes, pointers
  to live state, cross-document propagation).
- §Wave Gates precedent → decision-log §Wave Gates lineage
  (Wave 1/2/3 + Wave-S full gate; Flow 6 by precedent per
  memory `wave-closure-flow6-precedent.md`).
- P-B3.3 demand-driven pacing →
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §"Premises".
- ADR-0050 Amendment-log shape for future register-row edits
  in-place → [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md) §Consequence 4.

**New contribution markers (R5).**

- The OQ Register's column shape (source ADR / OQ id /
  description / trigger / status) is **new contribution
  proposed here, requires review**. No prior register exists
  to inherit a shape from.
- The status vocabulary **reuses the existing decision-log
  §"Status Vocabulary"** (`open` / `resolved-adr`) rather than
  minting new terms. When a B3-N (or amendment) consumes an
  OQ, the OQ row's status flips to `resolved-adr` and the
  description column gains the consuming ADR's path link.
  This carries forward the meaning the decision-log already
  commits and avoids two vocabularies in the same file.
- The deliberate scoping of v0 to *labeled* OQs only is
  surfaced as [OQ-1](#open-questions) of this D0 rather than
  as a separate R5 marker. Reviewers may push back to either
  widen v0 or formalize the labeling rule going forward; the
  Open-Questions seam is the canonical surface for that push-
  back per AC-6.

---

## Consequences

If Option A is ratified:

1. The Flow 6 PR lands [Appendix A](#a1--explicitly-labeled-oqs)
   inside the decision-log under a new H2 section "Open Questions
   Register" placed immediately after §Wave Gates and before
   §"Recommended Next Sequence". The decision-log's `Last
   updated` line gains the dated note. No B-row is opened.
2. The post-Wave-3 session loop step 10's decision-log update
   discipline gains a one-line extension: "if the promoted ADR
   labeled new OQs, add register rows for them in the same PR."
   This extension is itself a Flow 6 refresh of the playbook,
   not an amendment.
3. The maintenance-cost expectation is recorded — a row per
   promoted ADR with labeled OQs, plus a status flip on B3-N
   consumption.
4. Triggers T-A.1 and T-A.2 sit in the register's §Open
   Questions as the named demand signal that opens Option B.
   No B3-N row pre-allocated; the row is opened when a trigger
   fires.

If Option B is ratified:

1. A new B3-`<N>` row opens (B-row number operator-reserved at
   ratification time per §Promotion target); this D0 becomes
   the seed of the B3-`<N>` study (renamed
   `studies/decisions/2026-05-31-b3-<N>-oq-index-tooling.md`).
2. The register itself does not land today; it lands as the
   first implementation slice of the B3-`<N>` ADR, under the
   operator-authorized R4 scope-collapse precedent (ADR-0054
   onwards) if the operator authorizes collapse, otherwise as
   a follow-on session.
3. Sub-choice (B-i / B-ii / B-iii) is scoped inside the
   B3-`<N>` study.
4. The Flow 6 path is closed off as inappropriate for tooling;
   future tooling-shaped artifacts of the same kind cite the
   B3-`<N>` ADR as precedent.

If neither option is ratified (the operator overrides the
recommendation and elects something else, e.g., scope the v0
register to all deferrals including unlabeled), the D0 is
re-drafted under the new framing and re-circulated.

---

## Open Questions

- **OQ-1 — Unlabeled-prose deferrals.** v0 scopes the register
  to OQ-N-labeled entries only. Many ADRs (every accepted ADR
  through ADR-0035 except ADR-0032 / ADR-0040 / ADR-0042 /
  ADR-0043 / ADR-0044, plus several post-Wave-3 entries) carry
  prose deferrals in §Notes without OQ-N labels (illustrative:
  [ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md) §Notes,
  [ADR-0014](../../docs/adr/0014-trigger-handler-contract.md) §Notes,
  [ADR-0024](../../docs/adr/0024-window-semantics.md) §Notes,
  [ADR-0027](../../docs/adr/0027-record-mode-cost-guardrails.md) §Notes,
  [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md) §Consequence 9).
  Whether these belong in the same register, in a parallel
  "Unlabeled deferrals" register, or stay out of scope
  indefinitely (the absence of an OQ-N label IS the signal)
  is **out-of-scope for current cycle** — decided by the first
  reader who hits a B3-N candidate against an unlabeled
  deferral and cannot find it in the register. [Appendix B](#appendix-b--unlabeled-prose-deferrals--not-in-v0)
  inventories the unlabeled deferrals so the operator can see
  the gap without merging the decision now.

- **OQ-2 — OQ-labeling convention going forward.** The
  labeling discipline is heterogeneous across the ADR
  history (per Appendix B observation (m)): most pre-Wave-3
  ADRs (0001–0035) use unlabeled prose deferrals; the
  labeled ADRs use flat `OQ-N` (most), one uses gap-prefixed
  `OQ-G3.1` ([ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)),
  and the B3-6 / B3-7 cluster (now ADR-0058 / ADR-0059) uses
  flat numbering against a source-study OQ space. Whether to
  commit a single labeling convention (e.g., flat `OQ-N`
  only) so the register's lookup grammar is uniform is
  **out-of-scope for current cycle** — the first labeling-
  collision incident reveals whether uniformity is needed.

- **OQ-3 — Cross-OQ adjacency clustering.** The B3-6 / B3-7
  cluster surfaces OQs that share an emission-slice trigger
  (ADR-0055 / ADR-0056 emission gap; B3-6 OQ-4 + B3-7 OQ-3 +
  B3-7 OQ-6 all wait on the same telemetry to land). Whether
  the register should support cluster-tags (so an operator
  looking at one OQ sees its adjacent partners) is
  **out-of-scope for current cycle** — premature until the
  register has been used by a real B3-N candidate session.

- **OQ-4 — Trigger T-A.1 and T-A.2 as opening signal for
  Option B.** §Recommendation commits two triggers as the
  demand signal that opens Option B. Whether the triggers
  themselves are the right shape (drift incident granularity?
  three-PR threshold for T-A.2?) is **out-of-scope for
  current cycle** — the first trigger firing reveals whether
  the threshold needs adjustment.

---

## Promotion target

**If Option A is ratified**: no ADR. The Flow 6 PR body of
this branch (`docs/decision/oq-register-classification`) is the
audit-trail surface per
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6 §"Audit
trail". The Flow 6 PR's edit-set is:
[`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md)
(new §"Open Questions Register" section + `Last updated` entry).
This D0 stays in `studies/decisions/` as the reasoning artifact
that grounds the PR body; per R8, no ADR back-links to it.

**If Option B is ratified**: this D0 is renamed
`studies/decisions/2026-05-31-b3-<N>-oq-index-tooling.md` and
becomes the B3-`<N>` study. The B-row number `<N>` and the ADR
number are operator-reserved at ratification time per
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5
§"Operator-side responsibilities" — ADR-number reservation is
explicit there; B-row number reservation is the parallel
discipline. The working figures used elsewhere in this D0
("B3-8", "ADR-0060") are indicative only and are confirmed or
revised by the operator at the moment the branch is exercised.
The decision-log gets a new B3-`<N>` row in the B3 table.

**Both paths**: the operator's ratification of the lane is
the gate. This D0 hard-stops at PR open for ratification per
the session prompt; no merge follows ratification without an
explicit go-ahead.

---

## Appendix A — Proposed v0 register content

### A.1 — Explicitly labeled OQs

The table below is the seed content for the proposed
"Open Questions Register" section. Format identical to what
would land in the decision-log under Option A; identical to
what would seed the B3-`<N>` study's implementation slice
under Option B.

Sort order: source ADR ascending, OQ identifier ascending.

| Source ADR | OQ id | One-line description | Trigger condition (per P-B3.3) | Status |
|---|---|---|---|---|
| [ADR-0032](../../docs/adr/0032-baseline-strategy.md) | OQ-1 | Strict-P2 baseline snapshot — per-execution baseline-row-set snapshotting for literal byte-identical re-runs | Concrete operational signal (incident or audit finding) that source-state delta matters in practice | open |
| [ADR-0032](../../docs/adr/0032-baseline-strategy.md) | OQ-2 | Cross-entity baselines — `params.baseline.cross_entity_scope` extension | Concrete need surfaces for cross-entity comparison; CODEOWNERS ownership-boundary review required | open |
| [ADR-0040](../../docs/adr/0040-entity-onboarding-workflow.md) | OQ-1 | Engine-level onboarding-channel mechanism — `_owners.yaml` v3 `onboarding: true` + `EnvConfig.OnboardingChannel` override | Marked B2 follow-up — surfaces when shared-substrate workaround friction crosses threshold | open |
| [ADR-0040](../../docs/adr/0040-entity-onboarding-workflow.md) | OQ-2 | Channel-reachability linter extension — Slack-API / SMTP / PagerDuty-API pinging | Marked B2 follow-up; bounded by ADR-0034's no-substrate-from-linter posture (would need ADR-0047 follow-on) | open |
| [ADR-0042](../../docs/adr/0042-release-engineering-invariants.md) | OQ-1 | Image registry choice — registry where `dq-<binary>:<tag>` images push | ADR-0008 host-primitive follow-up; substituted concurrently with `PLACEHOLDER-org/` per ADR-0015 §4 | resolved-adr ([ADR-0054](../../docs/adr/0054-engine-image-registry-amendment.md)) |
| [ADR-0042](../../docs/adr/0042-release-engineering-invariants.md) | OQ-2 | Release-cadence rhythm — when `engine-v*` tags push (weekly / on-demand / gate-driven) | Concrete release-cadence signal surfaces from operating the platform | open |
| [ADR-0043](../../docs/adr/0043-logging-contract-specifics.md) | OQ-1 | `DQ_LOG_LEVELS` extension to `tools/` long-running binaries | Long-running tool binary (e.g., a daemon) actually lands in `tools/` | open |
| [ADR-0043](../../docs/adr/0043-logging-contract-specifics.md) | OQ-2 | Per-call-site level overrides (beyond per-package) | Concrete operator signal demonstrates per-call-site need (has not surfaced) | open |
| [ADR-0044](../../docs/adr/0044-external-artifact-references.md) | OQ-1 | Cross-rule reference deduplication — content-addressed prefix to dedup inlined copies | Manifest size becomes operationally significant; reopens ADR-0005 §1 exclusive-prefix layout | open |
| [ADR-0044](../../docs/adr/0044-external-artifact-references.md) | OQ-2 | External reference for additional param types — reference value lists and lookup tables | Catalog additions surface concrete demand; standalone regex catalogs require separate ADR (expression-bearing) | open |
| [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md) | OQ-1 | `/resolve-b2` and `/resolve-b3` — separate commands vs. parametrized `/resolve-b <tier> <slug>` | The second post-Wave-3 B2 or B3 session following `post-wave3-session-loop.md`; divergence between B2 vs. B3 grounding becomes empirically visible | open |
| [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md) | OQ-2 | `/sync-agents` extension to skills and playbooks (today: slash-commands only) | The first new contributor learning the ADR-0051 artifacts surfaces drift-detection gap | open |
| [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md) | OQ-G3.1 | Force-push deny entries — `git push --force` / `git push -f` in `.claude/settings.json` deny block | Near-miss force-push or pattern across sessions; bounded by existing CLAUDE.md confirmation gate | open |
| [ADR-0052](../../docs/adr/0052-session-reading-router.md) | OQ-1 | `/start-session <type>` command — consume Clause 2 table to surface per-type reading set at session open | First B3 / B2 session after router landing when cost of manually re-reading router becomes visible | open |
| [ADR-0052](../../docs/adr/0052-session-reading-router.md) | OQ-2 | `session-governance` skill consumption of the router (session-open trigger phrase coverage) | First `session-governance` skill update following the router landing; harness verification step | open |
| [ADR-0052](../../docs/adr/0052-session-reading-router.md) | OQ-3 | Routing axis composition — session type vs. artifact type vs. composed two-axis | Future B3-N entry reveals within-row variance is material (e.g., B3 producing study vs. tooling artifact) | open |
| [ADR-0052](../../docs/adr/0052-session-reading-router.md) | OQ-4 | Wave-S session type — separate router row for Wave-S follow-up sessions | Wave-S full gate reveals new session shape (full gate met 2026-05-25 per ADR-0027 — see decision-log §Wave Gates) | open |
| [ADR-0052](../../docs/adr/0052-session-reading-router.md) | OQ-5 | Attention-cost measurement for the router's cost-savings claim | Forcing function: measurement-decision PR opens by the third post-router session at the latest | open |
| [ADR-0053](../../docs/adr/0053-record-mode-skill.md) | OQ-1 | `make lint-skill-citations` — verify each cited `file:line` in skill references resolves at CI time | First citation-drift incident (record-mode PR ships with stale skill citations) or operator signal that R4-bundled discipline misses drift | open |
| [ADR-0053](../../docs/adr/0053-record-mode-skill.md) | OQ-2 | Track C scope and its relationship to record-mode-conventions skill content | Track C session opens (operator-stated next milestone, deferred) | open |
| [ADR-0053](../../docs/adr/0053-record-mode-skill.md) | OQ-3 | Skill description shape and harness-load reliability (inherits ADR-0051 Clause 3 risk) | Implementation slice's harness verification reveals a phrase that fails to load reliably | open |
| [ADR-0053](../../docs/adr/0053-record-mode-skill.md) | OQ-4 | Whether `engine/internal/runner/doc.go` ships alongside the skill | Implementation slice per ADR-0052 §6.2 row 6 lands under closed Wave-3 P4 runner scaffolding; or follow-up B3-N if new conventions surface | open |
| [ADR-0058](../../docs/adr/0058-record-runner-commit-after-dispatch.md) | OQ-1 | Concurrent runner posture — per-partition fan-out re-opening commit ordering / serialization | Concrete demand for multi-goroutine consumer surfaces (single-goroutine is committed shape today) | open |
| [ADR-0058](../../docs/adr/0058-record-runner-commit-after-dispatch.md) | OQ-2 | Partitions that produce only late-dropped records — periodic offset-advance policy | Concrete signal that broker-retention-bounded offset stall is operationally painful | open |
| [ADR-0058](../../docs/adr/0058-record-runner-commit-after-dispatch.md) | OQ-3 | Commit failure retry / back-off policy — bounded retry vs. β warning-log-and-skip | Operational signal that broker connectivity is flaky | resolved-adr ([ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md)) |
| [ADR-0058](../../docs/adr/0058-record-runner-commit-after-dispatch.md) | OQ-4 | Per-window or per-batch commit-failure metric (`dq_record_commit_failures_total`) | Next emission-slice session re-scopes ADR-0039 inventory (adjacent to ADR-0059 OQ-3 and OQ-6) | open |
| [ADR-0058](../../docs/adr/0058-record-runner-commit-after-dispatch.md) | OQ-5 | Operator-rerun path for record-mode (replay of Kafka offset range) | ADR-0024 §Notes deferral stands; future ADR defines stream-rerun semantics | open |
| [ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md) | OQ-1 | Operator-tunable retry parameters — promote `max_attempts` / `base` from `const` to env-var or runner-config | Production operational signal reveals different optimal shape (longer back-off / fewer attempts) | open |
| [ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md) | OQ-2 | Transient-vs-permanent error classification (substrate-agnostic `RecordConsumer` predicate) | Operational pain from wasted retry budget on permanent errors (e.g., authentication failure) | open |
| [ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md) | OQ-3 | Retry-attempt observability metric (`dq_record_commit_retries_total`) | Next emission-slice session re-scopes ADR-0039 inventory (adjacent to ADR-0058 OQ-4 and ADR-0059 OQ-6) | open |
| [ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md) | OQ-4 | Per-attempt timeout for `consumer.Commit` (broker-hang failure mode) | Operational signal that broker hangs (rather than errors) is a realistic failure mode | open |
| [ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md) | OQ-5 | Total retry budget across a session (circuit-breaker for extended outage) | Production extended-outage incident reveals per-call retry is insufficient | open |
| [ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md) | OQ-6 | Quantitative stall-budget calibration — observed poll-batch processing time vs. retry stall | Emission slice wires commit-RPC histogram (adjacent to ADR-0058 OQ-4 and ADR-0059 OQ-3); sufficient observation window accumulated | open |

**Row count:** 32 labeled OQs across 10 ADRs.
**Resolved-adr today:** 2 — ADR-0042 OQ-1 by
[ADR-0054](../../docs/adr/0054-engine-image-registry-amendment.md);
ADR-0058 OQ-3 by
[ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md).
Net open: 30.

### A.2 — Adjacency cluster note (informational, not in v0 register)

Three OQs converge on the same emission-slice trigger and would
plausibly be consumed by the same future B3-N:

- ADR-0058 OQ-4 (`dq_record_commit_failures_total`)
- ADR-0059 OQ-3 (`dq_record_commit_retries_total`)
- ADR-0059 OQ-6 (commit-RPC histogram for stall-budget
  calibration)

Whether the register exposes adjacency clusters is in
[OQ-3](#open-questions) of this D0.

---

## Appendix B — Unlabeled prose deferrals — NOT in v0

The following ADRs carry prose deferrals in §Notes (or §Consequences)
that are *not* `OQ-N`-labeled. v0 deliberately excludes them. The
operator can see the scope of the deferred-but-unlabeled surface
and decide separately ([OQ-1](#open-questions)).

The table lists examples only — a quantitative count column was
omitted from this appendix because §Notes phrasing draws an
unstable line between three categories that a column would
conflate: (i) genuine deferrals ("reserved until concrete
operational signal"), (ii) already-resolved follow-ups whose
§Notes paragraph is now historical narrative (e.g.,
[ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md)'s
compatibility-window follow-up resolved by
[ADR-0035](../../docs/adr/0035-compatibility-window-duration.md);
[ADR-0042](../../docs/adr/0042-release-engineering-invariants.md)
OQ-1 resolved by
[ADR-0054](../../docs/adr/0054-engine-image-registry-amendment.md)),
and (iii) explicitly closed-off non-applicabilities (e.g.,
[ADR-0030](../../docs/adr/0030-manifest-cryptographic-posture.md)
"quantitative ceilings … do not apply here";
[ADR-0047](../../docs/adr/0047-lint-substrate-access.md)
"mutating endpoints … are deliberately out-of-scope"). Any
quantitative aggregate over this surface needs a per-bullet
re-walk; the v0 register's labeled-OQ count (32) is a clean
denominator the unlabeled surface lacks.

| ADR | Examples (verbatim §Notes phrasing fragments) |
|---|---|
| [ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md) | "compatibility window duration … is a follow-up" (resolved by [ADR-0035](../../docs/adr/0035-compatibility-window-duration.md)) / "cryptographic posture … is also a follow-up" (addressed by [ADR-0030](../../docs/adr/0030-manifest-cryptographic-posture.md)) / "linter-pinning mechanism … is a Wave-3 sub-decision" |
| [ADR-0002](../../docs/adr/0002-run-identity-and-idempotency.md) | window-boundary alignment / sub-second precision / namespace prefixes / `ci-dry-run` enum |
| [ADR-0003](../../docs/adr/0003-result-write-model.md) | dialect choices / local-emulator fidelity / retention parameters |
| [ADR-0004](../../docs/adr/0004-failure-scope.md) | pre-check validations / DSL construct / per-env halt ceiling |
| [ADR-0005](../../docs/adr/0005-manifest-publication-semantics.md) | lifecycle retention / rollback path / cryptographic posture / hash-algorithm migration |
| [ADR-0006](../../docs/adr/0006-alert-routing-contract.md) | channel-type registry / consumer shape / owner-existence enforcement |
| [ADR-0007](../../docs/adr/0007-loader-scheduler-retry-failure-semantics.md) | trigger marker / transient classification / pre-check mechanism / admin abort endpoint |
| [ADR-0008](../../docs/adr/0008-git-host.md) | alternative CI runner / artifact registry / branch-protection specifics |
| [ADR-0009](../../docs/adr/0009-multi-agent-contract.md) | `.codex/` playbooks / permission model / drift-detection CI |
| [ADR-0010](../../docs/adr/0010-substrate-posture.md) | emulator images / sandbox bootstrap / fidelity-gap docs / logs collector / admin-API mock |
| [ADR-0011](../../docs/adr/0011-documentation-language.md) | language-marker CI enforcement / future languages |
| [ADR-0012](../../docs/adr/0012-tag-conventions.md) | tools-lint scope / schema-only release tag / pre-release rules / tag-CI-gate specifics |
| [ADR-0013](../../docs/adr/0013-wave3-sequencing.md) | `/critique-adr` skill / Phase 4 sub-phases / tools-lint scope / pre-release rules / ADR numbering convention |
| [ADR-0014](../../docs/adr/0014-trigger-handler-contract.md) | `/manifestz` endpoint / error-code taxonomy / v2 path-bump triggers / self-link content format / authentication ADR / rate-limit ADR / gRPC variant |
| [ADR-0015](../../docs/adr/0015-codeowners.md) | GitHub-org identifier / sync-agents CI gate / publisher/loader defense-in-depth / per-entity refinement / governance-lane CODEOWNERS |
| [ADR-0016](../../docs/adr/0016-workspace-tooling.md) | module split / go-version pin / per-tool `go.mod` |
| [ADR-0017](../../docs/adr/0017-substrate-posture-amendment.md) | emulator choice / future substrate evaluation |
| [ADR-0018](../../docs/adr/0018-environment-configuration-model.md) | per-bucket coupling / future env-spanning CLI / deployment-bucket variability |
| [ADR-0019](../../docs/adr/0019-infrastructure-tooling.md) | Kustomize alternative / cluster-side validation / kubectl form / base manifest neutrality |
| [ADR-0020](../../docs/adr/0020-wave-s-launch.md) | owners-capability removal / executions-table shape / record-mode alert dedup / Wave-S shape composition |
| [ADR-0021](../../docs/adr/0021-mode-as-primitive.md) | diagnostic wording / multiple rules per file / loader metric / owners schema cadence / unified-vs-parallel runner |
| [ADR-0022](../../docs/adr/0022-kind-catalog.md) | catalog format YAML/JSON / non-Go handlers / `dq-catalog` CLI / per-kind docs / `record.schema_conformance` placement |
| [ADR-0023](../../docs/adr/0023-sources-schema.md) | Kafka watermark/offset semantics / multi-broker authentication / per-env source defaults / `partition_column` validation |
| [ADR-0024](../../docs/adr/0024-window-semantics.md) | partition-combination policy / window alignment ergonomics / `lateness_tolerance: 0s` strictness / operator-rerun / executions partition column / future substrate windowing |
| [ADR-0025](../../docs/adr/0025-aggregation-and-runner-shape.md) | OTel labelling / panic recovery / backpressure / set-mode backfill / factor-list extension / ADR-numbering shift |
| [ADR-0026](../../docs/adr/0026-failure-scope-aggregated.md) | catalog defaults agreement / privacy bounds (B1-6) / sliding aggregation / pluggable aggregation / retry mechanics / partial handler failures / `degraded` routing |
| [ADR-0027](../../docs/adr/0027-record-mode-cost-guardrails.md) | ceiling values / sliding observation period / backoff factor / unified per-entity budget / lint pre-flight / broker-side enforcement / dead-letter routing / cost-dimension extensibility |
| [ADR-0028](../../docs/adr/0028-kafka-substrate-row.md) | image choice / contributor onboarding doc note / in-cluster Kafka access / future substrate amendment |
| [ADR-0029](../../docs/adr/0029-bigquery-cost-ceilings.md) | per-env defaults tuning / sentinel-collision growth / runbook annotation legend |
| [ADR-0030](../../docs/adr/0030-manifest-cryptographic-posture.md) | signature algorithm + key-storage at reopen / audit-log retention (note: quantitative ceilings explicitly do **not** apply here) |
| [ADR-0031](../../docs/adr/0031-evidence-retention-parameters.md) | retention spreads tuning / `tools/retention` binary / `SampleContentAllowlist` lint / audit-log retention |
| [ADR-0033](../../docs/adr/0033-scheduler-catchup-behavior.md) | grammar extensions / external-monitor cost / engine-side scheduler |
| [ADR-0034](../../docs/adr/0034-local-testing-strategy.md) | benchmark tier (7th) / fuzz tier (8th) / sandbox first-consumer slice |
| [ADR-0035](../../docs/adr/0035-compatibility-window-duration.md) | 90-day floor amendment / auto-generated compatibility state / per-deployment overrides |
| [ADR-0039](../../docs/adr/0039-dashboard-contract.md) | pre-aggregated `dq_entity_rollup` / per-metric cardinality ceiling / `evidence_summary` field inventory |
| [ADR-0041](../../docs/adr/0041-stream-reporting-continuity.md) | cross-mode `dq_entity_health_score` |
| [ADR-0045](../../docs/adr/0045-baseline-dashboard-substrate.md) | templated variables / cost-panel field documentation (cites ADR-0039 OQ-3) / specialized dashboards |
| [ADR-0046](../../docs/adr/0046-onboarding-channel-override.md) | channel-naming convention / multi-purpose override / per-category map |
| [ADR-0047](../../docs/adr/0047-lint-substrate-access.md) | `DQ_LINT_*` env-var prefix / adapter packaging / lint-reachability 7th tier (note: mutating endpoints are explicitly closed-off, **not** deferred) |
| [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md) | `docs(critique):` scope split / two-round cap as future-amendable |
| [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md) | release-cadence ADR / 2026-08-24 date shift / pre-1.0 minor-vs-patch boundary |
| [ADR-0054](../../docs/adr/0054-engine-image-registry-amendment.md) | CI secret names / base-tag posture (cites ADR-0042 OQ-2) |
| [ADR-0055](../../docs/adr/0055-metric-emission-slice-scope.md) | emission-slice operational details / Condition 2 borderline carry-forward |
| [ADR-0056](../../docs/adr/0056-panel-5-lighting-slice.md) | weak/strong reading carry-forward / no-ADR-0033-reopening / Condition 1+3 D0 carry-forward |
| [ADR-0057](../../docs/adr/0057-single-user-codeowners-amendment.md) | branch-protection precondition audit / deny-block independence carry-forward / second-order trade-offs (a)–(e) |

Two qualitative observations the operator can use to decide
[OQ-1](#open-questions):

- (m) Labeled OQs cluster in post-Wave-3 ADRs (ADR-0040 / 0042 /
  0043 / 0044 onward); pre-Wave-3 ADRs (0001–0035) generally
  use unlabeled prose. The labeling discipline emerged
  organically; it was never an AC.
- (n) The §Notes prose carries *signal about intent*: a follow-
  up named with "Reserved as a B2 follow-up" reads stronger
  than a follow-up mentioned in a parenthetical as
  "operational detail". A future labeling rule could promote
  only the stronger forms; the weaker forms stay implicit.

These observations are the seed of OQ-1 (unlabeled-prose
deferral scope) and OQ-2 (labeling convention going forward).
Both deferred until the v0 register has been used in anger.
