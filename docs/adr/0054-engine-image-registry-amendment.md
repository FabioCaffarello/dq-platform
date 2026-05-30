<!-- path: docs/adr/0054-engine-image-registry-amendment.md -->

# ADR-0054 — Engine Image Registry (amends ADR-0042)

- **Status:** accepted (amends ADR-0042)
- **Date:** 2026-05-30

---

## Context

[ADR-0042](./0042-release-engineering-invariants.md)
committed the cross-workspace release-engineering contract
on 2026-05-26. Its Clause 1 §"Docker invariants" specifies
the image-name shape as `dq-<binary-name>:<tag>` (e.g.,
`dq-engine:1.2.0`), with the image tag derived from the
git tag by stripping the `<workspace>-v` prefix.
Consequence 10 §OQ-1 left the **registry choice**
explicitly deferred:

> "OQ-1: Image registry choice. This ADR commits the
> image-name shape (`dq-<binary>:<tag>`) but not the
> registry where images are pushed. … Reserved as a
> follow-up to that session."

Three downstream artifacts have carried that deferral as
a load-bearing TBD since ADR-0042 promoted:

- `engine/Dockerfile` (B2-28's slice) builds locally and
  produces `dq-engine:<tag>` per the Makefile's
  `build-engine-image` target. The image is structurally
  correct but carries no registry prefix, so it cannot
  be pushed.
- `.github/workflows/engine-image.yml` runs on every PR
  + push to main, calls `make build-engine-image`, and
  inspects the resulting image's `User` field. The
  workflow comments at lines 6–10 name the deferral
  directly: *"it does NOT push the image to any registry
  because the registry choice is still TBD per ADR-0042
  OQ-1"*.
- `deploy/base/deployment.yaml:55` pinned `image:
  dq-engine:placeholder` — a string `kubectl apply`
  cannot resolve to any pullable image.

This ADR closes OQ-1 by committing **Docker Hub** with
the `fabiocaffarello` namespace as the registry, amends
ADR-0042 Clause 1's image-name row to a two-part shape
(local-build unchanged; published-image new), and
commits the push-step contract the workflow honors. It
also applies the amendment to the three downstream
artifacts in the same change set (see §Notes for the
operator-authorized R4 scope collapse that consolidates
the contract commitment and the wiring slice into one
session).

The principles bearing on this decision are **P4** (cost
is a first-class constraint — Docker Hub's pull-volume
posture and PAT-rotation discipline are real operating
costs traded against the OIDC-chain setup the other three
candidate substrates carry), **P5** (evolution must be
contract-driven — the amendment ships under a published
ADR shape so future re-platforming follows the same
discipline), and **R3** (do not revisit settled
architecture — ADR-0042 Clauses 2, 3, 4 stand unamended;
only Clause 1's image-name row and Consequence 10 §OQ-1
are touched). **R5** (own the pattern, name the
substrate) governs the option-space framing: four
registry substrates were considered in the originating
B2-36 study, all four sitting inside R5's
commodity-environment "and equivalents" exemption (Docker
Hub on Docker; GitHub Container Registry on GitHub;
Google Artifact Registry on GCP per
[ADR-0010](./0010-substrate-posture.md); AWS ECR on AWS).

This ADR follows the
[ADR-0017](./0017-substrate-posture-amendment.md)
standalone-superseding amendment pattern (rather than the
[ADR-0050](./0050-v1-retirement-engine-release.md)
§Consequence 4 in-place Amendment-log convention) because
the amendment carries architectural-prose impact —
registry rationale, auth posture, push semantics — not
just structured-data row touch. ADR-0050 §Consequence
4's convention is reserved for amendments that "touch
only structured data"; this amendment exceeds that scope.
The `Status: accepted (amends ADR-0042)` line follows the
[`adr-writing`](../../.claude/skills/adr-writing/SKILL.md)
A4 idiom.

---

## Decision

### Clause 1 — Registry choice: Docker Hub, `fabiocaffarello` namespace

The engine image's published-image registry is **Docker
Hub** (`docker.io`) under the `fabiocaffarello`
namespace. The choice rests on four properties this
project's specific operational context already carries:

- **Pull volume is low** — one engine pod per cluster
  with restarts measured in days, plus CI rebuilds
  measured in pushes-per-day. Docker Hub's free-tier
  rate limit is comfortably sufficient without paid
  tier.
- **Mental-model simplicity** — no OIDC chain, no
  Workload Identity Federation hop, no cross-cloud
  account to manage. A contributor or operator pulling
  the image from any context does so without
  configuring per-cluster image-pull secrets when the
  namespace is public.
- **PAT rotation discipline** is the only meaningful
  operational cost Docker Hub carries vs. the other
  three substrates considered (GitHub Container
  Registry, Google Artifact Registry, AWS ECR). The
  rotation is a routine credential-hygiene item the
  operator already manages for other accounts.
- **No re-platforming pressure today** — the engine
  runtime substrate per
  [ADR-0010](./0010-substrate-posture.md) is GCP
  (BigQuery, GCS, Pub/Sub), but the deployment surface
  (`deploy/base/` is tool-neutral per
  [ADR-0019](./0019-deploy-tooling.md)) does not pin a
  Kubernetes substrate. Docker Hub does not bind the
  registry decision to a specific cloud.

The other three options (GHCR, Artifact Registry, ECR)
are documented in the originating B2-36 study's
§Considered Options as a starting point for any future
session that re-opens the choice; future re-platforming
rides its own amendment ADR.

### Clause 2 — Amended image-name shape (replaces ADR-0042 Clause 1's image-name row)

[ADR-0042](./0042-release-engineering-invariants.md)
Clause 1's image-name row read:

> "Image name = `dq-<binary-name>:<tag>` (e.g.,
> `dq-engine:1.2.0`) | Mandatory"

This amendment replaces that single row with a two-part
shape that distinguishes local-build flow from
published-image flow:

| Context | Image-name shape | Binding |
|---|---|---|
| **Local build** — the `make build-engine-image` target producing an image for `docker compose` and single-host operator workflows | `dq-<binary-name>:<tag>` (e.g., `dq-engine:1.2.0`) — unchanged from ADR-0042 | Mandatory |
| **Published image** — the image the CI workflow uploads to the registry and the image `deploy/base/deployment.yaml` references | `<registry-host>/<namespace>/dq-<binary-name>:<tag>` (concretely `docker.io/fabiocaffarello/dq-engine:1.2.0`; the shorthand `fabiocaffarello/dq-engine:1.2.0` resolves identically because `docker.io` is the Docker daemon's default registry) | Mandatory |

ADR-0042 Clause 3's tag-derivation rule
(`engine-v1.2.0` → `1.2.0` via prefix-stripping) is
**preserved unamended**. The `<tag>` value is identical
between the local and published shapes.

ADR-0042 Clause 1's other rows (Dockerfile location,
multi-stage requirement, non-root /
`readOnlyRootFilesystem` / dropped capabilities /
`RuntimeDefault` seccomp, per-workspace base-image /
build-cache / healthcheck deferral) are **preserved
unamended** — only the image-name row changes.

### Clause 3 — Push-step behavioral contract

The CI workflow that builds the engine image
(`.github/workflows/engine-image.yml` per ADR-0042
Clause 1's per-workspace CI deferral) gains a push step
honoring four behaviors:

1. **Authenticated push to Docker Hub.** The push step
   authenticates to Docker Hub using a Personal Access
   Token (PAT) supplied via a CI secret. The PAT and
   the username are operator-provisioned in the
   repository's secrets store. The auth step uses the
   official Docker login action.
2. **Image published under the Clause 2 published-image
   shape.** The locally-built `dq-engine:<tag>` image
   is re-tagged to
   `docker.io/fabiocaffarello/dq-engine:<tag>` and
   pushed.
3. **Tagged-build-only activation.** The push activates
   only when the workflow runs against a pushed
   `engine-v*` git tag, not on every PR or main push.
   Untagged builds remain build-only — the existing
   image inspection step continues to run on PRs and
   main pushes.
4. **Pushed digest surfaced for downstream pinning.**
   The pushed image's digest is surfaced in the workflow
   summary (`$GITHUB_STEP_SUMMARY`) so the
   `deploy/base/deployment.yaml` reference can pin by
   digest for the prod overlay. The load-bearing
   property digest pinning preserves is
   **reproducibility**: the deployment manifest pins to
   the exact image bytes the CI lane built, so a
   tag-overwrite at the registry layer cannot silently
   swap the image under the deployment. ADR-0042
   Clause 1's runtime posture (non-root,
   `readOnlyRootFilesystem`, `RuntimeDefault` seccomp,
   dropped capabilities) is enforced by the pod's
   `securityContext` independently of the digest.

PAT rotation is **operator-side discipline**. A
recommended cadence and escalation conditions are
documented in the originating B2-36 study's §Open
Questions OQ-1; rotation events are operator decisions
acted on directly against the secrets store.

### Clause 4 — Closes ADR-0042 Consequence 10 §OQ-1

ADR-0042 Consequence 10 §OQ-1 ("Image registry choice…
Reserved as a follow-up to that session") transitions to
**Resolved by ADR-0054**.

### Clause 5 — Applied effects: the three downstream artifacts

This ADR applies the amendment to the three artifacts
that have carried OQ-1 as a load-bearing TBD:

- **`engine/Dockerfile`** — no edit. The local-build
  flow produces `dq-engine:<tag>` per Clause 2's local
  row; the Dockerfile is correct as-is. The
  registry-prefixed published-image shape is applied at
  push time (Clause 3 item 2), not in the Dockerfile.
- **`.github/workflows/engine-image.yml`** — edited.
  The workflow gains a push step honoring Clause 3's
  four behaviors. The push step requires two repository
  secrets (`DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`)
  whose names are this ADR's choice; the operator
  provisions the secret values directly in the
  repository's secrets store.
- **`deploy/base/deployment.yaml`** — edited. The
  literal `image: dq-engine:placeholder` flips to
  `image: docker.io/fabiocaffarello/dq-engine:0.1.0`
  (the tag matches the current `EngineVersion` baseline
  in `engine/internal/env/{local,qa,prod}.go`). The
  per-line comment that documented the OQ-1 deferral
  is updated to reference this ADR. Overlays at
  `deploy/overlays/{local,qa,prod}/` may further patch
  the image reference (pin by digest, override the tag
  per env) per ADR-0042 Clause 4 + ADR-0019 Kustomize
  overlay framing; the base value is the deterministic
  default.

### Clause 6 — What this ADR does NOT change

R3 hygiene boundaries that this amendment honors:

- **ADR-0042 Clauses 2, 3, 4** stand unamended.
  Make-target inventory, versioning invariants, and
  deploy-validation posture are preserved.
- **ADR-0042 Clause 1's other rows** stand unamended.
  Only the image-name row is touched (Clause 2 above).
- **ADR-0042 Consequence 10 §OQ-2** (release-cadence
  rhythm) stands deferred. This amendment closes OQ-1
  only.
- **The local-build flow** is unchanged. `make
  build-engine-image` continues to produce
  `dq-engine:<tag>` for `docker compose` and single-host
  operator workflows.
- **The engine binary** is unchanged. No code, no env
  config, no dependency moves under this ADR.

---

## Consequences

1. **Docker Hub `fabiocaffarello` is the engine image
   registry.** The CI workflow pushes
   `docker.io/fabiocaffarello/dq-engine:<tag>` on
   `engine-v*` git tag pushes; `deploy/base/deployment.yaml`
   references the same image path. OQ-1 closes.

2. **The image-name shape is in two parts now.** Local
   build keeps `dq-engine:<tag>` (the ADR-0042 Clause 1
   shape); published image carries the
   `docker.io/fabiocaffarello/dq-engine:<tag>` prefix.
   Contributors and operators know which shape applies
   in which context. The `<tag>` value is identical
   across both shapes — ADR-0042 Clause 3's
   prefix-stripping derivation is preserved.

3. **The CI workflow push step requires two
   operator-provisioned secrets:** `DOCKERHUB_USERNAME`
   (string value `fabiocaffarello`) and
   `DOCKERHUB_TOKEN` (Docker Hub PAT with `Write`
   scope on `fabiocaffarello/dq-engine`). Until these
   are provisioned, the workflow's push step fails
   loudly on the `docker login` step — by design;
   silent skips are not the chosen failure mode.
   Build + inspect steps continue to run on PRs and
   main pushes regardless of secret presence (the
   secrets are only consumed by the push step, which
   is conditional on the `engine-v*` tag event).

4. **The first `engine-v*` git tag triggers the first
   push.** Until an `engine-v*` tag is pushed, no
   image lands in the registry. Once the first tag
   pushes (operator action), `kubectl apply -k
   deploy/base/` resolves the deployment image
   successfully. Between this ADR's acceptance and the
   first tag push, the deployment manifest references
   a not-yet-existing image — the *shape* is
   structurally correct; the *image* is the operator's
   next action.

5. **Reproducibility via digest pinning is available
   in overlays.** The CI workflow surfaces the pushed
   image's digest in
   `$GITHUB_STEP_SUMMARY`. Overlays at `deploy/overlays/`
   may pin by digest (`@sha256:…`) instead of by tag
   when the env demands stronger reproducibility (prod
   posture per ADR-0042 Clause 1's pinned-by-digest
   note on `deploy/base/deployment.yaml:58`). The
   base manifest keeps the tag-pinned shape as the
   deterministic baseline.

6. **Future re-platforming has a documented option-
   space starting point.** The originating B2-36
   study's §Considered Options enumerates GitHub
   Container Registry, Google Artifact Registry, AWS
   ECR, and Docker Hub. A future session that re-opens
   the registry choice (operational signal, cost
   change, security posture shift) has the comparison
   recorded and does not re-derive the trade-off math
   from scratch.

7. **R3 hygiene preserved.** ADR-0042 Clauses 2, 3, 4
   stand unamended; Clause 1's non-image-name rows
   stand unamended; Consequence 10 §OQ-2 stands
   deferred. Reviewers verify by reading this ADR's
   Clause 6 + Notes alongside ADR-0042's unchanged
   sections.

8. **R5 hygiene preserved.** The four registry
   substrates considered in the originating study sit
   inside R5's commodity-environment "and equivalents"
   exemption. The choice rests on this project's
   specific operational context (pull volume,
   credential-hygiene posture, mental-model
   simplicity), not on external pattern.

9. **The amendment-via-standalone-ADR pattern from
   [ADR-0017](./0017-substrate-posture-amendment.md)
   is reused.** This ADR's `Status: accepted (amends
   ADR-0042)` line follows the
   [`adr-writing`](../../.claude/skills/adr-writing/SKILL.md)
   A4 idiom. ADR-0042 itself is preserved unchanged on
   disk; the amended state is read by combining
   ADR-0042's Clauses 2–4 + Clause 1's non-image-name
   rows with this ADR's Clauses 1–4.

10. **B2-36 closes with `resolved-adr`.** The
    decision-log row transitions from
    `resolved-study` to `resolved-adr` with this
    ADR's filename. No follow-on B-row is registered
    by this ADR — the wiring that the originating
    study deferred is **applied in this same change
    set** per the operator-authorized scope collapse
    documented in §Notes.

---

## Notes

### Operator-authorized R4 scope collapse

The originating B2-36 study's locked premise P-AmRR.2
deferred the **wiring slice** (the
`.github/workflows/engine-image.yml` push-step edit,
the CI secret plumbing, and the `deploy/base/deployment.yaml`
placeholder flip) to a follow-on B-row per
[`CLAUDE.md`](../../CLAUDE.md) §3 R4 (one topic per
session). At promotion time, the operator authorized
collapsing the wiring slice into this ADR's PR — the
contract commitment and the applied effects ship
together rather than separately.

The rationale carried by the operator's directive: the
amendment's substance is small (a registry choice and
the two-line workflow + manifest edits that follow
mechanically from it), and the in-flight context (the
originating study just merged, the registry choice is
fresh, the CI lane to apply against is the one this
session is editing) does not benefit from a separate
session's re-grounding. The R4 split that P-AmRR.2
committed was a default discipline; the operator
overrides defaults when context warrants.

This Note records the override **on the durable
audit surface** so future sessions reading ADR-0054 see
the scope decision was made deliberately, not absorbed
silently. Future B2-row amendments that follow B2-36's
shape should default to the P-AmRR.2-style split
(contract + applied slice in two sessions) unless the
operator authorizes a collapse for analogous reasons.

### CI secret names

This ADR chooses `DOCKERHUB_USERNAME` and
`DOCKERHUB_TOKEN` as the two repository secret names
the push step consumes. The naming follows the
convention the `docker/login-action@v3` action's
documentation surfaces, but the names themselves are
this ADR's choice (operator-side reservation, made at
promotion time, not at B2-36 study time). A future
change to the secret names (e.g., a multi-environment
rotation where each env has its own PAT) is an
amendment surface; the names committed here are the
v1 binding.

### Why `0.1.0` as the base manifest tag

`deploy/base/deployment.yaml` flips from
`dq-engine:placeholder` to
`docker.io/fabiocaffarello/dq-engine:0.1.0`. The `0.1.0`
tag matches the current `EngineVersion` baseline in
`engine/internal/env/{local,qa,prod}.go`. Until the
first `engine-v0.1.0` git tag pushes and the workflow
uploads the image, the deployment manifest references
a not-yet-existing image — same structural posture as
the `:placeholder` state, but with the correct path.

The `latest` tag was considered and rejected: `latest`
is a mutable rolling tag, and the originating study's
Clause 3 item 4 commits **reproducibility** as a
load-bearing property of digest pinning. A `latest` base
tag normalizes the anti-pattern of mutable references at
the manifest layer; a concrete-version base tag pins by
intent, with overlays optionally upgrading to digest
pinning per environment.

Future base-tag bumps (e.g., when the engine releases
`0.2.0` per [ADR-0050](./0050-v1-retirement-engine-release.md))
are operator-side discipline applied at release time.
Whether the base manifest tracks the latest released
version or pins to a specific historical version is a
release-cadence decision deferred to ADR-0042
Consequence 10 §OQ-2.

### Open question carried forward from the originating study

The originating B2-36 study registered four §Open
Questions: PAT rotation cadence (OQ-1); multi-arch image
build (OQ-2 — the Dockerfile already honors
`TARGETARCH`); image signing via cosign/Sigstore (OQ-3);
public vs. private Docker Hub namespace (OQ-4, with
default-assumption-public committed). None of these
block this ADR's acceptance. Each carries forward to a
future amendment session if and when its trigger
condition surfaces.

### Why the base manifest does not gate on secret presence

> **Amended 2026-05-30 (post-acceptance, operator-
> authorized).** This block originally committed a
> "fail-loudly-on-missing-secrets" posture for the publish
> step. A subsequent operator-authorized override aligned
> the publish step's gating with the established
> `sandbox.yml` / `lint-reachability.yml` workflow
> discipline — silent skip when the publish gate is
> closed, with a visible notice block in the run summary.
> The original posture below is preserved for the audit
> trail; the **current** posture is recorded in the
> sibling block "Publish-gate dual-authorization
> (post-acceptance override)" below.

The `.github/workflows/engine-image.yml` push step is
**conditional on the `engine-v*` tag event**, not on the
secrets' presence. If the secrets are missing when an
`engine-v*` tag is pushed, the push step fails loudly at
`docker login` — by design. The build + inspect steps
that run on every PR and main push do not consume
secrets and continue to operate identically to the
pre-amendment state. This means:

- PR and main builds keep working unchanged on every
  contributor's PR, regardless of whether they have
  access to the Docker Hub secrets.
- The first `engine-v*` tag push surfaces the secret
  state — if the operator has not yet provisioned the
  secrets, the workflow fails with a clear `docker
  login` error pointing at the missing reference;
  there is no silent skip.

This posture is committed by Clause 3 item 1's
"authenticated push" language and Clause 5's "fails
loudly" framing.

### Publish-gate dual-authorization (post-acceptance override)

A subsequent operator-authorized override aligned the
publish step's gating with the established
[`sandbox.yml`](../../.github/workflows/sandbox.yml) /
[`lint-reachability.yml`](../../.github/workflows/lint-reachability.yml)
workflow discipline. The publish steps in
`.github/workflows/engine-image.yml` are now dual-gated:
they activate only when **both** conditions hold:

1. **The push event is an `engine-v*` tag.** (Unchanged
   from Clause 3 item 3.)
2. **The publish gate is open** — either
   `vars.ENGINE_IMAGE_PUBLISH_ENABLED` is set to
   `'true'` *or* `secrets.DOCKERHUB_TOKEN` is non-empty.

If condition 1 holds but condition 2 does not, the
publish steps silently skip and a notice step emits a
"## ADR-0054 publish stage — skipped" block to
`$GITHUB_STEP_SUMMARY` explaining the gate is closed and
naming the two ways to open it. The build + inspect
stage continues to run on every PR + main push +
engine-v* tag push regardless of gate state.

Rationale for the override: the original "fail loudly"
posture surfaces missing secrets as a workflow failure
on the first `engine-v*` tag push, which couples the
release-cut moment (tag push) to the operator's
secret-provisioning state. The dual-gate decouples the
two — an operator can cut release tags without
provisioning publish credentials yet, and the publish
moment is a separate, explicit operator opt-in (set the
variable or provision the token). This matches how
`sandbox.yml` and `lint-reachability.yml` already gate
their respective workloads.

The override is documented here on the durable audit
surface rather than re-promoted via a new amendment ADR
because: (a) the change is a Notes posture clarification,
not a Decision clause change; (b) Clause 3's behavioral
contract (authenticated push of a tagged build to the
committed path) is unchanged — only the *activation
criterion* tightens; (c) the change aligns with prior
workflow-discipline patterns committed by sandbox.yml /
lint-reachability.yml, not new ground. A future
substantive amendment that changes Clause 3 itself would
follow the standalone-superseding-ADR pattern.

The sibling block above ("Why the base manifest does not
gate on secret presence") is preserved verbatim with an
amendment banner for the audit trail.
