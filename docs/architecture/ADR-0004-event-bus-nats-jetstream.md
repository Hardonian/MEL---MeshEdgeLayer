# ADR-0004-event-bus-nats-jetstream: Internal event bus standardizes on NATS JetStream

## Status
Accepted

## Decision
Use NATS+JetStream for durable event flow and replay-friendly incident/event pipelines; MEL will not build a custom broker.

## Consequences
- Reduces lock-in and recurring spend for core MEL operation.
- Keeps operator-truth claims bounded to locally verifiable evidence.
- Preserves swap-friendly provider seams for future implementation changes.
