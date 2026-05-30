<!-- path: studies/critiques/2026-05-30-adr-0056-panel-5-lighting-slice-critique-1.md -->

# Critique — `docs/adr/0056-panel-5-lighting-slice.md` — round 1

## Blocking findings

**None.** R6 path header present. A1 four-section structure
(Context / Decision / Consequences / Notes). A2 metadata correct
(Status `accepted`, ISO date). A7 new-contribution marker placed
in §Notes for the Condition 1 + 3 carry-forward. R5 commodity
exemptions (Prometheus, BigQuery, etc.) correctly applied. R8
forward-only — no back-links into `studies/`. Decision is five
clauses; Consequences twelve; Notes four paragraphs. The
§Decision is internally coherent against the ratified D0; the
additive-`source`-label mechanism is named as the load-bearing
mechanism in three places (Clause 2, Notes "Condition 1 + 3 D0
carry-forward", Consequence 3).

## Important findings

- **[important] AC-2: "Consequences → item 6" — arithmetic error
  in the per-package metric count.** The text claims *"RunnerMetrics
  + LoaderMetrics + SchedulerProxyMetrics together cover six
  (runner) + one (loader) + four (scheduler-proxy) = eleven
  metric series, matching the eight ADR-0039 inventory metrics
  plus the three constant-zero engine-non-derivable label combos."*
  The actual counts are: RunnerMetrics has **5** metric families
  (RunsTotal, ChecksEvaluatedTotal, RunDurationSeconds,
  CheckDurationSeconds, BytesScanned — verified against
  `engine/internal/metrics/registry.go`); LoaderMetrics has **1**
  (RefreshFailuresTotal); SchedulerProxyMetrics has **2** metric
  families (QueueDepth, SchedulerTriggersManaged) which expand to
  **4** label combinations. The "six (runner)" is wrong — should
  be "five". The sum "= eleven" conflates families and series.
  Pick one framing: either *"5 + 1 + 2 = 8 metric families,
  matching ADR-0039's 8-metric inventory; the SchedulerProxy 2
  families expand to 4 label combinations"* OR *"5 runner-side
  series + 1 loader-side series + 4 scheduler-proxy series = 10
  series"*. The 11 doesn't reconcile either way; a reviewer who
  walks the math gets stuck.

## Minor findings

- **[minor] AC-2: "Decision → Clause 5" — startup-time
  `setZeroes()` not committed in the ADR.** The implementation
  calls `setZeroes()` once at loop entry (before the for-select)
  so the three constant-zero gauges are present on the first
  scrape regardless of the first tick's timing. Clause 5's
  per-tick description is correct but silent on this startup
  behavior. Add a one-line note: *"The three constant-zero
  series are also set once at loop entry so the first Prometheus
  scrape always finds them present, independent of first-tick
  timing."* Reviewers reading just the ADR can otherwise infer
  (incorrectly) that the first scrape between engine boot and
  first tick would render `"no data"` for the three.

- **[minor] AC-2: "Consequences → item 11" — "ADR-0039 OQ-1" is
  the wrong attribution.** The "panel 5 lighting" deferral
  originated as **B3-4 OQ-1** (in the originating study
  `studies/decisions/2026-05-30-b3-metric-emission-slice-scope.md`),
  then was carried forward inside **ADR-0055 §OQ-1** (the
  promotion ADR). ADR-0039 has its own OQ-1 / OQ-2 / OQ-3 on
  different topics (evidence-summary inventory, cardinality
  ceiling, etc.). Rephrase as *"B3-4 OQ-1 (carried forward in
  ADR-0055 §OQ-1) is resolved."* Avoids cross-citing the wrong
  ADR's OQ namespace.

- **[minor] AC-2: "Consequences → item 5" — drop the "and the
  orphan-test mock if present" hedge.** `engine/internal/orphan`
  defines its own narrower `Scanner` interface (only
  `ListRunningOlderThan` + `WriteExecutionRow`), and
  `orphan_test.go`'s `mockScanner` satisfies that interface, NOT
  the full `Store`. The interface extension does not require any
  change to `mockScanner`. Drop the conditional clause to state
  the fact cleanly: *"All three Store implementations
  (BigQueryStore, results_test.go mockStore, runner_test.go
  inMemStore) implement the new method; orphan's narrower
  Scanner interface is unaffected."*

- **[minor] AC-2: "Decision → Clause 5 → cadence trade-off
  paragraph" — "stale-by-up-to-15s values" is loose.** The
  actual staleness window is *"up to one tick interval (15s)
  between successive scheduler-proxy ticks"*, not a generic
  "15s". A scrape immediately after a tick reads fresh values; a
  scrape immediately before reads near-15s-stale. The current
  wording reads as if all scrapes are 15s stale, which
  understates the cadence quality. Tighten to *"stale by up to
  one tick interval (15s) in the worst case (a scrape that
  lands just before the next tick)"*.

- **[minor] AC-2: "Notes → Critique rounds" — forward-looking
  phrasing will be stale after this critique applies.** The text
  claims *"This ADR's Decision survived one /critique round
  before promotion (round-1 disposition recorded in the PR
  body's Critique result table)"* — at write-time the critique
  had not happened. Update post-application to record the
  actual round-1 disposition explicitly (e.g., *"0 blocking / 1
  important / 5 minor; 1 important applied"*) so the Notes
  block is self-contained and doesn't depend on the PR body to
  be load-bearing.

## Disposition reminder

`/critique` runs in a session whose author also wrote the ADR;
the author-equals-reviewer circularity (ADR-0051 §Consequence 7)
applies. The Condition 1 + 3 D0 was operator-ratified mid-PR-#113
and carried into ADR-0056 §Notes; this critique does not
re-litigate the ratification, it only checks whether the ADR
text is internally consistent against the ratified reading and
against the implementation slice that landed in the same PR.
The implementation surface itself was self-verified against
AC-W3-3 + AC-W3-7 per ADR-0052 §6.4 row 6 + ADR-0048 §"Skip"
path, with `make lint`, `make test-engine`, `make demo-p6`, and
the standalone `/metrics` + Grafana proxy smoke covering the
load-bearing assertions.
