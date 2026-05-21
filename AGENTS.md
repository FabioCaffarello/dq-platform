<!-- path: AGENTS.md -->
<!-- audience: any AI coding agent that reads AGENTS.md by convention (Codex CLI, other tools) -->
<!-- status: living document, kept in sync with CLAUDE.md via /sync-agents -->

# Agents Operating Contract — DQ Platform

`AGENTS.md` is the cross-agent convention file. Tools that look for it
(Codex CLI and others) should treat this file as the entry point.

For the **full** operating contract, including project context, hard
rules, principles, and slash commands, read:

- **[`CLAUDE.md`](./CLAUDE.md)** — primary contract, always current.

The rules in `CLAUDE.md` apply to every AI agent in this repository,
not only to Claude Code. If you are a different tool, you are still
bound by those rules.

---

## Quick summary for first-time agents

You are in a monorepo that is being bootstrapped from zero. The
project is a long-lived Data Quality platform organized as five
workspaces (`engine/`, `rules/`, `tools/`, `deploy/`, `docs/`). The
project runs in three sequential waves:

- **Wave 1 — closed.** All seven `B0` decision-log items are at
  `resolved-study` in `studies/decisions/`. Gate met (7 of 7).
- **Wave 2 — closed.** The consolidated platform-decisions study
  (W2-1 through W2-5: Git host, multi-agent contract, Docker Compose
  substrate posture, documentation language, per-workspace tag
  conventions) is at `resolved-study`. Gate met (5 of 5).
- **Wave 3 — unblocked, not yet started.** Scaffolds every
  workspace, backed by the decisions resolved in Waves 1 and 2.
  R1 (no production code during Waves 1 and 2) no longer applies
  once a Wave 3 scaffolding session is explicitly opened by the
  project lead.

Read, in this order, before producing anything:

1. `CLAUDE.md`
2. Every playbook in `.claude/playbooks/` (operational protocol).
3. `studies/foundation/README.md`
4. Every numbered document under `studies/foundation/`, in order.

Then check `studies/foundation/06-decision-log.md` for the current
state of open decisions, and skim
`studies/decisions/2026-05-21-platform-decisions-wave2.md` for the
Wave 2 commitments that shape every Wave 3 scaffolding choice.

---

## What does not change between agents

- The hard rules R1–R8 in `CLAUDE.md`.
- The six platform principles P1–P6 in `CLAUDE.md`.
- The three-wave sequence.
- The path-header convention on every produced markdown file.
- English as the default for technical artifacts (confirmed in
  Wave 2). Portuguese is permitted for onboarding guides when the
  file opens with a one-line language marker.
- The R5 naming rule: **no produced artifact names internal
  projects of the organization, sibling-team systems, or external
  prior art by name as justification**. Commodity infrastructure
  the platform runs on (BigQuery, GCS, Pub/Sub, OIDC, Kafka,
  Prometheus, OpenTelemetry, Kubernetes, Go, Docker, JSON Schema,
  and equivalents) is exempt — these are environment, not borrowed
  ideas.

If a different agent has tool-specific conventions (its own
file-format expectations, for example), those are accommodated
**inside** the rules above, not by overriding them.

---

## When agents disagree

If two agents produce conflicting outputs for the same decision, the
human resolves the conflict and a single authoritative document lives
in `studies/decisions/`. Do not merge agent outputs silently.
