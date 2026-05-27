<!-- path: docs/adr/0047-lint-substrate-access.md -->

# ADR-0047 — `dq-lint` Substrate-Access Posture

- **Status:** accepted
- **Date:** 2026-05-27

---

## Context

[ADR-0040](./0040-entity-onboarding-workflow.md) §"Tier 1
→ Tier 2 criterion 4" commits **channel reachability**
as a manual verification step in the onboarding workflow:
each channel reference in
`_owners.yaml.entities.<entity>.channels` is manually
checked to confirm the Slack channel exists and is
writable from the engine's poster, the email address
accepts mail, the PagerDuty service exists. The check
is manual because the linter (`dq-lint`) is the natural
home for such automation, but the linter today is a
**unit-no-substrate** binary per
[ADR-0034](./0034-local-testing-strategy.md) §"Six-tier
test-type taxonomy" — no network access, no substrate
dependencies, runs anywhere `go` is available.

ADR-0040 §"Open Questions OQ-2" registered the linter
extension that would automate criterion 4 as a B2
follow-up but deferred it on two grounds: (a) the
manual check is adequate for v1 onboarding cadence, and
(b) extending the linter into substrate access requires
**committing a substrate-access posture** that ADR-0034
deliberately did not provide for tools-tier binaries.

What this ADR commits is that posture: when can the
linter reach external substrates, what is the credential
model, what are the failure semantics, where does CI
gate it. The actual reachability-checking implementation
remains **deferred** — implementation lands as a separate
B2 follow-up paced by concrete operator signal that the
manual criterion 4 check has become cumbersome at the
platform's onboarding cadence.

Existing commitments this ADR builds on:

- ADR-0034 §"`unit-no-substrate`" commits the default
  posture for `tools/lint` (no substrate; runs anywhere).
  This ADR does NOT amend that default — the default
  stays the default.
- ADR-0034 §"`integration-sandbox`" commits the
  substrate-access tier for tests that need real
  substrate fidelity. The linter's reachability mode is
  conceptually adjacent (real-substrate access, opt-in,
  separate CI lane); this ADR borrows the
  integration-sandbox tier's posture for the linter
  binary.
- ADR-0040 Tier 1 → Tier 2 criterion 4 commits the
  reachability check as manual at v1. This ADR's
  follow-up (when the actual implementation lands)
  would amend criterion 4 to make the linter run the
  reachability check; criterion 4 stays manual until
  then.
- [ADR-0006](./0006-alert-routing-contract.md) commits
  the `channels` field shape on `_owners.yaml` (`<type>:<id>`
  per-channel reference). This ADR cites that shape as
  the input the reachability check parses; it does not
  amend the shape.

The principles bearing on the decision are **P3**
(ownership is explicit — operators own credential
provisioning; the platform owns the linter contract),
**P4** (cost is a first-class constraint — gating the
default PR lane on substrate-access checks would amplify
PR latency and substrate API quota), and **P5**
(evolution must be contract-driven — the posture is a
documented contract operators can rely on when wiring
the reachability lane).

---

## Decision

### Default posture — unchanged from ADR-0034

`dq-lint` continues to run as a **unit-no-substrate**
binary by default. No flag is set; no env vars are
read; no network access happens; no credentials are
required. The binary is callable from anywhere `go` is
available and produces the same exit codes regardless
of substrate availability.

The default-posture commitment is **load-bearing**:
operators wiring lint into their pre-commit hooks, IDE
integrations, and PR checks rely on the binary being
deterministic, offline, and fast. The reachability
extension does NOT change this default.

### Extended posture — opt-in reachability mode

A new CLI flag `-check-channel-reachability` (or
equivalent) opts into the extended posture. When the
flag is set:

- The linter walks every channel reference in the
  loaded `_owners.yaml` and dispatches per channel
  type to the matching reachability adapter.
- Each adapter performs ONE non-mutating check per
  channel: HTTP GET against the Slack
  `conversations.info` API for `slack:` channels; SMTP
  DNS lookup (no message sent) for `email:` channels;
  HTTP GET against the PagerDuty
  `/services/{id}` API for `pagerduty:` channels.
- The check is **best-effort and per-channel**. A
  failed lookup on one channel does not abort the
  walk; every channel produces an outcome.
- Outcomes are **warnings**, not validation errors.
  Channels might fail temporarily — a 30-second blip
  in Slack should not block a PR. The linter's exit
  code is unaffected by reachability outcomes.
- Channels of unknown type (`<type>:<id>` where
  `<type>` is not in the adapter registry) are reported
  as `skipped — unknown channel type` warnings, not
  errors. The catalog of supported channel types is
  the adapter registry; adding a new type is a
  follow-up code change.

### Per-substrate adapter model

The linter contains one adapter per channel `<type>`.
At this contract level the committed adapters are:

| Channel type | Adapter behavior | Credential env var |
|---|---|---|
| `slack:` | HTTP GET `https://slack.com/api/conversations.info?channel=<id>` with bot-token auth; `ok: true` + matching channel name = pass | `DQ_LINT_SLACK_TOKEN` |
| `email:` | DNS MX lookup on `<id>`'s domain; one or more MX records = pass | (none — DNS-only) |
| `pagerduty:` | HTTP GET `https://api.pagerduty.com/services/<id>` with API-key auth; HTTP 200 = pass | `DQ_LINT_PAGERDUTY_KEY` |

The adapter set is **extensible** — new channel types
land additively as new adapters. The registry's
type-string mapping mirrors the catalog of supported
types for `dq_check_results.evidence_summary` and
similar grow-additively surfaces.

Adapters are **never mutating**. The reachability
check reads; it does NOT post test messages. A
"reachability" test message that lands in the channel
during an onboarding PR would be operationally
disruptive. The check confirms the channel exists +
the credential has the right scope to read it; the
authoritative "engine can post to this channel" check
lands at first-real-alert time.

### Credential model

Credentials are provided via environment variables, one
per substrate. The CI lane provides them via repository
secrets:

- `DQ_LINT_SLACK_TOKEN` — Slack bot token with
  `channels:read` scope.
- `DQ_LINT_PAGERDUTY_KEY` — PagerDuty API key with
  read access to the services in scope.
- (Email checks use only DNS; no credential.)

When a flag-set invocation finds the matching env var
absent, the adapter for that channel type **skips
cleanly** with a clear `skipped — credential absent`
warning. Skipping is NOT a failure — operators
exercising the lint flag locally without sandbox
credentials see a clean skip path.

This mirrors the integration-sandbox tier's posture
from ADR-0034 §"`integration-sandbox`" + B2-18: the
test skips when credentials are unset; the CI lane
provides them when configured.

### CI-lane separation

The reachability check is **NOT** wired into the
default PR lane. A separate workflow
(`.github/workflows/lint-reachability.yml` or
equivalent) runs the flag-set lint invocation. The
workflow's trigger posture mirrors
`.github/workflows/sandbox.yml` (B2-18):

- **`workflow_dispatch`** — operators trigger on
  demand when reviewing a PR that touches owners.yaml.
- **`schedule`** — weekly drift catcher; commented
  out at the ADR's acceptance, uncommented when the
  reachability lane stabilises.
- Gated on `vars.LINT_REACHABILITY_ENABLED == 'true'`
  or the presence of the matching secret(s).

The default PR lane is explicitly NOT gated on the
reachability lane because:

- **Substrate quota.** Slack and PagerDuty rate-limit
  API calls. Gating every PR on per-channel HTTP
  lookups multiplies quota consumption for no
  operational benefit (most PRs do not touch owners).
- **Flake risk.** Network calls fail transiently;
  gating the default lane on substrate availability
  amplifies CI flakiness.
- **Adequacy.** ADR-0040 criterion 4 is a Tier 1 →
  Tier 2 gate, not a PR-merge gate. The onboarding
  PR landing reachability is a single point in time;
  the manual check at promotion is adequate.

### Failure semantics

All reachability outcomes are **warnings**:

- `pass` — adapter confirmed the channel exists.
  Logged as informational.
- `fail` — adapter ran but the channel was not
  reachable (HTTP 404, no MX records, etc.). Logged
  as warning with the channel name + the adapter's
  observation. Exit code unchanged.
- `skipped — credential absent` — the adapter's env
  var was unset. Logged as warning. Exit code
  unchanged.
- `skipped — unknown channel type` — the channel
  type has no adapter. Logged as warning. Exit code
  unchanged.

The linter's overall exit code is determined entirely
by the existing validation rules (schema, owners,
cross-checks, catalog conformance). Reachability does
not vote.

Operators using the reachability lane treat warnings
as **review signal**, not gate signal. The PR reviewer
inspects the warning list and asks for fixes when
appropriate.

### Implementation — deferred

The actual `tools/lint` extension implementing the
reachability check + the CI workflow is **deferred**.
ADR-0040 OQ-2's framing — "follow-up when concrete
signal demonstrates the manual check is too
cumbersome" — applies. At the platform's current
onboarding cadence (two entities total: customer +
orders_stream), the manual criterion 4 check is
adequate.

A new B2 row registers the implementation slice. It
ships:

- The `-check-channel-reachability` flag on
  `tools/lint` (or a sibling subcommand).
- The Slack / email / PagerDuty adapters.
- The `.github/workflows/lint-reachability.yml`
  workflow (workflow_dispatch + commented schedule;
  gated on `LINT_REACHABILITY_ENABLED` var or secret
  presence).
- Unit tests for the adapters (against recorded /
  mocked HTTP responses).
- A new `make lint-reachability` target.

The new B2 row stays `open` until concrete operator
signal (e.g., an onboarding PR cycle where the manual
check was missed or repeatedly delayed) demonstrates
the implementation's value.

### Why this does not amend ADR-0034

ADR-0034 §"Six-tier test-type taxonomy" commits the
tier mapping for **tests**. The linter's reachability
mode is not a test tier — it is a tool's optional
extended posture. Adding "lint substrate access" as a
seventh test tier would over-fit; the linter is one
binary, not a test surface. Keeping ADR-0034's tier
inventory unchanged preserves its load-bearing
posture (operators reading the tier map for test
placement decisions see exactly six options).

The linter's default posture stays committed by
ADR-0034 §"`unit-no-substrate`"; this ADR commits
only the opt-in extension surface.

### Why this does not amend ADR-0040

ADR-0040 Tier 1 → Tier 2 criterion 4 stays at
"manual verification" until the implementation slice
lands. When that slice lands, an ADR-0040 amendment
(or a successor ADR) replaces "manual" with "linter
runs reachability per ADR-0047 in the
lint-reachability lane; reviewer treats warnings as
gate signal". Until then, the criterion is unchanged.

### Why this does not amend ADR-0006

ADR-0006 commits `<type>:<id>` as the per-channel
wire encoding. This ADR cites that encoding as the
input the adapter registry parses; it does not amend
the encoding. The adapter registry's per-type
discovery (which adapter handles `slack:` vs `email:`
vs `pagerduty:`) is **internal** to the linter and
not part of ADR-0006's commitment surface.

---

## Consequences

1. **`dq-lint` substrate-access posture is now an
   explicit contract.** The default-posture
   (unit-no-substrate, offline, deterministic) stays
   load-bearing. The extended posture (opt-in via
   flag, network access, per-substrate adapters,
   warnings-not-errors) is committed for the future
   implementation slice.

2. **No code change ships from this ADR.** The
   `tools/lint` binary is unchanged; the actual
   reachability extension defers to a B2 follow-up.

3. **Per-substrate adapter inventory is the
   contract.** Three adapters (Slack, email,
   PagerDuty) at the ADR's acceptance; new types
   land additively without ADR amendment as the
   channel-type registry grows.

4. **Credentials live in env vars per substrate.**
   `DQ_LINT_SLACK_TOKEN` and `DQ_LINT_PAGERDUTY_KEY`
   are the committed variable names; CI lanes
   provide them via repository secrets when
   configured. Email checks use only DNS (no
   credential).

5. **The CI lane is separate from the default PR
   lane.** A `lint-reachability` workflow with
   workflow_dispatch + (commented) schedule trigger
   runs the flag-set invocation; the default PR
   lane is unaffected.

6. **Failure semantics are uniformly warnings.**
   No reachability outcome influences the linter's
   exit code. Operators using the lane treat the
   warnings as review signal, not merge gate.

7. **The implementation is deferred to a new B2
   follow-up** paced by concrete operator signal
   that the manual ADR-0040 criterion 4 check has
   become cumbersome at the platform's onboarding
   cadence. Until then, criterion 4 stays manual.

8. **B2-26 closes.** The decision-log B2-26 row
   moves to `resolved-adr`. One new B2 row
   registers the implementation slice with `open`
   status.

9. **ADR-0034, ADR-0040, ADR-0006 are preserved.**
   This ADR layers a tool-posture contract on top
   of their commitments without amending them.

10. **Three deferred items are flagged out-of-scope:**
    additional channel-type adapters beyond the
    three at v1 (lands additively when new channel
    types are added to the catalog); a credential-
    less reachability mode that uses each
    substrate's public unauthenticated surface
    (most substrates don't have one; deferred until
    one materialises); per-adapter parallelism /
    rate-limiting (deferred until reachability lane
    cycle time becomes a concern).

---

## Notes

- The recommended `DQ_LINT_*` env-var prefix follows
  the platform's `DQ_*` convention (engine reads
  `DQ_ENV`, `DQ_LOG_LEVELS` per ADR-0043; the linter
  adopts the same prefix for its substrate-access
  credentials).
- Slack's `conversations.info` API and PagerDuty's
  `/services/{id}` endpoint were chosen because both
  are **read-only** and consume nominal rate-limit
  budget. Mutating endpoints (test-message posting,
  PagerDuty incident creation) are deliberately
  out-of-scope per §"Adapters are never mutating".
- The deferred implementation slice may choose to
  package the adapters as a separate Go module
  under `tools/lint/reachability/` or inline them in
  the existing `tools/lint/` module. The choice is
  an implementation detail; this ADR does not
  commit to a layout.
- A future amendment could ship the seventh test
  tier (`lint-reachability`) under ADR-0034 if the
  reachability lane grows into something resembling
  a proper test surface (multiple binaries, shared
  fixtures, dedicated runner). v1 keeps it as a
  tool extension, not a tier.
