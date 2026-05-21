<!-- path: docs/adr/0008-git-host.md -->

# ADR-0008 — Git Host

- **Status:** accepted
- **Date:** 2026-05-21

---

## Context

The platform is a single monorepo with five product workspaces
(`engine/`, `rules/`, `tools/`, `deploy/`, `docs/`) plus
auxiliary configuration directories. The chosen Git host
governs every downstream CI surface: branch-protection rules
that gate the byte-equality schema mirror check (ADR-0001),
CODEOWNERS syntax that protects the schema source, the
default CI runner, the container/artifact registry, and the
unforgeability mechanism for the linter version pin.

This ADR records the host choice and its direct consequences
for downstream scaffolding. Sub-decisions about runner,
registry, and branch-protection specifics are explicitly
deferred.

---

## Decision

The Git host is **GitHub**.

This is a project-owner directive (recorded as committed
input, not the outcome of an option-space analysis). The
foundation-document placeholder reading `.gitlab/ or .github/`
collapses to `.github/` from Wave 3 onward.

**GitHub Actions is the default CI runner.** Alternative
runners (self-hosted, external) are a future sub-decision and
are not committed here.

**The unforgeable linter-pin mechanism required by ADR-0001
must be implemented on the chosen host's primitives** —
digest pinning of a registry image, a content-addressed
artifact store reference, or another equivalent. The specific
mechanism is a scaffolding design item; this ADR commits the
constraint, not the implementation.

---

## Consequences

1. **`.github/` is the canonical CI configuration directory.**
   The foundation-document placeholder reading
   `.gitlab/ or .github/` no longer carries `.gitlab/` as a
   live option; any reference to it is purely historical.

2. **CODEOWNERS uses GitHub-native syntax.** The CODEOWNERS
   protection required by ADR-0001 over
   `engine/internal/dsl/schema/` and `rules/_schema/` follows
   the GitHub `CODEOWNERS` file format.

3. **GitHub Actions is the CI runner default.** The
   byte-equality schema mirror check from ADR-0001, the
   linter, build, and test pipelines, and any future
   pre-publish manifest verifications (ADR-0005) run as
   GitHub Actions workflows unless and until an alternative
   runner is chosen.

4. **The linter version pin lives in CI configuration of the
   chosen host.** Whether the unforgeability requirement
   (ADR-0001) is satisfied by image digest pinning of a
   GitHub Container Registry image, by another
   content-addressed artifact mechanism, or by an
   equivalent primitive is a scaffolding decision; this ADR
   commits only the host and the constraint that the
   mechanism be available on it.

5. **Container/artifact registry choice is deferred.**
   GitHub Container Registry is one option; alternative
   registries reachable from GitHub Actions are equally
   permissible. The choice does not affect any commitment
   recorded here.

6. **Branch protection that gates the byte-equality CI
   check (ADR-0001) uses GitHub branch-protection
   primitives.** The specific required-check configuration
   (which workflows must pass, whether reviews are required,
   whether the check is non-bypassable for administrators)
   is a scaffolding decision.

---

## Notes

- A future alternative CI runner (self-hosted, external) is
  a follow-up sub-decision. The default holds until that
  sub-decision is opened.
- The container/artifact registry choice is a follow-up
  sub-decision, independent of all other commitments in this
  ADR.
- Branch-protection and required-check specifics land with
  the Wave 3 CI scaffolding.
