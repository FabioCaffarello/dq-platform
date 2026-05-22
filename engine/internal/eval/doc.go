// path: engine/internal/eval/doc.go

// Package eval implements the first real CheckEvaluator scaffolded
// by W3-P6c. The package is the engine's data-quality evaluation
// surface: given a runner.CheckSpec and a runner.TriggerRequest,
// it compiles the check into BigQuery SQL, executes the query,
// and returns the runner.Evaluation per ADR-0004 CC1.
//
// The package boundary deliberately keeps the runner free of
// BigQuery imports: the runner stays an abstraction over
// results.Store; this package owns the cloud.google.com/go/bigquery
// dependency. The exported Evaluator type satisfies
// runner.CheckEvaluator via duck typing.
//
// Phase-6c ships one check kind:
//
//   - row_count_positive — SELECT COUNT(*) FROM <source-table>,
//     ResultPass when count > 0, ResultFail when count == 0,
//     ResultError on any BigQuery failure (ADR-0004 CC1).
//
// Additional kinds (null_ratio, freshness, uniqueness, etc.) land
// additively in follow-up sessions. The dispatch switch in
// Evaluate returns ResultError for any unrecognized kind so a
// trigger carrying an unknown kind never silently passes.
//
// ADR-0001 commits `kind` as the discriminator field on each
// check; the v1 JSON schema (engine/internal/dsl/schema/v1.schema.json)
// accepts any string for kind, with the closed-enum tightening
// deferred to a future schema version. ADR-0013 §"Phase 6" places
// this scaffold inside the first-onboarded-entity end-to-end demo.
//
// New contributions proposed here, requires review:
//
//   - The row_count_positive predicate (count > 0 ⇒ pass) and its
//     EvidenceSummary field set (row_count, threshold, table_ref,
//     kind) are not directly committed by any prior ADR. The
//     mapping is intuitive but should be ratified by an ADR
//     amendment when a richer threshold model lands.
//   - The DQ_SOURCE_PROJECT / DQ_SOURCE_DATASET env var naming
//     is internal engine configuration; no ADR commits these
//     names. They follow the DQ_BIGQUERY_PROJECT /
//     DQ_BIGQUERY_DATASET pattern for symmetry.
package eval
