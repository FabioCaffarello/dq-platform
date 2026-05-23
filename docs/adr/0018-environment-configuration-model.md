<!-- path: docs/adr/0018-environment-configuration-model.md -->

# ADR-0018 — Environment Configuration Model

- **Status:** accepted
- **Date:** 2026-05-23

---

## Context

The platform's runtime depends on per-environment wiring — GCP
project identifiers, object-store bucket names, BigQuery dataset
identifiers, Pub/Sub topic names, refresh intervals, the listener
address — that varies between local development, sandbox-cloud
verification, and production. ADR-0013 §"Phase 7" names exactly
three first-class environments (`local`, `qa`, `prod`) and
explicitly conditions Phase 7 of Wave 3 (the Kubernetes-manifest
scaffolding) on a settled configuration model.

Foundation 04 §PAT-4 ("Typed multi-environment configuration")
already commits the shape the engine should evolve toward — one Go
file per environment under `engine/internal/env/`, a typed
`EnvConfig` struct, selection at startup, no dynamic discovery, no
implicit fallbacks. PAT-4 settles the file-layout shape but leaves
three structural questions open: which bucket each configuration
item lives in overall (code, deployment, or data); which
environments are first-class; and how environments are isolated at
the substrate level.

Foundation 01 §P2 commits that the engine is deterministic — the
same rule version, time window, and source state must produce the
same execution semantics. Behaviour driven by an environment
variable is therefore a defect, not a feature. The configuration
model must keep *wiring* (per-environment deployment values) cleanly
separated from *behaviour* (which belongs in code or in versioned
data).

ADR-0010's capability matrix splits substrates into **Yes**
(exercisable locally), **Partial**, and **No** (sandbox-required)
rows. ADR-0017 amends the matrix's object-store CAS row to
**Partial**. These commitments imply but do not directly commit
that local and sandbox environments are physically isolated at the
substrate level.

This ADR commits five tightly-coupled decisions: the three-bucket
configuration model (where each kind of configuration lives
overall); the closed set of first-class environments (`local`,
`qa`, `prod`); the substrate-isolation posture (separate GCP project
per non-local environment); the PAT-4 confirmation with one
binding refinement (the every-field-in-every-env enforcement
invariant); and the CLI tools' relationship to environments (flag-
based, env-agnostic). These decisions are coupled tightly enough
that they belong in one ADR.

**Out of scope of this ADR:**

- Specific per-environment cost ceilings (foundation 05 §"Cost
  Discipline" defers these to their own future decision).
- The Kubernetes overlay tool used by Wave 3 Phase 7 — ADR-0019
  (infrastructure tooling) commits Kustomize as the choice; this
  ADR commits the *shape* of per-environment wiring without
  binding the overlay tool.
- The secrets-storage primitive (KMS / external vault /
  sealed-secrets) — a separate future decision.
- `DQ_ENV` value canonicalization rules and the precise
  enforcement mechanism of the every-field-in-every-env
  invariant — these are implementation choices of the Wave 3
  Phase 7 refactor session.
- Cross-region or multi-tenant production layouts — this ADR
  commits one `prod` GCP project at v1; expansion to multiple
  production projects is a future amendment.
- OIDC service-identity flows beyond what ADR-0010 already
  commits.

---

## Decision

### 1. Three-bucket configuration model — code, deployment, data

Every configuration item lives in exactly one of three buckets.
Each bucket has a primary source-of-truth location, a primary
evolution channel, and a primary audience.

- **Code (compile-time invariants).** Source of truth: the
  engine's Go source. Evolution channel: an ADR plus a Go code
  change plus a version bump. Audience: engineers. Examples
  include the schema versions (ADR-0001), the `execution_id`
  formula (ADR-0002 §1), the closed `trigger_source` enum
  (ADR-0002 §3), the closed `ExecutionStatus` enum (ADR-0003
  §6), and the engine's strict request decoder (ADR-0014 §2).
- **Deployment (per-environment wiring).** Source of truth: the
  deployment artifact (Wave 3 Phase 7 Kubernetes manifests plus
  per-env overlays). Evolution channel: a deployment change,
  with no engine rebuild required. Audience: operators.
  Examples include GCP project identifiers, object-store bucket
  names, BigQuery dataset names, Pub/Sub topic names, the
  engine's listener address, refresh intervals, log level, and
  the environment selector itself.
- **Data (control-plane content).** Source of truth: the
  object store (rule manifests) or the repository (the
  `_owners.yaml` file). Evolution channel: `dq-manifest publish`
  for rule content (ADR-0005 §4), or a repository commit and
  redeploy of the rules workspace for `_owners.yaml`.
  Audience: domain teams and alert owners. Examples include
  rule YAMLs, `_owners.yaml`, and the manifest pointer.

The bucket split is binding. A future temptation to add an
environment variable that controls *behaviour* (e.g.,
`DQ_BEHAVIOR_FLAG=…`) must be answered "behaviour belongs in
code or in data, not deployment" and rerouted.

The data bucket admits more than one evolution channel by design.
Rule YAMLs flow through `dq-manifest publish` (CAS-versioned, per
ADR-0005); `_owners.yaml` flows through repository commits because
the linter cross-checks it at lint time (ADR-0006 §9). A
contributor must not move `_owners.yaml` into `yamls/by-hash/`
thinking it should match rule YAMLs — the channel difference is
deliberate.

The code and deployment buckets do not admit alternate channels.

### 2. First-class environments — `local`, `qa`, `prod`

The set of first-class environments is closed at three.

- **`local`** — developer laptops and CI. Uses the Compose
  substrate per ADR-0010. **Yes** rows of the capability matrix
  are exercised here; **Partial** and **No** rows are not.
- **`qa`** — a dedicated sandbox GCP project. The sandbox-
  required ADR-0010 rows are wired here: object-store CAS
  enforcement (per ADR-0017's amended row), OIDC service
  identity, the unforgeable linter pin. Used for
  pre-production verification of rule changes and engine
  releases.
- **`prod`** — the production GCP project, the canonical
  data-plane. The same wiring as `qa` plus per-environment
  configuration values (alert routing channels, listener
  posture) that match production.

Adding a fourth environment (e.g., a `staging` tier) requires
amending this ADR. The closed set keeps the deployment surface
finite — exactly three overlay directories, exactly three Go
files under `engine/internal/env/`, exactly three CI lanes if
the CI shape ever fans out by env.

Existing test fixtures that mention environment names not in the
closed set (e.g., a fixture asserting that the parser accepts
arbitrary keys in severity-override maps) are testing parser
behaviour, not asserting that the named env exists. The fixtures
do not need to change.

### 3. Substrate isolation — separate GCP project per non-local environment

`qa` and `prod` each own a dedicated GCP project. Object-store
buckets, BigQuery projects, BigQuery datasets, and Pub/Sub
topics live inside their environment's project; no substrate
resource is shared across environments. `local` is the Compose
substrate per ADR-0010 — not a GCP project, but isolated by
virtue of being local-only.

Two alternatives are rejected:

- **Shared GCP project with per-env BigQuery datasets and
  buckets.** Lower operational overhead but turns the
  environment boundary into a naming convention rather than IAM;
  a misconfigured `DQ_BIGQUERY_DATASET` could route `prod`
  writes to the `qa` dataset.
- **Shared GCP project with prefix conventions across all
  resources.** Strictly worse — more naming-convention surface
  for the platform to police, with no compensating benefit.

The separate-project posture follows from foundation 01 §P2
(deterministic engine, behaviour cannot be env-driven) and §P3
(explicit ownership). IAM-per-project gives each environment a
real owner; shared-project layouts diffuse ownership across
whoever has IAM access to the shared project.

Three practical consequences fall out: (a) IAM is the enforcement
boundary, not a naming convention — a `qa`-credentialed service
account cannot read `prod` data without an explicit grant; (b)
billing, audit logs, and quota are naturally per-environment —
a cost spike in `qa` cannot drain `prod`'s budget, and a `prod`
audit trail is not entangled with `qa` debugging events; (c) a
misconfigured deployment variable typically fails loud (the IAM
grant doesn't exist) rather than silently routing writes to the
wrong environment.

Cross-project access — for example, an engine in `qa` reading
source data from another GCP project — is out of scope at v1. If
it becomes a requirement, the cross-project IAM grant is an
explicit deployment artifact in the relevant overlay, not an
implicit fallback.

### 4. PAT-4 confirmed; every-field-in-every-env invariant

Foundation 04 §PAT-4 is confirmed without revision: one Go file
per environment under `engine/internal/env/` (`local.go`,
`qa.go`, `prod.go`); each file declares an `EnvConfig` struct of
identical shape; selection at startup via an env-selector
variable; no dynamic discovery, no inheritance chains, no
implicit fallbacks.

This ADR adds one binding refinement that PAT-4's "no dynamic
discovery, no implicit fallbacks" wording is consistent with but
does not directly commit:

> **Every field appears in every environment file.** Adding a
> field to one environment file must fail the build until the
> same field is added to all environment files. If `qa.go`
> declares a field that `prod.go` does not, the build cannot
> proceed.

The enforcement mechanism — positional struct literals, code
generation from a single source list, a CI-side reflection check
asserting field-set equality across per-env declarations, or
another approach — is the responsibility of the engine
configuration package itself. Go's named-field struct literals
do not on their own produce a compile error when fields are
omitted; that aspect is what makes the mechanism choice a real
implementation decision rather than a triviality. This ADR
commits the invariant; the package commits the mechanism.

Refactor scope:

- The `cmd/dq-engine/main.go` startup is reduced to a single
  call that returns the typed `EnvConfig` for the active
  environment.
- The selector validates that the selected value is one of
  `local` / `qa` / `prod`; other values fail loudly at startup.
- Emulator-only overrides (`STORAGE_EMULATOR_HOST`,
  `BIGQUERY_EMULATOR_HOST`, `PUBSUB_EMULATOR_HOST`) remain
  ad-hoc environment variables — they are substrate concerns
  honoured by the GCP SDKs directly, not application
  configuration.

The `EnvConfig` struct's field set is the union of what the
existing ad-hoc startup wires — no shrinkage, no new behaviour
fields. Any *new* field added later is a separate decision; the
every-field invariant ensures the addition is visible across all
environments.

### 5. CLI tools — flag-based, environment-agnostic

The platform's CLI tools (`dq-manifest`, `dq-lint`, and any
similarly shaped successor) stay flag-based. They do not adopt a
PAT-4-shaped env package; they have no notion of a "current
environment" while running.

CLI invocations are one-shot CI calls. Each `dq-manifest publish`
targets a single bucket; each `dq-lint` validates a single rules
tree. Flags are the natural input shape; flags *are*
deployment-time inputs by another name, and the env model
classifies CI workflow flag values as deployment-bucket items.

No flag or environment variable changes *what* either tool does
— only *where* the artifacts live. There is no env-driven
behaviour to isolate. Adding a typed env struct would force every
CI workflow that invokes a CLI to set an environment selector in
addition to the bucket/project flags, with no compensating
isolation benefit.

If a future tool genuinely spans environments — for example, a
hypothetical "promote ruleset from qa to prod" utility — the
session that introduces it decides whether to adopt the PAT-4
shape. This ADR's commitment is that today's CLIs do not.

---

## Consequences

1. The three buckets (code / deployment / data), the three
   environments (`local` / `qa` / `prod`), the separate-GCP-
   project-per-non-local-environment posture, the PAT-4 shape
   with the every-field-in-every-env invariant, and the
   flag-based CLI surface together form the platform's
   configuration posture going forward. Future configuration
   work consumes these commitments.

2. Future configuration items must be classified into one of
   the three buckets at the point of introduction. The PR
   description for any new environment variable, rule field, or
   deployment value names its bucket explicitly. A new value
   that does not fit cleanly into a bucket reopens this ADR.

3. The `_owners.yaml` file's evolution channel — repository
   commits, not `dq-manifest publish` — is a deliberate
   deviation from full data-plane semantics. The data bucket
   admits this deviation because the linter cross-checks
   `_owners.yaml` at lint time per ADR-0006 §9; moving the file
   into `yamls/by-hash/` would defeat the cross-check.

4. The three-bucket split prohibits a fourth bucket. A future
   contributor proposing a "runtime feature flag" bucket or a
   "remote configuration service" bucket reopens this ADR.

5. The `local` environment is not a GCP project. The engine
   binary receives a project-shaped identifier (e.g.,
   `dq-local`) via the local overlay; the BigQuery emulator
   interprets that identifier as a namespace, not as a billing
   entity. The deployment-bucket value carries through; the
   substrate interpretation differs.

6. Provisioning the `qa` and `prod` GCP projects is an
   operational prerequisite for the first cloud deployment. It
   is not covered by the Wave 3 Phase 7 scaffolding scope; the
   Phase 7 artifact carries the placeholder identifiers (e.g.,
   `dq-{qa,prod}-PLACEHOLDER`) that the operational session
   substitutes mechanically when the real GCP projects are
   created.

7. Cross-project IAM grants — when they become necessary — are
   explicit deployment artifacts in the relevant overlay, not
   implicit fallbacks. The default posture is no cross-project
   access; the IAM grant is the audit trail that records the
   deliberate exception.

8. The every-field-in-every-env invariant prevents drift between
   environments at build time. A contributor adding a field to
   `qa.go` cannot land the change without also adding it to
   `local.go` and `prod.go`. The compiler (or a CI check, if
   the package uses a reflection-based mechanism) is the
   enforcement surface.

9. The engine's existing ad-hoc environment-variable startup
   migrates to the typed `engine/internal/env/` package in
   Wave 3 Phase 7's first scaffolding step. The migration is
   union-preserving: every existing wiring value gets a typed
   field; no field is dropped silently and no behaviour field is
   added in the same change.

10. Integration tests that previously constructed environment
    variables directly migrate to constructing `EnvConfig`
    values directly. The configuration surface becomes typed at
    the test boundary as well as at the binary boundary.

11. The CLIs' flag-based shape lets each CI workflow encode the
    environment it targets without coupling the CLI binary to a
    PAT-4-shaped struct. A future CI workflow that publishes to
    `qa` and to `prod` sets the appropriate flags per
    invocation; no CLI-side change is required.

12. The `qa` environment is the default landing zone for
    sandbox-required validation. ADR-0017's sandbox lane runs
    against `qa`; the unforgeable linter pin (ADR-0001) is
    validated against `qa`; the OIDC sandbox flows (ADR-0010)
    are validated against `qa`. `prod` runs only after `qa`
    validation passes.

13. Subsequent decisions consume this ADR's commitments. The
    per-environment cost ceilings (a future B1 per foundation
    05) consume the three-environment enumeration; the
    secrets-storage primitive consumes the deployment-bucket
    commitment; the Wave 3 Phase 7 Kubernetes overlay
    scaffolding consumes the substrate-isolation commitment.

14. Reopening this ADR is required to add a fourth first-class
    environment, to introduce a fourth configuration bucket, to
    move from separate-GCP-project-per-env to a shared-project
    layout, or to revise the every-field-in-every-env
    invariant. Adding a new typed field to `EnvConfig` is not a
    reopen — it is a routine code change subject to the
    every-field invariant.

---

## Notes

- The five sub-decisions are coupled tightly enough that one
  ADR rather than five separate ones is the right shape. The
  three-bucket model and the substrate-isolation posture only
  make sense together; the PAT-4 confirmation only makes sense
  with the closed environment set; the CLI ruling only makes
  sense in contrast to the engine's typed struct.

- The every-field-in-every-env invariant is the only refinement
  this ADR makes to PAT-4's letter. Every other element of
  PAT-4 — the per-env Go file layout, the typed struct, the
  selector at startup, the prohibition on dynamic discovery —
  is carried forward unchanged.

- The flag-based CLI ruling intentionally leaves the door open
  for a future env-spanning tool to adopt PAT-4 if needed. The
  ruling is "today's CLIs do not need it", not "no CLI may
  ever need it". The session that introduces such a tool
  decides.

- The deployment-bucket items committed in §1 are the
  configuration values that *change between environments*. A
  configuration value that is the same in every environment is
  a code-bucket item even if it is currently injected via an
  environment variable at boot; the bucket classification
  follows the variability, not the wiring mechanism.
