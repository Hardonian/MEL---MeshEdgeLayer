# MEL Simulation and Risk Model

## Overview

The MEL Simulation Layer provides predictive, evidence-based what-if analysis for control actions before execution. It answers the operator's question: "What will happen if I take this action?"

### Core Principles

- **Deterministic**: Same inputs always produce same outputs
- **Bounded**: Does not attempt to simulate everything - focuses on actionable predictions
- **Evidence-based**: Uses real MEL state, not speculative modeling
- **Explainable**: Every conclusion includes reasoning and evidence references
- **Honest about limitations**: Clearly states what MEL does NOT simulate

### What This Enables

- Evaluate potential actions BEFORE execution
- Estimate likely outcomes and risks
- Detect conflicts and unsafe conditions
- Preview policy decisions
- Provide operator guidance (safe-to-act signals)
- Strengthen operator trust through transparency

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Simulation Engine                             │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │   Policy     │  │    Risk      │  │   Blast Radius       │  │
│  │   Preview    │  │   Scoring    │  │   Prediction         │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │   Conflict   │  │   Outcome    │  │   Safe-to-Act        │  │
│  │  Detection   │  │  Branching   │  │   Decision           │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SimulationResult                               │
│  (Predicted Outcome + Risk Assessment + Safe-to-Act Guidance)    │
└─────────────────────────────────────────────────────────────────┘
```

## Input Surface

### Available Signals

The simulation engine uses these real MEL signals:

| Signal | Source | Usage |
|--------|--------|-------|
| Incidents | `db.TransportAlerts` | Active issues affecting targets |
| Diagnostics | `diagnostics.RunAllChecks()` | Health check results |
| Transport Health | `status.TransportIntelligence` | Real-time transport state |
| Mesh Topology | `status.MeshDrilldown` | Node/segment relationships |
| Action History | `db.ControlActions()` | Historical success patterns |
| Policy Config | `config.Control` | Admission rules and constraints |
| Freshness Signals | `selfobs.FreshnessTracker` | Data age and confidence |
| Control Reality | `control.ActionReality` | Action type characteristics |

### Signal Quality

The engine explicitly tracks:
- **Freshness**: How old is the input data?
- **Confidence**: How certain is the signal?
- **Completeness**: Are there gaps in the data?
- **Provenance**: Where did the signal originate?

## Simulation Model

### Deterministic Evaluation Pipeline

For each candidate action, the engine evaluates:

1. **Preconditions**: Are required conditions met?
   - Transport existence
   - Configuration validity
   - Database connectivity

2. **State Inspection**: What is the current system state?
   - Transport runtime state
   - Mesh health score
   - Active alerts

3. **Component Identification**: What would be affected?
   - Direct target (transport/segment/node)
   - Cascading dependencies
   - Shared segments

4. **Transition Prediction**: What state changes would occur?
   - Expected end state
   - Side effects
   - Rollback capability

### Outputs

- **Predicted Outcome**: Success probability, expected state, side effects
- **Risk Assessment**: Risk level, factors, mitigations
- **Policy Preview**: Admission result, denials, prerequisites
- **Blast Radius**: Impact scope and classification
- **Conflicts**: Detected issues with resolutions
- **Outcome Branches**: Best/expected/worst scenarios
- **Safe-to-Act**: Clear operator guidance

## Risk Scoring

### Risk Levels

| Level | Score | Automation | Description |
|-------|-------|------------|-------------|
| None | 0.0-0.1 | Allowed | No significant risk |
| Low | 0.1-0.3 | Allowed | Minor, well-understood risks |
| Medium | 0.3-0.6 | Conditional | Requires acknowledgment |
| High | 0.6-0.8 | Blocked | Significant risk, manual only |
| Critical | 0.8-1.0 | Blocked | Severe risk, strongly discouraged |

### Contributing Factors

Risk scoring considers:

1. **Action Type Risk**: Inherent risk from `ActionReality` matrix
2. **Transport State**: Current health and stability
3. **Mesh Context**: Correlated failures, degraded segments
4. **Historical Patterns**: Success/failure rates from action history
5. **Dependency Confidence**: Episode correlation, attribution strength
6. **Data Freshness**: Age of supporting evidence

### Transparency

Every risk score includes:
- Contributing factors with weights
- Evidence references
- Confidence intervals
- Limitations acknowledged

## Policy Preview

### Admission Results

- **allowed**: Action would be admitted for execution
- **denied**: Action would be denied with specific reason
- **advisory**: Action would be advisory-only (not auto-executed)
- **conditional**: Action requires prerequisites to be met
- **unknown**: Insufficient information to determine

### Safety Checks

The preview evaluates all safety checks without side effects:

- Evidence persistence (transient vs persistent)
- Confidence threshold
- Policy allows action type
- Cooldown satisfied
- Budget available
- No conflicting active actions
- Reversibility
- Blast radius known
- Actuator exists
- Alternate path exists
- Attribution strength (for suppression)

### Mode-Aware Results

Results adapt to control mode:
- **disabled**: All actions would be mode-denied
- **advisory**: All actions would be advisory-only
- **guarded_auto**: Full evaluation with safety checks

## Blast Radius Prediction

### Impact Classification

| Class | Score | Description |
|-------|-------|-------------|
| Isolated | 0-0.25 | Single transport/node |
| Segmented | 0.25-0.75 | Multiple related transports/nodes |
| Systemic | 0.75-1.0 | Mesh-wide or control-path impact |

### Action Type Profiles

Each action type has an inherent impact profile:

| Action | Radius | Scope | Recovery |
|--------|--------|-------|----------|
| restart_transport | 0.30 | transport | 15s |
| backoff_increase | 0.15 | transport | 5s |
| deprioritize | 0.40 | mesh | 30s |
| suppress | 0.20 | segment | 10s |
| health_recheck | 0.05 | local | immediate |

### Cascading Effects

The engine predicts potential cascading:
- Correlated failure propagation
- Segment degradation spread
- Topic-based impact grouping

## Conflict Detection

### Conflict Types

1. **Active Conflicts**: Incompatible with running actions
2. **Duplicate Actions**: Same action recently taken
3. **Sequence Violations**: Violates recovery ordering
4. **Stale Data**: Acting on outdated information
5. **Cooldown Violations**: Within cooldown window
6. **Dependency Conflicts**: Prerequisites not met
7. **Safety Conflicts**: Unsafe given current alerts
8. **Self-Degradation**: MEL health issues present

### Severity Levels

- **critical**: Block action, require resolution
- **major**: Warn strongly, suggest alternatives
- **moderate**: Note in guidance
- **minor**: Informational

## Outcome Branching

### Scenario Types

- **best_case**: Optimistic but realistic outcome
- **expected_case**: Most likely based on evidence
- **worst_case**: Pessimistic but possible outcome

### Evidence-Based Probabilities

Probabilities are derived from:
- Action type success rates
- Mesh health context
- Historical patterns
- Current degradation state

### Assumptions

Each branch lists key assumptions:
- What must be true for this outcome
- What could change the outcome
- Confidence in assumptions

## Safe-to-Act Decision

### Decision Categories

- **SAFE_TO_ACT**: All checks passed, proceed with confidence
- **SAFE_AFTER_CONDITION**: Safe if prerequisites are met
- **NOT_SAFE**: Should not proceed, risks too high
- **INSUFFICIENT_DATA**: More information needed

### Decision Inputs

The evaluator considers:
- Risk assessment
- Conflict report
- Policy preview
- Blast radius prediction
- Outcome predictions

### Operator Guidance

For each decision, provides:
- Primary reason
- Supporting details
- Missing prerequisites
- Suggested next steps
- Alternative actions

## Limitations and Guarantees

### What MEL Simulates

✅ **Simulated**:
- Transport-level state transitions
- Policy admission logic
- Historical action patterns
- Mesh topology relationships
- Alert and degradation correlations
- Data freshness and confidence

### What MEL Does NOT Simulate

❌ **Not Simulated**:
- External network conditions (outside MEL's observation)
- Hardware-level failures without transport symptoms
- Meshtastic protocol-level behaviors beyond transport state
- Future events not indicated by current evidence
- Other operator actions concurrent with simulation

### Guarantees

1. **Determinism**: Same inputs → same outputs
2. **Bounded Complexity**: Completes in predictable time
3. **Evidence Grounding**: Predictions based on observable state
4. **Transparency**: All conclusions explainable
5. **Honest Uncertainty**: Unknowns clearly identified

### Non-Guarantees

1. **Outcome Certainty**: Real outcomes may differ
2. **Exhaustive Coverage**: Not all failure modes modeled
3. **Future Prediction**: Only extrapolates from current state

## Operator Usage

### CLI Command

```bash
# Simulate a transport restart
mel simulate action restart_transport --transport mqtt-primary --config mel.json

# Output as JSON for automation
mel simulate action backoff_increase --transport serial --format json

# Save results to file
mel simulate action health_recheck --transport mqtt --output simulation.json
```

### Interpreting Results

1. Check **Safe-to-Act Decision** first
2. Review **Risk Assessment** for concerns
3. Examine **Conflicts** for blocking issues
4. Consider **Outcome Branches** for planning
5. Note **Missing Prerequisites** if conditional

### When to Use

- Before manual action execution
- When evaluating automation recommendations
- During incident response planning
- For operator training and familiarization

## API Reference

### SimulationInput

```go
type SimulationInput struct {
    CandidateAction   control.ControlAction
    MeshState         status.MeshDrilldown
    Diagnostics       diagnostics.DiagnosticReport
    ActionHistory     []db.ControlActionRecord
    CurrentTime       time.Time
    Configuration     config.Config
}
```

### SimulationResult

```go
type SimulationResult struct {
    PredictedOutcome  PredictedOutcome
    RiskAssessment    RiskAssessment
    PolicyPreview     PolicyPreview
    BlastRadius       BlastRadiusPrediction
    Conflicts         ConflictReport
    OutcomeBranches   []OutcomeBranch
    SafeToAct         SafeToActDecision
    Metadata          SimulationMetadata
}
```

### Key Methods

```go
// Create engine
engine := simulation.NewEngine(cfg, db)

// Run simulation
result, err := engine.Simulate(input)

// Simple API
result, err := simulation.SimulateSimple(cfg, db, actionType, transportName)

// Batch simulation
results, err := engine.SimulateBatch(inputs)
```

## Integration Points

### Recommendation Enrichment

```go
enriched := simulation.EnrichRecommendation(rec, result)
// Adds risk assessment, safety guidance to recommendations
```

### Control Decision Enhancement

```go
enriched := simulation.EnrichControlDecision(decision, result)
// Adds predictive outcomes, alternatives to decisions
```

### Middleware

```go
middleware := simulation.NewMiddleware(engine)
wrapped := middleware.WrapEvaluation(evaluateFunc)
```

## Testing

### Determinism Tests

Verify same inputs produce same outputs across runs.

### Edge Cases

- Nil/empty inputs
- Missing database
- Stale data
- Conflicting states

### Integration Tests

- End-to-end simulation pipeline
- CLI command execution
- API serialization

## Version History

- **v1.0**: Initial simulation layer with deterministic evaluation, risk scoring, policy preview, blast radius prediction, conflict detection, outcome branching, and safe-to-act decision layer.
