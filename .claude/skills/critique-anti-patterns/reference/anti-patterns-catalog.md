<!-- path: .claude/skills/critique-anti-patterns/reference/anti-patterns-catalog.md -->

# Anti-patterns — catalog

Eight anti-patterns a critique must flag. Each is marked:

- **documented** — there is a real instance or verbatim template in
  the repository.
- **preventive** — no incident is preserved (critique findings are
  ephemeral in this project), but the protocol forbids it.

For "preventive" patterns: the absence of incidents is the project's
discipline working, not the absence of risk. Catch them by sweep.

---

## B1 — Reaction-style feedback in critique output

**Status:** documented (verbatim list in protocol).

**Definition:** Saying "this is bad / unclear / weird", "I don't like
X", "the previous was better", or "just rewrite this" instead of
giving a labeled, actionable finding.

**Source:** `.claude/playbooks/feedback-protocol.md:48-56`.

**Catch:** inspect your own critique output before posting. Every
finding must be one sentence in the template
`[severity] R/P/AC: section — change.`

---

## B2 — Prior-art-as-justification (R5)

**Status:** preventive (no incident known).

**Definition:** Citing an external product, vendor, sibling team's
internal project, or third-party design as the reason for an
architectural choice. CLAUDE.md R5 forbids this; AC-5 verifies it.

**Allowed:** environment commodities — BigQuery, Kafka, Pub/Sub,
OIDC, Prometheus, OpenTelemetry, Kubernetes, Go, Docker, slog, JSON
Schema. "We use X" is fine; "we are doing Y because X does Y" is not.

**Source:** `CLAUDE.md` §3 R5 + `.claude/playbooks/acceptance-criteria.md:20`.

**Catch:** AC-5 scan during critique. Grep options sections for
product names; for any hit, verify it's named as environment, not
borrowed idea.

**Template finding:** `R5: "Considered Options" — option 3 cites a
vendor by name as justification. Rewrite in our own terms.`
(Source: `feedback-protocol.md:35-36`.)

---

## B3 — Hidden commitments in Open Questions (AC-6)

**Status:** preventive (no incident known).

**Definition:** An Open Questions section item that quietly commits a
position rather than asking a question. The OQ section must be
question-shaped or explicitly out-of-scope.

**Source:** `.claude/playbooks/acceptance-criteria.md:21`.

**Catch:** every OQ item must be (a) resolved in the study, (b) marked
`out-of-scope for current cycle` with a one-line reason, or
(c) deferred with a forward-pointer. Anything else is a hidden
commitment.

**Template finding:** `AC-6: "Open Questions" — three items are
unresolved with no out-of-scope marker. Either resolve or defer
explicitly.` (Source: `feedback-protocol.md:43-44`.)

---

## B4 — Citation drift (R5 / AC-4)

**Status:** preventive (no incident known).

**Definition:** A claim cites a foundation doc or prior ADR but the
cited text does not actually support the claim. The citation looks
right; the underlying text doesn't.

**Source:** `CLAUDE.md` §3 R5 + `acceptance-criteria.md:19`.

**Catch:** open every load-bearing citation during critique and read
the surrounding context. If the cited section does not support the
claim, flag.

**Template finding:** `AC-4: "Recommendation" — cites
03-boundary-contract.md §"Compatibility window" as authority but that
section explicitly defers the duration to ADR-0035. Cite ADR-0035
directly or rewrite the claim.`

---

## B5 — "Additive" mislabel for breaking changes (P5)

**Status:** preventive (no incident known).

**Definition:** A change labeled `additive` or `non-breaking` that
is actually breaking. CLAUDE.md P5 requires evolution under a
published compatibility contract; mislabeling subverts that.

**The breaking list** (from `studies/foundation/03-boundary-contract.md:98-105`):

- a field is removed or renamed
- a field's type changes incompatibly
- a constraint becomes stricter (a previously-valid rule becomes
  invalid)
- a default changes in a way that alters runtime behavior
- an enum value is removed

**The non-breaking list** (`03-boundary-contract.md:107-113`):

- a new optional field is added
- a new enum value is added (only if existing consumers tolerate
  unknown values)
- a constraint is relaxed (a previously-invalid rule becomes valid)
- documentation strings are updated

**Catch:** compare the change against both lists explicitly. If it
appears on the breaking list, it is breaking even if the author
labeled it additive.

**Template finding:** `P5: "Schema changes" — change #2 removes the
`severity` enum value `informational`, which is on the breaking list
(03-boundary-contract.md:98-105). Bump schema version and revise the
non-breaking-change label.`

---

## B6 — Strawman options (AC-3)

**Status:** preventive (no incident known — explored studies present
fair options, e.g., `studies/decisions/2026-05-24-b0-s2-kind-catalog.md`
§"Considered Options").

**Definition:** An Options-Considered section where the non-preferred
options are weakened so the recommendation looks obvious. Asymmetric
pros/cons; rejection reasons that are dismissive ("this would be too
complex") rather than architectural ("this violates G3").

**Source:** `acceptance-criteria.md:18` (AC-3 requires ≥2 options
considered — fairness is implicit in the spirit of the criterion).

**Catch:** every rejected option must list both pros and cons.
Rejection reasons must cite a principle, rule, or constraint —
"too complex" alone is not architectural.

**Template finding:** `AC-3: "Considered Options" — option B lists
only cons; the pros that motivated including it are not stated. Add
balanced pros/cons or remove the option.`

---

## B7 — Vocabulary drift cross-section (P5 / AC-2)

**Status:** preventive (no incident known — the codebase maintains
glossary discipline; see e.g., `kind` used consistently throughout
the kind-catalog study).

**Definition:** The same concept named differently across sections
of one document. Examples that *would* be drift:

- `kind` vs `check type` vs `rule type` in the same study
- `schema` vs `shape` vs `structure` for the same JSON Schema
- `entity` vs `dataset` vs `table` for the same DSL concept

**Source:** `CLAUDE.md` §4 P5 (evolution must be contract-driven —
shared vocabulary is part of the contract) + `acceptance-criteria.md:17`.

**Catch:** scan the document for synonyms of any noun used in the
DSL or in the foundation docs. Pick one and use it everywhere; if a
substitute is needed for cadence, declare it in a glossary block.

**Template finding:** `P5: "Decision" §1 uses "kind" while §3 uses
"check type" for the same concept. Pick one (kind is the canonical
DSL term per ADR-0022) and use it throughout.`

---

## B8 — Unaddressed blocking findings (AC-9)

**Status:** documented (verbatim template in protocol).

**Definition:** A Recommendation that ignores a `blocking` finding
from a prior critique round without either resolving it in the study
or explicitly deferring it in Open Questions with a rationale.

**Source:** `.claude/playbooks/acceptance-criteria.md:24,29-34`.

A blocking finding the author *disagrees with* is not automatically
deferred — the author must rebut it in the study itself (typically in
Decision Drivers or Recommendation) and let the next critique surface
whether the rebuttal holds (`acceptance-criteria.md:29-34`).

**Catch:** cross-reference latest critique findings against the study
body and Open Questions.

**Template finding:** `AC-9: "Recommendation" — critique finding #2
(blocking) is not addressed and not deferred.`
(Source: `feedback-protocol.md:45-46`.)

---

## Worked example — the only documented critique cycle

Commit `926e3e5` (PR #10, ADR-0014 HTTP trigger handler) is the one
fully-documented multi-round critique in the repository:

> Critique round 1: 1 blocking + 1 important + 5 minor findings;
> all addressed in the same revision pass (decoder unknown-field
> detection refactored to two-pass via map[string]json.RawMessage
> + reflection-derived JSON tag set; defer recover + alert in
> handler panic path).

(Source: commit `926e3e5` message body.)

The findings themselves are not preserved beyond this summary. What
remains is the **shape** of a clean cycle: severity-bucketed findings,
revision pass that addresses all blocking + important in one
iteration, summary of the substantive fixes in the commit message
when the work lands.

When running `/critique`, aim for this shape: bucket findings by
severity, name each with a label, and keep the revision pass focused.
