// path: tools/manifest/main.go

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// Exit codes are part of the dq-manifest CLI contract; tooling
// (CI lanes, operator wrapper scripts) keys on them. Do not
// change without an ADR.
const (
	exitOK               = 0
	exitVerificationFail = 1
	exitOperationalFail  = 2
	exitCASLost          = 3
	exitUsage            = 64 // BSD convention; reserved for argv-parse failures
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(exitUsage)
	}
	switch os.Args[1] {
	case "publish":
		os.Exit(runPublish(os.Args[2:]))
	case "-h", "--help", "help":
		printUsage()
		os.Exit(exitOK)
	default:
		fmt.Fprintf(os.Stderr, "dq-manifest: unknown subcommand %q\n\n", os.Args[1])
		printUsage()
		os.Exit(exitUsage)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `dq-manifest — publish a DQ Platform ruleset manifest per ADR-0005.

Usage:
  dq-manifest publish [flags]

Flags (publish):
  -rules <dir>                    Directory tree to walk for rule YAMLs (default: rules/)
  -schema-mirror <dir>            Schema mirror directory (default: rules/_schema/)
  -bucket <name>                  Object-store bucket (required)
  -ruleset-version <rules-vX.Y.Z> Ruleset version identifier (required)
  -engine-compatibility <range>   Engine semver range, e.g. ">=0.1.0, <1.0.0" (required)
  -linter-used <tools-lint-vX.Y.Z> Linter release identifier (required)
  -supported-schema-versions 1[,2,...]  Engine's accepted set (default: 1)
  -dry-run                        Verify + log without writing
  -storage-emulator-host <host>   Override the GCS endpoint (e.g. localhost:4443).
                                  Honored if non-empty; ignored otherwise.

Exit codes:
  0  publish OK
  1  pre-publish verification failed (content problem; operator fixes rules)
  2  operational failure (bucket missing, network, etc.)
  3  CAS precondition failed (pointer lost; operator retries)
  64 usage error
`)
}

func runPublish(args []string) int {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	var (
		rulesDir            = fs.String("rules", "rules", "rule YAML directory")
		schemaMirror        = fs.String("schema-mirror", "rules/_schema", "schema mirror directory")
		bucket              = fs.String("bucket", "", "object-store bucket (required)")
		rulesetVersion      = fs.String("ruleset-version", "", "ruleset version identifier (required)")
		engineCompatibility = fs.String("engine-compatibility", "", "engine semver range (required)")
		linterUsed          = fs.String("linter-used", "", "linter release identifier (required)")
		supportedVersions   = fs.String("supported-schema-versions", "1", "comma-separated set of supported schema versions")
		dryRun              = fs.Bool("dry-run", false, "verify + log without writing")
		storageEmulatorHost = fs.String("storage-emulator-host", "", "override the GCS endpoint (e.g. localhost:4443)")
	)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *bucket == "" || *rulesetVersion == "" || *engineCompatibility == "" || *linterUsed == "" {
		fmt.Fprintln(os.Stderr, "dq-manifest publish: -bucket, -ruleset-version, -engine-compatibility, -linter-used are required")
		return exitUsage
	}

	supported, err := parseIntList(*supportedVersions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-manifest publish: -supported-schema-versions: %v\n", err)
		return exitUsage
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx := context.Background()

	client, err := newStorageClient(ctx, *storageEmulatorHost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-manifest publish: create storage client: %v\n", err)
		return exitOperationalFail
	}
	defer client.Close()

	pub, err := New(Config{
		Store:                   NewGCSStore(client, *bucket),
		RulesetVersion:          *rulesetVersion,
		EngineCompatibility:     *engineCompatibility,
		LinterUsed:              *linterUsed,
		SchemaMirrorDir:         *schemaMirror,
		SupportedSchemaVersions: supported,
		Logger:                  logger,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-manifest publish: %v\n", err)
		return exitUsage
	}

	result, err := pub.Publish(ctx, Options{
		RulesDir: *rulesDir,
		DryRun:   *dryRun,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrVerificationFailed):
			fmt.Fprintf(os.Stderr, "dq-manifest publish: %v\n", err)
			return exitVerificationFail
		case errors.Is(err, ErrPreconditionFailed):
			fmt.Fprintf(os.Stderr, "dq-manifest publish: %v\n", err)
			return exitCASLost
		default:
			fmt.Fprintf(os.Stderr, "dq-manifest publish: %v\n", err)
			return exitOperationalFail
		}
	}
	fmt.Fprintf(os.Stderr, "dq-manifest publish: OK ruleset_version=%s manifest_hash=%s rules=%d pointer_gen=%d dry_run=%v\n",
		result.RulesetVersion, result.ManifestHash, result.RulesPublished, result.PointerGen, *dryRun)
	return exitOK
}

func parseIntList(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q: %w", p, err)
		}
		if n <= 0 {
			return nil, fmt.Errorf("supported schema version must be positive; got %d", n)
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, errors.New("supported schema versions list is empty")
	}
	return out, nil
}

// newStorageClient constructs the GCS client. When
// storageEmulatorHost is non-empty, the client is wired to the
// local emulator without authentication (commodity-emulator
// posture per ADR-0010 §3.2). Otherwise the production GCS
// endpoint is used with default authentication.
func newStorageClient(ctx context.Context, storageEmulatorHost string) (*storage.Client, error) {
	if storageEmulatorHost != "" {
		return storage.NewClient(ctx,
			option.WithoutAuthentication(),
			option.WithEndpoint("http://"+storageEmulatorHost+"/storage/v1/"),
		)
	}
	return storage.NewClient(ctx)
}
