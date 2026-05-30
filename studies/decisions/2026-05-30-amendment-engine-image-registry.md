<!-- path: studies/decisions/2026-05-30-amendment-engine-image-registry.md -->

# B2-36 — Amendment to ADR-0042: engine image registry

## Metadata

- **Wave reference:** Amendment to
  [ADR-0042](../../docs/adr/0042-release-engineering-invariants.md)
  §"Clause 1 — Docker invariants" + §Consequences §10 (OQ-1).
  Decision-log placement: B2-36 (implementation-phase row
  for an amendment-class outcome — same convention as
  [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md)'s
  B2-20 v1-retirement amendment).
- **Status:** draft (B2-36 amendment, session 1;
  post-round-2-critique; two-round cap reached per
  [`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  step 7; ready to move to `resolved-study` after
  decision-log row update lands).
- **Last updated:** 2026-05-30.
- **Upstream resolved:**
  [ADR-0042](../../docs/adr/0042-release-engineering-invariants.md)
  (Clause 1 image-name shape + Consequence 10 OQ-1
  deferral — this study amends Clause 1's image-name row
  and closes OQ-1);
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
  §(a) (outcome taxonomy: this study verifies the work as
  Amendment, not B3, not B2-standalone);
  [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md)
  §Consequence 4 (Amendment-log subsection convention
  available for the amendment ADR if the amendment shape
  permits in-place data-row touch; this study's
  amendment is prose-substantial enough to ride the
  [ADR-0017](../../docs/adr/0017-substrate-posture-amendment.md)
  standalone-superseder pattern instead);
  [ADR-0017](../../docs/adr/0017-substrate-posture-amendment.md)
  (standalone-superseding-ADR amendment pattern this
  study's promotion target follows);
  [ADR-0010](../../docs/adr/0010-substrate-posture.md) /
  [ADR-0019](../../docs/adr/0019-deploy-tooling.md) (the
  GCP-substrate posture and Kustomize deploy tooling that
  bound on the registry decision).
- **Eligibility check (ADR-0049 §(a) outcome
  classification):**
  - **§(a) outcome — Amendment.** This study **modifies**
    [ADR-0042](../../docs/adr/0042-release-engineering-invariants.md)'s
    Clause 1 image-name shape (`Image name = dq-<binary-name>:<tag>`)
    by extending it to carry a registry prefix
    (`<registry-host>/<namespace>/dq-<binary-name>:<tag>`),
    and closes Consequence 10's OQ-1. Per ADR-0049 §(a)'s
    test ("does this proposal change what the ADR decided,
    or does it build on top of what the ADR decided? If
    the former, amendment"), this is **amendment** — the
    image-name shape changes. ✅ Amendment.
  - **Why not B3.** ADR-0049 §(a) B3 requires expanding an
    existing capability without rewriting (P-B3.1). This
    proposal does not add a new capability; it changes the
    shape of a committed contract (the image-name pattern).
    A B3-N entry would be wrong because B3 entries cannot
    rewrite committed contract shape — per
    [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
    §(a) Amendment vs. B3, "amendments live under the
    originating ADR's supersession chain (or as a follow-up
    ADR superseding the originating one), never under B3."
    ❌ Not B3.
  - **Why not B2-standalone.** B2 (implementation-phase
    decision against an in-flight wave) does not fit
    cleanly because Wave 3 closed 2026-05-23 and
    [ADR-0042](../../docs/adr/0042-release-engineering-invariants.md)
    promoted 2026-05-26. The deferred OQ-1 is *inside an
    accepted ADR*, not inside an in-flight wave. The
    decision-log B2-36 placement is the standard convention
    for amendment-class outcomes (precedent: B2-20 v1
    retirement amendment landing as
    [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md));
    B2 here is a *decision-log placement axis* distinct
    from the *§(a) outcome axis*. ❌ Not B2-standalone.
  - **Why not Rejected.** The proposal closes a deferred
    OQ that
    [ADR-0042](../../docs/adr/0042-release-engineering-invariants.md)
    explicitly reserved; there is no "no lane fits"
    surface. ❌ Not Rejected.
- **Constraint envelope:**
  [ADR-0042](../../docs/adr/0042-release-engineering-invariants.md)
  Clause 1 (Docker invariants — the row this study
  amends), Clause 3 (versioning — preserved unamended),
  Consequence 10 (OQ-1 — closes);
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
  §(a) (Amendment outcome classification — the
  authoritative test);
  [ADR-0010](../../docs/adr/0010-substrate-posture.md) +
  [ADR-0019](../../docs/adr/0019-deploy-tooling.md) (the
  GCP-substrate posture and Kustomize deploy tooling
  remain authoritative; this study works inside their
  envelope);
  [`CLAUDE.md`](../../CLAUDE.md) §3 R1–R8 (R1 historical;
  R3 do-not-revisit applies to ADR-0042's *other* clauses
  — only Clause 1's image-name row and Consequence 10
  OQ-1 are touched; R5 own-the-pattern applies to the
  registry comparison below);
  [`CLAUDE.md`](../../CLAUDE.md) §4 P1–P6 (especially P4
  cost-as-first-class — registry pull rate limits and
  storage cost surface here; P5 contract-driven — the
  amendment ships under a published ADR shape).
- **Locked premises** (operator-declared, not litigated
  here):
  - **P-AmRR.1** — **Operator-chosen registry: Docker Hub
    `fabiocaffarello` namespace.** The operator declared
    Docker Hub (`docker.io/fabiocaffarello/dq-engine`) as
    the registry destination at session open. The study
    presents the option space below (D1) for the
    decision-log record and for future reviewers; the
    Recommendation lands on Docker Hub per operator
    declaration. Future re-platforming (e.g., GKE workload
    eventually justifying Artifact Registry) rides its own
    amendment ADR; this amendment does not pre-commit a
    migration path.
  - **P-AmRR.2** — **Scope: registry choice + image-name
    pattern + push-step contract + deployment-manifest
    flip *shape*.** The amendment commits four
    contract-layer items: which registry; the image-name
    pattern (local + published); the push step's
    behavioral contract (what must happen); and the
    eventual image-reference shape the
    `deploy/base/deployment.yaml` placeholder flips to.
    The actual **wiring** — the
    [`.github/workflows/engine-image.yml`](../../.github/workflows/engine-image.yml)
    edit, the CI secret plumbing, and the placeholder
    flip itself in
    [`deploy/base/deployment.yaml`](../../deploy/base/deployment.yaml)
    — is deferred to a follow-on B-row per
    [`CLAUDE.md`](../../CLAUDE.md) §3 R4 (one topic per
    session). The amendment commits *what*; the wiring
    slice commits *how*.
  - **P-AmRR.3** — **No platform code or workflow file
    changes in this study.** R1 is historical, but R4
    binds: this study touches only `studies/decisions/`,
    `studies/critiques/`, and the decision-log row. The
    workflow edit and the manifest flip belong to the
    deferred B-row.
  - **P-AmRR.4** — **ADR-0042's other clauses are
    preserved.** R3 (do not revisit settled architecture)
    applies. Clause 2 (Make invariants), Clause 3
    (versioning), Clause 4 (deploy-validation) are
    unamended; this study touches only Clause 1's
    image-name row and Consequence 10's OQ-1 closure.
  - **P-AmRR.5** — **Amendment shape: standalone
    superseding ADR per the
    [ADR-0017](../../docs/adr/0017-substrate-posture-amendment.md)
    pattern**, not an in-place Amendment-log subsection
    per
    [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md)
    §Consequence 4. Rationale: the amendment carries
    *architectural-prose impact* (the registry choice
    rationale, the auth posture, the push semantics) not
    just *structured-data row touch*. ADR-0050
    §Consequence 4's convention is for amendments that
    "touch only structured data"; this amendment exceeds
    that scope.
- **Downstream open:** none enumerated. If `/critique`
  surfaces a blocking finding that requires a different
  registry or a different amendment shape, it is
  registered in §Open Questions and the study re-scopes.
- **Critique rounds:**
  round 1 preserved
  ([`studies/critiques/2026-05-30-amendment-engine-image-registry-critique-1.md`](../critiques/2026-05-30-amendment-engine-image-registry-critique-1.md)) —
  0 blocking / 4 important / 4 minor; all dispositioned in
  the Operator Response trailer;
  round 2 preserved
  ([`studies/critiques/2026-05-30-amendment-engine-image-registry-critique-2.md`](../critiques/2026-05-30-amendment-engine-image-registry-critique-2.md)) —
  0 blocking / 1 important / 3 minor; the important
  finding (push-step contract item 4 cited wrong
  property) applied in this revision; two minor applied,
  one accepted-as-is per
  [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)
  §"Skip" grammar.
- **Promotion target:**
  `docs/adr/0054-engine-image-registry-amendment.md` —
  provisionally the next available number (last landed is
  [ADR-0053](../../docs/adr/0053-record-mode-skill.md),
  2026-05-30; reservation is operator-side per
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clause 7, confirmed at `/promote-to-adr` time). The
  amendment ADR will carry `Status: accepted (amends
  ADR-0042)` per the
  [`adr-writing`](../../.claude/skills/adr-writing/SKILL.md)
  A4 idiom.

---

## Context

[ADR-0042](../../docs/adr/0042-release-engineering-invariants.md)
committed the cross-workspace release-engineering contract
on 2026-05-26 in four clauses. Clause 1 §"Docker invariants"
specifies the image name as
`dq-<binary-name>:<tag>` (e.g., `dq-engine:1.2.0`), with
the image-tag derived from the git tag by stripping the
`<workspace>-v` prefix. Consequence 10 §OQ-1 left the
**registry choice** explicitly deferred:

> "OQ-1: Image registry choice. This ADR commits the
> image-name shape (`dq-<binary>:<tag>`) but not the
> registry where images are pushed. Registry choice
> depends on the ADR-0008 host-primitive follow-up (the
> same operational session that substitutes
> `PLACEHOLDER-org/` per ADR-0015 §4). Reserved as a
> follow-up to that session."

Three downstream artifacts already carry that deferral as
a load-bearing TBD:

- [`engine/Dockerfile`](../../engine/Dockerfile) lines
  1–34 + 64–80 — the multi-stage build is complete and
  produces `dq-engine:<tag>` locally per the Makefile's
  `build-engine-image` target. The image is correct, but
  it carries no registry prefix, so it cannot be pushed.
- [`.github/workflows/engine-image.yml`](../../.github/workflows/engine-image.yml)
  lines 1–13 — the workflow runs on every PR + push to
  main, calls `make build-engine-image`, and inspects the
  resulting image's `User` field. The comment at lines
  6–10 names the deferral directly: *"it does NOT push
  the image to any registry because the registry choice
  is still TBD per ADR-0042 OQ-1 … When the registry
  lands, this workflow gains a push step in the same
  operational session that substitutes PLACEHOLDER-org/."*
- [`deploy/base/deployment.yaml`](../../deploy/base/deployment.yaml)
  line 55 pins `image: dq-engine:placeholder` — a string
  that `kubectl apply` cannot resolve to any pullable
  image, with the comment at lines 43–55 naming OQ-1 as
  the gating decision.

This study amends ADR-0042 to resolve OQ-1 and update
Clause 1's image-name row, so the three downstream
artifacts can flip from "TBD-marked structurally complete
state" to "pullable image pinned to a real registry path"
in a follow-on B-row (per P-AmRR.2).

The principles bearing on the decision are **P4** (cost
is a first-class constraint — registry pull rate limits,
storage cost, egress, and CI runner cost surface
explicitly in the option comparison below), **P5**
(evolution must be contract-driven — the amendment ships
under a published ADR shape so future re-platforming
follows the same discipline), and **R3** (do not revisit
settled architecture — Clauses 2, 3, 4 of ADR-0042 stand
unamended; only Clause 1's image-name row and Consequence
10 OQ-1 are touched). **R5** (own the pattern, name the
substrate) is load-bearing for the option comparison —
the four registry substrates are named, but the
*comparison criteria* (cost, auth, OIDC, image-path
shape) are defended on this project's own terms.

---

## Decision Drivers

### D1 — Registry choice: option space

Four registry substrates are in the option space,
enumerated in §Considered Options. The comparison is
framed in this project's specific operational context:
cost surface, auth posture, OIDC/CI integration shape,
and image-path shape. Per R5, the substrates sit inside
the commodity-environment exemption (Docker enumerated;
GitHub / GCP / AWS as platform-environment substrates).

The operator-declared choice (P-AmRR.1) is **Option D**.
The other three options are presented for the
decision-log record and as a documented option-space
starting point if a future session re-opens the choice
(see D6).

### D2 — Image-name pattern: shape change

[ADR-0042](../../docs/adr/0042-release-engineering-invariants.md)
Clause 1 currently commits `dq-<binary-name>:<tag>`. The
amendment changes the shape to
`<registry-host>/<namespace>/dq-<binary-name>:<tag>` — a
registry prefix is added. The local-build flow (the
`make build-engine-image` target producing
`dq-engine:<tag>` locally for `docker compose` and for
single-host operator workflows) is **preserved unamended**
— local images do not need a registry prefix to function.
The amendment commits both the local-shape (unchanged)
and the published-shape (new) so contributors know which
applies in which context.

### D3 — Push-step contract: what the workflow must do

The amendment ADR commits *what* the push step must do
(authenticate, push the built image to the chosen
registry under the chosen image path, surface the pushed
image digest for downstream pinning). The amendment does
**not** commit *how* — the actual workflow edit, secret
plumbing, and login mechanism land in the follow-on
B-row per P-AmRR.2.

### D4 — Preserve ADR-0042's other clauses

R3 binds: Clause 2 (Make invariants), Clause 3
(versioning — `engine-v*` prefix stripped to produce
`<tag>`; tags immutable; pipe character forbidden), and
Clause 4 (deploy-validation) stand unamended. The
amendment touches only Clause 1's image-name row and
Consequence 10 §OQ-1.

### D5 — `deploy/base/deployment.yaml` placeholder flip target

Today's `image: dq-engine:placeholder` is structurally
complete but unpullable. The amendment specifies the
*flip target shape* the deployment manifest must
eventually carry — referencing the published-image shape
committed by D2 — so the follow-on B-row that performs
the flip has a documented destination. The flip itself
(replacing the literal `dq-engine:placeholder` string in
the deployment manifest) is not this amendment's work.

### D6 — Future re-opens have a documented starting point

If a future session re-opens the registry choice (for any
reason — operational signal, cost change, security
posture shift), §Considered Options provides a documented
option-space starting point. The future session may
confirm Docker Hub, shift to one of the other three
substrates already documented, or expand the option space
(a fifth substrate not enumerated today). This study does
not anticipate the triggering signal or pre-commit the
option enumeration as exhaustive across time.

---

## Considered Options

Four registry substrates are in the option space, all
inside R5's commodity-environment exemption: Docker (and
its Docker Hub registry) is enumerated on R5's list;
GitHub Container Registry sits on GitHub, the platform's
git-hosting environment; Google Artifact Registry sits
on GCP, the platform's substrate environment per
[ADR-0010](../../docs/adr/0010-substrate-posture.md);
AWS ECR sits on AWS, a public-cloud equivalent to GCP
under R5's "and equivalents" clause (the platform does
not currently run on AWS, so the substrate is
named-but-out-of-current-scope).

Each option below names the substrate, the image-path
shape, the cost surface framed in this project's
specific operational context, the auth posture, and the
OIDC/CI integration relevant to the
[`.github/workflows/engine-image.yml`](../../.github/workflows/engine-image.yml)
push step that will eventually land in the follow-on
B-row.

### Option A — GitHub Container Registry (`ghcr.io`)

- **Image-path shape:**
  `ghcr.io/fabiocaffarello/dq-engine:<tag>`.
- **Cost surface:** public-image storage and pull is
  free; private-image storage is metered against GitHub
  Packages quotas tied to the GitHub account tier; CI
  builds pull from `ghcr.io` without rate limits when
  authenticated via the runner's `GITHUB_TOKEN`.
- **Auth posture:** the GitHub Actions runner ships with
  a per-job `GITHUB_TOKEN` automatically scoped to the
  repository's `packages: write` permission when
  declared in the workflow's `permissions:` block — no
  static credential rotation, no secret to manage. For
  external pulls (Kubernetes nodes, operator
  workstations), an image-pull secret with a personal
  access token or a GitHub App installation token is
  required.
- **OIDC/CI integration:** the `GITHUB_TOKEN` is the
  shortest auth chain from a GitHub-hosted CI runner to
  a `ghcr.io` push. No third-party identity exchange,
  no Workload Identity Federation hop.
- **Trade-off for this project:** zero ongoing
  credential maintenance from the CI side; tighter
  coupling to the GitHub-hosted CI surface; image pulls
  from non-GitHub Kubernetes nodes need image-pull
  secrets configured per-cluster.

### Option B — Google Artifact Registry

- **Image-path shape:**
  `<region>-docker.pkg.dev/<project>/<repo>/dq-engine:<tag>`
  (e.g.,
  `us-central1-docker.pkg.dev/fabiocaffarello-dq/dq-engine/dq-engine:<tag>`).
- **Cost surface:** storage metered per-GiB per GCP
  pricing; egress metered per-GiB to non-GCP networks
  (intra-region pulls into the same GCP project are
  free); Workload Identity Federation eliminates static
  credential rotation but adds an OIDC exchange hop on
  every pull.
- **Auth posture:** Workload Identity Federation lets a
  GitHub Actions runner exchange its OIDC token for a
  GCP access token bound to a service account with
  Artifact Registry writer permission — no long-lived
  GCP service-account key in CI secrets. For
  Kubernetes-side pulls into a GKE cluster running in
  the same project, the node service account is the
  identity; no image-pull secret is needed.
- **OIDC/CI integration:** the longest auth chain of
  the four options (OIDC token → STS exchange → GCP
  access token → Artifact Registry API), but the chain
  has no static-credential link.
- **Trade-off for this project:** the natural choice
  *if* the platform's eventual Kubernetes substrate is
  GKE (the engine's runtime substrate per ADR-0010 is
  already GCP — BigQuery, GCS, Pub/Sub). The
  platform's *current* deploy target is unspecified
  (`deploy/base/` is tool-neutral per
  [ADR-0019](../../docs/adr/0019-deploy-tooling.md)),
  so this option's benefits are partly latent.

### Option C — AWS Elastic Container Registry (ECR)

- **Image-path shape:**
  `<account>.dkr.ecr.<region>.amazonaws.com/dq-engine:<tag>`.
- **Cost surface:** symmetric to Option B but on AWS
  pricing; storage metered per-GiB; egress metered to
  non-AWS networks; intra-region pulls into the same
  account-region are free.
- **Auth posture:** AWS OIDC Provider lets a GitHub
  Actions runner exchange its OIDC token for STS
  credentials with the ECR push role.
- **OIDC/CI integration:** OIDC chain similar to GCP's,
  scoped to an IAM role in the chosen AWS account.
- **Trade-off for this project:** the platform's
  substrate is **not AWS** (per ADR-0010 the substrates
  are BigQuery + GCS + Pub/Sub, all GCP). ECR would
  bind the registry to a substrate that the platform
  does not otherwise touch — a deliberate cross-cloud
  posture with operational cost (a new AWS account to
  manage, a new IAM surface, a second cloud vendor
  bill). Out-of-substrate without justification.

### Option D — Docker Hub (`docker.io`)

- **Image-path shape:**
  `docker.io/fabiocaffarello/dq-engine:<tag>` (commonly
  shortened to `fabiocaffarello/dq-engine:<tag>` since
  `docker.io` is the Docker daemon's default registry).
- **Cost surface:** public-image storage is free; pull
  rate limits apply per Docker Hub's published policy
  (unauthenticated, free-tier authenticated, and paid-
  tier authenticated have distinct limits). For the
  platform's expected pull volume (one engine pod per
  cluster, restarts measured in days, plus CI rebuilds
  measured in pushes-per-day), the free-tier limit is
  comfortably sufficient without paid tier; if pull
  volume grows (multi-replica deployments, frequent
  pod restarts under autoscaling), the rate limit
  becomes a paid-tier or registry-mirror question.
- **Auth posture:** username + Personal Access Token
  (PAT) via `docker login`. The PAT is a long-lived
  credential stored in CI secrets; rotation is
  operator-side discipline. No native OIDC exchange
  with Docker Hub at this writing.
- **OIDC/CI integration:** no native OIDC. The PAT
  lives in the
  [`.github/workflows/engine-image.yml`](../../.github/workflows/engine-image.yml)
  secret store; the rotation cadence is operator-side
  bookkeeping. This is the only option with a static
  credential surface; the rotation discipline is the
  follow-on B-row's documented contract.
- **Trade-off for this project:** widest distribution
  surface (the engine image is pullable from any
  Kubernetes cluster anywhere without a registry-
  specific image-pull secret if the namespace is
  public); simplest mental model (no OIDC chain, no
  WIF setup, no cross-cloud account); cost of the
  static PAT rotation discipline traded against the
  cost of OIDC setup that the other three options
  carry.

---

## Recommendation

**Option D — Docker Hub (`docker.io/fabiocaffarello`).**

Per **P-AmRR.1**, the operator declared Docker Hub with
the `fabiocaffarello` namespace at session open. The
option comparison above (D1) establishes that all four
options are workable; the trade-off lands on operator-
declared preference, which carries weight here because:

- The platform's pull volume is low (one engine pod per
  cluster; restarts in days, not minutes), so Docker
  Hub's free-tier rate limit is well-sufficient.
- The PAT rotation discipline (Docker Hub's only
  meaningful operational cost vs. the other three
  options) is a routine credential-hygiene item the
  operator already manages for other accounts; adding
  one more to the rotation is small.
- The simplest mental model wins where no other option
  is dominant: a contributor or operator pulling the
  image from any context (`docker pull`, `kubectl`, a
  fresh laptop) does not need to configure an image-
  pull secret if the namespace is public.
- Future re-platforming onto GKE workload (Option B)
  or GitHub Enterprise (Option A) is documented in
  §Considered Options so the option-space comparison
  does not need to be re-derived; this amendment does
  not pre-commit a migration path (P-AmRR.1).

### Amended Clause 1 image-name shape

The amendment ADR commits the new image-name shape in
two parts:

| Context | Image-name shape |
|---|---|
| **Local build** (the `make build-engine-image` target producing an image for `docker compose` and for single-host operator workflows) | `dq-engine:<tag>` — **unchanged from ADR-0042 Clause 1**. The local flow does not need a registry prefix. |
| **Published image** (the image the
  [`.github/workflows/engine-image.yml`](../../.github/workflows/engine-image.yml)
  push step uploads, and the image
  [`deploy/base/deployment.yaml`](../../deploy/base/deployment.yaml)
  references after the placeholder is flipped) | `docker.io/fabiocaffarello/dq-engine:<tag>` (or the shorthand `fabiocaffarello/dq-engine:<tag>`; both resolve to the same image because `docker.io` is the daemon's default registry). |

The `<tag>` derivation rule from ADR-0042 Clause 3 is
**preserved unamended**: `engine-v1.2.0` → `1.2.0` via
prefix-stripping. The `<tag>` value is identical between
the local and published shapes.

### Push-step contract (the follow-on B-row implements)

The amendment ADR commits the push step's behavioral
contract without committing the implementation. The
follow-on B-row must satisfy four behaviors; the *how*
(specific tool invocations, secret names, CI step
wording) is the B-row's decision:

1. **Authenticated push to Docker Hub.** The push step
   authenticates to Docker Hub using a PAT supplied via
   a CI secret. The credential is operator-supplied; the
   B-row picks the secret name and the auth-step
   mechanics.
2. **Image published under the committed path.** The
   locally-built `dq-engine:<tag>` image is published as
   `docker.io/fabiocaffarello/dq-engine:<tag>`. The
   B-row picks whether this is a separate tag + push
   step or a single build-and-push invocation.
3. **Tagged-build-only activation.** The push activates
   only when the workflow runs against a pushed
   `engine-v*` tag, not on every PR or main push.
   Untagged builds remain build-only — the existing
   image inspection step stays. The B-row picks the
   conditional shape.
4. **Pushed digest surfaced for downstream pinning.**
   The pushed image's digest is surfaced in the workflow
   summary so the
   [`deploy/base/deployment.yaml`](../../deploy/base/deployment.yaml)
   flip can pin by digest for the prod overlay. The
   load-bearing property digest pinning preserves is
   **reproducibility**: the deployment manifest pins to
   the exact image bytes the CI lane built, so a
   tag-overwrite at the registry layer cannot silently
   swap the image under the deployment. ADR-0042
   Clause 1's runtime posture (non-root,
   readOnlyRootFilesystem, RuntimeDefault seccomp,
   dropped capabilities) is enforced by the pod's
   `securityContext` independently of the digest and
   remains intact whether or not the deployment pins by
   digest. The B-row picks the summary-emission
   mechanism.

PAT rotation is **operator-side discipline**. The
follow-on B-row documents a recommended cadence and
escalation conditions; the actual rotation events are
operator decisions.

### `deploy/base/deployment.yaml` flip

The amendment ADR commits the **shape** of the image
reference the deployment manifest must eventually carry:

- Replace `image: dq-engine:placeholder` (line 55) with
  `image: docker.io/fabiocaffarello/dq-engine:<tag>`
  where `<tag>` is the published image tag.
- The base manifest itself can land a static tag (e.g.,
  `latest` for staging, a specific version for prod);
  the overlay choice is the follow-on B-row's decision
  per ADR-0042 Clause 4 + ADR-0019's Kustomize-overlay
  framing.

The amendment ADR does **not** flip the placeholder
itself — that flip lives in the follow-on B-row per
P-AmRR.2.

### Consequence 10 OQ-1 closes

[ADR-0042](../../docs/adr/0042-release-engineering-invariants.md)
Consequence 10's OQ-1 ("Image registry choice. … Reserved
as a follow-up to that session.") closes by this
amendment. The amendment ADR's Status line carries
`accepted (amends ADR-0042)` per the
[`adr-writing`](../../.claude/skills/adr-writing/SKILL.md)
A4 idiom; the amendment ADR's Context block names
Clause 1's image-name row + Consequence 10 OQ-1 as the
amended surface.

### What the amendment does NOT change

- **ADR-0042 Clauses 2, 3, 4** stand unamended (D4 +
  P-AmRR.4). Make invariants, versioning rules, and
  deploy-validation posture are preserved.
- **ADR-0042 Clause 1 rows other than the image-name
  row** stand unamended. The Dockerfile-location
  invariant, the multi-stage requirement, the non-root /
  readOnlyRootFilesystem / dropped-capabilities /
  RuntimeDefault-seccomp posture, the `<tag>` derivation
  from `engine-v<semver>`, and the per-workspace
  base-image / build-cache / healthcheck deferral are all
  preserved.
- **ADR-0042 Consequence 10 OQ-2** (release-cadence
  rhythm) stands deferred (P-AmRR.4); this amendment
  closes OQ-1 only.
- **The local-build flow** is unchanged (D2 +
  Recommendation table). `make build-engine-image`
  continues to produce `dq-engine:<tag>` for
  `docker compose` and single-host operator workflows.
- **The
  [`.github/workflows/engine-image.yml`](../../.github/workflows/engine-image.yml)
  workflow file** is unchanged in this study. The
  workflow's build + inspect steps stay; the push step
  lands in the follow-on B-row.
- **The
  [`deploy/base/deployment.yaml`](../../deploy/base/deployment.yaml)
  placeholder** is unchanged in this study. The flip
  to the published shape lands in the follow-on B-row.

---

## Consequences

1. **ADR-0042 Clause 1's image-name row gains a registry
   prefix.** The current row "Image name = `dq-<binary-
   name>:<tag>`" becomes the local-build shape; the
   amendment adds a parallel published-image row
   `<registry-host>/<namespace>/dq-<binary-name>:<tag>`
   with the concrete published shape
   `docker.io/fabiocaffarello/dq-engine:<tag>`. The
   amendment ADR (ADR-0054 — provisional number) carries
   the updated row in its Decision section.

2. **OQ-1 closes; the amendment ADR commits a registry
   choice on the decision log.** Consequence 10 §OQ-1
   transitions from "Reserved as a follow-up" to
   "Resolved by ADR-0054". The decision log's "Last
   updated" entry names the closure.

3. **The image-name shape is in two parts:
   local-build (unchanged) and published-image (new).**
   Contributors and operators know which shape applies
   in which context: local `docker compose` and single-
   host workflows use the unprefixed shape; CI push +
   Kubernetes deployment use the registry-prefixed
   shape. The `<tag>` value is identical across both.

4. **A push-step contract is committed.** The follow-on
   B-row implements (1) `docker login` with PAT from CI
   secret, (2) `docker push` of the locally-built image
   to `docker.io/fabiocaffarello/dq-engine:<tag>`,
   (3) tagged-build-only activation, (4) digest
   surfacing in workflow summary. The contract is
   committed; the wiring is deferred.

5. **The `deploy/base/deployment.yaml` placeholder
   semantics are documented as transitional.** The
   placeholder remains structurally complete and
   unpullable until the follow-on B-row flips it. The
   amendment ADR commits the eventual flip shape
   (`docker.io/fabiocaffarello/dq-engine:<tag>`) so the
   B-row has a target.

6. **A new B2 row registers in the decision log for the
   push-step + manifest-flip wiring** per P-AmRR.2. The
   row's number is reserved operator-side at
   registration time (per
   [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
   Clause 7's reservation discipline analogous to ADR
   numbering — avoids collision with parallel sessions
   that may register B2 rows before this PR merges).
   The new row's expected output is the workflow edit +
   CI secret plumbing + `deploy/base/deployment.yaml`
   placeholder flip. The session shape is
   Implementation-slice per
   [ADR-0052](../../docs/adr/0052-session-reading-router.md)
   §6.2 row 6 landing under closed B-row B2-36 (this
   amendment) — same shape as ADR-0053's follow-on
   record-mode-conventions skill slice.

7. **Future re-platforming has a documented option-
   space starting point.** §Considered Options
   enumerates Options A (GHCR), B (Artifact Registry),
   C (ECR), and D (Docker Hub — chosen). A future
   amendment session that re-opens the choice (e.g.,
   on GKE migration) does not re-derive the trade-off
   math from scratch.

8. **R3 hygiene preserved.** ADR-0042 Clauses 2, 3, 4
   stand unamended (D4 + P-AmRR.4). Only Clause 1's
   image-name row and Consequence 10 OQ-1 are touched.
   Reviewers verify by reading the amendment ADR's
   Context block + the unchanged section list.

9. **R5 hygiene preserved.** §Considered Options names
   four commodity registry substrates (Docker Hub,
   GitHub Container Registry, Google Artifact Registry,
   AWS ECR) as environment, with comparison criteria
   defended on this project's terms (cost, auth, OIDC,
   image-path shape). No external pattern is cited as
   justification; the choice rests on operator
   declaration plus this project's pull-volume and
   credential-hygiene posture.

10. **The amendment ADR ships as a standalone
    superseding-ADR per the
    [ADR-0017](../../docs/adr/0017-substrate-posture-amendment.md)
    pattern, not as an in-place Amendment-log
    subsection per
    [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md)
    §Consequence 4.** Per P-AmRR.5, the amendment
    carries architectural-prose impact (registry
    rationale, auth posture, push semantics) not just
    structured-data row touch. ADR-0050 §Consequence
    4's convention is for amendments that "touch only
    structured data"; this amendment exceeds that
    scope. The amendment ADR's `Status: accepted
    (amends ADR-0042)` line names the relationship.

---

## Open Questions

- **OQ-1 — PAT rotation cadence.** The push-step
  contract uses a Docker Hub PAT (Option D's only
  static credential surface). What rotation cadence is
  operator-policy? Quarterly? On compromise signal
  only? The amendment ADR commits the existence of the
  rotation discipline; the cadence is operational
  governance and surfaces in the follow-on B-row's
  documented contract.
  *Out-of-scope for current cycle:* deferred to the
  follow-on B-row that documents the rotation
  procedure; if cadence is non-trivial, it lands as a
  runbook under `docs/runbooks/`.

- **OQ-2 — Multi-arch image (`linux/amd64` +
  `linux/arm64`).** The engine Dockerfile already
  honors `TARGETARCH` (line 31–33 + 59); the push step
  could push a multi-arch manifest via
  `docker buildx build --push --platform
  linux/amd64,linux/arm64`. Whether the published
  image is multi-arch is a separate decision from
  registry choice; this amendment does not commit it.
  *Out-of-scope for current cycle:* deferred to the
  follow-on B-row; the choice depends on whether
  arm64 Kubernetes nodes are in scope for the
  platform's deployment surface.

- **OQ-3 — Image signing (cosign / Sigstore).**
  Whether the published image is signed at push time is
  a separate amendment surface. The amendment ADR
  commits unsigned-push as the baseline; future
  signing is a follow-up amendment session.
  *Out-of-scope for current cycle:* trigger is a
  concrete supply-chain signal (a downstream consumer
  requiring signed images, a security review surfacing
  the gap).

- **OQ-4 — Public vs. private repository on Docker
  Hub.** Public namespace with image-pull-secret-less
  pulls is the simplest configuration; private
  namespace forces image-pull-secret distribution to
  every consuming Kubernetes cluster. The amendment
  ADR does not pre-commit; the operator chooses at
  the time of the follow-on B-row's PAT setup.
  **Default assumption: public** (the operator
  pre-declared the `fabiocaffarello` namespace without
  specifying visibility; public is the lighter-weight
  default that requires no image-pull-secret
  distribution and is the natural fit for an
  open-source-style namespace). If the operator
  declares private at B-row time, the follow-on B-row
  also lands the image-pull-secret distribution
  discipline.
  *Out-of-scope for current cycle:* deferred to the
  follow-on B-row.

---

## Promotion target

`docs/adr/0054-engine-image-registry-amendment.md`
(provisional; operator-side reservation confirmed at
[`/promote-to-adr`](../../.claude/commands/promote-to-adr.md)
time per
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
Clause 7). The amendment ADR carries
`Status: accepted (amends ADR-0042)` per the
[`adr-writing`](../../.claude/skills/adr-writing/SKILL.md)
A4 idiom; its Decision section commits the amended Clause
1 image-name row + the closed OQ-1; its Context names the
ADR-0017 standalone-superseder shape rationale.
