<!-- path: CONTRIBUTING.md -->

# Contributing to the DQ Platform

This guide is for contributors landing on the repository for the
first time and for returning contributors who need a refresher on
the four practical flows: adding a rule, running the local
end-to-end demo, opening a backlog decision, and closing a
Wave 3 session loop.

The companion documents:

- [`docs/glossary.md`](docs/glossary.md) — terms with
  codebase-specific meaning.
- [`docs/governance.md`](docs/governance.md) — review-time
  model: CODEOWNERS, the three review groups, contribution-time
  meta-flows (study→ADR, agent-contract sync).
- [`CLAUDE.md`](CLAUDE.md) — the authoritative operating
  contract for AI coding agents. Read it before any
  agent-driven session.

---

## Hard rules and platform principles

Canonical text lives in [`CLAUDE.md`](CLAUDE.md) §3 (R1–R8) and
§4 (P1–P6). One-line reminders:

- **R1** (production-code ban) **has lifted** — Wave 1 and Wave 2
  closed. Wave 3 produces real code, real config, real CI lanes.
- **R2** Do not invent requirements; record gaps as `TBD` or as
  explicit out-of-scope deferrals.
- **R3** Settled architectural decisions are final unless you
  spot a genuine inconsistency.
- **R4** One topic per session. Scope discipline is the only
  reason the loop stays reviewable.
- **R5** No external vendor, sibling-team, or prior-art names as
  justification. Environment commodities (GitHub, BigQuery, GCS,
  Pub/Sub, Kubernetes, OIDC, Prometheus, JSON Schema, etc.) are
  exempt.
- **R6** Every produced markdown file starts with an HTML
  path-header comment.
- **R7** Technical artifacts are in English (per
  [ADR-0011](docs/adr/0011-documentation-language.md)).
- **R8** Published artifacts (ADRs, `docs/*`, `CONTRIBUTING.md`)
  do not link backwards into `studies/`.

---

## Flow 1 — Add a rule for a new entity

The rules workspace (`rules/`) is editable by the
`@PLACEHOLDER-org/rules-authors` review group (see
[`docs/governance.md`](docs/governance.md) §2). The asymmetric
review model from [ADR-0001](docs/adr/0001-engine-rules-compatibility.md)
§C3 keeps `rules/_schema/` and `rules/_owners.yaml` under
platform-team review; per-entity rule YAMLs land under
domain-team review.

### 1.1 Create the rule YAML

Copy the shape of [`rules/customer.yaml`](rules/customer.yaml).
Pick an entity identifier matching `[A-Za-z0-9_.-]+` (the schema
mirror at [`rules/_schema/v1.schema.json`](rules/_schema/v1.schema.json)
enforces this pattern). Example for an `orders` entity:

```yaml
# path: rules/orders.yaml
version: 1
entity: orders
description: Order ingestion stream, hourly partitioned.
checks:
  - check_id: row_count_positive
    kind: row_count_positive
    description: Verifies the source table has at least one row.
```

The `version: 1` field is mandatory — the linter (`dq-lint`)
rejects any rule without it, per ADR-0001 Decision §1.

### 1.2 Add the `_owners.yaml` entry

The linter also requires an entry in
[`rules/_owners.yaml`](rules/_owners.yaml) for every entity that
has checks (per [ADR-0006](docs/adr/0006-alert-routing-contract.md)
§9 — "no alert without owner"). Add a sibling block:

```yaml
orders:
  owner: "@PLACEHOLDER-org/rules-authors"
  description: Order ingestion stream, hourly partitioned.
  channels:
    data_quality:
      - slack:#dq-orders
    operational:
      - email:oncall@example.com
```

The `owner` value uses the same `@org/team` syntax as
`/.github/CODEOWNERS` so the two surfaces can be cross-checked
(see [`docs/governance.md`](docs/governance.md) §2). At least
one alert category (`data_quality` or `operational`) must have a
non-empty channel list.

### 1.3 Validate locally

Run the linter:

```sh
make lint-rules
```

Behind the scenes this runs `dq-lint` against
`rules/_schema/v1.schema.json` and the `_owners.yaml` schema. The
linter reports per-file errors with line numbers; exit code is
non-zero on any failure. Fix issues at the source — the linter
is the first line of defense.

### 1.4 Open a PR

`rules/` edits land via PR. The playbook commits the branch
convention `wave-3/<phase>-<topic-slug>` for Wave 3 scaffolding
sessions; domain-team rule additions are outside Wave 3
scaffolding and may use a separate convention (e.g.,
`rules/<entity>-onboarding`). The branch convention for domain-
team rule edits is **a new contribution proposed here**; the
first such PR may revise it.
[`/.github/CODEOWNERS`](.github/CODEOWNERS) routes the per-entity
YAML edit to `@PLACEHOLDER-org/rules-authors`; the
`_owners.yaml` edit to `@PLACEHOLDER-org/platform-team` — both
reviews are required before merge.

---

## Flow 2 — Run `make demo-p6` locally

The end-to-end demo exercises the W2-3 §"C-W2-3.4" invariant —
*manifest publish → loader hash-short-circuit refresh →
execution write → operational alert publish* — entirely against
the local Compose substrate. No cloud sandbox access required.

### 2.1 Bring up the substrate

```sh
make up
```

This starts the Docker Compose stack: the BigQuery emulator,
the fake-gcs-server for object storage, and the Pub/Sub
emulator. The `--wait` flag in the target ensures every service
is healthy before the command returns.

If a service fails to start, run `docker compose ps` to inspect
the state; common causes are port conflicts on `8085` (Pub/Sub)
or `4443` (GCS emulator).

### 2.2 Run the demo

```sh
make demo-p6
```

The script
([`scripts/smoke/demo-p6.sh`](scripts/smoke/demo-p6.sh))
walks through nine steps: provisioning the local bucket and
dataset, linting `rules/`, publishing the manifest via
`dq-manifest`, starting `dq-engine` with a fast 2-second refresh
interval, waiting for `/readyz`, posting a trigger to
`/v1/trigger`, polling BigQuery for the terminal
`dq_executions` row, pulling the Pub/Sub topic for the
operational alert, and printing a green closure banner.

### 2.2.1 Other local test surfaces

The demo is the full-flow E2E lane. The platform also
exposes five other test tiers — unit-no-substrate,
integration-compose, integration-sandbox, smoke-substrate,
and config-validation. The full taxonomy + how-to-run-each
is in [`docs/dev/local-testing.md`](docs/dev/local-testing.md);
the authoritative posture is
[ADR-0034](docs/adr/0034-local-testing-strategy.md).

### 2.3 Tear down

```sh
make down
```

Stops the Compose stack and removes containers. BigQuery
datasets (`dq_fixture`, `dq_results_demo`) are intentionally
left in place so re-runs are idempotent; `make down` removes
the emulator container entirely, which clears those datasets.

### 2.4 Common troubleshooting

- **`/readyz` never comes up.** The engine waits for the
  manifest pointer to exist. Re-check that step 3 (manifest
  publish) succeeded — the script prints the published manifest
  hash on success.
- **`dq_executions` row never lands.** The trigger POST should
  return 202 with an `execution_id`; if it returns 4xx, the
  rule YAML probably fails the per-trigger schema check. Run
  `make lint-rules` first.
- **Pub/Sub message not received.** The local subscription is
  created at script start; if it was created against a stale
  topic from a previous demo, re-run after `make down && make up`.

---

## Flow 3 — Open a B-item

A B-item is a backlog decision tracked in the decision log at
`studies/foundation/06-decision-log.md`:

- **B0** blocks Wave 3 scaffolding (now closed; preserved for
  history).
- **B1** is "important — should be resolved before serious
  implementation".
- **B2** is "later — can be resolved as implementation reveals
  concrete needs".

### 3.1 When to open one

You discovered an architectural decision that:

- is not in the decision log yet, **and**
- a current scaffolding session needs resolved (B1 typically),
  or a future capability will depend on (B2).

Per the wave-3-sequencing ADR
([ADR-0013](docs/adr/0013-wave3-sequencing.md)) Consequence 9:
"a Wave 3 phase that needs an open B1 row resolves the B1 row
first, in a separate study session". Don't fold the decision
into the scaffolding session — R4 (one topic per session) makes
the trace unreadable.

### 3.2 How to add the row

Open the decision log and append the row to the appropriate
table (B1 or B2). Use the existing rows as templates: the
`Status` column starts as `open`, the `Key Question` is a
single sentence, the `Why It Matters` is one short paragraph,
the `Expected Output` names the document target.

### 3.3 How to resolve it

The Wave-1 loop still applies for any B-item resolution:

1. Open a fresh agent session, run `/clear`.
2. Ground via `/check-decision-backlog`.
3. Pick the B-item; verify upstreams are resolved.
4. Draft the study. For a B0 item, run `/resolve-b0 <slug>`.
   For B1 / B2 items, follow the same study shape (Context →
   Decision Drivers → Considered Options → Recommendation →
   Consequences → Open Questions → Promotion target) and write
   to `studies/decisions/<today>-<slug>.md` directly; the
   slash command's contract is B0-specific today and a
   generalized `/resolve-b` is a future option.
5. [H] Read end-to-end; ask the agent to re-frame if the
   question is wrong.
6. Run `/critique studies/decisions/<file>.md`.
7. Iterate up to two critique-revise rounds.
8. [H] Confirm Open Questions are either empty or marked
   out-of-scope.
9. Update the decision-log row to `resolved-study` with a link.
10. Commit on a feature branch; open a PR.

The full loop lives in
[`.claude/playbooks/wave-1-session-loop.md`](.claude/playbooks/wave-1-session-loop.md);
the acceptance criteria (AC-1 through AC-10) live in
[`.claude/playbooks/acceptance-criteria.md`](.claude/playbooks/acceptance-criteria.md).

---

## Flow 4 — Close a Wave 3 session loop

Wave 3 sessions scaffold one phase (or one sub-phase) per
session. The 10-step loop is documented in
[`.claude/playbooks/wave-3-session-loop.md`](.claude/playbooks/wave-3-session-loop.md);
the binary acceptance criteria (AC-W3-1 through AC-W3-10) live
in [`.claude/playbooks/wave-3-acceptance-criteria.md`](.claude/playbooks/wave-3-acceptance-criteria.md).

Short summary:

1. Fresh session, `/clear`.
2. Run `/check-decision-backlog`. Every upstream this scaffold
   cites must be `resolved-study` or `resolved-adr` — if any is
   `open`, stop and resolve it first (see Flow 3).
3. **[H]** Pick one phase / sub-phase from the Wave 3 — Phases
   table in the decision log. Stay inside one phase, one
   workspace, one slice.
4. Enter plan mode. The plan lists every B0/W2/B1 commitment
   cited, every file to create or modify, every AC-W3 row the
   scaffold satisfies, and explicit out-of-scope deferrals.
5. **[H]** Approve the plan, or ask the agent to re-scope.
6. Implement. Every produced markdown file gets a path header
   (R6). Code comments cite B0/W2 commitments only where
   load-bearing (AC-W3-3).
7. Self-verify each AC-W3 row to pass / fail / deferred-with-
   marker. Run the local gates (`make lint`, `make test-engine`,
   `make test-tools`, `make lint-rules`, plus `make validate-deploy`
   if `deploy/` is touched).
8. Run `/critique` against the scaffold. Address every
   `blocking` finding in the artifact. Max two rounds.
9. **[H]** Walk the diff. TODO / FIXME / `_TBD` markers must
   each carry an "out-of-scope for current cycle" reason or be
   resolved.
10. Open a PR via `gh pr create` on branch
    `wave-3/<phase>-<topic-slug>`. CI runs the workflows from
    Phase 2 (`lint`, `test`, `byte-equality`, `validate-deploy`).
    Once green and approved, merge via squash.

Direct-to-`main` commits ended at the Phase-4 sub-phase split
(commit `ee0d56f`); everything since W3-P4a lands via PR.

---

## Flow 5 — Close a post-Wave-3 session loop

Once Wave 3 closed, ongoing platform work flows through the
**post-Wave-3 evolutionary lane**: B2 follow-ups, B3 evolutionary
extensions per
[ADR-0049](docs/adr/0049-b3-evolutionary-launch.md), ADR
amendments, and ADR promotions. Flow 5 generalizes the PR-flow
contract of Flow 4 to that lane.

The 10-step loop for these sessions is documented in
[`.claude/playbooks/post-wave3-session-loop.md`](.claude/playbooks/post-wave3-session-loop.md);
it mirrors the Wave-1 loop in shape and inherits the same
acceptance criteria
([`.claude/playbooks/acceptance-criteria.md`](.claude/playbooks/acceptance-criteria.md)).

Short summary:

1. Fresh session, `/clear`.
2. Run `/check-decision-backlog`. For a B3 entry, additionally
   confirm the proposed work clears the
   [ADR-0049](docs/adr/0049-b3-evolutionary-launch.md) §(a)
   eligibility filter (all four conditions hold); for a B2
   entry, confirm the originating wave's gate is still met.
3. **[H]** Choose one B-item from the decision log. Stay inside
   one topic per session (R4).
4. Draft the study (if applicable) or plan the change. For
   non-study work, enter plan mode and list every cited
   upstream, every file to create or modify, every applicable
   gate, and explicit out-of-scope deferrals.
5. **[H]** Approve the plan or ask the agent to re-scope.
6. Implement. Path header on every markdown file (R6);
   English-only identifiers and comments (R7); R5 hygiene on
   every produced artifact.
7. Self-verify against the applicable acceptance criteria. Run
   the local gates that exist for the surface (`make lint`,
   `make test-engine`, `make test-tools`, `make lint-rules`,
   `make validate-deploy` if `deploy/` is touched, `make demo-p6`
   for end-to-end smoke).
8. Run `/critique` against the produced artifacts. Address every
   `blocking` finding in the artifact. Max two rounds per
   [`.claude/playbooks/wave-1-session-loop.md`](.claude/playbooks/wave-1-session-loop.md)
   step 7. Critique rounds are preserved per
   [ADR-0048](docs/adr/0048-critique-rounds-preservation.md)
   when the operator captures (operator-side; the agent emits
   stdout only).
9. **[H]** Walk the diff. TODO / FIXME / `_TBD` markers must
   each carry an out-of-scope-for-current-cycle reason or be
   resolved.
10. Open a PR via `gh pr create --base main`. The PR body lists
    the citation map (ADR / B-row references), the critique
    result (round counts, what was addressed), and a test plan
    (local gates run, manual verification steps, reviewer
    concurrence points). Once CI is green and the **[H]**
    reviewer approves, merge via the GitHub UI. **The agent
    never calls `gh pr merge`.**

### Branch naming for post-Wave-3 sessions

The Wave-3 convention `wave-3/<phase>-<topic-slug>` (Flow 4)
applied to scaffolding sessions only. Post-Wave-3 sessions have
not yet committed a single canonical convention; in practice
the slug prefixes operators have been passing are:

- `chore/` — tooling, CI, scripts, housekeeping that does not
  ship a study or an ADR;
- `feat/` — implementation slices that ship new artifacts
  committed by a prior ADR;
- `docs/decision/` — drafting or revising a study under
  `studies/decisions/`;
- `docs/adr/` — promoting a study to an ADR under `docs/adr/`.

These four slug prefixes are **new contribution proposed here,
requires review** (R5). They are recorded here so the harness
has a documented surface to defer to; reviewer concurrence is
the gate for fixing them as a canonical convention. The agent
does not invent new slug prefixes — it uses the slug provided
by the operator for the session, or asks if none is provided.

### Operator-side responsibilities

Two responsibilities sit explicitly with the operator (the
human contributor) and are not delegated to the agent:

- **ADR-number reservation under parallel PRs.** When two
  sessions might both promote a study around the same time,
  the operator tracks which `<NNNN>` numbers are reserved
  in-session and confirms when an agent's
  [`/promote-to-adr`](.claude/commands/promote-to-adr.md)
  proposes a number. The reservation is operator-side
  bookkeeping; the command makes the step explicit so it
  cannot be skipped silently.
- **Eligibility ratification for borderline B3-N readings.**
  When a B3-N session's eligibility under
  [ADR-0049](docs/adr/0049-b3-evolutionary-launch.md) §(a) is
  borderline (e.g., the family fit relies on an expansive
  reading of "adjacent tooling" or a similar clause), the
  operator ratifies the reading explicitly rather than
  absorbing it into the `/critique` output silently — the
  author and the reviewer share a session identity, and a
  critique-emitted eligibility ruling is structurally
  circular. The ratification is recorded in the round-2
  critique trailer (per
  [ADR-0048](docs/adr/0048-critique-rounds-preservation.md)
  preservation) and carries forward to the promoted ADR as a
  new-contribution marker per R5.

---

## What review will look like

Reviewers cite a rule (R1–R8), platform principle (P1–P6), or
acceptance criterion (AC-1…AC-10 for B-item studies, AC-W3-1…
AC-W3-10 for Wave 3 scaffolds) — never personal taste. The
full feedback protocol is in
[`.claude/playbooks/feedback-protocol.md`](.claude/playbooks/feedback-protocol.md).

CODEOWNERS routes review-required approvals automatically based
on which paths the PR touches; see
[`docs/governance.md`](docs/governance.md) §2 for the group
inventory and the path-rule pointer at
[`/.github/CODEOWNERS`](.github/CODEOWNERS).

---

## Commit conventions

Mirror the recent commit history (`git log --oneline -20`):

- `feat(<area>): <W3-Pxx> <topic>` — Wave 3 scaffolding sessions
  that ship runtime code (engine, tools, deploy). Example:
  `feat(engine): W3-P4c runner + failure-scope mapping`.
- `docs(<area>): <W3-Pxx> <topic>` — Wave 3 sessions that ship
  documentation. Example:
  `docs(governance): W3-P8b — publish CODEOWNERS + docs/governance.md`.
- `docs(studies): B<N>-<topic> — resolved-study` — closing a
  B-item study.
- `docs(adr): promote <slug> to ADR-NNNN` — study → ADR
  promotion.
- `chore: <change>` — repository hygiene.

Every commit body documents the citation map, the critique
result (blocking / important / minor counts; what was
addressed), and the AC self-check. Commits authored with agent
assistance carry the trailer:

    Co-authored-by: Claude Opus 4.7 (1M context) <noreply@anthropic.com>

---

## Where to read next

The companion-documents list at the top of this file covers the
fundamentals (glossary, governance, CLAUDE.md). The two
additional reading paths below are for contributors who already
have the fundamentals:

- **Architecture deep dive.** The foundation documents in
  `studies/foundation/` (`01-charter-and-principles.md` through
  `05-operational-discipline.md`, read in numbered order) carry
  the project framing. They live in `studies/` per their nature
  as foundational reasoning — published artifacts do not
  back-link into them, but contributors landing here for context
  may read them directly.
- **Decision state.** The decision log at
  `studies/foundation/06-decision-log.md` carries the Wave 3 —
  Phases table; scan it to see what's open, in-progress, or
  closed before picking the next session topic.
