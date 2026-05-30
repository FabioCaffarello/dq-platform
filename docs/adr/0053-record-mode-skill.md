<!-- path: docs/adr/0053-record-mode-skill.md -->

# ADR-0053 — `record-mode-conventions` Skill for the Agent Harness

- **Status:** accepted
- **Date:** 2026-05-30

---

## Context

The `.claude/skills/` directory carries four skills today:
[`adr-writing`](../../.claude/skills/adr-writing/SKILL.md)
(MADR shape and citation conventions),
[`critique-anti-patterns`](../../.claude/skills/critique-anti-patterns/SKILL.md)
(patterns to avoid in `/critique` output),
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
(seven Go conventions C1–C7 observed across
`engine/internal/`, each traced to a real `file:line`),
and
[`session-governance`](../../.claude/skills/session-governance/SKILL.md)
(cross-cutting session-governance discipline; committed
by [ADR-0051](./0051-claude-tooling-postwave3.md) Clause 3
via B3-1). Each encodes a discoverable contract surface
that future contributors and agents can rely on.

**Record-mode conventions have no equivalent surface.**
The record-mode code shipped under Wave-S (partial gate
met 2026-05-24; record-mode runner landed at sub-slice β
per [ADR-0024](./0024-window-semantics.md)) carries a
coherent set of seven conventions:

- A **substrate-agnostic consumer boundary** — the
  `RecordConsumer` interface and the `FetchedRecord`
  struct in `engine/internal/runner/record_runner.go`;
  the franz-go-backed implementation in
  `engine/internal/runner/kafka_consumer.go` maps
  `kgo.Record` → `FetchedRecord` at the boundary.
- A **β commit posture** — `DisableAutoCommit()` on the
  franz-go client plus manual `CommitUncommittedOffsets`
  after every successful `PollFetches`; per-attempt
  re-read of offset ranges per
  [ADR-0024](./0024-window-semantics.md) is a future
  slice.
- A **translation-at-boot boundary** — the runner package
  deliberately does not import `dsl/spec`; the engine
  binary's `buildRecordRunners` calls
  `RuleSpec.ToCheckSpecs()` (defined in `dsl/spec`) and
  constructs `runner.RecordSource` values at boot; the
  duplication of the `Source` shape on both sides of the
  boundary is protected by a reflect-based struct-mirror
  test in an external test package
  (`engine/internal/runner/struct_mirror_test.go`) that
  fails CI if any field is missing from either side.
- A **`TriggerDispatcher` interface** — declared
  consumer-side in `record_runner.go` so tests can inject
  a mock without standing up the full `*Runner`; a
  compile-time assertion (`var _ TriggerDispatcher =
  (*Runner)(nil)`) pins `*Runner` to the contract.
- A **watermark-driven window-close semantics** — the
  active window closes when the watermark advances past
  `active.end + lateness_tolerance` per
  [ADR-0024](./0024-window-semantics.md); a
  `LateDroppedCount` is surfaced on the closed-window
  `TriggerRequest`.
- A **single-goroutine state machine** — per-entity state
  is accessed only inside `Start`'s consumer poll loop;
  no internal locking; a `sync`-import sentinel
  (`var _ = sync.Mutex{}`) keeps the absence of
  mutex-protected fields visible at compile time.
- A **colocated test-doubles pattern** for the
  `CheckEvaluator` interface — the interface and three
  test doubles (`NoopEvaluator`, `FixedResultEvaluator`,
  `PerCheckEvaluator`) live in
  `engine/internal/runner/check_evaluator.go` so test
  wiring is a single-import action; the production
  `*eval.Evaluator` satisfies the interface implicitly
  via duck typing per the
  [`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
  C4 boundary discipline.

These conventions live today in doc comments scattered
across five files: `record_runner.go`,
`kafka_consumer.go`, `check_evaluator.go`,
`engine/cmd/dq-engine/main.go`'s `buildRecordRunners`,
and `struct_mirror_test.go`. Every record-mode PR opened
against the current main pays the cost of re-deriving
them from those scattered comments. The discoverability
gap exists now, has accumulated since sub-slice β
shipped, and is independent of any future-milestone
positioning.

This ADR closes the gap by committing a new skill —
`record-mode-conventions` — that consolidates the seven
conventions into a single discoverable contract surface,
mirroring the
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
shape: every rule traces to a real `file:line`.

**Eligibility carry-forward (`B3-3`; clean four-condition
pass).** Per
[ADR-0049](./0049-b3-evolutionary-launch.md) §(a)'s
four-condition filter, `B3-3` (the row whose promotion
this ADR is) clears all four conditions cleanly — no
borderlines, no operator-side ratification gate applied:

- **Condition 1 (P-B3.1, expands not rewrites)** — A
  new skill in
  [`.claude/skills/`](../../.claude/skills/) is
  structurally additive: no playbook, no ADR contract, no
  existing skill is rewritten. The four existing skills
  stay intact. The seven conventions consolidated by this
  skill already exist in code comments; the skill
  consolidates them.
- **Condition 2 (P-B3.4, in-scope family — Tooling
  extensions)** — Direct precedent reuse of
  [ADR-0051](./0051-claude-tooling-postwave3.md)
  Clause 1's expansive reading admitting the `.claude/`
  agent harness under
  [ADR-0049](./0049-b3-evolutionary-launch.md)
  §Per-family scope's "adjacent tooling" clause. This
  ADR does not extend the reading; it reuses the
  precedent for an artifact that lives inside the same
  admitted surface.
- **Condition 3 (P-B3.2, envelope conformance to
  [ADR-0020](./0020-wave-s-launch.md) /
  [ADR-0021](./0021-mode-as-primitive.md) /
  [ADR-0022](./0022-kind-catalog.md) /
  [ADR-0023](./0023-sources-schema.md))** — Trivial.
  The skill describes existing record-mode code; it does
  not change the substrate, mode primitive, kind
  catalog, or sources schema. The contracts behind the
  seven conventions are committed by ADR-0021 / 0023 /
  0024 and remain authoritative.
- **Condition 4 (additive-maintenance threshold)** —
  Crossed cleanly under the precedent set by
  [ADR-0051](./0051-claude-tooling-postwave3.md)
  Clause 3, which committed the
  [`session-governance`](../../.claude/skills/session-governance/SKILL.md)
  skill via B3-1 with the same "consolidates existing
  discipline into a discoverable skill surface"
  reasoning. The load-bearing motivation is the
  discoverability gap that exists today (every record-mode
  PR pays the cost of re-deriving the conventions from
  scattered doc comments); pre-positioning the harness
  for future record-mode work (Track C and beyond) is a
  bonus, not the rationale.

The principles bearing on this decision are **P5**
(evolution must be contract-driven — the skill is
itself a contract surface and evolves under the
published SKILL.md shape), **P6** (borrow patterns, not
baggage — the shape mirrors
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
because that shape fits this project's discipline, not
because some external convention prescribes it), and
**P3** (ownership is explicit — every convention names
the file that owns it; no implicit defaults). **R4** in
[`CLAUDE.md`](../../CLAUDE.md) §3 (one topic per
session) is load-bearing for the artifact split: this
ADR commits the contract; the actual
[`.claude/skills/record-mode-conventions/`](../../.claude/skills/)
SKILL.md and reference write is a separate
implementation slice (per Clause 7 below).

A scope note on library naming, carried forward from
the originating B3-3 study without pre-committing a new
R5 reading. The skill text (Clause 2 below) names
`franz-go` and its `kgo` types package as the Kafka
client library the production code uses. The naming is
**descriptive** of what the code does, not prescriptive
of a pattern source. Per [`CLAUDE.md`](../../CLAUDE.md)
§3 R5 ("we use X is fine; we are doing Y because X does
Y is not"), R5 is satisfied because the conventions
describe the code's existing shape — the
substrate-agnostic boundary is the project's chosen
pattern, and the β-commit posture is the project's
choice — neither convention is borrowed *because*
franz-go prescribes it. Whether R5's enumerated
exemption list ("BigQuery, Kafka, GCS, … and
equivalents") extends to a third-party Go client library
is interpretive and is not pre-committed here; either
reading reaches the same outcome (the naming is allowed
because it is descriptive).

---

## Decision

### Clause 1 — Commit a new skill at `.claude/skills/record-mode-conventions/`

A new skill directory ships at
`.claude/skills/record-mode-conventions/` mirroring the
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
shape:

- `SKILL.md` with frontmatter (`name`,
  multi-line `description`), a one-paragraph framing
  block naming the packages the conventions come from
  plus a pointer to the reference file, the seven
  numbered conventions S1–S7 (each with rule paragraph,
  a `≤6`-line code snippet or doc-comment excerpt, and
  a `file:line` citation), and an anti-patterns section.
- `reference/conventions.md` carrying each convention in
  fuller form: **Rule** / **Citation** / **Snippet**
  (`≤6` lines) / **Rationale** (one paragraph).

The directory is parallel to the four existing skill
directories. No existing skill is edited, deleted, or
relocated by this ADR.

### Clause 2 — The seven conventions S1–S7

The skill commits seven conventions. Each cites the
exact `file:line` range in `engine/` as of this ADR's
date. Citations are pre-validated against the merged
main; the implementation slice that writes the actual
SKILL.md (per Clause 7 below) verifies each citation
before shipping.

**S1 — Substrate-agnostic consumer boundary.** The
`RecordConsumer` interface is duck-typed and declared
consumer-side; `FetchedRecord` is the substrate-agnostic
record shape the runner consumes; the franz-go-backed
implementation maps `kgo.Record` → `FetchedRecord` at
the consumer boundary, isolating substrate-specific
types behind one file.

- Interface declaration:
  `engine/internal/runner/record_runner.go:31-47`.
- `FetchedRecord` shape:
  `engine/internal/runner/record_runner.go:49-58`.
- Franz-go implementation + mapping:
  `engine/internal/runner/kafka_consumer.go:13-25` and
  `engine/internal/runner/kafka_consumer.go:85-92`.

**S2 — β commit semantics: commit-after-fetch
(at-most-once); per-attempt re-read deferred.** The
franz-go client opts into `DisableAutoCommit()`; the
runner calls `CommitUncommittedOffsets` after every
successful `PollFetches`. The β posture commits after
the fetch, not after the dispatch — at-most-once on a
crash mid-dispatch. Per-attempt re-read of offset ranges
per [ADR-0024](./0024-window-semantics.md) is a future
slice; that slice will replace β commit with a
commit-after-dispatch flow keyed on the trigger's
successful return.

- DisableAutoCommit option:
  `engine/internal/runner/kafka_consumer.go:62-67`.
- Commit-after-fetch call + β-posture comment naming the
  future replacement:
  `engine/internal/runner/kafka_consumer.go:94-102` and
  `engine/internal/runner/kafka_consumer.go:18-22`.

**S3 — Translation-at-boot boundary: `dsl/spec.RuleSpec`
→ `runner.RecordSource` happens in the engine binary,
not in the runner package; guarded by a reflect-based
struct-mirror test in an external test package.** The
runner package deliberately does not import `dsl/spec`
— the engine binary's `buildRecordRunners` reads the
spec shape, calls `RuleSpec.ToCheckSpecs()` to produce
the `[]runner.CheckSpec` the runner consumes, and
constructs `runner.RecordSource` values at boot. The
duplication of the `Source` shape on both sides of the
boundary is protected by a reflection sweep that fails
CI if any field on one side is missing from the other.
The test lives in the external `runner_test` package on
purpose: an internal-test-package import of `dsl/spec`
would close an import cycle.

- Translation comment on `RecordSource`:
  `engine/internal/runner/record_runner.go:18-21`.
- Translation method on `RuleSpec`:
  `engine/internal/dsl/spec/spec.go:114-127`.
- Translation call site in the engine binary:
  `engine/cmd/dq-engine/main.go:624-703`.
- Struct-mirror test rationale and assertion:
  `engine/internal/runner/struct_mirror_test.go:1-60`.

**S4 — `TriggerDispatcher` interface: minimal dispatcher
contract declared consumer-side so tests can inject a
mock; compile-time assertion that `*Runner` satisfies
it.** The `RecordRunner` needs only
`Run(ctx, TriggerRequest) → (*ExecutionRow, error)` from
the inner runner. The interface is declared in
`record_runner.go` (consumer-side, duck-typed). A
compile-time assertion in the same file pins `*Runner`
to the contract: a future change to `Runner.Run`'s
signature breaks the assertion at compile time.

- Interface declaration:
  `engine/internal/runner/record_runner.go:92-97`.
- Compile-time assertion:
  `engine/internal/runner/record_runner.go:381`.

**S5 — Watermark-driven window-close semantics: close
fires when watermark > `active.end + lateness_tolerance`
per [ADR-0024](./0024-window-semantics.md);
`LateDroppedCount` surfaced on the trigger.** The
watermark advances monotonically as the max of record
timestamps seen so far. The active window closes when
the watermark crosses
`active.end + lateness_tolerance`; on close, a
`TriggerRequest` carries the accumulated records, the
window bounds, and the late-drop count. Strictly-later-
window records eagerly close the active window at
sub-slice β; a per-window parallel buffer is a follow-up
enhancement.

- Window-state machine in `handleFetched`:
  `engine/internal/runner/record_runner.go:230-316`.
- Close-and-dispatch path in `closeAndDispatch`:
  `engine/internal/runner/record_runner.go:318-359`.

**S6 — Single-goroutine state machine: per-entity state
accessed only inside `Start`'s poll loop; no internal
locking.** The runner doc comment names the invariant
("The runner is single-goroutine by construction (Start
runs one consumer poll loop); no internal locking is
required."). The per-entity state map is initialized in
`NewRecordRunner` and mutated only inside
`handleFetched` and `closeAndDispatch`, both called from
`Start`. A `sync`-import sentinel
(`var _ = sync.Mutex{}`) keeps the import explicit so a
future PR that adds mutex-protected fields surfaces the
addition in the diff.

- Invariant declaration:
  `engine/internal/runner/record_runner.go:110-115`.
- Single poll loop in `Start`:
  `engine/internal/runner/record_runner.go:193-227`.
- `sync`-import sentinel documenting the deliberate
  absence:
  `engine/internal/runner/record_runner.go:399-403`.

**S7 — `CheckEvaluator` boundary + colocated test
doubles: interface declared in `check_evaluator.go`
consumer-side; three test doubles (Noop / Fixed /
PerCheck) live in the same file so test wiring is a
single-import action; production evaluator lives in
`eval/` via duck typing.** The interface and its test
doubles live together because test wiring is a frequent
operation and an extra import adds cost without adding
value. The production `*eval.Evaluator` satisfies the
interface implicitly per the
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
C4 boundary discipline.

- Interface + colocated test doubles:
  `engine/internal/runner/check_evaluator.go:20-64`.
- Production satisfier (eval package doc.go):
  `engine/internal/eval/doc.go:9-13`.

### Clause 3 — Anti-patterns the record-mode code consistently avoids

The skill's anti-patterns section names patterns absent
from record-mode code today. The implementation slice
that writes the actual SKILL.md (Clause 7) validates
each absence (grep for the forbidden idiom in `engine/`
and confirm zero hits):

- **Substrate-specific types leaking out of
  `kafka_consumer.go`.** `kgo.*` does not appear outside
  that file; `FetchedRecord` is the only shape the rest
  of the runner sees.
- **Internal locking inside `RecordRunner`.** No
  `sync.Mutex` field on `RecordRunner` per the S6
  invariant; the imported `sync` package is referenced
  only by the explicit-import sentinel.
- **`dsl/spec` imported by the runner package.**
  Translation lives in the engine binary; the
  struct-mirror test (S3) guards the absence at CI time.
- **WHAT-style comments.** Same as the
  [`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
  anti-patterns — record-mode comments explain WHY,
  typically citing
  [ADR-0024](./0024-window-semantics.md) or the
  future-slice replacement of β commit.

### Clause 4 — SKILL.md frontmatter description and trigger phrases

The skill's frontmatter `description` covers the record-
mode surface end-to-end so the description-match
mechanism loads the skill when a contributor or agent
reaches a record-mode-relevant moment. The implementation
slice (Clause 7) writes the description against a
candidate trigger-phrase list:

- "record runner", "record mode", "RecordRunner",
  "RecordSource";
- "kafka consumer", "FetchedRecord", "consumer group";
- "TriggerDispatcher", "window close", "watermark",
  "lateness tolerance";
- "β commit", "commit-after-fetch", "DisableAutoCommit",
  "CommitUncommittedOffsets";
- "boot translation", "ToCheckSpecs", "struct mirror".

The final phrase list lands in the implementation slice
after harness-side verification that the description
reliably loads the skill on each phrase — the same risk
[ADR-0051](./0051-claude-tooling-postwave3.md) Clause 3
flagged for the
[`session-governance`](../../.claude/skills/session-governance/SKILL.md)
skill's verbatim-phrase triggers. If a phrase fails to
load reliably, the description is tightened in the same
slice; this ADR does not pre-commit a description that
may not load.

### Clause 5 — Reference file shape

`reference/conventions.md` repeats each convention in
fuller form, mirroring
[`go-coding-standards/reference/conventions.md`](../../.claude/skills/go-coding-standards/reference/conventions.md):

- Title and short framing paragraph.
- Per-convention block: **Rule** (one paragraph) /
  **Citation** (`file:line` ranges) / **Snippet**
  (≤6-line Go or doc-comment excerpt) / **Rationale**
  (one paragraph).
- Anti-patterns section repeating the absences from
  Clause 3 in fuller form.

The reference file is a discoverable secondary surface a
reviewer can navigate when the SKILL.md's brief shape is
insufficient for the moment.

### Clause 6 — Citation discipline and drift mitigation

Every rule in S1–S7 cites a real `file:line` range in
`engine/`. The citations are pre-validated against the
merged main as of this ADR's date. The implementation
slice (Clause 7) re-verifies each citation immediately
before shipping the SKILL.md, against whatever main
carries when the slice opens its PR.

Citation drift — a future PR that edits a cited file may
invalidate a citation pinned to `file:line` — is a known
risk this skill inherits from the
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
skill. Two mitigations are committed by this ADR:

- A PR that edits a cited file is responsible for
  updating the skill's citations in the same PR (R4-
  bundled). This is the default discipline.
- A future tooling extension may ship a
  `make lint-skill-citations` target (a grep-style sweep
  that confirms each cited `file:line` lands at a
  non-empty line in the cited file). Whether the tooling
  is worth the harness work is deferred to OQ-1 in
  Notes; it lands only after a concrete drift incident
  or operator signal.

### Clause 7 — Artifact split: this ADR commits the contract; SKILL.md write is a deferred implementation slice

This ADR commits the contract: seven conventions S1–S7
with pre-validated citations, the SKILL.md shape
(Clause 4), the reference file shape (Clause 5), the
anti-patterns set (Clause 3), and the citation
discipline (Clause 6). **The actual SKILL.md and
`reference/conventions.md` write is a follow-on
implementation slice on a separate operator-authorized
session** per
[`CLAUDE.md`](../../CLAUDE.md) §3 R4 (one topic per
session).

The implementation-slice session is an **Implementation
slice landing under a closed B-row** per
[ADR-0052](./0052-session-reading-router.md) §6.2 row 6
— the closed B-row is `B3-3`. The implementation slice
reads (beyond the always-on floor of
[ADR-0052](./0052-session-reading-router.md) Clause 1):
`post-wave3-session-loop.md` for the close-discipline;
`wave-3-acceptance-criteria.md` for AC-W3-3 (citation
discipline — every S1–S7 cites a real `file:line` that
exists in the merged main at the time the slice ships)
and AC-W3-7 (local build / lint / test gates — `make
lint` for the markdown surface; no engine / rules /
tools / deploy gates apply because no production-code
file is touched, so AC-W3-7 is vacuously satisfied for
the production-code surface); and `feedback-protocol.md`.

The artifact split mirrors the
[ADR-0051](./0051-claude-tooling-postwave3.md) shape:
the ADR committed the four-layer enforcement stack
(session-governance skill, post-wave3-session-loop
playbook, /open-pr command, settings hardening); the
six-artifact implementation slice landed on a separate
session and PR.

---

## Consequences

1. **A fifth skill ships in
   [`.claude/skills/`](../../.claude/skills/) —
   `record-mode-conventions/`.** The directory carries
   `SKILL.md` (seven conventions S1–S7 + anti-patterns +
   frontmatter description per Clause 4) and
   `reference/conventions.md` (per-convention fuller
   form per Clause 5). The implementation slice writes
   both files; this ADR pre-validates every citation.

2. **Record-mode PRs gain a discoverable convention
   surface.** A contributor opening a PR that touches
   `engine/internal/runner/record_runner.go`,
   `engine/internal/runner/kafka_consumer.go`,
   `engine/internal/runner/check_evaluator.go`, or the
   engine binary's record-mode boot translation loads
   the skill on description-match. The seven conventions
   become a reviewer surface, not a scavenger hunt
   through doc comments.

3. **No platform code changes.** This ADR is a pure
   harness-layer extension; the platform's runtime is
   unaffected. No engine, rules, tools, or deploy file
   is touched by this ADR. The implementation slice is
   likewise documents-only (markdown under
   `.claude/skills/record-mode-conventions/`).

4. **The
   [ADR-0051](./0051-claude-tooling-postwave3.md)
   Clause 3 precedent for new skills via B3-N entries
   is reused, not extended.** Same shape: B3-N study
   proposes the skill; the ADR promotion commits the
   contract; the implementation slice writes the files
   on a separate session. No new precedent opened.

5. **Future record-mode amendments amend the skill in
   place.** When β commit semantics are replaced by
   per-attempt re-read (the future slice named in S2's
   citation), the skill's S2 convention amends in place
   per [ADR-0050](./0050-v1-retirement-engine-release.md)
   §Consequence 4's Amendment-log convention. If a
   change reshapes the contract (e.g., the runner
   package starts importing `dsl/spec`, reversing S3),
   the skill is superseded by a fresh ADR per the
   [ADR-0017](./0017-substrate-posture-amendment.md)
   pattern. The choice is made inside the amending
   session per the
   [ADR-0052](./0052-session-reading-router.md) §6.2 ADR
   amendment row's discipline.

6. **Citation drift is a known risk with a documented
   mitigation.** Per Clause 6, a PR that edits a cited
   file updates the skill's citations in the same PR
   (R4-bundled). A future
   `make lint-skill-citations` target is deferred to
   OQ-1 in Notes; it ships only after a concrete drift
   incident or operator signal.

7. **The skill's description-match loading is verified
   by the implementation slice, not pre-committed.**
   Per Clause 4 — the description and the trigger phrase
   list land in the implementation slice after harness-
   side verification. This mirrors the
   [ADR-0051](./0051-claude-tooling-postwave3.md)
   Clause 3 discipline for the
   [`session-governance`](../../.claude/skills/session-governance/SKILL.md)
   skill's verbatim phrases.

8. **CI gates for the implementation slice are well-
   defined per
   [ADR-0052](./0052-session-reading-router.md) §6.2
   row 6.** AC-W3-3 (citation discipline — every S1–S7
   cites a real `file:line` at slice-ship time) is the
   load-bearing gate; AC-W3-7 (local build / lint /
   test) is vacuously satisfied for the production-code
   surface because the slice touches no production-code
   file. The slice's PR body declares the vacuous gate
   per the
   [ADR-0051](./0051-claude-tooling-postwave3.md)
   Clause 5 `/open-pr` discipline.

9. **The agent-harness-as-adjacent-tooling reading from
   [ADR-0051](./0051-claude-tooling-postwave3.md)
   Clause 1 is reused, not extended.** This ADR sits
   inside the surface that ADR-0051 admitted under
   [ADR-0049](./0049-b3-evolutionary-launch.md)
   §Per-family scope's "adjacent tooling" clause; it does
   not open a new expansive reading. Future B3-N entries
   that touch agent-harness territory continue to cite
   [ADR-0051](./0051-claude-tooling-postwave3.md)
   Clause 1 as the precedent; entries that touch
   adjacent-tooling surfaces not yet admitted (reviewer
   tooling, observability harness) follow the same
   new-contribution-requiring-review discipline rather
   than absorb a reading silently.

10. **No D0 ratification mechanism applies to this ADR.**
    Unlike `B3-1` and `B3-2`, `B3-3` cleared all four
    eligibility conditions without borderlines —
    Condition 2 by direct precedent reuse, Condition 4
    by direct precedent reuse, Conditions 1 and 3
    cleanly. The author-equals-reviewer circularity
    recognized by
    [ADR-0051](./0051-claude-tooling-postwave3.md)
    §Consequence 7 (relevant only to borderline
    eligibility readings) is not triggered here. This
    ADR is therefore a load-bearing precedent for *clean*
    B3-N entries — those that ride existing precedents
    without opening new readings. Future B3-N sessions
    can cite ADR-0053 as the example of a B3-N entry
    that does not need an operator-ratification gate.

---

## Notes

Four open questions remain explicitly out-of-scope for
this ADR; each has a named trigger condition.

- **OQ-1 — Citation-drift verification mechanism.** Per
  Clause 6, a future tooling extension may ship a
  `make lint-skill-citations` target that confirms each
  cited `file:line` in any skill's reference resolves
  to a non-empty line in the cited file at CI time.
  Whether this is worth the harness work depends on
  drift incidence. Resolved by the first citation-drift
  incident (a record-mode PR ships with stale skill
  citations) or by operator signal that the manual
  R4-bundled discipline is not catching drift in
  practice.

- **OQ-2 — Track C scope and its relationship to the
  skill's content.** Pre-positioning for Track C is
  named in §Consequences (item 2) as a beneficiary of
  the skill, but Track C scope is deferred to its own
  session. Track C may surface conventions that S1–S7
  do not cover (e.g., schema-conformance-check kind
  specifics, evidence-shape patterns for record-mode
  results). Those conventions ride their own B3-N
  entries; this ADR does not anticipate them.

- **OQ-3 — Skill description shape and harness load
  reliability.** Per Clause 4 — the description and
  trigger phrase list land in the implementation
  slice after harness-side verification. If any phrase
  fails to load reliably, the description is tightened
  in the same slice. The same risk
  [ADR-0051](./0051-claude-tooling-postwave3.md)
  Clause 3 carried for the
  [`session-governance`](../../.claude/skills/session-governance/SKILL.md)
  skill is inherited here.

- **OQ-4 — Whether `engine/internal/runner/doc.go`
  should ship alongside the skill.** The runner
  package currently does not have a `doc.go`. A short
  `doc.go` naming the runner package's owned surface
  and not-imports (per the
  [`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
  C4 boundary discipline) may be a useful complement
  to the skill. The likely lane for the addition is an
  Implementation slice per
  [ADR-0052](./0052-session-reading-router.md) §6.2
  row 6 landing under the closed Wave-3 P4 runner
  scaffolding; if the `doc.go` work surfaces new
  harness conventions instead of documenting existing
  ones, the lane is a follow-up B3-N entry. The choice
  is made when OQ-4 is taken up.

A practical note on the clean four-condition pass
recorded in §Context: this ADR is a precedent for B3-N
entries that ride existing precedents (in this case
[ADR-0051](./0051-claude-tooling-postwave3.md)
Clauses 1 and 3) without opening new expansive
readings. Future B3-N sessions whose eligibility check
clears all four conditions on existing precedents — and
that recognize no borderline interpretation — cite
ADR-0053 as the example of a B3-N entry that promotes
without an operator-ratification gate. Borderline B3-N
entries continue to follow the
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5
§"Operator-side responsibilities" ratification
mechanism committed by
[ADR-0051](./0051-claude-tooling-postwave3.md)
§Consequence 7 and propagated by
[ADR-0052](./0052-session-reading-router.md) §Context.
