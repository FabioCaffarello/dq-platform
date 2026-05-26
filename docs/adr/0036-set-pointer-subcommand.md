<!-- path: docs/adr/0036-set-pointer-subcommand.md -->

# ADR-0036 — `dq-manifest set-pointer` Rollback Subcommand

- **Status:** accepted
- **Date:** 2026-05-25

---

## Context

[ADR-0005](./0005-manifest-publication-semantics.md) §3
commits the pointer file (`manifests/latest.json`) as the
**single mutable control-plane object**; ADR-0005 §4 commits
that every publish and every rollback is exactly one
generation-conditional write to that pointer. The publisher
CLI (`dq-manifest publish`) implements §4 verbatim: it walks
the rules directory, hashes each YAML body and the manifest
body, content-addresses both at
`yamls/by-hash/sha256-<hex>.yaml` and
`manifests/by-hash/sha256-<hex>.json`, then CAS-writes the
pointer to reference the new manifest.

What the v1 CLI did **not** expose was the **rollback
primitive** — a CAS pointer write to an existing manifest
hash, without re-publishing rule bodies or recomputing
manifest content. Until this ADR, the operator runbook at
[`docs/runbooks/manifest-rollback.md`](../runbooks/manifest-rollback.md)
§3 offered two workarounds: re-publish the prior ruleset from
git (only works when the prior linter binary digest is
recoverable per ADR-0001 §C9), or hand-roll a pointer JSON
file and write it via `gcloud storage cp
--if-generation-match` (bypasses the CLI contract entirely;
operators construct the pointer body manually).

Both workarounds degraded rollback ergonomics exactly where
ergonomics matter most: incident response. Operators reaching
for rollback are under stress; a multi-step manual procedure
with hand-written JSON or a git-state reconstruction is the
wrong tool for that moment. The `gcloud storage cp` workaround
also created a **parallel authority** — a path to mutate the
control plane that did not flow through the CLI's exit-code
contract, observability emission, or safety checks.

The principles bearing on the decision are **P4** (cost is a
first-class constraint — rollback ergonomics during incident
response is operator cost; closing the bypass-via-gsutil gap
makes that cost smaller), **P3** (ownership must be explicit
— `dq-manifest` is the contract surface for manifest-state
changes; parallel authorities are harder to audit and reason
about), and **R3** (do not revisit settled architecture —
ADR-0005's pointer-as-single-mutable-control-plane is
preserved; the new subcommand uses the same primitive as a
new caller).

---

## Decision

The `dq-manifest` CLI exposes a first-class
`set-pointer` subcommand that runs the CAS pointer write
under the same exit-code contract as `publish`. The
subcommand verifies the target manifest body exists before
issuing the write, eliminating the dangling-pointer hazard
that the `gcloud storage cp` workaround silently permitted.

### Subcommand surface

```
dq-manifest set-pointer \
  -bucket <bucket-name>          \  required
  -manifest-hash <sha256-hex>    \  required; 64-char lowercase hex
  -dry-run                       \  optional; verify + log without CAS write
  -storage-emulator-host <host>     optional; for local-emulator runs
```

Flag conventions:

- `-manifest-hash` accepts the **hex digest without the
  `sha256:` prefix**. The prefix is part of the pointer-file
  representation, not the CLI input; the subcommand adds it
  when constructing the pointer body. Operators reading the
  prior pointer with `gcloud storage cat` see `sha256:<hex>`;
  they pass `<hex>` to `set-pointer`.
- `-dry-run` runs every verification and emits the planned
  pointer JSON to the structured log without issuing the
  CAS write. Useful for pre-prod validation and for
  inspecting what the rollback will do before committing.
- `-storage-emulator-host` mirrors `publish`'s flag of the
  same name (local Compose / sandbox emulator support per
  ADR-0010 §3.2).

### Execute sequence

The subcommand runs seven steps. Each step has a defined
failure mode that maps to an exit code:

1. **Validate hash shape.** `^[0-9a-f]{64}$`. A leading
   `sha256:` prefix, non-hex characters, or wrong length
   fails with exit 1.
2. **Read the target body** at
   `manifests/by-hash/sha256-<hex>.json`. A missing body
   fails with exit 1 and an error message that names the
   dangling-pointer safety property — rolling back to a
   non-existent hash would produce a pointer the engine
   fails closed on.
3. **Unmarshal the target body** and extract its
   `ruleset_version`. A malformed body or an empty
   `ruleset_version` fails with exit 1.
4. **Read the prior pointer** for forensic logging and the
   prior pointer generation for the CAS precondition. A
   missing pointer (no prior publish) is tolerated; the CAS
   write below uses `expectedGen=0`.
5. **If `-dry-run`,** emit a structured log line with the
   target hash, the target's ruleset version, the prior
   pointer's hash and ruleset version, the prior generation,
   and the planned pointer JSON. Return exit 0 with
   `PostPointerGen=0`. No mutation occurs.
6. **Build the new pointer JSON** with the target hash
   (prefixed with `sha256:`), the target's `ruleset_version`,
   and `published_at: now().UTC()`. The byte shape matches
   ADR-0005 §6 exactly.
7. **CAS-write the pointer** with `expectedGen` set to the
   prior generation. A successful write returns the new
   generation; a precondition failure (a concurrent publisher
   moved the pointer between step 4 and step 7) fails with
   exit 3, which the operator retries.

### Exit-code contract

`set-pointer` inherits `publish`'s exit-code mapping so
operator wrapper scripts that key on publish codes work for
rollback unchanged:

| Exit | Meaning |
|---|---|
| 0 | rollback OK (or dry-run success) |
| 1 | verification failed (target body missing, invalid hash shape, malformed target manifest, empty ruleset_version) |
| 2 | operational failure (bucket missing, network, etc.) |
| 3 | CAS precondition failed (pointer moved between read and write; retry) |
| 64 | usage error (missing required flag) |

### Runbook posture

[`docs/runbooks/manifest-rollback.md`](../runbooks/manifest-rollback.md)
§3 is rewritten so that `dq-manifest set-pointer` is the
primary path (three steps: recover the target hash, dry-run,
apply). The two prior workarounds become explicit fallbacks:

- **§3.fallback.A** — re-publish the prior ruleset from git
  is reserved for the forensic case where the operator wants
  to confirm the prior manifest reproduces from git
  (audit-grade reconstruction).
- **§3.fallback.B** — `gcloud storage cp
  --if-generation-match` is reserved for the edge case where
  `dq-manifest` itself is broken or unavailable.

### Internal design

A new file `tools/manifest/rollback.go` carries the
orchestration type:

```
type Rollback struct {
    store  Store
    now    func() time.Time
    logger *slog.Logger
}

type RollbackOptions struct {
    TargetHashHex string  // 64-char lowercase hex; no `sha256:` prefix
    DryRun        bool
}

type RollbackResult struct {
    TargetHash       string
    TargetRulesetVer string
    PriorHash        string
    PriorRulesetVer  string
    PriorPointerGen  int64
    PostPointerGen   int64
}
```

The `Store` interface (`WriteIfNotExists`, `ReadObject`,
`ReadPointerGeneration`, `CASWritePointer`) is the same one
the publisher consumes; the rollback adds no new substrate
surface. The implementation reuses `ErrObjectNotFound`,
`ErrVerificationFailed`, and `ErrPreconditionFailed` as
defined in the publisher package.

### Why this does not reopen ADR-0005

ADR-0005 §3 commits the pointer as the single mutable
control-plane object; §4 commits the CAS write as the only
mutation primitive. This ADR uses the same primitive — the
subcommand is a new caller, not a new mechanism. The publish
sequence (verify → write rule bodies → write manifest body →
CAS pointer) and the rollback sequence (verify target body
exists → CAS pointer) share step 4. ADR-0005 stays accepted
without amendment.

### Why this does not reopen ADR-0001 §C9

The CLI's `set-pointer` operation does not run any linter; it
copies the target manifest's existing content into the new
pointer. ADR-0001 §C9's unforgeable linter-pin contract
applies to NEW manifests being published — bodies the
publisher constructs and signs with `linter_used` —, not to
rollback. The rolled-back pointer carries the same
`ruleset_version` the target body already declares; the
target body's `linter_used` field is unchanged because the
body itself is unchanged.

---

## Consequences

1. **A new `set-pointer` subcommand ships in
   `tools/manifest/main.go`.** Reuses the existing Store
   interface; no new substrate surface; no new dependency.

2. **A new file `tools/manifest/rollback.go` carries the
   `Rollback` orchestration type.** Mirrors `Publisher` but
   for the rollback case.

3. **Unit tests in `tools/manifest/rollback_test.go` cover
   the safety properties:** happy path, target-body-missing
   (verification fail with dangling-pointer message), invalid
   hash shapes (empty, short, prefixed, uppercase, non-hex),
   malformed target body, empty `ruleset_version`, dry-run
   does-not-mutate, CAS race-loser (precondition fail).

4. **Integration tests in
   `tools/manifest/rollback_integration_test.go` (build tag
   `integration` per ADR-0034) cover end-to-end against
   fake-gcs-server:** publish A → publish B → set-pointer
   back to A → verify the pointer reads A's hash and carries
   A's `ruleset_version`; plus a dry-run-does-not-mutate
   integration case.

5. **The manifest-rollback runbook §3 is rewritten** with
   the new primary path; the two prior workarounds are
   demoted to explicit fallback sub-sections.

6. **The CLI exit-code contract extends to two subcommands**
   with the same mapping. The contract surface stays
   identical for operator wrapper scripts.

7. **The bypass-via-gsutil failure mode is closed for the
   common case.** `gcloud storage cp --if-generation-match`
   remains documented as the §3.fallback.B edge case (CLI
   unavailable), not the primary procedure.

8. **ADR-0005 and ADR-0001 are preserved.** The subcommand
   reuses the existing pointer primitive without amending
   either ADR.

9. **B2-10 closes.** The decision-log B2-10 row moves to
   `resolved-adr` (→ this ADR). The runbook §2 precondition
   no longer references a "TBD" CLI surface.

10. **One deferred follow-up is registered: `dq-manifest
    list-pointers` for prior-pointer recovery ergonomics.**
    Today operators discover prior manifest hashes via
    `gcloud storage ls gs://<bucket>/manifests/by-hash/`.
    A future CLI ergonomic could ship `dq-manifest
    list-pointers` that lists historical pointer generations
    and their hashes from a designated audit log. Reserved
    for B-item registration when concrete operational signal
    shows the gcloud-based discovery is too cumbersome.
