<!-- path: docs/adr/0030-manifest-cryptographic-posture.md -->

# ADR-0030 — Manifest Cryptographic Posture

- **Status:** accepted
- **Date:** 2026-05-25

---

## Context

The manifest is the artefact the engine consumes at runtime to know
which rules to evaluate, at which schema version, and under which
engine-compatibility constraint
([ADR-0001](./0001-engine-rules-compatibility.md)).
[ADR-0005](./0005-manifest-publication-semantics.md) committed the
publication primitive that makes manifest reads safe:

- sha256 content-addressing throughout
  (`manifests/by-hash/sha256-<hex>.json`,
  `yamls/by-hash/sha256-<hex>.yaml`);
- a single mutable pointer file (`manifests/latest.json`)
  protected by a generation-conditional CAS write;
- immutable `by-hash/` paths;
- a verify-write-CAS publish sequence.

[ADR-0010](./0010-substrate-posture.md) +
[ADR-0017](./0017-substrate-posture-amendment.md) committed the
substrate-posture contract the publication primitive relies on
(object-store CAS at the substrate level — Partial under the local
emulator per ADR-0017, Yes in the deployed substrate).
[ADR-0008](./0008-git-host.md) committed the CI host that runs the
publisher, with the unforgeable-linter-pin contract committed by
ADR-0001. [ADR-0019](./0019-infrastructure-tooling.md) +
the W3-P7c per-env overlays bind the publisher's CI service account
to the object-store bucket via GCP IAM.

The current defense layers against manifest tampering are:

- **Content addressing.** A modified manifest body has a different
  sha256, and the loader's hash verification fails before the body
  is accepted (ADR-0005 §"engine reads are two-step"; the loader
  in `engine/internal/loader/loader.go` enforces this).
- **CAS pointer write.** A concurrent publisher race surfaces a
  precondition-failed error rather than a silent overwrite
  (ADR-0005 §3-§4).
- **Substrate IAM.** Write access to the object-store bucket is
  granted via GCP IAM to the CI service account and the operator
  rollback role via the per-env overlay's
  `serviceaccount-annotations.yaml`.
- **Audit log.** Object-store write events are surfaced by the
  substrate's audit log (an environment commodity; retention
  duration is operational-session policy).
- **CODEOWNERS gate.** Rule changes flow through PR review with
  asymmetric ownership per
  [ADR-0015](./0015-codeowners.md); the manifest publisher is
  invoked only on a merge of a CODEOWNERS-approved change.

These layers are unevenly distributed between prevention and
detection. The loader's accept-or-reject gate is substrate IAM:
an attacker who compromises the CI service account's credentials
can publish a manifest the loader accepts (one preventive layer
broken). Content addressing is trivially recomputed by the
attacker; CAS is bypassed by the same IAM access. The remaining
layers are detective (audit log surfaces the breach) or scoped to
a different threat surface (CODEOWNERS gates rule **content**
changes via PR review, not manifest-validity at publish time).

This ADR resolves whether to add a cryptographic layer that
defends specifically against the two undefended threats — a
compromised CI service account or a compromised object-store IAM
— by introducing a signing/verification mechanism beyond the
existing sha256 checksums. The principles bearing on the decision
are **P4** (cost is a first-class constraint — key management has
real operational cost; under-justified signatures are platform
baggage), **P6** (borrow patterns, not baggage — adopt a
cryptographic posture only if the threat justifies it in this
platform's context), and **R3** (do not revisit settled
architectural decisions — ADR-0005's posture is settled; this
row must not reopen it without strong cause).

---

## Decision

### Defer signatures; keep the status-quo posture

The current posture (sha256 + CAS + substrate IAM + audit log +
CODEOWNERS PR review) is sufficient for the realistic threat
surface this platform faces. No signature layer is added at v1.
ADR-0005 is unchanged: §4 (verify-write-CAS sequence), §5
(manifest body field set), §6 (pointer file field set), and the
loader's `fetchAndVerify` sequence retain their current
commitments. The B0-5 reopener from the row that prompted this
ADR does not fire.

The deferral rests on three observations:

- **The threat surface is internal.** The DQ Platform is an
  internal data-quality service. Manifests are consumed only by
  the engine binary inside the platform's deployment envelope;
  they are not externally distributed, not consumed by external
  partners, not signed for cross-org trust. The realistic threat
  model is "an insider with partial credentials publishes a bad
  manifest", not "a remote attacker forges a signed artefact."
- **Key management has a recurring cost.** A signing key needs a
  generation procedure, storage (hardware-bound, key-management
  service, or env-var-bound; each has trade-offs), rotation
  cadence, revocation procedure, distribution to the loader, and
  a compromised-key incident response. The marginal defense
  added by signatures is one additional preventive layer; the
  operational cost is non-trivial.
- **IAM rotation is the v1 response.** The two undefended
  threats (compromised CI service account; compromised bucket
  IAM) are addressed operationally by rotating the affected
  credentials. Detection lives in the audit log. The deferral
  accepts this response posture explicitly rather than implicitly.

### Threat model

The threat model below is the authoritative reference for which
threats this ADR covers and which it explicitly does not:

| Threat | Defended by | Status |
|---|---|---|
| Tampering of manifest body after pointer write | Content addressing (sha256) + loader hash verification | **Defended** by ADR-0005 §"engine reads are two-step" |
| Concurrent publisher race | Generation-conditional CAS on pointer write | **Defended** by ADR-0005 §3-§4 |
| Unauthorized publisher (no CI access) | Substrate IAM on the object-store bucket | **Defended** by the per-env overlay's `serviceaccount-annotations.yaml` backed by ADR-0019 |
| Unauthorized rule change | CODEOWNERS PR review with asymmetric ownership | **Defended** by ADR-0015 |
| Compromised CI service account | None (CI has the credentials needed to publish) | **Not defended** — IAM rotation is the response |
| Compromised object-store bucket IAM | None | **Not defended** — IAM rotation is the response |
| Accidental publication of semantically wrong rules | Linter + CODEOWNERS + ADR-0001 byte-equality gate | **Defended** at PR-review time |
| Bit rot of stored manifest body | Sha256 verification at every load | **Defended** by ADR-0005 §"engine reads are two-step" |

The two "Not defended" rows are the threats signatures would
address. This ADR commits to leaving them at IAM-rotation
defense at v1.

### Trigger conditions for revisiting

The following operational signals reopen this ADR. A request to
add signatures **outside** these four conditions must be argued
from new evidence; the deferral is not infinitely extensible
without an explicit re-argument.

1. **An incident** where a compromised CI or bucket IAM led to a
   published manifest the loader accepted, AND the post-mortem
   identifies a signature-based control that would have prevented
   acceptance.
2. **Cross-organisation manifest distribution.** If a future ADR
   commits that manifests are consumed outside this platform's
   deployment envelope (e.g., a partner's engine binary reads
   this platform's manifest), the signature-for-cross-org-trust
   use case lands and this ADR reopens automatically.
3. **Regulatory requirement.** If a compliance regime applicable
   to the platform's data domain mandates cryptographic
   signatures on configuration artefacts, this ADR reopens to
   commit the specific scheme that meets the regulation.
4. **Substrate IAM model regression.** If the substrate-posture
   contract from ADR-0010 + ADR-0017 weakens (e.g., the CAS
   guarantee from ADR-0005 §9 is no longer enforceable in the
   deployed substrate), signatures and CAS defend against
   different threats — CAS prevents silent overwrite under
   concurrent publication; signatures detect tampering by an
   actor with substrate write access — so the signature layer
   becomes the natural compensating control alongside whatever
   fix the substrate regression itself requires, and this ADR
   reopens.

### Implementation path if revisited

When this ADR reopens under one of the four conditions above,
the implementation path is a **detached signature at the pointer
level**. The path is documented here so a future implementer
does not re-evaluate from scratch:

- The pointer file gains a `signature` field of shape
  `{ key_id, alg, value }`:

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

- The publisher signs the pointer body (including the
  `manifest_hash` field) at write time; the signed bytes are the
  same bytes the loader will verify.
- The loader fetches the public key at boot from a configurable
  location — a new `EnvConfig.ManifestVerifyKeyPath` field
  following the typed-env-config pattern from
  [ADR-0018](./0018-environment-configuration-model.md) (PAT-4).
- The loader's existing `fetchAndVerify` sequence gains a
  signature-verification step immediately after hash
  verification; verification failure raises an operational alert
  per [ADR-0006](./0006-alert-routing-contract.md) and refuses
  the swap per
  [ADR-0007](./0007-loader-scheduler-retry-failure-semantics.md)
  CC2.
- Key rotation: the loader supports multiple paired public keys
  keyed by `key_id`; a rotation window allows the publisher to
  switch signing keys without a loader restart.

An alternative path — wrapping the manifest body in a signed
envelope format — was considered and rejected for the eventual
reopen: it churns ADR-0005's content-addressing layout (the
`by-hash/` path would key on the envelope hash, changing
rollback ergonomics) for a modest defense increment over the
pointer-level approach. If stronger defense is needed at reopen
time, the pointer-level approach is followed by a stricter
content-addressing amendment, not by adopting an envelope
wholesale.

### Operator-facing security note

A separate security note ships at
`docs/security/manifest-cryptographic-posture.md` carrying the
threat-model table, the trigger conditions, and the
implementation path in operator-readable prose. The
`docs/security/` directory is new and is added alongside the
existing `docs/adr/`, `docs/runbooks/`, etc.

---

## Consequences

1. **No engine code change ships from this ADR.** The loader,
   publisher, evaluator, runner, and trigger handler are
   unchanged by this resolution.

2. **A new security note lands at
   `docs/security/manifest-cryptographic-posture.md`** carrying
   the threat-model table, the trigger conditions for
   revisiting, and the pointer-level implementation path. The
   `docs/security/` directory is the first entry; future
   security artefacts (incident-response playbooks, threat
   models for other surfaces) ship under the same directory
   following the same forward-only-prose pattern.

3. **`docs/README.md` is updated to advertise the new
   `docs/security/` directory** alongside the existing
   `docs/adr/`, `docs/runbooks/`, etc.

4. **ADR-0005 is unchanged.** The B0-5 reopener does not fire;
   §4 (verify-write-CAS sequence), §5 (manifest body field
   set), §6 (pointer file field set), and the loader's
   `fetchAndVerify` sequence retain their current commitments.

5. **No new B2 rows register from this ADR.** The
   implementation path for signature adoption is documented as
   prose; a B2 row is added only when one of the four trigger
   conditions fires and signatures actually need to be built.

6. **An operator-facing cross-reference is added to the
   existing manifest-rollback runbook** (or its successor)
   pointing readers to the security note for the cryptographic
   posture context. Rollback runbooks already cite ADR-0005;
   the additional cross-reference reduces the chance a future
   reader assumes signatures exist when they do not.

7. **The trigger conditions are auditable.** Future "should we
   add signatures?" requests are evaluated against the four
   conditions; the security note is the reference document.
   Requests that do not meet a condition are deferred under
   the existing posture; requests that do meet a condition
   reopen this ADR under the standard study → critique →
   promotion protocol.

8. **The two `Not defended` threats are documented, not
   silently accepted.** A future incident review can trace
   the platform's awareness of the gap and the explicit
   IAM-rotation response posture, instead of discovering the
   gap after the fact.

9. **Rule YAML bodies do not gain standalone signatures.**
   Rule YAMLs are content-addressed via the manifest's
   `yaml_hash` per ADR-0005, so a tampered YAML is caught by
   the manifest's hash chain. A standalone YAML signature
   would only defend against a rule YAML uploaded outside the
   manifest publication path; no scenario in the current
   threat surface justifies it. The deferral applies uniformly
   to manifests and to rule bodies.

10. **The platform's P4 + P6 commitments for cryptographic
    posture are now explicit.** This ADR is the record of why
    the platform does **not** ship signatures at v1, and
    under what conditions that decision is revisited. The
    deferral is auditable, not vague.

---

## Notes

- The 1 GB / 100 GB / 1 TB and similar quantitative ceilings
  carried by other cost-discipline ADRs do not apply here; this
  ADR is a posture commitment, not a per-env defaults policy.
- Specific algorithm choice and key-storage substrate for the
  deferred pointer-level signature path are reserved for the
  reopen-time decision. The current ADR commits the shape of
  the signature field but not the cryptographic primitive — at
  reopen time, the choice between asymmetric vs symmetric
  signing, the specific curve or key length, and the
  key-storage substrate (hardware-bound, key-management
  service, env-var-bound) is made against the threat surface
  the reopener identified.
- The substrate's audit-log retention duration is a separate
  concern not committed by this ADR. The threat model assumes
  the substrate retains object-store write events for long
  enough to attribute a malicious publication; the retention
  duration is operational-session policy deployed via the GCP
  project. A future row may register an audit-log retention
  decision if concrete operational signal (an incident, an
  audit-finding, or a retention-cost surprise) justifies one.
