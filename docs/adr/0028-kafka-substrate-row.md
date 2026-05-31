<!-- path: docs/adr/0028-kafka-substrate-row.md -->

# ADR-0028 — Kafka Substrate Row (Amends ADR-0010)

- **Status:** accepted (amends ADR-0010)
- **Date:** 2026-05-25

---

## Context

[ADR-0010](./0010-substrate-posture.md) committed the substrate
posture contract for set-oriented capability — a capability
matrix the local Docker Compose environment must satisfy plus the
**Yes / Partial / No** rows that govern which capabilities run
locally vs. require sandbox access. ADR-0010 scope-noted that
record-oriented capability would be addressed separately in
Wave-S.

Wave-S then committed an event-stream substrate via
[ADR-0024](./0024-window-semantics.md) — record-mode evaluations
read from a Kafka topic bounded by a tumbling watermark window
with a per-source `consumer_group` and per-partition offset
tracking. The substrate-posture contract has not yet committed
which capability rows the Kafka surface occupies in the matrix:

- Whether the publish + subscribe + per-partition-offset surface
  must run locally without sandbox access, the same way ADR-0010
  committed Pub/Sub publish/subscribe must.
- Whether the consumer group + watermark + offset semantics
  required by [ADR-0024](./0024-window-semantics.md) fall under
  **Yes** (full local fidelity) or **Partial** (commodity
  emulator gaps require sandbox lane).
- Whether the substrate-selection checkpoint from ADR-0010
  Consequence 6 extends to Kafka selection, or whether Kafka
  is a parallel substrate that has its own selection criteria.

The platform principles bearing on the decision: **P4**
(cost is a first-class constraint — the local lane must
exercise the record-mode runtime without sandbox spend); **R5**
(commodity substrates we run on are exempt from the
naming prohibition — Kafka is the environment; specific image
choices are scaffolding); **R8** (forward-only — this ADR is the
authoritative substrate-posture contract for Kafka and rewrites
the relevant capability rows in our own terms).

---

## Decision

[ADR-0010](./0010-substrate-posture.md)'s capability matrix is
amended with a Kafka substrate group. The amendment is in this
ADR rather than in ADR-0010 itself per the forward-only pattern
([CLAUDE.md §3 R8](../../CLAUDE.md)) — accepted ADRs are not
rewritten retroactively; an amendment ships as a new ADR that
the original gains a status pointer to.

### New capability rows

| Capability | Local emulation | Notes |
|---|---|---|
| Event stream: publish + subscribe with consumer groups | **Yes** | The ADR-0024 record-mode evaluation flow (a consumer group reads a topic bounded by a window) must be exercisable end-to-end without sandbox. Equivalent posture to ADR-0010's "Pub/Sub publish/subscribe" row. |
| Event stream: per-partition offset tracking with seek | **Yes** | ADR-0024 §C-B0S4.3 commits per-partition-offset re-reads on attempt retries. The substrate must expose offset commit / seek primitives the engine can call from a consumer group. |
| Event stream: watermark / lateness semantics | **Partial** | Watermark progression and the lateness-tolerance grace period from ADR-0024 are **engine-side concerns**; the substrate provides the message timestamps the engine derives watermarks from. The Partial row reflects that commodity emulators may not faithfully reproduce broker-side timestamp authority (server-set timestamps vs. producer-set timestamps); full-fidelity timestamp authority validation runs in the sandbox lane. |

The set-oriented rows from ADR-0010 (Pub/Sub publish/subscribe,
object store, tabular store, OIDC, etc.) are unchanged.

### Capability decomposition

The event-stream substrate is decomposed into three rows rather
than collapsed into one because each row has a distinct local-vs-
sandbox fidelity story:

- The **publish + subscribe with consumer groups** row mirrors
  ADR-0010's Pub/Sub row: commodity emulators handle this
  cleanly at the wire-protocol level. **Yes** is unambiguous.
- The **per-partition offset tracking with seek** row is the
  load-bearing capability for ADR-0024's per-attempt
  re-evaluation semantics. Commodity emulators support the
  offset surface (commit, seek, list); **Yes** matches that
  reality.
- The **watermark / lateness semantics** row is **Partial** for
  the same reason ADR-0010's tabular-store lazy-view row is
  Partial: commodity emulators have a known fidelity gap around
  the underlying primitive (broker-set timestamps in this case;
  window-function fidelity in the lazy-view case). The engine's
  watermark math is portable across the gap, so local validation
  of engine logic is still **Yes**-level; production-shape
  timestamp-authority semantics require the sandbox lane.

### Wave-3 scaffolding contract (extended)

The local Compose environment must, in addition to the rows
committed by [ADR-0010](./0010-substrate-posture.md):

- bring up an event-stream service that exposes the three
  capability rows above;
- include the service in the single bootstrapping command
  (`make up`);
- ship a smoke test against the publish + subscribe + consumer-
  group + offset-tracking surface.

The specific Kafka-compatible image choice (Redpanda,
Apache Kafka KRaft, etc.) is a scaffolding detail; it can change
without re-opening this ADR so long as the three capability rows
above are preserved.

### CI lane mapping

The CI lane split from [ADR-0010](./0010-substrate-posture.md)
extends naturally:

- **`local-runnable`** — covers the publish + subscribe + offset-
  tracking surface; the engine's watermark math is exercisable
  against the local stream.
- **`sandbox-required`** — covers the broker-side timestamp-
  authority semantics that the Partial row scope-notes.

The split mechanism is unchanged from
[ADR-0010](./0010-substrate-posture.md) (test labels / build
tags).

### Substrate-selection checkpoint (extended)

[ADR-0010](./0010-substrate-posture.md) Consequence 6 commits
that any future decision to select or change the platform's
object-storage, tabular-store, or publish-subscribe substrate
must verify the matrix capabilities. This ADR extends the
checkpoint to the event-stream substrate: any future decision to
select or change the platform's event-stream substrate must
verify the three capability rows above (publish + subscribe with
consumer groups; per-partition offset tracking with seek;
watermark / lateness semantics at least at the Partial level a
commodity emulator provides). A substrate missing any **Yes**
capability row reopens this ADR.

---

## Consequences

1. **The substrate-posture contract from
   [ADR-0010](./0010-substrate-posture.md) is the
   authoritative substrate matrix.** This ADR extends that
   contract with three Kafka capability rows; the rows from
   ADR-0010 are unchanged. ADR-0010 gains a status pointer to
   this ADR via the standard amendment-by-new-ADR pattern.

2. **The `local-runnable` CI lane gains an event-stream
   service.** The lane must bring up a Kafka-compatible
   service via `make up` and exercise the publish + subscribe
   + consumer-group + offset surface in a smoke test, matching
   the existing Pub/Sub posture.

3. **Watermark / lateness fidelity is sandbox-recommended.**
   Engine-side watermark math is portable across the local
   gap; production-shape timestamp-authority semantics are
   verified in the sandbox lane. This matches the
   ADR-0010 lazy-view row's posture (engine logic portable
   locally; full primitive fidelity in sandbox).

4. **Specific Kafka-compatible image choice is a scaffolding
   detail.** The matrix commits the capability rows, not the
   image. The Compose service can change images without
   re-opening this ADR so long as the rows are preserved.

5. **The substrate-selection checkpoint applies to Kafka.**
   Any future decision to swap the event-stream substrate
   must verify the three capability rows above. A substrate
   missing any **Yes** row reopens this ADR.

6. **ADR-0010's Wave-3 scaffolding contract is extended.**
   Phase 7 substrate scaffolding now provisions the event-
   stream service alongside Pub/Sub, the object store, and
   the tabular store. The smoke-test surface gains the
   event-stream smoke alongside the existing three.

---

## Notes

- Specific Kafka-compatible image choice is a Phase 7
  scaffolding detail. Image evaluation considers startup time,
  image size, single-broker support, KRaft mode (no separate
  ZooKeeper dependency), and the offset / consumer-group
  surface fidelity. The Compose service ships with a tag
  reference initially; digest pinning is the ADR-0008 follow-
  up that applies uniformly to every Compose service.
- The contributor onboarding artifact that documents sandbox
  access (per [ADR-0010](./0010-substrate-posture.md)
  Consequence 2) gains a note for the watermark / lateness
  Partial row. Routine record-mode contributor flows do not
  require sandbox access.
- The mechanism for in-cluster Kafka access (per-environment
  bootstrap addresses, IAM/SASL credentials, network exposure)
  is a per-environment Wave-3 overlay session detail. The
  engine reads the bootstrap address from typed env config
  (PAT-4 — see [ADR-0018](./0018-typed-env-config.md)).
- A future ADR may amend this contract if a substrate
  selection commits to a non-Kafka event-stream protocol; the
  amendment ships under the same forward-only pattern this ADR
  uses.
