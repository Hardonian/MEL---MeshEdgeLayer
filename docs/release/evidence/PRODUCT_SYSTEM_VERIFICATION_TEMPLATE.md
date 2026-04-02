# Product-System Verification Evidence Template

Date:
Release candidate:
Operator:

## Commands

- [ ] `make lint`
- [ ] `make test`
- [ ] `make build`
- [ ] `make product-verify`
- [ ] `make smoke`

## Evidence files

Attach command outputs and any supporting JSON/log artifacts.

## Deviations and caveats

Document any missing fixture/environment prerequisite (for example `.tmp/smoke.json`) and explicitly mark affected gates as not passed.

## Capability boundary check

- [ ] BLE ingest still labeled unsupported
- [ ] HTTP ingest still labeled unsupported
- [ ] No MEL RF routing execution claim introduced
- [ ] Advisory vs execution semantics preserved
