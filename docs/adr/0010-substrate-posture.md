<!-- path: docs/adr/0010-substrate-posture.md -->

# ADR-0010 — Substrate Posture (Local Compose Scope)

- **Status:** accepted; **amended in part by ADR-0017** (object-store CAS row revised from **Yes** to **Partial**; substrate-selection checkpoint gains a CAS-fidelity sub-criterion); **amended in part by [ADR-0028](./0028-kafka-substrate-row.md)** (capability matrix extended with three event-stream rows for record-mode runtime per ADR-0024; substrate-selection checkpoint extended to Kafka)
- **Date:** 2026-05-21

**Scope note (added 2026-05-23):** This ADR's set-oriented capability rows apply to BigQuery-backed evaluation. Record-oriented event-stream capability is committed in [ADR-0028](./0028-kafka-substrate-row.md).

---

## Context

The platform's runtime depends on several substrate
capabilities — a tabular store (BigQuery in deployed
environments), an object store (GCS in deployed environments)
with generation-conditional writes and content-addressed
paths, a publish-subscribe surface (Pub/Sub in deployed
environments), service-identity (OIDC), and a host-side
container/artifact registry. Each capability is named by an
earlier ADR:

- Tabular store with append-only writes and a lazy view that
  uses `ROW_NUMBER() OVER (PARTITION BY ... ORDER BY ...)`
  — ADR-0003.
- Object store with `manifests/by-hash/`, `yamls/by-hash/`,
  sha256 throughout, and a CAS-protected pointer — ADR-0005.
- Pub/Sub publish/subscribe of structured event payloads —
  ADR-0006.
- Loader refresh cadence, hash short-circuit, orphan-run
  detection, structured logs / metrics endpoint — ADR-0007.
- Byte-equality CI gate and unforgeable linter pin —
  ADR-0001, with the host-side mechanism from ADR-0008.

Contributors must exercise the platform without each session
requiring sandbox cloud access for routine work. At the same
time, some capabilities cannot be emulated with the fidelity
needed to validate production-shape behavior. This ADR
commits the substrate posture **in capability terms** —
which capabilities the local Docker Compose environment must
satisfy, and which require a sandbox cloud account — without
naming specific emulator artifacts or cloud projects (those
are scaffolding details).

---

## Decision

The platform adopts a **hybrid substrate posture**: a local
Docker Compose environment satisfies a defined set of
capabilities, and a sandbox cloud account satisfies the rest.
The cloud target is **not decided globally** in this ADR; the
ADR commits a capability matrix that any compliant substrate
must satisfy.

### Capability matrix

| Capability | Local emulation | Notes |
|---|---|---|
| Pub/Sub publish/subscribe | **Yes** | The ADR-0006 event payload must be exercisable end-to-end without sandbox. |
| Object store: generation-conditional pointer write | **Yes** | The ADR-0005 compare-and-swap on `manifests/latest.json` must be testable locally. |
| Object store: `by-hash/` immutability with sha256 | **Yes** | ADR-0005 layout: `manifests/by-hash/sha256-<hex>.json`, `yamls/by-hash/sha256-<hex>.yaml`. |
| Tabular store: append-only writes | **Yes** | The ADR-0003 append-only invariant must be verifiable locally. |
| Tabular store: lazy view with `ROW_NUMBER() OVER (PARTITION BY … ORDER BY …)` | **Partial** | Local emulator fidelity gap is known; sandbox-recommended for full end-to-end validation of the canonical-view semantics from ADR-0003. |
| OIDC / service identity for cross-substrate auth | **No (sandbox required)** | Production-shape identity flows cannot be emulated with fidelity. Local Compose may stand up a permissive development stub for local-only auth, but the stub does not satisfy production identity semantics and is never the path exercised in the sandbox lane. |
| Unforgeable linter image pin (ADR-0001) | **Partial** | Local digest pinning works for development; production unforgeability lives in the host's registry primitives — see ADR-0008. |
| Structured logs / metrics endpoint | **Yes** | The engine exposes a metrics endpoint locally; the ADR-0007 observability emission (log + metric + span per failure path) is reachable. Collector wiring is a scaffolding detail. |
| Orphan-run detection polling | **Yes** | The ADR-0007 periodic scan is engine-side logic; only requires the tabular store, which is **Yes** above. |
| Cost-ceiling enforcement (`status = aborted`) | **Yes** | Engine-side budget logic from ADR-0007; can be exercised against either local or sandbox tabular store. |

### Wave-3 scaffolding contract

The local Compose environment must, by the close of Phase 2:

- bring up every **Yes** capability via a single
  bootstrapping command;
- run a smoke test against each **Yes** capability;
- pass a lint pass.

Specific tooling choices (which Pub/Sub emulator image,
which object-store emulator, which tabular-store backend,
which secrets-management approach for the OIDC stub) are
scaffolding decisions; they can change without re-opening
this ADR so long as they preserve the capability rows.

The **No** and **Partial** rows are documented as
**sandbox-required** in a contributor onboarding artifact.

### CI lanes

Integration tests split into two lanes:

- **`local-runnable`** — runs against the local Compose
  environment; covers every **Yes** capability.
- **`sandbox-required`** — runs against the sandbox cloud
  account; covers the **No** and **Partial** rows.

The split lives in test labels or build tags, not in
directory layout.

---

## Consequences

1. **The capability matrix is the substrate posture
   contract.** Future emulator changes update this matrix;
   workspace-level documents do not redefine substrate
   expectations.

2. **Sandbox cloud access is a documented contributor
   onboarding artifact.** Contributors needing the **No**
   or **Partial** rows obtain sandbox access via that
   artifact. Routine contributor flows do not require
   sandbox access.

3. **The end-to-end "manifest publish → loader hash-short-
   circuit refresh → execution write → operational alert
   publish" flow runs locally without sandbox.** Every
   capability that flow depends on (object store
   generation-conditional + by-hash; tabular store
   append-only; Pub/Sub; structured logs) is in the **Yes**
   set. The "OIDC-bound publish" variant requires sandbox.

4. **The tabular-store lazy-view fidelity gap is not a
   blocker for local development.** Engine logic that
   depends on view semantics must be exercisable in unit
   tests against an abstraction layer; full-view fidelity
   is verified in the sandbox lane.

5. **Local digest pinning of the linter image is acceptable
   for development.** Production unforgeability lives in
   the host's registry primitives — see ADR-0008.

6. **Substrate selection is checkpointed against this
   matrix.** Any future decision to select or change the
   platform's object-storage, tabular-store, or
   publish-subscribe substrate must verify the substrate
   provides the capabilities in this matrix. A substrate
   missing generation-conditional writes, content-addressed
   paths with immutability, or any **Yes** capability
   reopens this ADR.

7. **Tests labelled `sandbox-required` cannot run in the
   `local-runnable` lane.** Build configuration prevents
   misrouting; sandbox-required tests gate on the presence
   of sandbox credentials.

---

## Notes

- Specific emulator image and version choices are a
  scaffolding detail. The matrix commits the capability,
  not the artifact.
- A one-command sandbox bootstrap (e.g., a make target) is
  a scaffolding onboarding follow-up.
- The mechanism for documenting the tabular-store-view
  fidelity gap to contributors writing storage-touching
  code is a scaffolding contributor-doc follow-up.
- Whether the structured-logs pipeline needs a local
  collector image in Compose or only a metrics endpoint
  that contributors curl directly is a scaffolding
  observability sub-decision; both options satisfy the
  capability row.
- Whether the operator-issued abort path from ADR-0007
  needs a local admin-API mock or is exercised only in the
  sandbox lane is a scaffolding admin-API design item.
