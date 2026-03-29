# Incident → Learning Loop

MEL compounds value when every meaningful incident improves future response quality.

## Trigger events
- Incident declarations.
- Dead-letter spikes / ingest failures.
- Control action failures or unexpected outcomes.
- Operator corrections of false positives/negatives.

## Mandatory loop outputs
For each significant trigger, produce at least one of each category where applicable:

1. **Test artifact**
   - Unit/integration/contract/UI test reproducing the failure mode.
2. **Runbook update**
   - Triage/remediation steps with evidence requirements.
3. **Rule/heuristic update**
   - Better detection, prioritization, or recommendation logic.
4. **Evidence schema improvement**
   - Additional typed fields to preserve uncertainty/provenance/state.
5. **Operator surface improvement**
   - Clearer drilldown, degraded signaling, or action outcome visibility.

## PR requirements for incident-derived work
- Link incident identifier/evidence source.
- State prior failure mode and why new checks prevent recurrence.
- Declare residual risk if behavior remains partial.

## Monthly compounding review (recommended)
- Top recurring incident signatures.
- False-positive/false-negative patterns.
- Action outcomes and approval latency.
- Dead-letter and ingest reliability trends.
- Resulting test/runbook/rule updates shipped.
