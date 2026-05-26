// path: engine/internal/logging/handler.go

package logging

import (
	"context"
	"log/slog"
	"strings"
)

// componentAttrKey is the slog attribute key the engine binary
// uses to thread per-package identity through the logger chain
// per ADR-0043 §"Implementation posture (deferred)" + Consequence
// #2. Each package's logger is constructed via
// `slog.With(componentAttrKey, "engine.<x>")` in main.go.
const componentAttrKey = "component"

// Handler is a slog.Handler that resolves per-record log levels
// per ADR-0043's longest-prefix-match rule:
//
//  1. When a caller invokes `logger.With(componentAttrKey, "X")`,
//     the resulting handler captures component = "X".
//  2. At Enabled() time, the handler resolves the captured
//     component against its Levels map by longest-prefix-match
//     at dot boundaries, falling back to RootLevel.
//  3. Handle() forwards the record to the base handler.
//
// Loggers without a `component` attribute resolve to RootLevel
// uniformly.
type Handler struct {
	base      slog.Handler
	levels    map[string]slog.Level
	rootLevel slog.Level
	component string // empty until WithAttrs captures "component"
}

// HandlerConfig configures a new Handler.
type HandlerConfig struct {
	// Base is the underlying slog.Handler that emits the log
	// record once Enabled() has admitted it. Typically the
	// process-wide JSON handler from main.go.
	Base slog.Handler

	// Levels is the parsed DQ_LOG_LEVELS map per ParseLogLevels.
	// May be empty or nil; in that case every record's level
	// check falls back to RootLevel.
	Levels map[string]slog.Level

	// RootLevel is the fallback when no Levels entry matches the
	// captured component (including when no component is captured).
	// Typically derived from EnvConfig.LogLevel; overridden by an
	// explicit `root:` entry in DQ_LOG_LEVELS, which the caller
	// already merged into Levels[RootKey] when populating this
	// field.
	RootLevel slog.Level
}

// NewHandler constructs a Handler. If cfg.Levels carries a RootKey
// entry, it overrides cfg.RootLevel — matching ADR-0043 §"Clause 5
// — Defaults and additivity" semantics where a `root:` entry in
// DQ_LOG_LEVELS replaces EnvConfig.LogLevel.
func NewHandler(cfg HandlerConfig) *Handler {
	root := cfg.RootLevel
	if lvl, ok := cfg.Levels[RootKey]; ok {
		root = lvl
	}
	levels := cfg.Levels
	if levels == nil {
		levels = map[string]slog.Level{}
	}
	return &Handler{
		base:      cfg.Base,
		levels:    levels,
		rootLevel: root,
	}
}

// Enabled reports whether the handler admits a record at the given
// level for the current captured component. Resolves via
// longest-prefix-match against the Levels map; falls back to
// rootLevel.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.resolveLevel()
}

// Handle forwards the record to the base handler. Component-aware
// filtering is performed by Enabled(); by the time Handle is
// called, the record has already been admitted.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	return h.base.Handle(ctx, r)
}

// WithAttrs returns a new handler whose base carries the added
// attributes. If any attribute has the componentAttrKey, the
// handler captures it for future Enabled() resolution.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newH := &Handler{
		base:      h.base.WithAttrs(attrs),
		levels:    h.levels,
		rootLevel: h.rootLevel,
		component: h.component,
	}
	for _, a := range attrs {
		if a.Key == componentAttrKey {
			newH.component = a.Value.String()
		}
	}
	return newH
}

// WithGroup returns a new handler whose base nests subsequent
// attributes inside the named group. The component capture is
// preserved across groups (a `slog.WithGroup("foo").With("component",
// ...)` call still captures the component at the top level of the
// handler's state, matching slog's own attribute semantics where
// attribute keys are namespaced under group prefixes — operators
// passing the component attribute through a non-empty group are
// using a non-standard convention this handler does not specially
// support).
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		base:      h.base.WithGroup(name),
		levels:    h.levels,
		rootLevel: h.rootLevel,
		component: h.component,
	}
}

// resolveLevel implements ADR-0043 §"Clause 3 — Precedence":
// longest-prefix-match at dot boundaries. If no entry matches,
// returns rootLevel.
//
// Examples (given levels = {root: warn, engine: info,
// engine.loader: debug}):
//
//	component "engine.loader"          → debug (exact)
//	component "engine.loader.refresh"  → debug (longest prefix `engine.loader`)
//	component "engine.runner"          → info  (longest prefix `engine`)
//	component "tools.lint"             → warn  (root fallback)
//	component ""                       → warn  (root fallback)
func (h *Handler) resolveLevel() slog.Level {
	if h.component == "" {
		return h.rootLevel
	}
	name := h.component
	for {
		if lvl, ok := h.levels[name]; ok {
			return lvl
		}
		idx := strings.LastIndex(name, ".")
		if idx < 0 {
			return h.rootLevel
		}
		name = name[:idx]
	}
}
