<!-- path: docs/security/manifest-cryptographic-posture.md -->

# Manifest Cryptographic Posture

> **Status:** v1 posture is **deferral** — no cryptographic
> signatures are added to manifests beyond the sha256 checksums
> already committed by ADR-0005. The authoritative commitment is
> [ADR-0030](../adr/0030-manifest-cryptographic-posture.md); this
> note is the operator-facing summary.

---

## What this note is for

Operators reading this note are typically here for one of
three reasons:

- **Investigating a manifest-related incident** (a manifest the
  loader accepted turned out to be wrong; or the loader rejected
  a manifest the publisher produced).
- **Evaluating a "should we add signatures to the manifest?"
  request** (from a contributor, an auditor, or a security review).
- **Onboarding to the platform's security posture** (what is
  defended; what is not; what the response posture is).

The note gives you the threat-model table, the trigger conditions
that would reopen the cryptographic-posture decision, and the
implementation path the platform commits to if signatures are
ever needed.

---

## Defense layers today

The current manifest-tampering defense, committed across several
ADRs:

- **Content addressing (sha256).** Every manifest body and rule
  YAML body is stored at a content-addressed path
  (`manifests/by-hash/sha256-<hex>.json`,
  `yamls/by-hash/sha256-<hex>.yaml`). A modified body has a
  different sha256, and the loader's hash verification fails
  before the body is accepted. Committed by
  [ADR-0005](../adr/0005-manifest-publication-semantics.md) §1
  + §7.
- **CAS pointer write.** The pointer file
  (`manifests/latest.json`) is the only mutable control-plane
  object, and every publish/rollback is exactly one
  generation-conditional write. Concurrent publishers cannot
  silently overwrite each other. Committed by ADR-0005 §3 + §4.
- **Substrate IAM.** Write access to the object-store bucket
  is granted via GCP IAM to the publisher's CI service account
  and the operator rollback role. Committed by the per-env
  overlay's `serviceaccount-annotations.yaml` backed by
  [ADR-0019](../adr/0019-infrastructure-tooling.md).
- **Audit log.** Object-store write events are surfaced by
  the substrate's audit log. Retention duration is operational-
  session policy (currently deployed via the GCP project).
- **CODEOWNERS PR review.** Rule-content changes flow through
  PR review with asymmetric ownership. The manifest publisher
  is invoked only on a merge of a CODEOWNERS-approved change.
  Committed by [ADR-0015](../adr/0015-codeowners.md).

These layers are **unevenly distributed** between prevention
and detection — the loader's accept-or-reject gate is substrate
IAM alone. The other layers either are bypassable by an actor
who has substrate IAM (content addressing, CAS), are detective
rather than preventive (audit log), or are scoped to a
different threat surface (CODEOWNERS gates rule **content**,
not manifest-validity at publish time).

---

## Threat model

The following table is the authoritative reference for which
manifest-tampering threats the v1 posture defends against and
which it explicitly does not.

| Threat | Defended by | Status |
|---|---|---|
| Tampering of manifest body after pointer write | Content addressing (sha256) + loader hash verification | **Defended** by ADR-0005 §"engine reads are two-step" |
| Concurrent publisher race | Generation-conditional CAS on pointer write | **Defended** by ADR-0005 §3-§4 |
| Unauthorized publisher (no CI access) | Substrate IAM on the object-store bucket | **Defended** by per-env overlay's `serviceaccount-annotations.yaml` backed by ADR-0019 |
| Unauthorized rule change | CODEOWNERS PR review with asymmetric ownership | **Defended** by ADR-0015 |
| Compromised CI service account | None (CI has the credentials needed to publish) | **Not defended** — IAM rotation is the response |
| Compromised object-store bucket IAM | None | **Not defended** — IAM rotation is the response |
| Accidental publication of semantically wrong rules | Linter + CODEOWNERS + ADR-0001 byte-equality gate | **Defended** at PR-review time |
| Bit rot of stored manifest body | Sha256 verification at every load | **Defended** by ADR-0005 §"engine reads are two-step" |

The two **Not defended** rows are the threats a signature layer
would address. The v1 posture leaves them at IAM-rotation
defense — operationally, if those credentials are compromised,
the response is to rotate them, identify the breach window via
the audit log, and review any manifests published during the
window.

---

## Trigger conditions for revisiting

The cryptographic-posture decision is **deferred**, not
permanently closed. The following operational signals reopen
[ADR-0030](../adr/0030-manifest-cryptographic-posture.md):

1. **An incident** where a compromised CI service account or
   bucket IAM led to a manifest the loader accepted, AND the
   post-mortem identifies a signature-based control that would
   have prevented acceptance.
2. **Cross-organisation manifest distribution.** If a future
   ADR commits that manifests are consumed outside this
   platform's deployment envelope (e.g., a partner's engine
   binary reads this platform's manifest), the
   signature-for-cross-org-trust use case lands and the
   posture decision reopens automatically.
3. **Regulatory requirement.** If a compliance regime
   applicable to the platform's data domain mandates
   cryptographic signatures on configuration artefacts, the
   posture decision reopens to commit the specific scheme
   that meets the regulation.
4. **Substrate IAM model regression.** If the substrate-
   posture contract from
   [ADR-0010](../adr/0010-substrate-posture.md) +
   [ADR-0017](../adr/0017-substrate-posture-amendment.md)
   weakens (e.g., the CAS guarantee from ADR-0005 §9 is no
   longer enforceable in the deployed substrate), signatures
   and CAS defend against different threats — CAS prevents
   silent overwrite under concurrent publication; signatures
   detect tampering by an actor with substrate write access —
   so the signature layer becomes the natural compensating
   control alongside whatever fix the substrate regression
   itself requires.

A request to add signatures **outside** these four conditions
must be argued from new evidence. The deferral is not
infinitely extensible without an explicit re-argument; the
posture decision is auditable.

---

## Implementation path if signatures are needed

When ADR-0030 reopens, the committed implementation path is a
**detached signature at the pointer level**. An alternative
path (wrapping the manifest body in a signed envelope) was
considered and rejected — it churns ADR-0005's content-
addressing layout for a modest defense increment, so the
pointer-level approach is the first move.

### Pointer-level signature shape

The pointer file (`manifests/latest.json`) gains a `signature`
field of shape `{ key_id, alg, value }`:

```jsonc
// Illustration — pointer file with detached signature
{
  "pointer_version": 1,
  "manifest_hash": "sha256:<hex>",
  "ruleset_version": "rules-v0.1.0",
  "published_at": "...",
  "signature": {
    "key_id": "<identifier>",
    "alg": "<signing-algorithm>",
    "value": "<base64-encoded-signature>"
  }
}
```

### Publisher behavior

The publisher signs the pointer body (including the
`manifest_hash` field) at write time. The signed bytes are the
same bytes the loader will verify. Signing key storage and
algorithm choice are reopen-time decisions — the platform does
not pre-commit a specific cryptographic primitive at this
deferral.

### Loader behavior

The loader fetches the public key at boot from a configurable
location — a new `EnvConfig.ManifestVerifyKeyPath` field
following the typed-env-config pattern from
[ADR-0018](../adr/0018-environment-configuration-model.md)
(PAT-4). The loader's existing `fetchAndVerify` sequence gains
a signature-verification step **immediately after** hash
verification:

```
1. Read pointer file.
2. Verify manifest body's sha256 matches pointer's manifest_hash. ← existing
3. Verify pointer signature against paired public key.          ← new
4. Run ADR-0001 contract checks.
5. Return parsed Manifest.
```

Signature-verification failure raises an operational alert per
[ADR-0006](../adr/0006-alert-routing-contract.md) and refuses
the swap per
[ADR-0007](../adr/0007-loader-scheduler-retry-failure-semantics.md)
CC2 — same refuse-swap posture the loader uses for hash
mismatches today.

### Key rotation

The loader supports multiple paired public keys keyed by
`key_id`. A rotation window allows the publisher to switch
signing keys without a loader restart: the publisher signs
with the new key under a new `key_id`; the loader holds both
the old and the new public key during the window; the old key
is retired once all manifests signed under it are out of the
loader's read horizon.

---

## Cross-references

- [ADR-0030](../adr/0030-manifest-cryptographic-posture.md) —
  the authoritative deferral commitment.
- [ADR-0005](../adr/0005-manifest-publication-semantics.md) —
  the publication primitive (sha256 + CAS).
- [ADR-0010](../adr/0010-substrate-posture.md) +
  [ADR-0017](../adr/0017-substrate-posture-amendment.md) —
  the substrate-posture contract the threat model rests on.
- [ADR-0007](../adr/0007-loader-scheduler-retry-failure-semantics.md)
  §2 — the loader's refuse-swap posture, which a future
  signature-verification step would extend.
- [`docs/runbooks/manifest-rollback.md`](../runbooks/manifest-rollback.md)
  — operator-facing playbook for rolling back the live
  manifest pointer. Refer here when the manifest-tampering
  threat model becomes operationally relevant during a
  rollback procedure.
