<!-- path: docs/adr/0057-single-user-codeowners-amendment.md -->

# ADR-0057 — Single-User CODEOWNERS Amendment

- **Status:** accepted (amends [ADR-0015](./0015-codeowners.md))
- **Date:** 2026-05-30

---

## Context

[ADR-0015](./0015-codeowners.md) committed a CODEOWNERS
review-ownership map built on three **GitHub-org team groups**
(`@PLACEHOLDER-org/platform-team`, `@PLACEHOLDER-org/sre`,
`@PLACEHOLDER-org/rules-authors`). ADR-0015 §4 ("Substitution
protocol") committed that the placeholder literal would be
exchanged for the real org slug *"in the operational session
that creates the production GitHub org per the ADR-0008
host-primitive follow-up"*. ADR-0015 §Consequence 1 committed
asymmetric-review enforcement *"requires the same review group
on both files"* for the schema-source/mirror byte-equality
pair; §Consequence 5 committed SRE-depth gating on
`/deploy/overlays/{qa,prod}/` via GitHub branch protection's
*"review from CODEOWNERS"* rule.

The repository stays under personal user `@FabioCaffarello` on
GitHub; no `@dq-platform` (or equivalent) org will be created.
The substrate ADR-0015 commits against does not exist and will
not. GitHub personal accounts cannot create or own team groups
— there is no `@FabioCaffarello/platform-team` to which the
three placeholder groups can be mechanically substituted.
Branch protection's *"Require review from CODEOWNERS"* rule
requires a review from a CODEOWNERS-listed account that is
not the PR author; in a single-user repo where the listed
owner is the author, the rule blocks every PR.

This amendment commits the **single-user owner model**: every
CODEOWNERS rule names `@FabioCaffarello`; `_owners.yaml` owner
values flip to match; ADR-0015's three-group-inventory and
substitution-to-real-org commitments are amended; the
asymmetric-review-as-mechanism framing in ADR-0015
§Consequences #1 and #5 becomes **moot** under the single-user
model (no second author exists for asymmetric review to
operate over).

The originating study is the merged amendment study at PR #117
(commit `e659ec3` on 2026-05-30). The study surfaced the
classification as Amendment per
[ADR-0049](./0049-b3-evolutionary-launch.md) §(a) (fails
Condition 1 / P-B3.1: this is rewrite of ADR-0015's decided
shape, not extension) and the load-bearing trade-offs in its
five-point second-order-consequences enumeration. Operator
ratification at PR #117 merge settled the classification.
A coupled OQ-1 (the `.claude/settings.json` deny block on
`Bash(gh pr merge *)` under single-user) was also operator-
ratified at PR #117 merge: **deny block STAYS** — harness-
side mechanical guards are independent of the review-model
surface.

The principles bearing on the decision are **P3** (ownership
must be explicit everywhere — satisfied in letter by the
single owner; what amends is the enforcement layer ADR-0015
built on top of P3, not P3 itself), **R3** (do not revisit
settled architecture — ADR-0001's asymmetric-review-model
commitments for the schema half, ADR-0006's "no alert without
owner" runtime semantic, ADR-0009's multi-agent contract,
and ADR-0037's lint-time cross-check are all PRESERVED; only
the ADR-0015 layer that depended on team-group enforcement
amends), and **R8** (this ADR is written for future readers
who land on it cold; the originating study is not back-linked
per `adr-writing` A5).

Per the operator-authorized **R4 scope collapse** at promotion
time (precedent [ADR-0054](./0054-engine-image-registry-amendment.md),
[ADR-0055](./0055-metric-emission-slice-scope.md),
[ADR-0056](./0056-panel-5-lighting-slice.md) §Notes), this ADR
ships in the same PR as its implementation slice (CODEOWNERS
substitution + `_owners.yaml` substitution + `CONTRIBUTING.md`
Flow 5 refresh + `ADR-0015` Status-line amendment marker +
decision-log "Last updated" entry). The structural choices in
Clauses 1–5 below are reviewer-load-bearing because they are
reviewable against the working substitution diff.

---

## Decision

### Clause 1 — Group inventory collapses to single owner

ADR-0015 §2 ("Group inventory — three groups") is amended.
The three placeholder groups
(`@PLACEHOLDER-org/platform-team`, `@PLACEHOLDER-org/sre`,
`@PLACEHOLDER-org/rules-authors`) collapse to a single
owner: **`@FabioCaffarello`** (the personal-user GitHub
account that owns the repository). Every rule in
`/.github/CODEOWNERS` names `@FabioCaffarello` — including
the rules ADR-0015 §3 committed at SRE-depth for
`/deploy/overlays/{qa,prod}/` (which now name
`@FabioCaffarello` once, not twice; there is no second
identity to list as SRE co-owner).

The path-rule table from ADR-0015 §3 is **structurally
preserved**: every original path rule remains, the
general-to-specific ordering remains, the explicit-line
duplications for `/.github/workflows/`, `/docs/adr/`, etc.
remain. Only the owner column flips. The structural
preservation is deliberate — if the operator's situation
ever changes and the repository moves to a GitHub org, the
path-rule table is the substrate a successor amendment
reactivates with new owner identifiers.

### Clause 2 — Substitution protocol amended

ADR-0015 §4 ("Substitution protocol") is amended. The
trigger *"in the operational session that creates the
production GitHub org per the ADR-0008 host-primitive
follow-up"* no longer obtains (no such session is planned).
The substitution lands as the implementation slice in this
ADR's PR per R4 collapse (precedent ADR-0054, ADR-0055,
ADR-0056 §Notes). The substitution is mechanical — every
`@PLACEHOLDER-org/*` token in `/.github/CODEOWNERS` and
`rules/_owners.yaml` becomes `@FabioCaffarello` — and is
reviewable against the diff in the same PR.

### Clause 3 — Asymmetric-review-as-mechanism framing becomes moot

ADR-0015 §Consequence 1 (*"requires the same review group
on both files"* — schema source + mirror byte-equality
asymmetric-review enforcement) and §Consequence 5 (SRE
depth on production overlays via GitHub branch protection's
*"review from CODEOWNERS"* rule) are amended. Under the
single-user model:

- §Consequence 1's *"requires the same review group on
  both files"* clause becomes **moot**. There is only one
  reviewer; the author IS the reviewer for every merge;
  asymmetric-review enforcement has no separation to
  enforce. The byte-equality CI gate ADR-0001 §C9 commits
  remains active; the review-group-on-both-files clause is
  vacuously satisfied (one owner = both halves' owner) but
  the asymmetric-review framing no longer describes the
  enforcement reality.
- §Consequence 5's *"GitHub branch-protection's 'review
  from CODEOWNERS' requirement gates them on both groups"*
  clause becomes **moot**. There is one owner, not two; the
  branch-protection rule's "review from a CODEOWNERS account
  that is not the PR author" requirement is structurally
  unsatisfiable when the only listed owner IS the author.
  The amendment commits the deployment precondition: the
  *"Require review from CODEOWNERS"* branch-protection rule
  MUST be off under the single-user model (or self-approval
  must be enabled where supported) — see §Notes for the
  precondition's audit surface.

The amendment **does not pretend** the enforcement still
operates. The asymmetric-review-as-mechanism layer ADR-0015
built on top of P3 (Ownership) is replaced under this
amendment by **author-equals-reviewer discipline** —
operator-as-person, not repo-structure-as-mechanism. This is
the deliberate, named trade-off the amendment commits (see
§Notes carry-forward).

### Clause 4 — `_owners.yaml` semantics preserved; ADR-0037 cross-check NOT amended

`rules/_owners.yaml`'s `owner:` field semantic is **preserved
verbatim** for the alerting consumer:
[ADR-0006](./0006-alert-routing-contract.md) §"no alert
without owner" routes alerts based on the per-entity
`channels` block (Slack/email/PagerDuty references); the
`owner` string identifies the human responsible. Under the
single-user amendment, `@FabioCaffarello` IS that human; the
runtime semantic is unchanged.

[ADR-0037](./0037-owner-codeowners-cross-check.md) — the
lint-time cross-check that requires every `owner` value in
`_owners.yaml` to appear in `/.github/CODEOWNERS` — is
**preserved without amendment**. The originating amendment
study's Consequence #4 framed this as "the lint-time
cross-check relaxes" — that framing was over-broad. ADR-0037
§"Parser scope" item 4 committed the parser's reviewer-token
regex as *"the GitHub CODEOWNERS reviewer-identifier shape
(`@<user>` or `@<org>/<team>`)"*; `tools/lint/codeowners.go`
implements the regex as `^@[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)?$`
which accepts the `@<user>` shape today. After substitution,
`@FabioCaffarello` appears in both `/.github/CODEOWNERS` and
`_owners.yaml`; the cross-check validates set-membership and
passes without any code change. This amendment is
single-amender (`amends ADR-0015` only); ADR-0037 stays
accepted as written.

### Clause 5 — CONTRIBUTING.md Flow 5 [H] reviewer refresh

`CONTRIBUTING.md` Flow 5's PR-close paragraph
(*"Once green and the [H] reviewer approves, merge via the
GitHub UI. The agent never calls gh pr merge."*) gains a
clause-block clarifying that under the single-user model
committed by this amendment, **the [H] reviewer IS the
operator-as-person** — author-equals-reviewer per ADR-0051
§Consequence 7. The merge act in the GitHub UI is the
ratification per CONTRIBUTING.md §"Operator-side
responsibilities". This is a Flow-6-shape factual
clarification (*"tight clarifications of existing rules or
principles whose substance is unchanged"* per CONTRIBUTING.md
Flow 6 §"Scope — what qualifies") — the substance of Flow 5's
[H] gate is unchanged (operator-as-person was already the
default for every prior [H] gate in the post-Wave-3
precedents — B3-4 D0, B3-5 D0, Wave-S declaration). The
refresh just names the substrate explicitly under
single-user.

---

## Consequences

1. **ADR-0015 is amended via this standalone ADR per
   `adr-writing` A4 idiom.** ADR-0015's Status line is
   updated in the same PR to record the amendment
   relationship: `accepted; **amended by ADR-0057** (group
   inventory collapses to single owner @FabioCaffarello;
   substitution protocol re-targeted; asymmetric-review-as-
   mechanism framing becomes moot under single-user)`.
   Same shape as ADR-0010's Status line records its
   amendments by ADR-0017 + ADR-0028.

2. **`/.github/CODEOWNERS` is rewritten** so every rule
   names `@FabioCaffarello`. Thirty-plus rule lines flip
   uniformly; the path-rule table structure (general-to-
   specific ordering; explicit-line duplications for
   `/.github/workflows/`, `/docs/adr/`, etc.) is preserved.
   The SRE-depth lines for `/deploy/overlays/{qa,prod}/`
   collapse from two-owner to one-owner.

3. **`rules/_owners.yaml`'s owner values flip to
   `@FabioCaffarello`** for both production entities
   (`customer`, `orders_stream`). The `mode`, `description`,
   and `channels` blocks are preserved unchanged — only the
   `owner` string flips.

4. **The dq-lint cross-check passes against the substituted
   files without code change.** ADR-0037 §"Parser scope"
   item 4 already commits the `@<user>` reviewer-token
   shape; the `tools/lint/codeowners.go` regex accepts it;
   the `CheckOwnersGroupMembership` cross-check validates
   set-membership tolerantly of the reviewer-token shape.
   `make lint-rules` + `make test-tools` continue to pass.
   The originating amendment study's Consequence #4 framing
   ("dq-lint relaxes") is corrected here — no relaxation
   was needed.

5. **CONTRIBUTING.md Flow 5 is refreshed** (single clause-
   block in the PR-close paragraph) clarifying the
   single-user reading of the [H] reviewer gate. This is
   Flow 6-shape per CONTRIBUTING.md Flow 6 §"Scope"; the
   substance of Flow 5's [H] gate is unchanged.

6. **The `.claude/settings.json` deny block on
   `Bash(gh pr merge *)` is preserved without change**, per
   the operator-ratified OQ-1 disposition from the
   originating amendment study (ratified at PR #117 merge).
   Harness-side mechanical guards are independent of the
   review-model surface; the deny block guards the *agent's*
   autonomy bound, not the *review model*. Under the
   amendment that collapses asymmetric review, the deny
   block becomes the **last mechanical guard** standing —
   it is more load-bearing, not less. See §Notes
   carry-forward.

7. **GitHub branch-protection rule *"Require review from
   CODEOWNERS"* MUST be off** for the repository under the
   single-user model (or self-approval must be enabled
   where supported). This is operator-side configuration on
   GitHub; the amendment documents it as a deployment
   precondition in §Notes but does not attempt to enforce
   it from inside the repo.

8. **ADR-0001, ADR-0006, ADR-0009, ADR-0037, ADR-0049, and
   ADR-0051 are preserved.** Their commitments are not
   amended — only ADR-0015's group-inventory + substitution-
   protocol + asymmetric-review-enforcement clauses are
   touched. ADR-0001's byte-equality CI gate stays active;
   ADR-0006's runtime alerting semantic is unchanged;
   ADR-0009's multi-agent contract source-of-truth posture
   is unchanged; ADR-0037's cross-check is shape-tolerant
   already; ADR-0049 §(a)'s amendment branch is the
   classification this ADR rides under; ADR-0051's
   author-equals-reviewer rationale (§Consequence 7) is the
   load-bearing rationale that makes the operator-as-person
   discipline coherent under single-user.

9. **No B-row is opened, and no decision-log B-row is
   updated.** The originating amendment study was initiated
   by operator session prompt without an originating B-row
   (PR #117 study Consequence #9 explicit on this point);
   the decision-log "Last updated" entry that records this
   amendment uses the precedent shape established by the
   Wave-S full-gate declaration (PR #116 — governance-event
   record without a B-row anchor). AC-10 from
   `acceptance-criteria.md` is satisfied vacuously per the
   originating study's Consequence #9.

10. **Future-org migration remains available.** If the
    operator's situation ever changes and the repository
    moves to a GitHub org, a successor amendment restores
    the three-group inventory. The path-rule table from
    ADR-0015 §3 is preserved structurally exactly so that
    migration is a uniform owner-column flip + branch-
    protection re-enablement, not a re-derivation of the
    CODEOWNERS file from scratch.

11. **Per-entity CODEOWNERS refinement remains additive.**
    ADR-0015 §11 reserved per-entity CODEOWNERS lines (e.g.,
    `/rules/customer.yaml @customer-team`) as an additive
    change that does not reopen ADR-0015. Under this
    amendment, that future additive is gated on the same
    successor amendment that restores team groups — until
    then, all per-entity routing lives in `_owners.yaml`
    per ADR-0015 §Consequence 3 (unchanged here).

12. **ADR-0042's Status-line amendment marker remains a
    deferred housekeeping item.** When ADR-0054 amended
    ADR-0042 on 2026-05-30, ADR-0042's Status line was not
    updated to record the amendment-by relationship per A4
    (current Status: just `accepted`). The more rigorous
    precedent is ADR-0010 (records every amender in its
    Status line). This ADR follows the ADR-0010 pattern for
    ADR-0015 (Consequence #1 above); the symmetric fix to
    ADR-0042 stays out of scope per R4 (one topic per
    session) — registered here as a forward-pointer for a
    future Flow 6 housekeeping PR.

---

## Notes

**Operator-authorized R4 scope collapse.** The originating
amendment study (PR #117) committed the contract change; this
ADR is the standalone-amendment promotion + implementation
slice in a single PR per operator authorization. Same
precedent as ADR-0054 (B2-36 amendment + wiring), ADR-0055
(B3-4 promotion + emission slice), ADR-0056 (B3-5 promotion +
panel-5 lighting slice). Per memory `r4-scope-collapse-precedent.md`
the structural condition is satisfied: ADR-0057 commits
load-bearing structural choices (single-owner model,
substitution protocol re-targeting, branch-protection
precondition) and the slice (CODEOWNERS + `_owners.yaml`
substitution + Flow 5 refresh) is the direct mechanical
realization.

**Second-order trade-offs (a)…(e) carry-forward per R5 + A7.**
Per the originating study's Recommendation §"Second-order
consequences":

- **(a)** Mechanical review enforcement via CODEOWNERS groups
  CEASES TO EXIST under the single-user amendment. No team
  groups; CODEOWNERS routes reviews to one user who is also
  every author; GitHub's *"Require review from CODEOWNERS"*
  branch-protection rule is structurally unsatisfiable. The
  amendment concedes the enforcement collapses and that
  CODEOWNERS becomes a documentation surface plus a PR-auto-
  assignment convenience.
- **(b)** The [H] reviewer gate (CONTRIBUTING.md Flow 5) and
  D0 operator-ratification discipline now live **entirely in
  the operator-as-person**, not in repo structure. The
  repository structure enforces no review separation; the
  operator IS the structure.
- **(c)** This is a **deliberate, named trade-off** — not an
  oversight. The amendment records the trade-off in this
  Notes block per A7 (new-contribution-requiring-review
  marker) so future readers landing on this ADR cold see the
  single-author trade-off was committed with eyes open.
- **(d)** P3 (Ownership must be explicit everywhere) is
  satisfied in letter — every path has an owner — but the
  asymmetric-review-as-mechanism layer ADR-0015 built on top
  of P3 becomes a discipline, not a mechanism. P3 itself is
  unchanged.
- **(e)** `_owners.yaml` owner values stay semantic for the
  alerting consumer (ADR-0006 runtime routing). The lint-time
  CODEOWNERS cross-check (ADR-0037) is shape-tolerant already
  and needs no relaxation; the two surfaces (runtime
  alerting + lint-time defense-in-depth) are separable and
  the amendment treats them separately.

**Deny-block independence carry-forward per OQ-1 ratification.**
Per memory `harness-deny-blocks-independent-of-review-model.md`
+ the operator-ratified OQ-1 disposition at PR #117 merge:
the `.claude/settings.json` deny block on
`Bash(gh pr merge *)` is independent of the CODEOWNERS
enforcement surface this amendment collapses. The deny block
guards the *agent's* autonomy bound; CODEOWNERS guards the
*review model*. The two surfaces are independent — and when
the review model weakens (as it does under single-user), the
deny block becomes **more** load-bearing, not less. The
amendment does NOT touch `.claude/settings.json`; the deny
block stays.

**Branch-protection precondition (Consequence #7) audit
surface.** The deployment precondition that *"Require review
from CODEOWNERS"* MUST be off (or self-approval enabled) is
operator-side GitHub configuration. The amendment does not
attempt to enforce branch-protection settings from inside
the repo. Operators / future operators landing on this ADR
must verify the precondition is satisfied; the symptom of
violation is "every PR blocked at merge time with a
CODEOWNERS-review-required error". This Notes paragraph is
the audit surface per A7 — future readers landing on this
ADR see the precondition explicitly.

**Critique rounds.** This ADR's Decision survived one
`/critique` round in the promotion session (round-1
disposition recorded in the PR body's Critique result
table). The originating amendment study survived one round
(1 = 0 blocking / 3 important / 5 minor with the three
importants applied at study time). The implementation code
in this PR (the substitution + Flow 5 refresh) is
self-verified against AC-W3-3 + AC-W3-7 per ADR-0052 §6.4
row 6 close-gates and ADR-0048 §"Skip" path for code-only
`/critique` rounds.
