// path: engine/internal/dsl/spec/spec.go

package spec

import (
	"dq-platform/engine/internal/runner"
)

// RuleSpec is the parsed form of one rule YAML body. Field names
// and YAML tags mirror the v1 JSON schema
// (engine/internal/dsl/schema/v1.schema.json).
type RuleSpec struct {
	Version     int       `yaml:"version"`
	Entity      string    `yaml:"entity"`
	Description string    `yaml:"description,omitempty"`
	Checks      []Check   `yaml:"checks"`
}

// Check is the per-check object inside RuleSpec.Checks.
type Check struct {
	CheckID     string `yaml:"check_id"`
	Kind        string `yaml:"kind"`
	Description string `yaml:"description,omitempty"`
}

// ToCheckSpecs translates the parsed rule's checks into the
// runner.CheckSpec slice the runner consumes via TriggerRequest.
// The runner only needs CheckID + Kind today (W3-P4c contract); the
// Description field is dropped on translation but preserved on the
// in-package RuleSpec for future-facing tooling that may surface
// per-check documentation.
func (r RuleSpec) ToCheckSpecs() []runner.CheckSpec {
	if len(r.Checks) == 0 {
		return nil
	}
	specs := make([]runner.CheckSpec, 0, len(r.Checks))
	for _, c := range r.Checks {
		specs = append(specs, runner.CheckSpec{
			CheckID: c.CheckID,
			Kind:    c.Kind,
		})
	}
	return specs
}
