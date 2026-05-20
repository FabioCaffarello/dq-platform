<!-- path: studies/decisions/2026-05-20-run-identity-and-idempotency.md -->

# B0-2 — Run Identity and Idempotency

## Metadata

- B0 reference: B0-2 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md).
- Status: draft (Wave 1, session 4).
- Last updated: 2026-05-20.
- Upstream resolved: none directly required, but B0-1
  ([`2026-05-20-engine-rules-compatibility.md`](./2026-05-20-engine-rules-compatibility.md))
  and B0-5
  ([`2026-05-20-manifest-publication-semantics.md`](./2026-05-20-manifest-publication-semantics.md))
  are referenced for the `ruleset_version` input.
- Downstream open: B0-3 (result write model), B0-4 (failure scope),
  B0-6 (alert routing), B0-7 (loader / scheduler / retry failure
  semantics).
- Promotion target: see final section.

---

## Context

Every quality evaluation the platform performs is a **run**: a
defined evaluation of a specific entity's rules over a specific
time window, triggered by a specific source. Reporting, alerting,
retries, operator reruns, and downstream analytics all depend on
being able to answer one question precisely: *"is this the same
run, or a different one?"*

The answer is an `execution_id`. Foundation doc
[`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
§"Run Identity and Idempotency" sketches the shape:

> A run is uniquely identified by a tuple:
>
>     execution_id = hash(
>         ruleset_version,
>         engine_version,
>         entity,
>         window_start,
>         window_end,
>         trigger_source
>     )
>
> The exact hash inputs and format are a B0 decision.

That sketch carries a tension: the surrounding prose says "the same
**logical** execution (same rules, same data window, same trigger)
produces the same `execution_id`", but `engine_version` is among
the listed inputs — meaning an engine upgrade between a scheduler
trigger and its retry would produce *different* ids for what is
operationally the same logical execution. This study resolves the
tension and locks the formula.

B0-2 — as recorded in the decision log:

> What uniquely defines a run, and how do reruns of the same window
> behave?

This study locks:

1. The exact set of inputs to `execution_id`.
2. The canonical encoding of those inputs.
3. The hash algorithm and output format.
4. The categories of "rerun" the platform distinguishes
   (scheduler-driven retry, operator-driven rerun, CI dry-run) and
   their identity semantics.
5. The idempotency invariants the platform commits to.

What this study does **not** decide:

- The physical storage shape of `dq_executions` (append vs upsert
  vs hybrid) — that is B0-3, and must respect the invariants
  locked here.
- How alert routing deduplicates on `execution_id` — that is B0-6,
  also respecting these invariants.
- What constitutes a check error vs entity error vs run error —
  that is B0-4.
- Retry-budget exhaustion semantics for the scheduler — that is
  B0-7.

The decision matters because once consumers (dashboards, alerting,
incident responders, downstream pipelines) start treating
`execution_id` as a stable key, **changing the formula is a
breaking event** for every consumer at once. This is the kind of
contract that must be locked carefully on the first pass.

---

## Decision Drivers

The decision must satisfy the following, in priority order.

1. **D1. Determinism.** Given identical inputs, the formula must
   produce an identical `execution_id` — across processes,
   replicas, engine restarts, and clock skew. This is P2 (engine
   behavior must be deterministic) applied to identity.

2. **D2. Scheduler-retry idempotency.** A scheduler-driven retry
   of the same trigger must produce the **same** `execution_id` as
   the original attempt. This is what makes the "no duplicate
   alerts" invariant in
   [`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
   §"Retry Semantics" implementable at all.

3. **D3. Operator-rerun distinguishability.** An operator-driven
   rerun must produce a **different** `execution_id` from the
   original, with an explicit audit link back to the original.
   Operator reruns are deliberate events; conflating them with the
   original would erase the audit trail.

4. **D4. Stability across engine versions for the same logical
   trigger.** A scheduler retry that happens to span an engine
   upgrade must still produce the same `execution_id`. The
   evaluator changing between attempts is operationally interesting
   (recorded as a per-attempt field, see Consequences) but does
   not change the logical run identity.

5. **D5. Consumability.** `execution_id` must be a bounded-length
   string suitable as a key in BigQuery tables, in URL paths
   (admin API), in alert payloads, and in log lines. Opaque to
   consumers; reproducible by anyone with the inputs.

6. **D6. Reproducibility from the inputs alone.** Anyone with the
   five inputs can re-compute the `execution_id` and verify it
   matches a recorded one. No hidden state, no engine-side
   randomness, no time-of-evaluation inputs.

7. **D7. No coupling to fields that don't affect the logical
   identity.** Including a field that changes for reasons unrelated
   to identity (engine deploys, log-rotation cycles, etc.) breaks
   D2 and D4. Inclusion must be justified by "this field
   participates in defining what the run *is*".

8. **D8. Forward-compatible enum surfaces.** The `trigger_source`
   input is an enum. New trigger sources will be added over time
   (e.g., when streaming arrives). The enum extension must be
   additive — adding a new value must not change any existing
   `execution_id`.

9. **D9. Algorithm alignment with B0-5.** B0-5
   ([`2026-05-20-manifest-publication-semantics.md`](./2026-05-20-manifest-publication-semantics.md))
   commits sha256 throughout the manifest layer. Using a different
   hash here would multiply the platform's hash-algorithm posture
   surface — two algorithms to maintain, two migration windows in
   any future deprecation. The semantic properties of sha256
   (collision resistance, preimage resistance) are sufficient for
   identity; alignment with B0-5 is the operational win, not
   redundant algorithm posture.

---

## Considered Options

### Option A — Minimal formula (entity + window only)

```
execution_id = sha256_hex(entity || window_start || window_end)
```

**Trade-offs.**

- Pro: simplest. Two reruns within the same window for the same
  entity always collide.
- Con: violates D7 inverted — a rules change between two runs
  produces "same id" for materially different evaluations. Same id,
  different content is the failure mode P2 (determinism) is
  designed to prevent.
- Con: cannot distinguish scheduler retry from operator rerun
  (both produce same id). Violates D3.

Reject.

### Option B — Foundation-doc baseline (six inputs, includes engine_version)

```
execution_id = sha256_hex(
    ruleset_version ||
    engine_version  ||
    entity          ||
    window_start    ||
    window_end      ||
    trigger_source
)
```

**Trade-offs.**

- Pro: maximally conservative — any change in any input produces
  a different id.
- Pro: "same id ⇒ same evaluator" holds; forensic investigators
  reading "execution X" know which engine produced it.
- Con: violates D4 — a scheduler retry that spans an engine
  upgrade gets a different id, fragmenting the logical run across
  two records. Operationally, this is "scheduler did its job
  twice" appearing as "two different runs", which is wrong.
- Con: minor engine patches (bug fixes that don't change check
  semantics) still invalidate idempotency. Engine deploys become
  identity-disruptive events.
- Con: the engine_version is operationally interesting metadata
  but does not *define* a run — it describes the evaluator, not
  the evaluation intent.

### Option C — Five inputs, exclude engine_version

```
execution_id = sha256_hex(
    ruleset_version ||
    entity          ||
    window_start    ||
    window_end      ||
    trigger_source
)
```

`engine_version` is **recorded** on every `dq_executions` row as
a per-attempt non-id field, but does not participate in the id.

**Trade-offs.**

- Pro: D2 and D4 satisfied — scheduler retries across engine
  upgrades preserve identity.
- Pro: identity tracks the *trigger intent*, not the evaluator.
  Operators reading a dashboard see "this trigger ran once,
  succeeded after retry" instead of "this trigger has two records
  because the engine was upgraded".
- Pro: P2 (determinism) is defended at the result-content level
  rather than at the identity level. Two engine versions producing
  different results for the same id shows up as different
  per-attempt content with the same logical id; forensic
  investigators see the engine_version field on the attempt row.
- Con: "same id, potentially different evaluator" is a real
  semantic. Consumers must be aware that `execution_id` keys a
  trigger-intent, not an evaluator-evaluation pair. This is a
  documentation burden.
- Con: a malicious or misconfigured engine could write content
  under an existing id from a prior engine; the per-attempt
  `engine_version` field protects this only after-the-fact.

### Option D — Two-tier (logical_run_id + execution_id)

Two ids per attempt:

```
logical_run_id = sha256_hex(
    ruleset_version || entity || window_start || window_end
)
execution_id = sha256_hex(
    logical_run_id || engine_version || trigger_source || attempt_nonce
)
```

- `logical_run_id` is what dashboards and alerting key on.
- `execution_id` is per-attempt and uniquely identifies a
  physical evaluation attempt.

**Trade-offs.**

- Pro: most expressive — logical identity and physical attempt
  identity are explicitly separated.
- Pro: consumers can choose their semantic — alert dedup on
  `logical_run_id`; forensic queries on `execution_id`.
- Con: every consumer must learn the two-level model. The
  documentation burden is significantly higher than C.
- Con: with `attempt_nonce` in the formula, scheduler retries no
  longer produce the same `execution_id` (only the same
  `logical_run_id`). Idempotency of the *execution_id* layer is
  abandoned; alert systems must shift to keying on
  `logical_run_id`.
- Con: introduces complexity that B0-3 (storage), B0-6 (alert
  routing), and every downstream consumer must navigate. The
  benefits accrue mostly to forensic investigation, which is
  served almost as well by Option C's per-attempt `engine_version`
  field.

Reject as over-engineering for the marginal forensic gain over
Option C.

---

## Recommendation

Adopt **Option C** — five inputs, `engine_version` excluded from
the formula and recorded as per-attempt metadata.

The recommendation is grounded in:

- foundation doc
  [`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
  §"Run Identity and Idempotency" — the framing of "same logical
  execution" the doc itself uses;
- foundation doc
  [`04-system-architecture.md`](../foundation/04-system-architecture.md)
  §"Execution Flow" — the trigger handler creates the plan and
  persists the row, before the engine version that processes it
  is necessarily fixed (a long-running queue could be drained by
  a different engine instance);
- foundation doc
  [`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
  §"Retry Semantics" — the explicit distinction between
  scheduler-driven retries (idempotent, same id) and
  operator-driven reruns (auditable, new id);
- prior decision
  [`2026-05-20-manifest-publication-semantics.md`](./2026-05-20-manifest-publication-semantics.md)
  — sha256 algorithm posture.

The specific commitments beyond what those documents state are
**new contribution proposed here, requires review**:

1. Exclude `engine_version` from the `execution_id` formula. The
   foundation doc lists it as an input; this study removes it on
   the D4/D7 grounds above and records it as per-attempt metadata
   instead. This trade-off privileges scheduler-retry idempotency
   (D4) over forensic legibility at the id layer; forensic
   legibility is preserved at the attempt-row layer via CC4 and
   actively surfaced via CC14. **New contribution proposed here,
   requires review.**
2. Canonical encoding: pipe-separated UTF-8 concatenation of the
   five inputs, with strict no-pipe constraint on input values.
   **New contribution proposed here, requires review.**
3. `trigger_source` is a closed enum with initial values
   `scheduler`, `operator-rerun`, `manual`, `ci-dry-run`. Extension
   is additive. **New contribution proposed here, requires review.**
4. Operator reruns record `supersedes_execution_id` as a field on
   the new run's row, pointing at the prior execution. The storage
   shape of this field is B0-3's call; this study commits the
   semantic. **New contribution proposed here, requires review.**
5. Six idempotency invariants the platform commits to (I1–I6 in
   CC9 below). **New contribution proposed here, requires review.**

---

## Consequences

Adopting this recommendation commits the platform to the following.

**CC1. `execution_id` formula is fixed.** Five inputs in the order
`ruleset_version || entity || window_start || window_end || trigger_source`.
Canonical encoding: each input is rendered as UTF-8 text per the
type definitions in CC2; the five renderings are joined with a
single ASCII pipe character (`|`); no escaping. The result is
hashed with sha256; the output is the lowercase hexadecimal
encoding of the 32-byte digest (64 characters). This is the
`execution_id`.

**CC2. Input type definitions are fixed.** Each input's wire form:

- `ruleset_version` (string) — the value of the manifest's
  `ruleset_version` field as published by B0-5, e.g.
  `rules-v2.4.7`. Treated as opaque text; no parsing. May not
  contain the ASCII pipe character; this is enforced upstream by
  the tag convention in foundation doc
  [`02-monorepo-topology.md`](../foundation/02-monorepo-topology.md)
  §"Tag conventions" (the convention restricts the format to a
  pipe-free character set) and by B0-5's manifest publisher
  rejecting any `ruleset_version` value containing `|`.
- `entity` (string) — the entity identifier as declared in the
  rule YAML and indexed by the engine's loader (foundation doc
  [`04-system-architecture.md`](../foundation/04-system-architecture.md)
  §"PAT-1"). May not contain the ASCII pipe character; the
  linter (foundation doc
  [`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
  §"Surface 2") rejects entity names containing `|`.
- `window_start` (string) — RFC 3339 UTC timestamp with second
  precision and trailing `Z`, e.g. `2026-05-20T14:00:00Z`.
  Fractional seconds are explicitly excluded; the value must
  match the regex `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z` exactly.
- `window_end` (string) — same form as `window_start` (same regex
  constraint).
- `trigger_source` (string) — one of the enum values committed in
  CC6.

**CC3. Scheduler-driven retries preserve `execution_id`.** When
the scheduler retries a trigger because the trigger handler was
unreachable (per foundation doc
[`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
§"Retry Semantics"), the retry passes identical inputs to the
formula. The `execution_id` is unchanged. The retry attempt is
recorded as an additional physical attempt under the same logical
identity; how that is stored is B0-3.

**CC4. Engine version is recorded per attempt, not in the id.**
Every persisted attempt row records `engine_version` as a non-id
field. A scheduler retry that spans an engine upgrade therefore
produces two attempt rows with the same `execution_id` and
different `engine_version` values. Forensic investigators reading
the attempt rows see the upgrade event explicitly.

**CC5. Operator-driven reruns produce a new `execution_id` and
record the audit link.** When an operator triggers a rerun via
the Admin API, the new trigger uses `trigger_source = "operator-rerun"`,
which differs from the original `trigger_source` ("scheduler" or
"manual"); the formula produces a different id. The new run's row
records `supersedes_execution_id` containing the original
`execution_id`. The exact storage shape of `supersedes_execution_id`
(column, JSONB field, separate table) is B0-3.

The boundary between `manual` (trigger API) and `operator-rerun`
(Admin API) is enforced at the **API layer**, not by convention:

- The trigger API **rejects** requests carrying
  `trigger_source = operator-rerun`. This value is producible
  only by the Admin API rerun endpoint, which mandatorily
  requires the prior `execution_id` as a parameter and records
  it as `supersedes_execution_id`.
- The Admin API rerun endpoint **rejects** requests carrying
  `trigger_source = manual` or `trigger_source = scheduler`. The
  endpoint is single-purpose.

The mapping between API path and `trigger_source` enum value is
**one-to-one, enforced at the API layer**. An operator who picks
the wrong endpoint receives an explicit error and must use the
correct one — discipline is not load-bearing for I6 (audit
reachability of operator reruns).

**CC6. `trigger_source` is a committed enum.** Initial values,
all lowercase, hyphen-separated:

- `scheduler` — periodic invocation from the scheduler subsystem.
- `manual` — one-off invocation via the trigger API by a human
  with appropriate permissions (not a rerun).
- `operator-rerun` — a deliberate re-evaluation of a prior run
  via the Admin API. Always paired with a
  `supersedes_execution_id` field (CC5).

Adding a new value to this enum is **additive** and does not
break existing `execution_id`s (no value can produce a different
encoding for an existing value). Removing or renaming a value is
breaking and requires a future ADR. A candidate fourth value —
`ci-dry-run` — is deferred to B0-3 per OQ-6; B0-3 adds it if and
when it creates a CI write target, under the additive extension
policy above.

**CC7. Hash algorithm is sha256, output is 64-char lowercase hex.**
Aligned with B0-5's sha256 commitment across the manifest and
content-addressed paths. `execution_id` strings are exactly 64
characters of `[0-9a-f]`. Storage columns sized to 64 characters
of bytes are safe; longer columns are also safe.

**CC8. `execution_id` is opaque to consumers.** Downstream
consumers (dashboards, alert dedup, incident triage tools) treat
`execution_id` as a string identifier with no internal structure.
Consumers MUST NOT parse, prefix-match, or attempt to derive
metadata from the bits of the id. Forensic reproduction of the id
from inputs is supported via the formula in CC1; that is the only
sanctioned use of the formula by anything outside the trigger
handler.

The formula is public (I5) but the id is opaque (this CC). These
are consistent, not contradictory: sha256's preimage resistance
means the id reveals nothing about its inputs to anyone who does
not already know them; anyone who does know the inputs can verify
they match. Reproduction of an id from inputs is sanctioned;
reverse-derivation of inputs from an id is computationally
infeasible.

**CC9. Idempotency invariants the platform commits to.** Six
invariants:

- **I1. Formula determinism.** The formula in CC1 produces the
  same output for the same inputs, across processes, replicas,
  restarts, and time.
- **I2. Scheduler-retry idempotency.** A scheduler-driven retry of
  the same trigger produces the same `execution_id` as the
  original attempt.
- **I3. Alert deduplication.** N scheduler-driven retries of the
  same `execution_id` produce at most one user-visible alert per
  failing check. (The deduplication mechanism is B0-6's call;
  this study commits only the invariant.)
- **I4. Reporting consistency.** A reporting query keyed on
  `execution_id` returns a single canonical view of that
  execution. The physical storage shape (number of rows, retry
  history representation) is B0-3's call; this study commits the
  invariant that the canonical view is single-rowed.
- **I5. Reproducibility.** Given the five inputs of a recorded
  execution, anyone can recompute the `execution_id` and verify
  it matches the recorded value.
- **I6. Audit reachability of operator reruns.** Every
  `operator-rerun` row carries a `supersedes_execution_id`
  pointing at a prior execution; chains of operator reruns form
  a queryable history.

**CC10. B0-3 storage model must respect I4.** B0-3 picks
append-only, upsert, or hybrid for `dq_executions`. Whichever it
picks, a single query by `execution_id` must return one canonical
row; multiple physical attempt rows (if hybrid) must be
collapsible by query.

**CC11. B0-6 alert routing must dedupe on `execution_id`.**
Specifically: if N events arrive for the same `execution_id` and
the same failing `check_id`, the alerting layer emits at most one
user-visible alert (I3). The exact dedup window and routing rules
are B0-6's call.

**CC12. Reserved characters and length bounds on inputs.** **No
input may contain the ASCII pipe character.** The protection
mechanism for each input is enumerated:

- `entity` — linter rule in `tools/lint` (per B0-1 / foundation
  doc
  [`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
  §"Surface 2") rejects entity names containing `|` at validation
  time.
- `ruleset_version` — tag convention in foundation doc
  [`02-monorepo-topology.md`](../foundation/02-monorepo-topology.md)
  §"Tag conventions" plus B0-5's manifest publisher rejection of
  any `ruleset_version` containing `|` (per CC2).
- `window_start`, `window_end` — RFC 3339 timestamp syntax per
  CC2 does not permit the pipe character.
- `trigger_source` — closed enum per CC6; no committed value
  contains `|`, and the extension policy implicitly forbids any
  future value from containing `|`.

This enumeration is exhaustive for the five inputs in CC1.
Adding a sixth input to the formula in a future ADR requires
defining an equivalent protection for that input.

Maximum input lengths are not bounded by this study (the hash
absorbs any length); storage-layer column sizes are B0-3.

**CC13. The formula is the contract.** Once published, the
formula in CC1 is a public contract. Changing it (adding or
removing inputs, changing the encoding, changing the algorithm)
is a **breaking change** for every consumer that has recorded an
`execution_id`. Such a change requires a future ADR with a
documented migration path and a compatibility window — analogous
to the schema-version migration protocol in B0-1
([`2026-05-20-engine-rules-compatibility.md`](./2026-05-20-engine-rules-compatibility.md)).

**CC14. Heterogeneous engine_versions across attempts are
operationally interesting, not erroneous.** When attempts of the
same `execution_id` carry different `engine_version` values (a
scheduler retry that spanned an engine upgrade per CC3), the
platform commits to **active visibility** rather than passive
recording:

- Any reporting query or admin tool that returns attempts for an
  `execution_id` must include the per-attempt `engine_version` in
  its **default projection** — not behind a flag, not in an
  expanded view.
- The canonical view per I4 returns the **latest attempt's**
  `engine_version` as the "effective evaluator" for that
  execution. Forensic queries that surface only the canonical
  view (without the per-attempt rows) are missing the upgrade
  event by construction and **must be flagged in review** as
  incomplete for incident-investigation purposes.

This converts CC4's passive metadata commitment into an active
visibility commitment: the platform does not merely *record* that
an upgrade happened; the platform's reporting surfaces *show* it
happened. **New contribution proposed here, requires review.**

---

## Open Questions

- **OQ-1. Exact storage shape of `supersedes_execution_id`.**
  Whether the audit link is a column on the same table, a JSONB
  field, a separate `dq_run_lineage` table, or another shape is
  **out-of-scope for current cycle** — it is B0-3.

- **OQ-2. Window-boundary alignment policy.** Whether
  `window_start` and `window_end` are required to align to specific
  boundaries (whole hours, whole days, schedule grid) is
  **out-of-scope for current cycle** — schedule normalization is
  B1-3 (scheduler catchup behavior). This study commits only that
  whatever the boundary policy is, both window timestamps are
  passed to the formula as RFC 3339 UTC second-precision strings.

- **OQ-3. Time-precision finer than seconds.** Whether sub-second
  precision (millisecond, nanosecond) is ever needed in `window_start`
  / `window_end` is **out-of-scope for current cycle** — second
  precision is the commitment here. A future move to sub-second
  precision is a breaking change to the formula (CC13).

- **OQ-4. Entity naming with namespace prefixes.** Whether
  `entity` is the bare entity name or includes a namespace prefix
  (e.g., `<source>.<entity>`) is **out-of-scope for current cycle**
  — entity naming is governed by the rules workspace's onboarding
  contract and is refined as part of Wave 3 scaffolding. This
  study commits only that the value passed to the formula is
  whatever the engine's loader (foundation doc
  [`04-system-architecture.md`](../foundation/04-system-architecture.md)
  §"PAT-1") uses as the indexing key.

- **OQ-5. Migration path for a future formula change.** If a
  future ADR changes the formula (CC13), how existing
  `execution_id`s coexist with new ones is **out-of-scope for
  current cycle**. A migration ADR is opened if and when needed.

- **OQ-6. CI dry-run write target and enum value.** Whether CI
  dry-runs write to a separate dry-run table, to a flagged
  partition of `dq_executions`, or to no table at all is
  **out-of-scope for current cycle** — it is B0-3's call. The
  `ci-dry-run` trigger_source value is **not** in the initial
  enum committed by CC6; it is a candidate enum value that B0-3
  may add if it creates a CI write target (additive per CC6
  extension policy). If B0-3 declines to create such a target,
  the enum remains as committed and CI dry-runs do not require a
  distinct trigger_source. Either way, CI dry-runs must not
  pollute production reporting state for the same `execution_id`.

No open question in this list blocks the identity model itself.
All items above are operational refinements or downstream-storage
concerns that the locked formula leaves open by design.

---

## Promotion target

This study is promoted during Wave 3 to:

    docs/adr/0002-run-identity-and-idempotency.md

The `0002` is provisional and assigned at promotion time. If the
Wave 3 ADR numbering convention orders by promotion date rather
than by B0 sequence, the number changes; the slug
(`run-identity-and-idempotency`) does not. This follows the same
convention adopted in
[`2026-05-20-engine-rules-compatibility.md`](./2026-05-20-engine-rules-compatibility.md)
(B0-1, Promotion target section).

The MADR ADR rewrites this study for an external-reviewer
audience (no `studies/` back-references per R8), folds in any
updates from B0-3 (result write model) and B0-6 (alert routing)
that intersect with the identity contract, and updates the
relevant sections of
[`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
to reference the ADR (specifically resolving the "exact hash
inputs and format are a B0 decision" pointer in §"Run Identity
and Idempotency").
