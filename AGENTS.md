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

You are in a long-lived Data Quality monorepo organized as five
workspaces (`engine/`, `rules/`, `tools/`, `deploy/`, `docs/`). The
five product workspaces carry real content. Work is organized into
three sequential waves (now closed), one wave-shaped capability
extension (Wave-S), and one demand-driven evolutionary lane (B3) —
Wave-S and B3 are structural peers of the original waves, not a
"Wave 4" or "Wave 5".

- **Wave 1 — closed.** Seven `B0` blocking decisions closed
  2026-05-21 and promoted to ADRs 0001–0007.
- **Wave 2 — closed.** Consolidated platform decisions (`W2-1`…
  `W2-5`) closed 2026-05-21 and promoted to ADRs 0008–0012 + 0015–0019.
- **Wave 3 — closed.** Workspace scaffolding (`W3-P0`…`W3-P8`) closed
  2026-05-23.
- **Wave-S — partial gate met.** Launched 2026-05-23 via ADR-0020;
  the record-mode partial gate (B0-S1 / B0-S2 / B0-S3) was met
  2026-05-24, unblocking record-mode code. Full-gate criteria remain
  in ADR-0020.
- **B3 — open.** Demand-driven evolutionary lane opened 2026-05-29
  via ADR-0049, restricted to kind / capability-mode / tooling
  extensions and filtered by the four-condition eligibility test in
  ADR-0049 §(a). No closure gate — stays open across the platform's
  operating life.

R1 (no production code during Waves 1 and 2) is now historical:
Waves 1–2 closed, and Wave 3 onward authorizes production code under
the decisions resolved in the earlier waves.

Read, in this order, before producing anything:

1. `CLAUDE.md` — primary contract, including §6's session reading
   router (committed by
   [ADR-0052](./docs/adr/0052-session-reading-router.md)) that
   maps your session type to the minimal playbook subset.
2. The playbook subset declared by `CLAUDE.md` §6.2 for your
   session type — six types: B2 follow-up, B3 entry, ADR
   amendment, ADR promotion, Flow 6 process edit, implementation
   slice landing under a closed B-row. Apply §6.3 (default-up) if
   the classification is borderline; apply §6.4 (output-artifact
   tie-breaker) if two rows could apply; skim §6.5
   (historical-skim set: `wave-1-session-loop.md`,
   `wave-3-session-loop.md`) only when explicitly reading prior
   sessions for shape-reference context. The playbook inventory
   is in §6.6.
3. `CONTRIBUTING.md` Flow 5 — the upstream authority for PR-flow
   in the post-Wave-3 lane. (`CONTRIBUTING.md` as a whole sits in
   `CLAUDE.md` §6.1's always-on floor; Flow 5 is the load-bearing
   PR-flow for post-Wave-3 sessions; Flow 6 is the
   operator-authorized direct-edit lane.)
4. `studies/foundation/README.md`
5. Every numbered document under `studies/foundation/`, in order.

Then check `studies/foundation/06-decision-log.md` for the current
state of every B-row and ADR, including Wave-S partial / full gate
status and B3 lane activity.

R1–R8 (CLAUDE.md §3) and P1–P6 (CLAUDE.md §4) are read by every
session unconditionally — the router (CLAUDE.md §6) governs only
the playbook layer and can never narrow rule or principle
reading.

---

## What does not change between agents

- The hard rules R1–R8 in `CLAUDE.md`.
- The six platform principles P1–P6 in `CLAUDE.md`.
- The waves taxonomy: three sequential waves (closed) plus Wave-S
  (wave-shaped capability extension) and B3 (open evolutionary lane)
  as structural peers.
- The path-header convention on every produced markdown file
  (R6 applies to markdown only).
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
