# Direct-node quickstart

1. Choose `serial` or `tcp`.
2. Copy a config example and `chmod 600` it.
3. Run `./bin/mel config validate --config <path>`.
4. Run `./bin/mel doctor --config <path>`.
5. Start `./bin/mel serve --config <path>`.
6. Visit the UI or `/api/v1/status` and confirm the transport becomes `connected_no_ingest` or `ingesting`.
7. Use `mel replay` or `/api/v1/messages` to confirm stored evidence.
