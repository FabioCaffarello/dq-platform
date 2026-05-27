<!-- path: studies/critiques/2026-05-27-b2-35-critique-rounds-preservation-critique-1.md -->

# B2-35 — Critique Round 1

## Metadata

- **Target study:** [`../decisions/2026-05-27-b2-35-critique-rounds-preservation.md`](../decisions/2026-05-27-b2-35-critique-rounds-preservation.md)
- **Round:** 1 of (cap 2 per `wave-1-session-loop.md` step 7)
- **Date:** 2026-05-27
- **`/critique` invocation:** 2026-05-27, after the study's draft commit `1d37a35`
- **Preservation status:** retroactively preserved (manual capture before ADR-0048 promotion)
- **Closing commit:** `519f020` — `docs(decision): close B2-35 critique-rounds preservation — round 1 cleared, no blocking findings`
- **Note:** This is the **first entry under `studies/critiques/`** in
  the project's history. Before B2-35 the only preserved critique
  round was commit `926e3e5` (W3-P4e HTTP trigger handler against
  ADR-0014), captured informally in the PR description for #10
  rather than as a standalone artifact. This file establishes the
  pattern the resolved study commits, applied retroactively to its
  own first round.

## Critique Output

> Reproduced verbatim from the `/critique` invocation. The original
> output structured findings as fenced code blocks rather than a
> table; that structure is preserved here.

**No `blocking` findings.** The study passes R1 (no production
code), R2 (no invented requirements; commits 926e3e5/d2fd4a5/060dd10
verified, skill-file count of 3 verified, two-round cap citation
from `wave-1-session-loop.md` step 7 verified), R5 (no
vendor/prior-art; "MADR" is established repo vocabulary per
`CLAUDE.md`, `README.md`, and ten existing studies), R6 (line 1
path-header present), R8 (`studies/critiques/` lives under
`studies/`). AC-1, AC-2, AC-3, AC-5, AC-6, AC-7 all pass.
AC-8/AC-9/AC-10 are post-critique gates and not applicable to this
round.

---

**Important findings (5):**

```text
[important] R5: "Recommendation §### D6" line 192 — "treated as an R3/R5-class
violation by the next critique pass" misapplies the cited rules. R3 (revisit
settled decisions) and R5 (cite-only-repo) don't cover undeclared critique
skips; the actual violation is of D6 in this study. Rewrite to cite the new
rule this study introduces, or label as "violates the D6 declaration
contract".

[important] R5: "Considered Options §Option 3 Cons" line 270-272 — cites
"anti-pattern B7 'strawman options'" but B7 in
`.claude/skills/critique-anti-patterns/SKILL.md` is "Vocabulary drift cross-
section". Strawman options is B6. Swap the number.

[important] R5: "Recommendation §### D6" lines 413-414 — "fire a blocking
finding citing R6-class visibility failure" invents an "R6-class" category.
R6 is strictly the path-header HTML comment rule; there is no generic
visibility class. Cite the new declaration-presence rule this study commits,
or remove the R6 reference and name the failure directly.

[important] AC-2: "Decision Drivers" lines 124-155 vs "Recommendation" lines
289-415 — the document uses "D1..D7" for Decision Drivers and "D1..D6" for
sub-decisions; Considered Options cons reference "D6 demands" (line 235) and
"D5-violating" (line 273) with no disambiguation. Rename Decision Drivers to
`DD-1..DD-7` (or drop the prefix entirely) so the D1..D6 sub-decision space
is unambiguous everywhere.

[important] AC-6: "Open Questions §OQ-1" lines 518-524 — deferred to "the
protocol-edit session," but that session's scope (per §### D5, lines 372-393)
is the `wave-1-session-loop.md` step-6 amendment, not artifact backfill. The
926e3e5/d2fd4a5 backfill has no natural home in that session. Either resolve
OQ-1 here, or defer to a named owner (e.g., "future operator-discretion
task; not gated on protocol-edit session").
```

**Minor findings (5):**

```text
[minor] AC-2: "Recommendation §Why this does not reopen R8" line 431 — R8 is
a hard rule (CLAUDE.md §3), not an architectural decision that can be
"reopened." The sibling "Why this does not reopen ADR-0014" subsection is
correctly framed (ADR-0014 is a decision); R8 is not. Rename to "Why this
respects R8" or fold the content into Context.

[minor] AC-4: "Open Questions §OQ-3" lines 530-533 — cites
`acceptance-criteria.md` as enforcing "path conventions for studies," but
AC-1 enforces R6 path-header presence, not filename/path conventions. Rephrase
to cite the actual mechanism (or drop the analogy).

[minor] AC-2: "Decision Drivers §D7" lines 151-155 — D7 commits an answer
("Skip-with-declaration in the study's Metadata is the seam this driver
names") inside an evaluation-criterion section, conflating driver with
sub-decision position. Strip the seam-naming sentence; let `### D6` in
Recommendation be the only place that commits the seam.

[minor] AC-2: "Recommendation §### D2" line 323 — the commit-message example
`docs(critique): B<N>-<M> — round <K> findings` introduces a new
conventional-commit scope without discussion. Either justify the new scope
in one sentence or reuse `docs(decision):` for continuity with existing
study commits.

[minor] AC-2: "Recommendation §### D6" lines 397-415 — the `Critique rounds:`
Metadata bullet grammar is underspecified (entry separator, allowed verbs
beyond "preserved"/"skipped", ordering when multiple rounds exist) for a
contract whose absence fires a `blocking` finding on the next pass. Add 2-3
lines of normative grammar so the next critique can deterministically detect
malformed declarations.
```

---

**Sweep totals:** 0 blocking · 5 important · 5 minor.

**Notes for the operator:**

- The three citation-drift findings (R3/R5/R6 misapplications) are
  the same defect type (B4 from the anti-patterns catalog) in three
  places — fixable as one revision pass.
- The Decision Drivers / Sub-decisions D-numbering collision is the
  highest-leverage `important` finding; fixing it disambiguates
  several other places in the document.
- OQ-1's "deferred to protocol-edit session" is the only `important`
  finding that requires a substantive decision (resolve here vs
  reassign owner), not just a textual edit.
- No findings touch the recommended Option 1 itself or any of the
  D1–D6 substantive positions — the recommendation survives this
  critique pass intact.

## Operator Response

**Strategy.** All 10 findings applied in one revision round; no
deferrals, no rejections. Disposition table covers F1–F5 + M1–M5
(the 10 critique findings) plus 2 closing-mechanics edits required
by AC-10 / Status flip. Total 12 edits in the close commit
(`519f020`).

| ID  | Finding (short)                                          | Disposition                                      |
|-----|----------------------------------------------------------|--------------------------------------------------|
| F1  | R3/R5-class → SD-6 declaration contract                  | **applied as recommended**                       |
| F2  | anti-pattern B7 → B6 (strawman options)                  | **applied as recommended**                       |
| F3  | R6-class visibility → SD-6 declaration-presence contract | **applied as recommended** (minor phrasing edit) |
| F4  | D-numbering collision                                    | **applied with variation** — operator chose Option C (both DD- and SD- prefixes) after the literal brief was shown to re-create the drift; AskUserQuestion surfaced the discrepancy and got explicit approval before applying |
| F5  | OQ-1 deferred to wrong session                           | **applied as recommended** (user-supplied rewrite text used verbatim; `D4` → `SD-4` typographic update for consistency with the F4 SD-prefix policy) |
| M1  | "Why this respects R8" rename                            | **applied as recommended**                       |
| M2  | OQ-3 cites AC-1's actual mechanism                       | **applied as recommended**                       |
| M3  | DD-7 seam-naming sentence stripped                       | **applied as recommended**                       |
| M4  | `docs(critique):` → `docs(decision):`                    | **applied as recommended** (operator confirmed via AskUserQuestion) |
| M5  | 3-line normative grammar in SD-6                         | **applied as recommended**                       |
| —   | Metadata Status flip (closing mechanics)                 | **AC compliance** — not a finding; required for resolved-study |
| —   | AC-10 decision-log row flip (closing mechanics)          | **AC compliance** — required by acceptance-criteria.md AC-10; operator-confirmed inclusion in this pass |

**Variations of note:**

- **F4 scope expansion.** The brief said "rename sub-decisions D1..D6 → SD-1..SD-6"
  but inspection showed several Cons references in Options 2/3 actually
  pointed to Decision Drivers (D5/D6/D7), so a literal rename would
  re-create the very drift F4 was meant to fix. Three resolution paths
  (A literal / B SD-only with prose disambiguation / C both prefixes)
  were presented; operator chose **C**. Result: all D-references are
  unambiguous by prefix; 15 DD-N references and 35 SD-N references
  in the resolved study; zero bare `D1..D7` references remain.
- **F3 phrasing.** Brief gave exact replacement text "violates the SD6
  declaration-presence contract introduced by this study"; applied as
  "violation of the SD-6 declaration-presence contract introduced by
  this study" (added hyphen for consistency; gerund→noun for sentence
  flow). Substance identical.
- **F5 typography.** Brief's rewrite said `D4`; applied as `SD-4` to
  match the SD-prefix policy enforced by F4.

**No findings rejected.** **No findings deferred to a future round.**
All `important` and `minor` findings closed in this revision pass,
so the second round is not exercised; the `Critique rounds:` Metadata
bullet (per SD-6) reads:

```
Critique rounds: 1 preserved (studies/critiques/2026-05-27-b2-35-critique-rounds-preservation-critique-1.md)
```

(Note: this bullet is not yet present in the resolved study's
Metadata block — the SD-6 contract is committed as of `resolved-study`
but the bullet itself becomes the operator's responsibility on the
next post-B2-35 critique. This retroactive preservation does not
amend the study to add the bullet; the contract takes effect for
critique rounds run after B2-35 reaches `resolved-study`.)
