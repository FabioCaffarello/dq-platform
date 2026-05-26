#!/usr/bin/env bash
# path: scripts/check-tag-scope.sh
#
# Validates that a workspace tag (e.g., engine-v1.2.0) diffs
# cleanly against its prior matching tag — every changed file
# must be under the workspace's path per ADR-0042 §"Clause 3 —
# Versioning invariants".
#
# This script is the shared implementation for two surfaces:
#
#   - `.github/workflows/tag-prefix-gate.yml` runs it on every
#     tag-push matching one of the committed prefixes.
#   - `make check-tag-scope TAG=<tag>` runs it locally so
#     operators can dry-run the gate before pushing.
#
# Usage:
#   scripts/check-tag-scope.sh <tag>
#
# Exits 0 when the tag's diff against its prior matching tag is
# fully under the workspace scope (or no prior tag exists).
# Exits 1 on any out-of-scope path, an unrecognized prefix, or
# an unresolvable tag reference.

set -euo pipefail

if [ "$#" -ne 1 ] || [ -z "$1" ]; then
  echo "usage: $0 <tag>" >&2
  echo "  example: $0 engine-v1.2.0" >&2
  exit 1
fi
tag="$1"

# Map prefix → workspace path. The five committed prefixes from
# ADR-0042 Clause 3 (extended by B2-3 to include tools-manifest-v).
case "$tag" in
  engine-v*)         prefix="engine-v";         scope="engine/" ;;
  rules-v*)          prefix="rules-v";          scope="rules/" ;;
  tools-lint-v*)     prefix="tools-lint-v";     scope="tools/lint/" ;;
  tools-manifest-v*) prefix="tools-manifest-v"; scope="tools/manifest/" ;;
  deploy-v*)         prefix="deploy-v";         scope="deploy/" ;;
  *)
    echo "error: tag $tag does not match any committed workspace prefix" >&2
    echo "       (engine-v|rules-v|tools-lint-v|tools-manifest-v|deploy-v)" >&2
    exit 1
    ;;
esac
echo "Prefix: ${prefix}* → workspace scope: ${scope}"

# Confirm the tag actually exists in the local repo.
if ! git rev-parse --quiet --verify "refs/tags/${tag}" >/dev/null; then
  echo "error: tag $tag does not exist locally (try \`git fetch --tags\`)" >&2
  exit 1
fi

# Find the immediately-prior matching tag, excluding the current.
# `-v:refname` is semver-style sort, which is correct for backport
# scenarios where a bugfix tag on an old line is created after a
# newer major (e.g., engine-v1.5.1 created after engine-v2.0.0 —
# the prior of v1.5.1 should be v1.5.0, not v2.0.0).
prior="$(git tag --list "${prefix}*" --sort=-v:refname | grep -vFx "$tag" | head -1 || true)"
if [ -z "$prior" ]; then
  echo "First tag with prefix ${prefix}* — no prior to diff against; gate passes by inception."
  exit 0
fi
echo "Prior matching tag: $prior"
echo "Comparing tree state $prior → $tag"

# Diff the two tree states. Any path that is not a descendant of
# the scope directory is a violation.
violations="$(git diff --name-only "$prior" "$tag" | grep -v "^${scope}" || true)"
if [ -n "$violations" ]; then
  echo "error: Tag $tag diffs against $prior include files outside ${scope}:" >&2
  printf '  %s\n' $violations >&2
  echo "error: Per ADR-0042 Clause 3, a workspace tag must point at a commit whose diff against the prior matching tag is fully under ${scope}." >&2
  echo "       Move cross-workspace changes to a separate commit and re-tag a later commit that touches only ${scope}." >&2
  exit 1
fi
echo "Tag $tag scope validated against ${scope}"
