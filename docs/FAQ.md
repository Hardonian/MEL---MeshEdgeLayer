# MEL — Frequently asked questions

Answers below are bounded by what the repository implements today. When in doubt, prefer [`docs/ops/limitations.md`](ops/limitations.md), the [transport matrix](ops/transport-matrix.md), and [canonical terminology](repo-os/terminology.md).

## What is MEL?

MEL (MeshEdgeLayer) is a **local-first** operator system for mesh and edge operations: it ingests evidence, ties incidents and control actions to attributable records, and surfaces **live, stale, historical, imported, partial, degraded, and unknown** states explicitly. It is not a generic SaaS dashboard and not a substitute for a mesh routing stack.

## Who is MEL for?

Operators and builders who need **audit-friendly** visibility and workflows: club or event nets, lab and homelab experimentation, field evaluation, and contributors extending transports or the UI. See [Getting started](getting-started/README.md) and [Community START_HERE](community/START_HERE.md).

## Does MEL route RF or prove coverage?

**No.** MEL does not implement mesh RF routing or propagation as a product feature. Any map or topology view reflects **persisted ingest evidence and interpretation**, not proof that packets traversed a particular path unless your evidence supports that claim.

## Which transports are supported?

| Path | Posture |
| --- | --- |
| Direct ingest (serial / TCP) | Supported — claims bounded to persisted, timestamped ingest |
| MQTT ingest | Supported — disconnects and partial ingest must stay visible |
| BLE ingest | **Unsupported** — do not imply partial support |
| HTTP ingest | **Unsupported** — do not imply partial support |
| Radio transmit / routing “by MEL” | **Not implemented** as a mesh-stack feature |

Same matrix appears in the root [README](../README.md) and [product honesty](product/HONESTY_AND_BOUNDARIES.md).

## What do “live” and “stale” mean?

They describe **evidence freshness in the database**, not vibes. **Live** means recent persisted ingest evidence exists; **stale** means evidence is too old for confident runtime inference. See [terminology](repo-os/terminology.md).

## Is local inference (LLM) canonical truth?

**No.** Any assistive inference is **non-canonical**. Deterministic records, calculators with explicit inputs, and audit events outrank narrative or model output. See [AGENTS.md](../AGENTS.md) and [architecture — intelligence layer](architecture/intelligence-layer.md).

## How do I try MEL without radios?

Use fixture-backed demo data: `make demo-seed` then `./bin/mel serve --config demo_sandbox/mel.demo.json`. That is **simulation for UI and regression**, not live fleet proof. See [QUICKSTART](getting-started/QUICKSTART.md) and [Scenario library](community/SCENARIO_LIBRARY.md).

## What should I run before sending a PR?

At minimum: `make lint`, `make test`, `make build`, `make smoke` (after `./bin/mel` exists). For release-shaped confidence, use `make premerge-verify`. See [CONTRIBUTING.md](../CONTRIBUTING.md) and [verification matrix](repo-os/verification-matrix.md).

## Where is the public “front door” vs the embedded console?

- **Repository + `docs/`**: canonical depth, runbooks, and truth contracts.
- **`site/`**: lightweight Next.js orientation (quick start, help, contribute, trust). It complements the CLI/TUI and embedded UI; it does not replace them.
- **Embedded UI**: ships with `./bin/mel serve` after `make build`.

## License?

MEL is under **GPL-3.0** — see [`LICENSE`](../LICENSE). Third-party dependencies have their own licenses (npm, Go modules).
