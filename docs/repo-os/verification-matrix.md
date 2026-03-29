# Verification Matrix (MEL)

This matrix defines minimum verification obligations by work category.

## A) Ingest / transport / fleet truth
**Required (release-blocking):**
- Unit tests for parsing/state transitions.
- Integration tests for ingest worker behavior (disconnect/reconnect/partial ingest).
- Degraded-state assertions (stale, no-transport, dead-letter visibility).
- Docs + support-matrix alignment check.

**Advisory:**
- Manual smoke with representative transport config.

## B) API contract / backend semantics
**Required (release-blocking):**
- Unit tests for service handlers and state derivation.
- Contract tests or endpoint assertions for response fields and status semantics.
- Error semantics checks (truthful 4xx/5xx mapping).

**Advisory:**
- Backward-compatibility notes for external integrations.

## C) UI truth rendering
**Required (release-blocking):**
- Component/page tests for live/stale/partial/imported distinctions.
- Tests for degraded banners/empty states.
- Verify UI does not infer certainty unavailable in API payload.

**Advisory:**
- Manual walkthrough screenshots for major operator-surface changes.

## D) Trusted control / approvals / execution
**Required (release-blocking):**
- Tests for lifecycle transitions and illegal transition rejection.
- Tests for approval-gated execution and audit events.
- Tests for failure path visibility (execution failed/cancelled/timeout).

**Advisory:**
- Manual end-to-end action trace in dev environment.

## E) Security / trust boundaries
**Required (release-blocking):**
- Tests for authn/authz denial paths.
- Tests for cross-scope/tenant access rejection where applicable.
- Validation that exported evidence/redactions match policy.

**Advisory:**
- Threat model delta note for boundary-affecting changes.

## F) Schema/config/runtime changes
**Required (release-blocking):**
- Migration determinism checks.
- Config validation tests and startup behavior checks.
- Upgrade/rollback path notes if compatibility is affected.

**Advisory:**
- `sqlite3` schema inspection snippets in PR evidence.

## G) Documentation-only changes that affect claims
**Required (release-blocking for claim changes):**
- Cross-check against implementation and support matrix.
- Update limitations/caveats where wording changed operator expectations.

## Standard command baseline
Run what is relevant and report failures honestly:
- `make lint`
- `make test`
- `make build`
- `make smoke`
- Frontend: `npm run lint`, `npm run typecheck`, `npx vitest run` (from `frontend/`)

If baseline commands fail due to known pre-existing issues, annotate clearly as residual risk; do not suppress.
