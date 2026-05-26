// path: engine/internal/env/local.go

package env

import "time"

// Local is the canonical configuration for the local environment.
// Values match the docker-compose substrate (see
// docker-compose.yml and the project IDs / bucket names hardcoded
// there) so `make demo-p6` and the engine binary's local runs
// share the same wiring without per-run env-var overrides.
//
// LoaderRefreshInterval is set to 2s so local iteration cycles
// (re-publish manifest, observe refresh) are fast. OrphanThreshold
// is shorter than the production default for the same reason.
var Local = EnvConfig{
	Name:                  NameLocal,
	EngineVersion:         "0.1.0",
	GCSBucket:             "dq-local",
	BigQueryProject:       "dq-local",
	BigQueryDataset:       "dq_results_local",
	PubSubProject:         "dq-local",
	PubSubTopic:           "dq-alerts-local",
	KafkaBootstrap:        "localhost:9092",
	HTTPAddr:              ":8080",
	LogLevel:              LogLevelInfo,
	LoaderRefreshInterval: 2 * time.Second,
	OrphanThreshold:       5 * time.Minute,
	OrphanScanInterval:    1 * time.Minute,
	RecordModeCost: RecordModeCost{
		MaxEvidenceSampleSize: 100,
		MaxConsumerLag:        5 * time.Minute,
		MaxLatenessTolerance:  5 * time.Minute,
		SampleStorageCapMB:    100,
	},
	EvidenceRetention: EvidenceRetention{
		// ADR-0031 §"Single-tier retention" — local env keeps
		// 30 days of evidence; matches the partition_expiration
		// applied by EnsureSchema.
		ResultsRetention: 30 * 24 * time.Hour,
	},
	// ADR-0033 §"Per-env SchedulerCatchupHorizon" — 1h keeps the
	// dev feedback loop tight; missed windows older than an hour
	// typically aren't worth catching up during single-developer
	// iteration.
	SchedulerCatchupHorizon: 1 * time.Hour,
}
