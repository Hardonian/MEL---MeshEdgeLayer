# ADR-0010-privacy-telemetry-retention-defaults: Privacy defaults enforce self-hosted and explicit telemetry opt-in

## Status
Accepted

## Decision
Base mode is self-hosted with telemetry disabled, retention/export controls explicit, and no hidden outbound analytics fallback.

## Consequences
- Reduces lock-in and recurring spend for core MEL operation.
- Keeps operator-truth claims bounded to locally verifiable evidence.
- Preserves swap-friendly provider seams for future implementation changes.
