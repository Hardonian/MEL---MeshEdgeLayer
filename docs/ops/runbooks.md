# MEL Operational Runbooks

This document provides step-by-step guidance for common operational scenarios in MEL deployments.

---

## Runbook 1: Transport Down / Reconnect Churn

### Symptom
Transport remains stuck in `attempting`, `configured_offline`, or `error` state. Logs show repeated connection attempts with intermittent successes followed by immediate disconnects.

### Impact
- Data ingestion halted for affected transport
- Potential data loss if buffer overflows
- Cascading alerts if multiple transports affected

### Diagnostic Steps

```bash
# Quick health check
mel doctor

# Inspect specific transport
mel inspect transport <transport-name>

# Follow transport logs
mel logs tail --transport <transport-name> --follow
```

Check these specific fields:
- `consecutive_timeouts`: Count of sequential timeout failures
- `retry_status`: Current retry mechanism state
- `last_error`: Most recent error message

### Expected Output (Healthy)
```json
{
  "name": "edge-node-01",
  "state": "ingesting",
  "consecutive_timeouts": 0,
  "last_connected": "2026-03-20T14:32:00Z",
  "retry_status": "idle"
}
```

### Resolution Steps

1. **Verify endpoint accessibility**
   ```bash
   curl -I <transport-endpoint>
   telnet <host> <port>
   ```

2. **Check credentials validity**
   ```bash
   mel inspect transport <name> --show-credentials-hashes
   # Compare with expected credential hashes
   ```

3. **Adjust timeout settings** (if network is latent)
   ```bash
   mel transport update <name> --connect-timeout 30s --read-timeout 60s
   ```

4. **Force reconnection**
   ```bash
   mel transport disconnect <name>
   mel transport connect <name>
   ```

### Prevention

- Set `reconnect_seconds` to appropriate value for network conditions (default: 5s, consider 30s+ for flaky networks)
- Configure `max_consecutive_timeouts` based on tolerance for temporary failures
- Use exponential backoff in transport configuration
- Monitor `consecutive_timeouts` metric in alerting

---

## Runbook 2: MQTT Subscribe Issues

### Symptom
MQTT transport shows `connected` state but no messages are being received. Broker shows active connection but no subscription activity.

### Impact
- Silent data loss - transport appears healthy but ingests nothing
- Delayed detection of mesh events
- Potential stale data in downstream consumers

### Diagnostic Steps

```bash
# Check detailed transport status
mel inspect transport <mqtt-transport> --verbose

# Verify subscription state
mel mqtt status <transport-name>

# Check broker-side subscription status
# (use broker's admin tools, e.g., mosquitto_sub for testing)
mosquitto_sub -h <broker> -t 'msh/#' -v -d
```

Check these configuration values:
- `topic_filter`: Should match `msh/#` or specific `msh/...` pattern
- `clean_session`: False for persistent subscriptions
- `qos`: 1 recommended for reliability

### Expected Output (Healthy)
```json
{
  "transport": "mqtt-mesh",
  "connected": true,
  "subscriptions": [
    {"topic": "msh/+/telemetry", "qos": 1, "active": true}
  ],
  "messages_received_1m": 145,
  "last_message_at": "2026-03-20T14:35:00Z"
}
```

### Resolution Steps

1. **Verify topic filter matches expected messages**
   ```bash
   # Test subscription with exact topic
   mosquitto_sub -h <broker> -t 'msh/2/json/#' -v
   
   # Check if messages are on different topic
   mosquitto_sub -h <broker> -t '#' -v | head -20
   ```

2. **Review broker permissions**
   ```bash
   # Check ACL if broker uses access control
   # Verify transport credentials have subscribe permission for topic
   ```

3. **Adjust clean_session setting**
   ```bash
   # If missing messages during reconnects, use persistent session
   mel mqtt update <name> --clean-session=false --client-id mel-fixed-id
   ```

4. **Verify QoS compatibility**
   ```bash
   # Upgrade to QoS 1 for at-least-once delivery
   mel mqtt update <name> --qos 1
   ```

### Prevention

- Always use `msh/` prefix in topic configurations for mesh traffic
- Set QoS to 1 for production workloads
- Use persistent sessions (`clean_session=false`) with fixed client IDs
- Include topic validation in transport provisioning

---

## Runbook 3: Dead Letter Growth

### Symptom
`dead_letters` table row count increasing rapidly. API shows growing dead letter queue.

### Impact
- Storage consumption increase
- Potential valid messages lost to dead letter
- Indication of systemic processing issues

### Diagnostic Steps

```bash
# Check dead letter statistics
mel api GET /api/v1/dead-letters --summary

# Review recent dead letters with reasons
mel api GET '/api/v1/dead-letters?limit=20&include_payload=true'

# Check counts by reason
mel db query "SELECT reason, COUNT(*) FROM dead_letters WHERE created_at > datetime('now', '-1 hour') GROUP BY reason"
```

Review the `reason` field for patterns:
- `protobuf_decode_error`: Malformed protobuf payload
- `unsupported_payload_type`: Unknown message type
- `database_constraint_violation`: Schema/constraint issues
- `transport_timeout`: Downstream processing timeout

### Expected Output (Healthy)
- Dead letter growth < 10/hour for active mesh
- Most common reason: transient `transport_timeout` (retried successfully)
- No `protobuf_decode_error` entries

### Resolution Steps

1. **Analyze payload format issues**
   ```bash
   # Extract sample dead letters
   mel db query "SELECT payload_hex FROM dead_letters WHERE reason='protobuf_decode_error' LIMIT 5"
   
   # Verify against expected schema
   protoc --decode_raw < decoded_payload.bin
   ```

2. **Check for incompatible clients**
   ```bash
   # Correlate dead letters with source transports
   mel db query "SELECT source_transport, COUNT(*) FROM dead_letters WHERE created_at > datetime('now', '-1 hour') GROUP BY source_transport"
   ```

3. **Adjust retention settings**
   ```bash
   # Reduce retention if growth is expected/acceptable
   mel config set dead_letter_retention_days 3
   
   # Run cleanup
   mel db cleanup dead-letters --older-than 72h
   ```

4. **Review transport health**
   ```bash
   mel doctor --check-transports
   ```

### Prevention

- Enforce protobuf schema validation at edge
- Version protobuf definitions carefully
- Monitor dead_letters table size with alerts
- Set appropriate retention based on storage budget

---

## Runbook 4: Anomaly Spikes

### Symptom
`/api/v1/transports/anomalies` showing elevated counts. Alerting on anomaly threshold exceeded.

### Impact
- May indicate network degradation
- Possible precursor to transport failures
- Increased operator cognitive load

### Diagnostic Steps

```bash
# Get current anomaly summary
mel api GET /api/v1/transports/anomalies

# View anomaly history
mel api GET '/api/v1/transports/anomalies?start=-1h&granularity=5m'

# Correlate with transport events
mel logs --filter 'level=warn' --since 1h
```

Check these patterns:
- Sudden spike vs gradual increase
- Correlation with specific transports or global
- Timing relative to deployments or network changes

### Expected Output (Healthy)
- Anomaly count < 5/hour per transport
- Brief spikes resolve within 15 minutes
- No correlation with data loss

### Resolution Steps

1. **Identify affected transports**
   ```bash
   mel api GET '/api/v1/transports/anomalies?group_by=transport'
   ```

2. **Review transport settings**
   ```bash
   # Check if thresholds are appropriate
   mel inspect transport <name> --show-anomaly-config
   ```

3. **Check for network issues**
   ```bash
   # Ping test to transport endpoints
   # Check network metrics if available
   ```

4. **Adjust anomaly thresholds** (if false positives)
   ```bash
   mel transport update <name> --anomaly-threshold-latency 500ms
   ```

### Prevention

- Set transport-specific thresholds based on baseline behavior
- Use anomaly patterns to predict failures
- Correlate anomalies with deployment events

---

## Runbook 5: Alert Storms

### Symptom
Multiple active alerts firing simultaneously on same or different transports. High alert volume in notification channels.

### Impact
- Alert fatigue and desensitization
- Delayed response to genuine issues
- Resource consumption from alert processing

### Diagnostic Steps

```bash
# List active alerts
mel alerts list --active

# Check for correlations
mel alerts correlations --since 1h

# Review alert episode IDs
mel api GET /api/v1/alerts/episodes
```

Check these patterns:
- Common episode ID across alerts
- Same underlying cause (network partition, etc.)
- Cascading failure pattern

### Expected Output (Healthy)
- Single root cause alert with related alerts grouped
- Episode IDs linking related alerts
- No more than 2-3 concurrent unrelated alerts

### Resolution Steps

1. **Identify root cause**
   ```bash
   mel alerts root-cause --episode <episode-id>
   ```

2. **Apply alert backoff**
   ```bash
   # Suppress related alerts temporarily
   mel alerts suppress --episode <episode-id> --duration 30m
   ```

3. **Consider transport restart**
   ```bash
   # If specific transport is flapping
   mel transport restart <name>
   ```

4. **Review control plane actions**
   ```bash
   # Check what control plane suggested vs executed
   mel control history --episode <episode-id>
   ```

### Prevention

- Configure alert grouping by episode
- Set alert rate limits per transport
- Use graduated severity (warning before critical)
- Implement alert correlation rules

---

## Runbook 6: Denied Control Actions

### Symptom
Control actions consistently denied in `guarded_auto` mode. Expected automated responses not executing.

### Impact
- Delayed incident response
- Manual intervention required
- Potential escalation of issues

### Diagnostic Steps

```bash
# Review control action history
mel control history --denied --since 1h

# Check specific denial reasons
mel api GET '/api/v1/control/history?filter=denied&include_reason=true'

# Review policy configuration
mel config show control_policy
```

Check denial reasons:
- `cooldown`: Action within cooldown period from previous action
- `budget`: Daily/weekly action budget exhausted
- `low_confidence`: Confidence score below threshold
- `missing_actuator`: Required actuator not available
- `policy_excluded`: Action type excluded by policy

### Expected Output (Healthy)
- < 10% denial rate for valid actions
- Denials primarily for `cooldown` (legitimate rate limiting)
- No `missing_actuator` denials

### Resolution Steps

1. **Review policy settings**
   ```bash
   mel config set control_policy.cooldown_seconds 30
   mel config set control_policy.min_confidence 0.7
   ```

2. **Check action budgets**
   ```bash
   mel control budget status
   mel control budget reset --transport <name>
   ```

3. **Verify actuator availability**
   ```bash
   mel actuator list --transport <name>
   ```

4. **Adjust policy for specific actions**
   ```bash
   mel policy allow --action restart_transport --mode guarded_auto
   ```

### Prevention

- Set realistic cooldown periods (balance safety vs responsiveness)
- Monitor budget consumption trends
- Ensure actuators are provisioned with transports
- Regular policy review and tuning

---

## Runbook 7: Operator Override Behavior

### Symptom
Control actions not executing despite healthy state. Automated decisions appear suppressed.

### Impact
- System not self-healing as designed
- Reliance on manual intervention
- Potential policy confusion

### Diagnostic Steps

```bash
# Check transport override flags
mel inspect transport <name> --show-control-config

# Review active overrides
mel api GET /api/v1/control/overrides

# Check suppress_auto_actions flag
mel config get suppress_auto_actions
```

Check these flags:
- `manual_only`: Only manual actions allowed
- `suppress_auto_actions`: Global suppression
- `override_episode`: Active episode override

### Expected Output (Healthy)
```json
{
  "transport": "node-01",
  "control_mode": "guarded_auto",
  "manual_only": false,
  "suppress_auto_actions": false,
  "active_overrides": []
}
```

### Resolution Steps

1. **Review override flags**
   ```bash
   mel inspect transport <name>
   # Check for manual_only: true
   ```

2. **Remove transport-specific override**
   ```bash
   mel transport update <name> --manual-only=false
   ```

3. **Check global suppression**
   ```bash
   mel config set suppress_auto_actions false
   ```

4. **Document intentional overrides**
   ```bash
   mel override annotate <transport> --reason "Manual maintenance window"
   ```

### Prevention

- Document all manual-only configurations
- Set expiration on temporary overrides
- Audit override usage regularly
- Clear overrides after maintenance windows

---

## Runbook 8: Restart Recovery State

### Symptom
After MEL restart, transports take extended time to reach `ingesting` state. Transports stuck in `historical_only`.

### Impact
- Delayed data ingestion post-restart
- Potential data gaps
- Extended recovery time

### Diagnostic Steps

```bash
# Check current transport states
mel transport list --format table

# Check historical state duration
mel api GET '/api/v1/transports?state=historical_only'

# Review runtime evidence
mel logs --component runtime --since startup
```

Check the state transition:
- `historical_only`: Reviewing retained data, not yet connecting
- `attempting`: Actively trying to connect
- `ingesting`: Fully operational

### Expected Output (Healthy)
```
Transport      State           Since
---------      -----           -----
node-01        ingesting       30s ago
node-02        ingesting       45s ago
mqtt-main      ingesting       15s ago
```

State transition timeline:
1. `historical_only` (0-10s): Processing retained messages
2. `attempting` (10-30s): Connecting to transport
3. `ingesting` (30s+): Normal operation

### Resolution Steps

1. **Verify retained data is present**
   ```bash
   mel db query "SELECT COUNT(*) FROM retained_messages WHERE transport='node-01'"
   ```

2. **Allow time for reconnect**
   ```bash
   # Wait up to 60s for normal recovery
   # Historical_only is expected initially
   ```

3. **Force transition if stuck**
   ```bash
   mel transport force-state <name> --state attempting
   ```

4. **Check for initialization errors**
   ```bash
   mel logs --level error --since startup
   ```

### Prevention

- Ensure clean shutdown with proper state persistence
- Size retained message buffer appropriately
- Monitor recovery time as SLO metric
- Avoid frequent restarts during high-traffic periods

---

## Runbook 9: Retained Active Actions

### Symptom
Actions showing as `active` long after expected completion. Control action list shows stale entries.

### Impact
- Incorrect view of system state
- Potential blocking of subsequent actions
- Resource leaks in action tracking

### Diagnostic Steps

```bash
# List active actions
mel control list --state active

# Check action lifecycle state
mel api GET '/api/v1/control/actions?state=active&include_timestamps=true'

# Review closure handling
mel logs --component control --filter 'closure'
```

Check these fields:
- `lifecycle_state`: active, completing, closed
- `closure_state`: pending, sent, acknowledged, timeout
- `created_at`: When action was initiated
- `expected_duration`: Configured timeout

### Expected Output (Healthy)
- No active actions older than 5 minutes
- All completed actions show `lifecycle_state: closed`
- Closure states properly tracked

### Resolution Steps

1. **Verify action completion**
   ```bash
   # Check if action actually succeeded
   mel control show <action-id>
   ```

2. **Check for missing closure states**
   ```bash
   # If closure acknowledgment missing
   mel control force-close <action-id> --reason "closure_timeout"
   ```

3. **Review closure handling**
   ```bash
   # Check closure channel health
   mel inspect transport <name> --show-closure-config
   ```

4. **Clean up stale actions**
   ```bash
   mel control cleanup --stale --dry-run
   mel control cleanup --stale
   ```

### Prevention

- Set appropriate action timeouts
- Ensure closure acknowledgment paths are reliable
- Monitor active action age
- Implement closure heartbeat

---

## Runbook 10: DB Growth / Retention

### Symptom
Database file growing beyond expected size. Storage alerts firing.

### Impact
- Disk space exhaustion
- Degraded query performance
- Potential write failures

### Diagnostic Steps

```bash
# Check table sizes
mel db stats --table-sizes

# Review retention settings
mel config show retention

# Check cleanup job status
mel db cleanup status
```

Key tables to monitor:
- `messages`: Primary message data
- `dead_letters`: Failed message storage
- `control_actions`: Action history
- `anomalies`: Anomaly records
- `logs`: Application logs

### Expected Output (Healthy)
```
Table            Size     Rows      Retention
-----            ----     ----      ---------
messages         1.2GB    5.2M      7 days
dead_letters     50MB     10K       3 days
control_actions  200MB    500K      30 days
logs             500MB    2.1M      7 days
```

### Resolution Steps

1. **Run vacuum to reclaim space**
   ```bash
   mel db vacuum
   # Note: This may lock database temporarily
   ```

2. **Adjust retention settings**
   ```bash
   mel config set retention.messages_days 5
   mel config set retention.dead_letters_days 2
   mel config set retention.logs_days 3
   ```

3. **Run manual cleanup**
   ```bash
   mel db cleanup --all
   ```

4. **Verify cleanup is running**
   ```bash
   mel db cleanup status
   # Should show last_run within last hour
   ```

5. **Check for runaway table**
   ```bash
   mel db query "SELECT name FROM sqlite_master WHERE type='table'" | xargs -I {} mel db stats --table {}
   ```

### Prevention

- Set retention based on business requirements and storage budget
- Monitor table growth rates
- Schedule regular vacuum during low-traffic periods
- Alert on unexpected growth patterns

---

## Runbook 11: History Lookup / Closure Interpretation

### Symptom
Difficulty understanding control action outcomes from history. Unclear whether actions succeeded or failed.

### Impact
- Difficult incident analysis
- Unclear operational patterns
- Audit/compliance challenges

### Diagnostic Steps

```bash
# Get detailed control history
mel control history --detailed --limit 50

# Review specific action
mel control show <action-id> --include-decisions

# Query with closure information
mel api GET '/api/v1/control/history?include_closure=true'
```

Understand these states:

| State | Meaning |
|-------|---------|
| `lifecycle_state` | Where action is in its lifecycle |
| `closure_state` | Status of acknowledgment/closing |
| `result` | Final outcome (success/failure) |

### State Combinations

| lifecycle_state | closure_state | result | Interpretation |
|-----------------|---------------|--------|----------------|
| closed | acknowledged | success | Action completed successfully |
| closed | timeout | failure | Action ran but closure never received |
| completing | pending | null | Action executing, awaiting closure |
| active | null | null | Action dispatched, no response yet |

### Expected Output (Healthy)
```json
{
  "action_id": "act-123",
  "lifecycle_state": "closed",
  "closure_state": "acknowledged",
  "result": "success",
  "decisions": [
    {"at": "2026-03-20T10:00:00Z", "decision": "approved", "confidence": 0.95}
  ],
  "outcome": {
    "transport_restarted": true,
    "downtime_seconds": 2
  }
}
```

### Resolution Steps

1. **Correlate decisions with outcomes**
   ```bash
   mel control analyze --action <action-id>
   ```

2. **Review decision confidence**
   ```bash
   mel control history --filter 'confidence<0.8' --include-outcomes
   ```

3. **Export for offline analysis**
   ```bash
   mel control history --format csv --since 7d > control_history.csv
   ```

4. **Set up closure tracking**
   ```bash
   mel config set control.require_closure_ack true
   ```

### Prevention

- Document state machine for operators
- Include closure timeout in action configuration
- Review low-confidence decisions regularly
- Correlate decisions with outcomes for policy tuning

---

## Quick Reference Commands

| Task | Command |
|------|---------|
| Health check | `mel doctor` |
| Transport status | `mel transport list` |
| Recent logs | `mel logs tail -n 100` |
| Active alerts | `mel alerts list --active` |
| Control history | `mel control history --since 1h` |
| DB stats | `mel db stats` |
| API check | `mel api GET /api/v1/health` |

---

## Escalation Path

1. **Level 1**: Follow runbook, use `mel doctor` for automated diagnosis
2. **Level 2**: Review logs, correlate events, check metrics
3. **Level 3**: Engage engineering - provide incident timeline and diagnostic output
4. **Emergency**: Use `mel emergency --suspend-control` to pause automated actions if needed
