# MEL Architecture & Trust-Boundary Overview (Repo-OS Supplement)

This is a practical boundary map for implementation and review; pair with detailed architecture docs under `docs/architecture/`.

## Core boundary map
1. **Transport ingest boundary**
   - Inputs: direct (serial/TCP), MQTT.
   - Risk: config-present mistaken as active ingest.
   - Required truth: runtime evidence + persisted ingest before success claims.

2. **Evidence persistence boundary**
   - Inputs: decoded packets/events/actions.
   - Risk: dropped/dead-lettered data masked by aggregate status.
   - Required truth: dead letters and parse failures remain visible/auditable.

3. **State derivation boundary**
   - Inputs: persisted evidence + freshness windows + heuristics.
   - Risk: stale/historical/imported data rendered as live.
   - Required truth: typed state semantics for live/stale/historical/imported/partial.

4. **Operator control boundary**
   - Inputs: action requests, approvals, execution attempts, outcomes.
   - Risk: lifecycle ambiguity or hidden bypass.
   - Required truth: explicit state transitions + approval separation + audit trail.

5. **Presentation boundary (API/UI/CLI)**
   - Inputs: derived state and evidence metadata.
   - Risk: wording stronger than backend certainty.
   - Required truth: presentation cannot exceed underlying evidence certainty.

6. **Security boundary**
   - Inputs: actor identity/capabilities, runtime config, control endpoints.
   - Risk: fail-open authorization or broad trust assumptions.
   - Required truth: least privilege, explicit denial semantics, auditable auth decisions.

## Architectural invariants
- Deterministic outputs over “smart-looking” hidden inference.
- Typed contracts over string/prose inference.
- Explicit degraded-state representation over omission.
- Claim strength must be proportional to available evidence.

## Review trigger
If a change crosses any boundary above, run:
- operator truth audit,
- transport truth audit,
- security/trust-boundary audit,
- trusted control governance checklist (if control path touched).
