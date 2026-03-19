# Transport interface contract

A transport is only considered supported in MEL when all of the following are true:

1. Real ingest code exists in `internal/transport/`.
2. The transport reports the shared state model.
3. `mel doctor` can describe or validate the transport path.
4. The transport is documented in operator docs.
5. Automated tests or smoke verification cover the path.

## Current interface expectations

Each transport implementation must:

- implement `Connect`, `Subscribe`, `Close`, `Health`, `MarkIngest`, and `MarkDrop`,
- expose a truthful `CapabilityMatrix`,
- surface explicit errors instead of swallowing them,
- avoid reporting ingest until the service confirms a SQLite write.

## Unsupported transport rule

If a transport does not meet the contract above, MEL must describe it as unsupported or partial. Do not add dead routes, stub success, or placeholder production code.
