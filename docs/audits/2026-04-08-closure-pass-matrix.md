# MEL closure pass matrix (2026-04-08)

Scope: repository-level pressure test across product truth, trust artifacts, contributor pull, and commercial legibility.

## Ranked findings matrix

| Rank | Area label(s) | Class | Blind spot / weak area | Root cause | Closure in this pass | Status |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Product, UX/docs, Packaging | Leverage | Time-to-first-proof still required stitching several commands/docs; new operators could run MEL but miss deterministic evidence capture and truth boundary prompts. | Demo pathways existed but were split across Make targets and runbooks. | Added one-command `make first-proof` flow (`scripts/first-proof.sh`) that seeds deterministic scenario data, generates evidence artifacts, and prints explicit claim boundaries. Wired into docs/site/readme. | Closed |
| 2 | Product, Trust, UX/docs | Leverage | “Useful quickly” path did not consistently foreground what MEL does **not** prove in the same step where first proof is generated. | Truth caveats were present but distributed across docs pages. | Embedded non-claim boundaries directly in first-proof script output and quickstart surfaces. | Closed |
| 3 | GTM/business, Packaging, OSS/community | Maintenance | Venture/trust/contributor artifacts existed but discoverability of “closure pass” priorities and rationale was implicit. | Information is broad and mature, but lacks a single ranked closure artifact for planning and review. | Added this ranked matrix document to capture prioritized blind spots, root causes, and closure outcomes in one auditable location. | Closed |
| 4 | Architecture, Moat | Moat | Compounding memory loop articulation exists, but first operator interaction still risks looking like “dashboard demo” instead of “evidence system.” | Early onboarding bias toward starting server before showing generated proof artifacts. | First-proof flow now forces evidence-run generation before serve step, nudging first interaction toward proof artifacts and replayable evidence. | Partially closed |

## Pressure-test summary per change

### `make first-proof` flow

- Improves operator trust: yes, deterministic artifacts generated before UI walk-through.
- Reduces ambiguity: yes, script prints explicit supported/unsupported boundaries.
- Improves time-to-first-proof: yes, one command replaces multi-doc stitching.
- Improves star/adopt/pilot/contribute path: yes, easier reproducible demo setup for evaluators and contributors.
- Strengthens moat: moderate; starts first run with evidence artifacts, not just dashboard visuals.
- UI-copyability resistance: moderate; evidence artifact generation + truth contract is behavior-level, not visual.

## Residual risk notes

- First-proof still depends on a local Go toolchain and successful binary build.
- Full release-grade confidence still requires `make premerge-verify` and optional site/frontend checks.
- No new claims were introduced for unsupported BLE/HTTP ingest or RF routing execution.
