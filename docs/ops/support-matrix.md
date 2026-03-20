# Support Matrix

## Transport Support Matrix

| Transport | Status | Hardware Verified | Doctor Coverage | Notes |
|-----------|--------|-------------------|-----------------|-------|
| Serial | Supported | Partial* | Yes | Requires stty, device access |
| TCP | Supported | Partial* | Yes | Raw Meshtastic framing only |
| MQTT | Supported | Yes | Partial | Subscribe only, no publish |
| serialtcp | Partial | No | Yes | TCP alias, not primary path |
| BLE | Unsupported | N/A | N/A | Explicitly not supported |
| HTTP | Unsupported | N/A | N/A | No ingest path exists |

*Hardware verified = exercised in repo environment

## Platform Support

| Platform | Status | Installation | Service | Notes |
|----------|--------|--------------|---------|-------|
| Linux amd64 | Supported | install-linux.sh | systemd | Primary target |
| Linux arm64 | Supported | Manual | systemd | Pi, embedded |
| Raspberry Pi | Supported | install-pi.md | systemd | Serial direct-node |
| Termux | Supported | install-termux.md | Manual | Android/development |

## Configuration Support

| Feature | Status | Caveats |
|---------|--------|---------|
| Single transport | Fully supported | Recommended |
| Hybrid MQTT+Direct | Supported | Dedupe requires verification |
| Multi-direct | Not recommended | Ownership contention |
| Auth enabled | Supported | Required for remote bind |
| Web UI | Supported | Can disable with features.web_ui=false |
| Metrics endpoint | Supported | JSON only, served on main API |

## Control Plane Support

| Mode | Status | Use Case |
|------|--------|----------|
| disabled | Supported | No automation |
| advisory | Supported (default) | Observe suggestions |
| guarded_auto | Supported | Limited automation |

## Action Reality

| Action | Executable | Advisory Only | Notes |
|--------|------------|---------------|-------|
| restart_transport | Yes | No | Bounded reconnect |
| resubscribe_transport | Yes | No | MQTT only |
| backoff_increase | Yes | No | Local transport |
| backoff_reset | Yes | No | Local transport |
| trigger_health_recheck | Yes | No | Async check |
| temporarily_deprioritize | No | Yes | No routing selector |
| temporarily_suppress_noisy_source | No | Yes | No suppression actuator |
| clear_suppression | No | Yes | No suppression actuator |

## Data Retention Support

- **SQLite**: Fully supported
- **Automatic retention**: Yes
- **Manual vacuum**: Yes (`mel db vacuum`)
- **Backup**: Yes (`mel backup create`)
- **Restore**: Validation only (`--dry-run`)

## Explicitly Not Supported

- BLE transport
- HTTP transport ingest
- Packet transmission/publishing
- Radio administration
- Encrypted SQLite at rest
- Prometheus exposition format (JSON only)
