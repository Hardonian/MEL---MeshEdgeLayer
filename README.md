# MEL — MeshEdgeLayer

<!-- BEGIN: REPO HERO -->
![MEL-MeshEdgeLayer — hero generated locally on the GPU stack](assets/repo-hero.png)
<!-- END: REPO HERO -->

**MEL is an evidence-first operator OS for mixed-channel mesh incidents and trusted control.**

If MEL cannot prove a claim from persisted runtime evidence, it should render **unknown/degraded**, not confidence theatre.

## Who MEL is for

Teams running field/edge operations that need:
- local-first operation with self-hosted defaults,
- explicit live vs stale vs historical truth boundaries,
- audited action lifecycle (submission → approval → execution → audit),
- incident memory that compounds over time.

## What MEL is not

- Not a mesh routing stack.
- Not BLE ingest support (unsupported today).
- Not HTTP ingest support (unsupported today).
- Not proof of RF propagation/routing success without evidence.
- Not an “AI truth engine” (local inference is assistive, never canonical truth).

## Why it matters

Most dashboards collapse uncertainty. MEL does the opposite: it keeps degraded/partial/unknown states explicit so operators can trust what they are seeing under stress.

## Quick try (10 minutes)

```bash
make build
./bin/mel init --config .tmp/mel.json
chmod 600 .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json
```

Open <http://127.0.0.1:8080>.

Operator orientation after first boot:

- ` /transports` → confirm ingest links are connected vs stalled (no fake “live” by default).
- ` /status` → inspect degraded/unknown conditions and heartbeat posture.
- ` /incidents` → review open queue and evidence strength.

No-radio deterministic evaluation:

```bash
make demo-seed
./bin/mel serve --config demo_sandbox/mel.demo.json
```

## Newcomer funnel

1. **Start here:** [docs/README.md](docs/README.md) or public orientation at `site/` (`/`, `/quickstart`, `/docs`, `/guide`).
2. **Try MEL:** [docs/getting-started/QUICKSTART.md](docs/getting-started/QUICKSTART.md), [docs/ops/evaluate-in-10-minutes.md](docs/ops/evaluate-in-10-minutes.md)
3. **Understand boundaries:** [docs/ops/support-matrix.md](docs/ops/support-matrix.md), [docs/ops/limitations.md](docs/ops/limitations.md), [docs/community/claims-vs-reality.md](docs/community/claims-vs-reality.md)
4. **Contribute safely:** [CONTRIBUTING.md](CONTRIBUTING.md)
5. **Operate for real:** [docs/ops/OPERATIONS_RUNBOOK.md](docs/ops/OPERATIONS_RUNBOOK.md)

## Verification commands (real targets)

- `make lint`
- `make test`
- `make build`
- `make smoke`
- `make verify`
- `make premerge-verify` (release-grade local chain)

## Contributing & support

- [CONTRIBUTING.md](CONTRIBUTING.md)
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- [SECURITY.md](SECURITY.md)
- [SUPPORT.md](SUPPORT.md)
- [Current status](docs/project/CURRENT_STATUS.md)

MEL is GPL-3.0 licensed; see [LICENSE](LICENSE).


## Repository Operations Standards

- Squash-only merges
- Auto-delete merged branches
- Weekly dependency update windows
- Security scanning in CI

---

---

## Related Hardonia projects

<p align="center">
  <a href="https://aiautomatedsystems.ca"><img src="https://img.shields.io/badge/AI_Automated_Systems-Visit-0f766e?style=for-the-badge&logo=cloudflare" alt="AI Automated Systems" /></a>
  <a href="https://github.com/Hardonian/ollama-router"><img src="https://img.shields.io/badge/ollama--router-181717?style=for-the-badge&logo=github" alt="ollama-router" /></a>
  <a href="https://github.com/Hardonian/ai-lab-audit-api"><img src="https://img.shields.io/badge/ai--lab--audit--api-181717?style=for-the-badge&logo=github" alt="ai-lab-audit-api" /></a>
  <a href="https://github.com/Hardonian/ai-lab-command-center"><img src="https://img.shields.io/badge/command--center-181717?style=for-the-badge&logo=github" alt="ai-lab-command-center" /></a>
  <a href="https://github.com/Hardonian/storefront"><img src="https://img.shields.io/badge/storefront-181717?style=for-the-badge&logo=github" alt="storefront" /></a>
</p>

<p align="center"><strong>Part of the <a href="https://aiautomatedsystems.ca">Hardonia</a> open-source + services stack.</strong></p>

<p align="center">
  <a href="https://aiautomatedsystems.ca/p/repo-rescue-saas-audit"><img src="https://img.shields.io/badge/Get_a-SaaS_Repo_Rescue_Audit-635BFF?style=for-the-badge&logo=stripe&logoColor=white" alt="SaaS Repo Rescue Audit" /></a>
</p>

<details>
<summary>What this audit covers</summary>

A fixed-scope review of **auth, billing, RLS, and webhook** correctness — the bugs that cost you customers and chargebacks. Runs locally on your infrastructure.
</details>
