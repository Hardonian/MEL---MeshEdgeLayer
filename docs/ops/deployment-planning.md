# Deployment planning (operator field guide)

MEL deployment planning is **advisory**: it uses **observed topology** (nodes, links, packet-derived signals) and optional **operator-supplied assumptions**. It does **not** simulate RF coverage, terrain, or map-based propagation.

## What a deployment plan is

A **deployment plan** is an operator-authored sequence of proposed steps (add node, improve uptime, bridge clusters, observe-only, etc.) stored in SQLite. Plans can reference a **versioned planning input set** so assumptions remain interpretable later.

## Evidence models

- **Topology-only**: analysis uses graph shape and mesh intelligence only; no operator assumptions were supplied (or none that the estimators consume).
- **Topology + operator assumptions**: you recorded assumptions (placement class, uptime intent, objectives). Some fields may be **stored but not yet used** by estimators — the API marks those explicitly.

## Resilience vs connectivity

**Current connectivity** is what transports and recent packets show. **Resilience** is a **structural** judgment from the observed graph (bridges, redundancy proxies, fragmentation). A mesh can be “up” but **fragile** if one bridge node carries critical paths. MEL does **not** claim a partition **will** occur — only that **loss of a node may increase fragmentation** in the observed model.

## Critical-node warnings

Per-node scores rank **attention priority** from topology heuristics, not RF proof. Treat **SPOF** labels as **probable structural risk**, not a guarantee.

## Comparing plans (low-regret vs upside)

Plan comparison ranks candidates using explicit dimensions: reversibility, observation burden, diagnostic value, assumption fragility, and coarse upside proxies. **Low-regret** favors reversible, diagnostic moves with limited disruption — not the same as “highest upside.” When uptime is unstable or evidence is weak, **wait and observe** often outranks buying hardware.

## Recording execution and validation

1. Start an **execution** (baseline graph hash and mesh assessment id are stored).
2. Mark **steps** as attempted with notes.
3. After your observation window, run **validation** to compare post-change metrics to baseline.

Verdicts are honest: **supported**, **contradicted**, **inconclusive**, **insufficient observation**, or confounded when multiple changes overlap. Validation loads the **baseline mesh assessment from the database** when available so “before” is not accidentally the same snapshot as “after.”

## Resilience advisory alerts

When topology is enabled, MEL may persist **synthetic** alerts under transport `planning` / type `advisory`. These are **not transport failures**; they surface structural concerns with evidence references. Stale advisories are resolved when conditions clear.

## CLI quick reference

- `mel plan create|list|show|edit|compare|inputs|mark-step|outcome`
- `mel recommend next`
- `mel mesh simulate|resilience|critical`
- `mel playbook suggest`

## API (selected)

- `GET /api/v1/planning/bundle` — full advisory bundle
- `GET /api/v1/planning/recommend/next` — consolidated “best next move” card
- `GET /api/v1/planning/advisory-alerts` — planning advisories
- `POST /api/v1/planning/input-versions` — versioned assumption set
- `POST /api/v1/planning/executions/start` — begin tracked execution
- `POST /api/v1/planning/executions/step` — mark step attempted
- `POST /api/v1/planning/executions/validate` — record validation outcome
- `GET /api/v1/planning/executions?plan_id=` — list executions
- `GET /api/v1/planning/executions/validations?execution_id=` — validation history

## Workflow examples (grounded)

- **Isolated lone wolf**: prefer one elevated stationary observation before adding endpoints; defer large changes without history.
- **Two-node fragile bridge**: add redundancy or improve uptime on the bridge class node before corridor growth.
- **Improve uptime before expansion**: if viability is intermittent, stabilization often beats new hardware.
- **Temporary event**: mark assumptions as temporary; keep observation windows short; validate with fragmentation/resilience deltas, not coverage claims.

## Honesty rules (enforced in product copy)

Avoid implying **optimal placement**, **guaranteed partition**, or **RF coverage improvement** unless backed by the stated evidence model. Prefer **likely**, **plausible**, **topology-only estimate**, and **advisory**.
