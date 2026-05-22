// path: engine/cmd/dq-engine/main.go

// Command dq-engine is the DQ Platform runtime binary.
//
// The binary loads the active manifest at startup (process-exit
// on failure per ADR-0007 CC1), starts the HTTP trigger handler
// (W3-P4e per ADR-0014 §1 eager-at-load — listener binds only
// after the initial load completes), starts two periodic loops
// (loader refresh per ADR-0007 CC9 and orphan-run detection per
// ADR-0007 CC11), and waits for SIGTERM / SIGINT for graceful
// shutdown.
//
// The runner is constructed at startup and invoked by the HTTP
// handler for every accepted trigger; in-flight executions are
// isolated against the manifest active at trigger acceptance per
// ADR-0007 §3.
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
	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"dq-platform/engine/internal/alerts"
	"dq-platform/engine/internal/api"
	"dq-platform/engine/internal/eval"
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
	PubSubProject         string // empty → reuse BigQueryProject
	PubSubTopic           string // empty → NoopPublisher (no alerts emitted)
	LoaderRefreshInterval time.Duration
	OrphanThreshold       time.Duration
	OrphanScanInterval    time.Duration
	HTTPAddr              string
	SourceProject         string // empty → eval.Evaluator defaults to BigQuery client project
	SourceDataset         string // empty → row_count_positive returns ResultError
	LogLevel              slog.Level

	// Endpoint overrides for the local emulator. Honored when
	// non-empty; ignored in production. PUBSUB_EMULATOR_HOST is
	// honored by the Pub/Sub SDK itself; the binary does not have
	// to plumb it (see cloud.google.com/go/pubsub/v2).
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

	// Alerts publisher per ADR-0006. If DQ_PUBSUB_TOPIC is empty
	// the binary runs with NoopPublisher — useful for local-dev
	// processes that don't need to depend on the Pub/Sub emulator.
	// The Pub/Sub SDK honors PUBSUB_EMULATOR_HOST automatically;
	// the binary does not have to plumb it.
	publisher, closePublisher, err := newAlertsPublisher(ctx, cfg, logger)
	if err != nil {
		logger.Error("create alerts publisher", "error", err.Error())
		os.Exit(1)
	}
	defer closePublisher()

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

	// Orphan detector (ADR-0007 CC11).
	detector, err := orphan.New(store, orphan.Config{
		EngineVersion: cfg.EngineVersion,
		Threshold:     cfg.OrphanThreshold,
		Logger:        logger,
		Publisher:     publisher,
	})
	if err != nil {
		logger.Error("new orphan detector", "error", err.Error())
		os.Exit(1)
	}

	// Evaluator (W3-P6c). The BigQuery-backed evaluator
	// dispatches on CheckSpec.Kind; P6c ships
	// `row_count_positive`. Construction is fail-soft: if
	// DQ_SOURCE_DATASET is unset the engine still starts, but
	// data-plane checks return ResultError with a clear
	// "source dataset not configured" diagnostic per ADR-0004 CC1.
	evaluator, err := eval.New(eval.Config{
		Client:        bqClient,
		SourceProject: cfg.SourceProject,
		SourceDataset: cfg.SourceDataset,
		Logger:        logger,
	})
	if err != nil {
		logger.Error("new evaluator", "error", err.Error())
		os.Exit(1)
	}

	// Runner shared between every HTTP trigger acceptance
	// (W3-P4e). Per ADR-0007 §3 the in-flight execution is
	// isolated against the manifest active at plan creation; the
	// trigger handler captures the manifest reference at
	// acceptance and passes its RulesetVersion through
	// TriggerRequest, overriding the runner's constructor-time
	// pin.
	r, err := runner.New(runner.Config{
		Store:          store,
		Evaluator:      evaluator,
		EngineVersion:  cfg.EngineVersion,
		RulesetVersion: initial.RulesetVersion,
		Logger:         logger,
		Publisher:      publisher,
	})
	if err != nil {
		logger.Error("new runner", "error", err.Error())
		os.Exit(1)
	}

	// HTTP trigger handler (W3-P4e per ADR-0014). The listener
	// binds only after the initial manifest load completes
	// (ADR-0014 §1 eager-at-load).
	apiHandler, err := api.NewHandler(api.HandlerConfig{
		Dispatcher:     r,
		ActiveManifest: current.get,
		EngineCtx:      ctx,
		Logger:         logger,
		Publisher:      publisher,
	})
	if err != nil {
		logger.Error("new api handler", "error", err.Error())
		os.Exit(1)
	}
	httpServer := api.NewServer(cfg.HTTPAddr, apiHandler, logger)

	var wg sync.WaitGroup
	wg.Add(3)
	go loaderRefreshLoop(ctx, &wg, logger, ldr, current, cfg.LoaderRefreshInterval)
	go orphanScanLoop(ctx, &wg, logger, detector, cfg.OrphanScanInterval)
	go func() {
		defer wg.Done()
		if err := httpServer.ListenAndServe(); err != nil {
			logger.Error("http server exited with error", "error", err.Error())
		}
	}()

	// Wait for signal.
	<-ctx.Done()
	logger.Info("shutdown signal received", "reason", ctx.Err())

	// Drain the HTTP listener before letting the periodic loops
	// wind down — this stops accepting new triggers and waits
	// (bounded) for in-flight handlers to return.
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http server shutdown returned error", "error", err.Error())
	}
	cancelShutdown()

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
		HTTPAddr:              ":8080",
		LogLevel:              slog.LevelInfo,
		StorageEmulatorHost:   os.Getenv("STORAGE_EMULATOR_HOST"),
		BigQueryEmulatorHost:  os.Getenv("BIGQUERY_EMULATOR_HOST"),
	}

	cfg.EngineVersion = os.Getenv("DQ_ENGINE_VERSION")
	cfg.GCSBucket = os.Getenv("DQ_GCS_BUCKET")
	cfg.BigQueryProject = os.Getenv("DQ_BIGQUERY_PROJECT")
	cfg.BigQueryDataset = os.Getenv("DQ_BIGQUERY_DATASET")
	cfg.PubSubProject = os.Getenv("DQ_PUBSUB_PROJECT")
	cfg.PubSubTopic = os.Getenv("DQ_PUBSUB_TOPIC")

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
	if v := os.Getenv("DQ_HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	cfg.SourceProject = os.Getenv("DQ_SOURCE_PROJECT")
	cfg.SourceDataset = os.Getenv("DQ_SOURCE_DATASET")
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

// newAlertsPublisher constructs the engine-process alerting
// publisher per ADR-0006. Returns the publisher plus a close
// function the caller defers; the close function is a no-op when
// no Pub/Sub topic is configured. Returns an error only when
// topic configuration is requested but the Pub/Sub client cannot
// be created — the binary exits non-zero in that case so a
// misconfigured deployment fails loudly at startup rather than
// silently swallowing alerts.
func newAlertsPublisher(ctx context.Context, cfg envConfig, logger *slog.Logger) (alerts.Publisher, func(), error) {
	if cfg.PubSubTopic == "" {
		logger.Info("DQ_PUBSUB_TOPIC not set; using NoopPublisher (no alerts will be emitted)")
		return alerts.NoopPublisher{}, func() {}, nil
	}
	project := cfg.PubSubProject
	if project == "" {
		project = cfg.BigQueryProject
	}
	client, err := pubsub.NewClient(ctx, project)
	if err != nil {
		return nil, nil, fmt.Errorf("pubsub.NewClient(project=%q): %w", project, err)
	}
	pub := alerts.NewPubSubPublisher(client, cfg.PubSubTopic)
	logger.Info("alerts publisher wired",
		"pubsub_project", project,
		"pubsub_topic", cfg.PubSubTopic,
	)
	closeFn := func() {
		// Close the publisher first to flush in-flight emits, then
		// close the underlying client.
		pub.Close()
		if err := client.Close(); err != nil {
			logger.Warn("pubsub client close", "error", err.Error())
		}
	}
	return pub, closeFn, nil
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
