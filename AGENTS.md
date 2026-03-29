# MEL Repo Operating System (Canonical Agent + Contributor Contract)

This file is the default operating context for all MEL work.
If any task conflicts with this contract, **narrow the claim or strengthen implementation + verification**.

## 1) Repo Identity

### What MEL is
MEL is an **incident-intelligence and trusted-control operating system** for mesh operations. It ingests evidence, preserves operational history, links incidents to actions, and exposes operator-facing truth with explicit uncertainty/degraded semantics.

### What MEL is not
- Not a generic dashboard skin.
- Not a mesh routing stack.
- Not proof of RF coverage/propagation/routing success unless evidence exists.
- Not authorized to imply transport/runtime support beyond code + verification evidence.

### Why operator-truth + no-theatre matter
MEL trust depends on operators believing that UI/API/CLI claims are bounded by evidence. A polished but overstated system destroys incident response quality and long-term defensibility.

## 2) Mission

Build MEL as a **truth-preserving observability + trusted-control system** where:
- observability distinguishes live, stale, partial, historical, and imported evidence;
- controls are governed by explicit lifecycle/approval/audit states;
- incident workflows compound institutional knowledge (tests, runbooks, heuristics, evidence schemas).

## 3) Hard Invariants (Non-Negotiable)

1. No fake transport support claims.
2. No fake live-state claims.
3. No collapsing stale/imported/partial/degraded into “healthy/current/live”.
4. No unsupported protocol/path implication (BLE/HTTP ingest, radio send path, etc.) without verified implementation.
5. No unsafe control-path claims (submission ≠ approval ≠ execution).
6. No hidden approval bypasses or silent escalation paths.
7. No claims stronger than evidence.
8. No silent auth/trust-boundary broadening.
9. No tenant/operator/action attribution ambiguity on control paths.
10. No degraded state without explicit machine-visible signaling.

## 4) Transport Truth Matrix (Current Repo Contract)

| Surface | State | Truth Contract |
|---|---|---|
| Direct ingest (serial/TCP) | Supported | Claim only what ingest workers persisted and timestamped. |
| MQTT ingest | Supported | Must surface broker/runtime disconnects and partial ingest explicitly. |
| BLE ingest | Unsupported | Must be labeled unsupported; no implied partial support. |
| HTTP ingest | Unsupported | Must be labeled unsupported; no UI/API optimism. |
| Radio transmit/routing execution by MEL | Not implemented as a mesh stack feature | Do not imply MEL itself performs RF routing/propagation execution. |

Evidence needed before claiming success:
- transport connected + ingest loop active;
- packet/evidence persisted;
- timestamps/source context available;
- failure/dead-letter visibility preserved.

Data-state truth labels:
- **Live**: current ingest evidence within freshness window.
- **Stale**: last evidence older than freshness window.
- **Historical**: prior persisted records, not current-state proof.
- **Imported/Offline**: externally sourced artifacts; never live fleet proof by default.
- **Partial/Degraded**: known gaps (disconnects, dead letters, scope gaps) explicitly surfaced.

## 5) Trust Boundaries

- **Operator intent vs system execution**: submission, approval, dispatch, execution result, and audit record are separate states.
- **Historical evidence vs live telemetry**: historical data can explain, not prove current runtime.
- **Imported/offline artifacts vs local live observations**: imported data is context, not direct runtime truth.
- **Local node/runtime truth vs mesh-level inference**: system must distinguish direct observations from inferred conclusions.
- **AuthN/AuthZ boundary**: capabilities and actor scopes must be explicit; avoid silent fail-open paths.

## 6) Common Failure Modes to Guard

- Stale data rendered as current.
- UI language stronger than backend evidence.
- Control-path ambiguity (queued vs approved vs executed).
- Docs drifting ahead of implementation.
- Unsupported capabilities implied through wording.
- Health score overclaiming certainty without causal evidence.
- Missing evidence chain between incident, recommendation, action, and outcome.
- “Temporary best-effort” fallback becoming canonical without explicit risk language.
- Imported/historical data presented as live mesh truth.

## 7) Verification Matrix (Required Thinking)

Every meaningful change must map to `docs/repo-os/verification-matrix.md`:
- backend/domain logic tests,
- API contract checks,
- UI truth rendering checks,
- degraded-state behavior checks,
- control lifecycle/approval/audit checks,
- docs/claim alignment checks.

## 8) Release Bar

Use `docs/repo-os/release-readiness.md` before merge/release claims.
Minimum bar:
- capability claims match implementation;
- degraded states explicitly represented;
- operator-facing wording bounded by evidence;
- verification evidence attached;
- caveats documented with concrete boundaries.

## 9) Anti-Patterns (Reject)

- Phrase parsing where typed truth contract should exist.
- UI-only completion without backend evidence path.
- Docs promising unsupported behavior.
- Broad refactors without operator-truth gain.
- Confidence language that masks uncertainty.
- Silent degradation/fail-open behavior.
- Historical replay represented as live mesh proof.
- Claiming anomaly/remediation intelligence without causal/evidence linkage.

## 10) Moat Priorities

Prefer work that compounds:
1. Operational memory (incident/action/evidence/outcome history).
2. Workflow lock-in (incident-linked controls, evidence packs, review surfaces).
3. Explainable intelligence (diagnosis/recommendation rationale).
4. Audit-grade evidence quality (replayable, attributable, bounded claims).
5. Pricing-power trust wedge (MEL proves/recommends/remembers beyond UI clones).

## 11) Work Classification (for PRs and planning)

- **Maintenance**: correctness, reliability, drift reduction; no new moat surface.
- **Leverage**: improves reusable workflow speed/safety with measurable operator trust gain.
- **Moat**: increases compounding data/decision/workflow advantage competitors cannot copy from UI alone.

Use `docs/repo-os/change-classification.md` for required PR labeling and evidence.

## 12) Default Pressure-Test Questions

For major features/refactors, answer all:
1. What can MEL infer/recommend/prove/remember from real history that a copied dashboard cannot?
2. Does this create compounding intelligence with usage?
3. Does this convert incidents/actions into reusable tests/runbooks/rules?
4. Does this deepen workflow centrality or switching cost?
5. Could a competitor copy this from UI alone?
6. Does this improve trusted control, anomaly intelligence, evidence quality, or operator trust?

## 13) Execution Playbook for Agents

1. Read this file + `docs/repo-os/README.md` before making broad changes.
2. Scope claims to implemented behavior; if uncertain, downgrade claim language.
3. Prefer typed contracts over prose heuristics.
4. Add/adjust tests or verification artifacts for changed truth/control paths.
5. Update relevant repo-os checklist if a new failure mode/class appears.
6. In final summary: include residual risk and explicit caveats.

## 14) Existing Environment Notes

- Use `make build`, `make lint`, `make test`, and `make smoke` as standard verification entry points.
- In this environment, some pre-existing failures may exist; report them honestly, do not mask with selective claims.
- Use `sqlite3` CLI for deterministic DB checks/migrations when needed.

