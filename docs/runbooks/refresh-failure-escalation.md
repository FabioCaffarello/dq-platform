<!-- path: docs/runbooks/refresh-failure-escalation.md -->

# Runbook — Refresh-failure escalation

Respond to the loader's refuse-swap path firing repeatedly:
the engine continues serving against the prior in-memory
manifest, but new manifest publications are not being
adopted.

The loader posture is asymmetric
([ADR-0007](../adr/0007-loader-scheduler-retry-failure-semantics.md)):

- **Startup failure (§CC1)** — the engine **exits**. Crash
  loop is operationally visible; this runbook does not cover
  startup failures.
- **Refresh failure (§CC2)** — the engine **refuses the
  swap**, retains the prior manifest in memory, logs +
  alerts operationally, and continues refresh attempts.
  After N consecutive failures (per B1-2, **TBD**), an
  escalation alert fires.

This runbook covers refresh failures only.

---

## 1. When to use

- The loader emits a `refresh: refuse-swap` log line one or
  more times in succession.
- An operational alert names "loader_refresh_failure" or
  "refuse-swap N-threshold exceeded".
- A manifest publication completed (verified per
  [`manifest-rollback.md`](manifest-rollback.md) §2 pointer
  check) but the engine continues to serve the prior
  ruleset version past one refresh-cadence tick.

## 2. Preconditions

- Read access to engine logs (the loader emits structured
  logs per ADR-0007 §CC14).
- Object-store read access to the manifest bucket
  (`gcloud storage ls`, `gcloud storage cat`).
- The current pointer's hash (from
  `gcloud storage cat gs://<bucket>/manifests/latest.json`)
  and the loader's in-memory hash (from the last successful
  refresh log line).

## 3. Procedure

### 3.A Classify the failure

The loader emits a structured log on every refresh attempt.
On failure the line carries one of three causes:

| Log cause                       | Likely root                              | Go to |
|--------------------------------|------------------------------------------|-------|
| `pointer_fetch_failed`         | Object-store availability or IAM         | 3.B   |
| `manifest_body_fetch_failed`   | Lifecycle purge / pointer/body drift     | 3.C   |
| `manifest_contract_check_failed` | B0-1 contract violation in new manifest | 3.D   |

### 3.B Pointer fetch failed

The loader could not read `manifests/latest.json`.

1. Check object-store availability:
   `gcloud storage ls gs://<bucket>/` — does it list?
2. If yes, check IAM on the pointer object specifically:
   `gcloud storage objects describe
   gs://<bucket>/manifests/latest.json`. Engine service-
   account needs read.
3. If yes, the failure was transient. The loader retries on
   the next tick; if the N-threshold has not yet fired,
   monitor for resolution. If it has fired, the escalation
   alert is already routed.

### 3.C Manifest body fetch failed

The pointer references a hash whose body blob does not
exist.

1. Read the pointer's `manifest_hash`:
   `gcloud storage cat gs://<bucket>/manifests/latest.json`.
2. Check the body's existence:
   `gcloud storage ls
   gs://<bucket>/manifests/by-hash/sha256-<hex>.json`.
3. **If the body is missing** → lifecycle purge bug or
   pointer/body publish-order race. The engine is failing
   closed per
   [ADR-0005](../adr/0005-manifest-publication-semantics.md)
   §CC6. **Roll back the pointer** to a manifest hash whose
   body still exists — see
   [`manifest-rollback.md`](manifest-rollback.md).
4. **If the body exists** → loader IAM on
   `manifests/by-hash/` blobs. Audit IAM.

### 3.D Manifest contract check failed

The loader fetched a manifest whose contract check fails per
[ADR-0001](../adr/0001-engine-rules-compatibility.md) §4:

- Unknown `version:` in a rule (the engine does not support
  that schema version).
- The manifest's declared `schema_versions_present` set
  differs from the set observed across packaged YAMLs.
- The engine's running version is outside the manifest's
  `engine_compatibility` semver range.

Each of these is a **publish-time error** the publisher
should have caught (per ADR-0001 §4 — manifest publisher is
the second of three independent verification gates). The
manifest reaching the loader despite that is itself a bug to
file, separate from this incident.

Immediate remediation:

1. Identify the offending field from the loader log line
   (it names which of the three checks failed and which
   value caused it).
2. **Roll back the pointer** — see
   [`manifest-rollback.md`](manifest-rollback.md). The prior
   manifest in memory is still serving correctly; rollback
   restores agreement between pointer and runtime.
3. Open a bug against the publisher: it should have rejected
   this manifest at publish time.

## 4. Verification

1. **`refresh: success` log line emits** on a subsequent
   refresh tick.
2. **In-memory hash equals pointer hash.** The loader's
   structured log on every refresh carries
   `pointer_hash` and `loaded_hash`; they match.
3. **No operational alert fires on the next N ticks** (where
   N is the alert-window B1-2 — TBD).

## 5. Rollback / escape

The refuse-swap posture is itself the rollback path: the
engine continues to serve against the prior in-memory
manifest while refresh failures repeat. There is **nothing
to roll back** from refuse-swap; the engine is operating in
its safe-default state.

If your remediation in 3.C / 3.D made things worse (e.g.,
the rollback target itself fails contract check), refuse-swap
catches that too — the engine stays on the working
in-memory manifest.

The only situation requiring active escape is **engine
restart with a broken pointer**. At startup the loader fails
closed per ADR-0007 §CC1 (process exit, not refuse-swap).
Before any engine restart while refresh is failing, verify
the pointer either points at a working manifest body or
roll the pointer back first.

## 6. Escalation

- **Pointer fetch fails AND object store appears healthy.**
  Likely IAM regression specific to the engine service
  account. Escalate to SRE.
- **Manifest body purged by lifecycle.** Confirm the
  lifecycle rule (per B1, TBD) is configured correctly;
  retention shorter than reasonable rollback horizon is a
  policy bug. Escalate to platform-team plus SRE.
- **Contract check fails on a manifest that passed the
  publisher.** Bug in the publisher's verification gates.
  Escalate to platform-team.
- **Crash-loop on engine restart while this incident is
  open.** Stop. Do not restart the engine until the pointer
  is verified healthy (refuse-swap protects you only while
  the engine is running; startup is strict).
