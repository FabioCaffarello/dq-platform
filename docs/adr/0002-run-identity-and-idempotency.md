<!-- path: docs/adr/0002-run-identity-and-idempotency.md -->

# ADR-0002 — Run Identity and Idempotency

- **Status:** accepted
- **Date:** 2026-05-21

**Scope note (added 2026-05-23):** This ADR currently applies to set-oriented capability (realized over BigQuery). Record-oriented capability is addressed separately in Wave-S (see ADR-0020 forthcoming).

---

## Context

Every quality evaluation the platform performs is a **run**: an
evaluation of a specific entity's rules over a specific time
window, triggered by a specific source. Reporting, alerting,
retries, operator reruns, and downstream analytics all depend
on being able to answer one question precisely: *"is this the
same run, or a different one?"*

The answer is an `execution_id`. Without a fixed identity
formula, scheduler retries fragment into multiple records (the
same trigger looks like two different runs), alert dedup cannot
key on identity, and operator reruns lose their audit link to
the original execution. Once consumers (dashboards, alerting,
incident responders) start treating `execution_id` as a stable
key, **changing the formula is a breaking event** for every
consumer at once.

A key design tension: the identity formula must produce the
same `execution_id` for a scheduler retry that spans an engine
upgrade (so retries remain idempotent), but it must produce a
different `execution_id` for an operator-driven rerun (so the
audit trail is preserved). Resolving the tension requires
choosing which inputs participate in the formula and which are
recorded per-attempt outside the formula.

---

## Decision

### 1. The `execution_id` formula

`execution_id` is computed from **five inputs**:

```
execution_id = sha256_hex(
    ruleset_version || entity || window_start || window_end || trigger_source
)
```

- Canonical encoding: UTF-8 text per the type definitions
  below, joined with a single ASCII pipe character (`|`), **no
  escaping**.
- Hash algorithm: sha256.
- Output: lowercase hexadecimal encoding of the 32-byte digest
  (exactly 64 characters of `[0-9a-f]`).

`engine_version` is **not** in the formula. It is recorded
per-attempt as a non-identity field (see Consequences).

### 2. Input type definitions

- **`ruleset_version`** (string) — the manifest's
  `ruleset_version` field as published, e.g., `rules-v2.4.7`.
  Treated as opaque text. May not contain `|`; the tag
  convention forbids it and the manifest publisher rejects
  any `ruleset_version` containing it.
- **`entity`** (string) — the entity identifier as declared
  in the rule YAML and indexed by the engine's loader. May not
  contain `|`; the linter rejects entity names containing it.
- **`window_start`** (string) — RFC 3339 UTC timestamp with
  second precision and trailing `Z`, matching the regex
  `\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z` exactly. Fractional
  seconds are explicitly excluded.
- **`window_end`** (string) — same form as `window_start`.
- **`trigger_source`** (string) — one of the closed enum
  values below.

### 3. `trigger_source` enum

Initial committed values (lowercase, hyphen-separated):

- `scheduler` — periodic invocation from the scheduler
  subsystem.
- `manual` — one-off invocation via the trigger API by a
  human with appropriate permissions (not a rerun).
- `operator-rerun` — deliberate re-evaluation of a prior run
  via the Admin API; always paired with a
  `supersedes_execution_id` audit field.

Adding a new value is **additive** and does not change any
existing `execution_id`. Removing or renaming a value is a
breaking change and requires a future ADR.

### 4. API-layer enforcement of the `manual` / `operator-rerun` split

The trigger API rejects requests carrying
`trigger_source = operator-rerun`; this value is producible
only by the Admin API rerun endpoint, which mandatorily
requires the prior `execution_id` as a parameter. The Admin
API rerun endpoint rejects requests carrying
`trigger_source = manual` or `trigger_source = scheduler`. The
mapping between API path and enum value is one-to-one,
enforced at the API layer — not by convention.

### 5. Scheduler retries vs. operator reruns

- **Scheduler-driven retry.** Identical five inputs ⇒
  identical `execution_id`. The retry attempt is recorded as
  an additional physical attempt under the same logical
  identity.
- **Operator-driven rerun.** The new trigger uses
  `trigger_source = operator-rerun`, which differs from the
  original's `scheduler` or `manual`, so the formula produces
  a different id. The new run's row records
  `supersedes_execution_id` pointing at the original.

---

## Consequences

1. **The formula is a public contract.** Anyone with the five
   inputs can reproduce an `execution_id` and verify it
   matches a recorded one. There is no hidden state, no
   engine-side randomness, no time-of-evaluation inputs.

2. **Scheduler retries are idempotent at the identity layer.**
   A retry that spans an engine upgrade produces the same
   `execution_id` as the original. Two attempt rows result,
   with the same `execution_id` and different
   `engine_version` values.

3. **Operator reruns are auditable.** Every `operator-rerun`
   row carries a `supersedes_execution_id` pointing at the
   prior execution; chains of operator reruns form a
   queryable history.

4. **`engine_version` is per-attempt metadata, actively
   surfaced.** Every persisted attempt row records
   `engine_version` as a non-identity field. Any reporting
   query or admin tool that returns attempts for an
   `execution_id` includes per-attempt `engine_version` in its
   **default projection** — not behind a flag, not in an
   expanded view. The canonical view returns the latest
   attempt's `engine_version` as the "effective evaluator".
   Forensic queries that surface only the canonical view
   without per-attempt rows are missing the upgrade event by
   construction and must be flagged in review as incomplete
   for incident investigation.

5. **No input may contain the ASCII pipe character.**
   Protection is enumerated per input:
   - `entity` — rejected by the linter.
   - `ruleset_version` — rejected by the manifest publisher
     and forbidden by tag conventions.
   - `window_start`, `window_end` — the RFC 3339 syntax does
     not permit `|`.
   - `trigger_source` — the closed enum contains no `|`, and
     the additive extension policy implicitly forbids future
     values from containing it.
   Adding a sixth input to the formula in a future ADR
   requires defining an equivalent protection.

6. **The hash output is 64 characters of lowercase hex.**
   Storage columns sized to 64 bytes are safe; longer columns
   are also safe. The algorithm is sha256, aligned with the
   manifest layer for a single hash-algorithm posture across
   the platform.

7. **`execution_id` is opaque to consumers.** Downstream
   consumers (dashboards, alert dedup, incident triage) treat
   the id as a string identifier with no internal structure.
   Consumers must not parse, prefix-match, or attempt to
   derive metadata from the bits of the id. Forensic
   reproduction from inputs (via the public formula) is the
   only sanctioned use.

8. **Six idempotency invariants the platform commits to.**
   - *Formula determinism.* Same inputs ⇒ same output, across
     processes, replicas, restarts, and time.
   - *Scheduler-retry idempotency.* Same trigger inputs ⇒
     same `execution_id`.
   - *Alert deduplication.* N scheduler retries of the same
     `execution_id` produce at most one user-visible alert
     per failing check.
   - *Reporting consistency.* A reporting query keyed on
     `execution_id` returns a single canonical view of that
     execution.
   - *Reproducibility.* Given the five inputs of a recorded
     execution, anyone can recompute the `execution_id` and
     verify it.
   - *Audit reachability of operator reruns.* Every
     `operator-rerun` row carries a
     `supersedes_execution_id`; chains form a queryable
     history.

9. **The storage layer must respect the canonical-view
   invariant.** Whatever physical shape `dq_executions` takes
   (append-only, upsert, hybrid), a single query by
   `execution_id` returns one canonical row; multiple
   physical attempt rows must be collapsible by query. The
   physical shape is specified by ADR-0003.

10. **Alert routing dedupes on `execution_id`.** If N events
    arrive for the same `execution_id` and same failing
    `check_id`, the alerting layer emits at most one
    user-visible alert. The dedup window and routing details
    are specified by ADR-0006.

11. **Changing the formula is a breaking change.** Adding or
    removing inputs, changing the encoding, or changing the
    algorithm requires a future ADR with a documented
    migration path and a compatibility window.

---

## Notes

- Window-boundary alignment (whether `window_start` /
  `window_end` must align to whole hours, days, or a schedule
  grid) is a follow-up decision. This ADR commits only that
  the formula accepts RFC 3339 second-precision UTC strings.
- Time precision finer than seconds is out of scope. A future
  move to sub-second precision would be a breaking change.
- Entity naming with namespace prefixes is governed by the
  rules workspace's onboarding contract, refined during
  scaffolding. The formula accepts whatever the loader uses
  as the indexing key.
- A `ci-dry-run` `trigger_source` value is a candidate
  additive enum extension if a CI write target is introduced;
  it is not in the initial committed enum.
