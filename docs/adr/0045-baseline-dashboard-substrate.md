<!-- path: docs/adr/0045-baseline-dashboard-substrate.md -->

# ADR-0045 — Baseline Dashboard Substrate

- **Status:** accepted
- **Date:** 2026-05-27

---

## Context

[ADR-0039](./0039-dashboard-contract.md) commits the
consumer contract that sits on top of the platform's two
reporting surfaces: the BigQuery-backed tables
(`dq_executions_current`, `dq_check_results`) and the
Prometheus-compatible `/metrics` endpoint. The contract
names every guaranteed column, metric, label, enum, and
evolution rule downstream consumers may rely on across
engine versions.

What ADR-0039 explicitly **defers** is the platform's own
example consumer dashboard. Foundation 05 §"Dashboards"
described that example dashboard at the panel level — run
success rate over time, per-entity check pass/fail rates,
cost per entity per day, alerting volume per owner,
scheduler health — and committed it to ship "during
Wave 3". Wave 3 closed with no dashboard shipped because
the panels driven by `/metrics` cannot exist until the
metric-emission code lands (ADR-0039 §"Baseline-dashboard
implementation — deferred"). ADR-0039 registered the
baseline-dashboard implementation as a B2 follow-up
(B2-24) paced post-Phase-4c metric emission so the panels
can smoke-test end-to-end.

What this ADR commits is the **substrate** the
baseline-dashboard B2 slice runs against and the
**workspace placement** of the dashboard artifact. The
substrate choice constrains every future consumer
dashboard the platform itself ships; downstream teams
remain free to consume the ADR-0039 contract with any
substrate they prefer per ADR-0039 §"Consequences" #1.

Existing commitments this ADR builds on:

- [ADR-0010](./0010-substrate-posture.md) §3.2 commits the
  metrics-endpoint's existence; ADR-0039 commits the path
  (`/metrics`).
- [ADR-0019](./0019-infrastructure-tooling.md) commits
  Kustomize as the deployment-manifest tool and frames the
  `deploy/` workspace as the canonical placement for
  deployment-time artefacts.
- Foundation 05 §"Local development substrate posture"
  names "local Prometheus + Grafana + Jaeger via
  docker-compose, mirroring the production observability
  shape" as the precedented observability stack. Grafana
  is the commodity dashboard substrate the foundation
  doc anticipated; this ADR makes the choice load-bearing.
- [ADR-0034](./0034-local-testing-strategy.md) commits the
  six-tier test taxonomy. Dashboard correctness lives in
  the e2e-demo tier (foundation 05's "manifest publish →
  loader refresh → execution write → alert publish" plus
  "dashboard panel renders the captured row").

The principles bearing on the decision are **P3**
(ownership is explicit — operators own substrate
provisioning; the platform owns the artefact),
**P4** (cost is a first-class constraint — choosing a
substrate the platform doesn't have to host avoids a
new service-operation burden), and **P5** (evolution
must be contract-driven — the dashboard artefact carries
a versioned shape downstream consumers can pin to).

---

## Decision

### Substrate — Grafana-compatible JSON dashboards

The platform's example consumer dashboard ships as a
**Grafana-compatible JSON dashboard model**. Grafana is
the commodity observability substrate the foundation
doc named for local-development parity and is the
substrate this ADR commits.

Two operating properties of Grafana matter for the choice:

- **Multi-data-source.** Grafana queries BigQuery (for
  the SQL-table panels per ADR-0039 §"Table contract")
  and Prometheus (for the `/metrics` panels per ADR-0039
  §"Metric contract") in the same dashboard. The two
  ADR-0039 consumer surfaces light up under a single
  visualisation layer with no glue code.
- **Declarative JSON artefact.** A Grafana dashboard is
  a JSON document the platform version-controls in
  `deploy/dashboards/`. The artefact does not require
  Grafana to be running anywhere — operators with no
  Grafana instance import the JSON into whatever
  Grafana they later provision; operators with a
  different substrate can read the panel inventory and
  reproduce them in their preferred tool.

Alternative substrates considered:

- **Vendor-managed BI tools** (named generically per R5
  — large multi-tenant analytics platforms accessed via
  HTTPS dashboards). The platform would gain a hosted
  surface with no operator burden, at the cost of every
  consumer needing the same vendor account and a
  per-tenant configuration surface that lives outside
  the repo. Rejected — the platform's repo-as-source-of-
  truth posture (foundation 02 §"Single monorepo") would
  fragment.
- **Embedded HTML dashboard** served by the engine
  itself. Avoids the substrate question entirely at the
  cost of pulling rendering / charting code into the
  engine binary. Rejected on P4 grounds (engine code
  surface stays small; observability lives outside the
  engine per ADR-0010).
- **Notebook-as-dashboard** (Jupyter, Hex, etc.). The
  artefact would be code + queries in one file. The
  platform has no notebook substrate today and adding
  one to support a single dashboard is an outsized
  step. Rejected.

The Grafana-JSON choice does NOT mandate that operators
run Grafana. Operators are free to read the JSON's
panel inventory and reproduce the panels in their
preferred substrate — the JSON file is the contract
surface for the panel definitions; the rendering layer
is operator-pick. Operators wanting the lowest-friction
path provision Grafana, point it at the two data
sources, and import the JSON.

### Workspace placement — `deploy/dashboards/`

The dashboard JSON lives at
`deploy/dashboards/baseline.json` under the existing
`deploy/` workspace per ADR-0019. The placement choice:

- **Adjacent to other deployment-time artefacts.**
  Kustomize overlays under `deploy/overlays/` already
  scope deployment-state to per-environment surfaces.
  Dashboards are deployment-state by the same logic
  (an environment's "what's running" includes its
  observability surface).
- **No new top-level workspace.** A six-workspace
  `dashboards/` directory was considered and rejected
  on R3 grounds — the foundation 02 monorepo topology
  commits five workspaces; adding a sixth for one
  artefact today over-fits the structural cost to the
  current need. If a future expansion makes the
  per-entity dashboard surface large enough to warrant
  its own workspace, the move is additive and can be
  done as an ADR amendment with a single PR.

The directory's `README.md` carries the operator-facing
import guide; the JSON itself is the artefact.

### Panel inventory — the five foundation-05-committed panels

The baseline-dashboard B2 slice ships the JSON
covering exactly the five panels foundation 05
§"Dashboards" committed:

1. **Run success rate over time.** Bar / time-series
   chart against `dq_executions_current` grouped by
   `recorded_at` time-bucket; success-vs-non-success
   ratio per bucket.
2. **Per-entity check pass/fail rate.** Table against
   `dq_check_results` joined with `dq_executions` for
   the `entity` column; grouped by `(entity,
   check_id)` × `result`.
3. **Cost per entity per day.** Time-series chart
   against `dq_check_results` extracting
   `JSON_VALUE(evidence_summary, '$.bytes_scanned')`;
   grouped by `entity` × `DATE(executed_at)`.
4. **Alerting volume per owner.** Counter chart against
   the Prometheus-emitted `dq_runs_total` metric per
   ADR-0039 §"Metric contract", joined to
   `_owners.yaml`'s owner string at panel-query time.
5. **Scheduler health.** Two gauges against
   `dq_queue_depth` and `dq_scheduler_triggers_managed`
   per ADR-0039 §"Metric contract".

Panels 1-3 query the BigQuery tables and light up
immediately the moment any rule produces results.
Panels 4-5 query the `/metrics` endpoint and light up
when the metric emission code lands per ADR-0039
§"Baseline-dashboard implementation — deferred". Both
panel sets ship in the same JSON; deferred-emission
panels render "no data" until emission lands.

### Versioning

The dashboard JSON carries an internal `version` field
per Grafana's convention and is bumped manually each
time a panel changes. The repo's git history is the
authoritative version surface; the JSON's internal
field exists so a Grafana instance can detect a
re-import as an upgrade.

Engine-major bumps that would break the dashboard
(per ADR-0039 §"Evolution rules") trigger a
corresponding dashboard JSON update. The dashboard
artefact is engine-version-bound the same way any
other ADR-0039 consumer is; the migration discipline
is identical (additive within major; breaking change
requires major bump + ADR-0035 N-1+90-day window).

### Local-development integration — deferred

Foundation 05 §"Local development substrate posture"
named local Prometheus + Grafana + Jaeger as the
intended local-development stack. The current
`docker-compose.yml` ships the substrate emulators
(BigQuery, GCS, Pub/Sub, Kafka) but not the
observability stack. Adding Prometheus + Grafana
services to compose so the dashboard can be exercised
locally is a structurally clean follow-up to this ADR
but expands the local-substrate footprint
non-trivially.

The compose extension is registered as a new B2 row
(numbered at close-step assignment). The baseline
dashboard JSON ships today; operators with a Grafana
instance import it now; the compose extension makes
the local-development surface match.

### Why this does not reopen ADR-0039

ADR-0039 commits the consumer contract (column
inventory, metric names, label cardinality, evolution
rules) the dashboard reads. This ADR picks one
specific consumer of that contract — the platform's
own example dashboard — and commits its substrate +
workspace placement. ADR-0039's contract surface is
preserved.

### Why this does not reopen ADR-0010 or ADR-0019

ADR-0010 commits the metrics-endpoint as a Yes-row
substrate capability. This ADR cites the endpoint
without amending its commitments. ADR-0019 commits
Kustomize for `deploy/` workspace tooling; the
dashboard artefact is a sibling under `deploy/`
following the same workspace convention. Neither
ADR is amended.

### Why this does NOT commit operator-side Grafana provisioning

Operator-side Grafana provisioning (deployment YAMLs,
ConfigMap auto-discovery, data-source wiring) is
outside the platform's commitment surface. Each
operator's Grafana posture is their own decision —
some run Grafana as a CRD-managed Operator, some as
a Helm-installed chart, some as a hosted SaaS
instance. The platform commits the dashboard
artefact + the data-source contract; the rendering
layer is operator-pick.

---

## Consequences

1. **The baseline dashboard ships as a single Grafana
   JSON file at `deploy/dashboards/baseline.json`.**
   Five panels per foundation 05 §"Dashboards": run
   success rate, per-entity check pass/fail, cost per
   entity per day, alerting volume per owner, scheduler
   health. Panels 1-3 query BigQuery and light up
   immediately; panels 4-5 query Prometheus and light
   up when metric emission lands.

2. **No engine code change ships from this ADR.** The
   dashboard is a deployment-time artefact; the engine
   binary is unaffected.

3. **No operator-side Grafana provisioning is
   committed.** Operators import the JSON into their
   own Grafana instance (or substitute a different
   substrate that consumes the same data sources). The
   `deploy/dashboards/README.md` documents the import
   workflow.

4. **Workspace placement: `deploy/dashboards/`.** No
   new top-level workspace; the dashboard artefact is
   a sibling under the existing `deploy/` per ADR-0019.
   If the platform later ships multiple dashboards
   (per-entity, alerting-specialised, etc.) the
   directory grows without further structural changes.

5. **Grafana-JSON is the contract surface for the
   dashboard artefact.** Operators using a different
   substrate read the JSON's panel inventory and
   reproduce the panels in their tool. The JSON's
   query strings are the canonical panel queries
   against the ADR-0039 contract.

6. **Versioning aligns with ADR-0039.** The dashboard
   evolves under the same compatibility model: additive
   within engine-major; breaking changes require an
   engine-major bump and the ADR-0035 N-1 + 90-day
   migration window. The dashboard JSON's internal
   `version` field is bumped manually on each panel
   change.

7. **Deferred-emission panels are committed today even
   though they render "no data" today.** Panels 4-5
   ship against the ADR-0039 metric contract that
   metric-emission code (deferred per `engine/internal/runner/runner.go`
   lines 15-27) will eventually satisfy. Shipping the
   panels now means the dashboard light-up at
   emission-land time is a JSON-already-present, no-PR
   moment.

8. **B2-24 closes.** The decision-log B2-24 row moves
   to `resolved-adr` once this ADR ships alongside the
   dashboard JSON in the same PR. One new B2 row
   registers the docker-compose extension that adds
   Prometheus + Grafana services so the local-
   development experience can exercise the dashboard
   end-to-end (assigned at close-step).

9. **Three deferred items remain reserved:** local
   Grafana/Prometheus in docker-compose (B2 follow-up);
   per-entity / specialised dashboards beyond the
   foundation-05 baseline (future ADRs when concrete
   operator demand surfaces); documented field-level
   inventory for `evidence_summary` JSON sub-fields
   the cost panel reads (ADR-0039 OQ-3, still open —
   the cost panel uses `bytes_scanned` which is
   undocumented today; the panel's query string
   documents the dependency, but the field-level
   inventory ADR amendment is a separate slice).

10. **The platform's P3 + P4 + P5 commitments for
    dashboarding are now explicit.** P3 (ownership):
    the platform owns the artefact; the operator owns
    the rendering substrate. P4 (cost): no new service
    operation burden — the platform doesn't host
    Grafana; the artefact is a static JSON file. P5
    (contract-driven evolution): the JSON's panel
    definitions are versioned against ADR-0039's
    contract surface.

---

## Notes

- Grafana panel definitions in the v1 JSON use simple
  query strings; richer features (templated variables
  for entity/time-range, alert rules tied to panels,
  panel-level annotations from CI deploys) are not
  committed at v1. Operator iteration sharpens the
  panels as the dashboard is used in anger.
- The `evidence_summary.bytes_scanned` field consumed
  by the cost panel is currently undocumented per
  ADR-0039 OQ-3. The cost panel ships with the field
  named in its query string; if the engine's
  `bytes_scanned` emission shape changes in the
  future, the panel updates with it. A future ADR
  amendment formalises the inventory.
- Future expansions worth flagging if they surface
  operational signal: a per-entity dashboard
  parameterised by the `entity` label; an
  alerting-specialised dashboard focusing on
  `_owners.yaml` channels; a cost-specialised
  dashboard for the BigQuery bytes-scanned trend.
  None are committed today.
