<!-- path: deploy/README.md -->

# `deploy/` — Infrastructure Configuration

The deploy workspace holds Kubernetes manifests,
environment overlays (`local`, `qa`, `prod`), and
infrastructure-as-code files for the DQ Platform.

Per
[B1-10's resolution](../studies/decisions/2026-05-21-b1-10-workspace-tooling.md),
this workspace is **not a Go module by default**. If a
future Go-based infrastructure tool is introduced (e.g., a
manifest generator), its module is added to
[`go.work`](../go.work) at that time.

## Scope (Wave 3)

Phase 7 scaffolds this workspace: Kubernetes manifests for
the engine, environment overlays for `local`, `qa`, `prod`.
Phase 7 has a dependency on the environment-configuration-
model decision (B1-4 in the decision log); the dependency
is resolved in a separate B1 study session before Phase 7
proceeds.

## Current state (Phase 2)

This directory exists for the empty-layout commitment from
[ADR-0013](../docs/adr/0013-wave3-sequencing.md). No
deployment manifests yet.
