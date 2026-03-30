import { useCallback, useEffect, useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
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
import { formatTimestamp, formatRelativeTime, type Incident } from '@/types/api'

// ─── Types ────────────────────────────────────────────────────────────────────

interface ReplaySegment {
  event_time: string
  event_type: string
  event_id?: string
  summary: string
  knowledge_posture: string
  evidence_refs?: string[]
}

interface ReplayView {
  kind: string
  incident_id: string
  incident: Incident
  replay_segments?: ReplaySegment[]
  knowledge_timeline?: ReplaySegment[]
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

function severityVariant(s?: string): 'critical' | 'warning' | 'secondary' {
  if (s === 'critical') return 'critical'
  if (s === 'high') return 'warning'
  return 'secondary'
}

function stateVariant(s?: string): 'success' | 'outline' {
  if (s === 'resolved' || s === 'closed') return 'success'
  return 'outline'
}

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
    </div>
  )
}

// ─── Replay timeline ──────────────────────────────────────────────────────────

function ReplayTimeline({ segments, truthNote, generatedAt }: {
  segments: ReplaySegment[]
  truthNote?: string
  generatedAt?: string
}) {
  const [expanded, setExpanded] = useState<Set<number>>(new Set())

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
          No timeline events found for this incident window. Evidence may have been pruned or the incident is too recent.
        </p>
      </div>
    )
  }

  return (
    <div className="rounded-xl border border-border/60 bg-muted/10 p-4 space-y-3">
      <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
        <GitBranch className="h-3.5 w-3.5" />
        Incident replay / timeline
        <span className="ml-auto font-normal normal-case tracking-normal">{segments.length} events</span>
      </div>

      {truthNote && (
        <p className="text-[11px] text-muted-foreground/70 border-l-2 border-muted pl-2.5 leading-snug">
          {truthNote}
        </p>
      )}

      <div className="relative">
        {/* Vertical line */}
        <div className="absolute left-[6px] top-0 bottom-0 w-px bg-border/60" />

        <ol className="space-y-0">
          {segments.map((seg, i) => {
            const isExp = expanded.has(i)
            const hasRefs = (seg.evidence_refs?.length ?? 0) > 0
            return (
              <li key={seg.event_id ?? i} className="relative flex gap-3 pl-6 pb-4 last:pb-0">
                {/* Dot */}
                <Circle className={clsx('absolute left-0 h-3.5 w-3.5 shrink-0 fill-current top-0.5', postureColor(seg.knowledge_posture))} />

                <div className="flex-1 min-w-0">
                  <div className="flex flex-wrap items-start gap-x-3 gap-y-0.5">
                    <span className="text-xs font-medium text-foreground leading-snug">{seg.summary || toWords(seg.event_type)}</span>
                    <span className={clsx('text-[10px] font-semibold uppercase tracking-wide shrink-0', postureColor(seg.knowledge_posture))}>
                      {postureLabel(seg.knowledge_posture)}
                    </span>
                  </div>
                  <div className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-[11px] text-muted-foreground">
                    <span className="inline-flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      {seg.event_time ? formatRelativeTime(seg.event_time) : '—'}
                    </span>
                    <code className="text-muted-foreground/60">{toWords(seg.event_type)}</code>
                    {hasRefs && (
                      <button
                        type="button"
                        onClick={() => toggle(i)}
                        className="inline-flex items-center gap-0.5 text-[10px] text-muted-foreground hover:text-foreground"
                      >
                        {isExp ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                        refs
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

// ─── Main page ────────────────────────────────────────────────────────────────

export function IncidentDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [inc, setInc] = useState<Incident | null>(null)
  const [replay, setReplay] = useState<ReplayView | null>(null)
  const [loading, setLoading] = useState(true)
  const [replayLoading, setReplayLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [replayError, setReplayError] = useState<string | null>(null)
  const [replayOpen, setReplayOpen] = useState(false)

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

  function handleReplayOpen() {
    setReplayOpen(true)
    if (!replay && !replayLoading) void loadReplay()
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
              {hasIntel && <Badge variant={evidenceStrengthVariant(intel.evidence_strength)}>{intel.evidence_strength} evidence</Badge>}
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
                <Link to="/control-actions" className="inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline">
                  View in control actions →
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
                        className="flex items-center gap-3 rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs hover:border-border/80 hover:bg-card/70 transition-colors"
                      >
                        <span className="font-mono text-muted-foreground shrink-0">{s.incident_id.slice(0, 12)}</span>
                        {s.title && <span className="flex-1 truncate">{s.title}</span>}
                        {s.state && <Badge variant={s.state === 'resolved' ? 'success' : 'secondary'}>{s.state}</Badge>}
                        {s.weighted_score != null && (
                          <span className="text-muted-foreground/60 shrink-0">{(s.weighted_score * 100).toFixed(0)}%</span>
                        )}
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
                        <p className="font-medium text-foreground">{g.title}</p>
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
            segments={replay.replay_segments ?? replay.knowledge_timeline ?? []}
            truthNote={replay.truth_note}
            generatedAt={replay.generated_at}
          />
        ) : null}
      </div>

      {/* Recommendation outcomes (from replay) */}
      {replay && (replay.recommendation_outcomes?.length ?? 0) > 0 && (
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
