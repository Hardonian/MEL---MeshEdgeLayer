# ADR-0002: Runtime layers and truthful RC1 scope

## Status
Accepted

## Decision
MEL RC1 keeps one daemon and one CLI. MQTT remains the only production-claimed transport. Unsupported transports stay visible but explicitly unsupported.

## Consequences
- Operators get a smaller and easier-to-review runtime.
- CLI, API, and UI reuse the same privacy and policy primitives.
- Future transport work must earn support status through code and tests before docs can promote it.
