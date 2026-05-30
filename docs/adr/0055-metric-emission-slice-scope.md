<!-- path: docs/adr/0055-metric-emission-slice-scope.md -->

# ADR-0055 — Metric Emission Slice Scope

- **Status:** accepted
- **Date:** 2026-05-30

---

## Context

[ADR-0039](./0039-dashboard-contract.md) §"Metric contract" committed
an eight-metric Prometheus-compatible inventory served from the
`/metrics` endpoint, with named labels grounded in the substrate
ADRs (`status` per [ADR-0003](./0003-result-write-model.md);
`result` per [ADR-0004](./0004-failure-scope.md); `mode` per
[ADR-0021](./0021-mode-as-primitive.md); `error_class` per
[ADR-0007](./0007-loader-scheduler-retry-failure-semantics.md);
`trigger_source` per
[ADR-0002](./0002-run-identity-and-idempotency.md)). ADR-0039
§"Baseline-dashboard implementation — deferred" + Consequence 2
deferred the emission code itself, preserving the contract surface.

[ADR-0045](./0045-baseline-dashboard-substrate.md) shipped
`deploy/dashboards/baseline.json` against that contract; B2-33
brought Prometheus + Grafana into `docker-compose.yml`. Panels 1–3
of the baseline dashboard query BigQuery and light up immediately;
panels 4–5 query Prometheus and render "no data" because the
engine binary emits zero metrics today. The remaining gap is the
engine-side emission code itself plus the `/metrics` HTTP route.

This ADR is the **promotion of B3-4** under the post-Wave-3
evolutionary lane committed by
[ADR-0049](./0049-b3-evolutionary-launch.md). B3-4 surfaced an
ADR-0049 §(a) Condition 2 borderline (engine-runtime emission as
the Tooling-extensions family stretches past the lint-extension
canonical example). The operator ratified the borderline mid-PR
per `CONTRIBUTING.md` Flow 5 §"Operator-side responsibilities".
The reading inherits ADR-0051 Clause 1's adjacent-tooling
precedent without opening any new expansive reading beyond it.
Per `CLAUDE.md` R5 + A7 of the `adr-writing` skill, the reading
is recorded here as **new contribution requiring review** and is
reviewed in this ADR.

This ADR also lands its implementation slice in the same PR per
an **operator-authorized R4 scope collapse**, precedent
[ADR-0054](./0054-engine-image-registry-amendment.md) §Notes
("operator-authorized R4 scope collapse at promotion time").

The principles bearing on the decision are **P1** (rules remain
declarative — emission code reads from the same execution rows
the contract already commits; no escape hatch is introduced),
**P2** (deterministic behavior — same trigger + same data state
produce the same metric increments at every emission site),
**P3** (ownership is explicit — every metric carries an `entity`
or `error_class` label that maps back to an owner per
[ADR-0006](./0006-alert-routing-contract.md)), **P4** (cost is
first-class — the cardinality posture is preserved without
re-litigation), and **P5** (evolution is contract-driven — the
slice expands ADR-0039's coverage without changing its shape).

---

## Decision

The slice is committed in six clauses (library, route, emitter
convention, runner emissions, loader emissions, cardinality
posture), plus a Notes block that records the R4 collapse and
the ratified Condition-2 reading per R5 + A7.

### Clause 1 — Library: `github.com/prometheus/client_golang`

The engine binary depends on `github.com/prometheus/client_golang`
as a direct module-level dependency for metric emission. The
library is the canonical Go producer of the Prometheus exposition
format the existing dashboard and scrape stack already consume
(per ADR-0045 + B2-33). An OpenTelemetry-exporter alternative
configured for Prometheus output was considered and rejected —
the additional SDK indirection layer is speculative against the
current scope (no consumer of this slice asks for OTLP push or
multi-exporter routing today; P1, P2, P5 favor the simpler
direct producer).

Future OTLP-push migration remains available as a separate
decision if concrete demand surfaces; this ADR does not foreclose
it.

### Clause 2 — Route: `GET /metrics` on the shared mux

The trigger handler's HTTP server
(`engine/internal/api/server.go`) mounts `GET /metrics` on the
same `http.ServeMux` that already serves `/v1/trigger`,
`/healthz`, and `/readyz`. The route handler is
`promhttp.HandlerFor` bound to the engine's prometheus.Registry.
Single listener on `cfg.HTTPAddr` per
[ADR-0014](./0014-trigger-handler-contract.md) §1; no separate
metrics listener is introduced.

This honors ADR-0039 §"Endpoint" verbatim — "the engine binary
serves a Prometheus-compatible scrape endpoint at `/metrics` on
its primary HTTP port".

### Clause 3 — Per-package emitter convention

A new package `engine/internal/metrics` owns the prometheus
Registry and the per-package Metrics structs. Each consuming
package (runner, loader) takes its own typed Metrics struct via
its `Config` (constructor injection), matching the prevailing
shape for `Logger`, `Publisher`, and `Evaluator`. The convention
mirrors [ADR-0043](./0043-logging-contract-specifics.md)'s
per-package component-attr convention in spirit: each package
owns a stable handle set; the engine binary wires them once at
startup.

Concrete shape committed:

- `metrics.Registry` — wraps `*prometheus.Registry`; provides
  `Handler() http.Handler` for the route.
- `metrics.RunnerMetrics` — holds the five runner-side handles
  (Clause 4).
- `metrics.LoaderMetrics` — holds the one loader-side handle
  (Clause 5).
- `metrics.NoopRunnerMetrics()` /
  `metrics.NoopLoaderMetrics()` — return Metrics structs
  registered against a throwaway registry, safe for tests that
  do not assert emission.

The package is the central inventory matching ADR-0039
§"Metric contract" verbatim. Consumer packages depend on
`engine/internal/metrics`; the metrics package does not import
runner or loader internals.

### Clause 4 — Runner-side emissions (five of six)

The runner package emits five of the eight ADR-0039 metrics, at
the call sites named below. Each emission honors ADR-0039's
label set verbatim.

| Metric | Emission site (`engine/internal/runner/runner.go`) | Labels |
|---|---|---|
| `dq_runs_total` | `writeTerminalRow` and `writePreCheckErrorRow` after the durable write returns | `entity`, `status`, `trigger_source`, `mode` |
| `dq_run_duration_seconds` | Same sites as above; histogram observes `completed_at - started_at` | `entity`, `status`, `mode` |
| `dq_checks_evaluated_total` | Inside the per-check loop, after `WriteCheckResultRow` returns | `entity`, `check_id`, `result`, `mode` |
| `dq_check_duration_seconds` | Same as above; histogram observes per-check evaluator duration | `entity`, `check_id`, `mode` |
| `dq_bytes_scanned` | Same as above; gauge `Set` from the evaluator's evidence summary | `entity`, `check_id` |

For `dq_bytes_scanned` specifically, when the evidence summary's
`bytes_scanned` sub-field is absent (ADR-0039 OQ-3 — the
sub-field is undocumented), the gauge is set to **zero**. The
choice resolves OQ-6 from the originating study: zero preserves
time-series continuity (the gauge series stays defined) and is
distinguishable from a real positive scan value via the
operator's panel. Alternative resolutions (skip emission;
emit NaN) were considered and rejected — skip introduces gaps
in the series that confuse range queries; NaN is non-portable
across collector versions.

Emission happens **after** the durable write returns in every
case, so a Store-write failure cannot produce a metric without
its backing row. The reverse pattern (emit-then-write) is
expressly out of scope.

### Clause 5 — Loader-side emission (one metric)

The loader package emits `dq_loader_refresh_failures_total` on
every error return from `Loader.Refresh`. The `error_class`
label follows a five-value closed-but-additive enum concretized
from ADR-0007 §1's enumerated failure surface (pointer read,
manifest body fetch, hash verification, compatibility-contract,
PAT-1). The mapping:

| `error_class` value | Triggered by |
|---|---|
| `pointer_read` | `readPointer` failure (`store.ReadObject`, JSON unmarshal, `pointer_version != 1`, `stripSha256Prefix` on pointer) |
| `body_fetch` | `fetchAndVerify` `store.ReadObject` failure on the content-addressed body |
| `hash_mismatch` | sha256 mismatch inside `fetchAndVerify` |
| `parse_error` | `json.Unmarshal` failure on the manifest body |
| `compatibility_contract` | `runContractChecks` failure (engine_version, schema_versions_present, manifest_version, PAT-1 fail-fast cases) |

ADR-0007 §12 commits the three-channel emission obligation
without enumerating the closed `error_class` enum. The five
values above are **new contribution requiring review** per R5
+ A7 — derived from ADR-0007 §1's failure surface but enumerated
here for the first time. Future loader failure modes extend the
enum additively per ADR-0039's evolution rules.

The binding from `Loader.Refresh`'s error returns to the
`error_class` label is by **sentinel errors + `errors.Is`**, not
by matching wrapped-error message text. The loader exports three
sentinels (`ErrHashMismatch`, `ErrBodyFetch`, `ErrParseError`);
the classifier dispatches on `errors.Is` against each, and
everything else routes to `compatibility_contract` by
elimination. The sentinel discipline keeps the binding resilient
to wording changes in any of the loader's `fmt.Errorf` wrapping
sites — refactoring an error message does not drift the metric
label.

`Loader.Load` (startup-mode) does not emit. Startup failures are
unconditionally fatal per ADR-0007 §1; the existing log + exit
channels carry the operational signal, and a startup-emission
would not survive long enough to be useful.

### Clause 6 — Cardinality posture preserved

ADR-0039 §"Cardinality posture" deferred a numeric per-metric
cardinality ceiling to a future ADR "if cardinality growth
produces ingest failures in production". This ADR preserves
that deferral as written — no numeric ceiling is committed,
no per-metric label-pruning rule is committed. The clause
exists so the preservation is auditable.

The label sources are unchanged from ADR-0039 §"Label value
sources"; the per-emission-site code reads them from the same
substrate fields ADR-0039 already commits.

---

## Consequences

1. **B3-4 closes at `resolved-adr` via this ADR + the
   implementation slice landing in the same PR (operator-
   authorized R4 scope collapse, precedent ADR-0054
   §Notes).** The decision-log row updates accordingly.

2. **The engine binary becomes a two-channel observer** (log
   + metric). The span channel from ADR-0007 §12 stays a
   separate scope (OQ-2 from the originating study; lands in
   a follow-on session).

3. **Panel 4 of `deploy/dashboards/baseline.json` lights up
   immediately on next deployment.** `dq_runs_total` emits
   on every terminal-row write; the panel's
   `sum by (entity) (rate(dq_runs_total{status!="success"}[5m]))`
   expression resolves against real data once Prometheus
   scrapes the engine.

4. **Panel 5 stays dark.** ADR-0033 fixes the scheduler as
   external; `dq_queue_depth` and
   `dq_scheduler_triggers_managed` describe scheduler-tracked
   state the engine binary cannot observe. Lighting panel 5
   routes through a separate decision (ADR-0033 amendment
   authorizing engine-side proxies OR a scheduler-binary
   instrumentation slice) per the originating study OQ-1.

5. **`github.com/prometheus/client_golang` becomes a direct
   module dependency.** The library was previously transitive
   only via the BigQuery SDK chain. The direct dependency
   surfaces in `engine/go.mod` and is owned by the engine
   workspace; the existing build, lint, and test gates pick
   it up without further wiring.

6. **`engine/internal/metrics` is the central inventory.** Any
   future addition to ADR-0039's metric set lands in this
   package first; consuming packages take handles via their
   Config. The package replaces the "metric emission deferred"
   doc-comment in `engine/internal/runner/runner.go` package
   doc with a pointer to this ADR and to the metrics package.

7. **The `error_class` enum for
   `dq_loader_refresh_failures_total` is concretized here.**
   Five values (`pointer_read`, `body_fetch`, `hash_mismatch`,
   `parse_error`, `compatibility_contract`) cover ADR-0007 §1's
   enumerated failure surface. Future loader failure modes
   extend the enum additively per ADR-0039 §"Evolution rules".
   The label binding uses sentinel errors + `errors.Is` (the
   loader exports `ErrHashMismatch`, `ErrBodyFetch`,
   `ErrParseError`); the binding is not coupled to any
   `fmt.Errorf` message text, so wording changes in the loader's
   error returns cannot silently drift the metric label.

8. **`dq_bytes_scanned` reports zero when the
   `evidence_summary.bytes_scanned` sub-field is absent**
   (OQ-6 from the originating study, resolved). The gauge
   series stays defined; operators can distinguish from real
   scan values via the panel range. ADR-0039 OQ-3
   (`evidence_summary` field-level inventory) remains a
   separate deferred item.

9. **Cardinality posture is preserved.** ADR-0039 §"Cardinality
   posture" continues to govern; no numeric ceiling is
   committed here; no per-metric label-pruning rule is
   committed. If scrape-pressure surfaces in production, a
   separate ADR commits the ceiling.

10. **Test surface lands with the slice.** `engine/internal/metrics`
    carries a unit test asserting Registry construction +
    Handler-served exposition format. `engine/internal/api`
    gains a `/metrics` integration test asserting the route
    returns 200 + Prometheus content-type + the expected
    metric family names. Runner and loader tests gain assertions
    on per-metric increments via the `prometheus/testutil`
    helpers.

11. **ADR-0039, ADR-0033, ADR-0007, ADR-0010, ADR-0043,
    ADR-0045, ADR-0049, ADR-0051 are preserved.** The slice
    satisfies ADR-0039's contract surface where today nothing
    emits; it does not reshape any committed contract.
    ADR-0043's per-package component-attr convention is the
    structural reference for the per-package emitter
    convention.

12. **Three follow-on scopes remain explicitly deferred:**
    span/tracing channel from ADR-0007 §12 (OQ-2); panel-5
    lighting (OQ-1); cardinality ceiling (continues ADR-0039's
    deferral; OQ-4 from the originating study). None of these
    blocks any panel that this ADR commits to lighting.

---

## Notes

**Operator-authorized R4 scope collapse.** The originating
study (B3-4) scoped the slice; the implementation code was
flagged as a separate session per R4. The operator authorized
collapsing both into a single PR at promotion time, precedent
[ADR-0054](./0054-engine-image-registry-amendment.md) §Notes.
The collapse rationale: the four structural choices this ADR
commits (library, route, emitter convention, per-metric
emission sites) are reviewer-load-bearing precisely because
they are reviewable against working code. Splitting the ADR
from the code would force the ADR reviewer to evaluate
prose-only choices that the code would either validate or
falsify in the next session.

**Condition 2 borderline ratification carry-forward.** Per R5
+ A7 of the `adr-writing` skill, the operator-ratified
ADR-0049 §(a) Condition 2 reading admitting engine-runtime
emission as the Tooling-extensions family is recorded here as
**new contribution requiring review**. The reading does not
open any new expansive reading beyond ADR-0051 Clause 1's
adjacent-tooling precedent. The mid-PR ratification disposition
itself is also a new contribution (prior B3-N entries ratified
at-merge); the precedent stands for future B3-N entries to
follow either path.

**Critique rounds.** This ADR's Decision survived one
`/critique` round (0 blocking / 2 important / 5 minor; 2
important applied — sentinel-based `error_class` binding noted
in Clause 5 + Consequence 7, and Clause 1's rejection rationale
rewritten without external-principle naming; 5 minor deferred
under the two-round cap). The originating B3-4 study survived
two `/critique` rounds (1 = 0 blocking / 3 important / 5 minor;
2 = ratification trailer). The implementation code in this PR
is self-verified against AC-W3-3 + AC-W3-7 per ADR-0052 §6.4
row 6 close-gates and ADR-0048 §"Skip" path for code-only
`/critique` rounds.

**No ADR-0033 reopening.** Panel 5 stays dark by design. The
external-scheduler posture committed by ADR-0033 is preserved
verbatim; this ADR does not authorize any engine-side proxy
for scheduler-tracked state.
