<!-- path: studies/foundation/06-decision-log.md -->

# 06 — Decision Log

## Metadata

- Purpose: track every platform decision that must be resolved before
  implementation. Each row notes priority, current status, the
  rationale for why it matters, and the location of the resolving
  document.
- Audience: project lead, platform engineers, anyone planning the next
  session of work.
- Status: living document. Update whenever a decision changes state.
- Last updated: 2026-05-20
- Promotion target: this document stays in `studies/foundation/` for
  the project's lifetime. Resolved decisions are promoted to ADRs
  under `docs/adr/` during Wave 3; rows here keep the link.

---

## Prioritization Model

- **B0** — blocking. Must be resolved before Wave 3 (scaffolding) can
  start. Each B0 gets its own dated decision document in
  `studies/decisions/`.
- **B1** — important. Should be resolved before serious implementation
  starts. Some are refinements of B0 outcomes.
- **B2** — later. Can be resolved as implementation reveals concrete
  needs. Documented here so they are not forgotten.

## Status Vocabulary

- **open** — no work has started.
- **in-progress** — a draft exists in `studies/decisions/` but is not
  yet finalized.
- **resolved-study** — a complete study exists in
  `studies/decisions/`; ready for promotion when Wave 3 starts.
- **resolved-adr** — the study has been promoted to an ADR under
  `docs/adr/`.

---

## B0 — Blocking Decisions (Wave 1 Scope)

| # | Topic | Status | Key Question | Why It Matters | Expected Output |
|---|---|---|---|---|---|
| B0-1 | Engine ↔ rules compatibility | [resolved-study](../decisions/2026-05-20-engine-rules-compatibility.md) | How does the rules workspace declare which schema and linter contract it follows, given both live in the same repository? | Without an explicit contract, the monorepo lets boundaries erode silently. | ADR + boundary contract refinement |
| B0-2 | Run identity and idempotency | open | What uniquely defines a run, and how do reruns of the same window behave? | Reporting trust and alert deduplication depend on it. | Execution semantics ADR |
| B0-3 | Result write model | open | Are `dq_executions` and `dq_check_results` append-only, upserted, or hybrid? | Impacts retries, lineage, and dashboard accuracy. | Storage design ADR |
| B0-4 | Failure scope | open | When one check errors operationally, does the entity error, degrade, or partially complete? | Incidents and alerting become inconsistent without a single policy. | Failure-semantics ADR plus runbook |
| B0-5 | Manifest publication semantics | open | What guarantees atomic, reversible ruleset publication to object storage? | Runtime safety depends on manifest discipline. | Control-plane contract ADR |
| B0-6 | Alert routing contract | open | What fields live in `_owners.yaml`, what stays in engine config, and what is deduplicated on the data itself? | Prevents alerting logic from becoming hardcoded chaos. | Governance doc + owners schema |
| B0-7 | Loader / scheduler / retry failure semantics | open | What exactly causes ruleset load failure, scheduler reconciliation failure, and retry budget exhaustion? | The fail-fast registry pattern only helps if failure semantics are explicit and consistent. | Loader and scheduler ADRs |

---

## B1 — Important Decisions (Pre-Implementation Scope)

| # | Topic | Status | Key Question | Why It Matters | Expected Output |
|---|---|---|---|---|---|
| B1-1 | Baseline strategy | open | Where do moving averages and historical references come from, and what happens with sparse history? | Volume and freshness checks depend on consistent history semantics. | Check design note |
| B1-2 | BigQuery cost ceilings | open | What are the per-environment limits for window size, concurrency, failed samples, and dry-run enforcement? | Cost drift is predictable; designing around it is cheap. | Operations doc + defaults policy |
| B1-3 | Scheduler catchup behavior | open | How are catchup, missed windows, and manual triggers represented? | A scheduler without precise semantics causes duplicate or missing evaluations. | Scheduling design note |
| B1-4 | Environment configuration model | open | Which configuration lives in code, deployment, or data, and how are `local`, `qa`, and `prod` isolated? | Prevents configuration sprawl and implicit behavior drift. | Env strategy ADR |
| B1-5 | Local testing strategy | open | What can be tested offline, what needs BigQuery sandbox access, and how is generated SQL inspected? | Developer experience shapes long-term quality. | Dev guide + tooling scope |
| B1-6 | Evidence retention parameters | open | How many violating samples per check, for how long, under what privacy constraints? | Storage cost and privacy compliance depend on it. | Storage and security note |
| B1-7 | Compatibility window duration | open | How long is each schema version supported after its successor is released? | Migration ergonomics for domain teams. | Boundary contract refinement |
| B1-8 | Manifest cryptographic posture | open | Does the manifest carry signatures beyond checksums? Who signs it? | Defense in depth against tampering or accidental publication. | Security note |
| B1-9 | CODEOWNERS finalization | open | Final team names and path rules for the asymmetric review model. | Enforces the boundary at PR-review time. | CODEOWNERS file |
| B1-10 | Workspace tooling stack | open | Confirm Go workspaces (`go.work`) as the tooling choice and finalize the per-tool module structure. | Affects every CI pipeline file. | Topology ADR refinement |

---

## B2 — Later Decisions (Implementation-Phase Scope)

| # | Topic | Status | Key Question | Why It Matters | Expected Output |
|---|---|---|---|---|---|
| B2-1 | External artifact references in DSL | open | Will the DSL ever allow helper files (auxiliary SQL, reference payloads) resolved relative to the rule origin? | The capability is useful but dangerous if it becomes an escape hatch. | DSL evolution ADR |
| B2-2 | Logging contract specifics | open | Which package names and override syntax are officially supported by `DQ_LOG_LEVELS`? | Good observability patterns help only if standardized. | Operations doc |
| B2-3 | Release engineering invariants | open | Which Docker, Make, and versioning invariants are mandatory across all workspaces? | Long-term repo health depends on consistent ergonomics. | Release engineering doc |
| B2-4 | Stream reporting continuity | open | How will stream-runner results align with batch result tables and identifiers? | Future migration should not fracture observability. | Stream evolution design note |
| B2-5 | Entity onboarding workflow | open | What exact checklist determines when a new entity is ready for test channel and later for production alerting? | Governance quality depends on repeatable onboarding. | Runbook + checklist |
| B2-6 | Dashboard contract | open | Which metrics and dimensions are guaranteed for downstream consumers (Looker, Grafana, etc.)? | Avoids each consumer inventing its own interpretation. | Reporting contract |
| B2-7 | Documentation site generator | open | Does `docs/` get a static site generator, or stay as raw markdown? | Affects how documentation is discovered by non-developers. | Documentation infrastructure note |
| B2-8 | Infrastructure tooling | open | Kustomize, Helm, Terraform, or a combination for `deploy/`? | Affects deployment ergonomics and environment isolation. | Infrastructure ADR |

---

## Wave 2 — Platform Decisions (Single Consolidated Document)

These are not in the priority backlog because they are a separate
wave with a single decision document covering all of them. They are
listed here so the log is complete.

| # | Topic | Status |
|---|---|---|
| W2-1 | Git host choice (affects CI artifact location and syntax) | open |
| W2-2 | Multi-agent contract — finalize `.claude/`, `.codex/`, `AGENTS.md` | open |
| W2-3 | Docker Compose local scope — which services emulated, which sandboxed | open |
| W2-4 | Documentation language policy (English / Portuguese / mixed) | open |
| W2-5 | Per-workspace tag prefix conventions (confirm or revise) | open |

---

## Process

### When a decision moves from `open` to `in-progress`

A draft document exists in `studies/decisions/` (typically created by
the `/resolve-b0` command for B0 items, or by ad-hoc study for B1/B2).
The row is updated with a link to the draft.

### When a decision moves to `resolved-study`

The study is complete: its Open Questions section is either empty or
contains only items explicitly accepted as out-of-scope for the
current cycle. The row is updated with the final study path.

### When a decision moves to `resolved-adr`

The study has been rewritten as an ADR under `docs/adr/` during Wave
3. The row keeps the original study link **and** adds the ADR link.
The study stays in `studies/decisions/` as historical reasoning; it
is not deleted.

### When a new decision is discovered

If a working session reveals a decision that is not yet in the log,
the session adds it before resolving it. Decisions found mid-session
should be tracked here even if not immediately worked on. This keeps
the log honest.

---

## Wave Gates

Use this section to confirm whether the project can advance.

### Wave 1 gate (B0 complete)

Pass when **every B0 row** is at status `resolved-study` or
`resolved-adr`. Currently: 1 of 7 resolved.

### Wave 2 gate (platform decisions complete)

Pass when the consolidated Wave 2 decisions document exists in
`studies/decisions/` and addresses every W2 row.

### Wave 3 readiness

Pass when both Wave 1 gate and Wave 2 gate have passed. Wave 3 (full
scaffolding) cannot start before this.

---

## Recommended Next Sequence

1. Resolve B0-1 (compatibility model) — it underpins B0-5 and B0-7.
2. Resolve B0-5 (manifest semantics) — required before loader
   semantics can be specified.
3. Resolve B0-2 (run identity) — required before result write model.
4. Resolve B0-3 (result write model) — depends on B0-2.
5. Resolve B0-4 (failure scope) — depends on B0-2 and B0-3.
6. Resolve B0-7 (loader and scheduler failures) — depends on B0-1,
   B0-5, B0-4.
7. Resolve B0-6 (alert routing) — depends on B0-4.
8. Run Wave 2 consolidated decision session.
9. Begin Wave 3 scaffolding.

This ordering minimizes rework: each decision builds on stable ground
from the previous.
