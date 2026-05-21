<!-- path: studies/decisions/2026-05-21-wave3-sequencing.md -->

# Wave 3 — Sequencing Study

## Metadata

- **Wave reference:** Wave 3 (sequencing)
- **Status:** resolved-study (two critique rounds; round 2 cleared
  with zero blocking and zero important findings)
- **Last updated:** 2026-05-21
- **Upstream resolved:** B0-1, B0-2, B0-3, B0-4, B0-5, B0-6, B0-7,
  W2-1, W2-2, W2-3, W2-4, W2-5
- **Downstream:** every Wave 3 scaffolding session
- **Promotion target:** `docs/adr/0013-wave3-sequencing.md`
- **Loop discipline:** governed by
  `.claude/playbooks/wave-3-session-loop.md` and
  `.claude/playbooks/wave-3-acceptance-criteria.md`

---

## Context

Wave 1 closed with seven `B0` decisions at `resolved-study`
(gate met, 7/7). Wave 2 closed with five `W2` items at
`resolved-study` (gate met, 5/5). Wave 3 — scaffolding the five
product workspaces (`engine/`, `rules/`, `tools/`, `deploy/`,
`docs/`) plus the root infrastructure — is unblocked.

Wave 3 is **larger and more entangled** than Waves 1 and 2.
Without an explicit phase ordering, sessions can:

- scaffold a downstream surface before its upstream exists
  (e.g., `rules/_schema/v1.schema.json` byte-mirror before
  `engine/internal/dsl/schema/v1.schema.json` source — the
  B0-1 C2 CI gate would have nothing to compare against);
- collide on shared files (two sessions both editing
  `go.work`);
- build on capability rows that haven't been wired up
  (e.g., emit Pub/Sub events before the local emulator from
  W2-3's capability matrix is part of `docker-compose.yml`);
- cite ADRs that haven't been promoted yet, leaving brittle
  forward-references to `studies/` from production code
  (R8 violation).

This study picks the **phase structure** that prevents those
failure modes. It does not pick concrete artifacts (emulator
images, exact Go module layout, CI runner version pins) —
those are sub-decisions for the sessions inside each phase.

---

## Decision Drivers

D1. **Dependency order.** No scaffolding unit may cite an
upstream that hasn't been scaffolded. The CI byte-equality gate
(B0-1 C2) is the canonical example: it requires both the engine
schema source and the rules schema mirror to exist before it
can be wired up.

D2. **Loop discipline (R4).** Each session is one unit; each
unit sits inside one phase. Crossing phase boundaries
mid-session is the equivalent of revisiting a settled B0 — it
silently expands scope and erodes the quality bar.

D3. **CI gates land last among schema layers.** Wiring a
mandatory gate before its inputs exist creates a noisy
red-on-main, which trains contributors to ignore the gate.

D4. **Capability-matrix coverage (W2-3 §3.3).** Each **Yes**
row in the W2-3 capability matrix should be exercisable by a
local test or runnable command by the end of Wave 3. Phases
that produce those tests must come after the phase that wires
up the capability.

D5. **ADR promotion before code that cites ADRs.** User
directive in the Wave 3 protocol session, 2026-05-21:
`protocolo então ADRs` (Portuguese; English: "protocol, then
ADRs"). Scaffolding sessions that cite an ADR path must run
after the ADR exists; sessions that cite a study path
(Phases 0, 1) run before.

D6. **R5 hygiene at scaffold scale.** Code is harder to police
than documents — variable names, package paths, comment
references, and external-vendor SDK imports all carry the risk
of naming sibling-team or prior-art systems by name. The
sequencing must make this visible early (Phase 2 root
infrastructure) so the convention takes root before larger
scaffolds land.

D7. **Cost-as-first-class (P4).** Phases that need substrate
(Phase 4 onward) consume the local Compose substrate from
Phase 2. Reversing the order forces contributor onboarding
through cloud sandbox for early phases — unacceptable.

---

## Considered Options

- **(A) Workspace-sequential.** Finish `engine/` completely,
  then `rules/`, then `tools/`, then `deploy/`, then `docs/`.
  Simple to reason about; fails on D1 because the B0-1 C2 CI
  gate cannot be wired without both `engine/` and `rules/`
  schema artifacts existing, so either `engine/` is "finished"
  but with the gate deferred, or the workspace boundary leaks.
- **(B) Vertical slice.** Pick one end-to-end flow (manifest
  publish → loader refresh → execution write → operational
  alert), scaffold the **minimum** of every workspace to
  support it, then iterate with more flows. Maximizes early
  end-to-end signal; risks (i) leaving every workspace
  half-scaffolded for a long time, (ii) heavy session
  switching between workspaces, which is hard to keep R4-clean.
- **(C) Phased: root-first, dependency-edges-second,
  capability-coverage-last.** Explicit phase boundaries; each
  phase is internally cohesive; phase order respects D1–D7.
  **Recommended.**

---

## Recommendation

**(C).** Eight phases (0 through 7) plus a final docs phase
(8). Phase boundaries are explicit; inside a phase, scaffolding
units can move in any order so long as they do not cross
phase lines.

Phases 0 and 1 are protocol and ADR work, not scaffolding —
they are listed so the complete Wave 3 picture is visible from
one document.

---

## Consequences — phase structure

The eight-phase partition below — Phase 0 (protocol), Phase 1
(ADR promotion), and Phases 2–8 (scaffolding work) — is a
**new contribution proposed here, requires review**. Each phase
internally cites the B0 / W2 commitments it implements, but
the partition shape itself does not appear in any foundation
document or prior decision. The first reader pass should treat
the phase boundaries as the central architectural commitment
of this study.

### Phase 0 — Protocol (this session)

Artifacts: this study;
`.claude/playbooks/wave-3-session-loop.md`;
`.claude/playbooks/wave-3-acceptance-criteria.md`.
**State:** in this session. Closes when this study is at
`resolved-study` and the two playbooks are committed.

### Phase 1 — ADR promotion

Run `/promote-to-adr` on each of the twelve resolved studies,
in the slug order committed by the Wave 2 study and by the
default B0 ordering:

| Source study | Target ADR |
|---|---|
| `2026-05-20-engine-rules-compatibility.md` (B0-1)            | `docs/adr/0001-engine-rules-compatibility.md` |
| `2026-05-20-run-identity-and-idempotency.md` (B0-2)          | `docs/adr/0002-run-identity-and-idempotency.md` |
| `2026-05-20-result-write-model.md` (B0-3)                    | `docs/adr/0003-result-write-model.md` |
| `2026-05-20-failure-scope.md` (B0-4)                         | `docs/adr/0004-failure-scope.md` |
| `2026-05-20-manifest-publication-semantics.md` (B0-5)        | `docs/adr/0005-manifest-publication-semantics.md` |
| `2026-05-21-alert-routing-contract.md` (B0-6)                | `docs/adr/0006-alert-routing-contract.md` |
| `2026-05-20-loader-scheduler-retry-failure-semantics.md` (B0-7) | `docs/adr/0007-loader-scheduler-retry-failure-semantics.md` |
| `2026-05-21-platform-decisions-wave2.md` §1 (W2-1)           | `docs/adr/0008-git-host.md` |
| `2026-05-21-platform-decisions-wave2.md` §2 (W2-2)           | `docs/adr/0009-multi-agent-contract.md` |
| `2026-05-21-platform-decisions-wave2.md` §3 (W2-3)           | `docs/adr/0010-substrate-posture.md` |
| `2026-05-21-platform-decisions-wave2.md` §4 (W2-4)           | `docs/adr/0011-documentation-language.md` |
| `2026-05-21-platform-decisions-wave2.md` §5 (W2-5)           | `docs/adr/0012-tag-conventions.md` |

Plus this sequencing study, promoted last so it can cite the
twelve ADRs by their final paths:

| Source study | Target ADR |
|---|---|
| `2026-05-21-wave3-sequencing.md`                             | `docs/adr/0013-wave3-sequencing.md` |

Each promotion is one session or batched, at the project
lead's discretion; the sequencing leaves that open
(OQ-W3-5 below). After Phase 1 closes, every decision-log row
gains a `resolved-adr` link alongside the existing
`resolved-study` link.

### Phase 2 — Root infrastructure

Artifacts: `go.work`, `Makefile`, `docker-compose.yml`,
`.github/` (CI lanes per W2-1; branch protection per
B0-1 C2), `.codex/AGENTS.md` (pointer file per W2-2 C-W2-2.2),
top-level `README.md`, and the empty top-level layout of each
of the five workspaces.

Phase 2 wires up the **W2-3 §3.3 capability matrix** locally:
the **Yes** rows must each be reachable from `make up` or
equivalent. The **No / Partial** rows are documented as
sandbox-required.

The Phase-2 closing contract — `make up` brings up every
**Yes** capability, `make test` smokes them, `make lint` runs —
is a **new contribution proposed here, requires review**.
Foundation 02 references a `Makefile` but does not commit to
these specific targets; the first Phase-2 session may revise the
target names, but the *coverage requirement* (every **Yes**
capability reachable locally) is load-bearing.

R1 ("no production code during Waves 1 and 2") ceased applying
the moment the Wave 2 gate closed; Phase 2 is the first phase
that *exercises* the relaxed R1 by producing production code.
R3–R8 still apply throughout Phase 2 onward.

### Phase 3 — Schema-layer scaffold

Artifacts:

- `engine/internal/dsl/schema/v1.schema.json` — canonical
  source of the rule schema (B0-1 C8 CODEOWNERS-protected).
- `rules/_schema/v1.schema.json` — byte-equal mirror of the
  engine source (B0-1 C1, C2).
- `tools/lint/` — minimum linter binary that enforces the
  byte-equality CI gate (B0-1 C2) and rejects rules without a
  `version:` field (B0-1 C4). Linter version pin lives in CI
  per B0-1 C10 with the host-primitive chosen in Phase 2.

CI lane: the byte-equality gate becomes a required check on
`main` (B0-1 C2). Closes when intentionally introducing a
difference between the two schema files causes the
byte-equality CI gate to fail (and reverting the difference
makes CI green again).

### Phase 4 — Engine runtime scaffold

Artifacts (engine-only; no rules content yet):

- **Loader** — startup-mode and refresh-mode behavior per
  B0-7 CC1, CC2, CC9 (hash short-circuit on `manifests/latest.json`).
- **Runner** — `execution_id` computation per B0-2 CC1, CC2;
  five inputs, pipe-separated, sha256 hex, no escaping.
  Trigger-source enum per B0-2 CC6.
- **Result write** — append-only writes to `dq_executions` and
  `dq_check_results` per B0-3 CC1; required columns per
  B0-3 CC3 and CC7; `dq_executions_current` lazy view per
  B0-3 CC2.
- **Failure scope** — execution-status mapping per B0-4 CC1,
  CC2, CC3; always-continue at check level per B0-4 CC4.
- **Orphan-run detection** — periodic scan per B0-7 CC11.

Phase 4 exercises the W2-3 §3.3 rows "Tabular store: append-only
writes" (**Yes**) and "Orphan-run detection polling" (**Yes**)
end-to-end. The "Tabular store: lazy view" (**Partial**) row
ships with a known fidelity gap, documented per
C-W2-3.6 / OQ-W2-3.3.

#### Sub-phase split (resolves OQ-W3-2)

The first Phase-4 session (2026-05-21) decided to split Phase 4
into four cohesive sub-phases along the natural data-flow
dependency edges. Each sub-phase is internally cohesive and
sized for a single Wave-3 session under the
`.claude/playbooks/wave-3-session-loop.md` discipline.

- **W3-P4a — Loader.** Manifest fetch + hash short-circuit +
  refuse-swap. Implements B0-7 CC1, CC2, CC9; consumer side
  of ADR-0005 §4 publication contract. Depends only on
  Phase 2's object store and Phase 3's schema mirror.

- **W3-P4b — Result-write layer.** Append-only storage for
  `dq_executions` + `dq_check_results` + lazy
  `dq_executions_current` view. Implements B0-3 CC1, CC2,
  CC3, CC7. Stand-alone subsystem; depends only on Phase 2's
  tabular store.

- **W3-P4c — Runner + failure-scope mapping.** The runtime
  path that wires the loader (W3-P4a) and the result-write
  layer (W3-P4b) together via `execution_id` computation
  (B0-2 CC1/CC2/CC6) and the failure-scope status mapping
  (B0-4 CC1/CC2/CC3/CC4). Also lands the pre-check
  entity-level validation (B0-7 CC8) and the per-failure-path
  observability emission (B0-7 CC14). Depends on both W3-P4a
  and W3-P4b.

- **W3-P4d — Orphan-run detection.** Periodic scan finalizing
  abandoned `running` rows to `aborted` (B0-7 CC10
  engine-restart / OOM / crash branches; B0-7 CC11). Depends
  only on W3-P4b.

**Dependency edges and recommended landing order.** W3-P4a
and W3-P4b are independent of each other; W3-P4d depends on
W3-P4b alone; W3-P4c depends on both W3-P4a and W3-P4b. The
recommended landing order is P4a → P4b → P4c → P4d so the
manifest-load and storage-layer foundations land before the
runner that orchestrates them, with the orphan-detection
rescue path closing the phase. The first session of W3-P4a
or W3-P4b may pick either order; either start is correct.

Each sub-phase closes with its own commit and decision-log
row update; the parent W3-P4 row in the Wave 3 — Phases
table is marked `split` and the four child rows (W3-P4a /
W3-P4b / W3-P4c / W3-P4d) carry the individual statuses.

### Phase 5 — Alerting scaffold

Artifacts:

- Pub/Sub publisher in the engine per B0-6 CC3, CC4 (event
  payload schema, source enum).
- Engine-side dedup per B0-6 CC5 (no duplicate
  `(execution_id, attempt_id, check_id, result)` within an
  attempt).
- `_owners.yaml` schema fragment per B0-6 CC1, with
  `schema_version`, `entities[]`, `owner`, `channels`.
- Linter rule per B0-6 CC9 (linter rejects entities without
  `_owners.yaml` entry).

Exercises W2-3 §3.3 row "Pub/Sub publish/subscribe" (**Yes**)
end-to-end.

### Phase 6 — `rules/` content first onboarded entity

Artifacts: one onboarded entity end-to-end through Phases 3–5.
A real entity, a real `_owners.yaml` entry, a real ruleset
manifest published to the Phase-2 local object store per
B0-5 CC1, CC3, CC4. The loader from Phase 4 picks it up; the
runner produces `dq_executions` rows; the alerting from
Phase 5 publishes events.

Closes when the end-to-end flow named in W2-3 C-W2-3.4
("manifest publish → loader hash-short-circuit refresh →
execution write → operational alert publish") runs locally
without sandbox.

### Phase 7 — `deploy/` scaffold

Artifacts: Kubernetes manifests for the engine, environment
overlays for `local`, `qa`, `prod`. Depends on B1-4
(environment configuration model) being resolved; if B1-4 is
still `open`, this phase blocks (or B1-4 resolves alongside
Phase 7 as a B1 study session — see OQ-W3-3).

### Phase 8 — `docs/` content beyond ADRs

Artifacts: glossary, governance, contribution guide, runbook
seeds. ADRs from Phase 1 are already in place; this phase
adds the prose surfaces around them.

---

### Cross-phase contracts

Every phase's sessions run under
`.claude/playbooks/wave-3-session-loop.md` and self-verify
against `.claude/playbooks/wave-3-acceptance-criteria.md`.

The decision log gains a **"Wave 3 — Phases"** subsection
alongside this study's commit; each phase row resolves to a
final commit reference when its phase closes. The subsection
structure itself is a **new contribution proposed here,
requires review** — no foundation document or prior decision
specifies how the decision log should evolve once scaffolding
starts. See §Promotion target for the exact log edits.

---

## Open Questions

- **OQ-W3-1.** Whether `/critique` is sufficient for
  scaffolding units, or whether a `/critique-scaffold` skill is
  needed (different blocking heuristics — build failures, AC-W3
  rows, R5 in code vs prose). **Out-of-scope for current cycle
  — start with `/critique`; split if the two shapes diverge
  enough that the same skill is producing low-signal output.**
  (new contribution proposed here, requires review)

- **OQ-W3-2.** Whether Phase 4 splits into sub-phases
  (loader / runner / result-write / failure-scope / orphan).
  **Resolved 2026-05-21 by the first Phase-4 session: Phase 4
  is split into four sub-phases (W3-P4a Loader, W3-P4b
  Result-write, W3-P4c Runner+failure-scope, W3-P4d Orphan
  detection); recommended landing order P4a → P4b → P4c →
  P4d. See §"Phase 4 — Engine runtime scaffold" above and
  the Wave 3 — Phases table in the decision log.**

- **OQ-W3-3.** Whether B1 rows (B1-1 through B1-10) resolve
  interleaved with Wave 3 phases or as a parallel stream.
  Phase 7 in particular depends on B1-4. **Out-of-scope for
  current cycle — sequenced by demand: a Wave 3 phase that
  needs an open B1 row resolves the B1 row first, in a
  separate study session.**

- **OQ-W3-4.** ADR numbering — the sequencing study takes
  `0013`, after the five W2 ADRs at `0008–0012`. If the
  Phase-1 promotion sessions pick a different B0 ordering than
  this study's table, the slot numbers shift, and `0013`
  shifts with them. **Out-of-scope for current cycle — the
  slug (`wave3-sequencing`) is the stable part; the number
  adjusts at promotion time.** (new contribution proposed
  here, requires review)

- **OQ-W3-5.** Whether Phase 1 ADR promotion runs as one
  session, one session per ADR, or batched (e.g., all B0 in
  one, all W2 in one). **Out-of-scope for current cycle — the
  first Phase-1 session picks the cadence and records it in
  the commit message.**

---

## Promotion target

`docs/adr/0013-wave3-sequencing.md`. Promoted at the **end**
of Phase 1, after the twelve upstream ADRs are in place, so
the sequencing ADR can cite each by its final path instead of
forward-referencing the studies.

**Decision-log update.** On approval of this study, two edits
land in `studies/foundation/06-decision-log.md`:

1. Add a **"Wave 3 — Phases"** subsection listing Phases 0
   through 8 with status `open` initially.
2. Annotate the existing **"Wave 3 readiness"** section
   (currently the gate-pending paragraph) with: "gate met as
   of 2026-05-21; phase progression tracked in the Wave 3 —
   Phases table above."

These updates land in the **same session** that commits this
study (per Wave 1 session loop step 9; same convention applied
to Wave 3 protocol bootstrap).
