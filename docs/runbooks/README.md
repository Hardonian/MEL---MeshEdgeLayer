# MEL Runbooks

Step-by-step procedures for incidents, degraded ingest, and recovery.

## Connectivity and ingest

- [Transport down / reconnect churn](common-connectivity.md)
- [Dead letter growth](dead-letters.md)
- [Partial fleet and scope gaps](partial-fleet-and-scope.md)

## Incident response and evidence

- [Incident investigation](incident-investigation.md)
- [Remote evidence import](remote-evidence-import.md)
- [Proofpack export](proofpack-export.md)
- [Support bundle interpretation](support-bundle-interpretation.md)

## Recovery and maintenance

- [System recovery](recovery.md)
- [Database maintenance](database-maintenance.md)

## Escalation path

1. Run `mel doctor --config <path>`.
2. Check [Troubleshooting](../ops/troubleshooting.md).
3. Follow the matching runbook above.
4. Export a support bundle if deeper analysis is needed.
