<!-- path: studies/decisions/2026-05-30-amendment-adr-0015-single-user-codeowners.md -->

# Amendment to ADR-0015 — Single-User CODEOWNERS Model

## Metadata

- Date: 2026-05-30
- Status: draft
- **Classification: Amendment to [ADR-0015](../../docs/adr/0015-codeowners.md) per [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §(a)** — this proposal modifies what ADR-0015 decided (the asymmetric review model backed by GitHub-org team groups). Not B3 (Condition 1, P-B3.1, fails: this is rewrite, not expansion). Not Flow 6 (substantive contract change, not factual refresh). The standalone-amendment idiom from `adr-writing` skill A4 applies: the promotion ADR carries `Status: accepted (amends ADR-0015)`; ADR-0015 itself stays accepted with its Status line updated to record the amendment relationship.
- Promotion target: [`docs/adr/0057-single-user-codeowners-amendment.md`](../../docs/adr/0057-single-user-codeowners-amendment.md) (provisional; operator reserves at promotion time per [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md) Clause 7; subject to the same numbering caveat per `adr-writing` A8).
- Critique rounds:
  - Round 1 — [capture](../critiques/2026-05-30-amendment-adr-0015-single-user-codeowners-critique-1.md) (0 blocking / 3 important / 5 minor); 3 important applied in this revision (AC-10 vacuous-satisfaction note added to Consequence #9; Consequence #6 Flow 5 refresh path committed as Flow 6 with PR-shape pushed to new OQ-5; OQ-1 recommendation header reframed as `Author's recommendation (NOT pre-decided; surfaced for operator ratification...): STAYS`); 5 minor deferred under the two-round cap.

---

## Context

The repository is and will remain under the personal user
`@FabioCaffarello` on GitHub. There is no `@dq-platform` (or
equivalent) GitHub org, and there will not be one. ADR-0015
committed an asymmetric review model that depends on **GitHub-org
team groups** for enforcement — a model that has no enforcement
substrate in a single-user repository.

### Load-bearing clauses from ADR-0015

ADR-0015 §2 ("Group inventory — three groups") commits the
three-group inventory verbatim:

> The review-ownership map uses exactly three groups. Each
> satisfies a distinct obligation from the prior ADRs.
>
> | Group placeholder | Responsibility |
> |---|---|
> | `@PLACEHOLDER-org/platform-team` | Default owner for everything not explicitly delegated … |
> | `@PLACEHOLDER-org/sre` | Co-owner of `/deploy/overlays/qa/` and `/deploy/overlays/prod/` … |
> | `@PLACEHOLDER-org/rules-authors` | Per-entity rule YAMLs under `/rules/` … |

ADR-0015 §4 ("Substitution protocol") commits that the placeholder
literal `PLACEHOLDER-org/` will be exchanged for the real org slug
in a mechanical, atomic per-file edit, *"in the operational
session that creates the production GitHub org per the ADR-0008
host-primitive follow-up"*.

ADR-0015 §Consequence 1 commits the enforcement mechanism:

> A merge request that edits the schema source but not the schema
> mirror fails the byte-equality CI gate committed by ADR-0001
> **and requires the same review group on both files** — there is
> no asymmetric-approval path through which one half can land
> without the other.

ADR-0015 §Consequence 5 commits the SRE depth on production
overlays:

> `/deploy/overlays/qa/` and `/deploy/overlays/prod/` … their
> CODEOWNERS lines list both `@platform-team` and `@sre`, so
> GitHub branch-protection's "review from CODEOWNERS"
> requirement gates them on both groups.

These three commitments — the three-group inventory, the
substitution-to-real-org protocol, and the GitHub branch-
protection-anchored enforcement — are the precise commitments
this amendment modifies.

### Load-bearing references to the same model from the repo's other artifacts

**`/.github/CODEOWNERS`** (the file ADR-0015 committed verbatim)
carries thirty-plus rules of the form
`/<path>/  @PLACEHOLDER-org/<group>`. The default rule
(`*  @PLACEHOLDER-org/platform-team`) plus the overrides for
`/rules/_schema/`, `/rules/_owners.yaml`,
`/deploy/overlays/{qa,prod}/`, `/CLAUDE.md`, etc., are all
expressed against the three placeholder team groups.

**`rules/_owners.yaml`** carries two entities (`customer`,
`orders_stream`); both name `@PLACEHOLDER-org/rules-authors` as
the owner. The linter (`dq-lint`, per ADR-0006 §9 and ADR-0015
§Consequence 4) requires every `owner` value to resolve to a
CODEOWNERS group.

**`CONTRIBUTING.md` Flow 5** commits the PR-close discipline:

> Once CI is green and the **[H]** reviewer approves, merge via
> the GitHub UI. **The agent never calls `gh pr merge`.**

The `[H]` reviewer is a human operator; in the original
multi-author shape ADR-0015 imagined, that human is somebody
other than the PR author drawn from a CODEOWNERS-routed team.

**`.claude/settings.json`** carries the deny block:

```json
"deny": [
  "Bash(gh pr merge *)",
  "Bash(git commit --no-verify*)",
  "Bash(git push --no-verify*)"
]
```

The `gh pr merge` deny entry is the mechanical guard that
backs CONTRIBUTING.md Flow 5's *"agent never calls `gh pr
merge`"* clause.

**`CLAUDE.md` P3** commits the principle the whole review
model implements:

> **P3. Ownership must be explicit everywhere.** No entity
> without owner, no alert without route, no repository area
> without policy.

---

## Decision Drivers

- **DD-1 — The substrate ADR-0015 commits against does not
  exist and will not.** ADR-0015 §4 commits the substitution to
  happen *"in the operational session that creates the production
  GitHub org"*. That session is now declared structurally
  out-of-scope by operator decision: the repo stays under
  `@FabioCaffarello` (personal user). The `PLACEHOLDER-org/`
  literal has no resolution; the substitution protocol's
  precondition fails permanently.
- **DD-2 — GitHub personal accounts have no team groups.**
  `@user/team` syntax is GitHub-org-only; personal accounts
  cannot create or own team groups. There is no
  `@FabioCaffarello/platform-team` (or equivalent) to which the
  three ADR-0015 groups can be mechanically substituted.
- **DD-3 — Single-author repos cannot self-approve.** GitHub's
  branch-protection rule *"Require review from CODEOWNERS"*
  requires a review from a CODEOWNERS-listed account that is
  not the PR author. In a single-user repo where the listed
  owner is the author, the rule blocks every PR. ADR-0015
  §Consequence 5's *"GitHub branch-protection's 'review from
  CODEOWNERS' requirement gates them on both groups"* clause has
  no satisfiable evaluation under the single-user model.
- **DD-4 — P3 commits ownership-as-fact, not
  ownership-as-mechanism.** *"Ownership must be explicit
  everywhere"* is satisfied by a single owner pointing at every
  path; what fails under the single-user model is the
  **asymmetric-review-as-enforcement** layer ADR-0015 built on top
  of P3.
- **DD-5 — `_owners.yaml` owner values are consumed by the
  alerting consumer at runtime, not by GitHub CODEOWNERS.**
  ADR-0006 §"no alert without owner" routes alerts based on the
  per-entity `channels` block plus the `owner` string;
  CODEOWNERS-group resolution is a lint-time defense-in-depth
  check, not the runtime contract. The two surfaces are
  separable.

---

## Considered Options

### Option A — Owner-single model (operator's stated direction)

Every CODEOWNERS rule names `@FabioCaffarello` as the owner.
The three placeholder groups (`@PLACEHOLDER-org/platform-team`,
`@PLACEHOLDER-org/sre`, `@PLACEHOLDER-org/rules-authors`)
collapse into one user. Asymmetric review is removed; the file
becomes a formality — it validates against GitHub's
CODEOWNERS-syntax rules and lets GitHub auto-assign the user as
reviewer on PR open, but it gates no real review separation
because the user is also every PR's author.

`_owners.yaml` owner values stay **semantic** in the sense the
alerting consumer cares about: `owner: "@FabioCaffarello"`
identifies the single human responsible. The lint-time
cross-check that *"every `owner` resolves to a CODEOWNERS
group"* (ADR-0006 §9) needs to be relaxed or re-stated against
the single-user shape — see Consequence #5.

The GitHub branch-protection rule *"Require review from
CODEOWNERS"* MUST be **off** under this option (or
self-approval must be enabled, where supported), because the
PR author is the only listed owner and cannot self-approve by
default.

This is the operator's decided direction (ratify-pending).

### Option B — Retain ADR-0015 placeholder shape, defer substitution indefinitely

Keep `PLACEHOLDER-org/` literals in place; document that the
substitution session is reserved without commitment. ADR-0015's
contract stays as-written; CODEOWNERS doesn't enforce anything
in practice (no real org exists for the placeholders to resolve
to); the contradiction sits unaddressed.

This is the status quo, and the contradiction it carries is the
reason this amendment exists. Rejected because it leaves
ADR-0015's commitments (group inventory, substitution protocol,
enforcement gates) read against a substrate that does not exist
— a permanent disagreement between the committed contract and
the operational reality.

### Option C — Remove `/.github/CODEOWNERS` entirely

Delete the file; remove the lint-time cross-check from
ADR-0006 §9 that requires `owner` strings to resolve to a
CODEOWNERS group; remove the supporting clauses from ADR-0015.

Rejected for three reasons. First, the file still has
**documentation value** even without enforcement — operators
reading `/.github/CODEOWNERS` learn which workspace paths the
project considers owned and at what review depth, the same way
they read ADR-0015. Second, GitHub auto-assigns the listed
owner on PR open, which is a small but real PR-routing
convenience even at single-user scale. Third, deleting the file
forecloses a future-org migration that cannot be ruled out — if
the project ever moves to an org, restoring the CODEOWNERS
shape from scratch costs more than amending it now and keeping
the formality.

### Option D — Personal-account team groups (technically not an option)

GitHub personal accounts cannot create team groups; the
`@user/team` syntax is org-only. There is no construction under
the single-user model that preserves the three-group inventory
mechanically.

---

## Recommendation

**Option A — owner-single model.** The repository stays under
`@FabioCaffarello`; `/.github/CODEOWNERS` is rewritten so every
rule names that single user; `rules/_owners.yaml`'s `owner`
values are updated to match.

The amendment promotion ADR (provisionally ADR-0057) commits
the contract change; the file substitution (rewriting
`/.github/CODEOWNERS` + `rules/_owners.yaml`) is the
implementation slice that the amendment authorizes. Whether
the substitution lands in the same PR as the promotion ADR
(R4 scope collapse precedent ADR-0054 / ADR-0055 / ADR-0056) or
in a follow-on PR is an operator decision at promotion time;
**this study commits neither path** — it commits the contract
change only.

### What this amendment commits (when promoted)

1. **Group inventory collapses to a single owner.** The three
   placeholder groups (`@PLACEHOLDER-org/platform-team`,
   `@PLACEHOLDER-org/sre`, `@PLACEHOLDER-org/rules-authors`)
   become `@FabioCaffarello`. ADR-0015 §2's inventory table
   is amended; the path-rule table in ADR-0015 §3 is amended
   to name `@FabioCaffarello` wherever it named the placeholder
   groups.
2. **Substitution protocol amended.** ADR-0015 §4's *"in the
   operational session that creates the production GitHub
   org"* trigger is replaced with *"as the implementation
   slice of the ADR-0015 single-user amendment (this ADR)"*.
3. **Asymmetric-review enforcement clause amended.** ADR-0015
   §Consequence 1's *"requires the same review group on both
   files"* and §Consequence 5's branch-protection-anchored
   gating clauses become **descriptive of the single-author
   reality**: the SAME human reviews both halves of every
   merge by construction; no asymmetric path exists because
   no second author exists. The enforcement-via-CODEOWNERS-
   group framing is replaced with enforcement-via-
   author-equals-reviewer-discipline.
4. **The `[H]` reviewer gate in CONTRIBUTING.md Flow 5
   remains in effect**, but is restated explicitly as
   operator-as-person rather than CODEOWNERS-routed. The
   discipline is unchanged; the substrate is.
5. **The `_owners.yaml` lint-time cross-check from ADR-0006
   §9** is amended to allow `@FabioCaffarello` (a personal
   user, not a `@org/team` reference) as a valid `owner`
   value. The runtime semantic stays unchanged (the alerting
   consumer routes per `channels` keyed by entity; the
   `owner` string is human-identification metadata, not the
   routing key).

### Second-order consequences this amendment surfaces explicitly (not buried)

These are the load-bearing facts the amendment is named after.
The amendment is **only honest if these are surfaced, not
hidden in the prose**.

- **(a) Mechanical review enforcement via CODEOWNERS groups
  CEASES TO EXIST.** ADR-0015's asymmetric-review model was a
  mechanism backed by GitHub team groups. Under the single-user
  amendment, no team groups exist; the file routes reviews to
  one user who is also every author; GitHub's *"Require review
  from CODEOWNERS"* branch-protection rule (which depends on a
  reviewer-not-author resolution) is structurally unsatisfiable.
  The amendment **does not pretend** the enforcement still
  operates; it concedes the enforcement collapses and that
  CODEOWNERS becomes a documentation surface plus a PR-
  auto-assignment convenience.

- **(b) The `[H]` reviewer gate (Flow 5) and D0 operator-
  ratification discipline now live ENTIRELY in the operator-as-
  person, not in repo structure.** Prior precedents in this
  session (B3-4, B3-5 D0 ratifications; ADR-0054/0055/0056 R4
  scope collapses; the Wave-S full-gate declaration) all
  rested on the discipline being **operator-side** per
  CONTRIBUTING.md Flow 5 §"Operator-side responsibilities"
  rather than mechanism-enforced. Under the single-user
  amendment, this is no longer one discipline among several —
  it is the **only** discipline. The repository structure
  enforces no review separation; the operator IS the structure.

- **(c) This is a DELIBERATE, NAMED trade-off — not an
  oversight.** The amendment surfaces the trade-off in its
  Notes block per `adr-writing` skill A7 (new-contribution-
  requiring-review marker) and via this study's explicit
  enumeration. Future contributors landing on this repository
  under a different account (if the operator's situation ever
  changes — see Open Question OQ-2) will see in the amendment
  ADR's Notes that the single-author trade-off was committed
  with eyes open, not by drift.

- **(d) P3 ("Ownership must be explicit everywhere") is
  satisfied in letter — every path has an owner — but the
  ASYMMETRIC-REVIEW-AS-MECHANISM layer ADR-0015 built on top
  of P3 is now a DISCIPLINE, not a mechanism.** P3 itself is
  unchanged. The amendment does not weaken P3; it weakens the
  enforcement layer that mechanized one specific way of
  implementing P3.

- **(e) `_owners.yaml` owner values stay SEMANTIC for the
  alerting consumer.** ADR-0006 §"no alert without owner" is
  about the alerting consumer routing alerts based on entity
  `channels`; the `owner` string is human-identification
  metadata that operators reading dashboards or pager pages
  see. The amendment changes the lint-time CODEOWNERS
  cross-check (a defense-in-depth surface) but does NOT
  conflate it with the runtime semantic. `_owners.yaml` keeps
  identifying who is responsible (a person, an email, a
  channel name) — the engine consumes this for ADR-0006
  routing as before. The two surfaces are separable, and the
  amendment treats them separately.

---

## Consequences

1. **ADR-0015 is amended via a standalone promotion ADR (per
   `adr-writing` A4 idiom).** The promotion ADR carries
   `Status: accepted (amends ADR-0015)`. ADR-0015's Status
   line is updated to record the amendment relationship per
   A4. Future readers of ADR-0015 see the amendment marker;
   future readers of the new ADR see the amends-ADR-0015
   marker. The supersession chain is bidirectional.

2. **`/.github/CODEOWNERS` is rewritten** so every rule names
   `@FabioCaffarello`. The file's structural shape (the path-
   rule table, the explicit-line duplications for
   `/.github/workflows/`, `/docs/adr/`, etc., the
   general-to-specific ordering) is **preserved** per
   ADR-0015's Notes section first bullet — those choices stay
   useful as future-refinement slots even though the single-
   user owner is uniform. The amendment changes the owner
   identifier, not the rule structure.

3. **`rules/_owners.yaml`'s `owner` values are updated to
   `@FabioCaffarello`.** The two production entities
   (`customer`, `orders_stream`) currently name
   `@PLACEHOLDER-org/rules-authors`; both flip in the same
   change as the CODEOWNERS update.

4. **The lint-time CODEOWNERS cross-check in `dq-lint`
   relaxes.** Today the check (committed by ADR-0006 §9 and
   B2-9 / ADR-0037 — owner ↔ CODEOWNERS-group cross-check)
   requires `owner` strings to match a `@org/team` reference
   in `/.github/CODEOWNERS`. Under the amendment, the check
   allows personal-user references (the single-user owner
   pattern) as valid. The exact code change in
   `tools/lint/` is implementation; the contract change is
   "personal-user references are valid CODEOWNERS targets in
   the linter's eyes" and lands with the amendment.

5. **The GitHub branch-protection rule *"Require review from
   CODEOWNERS"* MUST be off** for the repository under the
   single-user model (or self-approval must be enabled where
   supported). This is operator-side configuration on
   GitHub; the amendment ADR documents it as a deployment
   precondition of the single-user model. The amendment does
   not attempt to enforce branch-protection settings from
   inside the repo.

6. **The `[H]` reviewer gate in `CONTRIBUTING.md` Flow 5 is
   restated as operator-as-person.** Flow 5 §"[H] reviewer"
   today reads as if the reviewer is somebody other than the
   author (mechanism-enforced via CODEOWNERS). The amendment
   triggers a one-line refresh of Flow 5 making explicit that
   under the single-user model the `[H]` step is
   operator-as-person discipline. **The refresh rides Flow 6
   (factual clarification), not amendment to ADR-0051 or
   ADR-0009** — the substance of Flow 5's `[H]` gate is
   unchanged (operator-as-person was already the default for
   every prior `[H]` gate in this session's precedents — B3-4,
   B3-5, Wave-S declaration); the refresh just names the
   substrate explicitly under single-user. Flow 6's scope
   covers *"tight clarifications of existing rules or
   principles whose substance is unchanged"* — that fits
   exactly. Whether the Flow 6 refresh rides in the same PR
   as the amendment-promotion ADR (R4 collapse, precedent
   ADR-0054/0055/0056) or ships as a separate Flow 6 PR is
   surfaced as OQ-5 below.

7. **The `gh pr merge` deny block in `.claude/settings.json`
   keeps its open question** (see §Open Questions OQ-1).

8. **ADR-0001 (asymmetric-review model for schema half),
   ADR-0006 (no alert without owner), ADR-0009 (multi-agent
   contract), ADR-0037 (owner ↔ CODEOWNERS cross-check) are
   PRESERVED.** Their commitments are not amended — only
   ADR-0015's group-inventory and substitution-protocol
   clauses are touched. ADR-0006's runtime "no alert without
   owner" semantic, ADR-0001's byte-equality CI gate,
   ADR-0009's source-of-truth-for-multi-agent-contract
   posture, and ADR-0037's lint-time cross-check (with the
   personal-user relaxation in Consequence #4) all stay
   in force under the single-user reading.

9. **No B-row is opened.** Amendments live under the
   originating ADR's supersession chain per ADR-0049 §(a)
   "Amendment" branch, never under B3 or B2. This amendment
   was initiated by operator session prompt without an
   originating B-row (unlike B2-36 → ADR-0054 where the
   existing B2 row provided the anchor; here, there was no
   such row). The amendment-promotion session adds an "Earlier
   update" entry to the decision log noting the amendment per
   the precedent shape used by the Wave-S full-gate
   declaration (PR #116) for governance-event records without
   a B-row. **AC-10 from `acceptance-criteria.md` is satisfied
   vacuously** — there is no B-row to update to `resolved-study`
   because no B-row exists for this amendment; the literal
   AC-10 pattern is moot for amendments initiated without an
   originating B-row anchor, and the decision-log update
   lands at promotion-session close, not at this study
   session's close.

10. **Future-org migration remains available.** If the
    operator's situation ever changes and the repository
    moves to a GitHub org, a successor amendment to the
    single-user amendment (or to ADR-0015 directly) restores
    the three-group inventory. Nothing in this amendment
    forecloses that path; the structural shape ADR-0015
    committed (path-rule table, explicit-line duplications,
    SRE-depth pattern for prod overlays) is preserved
    intentionally for that reason.

---

## Open Questions

- **OQ-1 (load-bearing — to be contested by `/critique`):
  should the `.claude/settings.json` deny block on
  `Bash(gh pr merge *)` REMAIN under the single-user model?**
  Two readings.
  - **Stays (recommended in this study, not pre-decided).**
    The deny block depends on no GitHub org for its operation
    — it is a per-session, harness-side mechanical guard
    against the agent (this assistant, future agents) issuing
    `gh pr merge` autonomously. The CODEOWNERS-enforcement
    collapse described in this amendment makes the deny
    block **more** load-bearing, not less: it is the **last
    mechanical guard** standing once CODEOWNERS no longer
    enforces author/reviewer separation. The agent never
    merging is the discipline; the deny block is the
    mechanical backstop that makes the discipline hard to
    violate. Removing the deny block under the single-user
    amendment would conflate two independent surfaces: the
    *operator's review discipline* (Flow 5 [H] reviewer) and
    the *agent's autonomy bound* (no `gh pr merge` from the
    agent). The amendment changes the first; the second is
    untouched.
  - **Goes (counter-argument).** Under single-user, the
    operator IS the author IS the reviewer. They could
    legitimately self-merge their own PR via `gh pr merge`
    from a non-agent shell. The deny block makes this
    awkward for the operator's own scripting (any `Makefile`
    target that runs `gh pr merge` in a wrapper would be
    blocked when invoked through the agent's Bash tool). The
    asymmetry "agent never merges" was originally a corollary
    of the asymmetric review model; once asymmetric review
    collapses, the corollary might too.
  - **Author's recommendation (NOT pre-decided; surfaced for
    operator ratification per CONTRIBUTING.md Flow 5
    §"Operator-side responsibilities"): STAYS.** The two
    surfaces are independent. The deny block guards the
    *agent*, not the *review model*. The operator merging
    their own PR happens in the GitHub UI (or in their own
    non-agent shell); neither is affected by a deny block on
    the agent's `Bash(gh pr merge *)` invocation. The deny
    block enforces a stronger version of CONTRIBUTING.md
    Flow 5's *"agent never calls `gh pr merge`"* clause, and
    that clause is independent of CODEOWNERS enforcement.
    The counter-argument's "awkward scripting" concern is
    real but operator-resolvable by invoking the merge
    outside the agent's Bash sandbox (which is the intended
    workflow today regardless).
  - This OQ is the explicit author-equals-reviewer surface
    where `/critique` cannot self-ratify (ADR-0051
    §Consequence 7). The recommendation above is the author's
    reading; the operator settles it at merge act (per
    CONTRIBUTING.md Flow 5 *"otherwise implicitly resolved
    by the merge act"*) or by explicit mid-PR ratification
    (precedent B3-4 / B3-5 / Wave-S declaration).

- **OQ-2: future-org migration trigger** — when (if ever)
  the operator's situation changes such that the repository
  moves to a GitHub org, the single-user amendment is itself
  amended by a successor ADR. **Out-of-scope for current
  cycle resolution** — no demand signal today; preserved as
  a documented path forward in Consequence #10.

- **OQ-3: applicability of the relaxed `dq-lint` cross-check
  to the future-org case** — if the repository ever moves
  back to a `@org/team` model, the lint-time cross-check
  (relaxed in Consequence #4) needs to be re-tightened. The
  re-tightening is part of the successor amendment for the
  future-org case; OQ-3 is a forward-pointer, not a current
  decision. **Out-of-scope for current cycle resolution.**

- **OQ-4: substitution PR timing (R4 collapse vs follow-on
  slice)** — the amendment-promotion ADR's session decides
  whether the file substitution lands in the same PR as the
  ADR (R4 collapse precedent ADR-0054/0055/0056) or in a
  separate follow-on PR (precedent: nothing recent — most
  amendments collapsed). **Out-of-scope for current cycle
  resolution** — operator-decision at promotion time, not
  for this study to commit.

- **OQ-5: CONTRIBUTING.md Flow 5 refresh PR-shape** — the
  Flow 6 clarification committed in Consequence #6 can
  either ride the amendment-promotion PR (R4 collapse,
  precedent ADR-0054/0055/0056) or ship as a standalone
  Flow 6 PR. R4 collapse is consistent with recent
  precedent but the Flow 5 refresh is a CONTRIBUTING.md
  surface edit (operator-facing process documentation),
  which historically rides Flow 6 standalone. **Out-of-scope
  for current cycle resolution** — operator-decision at
  promotion time, paired with OQ-4 (the
  CODEOWNERS/_owners.yaml substitution timing). The
  refresh-text scope is *one clause-block in Flow 5
  §"[H] reviewer"*, not a sentence and not the entire Flow 5
  section.

---

## Promotion target

[`docs/adr/0057-single-user-codeowners-amendment.md`](../../docs/adr/0057-single-user-codeowners-amendment.md)
(provisional; operator reserves at promotion time per
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
Clause 7; subject to the same numbering caveat per
`adr-writing` A8). Status line on the promotion ADR will
read `Status: accepted (amends ADR-0015)` per `adr-writing`
A4 idiom. ADR-0015's Status line is updated in the same PR
to record the amendment relationship.
