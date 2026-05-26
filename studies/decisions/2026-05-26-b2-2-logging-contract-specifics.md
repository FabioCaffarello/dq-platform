<!-- path: studies/decisions/2026-05-26-b2-2-logging-contract-specifics.md -->

# B2-2 — Logging Contract Specifics

## Context

Foundation 04 §"PAT-5 — Modular logging contract"
promised a per-package logging override mechanism via
an environment variable:

```
DQ_LOG_LEVELS="root:INFO,engine.compilers:DEBUG,engine.scheduler:WARN"
```

The promise: global default level plus per-package
overrides, expressed as a small layer over Go's
`slog` standard-library package.

What the platform ships **today** is narrower than
that promise:

- `engine/internal/env/config.go:27` declares a typed
  `LogLevel` enum (`debug` / `info` / `warn` /
  `error`).
- `engine/cmd/dq-engine/main.go:78` constructs one
  `slog.NewJSONHandler` with a single global level
  drawn from `EnvConfig.LogLevel`.
- No `DQ_LOG_LEVELS` environment variable exists.
  No per-package mechanism is implemented.

Foundation 05 §"Metrics" + §"Logging" cites PAT-5 as
the per-package override surface but the engine binary
honors only one global level. The example values in
foundation 04 (`engine.compilers`, `engine.scheduler`)
are aspirational; the actual engine `internal/` tree
has different package names (`alerts`, `api`, `dsl`,
`env`, `eval`, `loader`, `orphan`, `results`,
`runner`).

The B2-2 row registered the question:

> Which package names and override syntax are
> officially supported by `DQ_LOG_LEVELS`? Good
> observability patterns help only if standardized.

What B2-2 must commit:

1. **The `DQ_LOG_LEVELS` syntax** — exact grammar
   formalized from foundation 04's example.
2. **The package-name inventory** — aligned to the
   actual engine code organization, not foundation
   04's aspirational names.
3. **Precedence rules** — when multiple entries
   could match a log call, which one wins.
4. **Malformed-value handling** — what does the
   engine do at boot when `DQ_LOG_LEVELS` is set but
   syntactically broken or references an unknown
   package?
5. **Default behavior** — what does an unset or empty
   `DQ_LOG_LEVELS` do? How does the new env var
   relate to the existing `EnvConfig.LogLevel`?
6. **Implementation posture** — the per-package
   override mechanism is design-only here; the
   slog-handler wiring is a B2 follow-up.

The principles bearing on the decision are **P5**
(evolution must be contract-driven — operators
relying on `DQ_LOG_LEVELS` need a stable,
documented contract their CI / runbook scripts can
key on), **P2** (deterministic behavior — the same
config under the same engine version must produce
the same per-package log resolution), and **R3** (do
not revisit settled architecture — foundation 04
PAT-5's shape is preserved; this ADR formalizes the
unspecified details).

---

## Decision Drivers

- **DD-1 — Foundation 04 committed the shape; the
  details were unspecified.** The example
  (`root:INFO,engine.compilers:DEBUG,engine.scheduler:WARN`)
  showed comma-separated pairs and uppercased levels
  but did not commit case-sensitivity, precedence,
  whitespace handling, unknown-package behavior, or
  malformed-value handling. The operations-doc
  expected output for B2-2 fills those gaps.

- **DD-2 — Package-name inventory must reflect
  actual code, not aspirational names.** Foundation
  04 §"Component Inventory" lists `Loader`,
  `Scheduler`, `Trigger API`, `Compilers`,
  `Coordinator`, `Reporter`, `Alerts` as logical
  components. The current `engine/internal/` tree
  ships under different package names —
  `engine.compilers` (a Phase-4 concept) became part
  of `engine.eval`; the in-engine `Scheduler`
  doesn't exist as a package because ADR-0033
  committed an external-scheduler posture;
  `Coordinator` and `Reporter` are realized inside
  `engine.runner`. Foundation 04's component
  inventory is preserved as a *logical* view; this
  ADR commits the *package-name* inventory for
  `DQ_LOG_LEVELS` against the actual code
  organization. The divergence between the two
  views is acceptable because foundation documents
  describe the platform's logical shape and ADRs
  commit the implemented contract.

- **DD-3 — Level value parse is case-insensitive
  with a lowercase canonical form.** Foundation 04
  PAT-5's example uses uppercase (`INFO`, `DEBUG`);
  the existing `EnvConfig.LogLevel` enum accepts
  lowercase (`debug` / `info` / `warn` / `error`)
  per `engine/internal/env/config.go:30-33`. Rather
  than forcing one form, the parser accepts both
  cases and normalizes to lowercase. An operator
  copying foundation 04's `DEBUG` literal works; the
  internal Go enum representation stays lowercase.
  Determinism (P2) is preserved — the same input
  string produces the same parsed result.

- **DD-4 — Longest-prefix-match precedence keeps the
  grammar intuitive.** A pair
  `engine:WARN,engine.loader:DEBUG` should apply
  `DEBUG` to `engine.loader` and `WARN` to other
  `engine.*` packages — the more-specific entry
  beats the less-specific. The `root` token is the
  shortest possible prefix (matches everything when
  no other entry matches).

- **DD-5 — Malformed pair shape is fatal at boot;
  unknown package names are silently ignored.** Two
  distinct categories of error:
  - **Syntactic error** (a pair without `:`, an
    empty level, a level outside the enum) — the
    engine cannot honor what it cannot parse;
    exit-at-startup per ADR-0007's loader-strict
    posture.
  - **Unknown package name** (e.g.,
    `engine.compilers:DEBUG` when the package
    doesn't exist) — silently ignored. The engine
    doesn't know all current/future package names
    statically (additive package additions are
    legal under ADR-0001's compatibility model);
    rejecting unknown packages would break
    backward-compatibility every time a new package
    lands.

- **DD-6 — Per-package mechanism is design-only;
  implementation is a B2 follow-up.** The slog
  handler that resolves a `component` attribute on
  each log call against the `DQ_LOG_LEVELS` map is a
  small but real code change. Following the
  design-only pattern set by prior ADRs (ADR-0030,
  ADR-0032, ADR-0033, ADR-0039, ADR-0041, ADR-0042),
  the contract lands now; the slog-handler wiring
  ships as a separate B2 slice.

- **DD-7 — `DQ_LOG_LEVELS` is additive to
  `EnvConfig.LogLevel`.** When `DQ_LOG_LEVELS` is
  unset or empty, the engine uses `EnvConfig.LogLevel`
  as the single global level (current behavior). When
  it's set, the `root:` entry (if present) replaces
  `EnvConfig.LogLevel`; other entries override per
  package. An unset `root:` falls back to
  `EnvConfig.LogLevel`.

---

## Considered Options

### Option 1 — Commit the contract (syntax + inventory + precedence + error handling + defaults) at the contract level; defer implementation to a B2 slice (recommended)

This ADR commits five contract clauses:

1. **Grammar** — formal syntax for `DQ_LOG_LEVELS`.
2. **Officially-supported package inventory** —
   aligned to current `engine/internal/` packages.
3. **Precedence** — longest-prefix-match.
4. **Error handling** — syntactic errors fatal;
   unknown packages silently ignored.
5. **Defaults** — additive to `EnvConfig.LogLevel`;
   unset means single global level (current
   behavior).

Implementation (slog handler with per-component
resolution + main.go wiring + tests) deferred to a B2
follow-up slice.

**Strengths.** Closes the operations-doc gap from
B2-2 with one cohesive contract. Follows the
design-only pattern from prior ADRs. Aligns the
package inventory to actual code rather than
foundation 04's aspirational names. Honors P2
(deterministic resolution per longest-prefix-match)
and P5 (documented contract for operators).

**Trade-offs.** The per-package override mechanism
still doesn't work end-to-end until the B2 slice
lands; operators wanting per-package verbosity today
have only the global `LogLevel`. Acceptable — the
contract is the load-bearing artefact for the
operations-doc expected output, and the
implementation is mechanical.

### Option 2 — Commit contract + implementation in this session

Same five clauses as Option 1 plus the slog handler
implementation, main.go wiring, env-config field, and
tests.

**Strengths.** Closes the gap fully in one pass.

**Trade-offs.** The slog handler is a real
implementation surface (attribute-based component
resolution, longest-prefix-match logic, override
caching). The platform's standing pattern is
design-only ADR + consumer-slice implementation
(ADR-0030 / ADR-0032 / etc.). Deviating without
strong cause conflates two reviews. Rejected: defer
implementation per DD-6.

### Option 3 — Commit only the syntax; defer inventory + everything else

Commit Clause 1 (grammar) and let the package
inventory, precedence, error handling, and defaults
be decided by the implementation slice.

**Strengths.** Smallest ADR.

**Trade-offs.** The operations doc B2-2 was registered
for would be incomplete — operators still wouldn't
know which package names to use. Splits one
governance decision into multiple ADRs. Rejected —
the contract IS the operations doc; ship it all.

---

## Recommendation

**Option 1.** Commit all five contract clauses at the
contract level; defer the slog-handler implementation
to a B2 follow-up slice.

### Clause 1 — `DQ_LOG_LEVELS` grammar

Formal grammar in PEG-like notation:

```
DQ_LOG_LEVELS := PAIR ("," PAIR)*
PAIR          := PACKAGE ":" LEVEL
PACKAGE       := IDENT ("." IDENT)*    // dot-separated identifier path
IDENT         := [A-Za-z][A-Za-z0-9_]*
LEVEL         := "debug" | "info" | "warn" | "error"      // case-insensitive
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

Tools (`tools/lint/`, `tools/manifest/`) are
**excluded** from `DQ_LOG_LEVELS` at this contract's
v1 because they are short-lived CLI invocations
where a single `LogLevel` is sufficient. The
per-package-override mechanism's cost (custom slog
handler, per-call-site `component` attribute) is
justified for long-running engines, not for binaries
that exit after one operation. If a future
long-running tool binary lands, OQ-1 below tracks
extending the contract; until then the exclusion is
deliberate.

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
  error}` enum.
- Any whitespace character anywhere in the value.
- A `PACKAGE` token that doesn't match the `IDENT`
  grammar from Clause 1 (e.g., starts with a digit,
  contains a hyphen).

**Unknown package names** (silently ignored):

- A pair like `engine.compilers:debug` where
  `engine.compilers` doesn't exist in the inventory.
  The pair is parsed successfully but never matches
  any log call. The engine logs ONE `info`-level
  startup line listing all ignored package names so
  operators can audit, but does not exit.

The asymmetric handling reflects DD-5: the engine
must parse the value to honor it; if parsing fails,
exit. But the engine cannot statically know all
current/future package names (additive packages
expand the inventory); rejecting unknown names would
break backward compatibility every time a new package
lands.

### Clause 5 — Defaults and additivity

- **`DQ_LOG_LEVELS` unset OR empty**: the engine uses
  `EnvConfig.LogLevel` as the single global level
  (current behavior). The new env var is additive,
  not replacing.
- **`DQ_LOG_LEVELS` set without a `root:` entry**:
  `EnvConfig.LogLevel` is the implicit `root` value;
  per-package entries override on top.
- **`DQ_LOG_LEVELS` set with a `root:` entry**: the
  `root:` value replaces `EnvConfig.LogLevel`.
- **Precedence under the env-var resolution order**:
  the existing env-var-to-`EnvConfig` machinery
  reads `DQ_LOG_LEVELS` and the parsed pairs are
  stored on `EnvConfig` as a structured field. The
  engine binary's `EnvConfig.LogLevel` field
  continues to honor its current shape; a new
  `EnvConfig.LogLevels` field carries the parsed map.

**ADR-0018 posture on the new field.** ADR-0018 §4
commits the `EnvConfig` field set as the union of
what existed at promotion time and adds the
explicit clause "Any new field added later is a
separate decision." This ADR IS that separate
decision for the `LogLevels` field. The
`EnvConfig.LogLevels` addition is a structured
refinement of the existing log-level surface (it
extends the same logging concept ADR-0018 already
buckets under "deployment-bucket items"), not a
new behavior category. ADR-0018 is preserved; this
ADR formally exercises ADR-0018's "separate
decision" clause for the logging refinement.

### Implementation posture (deferred)

The slog-handler implementation that resolves a
`component` attribute on each log call against the
parsed `EnvConfig.LogLevels` map is design-only here.
A B2 follow-up slice ships:

- A new `EnvConfig.LogLevels map[string]slog.Level`
  field (parsed from `DQ_LOG_LEVELS`).
- A custom slog handler wrapping the existing JSON
  handler that consults the map per log call.
- The package-instantiation convention: every package
  obtains its logger via `slog.With("component",
  "engine.<name>", ...)` so the handler can resolve.
- Unit tests covering grammar, precedence, error
  handling, defaults.
- The startup-time "ignored-package" audit log line.

### Why this does not reopen Foundation 04 / ADR-0007 / ADR-0018

- **Foundation 04 PAT-5** committed the shape; this
  ADR formalizes the details (DD-1). Foundation 04's
  example values (`engine.compilers`,
  `engine.scheduler`) are illustrative — the
  inventory clause supersedes them with the actual
  code organization, but PAT-5's shape commitment
  (env var, comma-separated pairs, per-package
  overrides) is preserved.
- **ADR-0007** committed the loader-strict posture
  (process-exit on startup failure). This ADR's
  Clause 4 syntactic-error handling cites + applies
  the same posture; no amendment.
- **ADR-0018** committed the env-config struct shape.
  This ADR's Clause 5 commits the additive
  `LogLevels` field, which is an additive extension
  to ADR-0018's struct per ADR-0018's own
  evolution rule.

---

## Consequences

1. **The `DQ_LOG_LEVELS` contract is formalized.** A
   five-clause spec (grammar, inventory, precedence,
   error handling, defaults) gives operators a stable
   contract their CI / runbook scripts can key on.

2. **The package-name inventory aligns to actual
   code.** Ten supported names (`root` +
   `engine.alerts`, `engine.api`, `engine.dsl`,
   `engine.env`, `engine.eval`, `engine.loader`,
   `engine.orphan`, `engine.results`, `engine.runner`)
   replace foundation 04's aspirational examples.

3. **The grammar is strict-no-whitespace.** Matches
   the no-pipe-character rule from W2-5: env-var
   values are primitive substrate inputs, not
   free-form strings.

4. **Precedence is longest-prefix-match at dot
   boundaries.** Operators can reason about resolution
   without consulting code (DD-4).

5. **Error handling is asymmetric.** Syntactic errors
   fatal (engine exits); unknown package names
   silently ignored (with a one-line startup audit).
   The asymmetry preserves additive-package backward
   compatibility (DD-5).

6. **`DQ_LOG_LEVELS` is additive to
   `EnvConfig.LogLevel`.** Unset means current
   behavior; set adds per-package overrides on top.

7. **Implementation is deferred to a B2 follow-up
   slice.** The custom slog handler, the
   `EnvConfig.LogLevels` field, the per-package
   instantiation convention, the audit log line, and
   the test surface ship as a separate session.
   Registered in the decision-log update.

8. **B2-2 closes.** The decision-log B2-2 row moves
   to `resolved-adr`. One new B2 row registers the
   implementation slice.

9. **Foundation 04 PAT-5 and ADR-0007 / ADR-0018 are
   preserved.** This ADR formalizes details against
   PAT-5's shape, cites ADR-0007's exit posture, and
   commits an additive extension to ADR-0018's
   `EnvConfig` per its own evolution rule.

---

## Open Questions

None blocking.

Two deferred items surfaced during drafting and are
explicitly **out-of-scope for current cycle**:

- **OQ-1: `DQ_LOG_LEVELS` for `tools/` binaries.**
  Clause 2 excludes `tools/lint/` and
  `tools/manifest/` from the inventory. If a future
  long-running tool binary lands (e.g., a daemon),
  extending the env-var contract to that workspace
  may be warranted. Reserved until a long-running
  tool binary actually exists.

- **OQ-2: Per-call-site level overrides.** Some
  observability frameworks allow per-call-site
  overrides (e.g., a specific `slog.LogAttrs` call
  carrying an explicit level). This ADR commits only
  per-package overrides; per-call-site is reserved
  until concrete operator signal demonstrates the
  need (which has not surfaced).

---

## Promotion target

`docs/adr/0043-logging-contract-specifics.md` —
next free ADR number. Ships the five-clause
`DQ_LOG_LEVELS` contract: grammar, package
inventory, precedence, error handling, defaults.
