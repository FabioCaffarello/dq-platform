<!-- path: docs/adr/0014-trigger-handler-contract.md -->

# ADR-0014 — HTTP Trigger Handler Contract

- **Status:** accepted
- **Date:** 2026-05-22

**Scope note (added 2026-05-23):** This ADR currently applies to set-oriented capability (realized over BigQuery). Record-oriented capability is addressed separately in Wave-S (see ADR-0020 forthcoming).

---

## Context

The engine exposes a single HTTP surface for accepting triggers
from external schedulers, operators, and manual invocation.
ADR-0002 §3 specifies that the surface accepts `trigger_source ∈
{scheduler, manual}` and dispatches to the runner that computes
`execution_id` per the formula in ADR-0002 §1. ADR-0003 §3 and §4
fix the persistence rows that result.

Six prior ADRs constrain how the surface behaves:

- **ADR-0001** — engine-load verification (load fails closed on
  contract mismatch; no partial loading).
- **ADR-0002** — `execution_id` formula, closed `trigger_source`
  enum, no-pipe input safety, API-layer enforcement of the
  manual / operator-rerun split.
- **ADR-0003** — append-only `dq_executions` persistence with
  the required column set and `attempt_id` derivation.
- **ADR-0005** — manifest publication semantics (single mutable
  pointer, content-addressed manifest body, CAS-published).
- **ADR-0006** — alert routing schema (`_owners.yaml`, channels,
  severity).
- **ADR-0007** — loader / scheduler / retry failure semantics
  (process exit on startup-mode failure, refuse-swap retain-prior
  on refresh failure, in-flight execution isolation against
  captured manifest, hash-short-circuit refresh, scheduler
  reconciliation best-effort, bounded retries).

This ADR commits the trigger handler's four contract elements
not directly determined by the prior ADRs: hydration timing,
request-decoder strictness, response shape, and health-endpoint
semantics across the loader's operational state space.

**Out of scope of this ADR:**

- Authentication / authorisation on `POST /v1/trigger`. The
  initial P4e implementation ships without authentication (see
  Consequence 16); the auth contract is a future ADR.
- Rate limiting and concurrency budgets on the trigger surface.
  A separate cost-discipline ADR; trigger-delivery bounded
  retries (ADR-0007 §7) and per-check bounded retries (ADR-0007
  §8) are the interim budget primitives.
- A gRPC variant of the trigger surface. HTTP only is committed
  here; gRPC is an additive surface for a separate ADR if and
  when an external scheduler demands it.
- TLS termination, observability headers, and tracing
  propagation. Operational details, not contract.

---

## Decision

### 1. Hydration timing — eager-at-load

The engine binds the HTTP listener **only after** the loader's
startup-mode load (ADR-0007 §1) completes successfully.
Subsequent refreshes swap the in-memory manifest reference
atomically; the handler always reads the active reference and
captures it for the duration of plan creation per ADR-0007 §3.

Lazy-per-trigger hydration is forbidden. A lazy handler
observing a published pointer while ruleset hydration is in
flight cannot distinguish "ruleset is being loaded right now"
from "ruleset failed compatibility validation" — but ADR-0001 §4
and ADR-0007 §1 commit the second case to process exit. Lazy
hydration would also permit a manifest swap between input
decode and plan dispatch, a race ADR-0007 §3 was written to
forbid.

### 2. Request decoder — strict

The handler decodes `POST /v1/trigger` request bodies under a
strict posture. Per-field invariants are enforced **before** the
`execution_id` is computed:

- UTF-8 validity on every string input.
- ASCII pipe (`|`) absence on every string input (per ADR-0002
  §2 input-safety rule).
- RFC 3339 UTC format on `window_start` and `window_end`.
- Closed-enum check on `trigger_source` (per ADR-0002 §3).
- Per-field length ceiling.

Unknown fields in the request body return `400` rather than
being silently dropped — this protects against
forward-incompatible schedulers that assume acceptance equals
support.

`trigger_source = operator-rerun` is rejected with `400` — that
value is the Admin API path's exclusive source per ADR-0002 §4.
The data-plane surface at `POST /v1/trigger` accepts exactly
`scheduler` and `manual`.

Lenient decoding is forbidden: byte-level deviation in any input
(e.g., `"+00:00"` coerced into `"Z"`) produces a different
`execution_id` and breaks idempotency. ADR-0002 §1's "public
contract" framing requires byte-identical inputs.

Decoder rejections return `400` with a structured error
envelope `{code, message, field}`. The exact code taxonomy
is a follow-up amendment to this ADR.

### 3. Response shape — separate API DTO

The handler returns an API DTO that is **distinct from** the
`dq_executions` persistence row. The storage contract (ADR-0003)
and the response contract evolve under separate channels:
P5 (contract-driven evolution) treats them as different
audiences — the storage schema serves the reporting layer
(ADR-0003 §2 lazy view), and the response shape serves external
schedulers and operator tooling.

Initial v1 DTO field set, served at `/v1/trigger`:

- `execution_id` — the value from the ADR-0002 §1 formula.
- `attempt_id` — the UUID assigned per ADR-0003 §4.
- `status` — initial value `running` per the ADR-0003 §6
  status enum.
- `accepted_at` — RFC 3339 UTC, handler-side timestamp at the
  point of acceptance.
- `self` — a URL fragment locating the execution's later state
  (e.g., `/v1/executions/{execution_id}`).

`accepted_at` is a handler-side timestamp distinct from the
persistence `started_at` column (ADR-0003 §3). The contract
permits the two to diverge if plan creation has any meaningful
latency.

The initial path is `/v1/trigger`; **v1-path-versioning is the
initial evolution channel**. Criteria that trigger a v2 path
bump (e.g., breaking-change accumulation threshold, minimum
notice period before a v2 cut) are a future amendment to this
ADR when accumulated breaking changes warrant it.

DTO additions are governed by an ADR-amendment process
consistent with P5: additive fields land under a minor
amendment to this ADR; breaking changes (field removals, type
changes) require a new ADR. A documented DTO ↔ `dq_executions`
field mapping lives alongside the handler implementation so the
two contracts are visibly separate but auditable.

### 4. Health endpoint semantics

Two endpoints — `GET /healthz` (liveness) and `GET /readyz`
(readiness) — reason about five operational loader states
derived from ADR-0007's state space. These state names are
synthesized here for handler-readiness reasoning; they are not
labels lifted verbatim from ADR-0007.

The five operational loader states the handler reasons about
are:

1. **Startup not complete** — first manifest load has not yet
   succeeded; the handler has not bound its listener.
2. **Refresh OK recent** — last refresh attempt succeeded; the
   active manifest is the most recently published manifest
   visible to the loader.
3. **Refuse-swap retain-prior** — last refresh failed; the
   loader retained the previously loaded manifest as the
   active reference, and the data plane is unaffected.
4. **Terminal refresh failure** — refresh has failed
   persistently; the loader has escalated to a high-severity
   operational alert. The prior manifest continues to serve.
5. **Scheduler reconciliation degraded** — one or more
   triggers' lifecycle operations have entered a `failing`
   state in the reconciliation loop; sibling triggers are
   unaffected.

Each state maps to a defined response:

| Loader state | `/healthz` | `/readyz` |
|--------------|------------|-----------|
| Startup not complete | unreachable | unreachable |
| Refresh OK recent | 200 | 200 |
| Refuse-swap retain-prior | 200 | **200** |
| Terminal refresh failure | 200 | **200** |
| Scheduler reconciliation degraded | 200 | 200 |

`/healthz` returns `200 OK` while the process is up; it does
not depend on manifest state. Liveness probes use this
endpoint.

`/readyz` returns `200 OK` once the first successful manifest
load completes (per the eager-at-load posture in §1) and stays
`200` for the lifetime of the process unless the process exits.
**Refresh failures — including terminal refresh failure — do
not flip `/readyz`.** ADR-0007's data-plane-isolation guarantee
(the prior manifest continues to serve while refresh retries)
would be defeated if the data-plane probe flipped on a refresh
failure. The escalation signal goes out of band via the
alerting channel committed by ADR-0006.

Scheduler reconciliation degraded is out of band for the
handler — it is a control-plane concern and its visibility is
the reconciliation log + the per-trigger `failing` state, not
a `/readyz` signal.

The decision to keep `/readyz` green during terminal refresh
failure is a contract commitment of this ADR: ADR-0007 commits
the data-plane isolation, but the readyz response shape is
specified here.

---

## Consequences

1. The trigger handler does not bind its HTTP listener until
   the loader's startup-mode load completes successfully. A
   crash-looping engine produces no listener and no synthetic
   readyz signal; the orchestration substrate is the failure
   surface.

2. Refresh-mode loads swap the in-memory manifest reference
   atomically. The handler reads the reference once per
   accepted trigger and captures it for the duration of plan
   creation per ADR-0007 §3. A refresh failure does not affect
   the handler's accept posture — the prior manifest remains
   the active reference.

3. `POST /v1/trigger` accepts exactly `trigger_source ∈
   {scheduler, manual}`. `trigger_source = operator-rerun` is
   rejected with `400` — it is the Admin API path's exclusive
   source per ADR-0002 §4.

4. Per-field invariants — UTF-8 validity, ASCII-pipe absence,
   RFC 3339 UTC regex on the two timestamps, closed-enum check,
   per-field length ceiling — are enforced before the
   `execution_id` is computed.

5. Unknown fields in the request body return `400` rather than
   being silently dropped. This protects against
   forward-incompatible schedulers that assume acceptance
   equals support.

6. Decoder rejections return `400` with a structured error
   envelope `{code, message, field}`. The exact code taxonomy
   is a follow-up amendment to this ADR.

7. The trigger handler returns a response DTO that is distinct
   from the `dq_executions` persistence row. The DTO ↔
   persistence mapping is documented alongside the handler
   implementation.

8. The initial v1 DTO field set is `{execution_id, attempt_id,
   status, accepted_at, self}`. The path is `/v1/trigger`;
   v1-path-versioning is the initial evolution channel.

9. `accepted_at` is a handler-side timestamp distinct from the
   persistence `started_at` column (ADR-0003 §3). The contract
   permits the two to diverge.

10. DTO additions land under a minor amendment to this ADR;
    breaking changes (field removals, type changes) require a
    new ADR.

11. `GET /healthz` returns `200 OK` while the process is up.
    `GET /readyz` returns `200 OK` once the first successful
    manifest load completes and stays `200` for the lifetime
    of the process unless the process exits — refresh failures
    do not flip `/readyz`.

12. Refresh failures and terminal refresh failure surface out
    of band via the alerting channel committed by ADR-0006,
    not via `/readyz`.

13. Scheduler reconciliation degraded is a control-plane
    concern and does not affect `/readyz`. Its visibility is
    the reconciliation log + the per-trigger `failing` state.

14. The trigger handler is the data-plane surface for the
    runner committed by ADR-0002, ADR-0003, ADR-0004, and
    ADR-0007. It is **not** a control-plane surface — it does
    not expose manifest publication, owner ingestion, or
    scheduler reconciliation.

15. The handler carries an `_owners.yaml` entry under the
    schema committed by ADR-0006. Operational alerts from the
    handler — crash-loop visible to the orchestration
    substrate, terminal-refresh-failure surfaced via the
    alerting channel, scheduler-reconciliation-degraded
    visibility — route through that entry's on-call mapping.
    P3 (ownership explicit everywhere) is satisfied by the
    ADR-0006 owner schema; no handler-specific ownership
    primitive is added.

16. The initial P4e implementation ships without
    authentication; the handler accepts unauthenticated
    requests until a follow-up auth ADR lands. Deployment
    posture (e.g., network-level isolation) is the interim
    control.

17. Endpoints committed by this ADR: `POST /v1/trigger`,
    `GET /healthz`, `GET /readyz`. All other endpoints (admin
    rerun, execution read, manifest inspection) are out of
    scope.

---

## Notes

- An optional `/manifestz` endpoint exposing the active
  manifest hash and last successful refresh timestamp is
  useful for operators but not contract-bearing; it may land
  as a follow-up amendment if operator workflows demand it.

- The five operational loader-state names (*Startup not
  complete*, *Refresh OK recent*, *Refuse-swap retain-prior*,
  *Terminal refresh failure*, *Scheduler reconciliation
  degraded*) are synthesized for handler-readiness reasoning
  over ADR-0007's state space. A reader grepping ADR-0007 for
  these exact names will not find them.

- Deferred items captured here are explicit follow-ups, not
  ambiguities: the error-code taxonomy, the v2 path-bump
  trigger criteria, the self-link content format, the
  authentication ADR, the rate-limit / concurrency-budget
  ADR, and the gRPC variant.
