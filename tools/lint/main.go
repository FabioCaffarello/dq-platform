// path: tools/lint/main.go

// Binary dq-lint validates rule YAMLs against the rules schema mirror.
// See ADR-0001 (compatibility contract), ADR-0002 (input-safety),
// ADR-0021 (mode-as-primitive), and ADR-0022 (kind catalog).
package main

import (
	"context"
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
		ownersSchemaV3Path = flag.String("owners-schema-v3", "rules/_schema/_owners.v3.schema.json",
			"path to the v3 _owners.yaml JSON Schema (must be the rules mirror); empty disables v3 (B2-25 / ADR-0046)")
		catalogPath = flag.String("catalog", "rules/_schema/catalog.v1.yaml",
			"path to the v1 kind catalog (must be the rules mirror); consumed only by v2 cross-checks")
		ownersPath = flag.String("owners", "rules/_owners.yaml",
			"path to the _owners.yaml; the file may be missing, in which case ownerless rules are still rejected per ADR-0006 CC9")
		rulesDir = flag.String("rules", "rules",
			"directory tree to walk for *.yaml files; the _schema/ subdirectory is excluded automatically")
		codeownersPath = flag.String("codeowners", ".github/CODEOWNERS",
			"path to .github/CODEOWNERS for the owner-group cross-check per ADR-0037; empty disables the check")
		noDeprecationWarnings = flag.Bool("no-deprecation-warnings", false,
			"suppress deprecation warnings for rules declaring a deprecated schema version per ADR-0035 (B2-21)")
		checkChannelReachability = flag.Bool("check-channel-reachability", false,
			"opt-in: walk every _owners.yaml channel reference and probe per-substrate reachability (Slack, email, PagerDuty) per ADR-0047. Credentials via DQ_LINT_SLACK_TOKEN / DQ_LINT_PAGERDUTY_KEY env vars (email uses DNS only). Outcomes are warnings — never influence exit code.")
		verbose = flag.Bool("v", false, "print each file as it is processed")
	)
	flag.Parse()

	schemaSet, err := LoadSchemaSet(*schemaPath, *schemaV2Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-lint: failed to load rule schemas: %v\n", err)
		os.Exit(exitOperationalError)
	}

	ownersSchemaSet, err := LoadOwnersSchemaSet(*ownersSchemaPath, *ownersSchemaV2Path, *ownersSchemaV3Path)
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

	codeowners, err := LoadCodeOwners(*codeownersPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-lint: %v\n", err)
		os.Exit(exitOperationalError)
	}
	if errs := CheckOwnersGroupMembership(owners, codeowners); len(errs) > 0 {
		results[*ownersPath] = append(results[*ownersPath], errs...)
	}

	// ADR-0035 §"Migration support level" deprecation warnings.
	// Emitted independently of validation errors; the lint exit
	// code reflects errors only (warnings are informational).
	if !*noDeprecationWarnings {
		warnings, werr := CheckDeprecatedSchemaVersions(*rulesDir)
		if werr != nil {
			fmt.Fprintf(os.Stderr, "dq-lint: deprecation-warning walk: %v\n", werr)
			os.Exit(exitOperationalError)
		}
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "%s: DEPRECATED: %s\n", w.Path, w.Message)
		}
	}

	// ADR-0047 / B2-34 reachability check — opt-in via flag. The
	// per-channel adapter outcomes are warnings; the lint exit
	// code is unaffected. Operators reviewing a PR see the
	// REACHABILITY: lines and decide whether to act.
	if *checkChannelReachability {
		ctx := context.Background()
		registry := NewDefaultRegistry()
		reachResults := CheckChannelReachability(ctx, owners, registry)
		for _, r := range reachResults {
			fmt.Fprintf(os.Stderr,
				"%s: REACHABILITY: entity=%q category=%q channel=%q status=%s — %s\n",
				*ownersPath, r.Entity, r.Category, r.Channel, r.Status, r.Reason)
		}
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
