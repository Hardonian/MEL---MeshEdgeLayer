# MEL — MeshEdgeLayer

**Truth-preserving incident intelligence and trusted control for mesh operations.**

![MEL Hero](assets/mel_hero_new_1774058412698.png)

MEL is a **local-first** operator system: it ingests mesh evidence, keeps incidents and actions attributable, and surfaces **live vs stale vs historical vs degraded** states explicitly. It is built for people who run real nodes and need an audit-friendly console — not a decorative map toy.

## Why star or try MEL

- **Evidence-first operator console** — incidents, proofpacks, action history, and diagnostics wired to what the database actually holds.
- **Honest transport semantics** — no pretending BLE/HTTP ingest or RF routing exists when it does not (see matrix below).
- **Deterministic demo fixtures** — seed credible UI states without radios: `make demo-seed` (Go-only rebuild of `./bin/mel` from committed web assets) then `./bin/mel serve --config demo_sandbox/mel.demo.json`.
- **Serious OSS engineering** — Go + embedded React UI, `make premerge-verify` release chain, repo-os governance in `docs/repo-os/`.

## Who it is for

| You are… | Start here |
| --- | --- |
| Mesh / Meshtastic operator | [Getting started](docs/getting-started/README.md) · [Adoption guide](docs/community/adoption-guide.md) |
| Club / event / hobby group | [Scenario library](docs/community/SCENARIO_LIBRARY.md) · [Topology cookbooks](topologies/README.md) |
| Contributor (any lane) | [Community START_HERE](docs/community/START_HERE.md) · [Contributor paths](docs/community/CONTRIBUTOR_PATHS.md) · [CONTRIBUTING.md](CONTRIBUTING.md) |
| Field tester | [Field testing](docs/community/FIELD_TESTING.md) |
| Evaluator / sponsor | [Product honesty](docs/product/HONESTY_AND_BOUNDARIES.md) · [Verification](#verification-entry-points) below |

## What MEL is

- A mesh **observability and control workflow** system grounded in persisted ingest evidence.
- An **incident and audit** surface with explicit degraded / partial semantics.
- A **local runtime** you can self-host without a mandatory cloud control plane.

## What MEL is not

- Not a mesh routing stack.
- Not proof of RF propagation or delivery unless your evidence supports it.
- Not authorized to imply unsupported transport paths (see matrix).

## Truth contract (short)

MEL does not claim more than evidence supports.

- **Live** — recent persisted ingest evidence exists.
- **Stale** — ingest evidence is old.
- **Historical** — prior records, not current runtime proof.
- **Imported/Offline** — external context, not live fleet proof by default.
- **Partial/Degraded** — known gaps are explicit and machine-visible.

Canonical wording: [`docs/repo-os/terminology.md`](docs/repo-os/terminology.md).

## Transport support matrix (current)

| Surface | State | Contract |
| --- | --- | --- |
| Direct ingest (serial/TCP) | Supported | Claim only persisted and timestamped ingest evidence. |
| MQTT ingest | Supported | Surface disconnects and partial ingest explicitly. |
| BLE ingest | Unsupported | Label unsupported; no implied partial support. |
| HTTP ingest | Unsupported | Label unsupported; no optimistic wording. |
| Radio transmit/routing by MEL | Not implemented as a mesh-stack feature | Do not imply MEL performs RF routing/propagation execution. |

## Quick start (live or eval)

```bash
make build
./bin/mel init --config .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json
```

Open <http://127.0.0.1:8080>.

**Sandbox (fixture-backed, no radio):**

```bash
make demo-seed
./bin/mel serve --config demo_sandbox/mel.demo.json
```

More detail: [QUICKSTART](docs/getting-started/QUICKSTART.md) · [Evaluate in 10 minutes](docs/ops/evaluate-in-10-minutes.md) · [Launch & demo runbook](docs/runbooks/launch-and-demo.md) · [Screenshot checklist](docs/ops/launch-screenshot-checklist.md).

## How to contribute (by role)

| Role | Doc |
| --- | --- |
| Any | [CONTRIBUTING.md](CONTRIBUTING.md), [AGENTS.md](AGENTS.md), [First PR paths](docs/contributor/FIRST_PR_PATHS.md) |
| Frontend | [Frontend contribution](docs/contributor/FRONTEND_CONTRIBUTION.md) |
| Transport / decoders | [Transport contribution](docs/contributor/TRANSPORT_CONTRIBUTION.md), [Protobuf extension](docs/contributor/protobuf-extension-guide.md) |
| Docs / runbooks | [Docs contribution](docs/contributor/DOCS_AND_RUNBOOK_CONTRIBUTION.md) |
| Architecture map | [ARCHITECTURE_MAP.md](docs/contributor/ARCHITECTURE_MAP.md) |

Issue templates include **field report**, **hardware compatibility**, **scenario**, and **integration** ideas (`.github/ISSUE_TEMPLATE/`). Label taxonomy: [`docs/community/issue-routing-labels.md`](docs/community/issue-routing-labels.md).

## What you can build here

- Operator workflows on top of the **API** ([API reference](docs/ops/api-reference.md)).
- **Fixtures and scenarios** for training and regression (`internal/demo/`, `make demo-verify`).
- **Frontend** panels that deepen incident / transport / diagnostics clarity without weakening truth boundaries.
- **Integrations** that respect trust separation (submission ≠ approval ≠ execution).

## Documentation map

- **Docs hub / IA**: [`docs/README.md`](docs/README.md) · [FAQ](docs/FAQ.md)
- **Public orientation site (optional Next.js)**: [`site/README.md`](site/README.md) — quick start, help, contribute, trust surfaces; canonical depth stays in `docs/` and the embedded UI
- **Community hub**: [`docs/community/README.md`](docs/community/README.md)
- Getting started: [`docs/getting-started/README.md`](docs/getting-started/README.md)
- Product system: [`docs/product/README.md`](docs/product/README.md)
- Release/support: [`docs/release/RELEASE_CRITERIA.md`](docs/release/RELEASE_CRITERIA.md)
- Communications hub blueprint: [`docs/architecture/communications-hub-blueprint.md`](docs/architecture/communications-hub-blueprint.md)
- Operations: [`docs/ops/OPERATIONS_RUNBOOK.md`](docs/ops/OPERATIONS_RUNBOOK.md)
- Runbooks: [`docs/runbooks/README.md`](docs/runbooks/README.md) (includes [launch & demo](docs/runbooks/launch-and-demo.md))
- Showcase captures: [`docs/showcase/README.md`](docs/showcase/README.md)
- Post-launch intake: [`docs/community/post-launch-playbook.md`](docs/community/post-launch-playbook.md)
- API reference: [`docs/ops/api-reference.md`](docs/ops/api-reference.md)
- Repo operating system: [`docs/repo-os/README.md`](docs/repo-os/README.md)
- Repo-local model spec: [`docs/repo-os/model-spec.md`](docs/repo-os/model-spec.md)
- Repo-local skills/checklists: [`docs/repo-os/skills/README.md`](docs/repo-os/skills/README.md)
- Privacy platform policy: [`docs/privacy/platform-policy.md`](docs/privacy/platform-policy.md)
- Known limitations: [`docs/ops/limitations.md`](docs/ops/limitations.md)
- Internal/private strategy notes: [`docs/internal/private/README.md`](docs/internal/private/README.md)

## Fast bootstrap (tooling)

- **Go**: 1.24+ (see `go.mod`).
- **Node**: 24.x for frontend targets (`.nvmrc`, `frontend/.nvmrc`). Run `. ./scripts/dev-env.sh` from repo root when using nvm.
- **Dev container**: [`.devcontainer/devcontainer.json`](.devcontainer/devcontainer.json) for VS Code / Codespaces-style environments (Go + Node + Python + `sqlite3`).

## Verification entry points

**Truthful layers (read before claiming “green”):**

| Command | What it actually proves |
| --- | --- |
| `make test` | **Go tests only** (`go test ./...`). Does not run ESLint, TypeScript, or Vitest. |
| `make lint` | `go vet` + **frontend ESLint** (runs `npm ci` in `frontend/` first). |
| `make frontend-verify` | Lint + typecheck + Vitest in `frontend/` (runs `npm ci` first). |
| `make build` | Frontend production build + copies into `internal/web/assets/` + Go binaries. |
| `make smoke` | End-to-end smoke against `./bin/mel` (build first). |
| **`make verify-stack`** or **`make check`** | **Single stack-shaped signal:** `lint` + `test` + `build` + `smoke` (includes frontend via `lint`/`build`). Not the same as `premerge-verify`. |
| `make premerge-verify` | Release-reality gate: reality check + product verify + full chain + churn guard (see script). |
| `make premerge-verify-fast` | Same chain with `VERIFY_SKIP_CLEAN_INSTALL=1` — not release-grade. |

Other useful targets:

```bash
make product-verify
make demo-verify
```

Notes:

- Direct frontend targets (`make frontend-install`, `make frontend-test`, etc.) fail fast unless your shell is on Node `24.x`. Use `. ./scripts/dev-env.sh` from repo root to activate Node 24 via nvm.
- `make smoke` requires `./bin/mel`; build it first with `make build-cli` or `make build`.
- `make premerge-verify` is the deterministic release-reality gate (clean frontend install + full chain). It runs a churn guard so chained verification keeps exactly one frontend `npm ci`.
- `make premerge-verify-fast` is local-only iteration mode (`VERIFY_SKIP_CLEAN_INSTALL=1`); it skips clean frontend install. Do not use it for release-grade claims.

Then apply the repo-os gates:

- [`docs/repo-os/verification-matrix.md`](docs/repo-os/verification-matrix.md)
- [`docs/repo-os/release-readiness.md`](docs/repo-os/release-readiness.md)

## Contributing

- Contributor guide: [`CONTRIBUTING.md`](CONTRIBUTING.md)
- Security policy: [`SECURITY.md`](SECURITY.md)
- Repo contract for agents/contributors: [`AGENTS.md`](AGENTS.md)
