<!-- path: studies/critiques/2026-05-30-adr-0057-single-user-codeowners-amendment-critique-1.md -->

# Critique — `docs/adr/0057-single-user-codeowners-amendment.md` — round 1

## Blocking findings

**None.** R6 path header present. A1 four-section structure
(Context / Decision / Consequences / Notes). A2 metadata correct
(Status `accepted (amends ADR-0015)`, ISO date). A4 standalone-
amendment idiom correctly applied (single-amender format per
`adr-writing` A4 exemplar). A7 new-contribution markers correctly
placed in §Notes for the (a)–(e) trade-offs, the OQ-1 deny-block
carry-forward, and the branch-protection precondition audit
surface. R5 commodity exemptions correctly applied (GitHub is
environment; the operator's actual handle is repo deployment-
substrate owner, not external prior-art). R8 forward-only (no
path-link back-references to `studies/`). Decision is five
clauses; Consequences twelve; Notes four paragraphs.

## Important findings

- **[important] R8 / A6: "Notes → Deny-block independence
  carry-forward" — reference to `memory
  harness-deny-blocks-independent-of-review-model.md` is a
  dangling pointer for repository readers.** The named memory
  file lives in the operator's global
  `.claude/projects/-Volumes-OWC-Express-1M2-Develop-dq-platform/memory/`
  directory — outside the repository. Future readers landing
  on `docs/adr/0057-…md` cold (e.g., a contributor cloning the
  repo without the operator's harness setup) cannot resolve
  the reference. R8 implies published ADRs are self-contained
  against the repository surface. Two fixes available: (a)
  drop the memory-file name and inline the principle into the
  Notes paragraph; (b) replace the memory-file pointer with a
  citation to the in-repo source-of-truth (PR #117 + the
  originating study's OQ-1 disposition). Option (a) is cleaner
  — the principle stands on its own merits without needing a
  forensic pointer.

- **[important] A4 / AC-2: ADR-0015's Status-line update uses
  "amended by"; the ADR-0010 precedent uses "amended in part
  by".** ADR-0010 §Status: *"accepted; **amended in part by
  ADR-0017** (object-store CAS row revised…); **amended in
  part by [ADR-0028](./0028-kafka-substrate-row.md)** (capability
  matrix extended…)"*. The "in part" qualifier is load-bearing
  — ADR-0017 amended one row of ADR-0010, not the whole ADR;
  ADR-0028 added rows without touching others. ADR-0057 has
  the same partial-amendment shape against ADR-0015: §2/§4/
  §Consequences #1+#5 amended; §3, Consequences #2/3/4/6/7/8/
  9/10/11, and §Notes preserved verbatim. The ADR-0015 Status
  update committed in this PR (`amended by ADR-0057`) elides
  the partial qualifier. Update ADR-0015's Status line to
  `accepted; **amended in part by [ADR-0057](./0057-single-user-codeowners-amendment.md)**
  (group inventory collapses…)` to match the more rigorous
  ADR-0010 precedent. The narrower phrasing matches reality
  (the path-rule table from ADR-0015 §3 is preserved verbatim;
  only §2/§4/§Consequences-#1+#5 amend).

## Minor findings

- **[minor] AC-2 / A2: "Notes → Critique rounds" — forward-
  looking phrasing at write-time will be stale after this
  critique applies.** Update post-application to record the
  round-1 disposition explicitly (e.g., `0 blocking / 2
  important / 5 minor; 2 important applied`).

- **[minor] R8: "Decision → Clause 4 — originating amendment
  study reference"** is borderline R8 — the phrase
  *"The originating amendment study's Consequence #4 framed
  this as 'the lint-time cross-check relaxes' — that framing
  was over-broad"* mentions the study contextually but does
  not back-link via a path. Tighten to *"The amendment-study
  reading that the lint-time cross-check 'relaxes' was
  over-broad — ADR-0037 §'Parser scope' item 4 already
  commits the @<user> reviewer-token shape, so no relaxation
  was needed"* — removes the contextual back-pointer entirely
  and surfaces the load-bearing fact (ADR-0037 §"Parser scope"
  item 4) more clearly.

- **[minor] AC-2: "Decision → Clause 4" — the load-bearing
  "ADR-0037 NOT amended" point is buried at the bottom of the
  clause's prose.** A one-line lead would help: *"**ADR-0037
  is preserved without amendment.**"* as the second sentence
  of Clause 4, then the supporting prose.

- **[minor] AC-2: "Decision → Clause 3" — multi-paragraph
  density.** The clause covers ADR-0015 §Consequence 1 and
  §Consequence 5 in separate sub-bullets, then commits the
  branch-protection precondition. Three distinct points in
  one clause; could split into Clause 3a/3b. Borderline polish.

- **[minor] Spec applicability**: AC-1…AC-10 are study-shape
  criteria; the promotion ADR is governed by `adr-writing`
  A1–A8 instead. The originating study (PR #117) already
  satisfied AC-1…AC-10. AC-3 (≥2 options), AC-6 (Open
  Questions), AC-7 (promotion target) are not directly
  applicable to an ADR shape. Polishing note for the spec
  itself, not a finding against the target.

## Disposition

`/critique` runs in a session whose author also wrote the ADR
(author-equals-reviewer circularity per ADR-0051 §Consequence
7). This is the second R4 collapse PR in the amendment cycle
(PR #117 study + PR #117 critique-1 + ratification at PR #117
merge; this PR is the promotion-ADR + implementation slice).
The amendment's classification was operator-ratified at the
originating PR's merge act; this round's findings are
author-emitted polish + structural-precision flags against the
promotion ADR's standalone clarity. The two important findings
(dangling memory-file pointer, "amended in part by" precision)
are both worth applying — they improve the ADR's standalone
readability and align it with the more rigorous ADR-0010
precedent.
