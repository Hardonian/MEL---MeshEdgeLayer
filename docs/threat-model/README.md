# Threat Model

MEL treats the following as first-class risks:
- passive RF metadata observation
- captured node or leaked channel settings
- MQTT broker leakage or misconfiguration
- public map exposure
- insecure local web/API binds
- malicious or flaky relays
- sybil-like spammy identities
- overshared retention and location history
- jurisdiction-specific ham radio encryption constraints

Controls in v0.1:
- localhost default binds
- privacy audit findings
- retention worker
- explicit empty states
- no fake routing claims
