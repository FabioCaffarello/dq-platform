// path: engine/internal/api/decoder.go

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"dq-platform/engine/internal/results"
)

// maxStringFieldLen caps every wire-layer string input. The
// ceiling protects against pathologically large payloads at the
// formula safety boundary committed by ADR-0014 §2 (per-field
// length ceiling). Listener-level payload-size enforcement is
// substrate-coupled and deferred per ADR-0014 OQ-MD-2.2.
const maxStringFieldLen = 256

// timestampLayout is the canonical RFC 3339 UTC second-precision
// format with literal Z suffix, mirroring runner.timestampLayout.
// `+00:00` is byte-distinct from `Z` and would produce a different
// execution_id under the ADR-0002 §1 formula; ADR-0014 §2 commits
// the API to the Z-only form.
const timestampLayout = "2006-01-02T15:04:05Z"

// timestampPattern enforces the Z-only RFC 3339 UTC shape at the
// API boundary. The pattern admits exactly the layout
// runner.Compute serializes its window inputs through — keeping
// the API decoder and the formula in lock-step on byte-equality.
var timestampPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)

// DecodeRequest reads a TriggerHTTPRequest from body under the
// strict-decoder posture committed by ADR-0014 §2: unknown JSON
// fields are rejected; per-field invariants (UTF-8 validity,
// ASCII-pipe absence per ADR-0002 §2, Z-only RFC 3339 UTC, closed
// trigger_source enum, length ceiling) are enforced before any
// downstream use of the values.
//
// The unknown-field check uses a two-pass decode (first into
// map[string]json.RawMessage, then into the typed struct) so the
// rejection does not depend on the Go standard library's error
// wording. The set of known field names is derived via reflection
// over the struct's `json:` tags so adding a field keeps the
// check in sync automatically.
//
// On success returns the validated request and nil error. On
// failure returns the partially-populated request (caller should
// not use it) and a non-nil *ErrorResponse describing the
// rejection. The handler maps a non-nil *ErrorResponse to HTTP
// 400 with the envelope from ADR-0014 §"Consequences" item 6.
func DecodeRequest(body io.Reader) (TriggerHTTPRequest, *ErrorResponse) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return TriggerHTTPRequest{}, &ErrorResponse{
			Code:    ErrCodeDecodeError,
			Message: err.Error(),
		}
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return TriggerHTTPRequest{}, &ErrorResponse{
			Code:    ErrCodeDecodeError,
			Message: "empty request body",
		}
	}

	// First pass: identify unknown fields by inventorying the
	// top-level JSON keys without decoding their values.
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return TriggerHTTPRequest{}, &ErrorResponse{
			Code:    ErrCodeDecodeError,
			Message: err.Error(),
		}
	}
	known := knownTriggerFields()
	for k := range keys {
		if _, ok := known[k]; !ok {
			return TriggerHTTPRequest{}, &ErrorResponse{
				Code:    ErrCodeUnknownField,
				Field:   k,
				Message: fmt.Sprintf("unknown field %q", k),
			}
		}
	}

	// Second pass: decode into the typed struct.
	var req TriggerHTTPRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return req, &ErrorResponse{
			Code:    ErrCodeDecodeError,
			Message: err.Error(),
		}
	}

	if env := validate(&req); env != nil {
		return req, env
	}
	return req, nil
}

// knownTriggerFields returns the set of JSON keys
// TriggerHTTPRequest accepts, derived once via reflection over the
// struct's `json:` tags. The result is cached so per-request
// decode does not pay the reflection cost.
var knownTriggerFields = sync.OnceValue(func() map[string]struct{} {
	fields := make(map[string]struct{})
	t := reflect.TypeOf(TriggerHTTPRequest{})
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		// Strip ",omitempty" and similar tag options.
		name := strings.Split(tag, ",")[0]
		if name == "" {
			continue
		}
		fields[name] = struct{}{}
	}
	return fields
})

// validate runs the per-field invariants from ADR-0014 §2 against
// the decoded request. Order of checks: presence → UTF-8 → length
// → pipe → format/enum. The earliest failure is returned so a
// caller fixing one issue can iterate without first having to
// satisfy later invariants.
func validate(req *TriggerHTTPRequest) *ErrorResponse {
	// 1. Required fields present.
	if req.Entity == "" {
		return fieldErr(ErrCodeMissingField, "entity", "entity is required")
	}
	if req.WindowStart == "" {
		return fieldErr(ErrCodeMissingField, "window_start", "window_start is required")
	}
	if req.WindowEnd == "" {
		return fieldErr(ErrCodeMissingField, "window_end", "window_end is required")
	}
	if req.TriggerSource == "" {
		return fieldErr(ErrCodeMissingField, "trigger_source", "trigger_source is required")
	}

	// 2. UTF-8 validity. ADR-0002 §2 names UTF-8 as the canonical
	//    string encoding for the formula inputs; non-UTF-8 input
	//    can never produce a stable execution_id.
	if !utf8.ValidString(req.Entity) {
		return fieldErr(ErrCodeInvalidUTF8, "entity", "entity is not valid UTF-8")
	}
	// window_start, window_end, trigger_source pass UTF-8 check
	// implicitly via the timestamp/enum patterns below.

	// 3. Length ceilings. The exact ceiling is implementation
	//    policy (ADR-0014 §2 commits "per-field length ceiling"
	//    without fixing the value).
	if len(req.Entity) > maxStringFieldLen {
		return fieldErr(ErrCodeInvalidFieldLength, "entity",
			fmt.Sprintf("entity exceeds %d bytes", maxStringFieldLen))
	}
	if len(req.WindowStart) > maxStringFieldLen {
		return fieldErr(ErrCodeInvalidFieldLength, "window_start",
			fmt.Sprintf("window_start exceeds %d bytes", maxStringFieldLen))
	}
	if len(req.WindowEnd) > maxStringFieldLen {
		return fieldErr(ErrCodeInvalidFieldLength, "window_end",
			fmt.Sprintf("window_end exceeds %d bytes", maxStringFieldLen))
	}
	if len(req.TriggerSource) > maxStringFieldLen {
		return fieldErr(ErrCodeInvalidFieldLength, "trigger_source",
			fmt.Sprintf("trigger_source exceeds %d bytes", maxStringFieldLen))
	}

	// 4. ASCII-pipe ban (ADR-0002 §2). The pipe character is the
	//    formula's input separator and is forbidden in any input
	//    to preserve byte-equality of the joined string.
	if strings.ContainsRune(req.Entity, '|') {
		return fieldErr(ErrCodePipeInInput, "entity", "entity contains forbidden ASCII pipe")
	}
	// window_start, window_end, trigger_source are checked by the
	// stricter format/enum patterns below; explicit pipe-checks
	// would be redundant.

	// 5. Window-format check (Z-only RFC 3339 UTC).
	if !timestampPattern.MatchString(req.WindowStart) {
		return fieldErr(ErrCodeInvalidWindowFormat, "window_start",
			"window_start must match YYYY-MM-DDTHH:MM:SSZ (Z-only RFC 3339 UTC)")
	}
	if !timestampPattern.MatchString(req.WindowEnd) {
		return fieldErr(ErrCodeInvalidWindowFormat, "window_end",
			"window_end must match YYYY-MM-DDTHH:MM:SSZ (Z-only RFC 3339 UTC)")
	}

	// 6. Window-order check. The execution_id formula does not
	//    enforce ordering; the contract does (and the runner
	//    double-checks).
	startTime, _ := time.Parse(timestampLayout, req.WindowStart)
	endTime, _ := time.Parse(timestampLayout, req.WindowEnd)
	if !endTime.After(startTime) {
		return fieldErr(ErrCodeInvalidWindowOrder, "window_end",
			"window_end must be strictly after window_start")
	}

	// 7. trigger_source closed-enum check. The data-plane surface
	//    accepts exactly {scheduler, manual}; operator-rerun is
	//    rejected per ADR-0014 §"Consequences" item 3 (Admin API
	//    path's exclusive source per ADR-0002 §4).
	switch req.TriggerSource {
	case string(results.TriggerScheduler), string(results.TriggerManual):
		// ok
	case string(results.TriggerOperatorRerun):
		return fieldErr(ErrCodeInvalidTriggerSrc, "trigger_source",
			"trigger_source operator-rerun is the Admin API path's exclusive source (ADR-0002 §4)")
	default:
		return fieldErr(ErrCodeInvalidTriggerSrc, "trigger_source",
			fmt.Sprintf("trigger_source %q is not a recognized enum value", req.TriggerSource))
	}

	return nil
}

// mustParseValidatedWindow converts a wire-format window timestamp
// to time.Time. Callers MUST have passed the value through
// validate() first — the regex timestampPattern admits exactly
// the layout timestampLayout can deserialize, so a parse failure
// here means the regex and the layout have diverged. That is a
// programming invariant violation, not a request-validation
// failure; the function panics with a diagnostic message so the
// divergence surfaces immediately rather than letting a zero time
// leak into the formula.
func mustParseValidatedWindow(s string) time.Time {
	t, err := time.Parse(timestampLayout, s)
	if err != nil {
		panic(fmt.Sprintf(
			"api: validated window timestamp failed to parse: %q (%v) — timestampPattern and timestampLayout have diverged",
			s, err,
		))
	}
	return t
}

// fieldErr is a small constructor that keeps the rejection sites
// readable. The leading-comment-as-rationale is recorded in the
// caller's surrounding context (the validator step), not here.
func fieldErr(code, field, message string) *ErrorResponse {
	return &ErrorResponse{
		Code:    code,
		Field:   field,
		Message: message,
	}
}
