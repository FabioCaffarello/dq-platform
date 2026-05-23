<!-- path: docs/glossary.md -->

# Glossary

Terms with **codebase-specific** meaning in the DQ Platform.
This document is for readers who already know Go, HTTP,
Kubernetes, and BigQuery and need to know what *this*
repository means by `execution_id`, `manifest`, `refuse-swap`,
or `_owners.yaml`.

Generic engineering vocabulary is intentionally absent: words
that mean the same thing here as anywhere else are not glossary
entries.

Each entry cites the artifact that defines it (an ADR, a
foundation document, or a decision-log row). Citations are
forward-only — see [`docs/README.md`](README.md) for the
reading conventions.

---

## How to read this glossary

- Terms are grouped by thematic surface, not alphabetized. The
  group order roughly follows the platform's data flow:
  governance → boundary → identity → storage → failure →
  manifest → loader/runner → trigger plane → alerting → DSL →
  environment.
- An *Alphabetical index* at the bottom lists every term with a
  jump-link.
- Where a term has more than one meaning in the wild (`status`,
  `result`, `manifest`), the entry pins the meaning *this
  platform* commits to.

---

## 1. Governance & process

### ADR
**Architecture Decision Record.** A MADR-style document under
[`docs/adr/`](adr/) capturing one architectural decision: its
context, the decision, and the consequences. ADRs are
forward-only: an ADR never links backward into `studies/`
(per `CLAUDE.md` R8). — *Source:*
[ADR-0011](adr/0011-documentation-language.md),
[`CLAUDE.md` R8](../CLAUDE.md).

### Study
A reasoning document under `studies/decisions/` capturing how
a decision was reached. Studies are scaffolding for ADRs and
are **not** part of the published repository product. Once a
study stabilizes, it is rewritten as an ADR; the study
remains for historical reasoning but is never cited from a
published artifact. — *Source:* `CLAUDE.md` R8.

### Resolved-study / resolved-adr
Two consecutive states a decision-log row can occupy.
`resolved-study` means a draft document exists in
`studies/decisions/`; `resolved-adr` means that study has been
promoted to a numbered ADR under `docs/adr/`. Both states
unblock downstream work; only `resolved-adr` is permanent.
— *Source:* `studies/foundation/06-decision-log.md`.

### B0 / B1 / B2
Decision priority tiers in the decision log. **B0** rows
*block* downstream scaffolding work and must resolve before
any related code lands. **B1** rows are quality-of-life
decisions that scaffolding sessions resolve on demand. **B2**
rows are long-tail decisions that resolve as the platform
matures. — *Source:*
`studies/foundation/06-decision-log.md`.

### W1 / W2 / W3
**Waves.** The three sequential operating phases of the
repository. Wave 1 resolved the blocking B0 decisions; Wave 2
resolved the cross-cutting platform decisions (`W2-1`…`W2-5`);
Wave 3 scaffolds every workspace. Each wave has an explicit
gate that requires human approval to cross. — *Source:*
`CLAUDE.md` §2, [ADR-0013](adr/0013-wave3-sequencing.md).

### Phase (W3-Pn)
A subdivision of Wave 3. Phases are numbered `W3-P0` …
`W3-P8` and ordered by dependency. A phase may split into
sub-phases (`W3-P4a`, `W3-P7c`, etc.) when the scope is too
large for one session. Sub-phase splits are recorded in the
decision-log row of the parent phase. — *Source:*
[ADR-0013](adr/0013-wave3-sequencing.md).

### CC (Commit Criterion)
A numbered, binary, verifiable statement appearing in a B0
resolved-study (`B0-2 CC1`, `B0-3 CC7`, etc.) that the
implementing scaffold must satisfy. CCs are the unit
referenced by scaffolding commit messages and code comments
when an ADR is the upstream contract. — *Source:* every B0
resolved-study under `studies/decisions/`.

### C-W2-N
A numbered commit invariant from one of the Wave 2 ADRs (e.g.
`C-W2-3.4` is invariant `3.4` from ADR-0010). Same role as a
CC but sourced from a W2 ADR rather than a B0 study. —
*Source:* [ADRs 0008–0012](adr/).

### AC-W3-N
A row in the Wave 3 acceptance criteria
(`.claude/playbooks/wave-3-acceptance-criteria.md`). Every
Wave 3 scaffolding unit must mark each AC-W3 row as **pass**,
**fail**, or **deferred with a marker** before opening its
pull request. — *Source:*
`.claude/playbooks/wave-3-acceptance-criteria.md`.

---

## 2. Schema, boundary, and the engine-rules contract

### Rule artifact
A YAML file under `rules/<entity>.yaml` declaring checks for
one entity. Every rule artifact carries a top-level
`version:` field naming the schema version it was written
against; missing `version:` is a hard error at the linter.
— *Source:* [ADR-0001 §1](adr/0001-engine-rules-compatibility.md).

### Ruleset
The set of rule artifacts that the engine evaluates at a
given moment. Identified at runtime by `ruleset_version`.
A ruleset is published as a *manifest* (see §6). — *Source:*
[ADR-0001](adr/0001-engine-rules-compatibility.md),
[ADR-0005](adr/0005-manifest-publication-semantics.md).

### `ruleset_version`
The published version identifier of a ruleset, formatted as
`rules-v<major>.<minor>.<patch>` per the W2-5 tag
conventions. Treated as opaque text by the engine. Forbidden
to contain the ASCII pipe character (it participates in the
`execution_id` formula). — *Source:*
[ADR-0002 §2](adr/0002-run-identity-and-idempotency.md),
[ADR-0012](adr/0012-tag-conventions.md).

### Engine schema
The canonical JSON Schema for the rule DSL, living under
`engine/internal/dsl/schema/v<N>.schema.json`. This is the
**source of truth**; every other copy is a mirror generated
mechanically from it. — *Source:*
[ADR-0001 §2](adr/0001-engine-rules-compatibility.md).

### Rules schema mirror
A byte-identical copy of the engine schema under
`rules/_schema/v<N>.schema.json`. The mirror exists so the
rules workspace is lintable without depending on engine
internals at lint time. **Never edited by hand.** — *Source:*
[ADR-0001 §2](adr/0001-engine-rules-compatibility.md).

### Byte-equality gate
A mandatory CI check that runs `cmp` / `diff` between each
engine schema and its rules-workspace mirror on every push
and every pull request. Any divergence blocks merge. Cannot
be downgraded to advisory. — *Source:*
[ADR-0001 §2](adr/0001-engine-rules-compatibility.md).

### Engine compatibility expression
A semver range carried in the manifest body declaring which
engine releases support every schema version present in the
ruleset. The engine refuses to load a manifest whose
expression excludes its running version. — *Source:*
[ADR-0001 §3](adr/0001-engine-rules-compatibility.md),
[ADR-0005 §5](adr/0005-manifest-publication-semantics.md).

### `linter_used`
A manifest field naming the linter release that validated
the ruleset. **Audit-only** — the engine does not read or
verify it at load time. The pin must be unforgeable in CI
configuration. — *Source:*
[ADR-0001 §3 / §5](adr/0001-engine-rules-compatibility.md).

---

## 3. Run identity and idempotency

### Run
One evaluation of a specific entity's rules over a specific
time window, triggered by a specific source. Every run has
exactly one `execution_id` and one or more `attempt_id`s.
— *Source:*
[ADR-0002 Context](adr/0002-run-identity-and-idempotency.md).

### `execution_id`
The deterministic identifier of a run. Computed as
`sha256_hex(ruleset_version || entity || window_start ||
window_end || trigger_source)` with `|` as separator and no
escaping. 64 characters of lowercase hex. **Opaque to
consumers** — do not parse, prefix-match, or derive metadata
from the bits. — *Source:*
[ADR-0002 §1](adr/0002-run-identity-and-idempotency.md).

### `attempt_id`
A UUID assigned by the trigger handler at the start of each
attempt of a given `execution_id`. A scheduler retry of the
same `execution_id` assigns a **new** `attempt_id`. Opaque,
non-sortable — across-attempt ordering uses `recorded_at`,
not `attempt_id`. — *Source:*
[ADR-0003 §4](adr/0003-result-write-model.md).

### `trigger_source`
The closed enum identifying who originated a run. Committed
values: `scheduler`, `manual`, `operator-rerun`. The trigger
API accepts only `scheduler` and `manual`; `operator-rerun`
is producible only by the Admin API rerun endpoint, enforced
at the API layer. — *Source:*
[ADR-0002 §3 / §4](adr/0002-run-identity-and-idempotency.md).

### Window
The time slice over which a run evaluates. Identified by
`(window_start, window_end)`, both RFC 3339 UTC timestamps
with second precision (`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`).
Fractional seconds are excluded. — *Source:*
[ADR-0002 §2](adr/0002-run-identity-and-idempotency.md).

### `engine_version`
The version of the engine binary that wrote a given row.
**Not** in the `execution_id` formula — it is per-attempt
metadata, recorded on every row and actively surfaced in
canonical reporting projections. A scheduler retry that spans
an upgrade produces two attempt rows with the same
`execution_id` and different `engine_version` values. —
*Source:*
[ADR-0002 §1 / Consequence 4](adr/0002-run-identity-and-idempotency.md).

### `supersedes_execution_id`
Nullable column on `dq_executions` populated only on the
first state-transition row of an `operator-rerun` attempt.
Its value is the `execution_id` of the prior run being
superseded. Chains of operator reruns are traversable via
self-join on this column — no separate lineage table. —
*Source:*
[ADR-0003 §5](adr/0003-result-write-model.md).

### Scheduler retry vs. operator rerun
**Scheduler retry** — identical five inputs ⇒ identical
`execution_id`; new `attempt_id`. **Operator rerun** —
`trigger_source = operator-rerun` differs from the original,
so the formula produces a *new* `execution_id`; the new row
carries `supersedes_execution_id` pointing at the original.
— *Source:*
[ADR-0002 §5](adr/0002-run-identity-and-idempotency.md).

---

## 4. Storage — append-only with a lazy canonical view

### `dq_executions`
Append-only run-level table. Composite logical key
`(execution_id, attempt_id, recorded_at)`. One row per state
transition. The engine performs **only `INSERT`** — no
`UPDATE`, no `DELETE` from any engine code path. — *Source:*
[ADR-0003 §1 / §3](adr/0003-result-write-model.md).

### `dq_check_results`
Append-only per-check table. Composite logical key
`(execution_id, attempt_id, check_id)`. One row per
(attempt × check). Same append-only discipline as
`dq_executions`. — *Source:*
[ADR-0003 §1 / §7](adr/0003-result-write-model.md).

### `dq_executions_current`
The **lazy** canonical view over `dq_executions`. Projects,
per `execution_id`, the row with the latest `recorded_at`.
Computed at query time — not materialized — to avoid
refresh-lag and derived-state-divergence failure modes.
Dashboards and alerting target this view; forensic queries
target the base table. — *Source:*
[ADR-0003 §2](adr/0003-result-write-model.md).

### Append-only
The platform's storage discipline: only `INSERT` from engine
code paths. Operator corrections (data purges, retroactive
fixes) go through out-of-band paths, not engine SQL. —
*Source:* [ADR-0003 §1 / Consequence 1](adr/0003-result-write-model.md).

### `evidence_summary`
A structured aggregate-counts field on `dq_check_results`
(rows scanned, rows failing, etc.). Distinct from
`sample_violating_rows`, which holds repeated records capped
at a configured limit. — *Source:*
[ADR-0003 §7](adr/0003-result-write-model.md).

---

## 5. Failure model

### Check `result`
The outcome of one check. Closed enum: `pass`, `fail`,
`degraded`, `error`. `pass`/`fail`/`degraded` mean the query
executed; `error` means it did not (compilation, missing
source, quota exhaustion, retry-budget exhaustion,
evaluation-budget timeout). — *Source:*
[ADR-0004 §1](adr/0004-failure-scope.md).

### Execution `status`
The outcome of one run. Closed enum: `running`, `success`,
`failed`, `error`, `aborted`. Computed by a **pure function**
of the check-result multiset — no operator discretion. —
*Source:* [ADR-0004 §2](adr/0004-failure-scope.md),
[ADR-0003 §6](adr/0003-result-write-model.md).

### Failure scope
The committed mapping from check-level `result` values to
execution-level `status`. Documented as a five-branch
mutually-exclusive decision procedure with one rule: only
"every check errored" (or a pre-check entity-level problem)
reaches `status = error`. Mixed-result executions stay at
`failed`. — *Source:*
[ADR-0004 §2 / §3](adr/0004-failure-scope.md).

### Continuation rule
The engine **always continues at the check level**: every
check listed in the manifest is attempted regardless of
sibling-check outcomes. A check-level `error` never aborts an
in-flight execution. — *Source:*
[ADR-0004 §4](adr/0004-failure-scope.md).

### Pre-check entity-level problem
An entity-level problem the engine detects **before** any
check has been attempted (e.g., source table missing). The
execution row is written directly with `status = error` and
no `dq_check_results` rows are produced. This is the only
path to `status = error` without all checks producing `error`
results. — *Source:*
[ADR-0004 §5](adr/0004-failure-scope.md),
[ADR-0007 §9](adr/0007-loader-scheduler-retry-failure-semantics.md).

### `status = aborted`
Reserved for **global engine halts** during in-flight
execution. The exhaustive set: cost ceiling exceeded mid-run,
engine restart, container OOM, global resource-limit
eviction, operator-issued abort. All other failures route
through paths that produce `status = error` or no row at all
— not `aborted`. — *Source:*
[ADR-0007 §10](adr/0007-loader-scheduler-retry-failure-semantics.md).

---

## 6. Manifest plane

### Manifest
The derived JSON artifact the engine consumes at runtime to
know which rules to evaluate, at which schema versions, and
under which engine-compatibility expression. **Derived, not
authored** — CI builds it from rule YAMLs and publishes it to
object storage. — *Source:*
[ADR-0005 Context / §5](adr/0005-manifest-publication-semantics.md).

### Pointer file
The single mutable control-plane object,
`manifests/latest.json`. Every "publish a new manifest" and
every "rollback" is **exactly one generation-conditional
write** (CAS) to this file. — *Source:*
[ADR-0005 §3 / §6](adr/0005-manifest-publication-semantics.md).

### Manifest body
The immutable JSON document at
`manifests/by-hash/sha256-<hex>.json`, referenced by the
pointer file. Carries the three semantic commitments from
the engine-rules contract: `schema_versions_present`,
`engine_compatibility`, `linter_used`. — *Source:*
[ADR-0005 §1 / §5](adr/0005-manifest-publication-semantics.md).

### Manifest publication
The publisher's atomic four-step sequence: (1) verify the
three pre-publish checks from ADR-0001, (2) write rule YAMLs
to `yamls/by-hash/`, (3) write the manifest body to
`manifests/by-hash/`, (4) issue a generation-conditional
write to `manifests/latest.json`. — *Source:*
[ADR-0005 §4](adr/0005-manifest-publication-semantics.md).

### Generation-conditional write (CAS)
A write to the pointer file that succeeds only if the
object's generation has not changed since the publisher read
it. Two concurrent publishers cannot both succeed; the loser
receives a precondition-failed error and surfaces it as a
publication failure. **Local emulators that do not enforce
this are tagged `Partial` in the substrate-posture matrix**
(see B1-11 for the amendment). — *Source:*
[ADR-0005 §3 / §4](adr/0005-manifest-publication-semantics.md).

### Content-addressed (by-hash)
Storage discipline for manifests and YAMLs: every object is
keyed by `sha256-<hex>` of its content. The hash algorithm is
encoded in the path prefix so a future algorithm migration is
a coexistence of differently-prefixed paths, not a path
rewrite. Manifests and YAMLs at `by-hash/` paths are
**immutable** — never modified, never deleted by the
publisher (lifecycle deletion is operator-driven). — *Source:*
[ADR-0005 §1 / §2 / §7](adr/0005-manifest-publication-semantics.md).

### Orphan hash
A `by-hash/` object whose publisher wrote it but then lost
the pointer CAS race. Unreachable from any pointer. Costs
only its storage footprint until lifecycle policy purges it.
**Not corrupting.** — *Source:*
[ADR-0005 Consequence 7](adr/0005-manifest-publication-semantics.md).

---

## 7. Loader, runner, scheduler

### Loader
The engine subsystem that reads the active manifest at
startup and at periodic refresh, runs the compatibility
checks from ADR-0001, and indexes rules by entity. Implements
the fail-fast registry pattern PAT-1. — *Source:*
[ADR-0007 Context / §1](adr/0007-loader-scheduler-retry-failure-semantics.md),
foundation 04 §PAT-1.

### PAT-1 — Fail-fast registry loading
The loader contract: read the manifest, fetch every YAML,
validate, index by entity, fail fast on duplicate entity
keys / schema validation / checksum mismatch / missing YAMLs.
**No partial loading. No silent skipping.** — *Source:*
foundation 04 §PAT-1.

### Startup-mode load
The loader's first manifest load. **Any failure** causes the
engine process to exit non-zero with a structured final log
line. Ops sees the crash-loop in container orchestration. —
*Source:*
[ADR-0007 §1](adr/0007-loader-scheduler-retry-failure-semantics.md).

### Refuse-swap
The loader's refresh-mode posture: when a refresh attempt
fails, **retain the prior manifest** as the active reference
(no swap), emit log + metric + operational alert, continue
serving trigger requests. The data plane is unaffected by
refresh failures. — *Source:*
[ADR-0007 §2](adr/0007-loader-scheduler-retry-failure-semantics.md).

### Hash short-circuit
The loader's steady-state optimization: on each refresh
cycle, compare the pointer's `manifest_hash` to the
currently-loaded manifest's hash. If equal, **skip the body
fetch** — content addressing guarantees the body is unchanged
if the hash is unchanged. — *Source:*
[ADR-0007 §4](adr/0007-loader-scheduler-retry-failure-semantics.md).

### In-flight execution isolation
A trigger accepted at time `T` evaluates against the manifest
active at `T`, even if a refresh swaps the active manifest
mid-execution. The captured `manifest_hash` lives as
**in-memory execution-context state**, not as a persisted
column on `dq_executions`. Forensic linkage uses
`ruleset_version` (which *is* persisted and is in the
`execution_id` formula). — *Source:*
[ADR-0007 §3](adr/0007-loader-scheduler-retry-failure-semantics.md).

### Runner
The engine subsystem that, given an accepted trigger, plans
the execution, dispatches per-check evaluations, applies the
failure-scope mapping, and writes terminal rows to
`dq_executions` and `dq_check_results`. — *Source:*
[ADR-0003 §4](adr/0003-result-write-model.md),
[ADR-0004 §2 / §4](adr/0004-failure-scope.md).

### Orphan trigger
A trigger present in the external scheduler but not in the
active manifest's rule set, **and** carrying the engine's
marker. The reconciliation loop deletes orphan triggers
silently — no alert. **The engine never modifies or deletes
triggers it did not create.** — *Source:*
[ADR-0007 §6](adr/0007-loader-scheduler-retry-failure-semantics.md).

### Orphan run
A `dq_executions` row with `status = running` whose
`started_at` is older than a configurable threshold. The
orphan detector writes a follow-up row with the same
`(execution_id, attempt_id)`, `status = aborted`, and an
`error_summary` identifying engine abandonment. — *Source:*
[ADR-0007 §11](adr/0007-loader-scheduler-retry-failure-semantics.md).

### Orphan detector
The periodic engine task that scans for orphan runs and
finalizes them. The follow-up row carries the
**orphan-detector instance's** `engine_version`, not the
abandoned engine's — making the abandonment event explicit
in forensic queries. — *Source:*
[ADR-0007 §11 / Consequence 9](adr/0007-loader-scheduler-retry-failure-semantics.md).

### Trigger-handler retry exhaustion
When the external scheduler cannot deliver a trigger after
the bounded retries, **no `dq_executions` row is written** —
the trigger never reached the engine. The scheduler emits an
operational alert with the would-be `execution_id` for
forensic linkage with later successful executions of the
same trigger. — *Source:*
[ADR-0007 §7](adr/0007-loader-scheduler-retry-failure-semantics.md).

### Check-level retry exhaustion
When a check's evaluation fails the bounded retries, the
check's row in `dq_check_results` is written with
`result = error` and the execution proceeds with sibling
checks (continuation rule). — *Source:*
[ADR-0007 §8](adr/0007-loader-scheduler-retry-failure-semantics.md).

---

## 8. Trigger plane (HTTP surface)

### `POST /v1/trigger`
The engine's single HTTP entry point for accepting triggers
from external schedulers, operators, and manual invocation.
Accepts exactly `trigger_source ∈ {scheduler, manual}`;
`operator-rerun` is rejected with `400`. — *Source:*
[ADR-0014 §2 / Consequence 3](adr/0014-trigger-handler-contract.md).

### Trigger handler — strict decoder
The decode posture: UTF-8 validity, ASCII-pipe absence, RFC
3339 UTC regex on the two timestamps, closed-enum check on
`trigger_source`, per-field length ceiling, **unknown fields
return `400`** (no silent drops). All enforced *before*
`execution_id` is computed. — *Source:*
[ADR-0014 §2](adr/0014-trigger-handler-contract.md).

### Eager-at-load hydration
The handler binds its HTTP listener **only after** the
loader's startup-mode load completes successfully. Lazy
per-trigger hydration is forbidden — it would permit a
manifest swap between input decode and plan dispatch (a race
the in-flight isolation guarantee forbids). — *Source:*
[ADR-0014 §1](adr/0014-trigger-handler-contract.md).

### API DTO (`/v1/trigger` response)
The response shape, distinct from the `dq_executions`
persistence row. Initial v1 field set:
`{execution_id, attempt_id, status, accepted_at, self}`.
DTO additions land under minor ADR amendments; breaking
changes require a new ADR. — *Source:*
[ADR-0014 §3](adr/0014-trigger-handler-contract.md).

### `accepted_at`
Handler-side timestamp at the point of trigger acceptance.
**Distinct from** `started_at` (the persistence column) —
the contract permits the two to diverge if plan creation has
meaningful latency. — *Source:*
[ADR-0014 §3 / Consequence 9](adr/0014-trigger-handler-contract.md).

### `/healthz` (liveness)
Returns `200 OK` while the process is up. Does **not** depend
on manifest state. Liveness probes use this endpoint. —
*Source:*
[ADR-0014 §4](adr/0014-trigger-handler-contract.md).

### `/readyz` (readiness)
Returns `200 OK` once the first successful manifest load
completes and stays `200` for the lifetime of the process
unless the process exits. **Refresh failures do not flip
`/readyz`** — the prior-manifest continues to serve under
refuse-swap, so a readiness flip would defeat data-plane
isolation. — *Source:*
[ADR-0014 §4 / Consequence 11](adr/0014-trigger-handler-contract.md).

---

## 9. Alerting

### `_owners.yaml`
The rules-workspace file declaring per-entity ownership and
routing references. The linter rejects any entity declared in
a rule file without a corresponding `_owners.yaml` entry —
"no alert without owner" is enforced at author time. —
*Source:*
[ADR-0006 §1 / §9](adr/0006-alert-routing-contract.md).

### Owner entry
One entry in `_owners.yaml`'s `entities` map. Required
fields: `owner` (CODEOWNERS group), `channels` (per-category
list of channel references). Optional: `severity_overrides`,
`description`. — *Source:*
[ADR-0006 §1](adr/0006-alert-routing-contract.md).

### Channel reference
A `(type, id)` pair declaring where alerts route. `type` is a
destination-class identifier (`slack`, `pagerduty`, `email`,
`webhook`, …) matching the engine deployment config's
channel-resolution table. `id` is the
environment-resolvable identifier humans read in review; the
concrete destination (webhook URL, service key) lives in
deployment config, not in `_owners.yaml`. — *Source:*
[ADR-0006 §2 / §3](adr/0006-alert-routing-contract.md).

### Alert category
The binary classification the engine sets on every event:
`data_quality` or `operational`. The mapping is fixed by
ADR-0004 and ADR-0006 §7 — the engine never deviates and
routing has no discretion to reassign categories. Check
`fail`/`degraded` → `data_quality`; check `error`,
execution `status` ∈ {`error`, `aborted`}, and all
loader/scheduler/trigger-handler/orphan-finalization
failures → `operational`. — *Source:*
[ADR-0004 §6](adr/0004-failure-scope.md),
[ADR-0006 §7](adr/0006-alert-routing-contract.md).

### Event payload
The structured event the engine publishes to Pub/Sub for
every alert-relevant action. Required fields: `entity`,
`category`, `event_source`, `recorded_at`. Optional fields
include `execution_id`, `attempt_id`, `check_id`, `result`,
`status`, `severity`, `error_summary`. Consumers must
tolerate unknown fields (additive payload evolution). —
*Source:*
[ADR-0006 §4](adr/0006-alert-routing-contract.md).

### Two-layer deduplication
The platform's dedup model. **Engine-side**: suppresses
literal-duplicate `(execution_id, attempt_id, check_id,
result)` tuples within an attempt. **Consumer-side**: enforces
"≤ 1 user-visible alert per failing check across N retries"
via a configurable window. Check-level dedup key
**excludes** `result` — a check fluctuating `fail` → `error`
→ `fail` collapses to one alert. — *Source:*
[ADR-0006 §5](adr/0006-alert-routing-contract.md).

### `event_source`
A required event-payload enum naming which engine subsystem
produced the alert: `runner`, `loader`, `scheduler`,
`orphan_detector`, `trigger_handler`. Used as part of the
execution-level dedup key so different components reporting
the same execution surface separately. — *Source:*
[ADR-0006 §4 / §5](adr/0006-alert-routing-contract.md).

---

## 10. DSL

### DSL
The declarative rule language. Three non-negotiable
properties: **rules are declarative** (P1 — no raw SQL, no
embedded expressions, no escape hatches), **engine behavior
is deterministic** (P2), and **evolution is contract-driven**
(P5). — *Source:* foundation 01 §P1 / §P2 / §P5,
[ADR-0001](adr/0001-engine-rules-compatibility.md).

### Check
One declarative validation inside a rule artifact. Identified
by `check_id` within its rule. Produces exactly one `result`
value per attempt. — *Source:*
[ADR-0004 §1](adr/0004-failure-scope.md),
[ADR-0003 §7](adr/0003-result-write-model.md).

### Check kind
The discriminator that selects which evaluator runs a check
(e.g., `row_count_positive`). Carried on the check's typed
specification and dispatched by the runner's evaluator
registry. New kinds are an additive evolution under the
schema-versioning contract. — *Source:*
[ADR-0001](adr/0001-engine-rules-compatibility.md),
[`engine/internal/eval/`](../engine/internal/eval/).

### Entity
The data asset (typically a BigQuery table) that a ruleset
evaluates. Indexed by the loader, declared in the rule's
`entity:` field, keyed in `_owners.yaml`. Forbidden to
contain the ASCII pipe character (it participates in the
`execution_id` formula). — *Source:*
[ADR-0002 §2](adr/0002-run-identity-and-idempotency.md),
[ADR-0006 §1](adr/0006-alert-routing-contract.md).

---

## 11. Environment

### Environment
One of the three first-class deployment contexts the platform
runs in: `local`, `qa`, `prod`. Closed set; adding a fourth
requires an ADR amendment. Substrate isolation is **a
separate GCP project per environment** — IAM is the boundary.
— *Source:* B1-4 MD-2 / MD-3.

### `DQ_ENV`
The single environment-selector environment variable read by
the engine binary at startup. Valid values: `local`, `qa`,
`prod`. All other per-environment configuration is resolved
from a typed `EnvConfig` struct in `engine/internal/env/`
keyed by this selector — not from individual environment
variables. — *Source:* B1-4 MD-4, foundation 04 §PAT-4.

### PAT-4 — Typed multi-environment configuration
The committed shape of the engine's per-environment wiring:
one Go file per environment under `engine/internal/env/`,
each declaring a typed `EnvConfig` struct of identical shape;
selection at startup via `DQ_ENV`; **no dynamic discovery,
no inheritance, no implicit fallbacks**. Adding a field to
one environment fails the build until every environment
adds it. — *Source:* foundation 04 §PAT-4, B1-4 MD-4.

---

## Alphabetical index

- [`/healthz` (liveness)](#healthz-liveness)
- [`/readyz` (readiness)](#readyz-readiness)
- [`_owners.yaml`](#_ownersyaml)
- [AC-W3-N](#ac-w3-n)
- [`accepted_at`](#accepted_at)
- [ADR](#adr)
- [Alert category](#alert-category)
- [API DTO (`/v1/trigger` response)](#api-dto-v1trigger-response)
- [Append-only](#append-only)
- [`attempt_id`](#attempt_id)
- [B0 / B1 / B2](#b0--b1--b2)
- [Byte-equality gate](#byte-equality-gate)
- [C-W2-N](#c-w2-n)
- [CC (Commit Criterion)](#cc-commit-criterion)
- [Channel reference](#channel-reference)
- [Check](#check)
- [Check kind](#check-kind)
- [Check `result`](#check-result)
- [Check-level retry exhaustion](#check-level-retry-exhaustion)
- [Content-addressed (by-hash)](#content-addressed-by-hash)
- [Continuation rule](#continuation-rule)
- [`dq_check_results`](#dq_check_results)
- [`DQ_ENV`](#dq_env)
- [`dq_executions`](#dq_executions)
- [`dq_executions_current`](#dq_executions_current)
- [DSL](#dsl)
- [Eager-at-load hydration](#eager-at-load-hydration)
- [Engine compatibility expression](#engine-compatibility-expression)
- [Engine schema](#engine-schema)
- [`engine_version`](#engine_version)
- [Entity](#entity)
- [Environment](#environment)
- [Event payload](#event-payload)
- [`event_source`](#event_source)
- [`evidence_summary`](#evidence_summary)
- [Execution `status`](#execution-status)
- [`execution_id`](#execution_id)
- [Failure scope](#failure-scope)
- [Generation-conditional write (CAS)](#generation-conditional-write-cas)
- [Hash short-circuit](#hash-short-circuit)
- [In-flight execution isolation](#in-flight-execution-isolation)
- [`linter_used`](#linter_used)
- [Loader](#loader)
- [Manifest](#manifest)
- [Manifest body](#manifest-body)
- [Manifest publication](#manifest-publication)
- [Orphan detector](#orphan-detector)
- [Orphan hash](#orphan-hash)
- [Orphan run](#orphan-run)
- [Orphan trigger](#orphan-trigger)
- [Owner entry](#owner-entry)
- [PAT-1 — Fail-fast registry loading](#pat-1--fail-fast-registry-loading)
- [PAT-4 — Typed multi-environment configuration](#pat-4--typed-multi-environment-configuration)
- [Phase (W3-Pn)](#phase-w3-pn)
- [Pointer file](#pointer-file)
- [`POST /v1/trigger`](#post-v1trigger)
- [Pre-check entity-level problem](#pre-check-entity-level-problem)
- [Refuse-swap](#refuse-swap)
- [Resolved-study / resolved-adr](#resolved-study--resolved-adr)
- [Rule artifact](#rule-artifact)
- [Rules schema mirror](#rules-schema-mirror)
- [Ruleset](#ruleset)
- [`ruleset_version`](#ruleset_version)
- [Run](#run)
- [Runner](#runner)
- [Scheduler retry vs. operator rerun](#scheduler-retry-vs-operator-rerun)
- [Startup-mode load](#startup-mode-load)
- [`status = aborted`](#status--aborted)
- [Study](#study)
- [`supersedes_execution_id`](#supersedes_execution_id)
- [Trigger handler — strict decoder](#trigger-handler--strict-decoder)
- [Trigger-handler retry exhaustion](#trigger-handler-retry-exhaustion)
- [`trigger_source`](#trigger_source)
- [Two-layer deduplication](#two-layer-deduplication)
- [W1 / W2 / W3](#w1--w2--w3)
- [Window](#window)
- `window_start` / `window_end` — see [Window](#window)
