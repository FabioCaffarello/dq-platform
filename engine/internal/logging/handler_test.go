// path: engine/internal/logging/handler_test.go

package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// newTestLogger returns a *slog.Logger backed by a Handler wired
// against the levels map, with output captured in buf.
func newTestLogger(buf *bytes.Buffer, levels map[string]slog.Level, root slog.Level) *slog.Logger {
	base := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewHandler(HandlerConfig{
		Base:      base,
		Levels:    levels,
		RootLevel: root,
	})
	return slog.New(h)
}

// linesFromBuf returns each non-empty JSON line from the captured
// buffer, decoded into a generic map for assertion.
func linesFromBuf(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("decode log line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

func TestHandler_RootFallback(t *testing.T) {
	// No component captured → root level applies.
	var buf bytes.Buffer
	logger := newTestLogger(&buf, nil, slog.LevelInfo)
	logger.Debug("debug-1")
	logger.Info("info-1")
	logger.Warn("warn-1")

	lines := linesFromBuf(t, &buf)
	if len(lines) != 2 {
		t.Fatalf("got %d log lines; want 2 (debug filtered, info+warn admitted)", len(lines))
	}
	if lines[0]["msg"] != "info-1" || lines[1]["msg"] != "warn-1" {
		t.Errorf("admitted lines = %v; want [info-1 warn-1]", lines)
	}
}

func TestHandler_RootKeyOverridesEnvConfigDefault(t *testing.T) {
	// ADR-0043 §"Clause 5": a `root:` entry in the parsed map
	// replaces the EnvConfig.LogLevel default.
	var buf bytes.Buffer
	logger := newTestLogger(&buf, map[string]slog.Level{
		RootKey: slog.LevelError,
	}, slog.LevelInfo)
	logger.Warn("warn-1")
	logger.Error("error-1")

	lines := linesFromBuf(t, &buf)
	if len(lines) != 1 || lines[0]["msg"] != "error-1" {
		t.Errorf("with root:error, admitted = %v; want only error-1", lines)
	}
}

func TestHandler_PerComponentResolution(t *testing.T) {
	levels := map[string]slog.Level{
		RootKey:         slog.LevelWarn,
		"engine":        slog.LevelInfo,
		"engine.loader": slog.LevelDebug,
	}

	var buf bytes.Buffer
	root := newTestLogger(&buf, levels, slog.LevelWarn)

	// Exact match: engine.loader → debug.
	buf.Reset()
	loader := root.With("component", "engine.loader")
	loader.Debug("loader-debug")
	loader.Info("loader-info")
	if got := len(linesFromBuf(t, &buf)); got != 2 {
		t.Errorf("engine.loader admitted %d; want 2 (both debug + info)", got)
	}

	// Longest-prefix match: engine.loader.refresh → debug
	// (matches `engine.loader` after one prefix strip).
	buf.Reset()
	loaderSub := loader.With("component", "engine.loader.refresh")
	loaderSub.Debug("loader-refresh-debug")
	if got := len(linesFromBuf(t, &buf)); got != 1 {
		t.Errorf("engine.loader.refresh admitted %d; want 1 (debug admitted via prefix)", got)
	}

	// Wildcard-prefix match: engine.runner → info (via `engine`).
	buf.Reset()
	runner := root.With("component", "engine.runner")
	runner.Debug("runner-debug")
	runner.Info("runner-info")
	if got := len(linesFromBuf(t, &buf)); got != 1 {
		t.Errorf("engine.runner admitted %d; want 1 (debug filtered, info admitted via `engine` prefix)", got)
	}

	// No prefix match: tools.lint → root warn.
	buf.Reset()
	other := root.With("component", "tools.lint")
	other.Info("tools-info")
	other.Warn("tools-warn")
	if got := len(linesFromBuf(t, &buf)); got != 1 {
		t.Errorf("tools.lint admitted %d; want 1 (info filtered, warn admitted via root)", got)
	}
}

func TestHandler_DotBoundaryNotSubstring(t *testing.T) {
	// Resolution must match at dot boundaries, not arbitrary
	// substring. `engine` should NOT match `engineroom` even
	// though "engine" is a prefix string of "engineroom".
	levels := map[string]slog.Level{
		"engine": slog.LevelDebug,
	}
	var buf bytes.Buffer
	root := newTestLogger(&buf, levels, slog.LevelWarn)

	engineroom := root.With("component", "engineroom")
	engineroom.Debug("should-be-filtered")
	engineroom.Info("should-be-filtered")
	engineroom.Warn("admitted-via-root")

	lines := linesFromBuf(t, &buf)
	if len(lines) != 1 || lines[0]["msg"] != "admitted-via-root" {
		t.Errorf("engineroom must not inherit engine:debug; got %v", lines)
	}
}

func TestHandler_ComponentAttrPropagatesToOutput(t *testing.T) {
	// The component attribute attached via With should appear in
	// the emitted JSON record so consumers can group by it.
	var buf bytes.Buffer
	root := newTestLogger(&buf, nil, slog.LevelInfo)
	logger := root.With("component", "engine.loader")
	logger.Info("hello")

	lines := linesFromBuf(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("got %d lines; want 1", len(lines))
	}
	if lines[0]["component"] != "engine.loader" {
		t.Errorf("component attr in record = %v; want engine.loader", lines[0]["component"])
	}
}

func TestHandler_NilLevelsTolerated(t *testing.T) {
	// NewHandler with nil Levels is equivalent to an empty
	// override map: every record resolves to RootLevel.
	var buf bytes.Buffer
	h := NewHandler(HandlerConfig{
		Base:      slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}),
		Levels:    nil,
		RootLevel: slog.LevelWarn,
	})
	logger := slog.New(h).With("component", "engine.loader")
	logger.Info("filtered")
	logger.Warn("admitted")

	lines := linesFromBuf(t, &buf)
	if len(lines) != 1 || lines[0]["msg"] != "admitted" {
		t.Errorf("nil levels should resolve everything to root; got %v", lines)
	}
}

func TestHandler_EnabledWithoutCallingHandle(t *testing.T) {
	// Verify Enabled() returns the correct boolean independent of
	// any subsequent Handle() call — the contract of slog.Handler
	// allows callers to skip expensive attribute computation when
	// Enabled() returns false.
	levels := map[string]slog.Level{
		"engine.loader": slog.LevelWarn,
	}
	h := NewHandler(HandlerConfig{
		Base:      slog.NewJSONHandler(new(bytes.Buffer), &slog.HandlerOptions{Level: slog.LevelDebug}),
		Levels:    levels,
		RootLevel: slog.LevelInfo,
	})
	loaderH := h.WithAttrs([]slog.Attr{slog.String("component", "engine.loader")})

	ctx := context.Background()
	if loaderH.Enabled(ctx, slog.LevelDebug) {
		t.Error("Enabled(debug) for engine.loader:warn = true; want false")
	}
	if loaderH.Enabled(ctx, slog.LevelInfo) {
		t.Error("Enabled(info) for engine.loader:warn = true; want false")
	}
	if !loaderH.Enabled(ctx, slog.LevelWarn) {
		t.Error("Enabled(warn) for engine.loader:warn = false; want true")
	}
	if !loaderH.Enabled(ctx, slog.LevelError) {
		t.Error("Enabled(error) for engine.loader:warn = false; want true")
	}
}

func TestHandler_WithGroupPreservesComponent(t *testing.T) {
	// WithGroup should not lose the captured component.
	levels := map[string]slog.Level{
		"engine.loader": slog.LevelError,
	}
	var buf bytes.Buffer
	h := NewHandler(HandlerConfig{
		Base:      slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}),
		Levels:    levels,
		RootLevel: slog.LevelInfo,
	})
	loaderH := h.WithAttrs([]slog.Attr{slog.String("component", "engine.loader")})
	groupedH := loaderH.WithGroup("attrs")
	logger := slog.New(groupedH)

	// engine.loader is pinned at error, so info should be filtered.
	logger.Info("filtered-info")
	logger.Error("admitted-error")

	lines := linesFromBuf(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("got %d lines; want 1", len(lines))
	}
	if lines[0]["msg"] != "admitted-error" {
		t.Errorf("admitted line msg = %v; want admitted-error", lines[0]["msg"])
	}
}
