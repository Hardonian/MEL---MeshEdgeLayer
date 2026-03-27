# MEL Operational Runbooks

This index provides step-by-step guidance for common operational scenarios.

---

## 🚦 Connectivity & Ingest

- [**Transport Down / Reconnect Churn**](common-connectivity.md) — Reclaiming live ingest after a link failure.
- [**MQTT Subscription Issues**](mqtt-troubleshoot.md) — Managing subscription filters and broker compliance.
- [**Dead Letter Growth**](dead-letters.md) — Analyzing and resolving ingest processing failures.

## 🛠️ Incident & Investigation

- [**Incident Investigation**](incident-investigation.md) — Following the evidence trail for mesh-wide issues.
- [**Evidence Import**](remote-evidence-import.md) — Bringing offline research data into a MEL instance.
- [**Support Bundle Interpretation**](support-bundle-interpretation.md) — Reading and trusting automated diagnostics.

## 🏥 System Health & Recovery

- [**System Recovery**](recovery.md) — Restoring service after a data directory or disk failure.
- [**Database Maintenance**](database-maintenance.md) — Vacuuming, retention tuning, and pruning.
- [**Upgrade Safety**](upgrade-safety.md) — Verifying schema parity and state preservation before minor/major updates.

---

## Escalation Path

1. Run `mel doctor` to verify the local environment.
2. Review the [Troubleshooting Guide](../ops/troubleshooting.md).
3. Follow the appropriate runbook above.
4. If the issue persists, collect a [Support Bundle](support-bundle-interpretation.md) for further analysis.
