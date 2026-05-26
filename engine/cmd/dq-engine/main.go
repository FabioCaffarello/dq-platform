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
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
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
	"dq-platform/engine/internal/dsl/catalog"
	"dq-platform/engine/internal/dsl/spec"
	"dq-platform/engine/internal/env"
	"dq-platform/engine/internal/eval"
	"dq-platform/engine/internal/loader"
	"dq-platform/engine/internal/logging"
	"dq-platform/engine/internal/orphan"
	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// startupConfig is the engine binary's startup-time configuration.
// The 13 application-config fields are sourced from the typed
// engine/internal/env package per foundation 04 §PAT-4 and B1-4
// MD-4 — the binary reads DQ_ENV at startup, calls env.Select to
// obtain the canonical EnvConfig for that env, and embeds it
// here so existing call sites that read cfg.GCSBucket / etc.
// continue to work via Go struct embedding.
//
// The two emulator-host overrides remain env-var driven (B1-4
// OQ-MD-4.1) — they are local-substrate concerns honored by the
// GCP SDKs directly, not application configuration.
//
// SlogLevel is the slog.Level resolved from EnvConfig.LogLevel at
// readEnv time so logging setup doesn't have to re-parse.
//
// LogLevels is the parsed DQ_LOG_LEVELS map per ADR-0043. Empty
// when the env var is unset; otherwise the keys are package names
// from ADR-0043's officially-supported inventory plus any
// intermediate-prefix wildcards the operator chose. IgnoredLogPackages
// names entries that parsed successfully but are not in the
// canonical inventory; main.go emits one info-level startup audit
// line listing them per ADR-0043 §"Clause 4".
type startupConfig struct {
	env.EnvConfig

	SlogLevel            slog.Level
	LogLevels            map[string]slog.Level
	IgnoredLogPackages   []string
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

	// Build the custom slog handler from ADR-0043. The base
	// JSON handler accepts every level; the logging.Handler
	// gates per-record admission based on the captured
	// `component` attribute resolved against cfg.LogLevels via
	// longest-prefix-match. Loggers without a `component`
	// attribute (this main() function's `logger`) resolve to
	// the root level — cfg.SlogLevel from EnvConfig.LogLevel,
	// or the `root:` override if DQ_LOG_LEVELS named one.
	baseHandler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	logHandler := logging.NewHandler(logging.HandlerConfig{
		Base:      baseHandler,
		Levels:    cfg.LogLevels,
		RootLevel: cfg.SlogLevel,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)
	logger.Info("dq-engine starting",
		"env", string(cfg.Name),
		"engine_version", cfg.EngineVersion,
		"bigquery_project", cfg.BigQueryProject,
		"bigquery_dataset", cfg.BigQueryDataset,
		"gcs_bucket", cfg.GCSBucket,
	)
	// ADR-0043 §"Clause 4" — unknown package names in
	// DQ_LOG_LEVELS are silently honored but reported in one
	// startup audit log line so operators can audit.
	if len(cfg.IgnoredLogPackages) > 0 {
		logger.Info("DQ_LOG_LEVELS: package names not in canonical inventory (still honored at resolution time)",
			"ignored_packages", cfg.IgnoredLogPackages,
			"adr_reference", "ADR-0043 §Clause 4",
		)
	}

	// Per-package loggers per ADR-0043 §"Implementation posture"
	// — each engine/internal/ package gets a logger that
	// carries `component=engine.<x>` as a fixed attribute. The
	// custom handler picks up the component on WithAttrs and
	// resolves per-record level against cfg.LogLevels.
	alertsLogger := logger.With("component", "engine.alerts")
	apiLogger := logger.With("component", "engine.api")
	evalLogger := logger.With("component", "engine.eval")
	loaderLogger := logger.With("component", "engine.loader")
	orphanLogger := logger.With("component", "engine.orphan")
	resultsLogger := logger.With("component", "engine.results")
	runnerLogger := logger.With("component", "engine.runner")

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
	publisher, closePublisher, err := newAlertsPublisher(ctx, cfg, alertsLogger)
	if err != nil {
		logger.Error("create alerts publisher", "error", err.Error())
		os.Exit(1)
	}
	defer closePublisher()

	// Result-write layer.
	store := results.NewBigQueryStore(bqClient, cfg.BigQueryProject, cfg.BigQueryDataset, resultsLogger)
	if err := store.EnsureSchema(ctx); err != nil {
		logger.Error("ensure schema", "error", err.Error())
		os.Exit(1)
	}

	// Loader — startup-mode load per ADR-0007 CC1. Process exits
	// non-zero on any failure.
	gcsStore := loader.NewGCSStore(gcsClient, cfg.GCSBucket)
	ldr, err := loader.New(gcsStore, loader.Config{
		EngineVersion:           cfg.EngineVersion,
		SupportedSchemaVersions: []int{1, 2},
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
		Logger:        orphanLogger,
		Publisher:     publisher,
	})
	if err != nil {
		logger.Error("new orphan detector", "error", err.Error())
		os.Exit(1)
	}

	// Evaluator (W3-P6c, extended to Wave-S sub-slice α). The
	// evaluator dispatches on CheckSpec.Kind via the handler
	// registry; New() registers set.row_count_positive (real
	// handler) and record.schema_conformance (stub until Wave-S
	// sub-slice β). Per ADR-0023 the rule's source descriptor
	// carries the BigQuery target; the evaluator no longer pins
	// a deployment-wide source.
	evaluator, err := eval.New(eval.Config{
		Client: bqClient,
		Logger: evalLogger,
	})
	if err != nil {
		logger.Error("new evaluator", "error", err.Error())
		os.Exit(1)
	}

	// Dispatcher startup invariant per ADR-0022 §C-B0S2.3: every
	// kind in the catalog must have a registered handler. The
	// catalog ships in the rules workspace at
	// rules/_schema/catalog.v1.yaml; the engine reads its
	// engine-side mirror at engine/internal/dsl/catalog/v1.yaml
	// at boot. Fail-fast if the registry diverges from the
	// catalog (extra registered kinds are fine; missing kinds
	// fail the boot).
	if err := verifyHandlerRegistryAgainstCatalog(evaluator, evalLogger); err != nil {
		logger.Error("dispatcher startup invariant failed (ADR-0022 §C-B0S2.3)", "error", err.Error())
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
		Logger:         runnerLogger,
		Publisher:      publisher,
	})
	if err != nil {
		logger.Error("new runner", "error", err.Error())
		os.Exit(1)
	}

	// CostGuardrails translates the deployment's ADR-0027
	// record-mode ceilings into the typed shape dsl/spec
	// consumes. resolveChecks below applies these per-rule at
	// trigger time; the record-mode rule loader below also
	// applies them at manifest-load time.
	costGuardrails := spec.CostGuardrails{
		MaxEvidenceSampleSize: cfg.RecordModeCost.MaxEvidenceSampleSize,
		MaxLatenessTolerance:  cfg.RecordModeCost.MaxLatenessTolerance,
	}

	// Resolve a trigger entity into the runner's []CheckSpec
	// (W3-P6d). Reads the entity's YAML body from the same
	// object store the loader uses, then parses it via the
	// dsl/spec strict parser. Per ADR-0027 the record-mode cost
	// guardrails are enforced here for set-mode rules too (the
	// cost function no-ops on set-mode; the check is uniform).
	//
	// ADR ties:
	//
	//   - ADR-0001 — `kind` is the discriminator on each check;
	//     parsed into runner.CheckSpec.Kind and dispatched by
	//     engine/internal/eval per check-kind.
	//   - ADR-0005 §1 — rule YAMLs are content-addressed at
	//     `yamls/by-hash/sha256-<hex>.yaml`; the manifest publisher
	//     (W3-P6a) wrote the body the closure reads here.
	//   - ADR-0007 §3 — in-flight executions isolate against the
	//     manifest active at plan creation; the closure captures
	//     the manifest reference via current.get() once per
	//     accepted trigger.
	//   - ADR-0027 — record-mode cost ceilings rejected here.
	//
	// Per the W3-P6d AskUserQuestion answer this is a per-trigger
	// read; caching at refresh time is deferred to a future
	// cost-discipline ADR.
	resolveChecks := func(reqCtx context.Context, entity string) ([]runner.CheckSpec, error) {
		m := current.get()
		if m == nil {
			return nil, fmt.Errorf("active manifest is unavailable")
		}
		var rule *loader.ManifestRule
		for i := range m.Rules {
			if m.Rules[i].Entity == entity {
				rule = &m.Rules[i]
				break
			}
		}
		if rule == nil {
			return nil, api.ErrEntityNotInManifest
		}
		body, err := gcsStore.ReadObject(reqCtx, rule.YamlPath)
		if err != nil {
			return nil, fmt.Errorf("read yaml body %q: %w", rule.YamlPath, err)
		}
		parsed, err := spec.Parse(body)
		if err != nil {
			return nil, fmt.Errorf("parse yaml %q: %w", rule.YamlPath, err)
		}
		if err := spec.EvaluateCost(parsed, costGuardrails); err != nil {
			return nil, err
		}
		return parsed.ToCheckSpecs(), nil
	}

	// HTTP trigger handler (W3-P4e per ADR-0014). The listener
	// binds only after the initial manifest load completes
	// (ADR-0014 §1 eager-at-load).
	apiHandler, err := api.NewHandler(api.HandlerConfig{
		Dispatcher:     r,
		ActiveManifest: current.get,
		ResolveChecks:  resolveChecks,
		EngineCtx:      ctx,
		Logger:         apiLogger,
		Publisher:      publisher,
	})
	if err != nil {
		logger.Error("new api handler", "error", err.Error())
		os.Exit(1)
	}
	httpServer := api.NewServer(cfg.HTTPAddr, apiHandler, apiLogger)

	// Record-mode runners per ADR-0024. For each record-mode
	// rule in the manifest, the engine starts a FranzConsumer-
	// backed RecordRunner. Each runner consumes its rule's
	// topic + consumer_group, accumulates per-window records,
	// and dispatches to the same Runner the HTTP handler uses
	// once a window closes.
	//
	// Per the β scope, manifest refresh does not yet restart
	// record runners — the set of record-mode rules is fixed
	// at boot. A future slice adds the per-rule lifecycle.
	recordRunners, err := buildRecordRunners(ctx, initial, gcsStore, r, cfg, costGuardrails, runnerLogger)
	if err != nil {
		logger.Error("build record runners", "error", err.Error())
		os.Exit(1)
	}
	logger.Info("record-mode runners constructed",
		"count", len(recordRunners),
		"adr_reference", "ADR-0024",
	)

	var wg sync.WaitGroup
	wg.Add(3 + len(recordRunners))
	go loaderRefreshLoop(ctx, &wg, loaderLogger, ldr, current, cfg.LoaderRefreshInterval)
	go orphanScanLoop(ctx, &wg, orphanLogger, detector, cfg.OrphanScanInterval)
	go func() {
		defer wg.Done()
		if err := httpServer.ListenAndServe(); err != nil {
			logger.Error("http server exited with error", "error", err.Error())
		}
	}()
	for i := range recordRunners {
		rr := recordRunners[i]
		go func() {
			defer wg.Done()
			if err := rr.Start(ctx); err != nil {
				logger.Error("record runner exited with error", "error", err.Error())
			}
		}()
	}

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

// readEnv resolves the startup configuration. DQ_ENV selects one
// of the closed-enum environments (local / qa / prod) committed
// by B1-4 MD-2; env.Select returns the canonical EnvConfig for
// that env. The two emulator-host overrides
// (STORAGE_EMULATOR_HOST / BIGQUERY_EMULATOR_HOST) remain
// env-var-driven per B1-4 §4.3.
func readEnv() (startupConfig, error) {
	envName := os.Getenv("DQ_ENV")
	if envName == "" {
		return startupConfig{}, fmt.Errorf("DQ_ENV is required (one of local|qa|prod per B1-4 MD-2)")
	}
	envCfg, err := env.Select(envName)
	if err != nil {
		return startupConfig{}, fmt.Errorf("DQ_ENV: %w", err)
	}
	slogLevel, err := envCfg.LogLevel.Slog()
	if err != nil {
		return startupConfig{}, fmt.Errorf("env %q: %w", envCfg.Name, err)
	}
	// DQ_LOG_LEVELS parse per ADR-0043. Syntactic errors are
	// fatal at startup (Clause 4 — engine exits non-zero);
	// unknown package names parse successfully and are recorded
	// for the startup audit log line below.
	parsed, err := logging.ParseLogLevels(os.Getenv("DQ_LOG_LEVELS"))
	if err != nil {
		// ParseLogLevels error messages already carry the
		// "DQ_LOG_LEVELS:" prefix, so we pass through verbatim
		// rather than double-wrapping.
		return startupConfig{}, err
	}
	return startupConfig{
		EnvConfig:            envCfg,
		SlogLevel:            slogLevel,
		LogLevels:            parsed.Levels,
		IgnoredLogPackages:   parsed.Ignored,
		StorageEmulatorHost:  os.Getenv("STORAGE_EMULATOR_HOST"),
		BigQueryEmulatorHost: os.Getenv("BIGQUERY_EMULATOR_HOST"),
	}, nil
}

func newGCSClient(ctx context.Context, cfg startupConfig) (*storage.Client, error) {
	if cfg.StorageEmulatorHost != "" {
		return storage.NewClient(ctx,
			option.WithoutAuthentication(),
			option.WithEndpoint("http://"+cfg.StorageEmulatorHost+"/storage/v1/"),
		)
	}
	return storage.NewClient(ctx)
}

func newBQClient(ctx context.Context, cfg startupConfig) (*bigquery.Client, error) {
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
func newAlertsPublisher(ctx context.Context, cfg startupConfig, logger *slog.Logger) (alerts.Publisher, func(), error) {
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

// buildRecordRunners scans the active manifest for record-mode
// rules per ADR-0021, parses each rule's YAML body, enforces
// the ADR-0027 cost guardrails, and constructs a FranzConsumer-
// backed RecordRunner for each. Returns an empty slice when no
// record-mode rules are present — set-mode-only manifests do
// not require the event-stream substrate.
func buildRecordRunners(
	ctx context.Context,
	manifest *loader.Manifest,
	gcsStore *loader.GCSStore,
	dispatcher runner.TriggerDispatcher,
	cfg startupConfig,
	guardrails spec.CostGuardrails,
	logger *slog.Logger,
) ([]*runner.RecordRunner, error) {
	var out []*runner.RecordRunner
	for i := range manifest.Rules {
		mr := manifest.Rules[i]
		body, err := gcsStore.ReadObject(ctx, mr.YamlPath)
		if err != nil {
			return nil, fmt.Errorf("read rule %q yaml body: %w", mr.Entity, err)
		}
		parsed, err := spec.Parse(body)
		if err != nil {
			return nil, fmt.Errorf("parse rule %q: %w", mr.Entity, err)
		}
		if parsed.Mode != spec.ModeRecord {
			continue
		}
		if err := spec.EvaluateCost(parsed, guardrails); err != nil {
			return nil, fmt.Errorf("rule %q rejected by cost guardrails: %w", mr.Entity, err)
		}
		if parsed.Source == nil || parsed.Source.Window == nil {
			return nil, fmt.Errorf("rule %q: record-mode requires source.kafka.window", mr.Entity)
		}
		windowDur, err := runner.ParseDuration(parsed.Source.Window.Duration)
		if err != nil {
			return nil, fmt.Errorf("rule %q: parse window.duration: %w", mr.Entity, err)
		}
		lateness, err := runner.ParseDuration(parsed.Source.Window.LatenessTolerance)
		if err != nil {
			return nil, fmt.Errorf("rule %q: parse window.lateness_tolerance: %w", mr.Entity, err)
		}
		consumer, err := runner.NewFranzConsumer(runner.FranzConsumerConfig{
			Brokers:       []string{cfg.KafkaBootstrap},
			ConsumerGroup: parsed.Source.ConsumerGroup,
			Topics:        []string{parsed.Source.Topic},
		})
		if err != nil {
			return nil, fmt.Errorf("rule %q: create kafka consumer: %w", mr.Entity, err)
		}
		rr, err := runner.NewRecordRunner(runner.RecordRunnerConfig{
			Consumer:       consumer,
			Dispatcher:     dispatcher,
			RulesetVersion: manifest.RulesetVersion,
			Logger:         logger,
			Sources: []runner.RecordSource{{
				Entity:            parsed.Entity,
				Topic:             parsed.Source.Topic,
				ConsumerGroup:     parsed.Source.ConsumerGroup,
				WindowDuration:    windowDur,
				LatenessTolerance: lateness,
				Checks:            parsed.ToCheckSpecs(),
			}},
		})
		if err != nil {
			return nil, fmt.Errorf("rule %q: new record runner: %w", mr.Entity, err)
		}
		logger.Info("record runner wired",
			"entity", parsed.Entity,
			"topic", parsed.Source.Topic,
			"consumer_group", parsed.Source.ConsumerGroup,
			"window_duration", windowDur,
			"lateness_tolerance", lateness,
			"adr_reference", "ADR-0024",
		)
		out = append(out, rr)
	}
	return out, nil
}

// verifyHandlerRegistryAgainstCatalog enforces ADR-0022 §C-B0S2.3:
// every kind in the engine-embedded catalog must have a registered
// handler. Extra registered kinds are fine (handlers may exist
// ahead of catalog adoption); missing kinds fail the boot.
//
// Sub-slice α posture: the catalog declares
// set.row_count_positive and record.schema_conformance; the
// evaluator registers both — the latter as a stub returning
// ResultError until Wave-S sub-slice β wires the real runtime.
func verifyHandlerRegistryAgainstCatalog(evaluator *eval.Evaluator, logger *slog.Logger) error {
	cat, err := catalog.Load()
	if err != nil {
		return fmt.Errorf("load embedded catalog: %w", err)
	}
	registered := make(map[string]struct{}, len(evaluator.RegisteredKinds()))
	for _, k := range evaluator.RegisteredKinds() {
		registered[k] = struct{}{}
	}
	var missing []string
	for _, k := range cat.Kinds {
		if _, ok := registered[k.Name]; !ok {
			missing = append(missing, k.Name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("catalog declares %d kind(s) with no registered handler: %v", len(missing), missing)
	}
	logger.Info("dispatcher startup invariant passed",
		"catalog_version", cat.Version,
		"catalog_kinds", len(cat.Kinds),
		"registered_kinds", len(evaluator.RegisteredKinds()),
		"adr_reference", "ADR-0022 §C-B0S2.3",
	)
	return nil
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
