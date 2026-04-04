# Start here

MEL (MeshEdgeLayer) is a **local-first** incident-intelligence and trusted-control surface for mesh operations. It is **not** a mesh routing stack and does **not** prove RF paths unless your ingest evidence supports that claim.

Pick one entry path:

## I want to run it (operator)

1. [Quickstart](../getting-started/QUICKSTART.md) — build, `mel init`, `mel doctor`, `mel serve`.
2. [Support matrix](../ops/support-matrix.md) and [limitations](../ops/limitations.md) — what is and is not implemented.
3. [Evaluate in 10 minutes](../ops/evaluate-in-10-minutes.md) — API and status sanity checks.
4. [Topology cookbook](../../topologies/README.md) — deployment shapes as documentation (not automatic provisioning).

## I want to try it without radios (sandbox)

1. `make build-cli` (or `make build`).
2. `make demo-seed` — seeds a deterministic scenario into `demo_sandbox/mel.demo.json` (see [Scenario library](SCENARIO_LIBRARY.md)).
3. `./bin/mel serve --config demo_sandbox/mel.demo.json` and open `http://127.0.0.1:8080`.

This is **fixture-backed** data for UX and workflow evaluation, not proof of your live mesh.

## I want to contribute code or docs

1. [CONTRIBUTING.md](../../CONTRIBUTING.md) — build, verification, PR checklist.
2. [Contributor paths by role](CONTRIBUTOR_PATHS.md) — frontend, transport, docs, field evidence.
3. [First PR paths](../contributor/FIRST_PR_PATHS.md) — small, bounded changes.
4. [Repo OS](../repo-os/README.md) — classification, verification matrix, release readiness.

## I ran it in the field and want to report results

1. [Field testing](FIELD_TESTING.md) — privacy and what to redact.
2. Open a **Field report** issue (`.github/ISSUE_TEMPLATE/field_report.md`) or **Hardware compatibility** (`.github/ISSUE_TEMPLATE/hardware_compatibility.md`).

## I represent a sponsor or enterprise evaluation

1. [Product honesty boundaries](../product/HONESTY_AND_BOUNDARIES.md) and [claims vs reality](claims-vs-reality.md).
2. Run the verification targets in the root [README.md](../../README.md) and attach outputs; do not treat green CI as field certification.
