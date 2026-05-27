<!-- path: .claude/skills/go-coding-standards/reference/conventions.md -->

# Go conventions — reference

Seven patterns. Each one: the rule, the citation, a verbatim snippet
(≤6 lines), and a one-paragraph rationale.

---

## C1 — slog via the `component` attribute (ADR-0043)

**Rule.** Logger is an optional `Config` field; constructor defaults
to a discarding handler. Per-package identity flows via the
`component` attribute, resolved by longest-prefix-match.

**Citation.** `engine/internal/logging/handler.go:11-16`:

```go
// componentAttrKey is the slog attribute key the engine binary
// uses to thread per-package identity through the logger chain
// per ADR-0043 §"Implementation posture (deferred)" + Consequence
// #2. Each package's logger is constructed via
// `slog.With(componentAttrKey, "engine.<x>")` in main.go.
const componentAttrKey = "component"
```

Construction site in `engine/internal/runner/runner.go:175-235`:

```go
type Config struct {
    Logger *slog.Logger
    // ...
}
// In New:
if cfg.Logger == nil {
    cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
}
```

**Rationale.** Keeping the logger optional means tests do not need
to wire one up; the discard default is safe. Threading per-package
identity through a single attribute (rather than per-package logger
instances) keeps the `DQ_LOG_LEVELS` per-package override grammar
(ADR-0043) coherent at runtime.

---

## C2 — Error wrapping

**Rule.** `fmt.Errorf("%w", err)` for wrapping. Package-scoped
sentinel errors (`var ErrXxx = errors.New(...)`) for boundary
rejection so callers can match with `errors.Is`. Package-prefix
context string at package boundaries.

**Citation.** `engine/internal/runner/execution_id.go:29-56`:

```go
var ErrPipeCharacterForbidden = errors.New("input contains forbidden ASCII pipe character (ADR-0002 CC2)")

func Compute(...) (string, error) {
    if strings.ContainsRune(rulesetVersion, '|') {
        return "", fmt.Errorf("ruleset_version: %w", ErrPipeCharacterForbidden)
    }
    // ... entity, trigger_source likewise
```

Package-prefix context at a constructor boundary in
`engine/internal/runner/runner.go:210-212`:

```go
if _, err := semver.NewVersion(cfg.EngineVersion); err != nil {
    return nil, fmt.Errorf("runner: EngineVersion %q is not valid semver: %w", cfg.EngineVersion, err)
}
```

**Rationale.** Sentinel errors document the closed set of failure
modes at a package boundary; `%w` preserves the chain so callers
can match. The package-prefix string (`"runner: "`) makes the
ultimate error message readable without requiring callers to wrap
again.

---

## C3 — PAT-4 typed env config (ADR-0018)

**Rule.** One Go file per environment (`local.go`, `qa.go`,
`prod.go`); each declares a `var` of type `EnvConfig` with
identical shape. Selection via `Select(name string) (EnvConfig, error)`
switching on a closed `Name` enum. Reflect-based exhaustiveness
test fails the build if any field is zero in any env.

**Citation.** `engine/internal/env/config.go:150-173`:

```go
var ErrUnknownEnv = errors.New("env: unknown DQ_ENV value")

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

Exhaustiveness test in `engine/internal/env/env_test.go:12-43`
(reflect over struct fields; flag any zero value per env).

**Rationale.** The "build fails until every env declares every
field" invariant is PAT-4's load-bearing property: no env can drift
silently. The exhaustiveness test enforces it at CI time.

---

## C4 — Package boundary discipline via `doc.go`

**Rule.** Each package has a `doc.go` that documents what it owns
and — load-bearing — what it deliberately does not import.
Cross-package contracts are duck-typed: the consumer declares the
interface, the provider satisfies it implicitly.

**Citation.** `engine/internal/eval/doc.go:9-13`:

```
The package boundary deliberately keeps the runner free of
BigQuery imports: the runner stays an abstraction over
results.Store; this package owns the cloud.google.com/go/bigquery
dependency. The exported Evaluator type satisfies
runner.CheckEvaluator via duck typing.
```

**Rationale.** Without explicit non-import declarations, the
runner could silently grow a BigQuery dependency in a future PR.
The `doc.go` makes the boundary reviewable: any PR that violates
it is visible in the diff. Duck-typing avoids a `pkg/contract/`
trap.

---

## C5 — `New*` constructors return concrete + error

**Rule.** Primary constructor signature: `func New(cfg Config) (*Type, error)`.
Required fields validated up front (return error); optional fields
defaulted inside the constructor (e.g., nil Logger → discard
handler). Return concrete type, not interface.

**Citation.** `engine/internal/runner/runner.go:203-216`:

```go
func New(cfg Config) (*Runner, error) {
    if cfg.Store == nil {
        return nil, errors.New("runner: Store is required")
    }
    if _, err := semver.NewVersion(cfg.EngineVersion); err != nil {
        return nil, fmt.Errorf("runner: EngineVersion %q is not valid semver: %w", cfg.EngineVersion, err)
    }
    // ... more validations, optional-field defaults
    return &Runner{...}, nil
}
```

**Rationale.** Returning concrete types lets callers compose
interfaces at the call site (cf. C4). Validating in the constructor
removes a class of "field-not-set" bugs that would otherwise
manifest deep inside `Run`.

---

## C6 — Table-driven tests with optional-type helpers

**Rule.** Test functions are named descriptively, not
`Test_subtest`. Optional-type helpers (`ptrCheck`, `ptrStr`) are
short, unexported, and live in the test file.

**Citation.** `engine/internal/alerts/alerts_test.go:17-35`:

```go
func ptrCheck(r results.CheckResult) *results.CheckResult { return &r }

func TestMapCategory_CheckPass_NoAlert(t *testing.T) {
    cat, emit := MapCategory(SourceRunner, ptrCheck(results.ResultPass), nil)
    if emit {
        t.Errorf("check=pass should not emit; got category=%q", cat)
    }
}
```

For scenarios that share shape, use table-driven tests; for
scenarios with distinct assertion vocabularies, use per-function
descriptive names.

**Rationale.** Descriptive names make failed-test output readable
without opening the test file. Optional-type helpers keep test
literals compact without introducing helper packages.

---

## C7 — No top-level `pkg/` or `shared/`

**Rule.** There is no `engine/internal/pkg/` or
`engine/internal/shared/`. When tempted to extract, ask first
whether the helper genuinely belongs in an existing domain
package.

**Citation.** Absence — confirmed by `find engine/internal -type d`
returning only domain packages.

**Rationale.** A `shared/` package becomes a magnet for unrelated
code over time; isolation via `doc.go` (C4) is the project's
discipline instead. Three lines of duplication are cheaper than a
premature abstraction (per `CLAUDE.md` "Doing tasks").

---

## Anti-patterns the code consistently avoids

These are absent from `engine/internal/` for a reason. Do not
introduce them.

- **WHAT-style comments.** Comments explain WHY, typically pointing
  at the ADR that committed the behavior. `// ADR-0002 CC2:
  input safety` is the canonical shape.
- **Defensive validation at non-boundary points.** Validation lives
  at constructors and at API decoder boundaries
  (`engine/internal/api/decoder.go`). Internal code trusts what
  the boundary already verified.
- **Feature flags or backwards-compat shims.** No
  `if os.Getenv("DQ_USE_OLD_X") == "true"` paths. Versioning lives
  at the schema layer (ADR-0001) and the env config layer
  (ADR-0018), not at runtime via flags.
