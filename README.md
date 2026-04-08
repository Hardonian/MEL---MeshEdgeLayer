# MEL — MeshEdgeLayer

**Local-first operator system for mesh incidents, evidence, and trusted control.**

![MEL Hero](assets/mel_hero_new_1774058412698.png)

MEL exists for teams who are tired of dashboards that guess, smooth over gaps, and call it “real-time.”
If the system does not know, it should say so.

## What MEL is (in 20 seconds)

MEL is an evidence-first operations layer for mixed mesh environments.

- Ingests transport evidence (serial/TCP/MQTT) and persists it with timestamps.
- Separates **live / stale / historical / partial / degraded / unknown** states.
- Tracks incident and action history with explicit lifecycle boundaries.
- Runs local-first with no mandatory cloud control plane.

## What MEL is not

- Not a mesh routing stack.
- Not proof of RF propagation unless your evidence proves it.
- Not fake support for unsupported paths.
- Not “AI truth.” Inference is assistive, not canonical.

## Why people star MEL

- **Truthful operator surfaces:** no dashboard astrology, no hidden certainty.
- **Bounded claims:** support matrix and limitations are explicit.
- **Audit-friendly operations:** incidents, actions, and evidence stay attributable.
- **Serious verification path:** reproducible build/test/smoke/release gates.

## Transport support matrix (current)

| Surface | State | Contract |
| --- | --- | --- |
| Direct ingest (serial/TCP) | Supported | Claim only persisted + timestamped evidence. |
| MQTT ingest | Supported | Must surface disconnects and partial ingest explicitly. |
| BLE ingest | Unsupported | Must be labeled unsupported. |
| HTTP ingest | Unsupported | Must be labeled unsupported. |
| Radio transmit/routing by MEL | Not implemented as mesh-stack feature | Do not imply MEL performs RF routing execution. |

## Quick start

```bash
make build
./bin/mel init --config .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json
```

Open <http://127.0.0.1:8080>.

### No-radio eval mode (deterministic fixtures)

```bash
make demo-seed
./bin/mel serve --config demo_sandbox/mel.demo.json
```

Useful paths:
- [Quickstart](docs/getting-started/QUICKSTART.md)
- [Evaluate in 10 minutes](docs/ops/evaluate-in-10-minutes.md)
- [Scenario library](docs/community/SCENARIO_LIBRARY.md)

## Who should use MEL

| You are… | Start here |
| --- | --- |
| Operator / evaluator | [Getting started](docs/getting-started/README.md) |
| Field tester | [Field testing guide](docs/community/FIELD_TESTING.md) |
| Contributor | [Community START_HERE](docs/community/START_HERE.md) · [CONTRIBUTING.md](CONTRIBUTING.md) |
| Sponsor / design partner | [Claims vs reality](docs/community/claims-vs-reality.md) · [Current status](docs/project/CURRENT_STATUS.md) |

## Contributing (safe first moves)

- [First PR paths](docs/contributor/FIRST_PR_PATHS.md)
- [Contributor map](docs/contributor-map.md)
- [Issue labels and routing](docs/community/issue-routing-labels.md)
- [Repo operating system](docs/repo-os/README.md)

## Documentation map

- [Docs hub](docs/README.md)
- [FAQ](docs/FAQ.md)
- [Product docs](docs/product/README.md)
- [Operations runbook](docs/ops/OPERATIONS_RUNBOOK.md)
- [Release criteria](docs/release/RELEASE_CRITERIA.md)
- [Known limitations](docs/ops/limitations.md)

Optional public orientation site: [`site/README.md`](site/README.md).

## Verification entry points

| Command | What it proves |
| --- | --- |
| `make test` | Go tests (`go test ./...`). |
| `make lint` | `go vet` + frontend/site ESLint. |
| `make build` | Frontend build + embedded assets + Go binaries. |
| `make smoke` | End-to-end smoke tests against built CLI. |
| `make verify-stack` / `make check` | `lint + test + build + smoke` stack signal. |
| `make premerge-verify` | Full release-reality chain (maintainer gate). |

Then apply repo-os verification/release gates:
- [`docs/repo-os/verification-matrix.md`](docs/repo-os/verification-matrix.md)
- [`docs/repo-os/release-readiness.md`](docs/repo-os/release-readiness.md)

## License

MEL is GPL-3.0. See [`LICENSE`](LICENSE).

## Security / support

- [SECURITY.md](SECURITY.md)
- [SUPPORT.md](SUPPORT.md)
- [AGENTS.md](AGENTS.md) (project contract)
