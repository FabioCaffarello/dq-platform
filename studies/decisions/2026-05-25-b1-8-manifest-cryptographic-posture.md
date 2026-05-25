<!-- path: studies/decisions/2026-05-25-b1-8-manifest-cryptographic-posture.md -->

# B1-8 — Manifest Cryptographic Posture

## Context

The manifest is the artefact the engine consumes at runtime to know
which rules to evaluate, at which schema version, and under which
engine-compatibility constraint (ADR-0001). ADR-0005 committed the
publication primitive that makes manifest reads safe: sha256
content-addressing throughout, a single mutable pointer file
(`manifests/latest.json`) protected by a generation-conditional CAS
write, immutable `by-hash/` paths, and a verify-write-CAS sequence.
ADR-0010 + ADR-0017 committed the substrate-posture contract that
the publication primitive relies on (object-store CAS at the
substrate level — Partial under the local emulator per ADR-0017, Yes
in the deployed substrate). ADR-0008 committed the CI host that
runs the publisher, with the unforgeable-linter-pin contract
committed by ADR-0001.

The current defense layers against manifest tampering:

- **Content addressing.** A modified manifest body has a different
  sha256, and the loader's hash verification fails before the body
  is accepted (ADR-0005 §"engine reads are two-step"; the loader
  in `engine/internal/loader/loader.go` enforces this).
- **CAS pointer write.** A concurrent publisher race surfaces a
  precondition-failed error rather than a silent overwrite
  (ADR-0005 §3-§4).
- **Substrate IAM.** Write access to the object-store bucket is
  granted via GCP IAM to the CI service account and the operator
  rollback role per ADR-0008 + the per-env overlay's
  `serviceaccount-annotations.yaml` (W3-P7c).
- **Audit log.** Object-store write events are surfaced by the
  substrate's audit log (out of scope for this study; an
  environment commodity).
- **CODEOWNERS gate.** Rule changes flow through PR review with
  asymmetric ownership per ADR-0015; the manifest publisher is
  invoked only on a merge of a CODEOWNERS-approved change.

What the current posture does **not** defend against:

- An attacker who has compromised the CI service account's
  credentials AND the CAS write surface AND the audit log retention.
  Such an attacker can publish a manifest the loader will accept.
- An attacker who has compromised the object-store bucket's IAM
  independently of CI (e.g., an over-broad GCS admin role granted
  to a separate service account). The CAS write is gated by the
  bucket, not by CI alone; bucket-level IAM compromise lets the
  attacker write a valid pointer with a valid hash.
- Accidental publication of a manifest that passes the three
  pre-publish verifications from ADR-0001 §C2 but represents
  semantically wrong rules (e.g., a rule with a typo that lints
  clean and publishes successfully). This threat is handled by
  the linter, the CODEOWNERS PR review, and the byte-equality CI
  gate; cryptographic signatures would not help.

B1-8's question — "does the manifest carry signatures beyond
checksums?" — is the question of whether to add a cryptographic
layer that defends specifically against the **first two threats**:
a compromised CI service account or a compromised object-store IAM.
A signed manifest would require the attacker to also possess a
signing key the loader has paired against a public key, raising the
bar from "two keys (CI + bucket)" to "three keys (CI + bucket +
signing)".

The principles bearing on the decision are P4 (Cost is a
first-class constraint — key management has operational cost;
under-justified signatures are platform baggage), P6 (Borrow
patterns, not baggage — adopt a cryptographic posture only if
the threat justifies it in our context), and R3 (Do not revisit
settled architectural decisions — ADR-0005's posture is settled;
B1-8 must not reopen it without strong cause). The B1-8 row in
the decision log explicitly flags this study as a potential B0-5
reopener; the study must resolve whether the reopener actually
fires.

---

## Decision Drivers

- **DD-1 — Threat surface is internal.** The DQ Platform is an
  internal data-quality service. Manifests are consumed only by
  the engine binary inside the platform's deployment envelope;
  they are not externally distributed, not consumed by external
  partners, not signed for cross-org trust. The realistic threat
  model is "an insider with partial credentials publishes a bad
  manifest", not "a remote attacker forges a signed artefact."
- **DD-2 — Existing defense layers are non-trivial but
  unevenly distributed across prevention and detection.** The
  loader's accept-or-reject gate is substrate IAM: an attacker
  who compromises the CI service account's credentials can
  publish a manifest the loader accepts (one preventive layer
  broken). Content addressing is trivially recomputed by the
  attacker; CAS is bypassed by the same IAM access. The
  remaining layers are detective (audit log surfaces the
  breach) or scoped to a different threat (CODEOWNERS PR
  review gates rule **content** changes but never enters the
  publisher's bypass-or-replace path). Signatures would add
  one *additional preventive* layer the attacker would also
  need to compromise — the marginal defense at a non-trivial
  operational cost (key management).
- **DD-3 — Key management has a recurring cost.** A signing key
  needs a generation procedure, storage (HSM-bound, KMS, or env-
  var-bound; each has trade-offs), rotation cadence, revocation
  procedure, distribution to the loader, and a compromised-key
  incident response. None of these is free; all of them add
  surface area the platform's operational model has to absorb
  alongside the existing IAM rotation surface.
- **DD-4 — B0-5 reopener risk.** If B1-8 commits signatures,
  ADR-0005's pointer file and manifest body field sets gain a
  `signature` field, and the loader's verify sequence gains a
  signature-verification step. Both are non-trivial amendments
  to a settled ADR. R3 raises the bar for the commitment that
  justifies the reopener.
- **DD-5 — Threshold for revisiting must be explicit.** Deferring
  signatures without naming the trigger conditions invites the
  question being reopened on weak grounds. The study must commit
  the specific operational signals that would justify revisiting
  (incident, regulatory requirement, scope expansion).

---

## Considered Options

### Option 1 — Defer signatures; keep status-quo posture (recommended)

The current posture (sha256 + CAS + substrate IAM + audit log +
CODEOWNERS PR review) is sufficient for the realistic threat
surface. B1-8 ships as a **security note** documenting:

- the current defense layers,
- the threats they cover and the threats they explicitly do not,
- the trigger conditions for revisiting,
- the implementation path if/when revisited.

ADR-0005 is unchanged. The loader is unchanged. The publisher is
unchanged. The B1-8 row in the decision log moves to
`resolved-adr` with the security note as its artefact.

**Strengths.** Zero operational cost; preserves ADR-0005 without
reopening; concentrates security investment on the layers that
actually carry the load (IAM, audit, CODEOWNERS); makes the
deferral honest by naming what we are not defending against.

**Trade-offs.** A future incident that exposes a gap signatures
would have closed will land in this study's trigger-condition
list as evidence that the deferral was wrong. The study commits
the trigger conditions in advance to keep the post-incident
question disciplined.

### Option 2 — Detached signature at the pointer level

The manifest publisher signs the pointer file's `manifest_hash`
field with a key the loader has paired against a public key. The
pointer file gains a `signature` field; the loader's verify
sequence gains a signature-verification step.

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

**Strengths.** Minimal change to ADR-0005's body format (the
manifest body itself is unchanged); the signature gates the
pointer-write surface specifically; loader-side verification is
a single additional step.

**Trade-offs.** Reopens ADR-0005's §6 pointer field set;
introduces a key the loader has to load + rotate; the loader
gains a failure mode (verification fails on a key-rotation
window mismatch); operationally this lands a key management
surface where there was none. The defense increment is real but
the cost is non-trivial.

### Option 3 — Signed envelope wrapping the manifest body

The manifest body is wrapped in a signed envelope format that
pairs the inner payload with a signed header + signature
triplet. The envelope's `by-hash/` path keys on the envelope's
hash, not on the inner manifest's hash; the loader unwraps and
verifies the signature before extracting the inner manifest.

**Strengths.** Defends against tampering of either the pointer
or the body (the body's signature binds the inner content);
stronger than Option 2.

**Trade-offs.** Reopens both §5 (manifest body field set) and §6
(pointer file field set); the loader gains envelope parsing as a
new dependency; the manifest body's `by-hash/` content
addressing now keys on the envelope hash, which means rollback
ergonomics change (an operator rolling back to a prior manifest
hash needs to know the envelope hash, not the body hash); this
churn compounds against the modest defense increment over
Option 2.

---

## Recommendation

**Option 1.** Defer signatures; ship a security note documenting
the current posture, the threat model, the trigger conditions for
revisiting, and the implementation path if needed.

### Threat model committed to the security note

The security note (filename: `docs/security/manifest-cryptographic-posture.md`)
commits the following layered threat model:

| Threat | Defended by | Status |
|---|---|---|
| Tampering of manifest body after pointer write | Content addressing (sha256) + loader hash verification | **Defended** by ADR-0005 §"engine reads are two-step" |
| Concurrent publisher race | Generation-conditional CAS on pointer write | **Defended** by ADR-0005 §3-§4 |
| Unauthorized publisher (no CI access) | Substrate IAM on the object-store bucket | **Defended** by the per-env overlay's `serviceaccount-annotations.yaml` (W3-P7c) backed by ADR-0019 |
| Unauthorized rule change | CODEOWNERS PR review with asymmetric ownership | **Defended** by ADR-0015 |
| Compromised CI service account | None (CI has the credentials needed to publish) | **Not defended** — IAM rotation is the response |
| Compromised object-store bucket IAM | None | **Not defended** — IAM rotation is the response |
| Accidental publication of semantically wrong rules | Linter + CODEOWNERS + ADR-0001 byte-equality gate | **Defended** at PR-review time |
| Bit rot of stored manifest body | Sha256 verification at every load | **Defended** by ADR-0005 §"engine reads are two-step" |

The two "Not defended" rows are the threats signatures would
address. The recommendation is to leave them at IAM-rotation
defense at v1.

### Trigger conditions for revisiting

The security note commits the following specific operational
signals that would justify revisiting B1-8:

1. **An incident** where a compromised CI or bucket IAM led to a
   published manifest the loader accepted, AND the post-mortem
   identifies a signature-based control that would have prevented
   acceptance.
2. **Cross-organisation manifest distribution.** If a future ADR
   commits that manifests are consumed outside this platform's
   deployment envelope (e.g., a partner's engine binary reads our
   manifest), the signature-for-cross-org-trust use case lands
   and B1-8 reopens automatically.
3. **Regulatory requirement.** If a compliance regime applicable
   to the platform's data domain mandates cryptographic signatures
   on configuration artefacts, B1-8 reopens to commit the
   specific scheme that meets the regulation.
4. **Substrate IAM model regression.** If the substrate-posture
   contract from ADR-0010 + ADR-0017 weakens (e.g., the CAS
   guarantee from §9 is no longer enforceable in the deployed
   substrate), signatures and CAS defend against different
   threats — CAS prevents silent overwrite under concurrent
   publication; signatures detect tampering by an actor with
   substrate write access — so the signature layer becomes the
   natural compensating control alongside whatever fix the
   substrate regression itself requires, and B1-8 reopens.

A request to add signatures **outside** these four conditions
must be argued from new evidence; the deferral is not infinitely
extensible without an explicit re-argument.

### Implementation path if revisited

When B1-8 reopens under one of the four conditions above, the
implementation path is **Option 2** (detached signature at the
pointer level). The security note documents the path so a future
implementer does not re-evaluate from scratch:

- Pointer gains a `signature: { key_id, alg, value }` field.
- The publisher signs the pointer body (including the
  `manifest_hash` field) at write time; the signed bytes are the
  same bytes the loader will verify.
- The loader fetches the public key at boot from a configurable
  location (a new `EnvConfig.ManifestVerifyKeyPath` field; same
  PAT-4 pattern as the rest of EnvConfig per ADR-0018).
- The loader's existing `fetchAndVerify` sequence gains a
  signature-verification step immediately after hash
  verification; verification failure raises an operational alert
  per ADR-0006 and refuses the swap per ADR-0007 CC2.
- Key rotation: the loader supports multiple paired public keys
  by `key_id`; a rotation window allows the publisher to switch
  signing keys without a loader restart.

Option 3 (signed envelope) is rejected as the implementation
path because it churns ADR-0005's content-addressing layout for
a modest defense increment over Option 2. If a stronger defense
is needed, Option 2 is followed by a stricter content-addressing
amendment, not by adopting an envelope wholesale.

### Why B1-8 does NOT reopen ADR-0005

The B1-8 row was flagged as a potential B0-5 reopener. With this
recommendation:

- ADR-0005's §5 manifest body field set is unchanged.
- ADR-0005's §6 pointer file field set is unchanged.
- ADR-0005's §4 verify-write-CAS sequence is unchanged.
- The loader's `fetchAndVerify` sequence is unchanged.

The reopener does not fire. ADR-0005 status remains `accepted`
without amendment. The B1-8 commitment is the security note
itself, not a change to ADR-0005.

---

## Consequences

1. **A new security note ships at
   `docs/security/manifest-cryptographic-posture.md`** carrying
   the threat model table, the trigger conditions for
   revisiting, and the implementation path if revisited. The
   `docs/security/` directory is new; the note is the first
   entry.

2. **`docs/README.md` is updated to advertise the new
   `docs/security/` directory** alongside the existing
   `docs/adr/`, `docs/runbooks/`, etc. Same forward-only
   pattern other directory additions used.

3. **ADR-0005 is unchanged.** The B1-8 reopener does not fire;
   §4, §5, §6, and the loader's verify sequence retain their
   current commitments.

4. **The decision log B1-8 row moves to `resolved-adr`** with
   the security note as the linked artefact. The "flagged as a
   potential B0-5 reopener if strengthened" note is replaced
   with a pointer to this study's trigger-condition list.

5. **The post-Wave-3 follow-up backlog loses B1-8 from its
   "Open B1 rows" list.** The reopener-conditional posture
   moves to the security note; the backlog's remaining open
   rows are B1-1, B1-3, B1-5, B1-6, B1-7 (one fewer than the
   pre-resolution count).

6. **No engine code change ships.** The loader, publisher,
   evaluator, runner, and trigger handler are unchanged by this
   resolution.

7. **No new B2 rows register.** The implementation path for
   signature adoption is documented in the security note as
   prose; a B2 row is added only when one of the four trigger
   conditions fires.

8. **The trigger conditions are auditable.** Future "should we
   add signatures?" requests are evaluated against the four
   conditions; the security note is the reference document.
   Requests that do not meet a condition are deferred under
   the existing posture; requests that do meet a condition
   reopen B1-8 under the standard study → critique → promotion
   protocol.

9. **An operator-facing note is added to the existing
   `docs/runbooks/manifest-rollback.md`** (or its successor)
   pointing readers to the security note for the cryptographic
   posture context. Rollback runbooks already cite ADR-0005;
   the additional cross-reference reduces the chance a future
   reader assumes signatures exist when they do not.

10. **The platform's P6 commitment is honored:** the security
    note adopts the posture that fits our context rather than
    borrowing a heavier cryptographic envelope from external
    designs. The deferred Option 2 path is described in our
    own terms.

---

## Open Questions

None blocking.

Three deferred items surfaced during drafting are explicitly
**out-of-scope for current cycle**:

- **OQ-1: Specific signing algorithm and key-storage substrate
  for Option 2.** If B1-8 reopens, the algorithm choice
  (asymmetric vs symmetric; specific curve / key length) and the
  key-storage substrate (HSM, KMS, env-var-bound) are concrete
  decisions to make at reopen time. The current security note
  commits the shape of the pointer-level signature field but not
  the cryptographic primitive. Deferred until the reopener fires.

- **OQ-2: Whether the rule YAML bodies themselves need
  signatures.** This study's threat model covers the manifest;
  rule YAMLs are content-addressed via the manifest's
  `yaml_hash` per ADR-0005, so a tampered YAML is caught by
  the manifest's hash chain. A standalone YAML signature would
  defend against a rule YAML uploaded outside the manifest
  publication path. Deferred: no scenario in the current
  threat surface justifies it.

- **OQ-3: Whether the audit-log retention duration warrants
  a separate study.** The threat model assumes the
  substrate's audit log retains object-store write events for
  long enough to attribute a malicious publication; the
  retention duration is a substrate-policy concern (deployed
  via the operational session's GCP project). Deferred
  indefinitely; no B2 row is registered until concrete
  operational signal — an incident, an audit-finding, or a
  retention-cost surprise — justifies one.

---

## Promotion target

`docs/adr/0030-manifest-cryptographic-posture.md` — promotes the
deferral decision + the threat-model table + the four trigger
conditions + the Option 2 implementation path into the
authoritative ADR. The security note at
`docs/security/manifest-cryptographic-posture.md` ships
alongside as the operator-facing artefact (forward-only prose;
no back-link into `studies/`).
