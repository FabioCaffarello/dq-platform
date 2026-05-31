<!-- path: studies/critiques/2026-05-31-cross-mode-parity-integration-test-critique-1.md -->

# Critique — Cross-mode table-shape parity integration test (Round 1)

- Target: `engine/internal/runner/parity_integration_test.go`
- Round: 1 (max 2 per `.claude/playbooks/post-wave3-session-loop.md` step 7)
- Run date: 2026-05-31
- Captured per [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md).

---

**No blocking findings.** R1 historical (waves closed); R5 / R6 / R7
/ R8 all pass; P1–P6 N/A for a test file; AC-1 / 2 / 3 / 4 do not
apply (Go test, not a B-item study).

---

[important] AC-W3-3 / P5: file header — record-mode aggregation per
ADR-0026 is **not** exercised by this test. `NoopEvaluator` returns
`ResultPass` for both modes, bypassing ADR-0026's per-record →
per-window threshold aggregation that produces the record-mode
`result` value. The test asserts **table-shape** parity (the user's
prompt), not aggregation-shape parity. Add an explicit one-line
out-of-scope note in the file doc-comment: "Aggregation-shape parity
(ADR-0026) is exercised by `record_runner_test.go` unit tests; this
slice asserts only the consumer-facing table contract."

[important] AC-W3-4: Phase-3 assertion (5), execution_id verification
(lines 231–261) — carries dead code. `setTrigger.RulesetVersion` is
left empty at construction; `Runner.Run` mutates an internal copy of
the trigger (line 305–306 of `runner.go`), not the caller's struct.
So `setRecomputed` at line 231 is always
`SHA256("", entity, ws, we, scheduler)`, which can never match
`setTerminal.ExecutionID` (computed with `"rules-v1.0.0"`). The
`if setRecomputed == setTerminal.ExecutionID` branch (line 244) is
dead; only the `else` ever runs. Collapse to a single
`Compute("rules-v1.0.0", …)` call and a single equality assertion.
Misleading code in a contract-guarding test invites future drift.

[important] AC-W3-9: decision-log "Last updated" entry — the plan
committed an entry under `studies/foundation/06-decision-log.md`
(post-Wave-3 loop step 9). The commit `12cabd5` ships the test file
only. Append the "Last updated: 2026-05-31 (cross-mode parity
integration test shipped against ADR-0041 …)" line before opening
the PR, or the close-discipline gate is incomplete.

[minor] AC-W3-2 / R7: `CheckSpec.Mode` literals (lines 99, 137) —
hardcoded `"set"` and `"record"` instead of
`string(results.ModeSet)` / `string(results.ModeRecord)`. ADR-0021
makes the enum the source of truth; tying the test to the typed
constant prevents a rename from going undetected.

[minor] AC-W3-2: repeated `"rules-v1.0.0"` literal (lines 127, 163;
248 disappears with the dead-code collapse) — extract a
`const testRulesetVersion = "rules-v1.0.0"` so the formula-input
constant is named.

[minor] AC-W3-3: assertion (7) comment cites ADR-0041
§"Cross-mode dashboard interpretation" Rule 1, whose example queries
`dq_executions_current` (the canonical view). The test queries the
base table with an inline `ROW_NUMBER()` projection because of
ADR-0010's lazy-view Partial row. The substitution is the same
semantic but worth one line of comment for the next reader.

[minor] AC-W3-3: TriggerSource parity not directly asserted —
`assertRequiredColumnsPopulated` only checks `TriggerSource != ""`.
Both rows must equal `results.TriggerScheduler` (record-mode
hardcodes it in `closeAndDispatch`; set-mode sets it in the
trigger). Equality is implicit in the execution_id lookup
succeeding, but a direct equality assertion would close the gap and
document the invariant.

[minor] Go-style nit: lines 244–246 carry the
`if cond { /* nothing */ } else { ... }` shape — invert to
`if cond != other { ... }`. Disappears once the dead-code finding
above is fixed.

---

## Disposition (round 1)

Three `important` findings — all applied in the round-1 revision
commit:

1. Out-of-scope marker added to file doc-comment naming ADR-0026
   aggregation-shape coverage as live in `record_runner_test.go`
   unit tests.
2. Dead-code collapse in assertion (5): one `Compute("rules-v1.0.0",
   ...)` call, one equality assertion, no inverted-branch shape.
3. Decision-log "Last updated" entry appended.

Minor findings 1–3 (`Mode` enum constants, `RulesetVersion` const,
view-vs-base-table note) — applied as part of the same revision
commit (cheap to do; tightens the artifact).

Minor findings 4–5 (TriggerSource direct equality, Go-style nit) —
the latter is resolved by the dead-code collapse; the former is
applied (one line, no risk).

No round 2 needed — every finding is applied, no rebuttal-shaped
disagreement.
