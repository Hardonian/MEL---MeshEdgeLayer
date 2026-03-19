# First 10 minutes

Use these checks on a fresh deployment:

- `./bin/mel config validate --config <path>`
- `./bin/mel doctor --config <path>`
- `./bin/mel status --config <path>`
- `./bin/mel transports list --config <path>`

What to expect:

- unsupported transport types stay marked unsupported,
- `historical_only` means the local database has prior packets for that transport,
- `connected_no_ingest` means the path is connected but no packet has been stored yet,
- `ingesting` means live traffic has been written to SQLite.
