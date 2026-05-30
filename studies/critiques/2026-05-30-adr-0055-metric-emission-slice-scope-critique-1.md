<!-- path: studies/critiques/2026-05-30-adr-0055-metric-emission-slice-scope-critique-1.md -->

# Critique — `docs/adr/0055-metric-emission-slice-scope.md` — round 1

## Blocking findings

**None.** R6 path header present
(`<!-- path: docs/adr/0055-metric-emission-slice-scope.md -->`).
R8 forward-only (no `studies/` back-links). A1 four-section
structure (Context / Decision / Consequences / Notes). A2
metadata (Status `accepted`, ISO-8601 Date). A7
new-contribution markers correctly placed (Clause 5 + Notes
for the Condition 2 carry-forward). Decision is six clauses;
Consequences twelve numbered items; Notes four paragraphs.
R5 commodity exemptions correctly applied (Prometheus,
OpenTelemetry, BigQuery, Kafka, etc.).

## Important findings

- **[important] P5 / AC-2: "Decision → Clause 5" — the
  `error_class` classifier relies on the loader's
  `fmt.Errorf` prefix strings (`"fetch manifest body "`,
  `"parse manifest JSON: "`) rather than sentinel errors.**
  The ADR commits the five enum values but is silent on what
  binds them to the loader's error returns. A future
  refactor of `loader.go`'s error messages would silently
  drift the metric label without breaking any test or
  contract surface visible to reviewers. Either (a) add a
  Consequence committing the loader's error-message
  prefixes as load-bearing for the classifier (operators
  reading ADR-0055 would then know that "innocuous wording
  fixes" in `loader.go` can break the metric), or (b)
  commit a sentinel-error path: export `ErrBodyFetch` and
  `ErrParseError` alongside the existing `ErrHashMismatch`,
  switch `classifyFetchAndVerifyError` to `errors.Is`, and
  note the sentinel choice in Clause 5. Option (b) is the
  more durable one; option (a) is the smaller follow-on.
  Either resolves the silent-drift hazard.

- **[important] AC-W3-10 / R5: "Decision → Clause 1" — the
  rejection rationale for the OpenTelemetry-exporter path
  uses "YAGNI" as the named principle.** YAGNI is an
  externally-coined acronym from the Extreme Programming
  literature; naming it as the justification ("rejected …
  as YAGNI SDK indirection") imports the principle by its
  external label rather than describing the trade-off in
  this project's own terms. Rewrite as something like
  "speculative against the current scope" or "an SDK
  indirection layer with no concrete demand from this
  scope's consumers". The substance survives the rewrite;
  the import goes away. (This finding is consistent with
  how round-1 of the originating study handled library
  candidates as functional descriptions rather than by
  their canonical names.)

## Minor findings

- **[minor] AC-2: "Consequences → item 6" — "replaces the
  'metric emission deferred' doc-comment" is past-tense
  aspirational** but the slice has already applied the edit.
  Restate as "the metric emission deferred doc-comment in
  `engine/internal/runner/runner.go` package doc is
  replaced by a pointer to this ADR and to the metrics
  package" so the consequence reads as present-tense fact,
  matching the surrounding items.

- **[minor] AC-2: "Notes → No ADR-0033 reopening" — the
  paragraph is defensive and is already covered by Clause 4
  ("the two scheduler-side metrics are external") and
  Consequence 4 (panel 5 stays dark).** Either remove the
  Notes paragraph or fold its load-bearing sentence
  ("ADR-0033 is preserved verbatim; no engine-side proxy
  for scheduler-tracked state is authorized") into
  Consequence 4. Avoids triplication.

- **[minor] AC-2: "Notes → Critique rounds" — the paragraph
  claims "This ADR's Decision survived one /critique round
  before promotion" but at the moment of writing, that
  round was still pending.** Update post-critique-application
  to either name the round-1 disposition explicitly (`0
  blocking / 2 important / 3 minor; 2 important applied`)
  or leave the count blank if dispositions are recorded
  only in the PR body. The current wording implies a closed
  round that did not yet exist when the prose was written.

- **[minor] AC-2: "Decision → Clause 5 → Loader.Load
  (startup-mode) does not emit" — the rationale "the metric
  would never be scraped before the process dies" is
  correct but slightly editorial.** The structural rationale
  is cleaner: startup failures are unconditionally fatal
  per ADR-0007 §1, so a startup-emission would not survive
  long enough to be a useful operational signal — the
  existing log+exit channels carry the signal instead.
  Tightens the sentence and grounds it in ADR-0007 §1
  rather than scrape-cadence.

- **[minor] A6: "Decision → Clause 4 → Emission table —
  `dq_bytes_scanned` description references "ADR-0039 OQ-3"
  inline** which is a B-level marker, not an ADR-section
  marker. Either restate as "the sub-field is undocumented
  in ADR-0039 §Consequence #12 (third deferred item)" or
  accept the OQ-NN form as a recognized cross-reference;
  either way, A6's "Section precision" form (`ADR-NNNN
  §Section`) is the prevailing convention this ADR
  otherwise follows.

## Disposition reminder

`/critique` runs in a session whose author also wrote the ADR;
the author-equals-reviewer circularity (ADR-0051 §Consequence
7) applies. The Condition 2 D0 ratification is already
operator-emitted in the prior round-2 trailer (PR #111); this
round's findings are author-emitted and represent the agent's
own self-review of the new ADR text. The operator decides
which findings rise to addressed-in-revision vs deferred-as-
minor.
