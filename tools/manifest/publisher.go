// path: tools/manifest/publisher.go

// Package main implements the dq-manifest publisher per
// ADR-0005 (manifest publication semantics).
//
// The Publisher type runs the four-step sequence in ADR-0005
// §4 verbatim:
//
//  1. Pre-publish verification (the three ADR-0001 contract
//     checks).
//  2. Write rule YAMLs to yamls/by-hash/sha256-<hex>.yaml.
//  3. Write manifest body to manifests/by-hash/sha256-<hex>.json.
//  4. CAS-write manifests/latest.json.
//
// Step ordering is load-bearing: writes happen only after
// verification, so a failed verification leaves no by-hash
// objects behind (the "no orphans on verification failure"
// invariant in ADR-0005 §4).
//
// The Publisher does not interpret the manifest at publish
// time — it stores it. The engine loader (engine/internal/
// loader) is the consumer; the JSON shape is the contract.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Publisher orchestrates one publication attempt. One Publisher
// per CLI invocation; fields are read-only after construction.
type Publisher struct {
	store                   Store
	rulesetVersion          string
	engineCompatibility     string
	linterUsed              string
	schemaMirrorDir         string
	supportedSchemaVersions []int
	now                     func() time.Time
	logger                  *slog.Logger
}

// Config gathers the fields required to construct a Publisher.
// All fields are required except Now and Logger, which default
// to time.Now and a discarding logger.
type Config struct {
	Store                   Store
	RulesetVersion          string
	EngineCompatibility     string // semver range string; the publisher stores it verbatim
	LinterUsed              string
	SchemaMirrorDir         string
	SupportedSchemaVersions []int
	Now                     func() time.Time
	Logger                  *slog.Logger
}

// Options is the per-Publish runtime input.
type Options struct {
	// RulesDir is the directory to walk for rule YAMLs. The
	// walker skips any `_schema` subdirectory at any depth and
	// any file whose name starts with an underscore (`_owners.
	// yaml`, future workspace-only files).
	RulesDir string

	// DryRun runs verification and computes the manifest hash
	// without writing any objects. Used by `dq-manifest publish
	// --dry-run` to validate a ruleset before publication.
	DryRun bool
}

// Result is the per-Publish success record. Populated even for
// DryRun (PointerGen=0 in that case).
type Result struct {
	ManifestHash   string // sha256 hex, no algorithm prefix
	RulesetVersion string
	RulesPublished int
	PointerGen     int64 // post-write generation; 0 in DryRun
}

// New constructs a Publisher from a Config. Returns an error
// for missing required fields.
func New(cfg Config) (*Publisher, error) {
	if cfg.Store == nil {
		return nil, errors.New("publisher: Store is required")
	}
	if cfg.RulesetVersion == "" {
		return nil, errors.New("publisher: RulesetVersion is required")
	}
	if strings.ContainsRune(cfg.RulesetVersion, '|') {
		// ADR-0002 input-safety: ruleset_version is one of the
		// five pipe-separated inputs to the execution_id hash.
		// The publisher rejects pipe in ruleset_version at the
		// surface so a misconfigured CI run cannot create a
		// manifest the engine will silently reject later.
		return nil, errors.New("publisher: RulesetVersion contains pipe character; forbidden by ADR-0002 input-safety")
	}
	if cfg.EngineCompatibility == "" {
		return nil, errors.New("publisher: EngineCompatibility is required")
	}
	if cfg.LinterUsed == "" {
		return nil, errors.New("publisher: LinterUsed is required")
	}
	if cfg.SchemaMirrorDir == "" {
		return nil, errors.New("publisher: SchemaMirrorDir is required")
	}
	// Stat the mirror dir once at construction so a typo in the
	// CLI flag fails fast with a single error instead of N
	// per-file errors at verification time.
	info, err := os.Stat(cfg.SchemaMirrorDir)
	if err != nil {
		return nil, fmt.Errorf("publisher: SchemaMirrorDir %q: %w", cfg.SchemaMirrorDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("publisher: SchemaMirrorDir %q is not a directory", cfg.SchemaMirrorDir)
	}
	if len(cfg.SupportedSchemaVersions) == 0 {
		return nil, errors.New("publisher: SupportedSchemaVersions must be non-empty")
	}
	clock := cfg.Now
	if clock == nil {
		clock = time.Now
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	supported := append([]int(nil), cfg.SupportedSchemaVersions...)
	sort.Ints(supported)
	return &Publisher{
		store:                   cfg.Store,
		rulesetVersion:          cfg.RulesetVersion,
		engineCompatibility:     cfg.EngineCompatibility,
		linterUsed:              cfg.LinterUsed,
		schemaMirrorDir:         cfg.SchemaMirrorDir,
		supportedSchemaVersions: supported,
		now:                     clock,
		logger:                  logger,
	}, nil
}

// Publish executes one publication attempt against the
// configured Store. Returns a Result on success; an error
// wrapping ErrVerificationFailed, ErrPreconditionFailed, or an
// operational error otherwise. The Publisher is single-use per
// invocation but stateless across invocations.
func (p *Publisher) Publish(ctx context.Context, opts Options) (*Result, error) {
	if opts.RulesDir == "" {
		return nil, errors.New("publish: Options.RulesDir is required")
	}

	// Step 1a — discover and parse rule YAMLs.
	rules, err := collectRules(opts.RulesDir)
	if err != nil {
		return nil, fmt.Errorf("collect rules from %s: %w", opts.RulesDir, err)
	}
	if len(rules) == 0 {
		// ADR-0005 does not name this case. The publisher rejects
		// it as a content-level error: an empty ruleset has no
		// downstream meaning and is almost certainly a misrouted
		// publish.
		return nil, fmt.Errorf("no rule YAMLs found under %s: %w", opts.RulesDir, ErrVerificationFailed)
	}

	// Step 1b — duplicate-entity check (ADR-0001 implicit:
	// one rule per entity per ruleset).
	if err := rejectDuplicateEntities(rules); err != nil {
		return nil, err
	}

	// Step 1c — derive the observed schema-version set.
	observed := observedSchemaVersions(rules)

	// Step 1d — verification 1 (supported schema versions).
	if err := verifySupportedSchemaVersions(observed, p.supportedSchemaVersions); err != nil {
		return nil, err
	}

	// Step 1e — verification 3 (schema mirror files present).
	// Verification 2 (declared set equals observed set) is
	// structurally satisfied: the publisher derives both, so
	// they cannot diverge.
	if err := verifySchemaMirrorPresent(observed, p.schemaMirrorDir); err != nil {
		return nil, err
	}

	// Step 1f — build the Manifest body. Rules sorted by entity
	// for deterministic JSON marshaling of the rules array.
	// GeneratedAt is current-wall-clock per ADR-0005 §5, so two
	// publish calls on identical content produce different
	// manifest hashes. Idempotency lives in ADR-0005 §2's
	// "by-hash objects are immutable" — a re-publish writes
	// nothing new under by-hash, only a fresh pointer.
	sort.Slice(rules, func(i, j int) bool { return rules[i].entity < rules[j].entity })
	manifestRules := make([]ManifestRule, 0, len(rules))
	for _, r := range rules {
		manifestRules = append(manifestRules, ManifestRule{
			Entity:   r.entity,
			YAMLPath: yamlByHashPath(r.hashHex),
			YAMLHash: r.hashHex,
		})
	}
	manifest := Manifest{
		ManifestVersion:       manifestSchemaVersion,
		RulesetVersion:        p.rulesetVersion,
		SchemaVersionsPresent: observed,
		EngineCompatibility:   p.engineCompatibility,
		LinterUsed:            p.linterUsed,
		GeneratedAt:           p.now().UTC(),
		Rules:                 manifestRules,
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	manifestHash := sha256Hex(manifestBytes)

	p.logger.Info("manifest computed",
		"ruleset_version", manifest.RulesetVersion,
		"manifest_hash", manifestHash,
		"rules", len(manifestRules),
		"schema_versions_present", observed,
		"dry_run", opts.DryRun,
	)

	if opts.DryRun {
		return &Result{
			ManifestHash:   manifestHash,
			RulesetVersion: p.rulesetVersion,
			RulesPublished: len(manifestRules),
			PointerGen:     0,
		}, nil
	}

	// Step 2 — write rule YAMLs to yamls/by-hash/.
	for _, r := range rules {
		key := yamlByHashPath(r.hashHex)
		if err := p.store.WriteIfNotExists(ctx, key, r.bytes); err != nil {
			if errors.Is(err, ErrAlreadyExists) {
				// Idempotent re-publish — by-hash objects are
				// immutable per ADR-0005 §2; same hash means same
				// content. Proceed.
				p.logger.Info("rule body already present (idempotent re-publish)",
					"entity", r.entity, "key", key, "yaml_hash", r.hashHex)
				continue
			}
			return nil, fmt.Errorf("write rule body %s: %w", key, err)
		}
	}

	// Step 3 — write manifest body to manifests/by-hash/.
	manifestKey := manifestByHashPath(manifestHash)
	if err := p.store.WriteIfNotExists(ctx, manifestKey, manifestBytes); err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			// Idempotent re-publish: by-hash objects are
			// immutable per ADR-0005 §2 (a matching sha256
			// digest is treated as identical content).
			p.logger.Info("manifest body already present (idempotent re-publish)",
				"key", manifestKey, "manifest_hash", manifestHash)
		} else {
			return nil, fmt.Errorf("write manifest body %s: %w", manifestKey, err)
		}
	}

	// Step 4 — CAS-write the pointer.
	expectedGen, err := p.store.ReadPointerGeneration(ctx, pointerPath)
	if err != nil {
		return nil, fmt.Errorf("read pointer generation: %w", err)
	}
	pointer := Pointer{
		PointerVersion: pointerSchemaVersion,
		ManifestHash:   "sha256:" + manifestHash,
		RulesetVersion: p.rulesetVersion,
		PublishedAt:    p.now().UTC(),
	}
	pointerBytes, err := json.Marshal(pointer)
	if err != nil {
		return nil, fmt.Errorf("marshal pointer: %w", err)
	}
	postGen, err := p.store.CASWritePointer(ctx, pointerPath, pointerBytes, expectedGen)
	if err != nil {
		// ErrPreconditionFailed is returned wrapped so the CLI
		// can pattern-match it via errors.Is for the dedicated
		// exit code.
		return nil, fmt.Errorf("CAS write pointer (expectedGen=%d): %w", expectedGen, err)
	}

	p.logger.Info("publish complete",
		"ruleset_version", p.rulesetVersion,
		"manifest_hash", manifestHash,
		"pointer_pre_gen", expectedGen,
		"pointer_post_gen", postGen,
	)
	return &Result{
		ManifestHash:   manifestHash,
		RulesetVersion: p.rulesetVersion,
		RulesPublished: len(manifestRules),
		PointerGen:     postGen,
	}, nil
}

// --- rule discovery / parsing ---

// rule is the in-memory result of walking one rule YAML.
type rule struct {
	path    string // path relative to RulesDir, used only for log/error context
	entity  string
	version int
	bytes   []byte // raw YAML bytes; written verbatim to yamls/by-hash/
	hashHex string // lowercase sha256 hex of bytes
}

// collectRules walks rulesDir and parses every *.yaml/*.yml
// file outside `_schema/` subdirectories and outside underscore-
// prefixed names (`_owners.yaml` etc.). Returns the parsed rules
// in walker order (the caller sorts deterministically before
// hashing the manifest).
func collectRules(rulesDir string) ([]rule, error) {
	var out []rule
	// Stat the rulesDir explicitly so a missing dir surfaces as
	// an operational error (CLI exit 2), distinguishable from a
	// present-but-empty dir (verification failure, exit 1).
	if _, err := os.Stat(rulesDir); err != nil {
		return nil, fmt.Errorf("stat rules dir %s: %w", rulesDir, err)
	}
	walkErr := filepath.WalkDir(rulesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if filepath.Base(path) == "_schema" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		if strings.HasPrefix(name, "_") {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		var doc struct {
			Entity  string `yaml:"entity"`
			Version int    `yaml:"version"`
		}
		if err := yaml.Unmarshal(body, &doc); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if doc.Entity == "" {
			return fmt.Errorf("%s: rule has empty `entity` field: %w", path, ErrVerificationFailed)
		}
		if doc.Version == 0 {
			return fmt.Errorf("%s: rule has missing or zero `version` field: %w", path, ErrVerificationFailed)
		}
		out = append(out, rule{
			path:    path,
			entity:  doc.Entity,
			version: doc.Version,
			bytes:   body,
			hashHex: sha256Hex(body),
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}

// rejectDuplicateEntities surfaces a verification failure when
// two rule YAMLs share the same `entity:` field. ADR-0001
// implicit invariant: one rule per entity per ruleset (the
// engine keys runtime evaluation on entity; two rules for the
// same entity would be ambiguous).
func rejectDuplicateEntities(rules []rule) error {
	seen := map[string]string{}
	var dupes []string
	for _, r := range rules {
		if prior, ok := seen[r.entity]; ok {
			dupes = append(dupes, fmt.Sprintf("%s (also in %s)", r.path, prior))
			continue
		}
		seen[r.entity] = r.path
	}
	if len(dupes) == 0 {
		return nil
	}
	sort.Strings(dupes)
	return fmt.Errorf("duplicate entity declarations: %v: %w", dupes, ErrVerificationFailed)
}

// observedSchemaVersions returns the sorted, deduplicated list
// of versions declared across the rule set. This is what the
// manifest's schema_versions_present field gets.
func observedSchemaVersions(rules []rule) []int {
	seen := map[int]struct{}{}
	for _, r := range rules {
		seen[r.version] = struct{}{}
	}
	out := make([]int, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Ints(out)
	return out
}

// --- path / hash helpers ---

// pointerPath is the single mutable control-plane object key
// per ADR-0005 §3.
const pointerPath = "manifests/latest.json"

func yamlByHashPath(hashHex string) string {
	// Algorithm prefix in the path itself per ADR-0005 §1.
	return "yamls/by-hash/sha256-" + hashHex + ".yaml"
}

func manifestByHashPath(hashHex string) string {
	return "manifests/by-hash/sha256-" + hashHex + ".json"
}

// sha256Hex returns the lowercase sha256 hex digest of b.
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
