<!-- path: .claude/plans/w3-p4e-http-trigger-handler.md -->

# W3-P4e — HTTP Trigger Handler (plan)

> **Renumbered from W3-P6b → W3-P4e on 2026-05-22.** P4e closes
> **Phase 4 (engine runtime)** — ADR-0013 §"Phase 4 — Engine
> runtime scaffold" enumerates loader, runner, result write,
> failure scope, and orphan detection; the runner is unexercisable
> without a trigger surface, so the HTTP trigger handler belongs
> inside Phase 4 as its closing sub-phase. **Phase 6 remains
> "first onboarded entity end-to-end"** exactly as ADR-0013
> specifies — one real `_owners.yaml`, one real published
> manifest, exercising Phases 3–5 against real content. The
> handler is part of the runtime that Phase 6 will exercise, not
> the entity-onboarding work itself.
>
> The decision-log row and the engine README still carry the old
> P6b labelling; both are listed as follow-ups below. ADR-0013
> itself has not been edited — its phasing narrative is intact
> (verified 2026-05-22).

---

## Status

**Plan body deferred.** This file is opened by the renumbering
session on 2026-05-22 to record the corrected sub-phase label.
The actual implementation plan (B0 / W2 citations, files
created, AC-W3 mapping, deferred items) is written when the
P4e implementation session opens — that session begins by
filling in this file under plan mode per
`.claude/playbooks/wave-3-session-loop.md` step 4.

---

## Upstream contract

The handler's contract is captured by the study produced in
the same session that created this file:

- `studies/decisions/2026-05-22-trigger-handler-contract.md` —
  four micro-decisions (MD-1 hydration timing, MD-2 strict-decoder
  posture, MD-3 response shape, MD-4 health endpoint semantics).
  Provisional promotion target: `docs/adr/0014-trigger-handler-contract.md`.

P4e implementation cannot begin until that study is at
`resolved-adr` (per `.claude/playbooks/wave-3-session-loop.md`
step 2 — every upstream commitment cited by the scaffold must
be at `resolved-study` or `resolved-adr`).

---

## Open follow-ups (separate sessions per R4)

The renumbering session deliberately did **not** edit the
following references; they belong with the P4e implementation
session or with the session that promotes the contract study:

- **`studies/foundation/06-decision-log.md` row 128.** Describes
  the W3-P6 split as `W3-P6a / W3-P6b / W3-P6c / W3-P6d` and
  names the HTTP trigger handler inside the P6 split. Action:
  remove the HTTP trigger handler from the P6 split; add a
  new W3-P4e row to the Phase 4 split section.
- **`studies/foundation/06-decision-log.md` row 130.** The
  `W3-P6b | HTTP trigger handler — exposes POST /v1/trigger,
  dispatches to runner per ADR-0002. Depends on W3-P6a.` row.
  Action: rename to `W3-P4e`, fix the dependency (depends on
  W3-P4a/b/c/d, not W3-P6a), and move it to the Phase 4 block.
- **`engine/README.md` line 162.** `- **W3-P6** — first
  onboarded entity end-to-end; HTTP / gRPC trigger handler.`
  conflates the trigger handler with Phase 6. Action: split into
  two README entries — `W3-P4e` (HTTP trigger handler) and
  `W3-P6` (first onboarded entity end-to-end, no trigger handler
  scope). Drop the `gRPC` mention unless ADR-0014 commits a gRPC
  variant (MD-1..MD-4 do not — gRPC is explicitly out of scope
  for the contract study).

These three follow-ups do not block the contract study's
critique pass. They are sequenced with the P4e implementation
PR so the decision-log update and the README update land
together with the engine code.

---

## When this file is filled in

The implementation session that opens this file is responsible
for the items listed in `.claude/playbooks/wave-3-session-loop.md`
step 4:

- Every B0 / W2 commitment the scaffold implements, by exact
  label (e.g., `ADR-0002 §3`, `ADR-0007 §1`).
- Every file to create or modify, with full path.
- Every AC-W3 row from
  `.claude/playbooks/wave-3-acceptance-criteria.md` the scaffold
  must satisfy.
- Deferred items with an explicit "out-of-scope for current
  cycle" reason per item.

Until then, this file's job is only to (a) park the corrected
sub-phase label and (b) keep the follow-ups visible so they are
not lost between sessions.
