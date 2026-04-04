# MEL Troubleshooting Guide

This guide helps operators diagnose and resolve common issues with MEL deployments. For detailed step-by-step procedures, refer to the [Operational Runbooks](../runbooks/README.md).

---

## Before You Begin

### Check MEL Version

Always start by verifying your MEL version:

```bash
mel version
```

Include this version when reporting issues or seeking support.

### Collect Diagnostics

Run the diagnostic collection before troubleshooting:

```bash
# Quick health check
mel doctor --config /etc/mel/config.json

# Collect comprehensive diagnostics
mel doctor --config /etc/mel/config.json > diagnostics-$(date +%Y%m%d).json

# See the full diagnostics collection guide
# ./diagnostics.md
```

For automated collection, use the script in [Diagnostics Collection](./diagnostics.md).

### How to Use This Guide

1. **Identify your symptom** - Find the section matching your issue
2. **Run diagnostics** - Use `mel doctor` to validate the environment
3. **Follow resolution steps** - Work through solutions in order
4. **Reference runbooks** - For complex issues, see detailed procedures in [Runbooks](../runbooks/README.md)
5. **Escalate if needed** - See [Getting Help](#getting-help) section

---

## Local Verification & Toolchain Issues

### Frontend Make Targets Fail with Node Runtime Contract Errors

**Symptom:** `make frontend-test`, `make frontend-typecheck`, or `make frontend-install` fails before running npm commands.

**Typical error:**
```text
[runtime-contract] Node 24.x required for make frontend-*.
```

**Why this happens:** Direct frontend make entrypoints are deterministic and require your current shell to already be on Node `24.x`.

**Resolution:**

```bash
# From repo root (preferred):
. ./scripts/dev-env.sh

# Confirm runtime:
node -v
# Expected: v24.x

# Retry command:
make frontend-test
```

If `dev-env.sh` cannot activate Node (for example, nvm is missing), install Node 24.x and re-run the command in a fresh shell.

---

## Configuration Issues

### Config File Permission Errors

**Symptom:** MEL refuses to start or `mel doctor` reports permission findings.

**Error messages:**
```
config file permissions too open
Operator config files must be chmod 600 before MEL will trust them
```

**Resolution:**

```bash
# Fix permissions (must be 0600 for production)
chmod 600 /etc/mel/config.json

# Verify
ls -la /etc/mel/config.json
# Should show: -rw-------
```

**Note:** MEL enforces `0600` permissions in production. The config contains sensitive credentials and must not be world-readable.

### Config Validation Failures

**Symptom:** Config syntax errors or missing required fields.

**Resolution:**

```bash
# Validate config without starting service
mel config validate --config /etc/mel/config.json

# Common issues and fixes:
# - Missing data_dir: mkdir -p /var/lib/mel/data
# - Invalid JSON: Use jq to validate: jq . /etc/mel/config.json
# - Missing bind address: Add "bind": {"api": "127.0.0.1:8080"} to config
```

### Invalid Transport Types

**Symptom:** Transport fails to initialize with unsupported type errors.

**Supported types:** `serial`, `tcp`, `mqtt`

**Unsupported types:** `ble`, `http` (explicitly not supported)

**Resolution:**

```bash
# Check your config for unsupported types
jq '.transports | keys[]' /etc/mel/config.json

# Verify transport type is supported
# Serial: "type": "serial", "device": "/dev/ttyUSB0"
# TCP: "type": "tcp", "endpoint": "192.168.1.100:4403"
# MQTT: "type": "mqtt", "broker": "mqtt.example.com:1883"
```

### Missing Required Fields

**Symptom:** Transport configuration incomplete.

**Required fields by type:**

| Type | Required Fields |
|------|-----------------|
| serial | `device`, `baud` |
| tcp | `endpoint` |
| mqtt | `broker`, `topic_filter` |

**Resolution:**

```bash
# Check transport configuration
mel inspect transport <transport-name> --config /etc/mel/config.json

# Example valid configurations:
# Serial: {"type": "serial", "device": "/dev/ttyUSB0", "baud": 115200}
# TCP: {"type": "tcp", "endpoint": "192.168.1.100:4403"}
# MQTT: {"type": "mqtt", "broker": "mqtt.example.com:1883", "topic_filter": "msh/#"}
```

---

## Startup Issues

### MEL Won't Start

**Symptom:** `mel serve` exits immediately or hangs.

**Diagnostic steps:**

```bash
# 1. Run pre-flight check
mel doctor --config /etc/mel/config.json

# 2. Check config validation
mel config validate --config /etc/mel/config.json

# 3. Try with debug logging
mel serve --config /etc/mel/config.json --debug
```

**Common causes:**

| Cause | Check | Resolution |
|-------|-------|------------|
| Config permissions | `ls -la /etc/mel/config.json` | `chmod 600` |
| Missing data dir | `ls -la /var/lib/mel` | `mkdir -p` |
| Database locked | `lsof /var/lib/mel/mel.db` | Stop other MEL instance |
| Port in use | `lsof -i :8080` | Change bind port |

### Port Already in Use

**Symptom:** `bind: address already in use` error.

**Resolution:**

```bash
# Find process using port
sudo lsof -i :8080

# Or using ss
sudo ss -tlnp | grep 8080

# Options:
# 1. Stop the other process
# 2. Change MEL bind port in config
#    "bind": ":8081"
# 3. Use a different interface
#    "bind": "127.0.0.1:8080"
```

### Database Permission Errors

**Symptom:** `database write failed: permission denied`

**Resolution:**

```bash
# Check database ownership
ls -la /var/lib/mel/mel.db

# Fix ownership (run as root or with sudo)
sudo chown mel:mel /var/lib/mel/mel.db
sudo chmod 600 /var/lib/mel/mel.db

# Verify data directory permissions
ls -la /var/lib/mel/
```

### Schema Migration Failures

**Symptom:** Database schema version mismatch errors.

**Resolution:**

```bash
# Check schema version
sqlite3 /var/lib/mel/mel.db "PRAGMA user_version;"

# Run doctor to check compatibility
mel doctor --config /etc/mel/config.json

# If migration needed, backup first
mel backup create --config /etc/mel/config.json --out /backup/pre-migration.tar.gz

# MEL handles migrations automatically on startup
# If manual intervention needed, see upgrades documentation
```

---

## Transport Issues

For detailed transport troubleshooting, see [Troubleshooting Transports](./troubleshooting-transports.md). For operational runbooks, see [Runbooks](../runbooks/README.md).

### Serial Device Not Found

**Symptom:** `configured_offline` state for serial transport.

**Quick checks:**

```bash
# Verify device exists
ls -la /dev/ttyUSB0

# Check device permissions
ls -la /dev/ttyUSB*

# Add user to dialout group (requires re-login)
sudo usermod -a -G dialout $USER

# Test with screen/minicom
screen /dev/ttyUSB0 115200
```

**See Runbook 1** in [Runbooks](../runbooks/README.md) for detailed reconnection procedures.

### Serial Permission Denied

**Symptom:** Cannot open serial device, permission errors.

**Resolution:**

```bash
# Check device permissions
ls -la /dev/ttyUSB0

# Add user to dialout group
sudo usermod -a -G dialout $USER

# Or use udev rule for persistent permissions
# /etc/udev/rules.d/99-meshtastic.rules:
# SUBSYSTEM=="tty", ATTRS{idVendor}=="xxxx", ATTRS{idProduct}=="xxxx", MODE="0660", GROUP="dialout"
```

### TCP Connection Refused

**Symptom:** TCP transport shows `configured_offline` or connection errors.

**Resolution:**

```bash
# Test connectivity
telnet <host> <port>

# Or using nc
nc -zv <host> <port>

# Verify Meshtastic framing is exposed
curl -v telnet://<host>:<port>

# Check firewall rules
sudo iptables -L | grep <port>
```

**See Runbook 1** in [Runbooks](../runbooks/README.md) for reconnection and timeout tuning.

### MQTT Connection Issues

**Symptom:** MQTT transport won't connect or shows `connected_no_ingest`.

**Quick checks:**

```bash
# Test broker connectivity
mosquitto_sub -h <broker> -t 'msh/#' -v -d

# Check credentials
mel inspect transport <mqtt-transport> --show-credentials-hashes

# Verify topic filter
grep topic_filter /etc/mel/config.json
```

**See Runbook 2** in [Runbooks](../runbooks/README.md) for detailed MQTT subscription troubleshooting.

### Transport Stuck in Error State

**Symptom:** Transport shows `error` state and won't recover.

**Resolution:**

```bash
# Check transport status
mel inspect transport <name> --config /etc/mel/config.json

# Review recent errors
mel logs tail --config /etc/mel/config.json | jq '.[] | select(.category == "transport")'

# Check retry status
mel api GET /api/v1/status | jq '.transports.<name>.retry_status'

# Force reconnection
mel transport disconnect <name>
mel transport connect <name>
```

**See Runbook 1** in [Runbooks](../runbooks/README.md) for transport down/reconnect procedures.

---

## Ingest Issues

### No Messages Being Stored

**Symptom:** Database message count not increasing, `total_messages` stagnant.

**Diagnostic steps:**

```bash
# Check last ingest time
mel status --config /etc/mel/config.json | jq '.last_successful_ingest'

# Check transport states
mel transports list --config /etc/mel/config.json

# Check for errors
mel logs tail --config /etc/mel/config.json | jq '.[] | select(.level == "error")'

# Verify ingest is working
mel replay --config /etc/mel/config.json --limit 5
```

**Common causes:**

1. **No active transports** - Enable at least one transport
2. **All transports offline** - Check network/serial connections
3. **No mesh traffic** - Generate test traffic
4. **Message deduplication** - Messages may be dropped as duplicates

### UI Shows No Nodes

**Symptom:** Web UI displays empty node list.

**Interpretation:** No real packets have been ingested yet. This is expected behavior when no traffic has been received.

**Resolution:**

```bash
# Check if any messages exist
mel nodes --config /etc/mel/config.json

# Generate test traffic on the mesh
# Or wait for natural mesh activity

# Check transport health
mel doctor --config /etc/mel/config.json

# Verify last_successful_ingest timestamp
mel status --config /etc/mel/config.json | jq '.last_successful_ingest'
```

### Messages Not Appearing

**Symptom:** Mesh traffic exists but MEL not storing messages.

**Resolution:**

```bash
# Check dead letters for processing failures
sqlite3 /var/lib/mel/mel.db "SELECT reason, COUNT(*) FROM dead_letters WHERE created_at > datetime('now', '-1 hour') GROUP BY reason;"

# Check transport packet counters
mel inspect transport <name> --config /etc/mel/config.json

# Verify protobuf decoding
mel logs tail --config /etc/mel/config.json | grep -i "decode\|protobuf"
```

**See Runbook 3** in [Runbooks](../runbooks/README.md) for dead letter analysis.

---

## Database Issues

### Database Growing Too Large

**Symptom:** Storage alerts, degraded performance.

**Quick check:**

```bash
# Check database size
mel db stats --config /etc/mel/config.json

# Check table sizes
sqlite3 /var/lib/mel/mel.db "SELECT name FROM sqlite_master WHERE type='table';" | xargs -I {} sh -c 'echo "Table: {}"; sqlite3 /var/lib/mel/mel.db "SELECT COUNT(*) FROM {};"'
```

### Vacuum Recommendations

**When to vacuum:**

- After bulk deletion
- Weekly for high-traffic deployments
- When database file is significantly larger than actual data

**Resolution:**

```bash
# Run vacuum (may lock database temporarily)
mel db vacuum --config /etc/mel/config.json

# Check space reclaimed
ls -lh /var/lib/mel/mel.db
```

**See Runbook 10** in [Runbooks](../runbooks/README.md) for detailed retention and cleanup procedures.

### Disk Space Warnings

**Symptom:** Low disk space, write failures.

**Resolution:**

```bash
# Check disk space
df -h /var/lib/mel

# Check retention settings
mel config show retention

# Reduce retention if needed
mel config set retention.messages_days 5
mel config set retention.dead_letters_days 2
mel config set retention.logs_days 3

# Run cleanup
mel db cleanup --all --config /etc/mel/config.json

# Vacuum to reclaim space
mel db vacuum --config /etc/mel/config.json
```

**Prevention:**

- Set retention based on storage budget
- Monitor table growth rates
- Schedule regular cleanup jobs

---

## Control Plane Issues

### Actions Not Executing

**Symptom:** Expected automated responses not occurring.

**Diagnostic steps:**

```bash
# Check control plane status
mel control status --config /etc/mel/config.json

# Review action history
mel control history --config /etc/mel/config.json

# Check for denied actions
mel control history --denied --since 1h --config /etc/mel/config.json
```

**See Runbook 6** in [Runbooks](../runbooks/README.md) for denied control actions troubleshooting.

### Understanding Denial Reasons

**Common denial reasons:**

| Reason | Meaning | Resolution |
|--------|---------|------------|
| `cooldown` | Action within cooldown period | Wait for cooldown or adjust policy |
| `budget` | Daily/weekly budget exhausted | Reset budget or increase limits |
| `low_confidence` | Confidence below threshold | Review decision criteria |
| `missing_actuator` | Required actuator unavailable | Verify actuator configuration |
| `policy_excluded` | Action excluded by policy | Update policy to allow action |

**Resolution:**

```bash
# Check denial details
mel api GET '/api/v1/control/history?filter=denied&include_reason=true'

# Review policy configuration
mel config show control_policy

# Adjust cooldown or confidence
mel config set control_policy.cooldown_seconds 30
mel config set control_policy.min_confidence 0.7
```

### Emergency Disable Behavior

**To pause all automated actions:**

```bash
# Suspend control plane
mel emergency --suspend-control --config /etc/mel/config.json

# Resume control plane
mel emergency --resume-control --config /etc/mel/config.json
```

**Use cases:**

- Maintenance windows
- Debugging unexpected behavior
- Incident response

---

## Performance Issues

### High CPU Usage

**Symptom:** MEL process consuming excessive CPU.

**Diagnostic steps:**

```bash
# Identify CPU-intensive threads
top -H -p $(pgrep mel)

# Check message ingest rate
mel status --config /etc/mel/config.json | jq '.ingest_rate_per_sec'

# Review logs for tight loops
mel logs tail --config /etc/mel/config.json | grep -i "retry\|reconnect"
```

**Common causes:**

- Transport reconnection churn
- High message volume
- Inefficient queries

**Resolution:**

```bash
# Check for transport flapping
mel api GET /api/v1/transports/anomalies

# Adjust reconnection backoff
mel transport update <name> --reconnect-seconds 30
```

### Slow Queries

**Symptom:** API responses slow, UI loading delays.

**Resolution:**

```bash
# Check database integrity
sqlite3 /var/lib/mel/mel.db "PRAGMA integrity_check;"

# Analyze and optimize
sqlite3 /var/lib/mel/mel.db "ANALYZE;"

# Check for missing indexes
sqlite3 /var/lib/mel/mel.db ".indexes"

# Vacuum if needed
mel db vacuum --config /etc/mel/config.json
```

### Memory Growth

**Symptom:** MEL memory usage increasing over time.

**Diagnostic steps:**

```bash
# Monitor memory usage
ps aux | grep mel

# Check for memory leaks in logs
mel logs tail --config /etc/mel/config.json | grep -i "memory\|gc"

# Review retained message counts
sqlite3 /var/lib/mel/mel.db "SELECT COUNT(*) FROM retained_messages;"
```

**Resolution:**

- Restart MEL service (if memory leak suspected)
- Reduce message retention settings
- Check for unclosed connections

---

## Getting Help

### What Information to Collect

Before filing an issue or escalating:

1. **Run diagnostics:**
   ```bash
   mel doctor --config /etc/mel/config.json > doctor-output.json
   mel status --config /etc/mel/config.json > status.json
   mel panel --format json --config /etc/mel/config.json > panel.json
   ```

2. **Collect recent logs:**
   ```bash
   mel logs tail --config /etc/mel/config.json > logs.json
   ```

3. **Gather version info:**
   ```bash
   mel version
   uname -a
   ```

4. **Document the issue:**
   - What you expected to happen
   - What actually happened
   - Steps to reproduce
   - Recent changes to config/environment

### Where to File Issues

- **GitHub Issues:** For bugs and feature requests
- **Discussions:** For questions and community support
- **Security Issues:** Follow security disclosure policy

### Support Escalation Path

**Level 1 - Self-Service:**
1. Run `mel doctor` for automated diagnosis
2. Review this troubleshooting guide
3. Check [Runbooks](../runbooks/README.md) for detailed procedures

**Level 2 - Community Support:**
1. Search existing GitHub issues
2. Ask in community discussions
3. Include diagnostic output from Level 1

**Level 3 - Engineering Support:**
1. Collect full diagnostic bundle
2. Document incident timeline
3. Provide sanitized config (passwords removed)
4. Include MEL version, OS, and architecture

**Emergency:**
- Use `mel emergency --suspend-control` to pause automated actions
- Follow incident response procedures in [Incident Triage](./incident-triage.md)

---

## Quick Reference

### Essential Commands

| Task | Command |
|------|---------|
| Health check | `mel doctor --config /etc/mel/config.json` |
| System status | `mel status --config /etc/mel/config.json` |
| Transport list | `mel transports list --config /etc/mel/config.json` |
| Recent logs | `mel logs tail --config /etc/mel/config.json` |
| Replay messages | `mel replay --config /etc/mel/config.json --limit 10` |
| Database vacuum | `mel db vacuum --config /etc/mel/config.json` |
| Control history | `mel control history --config /etc/mel/config.json` |

### Transport States

| State | Meaning |
|-------|---------|
| `disabled` | Transport explicitly disabled |
| `configured_not_attempted` | Not yet tried to connect |
| `attempting` | Currently trying to connect |
| `configured_offline` | Cannot reach endpoint/device |
| `connected_no_ingest` | Connected but no packets yet |
| `ingesting` | Fully operational |
| `historical_only` | Has past data, no live connection |
| `error` | Error state, check logs |

### Related Documentation

- [Diagnostics Collection](./diagnostics.md) - Comprehensive diagnostic procedures
- [Troubleshooting Transports](./troubleshooting-transports.md) - Transport-specific issues
- [Operational Runbooks](../runbooks/README.md) - Step-by-step operational procedures
- [Incident Triage](./incident-triage.md) - Structured incident response
- [CLI Reference](./cli-reference.md) - Complete command documentation
- [Configuration](./configuration-reference.md) - Configuration options and security rules
