// path: engine/cmd/dq-engine/main_test.go

package main

import (
	"errors"
	"log/slog"
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
