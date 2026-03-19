# Transport matrix

This file defines support claims for MEL.

| Transport | Support level | Evidence | Caveats |
| --- | --- | --- | --- |
| `serial` | Supported ingest transport | doctor checks, status truth, transport tests, smoke/CLI verification | Requires local tty ownership and `stty`. |
| `tcp` | Supported ingest transport | doctor checks, status truth, transport tests | Endpoint must expose Meshtastic framing. |
| `mqtt` | Supported ingest transport | status truth, metrics, smoke verification, transport tests | Subscribe-only; no publish/admin path. |
| `serialtcp` | Partial alias of direct TCP reader | status truth, transport tests | Not the primary documented workflow. |
| multi-transport ingest | Supported with caveats | dedupe handling plus doctor warnings | Operator must validate duplicate and ownership behavior. |
| `ble` | Unsupported | UI/CLI/doc truth only | Feature flags do not change this. |
| `http` | Unsupported | UI/CLI/doc truth only | No live ingest path exists. |

## Supported transport definition

A transport is supported only when MEL has:

- real ingest verification,
- doctor support,
- documentation coverage,
- automated tests or smoke evidence.
