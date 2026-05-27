<!-- path: docs/architecture/README.md -->

# DQ Platform — Architecture Overview

This directory is a **curated entry point** for new readers. It does
not replace the ADR record under [`../adr/`](../adr/) — it gives you
just enough mental model to read those ADRs in the right order.

Target reading time: **30 minutes** end to end.

---

## What DQ Platform is

DQ Platform is an internal Data Quality engine for curated data
assets. It evaluates trust in delivered data without coupling itself
to the production ingestion path: data flows independently; DQ
evaluates whether that data meets its declared quality contract and
emits structured alerts when it does not.

The platform speaks two modes — one for set-shaped data (BigQuery
tables, partitioned reads) and one for record-shaped data (Kafka
topics, windowed streams) — under a single rule grammar and a single
result store.

---

## The five mental-model anchors

These are the five concepts a reader needs to hold simultaneously
before any ADR will make full sense. Each anchor points at the file
that develops it in depth.

### 1. Multi-mode engine

The engine evaluates rules in one of two modes: `set` over a bounded
row set, or `record` over a bounded stream window. Mode is declared
on both the rule and the entity; the linter cross-checks them. This
is the architectural primitive that fans out everywhere else.
→ [`multi-mode-overview.md`](./multi-mode-overview.md)

### 2. DSL surface: entity → sources → checks → kinds → capability

A rule names an entity, declares its source(s), and lists checks.
Each check has a `kind` drawn from a closed catalog; the kind's
prefix (`set.*` / `record.*`) ties it to a capability. The DSL is
fully declarative — no SQL, no expressions, no escape hatches.
→ [`multi-mode-overview.md`](./multi-mode-overview.md) §"DSL surface"

### 3. Result store: `dq_executions` + `dq_check_results`

Every run lands two kinds of rows: one execution row (run-level
metadata, status, timing) and N check rows (per-check outcomes).
Both keyed by `execution_id` — a deterministic hash of the rule
version, entity, window, and trigger source. Set and record modes
write to the same tables.
→ [`request-flow.md`](./request-flow.md) §"Result write model"

### 4. Alert egress: Pub/Sub events routed by `_owners.yaml`

Failed or degraded checks emit events on Pub/Sub. Consumers fan
out to Slack, PagerDuty, or other destinations based on owner
metadata declared in `rules/_owners.yaml`. Egress is uniform
across modes; payload differs only in evidence shape.
→ [`request-flow.md`](./request-flow.md) §"Alert egress"

### 5. Workspace boundaries: `engine/` · `rules/` · `tools/` · `deploy/` · `docs/`

Five workspaces with five distinct identities. The engine runs
checks; the rules workspace authors them; tools support both;
deploy stands them up; docs explains the contract between them.
The boundaries are real even though everything lives in one
monorepo.
→ [`component-map.md`](./component-map.md)

---

## 30-minute reading path

1. **This README** (5 min) — you are here.
2. **[Multi-mode overview](./multi-mode-overview.md)** (8 min) — why two modes, what each one does, where the discriminator lives.
3. **[Component map](./component-map.md)** (7 min) — what the 11 engine packages and 5 tools each do, with one component diagram.
4. **[Request flow](./request-flow.md)** (8 min) — the eight-step lifecycle from trigger to alert, with per-mode sequence diagrams.
5. **[ADR reading order](./adr-reading-order.md)** (2 min) — the seven ADRs to read first, then the rest by cluster.

At the end, you have enough mental model to open any ADR and know
roughly where it fits.

---

## Where to go next

- **Whole truth, in order**: [`adr-reading-order.md`](./adr-reading-order.md) — load-bearing ADRs first, depth-on-demand grouped.
- **Operational concerns**: [`../runbooks/`](../runbooks/) — alert dedup debugging, entity onboarding, and other day-to-day playbooks.
- **Engineering setup**: [`../dev/`](../dev/) — local testing, schema migration.
- **Security posture**: [`../security/`](../security/) — evidence retention, manifest cryptographic posture.

If you find a contradiction between this overview and an ADR, **the
ADR wins**. Open an issue (or a PR amending this overview) so the
overview catches up.
