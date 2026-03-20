# MEL Operator Glossary

## Transport States

| State | Description |
|-------|-------------|
| **disabled** | Transport is configured but not enabled for use. No connection attempts will be made. |
| **configured_not_attempted** | Transport is enabled but a connection has not yet been attempted. |
| **attempting** | Currently in the process of trying to establish a connection. |
| **configured_offline** | Connection attempt failed after exhausting retry attempts. The transport is temporarily unavailable. See [episode ID](#health-terms) for cluster context. |
| **connected_no_ingest** | Transport is connected to the data source but no packets have been stored yet. Awaiting first message. |
| **ingesting** | Active packet ingest with [SQLite writes](#data-terms). This is the normal operating state for live data. |
| **historical_only** | Has prior [persisted](#data-terms) data but no live connection. Useful for [replay](#operational-terms) operations. |
| **error** | Error state with details available in `last_error` field. Check logs and [health score](#health-terms) for diagnosis. |

## Control Plane Terms

| Term | Definition |
|------|------------|
| **advisory mode** | Control plane evaluates actions but does not execute them. Recommended for testing new rules before enabling [guarded_auto](#control-plane-terms). |
| **guarded_auto mode** | Control plane evaluates and executes safe actions automatically. Requires validated [action reality](#control-plane-terms). |
| **action reality** | Whether an action has a working actuator. An action with no reality is advisory-only regardless of mode. |
| **blast radius** | Scope of impact for a control action. Determines how many nodes or regions could be affected. |
| **cooldown** | Waiting period enforced between actions on the same target to prevent thrashing. |
| **confidence score** | Reliability metric (0-1) for action decisions. Higher scores indicate more reliable predictions. |

## Data Terms

| Term | Definition |
|------|------------|
| **dead letter** | A message that failed MEL-side processing and could not be stored or routed. Usually indicates a schema mismatch or corruption. |
| **observation drop** | A message received from the transport but intentionally not stored. Differs from [dead letter](#data-terms) in that this is a filtering decision. |
| **rx_time** | Timestamp extracted from the mesh packet itself, representing when the packet was originally transmitted. Not to be confused with ingest time. |
| **persisted** | Data that has been stored in SQLite and is available for [replay](#operational-terms) and offline analysis. |
| **runtime** | Data currently active in memory. May not yet be [persisted](#data-terms). |

## Health Terms

| Term | Definition |
|------|------------|
| **health score** | 0-100 score reflecting overall transport health. Scores below 50 may trigger [alerts](#health-terms). Used to prioritize which transports need attention. |
| **episode ID** | Correlation identifier assigned to a cluster of related failures. All events in a [failure cluster](#health-terms) share the same episode ID. |
| **failure cluster** | A group of related transport failures occurring in temporal proximity, grouped under a single [episode ID](#health-terms). |
| **anomaly** | An unusual pattern in transport behavior detected by the health monitoring system. May or may not require intervention. |
| **alert** | An active condition requiring operator attention. Alerts have severity levels and may correlate with [anomalies](#health-terms) or state transitions. |

## Message Types

| Type | Description |
|------|-------------|
| **text** | Human-readable text messages, often from mesh node users. |
| **position** | Location data including GPS coordinates, altitude, and precision. |
| **node_info** | Node metadata including hardware info, firmware version, and capabilities. |
| **telemetry** | Device metrics such as battery level, air utilization, and channel utilization. |
| **unknown** | Unrecognized payload type that could not be decoded into a known schema. |

## Configuration Terms

| Term | Definition |
|------|------------|
| **direct-node** | A transport using direct serial or TCP connection to a radio device. |
| **MQTT ingest** | A transport that subscribes to MQTT broker topics for message ingestion. |
| **hybrid** | A deployment with multiple transport types configured simultaneously (e.g., [direct-node](#configuration-terms) + [MQTT ingest](#configuration-terms)). |
| **redaction** | Privacy-driven data masking applied to sensitive fields before [persisted](#data-terms) storage. |

## Operational Terms

| Term | Definition |
|------|------------|
| **doctor** | Offline diagnostic command for analyzing transport state, [health scores](#health-terms), and connection history. |
| **panel** | Compact operator status view showing key metrics across all transports at a glance. |
| **replay** | Query and retrieve [persisted](#data-terms) messages from historical storage for analysis or debugging. |
| **retention** | Data lifecycle management policies determining how long [persisted](#data-terms) data is kept before deletion. |
