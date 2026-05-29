// path: engine/cmd/dq-engine/main_test.go

package main

import (
	"errors"
	"log/slog"
	"strings"
	"testing"

	"dq-platform/engine/internal/env"
)

// TestReadEnv_HappyPath exercises the W3-P7a readEnv path with
// DQ_ENV=local. The check also acts as a guard against the
// startupConfig-embedding shadowing regression noted in the
// W3-P7a critique: if a future contributor accidentally adds a
// field on startupConfig that shadows an EnvConfig field via
// Go's struct-embedding rules, the cfg.GCSBucket / cfg.PubSubTopic
// reads here would surface as zero values and the test would
// fail.
func TestReadEnv_HappyPath(t *testing.T) {
	t.Setenv("DQ_ENV", "local")
	t.Setenv("STORAGE_EMULATOR_HOST", "")
	t.Setenv("BIGQUERY_EMULATOR_HOST", "")

	cfg, err := readEnv()
	if err != nil {
		t.Fatalf("readEnv: %v", err)
	}
	if cfg.Name != env.NameLocal {
		t.Errorf("cfg.Name = %q; want %q", cfg.Name, env.NameLocal)
	}
	if cfg.SlogLevel != slog.LevelInfo {
		t.Errorf("cfg.SlogLevel = %v; want %v", cfg.SlogLevel, slog.LevelInfo)
	}
	if cfg.GCSBucket == "" {
		t.Error("cfg.GCSBucket is empty; struct-embedding promotion broken?")
	}
	if cfg.BigQueryProject == "" {
		t.Error("cfg.BigQueryProject is empty; struct-embedding promotion broken?")
	}
	if cfg.PubSubTopic == "" {
		t.Error("cfg.PubSubTopic is empty; struct-embedding promotion broken?")
	}
	if cfg.LoaderRefreshInterval == 0 {
		t.Error("cfg.LoaderRefreshInterval is zero; struct-embedding promotion broken?")
	}
}

func TestReadEnv_MissingDQ_ENV(t *testing.T) {
	t.Setenv("DQ_ENV", "")

	_, err := readEnv()
	if err == nil {
		t.Fatal("readEnv() with empty DQ_ENV: expected error, got nil")
	}
}

func TestReadEnv_UnknownDQ_ENV(t *testing.T) {
	t.Setenv("DQ_ENV", "staging")

	_, err := readEnv()
	if err == nil {
		t.Fatal("readEnv() with unknown DQ_ENV: expected error, got nil")
	}
	if !errors.Is(err, env.ErrUnknownEnv) {
		t.Errorf("readEnv() error = %v; want wrap of env.ErrUnknownEnv", err)
	}
}

// TestValidateNoPlaceholders_NoOpForLocal confirms the explicit
// local-is-exempt branch: the substring is reserved as a marker and
// local declarations may use it freely without tripping the boot
// gate. The test injects a placeholder into a synthetic local config
// to make the no-op branch observable.
func TestValidateNoPlaceholders_NoOpForLocal(t *testing.T) {
	cfg := startupConfig{
		EnvConfig: env.EnvConfig{
			Name:      env.NameLocal,
			GCSBucket: "dq-local-PLACEHOLDER-rules",
		},
	}
	if err := validateNoPlaceholders(cfg); err != nil {
		t.Fatalf("validateNoPlaceholders for local: expected nil, got %v", err)
	}
}

// TestValidateNoPlaceholders_DetectsPlaceholdersInQA exercises the
// real env.QA declaration. The test documents that qa.go currently
// carries placeholders the operational session is expected to
// replace; the assertion is intentionally loose (substring match on
// "PLACEHOLDER" and at least one known field name) so a future PR
// that replaces individual placeholders doesn't break this test
// until every placeholder is gone — at which point qa.go is clean
// and validateNoPlaceholders no longer fires, which is the goal.
func TestValidateNoPlaceholders_DetectsPlaceholdersInQA(t *testing.T) {
	cfg := startupConfig{EnvConfig: env.QA}
	err := validateNoPlaceholders(cfg)
	if err == nil {
		t.Fatal("validateNoPlaceholders for env.QA: expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "PLACEHOLDER") {
		t.Errorf("error message %q: expected to contain \"PLACEHOLDER\"", msg)
	}
	if !strings.Contains(msg, "GCSBucket") {
		t.Errorf("error message %q: expected to name GCSBucket", msg)
	}
	if !strings.Contains(msg, `"qa"`) {
		t.Errorf("error message %q: expected to name the env (\"qa\")", msg)
	}
}

// TestValidateNoPlaceholders_CleanConfigPasses confirms that a qa
// config with every string field freed of the marker passes the
// gate. The synthetic config decouples this test from the evolution
// of qa.go.
func TestValidateNoPlaceholders_CleanConfigPasses(t *testing.T) {
	cfg := startupConfig{
		EnvConfig: env.EnvConfig{
			Name:              env.NameQA,
			EngineVersion:     "0.1.0",
			GCSBucket:         "dq-qa-real-rules",
			BigQueryProject:   "dq-qa-real",
			BigQueryDataset:   "dq_results_qa",
			PubSubProject:     "dq-qa-real",
			PubSubTopic:       "dq-alerts-qa",
			KafkaBootstrap:    "dq-qa-real-kafka:9092",
			HTTPAddr:          ":8080",
			LogLevel:          env.LogLevelInfo,
			OnboardingChannel: "slack:#dq-onboarding-qa",
		},
	}
	if err := validateNoPlaceholders(cfg); err != nil {
		t.Fatalf("validateNoPlaceholders for clean qa config: expected nil, got %v", err)
	}
}

// TestValidateNoPlaceholders_ReportsAllOffenders confirms the
// collect-everything-before-returning posture: an operator who
// fixes one placeholder field should not have to redeploy to
// discover the next. Two distinct placeholders must both appear in
// the message.
func TestValidateNoPlaceholders_ReportsAllOffenders(t *testing.T) {
	cfg := startupConfig{
		EnvConfig: env.EnvConfig{
			Name:            env.NameProd,
			GCSBucket:       "dq-prod-PLACEHOLDER-rules",
			BigQueryProject: "dq-prod-PLACEHOLDER",
		},
	}
	err := validateNoPlaceholders(cfg)
	if err == nil {
		t.Fatal("validateNoPlaceholders: expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "GCSBucket") {
		t.Errorf("error message %q: expected to name GCSBucket", msg)
	}
	if !strings.Contains(msg, "BigQueryProject") {
		t.Errorf("error message %q: expected to name BigQueryProject", msg)
	}
}

func TestReadEnv_EmulatorHostsPassThrough(t *testing.T) {
	t.Setenv("DQ_ENV", "local")
	t.Setenv("STORAGE_EMULATOR_HOST", "localhost:4443")
	t.Setenv("BIGQUERY_EMULATOR_HOST", "localhost:9050")

	cfg, err := readEnv()
	if err != nil {
		t.Fatalf("readEnv: %v", err)
	}
	if cfg.StorageEmulatorHost != "localhost:4443" {
		t.Errorf("StorageEmulatorHost = %q; want localhost:4443", cfg.StorageEmulatorHost)
	}
	if cfg.BigQueryEmulatorHost != "localhost:9050" {
		t.Errorf("BigQueryEmulatorHost = %q; want localhost:9050", cfg.BigQueryEmulatorHost)
	}
}
