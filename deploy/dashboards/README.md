<!-- path: deploy/dashboards/README.md -->

# `deploy/dashboards/` — Baseline Dashboard Artefact

> **Status:** the authoritative substrate + workspace
> commitment is
> [ADR-0045](../../docs/adr/0045-baseline-dashboard-substrate.md);
> the authoritative consumer contract is
> [ADR-0039](../../docs/adr/0039-dashboard-contract.md).
> This README is the operator-facing import guide.

---

## What this directory contains

`baseline.json` — a Grafana-compatible JSON dashboard
model covering the five panels foundation 05
§"Dashboards" committed:

1. **Run success rate over time** — BigQuery / panel 1
2. **Per-entity check pass/fail rate** — BigQuery / panel 2
3. **Cost per entity per day** — BigQuery / panel 3
4. **Alerting volume per entity** — Prometheus / panel 4 *(renders "no data" until metric emission lands)*
5. **Scheduler health** — Prometheus / panel 5 *(renders "no data" until metric emission lands)*

Panels 1-3 query the BigQuery results dataset and light
up immediately once any rule produces results.
Panels 4-5 query the engine's Prometheus-compatible
`/metrics` endpoint per ADR-0039 §"Metric contract".
The metric emission code is deferred per ADR-0039
§"Baseline-dashboard implementation — deferred"; until
that lands, the two Prometheus-backed panels render
"no data" — visible-but-empty by design so the
dashboard light-up at emission-land time is a JSON-
already-present, no-PR moment.

---

## Data sources

Two Grafana data sources are required:

### `dq-bigquery`

A BigQuery data source pointed at the per-environment
results project + dataset. The recommended Grafana
plugin is `grafana-bigquery-datasource` (the
plugin's UID `dq-bigquery` is referenced from
`baseline.json` so the import resolves cleanly).

- **Project:** the per-env BigQuery project (see
  `engine/internal/env/{local,qa,prod}.go` —
  `BigQueryProject` field).
- **Authentication:** Application Default Credentials
  or the per-env service account. Operators provision
  per their substrate's auth posture.
- **Default dataset:** the panels select via the
  `bq_dataset` template variable in the dashboard;
  default value `dq_results_local` (set in the JSON's
  `templating.list[0]`). Override per env via
  Grafana's URL-parameter or template-variable UI.

### `dq-prometheus`

A Prometheus data source pointed at a Prometheus
instance scraping the engine's `/metrics` endpoint.
The plugin's UID `dq-prometheus` is referenced from
`baseline.json`.

- **Scrape target:** the engine binary's HTTP port
  (default `:8080` per `engine/internal/env/*.HTTPAddr`)
  at path `/metrics`. ADR-0039 §"Metric contract"
  commits the endpoint path.
- **Scrape interval:** 15s is operationally typical;
  the dashboard's `refresh: 1m` field caps how often
  Grafana re-queries.

Once metric emission lands, the four metrics the
dashboard reads — `dq_runs_total`,
`dq_checks_evaluated_total`, `dq_queue_depth`,
`dq_scheduler_triggers_managed` — appear under
Prometheus's standard label-matching surface; the
panels light up without further dashboard changes.

---

## Importing the dashboard

### Path A — manual JSON import

1. In Grafana, click **Dashboards → New → Import**.
2. Paste the contents of `baseline.json` or upload
   the file.
3. When prompted, map the dashboard's two data sources
   (`dq-bigquery`, `dq-prometheus`) to the matching
   data sources already provisioned in your Grafana
   instance. If your data source UIDs differ, edit
   the JSON's per-panel `datasource.uid` fields before
   import.
4. Click **Import**. The dashboard appears at the
   UID `dq-baseline` under whatever folder your
   Grafana instance is configured for.

### Path B — Kubernetes ConfigMap with Grafana sidecar auto-discovery

Grafana's standard sidecar auto-discovery picks up
ConfigMaps labelled `grafana_dashboard: "1"`. To wire
this up, create a ConfigMap whose data contains
`baseline.json`:

```sh
kubectl create configmap dq-baseline-dashboard \
  --from-file=baseline.json=deploy/dashboards/baseline.json \
  --namespace=monitoring \
  --dry-run=client -o yaml | \
  kubectl label --local -f - grafana_dashboard=1 \
    --dry-run=client -o yaml | \
  kubectl apply -f -
```

Adjust `--namespace=monitoring` to match wherever
your Grafana instance's sidecar watches. The
sidecar reads the ConfigMap's data, drops the
JSON into Grafana's dashboard directory, and Grafana
loads it on the next sync tick.

A Kustomize overlay that wires this idempotently
under `deploy/overlays/{local,qa,prod}/` is a natural
follow-up; not committed at v1 because the local
docker-compose substrate doesn't yet run Grafana
(see §"Local development" below).

### Path C — different substrate

Operators using a non-Grafana substrate (a different
BI tool; a custom dashboard) read `baseline.json`'s
panel definitions and reproduce the panels in their
tool. The JSON's `targets[].rawSql` and
`targets[].expr` fields carry the canonical queries
against the ADR-0039 contract; replicate them in the
target substrate's query language and the panels
behave identically.

---

## Local development

Foundation 05 §"Local development substrate posture"
named "local Prometheus + Grafana + Jaeger via
docker-compose" as the intended local-development
stack. **B2-33 landed the Prometheus + Grafana
half**: `make up` brings both services alongside the
substrate emulators; the baseline dashboard is
auto-provisioned via Grafana's file-provider; the
Prometheus data source is auto-wired.

Once the stack is up:

- **Grafana**: `http://localhost:3000` — anonymous
  Viewer access is enabled per
  `docker-compose.yml`'s `GF_AUTH_ANONYMOUS_*` env
  vars (admin/admin if you need write access for
  in-UI tweaks). The dashboard is at
  `http://localhost:3000/d/dq-baseline`.
- **Prometheus**: `http://localhost:9090` — useful
  for inspecting which targets are up and querying
  the metrics directly. The `dq-engine` job is
  configured to scrape `host.docker.internal:8080`
  on the host; if the engine binary isn't running
  the target reports `down` (panels 4-5 then show
  "no data" because nothing has been scraped).

Workflow:

1. `make up` — brings substrate + observability stack
   up.
2. `make build-engine && ./bin/dq-engine` (or
   `make demo-p6`) — starts the engine listening on
   host port 8080; Prometheus scrapes immediately.
3. Open Grafana, navigate to the baseline dashboard.
   Panels 4-5 continue to render "no data" until
   metric emission code lands per ADR-0039
   §"deferred"; panels 1-3 (BigQuery) require a
   configured BigQuery data source — see below.

The **BigQuery data source is not auto-provisioned**
in the local stack. The `grafana-bigquery-datasource`
plugin is installed (Grafana startup auto-pulls it
from `GF_INSTALL_PLUGINS`), but the data source's
connection is unconfigured because the plugin's
emulator support is best-effort (the SDK's auth flow
the plugin uses doesn't faithfully terminate against
the BigQuery emulator). Operators wire the data
source manually in the Grafana UI:

1. **Connections → Data sources → Add new data
   source → BigQuery**.
2. Either authenticate against a sandbox GCP project
   (preferred, full fidelity) or attempt the
   emulator path via service-account JSON pointing
   at `tabular-store:9050` (best-effort; behaviour
   varies by plugin version).
3. Save the data source with UID `dq-bigquery` so
   the dashboard panels resolve cleanly.

Once the BigQuery data source is configured, panels
1-3 light up the moment any rule produces results
in `dq_executions` / `dq_check_results`.

**Note on Jaeger**: foundation 05 named the trio
(Prometheus + Grafana + Jaeger); only the first two
ship at B2-33. Adding Jaeger lands if/when tracing
emission code arrives in the engine; the
log + metric + span three-channel obligation from
ADR-0007 CC10 commits the span signal but
implementation defers per `engine/internal/runner/runner.go`
lines 15-27. No B2 row is registered for Jaeger at
B2-33 — the slice lands when tracing emission does.

---

## Versioning

The dashboard JSON carries an internal `version: 1`
field at its root. Bump the value manually each time
a panel changes; Grafana detects the bump as an
upgrade on re-import.

The repo's git history is the authoritative version
surface. ADR-0039 §"Evolution rules" commits the
compatibility model the dashboard inherits — additive
within engine-major; breaking changes require an
engine-major bump and the ADR-0035 N-1 + 90-day
migration window.

---

## Maturity disclaimer

This is a **seed**. The current `baseline.json`
covers the five foundation-05-committed panels with
minimum-viable queries; richer features (templated
variables for entity / time-range / mode; alert
rules tied to panels; panel-level annotations from
CI deploys; per-entity drill-down dashboards) are
not committed at v1. Operator iteration during real
use is the source of truth for sharpening the
panels.

Specifically known gaps:

- **`evidence_summary.bytes_scanned`** consumed by
  panel 3 is undocumented per ADR-0039 OQ-3. The
  panel ships with the field named in its query
  string; if the engine's `bytes_scanned` emission
  shape changes, the panel updates with it. A
  future ADR amendment formalises the inventory.
- **Owner-level rollup** for panel 4 is computed at
  panel-query time by joining the `entity` label to
  `_owners.yaml`. Until the engine surfaces an
  explicit `owner` label on metrics, the entity
  axis is the per-owner proxy (one owner per
  entity per ADR-0006 CC9). A future enhancement
  could ship an engine-side owner label or a
  pre-joined view; not committed at v1.
- **Cost-specialised / alerting-specialised /
  per-entity drill-down dashboards** are reserved
  for future operator-demand-driven slices per
  ADR-0045 §Notes.
