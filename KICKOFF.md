<!-- path: KICKOFF.md -->
<!-- audience: the human operator running Claude Code sessions in this repository -->
<!-- status: operational guide, not a rule. Adapt as you learn what works. -->

# Kickoff — Operating Claude Code on This Repository

This guide is for the **human operator**, not for the agent. It tells
you the exact sequence to run, what to expect, and what to push back
on. The agent reads `CLAUDE.md`. You read this.

Estimated time to complete Wave 1: about 7 focused sessions of
30–90 minutes each. Estimated time to complete Wave 2: one session.
Wave 3 then proceeds in parallel tracks across the five workspaces.

---

## Session 0 — Prepare the workspace (you, no agent)

Before opening Claude Code, **clean up the private reasoning
material** that this bootstrap replaces.

The bootstrap kit ships its own `studies/foundation/` with six
documents written from scratch. The previously-extracted material
(the original `seed/` and the original `studies/foundation/`
content) must not be committed and should ideally be removed from
disk so the agent does not accidentally read it.

```bash
cd ~/projects/dq-platform

# Move the original private material outside the repo (recommended):
mv seed ~/dq-platform-private-archive/seed
mv studies ~/dq-platform-private-archive/studies-original

# Now unzip the bootstrap kit into the cleaned workspace:
unzip /path/to/dq-platform-bootstrap-kit.zip

# Verify the new layout:
ls -la
```

You should see:

```
.git/                .gitignore           AGENTS.md
CLAUDE.md            KICKOFF.md           .claude/
studies/             dq-engine/           dq-rules/
```

The new `studies/` directory contains the foundation documents and
an empty `decisions/` folder.

The empty `dq-engine/` and `dq-rules/` from the previous extraction
can be removed — Wave 3 will create the workspaces under the new
names (`engine/`, `rules/`, etc.):

```bash
rmdir dq-engine dq-rules    # they are empty, this is safe
```

Initial commit:

```bash
git add -A
git status            # verify nothing unexpected is staged
git commit -m "chore: bootstrap workspace with CLAUDE.md, AGENTS.md, foundation, slash commands"
```

If you accidentally moved a `.DS_Store` or other system file, the
`.gitignore` should keep it out. Re-check `git status` to be sure.

---

## Session 1 — Calibration

This session is **diagnostic only**. Do not produce artifacts. The
goal is to confirm the agent reads the contract correctly.

Open Claude Code in the repository root. First prompt:

> Read `CLAUDE.md`, then `AGENTS.md`, then `studies/foundation/README.md`,
> then every numbered document under `studies/foundation/`, in order.
> After reading, produce:
>
> 1. A summary in up to 12 bullets of what you understand the project
>    is and where it is in the wave sequence.
> 2. A list of every assumption you would have made if you had not
>    read this material.
> 3. A list of every internal inconsistency or ambiguity you spot
>    across these documents.
> 4. The current state of the decision log (without running the slash
>    command yet — just from reading).
>
> Do not produce any new files. Do not edit anything. This is a
> read-only calibration.

**What to look for in the response:**

- ✅ The summary correctly identifies the three waves and that Wave 1
  is in progress.
- ✅ It names the seven B0 items, or at least most of them.
- ✅ It correctly identifies the five workspaces (`engine/`,
  `rules/`, `tools/`, `deploy/`, `docs/`).
- ✅ It does not propose to start coding.
- ✅ It does not reference any external system, vendor, or prior art
  by name.
- ❌ If it tries to write a file, the contract failed — revise
  `CLAUDE.md` and rerun.

When the calibration is clean, commit any `CLAUDE.md` revisions
made during it:

```bash
git add CLAUDE.md
git commit -m "docs: calibrate agent contract after session-1 dry run"
```

---

## Sessions 2–8 — Resolve the seven B0 items

**One B0 per session.** Start a fresh conversation for each one
(`/clear` in Claude Code). This keeps context narrow and prevents
decisions from contaminating each other.

The recommended order, from `06-decision-log.md`:

1. **B0-1** — engine ↔ rules compatibility (foundation for the
   others).
2. **B0-5** — manifest publication semantics (defines what a
   "ruleset" is at runtime).
3. **B0-2** — run identity and idempotency.
4. **B0-3** — result write model (depends on B0-2).
5. **B0-4** — failure scope (depends on B0-2 and B0-3).
6. **B0-7** — loader and scheduler failure semantics (depends on
   B0-1, B0-5, B0-4).
7. **B0-6** — alert routing contract (depends on B0-4).

For each session:

```
/resolve-b0 <slug>
```

For example:

```
/resolve-b0 engine-rules-compatibility-model
/resolve-b0 manifest-publication-semantics
/resolve-b0 run-identity-and-idempotency
```

The slugs you choose become the filenames. Pick them carefully — they
will be referenced for years.

**After the agent writes the draft:**

```
/critique studies/decisions/<the-file-it-just-wrote>.md
```

Read the critique. If `blocking` issues are present, ask the agent
to revise the original (not the critique). Iterate up to two more
times. After that, accept the document as the best Wave 1 can do and
let remaining doubts surface in its Open Questions section.

**Close each session with a commit:**

```bash
git add studies/
git commit -m "docs(decision): resolve B0-N — <topic>"
```

Run `/check-decision-backlog` at the start of each session to ground
yourself in what is left.

---

## Session 9 — Wave 2: platform decisions

Single session, single document. Open Claude Code and run:

> Produce a consolidated platform decisions document at
> `studies/decisions/<today>-platform-decisions.md` that resolves
> every W2 item listed in
> `studies/foundation/06-decision-log.md`:
>
> 1. Git host choice (affects every CI artifact).
> 2. Multi-agent contract for `.claude/`, `.codex/`, and `AGENTS.md`.
> 3. Docker Compose local scope: which services are emulated, which
>    require sandbox access.
> 4. Documentation language policy.
> 5. Per-workspace tag prefix conventions.
>
> For each item, follow the same structure as the B0 studies
> (Context, Decision Drivers, Considered Options, Recommendation,
> Consequences, Open Questions). One section per item, in one file.

Then critique it. Then commit.

---

## Wave gate — before starting Wave 3

Run `/check-decision-backlog`. The output should show:

- Every B0 row: `resolved-study` (✅ green check).
- The W2 consolidated decisions document present.
- The workspace directories (`engine/`, `rules/`, etc.) still absent
  or empty.

If any B0 is still `open` or `in-progress`, **do not** start Wave 3.
The gate exists for a reason. Resolving Wave 1 partially and
starting Wave 3 anyway is the failure mode the foundation documents
specifically warn against.

When the gate passes, commit a milestone marker:

```bash
git commit --allow-empty -m "milestone: Wave 1 + Wave 2 complete, ready for scaffolding"
```

---

## Wave 3 — Scaffolding (overview only; details earned by Wave 1)

Wave 3 is much more concrete because Wave 1 and 2 outputs tell you
exactly what each file should say. Approximate session breakdown:

### Top-level scaffolding

- Session A: root files (`README.md`, `CONTRIBUTING.md`, `CODEOWNERS`,
  `CHANGELOG.md`, `SECURITY.md`, `LICENSE`, `.editorconfig`,
  `.gitattributes`, `Makefile`, `go.work`, `docker-compose.yml`).

### `docs/` workspace

- Session B: `docs/architecture/` (overview, data-flow, components,
  storage, deployment) — derived from `04-system-architecture.md`.
- Session C: `docs/adr/` — promote every B0 study via
  `/promote-to-adr`.
- Session D: `docs/dsl/` (specification, check-catalog, cookbook,
  migration-guide, versioning).
- Session E: `docs/operations/` (runbooks, observability, release,
  onboarding-entity) — derived from `05-operational-discipline.md`.
- Session F: `docs/development/` (getting-started, testing,
  coding-standards, adding-check-type, branching).

### `engine/` workspace

- Session G: `engine/` Go module skeleton (`go.mod`, `cmd/`,
  `internal/` package READMEs and `doc.go` files, `Dockerfile`).
- Session H: schema artifact at
  `engine/internal/dsl/schema/v1.schema.json`.

### `rules/` workspace

- Session I: `rules/` structure (`_schema/v1.schema.json` mirror,
  `_owners.yaml` template, `_examples/` with one YAML per check
  type, `entities/` with one fully documented example).
- Session J: `rules/` documentation (`README.md`, `CONTRIBUTING.md`,
  contributor-facing tutorials).

### `tools/` workspace

- Session K: `tools/lint/` skeleton (Go module, CLI structure,
  stub commands).
- Session L: `tools/dryrun/` and `tools/publisher/` skeletons.

### `deploy/` workspace

- Session M: `deploy/k8s/` manifests (stubs for engine deployment,
  service accounts, OIDC configuration).

### CI

- Session N: CI templates (`.gitlab-ci.yml` or `.github/workflows/`,
  depending on Wave 2), with path-filtered jobs per workspace.

### Agent configuration

- Session O: workspace-specific `.claude/CLAUDE.md` files for each
  workspace if useful, and `.codex/` setup based on the Wave 2
  decision.

Each session commits as it closes. Use `/critique` liberally between
sessions.

---

## Discipline rules for the operator

A few patterns to defend against:

- **Resist scope creep within a session.** If the agent says "while
  I am here, I noticed X" — note it for a future session; do not let
  it expand the current one.
- **Re-read the document at the end of every session before
  committing.** Agents sometimes produce text that reads correctly
  but says the wrong thing in detail. Five minutes of human review
  saves hours later.
- **Watch for external-reference leaks.** Any time the agent's
  output mentions a vendor name, an internal project name, a known
  pattern by its proper name — push back. The contract is that
  every concept is described in our own terms.
- **Do not skip critique.** The whole reason this foundation cycle
  is rigorous is that critique is built in. Do not consider a study
  resolved until it has survived at least one critique pass.
- **Use plan mode for anything beyond one file.** In Claude Code,
  Shift+Tab activates plan mode. Wave 3 sessions especially benefit
  from this.
- **Keep commits small and named by output.** `docs(decision): ...`,
  `docs(adr): ...`, `chore(ci): ...`, `feat(engine): ...` (Wave 3
  only).

---

## When to update `CLAUDE.md`

`CLAUDE.md` is a living document. Update it when:

- A wave completes (mark Wave 1 as done, etc.).
- A rule turns out to be wrong or insufficient (the agent kept
  violating it the same way → the rule is the problem, not the
  agent).
- A new tool or workflow becomes part of the project.

Do **not** update `CLAUDE.md` to bend toward what the agent prefers.
That defeats the purpose.

---

## Sanity test: are we doing this right?

At any point, ask yourself:

1. Are `engine/`, `rules/`, `tools/`, `deploy/`, `docs/` still
   empty or absent? (✅ expected during Wave 1 and 2)
2. Does every artifact in `studies/decisions/` have a `Promotion
   target` line? (✅ required)
3. Has every B0 had a `/critique` round? (✅ required)
4. Did any produced artifact mention an external system, vendor, or
   prior art by name? (❌ if yes, fix it before committing)
5. Did the project skip Wave 2? (❌ if yes, fix it before
   continuing)

If all five answers are healthy, you are operating this repository
the way it was designed to be operated.
