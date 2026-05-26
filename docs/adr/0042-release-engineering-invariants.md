<!-- path: docs/adr/0042-release-engineering-invariants.md -->

# ADR-0042 — Release Engineering Invariants

- **Status:** accepted
- **Date:** 2026-05-26

---

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

- **The root `Makefile`** carries per-workspace
  `lint-<ws>`, `test-<ws>`, `build-<binary>` targets
  plus roll-up `lint` and `test` aggregators (lines
  22-83). The pattern is observable but not committed
  as a workspace-onboarding contract.
- **`docker-compose.yml`** brings up the local
  substrate (BigQuery emulator, fake-gcs-server,
  Pub/Sub emulator) per ADR-0010.
- **`deploy/base/`** carries the engine's Kubernetes
  manifests with `dq-engine:placeholder` as the image
  reference; no Dockerfile exists yet (per W3-P7b's
  posture that the image marker is committed and the
  release-engineering pipeline is the B2-3 follow-up).
- **W2-5** commits the per-workspace tag prefixes
  listed above, the no-pipe-character invariant
  (W2-5 §C-W2-5.3), and the literal ruleset-version
  flow into the manifest + `execution_id` hashing.

What's not yet committed at this ADR's writing:

- **No Dockerfile shape.** Today's
  `dq-engine:placeholder` image marker is a forward
  reference.
- **No Make-target inventory contract.** The
  `lint-<ws>` / `test-<ws>` / `build-<binary>`
  pattern is followed in practice but not committed
  as an onboarding contract for new workspaces.
- **No image-name ↔ git-tag derivation rule.** W2-5
  commits `engine-v1.2.0`; the image tag derivation
  has been ambiguous.
- **W2-5's deferred follow-ups remain open** —
  C-W2-5.5 (tag-prefix CI gate) and OQ-W2-5.1
  (second-tool prefix).
- **W3-P7c's deferred deeper-validation lane** for
  `kubectl kustomize` (kubeconform / kind cluster /
  equivalent).

The principles bearing on the decision are **P4**
(cost is a first-class constraint — release
engineering that diverges per workspace amplifies
operator and contributor cost over time), **P5**
(evolution must be contract-driven — the
workspace-onboarding contract must be documented so a
new workspace lands predictably), and **R3** (do not
revisit settled architecture — W2-5, ADR-0019,
ADR-0034 are preserved; this ADR extends them with
the workspace-onboarding contract on top).

---

## Decision

The cross-workspace release-engineering contract has
four clauses spanning Docker, Make, Versioning, and
Deploy-validation surfaces. Implementation slices
(the engine Dockerfile, the CI tag-gate, the
deeper-validation lane) ship as separate B2 follow-up
rows.

### Clause 1 — Docker invariants

Every workspace that ships a binary as a container
image follows these shape invariants. Workspaces that
don't ship images (`rules/`, `docs/`, `deploy/`) are
not bound by this clause.

| Invariant | Binding |
|---|---|
| Dockerfile lives at `<workspace>/Dockerfile` | Mandatory |
| Multi-stage build (builder stage + minimal runtime stage) | Mandatory |
| Runtime image is non-root with `readOnlyRootFilesystem: true`, dropped capabilities, `RuntimeDefault` seccomp | Mandatory (mirrors W3-P7b's pod-security posture in the image layer) |
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
`Makefile:32`) is the cross-workspace documentation
surface; every target carries one.

### Clause 3 — Versioning invariants

Committed by W2-5; this ADR confirms + closes two
open W2-5 items + adds the image-tag derivation:

| Invariant | Source |
|---|---|
| Per-workspace tag prefixes: `engine-v*`, `rules-v*`, `tools-lint-v*`, `tools-manifest-v*`, `deploy-v*` | W2-5 + this ADR (closes OQ-W2-5.1 — `tools-manifest-v*` is its own prefix) |
| Pipe character forbidden in any tag value | W2-5 §C-W2-5.3 |
| Tags immutable; force-tag forbidden | **New contribution proposed here, requires review** — W2-5 does not state this explicitly |
| Per-workspace tag-prefix CI gate (a push of `engine-v*` MUST diff only against `engine/`) | W2-5 §C-W2-5.5 — implementation deferred to a B2 follow-up |
| Image tag = git tag with `<workspace>-v` prefix stripped | **New contribution proposed here, requires review** |
| Pre-release suffix shape: `<workspace>-v<semver>-<suffix>` (e.g., `engine-v1.2.0-rc1`) | W2-5 §C-W2-5.4 (already pipe-free) |
| Pre-release publication rules (whether a pre-release manifest can write `latest.json`) | W2-5 §OQ-W2-5.3 — still open; deferred |

### Clause 4 — Deploy-manifest deeper-validation posture

Closes W3-P7c's deferral at the contract level:

- The v1 validation surface is `kubectl kustomize`
  rendering (Makefile `validate-deploy` target,
  lines 115-120). It catches YAML syntax errors,
  missing-resource references, patch-target
  mismatches, and strategic-merge conflicts.
- Deeper validation (field-name typos, deprecated
  API versions, schema compliance against the
  Kubernetes API surface) ships via a
  deeper-validation lane (e.g., `kubeconform`, a
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
   is committed.** Four clauses (Docker, Make,
   Versioning, Deploy-validation) live in one ADR. A
   new workspace onboarding tomorrow consults this
   ADR to know which targets to expose, which
   Dockerfile shape to ship, and which tag prefix
   to claim.

2. **Two open W2-5 items close.**
   `tools-manifest-v*` is committed as its own tag
   prefix (OQ-W2-5.1 → resolved). The per-workspace
   tag-prefix CI gate (C-W2-5.5) is committed at the
   contract level; implementation deferred to a B2
   follow-up.

3. **W3-P7c's deferral closes at the contract
   level.** A deeper-validation lane integrated into
   `validate-deploy` is the committed follow-up; the
   specific tool (e.g., `kubeconform`, kind-based
   cluster, or equivalent) is decided by the B2
   implementation slice.

4. **The image-name ↔ git-tag derivation is
   mechanical.** Stripping the `<workspace>-v`
   prefix from a git tag yields the image tag.
   Operators compute it without consulting a
   per-workspace lookup.

5. **The non-root + read-only-root pod-security
   posture from W3-P7b extends to the image layer.**
   Dockerfile invariants enforce the same posture
   inside the image so the pod's runtime guarantees
   are derivable from the image alone.

6. **The Make-target inventory is documented.** A
   new workspace MUST expose `lint-<ws>`,
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
   - **Deeper deploy-validation lane integrated into
     `validate-deploy`** (closes W3-P7c's deferral;
     specific tool decided by the slice).

8. **B2-3 closes.** The decision-log B2-3 row moves
   to `resolved-adr`. Three new B2 rows register
   the implementation slices per Consequence #7.

9. **W2-5, ADR-0019, ADR-0034, W3-P7b are
   preserved.** This ADR layers the
   workspace-onboarding contract on top of their
   commitments without amending.

10. **Two deferred items registered out-of-scope:**

    - **OQ-1: Image registry choice.** This ADR
      commits the image-name shape
      (`dq-<binary>:<tag>`) but not the registry
      where images are pushed. Registry choice
      depends on the ADR-0008 host-primitive
      follow-up (the same operational session that
      substitutes `PLACEHOLDER-org/` per ADR-0015
      §4). Reserved as a follow-up to that session.

    - **OQ-2: Release-cadence rhythm.** When is an
      `engine-v*` tag pushed? Weekly? On-demand?
      When a specific test-suite gate passes?
      Release cadence is operational governance, not
      workspace-contract invariants. Reserved until
      concrete release-cadence signal surfaces from
      operating the platform.
