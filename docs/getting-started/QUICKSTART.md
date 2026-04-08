# QUICKSTART

## Day 0 (10–15 minutes)

```bash
make build
./bin/mel init --config .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json
```

Open <http://127.0.0.1:8080> and verify:

- status page loads
- transport state is explicit (not implicitly healthy)
- incident queue is visible even if empty

## Fastest path to first proof (no hardware)

```bash
make first-proof
./bin/mel serve --config demo_sandbox/mel.first-proof.json
```

`make first-proof` creates a sandbox config, seeds deterministic data, and writes an evidence bundle so you can validate MEL workflows without claiming live RF proof.

## Fixture-backed UI (no radio)

```bash
make demo-seed
./bin/mel serve --config demo_sandbox/mel.demo.json
```

(`make demo-seed` rebuilds `./bin/mel` with Go only — no Node/npm required.)

See [Scenario library](../community/SCENARIO_LIBRARY.md) for other scenario IDs (`DEMO_SEED_SCENARIO=…`).

## Next reads

- [First hour guide](./FIRST_HOUR_GUIDE.md)
- [First incident guide](./FIRST_INCIDENT_GUIDE.md)
- [Common misreads](./COMMON_MISREADS.md)
