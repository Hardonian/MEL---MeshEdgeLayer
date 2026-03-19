# Evaluate MEL in 10 minutes

1. Build MEL with `make build`.
2. Copy a config example to `.tmp/`, rewrite data paths, and `chmod 600` the copied config.
3. Run `./bin/mel config validate --config .tmp/mel.json`.
4. Run `./bin/mel doctor --config .tmp/mel.json`.
5. Start `./bin/mel serve --debug --config .tmp/mel.json`.
6. Watch transport health move to `connected_no_ingest` or `ingesting`.
7. Query `/api/v1/status`, `/api/v1/messages`, and `/metrics`.
8. If doctor or status show `historical_only`, MEL is telling you that persisted evidence exists without claiming the path is live right now.
