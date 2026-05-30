<!-- path: studies/decisions/2026-05-30-b3-metric-emission-slice-scope.md -->

# B3-4 — Metric Emission Slice Scope

## Metadata

- Date: 2026-05-30
- Status: draft
- Decision-log row: B3-4 (tooling family)
- Promotion target: [`docs/adr/0055-metric-emission-slice-scope.md`](../../docs/adr/0055-metric-emission-slice-scope.md)
- Critique rounds: pending (this is the pre-critique draft)

---

## Context

[ADR-0039](../../docs/adr/0039-dashboard-contract.md) §"Metric
contract" committed an eight-metric Prometheus-compatible inventory
on the engine's `/metrics` endpoint, with named labels grounded in
the substrate ADRs (`status` per ADR-0003; `result` per ADR-0004;
`mode` per ADR-0021; `error_class` per ADR-0007; `trigger_source`
per ADR-0002). ADR-0039 §"Baseline-dashboard implementation —
deferred" + Consequence 2 explicitly deferred emission to a
"Phase-4c follow-up" while preserving the contract surface.

[ADR-0045](../../docs/adr/0045-baseline-dashboard-substrate.md) +
B2-24 shipped `deploy/dashboards/baseline.json`. Five panels: panels
1–3 query BigQuery against `dq_executions_current` /
`dq_check_results` and light up immediately; **panels 4–5 query
Prometheus and render "no data" until metric emission lands** per
the `description` field on each panel.

### What is emitted today vs what is pending

The inventory below cross-references each ADR-0039-committed metric
against the engine source tree. "Emitted" means a live emission
call site in non-test code exists today; "pending" means the
contract surface is committed but no emission code calls it.

| Metric | Status today | Producing surface |
|---|---|---|
| `dq_runs_total` | pending | runner terminal-row write (`engine/internal/runner/runner.go`) |
| `dq_checks_evaluated_total` | pending | runner per-check evaluation loop |
| `dq_run_duration_seconds` | pending | runner started_at → completed_at delta |
| `dq_check_duration_seconds` | pending | runner per-check evaluator |
| `dq_bytes_scanned` | pending | runner per-check evidence aggregation |
| `dq_loader_refresh_failures_total` | pending | loader refresh path (`engine/internal/loader/`) |
| `dq_queue_depth` | pending — **not engine-side** | external scheduler per [ADR-0033](../../docs/adr/0033-scheduler-catchup-behavior.md) §"External scheduler + advisory `schedule` field + per-env catchup horizon" |
| `dq_scheduler_triggers_managed` | pending — **not engine-side** | external scheduler per ADR-0033 |

The engine binary itself emits **zero** metrics today.
`engine/internal/api/server.go:45-50` registers only
`/v1/trigger`, `/healthz`, and `/readyz` — no `/metrics` route.
The runtime log channel is the only observability channel emitting
signals (per `engine/internal/runner/runner.go:15-27`'s package
doc, which carries the deferral comment ADR-0039 §Consequence 2
pointed to).

The Prometheus client library is not a direct dependency of
`engine/go.mod` — only `prometheus/client_model` appears as a
transitive of the BigQuery SDK chain. Likewise
`go.opentelemetry.io/otel/metric` is transitive only (BigQuery SDK
chain). So the emission slice both introduces a new direct
dependency AND wires the first metric call sites.

### Eligibility under ADR-0049 §(a)

Required per
[`.claude/playbooks/post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md)
step 2 for a B3-N entry; all four conditions must hold.

| # | Condition | Resolution |
|---|---|---|
| 1 | P-B3.1 — expands not rewrites | **Passes.** Slice satisfies the ADR-0039 contract surface where today nothing emits; no contract column / metric / label is changed. |
| 2 | P-B3.4 — in-scope family (kind / capability mode / tooling extensions) | **Borderline.** Routes through the tooling-extensions family via ADR-0049 §"Per-family scope" → "Tooling extensions" → "Captures … the engine dispatcher, and adjacent tooling that extend contract coverage without changing the contract shape." Engine-runtime emission is adjacent-tooling-shaped at this reading but stretches the family boundary past the lint-extension canonical example. Surfaced explicitly for operator ratification per [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5 §"Operator-side responsibilities" — author-equals-reviewer circularity (ADR-0051 §Consequence 7) means `/critique` cannot ratify on its own. |
| 3 | P-B3.2 — conforms to ADR-0020/0021/0022/0023 envelope | **Passes.** Slice honors `mode` as a label per ADR-0021 (mode-as-primitive); no envelope ADR is reshaped. |
| 4 | Additive-maintenance threshold — materially novel vs incremental | **Passes.** Introduces a direct external library dependency on a metrics client, a new HTTP route (`/metrics`), a new operator-facing failure class (cardinality / scrape pressure beyond ADR-0010's substrate-collector limit), and crosses the platform from one-channel (log only) to two-channel (log + metric) observability. |

Conditions 1, 3, 4 clear cleanly. Condition 2's borderline reading
is the only D0 surface; it carries forward to the promotion ADR as
**new contribution proposed here, requires review** (R5).

The temporal-classification test (B2 = pre-shipping against
in-flight wave; B3 = post-shipping against closed wave per ADR-0049
§(a)) reads to B3 — Wave 3 closed 2026-05-23; ADR-0039 was promoted
2026-05-26 (post-Wave-3) with the deferral marker pointing at
"Phase-4c", a phase that closed when Wave 3 did.

---

## Decision Drivers

- **DD-1 — Contract is binding; emission is the platform satisfying
  it.** ADR-0039 promised a stable consumer surface; today the
  platform does not honor the promise on the metric channel. Two
  shipped consumers (dashboard panels 4–5) read "no data". This is a
  visible gap.
- **DD-2 — Scheduler-side metrics are structurally separate.** Two of
  the eight ADR-0039 metrics (`dq_queue_depth`,
  `dq_scheduler_triggers_managed`) describe the **external**
  scheduler per ADR-0033 §"External scheduler". The engine binary
  cannot emit them without re-opening ADR-0033's external-scheduler
  posture. Any slice that promises to light panel 5 from engine code
  alone misrepresents what the engine can observe.
- **DD-3 — Library / route choice is the load-bearing structural
  decision; emission code is mechanical once the boundary is set.**
  Picking the Prometheus client library + the `/metrics` route
  shape + the per-package emitter convention (analogous to
  `engine/internal/logging/`'s component-attr convention per
  ADR-0043) are the contract-shaped choices. Subsequent per-metric
  emission PRs become near-mechanical applications of that
  boundary.
- **DD-4 — Minimal slice should leave the cardinality posture
  untouched.** ADR-0039 §"Cardinality posture" defers a numeric
  cardinality ceiling until "concrete cardinality-pressure signal".
  The first-emission slice should respect that deferral — emit the
  labels as committed, no relabeling, no aggregation.

---

## Considered Options

Three options for the minimal first-emission slice. Each scopes the
**first PR** that lands metric code; subsequent slices add metrics
incrementally.

### Option A — Library + `/metrics` route only; zero metrics emitted

Pick a Prometheus client library, add it as a direct dependency,
register a new `/metrics` route in `engine/internal/api/server.go`
serving the default Go runtime metrics only, and commit the
per-package emitter convention. No ADR-0039 metric emits.

- **Lights** neither panel 4 nor panel 5 (both still render "no
  data").
- **Strength.** Maximally minimal; sets the structural boundary in
  one reviewable PR. Subsequent slices add one metric at a time.
- **Weakness.** Does nothing visible against the user-facing gap.
  Reviewers cannot evaluate the library + route + convention
  choice end-to-end against a real metric.

### Option B — Engine-runtime metrics (six of eight), `/metrics` route

Pick the library, register `/metrics`, commit the per-package
emitter convention, and wire the six engine-runtime metrics that
the engine binary can directly observe:

1. `dq_runs_total` — runner terminal-row write.
2. `dq_checks_evaluated_total` — runner per-check loop.
3. `dq_run_duration_seconds` — runner started_at → completed_at.
4. `dq_check_duration_seconds` — runner per-check.
5. `dq_bytes_scanned` — runner per-check (from evidence
   aggregation; the bytes_scanned sub-field is undocumented per
   ADR-0039 OQ-3, so the gauge reports zero when absent rather
   than dropping the emission).
6. `dq_loader_refresh_failures_total` — loader refresh-path
   failure classifier per ADR-0007.

- **Lights** panel 4 ("alerting volume per entity") immediately
  via `dq_runs_total{status!="success"}`. Panel 5 remains dark
  pending the scheduler-side decision.
- **Strength.** Maximum visible impact reachable from engine code
  alone; closes 6 of 8 inventory gaps; honors ADR-0033's external-
  scheduler posture (the engine never claims to know what it
  cannot observe).
- **Weakness.** Larger first PR than Option A. Six metrics' worth
  of label-cardinality risk lands in one slice.

### Option C — Engine-runtime metrics + engine-side scheduler proxy

Option B plus a proxy for the two scheduler-side metrics, computed
from `dq_executions` table state (e.g., `dq_queue_depth{state="running"}`
counts terminal-pending execution rows; `dq_scheduler_triggers_managed`
counts distinct entities with recent triggers).

- **Lights** both panel 4 and panel 5.
- **Strength.** Closes the visible "no data" gap on every deferred
  panel in one slice.
- **Weakness.** Re-defines what the two metrics mean. ADR-0039
  committed `dq_queue_depth` as "Count of runs the scheduler
  currently tracks" — a proxy from `dq_executions` reports
  engine-observed run state, not scheduler-tracked state. That is
  a contract drift, not an emission. Honoring the slice would
  amend ADR-0039's label-source rule for those two metrics, which
  fails ADR-0049 §(a) Condition 1 (expands not rewrites — this is
  rewrite). Out of B3 scope; routes to **amendment** per ADR-0049
  §(a).

---

## Recommendation

**Option B.** It is the largest slice that holds inside the B3
envelope: it expands the platform's emission coverage from zero to
six of eight committed metrics, it respects ADR-0033's
external-scheduler posture (panel 5's `dq_queue_depth` and
`dq_scheduler_triggers_managed` cannot be honestly emitted from
engine code; the slice does not pretend otherwise), and it satisfies
the user-facing motivation by lighting panel 4 in one merge.

Panel 5 is scoped to a **separate decision** (Open Question §OQ-1
below). That decision either lands as an ADR-0033 amendment
authorizing engine-side proxies for the two metrics, or commits a
scheduler-binary instrumentation slice as its own B3-N / B2 row.
Neither belongs here.

Option A is rejected as not visible enough to verify the structural
choices end-to-end; Option C is rejected as a contract rewrite
disguised as an emission.

### What this study commits

- The eight-metric inventory state table above as the canonical
  pre-slice baseline.
- The minimal-slice scope as Option B: six engine-runtime metrics
  + `/metrics` route + per-package emitter convention.
- The deferral of panel 5 to a separate decision (OQ-1).
- The structural choices the slice's ADR must commit: which
  metrics client library; the `/metrics` route registration site;
  the per-package emitter convention; the test surface for the
  emission code; the cardinality posture (continues ADR-0039's
  no-numeric-ceiling deferral until the first scrape-pressure
  signal).

### What this study does NOT commit

- The library choice itself. Two reasonable candidates exist in
  the commodity-environment space named by R5
  (`prometheus/client_golang` and an `otel/metric` exporter
  configured with Prometheus output). Selection belongs to the
  promotion ADR after a same-session comparison; this scoping
  study only commits that one of them is picked.
- Any per-metric label cardinality ceiling.
- The per-package emitter naming convention's exact shape
  (mirrors `engine/internal/logging/`'s `component` attr
  convention but the parallel needs an emitter-side review).
- The emission code itself. Per the user's framing and R4 (one
  topic per session), the actual slice is a follow-on session
  whose B-row classification is set by the ADR that promotes
  this study.

---

## Consequences

1. **B3-4 reaches `resolved-study` with eligibility ratification
   pending for Condition 2.** The D0 borderline (engine-runtime
   emission as ADR-0049 tooling family) is carried forward to the
   promotion ADR as new-contribution-requires-review per R5.

2. **The promotion ADR closes B3-4 and commits the four structural
   choices listed above** (library, route, convention, test
   surface), plus the cardinality-posture continuation. Promotion
   target is `docs/adr/0055-metric-emission-slice-scope.md`
   (provisional number; operator reserves at promotion time per
   [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
   Clause 7).

3. **The emission code itself ships as a separate session** under
   the new ADR's authority. B-row classification (B2-style
   implementation slice landing under a closed ADR, vs a fresh
   B3-N for the wiring decisions) is set by the promotion ADR.

4. **Panel 5 stays dark after the slice lands.** This is honest —
   the engine binary cannot observe scheduler state per ADR-0033.
   The dashboard's panel-5 "no data" rendering remains accurate
   until OQ-1 resolves.

5. **ADR-0039, ADR-0033, ADR-0007, ADR-0010, ADR-0043, ADR-0045
   are preserved.** The slice satisfies ADR-0039's contract
   surface where today nothing emits; it does not reshape any
   committed contract. ADR-0043's per-package logging convention
   is the structural reference for the per-package emitter
   convention.

6. **`engine/internal/runner/runner.go:15-27`'s deferral comment
   becomes load-bearing once the slice lands.** That doc-comment
   currently says "queued for a Phase-4c follow-up that wires an
   otelslog-style slog handler". After the emission slice merges,
   the comment is updated in the same PR to reflect the chosen
   implementation shape.

---

## Open Questions

- **OQ-1: Panel 5 lighting (scheduler-side metrics).**
  `dq_queue_depth` and `dq_scheduler_triggers_managed` describe
  the external scheduler per ADR-0033. Lighting panel 5 needs a
  separate decision: either an ADR-0033 amendment authorizing
  engine-side proxies (with redefined label semantics), or a
  scheduler-binary instrumentation slice once the platform owns a
  scheduler binary. **Out-of-scope for current cycle** — different
  topic per R4; routes to its own B-row when concrete demand for
  panel-5 lighting surfaces.

- **OQ-2: Tracing channel (`ADR-0007 CC14` three-channel
  commitment).** ADR-0007 CC14 committed log + metric + **span**
  emission. This scoping study covers the metric channel only;
  span emission is a separate scope. **Out-of-scope for current
  cycle** — `runner.go:15-27` couples them in a deferral comment,
  but the span side is a different external dependency choice and
  belongs to its own study.

- **OQ-3: Metric library choice (Prometheus client vs OTel exporter).**
  Two commodity-environment libraries can produce
  Prometheus-compatible output. The choice is deferred to the
  promotion ADR's same-session comparison so the trade-offs
  (direct vs indirect dependency depth; OTel migration headroom;
  test ergonomics) are evaluated against working code rather
  than against speculation. **Out-of-scope for current scoping
  study** — by design, this study commits only that the choice
  is made at promotion time.

- **OQ-4: Cardinality ceiling.** ADR-0039 §"Cardinality posture"
  deferred a numeric per-metric ceiling to "a future ADR … if
  cardinality growth produces ingest failures in production".
  The first-emission slice respects that deferral; if scrape
  pressure surfaces post-emission, a separate ADR commits the
  ceiling. **Out-of-scope for current cycle** — continues the
  ADR-0039 deferral verbatim; no decision changes here.

- **OQ-5: Per-package emitter convention shape.** The convention
  mirrors `engine/internal/logging/`'s `component` attr per
  ADR-0043 in spirit, but the metric-side parallel (how each
  package obtains its named counter / histogram / gauge handles)
  is reviewer-load-bearing in the promotion ADR. **Out-of-scope
  for current cycle** — this study scopes that the convention is
  committed at promotion time, not its exact shape.

---

## Promotion target

[`docs/adr/0055-metric-emission-slice-scope.md`](../../docs/adr/0055-metric-emission-slice-scope.md)
(provisional; operator reserves at promotion time per ADR-0051
Clause 7).
