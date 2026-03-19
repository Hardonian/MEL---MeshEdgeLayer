# Diagnostics and runtime truth

## Doctor v2

`mel doctor` is the authoritative offline diagnostic command.

It checks:

- config validation,
- operator config file mode (`0600` required for production configs),
- SQLite write/read behavior,
- direct transport reachability checks where safe,
- persisted historical evidence by transport,
- actionable operator guidance.

## Shared transport states

These states are the contract used by doctor, status, UI, and `/api/v1/status`:

- `disabled`
- `configured_not_attempted`
- `attempting`
- `configured_offline`
- `connected_no_ingest`
- `ingesting`
- `historical_only`
- `error`

## Runtime versus persisted truth

MEL distinguishes three scopes:

- `config+persisted`: configuration and database evidence only
- `persisted_only`: historical evidence without live runtime proof
- `runtime+persisted`: active runtime observations plus database evidence

## Metrics

`/metrics` and `/api/v1/metrics` return JSON with:

- total messages,
- last ingest time,
- per-transport counters,
- recent ingest rate over the last five minutes.

## Replay and filtering

Use replay to inspect only what MEL actually stored:

```bash
./bin/mel replay --config /etc/mel/mel.json --limit 50
./bin/mel replay --config /etc/mel/mel.json --node 12345 --type text --limit 20
```
