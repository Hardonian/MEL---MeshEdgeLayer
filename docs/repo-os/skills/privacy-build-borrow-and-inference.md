# Skill: Privacy-First Architecture + Build-vs-Borrow + Local Inference Policy

Use when touching architecture, dependencies, telemetry/export behavior, model runtime, or integration strategy.

## Privacy/open/self-hosted checks
- [ ] No mandatory cloud coupling introduced.
- [ ] No hidden telemetry or silent outbound data paths.
- [ ] Retention/export/delete semantics are preserved or updated.
- [ ] Key/material boundaries are explicit.
- [ ] Optional integrations are clearly marked optional.

## Build-vs-borrow checks
- [ ] Feature is classified as MEL differentiator vs commodity primitive.
- [ ] Commodity capability prefers OSS component unless truth/privacy/cost constraints fail.
- [ ] Decision rationale is captured in ADR/docs for boundary-affecting changes.

## Local inference checks
- [ ] Inference remains assistive and non-canonical.
- [ ] Deterministic truth path remains available without inference runtime.
- [ ] Ollama default / llama.cpp advanced assumptions preserved.
- [ ] Compression-aware and CPU fallback behavior is explicit.
