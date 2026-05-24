<!-- path: docs/adr/0025-aggregation-and-runner-shape.md -->

# ADR-0025 — Aggregation and Runner Shape (P4 Retirement)

- **Status:** accepted
- **Date:** 2026-05-24

---

## Context

ADR-0020 launched Wave-S with four locked premises:
**P1** mode is the architectural primitive,
**P2** kind-prefix discipline,
**P3** capability is derived from mode,
**P4** execution unified-vs-parallel is reserved for B0-S5 with an
**objective decision criterion** as the deliverable.

P1, P2, and P3 are realised in the foundational triplet
([ADR-0021](./0021-mode-as-primitive.md) mode primitive,
[ADR-0022](./0022-kind-catalog.md) kind catalog,
[ADR-0023](./0023-sources-schema.md) sources schema). The first
Phase β ADR ([ADR-0024](./0024-window-semantics.md) window
semantics) committed the record-mode windowing model and
extended ADR-0002's `trigger_source` enum. **This ADR retires
P4** by committing the objective criterion, applying it to
record-mode, and shipping the runtime decisions that follow.

What remains for record-mode runtime to materialise after this
ADR: the engine binary's **shape** (one runner that switches on
mode per evaluation, or two parallel runners — one set, one
record — sharing the engine binary while running independent
evaluation loops), the **result-write schema** (does record-mode
reuse the set-mode `dq_executions` / `dq_check_results` schema
or commit a parallel write path), and the **within-window
aggregation seam** (where N per-record results in a window
combine into one check result per check per window).

The current engine state is set-mode-only:

- `engine/cmd/dq-engine/main.go` is the single engine binary
  entrypoint.
- `engine/internal/runner/` houses the runner package; the
  dispatcher in `engine/internal/eval/` invokes per-kind
  handlers per the [ADR-0022](./0022-kind-catalog.md) catalog
  dispatch model.
- The loader (per [ADR-0007](./0007-loader-scheduler-retry-failure-semantics.md))
  loads manifests and rules with v2 schema dispatch per
  ADR-0021; the loader's record-mode rejection error path (per
  ADR-0021) is in code but is scheduled for removal in the
  combined implementation commit that lands ADRs 0021–0024 (and
  this ADR) together.
- The result writer (per [ADR-0003](./0003-result-write-model.md))
  writes append-only rows to `dq_executions` and
  `dq_check_results`; ADR-0003 carries the 2026-05-23
  scope note pinning its set-orientation.

The platform principles that bear: **P1** (the runner-shape
decision must not leak into the declarative DSL beyond what mode +
kind + source already expose); **P2 mirror, foundation 01
§"Determinism"** (whichever shape is picked, same input → same
`execution_id` set — satisfied by ADR-0024's record-mode
`execution_id` formula); **P3** (capability is the runtime
realisation of the mode-as-primitive choice, surfaced now in the
runner-shape commitment); **P5** (evolution contract-driven —
ADR-0003, ADR-0007, ADR-0014 extensions live under
[ADR-0001](./0001-engine-rules-compatibility.md)'s additive-
within-major contract).

---

## Decision

### The objective criterion (P4 deliverable)

P4 is satisfied not by a runner-shape pick alone, but by an
**objective decision criterion** that any future runner-shape
question (a third mode added; a mode split further; modes
merged) can re-run without consulting this ADR's author. The
criterion is two named axes with named scales and a decision
table with an explicit tie-breaker.

**Axis 1 — Substrate-shape divergence.** How different is the
new mode's I/O pattern from the existing modes that would share
the runner? Three levels:

- **LOW** — same I/O pattern as an existing mode (same
  concurrency primitives, same backpressure shape, same state
  machine).
- **MEDIUM** — same family of I/O pattern (e.g., both polling-
  based) but different operational lifecycle.
- **HIGH** — fundamentally different I/O pattern (different
  concurrency primitives, different backpressure, different
  state machine, different failure modes).

**Threshold rule for Axis 1 (mismatch count).** Count
mismatches across four factors: **(i)** concurrency primitives,
**(ii)** backpressure shape, **(iii)** state machine,
**(iv)** liveness signal. **LOW** = 0 mismatches;
**MEDIUM** = 1–2 mismatches; **HIGH** = 3–4 mismatches. The
factor list is open for additive extension if a future mode
surfaces a fifth dimension; the threshold rule preserves
reproducibility — two analysts counting the same mismatches
reach the same level.

**Axis 2 — Failure-isolation requirement.** How critical is it
that one mode's failure does not propagate to another? Two
levels:

- **LOW** — mode-cross failure is operationally tolerable
  (both modes are well-proven and operationally similar;
  cross-failure rarely occurs and is easy to recover).
- **HIGH** — mode failure must not affect other modes (a new
  high-risk mode added to a proven runtime; or a compliance-
  driven separation between modes).

**Decision rule:**

| Axis 1 (divergence) | Axis 2 (isolation) | Runner shape |
|---|---|---|
| LOW | LOW | **Unified runner** (single loop dispatches by mode) |
| LOW | HIGH | **Parallel runners, single binary** (separate loops, shared upstream) |
| MEDIUM | LOW | **Unified runner** with mode-specific worker pools |
| MEDIUM | HIGH | **Parallel runners, single binary** |
| HIGH | LOW | **Parallel runners, single binary** |
| HIGH | HIGH | **Parallel runners, single binary** (consider OS-process parallelism only if OS-level isolation is operationally required) |

**Tie-breaker rule.** When axis assignment is ambiguous — when
an analyst could reasonably defend two adjacent levels for
either axis — default to the **more-isolation-friendly outcome**
(parallel runners, single binary). The cost of unnecessary
isolation is bounded (one extra runner loop and worker pool);
the cost of insufficient isolation is unbounded (a production
incident with cross-mode propagation). The asymmetry justifies
the conservative default.

**Single-binary form is the default** for parallel runners.
**OS-process parallelism** (separate engine binaries per mode)
is reserved for cases where OS-level isolation is operationally
required — e.g., when an upstream policy forbids shared OS
resources, or when one mode's CPU/memory characteristics
catastrophically interfere with another's. This ADR commits the
escalation procedure but does not commit any current need for
it.

The criterion is a reusable platform artefact: future
runner-shape questions re-run the same two-axis assessment
against the same decision table.

### Application to record-mode

- **Axis 1 (divergence) = HIGH.** BigQuery polling (set-mode)
  and Kafka stream consumption (record-mode) differ on all four
  factors: concurrency (request-response vs continuous pull),
  backpressure (BigQuery slot contention vs consumer lag), state
  machine (per-query vs per-partition offset + watermark per
  ADR-0024), liveness signal (query completion vs consumer
  heartbeat). Mismatch count = 4; threshold rule places this at
  HIGH.
- **Axis 2 (isolation) = HIGH.** Record-mode is v1 production
  with new failure modes (consumer crash, watermark stall per
  ADR-0024's idle-topic rejection, partition rebalance, broker
  outage); set-mode is operationally proven and serves the
  platform's only currently-shipping kind. Isolation is a hard
  requirement at the upper layers.

**Decision rule outcome: parallel runners, single binary.**

### Runner shape

Two runner loops, one per mode, in the same engine binary. Each
runner has its own worker pool, its own I/O layer, its own
panic recovery scope. **Shared upstream:** the loader, the
manifest reader, the HTTP trigger handler (per ADR-0014, which
only serves set-mode triggers and is unchanged by this ADR), the
alert publisher (per [ADR-0006](./0006-alert-routing-contract.md)),
the OTel observability pipeline, the result writer.
**Isolated downstream:** the two runner loops do not share
work-queues, worker pools, or panic recovery scopes. A panic in
the record-mode runner does not affect the set-mode runner.

```
engine/cmd/dq-engine/main.go
├── shared upstream: loader, manifest reader, HTTP trigger handler, alerter, OTel, writer
├── runner.SetRunner            (set-mode loop)
│   ├── BigQuery query path (existing ADR-0007 contract)
│   └── set-mode worker pool
└── runner.RecordRunner         (record-mode loop)
    ├── Kafka consumer path (per ADR-0023 + ADR-0024)
    └── record-mode worker pool
```

**Failure isolation scope.** The "hard requirement for
isolation" scopes to the **upper layers** — runner loop, worker
pool, kind handler, I/O layer. **Writer-side coupling remains a
known v1 limitation:** the writer is shared upstream, so a
writer-queue saturation event (BigQuery insertion lag) affects
both runners. Resolving writer-side isolation (separate write
queues / retry budgets per mode) is deferred to a follow-up if
operational signal motivates it.

### Result-write schema extension

ADR-0003's `dq_executions` table gains a new required column:

| Column | Type | Required | Description |
|---|---|---|---|
| `mode` | string (enum: `set`, `record`) | yes | The evaluation mode. Set on every row written by either runner. Joins to the rule's `mode:` field at the manifest version active when the run was identified. |

The column is **additive** under ADR-0001's compatibility
contract (additive-within-major); ADR-0003's append-only,
composite-primary-key, partition-by-date semantics are
unchanged. The `dq_executions_current` lazy view (per ADR-0003
§2) does not require modification — it projects all columns,
including the new `mode`. Consumers that already query
`dq_executions_current` for set-mode results automatically see
record-mode rows once they appear, with the new `mode` column
available for filtering.

**Cross-mode `attempt_id` namespace safety.** ADR-0002 commits
`attempt_id` as a per-`(execution_id, attempt)` sequence; the
attempt-id namespace is per-execution-id. Since `execution_id`
already differs across modes (distinct `entity` values
targeting distinct substrates; distinct `trigger_source` values
per ADR-0024), cross-mode attempt-id collision is structurally
impossible without further provision.

`dq_check_results` is unchanged. The per-check-per-window
aggregation produces one row per check per window per attempt,
identical in shape to set-mode's per-check-per-window output.

### Within-window aggregation seam

The seam lives at the **kind handler boundary**, not at the
runner loop or the writer. The record-mode runner, when a
window closes per ADR-0024's watermark trigger:

1. Reads the per-window batch of records from the Kafka
   consumer.
2. Invokes the kind handler (per ADR-0022's catalog dispatch
   model) with the batch.
3. The handler aggregates per-record outcomes into one
   `CheckResult` per check.
4. The runner writes one `dq_check_results` row per check.
5. Per-record violations land in evidence (an evidence-field
   structure inside the `CheckResult`; specific shape is the
   handler's responsibility, bounded by ADR-0003's
   evidence-retention policy).

The aggregation **function** — how N per-record outcomes map
to a single pass/fail/error/degraded check result per
[ADR-0004](./0004-failure-scope.md)'s status policy — is the
domain of the next Phase β decision (B0-S6 / forthcoming
ADR-0026). This ADR commits only the seam location: aggregation
happens **inside the kind handler**, before the runner writes;
the handler returns one result per check.

### HTTP trigger handler stays set-mode-only

[ADR-0014](./0014-trigger-handler-contract.md) commits the HTTP
trigger handler at `engine/internal/api/`. Record-mode is
event-driven — consumer triggers window closes per ADR-0024's
watermark — so no per-evaluation HTTP trigger is needed. The
handler is unchanged. If a future need arises (e.g.,
operator-rerun for a record-mode window range), a new HTTP
handler is added under the existing API surface; ADR-0014 is
not reopened.

### Lint cross-checks unchanged

This ADR adds **no new lint cross-checks**. Runner shape and
write-schema extension are runtime concerns, not rule-authoring
concerns. Ten cross-checks total (per ADRs 0021–0024) remain
the lint surface.

---

## Consequences

1. **P4 is retired.** The Wave-S launch ADR's fourth locked
   premise is satisfied. After this ADR's promotion, Wave-S has
   no remaining locked-premise commitments; all four premises
   (P1, P2, P3, P4) are realised in concrete schema, catalog,
   source, window, runner-shape, and aggregation-seam
   decisions.

2. **The objective criterion becomes a reusable platform
   artefact.** Future runner-shape questions (a third mode
   added; a mode split; modes merged) re-run the two-axis
   assessment against the same decision table and tie-breaker.
   The criterion outlives this ADR and is the documented
   procedure for any future runner-shape decision.

3. **The engine binary layout commits two parallel runner loops
   in one process.** One `engine/cmd/dq-engine/main.go`;
   `runner.SetRunner` and `runner.RecordRunner` as separate
   loops with separate worker pools and per-loop panic recovery,
   sharing the loader / manifest reader / HTTP trigger handler /
   alerter / OTel exporter / writer. Implementation lands in the
   combined implementation commit that closes ADRs 0021–0024
   alongside this ADR.

4. **Failure isolation scopes to the upper layers.** Runner
   loop, worker pool, kind handler, and I/O layer are isolated
   per-mode. **Writer-side coupling remains a known v1
   limitation** — writer-queue saturation propagates across
   modes. Resolving writer-side isolation is deferred to a
   follow-up if operational signal motivates it. This is honest
   about what v1 delivers; it is not a regression from the
   isolation goal.

5. **ADR-0003 gains an additive `mode` column on
   `dq_executions`.** The extension is additive under ADR-0001's
   compatibility contract; ADR-0003's append-only,
   composite-primary-key, partition-by-date semantics are
   unchanged. The `dq_executions_current` lazy view requires no
   modification. ADR-0003's set-oriented scope note remains in
   place; this ADR holds the record-mode extension.

6. **The within-window aggregation seam is at the kind handler
   boundary.** B0-S6 (failure scope aggregated, forthcoming
   ADR-0026) commits the aggregation function knowing the seam
   is at the kind handler. B0-S6 cannot move the seam without
   revisiting this ADR; B0-S6's policy fills the shape this
   ADR commits.

7. **B0-S7 inherits the per-runner shape for cost guardrails.**
   Cost guardrails bind throughput and lag budgets per runner.
   Record-mode lag is consumer-group lag tracked by the record
   runner; set-mode cost is BigQuery slot consumption tracked
   by the set runner. Per-runner isolation means budgets can be
   enforced and signalled per mode independently — except at
   the writer, where shared queue saturation must be addressed
   by the writer-isolation follow-up flagged in §4 above.

8. **The HTTP trigger handler (ADR-0014) stays set-mode-only.**
   ADR-0014 is unchanged. Record-mode operator-rerun (replay of
   a Kafka offset range with a captured window range), if and
   when committed, lands as a new HTTP handler under the
   existing API surface; ADR-0014 is not reopened.

9. **No new lint cross-checks.** Ten cross-checks total (per
   ADRs 0021–0024) remain the lint surface. Runner shape and
   write-schema extension are runtime, not rule-authoring,
   concerns.

10. **The criterion's tie-breaker biases conservatively.** When
    axis assignment is genuinely ambiguous, the tie-breaker
    defaults to parallel runners. The asymmetry of costs
    (bounded for unnecessary isolation; unbounded for
    insufficient isolation) justifies the default. Future
    runner-shape questions inherit this bias.

11. **Combined-commit pacing across ADRs 0021–0025.** The
    runtime artefacts committed by ADRs 0021–0025 (schema v2
    with `mode` + `params` + `source` + `window`, catalog file
    pair, ten lint cross-checks, dispatcher startup invariant,
    Kafka emulator + ADR-0010 amendment, two runner loops, the
    `mode` column on `dq_executions`, env-constants removal,
    loader rejection-path removal, atomic `customer.yaml`
    migration) all land together in a single combined
    implementation commit. If any artefact ships
    incrementally first, the schema-version-bump contingencies
    from ADR-0022 §C-B0S2.2 and ADR-0023 §C-B0S3.1 apply.

---

## Notes

- The combined implementation commit picks specific operational
  details that this ADR does not commit: how `runner.mode = set
  | record` attribute labelling is attached to OTel emissions
  (or whether mode is encoded in metric names); how per-runner
  panic recovery is organised across goroutines; whether
  writer-side backpressure signals are surfaced back to the
  runners independently or only via shared queue saturation.
  These are operator-pacing choices documented at the
  combined-commit time.
- Migration of existing `dq_executions` rows (backfill to
  `mode = 'set'` for rows written before this ADR) happens at
  the combined implementation commit. Whether the backfill is a
  one-shot SQL migration or done via a DEFAULT clause plus a
  follow-up backfill job is an implementation detail picked at
  commit time.
- The criterion's factor list for Axis 1 (concurrency
  primitives, backpressure shape, state machine, liveness
  signal) is open for additive extension. A future mode that
  surfaces a fifth dimension (e.g., per-record vs per-batch
  retry semantics) extends the factor list; the threshold rule
  re-counts mismatches with the expanded list.
- The forthcoming ADR-0010 amendment (Kafka substrate-matrix
  row, flagged by ADR-0023 §C-B0S3.3) may land before or after
  this ADR's promotion. If the amendment lands first at a
  number that this ADR's expected slot would occupy, this
  ADR's number shifts forward per ADR-0020 §Decision (Per-item
  ADR numbering). The shift propagates to any subsequent
  forthcoming Phase β ADRs (B0-S6, B0-S7) and is documented at
  the operator's pacing.
