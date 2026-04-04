# Field testing and community evidence

MEL’s value shows up under real radios, noisy environments, and imperfect configs. This guide keeps field contributions **honest** and **safe to share publicly**.

## What “field evidence” means here

- **Deterministic**: commands run, logs or API responses captured, version/commit noted.
- **Bounded**: you describe what was connected (transport type, rough role of the gateway) without claiming performance MEL did not measure.
- **Privacy-safe**: no precise coordinates, no API keys, no tenant secrets, no unredacted serial payloads if they contain PII.

## Before you share anything

1. Run **Privacy** in the operator console (and related diagnostics) if your build exposes them; see [Diagnostics](../ops/diagnostics.md); redact exports.
2. Prefer **relative** location language (“urban rooftop”, “indoor desk”) over coordinates.
3. If you paste config, remove passwords, broker URLs with credentials, and TLS material.

## Useful artifacts to attach

- Output of `./bin/mel doctor --config <path>` (redacted).
- Relevant lines from `./bin/mel status` or JSON from `/api/v1/status` (redacted).
- One or two screenshots following [Showcase and screenshots](SHOWCASE_AND_SCREENSHOTS.md) — with captions for **live vs historical vs demo-seeded**.

## How to report

- **Regression or bug**: [Bug report](../../.github/ISSUE_TEMPLATE/bug_report.md) with repro steps.
- **Field narrative + learnings (non-bug)**: [Field report](../../.github/ISSUE_TEMPLATE/field_report.md).
- **Device/gateway + transport combo**: [Hardware compatibility](../../.github/ISSUE_TEMPLATE/hardware_compatibility.md).

## What maintainers can and cannot promise

- Triaged issues may drive code or docs changes; field reports do **not** imply vendor certification or “supported hardware” lists unless the repo adds an explicitly bounded compatibility doc.
- BLE and HTTP ingest remain **unsupported** until implementation and tests say otherwise ([README transport matrix](../../README.md)).
