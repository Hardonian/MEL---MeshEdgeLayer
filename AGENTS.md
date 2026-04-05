# MEL Repo Operating System (Canonical Agent + Contributor Contract)

This file is the default operating context for all MEL work.
If any task conflicts with this contract, **narrow the claim or strengthen implementation + verification**.

## 1) MEL Identity (Canonical)

### What MEL is
MEL is a **truthful operator OS and communications OS** for mixed local communications under degraded conditions. It is an **incident-intelligence and trusted-control operating system** that preserves evidence, action history, and operational memory.

MEL is designed to be:
- evidence-first;
- local-first by default;
- privacy-first, open, and self-hosted friendly;
- explicit about degraded and unknown states.

### What MEL is not
- Not a generic dashboard skin.
- Not a generic messenger.
- Not an AI wrapper.
- Not a protocol zoo.
- Not a cloud-first SaaS dependency trap.
- Not a mesh routing stack.
- Not proof of RF coverage/propagation/routing success unless evidence exists.
- Not authorized to imply transport/runtime support beyond code + verification evidence.

## 2) Mission

Build MEL as a **truth-preserving observability + trusted-control system** where:
- observability distinguishes live, stale, partial, historical, imported/offline, degraded, and unknown evidence;
- controls are governed by explicit lifecycle/approval/audit states;
- incident workflows compound institutional knowledge (tests, runbooks, heuristics, evidence schemas, action outcomes);
- operator-facing claims remain bounded by deterministic evidence.

## 3) Non-Negotiable Invariants

1. No fake transport support claims.
2. No fake live-state claims.
3. No collapsing stale/imported/partial/degraded/unknown into “healthy/current/live”.
4. No unsupported protocol/path implication (BLE/HTTP ingest, radio send path, etc.) without verified implementation.
5. No unsafe control-path claims (submission ≠ approval ≠ dispatch ≠ execution ≠ audit).
6. No hidden approval bypasses or silent escalation paths.
7. No claims stronger than evidence.
8. No silent auth/trust-boundary broadening.
9. No tenant/operator/action attribution ambiguity on control paths.
10. No degraded state without explicit machine-visible signaling.
11. No local inference output as canonical system truth.
12. No fake scientific precision or causal storytelling without evidence.
13. No privacy-first wording without enforceable implementation consequences.

## 4) Deterministic Truth Hierarchy

Canonical truth ordering:
1. Typed deterministic runtime evidence (ingest records, state transitions, audit events).
2. Deterministic calculators and bounded heuristics with explicit inputs.
3. Inference/estimation layers labeled as assistive, non-canonical.
4. Narrative explanation text.

If levels conflict, higher level wins. UI/API/docs must mirror this hierarchy.

## 5) Transport & Mixed-Channel Truth Matrix

| Surface | State | Truth Contract |
|---|---|---|
| Direct ingest (serial/TCP) | Supported | Claim only what ingest workers persisted and timestamped. |
| MQTT ingest | Supported | Must surface broker/runtime disconnects and partial ingest explicitly. |
| BLE ingest | Unsupported | Must be labeled unsupported; no implied partial support. |
| HTTP ingest | Unsupported | Must be labeled unsupported; no UI/API optimism. |
| Radio transmit/routing execution by MEL | Not implemented as a mesh stack feature | Do not imply MEL itself performs RF routing/propagation execution. |

Mixed-channel truth must preserve:
- channel: LoRa / Bluetooth / Wi-Fi / mixed / unknown;
- path mode: direct / relayed / bridged / local-only / backhaul-assisted;
- support posture: supported / unsupported / sparse / degraded / unknown;
- processing flags: compression applied, assistive inference applied, privacy level.

Evidence required before claiming success:
- transport connected + ingest loop active;
- packet/evidence persisted;
- timestamps/source context available;
- failure/dead-letter visibility preserved.

## 6) Precision-Layering Canon (Scientific Observation)

When changing semantics, telemetry, diagnostics, health, or recommendations, reason and report through explicit layers:

1. **Semantics layer**: mesh, node, link, path, backhaul, nearby control, signal, evidence, proofpack, degraded, incident, action, health.
2. **Telemetry layer**: message/path/node/delivery/queue metrics, retries, latency, relay events, failures.
3. **Frequency/radio context layer**: band/frequency/signal posture, SNR/RSSI-like context (if available), retry bursts, channel occupancy, margin estimates.
4. **Physical/environment layer**: indoor/outdoor, mobility, terrain/building density, weather/temperature/humidity context, antenna/power/deployment context (if available).
5. **Sensor/context layer**: location, motion, battery/power, thermal, enclosure and environment sensors.
6. **Spatial/navigation layer**: site/zone/floor/area, weak-zone memory, node drift, gateway proximity, route geography.
7. **Algorithmic/calculator layer**: evidence sufficiency, delivery confidence, link/path stability, topology drift, relay dependence, proofpack completeness, snapshot completeness, action evidence strength.
8. **Mixed-channel truth layer**: channel + path + support + privacy/compression/assistive-processing posture.

Rules:
- Prefer scientific observation over narrative.
- Distinguish observed vs inferred vs estimated vs unknown.
- Treat physical/environment context as enrichment, not automatic root cause.
- Use deterministic calculators before ML/LLM escalation.
- Keep evidence sufficiency, proofpack completeness, snapshot completeness, and mixed-channel delivery confidence first-class.

## 7) Build-vs-Borrow Discipline

MEL should **build** durable differentiators:
- mixed-network truth model;
- incident intelligence;
- proofpacks and evidence packs;
- action-outcome memory and per-action snapshots;
- operator workflows;
- privacy/trust policy enforcement layers;
- local operational intelligence orchestration;
- evidence/audit trust layers.

MEL should **borrow OSS** for commodity primitives unless truth/privacy/cost constraints fail:
- crypto primitives;
- brokers/event bus;
- TURN/STUN;
- object storage;
- codecs/media compression;
- STT;
- local model serving/runtime;
- optional interop layers.

## 8) Privacy-First / Open / Self-Hosted Rules

- No mandatory cloud dependency for base MEL viability.
- No hidden telemetry; telemetry defaults must be explicit and privacy-preserving.
- Local data ownership with retention/export/delete semantics.
- Explicit key/material boundaries and secret handling.
- Optional integrations must be clearly marked optional.
- Open-source and self-hosted bias by default.
- Low recurring cost is a design constraint, not a marketing footnote.

## 9) Local Inference Policy (Assistive, Non-Canonical)

- Local inference is optional assistive compute, never canonical truth.
- Deterministic typed truth remains primary.
- Route inference by task and resource profile (foreground/background, single-thread/multi-thread).
- Ollama is the default easy path.
- llama.cpp is the advanced path.
- TurboQuant-compatible compression paths may be used experimentally with explicit caveats.
- CPU fallback must remain supported.
- Base MEL truth/control flows must remain viable even if assistive runtime is absent or failing.

## 10) Trust Boundaries

- **Operator intent vs execution**: submission, approval, dispatch, execution result, and audit record are separate states.
- **Historical vs live**: history explains context but does not prove current runtime.
- **Imported/offline vs local live observations**: imported data is context, not direct live truth.
- **Local truth vs mesh inference**: direct observations must be distinguishable from inferred conclusions.
- **AuthN/AuthZ boundaries**: capabilities and actor scopes must be explicit; avoid silent fail-open behavior.

## 11) Terminology Policy

Use simple operator-facing terms where precision is preserved: mesh, LoRa, frequency, Wi-Fi, Bluetooth, node, link, signal, action, evidence, proofpack, health, incident, degraded.

Use deeper technical terms only when needed for correctness. When both are needed, lead with the simple term and then clarify precisely.

## 12) Moat Priorities & Pressure Test

Prefer work that compounds:
1. Operational memory (incident/action/evidence/outcome history).
2. Workflow lock-in (incident-linked controls, evidence packs, review surfaces).
3. Explainable intelligence (diagnosis/recommendation rationale bounded by evidence).
4. Audit-grade evidence quality (replayable, attributable, bounded claims).
5. Local-first/private/self-hosted affordability + resilience.

For major features/refactors, answer:
1. What can MEL infer/recommend/prove/remember from real history that a copied dashboard cannot?
2. Does this compound with usage?
3. Does this convert incidents/actions into reusable tests/runbooks/rules?
4. Does this deepen workflow centrality/switching cost?
5. Could a competitor copy this from UI alone?
6. Does this improve trusted control, anomaly intelligence, evidence quality, or operator trust?

## 13) Work Classification (for PRs and planning)

- **Maintenance**: correctness, reliability, drift reduction; no new moat surface.
- **Leverage**: improves reusable workflow speed/safety with measurable operator-trust gain.
- **Moat**: increases compounding data/decision/workflow advantage competitors cannot copy from UI alone.

Use `docs/repo-os/change-classification.md` for required PR labeling and evidence.

## 14) Verification & Release Bar

Every meaningful change must map to `docs/repo-os/verification-matrix.md` and pass `docs/repo-os/release-readiness.md`.

Minimum bar:
- capability claims match implementation;
- degraded/unknown states explicitly represented;
- privacy/telemetry defaults validated;
- evidence sufficiency and proofpack/snapshot completeness semantics validated;
- canonical truth not coupled to local inference success;
- operator-facing wording bounded by evidence;
- verification evidence attached;
- caveats documented with concrete boundaries.

## 15) Execution Playbook for Agents

1. Read this file + `docs/repo-os/README.md` before broad changes.
2. Scope claims to implemented behavior; if uncertain, downgrade claim language.
3. Prefer typed contracts over prose heuristics.
4. Add/adjust tests or verification artifacts for changed truth/control/privacy paths.
5. Update relevant repo-os checklist if a new failure mode appears.
6. In final summary: include residual risk and explicit caveats.

## 16) Existing Environment Notes

- Use `make build`, `make lint`, `make test`, and `make smoke` as standard verification entry points.
- In this environment, some pre-existing failures may exist; report them honestly, do not mask with selective claims.
- Use `sqlite3` CLI for deterministic DB checks/migrations when needed.

## Cursor Cloud specific instructions

### Toolchain versions

- **Go 1.24+** is required (`go.mod` specifies `go 1.24`). The VM update script installs it to `/usr/local/go`. The Makefile automatically prefers `/usr/local/go/bin/go` when present.
- **Node.js 24.x** is required (enforced by `.nvmrc` and guard scripts in `scripts/require-node24.sh` and `frontend/scripts/require-node24.mjs`). The update script installs it via nvm. You must source nvm before running any Node/frontend commands: `. "$HOME/.nvm/nvm.sh" && nvm use 24`.

### Running services

- **Backend (`mel serve`)**: After `make build`, run `./bin/mel serve --config configs/mel.generated.json`. Serves API + embedded web UI on port 8080. Config file must have `chmod 600` permissions.
- **Frontend dev server**: `cd frontend && npm run dev` starts Vite on port 3000, proxying `/api` to `127.0.0.1:8080`. Requires the backend to be running.
- **Config initialization**: `./bin/mel init --config .tmp/mel.json --force` creates a fresh config. Then `chmod 600` the generated config before `mel serve`.
- **Demo data**: `./bin/mel demo seed --scenario healthy-private-mesh --config configs/mel.generated.json --force` seeds realistic mesh data for UI testing.

### Verification commands (see README.md for full list)

| Command | What it does |
|---|---|
| `make lint` | Go vet + frontend ESLint |
| `make test` | Go tests (`go test ./...`) |
| `make frontend-test-fast` | Frontend Vitest (skips clean install) |
| `make build` | Frontend build + Go binary compilation |
| `make smoke` | End-to-end smoke tests (requires built binary) |

### Gotchas

- The system Go (`/usr/bin/go`) is typically 1.22 which is too old; always ensure `/usr/local/go/bin/go` is on PATH or let the Makefile resolve it.
- `mel doctor` exit code 1 is expected on fresh installs with no transports; warnings about stale components are normal without active devices.
- `npm ci` in `frontend/` runs a `preinstall` guard that rejects non-24.x Node versions. If nvm is not sourced, frontend targets will fail with a clear error.
- Optional services (NATS, MinIO, Coturn, Ollama) are defined in `examples/deployment/docker-compose.yml` under profiles. They are not required for core development.
