// path: engine/internal/runner/execution_id.go

package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"dq-platform/engine/internal/results"
)

// pipeSeparator is the ASCII pipe character used to join the five
// inputs to the execution_id hash per ADR-0002 CC1. Per ADR-0002
// CC2 the inputs are NOT escaped; the formula's correctness
// depends on no input containing the pipe character.
const pipeSeparator = "|"

// timestampLayout is RFC 3339 UTC with second precision and
// trailing Z. ADR-0002 CC2 commits this exact shape for the
// window_start and window_end inputs. Fractional seconds are
// excluded; sub-second precision would be a breaking change to
// the formula (ADR-0002 CC13).
const timestampLayout = "2006-01-02T15:04:05Z"

// ErrPipeCharacterForbidden wraps every rejection that flags an
// input containing the ASCII pipe character. ADR-0002 CC2 input
// safety: the pipe character is forbidden in any of the five
// execution_id inputs.
var ErrPipeCharacterForbidden = errors.New("input contains forbidden ASCII pipe character (ADR-0002 CC2)")

// Compute returns the execution_id for a trigger per ADR-0002
// CC1:
//
//	sha256_hex(
//	  ruleset_version | entity | window_start | window_end | trigger_source
//	)
//
// where | is the ASCII pipe character and inputs are not escaped.
// The output is the 64-character lowercase hexadecimal encoding
// of the 32-byte sha256 digest per ADR-0002 CC7.
//
// Compute validates the pipe-safety invariant for ruleset_version,
// entity, and trigger_source before hashing. window_start /
// window_end are formatted by Compute itself (Go's time.Time API
// does not produce pipe characters) but must be representable in
// the canonical RFC 3339 UTC second-precision form — Compute
// truncates them to second precision and forces UTC; sub-second
// precision in the input is silently dropped (callers requiring
// strict rejection should validate at the API layer).
func Compute(rulesetVersion, entity string, windowStart, windowEnd time.Time, triggerSource results.TriggerSource) (string, error) {
	if strings.ContainsRune(rulesetVersion, '|') {
		return "", fmt.Errorf("ruleset_version: %w", ErrPipeCharacterForbidden)
	}
	if strings.ContainsRune(entity, '|') {
		return "", fmt.Errorf("entity: %w", ErrPipeCharacterForbidden)
	}
	if strings.ContainsRune(string(triggerSource), '|') {
		return "", fmt.Errorf("trigger_source: %w", ErrPipeCharacterForbidden)
	}

	// Force UTC + truncate to second precision per ADR-0002 CC2.
	wsCanonical := windowStart.UTC().Truncate(time.Second).Format(timestampLayout)
	weCanonical := windowEnd.UTC().Truncate(time.Second).Format(timestampLayout)

	joined := strings.Join([]string{
		rulesetVersion,
		entity,
		wsCanonical,
		weCanonical,
		string(triggerSource),
	}, pipeSeparator)

	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:]), nil
}
