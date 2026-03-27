# Partial fleet visibility and operator scope

MEL is **instance-first**: each process has one SQLite database. Evidence, timelines, and control actions in that database are **authoritative only for what this instance ingested and stored**.

## What `scope` in config does

Optional `scope` fields label **site** and **fleet** boundaries for exports, support bundles, and APIs. They do **not**:

- enable cross-instance synchronization in core,
- imply global ordering or fleet-wide health,
- prove RF coverage, topology, routes, or congestion.

When `scope.expected_fleet_reporter_count` is greater than 1, operators should expect **partial-fleet visibility**: this database still reflects **one** reporting instance unless you merge evidence out-of-band.

## Interpreting APIs

- `GET /api/v1/fleet/truth` and `mel fleet truth` return canonical **truth posture**, **visibility posture**, and **capability posture** (federation off; local execution only unless you add optional federation handlers).
- `GET /api/v1/version` includes `fleet_truth` when the database is available.

## Evidence and physics

- **Observation is not coverage.** Missing data is not proof of absence.
- **Repeated observations** from multiple gateways are **not** automatic flooding or redundancy proof; treat them as **symptom-level** evidence until instrumentation supports a stronger classification.

## Merge and dedupe

Use typed merge disposition (`internal/fleet`) when building tooling: preserve **per-observer** rows where keys collide across observers; do not collapse ambiguity into a single causal story without evidence.
