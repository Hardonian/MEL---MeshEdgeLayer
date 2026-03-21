# MEL CLI: Control Plane Operations

The `mel` CLI provides operators with a direct, high-fidelity interface to the MEL Control Plane.

---

## Direct Action (`mel control exec`)

Operators can manually execute any action that MEL's Reality Matrix supports. These actions bypass the "Guarded Automation" confidence check but still record a full audit trail.

```bash
# Force-restart a specific transport
mel control exec --action restart_transport --target mqtt-primary

# Manually trigger a health recheck
mel control exec --action trigger_health_recheck
```

---

## Status and History (`mel control status`)

Display the current control mode, active policies, and recent decision history.

```bash
mel control status --verbose
```

**Key Information:**

- `Mode`: One of `disabled`, `advisory`, `guarded_auto`.
- `Recent Denials`: See exactly why the automated layer blocked an action.
- `Active Actions`: View any actions that have not yet expired or reached a terminal state.

---

## Override & Freeze (`mel control freeze`)

Sometimes a physical transport is unstable enough that even automated restarts might cause more harm. Operators can "freeze" specific targets to prevent any automated actions from being proposed.

```bash
# Prevent any automated control of the 'serial-local' transport
mel control freeze --target serial-local --reason "manual debugging in progress"

# Thaw the transport back to normal operations
mel control thaw --target serial-local
```

---

## Audit Logs (`mel events --category control`)

All control decisions—whether allowed, denied, or manual—are stored in the central audit ledger.

```bash
# View the last 10 control decisions
mel events --category control --limit 10
```

---

## TUI Experience

The `mel panel` command launches an interactive terminal dashboard.

![MEL TUI Experience](/c:/Users/scott/.gemini/antigravity/brain/3857245b-4abd-4d41-9b8b-41da0a674b43/mel_tui_mockup_retro_future_terminal_1774057747796.png)

*MEL — Truthful Local-First Mesh Observability.*
