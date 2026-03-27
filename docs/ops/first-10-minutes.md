# First 10 Minutes with MEL

Welcome, operator. This guide gets you from installation to a running MEL instance in 10 minutes. Each section has a time budget—move on if you hit it.

---

## Minute 0-2: Verify Installation

### Check MEL is installed

```bash
mel version
```

**Expected output:**

```
MEL v0.8.2
  Go: go1.22.4
  SQLite: 3.45.1
  Build: 2024-03-15T14:23:00Z
```

### Verify binary location

```bash
which mel
```

**Expected:** Path to your binary (e.g., `/usr/local/bin/mel` or `./bin/mel` for local builds).

**If missing:** Check your `PATH` or re-install from the [installation guide](./installation.md).

---

## Minute 2-4: Initialize Configuration

### Create a configuration file

```bash
mel init --config ./mel.json
```

**Expected output:**

```
Configuration initialized: ./mel.json
  Node ID: mel-7f3a9b2e
  Data directory: ./mel-data/
  API port: 8080
```

### Review the generated config

```bash
cat mel.json
```

Key fields to verify:

| Field | Purpose | Default |
|-------|---------|---------|
| `node.id` | Unique identifier for this node | Auto-generated |
| `storage.data_dir` | Where packets and state live | `./mel-data/` |
| `bind.api` | Health and control interface | `127.0.0.1:8080` |
| `logging.level` | Verbosity of logs | `info` |

### Set proper permissions

```bash
chmod 600 ./mel.json
```

This prevents other users from reading potentially sensitive transport credentials.

---

## Minute 4-6: Validate and Doctor

### Validate configuration syntax

```bash
mel config validate --config ./mel.json
```

**Success output:**

```
Configuration valid.
  Node ID: mel-7f3a9b2e
  Storage: ./mel-data/
  API: :8080
  Transports: 0 configured
```

**Failure output:**

```
Configuration invalid:
  - storage.data_dir: directory does not exist: /bad/path
```

Fix errors before proceeding.

### Run doctor for system checks

```bash
mel doctor --config ./mel.json
```

**Healthy system output:**

```
System Check Results:
  [PASS] Binary: mel v0.8.2
  [PASS] Config: ./mel.json readable
  [PASS] Storage: ./mel-data/ writable
  [PASS] API port: :8080 available
  [WARN] Transports: no transports configured
```

### Interpret doctor output

| Status | Meaning | Action |
|--------|---------|--------|
| `[PASS]` | Check passed | None |
| `[WARN]` | Non-critical issue | Review, may proceed |
| `[FAIL]` | Critical blocker | Fix before starting |

Common warnings at this stage:
- `no transports configured` — expected; you'll add these later
- `API port already bound` — another MEL instance may be running

---

## Minute 6-8: Start MEL

### Start with debug logging

```bash
mel serve --debug --config ./mel.json
```

**Expected startup output:**

```
INFO[0000] MEL starting                                  version=0.8.2 node=mel-7f3a9b2e
INFO[0000] Storage initialized                           path=./mel-data/
INFO[0000] API server starting                           bind=:8080
DEBU[0000] Transport manager initialized                 transports=0
INFO[0000] MEL ready                                     uptime=0s
```

### Or start as a service (if systemd configured)

```bash
sudo systemctl start mel
sudo systemctl status mel
```

**Check logs:**

```bash
sudo journalctl -u mel -f
```

### Verify startup in logs

Look for these key log lines:

| Log | Meaning |
|-----|---------|
| `MEL starting` | Process booted |
| `Storage initialized` | SQLite database ready |
| `API server starting` | Health endpoints available |
| `MEL ready` | All subsystems initialized |

**Leave MEL running.** Open a new terminal for the next section.

---

## Minute 8-10: Verify Operation

### Check health endpoint (liveness only)

```bash
curl http://127.0.0.1:8080/healthz
```

**Expected:**

```json
{"ok":true}
```

This proves the HTTP process responds — **not** MQTT, serial, or ingest.

### Check readiness endpoint

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8080/readyz
curl http://127.0.0.1:8080/readyz
# Same semantics as:
curl http://127.0.0.1:8080/api/v1/readyz
```

**Expected:** HTTP **200** when ready (idle with no enabled transports, or at least one enabled transport **ingesting**); HTTP **503** when not ready or snapshot assembly failed. Body includes `ready`, `status`, `reason_codes`, `ingest_ready`, and `transports` evidence. Use `GET /api/v1/status` for full transport truth; use `mel doctor` for host-level checks.

### Check CLI status

```bash
mel status --config ./mel.json
```

**Expected:**

```
Node: mel-7f3a9b2e
  Uptime: 2m15s
  Storage: ./mel-data/ (3.2 MB)
  API: :8080 (listening)
  Transports: 0 active, 0 configured
```

### Check panel (operator dashboard)

```bash
mel panel --config ./mel.json
```

**Expected:**

```
┌─────────────────────────────────────────┐
│ MEL Panel          node: mel-7f3a9b2e   │
├─────────────────────────────────────────┤
│ Health:      ✅ healthy                 │
│ Uptime:      2m30s                      │
│ Storage:     3.2 MB                     │
│ Packets:     0                          │
│ Transports:  none configured            │
└─────────────────────────────────────────┘
```

---

## What to Expect: Transport States

When you configure transports, you'll see these states:

| State | Meaning | When to be concerned |
|-------|---------|---------------------|
| `unconfigured` | No transport of this type set up | Normal for unused types |
| `unsupported` | Transport type not available in this build | Normal for extension nodes |
| `historical_only` | Database has old packets, no live connection | Expected for restored nodes |
| `connected_no_ingest` | Transport connected, waiting for first packet | Normal during startup |
| `ingesting` | Live packets are being stored | Healthy operational state |
| `degraded` | Connected but dropping packets | Check logs, resource usage |
| `failed` | Connection lost or initialization failed | Immediate attention needed |

**State progression:**

```
unconfigured → connected_no_ingest → ingesting
                      ↓
               historical_only (on reconnect)
```

---

## If Something Goes Wrong

### Common issues and quick fixes

**`mel: command not found`**

```bash
# Add to PATH for this session
export PATH=$PATH:/path/to/mel/bin
# Or use full path
./bin/mel version
```

**`Config file not found: ./mel.json`**

```bash
# You may be in the wrong directory
cd /path/to/config
# Or specify full path
mel serve --config /etc/mel/mel.json
```

**`API port :8080 already in use`**

```bash
# Find the process
lsof -i :8080
# Or change port in config
# Edit mel.json: "bind": {"api": "127.0.0.1:8081"}
```

**`Storage directory not writable: ./mel-data/`**

```bash
# Fix permissions
mkdir -p ./mel-data
chmod 755 ./mel-data
# Or change data directory in config
```

**`healthz` returns unhealthy**

```bash
# Check detailed status
mel doctor --config ./mel.json
# Check logs for errors
mel serve --debug 2>&1 | grep -i error
```

### When to consult the troubleshooting guide

- Doctor shows `[FAIL]` for multiple checks
- Health endpoint returns 503 repeatedly
- MEL crashes within 30 seconds of startup
- Error messages mention "database locked" or "corruption"

→ See [troubleshooting.md](./troubleshooting.md)

### When to check runbooks

- Production outage (node marked failed)
- Data loss suspected (packet counts drop)
- Performance degradation (high latency)

→ See [runbooks/](../runbooks/)

---

## Next Steps

You're now running MEL. Here's where to go next:

| Goal | Resource |
|------|----------|
| Configure transports (MQTT, UDP, etc.) | [configuration.md](./configuration.md) |
| Set up MQTT ingestion | [../guides/mqtt-transport.md](../guides/mqtt-transport.md) |
| Set up UDP packet capture | [../guides/udp-transport.md](../guides/udp-transport.md) |
| Configure monitoring and alerts | [../guides/monitoring-setup.md](../guides/monitoring-setup.md) |
| Understand the full config schema | [configuration.md](./configuration.md) |
| Deploy to production | [deployment.md](./deployment.md) |

### Quick wins for the next hour

1. **Add your first transport:**
   ```bash
   mel transports add mqtt --config ./mel.json --broker tcp://broker.local:1883
   ```

2. **Watch live packets:**
   ```bash
   mel packets tail --config ./mel.json
   ```

3. **Export some data:**
   ```bash
   mel packets export --config ./mel.json --since 1h --format json
   ```

---

## Quick Reference Card

```bash
# Installation check
mel version

# Init and validate
mel init --config ./mel.json
chmod 600 ./mel.json
mel config validate --config ./mel.json
mel doctor --config ./mel.json

# Start and verify
mel serve --debug --config ./mel.json &
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/readyz
mel status --config ./mel.json
mel panel --config ./mel.json
```

**Status at a glance:**

| Check | Command | Good | Bad |
|-------|---------|------|-----|
| Health | `curl :8080/healthz` | `{"status":"healthy"}` | `unhealthy` or timeout |
| Ready | `curl :8080/readyz` | `{"status":"ready"}` | `not_ready` |
| CLI Status | `mel status` | Shows uptime | Shows errors |

---

Welcome to MEL. Your node is ready for configuration.
