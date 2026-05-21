// path: tools/manifest/doc.go

// Command dq-manifest publishes a ruleset manifest to object
// storage per ADR-0005 (manifest publication semantics). It
// walks a rules directory, verifies the contract triple per
// ADR-0001, writes the content-addressed YAML + manifest
// objects, and CAS-writes the pointer file.
//
// The CLI surface and exit-code contract are documented in
// main.go.
package main
