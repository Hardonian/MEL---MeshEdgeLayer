# MEL Diagnostics Collection Guide

This guide helps operators collect useful diagnostics for troubleshooting and support escalation.

---

## Quick Collection Script

Save this script as `collect-diagnostics.sh` for one-command diagnostics collection:

```bash
#!/bin/bash
# MEL Diagnostics Collection Script
# Usage: ./collect-diagnostics.sh [--config <path>] [--out <dir>]

set -euo pipefail

CONFIG="${CONFIG:-configs/mel.example.json}"
OUTPUT_DIR="${OUTPUT_DIR:-./mel-diagnostics-$(date +%Y%m%d-%H%M%S)}"

while [[ $# -gt 0 ]]; do
  case $1 in
    --config) CONFIG="$2"; shift 2 ;;
    --out) OUTPUT_DIR="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

echo "Collecting MEL diagnostics..."
echo "Config: $CONFIG"
echo "Output: $OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Core diagnostics
echo "[1/9] Running mel doctor..."
mel doctor --config "$CONFIG" > "$OUTPUT_DIR/doctor.json" 2>&1 || true

echo "[2/9] Collecting mel status..."
mel status --config "$CONFIG" > "$OUTPUT_DIR/status.json" 2>&1 || true

echo "[3/9] Collecting mel panel..."
mel panel --config "$CONFIG" > "$OUTPUT_DIR/panel.txt" 2>&1 || true
mel panel --format json --config "$CONFIG" > "$OUTPUT_DIR/panel.json" 2>&1 || true

echo "[4/9] Validating config..."
mel config validate --config "$CONFIG" > "$OUTPUT_DIR/config-validation.json" 2>&1 || true

echo "[5/9] Collecting recent logs..."
mel logs tail --config "$CONFIG" > "$OUTPUT_DIR/logs-recent.json" 2>&1 || true

echo "[6/9] Listing transports..."
mel transports list --config "$CONFIG" > "$OUTPUT_DIR/transports.json" 2>&1 || true

echo "[7/9] Counting messages..."
DB_PATH=$(mel status --config "$CONFIG" 2>/dev/null | jq -r '.database.path' 2>/dev/null || echo "")
if [[ -n "$DB_PATH" && -f "$DB_PATH" ]]; then
  sqlite3 "$DB_PATH" "SELECT COUNT(*) as total_messages FROM messages;" > "$OUTPUT_DIR/message-count.txt" 2>&1 || true
  sqlite3 "$DB_PATH" "SELECT COUNT(*) as dead_letters FROM dead_letters;" > "$OUTPUT_DIR/dead-letter-count.txt" 2>&1 || true
fi

echo "[8/9] Collecting control history..."
mel control history --limit 50 --config "$CONFIG" > "$OUTPUT_DIR/control-history.json" 2>&1 || true

echo "[9/9] Creating support bundle..."
mel export --config "$CONFIG" --out "$OUTPUT_DIR/support-bundle.json" 2>&1 || true

# Create summary
echo "Creating summary..."
cat > "$OUTPUT_DIR/SUMMARY.txt" << EOF
MEL Diagnostics Collection Summary
==================================
Generated: $(date -Iseconds)
Config: $CONFIG

Files Collected:
- doctor.json: Comprehensive system diagnostics
- status.json: Current system status snapshot
- panel.txt/json: Operator-facing status panel
- config-validation.json: Configuration validation results
- logs-recent.json: Recent audit log entries
- transports.json: Transport status and configuration
- message-count.txt: Total message count
- dead-letter-count.txt: Dead letter count
- control-history.json: Control plane action history
- support-bundle.json: Full data export (may be redacted)

Quick Checks:
EOF

# Add quick health indicators
if jq -e '.findings | length == 0' "$OUTPUT_DIR/doctor.json" >/dev/null 2>&1; then
  echo "- Doctor: PASS (no critical findings)" >> "$OUTPUT_DIR/SUMMARY.txt"
else
  echo "- Doctor: WARNING (see doctor.json for findings)" >> "$OUTPUT_DIR/SUMMARY.txt"
fi

echo ""
echo "Diagnostics collection complete: $OUTPUT_DIR"
echo "Review SUMMARY.txt for an overview"
```

Make it executable:
```bash
chmod +x collect-diagnostics.sh
```

Run it:
```bash
# With default config
./collect-diagnostics.sh

# With custom config
./collect-diagnostics.sh --config /etc/mel/config.json --out /tmp/mel-issue-123
```

---

## Manual Collection Steps

For operators who prefer manual collection or need specific diagnostics:

### System Health Check

```bash
# Comprehensive diagnostics (run this first)
mel doctor --config /etc/mel/config.json

# What it reveals:
# - Config file validation and permissions
# - Database accessibility (read/write)
# - Schema version compatibility
# - Transport connectivity
# - Privacy findings
# - Last successful ingest time
```

### Current Status

```bash
# System status snapshot
mel status --config /etc/mel/config.json

# What it reveals:
# - Uptime
# - Database size and message count
# - Node count
# - Per-transport state and metrics
# - Last successful ingest timestamp
```

### Operator Panel

```bash
# Human-readable status
mel panel --config /etc/mel/config.json

# Machine-readable panel
mel panel --format json --config /etc/mel/config.json

# What it reveals:
# - Overall system health state
# - Per-transport health scores
# - Message counts per transport
# - Quick command reference
```

### Configuration Validation

```bash
# Validate config without running service
mel config validate --config /etc/mel/config.json

# What it reveals:
# - Config syntax errors
# - Security issues (permissions)
# - Missing directories
# - Lint warnings with remediation
```

### Transport Diagnostics

```bash
# List all transports and states
mel transports list --config /etc/mel/config.json

# Inspect specific transport
mel inspect transport <transport-name> --config /etc/mel/config.json

# Inspect mesh-wide data
mel inspect mesh --config /etc/mel/config.json

# What it reveals:
# - Transport type and configuration
# - Runtime state vs persisted state
# - Connection metrics
# - Recent errors
# - Message counts by transport
# - Top talkers in mesh
```

### Log Analysis

```bash
# Recent audit logs
mel logs tail --config /etc/mel/config.json

# Replay recent messages
mel replay --config /etc/mel/config.json --limit 50

# Replay from specific node
mel replay --config /etc/mel/config.json --node 12345 --limit 20

# Replay specific message type
mel replay --config /etc/mel/config.json --type text --limit 20
```

### Control Plane History

```bash
# Recent control actions
mel control history --config /etc/mel/config.json

# Filtered by transport
mel control history --transport local-node --config /etc/mel/config.json

# Time range query
mel control history \
  --start 2026-03-20T00:00:00Z \
  --end 2026-03-21T00:00:00Z \
  --config /etc/mel/config.json
```

---

## Sanitization Guide

Before sharing diagnostics externally, sanitize sensitive data:

### Remove Passwords from Config

```bash
# Create sanitized config copy
jq 'walk(if type == "object" then with_entries(
  if .key | test("password|secret|token|key"; "i") then
    .value = "[REDACTED]"
  else
    .
  end
) else . end)' /etc/mel/config.json > config-sanitized.json
```

Fields automatically redacted:
- `password`
- `secret`
- `token`
- `key`
- `credential`

### Redact Node IDs (if needed)

```bash
# Anonymize node IDs in export
jq 'walk(if type == "object" then with_entries(
  if (.key | test("node_id|gateway_id"; "i")) and (.value | type == "string") then
    .value = "[NODE-" + (.value | tostring | split("") | map(ord) | add % 10000 | tostring) + "]"
  else
    .
  end
) else . end)' support-bundle.json > support-bundle-anonymized.json
```

### Handle Location Data

MEL has built-in privacy controls. Verify your settings:

```bash
# Check privacy configuration
mel privacy audit --config /etc/mel/config.json

# Key settings to verify:
# - privacy.redact_exports: Should be true for external sharing
# - privacy.redact_location: Controls location precision
```

If `redact_exports` is enabled, `mel export` automatically:
- Redacts payload text content
- Keeps metadata (timestamps, node IDs unless further anonymized)

### Manual Sanitization Checklist

Before sharing diagnostics:

- [ ] Passwords removed from config files
- [ ] API keys redacted
- [ ] Node IDs anonymized (if required by policy)
- [ ] Location coordinates verified as redacted
- [ ] Personal identifiers removed from message payloads
- [ ] Internal network addresses reviewed

### Sanitization Script

```bash
#!/bin/bash
# sanitize-for-sharing.sh - Prepare diagnostics for external sharing

INPUT_DIR="$1"
OUTPUT_DIR="$2"

if [[ -z "$INPUT_DIR" || -z "$OUTPUT_DIR" ]]; then
  echo "Usage: $0 <input-dir> <output-dir>"
  exit 1
fi

mkdir -p "$OUTPUT_DIR"

# Copy safe files
cp "$INPUT_DIR/doctor.json" "$OUTPUT_DIR/" 2>/dev/null || true
cp "$INPUT_DIR/status.json" "$OUTPUT_DIR/" 2>/dev/null || true
cp "$INPUT_DIR/panel.txt" "$OUTPUT_DIR/" 2>/dev/null || true
cp "$OUTPUT_DIR/transports.json" "$OUTPUT_DIR/" 2>/dev/null || true

# Sanitize logs (remove payload content)
if [[ -f "$INPUT_DIR/logs-recent.json" ]]; then
  jq 'map(del(.details_json))' "$INPUT_DIR/logs-recent.json" > "$OUTPUT_DIR/logs-recent.json"
fi

# Config must be manually reviewed - don't auto-copy
if [[ -f "$INPUT_DIR/config-validation.json" ]]; then
  cp "$INPUT_DIR/config-validation.json" "$OUTPUT_DIR/"
  echo "WARNING: config-validation.json copied - verify no sensitive data exposed"
fi

echo "Sanitized files in: $OUTPUT_DIR"
echo "REVIEW all files before sharing externally"
```

---

## Log Analysis

### Common Log Patterns

#### `transport_connected`

```json
{
  "category": "transport",
  "level": "info",
  "message": "Transport local-node connected"
}
```

**Interpretation:** Transport successfully established connection.
**Action:** Normal operation. Monitor for subsequent ingest events.

#### `transport_failed`

```json
{
  "category": "transport",
  "level": "error",
  "message": "Transport mqtt-broker connection failed: connection refused"
}
```

**Interpretation:** Transport could not establish or maintain connection.
**Common causes:**
- Network unreachable
- Wrong endpoint/port
- Authentication failure
- TLS certificate issues

**Action:**
1. Check endpoint reachability: `telnet <host> <port>`
2. Verify credentials in config
3. Check firewall rules
4. Review broker logs

#### `ingest_received`

```json
{
  "category": "ingest",
  "level": "info",
  "message": "Stored packet from node !abcd1234"
}
```

**Interpretation:** Message successfully received and persisted.
**Action:** Normal operation. Indicates live ingest working.

#### `ingest_dropped`

```json
{
  "category": "ingest",
  "level": "warn",
  "message": "Dropped duplicate packet"
}
```

**Interpretation:** Message deduplication prevented storage.
**Common causes:**
- Duplicate packet from multiple transports
- Retransmitted packet

**Action:** Normal for multi-transport setups. Review if excessive.

#### `db_error`

```json
{
  "category": "storage",
  "level": "error",
  "message": "Database write failed: disk I/O error"
}
```

**Interpretation:** SQLite operation failed.
**Common causes:**
- Disk full
- Database locked by another process
- Corruption
- Permission denied

**Action:**
1. Check disk space: `df -h`
2. Verify database permissions
3. Run integrity check: `sqlite3 mel.db "PRAGMA integrity_check;"`
4. Check for concurrent access

### Log Analysis Script

```bash
#!/bin/bash
# analyze-logs.sh - Quick log pattern analysis

LOGS_FILE="$1"

if [[ -z "$LOGS_FILE" ]]; then
  echo "Usage: $0 <logs-json-file>"
  exit 1
fi

echo "=== Log Analysis ==="
echo ""

echo "Error count by category:"
jq -r 'group_by(.category) | map({category: .[0].category, errors: map(select(.level == "error")) | length}) | .[] | "\(.category): \(.errors)"' "$LOGS_FILE"

echo ""
echo "Transport events:"
jq -r 'map(select(.category == "transport")) | group_by(.message) | map({message: .[0].message, count: length}) | sort_by(-.count) | .[] | "\(.count)x: \(.message)"' "$LOGS_FILE"

echo ""
echo "Recent errors:"
jq -r 'map(select(.level == "error")) | .[-5:] | .[] | "[\(.created_at)] \(.category): \(.message)"' "$LOGS_FILE"
```

---

## Metrics Interpretation

### Reading /metrics Output

The `/metrics` endpoint returns JSON with the following fields:

```json
{
  "total_messages": 15420,
  "nodes": 47,
  "last_ingest_time": "2026-03-20T14:32:11Z",
  "ingest_rate_per_sec": 2.3,
  "dead_letters_total": 15,
  "transports": {
    "local-node": {
      "messages": 8200,
      "state": "live"
    }
  }
}
```

### Key Metrics

#### `total_messages`

**Meaning:** Total messages stored in database.
**Healthy signs:**
- Steady increase during operation
- Non-zero after mesh activity

**Warning signs:**
- Stagnant during expected activity (possible ingest failure)
- Zero with live transports (configuration issue)

#### `ingest_rate_per_sec`

**Meaning:** Average messages ingested per second over last 5 minutes.
**Typical values:**
- Quiet mesh: 0.1-0.5 msg/s
- Active mesh: 1-5 msg/s
- Busy event: 10+ msg/s spikes

**Warning signs:**
- Zero with active mesh (transport down)
- Sudden drop (connectivity issue)
- Unusually high (possible loop/duplicate storm)

#### `dead_letters_total`

**Meaning:** Messages that failed processing and were quarantined.
**Healthy signs:**
- Low or zero count
- Stable over time

**Warning signs:**
- Increasing rapidly (systemic issue)
- High ratio to total_messages (>1% concerning)

**Investigation:**
```bash
# View recent dead letters
sqlite3 mel.db "SELECT transport_name, reason, created_at FROM dead_letters ORDER BY id DESC LIMIT 20;"
```

#### `control_metrics` (if control enabled)

```json
{
  "control_actions_total": 150,
  "control_decisions_total": 150,
  "actions_by_type": {
    "allow": 145,
    "block": 5
  }
}
```

**Meaning:** Control plane activity summary.
**Healthy signs:**
- Actions match decisions count
- Expected allow/block ratio for your policy

**Warning signs:**
- Actions != decisions (processing gap)
- Unexpected block rate (policy too aggressive)

### Metrics Trending

```bash
#!/bin/bash
# trend-metrics.sh - Track metrics over time

INTERVAL="${1:-60}"
COUNT="${2:-10}"
CONFIG="${CONFIG:-configs/mel.example.json}"

echo "Collecting metrics every ${INTERVAL}s for ${COUNT} iterations..."
echo "timestamp,total_messages,nodes,ingest_rate,dead_letters"

for i in $(seq 1 $COUNT); do
  TS=$(date -Iseconds)
  METRICS=$(mel status --config "$CONFIG" 2>/dev/null)
  TOTAL=$(echo "$METRICS" | jq -r '.database.message_count // 0')
  NODES=$(echo "$METRICS" | jq -r '.database.node_count // 0')
  # Calculate rate from two samples
  sleep $INTERVAL
  METRICS2=$(mel status --config "$CONFIG" 2>/dev/null)
  TOTAL2=$(echo "$METRICS2" | jq -r '.database.message_count // 0')
  RATE=$(echo "scale=2; ($TOTAL2 - $TOTAL) / $INTERVAL" | bc -l 2>/dev/null || echo "0")
  DL=$(sqlite3 $(jq -r '.database.path' <<< "$METRICS2") "SELECT COUNT(*) FROM dead_letters;" 2>/dev/null || echo "0")
  echo "$TS,$TOTAL2,$NODES,$RATE,$DL"
done
```

---

## Database Queries

Use these SQLite queries for deep investigation. Run with:

```bash
sqlite3 /path/to/mel.db "<query>"
```

### Recent Transport Errors

```sql
-- Recent transport errors from audit logs
SELECT 
  created_at,
  message,
  json_extract(details_json, '$.transport') as transport
FROM audit_logs 
WHERE category = 'transport' AND level = 'error'
ORDER BY id DESC 
LIMIT 20;
```

### Message Counts by Transport

```sql
-- Messages stored per transport
SELECT 
  transport_name,
  COUNT(*) as message_count,
  MIN(rx_time) as first_seen,
  MAX(rx_time) as last_seen
FROM messages 
GROUP BY transport_name
ORDER BY message_count DESC;
```

### Dead Letter Patterns

```sql
-- Dead letter summary by reason
SELECT 
  transport_name,
  reason,
  COUNT(*) as count,
  MAX(created_at) as last_occurrence
FROM dead_letters
GROUP BY transport_name, reason
ORDER BY count DESC;
```

```sql
-- Recent dead letter details
SELECT 
  created_at,
  transport_name,
  transport_type,
  reason,
  SUBSTR(payload_hex, 1, 50) as payload_preview
FROM dead_letters
ORDER BY id DESC
LIMIT 10;
```

### Control Action Outcomes

```sql
-- Control action summary
SELECT 
  transport_name,
  action,
  reason,
  COUNT(*) as count,
  MAX(created_at) as last_occurrence
FROM control_actions
GROUP BY transport_name, action, reason
ORDER BY count DESC;
```

### Node Activity Analysis

```sql
-- Most active nodes
SELECT 
  n.node_id,
  n.long_name,
  n.last_seen,
  COUNT(m.id) as message_count
FROM nodes n
LEFT JOIN messages m ON n.node_num = m.from_node
GROUP BY n.node_num
ORDER BY message_count DESC
LIMIT 20;
```

### Transport State History

```sql
-- Transport runtime status
SELECT 
  transport_name,
  runtime_state,
  total_messages,
  last_success_at,
  last_error
FROM transport_runtime_status
WHERE enabled = 1
ORDER BY transport_name;
```

### Hourly Message Volume

```sql
-- Messages per hour (last 24 hours)
SELECT 
  strftime('%Y-%m-%d %H:00', created_at) as hour,
  COUNT(*) as message_count
FROM messages
WHERE created_at > datetime('now', '-24 hours')
GROUP BY hour
ORDER BY hour;
```

### Database Health

```sql
-- Database integrity check
PRAGMA integrity_check;

-- Database size info
SELECT 
  page_count * page_size as size_bytes,
  page_count,
  page_size,
  freelist_count
FROM pragma_page_count(), pragma_page_size(), pragma_freelist_count();

-- Table row counts
SELECT 
  'messages' as table_name, COUNT(*) as rows FROM messages
UNION ALL
SELECT 'nodes', COUNT(*) FROM nodes
UNION ALL
SELECT 'dead_letters', COUNT(*) FROM dead_letters
UNION ALL
SELECT 'audit_logs', COUNT(*) FROM audit_logs;
```

### Connection Quality Analysis

```sql
-- Nodes with poor signal quality recently
SELECT 
  node_id,
  long_name,
  last_snr,
  last_rssi,
  last_seen
FROM nodes
WHERE last_snr < 5 OR last_rssi < -90
  AND last_seen > datetime('now', '-1 hour')
ORDER BY last_rssi ASC;
```

---

## Support Bundle Creation

### Using `mel export`

The export command creates a JSON bundle suitable for analysis:

```bash
# Export to file
mel export --config /etc/mel/config.json --out mel-support-$(date +%Y%m%d).json

# Export and compress
mel export --config /etc/mel/config.json | gzip > mel-support.json.gz

# Export to stdout for piping
mel export --config /etc/mel/config.json | jq '.nodes | length'
```

### Export Contents

The bundle includes (limited to last 250 records each):

- `nodes`: Node registry with metadata
- `messages`: Recent messages (may be redacted)
- `dead_letters`: Failed message records
- `audit_logs`: Recent system events
- `exported_at`: Timestamp
- `redacted`: Whether privacy redaction was applied

### Complete Backup vs Export

| Command | Use Case | Contents |
|---------|----------|----------|
| `mel export` | Analysis, support | Recent data only (250 rows), may be redacted |
| `mel backup create` | Full backup/restore | Complete database + config |

### Export Analysis Script

```bash
#!/bin/bash
# analyze-export.sh - Quick analysis of exported bundle

BUNDLE="$1"

if [[ -z "$BUNDLE" ]]; then
  echo "Usage: $0 <bundle.json>"
  exit 1
fi

echo "=== Export Analysis ==="
echo "Export time: $(jq -r '.exported_at' "$BUNDLE")"
echo "Redacted: $(jq -r '.redacted' "$BUNDLE")"
echo ""

echo "Node count: $(jq '.nodes | length' "$BUNDLE")"
echo "Message count: $(jq '.messages | length' "$BUNDLE")"
echo "Dead letters: $(jq '.dead_letters | length' "$BUNDLE")"
echo "Audit entries: $(jq '.audit_logs | length' "$BUNDLE")"
echo ""

echo "Messages by transport:"
jq -r '.messages | group_by(.transport_name) | map({name: .[0].transport_name, count: length}) | .[] | "  \(.name): \(.count)"' "$BUNDLE"

echo ""
echo "Dead letter reasons:"
jq -r '.dead_letters | group_by(.reason) | map({reason: .[0].reason, count: length}) | .[] | "  \(.reason): \(.count)"' "$BUNDLE"
```

---

## Automation Scripts

### Cron-based Health Checks

```bash
#!/bin/bash
# mel-health-check.sh - Run from cron for monitoring

CONFIG="/etc/mel/config.json"
ALERT_EMAIL="ops@example.com"
LOG_FILE="/var/log/mel-health.log"

# Run doctor and capture result
OUTPUT=$(mel doctor --config "$CONFIG" 2>&1)
EXIT_CODE=$?

# Log result
echo "$(date -Iseconds) exit=$EXIT_CODE" >> "$LOG_FILE"

# Alert on failure
if [[ $EXIT_CODE -ne 0 ]]; then
  echo "$OUTPUT" | mail -s "MEL Health Check Failed" "$ALERT_EMAIL"
fi

# Check for dead letter spike
DB_PATH=$(jq -r '.storage.database_path' "$CONFIG")
DEAD_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM dead_letters WHERE created_at > datetime('now', '-1 hour');")
if [[ $DEAD_COUNT -gt 100 ]]; then
  echo "Dead letter spike: $DEAD_COUNT in last hour" | mail -s "MEL Alert" "$ALERT_EMAIL"
fi
```

Add to crontab:
```
# Run every 5 minutes
*/5 * * * * /usr/local/bin/mel-health-check.sh
```

### Transport Flap Detection

```bash
#!/bin/bash
# detect-transport-flapping.sh

CONFIG="/etc/mel/config.json"
DB_PATH=$(jq -r '.storage.database_path' "$CONFIG")

# Count transport state changes in last 10 minutes
FLAPS=$(sqlite3 "$DB_PATH" <<EOF
SELECT transport_name, COUNT(*) as changes
FROM audit_logs
WHERE category = 'transport'
  AND message LIKE '%connected%'
  AND created_at > datetime('now', '-10 minutes')
GROUP BY json_extract(details_json, '$.transport')
HAVING changes > 3;
EOF
)

if [[ -n "$FLAPS" ]]; then
  echo "Transport flapping detected:"
  echo "$FLAPS"
fi
```

### Diagnostic Report Generator

```bash
#!/bin/bash
# generate-report.sh - Create HTML diagnostic report

CONFIG="$1"
OUTPUT="${2:-mel-report.html}"

if [[ -z "$CONFIG" ]]; then
  echo "Usage: $0 <config-path> [output-file]"
  exit 1
fi

# Collect data
DOCTOR=$(mel doctor --config "$CONFIG" 2>/dev/null || echo '{}')
STATUS=$(mel status --config "$CONFIG" 2>/dev/null || echo '{}')
PANEL=$(mel panel --format json --config "$CONFIG" 2>/dev/null || echo '{}')

# Generate HTML report
cat > "$OUTPUT" << 'HTMLHEAD'
<!DOCTYPE html>
<html>
<head>
  <title>MEL Diagnostics Report</title>
  <style>
    body { font-family: sans-serif; margin: 2em; }
    .section { margin: 2em 0; padding: 1em; border: 1px solid #ccc; }
    .healthy { color: green; }
    .warning { color: orange; }
    .critical { color: red; }
    pre { background: #f5f5f5; padding: 1em; overflow-x: auto; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
    th { background: #f0f0f0; }
  </style>
</head>
<body>
  <h1>MEL Diagnostics Report</h1>
  <p>Generated: $(date)</p>
HTMLHEAD

# Add sections
echo '<div class="section"><h2>System Status</h2><pre>' >> "$OUTPUT"
echo "$STATUS" | jq . >> "$OUTPUT"
echo '</pre></div>' >> "$OUTPUT"

echo '<div class="section"><h2>Health Panel</h2><pre>' >> "$OUTPUT"
echo "$PANEL" | jq . >> "$OUTPUT"
echo '</pre></div>' >> "$OUTPUT"

echo '<div class="section"><h2>Doctor Findings</h2>' >> "$OUTPUT"
FINDINGS=$(echo "$DOCTOR" | jq '.findings | length')
if [[ "$FINDINGS" == "0" ]]; then
  echo '<p class="healthy">No issues found</p>' >> "$OUTPUT"
else
  echo "<p class=\"critical\">$FINDINGS finding(s) detected</p>" >> "$OUTPUT"
  echo '<pre>' >> "$OUTPUT"
  echo "$DOCTOR" | jq '.findings' >> "$OUTPUT"
  echo '</pre>' >> "$OUTPUT"
fi
echo '</div>' >> "$OUTPUT"

echo '</body></html>' >> "$OUTPUT"

echo "Report generated: $OUTPUT"
```

---

## Escalation Checklist

Before escalating to support:

- [ ] Run `mel doctor` and address all critical findings
- [ ] Collect diagnostics using quick collection script
- [ ] Verify config with `mel config validate`
- [ ] Check transport states match expectations
- [ ] Review recent logs for error patterns
- [ ] Verify database accessibility
- [ ] Document recent changes to config/environment
- [ ] Sanitize data for external sharing
- [ ] Include:
  - MEL version (`mel version`)
  - OS and architecture
  - Transport types in use
  - Time issue first observed
  - Expected vs actual behavior

---

## Related Documentation

- [CLI Reference](./cli-reference.md) - Complete command documentation
- [Diagnostics](./diagnostics.md) - Runtime truth and diagnostic concepts
- [Troubleshooting](./troubleshooting.md) - Common issues and solutions
- [Troubleshooting Transports](./troubleshooting-transports.md) - Transport-specific issues
- [Incident Triage](./incident-triage.md) - Structured incident response
