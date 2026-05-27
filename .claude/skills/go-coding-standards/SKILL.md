<!-- path: .claude/skills/go-coding-standards/SKILL.md -->
---
name: go-coding-standards
description: Use when writing or reviewing Go code under engine/, tools/, or any Go module in this repo. Encodes the engine's existing conventions: slog with the component attribute per ADR-0043, error wrapping with sentinel errors and package-prefix context, PAT-4 typed env config (one file per env + reflect-based exhaustiveness test), package boundary discipline via doc.go coupling rules, New* constructors returning concrete + error, table-driven tests with optional-type helpers, and the explicit absence of top-level pkg/ or shared/ directories. Apply when adding a new package, writing or amending a test, or reviewing a Go diff.
---

# `go-coding-standards`

Patterns extracted from `engine/internal/` across packages `runner`,
`eval`, `env`, `alerts`, `results`, `loader`, `logging`, and `api`.
Every rule traces to a real file:line. For the full code snippets and
the rationale at length, see the reference file.

> Reference file:
> - `reference/conventions.md` — the seven patterns C1–C7 with file:line
>   citations and verbatim Go snippets (≤6 lines each).

---

## C1. slog via the `component` attribute

Loggers are an **optional Config field**, default to a discarding
no-op, and use the `component` attribute to thread per-package
identity for ADR-0043 level resolution.

```go
type Config struct {
    Logger *slog.Logger
    // ...
}
// In New(...):
if cfg.Logger == nil {
    cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
}
```

`engine/internal/runner/runner.go:175-235`.

The attribute key is `component`, set by the binary at the top of
`main.go` via `slog.With("component", "engine.<x>")`. The handler
resolves levels by longest-prefix-match at dot boundaries.
`engine/internal/logging/handler.go:16-25`.

Conventional attributes (per ADR-0043 + observed usage):
`component`, `execution_id`, `entity`, `check_id`. Never invent new
top-level attributes silently — extend the conventional set in the
relevant ADR first.

## C2. Error wrapping with sentinel errors

Package-scoped sentinel errors with `var ErrXxx = errors.New(...)`
for boundary rejection; `fmt.Errorf("%w", err)` for wrapping; a
package-prefix string when adding context at a package boundary.

```go
var ErrPipeCharacterForbidden = errors.New("input contains forbidden ASCII pipe character (ADR-0002 CC2)")

if strings.ContainsRune(rulesetVersion, '|') {
    return "", fmt.Errorf("ruleset_version: %w", ErrPipeCharacterForbidden)
}
```

`engine/internal/runner/execution_id.go:29-56`.

Package boundary context example:

```go
if _, err := semver.NewVersion(cfg.EngineVersion); err != nil {
    return nil, fmt.Errorf("runner: EngineVersion %q is not valid semver: %w", cfg.EngineVersion, err)
}
```

`engine/internal/runner/runner.go:210-212`.

## C3. PAT-4 typed env config

One Go file per environment (`local.go`, `qa.go`, `prod.go`); each
declares `var <EnvName> = EnvConfig{...}` with identical shape.
Selection via `Select(name string) (EnvConfig, error)`; the switch
returns `ErrUnknownEnv` for anything outside the closed enum.

```go
func Select(name string) (EnvConfig, error) {
    switch Name(name) {
    case NameLocal: return Local, nil
    case NameQA:    return QA, nil
    case NameProd:  return Prod, nil
    default:
        return EnvConfig{}, fmt.Errorf("%w: %q (want local|qa|prod)", ErrUnknownEnv, name)
    }
}
```

`engine/internal/env/config.go:162-173`.

Exhaustiveness is enforced by a reflect-based test that fails the
build if any `EnvConfig` field is the zero value in any environment.
`engine/internal/env/env_test.go:12-43`. This is the "build fails
until every env declares every field" invariant from PAT-4.

## C4. Package boundary discipline via `doc.go`

Each package's `doc.go` documents what it imports and — more
importantly — what it deliberately does **not**. Cross-package
contracts are duck-typed: the consumer declares the interface, the
provider satisfies it implicitly.

Exemplar: `engine/internal/eval/doc.go:9-13`:

> The package boundary deliberately keeps the runner free of
> BigQuery imports: the runner stays an abstraction over
> results.Store; this package owns the cloud.google.com/go/bigquery
> dependency. The exported Evaluator type satisfies
> runner.CheckEvaluator via duck typing.

When adding a new package: write `doc.go` first, name what the
package owns, name what it does not import.

## C5. `New*` constructors return concrete + error

The primary constructor returns `(*Type, error)`. It validates
required fields and defaults optional fields safely inside the
constructor — callers do not pre-fill optional fields.

```go
func New(cfg Config) (*Runner, error) {
    if cfg.Store == nil {
        return nil, errors.New("runner: Store is required")
    }
    // ... more validations
    if cfg.Logger == nil {
        cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
    }
    return &Runner{...}, nil
}
```

`engine/internal/runner/runner.go:203-216`.

Return concrete `*Runner`, not an interface. Callers compose
interfaces at the consumer side; constructors do not pre-narrow.

## C6. Table-driven tests with optional-type helpers

Test functions named descriptively (`TestCompute_HappyPath`, not
`Test_subtest`). Optional-type helpers are short and unexported
(`ptrCheck`, `ptrStr`).

```go
func ptrCheck(r results.CheckResult) *results.CheckResult { return &r }

func TestMapCategory_CheckPass_NoAlert(t *testing.T) {
    cat, emit := MapCategory(SourceRunner, ptrCheck(results.ResultPass), nil)
    if emit { t.Errorf("check=pass should not emit; got category=%q", cat) }
}
```

`engine/internal/alerts/alerts_test.go:17-35`.

Test data flows: table-driven (`tt := []struct{...}{...}; for _, c
:= range tt { ... }`) when scenarios share shape; per-function
descriptive name when each case warrants its own assertion vocabulary.

## C7. No top-level `pkg/` or `shared/`

Isolation, not extraction, is the default. There is no
`engine/internal/pkg/` or `engine/internal/shared/`. Coupling rules
are enforced horizontally through `doc.go` declarations.

When you reach for a shared helper: stop and ask whether the helper
genuinely belongs in one of the existing packages (likely `results`,
`logging`, or a closely related domain). If not, the duplication is
preferred over a premature `shared/`.

---

## Anti-patterns the code consistently avoids

Do not introduce any of these — they are absent from the engine for a reason:

- **Comments that explain WHAT.** Comments in this codebase explain
  *why* — typically pointing at the ADR that committed the behavior
  (`ADR-NNNN §"Section"` or `ADR-NNNN CCN`).
- **Defensive validation at non-boundary points.** Validation lives
  in constructors (`New`) and at API boundaries (`engine/internal/api/decoder.go`),
  never deep inside packages on values that have already been
  validated.
- **Feature flags / backwards-compat shims.** No `if os.Getenv("DQ_USE_OLD_X") == "true"` paths. Versioning happens at the schema layer
  (ADR-0001) and the env config layer (ADR-0018), not via runtime
  flags.
