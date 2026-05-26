<!-- path: docs/runbooks/manifest-rollback.md -->

# Runbook — Manifest rollback via CAS pointer write

Revert the engine to a prior known-good ruleset by pointing
`manifests/latest.json` at an earlier manifest hash.

The publication model — content-addressed manifest body at
`manifests/by-hash/sha256-<hex>.json`, CAS-protected pointer
file at `manifests/latest.json` — is the source of safety per
[ADR-0005](../adr/0005-manifest-publication-semantics.md) §4.
Rollback exploits the same primitive: prior manifest bodies
are immutable, so re-pointing at them is well-defined.

---

## 1. When to use

- A newly-published manifest deploys a regression: a rule
  fails too aggressively, a check kind is misconfigured, or
  the ruleset version itself was incorrect.
- The prior ruleset version is known good and its manifest
  hash is recoverable (either from `manifests/by-hash/` or
  from a prior `dq-manifest publish` log line).

Do **not** use this runbook for transient loader failures
(use [`refresh-failure-escalation.md`](refresh-failure-escalation.md))
or for stuck executions (use
[`orphan-run-remediation.md`](orphan-run-remediation.md)).

## 2. Preconditions

- Object-store write access to the manifest bucket (the
  bucket configured per environment in
  `engine/internal/env/{local,qa,prod}.go`).
- The prior manifest's hash. Recover by listing the bucket:
  `gcloud storage ls gs://<bucket>/manifests/by-hash/` — every
  immutable manifest body is here. The current pointer's hash
  is the first line of `manifests/latest.json` (read it with
  `gcloud storage cat`).
- The current pointer's generation number (for the CAS
  conditional write). `gcloud storage ls -L
  gs://<bucket>/manifests/latest.json` prints it under
  `Generation`.

## 3. Procedure

The primary path is `dq-manifest set-pointer` — the rollback
CLI surface committed by
[ADR-0036](../adr/0036-set-pointer-subcommand.md). It runs
the same CAS-conditional pointer write that `publish`
issues, but reads the target body's `ruleset_version`
instead of re-computing it. The subcommand verifies the
target manifest body exists before issuing the CAS write,
so a typo'd or purged hash fails closed at the CLI surface
rather than producing a dangling pointer.

1. **Recover the target hash.** List the immutable manifest
   bodies and pick the prior known-good:

   ```
   gcloud storage ls gs://<bucket>/manifests/by-hash/
   ```

   Each entry is `manifests/by-hash/sha256-<hex>.json`. The
   `<hex>` portion is what `set-pointer` takes (lowercase
   hex, no `sha256:` prefix). To confirm which ruleset
   version a hash carries, read the body:

   ```
   gcloud storage cat \
     gs://<bucket>/manifests/by-hash/sha256-<hex>.json \
     | jq .ruleset_version
   ```

2. **Dry-run the rollback** to validate inputs and inspect
   the planned pointer JSON before committing:

   ```
   dq-manifest set-pointer \
     -bucket <bucket> \
     -manifest-hash <hex> \
     -dry-run
   ```

   On success, the log line shows `target_hash`,
   `target_ruleset_version`, `prior_hash`,
   `prior_ruleset_version`, and the prior pointer
   generation. No pointer mutation occurs. Exit 1 indicates
   the target body is missing or malformed; fix the hash
   and retry.

3. **Apply the rollback** by dropping `-dry-run`:

   ```
   dq-manifest set-pointer \
     -bucket <bucket> \
     -manifest-hash <hex>
   ```

   On exit 0, the pointer was CAS-rewritten to the target
   hash and carries the target's `ruleset_version`. Exit 3
   indicates the pointer moved between the read and the
   write (a concurrent publisher won the race) — re-run
   `set-pointer` to retry; the CAS primitive guarantees the
   retry is safe.

### 3.fallback.A — re-publish the prior ruleset version (forensic)

This path is rarely needed; reserved for the case where
the operator wants to reconstruct the prior linter pin
exactly (e.g., to confirm a prior manifest was reproducible
from git for audit). If the prior linter binary digest
is unrecoverable, the re-published manifest hash will
differ from the original — at which point `set-pointer`
(§3 above) is the correct tool.

1. `git checkout <prior-tag>` (the tag matching the prior
   `ruleset-version`).
2. Run the same `dq-manifest publish` command that produced
   the original manifest:

   ```
   dq-manifest publish \
     -bucket <bucket> \
     -ruleset-version <prior-ruleset-version> \
     -engine-compatibility "<prior-range>" \
     -linter-used <prior-lint-version>
   ```

3. `dq-manifest` produces the same `manifests/by-hash/...`
   body (content-addressed; if the inputs match, the hash
   matches) and CAS-writes the pointer.

### 3.fallback.B — gsutil pointer rewrite (CLI unavailable)

Reserved for the edge case where `dq-manifest` itself is
broken or unavailable (binary missing, dependency outage).
Bypasses the CLI contract — prefer §3 above whenever
possible.

1. Read the current pointer's `Generation`:
   `gcloud storage ls -L gs://<bucket>/manifests/latest.json`
2. Build a new pointer file locally. The pointer shape per
   [ADR-0005](../adr/0005-manifest-publication-semantics.md)
   §"pointer body" is
   `{ pointer_version, manifest_hash, ruleset_version, published_at }`.
   Download the current `manifests/latest.json` as a starting
   template, then mutate three fields:
   - `manifest_hash` → the prior hash (the rollback target).
   - `ruleset_version` → read the prior manifest body at
     `gs://<bucket>/manifests/by-hash/sha256-<prior-hex>.json`
     and copy its `ruleset_version` field.
   - `published_at` → current UTC timestamp in RFC 3339
     format.
3. Write with the `if-generation-match` precondition:

   ```
   gcloud storage cp ./new-pointer.json \
     gs://<bucket>/manifests/latest.json \
     --if-generation-match=<current-generation>
   ```

   This either succeeds (pointer rewritten) or fails with
   `412 Precondition Failed` (someone else moved the pointer
   in the meantime — re-read and retry).

## 4. Verification

1. **Pointer points at the prior hash.**
   `gcloud storage cat gs://<bucket>/manifests/latest.json`
   shows `manifest_hash: sha256-<prior-hex>`.
2. **Loader refreshed to the prior manifest.** Engine logs
   emit a refresh line like `loader: refresh: pointer changed
   to <prior-hash>` within one refresh-cadence tick (per
   `DQ_LOADER_REFRESH_INTERVAL`).
3. **Engine `/readyz` still returns 200.** The engine
   continued serving in-flight executions against the prior
   manifest while the swap happened (per
   [ADR-0007](../adr/0007-loader-scheduler-retry-failure-semantics.md)
   §CC2 refuse-swap semantics; rollback is the inverse).
4. **A trigger executed against the rolled-back ruleset
   produces the expected `dq_executions` row.** Run
   `make demo-p6` against the local Compose substrate to
   smoke; in prod, trigger against a low-stakes entity first.

## 5. Rollback / escape

If the rollback itself breaks the engine (e.g., the prior
manifest's `engine_compatibility` range excludes the running
engine version, per
[ADR-0001](../adr/0001-engine-rules-compatibility.md) §C6):

- The engine **fails closed** (refuse-swap fires; the prior
  manifest in memory is the one that was running before the
  rollback attempt). No data corruption, but the rolled-back
  ruleset is not in effect.
- Either re-roll forward to the previous current manifest
  (same procedure, opposite hash), or roll back further to a
  manifest with a compatible `engine_compatibility` range.

If the prior `manifests/by-hash/<hash>.json` blob was purged
by lifecycle (per the B1-deferred retention policy — see
[ADR-0005](../adr/0005-manifest-publication-semantics.md)
notes), the pointer rewrite produces a **dangling pointer**
that the engine fails closed on per ADR-0005 §CC6. There is
no recovery from this state via rollback; re-publish forward
to a working manifest.

## 6. Escalation

- **CAS precondition fails repeatedly.** Pointer is being
  rewritten by another process. Find the writer
  (`gcloud storage objects describe` shows the metadata
  generation chain) before retrying. Escalate to
  platform-team.
- **Object-store IAM denies the rewrite.** Escalate to SRE.
- **Engine refuses to load any historical manifest.** The
  engine version itself may have crossed a compatibility
  boundary. Escalate to platform-team; an engine roll-back
  may be needed in addition to the manifest rollback.
