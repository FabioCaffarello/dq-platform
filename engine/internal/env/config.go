// path: engine/internal/env/config.go

package env

import (
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Name is the closed enum of first-class environment identifiers
// committed by B1-4 MD-2.
type Name string

const (
	NameLocal Name = "local"
	NameQA    Name = "qa"
	NameProd  Name = "prod"
)

// LogLevel is the typed log-level enum used by EnvConfig. A
// distinct string type (rather than slog.Level directly) keeps
// LogLevel's zero value distinguishable — slog.LevelInfo is the
// integer zero, which would defeat the reflect-based
// exhaustiveness test in env_test.go.
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// Slog converts the typed LogLevel to a slog.Level. Returns an
// error for any value outside the closed enum so a misconfigured
// per-env file fails loud at startup rather than silently
// defaulting.
func (l LogLevel) Slog() (slog.Level, error) {
	switch l {
	case LogLevelDebug:
		return slog.LevelDebug, nil
	case LogLevelInfo:
		return slog.LevelInfo, nil
	case LogLevelWarn:
		return slog.LevelWarn, nil
	case LogLevelError:
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("env: unknown log level %q (want debug|info|warn|error)", l)
	}
}

// EnvConfig is the typed configuration for one environment.
// Fields mirror the 13 application-config env vars the engine
// previously read at startup. Every field must be populated in
// every per-env declaration (local.go, qa.go, prod.go) per B1-4
// MD-4; the reflect-based test in env_test.go enforces this
// at CI time.
type EnvConfig struct {
	Name                  Name          // local / qa / prod
	EngineVersion         string        // semver per ADR-0001
	GCSBucket             string        // object store bucket (ADR-0005)
	BigQueryProject       string        // results project (ADR-0003)
	BigQueryDataset       string        // results dataset
	PubSubProject         string        // alerting project (ADR-0006)
	PubSubTopic           string        // alerting topic; the binary maps an absent topic to NoopPublisher at construction time
	SourceProject         string        // data-plane source project (W3-P6c)
	SourceDataset         string        // data-plane source dataset
	HTTPAddr              string        // listener address (ADR-0014)
	LogLevel              LogLevel      // debug | info | warn | error
	LoaderRefreshInterval time.Duration // ADR-0007 §4 cadence
	OrphanThreshold       time.Duration // ADR-0007 CC11 cutoff
	OrphanScanInterval    time.Duration // orphan ticker cadence
}

// ErrUnknownEnv is returned by Select for any input outside the
// closed Name enum.
var ErrUnknownEnv = errors.New("env: unknown DQ_ENV value")

// Select returns the typed EnvConfig for the named environment.
// The name must be one of the closed Name enum values (B1-4
// MD-2); any other input (including empty string) returns
// ErrUnknownEnv. Names are matched exactly with case sensitivity
// (committed as the W3-P7a posture per B1-4 OQ-MD-2.1's
// canonicalization deferral); a future amendment can relax to
// case-insensitive matching if it becomes a real ergonomic
// problem.
func Select(name string) (EnvConfig, error) {
	switch Name(name) {
	case NameLocal:
		return Local, nil
	case NameQA:
		return QA, nil
	case NameProd:
		return Prod, nil
	default:
		return EnvConfig{}, fmt.Errorf("%w: %q (want local|qa|prod)", ErrUnknownEnv, name)
	}
}
