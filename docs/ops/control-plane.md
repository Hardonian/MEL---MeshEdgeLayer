# MEL Control Plane

The MEL control plane evaluates mesh health conditions and proposes or executes remediation actions based on configured safety policies.

## Control Modes

| Mode | Description | Actions Executed |
|------|-------------|------------------|
| `disabled` | Control evaluation runs but no actions are allowed | Never |
| `advisory` (default) | Actions are evaluated and logged but NOT executed | Never |
| `guarded_auto` | Actions are evaluated and executed only if all safety checks pass | When all checks pass |

### Mode Selection Guide

- **disabled**: Use when investigating control behavior or during maintenance windows where no automated intervention should occur
- **advisory** (recommended default): Use to observe what MEL would do without risk; review logs to tune confidence thresholds and allowed actions before enabling automation
- **guarded_auto**: Use only after validation in advisory mode; requires all safety checks to pass for execution

## Operator approval and execution (truth)

When an action requires approval (`execution_mode: approval_required`), it stays in
`pending_approval` until an operator approves. MEL uses a **single-approver** model
(`required_approvals: 1`); there is no quorum.

- **`require_approval_for_action_types`**: explicit list match forces approval.
- **`require_approval_for_high_blast_radius`**: when enabled, stored classes `mesh` and
  `global` force approval. Other classes are **not** auto-promoted by this flag.

**Approval is not execution.** After approve, the action moves to `pending` and still
needs the **control executor** (`mel serve`). HTTP approve does not drain the full
queue; `mel action approve` may dequeue **one** slot in that CLI process when the
bounded queue has work.

## Configuration

Control settings are nested under the `control` key in MEL configuration:

```json
{
  "control": {
    "mode": "advisory",
    "emergency_disable": false,
    "allowed_actions": [
      "restart_transport",
      "resubscribe_transport",
      "backoff_increase",
      "backoff_reset",
      "temporarily_deprioritize_transport",
      "temporarily_suppress_noisy_source",
      "clear_suppression",
      "trigger_health_recheck"
    ],
    "max_actions_per_window": 8,
    "cooldown_per_target_seconds": 300,
    "require_min_confidence": 0.75,
    "allow_mesh_level_actions": false,
    "allow_transport_restart": false,
    "allow_source_suppression": false,
    "action_window_seconds": 900,
    "restart_cap_per_window": 2
  }
}
```

### Configuration Reference

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `mode` | string | `"advisory"` | Control mode: `"disabled"`, `"advisory"`, or `"guarded_auto"` |
| `emergency_disable` | bool | `false` | Master switch to disable all control actions |
| `allowed_actions` | []string | See above | Which action types are permitted for evaluation |
| `max_actions_per_window` | int | `8` | Maximum actions across all targets per time window |
| `cooldown_per_target_seconds` | int | `300` | Minimum seconds between actions on the same target |
| `require_min_confidence` | float | `0.75` | Minimum confidence threshold (0.5-1.0) |
| `allow_mesh_level_actions` | bool | `false` | Allow actions affecting mesh routing |
| `allow_transport_restart` | bool | `false` | Allow transport restart actions |
| `allow_source_suppression` | bool | `false` | Allow source suppression actions |
| `action_window_seconds` | int | `900` | Duration of action budget window |
| `restart_cap_per_window` | int | `2` | Maximum transport restarts per window |

## Action Types

| Action | Description | Typical Trigger |
|--------|-------------|-----------------|
| `restart_transport` | Restart a failing transport | Retry threshold exceeded |
| `resubscribe_transport` | Re-subscribe to MQTT topics | Subscription failure cluster |
| `backoff_increase` | Increase reconnect backoff | Failure storm/saturation detected |
| `backoff_reset` | Reset reconnect backoff | Recovery detected, normalize backoff |
| `temporarily_deprioritize_transport` | Deprioritize degraded transport | Mesh relative degradation detected |
| `temporarily_suppress_noisy_source` | Suppress noisy node | High-confidence noisy source attribution |
| `clear_suppression` | Clear suppression state | Suppression timeout or recovery |
| `trigger_health_recheck` | Trigger health check | Critical degraded segment |

## Safety Checks

For an action to execute in `guarded_auto` mode, **ALL** of the following checks must pass:

| Check | Description |
|-------|-------------|
| `evidence_pass` | Sufficient persisted evidence exists in anomaly history (≥2 time buckets) |
| `confidence_pass` | Confidence score ≥ `require_min_confidence` |
| `policy_pass` | Action type is in `allowed_actions` |
| `cooldown_pass` | Cooldown window elapsed since last action on target |
| `budget_pass` | Action budget not exceeded for current window |
| `conflict_pass` | No conflicting active action in flight |
| `reversibility_pass` | Action is reversible or has expiry |
| `blast_radius_known` | Blast radius is bounded and classified |
| `actuator_exists` | MEL ships a working actuator for this action |
| `alternate_path_exists` | For deprioritize actions, healthy alternate exists |

## Denial Reasons

When an action is denied, the control plane reports one of the following codes:

| Code | Meaning | Resolution |
|------|---------|------------|
| `mode` | Control mode is disabled/advisory | Change mode to `guarded_auto` |
| `policy` | Action not in `allowed_actions` | Add to allowed_actions or verify intent |
| `cooldown` | Still in cooldown window | Wait for cooldown to elapse |
| `budget` | Action budget exceeded | Wait for window to reset or increase budget |
| `low_confidence` | Below confidence threshold | Tune thresholds or review evidence quality |
| `transient` | Evidence insufficient | Wait for more persisted anomaly data |
| `missing_actuator` | No actuator shipped for this action | Action is advisory-only (see Reality Matrix) |
| `irreversible` | Action not reversible | Review blast radius classification |
| `conflict` | Conflicting action in flight | Wait for existing action to complete |
| `unknown_blast_radius` | Blast radius unknown | Action lacks safety classification |
| `no_alternate_path` | No healthy alternate for deprioritize | Ensure redundant transport paths |
| `attribution_weak` | Cannot strongly attribute noisy source | Wait for stronger signal |
| `override` | Operator override active | Check transport-level overrides |

## Lifecycle States

Actions progress through the following states:

| State | Description |
|-------|-------------|
| `pending` | Action queued for evaluation |
| `running` | Action currently executing |
| `completed` | Action finished (success or failure) |

## Closure States

When actions complete or expire, they receive a closure state:

| State | Description |
|-------|-------------|
| `recovered_and_closed` | Action succeeded and properly closed |
| `expired_and_reverted` | Action expired (timeout reached) |
| `superseded` | Replaced by newer action |
| `canceled_by_operator` | Operator manually canceled |

## Reality Matrix

This matrix documents which actions are **actually executable** versus advisory-only:

| Action | Executable | Advisory Only | Reason |
|--------|------------|---------------|--------|
| `restart_transport` | ✅ Yes | | Has working actuator; bounded blast radius |
| `resubscribe_transport` | ✅ Yes | | Has working actuator; local transport scope |
| `backoff_increase` | ✅ Yes | | Has working actuator; reversible |
| `backoff_reset` | ✅ Yes | | Has working actuator; reversible |
| `trigger_health_recheck` | ✅ Yes | | Has working actuator; local process scope |
| `temporarily_deprioritize_transport` | | ✅ Yes | No routing selector actuator in this build |
| `temporarily_suppress_noisy_source` | | ✅ Yes | No suppression actuator or metrics path |
| `clear_suppression` | | ✅ Yes | No suppression actuator in this build |

### Blast Radius Classification

| Class | Scope | Safe for Guarded Auto |
|-------|-------|----------------------|
| `local_transport` | Single transport instance | Yes |
| `local_process` | Local MEL process only | Yes |
| `unknown` | Not classified | No |

## Operator Guidance

### When to Use Each Mode

**Advisory Mode (Recommended Default)**
- New deployments observing mesh behavior
- Tuning confidence thresholds
- Validating allowed action lists
- Learning normal vs. abnormal patterns
- Before any production automation

**Guarded Auto Mode**
- Only after extended advisory observation
- When action patterns are well-understood
- With appropriate monitoring and alerting
- With rollback plan for emergency_disable

**Disabled Mode**
- Maintenance windows
- Debugging control behavior
- Investigations requiring static state

### Evaluating Control Suggestions in Advisory Mode

Review the control explanation output:

```json
{
  "mode": "advisory",
  "active_actions": [],
  "recent_actions": [],
  "denied_actions": [...],
  "policy_summary": {...},
  "reality_matrix": [...],
  "reasons_for_denial": [...]
}
```

Key fields to monitor:
- `denied_actions`: Actions that would execute if mode were `guarded_auto`
- `reasons_for_denial`: Why actions were blocked (tune config to address)
- `reality_matrix`: Which actions could actually execute vs. advisory-only

### Safety Implications of Guarded Auto

Before enabling `guarded_auto`:

1. **Verify actuator availability** - Check reality matrix for actions you expect to execute
2. **Review blast radius** - Understand scope of each enabled action
3. **Set conservative budgets** - Start with low `max_actions_per_window` and `restart_cap_per_window`
4. **Configure cooldowns** - Ensure adequate `cooldown_per_target_seconds` (minimum 30s)
5. **Test emergency disable** - Verify `emergency_disable` stops all automation
6. **Monitor actively** - Watch for unexpected action patterns

### Interpreting Denial Reasons

| If you see... | Check... |
|---------------|----------|
| Frequent `low_confidence` | Lower `require_min_confidence` or improve evidence quality |
| Frequent `transient` | Allow more time for anomaly persistence |
| Frequent `cooldown` | Increase `cooldown_per_target_seconds` if too aggressive |
| Frequent `budget` | Increase `max_actions_per_window` if legitimate |
| `missing_actuator` | Action is advisory-only; do not expect execution |
| `no_alternate_path` | Configure redundant transports before deprioritization |

## Transport-Level Overrides

Individual transports can override control behavior via these flags:

| Override | Field | Effect |
|----------|-------|--------|
| `manual_only` | `manual_only: true` | Transport excluded from all automation; operator manages directly |
| `suppress_auto_actions` | `suppress_auto_actions: true` | No automatic actions on this transport |
| `freeze_routing` | `freeze_routing: true` | Blocks `temporarily_deprioritize_transport` actions |

Example transport configuration with overrides:

```json
{
  "transports": [
    {
      "name": "critical-uplink",
      "type": "mqtt",
      "enabled": true,
      "manual_only": true,
      "endpoint": "mqtt.example.com:8883"
    },
    {
      "name": "backup-radio",
      "type": "serial",
      "enabled": true,
      "suppress_auto_actions": true,
      "serial_device": "/dev/serial/by-id/..."
    }
  ]
}
```

Override evaluation order:
1. `emergency_disable` (global) - stops all automation
2. Transport `manual_only` - excludes transport from consideration
3. Transport `suppress_auto_actions` - blocks auto actions on transport
4. Transport `freeze_routing` - blocks routing changes only
5. Standard safety checks

## Action Results

After execution attempts, actions receive result codes:

| Result | Meaning |
|--------|---------|
| `executed_successfully` | Action completed as intended |
| `executed_noop` | Action ran but had no effect (idempotent) |
| `denied_by_policy` | Blocked by policy check |
| `denied_by_cooldown` | Blocked by cooldown window |
| `failed_transient` | Failed but may succeed on retry |
| `failed_terminal` | Failed permanently |
| `expired` | Action expired before completion |
