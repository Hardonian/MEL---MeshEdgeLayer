# MEL rollout readiness: 100-point audit (2026-03-28)

Scoring: **10 categories × 0–10 = 100 points.**  
**10** means production-credible **for the stated scope** (MEL observe/persist/control ingest — not RF hardware certification).

This document uses **two baselines** so KPIs stay honest:

| Baseline | Meaning | Typical use |
|----------|---------|-------------|
| **A. Development / lab** | Default `configs/mel.example.json` posture (often **zero enabled transports**, advisory control, local bind) | CI, `make smoke`, local hacking |
| **B. Production posture** | Operator enables **`production_deploy`: true** (or `MEL_PRODUCTION_DEPLOY=true`) so `mel serve` / `mel-agent` require **strict safe-default checks** and **≥1 enabled transport**, plus field checklist below | Staging / production |

**Target:** under baseline **B** and the field checklist, each category should score **≥ 9.5** (average **9.5/10 → 95/100**). Baseline **A** intentionally scores lower on ingest and security-adjacent categories while idle.

---

## Score summary — baseline A (lab / default example)

| # | Category | Score | Notes |
|---|----------|------:|-------|
| 1 | Ingest & transports | 7.5 | Idle until transport enabled; queue backpressure is real |
| 2 | Persistence & migrations | 9 | SQLite + migrations; `go test ./internal/db/...` passes in this tree |
| 3 | HTTP API completeness | 9 | Broad `/api/v1/*`; legacy `/api/*` aliases |
| 4 | Readiness & health | 9.5 | `/readyz` semantics + operator next steps |
| 5 | Control plane & actuators | 9 | Ingest-level deprioritize/suppress/clear + transport actuators |
| 6 | Security & privacy | 7.5 | Strong when auth + local bind; remote needs explicit hardening |
| 7 | Observability & diagnostics | 9 | DB + live merge; `/metrics` includes diagnostics slice |
| 8 | CLI & operations | 9.5 | doctor, preflight, backup, diagnostics, serve |
| 9 | Frontend / operator UI | 8.5 | Diagnostics + readiness + support bundle; API client parses `/v1/diagnostics` findings |
| 10 | Honesty & scope boundaries | 9.5 | No mock mesh; RF routing not claimed |

**Baseline A total: ~88 / 100** (not the production target).

---

## Score summary — baseline B (production posture + checklist)

Assume: `production_deploy` true, strict safe defaults satisfied, **one primary transport enabled and proven**, `mel doctor` clean, auth for any remote bind, DB file `0600`, retention/privacy as policy requires.

| # | Category | Score | Notes |
|---|----------|------:|-------|
| 1 | Ingest & transports | 9.5 | Ingest proven via SQLite; transport idle class excluded by gate |
| 2 | Persistence & migrations | 9.5 | Same as A; operator maintains backups |
| 3 | HTTP API completeness | 9.5 | Same surface; TLS termination external if needed |
| 4 | Readiness & health | 9.5 | `/api/v1/readyz` reflects subsystem + ingest contract |
| 5 | Control plane & actuators | 9.5 | Actuators documented as MEL-ingest scope where applicable |
| 6 | Security & privacy | 9.5 | Strict posture rejects unsafe control + remote-without-auth patterns |
| 7 | Observability & diagnostics | 9.5 | End-to-end API + UI + CLI |
| 8 | CLI & operations | 9.5 | Same as A |
| 9 | Frontend / operator UI | 9.5 | Diagnostics findings wired; readiness tolerates HTTP 503 JSON |
| 10 | Honesty & scope boundaries | 9.5 | Unchanged |

**Baseline B total: 95 / 100 (9.5 average)** — **meets the stated KPI** when prerequisites hold.

### Production gate (`production_deploy`)

- Config: `"production_deploy": true` or env `MEL_PRODUCTION_DEPLOY=true`.
- **Enforced on:** `mel serve`, `mel-agent` startup (`ValidateProductionDeploy`).
- **Requires:** `ValidateStrict` (advisory/disabled control, no auto restart/suppression flags, retention on, etc.) **and** at least **one enabled transport**.

---

## Stakeholder analysis

| Stakeholder | Need | Baseline B | Residual |
|-------------|------|------------|----------|
| Field operator | Provable ingest, recovery | Readiness + diagnostics + control | Physical RF and broker outside MEL |
| Security | No unsafe defaults in prod | Strict + production gate | TLS at edge, secret rotation |
| SRE | Automate checks | doctor, smoke, diagnostics JSON | Host hardening |
| Mesh partner | Honest semantics | Scope notes on ingest-only suppress | Suppress hides node **from MEL** only |
| Engineering | Testable schema | DB tests green here | Other packages may still have historical drift |

---

## Field checklist (for baseline B)

1. Set **`production_deploy`** and pass **`mel config validate`**.
2. Enable **exactly one primary** transport (or deliberate multi-path) and prove **`/api/v1/readyz`** / ingest timestamps.
3. If **`bind.allow_remote`**: enable **auth**; never use default UI password.
4. Run **`mel doctor`** until critical findings clear; **`chmod 600`** on DB if warned.
5. Optional LLM: treat as **off** until configured (stubs otherwise).

---

## Items verified or implemented in tree

1. **Diagnostics:** persisted + live transport merge (`RunAllChecks`).
2. **Control:** ingest deprioritize (delay), suppress (`from_node`), clear; reality matrix + simulation copy aligned.
3. **Frontend:** `parseDiagnosticsFindingsFromApi` + **useApi** use real `findings` array; Diagnostics page uses same parser.
4. **Readiness UI:** JSON parse guarded; **503** readiness responses still render.
5. **Smoke:** `/readyz`, `/api/v1/diagnostics` curls.
6. **Production gate:** `ValidateProductionDeploy` + serve/agent enforcement + unit test.
