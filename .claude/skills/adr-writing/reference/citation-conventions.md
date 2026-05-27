<!-- path: .claude/skills/adr-writing/reference/citation-conventions.md -->

# ADR — citation conventions

Three citation forms, used distinctly. Pick the form that matches the
load the citation bears.

## Form 1 — bare narrative `ADR-NNNN`

For contextual citations where the reader does not need to navigate
or to a specific clause. Example: `ADR-0002 commits and ADR-0003's
append-only model already require...` — from `docs/adr/0024-…md:48`.

## Form 2 — load-bearing link `[ADR-NNNN](./NNNN-slug.md)`

For citations the reader is expected to follow. The relative path
form (`./NNNN-slug.md`) keeps the link valid regardless of where
`docs/adr/` is hosted. Example, from `docs/adr/0035-…md:109`:

> Manifests declaring a dropped version cause loader failure per
> [ADR-0007](./0007-loader-scheduler-retry-failure-semantics.md) CC1.

## Form 3 — section precision `ADR-NNNN §"Section Name"`

For citing a specific clause inside a referenced ADR. The section
name follows the heading verbatim. Example, from `docs/adr/0023-…md`:

> ADR-0020 §"Decision (Partial-Wave-S gate)" requires the
> foundational triplet to be at `resolved-adr` before promotion.

The `§` is the standard separator. Quote the section name when it
contains spaces; bare form is acceptable for single-word section names.

---

## Scope-note — verbatim template

```markdown
**Scope note (added YYYY-MM-DD):** This ADR currently applies to set-oriented capability (realized over BigQuery). Record-oriented capability is addressed separately in Wave-S (see ADR-0020 forthcoming).
```

Source: `docs/adr/0002-run-identity-and-idempotency.md:8` (canonical
form; the date and the parenthetical may vary, but the four
structural beats — "currently applies to", source-mode declaration,
"addressed separately in", and the forward-pointer — are stable).

Variant used by ADR-0010 (multi-amend, multi-mode):

```markdown
**Scope note (added 2026-05-23):** This ADR's set-oriented capability rows apply to BigQuery-backed evaluation. Record-oriented event-stream capability is committed in [ADR-0028](./0028-kafka-substrate-row.md).
```

(Source: `docs/adr/0010-substrate-posture.md:8`.)

---

## Status — amendment idiom

Single amender:

```markdown
- **Status:** accepted (amends ADR-NNNN)
```

Source: `docs/adr/0017-substrate-posture-amendment.md:5`.

Multi-amender (the amended ADR records each amender separately, each
with a parenthetical naming what was amended):

```markdown
- **Status:** accepted; **amended in part by ADR-NNNN** (description of what was amended); **amended in part by [ADR-MMMM](./MMMM-slug.md)** (description of what was amended)
```

Source: `docs/adr/0010-substrate-posture.md:5`.

The amender ADR carries its own full body — it is not a footnote
inside the amended ADR.

---

## New-contribution marker — verbatim template

```markdown
No prior foundation document, charter clause, or ADR commits a position on [topic]. The [posture] this ADR commits is **new contribution requiring review** and is reviewed in this ADR.
```

Source: `docs/adr/0044-external-artifact-references.md:34-38`.

`CLAUDE.md` R5 requires this marker when an architectural claim has
no prior backing. AC-5 verifies it during critique. Do not use the
marker for incremental decisions — it loses signal.

---

## Provisional numbering caveat — verbatim template

For studies declaring an expected ADR number:

```markdown
- **Promotion target:** `docs/adr/NNNN-slug.md`
  (subject to the same numbering caveat ADR-0020 §"Per-item ADR
  numbering" carries — `NNNN` is descriptive of the planned sequence).
```

Source clause: `docs/adr/0020-wave-s-launch.md:172-175`.

> Per-item ADR numbering. B0-S1 through B0-S7 promote in order to
> `docs/adr/0021-…` through `docs/adr/0027-…` respectively, modulo
> shifts if an unrelated promotion lands between B0-S items. The
> expected sequence is descriptive, not reserved.

ADR numbers are assigned at promotion time, not at study time. A
study that declares `0028` may promote to `0029` if an unrelated ADR
lands first.
