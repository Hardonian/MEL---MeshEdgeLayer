# Runbook: Common Connectivity & Ingest Issues

This runbook covers scenarios where transports are failing to connect or are connected but not ingesting data.

---

## 🚦 Transport Down / Reconnect Churn

### Symptom
Transport remains stuck in `attempting`, `configured_offline`, or `error` state. Logs show repeated connection attempts followed by immediate disconnects.

### Diagnostic Steps

```bash
# Quick health check
mel doctor

# Inspect specific transport
mel inspect transport <transport-name>

# Follow transport logs
mel logs tail --transport <transport-name> --follow
```

Check these specific fields in the JSON output:
- `consecutive_timeouts`: Sequential timeout failures.
- `retry_status`: Current backoff state.
- `last_error`: Most recent error message.

### Resolution Steps

1. **Verify Endpoint Accessibility**
   - For MQTT: `telnet <host> 1883`
   - For TCP: `telnet <host> <port>`
   - For Serial: `ls -l /dev/serial/by-id/...`

2. **Check Credential Hashes**
   ```bash
   mel inspect transport <name> --show-credentials-hashes
   ```
   Compare with expected hashes to ensure no configuration drift or corruption.

3. **Adjust Timeout Settings** (if network is latent)
   ```bash
   # Increase read/write timeouts if on high-latency links
   mel transport update <name> --read-timeout 30s --write-timeout 10s
   ```

4. **Force Reconnection**
   ```bash
   # Cycle the transport state machine
   mel transport disconnect <name>
   mel transport connect <name>
   ```

---

## 📡 MQTT Subscription Issues

### Symptom
MQTT transport shows `connected` but no messages are being received.

### Diagnostic Steps

```bash
# Verify subscription state
mel status --transport <transport-name>
```

### Resolution Steps

1. **Verify Topic Filter**
   Ensure your `topic` in `mel.json` matches the broker's structure (e.g., `msh/US/2/e/#`).

2. **Review Broker ACLs**
   Ensure the `client_id` used by MEL has `SUBSCRIBE` permissions for the configured topic.

3. **Check QoS Compatibility**
   MEL defaults to **QoS 1** for reliability. If your broker only supports QoS 0, you must update the config:
   ```json
   "mqtt_qos": 0
   ```

---

## Prevention & Best Practices

- **Use Persistent IDs**: For MQTT, use a fixed `client_id` to ensure persistent sessions work correctly.
- **Serial Persistence**: Always use `/dev/serial/by-id/...` instead of `/dev/ttyUSB0` to avoid device path shifts on reboot.
- **Monitor Timeouts**: Alert if `consecutive_timeouts` exceeds **3** for more than 5 minutes.
