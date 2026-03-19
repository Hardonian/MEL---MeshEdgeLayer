# Claims vs reality

This table exists to keep MEL's public language aligned with the current codebase.

| Claim area | Status | Current truth | Action in docs |
| --- | --- | --- | --- |
| "MEL supports Meshtastic" | Partial | MEL supports specific ingest paths only: serial direct-node, TCP direct-node, and MQTT. | Narrowed to transport-specific language. |
| "Direct-node support" | True with caveats | Serial and TCP direct-node ingest are implemented. Send/control paths are not. This repo pass verified code/tests and offline doctor behavior, not live hardware. | Kept, but limited to ingest and bounded verification language. |
| "MQTT support" | True with caveats | MQTT subscribe/ingest works. Publish/control do not. | Kept, with ingest-only caveat. |
| "BLE support" | False as a current feature | BLE is explicitly unsupported. | Downgraded to unsupported/planned. |
| "HTTP transport support" | False as a current feature | No live HTTP ingest path exists. | Downgraded to unsupported. |
| "Metrics endpoint" | False as a current feature | Config placeholders exist, but no listener starts. | Documented as reserved/no-op. |
| "Encrypted storage" | Misleading if implied | `storage.encryption_required` validates env presence only; MEL does not encrypt SQLite at rest. | Documented as validation-only. |
| "Web UI feature flag" | Previously misleading | `features.web_ui` now controls whether the HTML route is registered. | Repaired in code and docs. |
| "Restore" | Partial | Restore validation exists only as `--dry-run`. | Documented as dry-run only. |
