<!-- path: studies/critiques/2026-05-29-b2-20-v1-retirement-engine-release-critique-1.md -->

# B2-20 — Critique Round 1

## Metadata

- **Target study:** [`../decisions/2026-05-29-b2-20-v1-retirement-engine-release.md`](../decisions/2026-05-29-b2-20-v1-retirement-engine-release.md)
- **Round:** 1 of (cap 2 per `wave-1-session-loop.md` step 7)
- **Date:** 2026-05-29
- **`/critique` invocation:** 2026-05-29, after the initial draft
  (study file uncommitted at critique time; close commit follows
  this preservation commit).
- **Preservation status:** preserved under ADR-0048 protocol.
- **Closing commit:** TBD — the next commit in `git log`
  immediately following this preservation commit; commit message
  `docs(decision): B2-20 — round 1 cleared, study resolved`.

---

## Critique Output

> Reproduced verbatim from the `/critique` invocation. The original
> output structured findings as fenced code blocks; that structure
> is preserved here.

**Sweep coverage:** R1–R8, P1–P6, AC-1…AC-10, B1–B8 (substantive
anti-patterns).

```text
[important] AC-6 / B3: "Open Questions OQ-B220.1" — The Recommendation summary
table and §(c) both commit to "in-place amendment to ADR-0035". OQ-B220.1 then
re-opens the same decision as deferred-to-operator. Either resolve in §(c)
(drop OQ-B220.1) or remove the commitment from the summary table and §(c)
(keep OQ-B220.1 genuinely open). A committed default re-shelved as an OQ is a
hidden commitment.

[important] R5 / B4: "Recommendation §(c) ADR-0035 amendment mechanism" — The
ADR-0017 precedent citation reads "ADR-0017 amended ADR-0010 in place
(substrate-posture amendment landed as a separate follow-up ADR)" — these two
clauses contradict each other (in-place vs. separate-ADR). Verify ADR-0017's
actual mechanism and rewrite the precedent claim, or drop the precedent and
defend the in-place choice on its own merits.

[important] R2: "Considered Options Option B" and "Recommendation §(a)" — The
phrase "release window" / "engineering calendar's normal release cadence"
imports a concept that is not codified anywhere in this repo. The working
definition ("next weekday after floor expiry") is concrete; the framing is
not. Either (a) drop "release window" terminology and state the rule as "first
weekday on or after floor expiry, subject to no incident/merge-freeze", or
(b) mark "release-window codification" explicitly as new contribution proposed
here per R5/AC-4. Currently inferred without a marker.

[minor] R5 / AC-4: "Recommendation §(c) Amendment log subsection" — Adding a
dated "Amendment log" subsection to ADR-0035 is a new convention not used in
prior ADR amendments. Mark explicitly as "new contribution proposed here,
requires review" per R5, or cite an existing-ADR precedent that uses the same
shape.

[minor] P5: "Decision Drivers DD-B220.6 and Recommendation §(c)" — The
compatibility-state table evolution is governed by P5 (evolution must be
contract-driven) at least as much as by P4 (bounded complexity). Add an
explicit P5 citation to DD-B220.6 or §(c); the table is the contract surface
that future v(N) retirements depend on.

[minor] AC-6: "Open Questions OQ-B220.3" — The OQ states explicitly that the
v1-drop is not B3 work and that the v(N+1) introduction-side mechanism is
ADR-0049 §(b) territory — i.e., it does not affect any B2-20 decision. Either
re-mark as a pure forward-pointer (not an OQ) or remove. As written it adds
surface without an open question.

[minor] B7 / AC-2: "Recommendation §(a) and Considered Options Option A" —
The 2026-08-23 Sunday / 2026-08-24 Monday claim is load-bearing for the
committed release date. Verify the day-of-week calculation against a calendar
before promotion (Zeller / `date` command). A wrong day-of-week silently
shifts the precedent rule.

[minor] R2: "Consequences C-B220.2" — "the manifest publisher's
engine_compatibility field range, when emitted by the next publish run, must
include 0.2.0" leaves the mechanism ambiguous: is the range auto-derived from
the engine binary's version at publish time, or is it a hand-maintained field
in the publisher config? Add one line of clarification, or move to OQ if the
mechanism is undecided.

[minor] AC-2 / clarity: "Consequences Per-artifact" — `testdata/valid/`,
`testdata/invalid/`, `lint_test.go`, `spec_test.go` are referenced without
their workspace prefix. ADR-0035 §C4 uses the same unqualified shape, so the
study is consistent with its upstream, but the drop-PR reviewer will need the
fully-qualified paths (`tools/lint/testdata/...` etc.). Add the prefixes in
the Per-artifact list to make the cohort split unambiguous.
```

### Sweep totals

| Severity | Count |
|---|---|
| `blocking` | **0** |
| `important` | 3 |
| `minor` | 6 |

---

## Operator Response

Disposition of each finding, in critique-output order:

1. **`[important] AC-6 / B3` — OQ-B220.1 hidden commitment.**
   *Applied as recommended.* Resolved in §(c) by committing
   in-place amendment cleanly; OQ-B220.1 removed from Open
   Questions. §(c) now defends the in-place choice on its own
   merits (ADR-0035 §C2 explicitly commits the table as the
   long-lived authoritative reference; amending in place
   preserves a single source of truth, whereas superseding
   would fragment the truth across multiple ADRs).

2. **`[important] R5 / B4` — ADR-0017 citation drift.**
   *Applied with variation.* Verified ADR-0017's actual
   mechanism: it is a **separate follow-up ADR that amends
   ADR-0010** (R5 audit on `docs/adr/0017-substrate-posture-amendment.md`
   §"Status: accepted (amends ADR-0010)"). The original claim
   that ADR-0017 amended in place was wrong — ADR-0017 is in
   fact the superseding-style precedent. §(c) rewritten to
   acknowledge the precedent honestly and to defend the
   in-place choice on the basis that ADR-0017 amended a
   *taxonomy row reclassification* (scope change), whereas
   B2-20 amends a *compatibility-state row transition*
   (bookkeeping over a committed structure). Different
   surfaces, different mechanisms.

3. **`[important] R2` — "release window" terminology.**
   *Applied as recommended.* Option (a) chosen: dropped
   "release window" framing throughout §(a), Option B, and
   the summary table. The rule is now stated directly as
   "first weekday on or after floor expiry, subject to no
   incident/merge-freeze". OQ-B220.2 (release-cadence
   codification) retained as a genuine deferred OQ since it
   names a real future codification question.

4. **`[minor] R5 / AC-4` — Amendment log subsection.**
   *Applied as recommended.* §(c) now explicitly marks the
   "Amendment log" subsection as "new contribution proposed
   here, requires review" per R5/AC-4. No prior ADR uses this
   shape; the convention is proposed by this study.

5. **`[minor] P5` — P5 citation in DD-B220.6 / §(c).**
   *Applied as recommended.* DD-B220.6 rewritten to cite P5
   (evolution must be contract-driven) alongside the long-
   lived artifact framing. §(c) opening sentence cites P5 as
   the principle governing the table's role.

6. **`[minor] AC-6` — OQ-B220.3 forward-pointer.**
   *Applied as recommended.* OQ-B220.3 removed from Open
   Questions; the v(N+1) introduction-side note relocated to
   a one-line forward pointer in §Consequences C-B220.5
   ("90-day-floor precedent solidifies") since that is where
   the precedent-generalization claim lives.

7. **`[minor] B7 / AC-2` — Day-of-week verification.**
   *Applied as recommended.* Verified via `date -j -f "%Y-%m-%d"
   "2026-08-23" "+%A"` → `Sunday`; `date -j -f "%Y-%m-%d"
   "2026-08-24" "+%A"` → `Monday`. Both confirmed. No textual
   change needed; the claim stands.

8. **`[minor] R2` — engine_compatibility range mechanism.**
   *Applied as recommended.* C-B220.2 now clarifies that the
   `engine_compatibility` range is a hand-maintained
   publisher-config field (per ADR-0005 §"manifest body
   fields"), so the drop PR must include the publisher-config
   update alongside the engine version bump. One-line
   clarification added; no new OQ.

9. **`[minor] AC-2` — Per-artifact path qualification.**
   *Applied as recommended.* §Consequences Per-artifact list
   now uses fully-qualified workspace-prefixed paths
   (`tools/lint/testdata/valid/`, `engine/internal/dsl/spec/parse_test.go`,
   etc.). The cohort split §(d) text retains the upstream-
   consistent unqualified shape for narrative continuity, with
   one cross-reference to the Per-artifact list for the exact
   paths.

**Sweep result:** 9 findings; 0 blocking, 3 important, 6 minor;
all addressed in the revision pass. Per `wave-1-session-loop.md`
step 7 cap, no round-2 needed.

Per ADR-0048 §Skip, the study's Metadata now declares:
`Critique rounds: 1 preserved
(studies/critiques/2026-05-29-b2-20-v1-retirement-engine-release-critique-1.md)`.
