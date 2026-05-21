// path: tools/lint/doc.go
//
// Command dq-lint is the DQ Platform rules linter.
//
// The linter walks a directory tree of rule YAMLs and validates each
// against the schema mirror at rules/_schema/v<N>.schema.json. It
// enforces:
//
//   - rejection of rule YAMLs missing the top-level version field
//     (ADR-0001 C4);
//   - validation against the schema mirror at the declared version
//     (ADR-0001 C7);
//   - rejection of entity names containing the ASCII pipe character
//     (ADR-0002 input-safety) — enforced both via the schema
//     pattern and a belt-and-suspenders in-code check on schema
//     load (so a schema edit that weakens the pattern fails at
//     linter startup, not at runtime).
//
// Phase 5 extends the linter to reject entities without an
// _owners.yaml entry (ADR-0006). Phase 4+ extends the schema as
// the DSL grammar lands.
package main
