# Contributing to MEL

Welcome to the MeshEdgeLayer (MEL) contribution guide. This project is built to reduce entropy and increase structural coherence in mesh network operations.

**Role-based entry points:** [docs/community/START_HERE.md](docs/community/START_HERE.md) · [docs/community/CONTRIBUTOR_PATHS.md](docs/community/CONTRIBUTOR_PATHS.md) · [docs/contributor/FIRST_PR_PATHS.md](docs/contributor/FIRST_PR_PATHS.md)

**Community standards:** [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) · [SECURITY.md](SECURITY.md)

## Ground rules for contributors

1. **Keep MEL Honest**: No fake transports, no fake mesh data, and no dead routes.
2. **Narrow Public Claims**: If a feature is not yet fully implemented or verified, it must be explicitly labeled as such in both code and documentation.
3. **Preserve Compatibility**: Stock Meshtastic compatibility boundaries are sacred.
4. **Prefer Stdlib-Only Go**: Avoid adding external dependencies unless they are already vendored.
5. **No Undefined Access**: All errors must be handled and reported explicitly. Silencing an error is a violation of the "Truth-First" model.

---

## Architecture and truth boundaries

Before you begin, understand where MEL starts and stops:

- **Ingest Layers**: Currently serial direct-node, TCP direct-node, and MQTT are supported.
- **Explicit Limitations**: BLE, HTTP ingest, and radio transmission (send/publish) are **not** currently implemented.
- **Metrics**: A `/metrics` endpoint exists, but MEL does not yet ship a dedicated metrics server or exporter.
- **Control Plane**: Automated remediations must be guarded. No autonomous action without evidence provenance.

Review the [Known Limitations](docs/ops/limitations.md) for more details.

---

## Local development and build

### 1. Build and Verify

MEL uses a straightforward `Makefile` to maintain consistency:

```bash
# Full format, lint, test, build, and repo-os product checks
make verify

# Build only (outputs to bin/)
make build

# Run unit tests
make test

# Deterministic frontend install + lint/typecheck/test/build in one chained run
make premerge-verify

# Run the local smoke test suite
./scripts/smoke.sh
```

**Sandbox demo (fixture-backed UI data, no radio):** after `make build-cli`, `make demo-seed` then `./bin/mel serve --config demo_sandbox/mel.demo.json`. See [docs/community/SCENARIO_LIBRARY.md](docs/community/SCENARIO_LIBRARY.md).

**Dev container:** [`.devcontainer/devcontainer.json`](.devcontainer/devcontainer.json) (Go 1.24, Node 24, Python 3.12) for VS Code / Codespaces-style setups.

**Node 24 and Python:** `make lint` and `make product-verify` need Node `24.x` and `python3` (or `python`) on `PATH`. From the repo root, after installing [nvm](https://github.com/nvm-sh/nvm):

```bash
. ./scripts/dev-env.sh   # sources nvm, nvm use, checks Node 24 + Python
make lint
```

### 2. Frontend development

The control plane dashboard lives in `frontend/`.

```bash
cd frontend
nvm use # reads .nvmrc (Node 24.x required)
npm install
npm run dev      # Start dev server (Vite default: http://localhost:5173)
npm run lint     # Check styling
npm run typecheck
npm test         # Run vitest suite
```

Runtime contract:
- Frontend install/lint/typecheck/test/build are guarded by `frontend/scripts/require-node24.mjs`.
- Required Node version: `24.x` (`>=24 <25`, see `frontend/package.json` + `frontend/.nvmrc`).
- If Node is not 24.x, frontend commands fail fast with a runtime-contract error.
- `frontend-install` runs `npm ci` deterministically; in a single `make` invocation (`make frontend-verify build-cli`, `make verify`, or `make premerge-verify`), install runs once and downstream checks/build reuse that exact dependency state.

### 3. Verification standards

Every Pull Request must:

- Pass `make lint`.
- Pass `make frontend-typecheck`.
- Pass `make frontend-test`.
- Pass `make test`.
- Build `./bin/mel` before smoke (`make build-cli` or `make build`), then pass `make smoke`.
- Include verification steps in the description.

---

## Safe areas for expansion

- **Policy & Privacy Logic**: Refine redaction or trust boundary enforcement.
- **Documentation**: New runbooks or operator workflow improvements.
- **Transport Hardening**: Improve reconnection persistence or error reporting.
- **Protobuf Decoding**: Support new Meshtastic payloads (include fixtures/tests).

---

## MEL repo-OS discipline

All non-trivial changes must run through the repo operating system in `docs/repo-os/README.md`.

Minimum expectation per PR:

- Classify work as Maintenance / Leverage / Moat (`docs/repo-os/change-classification.md`).
- Run applicable truth/governance/security audits from `docs/repo-os/`.
- Meet verification obligations from `docs/repo-os/verification-matrix.md`.
- Pass release reality gate in `docs/repo-os/release-readiness.md` for capability-affecting changes.

## Pull request checklist

When submitting a PR, include:

- **Design Intent**: Why is this change necessary?
- **Operator Impact**: What does a user see or experience differently?
- **Verification Evidence**: Logs, test results, or manual proof.
- **Residual Risk**: What is still unknown or partial about this implementation?

**We optimize for a clean, deterministic repository.** If a change increases noise, it will be rejected in favor of a simpler, more coherent solution.

MEL is licensed under the **GNU General Public License v3.0** — see [`LICENSE`](LICENSE) in the repository root.

© 2026 Hardonian / MeshEdgeLayer Contributors.
