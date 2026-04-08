# MEL — MeshEdgeLayer

**An honest operator console for mesh incidents, evidence, and trusted control.**

![MEL Hero](assets/mel_hero_new_1774058412698.png)

MEL helps operators answer three questions without guessing:

1. **What is happening now?** (live vs stale vs historical)
2. **What do we actually know?** (evidence, not vibes)
3. **What was done, by whom, and with what result?** (audit trail)

If you care about local-first operations, degraded environments, and not lying to yourself with pretty dashboards, this repo is for you.

## Why MEL is different

- **Truth-first semantics:** MEL explicitly marks live, stale, historical, partial, degraded, and unknown states.
- **Evidence-backed workflow:** incidents, proofpacks, and action history are tied to persisted records.
- **No capability theatre:** unsupported paths stay labeled unsupported.
- **Local-first by default:** no mandatory cloud control plane for core operation.

## What MEL is / is not

### MEL is
- A local-first incident-intelligence and trusted-control system.
- An operator workflow layer over deterministic ingest and audit evidence.
- A practical platform for teams that need trustworthy comms operations under imperfect conditions.

### MEL is not
- A mesh routing stack.
- Proof of RF delivery or propagation by default.
- A generic dashboard skin, messenger clone, or AI wrapper.

## 10-minute quick start

```bash
make build
./bin/mel init --config .tmp/mel.json --force
chmod 600 .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json
```

Open <http://127.0.0.1:8080>.

Want a deterministic no-radio demo?

```bash
make demo-seed
./bin/mel serve --config demo_sandbox/mel.demo.json
```

## Start here (one path, not ten)

- **Run MEL fast:** [`docs/getting-started/QUICKSTART.md`](docs/getting-started/QUICKSTART.md)
- **Evaluate MEL honestly:** [`docs/ops/evaluate-in-10-minutes.md`](docs/ops/evaluate-in-10-minutes.md)
- **Contribute your first PR:** [`CONTRIBUTING.md`](CONTRIBUTING.md)
- **Understand project boundaries:** [`docs/repo-os/README.md`](docs/repo-os/README.md)
- **Browse docs map:** [`docs/README.md`](docs/README.md)

## Transport support (current truth)

| Surface | State | Contract |
| --- | --- | --- |
| Direct ingest (serial/TCP) | Supported | Claims must come from persisted, timestamped ingest evidence. |
| MQTT ingest | Supported | Disconnects and partial ingest must remain explicit. |
| BLE ingest | Unsupported | Must be labeled unsupported. |
| HTTP ingest | Unsupported | Must be labeled unsupported. |
| RF routing/transmit by MEL | Not implemented | MEL does not claim mesh-stack routing execution. |

## Contributing

We welcome serious contributions, including docs, UX clarity, diagnostics, tests, and transport hardening.

Start with:
- [`CONTRIBUTING.md`](CONTRIBUTING.md)
- [Issue templates](.github/ISSUE_TEMPLATE/)
- [`docs/contributor/FIRST_PR_PATHS.md`](docs/contributor/FIRST_PR_PATHS.md)

## Verification signals

| Command | What it proves |
| --- | --- |
| `make test` | Go tests only (`go test ./...`). |
| `make lint` | `go vet` + frontend ESLint + site ESLint. |
| `make build` | Frontend build/embed + Go binaries. |
| `make smoke` | End-to-end smoke against built `./bin/mel`. |
| `make verify-stack` | `lint + test + build + smoke` as one stack signal. |
| `make premerge-verify` | Release-grade verification chain. |

## License

GNU GPL v3.0. See [`LICENSE`](LICENSE).
