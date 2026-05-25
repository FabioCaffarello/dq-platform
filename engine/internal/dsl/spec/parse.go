// path: engine/internal/dsl/spec/parse.go

package spec

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// maxIdentifierLen is the per-field length ceiling for the entity
// and check_id strings, matching the v1 + v2 JSON schemas'
// maxLength: 200.
const maxIdentifierLen = 200

// identifierPattern matches the entity / check_id pattern from
// engine/internal/dsl/schema/v{1,2}.schema.json: ASCII
// alphanumeric plus underscore, dot, hyphen. Pipe is forbidden
// per ADR-0002 §2 input safety (the rule's entity flows into the
// execution_id formula).
var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// v2 kind pattern enforces the mode prefix committed by ADR-0022.
// The linter cross-checks against the catalog; the parser enforces
// the lexical shape so the dispatch downstream is safe.
var v2KindPattern = regexp.MustCompile(`^(set|record)\.[A-Za-z0-9_]+$`)

// v2 duration pattern matches the lexical form committed by
// ADR-0024 for source.kafka.window.duration and
// lateness_tolerance. The engine parses these at trigger time;
// the parser only enforces the lexical contract.
var v2DurationPattern = regexp.MustCompile(`^[0-9]+(ms|s|m|h)$`)

// v2 BigQuery identifier patterns mirror the v2 JSON schema.
var (
	v2BigQueryProjectPattern  = regexp.MustCompile(`^[a-z0-9-]+$`)
	v2BigQueryDatasetPattern  = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	v2BigQueryTablePattern    = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	v2BigQueryColumnPattern   = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	v2KafkaTopicPattern       = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	v2KafkaConsumerGrpPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

// Parse reads a rule YAML body and returns a validated RuleSpec.
// Unknown top-level or per-check fields are rejected (strict
// decode via KnownFields(true)), and per-version invariants are
// enforced before the spec is returned.
//
// The parser does not validate `kind` against the catalog; the
// linter does (engine/internal/dsl/catalog/v1.yaml is the
// source of truth). The handler registry's startup invariant
// (cmd/dq-engine) ensures every catalog kind has a registered
// handler; unregistered kinds surface as ResultError per
// ADR-0004 CC1 at evaluation time.
func Parse(body []byte) (RuleSpec, error) {
	var r RuleSpec
	dec := yaml.NewDecoder(bytes.NewReader(body))
	dec.KnownFields(true)
	if err := dec.Decode(&r); err != nil {
		return RuleSpec{}, fmt.Errorf("rule yaml decode: %w", err)
	}

	if err := validate(&r); err != nil {
		return RuleSpec{}, err
	}
	return r, nil
}

func validate(r *RuleSpec) error {
	switch r.Version {
	case SchemaVersionV1:
		return validateV1(r)
	case SchemaVersionV2:
		return validateV2(r)
	default:
		return fmt.Errorf("rule yaml: version %d is not supported (engine accepts v1 or v2 per ADR-0001)", r.Version)
	}
}

// validateV1 enforces the v1 invariants. v1 rules must not
// carry v2-only fields; if they do, the strict decoder will
// already have rejected them as unknown fields per the v1
// shape's lack of Mode / Source. But the spec struct now has
// optional Mode / Source fields (so v2 can decode), so we add
// explicit per-version rejection here.
func validateV1(r *RuleSpec) error {
	if r.Mode != "" {
		return errors.New("rule yaml: v1 must not carry `mode` (v2 only; ADR-0021)")
	}
	if r.Source != nil {
		return errors.New("rule yaml: v1 must not carry `source` (v2 only; ADR-0023)")
	}
	if err := validateIdentifier("entity", r.Entity); err != nil {
		return err
	}
	if len(r.Checks) == 0 {
		return errors.New("rule yaml: checks must be a non-empty array")
	}
	for i, c := range r.Checks {
		if err := validateCheckV1(i, c); err != nil {
			return err
		}
	}
	return nil
}

// validateV2 enforces ADRs 0021–0024 invariants: required Mode,
// required Source, source.type alignment with mode, kind prefix
// alignment with mode, per-check params shape (open map; the
// linter validates against the catalog).
func validateV2(r *RuleSpec) error {
	if err := validateIdentifier("entity", r.Entity); err != nil {
		return err
	}
	switch r.Mode {
	case ModeSet, ModeRecord:
		// ok
	case "":
		return errors.New("rule yaml: v2 requires `mode` (one of set|record per ADR-0021)")
	default:
		return fmt.Errorf("rule yaml: mode %q is not one of [set, record] (ADR-0021)", r.Mode)
	}
	if r.Source == nil {
		return errors.New("rule yaml: v2 requires `source` (per ADR-0023)")
	}
	if err := validateSourceV2(r.Mode, r.Source); err != nil {
		return err
	}
	if len(r.Checks) == 0 {
		return errors.New("rule yaml: checks must be a non-empty array")
	}
	for i, c := range r.Checks {
		if err := validateCheckV2(i, c, r.Mode); err != nil {
			return err
		}
	}
	return nil
}

func validateSourceV2(mode string, s *Source) error {
	expectedType, ok := expectedSourceTypeForMode(mode)
	if !ok {
		return fmt.Errorf("rule yaml: unknown mode %q (cannot align with source.type)", mode)
	}
	if s.Type != expectedType {
		return fmt.Errorf("rule yaml: mode %q requires source.type %q but found %q (ADR-0023 cross-check #7)",
			mode, expectedType, s.Type)
	}
	switch s.Type {
	case SourceTypeBigQuery:
		return validateBigQuerySource(s)
	case SourceTypeKafka:
		return validateKafkaSource(s)
	default:
		return fmt.Errorf("rule yaml: source.type %q is not one of [bigquery, kafka] (ADR-0023)", s.Type)
	}
}

func validateBigQuerySource(s *Source) error {
	if s.ProjectID == "" || !v2BigQueryProjectPattern.MatchString(s.ProjectID) {
		return fmt.Errorf("rule yaml: source.project_id %q does not match %s", s.ProjectID, v2BigQueryProjectPattern.String())
	}
	if s.DatasetID == "" || !v2BigQueryDatasetPattern.MatchString(s.DatasetID) {
		return fmt.Errorf("rule yaml: source.dataset_id %q does not match %s", s.DatasetID, v2BigQueryDatasetPattern.String())
	}
	if s.TableID == "" || !v2BigQueryTablePattern.MatchString(s.TableID) {
		return fmt.Errorf("rule yaml: source.table_id %q does not match %s", s.TableID, v2BigQueryTablePattern.String())
	}
	if s.PartitionColumn != "" && !v2BigQueryColumnPattern.MatchString(s.PartitionColumn) {
		return fmt.Errorf("rule yaml: source.partition_column %q does not match %s", s.PartitionColumn, v2BigQueryColumnPattern.String())
	}
	// Kafka-only fields must be empty on a BigQuery source.
	if s.Topic != "" || s.ConsumerGroup != "" || s.Window != nil {
		return errors.New("rule yaml: bigquery source must not carry kafka fields (topic/consumer_group/window)")
	}
	return nil
}

func validateKafkaSource(s *Source) error {
	if s.Topic == "" || !v2KafkaTopicPattern.MatchString(s.Topic) {
		return fmt.Errorf("rule yaml: source.topic %q does not match %s", s.Topic, v2KafkaTopicPattern.String())
	}
	if s.ConsumerGroup == "" || !v2KafkaConsumerGrpPattern.MatchString(s.ConsumerGroup) {
		return fmt.Errorf("rule yaml: source.consumer_group %q does not match %s", s.ConsumerGroup, v2KafkaConsumerGrpPattern.String())
	}
	if s.Window == nil {
		return errors.New("rule yaml: kafka source requires `window` (ADR-0024)")
	}
	if err := validateWindow(s.Window); err != nil {
		return err
	}
	// BigQuery-only fields must be empty on a Kafka source.
	if s.ProjectID != "" || s.DatasetID != "" || s.TableID != "" || s.PartitionColumn != "" {
		return errors.New("rule yaml: kafka source must not carry bigquery fields (project_id/dataset_id/table_id/partition_column)")
	}
	return nil
}

func validateWindow(w *Window) error {
	if w.Type != WindowTypeTumbling {
		return fmt.Errorf("rule yaml: window.type %q is not one of [tumbling] (ADR-0024)", w.Type)
	}
	if !v2DurationPattern.MatchString(w.Duration) {
		return fmt.Errorf("rule yaml: window.duration %q does not match duration grammar %s", w.Duration, v2DurationPattern.String())
	}
	if !v2DurationPattern.MatchString(w.LatenessTolerance) {
		return fmt.Errorf("rule yaml: window.lateness_tolerance %q does not match duration grammar %s", w.LatenessTolerance, v2DurationPattern.String())
	}
	return nil
}

func validateCheckV1(index int, c Check) error {
	if err := validateIdentifier(fmt.Sprintf("checks[%d].check_id", index), c.CheckID); err != nil {
		return err
	}
	if c.Kind == "" {
		return fmt.Errorf("rule yaml: checks[%d].kind must be a non-empty string", index)
	}
	if len(c.Params) != 0 {
		return fmt.Errorf("rule yaml: checks[%d].params is v2-only (ADR-0022)", index)
	}
	return nil
}

func validateCheckV2(index int, c Check, ruleMode string) error {
	if err := validateIdentifier(fmt.Sprintf("checks[%d].check_id", index), c.CheckID); err != nil {
		return err
	}
	if !v2KindPattern.MatchString(c.Kind) {
		return fmt.Errorf("rule yaml: checks[%d].kind %q does not match %s", index, c.Kind, v2KindPattern.String())
	}
	prefix := strings.SplitN(c.Kind, ".", 2)[0]
	if prefix != ruleMode {
		return fmt.Errorf("rule yaml: checks[%d].kind %q has prefix %q which does not match rule mode %q (ADR-0022 cross-check #4)",
			index, c.Kind, prefix, ruleMode)
	}
	return nil
}

// validateIdentifier applies the shared invariants for entity and
// check_id strings: non-empty, length-bounded, identifier-pattern
// match, no ASCII pipe.
func validateIdentifier(field, value string) error {
	if value == "" {
		return fmt.Errorf("rule yaml: %s must be a non-empty string", field)
	}
	if len(value) > maxIdentifierLen {
		return fmt.Errorf("rule yaml: %s exceeds %d bytes", field, maxIdentifierLen)
	}
	if strings.ContainsRune(value, '|') {
		return fmt.Errorf("rule yaml: %s contains forbidden ASCII pipe character (ADR-0002 §2)", field)
	}
	if !identifierPattern.MatchString(value) {
		return fmt.Errorf("rule yaml: %s %q does not match %s", field, value, identifierPattern.String())
	}
	return nil
}

// expectedSourceTypeForMode maps a mode value to the substrate
// descriptor type expected by ADR-0023.
func expectedSourceTypeForMode(mode string) (string, bool) {
	switch mode {
	case ModeSet:
		return SourceTypeBigQuery, true
	case ModeRecord:
		return SourceTypeKafka, true
	default:
		return "", false
	}
}
