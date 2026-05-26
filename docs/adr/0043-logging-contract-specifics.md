<!-- path: docs/adr/0043-logging-contract-specifics.md -->

# ADR-0043 — Logging Contract Specifics

- **Status:** accepted
- **Date:** 2026-05-26

---

## Context

Foundation 04 §"PAT-5 — Modular logging contract"
promised a per-package logging override mechanism
via an environment variable:

```
DQ_LOG_LEVELS="root:INFO,engine.compilers:DEBUG,engine.scheduler:WARN"
```

The promise: global default level plus per-package
overrides, expressed as a small layer over Go's
`slog` standard-library package.

What the platform ships today is narrower than that
promise:

- `engine/internal/env/config.go:27` declares a
  typed `LogLevel` enum (`debug` / `info` /
  `warn` / `error`).
- `engine/cmd/dq-engine/main.go:78` constructs one
  `slog.NewJSONHandler` with a single global level
  drawn from `EnvConfig.LogLevel`.
- No `DQ_LOG_LEVELS` environment variable exists.
  No per-package mechanism is implemented.

Foundation 05 §"Metrics" + §"Logging" cites PAT-5
as the per-package override surface but the engine
binary honors only one global level. The example
values in foundation 04 (`engine.compilers`,
`engine.scheduler`) are aspirational; the actual
engine `internal/` tree has different package names
(`alerts`, `api`, `dsl`, `env`, `eval`, `loader`,
`orphan`, `results`, `runner`).

This ADR commits the operations-doc deliverable B2-2
registered: which package names and override syntax
are officially supported by `DQ_LOG_LEVELS`.

The principles bearing on the decision are **P5**
(evolution must be contract-driven — operators
relying on `DQ_LOG_LEVELS` need a stable, documented
contract their CI / runbook scripts can key on),
**P2** (deterministic behavior — the same config
under the same engine version must produce the same
per-package log resolution), and **R3** (do not
revisit settled architecture — foundation 04 PAT-5's
shape is preserved; this ADR formalizes the
unspecified details).

---

## Decision

The `DQ_LOG_LEVELS` contract has five clauses
(grammar, package inventory, precedence, error
handling, defaults). Implementation (the custom
slog handler + main.go wiring + `EnvConfig.LogLevels`
field) is deferred to a B2 follow-up slice.

### Clause 1 — `DQ_LOG_LEVELS` grammar

Formal grammar in PEG-like notation:

```
DQ_LOG_LEVELS := PAIR ("," PAIR)*
PAIR          := PACKAGE ":" LEVEL
PACKAGE       := IDENT ("." IDENT)*    // dot-separated identifier path
IDENT         := [A-Za-z][A-Za-z0-9_]*
LEVEL         := "debug" | "info" | "warn" | "error"   // case-insensitive
```

The level value parse is **case-insensitive**:
`DEBUG`, `Debug`, and `debug` all canonicalize to
the lowercase `debug` internal form. Operators
copying foundation 04 PAT-5's uppercased example
work without translation.

**Whitespace handling**: whitespace immediately
adjacent to `,` (the pair separator) or `:` (the
inside-pair separator) is trimmed by the parser
before pair extraction. Leading + trailing
whitespace around the entire value is also trimmed.
Whitespace **inside** a `PACKAGE` or `LEVEL` token
(e.g., `engine .loader` or `de bug`) is a syntactic
error per Clause 4. Both of these are equivalent
parsed values:

```
DQ_LOG_LEVELS=root:info,engine.loader:debug,engine.runner:warn
DQ_LOG_LEVELS="root: info, engine.loader: debug, engine.runner: warn"
```

The trimming choice is operator-ergonomic: K8s
ConfigMap YAML, `.env` files, and quoted shell
values commonly carry a comfort space after the
comma; rejecting them as fatal would create a
footgun without protecting any invariant.

The reserved package name `root` is the shortest-
possible prefix; it matches any log call that no
other entry matches. `root:LEVEL` replaces
`EnvConfig.LogLevel` when present.

### Clause 2 — Officially-supported package inventory

The leaf names listed below are the **officially-
supported** entries an operator can name in
`DQ_LOG_LEVELS`. The inventory matches the current
`engine/internal/` tree as of this ADR:

| Leaf name | Maps to |
|---|---|
| `root` | Global default (replaces `EnvConfig.LogLevel`) |
| `engine.alerts` | `engine/internal/alerts/` |
| `engine.api` | `engine/internal/api/` |
| `engine.dsl` | `engine/internal/dsl/` |
| `engine.env` | `engine/internal/env/` |
| `engine.eval` | `engine/internal/eval/` |
| `engine.loader` | `engine/internal/loader/` |
| `engine.orphan` | `engine/internal/orphan/` |
| `engine.results` | `engine/internal/results/` |
| `engine.runner` | `engine/internal/runner/` — covers the set + record runners, the trigger-handler attempt machinery, the precheck logic, the Kafka consumer, the execution-ID computation, and the per-check evaluator dispatch (the package owns the runtime path from trigger acceptance through terminal-row write) |

**Intermediate prefixes are also valid override
keys.** Because Clause 3's precedence rule is
longest-prefix-match at dot boundaries, any
dot-separated prefix of a leaf name above can be
used as a wildcard. For example, `engine` (bare) is
a valid prefix that matches every `engine.*` leaf;
an operator setting `engine:warn` raises the entire
engine to `warn` without enumerating each leaf.

Additions to this inventory ship additively under
ADR-0001's compatibility model. A new
`engine/internal/<x>/` package automatically gains
the `engine.<x>` leaf name; the operations doc is
amended in the same PR that lands the package.

**Foundation 04 §"Component Inventory" describes
the platform's logical shape** (`Loader`,
`Scheduler`, `Trigger API`, `Compilers`,
`Coordinator`, `Reporter`, `Alerts`) and that view
is preserved. The package-name inventory above is
the *implemented* contract for `DQ_LOG_LEVELS`. The
two views differ deliberately: `Compilers` became
part of `engine.eval`; the in-engine `Scheduler`
doesn't exist as a package per ADR-0033's
external-scheduler posture; `Coordinator` and
`Reporter` are realized inside `engine.runner`.

Tools (`tools/lint/`, `tools/manifest/`) are
**excluded** from `DQ_LOG_LEVELS` at this contract's
v1 because they are short-lived CLI invocations
where a single `LogLevel` is sufficient. The
per-package-override mechanism's cost is justified
for long-running engines, not for binaries that
exit after one operation. OQ-1 tracks the future
extension if a long-running tool binary lands.

### Clause 3 — Precedence (longest-prefix-match)

When a log call's component name matches multiple
entries, the entry with the **longest matching
prefix** wins. Equality counts as a match.

Examples (assume
`DQ_LOG_LEVELS=root:warn,engine:info,engine.loader:debug`):

| Component | Resolved level | Why |
|---|---|---|
| `engine.loader` | `debug` | Exact match on `engine.loader` |
| `engine.loader.refresh` | `debug` | Longest matching prefix is `engine.loader` |
| `engine.runner` | `info` | Longest matching prefix is `engine` |
| `engine.alerts.publisher` | `info` | Longest matching prefix is `engine` |
| `tools.lint` | `warn` | No `engine.*` prefix matches; falls back to `root` |

The match is on **dot-separated prefix boundaries**,
not on arbitrary string prefix. `engine` matches
`engine.loader` (boundary at the dot) but NOT
`engineroom` (no boundary).

### Clause 4 — Error handling

Two categories of error at engine startup:

**Syntactic errors** (fatal — engine exits non-zero):

- A pair without `:` separator.
- A pair with empty package name (`:debug`) or empty
  level (`engine.loader:`).
- A level value outside the `{debug, info, warn,
  error}` enum (case-insensitive).
- Internal whitespace inside a `PACKAGE` or `LEVEL`
  token (whitespace adjacent to `,` or `:` is
  trimmed per Clause 1).
- A `PACKAGE` token that doesn't match the `IDENT`
  grammar from Clause 1 (e.g., starts with a digit,
  contains a hyphen).

**Unknown package names** (silently ignored):

- A pair like `engine.compilers:debug` where
  `engine.compilers` doesn't exist in the
  inventory. The pair is parsed successfully but
  never matches any log call. The engine logs one
  `info`-level startup line listing all ignored
  package names so operators can audit, but does
  not exit.

The asymmetric handling preserves additive-package
backward compatibility: the engine cannot statically
know all current/future package names (additive
packages expand the inventory), so rejecting
unknown names would break compatibility every time
a new package lands. But the engine must parse the
value to honor it; if parsing fails, exit.

### Clause 5 — Defaults and additivity

- **`DQ_LOG_LEVELS` unset OR empty**: the engine
  uses `EnvConfig.LogLevel` as the single global
  level (current behavior). The new env var is
  additive, not replacing.
- **`DQ_LOG_LEVELS` set without a `root:` entry**:
  `EnvConfig.LogLevel` is the implicit `root`
  value; per-package entries override on top.
- **`DQ_LOG_LEVELS` set with a `root:` entry**:
  the `root:` value replaces `EnvConfig.LogLevel`.
- **EnvConfig storage**: the parsed pairs are
  stored on `EnvConfig` as a structured map. The
  existing `EnvConfig.LogLevel` field continues to
  honor its current shape; a new
  `EnvConfig.LogLevels` field carries the parsed
  map.

**ADR-0018 posture on the new field.** ADR-0018 §4
commits the `EnvConfig` field set as the union of
what existed at promotion time and adds the explicit
clause "Any new field added later is a separate
decision." This ADR IS that separate decision for
the `LogLevels` field. The `EnvConfig.LogLevels`
addition is a structured refinement of the existing
log-level surface (it extends the same logging
concept ADR-0018 already buckets under
"deployment-bucket items"), not a new behavior
category. ADR-0018 is preserved; this ADR formally
exercises ADR-0018's "separate decision" clause for
the logging refinement.

### Implementation posture (deferred)

The slog-handler implementation that resolves a
`component` attribute on each log call against the
parsed `EnvConfig.LogLevels` map is design-only
here. A B2 follow-up slice ships:

- A new `EnvConfig.LogLevels map[string]slog.Level`
  field (parsed from `DQ_LOG_LEVELS`).
- A custom slog handler wrapping the existing JSON
  handler that consults the map per log call.
- The package-instantiation convention: every
  package obtains its logger via `slog.With(
  "component", "engine.<name>", ...)` so the handler
  can resolve.
- Unit tests covering grammar (including
  case-insensitivity and whitespace-trimming),
  precedence, error handling (syntactic-fatal vs
  unknown-name-ignored), defaults.
- The startup-time "ignored-package" audit log
  line.

### Why this does not reopen Foundation 04 / ADR-0007 / ADR-0018

- **Foundation 04 PAT-5** committed the shape; this
  ADR formalizes the details (DD-1). Foundation 04's
  example values (`engine.compilers`,
  `engine.scheduler`) were illustrative — the
  inventory clause supersedes them with the actual
  code organization, but PAT-5's shape commitment
  (env var, comma-separated pairs, per-package
  overrides) is preserved.
- **ADR-0007** committed the loader-strict posture
  (process-exit on startup failure). Clause 4's
  syntactic-error handling cites and applies the
  same posture; no amendment.
- **ADR-0018** commits the `EnvConfig` field set
  with an explicit "separate decision" clause for
  later additions. This ADR exercises that clause
  for the `LogLevels` field; no amendment.

---

## Consequences

1. **The `DQ_LOG_LEVELS` contract is formalized.**
   A five-clause spec gives operators a stable
   contract their CI / runbook scripts can key on.

2. **The package-name inventory aligns to actual
   code.** Ten supported leaf names (`root` +
   nine `engine.<x>` leaves) replace foundation
   04's aspirational examples. Foundation 04's
   logical-component view is preserved separately.

3. **The grammar is operator-ergonomic.**
   Case-insensitive level parse; whitespace
   trimmed around `,` and `:`. Internal whitespace
   inside identifiers remains fatal.

4. **Precedence is longest-prefix-match at dot
   boundaries.** Intermediate prefixes (e.g.,
   `engine:warn`) are valid wildcards; operators
   reason about resolution without consulting code.

5. **Error handling is asymmetric.** Syntactic
   errors fatal (engine exits); unknown package
   names silently ignored (with a one-line startup
   audit). The asymmetry preserves additive-package
   backward compatibility.

6. **`DQ_LOG_LEVELS` is additive to
   `EnvConfig.LogLevel`.** Unset means current
   behavior; set adds per-package overrides on top.

7. **Implementation is deferred to a B2 follow-up
   slice.** The custom slog handler, the
   `EnvConfig.LogLevels` field, the per-package
   instantiation convention, the audit log line,
   and the test surface ship as a separate session.

8. **B2-2 closes.** The decision-log B2-2 row moves
   to `resolved-adr`. One new B2 row registers the
   implementation slice.

9. **Foundation 04 PAT-5 and ADR-0007 / ADR-0018
   are preserved.** This ADR formalizes details
   against PAT-5's shape, cites ADR-0007's exit
   posture, and exercises ADR-0018's "separate
   decision" clause for the `LogLevels` field.

10. **Two deferred items registered out-of-scope:**

    - **OQ-1: `DQ_LOG_LEVELS` for `tools/`
      binaries.** Clause 2 excludes `tools/lint/`
      and `tools/manifest/` from the inventory. If
      a future long-running tool binary lands
      (e.g., a daemon), extending the env-var
      contract to that workspace may be warranted.
      Reserved until a long-running tool binary
      actually exists.

    - **OQ-2: Per-call-site level overrides.** Some
      observability frameworks allow per-call-site
      overrides (e.g., a specific `slog.LogAttrs`
      call carrying an explicit level). This ADR
      commits only per-package overrides;
      per-call-site is reserved until concrete
      operator signal demonstrates the need
      (which has not surfaced).
