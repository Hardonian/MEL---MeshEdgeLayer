# Hardware compatibility (community reporting)

MEL ingests mesh traffic through **configured transports** (serial/TCP/MQTT per the [README transport matrix](../../README.md)). The project does **not** ship a certified hardware compatibility matrix; this document defines how **you** can report what worked for you without implying official support.

## What we can truthfully say

- MEL’s supported ingest paths are defined by code + docs, not by a vendor list.
- Your report helps others choose gateways and cables — it is **anecdotal** unless maintainers promote it into bounded release notes.

## Report format (copy into the issue template)

Use [.github/ISSUE_TEMPLATE/hardware_compatibility.md](../../.github/ISSUE_TEMPLATE/hardware_compatibility.md). Minimum fields:

| Field | Notes |
| --- | --- |
| MEL version / commit | `./bin/mel version` |
| Host OS / arch | e.g. Debian arm64 |
| Radio / gateway role | e.g. USB serial to Meshtastic device; MQTT via Docker broker |
| Transport config type | `serial`, `tcp`, `mqtt` (as in your config) |
| Result | ingest observed / degraded / blocked — cite `mel doctor` or UI transport state |
| Caveats | firmware version range if known; no BLE path claims |

## Unsupported paths

- **BLE ingest** and **HTTP ingest** are unsupported; reports about them belong in feature/design discussions, not as “it works” compatibility claims without implementation proof.

## Related docs

- [Field testing](FIELD_TESTING.md)
- [Support matrix](../ops/support-matrix.md)
- [Limitations](../ops/limitations.md)
