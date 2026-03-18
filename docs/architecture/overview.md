# Architecture overview

MEL is a local edge collector for stock Meshtastic deployments.

## System position

- **Input side:** one or more configured ingest transports that MEL can actually open.
- **Core:** normalization, local persistence, privacy/policy evaluation, and audit evidence.
- **Output side:** local CLI commands, local HTTP API, local HTML UI, export bundles, and backup bundles.

## What MEL does not sit in front of

- It is not a firmware routing replacement.
- It is not a required cloud relay.
- It is not a radio control plane.
- It is not an authoritative mesh-state oracle beyond what it has locally observed.

## Runtime entrypoints

- `cmd/mel`: operator CLI, including `serve`.
- `cmd/mel-agent`: minimal daemon-style entrypoint around the same app service.

## Core references

- Runtime flow: `docs/architecture/runtime-flow.md`
- Transport flow: `docs/architecture/transport-flow.md`
- Layer map: `docs/architecture/layers.md`
- Product boundaries: `docs/product/what-mel-is-not.md`
