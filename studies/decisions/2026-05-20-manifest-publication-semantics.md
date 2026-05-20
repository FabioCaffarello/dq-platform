<!-- path: studies/decisions/2026-05-20-manifest-publication-semantics.md -->

# B0-5 — Manifest Publication Semantics

## Metadata

- B0 reference: B0-5 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md).
- Status: draft (Wave 1, session 3).
- Last updated: 2026-05-20.
- Upstream resolved: B0-1 (engine ↔ rules compatibility,
  [`2026-05-20-engine-rules-compatibility.md`](./2026-05-20-engine-rules-compatibility.md)).
- Downstream open: B0-7 (loader / scheduler / retry failure
  semantics).
- Promotion target: see final section.

---

## Context

The platform's runtime never reads source YAMLs directly. It reads a
**manifest** — a derived artifact, published by CI, that declares
which rules are active. The boundary contract document
([`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
§"Surface 3 — The Manifest") proposes the shape of that artifact and
sketches a write-new-then-swap publication pattern. The system
architecture document
([`04-system-architecture.md`](../foundation/04-system-architecture.md)
§"PAT-1 — Fail-fast registry loading") requires that the engine
either loads a manifest completely and correctly, or refuses — no
partial loading, no silent skipping.

B0-1 (resolved 2026-05-20) locked the manifest's **contents**: it
must carry three semantic commitments (set of schema versions
present in the ruleset; engine-compatibility expression; reference
to the linter release that validated the manifest, used only for
audit/reconstruction per G4). B0-1 explicitly deferred the
manifest's *shape, naming, encoding, and publication mechanics* to
this study.

B0-5 — as recorded in the decision log:

> What guarantees atomic, reversible ruleset publication to object
> storage?

This study locks **publication semantics**: the publication
primitive, the atomicity contract the engine reads against, the
race-resolution rule, the rollback procedure, the history-retention
shape, and the JSON field names B0-1 explicitly punted to here.

What this study does **not** decide:
- The publisher tool's source layout (Wave 3, lives in `tools/`).
- Loader failure modes when contract checks fail (B0-7).
- Per-environment lifecycle/retention parameters (B1).
- Cryptographic signing beyond checksums (B1-8).
- Local-emulator parity for object storage (Wave 2).

The decision matters because manifest publication is the **only**
control-plane path between authored rules and runtime execution.
Per operational imperative #3 in
[`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
("visible failure over silent degradation"), the publication path
must fail loudly or not at all — a half-published manifest that the
engine consumes as "complete" is the most dangerous failure mode
this platform can produce.

---

## Decision Drivers

The decision must satisfy the following, in priority order.

1. **D1. Engine never observes a partial manifest.** What the engine
   reads is either the complete prior state or the complete new
   state. There is no third option. This is PAT-1 stated as a
   constraint on the publisher.

2. **D2. Every manifest ever published must be retrievable.** G4
   (every execution must be reconstructable —
   [`04-system-architecture.md`](../foundation/04-system-architecture.md))
   requires that "which ruleset version was used" is answerable for
   any historical run. Manifest history is not an audit feature —
   it is a correctness requirement.

3. **D3. Concurrent publishers must not silently overwrite each
   other.** Two CI jobs from two near-simultaneous tag pushes can
   race. The publication primitive must either order them or refuse
   one — never accept both writes with a silent winner.

4. **D4. Rollback must be possible without re-running CI.** When a
   bad manifest is published (rules pass linter but trigger
   unintended behavior), an operator must be able to revert to a
   prior manifest in seconds, from an admin path, without
   re-resolving git state.

5. **D5. Publication failures must be loud and recoverable.** A
   network drop mid-publish must leave the system in a state where
   either (a) the prior manifest is still active and the new one is
   absent, or (b) the new manifest exists but is not yet reachable
   by the engine — never a state where the engine reads a corrupt
   or partial object. Stated positively: every observable state of
   the object store is a valid one. Per operational imperative #3
   in [`05-operational-discipline.md`](../foundation/05-operational-discipline.md).

6. **D6. History retention must have an explicit lifecycle.**
   Manifests are small (tens of KB typical) but unbounded growth is
   still a posture, not an accident. Cost is a first-class
   constraint (P4); the lifecycle policy must be stateable, even if
   exact parameters are deferred to B1.

7. **D7. B0-1 inputs are fixed.** The manifest's contract block
   carries three semantic commitments (schema-version set,
   engine-compatibility expression, linter reference for audit).
   The publisher must, before producing any manifest, verify the
   three things B0-1's C5 commits to. B0-5 takes both as input.

8. **D8. Substrate capability posture must be explicit, and the
   publication primitive must work in the local development
   environment.** Per
   [`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
   §"Local Development Posture", object storage is fully emulated
   locally. This study assumes the platform's object-storage
   substrate provides (a) single-object atomic writes and (b)
   generation-or-etag-conditional writes. (a) is universally
   available across modern object stores. (b) is widely but not
   universally available — it became broadly native to S3-shaped
   stores only recently, and not every object-store implementation
   exposes equivalent semantics; the assumption is therefore
   load-bearing. Every option below relies on it for race
   resolution. If a future substrate selection (Wave 2 emulator
   scope, Wave 3 production substrate, or any later replacement)
   chooses a store without (b), B0-5 is reopened — see CC13.

---

## Considered Options

Each option is described in terms of the **publication primitive**
(how the new state becomes the active state) and the **history
shape** (how prior manifests are preserved).

### Option A — Single-object swap (the foundation-document baseline)

The publisher writes the new manifest under a timestamped path,
then **copies** it to `manifests/latest.json`. The engine reads
`manifests/latest.json` on every refresh. Prior `latest.json` is
**moved** into `manifests/history/<timestamp>.json` before the
swap.

```
gs://<bucket>/rules/manifests/latest.json           # active
gs://<bucket>/rules/manifests/history/2026-...json  # all prior
gs://<bucket>/rules/current/entities/<entity>.yaml  # YAMLs of latest
gs://<bucket>/rules/history/<entity>/<...>.yaml     # YAMLs of prior
```

Atomicity per single-object overwrite (commodity guarantee in every
modern object store).

**Trade-offs.**

- Pro: simplest mechanism; one copy, one move, one engine read per
  refresh.
- Pro: D1 satisfied (single-object swap is atomic).
- Pro: D2 satisfied (history directory preserves everything),
  conditional on the history-move step succeeding before the swap.
- Con: violates D3 — two publishers writing to `latest.json` near
  simultaneously each succeed; the later overwrite wins; the
  earlier publisher reports success but its manifest is lost. No
  way to detect from the publisher side.
- Con: weakens D5 — the history-move-then-swap sequence is not
  atomic as a pair. If the move succeeds but the swap fails, the
  prior manifest is in `history/` and `latest.json` is unchanged;
  the next publisher overwrites without re-moving (history loses
  the moved entry on the next cycle, depending on implementation).
  These are recoverable edges but they require defensive operator
  scripts.
- Con: D4 (rollback) requires copying `history/<old>.json` back to
  `latest.json` and re-moving the currently-live `latest.json`
  into history. Two operations; if interrupted, the system is in
  an in-between state.

Acceptable as a starting point but does not solve the concurrency
problem. The foundation document itself flags this in
[`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
§"How it is published atomically": *"The exact mechanism (versioned
objects, generation conditions, lease semantics) is a Wave 1 B0
decision."* This option does not pick those mechanisms; B and C
below do.

### Option B — Single-object swap with generation-conditional write

Same physical layout as A, but the overwrite of `latest.json` is
**conditional on its current generation matching what the publisher
read** at the start of its run. If two publishers race, only the
one whose precondition holds succeeds; the other receives a
precondition-failed error and must abort or retry.

```
# at publisher start
current = head(manifests/latest.json) -> generation G0
# ... prepare new manifest ...
# at swap time
write(manifests/latest.json, new_content, if_generation_match=G0)
# succeeds only if generation is still G0
```

History preserved as in A (move-then-swap).

**Trade-offs.**

- Pro: D3 satisfied — concurrent publishers cannot silently
  overwrite each other. The losing publisher gets a clear, loud
  error.
- Pro: D1, D5 retained (atomicity is unchanged from A).
- Pro: D8 — generation-conditional writes are a commodity
  primitive present in every modern object store and supported by
  every reputable emulator.
- Con: D2's history dependency on the move-then-swap sequence is
  still present. A crash between move and swap leaves history with
  the moved entry but `latest.json` unchanged — operator must
  reconcile.
- Con: D4 rollback still requires two operations (move current
  `latest.json` to history, copy chosen historical manifest to
  `latest.json`). Both must succeed; if interrupted, in-between
  state exists.
- Con: the manifest object itself is mutable in place (every publish
  overwrites it). If a publisher writes a corrupt object before
  the engine refreshes, the engine sees the corruption. The
  precondition protects the *swap*, not the *content*.

Cleaner than A on concurrency; same weak history coupling.

### Option C — Lease-based publication

The publisher acquires a publication lease via a lease object with
a TTL (e.g., writes `manifests/lease.json` with a precondition that
it does not currently exist or has an expired TTL). Holds the lease,
performs publication, releases the lease. Other publishers attempting
to acquire while the lease is held abort or wait.

**Trade-offs.**

- Pro: D3 satisfied via mutual exclusion rather than CAS.
- Pro: enables multi-object atomicity in principle (the holder
  knows nobody else is writing) — could be used to make
  move-then-swap effectively atomic at the workflow level.
- Con: introduces distributed-lock complexity: lease TTL choice,
  what happens if the holder crashes during publication, lease
  renewal, clock-skew between publisher and object store.
- Con: D5 weakened — a crashed lease-holder leaves the lease
  effectively held until TTL expiry; new publishers are blocked
  for that window. Visible failure, but slow recovery.
- Con: D8 emulator parity is fine in principle, but the lease
  semantics are subtler to test than CAS.
- Con: solves a problem (multi-object atomicity) that the next
  option avoids by design.

Reject as over-engineering for the problem we have.

### Option D — Content-addressed manifests with CAS-protected pointer

The manifest object is written to a **content-hash-keyed** path
(`manifests/by-hash/sha256-<hex>.json`). The manifest is **never
overwritten** — every publish produces a new immutable object. A
small pointer file at `manifests/latest.json` contains only the
hash and metadata enough to identify which manifest is currently
active. The pointer is the only thing that races, and it is
written with generation-conditional precondition (CAS) for D3.

```
gs://<bucket>/rules/manifests/latest.json
    # pointer file, mutable, CAS-protected
gs://<bucket>/rules/manifests/by-hash/sha256-abc...def.json
    # manifest content, immutable, written once
gs://<bucket>/rules/manifests/by-hash/sha256-...older.json
    # every prior manifest, still here
gs://<bucket>/rules/yamls/by-hash/sha256-...yaml
    # rule YAMLs, also content-addressed
```

Pointer file shape (illustrative; this study commits these names):

```jsonc
{
  "pointer_version": 1,
  "manifest_hash": "sha256:abc...def",
  "ruleset_version": "rules-v2.4.7",
  "published_at": "2026-05-20T14:32:11Z"
}
```

Manifest content shape (illustrative; this study commits the named
fields, satisfying B0-1's three semantic commitments):

```jsonc
{
  "manifest_version": 1,
  "ruleset_version": "rules-v2.4.7",
  "schema_versions_present": [1],
  "engine_compatibility": ">=2.0.0 <3.0.0",
  "linter_used": "tools-lint-v2.3.1",
  "generated_at": "2026-05-20T14:32:11Z",
  "rules": [
    {
      "entity": "customer",
      "yaml_hash": "sha256:...",
      "yaml_path": "yamls/by-hash/sha256-...yaml"
    }
  ]
}
```

**Publication sequence:**

1. Publisher runs B0-1's three pre-publish verifications (C5 of
   the B0-1 study). On any failure: abort, no writes occur.
2. Publisher writes each rule YAML to `yamls/by-hash/sha256-<hex>.yaml`
   if not already present. **Idempotent by content** is a
   sub-commitment of CC2: writing the same content twice produces
   the same hash and the same path (a no-op); writing different
   content produces a different path. Re-running a publication
   from the same inputs never produces a different object at the
   same path.
3. Publisher computes the manifest content and its hash, writes
   `manifests/by-hash/sha256-<hex>.json` if not already present.
   Same idempotent-by-content sub-commitment as step 2. The
   manifest is immutable from the moment of write.
4. Publisher reads the current `manifests/latest.json` to obtain
   its current object-store generation `G0` (and verifies the
   currently-pointed manifest hash, for the rollback-detection
   path).
5. Publisher writes a new `manifests/latest.json` with the new
   `manifest_hash`, conditional on `if_generation_match=G0`. If the
   precondition fails, abort with a clear concurrency error.

**Engine read sequence:**

1. Read `manifests/latest.json` — one small object, atomic.
2. Read `manifests/by-hash/sha256-<hash>.json` — the manifest. If
   it does not exist, the pointer is dangling: fail closed (B0-7's
   domain).
3. Verify B0-1's three contract checks (C6 of the B0-1 study). If
   any fails, fail closed.
4. Fetch each rule YAML by its `yaml_hash`. Verify hash on read.
5. Index by entity (PAT-1).

**Trade-offs.**

- Pro: D1, D2, D3, D4, D5 satisfied directly.
- Pro: D2 is **automatic** — every manifest is preserved by its
  hash; no separate move-to-history step is needed; history
  cannot be lost by a crash mid-publish.
- Pro: D4 rollback is **a single pointer write** — change
  `latest.json` to point to a prior hash. Atomic and instant.
- Pro: D5 — the manifest object is immutable from the moment of
  write. A network drop during pointer update leaves the prior
  pointer (and thus the prior valid manifest) active. A network
  drop during manifest content write leaves an orphaned
  `by-hash/<...>.json` (cleanable later) and an unchanged pointer.
- Pro: D8 — relies only on commodity primitives (single-object
  atomic write, generation-conditional write, content
  addressability via client-computed hash). All available in
  emulators.
- Pro: idempotent retries — re-publishing the same input twice
  produces the same hashes and the same pointer update, with no
  spurious state changes.
- Con: engine startup is two reads instead of one (pointer +
  manifest). Both are small; the cost is negligible. Refresh path
  can short-circuit if the pointer's hash has not changed.
- Con: orphaned content-hash objects accumulate over time (a
  manifest hash that was written but whose pointer update lost
  its CAS race is orphaned). Cleanup is a B1 concern (lifecycle
  policy on the `by-hash/` prefix).
- Con: an operator who wants to "edit a manifest" cannot do so
  directly — the manifest is immutable. They must publish a new
  one. This is the intended posture but is a real change from
  Options A/B.

---

## Recommendation

Adopt **Option D** — content-addressed manifests with a
CAS-protected pointer file.

The recommendation is grounded in:

- foundation doc
  [`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
  §"Surface 3 — The Manifest" (manifest is a derived artifact, not
  authored; one-read consumption on engine startup);
- foundation doc
  [`04-system-architecture.md`](../foundation/04-system-architecture.md)
  §"PAT-1 — Fail-fast registry loading" (no partial loading,
  fail-fast on any inconsistency) and §"G4 — Every execution must
  be reconstructable" (history is a correctness requirement);
- foundation doc
  [`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
  §"Operating Posture" (determinism, visible failure);
- prior decision
  [`2026-05-20-engine-rules-compatibility.md`](./2026-05-20-engine-rules-compatibility.md)
  (B0-1 — three semantic commitments in the manifest, three
  pre-publish verifications, byte-equality CI gate already in
  place).

The specific commitments beyond what those documents state are
**new contribution proposed here, requires review**:

1. The manifest is **immutable** from the moment of write — no
   in-place overwrite of the same object key. New manifest →
   new content-hash key.
2. The active manifest is identified by a **separate small
   pointer file** (`manifests/latest.json`) whose only mutable
   responsibility is "which hash is live". CAS protects the
   pointer.
3. Rule YAMLs are **content-addressed** alongside manifests
   (`yamls/by-hash/sha256-<hex>.yaml`), enabling immutable
   provenance from manifest → YAML and removing the need for a
   parallel "current/" vs "history/" tree on the YAML side.
4. The manifest carries the field names committed below (Option D's
   illustrative blocks become this study's commitments), satisfying
   B0-1's three semantic commitments under the specific names
   `schema_versions_present`, `engine_compatibility`, and
   `linter_used`. B0-1's note that field names are this study's
   call is hereby resolved.

---

## Consequences

Adopting this recommendation commits the platform to the following.

**CC1. Object-store layout is fixed.** The publisher writes
exclusively under these prefixes:

```
manifests/latest.json
manifests/by-hash/sha256-<hex>.json
yamls/by-hash/sha256-<hex>.yaml
```

The `sha256-` prefix on the `by-hash/` paths is a deliberate
commitment: the hash algorithm is encoded in the path layout
itself, so an object's path identifies its algorithm. This is
consistent with the algorithm commitment in CC11 and means a
future migration to a successor algorithm (OQ-5) becomes a
coexistence of differently-prefixed paths, not a path-rewrite.
No other prefix is used by the publisher. Any code that reads
manifests reads from these prefixes; any code that lists prefixes
for discovery is a bug.

**CC2. Manifests are immutable.** Once a `manifests/by-hash/sha256-<hex>.json`
exists, it is never modified, never deleted by the publisher.
Lifecycle deletion (if any) is a separate, audited operator
action governed by B1 retention policy. The same applies to
`yamls/by-hash/sha256-<hex>.yaml`.

**CC3. The pointer file is the single mutable control-plane
object.** Every "publish a new manifest" operation and every
"rollback to a prior manifest" operation is **exactly one
generation-conditional write to `manifests/latest.json`**. There
is no second mutable object in the publication path.

**CC4. Concurrent publishers fail loudly.** Two publishers
running near simultaneously will both succeed at writing their
content-hash objects (idempotent if they computed the same
content; harmless if they computed different content because both
hashes are preserved) and **at most one will succeed at the
pointer write**. The loser receives a precondition-failed error
and surfaces it as a publication failure. No silent overwrite.

**CC5. Rollback is a single pointer write, bounded by lifecycle
retention.** An operator holding rollback authority can revert to
**any prior manifest still within the lifecycle retention window
declared in CC9** by issuing one generation-conditional write to
`manifests/latest.json` referencing the chosen historical
`manifest_hash`. No CI rerun, no second-object reconciliation, no
in-between state. A manifest whose `by-hash/` object has been
purged by the lifecycle policy is not rollback-eligible; an
operator attempting to point at a purged hash produces a dangling
pointer that the engine fails closed on per CC6. The exact
operator path (admin endpoint, CLI subcommand) is Wave 3.

**CC6. Engine reads against this layout are necessarily two-step.**
A pointer read followed by a manifest read keyed by the pointer's
hash is inherent to the content-addressed shape; this study commits
the shape, not the cadence. Hash verification at read time is
load-bearing for the integrity guarantee — any engine implementation
that elides hash verification on the manifest object or on a
referenced YAML object breaks the contract that content addressing
provides. Refresh cadence, caching behavior, short-circuit logic
(e.g., skipping the manifest read when the pointer hash is
unchanged), and the failure response when verification mismatches
are B0-7.

**CC7. B0-1's three pre-publish verifications run before any
write occurs.** The publisher's sequence is
*verify-then-write-content-then-CAS-pointer*. If verification
fails, no `by-hash/` objects are written. This satisfies B0-1 C5
without requiring the publisher to clean up after itself on
verification failure.

**CC8. The manifest header reachable via the pointer must carry
sufficient information for B0-1's C6 contract checks to run before
YAML content is fetched.** The committed manifest shape (CC11)
places `schema_versions_present`, `engine_compatibility`, and
`linter_used` in the manifest body — one read away from the
pointer, before any YAML hash needs to be resolved. The exact
ordering of contract checks vs. YAML fetches, and the failure
response when a check fails, is B0-7. B0-5 commits only that the
shape does not force B0-7 to read YAML bytes before discovering a
contract violation.

**CC9. Lifecycle policy is explicit, owned by the platform team,
declared in `deploy/`.** Content-addressed objects accumulate. The
platform commits to a **declared** lifecycle policy on the
`by-hash/` prefixes (retention duration, conditional deletion).
The policy is declared as configuration under `deploy/` (exact
file path is a Wave 3 implementation concern) and owned by the
platform-team CODEOWNERS group, the same group that owns the
rest of `deploy/` per foundation doc
[`02-monorepo-topology.md`](../foundation/02-monorepo-topology.md)
§"Ownership Boundaries". The exact retention duration, the
trigger for deletion (age, reference-count, explicit operator
action), and the per-environment overrides are B1 — see OQ-1.

**CC10. Orphaned hashes are recoverable, not corrupting.** A
publisher that writes a content-hash object and then loses its
pointer CAS race leaves an orphaned hash. The orphan does not
affect engine behavior (it is unreachable from any pointer) and
costs only its storage footprint until lifecycle policy purges
it. No special recovery action is required.

**CC11. Every committed field name in the manifest and pointer,
and the hash algorithm.** This study commits the following
identifiers and shapes. Items grounded in B0-1 or in the
foundation contract document are noted; items beyond those
sources are flagged **new contribution proposed here, requires
review**.

*Manifest body fields:*

- `manifest_version` (integer) — meta-version of the manifest
  schema itself, distinct from the DSL schema version. **New
  contribution proposed here, requires review.**
- `ruleset_version` (string) — the released ruleset tag, e.g.
  `rules-v2.4.7`. Foundation doc
  [`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
  §"Surface 3 — The Manifest" already names this field; reaffirmed.
- `schema_versions_present` (array of integers) — set of schema
  versions appearing across the packaged YAMLs. Grounded in B0-1
  (one of the three semantic commitments); B0-1 deferred the name
  here, name committed.
- `engine_compatibility` (string, syntax deferred to B0-1's OQ-2
  chain) — expression of which engine versions accept this
  manifest. Grounded in B0-1; name committed.
- `linter_used` (string) — identifier of the linter release that
  validated this manifest, audit-only per B0-1's C10. Grounded in
  B0-1; name committed.
- `generated_at` (RFC 3339 timestamp string) — publish-time
  timestamp, recorded in the manifest body so the manifest is
  self-describing without object-store metadata. **New
  contribution proposed here, requires review.**
- `rules` (array of objects) — list of active rules. Each entry
  has the shape `{ entity: string, yaml_path: string, yaml_hash: string }`.
  **New contribution proposed here, requires review** — this
  supersedes the wording in foundation doc
  [`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
  §"Surface 3 — The Manifest", which used `path` and `checksum`.
  The rename `path` → `yaml_path` and `checksum` → `yaml_hash`
  aligns the manifest with the content-addressed `yamls/by-hash/`
  prefix introduced in CC1.

*Pointer file fields:*

- `pointer_version` (integer) — meta-version of the pointer
  schema. **New contribution proposed here, requires review.**
- `manifest_hash` (string, sha256 hex with `sha256:` prefix) — the
  content hash of the currently-active manifest. **New
  contribution proposed here, requires review** — the pointer-file
  concept is itself new.
- `ruleset_version` (string) — same value as the referenced
  manifest's `ruleset_version`, duplicated for fast identification
  without fetching the manifest body. **New contribution proposed
  here, requires review.**
- `published_at` (RFC 3339 timestamp string) — pointer-write
  timestamp; may differ from the referenced manifest's
  `generated_at` if a prior manifest is re-pointed (rollback).
  **New contribution proposed here, requires review.**

*Hash algorithm:*

- The hash algorithm for content addressing of manifests and YAMLs
  is **sha256**, throughout this study and across all object-store
  paths (CC1). **New contribution proposed here, requires review.**
  See OQ-5 for migration to a successor algorithm.

B0-1 explicitly deferred all manifest field naming to this study;
this consequence is the resolution.

**CC12. Local development uses the same publication primitive.**
Per
[`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
§"Local Development Posture", object storage is fully emulated
locally. The publisher uses the same code path against the local
emulator as it does against production. No mode flag, no skipped
verifications, no alternate primitive in dev. This protects
against "works on my laptop" drift.

**CC13. Substrate selection is a checkpoint against D8.** Any
decision selecting or changing the platform's object-storage
substrate — Wave 2 emulator scope, Wave 3 production substrate,
or any later replacement — must verify the substrate provides
(a) single-object atomic writes and (b)
generation-or-etag-conditional writes. If a substrate without (b)
is selected, this study is reopened; the publication primitive
may need to be replaced with a lease-based approach (Option C
territory, with the trade-offs noted there) or with an auxiliary
coordinator outside object storage. The substrate selection
process must therefore reference this checkpoint explicitly. **New
contribution proposed here, requires review.**

---

## Open Questions

- **OQ-1. Lifecycle retention parameters for `by-hash/` prefixes.**
  How long content-addressed manifests and YAMLs are retained
  before lifecycle deletion is **out-of-scope for current cycle**
  — it is a parameter set, not a shape decision. Add a B1 row if
  no existing row covers it (B1-6 currently covers failed-sample
  retention; manifest/YAML retention is distinct).

- **OQ-2. Operator rollback endpoint.** The exact admin endpoint
  (HTTP method, path, authentication, audit-log shape) for
  "rollback to hash H" is **out-of-scope for current cycle** — it
  is a Wave 3 implementation concern. CC5 commits only to "one
  CAS-protected pointer write".

- **OQ-3. Cryptographic posture beyond checksums.** Whether the
  pointer file, the manifest, or both carry signatures (in
  addition to the content hashes that are intrinsic to E) is
  **out-of-scope for current cycle** — it is B1-8 (manifest
  cryptographic posture).

- **OQ-4. Refresh cadence and short-circuit behavior.** How often
  the engine re-reads `manifests/latest.json` and whether the
  engine skips re-reading the manifest body when the pointer's
  hash is unchanged is **out-of-scope for current cycle** — it is
  a B1 operational parameter, with the constraint that any cache
  must be coherent across all engine replicas.

- **OQ-5. Hash algorithm migration path.** The current algorithm
  commitment is sha256 (see CC11). If sha256 is ever deprecated or
  upgraded (sha512, BLAKE3, or another successor), how a
  mixed-hash store is handled — coexistence of differently-prefixed
  paths per CC1, rewriting, dual-pointer windows — is
  **out-of-scope for current cycle**. A migration ADR is opened
  if and when needed; this OQ defers only the migration question,
  not the algorithm choice.

- **OQ-6. Reactivation of an in-flight publication after a
  crashed publisher.** If a CI publisher crashes after writing
  some `by-hash/` objects but before the pointer write, whether a
  subsequent CI run can resume or must restart is
  **out-of-scope for current cycle** — the writes are idempotent
  by content, so resume is naturally safe; the operational
  guidance is Wave 3.

No open question in this list blocks the publication-semantics
shape. All items above are parameters or operational details on
top of the locked shape.

---

## Promotion target

This study is promoted during Wave 3 to:

    docs/adr/0005-manifest-publication-semantics.md

The `0005` is provisional and assigned at promotion time. If the
Wave 3 ADR numbering convention orders by promotion date rather
than by B0 sequence, the number changes; the slug
(`manifest-publication-semantics`) does not. This follows the
same convention adopted in
[`2026-05-20-engine-rules-compatibility.md`](./2026-05-20-engine-rules-compatibility.md)
(B0-1, Promotion target section).

The MADR ADR rewrites this study for an external-reviewer audience
(no `studies/` back-references per R8), folds in any updates from
B0-7 (loader failures) and B1 lifecycle decisions that intersect
with the publication shape, and updates
[`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
§"Surface 3 — The Manifest" to reference the ADR and reflect the
content-addressed layout.
