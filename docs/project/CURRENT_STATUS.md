# MEL Current Status (RC1)

*Last updated: 2026-03-21*

MEL is currently in **Release Candidate 1 (RC1)**. It is stable for field testing and local-first observability but should be used with an understanding of its current scope and boundaries.

## 🟢 What is Real Today

- **Zero-Theatre Ingest**: Serial, TCP, and MQTT transports are fully implemented with reconnection logic.
- **Persistence**: High-fidelity SQLite storage of all Meshtastic packet types (Text, Position, Telemetry, etc.).
- **Authoritative Doctor**: `mel doctor` provides 15+ health checks for host, database, and transport layers.
- **Web Dashboard**: Real-time observability of mesh health, nodes, and action history.
- **Control Plane**: Reality-matrix evaluation that can suggest (Advisory) or execute (Guarded Auto) remediation actions for disconnected transports.
- **Privacy Core**: Automated redaction of sensitive data in exports and built-in privacy auditing.

## 🟡 What is Evolving

- **BLE / HTTP Transports**: Not yet supported.
- **Metrics Server**: Config exists, but a dedicated Prometheus-compatible listener is coming in 0.2.0.
- **Backup Restore**: `backup restore` exists but requires `--dry-run` validation only. Active production-state restoration is still being hardened.
- **Action Reversibility**: Not all remediation actions are currently reversible.

## 🔴 What is Intentionally Not Claimed

- **Radio Control**: MEL does not currently perform radio admin operations (e.g., changing channel settings via radio packets).
- **Global Routing**: MEL is a local-first layer; it does not participate in mesh routing or store-and-forward.
- **At-Rest Encryption**: MEL relies on OS-level file permissions and disk encryption; it does not encrypt the SQLite file contents itself.

## 🚀 How to Help

We are looking for field testers to:
1. Run `mel doctor` in diverse host environments (RPi, ARM, x86).
2. Verify MQTT ingest stability over high-latency links.
3. Test Control Plane "Guarded Auto" mode with flaky Serial/TCP connections.
