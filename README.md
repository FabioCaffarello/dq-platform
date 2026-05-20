<!-- path: README.md -->

# DQ Platform

Internal data-quality engine for curated BigQuery assets, evolving
toward streaming validation over Kafka.

> **Status: Wave 1 of 3 — Foundation.**
> No production code yet. This is intentional — see
> [`KICKOFF.md`](./KICKOFF.md) for why.

---

## Three doors

| You are… | Start with |
|---|---|
| **operating a Claude Code session** in this repo | [`KICKOFF.md`](./KICKOFF.md) |
| **contributing to a decision** | [`CONTRIBUTING.md`](./CONTRIBUTING.md) |
| **looking for deep context** | [`studies/foundation/README.md`](./studies/foundation/README.md) (read in numerical order) |

---

## Repository today

```
CLAUDE.md            agent operating contract
AGENTS.md            cross-agent entry point
KICKOFF.md           human operator guide
.claude/             slash commands + playbooks
studies/             reasoning: foundation + decisions
```

## Repository eventually (Wave 3 — not yet created)

| Workspace | Will hold |
|---|---|
| `engine/` | Go runtime, DSL schema, scheduler, reporting, alerting |
| `rules/`  | YAML rule specifications by entity, owner metadata |
| `tools/`  | auxiliary CLIs (linter, dry-run, manifest publisher) |
| `deploy/` | Kubernetes manifests, infrastructure configuration |
| `docs/`   | architecture, ADRs, glossary, governance |

These directories do not exist yet. They appear in Wave 3, backed by
the decisions resolved in Wave 1 and Wave 2.

---

_Started: 2026-05-20. This README is intentionally thin — Wave 3
Session A will rewrite it._
