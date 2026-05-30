<!-- path: studies/critiques/2026-05-30-b3-6-record-runner-commit-after-dispatch-critique-1.md -->

# Critique — B3-6 record-runner commit-after-dispatch (round 1)

- Target: [`studies/decisions/2026-05-30-b3-6-record-runner-commit-after-dispatch.md`](../decisions/2026-05-30-b3-6-record-runner-commit-after-dispatch.md)
- Round: 1
- Date: 2026-05-30
- Result: **0 blocking / 4 important / 5 minor**

---

## Findings

**No `blocking` findings.** The study clears the R1–R8 / P1–P6
floor and the AC-1…AC-10 surface. The borderline reading flagged
in Condition 1 of the eligibility table is the only D0 surface
and it is registered explicitly per R5 + adr-writing A7 with
precedent citations, which is the correct disposition — not a
blocking violation.

---

### Important

- **[important] AC-4 / R5: "Recommendation" — the "new
  contribution proposed here, requires review" marker lives only
  in the eligibility table cell (Condition 1) and a parenthetical
  in `Consequences §1`. The Recommendation section itself names
  ADR-0002 / ADR-0024 / ADR-0003 as grounding but does not surface
  the R5 marker.** Add a one-line "**Reading carried forward as
  new contribution proposed here, requires review (R5) — see
  eligibility Condition 1.**" inside Recommendation so a reviewer
  reading the section in isolation sees the disposition without
  scrolling back to the eligibility table.

- **[important] P4: "Considered Options → Option B" — the
  per-trigger commit RPC rate is qualitatively bounded ("the cost
  is bounded by window-close rate") but not quantitatively
  scoped.** P4 (cost as first-class) warrants a concrete bound:
  "one commit RPC per closed window per entity; at the ADR-0024
  minimum `window.duration: 1s` and N concurrent record-mode
  rules, the upper bound is N RPCs/second to the broker; at
  typical durations (1m–5m) the rate is N/minute or lower." A
  reviewer scanning for cost-discipline language needs the number,
  not just the shape.

- **[important] DD-2 / P2: "Decision Drivers → DD-2" — the
  determinism argument leans on ADR-0024 + ADR-0003 but does not
  name the specific clause of ADR-0003 that the at-least-once
  promise depends on.** Cite ADR-0003 §"first-write-wins" (or the
  exact clause name in ADR-0003) so a reader can verify the dedup
  substrate is the one the study claims, not a different
  write-model clause. Without the named clause, DD-2 reads as an
  assertion rather than a citation.

- **[important] AC-6 / OQ-4: "Open Questions → OQ-4" — the
  commit-failure metric question is correctly marked out-of-scope,
  but the one-line reason ("the emission slice and this slice are
  independent; either ordering is admissible") is weaker than the
  AC-6 standard.** Either restate the reason as the
  operationally-shaped deferral ("a commit-failure counter is not
  load-bearing for the at-least-once correctness this slice
  commits; the broker-retention-bounded re-flow path is the
  recovery mechanism, not the metric") or fold OQ-4 into a forward
  pointer to a specific B-row category (e.g., "lands when a
  follow-up emission slice resolves panel-5 OQs in ADR-0056"). The
  current wording leaves the deferral motivated by ordering, not
  by scope.

### Minor

- **[minor] R5: "Context → What `kafka_consumer.go` already does
  right" + "Considered Options → Option B" — the name "franz-go"
  appears in five locations as the named library.** R5's
  environment exemption admits commodity infrastructure (Kafka,
  slog, JSON Schema, etc.); a specific Kafka client library sits
  on the borderline. The mentions are factually grounded (the
  existing import is `github.com/twmb/franz-go/pkg/kgo`) and
  necessary for implementation discussion, but two of the five
  could be generalized to "the Kafka client" / "the consumer
  library" without losing accuracy — specifically the line "On
  engine restart, franz-go connects, reads the broker's committed
  offset" (Context) and "franz-go's `MarkCommitRecords` computes
  the high-water mark per partition" (What this study does NOT
  commit). The Option B paragraph naming the franz-go API
  primitives is necessary; keep that one.

- **[minor] AC-3: "Considered Options → Option C" — Option C is
  "status quo + defer", which is a genuine option but reads as a
  strawman because the Context section already argues that the
  β-marker comment makes deferral untenable.** Either tighten
  Option C's framing (e.g., "Option C — defer with operator-facing
  documentation"; the operative trade-off is the documentation
  discipline, not the deferral itself) or accept that Option C
  exists primarily to satisfy AC-3's three-option pattern.

- **[minor] AC-2: "Decision Drivers → DD-5" — DD-5 is a single
  load-bearing sentence followed by the implementation
  alternative; the bullet would read more cleanly split into a
  one-sentence headline ("`Record` has no `Topic`; commit needs
  `(topic, partition, offset)`") and a second sentence explaining
  the parallel-field minimal-blast-radius choice.** Current
  paragraph density obscures that DD-5 is doing two things:
  stating a constraint *and* committing the implementation shape.

- **[minor] AC-2 / R8: "What this study does NOT commit → bullet
  5" — the bullet about `record-mode-conventions` skill update
  sits inside "What this study does NOT commit" but is actually
  committing the skill-side update ("S2 — β commit semantics — is
  updated in the same PR to drop the β-marker and point to
  ADR-0058"). This is a what-the-study-DOES-commit item
  miscategorized.** Move it to Consequences §6 (which already
  mentions the skill update) and remove the duplicated mention
  from "What this study does NOT commit", or rephrase it as "this
  study does not commit the skill-side wording — only that the S2
  convention update lands in the same PR."

- **[minor] P2: "Considered Options → Option B → Weaknesses" —
  the line "Records-only-in-partitions-with-only-late-drops never
  commit (edge case; record as OQ)" describes OQ-2 by content but
  does not cross-reference OQ-2 by name.** Add "(see OQ-2)" so the
  forward link is explicit.

---

## Acceptance criteria roll-up

| AC | Pass? | Note |
|---|---|---|
| AC-1 | yes | Path-header present. |
| AC-2 | yes | All sections present; ordering correct. Two `[minor]` density / categorization findings. |
| AC-3 | yes | Three options; one `[minor]` about Option C's strength. |
| AC-4 | yes | Grounded in ADR-0002/0024/0003; one `[important]` about surfacing the R5 marker in Recommendation. |
| AC-5 | yes | One `[minor]` on franz-go-mention density. |
| AC-6 | yes | One `[important]` about OQ-4 deferral rationale. |
| AC-7 | yes | Promotion target points at `docs/adr/0058-record-runner-commit-after-dispatch.md`. |
| AC-8 | in-progress | This is round 1. |
| AC-9 | n/a | This round; revision applies findings. |
| AC-10 | pending | Decision-log update is step 9 of the loop. |

## Summary

**0 blocking / 4 important / 5 minor.** Recommended disposition:
apply the 4 important findings in the revision; defer 4 of 5
minors (Option C reframing optional; the two AC-2 density findings
optional; the OQ-2 cross-reference is cheap and worth folding in).
The R5 franz-go-mention `[minor]` is worth a one-pass
generalization on the two non-load-bearing mentions. After
revision, the study is ready to advance to `resolved-study` and
promote to ADR-0058.
