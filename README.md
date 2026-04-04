# MEL — MeshEdgeLayer

**Incident intelligence and trusted control for mesh operations.**

![MEL Hero](assets/mel_hero_new_1774058412698.png)

MEL is a local-first operational system for mesh environments. It ingests evidence, preserves action and incident history, and exposes operator-facing truth with explicit degraded semantics.

## What MEL is

- A mesh observability and control workflow system.
- An evidence-first platform for incidents, actions, audits, and proofpacks.
- A local runtime that keeps operator trust boundaries explicit.

## What MEL is not

- Not a mesh routing stack.
- Not proof of RF propagation or routing success unless evidence exists.
- Not authorized to imply unsupported transport paths.

## Truth contract (short)

MEL does not claim more than evidence supports.

- **Live** means recent persisted ingest evidence exists.
- **Stale** means ingest evidence is old.
- **Historical** means prior records, not current runtime proof.
- **Imported/Offline** means external context, not live fleet proof by default.
- **Partial/Degraded** means known gaps are explicit and machine-visible.

See the canonical terminology guide: [`docs/repo-os/terminology.md`](docs/repo-os/terminology.md).

## Transport support matrix (current)

| Surface | State | Contract |
| --- | --- | --- |
| Direct ingest (serial/TCP) | Supported | Claim only persisted and timestamped ingest evidence. |
| MQTT ingest | Supported | Surface disconnects and partial ingest explicitly. |
| BLE ingest | Unsupported | Label unsupported; no implied partial support. |
| HTTP ingest | Unsupported | Label unsupported; no optimistic wording. |
| Radio transmit/routing by MEL | Not implemented as a mesh-stack feature | Do not imply MEL performs RF routing/propagation execution. |

## Quick start

```bash
make build
./bin/mel init --config .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json
```

Open <http://127.0.0.1:8080>.

For a guided evaluation path: [`docs/ops/evaluate-in-10-minutes.md`](docs/ops/evaluate-in-10-minutes.md).

## Documentation map

- Getting started: [`docs/getting-started/README.md`](docs/getting-started/README.md)
- Product system: [`docs/product/README.md`](docs/product/README.md)
- Release/support system: [`docs/release/RELEASE_CRITERIA.md`](docs/release/RELEASE_CRITERIA.md)
- Communications hub blueprint: [`docs/architecture/communications-hub-blueprint.md`](docs/architecture/communications-hub-blueprint.md)
- Operations: [`docs/ops/OPERATIONS_RUNBOOK.md`](docs/ops/OPERATIONS_RUNBOOK.md)
- Runbooks: [`docs/runbooks/README.md`](docs/runbooks/README.md) (includes [launch & demo](docs/runbooks/launch-and-demo.md))
- Post-launch intake: [`docs/community/post-launch-playbook.md`](docs/community/post-launch-playbook.md)
- API reference: [`docs/ops/api-reference.md`](docs/ops/api-reference.md)
- Repo operating system: [`docs/repo-os/README.md`](docs/repo-os/README.md)
- Repo-local model spec: [`docs/repo-os/model-spec.md`](docs/repo-os/model-spec.md)
- Repo-local skills/checklists: [`docs/repo-os/skills/README.md`](docs/repo-os/skills/README.md)
- Privacy platform policy: [`docs/privacy/platform-policy.md`](docs/privacy/platform-policy.md)
- Known limitations: [`docs/ops/limitations.md`](docs/ops/limitations.md)
- Internal/private strategy notes: [`docs/internal/private/README.md`](docs/internal/private/README.md)

## Verification entry points

Use these before making release-strength claims:

```bash
make lint
make frontend-typecheck
make frontend-test
make test
make build
make product-verify
make smoke
```

Notes:
- Frontend verification requires Node `24.x` (`frontend/.nvmrc`, `frontend/package.json`, and guard script enforce this).
- `make smoke` requires `./bin/mel`; build it first with `make build-cli` or `make build`.

Then apply the repo-os gates:

- [`docs/repo-os/verification-matrix.md`](docs/repo-os/verification-matrix.md)
- [`docs/repo-os/release-readiness.md`](docs/repo-os/release-readiness.md)

## Contributing

- Contributor guide: [`CONTRIBUTING.md`](CONTRIBUTING.md)
- Security policy: [`SECURITY.md`](SECURITY.md)
- Repo contract for agents/contributors: [`AGENTS.md`](AGENTS.md)
