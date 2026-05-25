// path: engine/internal/dsl/spec/spec.go

package spec

import (
	"dq-platform/engine/internal/runner"
)

// Schema versions the parser understands. v1 is the legacy
// shape; v2 carries mode, source, and per-check params per
// ADRs 0021–0024.
const (
	SchemaVersionV1 = 1
	SchemaVersionV2 = 2
)

// Mode values committed by ADR-0021. Mode is required on every
// v2 rule; v1 rules carry no mode and are treated as set-mode
// at translation time.
const (
	ModeSet    = "set"
	ModeRecord = "record"
)

// Source.Type values committed by ADR-0023. v1 rules have no
// source descriptor; v2 rules carry exactly one.
const (
	SourceTypeBigQuery = "bigquery"
	SourceTypeKafka    = "kafka"
)

// Window.Type values committed by ADR-0024. Only "tumbling" is
// supported at v2; future window types extend the enum
// additively per ADR-0001.
const (
	WindowTypeTumbling = "tumbling"
)

// RuleSpec is the parsed form of one rule YAML body. The field
// set covers v1 (legacy) and v2 (ADRs 0021–0024); per-version
// invariants are enforced by validate() in parse.go.
//
// v1 → v2 evolution:
//
//   - v2 requires Mode (ADR-0021) and Source (ADR-0023).
//   - v2 per-check Kind carries a mode prefix (ADR-0022).
//   - v2 per-check Params is an open map; per-kind shape is
//     validated by the linter against the catalog
//     (engine/internal/dsl/catalog/v1.yaml).
//
// Unknown fields are rejected at parse time via the YAML
// decoder's KnownFields(true) mode.
type RuleSpec struct {
	Version     int     `yaml:"version"`
	Entity      string  `yaml:"entity"`
	Mode        string  `yaml:"mode,omitempty"`
	Source      *Source `yaml:"source,omitempty"`
	Description string  `yaml:"description,omitempty"`
	Checks      []Check `yaml:"checks"`
}

// Source is the per-rule substrate descriptor committed by
// ADR-0023. Exactly one of BigQuery / Kafka is populated; the
// parser enforces the cross-field alignment with RuleSpec.Mode
// (mode=set ⇔ bigquery; mode=record ⇔ kafka).
//
// The substrate-specific addresses live inline on Source rather
// than nested under per-type sub-objects because v2 only has two
// substrates and inline fields keep the parsed shape flat for
// handler consumption. Future substrates extend additively with
// new Type discriminator values; the field set grows by one
// optional block per substrate, mirroring the schema's oneOf.
type Source struct {
	Type string `yaml:"type"`

	// BigQuery fields (set-mode; ADR-0023).
	ProjectID       string `yaml:"project_id,omitempty"`
	DatasetID       string `yaml:"dataset_id,omitempty"`
	TableID         string `yaml:"table_id,omitempty"`
	PartitionColumn string `yaml:"partition_column,omitempty"`

	// Kafka fields (record-mode; ADR-0024).
	Topic         string  `yaml:"topic,omitempty"`
	ConsumerGroup string  `yaml:"consumer_group,omitempty"`
	Window        *Window `yaml:"window,omitempty"`
}

// Window is the record-mode window descriptor committed by
// ADR-0024. v2 supports tumbling windows only; non-tumbling
// boundaries extend the Type enum additively.
//
// Duration and LatenessTolerance use the duration grammar
// validated at the linter layer (regex `^[0-9]+(ms|s|m|h)$`).
// The engine parses them into time.Duration at runtime when
// the record-mode runner consumes the rule.
type Window struct {
	Type              string `yaml:"type"`
	Duration          string `yaml:"duration"`
	LatenessTolerance string `yaml:"lateness_tolerance"`
}

// Check is the per-check object inside RuleSpec.Checks. Params
// is the v2 open map per ADR-0022 — the engine handler reads
// the per-kind shape (e.g., record.schema_conformance reads
// params.schema), and the linter validates the shape against
// the catalog's params_schema.
type Check struct {
	CheckID     string         `yaml:"check_id"`
	Kind        string         `yaml:"kind"`
	Description string         `yaml:"description,omitempty"`
	Params      map[string]any `yaml:"params,omitempty"`
}

// ToCheckSpecs translates the parsed rule's checks into the
// runner.CheckSpec slice the runner consumes via TriggerRequest.
// The runner reads CheckID, Kind, Mode, Source, and Params; the
// Description field is dropped on translation but preserved on
// the in-package RuleSpec for tooling that surfaces per-check
// documentation.
//
// For v1 rules (no Mode, no Source), the returned CheckSpecs
// carry Mode = "set" and Source = nil. The set-mode evaluator
// then sources its BigQuery target from the rule's Source when
// v2, or returns ResultError when Source is nil — the trigger
// handler is expected to pass v2 rules in steady state once the
// rule migration lands.
func (r RuleSpec) ToCheckSpecs() []runner.CheckSpec {
	if len(r.Checks) == 0 {
		return nil
	}
	mode := r.Mode
	if mode == "" {
		mode = ModeSet
	}
	var source *runner.RuleSource
	if r.Source != nil {
		source = &runner.RuleSource{
			Type:            r.Source.Type,
			ProjectID:       r.Source.ProjectID,
			DatasetID:       r.Source.DatasetID,
			TableID:         r.Source.TableID,
			PartitionColumn: r.Source.PartitionColumn,
			Topic:           r.Source.Topic,
			ConsumerGroup:   r.Source.ConsumerGroup,
		}
		if r.Source.Window != nil {
			source.Window = &runner.RuleWindow{
				Type:              r.Source.Window.Type,
				Duration:          r.Source.Window.Duration,
				LatenessTolerance: r.Source.Window.LatenessTolerance,
			}
		}
	}
	specs := make([]runner.CheckSpec, 0, len(r.Checks))
	for _, c := range r.Checks {
		specs = append(specs, runner.CheckSpec{
			CheckID: c.CheckID,
			Kind:    c.Kind,
			Mode:    mode,
			Source:  source,
			Params:  c.Params,
		})
	}
	return specs
}
