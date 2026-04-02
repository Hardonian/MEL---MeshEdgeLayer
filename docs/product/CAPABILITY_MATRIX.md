# Capability Matrix

Last reviewed: 2026-04-02

| Domain | Current state | Notes |
|---|---|---|
| Serial ingest | Available | Evidence-backed via persisted packets and status surfaces. |
| TCP ingest | Available | Requires Meshtastic framing at endpoint. |
| MQTT ingest | Available | Disconnects/partial ingest must remain explicit. |
| BLE ingest | Unsupported | No runtime ingest implementation. |
| HTTP ingest | Unsupported | No runtime ingest implementation. |
| Incident queue/workbench | Available | Operator workflow and triage surfaces present. |
| Control action governance | Available | Submission/approval/dispatch/execution/audit semantics present. |
| Guarded auto mode | Available with bounds | Executes only when policy + safety checks + actuator path align. |
| Advisory recommendations | Available | Non-canonical; bounded by evidence sufficiency. |
| Proofpack/export surfaces | Available | Exportable evidence and support bundle workflows present. |
| Backup create | Available | Backup bundle creation supported. |
| Backup restore | Validation-only | `--dry-run` required in current posture. |
| Local inference assist | Optional | Assistive only; canonical truth remains deterministic runtime evidence. |

## Canonical caveats

- “Configured” is not equal to “live.”
- “Recommendation” is not equal to “execution.”
- “Imported/offline” is not equal to “current runtime truth.”
