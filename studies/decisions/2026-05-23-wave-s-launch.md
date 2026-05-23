<!-- path: studies/decisions/2026-05-23-wave-s-launch.md -->

# Wave-S — Record-Oriented Capability Launch Study

## Metadata

- **Wave reference:** Wave-S (record-oriented capability)
- **Status:** resolved-study (Wave-S, session 1; two critique rounds; round 2 cleared with no blocking findings)
- **Last updated:** 2026-05-23
- **Upstream resolved:** Waves 1, 2, and 3 — all gates met
  (2026-05-21, 2026-05-21, 2026-05-23 respectively). The eight
  set-oriented ADRs that this wave most directly interacts with —
  [ADR-0002](../../docs/adr/0002-run-identity-and-idempotency.md),
  [ADR-0003](../../docs/adr/0003-result-write-model.md),
  [ADR-0004](../../docs/adr/0004-failure-scope.md),
  [ADR-0006](../../docs/adr/0006-alert-routing-contract.md),
  [ADR-0007](../../docs/adr/0007-loader-scheduler-retry-failure-semantics.md),
  [ADR-0010](../../docs/adr/0010-substrate-posture.md),
  [ADR-0014](../../docs/adr/0014-trigger-handler-contract.md), and
  [ADR-0017](../../docs/adr/0017-substrate-posture-amendment.md) —
  all carry a single-line scope note (added 2026-05-23) declaring
  that they apply to **set-oriented capability realized over
  BigQuery**, and pointing forward to ADR-0020 (this study's
  promotion target) as the record-oriented integration point.
- **Downstream open:** seven B0-S items, **B0-S1 … B0-S7**,
  enumerated in [Consequences §6.2](#62--b0-s-items).
- **Promotion target:** `docs/adr/0020-wave-s-launch.md`
  (provisional).
- **Locked premises** (operator-declared, not litigated in this
  study):
  - **P1** — *Mode is the primitive.* Set-oriented vs
    record-oriented is the architectural split of the platform;
    substrate (BigQuery, Kafka, Pub/Sub, etc.) is a deployment
    detail downstream of mode.
  - **P2** — *Kind prefix discipline.* Every DSL kind carries its
    mode as a name prefix: `set.*` for set-oriented, `record.*`
    for record-oriented.
  - **P3** — *Capability is derived from mode, not declared.* An
    entity in `record.*` mode inherits record-mode capability; no
    independent capability field is carried on the entity.
  - **P4** — *Execution unified-vs-parallel is deferred.* Whether
    the engine runs one unified runner that switches on mode or
    two parallel runners (one per mode) is deferred to **B0-S5**,
    which must produce an objective decision criterion before
    closing.
- **Loop discipline:** same protocol as Waves 1 / 2 / 3 — study →
  `/critique` (≥1 round) → operator acceptance → promotion to ADR.
  Per-B0-S items follow the `/resolve-b0` shape that landed B0-1 …
  B0-7. See
  [`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  and
  [`.claude/playbooks/acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md).

---

## Context

The DQ Platform's published architecture, as of the close of Wave 3
on 2026-05-23, describes a **set-oriented** data quality engine:
declarative rules are evaluated against tabular state in BigQuery
over time windows, with results written append-only into
`dq_executions` and `dq_check_results`, alerts routed via Pub/Sub on
attempt boundaries, and the runtime governed by an `execution_id`
formula keyed on window endpoints and trigger source. Every ADR
landed to date is consistent with that picture, and the eight ADRs
listed in the Metadata block now carry an explicit scope note saying
so.

What the platform is **not yet** is record-oriented. Foundation
documents
[`01-charter-and-principles.md`](../foundation/01-charter-and-principles.md)
(chartering — "Controlled evolution toward stream-compatible checks
over Kafka") and
[`04-system-architecture.md`](../foundation/04-system-architecture.md)
(§G5 — "Stream evolution preserves conceptual continuity") name
stream-based validation as a future capability of the platform from
its first session, but no ADR has ever committed an integration
point for that capability. As of today, record-oriented work lives
only as project aspiration.

The scope-note pass committed on 2026-05-23 makes this gap visible:
every set-oriented ADR now points forward to **ADR-0020** as the
integration point that does not yet exist. This study creates that
integration point by **launching Wave-S** — a new wave of decision
work, sequenced and gated like Waves 1 and 2, whose B0-S items
collectively define record-oriented capability without re-opening
any of the frozen set-oriented ADRs.

Three context facts shape the study:

1. **Set-oriented ADRs are frozen as set-oriented.** R3 forbids
   revisiting settled decisions without strong cause. Today's scope
   notes pin those eight ADRs to set-mode; record-mode must be
   *new* work, not a *backwards* re-edit.
2. **Mode-as-primitive (P1) is the architectural lever.** A
   substrate-organised framing leaves the question *do I evaluate
   a set, or do I evaluate a record?* implicit — and that question
   re-emerges underneath every substrate decision: result-write
   shape, identity formula, alert dedup, failure-scope semantics.
   P1 puts mode at the top of the abstraction tree on day one so
   the question is asked once, explicitly, and answered for every
   downstream contract. *(New contribution proposed here, requires
   review.)*
3. **Deferral has costs.** Every record-mode question raised in a
   set-oriented session so far (alert dedup for streams, loader
   semantics for topics, failure scope for per-record errors) has
   been deferred to "the stream wave". With no launching artefact,
   each deferral lands in a different study's Open Questions, and
   the platform accumulates orphan TBDs. ADR-0020 collects them.

This study does **not** resolve any of the seven B0-S items it
enumerates. It is a launch document. Each B0-S item will get its
own study, critique pass, and ADR on its own schedule.

---

## Decision Drivers

- **DD-S.1** — **R3 honoured.** Set-oriented ADRs are frozen as
  set-oriented via today's scope-note pass. Record-mode must be a
  new wave, not a re-edit of frozen ADRs.

- **DD-S.2** — **Mode-as-primitive is the architectural
  invariant.** The wave's framing must encode P1 at the wave
  level, not bury it inside per-ADR adjustments — otherwise the
  primitive drifts as soon as a new substrate is added. *(New
  contribution proposed here, requires review.)*

- **DD-S.3** — **Reuse the Wave 1 / 2 / 3 protocol.** The team and
  the playbooks already run a study → critique → promotion loop.
  Wave-S adopts it unchanged; the only delta is the `S` prefix on
  the B-item numbering.

- **DD-S.4** — **Allow partial closure.** Not all seven B0-S items
  need to resolve before any record-mode work can start. The
  foundational triplet (mode primitive, kind catalog, sources
  schema) must ship first; execution-shape decisions can lag.

- **DD-S.5** — **Sequencing minimizes rework.** The triplet S1 / S2
  / S3 ships as ADRs before the execution decisions S4 / S5 / S6 /
  S7 are opened, because every execution decision presumes the
  triplet and would be re-litigated if the triplet changed under it.

- **DD-S.6** — **Cost discipline is a first-class concern.**
  Foundation principle P4 ("cost is a first-class constraint")
  carries over verbatim, but record-mode introduces a different
  cost shape — throughput budgets, backpressure, dead-letter caps,
  consumer lag ceilings — that is not captured by the existing
  B1-2 row, whose scope is read here as **set-mode cost ceilings
  only** (window size, concurrency, failed samples, dry-run
  enforcement) under BigQuery. Wave-S elevates record-mode cost
  to a B0-S item; the priority asymmetry (B0-S7 vs. B1) is
  justified by **substrate coupling** — cost ceilings inherit
  from substrate semantics, and record-mode's substrate is itself
  a Wave-S decision (B0-S3), so cost work cannot be deferred to a
  later wave. Set-mode's substrate (BigQuery) was settled at Wave
  1 and cost work could safely defer to B1.

- **DD-S.7** — **Kind-prefix discipline guards the gap.** Between
  S1's promotion and S2's catalog completion, the engine will be
  half-built: it knows mode is the primitive, but the catalog of
  record-mode kinds is still in draft. The `set.*` / `record.*`
  prefix gate at the lint layer keeps the boundary explicit even
  while the catalog is partial. *(New contribution proposed here,
  requires review.)*

---

## Considered Options

The options below are **launch postures**, not implementation
options. They answer the question *"how should the platform absorb
record-oriented capability?"* — not *"how should record-mode work
internally?"* (The latter is the job of the B0-S items themselves.)

*Note on a P2-adjacent alternative:* a fifth shape — **mode preserved
as a metadata field on each kind, rather than as a name prefix** —
is set aside here by the **P2 lock** in Metadata (kind prefix
discipline is operator-declared and not litigated in this study).
Whether the catalog uses prefixed names (`set.row_count_positive`)
or unprefixed names with a `mode:` field is a B0-S2 sub-decision
that proceeds under P2's lock; it is **not** an Option E in this
launch enumeration.

### Option A — Defer indefinitely

**Shape.** Leave record-mode in foundation 03 §future. No
launching ADR. The 2026-05-23 scope notes remain pointed at an
ADR-0020 that does not exist. Record-mode questions continue to
land in adjacent studies as deferred Open Questions.

**Cost.** Every record-mode question encountered in any future
session becomes an unplanned amendment to an existing ADR or a
floating TBD in a B1/B2 row. The mode primitive (P1) drifts
because no document anchors it. The scope notes promise an
ADR-0020 that never materialises, which is a documentation defect
on day one.

**Verdict.** Rejected. The scope-note pass has already exposed the
line; further deferral is now actively misleading to readers.

### Option B — Single-ADR amendments to ADRs 0002–0017

**Shape.** Open an amendment ADR (or in-place revision) for each
of the eight set-oriented ADRs. Each amendment enumerates the
record-mode variant of that ADR's contract. Mode appears as a
section inside each ADR rather than as the wave's organising
primitive.

**Cost.** Doubles the scope of every amended ADR. Conflates two
modes inside a single document, defeating the MADR "one decision
per ADR" property. Erodes R3, since the frozen set-oriented ADRs
get edited rather than supplemented. Re-buries the mode question
that the scope-note pass just surfaced. Forces eight critique
rounds where one wave does the same job.

**Verdict.** Rejected. The shape contradicts P1: mode becomes a
sub-section of every ADR rather than the architectural primitive.

### Option C — Dedicated Wave-S with B0-S backlog (recommended)

**Shape.** A new wave, gated like Waves 1 and 2, with its own
backlog of seven B0-S items (S1 … S7) corresponding to the
record-oriented analogues of the foundational decisions made in
Wave 1. This launching study promotes to **ADR-0020**, the
integration point pointed to by every scope-noted ADR. Each B0-S
item gets its own study and promotes to an ADR (provisional
**0021 … 0027**) on its own schedule. The wave admits a **partial
gate**: once the foundational triplet (S1, S2, S3) is at
`resolved-adr`, record-mode kinds may enter the catalog and the
loader, even though the execution-shape decisions (S4 … S7)
remain open.

**Cost.** Standard wave overhead: seven critique loops, seven
ADRs, decision-log additions, playbook reuse. No set-oriented ADR
is touched.

**Verdict.** **Recommended.** Honours R3 (DD-S.1), encodes P1 at
the wave level (DD-S.2), reuses the existing protocol (DD-S.3),
supports partial closure (DD-S.4 / DD-S.5), elevates record-mode
cost to a first-class item (DD-S.6), and lands the kind-prefix
lint gate at S1 promotion (DD-S.7).

### Option D — Full DSL rewrite

**Shape.** Open a Wave 4 that collapses set and record into a
unified DSL taxonomy from scratch, rewriting ADRs 0002–0007 in
place. Mode is *eliminated* as a primitive in favour of a unified
kind catalog where set and record evaluation differ only in their
runtime adapters.

**Cost.** Discards the operationally proven set-oriented ADRs and
the runbook context (`docs/runbooks/`) built atop them. Re-opens
every settled decision under R3, requiring strong cause for each
— and R3 forbids exactly the bottom-up re-litigation this option
mandates. Likely costs are months of re-litigation for a benefit
that Option C achieves with a clean seam (the mode primitive).

**Verdict.** Rejected. No concrete benefit over Option C
identified; the costs are large and the precedent is poor.

---

## Recommendation

**Pick Option C — Dedicated Wave-S with B0-S backlog.**

Rationale, tied directly to drivers:

- **R3 (DD-S.1)** — Option C is the only posture that does not
  re-edit a frozen ADR. Set-oriented ADRs remain set-oriented;
  record-oriented capability is additive.
- **P1 (DD-S.2)** — Option C is the only posture that puts the
  mode primitive at the wave level. Options B and D dilute it
  into per-ADR sub-sections or eliminate it entirely.
- **Protocol reuse (DD-S.3)** — Option C is the only posture that
  re-uses the established playbook unchanged. Options A and D have
  no playbook; Option B's "amend eight ADRs in parallel" has no
  precedent.
- **Partial closure (DD-S.4) and sequencing (DD-S.5)** — Option C
  is the only posture that admits a partial gate. Option D forces
  all-at-once; Option B forces eight parallel amendments; Option A
  forces nothing and resolves nothing.
- **Cost as a first-class concern (DD-S.6)** — Option C elevates
  record-mode cost to B0-S7. Options A, B, D leave it to drift.
- **Kind-prefix discipline (DD-S.7)** — Option C lands the
  `set.*` / `record.*` lint gate at S1's promotion; no other
  option commits to it.

**One-line decision summary table:**

| Decision | Outcome |
|---|---|
| Launch posture | Dedicated Wave-S (Option C) |
| Wave scope | Seven B0-S items (S1 – S7) |
| Mode primitive | Yes (P1 locked) |
| Kind prefix | `set.*` / `record.*` (P2 locked) |
| Capability declaration | Derived from mode (P3 locked) |
| Execution unified-vs-parallel | Deferred to B0-S5 (P4 locked) |

---

## Consequences

### 6.1 — Cross-cutting consequences

- **C-S.1** — The eight set-oriented ADRs that received scope notes
  on 2026-05-23 gain **ADR-0020** as the explicit forward-pointer
  for record-mode capability. ADR-0020 is this study's promotion
  target; the scope notes' phrase "ADR-0020 forthcoming" is
  redeemed by Wave-S's first promotion.

- **C-S.2** — `studies/foundation/06-decision-log.md` gains a new
  **"Wave-S — Record-Oriented Capability Decisions" table**,
  parallel to the existing "Wave 2 — Platform Decisions" and
  "Wave 3 — Phases" tables, with the same column layout as the
  existing B0 table (`# | Topic | Status | Key Question | Why It
  Matters | Expected Output`). Rows are B0-S1 … B0-S7, each at
  status `open` on first registration. The **shape** is committed
  by this study; only the **edit itself** is deferred to a
  follow-up session.

- **C-S.3** — Wave-S becomes a tracked artefact in the
  decision-log's *Wave Gates* section: a **partial-Wave-S gate**
  and a **full Wave-S gate**, defined in §6.3 below.

- **C-S.4** — Until the partial-Wave-S gate is met, **no
  record-mode code is shipped**. This is the Wave-S analogue of R1
  (no production code during waves 1 and 2). The kind-prefix lint
  gate (C-S.5) is the one exception, because it ships with S1 and
  is itself part of the partial gate's payload. *(New contribution
  proposed here, requires review.)*

- **C-S.5** — The `set.*` / `record.*` **kind-prefix discipline
  lands at the lint layer** at S1's promotion time, even if S2's
  catalog is still in draft. This is what guarantees that
  half-built Wave-S state cannot leak record-mode kinds into
  set-mode runtime paths or vice versa. *(New contribution proposed
  here, requires review.)*

- **C-S.6** — The 2026-05-23 scope-note pass on ADRs 0002, 0003,
  0004, 0006, 0007, 0010, 0014, and 0017 references "ADR-0020
  forthcoming". To redeem that pointer, **the operator commits
  to landing this study's promotion to
  `docs/adr/0020-wave-s-launch.md` as the next ADR promotion in
  the repo** — no intervening promotion takes the ADR-0020 slot.
  Should the operator choose to land a different promotion first,
  the operator amends the scope notes' "ADR-0020 forthcoming"
  phrase in the same commit to point at the actually-assigned
  launch ADR number, and the per-item ADR slugs in this study's
  Promotion target section (provisional ADR-0021 … ADR-0027)
  shift in lockstep. *(New contribution proposed here, requires
  review.)*

### 6.2 — B0-S items

Each B0-S item below is one paragraph in the *decides / depends on
/ downstream* shape. None of these is resolved in this study; each
will be opened as its own study under the same `/resolve-b0`
protocol that produced B0-1 … B0-7.

#### B0-S1 — Mode as primitive

**Decides:** that `mode` is a typed, required field on the rule
artefact and on the entity declaration; that the kind catalog (S2)
and the source schema (S3) carry mode as their organising key;
that an entity's record-mode-or-set-mode capability is **derived
from its mode** (P3) and not carried as an independent capability
field; the exact yaml shape of the mode field (e.g., `mode: set`
vs `mode: record`) and its lint-time validation. **Depends on:**
P1, P2, P3 (operator-locked). **Downstream:** unblocks S2 (kinds
must declare their mode) and S3 (sources must match their entity's
mode); enables the kind-prefix lint gate committed in C-S.5; the
linter rule under `tools/lint/` lands at S1's promotion.

#### B0-S2 — Kind catalog

**Decides:** the registry of supported kinds, starting from the
existing `set.row_count_positive` (the only kind shipped in Wave 3
via [W3-P6c](../foundation/06-decision-log.md#wave-3--phases-scaffolding-sequencing))
and adding one or more inaugural `record.*` kinds whose shape is
chosen to exercise the record-mode plumbing minimally (likely a
single-record schema-conformance kind, with the specific kind
chosen during the B0-S2 study); how new kinds are added in the
future (governance step, schema-version bump under the
ADR-0001 compatibility contract); how a kind declares the source
shape it expects (set-mode source vs record-mode source). **Depends
on:** S1. **Downstream:** S3 (sources must validate against the
catalog); the schema files under `engine/schema/` and the mirrored
`rules/_schema/` gain a `record.*` half at S2's promotion.

#### B0-S3 — Sources schema

**Decides:** how a **source** is described in rule YAML in each
mode — a set source (a BigQuery table or view with its partition
column, table-ref, and dataset-ref, as ADR-0007 already presumes)
versus a record source (a stream substrate's topic/subscription
identifier with its watermark and offset-binding semantics, the
specific substrate to be picked during the B0-S3 study under
mode-derived capability per P3); how the source declaration cross-
checks against the kind catalog (S2) so that a `record.*` rule
cannot reference a BigQuery table; how the ADR-0007 loader path
extends to recognise record-mode sources without re-opening
ADR-0007's set-oriented contract. **Depends on:** S1, S2.
**Downstream:** unblocks **all** execution-layer decisions (S4 / S5
/ S6); is the last item of the foundational triplet, so its
promotion meets the partial-Wave-S gate (§6.3).

#### B0-S4 — Window semantics

*(Deferred. Opens only after the partial gate is met.)*

**Decides:** what "window" means for record-mode — tumbling,
sliding, session, watermark-bounded, or "no window, evaluate
per-record"; how watermarks interact with (or replace) the
ADR-0002 `execution_id` window-endpoint formula; whether
record-mode `execution_id` reuses the ADR-0002 formula or gains a
new shape under a `record.*` identity rule; how late-arrival
records are handled relative to the closed-window invariant
ADR-0002 currently encodes. **Depends on:** S1, S2, S3 promoted as
ADRs. **Downstream:** record-mode alert dedup (the record-mode
half of ADR-0006's per-attempt deduper), record-mode result-write
semantics (the record-mode half of ADR-0003's append-only model),
the record-mode trigger contract (the record-mode half of
ADR-0014's `execution_id` payload).

#### B0-S5 — Aggregation & unified-vs-parallel execution

*(Deferred; resolves P4.)*

**Decides:** whether the engine runs **one unified runner** that
switches on mode per evaluation, or **two parallel runners** (one
set, one record); the **objective decision criterion** P4
commits to, drafted during the B0-S5 study and likely combining
(a) operational blast-radius of a runner outage in each shape,
(b) the cost of duplicated lifecycle plumbing (loader, scheduler,
observability emission) under parallel runners, (c) whether
record-mode aggregations can reuse the set-mode `dq_executions` /
`dq_check_results` schema (with a `mode` column) or need a parallel
write path. The criterion is decided **before** the runner shape;
the criterion itself is the deliverable that satisfies P4's
deferral. *(New contribution proposed here, requires review.)* **Depends on:** S1,
S2, S3 promoted; S4 likely (window semantics influence aggregation
shape). **Downstream:** every runtime ADR — ADR-0007 (loader /
scheduler), ADR-0014 (trigger handler), and the engine binary
layout itself under `engine/cmd/`.

#### B0-S6 — Failure scope aggregated

*(Deferred.)*

**Decides:** how per-record failures aggregate into an entity-level
status, given that record-mode lacks the natural batch boundary
that ADR-0004 currently relies on; how the ADR-0004 status policy
(`pass` / `fail` / `error` / `degraded`) maps onto a continuous
stream of records (windowed rollup, sliding-fraction threshold,
per-watermark aggregation); whether per-record evidence is
retained, sampled, or dropped (B1-6 retention parameters need a
record-mode amendment). **Depends on:** S1, S4, S5. **Downstream:**
the record-mode half of ADR-0006 (alert routing), the record-mode
half of B1-6 (evidence retention), the new runbook seeds for
record-mode failure escalation under `docs/runbooks/`.

#### B0-S7 — Record-oriented cost guardrails

*(Deferred; first-class B0-S per DD-S.6.)*

**Decides:** the **throughput**, **backpressure**, **dead-letter**,
and **consumer-lag** ceilings that record-mode must respect under
each environment (local / qa / prod, per ADR-0018); how the
guardrails are enforced (engine-side, broker-side, or both); how
the guardrails compose with the existing BigQuery cost ceilings
(B1-2) so that an entity with both set-mode and record-mode rules
respects both budgets. **Depends on:** S1, S3, S5. **Downstream:**
the record-mode parallel of B1-2 lands in the decision log; the
runbook seeds for record-mode (refresh-failure escalation,
lag-spike remediation) gain numeric thresholds; ADR-0019's deploy
overlays under `deploy/overlays/` gain record-mode budget vars at
S7's promotion.

### 6.3 — Sequencing strategy and gate criteria

The B0-S items split into two phases on a deliberate boundary:

- **Phase Wave-S.α (foundational triplet) — D1 … D3 (S1, S2, S3).**
  Each gets a `/resolve-b0`-shaped study, one or more critique
  rounds, and a promotion to an ADR (provisional **ADR-0021,
  ADR-0022, ADR-0023**, subject to C-S.6). At the close of
  Phase α, the engine carries record-mode kinds in its catalog,
  the loader recognises record-mode sources, and the kind-prefix
  lint gate (C-S.5) is in place.
- **Phase Wave-S.β (execution shape) — D4 … D7 (S4, S5, S6, S7).**
  Each gets its own study and ADR. These items are interlocked
  (S5 may depend on S4, S6 on S5, S7 on S5) and so are not
  expected to ship in parallel; the sequencing inside Phase β is
  itself a small decision and is **not** committed by this study —
  it lands in the first B0-S4 study.
- **Sequencing rule (gate on promotion, not on study opening).**
  Phase β studies **may be drafted in parallel** with Phase α
  promotion. The dependencies declared in §6.2 gate at
  **promotion** granularity: no Phase β ADR is merged until its
  declared Phase α dependencies are at `resolved-adr`. This
  mirrors the parallel-drafting discipline used in Wave 1, where
  B0-1 … B0-7 studies overlapped in time and promotion order was
  the only gate.

**Partial-Wave-S gate criterion (explicit):**

> **The partial-Wave-S gate is met when B0-S1, B0-S2, and B0-S3 are
> all at status `resolved-adr` and the corresponding ADRs
> (provisional ADR-0021, ADR-0022, ADR-0023, subject to C-S.6) are
> merged into `docs/adr/`.**
>
> Until this gate is met, no record-mode code is shipped (C-S.4),
> with the single exception of the kind-prefix lint gate that
> ships *with* S1 (C-S.5). When met, the engine may execute
> record-mode kinds in catalog and loader paths; execution-shape
> behaviour (windowing, aggregation, failure scope, cost) remains
> open until the full gate.

**Full Wave-S gate criterion:** met when **all seven B0-S items**
are at status `resolved-adr` and their ADRs (provisional 0021 …
0027, subject to C-S.6) are merged. At full-gate, the platform has
a complete record-oriented capability, parallel in completeness to
the set-oriented capability that Wave 3 closed.

---

## Open Questions

- **OQ-S.1** — Does P3 (capability derived from mode) mean
  `rules/_owners.yaml`'s `capability:` field — if/where it appears
  — is **removed entirely**, or **kept as a redundant cross-check**?
  *Defer to B0-S1.* The B0-S1 study must decide explicitly.

- **OQ-S.2** — How does the existing `dq_executions` /
  `dq_check_results` write model (ADR-0003) treat record-mode runs
  — **parallel tables**, **same tables with an added `mode`
  column**, or **a derived view** layered on a single physical
  table per ADR-0010's lazy-view pattern? *Defer to B0-S5.* The
  decision is load-bearing for whether the runner is unified or
  parallel.

- **OQ-S.3** — Is record-mode **alert dedup** (ADR-0006's per-
  attempt deduper) keyed on the same `execution_id` formula as
  set-mode, or does record-mode require a new identity formula
  whose dedup window is watermark-bounded? *Defer to B0-S4 and
  B0-S5.* The answer depends on the windowing decision (S4) and
  the runner-shape decision (S5).

- **OQ-S.4** — Is **Wave-S itself** committed as one launching ADR
  (this study → ADR-0020) plus one ADR per B0-S item (0021+), or
  as a **single composite ADR** that absorbs all seven B0-S
  decisions? *Default: launching ADR (ADR-0020) plus per-item
  ADRs.* The operator confirms at the first promotion. If the
  operator picks the composite shape later, this Open Question is
  resolved retroactively and the per-item ADR slugs are released.

- **OQ-S.5** — Do foundation documents
  [`01-charter-and-principles.md`](../foundation/01-charter-and-principles.md)
  and
  [`04-system-architecture.md`](../foundation/04-system-architecture.md)
  need **post-Wave-S amendment notes** acknowledging the mode
  primitive once any B0-S item promotes? *Out of scope for this
  study.* Lift on first B0-S resolution.

---

## Promotion target

**Target:** `docs/adr/0020-wave-s-launch.md` *(provisional)*.

This launching study promotes to **ADR-0020** once at least one
round of `/critique` has been accepted by the operator and any
blocking findings are addressed. ADR-0020 is the integration
point that the 2026-05-23 scope-note pass on ADRs 0002, 0003,
0004, 0006, 0007, 0010, 0014, and 0017 explicitly points forward
to — the scope notes' phrase "ADR-0020 forthcoming" resolves at
that promotion.

Each of the seven B0-S items opens its own study under the
existing `/resolve-b0` protocol and promotes to its own ADR on its
own schedule (provisional ADR-0021 … ADR-0027, subject to C-S.6).
The partial-Wave-S gate criterion in [§6.3](#63--sequencing-strategy-and-gate-criteria)
gates record-mode code shipping; the full Wave-S gate criterion
closes the wave.

Per R8, the future ADR-0020 will be rewritten from this study, not
linked back to it. This study remains in `studies/decisions/` as
the reasoning artefact; ADR-0020 will read cold to a reviewer who
has never opened `studies/`.
