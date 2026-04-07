# Support for MEL (MeshEdgeLayer)

MEL is an open-source project (GPL-3.0). There is **no guaranteed response time** for unpaid community support. This document states what you can expect and how to get help without weakening truth boundaries.

## Self-service (start here)

| Need | Where |
| --- | --- |
| Install and first run | [docs/getting-started/QUICKSTART.md](docs/getting-started/QUICKSTART.md), [docs/getting-started/README.md](docs/getting-started/README.md) |
| Demo without hardware | `make demo-seed` then `./bin/mel serve --config demo_sandbox/mel.demo.json` — [QUICKSTART](docs/getting-started/QUICKSTART.md) |
| What is / is not supported | [README.md](README.md) transport matrix, [docs/product/HONESTY_AND_BOUNDARIES.md](docs/product/HONESTY_AND_BOUNDARIES.md), [docs/ops/limitations.md](docs/ops/limitations.md) |
| Compatibility | [docs/release/COMPATIBILITY_AND_SUPPORT_MATRIX.md](docs/release/COMPATIBILITY_AND_SUPPORT_MATRIX.md), [docs/ops/support-matrix.md](docs/ops/support-matrix.md), [docs/community/HARDWARE_COMPATIBILITY.md](docs/community/HARDWARE_COMPATIBILITY.md) |
| Operations and incidents | [docs/ops/OPERATIONS_RUNBOOK.md](docs/ops/OPERATIONS_RUNBOOK.md), [docs/runbooks/README.md](docs/runbooks/README.md) |
| Troubleshooting | [docs/ops/troubleshooting.md](docs/ops/troubleshooting.md), [docs/ops/troubleshooting-transports.md](docs/ops/troubleshooting-transports.md) |
| Privacy and data | [docs/release/PRIVACY_AND_DATA_POSTURE.md](docs/release/PRIVACY_AND_DATA_POSTURE.md), [docs/privacy/posture.md](docs/privacy/posture.md) |

## Community channels

- **Bug reports and feature work:** use GitHub Issues for this repository — see [docs/repo-os/canonical-github.md](docs/repo-os/canonical-github.md) for the canonical URL (templates under `.github/ISSUE_TEMPLATE/`).
- **Contributions:** [CONTRIBUTING.md](CONTRIBUTING.md), [docs/community/START_HERE.md](docs/community/START_HERE.md), [docs/community/CONTRIBUTOR_PATHS.md](docs/community/CONTRIBUTOR_PATHS.md).

## What maintainers will not do in public issues

- Invent RF coverage, routing success, or delivery guarantees from screenshots or narratives.
- Treat assistive inference (if present) as canonical system truth.
- Promise enterprise compliance, certifications, or SLAs that are not contractually agreed outside this repo.

## Commercial, pilot, or priority support

Packaging and honest commercial boundaries are described in [docs/product/EDITION_PACKAGING.md](docs/product/EDITION_PACKAGING.md). Internal planning notes (not contractual) live under [docs/internal/private/README.md](docs/internal/private/README.md). **There is no obligation to provide paid support**; when in doubt, assume community-best-effort only.

## Security-sensitive reports

Do **not** post secrets, session material, or precise location payloads in public issues. Follow [SECURITY.md](SECURITY.md).
