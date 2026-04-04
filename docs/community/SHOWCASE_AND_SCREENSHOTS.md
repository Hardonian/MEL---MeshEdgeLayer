# Showcase and screenshots

MEL should look like a serious operator console in still images — **without** overstating live mesh truth. Follow the same rules as the launch runbook.

## Canonical workflow

1. Read [Launch and demo runbook](../runbooks/launch-and-demo.md).
2. Read [Launch screenshot checklist](../ops/launch-screenshot-checklist.md).
3. Prefer **seeded demo** or **honest empty/degraded** states over fake “all green” staging.

## Where to put files

| Path | Purpose |
| --- | --- |
| `assets/` | README hero and other repo-level imagery checked into git |
| `docs/showcase/captures/` | Optional curated screenshots (add only when you have real files; see below) |

If `docs/showcase/captures/` is empty, capture locally first, then add PNG/WebP with a short caption in this section or in the root README.

## Captions (required discipline)

Every public screenshot should answer:

- **Live, historical, imported, or demo-seeded?**
- **Which transport(s) were active?**
- For topology: **ingest-derived context, not proof of RF path** (see [terminology](../repo-os/terminology.md)).

## Scripted checks before you claim “demo works”

```bash
make build-cli
make demo-verify
make smoke
```

## When automation cannot run

If headless CI cannot render the browser, document the exact manual steps and keep scripts truthful — do not check in placeholder images labeled as production captures.
