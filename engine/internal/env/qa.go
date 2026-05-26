// path: engine/internal/env/qa.go

package env

import "time"

// QA is the canonical configuration for the qa environment.
//
// Values prefixed with `dq-qa-PLACEHOLDER-` are placeholders; the
// operational session that provisions the real qa GCP project
// (per B1-4 C-MD-3.3) replaces them in a follow-up PR with the
// concrete project / bucket / dataset / topic names. The
// reflect-based exhaustiveness test treats the placeholders as
// populated (they are non-zero strings), so the package compiles
// and tests pass; an engine binary deployed against the
// placeholders fails loud at runtime when the GCP SDKs cannot
// resolve them.
//
// LoaderRefreshInterval is the production-default 30s; qa
// mirrors prod's posture so rule-evolution flows in qa exercise
// the same timing the prod data-plane uses.
//
// HTTPAddr binds on all interfaces (":8080") by default; the
// per-environment Service / Ingress configured by the W3-P7c
// overlay session is the production-hardening surface for
// external exposure. The bind address itself is a deployment-
// overlay concern that the operational session reviews
// alongside the GCP-project placeholders below.
var QA = EnvConfig{
	Name:                  NameQA,
	EngineVersion:         "0.1.0",
	GCSBucket:             "dq-qa-PLACEHOLDER-rules",
	BigQueryProject:       "dq-qa-PLACEHOLDER",
	BigQueryDataset:       "dq_results_qa",
	PubSubProject:         "dq-qa-PLACEHOLDER",
	PubSubTopic:           "dq-alerts-qa",
	KafkaBootstrap:        "dq-qa-PLACEHOLDER-kafka:9092",
	HTTPAddr:              ":8080",
	LogLevel:              LogLevelInfo,
	LoaderRefreshInterval: 30 * time.Second,
	OrphanThreshold:       1 * time.Hour,
	OrphanScanInterval:    5 * time.Minute,
	RecordModeCost: RecordModeCost{
		MaxEvidenceSampleSize: 1000,
		MaxConsumerLag:        1 * time.Hour,
		MaxLatenessTolerance:  30 * time.Minute,
		SampleStorageCapMB:    1000,
	},
	EvidenceRetention: EvidenceRetention{
		// ADR-0031 §"Single-tier retention" — qa keeps 90 days.
		ResultsRetention: 90 * 24 * time.Hour,
	},
	// ADR-0033 §"Per-env SchedulerCatchupHorizon" — 6h matches
	// the typical overnight integration-test window; an
	// integration-run kicked off in the evening catches up
	// cleanly the next morning.
	SchedulerCatchupHorizon: 6 * time.Hour,
}
