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

7. **Iterate.** Ask the agent to revise the **original study** (not
   the critique) to address `blocking` findings. Re-run `/critique`
   if needed. **Maximum two critique-revise rounds.** After that,
   accept the document as the best Wave 1 can do and let remaining
   doubts surface in Open Questions.

8. **[H] Check Open Questions.** Either:
   - the Open Questions section is empty, **or**
   - every item is explicitly marked "out-of-scope for current
     cycle" with a one-line reason.

   If neither, the study is not ready. Continue iterating or accept
   it as is and document why.

9. **Update the log.** Edit
   `studies/foundation/06-decision-log.md` — change the row for this
   B0 to `resolved-study` and add the link to the file written in
   step 4.

10. **Commit.**
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
