<!-- path: docs/adr/0048-critique-rounds-preservation.md -->

# ADR-0048 — Critique-Rounds Preservation

- **Status:** accepted
- **Date:** 2026-05-27

---

## Context

The `/critique` command at
[`.claude/commands/critique.md`](../../.claude/commands/critique.md)
runs an adversarial review of a decision document and emits
findings to stdout using the
`[severity] R/P/AC label: section — change` template defined in
[`.claude/playbooks/feedback-protocol.md`](../../.claude/playbooks/feedback-protocol.md).
[`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
step 6 invokes the command and step 7 caps the cycle at two
rounds per decision. The findings drive a revision pass on the
target document; once revised, only the diff (draft → revised)
persists in git. The critique transcript itself is ephemeral.

Empirically, more than thirty decision documents have passed
through one or more critique rounds, and only commit `926e3e5`
([ADR-0014](./0014-trigger-handler-contract.md), the W3-P4e
HTTP trigger handler) preserved its round-1 transcript
end-to-end — captured informally in the PR description for #10
rather than as a standalone repository artifact. Every other
round produced substance that was salvaged into
[`.claude/skills/critique-anti-patterns/`](../../.claude/skills/critique-anti-patterns/)
(extracted as a documented catalog in commit `060dd10`), but
no standalone transcript exists for any other round.

The gap matters for future critique calibration. The
anti-patterns catalog is denser than any single critique would
be, but it is abstract — it cannot show how a particular
`important` finding was scoped across rounds, why an operator
accepted a `minor` finding without revision, or how the
disposition table for a specific document evolved. Without
preserved transcripts, future critique passes rely only on
the abstracted catalog and on opaque draft→revision diffs.

The principles bearing on the decision are **P3** (ownership
is explicit — the preservation step is operator-owned), **P5**
(evolution must be contract-driven — the preservation shape,
including file path, content layout, and skip declaration, is
a documented contract that future critique runs follow), and
**R8** in [`CLAUDE.md`](../../CLAUDE.md) §3 (studies are
reasoning artifacts under `studies/`, not the published
repository — this constrains where preserved critiques may
live).

Two existing surfaces are load-bearing for what follows:

- The `/critique` command's stdout shape and severity ladder
  (`blocking` / `important` / `minor`) are committed by
  [`.claude/commands/critique.md`](../../.claude/commands/critique.md).
  This ADR does not amend the template, the severity ladder,
  or the command's stdout behavior.
- The two-round cap from
  [`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  step 7 is in force. A decision that needs more than two
  rounds is a signal that the decision is not yet ready, not
  a signal to add round 3.

---

## Decision

### Where — `studies/critiques/`

Preserved critique rounds live under
`studies/critiques/<date>-<slug>-critique-<N>.md`, where:

- `<date>` is the date the critique ran (matches the parent
  decision document's filename date in most cases; uses the
  critique-run date when the critique runs on a different day
  than the parent draft).
- `<slug>` matches the parent decision document's slug
  exactly (e.g., parent
  `2026-05-27-b2-35-critique-rounds-preservation.md` →
  critique
  `2026-05-27-b2-35-critique-rounds-preservation-critique-1.md`).
- `<N>` is the round number, 1-indexed. With the two-round
  cap in force, `<N>` is 1 or 2.

The `studies/critiques/` directory is created on first use;
it is absent from the repository until the first critique
under this contract runs. This is R8-consistent — the
`studies/` tree is scaffolding, present only where reasoning
has accumulated.

### When — separate commit, before the revision

The commit sequence is: draft commit → **critique commit** →
revision commit. The critique transcript is committed before
any revision to the parent document, so the revision diff
stays focused on changes the critique drove. The commit
message convention is `docs(decision): B<N>-<M> — critique
round <K> findings` — the same conventional-commit scope as
the parent document's commits, for continuity with existing
study commits.

### What — full stdout verbatim + operator-response trailer

Each preserved critique file contains:

1. The HTML path-header comment required by **R6**.
2. An H1 title: `# B<N>-<M> — Critique Round <K>`.
3. A `## Metadata` block with at minimum: target document
   path, round number, critique date, preservation status,
   and the closing commit hash.
4. A `## Critique Output` section containing the full
   `/critique` stdout output reproduced verbatim. Findings
   are placed inside fenced ```text blocks so the
   `[severity] R/P/AC label: section — change` template is
   preserved byte-for-byte and the sweep totals / operator
   notes carry through without reformatting.
5. A `## Operator Response` trailer with one entry per
   finding, recording its disposition as one of:
   - `applied as recommended` — the change landed in the
     revision as the finding described.
   - `applied with variation` — the change landed with
     operator-noted modification.
   - `deferred to Open Questions` — with the OQ bullet's
     text.
   - `deferred to a future round` — with a forward-pointer.
   - `accepted as-is` — with a one-line reason; typically
     for `minor` findings the operator chose not to act on.
   - `rejected` — with a one-line rationale.

The Operator Response trailer is what distinguishes a
preserved critique from a stripped-down findings dump: it
records the operator's judgement, not just the agent's
findings. This is what future critique passes read to
calibrate disposition. A preserved critique that omits the
trailer is contractually incomplete; the next adversarial
review may flag it.

### Retroactive — accept loss; catalog is the salvage

No reconstruction is attempted for critique rounds that ran
before this ADR's acceptance. The
[`.claude/skills/critique-anti-patterns/`](../../.claude/skills/critique-anti-patterns/)
corpus is the substantive salvage. Commit `926e3e5` remains
the bootstrap exemplar — its transcript lives in the PR
description for #10; whether to backfill that exemplar (or
any other pre-acceptance round) into `studies/critiques/`
is a **discretionary archival exercise**, not a gating
requirement.

The first critique round preserved under this contract is
the round-1 critique of the decision document that
introduced this ADR, captured retroactively in commit
`89f641c` as an explicit discretionary archival exercise
permitted by this section. It is the first entry under
`studies/critiques/` and serves as the exemplar of the
contract.

### Protocol edit — deferred

Two protocol files are affected by this ADR; only one is
edited:

- [`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  step 6 gains a sub-step specifying the capture target
  (`studies/critiques/<date>-<slug>-critique-<N>.md`), the
  capture mechanism (operator-side stdout redirection or
  copy-paste), the intermediate commit (per §"When" above),
  the operator-response trailer requirement (per §"What"
  above), and the skip-with-declaration seam (per §"Skip"
  below).
- [`.claude/commands/critique.md`](../../.claude/commands/critique.md)
  is **unchanged**. The command remains the
  adversarial-review-to-stdout surface defined today. This
  protects idempotency (re-running `/critique` does not
  overwrite preserved files), keeps the command's protocol
  footprint trivial, and avoids coupling the command to a
  path convention.

The `wave-1-session-loop.md` amendment is **deferred** to a
follow-up protocol-edit session. The amendment is bounded:
single playbook edit, no command rewrite, no new file
creation. Until the amendment lands, operators capture
critique stdout manually — the same path used for the
retroactive preservation that ships alongside this ADR's
acceptance.

### Skip — declared, not silent

A decision document's Metadata block gains an optional
`Critique rounds:` bullet listing each round's disposition.
Examples:

- `Critique rounds: 1 preserved
  (studies/critiques/<file>), 1 skipped (typo-only
  revision)` — explicit skip with reason.
- `Critique rounds: 2 preserved (<file-1>, <file-2>)` —
  full two-round cycle.
- `Critique rounds: 1 preserved (<file>)` — single round,
  no second round needed.

**Grammar:** rounds are listed in chronological order
(round 1 first); each entry uses one of two verbs —
`preserved (<filepath>)` or `skipped (<reason>)`; entries
are separated by commas within the bullet, which may span
multiple lines.

A study with no `Critique rounds:` bullet is treated by
the next adversarial review as having **undeclared
rounds**, and the review fires a `blocking` finding citing
violation of this declaration-presence contract. The seam
allows legitimate skips (typo-only revisions, operator-level
edits that don't warrant fresh critique) without permitting
hidden critique drops.

The bullet is **optional in studies that have not yet been
critiqued**; the contract takes effect on the first
post-acceptance critique run. A study can land at `draft`
status with no bullet; it cannot land at `resolved-study`
without one.

---

## Consequences

1. **Critique transcripts have a documented home.** Going
   forward, every critique round produced by `/critique` has
   a canonical preservation path under `studies/critiques/`.
   Future critique passes can read prior critiques verbatim
   to calibrate severity assignment and finding scope.

2. **The `critique-anti-patterns` skill gains a source
   pool.** New B-class anti-patterns can be added to
   [`.claude/skills/critique-anti-patterns/reference/anti-patterns-catalog.md`](../../.claude/skills/critique-anti-patterns/reference/anti-patterns-catalog.md)
   with citation to a specific preserved round, not just
   abstract reasoning.

3. **Commit cadence increases by up to N per decision.** A
   decision with two critique rounds now produces up to five
   commits (draft + critique-1 + revision-1 + critique-2 +
   revision-2) where the prior pattern was often two (draft
   + revised). Step 10 of `wave-1-session-loop.md` remains
   the single "log update + commit" step; the intermediate
   critique commits are operator-driven.

4. **Operator discipline becomes load-bearing.** The
   `/critique` command stays stdout-pure; the operator
   captures the output to the target file. The
   skip-with-declaration seam mitigates but does not
   eliminate the risk that an operator forgets to capture
   or to declare. The next adversarial review enforces
   declaration; capture itself is operator vigilance plus
   PR review.

5. **`studies/critiques/` appears organically.** The
   directory is absent from the repository until the first
   critique under this contract lands; thereafter it grows
   one file per round. This is R8-consistent — the
   `studies/` tree is scaffolding, present only where
   reasoning has accumulated.

6. **A follow-up protocol-edit session is registered.** The
   `wave-1-session-loop.md` step 6 amendment is gated on
   this ADR's acceptance and is bounded to a single
   playbook edit. The `/critique` command itself stays
   unchanged.

7. **The retroactive bootstrap landed alongside this ADR.**
   The decision document that introduced this ADR had its
   own round-1 critique preserved retroactively (commit
   `89f641c`). It is the first entry under
   `studies/critiques/` and serves as the contract's
   exemplar for the shape preserved files take.

8. **B2-35 closes.** The decision-log B2-35 row moves to
   `resolved-adr`. No new B2 row is registered for the
   protocol-edit follow-up at ADR acceptance time — the
   amendment is small enough to land as a single
   operator-discretion edit. A B2 row may be added later
   if the amendment surfaces protocol questions that
   require their own session.

9. **No engine, rules, tools, or deploy workspace is
   touched.** The full impact surface is `studies/` (where
   preserved critiques accumulate), `docs/adr/` (this
   file), and eventually
   `.claude/playbooks/wave-1-session-loop.md`. No code
   ships from this ADR.

10. **Three deferred items are flagged out-of-scope:**
    backfill of pre-acceptance rounds beyond the bootstrap
    exemplar (discretionary; not gated); `/critique` exit-
    code semantics for `blocking` findings (potential B3
    row if CI auditability of skip decisions becomes a
    concern); a linter for the slug-derivation convention
    (operator vigilance and PR review are deemed adequate
    at the platform's current cadence).

---

## Notes

- The contract treats critique transcripts as **reasoning
  artifacts**, not as published documentation. Per R8,
  `studies/critiques/` lives under `studies/`, not under
  `docs/`; the preserved files are not part of the
  repository's published surface.
- The Operator Response trailer's disposition vocabulary
  (`applied as recommended` / `applied with variation` /
  `deferred to Open Questions` / `deferred to a future
  round` / `accepted as-is` / `rejected`) is the
  recommended set, not a closed enum. Operators may add
  disposition labels as needed (e.g., `partially applied;
  remainder deferred`); future critiques will normalise
  vocabulary if drift becomes a concern.
- A study can have **zero preserved rounds at
  `resolved-study`** by declaring `Critique rounds: 0
  (no critique required; <reason>)` in its Metadata. This
  is an edge case — most decision documents that justify a
  study also justify a critique — but the seam exists for
  documents that are functionally pass-through (e.g.,
  re-statements of prior decisions, or one-line
  amendments).
- The conventional-commit scope `docs(decision):` is reused
  for critique commits to avoid scope proliferation. A
  future amendment may introduce `docs(critique):` if
  filtering critique commits from decision commits becomes
  operationally useful; until then, the same scope keeps
  `git log --grep` simpler.
- The two-round cap from `wave-1-session-loop.md` step 7
  remains in force. This ADR does not amend the cap.
