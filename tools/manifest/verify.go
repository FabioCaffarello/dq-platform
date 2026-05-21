// path: tools/manifest/verify.go

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// verifySupportedSchemaVersions implements the first pre-publish
// verification per ADR-0001: every rule's declared version must
// be a value the engine accepts (the supportedSchemaVersions
// set the publisher was configured with at startup).
//
// Returns an error wrapping ErrVerificationFailed when any rule
// names an unsupported version. The error message names every
// offending rule so the operator can fix all of them in one
// edit pass.
func verifySupportedSchemaVersions(observed []int, supported []int) error {
	allowed := map[int]struct{}{}
	for _, v := range supported {
		allowed[v] = struct{}{}
	}
	var unsupported []int
	for _, v := range observed {
		if _, ok := allowed[v]; !ok {
			unsupported = append(unsupported, v)
		}
	}
	if len(unsupported) == 0 {
		return nil
	}
	sort.Ints(unsupported)
	return fmt.Errorf("rule declares unsupported schema version(s) %v (engine supports %v): %w",
		unsupported, supported, ErrVerificationFailed)
}

// verifySchemaMirrorPresent implements the third pre-publish
// verification per ADR-0001: for every value N in the observed
// schema-version set, the mirror file
// `<schemaMirrorDir>/v<N>.schema.json` must exist.
//
// The publisher does not parse the mirror or validate rules
// against it — that is the linter's job (tools/lint). This
// verification only confirms the mirror file is present, so
// the engine + linter agree on the contract surface that
// accompanies the manifest.
func verifySchemaMirrorPresent(observed []int, schemaMirrorDir string) error {
	var missing []string
	for _, v := range observed {
		path := filepath.Join(schemaMirrorDir, fmt.Sprintf("v%d.schema.json", v))
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missing = append(missing, path)
				continue
			}
			return fmt.Errorf("stat schema mirror %s: %w", path, err)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("schema mirror file(s) missing: %v: %w", missing, ErrVerificationFailed)
}
