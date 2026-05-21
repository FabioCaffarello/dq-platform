// path: engine/internal/alerts/severity.go

package alerts

// Severity is the optional severity enum on alert events per
// ADR-0006 CC6. Set when `_owners.yaml` has a matching override
// for `(entity, check_id)`; otherwise omitted from the emitted
// event so the consumer applies the per-category default from
// engine deployment config (Phase 7 follow-up).
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)
