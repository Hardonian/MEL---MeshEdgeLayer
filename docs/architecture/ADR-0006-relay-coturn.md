# ADR-0006-relay-coturn: Relay/NAT traversal baseline is coturn

## Status
Accepted

## Decision
Use coturn for TURN/STUN capabilities; MEL exposes relay health and does not implement relay protocols itself.

## Consequences
- Reduces lock-in and recurring spend for core MEL operation.
- Keeps operator-truth claims bounded to locally verifiable evidence.
- Preserves swap-friendly provider seams for future implementation changes.
