<!-- path: studies/foundation/README.md -->

# Foundation

## Purpose

This directory holds the foundational documents for the DQ Platform.
Together, they answer six questions in a specific order:

1. **What is this project, who is it for, and under what principles?**
2. **How is the monorepo organized internally?**
3. **What is the logical contract between the engine and the rules?**
4. **What is the technical architecture, end to end?**
5. **How does the system behave under load, failure, and time?**
6. **Which decisions are still open, and which are resolved?**

These documents are the canonical source of truth for the project. They
are the basis for every ADR, every CI rule, every code review pushback,
every onboarding conversation.

## Status

Draft. The foundation cycle is in progress as part of Wave 1 of the
project. Documents will be revised as B0 decisions are resolved in
`studies/decisions/`.

## Reading order

Read in numerical order. Each document assumes the previous ones.

1. [`01-charter-and-principles.md`](./01-charter-and-principles.md) —
   mission, audiences, principles, workspace identities.
2. [`02-monorepo-topology.md`](./02-monorepo-topology.md) —
   directory structure, workspaces, ownership boundaries, CI strategy,
   release model.
3. [`03-boundary-contract.md`](./03-boundary-contract.md) —
   the logical contract between `engine/` and `rules/` (schema
   versioning, linter distribution, manifest format, compatibility
   windows).
4. [`04-system-architecture.md`](./04-system-architecture.md) —
   layers, flows, internal patterns (loader, scheduler, compilers,
   runner, reporting, alerting, testing, environment, logging).
5. [`05-operational-discipline.md`](./05-operational-discipline.md) —
   cost control, run identity, idempotency, failure scope, retry
   semantics, observability, evidence retention.
6. [`06-decision-log.md`](./06-decision-log.md) —
   living register of open and resolved platform decisions.

## What this directory is not

- **Not a runtime artifact.** Nothing here is loaded by the engine at
  runtime. Nothing here is published. These are reasoning artifacts.
- **Not the published documentation.** When a foundation conclusion is
  stable enough to guide implementation, it is **rewritten** as an ADR
  under `docs/adr/` or as a contributor doc under the appropriate
  workspace. The foundation document stays as historical reasoning.
- **Not a sandbox for code.** No Go, no YAML, no Dockerfile. The
  foundation phase is for thinking, not building.

## Promotion discipline

When a foundation document becomes stable enough to guide
implementation:

1. Promote enduring platform decisions into ADRs under `docs/adr/`.
2. Promote domain-authoring guidance into `rules/docs/` (during
   Wave 3).
3. Promote operational guidance into `docs/operations/` (during
   Wave 3).

Do **not** link the published artifacts back into `studies/`. The
foundation is scaffolding; the published artifacts are the building.
