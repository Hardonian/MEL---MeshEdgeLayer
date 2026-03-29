# MEL privacy-first platform policy

## Scope
This policy governs MEL communications-hub infrastructure decisions and defaults.

## Non-negotiable defaults
- Self-hosted operation first (`platform.mode=self_hosted`).
- No hidden outbound telemetry.
- No forced cloud dependency for core operation.
- No silent fallback from local runtime to external managed APIs.
- Explicit degraded/unavailable state when optional providers are unavailable.

## Data ownership and retention
- Operator controls local retention windows.
- Evidence export remains enabled by default for audit portability.
- Delete semantics are explicit and controlled by local policy.

## Key and secret handling boundaries
- Key custody stays with deployment owner.
- Secrets must be provided via local environment/config management.
- MEL does not claim end-to-end secrecy beyond configured and verified crypto providers.

## Telemetry policy
- `platform.telemetry.enabled=false` by default.
- Any outbound telemetry requires explicit `allow_outbound=true` and visible operator intent.
- Telemetry state must remain machine-visible for audits and proofpacks.

## Assistive intelligence policy
- AI output is non-canonical helper text only.
- Incident truth, support status, evidence posture, and control lifecycle remain deterministic and typed.
- If AI runtime is unavailable, MEL remains fully functional with honest assist status (`unavailable`, `partial`, `queued`).
