---
description: Draft a decision document for one B0 item in studies/decisions/.
argument-hint: <slug>
---

<!-- path: .claude/commands/resolve-b0.md -->

You are resolving one **blocking decision (B0)** for the DQ Platform.

Argument (slug): `$ARGUMENTS`

Before producing anything, re-ground yourself:

1. Read `CLAUDE.md` §3 (hard rules R1–R8) and §4 (principles P1–P6).
2. Read `.claude/playbooks/acceptance-criteria.md` — the study you
   write must meet every criterion before it can be closed.
3. Read `studies/foundation/06-decision-log.md` and locate the B0 row
   whose topic matches the slug. If no row matches, stop and ask the
   operator before guessing.
4. Read every foundation document referenced by that B0 row's
   "Why It Matters" and "Key Question" columns.

Then write a new file at:

    studies/decisions/<today>-<slug>.md

where `<today>` is the current date in `YYYY-MM-DD`.

The file must contain, in this order:

- HTML path-header comment (R6).
- Title — short, names the decision.
- **Context** — what problem the decision resolves, with references
  to the foundation docs that motivate it.
- **Decision Drivers** — the constraints the decision must satisfy.
- **Considered Options** — at least two, with trade-offs for each.
- **Recommendation** — chosen option, with justification.
- **Consequences** — what this commits the platform to.
- **Open Questions** — empty, or every item explicitly marked
  "out-of-scope for current cycle".
- **Promotion target** — concrete `docs/adr/<NNNN>-<slug>.md`
  filename to be created during Wave 3.

Hard guardrails:

- **R1:** no Go, no YAML rules, no Dockerfile, no CI. Code-shaped
  illustrations only inside fenced markdown blocks.
- **R5:** do not name vendors, sibling-team internal projects, or
  external prior art as justification. Environment commodities
  (BigQuery, Kafka, Pub/Sub, OIDC, Prometheus, OpenTelemetry,
  Kubernetes, Go, Docker, slog, JSON Schema, etc.) are allowed.
- **R6:** path header on every produced markdown file.

After writing, stop. Do **not** run `/critique` yourself — the
operator runs it next, per
`.claude/playbooks/wave-1-session-loop.md` step 6.
