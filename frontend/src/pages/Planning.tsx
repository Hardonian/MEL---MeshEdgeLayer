import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { EmptyState } from '@/components/ui/EmptyState'

interface BestNextMove {
  title: string
  summary_lines: string[]
  evidence_anchors?: string[]
  uncertainty_notes?: string[]
  wait_observe_rationale?: string
  would_validate_with?: string[]
  primary_verdict: string
  recommendation_key?: string
  evidence_classification: string
}

interface PlanningBundle {
  evidence_model: string
  graph_hash?: string
  mesh_assessment_id?: string
  transport_connected: boolean
  topology_enabled: boolean
  best_next_move?: BestNextMove
  wait_versus_expand_hint?: string
  resilience: {
    resilience_score: number
    redundancy_score: number
    partition_risk_score: number
    fragility_explanation: string[]
    next_best_move_summary: string
    confidence: { level: string; score: number; missing_inputs?: string[] }
  }
  node_profiles: Array<{
    node_num: number
    short_name?: string
    critical_node_score: number
    partition_risk_score: number
    resilience_score: number
    spof_class: string
    recovery_priority: number
  }>
  ranked_next_plans: Array<{
    rank: number
    id: string
    title: string
    verdict: string
    benefit_band: string
    lines: string[]
  }>
  playbooks: Array<{
    class: string
    title: string
    summary: string
    minimum_viable_milestone: string
    steps: Array<{ order: number; title: string; rationale: string; observe_hours: number }>
    limits: string[]
  }>
  limits: string[]
  computed_at: string
}

interface AdvisoryAlertRow {
  id: string
  severity: string
  reason: string
  summary: string
  cluster_key?: string
  contributing_reasons?: string[]
  trigger_condition?: string
}

interface PlanComparison {
  compared_ids: string[]
  ranked_by_upside: Array<{
    id: string
    label: string
    upside: string
    dimensions: { low_regret_score: number; assumption_fragility_score: number }
    narrative_lines?: string[]
  }>
  ranked_by_low_regret?: Array<{ id: string; label: string }>
  low_regret_pick_id?: string
  best_upside_pick_id?: string
  best_resilience_pick_id?: string
  best_diagnostic_pick_id?: string
  wait_observe_option_id?: string
  ranking_could_change_if?: string[]
  summary_lines: string[]
  evidence_classification: string
}

interface PlanExecution {
  execution_id: string
  plan_id: string
  status: string
  started_at: string
  observation_horizon_hours: number
}

interface EvidenceSignal {
  id: string
  message: string
}

function includesAny(text: string, patterns: string[]): boolean {
  return patterns.some((p) => text.includes(p))
}

function deriveEvidenceSignals(bundle: PlanningBundle, advisories: AdvisoryAlertRow[]): EvidenceSignal[] {
  const bn = bundle.best_next_move
  const source = [bundle.evidence_model, ...(bundle.limits ?? []), ...(bn?.uncertainty_notes ?? [])]
    .join(' ')
    .toLowerCase()

  const signals: EvidenceSignal[] = []

  if (includesAny(source, ['baseline is missing', 'no baseline mesh assessment id', 'baseline snapshot was pruned'])) {
    signals.push({
      id: 'baseline-missing',
      message: 'Baseline evidence is missing or unavailable; before/after deltas are directional only.',
    })
  }

  if (
    bundle.mesh_assessment_id &&
    includesAny(source, ['confounded', 'same live compute', 'same mesh_assessment_id', 'non-causal'])
  ) {
    signals.push({
      id: 'confounded',
      message:
        'Current and baseline references are potentially confounded (same or concurrent assessment context), so causality is not established.',
    })
  }

  if (includesAny(source, ['directional', 'inconclusive', 'no strong directional signal'])) {
    signals.push({
      id: 'directional',
      message: 'Validation remains directional/inconclusive and does not prove propagation or RF behavior.',
    })
  }

  if (includesAny(source, ['graph hash changed', 'topology drift', 'graph-shape concerns', 'concurrent operational changes'])) {
    signals.push({
      id: 'topology-drift',
      message: 'Graph/topology drift is present, so trend comparison can be confounded by concurrent changes.',
    })
  }

  if (
    advisories.length === 0 &&
    includesAny(source, ['directional', 'confounded', 'inconclusive', 'baseline'])
  ) {
    signals.push({
      id: 'empty-advisories-not-clear',
      message: 'No advisory rows here is not an all-clear; evidence uncertainty still applies.',
    })
  }

  const limitedConfidence =
    bundle.resilience.confidence.level !== 'high' ||
    (bn?.uncertainty_notes?.length ?? 0) > 0 ||
    includesAny((bn?.evidence_classification ?? '').toLowerCase(), ['directional', 'confounded', 'inconclusive'])

  if (limitedConfidence) {
    signals.push({
      id: 'limited-confidence',
      message: 'Recommendations exist, but confidence is limited by missing or non-independent evidence.',
    })
  }

  return signals
}

export function Planning() {
  const [bundle, setBundle] = useState<PlanningBundle | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [advisories, setAdvisories] = useState<AdvisoryAlertRow[]>([])
  const [compareIds, setCompareIds] = useState('')
  const [comparison, setComparison] = useState<PlanComparison | null>(null)
  const [compareErr, setCompareErr] = useState<string | null>(null)
  const [compareLoading, setCompareLoading] = useState(false)
  const [execPlanId, setExecPlanId] = useState('')
  const [executions, setExecutions] = useState<PlanExecution[]>([])
  const [execErr, setExecErr] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const [bRes, aRes] = await Promise.all([
          fetch('/api/v1/planning/bundle'),
          fetch('/api/v1/planning/advisory-alerts'),
        ])
        if (!bRes.ok) throw new Error(`bundle HTTP ${bRes.status}`)
        const data = (await bRes.json()) as PlanningBundle
        if (!cancelled) {
          setBundle(data)
          setError(null)
        }
        if (aRes.ok) {
          const adv = (await aRes.json()) as { alerts?: AdvisoryAlertRow[] }
          if (!cancelled) setAdvisories(adv.alerts ?? [])
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Failed to load planning bundle')
          setBundle(null)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  async function runCompare() {
    const ids = compareIds
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
    if (ids.length < 1) {
      setCompareErr('Enter one or more plan IDs (comma-separated).')
      return
    }
    setCompareLoading(true)
    setCompareErr(null)
    try {
      const res = await fetch('/api/v1/planning/compare', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ plan_ids: ids }),
      })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as PlanComparison
      setComparison(data)
    } catch (e) {
      setCompareErr(e instanceof Error ? e.message : 'Compare failed')
      setComparison(null)
    } finally {
      setCompareLoading(false)
    }
  }

  async function loadExecutions() {
    const id = execPlanId.trim()
    if (!id) {
      setExecErr('Enter a plan ID')
      return
    }
    setExecErr(null)
    try {
      const res = await fetch(`/api/v1/planning/executions?plan_id=${encodeURIComponent(id)}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as { executions: PlanExecution[] }
      setExecutions(data.executions ?? [])
    } catch (e) {
      setExecErr(e instanceof Error ? e.message : 'Failed to load executions')
      setExecutions([])
    }
  }

  if (loading) {
    return (
      <div className="p-6">
        <PageHeader title="Deployment planning" description="Loading…" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6 space-y-4">
        <PageHeader title="Deployment planning" description="Bounded what-if and resilience (topology evidence, not RF maps)" />
        <EmptyState title="Planning unavailable" description={error} />
      </div>
    )
  }

  if (!bundle) {
    return null
  }

  const bn = bundle.best_next_move
  const evidenceSignals = deriveEvidenceSignals(bundle, advisories)

  return (
    <div className="p-6 space-y-6 max-w-6xl mx-auto">
      <PageHeader
        title="Deployment planning"
        description="Next-step moves, resilience, and playbooks from observed mesh evidence. Not RF coverage simulation."
      />

      <Card className="p-4 border-amber-500/30 bg-amber-500/5">
        <p className="text-sm text-muted-foreground leading-relaxed" data-testid="planning-evidence-banner">
          {bundle.evidence_model}
        </p>
        <div className="flex flex-wrap gap-2 mt-3">
          <Badge variant={bundle.transport_connected ? 'default' : 'secondary'}>
            Transport: {bundle.transport_connected ? 'connected' : 'not connected'}
          </Badge>
          <Badge variant={bundle.topology_enabled ? 'default' : 'secondary'}>
            Topology: {bundle.topology_enabled ? 'enabled' : 'disabled'}
          </Badge>
          {bundle.mesh_assessment_id && (
            <Badge variant="outline">Assessment {bundle.mesh_assessment_id}</Badge>
          )}
        </div>
        {evidenceSignals.length > 0 && (
          <div className="mt-3 border-t border-amber-500/20 pt-3" data-testid="planning-evidence-signals">
            <p className="text-xs font-medium text-muted-foreground mb-2">Evidence posture (operator caution)</p>
            <ul className="space-y-2">
              {evidenceSignals.map((signal) => (
                <li key={signal.id} className="text-sm text-foreground flex items-start gap-2">
                  <Badge variant="warning" className="mt-0.5">
                    Caution
                  </Badge>
                  <span>{signal.message}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
        {bundle.wait_versus_expand_hint && (
          <p className="text-sm mt-3 border-t border-amber-500/20 pt-3">
            <span className="font-medium text-foreground">Wait vs expand: </span>
            {bundle.wait_versus_expand_hint}
          </p>
        )}
      </Card>

      {bn && (
        <Card className="p-4 border-border">
          <h3 className="font-semibold mb-2">Best next move (consolidated)</h3>
          <div className="flex flex-wrap gap-2 mb-2">
            <Badge variant="outline">{bn.primary_verdict}</Badge>
            <Badge variant="secondary">{bn.evidence_classification}</Badge>
            {bn.recommendation_key && (
              <span title="Key for outcome history">
                <Badge variant="outline">{bn.recommendation_key}</Badge>
              </span>
            )}
          </div>
          <p className="font-medium">{bn.title}</p>
          <ul className="mt-2 list-disc list-inside text-sm text-muted-foreground space-y-1">
            {(bn.summary_lines ?? []).map((line, i) => (
              <li key={i}>{line}</li>
            ))}
          </ul>
          {bn.wait_observe_rationale && (
            <p className="text-sm mt-3 text-amber-700 dark:text-amber-400">
              <span className="font-medium">Why wait/observe: </span>
              {bn.wait_observe_rationale}
            </p>
          )}
          {(bn.would_validate_with?.length ?? 0) > 0 && (
            <div className="mt-3 text-sm">
              <span className="text-muted-foreground">What would validate this after action:</span>
              <ul className="list-disc list-inside mt-1">
                {(bn.would_validate_with ?? []).map((x, i) => (
                  <li key={i}>{x}</li>
                ))}
              </ul>
            </div>
          )}
          {(bn.uncertainty_notes?.length ?? 0) > 0 && (
            <ul className="mt-2 text-xs text-muted-foreground list-disc list-inside" data-testid="planning-uncertainty-notes">
              {(bn.uncertainty_notes ?? []).map((x, i) => (
                <li key={i}>{x}</li>
              ))}
            </ul>
          )}
        </Card>
      )}

      <div className="grid gap-4 md:grid-cols-2">
        <Card className="p-4">
          <h3 className="font-semibold mb-2">Resilience summary</h3>
          <dl className="grid grid-cols-2 gap-2 text-sm">
            <dt className="text-muted-foreground">Resilience</dt>
            <dd>{bundle.resilience.resilience_score.toFixed(2)}</dd>
            <dt className="text-muted-foreground">Redundancy</dt>
            <dd>{bundle.resilience.redundancy_score.toFixed(2)}</dd>
            <dt className="text-muted-foreground">Partition risk</dt>
            <dd>{bundle.resilience.partition_risk_score.toFixed(2)}</dd>
            <dt className="text-muted-foreground">Confidence</dt>
            <dd>{bundle.resilience.confidence.level}</dd>
          </dl>
          <p className="mt-3 text-sm">{bundle.resilience.next_best_move_summary}</p>
          <ul className="mt-2 list-disc list-inside text-sm text-muted-foreground space-y-1">
            {bundle.resilience.fragility_explanation.map((x, i) => (
              <li key={i}>{x}</li>
            ))}
          </ul>
        </Card>

        <Card className="p-4">
          <h3 className="font-semibold mb-2">Ranked next moves</h3>
          {bundle.ranked_next_plans.length === 0 ? (
            <p className="text-sm text-muted-foreground">No mesh recommendations in this snapshot.</p>
          ) : (
            <ul className="space-y-3">
              {bundle.ranked_next_plans.slice(0, 6).map((r) => (
                <li key={r.id} className="text-sm border-b border-border/50 pb-2 last:border-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-medium">
                      {r.rank}. {r.title}
                    </span>
                    <Badge variant="outline">{r.verdict}</Badge>
                    <Badge variant="secondary">{r.benefit_band}</Badge>
                  </div>
                  {r.lines.slice(0, 2).map((line, i) => (
                    <p key={i} className="text-muted-foreground mt-1">
                      {line}
                    </p>
                  ))}
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>

      <Card className="p-4">
        <h3 className="font-semibold mb-2">Resilience advisory alerts</h3>
        <p className="text-xs text-muted-foreground mb-3">
          Synthetic advisories (transport planning/advisory). Not transport failures — graph-shape concerns with evidence references.
        </p>
        {advisories.length === 0 ? (
          <p className="text-sm text-muted-foreground" data-testid="planning-advisories-empty">
            No active advisories in this response. If the alerts API failed or returned empty, this does not prove the mesh is free of risk — only that no advisory rows were returned here.
          </p>
        ) : (
          <ul className="space-y-2 text-sm">
            {advisories.map((a) => (
              <li key={a.id} className="border-b border-border/40 pb-2 last:border-0">
                <div className="flex flex-wrap gap-2 items-center">
                  <Badge variant={a.severity === 'warning' ? 'warning' : 'secondary'}>{a.severity}</Badge>
                  <span className="font-mono text-xs">{a.reason}</span>
                </div>
                <p className="mt-1">{a.summary}</p>
              </li>
            ))}
          </ul>
        )}
      </Card>

      <Card className="p-4">
        <h3 className="font-semibold mb-2">Compare plans</h3>
        <p className="text-sm text-muted-foreground mb-2">
          Enter saved plan IDs (comma-separated). Uses topology-bounded ranking with explicit low-regret and fragility dimensions.
        </p>
        <div className="flex flex-wrap gap-2 items-center">
          <input
            className="flex-1 min-w-[200px] rounded border border-border bg-background px-2 py-1 text-sm"
            placeholder="plan-abc, plan-def"
            value={compareIds}
            onChange={(e) => setCompareIds(e.target.value)}
          />
          <button
            type="button"
            className="text-sm rounded bg-primary text-primary-foreground px-3 py-1 disabled:opacity-50"
            disabled={compareLoading}
            onClick={() => void runCompare()}
          >
            {compareLoading ? 'Comparing…' : 'Compare'}
          </button>
        </div>
        {compareErr && <p className="text-sm text-destructive mt-2">{compareErr}</p>}
        {comparison && (
          <div className="mt-4 space-y-2 text-sm">
            <p className="text-muted-foreground">{comparison.summary_lines[0]}</p>
            <div className="flex flex-wrap gap-2">
              {comparison.low_regret_pick_id && (
                <Badge variant="outline">Low-regret: {comparison.low_regret_pick_id}</Badge>
              )}
              {comparison.best_upside_pick_id && (
                <Badge variant="outline">Upside: {comparison.best_upside_pick_id}</Badge>
              )}
              {comparison.best_diagnostic_pick_id && (
                <Badge variant="outline">Diagnostic: {comparison.best_diagnostic_pick_id}</Badge>
              )}
              {comparison.wait_observe_option_id && (
                <Badge variant="secondary">Wait/observe: {comparison.wait_observe_option_id}</Badge>
              )}
            </div>
            <p className="text-xs text-muted-foreground">Evidence model: {comparison.evidence_classification}</p>
            <ul className="list-disc list-inside">
              {comparison.ranked_by_upside.slice(0, 5).map((r) => (
                <li key={r.id}>
                  {r.label}: low-regret {r.dimensions.low_regret_score.toFixed(2)}, fragility{' '}
                  {r.dimensions.assumption_fragility_score.toFixed(2)}
                </li>
              ))}
            </ul>
          </div>
        )}
      </Card>

      <Card className="p-4">
        <h3 className="font-semibold mb-2">Plan execution history</h3>
        <p className="text-sm text-muted-foreground mb-2">
          Load tracked executions for a plan (start and validate via API or <code className="text-xs">mel plan</code>).
        </p>
        <div className="flex flex-wrap gap-2 items-center">
          <input
            className="flex-1 min-w-[200px] rounded border border-border bg-background px-2 py-1 text-sm"
            placeholder="plan id"
            value={execPlanId}
            onChange={(e) => setExecPlanId(e.target.value)}
          />
          <button
            type="button"
            className="text-sm rounded border border-border px-3 py-1"
            onClick={() => void loadExecutions()}
          >
            Load
          </button>
        </div>
        {execErr && <p className="text-sm text-destructive mt-2">{execErr}</p>}
        {executions.length > 0 && (
          <ul className="mt-3 space-y-2 text-sm">
            {executions.map((ex) => (
              <li key={ex.execution_id} className="border-b border-border/40 pb-2">
                <span className="font-mono text-xs">{ex.execution_id}</span>
                <span className="text-muted-foreground"> — {ex.status}</span>
                <span className="text-muted-foreground"> — started {ex.started_at}</span>
                {ex.observation_horizon_hours > 0 && (
                  <span className="text-muted-foreground"> — observe {ex.observation_horizon_hours}h</span>
                )}
              </li>
            ))}
          </ul>
        )}
      </Card>

      <Card className="p-4">
        <h3 className="font-semibold mb-3">Playbooks (field-guide)</h3>
        {bundle.playbooks.length === 0 ? (
          <p className="text-sm text-muted-foreground">No playbooks for this state.</p>
        ) : (
          <div className="space-y-4">
            {bundle.playbooks.map((pb) => (
              <div key={pb.class} className="rounded-lg border border-border p-3">
                <div className="flex items-start justify-between gap-2 flex-wrap">
                  <h4 className="font-medium">{pb.title}</h4>
                  <Badge variant="outline">{pb.class}</Badge>
                </div>
                <p className="text-sm text-muted-foreground mt-1">{pb.summary}</p>
                <p className="text-sm mt-2">
                  <span className="text-muted-foreground">Milestone:</span> {pb.minimum_viable_milestone}
                </p>
                <ol className="mt-2 space-y-2 list-decimal list-inside text-sm">
                  {pb.steps.map((s) => (
                    <li key={s.order}>
                      <span className="font-medium">{s.title}</span>
                      <span className="text-muted-foreground">
                        {' '}
                        — {s.rationale} (observe ~{s.observe_hours}h)
                      </span>
                    </li>
                  ))}
                </ol>
              </div>
            ))}
          </div>
        )}
      </Card>

      <Card className="p-4">
        <h3 className="font-semibold mb-2">Node criticality (observed graph)</h3>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-muted-foreground border-b">
                <th className="py-2 pr-4">#</th>
                <th className="py-2 pr-4">Node</th>
                <th className="py-2 pr-4">Critical</th>
                <th className="py-2 pr-4">Partition risk</th>
                <th className="py-2 pr-4">SPOF</th>
              </tr>
            </thead>
            <tbody>
              {bundle.node_profiles.slice(0, 15).map((n) => (
                <tr key={n.node_num} className="border-b border-border/40">
                  <td className="py-2 pr-4">{n.recovery_priority}</td>
                  <td className="py-2 pr-4">{n.short_name || n.node_num}</td>
                  <td className="py-2 pr-4">{n.critical_node_score.toFixed(2)}</td>
                  <td className="py-2 pr-4">{n.partition_risk_score.toFixed(2)}</td>
                  <td className="py-2 pr-4">{n.spof_class}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      <Card className="p-4">
        <h3 className="font-semibold mb-2">Limits</h3>
        <ul className="list-disc list-inside text-sm text-muted-foreground space-y-1">
          {bundle.limits.map((x, i) => (
            <li key={i}>{x}</li>
          ))}
        </ul>
        <p className="text-xs text-muted-foreground mt-3">Computed {bundle.computed_at}</p>
      </Card>
    </div>
  )
}
