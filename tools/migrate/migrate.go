// path: tools/migrate/migrate.go

package main

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// BigQuerySource carries the per-rule substrate-binding fields
// the v2 schema requires under `source` per ADR-0023. The v1
// rule format has no source block; the operator supplies the
// values when invoking the migrator.
type BigQuerySource struct {
	ProjectID string
	DatasetID string
	TableID   string
}

// Validate returns an error when any required field is empty.
// The v2 schema's `additionalProperties: false` + `required`
// list (rules/_schema/v2.schema.json) makes all three fields
// mandatory; missing values would produce a lint failure
// downstream, so we fail loud here instead.
func (s BigQuerySource) Validate() error {
	if s.ProjectID == "" {
		return errors.New("BigQuerySource: project_id is required")
	}
	if s.DatasetID == "" {
		return errors.New("BigQuerySource: dataset_id is required")
	}
	if s.TableID == "" {
		return errors.New("BigQuerySource: table_id is required")
	}
	return nil
}

// v1Doc is the typed shape of a v1 rule for the migrator.
// Mirrors `engine/internal/dsl/spec`'s v1 fields plus the
// `version` discriminator. Unknown fields are NOT
// rejected here — the migrator's job is to emit a v2 YAML;
// upstream `dq-lint` is the gate for "is this v1 well-formed".
type v1Doc struct {
	Version     int       `yaml:"version"`
	Entity      string    `yaml:"entity"`
	Description string    `yaml:"description,omitempty"`
	Checks      []v1Check `yaml:"checks"`
}

type v1Check struct {
	CheckID     string         `yaml:"check_id"`
	Kind        string         `yaml:"kind"`
	Description string         `yaml:"description,omitempty"`
	Params      map[string]any `yaml:"params,omitempty"`
}

// v2Doc is the typed shape the migrator emits. Field
// ordering on the wire matches the rules/customer.yaml v2
// production migration (B2-19 PR #72) so review diffs are
// minimal for operators familiar with the existing rule.
type v2Doc struct {
	Version     int       `yaml:"version"`
	Entity      string    `yaml:"entity"`
	Mode        string    `yaml:"mode"`
	Description string    `yaml:"description,omitempty"`
	Source      v2Source  `yaml:"source"`
	Checks      []v2Check `yaml:"checks"`
}

type v2Source struct {
	Type      string `yaml:"type"`
	ProjectID string `yaml:"project_id"`
	DatasetID string `yaml:"dataset_id"`
	TableID   string `yaml:"table_id"`
}

type v2Check struct {
	CheckID     string         `yaml:"check_id"`
	Kind        string         `yaml:"kind"`
	Description string         `yaml:"description,omitempty"`
	Params      map[string]any `yaml:"params,omitempty"`
}

// V1ToV2 transforms a v1 rule YAML body into its v2 equivalent.
// The transform is mechanical:
//
//   - `version: 1` → `version: 2`
//   - new `mode: set` (the only mode v1 implicitly supported was
//     BigQuery-backed set-mode; record-mode is v2-only per
//     ADR-0021)
//   - new `source` block populated from the caller-supplied
//     BigQuerySource per ADR-0023
//   - each check's `kind` value is prefixed `set.` per ADR-0022
//     unless the value already carries the prefix (defensive;
//     v1 schema rejected the prefix so this branch should not
//     fire in practice)
//
// `entity`, top-level `description`, and per-check fields
// (check_id, description, params) are copied byte-equivalently.
//
// Errors fire for: non-v1 input (caller should not invoke
// V1ToV2 on v2+); missing entity / checks; empty / invalid
// BigQuerySource.
func V1ToV2(rawV1 []byte, src BigQuerySource) ([]byte, error) {
	if err := src.Validate(); err != nil {
		return nil, err
	}

	var in v1Doc
	dec := yaml.NewDecoder(strings.NewReader(string(rawV1)))
	dec.KnownFields(true)
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("V1ToV2: parse v1 yaml: %w", err)
	}

	if in.Version != 1 {
		return nil, fmt.Errorf("V1ToV2: input declares version %d; want 1 (v2+ rules need no migration to v2)", in.Version)
	}
	if in.Entity == "" {
		return nil, errors.New("V1ToV2: input has empty entity")
	}
	if len(in.Checks) == 0 {
		return nil, errors.New("V1ToV2: input has empty checks array (v2 schema requires at least one)")
	}

	out := v2Doc{
		Version:     2,
		Entity:      in.Entity,
		Mode:        "set",
		Description: in.Description,
		Source: v2Source{
			Type:      "bigquery",
			ProjectID: src.ProjectID,
			DatasetID: src.DatasetID,
			TableID:   src.TableID,
		},
		Checks: make([]v2Check, 0, len(in.Checks)),
	}
	for _, c := range in.Checks {
		if c.CheckID == "" || c.Kind == "" {
			return nil, fmt.Errorf("V1ToV2: check %q has empty check_id or kind", c.CheckID)
		}
		out.Checks = append(out.Checks, v2Check{
			CheckID:     c.CheckID,
			Kind:        promoteKindToSetPrefix(c.Kind),
			Description: c.Description,
			Params:      c.Params,
		})
	}

	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(out); err != nil {
		return nil, fmt.Errorf("V1ToV2: emit v2 yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("V1ToV2: close encoder: %w", err)
	}
	return []byte(buf.String()), nil
}

// promoteKindToSetPrefix prepends `set.` to a v1 kind value
// unless it already carries the prefix. v1's catalog accepted
// only unprefixed kinds (`row_count_positive`); v2 per
// ADR-0022 requires mode-prefixed kinds. The defensive branch
// (kind already prefixed) handles operator-mid-migration
// edits where the source file declared `version: 1` but the
// kind was already updated; the migrator silently accepts it.
func promoteKindToSetPrefix(kind string) string {
	if strings.HasPrefix(kind, "set.") || strings.HasPrefix(kind, "record.") {
		return kind
	}
	return "set." + kind
}
