# Transport and ingest contribution

## Contract

Read [transport-interface-contract.md](transport-interface-contract.md) and the canonical support matrix in the root [README.md](../../README.md).

**Supported today (ingest)**: direct serial, TCP direct-node, MQTT — only claim what tests and code prove.

**Explicitly unsupported today**: BLE ingest, HTTP ingest. Do not add UI or docs that imply partial support.

**Not a mesh stack**: MEL does not perform RF routing or propagation execution as a product feature; keep language aligned with [AGENTS.md](../../AGENTS.md).

## Where to look in code

- CLI wiring: `cmd/mel/`
- Ingest and transport implementations: under `internal/` (use ripgrep for `mqtt`, `serial`, transport type enums).
- Decoding / protobuf: [protobuf-extension-guide.md](protobuf-extension-guide.md), `internal/` packages dealing with payloads.

## Verification expectations

| Change type | Minimum verification |
| --- | --- |
| Decoder / parsing | Unit tests with fixtures in-repo |
| Reconnect / backoff | Tests + `make smoke` when behavior is user-visible |
| New transport | **Out of scope** unless explicitly approved — would require matrix + tests + docs |

## Docs you must update when behavior changes

- Root `README.md` transport table if support posture changes.
- `docs/ops/limitations.md` if operator-visible limits move.
- [transport truth audit](../repo-os/transport-truth-audit.md) checklist for semantics changes.
