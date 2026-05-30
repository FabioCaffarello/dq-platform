<!-- path: studies/critiques/2026-05-30-amendment-engine-image-registry-critique-1.md -->

# B2-36 — Critique Round 1

## Metadata

- **Target study:**
  [`studies/decisions/2026-05-30-amendment-engine-image-registry.md`](../decisions/2026-05-30-amendment-engine-image-registry.md).
- **Round:** 1.
- **Date:** 2026-05-30.
- **Preservation status:** preserved (this file is the
  round-1 capture; round-2 capture will be a sibling
  `-critique-2.md` if a second round runs).
- **Closing commit hash:** filled in at the close-PR step
  of the post-Wave-3 session loop.

## Critique Output

```text
Adversarial review of
studies/decisions/2026-05-30-amendment-engine-image-registry.md.
Grounding: CLAUDE.md §3 (R1-R8), §4 (P1-P6),
acceptance-criteria.md (AC-1..AC-10), feedback-protocol.md,
ADR-0049 §(a) outcome taxonomy.

[no-blocking] No R1, R2, R3, R6, R7 violation found.
R3 (do not revisit settled architecture) honored
explicitly — ADR-0042 Clauses 2-4 named as preserved
unamended; only Clause 1 image-name row + Consequence 10
OQ-1 touched. R4 stays inside scope (study only). R6
path header on line 1. R8 N/A pre-promotion.

§(a) outcome classification reads as a clean Amendment
per the Metadata block — the test ("does this change
what the ADR decided?") returns yes for the image-name
shape row. B3 / B2-standalone / Rejected ruled out with
specific reasons. Decision-log placement on B2 with
B2-36 number follows the ADR-0050 precedent
(amendment-class outcome riding a B2 row).

[important] R5: "Considered Options framing — 'not on
industry preference'" — the opening line of §Considered
Options says "The comparison is on this project's terms
(cost surface, auth posture, OIDC/CI integration,
image-path shape), not on industry preference." The
phrase "not on industry preference" subtly references an
external default it positions against. R5 prefers that
the study describe what it does on this project's terms
without referencing external comparison shapes — the
"this project's terms" framing stands on its own.
Suggest: drop "not on industry preference" or rephrase
to "framed in this project's specific operational
context."

[important] R5 / P5: "Decision Drivers D6 — Future
re-platforming has a documented starting point" — D6
enumerates concrete vendor migration paths ("re-platforms
onto GKE workload", "onto GitHub Enterprise", "onto
AWS"). This is forward-positioning that the amendment
ADR does not need: the amendment commits the current
choice; future amendments handle future choices. D6's
enumeration also borders R5 by naming specific vendor
products as future destinations. Suggest: tighten D6 to
"If a future session re-opens the registry choice,
§Considered Options provides a documented option-space
starting point" — the substrates are already named in
§Considered Options; D6 does not need to repeat the
enumeration.

[important] P5: "Recommendation §Push-step contract —
pre-commits specific secret names" — items 1 and 4 of
the push-step contract pre-commit implementation
details: "suggested secret name: DOCKERHUB_PAT; the
username secret is DOCKERHUB_USERNAME, expected value
fabiocaffarello." Per P-AmRR.2 the wiring is deferred
to the follow-on B-row; secret naming is wiring, not
contract. The contract should say "the operator supplies
the PAT via a CI secret"; the B-row picks the names.
Suggest: drop the specific secret-name suggestions from
the contract; let the B-row decide.

[important] P5: "Recommendation §Push-step contract —
over-specifies the push tool invocations" — items 1
and 2 prescribe specific tool commands ("docker login
with -u and --password-stdin", "docker tag + docker
push, or a single docker buildx build --push step").
This is mechanical implementation detail that belongs
in the implementation slice. The contract should say
*what* must happen (authenticated push of the
locally-built image to the chosen path on tagged
builds, with digest surfaced for downstream pinning),
not *how*. Suggest: simplify items 1-4 to describe the
desired behavior at the contract level without
prescribing the tool invocations.

[minor] AC-5: "Considered Options Option C — AWS ECR
as commodity substrate" — AWS as a public cloud is not
literally on R5's exemption list ("BigQuery, Kafka, GCS,
Pub/Sub, OIDC, Prometheus, OpenTelemetry, Kubernetes,
Go, Docker, slog, JSON Schema, and equivalents"). It is
arguably "equivalent" to GCP under the "and equivalents"
clause, but the equivalence is implicit. The mention is
descriptive of an option-space substrate, which is fine.
Could be tightened with a one-line note in the framing
of §Considered Options acknowledging that the four
registry substrates are commodity environment per R5's
"and equivalents" clause.

[minor] AC-2: "Locked premises P-AmRR.2 — scope omits
the deployment-manifest flip shape" — P-AmRR.2 says
"Scope: registry choice + image-name pattern + push-step
contract only." But §Recommendation also commits the
deploy/base/deployment.yaml flip shape (the eventual
published-image reference). Either include the flip
shape in P-AmRR.2's scope or move it out of §Recommendation.
Suggest: extend P-AmRR.2 to "Scope: registry choice +
image-name pattern + push-step contract + deployment-
manifest flip shape (the eventual published-image
reference shape); the actual placeholder flip itself
deferred to the follow-on B-row."

[minor] AC-2: "Decision Drivers D5 — overlap with D2"
— D5 ("deploy/base/deployment.yaml placeholder
semantics") restates much of D2 (image-name pattern).
The two are reconcilable but the duplication adds
length. Suggest: keep them separate (D2 = image-name
shape; D5 = deployment manifest flip target) but tighten
D5 to focus on the placeholder flip target without
re-explaining D2's published shape.

[minor] P5: "Consequences §6 — premature B2-37
numbering" — Consequence 6 says "A new B-row (B2-37 —
successor to this study's B2-36 amendment row)". The
follow-on B-row number is operator-reserved at
registration time per ADR-0051 Clause 7's reservation
discipline (analogous to ADR-number reservation). Pre-
committing "B2-37" in this study collides with parallel
sessions that may register B2 rows before this PR
merges. Suggest: change "B2-37" to "a new B2 row
(number reserved operator-side at registration time)".

Acceptance criteria sweep:
- AC-1 (path header): pass.
- AC-2 (required sections in order): pass; minor
  tightening per F6/F7 noted.
- AC-3 (≥2 options): pass — four options A/B/C/D.
- AC-4 (Recommendation grounded): pass — cites ADR-0042,
  -0049, -0050, -0017, -0010, -0019, -0051, -0052, -0053
  + foundation docs.
- AC-5 (no external naming as justification): pass with
  [important] F1 (industry-preference phrasing) and
  [minor] F5 (AWS exemption clarity) noted.
- AC-6 (Open Questions marked): pass — OQ-1..OQ-4
  carry out-of-scope markers.
- AC-7 (Promotion target concrete): pass —
  docs/adr/0054-engine-image-registry-amendment.md with
  Status: accepted (amends ADR-0042) per A4.
- AC-8 (≥1 critique round): pass after this file
  commits.
- AC-9 (blocking findings addressed): N/A — no blocking
  in round 1.
- AC-10 (decision-log row updated): pending step 9 of
  the loop; not yet a violation.

Summary: 0 blocking / 4 important / 4 minor. The four
important findings cluster around §Considered Options
framing (F1 industry-preference phrase; F2 D6 vendor
enumeration) and §Recommendation push-step contract
(F3 secret names + F4 tool invocations pre-commit
implementation detail that belongs in the B-row).
Round 2 expected after the operator dispositions the
round-1 findings.
```

## Operator Response

- **[important] R5 industry-preference phrasing** —
  *applied as recommended*: §Considered Options framing
  will drop "not on industry preference" and lean on
  the "this project's terms" phrasing only.

- **[important] R5/P5 D6 vendor enumeration** —
  *applied as recommended*: D6 will tighten to "If a
  future session re-opens the registry choice,
  §Considered Options provides a documented option-space
  starting point" without enumerating specific
  re-platforming targets.

- **[important] P5 push-step contract secret names** —
  *applied as recommended*: §Recommendation push-step
  contract will drop the specific `DOCKERHUB_PAT` /
  `DOCKERHUB_USERNAME` naming and say "the operator
  supplies the PAT via a CI secret"; the B-row chooses
  names.

- **[important] P5 push-step contract tool invocations**
  — *applied as recommended*: contract items 1–4 will
  describe behavior (authenticated push, tagged-build-
  only, digest surfacing) without prescribing
  `docker login` / `docker buildx` invocations.

- **[minor] AC-5 R5 "and equivalents" clarity for AWS**
  — *applied as recommended*: §Considered Options framing
  will add a one-line note that the four substrates are
  commodity environment per R5's "and equivalents"
  clause.

- **[minor] AC-2 P-AmRR.2 scope extension** —
  *applied as recommended*: P-AmRR.2 will extend the
  scope to include the deployment-manifest flip shape
  (the eventual published-image reference shape), with
  the actual flip itself deferred.

- **[minor] AC-2 D5 tightening** — *applied as
  recommended*: D5 will focus on the placeholder flip
  target without re-explaining D2's published shape.

- **[minor] P5 Consequences §6 B2-37 pre-numbering** —
  *applied as recommended*: Consequence 6 will name "a
  new B2 row (number reserved operator-side at
  registration time)" instead of pre-committing B2-37.

All findings dispositioned. Revision follows in the next
commit on this branch.
