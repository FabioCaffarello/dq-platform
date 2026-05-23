<!-- path: docs/adr/0007-loader-scheduler-retry-failure-semantics.md -->

# ADR-0007 — Loader, Scheduler, and Retry Failure Semantics

- **Status:** accepted
- **Date:** 2026-05-21

**Scope note (added 2026-05-23):** This ADR currently applies to set-oriented capability (realized over BigQuery). Record-oriented capability is addressed separately in Wave-S (see ADR-0020 forthcoming).

---

## Context

The engine's runtime has three failure-prone boundaries:

- **Manifest load.** Reading the active manifest at engine
  startup and at periodic refresh. Failure modes include
  pointer or body fetch errors, hash verification failure,
  compatibility-contract failure (ADR-0001), and PAT-1
  fail-fast cases (duplicate entity keys, schema validation,
  checksum mismatches, missing YAMLs).
- **Scheduler reconciliation.** Creating, updating, and
  deleting external scheduler triggers to match the active
  manifest's rule set.
- **Retries.** Two distinct retry layers — trigger delivery
  from the external scheduler to the engine, and check-level
  evaluation against the data substrate.

These failures must be handled asymmetrically. The
data-plane premise of "the manifest is the trust boundary"
means a half-loaded manifest is worse than no manifest;
startup failures must be loud and the engine must die. Once
in service, however, a refresh failure that retains a known-
good manifest is preferable to a panic; the data-plane
should continue serving from the prior manifest while the
operational signal is raised.

Retries must be bounded: unbounded retries hide the failure
from the operational signal that this ADR commits to making
visible.

---

## Decision

### 1. Loader at startup — process exit on any failure

When the engine starts and attempts its first manifest load,
any failure in the load pipeline causes the engine process
to exit with non-zero status and a structured final log
line naming the specific failure. The failure surface
includes:

- pointer read failure;
- manifest body fetch failure;
- hash verification failure (manifest or referenced YAML);
- compatibility-contract failure per ADR-0001;
- PAT-1 fail-fast cases (duplicate entity keys, schema
  validation failure, missing referenced YAMLs).

Ops sees the crash-loop in container orchestration and
investigates the upstream cause (manifest publication bug,
schema migration without engine support, etc.).

### 2. Loader at refresh — refuse swap, retain prior manifest

When the engine is running with a previously loaded
manifest and a refresh attempt fails for any reason in
section 1, the engine:

- retains the previously loaded manifest in memory as the
  active ruleset (no swap occurs);
- emits a structured log at WARN level naming the failing
  check and the manifest hash that failed to load;
- emits a metric counter increment;
- emits an operational alert per the routing ADR (ADR-0006);
- continues to attempt subsequent refresh cycles at the
  configured cadence.

After **N consecutive refresh failures** (parameter
deferred), the engine escalates: the operational alert is
promoted to high severity and a separate emission is
tagged "manifest refresh persistently failing".

The prior manifest continues to serve trigger requests
during all of this; the data-plane is unaffected.

### 3. In-flight executions complete against their loaded manifest

Once a trigger has been accepted and a plan has been
created, the execution proceeds against the manifest active
at plan creation. A subsequent refresh that swaps the
active manifest does not affect in-flight executions
because the engine retains the `manifest_hash` as
**in-memory execution-context state**, not as a persisted
column on `dq_executions`. Forensic linkage between a
persisted execution row and a specific manifest is
provided by `ruleset_version` (persisted, and part of the
`execution_id` formula).

### 4. Refresh cadence with hash short-circuit

The engine re-fetches `manifests/latest.json` on a
configurable interval. On each fetch:

- the engine compares the pointer's `manifest_hash` to the
  currently-loaded manifest's hash;
- if equal, the engine **does not** re-fetch the manifest
  body — content addressing guarantees the body is
  unchanged if the hash is unchanged;
- if different, the engine fetches the new manifest body,
  runs the load pipeline (compatibility checks + hash
  verification + PAT-1), and either swaps to the new
  manifest (success) or executes section 2 (failure).

Pointer-only reads are cheap; the short-circuit makes the
steady state essentially free. The short-circuit's safety
depends on the immutability guarantee from ADR-0005 holding
in practice (publisher discipline plus substrate
behavior); a future strengthening of manifest cryptographic
posture would discharge this dependency.

### 5. Scheduler reconciliation — per-operation best-effort

Lifecycle operations (create / update / delete triggers,
status inspection, orphan cleanup) are attempted
**independently per trigger**. If one fails:

- the trigger's state is marked `failing`;
- a structured log records the failure;
- the periodic reconciliation loop retries on its next
  iteration;
- after **M consecutive failures** for the same trigger
  (parameter deferred), an operational alert is emitted.

Sibling triggers in the same pass are attempted regardless
of one trigger's failure. The periodic loop is the
correctness mechanism.

### 6. Orphan triggers — cleaned up, not flagged

A trigger present in the external scheduler but not in the
active manifest's rule set is an **orphan** if and only if
it carries the engine's marker. The reconciliation loop
detects orphans and deletes them; orphan detection is not
a failure mode and produces no alert.

**The engine never modifies or deletes triggers it did not
create** — orphan detection applies only to triggers
carrying the engine's marker. The specific marker shape
(name prefix, metadata label, tag) is a scaffolding
detail; the abstract invariant is committed here.

### 7. Trigger-handler retries — bounded, exhaustion-alerted

When the scheduler retries a trigger and the trigger
handler is unreachable:

- retries are bounded by a configurable maximum;
- each retry uses identical inputs and therefore produces
  the identical `execution_id`;
- on exhaustion, **no `dq_executions` row is written** —
  the trigger never reached the engine, so no plan was
  created;
- the scheduler emits an operational alert "trigger T
  failed to deliver after N attempts"; the payload
  includes the would-be `execution_id` for forensic
  linkage with later successful executions of the same
  trigger.

The scheduler may schedule the trigger again on its next
cadence; that next trigger uses identical inputs and
therefore the identical `execution_id`.

### 8. Check-level retries — bounded, exhaustion produces `result = error`

When a check's evaluation fails with a transient error
(quota, network blip, intermittent timeout):

- the runner retries up to a configurable maximum;
- on exhaustion, the check's row in `dq_check_results` is
  written with `result = error`;
- the execution proceeds with other checks (always-
  continue per ADR-0004);
- the execution-status mapping is computed at execution
  finalization from the resulting result multiset.

Non-transient errors (query compilation error, missing
source detected mid-check, type mismatch) skip retries and
fail immediately to `result = error`. The transient-vs-non-
transient classification is implementation-shaped.

### 9. Pre-check entity-level validation

When the trigger handler creates an execution plan, it
determines source-table existence via a **lightweight
pre-check operation** (substantially cheaper than evaluating
any individual check). If the determination is "source not
present", the trigger handler writes the execution row
directly with `status = error` and no check rows are
produced.

The exact mechanism (metadata-read API call,
`information_schema` query, cached metadata layer, etc.) is
a scaffolding detail. The pre-check covers source-table
existence **only**. Other potential pre-checks (partition
column presence, schema compatibility) are not validated
pre-check and surface as check-level `result = error`.
Compatibility-contract checks from ADR-0001 do not repeat
at plan time; they are already enforced at load time.

### 10. `status = aborted` halt conditions (enumerated)

The execution-status branch for "global engine halt →
aborted" is populated by exactly:

- **Cost ceiling exceeded mid-execution.** Per-run bytes-
  scanned ceiling breached; engine halts the in-flight
  execution and writes a follow-up row with
  `status = aborted`.
- **Engine restart during in-flight execution.** Process
  restart for any reason (planned shutdown, container
  restart, OOM kill, startup-failure cascade); the
  `running` row is left dangling and finalized by orphan
  detection as `status = aborted`.
- **Container OOM / crash mid-execution.** Same path —
  process dies; orphan detection finalizes.
- **Global resource limit.** Per-environment concurrency
  budget exceeded for long enough that the execution is
  force-evicted from the run queue.
- **Operator-issued abort.** Admin API endpoint allows
  operators to abort a specific in-flight execution; the
  abort writes a follow-up row with `status = aborted`
  and the operator's identity in the audit fields.

All other failures route through sections 1, 2, 5, 7, 8,
or 9 — none of those produce `aborted`.

### 11. Orphan-run detection

A periodic engine task scans `dq_executions` (via
`dq_executions_current`) for rows with `status = running`
and `started_at` older than a configurable threshold (must
exceed the maximum expected execution duration). For each
match, the engine writes a follow-up row with the same
`execution_id` and `attempt_id`, `status = aborted`, and an
`error_summary` identifying engine abandonment.

The orphan-aborted row carries the **orphan-detector
engine instance's** `engine_version`, not the abandoned
engine's. Different `engine_version` values within a
single attempt's lifecycle is the expected pattern for the
orphan-finalization case. Forensic queries on the base
table see the abandonment event; the canonical view
returns the orphan-detector's `engine_version` as the
"effective evaluator" — operators reading the canonical
view see the orphan-detector; investigators reading the
base table see both.

### 12. Observability emission

Every failure path from this ADR emits, at minimum:

- a structured log line with `execution_id` (when
  applicable), `entity`, `check_id` (when applicable), and
  a failure-type identifier;
- a counter increment;
- an OpenTelemetry span recording the failure with
  attributes matching the log fields.

No failure is silent.

---

## Consequences

1. **Asymmetric handling.** Strict at load (process exit),
   best-effort at refresh and scheduler reconciliation
   (retain prior state, periodic convergence), bounded at
   the retry layers (trigger delivery, check evaluation).
   This is what keeps "manifest is the trust boundary"
   honest while not panicking on every transient hiccup.

2. **The data-plane is isolated from refresh failures.**
   Prior manifest continues to serve trigger requests
   while refresh retries. Refresh failure alone does not
   stop the engine.

3. **Hash-short-circuit makes steady-state free.** When the
   pointer hash matches the loaded manifest's hash, the
   engine does not re-fetch the body. Pointer-only reads
   are cheap; this is what makes a tight refresh cadence
   affordable.

4. **In-flight executions are isolated from refreshes.**
   `manifest_hash` is in-memory execution-context state,
   not a persisted column. The persisted forensic linkage
   is via `ruleset_version`.

5. **Orphan detection is fail-safe.** The engine only
   touches triggers it created. Sibling-team scheduler
   entries that happen to share infrastructure are never
   modified by orphan cleanup.

6. **Trigger-handler exhaustion produces no execution row.**
   The trigger never reached the engine; the operational
   signal lives in the scheduler's emit, with the
   forensically-computable `execution_id` in the payload.

7. **Check-level retries are bounded.** Exhaustion produces
   `result = error`; the execution proceeds with sibling
   checks; the execution-status mapping (ADR-0004) is
   computed at finalization.

8. **`status = aborted` is reserved for global engine
   halts.** Loader, scheduler, trigger-handler, and
   check-level retry failures all route through paths that
   produce `status = error` or no row at all — not
   `aborted`.

9. **Orphan finalization writes a follow-up row with the
   orphan-detector's `engine_version`.** Different
   `engine_version` values within a single attempt's
   lifecycle is the expected pattern for this case;
   ADR-0002's active-visibility commitment for
   `engine_version` makes the event observable.

10. **Every failure emits log + metric + span.** No silent
    failures. The exact metric and span names are
    scaffolding details; the three-channel commitment is
    binding.

11. **Compatibility-contract failures (ADR-0001) and hash-
    verification failures (ADR-0005) both flow through
    loader failure paths.** No third failure mode; no
    salvage path. The engine fails closed.

12. **Numeric parameters are deferred.** Trigger-handler
    retry maximum, check-level retry maximum and backoff,
    refresh cadence, refresh-failure escalation N,
    scheduler-reconciliation-failure threshold M, orphan
    detection threshold and scan cadence, cost ceiling,
    and concurrency-budget eviction threshold are all
    follow-up parameters with per-environment values.
    This ADR commits the **shapes**; the values are
    scaffolding parameters.

---

## Notes

- The engine's trigger marker (name prefix, metadata
  label, tag, etc.) is a scaffolding detail. The
  invariant — the engine touches only its own triggers —
  is the contract.
- The transient-vs-non-transient classification at the
  check-retry layer is a small allowlist of error types;
  its specific membership is a scaffolding detail.
- The pre-check mechanism (metadata-read API,
  `information_schema`, cached metadata) is selected
  during scaffolding; the contract is "lightweight" and
  "source-table existence only".
- An admin endpoint for operator-issued aborts is a
  follow-up scaffolding item.
