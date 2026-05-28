<!-- path: studies/decisions/2026-05-27-b3-evolutionary-launch.md -->

# B3 — Evolutionary Launch Study

## Metadata

- **Wave reference:** B3 (evolutionary lane; post-Wave-3, post-Wave-S
  launch).
- **Status:** resolved-study (B3, session 1; one critique round; round 1 cleared with no blocking findings).
- **Last updated:** 2026-05-27.
- **Upstream resolved:** Waves 1, 2, and 3 — all gates met
  (2026-05-21, 2026-05-21, 2026-05-23 respectively). Wave-S launched
  on 2026-05-23 with full B0-S triplet resolved by 2026-05-24. The
  four ADRs that B3 must conform to (per locked premise P2) —
  [ADR-0020](../../docs/adr/0020-wave-s-launch.md) (Wave-S launch and
  substrate-as-deployment-detail principle),
  [ADR-0021](../../docs/adr/0021-mode-as-primitive.md) (mode as
  primitive),
  [ADR-0022](../../docs/adr/0022-kind-catalog.md) (kind catalog and
  catalog-evolution rules), and
  [ADR-0023](../../docs/adr/0023-sources-schema.md) (sources schema
  and substrate binding per mode) — are the constraint envelope
  this study operates within.
- **Downstream open:** none enumerated yet. B3-N items will be
  registered when concrete demand surfaces, matching the demand-
  driven pacing committed for post-Wave-3 follow-ups in
  [`06-decision-log.md`](../foundation/06-decision-log.md) §Recommended
  Next Sequence and the B2 "as implementation reveals concrete needs"
  semantics in §Prioritization Model.
- **Promotion target:** `docs/adr/0049-b3-evolutionary-launch.md`
  (provisional; the slot may shift if intervening ADRs land before
  promotion).
- **Locked premises** (operator-declared, not litigated in this
  study):
  - **P1** — *Evolutionary work expands existing capabilities; does
    not rewrite.* B3 entries extend a contract already committed by
    a promoted ADR; reshaping a committed contract is amendment or
    a new wave, not B3.
  - **P2** — *B3 conforms to ADR-0020 / 0021 / 0022 / 0023
    constraints.* Substrate-as-deployment-detail (0020), mode as
    primitive (0021), kind catalog and its evolution rules (0022),
    and mode-to-source binding (0023) are the constraint envelope.
  - **P3** — *B3-N items are demand-driven.* Rows are born when
    specific demand arises, matching the B1/B2 pacing pattern
    (work-pacing is demand-driven; rows are enumerated when
    discovered and triaged on demand, never pre-planned by
    schedule).
  - **P4** — *Restricted to three families.* Kind family extensions,
    capability mode extensions, and tooling extensions. No fourth
    family is admitted without re-opening this launch.
- **Loop discipline:** same protocol as Waves 1 / 2 / 3 / S — study
  → `/critique` (≥1 round) → operator acceptance → promotion to ADR.
  Per-B3-N items follow the same shape that landed B0-1 … B0-7 and
  B0-S1 … B0-S7. See
  [`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  and
  [`.claude/playbooks/acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md).

---

## Context

The platform has reached a steady state where every architectural
contract — set-oriented and record-oriented — is either committed by
a promoted ADR or scheduled to be promoted under a launched wave.
Waves 1, 2, and 3 closed the set-oriented capability through Wave 3
on 2026-05-23. Wave-S launched the record-oriented capability on the
same day; the foundational triplet B0-S1 / B0-S2 / B0-S3 reached
`resolved-adr` by 2026-05-24, satisfying the partial Wave-S gate for
record-mode code shipping.

From this steady state forward, the work that arrives is no longer
about *first commits* to a contract. It is about **expanding** a
contract that already exists. A new kind name to register under
ADR-0022's catalog. A new lint cross-check that refines an existing
mode-prefix enforcement under ADR-0021. A new manifest-publisher
subcommand that extends contract coverage without changing the
contract shape. These items are real, predictable, and continuous —
but they fit nowhere on the current decision-log map.

Three governance gaps motivate this launch:

1. **B2 conflates two distinct cadences.** B2 was defined as "later
   decisions … can be resolved as implementation reveals concrete
   needs" (`06-decision-log.md` §Prioritization Model). Post-Wave-3
   evolutionary work *does* match the demand-driven pacing, but it
   does **not** match the *implementation-phase* framing — it
   surfaces *after* implementation has shipped, not as part of it.
   Folding evolution into B2 blurs "we are still building v1" with
   "we are extending v1+ in production".

2. **ADR-0022's catalog evolution is a PR-level mechanism, not a
   decision-history layer.** ADR-0022 §evolution rules (lines
   161–200) authorize additive kind additions via PR under
   CODEOWNERS dual review — no ADR ceremony required for additive
   change. That mechanism is correct at the artifact level, but it
   produces *no* decision-history record of the *intent* that
   authorized the new kind. When the intent reshapes contributor
   expectations (a new kind whose params are non-trivial, a kind
   whose source-mode binds it to a substrate decision), the
   platform needs a record above the catalog PR.

3. **Without an explicit out-of-scope list, evolutionary work drifts
   into substrate / performance / API-evolution territory.** Each
   of those families is *new wave* work, not B3. ADR-0020 fixes
   substrate as a deployment detail downstream of mode; performance
   tuning does not extend capability (it does not satisfy P1);
   API-evolution rewrites the contract (it violates P1). If B3 has
   no explicit out-of-scope clause, every "while we're at it" item
   accumulates here and dilutes the eligibility filter.

This study launches B3 as a parallel lane sitting alongside Wave 2,
Wave 3, and Wave-S in the decision log. The B-tier name signals
demand-driven cadence (matching B1/B2's pacing semantics); the
wave-style placement signals categorical peer (B3 is a structural
peer of Wave 2 / Wave 3 / Wave-S, not a fourth priority tier above
them). The study fixes an eligibility criterion that distinguishes
B3 from B2 / amendment / rejected, and enumerates the three
in-scope families and the three out-of-scope candidates explicitly.
It does not open any B3-N item. Per P3, each B3-N item arrives only
when concrete demand surfaces.

---

## Decision Drivers

- **DD-B3.1** — **No silent additions.** Evolutionary work must not
  bypass decision history. ADR-0022 §evolution permits additive
  catalog PRs without ADR ceremony; that is correct for the
  artifact, but the *intent* behind any non-trivial extension
  benefits from a study trail. B3 captures that intent layer.

- **DD-B3.2** — **Unambiguous eligibility.** A contributor proposing
  an extension must be able to tell, from this launch study alone,
  whether the proposal is B3, B2, an amendment, or out-of-scope.
  Operator arbitration on a per-case basis is a failure mode.

- **DD-B3.3** — **Narrow scope.** Substrate, performance, and
  API-evolution are each large enough to constitute their own
  wave. Folding any of them into B3 makes the eligibility filter
  useless: every "extension" proposal becomes a triage exercise
  rather than a check against a list.

- **DD-B3.4** — **Demand-driven pacing.** Per P3 and the B2
  precedent, B3-N items are not pre-enumerated. Evolution that has
  no concrete demand is speculation; pre-listing items would
  invite work-against-the-list pressure that violates the
  demand-driven posture (P3).

- **DD-B3.5** — **Constraint-envelope conformance.** Per P2, every
  B3 entry must conform to ADR-0020 (substrate downstream of mode),
  ADR-0021 (mode as primitive), ADR-0022 (kind catalog evolution
  rules), and ADR-0023 (sources schema + mode-to-substrate
  binding). A proposed B3 entry that conflicts with any of these
  is not B3 — it is either amendment (modifying the ADR's
  decision) or a new wave (re-opening the framing).

- **DD-B3.6** — **Forward-only artifact.** Per R8, the published
  ADR-0049 must read cold to a reviewer who has never opened
  `studies/`; the eligibility criterion and out-of-scope list are
  the load-bearing content that must survive promotion intact.

---

## Considered Options

### Option A — Single launch ADR with explicit eligibility criterion

One launch study (this document), one promoted ADR (provisional
ADR-0049). Three families enumerated as in-scope (kind, capability
mode, tooling). Three candidates enumerated as out-of-scope
(substrate, performance, API evolution). Eligibility branches fixed:
B3 / B2 / amendment / rejected. B3-N items are registered
demand-driven against this single contract; each individual B3-N
item runs the standard study → critique → ADR loop.

### Option B — Per-family sub-launches (three launch ADRs)

One launch ADR per family: a kind-family evolutionary launch ADR, a
capability-mode evolutionary launch ADR, a tooling evolutionary
launch ADR. Each family gets its own gate, its own eligibility
prose, its own out-of-scope list. B3 in the decision log becomes
three parallel tables (B3-K, B3-M, B3-T) rather than one.

### Option C — No-launch (items just appear)

Evolutionary items are added directly to the decision log when
discovered, either under B2 (extending the implementation-phase
table) or under a new ad-hoc label. No governing launch study, no
single eligibility filter; contributors arbitrate case-by-case
against the ADRs in scope. The catalog PR mechanism (ADR-0022
§evolution) carries the artifact layer; the decision-history layer
above it is left implicit.

---

## Recommendation

**Pick Option A — single launch ADR with explicit eligibility
criterion.**

Rationale, tied directly to drivers:

- **No-silent-additions (DD-B3.1)** — Option A creates a single
  document a contributor must register against before any B3-N
  item arrives. Option C produces silent additions by construction
  (no governing artifact). Option B produces three governing
  artifacts that share the same eligibility question, which
  triples drift surface for no isolation gain.
- **Unambiguous eligibility (DD-B3.2)** — Option A fixes a single
  four-branch criterion (§(a) below). Option C requires
  case-by-case arbitration. Option B fragments the criterion across
  three documents that will inevitably drift in phrasing.
- **Narrow scope (DD-B3.3)** — Option A enumerates the
  out-of-scope list once (§(b) below). Option C has no
  out-of-scope mechanism. Option B forces each family-launch to
  re-state the substrate/performance/API exclusions, multiplying
  the surface where the exclusions could erode.
- **Demand-driven pacing (DD-B3.4)** — Option A makes pacing a
  property of the launch (§(c) below: B3 stays open indefinitely,
  pacing is demand-driven). Options B and C do not change this,
  but Option B's three tables create three places where the
  "pre-enumeration" temptation could resurface.
- **Constraint-envelope conformance (DD-B3.5)** — Option A makes
  conformance to ADR-0020/0021/0022/0023 a single named criterion
  branch. Option C has no central place for the conformance
  check; it would re-emerge in every B3-N study individually.
- **Forward-only artifact (DD-B3.6)** — Option A produces one ADR
  that survives promotion cleanly. Option B produces three; the
  cross-reference burden between them at promotion time is
  non-trivial.

**One-line decision summary table:**

| Decision | Outcome |
|---|---|
| Launch posture | Single launch ADR (Option A) |
| In-scope families | Kind family, capability mode, tooling (P4) |
| Out-of-scope candidates | Substrates, performance, API evolution |
| Eligibility branches | B3 / B2 / amendment / rejected |
| Pacing | Demand-driven (P3); rows born on demand |
| Closure | B3 stays open indefinitely |
| Constraint envelope | ADR-0020 / 0021 / 0022 / 0023 (P2) |

### (a) Eligibility criterion

A proposed capability extension qualifies as:

- **B3** — iff **all four** conditions hold: (i) it expands existing
  capability without rewriting it (P1); (ii) it falls inside one
  of the three families enumerated in (b) below as in-scope (P4);
  (iii) it conforms to ADR-0020, 0021, 0022, and 0023 (P2); and
  (iv) it crosses the additive-maintenance threshold — proposing
  capability extension beyond the incremental evolution paths
  authorized by relevant ADRs (e.g., ADR-0022 §evolution for kind
  families; analogous evolution clauses for capability modes and
  tooling). When the proposed change is materially novel rather
  than incremental, B3 path applies. Examples: a new kind whose
  params reshape contributor expectations; a lint cross-check
  that refines mode-prefix enforcement under a non-trivial rule;
  a tooling subcommand whose contract coverage adds a new failure
  mode the operator must understand.

- **B2** — iff the proposal is an implementation-phase decision
  surfacing *inside* an existing wave's gate — a v1 capability
  deferred to "as implementation reveals concrete needs"
  (`06-decision-log.md` §Prioritization Model). The line is
  temporal and contractual: B2 is *pre-shipping* against an
  in-flight wave; B3 is *post-shipping* against a closed wave. A
  Wave-S item that surfaces while Wave-S's full gate is still open
  is B2-S, not B3.

- **Amendment** — iff the proposal *modifies* (not extends) the
  decided shape of an existing ADR. Amendments live under the
  originating ADR's supersession chain (or as a follow-up ADR
  superseding the originating one), never under B3. The test is:
  "does this proposal change what the ADR decided, or does it
  build on top of what the ADR decided?" If the former,
  amendment; if the latter, B3 (given the other three conditions).

- **Rejected** — iff the proposal falls outside the three
  in-scope families *and* outside any active wave's gate. Surfaced
  explicitly so the decision log records the rejection rather than
  the silence. A rejected outcome is a one-line entry in the
  decision log linking to a short rationale note; no full study
  is required.

### (b) What B3 is NOT

The launch ADR explicitly closes these candidates as out-of-scope:

- **Candidate C — substrates.** BigQuery, Kafka, alternative
  streaming layers, alternative warehouses, alternative
  pub/sub vehicles. **Rationale:** ADR-0020 fixes substrate as a
  deployment detail downstream of mode — "Substrate (BigQuery,
  Kafka, Pub/Sub, etc.) is a deployment detail downstream of
  mode" (ADR-0020 §Locked architectural premises, lines 56–65).
  Substrate evolution either lives inside an existing mode (and
  is then an amendment for that mode's substrate ADR) or
  constitutes a *new mode* (and is a new Wave-S-style launch).
  B3 has no path to handle substrate change because handling
  substrate change at the B3 layer would invert the ADR-0020
  primitive — mode would become a function of substrate rather
  than the other way around. Locked.

- **Candidate E — performance work.** Engine throughput tuning,
  BigQuery query-cost optimization, scheduler concurrency
  budgeting, runner-side parallelism adjustments. **Rationale:**
  Performance is operational, not capability-extending. A
  throughput improvement does not enable a contract the platform
  did not already support; it adjusts the *rate* at which an
  already-supported contract executes. This fails P1 (B3 must
  *expand* capability). Performance work belongs to B2
  (implementation-phase operational decisions, e.g., B1-2-style
  cost ceilings), dedicated performance studies, or runbook
  changes — none of which carry a B3 label.

- **Candidate F — API evolution.** Rule-schema versioning beyond
  the v1→v2 path opened by ADR-0021; CLI / manifest API
  reshaping; loader contract changes; engine-side public API
  surface changes. **Rationale:** API evolution *rewrites* the
  contract it touches; it does not *extend* a contract that
  remains intact. This fails P1 directly. API evolution
  requires its own launch — a parallel-scope wave on the model
  of Wave-S, or an explicit supersession of the governing ADR
  via amendment — not a B3 entry. The schema v1→v2 cadence
  committed by ADR-0021 §Notes is the only API-evolution
  mechanism currently sanctioned; future cadence changes are
  themselves out-of-scope for B3.

### (c) When B3 closes

**B3 stays open indefinitely.** Unlike Waves 1, 2, and 3 (which
had explicit gate criteria and definite closure events) and
Wave-S (which has a partial-gate and a full-gate criterion in
ADR-0020), B3 has no gate. Evolution is continuous by definition
(P1: B3 extends what exists), and the platform extends itself
across its operating life.

B3 closes only if the platform itself is sunset, replaced, or
re-launched under a new framing that supersedes ADR-0020 /
0021 / 0022 / 0023. In that event, B3 closes alongside the
ADRs it conforms to, and a new evolutionary lane is launched
inside the successor framing.

Each individual B3-N item closes via the standard loop: study →
critique → ADR. The B3-launch row itself moves to `resolved-adr`
once this study is promoted to ADR-0049, and the section in
`06-decision-log.md` remains live for new B3-N entries
indefinitely.

---

## Consequences

### 6.1 — Cross-cutting consequences

- **C-B3.1** — A contributor onboarding to evolutionary work has
  exactly one document to read (the launch study or its promoted
  ADR) and one table row to scan (`B3-launch` in
  `06-decision-log.md`) before proposing an extension. The
  governance surface for evolution is fixed at one document,
  one row, one decision tree.

- **C-B3.2** — Catalog PRs landing under ADR-0022 §evolution
  (additive kind additions via CODEOWNERS dual review, no ADR
  ceremony) are unaffected. B3 governs the *decision-history*
  layer above the catalog, not the catalog PR cadence itself. A
  routine additive kind addition stays a catalog PR; a kind
  addition carrying intent that benefits from study trail
  becomes a B3-N item.

- **C-B3.3** — The decision-log gains a new triage vocabulary —
  **B3 / B2 / amendment / rejected** — as the official outcome
  set for proposed extensions. The `rejected` outcome
  previously had no place in the decision log; it now has one,
  via a one-line entry recording the rejection rather than the
  silence.

- **C-B3.4** — Substrate, performance, and API-evolution
  candidates are closed off explicitly with rationale that cites
  ADR-0020 (for substrate), P1 (for performance), and P1 (for
  API). Future ambiguity gets adjudicated against the rationale
  recorded here, not against operator memory.

- **C-B3.5** — Per R8, the future ADR-0049 will be rewritten from
  this study, not linked back to it. This study remains in
  `studies/decisions/` as the reasoning artefact; ADR-0049 will
  read cold to a reviewer who has never opened `studies/`. The
  load-bearing artefacts in the promoted ADR are §(a)
  eligibility, §(b) out-of-scope, and §(c) closure.

### 6.2 — Per-family consequences

#### Kind family extensions

- **Decides:** B3 captures the *intent* behind a new kind whose
  shape reaches beyond a routine additive PR — e.g., a kind
  whose params reshape contributor expectations, or whose
  source-mode binds it to a substrate that ADR-0023 has
  decided. A routine `set.*` or `record.*` kind whose params
  are trivial and whose source-mode is already-bound stays a
  catalog PR; a kind whose intent benefits from study trail
  becomes a B3-N entry.
- **Depends on:** ADR-0022 §evolution rules (additive PR
  threshold vs. ADR-required threshold, lines 161–200);
  ADR-0021 mode-as-primitive grammar (`^(set|record)\..+$`).
- **Downstream:** future B3-N items as new kinds with
  non-trivial intent are proposed.

#### Capability mode extensions

- **Decides:** B3 captures extensions to mode-derived behavior
  *within* the existing set/record duo — a new lint cross-check
  that refines mode-prefix enforcement, a new derived-capability
  signal exposed to the loader or scheduler, a refinement to
  how capability is dispatched at runtime. Adding a **new
  mode** (a third top-level mode alongside `set` and `record`)
  is *out* of B3; it would constitute a Wave-S-scale launch
  because it would re-open ADR-0021's mode-as-primitive
  framing.
- **Depends on:** ADR-0021 (mode field, lint cross-checks,
  capability-derived-from-mode rule); ADR-0023 (mode-to-source
  binding, since a mode extension may interact with source
  binding).
- **Downstream:** future B3-N items as derived-capability
  surfaces evolve.

#### Tooling extensions

- **Decides:** B3 captures additions to `tools/lint/`, the
  manifest publisher, the dry-run runner, the engine
  dispatcher, and adjacent tooling that **extend** contract
  coverage without changing the contract shape. New lint
  cross-checks beyond the eight committed by ADR-0021 /
  ADR-0022 / ADR-0023 are the canonical example. Reshaping
  loader contracts is amendment, not B3 (it violates P1 —
  loader contract reshape is rewrite).
- **Depends on:** ADR-0021 (lint cross-check inventory #1–#4,
  loader v2 schema dispatch); ADR-0022 (lint cross-checks #5,
  #6 — catalog membership, per-kind params); ADR-0023 (lint
  cross-checks #7, #8 — `source.type` vs. mode, `source.type`
  vs. catalog `source_mode`).
- **Downstream:** future B3-N items as tooling gaps surface
  during evolutionary work.

---

## Open Questions

- **OQ-B3.1** — Should B3 entries be tag-prefixed by family
  (e.g., `B3-K-1` for kind, `B3-M-1` for mode, `B3-T-1` for
  tooling) or kept flat (`B3-1`, `B3-2`, …)? *Defer to first
  B3-N registration.* The first B3-N item's family becomes the
  precedent. The launch ADR does not commit either shape.

- **OQ-B3.2** — Does the **rejected** outcome require its own
  entry in `06-decision-log.md` §Status Vocabulary, or is it
  implicit in the eligibility branches? *Defer until the first
  rejected case lands.* The first rejection will reveal
  whether the vocabulary needs explicit accommodation.

- **OQ-B3.3** — Should ADR-0049 enumerate examples of past
  capability additions (already-shipped kind additions in the
  catalog, already-shipped lint cross-checks) retroactively as
  B3 entries to seed the lane, or is B3 strictly forward-
  looking? *Out of scope for this study; default is forward-
  looking per P3 demand-driven posture.* Retroactive enumeration
  would contradict P3 (past items had no live demand at the
  moment of this launch). Revisit only if a concrete retroactive
  need emerges.

- **OQ-B3.4** — When a proposed extension is borderline between
  the "additive catalog PR" path (ADR-0022 §evolution) and the
  "B3-N entry" path, who arbitrates? *Default: the CODEOWNERS
  reviewer assigned to the PR.* The reviewer can hold the PR
  until a B3-N study lands. Confirm at first borderline case.

- **OQ-B3.5** — Does B3 need its own playbook under
  `.claude/playbooks/`, parallel to `wave-1-session-loop.md`?
  *Defer to first B3-N session.* The B3-N study workflow is
  expected to be the same as the B0-S study workflow, so a new
  playbook may be unnecessary. The first B3-N session will
  reveal whether deltas exist.

---

## Promotion target

**Target:** `docs/adr/0049-b3-evolutionary-launch.md`
*(provisional)*.

This launching study promotes to **ADR-0049** once at least one
round of `/critique` has been accepted by the operator and any
blocking findings are addressed. The provisional slot may shift
if intervening ADRs land before promotion; the slug
`b3-evolutionary-launch` is stable regardless of slot.

Per R8, the future ADR-0049 will be rewritten from this study,
not linked back to it. This study remains in `studies/decisions/`
as the reasoning artefact; ADR-0049 will read cold to a reviewer
who has never opened `studies/`. The load-bearing content the
ADR must carry intact is:

1. §(a) eligibility criterion — the four-branch decision tree.
2. §(b) what B3 is NOT — the three out-of-scope candidates with
   their cited rationale (ADR-0020 substrate-as-deployment-
   detail for Candidate C; P1 for Candidates E and F).
3. §(c) closure semantics — B3 stays open indefinitely.

Each B3-N item that arrives after promotion opens its own study
under the standard study → critique → ADR loop and promotes to
its own ADR slot on its own schedule, conforming to the
constraint envelope (ADR-0020 / 0021 / 0022 / 0023) per P2.
