<!-- path: .claude/playbooks/wave-1-session-loop.md -->

# Wave 1 — Session Loop

The canonical 10-step loop for resolving one B0 decision. Each step
is either an **agent action** or a **human decision point [H]**.
Drawn from `KICKOFF.md` Sessions 2–8, formalized here so the agent
can be pointed at it as required reading.

---

## The loop

1. **Open** a fresh Claude Code session in the repository root. Run
   `/clear` to drop any prior context.

2. **Ground.** Run `/check-decision-backlog` to see the current
   status of every B0 row.

3. **[H] Choose one B0.** Pick exactly one item whose dependencies
   (per the "Recommended Next Sequence" section in
   `studies/foundation/06-decision-log.md`) are already resolved. If
   none are unblocked, stop — work on dependencies first.

4. **Draft.** Run `/resolve-b0 <slug>`. The agent writes
   `studies/decisions/<today>-<slug>.md`. Wait until it stops; do
   not nudge mid-draft.

5. **[H] Read end-to-end.** Open the draft. Ask: does it frame the
   right question, in the right scope?
   - If yes → go to step 6.
   - If no → ask the agent to re-frame and rewrite. **Do not edit
     the draft yourself** — let the agent regenerate so the
     reasoning stays consistent.

6. **Critique.** Run `/critique studies/decisions/<file>.md`. Read
   the findings.

   **Preserve the round** per
   [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md):

   - Capture the `/critique` stdout to a file at
     `studies/critiques/<today>-<slug>-critique-<N>.md`, where
     `<slug>` matches the parent study's slug exactly and `<N>`
     is the round number (1-indexed; this round is `1`, the
     optional second round in step 7 is `2`).
   - The capture is operator-side — `/critique` itself stays
     stdout-pure. Redirect
     (`claude /critique ... > studies/critiques/...md`) or
     copy-paste from the session transcript.
   - Wrap the captured findings in the file shape ADR-0048
     §"What" commits: path-header HTML comment (R6), H1 title
     `# B<N>-<M> — Critique Round <K>`, `## Metadata` block
     (target study, round, date, preservation status, closing
     commit hash — the latter filled in at step 10),
     `## Critique Output` containing the verbatim stdout inside
     a fenced ```text block, and `## Operator Response` trailer
     with one entry per finding using the disposition vocabulary
     from ADR-0048 §"What" (`applied as recommended` /
     `applied with variation` / `deferred to Open Questions` /
     `deferred to a future round` / `accepted as-is` /
     `rejected`).
   - Commit the critique file as an intermediate commit, before
     any revision lands:
     ```
     git add studies/critiques/
     git commit -m "docs(decision): B<N>-<M> — critique round <K> findings"
     ```
   - The sequence is draft → **critique (commit)** → revision
     (commit at step 7).

7. **Iterate.** Ask the agent to revise the **original study** (not
   the critique) to address `blocking` findings. Commit the
   revision as its own commit. Re-run `/critique` if needed —
   **preserve round 2 per step 6's contract** (capture to
   `studies/critiques/<today>-<slug>-critique-2.md`, intermediate
   commit, operator-response trailer). **Maximum two
   critique-revise rounds.** After that, accept the document as
   the best Wave 1 can do and let remaining doubts surface in
   Open Questions.

8. **[H] Check Open Questions and Critique rounds.** Two gates must
   pass before the study can move to `resolved-study`:

   - **Open Questions** — either the section is empty, **or**
     every item is explicitly marked "out-of-scope for current
     cycle" with a one-line reason.
   - **Critique rounds bullet** — the study's Metadata block has
     a `Critique rounds:` bullet listing each round's disposition
     per [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)
     §"Skip" grammar: rounds in chronological order; each entry
     `preserved (<filepath>)` or `skipped (<reason>)`; entries
     comma-separated; the bullet may span multiple lines. Absence
     of the bullet is itself a `blocking` finding under ADR-0048.

   If either gate fails, the study is not ready. Continue
   iterating or accept it as is and document why.

9. **Update the log.** Edit
   `studies/foundation/06-decision-log.md` — change the row for this
   B0 to `resolved-study` and add the link to the file written in
   step 4.

10. **Commit.** This is the **close commit** — log update only.
    Intermediate critique-and-revision commits already landed in
    steps 6 and 7.
    ```
    git add studies/
    git commit -m "docs(decision): resolve B0-N — <topic>"
    ```

---

## When to abort the session (no commit)

Stop the loop **without committing** if any of the following
appears:

- **R1 violation.** The agent wrote production code (Go, YAML
  rules, Dockerfile, CI). Roll back the draft, re-prompt.
- **R5 violation.** The draft cites a vendor, sibling-team internal
  project, or external prior art by name as justification. Roll
  back the draft, re-prompt.
- **R6 violation.** The draft is missing its path-header comment.
  Fix the draft (or re-prompt).
- **Unresolved Open Questions at step 8** and you are not willing
  to defer them. Better to leave the B0 as `in-progress` than
  commit a half-baked decision.

---

## Why the [H] decision points exist

Steps 3, 5, and 8 are the points where the human's judgment is
load-bearing. Letting the agent pick the next B0, re-frame its own
draft, or self-approve Open Questions defeats the purpose of the
critique cycle. These three checks are how Wave 1 stays honest.
