# Verification Matrix (MEL)

This matrix defines minimum verification obligations by work category.

## A) Ingest / transport / fleet truth
**Required (release-blocking):**
- Unit tests for parsing/state transitions.
- Integration tests for ingest worker behavior (disconnect/reconnect/partial ingest).
- Degraded-state assertions (stale, no-transport, dead-letter visibility).
- Mixed-channel posture assertions where applicable (channel/path/support/degraded/unknown).
- Docs + support-matrix alignment check.

**Advisory:**
- Manual smoke with representative transport config.

## B) API contract / backend semantics
**Required (release-blocking):**
- Unit tests for service handlers and state derivation.
- Contract tests or endpoint assertions for response fields and status semantics.
- Explicit assertions for observed/inferred/estimated/unknown fields where applicable.
- Error semantics checks (truthful 4xx/5xx mapping).

**Advisory:**
- Backward-compatibility notes for external integrations.

## C) UI truth rendering
**Required (release-blocking):**
- Component/page tests for live/stale/partial/imported/degraded/unknown distinctions.
- Tests for degraded banners/empty states.
- Verify UI does not infer certainty unavailable in API payload.

**Advisory:**
- Manual walkthrough screenshots for major operator-surface changes.

## D) Trusted control / approvals / execution
**Required (release-blocking):**
- Tests for lifecycle transitions and illegal transition rejection.
- Tests for approval-gated execution and audit events.
- Tests for failure path visibility (execution failed/cancelled/timeout).
- Per-action snapshot completeness truth asserted (`complete` / `partial` / `unavailable`) when evidence retrieval is degraded.

**Advisory:**
- Manual end-to-end action trace in dev environment.

## E) Security / trust boundaries
**Required (release-blocking):**
- Tests for authn/authz denial paths.
- Tests for cross-scope/tenant access rejection where applicable.
- Validation that exported evidence/redactions match policy.
- Validation that no new hidden telemetry/export path was introduced.

**Advisory:**
- Threat model delta note for boundary-affecting changes.

## F) Schema/config/runtime changes
**Required (release-blocking):**
- Migration determinism checks.
- Config validation tests and startup behavior checks.
- Runtime policy posture checks (`/api/v1/platform/posture`) for telemetry/export/delete/inference truth.
- Upgrade/rollback path notes if compatibility is affected.
- Runtime fallback truth checks (base functionality remains truthful without optional inference components).

**Advisory:**
- `sqlite3` schema inspection snippets in PR evidence.

## G) Documentation-only changes that affect claims
**Required (release-blocking for claim changes):**
- Cross-check against implementation and support matrix.
- Update limitations/caveats where wording changed operator expectations.
- Verify language aligns with terminology policy and no-theatre constraints.

## H) Evidence export / proofpack
**Required (release-blocking):**
- Unit tests for proofpack assembly logic (full, sparse, partial-failure cases).
- Evidence gap markers present for all degraded/missing evidence paths.
- Proofpack + snapshot completeness truth asserted (`complete` / `partial` / `unavailable`).
- Audit trail verification (proofpack export logged to RBAC audit + timeline).
- API contract check (correct HTTP status codes, capability enforcement).
- Policy-gate check for export/delete semantics (`platform.retention.allow_export/allow_delete`).

**Advisory:**
- Manual proofpack download from UI for a real incident.
- Schema version check in exported JSON.

## I) Local inference integration (when touched)
**Required (release-blocking when changed):**
- Tests/checks proving canonical truth does not depend on inference runtime success.
- Explicit fallback behavior verification (runtime unavailable, timeout, resource exhaustion).
- Assistive-output labeling checks (non-canonical wording).

**Advisory:**
- Performance snapshots for foreground vs background routing behavior.

## Standard command baseline
Run what is relevant and report failures honestly:
- `make lint`
- `make test`
- `make build`
- `make smoke`
- Frontend: `npm run lint`, `npm run typecheck`, `npx vitest run` (from `frontend/`)

If baseline commands fail due to known pre-existing issues, annotate clearly as residual risk; do not suppress.
