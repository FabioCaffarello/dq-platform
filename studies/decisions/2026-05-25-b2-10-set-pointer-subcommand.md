<!-- path: studies/decisions/2026-05-25-b2-10-set-pointer-subcommand.md -->

# B2-10 — `dq-manifest set-pointer` Rollback Subcommand

## Context

[ADR-0005](../../docs/adr/0005-manifest-publication-semantics.md)
§3 commits the pointer file (`manifests/latest.json`) as the
**single mutable control-plane object**; ADR-0005 §4 commits
that every publish and every rollback is exactly one
generation-conditional write to that pointer. The publisher
CLI implements §4 verbatim: `dq-manifest publish` runs the
four-step sequence (verify → write rule bodies → write
manifest body → CAS-write pointer).

What `dq-manifest` does NOT expose at v1: the **rollback
primitive** — a CAS-conditional pointer write to an existing
manifest hash, without re-publishing anything else.
[`docs/runbooks/manifest-rollback.md`](../../docs/runbooks/manifest-rollback.md)
§3 acknowledges this gap and offers two workarounds:

- **§3.A** re-publish the prior ruleset from git — only
  works if the prior linter binary digest is recoverable
  (per ADR-0001 §C9 unforgeable pin); otherwise the new
  manifest hash differs from the original.
- **§3.B** write the pointer directly via `gcloud storage cp
  --if-generation-match` — bypasses the CLI contract
  entirely; operators hand-roll the pointer JSON.

Both workarounds work but degrade rollback ergonomics
exactly when it matters most (incident response). B2-10
asks: should `dq-manifest` expose a first-class
`set-pointer <hash>` subcommand that runs the CAS pointer
write under the same exit-code contract as `publish`?

The principles bearing on the decision are **P4** (cost is
a first-class constraint — operational simplicity counts;
rollback ergonomics during incident response is operator
cost), **P3** (ownership is explicit — `dq-manifest` is the
contract surface for manifest-state changes; bypassing it
via raw `gsutil` creates a parallel authority that's harder
to audit), and **R3** (do not revisit settled architecture —
ADR-0005's pointer-as-mutable-control-plane is preserved;
the subcommand uses the same primitive).

What B2-10 must commit:

1. **The subcommand shape** — what flags, what behavior,
   what failure modes.
2. **The safety invariants** — what does the CLI check
   before issuing the pointer write?
3. **The runbook update** — `docs/runbooks/manifest-rollback.md`
   §3 replaces the workarounds with the new primitive.
4. **The exit-code contract extension** — `set-pointer`
   inherits `publish`'s exit codes for symmetry.

---

## Decision Drivers

- **DD-1 — Bypass-via-gsutil is the failure mode B2-10
  addresses.** Today's runbook §3.B has operators
  hand-rolling pointer JSON and invoking `gcloud storage
  cp`. The CLI is the contract surface (`dq-manifest
  publish` enforces field shapes, exit codes, observability
  emission); raw gsutil bypasses all of that. Closing the
  bypass closes the parallel-authority risk.
- **DD-2 — Rollback ergonomics during incident response
  matter most.** Operators reaching for rollback are
  under stress; a single subcommand with a clear flag set
  beats a multi-step manual procedure with hand-written
  JSON every time.
- **DD-3 — The primitive is the same as publish step 4.**
  ADR-0005 §4 step 4 is the CAS-conditional pointer write
  — both publish and rollback share it. The subcommand
  reuses the existing `Store.CASWritePointer` +
  `Store.ReadPointerGeneration` interfaces without
  introducing new substrate surface.
- **DD-4 — Safety: verify target before pointing.** A
  pointer write that points at a non-existent
  `manifests/by-hash/sha256-<hash>.json` produces a
  **dangling pointer** that the engine fails closed on
  per ADR-0007 + ADR-0005's "engine reads are two-step"
  invariant. The subcommand MUST verify the target body
  exists before writing the pointer; this is the
  "no operations under uncertainty" safety property the
  CLI brings over the gsutil workaround.
- **DD-5 — Pointer body recovery: read the target
  manifest's `ruleset_version`.** ADR-0005 §6 commits
  that the pointer carries `ruleset_version` duplicated
  from the referenced manifest body. The subcommand
  must read the target body and copy the field so the
  new pointer JSON is well-formed.
- **DD-6 — Exit-code symmetry with `publish`.** Operators
  scripting rollback should see the same exit codes
  `publish` returns (0 OK / 1 verification / 2
  operational / 3 CAS / 64 usage). The "verification"
  layer for `set-pointer` is the target-exists check
  (DD-4) + the bucket / hash / engine-compat sanity.

---

## Considered Options

### Option 1 — `dq-manifest set-pointer` subcommand with target verification (recommended)

A new subcommand:

```
dq-manifest set-pointer \
  -bucket <bucket> \
  -manifest-hash <sha256-hex> \
  [-dry-run] \
  [-storage-emulator-host <host>]
```

Behavior:

1. **Verify target body exists** at
   `manifests/by-hash/sha256-<hash>.json`. Missing →
   exit 1 (verification fail) with a clear error.
2. **Read target body** + extract `ruleset_version`.
3. **Read current pointer generation** (0 if the
   pointer doesn't exist yet — edge case, normally
   only happens on first publish).
4. **Build pointer JSON** with the target hash, the
   target's `ruleset_version`, and `published_at:
   now()`. Pointer format byte-for-byte matches ADR-0005
   §6.
5. **CAS-write the pointer**. CAS loss → exit 3 (same
   as publish).
6. **On success, emit a structured log line** with
   the target hash, the prior pointer hash (read from
   the prior pointer before the CAS write), the
   pre-generation, and the post-generation. Mirrors
   `publish`'s observability emission.

Safety:

- The verify-target-exists step catches the dangling-
  pointer hazard at the CLI surface, BEFORE the CAS
  write.
- The CAS write inherits the same race-loser semantics
  as publish: the loser surfaces `ErrPreconditionFailed`
  and exits 3.
- No new substrate surface — reuses
  `Store.ReadObject`, `Store.ReadPointerGeneration`,
  `Store.CASWritePointer`.

**Strengths.** Closes the bypass-via-gsutil gap;
preserves ADR-0005 §3 + §4; reuses existing Store
interface; symmetric exit codes; observability emission
matches publish. Operational ergonomics: one command +
two flags.

**Trade-offs.** Adds a new CLI subcommand surface
(~150 LOC of new code; ~100 LOC of tests). Modest cost
for the gap closure.

### Option 2 — Reuse `publish` with a `-rollback` flag

Don't add a new subcommand; extend `publish` with a
`-rollback -manifest-hash <hash>` flag that, when set,
skips the verify + write-bodies steps and only does the
CAS pointer write.

**Strengths.** No new subcommand; reuses publish
plumbing.

**Trade-offs.** Conflates two distinct operations
under one subcommand name. The publish flag set is
already 8 flags; adding a 9th that changes the
operation entirely is operator-hostile (which flags
apply to "publish" vs "rollback"?). The runbook
mental model is "rollback is a different operation
from publish" — the CLI should match. Rejected — the
ergonomics regression outweighs the avoid-new-
subcommand benefit.

### Option 3 — Status quo (keep the runbook §3.B
gsutil workaround)

Don't add a CLI surface; operators continue to use
`gcloud storage cp --if-generation-match`.

**Strengths.** Zero implementation cost.

**Trade-offs.** Bypass-via-gsutil is the explicit
failure mode B2-10 was registered to address. The
runbook §3.B is operator-hostile and bypasses the
CLI contract (DD-1). Rejected — B2-10's expected
output is "CLI design note + subcommand under
`tools/manifest/`", which Option 3 explicitly
doesn't deliver.

---

## Recommendation

**Option 1.** New `set-pointer` subcommand with
target verification.

### Subcommand surface

```
dq-manifest set-pointer \
  -bucket <bucket-name>                    \  required
  -manifest-hash <sha256-hex>              \  required; no `sha256:` prefix
  -dry-run                                 \  optional; verify + log without CAS write
  -storage-emulator-host <host>            \  optional; for local-emulator runs
```

Flag conventions:

- `-manifest-hash` accepts the **hex digest without
  the `sha256:` prefix** — the prefix is part of the
  pointer-file representation, not the CLI input. The
  prefix is added when constructing the pointer body.
  Operators reading the prior pointer with `gcloud
  storage cat` see `sha256:<hex>`; they pass `<hex>`
  to `set-pointer`. The runbook §3 documents this
  convention.
- `-dry-run` runs all verifications + emits the
  planned pointer JSON to stderr without issuing the
  CAS write. Useful for "what would this do?"
  validation in pre-prod.
- `-storage-emulator-host` is identical to `publish`'s
  flag of the same name (local Compose / sandbox
  emulator support).

### Internal design

A new file `tools/manifest/rollback.go` (~120 LOC)
exposes:

```
type Rollback struct {
    store Store
    now   func() time.Time
    logger *slog.Logger
}

type RollbackConfig struct {
    Store  Store
    Now    func() time.Time
    Logger *slog.Logger
}

type RollbackOptions struct {
    TargetHashHex string  // e.g. "abc123..."  (no `sha256:` prefix)
    DryRun        bool
}

type RollbackResult struct {
    TargetHash       string  // the input hash
    TargetRulesetVer string  // copied from target body
    PriorHash        string  // the prior pointer's hash (forensic)
    PriorPointerGen  int64
    PostPointerGen   int64   // 0 in DryRun
}

func NewRollback(cfg RollbackConfig) (*Rollback, error) { ... }
func (r *Rollback) Execute(ctx context.Context, opts RollbackOptions) (*RollbackResult, error) { ... }
```

The execute path mirrors `Publisher.Publish` step 4
exactly + adds a target-verification preamble:

1. Validate `TargetHashHex` matches the
   `^[0-9a-f]{64}$` pattern.
2. Read target body at
   `manifests/by-hash/sha256-<hex>.json`. Missing →
   `ErrVerificationFailed` (exit 1).
3. Unmarshal target body; extract `ruleset_version`.
4. Read current pointer (for forensic prior-hash
   logging); read current pointer generation.
5. If `DryRun`, emit planned pointer JSON, return
   `PostPointerGen=0`.
6. Build pointer JSON, CAS-write. CAS loss →
   `ErrPreconditionFailed` (exit 3).
7. Return populated `RollbackResult`.

### `main.go` wiring

A new `case "set-pointer"` branch in `main.go` parses
flags, builds a `*Rollback`, invokes `Execute`, and
maps result/error to exit code. Exit-code contract
mirrors `publish`:

| Exit | Meaning |
|---|---|
| 0 | rollback OK (or dry-run success) |
| 1 | verification failed (target body missing; invalid hash; malformed target manifest) |
| 2 | operational failure (bucket missing, network, etc.) |
| 3 | CAS precondition failed (pointer moved between read and write; retry) |
| 64 | usage error (missing required flag) |

### Runbook update

`docs/runbooks/manifest-rollback.md` §3 is rewritten:

- **§3 — Procedure** becomes a single primary path
  using `dq-manifest set-pointer`. Three steps:
  recover the target hash from `manifests/by-hash/`,
  invoke `set-pointer -dry-run` to validate, invoke
  `set-pointer` for real.
- **§3.A** (re-publish prior ruleset) drops as the
  primary path; it stays in the runbook as a
  fallback ONLY when the operator wants to
  reconstruct the prior linter pin (rare; mostly
  forensic).
- **§3.B** (gsutil cp workaround) drops as the
  primary path; it stays as a §3.fallback for the
  edge case where `dq-manifest` itself is broken or
  unavailable.

### Why this does NOT reopen ADR-0005

ADR-0005 §3 commits the pointer as the single
mutable control-plane object; §4 commits the CAS
write as the only mutation primitive. B2-10 uses the
same primitive — the subcommand is a new caller, not
a new mechanism. ADR-0005 stays accepted without
amendment.

### Why this does NOT reopen ADR-0001 §C9 (unforgeable linter pin)

The CLI's `set-pointer` operation does not run any
linter; it copies the target manifest's existing
content into the new pointer. The linter-pin contract
applies to NEW manifests being published, not to
rollback. ADR-0001 §C9 stays preserved.

---

## Consequences

1. **A new `set-pointer` subcommand ships in
   `tools/manifest/main.go`.** Reuses the existing
   Store interface; no new substrate surface; no new
   dependency.

2. **A new file `tools/manifest/rollback.go` (~120 LOC)
   carries the `Rollback` orchestration type.** Mirrors
   `Publisher` but for the rollback case.

3. **Unit tests in `tools/manifest/rollback_test.go`
   cover:**
   - Happy path against the in-memory `fakeStore`
     used by `publisher_test.go`.
   - Target-body-missing → `ErrVerificationFailed`.
   - Invalid hex → `ErrVerificationFailed`.
   - Malformed target body → `ErrVerificationFailed`.
   - CAS race-loser → `ErrPreconditionFailed`.
   - Dry-run emits planned pointer JSON; no
     `CASWritePointer` call.

4. **Integration tests in
   `tools/manifest/rollback_integration_test.go`
   (build tag `integration` per ADR-0034) cover:**
   - End-to-end rollback against fake-gcs-server:
     publish manifest A → publish manifest B →
     `set-pointer` to A → verify pointer reads A's
     hash + A's `ruleset_version`.

5. **The manifest-rollback runbook §3 is rewritten**
   per the §"Runbook update" section above. The new
   primary path is `dq-manifest set-pointer`; the
   workarounds are demoted to §3.fallback for edge
   cases.

6. **The CLI exit-code contract gains a fifth
   subcommand-shared mapping.** `set-pointer` returns
   the same `{0, 1, 2, 3, 64}` codes as `publish`.
   Operator wrapper scripts that already key on
   publish exit codes work unchanged for
   `set-pointer`.

7. **The bypass-via-gsutil failure mode is closed.**
   Operators have a first-class CLI surface for the
   rollback operation; `gcloud storage cp
   --if-generation-match` is the §3.fallback only.

8. **ADR-0005 + ADR-0001 are preserved.** The
   subcommand reuses the existing pointer primitive
   without amending either ADR.

9. **B2-10 closes.** The decision-log B2-10 row moves
   to `resolved-adr` (→ ADR-0036). The runbook §2
   precondition's "TBD" reference to the new
   subcommand is replaced with a concrete CLI
   reference.

---

## Open Questions

None blocking.

One deferred item surfaced during drafting and is
explicitly **out-of-scope for current cycle**:

- **OQ-1: `dq-manifest list-pointers` subcommand for
  prior-pointer recovery.** Today operators discover
  prior manifest hashes via `gcloud storage ls
  gs://<bucket>/manifests/by-hash/`. A future CLI
  ergonomic could ship `dq-manifest list-pointers`
  that lists historical pointer generations + their
  hashes from a designated audit log. Reserved until
  concrete operational signal shows the gcloud-based
  discovery is too cumbersome.

---

## Promotion target

`docs/adr/0036-set-pointer-subcommand.md` — ships the
subcommand-surface commitment, the safety-invariant
contract (verify target body before pointer write),
the exit-code mapping, and the runbook-rewrite
posture.
