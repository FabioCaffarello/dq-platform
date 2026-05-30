<!-- path: docs/adr/0052-session-reading-router.md -->

# ADR-0052 — Session Reading Router for `CLAUDE.md` §6

- **Status:** accepted
- **Date:** 2026-05-30

---

## Context

[`CLAUDE.md`](../../CLAUDE.md) §6 currently prescribes a
**uniform required-reading set** at session start: every
session is asked to read every playbook under
`.claude/playbooks/`, alongside the foundation documents.
The corpus today is six playbooks totaling roughly 700
lines: `wave-1-session-loop.md` (~140 lines; Wave 1 closed
2026-05-21), `wave-3-session-loop.md` (~170 lines; Wave 3
closed 2026-05-23), `post-wave3-session-loop.md` (~225
lines; the current operational loop committed by
[ADR-0051](./0051-claude-tooling-postwave3.md) Clause 4),
`acceptance-criteria.md` (~35 lines; B0-shaped, inherited
by B2 / B3 sessions), `wave-3-acceptance-criteria.md` (~70
lines; scaffold-shaped), and `feedback-protocol.md` (~65
lines; shared across all study/review work).

The uniform prescription was correct when written. During
Wave 3, the active loop was the Wave-3 loop, and the
Wave-1 loop was the shape it mirrored. A contributor
opening a session could realistically need every playbook
in scope.

The operating shape has changed. Three concrete sessions
that have run since Wave 3 closed illustrate the cost: the
B3-1 session that produced
[ADR-0051](./0051-claude-tooling-postwave3.md); a
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6 process
edit to [`CLAUDE.md`](../../CLAUDE.md) for post-Wave-3
state; and a Flow 6 in-place ADR-correction PR. In each
case the load-bearing reading was a small, specific
subset; the remaining playbooks were inert and reading them
was not load-bearing for that session's output.

Two governance gaps motivate fixing this at the §6 layer
rather than letting each session improvise:

1. **No documented mapping from session type to minimal
   reading set.** A contributor opening a
   [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6
   process-edit session today has no way to know — without
   re-deriving from the playbook texts — which playbooks
   are actually load-bearing for that session. The uniform
   "read all" prescription over-reads; an under-reading
   improvisation under-reads with no audit surface.

2. **No taxonomy of post-Wave-3 session types.** Six
   distinct session types operate in the post-Wave-3 lane
   — **B2 follow-up**, **B3 entry** per
   [ADR-0049](./0049-b3-evolutionary-launch.md),
   **ADR amendment** (in the
   [ADR-0050](./0050-v1-retirement-engine-release.md)
   §Consequence 4 Amendment-log shape), **ADR promotion**,
   **[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6
   process edit**, and **implementation slice landing
   under a closed B-row**. Each has its own grounding
   needs. They are currently scattered across
   `post-wave3-session-loop.md`,
   [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5 and
   Flow 6, [ADR-0050](./0050-v1-retirement-engine-release.md)
   §Consequence 4, and `wave-3-acceptance-criteria.md`. No
   single surface enumerates the six types or maps them
   to required reading.

This ADR closes both gaps by committing a **session
reading router** in [`CLAUDE.md`](../../CLAUDE.md) §6.

**Eligibility carry-forward (`B3-2`; D0
operator-ratified, not critique-derived).**
[ADR-0049](./0049-b3-evolutionary-launch.md) §(a) commits
the four-condition eligibility filter for B3-N entries.
For `B3-2` (the row whose promotion this ADR is), two of
the four conditions cleared cleanly and two were
borderline:

- **Condition 2 (in-scope family — Tooling extensions)**
  passes by **direct precedent reuse** of
  [ADR-0051](./0051-claude-tooling-postwave3.md) Clause 1's
  expansive reading admitting the `.claude/` agent harness
  under [ADR-0049](./0049-b3-evolutionary-launch.md)
  §Per-family scope's "adjacent tooling" clause. This ADR
  does **not** extend that reading further; it reuses the
  precedent for an artifact that lives inside the same
  admitted surface ([`CLAUDE.md`](../../CLAUDE.md) §6
  itself, which `.claude/`-harness playbooks already
  reference).
- **Condition 3 (envelope conformance to
  [ADR-0020](./0020-wave-s-launch.md) /
  [ADR-0021](./0021-mode-as-primitive.md) /
  [ADR-0022](./0022-kind-catalog.md) /
  [ADR-0023](./0023-sources-schema.md))** passes trivially:
  no substrate, mode, kind catalog, or sources schema row
  is touched.
- **Condition 1 (P-B3.1, expands not rewrites)** is
  **borderline**. The router preserves R1–R8 (`CLAUDE.md`
  §3) and P1–P6 (`CLAUDE.md` §4) reading verbatim; every
  playbook in the corpus stays on disk unchanged. What
  narrows is the per-session-type playbook reading
  prescription in [`CLAUDE.md`](../../CLAUDE.md) §6. Whether
  replacing "read all playbooks" with "read the subset
  declared by the router" counts as *rewriting* the §6
  prescription or as *extending* it with a routing layer
  that preserves the prescription's intent (every relevant
  playbook still loads when its session type runs) is
  interpretive.
- **Condition 4 (additive-maintenance threshold)** is
  **borderline**. The router commits a new
  session-type taxonomy that reshapes contributor
  expectations (future playbook additions declare their
  router applicability). An alternate reading frames the
  proposal as a tight clarification under
  [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6.

The borderline readings on Conditions 1 and 4 are
**operator-ratified, not critique-derived**, and this
ADR propagates that distinction per the
new-contribution-requiring-review discipline in
[`CLAUDE.md`](../../CLAUDE.md) §3 R5. The rationale is
the **author-equals-reviewer circularity** committed by
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5
§"Operator-side responsibilities":
[ADR-0051](./0051-claude-tooling-postwave3.md)
§Consequence 7 names it explicitly. When the agent
running `/critique` and the agent producing the artifact
share a session identity, a `/critique`-emitted
eligibility ruling is structurally circular — the author
cannot ratify its own eligibility reading by emitting
critique findings. Ratification therefore lives with the
operator, surfaces explicitly to the operator (in the
study and in the promotion-PR body), and carries forward
to the promoted ADR as a marker — not as a settled
position. **The borderline readings on Conditions 1 and 4
this ADR commits are new contribution requiring review.**
Future B3-N sessions that need to revisit Condition 1 or
Condition 4 cite ADR-0052 as the precedent for the
ratification mechanism; they do not absorb the readings
as settled.

The principles bearing on this decision are **P5**
(evolution must be contract-driven — the router is a
contract with the contributor population, and its shape
evolves under [`CLAUDE.md`](../../CLAUDE.md) §6's
published surface), **P6** (borrow patterns, not baggage —
the router is defended on its fit to this project's
session shape, not on resemblance to external agent-harness
convention), and **P3** (ownership is explicit — every
session type is named, every reading set is enumerated,
no implicit defaults). **R4** in
[`CLAUDE.md`](../../CLAUDE.md) §3 (one topic per session) is
load-bearing for the artifact split: this ADR commits the
router as a contract; the [`CLAUDE.md`](../../CLAUDE.md) §6
text edit that lands the router is a separate
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6 PR in a
different session. **R8** is load-bearing for this ADR's
prose: a reader of this ADR alone, without opening any
file under `studies/`, must understand what the router
decides and why.

---

## Decision

### Clause 1 — Always-on floor

Every session, regardless of type, reads the following
four surfaces in full at session open:

- [`CLAUDE.md`](../../CLAUDE.md) §1–8 in full. §1's
  "read this entire file before producing any output"
  mandate covers R1–R8 (§3), P1–P6 (§4), the phase/lane
  taxonomy (§2), the session-reading router (§6,
  including the router itself as an intentional
  self-reference — every session reads §6 to find its
  own row), and the slash-command inventory (§7).
- [`AGENTS.md`](../../AGENTS.md) (cross-agent convention
  file; rebinds the hard rules to non-Claude agents).
- [`CONTRIBUTING.md`](../../CONTRIBUTING.md) (PR-flow
  contract, authoritative per
  [ADR-0051](./0051-claude-tooling-postwave3.md) Clause 2;
  Flow 5 is the post-Wave-3 PR-flow; Flow 6 is the
  direct-edit lane).
- [`studies/foundation/06-decision-log.md`](../../studies/foundation/06-decision-log.md)
  (live state surface for every B-row, ADR status, and
  Wave-S gate status). Per
  [`CLAUDE.md`](../../CLAUDE.md) R8 the foundation
  directory is exempt from the "studies are not the
  product" restriction: foundation documents are the
  product (cross-agent live-state surface), distinct
  from the reasoning artifacts in `studies/decisions/`
  and `studies/critiques/`.

R1–R8 and P1–P6 reading is **mandatory for every session
without exception**. The router governs only the playbook
layer; it never narrows rule or principle reading.

### Clause 2 — Six session-type rows

The router maps six session types to a minimal
required-reading playbook subset (beyond the
always-on floor of Clause 1). The six types are the
exhaustive set under the current operating shape.

| Session type | Trigger / when to use | Minimal required playbook reading (beyond the floor) |
|---|---|---|
| **B2 follow-up** | A B-row marked `B2` in the decision log; the session resolves an implementation-phase decision against an in-flight wave. | `post-wave3-session-loop.md` (step 2's wave-gate confirmation is load-bearing); `acceptance-criteria.md` (AC-1…AC-10 — B2 studies inherit B0 study shape per [ADR-0051](./0051-claude-tooling-postwave3.md) Notes OQ-1); `feedback-protocol.md`. |
| **B3 entry** | A B-row marked `B3-N` in the decision log; the [ADR-0049](./0049-b3-evolutionary-launch.md) §(a) eligibility filter must clear before drafting. | `post-wave3-session-loop.md` (step 2's eligibility-check sub-step is load-bearing); `acceptance-criteria.md`; `feedback-protocol.md`; [ADR-0049](./0049-b3-evolutionary-launch.md) §(a) and §(b). |
| **ADR amendment** | An in-place edit to an existing ADR — either a structured-data row amendment or an Amendment-log subsection per [ADR-0050](./0050-v1-retirement-engine-release.md) §Consequence 4. No decision rewrite. | `post-wave3-session-loop.md` step 10 (the PR-flow close); `feedback-protocol.md`; the originating ADR; [ADR-0050](./0050-v1-retirement-engine-release.md) §Consequence 4 (Amendment-log convention). `acceptance-criteria.md` optional — only if the amendment produces a study. |
| **ADR promotion** | A `resolved-study` is being promoted via [`/promote-to-adr`](../../.claude/commands/promote-to-adr.md). | `post-wave3-session-loop.md` step 10 (the PR-flow close); `acceptance-criteria.md` (the source study must have cleared AC-1…AC-10 before promotion); `feedback-protocol.md` (the promotion may surface critique-style feedback on the proposed ADR text); the [`/promote-to-adr`](../../.claude/commands/promote-to-adr.md) command spec. |
| **Flow 6 process edit** | An operator-authorized direct edit to `CLAUDE.md` / `AGENTS.md` / `.codex/AGENTS.md` per [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6. | [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6 (scope-and-gate; the load-bearing contract — Flow 6 explicitly inherits Flow 5 PR-flow). `post-wave3-session-loop.md` step 10 (the load-bearing playbook content for PR-flow close; Flow 6 inherits the rest of Flow 5 through [`CONTRIBUTING.md`](../../CONTRIBUTING.md), not through the playbook). `feedback-protocol.md` optional; **load-bearing if `/critique` is run** (the [`/critique`](../../.claude/commands/critique.md) command grounds on it). |
| **Implementation slice landing under a closed B-row** | A code or scaffold slice that lands the artifacts committed by a closed B-row's ADR (e.g., an [ADR-0051](./0051-claude-tooling-postwave3.md) follow-on slice shipping the six artifacts the ADR commits, or a deferred Wave-3 capability-matrix row landing under a Wave-3 closure). | `post-wave3-session-loop.md` (the close-discipline); `wave-3-acceptance-criteria.md` (AC-W3-3 load-bearing for citation discipline; AC-W3-7 load-bearing for local build / lint / test gates — both are scaffold-shaped semantics that apply to any post-Wave-3 implementation slice); `feedback-protocol.md`. `acceptance-criteria.md` optional — only if the slice produces a follow-up study. |

The trigger column is the concrete classification test
for each row. The minimal-reading column is the load-bearing
playbook subset for sessions of that type, beyond the
always-on floor of Clause 1.

Three boundary cases that *might* look like new session
types collapse into the six rows above:

- **B-row triage** (deciding whether a newly surfaced item
  is `B2` / `B3` / `amendment` / `rejected` per
  [ADR-0049](./0049-b3-evolutionary-launch.md) §(a)) is a
  one-pass classification that runs *before* the session
  opens; it does not produce a study, an ADR, or a
  scaffold. The operator reads
  [ADR-0049](./0049-b3-evolutionary-launch.md) §(a) and
  the decision log — both already in the always-on floor.
  No router row needed.
- **Study revival** (re-opening a previously deferred
  study) follows the originating row's type — `B2` if
  originally B2, `B3` if originally B3. The router routes
  by *current* session type, not by originating wave.
- **ADR supersession** (the
  [ADR-0017](./0017-substrate-posture-amendment.md)
  pattern) is a special case of ADR amendment when the
  amendment reshapes architectural prose rather than
  touching only structured data (the
  [ADR-0050](./0050-v1-retirement-engine-release.md)
  §Consequence 4 boundary). The "ADR amendment" row
  already covers it; the in-place vs. supersession choice
  is made inside the session.

Future session types beyond the six (a rule-promotion
lane, a release-cut lane, an automation lane) require a
router-row addition through a follow-up B3-N entry; the
router is not silently extended.

### Clause 3 — Default-up safety rule

If a contributor or agent is unsure between two session
types — e.g., between a [`CONTRIBUTING.md`](../../CONTRIBUTING.md)
Flow 6 process edit and a B3 entry on a borderline
materiality call — **default to the next-larger reading
set**. The larger set is a superset; reading it does not
violate any rule and costs at most one extra playbook.
The router narrows confidently-classified sessions;
uncertain classification reverts to the
universal-prescription behavior for that session only.

The default-up rule is the safety belt against the
under-reading drift risk that any narrowed prescription
carries. A misclassified session that defaults up loses
the cost-savings benefit for one session but does not
violate any rule or principle; a misclassified session
that defaults down may under-read a load-bearing
playbook.

### Clause 4 — Output-artifact row-boundary tie-breaker

Where two rows in Clause 2 could plausibly apply
(canonical case: a B2 follow-up that ships an
implementation slice — the originating B-row is B2, and
the session lands an implementation under a closed
B-row), the tie-breaker is the session's **output
artifact**:

- The **B2 follow-up**, **B3 entry**, **ADR amendment**,
  and **ADR promotion** rows apply when the session's
  output is a study or an ADR document. The close-gate
  is AC-1…AC-10 from `acceptance-criteria.md`.
- The **Implementation slice** row applies when the
  session's output is code or scaffold. The close-gates
  are AC-W3-3 and AC-W3-7 from
  `wave-3-acceptance-criteria.md`.
- The **Flow 6 process edit** row applies when the
  session's output is an edit to a process document
  (`CLAUDE.md` / `AGENTS.md` / `.codex/AGENTS.md`). The
  close-gate is the Flow 6 scope clause in
  [`CONTRIBUTING.md`](../../CONTRIBUTING.md).

A B2 follow-up that both produces a study *and* lands an
implementation slice runs as **two separate sessions**
per [`CLAUDE.md`](../../CLAUDE.md) R4 (one topic per
session). The study session reads the B2 row; the
implementation-slice session reads the Implementation
slice row. The tie-breaker resolves to one of the six
Clause 2 rows in every case — it is a disambiguation
between rows, not an independent behavior axis.

### Clause 5 — Historical-skim set

Two playbooks are explicitly historical as of this ADR's
date. They are preserved verbatim on disk (no edits, no
deletions, no relocation), and they are not load-bearing
for any of the six session types in Clause 2:

- `wave-1-session-loop.md` — Wave 1 closed 2026-05-21.
  Shape reference for the `post-wave3-session-loop.md`
  ten-step structure.
- `wave-3-session-loop.md` — Wave 3 closed 2026-05-23.
  Shape reference for the PR-flow discipline now
  authoritative in [`CONTRIBUTING.md`](../../CONTRIBUTING.md)
  Flow 5.

Skim only when explicitly reading prior sessions for
shape-reference context. The third Wave-3-era playbook,
`wave-3-acceptance-criteria.md`, is **not** in the
historical-skim set: its AC-W3-3 and AC-W3-7 rows are
load-bearing for the Implementation slice row of
Clause 2.

### Clause 6 — Preservation of the rule and principle floor

R1–R8 ([`CLAUDE.md`](../../CLAUDE.md) §3) and P1–P6
([`CLAUDE.md`](../../CLAUDE.md) §4) apply to every
session unconditionally. The router never narrows rule or
principle reading; [`CLAUDE.md`](../../CLAUDE.md) §1's
"read the entire file before producing any output"
mandate covers the floor before §6 even applies.
Specifically:

- R6 (path header on every produced markdown file)
  applies to every produced artifact regardless of
  session type.
- R8 (studies are not the published product) applies
  whenever a promoted ADR is written.
- The [`session-governance`](../../.claude/skills/session-governance/SKILL.md)
  skill (committed by
  [ADR-0051](./0051-claude-tooling-postwave3.md)
  Clause 3) loads on description-match basis at the PR /
  branch / commit moments, independent of session type.
  It does not consult the router; it triggers on
  contributor phrases.
- No playbook is edited, deleted, or relocated by this
  ADR. The router is purely additive at the playbook
  layer.

---

## Consequences

1. **Each session reads a smaller, named playbook
   subset.** By line count, the maximum saving (a
   [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6
   process edit) is roughly 600 lines of playbook text
   avoided (~85% of the current corpus by line count). The
   minimum saving (an implementation-slice or B3 entry) is
   roughly 280 lines of historical-skim content (Wave-1
   loop, Wave-3 loop). The numbers are **by line count,
   not benchmarked against attention cost** — structured
   prose and dense code do not carry equivalent reading
   load. The first 2–3 post-router sessions provide an
   empirical measurement window; the measurement-decision
   PR opens by the third post-router session at the
   latest, regardless of whether 2–3 sessions have
   surfaced enough signal.

2. **Session-type taxonomy becomes a load-bearing concept
   for the harness.** Future playbook additions declare
   their session-type applicability against the Clause 2
   table; future playbooks that apply universally become
   part of the Clause 1 always-on floor explicitly. The
   router itself becomes a contract surface that future
   B3-N entries amend.

3. **The R1–R8 and P1–P6 reading floor is preserved
   without exception.** [`CLAUDE.md`](../../CLAUDE.md) §1's
   "read the entire file before producing any output"
   mandate covers the floor before §6 even applies. A
   session that follows the router reads strictly more
   than R/P; it never reads less.

4. **The Wave-1 and Wave-3 session-loop playbooks become
   explicitly historical.** Content is preserved verbatim;
   their relegation to the Clause 5 historical-skim set
   matches their closed-wave status without erasing the
   audit trail. The `post-wave3-session-loop.md` becomes
   the only currently-active loop, and the published
   [`CLAUDE.md`](../../CLAUDE.md) §6 surface will say so
   directly when the follow-on
   [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6 PR
   lands the router text.

5. **A new session type requires a router-row addition
   through a follow-up B3-N entry.** The router does not
   silently extend. A future automation lane or
   release-cut lane will be born with its own row, not
   absorbed into "B3 entry" or "Flow 6 process edit" as a
   convenience.

6. **The
   [`session-governance`](../../.claude/skills/session-governance/SKILL.md)
   skill is unaffected by the router.** Its
   description-match triggering on PR / branch / commit
   phrases (per
   [ADR-0051](./0051-claude-tooling-postwave3.md) Clause 3)
   is orthogonal to the router. The skill loads when the
   contributor reaches the PR moment, regardless of
   session type. No edit to the skill is committed by this
   ADR.

7. **Under-reading drift is mitigated by the default-up
   safety rule (Clause 3) and the explicit trigger column
   (Clause 2).** A session that mis-classifies its own
   type at session open will either default up (the safe
   case) or, if it commits to a narrower type that turns
   out wrong, abort per the abort criteria in
   `post-wave3-session-loop.md` §"When to abort the
   session" and re-open with the correct classification.

8. **The router lands in
   [`CLAUDE.md`](../../CLAUDE.md) §6 via a follow-on
   [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6 PR
   in a separate session.** This ADR commits the contract
   shape; the constitutional-document edit lands in its
   own session under R4. The Flow 6 PR is the second
   load-bearing example of the direct-edit lane after
   the 2026-05-30
   [`CLAUDE.md`](../../CLAUDE.md) refresh.

9. **[`/sync-agents`](../../.claude/commands/sync-agents.md)
   coverage of [`CLAUDE.md`](../../CLAUDE.md) §6 is
   preserved and the router landing pressures
   [ADR-0051](./0051-claude-tooling-postwave3.md) Notes
   OQ-2 toward resolution.** The
   [`/sync-agents`](../../.claude/commands/sync-agents.md)
   drift-check already covers the required-reading
   section, so the router lands inside the drift surface
   without expanding its scope. The structural change to
   §6 — introducing a new taxonomy that future playbook
   additions must register against — is exactly the kind
   of surface that pressures OQ-2's resolution. The first
   time a new playbook is added without a router row, the
   OQ-2 ruling becomes load-bearing.

10. **Per-B3-N items continue to follow the same loop.**
    No new playbook is introduced by this ADR. Existing
    `post-wave3-session-loop.md`, `acceptance-criteria.md`,
    and `feedback-protocol.md` apply unchanged. If a
    router-specific delta emerges from the first
    post-router sessions, a follow-up edit is registered
    then — not pre-emptively.

11. **The agent-harness-as-adjacent-tooling reading from
    [ADR-0051](./0051-claude-tooling-postwave3.md)
    Clause 1 is reused, not extended.** This ADR sits
    inside the surface that ADR-0051 admitted under
    [ADR-0049](./0049-b3-evolutionary-launch.md)
    §Per-family scope's "adjacent tooling" clause; it does
    not open a new expansive reading. Future B3-N entries
    that touch agent-harness territory continue to cite
    ADR-0051 Clause 1 as the precedent; entries that touch
    adjacent-tooling surfaces not yet admitted (reviewer
    tooling, observability harness) follow the same
    new-contribution-requiring-review discipline rather
    than absorb a reading silently.

12. **The D0 ratification mechanism is committed as
    precedent for future borderline B3-N readings.** The
    author-equals-reviewer circularity recognized by
    [ADR-0051](./0051-claude-tooling-postwave3.md)
    §Consequence 7 — `/critique` cannot self-ratify its
    own eligibility reading — is now load-bearing for two
    B3-N rows (`B3-1`, `B3-2`). Future borderline B3-N
    sessions cite ADR-0052 (alongside ADR-0051
    §Consequence 7) as the precedent for the
    operator-side ratification step.

---

## Notes

Five open questions remain explicitly out-of-scope for
this ADR; each has a named trigger condition.

- **OQ-1 —
  [`/start-session <type>`](../../.claude/commands/) command.**
  Whether a new command should consume the Clause 2 table
  and surface the per-type reading set at session open is
  a follow-on tooling question, not load-bearing for this
  ADR. Resolved by the first B3 / B2 session run *after*
  the router lands in [`CLAUDE.md`](../../CLAUDE.md) §6,
  when the cost of manually re-reading the router becomes
  visible enough to justify the command. Until then, the
  router is read directly from
  [`CLAUDE.md`](../../CLAUDE.md) §6.

- **OQ-2 —
  [`session-governance`](../../.claude/skills/session-governance/SKILL.md)
  skill consumption of the router.** Whether the skill's
  description triggers should cover the session-open
  moment (loading the per-type reading set) in addition
  to the PR / branch / commit moments is a follow-on
  tooling question. The skill today triggers on verbatim
  phrases at the PR-flow moments only; adding
  session-open triggers requires harness verification
  that the description-match still loads reliably.
  Resolved by the first session-governance-skill update
  following the router landing.

- **OQ-3 — Routing axis: session type vs. artifact type
  vs. composition of both.** An alternate routing axis
  (route on the artifact being produced — study /
  scaffold / ADR / process edit — rather than on the
  session type) collapses three of the six session types
  into "produces a study" but does not distinguish
  between B2 and B3 (both produce studies). The
  session-type axis committed by Clause 2 is the
  recommended axis because it preserves the
  eligibility-check distinction (B2 vs. B3) that
  [ADR-0049](./0049-b3-evolutionary-launch.md) §(a)
  commits. A third possibility surfaced during this
  ADR's drafting: the two axes may **compose** — route
  on session type first, then route on artifact type
  within each row. Composition would refine, e.g., the
  B3-entry row to "B3 entry producing a study" vs.
  "B3 entry producing a tooling artifact" with different
  reading subsets. This ADR does not commit composition;
  if a future B3-N entry reveals that within-row variance
  is material, composition is the documented re-shape
  path.

- **OQ-4 — Wave-S session type.** Wave-S is still
  partially open (full-gate criteria pending per
  [ADR-0020](./0020-wave-s-launch.md)). A hypothetical
  Wave-S follow-up session (B0-S4 / B0-S5 / …) would
  arguably need its own router row, but the partial gate
  met 2026-05-24 means Wave-S sessions that have run
  since then are effectively B3 entries against the
  partial-gate envelope. The router does not commit a
  separate Wave-S row at this time; if the full gate
  reveals new session shape, a follow-up B3-N adds it.

- **OQ-5 — Attention-cost measurement for the router's
  cost-savings claim.** Consequence 1 above commits a
  by-line-count saving without benchmarking against
  actual attention cost. The first 2–3 post-router
  sessions provide an empirical measurement window; the
  operator notes for each session (a) which router row
  applied, (b) which playbooks were actually read in full
  vs. skimmed, and (c) whether any playbook outside the
  per-type set turned out to be load-bearing mid-session.
  **Forcing function:** the measurement-decision PR opens
  by the **third post-router session at the latest**,
  regardless of whether 2–3 sessions have surfaced enough
  signal — this prevents indefinite deferral. If the
  attention-cost saving turns out to be materially
  smaller than the line-count saving, a follow-up B3-N
  entry revisits the router shape.

A practical note on the D0 ratification mechanism
committed in §Context: when a future B3-N session's
eligibility reading under
[ADR-0049](./0049-b3-evolutionary-launch.md) §(a) is
borderline, the operator-side ratification is recorded in
the round-2 critique trailer per
[ADR-0048](./0048-critique-rounds-preservation.md)
preservation discipline and per
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5
§"Operator-side responsibilities". The promoted ADR
propagates the ratification as a
new-contribution-requiring-review marker per
[`CLAUDE.md`](../../CLAUDE.md) §3 R5 (and per A7 of the
[`adr-writing`](../../.claude/skills/adr-writing/SKILL.md)
skill). This ADR is the second
load-bearing example of the mechanism after
[ADR-0051](./0051-claude-tooling-postwave3.md); future
borderline B3-N readings cite both as precedent.

A "rejected" outcome distinction registered by this ADR
for future use: a proposal whose §(a) Conditions 1 or 4
fail in a way that re-routes the substance to
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6 or to an
amendment ADR is **not** the same as
[ADR-0049](./0049-b3-evolutionary-launch.md) §(a)
`rejected`. §(a) `rejected` carries the stronger
semantic "no lane fits at all"; a re-routed proposal is
"ineligible for B3 but valid elsewhere". The decision log
should record the distinction explicitly when it arises,
rather than collapsing both cases into the same outcome
label. The
[ADR-0049](./0049-b3-evolutionary-launch.md)
§Consequence 8 deferral on the `rejected`-outcome
vocabulary integration may pick this distinction up when
it lands.
