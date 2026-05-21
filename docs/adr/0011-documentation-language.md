<!-- path: docs/adr/0011-documentation-language.md -->

# ADR-0011 — Documentation Language

- **Status:** accepted
- **Date:** 2026-05-21

---

## Context

The platform's documentation surfaces serve two distinct
audiences with different ergonomic needs:

- **Technical artifacts** — ADRs, JSON Schemas, READMEs in
  every workspace, code comments, and contract documents —
  are read by current contributors, future contributors, and
  external reviewers across linguistic boundaries. They need
  a stable single language.
- **Onboarding guides** — internal walkthroughs, getting-
  started notes for new team members — are read primarily by
  the immediate team. Their value comes from being
  accessible in the reader's first language.

Without a committed language policy, the two surfaces drift,
mixed-language artifacts proliferate, and the
study-to-ADR promotion step (forbidden from back-linking
into `studies/` by R8) becomes a translation task even when
the underlying study is already in the target language.

---

## Decision

### 1. Technical artifacts are written in English

The following artifacts are **English**:

- ADRs under `docs/adr/`.
- JSON Schemas under `engine/internal/dsl/schema/` and
  `rules/_schema/`.
- READMEs in every workspace.
- Code comments in every workspace.
- Contract documents (manifest schemas, `_owners.yaml`
  schemas, event payload schemas).
- Decision-log entries and similar structured records.

### 2. Onboarding guides may be in Portuguese, with a marker

Onboarding guides, getting-started notes, and similar
contributor-facing internal walkthroughs may be written in
Portuguese. Each such file must open with a one-line
**language marker** — for example:

```markdown
> Language: Portuguese (Brazilian)
```

The marker is the file's first non-header content. It exists
so a contributor opening a file in their browser, IDE, or
diff viewer immediately knows the language before reading.

### 3. Study-to-ADR promotion is a language-normalization step

The promotion step from `studies/decisions/<file>.md` to
`docs/adr/<NNNN>-<slug>.md` (forbidden from back-linking per
R8) is also a translation step when the source study
contains Portuguese. ADRs are technical artifacts and
therefore English.

---

## Consequences

1. **Default language is English.** Contributors writing any
   technical artifact use English unless the artifact's
   type explicitly permits Portuguese (the onboarding-guide
   carve-out).

2. **The language marker is a binary check.** Either an
   onboarding-guide file opens with the marker or it does
   not. A reviewer sees the omission immediately.

3. **Mixed-language technical artifacts are not permitted.**
   An ADR with one Portuguese paragraph is non-compliant
   regardless of the rest of the file.

4. **The promotion step has a language obligation.** A
   study written partly in Portuguese must be normalized to
   English when promoted to an ADR. The
   `/promote-to-adr` skill (or its successor) should
   surface this obligation.

5. **Verbatim citation of a Portuguese user directive in an
   English technical artifact is permitted with an inline
   English translation alongside.** This is how a study or
   ADR records that the original input was Portuguese
   without violating the English-default for technical
   artifacts.

---

## Notes

- Whether the language-marker convention is enforced by a
  CI lint or remains advisory is a follow-up CI design
  item. The default until a Portuguese onboarding guide
  actually exists is advisory; lint enforcement becomes
  worthwhile once the surface area is non-empty.
- Future languages beyond English and Portuguese (Spanish,
  French, etc.) follow the same model: technical artifacts
  in English; non-technical artifacts in the local language
  with a marker. Adding a new local language does not
  require reopening this ADR.
