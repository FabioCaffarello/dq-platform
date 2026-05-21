// path: tools/lint/doc.go
//
// Package lint is the DQ Platform rules linter.
//
// The package root is currently a placeholder; the linter binary
// lands in Wave 3 Phase 3. Phase 3 introduces:
//   - rejection of rule YAMLs missing the top-level version field
//     (ADR-0001 C4);
//   - validation against the schema mirror at rules/_schema/
//     (ADR-0001 C7);
//   - byte-equality CI gate orchestration (ADR-0001 C2);
//   - input-safety rejection of entity names containing the
//     ASCII pipe character (ADR-0002 input-safety).
//
// Phase 5 extends the linter to reject entities without an
// _owners.yaml entry (ADR-0006).
package lint
