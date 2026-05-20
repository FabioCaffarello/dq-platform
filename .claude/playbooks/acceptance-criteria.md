<!-- path: .claude/playbooks/acceptance-criteria.md -->

# Acceptance Criteria — When a B0 Study is Ready to Close

These criteria are **binary and verifiable**. A study cannot move to
`resolved-study` in `studies/foundation/06-decision-log.md` until
**every** criterion passes.

Referenced by `.claude/playbooks/wave-1-session-loop.md` step 8 and
by `.claude/commands/critique.md`.

---

| # | Criterion | How to verify |
|---|---|---|
| **AC-1** | File starts with an HTML path-header comment (R6). | `head -1 <file>` is `<!-- path: ... -->`. |
| **AC-2** | Has all required sections in order: Context, Decision Drivers, Considered Options, Recommendation, Consequences, Open Questions, Promotion target. | Visual scan; section headings present. |
| **AC-3** | At least two options are considered (not just the recommendation). | Count subsections under "Considered Options". |
| **AC-4** | The Recommendation is grounded in either (a) a foundation doc, (b) a prior decision in `studies/decisions/`, or (c) explicitly marked "new contribution proposed here, requires review" (R5). | Inline citation or labeled marker present. |
| **AC-5** | No external vendor, sibling-team internal project, or prior-art name appears as justification (R5). Environment commodities (BigQuery, Kafka, Pub/Sub, OIDC, etc.) are allowed. | Grep / manual scan. |
| **AC-6** | Open Questions section is either empty, **or** every item is explicitly marked "out-of-scope for current cycle" with a one-line reason. | Visual scan. |
| **AC-7** | Promotion target line points to a concrete `docs/adr/<NNNN>-<slug>.md` filename. | Grep for `docs/adr/`. |
| **AC-8** | The study has survived at least one `/critique` round. | Session history; commit messages. |
| **AC-9** | All `blocking` findings from the most recent critique are either resolved in the study **or** explicitly deferred in Open Questions with a rationale. | Cross-reference latest critique vs. the study. |
| **AC-10** | The matching row in `studies/foundation/06-decision-log.md` is updated to `resolved-study` and links to the file. | Open the log and check. |

---

## Note on AC-9

A `blocking` finding that the author *disagrees with* is not
automatically deferred — the author must rebut it in the study
itself (typically in Decision Drivers or Recommendation) and let the
next critique surface whether the rebuttal holds.
