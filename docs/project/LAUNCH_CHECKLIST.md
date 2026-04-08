# MEL Launch Checklist (RC1)

## Last updated: 2026-04-08

This checklist is the release bar for public-facing MEL claims. Treat every unchecked item as a concrete launch risk.

## 1) Build + verification integrity

- [ ] **Core verify stack passes**: `make verify-stack` (lint + test + build + smoke) completes.
- [ ] **Release-grade verification passes**: `make premerge-verify` (or `make premerge-verify-fast` for scoped frontend/site edits) completes with evidence.
- [ ] **Cross-platform binary build**: `make build-cross` succeeds for supported targets.
- [ ] **Version traceability**: `./bin/mel version` reports expected commit and build metadata.

## 2) Runtime truth and degraded-state honesty

- [ ] **Doctor posture is explicit**: `./bin/mel doctor --config ...` reports missing/partial/degraded conditions instead of implied health.
- [ ] **Readiness semantics remain bounded**: `/readyz`, `/api/v1/readyz`, and `/api/v1/status` behave consistently with docs.
- [ ] **Transport support contract is preserved**: serial/TCP and MQTT only; BLE/HTTP ingest remain clearly unsupported in UI/docs.
- [ ] **Live vs stale vs historical/imported states stay distinct** across status, incidents, and exports.

## 3) Privacy, trust, and control safety

- [ ] **Config hardening enforced**: config files used for `mel serve` are permissioned `0600`.
- [ ] **No hidden telemetry broadening**: outbound behavior is explicit and operator-configurable.
- [ ] **Control lifecycle separation visible**: submission → approval → dispatch → execution → audit states remain distinct and attributable.
- [ ] **Security and disclosure paths are current**: `SECURITY.md` and support docs match implementation behavior.

## 4) Public surface and OSS launch posture

- [ ] **Repository entrypoint is coherent**: `README.md` + `docs/README.md` route newcomers to quickstart, trust boundaries, and support matrix without overlap drift.
- [ ] **Public Next.js orientation site verifies**: `make site-verify` passes and routes (`/`, `/quickstart`, `/docs`, `/guide`, `/trust`, `/contribute`) are accurate.
- [ ] **No dead or stale launch links** in README/docs/site critical paths.
- [ ] **Operator language remains honest**: no implied AI canonical truth, mesh-routing claims, or unsupported feature overreach.

## 5) Evidence package for release decisions

- [ ] **Command transcript captured**: exact verification commands and outcomes are attached to PR/release notes.
- [ ] **Caveats are explicit**: unresolved gaps are documented with bounded impact and follow-up owner.
- [ ] **Claims-vs-reality check updated**: public wording remains within implemented behavior boundaries.
