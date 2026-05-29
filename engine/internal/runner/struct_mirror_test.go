// path: engine/internal/runner/struct_mirror_test.go

// This file lives in package runner_test (external test package) on
// purpose: dsl/spec imports runner (for runner.CheckSpec via
// spec.ToCheckSpecs), so an internal-test-package import of dsl/spec
// would close an import cycle. The external test package compiles as
// a separate unit and breaks the cycle while still letting one
// reflection sweep see both struct shapes.
package runner_test

import (
	"reflect"
	"testing"

	"dq-platform/engine/internal/dsl/spec"
	"dq-platform/engine/internal/runner"
)

// TestExhaustive_RuleSourceMirrorsDSLSpecSource and its sibling
// below guard the documented coupling discipline that keeps the
// runner package free of dsl/spec dependencies at runtime (see the
// doc comment on runner.RuleSource and record_runner.go's note that
// "the runner package does not depend on dsl/spec so the engine
// binary translates at boot"). The shapes are duplicated by design;
// the price of that duplication is that a new field on dsl/spec.Source
// must be propagated to runner.RuleSource by hand. Without a gate,
// the propagation step is invisible until something downstream breaks.
//
// The test is reflection-based on purpose: it asserts that the field
// set of one side matches the field set of the other, by name and
// by type, with one carve-out for the deliberately-distinct boundary
// type (the Window pointer field, where each side references its
// own package-local Window type — that interior shape is covered by
// the second test).
//
// Failure mode: a contributor who adds a field to dsl/spec.Source
// but forgets the corresponding addition on runner.RuleSource (or
// vice versa) gets a CI failure here, naming the orphan field and
// the side it was missing from.
//
// Status under R5 (CLAUDE.md §3): the *contract* being protected
// (foundation 04 coupling discipline) is pre-existing; the *form*
// of the protection (a reflection-based mirror test, external test
// package to dodge the import cycle) is a new contribution proposed
// here, requires review.
func TestExhaustive_RuleSourceMirrorsDSLSpecSource(t *testing.T) {
	runnerType := reflect.TypeOf(runner.RuleSource{})
	specType := reflect.TypeOf(spec.Source{})

	runnerFields := fieldMap(runnerType)
	specFields := fieldMap(specType)

	for name, specField := range specFields {
		runnerField, ok := runnerFields[name]
		if !ok {
			t.Errorf("dsl/spec.Source has field %q (type %s) "+
				"but runner.RuleSource does not — propagate the field "+
				"to keep the runner ↔ dsl/spec boundary contract honest "+
				"(see runner.RuleSource doc comment; foundation 04 coupling discipline)",
				name, specField.Type)
			continue
		}
		assertFieldTypesMatch(t, "RuleSource", name, runnerField.Type, specField.Type)
	}
	for name, runnerField := range runnerFields {
		if _, ok := specFields[name]; !ok {
			t.Errorf("runner.RuleSource has field %q (type %s) "+
				"but dsl/spec.Source does not — either remove the orphan "+
				"or add the matching field on the dsl/spec side "+
				"(see runner.RuleSource doc comment; foundation 04 coupling discipline)",
				name, runnerField.Type)
		}
	}
}

// TestExhaustive_RuleWindowMirrorsDSLSpecWindow covers the interior
// shape that the RuleSource test deliberately delegates here. Same
// failure mode and same rationale; no carve-out needed because every
// field on both sides is a string today.
func TestExhaustive_RuleWindowMirrorsDSLSpecWindow(t *testing.T) {
	runnerType := reflect.TypeOf(runner.RuleWindow{})
	specType := reflect.TypeOf(spec.Window{})

	runnerFields := fieldMap(runnerType)
	specFields := fieldMap(specType)

	for name, specField := range specFields {
		runnerField, ok := runnerFields[name]
		if !ok {
			t.Errorf("dsl/spec.Window has field %q (type %s) "+
				"but runner.RuleWindow does not — propagate the field "+
				"to keep the runner ↔ dsl/spec boundary contract honest",
				name, specField.Type)
			continue
		}
		if runnerField.Type != specField.Type {
			t.Errorf("RuleWindow field %q: runner type %s, dsl/spec type %s — types must match",
				name, runnerField.Type, specField.Type)
		}
	}
	for name, runnerField := range runnerFields {
		if _, ok := specFields[name]; !ok {
			t.Errorf("runner.RuleWindow has field %q (type %s) "+
				"but dsl/spec.Window does not — either remove the orphan "+
				"or add the matching field on the dsl/spec side",
				name, runnerField.Type)
		}
	}
}

func fieldMap(t reflect.Type) map[string]reflect.StructField {
	m := make(map[string]reflect.StructField, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		m[f.Name] = f
	}
	return m
}

// assertFieldTypesMatch enforces type equality between paired fields,
// with one deliberate carve-out: the Window pointer field on RuleSource
// references runner.RuleWindow on one side and dsl/spec.Window on the
// other. Those are intentionally distinct named types (the duplication
// the second test exists to keep aligned). The carve-out is keyed on
// field name + pointer kind + element type names so a future rename of
// the field or a change of kind reopens the gate. If the duplication
// is ever collapsed into a single shared type, delete this carve-out
// and let the test enforce strict type equality everywhere.
func assertFieldTypesMatch(t *testing.T, parent, name string, runnerType, specType reflect.Type) {
	t.Helper()
	if runnerType == specType {
		return
	}
	if name == "Window" &&
		runnerType.Kind() == reflect.Pointer &&
		specType.Kind() == reflect.Pointer &&
		runnerType.Elem().Name() == "RuleWindow" &&
		specType.Elem().Name() == "Window" {
		return
	}
	t.Errorf("%s field %q: runner type %s, dsl/spec type %s — types must match "+
		"(the only sanctioned divergence is the Window pointer pair, covered separately)",
		parent, name, runnerType, specType)
}
