// path: tools/manifest/rollback.go

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"time"
)

// Rollback orchestrates one CAS-conditional pointer-write that
// re-points `manifests/latest.json` at a prior manifest hash.
// Implements the rollback half of ADR-0005's pointer-as-single-
// mutable-control-plane invariant: every rollback is exactly one
// generation-conditional write to the pointer (ADR-0005 §3 + §4
// step 4).
//
// The Rollback does NOT write any new manifest body — it copies
// the target body's `ruleset_version` field into the new pointer
// JSON and CAS-writes. The body itself is immutable per ADR-0005
// §2; the rollback simply repoints at it.
//
// Use this for incident-response rollback per
// docs/runbooks/manifest-rollback.md. The `set-pointer`
// subcommand in main.go is the CLI surface.
type Rollback struct {
	store  Store
	now    func() time.Time
	logger *slog.Logger
}

// RollbackConfig gathers the fields required to construct a
// Rollback. Store is required; Now defaults to time.Now;
// Logger defaults to a discarding logger.
type RollbackConfig struct {
	Store  Store
	Now    func() time.Time
	Logger *slog.Logger
}

// RollbackOptions is the per-Execute input.
type RollbackOptions struct {
	// TargetHashHex is the 64-char lowercase hex sha256 digest
	// of the target manifest body. NOT prefixed with `sha256:`
	// — that prefix is part of the pointer-file representation
	// and is added when constructing the pointer body.
	TargetHashHex string

	// DryRun runs all verifications + emits the planned pointer
	// JSON without issuing the CAS write. Useful for "what
	// would this do?" validation before committing.
	DryRun bool
}

// RollbackResult is the per-Execute success record.
type RollbackResult struct {
	TargetHash        string // the input hash (no `sha256:` prefix)
	TargetRulesetVer  string // copied from target body
	PriorHash         string // the prior pointer's hash (forensic); empty if no prior pointer
	PriorRulesetVer   string // the prior pointer's ruleset_version (forensic); empty if no prior pointer
	PriorPointerGen   int64  // pre-CAS generation; 0 if pointer did not exist
	PostPointerGen    int64  // post-CAS generation; 0 in DryRun
}

// hashHexPattern matches a 64-char lowercase hex string.
var hashHexPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// NewRollback constructs a Rollback from a RollbackConfig.
// Returns an error when Store is missing.
func NewRollback(cfg RollbackConfig) (*Rollback, error) {
	if cfg.Store == nil {
		return nil, errors.New("rollback: Store is required")
	}
	clock := cfg.Now
	if clock == nil {
		clock = time.Now
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Rollback{
		store:  cfg.Store,
		now:    clock,
		logger: logger,
	}, nil
}

// Execute runs one rollback attempt against the configured
// Store. Returns a RollbackResult on success; an error wrapping
// ErrVerificationFailed, ErrPreconditionFailed, or an
// operational error otherwise.
//
// Sequence:
//
//  1. Validate TargetHashHex shape (64-char lowercase hex).
//  2. Read target manifest body at
//     manifests/by-hash/sha256-<hex>.json. Missing →
//     ErrVerificationFailed (avoid dangling pointer per
//     DD-4 in the B2-10 study).
//  3. Unmarshal target body; extract `ruleset_version` (the
//     pointer carries it duplicated per ADR-0005 §6).
//  4. Read prior pointer (for forensic prior-hash logging) +
//     prior pointer generation.
//  5. If DryRun, emit planned pointer JSON to the logger and
//     return PostPointerGen=0.
//  6. Build pointer JSON with target hash + target's
//     ruleset_version + now() as published_at.
//  7. CAS-write the pointer with expectedGen = priorGen.
//     Race-loser → ErrPreconditionFailed.
func (r *Rollback) Execute(ctx context.Context, opts RollbackOptions) (*RollbackResult, error) {
	// Step 1 — validate hash shape.
	if !hashHexPattern.MatchString(opts.TargetHashHex) {
		return nil, fmt.Errorf("rollback: target hash %q is not a 64-char lowercase hex sha256: %w",
			opts.TargetHashHex, ErrVerificationFailed)
	}

	// Step 2 — read target body to verify it exists.
	targetKey := manifestByHashPath(opts.TargetHashHex)
	targetBody, err := r.store.ReadObject(ctx, targetKey)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			return nil, fmt.Errorf("rollback: target manifest body %s does not exist (rolling back to a non-existent hash would produce a dangling pointer the engine fails closed on): %w",
				targetKey, ErrVerificationFailed)
		}
		return nil, fmt.Errorf("rollback: read target body %s: %w", targetKey, err)
	}

	// Step 3 — unmarshal target body, extract ruleset_version.
	var targetManifest Manifest
	if err := json.Unmarshal(targetBody, &targetManifest); err != nil {
		return nil, fmt.Errorf("rollback: parse target body %s: %w: %w", targetKey, err, ErrVerificationFailed)
	}
	if targetManifest.RulesetVersion == "" {
		return nil, fmt.Errorf("rollback: target body %s has empty ruleset_version: %w", targetKey, ErrVerificationFailed)
	}

	// Step 4 — read prior pointer (for forensic logging) + generation.
	priorPointer, priorGen, err := r.readPriorPointer(ctx)
	if err != nil {
		return nil, fmt.Errorf("rollback: read prior pointer: %w", err)
	}

	result := &RollbackResult{
		TargetHash:       opts.TargetHashHex,
		TargetRulesetVer: targetManifest.RulesetVersion,
		PriorHash:        priorPointer.ManifestHash,
		PriorRulesetVer:  priorPointer.RulesetVersion,
		PriorPointerGen:  priorGen,
		PostPointerGen:   0,
	}

	// Step 5 — DryRun: emit planned pointer JSON and return.
	newPointer := Pointer{
		PointerVersion: pointerSchemaVersion,
		ManifestHash:   "sha256:" + opts.TargetHashHex,
		RulesetVersion: targetManifest.RulesetVersion,
		PublishedAt:    r.now().UTC(),
	}
	plannedBytes, err := json.Marshal(newPointer)
	if err != nil {
		return nil, fmt.Errorf("rollback: marshal pointer: %w", err)
	}
	if opts.DryRun {
		r.logger.Info("rollback dry-run",
			"target_hash", opts.TargetHashHex,
			"target_ruleset_version", targetManifest.RulesetVersion,
			"prior_hash", priorPointer.ManifestHash,
			"prior_ruleset_version", priorPointer.RulesetVersion,
			"prior_pointer_gen", priorGen,
			"planned_pointer_bytes", string(plannedBytes),
		)
		return result, nil
	}

	// Step 6 + 7 — CAS-write the pointer.
	postGen, err := r.store.CASWritePointer(ctx, pointerPath, plannedBytes, priorGen)
	if err != nil {
		return nil, fmt.Errorf("rollback: CAS write pointer (expectedGen=%d): %w", priorGen, err)
	}

	result.PostPointerGen = postGen
	r.logger.Info("rollback complete",
		"target_hash", opts.TargetHashHex,
		"target_ruleset_version", targetManifest.RulesetVersion,
		"prior_hash", priorPointer.ManifestHash,
		"prior_ruleset_version", priorPointer.RulesetVersion,
		"prior_pointer_gen", priorGen,
		"post_pointer_gen", postGen,
	)
	return result, nil
}

// readPriorPointer reads the current pointer body + its
// generation, returning (zero-value, 0, nil) when no pointer
// exists yet (edge case — first publish would have created it,
// so a missing pointer at rollback time is unusual but not
// fatal — the CAS write below with expectedGen=0 + DoesNotExist
// precondition will fail loudly if the pointer did exist
// between the read here and the write below, which is the
// race-loser branch).
func (r *Rollback) readPriorPointer(ctx context.Context) (Pointer, int64, error) {
	priorGen, err := r.store.ReadPointerGeneration(ctx, pointerPath)
	if err != nil {
		return Pointer{}, 0, fmt.Errorf("read pointer generation: %w", err)
	}
	if priorGen == 0 {
		// Pointer does not exist yet — unusual but not fatal.
		return Pointer{}, 0, nil
	}
	body, err := r.store.ReadObject(ctx, pointerPath)
	if err != nil {
		// Generation read succeeded but body read failed —
		// transient operational issue; surface it.
		return Pointer{}, priorGen, fmt.Errorf("read pointer body: %w", err)
	}
	var prior Pointer
	if err := json.Unmarshal(body, &prior); err != nil {
		// Malformed current pointer — surface as operational
		// rather than verification failure (the rollback is
		// not responsible for the prior pointer's shape).
		return Pointer{}, priorGen, fmt.Errorf("parse prior pointer: %w", err)
	}
	return prior, priorGen, nil
}
