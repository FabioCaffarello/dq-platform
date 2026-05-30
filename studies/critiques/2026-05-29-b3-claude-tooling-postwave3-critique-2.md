<!-- path: studies/critiques/2026-05-29-b3-claude-tooling-postwave3-critique-2.md -->

# B3 — Claude Tooling Post-Wave-3 — Critique Round 2

## Metadata

- **Target study:** [`../decisions/2026-05-29-b3-claude-tooling-postwave3.md`](../decisions/2026-05-29-b3-claude-tooling-postwave3.md)
- **Round:** 2 of (cap 2 per `wave-1-session-loop.md` step 7).
- **Date:** 2026-05-29.
- **`/critique` invocation:** 2026-05-29, after the round-1 revision
  pass (+154 lines, 17 edits, 7 sections; round-1 disposition
  table fully applied per the operator's round-1 meta-evaluation).
- **Preservation status:** preserved under ADR-0048 protocol.
- **Closing commit:** TBD — the close commit follows this
  preservation commit and the touch-up revision commit; commit
  message `docs(decision): B3 claude-tooling — round 2 cleared,
  study resolved`.

---

## Critique Output

> Reproduced verbatim from the round-2 `/critique` invocation.
> Structured findings are kept in their original prose form (no
> fenced code block) because the round-2 output mixed prose
> sub-headers (Varredura A / Varredura B / Round-2 D0 ruling /
> AC sweep / Summary) with the labeled findings — preserving the
> sub-header structure makes the trailer disposition unambiguous.

**Sweep coverage:** R1–R8, P1–P6, AC-1…AC-10, B1–B8 (substantive
anti-patterns), plus operator-flagged regression checks
(Decision Drivers numbering contiguity; Artifacts numbering
contiguity; vocabulary; D0 × Consequences coherence; D7
boundary).

### Varredura A — confirmation of round-1 dispositions

Confirming each of the 3 blockers + 5 importants from round 1:

- **B#1 branch-invention.** Cleared. `grep -E 'b3/<slug>|b2/<slug>'`
  returns zero hits across the entire document. Step 10 of
  Artifact #2 now reads "the slug provided by the operator for the
  session (provisional per the governance contract, pending the
  post-Wave-3 `CONTRIBUTING.md` extension in §Recommendation #6)". ✅
- **B#2 OQ-3 → D0.** Cleared. `grep OQ-3` returns zero. D0 stands
  at §Decision Drivers (line 222) as a precondition, not a
  question — the body explicitly says "D0 is a precondition the
  study acknowledges must clear round 1; it is not an Open Question
  because the entire Recommendation is conditional on it."
  Cross-referenced consistently in 5 places (Metadata line 49,
  D0 body line 222–239, Recommendation lead-in line 565,
  Consequences point 8 line 857–868). ✅
- **B#3 CONTRIBUTING upstream authority.** Cleared with variation.
  D7 at line 282 + Artifact #6 at line 752 + three deference
  clauses in Artifacts #1/#2/#3 (lines 597, 626, 634, 660). The
  D7 scope boundary holds: Artifact #6 explicitly says "The
  Wave-3 section text is **not** rewritten (R4 — one topic per
  session); the extension adds a new sibling section beside it
  (suggested heading: *Post-Wave-3 evolutionary lane*)." ✅
- **I#1 force-push out of G3.** Cleared. JSON deny block contains
  only three entries (`gh pr merge`, `git commit *--no-verify*`,
  `git push *--no-verify*`); force-push moved to §Open Questions
  OQ-G3.1 at line 916. ✅
- **I#2 eligibility-step marker.** Cleared. Artifact #2 step 2
  carries the explicit marker at line 619: "**The
  eligibility-check step is new contribution proposed here,
  requires review** (R5)". ✅
- **I#3 deny-syntax.** **NOT FULLY CLEARED — see important
  finding #1 below.** The justification text introduced a new
  self-contradiction that wasn't present in round 1.
- **I#4 skill-trigger.** Cleared. Artifact #1 now uses verbatim
  trigger phrases ("create a branch", "git switch",
  "git checkout -b", "first commit", "open a PR",
  "gh pr create", "merge the PR") and explicitly disclaims a
  "session open" trigger; new-contribution marker at line 588. ✅
- **I#5 ✅ table inconsistency.** Cleared. Eligibility check
  condition 2 (line 47–51) now reads "Pending — promoted to a
  §Decision Drivers D0 precondition" instead of the mixed
  "✅ (subject to reviewer confirmation)". ✅

Eight of eight round-1 items confirmed; one (I#3) carries a
residual issue noted below as a fresh important.

### Varredura B — regression findings

#### Important

- **[important] AC-4 / B4 citation drift: "§Recommendation —
  Artifact #4, deny-syntax justification (lines 720–724)"** —
  the text states "the current `settings.local.json` only uses
  the suffix shape `Bash(<prefix> *)`" and then claims "the
  per-call-site shape is conservative and matches a pattern the
  harness is known to honor." But the new deny
  `Bash(git commit *--no-verify*)` has `*` in the middle of the
  argument (`git commit ` prefix + `*--no-verify*` middle/suffix),
  which is *not* the `Bash(<prefix> *)` shape — that shape has
  `*` only after the final space, never embedded. The
  justification contradicts the precedent it cites. Two fixes
  equally acceptable:
  - (a) mark the per-call-site deny as **"new contribution
    proposed here, requires review"** (R5) pending harness-docs
    verification — the conservative move; or
  - (b) rewrite as `Bash(git commit --no-verify*)` and
    `Bash(git push --no-verify*)` (pure suffix; catches
    `--no-verify` only as the last argument; misses cases where
    `--no-verify` precedes other args, but stays inside the
    precedent).
  Recommend (a) — the operator's prior disposition on I#3 was
  "pending harness-docs check" and the conservative marker fits.

- **[important] AC-9 / temporal drift: "§Metadata line 50, D0
  line 236, Recommendation lead-in line 567, Consequences point
  8 line 863 — 'in round 1' phrasing"** — four places frame the
  Recommendation as conditional on D0 clearing "in round 1".
  Round 1 already concluded; round 2 is in flight now; the
  substantive D0 ruling (does `.claude/` qualify as "adjacent
  tooling" under ADR-0049 §Per-family scope?) was never emitted
  by either round (round 1 flagged it as borderline +
  new-contribution; round 2 — this round — issues no blocking
  against the reading, see §Round-2 D0 ruling below). The "in
  round 1" reference creates ambiguity about whether the ruling
  has happened. Replace with "in `/critique`" (round-agnostic)
  or "before the study moves to `resolved-study`" in all four
  call sites. Mechanical edit; resolves the ambiguity without
  changing semantics.

#### Minor

- **[minor] B7 vocab residue: "§Context line 125 — 'operating
  cadence is now post-Wave-3 evolutionary work'"** — the
  canonical term "post-Wave-3 evolutionary lane" appears 5×
  consistently across the document; line 125 still says
  "post-Wave-3 evolutionary work" referring to the same noun
  (the lane the project is currently in). Other compound nouns
  (loop, playbook, session, section, tooling, demand-driven,
  Make) are distinct concepts and don't drift. Line 125 is the
  only residual ambiguity. Either tighten to "post-Wave-3
  evolutionary lane" for full consistency or keep the "work"
  framing if the distinction is intentional (lane = the
  structure; work = the items flowing through it) and add a
  one-line note clarifying.

- **[minor] AC-2 / lead-in count alignment: "§Recommendation
  lead-in line 567"** — says "three new files (skill, playbook,
  command), one settings hardening, and two edits". §Consequences
  point 1 line 799 says "Six small artifacts replace five
  ad-hoc disciplines." The arithmetic matches (3 + 1 + 2 = 6),
  but the Recommendation lead-in does not count the disclaimer
  item (#7) and Consequences does not name the settings
  hardening explicitly. Both are correct under their own
  framings, but a reader cross-referencing the two will pause.
  Pure polish; the alignment isn't broken, just non-obvious.

### Round-2 D0 ruling (substantive, not a finding)

This is the load-bearing review point operator-flagged in the
round-1 disposition table (B#2). I rule explicitly so the study
can advance:

> **D0 holds for the purposes of round 2.** The "Tooling
> extensions" family under ADR-0049 §Per-family scope reads
> "additions to `tools/lint/`, the manifest publisher, the
> dry-run runner, the engine dispatcher, **and adjacent tooling**
> that extend contract coverage without changing the contract
> shape." `.claude/` is adjacent tooling: it governs the
> contributor process that produces every artifact (studies,
> ADRs, scaffolds, PRs). The work proposed extends governance
> coverage to the post-Wave-3 evolutionary lane without reshaping
> the draft → critique → accept → promote contract. The
> borderline-interpretation marker (D0, line 229) is honest
> about the canonical examples being platform-runtime tools, but
> the "adjacent tooling" clause is explicit enough to admit
> agent-harness extensions. No blocking finding emitted against
> the reading.

D0 cleared; §Recommendation stands.

> **Operator note on this ruling:** see §Operator Response
> §"D0 ratification" below. The round-2 ruling is an *input* to
> the operator's eligibility decision, not the decision itself —
> the operator ratifies D0 explicitly to avoid the
> author-equals-reviewer circularity that round-1 B#2 flagged.

### AC-1 … AC-10 sweep

- **AC-1** path header — line 1
  `<!-- path: studies/decisions/2026-05-29-b3-claude-tooling-postwave3.md -->`. ✅
- **AC-2** sections in order — Metadata, Context, Decision
  Drivers, Considered Options, Recommendation, Consequences,
  Open Questions, Promotion target — confirmed via
  `grep -n "^## "`. ✅
- **AC-3** ≥2 options per gap — G1 (3), G2 (3), G3 (3), G4 (3),
  G5 (3). ✅
- **AC-4** Recommendation grounded or marked new-contribution —
  4 new-contribution markers present (D0 reading; skill trigger;
  eligibility-check step; CONTRIBUTING slug forms); deny-syntax
  claim falls short, see important finding #1.
- **AC-5** no vendor / sibling-team / prior-art names as
  justification. ✅
- **AC-6** Open Questions — OQ-1, OQ-2, OQ-G3.1, all marked
  "out-of-scope for current cycle" with reasons. ✅
- **AC-7** Promotion target concrete —
  `docs/adr/0051-claude-tooling-postwave3.md` with ADR-0020
  caveat verbatim per `adr-writing` A8. ✅
- **AC-8** ≥1 critique round — round 1 done; this is round 2. ✅
- **AC-9** blocking findings from prior round addressed — 3 of
  3 round-1 blockers resolved (B#1, B#2, B#3 cleared above). ✅
- **AC-10** decision log updated — pending close commit; not
  assessable from the artifact alone. N/A this round.

### Sweep totals

| Severity | Count |
|---|---|
| `blocking` | **0** |
| `important` | 2 |
| `minor` | 2 |

Per `.claude/playbooks/wave-1-session-loop.md` step 7, the study
has now passed the **maximum two critique-revise rounds** budget
after this round closes. If the two importants land in the next
pass, the study can move to `resolved-study`; if any cannot be
resolved, document the rebuttal in the study per
`acceptance-criteria.md` Note on AC-9.

---

## Operator Response

### D0 ratification (operator decision, not a finding)

Per the operator's round-2 meta-evaluation §3 ("O ruling D0 —
o ponto que exige SEU julgamento"), the D0 substantive ruling
emitted by the round-2 critique is treated as **input** to the
operator's eligibility decision, not as the decision itself. The
operator ratifies D0 explicitly to convert the circular
"author-equals-reviewer" self-approval into a traceable human
decision (the structural issue round-1 B#2 surfaced):

> **D0 eligibility ratificada pelo operador, com base no ruling
> do round 2 como input; a leitura expansiva de 'adjacent
> tooling' para agent-harness tooling é new contribution e fica
> registrada como tal no ADR-0051.**

Two consequences this ratification commits:

1. **The expansive reading of "adjacent tooling" carries forward
   to ADR-0051 as new contribution.** The ADR-0051 promotion of
   this study must mark the agent-harness-as-adjacent-tooling
   interpretation explicitly (R5 / `adr-writing` A7
   new-contribution marker), not bury it as if ADR-0049
   §Per-family scope already committed it. Future B3-N tooling
   sessions that target agent-harness extensions can then cite
   ADR-0051 as precedent.

2. **The author-equals-reviewer circularity is recognized as a
   structural risk of single-agent sessions.** Future eligibility
   rulings under ADR-0049 §(a) for borderline B3-N items should
   be ratified by the operator explicitly, not absorbed into the
   `/critique` output silently. This is registered here as
   operator-side discipline; no playbook edit is committed by
   this critique (R4 — out of scope for the B3 claude-tooling
   study's own touch-up).

### Disposition of round-2 findings

In critique-output order:

1. **`[important] AC-4 / B4 citation drift` — deny-syntax
   self-contradiction (Artifact #4 lines 720–724).**
   *Applied with variation, combining options (a) and (b).* The
   touch-up does both: (i) rewrites the deny entries from
   `Bash(git commit *--no-verify*)` /
   `Bash(git push *--no-verify*)` to the pure-suffix shape
   `Bash(git commit --no-verify*)` /
   `Bash(git push --no-verify*)` matching the precedent
   `Bash(<prefix> *)`; AND (ii) marks the per-call-site deny
   syntax explicitly as "new contribution proposed here,
   requires review" pending harness-docs verification. The
   suffix shape catches `--no-verify` only as the last
   argument; the new-contribution marker acknowledges the
   coverage is partial until verified. The operator's
   round-2 meta-evaluation called for (a)+(b) together; this
   matches.

2. **`[important] AC-9` — "in round 1" temporal drift (4
   sites).** *Applied as recommended.* All four occurrences
   (Metadata line 50, D0 line 236, Recommendation lead-in line
   567, Consequences point 8 line 863) rewritten to
   "in `/critique`" (round-agnostic). Mechanical edit; semantics
   preserved.

3. **`[minor] B7` — vocab residue line 125 ("evolutionary work"
   vs "lane").** *Applied as recommended.* Standardized on
   "post-Wave-3 evolutionary lane" per operator's round-2
   meta-evaluation §2 — the lane/work distinction was not
   intentional and is dropped.

4. **`[minor] AC-2` — lead-in count alignment (Recommendation
   line 567 vs Consequences point 1 line 799).** *Applied as
   recommended.* §Consequences point 1 reworded to explicitly
   enumerate the same six items the Recommendation lead-in
   counts, so a cross-referencing reader sees identical
   accounting.

### Round budget

The two critique-revise rounds budgeted by
`.claude/playbooks/wave-1-session-loop.md` step 7 are now spent.
The touch-up pass (this Operator Response) is **mechanical
correction of the four named findings, not a third critique
round**. After the touch-up commit lands, the study moves to
`resolved-study` per `acceptance-criteria.md` AC-1…AC-10 without
a new critique invocation. Future round-3-or-beyond critique on
this study is out-of-scope per the budget; remaining doubts
become Open Questions under
`acceptance-criteria.md` Note on AC-9.
