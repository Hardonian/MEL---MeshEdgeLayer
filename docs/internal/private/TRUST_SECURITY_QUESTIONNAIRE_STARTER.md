# Trust and security questionnaire starter (internal)

Use in procurement conversations. **Answer from shipped behavior and docs**; where unknown, say unknown.

## Data and residency

1. Where does operational data live by default? (local SQLite; see deployment docs.)
2. What leaves the host when MQTT, backups, or remote UI are enabled?
3. Is at-rest encryption built into MEL? (See [SECURITY.md](../../../SECURITY.md) — current posture.)

## Authentication and exposure

4. Default bind: localhost or all interfaces?
5. What auth models exist? (See [docs/ops/auth-rbac-model.md](../../ops/auth-rbac-model.md).)
6. How are secrets handled (config file permissions, env)?

## Telemetry

7. Is there hidden phone-home telemetry? (Defaults must be explicit; see [docs/privacy/posture.md](../../privacy/posture.md).)

## Control plane

8. How are control actions governed (lifecycle states)? ([docs/ops/CONTROL_PLANE_TRUST.md](../../ops/CONTROL_PLANE_TRUST.md).)
9. Does MEL transmit RF or execute radio admin on behalf of operators? (Current product boundary: no mesh-stack routing/transmit feature as described in root README.)

## Integrity and audit

10. What audit trails exist for actions and ingest?
11. How are exports redacted by default? ([docs/runbooks/proofpack-export.md](../../runbooks/proofpack-export.md), support bundle guides.)

## Incident response (project)

12. How are security issues reported? ([SECURITY.md](../../../SECURITY.md).)
13. Is there a commercial SLA? (Community: see [SUPPORT.md](../../../SUPPORT.md) and [SUPPORT_POSTURE_TRUTH.md](./SUPPORT_POSTURE_TRUTH.md).)
