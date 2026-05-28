<!-- path: studies/critiques/2026-05-27-b3-evolutionary-launch-critique-1.md -->

# B3-launch — Critique Round 1

## Metadata

- **Target study:** [`../decisions/2026-05-27-b3-evolutionary-launch.md`](../decisions/2026-05-27-b3-evolutionary-launch.md)
- **Round:** 1 of (cap 2 per `wave-1-session-loop.md` step 7)
- **Date:** 2026-05-27
- **`/critique` invocation:** 2026-05-27, after the initial launch
  draft (study file uncommitted at critique time; close commit
  follows this preservation commit)
- **Preservation status:** preserved under ADR-0048 protocol (first
  formal instance)
- **Closing commit:** TBD — the next commit in `git log` immediately
  following this preservation commit; commit message
  `docs(decision): B3-launch — round 1 cleared, study resolved`
- **Note:** This is the **first preservation under the formal
  ADR-0048 protocol** since the playbook amendment landed in commit
  `9578ec0` (`docs(playbook): amend wave-1-session-loop.md step 6
  per ADR-0048`). The prior `studies/critiques/` entry —
  [`2026-05-27-b2-35-critique-rounds-preservation-critique-1.md`](./2026-05-27-b2-35-critique-rounds-preservation-critique-1.md)
  — was retroactively preserved as the bootstrap exemplar of the
  contract itself, before ADR-0048 promotion (`9392f21`) and before
  the playbook amendment. B3-launch is the first **prospective**
  preservation under the protocol's normal operation.

## Critique Output

> Reproduced verbatim from the `/critique` invocation. Findings use
> the `[severity] <R/P/AC label>: <section> — <change>.` template
> per `.claude/playbooks/feedback-protocol.md`. Markdown headings
> from the original output are flattened to bold severity prefixes
> with fenced `text` blocks (matching the B2-35 precedent) to avoid
> heading collisions inside this preservation file.

Re-grounded against `CLAUDE.md` §3 (R1–R8), §4 (P1–P6), and
`.claude/playbooks/acceptance-criteria.md` (AC-1–AC-10). Read the
target end-to-end.

**Blocking findings (2):**

```text
[blocking] R7: "Recommendation" §(a), §(b), "Consequences" C-B3.3,
"Open Questions" OQ-B3.2 — the Portuguese term `descartado` is used
as a load-bearing vocabulary item across the study's four-branch
eligibility criterion in an otherwise-English technical artifact
bound for promotion to ADR-0049. R7 says "ADRs, schemas, technical
READMEs, code comments, and contract documents are in English"; this
study has no language marker and is not an onboarding guide. Either
replace `descartado` consistently with an English equivalent (e.g.,
`discarded`, `out-of-scope`, or `rejected`) or — if the operator
intends it as a deliberate term of art — declare it as such in one
explicit gloss in §(a) and keep usage consistent thereafter.

[blocking] R2/R5: "Consequences" C-B3.4 (lines 379–384) — "The
Wave-S full-gate (all seven B0-S items at `resolved-adr`) remains a
prerequisite for record-mode B3 entries" introduces a new constraint
that is not committed by ADR-0020. ADR-0020 gates record-mode code
shipping on the partial gate (B0-S1/2/3); the full gate closes the
wave but is not stated to gate downstream evolutionary entries. This
is a new contribution proposed by this study; either label it
explicitly as "new contribution proposed here, requires review" per
R5, or remove the claim and let B3-N items adjudicate their own
Wave-S gate dependency at filing time.
```

**Important findings (4):**

```text
[important] R3: "Recommendation" §(b) Candidate C (lines 292–294) —
"Substrate evolution either lives inside an existing mode (and is
then a B2 or amendment for that mode's substrate ADR)" contradicts
§(a)'s temporal split. §(a) commits "B2 is pre-shipping against an
in-flight wave; B3 is post-shipping against a closed wave." After
Wave 3 / Wave-S close, a substrate change against ADR-0020 /
ADR-0023 cannot be B2 — it can only be amendment or a new wave.
Remove the "B2 or" option from Candidate C rationale; keep
"amendment or new Wave-S-style launch".

[important] AC-6: "Open Questions" OQ-B3.3 — the resolution marker
"Recommendation: forward-looking." does not match AC-6's requirement
that every item be "explicitly marked 'out-of-scope for current
cycle' with a one-line reason." A recommendation is not a defer-
marker. Reword to either "Out of scope for this study; default is
forward-looking." or "Defer to first retroactive-demand case;
default forward-looking." followed by the one-line reason (currently
the P3-demand-driven rationale is fine).

[important] AC-6 / status convention: "Metadata" Status (line 9) —
"resolved-study (B3, session 1; pending /critique)" is premature
against the Wave-S precedent, where "resolved-study" was only set
after critique rounds cleared. Per `06-decision-log.md` §Status
Vocabulary, a study with critique still pending is "in-progress".
Restate as "in-progress (B3, session 1; pending /critique)" and let
the operator promote the status to "resolved-study" after this
critique round clears.

[important] R2 / P3: "Recommendation" §(a) B3 condition (iv) (lines
249–256) — the "intent beyond additive-PR scope already authorized
by ADR-0022 §evolution" threshold is well-defined for the kind
family but undefined for the capability-mode and tooling families,
which have no analogous "additive-PR" pathway documented. A
contributor proposing a new lint cross-check or a new derived-
capability signal cannot tell from condition (iv) whether their
proposal crosses the intent threshold, because there is no PR-level
evolution clause to compare against. Add a parallel threshold for
capability-mode and tooling extensions (or generalize condition (iv)
explicitly to all three families).
```

**Minor findings (4):**

```text
[minor] R5/R8: "Recommendation" §(b) labelling rationale (lines
325–328) — the "labels C, E, F leave room for in-scope families
under labels A, B, D should the launch ADR want to enumerate them
symmetrically" aside is meta-commentary about the study's own
labelling scheme and will be confusing in the promoted ADR. Either
enumerate the in-scope families as A, B, D in §(b) so the symmetry
is concrete, or remove the rationale paragraph entirely — the C/E/F
labels can stand on their own.

[minor] P1 vs P3: "Decision Drivers" DD-B3.4 (lines 140–144) —
attributing the no-pre-enumeration rule to a P1 violation is a
stretch; pre-listing speculative B3-N items is not "rewriting"
capability, it is just speculation. Re-cite to P3 (B3-N items are
demand-driven) instead, which is the locked premise actually
violated by pre-enumeration.

[minor] AC-6 / OQ-B3.4: "Open Questions" OQ-B3.4 (lines 478–484) —
"the contributor cannot unilaterally skip the B3 path" overstates
the enforcement mechanism, since the only gate is the CODEOWNERS
reviewer's discretion on the PR. Soften to "the reviewer can hold
the PR until the B3-N study lands" or similar, to keep the claim
consistent with the actual gate.

[minor] R6/AC-2: "Context" (lines 112–117) — the closing paragraph
announces that B3 sits "as a parallel lane sitting alongside Wave 2,
Wave 3, and Wave-S in the decision log" but does not document why a
B-tier name (suggesting placement after B0/B1/B2) was chosen over a
Wave-style placement. A future reader will see the structural
inconsistency (B-name, Wave-placement). One sentence in Context
naming the operator's framing choice and its rationale would
forestall the question.
```

---

**Sweep totals:** 2 blocking · 4 important · 4 minor.

**Closing summary (verbatim from the `/critique` output):**

- 2 blocking — R7 (Portuguese term in a technical artifact) and
  R2/R5 (Wave-S full-gate-as-record-mode-B3-prerequisite introduced
  without "new contribution" marker).
- 4 important — R3 §(b) internal contradiction; AC-6 OQ-B3.3 marker
  phrasing; AC-6 Metadata Status premature; condition (iv)
  threshold undefined for mode/tooling families.
- 4 minor — §(b) labelling aside; DD-B3.4 P1-vs-P3 attribution;
  OQ-B3.4 enforcement overreach; Context placement-rationale gap.

The study was not ready to move to `resolved-study` until the two
blocking findings were resolved (or rebutted explicitly per
`acceptance-criteria.md` AC-9 note). The next section records the
operator's disposition.

## Operator Response

**Strategy.** All 10 findings applied in one revision round; no
deferrals, no rejections. Two findings (B2 and F6) involved the
operator choosing between alternatives that the critique offered;
rationale paths are documented in the Variations block. The
Metadata Status flip from `pending /critique` to `round 1 cleared`
is folded into F5 per the operator's "After all fixes" close
directive.

| ID  | Finding (short)                                          | Disposition                                      |
|-----|----------------------------------------------------------|--------------------------------------------------|
| B1  | R7 Portuguese term `descartado`                          | **applied as recommended** — operator chose `rejected` of the three suggested alternatives (`discarded` / `out-of-scope` / `rejected`); no gloss-as-term-of-art declaration retained |
| B2  | Wave-S full-gate claim not anchored to ADR-0020          | **applied with variation** — operator chose critique's "remove the claim" alternative over "label as new contribution per R5"; see Variations below |
| F3  | §(b) Candidate C contradiction ("B2 or" prefix)          | **applied as recommended** |
| F4  | OQ-B3.3 marker phrasing (AC-6)                           | **applied as recommended** — exact replacement `Out of scope for this study; default is forward-looking per P3 demand-driven posture.` |
| F5  | Metadata Status premature                                | **applied with variation** — operator skipped the intermediate `in-progress` state and moved directly to `resolved-study (B3, session 1; one critique round; round 1 cleared with no blocking findings)`; see Variations below |
| F6  | §(a) condition (iv) threshold undefined for mode/tooling | **applied with variation** — operator chose critique's "generalize" alternative over "add a parallel threshold for each family"; see Variations below |
| M1  | §(b) labelling-rationale paragraph removed               | **applied as recommended** |
| M2  | DD-B3.4 attribution P1 → P3                              | **applied as recommended** |
| M3  | OQ-B3.4 enforcement softened                             | **applied as recommended** — exact replacement `reviewer can hold the PR until a B3-N study lands` |
| M4  | Context: B-tier name + Wave-style placement rationale    | **applied with minor variation** — critique suggested "one sentence"; operator landed two sentences (B-tier name signal + Wave-style placement signal) to give each signal a complete clause |

**Variations of note:**

- **B2 — claim removal vs labelling.** The critique offered two
  alternatives: (1) label the Wave-S-full-gate-as-record-mode-B3-
  prerequisite claim as "new contribution proposed here, requires
  review" per R5, retaining the claim with disclaimer; or
  (2) remove the claim entirely. Operator chose (2). Rationale:
  option (1) would have committed the platform to enforcing the
  Wave-S full-gate as a B3-N filing prerequisite, even with the
  "requires review" hedge — and that enforcement has no ADR-0020
  anchorage. Removing the claim delegates the question to each
  B3-N item's own filing-time gate analysis, consistent with P3
  (demand-driven; rows born on demand), and avoids committing the
  platform to an unanchored constraint at launch time.

- **F5 — intermediate state skipped.** The critique's strict
  reading would have had the document transition through
  `in-progress (B3, session 1; pending /critique)` before reaching
  `resolved-study (B3, session 1; one critique round; round 1
  cleared with no blocking findings)`. Operator collapsed the
  transition because the intermediate state never persisted in any
  commit — the round opened, fired, and closed in a single
  session. The final status string accurately reflects the file
  at Commit-2 (close) time.

- **F6 — generalize vs parallel thresholds.** The critique offered
  two alternatives: "add a parallel threshold for capability-mode
  and tooling extensions, **or generalize condition (iv) explicitly
  to all three families**". Operator chose the second alternative.
  Rationale: parallel thresholds would have introduced three
  family-specific clauses that must be kept in sync as the platform
  evolves; a single generalized "additive-maintenance threshold"
  with parenthetical pointers (`e.g., ADR-0022 §evolution for kind
  families; analogous evolution clauses for capability modes and
  tooling`) covers all three families today without committing the
  document to maintain three threshold definitions.

**No findings rejected.** **No findings deferred to a future round.**
All `blocking`, `important`, and `minor` findings closed in this
single revision pass; round 2 (cap per `wave-1-session-loop.md`
step 7) is not exercised.

Per the SD-6 declaration contract committed by ADR-0048, future
critique rounds run *after* the B3-launch study reaches
`resolved-study` would add a `Critique rounds:` Metadata bullet to
the study referencing preservation files. The launch study's
Metadata does not currently carry that bullet — the contract takes
effect prospectively from B3-launch's `resolved-study` close
forward, and no post-close round is currently anticipated.
