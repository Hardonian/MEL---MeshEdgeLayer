# MEL Incident Triage Checklist

**Incident ID:** ________________  
**Engineer:** ________________  
**Start Time:** ________________  
**Severity:** □ SEV1 □ SEV2 □ SEV3 □ SEV4

---

## Initial Assessment (2 minutes)

- [ ] **Check MEL process health**
  ```bash
  systemctl status mel
  ps aux | grep mel
  pgrep -a mel
  ```

- [ ] **Quick health check**
  ```bash
  curl -s http://localhost:8080/healthz
  curl -s http://localhost:8080/readyz
  ```

- [ ] **Check for recent restarts or crashes**
  ```bash
  journalctl -u mel --since "30 minutes ago"
  systemctl status mel | grep -E "(Active|since)"
  ```

---

## Configuration Validation

- [ ] **Config file permissions (must be 0600)**
  ```bash
  ls -la /etc/mel/config.yaml
  stat -c "%a" /etc/mel/config.yaml
  ```

- [ ] **Config validation**
  ```bash
  mel config validate
  ```

- [ ] **Check for unsupported transport types**
  ```bash
  grep -E "type:\s*(http|grpc|mqtt)" /etc/mel/config.yaml
  # Verify types are in supported list
  ```

- [ ] **Verify retention settings aren't causing issues**
  ```bash
  grep retention /etc/mel/config.yaml
  df -h /var/lib/mel
  ```

---

## Transport Health Assessment

- [ ] **Run mel doctor**
  ```bash
  mel doctor
  ```

- [ ] **Check transport states in /api/v1/status**
  ```bash
  curl -s http://localhost:8080/api/v1/status | jq '.transports'
  ```

- [ ] **Identify transports in error state**
  ```bash
  curl -s http://localhost:8080/api/v1/status | jq '.transports[] | select(.state == "error")'
  ```

- [ ] **Check for correlated failures across transports**
  ```bash
  curl -s http://localhost:8080/api/v1/status | jq '.transports | group_by(.error) | map({error: .[0].error, count: length})'
  ```

---

## Database Health

- [ ] **Check database file exists and is writable**
  ```bash
  ls -la /var/lib/mel/mel.db
  test -w /var/lib/mel/mel.db && echo "Writable" || echo "NOT writable"
  ```

- [ ] **Check schema version**
  ```bash
  sqlite3 /var/lib/mel/mel.db "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"
  mel doctor | grep -i schema
  ```

- [ ] **Look for DB errors in logs**
  ```bash
  journalctl -u mel --since "1 hour ago" | grep -iE "(sqlite|database|db error)"
  ```

- [ ] **Check disk space for data directory**
  ```bash
  df -h /var/lib/mel
  du -sh /var/lib/mel
  ```

---

## Ingest Pipeline

- [ ] **Verify at least one transport is enabled**
  ```bash
  curl -s http://localhost:8080/api/v1/status | jq '.transports[] | select(.enabled == true) | .name'
  ```

- [ ] **Check last_successful_ingest timestamp**
  ```bash
  curl -s http://localhost:8080/api/v1/status | jq '.last_successful_ingest'
  date -d @$(curl -s http://localhost:8080/api/v1/status | jq '.last_successful_ingest')
  ```

- [ ] **Check dead_letters count**
  ```bash
  sqlite3 /var/lib/mel/mel.db "SELECT COUNT(*) FROM dead_letters WHERE created_at > datetime('now', '-24 hours');"
  curl -s http://localhost:8080/api/v1/status | jq '.dead_letters_count'
  ```

- [ ] **Look for recent messages in /api/v1/messages**
  ```bash
  curl -s "http://localhost:8080/api/v1/messages?limit=5" | jq '.messages | length'
  ```

---

## Control Plane (if applicable)

- [ ] **Check control mode (advisory vs guarded_auto)**
  ```bash
  curl -s http://localhost:8080/api/v1/control/status | jq '.mode'
  grep -E "mode:\s*(advisory|guarded_auto)" /etc/mel/config.yaml
  ```

- [ ] **Review recent control decisions**
  ```bash
  curl -s http://localhost:8080/api/v1/control/history?limit=10
  mel control history --limit 10
  ```

- [ ] **Check for denied actions and reasons**
  ```bash
  curl -s http://localhost:8080/api/v1/control/history | jq '.decisions[] | select(.action == "deny")'
  ```

- [ ] **Verify emergency_disable is false**
  ```bash
  curl -s http://localhost:8080/api/v1/control/status | jq '.emergency_disabled'
  grep emergency_disabled /etc/mel/config.yaml
  ```

---

## Common Scenarios with Decision Trees

### 1. "No data ingesting"

```
□ Check transport config
  └─ □ mel config validate
  └─ □ grep -A5 "transports:" /etc/mel/config.yaml
     └─ Config looks valid? ──YES──> □ Check connectivity
                                      └─ □ telnet <endpoint> <port>
                                      └─ □ curl -v <endpoint>
                                         └─ Connectivity OK? ──YES──> □ Check permissions
                                                                       └─ □ Check API keys/tokens
                                                                       └─ □ Verify TLS certificates
                                                                       └─ □ Check firewall rules
```

### 2. "Transport in error state"

```
□ Check logs
  └─ □ journalctl -u mel -f
     └─ □ Identify error pattern
        └─ □ Check dead letters
           └─ □ sqlite3 /var/lib/mel/mel.db "SELECT * FROM dead_letters ORDER BY created_at DESC LIMIT 5;"
              └─ □ Review recent incidents
                 └─ □ mel logs tail --since "2h"
```

### 3. "High dead letter count"

```
□ Check for malformed messages
  └─ □ sqlite3 /var/lib/mel/mel.db "SELECT error_reason, COUNT(*) FROM dead_letters GROUP BY error_reason;"
     └─ Pattern identified? ──YES──> □ Check transport timeout settings
                                      └─ □ grep -E "(timeout|retry)" /etc/mel/config.yaml
                                      └─ □ Adjust batch_size if needed
```

### 4. "Control actions denied"

```
□ Check mode
  └─ □ curl -s http://localhost:8080/api/v1/control/status | jq '.mode'
     └─ Mode = advisory? ──YES──> □ Check policy
                                   └─ □ grep -A20 "policy:" /etc/mel/config.yaml
                                      └─ Policy looks OK? ──YES──> □ Check cooldown/budget
                                                                    └─ □ curl http://localhost:8080/api/v1/control/status | jq '.budget_remaining,.cooldown_until'
```

### 5. "Database errors"

```
□ Check disk space
  └─ □ df -h /var/lib/mel
     └─ Disk OK? ──YES──> □ Check permissions
                           └─ □ ls -la /var/lib/mel/
                              └─ Permissions OK? ──YES──> □ Check schema version
                                                           └─ □ mel doctor
                                                           └─ □ sqlite3 /var/lib/mel/mel.db "PRAGMA user_version;"
```

---

## Data Collection for Escalation

**Attach the following to your escalation ticket:**

- [ ] **mel doctor output**
  ```bash
  mel doctor > /tmp/mel-doctor-$(date +%Y%m%d-%H%M%S).txt
  ```

- [ ] **mel status output**
  ```bash
  mel status --json > /tmp/mel-status-$(date +%Y%m%d-%H%M%S).json
  ```

- [ ] **Recent audit logs**
  ```bash
  mel logs tail --since "1 hour" > /tmp/mel-logs-$(date +%Y%m%d-%H%M%S).txt
  ```

- [ ] **Control history (if applicable)**
  ```bash
  curl -s http://localhost:8080/api/v1/control/history?limit=50 > /tmp/mel-control-$(date +%Y%m%d-%H%M%S).json
  ```

- [ ] **Dead letters sample**
  ```bash
  sqlite3 /var/lib/mel/mel.db ".mode json" "SELECT * FROM dead_letters ORDER BY created_at DESC LIMIT 20;" > /tmp/mel-deadletters-$(date +%Y%m%d-%H%M%S).json
  ```

- [ ] **Config (sanitized)**
  ```bash
  cat /etc/mel/config.yaml | sed -E 's/(password|token|key|secret): .+/\1: [REDACTED]/' > /tmp/mel-config-$(date +%Y%m%d-%H%M%S).yaml
  ```

**Bundle for escalation:**
```bash
tar czf /tmp/mel-incident-$(date +%Y%m%d-%H%M%S).tar.gz /tmp/mel-*-$(date +%Y%m%d-%H%M%S).*
```

---

## Severity Classification

| Severity | Criteria | Response Time |
|----------|----------|---------------|
| **SEV1** | Complete ingest failure with data loss risk | Immediate |
| **SEV2** | Partial ingest failure or control plane issues | < 30 min |
| **SEV3** | Degraded performance or advisory warnings | < 4 hours |
| **SEV4** | Cosmetic issues or documentation gaps | < 24 hours |

### Classification Guide

**SEV1 Indicators:**
- [ ] All transports in error state
- [ ] No successful ingest in > 15 minutes
- [ ] Database corruption detected
- [ ] Disk full on data partition
- [ ] emergency_disabled = true (unintended)

**SEV2 Indicators:**
- [ ] > 50% of transports in error state
- [ ] Dead letter count rising rapidly
- [ ] Control plane rejecting valid actions
- [ ] Single transport failure with no redundancy

**SEV3 Indicators:**
- [ ] Elevated error rate but ingest continuing
- [ ] Advisory warnings in logs
- [ ] Performance degradation > 20%
- [ ] Intermittent transport failures

**SEV4 Indicators:**
- [ ] Log noise or false-positive warnings
- [ ] Dashboard display issues
- [ ] Documentation inconsistencies
- [ ] Non-production environment issues

---

## Resolution Notes

**Root Cause:** _______________________________________________

**Action Taken:** _______________________________________________

**Prevention Measures:** _______________________________________________

**End Time:** ________________  
**Time to Resolution:** ________________

**Sign-off:** ________________

---

## Quick Reference Commands

```bash
# Full status in one command
mel doctor && mel status

# Watch live logs
journalctl -u mel -f

# Quick health check
curl -s http://localhost:8080/healthz && curl -s http://localhost:8080/readyz

# Restart MEL (if needed)
systemctl restart mel

# Check all transports
curl -s http://localhost:8080/api/v1/status | jq '.transports[] | {name, state, enabled}'
```
