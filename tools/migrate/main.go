// path: tools/migrate/main.go

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// Exit codes form part of the dq-migrate CLI public contract.
// They mirror the convention used by sibling tools (dq-lint,
// dq-manifest, dq-dryrun): 0 success, 1 validation failure
// (migration rejected), 2 operational failure (I/O), 64 user
// usage error (bad flags). Scripts and CI lanes key on these
// codes.
const (
	exitOK               = 0
	exitMigrationFailed  = 1
	exitOperationalError = 2
	exitUsage            = 64
)

func main() {
	var (
		fromVersion = flag.String("from", "v1", "input schema version (v1 only supported at v1 of this binary)")
		toVersion   = flag.String("to", "v2", "output schema version (v2 only supported at v1 of this binary)")
		bqProject   = flag.String("bigquery-project", "",
			"v2 source.project_id value (required for v1→v2 — v1 has no source block; operator owns the substrate binding per ADR-0023)")
		bqDataset = flag.String("bigquery-dataset", "",
			"v2 source.dataset_id value (required for v1→v2)")
		bqTable = flag.String("bigquery-table", "",
			"v2 source.table_id value (required for v1→v2)")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `dq-migrate — rule YAML schema-version migrator (ADR-0035 / B2-23)

Usage:
  dq-migrate -from=v1 -to=v2 \
    -bigquery-project=<id> -bigquery-dataset=<id> -bigquery-table=<id> \
    <rule-path>

Reads a v1 rule YAML from <rule-path>, emits the v2 equivalent
on stdout. The bigquery flags supply the v2 source descriptor
(ADR-0023) that v1 had no place for; the operator lifts the
values from engine/internal/env/{local,qa,prod}.go or the
deploy overlay's ConfigMap.

Use shell redirection ("> new-path") to write the result;
dq-migrate does NOT overwrite the input file in place.

Exit codes:
  0  migration succeeded; stdout carries the v2 YAML
  1  migration failed (invalid input; unsupported version pair)
  2  operational error (I/O; YAML parse / emit)
  64 user usage error (bad flags)

Flags:
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *fromVersion != "v1" || *toVersion != "v2" {
		fmt.Fprintf(os.Stderr,
			"dq-migrate: unsupported version pair -from=%q -to=%q (only v1→v2 is implemented at this binary's v1; future v(N)→v(N+1) deltas land additively)\n",
			*fromVersion, *toVersion)
		os.Exit(exitUsage)
	}

	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "dq-migrate: expected exactly one positional argument (rule path); got %d\n", len(args))
		flag.Usage()
		os.Exit(exitUsage)
	}
	rulePath := args[0]

	src := BigQuerySource{
		ProjectID: *bqProject,
		DatasetID: *bqDataset,
		TableID:   *bqTable,
	}
	if err := src.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "dq-migrate: %v (v1→v2 requires -bigquery-project / -bigquery-dataset / -bigquery-table)\n", err)
		os.Exit(exitUsage)
	}

	raw, err := os.ReadFile(rulePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-migrate: read %s: %v\n", rulePath, err)
		os.Exit(exitOperationalError)
	}

	out, err := V1ToV2(raw, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-migrate: migration failed: %v\n", err)
		os.Exit(exitMigrationFailed)
	}

	if _, err := io.Copy(os.Stdout, &writerFrom{data: out}); err != nil {
		fmt.Fprintf(os.Stderr, "dq-migrate: write stdout: %v\n", err)
		os.Exit(exitOperationalError)
	}
	os.Exit(exitOK)
}

// writerFrom adapts a []byte to io.Reader without pulling in
// the bytes package (keeps the binary's import surface tight).
type writerFrom struct {
	data []byte
	off  int
}

func (w *writerFrom) Read(p []byte) (int, error) {
	if w.off >= len(w.data) {
		return 0, io.EOF
	}
	n := copy(p, w.data[w.off:])
	w.off += n
	return n, nil
}
