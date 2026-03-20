# Known Issues and Limitations

This document outlines known issues, intentional design limitations, and operational constraints in MEL. Items listed here are not bugs and will not be addressed as defect fixes unless explicitly noted in the [Future Roadmap](#future-roadmap).

---

## By Design (Not Bugs)

| Limitation | Description | Rationale |
|------------|-------------|-----------|
| **No packet transmission or radio administration** | MEL does not transmit mesh packets or administer radio parameters. It observes and records telemetry only. | Scope boundary: MEL is a monitoring and control plane, not a mesh stack. Radio administration is the responsibility of the underlying mesh firmware. |
| **BLE transport explicitly unsupported** | Bluetooth Low Energy is not implemented as a transport option. | Architectural decision: BLE mesh implementations vary significantly; support would require per-vendor integration. |
| **HTTP transport ingest does not exist** | MEL cannot receive telemetry via HTTP POST/PUT. | Design choice: MQTT and direct serial/TCP are the supported ingest paths. HTTP would require additional server infrastructure. |
| **Restore is validation-only** | The restore command requires `--dry-run` and performs validation without applying changes. | Safety mechanism: Prevents accidental bulk restoration of stale or incompatible configurations. |
| **Advisory-only control actions** | Some control plane actions have no actuator backing and serve as recommendations only. | Implementation gap: Hardware actuators may not exist on all target nodes; actions are logged for operator review. |

### Workarounds for By Design Limitations

- **Packet transmission/radio admin**: Use the underlying mesh firmware's CLI or management interface (e.g., Meshtastic CLI, custom AT commands).
- **BLE transport**: Bridge BLE mesh traffic to MQTT using an external gateway (e.g., Home Assistant BLE-MQTT bridge, custom RPi bridge).
- **HTTP ingest**: Deploy a lightweight MQTT-to-HTTP bridge (e.g., Node-RED, custom Flask/FastAPI service) to translate HTTP POSTs to MQTT publishes.
- **Restore limitations**: Manually apply validated configurations through the control plane or direct node access.
- **Advisory actions**: Monitor action logs and implement manual or external automation to execute recommendations.

---

## Hardware Verification Limitations

| Limitation | Impact | Verification Status |
|------------|--------|---------------------|
| **Serial/TCP direct-node** | Direct serial or TCP connections to nodes may behave differently across hardware variants. | Verified in repo with reference hardware; not tested with all production hardware variants. |
| **Hardware endurance claims** | Claims about flash endurance, temperature ranges, or power consumption are based on datasheet specifications, not field testing. | Requires deployment-specific validation; operators should test in their target environment. |
| **RS485 bus wiring correctness** | RS485 half-duplex wiring (termination, biasing, grounding) correctness is not verified by MEL. | Operator must verify physical layer implementation; bus errors may manifest as dropped telemetry. |

### Workarounds for Hardware Verification Limitations

- **Serial/TCP direct-node**: Test direct connections in a staging environment with your specific hardware before production deployment.
- **Hardware endurance**: Implement health monitoring and proactive replacement schedules based on your environment's stress factors.
- **RS485 wiring**: Use a verified RS485 wiring guide; employ a bus analyzer or oscilloscope to verify signal integrity during deployment.

---

## Operational Limitations

| Limitation | Details |
|------------|---------|
| **Raw bytes telemetry storage** | Telemetry payloads are stored as raw bytes. Full schema parsing is not vendored in-repo; operators must implement their own deserializers. |
| **Hybrid multi-transport verification** | When using multiple transports (e.g., MQTT + direct serial), duplicate message detection requires operator-level verification. MEL does not deduplicate across transports. |
| **MQTT QoS guarantees** | QoS 1/2 guarantees depend on the broker implementation. MEL does not verify broker compliance. |
| **Reconnect behavior** | Reconnection logic is bounded (exponential backoff with max retry) but not guaranteed to succeed in all network conditions. |

### Workarounds for Operational Limitations

- **Raw bytes storage**: Maintain a schema registry external to MEL; use the export API to retrieve raw payloads for deserialization.
- **Hybrid multi-transport**: Implement message deduplication using sequence numbers or timestamps in your consumer pipeline.
- **MQTT QoS**: Use a verified broker (Mosquitto, RabbitMQ, EMQX) and monitor broker-side metrics for dropped messages.
- **Reconnect behavior**: Monitor connection state via health endpoints; implement external watchdog/restart logic if reconnection fails persistently.

---

## Control Plane Limitations

| Limitation | Description |
|------------|-------------|
| **guarded_auto actuator dependency** | The `guarded_auto` control mode only executes actions when a working actuator is confirmed present. Nodes without actuator support will not auto-execute. |
| **Source suppression is advisory** | Source suppression commands are logged but not enforced; there is no suppression actuator to block traffic at the mesh level. |
| **Routing changes are advisory** | Routing change recommendations are logged but not verified against actual routing table updates. No routing selector actuator is implemented. |
| **Mesh-level actions disabled** | Mesh-wide actions (e.g., global config push) are disabled by default to prevent accidental widespread impact. |

### Workarounds for Control Plane Limitations

- **guarded_auto**: Ensure actuators are present and healthy before enabling `guarded_auto`; use `advisory` mode for nodes without actuators.
- **Source suppression**: Implement suppression at the broker or firewall level; use MEL logs to drive external enforcement.
- **Routing changes**: Verify routing changes manually through node CLI or management interface; treat MEL recommendations as investigation prompts.
- **Mesh-level actions**: Enable mesh-level actions explicitly via feature flag after testing on a subset of nodes.

---

## Performance Considerations

| Limitation | Details | Thresholds |
|------------|---------|------------|
| **SQLite suitability** | SQLite is suitable for edge deployments (single-node, low volume). It is not appropriate for high-throughput central aggregation. | >1000 writes/sec or >10GB data requires external database. |
| **Export record limit** | Export endpoints are limited to the last 250 records per category to prevent memory exhaustion. | Use pagination or time-range filters for larger exports. |
| **History query pagination** | History queries are paginated with configurable limits; unbounded queries are rejected. | Default limit: 1000 records; max limit: 10000 records. |

### Workarounds for Performance Limitations

- **SQLite**: For central aggregation, configure MEL to forward to PostgreSQL/MySQL via the external database connector (if available) or implement a custom export pipeline.
- **Export limits**: Use time-range parameters to retrieve data in chunks; implement external ETL for bulk archival.
- **Pagination**: Use cursor-based pagination in your client; do not rely on large page sizes for real-time queries.

---

## Monitoring Limitations

| Limitation | Details |
|------------|---------|
| **JSON-only metrics endpoint** | The `/metrics` endpoint returns JSON only. Prometheus exposition format is not supported. |
| **Cloud monitoring requires external work** | Integration with cloud monitoring services (Datadog, CloudWatch, etc.) is not built-in. |
| **No built-in alerting** | MEL does not include an alerting system; detection relies on external polling of health/metrics endpoints. |

### Workarounds for Monitoring Limitations

- **JSON metrics**: Use a Prometheus JSON exporter (e.g., `prometheus-json-exporter`) to bridge MEL metrics to Prometheus.
- **Cloud monitoring**: Deploy a sidecar or external agent to scrape MEL endpoints and forward to your cloud provider.
- **Alerting**: Implement polling via your existing monitoring stack (Prometheus Alertmanager, Nagios, custom scripts) against MEL health endpoints.

---

## Privacy Limitations

| Limitation | Details |
|------------|---------|
| **storage.encryption_required validation only** | The `storage.encryption_required` setting validates environment variables only. It does not enable or verify at-rest database encryption. |
| **Position redaction in exports only** | Position redaction applies to export operations; raw positions may still exist in the database. |
| **Trust list filtering not implemented** | Filtering based on trust lists (allowlist/blocklist) is not implemented for telemetry ingest. |

### Workarounds for Privacy Limitations

- **Encryption validation**: Enable full-disk encryption or database-level encryption at the infrastructure layer (e.g., LUKS, SQLite SQLCipher).
- **Position redaction**: Implement database-level purging or masking if position retention is a concern; treat exports as the only privacy-safe output.
- **Trust list filtering**: Implement filtering at the MQTT broker (ACLs) or via a pre-processing proxy before telemetry reaches MEL.

---

## Future Roadmap

The following limitations are candidates for future releases:

| Limitation | Planned Status | ETA |
|------------|---------------|-----|
| HTTP transport ingest | Under consideration | TBD |
| Prometheus metrics format | Planned | Next minor release |
| At-rest encryption | Under investigation | TBD |
| Trust list filtering | Backlog | TBD |
| Mesh-level actions default enable | Not planned | N/A |
| BLE transport support | Not planned | N/A |

For updates on roadmap items, see [docs/roadmap/](./roadmap/).

---

## Reporting New Issues

If you encounter behavior not documented here that appears to be a bug rather than a design limitation:

1. Search existing issues in the repository
2. Document reproduction steps with hardware/firmware versions
3. Include logs and configuration (sanitized)
4. File an issue in the appropriate tracker

Do not file defects for limitations explicitly documented in this file.
