# Community adoption guide

MEL fits best as a **local-first** edge companion for Meshtastic-oriented operators who want **evidence-persistent** visibility, **governed control semantics**, and explicit **degraded / partial** signaling — not a generic cloud dashboard.

## Pick your lane

| Lane | Start |
| --- | --- |
| First-time operator | [Getting started: Quickstart](../getting-started/QUICKSTART.md) |
| Club / event / “try before hardware” | [Scenario library](SCENARIO_LIBRARY.md) + `make demo-seed` |
| Contributor | [START_HERE](START_HERE.md) → [CONTRIBUTOR_PATHS](CONTRIBUTOR_PATHS.md) |
| Field tester | [Field testing](FIELD_TESTING.md) + [field report template](../../.github/ISSUE_TEMPLATE/field_report.md) |
| Hardware anecdote | [Hardware compatibility](HARDWARE_COMPATIBILITY.md) |

## Start here (minimal honest footprint)

1. One host, one radio, **one** enabled transport (serial, TCP direct-node, or MQTT — see [README transport matrix](../../README.md)).
2. `make build` → `mel init` / validated config → `mel doctor` → `mel serve`.
3. Open the console; confirm the **transport** surface and any **truth contract** strip match your ingest reality.
4. Run **Privacy** diagnostics before widening exposure (bind, auth, map reporting).

## Role sketches (documentation-level)

These are **posture guides**, not promises of automation:

- **RPi / small-board gateway**: serial or TCP to a local radio; MQTT to a broker on the same host or LAN. Keep config mode `0600`; see [Installation](../getting-started/installation.md).
- **Laptop observer**: run `mel serve` bound to loopback first; expand bind only after privacy review.
- **MQTT-only ingest**: valid when your broker path is the evidence source; still label bridge vs RF honestly in write-ups.
- **Event ops**: prefer explicit degraded banners over “all green” demos; capture states per [Showcase and screenshots](SHOWCASE_AND_SCREENSHOTS.md).

## What to tell newcomers

- MEL is an **incident intelligence + trusted-control OS**: incidents, audits, proofpacks, and action history compound over time.
- **Topology and maps** summarize observed context; they do not prove RF paths or delivery unless backed by ingest evidence.
- **Assistive** features (if enabled) are non-canonical; deterministic records win.

## Topology cookbooks

Deployment shapes and memory notes live under [`topologies/`](../../topologies/README.md). They document patterns; they are not an installer.

## For evaluators and contributors

- Quick technical eval: [Evaluate MEL in 10 minutes](../ops/evaluate-in-10-minutes.md)
- Contributor path: [CONTRIBUTING.md](../../CONTRIBUTING.md), [docs/repo-os/README.md](../repo-os/README.md)
- Post-launch intake routing: [Post-launch playbook](post-launch-playbook.md)
- Demo / screenshots: [Launch and demo runbook](../runbooks/launch-and-demo.md)
- Claims vs code: [Claims vs reality](claims-vs-reality.md)
