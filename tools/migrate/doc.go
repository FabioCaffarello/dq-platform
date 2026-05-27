// path: tools/migrate/doc.go

// Binary dq-migrate emits a migrated rule YAML between schema
// versions per ADR-0035's compatibility-state model. The
// migrate tool is operator ergonomic — the v1 escape hatch
// committed by ADR-0035 §"Per-deployment escape hatch" is
// engine pinning, not this binary; the binary's role is to
// reduce the manual edit surface for the field renames +
// structural transforms the v1 → v(N+1) deltas commit.
//
// At v1 the tool supports only the v1 → v2 migration — the
// only schema delta the platform has executed. Future
// v(N) → v(N+1) deltas land additively as new flag-pair
// values; the §"Forward pattern" section of
// `docs/dev/schema-migration.md` commits the convention.
//
// Usage:
//
//	dq-migrate -from=v1 -to=v2 \
//	  -bigquery-project=<id> -bigquery-dataset=<id> -bigquery-table=<id> \
//	  rules/<entity>.yaml
//
// The bigquery flags supply the v2 `source` descriptor's
// fields (ADR-0023). They are required for v1 → v2 because v1
// has no source block; the operator owns the substrate
// binding decision (typically lifted from
// `engine/internal/env/{local,qa,prod}.go` or the deploy
// overlay's ConfigMap).
//
// Output is the migrated YAML on stdout. Exit codes:
//
//	0  migration succeeded; stdout carries the v2 YAML
//	1  migration failed (invalid input; unsupported version pair)
//	2  operational error (I/O; YAML parse / emit)
//	64 user usage error (bad flags)
//
// The operator pipes stdout to the target file, or uses shell
// redirection. The tool does NOT write to the input file in
// place to preserve operator review before any source mutation.
package main
