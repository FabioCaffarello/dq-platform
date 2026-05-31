<!-- path: studies/critiques/2026-05-31-fix-eval-integration-test-build-critique-1.md -->

# Critique — Unbreak eval integration test build (Round 1)

- Target: `engine/internal/eval/evaluator_integration_test.go`
- Round: 1 (max 2 per `.claude/playbooks/post-wave3-session-loop.md` step 7)
- Run date: 2026-05-31
- Captured per [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md).

---

**No blocking findings. No important findings.** R1 historical;
R5 / R6 / R7 / R8 all pass; P1–P6 N/A for a test file;
AC-1…AC-4 do not apply (Go test, not a B-item study). The
refactor preserves the original test's intent (pass on 3 rows,
fail on 0 rows, error on missing table, end-to-end runner →
evaluator → BQ wire) and tracks the post-Wave-S evolutions of
the eval package (`KindRowCountPositive` → `KindSetRowCountPositive`
per ADR-0022; per-rule `CheckSpec.Source` per ADR-0023).

---

[minor] AC-W3-9: PR #123's body flagged this build break as a
follow-up housekeeping item. A one-line "Last updated" entry in
`studies/foundation/06-decision-log.md` closing that loop is
appropriate (eval integration coverage restored, four tests
green) — not load-bearing for any AC because no B-row or ADR
Consequence row is being closed, but the audit trail matches
the pattern from prior chore-shaped fixes.

[minor] AC-W3-3: each `runner.CheckSpec` literal could carry
`Mode: string(results.ModeSet)` to align with the parity-test
precedent committed in PR #123. The evaluator doesn't read
`spec.Mode` for `set.row_count_positive` (the kind prefix
already determines the mode), so omitting it doesn't break the
test — but the parity test's convention of always populating
Mode tightens the cross-test consistency story.

[minor] AC-W3-3 / `bqSource` helper: does not expose
`PartitionColumn`. None of the four tests need it today, but a
future integration test exercising the B2-12 partition-pruning
path (ADR-0029 §"row_count_positive cost gap") will need to
either extend the helper or construct `RuleSource` inline.
YAGNI today.

[minor] AC-W3-2 / `TestIntegration_RowCountPositive_ErrorOnMissingTable`:
entity name `"does_not_exist"` conveys test intent but breaks
the convention that `entity` names a real domain object. The
test's `Source.TableID = "does_not_exist"` is what actually
drives the missing-table branch.

[minor] AC-W3-2 / `stdTrigger` doc-comment: phrasing reads
awkwardly. Tighten to "Direct-Evaluate tests bypass
`validateTrigger`; the runner-routed test path needs
`WindowEnd > WindowStart`. Endpoints are populated uniformly
for hygiene."

[minor] AC-W3-7: dataset uniqueness uses `time.Now().UnixNano()`
without `t.Name()`. Theoretical collision under highly-parallel
test execution; matches existing convention across the codebase
(`runner_integration_test.go`, `results_integration_test.go`).
Codebase-wide hygiene task, not this slice.

[minor] AC-W3-7 / `int64` type assertion: silent comma-ok form.
If the evidence-summary contract changes the type, the test
fails with `row_count = 0; want 3`, which is correct but
misleading. Matches pre-existing style.

---

## Disposition (round 1)

Operator directive: open the PR (no touch-up). The six minor
findings are accepted-as-is for this slice:

- Finding 1 (decision-log entry) — recorded in this critique
  capture; the close-PR body carries the same audit trail
  durably. Skipping the explicit decision-log entry to keep the
  chore-shaped scope tight; the PR body itself names the
  W3-P6c build break being closed.
- Finding 2 (`Mode: "set"` on CheckSpec) — cosmetic
  cross-consistency; the kind-prefix already determines the
  mode for `set.row_count_positive`.
- Findings 3–6 — YAGNI / cosmetic / codebase-wide.

No round 2 — every finding is dispositioned without rebuttal.
