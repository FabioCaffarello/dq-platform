<!-- path: studies/critiques/2026-05-31-b3-7-record-runner-commit-retry-critique-1.md -->

# Critique — B3-7 record-runner commit retry (round 1)

- Target: [`studies/decisions/2026-05-31-b3-7-record-runner-commit-retry.md`](../decisions/2026-05-31-b3-7-record-runner-commit-retry.md)
- Round: 1
- Date: 2026-05-31
- Result: **0 blocking / 4 important / 5 minor**

---

## Findings

**No `blocking` findings.** The decision shape (exponential
back-off with jitter, bounded retry budget, ADR-0058 §Clause 2
terminal preserved) is sound; eligibility analysis with the
Condition 1 borderline reading is correctly disposed per R5 +
adr-writing A7. The parameter-math is internally inconsistent
and load-bearing for DD-2, and two R5-spirit phrasings appeal
to unnamed external authority — correctable without reshaping
the decision.

---

### Important

- **[important] P4 / AC-2: "Recommendation → Option C parameter
  math" — the `1.6s` worst-case bound is incorrect at the
  chosen β parameters.** With `base = 100ms`, `max_attempts =
  3`, and back-offs computed as `random_uniform(0, base ×
  2^attempt)`, the back-offs after failures 1 and 2 are
  `random(0, 200ms)` and `random(0, 400ms)` — worst case
  `600ms`, expected `~150ms`. The `cap = 800ms` is **dead
  code** at these parameters: the cap only activates when
  `base × 2^attempt > cap`, which requires `attempt ≥
  log2(800/100) = 3` — i.e., the 4th back-off, which never
  fires under `max_attempts = 3`. Pick one of three fixes:
  (a) correct the math throughout; (b) adjust parameters so
  the cap is load-bearing (e.g., `max_attempts = 5`);
  (c) remove the cap and document the worst-case as `sum(base
  × 2^attempt for attempt in 1..N-1)`. **Option (c)
  recommended** — cleanest structural fix.

- **[important] R5 / AC-5: "Considered Options → Option C →
  Strengths" — "a known shape for retry policies in the wider
  distributed-systems literature" implicitly appeals to
  unnamed external authority.** R5 requires the repo to "read
  as if it were the only source of these ideas." The phrase
  gestures at external prior art without naming the source —
  exactly the pattern the rule prohibits in spirit even when
  no specific name appears. Delete the clause; DD-3 already
  defends the choice (thundering-herd avoidance) on its own
  merits.

- **[important] R5 / AC-5: "Recommendation → DD-3" — "the
  algorithm is well-known and doesn't require a citation to
  any specific implementation"** — same R5-spirit issue.
  "Well-known" appeals to unstated external consensus. Defend
  the algorithm on its own merits in this repo's terms
  (jitter de-synchronizes concurrent retries against a
  recovering broker — that *is* the defense; the "well-known"
  hedge adds nothing).

- **[important] R2 / AC-2: "Decision Drivers → DD-2" +
  "Consequences §3" — the "typical poll-batch processing is
  10–100ms" and "1.6s is 16–160× that" claims are asserted
  without grounding.** No prior decision, measurement, or
  benchmark is cited; the numbers are author intuition. R2
  prohibits fabricating consensus that doesn't exist. Either
  (a) ground the claim, (b) weaken to qualitative
  ("expected to be small relative to the retry budget"), or
  (c) move the quantification to an OQ pending operational
  signal. Coupled to the math finding — if worst case is
  `600ms` not `1.6s`, DD-2's multiplier was wrong on both
  sides.

### Minor

- **[minor] R5: "Considered Options → Option C" — "full
  jitter" is industry terminology originating in a specific
  external source.** Has become broadly used; not a literal
  R5 violation. Cleaner shape: describe as "uniform-random
  back-off whose upper bound grows exponentially per
  attempt." Optional.

- **[minor] R5: "Open Questions → OQ-2" — names
  `kerr.IsRetriable from franz-go` as a specific external
  API.** Generalize to "the Kafka client library may expose
  a retry-classification predicate." Parallel to the
  franz-go-mention minor in the B3-6 critique.

- **[minor] AC-2 / R8: "Consequences → §8" — "S2 keeps its
  rule wording" is imprecise.** S2's rule wording is *not*
  unchanged — the retry envelope grafts onto the
  commit-after-dispatch posture, extending the rule's scope.
  Tighten to "S2's commit-after-dispatch rule wording stays
  as the spine; a retry-envelope paragraph is appended with
  new `record_runner.go` citations."

- **[minor] AC-2: "Consequences → §7" — "the engine already
  builds on a Go version that includes it" is unverified.**
  Either confirm against `engine/go.mod` at implementation
  time or weaken to "stdlib in Go 1.22+; the implementation
  slice verifies the engine's Go version is ≥ 1.22 before
  importing `math/rand/v2`."

- **[minor] AC-2: "Open Questions → OQ-4" — "subsequent
  retries never fire" is imprecise in mechanism.** They don't
  fire because the parent context's deadline already expired
  during the hang; tighten to "subsequent retries are
  pre-empted because the parent context error short-circuits
  the retry loop."

---

## Acceptance criteria roll-up

| AC | Pass? | Note |
|---|---|---|
| AC-1 | yes | Path-header present. |
| AC-2 | partial | One `[important]` (math) + two `[minor]` on §7 / §8 / OQ-4. |
| AC-3 | yes | Three options. |
| AC-4 | yes | Grounded in DDs + ADR-0058. R5-marker present. |
| AC-5 | partial | Two `[important]` R5-spirit issues + two `[minor]` R5 surface mentions. |
| AC-6 | yes | All 5 OQs marked out-of-scope. |
| AC-7 | yes | Promotion target points at `docs/adr/0059-record-runner-commit-retry.md`. |
| AC-8 | in-progress | This is round 1. |
| AC-9 | n/a | Revision applies findings. |
| AC-10 | pending | Decision-log update is step 9 of the loop. |

## Summary

**0 blocking / 4 important / 5 minor.** Recommended disposition:
apply all 4 important findings; apply the two AC-2 wording
minors (§7, §8) — cheap. Defer the `kerr.IsRetriable` minor and
the "full jitter" terminology minor under the two-round cap.
Math fix: **Option (c) — remove the cap** since dead code at the
chosen parameters; simpler is more defensible than reverse-
engineering parameters to make a cap matter. After revision, the
study is ready to advance to `resolved-study` and promote to
ADR-0059.
