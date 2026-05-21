// path: engine/internal/alerts/source.go

package alerts

// EventSource is the closed enum for the `event_source` field
// per ADR-0006 CC4. The emitter sets this; consumer-side dedup
// keys on (execution_id, event_source) for execution-level
// events so different engine components reporting failures of
// the same execution are surfaced separately.
type EventSource string

const (
	// SourceRunner is set by the runner package on check-level
	// and execution-level alerts emitted during normal trigger
	// evaluation.
	SourceRunner EventSource = "runner"

	// SourceLoader is set by the engine binary when emitting
	// operational alerts for loader refresh failures (ADR-0007
	// CC2). Reserved for follow-up wiring; Phase 5 does not
	// invoke it from the loader package.
	SourceLoader EventSource = "loader"

	// SourceScheduler is set when emitting alerts for
	// scheduler-reconciliation failures (ADR-0007 CC4).
	// Reserved for follow-up wiring.
	SourceScheduler EventSource = "scheduler"

	// SourceOrphanDetector is set by the orphan detector when
	// publishing the operational alert that accompanies the
	// follow-up `aborted` row per ADR-0007 CC11.
	SourceOrphanDetector EventSource = "orphan_detector"

	// SourceTriggerHandler is set when the trigger handler
	// emits an exhaustion alert (ADR-0007 CC6). Reserved for
	// follow-up wiring once the trigger handler lands.
	SourceTriggerHandler EventSource = "trigger_handler"
)
