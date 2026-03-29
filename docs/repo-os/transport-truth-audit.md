# Transport Truth & Degraded-State Audit

Use for ingest/transports/status/mesh health/docs claims.

## Matrix correctness
- [ ] Direct ingest (serial/TCP) and MQTT support statements match implementation.
- [ ] BLE/HTTP ingest and other non-implemented paths are explicitly marked unsupported.
- [ ] No wording implies MEL performs RF routing/propagation proof.

## Evidence thresholds
- [ ] “Connected” requires runtime evidence, not config presence.
- [ ] “Receiving/active” requires persisted ingest evidence.
- [ ] “Healthy transport” requires freshness + error/dead-letter context.

## Degraded-state semantics
- [ ] Disconnect, reconnect, partial ingest, and backlog/dead-letter conditions are represented.
- [ ] Stale data is not merged into live status.
- [ ] Historical/imported records are labeled as non-live context.

## Failure transparency
- [ ] Transport errors include actionable source/context where possible.
- [ ] No fake packet/routing/coverage/flood success semantics are introduced.

## Test expectations
- [ ] Added/updated tests cover stale/live transitions.
- [ ] Added/updated tests cover partial ingest and transport disconnect handling.
