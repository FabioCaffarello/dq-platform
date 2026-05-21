<!-- path: docs/adr/0013-wave3-sequencing.md -->

# ADR-0013 — Wave 3 Scaffolding Sequencing

- **Status:** accepted
- **Date:** 2026-05-21

---

## Context

Wave 1 (seven `B0` decisions) and Wave 2 (five `W2`
platform decisions) are closed. Their commitments are
recorded in ADRs 0001–0012:

- **ADR-0001** — engine ↔ rules compatibility.
- **ADR-0002** — run identity and idempotency.
- **ADR-0003** — result write model.
- **ADR-0004** — failure scope.
- **ADR-0005** — manifest publication semantics.
- **ADR-0006** — alert routing contract.
- **ADR-0007** — loader / scheduler / retry failure
  semantics.
- **ADR-0008** — Git host.
- **ADR-0009** — multi-agent contract.
- **ADR-0010** — substrate posture (local Compose scope).
- **ADR-0011** — documentation language.
- **ADR-0012** — per-workspace tag conventions.

Wave 3 — scaffolding the five product workspaces (`engine/`,
`rules/`, `tools/`, `deploy/`, `docs/`) plus the root
infrastructure — must turn these commitments into running
code, configuration, and CI.

Wave 3 is larger and more entangled than Waves 1 and 2.
Without an explicit phase ordering, sessions can scaffold a
downstream surface before its upstream exists (e.g., a
schema mirror before its source), collide on shared files,
build on capability rows that have not been wired up, or
cite ADRs that have not been promoted — leaving brittle
forward references.

This ADR commits the **phase structure** that prevents
those failure modes. It does not pick concrete artifacts
(emulator images, exact Go module layout, CI runner version
pins); those are sub-decisions for the sessions inside each
phase.

---

## Decision

Wave 3 proceeds in **nine phases**, numbered 0 through 8.
Phase boundaries are explicit; inside a phase, scaffolding
units can move in any order so long as they do not cross
phase lines. Every Wave 3 session runs under
`.claude/playbooks/wave-3-session-loop.md` and self-verifies
against `.claude/playbooks/wave-3-acceptance-criteria.md`.

### Why this ordering

Seven drivers shape the order:

1. **Dependency order.** No scaffolding unit may cite an
   upstream that has not been scaffolded. The byte-equality
   CI gate from ADR-0001 is the canonical example — it
   requires both the engine schema source and the rules
   schema mirror to exist before it can be wired up.
2. **Loop discipline.** Each session is one unit; each unit
   sits inside one phase. Crossing phase boundaries
   mid-session expands scope and erodes the quality bar
   that the playbooks enforce.
3. **CI gates land last among schema layers.** Wiring a
   mandatory gate before its inputs exist creates a noisy
   red-on-main that trains contributors to ignore the
   gate.
4. **Capability-matrix coverage.** Each **Yes** row in the
   ADR-0010 capability matrix is exercisable by a local
   test or runnable command by the end of Wave 3. Phases
   that produce those tests come after the phase that
   wires up the capability.
5. **ADR-promotion-before-citation.** Scaffolding sessions
   that cite an ADR run after the ADR exists. Phase 1
   produces ADRs 0001–0013; Phase 2 onward cites them.
6. **R5 hygiene at scaffold scale.** Code is harder to
   police than documents — variable names, package paths,
   comment references, external-vendor SDK imports all
   carry the risk of naming sibling-team or prior-art
   systems. The sequencing surfaces this discipline early
   (Phase 2 root infrastructure) so it takes root before
   larger scaffolds land.
7. **Cost-as-first-class (platform principle P4).** Phases
   that need substrate (Phase 4 onward) consume the local
   Compose substrate stood up in Phase 2. Reversing the
   order would force contributor onboarding through cloud
   sandbox access for every routine flow — unacceptable
   per P4. Phase 2 must therefore land before any phase
   that exercises substrate-bound code.

### Phases

**Phase 0 — Protocol** *(closed; commit `25e06ab`,
2026-05-21).*

Sequencing ADR (this document), Wave 3 session-loop
playbook, Wave 3 acceptance-criteria playbook. Bootstrap of
the loop discipline.

**Phase 1 — ADR promotion** *(closes with the commit that
lands this ADR).*

Twelve resolved studies promoted to `docs/adr/0001–0012`,
plus this sequencing ADR at `docs/adr/0013`. Every B0 row
and every W2 row in the decision log carries a
`resolved-adr` link.

**Phase 2 — Root infrastructure.**

`go.work`, `Makefile`, `docker-compose.yml`, `.github/`
(CI lanes per ADR-0008; branch protection that makes the
byte-equality gate from ADR-0001 non-bypassable),
`.codex/AGENTS.md` (pointer file per ADR-0009),
top-level `README.md`, and the empty top-level layout of
each of the five workspaces.

Phase 2 wires up the ADR-0010 capability matrix locally:
the **Yes** rows must each be reachable from `make up` or
equivalent. The **No / Partial** rows are documented as
sandbox-required.

**Closes when:** `make up` brings up every **Yes**
capability; `make test` smokes them; `make lint` runs.
Phase 2 is the first phase that exercises production
code (R1 ceased applying when the Wave 2 gate closed);
R3–R8 still apply throughout.

**Phase 3 — Schema-layer scaffold.**

`engine/internal/dsl/schema/v1.schema.json` (canonical
schema source); `rules/_schema/v1.schema.json` (byte-equal
mirror); `tools/lint/` (minimum linter binary enforcing
the byte-equality CI gate per ADR-0001 and rejecting rules
without a `version:` field).

**Closes when:** intentionally introducing a difference
between the two schema files causes the byte-equality CI
gate to fail (and reverting the difference makes CI green
again).

**Phase 4 — Engine runtime scaffold.**

Engine-only (no rules content yet):

- **Loader** — startup-mode and refresh-mode behavior per
  ADR-0007.
- **Runner** — `execution_id` computation per ADR-0002
  (five inputs, pipe-separated, sha256 hex, no escaping;
  `trigger_source` enum).
- **Result write** — append-only writes to `dq_executions`
  and `dq_check_results` per ADR-0003;
  `dq_executions_current` lazy view per ADR-0003.
- **Failure scope** — execution-status mapping per
  ADR-0004; always-continue at the check level.
- **Orphan-run detection** — periodic scan per ADR-0007.

Phase 4 exercises the "tabular store: append-only writes"
and "orphan-run detection polling" capability-matrix rows
from ADR-0010 end-to-end. The "tabular store: lazy view"
fidelity gap ships with the sandbox-required validation
documented under ADR-0010.

May split into sub-phases (loader / runner / write /
failure scope / orphan); the first Phase-4 session
decides.

**Phase 5 — Alerting scaffold.**

Pub/Sub publisher in the engine per ADR-0006 (event
payload schema, `event_source` enum); engine-side dedup
per ADR-0006 (no duplicate
`(execution_id, attempt_id, check_id, result)` tuple
within an attempt); `_owners.yaml` schema fragment per
ADR-0006 with `schema_version`, `entities[]`, `owner`,
`channels`; linter rule rejecting entities without an
`_owners.yaml` entry.

Phase 5 exercises the "Pub/Sub publish/subscribe"
capability-matrix row from ADR-0010 end-to-end.

**Phase 6 — `rules/` first onboarded entity.**

One real entity end-to-end through Phases 3–5: a real
`_owners.yaml` entry, a real ruleset manifest published
to the local object store per ADR-0005, the loader from
Phase 4 picks it up, the runner produces `dq_executions`
rows, the alerting from Phase 5 publishes events.

**Closes when:** the flow "manifest publish → loader
hash-short-circuit refresh → execution write →
operational alert publish" runs locally without sandbox.

**Phase 7 — `deploy/` scaffold.**

Kubernetes manifests for the engine, environment overlays
for `local`, `qa`, `prod`. Depends on the environment-
configuration-model decision (open in the decision log's
B1 list); if that decision is still `open` when Phase 7
begins, it is resolved in a B1 study session before
Phase 7 proceeds.

**Phase 8 — `docs/` content beyond ADRs.**

Glossary, governance, contribution guide, runbook seeds.
ADRs from Phase 1 are already in place; this phase adds
the prose surfaces around them.

---

## Consequences

1. **Loop discipline is uniform across phases.** Every
   Wave 3 session runs the same loop (plan-mode + AC-W3
   self-check + critique rounds + decision-log update +
   commit). The playbooks at `.claude/playbooks/` are the
   contract; this ADR is the sequencing dimension layered
   on top of them.

2. **Dependency safety is structural, not author-
   judgement.** A Phase-N session may not cite a Phase-M
   surface where M > N. The compatibility-contract checks
   in ADR-0001 cannot be wired in Phase 2 because Phase 2
   does not yet have the schema layer; they wait for
   Phase 3.

3. **The capability-matrix coverage commitment is bounded
   in time.** Every **Yes** row in ADR-0010 is exercisable
   by a local test or runnable command by the end of
   Phase 5 (the alerting phase that completes the
   end-to-end signal path). The **No** and **Partial**
   rows remain sandbox-required throughout Wave 3.

4. **Each phase has a closing condition.** Phase 2 closes
   on a working local Compose. Phase 3 closes on the
   byte-equality gate failing intentionally and passing
   when reverted. Phase 6 closes on the end-to-end flow.
   The closing condition is what makes the next phase
   safe to start.

5. **B0 / W2 decisions are no longer reopenable.** They
   are committed in ADRs 0001–0012. A Wave 3 session
   discovering a gap in an upstream commitment writes a
   new ADR superseding the relevant prior ADR; it does
   not silently work around the gap.

6. **R1 ceased applying when the Wave 2 gate closed.**
   Production code becomes permissible from Phase 2
   onward. R3–R8 still apply throughout Wave 3. R5 in
   particular needs vigilance at scaffold scale —
   variable names, package paths, and external-SDK
   imports all carry the risk of naming sibling-team or
   prior-art systems.

7. **The decision log gains a Wave 3 — Phases table.**
   Each phase row resolves to the final commit reference
   that closes the phase. Phase 0 closed in commit
   `25e06ab`; Phase 1 closes in the commit that lands
   this ADR.

8. **Phase 4 may split.** Loader, runner, result-write,
   failure-scope, and orphan-detection are different
   surfaces; the first Phase-4 session decides whether
   the phase splits into sub-phases and updates this ADR
   if it does.

9. **B1 rows may interleave with Wave 3 phases by
   demand.** A Wave 3 phase that needs an open B1 row
   resolves the B1 row first, in a separate study
   session, before the phase proceeds. Phase 7 in
   particular has a known B1 dependency on the
   environment-configuration-model decision.

10. **Phase 1 closes in this commit.** All thirteen
    Phase-1 ADRs exist; every resolved B0 and W2 row in
    the decision log carries a `resolved-adr` link. Wave 3
    progression after this commit is Phase-2-onward.

---

## Notes

- The cadence of Phase 1 was batched-by-wave: Session A
  produced ADRs 0001–0007 in one commit; Session B
  produced ADRs 0008–0012 in one commit; Session C (this
  commit) produced ADR-0013. Each batch was internally
  cohesive and amenable to a single critique pass.
- Whether the project needs a separate `/critique-adr`
  skill (distinct from `/critique`) is a Wave-3 protocol
  follow-up. `/critique` was used throughout Phase 1; the
  study-vs-ADR mismatch produced minor polish findings
  but no blocking issues. The split is worth pursuing
  only if Phase-2-onward sessions produce critiques that
  the current skill cannot evaluate cleanly.
- Whether Phase 4 splits into sub-phases is decided by
  the first Phase-4 session, not predetermined here.
- The `tools-lint-v*` tag-prefix scope question (whether
  it covers the whole `tools/` directory or only the
  linter binary) is revisited when a second tool binary
  lands; per ADR-0012.
- Pre-release publication rules (whether a
  `rules-v*.*.*-rc*` value may write
  `manifests/latest.json`) are a follow-up to ADR-0005,
  independent of the Wave 3 phase structure.
- The ADR numbering convention (Phase-1 promotes seven
  B0 + five W2 + one sequencing into 0001–0013) is now
  fixed. Future ADRs continue from 0014.
