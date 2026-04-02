import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { EmptyState } from '@/components/ui/EmptyState'
import type { Incident } from '@/types/api'

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

function withReturnParam(targetPath: string, returnPath: string): string {
  if (!returnPath.startsWith('/')) return targetPath
  const joiner = targetPath.includes('?') ? '&' : '?'
  return `${targetPath}${joiner}return=${encodeURIComponent(returnPath)}`
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

function planningBestMoveAdvisoryNote(bundle: PlanningBundle, incidentId: string | null): string {
  const parts: string[] = [
    'Advisory ranking from the planning bundle — not RF or delivery proof.',
  ]
  if (!bundle.transport_connected) {
    parts.push('Transport disconnected in this snapshot; graph may be stale.')
  }
  if (!bundle.topology_enabled) {
    parts.push('Topology model off — resilience scores are not full graph-backed.')
  }
  if (incidentId) {
    parts.push(`Verify against incident ${incidentId.slice(0, 8)}… replay, topology focus, and control queue before acting.`)
  } else {
    parts.push('Verify against incident detail, topology, and diagnostics before acting.')
  }
  return parts.join(' ')
}

function planningDecisionBoard(bundle: PlanningBundle, evidenceSignals: EvidenceSignal[]) {
  const bn = bundle.best_next_move
  const unknowns: string[] = []
  if (bundle.evidence_flags?.baseline_missing) unknowns.push('Baseline snapshot missing — deltas are directional only.')
  if (bundle.evidence_flags?.inconclusive || bundle.evidence_flags?.directional_only) {
    unknowns.push('Outcome remains inconclusive for RF or propagation — graph-only bounds.')
  }
  if (!bundle.transport_connected) unknowns.push('Transport not connected in this snapshot — planning may reflect stale graph.')
  if (!bundle.topology_enabled) unknowns.push('Topology model disabled — resilience scores are not graph-backed.')
  for (const s of evidenceSignals) {
    if (s.id === 'confounded' || s.id === 'topology-drift') unknowns.push(s.message)
  }
  const unsupported: string[] = []
  if (!bundle.topology_enabled) unsupported.push('Topology-derived planning surfaces are limited while topology is off.')

  const weakAssumptions = (bn?.uncertainty_notes ?? []).slice(0, 4)
  const inferredTradeoffs = (bundle.resilience.fragility_explanation ?? []).slice(0, 3)
  const missingEvidence = (bundle.resilience.confidence.missing_inputs ?? []).slice(0, 4)
  const nextCheck: string[] = []
  if (bn?.would_validate_with?.length) nextCheck.push(...bn.would_validate_with.slice(0, 3))
  if (bn?.wait_observe_rationale) nextCheck.push(`Wait/observe: ${bn.wait_observe_rationale}`)
  if (nextCheck.length === 0 && bundle.resilience.next_best_move_summary) {
    nextCheck.push(bundle.resilience.next_best_move_summary)
  }

  return { unknowns, unsupported, weakAssumptions, inferredTradeoffs, missingEvidence, nextCheck }
}

export function Planning() {
  const [searchParams] = useSearchParams()
  const incidentIdParam = (searchParams.get('incident') || '').trim()
  const returnParam = (searchParams.get('return') || '').trim()
  const [incidentCtx, setIncidentCtx] = useState<Incident | null>(null)
  const [incidentErr, setIncidentErr] = useState<string | null>(null)

  const [denseLayout, setDenseLayout] = useState(true)
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
    if (!incidentIdParam) {
      setIncidentCtx(null)
      setIncidentErr(null)
      return
    }
    let cancelled = false
    ;(async () => {
      try {
        const res = await fetch(`/api/v1/incidents/${encodeURIComponent(incidentIdParam)}`)
        if (!res.ok) {
          if (!cancelled) {
            setIncidentCtx(null)
            setIncidentErr(`HTTP ${res.status}`)
          }
          return
        }
        const data = (await res.json()) as Incident
        if (!cancelled) {
          setIncidentCtx(data)
          setIncidentErr(null)
        }
      } catch {
        if (!cancelled) {
          setIncidentCtx(null)
          setIncidentErr('Failed to load incident')
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [incidentIdParam])

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
  const board = planningDecisionBoard(bundle, evidenceSignals)

  return (
    <div className={denseLayout ? 'p-4 space-y-4 max-w-6xl mx-auto' : 'p-6 space-y-6 max-w-6xl mx-auto'}>
      {incidentIdParam && (
        <div
          className="rounded-lg border border-primary/20 bg-primary/5 px-3 py-2 text-xs text-muted-foreground"
          role="region"
          aria-label="Planning context incident"
        >
          {incidentErr && <p className="text-warning">Incident {incidentIdParam}: {incidentErr}</p>}
          {!incidentErr && incidentCtx && (
            <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
              <span>
                Planning with incident context:{' '}
                <Link
                  to={withReturnParam(`/incidents/${encodeURIComponent(incidentCtx.id)}`, returnParam)}
                  className="font-medium text-primary hover:underline"
                >
                  {incidentCtx.title || incidentCtx.id.slice(0, 12)}
                </Link>
              </span>
              <span className="text-muted-foreground/50 hidden sm:inline" aria-hidden>
                ·
              </span>
              <Link
                to={withReturnParam(`/incidents/${encodeURIComponent(incidentCtx.id)}?replay=1`, returnParam)}
                className="font-medium text-primary hover:underline"
              >
                Replay
              </Link>
              <span className="text-muted-foreground/50 hidden sm:inline" aria-hidden>
                ·
              </span>
              <Link
                to={withReturnParam(`/topology?incident=${encodeURIComponent(incidentCtx.id)}&filter=incident_focus`, returnParam)}
                className="font-medium text-primary hover:underline"
              >
                Topology focus
              </Link>
              <span className="text-muted-foreground/50 hidden sm:inline" aria-hidden>
                ·
              </span>
              <Link
                to={withReturnParam(`/control-actions?incident=${encodeURIComponent(incidentCtx.id)}`, returnParam)}
                className="font-medium text-primary hover:underline"
              >
                Control queue
              </Link>
              <span className="text-muted-foreground/50 hidden sm:inline" aria-hidden>
                ·
              </span>
              <Link to="/diagnostics" className="font-medium text-muted-foreground hover:text-foreground hover:underline">
                Support bundle
              </Link>
            </span>
          )}
          {!incidentErr && !incidentCtx && <span>Loading incident for cross-link…</span>}
        </div>
      )}

      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <PageHeader
          title="Deployment planning"
          description="Next-step moves, resilience, and playbooks from observed mesh evidence. Not RF coverage simulation."
        />
        <div className="flex flex-wrap items-center gap-2 shrink-0">
          {returnParam.startsWith('/') && (
            <Link to={returnParam} className="text-xs font-semibold text-primary hover:underline px-1">
              ← Back
            </Link>
          )}
          <button
            type="button"
            onClick={() => setDenseLayout((d) => !d)}
            className="shrink-0 rounded-lg border border-border/70 bg-background px-3 py-2 text-xs font-semibold hover:bg-muted/40"
            aria-pressed={denseLayout}
          >
            {denseLayout ? 'Comfortable layout' : 'Dense layout'}
          </button>
        </div>
      </div>

      <Card className={`border-warning/25 bg-warning/5 ${denseLayout ? 'p-3' : 'p-4'}`}>
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
          <div className="mt-3 border-t border-warning/20 pt-3" data-testid="planning-evidence-signals">
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
          <p className="mt-3 border-t border-warning/20 pt-3 text-sm">
            <span className="font-medium text-foreground">Wait vs expand: </span>
            {bundle.wait_versus_expand_hint}
          </p>
        )}
      </Card>

      <div data-testid="planning-decision-board">
        <Card className={`border-border/80 ${denseLayout ? 'p-3' : 'p-4'}`}>
        <h3 className={`font-semibold mb-2 ${denseLayout ? 'text-sm' : ''}`}>Decision board</h3>
        <p className="text-[11px] text-muted-foreground mb-3">
          Scan-first layout: what is known vs unknown vs unsupported, then what to check next. Does not add simulation beyond the planning bundle.
        </p>
        <div className={`grid gap-3 ${denseLayout ? 'sm:grid-cols-2 lg:grid-cols-3' : 'md:grid-cols-2 lg:grid-cols-3'}`}>
          <div className="rounded-lg border border-border/60 p-2.5">
            <p className="text-[11px] font-semibold text-foreground uppercase tracking-wide mb-1">Known (from bundle)</p>
            <ul className="text-xs text-muted-foreground list-disc list-inside space-y-0.5">
              <li>Transport: {bundle.transport_connected ? 'connected' : 'not connected'}</li>
              <li>Topology model: {bundle.topology_enabled ? 'enabled' : 'disabled'}</li>
              {bn?.primary_verdict && <li>Verdict: {bn.primary_verdict}</li>}
              <li>Resilience confidence: {bundle.resilience.confidence.level}</li>
            </ul>
          </div>
          <div className="rounded-lg border border-warning/20 bg-warning/5 p-2.5">
            <p className="text-[11px] font-semibold text-foreground uppercase tracking-wide mb-1">Unknown / confounded</p>
            {board.unknowns.length === 0 ? (
              <p className="text-xs text-muted-foreground">No extra unknown flags beyond the evidence banner.</p>
            ) : (
              <ul className="text-xs text-muted-foreground list-disc list-inside space-y-0.5">
                {board.unknowns.map((u, i) => (
                  <li key={i}>{u}</li>
                ))}
              </ul>
            )}
          </div>
          <div className="rounded-lg border border-border/60 p-2.5">
            <p className="text-[11px] font-semibold text-foreground uppercase tracking-wide mb-1">Unsupported / gated</p>
            {board.unsupported.length === 0 ? (
              <p className="text-xs text-muted-foreground">No explicit unsupported gates in this view.</p>
            ) : (
              <ul className="text-xs text-muted-foreground list-disc list-inside space-y-0.5">
                {board.unsupported.map((u, i) => (
                  <li key={i}>{u}</li>
                ))}
              </ul>
            )}
          </div>
          <div className="rounded-lg border border-border/60 p-2.5 sm:col-span-2 lg:col-span-1">
            <p className="text-[11px] font-semibold text-foreground uppercase tracking-wide mb-1">Weak assumptions</p>
            {board.weakAssumptions.length === 0 ? (
              <p className="text-xs text-muted-foreground">—</p>
            ) : (
              <ul className="text-xs text-muted-foreground list-disc list-inside space-y-0.5">
                {board.weakAssumptions.map((u, i) => (
                  <li key={i}>{u}</li>
                ))}
              </ul>
            )}
          </div>
          <div className="rounded-lg border border-border/60 p-2.5 sm:col-span-2 lg:col-span-1">
            <p className="text-[11px] font-semibold text-foreground uppercase tracking-wide mb-1">Inferred tradeoffs (graph)</p>
            {board.inferredTradeoffs.length === 0 ? (
              <p className="text-xs text-muted-foreground">—</p>
            ) : (
              <ul className="text-xs text-muted-foreground list-disc list-inside space-y-0.5">
                {board.inferredTradeoffs.map((u, i) => (
                  <li key={i}>{u}</li>
                ))}
              </ul>
            )}
          </div>
          <div className="rounded-lg border border-border/60 p-2.5 sm:col-span-2 lg:col-span-1">
            <p className="text-[11px] font-semibold text-foreground uppercase tracking-wide mb-1">Missing evidence inputs</p>
            {board.missingEvidence.length === 0 ? (
              <p className="text-xs text-muted-foreground">None listed on resilience confidence.</p>
            ) : (
              <ul className="text-xs text-muted-foreground list-disc list-inside space-y-0.5">
                {board.missingEvidence.map((u, i) => (
                  <li key={i}>{u}</li>
                ))}
              </ul>
            )}
          </div>
          <div className="rounded-lg border border-primary/15 bg-primary/5 p-2.5 sm:col-span-2 lg:col-span-3">
            <p className="text-[11px] font-semibold text-foreground uppercase tracking-wide mb-1">Recommended next check</p>
            <ul className="text-xs text-foreground list-disc list-inside space-y-0.5">
              {board.nextCheck.slice(0, 5).map((u, i) => (
                <li key={i}>{u}</li>
              ))}
            </ul>
          </div>
        </div>
        </Card>
      </div>

      {denseLayout ? (
        <div className="grid gap-3 lg:grid-cols-3" data-testid="planning-dense-grid">
          {bn && (
            <Card className="p-3 border-border lg:col-span-1">
              <h3 className="text-sm font-semibold mb-2">Best next move</h3>
              <div className="flex flex-wrap gap-1.5 mb-2">
                <Badge variant="outline">{bn.primary_verdict}</Badge>
                <Badge variant="secondary">{bn.evidence_classification}</Badge>
                {bn.recommendation_key && (
                  <span title="Key for outcome history">
                    <Badge variant="outline">{bn.recommendation_key}</Badge>
                  </span>
                )}
              </div>
              <p className="text-xs text-foreground/90 border-l-2 border-warning/35 pl-2 mb-2 leading-snug" role="status">
                {planningBestMoveAdvisoryNote(bundle, incidentIdParam || null)}
              </p>
              <p className="text-sm font-medium leading-snug">{bn.title}</p>
              <ul className="mt-2 list-disc list-inside text-xs text-muted-foreground space-y-0.5">
                {(bn.summary_lines ?? []).slice(0, 4).map((line, i) => (
                  <li key={i}>{line}</li>
                ))}
              </ul>
              {bn.wait_observe_rationale && (
                <p className="mt-2 text-xs text-warning leading-snug">
                  <span className="font-medium">Wait/observe: </span>
                  {bn.wait_observe_rationale}
                </p>
              )}
              {(bn.would_validate_with?.length ?? 0) > 0 && (
                <div className="mt-2 text-xs">
                  <span className="text-muted-foreground">Validate with:</span>
                  <ul className="list-disc list-inside mt-0.5">
                    {(bn.would_validate_with ?? []).slice(0, 3).map((x, i) => (
                      <li key={i}>{x}</li>
                    ))}
                  </ul>
                </div>
              )}
              {(bn.uncertainty_notes?.length ?? 0) > 0 && (
                <ul className="mt-2 text-[11px] text-muted-foreground list-disc list-inside" data-testid="planning-uncertainty-notes">
                  {(bn.uncertainty_notes ?? []).map((x, i) => (
                    <li key={i}>{x}</li>
                  ))}
                </ul>
              )}
            </Card>
          )}

          <Card className="p-3 lg:col-span-1">
            <h3 className="text-sm font-semibold mb-2">Resilience</h3>
            <dl className="grid grid-cols-2 gap-x-2 gap-y-1 text-xs">
              <dt className="text-muted-foreground">Resilience</dt>
              <dd className="font-mono">{bundle.resilience.resilience_score.toFixed(2)}</dd>
              <dt className="text-muted-foreground">Redundancy</dt>
              <dd className="font-mono">{bundle.resilience.redundancy_score.toFixed(2)}</dd>
              <dt className="text-muted-foreground">Partition</dt>
              <dd className="font-mono">{bundle.resilience.partition_risk_score.toFixed(2)}</dd>
              <dt className="text-muted-foreground">Confidence</dt>
              <dd>{bundle.resilience.confidence.level}</dd>
            </dl>
            <p className="mt-2 text-xs leading-snug">{bundle.resilience.next_best_move_summary}</p>
            <ul className="mt-2 list-disc list-inside text-[11px] text-muted-foreground space-y-0.5 max-h-28 overflow-y-auto">
              {bundle.resilience.fragility_explanation.map((x, i) => (
                <li key={i}>{x}</li>
              ))}
            </ul>
          </Card>

          <Card className="p-3 lg:col-span-1">
            <h3 className="text-sm font-semibold mb-2">Ranked plans</h3>
            {bundle.ranked_next_plans.length === 0 ? (
              <p className="text-xs text-muted-foreground">No recommendations in this snapshot.</p>
            ) : (
              <ul className="space-y-2 max-h-[320px] overflow-y-auto pr-1">
                {bundle.ranked_next_plans.slice(0, 8).map((r) => (
                  <li key={r.id} className="text-xs border-b border-border/40 pb-2 last:border-0">
                    <div className="flex items-center gap-1.5 flex-wrap">
                      <span className="font-medium text-foreground">
                        {r.rank}. {r.title}
                      </span>
                      <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                        {r.verdict}
                      </Badge>
                      <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
                        {r.benefit_band}
                      </Badge>
                    </div>
                    {r.lines.slice(0, 1).map((line, i) => (
                      <p key={i} className="text-muted-foreground mt-1 leading-snug">
                        {line}
                      </p>
                    ))}
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>
      ) : (
        <>
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
              <p className="text-sm text-foreground/90 border-l-2 border-warning/35 pl-2.5 mb-3 leading-snug" role="status">
                {planningBestMoveAdvisoryNote(bundle, incidentIdParam || null)}
              </p>
              <p className="font-medium">{bn.title}</p>
              <ul className="mt-2 list-disc list-inside text-sm text-muted-foreground space-y-1">
                {(bn.summary_lines ?? []).map((line, i) => (
                  <li key={i}>{line}</li>
                ))}
              </ul>
              {bn.wait_observe_rationale && (
                <p className="mt-3 text-sm text-warning">
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
        </>
      )}

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
