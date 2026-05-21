<!-- path: studies/decisions/2026-05-21-b1-11-substrate-posture-amendment.md -->

# B1-11 — Substrate Posture Amendment: Object-Store CAS Row

## Metadata

- **B1 reference:** B1-11 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md).
- **Status:** resolved-study (single critique round; the amendment
  is grounded in Phase 2 emulator-evaluation evidence).
- **Last updated:** 2026-05-21
- **Upstream resolved:** W2-3 / ADR-0010 substrate posture
  ([study](./2026-05-21-platform-decisions-wave2.md);
  promoted to
  [`docs/adr/0010-substrate-posture.md`](../../docs/adr/0010-substrate-posture.md)).
  B0-5 manifest publication semantics
  ([promoted](../../docs/adr/0005-manifest-publication-semantics.md))
  uses the row this study amends.
- **Downstream:** Wave 3 Phase 3 onward — any phase whose
  local smoke or integration tests assume CAS enforcement
  must read this study (and its eventual amendment ADR) for
  the actual fidelity posture.
- **Promotion target:** `docs/adr/0015-substrate-posture-amendment.md`
  (an amendment ADR that supersedes the specific
  capability-matrix row identified below; ADR-0014 is
  reserved for B1-10's promotion).

---

## Context

ADR-0010 (substrate posture) commits a capability matrix
classifying each substrate row as **Yes** (locally
emulatable), **No** (sandbox required), or **Partial**
(locally partial; sandbox for full fidelity). One row
reads:

> Object store: generation-conditional pointer write — **Yes**
> — B0-5 CC3 compare-and-swap on `manifests/latest.json`
> must be testable locally.

Wave 3 Phase 2 (root infrastructure) selected a commodity
emulator for the object-store row and built a smoke test
that exercises the ADR-0005 publication primitive end-to-
end. The smoke verified that:

- bucket creation, object write, and round-trip read all
  work locally;
- the content-addressed `by-hash/sha256-<hex>` layout
  required by ADR-0005 §1 works locally;
- the `ifGenerationMatch` query parameter is accepted by
  the emulator without an error.

The smoke also discovered that **the emulator does not
enforce** the `ifGenerationMatch` precondition: a write
with a stale `ifGenerationMatch` value returns HTTP 200,
not the HTTP 412 Precondition Failed that ADR-0005 §4
("verify-then-write-content-then-CAS-pointer") relies on.

Three commodity emulators were evaluated during Phase 2:

| Emulator | CAS enforcement | Why rejected |
|---|---|---|
| `fsouza/fake-gcs-server:latest` | accepts stale-generation writes silently | covers everything else; only fails on CAS — **selected with documented gap** |
| `gcr.io/cloud-storage-image/storage-testbench` | enforces preconditions | requires Google Artifact Registry auth not available to commodity contributor workflows |
| `oittaa/gcp-storage-emulator:latest` | enforces preconditions | does not implement the `/upload/storage/v1/b/<bucket>/o?uploadType=media` endpoint that ADR-0005's publication primitive uses |

No commodity emulator examined satisfies both
"locally-runnable without auth gating" **and** "enforces
`ifGenerationMatch`". The ADR-0010 row as written cannot
be satisfied by any single commodity image in 2026-05.

This study revises the row to match reality and locks the
amendment as a tracked decision so downstream phases do
not silently build on an aspirational commitment.

---

## Decision Drivers

1. **D1. Honest substrate posture.** A capability matrix
   that overstates local fidelity trains contributors to
   write code that "works locally, breaks in production".
   The matrix must match what commodity emulators actually
   deliver.

2. **D2. Inherit substrate evaluation evidence across
   sessions.** Future substrate decisions should inherit
   Phase 2's emulator-evaluation findings rather than
   re-derive them; the amendment ADR carries the
   architectural rationale forward as a forward-only
   record.

3. **D3. Preserve the ADR-0005 publication contract.** The
   `verify-then-write-content-then-CAS-pointer` discipline
   is load-bearing in production — concurrent publishers
   must fail loudly, no silent overwrite. Whatever the
   local emulator does, the production substrate must
   enforce CAS.

4. **D4. Match the existing fidelity-gap pattern.** ADR-0010
   already documents one Partial row (the tabular-store
   lazy-view fidelity gap). The same pattern applies here:
   local emulator covers the API surface; full-fidelity
   validation runs in the sandbox-required CI lane.

5. **D5. Cost-as-first-class (P4).** Forcing every
   contributor to obtain sandbox cloud access for routine
   CAS-touching work would be unacceptable. The amendment
   keeps Phase 2's "routine flows run locally" property
   intact for the surfaces commodity emulators do support.

---

## Considered Options

- **(A) Build a precondition-enforcing proxy** in front of
  fake-gcs-server. Restores `Yes` to the row at the cost
  of a small custom Go proxy binary plus ongoing
  maintenance. Rejected: custom infrastructure to
  re-implement a feature that the production substrate
  provides is the wrong cost / risk trade. The same
  contributor cost (sandbox access for full-fidelity CAS
  validation) lives on the engine team forever.

- **(B) Switch to a paid emulator** (Google's storage-
  testbench via Artifact Registry, or an equivalent). Restores
  `Yes` at the cost of every contributor needing
  authenticated access to a Google Cloud project. Rejected:
  blocks D5 (cost-as-first-class) and contradicts ADR-0010's
  capability-matrix framing where `Yes` means
  "exercisable by a local test or runnable command".

- **(C) Amend the row to `Partial` with the matching
  fidelity-gap note**, parallel to the tabular-store lazy-
  view row. The selected commodity emulator covers the API
  surface; production-shape CAS enforcement runs in the
  sandbox-required CI lane. **Recommended.**

---

## Recommendation

Adopt **(C)** — amend ADR-0010 to revise the
"Object store: generation-conditional pointer write" row
from `Yes` to `Partial`, with the following row text
proposed for the amendment ADR:

> Object store: generation-conditional pointer write —
> **Partial** — `ifGenerationMatch` query parameter is
> accepted locally without error; precondition enforcement
> is not faithful in commodity emulators (stale-generation
> writes succeed locally). The ADR-0005 §4 publication
> primitive (`verify-then-write-content-then-CAS-pointer`)
> runs locally for the API-surface portion; full-fidelity
> CAS enforcement runs in the sandbox-required CI lane.

The amendment ADR also updates the substrate-selection
checkpoint paragraph of ADR-0010 to add a CAS-fidelity
sub-criterion; the proposed text and its rationale are in
Consequence 7 below.

The recommendation is grounded in foundation 02 (substrate
posture is a Wave 2 concern), W2-3 (the capability-matrix
framing), and the Phase 2 evidence above. The specific
text changes (the row revision and the substrate-selection
amendment paragraph) are
**new contribution proposed here, requires review**.

---

## Consequences

1. **The substrate posture document and the deployment
   substrate now agree.** ADR-0010's matrix and
   `docker-compose.yml` will both classify the CAS row as
   `Partial` once the amendment ADR lands.

2. **Production-shape CAS enforcement is sandbox-required,
   not Phase-2-blocking.** The ADR-0005 publication primitive
   runs locally for the API surface and for the by-hash
   immutability part; the precondition-enforcement portion
   is validated in the sandbox-required CI lane (same lane
   that already covers the tabular-store lazy-view
   fidelity gap).

3. **Downstream phases inherit the corrected posture.**
   Phase 4 (engine runtime) and Phase 6 (first onboarded
   entity end-to-end) reference ADR-0010 for capability
   coverage. After the amendment ADR lands, those phases
   read the amended row and design their local tests
   accordingly.

4. **The Wave 3 sequencing ADR (ADR-0013) is unaffected.**
   The phase structure does not change; only ADR-0010's
   row is revised.

5. **The amendment is reversible via the substrate-
   selection checkpoint mechanism.** If a faithfully-
   enforcing commodity emulator becomes available without
   auth gating, the row is re-amended back to `Yes` via a
   follow-up ADR, evaluated against the substrate-
   selection checkpoint (Consequence 7 below). The
   amendment ADR carries this as an explicit reversibility
   clause; no separate reversal mechanism is needed.

6. **The B0-5 publication primitive does not change.**
   ADR-0005's `verify-then-write-content-then-CAS-pointer`
   discipline remains; only the locally-testable scope of
   that discipline narrows. The production substrate
   continues to enforce CAS; the local substrate exercises
   what it can.

7. **The substrate-selection checkpoint paragraph in
   ADR-0010 gains a CAS-fidelity sub-criterion.** The
   amendment ADR adds the following text to ADR-0010's
   substrate-selection checkpoint paragraph:

   > Substrates evaluated against this matrix must also be
   > evaluated for `ifGenerationMatch` enforcement
   > fidelity. A substrate that accepts stale-generation
   > writes without returning HTTP 412 is treated as
   > **Partial** for this row, not **Yes**, regardless of
   > API-surface coverage.

   Future substrate evaluations must verify CAS
   enforcement, not just API-surface coverage. **New
   contribution proposed here, requires review.**

---

## Open Questions

- **OQ-B1-11.1.** Whether a small precondition-enforcing
  proxy (Option A above) might still be worth writing later
  as a tooling investment, separately from the amendment.
  **Out-of-scope for current cycle — revisit if and when a
  Wave 3 phase produces enough sandbox-only CI churn that
  the proxy investment becomes worth it.** New contribution
  proposed here, requires review.

- **OQ-B1-11.2.** Whether to track a periodic "re-check
  commodity-emulator CAS support" task (so the row could
  flip back to `Yes` if a faithful emulator emerges).
  **Out-of-scope for current cycle — the substrate-
  selection checkpoint paragraph of the amendment ADR
  naturally surfaces this question whenever a substrate
  swap is proposed.**

- **OQ-B1-11.3.** Whether other capability-matrix rows
  hide similar fidelity gaps that have not yet been
  discovered. **Out-of-scope for current cycle — Phase
  2's emulator evaluation covered the rows Phase 2
  stands up; further fidelity gaps surface as later
  phases exercise their respective capabilities.**

---

## Promotion target

This study is promoted during a future ADR-promotion
session to:

    docs/adr/0015-substrate-posture-amendment.md

The `0015` slot is the next available after ADR-0014
(reserved for B1-10's workspace-tooling promotion). The
slug (`substrate-posture-amendment`) is the stable part;
the number adjusts at promotion time if the ADR numbering
convention shifts.

The amendment ADR **supersedes** the specific
capability-matrix row identified in this study, plus the
substrate-selection checkpoint paragraph addition. The
rest of ADR-0010 stands. The decision-log B1-11 row
flips from `resolved-study` to `resolved-adr` at promotion
time and gains the link to ADR-0015.
