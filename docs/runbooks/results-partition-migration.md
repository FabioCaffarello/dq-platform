<!-- path: docs/runbooks/results-partition-migration.md -->

# Runbook — Migrate non-partitioned dq_executions / dq_check_results to time-partitioned tables

One-time procedure for any deployment whose `dq_executions` and
`dq_check_results` tables predate
[ADR-0031](../adr/0031-evidence-retention-parameters.md)'s
partitioning + partition-expiration posture.

ADR-0031 §"Single-tier retention via BigQuery partition
expiration" §2 commits: pre-existing non-partitioned tables
require a one-time `CREATE TABLE ... AS SELECT ... PARTITION
BY DATE(recorded_at)` followed by a rename. The current
`EnsureSchema` (per `engine/internal/results/bigquery_store.go`)
creates partitioned tables on first contact, so qa/prod
deployments provisioned after ADR-0031 are green-field and
**do not need this runbook**. Use it only when:

- An existing deployment has accumulated data in
  non-partitioned `dq_executions` / `dq_check_results`
  tables.
- The dataset must retain that history (a routine `dq-local`
  wipe + re-`EnsureSchema` is the simpler alternative when
  history is disposable).

---

## 1. When to use

- The engine log emits a warning at startup like
  `EnsureSchema: existing table is not partitioned; partition
  expiration will not apply`. (The engine treats this as a
  warning rather than a startup failure to preserve the
  existing tables; the partitioning gap is a known follow-up.)
- A deployment is rolling forward through the ADR-0031 release
  and the operator wants partition-expiration to take effect
  immediately rather than waiting for the natural data
  rollover.
- A periodic audit confirms `INFORMATION_SCHEMA.TABLES.table_partitioning_type`
  is `NULL` (i.e., not partitioned) for one or both target
  tables in a deployment that should be partitioned.

Do **not** use this runbook for:

- Green-field deployments where the tables don't exist yet
  (`EnsureSchema` creates them already partitioned).
- A `dq-local` development dataset where history is
  disposable — `DROP TABLE` + restart the engine is faster
  and avoids the temporary-table dance.

## 2. Preconditions

- BigQuery write access to the target project + dataset
  (the same role the engine binary uses at runtime, with
  additional `bigquery.tables.create` and
  `bigquery.tables.delete` if the role is narrower).
- Engine binary version that ships ADR-0031's `EnsureSchema`
  extension (`engine` package version ≥ the release that
  includes the `TimePartitioning` field in
  `bigquery_store.go`).
- Operator awareness that the procedure is **not online**:
  during step 3.4 the engine sees a brief "table missing"
  window between the drop and the rename. Pause the engine
  binary before step 3.4 if writes can't tolerate that window;
  for production deployments this means scaling the deployment
  to zero replicas first.

## 3. Procedure

The procedure repeats for each of the two tables. The example
below uses `dq_executions`; repeat for `dq_check_results`,
substituting the table name + the schema differences (ADR-0003
§3 vs §7).

### 3.1 Identify the target dataset

Set environment shorthand for the rest of the steps:

```sh
export PROJECT="your-bq-project"
export DATASET="dq_results_prod"   # or dq_results_qa, etc.
export RETENTION_DAYS=730           # match per-env ResultsRetention (ADR-0031)
```

Confirm the tables exist and are not partitioned:

```sql
SELECT table_name, partition_column_name, partition_column_type
FROM `${PROJECT}.${DATASET}.INFORMATION_SCHEMA.TABLES`
WHERE table_name IN ('dq_executions', 'dq_check_results')
```

A `NULL` in `partition_column_name` confirms the table needs
migration.

### 3.2 Create the partitioned successor table

```sql
CREATE TABLE `${PROJECT}.${DATASET}.dq_executions_partitioned`
PARTITION BY DATE(recorded_at)
OPTIONS (
  partition_expiration_days = 730,    -- match ${RETENTION_DAYS}
  description = "Partitioned successor for the B2-13 migration; replaces dq_executions"
)
AS SELECT * FROM `${PROJECT}.${DATASET}.dq_executions`
```

For `dq_check_results`, the partition column is `executed_at`
(per ADR-0003 §7), not `recorded_at`:

```sql
CREATE TABLE `${PROJECT}.${DATASET}.dq_check_results_partitioned`
PARTITION BY DATE(executed_at)
OPTIONS (
  partition_expiration_days = 730,
  description = "Partitioned successor for the B2-13 migration; replaces dq_check_results"
)
AS SELECT * FROM `${PROJECT}.${DATASET}.dq_check_results`
```

The `CREATE TABLE AS SELECT` is a single BigQuery job that
scans the original table once and writes the partitioned copy.
Cost is the source table's bytes-scanned, billed once.

### 3.3 Verify row counts match

Before swapping, confirm the row counts match exactly:

```sql
SELECT
  (SELECT COUNT(*) FROM `${PROJECT}.${DATASET}.dq_executions`) AS original,
  (SELECT COUNT(*) FROM `${PROJECT}.${DATASET}.dq_executions_partitioned`) AS partitioned
```

If the counts diverge, the CREATE TABLE AS SELECT job failed
partway. Drop the partitioned table, investigate the job
history in the BigQuery console, and re-run step 3.2.

Do the same verification for `dq_check_results`.

### 3.4 Swap the tables (engine paused)

> **Pause the engine first** if writes can't tolerate the
> brief "table missing" window between DROP and RENAME. For
> production deployments: scale `deployment/dq-engine` to
> zero replicas via the deploy-overlay's normal scaling
> mechanism. For local dev, `make down` is sufficient.

```sql
-- Atomic-ish swap: BigQuery executes these sequentially.
DROP TABLE `${PROJECT}.${DATASET}.dq_executions`;
ALTER TABLE `${PROJECT}.${DATASET}.dq_executions_partitioned`
  RENAME TO dq_executions;

DROP TABLE `${PROJECT}.${DATASET}.dq_check_results`;
ALTER TABLE `${PROJECT}.${DATASET}.dq_check_results_partitioned`
  RENAME TO dq_check_results;
```

Resume the engine. `EnsureSchema` at boot sees the now-
partitioned tables and is a no-op (the schemas match).

### 3.5 Verify partitioning is active

```sql
SELECT table_name, partition_column_name, partition_column_type,
       partition_expiration_ms
FROM `${PROJECT}.${DATASET}.INFORMATION_SCHEMA.TABLES`
WHERE table_name IN ('dq_executions', 'dq_check_results')
```

`partition_column_name` must be `recorded_at` / `executed_at`
respectively; `partition_expiration_ms` must equal
`RETENTION_DAYS * 86_400_000`.

## 4. Verification

1. **Engine resumed without `EnsureSchema` warnings.** The
   warning that motivated the runbook (per §1) is now absent.
2. **First post-migration trigger writes succeed.** Submit a
   manual trigger (e.g., via `dq-manifest`-driven publish +
   the existing test-channel entity) and confirm a new row
   appears in `dq_executions` / `dq_check_results`. Check that
   the row's `recorded_at` falls in a now-existing partition:

   ```sql
   SELECT execution_id, recorded_at,
          _PARTITIONTIME AS partition_time
   FROM `${PROJECT}.${DATASET}.dq_executions`
   WHERE execution_id = '<the-new-execution-id>'
   ```

3. **Existing observability queries still return.** The
   `dq_executions_current` view (ADR-0003 §2) is independent
   of the underlying partitioning and continues to work; any
   downstream dashboards reading the view see no change.
4. **Partition expiration is scheduled.** BigQuery shows the
   `partition_expiration_ms` value matching the per-env
   `EvidenceRetention.ResultsRetention` constant on the
   engine binary.

## 5. Rollback / escape

The procedure is **non-reversible without data loss** once
step 3.4 completes, because the original `dq_executions` /
`dq_check_results` tables are dropped. Two escape options:

- **Before step 3.4** — drop the new partitioned tables
  (`DROP TABLE dq_executions_partitioned`,
  `DROP TABLE dq_check_results_partitioned`) and the original
  tables remain intact. No data loss.
- **After step 3.4** — if BigQuery's table-undeletion window
  is still open (typically 7 days post-drop in standard BQ
  configurations), the original tables may be restorable via
  `bq cp -f` from the `@<timestamp>` snapshot. This is
  substrate-specific and may not be available in all
  deployments; treat as best-effort.

To minimize blast radius:

- Run on a non-production dataset first (`dq_results_qa`) and
  verify the engine boots cleanly + a manual trigger lands.
- Take a BigQuery snapshot before step 3.4 if the substrate
  + cost budget allows:

  ```sh
  bq cp ${PROJECT}:${DATASET}.dq_executions \
        ${PROJECT}:${DATASET}.dq_executions_snapshot_$(date +%Y%m%d)
  ```

## 6. Escalation

- **CREATE TABLE AS SELECT fails partway.** The partitioned
  successor table is left in a partial state. Drop it and
  investigate via the BigQuery job-history UI; common causes
  are slot-pool contention (raise the job priority) or
  per-query bytes-billed limits (the source table is large
  enough to exceed a per-project cap). Escalate to SRE.
- **Row counts diverge in step 3.3.** Same root cause as
  above. Drop and retry; if it fails repeatedly, the source
  table may have a corrupted partition that BigQuery can't
  resolve — escalate to SRE.
- **Engine refuses to write after step 3.4.** Check that the
  rename completed (`SELECT * FROM
  ${PROJECT}.${DATASET}.INFORMATION_SCHEMA.TABLES WHERE
  table_name = 'dq_executions'` returns one row with
  `partition_column_name = recorded_at`). If the rename
  failed and only `dq_executions_partitioned` exists, rerun
  the ALTER TABLE rename. If the engine still refuses, check
  the IAM role grants — the rename may have lost the
  engine's `bigquery.tables.updateData` permission on the
  new identity. Escalate to platform-team.
- **Production deployment can't tolerate the "table
  missing" window.** This runbook is not online. Future
  improvement: a write-shadow pattern where the engine writes
  to both tables during a transition window. Reserved as a
  future B2 row when the operational signal demonstrates
  it's needed.
