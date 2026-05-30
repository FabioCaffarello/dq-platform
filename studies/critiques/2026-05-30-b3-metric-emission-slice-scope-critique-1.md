<!-- path: studies/critiques/2026-05-30-b3-metric-emission-slice-scope-critique-1.md -->

# Critique — `studies/decisions/2026-05-30-b3-metric-emission-slice-scope.md` — round 1

## Blocking findings

**None.** The study has its R6 path header, the seven required
sections in order, three considered options (≥2 per AC-3), every
recommendation grounded in a prior ADR (AC-4), no sibling-team or
external prior-art naming as justification (AC-5), every Open
Question explicitly marked "Out-of-scope for current cycle" with a
one-line reason (AC-6), and a concrete `docs/adr/0055-…` promotion
target (AC-7). The borderline ADR-0049 §(a) Condition 2 is surfaced
explicitly with a new-contribution-requires-review marker rather
than papered over.

## Important findings

- **[important] AC-5 / R5: "Recommendation → What this study does
  NOT commit" — the library candidates are named as
  `prometheus/client_golang` and "an `otel/metric` exporter
  configured with Prometheus output".** Prometheus and
  OpenTelemetry are R5-exempt commodity environments, but naming a
  specific Go library by its package path edges into prior-art
  naming as the source of a choice. Rewrite as functional
  descriptions ("the canonical Go client for the Prometheus
  exposition format" / "an OpenTelemetry metrics exporter
  configured for Prometheus output") so the choice space is
  described in our own terms.

- **[important] P5 / AC-2: "Considered Options → Option B item 5
  (`dq_bytes_scanned`)" — the line "the gauge reports zero when
  absent rather than dropping the emission" pre-commits an
  emission-side design decision that belongs to the promotion
  ADR.** A scoping study should name the gap (the field is
  undocumented per ADR-0039 OQ-3) without pre-deciding the gauge's
  behavior on absence. Move the zero-vs-drop behavior to OQ-5 (or
  open a new OQ-6) so the promotion ADR resolves it against
  working code.

- **[important] AC-2: "Recommendation → What this study commits" —
  bullet 4 lists "the cardinality posture (continues ADR-0039's
  no-numeric-ceiling deferral until the first scrape-pressure
  signal)" as something the slice's ADR must commit.** That is
  verbatim what ADR-0039 §"Cardinality posture" already commits;
  restating it as a slice-ADR commitment risks the slice ADR
  re-litigating the deferral. Tighten to "preserves ADR-0039's
  cardinality posture (no re-litigation)".

## Minor findings

- **[minor] AC-2: "Metadata → Decision-log row" — "B3-4 (tooling
  family)" is asserted as if final, but the family fit is the
  borderline reading the study itself surfaces.** Suffix with
  "(tooling family — pending operator ratification per D0 below)"
  so the metadata line matches the eligibility-check disposition.

- **[minor] P3: "Decision Drivers → DD-1" — the phrase "two
  shipped consumers (dashboard panels 4–5) read 'no data'" is
  slightly imprecise.** Panels are configuration that drive a
  query, not the consumer at runtime. Rewrite as "two shipped
  dashboard panels (4–5) render 'no data' against the metric
  inventory" so the ownership picture matches ADR-0039's
  "consumer" framing.

- **[minor] AC-2: "Context → inventory table" — column 3
  "Producing surface" is inconsistent across rows** (engine-side
  rows name a file path; scheduler-side rows name an ADR +
  abstract surface). Either rename to "Producing site" and add an
  ADR pointer to every row, or split into two columns ("Producing
  code path" / "Governed by"). The current shape is readable but
  not uniform.

- **[minor] AC-2: "Consequences → item 6" — "becomes load-bearing
  once the slice lands" is awkward.** The doc-comment is already
  load-bearing as a deferral marker; what changes is that it is
  updated. Rephrase as "is updated in the same PR that lands the
  emission slice, to reflect the implementation shape".

- **[minor] AC-6: "Open Questions → OQ-1" — the phrasing "either
  an ADR-0033 amendment authorizing engine-side proxies (with
  redefined label semantics), or a scheduler-binary instrumentation
  slice" presents two paths but does not name the test that picks
  between them.** Add one-line: "the choice is whichever surfaces
  concrete demand first per P-B3.3 (demand-driven pacing)".

## Disposition reminder

`/critique` runs in a session whose author also wrote the artifact
(ADR-0051 §Consequence 7 author-equals-reviewer circularity). The
Condition 2 D0 borderline reading is operator-ratifiable per
`CONTRIBUTING.md` Flow 5 §"Operator-side responsibilities"; this
critique surfaces the marker but does not ratify it.
