<!-- path: studies/decisions/2026-05-22-trigger-handler-contract.md -->

# W3-P4e — HTTP Trigger Handler Contract

## Metadata

- **Reference:** W3-P4e (renumbered from W3-P6b on 2026-05-22;
  see plan `.claude/plans/w3-p4e-http-trigger-handler.md`).
  ADR-0013 §"Phase 4 — Engine runtime scaffold" places the
  trigger handler inside Phase 4; the decision-log row 130
  still carries the old W3-P6b label and is listed as a
  follow-up in the plan file.
- **Status:** draft (Wave 3, P4e contract preflight — first
  pass, awaiting `/critique`).
- **Last updated:** 2026-05-22.
- **Upstream resolved:**
  [ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md),
  [ADR-0002](../../docs/adr/0002-run-identity-and-idempotency.md),
  [ADR-0003](../../docs/adr/0003-result-write-model.md),
  [ADR-0004](../../docs/adr/0004-failure-scope.md),
  [ADR-0005](../../docs/adr/0005-manifest-publication-semantics.md),
  [ADR-0007](../../docs/adr/0007-loader-scheduler-retry-failure-semantics.md).
  Sequencing per [ADR-0013](../../docs/adr/0013-wave3-sequencing.md).
- **Downstream open:** P4e engine implementation (HTTP handler,
  router wiring, `/healthz` / `/readyz`). Deferred to a separate
  session per R4; opens only after this study reaches
  `resolved-adr`.
- **Promotion target:** `docs/adr/0014-trigger-handler-contract.md`
  (provisional; next ADR slot after ADR-0013).
- **Sub-decisions:** four — MD-1 (hydration timing), MD-2
  (strict-decoder posture), MD-3 (response shape), MD-4 (health
  endpoint semantics across the loader states derived from
  ADR-0007).

---

## Context

ADR-0002 §3 commits the engine to a `POST /v1/trigger` surface
that accepts `trigger_source ∈ {scheduler, manual}` and
dispatches each accepted trigger to the runner that computes
`execution_id` and writes a `dq_executions` row per ADR-0003.
ADR-0002 §4 also commits an admin-API surface for
`operator-rerun` — out of scope here; this study covers only
the data-plane trigger surface.

Six prior ADRs already constrain the handler's behaviour:

- **ADR-0001 §4** (engine-load verification — abort on
  compatibility-contract mismatch, no partial loading, no
  silent skipping) constrains **when** the handler may accept
  its first request.
- **ADR-0002 §1–§4** (the `execution_id` formula, the closed
  `trigger_source` enum, the one-to-one path↔enum mapping
  enforced at the API layer, and the no-pipe input-safety
  rule) constrains **what payload shape** the handler decodes.
- **ADR-0003 §1, §3, §4** (append-only `dq_executions`, the
  required column set, and `attempt_id` derivation) constrains
  **what the handler writes** and therefore the persistence
  type the response must — or must not — reuse.
- **ADR-0005 §3–§4** (single mutable pointer, CAS-published
  manifest content) constrains **what "ruleset is live" means**
  and therefore the readiness model.
- **ADR-0007 §1–§5** (startup-exit, refresh retain-prior with
  N-failure escalation, in-flight execution isolation, hash
  short-circuit refresh, scheduler reconciliation best-effort)
  enumerates **the loader states the handler must reason about**
  for `/healthz` / `/readyz`.
- **ADR-0013 §"Phase 4"** places this artifact inside the
  engine-runtime scaffold; it depends on the loader, runner,
  result-write, failure-scope, and orphan-detection sub-phases
  already merged (P4a–P4d).

What this study leaves to the implementation session and to
later contract work, **out of scope for current cycle**:

- Authentication / authorisation on `POST /v1/trigger`.
  Deferred — substrate-coupled (W2-3 capability matrix) and not
  part of the trigger-formula contract.
- Rate limiting and concurrency budgets per P4. Deferred — the
  P4 budget primitive is itself a future ADR (cost-discipline
  scope, not trigger-handler scope).
- A gRPC variant of `/v1/trigger`. Deferred — the runner
  contract is identical regardless of wire format; a gRPC
  variant is an additive surface, not a contract amendment.
- TLS termination, observability headers, and tracing
  propagation. Deferred — operational details, not contract.

### Structural extension (AC-2 deviation, recorded explicitly)

The study consolidates four micro-decisions in one document
following the precedent of
[`2026-05-21-platform-decisions-wave2.md`](./2026-05-21-platform-decisions-wave2.md).
Each MD-N has its own §N.1–§N.5 mini-MADR block (Drivers,
Options, Recommendation, Consequences, Open Questions). The
top-level AC-2 sections (Context, Decision Drivers, Considered
Options, Recommendation, Consequences, Open Questions,
Promotion target) are all present at the outer level; the
per-MD blocks live between the outer Recommendation (meta)
and the outer cross-cutting Consequences.

---

## Decision Drivers (cross-cutting)

D1. **No erosion of prior commitments.** Every MD-N must honour
the cited ADR sections (ADR-0001, 0002, 0003, 0004, 0005, 0007).
Charter principle P5 (contract-driven evolution) makes this
non-negotiable.

D2. **Determinism (P2).** The handler validates the inputs the
`execution_id` formula consumes; bytes flowing through must be
byte-identical to formula inputs. Any behaviour that introduces
hidden state, time-of-evaluation inputs, or non-idempotent
decoding violates ADR-0002 §1's "no hidden state, no engine-side
randomness".

D3. **Operational visibility (P3).** The handler is the
external surface against which external schedulers, operators,
and probes interact. Owner visibility into accept / reject /
ready posture must be unambiguous; ambiguous health signals
are an ownership violation.

D4. **Cost discipline (P4).** The handler must not amplify
load on the substrate (manifest object store, tabular store).
Eager hydration once, lazy probes, and bounded retries are
the cost shape the prior ADRs already commit to.

D5. **Phase-4 closure.** This study is the last gate before
P4e implementation. It must specify enough that no
contract-bearing architectural decisions remain in scope of
ADR-0014.

---

## Considered Options (meta-shape)

How to structure this study's output:

- **(A) One consolidated study with four mini-MADR blocks**
  (this document). Cross-MD coupling — especially MD-1 ↔ MD-4
  (hydration completion gates readiness) and MD-2 ↔ MD-3
  (decoded payload feeds the response shape) — stays visible
  in one read. **Recommended.**
- **(B) Four independent dated studies.** Higher per-item
  ceremony, harder to keep coupling visible, and the four MDs
  share their entire upstream-ADR citation set.
- **(C) Defer all four to the P4e implementation PR.** Lets
  architecture decisions land mixed with code; rejected by
  R3 (settled architecture is not revisited during code) and
  by the Wave 3 session loop step 4 (the plan must list every
  commitment by exact label — implying the label set exists).

---

## Recommendation (meta)

Adopt **(A)**. Per-MD summary:

| MD | Topic | Decision (one line) |
|----|-------|---------------------|
| MD-1 | Hydration timing | **Eager-at-load.** Handler accepts requests only after first successful manifest load. |
| MD-2 | Strict-decoder posture | **Strict.** Unknown fields rejected; per-field invariants enforced before `execution_id` is computed. |
| MD-3 | Response shape | **Separate API DTO.** Persistence row is not the response contract. |
| MD-4 | Health endpoint semantics | **Split `/healthz` + `/readyz`.** `/readyz` is unreachable during *startup not complete*; returns 200 in *refresh OK recent*, *refuse-swap retain-prior*, *terminal refresh failure*, and *scheduler reconciliation degraded*. |

Details follow in §§1–4.

---

## 1. MD-1 — Hydration timing

### 1.1 Drivers

- **ADR-0001 §4** (engine load fails closed on contract
  mismatch — no partial loading, no silent skipping). A
  handler that accepts triggers before load is complete must
  either reject them (defeating the surface's purpose) or
  serve them against a partial ruleset (forbidden).
- **ADR-0007 §1** (Loader at startup — process exit on any
  failure). Startup-mode failure crashes the process; there is
  no "degraded startup" state the handler could meaningfully
  serve.
- **ADR-0007 §2** (Loader at refresh — refuse swap, retain
  prior manifest). Refresh is the only ongoing source of
  manifest state mutation; the handler must observe the
  active manifest, not a candidate.

### 1.2 Options

- **(A) Eager-at-load.** Handler binds the HTTP port only
  after the loader reports first successful manifest hydration.
  Subsequent refreshes swap the in-memory manifest atomically;
  handler always reads the active reference.
- **(B) Lazy-per-trigger.** Handler binds the port at startup;
  each incoming trigger triggers a pointer read (or pointer-cache
  read) and hydrates the referenced ruleset on demand if not
  already cached.

### 1.3 Recommendation

**(A) Eager-at-load.** Grounded in ADR-0001 §4 and ADR-0007 §1.

Lazy-per-trigger is incompatible with two prior commitments:

- ADR-0001 §4 forbids partial loading. A lazy handler observing
  pointer A while ruleset hydration is still in flight cannot
  distinguish "ruleset is being loaded right now" from "ruleset
  failed compatibility validation" — but ADR-0007 §1 says the
  second case exits the process. Lazy hydration introduces a
  state ADR-0007 explicitly excludes.
- ADR-0002 §1's "no hidden state" extends to the input set the
  handler uses. The active manifest must be deterministic for
  the duration of plan creation; lazy hydration permits a
  pointer refresh between decode and dispatch, opening a
  race ADR-0007 §3 (in-flight execution isolation) was written
  to forbid.

### 1.4 Consequences

- **C-MD-1.1.** The handler does not bind its HTTP listener
  until the loader's startup-mode load (ADR-0007 §1) completes
  successfully. A crash-looping engine produces no listener and
  no synthetic readyz signal; the orchestration substrate is
  the failure surface.
- **C-MD-1.2.** Refresh-mode loads (ADR-0007 §2, §4) swap the
  in-memory manifest reference atomically. The handler reads
  the reference once per accepted trigger and captures it for
  the duration of plan creation per ADR-0007 §3.
- **C-MD-1.3.** A refresh failure (ADR-0007 §2) does not
  affect the handler's accept posture — the prior manifest
  remains the active reference. The handler does not need to
  observe the refresh failure directly.
- **C-MD-1.4.** Hydration of multiple parallel manifest
  versions (canary / shadow) is not committed by this study
  — see OQ-MD-1.1.

### 1.5 Open Questions

- **OQ-MD-1.1.** Canary / shadow manifest hydration for
  before/after comparison runs.
  **Out-of-scope for current cycle — not committed by ADR-0005
  or ADR-0007; would require a multi-active-manifest model
  those ADRs do not anticipate.** (new contribution proposed
  here, requires review — flagged as a future ADR if and when
  shadow runs become a requirement.)

---

## 2. MD-2 — Strict-decoder posture

### 2.1 Drivers

- **ADR-0002 §1** (the `execution_id` formula).
  `execution_id = sha256_hex(ruleset_version || entity ||
  window_start || window_end || trigger_source)` — five inputs,
  pipe-joined, no escaping, UTF-8 canonical. Any deviation in
  input bytes produces a different `execution_id` and breaks
  idempotency.
- **ADR-0002 §2** (input type definitions and the no-pipe
  rule — `\|` forbidden inside any input).
- **ADR-0002 §3** (closed `trigger_source` enum:
  `scheduler`, `manual`, `operator-rerun`).
- **ADR-0002 §4** (API-layer enforcement of the manual /
  operator-rerun split — one-to-one path↔enum mapping,
  not by convention).

### 2.2 Options

- **(A) Strict decoder.** Unknown fields rejected; exact
  type checks; RFC 3339 UTC regex on `window_start` /
  `window_end`; closed-enum check on `trigger_source`;
  ASCII-pipe check on every string input; per-input length
  ceiling.
- **(B) Lenient decoder.** Unknown fields ignored; type
  coercion (numeric → string, etc.) permitted; soft
  validation with warning emission.

### 2.3 Recommendation

**(A) Strict.** Grounded in ADR-0002 §1, §2, §3, §4.

Lenient decoding is incompatible with ADR-0002 §1's "public
contract" framing: anyone with the five inputs must be able to
reproduce the `execution_id`. If the handler accepts variants
that coerce to the same logical input but differ in bytes
(e.g. `"+00:00"` vs `"Z"`), two callers with the same intent
produce different `execution_id`s. ADR-0002 §1 already commits
to UTF-8-canonical, pipe-joined, no-escaping inputs; strict
decode is the only posture that preserves the contract.

### 2.4 Consequences

- **C-MD-2.1.** `POST /v1/trigger` **rejects** requests
  carrying `trigger_source = operator-rerun` with `400`. The
  `operator-rerun` value is the Admin API path's exclusive
  source per ADR-0002 §4.
- **C-MD-2.2.** `POST /v1/trigger` accepts exactly
  `trigger_source ∈ {scheduler, manual}`. Any other value
  returns `400`.
- **C-MD-2.3.** Per-field invariants — UTF-8 validity,
  ASCII-pipe absence, RFC 3339 UTC regex on the two timestamps,
  closed-enum check, per-field length ceiling — are all
  enforced **before** the `execution_id` is computed.
  ADR-0002's contract requires that the formula be applied to
  validated inputs.
- **C-MD-2.4.** Unknown fields in the request body return
  `400` rather than being silently dropped. This protects
  against forward-incompatible schedulers that assume
  acceptance equals support.
- **C-MD-2.5.** Decoder rejections return `400` with a
  structured error envelope `{code, message, field}`. The
  exact `code` taxonomy (e.g. `INVALID_TRIGGER_SOURCE`,
  `INVALID_WINDOW_FORMAT`, `PIPE_IN_INPUT`,
  `UNKNOWN_FIELD`) is **new contribution proposed here,
  requires review** — see OQ-MD-2.1.

### 2.5 Open Questions

- **OQ-MD-2.1.** Concrete error-code taxonomy for `400`
  responses.
  **Out-of-scope for current cycle — taxonomy is a public
  API contract that should land with ADR-0014 promotion,
  not be sprinkled in implementation.** (new contribution
  proposed here, requires review)
- **OQ-MD-2.2.** Maximum-payload size for `POST /v1/trigger`.
  **Out-of-scope for current cycle — substrate-coupled
  (ingress configuration); the per-field length ceilings in
  C-MD-2.3 are sufficient for the formula's safety.**

---

## 3. MD-3 — Response shape

### 3.1 Drivers

- **ADR-0003 §1** (`dq_executions` is append-only; one row
  per state transition, INSERT-only).
- **ADR-0003 §3** (required columns on `dq_executions`:
  `execution_id`, `attempt_id`, `recorded_at`, `status`,
  `engine_version`, `ruleset_version`, `entity`,
  `trigger_source`, `started_at`, `completed_at`,
  `error_summary`, `supersedes_execution_id`).
- **ADR-0003 §4** (`attempt_id` derivation — assigned at
  trigger acceptance; the handler is responsible for the
  initial `running` row).
- **Charter principle P5** (contract-driven evolution —
  schema and API evolve under a published compatibility
  contract).

### 3.2 Options

- **(A) Return the `dq_executions` row directly.** The
  storage type is the response contract; any column added
  to `dq_executions` is automatically exposed.
- **(B) Return a separate API DTO.** A documented subset
  of fields the API contract guarantees; storage schema and
  response shape evolve independently.

### 3.3 Recommendation

**(B) Separate API DTO.** **New contribution proposed here,
requires review** — ADR-0003 commits the storage contract
but does not commit a response contract.

The two contracts have different audiences:

- The storage schema's audience is the reporting layer and
  the lazy view defined by ADR-0003 §2 — a tabular consumer
  with full-column expectations.
- The response shape's audience is the external scheduler
  and operator tooling — clients that care about
  acknowledgment, not about forensic columns.

Coupling them ties the public API surface to storage decisions
(e.g. adding a `supersedes_execution_id` audit column in a
later ADR would silently expand the response payload). P5
makes the API contract a separate evolution channel.

Initial DTO field set (also **new contribution proposed here,
requires review**, to be ratified at ADR-0014 promotion):

- `execution_id` (string, the value from the ADR-0002 formula)
- `attempt_id` (string, the UUID assigned per ADR-0003 §4)
- `status` (string, initial value `running` per the ADR-0003
  §6 enum)
- `accepted_at` (RFC 3339 UTC, handler-side timestamp at the
  point of acceptance)
- `self` (string, a URL fragment locating this execution's
  later state — e.g. `/v1/executions/{execution_id}`; the
  read API itself is out of scope)

### 3.4 Consequences

- **C-MD-3.1.** ADR-0014 enumerates the initial DTO field
  set (above) as the committed v1 contract, served at
  `/v1/trigger` — v1-path-versioning is the initial evolution
  channel.
- **C-MD-3.2.** A documented DTO ↔ `dq_executions` field
  mapping lives alongside the handler implementation so the
  two contracts are visibly separate but auditable.
- **C-MD-3.3.** `accepted_at` is a handler-side timestamp,
  distinct from the persistence `started_at` (ADR-0003 §3).
  In practice they can be equal; the contract permits them
  to diverge if plan creation has any meaningful latency.
- **C-MD-3.4.** DTO additions are governed by an
  ADR-amendment process consistent with P5; additive
  fields land under a minor ADR amendment, breaking
  changes require a new ADR.

### 3.5 Open Questions

- **OQ-MD-3.1.** Criteria that trigger a v2 path bump
  (e.g., breaking-change accumulation threshold, minimum
  notice period before a v2 cut).
  **Out-of-scope for current cycle — v1-path-versioning is
  committed in C-MD-3.1; the v2-bump trigger criteria are
  a future amendment when accumulated breaking changes
  warrant it.**
- **OQ-MD-3.2.** Self-link content (relative path vs
  absolute URL vs self-link returned with explicit
  media-type negotiation contract).
  **Out-of-scope for current cycle — the read API is itself
  out of scope.** (new contribution proposed here, requires
  review)

---

## 4. MD-4 — Health endpoint semantics across the loader states

### 4.1 Drivers

- **ADR-0007** (loader-and-scheduler failure semantics — the
  full operational state space the handler must reason about,
  including data-plane isolation from refresh failures and
  trigger-delivery exhaustion not producing an execution row).

The five operational loader states the handler reasons about
for `/healthz` / `/readyz` are listed below. These names are
synthesized here for handler-readiness reasoning over ADR-0007's
state space; they are not labels lifted verbatim from ADR-0007
— a future reader grepping ADR-0007 for these exact names will
not find them.

1. **Startup not complete.** First manifest load has not yet
   succeeded. The handler has not bound its listener.
2. **Refresh OK recent.** Last refresh attempt succeeded; the
   active manifest is the most recently published manifest
   visible to the loader.
3. **Refuse-swap retain-prior.** Last refresh failed; the
   loader retained the previously loaded manifest as the
   active reference, and the data plane is unaffected.
4. **Terminal refresh failure.** Refresh has failed
   persistently; the loader has escalated to a high-severity
   operational alert. The prior manifest continues to serve.
5. **Scheduler reconciliation degraded.** One or more
   triggers' lifecycle operations have entered a `failing`
   state in the reconciliation loop; sibling triggers are
   unaffected.

### 4.2 Options

- **(A) Single `/healthz`** returning binary up/down. A
  scheduler probe gets one bit. Cheap; ambiguous between
  liveness and readiness.
- **(B) Split `/healthz` + `/readyz`.**
  `/healthz` is liveness (process is up); `/readyz` is
  readiness (handler will accept triggers). External probes
  and external schedulers consult the appropriate endpoint.
- **(C) Three-tier `/healthz` + `/readyz` + `/manifestz`.**
  `/manifestz` exposes the active manifest hash and the
  last successful refresh timestamp. Useful for operators;
  the scheduler probe needs only `/readyz`.

### 4.3 Recommendation

**(B) Split `/healthz` + `/readyz`.** Each of the five
operational loader states maps to a defined response:

| Loader state | `/healthz` | `/readyz` |
|--------------|------------|-----------|
| Startup not complete | unreachable | unreachable |
| Refresh OK recent | 200 | 200 |
| Refuse-swap retain-prior | 200 | **200** |
| Terminal refresh failure | 200 | **200** |
| Scheduler reconciliation degraded | 200 | 200 |

Keeping `/readyz` green during refuse-swap retain-prior
**and** terminal refresh failure is grounded in ADR-0007's
data-plane-isolation guarantee (the prior manifest continues
to serve while refresh retries). The escalation signal goes
out of band via the alerting channel committed by ADR-0006,
not via `/readyz`. An external scheduler stopping its
traffic just because the manifest refresh is degraded would
defeat ADR-0007's data-plane-isolation guarantee.

The explicit decision to keep `/readyz` green during
terminal refresh failure is **new contribution proposed
here, requires review** — ADR-0007 commits the data-plane
isolation but does not directly commit the readyz response
shape. Scheduler reconciliation degraded is out of band for
the handler — it is a control-plane concern, not a `/readyz`
signal.

### 4.4 Consequences

- **C-MD-4.1.** `/healthz` returns `200 OK` while the
  process is up. It does not depend on manifest state.
  Liveness probes use this endpoint.
- **C-MD-4.2.** `/readyz` returns `200 OK` once the first
  successful manifest load completes (C-MD-1.1). For the
  rest of the process's lifetime, `/readyz` stays at `200`
  unless the process exits — refresh failures do not flip
  `/readyz`.
- **C-MD-4.3.** Refresh failures and refresh escalation
  surface out of band via the alerting channel committed by
  ADR-0006, **not** via `/readyz`.
- **C-MD-4.4.** Scheduler reconciliation degraded
  (per-trigger lifecycle failures per ADR-0007) is a
  control-plane concern and does not affect `/readyz`. Its
  visibility is the reconciliation log + the per-trigger
  `failing` state, not a handler endpoint.
- **C-MD-4.5.** An optional `/manifestz` exposing the
  active manifest hash and last successful refresh timestamp
  is recorded as a deferred open question (OQ-MD-4.1) — it
  is useful for operators but not contract-bearing.

### 4.5 Open Questions

- **OQ-MD-4.1.** Adoption of `/manifestz` (option C in §4.2).
  **Out-of-scope for current cycle — useful for operators but
  not required to satisfy the loader-state contract. May land
  as a follow-up ADR if operator workflows demand it.** (new
  contribution proposed here, requires review)
- **OQ-MD-4.2.** Specific status code for `/readyz` during
  the brief startup window before first load completes —
  `503 Service Unavailable` is the conventional choice but is
  not technically required because the handler does not bind
  its listener until the load completes (C-MD-1.1), so the
  endpoint is unreachable rather than 503.
  **Out-of-scope for current cycle — implementation note,
  not contract.**

---

## Consequences (cross-cutting)

The four MDs together commit the engine to a single HTTP
surface at P4e:

- **CC1.** Three endpoints: `POST /v1/trigger`, `GET /healthz`,
  `GET /readyz`. All other endpoints (admin rerun, execution
  read, manifest inspection) are explicitly out of scope for
  this study.
- **CC2.** The handler is the data-plane surface for the
  runner committed by ADR-0002 / ADR-0003 / ADR-0004 /
  ADR-0007. It is **not** a control-plane surface — it does
  not expose manifest publication, owner ingestion, or
  scheduler reconciliation.
- **CC3.** The handler's contract surface — accept rules,
  decode posture, response shape, readiness posture — is
  fully specified by MD-1..MD-4 once promoted to ADR-0014.
  No contract-bearing architectural decisions remain in
  scope of ADR-0014.
- **CC4.** Phase 6 ("first onboarded entity end-to-end" per
  ADR-0013) **exercises** this handler against real content
  but does not amend its contract. Any onboarding-driven
  contract change must round-trip through this ADR slot.
- **CC5.** The handler carries an `_owners.yaml` entry under
  the schema committed by ADR-0006. Operational alerts from
  the handler (crash-loop visible to the orchestration
  substrate, terminal-refresh-failure surfaced via the
  alerting channel, scheduler-reconciliation-degraded
  visibility) route through that entry's on-call mapping. P3
  (ownership explicit everywhere) is satisfied by the
  ADR-0006 owner schema; no handler-specific ownership
  primitive is added.
- **CC6.** P4e initial implementation ships without
  authentication; the handler accepts unauthenticated
  requests until the auth ADR (deferred OQ-CC.1) lands.
  Deployment posture (e.g. network-level isolation) is the
  interim control. This commitment is **new contribution
  proposed here, requires review**.

---

## Open Questions (cross-cutting)

- **OQ-CC.1.** Authentication / authorisation on
  `POST /v1/trigger`.
  **Out-of-scope for current cycle — substrate-coupled
  (W2-3 capability matrix); the interim no-auth posture is
  committed in CC6.** (new contribution proposed here,
  requires review)
- **OQ-CC.2.** Rate limiting and concurrency budgets on
  `POST /v1/trigger`.
  **Out-of-scope for current cycle — cost-discipline (P4) is
  its own future ADR. The trigger-delivery bounded retries
  (ADR-0007 §7) and the per-check bounded retries (ADR-0007
  §8) are the interim budget primitives.**
- **OQ-CC.3.** gRPC variant of `/v1/trigger`.
  **Out-of-scope for current cycle — additive surface, not a
  contract amendment; will be a separate ADR if and when an
  external scheduler demands it.**
- **OQ-CC.4.** TLS termination, observability headers, and
  tracing propagation.
  **Out-of-scope for current cycle — operational concerns, not
  contract.**

---

## Promotion target

`docs/adr/0014-trigger-handler-contract.md`.

ADR-0014 follows the slot convention established by prior
studies (latest accepted ADR is ADR-0013;
[`2026-05-21-wave3-sequencing.md`](./2026-05-21-wave3-sequencing.md)
promoted into that slot). On promotion, ADR-0014 carries the
four MD-N recommendations as separate Decision sub-sections;
the per-MD Consequences (`C-MD-N.M`) and per-MD Open Questions
(`OQ-MD-N.M`) are renumbered into a single ADR-level
Consequence list and Open Question list, as is the precedent
for prior promoted ADRs.
