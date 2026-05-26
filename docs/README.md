<!-- path: docs/README.md -->

# `docs/` — Documentation Workspace

The docs workspace holds cross-workspace documentation that
is part of the published repository product (per
`CLAUDE.md` R8):

- **[`adr/`](adr/)** — Architecture Decision Records
  (MADR-aligned). ADRs `0001–0014` cover Wave 1, Wave 2,
  the Wave 3 sequencing, and the HTTP trigger handler
  contract.
- **[`glossary.md`](glossary.md)** — canonical terminology
  for terms with codebase-specific meaning (lands in
  W3-P8a).
- **[`governance.md`](governance.md)** — review model, the
  three review groups, contribution-time flows (lands in
  W3-P8b).
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) at the repository
  root — the four practical contributor flows (add a rule, run
  `make demo-p6`, open a B-item, close a Wave 3 session loop).
  Published at the root per GitHub convention (lands in
  W3-P8c).
- **[`runbooks/`](runbooks/)** — operator-facing playbooks for
  the four common incident classes: manifest rollback,
  orphan-run remediation, alert-dedup debugging, refresh-
  failure escalation (lands in W3-P8d).
- **[`security/`](security/)** — operator-facing security
  notes carrying threat models, posture commitments, and
  trigger conditions for revisiting cryptographic/posture
  decisions. First entry lands with ADR-0030
  (manifest cryptographic posture).
- **[`dev/`](dev/)** — contributor-facing developer
  guides. First entry is the local-testing guide that
  lands with ADR-0034.

## Current state (post-Wave-3)

This directory holds:

- `adr/0001–0007` — Wave 1 commitments (B0).
- `adr/0008–0012` — Wave 2 commitments (W2).
- `adr/0013` — the Wave 3 phase-sequencing ADR.
- `adr/0014` — the HTTP trigger handler contract (W3-P4e).
- `adr/0015–0019` — Wave 3 sub-decisions promoted at gate
  closure (CODEOWNERS, workspace tooling, substrate-posture
  amendment, env config, infrastructure tooling).
- `adr/0020–0028` — Wave-S commitments
  (record-oriented capability).
- `adr/0029` — set-mode BigQuery cost ceilings (resolves B1-2).
- `adr/0030` — manifest cryptographic posture (resolves B1-8;
  deferral with auditable trigger conditions).
- `adr/0031` — evidence retention parameters (resolves B1-6;
  single-tier partition-expiration retention + sample-content
  allowlist).
- `adr/0032` — baseline strategy (resolves B1-1;
  platform-history + static baselines design; design-only,
  implementation deferred to first baselined kind's slice).
- `adr/0033` — scheduler catchup behavior (resolves B1-3;
  external-scheduler contract + advisory `schedule` field
  + per-env catchup horizon; design-only, implementation
  deferred to first scheduler-consumer slice).
- `adr/0034` — local testing strategy (resolves B1-5;
  six-tier test-type taxonomy + build-tag posture +
  fixture-tree convention + tooling scope inventory;
  documentation-only ADR + new dev guide at
  `docs/dev/local-testing.md`).
- `adr/0035` — compatibility window duration (resolves
  B1-7; N-1 + 90-day calendar-time floor for schema
  versions; engine-binary-bound drop mechanism; closes
  the B1 backlog).
- `adr/0036` — `dq-manifest set-pointer` rollback subcommand
  (resolves B2-10; first-class CLI surface for the
  CAS-conditional pointer rewrite primitive; closes the
  bypass-via-gsutil failure mode; runbook §3 rewritten
  around the new primary path).
- `adr/0037` — `_owners.yaml` owner ↔ CODEOWNERS-group
  linter cross-check (resolves B2-9; new `-codeowners` flag
  on `dq-lint` extends the existing validation-error exit
  code with a group-membership check; closes the timing
  gap where a stale or typo'd `owner:` previously only
  failed at PR-review time).
- `glossary.md` — codebase-specific terminology (W3-P8a).
- `governance.md` — review model and contribution-time
  flows (W3-P8b).
- `runbooks/` — operator-facing playbooks (W3-P8d).
- `security/` — operator-facing security notes
  (introduced 2026-05-25 with ADR-0030;
  evidence-retention note added with ADR-0031).

The contributor-facing guide lives at the repository root as
[`../CONTRIBUTING.md`](../CONTRIBUTING.md) per GitHub convention
(W3-P8c).

## Reading conventions

- All technical documents in this workspace are in
  **English** per
  [ADR-0011](adr/0011-documentation-language.md).
- ADRs are forward-only: they do not link backwards into
  `studies/` (per `CLAUDE.md` R8). The studies that
  originated each ADR remain in
  [`../studies/decisions/`](../studies/decisions/) for
  historical reasoning.
