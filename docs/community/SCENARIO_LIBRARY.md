# Scenario library (demo / replay)

Scenarios are **deterministic fixtures** for UX evaluation, tests, and honest screenshots. They are defined in code and listed by the CLI.

## List scenarios

After `make build-cli`:

```bash
./bin/mel demo scenarios
./bin/mel demo scenarios --json
```

Source of truth: `internal/demo/catalog.go` (IDs, titles, summaries, operator narratives).

## Seed a scenario into a local database

1. Create or reuse a sandbox config:

   ```bash
   ./bin/mel demo init-sandbox --out demo_sandbox/mel.demo.json
   chmod 600 demo_sandbox/mel.demo.json
   ```

2. Seed:

   ```bash
   ./bin/mel demo seed --scenario <scenario-id> --config demo_sandbox/mel.demo.json
   ```

3. Serve the UI:

   ```bash
   ./bin/mel serve --config demo_sandbox/mel.demo.json
   ```

Or use `make demo-seed` from the repo root (see [Makefile](../../Makefile)) for a one-command seed of the default scenario.

## Evidence bundle capture (CLI)

For scripted proof (CI-style):

```bash
./scripts/demo-evidence.sh <scenario-id> [config-path]
```

Requires `./bin/mel` built. Output locations are printed by the script; treat bundles as **synthetic** unless labeled otherwise.

## Contributing new scenarios

1. Add a scenario struct in `internal/demo/catalog.go` with a stable `ID` and honest summary.
2. Extend tests in `internal/demo/demo_test.go` if behavior is non-trivial.
3. Run `make demo-verify`.
4. Document the scenario ID here in a short table (optional row) or rely on `mel demo scenarios` as canonical list — avoid duplicating long narratives in two places.

## Honesty guardrail

Scenarios **simulate** mesh and transport stress for the operator console. They do not demonstrate RF coverage, real broker security, or production scale unless you run separate, explicit tests and say so in your write-up.
