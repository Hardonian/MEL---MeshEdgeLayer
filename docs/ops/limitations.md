# Known Issues and Limitations

This document outlines known issues, intentional design limitations, and operational constraints in MEL. These are not bugs and will not be addressed as defect fixes unless explicitly noted in the [Roadmap](../roadmap/ROADMAP_EXECUTION.md).

---

## By Design (Not Bugs)

| Limitation | Description | Rationale |
|------------|-------------|-----------|
| **No Radio Transmission or Administration** | MEL does not transmit mesh packets or administer radio parameters. It observes and records telemetry only. | Scope boundary: MEL is a monitoring and control plane, not a mesh stack. |
| **BLE Transport Explicitly Unsupported** | Bluetooth Low Energy is not implemented as a transport option. | Architectural decision: BLE mesh implementations vary significantly; support would require per-vendor integration. |
| **HTTP Transport Ingest Not Present** | MEL cannot receive telemetry via HTTP POST/PUT. | Design choice: MQTT and direct serial/TCP are the only supported ingest paths. |
| **Restore is Validation-Only** | The `restore` command requires `--dry-run` and performs validation without applying changes. | Safety mechanism: Prevents accidental bulk restoration of stale or incompatible configurations. |
| **Advisory-Only Control Actions** | Some control plane actions have no actuator backing and serve as recommendations only. | Implementation gap: Hardware actuators may not exist on all target nodes. |

---

## Hardware & Verification Limitations

- **Serial/TCP Hardware Variants**: Direct connections to nodes may behave differently across hardware variants. MEL is verified against reference hardware only.
- **Hardware Endurance**: Claims about flash endurance, temperature ranges, or power consumption are based on datasheet specifications, not field testing.
- **RS485 Bus Wiring**: Signal integrity on RS485 half-duplex wiring (termination, biasing, grounding) is not verified by MEL.
- **Spectrum Proof**: MEL cannot prove spectrum health or RF coverage from repo-local tests alone.

---

## Operational Limitations

- **Raw Bytes Telemetry Storage**: Telemetry payloads are stored as raw bytes. Operators must implement their own deserializers if specific payload parsing is not already vendored.
- **Hybrid Multi-Transport Verification**: When combining MQTT and direct serial, duplicate message detection requires operator-level verification.
- **MQTT QoS**: QoS 1/2 guarantees depend strictly on the broker implementation.
- **Reconnect Behavior**: Reconnection logic is bounded (exponential backoff) but not guaranteed in all network conditions.

---

## Control Plane Limitations

- **GuardedAuto Actuator Dependency**: The `guarded_auto` mode only executes actions when a working actuator is confirmed present.
- **Source Suppression is Advisory**: Source suppression commands are logged but not enforced at the mesh level.
- **Routing Changes are Advisory**: Routing recommendations are logged but not verified against actual routing table updates.
- **Mesh-Level Actions Disabled**: Global config pushes are disabled by default to prevent widespread accidental impact.

---

## Performance Considerations

- **SQLite Suitability**: SQLite is for edge deployments. Use an external database if you exceed **1000 writes/sec** or **10GB** data volume.
- **Export Record Limit**: Export endpoints are limited to the last 250 records per category to prevent memory exhaustion.
- **Pagination**: History queries use pagination. Unbounded queries are rejected.

---

## Privacy Constraints

- **Encryption Validation Only**: The `storage.encryption_required` setting validates environment variables only; it does not enable at-rest database encryption.
- **Position Redaction**: Redaction is applied during export operations. Raw positions may still exist in the database unless specifically purged.
- **Trust List Filtering**: Filtering based on trust lists (allowlist/blocklist) is not yet implemented for telemetry ingest.

---

## Reporting New Issues

If you encounter behavior not documented here that appears to be a defect:
1. Search existing issues in the repository.
2. Document reproduction steps with hardware/firmware versions.
3. Include logs and configuration (sanitized).
4. File an issue in the appropriate tracker.

**Do not file defects for limitations documented in this file.**
