# ADR-0002: Runtime layers and truthful RC1 scope

## Status
Accepted

## Decision
MEL RC1 keeps one daemon and one CLI. Production-claimed ingest now includes serial direct-node, TCP direct-node, and MQTT. Unsupported transports stay visible but explicitly unsupported, and control-path features stay undocumented until code and verification exist.

## Consequences
- Operators get a smaller and easier-to-review runtime.
- CLI, API, and UI reuse the same privacy and policy primitives.
- Transport support claims must track the real capability matrix rather than aspirational design notes.
- Future transport or control work must earn support status through code and tests before docs can promote it.
