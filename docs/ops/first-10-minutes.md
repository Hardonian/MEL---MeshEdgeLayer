# First 10 Minutes (Ops Redirect)

Canonical quickstart guidance lives here:
- [`docs/getting-started/first-10-minutes.md`](../getting-started/first-10-minutes.md)

This redirect avoids maintaining two competing quickstart narratives.

## Operator truth reminders

- `/healthz` is liveness only.
- `/readyz` and `/api/v1/readyz` represent readiness semantics.
- Use `/api/v1/status` for full transport evidence.
- Use `mel doctor` for host-level checks.
