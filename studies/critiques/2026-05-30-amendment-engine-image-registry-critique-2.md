<!-- path: studies/critiques/2026-05-30-amendment-engine-image-registry-critique-2.md -->

# B2-36 — Critique Round 2

## Metadata

- **Target study:**
  [`studies/decisions/2026-05-30-amendment-engine-image-registry.md`](../decisions/2026-05-30-amendment-engine-image-registry.md)
  (post-round-1-revision state).
- **Round:** 2 (final per
  [`.claude/playbooks/wave-1-session-loop.md`](../../.claude/playbooks/wave-1-session-loop.md)
  step 7 — two-round cap).
- **Date:** 2026-05-30.
- **Preservation status:** preserved (this file is the
  round-2 capture; round 2 is the final round under the
  two-round cap).
- **Closing commit hash:** filled in at the close-PR step
  of the post-Wave-3 session loop.

## Critique Output

```text
Adversarial review of the round-1-revised study.
Grounding: CLAUDE.md §3 / §4, acceptance-criteria.md,
feedback-protocol.md, the round-1 capture's Operator
Response trailer to confirm dispositions landed.

[no-blocking] No R1, R2, R3, R5, R6, R7, R8 violation
found. The round-1 important findings landed cleanly:
 - F1: "not on industry preference" removed; D1 +
   §Considered Options framing lean on
   "this project's specific operational context".
 - F2: D6 dropped GKE / GitHub Enterprise / AWS
   migration enumeration; reads as a future-session
   anchor only.
 - F3: push-step contract drops DOCKERHUB_PAT /
   DOCKERHUB_USERNAME naming.
 - F4: push-step contract drops docker login + docker
   buildx tool-invocation pre-commits; describes
   behavior at the contract level.
The four minor findings landed cleanly too.

[important] P5: "Recommendation §Push-step contract item
4 — 'preserving ADR-0042 Clause 1's
readOnlyRootFilesystem + non-root posture even if a tag
is later overwritten upstream'" — the digest-pinning
rationale is correct but the *invocation* of ADR-0042
Clause 1's security posture in the push-step contract
overreaches. Clause 1's posture binds the *runtime
image*: non-root user, readOnlyRootFilesystem,
RuntimeDefault seccomp, dropped capabilities. Tag
overwrite (where the registry holds the tag but the
underlying manifest digest changes) breaks
*reproducibility*, not the runtime security posture
directly — the swapped image would still run as nonroot
on a readOnlyRootFilesystem unless its base layer
changed. The accurate rationale is "digest pinning
preserves *reproducibility*: the deployment manifest
pins to the exact bytes the CI lane built, so a
tag-overwrite at the registry layer cannot silently
swap the image under the deployment." Suggest: rephrase
contract item 4's rationale to commit *reproducibility*
(the load-bearing property digest pinning preserves),
not Clause 1's runtime posture (which the pod's
securityContext enforces independently of the digest).

[minor] AC-2: "Decision Drivers D6 — implicit
four-option-exhaustiveness" — D6 says "the future
session reads the four options and either confirms
Docker Hub or shifts to a different substrate via its
own amendment ADR." This implies the option space is
exhaustive at four. The future session may surface a
fifth (a self-hosted registry; a registry that did not
exist at this writing). Pure style; the framing is
descriptive of *this study's* option enumeration, not
prescriptive of the future. Could be tightened to "the
future session reads §Considered Options as a starting
point — it may confirm Docker Hub, shift to one of the
other three substrates, or expand the option space."

[minor] AC-6: "Open Questions OQ-4 — public vs.
private namespace" — OQ-4 marks this deferred to the
follow-on B-row. Operator pre-declared the
`fabiocaffarello` namespace without specifying
public/private. The default assumption matters for
B-row planning (a private namespace forces image-pull-
secret distribution to every consuming cluster).
Suggest: note in OQ-4 that the *default assumption* is
public unless the operator declares otherwise at the
B-row's PAT setup; this clarifies operator intent
without forcing a decision here.

[minor] AC-2: "Recommendation §What the amendment does
NOT change — overlap with P-AmRR.4 and D4" — the
"What the amendment does NOT change" list at the end
of §Recommendation duplicates several items already
covered in P-AmRR.4 and D4 (ADR-0042 Clauses 2-4
preserved; local-build unchanged). The duplication
helps reviewers locate boundaries in one place, even
if individual items appear elsewhere. Accept-as-is —
the final consolidation list is useful.

Acceptance criteria sweep (post-round-1-revision):
- AC-1 (path header): pass.
- AC-2 (required sections in order): pass.
- AC-3 (≥2 options): pass — four options A/B/C/D.
- AC-4 (Recommendation grounded): pass.
- AC-5 (no external naming as justification): pass.
- AC-6 (Open Questions marked): pass — OQ-1..OQ-4
  carry out-of-scope markers; minor F14 tightening
  noted for OQ-4.
- AC-7 (Promotion target concrete): pass.
- AC-8 (≥1 critique round): pass — round 1 preserved;
  round 2 will be on commit.
- AC-9 (blocking findings addressed): N/A — no blocking
  in either round.
- AC-10 (decision-log row updated): pending step 9 of
  the loop; not yet a violation.

Summary: 0 blocking / 1 important / 3 minor. The single
important finding (push-step contract item 4 cites
Clause 1's runtime posture where it should cite
reproducibility) is the material gap surfaced by the
round-1 revision. No round 3 under the two-round cap —
the finding lands in this revision and the study moves
to resolved-study after operator dispositions the
round-2 trailer.
```

## Operator Response

- **[important] P5 push-step contract item 4 — wrong
  property cited** — *applied as recommended*: item 4's
  rationale will rewrite to commit *reproducibility*
  (the property digest pinning preserves) instead of
  ADR-0042 Clause 1's runtime posture (which the pod's
  securityContext enforces independently of the
  digest).

- **[minor] AC-2 D6 four-option-exhaustiveness** —
  *applied as recommended*: D6 tightens to "§Considered
  Options provides a starting point — the future
  session may confirm, shift, or expand the option
  space."

- **[minor] AC-6 OQ-4 public-default assumption** —
  *applied as recommended*: OQ-4 notes the *default
  assumption* is public namespace unless the operator
  declares otherwise at the B-row's PAT setup.

- **[minor] AC-2 §Recommendation "What the amendment
  does NOT change" list overlap** — *accepted as-is*:
  the final consolidation list helps reviewers locate
  boundaries in one place.

Two-round cap reached. Round-2 dispositions land in the
next commit on this branch; the study moves to
`resolved-study` after the decision-log row update.
