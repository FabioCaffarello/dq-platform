<!-- path: studies/critiques/2026-05-30-b3-panel-5-lighting-strategy-critique-2.md -->

# Critique — `studies/decisions/2026-05-30-b3-panel-5-lighting-strategy.md` — round 2 (ratification trailer)

Round 2 was not invoked as a fresh adversarial pass — round 1
landed at 0 blocking, the four important findings were applied,
and the five minor findings were deferred under the two-round
cap. This round-2 capture exists solely to record the
operator-side ratification of the round-1 eligibility-check D0
per [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)
preservation contract and per
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5 §"Operator-side
responsibilities".

## Findings

**None.** No new blocking, important, or minor findings surfaced
in round 2. Round 1's disposition stands.

## D0 ratification (operator-side, mid-PR-#113)

**2026-05-30, mid-PR-#113 ratification:** the operator ratified
the two coupled D0s on [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
§(a) Conditions 1 (P-B3.1 expands-not-rewrites) and 3 (P-B3.2
envelope conformance) for B3-5 by adopting the **weak reading**
of [ADR-0039](../../docs/adr/0039-dashboard-contract.md)
§"Metric contract" Meaning column wording *"Count of runs the
scheduler currently tracks, split by state"* AND the **A.y
sub-path** (emit constant zero for engine-non-derivable series,
not drop them).

Under that ratified reading:

- **Condition 1 (P-B3.1)** passes. The committed contract surface
  is the gauge's semantics (queue depth, label `state`); the
  source identification in the Meaning column described the
  ADR-0039-time-known emitter without committing the source as
  a label-source rule. Emitting the same gauge name from an
  engine-derived source — combined with the additive
  `source="engine"` label per ADR-0039 §"Evolution rules" — is
  extension, not rewrite.

- **Condition 3 (P-B3.2)** passes. The additive `source` label
  is the load-bearing mechanism: by self-identifying the gauge
  emission as engine-derived, the engine no longer claims
  scheduler-internal knowledge it doesn't have, so ADR-0033's
  external-scheduler boundary is preserved verbatim. ADR-0020 /
  0021 / 0022 / 0023 envelope is untouched.

- **Sub-path A.y over A.x.** A.x (dropping
  `dq_scheduler_triggers_managed` from engine emission) was
  amendment-shaped under either reading per the §Option A 2×2
  table in the study; A.y (constant zero for engine-non-derivable
  series) is the only cell of the 2×2 that's B3-eligible under
  the weak reading.

The ratification is **mid-PR** (recorded before merge) rather
than at-merge — same precedent shape established by B3-4 / PR
#111 (operator-ratified Condition 2 D0 mid-PR-#111 on
2026-05-30). The mid-PR disposition makes the PR body + in-tree
artifacts reflect the ratified state at [H] reviewer concurrence
time.

This trailer is the ratification artifact per
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5 §"Operator-side
responsibilities" — "eligibility ratification for borderline
B3-N readings is operator-side, recorded in the round-2 critique
trailer, carried forward to the promoted ADR as a
new-contribution-requires-review marker per R5". The
author-equals-reviewer circularity (`/critique` cannot
self-ratify its own eligibility reading per
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
§Consequence 7) is satisfied: the ratification is
operator-emitted, not critique-emitted.

Carry-forward to ADR-0056 lands in the promotion commit per A7
of [`.claude/skills/adr-writing/SKILL.md`](../../.claude/skills/adr-writing/SKILL.md).
The carry-forward marker is **new contribution requiring
review**: the operator-ratified weak reading of ADR-0039's
"scheduler currently tracks" wording, plus the structural
mechanism (additive `source` label as the extension-shaped
mechanism that satisfies Condition 3 under the weak reading),
are the precedented interpretations future B-rows touching
ADR-0039 wording can inherit.

## Follow-on disposition

The ratification unlocks **Path A.y** as the actionable B3-N
implementation slice for OQ-1 from ADR-0055 / B3-4. The follow-on
session ships:

- `dq_queue_depth{state="running",source="engine"}` emitted from
  a `Store.LatestExecutionPerEntityCheck`-style reader per
  Prometheus scrape interval, partition-pruned per ADR-0031.
- `dq_queue_depth{state="scheduled",source="engine"}` emitted as
  constant zero (engine cannot derive scheduler-tracked
  scheduled state).
- `dq_scheduler_triggers_managed{state="healthy",source="engine"}`
  and `dq_scheduler_triggers_managed{state="errored",source=
  "engine"}` emitted as constant zero (engine cannot derive
  trigger-management state).

ADR-0039 §"Metric contract" Meaning column is **not re-written**
under the weak reading — the additive `source` label is the
mechanism that reconciles the engine emission with the existing
wording. Operators reading the dashboard see panel 5 light
partially (the `running` count is real; the rest are flat at
zero) with the `source="engine"` label self-identifying the
emission.

The follow-on session is a separate B-row (not folded into B3-5
under R4 — that's the same shape B3-4 followed before the
operator-authorized R4 collapse for ADR-0055 was invoked on
PR #112). The B-row classification (fresh B3-N for additional
wiring decisions vs B2-style implementation slice under closed
B3-5) is the operator's call at promotion time.
