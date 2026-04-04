# Launch and demo runbook

Repeatable, **honest** demo paths for screenshots, partner calls, and README imagery. If a step cannot be reproduced from the repo, it does not belong in launch collateral.

## Preconditions

- Build: `make build-cli` (or `make build`).
- Config: copy an example config, set `storage.database_path`, `chmod 600` the file.
- Validate: `./bin/mel config validate --config <path>` then `./bin/mel doctor --config <path>`.
- Node **24.x** only if you touch `frontend/` (see `frontend/.nvmrc`).

## Golden-path screens (show these first)

Use live data when available; otherwise prefer **explicit empty/degraded** states over fabricated “healthy” demos.

1. **Command surface (home)** — transport pill, shift/teaser cards, stale/degraded banners if applicable.
2. **Incidents** — open queue with evidence-strength / sparsity visible.
3. **Incident detail** — replay link, handoff section, bounded wording on what changed.
4. **Topology** — treat as **observed context**; caption screenshots accordingly.
5. **Control actions** — approval / queue / denial visibility (advisory vs executable per reality matrix).
6. **Settings** — operator truth contract block + runtime truth strip.
7. **Diagnostics / Privacy** — findings when posture is non-default (redacted exports).

## Screenshot checklist

- [ ] Visible **version** or build identity (Settings or status) when claiming a release.
- [ ] No precise locations or secrets in frame.
- [ ] Caption states **live vs historical vs degraded** if ambiguous on static images.
- [ ] Topology images include a line of context: “last-seen / ingest-derived, not path proof.”

## Scripted checks

```bash
make demo-verify    # demo scenarios + evidence script (after build-cli)
make smoke          # requires ./bin/mel
```

## Evaluation shortcut

For a skeptical pass in minutes: [Evaluate MEL in 10 minutes](../ops/evaluate-in-10-minutes.md).

## Honesty guardrails

- Do not imply BLE or HTTP ingest, or MEL-performed RF routing, unless implementation and tests justify it (they currently do not).
- Prefer `docs/product/HONESTY_AND_BOUNDARIES.md` and `docs/repo-os/terminology.md` over ad hoc marketing copy.
