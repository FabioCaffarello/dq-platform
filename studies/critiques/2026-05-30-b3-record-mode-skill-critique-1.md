<!-- path: studies/critiques/2026-05-30-b3-record-mode-skill-critique-1.md -->

# B3-3 — Critique Round 1

## Metadata

- **Target study:**
  [`studies/decisions/2026-05-30-b3-record-mode-skill.md`](../decisions/2026-05-30-b3-record-mode-skill.md).
- **Round:** 1.
- **Date:** 2026-05-30.
- **Preservation status:** preserved (this file is the round-1
  capture; round-2 capture will be a sibling
  `-critique-2.md` if a second round runs).
- **Closing commit hash:** filled in at the close-PR step of
  the post-Wave-3 session loop.

## Critique Output

```text
Adversarial review of
studies/decisions/2026-05-30-b3-record-mode-skill.md.
Grounding: CLAUDE.md §3 (R1-R8), §4 (P1-P6),
acceptance-criteria.md (AC-1..AC-10),
feedback-protocol.md, ADR-0049 §(a) eligibility filter.
Spot-checked citations: record_runner.go line 31 starts
the RecordConsumer comment block; record_runner.go:381
is the *Runner-satisfies-TriggerDispatcher assertion;
record_runner.go:403 is the var _ = sync.Mutex{}
sentinel; kafka_consumer.go:62-67 covers the kgo.Opt
block including DisableAutoCommit. Citation set largely
checks out.

[no-blocking] No R1, R2, R3, R4, R5, R6, R7, R8
violation found in the draft. R1 historical. R4 stays
inside one topic (proposes one skill; implementation
slice deferred). R5 — the franz-go / kgo naming is
descriptive of the library the code actually uses, not
prescriptive (see [minor] note F2). R6 path header on
line 1 confirmed. Eligibility check at the Metadata
block passes all four conditions of ADR-0049 §(a)
without borderlines — Condition 4 reuses the
ADR-0051 Clause 3 precedent (a new skill consolidating
existing discipline) cleanly.

[important] P5: "Locked premises P-B3RM.5 + Context
'Why now' block — Track C as forcing function" —
P-B3RM.5 leans heavily on Track C as the load-bearing
"why now". But Track C scope is explicitly deferred
(OQ-2). The pre-positioning argument risks reading as
circular: "we need the skill because Track C needs it"
combined with "Track C scope is TBD". The discoverability
gap for record-mode PRs *today* (~5 record-mode files
across runner + main.go + struct_mirror_test, with
conventions scattered across doc comments) is a more
direct motivation that does not depend on speculative
future scope. Suggest: down-weight Track C in P-B3RM.5
and Context "Why now", up-weight today's discoverability
benefit. Track C becomes a bonus rather than the
load-bearing rationale.

[important] P5: "Recommendation §The seven conventions
— S3 translation-at-boot" — S3 cites the engine binary's
buildRecordRunners (main.go:624-703) as the translation
site but does NOT cite the actual translation method,
RuleSpec.ToCheckSpecs at engine/internal/dsl/spec/spec.go:
114-127. Without that citation, S3 reads as if the
engine binary does the translation inline; the actual
translation goes through a named method on RuleSpec
that produces []runner.CheckSpec from the parsed rule's
checks. Suggest: add the dsl/spec.go:114-127
citation to S3's citation set; explain that the engine
binary calls ToCheckSpecs() to produce the
runner.CheckSpec slice that the runner consumes via
TriggerRequest.

[important] P5: "Recommendation §The seven conventions
— S6 single-goroutine state machine" — S6 cites the doc
comment naming the invariant and the Start poll loop.
But the *verification* of the no-internal-locking
claim (no sync.Mutex field on RecordRunner) is the
load-bearing assertion. The anti-patterns section
mentions the record_runner.go:403 var _ = sync.Mutex{}
sentinel separately, but S6's own citation set should
include it: the sentinel is the explicit-import marker
that documents the deliberate absence of mutex-protected
fields. Suggest: extend S6's citations to include
record_runner.go:399-403 (the sync-import sentinel
comment + the var line).

[minor] R5: "Recommendation §S1 / §S2 — franz-go and
kgo naming" — franz-go (the Kafka client library) and
kgo (its types package) are named in S1 (FetchedRecord
mapping) and S2 (DisableAutoCommit, CommitUncommitted-
Offsets). R5 exempts commodity substrates (Kafka, GCS,
OIDC, etc.); third-party Go libraries that implement a
client protocol sit on the boundary. The naming here is
descriptive of what the production code uses, not
prescriptive ("we use franz-go" is fine; "we are doing X
because franz-go does X" is not). The study reads as
descriptive. Suggest a one-line clarification: in
§Recommendation framing, note that franz-go is named as
the library used in production code (substrate-adjacent
environment), not as a pattern source. This heads off
the same critique at ADR-promotion time.

[minor] AC-7: "Promotion target — A8 form" — the
adr-writing skill A8 names a specific verbatim form
("subject to the same numbering caveat ADR-0020
§'Per-item ADR numbering' carries — NNNN is descriptive
of the planned sequence"). The study's Promotion target
line uses a different shorthand ("provisional; operator-
side reservation confirmed at /promote-to-adr time per
ADR-0051 Clause 7"). The B3-2 study used the same
shorthand and it landed cleanly, so the precedent
exists. Probably acceptable as-is; raised so the
implementation slice and promotion session don't
re-derive whether A8 binds.

[minor] AC-2: "Recommendation §Skill description
(frontmatter) — dense paragraph" — the draft
description runs ~25 lines mixing the "what" of the
skill, the seven-convention summary, and the trigger
phrase list. The current shape mirrors go-coding-
standards' description in length and style. Could be
tightened by splitting "description" (prose) from
"trigger phrases" (bulleted list), making the
implementation slice's grep cleaner. No structural
issue; style note only.

[minor] AC-6: "Open Questions §OQ-4 — runner package
doc.go addition" — OQ-4 labels the follow-up as "a
follow-up B2 entry (implementation-phase scaffolding)".
Wave 3 is closed, so B2 (implementation-phase decisions
against an in-flight wave) is the wrong lane label.
The correct lane under the post-Wave-3 router is either
the Implementation slice row (ADR-0052 §6.2 row 6 —
landing scaffolding under a closed B-row) if the runner
package closure traces to a closed B-row, OR a new B3-N
entry if the doc.go addition is itself a new harness
extension. Suggest: re-label OQ-4 as "a follow-up
implementation slice landing under the closed Wave-3
runner scaffolding closure (or a separate B3-N entry if
the doc.go addition surfaces new harness conventions)";
defer the choice to whenever OQ-4 is taken up.

Acceptance criteria sweep:
- AC-1 (path header): pass.
- AC-2 (required sections in order): pass.
- AC-3 (≥2 options): pass — four options A/B/C/D.
- AC-4 (Recommendation grounded): pass — cites
  ADR-0049, -0051, -0052, -0024, -0021, -0050, -0017,
  CLAUDE.md, foundation 04 coupling discipline.
- AC-5 (no external naming as justification): pass with
  [minor] note F2 above (franz-go naming clarification).
- AC-6 (Open Questions marked): pass with [minor] note
  F7 above (OQ-4 lane re-label).
- AC-7 (Promotion target concrete): pass — docs/adr/
  0053-record-mode-skill.md.
- AC-8 (≥1 critique round): pass after this file
  commits.
- AC-9 (blocking findings addressed): N/A — no blocking
  findings in round 1.
- AC-10 (decision-log row updated): pending step 9 of
  the loop; not yet a violation.

Summary: 0 blocking / 3 important / 4 minor. The three
important findings cluster on §Recommendation: Track C
forcing-function weighting (F1), S3 missing
ToCheckSpecs citation (F2/F4), S6 missing sync.Mutex
sentinel citation (F5). Round 2 expected after the
operator dispositions the round-1 findings.
```

## Operator Response

- **[important] P5 Track C forcing function** —
  *applied as recommended*: P-B3RM.5 and the Context
  "Why now" block will down-weight Track C and up-weight
  today's discoverability benefit (the gap exists now,
  not contingent on Track C scope).

- **[important] P5 S3 missing ToCheckSpecs citation** —
  *applied as recommended*: S3 will add the
  `engine/internal/dsl/spec/spec.go:114-127` citation
  for the `ToCheckSpecs` method and explain that the
  engine binary calls it during boot translation.

- **[important] P5 S6 missing sync.Mutex sentinel
  citation** — *applied as recommended*: S6's citation
  set will include `engine/internal/runner/record_runner.go:399-403`
  (the sync-import sentinel that documents the
  deliberate absence).

- **[minor] R5 franz-go naming clarification** —
  *applied as recommended*: §Recommendation will add a
  one-line scope note that franz-go is named as the
  library the production code uses (substrate-adjacent
  environment), not as a pattern source.

- **[minor] AC-7 Promotion target A8 form** —
  *accepted as-is*: the B3-2 precedent for the
  shorthand stands; no change required.

- **[minor] AC-2 Skill description density** —
  *accepted as-is*: the current shape mirrors
  go-coding-standards' description in length and style;
  no structural issue.

- **[minor] AC-6 OQ-4 lane re-label** —
  *applied as recommended*: OQ-4 will name the
  Implementation slice row (ADR-0052 §6.2 row 6) as
  the likely lane, with a fallback to a new B3-N entry
  if the doc.go surfaces new harness conventions.

All findings dispositioned. Revision follows in the next
commit on this branch.
