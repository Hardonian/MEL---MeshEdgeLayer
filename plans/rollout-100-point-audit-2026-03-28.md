# MEL rollout readiness: 100-point audit (2026-03-28)

Scoring: each category is **0–10** (10 = production-credible for stated scope). **Total /100**. This audit is evidence-based against the repository; it does not certify RF hardware or third-party brokers.

## Score summary

| Category | Score | Notes |
|----------|------:|-------|
| 1. Ingest & transports | 8 | Multi-transport; explicit idle when unconfigured; ingest queue backpressure is real |
| 2. Persistence & migrations | 8 | SQLite + deterministic migrations; schema drift in some legacy tests remains a repo risk |
| 3. HTTP API completeness | 8 | Broad v1 surface; legacy `/api/*` aliases maintained |
| 4. Readiness & health | 9 | `/readyz` + `readiness.Evaluate` with explicit operator next steps |
| 5. Control plane & actuators | 7 | Restart/resubscribe/backoff/recheck + **ingest-level** deprioritize/suppress/clear (see below) |
| 6. Security & privacy | 7 | Auth paths exist; operator must lock down bind + secrets for field use |
| 7. Observability & diagnostics | 8 | Diagnostics merge DB + live transport evidence; metrics embed findings |
| 8. CLI & operations | 8 | doctor, status, backup, export, diagnostics, serve |
| 9. Frontend / operator UI | 6 | Functional dashboard; not all audit gaps from Phase 0 plan are UI-complete |
| 10. Honesty & scope boundaries | 9 | No mock mesh claims; Meshtastic routing not asserted as controlled by MEL |

**Total: 78 / 100**

### Category 5 detail (actuators)

- **Shipped and real:** transport interrupt (restart/resubscribe), backoff multiplier, health recheck, **ingest deprioritization** (bounded worker delay per packet until expiry), **per-transport per-`from_node` ingest drop** until expiry, **clear** of those windows.
- **Explicitly not Meshtastic RF routing:** deprioritize does not change firmware or mesh routing tables; suppress drops **decoded ingest** in MEL only (RF may still carry packets).

---

## Stakeholder analysis

| Stakeholder | Primary need | How MEL serves it | Residual risk |
|-------------|--------------|-------------------|---------------|
| Field operator | Know if ingest is real; recover transports | `/readyz`, status, diagnostics, control actions | Must configure transport correctly; LLM assist remains optional/stub without keys |
| Security / compliance | No silent data exfil; redacted exports | Privacy flags, export redaction, auth knobs | Remote bind + weak credentials if misconfigured |
| Platform / SRE | Automate checks, support bundle | `mel doctor`, diagnostics JSON, support zip | Disk/DB permissions on shared hosts |
| Mesh community partner | Honest semantics | Docs + API say “observe/persist” not “fix RF” | Suppression can hide a node **from MEL** while RF continues |
| Engineering | Maintainable schema & tests | Migrations, package boundaries | Some test packages have historical drift (noted in AGENTS.md) |

---

## Checklist completion (rollout)

| Item | Status |
|------|--------|
| Enable primary transport in prod config | **Operator** — document only; smoke uses idle config by design |
| Use `/readyz` + `/api/v1/status` for ingest truth | **Verified** in `scripts/smoke.sh` (readyz tolerates 503) |
| Use `/api/v1/diagnostics` | **Verified** in smoke |
| Control: deprioritize / suppress / clear | **Implemented** as ingest-level actuators + matrix updated |
| LLM automation | **Out of scope** — remains stub without operator configuration |

---

## Items addressed in this iteration

1. **Diagnostics:** merge persisted `transport_runtime_*` with live `transport.Health` (prior commit).
2. **Control actuators:** real ingest-side deprioritization, attributed-node suppression, and clear; `DefaultActionRealityMatrix` and simulation/risk text aligned.
3. **Attribution:** `suppressionAttribution` fills `TargetNode` from numeric `from_node` when strong.
4. **Smoke:** curls `/readyz`, `/api/v1/readyz` (non-fatal), `/api/v1/diagnostics`.
