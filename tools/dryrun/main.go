// path: tools/dryrun/main.go

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
)

// Exit codes follow the same contract as the other tools/* CLIs:
// 0 OK, 1 verification (cost ceiling exceeded, compile failure),
// 2 operational, 64 usage.
const (
	exitOK               = 0
	exitVerificationFail = 1
	exitOperationalFail  = 2
	exitUsage            = 64
)

func main() {
	var (
		rulesDir         = flag.String("rules", "rules", "Directory tree to walk for rule YAMLs")
		bqProject        = flag.String("bigquery-project", "", "BigQuery project for the dry-run client (required)")
		bqEmulatorHost   = flag.String("bigquery-emulator-host", "", "Local emulator host (e.g., localhost:9050); honored if non-empty; otherwise real GCP credentials are used")
		maxBytesScanned  = flag.Int64("max-bytes-scanned", 0, "Per-rule bytes-scanned ceiling; 0 disables enforcement (still reports estimates)")
		verbose          = flag.Bool("v", false, "Print per-rule estimates + skips")
	)
	flag.Parse()
	if *bqProject == "" {
		fmt.Fprintln(os.Stderr, "dq-dryrun: -bigquery-project is required")
		os.Exit(exitUsage)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx := context.Background()

	client, err := newBQClient(ctx, *bqProject, *bqEmulatorHost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-dryrun: create BigQuery client: %v\n", err)
		os.Exit(exitOperationalFail)
	}
	defer client.Close()

	r, err := New(Config{
		RulesDir:              *rulesDir,
		MaxBytesScannedPerRun: *maxBytesScanned,
		BigQueryClient:        client,
		Logger:                logger,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-dryrun: %v\n", err)
		os.Exit(exitUsage)
	}

	rep, runErr := r.Run(ctx)

	// Always print the report — even on error, operators want
	// to see what was estimated up to the failure point.
	printReport(rep, *verbose)

	if runErr != nil {
		switch {
		case errors.Is(runErr, ErrCostCeilingExceeded):
			fmt.Fprintf(os.Stderr, "dq-dryrun: %v\n", runErr)
			os.Exit(exitVerificationFail)
		case errors.Is(runErr, ErrCompileFailed):
			fmt.Fprintf(os.Stderr, "dq-dryrun: %v\n", runErr)
			os.Exit(exitVerificationFail)
		default:
			fmt.Fprintf(os.Stderr, "dq-dryrun: %v\n", runErr)
			os.Exit(exitOperationalFail)
		}
	}
	os.Exit(exitOK)
}

// newBQClient constructs the BigQuery client honoring the
// emulator-host override pattern used by the other engine
// binaries.
func newBQClient(ctx context.Context, project, emulatorHost string) (*bigquery.Client, error) {
	if emulatorHost != "" {
		return bigquery.NewClient(ctx, project,
			option.WithoutAuthentication(),
			option.WithEndpoint("http://"+emulatorHost),
		)
	}
	return bigquery.NewClient(ctx, project)
}

// printReport emits a human-readable summary of one Runner.Run
// result. Skipped rules are listed only with -v to keep clean
// output uncluttered.
func printReport(rep *Report, verbose bool) {
	for _, e := range rep.Estimates {
		fmt.Fprintf(os.Stderr, "→ %s entity=%q check=%q kind=%q bytes=%d\n",
			e.RulePath, e.Entity, e.CheckID, e.Kind, e.TotalBytesProcessed)
	}
	if verbose {
		for _, s := range rep.Skipped {
			fmt.Fprintf(os.Stderr, "↷ %s entity=%q skipped: %s\n", s.RulePath, s.Entity, s.Reason)
		}
	}
	fmt.Fprintf(os.Stderr, "dq-dryrun: %d estimates, %d skipped, total_bytes=%d\n",
		len(rep.Estimates), len(rep.Skipped), rep.TotalBytesProcessed)
}
