<!-- path: docs/adr/0037-owner-codeowners-cross-check.md -->

# ADR-0037 — `_owners.yaml` Owner ↔ CODEOWNERS-Group Linter Cross-Check

- **Status:** accepted
- **Date:** 2026-05-25

---

## Context

[ADR-0006](./0006-alert-routing-contract.md) §9 commits the
linter (`dq-lint`) as the **first enforcement point** for
"no alert without owner": every entity declared in a rule
YAML must have an entry in `_owners.yaml`, and an MR
introducing an entity without one fails CI. The linter
already enforces that invariant via
`CheckRulesHaveOwners` in `tools/lint/owners.go`.

What ADR-0006 §9 did not commit was a check that the
**value** of the `owner:` field inside each `_owners.yaml`
entry resolves to a real review group. Today an operator
can write any non-empty string under `owner:` — the schema
accepts it (`type: string, minLength: 1`), the
`CheckRulesHaveOwners` cross-check accepts it (the entity
is declared), and the misalignment only surfaces at
PR-review time when GitHub's CODEOWNERS evaluation fails
to route the change to a real human group.

[ADR-0015](./0015-codeowners.md) §2 commits the **group
inventory** (`@PLACEHOLDER-org/platform-team`,
`@PLACEHOLDER-org/sre`,
`@PLACEHOLDER-org/rules-authors`) and §3 commits the
**path-rule table** that maps repository paths to those
groups. The `_owners.yaml` `owner:` field is meant to
reference one of those groups (until per-entity
refinement lands as an additive change per ADR-0015 §11).
ADR-0015 §"Notes" reserved as a follow-up "defense-in-depth
`_owners.yaml` validation at the manifest publisher and
engine loader, beyond the linter's first-line check" — but
the cheapest, earliest enforcement point is the linter
itself, and that follow-up is what this ADR delivers.

The principles bearing on the decision are **P3**
(ownership is explicit — a typo'd owner is silently
un-owned until GitHub's CODEOWNERS engine reports it at
PR-open time; lint-time enforcement makes the unfix-able
state unrepresentable in the committed artefact), **P4**
(cost is a first-class constraint — the lint pass is
already in CI; one file parse plus one set-membership
check is the cheapest enforcement layer that closes the
gap), and **R3** (do not revisit settled architecture —
ADR-0006 §9 and ADR-0015 are preserved; this ADR extends
the linter under their existing contracts).

---

## Decision

The `dq-lint` binary gains a CODEOWNERS-group cross-check
that rejects `_owners.yaml` entries whose `owner:` value
does not match any group identifier appearing as a
reviewer token in `.github/CODEOWNERS`. The check is
gated by a new optional flag and uses the linter's
existing validation-error exit code (1).

### CLI surface

```
dq-lint \
  -codeowners <path>           \   NEW; default .github/CODEOWNERS;
                                    empty disables the cross-check
  …existing flags unchanged…
```

The default value lets `make lint` and CI invocations get
the check for free; the empty-string disable lets the
linter's own fixture-only test harnesses (which construct
ad-hoc `rules/` trees without a parent CODEOWNERS file)
continue to pass.

Behavior matrix:

| `-codeowners` value | CODEOWNERS readable | Result |
|---|---|---|
| default (`.github/CODEOWNERS`) | yes | check runs; misses → exit 1 |
| default | no (file missing) | exit 2 (operational error) |
| `""` | n/a | check disabled (exit-0 path unchanged) |
| explicit path | yes | check runs |
| explicit path | no | exit 2 (operational error) |

### Parser scope

The parser is line-oriented and reads only what it needs
to harvest the group inventory:

1. Strip trailing CR; trim whitespace.
2. Skip blank lines and lines beginning with `#`.
3. Split on whitespace. **Discard the first field** (the
   CODEOWNERS path-pattern column).
4. From the remaining fields, keep each token matching
   `^@[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)?$` — the GitHub
   CODEOWNERS reviewer-identifier shape (`@<user>` or
   `@<org>/<team>`). Insert into the inventory set.

The path-pattern discard at step 3 is what protects the
parser from false-positive group entries: the default-path
token `*` (CODEOWNERS line 10), any `/`-anchored path
pattern, and any pathological glob never reach the
reviewer-token regex.

Bare email reviewers (a CODEOWNERS-permitted shape) are
ignored because `_owners.yaml`'s `owner:` is committed to
be a group identifier per ADR-0015 §2. If a future ADR
amends ADR-0015's group-inventory commitment to include
individual reviewers, the parser extends additively to
accept the email shape.

### Cross-check shape

`CheckOwnersGroupMembership(owners, groups) []ValidationError`
iterates the loaded owners set in sorted-name order. For
each entity:

- If the entry's `owner:` is empty, skip — JSON Schema
  validation in `LoadOwners` already requires `minLength:
  1`; double-reporting is suppressed.
- If the entry's `owner:` is present in the loaded
  CODEOWNERS inventory, accept.
- Otherwise, emit one `ValidationError` per miss with a
  message of the shape:

  ```
  ADR-0037: entity "<name>" owner "<value>" does not match
  any CODEOWNERS group in <path>; valid groups: [<sorted
  inventory>]
  ```

  The valid-groups list is the deduplicated inventory of
  reviewer tokens from the loaded CODEOWNERS file, sorted
  alphabetically for stable diagnostic output.

The function tolerates a nil or empty `groups` argument
(returns nil), and tolerates a nil `owners` argument
(returns nil). The call site in `main.go` does not need
guards.

### Scope bounds

The cross-check validates one property: **the owner
field's literal string is a member of the CODEOWNERS
group inventory**. It does NOT:

- Validate path-rule semantics (which group owns which
  path).
- Enforce that the rule under `/rules/<entity>.yaml`'s
  CODEOWNERS group equals the `_owners.yaml` entry's
  `owner:` field.
- Implement Levenshtein suggestions for near-misses (the
  diagnostic lists the full inventory; operators pick from
  it).

Both of the first two require committing the per-entity
CODEOWNERS refinement that ADR-0015 §11 reserves as an
additive future change, and are reserved as separate
follow-up B-items.

### Placeholder-substitution compatibility

ADR-0015 §4 commits that `PLACEHOLDER-org/` is intentional
literal text that survives in the committed file until
the operational substitution session. ADR-0015
Consequence #4 commits that the operational session
adjusts `_owners.yaml` in the same change so the literal
substitution stays consistent across both files.

The cross-check parses CODEOWNERS verbatim, so it sees
the same placeholder literals on both sides today
(`@PLACEHOLDER-org/rules-authors` in CODEOWNERS line 20
and in `_owners.yaml`). After the operational substitution
session, Consequence #4 guarantees both files carry the
real org slug; the cross-check continues to pass without
modification.

### Why this does not reopen ADR-0015

ADR-0015 §2 commits the group inventory and §3 commits
the path-rule table. The cross-check reads the *inventory*
without making any claim about the *table* (which groups
own which paths). ADR-0015 §11 reserves per-entity
CODEOWNERS refinement as additive; the cross-check does
not preempt or block that future change. The placeholder-
substitution discipline from ADR-0015 §4 + Consequence #4
is preserved because both files carry the same literal
on both sides at all points in the substitution lifecycle.

### Why this does not reopen ADR-0006

ADR-0006 §9 commits the linter as the first enforcement
point for "no alert without owner". This ADR adds a
second invariant the same enforcement layer checks ("the
owner value resolves to a real group"); both are
properties the linter validates under the same exit-code
contract. The new check is additive within ADR-0006 §9's
"linter is the first enforcement point" commitment.

---

## Consequences

1. **A new file `tools/lint/codeowners.go` ships** carrying
   the parser and the `CodeOwnersGroups` membership-set
   type. The parser is line-oriented, discards the path-
   pattern column, and harvests reviewer tokens matching
   the `@<user>` or `@<org>/<team>` shape.

2. **`tools/lint/owners.go` gains
   `CheckOwnersGroupMembership`** — the cross-check that
   compares `Owners.Entities[*].Owner` against the loaded
   CODEOWNERS group inventory. `OwnerEntity` gains an
   `Owner` field so `LoadOwners` can capture the value
   for the cross-check to consume.

3. **`tools/lint/main.go` gains the `-codeowners` flag**
   (default `.github/CODEOWNERS`; empty disables) and the
   wiring that invokes the cross-check after
   `CheckRulesHaveOwners`. Errors flow through the
   existing exit-code path (1 on validation error, 2 on
   operational error).

4. **Unit tests in `tools/lint/codeowners_test.go` and
   the extended `owners_test.go`** cover parser edge
   cases (empty file, comment-only, default-path token
   discarded, multiple reviewers per line, dedup, bare
   email ignored, bare user token accepted, path without
   reviewer skipped, real repository CODEOWNERS golden
   test, nil-safe receivers), cross-check happy path,
   unknown-group rejection, multiple misses, empty-owner
   skip, and the disable case via empty groups.

5. **`make lint` and CI get the check by default.** The
   Makefile target `lint-rules` does not pass
   `-codeowners` explicitly; it inherits the default
   `.github/CODEOWNERS` path. Existing CI lanes pass
   unchanged because the production `_owners.yaml` and
   the production `.github/CODEOWNERS` reference the same
   `@PLACEHOLDER-org/...` literals.

6. **Defense-in-depth at three layers stays the future
   target.** The lint check is layer #1 (cheapest).
   GitHub's CODEOWNERS review routing is layer #2 (review
   time). The publisher- and loader-side validation
   reserved by ADR-0015 §"Notes" is layer #3 (production
   write- and read-time); this ADR does not preempt or
   supersede that follow-up — it reduces its urgency by
   closing the timing gap at the cheapest layer first.

7. **`_owners.yaml`'s `owner:` schema constraint
   tightens at the linter, not at JSON Schema.** The
   schema continues to declare `owner: { type: string,
   minLength: 1 }` — JSON Schema cannot express
   "membership in a dynamically-loaded set". The cross-
   check is the dynamic enforcement layer.

8. **`dq-lint`'s contract surface grows by one optional
   flag and one optional cross-check.** Operators running
   the linter without `-codeowners=""` get the check; CI
   gets it by default; fixture-only test harnesses opt
   out via the empty value.

9. **B2-9 closes.** The decision-log B2-9 row moves to
   `resolved-adr` (→ this ADR). OQ-B1-9.3 stays open as
   the publisher/loader-side defense-in-depth follow-up.

10. **Two deferred items are explicitly reserved as
    out-of-scope** and registered for future B-items:
    per-entity CODEOWNERS-path cross-check (waits for
    ADR-0015 §11's additive per-entity CODEOWNERS lines
    to land), and email-reviewer support (waits for a
    future ADR amending ADR-0015 §2's group-only
    commitment).
