// path: engine/internal/api/dto.go

package api

// TriggerHTTPRequest is the wire-layer JSON body for POST
// /v1/trigger. The four input fields are the API surface for the
// five-input execution_id formula committed by ADR-0002 §1; the
// fifth input (ruleset_version) is sourced from the engine's
// active manifest at trigger acceptance per ADR-0007 §3.
//
// All fields are required. Unknown JSON fields are rejected with
// 400 per ADR-0014 §2 (forward-incompatibility protection).
type TriggerHTTPRequest struct {
	// Entity is the entity identifier for the run. Must be valid
	// UTF-8, free of the ASCII pipe character (ADR-0002 §2), and
	// within the per-field length ceiling.
	Entity string `json:"entity"`

	// WindowStart is the execution window's lower bound. The wire
	// format is RFC 3339 UTC with the literal Z suffix per
	// ADR-0014 §2 (Z-only — `+00:00` is rejected to keep the
	// execution_id formula's byte-equality contract per ADR-0002
	// §1).
	WindowStart string `json:"window_start"`

	// WindowEnd is the execution window's upper bound, same
	// format constraints as WindowStart. Must be strictly after
	// WindowStart.
	WindowEnd string `json:"window_end"`

	// TriggerSource is one of the closed enum values
	// {"scheduler", "manual"}. `operator-rerun` is the Admin API
	// path's exclusive source per ADR-0002 §4 — the data-plane
	// path rejects it with 400.
	TriggerSource string `json:"trigger_source"`
}

// TriggerHTTPResponse is the v1 response DTO returned on 200 OK
// per ADR-0014 §3. It is intentionally distinct from
// results.ExecutionRow: the storage contract (ADR-0003) and the
// response contract evolve under separate channels per P5.
//
// The DTO is returned at the point of acceptance — `Status` is
// always the literal "running" per ADR-0014 §3. The terminal
// status materializes asynchronously in dq_executions and is read
// out of band by future read-API surfaces.
type TriggerHTTPResponse struct {
	// ExecutionID is the 64-char lowercase hex value from the
	// ADR-0002 §1 formula computed by the handler at acceptance.
	ExecutionID string `json:"execution_id"`

	// AttemptID is the UUID minted by the handler at acceptance
	// per ADR-0003 §4. The runner uses this same value when
	// writing the running and terminal rows so the DTO and the
	// persisted rows carry the same identifier.
	AttemptID string `json:"attempt_id"`

	// Status is always "running" at acceptance per ADR-0014 §3.
	Status string `json:"status"`

	// AcceptedAt is the handler-side timestamp, RFC 3339 UTC with
	// the literal Z suffix. Distinct from the persistence
	// `started_at` column per ADR-0014 §3 (the two may diverge
	// if plan creation has any meaningful latency).
	AcceptedAt string `json:"accepted_at"`

	// Self is the relative URL fragment locating the execution's
	// later state, of the shape `/v1/executions/{execution_id}`.
	// The read API itself is out of scope (ADR-0014 Context); the
	// fragment is published so consumers can construct future
	// reads once that API lands.
	Self string `json:"self"`
}

// ErrorResponse is the structured envelope returned with 400
// responses per ADR-0014 §"Consequences" item 6. The Code values used here
// are working placeholders pending the OQ-MD-2.1 follow-up
// amendment that fixes the public taxonomy.
type ErrorResponse struct {
	// Code identifies the rejection reason. Subject to taxonomy
	// stabilization per ADR-0014 OQ-MD-2.1.
	Code string `json:"code"`

	// Message is a human-readable description of the rejection.
	Message string `json:"message"`

	// Field, when set, is the JSON pointer or field name that
	// caused the rejection. Omitted when the rejection is not
	// attributable to a single field (e.g., decode-level
	// failures).
	Field string `json:"field,omitempty"`
}

// Error-code constants. These are placeholders pending ADR-0014
// OQ-MD-2.1 taxonomy ratification; the implementation cites them
// at every rejection site so the taxonomy can be updated in one
// place when the amendment lands.
const (
	ErrCodeUnknownField        = "UNKNOWN_FIELD"
	ErrCodeDecodeError         = "DECODE_ERROR"
	ErrCodeMissingField        = "MISSING_FIELD"
	ErrCodeInvalidUTF8         = "INVALID_UTF8"
	ErrCodePipeInInput         = "PIPE_IN_INPUT"
	ErrCodeInvalidFieldLength  = "INVALID_FIELD_LENGTH"
	ErrCodeInvalidWindowFormat = "INVALID_WINDOW_FORMAT"
	ErrCodeInvalidWindowOrder  = "INVALID_WINDOW_ORDER"
	ErrCodeInvalidTriggerSrc   = "INVALID_TRIGGER_SOURCE"
	ErrCodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	ErrCodeInternal            = "INTERNAL_ERROR"
)
