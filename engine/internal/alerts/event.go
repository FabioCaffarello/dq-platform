// path: engine/internal/alerts/event.go

package alerts

import (
	"time"

	"dq-platform/engine/internal/results"
)

// Event matches the ADR-0006 §4 alert payload schema. Fields
// marked optional in the ADR are pointer-typed so JSON
// serialization omits absent values via `omitempty`.
//
// Consumers must tolerate unknown fields per ADR-0006 §4
// (additive evolution policy). The engine emits exactly one
// Event per alert.
type Event struct {
	// Identity fields per ADR-0006 §4.
	ExecutionID *string `json:"execution_id,omitempty"` // absent for engine-startup events
	AttemptID   *string `json:"attempt_id,omitempty"`   // absent when ExecutionID absent
	Entity      string  `json:"entity"`                 // required (must match _owners.yaml entry)
	CheckID     *string `json:"check_id,omitempty"`     // present for check-level events

	// Routing fields per ADR-0006 §4.
	Category    Category    `json:"category"`           // required (ADR-0006 CC7)
	Severity    *Severity   `json:"severity,omitempty"` // omitted when no _owners.yaml override matches
	EventSource EventSource `json:"event_source"`       // required (ADR-0006 CC4)

	// Status / result fields per ADR-0006 §4. Exactly one of
	// these is set for runner-sourced events (Result for
	// check-level, Status for execution-level). Non-runner
	// sources may set neither (loader / scheduler / trigger-
	// handler startup-style events).
	Result *results.CheckResult     `json:"result,omitempty"`
	Status *results.ExecutionStatus `json:"status,omitempty"`

	// Context fields.
	RecordedAt   time.Time `json:"recorded_at"`             // required, RFC 3339 second precision
	ErrorSummary *string   `json:"error_summary,omitempty"` // present for error / aborted / loader-failure events
}
