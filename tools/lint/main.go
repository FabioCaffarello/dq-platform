// path: tools/lint/main.go

// Binary dq-lint validates rule YAMLs against the rules schema mirror.
// See ADR-0001 (compatibility contract), ADR-0002 (input-safety),
// ADR-0021 (mode-as-primitive), and ADR-0022 (kind catalog).
package main

import (
	"flag"
	"fmt"
	"os"
)

// Exit codes are part of the linter's public CLI contract; tooling
// (CI, make targets, IDEs) keys on them. Do not change without an ADR.
const (
	exitOK               = 0
	exitValidationError  = 1
	exitOperationalError = 2
)

func main() {
	var (
		schemaPath = flag.String("schema", "rules/_schema/v1.schema.json",
			"path to the v1 rule schema (must be the rules mirror, not the engine source)")
		schemaV2Path = flag.String("schema-v2", "rules/_schema/v2.schema.json",
			"path to the v2 rule schema (must be the rules mirror); empty disables v2")
		ownersSchemaPath = flag.String("owners-schema", "rules/_schema/_owners.v1.schema.json",
			"path to the v1 _owners.yaml JSON Schema (must be the rules mirror)")
		ownersSchemaV2Path = flag.String("owners-schema-v2", "rules/_schema/_owners.v2.schema.json",
			"path to the v2 _owners.yaml JSON Schema (must be the rules mirror); empty disables v2")
		catalogPath = flag.String("catalog", "rules/_schema/catalog.v1.yaml",
			"path to the v1 kind catalog (must be the rules mirror); consumed only by v2 cross-checks")
		ownersPath = flag.String("owners", "rules/_owners.yaml",
			"path to the _owners.yaml; the file may be missing, in which case ownerless rules are still rejected per ADR-0006 CC9")
		rulesDir = flag.String("rules", "rules",
			"directory tree to walk for *.yaml files; the _schema/ subdirectory is excluded automatically")
		verbose = flag.Bool("v", false, "print each file as it is processed")
	)
	flag.Parse()

	schemaSet, err := LoadSchemaSet(*schemaPath, *schemaV2Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-lint: failed to load rule schemas: %v\n", err)
		os.Exit(exitOperationalError)
	}

	ownersSchemaSet, err := LoadOwnersSchemaSet(*ownersSchemaPath, *ownersSchemaV2Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-lint: failed to load owners schemas: %v\n", err)
		os.Exit(exitOperationalError)
	}

	catalog, err := LoadCatalog(*catalogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-lint: failed to load catalog %q: %v\n", *catalogPath, err)
		os.Exit(exitOperationalError)
	}

	owners, ownersErrs, err := LoadOwners(ownersSchemaSet, *ownersPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-lint: %v\n", err)
		os.Exit(exitOperationalError)
	}

	results, filesProcessed, err := ValidateRulesDir(schemaSet, catalog, owners, *rulesDir, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-lint: %v\n", err)
		os.Exit(exitOperationalError)
	}

	if len(ownersErrs) > 0 {
		// Schema-validation errors on _owners.yaml itself.
		results[*ownersPath] = append(results[*ownersPath], ownersErrs...)
	}

	ownersCheckResults, err := CheckRulesHaveOwners(owners, *rulesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-lint: %v\n", err)
		os.Exit(exitOperationalError)
	}
	for path, errs := range ownersCheckResults {
		results[path] = append(results[path], errs...)
	}

	if len(results) == 0 {
		if *verbose {
			fmt.Fprintf(os.Stderr, "dq-lint: no validation errors (%d files OK)\n", filesProcessed)
		}
		os.Exit(exitOK)
	}

	for path, errs := range results {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "%s: %s\n", path, e)
		}
	}
	os.Exit(exitValidationError)
}
