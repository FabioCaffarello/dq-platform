<!-- path: docs/adr/0017-substrate-posture-amendment.md -->

# ADR-0017 — Substrate Posture Amendment: Object-Store CAS Row

- **Status:** accepted (amends ADR-0010)
- **Date:** 2026-05-23

**Scope note (added 2026-05-23):** This ADR currently applies to set-oriented capability (realized over BigQuery). Record-oriented capability is addressed separately in Wave-S (see ADR-0020 forthcoming).

---

## Context

ADR-0010 commits a capability matrix that any compliant substrate
must satisfy and pairs it with a local Docker Compose environment
that must, by the close of Wave 3 Phase 2, bring up every **Yes**
capability via a single bootstrapping command. One row of that
matrix reads:

> Object store: generation-conditional pointer write — **Yes** —
> the ADR-0005 compare-and-swap on `manifests/latest.json` must be
> testable locally.

The Phase 2 root-infrastructure session evaluated three commodity
object-store emulators against this row. The evaluation produced
the following finding: **no commodity emulator examined satisfies
both "locally runnable without auth gating" and "enforces
`ifGenerationMatch`"**.

| Emulator                                      | `ifGenerationMatch` enforcement | Status                                                                     |
|-----------------------------------------------|---------------------------------|----------------------------------------------------------------------------|
| `fsouza/fake-gcs-server`                      | accepts stale-generation writes silently (HTTP 200) | covers every other capability the platform exercises locally                |
| `gcr.io/cloud-storage-image/storage-testbench` | enforces preconditions correctly | requires authenticated access to a Google Cloud project — incompatible with routine contributor flows |
| `oittaa/gcp-storage-emulator`                  | enforces preconditions correctly | does not implement the media-upload endpoint that the ADR-0005 publication primitive uses |

The first emulator is the only candidate runnable by contributors
without sandbox cloud access; it does not enforce `ifGenerationMatch`.
The remaining two enforce preconditions correctly but are excluded
on grounds that contradict the **Yes** definition in ADR-0010
("exercisable by a local test or runnable command").

The ADR-0010 row as written therefore cannot be satisfied by any
commodity emulator available at the time of Phase 2's evaluation.
The deployed `docker-compose.yml` and ADR-0010's matrix disagree on
the fidelity of the local CAS surface, and downstream phases
that read the matrix risk designing local tests against a
capability commodity emulators do not deliver.

This ADR amends ADR-0010 to revise the affected row and to add a
substrate-selection sub-criterion that prevents the same gap from
arising the next time a substrate is evaluated.

---

## Decision

### 1. Amend the object-store CAS row to `Partial`

The "Object store: generation-conditional pointer write" row of
ADR-0010's capability matrix is revised from **Yes** to **Partial**.
The amended row reads:

> Object store: generation-conditional pointer write — **Partial**
> — `ifGenerationMatch` query parameter is accepted by the local
> object-store emulator without an error; precondition enforcement
> is not faithful in commodity emulators (a write with a stale
> `ifGenerationMatch` value succeeds locally where the production
> substrate returns HTTP 412 Precondition Failed). The ADR-0005 §4
> publication primitive
> (`verify → write content by-hash → CAS pointer`) runs locally for
> the API-surface portion; full-fidelity CAS enforcement runs in
> the `sandbox-required` CI lane.

This places the row in the same shape as the existing
"Tabular store: lazy view with `ROW_NUMBER() OVER (...)`" row —
a documented `Partial` with a fidelity-gap note and a
sandbox-required validation path.

### 2. Add a CAS-fidelity sub-criterion to ADR-0010's substrate-selection checkpoint

ADR-0010 Consequence 6 commits that "any future decision to select
or change the platform's object-storage, tabular-store, or
publish-subscribe substrate must verify the substrate provides the
capabilities in this matrix". This ADR extends that checkpoint with
the following sub-criterion:

> Substrates evaluated against this matrix must also be evaluated
> for `ifGenerationMatch` enforcement fidelity. A substrate that
> accepts stale-generation writes without returning HTTP 412 is
> treated as **Partial** for the object-store CAS row, not **Yes**,
> regardless of API-surface coverage.

Future substrate evaluations therefore verify CAS enforcement, not
just API-surface coverage. The same discipline is recommended (but
not required by this ADR) for capabilities outside the CAS row;
each subsequent ADR that touches the matrix may add its own
fidelity sub-criterion.

### 3. Preserve every other ADR-0010 commitment

The remainder of ADR-0010 — every other matrix row, the
Wave-3 scaffolding contract, the CI lane split, every other
consequence — stands unchanged. This ADR's scope is the single row
amended in §1 and the sub-criterion added in §2.

### 4. Preserve every ADR-0005 commitment

The ADR-0005 publication primitive
(`verify → write content by-hash → CAS pointer`) is unchanged. The
production substrate continues to enforce the CAS step; this
amendment narrows only the **locally testable scope** of that
discipline, not the discipline itself.

---

## Consequences

1. **ADR-0010's matrix now matches the deployed substrate.**
   The amended row classifies the CAS surface as `Partial`, and
   `docker-compose.yml` ships the commodity emulator that
   delivers `Partial` fidelity. The two artifacts no longer
   contradict each other.

2. **The `sandbox-required` CI lane gains the CAS-enforcement
   validation.** Production-shape CAS enforcement is exercised in
   the sandbox lane (alongside the tabular-store lazy-view
   fidelity validation already routed there). The `local-runnable`
   CI lane validates the API surface, the by-hash layout, and the
   round-trip read/write — everything except precondition
   enforcement.

3. **Downstream phases inherit the corrected posture.** Wave 3
   Phase 4 (engine runtime), Phase 5 (alerting), and Phase 6
   (first onboarded entity end-to-end) reference ADR-0010 for
   capability coverage. After this ADR lands, those phases read
   the amended row and design their local tests against the
   capability commodity emulators actually deliver — not the
   capability ADR-0010 originally aspired to.

4. **The end-to-end "manifest publish → loader hash-short-
   circuit refresh → execution write → operational alert publish"
   flow still runs locally without sandbox** for the
   non-precondition-enforcement portion. Contributors can exercise
   the entire ADR-0005 publication primitive locally; what they
   cannot exercise locally is the concurrent-publisher CAS race,
   which requires the sandbox.

5. **The amendment is reversible by a future ADR.** If a
   commodity object-store emulator that enforces
   `ifGenerationMatch` correctly **and** runs without sandbox
   cloud access becomes available, a future ADR may re-amend
   the row to **Yes** — evaluated against this ADR's
   sub-criterion (§2). The sub-criterion is the durable artifact
   even after the specific commodity-emulator landscape changes.

6. **The B0-5 publication primitive is unchanged.** ADR-0005's
   `verify → write content by-hash → CAS pointer` discipline
   remains the production contract; only the locally-testable
   scope narrows. Production substrates continue to enforce CAS.

7. **Wave 3 sequencing is unaffected.** ADR-0013's phase
   structure does not change. The amendment does not move any
   capability between phases or reopen any closed phase.

8. **Reopening this amendment** is required if a future evaluation
   discovers that the production substrate itself fails to enforce
   `ifGenerationMatch` — a contingency the platform would treat
   as a substrate-selection emergency rather than a routine
   amendment. Reversing the **Partial** classification when a
   faithful commodity emulator emerges is a routine follow-up ADR,
   not a reopen.

9. **Future capability-row amendments follow this ADR's shape.**
   When a Wave 3 (or later) session discovers that another
   ADR-0010 row's actual local fidelity does not match its
   committed classification, the response is a similarly shaped
   amendment ADR: revise the affected row, optionally add a
   fidelity sub-criterion, leave every other commitment of
   ADR-0010 intact.

---

## Notes

- The amendment intentionally preserves the **Yes** classification
  for "Object store: `by-hash/` immutability with sha256". The
  by-hash immutability surface is enforced by the platform's
  publication primitive (it never re-writes a `by-hash/` path with
  the same hash); commodity emulators that accept stale-generation
  writes on the pointer file still serve the by-hash layout
  faithfully because the platform code does not depend on the
  emulator to enforce immutability at that layer.

- The amendment does not name the specific emulator chosen for
  Phase 2; that artifact is a scaffolding detail per ADR-0010's
  framing. The ADR commits the capability shape ("Partial with
  the fidelity-gap note above"); the deployed emulator can be
  swapped without reopening this ADR so long as the replacement
  satisfies the amended row.

- The CAS-fidelity sub-criterion in §2 is written so that **any**
  substrate evaluation — including a swap of the production
  object store to a non-GCS backend — runs the same fidelity
  check. The wording is host-neutral.
