<!-- path: README.md -->

# DQ Platform

The DQ Platform is a Data Quality engine for curated data
assets, designed for visibility, ownership, and operational
actionability of data quality posture. The platform is
organized as a single monorepo with five logical workspaces:

- **[`engine/`](engine/)** — Go runtime, DSL schema source
  of truth, compilers, scheduler integration, reporting,
  alerting.
- **[`rules/`](rules/)** — declarative YAML rule
  specifications by entity, owner metadata, governance
  workflow.
- **[`tools/`](tools/)** — auxiliary CLIs (linter, dry-run
  runner, manifest publisher).
- **[`deploy/`](deploy/)** — Kubernetes manifests,
  infrastructure configuration, environment definitions.
- **[`docs/`](docs/)** — cross-workspace documentation,
  ADRs under [`docs/adr/`](docs/adr/), glossary,
  governance.

## Documents

- **[`CLAUDE.md`](CLAUDE.md)** — the canonical operating
  contract for AI coding agents in this repository (hard
  rules R1–R8, platform principles P1–P6, slash-command
  catalog).
- **[`AGENTS.md`](AGENTS.md)** — cross-agent convention
  entry point; thin pointer to `CLAUDE.md`.
- **[`docs/adr/`](docs/adr/)** — Architecture Decision
  Records (MADR-aligned). ADRs `0001–0013` cover Wave 1
  (compatibility, identity, storage, failure scope,
  manifest publication, alerting, loader semantics) and
  Wave 2 (Git host, multi-agent contract, substrate
  posture, documentation language, tag conventions) plus
  the Wave 3 phase sequencing.
- **[`studies/foundation/06-decision-log.md`](studies/foundation/06-decision-log.md)**
  — current state of every decision (open, in-progress,
  resolved-study, resolved-adr) plus the wave gates.
- **[`KICKOFF.md`](KICKOFF.md)** — human-operator session
  guide.

## Substrate posture

The local development substrate is governed by
[ADR-0010](docs/adr/0010-substrate-posture.md). The local
`docker-compose.yml` brings up the substrate capabilities
that satisfy the ADR-0010 matrix:

- Pub/Sub publish/subscribe.
- Object store with generation-conditional pointer writes
  and content-addressed bodies (sha256).
- Tabular store with append-only writes.

Production-shape identity (OIDC) and full tabular-store
lazy-view fidelity require sandbox cloud access; see
ADR-0010 for the full matrix.

## Run locally

```sh
make up          # bring up the local docker-compose substrate
make test        # run substrate smoke tests + go tests
make lint        # lint the workspace
make down        # tear down the local docker-compose substrate
make help        # list every Makefile target
```

## Wave status

- **Wave 1** (seven `B0` blocking decisions) — closed.
  ADRs `0001–0007`.
- **Wave 2** (five `W2` platform decisions) — closed.
  ADRs `0008–0012`.
- **Wave 3** — Phase 0 (protocol) and Phase 1 (ADR
  promotion) closed; Phase 2 (root infrastructure) closes
  with this commit. Phases 3–8 remain open; see
  [ADR-0013](docs/adr/0013-wave3-sequencing.md) for the
  phase structure.

## Language

Technical artifacts (ADRs, schemas, READMEs, code comments,
contract documents) are in **English** per
[ADR-0011](docs/adr/0011-documentation-language.md).
Internal onboarding guides may be written in Portuguese
when the file opens with a one-line language marker.
