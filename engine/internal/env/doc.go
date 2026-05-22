// path: engine/internal/env/doc.go

// Package env is the engine's typed multi-environment
// configuration package, scaffolded by W3-P7a per foundation 04
// §"PAT-4 — Typed multi-environment configuration" and B1-4 MD-4.
//
// The package replaces the engine binary's prior ad-hoc
// readEnv() that read 13 application-config env vars at startup.
// PAT-4 commits the model:
//
//   - One Go file per environment (local.go, qa.go, prod.go).
//   - Each file declares a typed EnvConfig var with the same
//     shape.
//   - Selection at startup via the DQ_ENV environment variable.
//   - No dynamic discovery, no inheritance chains, no implicit
//     fallbacks. Each per-env value is canonical for that env.
//
// B1-4 MD-2 commits the closed set of first-class environments
// (local, qa, prod). B1-4 MD-3 commits separate GCP projects per
// environment as the isolation boundary. B1-4 MD-4 commits the
// every-field-in-every-env rule: adding a field to EnvConfig
// requires populating it in every per-env file, enforced at CI
// time by TestExhaustive_AllFieldsPopulatedInAllEnvs in
// env_test.go.
//
// The two emulator-host overrides (STORAGE_EMULATOR_HOST and
// BIGQUERY_EMULATOR_HOST) remain env-var-driven and are not part
// of EnvConfig — they are local-substrate concerns honored by
// the GCP SDKs directly, not application configuration (B1-4
// OQ-MD-4.1 deferred this question to this refactor; the answer
// is "stays as env vars").
package env
