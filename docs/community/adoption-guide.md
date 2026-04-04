# Community adoption guide

MEL fits best as a **local-first** edge companion for Meshtastic-oriented operators who want **evidence-persistent** visibility, **governed control semantics**, and explicit **degraded / partial** signaling — not a generic cloud dashboard.

## Start here (minimal honest footprint)

1. One host, one radio, **one** enabled transport (serial, TCP direct-node, or MQTT — see README matrix).
2. `make build` → `mel init` / validated config → `mel doctor` → `mel serve`.
3. Open the console; confirm the **transport pill** and any **truth contract** strip match your ingest reality.
4. Run **Privacy** diagnostics before widening exposure (bind, auth, map reporting).

## What to tell newcomers

- MEL is an **incident intelligence + trusted-control OS**: incidents, audits, proofpacks, and action history compound over time.
- **Topology and maps** summarize observed context; they do not prove RF paths or delivery unless backed by ingest evidence.
- **Assistive** features (if enabled) are non-canonical; deterministic records win.

## For evaluators and contributors

- Quick technical eval: [Evaluate MEL in 10 minutes](../ops/evaluate-in-10-minutes.md)
- Contributor path: [CONTRIBUTING.md](../../CONTRIBUTING.md), [docs/repo-os/README.md](../repo-os/README.md)
- Post-launch intake routing: [Post-launch playbook](post-launch-playbook.md)
- Demo / screenshots: [Launch and demo runbook](../runbooks/launch-and-demo.md)
- Claims vs code: [Claims vs reality](claims-vs-reality.md)
