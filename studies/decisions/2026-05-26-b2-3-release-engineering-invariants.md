<!-- path: studies/decisions/2026-05-26-b2-3-release-engineering-invariants.md -->

# B2-3 — Release Engineering Invariants

## Context

The repository carries three Go modules (`engine/`,
`tools/lint/`, `tools/manifest/`), a `rules/` workspace
containing YAML rule artefacts and a schema mirror, a
`deploy/` workspace with Kubernetes manifests under
`base/` + `overlays/{local,qa,prod}/`, and a `docs/`
workspace. Each workspace evolves independently per
W2-5's per-workspace tag convention, which committed
four prefixes (`engine-v*`, `rules-v*`,
`tools-lint-v*`, `deploy-v*`) and deferred the
second-tools-binary case to OQ-W2-5.1 — a question
that became live when `tools/manifest/` landed and
that this ADR closes.

Several release-engineering surfaces exist already:

- **The root `Makefile`** carries
  per-workspace `lint-<ws>`, `test-<ws>`,
  `build-<binary>` targets plus roll-up `lint` and
  `test` aggregators (see `Makefile` lines 22-83). The
  pattern is observable but not committed as a
  workspace-onboarding contract.
- **`docker-compose.yml`** brings up the local
  substrate (BigQuery emulator, fake-gcs-server,
  Pub/Sub emulator) per ADR-0010.
- **`deploy/base/`** carries the engine's Kubernetes
  manifests with `dq-engine:placeholder` as the image
  reference; no Dockerfile exists yet (per W3-P7b's
  posture that the image marker is committed and the
  release-engineering pipeline is the B2-3 follow-up).
- **W2-5** commits the per-workspace tag prefixes
  (e.g., `engine-v1.2.0`), the no-pipe-character
  invariant (W2-5 §C-W2-5.3), and the literal
  ruleset-version flow into manifest +
  `execution_id` hashing.

What's not yet committed:

- **No Dockerfile shape.** Today's `dq-engine:placeholder`
  image marker is a forward reference. When the engine
  starts shipping as a real container image, what
  Dockerfile shape do all binary-producing workspaces
  follow? Multi-stage? Base image posture?
  Non-root runtime per W3-P7b's existing pod-security
  posture?
- **No Make-target inventory contract.** The
  `lint-<ws>` / `test-<ws>` / `build-<binary>` pattern
  is followed in practice but not committed. A new
  workspace landing tomorrow has no documented set of
  targets it must expose.
- **No image-name ↔ git-tag derivation rule.** W2-5
  commits the git-tag shape (`engine-v1.2.0`). What
  image tag does that produce? `dq-engine:1.2.0`?
  `dq-engine:engine-v1.2.0`? An operator running `helm
  template ... --set engineImage=dq-engine:X` needs
  a deterministic answer.
- **W2-5's deferred questions are still open**:
  - **C-W2-5.5** — per-workspace tag prefix enforcement
    (a CI gate rejecting `engine-v*` pointing at a
    `rules/` change) was flagged as "new contribution
    proposed here, requires review" and never landed.
  - **OQ-W2-5.1** — whether `tools-lint-v*` covers all
    of `tools/` or only the linter. A second tool
    (`tools/manifest`) has since landed.
- **No deeper deploy-manifest validation lane.** W3-P7c
  closing notes explicitly deferred `kubeconform`
  integration / kind-based cluster validation to B2-3
  or a Phase-7 follow-up.

The B2-3 row registered the question at the W3
backlog-numbering step:

> Which Docker, Make, and versioning invariants are
> mandatory across all workspaces? Long-term repo
> health depends on consistent ergonomics.

The principles bearing on the decision are **P4**
(cost is a first-class constraint — release engineering
that diverges per workspace amplifies operator and
contributor cost over time), **P5** (evolution must be
contract-driven — the workspace-onboarding contract
must be documented so a new workspace lands
predictably), and **R3** (do not revisit settled
architecture — W2-5, ADR-0019, ADR-0034 are preserved;
this ADR extends them with the workspace-onboarding
contract on top).

What B2-3 must commit:

1. **The cross-workspace release-engineering
   invariants** — what every workspace MUST satisfy
   to be releasable.
2. **The Docker shape** for binary-producing
   workspaces.
3. **The Make-target inventory** every workspace MUST
   expose.
4. **The image-name ↔ git-tag derivation** rule.
5. **W2-5's deferred follow-ups** that this ADR closes
   (or explicitly defers further).
6. **The deferred implementation slices** —
   Dockerfile content, CI tag gate, kubeconform lane.

---

## Decision Drivers

- **DD-1 — Observable convention deserves a committed
  contract.** The Makefile lines 22-83 already follow
  the `lint-<ws>` / `test-<ws>` / `build-<binary>`
  pattern. Committing this as a contract turns a
  per-session-improvised convention into a documented
  invariant new workspaces inherit predictably.

- **DD-2 — Cross-workspace consistency reduces
  contributor cost.** A contributor running `make
  lint` on a clean checkout expects every workspace
  to be linted; the root aggregator (`lint:
  lint-engine lint-tools`) currently delivers that. If
  a new workspace adds `lint-newws` without updating
  the aggregator, `make lint` quietly skips it. The
  contract commits both the per-workspace target AND
  the aggregator-update obligation.

- **DD-3 — Versioning invariants from W2-5 are
  already committed; this ADR cites + extends.** The
  per-workspace tag prefixes, the no-pipe-character
  invariant, and the literal ruleset-version flow are
  all already settled. This ADR commits the unfinished
  W2-5 follow-ups (C-W2-5.5 CI gate; OQ-W2-5.1 second-
  tool prefix) and adds the image-tag derivation
  rule.

- **DD-4 — Docker invariants must be committable
  without shipping a Dockerfile.** The Dockerfile
  itself is deferred (the engine binary doesn't yet
  ship as an image; `dq-engine:placeholder` is a
  forward reference per W3-P7b). The *shape*
  invariants — multi-stage build, minimal final
  image, non-root runtime, fixed `<workspace>/Dockerfile`
  path — can be committed now and bind the future
  Dockerfile when it lands. This matches the
  honest-gap pattern from ADR-0030 / ADR-0041.

- **DD-5 — Image tag derives mechanically from the
  git tag.** The simplest rule: strip the
  `<workspace>-v` prefix and the result is the image
  tag (e.g., `engine-v1.2.0` → `dq-engine:1.2.0`).
  Any operator can compute the image tag from a git
  tag without consulting a per-workspace lookup.

- **DD-6 — Make-target inventory is a small fixed
  set.** Three verbs cover the cross-workspace
  contract: `lint`, `test`, `build`. Substrate-
  dependent verbs (integration tests, smoke tests)
  are conditional and follow an additive convention
  rather than a mandatory one. Build target is
  conditional on the workspace producing a binary
  (the `rules/` workspace has no binary to build).

- **DD-7 — Defer the implementation slices.** The
  Dockerfile content, the CI tag-gate, the
  kubeconform validation lane — each is a real
  deliverable that needs its own session. Following
  the design-only pattern set by ADR-0030 / ADR-0032
  / ADR-0033 / ADR-0039 / ADR-0041, this ADR commits
  the invariants; the implementations are
  individually registered as B2 follow-up rows.

- **DD-8 — Bound the scope.** Release engineering
  spans CI, image registry choice, secrets posture,
  release-cadence rhythm, rollback procedures, and
  more. R4 (one topic per session) limits this ADR
  to the *cross-workspace invariants*. Single-
  workspace release decisions (e.g., the engine's
  specific Dockerfile multi-stage targets) are
  per-workspace work, not cross-workspace contract.
  CI image-build pipeline, secrets posture, registry
  choice are all deferred.

---

## Considered Options

### Option 1 — Commit cross-workspace invariants now; defer implementation slices to B2 follow-ups (recommended)

This ADR commits the **release-engineering invariants
contract** spanning Docker, Make, and versioning
surfaces. Implementation slices (the engine
Dockerfile, the CI tag-gate, kubeconform integration)
ship as separate B2 follow-up rows.

The contract has four clauses:

**Clause 1 — Docker invariants** (binding when a
workspace ships a binary as a container image; not
binding for workspaces that don't ship images):

- The Dockerfile lives at `<workspace>/Dockerfile`
  (e.g., `engine/Dockerfile`, `tools/lint/Dockerfile`).
- The build is multi-stage: a builder stage compiles
  the Go binary; a minimal runtime stage carries only
  the binary and its runtime dependencies.
- The runtime image runs as **non-root** with
  `readOnlyRootFilesystem: true`, dropped Linux
  capabilities, and a seccomp profile of
  `RuntimeDefault` — mirroring the pod-security
  posture W3-P7b already commits.
- The image name follows `dq-<binary-name>:<tag>`
  (e.g., `dq-engine:1.2.0`, `dq-lint:0.1.0`,
  `dq-manifest:0.1.0`).
- The image tag derives from the git tag by
  stripping the `<workspace>-v` prefix (e.g.,
  `engine-v1.2.0` → `dq-engine:1.2.0`).

**Clause 2 — Make invariants** (binding for every
workspace; rules-workspace exception noted):

- Every binary-producing workspace MUST expose three
  Make verbs from the root Makefile:
  - `lint-<workspace-name>` — vets / lints the
    workspace's source.
  - `test-<workspace-name>` — runs the workspace's
    unit tests (no substrate).
  - `build-<binary-name>` — builds the workspace's
    binary into `bin/<binary-name>`.
- Workspaces that don't produce a Go binary
  (`rules/`, `docs/`, `deploy/`) skip the `build`
  verb; the `lint-rules` / `validate-deploy` pattern
  for those workspaces is documented but the
  binary-build verb is not mandatory.
- Root aggregator targets `lint` and `test` MUST
  include every per-workspace target. A new
  workspace landing without updating the aggregator
  is a contract violation; the existing CI lane
  catches this (lint or test never running against
  the new workspace's code surfaces in coverage gaps
  long before a release).
- Substrate-dependent test targets follow the
  `test-<ws>-integration` suffix convention and are
  excluded from the root `test` aggregator.

**Clause 3 — Versioning invariants** (citing W2-5 +
closing W2-5's deferred follow-ups):

- Per-workspace tag prefixes are W2-5's committed
  set extended here: `engine-v*`, `rules-v*`,
  `tools-lint-v*`, `tools-manifest-v*`, `deploy-v*`.
  **Closes OQ-W2-5.1**: `tools-manifest-v*` is the
  prefix for the manifest publisher binary,
  **separate from** `tools-lint-v*`. Each
  `tools/<binary>` directory gets its own prefix.
- Tags are immutable. Force-tagging is forbidden
  (any tag rewrite is treated as a new tag at the
  CI surface). **New contribution proposed here,
  requires review** — W2-5 does not state this
  explicitly.
- No pipe character anywhere in any tag value
  (W2-5 §C-W2-5.3 carried forward).
- **Closes C-W2-5.5**: the per-workspace tag-prefix
  CI gate is committed as a new lane that rejects a
  push of `engine-v*` whose diff-from-prior-tag
  touches files outside `engine/`, and equivalently
  for other prefixes. Implementation is deferred to
  a B2 follow-up; the contract is binding now.
- The image tag derives from the git tag by
  stripping the `<workspace>-v` prefix
  (`engine-v1.2.0` → `dq-engine:1.2.0`).

**Clause 4 — Deploy-manifest deeper-validation
posture** (closing W3-P7c's deferral):

- The cluster-free `kubectl kustomize` lane
  (Makefile lines 115-120) is the v1 validation
  surface. It catches YAML syntax errors, missing-
  resource references, patch-target mismatches, and
  strategic-merge conflicts.
- Deeper schema validation — field-name typos like
  `replicass:`, deprecated API versions, etc. —
  ships as a B2 follow-up via a deeper-validation
  lane (e.g., `kubeconform`, a kind-based cluster,
  or equivalent) integrated into the existing
  `make validate-deploy` target. The specific tool
  is deferred to the B2 follow-up slice; the
  contract binds the follow-up to extend the
  existing target, not introduce a new CI lane.

**Strengths.** Closes the cross-workspace contract
gap (DD-1 / DD-2) without dragging implementation
into the same session (DD-7). Closes two open W2-5
follow-ups (OQ-W2-5.1, C-W2-5.5) and the W3-P7c
deferral. Honors the design-only pattern. Bounded
scope per DD-8.

**Trade-offs.** Implementation slices remain
deferred: the engine Dockerfile (the most acute
need — `dq-engine:placeholder` is a live image
marker), the CI tag-gate, the kubeconform lane.
Each is a separate B2 row, registered in the
decision-log update.

### Option 2 — Commit invariants + ship the engine Dockerfile in this session

Same contract as Option 1 plus the
`engine/Dockerfile` ships under this ADR.

**Strengths.** Closes the most acute implementation
gap immediately.

**Trade-offs.** Dockerfile content is a real
deliverable: base-image choice, build-cache layout,
runtime-user setup, healthcheck wiring, etc. Each
requires its own review surface. Committing the
contract + the Dockerfile in one session conflates
two reviews. Following the precedent of ADR-0030 /
ADR-0032 / ADR-0033 / ADR-0039 / ADR-0041 (each
committed design without shipping the consumer
code), the Dockerfile lands as a separate slice.
Rejected.

### Option 3 — Commit only the Docker shape; defer Make and versioning extensions

Commit Clause 1 only; leave Make conventions and
W2-5 follow-up closure to separate ADRs.

**Strengths.** Smallest commit.

**Trade-offs.** Splits one logical decision into
three ADRs. The contributor-facing contract is
"here's how a workspace integrates into release";
splitting it across multiple ADRs means a new
workspace must consult three ADRs to onboard.
Rejected — bundle the cross-workspace contract in
one ADR.

---

## Recommendation

**Option 1.** Cross-workspace release-engineering
invariants in one ADR; implementation slices deferred
to B2 follow-up rows.

### Clause 1 — Docker invariants

Every workspace that ships a binary as a container
image follows these shape invariants:

| Invariant | Binding |
|---|---|
| Dockerfile lives at `<workspace>/Dockerfile` | Mandatory |
| Multi-stage build (builder stage + minimal runtime stage) | Mandatory |
| Runtime image is non-root with `readOnlyRootFilesystem: true`, dropped capabilities, `RuntimeDefault` seccomp | Mandatory (mirrors W3-P7b) |
| Image name = `dq-<binary-name>:<tag>` (e.g., `dq-engine:1.2.0`) | Mandatory |
| Image tag = git tag with `<workspace>-v` prefix stripped (e.g., `engine-v1.2.0` → `dq-engine:1.2.0`) | Mandatory |
| Specific base image, build-cache layout, healthcheck wiring | Per-workspace; not cross-workspace contract |

The Dockerfile itself, the base-image choice, and
the CI build-and-push pipeline are deferred to a B2
follow-up slice per workspace.

### Clause 2 — Make invariants

Every workspace contract is the inventory of root
Make targets that name the workspace:

| Verb | Pattern | Binding |
|---|---|---|
| Lint | `lint-<workspace>` | Mandatory for every workspace with source files |
| Test | `test-<workspace>` | Mandatory for every workspace with testable code |
| Build | `build-<binary>` | Mandatory for binary-producing workspaces; n/a for `rules/`, `docs/`, `deploy/` |
| Integration test | `test-<workspace>-integration` | Optional; uses Go build tag `integration` per ADR-0034 |
| Substrate verbs | `up`, `down`, `smoke-substrate`, `demo-p6` | Cross-workspace; not per-workspace |

Root aggregators MUST include every per-workspace
target:

- `lint:` MUST include every `lint-<ws>` target.
- `test:` MUST include every `test-<ws>` target.
- A new workspace landing without updating the
  aggregator is a contract violation; review-time
  enforcement is the platform-team CODEOWNERS rule
  on `/Makefile`.

Help-text convention (`## ` after the target name,
parsed by the existing `awk` snippet in
`Makefile:32`) is the cross-workspace
documentation surface; every target carries one.

### Clause 3 — Versioning invariants

Committed by W2-5; this ADR confirms + closes two
open W2-5 items + adds the image-tag derivation:

| Invariant | Source |
|---|---|
| Per-workspace tag prefixes: `engine-v*`, `rules-v*`, `tools-lint-v*`, `tools-manifest-v*`, `deploy-v*` | W2-5 + this ADR (closes OQ-W2-5.1 — `tools-manifest-v*` is its own prefix) |
| Pipe character forbidden in any tag value | W2-5 §C-W2-5.3 |
| Tags immutable; force-tag forbidden | New contribution proposed here, requires review |
| Per-workspace tag-prefix CI gate (a push of `engine-v*` MUST diff only against `engine/`) | W2-5 §C-W2-5.5 — implementation deferred to a B2 follow-up |
| Image tag = git tag with `<workspace>-v` prefix stripped | New contribution proposed here, requires review |
| Pre-release suffix shape: `<workspace>-v<semver>-<suffix>` (e.g., `engine-v1.2.0-rc1`) | W2-5 §C-W2-5.4 (already pipe-free) |
| Pre-release publication rules (whether a pre-release manifest can write `latest.json`) | W2-5 §OQ-W2-5.3 — still open; deferred |

### Clause 4 — Deploy-manifest deeper-validation posture

Closes W3-P7c's deferral:

- The v1 validation surface is `kubectl kustomize`
  rendering (Makefile `validate-deploy` target,
  lines 115-120). It catches YAML syntax errors,
  missing-resource references, patch-target
  mismatches, and strategic-merge conflicts.
- Deeper validation (field-name typos, deprecated
  API versions, schema compliance against the
  Kubernetes API surface) ships via a deeper-
  validation lane (e.g., `kubeconform`, a
  kind-based cluster, or equivalent) integrated
  into the existing `make validate-deploy` target.
  The specific tool is deferred to the B2 follow-up
  slice; the follow-up extends the existing target,
  does NOT introduce a new top-level CI lane.

### Why this does not reopen W2-5 / ADR-0019 / ADR-0034 / W3-P7

- **W2-5** is cited as the source of versioning
  invariants. The two unfinished W2-5 items
  (OQ-W2-5.1 second-tool prefix, C-W2-5.5 CI gate)
  close here; W2-5's settled clauses are preserved
  without amendment.
- **ADR-0019** committed Kustomize as the deploy
  tooling. Clause 4 cites it; no amendment.
- **ADR-0034** committed the local-testing taxonomy
  including the `integration` build tag. Clause 2's
  `test-<ws>-integration` convention cites it
  without amendment.
- **W3-P7b** committed the pod-security posture
  (non-root, readOnlyRootFilesystem, dropped caps,
  RuntimeDefault seccomp). Clause 1 mirrors it for
  the *image* posture (the pod inherits the image's
  runtime properties), closing the symmetry between
  pod manifest and image shape. No amendment to
  W3-P7b.

---

## Consequences

1. **A cross-workspace release-engineering contract
   is committed.** Three clauses (Docker, Make,
   Versioning) plus the deploy-validation extension
   live in one ADR. A new workspace onboarding
   tomorrow consults this ADR to know which targets
   to expose, which Dockerfile shape to ship, and
   which tag prefix to claim.

2. **Two open W2-5 items close.** `tools-manifest-v*`
   is committed as its own tag prefix
   (OQ-W2-5.1 → resolved). The per-workspace
   tag-prefix CI gate (C-W2-5.5) is committed at the
   contract level; implementation deferred to a B2
   follow-up.

3. **W3-P7c's deferral closes at the contract
   level.** A deeper-validation lane integrated
   into `validate-deploy` is the committed
   follow-up; the specific tool (e.g.,
   `kubeconform`, kind-based cluster, or
   equivalent) is decided by the B2 implementation
   slice.

4. **The image-name ↔ git-tag derivation is
   mechanical.** Stripping the `<workspace>-v` prefix
   from a git tag yields the image tag. Operators
   compute it without consulting a per-workspace
   lookup.

5. **The non-root + read-only-root pod-security
   posture from W3-P7b extends to the image
   layer.** Dockerfile invariants enforce the same
   posture inside the image so the pod's runtime
   guarantees are derivable from the image alone.

6. **The Make-target inventory is documented.**
   A new workspace MUST expose `lint-<ws>`,
   `test-<ws>`, and (if it produces a binary)
   `build-<binary>`. Root aggregators MUST include
   every per-workspace target — the platform-team
   CODEOWNERS rule on `/Makefile` enforces this at
   review time.

7. **Three new B2 rows register the implementation
   slices** that this ADR commits at the contract
   level:
   - **Engine Dockerfile** (closes the
     `dq-engine:placeholder` forward reference from
     W3-P7b).
   - **Per-workspace tag-prefix CI gate** (closes
     W2-5 §C-W2-5.5).
   - **Deeper deploy-validation lane integrated
     into `validate-deploy`** (closes W3-P7c's
     deferral; specific tool decided by the slice).

8. **B2-3 closes.** The decision-log B2-3 row moves
   to `resolved-adr`. Three new B2 rows register
   the implementation slices per Consequence #7.

9. **W2-5, ADR-0019, ADR-0034, W3-P7b are
   preserved.** This ADR layers the
   workspace-onboarding contract on top of their
   commitments without amending.

---

## Open Questions

None blocking.

Two deferred items surfaced during drafting and are
explicitly **out-of-scope for current cycle**:

- **OQ-1: Image registry choice.** This ADR commits
  the image-name shape (`dq-<binary>:<tag>`) but not
  the registry where images are pushed. Registry
  choice depends on the ADR-0008 host-primitive
  follow-up (the same operational session that
  substitutes `PLACEHOLDER-org/` per ADR-0015 §4).
  Reserved as a follow-up to that session.

- **OQ-2: Release-cadence rhythm.** When is an
  `engine-v*` tag pushed? Weekly? On-demand? When a
  specific test-suite gate passes? Release cadence
  is operational governance, not workspace-contract
  invariants. Reserved until concrete
  release-cadence signal surfaces from operating
  the platform.

---

## Promotion target

`docs/adr/0042-release-engineering-invariants.md` —
next free ADR number. Ships the four-clause
cross-workspace release-engineering contract
(Docker, Make, Versioning, Deploy-validation).
