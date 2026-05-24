<!-- path: studies/decisions/2026-05-24-b0-s5-aggregation-and-runner-shape.md -->

# B0-S5 — Aggregation and Runner Shape

## Metadata

- **B-item reference:** B0-S5 (Wave-S Phase β, item 2 of 4) — resolves P4
- **Status:** resolved-study (Wave-S, B0-S5; one critique round; round 1 cleared with no blocking findings)
- **Last updated:** 2026-05-24
- **Upstream resolved:** [ADR-0020](../../docs/adr/0020-wave-s-launch.md)
  (Wave-S launch — locks P1–P4; this study retires P4's deferral);
  [ADR-0021](../../docs/adr/0021-mode-as-primitive.md) (mode primitive
  at schema level); [ADR-0022](../../docs/adr/0022-kind-catalog.md)
  (kind catalog with `source_mode` per kind);
  [ADR-0023](../../docs/adr/0023-sources-schema.md) (Kafka substrate +
  source declaration); [ADR-0024](../../docs/adr/0024-window-semantics.md)
  (tumbling watermark-bounded windows + record-mode `execution_id`
  shape);
  [ADR-0002](../../docs/adr/0002-run-identity-and-idempotency.md)
  (set-mode run identity); [ADR-0003](../../docs/adr/0003-result-write-model.md)
  (set-mode write model — `dq_executions` + `dq_check_results`,
  both append-only);
  [ADR-0007](../../docs/adr/0007-loader-scheduler-retry-failure-semantics.md)
  (set-mode loader and scheduler);
  [ADR-0014](../../docs/adr/0014-trigger-handler-contract.md) (HTTP
  trigger handler — set-mode).
- **Downstream open:** B0-S6 (failure scope aggregated — consumes the
  within-window aggregation seam committed here); B0-S7 (cost
  guardrails — binds to the runner-shape decision, particularly to
  worker-pool dimensions and per-mode lag budgets).
- **Promotion target:** `docs/adr/0025-aggregation-and-runner-shape.md`
  (subject to ADR-0020 §Decision (Per-item ADR numbering); `0025`
  assumes ADR-0024 lands as 0024 and no unrelated promotion or
  ADR-0010 amendment intervenes).
- **Loop discipline:** same as B0-S1–S4 — `/resolve-b0` study →
  `/critique` (≥1 round) → operator acceptance → `/promote-to-adr`.
- **Significance:** **B0-S5 retires P4's deferral.** ADR-0020 §Decision
  (Locked architectural premises) §P4 explicitly committed P4's
  resolution to B0-S5 with an objective decision criterion as the
  deliverable. This study produces the criterion **before** picking
  the runner shape, then applies the criterion to the current
  platform state.

---

## Context

The Wave-S foundational triplet committed mode-as-primitive
(ADR-0021), the kind catalog (ADR-0022), and the Kafka source
substrate with tumbling watermark-bounded windows (ADR-0023 +
ADR-0024). What remains for record-mode runtime to ship is the
**engine binary's shape**: does one runner switch on mode per
evaluation, or do two parallel runners — one set, one record —
share the engine binary while running independent evaluation
loops?

ADR-0020 §Decision §B0-S5 commits the scope:

> Decides whether the engine runs **one unified runner** that
> switches on mode per evaluation, or **two parallel runners**
> (one set, one record); the **objective decision criterion** P4
> commits to, drafted during the B0-S5 study and likely combining
> (a) operational blast-radius of a runner outage in each shape,
> (b) the cost of duplicated lifecycle plumbing (loader,
> scheduler, observability emission) under parallel runners, (c)
> whether record-mode aggregations can reuse the set-mode
> `dq_executions` / `dq_check_results` schema (with a `mode`
> column) or need a parallel write path. The criterion is decided
> **before** the runner shape; the criterion itself is the
> deliverable that satisfies P4's deferral.

The current engine state is set-mode-only:

- `engine/cmd/dq-engine/main.go` is the single engine binary
  entrypoint.
- `engine/internal/runner/` is the runner package, called by
  `engine/internal/eval/` to dispatch checks per `CheckSpec.Kind`
  (per ADR-0007's loader contract, ADR-0014's HTTP trigger
  handler, and ADR-0002's `execution_id` computation).
- The loader (per ADR-0007) loads manifests and rules from the
  object store, with mode-field dispatch per ADR-0021 — but the
  loader currently rejects `mode: record` rules pending B0-S3's
  partial-Wave-S gate (which is now met in spec, per ADR-0023
  §C-B0S3.6, but unwiring the rejection waits for the combined
  implementation commit).
- The write model (per ADR-0003) commits `dq_executions` +
  `dq_check_results` as both append-only with composite primary
  keys; ADR-0003 carries the 2026-05-23 scope note declaring
  set-oriented capability.

Four sub-decisions live inside B0-S5's scope:

1. **The objective criterion** — a decision procedure that any
   future runner-shape question (third mode added; mode split
   further; mode merged) can re-run. The criterion is the
   deliverable that satisfies P4.
2. **The pick for record-mode** — applying the criterion to the
   current platform state (one set mode, one new record mode).
3. **Result-write schema reuse** — does record-mode reuse the
   set-mode `dq_executions` + `dq_check_results` tables (additive
   extension of ADR-0003 with a `mode` column) or commit a
   parallel write path?
4. **Within-window aggregation seam** — how N per-record check
   results aggregate into one `dq_check_results` row per check
   per window. The aggregation policy itself is B0-S6; B0-S5
   commits the seam location.

Platform principles bearing on the design: **P1** (declarative —
the runner-shape decision must not leak into the declarative DSL
beyond what's already exposed via mode + kind + source); **P2
mirror, foundation 01 §"Determinism"** (whichever shape is
picked, same input → same `execution_id` set, which both shapes
satisfy because ADR-0024 already committed the `execution_id`
formula); **P3** (capability derived from mode — the runner
shape is the runtime realisation of the mode-as-primitive choice
from ADR-0021); **P5** (evolution contract-driven — ADR-0003,
ADR-0007, ADR-0014 extensions live under ADR-0001's compatibility
contract).

---

## Decision Drivers

- **DD-S5.1** — **The criterion is the deliverable.** P4 is
  satisfied not by a runner-shape pick but by an **objective
  decision criterion** that future runner-shape questions can
  apply. The criterion must be (a) named axes with named scales,
  (b) a decision rule mapping axis assessments to a runner-shape
  outcome, (c) reproducible by a future operator without
  consulting B0-S5's author.

- **DD-S5.2** — **R3 honoured on set-mode contracts.** ADR-0003
  (write model), ADR-0007 (loader/scheduler), and ADR-0014
  (trigger handler) carry set-oriented scope notes. Record-mode
  extensions are additive — none of those ADRs is reopened. The
  write-model extension (a `mode` column on `dq_executions`) is
  additive within ADR-0003's append-only contract under ADR-0001
  (additive-within-major). The loader extension is the v2 schema
  dispatch + record-mode acceptance (already committed by
  ADR-0021 / ADR-0023). The HTTP trigger handler is not extended
  for record-mode in this ADR — record-mode is event-driven (no
  per-evaluation trigger); the existing handler stays
  set-mode-only.

- **DD-S5.3** — **Operational learnings from Wave 3 inform the
  criterion.** The set-mode engine is operationally proven
  (Wave 3 close on 2026-05-23). Record-mode is brand new and
  high-risk: new substrate (Kafka), new I/O pattern (continuous
  consumption vs scheduled poll), new failure modes (consumer
  lag, watermark stall, partition rebalance). Failure isolation
  from the proven set-mode runtime is operationally desirable.

- **DD-S5.4** — **Substrate-shape divergence is HIGH.** BigQuery
  polling and Kafka stream consumption are fundamentally
  different I/O models: different concurrency primitives, different
  backpressure shapes, different liveness signals, different
  state machines. A unified runner that switches on mode per
  evaluation would carry runtime-level mode-switches at every
  layer — concurrency control, backpressure handling, lifecycle,
  observability emission. The mode-switching cost is
  proportional to the divergence.

- **DD-S5.5** — **Combined-implementation-commit pacing.** ADRs
  0021–0024 are committed in markdown but the combined
  implementation commit (which lands all four ADRs' runtime
  artefacts together — per ADR-0022 §C-B0S2.2's path-(a)
  preference) has not yet shipped. Whichever shape B0-S5 picks
  must be wire-compatible with that combined commit. The pick
  shapes the engine binary at `engine/cmd/dq-engine/main.go` and
  the runner package layout under `engine/internal/runner/` /
  `engine/internal/eval/`.

- **DD-S5.6** — **Result-write schema reuse minimises query-
  layer divergence.** Set-mode `dq_executions` and
  `dq_check_results` are the query surface for dashboards,
  alerting, and reporting. Adding a parallel record-mode table
  pair would force every consumer (Looker, Grafana, alerting
  rules) to query both tables and union them. A `mode` column on
  the existing tables is additive at the storage layer and
  zero-impact at the query layer (consumers ignore the column
  if they only care about set-mode; consumers that need both
  modes filter or group by it).

- **DD-S5.7** — **Within-window aggregation is B0-S6's policy
  domain.** B0-S5 commits the **seam** — the layer at which the
  record-mode runner produces one `dq_check_results` row per
  check per window, having aggregated N per-record outcomes
  inside that window. The specific aggregation function (count
  thresholds, severity mapping, status enum projection) is
  B0-S6's responsibility. B0-S5 ensures the seam is at the
  per-window boundary and that the aggregation runs inside the
  kind handler (per ADR-0022's catalog dispatch model), not in
  the write layer.

- **DD-S5.8** — **Failure isolation at the runner-loop and I/O
  layers is a hard requirement for v1.** A record-mode failure
  (consumer crash, watermark stall, Kafka broker outage) at the
  consumer / runner loop / kind handler layers must not
  propagate to the set-mode runner. Set-mode is operationally
  proven and serves the platform's only currently-shipping kind
  (`set.row_count_positive` on `customer.yaml`); record-mode is
  v1 production. **Writer-side coupling remains a known v1
  limitation.** Per C-B0S5.3 the writer is a shared upstream
  resource and OQ-B0S5.3 (Shared backpressure signal) flags
  writer-queue saturation as a cross-mode coupling that v1
  does not resolve. The isolation requirement therefore scopes
  to the **upper layers** — runner loop, worker pool, I/O — not
  to the writer; resolving writer-side isolation (separate
  write queues / retry budgets per mode) is deferred to a
  follow-up if operational signal motivates it.
  *(New contribution proposed here, requires review.)*

---

## The Objective Criterion (P4 deliverable)

The criterion below is the deliverable that satisfies P4's
deferral. It is named axes, named scales, and a decision rule.
Any future runner-shape question (a third mode added; a mode
split further; modes merged) re-runs this criterion.

### Axes

**Axis 1 — Substrate-shape divergence.** How different is the
new mode's I/O pattern from the existing modes that would share
the runner?

- **LOW** — same I/O pattern as an existing mode (same
  concurrency primitives, same backpressure shape, same state
  machine). Example: a new mode that targets a different SQL
  warehouse but uses the same polling model as the existing
  set-mode.
- **MEDIUM** — same family of I/O pattern (e.g., both polling-
  based) but different operational lifecycle (different
  authentication, different retry shapes, different per-request
  bounds). Example: a SQL warehouse that requires session
  authentication where set-mode uses service-identity OIDC.
- **HIGH** — fundamentally different I/O pattern (different
  concurrency primitives, different backpressure, different
  state machine, different failure modes). Example: continuous
  stream consumption vs scheduled polling.

**Threshold rule for Axis 1 (mismatch count).** Count mismatches
across four factors: **(i)** concurrency primitives (e.g.,
request-response vs continuous pull), **(ii)** backpressure
shape (e.g., slot contention vs consumer lag), **(iii)** state
machine (e.g., per-query vs per-partition offset + watermark),
**(iv)** liveness signal (e.g., query completion vs consumer
heartbeat). **LOW** = 0 mismatches; **MEDIUM** = 1–2 mismatches;
**HIGH** = 3–4 mismatches. The factor list is open for additive
extension if a future mode surfaces a fifth axis dimension; the
threshold rule preserves reproducibility — two analysts counting
the same mismatches reach the same level.

**Axis 2 — Failure-isolation requirement.** How critical is it
that one mode's failure does not propagate to another?

- **LOW** — mode-cross failure is operationally tolerable.
  Example: both modes are well-proven and operationally similar;
  cross-failure rarely occurs and is easy to recover.
- **HIGH** — mode failure must not affect other modes. Example:
  a new high-risk mode added to a proven runtime; or a
  compliance-driven separation between modes.

### Decision rule

| Axis 1 (divergence) | Axis 2 (isolation) | Runner shape |
|---|---|---|
| LOW | LOW | **Unified runner** (single loop dispatches by mode) |
| LOW | HIGH | **Parallel runners, single binary** (separate loops, shared upstream) |
| MEDIUM | LOW | **Unified runner** with mode-specific worker pools |
| MEDIUM | HIGH | **Parallel runners, single binary** |
| HIGH | LOW | **Parallel runners, single binary** |
| HIGH | HIGH | **Parallel runners, single binary** (consider OS-process parallelism only if OS-level isolation is operationally required) |

The single-binary form is the default for parallel runners.
**OS-process parallelism** (separate engine binaries per mode) is
reserved for cases where OS-level isolation is operationally
required — e.g., when an upstream policy forbids shared OS
resources, or when one mode's CPU/memory characteristics
catastrophically interfere with another's. This study commits
the procedure for that escalation but does not commit any current
need for it.

**Tie-breaker rule.** When axis assignment is ambiguous — when an
analyst could reasonably defend two adjacent levels for either
axis — default to the **more-isolation-friendly outcome**
(parallel runners, single binary). The cost of unnecessary
isolation is bounded (one extra runner loop and worker pool); the
cost of insufficient isolation is unbounded (a production
incident with cross-mode propagation). The asymmetry of those
costs justifies the conservative default.

### Application to record-mode

- **Axis 1 (divergence) = HIGH.** BigQuery polling (set-mode)
  and Kafka stream consumption (record-mode) differ at every
  runtime layer: concurrency (request-response vs continuous
  pull), backpressure (BigQuery slot contention vs consumer
  lag), state machine (per-window query vs per-partition offset
  + watermark), liveness signal (query completion vs consumer
  heartbeat).
- **Axis 2 (isolation) = HIGH.** Record-mode is v1 production
  with new failure modes (consumer crash, watermark stall,
  partition rebalance, broker outage); set-mode is
  operationally proven and serves currently-shipping production
  rules. Isolation is a hard requirement (DD-S5.8).

**Decision rule outcome: parallel runners, single binary.**

---

## Considered Options

The four options below differ on **runner shape**. All four
assume the objective criterion is committed as above; the
variation is which point on the criterion's matrix applies to
record-mode given current operational signal.

### Option A — Unified runner with mode-switches per evaluation

**Shape.** A single runner loop dispatches by mode at each
evaluation: pick rule from manifest → check `rule.mode` →
dispatch to set-mode I/O path or record-mode I/O path. One
worker pool serves both modes. One `engine/cmd/dq-engine/main.go`
entrypoint, one runner package, mode-dispatched I/O layers.

**Cost.** Substrate-shape divergence (Axis 1 = HIGH) forces
mode-switches at every runtime layer: connection pooling for
BigQuery vs Kafka clients, backpressure handling for HTTP-style
slot contention vs stream consumer lag, lifecycle for per-query
vs per-partition state. Failure isolation (Axis 2 = HIGH) is
not achievable — one runner = one failure domain. Record-mode's
new failure modes (broker outage, watermark stall) can
propagate to set-mode by exhausting shared worker-pool capacity
or by triggering shared panic recovery that affects in-flight
set-mode work.

**Verdict.** Rejected. Fails the criterion at both axes.

### Option B — Parallel runners, single binary (recommended)

**Shape.** Two runner loops, one per mode, in the same engine
binary. Each runner has its own worker pool, its own I/O layer,
its own lifecycle. **Shared upstream:** the loader (ADR-0007),
the manifest reader (ADR-0005), the HTTP trigger handler
(ADR-0014, which only serves set-mode triggers and is unchanged
by this ADR), the alert publisher (ADR-0006), the OTel
observability pipeline, the result writer (ADR-0003 extended
with a `mode` column). **Isolated downstream:** the two runner
loops do not share work-queues, worker pools, or panic recovery
scopes. A panic in the record-mode runner does not affect the
set-mode runner.

```
engine/cmd/dq-engine/main.go
├── shared upstream: loader, manifest reader, HTTP trigger handler, alerter, OTel, writer
├── runner.SetRunner            (set-mode loop)
│   ├── BigQuery query path (existing ADR-0007 contract)
│   └── set-mode worker pool
└── runner.RecordRunner         (record-mode loop)
    ├── Kafka consumer path (new, per ADR-0023 + ADR-0024)
    └── record-mode worker pool
```

**Cost.** Two runner loops vs one; two worker pools vs one. The
duplication is bounded — shared upstream captures most of the
plumbing. A single deploy unit (one binary, one container, one
Kubernetes deployment per ADR-0019). Per-runner failure
isolation via separate worker pools + per-loop panic recovery.

**Verdict.** Recommended. Matches the criterion's
HIGH-divergence + HIGH-isolation outcome.

### Option C — Parallel runners, separate binaries (OS-process parallelism)

**Shape.** Two engine binaries: `dq-engine-set` and
`dq-engine-record`. Each is its own process, its own container,
its own Kubernetes deployment. Two manifests in Kustomize
overlays under `deploy/overlays/` (per ADR-0019). True
OS-level failure isolation — a process crash affects only one
mode.

**Cost.** Doubles the deploy surface: two container images, two
Kubernetes deployments, two OTel pipelines, two log streams,
two manifests under each environment overlay. The upstream
shared resources (loader, manifest reader, OTel exporter) are
duplicated at the OS level — each process maintains its own
state. Operational complexity doubles without a current
operational signal motivating it.

**Verdict.** Rejected for v1. Reserved per the criterion's
escalation path when OS-level isolation becomes operationally
required.

### Option D — Unified runner with mode-specific worker pools

**Shape.** A single runner loop dispatches by mode but maintains
separate worker pools per mode. The dispatch happens before
work enters a pool, so worker-pool exhaustion in one mode does
not consume the other's capacity. One binary, one runner code
path, two worker pools.

**Cost.** Partial failure isolation (worker-pool-level only).
Other shared concerns (the runner's main loop, panic recovery,
shared backpressure signal) still link the two modes. The
substrate-shape divergence (Axis 1 = HIGH) still forces
mode-switches inside the runner loop. Marginally better than
Option A on isolation but does not fully match Option B's
isolation.

**Verdict.** Rejected. The criterion's HIGH/HIGH outcome
mandates parallel runners; mode-specific worker pools alone do
not constitute parallel runners.

---

## Recommendation

**Pick Option B — Parallel runners, single binary.**

Rationale, tied directly to the criterion and drivers:

- **DD-S5.1 / The criterion.** Axis 1 = HIGH (substrate
  divergence between BigQuery polling and Kafka streaming);
  Axis 2 = HIGH (record-mode is v1 production with new failure
  modes, isolation is non-negotiable per DD-S5.8). The
  criterion's HIGH/HIGH row in the decision rule yields
  "parallel runners, single binary".
- **DD-S5.2 (R3 on set-mode contracts).** No set-mode contract
  is reopened. ADR-0003 gains an additive `mode` column on
  `dq_executions` (additive within ADR-0001's compatibility
  contract); ADR-0007's set-mode loader contracts are unchanged
  (the v2 schema dispatch was already committed by ADR-0021);
  ADR-0014's HTTP trigger handler is unchanged (record-mode is
  event-driven, no per-evaluation trigger).
- **DD-S5.3 (operational learning).** Set-mode is proven;
  record-mode is unproven. Parallel runners protect the proven
  runtime from the unproven one until record-mode accumulates
  its own operational signal.
- **DD-S5.4 (substrate divergence).** HIGH divergence forces
  parallel I/O layers anyway; sharing a runner loop would
  add mode-switching cost at every layer with no benefit.
- **DD-S5.5 (combined-commit pacing).** The combined
  implementation commit lands two runner loops side by side
  inside `engine/cmd/dq-engine/main.go`, alongside the schema
  v2 / catalog / loader / Kafka emulator artefacts from ADRs
  0021–0024.
- **DD-S5.6 (write-schema reuse).** Both runners write to the
  same `dq_executions` and `dq_check_results` tables, with a
  new `mode` column on `dq_executions` distinguishing the
  rows. Query layer is unchanged for set-only consumers;
  mode-aware consumers filter or group by `mode`.
- **DD-S5.7 (aggregation seam).** The record-mode runner
  invokes the kind handler (per ADR-0022's catalog dispatch)
  with the per-window batch of records. The handler aggregates
  per-record outcomes into one check result per check per
  window before returning to the runner. The runner writes one
  `dq_check_results` row per check; per-record violations land
  in evidence per ADR-0003. The aggregation **function** is
  B0-S6's policy; the aggregation **seam** is the kind handler
  boundary.
- **DD-S5.8 (isolation hard requirement).** Per-runner worker
  pools + per-loop panic recovery satisfy v1's isolation
  requirement without requiring OS-process parallelism.

### Result-write schema extension

ADR-0003's `dq_executions` table gains a new column:

| Column | Type | Required | Description |
|---|---|---|---|
| `mode` | string (enum: `set`, `record`) | yes | The evaluation mode. Set on every row written by either runner. Joins to the rule's `mode:` field at the manifest version active when the run was identified. |

The column is **additive** under ADR-0001's compatibility
contract (additive-within-major); ADR-0003's append-only,
composite-primary-key, partition-by-date semantics are unchanged.
The `dq_executions_current` lazy view (per ADR-0003 §2) does
not need modification — it projects all columns, including the
new `mode`.

**Cross-mode `attempt_id` namespace safety.** ADR-0002 commits
`attempt_id` as a per-`(execution_id, attempt)` sequence; the
attempt-id namespace is per-execution-id. Since `execution_id`
already differs across modes (distinct `entity` values targeting
distinct substrates; distinct `trigger_source` values per
ADR-0024), cross-mode attempt-id collision is structurally
impossible without further provision.

`dq_check_results` is unchanged. The per-check-per-window
aggregation produces one row per check per window per attempt,
identical in shape to set-mode's per-check-per-window output.

### Within-window aggregation seam

The seam lives at the **kind handler boundary**, not at the
runner loop or the writer. The record-mode runner, when a
window closes:

1. Reads the per-window batch of records from the Kafka consumer
   (per ADR-0024's window-close trigger).
2. Invokes the kind handler (per ADR-0022's catalog dispatch
   model) with the batch.
3. The handler aggregates per-record outcomes into one
   `CheckResult` per check (the same `CheckResult` type
   `engine/internal/runner` already returns for set-mode).
4. The runner writes one `dq_check_results` row per check.
5. Per-record violations land in evidence (an evidence-field
   structure inside the `CheckResult`; specific shape is the
   handler's responsibility, bounded by ADR-0003's
   evidence-retention policy).

The aggregation **function** — how N per-record outcomes map
to a single pass/fail/error/degraded check result — is B0-S6's
policy. B0-S5 commits only that aggregation happens **inside
the kind handler**, before the runner writes; the handler
returns one result per check.

### Lint cross-checks added

This study adds **no new lint cross-checks**. Runner shape and
write-schema extension are runtime concerns, not rule-authoring
concerns. The ten lint cross-checks committed by ADRs 0021,
0022, 0023, 0024 cover the rule-authoring surface; B0-S5's
decisions surface no new rule-shape constraints.

### One-line decision summary table

| Decision | Outcome |
|---|---|
| Objective criterion (P4 deliverable) | Two-axis matrix (substrate-shape divergence × failure-isolation) with named scales and a decision rule; committed in §"The Objective Criterion" above |
| Runner shape for record-mode | Parallel runners, single binary (HIGH/HIGH outcome of criterion applied to current platform) |
| Engine binary layout | One `engine/cmd/dq-engine/main.go` entrypoint; `runner.SetRunner` + `runner.RecordRunner` as parallel loops sharing upstream plumbing |
| Result-write schema | Same `dq_executions` + `dq_check_results` tables with additive `mode` column on `dq_executions` per ADR-0001 (no parallel write path) |
| Within-window aggregation seam | Kind handler boundary (aggregation happens inside the handler before the runner writes) |
| Aggregation function | Deferred to B0-S6 (B0-S5 commits the seam, not the policy) |
| OS-process parallelism | Reserved per criterion's escalation path; not committed in v1 |
| Lint cross-checks added | None (zero) |

---

## Consequences

### Cross-cutting consequences

- **C-B0S5.1** — **P4 is retired.** The Wave-S launch ADR's
  fourth locked premise ("execution unified-vs-parallel is
  deferred to B0-S5, which must produce an objective decision
  criterion before closing") is satisfied. The criterion is
  committed in §"The Objective Criterion"; the runner-shape
  outcome (parallel, single binary) is the criterion's
  application to the current platform. *(New contribution
  proposed here, requires review.)*

- **C-B0S5.2** — **ADR-0003 gains an additive `mode` column on
  `dq_executions`.** The extension is additive under
  ADR-0001's compatibility contract; ADR-0003's append-only,
  composite-primary-key, partition-by-date semantics are
  unchanged. The `dq_executions_current` lazy view does not
  require modification. *(New contribution proposed here,
  requires review.)*

- **C-B0S5.3** — **The engine binary layout commits two
  parallel runner loops in one process.** One
  `engine/cmd/dq-engine/main.go`; `runner.SetRunner` and
  `runner.RecordRunner` as separate loops with separate worker
  pools, sharing the loader / manifest reader / HTTP trigger
  handler / alerter / OTel exporter / writer. Per-runner panic
  recovery isolates failures; one runner's crash does not
  propagate to the other. Implementation lands in the combined
  implementation commit that closes ADRs 0021–0024 alongside
  this ADR's eventual promotion to ADR-0025.

- **C-B0S5.4** — **The within-window aggregation seam is at the
  kind handler boundary.** The record-mode runner invokes the
  kind handler with the per-window batch; the handler returns
  one `CheckResult` per check, with per-record violations in
  evidence. The aggregation function is B0-S6's policy; B0-S5
  commits only the seam location. *(New contribution proposed
  here, requires review.)*

- **C-B0S5.5** — **B0-S6 inherits the seam contract.** B0-S6
  (failure scope aggregated) commits the aggregation function
  (count thresholds, severity mapping, status enum projection)
  knowing the seam is at the kind handler. B0-S6 cannot move
  the seam without revisiting B0-S5; B0-S6's policy fills the
  shape B0-S5 commits.

- **C-B0S5.6** — **B0-S7 inherits the per-runner shape for
  cost guardrails.** B0-S7 (cost guardrails) binds throughput
  and lag budgets per runner. Record-mode lag is consumer-group
  lag tracked by the record runner; set-mode cost is BigQuery
  slot consumption tracked by the set runner. The per-runner
  isolation means budgets can be enforced and signalled per
  mode independently.

- **C-B0S5.7** — **The HTTP trigger handler (ADR-0014) stays
  set-mode-only.** Record-mode is event-driven (consumer
  triggers window closes per ADR-0024's watermark); no per-
  evaluation HTTP trigger is needed. The handler at
  `engine/internal/api/` is unchanged. If a future need arises
  (e.g., operator-rerun for a record-mode window range), a new
  HTTP handler is added under the existing API surface; ADR-0014
  is not reopened.

- **C-B0S5.8** — **The `dq_executions_current` lazy view's
  semantics extend to record-mode without modification.** The
  view (per ADR-0003 §2) projects per-(entity, ruleset_version,
  window) the most-recent attempt's row; with the additive
  `mode` column on `dq_executions`, the projection works
  identically for record-mode rows. Consumers that already
  query `dq_executions_current` for set-mode results
  automatically see record-mode rows once they appear, with
  the new `mode` column available for filtering.

- **C-B0S5.9** — **The criterion is reusable.** Future
  runner-shape questions (a third mode added; a mode split
  further; modes merged into one) re-run the criterion's
  axis assessments and apply the decision rule. The criterion
  outlives this study and becomes part of the platform's
  decision-making toolkit. *(New contribution proposed here,
  requires review.)*

### Per-artefact consequences

- **`engine/internal/dsl/schema/v2.schema.json`** — unchanged
  by this ADR. The runner shape is invisible to the rule
  schema.

- **`rules/_schema/v2.schema.json`** — unchanged (byte-equal
  mirror).

- **`engine/cmd/dq-engine/main.go`** — extended to start two
  parallel runner loops (set + record) inside the same
  process. Shared upstream initialisation (loader, manifest
  reader, HTTP trigger handler, alerter, OTel, writer) happens
  once; each runner consumes its own portion. Implementation
  lands in the combined implementation commit.

- **`engine/internal/runner/`** — new types `SetRunner` and
  `RecordRunner` (or the existing `Runner` type is split into
  two). Each runner has its own worker pool, its own panic
  recovery scope, its own I/O layer (BigQuery query path for
  set; Kafka consumer for record). The shared types (`CheckSpec`,
  `Evaluation`, `CheckResult`) stay common.

- **`engine/internal/eval/`** — the dispatcher
  (`evaluator.go`) extends to dispatch record-mode kinds to
  record-mode handlers (e.g., `record_schema_conformance.go`
  per ADR-0022). The startup invariant from ADR-0022 §Decision
  (every catalog entry has a registered handler) extends to
  both runners — each runner validates its own subset of the
  catalog at boot, or the engine validates the full catalog
  once at boot and dispatches per-mode.

- **`dq_executions` schema (deployed; BigQuery)** — gains a
  required `mode` column (enum: `set`, `record`). Schema
  migration for the existing `dq_executions` table happens at
  the combined implementation commit. Existing rows (set-mode
  only) are backfilled with `mode = 'set'`. New rows from
  either runner carry their mode value.

- **`dq_check_results` schema** — unchanged. Per-check-per-
  window output shape is identical across modes.

- **`docs/adr/0003-result-write-model.md`** — scope-noted as
  set-oriented. The `mode` column addition is additive under
  ADR-0001's compatibility contract; ADR-0003's scope note is
  unchanged because this ADR (ADR-0025) holds the record-mode
  extension. ADR-0003 does not need amendment.

- **No changes to ADR-0007, ADR-0014, ADR-0001, ADR-0002,
  ADR-0006 contracts.** All set-mode contracts hold. ADR-0002's
  `trigger_source` enum extension (`stream-watermark` per
  ADR-0024) is unchanged by this ADR.

- **No new lint cross-checks.** The lint binary remains at ten
  cross-checks (the count after ADR-0024's promotion).

---

## Open Questions

- **OQ-B0S5.1** — **Per-runner OTel attribute labelling.**
  Observability emission is shared (one OTel pipeline) but each
  runner produces its own metrics, spans, and logs. Whether
  every emission carries an explicit `runner.mode = set | record`
  attribute (so dashboards and alerts can filter), or whether
  the engine's metric naming convention encodes the mode in the
  metric name (e.g., `dq_engine_set_evaluations_total` vs
  `dq_engine_record_evaluations_total`), is an observability
  design question. *Out of scope for current cycle;* the
  combined implementation commit picks one and documents it.
  The attribute-labelling approach **keeps the metric namespace
  flat and lets dashboards / alert rules filter on a typed
  attribute rather than parsing the metric name**, and is the
  default until operational signal motivates otherwise.

- **OQ-B0S5.2** — **Per-runner panic recovery scope.** Go's
  `recover()` only catches panics in the same goroutine. The
  per-runner panic recovery requires that each runner's worker
  pool has its own panic-catching wrapper around every goroutine
  it spawns. Whether the engine adopts a single panic-recovery
  utility shared between runners (with mode-aware error
  enrichment) or each runner ships its own panic recovery is a
  code-organisation question. *Out of scope for current cycle;*
  the combined implementation commit decides, with operator
  visibility into the choice.

- **OQ-B0S5.3** — **Shared backpressure signal.** Both runners
  share the result writer (BigQuery insertion). If BigQuery
  insertion lags, both runners are affected. Whether the
  writer surfaces a backpressure signal back to each runner
  (e.g., "writer is at 80% capacity, slow down") or each runner
  independently observes writer latency is an operational-
  feedback design question. *Out of scope for current cycle;*
  the v1 implementation may rely on writer-side queue
  saturation alone, with per-runner backpressure as a future
  enrichment if needed.

- **OQ-B0S5.4** — **Combined-commit ordering when ADR-0010
  amendment is interleaved.** ADR-0023 §C-B0S3.3 flagged a
  forthcoming ADR-0010 amendment for the Kafka substrate row.
  Whether that amendment lands before or after this ADR
  (ADR-0025) affects per-item numbering per ADR-0020 §Decision
  (Per-item ADR numbering). If the amendment lands first at
  ADR-0025 (this study's expected number), this study shifts to
  ADR-0026. *Out of scope for current cycle;* the operator
  decides amendment-vs-study order at promotion time.

- **OQ-B0S5.5** — **Future B0-S6 aggregation policy
  granularity.** B0-S5 commits the seam at the kind handler;
  B0-S6 commits the function. Whether B0-S6's aggregation
  function is per-kind (each catalog entry declares its own
  aggregation rule) or per-mode (one policy for all record-mode
  kinds) is B0-S6's question. *Defer to B0-S6;* B0-S5 supports
  either shape — the kind handler is free to implement any
  aggregation function the catalog declares.

- **OQ-B0S5.6** — **Migration of existing `dq_executions`
  rows.** The combined implementation commit backfills existing
  set-mode rows with `mode = 'set'`. Whether the backfill is a
  one-shot SQL migration or done via a DEFAULT clause + a
  follow-up backfill job is an implementation detail. *Out of
  scope for current cycle;* the combined implementation commit
  picks the simpler path that fits BigQuery's DDL semantics.

---

## Promotion target

**Target:** `docs/adr/0025-aggregation-and-runner-shape.md`.

This study promotes to **ADR-0025** once at least one round of
`/critique` has been accepted by the operator and any blocking
findings are addressed. ADR-0025 is the second Phase β ADR of
Wave-S (after ADR-0024's window-semantics promotion). Per
ADR-0020 §Decision (Per-item ADR numbering), the `0025` slot is
descriptive; if the forthcoming ADR-0010 amendment ADR (Kafka
row, flagged by ADR-0023 §C-B0S3.3) lands first, this study
promotes to ADR-0026 and the per-item slugs shift.

ADR-0025's promotion is the **P4-retirement event** — the final
locked premise from ADR-0020 is satisfied at this ADR's
promotion. After ADR-0025 promotes, Wave-S has no remaining
locked-premise commitments; all four premises (P1–P4) are
realised in concrete schema, catalog, source, runner-shape, and
aggregation-seam decisions.

The promotion commit lands the artefacts committed in
§Consequences above:

1. The objective criterion (rewritten as forward-only ADR prose;
   becomes a reusable platform artefact).
2. The runner-shape decision (parallel, single binary) and the
   engine binary layout (one entrypoint, two runner loops,
   shared upstream).
3. The result-write schema extension (additive `mode` column
   on `dq_executions`).
4. The within-window aggregation seam location (kind handler
   boundary).

Per R8, the future ADR-0025 will be rewritten from this study,
not linked back to it.
