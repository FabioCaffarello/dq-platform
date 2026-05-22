<!-- path: studies/decisions/2026-05-22-b1-4-environment-configuration-model.md -->

# B1-4 — Environment Configuration Model

## Metadata

- **B1 reference:** B1-4 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md)
  (row at line 64).
- **Status:** draft (resolved-study after the critique pass closes).
- **Last updated:** 2026-05-22.
- **Upstream resolved:**
  - Foundation 01 (`studies/foundation/01-charter-and-principles.md`)
    §"P2 — Deterministic engine" — the ban on env-driven behaviour.
  - Foundation 04 (`studies/foundation/04-system-architecture.md`)
    §"PAT-4 — Typed multi-environment configuration" — the shape
    of the engine's configuration package; this study confirms
    PAT-4 and elaborates two structural choices PAT-4 leaves open.
  - Foundation 05 (`studies/foundation/05-operational-discipline.md`)
    §"Cost Discipline" — the recognition that per-env budgets are
    a distinct future B1.
  - ADR-0010 (substrate posture) — the **Yes / Partial / No**
    capability split that maps cleanly onto `local` vs
    `qa`/`prod`.
  - ADR-0013 §"Phase 7" — names `local`, `qa`, `prod` as the
    three first-class environments and explicitly defers
    Kubernetes overlay work to the post-B1-4 Phase-7 session.
  - ADR-0005 §4 (manifest publication) — the CAS-protected
    data-plane content model.
  - ADR-0006 §1–§4 (alert routing schema) — `_owners.yaml` as
    versioned data-plane content shipped with the rules
    workspace.
- **Downstream open:**
  - **W3-P7** scaffolding session (Kubernetes manifests +
    overlays) cannot begin until this study reaches
    `resolved-study`; ADR-0013 §"Phase 7" explicitly conditions
    Phase 7 on B1-4 closure.
  - Future B1: per-env cost ceilings (deferred per foundation
    05).
  - Future B1: secrets-storage primitive (KMS / Vault /
    sealed-secrets / etc.).
- **Promotion target:** see final section.

---

## Context

The platform's runtime today reads twelve ad-hoc env vars at
startup (`engine/cmd/dq-engine/main.go` `readEnv()` —
`DQ_ENGINE_VERSION`, `DQ_GCS_BUCKET`, `DQ_BIGQUERY_PROJECT`,
`DQ_BIGQUERY_DATASET`, `DQ_PUBSUB_PROJECT`, `DQ_PUBSUB_TOPIC`,
`DQ_LOADER_REFRESH_INTERVAL`, `DQ_ORPHAN_THRESHOLD`,
`DQ_ORPHAN_SCAN_INTERVAL`, `DQ_HTTP_ADDR`, `DQ_SOURCE_PROJECT`,
`DQ_SOURCE_DATASET`, plus three emulator-only overrides). The
CLI tools (`tools/manifest`, `tools/lint`) read no env vars and
take all inputs as flags. The local Compose stack hard-codes
`dq-local` everywhere. No deployed environment manifest exists
yet.

Foundation 04 §"PAT-4" already commits the shape the engine
should evolve toward:

> The engine runs in multiple environments (local, qa, prod,
> possibly more). The configuration model is:
> - one Go file per environment in `engine/internal/env/`;
> - each file declares a typed `EnvConfig` struct with the same
>   shape;
> - selection at startup via an environment variable (e.g.
>   `DQ_ENV=qa`);
> - no dynamic discovery, no inheritance chains, no implicit
>   fallbacks.
> 
> If a new field is added to one environment, the build fails
> until it is added to all environments. This forces deliberate
> decisions and makes drift impossible.

PAT-4 settles the *shape* of the configuration package but
leaves three structural choices open that the deploy
scaffolding (W3-P7) needs answered:

1. **Where does each piece of configuration live overall?**
   PAT-4 covers the engine's deployment-time wiring; it does
   not commit a split between code, deployment, and data.
2. **Which environments are first-class?** PAT-4 mentions
   "local, qa, prod, possibly more" — open-ended; ADR-0013
   §"Phase 7" names exactly three. B1-4 commits the closed set.
3. **How are environments isolated at the substrate?** PAT-4
   commits *per-environment* configuration but not *what*
   isolation the configuration encodes (separate GCP projects?
   shared project with naming convention?). ADR-0010's
   capability matrix implies the answer but doesn't make it
   explicit.

B1-4 commits all three plus two follow-ons: (a) the runtime
refactor scope that moves the current twelve-env-var startup
into a PAT-4-shaped typed package; and (b) the CLI tools'
relationship to environments (they remain flag-based; flags
*are* deployment-time inputs by another name).

### Structural extension (AC-2 deviation, recorded explicitly)

The study consolidates five micro-decisions (MD-1 through
MD-5) in one document, mirroring the precedent of
[`2026-05-21-platform-decisions-wave2.md`](./2026-05-21-platform-decisions-wave2.md)
and
[`2026-05-22-trigger-handler-contract.md`](./2026-05-22-trigger-handler-contract.md).
Each MD-N has its own §N.1–§N.5 mini-MADR block (Drivers,
Options, Recommendation, Consequences, Open Questions). The
top-level AC-2 sections (Context, Decision Drivers, Considered
Options, Recommendation, Consequences, Open Questions,
Promotion target) are all present at the outer level; the
per-MD blocks live between the outer Recommendation (meta) and
the outer cross-cutting Consequences.

### Out of scope (deferrals)

| Topic | Why deferred |
|---|---|
| Specific cost ceilings per env (foundation 05) | Cost ceilings are a separate future B1 — foundation 05 §"Cost Discipline" states explicitly that "the exact values for the ceilings and budgets are environment-specific and resolved as B1 decisions." |
| Choice between Kustomize and Helm (or anything else) for the overlay tool | W3-P7 scaffolding decision; B1-4 commits the *shape* of per-env config (what fields exist, how they're isolated) but not the overlay layout tool. |
| Secrets-storage primitive (KMS / Vault / sealed secrets) | A separate future B1; B1-4 commits that secrets are deployment-bucket items, not how they're stored. |
| `DQ_ENV` value canonicalization rules | Implementation detail of the typed env package; lands with W3-P7. |
| Cross-region / multi-region production layout | Out of scope for v1; one `prod` GCP project per the recommendation in MD-3. |
| OIDC service-identity flows | Sandbox-only per ADR-0010; orthogonal to env model. |

---

## Decision Drivers (cross-cutting)

D1. **Foundation 01 §P2 — deterministic engine.** Same rule
version, time window, and source state must produce the same
execution semantics. Behaviour driven by an env var is a
defect, not a feature. The configuration model must isolate
*wiring* from *behaviour*; behaviour belongs in code (compile-
time invariants) or data (rule YAMLs, owners file).

D2. **Foundation 04 §PAT-4.** The typed multi-environment
package is the committed shape. B1-4 cannot revisit PAT-4; it
can only confirm and elaborate it.

D3. **ADR-0013 §"Phase 7" enumeration.** Three environments —
`local`, `qa`, `prod` — are named in the Wave-3 sequencing ADR.
Adding a fourth (e.g., `staging`) would itself be an ADR
amendment, not a B1-4 sub-decision.

D4. **ADR-0010 capability-matrix split.** **Yes** rows are
exercised end-to-end on `local`; **Partial** / **No** rows are
sandbox-required. The env model must honour this split —
`local` cannot pretend to have a CAS-faithful object store, for
instance.

D5. **R5 / P6 — borrow patterns, not baggage.** Per-environment
GCP projects, per-env Kubernetes overlays, and typed config
structs are commodity engineering practices; this study names
them in their own terms and on their own merits, without citing
sibling-platform or prior-art systems.

D6. **W3-P7 unblock.** This study's first downstream consumer
is the W3-P7 scaffolding session. The study must commit
enough that W3-P7 has no remaining architectural decisions —
only mechanical wiring.

---

## Considered Options (meta-shape)

How to structure this study's output:

- **(A) One consolidated study with five mini-MADR blocks**
  (this document). Cross-MD coupling — MD-1 (the buckets) ↔
  MD-3 (the isolation that the deployment bucket encodes), and
  MD-4 (the engine refactor) ↔ MD-2 (the env list it must
  encode) — stays visible in one read. **Recommended.**
- **(B) Five independent dated studies.** Higher per-item
  ceremony; cross-MD coupling harder to keep coherent; B1-10
  and B1-11 set the precedent for single-topic B1 studies, but
  B1-4's sub-decisions are tightly coupled and don't naturally
  split.
- **(C) Defer all five to the W3-P7 PR.** Inverts the wave-3
  session-loop step 2 rule that upstream decisions resolve
  before downstream code. Rejected.

---

## Recommendation (meta)

Adopt **(A)**. Per-MD summary:

| MD | Topic | Decision (one line) |
|----|-------|---------------------|
| MD-1 | Three-bucket configuration model | **Code / deployment / data.** Every config item lives in exactly one bucket. |
| MD-2 | First-class environments | **`local`, `qa`, `prod`.** Closed set; additional envs require ADR amendment. |
| MD-3 | Substrate isolation | **Separate GCP project per environment.** IAM is the boundary. |
| MD-4 | PAT-4 confirmation + refactor scope | **Confirm PAT-4; refactor `cmd/dq-engine/main.go` `readEnv()` into the typed `engine/internal/env/` package in W3-P7's first step.** |
| MD-5 | CLI tools (manifest, lint) | **Flag-based, env-agnostic.** Flags are deployment-time inputs; no DQ_ENV-style typed struct needed. |

Details follow in §§1–5.

---

## 1. MD-1 — Three-bucket configuration model

### 1.1 Drivers

- **Foundation 01 §P2** (deterministic-engine) bans env-driven
  behaviour. Any configuration item that changes behaviour
  belongs in code (compile-time invariant) or data (CAS-
  versioned content); deployment is the *wiring* bucket only.
- **ADR-0001** commits the rule-YAML grammar in `engine/internal/dsl/schema/v1.schema.json`
  as the schema source of truth — that file is committed to
  the repository, so its evolution is a code change, not a
  deployment change. Rules themselves (entity + check_id +
  kind) are data, published via the manifest publisher.
- **ADR-0005 §4** (manifest publication) commits the
  CAS-protected pointer + content-addressed body layout —
  data-plane content has its own evolution channel
  (`dq-manifest publish`), distinct from both code (rebuild)
  and deployment (redeploy).
- **ADR-0006** (alert routing) commits `_owners.yaml` as
  data shipped with the rules workspace — owner identity is
  data, not deployment wiring.

### 1.2 Options

- **(A) Three buckets — code, deployment, data.** Every
  configuration item lives in exactly one. **Recommended.**
- **(B) Two buckets — code and runtime.** Collapse deployment
  and data into "runtime". Loses the CAS-vs-redeploy
  distinction that ADR-0005 already commits.
- **(C) Free-form, no buckets.** Lets every item be evaluated
  case-by-case. Loses the binding split — "env var that
  controls behaviour" becomes a per-item argument every time
  it comes up.

### 1.3 Recommendation

**(A) Three buckets — code, deployment, data.** Each bucket
has a primary source-of-truth location, a primary evolution
channel, and a primary audience. The data bucket admits more
than one evolution channel by design (see C-MD-1.3 below for
the `_owners.yaml` carve-out); the code and deployment buckets
do not:

- **Code (compile-time invariants).** Source of truth: the
  engine's Go source. Evolution channel: ADR + Go code change
  + version bump. Audience: engineers. Examples: schema
  versions; the `execution_id` formula (ADR-0002 §1); the
  closed `trigger_source` enum (ADR-0002 §3); the closed
  `ExecutionStatus` enum (ADR-0003 §6); the strict decoder
  (ADR-0014 §2); the row_count_positive predicate (ADR-0014
  pending; W3-P6c).
- **Deployment (per-environment wiring).** Source of truth:
  the deployment artifact (W3-P7 Kubernetes manifests +
  overlays). Evolution channel: deployment change (no engine
  rebuild). Audience: operators. Examples: GCP project IDs;
  bucket names; BigQuery dataset names; Pub/Sub topic names;
  the engine's listener address; refresh intervals; log
  level; the `DQ_ENV` selector itself.
- **Data (control-plane content).** Source of truth: the
  object store (or repository for owners). Evolution channel:
  `dq-manifest publish` (rules) or repository commit + redeploy
  of the rules workspace (`_owners.yaml`). Audience: domain
  teams + alert owners. Examples: rule YAMLs; the
  `_owners.yaml` file; the manifest pointer.

The split is binding. A future temptation to add an
`DQ_BEHAVIOR_FLAG=…` env var must be answered "behaviour
belongs in code or in data, not deployment" and rerouted.

### 1.4 Consequences

- **C-MD-1.1.** Every existing engine env var fits in the
  deployment bucket (per the audit in the Context). The audit
  is a load-bearing input to MD-4's refactor scope.
- **C-MD-1.2.** Future configuration items must be classified
  at the point of introduction. The PR description for any
  new env var or rule field must name its bucket explicitly.
- **C-MD-1.3.** The `_owners.yaml` file's location — in the
  rules workspace, shipped via repository commits rather than
  via `dq-manifest publish` — is a deliberate deviation from
  full data-plane semantics. It is data (audience: alert
  owners), but its evolution channel is repository-based
  because the linter cross-checks it at lint time per ADR-0006
  CC9. This is consistent with the bucket model — the data
  bucket allows multiple evolution channels — but is worth
  calling out so a future contributor doesn't move `_owners.yaml`
  into `yamls/by-hash/` thinking it should match rule YAMLs.
- **C-MD-1.4.** The three-bucket split prohibits a fourth
  bucket. If a future contributor proposes a "runtime
  database" or "remote feature flag" bucket, it requires
  amending B1-4 (or this study's promoted ADR), not
  side-stepping the model.

### 1.5 Open Questions

- **OQ-MD-1.1.** Whether the linter should statically detect
  attempts to introduce a fourth bucket (e.g., a Go env var
  that controls behaviour rather than wiring). **Out-of-scope
  for current cycle — tooling guard; lands as a follow-up
  linter capability if drift is observed.**

---

## 2. MD-2 — First-class environments

### 2.1 Drivers

- **ADR-0013 §"Phase 7"** names exactly three environments
  (`local`, `qa`, `prod`). B1-4 must commit a closed set so
  W3-P7 can scaffold a finite number of overlays.
- **Foundation 04 PAT-4** mentions "local, qa, prod, possibly
  more" — open-ended; B1-4 closes the open-endedness.
- **ADR-0010 capability matrix** splits cleanly along Yes
  (local-friendly) vs Partial/No (sandbox-required), which
  matches a two-tier "local vs cloud-deployed" cut. The cloud
  tier in turn has at least one non-production tier (qa) and
  one production tier (prod).

### 2.2 Options

- **(A) Three first-class environments: `local`, `qa`, `prod`.**
  **Recommended.**
- **(B) Two environments: `local` and `cloud`, with `cloud`
  parameterized.** Smaller surface, but conflates qa and prod;
  a qa-vs-prod split is a load-bearing operational distinction
  (separate IAM, separate billing, separate read-after-write
  expectations).
- **(C) Four-plus environments: add `staging`, `dev`, etc.**
  Defensible if the team already runs them; not yet the case
  for this platform. Closing the set keeps scope manageable.

### 2.3 Recommendation

**(A) Three first-class environments — `local`, `qa`, `prod`.**

- `local` — developer laptops + CI. Uses the Compose substrate
  per ADR-0010. **Yes** rows from the capability matrix are
  exercisable; **Partial** / **No** rows are not. The
  `dq-fixture` source dataset convention from W3-P6d's demo
  script lives in this env.
- `qa` — a dedicated sandbox cloud GCP project. The sandbox-
  required ADR-0010 rows (**Partial** / **No**) are wired here:
  CAS enforcement, OIDC service identity, the unforgeable
  linter pin. Used for pre-production verification of rule
  changes and engine releases.
- `prod` — the production GCP project, the canonical
  data-plane. The same wiring as `qa` plus per-env
  configuration values (cost ceilings, alert routing
  channels) that match production posture.

The set is closed at v1. Adding a fourth environment requires
amending the promoted ADR (or this study, before promotion).

### 2.4 Consequences

- **C-MD-2.1.** W3-P7 scaffolds exactly three overlay
  directories (e.g., `deploy/overlays/local/`,
  `deploy/overlays/qa/`, `deploy/overlays/prod/`). The
  underlying overlay tool is W3-P7's choice (MD-1 deferral
  list).
- **C-MD-2.2.** Foundation 04 PAT-4's `engine/internal/env/`
  package contains exactly three files (`local.go`, `qa.go`,
  `prod.go`) once the W3-P7-step-1 refactor lands. Adding a
  file requires this study's amendment.
- **C-MD-2.3.** Tests that span multiple envs (e.g., a future
  CI lane that runs `qa`-shaped integration tests) name the
  env explicitly via `DQ_ENV` — no string-matching against
  hostnames or other implicit signals.
- **C-MD-2.4.** The owners test fixture at
  `tools/lint/testdata/owners/valid/_owners.yaml` mentions
  `prod` and `staging` in severity overrides. `staging` is
  **not** a first-class environment under MD-2; the fixture is
  testing the *parser's* ability to handle arbitrary env keys
  in severity overrides, not asserting that `staging` exists.
  No fixture change is required.

### 2.5 Open Questions

- **OQ-MD-2.1.** Naming canonicalization — should `DQ_ENV`
  values be lowercase exactly (`local`/`qa`/`prod`), or
  case-insensitive? **Out-of-scope for current cycle —
  implementation detail of the typed env package; lands with
  W3-P7's first refactor step.** (new contribution proposed
  here, requires review)
- **OQ-MD-2.2.** Whether CI fans out to multiple envs (e.g.,
  a `qa` lane that runs sandbox-required tests). **Out-of-
  scope for current cycle — covered by W3-P7's CI shape work,
  itself deferred.**

---

## 3. MD-3 — Substrate isolation

### 3.1 Drivers

- **Foundation 01 §P2** (deterministic-engine) — env-driven
  behaviour is a defect. The *isolation* boundary must be
  physical (IAM, project boundary) so a misconfigured env
  var cannot cross it.
- **ADR-0010 capability matrix** distinguishes capabilities
  that are exercisable locally from those that require
  sandbox cloud access. The sandbox capabilities are anchored
  to GCP projects; the matrix implies one project per non-local
  env.
- **ADR-0005 §4** (manifest publication) — manifest pointers,
  bodies, and rule YAMLs live in an object store. The bucket
  is named per env. A shared bucket would force naming
  conventions to carry the isolation; a per-env bucket makes
  IAM carry it.
- **Foundation 01 §P3** (ownership explicit) — environment
  ownership must be explicit. IAM-per-project gives each
  environment a real owner; shared-project-with-naming-
  conventions diffuses ownership across whoever has IAM access
  to the shared project.

### 3.2 Options

- **(A) Separate GCP project per environment.** Each of
  `local`, `qa`, `prod` owns its own GCP project. **Recommended.**
- **(B) Shared GCP project, per-env BigQuery datasets and
  buckets.** Lower operational overhead (one billing entity,
  one IAM tree). Higher risk: misconfigured `DQ_BIGQUERY_DATASET`
  could write `prod` rows to the `qa` dataset; the isolation
  boundary becomes a naming convention rather than IAM.
- **(C) Shared GCP project with prefix conventions.** Same
  pattern as (B) but for buckets / topics rather than only
  datasets. Strictly worse than (B): more naming convention
  surface to police.

### 3.3 Recommendation

**(A) Separate GCP project per environment.** **New
contribution proposed here, requires review** — foundation 04
§PAT-4 commits the configuration shape; ADR-0010's
capability matrix implies a local-vs-sandbox split; neither
directly commits project-per-env. B1-4 makes the commitment
explicit so a future amendment can ratify or reject it on its
own merits.

`local` is the Compose substrate (not a GCP project per se,
but isolated by virtue of being local-only). `qa` and `prod`
each own a dedicated GCP project. Object store buckets,
BigQuery projects, BigQuery datasets, and Pub/Sub topics live
inside their env's project; no resource is shared across envs.

This is the cleanest isolation boundary:

- IAM is per-project. The default GCP IAM posture is no
  cross-project access; a `qa`-credentialed service account
  cannot read `prod` data without an explicit grant. The
  boundary is enforced by IAM, not by a naming convention.
- Billing, audit logs, and quota are naturally per-env. A
  cost spike in `qa` cannot drain `prod`'s budget; a `prod`
  outage's audit trail is not entangled with `qa` debugging
  events.
- A misconfigured deployment variable (e.g., a typo in the
  `prod` overlay's `DQ_BIGQUERY_PROJECT`) typically fails
  loud (the IAM grant doesn't exist) rather than silently
  routing writes to the wrong env.

Considered and rejected: **(B)** and **(C)**, both of which
make the env boundary a naming convention. The deterministic-
engine driver (P2) and the explicit-ownership driver (P3)
both push toward IAM-as-boundary, which only (A) provides.

### 3.4 Consequences

- **C-MD-3.1.** W3-P7's overlay artifacts encode per-env GCP
  project IDs as their wiring. The overlays are the
  ground-truth source for "which project does this env touch?".
- **C-MD-3.2.** The `local` environment is *not* a GCP
  project; it is the Compose substrate per ADR-0010. The
  engine binary still receives `DQ_BIGQUERY_PROJECT=dq-local`
  via the local overlay; the BigQuery emulator interprets the
  project ID as a namespace, not as a billing entity.
- **C-MD-3.3.** Provisioning the `qa` and `prod` GCP projects
  is an operational prerequisite for the first deployment;
  it is not in the W3-P7 scaffolding scope. The W3-P7 PR
  documents this prerequisite in its README.
- **C-MD-3.4.** Cross-project access (e.g., the engine in
  `qa` reading source data from another GCP project) is
  out of scope at v1. If it becomes a requirement, the
  cross-project IAM grant is an explicit deployment artifact
  in the relevant overlay, not an implicit fallback.

### 3.5 Open Questions

- **OQ-MD-3.1.** Whether `prod` is a single GCP project or
  multiple (regional / per-tenant). **Out-of-scope for
  current cycle — v1 of the platform commits one `prod`
  GCP project; cross-region / multi-tenant layouts are a
  separate future ADR.**
- **OQ-MD-3.2.** Whether the `qa` project gets full mirror
  parity with `prod` (every prod resource has a qa twin) or
  selective parity (only resources the rule-evolution flow
  exercises). **Out-of-scope for current cycle — a W3-P7
  follow-up; the v1 overlay is "everything mirrored" by
  default; selective parity is a future optimization.**

---

## 4. MD-4 — PAT-4 confirmation + the runtime refactor scope

### 4.1 Drivers

- **Foundation 04 §PAT-4** is already a committed pattern.
  This study cannot revisit it; it can only confirm and
  elaborate.
- **`engine/cmd/dq-engine/main.go` `readEnv()`** today is the
  twelve-env-var ad-hoc startup. It does not conform to
  PAT-4. The refactor must land somewhere.
- **ADR-0013 §"Phase 7"** is the next scaffolding phase. It
  is the natural host for the refactor: the typed env package
  is what W3-P7's overlays write into.

### 4.2 Options

- **(A) Confirm PAT-4; refactor lands as W3-P7's first
  scaffolding step.** **Recommended.**
- **(B) Confirm PAT-4; refactor lands as a separate
  pre-W3-P7 session.** Adds a session boundary without
  changing the work. Defensible if W3-P7 turns out to be too
  large; rejected on the basis of cohesion.
- **(C) Reopen PAT-4 itself.** R3 (settled architecture not
  revisited without strong cause) forbids this without a
  concrete identified inconsistency, and none has surfaced.
  Rejected.

### 4.3 Recommendation

**(A) Confirm PAT-4; refactor lands as W3-P7's first
scaffolding step.**

PAT-4 commits the shape: one Go file per environment in
`engine/internal/env/`, typed `EnvConfig` struct, selection
via an env-selector variable (illustrated as `DQ_ENV` in
foundation 04; the concrete selector name is committed in
W3-P7's refactor step), no dynamic discovery or implicit
fallbacks.

B1-4 elaborates PAT-4 with one binding rule. **New
contribution proposed here, requires review** — PAT-4's
"no dynamic discovery, no inheritance chains, no implicit
fallbacks" wording is consistent with the rule but does not
directly commit it:

> **Every field appears in every env file.** Adding a field
> to one env file must fail the local build of the engine
> until the same field is added to all env files. If `qa.go`
> declares a field that `prod.go` does not, the build
> cannot proceed — at minimum via a CI-enforced check, ideally
> via compile-time evidence as well.

The enforcement *mechanism* is committed to W3-P7's refactor
step, not pre-decided here. Several mechanisms satisfy the
invariant — positional struct literals; a generated env
file produced from a single source list; a CI-side lint check
asserting field-set equality across per-env declarations; or
the Go toolchain's struct-tag exhaustiveness primitives. Go's
named-field struct literals (`EnvConfig{Field1: x}`) do *not*
on their own produce a compile error when fields are omitted;
that aspect is what makes mechanism choice a real W3-P7
decision rather than a triviality. B1-4 commits the invariant;
W3-P7 picks the mechanism that delivers it.

The refactor scope:

- Replace `cmd/dq-engine/main.go` `readEnv()` with a single
  `env.Select(<selector>)` call that returns the typed
  `EnvConfig` for the active env.
- The selector validates that the selected value is one of
  the closed set from MD-2 (`local` / `qa` / `prod`); other
  values fail loud at startup.
- Emulator-only overrides (`STORAGE_EMULATOR_HOST`, etc.)
  remain env vars — they are local-substrate concerns, not
  application configuration.

### 4.4 Consequences

- **C-MD-4.1.** W3-P7's first scaffolding step creates the
  `engine/internal/env/` package and migrates the existing
  twelve env-var reads into it. The W3-P7 PR's body documents
  this as "Step 1: PAT-4 refactor".
- **C-MD-4.2.** The `EnvConfig` struct's field set is the
  union of what the twelve existing env vars wire — no
  shrinkage, no new behaviour fields. Any *new* field added
  later is a separate sub-decision.
- **C-MD-4.3.** Existing integration tests (which currently
  construct env vars directly) migrate to constructing
  `EnvConfig` values directly. The test surface for
  configuration becomes typed.
- **C-MD-4.4.** A future contributor temptation to add a
  field to one env file but not the others is mechanically
  prevented by the compiler.

### 4.5 Open Questions

- **OQ-MD-4.1.** Whether emulator-only env vars
  (`STORAGE_EMULATOR_HOST`, `BIGQUERY_EMULATOR_HOST`,
  `PUBSUB_EMULATOR_HOST`) move into the typed struct or stay
  as ad-hoc env vars. **Out-of-scope for current cycle — they
  are substrate overrides honoured by the GCP SDKs directly
  (PAT-4 commits engine configuration, not SDK plumbing).
  The W3-P7 refactor session can revisit if a clean place
  emerges.**
- **OQ-MD-4.2.** Whether the typed struct includes
  validation tags (e.g., `validate:"required,fqdn"`) or
  relies on plain Go zero-value semantics. **Out-of-scope
  for current cycle — implementation choice in the W3-P7
  refactor; plain Go is the default.**

---

## 5. MD-5 — CLI tools (`dq-manifest`, `dq-lint`)

### 5.1 Drivers

- **Both CLIs are one-shot CI invocations.** Each
  `dq-manifest publish` run targets a single bucket; each
  `dq-lint` run validates a single rules tree. They do not
  hold state across invocations; they have no notion of
  "current environment" while running.
- **Flags are deployment-time inputs.** A CI workflow that
  invokes `dq-manifest publish --bucket dq-qa-rules` is
  encoding the env's wiring at invocation time; flags are
  what env-typed configuration would be in a longer-running
  process.
- **No env-driven behaviour exists in either CLI.** The
  linter validates against a fixed schema mirror; the
  publisher writes a fixed-shape manifest. There is no flag
  or env var that changes *what* either tool does — only
  *where* the artifacts live.

### 5.2 Options

- **(A) CLIs stay flag-based; no typed env struct.**
  **Recommended.**
- **(B) CLIs adopt a PAT-4-shaped env package mirroring the
  engine.** Adds complexity for no isolation benefit — each
  CI workflow that invokes a CLI would have to set `DQ_ENV`
  in addition to the bucket/project flags. Rejected.

### 5.3 Recommendation

**(A) CLIs stay flag-based.** Flags are deployment-time
inputs by another name; the env model classifies CI workflow
flag values as deployment-bucket items. The `dq-manifest`
and `dq-lint` binaries themselves are env-agnostic
artifacts; the env-specific knowledge lives in the workflow
that invokes them.

### 5.4 Consequences

- **C-MD-5.1.** Future CI workflow PRs that wire `dq-manifest
  publish` for `qa` and `prod` set the appropriate flags
  per env; no CLI-side change is required.
- **C-MD-5.2.** The `dq-lint` invocation in `make demo-p6`
  (W3-P6d) is local-env-specific by virtue of running on the
  local Compose substrate; no env-awareness in the linter is
  needed.

### 5.5 Open Questions

- **OQ-MD-5.1.** Whether a future tool that *does* span
  envs (e.g., a hypothetical "promote-ruleset-from-qa-to-prod"
  utility) needs PAT-4-shaped configuration. **Out-of-scope
  for current cycle — such a tool does not exist yet; the
  decision is taken in the session that introduces it.**

---

## Consequences (cross-cutting)

The five MDs together commit the platform to a specific
configuration posture going forward:

- **CC1.** Three buckets (code / deployment / data); three
  first-class environments (`local` / `qa` / `prod`);
  separate GCP project per environment; PAT-4 confirmed with
  the every-field-in-every-env compiler-enforcement rule;
  CLIs flag-based.
- **CC2.** W3-P7 is unblocked. Its scope:
  - Step 1: PAT-4 refactor — move `cmd/dq-engine/main.go`
    `readEnv()` into `engine/internal/env/{local,qa,prod}.go`
    with a `Select(DQ_ENV) EnvConfig` selector.
  - Step 2+: Kubernetes manifests (`deploy/base/`) + three
    overlays (`deploy/overlays/local/`, `…/qa/`, `…/prod/`).
    The overlay tool (Kustomize / Helm / etc.) is W3-P7's
    own decision.
- **CC3.** Any future configuration item must be classified
  into one of the three buckets at the point of introduction.
  PR descriptions for new env vars / rule fields must name
  their bucket explicitly.
- **CC4.** Provisioning the `qa` and `prod` GCP projects is
  an operational prerequisite for the first cloud deployment;
  it is not in the W3-P7 scaffolding scope. The W3-P7 PR
  documents this prerequisite.
- **CC5.** Subsequent B1 sessions land on top of this study:
  per-env cost ceilings (foundation 05) consume the env
  enumeration committed here; the secrets-storage primitive
  consumes the deployment-bucket commitment.

---

## Open Questions (cross-cutting)

- **OQ-CC.1.** Promotion ordering with B1-10 and B1-11.
  **Out-of-scope for current cycle — both B1-10 and B1-11
  carry stale provisional ADR slots (their study metadata
  reserved `0014` and `0015`, but ADR-0014 was taken by
  the W3-P4e trigger-handler-contract promotion in PR #9;
  see the §"Promotion target" section below for the same
  framing). The promotion session that runs first re-numbers
  to the actual next free slot at that time; B1-10 / B1-11 /
  B1-4 may promote in any order.** (new contribution
  proposed here, requires review)
- **OQ-CC.2.** Whether a future "platform-test" env (an
  inter-team integration sandbox) is added. **Out-of-scope
  for current cycle — the three-env closed set is firm;
  adding a fourth is an explicit amendment.**
- **OQ-CC.3.** Documentation of the env model for
  contributors. **Out-of-scope for current cycle — W3-P8
  (`docs/` content) is the natural host for an "environments"
  contributor doc, sourcing this study + the promoted ADR.**

---

## Promotion target

`docs/adr/0016-environment-configuration-model.md`
(provisional).

The next free ADR slot at the time of writing is `0015`
(ADR-0014 is W3-P4e's trigger-handler-contract; ADRs
0001–0013 are the Phase-1 promotions). Both B1-10 and B1-11
have provisional slots (`0014` and `0015` per their own study
metadata at the time they were written) that are stale —
ADR-0014 was taken by the P4e promotion. B1-4's study
provisionally claims `0016` to leave `0015` for whichever of
B1-10 or B1-11 promotes first; the promotion session can
re-number if a different free slot is more convenient. The
provisional slot is a citation hint for cross-study coherence,
not a hard reservation.

On promotion, ADR-0016 carries the five MD-N recommendations
as separate Decision sub-sections; the per-MD Consequences
(`C-MD-N.M`) and per-MD Open Questions (`OQ-MD-N.M`) are
renumbered into a single ADR-level Consequence list and Open
Question list, as is the precedent for prior promoted ADRs.
