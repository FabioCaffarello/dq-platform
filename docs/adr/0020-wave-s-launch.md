<!-- path: docs/adr/0020-wave-s-launch.md -->

# ADR-0020 — Wave-S: Record-Oriented Capability Launch

- **Status:** accepted
- **Date:** 2026-05-23

---

## Context

The platform's published architecture, as of the close of Wave 3, is
set-oriented: declarative rules evaluate tabular state in BigQuery
over time windows, with results written append-only into
`dq_executions` and `dq_check_results`, alerts routed via Pub/Sub on
attempt boundaries, and the runtime governed by an `execution_id`
formula keyed on window endpoints and trigger source. Eight ADRs
that previously read as if universal — **ADR-0002** (run identity),
**ADR-0003** (result write model), **ADR-0004** (failure scope),
**ADR-0006** (alert routing), **ADR-0007** (loader, scheduler,
retry), **ADR-0010** (substrate posture), **ADR-0014** (HTTP trigger
handler), and **ADR-0017** (substrate amendment) — now carry an
explicit scope note declaring set-oriented capability scope and
forward-pointing to this ADR.

Stream-based / record-oriented validation has been named as a future
capability of the platform from the project's first session, but no
ADR has committed an integration point for that capability. The
absence is documented as a forward pointer in the eight scope-noted
ADRs; this ADR resolves it.

**ADR-0020 is the integration point.** It launches a new wave
("Wave-S") whose seven B0-S items collectively define record-oriented
capability without reopening any set-oriented ADR. The set-oriented
contracts ADR-0002…ADR-0017 stay set-mode; record-mode is additive
work.

Wave-S is a new wave parallel to Waves 1 and 2 in shape: a backlog
of B0-S decisions, each resolved by its own study under the
`/resolve-b0` protocol that landed B0-1 through B0-7, each promoted
to a per-item ADR on its own schedule. The wave admits a **partial
gate**: once the foundational triplet (B0-S1, B0-S2, B0-S3) is
`resolved-adr`, record-mode kinds may enter the catalog and the
loader, even though the execution-shape decisions (B0-S4…B0-S7)
remain open.

---

## Decision

### Locked architectural premises

These four premises govern the wave. They are not relitigated by any
B0-S item.

1. **Mode is the primitive.** Set-oriented vs record-oriented is the
   architectural split of the platform. Substrate (BigQuery, Kafka,
   Pub/Sub, etc.) is a deployment detail downstream of mode. Past
   tendencies to organise the platform along substrate axes leave the
   *do I evaluate a set, or do I evaluate a record?* question
   implicit, and that question re-emerges underneath every substrate
   decision — result-write shape, identity formula, alert dedup,
   failure-scope semantics. Mode at the top of the abstraction tree
   asks the question once and answers it for every downstream
   contract.

2. **Kind prefix discipline.** Every DSL kind carries its mode as a
   name prefix: `set.*` for set-oriented (e.g., the existing
   `set.row_count_positive` shipped in Wave 3), `record.*` for
   record-oriented. The prefix is enforced at the lint layer.

3. **Capability is derived from mode, not declared.** An entity in
   `record.*` mode inherits record-mode capability. No independent
   capability field is carried on the entity.

4. **Execution unified-vs-parallel is reserved for B0-S5.** Whether
   the engine runs one unified runner that switches on mode per
   evaluation, or two parallel runners (one set, one record), is the
   subject of B0-S5. B0-S5 must produce an objective decision
   criterion (combining operational blast-radius, duplicated-plumbing
   cost, and schema-reuse feasibility) **before** the runner shape is
   picked.

### The seven B0-S items

Wave-S is the union of these seven decisions. Each opens as its own
study and promotes to its own ADR.

- **B0-S1 — Mode as primitive.** Decides the typed `mode` field on
  the rule artefact and on the entity declaration; the yaml shape
  (`mode: set` vs `mode: record`) and its lint-time validation; the
  rule that the kind catalog (B0-S2) and the source schema (B0-S3)
  carry mode as their organising key. Promotion lands the
  kind-prefix lint gate.

- **B0-S2 — Kind catalog.** Decides the registry of supported
  `set.*` and `record.*` kinds (starting from the existing
  `set.row_count_positive` and adding one or more inaugural
  `record.*` kinds), the governance process for adding kinds, the
  schema-version bump rule under the ADR-0001 compatibility
  contract, and the way a kind declares the source shape it expects.

- **B0-S3 — Sources schema.** Decides how a source is described in
  rule YAML in each mode — set source (BigQuery table or view with
  partition column, table-ref, dataset-ref, as ADR-0007 already
  presumes) versus record source (stream substrate topic /
  subscription with watermark and offset-binding semantics, the
  specific substrate picked under mode-derived capability); how the
  source declaration cross-checks against the kind catalog; and how
  the ADR-0007 loader path extends to recognise record-mode sources
  without reopening ADR-0007's set-oriented contract.

- **B0-S4 — Window semantics.** Decides what "window" means for
  record-mode (tumbling, sliding, session, watermark-bounded, or
  "no window — evaluate per-record"), how watermarks interact with
  (or replace) the ADR-0002 `execution_id` window-endpoint formula,
  whether record-mode `execution_id` reuses the ADR-0002 formula or
  gains a new shape, and how late-arrival records are handled
  relative to the closed-window invariant ADR-0002 encodes.

- **B0-S5 — Aggregation and unified-vs-parallel execution.**
  Decides the runner shape per the objective criterion above, and
  whether record-mode aggregations reuse the set-mode
  `dq_executions` / `dq_check_results` schema (with an added `mode`
  column) or need a parallel write path.

- **B0-S6 — Failure scope aggregated.** Decides how per-record
  failures aggregate into an entity-level status when record-mode
  lacks the natural batch boundary that ADR-0004 currently relies
  on; how the ADR-0004 status policy (`pass` / `fail` / `error` /
  `degraded`) maps onto a continuous stream of records (windowed
  rollup, sliding-fraction threshold, per-watermark aggregation);
  whether per-record evidence is retained, sampled, or dropped.

- **B0-S7 — Record-oriented cost guardrails.** Decides the
  throughput, backpressure, dead-letter, and consumer-lag ceilings
  that record-mode must respect under each environment per
  ADR-0018, how the guardrails are enforced (engine-side,
  broker-side, or both), and how the guardrails compose with the
  set-mode BigQuery cost ceilings (B1-2) so that an entity with
  rules in both modes respects both budgets.

### Sequencing and gates

**Foundational triplet first.** B0-S1, B0-S2, and B0-S3 ship as
ADRs before B0-S4 through B0-S7 are opened-for-promotion. Every
execution-shape decision (S4 onward) presumes the triplet, so
sequencing the triplet first minimizes rework.

**Promotion is the gate, not study opening.** Phase β studies
(B0-S4…B0-S7) may be drafted in parallel with Phase α promotion. No
Phase β ADR is merged until its declared Phase α dependencies are
at `resolved-adr`. This mirrors the parallel-drafting discipline
that landed B0-1 through B0-7 in Wave 1, where promotion order was
the only gate.

**Partial-Wave-S gate.** The gate is met when B0-S1, B0-S2, and
B0-S3 are at status `resolved-adr` and their ADRs are merged into
`docs/adr/`. Until the gate is met, no record-mode code ships,
with the single exception of the kind-prefix lint gate that ships
with B0-S1 (the gate's enforcement payload is part of B0-S1's own
deliverable). Once met, the engine may carry record-mode kinds in
the catalog and the loader; execution-shape behaviour (windowing,
aggregation, failure scope, cost) remains open until the full
gate.

**Full Wave-S gate.** Met when all seven B0-S items are at status
`resolved-adr` and their ADRs are merged. At full-gate, the
platform has a complete record-oriented capability parallel in
completeness to the set-oriented capability that Wave 3 closed.

**Per-item ADR numbering.** B0-S1 through B0-S7 promote in order to
`docs/adr/0021-…` through `docs/adr/0027-…` respectively, modulo
shifts if an unrelated promotion lands between B0-S items. The
expected sequence is descriptive, not reserved.

---

## Consequences

1. **Set-oriented ADRs remain set-mode-scoped.** ADR-0002,
   ADR-0003, ADR-0004, ADR-0006, ADR-0007, ADR-0010, ADR-0014, and
   ADR-0017 are not reopened. Record-mode is additive via Wave-S;
   the scope notes those ADRs carry are now redeemed by this ADR.

2. **No record-mode code ships until the partial gate.** Wave-S
   carries an R1-analogue: between today and the close of the
   partial gate, the engine, the loader, the runner, the result-
   write layer, and the alerting layer remain set-mode-only. The
   kind-prefix lint gate (which ships with B0-S1) is the single
   exception, because it is itself part of the partial gate's
   payload — it guarantees that half-built Wave-S state cannot leak
   record-mode kinds into set-mode runtime paths or vice versa.

3. **Kind-prefix discipline lands at the lint layer at B0-S1
   promotion.** The `set.*` / `record.*` enforcement is in
   `tools/lint/` from the moment B0-S1 lands, even if the B0-S2
   catalog is still in draft. This is what makes Phase α safely
   incremental.

4. **The decision log carries a Wave-S section.** The "Wave-S —
   Record-Oriented Capability Decisions" table sits in
   `studies/foundation/06-decision-log.md` parallel to the Wave 2
   and Wave 3 tables, with rows B0-S1 through B0-S7. Each row
   advances from `open` → `resolved-study` → `resolved-adr` as its
   B0-S resolves.

5. **B1-2's scope is set-mode cost ceilings only.** The existing
   B1-2 row (window size, concurrency, failed samples, dry-run
   enforcement) is read as set-mode under BigQuery. B0-S7 is the
   record-mode parallel. The priority asymmetry — B0-S7 at B0
   while B1-2 is at B1 — reflects substrate coupling: cost ceilings
   inherit from substrate semantics, and record-mode's substrate is
   itself a Wave-S decision (B0-S3), so record-mode cost work
   cannot be deferred to a later wave. Set-mode's substrate
   (BigQuery) was settled at Wave 1, so cost work could safely
   defer to B1.

6. **B0-S5 owns the runner-shape decision.** No ADR before B0-S5
   commits the unified-vs-parallel runner. The B0-S5 study
   produces the objective criterion (which is itself the
   deliverable of the locked-premise deferral) before the runner
   shape is picked.

7. **Foundation amendments lift on first B0-S resolution.**
   Foundation documents
   `studies/foundation/01-charter-and-principles.md` and
   `studies/foundation/04-system-architecture.md` may need
   amendment notes acknowledging the mode primitive once any B0-S
   item promotes. The amendment is lifted at first B0-S resolution
   and lives outside any individual B0-S study.

8. **Wave-S follows the same loop discipline as Wave 1.** Each
   B0-S item runs under
   `.claude/playbooks/wave-1-session-loop.md` and self-verifies
   against `.claude/playbooks/acceptance-criteria.md`, with the `S`
   prefix on the B-item numbering as the only delta.

---

## Notes

- The decision on whether `rules/_owners.yaml`'s `capability:`
  field is removed entirely under the mode-derives-capability
  premise, or kept as a redundant cross-check, lives inside
  B0-S1.
- The decision on whether `dq_executions` / `dq_check_results`
  treat record-mode runs as parallel tables, same tables with an
  added `mode` column, or a derived view layered on a single
  physical table per the ADR-0010 lazy-view pattern, lives inside
  B0-S5.
- The decision on whether record-mode alert dedup reuses the
  ADR-0006 per-attempt deduper keyed on the same `execution_id`
  formula, or gains a new identity formula with a watermark-bounded
  dedup window, lives inside B0-S4 and B0-S5.
- Whether Wave-S itself remains one launching ADR (this one) plus
  one ADR per B0-S item, or collapses to a single composite ADR
  later, is a meta-choice the operator confirms at the first B0-S
  promotion. The default is per-item ADRs.
- The Wave-S gate criteria (partial and full) are added to the
  decision log's *Wave Gates* section by a follow-up edit; the
  criteria themselves are committed here.
