// path: engine/cmd/dq-engine/main.go

// Command dq-engine is the DQ Platform runtime binary.
//
// Phase 4c scope: minimal wiring that demonstrates the loader,
// result-write layer, and orphan detector run together. The
// binary loads the active manifest at startup (process-exit on
// failure per ADR-0007 CC1), starts two periodic loops (loader
// refresh per ADR-0007 CC9 and orphan-run detection per
// ADR-0007 CC11), and waits for SIGTERM / SIGINT for graceful
// shutdown.
//
// The HTTP / gRPC trigger handler is deferred to a later phase.
// Without a trigger surface the binary's Runner is instantiated
// but never invoked at runtime; the runner is exercised through
// Go integration tests instead. The wiring proves the full
// stack assembles cleanly.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"dq-platform/engine/internal/loader"
	"dq-platform/engine/internal/orphan"
	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// envConfig collects the environment variables the binary reads
// at startup. Future scaffolding (B1-4 environment configuration
// model) may refine the surface; this is the minimum workable
// twelve-factor default.
type envConfig struct {
	EngineVersion         string
	GCSBucket             string
	BigQueryProject       string
	BigQueryDataset       string
	LoaderRefreshInterval time.Duration
	OrphanThreshold       time.Duration
	OrphanScanInterval    time.Duration
	LogLevel              slog.Level

	// Endpoint overrides for the local emulator. Honored when
	// non-empty; ignored in production.
	StorageEmulatorHost  string
	BigQueryEmulatorHost string
}

func main() {
	flag.Parse()
	cfg, err := readEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dq-engine: %v\n", err)
		os.Exit(2)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)
	logger.Info("dq-engine starting",
		"engine_version", cfg.EngineVersion,
		"bigquery_project", cfg.BigQueryProject,
		"bigquery_dataset", cfg.BigQueryDataset,
		"gcs_bucket", cfg.GCSBucket,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	gcsClient, err := newGCSClient(ctx, cfg)
	if err != nil {
		logger.Error("create GCS client", "error", err.Error())
		os.Exit(1)
	}
	defer gcsClient.Close()

	bqClient, err := newBQClient(ctx, cfg)
	if err != nil {
		logger.Error("create BigQuery client", "error", err.Error())
		os.Exit(1)
	}
	defer bqClient.Close()

	// Result-write layer.
	store := results.NewBigQueryStore(bqClient, cfg.BigQueryProject, cfg.BigQueryDataset, logger)
	if err := store.EnsureSchema(ctx); err != nil {
		logger.Error("ensure schema", "error", err.Error())
		os.Exit(1)
	}

	// Loader — startup-mode load per ADR-0007 CC1. Process exits
	// non-zero on any failure.
	gcsStore := loader.NewGCSStore(gcsClient, cfg.GCSBucket)
	ldr, err := loader.New(gcsStore, loader.Config{
		EngineVersion:           cfg.EngineVersion,
		SupportedSchemaVersions: []int{1},
	})
	if err != nil {
		logger.Error("new loader", "error", err.Error())
		os.Exit(1)
	}
	initial, err := ldr.Load(ctx)
	if err != nil {
		logger.Error("initial manifest load failed (ADR-0007 CC1)", "error", err.Error())
		os.Exit(1)
	}
	logger.Info("initial manifest loaded",
		"ruleset_version", initial.RulesetVersion,
		"manifest_hash", initial.Hash,
		"rules", len(initial.Rules),
	)

	current := &manifestHolder{}
	current.set(initial)

	// Orphan detector (ADR-0007 CC11). The Runner from the
	// runner package is constructed but not exercised at runtime
	// in Phase 4c; the binary holds it so Phase 6 (HTTP trigger
	// handler) can use the same wiring.
	detector, err := orphan.New(store, orphan.Config{
		EngineVersion: cfg.EngineVersion,
		Threshold:     cfg.OrphanThreshold,
		Logger:        logger,
	})
	if err != nil {
		logger.Error("new orphan detector", "error", err.Error())
		os.Exit(1)
	}

	r, err := runner.New(runner.Config{
		Store:          store,
		EngineVersion:  cfg.EngineVersion,
		RulesetVersion: initial.RulesetVersion,
		Logger:         logger,
	})
	if err != nil {
		logger.Error("new runner", "error", err.Error())
		os.Exit(1)
	}
	_ = r // Phase 4c: held but not exercised. Phase 6 wires triggers.

	var wg sync.WaitGroup
	wg.Add(2)
	go loaderRefreshLoop(ctx, &wg, logger, ldr, current, cfg.LoaderRefreshInterval)
	go orphanScanLoop(ctx, &wg, logger, detector, cfg.OrphanScanInterval)

	// Wait for signal.
	<-ctx.Done()
	logger.Info("shutdown signal received", "reason", ctx.Err())

	// Give goroutines a bounded window to finish their current
	// iteration. signal.NotifyContext cancellation already
	// propagated through; we just wait.
	shutdownDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(shutdownDone)
	}()
	select {
	case <-shutdownDone:
		logger.Info("dq-engine shut down cleanly")
	case <-time.After(10 * time.Second):
		logger.Warn("dq-engine shutdown timed out; exiting anyway")
	}
}

func readEnv() (envConfig, error) {
	cfg := envConfig{
		LoaderRefreshInterval: 30 * time.Second,
		OrphanThreshold:       time.Hour,
		OrphanScanInterval:    5 * time.Minute,
		LogLevel:              slog.LevelInfo,
		StorageEmulatorHost:   os.Getenv("STORAGE_EMULATOR_HOST"),
		BigQueryEmulatorHost:  os.Getenv("BIGQUERY_EMULATOR_HOST"),
	}

	cfg.EngineVersion = os.Getenv("DQ_ENGINE_VERSION")
	cfg.GCSBucket = os.Getenv("DQ_GCS_BUCKET")
	cfg.BigQueryProject = os.Getenv("DQ_BIGQUERY_PROJECT")
	cfg.BigQueryDataset = os.Getenv("DQ_BIGQUERY_DATASET")

	if cfg.EngineVersion == "" {
		return cfg, errors.New("DQ_ENGINE_VERSION is required")
	}
	if cfg.GCSBucket == "" {
		return cfg, errors.New("DQ_GCS_BUCKET is required")
	}
	if cfg.BigQueryProject == "" {
		return cfg, errors.New("DQ_BIGQUERY_PROJECT is required")
	}
	if cfg.BigQueryDataset == "" {
		return cfg, errors.New("DQ_BIGQUERY_DATASET is required")
	}

	if v := os.Getenv("DQ_LOADER_REFRESH_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return cfg, fmt.Errorf("DQ_LOADER_REFRESH_INTERVAL: %w", err)
		}
		cfg.LoaderRefreshInterval = d
	}
	if v := os.Getenv("DQ_ORPHAN_THRESHOLD"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return cfg, fmt.Errorf("DQ_ORPHAN_THRESHOLD: %w", err)
		}
		cfg.OrphanThreshold = d
	}
	if v := os.Getenv("DQ_ORPHAN_SCAN_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return cfg, fmt.Errorf("DQ_ORPHAN_SCAN_INTERVAL: %w", err)
		}
		cfg.OrphanScanInterval = d
	}
	if v := os.Getenv("DQ_LOG_LEVEL"); v != "" {
		switch strings.ToLower(v) {
		case "debug":
			cfg.LogLevel = slog.LevelDebug
		case "info":
			cfg.LogLevel = slog.LevelInfo
		case "warn":
			cfg.LogLevel = slog.LevelWarn
		case "error":
			cfg.LogLevel = slog.LevelError
		default:
			return cfg, fmt.Errorf("DQ_LOG_LEVEL: unknown value %q (expected debug/info/warn/error)", v)
		}
	}

	return cfg, nil
}

func newGCSClient(ctx context.Context, cfg envConfig) (*storage.Client, error) {
	if cfg.StorageEmulatorHost != "" {
		return storage.NewClient(ctx,
			option.WithoutAuthentication(),
			option.WithEndpoint("http://"+cfg.StorageEmulatorHost+"/storage/v1/"),
		)
	}
	return storage.NewClient(ctx)
}

func newBQClient(ctx context.Context, cfg envConfig) (*bigquery.Client, error) {
	if cfg.BigQueryEmulatorHost != "" {
		return bigquery.NewClient(ctx, cfg.BigQueryProject,
			option.WithoutAuthentication(),
			option.WithEndpoint("http://"+cfg.BigQueryEmulatorHost),
		)
	}
	return bigquery.NewClient(ctx, cfg.BigQueryProject)
}

// manifestHolder is the in-process atomic holder for the current
// manifest. The loader refresh loop swaps it; the runner (when
// Phase 6 wires triggers) reads it.
type manifestHolder struct{ v atomic.Pointer[loader.Manifest] }

func (h *manifestHolder) set(m *loader.Manifest) { h.v.Store(m) }
func (h *manifestHolder) get() *loader.Manifest  { return h.v.Load() }

// loaderRefreshLoop ticks on the configured interval and calls
// Loader.Refresh per ADR-0007 CC9. Refuse-swap is implicit: on
// any error, the in-memory manifest is unchanged.
func loaderRefreshLoop(ctx context.Context, wg *sync.WaitGroup, logger *slog.Logger, ldr *loader.Loader, current *manifestHolder, interval time.Duration) {
	defer wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cur := current.get()
			newManifest, swapped, err := ldr.Refresh(ctx, cur.Hash)
			if err != nil {
				logger.Warn("manifest refresh failed (refuse-swap; retaining prior manifest per ADR-0007 CC2)",
					"current_hash", cur.Hash,
					"error", err.Error(),
				)
				continue
			}
			if swapped {
				current.set(newManifest)
				logger.Info("manifest refreshed",
					"old_hash", cur.Hash,
					"new_hash", newManifest.Hash,
					"new_ruleset_version", newManifest.RulesetVersion,
				)
			}
		}
	}
}

// orphanScanLoop ticks on the configured interval and calls
// Detector.RunOnce per ADR-0007 CC11. Per-row write failures are
// returned in errs and logged; the loop continues on its cadence.
func orphanScanLoop(ctx context.Context, wg *sync.WaitGroup, logger *slog.Logger, detector *orphan.Detector, interval time.Duration) {
	defer wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			finalized, errs, err := detector.RunOnce(ctx)
			if err != nil {
				logger.Warn("orphan scan failed", "error", err.Error())
				continue
			}
			if finalized > 0 || len(errs) > 0 {
				logger.Info("orphan scan completed",
					"finalized", finalized,
					"per_row_errors", len(errs),
				)
			}
			for _, e := range errs {
				logger.Warn("orphan finalization failed", "error", e.Error())
			}
		}
	}
}
