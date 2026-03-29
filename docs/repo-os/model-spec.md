# MEL Repo-Local Model Spec (Execution Behavior Contract)

Purpose: make MEL execution alignment robust even when prompts are short, imperfect, or phrased differently.

## 1) Identity and mission lock
Any agent operating in this repo must preserve:
- MEL as truthful operator OS / communications OS;
- incident-intelligence + trusted-control posture;
- local-first, privacy-first, open/self-hosted architecture bias;
- evidence-first and deterministic canonical truth.

## 2) Hard behavior constraints
Agents must:
- refuse overclaims unsupported by implementation + verification evidence;
- keep degraded/partial/unknown states explicit and machine-visible;
- preserve support matrix boundaries (no BLE/HTTP ingest claims; no MEL RF routing execution claims);
- treat local inference as assistive/non-canonical;
- preserve control lifecycle separation (submitted/approved/dispatched/executed/audited);
- preserve tenant and trust boundaries.

## 3) Truth hierarchy
Default to this order:
1. typed deterministic evidence;
2. deterministic calculators + bounded heuristics;
3. assistive inference/estimation;
4. prose narrative.

## 4) Precision-layering requirement
For major diagnostics/health/recommendation changes, reason explicitly through:
1. semantics,
2. telemetry,
3. frequency/radio context,
4. physical/environment context,
5. sensor context,
6. spatial/navigation context,
7. algorithmic/calculator outputs,
8. mixed-channel truth posture.

Observed/inferred/estimated/unknown distinctions are mandatory.

## 5) Privacy/open/self-hosted requirement
Agents must preserve:
- no mandatory cloud dependency;
- no hidden telemetry;
- local data ownership + retention/export/delete semantics;
- explicit key/material boundaries;
- optional integrations explicitly labeled;
- recurring-cost awareness in architecture choices.

## 6) Build-vs-borrow policy
Build MEL-specific differentiators (truth model, incident intelligence, proofpacks, action memory, trust policy orchestration).
Borrow OSS for commodity primitives unless truth/privacy/cost constraints fail.

## 7) Local inference policy
- Ollama default path, llama.cpp advanced path.
- Compression-aware routing (including TurboQuant-compatible experiments) allowed with caveats.
- CPU fallback required.
- Canonical truth and base MEL viability cannot depend on inference runtime success.

## 8) Output/reporting requirements
Implementation reports should include:
- classification (Maintenance/Leverage/Moat),
- verification commands/results,
- residual risks/caveats,
- explicit statement of any narrowed claims.

## 9) Canon alignment maintenance
When updating repo doctrine, agents must update canonical artifacts in-place (not create competing guidance) and keep `AGENTS.md`, this model spec, repo-os gates, and README entrypoints aligned.
