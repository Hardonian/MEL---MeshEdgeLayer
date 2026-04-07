# Pilot readiness checklist (internal)

Use this for **design partners** and **time-boxed pilots**. It is an internal operating aid, not a customer contract.

## Preconditions

- [ ] Build path documented: `make build` (or agreed artifact) produces `./bin/mel`.
- [ ] Config path, bind address, and auth posture agreed (localhost vs reviewed remote exposure).
- [ ] Supported transports for the pilot are **explicitly** one of: serial, TCP, MQTT — not BLE/HTTP unless support has changed in-tree.
- [ ] Operators have read [docs/product/HONESTY_AND_BOUNDARIES.md](../../product/HONESTY_AND_BOUNDARIES.md) or equivalent summary.

## Week 0 — time to first proof

- [ ] `mel doctor` run captured; warnings understood (no transports vs misconfig).
- [ ] Either live ingest **or** fixture path: `make demo-seed` + `demo_sandbox/mel.demo.json` — labeled correctly as live vs simulated.
- [ ] Incident queue and at least one diagnostic surface exercised.

## Week 1–2 — operational memory

- [ ] At least one incident lifecycle walked (create/triage/close or your local workflow).
- [ ] One export or support-bundle style handoff generated (redacted as appropriate).
- [ ] Degraded or stale state observed **without** forcing narrative; UI/docs match [docs/repo-os/terminology.md](../../repo-os/terminology.md).

## Exit criteria (suggested)

- [ ] Three incident-style cycles or equivalent operational exercises completed.
- [ ] List of gaps classified Maintenance / Leverage / Moat ([docs/repo-os/change-classification.md](../../repo-os/change-classification.md)).
- [ ] Decision: continue, extend pilot, or stop — with honest capability boundaries.

## References

- [DESIGN_PARTNER_PLAN.md](./DESIGN_PARTNER_PLAN.md)
- [docs/founder-demo-script.md](../../founder-demo-script.md)
- [docs/runbooks/launch-and-demo.md](../../runbooks/launch-and-demo.md)
