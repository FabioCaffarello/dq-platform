<!-- path: studies/decisions/2026-05-30-b3-record-mode-skill.md -->

# B3-3 — `record-mode-conventions` skill for the agent harness

## Metadata

- **Wave reference:** B3 (evolutionary lane; tooling extensions
  family per
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
  §Per-family scope; agent-harness sits inside the
  "adjacent tooling" reading admitted by
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clause 1 — this study reuses the precedent, it does not
  open a new expansive reading; ADR-0052 confirmed the
  precedent remains active for harness extensions).
- **Status:** resolved-study (B3-3; closed 2026-05-30 at the
  merge of PR #104, merge commit `e6efd99`; two-round
  critique cap reached per
  [`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  step 7; no D0 ratification needed — eligibility under
  ADR-0049 §(a) passed all four conditions cleanly without
  borderlines per the Metadata block's eligibility check).
- **Last updated:** 2026-05-30.
- **Upstream resolved:**
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) (B3
  eligibility filter and family list);
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clauses 1, 3 (the "adjacent tooling" precedent admitting
  the `.claude/` harness; the precedent for new skills
  added via B3-N entries — Clause 3 committed
  `session-governance` via B3-1);
  [ADR-0052](../../docs/adr/0052-session-reading-router.md)
  (the router that classifies B3-N study sessions and
  identifies this work as a B3-entry session);
  [ADR-0024](../../docs/adr/0024-window-semantics.md)
  (record-mode window-close semantics — the contract behind
  S5 below);
  [ADR-0021](../../docs/adr/0021-mode-as-primitive.md) (mode
  as primitive — the contract behind the record-mode
  package boundary).
- **Eligibility check (ADR-0049 §(a)):**
  - **Condition 1 — P-B3.1, expands not rewrites.** A new
    skill in
    [`.claude/skills/`](../../.claude/skills/) is
    structurally additive: no playbook, no ADR, no
    existing skill is rewritten; the conventions already
    live in code comments and the skill consolidates them
    under a discoverable surface. The three existing
    skills
    ([`adr-writing`](../../.claude/skills/adr-writing/SKILL.md),
    [`critique-anti-patterns`](../../.claude/skills/critique-anti-patterns/SKILL.md),
    [`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md))
    and the
    [`session-governance`](../../.claude/skills/session-governance/SKILL.md)
    skill committed by
    [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
    Clause 3 stay intact. ✅
  - **Condition 2 — P-B3.4, in-scope family — Tooling
    extensions.** Direct precedent reuse of
    [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
    Clause 1's expansive reading admitting the `.claude/`
    agent harness under
    [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
    §Per-family scope's "adjacent tooling" clause. The
    new skill lives inside that admitted surface — same
    family as the existing three skills and the
    `session-governance` skill. No new reading opened. ✅
  - **Condition 3 — P-B3.2, conforms to
    ADR-0020 / 0021 / 0022 / 0023.** The skill describes
    *existing* record-mode code; it does not change the
    substrate, mode primitive, kind catalog, or sources
    schema. The contracts behind S1–S7 below are committed
    by ADR-0021 / 0023 / 0024 and remain authoritative;
    the skill cites them, it does not amend them. ✅
  - **Condition 4 — additive-maintenance threshold.** The
    skill becomes a **discoverable contract surface** that
    future PRs to record-mode code must respect. Today the
    seven conventions live in code comments scattered
    across
    [`engine/internal/runner/record_runner.go`](../../engine/internal/runner/record_runner.go),
    [`engine/internal/runner/kafka_consumer.go`](../../engine/internal/runner/kafka_consumer.go),
    [`engine/internal/runner/check_evaluator.go`](../../engine/internal/runner/check_evaluator.go),
    [`engine/cmd/dq-engine/main.go`](../../engine/cmd/dq-engine/main.go),
    and
    [`engine/internal/runner/struct_mirror_test.go`](../../engine/internal/runner/struct_mirror_test.go).
    A contributor opening a record-mode PR today has no
    single surface enumerating them. Pre-positioning the
    harness for **Track C** (the operator-stated next
    milestone, deferred for separate scoping) is the
    concrete forcing function: a Track C PR will touch
    record-mode code and needs the conventions to be
    discoverable. Same shape as
    [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
    Clause 3, which committed the
    `session-governance` skill via B3-1 with the same
    "consolidates existing discipline into a discoverable
    skill surface" reasoning. ✅
- **Constraint envelope:**
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §(a)
  (eligibility), §(b) (out-of-scope), §Per-family scope
  (Tooling extensions);
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clauses 1 (adjacent-tooling precedent), 3 (new-skill
  precedent — `session-governance`), 8 (B3-N flat
  labeling);
  [ADR-0052](../../docs/adr/0052-session-reading-router.md)
  Clause 2 (the B3-entry row of the session reading
  router — this session reads `post-wave3-session-loop.md`
  + `acceptance-criteria.md` + `feedback-protocol.md` +
  ADR-0049 §(a));
  [`CLAUDE.md`](../../CLAUDE.md) §3 R1–R8 (especially R4
  one-topic, R5 own-the-pattern, R6 path header, R8
  studies-not-the-product);
  [`CLAUDE.md`](../../CLAUDE.md) §4 P1–P6 (especially P5
  contract-driven evolution; the skill is itself a
  contract surface).
- **Locked premises** (operator-declared, not litigated here):
  - **P-B3RM.1** — The conventions to capture are the ones
    that **already exist** in the record-mode code. This
    study does not propose new conventions; it documents
    extant ones. New conventions (e.g., a future
    per-attempt re-read semantics replacing β commit) ride
    their own ADRs and skill updates.
  - **P-B3RM.2** — The skill must trace **every** rule to
    a real `file:line` citation, exactly like
    [`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
    does (the user-stated contract). A rule without a
    citation is not a rule; it is a guess.
  - **P-B3RM.3** — This study is **the contract proposal**;
    the actual `SKILL.md` + `reference/` write is the
    follow-on implementation slice on a separate session
    per R4. Same shape as
    [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
    (study + ADR + deferred implementation slice). The
    citations listed below are pre-validated so the
    implementation slice is mechanical.
  - **P-B3RM.4** — No platform code changes. R1 is
    historical now (Wave 3 closed), but R4 is load-bearing:
    this study touches only `studies/decisions/`,
    `studies/critiques/`, and the decision-log row. The
    actual skill files (`.claude/skills/record-mode-conventions/SKILL.md`
    + `reference/conventions.md`) land in the follow-on
    implementation slice.
  - **P-B3RM.5** — The **load-bearing motivation is
    today's discoverability gap** for record-mode PRs.
    The seven conventions live in scattered doc comments
    across five files; every record-mode PR opened today
    pays the cost of re-deriving them. Consolidating the
    conventions under a discoverable skill surface
    delivers the benefit immediately, independent of
    future scope. **Pre-positioning for Track C is a
    bonus**, not the load-bearing rationale: Track C is
    the operator-stated next milestone (scope and
    definition deferred to its own session); the skill
    happens to be useful when Track C lands, but the
    discoverability cost has already accumulated and the
    skill answers it now. New conventions Track C may
    surface ride their own B3-N entries; this study does
    not anticipate them.
- **Downstream open:** none enumerated. If `/critique`
  surfaces a blocking finding that requires an eighth
  convention or removing one of S1–S7, it is registered
  in §Open Questions and the study re-scopes — it does
  not silently grow.
- **Critique rounds:**
  round 1 preserved
  ([`studies/critiques/2026-05-30-b3-record-mode-skill-critique-1.md`](../critiques/2026-05-30-b3-record-mode-skill-critique-1.md)) —
  0 blocking / 3 important / 4 minor; all dispositioned in
  the Operator Response trailer;
  round 2 preserved
  ([`studies/critiques/2026-05-30-b3-record-mode-skill-critique-2.md`](../critiques/2026-05-30-b3-record-mode-skill-critique-2.md)) —
  0 blocking / 1 important / 3 minor; the important
  finding (R5 scope-note framing) applied in this
  revision; three minor findings accepted-as-is per
  [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)
  §"Skip" grammar.
- **Promotion target:**
  `docs/adr/0053-record-mode-skill.md` — provisionally
  the next available number at the time of writing (last
  landed is
  [ADR-0052](../../docs/adr/0052-session-reading-router.md),
  2026-05-30; reservation is operator-side per
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clause 7, confirmed at `/promote-to-adr` time).

---

## Context

The `.claude/skills/` directory carries three skills today
([`adr-writing`](../../.claude/skills/adr-writing/SKILL.md),
[`critique-anti-patterns`](../../.claude/skills/critique-anti-patterns/SKILL.md),
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md))
plus the
[`session-governance`](../../.claude/skills/session-governance/SKILL.md)
skill committed by
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
Clause 3 via B3-1. Each encodes a discoverable contract
surface a contributor can rely on:

- `adr-writing` — MADR shape, citation conventions, R8
  forward-only discipline.
- `critique-anti-patterns` — patterns to avoid in
  `/critique` output.
- `go-coding-standards` — seven Go conventions (C1–C7)
  observed across `engine/internal/`, each traced to a
  real `file:line`.
- `session-governance` — cross-cutting session-governance
  discipline (G1–G6).

**Record-mode conventions have no equivalent surface.**
The record-mode code shipped under Wave-S (the partial
gate met 2026-05-24, with the record-mode runner landing
at sub-slice β per
[ADR-0024](../../docs/adr/0024-window-semantics.md))
carries a coherent set of conventions:

- a **substrate-agnostic consumer boundary** (the
  `RecordConsumer` interface and the `FetchedRecord`
  struct);
- a **β commit posture** (disable auto-commit + manual
  commit-after-fetch; per-attempt re-read deferred to a
  future slice);
- a **translation-at-boot boundary** (the runner package
  does not depend on `dsl/spec`; the engine binary
  translates `spec.RuleSpec` → `runner.RecordSource` at
  boot, with a reflect-based struct-mirror test guarding
  the duplication);
- a **TriggerDispatcher abstraction** (the inner runner
  presented to the record runner as a minimal interface
  so tests can inject a mock without standing up the
  full `*Runner`);
- a **watermark-driven window-close semantics** (close
  fires when the watermark advances past
  `active.end + lateness_tolerance` per ADR-0024);
- a **single-goroutine state machine** (the record
  runner's per-entity state is accessed only from inside
  `Start`'s poll loop; no internal locking);
- a **colocated test-doubles pattern** for the
  `CheckEvaluator` interface (three test doubles —
  `NoopEvaluator`, `FixedResultEvaluator`,
  `PerCheckEvaluator` — live in the same file as the
  interface, so test wiring is a single-import action).

Today these conventions live in:

- Doc comments inside
  [`engine/internal/runner/record_runner.go`](../../engine/internal/runner/record_runner.go)
  (the `RecordConsumer` interface comment naming the
  franz-go disabled-auto-commit pattern; the
  `RecordRunner` doc comment naming the single-goroutine
  invariant; the `TriggerDispatcher` interface comment
  naming the test-mock motivation).
- Doc comments inside
  [`engine/internal/runner/kafka_consumer.go`](../../engine/internal/runner/kafka_consumer.go)
  (the β-commit posture comment naming the per-attempt
  re-read deferral; the `DisableAutoCommit` opt and the
  manual `CommitUncommittedOffsets` call).
- Doc comments inside
  [`engine/internal/runner/check_evaluator.go`](../../engine/internal/runner/check_evaluator.go)
  (the no-op-default rationale and the test-double
  block).
- The boot-time translation in
  [`engine/cmd/dq-engine/main.go`](../../engine/cmd/dq-engine/main.go)
  `buildRecordRunners`.
- The struct-mirror invariant test in
  [`engine/internal/runner/struct_mirror_test.go`](../../engine/internal/runner/struct_mirror_test.go).

A contributor opening a record-mode PR today must
discover these conventions by reading code. The skill
consolidates them into one discoverable surface,
trace-to-`file:line` per the
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
shape, so the next contributor — and the next agent — has
a contract to read against.

**Why now: the discoverability gap exists today.** Every
record-mode PR opened against the current main pays the
cost of re-deriving the seven conventions from doc
comments scattered across five files
(`record_runner.go`, `kafka_consumer.go`,
`check_evaluator.go`, `engine/cmd/dq-engine/main.go`'s
`buildRecordRunners`, and `struct_mirror_test.go`). The
cost has already accumulated since sub-slice β shipped;
consolidating the conventions under a discoverable skill
surface delivers the benefit immediately, independent of
any future milestone. The
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
Clause 3 precedent (the
[`session-governance`](../../.claude/skills/session-governance/SKILL.md)
skill, committed via B3-1) committed a skill that
consolidates existing discipline into a discoverable
contract surface; this study reuses that shape.

**Pre-positioning for Track C is a bonus.** Track C is
the operator-stated next milestone, and the record-mode
surface is where it will land — so a Track C PR will
benefit from the skill on day one. But Track C's scope
and definition are deferred, and this study does not
hang its eligibility on Track C's needs: the immediate
discoverability benefit stands on its own. If Track C
were postponed indefinitely, the skill would still pay
for itself.

The principles bearing on this decision are **P5**
(evolution must be contract-driven — the skill is itself
a contract surface and evolves under the published
SKILL.md shape) and **P6** (borrow patterns, not baggage —
the shape mirrors
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
because that shape fits this project's discipline, not
because some external convention prescribes it).
[`CLAUDE.md`](../../CLAUDE.md) R4 (one topic per session)
is load-bearing for the artifact split — this study
proposes the skill as a contract; the actual SKILL.md
write is a separate session per P-B3RM.3.

---

## Decision Drivers

### D1 — Every rule traces to a real `file:line`

P-B3RM.2 is non-negotiable per the user-stated contract:
the skill must trace every rule to a real `file:line`,
exactly like
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
does. A rule without a citation is a guess. The
implementation slice (the SKILL.md + reference write) must
**verify each citation lands at the right line before
shipping**. The validation step lives in the
implementation-slice acceptance criteria (AC-W3-3
citation discipline applies — see ADR-0052 §6.2's
Implementation slice row).

### D2 — Capture extant conventions only; do not propose new ones

P-B3RM.1 fixes the scope: the seven conventions S1–S7 below
exist in the record-mode code today. The study captures
them; it does not propose new conventions. If `/critique`
surfaces a "this looks like a missing convention" comment,
the right response is either (a) find the citation in
existing code and add it as an eighth convention, or (b)
defer it as a follow-up B3-N entry — never absorb a new
convention into this study silently.

### D3 — Mirror the `go-coding-standards` shape

The skill's shape must mirror
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md):

- `SKILL.md` with frontmatter (`name` + rich
  multi-line `description`).
- One-paragraph framing naming the packages the conventions
  come from + a pointer to the reference file.
- Numbered conventions (S1 … Sn), each with a rule
  paragraph, a `≤6`-line code snippet (or doc-comment
  block), and a `file:line` citation.
- An anti-patterns section listing what record-mode code
  consistently **does not** do.
- A separate `reference/` markdown file repeating each
  convention in fuller form: Rule / Citation / Snippet /
  Rationale.

This shape is the precedent the skill follows; deviating
without rationale is a P6 violation (borrow-patterns
discipline: this project's shape, not invent a new one).

### D4 — Seven conventions is the right number; not eight, not five

The seven conventions S1–S7 below cover the record-mode
surface end-to-end: the consumer boundary (S1), the commit
posture (S2), the boot-time translation (S3), the
dispatcher interface (S4), the window-close semantics
(S5), the goroutine discipline (S6), and the evaluator
boundary (S7). Adding an eighth that does not have a
present citation would violate D2; collapsing two
unrelated conventions to reach five would violate the
trace-to-citation contract by forcing one citation to
stand for two rules. The number is the number the code
carries.

### D5 — The study is the contract; the SKILL.md write is the implementation slice

P-B3RM.3 fixes the artifact split. This study commits the
contract: seven conventions, file:line pre-validated, the
description-frontmatter shape, the reference structure.
The implementation slice writes the actual SKILL.md and
the reference file, runs the local AC-W3 gates, and lands
via its own PR. Bundling implementation into this study
PR would violate R4 (the study is the contract proposal;
the write is the implementation).

---

## Considered Options

### Option A — New `record-mode-conventions` skill mirroring `go-coding-standards`

A new directory at
`.claude/skills/record-mode-conventions/` carrying
`SKILL.md` (frontmatter + seven conventions S1–S7 + anti-
patterns) and `reference/conventions.md` (the seven
conventions in fuller form). The skill's description
triggers on the contributor phrases an agent encounters
when touching record-mode code: "record runner", "record
mode", "kafka consumer", "FetchedRecord", "RecordSource",
"TriggerDispatcher", "window close", "watermark",
"consumer group", "β commit".

**Trade-offs.** Explicit, discoverable, parallel to the
existing four skills. Adds one new skill directory to
maintain. Mirrors the
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
shape exactly per D3. Pre-positions for Track C per
P-B3RM.5. The implementation slice is mechanical because
this study pre-validates every citation.

### Option B — Extend `go-coding-standards` in place with record-mode conventions

Append the seven conventions to the existing
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
SKILL.md as C8–C14, and extend the reference file
similarly.

**Trade-offs.** Avoids a new skill directory. **But**: the
existing skill's description is scoped to "writing or
reviewing Go code under engine/, tools/, or any Go module
in this repo" — broad. Record-mode conventions are
narrower, scoped to the record-mode surface (the runner
package + the engine binary's record-mode boot
translation). Mixing them inflates the existing skill's
description and risks the skill triggering on every Go PR
even when record-mode is not in scope. The narrower
scoping is itself a discoverability win (the skill loads
when it is relevant; not when it is not). The existing
four skills are each narrowly scoped on purpose; this
option breaks that pattern.

### Option C — Defer until Track C lands and the conventions surface during implementation

Don't write the skill now; wait until a Track C PR
surfaces a concrete need.

**Trade-offs.** Delays the cost; no harness work needed
today. **But**: the conventions already exist in code, so
the discoverability cost is already paid by every
record-mode PR until the skill ships. Pre-positioning is
P-B3RM.5's load-bearing rationale; deferring loses that
benefit and forces the first Track C PR to re-derive the
conventions or absorb them silently. ADR-0051 Clause 3's
precedent set the pattern for committing skills *before*
the corresponding work lands; this option diverges from
that precedent without a strong reason.

### Option D — Inline conventions as expanded `doc.go` comments only

Write a `engine/internal/runner/doc.go` that documents the
seven conventions in full, and stop. Don't add a skill.

**Trade-offs.** Stays inside the engine code; no harness
work. **But**: `doc.go` documents what the package
imports and what it does not (per `go-coding-standards`
C4); a long doc.go with seven detailed convention
descriptions strains that idiom and is not a
discoverable harness surface — an agent looking up a
convention has to browse the package, not load a skill.
The discoverability benefit of a skill surface is lost.

---

## Recommendation

**Option A — new `record-mode-conventions` skill mirroring
`go-coding-standards`.**

It is the only option that meets all of D1–D5 cleanly.
Option B inflates the existing skill's scope and breaks
the per-skill narrow-description pattern the existing
four skills follow. Option C defers a benefit that is
already available and diverges from
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
Clause 3's precedent. Option D loses the discoverability
benefit and strains the `doc.go` idiom.

### The seven conventions

The seven conventions the implementation slice will
encode in the new skill, with their pre-validated
citations. Each cites the exact `file:line` range in the
merged main as of this study.

**Scope note on library naming.** Two conventions below
(S1, S2) name `franz-go` and its `kgo` types package as
the Kafka client library the production code uses. The
naming is **descriptive** of what the code does, not
prescriptive of a pattern source. Per
[`CLAUDE.md`](../../CLAUDE.md) §3 R5 ("we use X" is
fine; "we are doing Y because X does Y" is not), R5 is
satisfied because S1 / S2 describe the code's existing
shape — the substrate-agnostic boundary in S1 is the
project's chosen pattern, and S2 documents the β-commit
posture the project chose for the franz-go client wiring
— neither convention is borrowed *because franz-go
prescribes it*. Whether R5's enumerated exemption list
("BigQuery, Kafka, GCS, … and equivalents") extends to a
third-party Go client library like franz-go is
interpretive and is not pre-committed here; either
reading reaches the same outcome (the naming is allowed
because it is descriptive). The promotion session can
land a tighter or looser reading if the reviewer
prefers.

**S1 — Substrate-agnostic consumer boundary.**

`RecordConsumer` is a duck-typed interface declared
consumer-side; `FetchedRecord` is the substrate-agnostic
record shape the runner consumes; the franz-go-backed
implementation maps `*kgo.Record → FetchedRecord` at the
consumer boundary.

- Interface declaration:
  [`engine/internal/runner/record_runner.go:31-47`](../../engine/internal/runner/record_runner.go).
- `FetchedRecord` shape:
  [`engine/internal/runner/record_runner.go:49-58`](../../engine/internal/runner/record_runner.go).
- Franz-go implementation + mapping:
  [`engine/internal/runner/kafka_consumer.go:13-25`](../../engine/internal/runner/kafka_consumer.go),
  [`engine/internal/runner/kafka_consumer.go:85-92`](../../engine/internal/runner/kafka_consumer.go).

**S2 — β commit semantics: commit-after-fetch (at-most-once);
per-attempt re-read deferred.**

`DisableAutoCommit()` is set on the franz-go client; the
runner calls `CommitUncommittedOffsets` after every
successful `PollFetches`. The β posture commits after the
fetch, not after the dispatch — at-most-once on a crash
mid-dispatch. Per-attempt re-read of offset ranges per
[ADR-0024](../../docs/adr/0024-window-semantics.md)
is a future slice; that slice replaces β commit with a
commit-after-dispatch flow keyed on the trigger's
successful return.

- DisableAutoCommit:
  [`engine/internal/runner/kafka_consumer.go:62-67`](../../engine/internal/runner/kafka_consumer.go).
- Commit-after-fetch + β-posture comment naming the
  future replacement:
  [`engine/internal/runner/kafka_consumer.go:94-102`](../../engine/internal/runner/kafka_consumer.go),
  [`engine/internal/runner/kafka_consumer.go:18-22`](../../engine/internal/runner/kafka_consumer.go).

**S3 — Translation-at-boot boundary: `dsl/spec.RuleSpec`
→ `runner.RecordSource` happens in the engine binary,
not in the runner package; guarded by a reflect-based
struct-mirror test in an external test package.**

The runner package deliberately does not import `dsl/spec`
— the engine binary's `buildRecordRunners` reads the spec
shape, calls `RuleSpec.ToCheckSpecs()` (defined in
`dsl/spec`) to produce the `[]runner.CheckSpec` the
runner consumes, and constructs `runner.RecordSource`
values at boot. The duplication of the `Source` shape on
both sides of the boundary is protected by a reflection
sweep in `runner_test` (external test package, chosen to
dodge the import cycle that an internal test would
close) that fails CI if any field on one side is missing
from the other.

- Translation comment on `RecordSource`:
  [`engine/internal/runner/record_runner.go:18-21`](../../engine/internal/runner/record_runner.go).
- Translation method on `RuleSpec`:
  [`engine/internal/dsl/spec/spec.go:114-127`](../../engine/internal/dsl/spec/spec.go)
  (`ToCheckSpecs` produces the `runner.CheckSpec` slice
  the runner consumes via `TriggerRequest`).
- Translation call site in the engine binary:
  [`engine/cmd/dq-engine/main.go:624-703`](../../engine/cmd/dq-engine/main.go)
  (`buildRecordRunners` parses the rule body, calls
  `parsed.ToCheckSpecs()`, and passes the result through
  `RecordRunnerConfig.Sources[].Checks`).
- Struct-mirror test rationale + assertion:
  [`engine/internal/runner/struct_mirror_test.go:1-60`](../../engine/internal/runner/struct_mirror_test.go).

**S4 — `TriggerDispatcher` interface: minimal dispatcher
contract declared consumer-side so tests can inject a
mock; compile-time assertion that `*Runner` satisfies
it.**

The `RecordRunner` needs only `Run(ctx, TriggerRequest)
→ (*ExecutionRow, error)` from the inner runner. The
interface is declared in `record_runner.go` (consumer-
side, duck-typed). A compile-time assertion in the same
file pins `*Runner` to the contract: a future change to
`Runner.Run`'s signature breaks the assertion at compile
time.

- Interface declaration:
  [`engine/internal/runner/record_runner.go:92-97`](../../engine/internal/runner/record_runner.go).
- Compile-time assertion:
  [`engine/internal/runner/record_runner.go:381`](../../engine/internal/runner/record_runner.go).

**S5 — Watermark-driven window-close semantics: close
fires when watermark > `active.end + lateness_tolerance`
per ADR-0024; `LateDroppedCount` surfaced on the trigger.**

The watermark advances monotonically as the max of
record timestamps seen so far. The active window closes
when the watermark crosses
`active.end + lateness_tolerance`; on close, a
`TriggerRequest` carries the accumulated records, the
window bounds, and the late-drop count. Strictly-later-
window records eagerly close the active window at
sub-slice β; a per-window parallel buffer is a follow-up
enhancement.

- Window-state machine:
  [`engine/internal/runner/record_runner.go:230-316`](../../engine/internal/runner/record_runner.go).
- Close-and-dispatch path:
  [`engine/internal/runner/record_runner.go:318-359`](../../engine/internal/runner/record_runner.go).
- ADR backing:
  [ADR-0024](../../docs/adr/0024-window-semantics.md)
  §"window close" (cited by the runner code at
  [`engine/internal/runner/record_runner.go:209`](../../engine/internal/runner/record_runner.go),
  [`engine/internal/runner/record_runner.go:273`](../../engine/internal/runner/record_runner.go),
  [`engine/internal/runner/record_runner.go:292-293`](../../engine/internal/runner/record_runner.go),
  [`engine/internal/runner/record_runner.go:348`](../../engine/internal/runner/record_runner.go)).

**S6 — Single-goroutine state machine: per-entity state
accessed only inside `Start`'s poll loop; no internal
locking.**

The runner doc comment names the invariant ("The runner
is single-goroutine by construction (Start runs one
consumer poll loop); no internal locking is required.").
The per-entity state map is initialized in
`NewRecordRunner` and mutated only inside `handleFetched`
and `closeAndDispatch`, both called from `Start`. A
future code path that calls these from a different
goroutine breaks the invariant — the doc comment is the
load-bearing reviewer surface, and a `sync` import
sentinel makes the absence of mutex-protected fields
visible at compile time.

- Invariant declaration:
  [`engine/internal/runner/record_runner.go:110-115`](../../engine/internal/runner/record_runner.go).
- Single poll loop:
  [`engine/internal/runner/record_runner.go:193-227`](../../engine/internal/runner/record_runner.go).
- `sync`-import sentinel documenting the deliberate
  absence:
  [`engine/internal/runner/record_runner.go:399-403`](../../engine/internal/runner/record_runner.go)
  (the `var _ = sync.Mutex{}` reference keeps the
  import explicit so a future PR that adds
  mutex-protected fields does not need to re-add the
  import — surfacing the addition in the diff).

**S7 — `CheckEvaluator` boundary + colocated test
doubles: interface declared in `check_evaluator.go`
consumer-side; three test doubles (Noop / Fixed /
PerCheck) live in the same file so test wiring is a
single-import action; production evaluator lives in
`eval/` via duck typing.**

The interface and its test doubles live together because
test wiring is a frequent operation and an extra import
adds cost without adding value. The production
`*eval.Evaluator` satisfies the interface implicitly via
duck typing per the
[`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
C4 boundary discipline.

- Interface + test doubles:
  [`engine/internal/runner/check_evaluator.go:20-64`](../../engine/internal/runner/check_evaluator.go).
- Production satisfier (eval package):
  [`engine/internal/eval/doc.go:9-13`](../../engine/internal/eval/doc.go)
  ("The exported Evaluator type satisfies
  runner.CheckEvaluator via duck typing.").

### Anti-patterns the record-mode code consistently avoids

The skill's anti-patterns section will name the patterns
absent from record-mode code today. The implementation
slice validates each absence (e.g., grep for the
forbidden idiom and confirm zero hits in `engine/`):

- **Substrate-specific types leaking out of
  `kafka_consumer.go`.** `kgo.*` does not appear outside
  `kafka_consumer.go`; `FetchedRecord` is the only shape
  the rest of the runner sees.
- **Internal locking inside `RecordRunner`.** No
  `sync.Mutex` field on `RecordRunner` per the S6
  invariant; the imported `sync` package is referenced
  only by the explicit-import sentinel at
  [`engine/internal/runner/record_runner.go:403`](../../engine/internal/runner/record_runner.go).
- **`dsl/spec` imported by the runner package.**
  Translation lives in the engine binary; the struct-
  mirror test (S3) guards the absence.
- **WHAT-style comments.** Same as
  [`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
  anti-patterns — record-mode comments explain WHY,
  typically citing ADR-0024 or the future-slice
  replacement of β commit.

### Skill description (frontmatter)

The implementation slice will set the SKILL.md
description to (draft form; final wording lands in the
implementation slice):

> Use when writing, reviewing, or extending record-mode /
> stream-processing code under
> `engine/internal/runner/` (record_runner.go,
> kafka_consumer.go, check_evaluator.go) or the engine
> binary's record-mode boot translation. Encodes seven
> conventions S1–S7 that the existing record-mode code
> carries: substrate-agnostic consumer boundary
> (RecordConsumer interface + FetchedRecord shape);
> β commit semantics (commit-after-fetch; per-attempt
> re-read deferred per ADR-0024); translation-at-boot
> boundary (runner package free of dsl/spec; engine
> binary translates RuleSpec → RecordSource at boot;
> guarded by reflect-based struct-mirror test);
> TriggerDispatcher interface for testability; watermark-
> driven window-close semantics per ADR-0024; single-
> goroutine state machine; colocated CheckEvaluator test
> doubles. Apply when touching the record-mode runner,
> the franz-go consumer, the boot-time translation, or
> the inner evaluator boundary.

The trigger phrases (the contributor phrases that load
the skill on description-match) include: "record runner",
"record mode", "kafka consumer", "FetchedRecord",
"RecordSource", "TriggerDispatcher", "window close",
"watermark", "lateness tolerance", "consumer group",
"β commit", "commit-after-fetch", "boot translation".
The final phrase list lands in the implementation slice
after harness-side verification that the description
reliably loads the skill on each phrase.

### Reference file shape

`reference/conventions.md` repeats each convention in
fuller form, mirroring
[`go-coding-standards/reference/conventions.md`](../../.claude/skills/go-coding-standards/reference/conventions.md):

- Title.
- Framing paragraph.
- Per-convention block: **Rule** / **Citation** /
  **Snippet** (≤6 lines) / **Rationale** (one paragraph).
- Anti-patterns section repeating the absences.

---

## Consequences

1. **A fifth skill ships in `.claude/skills/`,
   `record-mode-conventions/`, mirroring the
   `go-coding-standards` shape.** The directory carries
   `SKILL.md` (seven conventions S1–S7 + anti-patterns)
   and `reference/conventions.md` (fuller per-convention
   form). The implementation slice writes both files; this
   study pre-validates every citation.

2. **Record-mode PRs gain a discoverable convention
   surface.** A contributor opening a PR that touches
   `engine/internal/runner/record_runner.go`,
   `kafka_consumer.go`, `check_evaluator.go`, or the
   engine binary's record-mode boot translation loads the
   skill on description-match. The seven conventions
   become a reviewer surface, not a scavenger hunt
   through doc comments.

3. **The harness is pre-positioned for Track C.** The
   skill exists before Track C lands. A Track C PR
   touching record-mode code has a discoverable
   convention surface from session one. P-B3RM.5 commits
   this as the forcing function; the skill does not
   anticipate Track C's content, only its location.

4. **No platform code changes.** The study commits the
   contract; the SKILL.md + reference write is the
   follow-on implementation slice on a separate session
   per R4 + P-B3RM.3. No engine, rules, tools, or deploy
   code moves in this study.

5. **The
   [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
   Clause 3 precedent for new skills via B3-N entries is
   reused.** Same shape: B3-N study proposes the skill;
   the ADR promotion commits the contract; the
   implementation slice writes the files on a separate
   session. No new precedent opened.

6. **Future record-mode amendments amend the skill.**
   When β commit semantics are replaced by per-attempt
   re-read (the future slice named in S2's citation),
   the skill's S2 convention amends in place per
   [ADR-0050](../../docs/adr/0050-v1-retirement-engine-release.md)
   §Consequence 4's Amendment-log convention (or the
   skill superseded by a fresh ADR if the change
   reshapes the contract — the
   [ADR-0017](../../docs/adr/0017-substrate-posture-amendment.md)
   pattern). The choice is made inside the amending
   session per the
   [ADR-0052](../../docs/adr/0052-session-reading-router.md)
   §6.2 ADR amendment row's discipline.

7. **CI gates for the implementation slice are
   well-defined.** The implementation slice runs against
   AC-W3-3 (citation discipline — every S1–S7 cites a
   real `file:line` that exists in the merged main when
   the slice lands) and AC-W3-7 (local build / lint /
   test gates — `make lint` for the markdown surface; no
   engine / rules / tools / deploy gates apply because
   no production-code file is touched). The
   implementation slice's PR body declares AC-W3-7's
   markdown-lint gate as vacuous if no markdown-lint
   tool is wired (per
   [ADR-0052](../../docs/adr/0052-session-reading-router.md)
   §6.2 Implementation slice row's load-bearing AC-W3
   rows).

8. **A small risk surfaces around citation drift.** A
   citation pinned to `file:line` will go stale if the
   underlying file is edited. The implementation slice
   pins citations against the current main; future
   edits to the cited files must update the skill's
   citations in the same PR (R4-bundled). This is the
   same drift risk
   [`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
   carries; OQ-1 below names a verification mechanism if
   the drift becomes load-bearing.

9. **The skill's frontmatter description shape is
   committed.** The implementation slice writes the
   description per the §Recommendation draft above. If
   harness-side description-match testing reveals that
   the description does not load reliably on the named
   trigger phrases (the same risk
   [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
   Clause 3 flagged for `session-governance`), the
   implementation slice surfaces the gap and a follow-up
   B3-N entry revisits the description shape. This study
   does not pre-commit a description that may not load.

---

## Open Questions

- **OQ-1 — Citation-drift verification mechanism.** The
  seven conventions S1–S7 each cite a `file:line` range.
  A future PR that edits a cited file may invalidate the
  citation. The implementation slice could ship a
  `make lint-skill-citations` target (a grep-style sweep
  that confirms each `file:line` citation lands at a
  non-empty line in the cited file). Whether this is
  worth the harness work depends on how often the cited
  files change post-Wave-S. Resolved by the first
  citation-drift incident: if a record-mode PR ships
  with stale skill citations, the verification target is
  scoped via a follow-up B3-N entry.
  *Out-of-scope for current cycle:* deferred until a
  concrete drift incident or a Track C PR surfaces the
  cost.

- **OQ-2 — Track C scope and the skill's relationship to
  Track C's content.** P-B3RM.5 commits pre-positioning
  but not Track C's scope. Track C may surface
  conventions that the seven S1–S7 do not cover (e.g.,
  schema-conformance-check kind specifics, evidence-
  shape patterns for record-mode results). Those
  conventions ride their own B3-N entries; this study
  does not anticipate them.
  *Out-of-scope for current cycle:* Track C is the
  trigger; deferred until that session opens.

- **OQ-3 — Skill description shape and harness load
  reliability.** The draft description in §Recommendation
  lists ~12 trigger phrases. Whether the description-
  match mechanism reliably loads the skill on each phrase
  is a harness-side question carried forward from
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clause 3's same flag. The implementation slice
  verifies on a sample of the phrases; if any phrase
  fails to load reliably, the description is tightened
  in the same slice.
  *Out-of-scope for current cycle:* surfaces in the
  implementation slice's test plan, not in this study.

- **OQ-4 — Whether `engine/internal/runner/doc.go`
  should ship alongside the skill.** Option D in
  §Considered Options was rejected as the *only*
  surface, but a short `doc.go` naming the runner
  package's owned surface and not-imports (per the
  [`go-coding-standards`](../../.claude/skills/go-coding-standards/SKILL.md)
  C4 boundary discipline) may be a useful complement to
  the skill. The runner package currently does not have
  a `doc.go`. Whether to add one is a separate question
  from this study's scope; if added, it should cite the
  skill rather than duplicate it.
  *Out-of-scope for current cycle:* the likely lane is
  an **implementation slice** per
  [ADR-0052](../../docs/adr/0052-session-reading-router.md)
  §6.2 row 6 landing under whichever Wave-3 runner
  closure the package traces to (the runner package was
  scaffolded as part of Wave 3 P4); if the doc.go work
  surfaces new harness conventions instead of
  documenting existing ones, the lane is a follow-up
  B3-N entry. The choice is made when OQ-4 is taken up.

---

## Promotion target

`docs/adr/0053-record-mode-skill.md` (provisional;
operator-side reservation confirmed at
[`/promote-to-adr`](../../.claude/commands/promote-to-adr.md)
time per
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
Clause 7).
