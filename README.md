# MEL — MeshEdgeLayer

**Operator OS for mesh incidents, evidence, and trusted control — local-first, explicit under uncertainty.**

![MEL Hero](assets/mel_hero_new_1774058412698.png)

MEL is for teams running real-world mixed-channel operations who need truth, not dashboard optimism.
If MEL does not have evidence, it must say **unknown**.

## What MEL is

- Evidence-first ingest + operations system for mixed mesh environments.
- Persists timestamped runtime evidence and preserves action/audit history.
- Keeps live, stale, historical, partial, degraded, and unknown states distinct.
- Runs local-first (self-hosted friendly, no mandatory cloud control plane).

## What MEL is not

- Not a mesh routing stack.
- Not proof of RF propagation without evidence.
- Not fake support for unsupported paths.
- Not “AI truth” (local inference is assistive, never canonical truth).

## Why MEL matters

- **Trust under pressure:** no invented certainty.
- **Bounded claims:** support matrix and degraded states are explicit.
- **Audit-grade operations:** incidents, decisions, and outcomes stay attributable.
- **Launch-grade verification chain:** lint/test/build/smoke/release gates are reproducible.

## Transport support matrix (current)

| Surface | State | Contract |
| --- | --- | --- |
| Direct ingest (serial/TCP) | Supported | Claim only persisted + timestamped evidence. |
| MQTT ingest | Supported | Surface disconnects and partial ingest explicitly. |
| BLE ingest | Unsupported | Must be labeled unsupported. |
| HTTP ingest | Unsupported | Must be labeled unsupported. |
| Radio transmit/routing by MEL | Not implemented as mesh-stack feature | Do not imply MEL performs RF routing execution. |

## Try MEL in 10 minutes

```bash
make build
./bin/mel init --config .tmp/mel.json
chmod 600 .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json
```

Open <http://127.0.0.1:8080>.

No-radio deterministic evaluation:

```bash
make demo-seed
./bin/mel serve --config demo_sandbox/mel.demo.json
```

## Public docs funnel

1. **Start here:** [docs/README.md](docs/README.md)
2. **Try MEL:** [docs/getting-started/QUICKSTART.md](docs/getting-started/QUICKSTART.md), [docs/ops/evaluate-in-10-minutes.md](docs/ops/evaluate-in-10-minutes.md)
3. **Understand boundaries:** [docs/ops/support-matrix.md](docs/ops/support-matrix.md), [docs/ops/limitations.md](docs/ops/limitations.md), [docs/community/claims-vs-reality.md](docs/community/claims-vs-reality.md)
4. **Contribute safely:** [CONTRIBUTING.md](CONTRIBUTING.md), [docs/contributor/FIRST_PR_PATHS.md](docs/contributor/FIRST_PR_PATHS.md)
5. **Operate for real:** [docs/ops/OPERATIONS_RUNBOOK.md](docs/ops/OPERATIONS_RUNBOOK.md)

## Verification entry points

- `make lint` — Go vet + frontend/site ESLint
- `make test` — Go tests (`go test ./...`)
- `make build` — frontend build + embedded assets + Go binaries
- `make smoke` — end-to-end smoke checks against built CLI
- `make first-proof` — deterministic proof bundle workflow
- `make premerge-verify` — release-reality verification chain

Repo-OS release gates:
- [`docs/repo-os/verification-matrix.md`](docs/repo-os/verification-matrix.md)
- [`docs/repo-os/release-readiness.md`](docs/repo-os/release-readiness.md)

## Contribute and support

- [CONTRIBUTING.md](CONTRIBUTING.md)
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- [SECURITY.md](SECURITY.md)
- [SUPPORT.md](SUPPORT.md)
- [Current project status](docs/project/CURRENT_STATUS.md)
- [FAQ](docs/FAQ.md)

MEL is GPL-3.0 licensed; see [LICENSE](LICENSE).
