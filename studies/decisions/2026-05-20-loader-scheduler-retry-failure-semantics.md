<!-- path: studies/decisions/2026-05-20-loader-scheduler-retry-failure-semantics.md -->

# B0-7 — Loader / Scheduler / Retry Failure Semantics

## Metadata

- B0 reference: B0-7 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md).
- Status: draft (Wave 1, session 7).
- Last updated: 2026-05-20.
- Upstream resolved: B0-1
  ([`2026-05-20-engine-rules-compatibility.md`](./2026-05-20-engine-rules-compatibility.md)),
  B0-5
  ([`2026-05-20-manifest-publication-semantics.md`](./2026-05-20-manifest-publication-semantics.md)),
  B0-4
  ([`2026-05-20-failure-scope.md`](./2026-05-20-failure-scope.md)),
  with indirect inputs from B0-2
  ([`2026-05-20-run-identity-and-idempotency.md`](./2026-05-20-run-identity-and-idempotency.md))
  and B0-3
  ([`2026-05-20-result-write-model.md`](./2026-05-20-result-write-model.md)).
- Downstream open: B0-6 (alert routing contract) — the operational
  alerts emitted by this study's failure paths flow through B0-6's
  routing.
- Promotion target: see final section.

---

## Context

Three failure surfaces in the engine are still unspecified, and
each is referenced by an upstream B0 as "B0-7's call":

- **Loader failure response** — what the engine does when a
  manifest fails to load (B0-1 contract checks fail, B0-5 hash
  verification fails, PAT-1 fail-fast cases fire). B0-1 commits
  "fail closed"; B0-5 commits the read shape; **B0-7 picks the
  failure mode** (process exit / refuse-swap / fallback / hot-reload
  disable).
- **Scheduler reconciliation failure response** — what happens
  when PAT-2 (lifecycle-aware scheduler integration) operations
  fail mid-reconciliation. Foundation doc
  [`04-system-architecture.md`](../foundation/04-system-architecture.md)
  §"PAT-2" describes the lifecycle (deploy / delete / status /
  orphan cleanup / periodic reconciliation); B0-7 picks the
  failure semantics for each step.
- **Retry budget exhaustion** — what happens when scheduler-driven
  trigger retries or check-level transient-error retries run out.
  Foundation doc
  [`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
  §"Retry Semantics" describes three retry categories; B0-4 CC1
  classifies exhausted check-level retries as `result = error`;
  **B0-7 picks the bounds and the exhaustion outcomes**.

B0-4 additionally deferred two specific items to B0-7:

- **`status = aborted` conditions** (B0-4 CC2 branch 1, B0-4
  CC6) — the exact set of halts that produce `aborted` vs.
  `error`.
- **Pre-check validation mechanism** (B0-4 CC5) — when in the
  engine lifecycle pre-check validations run (load-time vs.
  plan-time vs. lazy), which specific validations are performed,
  and how they relate to B0-1's load-time contract checks. B0-4
  commits only the state-transition outcome; B0-7 picks the
  mechanism.

B0-7 — as recorded in the decision log:

> What exactly causes ruleset load failure, scheduler
> reconciliation failure, and retry budget exhaustion?

This study locks:

1. **Loader failure response** at engine startup vs. at refresh
   (two distinct postures).
2. **Scheduler reconciliation failure response** — per-operation
   best-effort with periodic convergence.
3. **Retry bounds and exhaustion outcomes** at the
   trigger-handler level and the check-runner level.
4. **`status = aborted` halt conditions** (the enumeration
   B0-4 deferred).
5. **Pre-check validation mechanism** (the lifecycle placement
   B0-4 deferred).
6. **Manifest refresh cadence shape** and the short-circuit
   behavior on unchanged pointer hash (B0-5 CC6 framing
   resolved here).
7. **Orphan-run detection** — the mechanism for finalizing
   `running` rows whose engine instance died mid-execution.

What this study does **not** decide:

- Specific numeric retry counts, backoff schedules, refresh
  intervals, orphan-detection thresholds — B1 picks numeric
  parameters per environment.
- The alert routing for the operational alerts emitted by these
  failure paths — B0-6.
- The shape of the Admin API surface that exposes scheduler
  reconciliation status — Wave 3.
- The observability emission format (log structure, metric
  names, OpenTelemetry span shape) — Wave 3, informed by
  foundation doc 04 PAT-5 and §"Observability Contract".

The decision matters because the engine's operational
trustworthiness depends on every failure being **classifiable and
explainable**. A failure that produces an ambiguous outcome —
"the engine kept running but I don't know what it loaded" — is
the failure mode foundation doc 05 imperative #3 (visible failure
over silent degradation) is designed to prevent. B0-7 closes the
last remaining gaps in the failure-response surface.

---

## Decision Drivers

The decision must satisfy the following, in priority order.

1. **D1. Fail-closed contracts (B0-1 + PAT-1).** Every contract
   check the loader runs (B0-1 C6, B0-5 hash verification, PAT-1
   schema validation / duplicate keys / missing YAMLs) MUST
   produce a failure when it fails — no silent fallback to a
   permissive interpretation. This is B0-1's locked invariant.

2. **D2. Continuity of in-flight work.** An execution already
   running against a loaded manifest must complete against that
   manifest, even if a refresh attempt fails mid-flight. The
   refresh failure is a control-plane event; the in-flight
   execution is a data-plane event; they do not collide.

3. **D3. Convergence over atomicity (PAT-2).** Scheduler
   reconciliation against an external scheduler is a convergence
   loop, not a transaction. Each operation can succeed or fail
   independently; the periodic loop is the platform's
   correctness mechanism. Atomic all-or-nothing reconciliation
   is brittle in distributed settings and incompatible with
   PAT-2's lifecycle framing.

4. **D4. Bounded retries.** Every retry path has a configurable
   maximum; no unbounded retry storms. Foundation doc 05
   §"Retry Semantics" sketches "three attempts" for trigger
   retries and "two retries with exponential backoff" for
   check-level transient errors; B0-7 commits the shape (bounded,
   configurable) and defers the numerics to B1.

5. **D5. Visible failure (foundation doc 05 imperative #3).**
   Every failure mode produces (a) a structured log line with
   enough context for triage and (b) an operational alert (per
   B0-4 CC7's category mapping). No failure mode is silent.

6. **D6. Determinism (P2).** Given the same inputs and the same
   environment state, the engine produces the same failure
   response. No environment-dependent fallback paths; no
   "sometimes retry, sometimes not" heuristics.

7. **D7. Identity stability across retries (B0-2 CC3).**
   Scheduler retries reuse the same `execution_id`. Retry-budget
   exhaustion at the trigger-handler level does not produce a
   new `execution_id`; the alert payload references the original
   `execution_id` for forensic linkage.

8. **D8. Storage contract (B0-3 + B0-4).** Every failure outcome
   maps to a status value already committed by B0-3 CC6 (and
   refined by B0-4 CC2's pure-function mapping):
   `running` / `success` / `failed` / `error` / `aborted`. No
   new status values; B0-7 fills in which conditions reach
   `error` vs. `aborted` via the loader and pre-check paths.

9. **D9. Observability is part of the contract (foundation doc
   05 §"Observability Contract").** Every failure path emits
   structured logs with `execution_id`, `entity`, `check_id`
   when applicable; metrics counters tick; traces span the
   failure event. The format is Wave 3, but the emission is a
   B0-7 commitment. **D9 is the enforcement mechanism for D5** —
   without committed emission, "visible failure" is aspirational;
   the contract on emission is what makes the policy
   operationally real.

---

## Considered Options

The decision has three coupled sub-policies. Options below are
**composite postures** combining each sub-decision; differences
across options are in the asymmetry of strictness across the
three surfaces.

### Option A — Symmetric strict (everything fails hard)

**Loader**: process exit on any failure, whether at startup or
during refresh. The engine never operates on a known-invalid
manifest; if the current refresh fails, the engine exits and
ops restarts it (presumably after fixing the upstream).

**Scheduler**: atomic all-or-nothing reconciliation. If any
trigger create/update/delete operation in a reconciliation pass
fails, the whole pass is rolled back; engine retries the
complete pass on the next periodic cycle.

**Retries**: bounded with strict exhaustion (single configured
maximum; on exhaustion, no further attempts until manual
intervention).

**Trade-offs.**

- Pro: simplest mental model — every failure is fatal at its
  scope.
- Pro: maximum fail-loud posture.
- Con: violates D2 — a refresh failure mid-execution kills the
  engine and the in-flight execution dies with it (the
  `running` row will be marked `aborted` by orphan-detection,
  but the work is lost).
- Con: violates D3 — atomic scheduler reconciliation against an
  external system is operationally infeasible; transient
  failures (network blip, scheduler quota) would cascade into
  engine restarts.
- Con: brittle to transient infrastructure events. A 30-second
  scheduler API outage produces an engine restart cascade and
  alarm storm.
- Con: identical behavior at startup and refresh ignores the
  meaningful difference (no-prior-state vs. prior-state-exists).

Reject on D2, D3, and operational robustness grounds.

### Option B — Symmetric graceful (everything degrades gracefully)

**Loader**: refuse-swap on any failure, both at startup and at
refresh. At startup with no prior manifest, the engine starts
but refuses to serve trigger requests (returns 503) until a
valid manifest loads.

**Scheduler**: best-effort with retry. Individual operations
attempted; failures marked; periodic loop converges.

**Retries**: bounded with graceful exhaustion (after maximum,
log + alert + give up; no manual-intervention requirement).

**Trade-offs.**

- Pro: D2 satisfied — refresh failure does not affect in-flight
  work.
- Pro: D3 satisfied — convergence loop is the correctness
  mechanism.
- Con: startup with no valid manifest produces a running engine
  that refuses traffic — operationally confusing ("the engine is
  up but doesn't work"). Crash-loop is clearer.
- Con: violates the "fail-loud at startup" intuition — a fresh
  deployment with a broken manifest looks healthy from process
  monitoring but is operationally broken.
- Con: violates the spirit of D1 partially — "fail closed"
  at startup with no fallback is arguably "process exit"; a
  503-emitting engine is a softer interpretation.

Reject on the startup-posture grounds. The data-plane behavior
is correct but the control-plane posture at startup is wrong.

### Option C — Asymmetric (strict load + best-effort scheduler + bounded retries)

**Loader at startup**: process exit. No prior manifest in
memory; no fallback possible; ops sees crash-loop and
investigates. The crash-loop is the right operational signal.

**Loader at refresh**: refuse swap, retain the prior manifest
in memory, log + alert operationally, continue periodic refresh
attempts. After N consecutive refresh failures (B1), escalate.
In-flight executions complete against the prior manifest (D2).

**Scheduler reconciliation**: best-effort per operation +
periodic convergence. Each create/update/delete is attempted
independently; failures mark the trigger as `failing` (per
PAT-2); periodic loop retries.

**Retries (trigger-handler and check-level)**: bounded with
configurable maximum; on exhaustion, the failure is recorded in
the appropriate place (no `dq_executions` row for trigger
exhaustion; `result = error` in `dq_check_results` for check
exhaustion per B0-4 CC1).

**Trade-offs.**

- Pro: D1 + D2 + D3 + D4 all satisfied.
- Pro: startup failure crash-loops loudly (the right signal);
  refresh failure does not disrupt in-flight work (continuity);
  scheduler converges over time (PAT-2's framing).
- Pro: the asymmetry mirrors the structural asymmetry of the
  surfaces themselves: the loader is a contract enforcement
  surface (strict by nature); the scheduler is a convergence
  surface (best-effort by nature).
- Con: two different loader behaviors (startup vs. refresh) is
  more to document and reason about than a symmetric posture.
  The asymmetry must be justified explicitly.
- Con: "continue refresh attempts after a failure" requires an
  escalation policy after N consecutive failures, which is a
  parameter B1 must pick. Without an escalation policy, a
  refresh stuck against a permanently-broken manifest produces
  unbounded operational alerts.

### Option D — Lazy (defer everything)

Option D is presented as the **contrastive baseline** — the
foundation-doc-implicit posture if no policy were committed
here. A "lazy across the board" stance is structurally
unacceptable per B0-1 ("fail closed") and foundation doc 05
imperative #3 ("visible failure"); any steel-manned alternative
to Option C with bounded retries collapses into Option C with a
different retry-exhaustion shape (e.g., slower-backoff after
the bound rather than emitting "give up" alerts immediately).
The retry-exhaustion shape is a sub-decision within Option C
(see CC6 / CC7).

**Loader**: accept partial loads; log unknown-`version` rules
as warnings; continue with whatever loaded.

**Scheduler**: best-effort; no atomicity; periodic loop
attempts whatever it can.

**Retries**: unbounded with backoff.

**Trade-offs.**

- Pro: maximum tolerance for transient failures.
- Con: violates D1 hard — silently accepting partial loads is
  the exact failure mode B0-1's "fail closed" prohibits.
- Con: violates D4 — unbounded retries can cascade resource
  exhaustion.
- Con: violates D5 — "log as warning and continue" is silent
  degradation by another name.

Reject on D1 and D5.

---

## Recommendation

Adopt **Option C** — asymmetric posture: strict at the loader
(with startup vs. refresh differentiation), best-effort at the
scheduler reconciliation loop, bounded retries at both
trigger-handler and check-runner layers.

The recommendation is grounded in:

- prior decision
  [`2026-05-20-engine-rules-compatibility.md`](./2026-05-20-engine-rules-compatibility.md)
  (B0-1) — C6 "must not fail open" on three contract checks;
  the failure mode (exit / refuse-swap) is this study's call.
- prior decision
  [`2026-05-20-manifest-publication-semantics.md`](./2026-05-20-manifest-publication-semantics.md)
  (B0-5) — CC6 read shape, CC8 manifest-header carries enough
  for contract checks; this study locks the engine-side response.
- prior decision
  [`2026-05-20-failure-scope.md`](./2026-05-20-failure-scope.md)
  (B0-4) — CC2 status mapping, CC5 pre-check entity-level
  problem → `status = error`; this study picks the lifecycle
  placement.
- foundation doc
  [`04-system-architecture.md`](../foundation/04-system-architecture.md)
  §"PAT-1" (fail-fast registry loading) and §"PAT-2"
  (lifecycle-aware scheduler integration).
- foundation doc
  [`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
  §"Run Identity and Idempotency", §"Retry Semantics",
  §"Operating Posture" (imperative #3 visible failure).

The specific commitments beyond what those documents state are
**new contribution proposed here, requires review**:

1. **Loader posture is asymmetric: process exit at startup,
   refuse-swap-with-retain-prior at refresh.** The asymmetry is
   load-bearing — neither symmetric posture (Option A or B)
   satisfies both D1 (fail-closed at startup) and D2 (continuity
   of in-flight work at refresh). **New contribution proposed
   here, requires review.**

2. **Refresh cadence is periodic + short-circuit on unchanged
   pointer hash.** The engine re-fetches `manifests/latest.json`
   on a configurable cadence (B1 picks the default); if the
   pointer's `manifest_hash` equals the currently-loaded
   manifest's hash, the engine **does not** re-fetch the
   manifest body. This resolves B0-5 CC6's deferred
   short-circuit decision in favor of "always short-circuit
   when pointer hash matches". **New contribution proposed
   here, requires review.**

3. **Pre-check entity-level validation runs at plan creation
   time, not at load time.** The trigger handler validates the
   entity's source-table existence with a single metadata read
   when constructing the execution plan (foundation doc 04
   §"Execution Flow" step 2). B0-1 C6 contract checks already
   run at load time and do not repeat at plan time. This
   resolves B0-4 CC5's deferred mechanism. **New contribution
   proposed here, requires review.**

4. **Orphan-run detection finalizes stuck `running` rows.**
   A periodic engine task scans `dq_executions` for rows with
   `status = running` older than a configurable threshold (B1
   picks the value); for each such row, the engine writes a
   follow-up row (per B0-3 CC1 append-only) with
   `status = aborted` and an error_summary indicating
   "engine instance abandoned this execution". **New
   contribution proposed here, requires review.**

5. **The set of conditions producing `status = aborted` is
   enumerated explicitly** (CC10 below) — closing B0-4's
   deferred boundary. **New contribution proposed here,
   requires review.**

---

## Consequences

Adopting this recommendation commits the platform to the
following.

**CC1. Loader at startup: process exit on any failure.** When
the engine starts and attempts its first manifest load, any
failure in the load pipeline — pointer read, hash verification,
manifest fetch, B0-1 contract checks, PAT-1 fail-fast cases
(duplicate entity keys, schema validation failures, checksum
mismatches, missing referenced YAMLs) — causes the engine
process to exit with a non-zero status and a structured
final log line naming the specific failure. Ops sees the
crash-loop in container orchestration and investigates the
upstream cause (manifest publication bug, schema migration
without engine support, etc.).

**CC2. Loader at refresh: refuse swap, retain prior manifest
in memory.** When the engine is running with a previously
loaded manifest and a refresh attempt fails for any of the
same reasons listed in CC1, the engine:

- retains the previously loaded manifest in memory as the
  active ruleset (no swap occurs);
- emits a structured log at WARN level naming the failing
  check and the manifest hash that failed to load;
- emits a metric counter increment (`dq_engine_refresh_failures_total`,
  exact name Wave 3);
- emits an operational alert (per B0-4 CC7's "operational"
  category, routed by B0-6);
- continues to attempt subsequent refresh cycles at the
  configured cadence.

After **N consecutive refresh failures** (B1 picks N — see
OQ-2), the engine escalates: operational alert promoted to
high severity (B0-6's call within category) plus a separate
emission tagged "manifest refresh persistently failing".

The prior manifest continues to serve trigger requests during
all of this; the data-plane is unaffected.

**CC3. In-flight executions complete against their loaded
manifest.** Once a trigger has been accepted and a plan has
been created (foundation doc 04 §"Execution Flow" step 2-3),
the execution proceeds against the manifest that was active at
plan creation. A subsequent refresh that swaps the active
manifest does not affect in-flight executions because the
engine retains the `manifest_hash` (from B0-5's pointer file)
as **in-memory execution-context state**, not as a persisted
column on `dq_executions`. B0-3's storage contract is
unchanged — no `manifest_hash` field is added to the table.
Forensic linkage between a persisted execution row and a
specific manifest is provided by `ruleset_version` (which IS
persisted per B0-3 CC3 and is part of the `execution_id`
formula per B0-2 CC1); the in-memory `manifest_hash` is
engine-internal and exists for refresh-isolation purposes only.
This satisfies D2.

**CC4. Scheduler reconciliation is per-operation best-effort
with periodic convergence.** The PAT-2 lifecycle operations
(create / update / delete triggers, status inspection, orphan
cleanup) are attempted **independently** per trigger. If a
single operation fails:

- the trigger's state is marked `failing` (per PAT-2's framing
  of trigger states — `enabled` / `paused` / `failing`);
- a structured log records the failure with the trigger
  identifier and the upstream cause;
- the periodic reconciliation loop (running on a configurable
  cadence per environment, B1 picks) retries the operation on
  its next iteration;
- after **M consecutive failures** for the same trigger (B1
  picks M — see OQ-3), an operational alert is emitted.

Engine code paths never abort a reconciliation pass because of
one trigger's failure; sibling triggers in the same pass are
attempted independently. The periodic loop is the correctness
mechanism (D3).

**CC5. Orphan triggers are cleaned up, not flagged as
failures — engine touches only its own triggers.** A trigger
present in the external scheduler but not in the active
manifest's rules list is an orphan, **provided it carries the
engine's marker** (see below). The reconciliation loop detects
orphans and deletes them; orphan detection is **not** a
failure mode and produces no alert. (A delete operation that
itself fails proceeds per CC4 — the trigger gets marked
`failing` and is retried.)

The abstract identification contract is: **the engine
identifies its own triggers via a marker set at trigger-create
time** (a name prefix such as `dq-`, a metadata label, a tag,
or an equivalent stable identifier). **The engine never modifies
or deletes triggers it did not create** — orphan detection
applies only to triggers carrying the engine's marker. This
protects against accidental deletion of unrelated scheduler
triggers that happen to share infrastructure (e.g., a sibling
team's scheduler entries).

The specific marker shape (name prefix, metadata label, tag,
etc.) is Wave 3 implementation — see OQ-7 (the Admin API
surface for inspecting scheduler reconciliation status uses
the same marker). CC5 commits the abstract invariant; Wave 3
picks the concrete marker.

**CC6. Trigger-handler retries are bounded and produce
operational alerts on exhaustion.** When the scheduler retries
a trigger and the trigger handler is unreachable (network
failure, engine restart, etc.):

- retries are bounded by a configurable maximum (foundation
  doc 05 sketches three; B1 picks — see OQ-1);
- each retry uses identical inputs and therefore produces the
  identical `execution_id` per B0-2 CC3;
- on exhaustion, **no `dq_executions` row is written** — the
  trigger never reached the engine, so no plan was created;
- the scheduler emits an operational alert "trigger T failed
  to deliver after N attempts" (per B0-4 CC7 operational
  category); the alert payload includes the would-be
  `execution_id` (computable from the trigger's inputs) for
  forensic linkage with subsequent successful executions of
  the same trigger.

The scheduler may schedule the trigger again on its next
scheduled cadence; that next trigger uses identical inputs and
therefore the identical `execution_id`, and produces a fresh
attempt under that id (per B0-2 CC3 and B0-3 CC4's UUID
`attempt_id`).

**CC7. Check-level retries are bounded and produce
`result = error` on exhaustion.** When a check's evaluation
fails with a transient error (BigQuery quota, network blip,
intermittent timeout):

- the runner retries the check up to a configurable maximum
  (foundation doc 05 sketches two with exponential backoff;
  B1 picks — see OQ-1);
- on exhaustion, the check's row in `dq_check_results` is
  written with `result = error` (per B0-4 CC1's classification
  of "exceeded retry budget for transient errors");
- the execution proceeds with other checks (per B0-4 CC4
  always-continue);
- the execution-status mapping (B0-4 CC2) is computed at
  execution finalization from the resulting result multiset.

Non-transient errors (query compilation error, missing source
detected mid-check, type mismatch) skip retries — they fail
immediately to `result = error` because retrying would not
change the outcome. The transient-vs-non-transient
classification is Wave 3 implementation detail (a small
allowlist of error types treated as transient); CC7 commits
only that the classification exists.

**CC8. Pre-check validation runs at plan creation, validates
source-table existence only — lightweight by contract.** When
the trigger handler creates an execution plan (foundation doc
04 §"Execution Flow" step 2), it determines source-table
existence for the entity via a **lightweight pre-check
operation** — substantially cheaper than evaluating any
individual check; the operation must not dominate
plan-creation latency. If the determination is "source not
present", the trigger handler writes the execution row
directly with `status = error` (per B0-4 CC5) and no check
rows are produced.

The exact mechanism (a metadata-read API call such as BigQuery
`tables.get`, an information_schema query, a cached-metadata
layer, derivation from B0-1's load-time validations, or
another approach) is Wave 3 implementation — see OQ-8. CC8
commits only: (a) the determination happens at plan time,
(b) it covers source-table existence only — other potential
pre-checks (partition column presence, source-table schema
compatibility with the rule) are **not** validated pre-check
and surface as check-level `result = error` per CC7, and
(c) the operation is lightweight (the cost-bound is
operational).

This is deliberate: only the "every-check-will-error" case
warrants a pre-check shortcut; partial-evaluability conditions
go through the regular check path.

B0-1 C6 contract checks (the three checks at engine load) do
**not** repeat at plan time. They are already enforced at
load time per CC1 / CC2; an in-memory manifest reaching plan
time has already passed them.

**CC9. Refresh cadence is periodic with hash short-circuit.**
The engine re-fetches `manifests/latest.json` on a configurable
interval (B1 picks the default — foundation doc 05's existing
references suggest 60-second-class intervals; OQ-4). On each
fetch:

- the engine compares the pointer's `manifest_hash` to the
  currently-loaded manifest's hash;
- if they are equal, the engine **does not** re-fetch the
  manifest body — the active ruleset is up-to-date by
  definition (B0-5's content-addressing guarantees the manifest
  body has not changed if the hash has not changed). This is
  the short-circuit.
- if they differ, the engine fetches the new manifest body,
  runs the load pipeline (B0-1 contract checks, B0-5 hash
  verification, PAT-1 fail-fast), and either swaps to the new
  manifest (success) or executes CC2 (failure).

Pointer-only reads are cheap (a small JSON object); the
short-circuit makes the steady state — same manifest active
indefinitely — essentially free. This resolves B0-5 OQ-4's
framing question in favor of "always short-circuit when
pointer hash matches".

**Dependency on B0-5 immutability.** The short-circuit's
safety depends on B0-5 CC2 (manifests are immutable) holding
in practice — i.e., `by-hash/` objects are not overwritten and
the pointer is not forged. B0-5 CC2 commits publisher
discipline (the publisher never overwrites); substrate-level
enforcement of immutability (object versioning, retention
locks) is a B0-5 CC13 / W2-1 substrate-selection concern. In
a normal operational threat model (controlled write access, CI
is the only writer to the manifest store), the assumption
holds and the short-circuit is sound. If B0-5 OQ-3
(cryptographic posture beyond checksums) is later strengthened
— e.g., manifest signing — CC9's short-circuit can rely on a
cryptographic guarantee rather than operational discipline,
and the dependency is discharged. CC9 commits the short-circuit
**conditional on B0-5's immutability assumption holding**; any
future revision of B0-5 OQ-3 should revisit this dependency.

**CC10. `status = aborted` halt conditions are enumerated.**
B0-4 CC2 branch 1 ("global engine halt → aborted") is
populated by exactly the following conditions:

- **Cost ceiling exceeded mid-execution.** The per-run
  bytes-scanned ceiling (foundation doc 05 §"Cost Discipline")
  is breached; the engine halts the in-flight execution and
  writes a follow-up row with `status = aborted`.
- **Engine restart during in-flight execution.** When the
  engine process itself is restarted (planned shutdown,
  container restart, OOM kill, startup-failure cascade on a
  graceful redeploy, etc.) while an execution is in-flight,
  the `running` row is left dangling and is finalized by
  orphan-run detection (CC11) as `status = aborted`. Per
  CC3, refresh failure alone does not cause this — only an
  actual process-level restart does. The orphan-detection
  mechanism is what couples this CC10 bullet to a concrete
  `status = aborted` outcome.
- **Container OOM / crash mid-execution.** The engine process
  dies; the `running` row is left dangling; CC11 finalizes it.
- **Global resource limit.** Per-environment concurrency
  budget (foundation doc 05 §"Cost Discipline" — "concurrency
  budgets per environment") exceeded for long enough that the
  execution is force-evicted from the run queue. Exact
  threshold is B1.
- **Operator-issued abort.** An Admin API endpoint (Wave 3)
  allows operators to abort a specific in-flight execution;
  the abort writes a follow-up row with `status = aborted`
  and the operator's identity in the audit fields.

All other failures route through CC1 / CC2 (loader),
CC4 (scheduler reconciliation), CC6 (trigger-handler retries),
CC7 (check-level retries), or CC8 (pre-check) — none of those
produce `aborted`.

**CC11. Orphan-run detection.** A periodic engine task scans
`dq_executions` (via `dq_executions_current` per B0-3 CC2) for
rows with `status = running` and `started_at` older than a
configurable threshold (B1 — see OQ-5; the threshold should
exceed the maximum expected execution duration in the
environment). For each row matching the criterion, the engine
writes a follow-up row (per B0-3 CC1 append-only) with the
same `execution_id` and `attempt_id`, `status = aborted`, and
`error_summary = "engine instance abandoned this execution"`
(exact wording Wave 3).

The orphan-aborted row carries the **orphan-detector engine
instance's `engine_version`**, not the abandoned engine's —
different `engine_version` values **within a single attempt's
lifecycle rows** is the expected pattern for the
orphan-finalization case. B0-2 CC14's visibility commitment
applies: forensic queries see the abandoned engine's
`engine_version` on the `running` row and the orphan-detector's
`engine_version` on the `aborted` row; both are accurate,
neither is overwritten. The `dq_executions_current` canonical
view per B0-3 CC2 returns the latest row, so the
orphan-detector's `engine_version` appears as the "effective
evaluator" in the canonical projection — operators reading
the canonical view see the orphan-detector's version, and
forensic queries on the base table see the abandonment event.

This handles the container-OOM / crash case (CC10 third
bullet) without requiring the engine that crashed to write
its own death certificate.

**CC12. B0-1 C6 failure mode is process-exit-at-startup,
refuse-swap-at-refresh.** B0-1 deferred the failure mode to
B0-7. This study commits it: B0-1 contract check failures
flow through CC1 (startup) or CC2 (refresh). There is no
third failure mode. This commitment is symmetric with B0-5
hash verification failures (CC13) — both flow through the
same loader-failure path.

**CC13. B0-5 hash verification failure mode matches CC12.**
Manifest content-hash mismatch, YAML hash mismatch, and
dangling-pointer cases (pointer hash not present in
`by-hash/` per B0-5 CC10's orphan scenario inverted) are all
loader failures and flow through CC1 / CC2. The engine does
not attempt any salvage path; the failure is loud and the
remediation is upstream (re-publish or rollback per B0-5
CC5).

**CC14. Observability emission is committed.** Every failure
path from CC1 / CC2 / CC4 / CC6 / CC7 / CC8 / CC10 / CC11
emits, at minimum:

- a structured log line with `execution_id` (when applicable),
  `entity`, `check_id` (when applicable), and a failure-type
  identifier;
- a counter increment (Wave 3 picks names; aligned with
  foundation doc 05 §"Observability Contract");
- an OpenTelemetry span recording the failure with attributes
  matching the log fields.

The exact emission shape is Wave 3, but B0-7 commits that
every failure path emits all three (log + metric + span). No
failure is silent (D5 enforced).

**CC15. Numeric parameters are B1.** The following are
explicitly deferred to B1 with the per-environment
differentiation appropriate to each:

- Trigger-handler retry maximum count (CC6).
- Check-level retry maximum count (CC7).
- Check-level retry backoff schedule (CC7).
- Refresh cadence default (CC9).
- N consecutive refresh-failure escalation threshold (CC2).
- M consecutive scheduler-reconciliation-failure threshold
  per trigger (CC4).
- Orphan-run detection threshold age (CC11).
- **Orphan-run detection scan cadence (CC11)** — how often the
  periodic scan runs; must be frequent enough that the
  detection threshold is operationally meaningful but not so
  often that it dominates engine workload.
- Cost ceiling for `aborted` (CC10 first bullet).
- Concurrency-budget eviction threshold (CC10 fourth bullet).

B0-7 commits the **shapes** of these parameters (bounded,
configurable, per-environment-tunable); foundation doc 05
sketches order-of-magnitude defaults; B1 commits the actual
values.

---

## Open Questions

- **OQ-1. Numeric retry maxima.** Trigger-handler retry count
  and check-level retry count (with backoff schedule) are
  **out-of-scope for current cycle** — B1, picking
  per-environment values. Foundation doc 05 sketches three and
  two respectively; CC6 and CC7 commit only "bounded,
  configurable".

- **OQ-2. N consecutive refresh-failure escalation threshold.**
  The number of consecutive refresh failures that escalates a
  WARN-level operational alert to high-severity is
  **out-of-scope for current cycle** — B1, likely
  per-environment (more sensitive in prod, less in qa).

- **OQ-3. M consecutive scheduler-reconciliation-failure
  threshold.** Per-trigger threshold before an operational
  alert is emitted is **out-of-scope for current cycle** — B1,
  per-environment.

- **OQ-4. Refresh cadence default.** Suggested order of
  magnitude is 60 seconds per foundation doc 05's framing, but
  the exact default and per-environment overrides are
  **out-of-scope for current cycle** — B1.

- **OQ-5. Orphan-run detection threshold age.** The age above
  which a `running` row is treated as orphaned is
  **out-of-scope for current cycle** — B1, must exceed the
  maximum expected execution duration in the environment.

- **OQ-6. Transient-vs-non-transient error classification for
  check-level retries.** The exact allowlist of error types
  treated as transient (eligible for retry) versus non-transient
  (immediate `result = error`) is **out-of-scope for current
  cycle** — Wave 3 implementation. CC7 commits only that the
  classification exists and that non-transient errors skip
  retries.

- **OQ-7. Admin API surface for scheduler status and operator
  abort.** The shape of the Admin API endpoints that expose
  scheduler reconciliation status (CC4) and allow operator
  abort (CC10 fifth bullet) is **out-of-scope for current
  cycle** — Wave 3 implementation.

- **OQ-8. Exact metadata-read mechanism for pre-check
  source-table existence.** Whether the engine uses BigQuery
  `tables.get` API, an information_schema query, or a cached
  metadata layer is **out-of-scope for current cycle** — Wave 3
  implementation. CC8 commits only that the read happens once
  per execution at plan time and that "not found" produces
  `status = error`.

- **OQ-9. Structured-log field names, metric names, span
  names.** The exact names of the observability emissions
  committed in CC14 are **out-of-scope for current cycle** —
  Wave 3, informed by foundation doc 04 PAT-5 and §"Observability
  Contract".

No open question in this list blocks the failure-semantics
shape. All items above are parameters or implementation
details on top of the locked policy.

---

## Promotion target

This study is promoted during Wave 3 to:

    docs/adr/0007-loader-scheduler-retry-failure-semantics.md

The `0007` is provisional and assigned at promotion time. If
the Wave 3 ADR numbering convention orders by promotion date
rather than by B0 sequence, the number changes; the slug
(`loader-scheduler-retry-failure-semantics`) does not. This
follows the same convention adopted in
[`2026-05-20-engine-rules-compatibility.md`](./2026-05-20-engine-rules-compatibility.md)
(B0-1, Promotion target section).

The MADR ADR rewrites this study for an external-reviewer
audience (no `studies/` back-references per R8), folds in any
updates from B0-6 (alert routing) that intersect with the
operational alerts emitted by these failure paths, and updates
the relevant sections of foundation doc
[`04-system-architecture.md`](../foundation/04-system-architecture.md)
(PAT-1, PAT-2) and
[`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
(§"Retry Semantics", §"Failure Scope") to reference the ADR's
locked policy.

A Wave 3 operations document under `docs/operations/` will
operationalize this policy into runbook entries, complementing
the runbook authored against B0-4 (CC11 of B0-4); the two
documents share the operator-response framing introduced by
B0-4.
