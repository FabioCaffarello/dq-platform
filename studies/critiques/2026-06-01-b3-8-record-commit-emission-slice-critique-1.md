<!-- path: studies/critiques/2026-06-01-b3-8-record-commit-emission-slice-critique-1.md -->

# Round 1 — Critique of B3-8 (Record-Commit Emission Slice)

- Target: [`studies/decisions/2026-05-31-b3-8-record-commit-emission-slice.md`](../decisions/2026-05-31-b3-8-record-commit-emission-slice.md)
- Round: 1
- Date: 2026-06-01
- Reviewer: Claude (adversarial critique per `.claude/commands/critique.md`)
- Count: **0 blocking / 5 important / 5 minor**
- Disposition summary: all 5 important applied to the study in the same revision; 5 minor deferred under the two-round cap per [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md).

---

## Re-grounding

- `CLAUDE.md` §3 (R1–R8) + §4 (P1–P6) re-read.
- `.claude/playbooks/acceptance-criteria.md` (AC-1…AC-10) re-read.
- `.claude/playbooks/feedback-protocol.md` (R/P/AC label template) re-read.
- Target study read end-to-end.

---

## Blocking

**None.** The study clears AC-1 (path header at line 1), AC-2 (all required sections present in order), AC-3 (four options considered), AC-4 (Recommendation grounded in prior decisions + DD chain), AC-5 (no sibling-team or prior-art naming; Prometheus is environment-commodity per R5 exemption), AC-6 (all four OQs marked "Out-of-scope for current cycle" with one-line reasons), AC-7 (Promotion target points to a concrete `docs/adr/0060-record-commit-emission-slice.md` filename). R1 doesn't apply (post-Wave-3 lane); the §Recommendation's R5 "carried-forward reading" list is correctly empty (no new expansive reading proposed; precedent reuse is direct).

---

## Important

**I-1. R2: "Decision Drivers (DD-6)" + "Recommendation — What this study commits (Histogram bucket boundaries)"** — the cite `~100–400ms per ADR-0059 §Consequence 3` is wrong on both axes. ADR-0059 §Consequence 3 is about context cancellation; the typical/expected/worst-case numbers (`~100ms` typical, `~150ms` expected, `600ms` worst-case) live in §Consequence 2, and `~100–400ms` is not in either consequence. Rewrite to cite §Consequence 2 with its actual numbers or drop the parenthetical.

**I-2. P4: "Decision Drivers (DD-3)"** — the claim "the per-cycle aggregate ... is operator-derivable from `commit_attempts × per-attempt-duration`" omits back-off sleep time. Per ADR-0059 §Clause 3's loop, per-cycle aggregate = Σ per-attempt durations + Σ back-off durations (up to `200ms` + `400ms` upper bounds at β parameters). The multiplication formula drops the back-off summand. Either spell out both summands or drop the formula and let DD-3 stand on "per-attempt is the load-bearing primitive for ADR-0059 OQ-6's calibration question" alone.

**I-3. P4: "Considered Options (Option A — emission site, failures counter row)" + "Recommendation — What this study commits (Implementation site)" + "Consequences (#4)"** — failures-counter emission "on non-nil return" includes context-cancellation (`ctx.Err()`), which ADR-0059 §Clause 5 explicitly distinguishes from warning-log + skip ("Context-cancellation returns from `commitWithRetry` without warning-logging since shutdown is operator-driven, not a failure mode"). Without the distinction, `dq_record_commit_failures_total` increments on every clean engine shutdown that catches an in-flight commit. Tighten to "on non-nil return that is not `context.Canceled` / `context.DeadlineExceeded`" so the counter tracks broker failures, not shutdown signals.

**I-4. AC-2 + P5: "Consequences (#6)"** — flipping ADR-0059 OQ-6 to `resolved-adr` because the histogram wires conflates OQ-6's literal description ("Quantitative stall-budget calibration — observed poll-batch processing time vs. retry stall", per the OQ Register row) with its prerequisite (the wiring). The OQ Register §"Scope and conventions" defines `resolved-adr` as "consumed by a subsequent ADR or amendment"; ADR-0060 enables the calibration but does not perform it. Pick one posture: (a) keep ADR-0059 OQ-6 at `open` with ADR-0060 linked in its description as the enabling slice and have ADR-0060's OQ-2 carry the still-unresolved calibration; or (b) commit the conflation with a one-line register convention note that "wiring closes the OQ; downstream calibration is a new OQ" (and amend the OQ Register §"Scope and conventions" in the same PR).

**I-5. P4: "Consequences (#8)"** — cardinality math undercounts. A Prometheus histogram emits N+1 cumulative `_bucket` series (including `+Inf`) plus `_count` and `_sum` per labelset → N+3 series per labelset. With `prometheus.DefBuckets`'s 11 explicit buckets, the histogram contributes `entity × 14` series, not `entity × 12`. Counter contributions: failures = `entity × 1`, retries = `entity × 2`. Total `entity × 17`, not `entity × 16`. Absolute numbers are small either way, but quantitative cardinality claims are reviewer-load-bearing (same shape as B3-7's round-1 cap-parameter math correction).

---

## Minor

**M-1. AC-4: "Decision Drivers (DD-4)"** — "the latter conflates retried-recovered cases with terminal failures" is imprecise; a per-attempt counter would *distinguish* attempts (not conflate them). The intended argument is that per-attempt isn't the operator-load-bearing primitive — tighten the phrasing.

**M-2. AC-2: "Considered Options (Option A — Increment / observe semantics column, histogram row)"** — cell says "histogram bucket boundaries deferred to OQ (DD-6)" but §Recommendation commits `prometheus.DefBuckets` as β. Update cell to "β commits `prometheus.DefBuckets`; re-tuning deferred to OQ-1".

**M-3. AC-5: "Decision Drivers (DD-1)"** — PromQL example `rate(...{outcome="exhausted"}[5m])` uses ellipsis where the "three names match three contract surfaces" claim needs the metric *name*. Spell out the actual metric names in the queries.

**M-4. AC-6: "Open Questions (OQ-1 vs OQ-2)"** — both OQs wait on production signal but address different surfaces. Add a one-line cross-reference: OQ-1 is histogram bucket boundaries; OQ-2 is using the histogram to calibrate ADR-0059 §Clause 2's β parameters.

**M-5. AC-2: "Context — Eligibility under ADR-0049 §(a) (Condition 2 cell)"** — the analogy "Same disposition shape as B3-3 / ADR-0053" is correct as meta-pattern but B3-3 reused ADR-0051 Clause 1, not ADR-0055. Either name both precedent chains explicitly or drop the B3-3 analogy and cite ADR-0055 alone.

---

## Disposition (round 1)

All 5 important findings applied to the study in the same revision:

- **I-1** applied — DD-6 + Recommendation §"Histogram bucket boundaries" rewritten to cite ADR-0059 §Consequence 2 with its actual numbers (`~100ms` typical, `~150ms` expected, `600ms` worst-case). Wrong "100–400ms" range and wrong §Consequence 3 cite both dropped.
- **I-2** applied — DD-3 multiplication formula dropped. Replaced with explicit back-off derivation citing ADR-0059 §Clause 3's loop shape and §Clause 5's `commit_attempts` log-field reconstruction path.
- **I-3** applied at all three sites — Considered Options Option A failures-counter row, Recommendation §"Implementation site" bullet, and Consequences #4 — each tightened to "non-nil return that is not `context.Canceled` / `context.DeadlineExceeded`" with ADR-0059 §Clause 5 shutdown-distinction cite.
- **I-4** applied via posture **(a)** — ADR-0059 OQ-6 stays at `open` in the OQ Register; ADR-0060 links as the enabling slice in OQ-6's description column; ADR-0060's OQ-2 carries the still-unresolved calibration analysis. Consequences #6 rewritten to flip two rows + extend one description. §Open Questions OQ-2 tightened to reflect the wiring-enables-calibration framing. Posture-(a) chosen because it preserves the OQ Register §"Scope and conventions" `resolved-adr` semantic ("consumed by a subsequent ADR or amendment") without amending the register's conventions in the same PR (the conflation-explicit posture (b) would have expanded R4 scope to include a register-convention amendment).
- **I-5** applied — Consequences #8 cardinality math corrected to `entity × 17` with the decomposition shown: failures `entity × 1`, retries `entity × 2`, histogram `entity × 14` (12 cumulative `_bucket` series including `+Inf`, plus `_count` and `_sum`).

5 minor findings deferred under the two-round cap per [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md). They surface in the round-1 capture above so a future reader can audit which polish was traded for closing the round.

---

## Round 1 close

Round-1 disposition applied. The study advances to operator [H] gate at post-Wave-3 session-loop step 8 (check completeness) — the §Open Questions all carry "Out-of-scope for current cycle" markers; the Metadata `Critique rounds:` bullet now records this round's disposition per ADR-0048 §"Skip" grammar. Step 9 (decision-log row flip to `resolved-study`) and step 10 (PR open) follow on operator approval.
