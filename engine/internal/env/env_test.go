// path: engine/internal/env/env_test.go

package env

import (
	"errors"
	"log/slog"
	"reflect"
	"testing"
)

// TestExhaustive_AllFieldsPopulatedInAllEnvs reflects over every
// per-env EnvConfig var and asserts that every exported field
// has a non-zero value. This is the CI-enforced mechanism B1-4
// MD-4 committed for the every-field-in-every-env rule: a
// future contributor who adds a field to EnvConfig but forgets
// to populate it in (say) qa.go will see this test fail in CI.
//
// "Non-zero" is the operative bar — placeholder string values
// like "dq-qa-PLACEHOLDER-rules" are non-zero and pass; an
// accidental empty-string slip does not. For numeric / Duration
// fields, a zero value (0 ns) is also a real signal that the
// field was forgotten — local should not have zero refresh
// interval.
func TestExhaustive_AllFieldsPopulatedInAllEnvs(t *testing.T) {
	envs := map[Name]EnvConfig{
		NameLocal: Local,
		NameQA:    QA,
		NameProd:  Prod,
	}
	for name, cfg := range envs {
		v := reflect.ValueOf(cfg)
		typ := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := typ.Field(i)
			if v.Field(i).IsZero() {
				t.Errorf("env %q: field %q is the zero value; "+
					"every EnvConfig field must be populated in every per-env declaration (B1-4 MD-4)",
					name, field.Name)
			}
		}
	}
}

func TestSelect_RecognizedNames(t *testing.T) {
	cases := []struct {
		input string
		want  EnvConfig
	}{
		{"local", Local},
		{"qa", QA},
		{"prod", Prod},
	}
	for _, tc := range cases {
		got, err := Select(tc.input)
		if err != nil {
			t.Errorf("Select(%q) returned error: %v", tc.input, err)
			continue
		}
		if got.Name != tc.want.Name {
			t.Errorf("Select(%q).Name = %q; want %q", tc.input, got.Name, tc.want.Name)
		}
	}
}

func TestSelect_UnknownName_ReturnsErrUnknownEnv(t *testing.T) {
	// Each rejection input exercises a different typo / case
	// pattern the case-sensitive matcher must reject so a future
	// contributor who accidentally relaxes Select to e.g.
	// strings.ToLower or strings.TrimSpace produces a visible
	// test failure here.
	for _, input := range []string{
		"staging",   // unknown env name
		"dev",       // unknown env name
		"LOCAL",     // full-uppercase
		"Local",     // capitalized first letter
		"loCal",     // interior mixed case
		" local",    // leading whitespace
		"local ",    // trailing whitespace
		"\tlocal",   // leading tab
	} {
		_, err := Select(input)
		if !errors.Is(err, ErrUnknownEnv) {
			t.Errorf("Select(%q) error = %v; want wrap of ErrUnknownEnv", input, err)
		}
	}
}

func TestSelect_EmptyName_ReturnsErrUnknownEnv(t *testing.T) {
	_, err := Select("")
	if !errors.Is(err, ErrUnknownEnv) {
		t.Errorf("Select(\"\") error = %v; want wrap of ErrUnknownEnv", err)
	}
}

func TestLogLevel_Slog_Recognized(t *testing.T) {
	cases := []struct {
		level LogLevel
		want  slog.Level
	}{
		{LogLevelDebug, slog.LevelDebug},
		{LogLevelInfo, slog.LevelInfo},
		{LogLevelWarn, slog.LevelWarn},
		{LogLevelError, slog.LevelError},
	}
	for _, tc := range cases {
		got, err := tc.level.Slog()
		if err != nil {
			t.Errorf("LogLevel(%q).Slog() returned error: %v", tc.level, err)
			continue
		}
		if got != tc.want {
			t.Errorf("LogLevel(%q).Slog() = %v; want %v", tc.level, got, tc.want)
		}
	}
}

func TestLogLevel_Slog_Unknown(t *testing.T) {
	for _, input := range []LogLevel{"", "verbose", "trace", "INFO"} {
		_, err := input.Slog()
		if err == nil {
			t.Errorf("LogLevel(%q).Slog() expected error; got nil", input)
		}
	}
}
