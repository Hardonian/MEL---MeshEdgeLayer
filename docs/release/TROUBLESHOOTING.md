# Troubleshooting

## Fast triage path

1. Verify configuration (`mel config validate`).
2. Check transport health (`mel doctor`, `mel status`).
3. Confirm whether data is live, stale, or absent.
4. Review dead letters and incident timeline.
5. Export support bundle before invasive changes.

## Common confusion traps

- Configured transport is not the same as active ingest.
- Advisory recommendation is not an executed action.
- Historical evidence does not prove current runtime state.

## Reference docs

- [Ops troubleshooting](../ops/troubleshooting.md)
- [Transport troubleshooting](../ops/troubleshooting-transports.md)
- [Dead letters runbook](../runbooks/dead-letters.md)
