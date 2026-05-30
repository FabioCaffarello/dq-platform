<!-- path: studies/decisions/2026-05-30-b3-panel-5-lighting-strategy.md -->

# B3-5 — Panel 5 Lighting Strategy

## Metadata

- Date: 2026-05-30
- Status: draft
- Decision-log row: B3-5 (family fit derived in §Eligibility below; not assumed at registration)
- Promotion target: [`docs/adr/0056-panel-5-lighting-strategy.md`](../../docs/adr/0056-panel-5-lighting-strategy.md) (provisional; operator reserves at promotion time per [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md) Clause 7)
- Critique rounds:
  - Round 1 — [capture](../critiques/2026-05-30-b3-panel-5-lighting-strategy-critique-1.md) (0 blocking / 4 important / 5 minor); 4 important applied in the revision; minor deferred under the two-round cap.
  - Round 2 — [capture](../critiques/2026-05-30-b3-panel-5-lighting-strategy-critique-2.md) (0 blocking / 0 important / 0 minor) — ratification-trailer round, no fresh adversarial pass. Operator ratified the two coupled D0s on ADR-0049 §(a) Conditions 1 + 3 mid-PR-#113 per CONTRIBUTING.md Flow 5 by adopting the **weak reading** of ADR-0039 §"Metric contract" Meaning column wording plus the **A.y sub-path** (constant zero for engine-non-derivable series); the reading carries forward to ADR-0056 §Notes as new-contribution-requires-review per R5 + A7.

---

## Context

[ADR-0039](../../docs/adr/0039-dashboard-contract.md) §"Metric
contract" committed an eight-metric Prometheus inventory.
[ADR-0055](../../docs/adr/0055-metric-emission-slice-scope.md)
shipped emission for six of the eight; two — `dq_queue_depth` and
`dq_scheduler_triggers_managed` — are committed at the contract
layer but **the engine binary does not emit them today**. As a
direct consequence, `deploy/dashboards/baseline.json` panel #5
("Scheduler health") renders `"no data"` against both gauges.

B3-4's originating study (`2026-05-30-b3-metric-emission-slice-scope.md`)
deferred this lighting as **OQ-1**, with a one-sentence sketch of
two possible follow-ons: an amendment authorizing engine-side
proxies, or a scheduler-binary instrumentation slice. The text
named "ADR-0033 amendment" as the amendment target. That
attribution is **incorrect** on re-reading
[ADR-0033](../../docs/adr/0033-scheduler-catchup-behavior.md)
end-to-end: ADR-0033 commits the external-scheduler posture
(engine has no internal scheduler; lifecycle operations are
external best-effort per [ADR-0007](../../docs/adr/0007-loader-scheduler-retry-failure-semantics.md)
§5 + §6) but never names the metrics `dq_queue_depth` or
`dq_scheduler_triggers_managed`. The metric semantics live in
ADR-0039 §"Metric contract", whose Meaning column reads (verbatim):

> `dq_queue_depth` (gauge, labels `state` = `scheduled` |
> `running`): **Count of runs the scheduler currently tracks,
> split by state.**

> `dq_scheduler_triggers_managed` (gauge, labels `state` =
> `healthy` | `errored`): **Count of triggers the scheduler
> currently manages, split by state.**

So an amendment-shaped path, if pursued, would amend ADR-0039
(not ADR-0033). This study triages three concrete paths
(**Path A**, **Path B**, **Path C** below), but it deliberately
treats the §(a) classification of each path as a **hypothesis to
test**, not as inherited fact. In particular, whether Path A is
amendment or B3 hinges on a substantive interpretive question
the study must defend in the open: is ADR-0039's "scheduler
currently tracks" phrasing a **committed label-source rule** (in
which case redefining it is amendment) or **incidental
description of the only known emitter at ADR-0039 time** (in which
case emitting the same gauge from a different source is a clean
extension)? Both readings are defensible from the ADR-0039 text
alone; the §Eligibility section surfaces this as a D0
precondition for operator ratification per
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5
§"Operator-side responsibilities".

The principles bearing on the decision are **P3** (ownership is
explicit — operator panels read against committed labels;
redefining what `dq_queue_depth` means changes downstream
consumer interpretation), **P5** (evolution is contract-driven —
whichever path lands, the consumer surface either remains stable
or its compatibility cost is recorded explicitly), and **R3** (do
not revisit settled architecture — ADR-0033's external-scheduler
posture and ADR-0039's metric inventory are both preserved as
written unless an amendment is committed).

### Eligibility under ADR-0049 §(a)

Required per
[`.claude/playbooks/post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md)
step 2 for a B3-N entry. All four conditions must hold for B3.

| # | Condition | Disposition |
|---|---|---|
| 1 | **P-B3.1 — expands not rewrites** | **D0 — operator-ratified mid-PR-#113 on 2026-05-30** per [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5 §"Operator-side responsibilities" (trailer captured in [`studies/critiques/2026-05-30-b3-panel-5-lighting-strategy-critique-2.md`](../critiques/2026-05-30-b3-panel-5-lighting-strategy-critique-2.md)). Operator adopted the **weak reading**: the "scheduler currently tracks" phrase describes the only emitter known at ADR-0039 time but the committed semantics is "gauge of queue depth, label `state`". Path A.y is an extension — emitting the same metric from an engine-derived source with an additive `source="engine"` label per ADR-0039 §"Evolution rules". Condition 1 passes. The reading carries forward to ADR-0056 as new-contribution-requires-review per R5 + A7. |
| 2 | **P-B3.4 — in-scope family** | **Passes (Tooling extensions).** Path A — if eligible at all under Condition 1 — adds engine-runtime emission to two existing metric names, mirroring exactly the precedent set by ADR-0055's six-metric slice under the operator-ratified ADR-0049 Condition 2 reading (engine-runtime emission as Tooling-extensions per ADR-0051 Clause 1's adjacent-tooling reading). No new family fit; no new expansive reading. |
| 3 | **P-B3.2 — conforms to ADR-0020 / 0021 / 0022 / 0023 envelope** | **D0 — operator-ratified mid-PR-#113 on 2026-05-30** (same trailer as Condition 1). Resolution under the weak reading: the additive `source="engine"` label is the load-bearing mechanism — by self-identifying the gauge emission as engine-derived, the engine no longer claims scheduler-internal knowledge it doesn't have, so ADR-0033's external-scheduler boundary is preserved verbatim. ADR-0020 / 0021 / 0022 / 0023 envelope is untouched. Condition 3 passes. |
| 4 | **Additive-maintenance threshold** | **Passes.** Emitting two new metric series from new engine call sites (the runner's in-flight execution count + the engine binary's known-trigger registry) crosses materially-novel ground: new gauges with new sample collection cadence + a new cross-package state surface (engine binary reading runner's in-flight count or scheduler-side data the engine doesn't own). |

**Summary.** Two D0s surfaced (Conditions 1 + 3). They were
**coupled** — the resolution of Condition 1 determined
Condition 3's resolution. The operator ratified the **weak
reading** of ADR-0039's "scheduler currently tracks" wording
mid-PR-#113 on 2026-05-30 per CONTRIBUTING.md Flow 5, plus
the **A.y sub-path** (see §Considered Options → Option A's
2×2 table for sub-path classification). Trailer captured in
[`studies/critiques/2026-05-30-b3-panel-5-lighting-strategy-critique-2.md`](../critiques/2026-05-30-b3-panel-5-lighting-strategy-critique-2.md).
The reading carries forward to the promotion ADR (ADR-0056) as
**new contribution proposed here, requires review** (R5 + A7).
Structural-precedent shape mirrors B3-4 / PR #111 mid-PR
ratification (different conditions: B3-4 ratified Condition 2;
B3-5 ratified Conditions 1 + 3); B3-2 / PR #101 also mid-or-at-
merge ratification of two coupled D0s.

---

## Decision Drivers

- **DD-1 — Panel 5's dark state is honest but costs operator
  visibility.** Two gauges in ADR-0039's inventory render empty;
  operators reading panel 5 today see no signal at all about
  scheduler health.
- **DD-2 — ADR-0033 fixes the scheduler as external.** Any path
  that has the engine claim knowledge of scheduler-internal state
  is in tension with this commitment, regardless of how the
  ADR-0039 wording is read.
- **DD-3 — The platform owns no scheduler binary today.**
  ADR-0033 §"Why this does NOT commit specific scheduler tooling"
  is explicit: "The platform supports any external scheduler that
  can emit HTTP POST `/v1/trigger`." A scheduler-binary path
  presupposes the platform deciding to own one — a wave-scale
  decision well beyond B3 envelope.
- **DD-4 — Reading-of-record on ADR-0039's Meaning column is not
  re-settled by this study.** ADR-0039 was promoted four days
  ago (2026-05-26); no operator interpretation of "scheduler
  currently tracks" has been committed since. This study surfaces
  the ambiguity but does not pick a reading — that's the D0.

---

## Considered Options

Three concrete paths. Each path's §(a) classification is **derived
inside the option**, not inherited from the originating study's
sketch.

### Option A — Engine-side emission of both gauges (interpretive question on amendment vs extension)

The engine binary emits both `dq_queue_depth` and
`dq_scheduler_triggers_managed` from sources the engine binary
itself can observe:

- `dq_queue_depth{state="running"}` = count of `dq_executions`
  rows whose `status="running"` and `recorded_at` more recent
  than the engine's orphan-detection threshold. Observable via
  the existing `Store` reader surface (no new query template).
- `dq_queue_depth{state="scheduled"}` = always 0 from the engine
  binary (the engine has no concept of "scheduled, not yet
  triggered"; that's external scheduler state). Honest reporting
  via constant zero, OR drop the `state="scheduled"` series
  entirely from engine emission (panel 5's query may need adjustment).
- `dq_scheduler_triggers_managed{state=*}` = no engine-derivable
  signal. Two sub-paths:
  - **A.x** = drop the metric from engine emission entirely
    (panel 5 partial light only).
  - **A.y** = emit constant 0 for both states (panel 5 fully
    plotted but the `errored` line never deviates).

The §(a) classification of Option A turns on **two** axes: the
Condition 1 / 3 D0 (weak vs strong reading of ADR-0039) AND
the sub-path choice (A.x = drop the unsalvageable metric vs
A.y = emit constant zero). The 2×2 table:

| Reading × Sub-path | A.x (drop `dq_scheduler_triggers_managed`) | A.y (emit constant zero) |
|---|---|---|
| **Weak** (gauge meaning is the committed contract; source incidental) | **Amendment.** Dropping a metric from engine emission ≠ the inventory in ADR-0039. Even under the weak reading, the metric *name* is committed; engine declining to emit it makes the inventory and the engine ship out-of-sync. | **Extension (B3-eligible).** Metric remains in the inventory; engine emits a value; information content is minimal (constant 0) but the gauge is present. Closest fit to a clean B3 extension under the weak reading. |
| **Strong** (source is part of committed label-source rule) | **Amendment.** Both the source change AND the metric drop amend the contract. | **Amendment.** The source change alone amends the contract; the constant-zero value-shape doesn't rescue it. |

Three of the four cells are amendment-shaped; only the
(weak-reading, A.y) cell is B3-eligible. The operator's
ratification on Condition 1 (weak vs strong) plus a separate
disposition on A.x vs A.y is required to land Option A inside
the B3 envelope. The §Eligibility table's Conditions 1 + 3
both resolve to "extension" only in the (weak, A.y) cell.

**Cost surface (orthogonal to amendment-vs-extension):** A new
read on the engine's Store interface every Prometheus scrape
interval (15s in compose; per-env operator-tunable in
production). Partition-pruned by `recorded_at` per ADR-0031, so
the per-scrape scan cost is bounded; ADR-0039 §"Cardinality
posture" preservation per ADR-0055 §Clause 6 still applies.

### Option B — Scheduler-binary instrumentation slice

The platform commits to owning a scheduler binary that emits the
two scheduler-side metrics natively. This:

1. Re-opens ADR-0033's "external scheduler" framing (the engine
   suite would now own a scheduler).
2. Triggers a Wave-S-style launch decision per ADR-0049 §(b)'s
   substrate / new-mode exemption pattern — a new platform
   component with its own deployment surface, observability,
   release cadence, and HA story (per ADR-0033 §Notes —
   "leader election or distributed locks to avoid duplicate
   trigger emission").
3. Fails ADR-0049 §(a) Condition 2 cleanly — owning a scheduler
   binary is not in any of the three B3-eligible families (kind /
   capability mode / tooling). It is wave-scale work, not B3
   evolutionary work.

**Classification:** **Rejected** per ADR-0049 §(a). The §(a)
Rejected definition is *iff the proposal falls outside the
three in-scope families **and** outside any active wave's
gate*. Option B fails Condition 2 (not in any of kind /
capability mode / tooling) AND is outside any active wave's
gate (no scheduler-binary wave exists or is in flight). Both
clauses hold. The Rejected branch explicitly allows the
one-line decision-log entry pointing at a rationale note; this
study is that rationale. No B-row is opened for Option B; the
decision-log B3-5 row's "Earlier update" entry (registered at
study close) carries the rejection forward. If concrete demand
for an in-platform scheduler surfaces later, that's a
wave-launch decision — separate framing, separate ADR, not a
re-opening of Option B here.

### Option C — Permanent deferral; panel 5 stays dark

Document panel 5's `"no data"` state as the platform's honest
position. The dashboard panel description (already in
`baseline.json`) becomes the operator's runbook entry: "panel 5
is dark by design; if scheduler health visibility is needed,
operators wire it from their external scheduler's own
observability."

**Classification:** Not a §(a) entry at all — it's the absence
of a path. Logged as a one-line decision per ADR-0049 §(a)
"rejected" branch's documentation discipline.

---

## Recommendation

**The D0 has been ratified — weak reading + A.y sub-path
(2026-05-30, mid-PR-#113).** Path A.y is the actionable B3
follow-on. The contingent-recommendation framing below is
preserved verbatim so future readers can see the bifurcation
shape the study triaged; the actionable branch is §1.

1. **If the operator ratifies the weak reading of ADR-0039**
   (the gauge meaning is the committed semantics; the
   "scheduler" word in the Meaning column was the
   ADR-0039-time-known emitter, not a label-source commitment),
   AND adopts A.y over A.x (per the §Option A 2×2 table —
   A.x is amendment under either reading):
   - Path A.y becomes a B3 follow-on slice in the cleanest of
     the four cells.
   - Recommended follow-on: a new implementation slice landing
     under closed B3-5 that emits
     `dq_queue_depth{state="running"}` from a
     `Store.LatestExecutionPerEntityCheck`-style reader, emits
     constant zero for the other series the engine cannot
     derive, and adds an **additive label** (e.g., `source=
     "engine"` alongside the existing `state` label) so the
     engine's emission self-identifies as engine-derived
     without changing the metric name. Additive labels are
     explicitly allowed per ADR-0039 §"Evolution rules" within
     an engine-major version; the label addition is
     extension-shaped, not amendment-shaped.
   - **Renaming the metric** (e.g., `dq_queue_depth` →
     `dq_engine_inflight_runs`) is **out of scope for the
     weak-reading B3-eligible path** — the metric name is part
     of ADR-0039's committed contract surface (operators key
     on `dq_runs_total` etc. exactly because the name is the
     contract). A rename would route through amendment
     regardless of the Condition 1 reading. The additive-label
     mechanism is the only weak-reading-compatible way to
     surface the source distinction without amending the
     contract.

2. **If the operator ratifies the strong reading of ADR-0039**
   (the source IS part of the committed label-source rule;
   "scheduler currently tracks" is binding):
   - Path A is amendment-shaped, exits B3 scope, and routes
     through the amendment process. The amendment ADR amends
     ADR-0039 §"Metric contract" and is justified by the
     redefinition of `dq_queue_depth`'s Meaning column.
   - The amendment ADR is a separate session per R4 (one topic
     per session).

3. **In either ratification outcome, Option B remains out of
   scope** until concrete demand for the platform to own a
   scheduler binary surfaces. Option B is documented here as
   wave-scale; no row is opened for it in this session.

4. **Option C (permanent deferral)** is the implicit fallback if
   neither ratification produces actionable demand. No B-row
   needed; the existing `baseline.json` panel-5 description and
   `dq-platform/deploy/dashboards/README.md` together carry the
   "dark by design" reading already.

### What this study commits

- The ADR-0049 §(a) eligibility analysis above, with the two
  D0s on Conditions 1 + 3 explicitly tied to the
  ADR-0039-wording interpretive question.
- The three-option triage with each option's §(a) classification
  derived from defensible reading of ADR-0033 + ADR-0039 + the
  envelope ADRs (0020 / 0021 / 0022 / 0023), not inherited from
  the originating study's sketch.
- The follow-on dispositions above conditional on each
  ratification branch.

### What this study does NOT commit

- A pre-selected path. The triage is the artifact; the
  ratification is the next step.
- Any change to ADR-0033, ADR-0039, or the existing emission
  slice from ADR-0055. The engine's metric channel today is
  honest about what it does and does not emit.
- The exact reframing of ADR-0039's Meaning column wording (if
  the weak-reading branch is ratified). That belongs to the
  follow-on slice's ADR, not to this triage.
- Any decision about Option B. Wave-scale work needs its own
  framing; not a B3 surface.

---

## Consequences

1. **B3-5 reaches `resolved-study` with two D0s
   (Conditions 1 + 3) pending operator ratification.** Same
   precedent shape as B3-2 (Conditions 1 + 4 ratified at
   merge); recorded in the round-2 critique trailer per
   ADR-0048 + CONTRIBUTING.md Flow 5 §"Operator-side
   responsibilities".

2. **No code or contract change ships from this session.** The
   study is design-and-triage-only; the follow-on session
   (whichever branch the ratification picks) ships the actual
   change.

3. **ADR-0039's Meaning column ambiguity is now surfaced as
   a load-bearing interpretive question.** Future contract
   drafters reading ADR-0039 will find this study via the
   decision-log B3-5 row's link and can see the dual-reading
   structure explicitly. The ratification commits the reading
   for B3-5's follow-on; future B-rows that touch the same
   wording inherit the ratified reading by precedent.

4. **The originating B3-4 OQ-1's "ADR-0033 amendment"
   attribution is corrected inline in this study's Context.**
   ADR-0055 itself is not touched — touching a merged ADR is
   amendment territory governed by ADR-0050 §Consequence 4
   (in-place Amendment-log) or by a standalone amendment ADR,
   and the originating OQ-1 attribution is a study-side
   sketch (not a contract surface), not material enough to
   justify either mechanism. The inheritance chain that
   carries the correction to future readers: B3-5 study
   Context → decision-log B3-5 row "Earlier update" entry
   (registered at study close) → the row's "Earlier update"
   entry from B3-4 (still pointing at B3-4's OQ-1 sketch). A
   future reader who walks the chain sees the correction
   surfaced explicitly without needing ADR-0055 itself to be
   updated.

5. **Option B (scheduler-binary) is Rejected per ADR-0049 §(a).**
   The §(a) Rejected branch holds (fails Condition 2 AND
   outside any active wave's gate); the Rejected outcome's
   one-line-entry discipline is satisfied by the decision-log
   B3-5 row's update at study close. No new B-row is opened
   for Option B. If concrete operator demand for an
   in-platform scheduler surfaces later, that is a wave-launch
   decision (per ADR-0033 §Notes — "deferred indefinitely
   unless concrete operator demand surfaces"), not a re-opening
   of this rejection.

6. **Option C (permanent deferral) requires no new ADR.** The
   existing `baseline.json` panel-5 description + `deploy/
   dashboards/README.md` maturity disclaimer (per ADR-0045
   §"Local-development integration" Consequences) together carry
   the "dark by design" reading. Option C is a non-action by
   construction.

7. **The follow-on slice (if Path A is actionable) needs its
   own R6 / R8 / AC discipline.** This study does not pre-empt
   the follow-on's structure; standard post-Wave-3 loop applies
   to it.

---

## Open Questions

- **OQ-1: Operator interpretation of ADR-0039's "scheduler
  currently tracks" phrasing.** Out of scope for this study —
  it IS the D0 ratification the operator owns per
  CONTRIBUTING.md Flow 5. Not deferred to a future cycle;
  surfaced explicitly so the ratification produces a
  precedented reading future B-rows can inherit. **Out-of-scope
  for current cycle resolution** — operator-decision-shaped,
  not author-decision-shaped.

- **OQ-2: Reframing wording for ADR-0039's Meaning column if
  Path A is ratified under the weak reading.** Out of scope for
  this study — belongs to the follow-on slice's ADR. The reframing
  itself is borderline amendment territory and gets its own
  critique cycle there.

- **OQ-3: Scheduler-binary cadence and HA shape if Path B is
  ever pursued.** Out of scope — wave-scale design topic;
  reserved until concrete demand for in-platform scheduler
  surfaces (per ADR-0033 §Notes — "deferred indefinitely
  unless concrete operator demand surfaces"). **Out-of-scope
  for current cycle** — no demand signal today.

- **OQ-4: Whether `dq_scheduler_triggers_managed` is salvageable
  at all under Path A.** Under the weak reading, the engine can
  emit `dq_queue_depth{state="running"}` from observable
  in-flight count; but `dq_scheduler_triggers_managed{state=
  healthy|errored}` has no engine-derivable analog. The follow-on
  slice ADR decides whether to drop the metric from engine
  emission, emit constant zero, or rename it. **Out-of-scope for
  current cycle** — belongs to the follow-on slice ADR; the
  triage doesn't need to resolve it.

- **OQ-5: Cost-aware scrape-frequency adjustment for the new
  reader call.** Path A introduces a new Store read per
  Prometheus scrape (15s default). Partition-pruned per
  ADR-0031, so per-scrape cost is bounded — but a tighter prod
  scrape cadence would amplify it. **Out-of-scope for current
  cycle** — sizing/tuning is a follow-on slice concern; this
  triage notes the cost surface exists.

---

## Promotion target

[`docs/adr/0056-panel-5-lighting-strategy.md`](../../docs/adr/0056-panel-5-lighting-strategy.md)
(provisional; operator reserves at promotion time per
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md) Clause 7).

The promotion ADR shape depends on the ratification:

- If the **weak reading** is ratified: the promotion ADR commits
  Path A as the actionable path, records the ratified D0 reading
  as new-contribution-requiring-review per R5 + A7, and points to
  the follow-on slice (a separate session per R4).
- If the **strong reading** is ratified: the promotion ADR
  commits Option C (or routes to a separate amendment session
  per ADR-0049 §(a)); Path A is documented as amendment-shaped
  in the supersession-chain ADR rather than promoted as a B3
  outcome here.
