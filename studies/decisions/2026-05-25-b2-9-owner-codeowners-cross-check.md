<!-- path: studies/decisions/2026-05-25-b2-9-owner-codeowners-cross-check.md -->

# B2-9 — `_owners.yaml` Owner ↔ CODEOWNERS-Group Linter Cross-Check

## Context

[ADR-0006](../../docs/adr/0006-alert-routing-contract.md) §9
commits the linter (`dq-lint`) as the **first enforcement
point** for "no alert without owner": every entity declared
in a rule YAML must have an entry in `_owners.yaml`, and an
MR introducing an entity without one fails CI. The linter
already issues that CC9 check (`CheckRulesHaveOwners` in
`tools/lint/owners.go`).

What ADR-0006 §9 did **not** commit was a check that the
**value** of the `owner:` field inside each `_owners.yaml`
entry resolves to a real review group. Today an operator can
write:

```yaml
entities:
  customer:
    owner: "@PLACEHOLDER-org/typo'd-group"     # not in CODEOWNERS
    mode: set
    channels:
      data_quality:
        - slack:#dq-customer
```

and the linter accepts it — the schema is satisfied (the
`owner` field is a `minLength: 1` string), `CheckRulesHaveOwners`
is satisfied (the entity is declared), and the misalignment
only surfaces at PR-review time when GitHub's CODEOWNERS
evaluation fails to route the change to a real human group.

[ADR-0015](../../docs/adr/0015-codeowners.md) §2 commits the
**group inventory** (three groups today:
`@PLACEHOLDER-org/platform-team`,
`@PLACEHOLDER-org/sre`,
`@PLACEHOLDER-org/rules-authors`) and §3 commits the
**path-rule table** that maps repository paths to those
groups. The `_owners.yaml`'s `owner:` field is meant to
reference one of those groups (until per-entity refinement
lands as an additive change per ADR-0015 §11). The
cross-check that ADR-0015 §"Notes" reserved as an explicit
follow-up — "defense-in-depth `_owners.yaml` validation at
the manifest publisher and engine loader, beyond the
linter's first-line check" — has a cheaper, earlier
enforcement point: the linter itself.

The decision-log row that registered this follow-up was
**B2-9** at W3-P8b closure:

> Should `dq-lint` parse `.github/CODEOWNERS` and reject
> `_owners.yaml` entries whose `owner:` does not correspond
> to an existing CODEOWNERS group? ADR-0006 §9 commits the
> linter as the first enforcement point for "no alert
> without owner"; without the cross-check, a stale or
> typo'd group reference only fails at PR-review time.
> Defense-in-depth complement to OQ-B1-9.3 (publisher/
> loader-side validation).

The principles bearing on the decision are **P3** (ownership
is explicit — a typo'd owner is silently un-owned until
GitHub's CODEOWNERS engine reports it; lint-time enforcement
makes the unfix-able state unrepresentable in the committed
artefact), **P4** (cost is a first-class constraint — the
lint pass is already in CI; adding one parse + one membership
check is the cheapest enforcement layer that closes the
gap), and **R3** (do not revisit settled architecture —
ADR-0006 §9 and ADR-0015 are preserved; this study extends
the linter under their existing contracts).

What B2-9 must commit:

1. **The check's scope** — what subset of CODEOWNERS the
   linter parses, and what it does NOT.
2. **The failure mode** — when no CODEOWNERS file is
   present, when a referenced group is unknown, and when
   the cross-check is disabled.
3. **The placeholder-substitution compatibility** — the
   `@PLACEHOLDER-org/...` literal discipline from ADR-0015
   §4 must continue to work; the linter must see the same
   literals on both sides until the operational substitution
   session swaps them.
4. **The CLI surface** — what flag controls the cross-check
   and what the default is.

---

## Decision Drivers

- **DD-1 — Failure-mode timing.** Today a stale or typo'd
  `owner:` string only surfaces when a PR opens and GitHub
  evaluates CODEOWNERS; the author waits for review-routing
  to break before learning. Moving the check to lint time
  surfaces it at `make lint`, the same loop authors already
  run pre-commit.

- **DD-2 — No new substrate; reuse the existing linter
  exit-code contract.** `dq-lint` already exits 1 on
  validation errors and 2 on operational errors (per
  `tools/lint/main.go`). The cross-check fits the
  validation-error branch with no new exit code, no new CI
  lane, and no new external dependency.

- **DD-3 — Defense-in-depth, not single-line-of-defense.**
  The cross-check does NOT replace ADR-0006 §9's CODEOWNERS
  review-routing — the review group still has to approve
  the MR. The cross-check is the cheaper first line of
  defense: a misaligned `owner:` value fails locally at
  lint time before any commit reaches GitHub. The publisher
  and loader follow-up reserved by ADR-0015 §"Notes" stays
  open as a higher-cost defense-in-depth complement.

- **DD-4 — Bound the validation scope.** The linter parses
  CODEOWNERS to extract its **group inventory** — the set
  of tokens of the form `@<org>/<team>` that appear as
  reviewers anywhere in the file. Scope is enforced
  behaviorally by the parse order, not by the reviewer-
  token regex alone: the parser discards the first
  whitespace-separated field on every rule line (the
  path-pattern column) before applying the reviewer-token
  regex to the remaining fields. This drops `*` (the
  default-path token on CODEOWNERS line 10) and any other
  path pattern without ever testing it against the
  identifier regex. The linter does NOT validate path-
  rule semantics (which group owns which path), and does
  NOT enforce that the rule under `/rules/<entity>.yaml`'s
  CODEOWNERS group equals the `_owners.yaml` entry's
  `owner:` field. Both of those are beyond the scope of
  "is this owner a real review group?" and would require
  committing the per-entity CODEOWNERS refinement ADR-0015
  §11 reserves as an additive future change.

- **DD-5 — Placeholder-substitution compatibility.**
  ADR-0015 §4 commits that `PLACEHOLDER-org/` is intentional
  literal text that survives in the committed file until
  the operational substitution session, and ADR-0015
  Consequence #4 commits that the operational session
  adjusts `_owners.yaml` in the same change so the literal
  substitution stays consistent across both files. The
  `_owners.yaml` entries today reference
  `@PLACEHOLDER-org/rules-authors` and CODEOWNERS line 20
  carries the same literal. The cross-check parses
  CODEOWNERS verbatim, so it sees the same placeholder
  literals on both sides; the check passes today and
  continues to pass after the mechanical substitution
  because Consequence #4 guarantees both files are edited
  in the same MR.

- **DD-6 — Missing CODEOWNERS file: warn or fail?** Two
  failure-mode options exist when the linter cannot read
  `.github/CODEOWNERS`:
  - **Loud:** treat as operational error (exit 2). Reflects
    "CODEOWNERS is part of the contract surface; its
    absence is a misconfiguration."
  - **Quiet:** treat as the check being disabled (exit 0).
    Reflects "the linter can run outside the repository
    root, e.g., in a fixture-only test harness."
  The repository today commits CODEOWNERS at
  `.github/CODEOWNERS` and CI always runs from the
  repository root, so the loud failure mode is the
  consistent default. An explicit `-codeowners=""` flag
  disables the cross-check for fixture-only test harnesses
  per DD-7 below.

- **DD-7 — Disable for fixture-only test harnesses.** The
  linter's existing `lint_test.go` fixtures construct
  ad-hoc `rules/` trees that do not carry a parent
  CODEOWNERS file; the cross-check must be opt-out so
  those tests continue to pass without manufacturing a
  CODEOWNERS fixture per test. An empty `-codeowners=""`
  value disables the check; existing tests pass the empty
  string.

---

## Considered Options

### Option 1 — Lint-time owner-membership cross-check (recommended)

A new lint pass parses `.github/CODEOWNERS`, extracts the
set of group identifiers (tokens matching
`@<org>/<team>`), and rejects any `_owners.yaml` entry
whose `owner:` value is not in that set. Internals:

```
// tools/lint/codeowners.go
type CodeOwnersGroups map[string]struct{}

func LoadCodeOwners(path string) (CodeOwnersGroups, error) { ... }
```

`LoadCodeOwners` ignores comment lines (leading `#`), blank
lines, and the path-pattern column on each rule line. From
the remaining whitespace-separated tokens it keeps the ones
that match `^@[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)?$`
(GitHub's user/group identifier shape: a leading `@`
followed by org/team or just user). The membership set is
the union across all rule lines.

`CheckOwnersGroupMembership` (new function in `owners.go`)
iterates `Owners.Entities`, looks up each entry's `owner`
value in the membership set, and emits one
`ValidationError` per miss. The error keys on
`rules/_owners.yaml` and quotes the offending value plus
the closest matches from the set (Levenshtein-1 suggestion
where cheap; no suggestion when ambiguous).

Wired into `main.go` via a new flag:

```
-codeowners <path>   path to .github/CODEOWNERS
                     (default: .github/CODEOWNERS).
                     Empty string disables the cross-check.
```

Behavior matrix:

| `-codeowners` value | CODEOWNERS readable | Result |
|---|---|---|
| default (`/.github/CODEOWNERS`) | yes | check runs; misses → exit 1 |
| default | no (file missing) | exit 2 (operational error) |
| `""` | n/a | check disabled (exit 0 path unchanged) |
| explicit path | yes | check runs |
| explicit path | no | exit 2 (operational error) |

**Strengths.** Closes the timing gap from DD-1 with the
cheapest enforcement point. Reuses the existing linter
exit-code contract (no new code paths in CI). Preserves
the placeholder discipline from DD-5 (no string-massaging
on either side). Scope is bounded per DD-4 (no path-rule
semantics; no per-entity routing claims).

**Trade-offs.** Adds one new file (~60 LOC) plus one
function in `owners.go` (~30 LOC) plus tests (~150 LOC).
Modest cost for the gap closure. The parser is
deliberately permissive — it ignores anything that does
not look like a group identifier — so future CODEOWNERS
additions (path patterns with email addresses,
`secret_scanning` rules, etc.) do not produce false
positives.

### Option 2 — Lint check + publisher/loader-side validation

Land the lint check from Option 1 AND extend the manifest
publisher and engine loader to validate `_owners.yaml`'s
`owner:` strings against CODEOWNERS at publish time and
load time. This is what ADR-0015 §"Notes" reserved as a
follow-up.

**Strengths.** Defense-in-depth at three layers (lint /
publish / load) instead of two (lint / GitHub review
routing).

**Trade-offs.** Out of scope for B2-9. Publisher-side
validation requires committing how CODEOWNERS travels to
the publisher's working environment (the publisher does
not read the repository tree today; it reads `rules/` from
a build artefact). Engine-loader-side validation reopens
ADR-0007's loader-verification set — a larger contract
change than B2-9 was registered to commit. Both reserved
as separate follow-ups (OQ-B1-9.3 covers them) and not
merged into this study to keep the scope tight per CLAUDE.md
R4 ("one topic per session").

### Option 3 — Status quo (no lint-time cross-check)

Leave the cross-check to GitHub's CODEOWNERS engine at
PR-review time. Operators learn at review time, not at
lint time.

**Strengths.** Zero implementation cost.

**Trade-offs.** Fails B2-9's expected output ("CLI design
note + subcommand under `tools/manifest/`" — that's the
sibling B2-10 row; B2-9's expected output is "linter rule
design note + `dq-lint` extension"). Leaves the failure-
mode timing gap (DD-1) open. Rejected.

---

## Recommendation

**Option 1.** Lint-time owner-membership cross-check with
the `-codeowners <path>` flag (default
`.github/CODEOWNERS`; empty disables).

### CLI surface

```
dq-lint \
  -schema rules/_schema/v1.schema.json \
  -schema-v2 rules/_schema/v2.schema.json \
  -owners-schema rules/_schema/_owners.v1.schema.json \
  -owners-schema-v2 rules/_schema/_owners.v2.schema.json \
  -catalog rules/_schema/catalog.v1.yaml \
  -owners rules/_owners.yaml \
  -rules rules \
  -codeowners .github/CODEOWNERS     \  NEW; empty disables
```

### Internal design

A new file `tools/lint/codeowners.go` exposes:

```
// CodeOwnersGroups is the set of @org/team and @user
// identifiers that appear anywhere in a parsed CODEOWNERS
// file. The linter cross-checks _owners.yaml's owner
// values against this set.
type CodeOwnersGroups struct {
    set  map[string]struct{}
    Path string  // for diagnostics; empty if not loaded
}

func LoadCodeOwners(path string) (*CodeOwnersGroups, error)
func (g *CodeOwnersGroups) Contains(group string) bool
func (g *CodeOwnersGroups) Slice() []string  // sorted, for diagnostics
```

`LoadCodeOwners` returns an empty set + nil error when
`path == ""` (the disable case). Otherwise it reads the
file; an I/O error returns `(nil, err)`, which `main.go`
maps to exit 2.

The parse is line-oriented:

1. Strip trailing CR.
2. Trim. Skip if empty or starts with `#`.
3. Split on whitespace. Discard the first field (the path
   pattern).
4. Of the remaining fields, keep each that matches
   `^@[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)?$`. Insert into
   the set.

The regex matches:
- `@user` (a single user, e.g., `@octocat`)
- `@org/team` (a team, e.g., `@PLACEHOLDER-org/platform-team`)

It rejects bare emails (CODEOWNERS allows them; the linter
ignores them — `_owners.yaml`'s `owner:` is always a group
identifier per ADR-0015 §2), and bare non-`@` strings.

### Cross-check function

`owners.go` gains:

```
// CheckOwnersGroupMembership verifies every owner: value
// in the owners set resolves to a CodeOwnersGroups entry.
// A nil or empty groups argument disables the check
// (returns nil). Returns one ValidationError per miss,
// keyed on the _owners.yaml path.
func CheckOwnersGroupMembership(
    owners *Owners,
    groups *CodeOwnersGroups,
) []ValidationError
```

Error message shape:

```
rules/_owners.yaml: entity %q: owner %q does not match any
CODEOWNERS group in %s; valid groups: [@PLACEHOLDER-org/...
@PLACEHOLDER-org/...]
```

The "valid groups" suggestion is the sorted slice from
`CodeOwnersGroups.Slice()`. No Levenshtein magic in v1;
the suggestion is "here's the inventory you can pick
from."

### main.go wiring

`LoadCodeOwners` returns a non-nil `*CodeOwnersGroups` for
every successful call — including the empty-path disable
case, where the returned set is empty. `CheckOwnersGroupMembership`
treats a nil-or-empty set as the disable signal and returns
no errors. The two together mean the call site does not
need a nil guard:

```
codeowners, err := LoadCodeOwners(*codeownersPath)
if err != nil {
    fmt.Fprintf(os.Stderr, "dq-lint: %v\n", err)
    os.Exit(exitOperationalError)
}
if errs := CheckOwnersGroupMembership(owners, codeowners); len(errs) > 0 {
    results[*ownersPath] = append(results[*ownersPath], errs...)
}
```

The append matches the slice-append pattern already used
by `CheckRulesHaveOwners`'s caller a few lines above.

### Test surface

`tools/lint/codeowners_test.go` covers the parser:

- Empty file → empty set.
- Comment-only file → empty set.
- Single rule line → one group.
- Multiple rule lines with overlap → deduplicated set.
- Bare email reviewer → ignored.
- Path patterns with special characters (`/rules/_schema/`)
  → discarded; only the reviewer tokens kept.
- Real `.github/CODEOWNERS` golden test (committed
  fixture).

`tools/lint/owners_test.go` extends with:

- `CheckOwnersGroupMembership` happy path: known group →
  no error.
- Unknown group → one error per miss.
- Nil groups → no error (disable case).
- Empty groups → no error (degenerate disable case).
- Multiple unknown groups → multiple errors, one per
  entity.

### Why this does not reopen ADR-0015

ADR-0015 §2 commits the group inventory and §3 commits the
path-rule table. The cross-check reads the *inventory* (a
shape this study does not change) without making any claim
about the *table* (which groups own which paths). ADR-0015
§11 reserves per-entity CODEOWNERS refinement as additive;
the cross-check does not preempt or block that future
change.

### Why this does not reopen ADR-0006

ADR-0006 §9 commits the linter as the first enforcement
point for "no alert without owner". This study adds a
second invariant the same enforcement layer checks ("the
owner value resolves to a real group"); both are
properties the linter validates at the same exit-code
contract. The new check is additive within ADR-0006's
"linter is the first enforcement point" commitment.

---

## Consequences

1. **A new file `tools/lint/codeowners.go` ships** carrying
   the parser + the membership-set type. ~60 LOC.

2. **`tools/lint/owners.go` gains
   `CheckOwnersGroupMembership`** (~30 LOC) — the
   cross-check that compares `Owners.Entities[*].Owner`
   against the loaded CODEOWNERS group set.

3. **`tools/lint/main.go` gains the `-codeowners` flag**
   (default `.github/CODEOWNERS`; empty disables) and the
   wiring that invokes the cross-check after
   `CheckRulesHaveOwners`.

4. **Unit tests in `tools/lint/codeowners_test.go` and the
   extended `owners_test.go`** cover parser edge cases and
   cross-check happy + unhappy paths.

5. **The `_owners.yaml`'s `owner:` schema constraint
   tightens at the linter, not at JSON Schema.** The
   schema continues to declare `owner: { type: string,
   minLength: 1 }` — JSON Schema cannot express
   "membership in a dynamically-loaded set". The cross-
   check is the dynamic enforcement layer.

6. **The placeholder-substitution discipline from
   ADR-0015 §4 is preserved.** Today both files reference
   `@PLACEHOLDER-org/...`; the cross-check passes. After
   the operational substitution session both files
   reference the real org slug atomically; the cross-check
   continues to pass.

7. **Defense-in-depth at three layers stays the future
   target.** The lint check is layer #1 (cheapest).
   GitHub's CODEOWNERS review routing is layer #2 (review
   time). The publisher- and loader-side validation
   reserved by ADR-0015 §"Notes" is layer #3 (production
   write- and read-time); B2-9 does not preempt or supersede
   that follow-up — it reduces its urgency by closing the
   timing gap at the cheapest layer first.

8. **`dq-lint`'s contract surface grows by one optional
   flag and one optional cross-check.** Operators running
   the linter without `-codeowners=""` get the check; CI
   gets it by default; fixture-only test harnesses opt out
   via the empty value.

9. **B2-9 closes.** The decision-log B2-9 row moves to
   `resolved-adr` (→ ADR-0037). OQ-B1-9.3 stays open as
   the publisher/loader-side follow-up.

10. **One implementation note: this is a `dq-lint`
    extension, not a new binary.** The CLI binary name,
    the exit-code mapping, and the existing flag set stay
    unchanged. The cross-check ships under the same
    `tools/lint/` workspace as one additive flag + one
    new file.

---

## Open Questions

None blocking.

Two deferred items surfaced during drafting and are
explicitly **out-of-scope for current cycle**:

- **OQ-1: Per-entity CODEOWNERS-path cross-check.**
  ADR-0015 §11 reserves per-entity CODEOWNERS refinement
  as an additive future change (e.g.,
  `/rules/customer.yaml @customer-team`). When that lands,
  the linter could be extended to check that an entity's
  `_owners.yaml` `owner:` value equals the CODEOWNERS
  group that owns the matching `/rules/<entity>.yaml` path.
  Reserved until per-entity CODEOWNERS lines exist.

- **OQ-2: Email-reviewer support.** GitHub allows bare
  emails (e.g., `user@example.com`) as CODEOWNERS
  reviewers. The current parser ignores them because
  `_owners.yaml`'s `owner:` is committed to be a group
  identifier per ADR-0015 §2. If a future ADR amends
  ADR-0015's group-inventory commitment to include
  individual reviewers, the parser extends to accept the
  email shape. Reserved until that ADR lands.

---

## Promotion target

`docs/adr/0037-owner-codeowners-cross-check.md` — next free
ADR number (the prior promoted ADR is 0036, the B2-10
set-pointer subcommand). Ships the linter-extension
commitment, the parser scope, the cross-check shape, and
the `-codeowners` flag contract.
