<!-- path: .claude/playbooks/wave-3-acceptance-criteria.md -->

# Acceptance Criteria — When a Wave 3 Scaffolding Unit is Done

These criteria are **binary and verifiable**. A scaffolding unit
cannot be committed until **every** criterion either passes or
is explicitly deferred with an "out-of-scope for current cycle"
marker.

Referenced by `.claude/playbooks/wave-3-session-loop.md` step 7
and by `/critique` (and by a future `/critique-scaffold` if the
shape diverges from Wave 1 critique enough to warrant a split).

The shape mirrors `.claude/playbooks/acceptance-criteria.md`
(Wave 1 AC-1 through AC-10), but the criteria are
scaffold-shaped, not decision-shaped.

---

| # | Criterion | How to verify |
|---|---|---|
| **AC-W3-1** | Every produced markdown file starts with an HTML path-header comment (R6). | `head -1 <file>` is `<!-- path: ... -->`. |
| **AC-W3-2** | Every produced source file (Go, YAML, Dockerfile, shell, JSON Schema, etc.) uses English-only identifiers and comments (R7). | Grep / manual scan. |
| **AC-W3-3** | Every **load-bearing** implementation cites the B0 or W2 commitment it implements, either in a leading code comment or in the commit message body. Routine code does not need citations. | Grep for `B0-`, `W2-`, `CC`, `C-W2-` labels in changed files; cross-reference with the commit message. |
| **AC-W3-4** | The scaffold stays inside its declared unit scope (R4). No incidental refactors of unrelated workspaces or surfaces. | `git diff --stat` matches the plan file's "files to create or modify" list. |
| **AC-W3-5** | The scaffold has survived at least one `/critique` round. | Session history; commit message body references the round. |
| **AC-W3-6** | TODO / FIXME / `_TBD` markers each carry an explicit deferral with a one-line reason (mirrors AC-6 for Open Questions). | `grep -nE 'TODO\|FIXME\|_TBD'` shows every match has a same-line or following-line reason. |
| **AC-W3-7** | Local build, lint, and test gates that exist for this surface at this point all pass. (If no gates yet exist for the surface, the AC row is vacuously satisfied — note this in the plan.) | Run the gates; record in the session. |
| **AC-W3-8** | If the unit lives under a Wave 2 §3.3 capability-matrix row marked **Yes**, that capability is exercised by at least one local test or one runnable command produced by this unit. **No** and **Partial** rows are exempt — they are sandbox or future-phase. | Manual cross-reference against the W2 §3.3 table. |
| **AC-W3-9** | If a B0 / W2 / B1 decision-log row named this surface area as an expected output, the row is updated to point at the produced scaffold path (or a follow-up phase row is added). | Open `studies/foundation/06-decision-log.md` after the change. |
| **AC-W3-10** | R5 hygiene: no produced code, comment, commit message, or documentation file names a sibling-team / internal-project / prior-art system as justification. Commodity infrastructure (BigQuery, GCS, Pub/Sub, OIDC, Kafka, Prometheus, OpenTelemetry, Kubernetes, Go, Docker, JSON Schema, and equivalents) is exempt as environment. | Grep / manual scan. |

---

## Note on AC-W3-3 (when to cite)

A citation in a code comment is for **load-bearing** references
— the places where a future reader would otherwise wonder *why*
a constant, a schema field, an enum value, or an invariant has
the exact shape it does. Examples:

- The five-input `execution_id` formula (B0-2 CC1) — cite, because
  the input order is hash-invariant.
- The `dq_executions` append-only constraint (B0-3 CC1) — cite at
  the write path, because a future maintainer might "optimize"
  by adding an UPDATE.
- The pipe-character ban in tag prefixes (W2-5 C-W2-5.3) — cite
  at the tag-validation site, because the prohibition is not
  intuitive.

Routine code — a helper function, a struct, a config getter —
does not need citations. The commit message body carries the
full citation map for the unit.

## Note on AC-W3-5 (critique rebuttal)

A scaffolding unit whose author **disagrees** with a critique
finding is rebuttable in the same way as a Wave 1 study: rebut
in the artifact (a code comment, a README note, or the
session-summary markdown), let the next critique round surface
whether the rebuttal holds. Maximum two rounds — after that, the
unit ships as-is and the remaining doubt becomes an explicit
TODO with a deferral marker (which then has to satisfy AC-W3-6).

## Note on AC-W3-7 (vacuous satisfaction is fine)

Early Wave 3 phases will scaffold surfaces that have no build or
test gates yet. AC-W3-7 is vacuously satisfied in those cases —
record this in the plan file ("no gates exist for this surface;
AC-W3-7 vacuous") so it is auditable. As gates land, later units
that touch the same surface must pass them.
