# Known limitations

- BLE transport is explicitly unsupported.
- HTTP transport is explicitly unsupported.
- MEL does not send packets or administer radios.
- MEL does not claim hardware validation for serial or TCP radios in this container environment.
- MEL cannot prove “100% reliability”; it can only expose retries, heartbeats, timeouts, dead letters, and stored-message evidence that actually occurred.
- Telemetry payloads are stored truthfully as raw payload evidence until a full vendored telemetry protobuf schema exists in-repo.
- Hybrid multi-transport deployments still require operator validation for duplicate behavior and radio ownership.
- Browser screenshot evidence for the updated UI was not captured in this environment because browser tooling was unavailable.
