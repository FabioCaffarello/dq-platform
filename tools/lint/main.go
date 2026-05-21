// path: tools/lint/main.go

// Binary dq-lint validates rule YAMLs against the rules schema mirror.
// See ADR-0001 (compatibility contract) and ADR-0002 (input-safety).
package main

import (
	"flag"
	"fmt"
	"os"
)

// Exit codes are part of the linter's public CLI contract; tooling
// (CI, make targets, IDEs) keys on them. Do not change without an ADR.
const (
	exitOK              = 0
	exitValidationError = 1
	exitOperationalError = 2
)

func main() {
	var (
		schemaPath = flag.String("schema", "rules/_schema/v1.schema.json",
			"path to the schema file to validate against (must be the rules mirror, not the engine source)")
		rulesDir = flag.String("rules", "rules",
			"directory tree to walk for *.yaml files; the _schema/ subdirectory is excluded automatically")
		verbose = flag.Bool("v", false, "print each file as it is processed")
	)
	flag.Parse()

	schema, err := LoadSchema(*schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-lint: failed to load schema %q: %v\n", *schemaPath, err)
		os.Exit(exitOperationalError)
	}

	results, filesProcessed, err := ValidateRulesDir(schema, *rulesDir, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-lint: %v\n", err)
		os.Exit(exitOperationalError)
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
