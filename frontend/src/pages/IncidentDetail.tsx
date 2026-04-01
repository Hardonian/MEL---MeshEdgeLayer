import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams, Link, useNavigate, useSearchParams } from 'react-router-dom'
import {
  ArrowLeft,
  AlertTriangle,
  Clock,
  User,
  RefreshCw,
  Download,
  Shield,
  Activity,
  HelpCircle,
  CheckCircle2,
  XCircle,
  Eye,
  FileText,
  Zap,
  Link2,
  ChevronDown,
  ChevronUp,
  BookOpen,
  GitBranch,
  Circle,
} from 'lucide-react'
import { clsx } from 'clsx'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { CopyButton } from '@/components/ui/CopyButton'
import { useToast } from '@/components/ui/Toast'
import { useOperatorContext } from '@/hooks/useOperatorContext'
import { useControlStatus } from '@/hooks/useApi'
import { formatTimestamp, formatRelativeTime, type ControlActionRecord, type Incident } from '@/types/api'
import {
  evidenceStrengthLabel,
  guidanceConfidenceLabel,
  runbookStrengthOperatorLabel,
  wirelessConfidencePostureLabel,
  wirelessEvidencePostureLabel,
} from '@/utils/evidenceSemantics'
import { controlActionExecPhase } from '@/utils/controlActionPhase'

// ─── Types ────────────────────────────────────────────────────────────────────

interface ReplaySegment {
  event_time: string
  event_type: string
  event_id?: string
  summary: string
  knowledge_posture: string
  event_class?: string
  actor_id?: string
  severity?: string
  scope_posture?: string
  timing_posture?: string
  resource_id?: string
  details?: Record<string, unknown>
  evidence_refs?: string[]
}

interface ReplayView {
  kind: string
  incident_id: string
  incident: Incident
  replay_segments?: ReplaySegment[]
  knowledge_timeline?: ReplaySegment[]
  replay_meta?: {
    schema_version?: string
    window_from?: string
    window_to?: string
    timeline_event_count?: number
    recommendation_outcome_count?: number
    combined_segment_count?: number
    sparse_timeline?: boolean
    ordering_posture_note?: string
  }
  recommendation_outcomes?: Array<{
    id: string
    recommendation_id: string
    outcome: string
    actor_id?: string
    note?: string
    created_at: string
  }>
  bounded_counterfactual_ranking?: {
    statement: string
    top?: Array<{ id: string; rank_score: number; strength: string }>
    second?: Array<{ id: string; rank_score: number; strength: string }>
  }
  truth_note?: string
  generated_at?: string
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function toWords(v: string | undefined) {
  return (v || '').replace(/_/g, ' ').trim()
}

function postureColor(posture: string): string {
  if (posture.includes('operator') || posture.includes('adjudicat')) return 'text-info'
  if (posture.includes('control')) return 'text-warning'
  if (posture.includes('observed')) return 'text-success'
  return 'text-muted-foreground'
}

function postureLabel(posture: string): string {
  switch (posture) {
    case 'observed_persisted_event': return 'Observed'
    case 'observed_operator_or_system_event': return 'Operator / system'
    case 'observed_control_plane_event': return 'Control plane'
    case 'observed_control_lifecycle_event': return 'Control lifecycle'
    default: return toWords(posture) || 'Unknown'
  }
}

/** Groups segments for replay filter chips (driven by backend event_class when present). */
function replayFilterBucket(seg: ReplaySegment): 'incident' | 'control' | 'workflow' | 'operator' | 'evidence' {
  const c = (seg.event_class || '').trim()
  switch (c) {
    case 'control_action':
    case 'control_lifecycle':
      return 'control'
    case 'workflow':
    case 'handoff':
      return 'workflow'
    case 'operator_annotation':
    case 'operator_adjudication':
      return 'operator'
    case 'evidence_export':
    case 'imported_evidence':
      return 'evidence'
    default:
      return 'incident'
  }
}

const REPLAY_FILTER_OPTIONS = [
  { id: 'all' as const, label: 'All' },
  { id: 'incident' as const, label: 'Incident' },
  { id: 'control' as const, label: 'Control' },
  { id: 'workflow' as const, label: 'Workflow / handoff' },
  { id: 'operator' as const, label: 'Notes / outcomes' },
  { id: 'evidence' as const, label: 'Evidence / import' },
]

type ReplayFilterId = typeof REPLAY_FILTER_OPTIONS[number]['id']

function eventClassShortLabel(seg: ReplaySegment): string {
  const c = (seg.event_class || '').trim()
  if (!c) return toWords(seg.event_type)
  return toWords(c)
}

function severityVariant(s?: string): 'critical' | 'warning' | 'secondary' {
  if (s === 'critical') return 'critical'
  if (s === 'high') return 'warning'
  return 'secondary'
}

function stateVariant(s?: string): 'success' | 'outline' {
  if (s === 'resolved' || s === 'closed') return 'success'
  return 'outline'
}

const WORKFLOW_REVIEW_OPTIONS = [
  { value: 'open', label: 'Open' },
  { value: 'acknowledged', label: 'Acknowledged' },
  { value: 'investigating', label: 'Investigating' },
  { value: 'mitigated', label: 'Mitigated' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'follow_up_needed', label: 'Follow-up needed' },
  { value: 'pending_review', label: 'Pending review' },
  { value: 'resolved_review', label: 'Resolved (review)' },
  { value: 'closed_review', label: 'Closed (review)' },
] as const

function evidenceStrengthVariant(s?: string): 'success' | 'warning' | 'secondary' {
  if (s === 'strong') return 'success'
  if (s === 'moderate') return 'warning'
  return 'secondary'
}

function outcomeVariant(o: string): 'success' | 'critical' | 'secondary' {
  if (o === 'improvement_observed') return 'success'
  if (o === 'deterioration_observed') return 'critical'
  return 'secondary'
}

function proofpackCompletenessVariant(completeness: string): 'success' | 'warning' | 'secondary' {
  if (completeness === 'complete') return 'success'
  if (completeness === 'partial') return 'warning'
  return 'secondary'
}

function defaultProofpackFilename(id: string) {
  return `proofpack-${id || 'incident'}.json`
}

function incidentTopologyFocusNodeNum(inc: Incident): number | null {
  const rt = (inc.resource_type || '').toLowerCase()
  const rid = (inc.resource_id || '').trim()
  if (rt === 'mesh_node' || rt === 'node') {
    const n = parseInt(rid.replace(/\D/g, '') || '0', 10)
    if (Number.isFinite(n) && n > 0) return n
  }
  for (const d of inc.intelligence?.implicated_domains ?? []) {
    if ((d.domain || '').toLowerCase() !== 'mesh_topology') continue
    for (const ref of d.evidence_refs ?? []) {
      const m = /^node[:_]?(\d+)$/i.exec(ref.trim())
      if (m) {
        const n = parseInt(m[1], 10)
        if (Number.isFinite(n) && n > 0) return n
      }
    }
  }
  return null
}

function InvestigationGuidePanel({ inc }: { inc: Incident }) {
  const intel = inc.intelligence
  if (!intel) return null

  const topoNum = incidentTopologyFocusNodeNum(inc)
  const evPosture = intel.wireless_context ? wirelessEvidencePostureLabel(intel.wireless_context.evidence_posture) : null
  const confPosture = intel.wireless_context ? wirelessConfidencePostureLabel(intel.wireless_context.confidence_posture) : null
  const gaps = [...(intel.sparsity_markers ?? []), ...(intel.wireless_context?.evidence_gaps ?? [])]
  const inspectNext = intel.wireless_context?.inspect_next ?? []

  return (
    <Card data-testid="incident-investigation-guide">
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Investigation guide (bounded)</CardTitle>
        <p className="text-xs text-muted-foreground mt-1">
          Deterministic checklist from stored intelligence — not automation or root-cause AI. Verify against transports, replay, and topology
          before control actions.
        </p>
      </CardHeader>
      <CardContent className="space-y-4 pt-0">
        <div className="flex flex-wrap gap-2 items-start">
          <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wide w-full sm:w-auto">Evidence posture</span>
          <span title={evidenceStrengthLabel(intel.evidence_strength)} className="inline-flex">
            <Badge variant={evidenceStrengthVariant(intel.evidence_strength)}>{intel.evidence_strength ?? 'unknown'} strength</Badge>
          </span>
          {intel.degraded && <Badge variant="warning">Degraded intel</Badge>}
          {evPosture && <Badge variant={evPosture.variant}>{evPosture.label}</Badge>}
          {confPosture && <Badge variant={confPosture.variant}>{confPosture.label}</Badge>}
        </div>

        {(intel.degraded_reasons?.length ?? 0) > 0 && (
          <div className="rounded-lg border border-warning/25 bg-warning/5 px-3 py-2 text-xs">
            <p className="font-medium text-foreground">What not to assume</p>
            <ul className="mt-1 list-disc pl-4 text-muted-foreground space-y-0.5">
              {intel.degraded_reasons!.map((r, i) => (
                <li key={i}>{r}</li>
              ))}
            </ul>
          </div>
        )}

        {gaps.length > 0 && (
          <div>
            <p className="text-xs font-semibold text-foreground mb-1">What remains missing / sparse</p>
            <ul className="text-xs text-muted-foreground list-disc pl-4 space-y-0.5">
              {gaps.slice(0, 10).map((g, i) => (
                <li key={i}>{g}</li>
              ))}
            </ul>
          </div>
        )}

        {inspectNext.length > 0 && (
          <div>
            <p className="text-xs font-semibold text-foreground mb-1">Verify first (from wireless context)</p>
            <ul className="text-xs text-muted-foreground list-disc pl-4 space-y-0.5">
              {inspectNext.slice(0, 6).map((x, i) => (
                <li key={i}>{x}</li>
              ))}
            </ul>
          </div>
        )}

        <div className="flex flex-wrap gap-2 pt-1 border-t border-border/40">
          <Link
            to={`/incidents/${encodeURIComponent(inc.id)}?replay=1`}
            className="inline-flex items-center gap-1 rounded-lg border border-border/70 bg-card/50 px-2.5 py-1.5 text-xs font-medium text-primary hover:bg-muted/40"
          >
            <Activity className="h-3.5 w-3.5" />
            Replay / timeline
          </Link>
          <Link
            to={`/topology?incident=${encodeURIComponent(inc.id)}&filter=incident_focus${topoNum != null ? `&select=${topoNum}` : ''}`}
            className="inline-flex items-center gap-1 rounded-lg border border-border/70 bg-card/50 px-2.5 py-1.5 text-xs font-medium text-primary hover:bg-muted/40"
          >
            <GitBranch className="h-3.5 w-3.5" />
            Topology{topoNum != null ? ` (node ${topoNum})` : ''}
          </Link>
          <Link
            to={`/planning?incident=${encodeURIComponent(inc.id)}`}
            className="inline-flex items-center gap-1 rounded-lg border border-border/70 bg-card/50 px-2.5 py-1.5 text-xs font-medium text-primary hover:bg-muted/40"
          >
            Planning
          </Link>
          <Link
            to={`/control-actions?incident=${encodeURIComponent(inc.id)}`}
            className="inline-flex items-center gap-1 rounded-lg border border-border/70 bg-card/50 px-2.5 py-1.5 text-xs font-medium text-primary hover:bg-muted/40"
          >
            Control queue
          </Link>
          <Link
            to="/diagnostics"
            className="inline-flex items-center gap-1 rounded-lg border border-border/70 bg-card/50 px-2.5 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted/40 hover:text-foreground"
          >
            Support bundle
          </Link>
        </div>
      </CardContent>
    </Card>
  )
}

function filenameFromDisposition(cd: string | null, fallback: string): string {
  if (!cd) return fallback
  const m = cd.match(/filename\*?=(?:UTF-8''|")?([^";]+)/i)
  if (!m?.[1]) return fallback
  try { return decodeURIComponent(m[1].replace(/"/g, '').trim()) } catch { return fallback }
}

// ─── Proofpack download button ────────────────────────────────────────────────

function ProofpackButton({ incidentId }: { incidentId: string }) {
  const [state, setState] = useState<'idle' | 'loading' | 'error'>('idle')
  const [err, setErr] = useState('')

  async function download() {
    setState('loading')
    setErr('')
    try {
      const res = await fetch(`/api/v1/incidents/${encodeURIComponent(incidentId)}/proofpack?download=true`)
      if (!res.ok) {
        setErr(res.status === 403 ? 'Insufficient permissions.' : res.status === 404 ? 'Not found.' : `HTTP ${res.status}`)
        setState('error')
        return
      }
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filenameFromDisposition(res.headers.get('content-disposition'), defaultProofpackFilename(incidentId))
      document.body.appendChild(a); a.click(); a.remove()
      URL.revokeObjectURL(url)
      setState('idle')
    } catch {
      setErr('Network error.')
      setState('error')
    }
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <button type="button" onClick={() => void download()} disabled={state === 'loading'} className="button-secondary text-xs">
        <Download className="h-3.5 w-3.5" />
        {state === 'loading' ? 'Assembling…' : 'Export proofpack'}
      </button>
      <span className="text-[10px] text-muted-foreground/60">Snapshot at request-time. Review evidence_gaps.</span>
      {state === 'error' && <span className="text-xs text-critical">{err}</span>}
    </div>
  )
}

// ─── Proofpack completeness panel ─────────────────────────────────────────────

function LinkedControlActionsPanel({ inc }: { inc: Incident }) {
  const ctx = useOperatorContext()
  const { data: ctrlData, refresh: refreshCtrl } = useControlStatus()
  const linked = useMemo(() => inc.linked_control_actions ?? [], [inc.linked_control_actions])
  const canReadActions = ctx.trustUI?.read_actions === true || ctx.capabilities?.includes('read_actions')
  const emergencyOff = ctrlData?.emergency_disable === true
  const matrix = ctrlData?.reality_matrix ?? []

  useEffect(() => {
    void refreshCtrl()
  }, [inc.id, refreshCtrl])

  const grouped = useMemo(() => {
    const awaiting: typeof linked = []
    const inFlight: typeof linked = []
    const done: typeof linked = []
    for (const a of linked) {
      const ls = (a.lifecycle_state || '').toLowerCase()
      if (ls === 'pending_approval') awaiting.push(a)
      else if (ls === 'pending' || ls === 'running') inFlight.push(a)
      else done.push(a)
    }
    return { awaiting, inFlight, done }
  }, [linked])

  function matrixRowFor(type: string) {
    return matrix.find((m) => m.action_type === type)
  }

  if (!canReadActions && linked.length === 0) {
    return (
      <div className="rounded-xl border border-border/60 bg-muted/10 p-4">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-1">
          <Zap className="h-3.5 w-3.5" />
          Linked control actions
        </div>
        <p className="text-xs text-muted-foreground">
          Your session may lack read_actions — open the control queue with appropriate credentials to see incident-linked rows.
        </p>
        <Link
          to={`/control-actions?incident=${encodeURIComponent(inc.id)}`}
          className="mt-2 inline-flex text-xs font-semibold text-primary hover:underline"
        >
          Control queue (filtered) →
        </Link>
      </div>
    )
  }

  return (
    <div className="rounded-xl border border-border/60 bg-muted/10 p-4 space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
          <Zap className="h-3.5 w-3.5" />
          Control actions linked to this incident
        </div>
        <Link
          to={`/control-actions?incident=${encodeURIComponent(inc.id)}`}
          className="text-[11px] font-semibold text-primary hover:underline"
        >
          Full queue →
        </Link>
      </div>
      <p className="text-[11px] text-muted-foreground leading-snug">
        Rows where <code className="font-mono text-[10px]">incident_id</code> matches this incident. Approval, queue, and execution remain
        separate states — see lifecycle on each row.
      </p>
      {emergencyOff && (
        <p className="text-xs text-warning border border-warning/25 rounded-lg px-3 py-2 bg-warning/5">
          Control emergency disable is on for this instance — new execution may be blocked regardless of approval state.
        </p>
      )}
      {linked.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No linked control rows yet. If you expect actions, check the queue; linkage requires{' '}
          <code className="font-mono text-xs">incident_id</code> on the action record.
        </p>
      ) : (
        <div className="space-y-3">
          {grouped.awaiting.length > 0 && (
            <div>
              <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-warning mb-1.5">Awaiting approval</p>
              <ul className="space-y-2">
                {grouped.awaiting.map((a) => (
                  <LinkedActionRow key={a.id} incidentId={inc.id} action={a} matrixRow={matrixRowFor(a.action_type)} />
                ))}
              </ul>
            </div>
          )}
          {grouped.inFlight.length > 0 && (
            <div>
              <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-info mb-1.5">Queued / executing</p>
              <ul className="space-y-2">
                {grouped.inFlight.map((a) => (
                  <LinkedActionRow key={a.id} incidentId={inc.id} action={a} matrixRow={matrixRowFor(a.action_type)} />
                ))}
              </ul>
            </div>
          )}
          {grouped.done.length > 0 && (
            <div>
              <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground mb-1.5">Completed / terminal</p>
              <ul className="space-y-2">
                {grouped.done.map((a) => (
                  <LinkedActionRow key={a.id} incidentId={inc.id} action={a} matrixRow={matrixRowFor(a.action_type)} />
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function LinkedActionRow({
  incidentId,
  action: a,
  matrixRow,
}: {
  incidentId: string
  action: ControlActionRecord
  matrixRow?: { reversible?: boolean; blast_radius_class?: string; notes?: string; advisory_only?: boolean }
}) {
  const phase = controlActionExecPhase(a)
  const rev = matrixRow?.reversible === true ? 'Reversible (policy matrix)' : matrixRow?.reversible === false ? 'Treat as hard to reverse' : null
  const blast = matrixRow?.blast_radius_class && matrixRow.blast_radius_class !== 'unknown' ? matrixRow.blast_radius_class : null

  return (
    <li className="rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="font-medium text-foreground">{a.action_type}</p>
          <p className="font-mono text-[10px] text-muted-foreground/80 truncate mt-0.5">{a.id}</p>
        </div>
        <Badge variant={phase.variant}>{phase.label}</Badge>
      </div>
      <div className="mt-1.5 flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-muted-foreground">
        {a.result && <span>result: <span className="text-foreground">{a.result}</span></span>}
        {blast && (
          <span title={matrixRow?.notes || undefined}>
            blast: <span className="text-foreground">{blast}</span>
          </span>
        )}
        {rev && <span>{rev}</span>}
        {matrixRow?.advisory_only && <span className="text-warning">advisory-only type</span>}
      </div>
      <div className="mt-2 flex flex-wrap gap-2">
        <Link
          to={`/control-actions?incident=${encodeURIComponent(incidentId)}`}
          className="text-[11px] font-semibold text-primary hover:underline"
        >
          Open in queue
        </Link>
        <a
          href={`/api/v1/control/actions/${encodeURIComponent(a.id)}/inspect`}
          className="text-[11px] font-semibold text-muted-foreground hover:text-foreground"
          target="_blank"
          rel="noopener noreferrer"
        >
          Inspect API (new tab)
        </a>
      </div>
    </li>
  )
}

function ProofpackCompletenessPanel({ inc }: { inc: Incident }) {
  const trace = inc.intelligence?.action_outcome_trace
  const wirelessGaps = inc.intelligence?.wireless_context?.evidence_gaps ?? []
  const sparsityMarkers = inc.intelligence?.sparsity_markers ?? []
  const isDegraded = inc.intelligence?.degraded === true
  const degradedReasons = inc.intelligence?.degraded_reasons ?? []

  const completeness = trace?.completeness ?? (isDegraded ? 'partial' : 'unavailable')
  const variant = proofpackCompletenessVariant(completeness)

  const allGaps: string[] = [
    ...wirelessGaps,
    ...sparsityMarkers,
    ...degradedReasons.map((r) => toWords(r)),
  ].filter(Boolean)

  const snapshotTotal = trace?.expected_snapshot_writes ?? 0
  const snapshotPersisted = trace?.persisted_snapshot_count ?? 0
  const snapshotFailed = trace?.snapshot_write_failures ?? 0
  const pct = snapshotTotal > 0 ? Math.round((snapshotPersisted / snapshotTotal) * 100) : null

  return (
    <div className="rounded-xl border border-border/60 bg-muted/10 p-4 space-y-3">
      <div className="flex items-center justify-between flex-wrap gap-2">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
          <Shield className="h-3.5 w-3.5" />
          Proofpack completeness
        </div>
        <Badge variant={variant}>{toWords(completeness)}</Badge>
      </div>

      {trace && (
        <div className="space-y-2">
          {pct !== null && (
            <div>
              <div className="flex justify-between text-[11px] text-muted-foreground mb-1">
                <span>Snapshot coverage</span>
                <span>{snapshotPersisted}/{snapshotTotal} ({pct}%)</span>
              </div>
              <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                <div
                  className={clsx('h-full rounded-full transition-all', pct === 100 ? 'bg-success' : pct > 50 ? 'bg-warning' : 'bg-critical')}
                  style={{ width: `${pct}%` }}
                />
              </div>
            </div>
          )}
          <div className="flex flex-wrap gap-2 text-xs">
            <span className="text-muted-foreground">Retrieval: <span className="text-foreground">{trace.snapshot_retrieval_status}</span></span>
            {snapshotFailed > 0 && <Badge variant="warning">write failures: {snapshotFailed}</Badge>}
          </div>
          {trace.snapshot_retrieval_error && (
            <p className="text-xs text-warning">{trace.snapshot_retrieval_error}</p>
          )}
        </div>
      )}

      {allGaps.length > 0 && (
        <div className="rounded-lg border border-warning/25 bg-warning/5 px-3 py-2 text-xs">
          <p className="font-medium text-foreground mb-1">Evidence gaps / sparsity</p>
          <ul className="space-y-0.5">
            {allGaps.slice(0, 6).map((g, i) => (
              <li key={i} className="flex items-start gap-1.5 text-muted-foreground">
                <AlertTriangle className="h-3 w-3 shrink-0 mt-0.5 text-warning" />
                {g}
              </li>
            ))}
          </ul>
        </div>
      )}

      {completeness === 'unavailable' && !trace && (
        <p className="text-xs text-muted-foreground">
          No snapshot traceability data available for this incident. Proofpack may still export available evidence.
        </p>
      )}

      <ProofpackButton incidentId={inc.id} />
      <p className="text-[11px] text-muted-foreground">
        For host/runtime continuity (not incident proof), use{' '}
        <Link to="/diagnostics" className="font-medium text-primary hover:underline">
          Diagnostics → support bundle
        </Link>
        .
      </p>
    </div>
  )
}

// ─── Replay timeline ──────────────────────────────────────────────────────────

function ReplayTimeline({ segments, truthNote, generatedAt, replayMeta }: {
  segments: ReplaySegment[]
  truthNote?: string
  generatedAt?: string
  replayMeta?: ReplayView['replay_meta']
}) {
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const [filter, setFilter] = useState<ReplayFilterId>('all')
  const [newestFirst, setNewestFirst] = useState(false)

  const filtered = useMemo(() => {
    if (filter === 'all') return segments
    return segments.filter((s) => replayFilterBucket(s) === filter)
  }, [segments, filter])

  const ordered = useMemo(() => {
    if (!newestFirst) return filtered
    return [...filtered].reverse()
  }, [filtered, newestFirst])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement || e.target instanceof HTMLSelectElement) return
      if (!e.altKey) return
      const n = Number.parseInt(e.key, 10)
      if (n >= 0 && n < REPLAY_FILTER_OPTIONS.length) {
        e.preventDefault()
        setFilter(REPLAY_FILTER_OPTIONS[n]!.id)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  function toggle(i: number) {
    setExpanded((prev) => {
      const next = new Set(prev)
      next.has(i) ? next.delete(i) : next.add(i)
      return next
    })
  }

  if (segments.length === 0) {
    return (
      <div className="rounded-xl border border-border/60 bg-muted/10 p-4">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2">
          <GitBranch className="h-3.5 w-3.5" />
          Incident replay / timeline
        </div>
        <p className="text-sm text-muted-foreground">
          No timeline events in the replay window ({replayMeta?.window_from ?? '—'} → {replayMeta?.window_to ?? '—'}). Evidence may have been pruned, notes may use a different ref, or the incident is outside the bounded window.
        </p>
        {replayMeta?.ordering_posture_note && (
          <p className="mt-2 text-[11px] text-muted-foreground/70 border-l-2 border-muted pl-2.5">{replayMeta.ordering_posture_note}</p>
        )}
      </div>
    )
  }

  return (
    <div className="rounded-xl border border-border/60 bg-muted/10 p-4 space-y-3" role="region" aria-label="Incident replay timeline">
      <div className="flex flex-wrap items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
        <GitBranch className="h-3.5 w-3.5" />
        Incident replay / timeline
        <span className="ml-auto font-normal normal-case tracking-normal">
          {ordered.length}{filter !== 'all' ? ` / ${segments.length}` : ''} events
        </span>
      </div>

      {replayMeta && (
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-[10px] text-muted-foreground/80 font-mono">
          {replayMeta.window_from && replayMeta.window_to && (
            <span>Window: {replayMeta.window_from.slice(0, 16)}… → {replayMeta.window_to.slice(0, 16)}…</span>
          )}
          {replayMeta.timeline_event_count != null && (
            <span>DB timeline rows: {replayMeta.timeline_event_count}</span>
          )}
          {replayMeta.sparse_timeline && (
            <span className="text-warning">Sparse timeline</span>
          )}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-1.5" role="toolbar" aria-label="Replay filters">
        {REPLAY_FILTER_OPTIONS.map((o, idx) => (
          <button
            key={o.id}
            type="button"
            onClick={() => setFilter(o.id)}
            title={`Alt+${idx}`}
            className={clsx(
              'rounded-md border px-2 py-1 text-[10px] font-semibold transition-colors',
              filter === o.id
                ? 'border-primary/50 bg-primary/10 text-foreground'
                : 'border-border/50 bg-card/40 text-muted-foreground hover:border-border/80',
            )}
          >
            {o.label}
            <span className="ml-1 font-mono text-muted-foreground/50">{idx}</span>
          </button>
        ))}
        <button
          type="button"
          onClick={() => setNewestFirst((v) => !v)}
          className="ml-auto rounded-md border border-border/50 bg-card/40 px-2 py-1 text-[10px] font-semibold text-muted-foreground hover:border-border/80"
        >
          {newestFirst ? 'Order: newest first' : 'Order: oldest first'}
        </button>
      </div>

      {truthNote && (
        <p className="text-[11px] text-muted-foreground/70 border-l-2 border-muted pl-2.5 leading-snug">
          {truthNote}
        </p>
      )}
      {replayMeta?.ordering_posture_note && (
        <p className="text-[10px] text-muted-foreground/60 border-l-2 border-border/40 pl-2.5 leading-snug">
          {replayMeta.ordering_posture_note}
        </p>
      )}

      {filter !== 'all' && ordered.length === 0 && (
        <p className="text-xs text-muted-foreground">No events in this filter; try &quot;All&quot;.</p>
      )}

      <div className="relative">
        <div className="absolute left-[6px] top-0 bottom-0 w-px bg-border/60" aria-hidden />

        <ol className="space-y-0">
          {ordered.map((seg, i) => {
            const isExp = expanded.has(i)
            const hasRefs = (seg.evidence_refs?.length ?? 0) > 0
            const hasDetails = seg.details != null && Object.keys(seg.details).length > 0
            return (
              <li key={`${seg.event_id ?? 'ev'}-${i}`} className="relative flex gap-3 pl-6 pb-4 last:pb-0">
                <Circle className={clsx('absolute left-0 h-3.5 w-3.5 shrink-0 fill-current top-0.5', postureColor(seg.knowledge_posture))} />

                <div className="flex-1 min-w-0">
                  <div className="flex flex-wrap items-start gap-x-3 gap-y-0.5">
                    <span className="text-xs font-medium text-foreground leading-snug">{seg.summary || toWords(seg.event_type)}</span>
                    <Badge variant="outline" className="text-[9px] px-1 py-0 font-mono normal-case tracking-normal">
                      {eventClassShortLabel(seg)}
                    </Badge>
                    <span className={clsx('text-[10px] font-semibold uppercase tracking-wide shrink-0', postureColor(seg.knowledge_posture))}>
                      {postureLabel(seg.knowledge_posture)}
                    </span>
                  </div>
                  <div className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-[11px] text-muted-foreground">
                    <span className="inline-flex items-center gap-1" title={seg.event_time ? formatTimestamp(seg.event_time) : undefined}>
                      <Clock className="h-3 w-3" />
                      {seg.event_time ? (
                        <>
                          <span>{formatRelativeTime(seg.event_time)}</span>
                          <span className="text-muted-foreground/50 hidden sm:inline">· {formatTimestamp(seg.event_time)}</span>
                        </>
                      ) : '—'}
                    </span>
                    {seg.actor_id && <span className="text-muted-foreground/70">actor {seg.actor_id}</span>}
                    {seg.timing_posture && seg.timing_posture !== 'local_ordered' && (
                      <span className="text-warning/80 text-[10px]">timing: {seg.timing_posture}</span>
                    )}
                    {seg.scope_posture && seg.scope_posture !== 'local' && (
                      <span className="text-[10px] text-muted-foreground/70">scope: {seg.scope_posture}</span>
                    )}
                    <code className="text-muted-foreground/60">{toWords(seg.event_type)}</code>
                    {(hasRefs || hasDetails) && (
                      <button
                        type="button"
                        onClick={() => toggle(i)}
                        className="inline-flex items-center gap-0.5 text-[10px] text-muted-foreground hover:text-foreground"
                        aria-expanded={isExp}
                      >
                        {isExp ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                        details
                      </button>
                    )}
                  </div>
                  {isExp && hasRefs && (
                    <ul className="mt-1.5 space-y-0.5">
                      {seg.evidence_refs!.map((ref) => (
                        <li key={ref} className="text-[10px] font-mono text-muted-foreground/70">{ref}</li>
                      ))}
                    </ul>
                  )}
                  {isExp && hasDetails && (
                    <pre className="mt-1.5 max-h-32 overflow-auto rounded border border-border/40 bg-muted/20 p-2 text-[10px] font-mono text-muted-foreground/80 whitespace-pre-wrap">
                      {JSON.stringify(seg.details, null, 2)}
                    </pre>
                  )}
                </div>
              </li>
            )
          })}
        </ol>
      </div>

      {generatedAt && (
        <p className="text-[10px] text-muted-foreground/50">Replay assembled {formatTimestamp(generatedAt)}</p>
      )}
    </div>
  )
}

// ─── Section wrapper ──────────────────────────────────────────────────────────

function Section({ title, icon, children }: { title: string; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
        {icon}
        {title}
      </div>
      {children}
    </div>
  )
}

function buildHandoffStructured(inc: Incident) {
  const intel = inc.intelligence
  const gaps = [
    ...(intel?.wireless_context?.evidence_gaps ?? []),
    ...(intel?.sparsity_markers ?? []),
  ]
  const similar = (intel?.similar_incidents ?? []).slice(0, 5).map((s) => ({
    incident_id: s.incident_id,
    title: s.title,
    state: s.state,
    occurred_at: s.occurred_at,
    weighted_score: s.weighted_score,
    insufficient_evidence: s.insufficient_evidence,
    match_explanation: s.match_explanation?.slice(0, 4),
    matched_dimensions: s.matched_dimensions,
  }))
  return {
    kind: 'mel_handoff_summary/v1',
    generated_note: 'Structured continuity snapshot — not a proofpack; use Export proofpack or escalation bundle for evidence chain.',
    incident: {
      id: inc.id,
      title: inc.title,
      state: inc.state,
      review_state: inc.review_state,
      severity: inc.severity,
      category: inc.category,
      resource: { type: inc.resource_type, id: inc.resource_id },
      occurred_at: inc.occurred_at,
      updated_at: inc.updated_at,
      resolved_at: inc.resolved_at,
      owner_actor_id: inc.owner_actor_id,
    },
    narrative: {
      summary: inc.summary,
      handoff_summary: inc.handoff_summary,
      investigation_notes: inc.investigation_notes,
      resolution_summary: inc.resolution_summary,
      lessons_learned: inc.lessons_learned,
      closeout_reason: inc.closeout_reason,
    },
    intelligence_posture: intel
      ? {
          evidence_strength: intel.evidence_strength,
          degraded: intel.degraded,
          degraded_reasons: intel.degraded_reasons,
          signature_label: intel.signature_label,
          signature_match_count: intel.signature_match_count,
        }
      : undefined,
    uncertainty: {
      evidence_and_sparsity_gaps: gaps,
    },
    pending: {
      pending_action_ids: inc.pending_actions?.filter(Boolean) ?? [],
    },
    next_checks: (intel?.investigate_next ?? []).slice(0, 8).map((g) => ({
      id: g.id,
      title: g.title,
      rationale: g.rationale,
      confidence: g.confidence,
    })),
    similar_incidents_compact: similar,
    deep_links: {
      incident: `/incidents/${inc.id}`,
      control_actions: `/control-actions?incident=${encodeURIComponent(inc.id)}`,
      topology: `/topology?incident=${encodeURIComponent(inc.id)}&filter=incident_focus`,
      planning: `/planning?incident=${encodeURIComponent(inc.id)}`,
      replay: `/incidents/${encodeURIComponent(inc.id)}?replay=1`,
      diagnostics_support_bundle: '/diagnostics',
    },
  }
}

function buildHandoffExportText(inc: Incident): string {
  const intel = inc.intelligence
  const lines: string[] = [
    'MEL — incident handoff summary (paste into ticket or runbook)',
    `Incident: ${inc.title || inc.id}`,
    `ID: ${inc.id}`,
    `State (system): ${inc.state || 'unknown'}`,
    `Review / workflow: ${inc.review_state || 'open'}`,
    `Severity: ${inc.severity || '—'}`,
    `Occurred: ${inc.occurred_at || '—'}`,
    `Updated: ${inc.updated_at || '—'}`,
    `Resource: ${inc.resource_type || '—'} / ${inc.resource_id || '—'}`,
    '',
    'What we know (bounded):',
    inc.summary || '(no summary)',
    '',
  ]
  if (intel?.evidence_strength) {
    lines.push(`Evidence strength (intel): ${intel.evidence_strength}`)
  }
  if (intel?.signature_match_count != null && intel.signature_match_count > 1) {
    lines.push(`Signature recurrence (instance history): ${intel.signature_match_count} matches — structural bucket, not causal.`)
  }
  if (intel?.degraded) {
    lines.push('Intelligence degraded: yes (treat guidance as non-causal)')
    if (intel.degraded_reasons?.length) {
      lines.push(`Reasons: ${intel.degraded_reasons.join('; ')}`)
    }
  }
  if ((intel?.sparsity_markers?.length ?? 0) > 0) {
    lines.push(`Sparsity markers: ${intel!.sparsity_markers!.join('; ')}`)
  }
  lines.push('')
  lines.push('Recorded handoff narrative:')
  lines.push(inc.handoff_summary || '(none)')
  lines.push('')
  lines.push('Investigation notes:')
  lines.push(inc.investigation_notes || '(none)')
  lines.push('')
  lines.push('Resolution / closeout (if any):')
  lines.push(inc.resolution_summary || '(none)')
  if (inc.lessons_learned) lines.push(`Lessons: ${inc.lessons_learned}`)
  if (inc.closeout_reason) lines.push(`Closeout: ${inc.closeout_reason}`)
  lines.push('')
  lines.push('Pending action IDs (referenced on incident):')
  const p = inc.pending_actions?.filter(Boolean) ?? []
  if (p.length === 0) lines.push('(none listed)')
  else for (const id of p) lines.push(`- ${id}`)
  lines.push('')
  lines.push('Similar prior incidents (deterministic / explainable in UI):')
  const sim = intel?.similar_incidents ?? []
  if (sim.length === 0) {
    lines.push('(none listed — may be sparse history)')
  } else {
    for (const s of sim.slice(0, 5)) {
      const expl = (s.match_explanation?.length ? s.match_explanation.join('; ') : s.similarity_reason?.join('; ')) || 'see incident detail'
      lines.push(`- ${s.incident_id} state=${s.state ?? '?'} score=${s.weighted_score != null ? s.weighted_score.toFixed(2) : 'n/a'} weak=${s.insufficient_evidence ? 'yes' : 'no'}`)
      lines.push(`  ${expl}`)
    }
  }
  lines.push('')
  lines.push('What remains uncertain:')
  if ((intel?.wireless_context?.evidence_gaps?.length ?? 0) > 0) {
    for (const g of intel!.wireless_context!.evidence_gaps!) lines.push(`- ${g}`)
  } else {
    lines.push('- See proofpack evidence_gaps and intelligence panels in MEL.')
  }
  lines.push('')
  lines.push('Next checks (suggested):')
  const next = intel?.investigate_next?.slice(0, 5) ?? []
  if (next.length === 0) {
    lines.push('- Open replay/timeline and topology for this incident window in MEL.')
  } else {
    for (const g of next) lines.push(`- ${g.title}: ${g.rationale}`)
  }
  lines.push('')
  lines.push(`Deep link: /incidents/${inc.id}`)
  lines.push('This paste export is a snapshot; canonical evidence lives in MEL proofpack / DB. Use “Handoff JSON” or escalation bundle for machine-readable continuity.')
  return lines.join('\n')
}

function WorkflowPanel({ inc, onSaved }: { inc: Incident; onSaved: () => void }) {
  const ctx = useOperatorContext()
  const { addToast } = useToast()
  const [reviewState, setReviewState] = useState(inc.review_state || 'open')
  const [notes, setNotes] = useState(inc.investigation_notes || '')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setReviewState(inc.review_state || 'open')
    setNotes(inc.investigation_notes || '')
  }, [inc.id, inc.review_state, inc.investigation_notes])

  const canWrite = ctx.trustUI?.incident_mutate === true

  async function saveWorkflow() {
    setSaving(true)
    try {
      const res = await fetch(`/api/v1/incidents/${encodeURIComponent(inc.id)}/workflow`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          review_state: reviewState,
          investigation_notes: notes,
        }),
      })
      if (res.status === 403) {
        addToast({ type: 'error', title: 'Cannot save', message: 'Missing incident update capability.' })
        return
      }
      if (!res.ok) {
        addToast({ type: 'error', title: 'Save failed', message: `HTTP ${res.status}` })
        return
      }
      addToast({ type: 'success', title: 'Workflow saved', message: 'Review state and notes persisted locally on this MEL instance.' })
      await onSaved()
    } catch {
      addToast({ type: 'error', title: 'Save failed', message: 'Network error.' })
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Workflow & investigation</CardTitle>
        <p className="text-xs text-muted-foreground mt-1">
          Review state is operator workflow on this instance (single-operator honest mode). It does not imply multi-user coordination.
        </p>
      </CardHeader>
      <CardContent className="space-y-3 pt-0">
        {!canWrite && (
          <p className="text-xs text-warning border border-warning/25 rounded-lg px-3 py-2 bg-warning/5">
            Read-only: your session cannot PATCH incident workflow. Notes and state changes require incident_mutate.
          </p>
        )}
        <div>
          <label htmlFor="mel-inc-review-state" className="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
            Review state
          </label>
          <select
            id="mel-inc-review-state"
            className="mt-1 w-full rounded-lg border border-border/60 bg-background px-3 py-2 text-sm"
            value={reviewState}
            onChange={(e) => setReviewState(e.target.value)}
            disabled={!canWrite}
          >
            {WORKFLOW_REVIEW_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor="mel-inc-notes" className="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
            Investigation notes
          </label>
          <textarea
            id="mel-inc-notes"
            className="mt-1 w-full min-h-[100px] rounded-lg border border-border/60 bg-background px-3 py-2 text-sm"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            disabled={!canWrite}
            placeholder="Observed facts, hypotheses (labeled), what you checked…"
          />
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => void saveWorkflow()}
            disabled={!canWrite || saving}
            className="button-secondary text-xs"
          >
            {saving ? 'Saving…' : 'Save workflow'}
          </button>
          <Link
            to={`/control-actions?incident=${encodeURIComponent(inc.id)}`}
            className="inline-flex items-center gap-1 rounded-lg border border-border/60 px-3 py-1.5 text-xs font-semibold hover:bg-muted/50"
          >
            <Zap className="h-3.5 w-3.5" />
            Control actions for this incident
          </Link>
        </div>
      </CardContent>
    </Card>
  )
}

function HandoffExportPanel({ inc }: { inc: Incident }) {
  const text = buildHandoffExportText(inc)
  const structured = buildHandoffStructured(inc)
  const jsonText = JSON.stringify(structured, null, 2)
  const [escState, setEscState] = useState<'idle' | 'loading' | 'error'>('idle')
  const [escErr, setEscErr] = useState('')

  async function downloadEscalationBundle() {
    setEscState('loading')
    setEscErr('')
    try {
      const res = await fetch(`/api/v1/incidents/${encodeURIComponent(inc.id)}/escalation-bundle`)
      if (!res.ok) {
        setEscErr(res.status === 403 ? 'Export disabled by policy or insufficient permissions.' : `HTTP ${res.status}`)
        setEscState('error')
        return
      }
      const data = await res.json()
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `escalation-bundle-${inc.id.slice(0, 12)}.json`
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
      setEscState('idle')
    } catch {
      setEscErr('Network error.')
      setEscState('error')
    }
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle className="text-base">Shift continuity / handoff</CardTitle>
          <div className="flex flex-wrap gap-2">
            <CopyButton value={text} label="Copy plain summary" className="button-secondary text-xs" />
            <CopyButton value={jsonText} label="Copy handoff JSON" className="button-secondary text-xs" />
          </div>
        </div>
        <p className="text-xs text-muted-foreground mt-1">
          Plain text for chat/tickets; JSON is a structured continuity snapshot. Neither replaces the proofpack for canonical evidence.
        </p>
      </CardHeader>
      <CardContent className="pt-0 space-y-3">
        <div className="rounded-lg border border-border/50 bg-muted/15 px-3 py-2.5 text-[11px] text-muted-foreground space-y-1.5">
          <p className="font-semibold text-foreground">What to export when</p>
          <ul className="list-disc pl-4 space-y-1">
            <li>
              <span className="text-foreground">Plain / JSON handoff</span> — human or machine-readable{' '}
              <em className="not-italic text-muted-foreground">continuity</em>; not canonical proof.
            </li>
            <li>
              <span className="text-foreground">Escalation bundle</span> — support-oriented package when policy allows (may include proof
              assembly summary).
            </li>
            <li>
              <span className="text-foreground">Proofpack</span> — strongest bundled evidence export MEL assembles for this incident (still
              review gaps in-file).
            </li>
            <li>
              <span className="text-foreground">Diagnostics support bundle</span> —{' '}
              <Link to="/diagnostics" className="text-primary font-medium hover:underline">
                host/runtime
              </Link>{' '}
              continuity; separate from incident proof.
            </li>
          </ul>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => void downloadEscalationBundle()}
            disabled={escState === 'loading'}
            className="button-secondary text-xs inline-flex items-center gap-1.5 min-h-[44px] sm:min-h-0 touch-manipulation"
          >
            <Download className="h-3.5 w-3.5" />
            {escState === 'loading' ? 'Downloading…' : 'Download escalation bundle'}
          </button>
          <span className="text-[10px] text-muted-foreground/70">
            Includes proofpack assembly summary + linked control rows when export policy allows.
          </span>
        </div>
        {escState === 'error' && escErr && <p className="text-xs text-critical">{escErr}</p>}
        <pre className="max-h-48 overflow-auto rounded-lg border border-border/50 bg-muted/20 p-3 text-[11px] text-muted-foreground whitespace-pre-wrap font-mono">
          {text}
        </pre>
      </CardContent>
    </Card>
  )
}

// ─── Main page ────────────────────────────────────────────────────────────────

export function IncidentDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const [inc, setInc] = useState<Incident | null>(null)
  const [replay, setReplay] = useState<ReplayView | null>(null)
  const [loading, setLoading] = useState(true)
  const [replayLoading, setReplayLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [replayError, setReplayError] = useState<string | null>(null)
  const [replayOpen, setReplayOpen] = useState(() => searchParams.get('replay') === '1')

  const load = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`/api/v1/incidents/${encodeURIComponent(id)}`)
      if (res.status === 404) throw new Error('Incident not found.')
      if (res.status === 403) throw new Error('Insufficient permissions.')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as Incident
      setInc(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load incident')
    } finally {
      setLoading(false)
    }
  }, [id])

  const loadReplay = useCallback(async () => {
    if (!id) return
    setReplayLoading(true)
    setReplayError(null)
    try {
      const res = await fetch(`/api/v1/incidents/${encodeURIComponent(id)}/replay`)
      if (!res.ok) {
        if (res.status === 503) {
          setReplayError('Replay not available on this instance.')
        } else {
          setReplayError(`HTTP ${res.status}`)
        }
        return
      }
      setReplay(await res.json() as ReplayView)
    } catch {
      setReplayError('Could not load replay data.')
    } finally {
      setReplayLoading(false)
    }
  }, [id])

  useEffect(() => { void load() }, [load])

  useEffect(() => {
    if (searchParams.get('replay') === '1') {
      setReplayOpen(true)
    }
  }, [searchParams])

  useEffect(() => {
    if (!replayOpen || !id) return
    if (replay || replayLoading || replayError) return
    void loadReplay()
  }, [replayOpen, id, replay, replayLoading, replayError, loadReplay])

  function handleReplayOpen() {
    setReplayOpen(true)
    setSearchParams((prev) => {
      const n = new URLSearchParams(prev)
      n.set('replay', '1')
      return n
    }, { replace: true })
  }

  if (loading) return <Loading message="Loading incident…" />

  if (error) {
    return (
      <div className="p-8 max-w-2xl mx-auto space-y-4">
        <Link to="/incidents" className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="h-4 w-4" /> Back to incidents
        </Link>
        <AlertCard variant="critical" title="Could not load incident" description={error} action={
          <button type="button" onClick={() => void load()} className="button-secondary text-xs">Retry</button>
        } />
      </div>
    )
  }

  if (!inc) return null

  const intel = inc.intelligence
  const hasIntel = !!intel
  const seenBefore = (intel?.signature_match_count ?? 0) > 1
  const replaySegs = replay?.replay_segments ?? replay?.knowledge_timeline ?? []
  const outcomesInTimeline = replaySegs.some((s) => s.event_type === 'recommendation_outcome')

  return (
    <div className="max-w-4xl mx-auto space-y-5 pb-12">
      {/* Back nav */}
      <div className="flex items-center gap-3 flex-wrap">
        <button
          type="button"
          onClick={() => navigate(-1)}
          className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          Back
        </button>
        <Link to="/incidents" className="text-sm text-muted-foreground hover:text-foreground">
          All incidents
        </Link>
        <span className="text-muted-foreground/40">/</span>
        <span className="text-sm text-foreground font-mono truncate max-w-[200px]">{inc.id.slice(0, 16)}…</span>
        <button
          type="button"
          onClick={() => void load()}
          className="ml-auto inline-flex items-center gap-1.5 rounded-lg border border-border/60 bg-card/50 px-2.5 py-1.5 text-xs text-muted-foreground hover:text-foreground"
        >
          <RefreshCw className="h-3.5 w-3.5" />
          Refresh
        </button>
      </div>

      {/* Header card */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex flex-wrap items-start gap-3">
            <AlertTriangle className={clsx('h-5 w-5 shrink-0 mt-0.5', inc.severity === 'critical' ? 'text-critical' : inc.severity === 'high' ? 'text-warning' : 'text-muted-foreground')} />
            <div className="flex-1 min-w-0">
              <CardTitle className="text-lg leading-snug">{inc.title || inc.id}</CardTitle>
              <div className="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <span className="font-mono">{inc.id.slice(0, 20)}</span>
                {inc.occurred_at && (
                  <span className="inline-flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    {formatRelativeTime(inc.occurred_at)}
                  </span>
                )}
                {inc.owner_actor_id && (
                  <span className="inline-flex items-center gap-1">
                    <User className="h-3 w-3" />
                    {inc.owner_actor_id}
                  </span>
                )}
              </div>
            </div>
            <div className="flex flex-wrap gap-1.5">
              {inc.state && <Badge variant={stateVariant(inc.state)}>{inc.state}</Badge>}
              {inc.severity && <Badge variant={severityVariant(inc.severity)}>{inc.severity}</Badge>}
              {hasIntel && (
                <span title={evidenceStrengthLabel(intel.evidence_strength)} className="inline-flex">
                  <Badge variant={evidenceStrengthVariant(intel.evidence_strength)}>{intel.evidence_strength} evidence</Badge>
                </span>
              )}
              {seenBefore && <Badge variant="warning">seen {intel!.signature_match_count}x</Badge>}
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-0 space-y-4">
          {inc.summary && (
            <p className="text-sm leading-relaxed text-muted-foreground">{inc.summary}</p>
          )}
          {hasIntel && (
            <div className="flex flex-wrap gap-2">
              {intel.signature_label && (
                <Badge variant="outline"><Activity className="h-3 w-3" />{intel.signature_label}</Badge>
              )}
              {intel.wireless_context && (
                <Badge variant="outline">{toWords(intel.wireless_context.classification)}</Badge>
              )}
              {(intel.similar_incidents?.length ?? 0) > 0 && (
                <Badge variant="secondary">{intel.similar_incidents!.length} similar prior incidents</Badge>
              )}
            </div>
          )}
          <div className="flex flex-wrap gap-x-6 gap-y-1 text-xs text-muted-foreground border-t border-border/40 pt-3">
            {inc.occurred_at && <span>Occurred: {formatTimestamp(inc.occurred_at)}</span>}
            {inc.updated_at && <span>Updated: {formatTimestamp(inc.updated_at)}</span>}
            {inc.resolved_at && <span>Resolved: {formatTimestamp(inc.resolved_at)}</span>}
            {inc.category && <span>Category: {inc.category}</span>}
            {inc.review_state && <span>Review: {toWords(inc.review_state)}</span>}
          </div>
        </CardContent>
      </Card>

      {/* Proofpack completeness */}
      <ProofpackCompletenessPanel inc={inc} />

      <LinkedControlActionsPanel inc={inc} />

      {hasIntel && <InvestigationGuidePanel inc={inc} />}

      <div className="grid gap-5 lg:grid-cols-2">
        <WorkflowPanel inc={inc} onSaved={() => void load()} />
        <HandoffExportPanel inc={inc} />
      </div>

      {/* Two-column body */}
      <div className="grid gap-5 lg:grid-cols-2">

        {/* Handoff + investigation */}
        <div className="space-y-4">
          {/* Handoff summary */}
          <Section title="Handoff summary" icon={<FileText className="h-3.5 w-3.5" />}>
            <div className={clsx(
              'rounded-lg border px-3 py-2.5 text-sm',
              inc.handoff_summary ? 'border-border/60 bg-card/50' : 'border-dashed border-border/50 bg-muted/20 text-muted-foreground'
            )}>
              {inc.handoff_summary || 'No handoff summary recorded.'}
            </div>
          </Section>

          {/* Investigation notes */}
          {inc.investigation_notes && (
            <Section title="Investigation notes" icon={<Eye className="h-3.5 w-3.5" />}>
              <div className="rounded-lg border border-border/60 bg-card/50 px-3 py-2.5 text-sm whitespace-pre-wrap">{inc.investigation_notes}</div>
            </Section>
          )}

          {/* Resolution */}
          {(inc.resolution_summary || inc.lessons_learned || inc.closeout_reason) && (
            <Section title="Resolution" icon={<CheckCircle2 className="h-3.5 w-3.5" />}>
              <div className="space-y-2 rounded-lg border border-border/60 bg-card/50 px-3 py-2.5 text-sm">
                {inc.resolution_summary && <p>{inc.resolution_summary}</p>}
                {inc.lessons_learned && (
                  <p className="text-muted-foreground"><span className="font-medium text-foreground">Lessons: </span>{inc.lessons_learned}</p>
                )}
                {inc.closeout_reason && (
                  <p className="text-muted-foreground"><span className="font-medium text-foreground">Closeout: </span>{toWords(inc.closeout_reason)}</p>
                )}
              </div>
            </Section>
          )}

          {/* Risks */}
          {(inc.risks?.length ?? 0) > 0 && (
            <Section title="Risks" icon={<AlertTriangle className="h-3.5 w-3.5" />}>
              <ul className="space-y-1">
                {inc.risks!.map((r, i) => (
                  <li key={i} className="flex items-start gap-2 text-xs text-muted-foreground">
                    <AlertTriangle className="h-3 w-3 shrink-0 mt-0.5 text-warning" />
                    {r}
                  </li>
                ))}
              </ul>
            </Section>
          )}

          {/* Referenced actions */}
          {(inc.pending_actions?.filter(Boolean).length ?? 0) > 0 && (
            <Section title="Referenced action IDs" icon={<Zap className="h-3.5 w-3.5" />}>
              <div className="space-y-1.5">
                {inc.pending_actions!.filter(Boolean).map((actionId) => (
                  <div key={actionId} className="flex items-center gap-2 rounded-lg border border-border/50 bg-card/40 px-2.5 py-1.5 text-xs">
                    <code className="flex-1 truncate font-mono text-muted-foreground">{actionId.slice(0, 24)}…</code>
                  </div>
                ))}
                <Link
                  to={`/control-actions?incident=${encodeURIComponent(inc.id)}`}
                  className="inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline"
                >
                  View control actions for this incident →
                </Link>
              </div>
            </Section>
          )}
        </div>

        {/* Intelligence */}
        <div className="space-y-4">
          {hasIntel && (
            <>
              {/* Similar incidents */}
              {(intel.similar_incidents?.length ?? 0) > 0 && (
                <Section title="Similar prior incidents" icon={<Link2 className="h-3.5 w-3.5" />}>
                  <div className="space-y-1.5">
                    {intel.similar_incidents!.map((s) => (
                      <Link
                        key={s.incident_id}
                        to={`/incidents/${s.incident_id}`}
                        className="block rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs hover:border-border/80 hover:bg-card/70 transition-colors"
                      >
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-mono text-muted-foreground shrink-0">{s.incident_id.slice(0, 12)}</span>
                          {s.title && <span className="flex-1 min-w-0 truncate font-medium text-foreground">{s.title}</span>}
                          {s.state && <Badge variant={s.state === 'resolved' ? 'success' : 'secondary'}>{s.state}</Badge>}
                          {s.insufficient_evidence && <Badge variant="warning">weak match</Badge>}
                          {s.match_category && <Badge variant="outline">{toWords(s.match_category)}</Badge>}
                          {s.weighted_score != null && (
                            <span className="text-muted-foreground/60 shrink-0 font-mono" title="Deterministic fingerprint score; not ML confidence">
                              {(s.weighted_score * 100).toFixed(0)}%
                            </span>
                          )}
                        </div>
                        {(s.match_explanation?.length || s.similarity_reason?.length) ? (
                          <ul className="mt-1.5 space-y-0.5 text-[10px] text-muted-foreground leading-snug border-t border-border/30 pt-1.5">
                            {(s.match_explanation ?? s.similarity_reason ?? []).slice(0, 4).map((line, i) => (
                              <li key={i} className="flex gap-1.5">
                                <span className="text-muted-foreground/40 shrink-0">·</span>
                                <span>{line}</span>
                              </li>
                            ))}
                          </ul>
                        ) : null}
                      </Link>
                    ))}
                  </div>
                </Section>
              )}

              {/* Investigate next */}
              {(intel.investigate_next?.length ?? 0) > 0 && (
                <Section title="Investigate next" icon={<HelpCircle className="h-3.5 w-3.5" />}>
                  <div className="space-y-1.5">
                    {intel.investigate_next!.slice(0, 5).map((g) => (
                      <div key={g.id} className="rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs">
                        <div className="flex flex-wrap items-center gap-2">
                          <p className="font-medium text-foreground">{g.title}</p>
                          <span title={guidanceConfidenceLabel(g.confidence)} className="inline-flex">
                            <Badge variant="outline">{g.confidence} confidence</Badge>
                          </span>
                        </div>
                        <p className="mt-0.5 text-muted-foreground">{g.rationale}</p>
                      </div>
                    ))}
                  </div>
                </Section>
              )}

              {/* Runbook recommendations */}
              {(intel.runbook_recommendations?.length ?? 0) > 0 && (
                <Section title="Runbook recommendations" icon={<BookOpen className="h-3.5 w-3.5" />}>
                  <div className="space-y-1.5">
                    {intel.runbook_recommendations!.slice(0, 4).map((r) => (
                      <div key={r.id} className="rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs">
                        <div className="flex items-center gap-2">
                          {r.rank_score != null && <span className="text-muted-foreground shrink-0">{r.rank_score.toFixed(2)}</span>}
                          <span className="font-medium text-foreground flex-1">{r.title}</span>
                          <Badge variant="outline">{toWords(r.strength)}</Badge>
                        </div>
                        <p className="mt-1 text-[10px] text-muted-foreground leading-snug">{runbookStrengthOperatorLabel(r.strength)}</p>
                        {r.rationale && <p className="mt-1 text-muted-foreground">{r.rationale}</p>}
                      </div>
                    ))}
                  </div>
                </Section>
              )}

              {/* Action outcome memory */}
              {(intel.action_outcome_memory?.length ?? 0) > 0 && (
                <Section title="Historical action outcomes" icon={<Zap className="h-3.5 w-3.5" />}>
                  <p className="text-[11px] text-muted-foreground mb-1.5">Association only — does not establish causality.</p>
                  <div className="space-y-2">
                    {intel.action_outcome_memory!.map((m) => (
                      <div key={m.action_type} className="rounded-lg border border-border/50 bg-card/40 p-3 text-xs">
                        <div className="flex flex-wrap items-center gap-1.5">
                          <span className="font-medium text-foreground">{m.action_label || toWords(m.action_type)}</span>
                          <Badge variant="outline">n={m.sample_size}</Badge>
                          <Badge variant={m.outcome_framing === 'improvement_observed' ? 'success' : m.outcome_framing === 'deterioration_observed' ? 'critical' : 'secondary'}>
                            {toWords(m.outcome_framing)}
                          </Badge>
                        </div>
                        <div className="mt-1.5 flex flex-wrap gap-3 text-muted-foreground">
                          <span className="inline-flex items-center gap-1"><CheckCircle2 className="h-3 w-3 text-success" /> {m.improvement_observed_count} improved</span>
                          <span className="inline-flex items-center gap-1"><XCircle className="h-3 w-3 text-critical" /> {m.deterioration_observed_count} deteriorated</span>
                          <span className="inline-flex items-center gap-1"><HelpCircle className="h-3 w-3" /> {m.inconclusive_count} inconclusive</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </Section>
              )}

              {/* Degraded warning */}
              {intel.degraded && (
                <div className="rounded-lg border border-warning/30 bg-warning/5 px-3 py-2.5 text-xs">
                  <div className="flex items-start gap-2">
                    <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-warning" />
                    <div>
                      <p className="font-medium text-foreground">Intelligence limited by available evidence</p>
                      <p className="mt-0.5 text-muted-foreground">Treat as investigative guidance, not causal proof.</p>
                    </div>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* Replay timeline section */}
      <div>
        {!replayOpen ? (
          <button
            type="button"
            onClick={handleReplayOpen}
            className="inline-flex items-center gap-2 rounded-xl border border-border/60 bg-muted/20 px-4 py-2.5 text-sm font-medium text-muted-foreground hover:bg-muted/40 hover:text-foreground transition-colors"
          >
            <GitBranch className="h-4 w-4" />
            Load incident replay / timeline
          </button>
        ) : replayLoading ? (
          <div className="rounded-xl border border-border/60 bg-muted/10 p-4 text-sm text-muted-foreground">
            Assembling replay…
          </div>
        ) : replayError ? (
          <AlertCard variant="warning" title="Replay unavailable" description={replayError} />
        ) : replay ? (
          <ReplayTimeline
            segments={replaySegs}
            truthNote={replay.truth_note}
            generatedAt={replay.generated_at}
            replayMeta={replay.replay_meta}
          />
        ) : null}
      </div>

      {/* Recommendation outcomes (from replay) */}
      {replay && (replay.recommendation_outcomes?.length ?? 0) > 0 && !outcomesInTimeline && (
        <div className="rounded-xl border border-border/60 bg-muted/10 p-4 space-y-3">
          <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
            <CheckCircle2 className="h-3.5 w-3.5" />
            Recommendation outcomes recorded
          </div>
          <div className="space-y-2">
            {replay.recommendation_outcomes!.map((o) => (
              <div key={o.id} className="flex flex-wrap items-center gap-2 rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs">
                <code className="font-mono text-muted-foreground">{o.recommendation_id.slice(0, 16)}</code>
                <Badge variant={outcomeVariant(o.outcome)}>{toWords(o.outcome)}</Badge>
                {o.actor_id && <span className="text-muted-foreground">by {o.actor_id}</span>}
                {o.note && <span className="text-muted-foreground italic">{o.note}</span>}
                <span className="ml-auto text-muted-foreground/60">{formatRelativeTime(o.created_at)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
