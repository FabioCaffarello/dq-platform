<!-- path: docs/adr/0049-b3-evolutionary-launch.md -->

# ADR-0049 — B3 Evolutionary Launch

- **Status:** accepted
- **Date:** 2026-05-29

---

## Context

Waves 1, 2, and 3 closed the set-oriented capability of the platform
on 2026-05-21 / 2026-05-21 / 2026-05-23, and Wave-S launched the
record-oriented capability on 2026-05-23 with its foundational
triplet ([ADR-0021](./0021-mode-as-primitive.md),
[ADR-0022](./0022-kind-catalog.md),
[ADR-0023](./0023-sources-schema.md)) reaching `resolved-adr` by
2026-05-24. From this steady state forward, the work that arrives is
no longer about *first commits* to a contract. It is about
**expanding** contracts that already exist: a new kind name registered
under [ADR-0022](./0022-kind-catalog.md)'s catalog, a new lint
cross-check refining mode-prefix enforcement under
[ADR-0021](./0021-mode-as-primitive.md), a manifest-publisher
subcommand that extends contract coverage without changing the
contract shape.

Three governance gaps motivate a dedicated lane for that work:

1. **B2 conflates two distinct cadences.** B2 in
   [`studies/foundation/06-decision-log.md`](../../studies/foundation/06-decision-log.md)
   §Prioritization Model is defined as "later decisions … can be
   resolved as implementation reveals concrete needs". Post-Wave-3
   evolutionary work matches the demand-driven pacing but does **not**
   match the *implementation-phase* framing — it surfaces *after*
   implementation has shipped, not as part of it. Folding evolution
   into B2 blurs "we are still building v1" with "we are extending
   v1+ in production".

2. **[ADR-0022](./0022-kind-catalog.md) §evolution authorizes
   catalog additions at the PR layer, not the decision-history
   layer.** Additive kind additions ship via CODEOWNERS dual review
   with no ADR ceremony required. That mechanism is correct for the
   artifact, but it produces no decision-history record of the
   *intent* that authorized the new kind. When the intent reshapes
   contributor expectations (a kind whose params are non-trivial, a
   kind whose source-mode binds it to a substrate decision), the
   platform needs a record above the catalog PR.

3. **Without an explicit out-of-scope list, evolutionary work drifts
   into substrate / performance / API-evolution territory.** Each of
   those is *new wave* work, not evolutionary work.
   [ADR-0020](./0020-wave-s-launch.md) fixes substrate as a deployment
   detail downstream of mode; performance tuning does not extend
   capability; API-evolution rewrites the contract. Without an
   explicit exclusion clause, every "while we're at it" item
   accumulates here and dilutes any eligibility filter.

The principles bearing on this decision are **P1** (rules must remain
declarative — extensions never introduce escape hatches), **P3**
(ownership is explicit — evolutionary entries register against named
owners and named families), **P5** (evolution must be contract-driven —
the eligibility filter and out-of-scope list are themselves a
published contract), and **P6** (borrow patterns, not baggage —
evolutionary work expands the platform on its own terms, not by
importing patterns whole). **R8** in
[`CLAUDE.md`](../../CLAUDE.md) §3 is load-bearing for the artifact
shape: this ADR is rewritten for the new audience and reads cold to a
reviewer who never opens `studies/`.

---

## Decision

### Launch posture

A single launch ADR — this document — opens **B3** as an evolutionary
lane. B3 sits in
[`studies/foundation/06-decision-log.md`](../../studies/foundation/06-decision-log.md)
as a structural peer of Wave 2, Wave 3, and Wave-S, not as a fourth
priority tier above B0/B1/B2. The B-tier name signals demand-driven
cadence (matching B1/B2 pacing semantics); the wave-style placement
signals categorical peer status. Per-B3-N items follow the same loop
that landed every prior decision: study → `/critique` (≥1 round) →
operator acceptance → promotion to ADR, as committed by
[`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
and
[`.claude/playbooks/acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md).

### Premises

B3 conforms to four non-litigated premises:

- **P-B3.1 — Evolutionary work expands existing capabilities; does
  not rewrite.** B3 entries extend a contract already committed by a
  promoted ADR. Reshaping a committed contract is *amendment* or a
  *new wave*, not B3.
- **P-B3.2 — B3 conforms to the constraint envelope.**
  [ADR-0020](./0020-wave-s-launch.md) (substrate as deployment detail
  downstream of mode), [ADR-0021](./0021-mode-as-primitive.md) (mode
  as primitive), [ADR-0022](./0022-kind-catalog.md) (kind catalog
  and its evolution rules), and
  [ADR-0023](./0023-sources-schema.md) (sources schema and
  mode-to-substrate binding) are the envelope. A proposed B3 entry
  that conflicts with any of these is not B3 — it is either
  amendment (modifying the ADR's decision) or a new wave (re-opening
  the framing).
- **P-B3.3 — Demand-driven pacing.** B3-N rows are born when concrete
  demand arises. Pre-listing items invites work-against-the-list
  pressure that violates the demand-driven posture.
- **P-B3.4 — Restricted to three families.** Kind family extensions,
  capability mode extensions, and tooling extensions. No fourth
  family is admitted without re-opening this launch.

### (a) Eligibility criterion

A proposed capability extension qualifies as:

- **B3** — iff **all four** conditions hold:
  1. it expands existing capability without rewriting it (P-B3.1);
  2. it falls inside one of the three in-scope families enumerated
     in §(b) (P-B3.4);
  3. it conforms to [ADR-0020](./0020-wave-s-launch.md),
     [ADR-0021](./0021-mode-as-primitive.md),
     [ADR-0022](./0022-kind-catalog.md), and
     [ADR-0023](./0023-sources-schema.md) (P-B3.2); and
  4. it crosses the additive-maintenance threshold — proposing
     capability extension beyond the incremental evolution paths
     authorized by relevant ADRs (e.g.,
     [ADR-0022](./0022-kind-catalog.md) §evolution for kind
     families; analogous evolution clauses for capability modes and
     tooling). When the change is materially novel rather than
     incremental, the B3 path applies. Canonical examples: a new
     kind whose params reshape contributor expectations; a lint
     cross-check that refines mode-prefix enforcement under a
     non-trivial rule; a tooling subcommand whose contract coverage
     adds a new failure mode the operator must understand.

- **B2** — iff the proposal is an implementation-phase decision
  surfacing *inside* an existing wave's gate — a v1 capability
  deferred to "as implementation reveals concrete needs". The line
  is temporal and contractual: B2 is *pre-shipping* against an
  in-flight wave; B3 is *post-shipping* against a closed wave. A
  Wave-S item that surfaces while Wave-S's full gate is still open
  is B2-S, not B3.

- **Amendment** — iff the proposal *modifies* (not extends) the
  decided shape of an existing ADR. Amendments live under the
  originating ADR's supersession chain (or as a follow-up ADR
  superseding the originating one), never under B3. The test is:
  "does this proposal change what the ADR decided, or does it build
  on top of what the ADR decided?" If the former, amendment; if the
  latter, B3 (given the other three conditions).

- **Rejected** — iff the proposal falls outside the three in-scope
  families *and* outside any active wave's gate. Surfaced
  explicitly so the decision log records the rejection rather than
  the silence. A rejected outcome is a one-line entry in the
  decision log linking to a short rationale note; no full study is
  required.

### (b) What B3 is NOT — out-of-scope candidates

The following candidates are closed off at launch:

- **Out-of-scope C — substrates.** BigQuery, Kafka, alternative
  streaming layers, alternative warehouses, alternative pub/sub
  vehicles. **Rationale:** [ADR-0020](./0020-wave-s-launch.md)
  fixes substrate as a deployment detail downstream of mode.
  Substrate evolution either lives inside an existing mode (and is
  then an amendment for that mode's substrate ADR) or constitutes a
  *new mode* (and is a new Wave-S-style launch). B3 has no path to
  handle substrate change because doing so at the B3 layer would
  invert the [ADR-0020](./0020-wave-s-launch.md) primitive — mode
  would become a function of substrate rather than the other way
  around.

- **Out-of-scope E — performance work.** Engine throughput tuning,
  BigQuery query-cost optimization, scheduler concurrency budgeting,
  runner-side parallelism adjustments. **Rationale:** Performance
  is operational, not capability-extending. A throughput improvement
  does not enable a contract the platform did not already support;
  it adjusts the *rate* at which an already-supported contract
  executes. This fails P-B3.1 (B3 must *expand* capability).
  Performance work belongs to B2 (implementation-phase operational
  decisions), dedicated performance studies, or runbook changes —
  none of which carry a B3 label.

- **Out-of-scope F — API evolution.** Rule-schema versioning beyond
  the v1→v2 path opened by [ADR-0021](./0021-mode-as-primitive.md);
  CLI / manifest API reshaping; loader contract changes; engine-side
  public API surface changes. **Rationale:** API evolution
  *rewrites* the contract it touches; it does not *extend* a
  contract that remains intact. This fails P-B3.1 directly. API
  evolution requires its own launch — a parallel-scope wave on the
  model of Wave-S, or an explicit supersession of the governing ADR
  via amendment — not a B3 entry. The schema v1→v2 cadence committed
  by [ADR-0021](./0021-mode-as-primitive.md) §Notes is the only
  API-evolution mechanism currently sanctioned; future cadence
  changes are themselves out-of-scope for B3.

### (c) When B3 closes

**B3 stays open indefinitely.** Unlike Waves 1, 2, and 3 (which had
explicit gate criteria and definite closure events) and Wave-S
(which has a partial-gate and a full-gate criterion in
[ADR-0020](./0020-wave-s-launch.md)), B3 has no gate. Evolution is
continuous by definition (P-B3.1: B3 extends what exists), and the
platform extends itself across its operating life.

B3 closes only if the platform itself is sunset, replaced, or
re-launched under a new framing that supersedes
[ADR-0020](./0020-wave-s-launch.md) /
[ADR-0021](./0021-mode-as-primitive.md) /
[ADR-0022](./0022-kind-catalog.md) /
[ADR-0023](./0023-sources-schema.md). In that event, B3 closes
alongside the ADRs it conforms to, and a new evolutionary lane is
launched inside the successor framing.

Each individual B3-N item closes via the standard loop: study →
critique → ADR. The B3 section in
[`studies/foundation/06-decision-log.md`](../../studies/foundation/06-decision-log.md)
remains live for new B3-N entries indefinitely.

### Per-family scope

The three in-scope families resolve as follows. Each family's
"depends on" list identifies the ADR(s) that govern its constraint
envelope; a B3-N item in the family must conform to those ADRs to
satisfy P-B3.2.

#### Kind family extensions

- **Captures:** the *intent* behind a new kind whose shape reaches
  beyond a routine additive PR — a kind whose params reshape
  contributor expectations, or whose source-mode binds it to a
  substrate that [ADR-0023](./0023-sources-schema.md) has decided.
  A routine `set.*` or `record.*` kind whose params are trivial and
  whose source-mode is already-bound stays a catalog PR; a kind
  whose intent benefits from study trail becomes a B3-N entry.
- **Depends on:** [ADR-0022](./0022-kind-catalog.md) §evolution
  rules (additive PR threshold vs. ADR-required threshold);
  [ADR-0021](./0021-mode-as-primitive.md) mode-as-primitive grammar
  (`^(set|record)\..+$`).

#### Capability mode extensions

- **Captures:** extensions to mode-derived behavior *within* the
  existing set/record duo — a new lint cross-check that refines
  mode-prefix enforcement, a new derived-capability signal exposed
  to the loader or scheduler, a refinement to how capability is
  dispatched at runtime. Adding a **new mode** (a third top-level
  mode alongside `set` and `record`) is *out* of B3; it would
  constitute a Wave-S-scale launch because it would re-open
  [ADR-0021](./0021-mode-as-primitive.md)'s mode-as-primitive
  framing.
- **Depends on:** [ADR-0021](./0021-mode-as-primitive.md) (mode
  field, lint cross-checks, capability-derived-from-mode rule);
  [ADR-0023](./0023-sources-schema.md) (mode-to-source binding,
  since a mode extension may interact with source binding).

#### Tooling extensions

- **Captures:** additions to `tools/lint/`, the manifest publisher,
  the dry-run runner, the engine dispatcher, and adjacent tooling
  that **extend** contract coverage without changing the contract
  shape. New lint cross-checks beyond the eight committed by
  [ADR-0021](./0021-mode-as-primitive.md) /
  [ADR-0022](./0022-kind-catalog.md) /
  [ADR-0023](./0023-sources-schema.md) are the canonical example.
  Reshaping loader contracts is *amendment*, not B3 (it violates
  P-B3.1 — loader contract reshape is rewrite).
- **Depends on:** [ADR-0021](./0021-mode-as-primitive.md) (lint
  cross-checks #1–#4, loader v2 schema dispatch);
  [ADR-0022](./0022-kind-catalog.md) (lint cross-checks #5, #6 —
  catalog membership, per-kind params);
  [ADR-0023](./0023-sources-schema.md) (lint cross-checks #7, #8 —
  `source.type` vs. mode, `source.type` vs. catalog `source_mode`).

---

## Consequences

1. **One document, one row, one decision tree.** A contributor
   onboarding to evolutionary work reads exactly one ADR (this one)
   and scans exactly one table row (`B3-launch` in
   [`studies/foundation/06-decision-log.md`](../../studies/foundation/06-decision-log.md))
   before proposing an extension. The governance surface for
   evolution is fixed.

2. **Catalog PRs are unaffected.** Additive kind additions landing
   under [ADR-0022](./0022-kind-catalog.md) §evolution via
   CODEOWNERS dual review continue exactly as committed there. B3
   governs the *decision-history* layer above the catalog, not the
   catalog PR cadence itself. A routine additive kind addition stays
   a catalog PR; an addition whose intent benefits from study trail
   becomes a B3-N item.

3. **A new triage vocabulary lands in the decision log.** The four
   outcomes — **B3 / B2 / amendment / rejected** — are now the
   official outcome set for proposed extensions. The `rejected`
   outcome previously had no place in the log; it now has one, via a
   one-line entry recording the rejection rather than the silence.

4. **Substrate, performance, and API-evolution are closed off
   explicitly.** Future ambiguity is adjudicated against the
   rationale recorded in §(b) above — citing
   [ADR-0020](./0020-wave-s-launch.md) for substrate and P-B3.1 for
   performance and API — not against operator memory.

5. **B3 stays open across the platform's operating life.** Unlike
   every prior wave, B3 has no completion gate. The decision-log
   `B3-launch` row reaches `resolved-adr` on acceptance of this
   ADR; the B3 section remains live for new B3-N entries
   indefinitely.

6. **Per-B3-N items follow the same loop as every prior decision.**
   No new playbook is introduced at launch; existing
   [`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
   and
   [`.claude/playbooks/acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md)
   apply unchanged. If a B3-specific delta emerges from the first
   B3-N session, a follow-up edit to those playbooks is registered
   then — not pre-emptively.

7. **Tag-prefixing for B3-N rows is deferred.** Whether B3 rows are
   family-prefixed (e.g., `B3-K-1`, `B3-M-1`, `B3-T-1`) or kept flat
   (`B3-1`, `B3-2`, …) is decided by the first B3-N registration's
   precedent. This ADR does not commit either shape.

8. **The `rejected` outcome's vocabulary integration is deferred.**
   Whether the `rejected` outcome requires its own entry in
   [`studies/foundation/06-decision-log.md`](../../studies/foundation/06-decision-log.md)
   §Status Vocabulary, or stays implicit in the eligibility
   branches, is decided when the first rejected case lands. The
   first rejection will reveal whether the vocabulary needs
   explicit accommodation.

9. **Borderline-case arbitration defaults to the CODEOWNERS
   reviewer.** When a proposed extension is borderline between the
   "additive catalog PR" path
   ([ADR-0022](./0022-kind-catalog.md) §evolution) and the "B3-N
   entry" path, the CODEOWNERS reviewer assigned to the PR can hold
   the PR until a B3-N study lands. The first borderline case
   confirms or revises this default.
