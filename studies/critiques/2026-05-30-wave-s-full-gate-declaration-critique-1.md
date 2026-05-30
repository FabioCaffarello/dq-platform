<!-- path: studies/critiques/2026-05-30-wave-s-full-gate-declaration-critique-1.md -->

# Critique — Wave-S full gate declaration (PR-shaped, not study-shaped) — round 1

Round 1 is **short and targeted** per the operator's
instruction: the adversarial focus is *"does a gate closure need
ADR ceremony or does the decision log suffice?"* This is a
meta-classification question on the work this PR ships, not an
R/P/AC compliance review of any single file (no study artifact
exists — Flow 6 work has no study by definition; the decision-log
entry is the load-bearing artifact). The format mirrors round-1
captures for prior B3 sessions per
[ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)
preservation contract; the disposition discipline mirrors the
author-equals-reviewer rule from
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
§Consequence 7.

## Adversarial questions

### Q1. Does CONTRIBUTING.md Flow 6 actually cover this work?

**Yes, verbatim.** Flow 6 §"Scope — what qualifies" enumerates
*"Factual refreshes: phase status, playbook/command/skill lists,
pointers to live state, **dated closure notes**."* The
Wave-S-full-gate declaration IS a dated closure note — the
criterion in ADR-0020 was satisfied 2026-05-25; this PR records
the date in the canonical live-state surface (the decision log)
and converges the navigation surfaces (CLAUDE.md §2.2 +
AGENTS.md) that lagged. No new rule, no contract amendment.

### Q2. Does the precedent argument hold?

**Yes — Wave 1, Wave 2, and Wave 3 all closed without dedicated
closure ADRs.** The decision-log §Wave Gates section already
carries:

- *"Wave 1 gate (B0 complete) — gate met as of 2026-05-21"*
- *"Wave 2 gate (platform decisions complete) — gate met as of
  2026-05-21"*
- *"Wave 3 readiness — gate met as of 2026-05-21"*
- *"Wave 3 completion gate (all phases scaffolded) — gate met as
  of 2026-05-23"*

None of these has a corresponding "wave closure ADR". Each is a
derived fact recorded in §Wave Gates when its criterion was
satisfied. Wave-S following the same pattern preserves
continuity; introducing ADR ceremony for closure here would be
the novelty, not the safe default.

### Q3. Is the "downstream classification consequence" argument decisive enough to require ADR ceremony?

**No.** The argument for ADR is that gate-state changes the
ADR-0049 §(a) classification of future record-mode B-rows (from
B2-S to B3). The consequence is real, but it is **derived** from
two existing artifacts:
- ADR-0020 §"Full Wave-S gate" commits the criterion.
- ADR-0049 §(a) commits that classification turns on whether the
  surfacing item lands "inside an existing wave's gate" or
  "post-shipping against a closed wave".

The decision-log entry binds the predicate (gate-met date) to
both contracts via explicit citation. Future B-rows reading the
decision log get the citation chain end-to-end. An ADR would be
a parallel surface saying the same thing — useful if no other
durable surface existed, redundant when the decision log
already is that surface per CLAUDE.md §2.4 ("the canonical
live-state surface for every B-row and ADR is
`studies/foundation/06-decision-log.md`. Wave gates ... are all
reflected there in the 'Last updated' history.").

### Q4. What if a future operator overlooks the §Wave Gates entry?

**Defended by the always-on-floor router.** Per CLAUDE.md §6.1
+ ADR-0052, every session reads `studies/foundation/06-decision-log.md`
regardless of session type. The §Wave Gates section is part of
that always-on floor. The CLAUDE.md §2.2 + AGENTS.md flips in
this PR are the secondary navigation surfaces (reading order
points back at the decision log per CLAUDE.md §6.4 + AGENTS.md
line 38-40).

### Q5. Could a third-party reviewer argue this PR is "too big" for Flow 6?

**Yes, that is the substantive D0** — and it is exactly what
this PR surfaces explicitly in the PR body for operator
ratification per CONTRIBUTING.md Flow 5 §"Operator-side
responsibilities". The PR body does not pre-decide the
ratification; the author-equals-reviewer circularity (ADR-0051
§Consequence 7) means this critique cannot self-ratify either.
The D0 stays operator-owned until the merge act resolves it.

## Findings

**0 blocking / 0 important / 0 minor.**

No blocking finding: R1 / R5 / R6 / R8 not at risk; no contract
amendment; no rule introduced. AC-1…AC-10 don't apply (no study
shape).

No important finding: the precedent argument (Q2) is sound; the
downstream-consequence counter (Q3) holds; the always-on-floor
defense (Q4) is structurally adequate.

No minor finding: the decision-log entry's evidence list (7
B0-S → 7 ADRs with commit hashes) gives future readers
forensically-checkable grounding; the §Wave Gates section
mirrors the prior gate entries' shape.

## Disposition

The Flow 6 recommendation **survives the adversarial round**.
The D0 (Flow 6 vs ADR-ceremony) is surfaced explicitly in the
PR body for operator ratification; this critique does not
ratify it (author-equals-reviewer circularity per ADR-0051
§Consequence 7). If ratification swings to ADR-ceremony, the
decision-log entry already shipped in this PR carries the
forensically-checkable record forward — a follow-on session
drafts the closure ADR in addition.

## Hard stop

Per the session's explicit framing: the agent does NOT ratify
the D0 in any `/critique` output. The ratification is
operator-side per `CONTRIBUTING.md` Flow 5; recorded in the
round-2 trailer per ADR-0048 if the operator chooses to
explicit-ratify mid-PR (precedent B3-4 / B3-5), otherwise
implicitly resolved by the merge act per CONTRIBUTING.md Flow 5
§"Operator-side responsibilities".
