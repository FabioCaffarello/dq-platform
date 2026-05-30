<!-- path: docs/adr/0051-claude-tooling-postwave3.md -->

# ADR-0051 — Claude Tooling Post-Wave-3 Extension

- **Status:** accepted
- **Date:** 2026-05-29

---

## Context

Waves 1, 2, and 3 closed on 2026-05-21 / 2026-05-21 / 2026-05-23.
Wave-S launched 2026-05-23; its foundational triplet
([ADR-0021](./0021-mode-as-primitive.md),
[ADR-0022](./0022-kind-catalog.md),
[ADR-0023](./0023-sources-schema.md)) reached `resolved-adr` by
2026-05-24. [ADR-0049](./0049-b3-evolutionary-launch.md) launched
B3 on 2026-05-29 as a structural peer of those waves —
a demand-driven evolutionary lane restricted to three families
(kind, capability mode, tooling extensions) and conformant to the
ADR-0020 / 0021 / 0022 / 0023 envelope. The platform's operating
cadence from that point forward is the **post-Wave-3 evolutionary
lane**, landing through pull requests against `main` per the
PR-flow contract committed by
[`.claude/playbooks/wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
step 10.

The `.claude/` agent harness — the commands, playbooks, skills,
and settings file that govern how contributors (human and agent)
produce every study, ADR, scaffold, and PR in this repository —
was written for an earlier operating shape and still reads as if
Wave 1, Wave 2, and Wave 3 were the active modes. Five gaps
surface when a contributor opens a session today:

1. **Phase lacuna.**
   [`.claude/commands/promote-to-adr.md`](../../.claude/commands/promote-to-adr.md)
   gates on "every B0 row at `resolved-study` or `resolved-adr`"
   plus the Wave-2 consolidated document — a Wave-1 gate that
   has always passed since 2026-05-21. There is no session-loop
   playbook for B2 or B3 and no `/resolve-b2` or `/resolve-b3`
   command; B2 and B3 sessions have been manually reusing
   `/resolve-b0`.
2. **PR-flow buried in a playbook step.** The branch-then-PR
   discipline (create the branch before the first commit;
   confirm branch; never commit on `main`; open the PR and
   **stop** without merging) lives only in
   [`.claude/playbooks/wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
   step 10. Sessions outside Wave 3 re-derive the discipline
   from the playbook by hand each session.
3. **Settings posture too broad.** The current
   [`.claude/settings.json`](../../.claude/settings.json)
   allows `Bash(gh pr *)`, which transitively authorizes
   `gh pr merge` — the operation `wave-3-session-loop.md` step 10
   explicitly reserves for the `[H]` reviewer after CI passes.
   There is no deny entry for `--no-verify` (which the harness
   would otherwise honor) and the post-Wave-3 Make targets are
   not covered.
4. **No skill for cross-cutting session governance.** The three
   existing skills
   ([`adr-writing`](../../.claude/skills/adr-writing/SKILL.md),
   [`critique-anti-patterns`](../../.claude/skills/critique-anti-patterns/SKILL.md),
   [`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md))
   encode authoring, review, and coding conventions. None
   encodes the cross-cutting session-governance discipline that
   every session needs (branch-before-first-commit;
   confirm-not-main; never `gh pr merge`; never `--no-verify`;
   cite the ADR or B-row in every produced artifact; one topic
   per session).
5. **ADR number unreserved at promotion time.** The current
   `/promote-to-adr` derives the next available number at write
   time. Under direct-to-main commits this was a non-issue.
   Under post-Wave-3 PR-flow with parallel branches in flight,
   two simultaneous promotions can both compute the same
   `<NNNN>` and collide when the second PR merges. The current
   command has no reservation step.

This ADR closes the five gaps with the minimum-viable intervention
per gap. Two cross-cutting positions also land here:

**The eligibility reading admitting agent-harness as "adjacent
tooling".**
[ADR-0049](./0049-b3-evolutionary-launch.md) §Per-family scope
defines the **Tooling extensions** family as "additions to
`tools/lint/`, the manifest publisher, the dry-run runner, the
engine dispatcher, **and adjacent tooling** that extend contract
coverage without changing the contract shape." The canonical
examples in that clause are platform-runtime tools; admitting the
agent harness — which extends the *governance* contract's coverage
(the draft → critique → accept → promote loop's discipline)
without reshaping it — is an expansive reading. **The
agent-harness-as-adjacent-tooling reading this ADR commits is
new contribution requiring review** and is reviewed in this ADR.
Future B3-N tooling sessions targeting agent-harness extensions
can cite ADR-0051 as precedent; future sessions targeting
adjacent-tooling families not yet committed (e.g., reviewer
tooling, observability harness) must follow the same
new-contribution discipline rather than absorb the reading
silently.

**`CONTRIBUTING.md` as the upstream authority for PR-flow.**
The branch-then-PR contract today lives in `CONTRIBUTING.md`
scoped to Wave-3 only. The post-Wave-3 evolutionary lane needs
the same contract generalized. This ADR commits `CONTRIBUTING.md`
as the upstream documentary surface for PR-flow; the four
in-harness layers (skill, command, settings, playbook) defer to
it. Updates to the PR-flow contract land in `CONTRIBUTING.md`
first; the harness layers follow.

The principles bearing on this decision are **P5** (evolution
must be contract-driven — the `.claude/` harness is itself a
contract with the contributor population, and it evolves under
a published shape just like every other artifact) and **P6**
(borrow patterns, not baggage — every proposed artifact is
defended on the fit to this project's workflow, not on
resemblance to external agent-harness convention). **R4** in
[`CLAUDE.md`](../../CLAUDE.md) §3 (one topic per session) is
load-bearing for the `CONTRIBUTING.md` boundary: the Wave-3
section text is **not** rewritten; the post-Wave-3 section is
added beside it.

---

## Decision

### Clause 1 — Eligibility reading

`B3-1` qualifies under [ADR-0049](./0049-b3-evolutionary-launch.md)
§(a) as a **Tooling extensions** entry on the expansive reading
of "adjacent tooling" introduced in §Context. The four eligibility
conditions hold:

1. **Expands not rewrites** (P-B3.1) — the four-step
   draft → critique → accept → promote loop stays unchanged in
   shape; the `.claude/` harness extends its coverage of the
   post-Wave-3 lane without reshaping it.
2. **In-scope family** (P-B3.4) — Tooling extensions, on the
   expansive "adjacent tooling" reading committed in §Context
   and marked there as new contribution requiring review.
3. **Conforms to envelope** (P-B3.2) — this ADR touches no
   substrate decision, no mode primitive, no kind catalog entry,
   no sources schema row. ADR-0020 / 0021 / 0022 / 0023 are
   unaffected.
4. **Crosses additive-maintenance threshold** — the five gaps
   reshape contributor expectations (a first-class PR-flow skill,
   a phase-aware promotion gate, a settings posture that no
   longer authorizes merge, an upstream-authority layer in
   `CONTRIBUTING.md`); the work is material enough to warrant a
   study trail and an ADR above a routine catalog PR.

### Clause 2 — `CONTRIBUTING.md` as upstream authority for PR-flow

`CONTRIBUTING.md` is the documented contract surface for the
branch-then-PR discipline across **all** post-Wave-1 work
(Wave 3 scaffolding, B2 / B3 evolutionary lane, amendments).
A new section in `CONTRIBUTING.md` — added beside the existing
Wave-3 section, not replacing it — generalizes the PR-flow
contract from `wave-3-session-loop.md` step 10 to cover the
post-Wave-3 evolutionary lane. The in-harness layers (skill,
command, settings, playbook) committed by Clauses 3–7 below
read from `CONTRIBUTING.md` as the authority; updates to the
PR-flow contract land in `CONTRIBUTING.md` first and the
harness layers follow.

The provisional branch-slug forms operators currently pass
(`chore/`, `feat/`, `docs/decision/`, `docs/adr/`) are recorded
in the new `CONTRIBUTING.md` section as **new contribution
requiring review** (R5) pending operator ratification. Until
that review lands, the slugs stay provisional; the playbook
and the skill reference the section by anchor rather than
re-stating the slugs.

### Clause 3 — Session-governance skill

A new skill ships at `.claude/skills/session-governance/`
matching the shape of the three existing skills (frontmatter
`name` + `description`; `SKILL.md` + `reference/`; file:line
citations to upstream sources). The skill encodes the
cross-cutting session-governance discipline:

- create the feature branch before the first commit using the
  slug provided by the operator (provisional per Clause 2);
- confirm the branch is not `main` via
  `git branch --show-current` (fallback
  `git rev-parse --abbrev-ref HEAD`);
- never call `gh pr merge` (reserved for the `[H]` reviewer);
- never pass `--no-verify` to `git commit` or `git push`;
- cite the ADR or B-row in every produced artifact;
- stay inside R4 (one topic per session).

The skill's description triggers on verbatim contributor
phrases at the load-bearing moments — "create a branch",
"git switch", "git checkout -b", "first commit", "open a PR",
"gh pr create", "merge the PR". The description does **not**
rely on a hypothetical "session open" trigger; the existing
three skills load on description-match basis only, and the
verbatim-phrase trigger shape committed here is **new
contribution requiring review** pending verification that the
phrases reliably load the skill in practice.

`CONTRIBUTING.md` is the upstream authority the skill reads
from per Clause 2.

### Clause 4 — Post-Wave-3 session-loop playbook

A new playbook ships at
`.claude/playbooks/post-wave3-session-loop.md`, mirroring
[`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
in shape: ten steps with `[H]` decision points at the same
positions; draft → `/critique` preserved per
[ADR-0048](./0048-critique-rounds-preservation.md) →
revision → log update. Two deltas from the Wave-1 loop:

- **Step 2 (grounding) adds an
  [ADR-0049](./0049-b3-evolutionary-launch.md) §(a)
  eligibility check** before the agent proceeds. The
  eligibility-check step in the loop shape is **new
  contribution requiring review** — no existing loop carries
  one. If the check fails, the session aborts.
- **Step 10 (commit) is replaced by the PR-flow contract
  authoritative in `CONTRIBUTING.md`** per Clause 2 — create
  the feature branch using the operator-provided slug;
  confirm the branch; stage; commit; push; `gh pr create`;
  **stop** without merging.

### Clause 5 — `/open-pr` command

A new command ships at `.claude/commands/open-pr.md`. The
command runs the PR-opening checklist:

- confirm the current branch is not `main`;
- show `git log main..HEAD` so the operator sees the staged
  commits;
- run the gates applicable to the unit (`make lint-*`,
  `make test-*`, `make validate-deploy` — enumerated by the
  `session-governance` skill);
- run `gh pr create --base main` with a PR body that lists
  the citation map (ADR / B-row references), the critique
  result, and a test plan;
- **stop**; never call `gh pr merge`.

The command reads from `CONTRIBUTING.md` per Clause 2; the
checklist mirrors the upstream contract documented there.

### Clause 6 — Hardened `.claude/settings.json`

The settings file is hardened so the harness's enforcement layer
matches the stated discipline. Three changes:

1. **`Bash(gh pr *)` is replaced by an explicit allowlist** of
   `Bash(gh pr create *)`, `Bash(gh pr view *)`,
   `Bash(gh pr list *)`, `Bash(gh pr diff *)`,
   `Bash(gh pr checks *)`. `gh pr merge` is moved to deny.
2. **A `deny` block is introduced** with entries for
   `Bash(gh pr merge *)`, `Bash(git commit --no-verify*)`, and
   `Bash(git push --no-verify*)`. The `deny` block and its
   per-call-site `--no-verify` entries are **new contribution
   requiring review** pending harness-docs verification of
   `deny` semantics; the per-call-site coverage catches
   `--no-verify` only when it appears immediately after the
   command prefix (`git commit --no-verify ...` /
   `git push --no-verify ...`). Cases where `--no-verify`
   follows other arguments are **not** caught by these
   entries; the in-session confirmation gate remains the
   safeguard. Tightening to a substring pattern (`*` embedded
   in the argument) is parked until harness-docs are checked.
3. **Post-Wave-3 Make targets are added explicitly** —
   `make test-tools *`, `make test-engine-sandbox *`,
   `make validate-deploy *`, `make demo-p6 *`, and the
   read-only `git branch --show-current` /
   `git rev-parse --abbrev-ref HEAD` /
   `git log main..HEAD` invocations the new playbook uses.

R6 path-header convention does not apply to the settings file
(JSON has no comment syntax; R6 governs markdown only).

### Clause 7 — `/promote-to-adr` edits

[`.claude/commands/promote-to-adr.md`](../../.claude/commands/promote-to-adr.md)
is edited to drop the stale Wave-1 gate and to add a number
reservation step.

- **Wave-1 gate replaced.** The current gate ("every B0 row at
  `resolved-study` or `resolved-adr` plus the Wave-2
  consolidated document") has always passed since 2026-05-21
  and now produces a noise-only check. The replacement is a
  post-Wave-3 recognition step: confirm the study being
  promoted carries `Status: resolved-study`, that its
  `Promotion target` line names a concrete
  `docs/adr/<NNNN>-<slug>.md` filename, and that the promotion
  is happening under post-Wave-3 PR-flow per
  `CONTRIBUTING.md`. The historical Wave-1 gate remains
  referenced in the command's body as a one-line audit note.
- **Reservation step added.** Before writing the ADR file, the
  agent posts a message naming the proposed `<NNNN>` and pauses
  for operator acknowledgement. The operator confirms the
  number or supplies a different one. The reservation is
  operator-side; the command makes the step explicit so it
  cannot be skipped silently. This closes the parallel-PR
  collision risk that surfaced as B3-1 ran in parallel with
  B2-20.

### Clause 8 — `B3-N` labeling precedent (flat)

This ADR — promoted from `B3-1` — sets the B3-N labeling
precedent committed by
[ADR-0049](./0049-b3-evolutionary-launch.md) §Consequence 7:
**B3-N rows are flat (`B3-N`), not family-prefixed.** Future
B3-N rows continue the flat sequence. Family-prefixing
(`B3-K-N` / `B3-M-N` / `B3-T-N`) remains available as a
future ADR-0049 amendment if volume justifies it; until then,
no harness or registry layer treats family-prefix as a
recognized B3-row shape.

### Clause 9 — Self-modifying bootstrap

This ADR is promoted via the **unmodified** version of
`/promote-to-adr` — the version Clause 7 modifies takes effect
only after this ADR and its implementation slice land.
Operator-side reservation discipline applied informally to
this promotion: the provisional `0051` number declared by the
study at registration time is the reservation. Subsequent
B3-N promotions use the modified command with the formal
reservation step.

The expansive eligibility reading committed by Clause 1 is
itself a self-modifying contract: the agent harness extends
its own coverage to a category (agent-harness tooling) that
ADR-0049's canonical examples did not enumerate. The
new-contribution markers on Clauses 1, 2, 3, 4, and 6 are the
review surface for that expansion.

---

## Consequences

1. **Six items land in the follow-on implementation slice.**
   Three new files (the `session-governance` skill, the
   `post-wave3-session-loop.md` playbook, the `/open-pr`
   command); one settings hardening
   (`.claude/settings.json`); two edits (one to
   `.claude/commands/promote-to-adr.md`; one to
   `CONTRIBUTING.md` — new post-Wave-3 section per Clause 2).
   The implementation slice is bounded; no engine, rules,
   tools, or deploy code changes.

2. **PR-flow discipline becomes auditable across sessions.**
   The `session-governance` skill (Clause 3) is the in-session
   reference; `/open-pr` (Clause 5) is the procedural reference;
   `.claude/settings.json` (Clause 6) is the mechanical
   reference; `CONTRIBUTING.md` (Clause 2) is the upstream
   contract. Four layers; none alone is sufficient; together
   they make the discipline hard to skip. A session that
   violates branch-before-first-commit, `gh pr merge`
   prohibition, or `--no-verify` prohibition is in measurable
   violation of a named clause.

3. **The settings file matches the stated contract.** The
   harness no longer rubber-stamps `gh pr merge`; the operator
   must explicitly approve it at runtime. `--no-verify` cannot
   be passed silently in the two call sites Clause 6 covers.
   Post-Wave-3 Make targets no longer generate permission
   prompts.

4. **`B3-N` labeling is fixed by precedent.** Future B3-N
   contributors find the flat-labeling convention in the B3
   section preamble of the decision log; no per-row decision
   is required. If volume later justifies family-prefixing,
   the change is a single ADR-0049 amendment, not a
   per-row migration.

5. **`/promote-to-adr` collisions cannot land silently.**
   Clause 7's reservation step makes the step explicit; two
   parallel sessions cannot both compute the same `<NNNN>`
   without operator acknowledgement.

6. **The Wave-1 session-loop playbook stays preserved as
   history.** The new post-Wave-3 loop (Clause 4) is
   shape-isomorphic to the Wave-1 loop; the Wave-1 loop is
   not edited. Contributors moving from Wave-1 work to B3
   work read the same loop shape.

7. **The author-equals-reviewer circularity is recognized as a
   structural risk of single-agent eligibility rulings.** The
   D0 ratification protocol applied at B3-1's resolution
   (round-2 critique emits a substantive ruling; operator
   ratifies explicitly; the ratification is the load-bearing
   decision, not the critique output) is registered here as
   operator-side discipline for future borderline B3-N
   eligibility readings under
   [ADR-0049](./0049-b3-evolutionary-launch.md) §(a). No
   playbook edit is committed by this ADR; the discipline is
   operator-side and the next borderline B3-N reveals whether
   it needs codification.

8. **No engine / rules / tools / deploy code changes.** This
   ADR is a pure-tooling B3-N entry at the documentation and
   harness layers; the platform's runtime is unaffected.

9. **Future agent-harness B3-N entries cite ADR-0051 as
   precedent.** The expansive reading of "adjacent tooling"
   that admits agent-harness tooling under ADR-0049 is now
   committed. Future B3-N rows for adjacent-tooling families
   not yet committed (reviewer tooling, observability harness)
   follow the same new-contribution-requiring-review
   discipline rather than absorb the reading silently.

---

## Notes

Three open questions remain explicitly out-of-scope for this
ADR; each has a named trigger condition:

- **OQ-1 — `/resolve-b2` and `/resolve-b3`: separate commands
  vs. one `/resolve-b <tier> <slug>` parametrized command.**
  Resolved by the second post-Wave-3 B2 or B3 session
  following the new `post-wave3-session-loop.md`, when the
  divergence between B2 grounding (originating wave) and B3
  grounding ([ADR-0049](./0049-b3-evolutionary-launch.md) §(a)
  eligibility) becomes empirically visible across at least
  two sessions. Until then,
  [`.claude/commands/resolve-b0.md`](../../.claude/commands/resolve-b0.md)
  plus manual re-use is acceptable.

- **OQ-2 — `/sync-agents` coverage for skills, playbooks, and
  command inventory.** Today
  [`.claude/commands/sync-agents.md`](../../.claude/commands/sync-agents.md)
  detects drift in the hard-rules section, the principles
  section, the slash-command list, the waves narrative, the
  path-header convention, and the required-reading list. The
  slash-command list is already in scope; whether the skill
  list and the playbook list join it is a follow-on tooling
  decision. The first new contributor learning the artifacts
  committed by this ADR reveals whether the drift-check needs
  to extend.

- **OQ-G3.1 — Force-push deny entries in
  `.claude/settings.json`.** Adding `git push --force`
  and `git push -f` to the deny block is a related hardening
  but exceeded the original gap scope this ADR addresses.
  Force-push to `main` is already covered by
  [`CLAUDE.md`](../../CLAUDE.md) §"Executing actions with
  care" as a destructive operation requiring confirmation.
  Resolved by a future post-Wave-3 tooling session paced by
  concrete operator signal (a near-miss force-push, or a
  pattern across sessions); until then, the in-session
  confirmation gate is the safeguard.

---

## Amendment log

- **2026-05-30** — In-place naming-drift correction. Five
  citations of the harness settings file in this ADR (Context
  Gap 3, Clause 6 heading, Consequence 1, Consequence 2,
  OQ-G3.1) were updated from `.claude/settings.local.json` to
  `.claude/settings.json`. The shared team posture is held in
  the tracked `.claude/settings.json`; `.claude/settings.local.json`
  is per-user local state (gitignored at `.gitignore:25` —
  "Claude Code per-user local state (the rest of `.claude/` is
  tracked)") and cannot carry a contract that binds all
  contributors. The actual file `.claude/settings.json` already
  carried the full Clause 6 hardening at the time of this
  amendment: explicit `gh pr create / view / list / diff / checks`
  allowlist; deny block for `gh pr merge`, `git commit --no-verify`,
  `git push --no-verify`; and the post-Wave-3 Make targets
  (`make test-tools`, `make test-engine-sandbox`,
  `make validate-deploy`, `make demo-p6`) plus the read-only
  `git branch --show-current` / `git rev-parse --abbrev-ref HEAD`
  / `git log main..HEAD` invocations. Permission semantics
  unchanged; downstream artifacts
  ([`.claude/skills/session-governance/SKILL.md`](../../.claude/skills/session-governance/SKILL.md),
  [`.claude/commands/open-pr.md`](../../.claude/commands/open-pr.md),
  [`.claude/skills/session-governance/reference/governance-checklist.md`](../../.claude/skills/session-governance/reference/governance-checklist.md))
  already cite `.claude/settings.json` and are unaffected.
  Landed under [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6
  as an operator-authorized direct edit (satellite catching up
  to source of truth — here the ADR is the satellite that
  drifted from the actual artifact). Amendment-log shape
  follows the convention committed by
  [ADR-0050](./0050-v1-retirement-engine-release.md)
  §Consequence 4.
