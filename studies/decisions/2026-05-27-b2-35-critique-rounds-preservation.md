<!-- path: studies/decisions/2026-05-27-b2-35-critique-rounds-preservation.md -->

# B2-35 — Critique-Rounds Preservation

## Metadata

- **B2 reference:** B2-35 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md)
  (row at line 148, "Critique-rounds preservation", expected
  output "Draft study at this path; provisional promotion to
  `docs/adr/0048-critique-rounds-preservation.md` once
  `resolved-study`").
- **Status:** draft (resolved-study after the critique pass).
- **Last updated:** 2026-05-27.
- **Upstream resolved:**
  - **ADR-0014** ([`docs/adr/0014-trigger-handler-contract.md`](../../docs/adr/0014-trigger-handler-contract.md))
    — the only B0/B1/B2-vintage decision in this repository
    whose critique round was fully preserved end-to-end, via
    commit `926e3e5` ("feat(engine): W3-P4e HTTP trigger
    handler + critique round-1 fixes (#10)"). The preserved
    artifact lives in the PR description for #10 and in the
    follow-up commit `d2fd4a5` ("docs(studies): W3-P4e
    trigger-handler-contract — critique round-2 fixes"),
    which is the closest existing analogue to what this study
    proposes to standardize.
  - Commit `060dd10` ("chore(skills): extract three
    `.claude/skills/` — adr-writing, critique-anti-patterns,
    go-coding-standards") — the salvage of substantive
    learnings from lost rounds into
    `.claude/skills/critique-anti-patterns/`.
  - [`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
    step 6 — the current critique step, which carries no
    preservation requirement and is the protocol surface
    this study amends.
  - [`.claude/commands/critique.md`](../../.claude/commands/critique.md)
    — the current `/critique` command, which prints findings
    to stdout and does not write a file.
  - [`.claude/playbooks/feedback-protocol.md`](../../.claude/playbooks/feedback-protocol.md)
    — the finding template (`[severity] R/P/AC: section —
    change`) that the critique already uses; preservation
    does not modify this.
  - **R8** in `CLAUDE.md` §3 — studies are reasoning
    artifacts under `studies/`, not the published repository.
    This constrains where preserved critiques may live.
- **Downstream open:**
  - **ADR-0048** (provisional) — promotion target; slot may
    shift if other promotions land first. No scope-note pass
    redemption is required because the ADR-0020 pass already
    redeemed the prior overload.
  - **Protocol-edit session** that actually amends
    `.claude/playbooks/wave-1-session-loop.md` step 6 and (if
    confirmed) leaves `.claude/commands/critique.md`
    unchanged. That session is gated on this study reaching
    `resolved-study`; it does not run in this session per
    R1.
  - **First post-B2-35 critique round** that exercises the
    preservation path — the implicit acceptance test for
    whether the chosen shape works in practice.
- **Promotion target:** see final section.

## Context

The decision is procedural, not architectural. Wave 1 and the B2
series have produced 30+ studies under
[`studies/decisions/`](.). Every one of those studies passed
through the `/critique` step from
[`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
at least once, with many going through the two-round cap. The
*revisions* those critiques produced are preserved by git history
(draft commit → revised commit); the *critique transcripts
themselves* are not.

The single exception is **ADR-0014** — the W3-P4e HTTP trigger
handler — whose critique round-1 fixes shipped in commit
`926e3e5` with the critique transcript reproduced in the PR
description for #10, and whose critique round-2 fixes shipped
in commit `d2fd4a5` with a similar pattern. Every other study in
the repository shows only the *outcome* of its critique cycle,
not the cycle itself.

The substantive learnings from those lost rounds were salvaged
into `.claude/skills/critique-anti-patterns/` in commit
`060dd10`. That extraction produced three artifacts:

- [`.claude/skills/critique-anti-patterns/SKILL.md`](../../.claude/skills/critique-anti-patterns/SKILL.md)
  — the skill definition, finding template, severity scheme,
  six exemplar findings, four reaction-style anti-patterns,
  and seven substantive anti-patterns (prior-art-as-justification,
  hidden commitments, citation drift, "additive" mislabel,
  vocabulary drift, strawman options, unaddressed blocking
  findings).
- [`.claude/skills/critique-anti-patterns/reference/finding-template.md`](../../.claude/skills/critique-anti-patterns/reference/finding-template.md)
  — six labeled exemplar findings + four reaction-style
  anti-patterns, verbatim from
  [`feedback-protocol.md`](../../.claude/playbooks/feedback-protocol.md).
- [`.claude/skills/critique-anti-patterns/reference/anti-patterns-catalog.md`](../../.claude/skills/critique-anti-patterns/reference/anti-patterns-catalog.md)
  — eight anti-patterns (B1–B8) with each tagged "documented"
  or "preventive".

These artifacts prove the *substance* survived the lost rounds.
They do not, however, give future sessions a real critique
transcript to calibrate against — only the abstracted catalog.
This is acceptable for anti-pattern coverage (the catalog is
denser than any single critique would be) but poor for case-by-case
reasoning about why a particular finding was deferred vs.
blocking, why an operator accepted a `minor` finding without
revision, or how a `blocking` finding's resolution was scoped
across rounds.

The gap is bounded: only critique *transcripts* are missing.
The decision-log rows, the studies themselves, the promoted
ADRs, and the salvaged anti-patterns catalog are all present.
B2-35 closes the transcript gap going forward; it does not try
to reconstruct what was lost (see D4).

**Scope guardrail.** This study commits the preservation
shape only. It does not modify the finding template, the
severity scheme (`blocking` / `important` / `minor`), the
two-round cap from
[`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
step 7, or the substance of what `/critique` evaluates. The
adversarial-review nature of the cycle is unchanged.

## Decision Drivers

- **D1.** Preservation must not break **R8**: studies are
  scaffolding, not the published repository. Critique
  transcripts are reasoning artifacts, so they live under
  `studies/`, not `docs/`.
- **D2.** Preservation must not balloon the commit count per
  decision to the point that step 10 of
  [`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  becomes confusing. One additional commit per critique round
  is acceptable; more is not.
- **D3.** Operators must be able to see, from a single study
  file's Metadata block, whether and how many critique rounds
  it went through. Hidden critique skips are the failure mode
  to guard against.
- **D4.** The `/critique` command must remain idempotent and
  safe to re-run (no destructive side effects on previously
  preserved rounds; no implicit file overwrites).
- **D5.** Retroactive cost must be bounded. Wave 1 closed
  long ago; the B2 series is the active surface. Reconstructing
  30+ lost rounds is not a useful expenditure of the platform's
  time budget.
- **D6.** The protocol change must be auditable from the
  [`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  diff alone — i.e., a single playbook edit captures the
  whole new preservation contract, with no hidden behavior
  living elsewhere.
- **D7.** The preservation mechanism must accommodate
  legitimate skip cases (e.g., a typo-only revision, an
  operator-level edit that doesn't warrant a fresh critique)
  without becoming a hidden critique drop. Skip-with-declaration
  in the study's Metadata is the seam this driver names.

## Considered Options

### Option 1 — `studies/critiques/` directory, separate commit, full output, accept retroactive loss, `/critique` stays stdout-pure, skip-with-declaration (recommended)

- **Shape:** New sibling directory under `studies/` —
  `studies/critiques/<date>-<slug>-critique-<N>.md` — one
  file per critique round per study, where `<slug>` matches
  the slug of the parent study file under
  `studies/decisions/`. Filename example:
  `studies/critiques/2026-05-27-b2-35-critique-rounds-preservation-critique-1.md`.
- **D1 (where):** `studies/critiques/`. Lives under `studies/`
  per R8.
- **D2 (when):** Separate commit before the revision commit.
  Sequence: draft → critique (commit) → revision (commit).
- **D3 (what):** Full `/critique` stdout preserved verbatim
  inside a fenced `text` block, followed by an "Operator
  response" trailer recording per-finding disposition
  (`addressed` / `accepted-as-is with reason` /
  `deferred to OQ` / `out-of-scope per AC`).
- **D4 (retroactive):** Accept loss for all pre-B2-35
  studies. The anti-patterns catalog is the salvage. ADR-0014
  / commit `926e3e5` is the bootstrap exemplar but is **not**
  required to be backfilled into `studies/critiques/` in this
  session (see Open Questions).
- **D5 (protocol):** `wave-1-session-loop.md` step 6
  amended to specify that the operator captures the
  `/critique` stdout to the named file before the next
  iteration. `/critique` command itself is unchanged
  (stays stdout-pure, idempotent). Operator captures via
  shell redirection or copy-paste.
- **D6 (guarantee):** Skip is allowed if declared in the
  parent study's Metadata block under a `Critique rounds:`
  bullet (e.g., `Critique rounds: 1 preserved
  (studies/critiques/...-critique-1.md), 1 skipped (typo
  fix)`). A skipped round with no declaration is treated as
  an R3/R5-class violation by the next critique pass.

**Pros:** Clean separation between studies (the decisions)
and critiques (the reasoning that shaped them); each critique
is addressable by path; new directory appears organically when
the first post-B2-35 critique runs (R8-consistent); `/critique`
command surface stays trivial; operator workflow matches the
existing capture-stdout-to-file pattern used by `/critique`'s
sibling commands.

**Cons:** Adds one commit per critique round; relies on
operator discipline to capture stdout (the skip-rate risk D7
names); requires the slug to be derivable from the parent
study filename (one-to-one mapping; minor maintenance cost).

**Verdict:** Recommended. Minimal protocol surface change,
maximal R8 / R1 alignment, no command rewrite.

### Option 2 — Per-study subdirectory, atomic commit, full output, accept loss, `/critique` writes file, mandatory

- **Shape:** Each study under `studies/decisions/` is
  promoted from a single file to a directory:
  `studies/decisions/2026-05-27-b2-35-critique-rounds-preservation/study.md`
  + `critique-1.md` + `critique-2.md` siblings. The study
  filename loses its `.md` extension at the path level.
- **D1 (where):** `studies/decisions/<slug>/critique-N.md`.
- **D2 (when):** Atomic commit bundling critique + revision.
- **D3 (what):** Full critique output verbatim.
- **D4 (retroactive):** Accept loss, but every existing
  study must be migrated from file → directory shape, even
  empty ones. ~30 file-rename operations.
- **D5 (protocol):** `/critique` command rewritten to write
  the file directly to the parent study's directory.
- **D6 (guarantee):** Mandatory — no critique = no commit,
  enforced by a hook or convention.

**Pros:** Critiques live adjacent to the study they critique
(zero indirection); mandatory guarantee removes the skip-rate
risk; atomic commit makes the audit trail compact.

**Cons:** Disruptive — all existing studies must be
restructured, even though their critiques are lost. Atomic
commits hide the draft→critique→revision sequence in a single
diff (loses the visibility D6 demands). `/critique` writing a
file directly couples the command to a path convention and
makes the command non-trivially stateful (D4-violating).
Mandatory guarantee with no skip-with-declaration seam
violates D7.

**Verdict:** Rejected. Restructuring cost too high for a
gap that has already accepted loss; mandatory-no-skip is
too rigid; command rewrite expands the protocol surface
without proportionate gain.

### Option 3 — `docs/critique-history/` published, atomic commit, findings-only, reconstruct retroactively, `/critique` writes file, mandatory

- **Shape:** Critique transcripts live as published
  artifacts under `docs/critique-history/`, intended to be
  readable by anyone browsing the repo.
- **D1 (where):** `docs/critique-history/`. Directly under
  the published tree.
- **D2 (when):** Atomic commit.
- **D3 (what):** Findings only (verbatim) — no raw
  `/critique` stdout, no operator response trailer.
- **D4 (retroactive):** Reconstruct lost rounds from git
  blame, PR comments, and prior commit messages where
  feasible. Up to 30+ reconstruction passes.
- **D5 (protocol):** `/critique` writes the file.
- **D6 (guarantee):** Mandatory.

**Pros:** Critique history becomes discoverable to anyone
reading `docs/` — useful onboarding signal for new
contributors.

**Cons:** **Violates R8** — published artifacts are not
reasoning artifacts; critiques are explicitly reasoning. The
findings-only D3 position loses the operator-response signal
the catalog itself identifies as load-bearing
(anti-pattern B7 "strawman options" is recognizable only
when you can see the operator's disposition of each
finding). Reconstruction is high effort and low fidelity
(D5-violating). Mandatory-no-skip is too rigid (D7-violating).

**Verdict:** Rejected outright on R8 grounds. Even
ignoring R8, the findings-only content shape, the
reconstruction posture, and the mandatory-no-skip stance
each disqualify it independently.

## Recommendation

Adopt **Option 1**. The shape is minimal: one new sibling
directory under `studies/`, one new commit per critique round,
zero changes to the `/critique` command, one targeted edit to
[`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
step 6. The chosen positions for each sub-decision below
formalize what Option 1 commits.

### D1 — Where: `studies/critiques/`

Preserved critique rounds live at
`studies/critiques/<date>-<slug>-critique-<N>.md`, where:

- `<date>` matches the date in the parent study's filename
  (the date the critique ran, not the date the study was
  drafted).
- `<slug>` matches the slug of the parent study file
  exactly (e.g., parent
  `2026-05-27-b2-35-critique-rounds-preservation.md` →
  critique
  `2026-05-27-b2-35-critique-rounds-preservation-critique-1.md`).
- `<N>` is the round number, 1-indexed. With the two-round
  cap from
  [`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  step 7, `<N>` is 1 or 2.

The directory is created on first use; it is absent from the
repository until the first post-B2-35 critique runs
(R8-consistent: studies are scaffolding, optional).

### D2 — When: separate commit, before the revision

Sequence: draft commit → **critique commit** → revision
commit. The critique transcript is committed *before* any
revision is made to the parent study, so the operator's
revision diff stays focused on the changes the critique
drove. Step 10 of
[`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
gains one bullet allowing this intermediate commit:

```
git add studies/critiques/
git commit -m "docs(critique): B<N>-<M> — round <K> findings"
```

### D3 — What: full stdout verbatim + operator-response trailer

Each `studies/critiques/<date>-<slug>-critique-<N>.md` file
contains:

1. The R6 path-header HTML comment.
2. A single H1 title: `# B<N>-<M> — Critique Round <K>`.
3. A short Metadata block (parent study link, date, round
   number, `/critique` invocation timestamp).
4. A `## Findings` section containing the full `/critique`
   stdout output preserved verbatim inside a fenced ```text
   block. The fence preserves whitespace and the
   `[severity] R/P/AC: section — change` finding template
   exactly.
5. A `## Operator response` section with one bullet per
   finding, recording its disposition as one of:
   - `addressed in revision` (with one-line summary of what
     changed),
   - `accepted as-is` (with one-line reason),
   - `deferred to Open Questions` (with the OQ bullet's
     text),
   - `out-of-scope per AC-<N>` (citing the acceptance
     criterion).

The operator-response trailer is what allows future sessions
to reconstruct the operator's judgement, not just the
findings — without it, the preserved transcript reduces to
Option 3's findings-only stance.

### D4 — Retroactive: accept loss; the catalog is the salvage

No reconstruction is attempted for pre-B2-35 studies. The
`.claude/skills/critique-anti-patterns/` corpus (extracted in
commit `060dd10`) is the substantive salvage. The
ADR-0014 / commit `926e3e5` / `d2fd4a5` pair is
acknowledged as the bootstrap exemplar; whether to backfill
that pair's preserved transcript into `studies/critiques/`
is left as an Open Question for the protocol-edit session,
not as a gating dependency for this study.

Going forward, the first post-B2-35 critique round (whether
on this study or on the next decision in flight) is the
first entry in `studies/critiques/`.

### D5 — Protocol changes: `wave-1-session-loop.md` step 6 amends; `/critique` unchanged

The downstream protocol-edit session (gated on this study
reaching `resolved-study`) makes exactly two changes:

1. **`wave-1-session-loop.md` step 6** — extend the step's
   body to specify:
   - the capture target (`studies/critiques/<date>-<slug>-critique-<N>.md`),
   - the capture mechanism (operator-side stdout redirection
     or copy-paste; `/critique` stays stdout-pure),
   - the intermediate commit (per D2 above),
   - the operator-response trailer requirement (per D3
     above),
   - the skip-with-declaration seam (per D6 below).

2. **`.claude/commands/critique.md`** — *no change*. The
   command remains the adversarial-review-to-stdout
   surface defined today. This protects D4 (`/critique`
   stays idempotent and stateless) and keeps the
   command's protocol footprint trivial.

The protocol-edit session does **not** run inside this
B2-35 session per R1 (no protocol-file edits during study
drafts).

### D6 — Skip is allowed if declared in study Metadata

A study's Metadata block gains an optional `Critique rounds:`
bullet listing each round's disposition. Examples:

- `Critique rounds: 1 preserved
  (studies/critiques/.../critique-1.md), 1 skipped
  (typo-only revision)` — explicit skip with reason.
- `Critique rounds: 2 preserved
  (studies/critiques/.../critique-1.md,
  studies/critiques/.../critique-2.md)` — full two-round
  cycle.
- `Critique rounds: 1 preserved
  (studies/critiques/.../critique-1.md)` — single round,
  no second round needed.

A study with no `Critique rounds:` bullet is treated by the
next adversarial review as having undeclared rounds, and the
review will fire a `blocking` finding citing R6-class
visibility failure. This is the seam D7 names: skips are
permitted, hidden skips are not.

### Why this does not reopen ADR-0014

ADR-0014 is the only ADR in the repository whose critique
round was preserved end-to-end (in the PR description for
#10 and commit `d2fd4a5`). This study **does not modify
ADR-0014**. The ADR's content — eager-at-load hydration,
strict request decoder, separate API DTO, health-endpoint
semantics — is untouched. What this study does is
*standardize* the preservation pattern that ADR-0014's
shipping happened to follow informally. ADR-0014 remains
the bootstrap exemplar; whether its preserved transcript
is migrated into `studies/critiques/` is an Open Question
below, not a contract change to ADR-0014 itself.

### Why this does not reopen R8

R8 in `CLAUDE.md` §3 says studies are reasoning artifacts
and are not part of the published repository. This study
respects R8: `studies/critiques/` lives under `studies/`,
not under `docs/`. Option 3 (`docs/critique-history/`) was
rejected explicitly on R8 grounds. R8's principle —
"studies are scaffolding; ADRs are the building" — extends
naturally to critiques: critiques are scaffolding for the
study they shaped, so they live in the same scaffolding
tree.

## Consequences

### Positive

- **Future sessions can read a prior critique verbatim.**
  Once a few rounds accumulate under
  `studies/critiques/`, new critique passes can calibrate
  severity assignment and finding scope against real prior
  examples, not just the abstracted anti-patterns
  catalog.
- **The anti-patterns catalog has a concrete source pool
  to grow from.** New B-class anti-patterns can be added
  to
  [`.claude/skills/critique-anti-patterns/reference/anti-patterns-catalog.md`](../../.claude/skills/critique-anti-patterns/reference/anti-patterns-catalog.md)
  with citation to a specific preserved round.
- **The audit trail becomes single-file-per-round.** Step
  6 + step 7 of
  [`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  produce a one-to-one file mapping (one critique commit
  → one file → one round), so the operator's record-keeping
  burden is bounded and addressable.
- **The skip-with-declaration seam makes the loop
  honest.** Hidden critique drops were previously
  undetectable from the repo state alone; the
  `Critique rounds:` Metadata bullet makes them visible.

### Negative

- **Separate commit per critique round increases the
  commit count per B-item.** A B-item with two rounds now
  produces up to 5 commits (draft + critique-1 + revision-1
  + critique-2 + revision-2) where the prior pattern was
  often 2 (draft + revised). Step 10 of the session loop
  remains the single "log update + commit" step; the
  intermediate critique commits are operator-driven.
- **Operator must remember to capture `/critique`
  stdout.** This is the skip-rate risk D7 names. The
  skip-with-declaration seam mitigates it but does not
  eliminate it — an operator can still forget to declare.
- **Slug derivation has minor maintenance cost.** If a
  parent study's filename is ever renamed, its
  corresponding critique files must be renamed in
  lockstep. This is unlikely in practice (studies are
  immutable after `resolved-study`) but is worth naming.

### Operational

- The protocol-edit session that amends
  [`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  step 6 cannot begin until this study reaches
  `resolved-study`. That session's scope is bounded to the
  single playbook edit described in D5.
- The first post-B2-35 critique round is the implicit
  acceptance test. If it surfaces a mismatch between this
  study's prescription and operator reality, the protocol
  edit (not this study) is the right surface to amend.
- `/critique` is *not* edited in the downstream session.
  The command's spec stays as-is.

### Repository impact

- A new `studies/critiques/` directory appears the first
  time a post-B2-35 critique runs. Until then it is
  absent — R8-consistent and R1-consistent (no Wave-1-era
  directories created prematurely).
- No engine, rules, tools, or deploy workspace is
  touched. The full impact surface is
  `studies/foundation/06-decision-log.md` (this row),
  `studies/decisions/2026-05-27-b2-35-critique-rounds-preservation.md`
  (this study), `studies/critiques/` (created on first
  use), and the future
  [`docs/adr/0048-critique-rounds-preservation.md`](../../docs/adr/) (provisional).

## Open Questions

- **OQ-1.** Should commit `926e3e5`'s preserved critique
  round-1 (currently in the PR description for #10) and
  commit `d2fd4a5`'s preserved critique round-2 be
  backfilled into `studies/critiques/` as the bootstrap
  exemplar pair? Or left where they are, with this study
  pointing at them as the de facto exemplars? *Deferred to
  the protocol-edit session; out-of-scope for this study.*
- **OQ-2.** Should `/critique`'s exit code differ when
  `blocking` findings are present (e.g., exit 1 for
  blocking, 0 otherwise), so that an operator's skip
  decision is auditable from CI? *Out-of-scope for B2-35;
  potential B3 row.*
- **OQ-3.** Does the `studies/critiques/` slug convention
  need to be enforced by a linter (analogous to the path
  conventions
  [`.claude/playbooks/acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md)
  enforces for studies)? Likely no — the convention is
  one-to-one with parent study filenames, and divergence
  is self-evident from a directory listing — but the
  question survives the study draft. *Out-of-scope for
  B2-35; revisit if the operator-discipline risk
  materializes.*

All other questions surfaced during drafting are decided
explicitly in D1–D6 or rejected with reason in the
Considered Options section.

## Promotion target

[`docs/adr/0048-critique-rounds-preservation.md`](../../docs/adr/)
(provisional). Slot **0048** is the next available ADR
number as of 2026-05-27 (current high is ADR-0047,
[`docs/adr/0047-lint-substrate-access.md`](../../docs/adr/0047-lint-substrate-access.md),
"Channel-reachability linter substrate-access posture"). The
slot may shift if other promotions land first; the title
and content survive any renumbering.

**No scope-note pass redemption is required** because the
ADR-0020 pass already redeemed the prior overload. The
promotion target is committed at the standard
MADR-ADR shape: Title / Status / Context / Decision Drivers
/ Considered Options / Decision Outcome (mapping D1–D6
verbatim) / Consequences. The promoted ADR is the
load-bearing artifact for future critique runs; this study
is the reasoning behind it.
