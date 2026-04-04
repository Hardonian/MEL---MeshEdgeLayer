# Contributing to MEL

Welcome to the MeshEdgeLayer (MEL) contribution guide. This project is built to reduce entropy and increase structural coherence in mesh network operations.

## 🏁 Ground Rules for Contributors

1. **Keep MEL Honest**: No fake transports, no fake mesh data, and no dead routes.
2. **Narrow Public Claims**: If a feature is not yet fully implemented or verified, it must be explicitly labeled as such in both code and documentation.
3. **Preserve Compatibility**: Stock Meshtastic compatibility boundaries are sacred.
4. **Prefer Stdlib-Only Go**: Avoid adding external dependencies unless they are already vendored.
5. **No Undefined Access**: All errors must be handled and reported explicitly. Silencing an error is a violation of the "Truth-First" model.

---

## 🏗️ Architecture & Truth Boundaries

Before you begin, understand where MEL starts and stops:

- **Ingest Layers**: Currently serial direct-node, TCP direct-node, and MQTT are supported.
- **Explicit Limitations**: BLE, HTTP ingest, and radio transmission (send/publish) are **not** currently implemented.
- **Metrics**: A `/metrics` endpoint exists, but MEL does not yet ship a dedicated metrics server or exporter.
- **Control Plane**: Automated remediations must be guarded. No autonomous action without evidence provenance.

Review the [Known Limitations](docs/ops/limitations.md) for more details.

---

## 🛠️ Local Development & Build

### 1. Build and Verify

MEL uses a straightforward `Makefile` to maintain consistency:

```bash
# Full lint, test, and build pass
make verify

# Build only (outputs to bin/)
make build

# Run unit tests
make test

# Run the local smoke test suite
./scripts/smoke.sh
```

### 2. Frontend Development

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

### 3. Verification Standards

Every Pull Request must:

- Pass `make lint`.
- Pass `make frontend-typecheck`.
- Pass `make frontend-test`.
- Pass `make test`.
- Build `./bin/mel` before smoke (`make build-cli` or `make build`), then pass `make smoke`.
- Include verification steps in the description.

---

## 🔌 Safe Areas for Expansion

- **Policy & Privacy Logic**: Refine redaction or trust boundary enforcement.
- **Documentation**: New runbooks or operator workflow improvements.
- **Transport Hardening**: Improve reconnection persistence or error reporting.
- **Protobuf Decoding**: Support new Meshtastic payloads (include fixtures/tests).

---

## 🧭 MEL Repo-OS Discipline

All non-trivial changes must run through the repo operating system in `docs/repo-os/README.md`.

Minimum expectation per PR:

- Classify work as Maintenance / Leverage / Moat (`docs/repo-os/change-classification.md`).
- Run applicable truth/governance/security audits from `docs/repo-os/`.
- Meet verification obligations from `docs/repo-os/verification-matrix.md`.
- Pass release reality gate in `docs/repo-os/release-readiness.md` for capability-affecting changes.

## ⚓ Pull Request Checklist

When submitting a PR, include:

- **Design Intent**: Why is this change necessary?
- **Operator Impact**: What does a user see or experience differently?
- **Verification Evidence**: Logs, test results, or manual proof.
- **Residual Risk**: What is still unknown or partial about this implementation?

**We optimize for a clean, deterministic repository.** If a change increases noise, it will be rejected in favor of a simpler, more coherent solution.

MEL is licensed under the **Apache-2.0 License**.
© 2026 Hardonian / MeshEdgeLayer Contributors.
