# Runbook: Dead Letter Growth & Analysis

Dead letters are messages that were successfully received but failed to be processed or persisted properly. This runbook helps you identify why they're growing and how to resolve the underlying causes.

---

## 🚦 Symptom & Diagnostic Steps

### Symptom

`dead_letters` table row count increasing rapidly. The Web Dashboard shows growing dead letter queue alerts.

### Diagnostic Steps

```bash
# Check dead letter statistics
mel status --include-dead-letters

# Review recent dead letters with reasons
mel db query "SELECT reason, COUNT(*) FROM dead_letters WHERE created_at > datetime('now', '-1 hour') GROUP BY reason"
```

Review the `reason` field for common crash/fail signatures:

- `protobuf_decode_error`: Malformed Protobuf payload from node.
- `unsupported_payload_type`: Message type not yet supported by MEL.
- `database_constraint_violation`: DB schema/constraint mismatch.
- `transport_timeout`: Downstream processing link timed out.

### Healthy Baseline

- Dead letter growth: **< 10/hour** for a medium-to-large active mesh.
- Most common reason: Transient `transport_timeout` (often retried successfully).
- Critical failure sign: Large clusters of `protobuf_decode_error`.

---

## 🛠️ Resolution Steps

### 1. Identify Source & Format

```bash
# Correlate dead letters with source transport
mel db query "SELECT transport_name, COUNT(*) FROM dead_letters GROUP BY transport_name"
```

If one transport is responsible for all failures, focus on its settings (e.g., baud rate, TCP stream integrity).

### 2. Review Payload Content (Safely)

```bash
# Extract a sample dead letter payload (first 50 chars)
mel db query "SELECT SUBSTR(payload_hex, 1, 50) as preview FROM dead_letters LIMIT 5"
```

Check if the payload is purely garbage (e.g., serial noise) or a structured packet with a missing decoder.

### 3. Adjust Retention & Cleanup

```javascript
// mel.json
"storage": {
  "dead_letter_retention_days": 3
}
```

If the dead letters are known garbage (e.g., noise on a serial line), reduce retention to save disk space.

---

## 🚀 Prevention

- **Monitor Rate of Change**: Alert when your `dead_letter_growth` exceeds **1%** of your `ingest_rate`.
- **Verify Transport Integrity**: For serial nodes, check for EMI/RFI cable noise if `protobuf_decode_errors` are frequent.
- **Support New Payloads**: If dead letters are `unsupported_payload_type`, share a sample with the [MEL contributors](../../CONTRIBUTING.md) to add support.
