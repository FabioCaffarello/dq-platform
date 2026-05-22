// path: engine/internal/dsl/spec/doc.go

// Package spec parses rule YAML bodies into Go types matching the
// v1 schema committed by ADR-0001. The canonical JSON schema lives
// at engine/internal/dsl/schema/v1.schema.json and is mirrored to
// rules/_schema/v1.schema.json by the byte-equality CI gate.
//
// This package is the runtime YAML-parsing surface scaffolded by
// W3-P6d to bridge the manifest publisher's by-hash YAML storage
// (W3-P6a) and the HTTP trigger handler's check resolution
// (W3-P4e). The handler reads a YAML body from the object store
// via the engine's resolver closure, then calls spec.Parse to
// translate it into a structured RuleSpec; the closure converts
// the per-check entries to runner.CheckSpec values the runner
// dispatches to the evaluator (W3-P6c).
//
// Strictness contract:
//
//   - The parser uses yaml.v3's KnownFields(true) so unknown YAML
//     keys reject — matching the schema's additionalProperties:false
//     constraint.
//   - version must be the integer literal 1; any other value
//     rejects per ADR-0001 (per-rule version declaration is
//     mandatory).
//   - entity and per-check check_id must match
//     ^[A-Za-z0-9_.-]+$, max 200 bytes, free of the ASCII pipe
//     per ADR-0002 §2 input safety.
//   - kind is a non-empty string (v1 schema is permissive; the
//     evaluator dispatches on kind and returns ResultError for
//     unrecognized kinds per ADR-0004 CC1).
//
// The parser is intentionally minimal: it does not validate
// kind values against any whitelist (that is the evaluator's
// responsibility, not the parser's). It does not load the
// _owners.yaml file (the linter enforces owner coverage per
// ADR-0006 CC9 at lint time; the runtime engine trusts the
// linted ruleset).
package spec
