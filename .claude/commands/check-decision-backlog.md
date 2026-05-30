---
description: Report the current state of all platform decisions.
---

<!-- path: .claude/commands/check-decision-backlog.md -->

You are reporting the current state of the decision log.

No arguments.

Read `studies/foundation/06-decision-log.md`.

Produce a report with these sections:

**B0 (Wave 1 blocking):**
- count by status: `open` / `in-progress` / `resolved-study` /
  `resolved-adr`.
- list each B0 row with its current status and, if any, link to its
  study file.

**B1 (Important):**
- count by status only.

**B2 (Later):**
- count by status only.

**W2 (Wave 2 platform decisions):**
- whether the consolidated W2 document exists in `studies/decisions/`.

**W3 phases (Wave 3 scaffolding):**
- count of phases closed vs total; list any open / in-progress
  sub-phase rows by name.

**B0-S (Wave-S record-oriented capability):**
- count by status. List any B0-S row that is not at
  `resolved-adr` (Wave-S full gate depends on all seven being
  resolved-adr per ADR-0020 §"Full Wave-S gate").

**B3 (post-Wave-3 evolutionary lane):**
- count by status. Note that B3 has no closure gate per
  ADR-0049 §(c) — open entries are expected steady-state. Flag
  any B3 row at `draft` or `resolved-study` (not yet promoted
  to ADR) as in-flight work.

**Wave gates:**
- Wave 1 gate: `X of 7 B0 resolved-adr` (PASS if 7/7, else BLOCK).
- Wave 2 gate: PASS if the consolidated W2 document exists AND
  all five W2 rows are `resolved-adr`, else BLOCK.
- Wave 3 readiness: PASS if both Wave 1 and Wave 2 gates pass.
- Wave 3 completion: PASS if every Wave-3-phase row is closed
  (per the §"Wave 3 — Phases" table), else report the open
  phase count.
- Wave-S full gate: PASS if every B0-S row is `resolved-adr`
  per ADR-0020 §"Full Wave-S gate", else BLOCK. The decision
  log's §"Wave Gates" section carries the declared-met date
  (see Wave-S full-gate-declaration precedent in PR #116 /
  2026-05-30).

**Next recommended action:**

The Wave-1 era *"Recommended Next Sequence"* table in the
decision log is historical and superseded — per the log's own
header on that section. Today's triage is **demand-driven**:

1. **Operational unblocks first.** If a `PLACEHOLDER`
   substitution (`dq-{qa,prod}-PLACEHOLDER-*` identifiers in
   `engine/internal/env/{qa,prod}.go` + matching deploy
   overlays) or a missing CLI surface is blocking imminent
   operational work, resolve it before any B-row session.
2. **B3 lane** for capability extensions inside the three
   in-scope families (kind, capability mode, tooling) per
   ADR-0049 §(b). Surface a B-row when concrete demand
   arises; apply ADR-0049 §(a) four-condition eligibility
   filter (operator-side D0 ratification per CONTRIBUTING.md
   Flow 5 for borderline reads). Demand-driven — do NOT
   pre-enumerate B3 entries.
3. **Amendment** for proposals that modify (not extend) the
   decided shape of an existing ADR per ADR-0049 §(a)
   Amendment branch. Promotes via the standalone-amendment
   idiom (`adr-writing` A4) with the originating ADR's
   Status line updated per the ADR-0010 / ADR-0015 precedent
   (`accepted; **amended in part by ADR-NNNN** (...)`).
4. **Flow 6** (CONTRIBUTING.md Flow 6) for factual refreshes,
   tight clarifications, dated closure notes, and process-
   document maintenance whose substance is unchanged.

If multiple lanes have demand simultaneously, route operator-
authorized R4 scope collapses when the structural condition
holds (precedent ADR-0054, ADR-0055, ADR-0056, ADR-0057 —
contract-shaped choices ride with the implementation slice
in one PR).

**Consistency check:**

For each B-row marked `in-progress`, `draft`, or
`resolved-study` (across B0 / B1 / B2 / B0-S / B3), verify
the linked study file actually exists in
`studies/decisions/`. Flag any mismatch.

For each `resolved-adr` row, verify the linked ADR file
exists under `docs/adr/`. Flag any mismatch.

For each ADR carrying `Status: accepted (amends ADR-NNNN)`
in its metadata, verify the amended ADR's Status line carries
the reciprocal `**amended in part by ADR-NNNN**` marker per
the ADR-0010 / ADR-0015 precedent. Flag any missing
reciprocal marker as a Flow 6 housekeeping candidate (not a
blocker).

Print the report and stop. Do not modify any files.
