// path: engine/internal/loader/compat.go

package loader

import (
	"errors"
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// ErrContractMismatch wraps every ADR-0001 contract-check failure
// (manifest_version unsupported, engine_compatibility unsatisfied,
// schema_versions_present not engine-supported). The wrapped inner
// error carries the specific check that failed.
var ErrContractMismatch = errors.New("manifest contract check failed")

// runContractChecks executes the three ADR-0001 load-time checks
// against the parsed Manifest. Returns nil on success; an error
// wrapping ErrContractMismatch on any failure.
//
// The three checks per ADR-0001:
//
//  1. manifest_version is one this loader supports (currently {1}).
//     Future v2 lands additively; this loader rejects unknown
//     versions to fail closed (PAT-1 "no partial loading").
//  2. The engine's running version satisfies the manifest's
//     engine_compatibility semver range.
//  3. Every value in schema_versions_present is in the engine's
//     supported set.
//
// linter_used is intentionally not checked — ADR-0001 / ADR-0005
// commit it as audit-only metadata.
func runContractChecks(m *Manifest, engineVersion *semver.Version, supportedManifestVersions []int, supportedSchemaVersions []int) error {
	if !containsInt(supportedManifestVersions, m.ManifestVersion) {
		return fmt.Errorf("%w: manifest_version %d is not in engine-supported set %v",
			ErrContractMismatch, m.ManifestVersion, supportedManifestVersions)
	}

	constraint, err := semver.NewConstraint(m.EngineCompatibility)
	if err != nil {
		return fmt.Errorf("%w: engine_compatibility %q is not a valid semver constraint: %v",
			ErrContractMismatch, m.EngineCompatibility, err)
	}
	if !constraint.Check(engineVersion) {
		return fmt.Errorf("%w: engine version %s does not satisfy manifest's engine_compatibility %q",
			ErrContractMismatch, engineVersion, m.EngineCompatibility)
	}

	for _, v := range m.SchemaVersionsPresent {
		if !containsInt(supportedSchemaVersions, v) {
			return fmt.Errorf("%w: schema_versions_present includes %d, which is not in engine-supported set %v",
				ErrContractMismatch, v, supportedSchemaVersions)
		}
	}
	return nil
}

func containsInt(xs []int, x int) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
