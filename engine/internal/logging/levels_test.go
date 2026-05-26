// path: engine/internal/logging/levels_test.go

package logging

import (
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

func TestParseLogLevels_UnsetOrEmpty(t *testing.T) {
	for _, raw := range []string{"", "  ", "\t", "\n"} {
		t.Run("empty"+raw, func(t *testing.T) {
			res, err := ParseLogLevels(raw)
			if err != nil {
				t.Fatalf("ParseLogLevels(%q): %v", raw, err)
			}
			if len(res.Levels) != 0 {
				t.Errorf("Levels = %v; want empty", res.Levels)
			}
			if len(res.Ignored) != 0 {
				t.Errorf("Ignored = %v; want empty", res.Ignored)
			}
		})
	}
}

func TestParseLogLevels_HappyPath(t *testing.T) {
	res, err := ParseLogLevels("root:info,engine.loader:debug,engine.runner:warn")
	if err != nil {
		t.Fatalf("ParseLogLevels: %v", err)
	}
	want := map[string]slog.Level{
		"root":          slog.LevelInfo,
		"engine.loader": slog.LevelDebug,
		"engine.runner": slog.LevelWarn,
	}
	if !reflect.DeepEqual(res.Levels, want) {
		t.Errorf("Levels = %v; want %v", res.Levels, want)
	}
	if len(res.Ignored) != 0 {
		t.Errorf("Ignored = %v; want empty (all keys canonical)", res.Ignored)
	}
}

func TestParseLogLevels_CaseInsensitive(t *testing.T) {
	// ADR-0043 §"Clause 1 — Grammar": DEBUG, Debug, and debug all
	// canonicalize to slog.LevelDebug.
	res, err := ParseLogLevels("root:INFO,engine.loader:Debug,engine.runner:WARN,engine.alerts:ErRoR")
	if err != nil {
		t.Fatalf("ParseLogLevels: %v", err)
	}
	want := map[string]slog.Level{
		"root":          slog.LevelInfo,
		"engine.loader": slog.LevelDebug,
		"engine.runner": slog.LevelWarn,
		"engine.alerts": slog.LevelError,
	}
	if !reflect.DeepEqual(res.Levels, want) {
		t.Errorf("Levels = %v; want %v", res.Levels, want)
	}
}

func TestParseLogLevels_WhitespaceTrim(t *testing.T) {
	// ADR-0043 §"Clause 1 — Whitespace handling": whitespace
	// immediately adjacent to ',' or ':' is trimmed; leading +
	// trailing whitespace on the whole value is trimmed.
	res, err := ParseLogLevels("  root: info , engine.loader: debug , engine.runner :warn  ")
	if err != nil {
		t.Fatalf("ParseLogLevels: %v", err)
	}
	want := map[string]slog.Level{
		"root":          slog.LevelInfo,
		"engine.loader": slog.LevelDebug,
		"engine.runner": slog.LevelWarn,
	}
	if !reflect.DeepEqual(res.Levels, want) {
		t.Errorf("Levels = %v; want %v", res.Levels, want)
	}
}

func TestParseLogLevels_IgnoredPackagesTracked(t *testing.T) {
	// Unknown package names are silently honored (kept in Levels)
	// but reported in Ignored for the caller's audit-log line.
	res, err := ParseLogLevels("root:info,engine.unknown:debug,engine.compilers:debug")
	if err != nil {
		t.Fatalf("ParseLogLevels: %v", err)
	}
	wantLevels := map[string]slog.Level{
		"root":             slog.LevelInfo,
		"engine.unknown":   slog.LevelDebug,
		"engine.compilers": slog.LevelDebug,
	}
	if !reflect.DeepEqual(res.Levels, wantLevels) {
		t.Errorf("Levels = %v; want %v", res.Levels, wantLevels)
	}
	wantIgnored := []string{"engine.compilers", "engine.unknown"}
	if !reflect.DeepEqual(res.Ignored, wantIgnored) {
		t.Errorf("Ignored = %v; want %v (sorted)", res.Ignored, wantIgnored)
	}
}

func TestParseLogLevels_SyntacticErrors(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		wantSub  string // expected substring of error message
	}{
		{"missing-colon", "engine.loaderdebug", "missing ':' separator"},
		{"empty-pkg", ":debug", "empty package name"},
		{"empty-level", "engine.loader:", "empty level value"},
		{"bad-level", "engine.loader:trace", "is not one of debug|info|warn|error"},
		{"leading-dot", ".engine:debug", "empty segment"},
		{"trailing-dot", "engine.:debug", "empty segment"},
		{"doubled-dot", "engine..loader:debug", "empty segment"},
		{"hyphen-in-pkg", "engine-loader:debug", "does not match grammar"},
		{"digit-first", "1engine:debug", "does not match grammar"},
		{"internal-space-pkg", "engine .loader:debug", "does not match grammar"},
		{"empty-pair", "root:info,,engine.loader:debug", "empty pair"},
		{"duplicate-pkg", "engine.loader:debug,engine.loader:warn", "appears more than once"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseLogLevels(tc.raw)
			if err == nil {
				t.Fatalf("ParseLogLevels(%q) returned nil; want error", tc.raw)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestParseLogLevels_AllCanonicalInventoryAccepted(t *testing.T) {
	// Verify every CanonicalInventory entry parses without being
	// reported as ignored — the inventory itself must be
	// internally consistent.
	for _, name := range CanonicalInventory {
		res, err := ParseLogLevels(name + ":info")
		if err != nil {
			t.Fatalf("ParseLogLevels(%q:info): %v", name, err)
		}
		if len(res.Ignored) != 0 {
			t.Errorf("canonical name %q produced Ignored=%v; CanonicalInventory must be self-consistent",
				name, res.Ignored)
		}
	}
}

func TestParseLogLevels_IntermediatePrefixAllowed(t *testing.T) {
	// ADR-0043 §"Clause 2": "engine" (bare) is in
	// CanonicalInventory as a deliberate wildcard. Other
	// intermediate prefixes parse but are reported as ignored —
	// they still work at resolution time (handler does
	// longest-prefix-match), but the inventory only lists
	// curated entry points.
	res, err := ParseLogLevels("engine.dsl.schema:debug")
	if err != nil {
		t.Fatalf("ParseLogLevels: %v", err)
	}
	if _, ok := res.Levels["engine.dsl.schema"]; !ok {
		t.Errorf("Levels missing engine.dsl.schema entry")
	}
	if len(res.Ignored) != 1 || res.Ignored[0] != "engine.dsl.schema" {
		t.Errorf("Ignored = %v; want [engine.dsl.schema]", res.Ignored)
	}
}
