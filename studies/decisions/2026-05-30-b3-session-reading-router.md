<!-- path: studies/decisions/2026-05-30-b3-session-reading-router.md -->

# B3-2 — Session Reading Router for `CLAUDE.md` §6

## Metadata

- **Wave reference:** B3 (evolutionary lane; tooling extensions
  family per
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
  §Per-family scope; agent-harness sits inside the "adjacent
  tooling" reading admitted by
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clause 1 — this study reuses that precedent, it does not
  open a new expansive reading).
- **Status:** draft (B3-2, session 1; post-round-1-critique,
  pre-round-2).
- **Last updated:** 2026-05-30.
- **Upstream resolved:**
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) (B3
  eligibility filter and family list);
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clauses 1, 2, 4 (the "adjacent tooling" precedent admitting
  the `.claude/` harness; `CONTRIBUTING.md` as upstream
  authority; the `post-wave3-session-loop.md` playbook this
  router maps for one of its session types);
  [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md)
  §Consequence 4 (the Amendment-log subsection convention this
  router will route ADR-amendment sessions through);
  [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)
  (critique-rounds preservation contract; this study preserves
  rounds against it);
  [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5 (the
  PR-flow contract for post-Wave-3 sessions, named by the
  router as one of the always-on reads) and Flow 6 (the
  direct-edit lane the router names as a distinct session
  type).
- **Eligibility check (ADR-0049 §(a)):**
  - **Condition 1 — P-B3.1, expands not rewrites.** The router
    adds a *routing layer* over `CLAUDE.md` §6's current
    universal "read all playbooks" prescription. No playbook is
    edited; no rule R1–R8 is modified; no principle P1–P6 is
    altered. What the router commits is a per-session-type
    mapping from the existing playbook set to a minimal
    required-reading subset, plus an "always-on floor" that
    every session reads regardless of type. The substance of
    every playbook is preserved verbatim. **Borderline of
    interpretation acknowledged:** replacing a universal
    prescription with a per-type prescription is structurally
    additive at the playbook layer (each playbook stays
    intact) but textually narrows one sentence in `CLAUDE.md`
    §6 from "read them at the start of every session" to "read
    the subset declared by the router". Whether that textual
    narrowing counts as a *rewrite* of the §6 prescription —
    and therefore as a P-B3.1 violation — or as an *extension*
    that preserves the prescription's underlying intent
    (every relevant playbook still loads when its session type
    runs) is the load-bearing question. Resolution is parked
    as a D0 precondition in §Decision Drivers and surfaces in
    `/critique` as an explicit ruling. ⚠️ pending operator
    ratification per
    [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5
    §"Operator-side responsibilities".
  - **Condition 2 — P-B3.4, in-scope family — Tooling
    extensions.** Family fit is **direct precedent reuse**, not
    a new reading. [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
    Clause 1 already committed the expansive reading admitting
    the `.claude/` agent harness — including `CLAUDE.md` itself
    — under ADR-0049 §Per-family scope's "and adjacent
    tooling" clause; this study lives inside that admitted
    surface and does not extend it further. Operator
    ratification on the family fit is already on record in the
    B3-1 close trailer (ratified 2026-05-29 during the
    `/critique` round-2 trailer per
    [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5
    §"Operator-side responsibilities"). ✅
  - **Condition 3 — P-B3.2, conforms to
    ADR-0020 / 0021 / 0022 / 0023.** The router touches no
    substrate decision, no mode primitive, no kind catalog
    entry, no sources schema row. Envelope is untouched. ✅
  - **Condition 4 — additive-maintenance threshold.** The
    router commits a new **session-type taxonomy** (B2
    follow-up / B3 entry / ADR amendment / ADR promotion /
    Flow 6 process edit) that becomes load-bearing for the
    harness: future playbook additions must declare which
    session types they belong to, and the
    `session-governance` skill will eventually read the router
    to know which playbooks to surface at session open. That
    materially reshapes contributor expectations above a
    routine documentation refresh — it changes how an agent
    or operator grounds itself at session start. **Borderline
    of materiality acknowledged:** an alternate reading frames
    this as a "tight clarification of an existing rule whose
    substance is unchanged" and routes it through
    [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6 instead
    (operator-authorized direct edit, no B-row). The argument
    for the B3 path: the new taxonomy crosses the threshold
    because it introduces a *concept* the harness did not
    previously carry, and `/check-decision-backlog` plus a
    future `/critique` will need to recognize the session
    type to apply the right gates. The argument for the
    Flow 6 path: the actual edit is a single small table in
    a single section of a single process document. Resolution
    is parked as a D0 precondition in §Decision Drivers and
    surfaces in `/critique` as an explicit ruling. ⚠️ pending
    operator ratification per
    [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5
    §"Operator-side responsibilities".
- **Constraint envelope:**
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §(a)
  (eligibility), §(b) (out-of-scope), §Per-family scope
  (Tooling extensions);
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clause 1 (adjacent-tooling precedent), Clause 2
  (`CONTRIBUTING.md` as upstream authority for PR-flow),
  Clause 4 (the `post-wave3-session-loop.md` shape that the
  router maps for B2 / B3 / amendment / promotion sessions);
  [`CLAUDE.md`](../../CLAUDE.md) §3 R1–R8 (especially R4
  one-topic-per-session, R5 own-the-pattern, R6 path header,
  R8 studies-are-not-the-product);
  [`CLAUDE.md`](../../CLAUDE.md) §4 P1–P6 (especially P5
  contract-driven evolution; the router is itself a contract
  with the contributor population and evolves under §6's
  published shape).
- **Locked premises** (operator-declared, not litigated here):
  - **P-B3RR.1** — The router is **additive over the floor**.
    R1–R8 (CLAUDE.md §3) and P1–P6 (CLAUDE.md §4) reading is
    mandatory for every session and is not subject to the
    router. `CLAUDE.md` §1 already mandates reading the entire
    file before producing output; the router does not change
    that mandate. The router governs only which **playbooks**
    a session loads, not which **rules** apply.
  - **P-B3RR.2** — Wave-1 and Wave-3 playbooks are historical
    as of this study's date. Their content is preserved
    verbatim (no edits, no deletions); the router classifies
    them as "skim-if-curious" for every current session type.
    Their preservation answers an audit-trail need; their
    removal from any current session's required-reading set
    answers a session-cost need.
  - **P-B3RR.3** — No platform code changes. R1 is historical
    now that Wave 3 closed, but R4 (one-topic-per-session)
    still applies: this study touches only
    `CLAUDE.md` §6, the decision log row addition, and (if
    promoted) ADR-0052. No engine, rules, tools, or deploy
    files move.
  - **P-B3RR.4** — The **six** session types named by the
    router (B2 follow-up / B3 entry / ADR amendment / ADR
    promotion / Flow 6 process edit / implementation slice
    landing under a closed B-row — the sixth row added in
    round-1 revision per F5) are the exhaustive set under the
    current operating shape. Three boundary cases were
    considered and resolved:
    - **B-row triage** (the operator deciding whether a newly
      surfaced item is B2 / B3 / amendment / rejected per
      ADR-0049 §(a)). Triage is a one-pass classification
      that runs *before* the session opens; it does not
      itself produce a study, an ADR, or a scaffold. The
      operator reads ADR-0049 §(a) and the decision log,
      both already in the always-on floor. No router row
      needed.
    - **Study revival** (re-opening a previously deferred
      study). Revival follows the originating row's type
      (B2 if originally B2; B3 if originally B3) — the
      router routes by *current* session type, not by
      originating wave. No separate row.
    - **ADR supersession** (the
      [ADR-0017](../../docs/adr/0017-substrate-posture-amendment.md)
      pattern). Supersession is a special case of ADR
      amendment when the amendment reshapes architectural
      prose rather than touching only structured data
      (ADR-0050 §Consequence 4 boundary). The router's
      "ADR amendment" row already covers it; the
      ADR-0050 in-place vs. ADR-0017 supersession choice
      is made inside the session.
    Future session types beyond the six (e.g., a
    rule-promotion lane, a release-cut lane, an automation
    lane) require a router-row addition through a follow-up
    B3-N entry; the router is not silently extended.
- **Downstream open:** none enumerated. If `/critique`
  surfaces a blocking finding that requires a sixth session
  type or a different routing axis (e.g., routing on artifact
  type rather than session type), it is registered in §Open
  Questions and the study re-scopes — it does not silently
  grow.
- **Critique rounds:**
  round 1 preserved
  ([`studies/critiques/2026-05-30-b3-session-reading-router-critique-1.md`](../critiques/2026-05-30-b3-session-reading-router-critique-1.md)) —
  0 blocking / 5 important / 5 minor; all dispositioned in
  the Operator Response trailer per
  [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)
  §"Skip" grammar.
- **Promotion target:**
  `docs/adr/0052-session-reading-router.md` — provisionally
  the next available number at the time of writing (last
  landed is
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md),
  2026-05-29; reservation is operator-side per
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clause 7, confirmed at `/promote-to-adr` time).

---

## Context

`CLAUDE.md` §6 currently prescribes a **uniform required-reading
set** at session start: every session is asked to read every
playbook under `.claude/playbooks/`, alongside the foundation
documents. The six playbooks total ~700 lines:

- [`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  (~140 lines; Wave 1 closed 2026-05-21).
- [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
  (~170 lines; Wave 3 closed 2026-05-23).
- [`post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md)
  (~225 lines; the current operational loop).
- [`acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md)
  (~35 lines; B0-shaped, inherited by B2 / B3 sessions).
- [`wave-3-acceptance-criteria.md`](../../.claude/playbooks/wave-3-acceptance-criteria.md)
  (~70 lines; scaffold-shaped, Wave 3 only).
- [`feedback-protocol.md`](../../.claude/playbooks/feedback-protocol.md)
  (~65 lines; shared across all study/review work).

The uniform prescription was correct when written: during Wave 3,
the active loop was the Wave-3 loop, and the Wave-1 loop was the
shape it mirrored. A contributor opening a session could realistically
need every playbook in scope.

Today, the operating shape has changed. Three concrete sessions
that have run since Wave 3 closed illustrate the cost:

1. **B3-1 (the `.claude/`-harness extension itself).** A B3 entry
   that required `post-wave3-session-loop.md` (which it was
   *producing*), `acceptance-criteria.md`, `feedback-protocol.md`,
   ADR-0049 §(a), and the prior `wave-1-session-loop.md` *as the
   shape reference*. The Wave-3 acceptance criteria and Wave-3
   session loop were read but never load-bearing for the session's
   output.
2. **The 2026-05-30 CLAUDE.md refresh (Flow 6, PR #98).** A Flow 6
   direct edit to `CLAUDE.md` for post-Wave-3 state. Required
   reading: CONTRIBUTING.md Flow 6 + PR-flow step from
   `post-wave3-session-loop.md` step 10. The five playbooks beyond
   step 10 were read but not load-bearing; the Wave-1 / Wave-3
   loops and acceptance-criteria sets were inert.
3. **The 2026-05-30 ADR-0051 settings-naming-drift PR (Flow 6).**
   A single-file in-place correction to an ADR's prose. Same
   minimal load-bearing reading as case 2; same inert majority.

The pattern is generalizable. Each currently-active session type
loads a small, specific subset of the playbook corpus; the
remaining playbooks are either historical (Wave 1, Wave 3) or
scope-specific (Wave 3 acceptance criteria), and reading them is
not load-bearing for that session's output. The uniform
prescription forces every session to pay the cost of reading
non-load-bearing artifacts.

Two governance gaps motivate fixing this at the §6 layer rather
than letting each session improvise:

1. **No documented mapping from session type to minimal reading
   set.** A contributor opening a Flow 6 process-edit session
   today has no way to know — without re-deriving from the
   playbook texts — which playbooks are actually load-bearing
   for that session. The uniform "read all" prescription
   over-reads; an under-reading improvisation under-reads with no
   audit surface.
2. **No taxonomy of post-Wave-3 session types.** Six distinct
   session types operate in the post-Wave-3 lane (B2 follow-up,
   B3 entry, ADR amendment, ADR promotion, Flow 6 process edit,
   and implementation slice landing under a closed B-row — the
   sixth surfaced explicitly in round-1 critique F5), each with
   its own grounding needs. They are scattered across
   `post-wave3-session-loop.md`, `CONTRIBUTING.md` Flow 5,
   `CONTRIBUTING.md` Flow 6, [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md)
   §Consequence 4 (the Amendment-log convention), and
   [`wave-3-acceptance-criteria.md`](../../.claude/playbooks/wave-3-acceptance-criteria.md)
   (the scaffold-shaped AC-W3 set that still governs
   implementation slices). No single surface enumerates the
   six types or maps them to required reading.

This study proposes a **session reading router** in `CLAUDE.md`
§6: a short decision table that maps session type to minimal
required-reading set, with an explicit always-on floor and an
explicit historical-skim set. R1–R8 and P1–P6 are not subject
to the router; they apply to every session unconditionally.

---

## Decision Drivers

### D0 — Operator ratification on borderline conditions (precondition)

The eligibility reading committed by the Metadata block parks
two borderline conditions:

- **Condition 1 (P-B3.1, expands not rewrites)** — does
  narrowing the §6 prescription from universal to per-type
  count as a rewrite, or as an extension that preserves the
  prescription's underlying intent?
- **Condition 4 (additive-maintenance threshold)** — does the
  new session-type taxonomy cross the materiality threshold
  for B3, or is this a Flow 6 tight clarification?

D0 is a **precondition** on advancing this study to
`resolved-study`: the Recommendation below is conditional on
both readings clearing during `/critique` and operator
ratification per
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5
§"Operator-side responsibilities". The author-equals-reviewer
circularity that B3-1 surfaced (ADR-0051 §Consequence 7)
applies here too: an agent-side `/critique` cannot
self-ratify its own eligibility reading. The ratification is
recorded in the round-2 critique trailer (per
[ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md))
and carries forward to the promoted ADR as a
new-contribution marker per R5.

If D0 fails (the operator rejects either reading), two
distinct mechanisms apply and must not be conflated:

- **Formal ADR-0049 §(a) outcome.** A failed eligibility
  reading registers in the decision log as one of the
  §(a) outcomes — `rejected` if the proposal genuinely
  falls outside the three in-scope families *and* outside
  any active wave's gate, or as `amendment` / `B2` if the
  re-triage names a different lane. A Condition-4
  materiality failure on its own does **not** make the
  proposal "outside the family" (the agent-harness fit
  under Tooling extensions is settled by ADR-0051
  Clause 1); it just makes B3 the wrong lane for the
  materiality of the change. In that case the §(a)
  outcome is `rejected` only if no other lane fits.
- **Practical re-routing follow-up.** If the proposal's
  *substance* is still worth shipping after a failed §(a)
  reading, the operator opens a new PR under the
  appropriate flow — Flow 6 if the substance is a tight
  clarification of an existing rule whose substance is
  unchanged; an amendment ADR if Condition 1 fails because
  the change reshapes a committed contract. The re-routing
  PR is a separate session with separate provenance, not a
  continuation of B3-2.

The decision log records the §(a) outcome; the re-routing
PR carries the substance. Both surfaces remain auditable.

### D1 — Preserve R1–R8 and P1–P6 reading floor

R1–R8 and P1–P6 are non-negotiable for every session
(CLAUDE.md §1: "Read this entire file before producing any
output"). The router never narrows the rule and principle
reading; it narrows only the playbook reading. This is the
load-bearing safety floor. A router that erodes the rule
reading by accident is **not acceptable** regardless of
session-cost savings.

### D2 — Preserve Wave-1 and Wave-3 playbook content verbatim

Wave-1 and Wave-3 playbooks remain on disk unchanged after
this study lands. The router classifies them as
historical-skim — it does not delete them, edit them, or
relocate them. Their audit-trail value is permanent; their
load-bearing value for current sessions is zero. The router
makes that distinction visible without erasing the history.

### D3 — Minimum sufficient set per session type

For each session type, the required-reading set is the
*minimum* that makes the session's load-bearing decisions
auditable against R1–R8, P1–P6, and the relevant acceptance
criteria. A reading set that omits a playbook the session
genuinely depends on is **wrong** even if it saves cost. The
six session-type-to-reading-set mappings below are
defended individually.

### D4 — Future session types extend the router, not bypass it

The router commits six session types as the current
exhaustive set (per P-B3RR.4's boundary-case analysis). A
future session type (a rule-promotion lane, a release-cut
lane, an automation lane) requires a router-row addition
through a follow-up B3-N entry. The
router is not silently extended; new session types ride
the same B3 study trail this proposal does.

### D5 — Cross-cutting concerns surface through always-on reads, not through per-type reads

Some concerns (PR-flow discipline, decision-log state,
critique-round preservation) cross every session type. They
ride the always-on floor (CONTRIBUTING.md, the decision log)
rather than being duplicated in per-type rows. This keeps
the per-type rows narrow and the floor explicit.

---

## Considered Options

### Option A — Per-type subset router with explicit always-on floor

A new short decision table in `CLAUDE.md` §6. Six rows
(one per session type — the sixth added in round-1 revision
per F5), three columns:

- Session type — short label.
- Trigger — when this type applies.
- Minimal required playbook reading — the load-bearing subset
  for this type.

Above the table, an always-on floor block listing CLAUDE.md,
AGENTS.md, CONTRIBUTING.md, and the decision log as
unconditional reads, plus a default-up safety rule for
borderline classifications.

Below the table, a historical-skim block listing the Wave-1
and Wave-3 session-loop playbooks as "not load-bearing for
any current session type; skim if you are reading prior
sessions for context". `wave-3-acceptance-criteria.md` is
**not** in the historical-skim set under Option A's revised
shape — it is load-bearing for implementation-slice sessions
(round-1 F5).

**Trade-offs.** Explicit; auditable; reviewable per session
type. Adds a new taxonomy that future playbook additions must
declare against. Surface is one table in one section, so the
maintenance cost is localized. Requires keeping the router in
sync with the playbook corpus (adding a playbook means adding
a column entry per session-type row, or declaring why the
playbook is universal).

### Option B — Keep `CLAUDE.md` §6 as-is; add a session-type guide elsewhere

Leave the universal-read prescription in place; ship a new
guide (e.g., `.claude/playbooks/session-types.md`) that lists
the six session types and their load-bearing reading sets.
Sessions read the guide first, then narrow their own reading
informally.

**Trade-offs.** No edit to `CLAUDE.md`; lower disruption to
the constitutional document. Loses the cost-savings benefit:
each session still has to read every playbook because §6
still says so. The guide adds a seventh playbook to read,
not narrows the existing six. Risks formalizing two
contradictory prescriptions (§6 says "read all"; the guide
says "read subset"), which is worse than not adding the
guide at all.

### Option C — Mark Wave-1 and Wave-3 playbooks as historical in §6; no per-type taxonomy

A minimal `CLAUDE.md` §6 edit: tag the two historical
playbooks with "(historical reference, skim only)" and leave
the rest universal. Drops the universal-reading cost by ~310
lines but does not distinguish between, say, a B3 entry and
a Flow 6 process edit — both still read
`post-wave3-session-loop.md` end-to-end, even though Flow 6
only needs step 10.

**Trade-offs.** Smallest edit; preserves universal-reading
discipline for the current playbooks. Captures the
"historical" signal that the current §6 buries in a phrase
per row. Does not address the second governance gap
(no taxonomy of session types). Sessions continue to
over-read the current loop and acceptance criteria for
session types whose load-bearing surface is narrower.

### Option D — Tiered reading: Tier 1 (always) / Tier 2 (current ops) / Tier 3 (historical)

Three explicit tiers in `CLAUDE.md` §6 instead of a
per-type table. Tier 1 = R/P + decision log + CONTRIBUTING +
AGENTS. Tier 2 = every current playbook
(`post-wave3-session-loop.md`, `acceptance-criteria.md`,
`feedback-protocol.md`). Tier 3 = Wave-1, Wave-3, and Wave-3
acceptance criteria. Sessions read Tier 1 + Tier 2; Tier 3
optional.

**Trade-offs.** Cleaner conceptual model than A (tiers are
simpler than session types). Does not distinguish between
session types whose Tier 2 reads diverge — Flow 6 needs only
the PR-flow step of the post-wave3 loop, not the whole
playbook; an ADR promotion session has a different load-bearing
subset than a B3 entry. Tier semantics are less directly
actionable than session-type-to-playbook map; a contributor
opening a Flow 6 session still has to derive *which Tier 2
content* applies. The cost-savings benefit is captured for
historical playbooks (~310 lines) but not for per-session
narrowing within Tier 2 (~225 lines of post-wave3 loop that
Flow 6 mostly skips).

---

## Recommendation

**Option A — Per-type subset router with explicit always-on
floor.**

It is the only option that directly addresses **both**
governance gaps surfaced in §Context (no session-type
taxonomy; no documented mapping from type to reading set).
Option B preserves the existing universal prescription and
loses the cost-savings benefit. Option C addresses the
historical-playbook split but leaves the per-session
over-reading inside the current playbook corpus untouched.
Option D captures the historical/current split but not the
per-session-type narrowing inside the current set.

D3 is the load-bearing test: each session type's minimal
reading set must be the smallest subset that keeps the
session's decisions auditable against R/P/AC. Option A is
the only option where that subset is named per type, so it
is the only option where D3 is actually defensible row by
row.

The router lives as a new sub-section under `CLAUDE.md` §6,
before the existing playbook list. The existing list is
preserved verbatim immediately below the router as the
canonical descriptions of each playbook; the router maps
session types into that list. R1–R8 and P1–P6 stay where
they are (§3 and §4); §6 governs only the playbook surface.

### The router

**Always-on floor** (every session, regardless of type):

- [`CLAUDE.md`](../../CLAUDE.md) §1–8 in full. §1's
  "read this entire file before producing any output"
  mandate covers R1–R8 (§3), P1–P6 (§4), the phase/lane
  taxonomy (§2), the session-reading router (§6, including
  this router as the intentional self-reference — every
  session reads §6 to find its own row), and the slash
  commands (§7).
- [`AGENTS.md`](../../AGENTS.md) (cross-agent convention
  file; rebinds the rules to non-Claude agents).
- [`CONTRIBUTING.md`](../../CONTRIBUTING.md) (PR-flow
  contract, authoritative per ADR-0051 Clause 2; Flow 5 is
  the post-Wave-3 PR-flow; Flow 6 is the direct-edit lane).
- [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md)
  (live state surface for every B-row, ADR status, and
  Wave-S gate status).

**Default-up safety rule.** If a contributor or agent is
unsure between two session types — e.g., between Flow 6
process edit and B3 entry on a borderline materiality
call — **default to the next-larger reading set**. The
larger set is a superset; reading it does not violate any
rule and costs at most one extra playbook. The router
narrows confidently-classified sessions; uncertain
classification reverts to the universal-prescription
behavior for that session only.

**Per-type reading sets:**

| Session type | Trigger / when to use | Minimal required playbook reading (beyond the floor) |
|---|---|---|
| **B2 follow-up** | A B-row marked B2 in the decision log; resolves an implementation-phase decision against an in-flight wave. | [`post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md) (step 2's wave-gate confirmation is load-bearing); [`acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md) (AC-1…AC-10 — B2 studies inherit B0 study shape per ADR-0051 OQ-1); [`feedback-protocol.md`](../../.claude/playbooks/feedback-protocol.md). |
| **B3 entry** | A B-row marked B3 in the decision log; ADR-0049 §(a) eligibility filter must clear before drafting. | [`post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md) (step 2's eligibility-check sub-step is load-bearing); [`acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md); [`feedback-protocol.md`](../../.claude/playbooks/feedback-protocol.md); [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §(a) and §(b). |
| **ADR amendment** | In-place edit to an existing ADR (structured-data row amendment or Amendment-log subsection per ADR-0050 §Consequence 4); no decision rewrite. | [`post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md) step 10 (PR-flow close); [`feedback-protocol.md`](../../.claude/playbooks/feedback-protocol.md); the originating ADR; [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md) §Consequence 4 (Amendment-log convention). [`acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md) optional — only if the amendment produces a study. |
| **ADR promotion** | Running [`/promote-to-adr`](../../.claude/commands/promote-to-adr.md) on a `resolved-study`. | [`post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md) step 10 (PR-flow close); [`acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md) (the source study must have cleared AC-1…AC-10 before promotion); [`feedback-protocol.md`](../../.claude/playbooks/feedback-protocol.md) (the promotion may surface critique-style feedback on the proposed ADR text); [`/promote-to-adr`](../../.claude/commands/promote-to-adr.md) command spec. |
| **Flow 6 process edit** | Operator-authorized direct edit to `CLAUDE.md` / `AGENTS.md` / `.codex/AGENTS.md` per [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6. | [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6 (scope-and-gate; the load-bearing contract — Flow 6 explicitly inherits Flow 5 PR-flow); [`post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md) step 10 (the load-bearing playbook content for PR-flow close — Flow 6 inherits the rest of Flow 5 through CONTRIBUTING.md, not through the playbook). [`feedback-protocol.md`](../../.claude/playbooks/feedback-protocol.md) optional; **load-bearing if `/critique` is run** (the [`/critique`](../../.claude/commands/critique.md) command grounds on it). |
| **Implementation slice landing under a closed B-row** | A code or scaffold slice that lands the artifacts committed by a closed B-row's ADR (e.g., an ADR-0051 follow-on slice shipping the four named artifacts; a Wave-3 follow-up slice landing a deferred capability matrix row). | [`post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md) (the close-discipline); [`wave-3-acceptance-criteria.md`](../../.claude/playbooks/wave-3-acceptance-criteria.md) (AC-W3-3 load-bearing for citation discipline; AC-W3-7 load-bearing for local build/lint/test gates — both are scaffold-shaped semantics that apply to any post-Wave-3 implementation slice); [`feedback-protocol.md`](../../.claude/playbooks/feedback-protocol.md). [`acceptance-criteria.md`](../../.claude/playbooks/acceptance-criteria.md) optional — only if the slice produces a follow-up study. |

**Historical-skim set** (not load-bearing for any current
session type; preserved for audit-trail and shape-reference
purposes; skim only when explicitly reading prior sessions
for context):

- [`wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  — Wave 1 closed 2026-05-21. Shape reference for the
  `post-wave3-session-loop.md` ten-step structure.
- [`wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
  — Wave 3 closed 2026-05-23. Shape reference for the PR-flow
  discipline now authoritative in CONTRIBUTING.md Flow 5.

Note: [`wave-3-acceptance-criteria.md`](../../.claude/playbooks/wave-3-acceptance-criteria.md)
(AC-W3-1…AC-W3-10) was initially placed in the
historical-skim set in this study's draft, but round-1
critique F5 surfaced that AC-W3-3 and AC-W3-7 are still
load-bearing for implementation-slice sessions landing
under a closed B-row. It is therefore named load-bearing in
the **Implementation slice** row above; the rest of the
AC-W3 set (path-header, English, R5 hygiene, etc.) overlaps
with R1–R8 and is read through the always-on floor.

### What the router does not change

- **R1–R8 reading is mandatory for every session.** §3 still
  says so; §6 does not override §3.
- **P1–P6 application is universal.** §4 still says so; §6
  does not override §4.
- **R6 path-header convention** applies to every produced
  markdown file regardless of session type.
- **R8 studies-are-not-the-product** applies whenever a
  promoted ADR is written.
- **The
  [`session-governance`](../../.claude/skills/session-governance/SKILL.md)
  skill** loads on description-match basis at the PR / branch
  / commit moments, independent of session type. It does not
  consult the router; it triggers on contributor phrases.
- **No playbook is edited, deleted, or relocated.** The
  router is purely additive at the playbook layer.

---

## Consequences

1. **Each session reads less; the per-session line count
   drops.** The maximum saving (a Flow 6 process edit) is
   roughly 600 lines of playbook text avoided (~85% of the
   current corpus by line count). The minimum saving (an
   implementation-slice or B3 entry) is roughly 280 lines
   of historical-skim content (Wave-1 loop, Wave-3 loop).
   The numbers are **by line count, not benchmarked against
   attention cost** — structured prose and dense code do not
   carry equivalent reading load, and the actual session-load
   drop may be smaller in practice. The first 2–3 post-router
   sessions provide a measurement window; OQ-5 below commits
   to revisit the cost claim with concrete evidence.

2. **Session-type taxonomy becomes a load-bearing concept.**
   Future playbook additions declare their session-type
   applicability in the router; future playbooks that apply
   universally become part of the always-on floor explicitly.
   The router itself becomes a contract surface that future
   B3-N entries amend.

3. **R1–R8 and P1–P6 reading floor is preserved.** D1 is
   satisfied. The router never narrows rule or principle
   reading; CLAUDE.md §1's "read the entire file" mandate
   covers the floor before §6 even applies.

4. **Wave-1 and Wave-3 playbooks become explicitly
   historical.** D2 is satisfied — content is preserved
   verbatim; their relegation to the historical-skim set
   matches their closed-wave status without erasing the
   audit trail. The `post-wave3-session-loop.md` becomes
   the only currently-active loop, and §6 says so directly.

5. **A new session type requires a router-row addition
   through a follow-up B3-N entry.** D4 is satisfied — the
   router does not silently extend. A future automation lane
   or release-cut lane will be born with its own row, not
   absorbed into "B3 entry" or "Flow 6 process edit" as a
   convenience.

6. **The `session-governance` skill is unaffected.** Its
   description-match triggering on PR / branch / commit
   phrases (per
   [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
   Clause 3) is orthogonal to the router. The skill loads
   when the contributor reaches the PR moment, regardless of
   session type. No edit to
   [`SKILL.md`](../../.claude/skills/session-governance/SKILL.md)
   is committed by this study.

7. **A small risk surfaces around under-reading drift.** A
   session that mis-classifies its own type (e.g., what the
   contributor thinks is a Flow 6 edit is actually a B3
   entry because the change is more material than it looks)
   will under-read. Mitigation: the router includes the
   trigger column so the classification has a concrete test,
   and the always-on floor (which includes CONTRIBUTING.md
   Flow 6's scope clause) catches the misclassification on
   read. If a session begins as Flow 6 and the operator
   realizes mid-session it should be B3, the session aborts
   per
   [`post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md)
   §"When to abort the session" and re-opens with the
   correct classification.

8. **`/check-decision-backlog` is unaffected; future
   tooling may consult the router.** The current command
   does not need to read the router (it already returns the
   B-row backlog without session-type filtering). A future
   tooling extension (perhaps a `/start-session <type>`
   command, or a session-governance-skill enhancement that
   surfaces the router subset at session open) would consume
   the router as data. Such a tooling extension is deferred
   to a follow-up B3-N entry and is **not** committed by
   this study.

9. **`/sync-agents` coverage of `CLAUDE.md` §6 is preserved
   and may force ADR-0051 OQ-2 earlier.** The router lives
   inside §6. The
   [`/sync-agents`](../../.claude/commands/sync-agents.md)
   drift-check already covers the required-reading section,
   so the router itself lands inside the drift surface
   without expanding its scope. However, a structural change
   to §6 (introducing a new taxonomy that future playbook
   additions must register against) is exactly the kind of
   surface that pressures
   [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
   Notes OQ-2 (`/sync-agents` coverage for skills, playbooks,
   and command inventory). The router landing is registered
   here as a concrete forcing function for OQ-2's resolution;
   the first time a new playbook is added without a router
   row, the OQ-2 ruling becomes load-bearing.

10. **The promoted ADR-0052 commits the router as a
    contract.** Future router amendments (a new session type;
    a re-mapping of a session type's reading set) ride
    follow-up B3-N entries that amend or supersede
    ADR-0052. The router is no longer free-form prose; it
    is contract-driven (P5).

---

## Open Questions

- **OQ-1 — `/start-session <type>` command.** Whether a new
  command should consume the router and surface the per-type
  reading set at session open is a follow-on tooling
  question, not load-bearing for this study. Resolved by the
  first B3 / B2 session run *after* this router lands, when
  the cost of manually re-reading the router becomes visible
  enough to justify the command. Until then, the router is
  read directly from `CLAUDE.md` §6.
  *Out-of-scope for current cycle:* this study commits the
  router; the tooling that consumes it is a separate B3-N.

- **OQ-2 — `session-governance` skill consumption of the
  router.** Whether the skill's description triggers should
  cover the session-open moment (loading the per-type
  reading set) in addition to the PR / branch / commit
  moments is a follow-on tooling question. The skill today
  triggers on verbatim phrases at the PR-flow moments only;
  adding session-open triggers requires harness verification
  that the description-match still loads reliably. Resolved
  by the first session-governance-skill update following
  this router landing.
  *Out-of-scope for current cycle:* deferred to whenever the
  skill's coverage is next revisited.

- **OQ-3 — Routing axis: session type vs. artifact type vs.
  composition of both.** An alternate routing axis (route on
  the artifact being produced — study / scaffold / ADR /
  process edit — rather than on the session type) was
  considered briefly during drafting. It collapses three of
  the now-six session types into "produces a study" but does
  not distinguish between B2 and B3 (both produce studies).
  The session-type axis is the recommended axis because it
  preserves the eligibility-check distinction (B2 vs. B3)
  that ADR-0049 §(a) commits. A third possibility surfaced
  in round-1 critique F9: the two axes may **compose** —
  route on session type first, then route on artifact type
  within each row. Composition would refine, e.g., the B3
  entry row to "B3 entry producing a study" vs. "B3 entry
  producing a tooling artifact" with different reading
  subsets. The current router does not commit composition;
  if a future B3-N entry reveals that within-row variance is
  material, composition is the documented re-shape path.
  *Out-of-scope for current cycle:* documented here so the
  trade-off is not re-derived from scratch.

- **OQ-4 — Wave-S session type.** Wave-S is still
  partially open (full-gate criteria pending per
  [ADR-0020](../../docs/adr/0020-wave-s-launch.md)). A
  hypothetical Wave-S follow-up session (B0-S4 / B0-S5 / …)
  would arguably need its own router row, but the partial
  gate met 2026-05-24 means Wave-S sessions that have run
  since then are effectively B3 entries against the
  partial-gate envelope. The router does not commit a
  separate Wave-S row at this time; if the full gate
  reveals new session shape, a follow-up B3-N adds it.
  *Out-of-scope for current cycle:* Wave-S full gate is
  the trigger.

- **OQ-5 — Attention-cost measurement for the router's
  cost-savings claim.** §Consequences §1 commits a
  by-line-count saving (~85% for Flow 6; ~280-line
  historical-skim floor for B3 / implementation-slice)
  without benchmarking against actual attention cost.
  Structured-prose lines and dense-code lines do not
  carry equivalent reading load. The first 2–3
  post-router sessions provide a measurement window; the
  operator notes for each session (a) which router row
  applied, (b) which playbooks were actually read in full
  vs. skimmed, and (c) whether any playbook outside the
  per-type set turned out to be load-bearing mid-session.
  If the attention-cost saving turns out to be materially
  smaller than the line-count saving, a follow-up B3-N
  entry revisits the router shape.
  *Out-of-scope for current cycle:* deferred to the first
  three post-router sessions for empirical evidence.

---

## Promotion target

`docs/adr/0052-session-reading-router.md` (provisional;
operator-side reservation confirmed at
[`/promote-to-adr`](../../.claude/commands/promote-to-adr.md)
time per
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
Clause 7).
