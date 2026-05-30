<!-- path: studies/decisions/2026-05-29-b3-claude-tooling-postwave3.md -->

# B3 — Claude Tooling Post-Wave-3 Extension

## Metadata

- **Wave reference:** B3 (evolutionary lane; tooling extensions family
  per [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
  §Per-family scope).
- **Status:** draft (B3, session 2; pre-critique).
- **Last updated:** 2026-05-29.
- **Upstream resolved:**
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) (B3
  launch — eligibility filter and family list);
  [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)
  (critique-rounds preservation contract; the post-Wave-3 loop
  inherits it);
  [`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  (the loop shape the post-Wave-3 loop mirrors);
  [`.claude/playbooks/wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
  step 10 (the PR-flow contract that the new session-governance skill
  and `/open-pr` command extract and reuse);
  [`.claude/playbooks/acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md)
  (AC-1…AC-10 — the gate the post-Wave-3 loop still runs against
  for B3-N entries).
- **Eligibility check (ADR-0049 §(a)):**
  - **Condition 1 — P-B3.1, expands not rewrites.** The `.claude/`
    harness already exists. This study adds one new skill, one new
    playbook, one new command, and edits one command + one settings
    file. The draft → `/critique` → operator acceptance → promotion
    loop stays unchanged in shape. No existing playbook is rewritten;
    `wave-1-session-loop.md` and `wave-3-session-loop.md` continue
    to govern their respective phases. ✅
  - **Condition 2 — P-B3.4, in-scope family.** Family fit is
    **Tooling extensions** under ADR-0049 §Per-family scope, which
    captures "additions to `tools/lint/`, the manifest publisher, the
    dry-run runner, the engine dispatcher, **and adjacent tooling**
    that extend contract coverage without changing the contract
    shape." `.claude/` is adjacent tooling: it governs the contributor
    process that produces every artifact the platform ships
    (studies, ADRs, scaffolds, PRs). This work extends governance
    coverage to the post-Wave-3 evolutionary lane without reshaping the
    draft → critique → accept → promote contract. Borderline
    interpretation acknowledged: the canonical examples in §Per-family
    scope are platform runtime tools, not agent harness files. The
    "adjacent tooling" clause is load-bearing for this study.
    New contribution proposed here, requires review during
    `/critique`. **Pending — promoted to a §Decision Drivers D0
    precondition.** The Recommendation below is conditional on
    the ruling clearing in `/critique` before the study moves
    to `resolved-study`.
  - **Condition 3 — P-B3.2, conforms to ADR-0020/0021/0022/0023.**
    The study touches no substrate decision, no mode primitive, no
    kind catalog entry, no sources schema row. ✅
  - **Condition 4 — additive-maintenance threshold.** The five gaps
    below reshape contributor expectations (a first-class PR-flow
    skill, a phase-aware promotion gate, a settings posture that
    no longer authorizes merge) — material enough to warrant a
    study trail above a routine catalog PR. ✅
- **Constraint envelope:**
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §(a)
  (eligibility), §(b) (out-of-scope), §Per-family scope
  (Tooling extensions);
  [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)
  (preserved critique rounds — the new playbook inherits the
  capture-and-commit contract);
  [`CLAUDE.md`](../../CLAUDE.md) §3 R1–R8 (especially R4
  one-topic-per-session, R5 own-the-pattern, R6 path header);
  [`CLAUDE.md`](../../CLAUDE.md) §4 P1–P6.
- **Locked premises** (operator-declared, not litigated here):
  - **P-B3CT.1** — The post-Wave-3 evolutionary lane is the
    project's current operating phase. Work lands via PR (per
    [`.claude/playbooks/wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
    step 10's direct-to-main carve-out, which closes at the
    Phase-4 sub-phase split). This study does not re-litigate the
    PR-only posture.
  - **P-B3CT.2** — Minimal intervention per gap. Where an existing
    playbook or command can be extended with one line or one
    section, prefer the edit over a new file. Where the discipline
    is genuinely new (PR-flow as a first-class contributor
    obligation), prefer a new artifact in the same shape as the
    existing ones.
  - **P-B3CT.3** — No platform code changes. R1 still applies for
    the duration of this study; the proposed `settings.local.json`
    payload appears as a fenced markdown block in §Recommendation,
    not as a real file edit.
  - **P-B3CT.4** — The five gaps were enumerated by the operator at
    session open and are treated as the binding scope. Adjacent
    issues surfaced during drafting are listed in §Open Questions,
    never absorbed silently (R4).
- **Downstream open:** none enumerated. If `/critique` surfaces
  blocking findings that require a sixth gap, it is registered
  there and the study re-scopes — it does not silently grow.
- **Promotion target:**
  `docs/adr/0051-claude-tooling-postwave3.md` — provisionally
  the next available number at the time of writing (last landed
  is [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md),
  2026-05-29). Subject to the
  [ADR-0020](../../docs/adr/0020-wave-s-launch.md) §"Per-item
  ADR numbering" caveat per
  [`adr-writing`](../../.claude/skills/adr-writing/SKILL.md) A8:
  > Per-item ADR numbering. B0-S1 through B0-S7 promote in order
  > to `docs/adr/0021-…` through `docs/adr/0027-…` respectively,
  > modulo shifts if an unrelated promotion lands between B0-S
  > items. The expected sequence is descriptive, not reserved.

  If B-2a / B-2b (or any other in-flight study) is promoted
  before this one lands, the number slides — exactly the
  scenario the caveat covers. The G5 reservation mechanism
  proposed in §Recommendation is the procedural backstop for
  this study's own promotion.
- **Loop discipline:** standard B3 protocol — draft → `/critique`
  (≥1 round, preserved under `studies/critiques/` per
  [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md))
  → operator acceptance → promotion to ADR.

---

## Context

The platform's operating phase has shifted. Waves 1, 2, and 3 closed
on 2026-05-21 / 2026-05-21 / 2026-05-23; Wave-S launched 2026-05-23
and its foundational triplet reached `resolved-adr` by 2026-05-24;
B3 launched 2026-05-29 via
[ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md). The
operating cadence is now the post-Wave-3 evolutionary lane,
landing through pull requests against `main` — never direct
commits — per
[`.claude/playbooks/wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
step 10 (everything from W3-P4a onward; no carve-out for protocol
changes).

The `.claude/` harness, however, was written for an earlier shape of
the work and still reads as if Wave 1, Wave 2, and Wave 3 were the
operating modes. Concretely, five gaps surface when a contributor
opens a session today:

- **G1 — Phase lacuna.**
  [`.claude/commands/promote-to-adr.md`](../../.claude/commands/promote-to-adr.md)
  lines 12–20 gate on "every B0 row at `resolved-study` or
  `resolved-adr`" plus the Wave 2 consolidated document — a Wave-1
  gate that has now always passed for two weeks. There is no
  session-loop playbook for B2 or B3 (only
  [`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  and
  [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)),
  and there is no `/resolve-b2` or `/resolve-b3`. The B2-20 study
  landed 2026-05-29 by manually reusing `/resolve-b0` — the harness
  has been silently overloaded.

- **G2 — PR-flow buried in a playbook step.** The branch-then-PR
  discipline (create the branch before the first commit; confirm
  branch; never commit on `main`; open the PR; **stop** without
  merging) lives only in
  [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
  step 10 (lines 81–118). There is no reusable skill or command that
  encodes it. Sessions outside Wave 3 — B2-20 last week, this B3
  session today — re-derive the discipline from the playbook every
  time, by hand.

- **G3 — Settings posture too broad.** The current
  [`.claude/settings.local.json`](../../.claude/settings.local.json)
  allows `Bash(gh pr *)` (lines 14), which transitively authorizes
  `gh pr merge` — the one PR operation
  [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
  step 10 lines 103–105 explicitly reserves for the [H] reviewer
  after CI passes. There is no deny entry for `--no-verify` (which
  the harness would otherwise honor — bypassing pre-commit hooks is
  one of the destructive operations
  [`CLAUDE.md`](../../CLAUDE.md) §"Executing actions with care"
  lists as warranting confirmation). The Wave-0 / post-Wave-3 Make
  targets (`make demo-p6`, `make test-tools`, `make validate-deploy`,
  `make test-engine-sandbox`) appear nowhere — every session re-asks
  for them.

- **G4 — No skill for session governance.**
  [`adr-writing`](../../.claude/skills/adr-writing/SKILL.md),
  [`critique-anti-patterns`](../../.claude/skills/critique-anti-patterns/SKILL.md),
  and
  [`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
  encode authoring, review, and coding conventions respectively.
  None of them encodes the cross-cutting **session-governance**
  discipline that every session needs: branch before first commit;
  confirm branch is not `main`; never run `gh pr merge`; never pass
  `--no-verify`; cite the ADR or B-row in every produced artifact;
  one topic per session (R4). Today this discipline is encoded
  only as prose in
  [`CLAUDE.md`](../../CLAUDE.md) §"Executing actions with care",
  [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
  step 10, and [R4 in CLAUDE.md](../../CLAUDE.md) §3. A skill —
  matching the shape of the three existing ones (frontmatter
  `name` + `description`; `SKILL.md` + `reference/`; file:line
  citations) — would let the harness load it automatically when
  the matched trigger fires.

- **G5 — ADR number unreserved at promotion time.**
  [`.claude/commands/promote-to-adr.md`](../../.claude/commands/promote-to-adr.md)
  line 28 instructs the agent to "determine the next available ADR
  number under `docs/adr/`" at write time. Under direct-to-main
  commits this was a non-issue. Under post-Wave-3 PR-flow with
  parallel branches in flight — B2-20 (ADR-0050) and this study
  (target ADR-NNNN) overlap today — two simultaneous promotions can
  both compute the same `<NNNN>` and collide when the second PR
  merges. The current command has no reservation step.

Together, the five gaps mean that today's `.claude/` harness still
encodes the platform's Wave-1 operating mode while the operating
mode itself has moved on by two waves and one evolutionary launch.
Closing the five gaps with the minimum-viable intervention is the
purpose of this study.

The principles bearing on this study are **P5** (evolution must be
contract-driven — the agent harness is itself a contract with the
contributor population, and it evolves under a published shape just
like every other artifact) and **P6** (borrow patterns, not baggage —
the four new artifacts described in §Recommendation are defended on
the fit to our own workflow, not on resemblance to any external
agent-harness convention).

---

## Decision Drivers

- **D0 — Eligibility precondition under
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §(a).**
  This study reads the "Tooling extensions" family
  ([ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
  §Per-family scope) to include the `.claude/` agent harness as
  "adjacent tooling" — alongside the canonical examples
  (`tools/lint/`, the manifest publisher, the dry-run runner,
  the engine dispatcher). This reading is **new contribution
  proposed here, requires review** (R5). It is the foundational
  ruling for this study: if `/critique` rules against the
  reading, the §Recommendation below does not stand and the
  study re-scopes (either as an ADR-0049 amendment proposing a
  new B3 sub-family for agent-harness extensions, or as a
  Wave-2-style consolidated decision outside B3). D0 is a
  precondition the study acknowledges must clear in `/critique`
  before the study moves to `resolved-study`; it is not an Open
  Question because the entire Recommendation is conditional on
  it. The §Consequences below preserve the re-scope path in
  point 8.

- **D1 — Phase awareness.** The harness must reflect that B3 is
  open, B2 is mid-stream, and the Wave-1/Wave-2 gates have closed
  permanently. A contributor reading `/promote-to-adr` today gets a
  vestigial Wave-1 gate; this must be fixed at the command level,
  not papered over.

- **D2 — PR-flow as first-class discipline.** The branch-then-PR
  discipline now applies to every session, not just Wave-3
  scaffolds. It must be a reusable artifact (skill + command), not
  a copy-paste from playbook step 10.

- **D3 — Settings posture matches stated discipline.** What
  [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
  step 10 lines 103–105 *says* ("the PR is merged via the GitHub
  UI" by the [H] reviewer) must be what the settings file
  *authorizes*. A wildcard that allows `gh pr merge` contradicts
  the stated contract. The settings file is the harness's
  enforcement layer; it must align with the playbook.

- **D4 — Minimal intervention per gap.** Each gap closes with the
  smallest artifact that lets the harness know the shift has
  happened. New artifacts mirror the shape of existing ones (skills
  with `SKILL.md` + `reference/`; playbooks with the 10-step loop;
  commands with frontmatter + path-header). Two of the five gaps
  close with one-line edits to an existing command (G1, G5); only
  three need new files (G2, G3, G4).

- **D5 — Eligibility-conforming under B3.** Every artifact this
  study proposes stays inside the "Tooling extensions" family per
  ADR-0049 §Per-family scope. None reshapes a mode, a kind catalog
  entry, a substrate decision, or a sources-schema row. The
  borderline-interpretation note in §Metadata applies and is the
  load-bearing review point during `/critique`.

- **D6 — Discoverability for future sessions.** A new contributor
  (human or agent) opening a session next quarter must find the
  PR-flow discipline in an obvious place — not buried in a
  playbook step that they may not read until they're already
  about to commit on `main`. A skill triggered on a
  branch-or-commit-or-PR phrase is the natural surface.

- **D7 — `CONTRIBUTING.md` as the upstream authority for
  PR-flow.** The branch-then-PR contract today lives in
  `CONTRIBUTING.md` scoped to Wave-3 only. The post-Wave-3
  evolutionary lane needs the same contract generalized, and
  the `.claude/` artifacts proposed below (the new playbook,
  the `/open-pr` command, the `session-governance` skill) must
  defer upward to `CONTRIBUTING.md` as the documented surface —
  not codify a parallel contract that can drift silently. The
  consequence is one additional edit in §Recommendation: extend
  `CONTRIBUTING.md` with a post-Wave-3 section that generalizes
  the PR-flow today scoped to Wave-3. The Wave-3-specific text
  is **not** rewritten (R4 — one topic per session); the
  extension adds a new section beside it.

---

## Considered Options

For each of the five gaps, this section lists at least two viable
options with real trade-offs. Strawman options (B6 in
[`critique-anti-patterns`](../../.claude/skills/critique-anti-patterns/SKILL.md))
are out — every rejected option below carries both pros and cons.

### G1 — Phase awareness in the harness

The harness needs to (a) replace the stale Wave-1 gate in
`/promote-to-adr` and (b) provide a session-loop for post-Wave-3
work that mirrors the Wave-1 loop's discipline.

- **Option G1-A — Post-Wave-3 session-loop playbook + gate edit.**
  Add
  `.claude/playbooks/post-wave3-session-loop.md` mirroring
  [`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  in shape (10 steps with [H] decision points; draft → critique
  preserved per ADR-0048 → revision → log update → close commit)
  but with: (i) the
  [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
  step 10 PR-flow embedded as the close-commit substitute;
  (ii) an ADR-0049 §(a) eligibility-check step as the grounding
  step. Edit
  [`.claude/commands/promote-to-adr.md`](../../.claude/commands/promote-to-adr.md)
  to drop the Wave-1 gate, recognize the post-Wave-3 evolutionary lane, and
  add the reservation step from G5-A.
  - *Pros:* one new playbook; one edit to an existing command;
    contributor reads exactly one loop file for any B2 or B3
    session; the Wave-1 loop stays preserved as Wave-1 history.
  - *Cons:* a second post-loop is one more file in
    `.claude/playbooks/`; if B2 and B3 ever diverge in shape, a
    third loop becomes necessary.

- **Option G1-B — Parametrize `/resolve-b` into one command and
  drop the wave-specific loops.** Replace `/resolve-b0` with
  `/resolve-b <tier> <slug>` where `<tier>` is `b0`, `b2`, or
  `b3` (the B1 backlog closed 2026-05-25 per
  [`06-decision-log.md`](../foundation/06-decision-log.md) line
  46 — no B1-tier sessions remain, so `b1` is not in the
  parameter set). Collapse the three loops into one
  parametrized `session-loop.md`.
  - *Pros:* one playbook, one command, less surface area.
  - *Cons:* the Wave-1 loop's [H] decision points (step 3 "choose
    a B0 whose dependencies are resolved") are wave-specific and
    don't map cleanly onto B3 demand-driven pacing (P-B3.3 — rows
    are born when demand arises, not picked from a queue). A
    parametrized command hides this divergence inside a switch
    statement. The Wave-1 loop has historical value as the
    canonical shape; replacing it loses the audit trail.

- **Option G1-C — Two separate commands `/resolve-b2` and
  `/resolve-b3`.** Spawn two new commands matching the shape of
  `/resolve-b0`, each with its own playbook.
  - *Pros:* each command is precise to its tier; the harness can
    trigger different grounding reads (B2 reads ADR-0035-style
    upstream; B3 reads ADR-0049 eligibility).
  - *Cons:* three near-identical commands; the divergence between
    B2 and B3 is small in practice (both are post-Wave-3 demand-
    driven; both produce a study with the same shape). Spawning
    twins for a difference this thin invites copy-paste drift.

**Recommendation for G1: Option A.** One new playbook
(`post-wave3-session-loop.md`); one edit to
`/promote-to-adr.md`. The third command question (one parametrized
vs two separate) is parked in §Open Questions OQ-1 — it's a
follow-on decision that the first one or two post-Wave-3 sessions
will resolve empirically.

### G2 — PR-flow discipline as a reusable artifact

The branch-then-PR discipline currently lives only in
[`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
step 10 lines 81–118. Three places it could be extracted to:

- **Option G2-A — Promote PR-flow to a new skill
  (`session-governance`).** Codify the discipline in
  `.claude/skills/session-governance/SKILL.md` matching the shape
  of the three existing skills (frontmatter `name` + `description`;
  `SKILL.md` + `reference/`; file:line citations to the playbook,
  CLAUDE.md, and the relevant ADRs). The skill encodes: create the
  branch before the first commit; confirm with
  `git branch --show-current` (fallback `git rev-parse --abbrev-ref
  HEAD`); never commit on `main`; open the PR via `gh pr create`
  and **stop** (never `gh pr merge`); never pass `--no-verify`;
  cite the ADR or B-row in every produced artifact; one topic per
  session (R4).
  - *Pros:* skills are the harness's mechanism for cross-cutting
    discipline that triggers on context (vs commands which trigger
    on slash); the contributor population already reads the three
    existing skills as the project's canonical conventions;
    matches the precedent of
    [`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
    (also a discipline, also encoded as a skill).
  - *Cons:* one more skill directory; the skill's frontmatter
    description must be precise enough that the harness loads it
    when relevant and not when not.

- **Option G2-B — Promote PR-flow to a slash command (`/open-pr`
  alone, no skill).** A `/open-pr` command runs the PR-opening
  checklist — but leaves the cross-cutting discipline (never on
  `main`, never `--no-verify`, cite the ADR) implicit.
  - *Pros:* one new artifact; matches the
    one-command-per-procedure precedent of `/resolve-b0`,
    `/critique`, `/promote-to-adr`.
  - *Cons:* commands are slash-triggered — the contributor has to
    *remember* to run them. The PR-flow discipline must apply
    whether or not `/open-pr` was the entry point (e.g., when the
    contributor types `git commit` directly). A skill triggers
    automatically when the matched context fires; a command does
    not.

- **Option G2-C — Leave the PR-flow in
  [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
  step 10 only; encode in the new post-Wave-3 playbook the
  instruction "follow step 10 of `wave-3-session-loop.md`".**
  Cross-reference instead of extract.
  - *Pros:* zero new artifacts; no risk of drift between the two
    copies.
  - *Cons:* the contributor reading the new post-Wave-3 playbook
    has to jump to a Wave-3 playbook to learn the close move,
    which is exactly the lacuna G1 is trying to close. The
    cross-reference also doesn't give the harness a skill to load.

**Recommendation for G2: Option A + Option B together.** A new
skill `session-governance` for the cross-cutting discipline; a new
command `/open-pr` for the PR-opening checklist (G2 + G5 below).
Both artifacts cite each other; neither replaces the other.

### G3 — Settings posture

Three ways to align the settings file with the stated discipline:

- **Option G3-A — Replace `Bash(gh pr *)` with a specific allowlist
  (`create`, `view`, `list`, `diff`, `checks`, no `merge`); add a
  deny entry for any command containing `--no-verify`; extend the
  allowlist to cover the missing Make targets (`make demo-p6`,
  `make test-tools`, `make validate-deploy`,
  `make test-engine-sandbox`, etc.).** Proposed JSON in
  §Recommendation as a fenced markdown block (R1 — this is still
  a study).
  - *Pros:* settings now match the playbook's stated contract;
    `gh pr merge` requires explicit user approval at runtime
    (matching D3); `--no-verify` is denied by the harness; the
    Make targets stop generating permission prompts.
  - *Cons:* every new `gh pr` subcommand introduced upstream (e.g.,
    `gh pr ready`, `gh pr lock`) needs an explicit allowlist
    entry — the file becomes more maintenance.

- **Option G3-B — Keep the wildcard; document the merge prohibition
  in the playbook only.** Lean on the contributor reading the
  playbook before committing.
  - *Pros:* zero settings churn.
  - *Cons:* exactly the failure mode the discipline is meant to
    prevent — a contributor (human or agent) skips the playbook
    read and the harness rubber-stamps `gh pr merge`. D3 fails.

- **Option G3-C — Split into user vs project settings.** Move the
  permissive `gh pr *` to the user's global settings; keep the
  project's `settings.local.json` minimal.
  - *Pros:* per-contributor flexibility.
  - *Cons:* the project's enforcement is now invisible from the
    repository — a contributor reading the repo can't see what
    the harness actually authorizes. D3 fails for repository
    reviewers; the contract is no longer auditable.

**Recommendation for G3: Option A.** The proposed JSON is in
§Recommendation as a fenced markdown block.

### G4 — Codifying session governance as a skill

This is the same artifact discussed in G2-A. Two options for where
the content lives:

- **Option G4-A — New skill
  `.claude/skills/session-governance/`** with `SKILL.md` +
  `reference/`. Frontmatter description names the triggers:
  *use whenever a session opens or before any commit or PR
  action*. Body codifies the rules from
  [`CLAUDE.md`](../../CLAUDE.md) §"Executing actions with care",
  R4 (one-topic-per-session), R5 (cite, never name prior art),
  R6 (path header on every markdown file), and the
  branch-then-PR contract from
  [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
  step 10. File:line citations to all sources.
  - *Pros:* matches the existing skill shape; triggers on context;
    the contributor population already reads the three existing
    skills as the project's conventions.
  - *Cons:* the skill duplicates one-line policies that also
    appear in CLAUDE.md; the skill and CLAUDE.md must stay in
    sync (drift-check could be added to `/sync-agents`).

- **Option G4-B — Extend
  [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
  with a standalone "Session governance" section.** Keep the
  policy in the playbook the contributor is already reading for
  the loop shape.
  - *Pros:* zero new artifacts.
  - *Cons:* the discipline is cross-cutting — it applies whether
    the session is following the Wave-3 loop or the post-Wave-3
    loop. Pinning it to one playbook hides it from sessions
    following the other.

- **Option G4-C — Extend
  [`feedback-protocol.md`](../../.claude/playbooks/feedback-protocol.md).**
  Wedge the policy into the existing feedback playbook.
  - *Pros:* zero new artifacts.
  - *Cons:* feedback-protocol governs reviewer feedback shape, not
    the contributor's own discipline. Wrong file.

**Recommendation for G4: Option A.** The skill ships at
`.claude/skills/session-governance/`; the drift-check addition to
`/sync-agents` is parked in §Open Questions OQ-2.

### G5 — ADR number reservation under parallel PRs

Two ways to prevent the collision:

- **Option G5-A — Reservation step inside the edited
  `/promote-to-adr.md`.** Add a first-step instruction: "before
  writing the ADR file, post a message naming the proposed
  `<NNNN>` and pause for operator acknowledgement. If two
  sessions are open in parallel and the operator confirms the
  proposed number is taken, the agent re-derives the next free
  number." The reservation is operator-side bookkeeping (the
  operator tracks reserved numbers across sessions); the
  command makes the reservation step explicit.
  - *Pros:* one-line edit to an existing command; no new file;
    inherits the existing
    [ADR-0020](../../docs/adr/0020-wave-s-launch.md) §"Per-item
    ADR numbering" descriptive-not-reserved framing — the
    reservation is per-session, not per-decision-log row.
  - *Cons:* the reservation is operator-side; if the operator
    runs two parallel sessions and forgets which numbers are
    reserved, the collision still happens. The fix is
    procedural, not mechanical.

- **Option G5-B — Auto-allocate at PR merge time.** A pre-merge
  hook rewrites the ADR filename to the next free number,
  rewrites all in-file references to itself, and updates the
  decision log. No reservation step needed.
  - *Pros:* mechanical; no operator-side bookkeeping.
  - *Cons:* the ADR file path is referenced from the study, from
    the decision log, and (once promoted) potentially from other
    ADRs. A merge-time rewrite touches all of them — the blast
    radius is too large for a one-merge-one-PR rewrite. Reviewers
    can no longer see the final file path until the merge runs.
    Fails the "reviewers see what they approve" property.

- **Option G5-C — Separate `/reserve-adr-number` command.** A
  pre-flight command that the operator runs to mark `<NNNN>` as
  taken in a shared file (e.g., `docs/adr/.reserved.txt`).
  - *Pros:* mechanical; reservation is visible in-repo.
  - *Cons:* one more command; one more file; the shared file
    becomes a contention point of its own (two parallel
    `/reserve-adr-number` runs can race on the file itself).
    Heavier than the problem warrants today, when only two
    parallel sessions are likely at once.

**Recommendation for G5: Option A.** One-line edit to
`/promote-to-adr.md`; reservation is operator-side; the harness
makes the step explicit so it cannot be skipped silently.

---

## Recommendation

Conditional on D0 (eligibility precondition under
[ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §(a))
clearing in `/critique` before the study moves to
`resolved-study`, close the five gaps with **three new files
(skill, playbook, command), one settings hardening, and two
edits (`promote-to-adr.md`, `CONTRIBUTING.md`)**, all inside the
"Tooling extensions" family per ADR-0049 §Per-family scope.

### Artifacts to produce (in the promotion / Wave-3 follow-on)

1. **New skill — `.claude/skills/session-governance/SKILL.md`**
   (closes G4; supports G2). Frontmatter:
   `name: session-governance`; description triggers on verbatim
   phrases the contributor or agent is likely to use at the
   load-bearing moments: *"create a branch"*, *"git switch"*,
   *"git checkout -b"*, *"first commit"*, *"open a PR"*,
   *"gh pr create"*, *"merge the PR"*. The skill does **not**
   rely on a hypothetical "session open" trigger — the existing
   three skills load on description-match basis only
   ([`adr-writing`](../../.claude/skills/adr-writing/SKILL.md),
   [`critique-anti-patterns`](../../.claude/skills/critique-anti-patterns/SKILL.md),
   [`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
   all use "Use when…" / "Apply when…" framing keyed to the
   work being done, not the session lifecycle). The trigger
   shape proposed here is **new contribution proposed here,
   requires review** (R5) — pending verification that the
   verbatim phrases reliably load the skill in practice. Body
   codifies the contract (branch-before-first-commit; confirm
   branch via `git branch --show-current` / fallback
   `git rev-parse --abbrev-ref HEAD`; never `gh pr merge`;
   never `--no-verify`; cite ADR/B-row in every produced
   artifact; R4 one-topic-per-session) with file:line citations
   to [`CLAUDE.md`](../../CLAUDE.md) §"Executing actions with
   care" and §3 R4–R6, to `CONTRIBUTING.md` per D7 (the upstream
   authority for PR-flow), and to
   [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
   step 10. `reference/` holds the long-form rationale and the
   verbatim PR-flow checklist. **This artifact reads from
   `CONTRIBUTING.md`; updates to the PR-flow contract land in
   `CONTRIBUTING.md` first and the skill follows.**

2. **New playbook —
   `.claude/playbooks/post-wave3-session-loop.md`** (closes G1).
   Mirrors
   [`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
   in shape: 10 steps with [H] decision points at the same
   positions; draft → `/critique` preserved per
   [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)
   → revision → log update. Two deltas from the Wave-1 loop:
   - **Step 2 (grounding) adds an ADR-0049 §(a) eligibility
     check** before the agent proceeds — confirm the proposed
     entry falls inside one of the three families, conforms to
     the ADR-0020/0021/0022/0023 envelope, and crosses the
     additive-maintenance threshold. If the check fails, the
     session aborts and the row is re-triaged. **The
     eligibility-check step is new contribution proposed here,
     requires review** (R5) — no existing loop playbook carries
     an eligibility-check step in its 10-step shape; the Wave-1
     loop grounds on `/check-decision-backlog`, the Wave-3 loop
     grounds on upstream `resolved-study` / `resolved-adr`
     confirmation.
   - **Step 10 (commit) is replaced by the PR-flow contract
     authoritative in `CONTRIBUTING.md`** per D7 — create the
     feature branch using the slug provided by the operator for
     the session (provisional per the governance contract,
     pending the post-Wave-3 `CONTRIBUTING.md` extension in
     §Recommendation #6); confirm the branch is not `main` via
     `git branch --show-current`; stage; commit on the branch;
     push; `gh pr create`; **stop** without merging.
     Authoritative reference for the PR-flow contract is
     `CONTRIBUTING.md` (D7); the new `session-governance` skill
     is the in-session reading layer.

   **This artifact reads from `CONTRIBUTING.md`; updates to the
   PR-flow contract land in `CONTRIBUTING.md` first and the
   playbook follows.**

3. **New command — `.claude/commands/open-pr.md`** (closes G2;
   inputs into G5). Frontmatter `description: Open a PR from the
   current feature branch against main, following the PR-flow
   checklist`. Body runs the checklist:
   - confirm the current branch is **not** `main` using
     `git branch --show-current` (fallback
     `git rev-parse --abbrev-ref HEAD`);
   - show `git log main..HEAD` so the operator sees the staged
     commits;
   - run the gates applicable to the unit (the relevant
     `make lint-*`, `make test-*`, `make validate-deploy` —
     enumerated by the new `session-governance` skill);
   - run `gh pr create --base main` with a PR body that lists
     the citation map (ADR / B-row references), the critique
     result, and a test plan (matching
     [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
     step 10 lines 91–99);
   - **stop**. Never call `gh pr merge`.

   **This artifact reads from `CONTRIBUTING.md` per D7; the
   PR-opening checklist mirrors the contract documented there
   and updates land in `CONTRIBUTING.md` first.**

4. **Hardened `.claude/settings.local.json`** (closes G3). The
   shape proposed below — fenced markdown block, not a real file
   edit (R1: this is a study). The Wave-3 follow-on PR that
   promotes the ADR may include the real file edit; this study
   may not. R6 path-header convention does not apply: JSON has no
   comment syntax, and R6 in
   [`CLAUDE.md`](../../CLAUDE.md) §3 governs markdown files only.

   ```json
   {
     "permissions": {
       "allow": [
         "Bash(git checkout *)",
         "Bash(git pull *)",
         "Bash(git fetch *)",
         "Bash(git branch --show-current)",
         "Bash(git rev-parse --abbrev-ref HEAD)",
         "Bash(git log main..HEAD)",
         "Bash(go vet *)",
         "Bash(go test *)",
         "Bash(make up *)",
         "Bash(make down *)",
         "Bash(make lint *)",
         "Bash(make test-engine-integration *)",
         "Bash(make test-tools *)",
         "Bash(make test-engine-sandbox *)",
         "Bash(make validate-deploy *)",
         "Bash(make demo-p6 *)",
         "Bash(gh pr create *)",
         "Bash(gh pr view *)",
         "Bash(gh pr list *)",
         "Bash(gh pr diff *)",
         "Bash(gh pr checks *)",
         "Bash(echo \"vet exit: $?\")",
         "Bash(echo \"test exit: $?\")",
         "Bash(echo \"lint exit: $?\")"
       ],
       "deny": [
         "Bash(gh pr merge *)",
         "Bash(git commit --no-verify*)",
         "Bash(git push --no-verify*)"
       ]
     }
   }
   ```

   Three changes from the current file:
   - `Bash(gh pr *)` is replaced with five specific allowlist
     entries (`create`, `view`, `list`, `diff`, `checks`); `merge`
     is moved to deny.
   - A `deny` block is introduced with two explicit per-call-site
     `--no-verify` entries (`git commit --no-verify*` and
     `git push --no-verify*`). The entries use the pure-suffix
     shape `Bash(<prefix> *)` precedented by the current
     [`settings.local.json`](../../.claude/settings.local.json)
     allow list — `--no-verify` appears immediately after the
     fixed command prefix, with the trailing `*` absorbing any
     subsequent arguments. **Use of a `deny` block, and the
     per-call-site deny entries, are new contribution proposed
     here, requires review** (R5) — the current settings file
     has only an `allow` list and the harness's permission
     semantics for `deny` and for `--no-verify` as the
     load-bearing token in the suffix are unverified. Pending
     harness-docs check, the coverage of these entries is
     partial: they catch `--no-verify` only when it appears
     immediately after `git commit` / `git push` (as the first
     argument). Cases where `--no-verify` follows other
     arguments (e.g., `git commit -m "msg" --no-verify`) are
     **not** caught by these entries; the in-session
     confirmation gate remains the safeguard for those.
     Tightening to a substring pattern (`Bash(git commit *--no-verify*)`)
     would cover more cases but introduces a wildcard shape
     (`*` in the middle of the argument) not precedented by
     the current settings file — that escalation is parked
     until harness-docs are checked.
   - Wave-0 / post-Wave-3 Make targets are added explicitly.

   Force-push denies (`git push --force`, `git push -f`) are
   recognized as a related hardening but **out of scope for
   this study's G3** — see §Open Questions OQ-G3.1.

### Edits to existing files

5. **Edit `.claude/commands/promote-to-adr.md`** (closes G1's
   gate update; closes G5). Two changes:
   - **Replace the Wave-1 gate (lines 12–20)** with a post-Wave-3
     recognition step: confirm that the study being promoted
     carries `Status: resolved-study`, that its
     `Promotion target` line names a concrete
     `docs/adr/<NNNN>-<slug>.md` filename, and that the
     promotion is happening under post-Wave-3 PR-flow (per the
     new `session-governance` skill). The historical Wave-1 gate
     stays referenced in the command's body as a one-line note
     ("Wave-1 gate met 2026-05-21; preserved here for audit").
   - **Add a reservation step** (closes G5): before writing the
     ADR file, the agent posts a message naming the proposed
     `<NNNN>` and pauses for operator acknowledgement. The
     operator confirms or supplies a different number. The
     command's body explains the reservation is operator-side
     and is needed because parallel PR branches can race on the
     same `<NNNN>`.

6. **Edit `CONTRIBUTING.md`** (closes D7; new upstream-authority
   layer). Extend `CONTRIBUTING.md` with a new section that
   generalizes the PR-flow contract today scoped to Wave-3
   (`wave-3/<phase>-<topic-slug>` branch convention; create
   branch before first commit; never commit on `main`; never
   `gh pr merge`; never `--no-verify`; one-topic-per-session)
   to cover the post-Wave-3 evolutionary lane. The Wave-3
   section text is **not** rewritten (R4 — one topic per
   session); the extension adds a new sibling section beside
   it (suggested heading: *"Post-Wave-3 evolutionary lane"*).
   The provisional branch-slug forms operators have been
   passing (`chore/`, `feat/`, `docs/decision/`, `docs/adr/`)
   are recorded in the new section as **new contribution
   proposed here, requires review** (R5) — pending review they
   stay provisional; the playbook and the skill cite the
   section by anchor rather than re-stating the slugs. This
   edit lands **before** the `.claude/` artifacts so the
   artifacts have an upstream contract to defer to.

7. **No edit to other commands or playbooks in this study.**
   `wave-1-session-loop.md`, `wave-3-session-loop.md`,
   `acceptance-criteria.md`, `wave-3-acceptance-criteria.md`,
   `feedback-protocol.md`, `/resolve-b0`, `/critique`,
   `/sync-agents`, `/check-decision-backlog` stay as-is. The
   post-Wave-3 loop cites them; it does not rewrite them.

### What this commits the harness to

- **Phase awareness:** the harness now has a post-Wave-3 loop and
  a phase-aware promotion command. B2 and B3 sessions stop
  manually re-deriving the discipline.
- **PR-flow as discipline:** the `session-governance` skill
  loads on branch / commit / PR-shaped trigger phrases (per
  Artifact #1); `/open-pr` runs the PR-opening checklist; the
  hardened settings file enforces the discipline mechanically;
  `CONTRIBUTING.md` is the upstream contract all three defer
  to. Four layers (CONTRIBUTING + skill + command + settings);
  none of them alone is sufficient; together they make the
  discipline hard to skip and updates land in one place.
- **ADR collision prevention:** `/promote-to-adr` has an explicit
  reservation step. Two parallel sessions cannot both compute
  the same `<NNNN>` silently.

---

## Consequences

1. **Six items land — three new files, one settings hardening,
   two edits — replacing five ad-hoc disciplines.** Same
   accounting as §Recommendation lead-in: three new files
   (skill, playbook, command); one settings hardening
   (`.claude/settings.local.json`); two edits (one to
   `promote-to-adr.md`; one to `CONTRIBUTING.md` — new
   post-Wave-3 section per D7). Total surface increase: one
   skill directory + two new markdown files + three modified
   files — small enough to land in a single follow-on PR, but
   the PR sequence matters (see point 9).

2. **The PR-flow discipline becomes auditable across sessions.**
   Today, whether a session followed the discipline is judged by
   reading the PR. After this study lands, the
   `session-governance` skill is the audit reference — any
   session that violates branch-before-first-commit,
   `gh pr merge`-prohibition, or `--no-verify`-prohibition is
   in measurable violation of a named skill.

3. **The settings file matches the stated contract.** The harness
   no longer rubber-stamps `gh pr merge`; the operator must
   explicitly approve at runtime. `--no-verify` cannot be silently
   passed. Wave-0 / post-Wave-3 Make targets no longer generate
   permission prompts. D3 (§Decision Drivers) is satisfied.

4. **The post-Wave-3 loop preserves the Wave-1 loop's discipline.**
   The new loop is shape-isomorphic to
   [`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md):
   ten steps; [H] decision points at the same positions; critique
   preservation per
   [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md);
   acceptance criteria
   ([`acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md))
   AC-1 through AC-10 still gate every study. Contributors moving
   from Wave-1 work to B3 work read the same loop they already
   know.

5. **One-line edits prevent silent collisions.** The reservation
   step in `/promote-to-adr` is operator-side, but the command
   makes the step explicit. A future contributor reading the
   command knows the reservation is required; an agent running
   the command cannot skip it without the operator noticing.

6. **Future B3-N tooling sessions follow the same path.** This
   study is itself a B3-N entry in the "Tooling extensions"
   family. The path it walks
   (study → `/critique` → operator acceptance → promotion via
   PR; the new `post-wave3-session-loop.md` is the loop) becomes
   the path for the next harness extension — whether that is
   adding `/resolve-b2` and `/resolve-b3` (OQ-1), extending
   `/sync-agents` to cover commands (OQ-2), or any other tooling
   gap that surfaces under demand-driven pacing.

7. **No platform code changes; no rule changes.** R1 (no
   production code during studies) is satisfied; this is a
   pure-tooling B3-N entry. The new artifacts ship under
   `.claude/` only; the platform's runtime, rules, and tooling
   modules (`engine/`, `rules/`, `tools/`, `deploy/`) are
   unchanged.

8. **Borderline eligibility interpretation is a precondition,
   not an Open Question.** §Decision Drivers D0 acknowledges
   that the "Tooling extensions" family was canonically
   illustrated with platform-runtime examples in ADR-0049
   §Per-family scope. The entire §Recommendation above is
   conditional on `/critique` clearing the "adjacent tooling =
   agent-harness" reading before the study moves to
   `resolved-study`. If the ruling goes
   against the reading, the study re-scopes — either as an
   ADR-0049 amendment proposing a new B3 sub-family for
   agent-harness extensions, or as a Wave-2-style consolidated
   decision outside B3. The interpretation does not silently
   land; the precondition is the load-bearing review point.

9. **Promotion of this study uses the *unmodified*
   `/promote-to-adr`.** The study modifies `/promote-to-adr`
   (Artifact #5 — drop Wave-1 gate; add ADR-number reservation
   step). The promotion of this study itself runs against the
   *current* version of the command, not the modified one —
   the modified version only takes effect after this study's
   own promotion PR merges. The operator-side reservation
   discipline proposed in #5 applies to this study informally
   (the provisional `0051` number declared in §Metadata
   Promotion target is the reservation); the formal command
   step lands one cycle later. Future B3-N promotions after
   this one is merged use the modified command.

   This is an instance of a self-modifying tool bootstrapping
   itself: the first application of the new contract is
   informal because the contract is what's being installed.
   Subsequent applications are formal.

---

## Open Questions

- **OQ-1 — `/resolve-b2` and `/resolve-b3`: separate commands, or
  one `/resolve-b <tier> <slug>` parametrized command?**
  *Out-of-scope for current cycle.* Resolved by the **second**
  post-Wave-3 B2 or B3 session following the new
  `post-wave3-session-loop.md` — when the divergence between
  B2 grounding (originating wave; e.g., ADR-0035-style
  upstream) and B3 grounding (ADR-0049 eligibility filter)
  becomes empirically visible across at least two sessions.
  Until then, the current `/resolve-b0` plus manual re-use is
  acceptable for one more cycle.

- **OQ-2 — Does `/sync-agents` need to cover the slash-command
  inventory and the skill list, not just CLAUDE.md /
  AGENTS.md / .codex/AGENTS.md?**
  *Out-of-scope for current cycle.* Today `/sync-agents` (per
  [`.claude/commands/sync-agents.md`](../../.claude/commands/sync-agents.md)
  lines 17–25) detects drift in the hard-rules section, the
  principles section, the slash-command **list**, the waves
  narrative, the path-header convention, and the
  required-reading list. The slash-command list is already in
  scope; whether the skill list and the playbook list join it
  is a follow-on tooling decision. The first new contributor
  who has to learn the new artifacts will reveal whether the
  drift-check needs to extend. Until then, the contributor
  reads `.claude/skills/` and `.claude/playbooks/` directly.

- **OQ-G3.1 — Force-push denies in `settings.local.json`.**
  *Out-of-scope for current cycle.* Adding `git push --force`
  and `git push -f` to the deny block is a related hardening
  but exceeds the operator-enumerated G3 scope
  (`gh pr merge` restriction, `--no-verify` negation, missing
  Make targets). Force-push to `main` is already covered by
  [`CLAUDE.md`](../../CLAUDE.md) §"Executing actions with
  care" as a destructive operation requiring confirmation.
  Resolved by a future post-Wave-3 tooling session paced by
  concrete operator signal (a near-miss force-push, or a
  pattern across sessions). Until then, the in-session
  confirmation gate is the safeguard.

---

## Promotion target

`docs/adr/0051-claude-tooling-postwave3.md` — provisionally
the next available number at the time of writing (last landed
is [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md),
2026-05-29). Subject to the
[ADR-0020](../../docs/adr/0020-wave-s-launch.md) §"Per-item
ADR numbering" caveat per
[`adr-writing`](../../.claude/skills/adr-writing/SKILL.md) A8:
the expected sequence is descriptive, not reserved. If
B-2a / B-2b (or any other in-flight study) is promoted before
this one lands, the number slides. The G5 reservation
mechanism in §Recommendation #5 is the procedural backstop
for parallel-PR collision; for this study's own promotion the
backstop is informal (per §Consequences point 9 —
self-modifying tool bootstrap).
