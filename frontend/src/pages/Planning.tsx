import { useEffect, useRef, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { EmptyState } from '@/components/ui/EmptyState'
import { usePageHotkeys } from '@/hooks/usePageHotkeys'

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

interface PlanningEvidenceFlags {
  baseline_missing?: boolean
  confounded_same_assessment_context?: boolean
  directional_only?: boolean
  inconclusive?: boolean
  topology_or_graph_drift_detected?: boolean
  limited_confidence?: boolean
  no_advisories?: boolean
  recommendation_present_with_uncertain_evidence?: boolean
}

interface PlanningBundle {
  evidence_model: string
  graph_hash?: string
  mesh_assessment_id?: string
  transport_connected: boolean
  topology_enabled: boolean
  evidence_flags?: PlanningEvidenceFlags
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
interface AdvisoryAlertsResponse {
  alerts?: AdvisoryAlertRow[]
  evidence_flags?: PlanningEvidenceFlags
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

function deriveEvidenceSignalsFromLegacyText(bundle: PlanningBundle, advisories: AdvisoryAlertRow[]): EvidenceSignal[] {
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

function deriveEvidenceSignals(bundle: PlanningBundle, advisories: AdvisoryAlertRow[], advisoryFlags?: PlanningEvidenceFlags): EvidenceSignal[] {
  const flags = {
    ...bundle.evidence_flags,
    ...advisoryFlags,
  }
  const hasTypedFlags = Object.values(flags).some((v) => v === true || v === false)
  const signals: EvidenceSignal[] = []
  if (flags.baseline_missing) {
    signals.push({ id: 'baseline-missing', message: 'Baseline evidence is missing or unavailable; before/after deltas are directional only.' })
  }
  if (flags.confounded_same_assessment_context) {
    signals.push({
      id: 'confounded',
      message: 'Current and baseline references are potentially confounded (same or concurrent assessment context), so causality is not established.',
    })
  }
  if (flags.directional_only || flags.inconclusive) {
    signals.push({
      id: 'directional',
      message: 'Validation remains directional/inconclusive and does not prove propagation or RF behavior.',
    })
  }
  if (flags.topology_or_graph_drift_detected) {
    signals.push({
      id: 'topology-drift',
      message: 'Graph/topology drift is present, so trend comparison can be confounded by concurrent changes.',
    })
  }
  if (flags.no_advisories) {
    signals.push({
      id: 'empty-advisories-not-clear',
      message: 'No advisory rows here is not an all-clear; evidence uncertainty still applies.',
    })
  }
  if (flags.limited_confidence || flags.recommendation_present_with_uncertain_evidence) {
    signals.push({
      id: 'limited-confidence',
      message: 'Recommendations exist, but confidence is limited by missing or non-independent evidence.',
    })
  }
  if (signals.length > 0 || hasTypedFlags) {
    return signals
  }
  // Backward-compatibility path for older API payloads that predate typed evidence_flags.
  // Typed flags remain the canonical operator-truth surface when present (including explicit false values).
  return deriveEvidenceSignalsFromLegacyText(bundle, advisories)
}

export function Planning() {
  const [bundle, setBundle] = useState<PlanningBundle | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [advisories, setAdvisories] = useState<AdvisoryAlertRow[]>([])
  const [advisoryFlags, setAdvisoryFlags] = useState<PlanningEvidenceFlags | undefined>(undefined)
  const [compareIds, setCompareIds] = useState('')
  const [comparison, setComparison] = useState<PlanComparison | null>(null)
  const [compareErr, setCompareErr] = useState<string | null>(null)
  const [compareLoading, setCompareLoading] = useState(false)
  const [execPlanId, setExecPlanId] = useState('')
  const [executions, setExecutions] = useState<PlanExecution[]>([])
  const [execErr, setExecErr] = useState<string | null>(null)
  const [showAllPlans, setShowAllPlans] = useState(false)
  const sectionRefs = useRef<Record<string, HTMLElement | null>>({})
  const compareInputRef = useRef<HTMLInputElement>(null)

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
          const adv = (await aRes.json()) as AdvisoryAlertsResponse
          if (!cancelled) {
            setAdvisories(adv.alerts ?? [])
            setAdvisoryFlags(adv.evidence_flags)
          }
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

  usePageHotkeys([
    { key: '/', description: 'Focus compare input', handler: () => compareInputRef.current?.focus() },
    { key: '1', description: 'Jump to posture', handler: () => sectionRefs.current.posture?.scrollIntoView({ behavior: 'smooth' }) },
    { key: '2', description: 'Jump to compare', handler: () => sectionRefs.current.compare?.scrollIntoView({ behavior: 'smooth' }) },
    { key: '3', description: 'Jump to playbooks', handler: () => sectionRefs.current.playbooks?.scrollIntoView({ behavior: 'smooth' }) },
    { key: 'o', description: 'Show more ranked plans', handler: () => setShowAllPlans(true) },
    { key: 'c', description: 'Condense ranked plans', handler: () => setShowAllPlans(false) },
  ])

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

  if (loading) return <div className="p-6"><PageHeader title="Deployment planning" description="Loading…" /></div>
  if (error) {
    return (
      <div className="p-6 space-y-4">
        <PageHeader title="Deployment planning" description="Bounded what-if and resilience (topology evidence, not RF maps)" />
        <EmptyState title="Planning unavailable" description={error} />
      </div>
    )
  }
  if (!bundle) return null

  const bn = bundle.best_next_move
  const evidenceSignals = deriveEvidenceSignals(bundle, advisories, advisoryFlags)
  const knownFacts = [
    `Transport is ${bundle.transport_connected ? 'connected' : 'not connected'}.`,
    `Topology model is ${bundle.topology_enabled ? 'enabled' : 'disabled'}.`,
    `Resilience score ${bundle.resilience.resilience_score.toFixed(2)} with ${bundle.resilience.confidence.level} confidence.`,
  ]
  const inferredFacts = [bundle.resilience.next_best_move_summary, ...(bn?.summary_lines ?? []).slice(0, 2)].filter(Boolean)
  const cautionFacts = [...evidenceSignals.map((s) => s.message), ...(bn?.uncertainty_notes ?? [])]
  const unsupportedFacts = [...bundle.limits]

  return (
    <div className="p-4 md:p-6 space-y-4 max-w-6xl mx-auto">
      <PageHeader
        title="Deployment planning"
        description="Dense planning posture from observed mesh evidence. Not RF coverage simulation and not route certainty."
      />

      <p className="text-xs text-muted-foreground" data-testid="planning-evidence-banner">{bundle.evidence_model}</p>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4" ref={(el) => (sectionRefs.current.posture = el)}>
        <PostureCard title="Known" lines={knownFacts} variant="default" />
        <PostureCard title="Inferred" lines={inferredFacts} variant="secondary" />
        <PostureCard title="Requires caution" lines={cautionFacts.length > 0 ? cautionFacts : ['No explicit caution markers in this payload.']} variant="warning" testId="planning-evidence-signals" />
        <PostureCard title="Unsupported / limits" lines={unsupportedFacts.length > 0 ? unsupportedFacts : ['No explicit limit lines returned.']} variant="outline" />
      </div>

      {bn && (
        <Card className="p-3 md:p-4 border-border">
          <div className="flex flex-wrap items-center gap-2 mb-2">
            <h3 className="font-semibold mr-2">Best next move</h3>
            <Badge variant="outline">{bn.primary_verdict}</Badge>
            <Badge variant="secondary">{bn.evidence_classification}</Badge>
            {bn.recommendation_key && <Badge variant="outline">{bn.recommendation_key}</Badge>}
          </div>
          <p className="font-medium text-sm md:text-base">{bn.title}</p>
          {(bn.summary_lines ?? []).length > 0 && (
            <ul className="mt-2 list-disc list-inside text-sm text-muted-foreground space-y-1">
              {(bn.summary_lines ?? []).map((line, i) => <li key={i}>{line}</li>)}
            </ul>
          )}
          {bn.wait_observe_rationale && <p className="mt-2 text-xs text-warning"><span className="font-medium">Why wait/observe:</span> {bn.wait_observe_rationale}</p>}
        </Card>
      )}

      <div className="grid gap-3 lg:grid-cols-2">
        <Card className="p-3 md:p-4">
          <h3 className="font-semibold mb-2 text-sm">Resilience + fragility</h3>
          <dl className="grid grid-cols-2 gap-y-1 text-xs md:text-sm">
            <dt className="text-muted-foreground">Resilience</dt><dd>{bundle.resilience.resilience_score.toFixed(2)}</dd>
            <dt className="text-muted-foreground">Redundancy</dt><dd>{bundle.resilience.redundancy_score.toFixed(2)}</dd>
            <dt className="text-muted-foreground">Partition risk</dt><dd>{bundle.resilience.partition_risk_score.toFixed(2)}</dd>
            <dt className="text-muted-foreground">Confidence</dt><dd>{bundle.resilience.confidence.level}</dd>
          </dl>
          <ul className="mt-2 list-disc list-inside text-xs md:text-sm text-muted-foreground space-y-1">
            {bundle.resilience.fragility_explanation.slice(0, 4).map((x, i) => <li key={i}>{x}</li>)}
          </ul>
        </Card>

        <Card className="p-3 md:p-4">
          <div className="flex items-center justify-between gap-2 mb-2">
            <h3 className="font-semibold text-sm">Ranked next moves</h3>
            <button type="button" className="text-xs underline" onClick={() => setShowAllPlans((v) => !v)}>
              {showAllPlans ? 'Condense list' : 'Show more'}
            </button>
          </div>
          {bundle.ranked_next_plans.length === 0 ? (
            <p className="text-xs text-muted-foreground">No recommendations in this snapshot.</p>
          ) : (
            <ul className="space-y-2 text-xs md:text-sm">
              {bundle.ranked_next_plans.slice(0, showAllPlans ? 10 : 4).map((r) => (
                <li key={r.id} className="border-b border-border/50 pb-2 last:border-0">
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <span className="font-medium">{r.rank}. {r.title}</span>
                    <Badge variant="outline">{r.verdict}</Badge>
                    <Badge variant="secondary">{r.benefit_band}</Badge>
                  </div>
                  {r.lines.slice(0, 1).map((line, i) => <p key={i} className="text-muted-foreground mt-0.5">{line}</p>)}
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>

      <Card className="p-3 md:p-4">
        <h3 className="font-semibold mb-2 text-sm">Resilience advisory alerts</h3>
        {advisories.length === 0 ? (
          <p className="text-xs md:text-sm text-muted-foreground" data-testid="planning-advisories-empty">
            No advisory rows returned in this response. This is not an all-clear and does not prove low risk.
          </p>
        ) : (
          <ul className="space-y-2 text-xs md:text-sm">
            {advisories.map((a) => (
              <li key={a.id} className="border-b border-border/40 pb-2 last:border-0">
                <div className="flex flex-wrap gap-2 items-center">
                  <Badge variant={a.severity === 'warning' ? 'warning' : 'secondary'}>{a.severity}</Badge>
                  <span className="font-mono text-[11px]">{a.reason}</span>
                </div>
                <p className="mt-1">{a.summary}</p>
              </li>
            ))}
          </ul>
        )}
      </Card>

      <div className="grid gap-3 lg:grid-cols-2" ref={(el) => (sectionRefs.current.compare = el)}>
        <Card className="p-3 md:p-4">
          <h3 className="font-semibold mb-2 text-sm">Compare plans</h3>
          <div className="flex flex-wrap gap-2 items-center">
            <input
              ref={compareInputRef}
              className="flex-1 min-w-[170px] rounded border border-border bg-background px-2 py-1 text-sm"
              placeholder="plan-abc, plan-def"
              value={compareIds}
              onChange={(e) => setCompareIds(e.target.value)}
            />
            <button type="button" className="text-sm rounded bg-primary text-primary-foreground px-3 py-1 disabled:opacity-50" disabled={compareLoading} onClick={() => void runCompare()}>
              {compareLoading ? 'Comparing…' : 'Compare'}
            </button>
          </div>
          {compareErr && <p className="text-sm text-destructive mt-2">{compareErr}</p>}
          {comparison && (
            <div className="mt-3 space-y-2 text-xs md:text-sm">
              <p className="text-muted-foreground">{comparison.summary_lines[0]}</p>
              <div className="overflow-x-auto">
                <table className="w-full text-xs">
                  <thead>
                    <tr className="text-left border-b border-border/50">
                      <th className="py-1 pr-2">Plan</th>
                      <th className="py-1 pr-2">Low-regret</th>
                      <th className="py-1 pr-2">Fragility</th>
                    </tr>
                  </thead>
                  <tbody>
                    {comparison.ranked_by_upside.slice(0, 6).map((r) => (
                      <tr key={r.id} className="border-b border-border/30">
                        <td className="py-1 pr-2">{r.label}</td>
                        <td className="py-1 pr-2">{r.dimensions.low_regret_score.toFixed(2)}</td>
                        <td className="py-1 pr-2">{r.dimensions.assumption_fragility_score.toFixed(2)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <p className="text-[11px] text-muted-foreground">Evidence model: {comparison.evidence_classification}</p>
            </div>
          )}
        </Card>

        <Card className="p-3 md:p-4">
          <h3 className="font-semibold mb-2 text-sm">Plan execution history</h3>
          <div className="flex flex-wrap gap-2 items-center">
            <input className="flex-1 min-w-[170px] rounded border border-border bg-background px-2 py-1 text-sm" placeholder="plan id" value={execPlanId} onChange={(e) => setExecPlanId(e.target.value)} />
            <button type="button" className="text-sm rounded border border-border px-3 py-1" onClick={() => void loadExecutions()}>Load</button>
          </div>
          {execErr && <p className="text-sm text-destructive mt-2">{execErr}</p>}
          {executions.length > 0 && (
            <ul className="mt-3 space-y-2 text-xs md:text-sm">
              {executions.map((ex) => (
                <li key={ex.execution_id} className="border-b border-border/40 pb-2">
                  <span className="font-mono text-[11px]">{ex.execution_id}</span>
                  <span className="text-muted-foreground"> — {ex.status} — {ex.started_at}</span>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>

      <Card className="p-3 md:p-4">
        <div className="flex items-center justify-between gap-2 mb-2" ref={(el) => (sectionRefs.current.playbooks = el)}>
          <h3 className="font-semibold text-sm">Playbooks</h3>
          <p className="text-[11px] text-muted-foreground">Field-guide summaries with bounded observations.</p>
        </div>
        {bundle.playbooks.length === 0 ? (
          <p className="text-sm text-muted-foreground">No playbooks for this state.</p>
        ) : (
          <div className="space-y-3">
            {bundle.playbooks.map((pb) => (
              <div key={pb.class} className="rounded-lg border border-border p-3">
                <div className="flex items-start justify-between gap-2 flex-wrap">
                  <h4 className="font-medium text-sm">{pb.title}</h4>
                  <Badge variant="outline">{pb.class}</Badge>
                </div>
                <p className="text-xs text-muted-foreground mt-1">{pb.summary}</p>
                <ol className="mt-2 space-y-1 list-decimal list-inside text-xs">
                  {pb.steps.slice(0, 4).map((step) => (
                    <li key={step.order}><span className="font-medium">{step.title}</span> <span className="text-muted-foreground">— {step.rationale}</span></li>
                  ))}
                </ol>
              </div>
            ))}
          </div>
        )}
      </Card>

      <Card className="p-3 md:p-4">
        <h3 className="font-semibold mb-2 text-sm">Node criticality (observed graph)</h3>
        <div className="overflow-x-auto">
          <table className="w-full text-xs md:text-sm">
            <thead>
              <tr className="text-left text-muted-foreground border-b">
                <th className="py-2 pr-3">#</th><th className="py-2 pr-3">Node</th><th className="py-2 pr-3">Critical</th><th className="py-2 pr-3">Partition</th><th className="py-2 pr-3">SPOF</th>
              </tr>
            </thead>
            <tbody>
              {bundle.node_profiles.slice(0, 15).map((n) => (
                <tr key={n.node_num} className="border-b border-border/40">
                  <td className="py-2 pr-3">{n.recovery_priority}</td><td className="py-2 pr-3">{n.short_name || n.node_num}</td><td className="py-2 pr-3">{n.critical_node_score.toFixed(2)}</td><td className="py-2 pr-3">{n.partition_risk_score.toFixed(2)}</td><td className="py-2 pr-3">{n.spof_class}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="text-[11px] text-muted-foreground mt-2">Computed {bundle.computed_at}</p>
      </Card>
    </div>
  )
}

function PostureCard({ title, lines, variant, testId }: { title: string; lines: string[]; variant: 'default' | 'secondary' | 'warning' | 'outline'; testId?: string }) {
  const tone =
    variant === 'warning'
      ? 'border-warning/30 bg-warning/5'
      : variant === 'secondary'
        ? 'border-border bg-muted/25'
        : 'border-border bg-card'

  return (
    <Card className={`p-3 ${tone}`}>
      <h3 className="text-xs font-semibold uppercase tracking-wide mb-2">{title}</h3>
      <ul className="space-y-1 text-xs text-muted-foreground" data-testid={testId}>
        {lines.slice(0, 5).map((line, i) => (
          <li key={`${title}-${i}`} className="leading-relaxed">{line}</li>
        ))}
      </ul>
    </Card>
  )
}
