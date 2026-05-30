<!-- path: docs/runbooks/amendment-log.md -->

# Runbook — Land an in-place data-row amendment on a committed ADR

Operational procedure for the "Amendment log" convention committed
by [ADR-0050](../adr/0050-v1-retirement-engine-release.md) §(c).
Use this when a slice needs to mutate a data row inside a long-
lived structure (table, list, configuration matrix) that an earlier
ADR has committed as an authoritative reference, **without** any
change to the surrounding prose.

The convention exists so future v(N) retirements, env-matrix
extensions, and similar bookkeeping changes have a uniform shape
that preserves single-source-of-truth and avoids fragmenting state
across superseding ADRs that drop a chain of pointers.

---

## 1. When to use

Use this runbook when **all four** of these conditions hold:

- The change is a mutation of one or more **rows / cells** inside a
  structure (table, enumerated list, configuration matrix) that an
  earlier ADR explicitly committed as a long-lived authoritative
  reference.
- The structure's **shape is unchanged** — no new columns, no new
  status names, no new field types beyond what the host ADR already
  enumerates.
- The change has **no architectural-prose impact** anywhere in the
  host ADR (Context, Decision body, Consequences).
- The **driving rationale lives in a separate ADR** (the
  "downstream ADR" that ships the underlying change). The host
  amendment is bookkeeping; the rationale is cited, not duplicated.

Canonical example: a v(N) retirement transitioning a row in
ADR-0035's compatibility-state table from `deprecated` to
`retired` and filling the new-but-already-committed `dropped`
field. ADR-0050's v1-retirement is the first instance of this
shape.

Do **not** use this runbook for:

- Changes that modify the host structure's shape — adding a
  column, introducing a new status name, widening a field's
  type. Those require an amendment to the host structure itself
  before any row mutation, and the structural change is not
  bookkeeping.
- Reclassifying a row in a *taxonomy* (e.g., moving a substrate
  capability from `Local-only` to `Partial`). That is the
  explicit precedent of
  [ADR-0017](../adr/0017-substrate-posture-amendment.md) — a
  superseding ADR carries the reclassification together with the
  prose impact it implies. ADR-0050 §(c) calls out this
  distinction directly.
- Changes that require editing prose anywhere else in the host
  ADR. The "no prose impact" gate is the load-bearing criterion;
  if the host ADR's narrative needs an update, the change is
  scope, not bookkeeping.
- Renaming the host structure's columns or the structure itself.
  Those are structural and require an amendment to the structure
  before any row mutation.

If any condition fails, the change requires a superseding ADR or
an amendment ADR — not this runbook.

## 2. Preconditions

- The host ADR exists in `docs/adr/` and its committed structure
  (the table / list / matrix being touched) is intact in the
  current `main`. Confirm before editing: a structural drift on
  `main` since the host ADR landed indicates the structure is no
  longer the single source of truth and a superseding ADR is the
  correct path.
- The downstream ADR that drives the change is at status
  `accepted` (or being accepted in the same merge cycle). The
  Amendment log entry cites it by number; a missing or
  not-yet-accepted downstream ADR means the amendment is
  premature.
- The amendment date is known. The default is the merge date of
  the downstream PR; absolute dates only (no "Thursday" or "next
  release").
- Write access to the host ADR file. The host ADR lives in
  `docs/adr/`; CODEOWNERS routes review to platform-team.

## 3. Procedure

The procedure is the same whether this is the first amendment to
the host ADR or a subsequent one. The first amendment creates the
"Amendment log" subsection; subsequent amendments append to it.

### 3.1 Identify the row(s) and the committed fields

Open the host ADR. Locate the row to mutate. Confirm the fields
being written are already enumerated by the host ADR's committed
shape (column headers, list keys, matrix labels). If a field is
not yet committed by the host ADR, stop: this is structural
change, not bookkeeping.

### 3.2 Make the in-place edit

Edit the row's cells in the host ADR's table or list. Constraints:

- **Do not delete the row.** A retiring row stays in the
  structure with its status updated; deletion would erase the
  historical identity the structure exists to preserve. ADR-0050
  §(e) commits "No row removal" for the compatibility-state
  table; treat the same posture as the default for any
  amendment-log mutation.
- **Preserve historical-context fields verbatim.** A field whose
  original value carries historical meaning stays in the row
  even if a newer field operationally supersedes it. Example
  from the v1 drop: the row's `earliest_drop` value (the 90-day
  floor date the ADR-0035 deprecation set) remains in the row
  alongside the newly-filled `dropped` field; the two together
  document both the contract and its execution.
- **Edit only the cells you are amending.** Touch one row; leave
  adjacent rows untouched. The diff should read as a row update,
  not a table reshape.

### 3.3 Add or extend the Amendment log subsection

The host ADR gains a subsection titled `## Amendment log` at the
bottom, after `## Consequences`. The first amendment creates the
subsection; subsequent amendments append entries in chronological
order. Each entry uses this shape:

```markdown
## Amendment log

- **YYYY-MM-DD** — Row(s) affected: `<row identifier>`. Rationale:
  ships the change committed by
  [ADR-XXXX](./XXXX-<slug>.md) §<section>. <One short
  sentence summarising what changed inside the row.>
```

Concrete example (the v1-drop amendment ADR-0050 will land
against ADR-0035):

```markdown
## Amendment log

- **2026-08-24** — Row(s) affected: `v1` of the compatibility-
  state table. Rationale: ships the drop committed by
  [ADR-0050](./0050-v1-retirement-engine-release.md) §(c).
  Status transitions `deprecated` → `retired`; the previously-
  committed `dropped` field is populated as
  `2026-08-24 / engine-v0.2.0`.
```

The entry is the minimum that lets a reader land on the host ADR
and reconstruct what changed, when, and why, without following
pointers to find the canonical state — the structure remains the
single source of truth.

### 3.4 Land the amendment

Land the host-ADR edit in **the same PR as the downstream
change** when the timing allows. The reviewer sees both the
driving change and its bookkeeping side-effect in one diff;
operators see them ship atomically.

When the downstream PR has already merged and the amendment was
missed, land a **separate follow-up PR carrying only the
bookkeeping**. The follow-up PR's body cites the downstream PR
that should have included the amendment, so the link survives
in PR history.

Direct edits to ADRs already in `accepted` status follow the
host ADR's own amendment mechanism (which is the convention
this runbook documents); no separate approval ceremony is
required beyond standard PR review.

## 4. Verification

After the PR lands, the host ADR diff must show **only**:

1. The row mutation in the targeted structure (one or more cells
   updated; no row added or deleted; no column added; no
   adjacent row touched).
2. The Amendment log subsection (created or extended) with the
   new entry.

Run the checks:

- The host structure's existing rows still parse — each row has
  the same column count as before; the status ladder (or
  equivalent enumeration) still covers every row's value.
- The Amendment log entry cites a real ADR by number that exists
  in `docs/adr/` at `accepted` status.
- A reader who opens **only the host ADR** (no other tabs, no
  other repository state) sees the current state of every row.
  Cross-reference state living outside the host structure
  defeats the convention.
- No prose elsewhere in the host ADR is touched. `git diff
  docs/adr/<host>.md` shows changes restricted to the structure
  section and the Amendment log subsection.

If any check fails, the change is misclassified; treat it as a
structural change and follow §5.

## 5. Rollback / escape

Per the runbook directory's append-only posture
([ADR-0003](../adr/0003-result-write-model.md)), "rollback" of
an amendment is a **forward corrective amendment**, not a silent
revert. The historical entry stays in the Amendment log; a
second entry with the new date documents the correction and its
rationale.

Two corrective paths cover the practical cases:

- **Wrong row updated, or wrong fields written.** Land a second
  amendment that returns the row to its prior state (or to the
  intended state) and write an Amendment log entry citing the
  original entry as the one being corrected. Both entries
  remain in the log; the latest entry is the operative one.
- **Misclassification: the change actually had prose impact.**
  The "no prose impact" gate (§1) was breached after the fact —
  e.g., a comment or paragraph elsewhere in the host ADR now
  reads false. Revert the in-place row edit and author a
  superseding ADR following the
  [ADR-0017](../adr/0017-substrate-posture-amendment.md)
  pattern. The superseding ADR carries both the row change and
  the prose update, and its Context section briefly notes that
  the in-place attempt was inappropriate and was reverted.

The misclassification escape is rare but explicitly allowed;
forcing the change through as an in-place amendment when the
gate was breached erodes the structure the convention exists to
protect.

## 6. Escalation

- **Ambiguity about whether a change is bookkeeping or scope.**
  Escalate to platform-team via PR review before merge. Default
  to "scope" when uncertain — the cost of an unnecessary
  superseding ADR is one-time and visible; the cost of
  mis-using in-place amendment is structural erosion that
  compounds across future amendments.
- **Repeated amendments to the same row in short succession.**
  A signal that the host structure may be incomplete (a column
  or status name is missing that would make the row stable).
  Open a B-row in `studies/foundation/06-decision-log.md`
  proposing a structural revisit, or author an amendment ADR
  that extends the host structure; the in-place pattern is not
  the right vehicle for the next round.
- **The host ADR is itself under amendment review when the slice
  arrives.** Coordinate with the open amendment author before
  landing a parallel in-place edit; two amendments to the same
  table in flight on different branches risk losing one entry
  to a merge conflict.
- **CODEOWNERS escalation.** All ADR changes route to platform-
  team per [`/.github/CODEOWNERS`](../../.github/CODEOWNERS);
  see [`../governance.md`](../governance.md) §2. A
  bookkeeping-classed amendment still receives the standard
  ADR review treatment.
