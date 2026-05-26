// path: engine/internal/env/prod.go

package env

import "time"

// Prod is the canonical configuration for the prod environment.
//
// Values prefixed with `dq-prod-PLACEHOLDER-` are placeholders;
// the operational session that provisions the real prod GCP
// project (per B1-4 C-MD-3.3) replaces them in a follow-up PR.
// The reflect-based exhaustiveness test treats the placeholders
// as populated (non-zero strings) so the package compiles and
// tests pass; an engine binary deployed against the placeholders
// fails loud at runtime.
//
// Production posture: 30s refresh, 1h orphan threshold, 5m scan
// — matches the production-default tunables foundation 05's cost
// discipline section calls out as the canonical "data plane is
// not a tight-loop system" cadence.
//
// HTTPAddr binds on all interfaces (":8080") by default; the
// production Service / Ingress configured by the W3-P7c overlay
// session is the production-hardening surface for external
// exposure. The bind address itself is a deployment-overlay
// concern that the operational session reviews alongside the
// GCP-project placeholders below.
var Prod = EnvConfig{
	Name:                  NameProd,
	EngineVersion:         "0.1.0",
	GCSBucket:             "dq-prod-PLACEHOLDER-rules",
	BigQueryProject:       "dq-prod-PLACEHOLDER",
	BigQueryDataset:       "dq_results_prod",
	PubSubProject:         "dq-prod-PLACEHOLDER",
	PubSubTopic:           "dq-alerts-prod",
	KafkaBootstrap:        "dq-prod-PLACEHOLDER-kafka:9092",
	HTTPAddr:              ":8080",
	LogLevel:              LogLevelInfo,
	LoaderRefreshInterval: 30 * time.Second,
	OrphanThreshold:       1 * time.Hour,
	OrphanScanInterval:    5 * time.Minute,
	RecordModeCost: RecordModeCost{
		MaxEvidenceSampleSize: 10000,
		MaxConsumerLag:        1 * time.Hour,
		MaxLatenessTolerance:  30 * time.Minute,
		SampleStorageCapMB:    10000,
	},
	EvidenceRetention: EvidenceRetention{
		// ADR-0031 §"Single-tier retention" — prod keeps 365 days.
		ResultsRetention: 365 * 24 * time.Hour,
	},
	// ADR-0033 §"Per-env SchedulerCatchupHorizon" — 24h tolerates
	// a full day of scheduler downtime without losing forensic
	// fidelity; longer horizons amplify the cost spike at
	// recovery and would benefit from an explicit backfill
	// posture instead.
	SchedulerCatchupHorizon: 24 * time.Hour,
}
