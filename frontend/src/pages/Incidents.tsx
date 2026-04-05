import { useIncidents } from '@/hooks/useIncidents'
import { useOperatorContext } from '@/hooks/useOperatorContext'
import { useVersionInfo } from '@/hooks/useVersionInfo'
import {
  incidentTopologyFocusNodeNum,
  openIncidentShiftWhyLine,
  sortOpenIncidentsForShiftStart,
  type IncidentWorkQueueWhyContext,
} from '@/utils/operatorWorkflow'
import { partitionOpenIncidentsForWorkbench } from '@/utils/incidentWorkbench'
import {
  incidentMemoryDecisionCue,
  operatorCanReadLinkedControlRows,
  resolvedIncidentActionVisibility,
} from '@/utils/incidentOperatorTruth'
import { operatorExportReadinessFromVersion } from '@/utils/operatorExportReadiness'
import { countV2QueueRowsMissingLex } from '@/utils/incidentQueueSort'
import { PageHeader } from '@/components/ui/PageHeader'
import { OperatorTruthRibbon } from '@/components/ui/OperatorTruthRibbon'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { EmptyState } from '@/components/ui/EmptyState'
import { IncidentRationaleSummary } from '@/components/incidents/IncidentRationaleSummary'
import { formatTimestamp, formatRelativeTime, type Incident } from '@/types/api'
import {
  ClipboardCopy,
  Download,
  RefreshCw,
  AlertTriangle,
  Clock,
  User,
  ArrowRight,
  ChevronDown,
  ChevronUp,
  Eye,
  Shield,
  Zap,
  CheckCircle2,
  XCircle,
  HelpCircle,
  Activity,
  FileText,
  Link2,
  GitBranch,
  ListOrdered,
  Compass,
  ClipboardList,
} from 'lucide-react'
import { clsx } from 'clsx'
import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

function isOpenIncident(inc: Incident): boolean {
  const s = (inc.state || '').toLowerCase()
  return s !== 'resolved' && s !== 'closed'
}

function copyText(text: string) {
  void navigator.clipboard.writeText(text)
}

function toWords(value: string | undefined): string {
  return (value || '').replace(/_/g, ' ').trim()
}

function outcomeFramingLabel(value: string | undefined): string {
  switch (value) {
    case 'improvement_observed':
      return 'Improvement observed'
    case 'deterioration_observed':
      return 'Deterioration observed'
    case 'mixed_historical_evidence':
      return 'Mixed evidence'
    case 'insufficient_evidence':
      return 'Insufficient evidence'
    case 'no_clear_post_action_signal':
      return 'No clear signal'
    default:
      return toWords(value) || 'Unknown'
  }
}

function wirelessClassificationLabel(value: string | undefined): string {
  switch (value) {
    case 'lora_mesh_pressure':
      return 'LoRa / frequency pressure'
    case 'wifi_backhaul_instability':
      return 'Wi-Fi backhaul instability'
    case 'mixed_path_degradation':
      return 'Mixed-path degradation'
    case 'sparse_evidence_incident':
      return 'Sparse evidence'
    case 'unsupported_wireless_domain_observed':
      return 'Unsupported wireless domain'
    case 'recurring_unknown_pattern':
      return 'Recurring unknown pattern'
    default:
      return toWords(value) || 'Unclassified'
  }
}

function humanizeReasonCode(value: string | undefined): string {
  const text = toWords(value)
  if (!text) return 'No additional context'
  return text.charAt(0).toUpperCase() + text.slice(1)
}

function triageBadgeLabels(codes: string[] | undefined): string[] {
  if (!codes?.length) return []
  const skip = new Set(['open_routine'])
  const out: string[] = []
  for (const c of codes) {
    if (skip.has(c)) continue
    out.push(c.replace(/_/g, ' '))
    if (out.length >= 2) break
  }
  return out
}

function sparsityMarkerLabel(code: string): string {
  switch (code) {
    case 'limited_correlated_evidence':
      return 'Limited correlated evidence in the observation window'
    default:
      return humanizeReasonCode(code)
  }
}

function snapshotCompletenessTone(value: string | undefined): 'secondary' | 'warning' | 'outline' {
  if (value === 'partial') return 'warning'
  if (value === 'complete') return 'secondary'
  return 'outline'
}

function defaultProofpackFilename(incidentId: string): string {
  return `proofpack-${incidentId || 'incident'}.json`
}

function filenameFromDisposition(contentDisposition: string | null, fallback: string): string {
  if (!contentDisposition) return fallback
  const match = contentDisposition.match(/filename\*?=(?:UTF-8''|")?([^";]+)/i)
  if (!match || !match[1]) return fallback
  const value = match[1].replace(/"/g, '').trim()
  try {
    return decodeURIComponent(value)
  } catch {
    return value || fallback
  }
}

function evidenceStrengthVariant(strength: string | undefined): 'success' | 'warning' | 'secondary' {
  if (strength === 'strong') return 'success'
  if (strength === 'moderate') return 'warning'
  return 'secondary'
}

function replaySemanticBadgeVariant(semantic: Incident['replay_summary'] extends infer T ? T extends { semantic?: infer U } ? U : never : never): 'warning' | 'secondary' | 'outline' {
  switch (semantic) {
    case 'active_changing':
    case 'sparse':
    case 'partial':
    case 'unavailable':
      return 'warning'
    case 'cooling_off':
    case 'quiet_recently':
      return 'secondary'
    default:
      return 'outline'
  }
}

function replaySemanticLabel(semantic: string | undefined): string {
  switch (semantic) {
    case 'active_changing':
      return 'Replay active'
    case 'cooling_off':
      return 'Replay cooling'
    case 'quiet_recently':
      return 'Replay quiet'
    case 'sparse':
      return 'Replay sparse'
    case 'no_history':
      return 'Replay no history'
    case 'partial':
      return 'Replay partial'
    case 'unavailable':
      return 'Replay unavailable'
    default:
      return toWords(semantic) || 'Replay'
  }
}

function outcomeMemoryScanLine(mem: NonNullable<Incident['intelligence']>['action_outcome_memory']): string | null {
  if (!mem?.length) return null
  const top = mem[0]
  if (!top) return null
  const label = top.action_label || toWords(top.action_type) || 'action'
  return `${label}: ${outcomeFramingLabel(top.outcome_framing)} (n=${top.sample_size})`
}

export function Incidents() {
  const { data, loading, error, refresh } = useIncidents()
  const ctx = useOperatorContext()
  const versionInfo = useVersionInfo()
  const [searchParams, setSearchParams] = useSearchParams()
  const focusIncidentId = (searchParams.get('focus') || '').trim()

  useEffect(() => {
    if (!focusIncidentId) return
    const el = document.getElementById(`mel-workbench-row-${focusIncidentId}`)
    if (!el) return
    const reduceMotion = typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
    el.scrollIntoView({ block: 'nearest', behavior: reduceMotion ? 'auto' : 'smooth' })
  }, [focusIncidentId, loading, data])

  if (loading && !data) {
    return <Loading message="Loading incidents..." />
  }

  if (error && !data) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to load incidents"
          description={error}
          action={
            <button
              type="button"
              onClick={() => void refresh()}
              className="button-danger"
            >
              Retry
            </button>
          }
        />
      </div>
    )
  }

  const incidents = data || []
  const openIncidents = incidents.filter(isOpenIncident)
  const closedIncidents = incidents.filter((i) => !isOpenIncident(i))
  const canHandoff = ctx.trustUI?.incident_handoff_write === true
  const canMutate = ctx.trustUI?.incident_mutate === true

  const exportPosture = versionInfo.data?.platform_posture?.evidence_export_delete
  const canReadLinkedActions = operatorCanReadLinkedControlRows({
    loading: ctx.loading,
    error: ctx.error,
    trustUI: ctx.trustUI,
    capabilities: ctx.capabilities ?? [],
  })
  const workbenchWhyCtx: IncidentWorkQueueWhyContext = {
    exportEnabled: exportPosture?.export_enabled,
    exportPolicyUnknown: !versionInfo.loading && versionInfo.error != null && exportPosture == null,
    canReadLinkedActions,
  }
  const exportReadiness = operatorExportReadinessFromVersion(versionInfo.data, versionInfo.error ?? null)
  const proofpackExportBlocked = exportReadiness.semantic === 'policy_limited' || exportReadiness.semantic === 'unknown_partial'
  const proofpackExportBlockedReason = exportReadiness.summary

  const { needsAttention, backlog } = partitionOpenIncidentsForWorkbench(incidents, workbenchWhyCtx)
  const openSortedFull = sortOpenIncidentsForShiftStart(openIncidents, workbenchWhyCtx)
  const v2QueueRowsMissingLex = countV2QueueRowsMissingLex(openIncidents)

  const clearFocusParam = () => {
    setSearchParams(
      (prev) => {
        const n = new URLSearchParams(prev)
        n.delete('focus')
        return n
      },
      { replace: true },
    )
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <PageHeader
          title="Incidents"
          subtitle="Mesh operations cockpit"
          description="Open-work queue: same priority signals as Command surface — review state, control gates, evidence posture, recurrence — then handoff and export paths."
        />
        <button
          type="button"
          onClick={() => {
            void refresh()
            void ctx.refresh()
          }}
          className="button-secondary"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      <OperatorTruthRibbon summary="Priorities and evidence strength come from stored incident intelligence and ingest — not from guessed mesh health. Sparse or degraded markers mean explicit gaps, not hidden certainty." />

      {ctx.error && (
        <AlertCard variant="warning" title="Operator context unavailable" description={ctx.error} />
      )}

      {versionInfo.error && (
        <p className="text-xs text-warning border border-warning/25 rounded-sm px-3 py-2 bg-warning/5" role="status">
          Version / export policy not loaded ({versionInfo.error}). Queue “why” lines may omit export gates — confirm under Settings before
          choosing proofpack or escalation.
        </p>
      )}

      {!versionInfo.loading && (
        <div
          className={clsx(
            'rounded-md border px-3 py-2.5 text-xs',
            exportReadiness.semantic === 'available'
              ? 'border-success/25 bg-success/5 text-muted-foreground'
              : exportReadiness.semantic === 'policy_limited'
                ? 'border-critical/30 bg-critical/5 text-foreground'
                : exportReadiness.semantic === 'degraded'
                  ? 'border-warning/30 bg-warning/5 text-foreground'
                  : 'border-warning/25 bg-warning/5 text-foreground',
          )}
          role="status"
          aria-live="polite"
          data-testid="workbench-export-readiness"
        >
          <span className="font-semibold text-foreground">Export / bundle readiness: </span>
          {exportReadiness.summary}
          {exportReadiness.evidenceBasis.length > 0 && (
            <p className="mt-1 text-mel-xs font-mono text-muted-foreground/90 break-all">
              evidence_basis: {exportReadiness.evidenceBasis.slice(0, 4).join(' · ')}
            </p>
          )}
          {exportReadiness.blockers.length > 0 && (
            <ul className="mt-2 list-disc pl-4 text-mel-sm text-muted-foreground space-y-0.5">
              {exportReadiness.blockers.map((b) => (
                <li key={b.code}>
                  <span className="font-mono text-foreground/80">{b.code}</span>
                  {b.summary ? ` — ${b.summary}` : ''}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      {openIncidents.some((i) => i.triage_signals?.queue_ordering_contract) && (
        <p className="text-mel-sm text-muted-foreground border border-border/40 rounded-sm px-3 py-2" data-testid="workbench-queue-contract-note">
          Open rows use server <span className="font-mono">triage_signals.queue_sort_key_lex</span> when present (workbench v2); otherwise{' '}
          <span className="font-mono">queue_sort_primary</span> then recency — same contract as incident detail; presentation-only toggles stay local.
        </p>
      )}
      {v2QueueRowsMissingLex > 0 && (
        <p className="text-mel-sm text-warning border border-warning/30 bg-warning/5 rounded-sm px-3 py-2" role="status">
          Queue ordering degraded for {v2QueueRowsMissingLex} open incident{v2QueueRowsMissingLex > 1 ? 's' : ''}: server reports
          <span className="font-mono"> open_incident_workbench_v2 </span>
          without
          <span className="font-mono"> queue_sort_key_lex</span>. MEL keeps those rows behind fully keyed rows and labels this as partial truth.
        </p>
      )}

      {focusIncidentId && (
        <div className="flex flex-wrap items-center gap-2 rounded-sm border border-info/25 bg-info/5 px-3 py-2 text-xs text-muted-foreground">
          <span>
            Returned to workbench row <span className="font-mono text-foreground">{focusIncidentId.slice(0, 14)}…</span>
          </span>
          <button type="button" onClick={clearFocusParam} className="text-primary font-semibold hover:underline min-h-[44px] sm:min-h-0 px-1">
            Clear highlight
          </button>
        </div>
      )}

      {!canHandoff && !ctx.loading && (
        <div className="flex items-center gap-2 rounded-md border border-info/20 bg-info/5 px-4 py-2.5 text-xs text-muted-foreground">
          <Eye className="h-3.5 w-3.5 text-info" />
          Read-only view. Your credentials do not include incident_handoff_write.
        </div>
      )}

      {/* Summary stats */}
      <div className="flex flex-wrap gap-3">
        <div className={clsx(
          'flex items-center gap-2 rounded-full border px-3 py-1.5 text-mel-sm font-semibold uppercase tracking-[0.16em]',
          openIncidents.length > 0 ? 'border-warning/25 bg-warning/8 text-warning' : 'border-success/20 bg-success/8 text-success'
        )}>
          <span className={clsx('h-1.5 w-1.5 rounded-full', openIncidents.length > 0 ? 'bg-warning' : 'bg-success')} />
          {openIncidents.length} open
        </div>
        <div className="flex items-center gap-2 rounded-full border border-border/60 bg-muted/30 px-3 py-1.5 text-mel-sm font-semibold uppercase tracking-[0.16em] text-muted-foreground">
          {closedIncidents.length} resolved
        </div>
        <div className="flex items-center gap-2 rounded-full border border-border/60 bg-muted/30 px-3 py-1.5 text-mel-sm font-semibold uppercase tracking-[0.16em] text-muted-foreground">
          {incidents.length} total
        </div>
      </div>

      {openIncidents.length === 0 ? (
        <EmptyState
          type="no-data"
          title="No open incidents"
          description={
            incidents.length === 0
              ? 'No incidents in the recent list. When transport or system disruptions are detected, they appear here with intelligence and handoff context.'
              : 'All recent incidents are resolved or closed.'
          }
        />
      ) : (
        <div className="space-y-4">
          <div
            id="mel-incident-workbench"
            className="rounded-md border border-border/60 bg-muted/10 px-3 py-2.5 text-xs text-muted-foreground"
            data-testid="incident-workbench-strip"
          >
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
              <span className="inline-flex items-center gap-1.5 font-semibold text-foreground">
                <ListOrdered className="h-3.5 w-3.5 shrink-0" aria-hidden />
                Workbench order
              </span>
              <span className="text-mel-sm">
                Matches Command surface: follow-up / control gates → sparse or degraded intel → recurrence → rest by last update.
              </span>
            </div>
            <p className="mt-1.5 text-mel-sm leading-snug">
              <span className="text-foreground font-medium">{needsAttention.length}</span> row{needsAttention.length === 1 ? '' : 's'} in
              “needs attention” (explicit review, control gates, partial action visibility / degraded action trace, sparse or degraded
              intel, or history-without-linkage signals).{' '}
              <span className="text-foreground font-medium">{backlog.length}</span> in backlog — still open, lower immediate pressure by the
              same deterministic rules.
            </p>
            {!canReadLinkedActions && !ctx.loading && ctx.error == null && (
              <p className="mt-2 text-mel-sm text-warning border border-warning/20 rounded-sm px-2 py-1.5 bg-warning/5" role="status">
                Linked control rows are hidden without read_actions — ordering still uses refs on the record; open the control queue to see
                FK-linked work.
              </p>
            )}
          </div>

          {needsAttention.length > 0 && (
            <section className="space-y-3" aria-labelledby="mel-workbench-needs-heading">
              <h2
                id="mel-workbench-needs-heading"
                className="flex items-center gap-2 text-mel-sm font-semibold uppercase tracking-[0.2em] text-warning"
              >
                <AlertTriangle className="h-3.5 w-3.5" />
                Needs attention first
              </h2>
              <div className="space-y-4">
                {needsAttention.map((inc) => (
                  <IncidentCard
                    key={inc.id}
                    incident={inc}
                    canMutate={canMutate}
                    canReadLinkedActions={canReadLinkedActions}
                    workbenchWhyContext={workbenchWhyCtx}
                    proofpackExportBlocked={proofpackExportBlocked}
                    proofpackExportBlockedReason={proofpackExportBlockedReason}
                    workbenchSection="needs"
                    workbenchHighlight={focusIncidentId === inc.id}
                  />
                ))}
              </div>
            </section>
          )}

          {backlog.length > 0 && (
            <section className="space-y-3" aria-labelledby="mel-workbench-backlog-heading">
              <h2
                id="mel-workbench-backlog-heading"
                className="flex items-center gap-2 text-mel-sm font-semibold uppercase tracking-[0.2em] text-muted-foreground"
              >
                <Clock className="h-3.5 w-3.5" />
                Open backlog
              </h2>
              <div className="space-y-4">
                {backlog.map((inc) => (
                  <IncidentCard
                    key={inc.id}
                    incident={inc}
                    canMutate={canMutate}
                    canReadLinkedActions={canReadLinkedActions}
                    workbenchWhyContext={workbenchWhyCtx}
                    proofpackExportBlocked={proofpackExportBlocked}
                    proofpackExportBlockedReason={proofpackExportBlockedReason}
                    workbenchSection="backlog"
                    workbenchHighlight={focusIncidentId === inc.id}
                  />
                ))}
              </div>
            </section>
          )}

          {needsAttention.length === 0 && backlog.length === 0 && openSortedFull.length > 0 && (
            <div className="space-y-4">
              {openSortedFull.map((inc) => (
                <IncidentCard
                  key={inc.id}
                  incident={inc}
                  canMutate={canMutate}
                  canReadLinkedActions={canReadLinkedActions}
                  workbenchWhyContext={workbenchWhyCtx}
                  proofpackExportBlocked={proofpackExportBlocked}
                  proofpackExportBlockedReason={proofpackExportBlockedReason}
                  workbenchSection="open"
                  workbenchHighlight={focusIncidentId === inc.id}
                />
              ))}
            </div>
          )}
        </div>
      )}

      {closedIncidents.length > 0 && (
        <section className="space-y-3 pt-2">
          <h2 className="flex items-center gap-2 text-mel-sm font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            <CheckCircle2 className="h-3.5 w-3.5" />
            Resolved incidents
          </h2>
          <div className="space-y-3">
            {closedIncidents.map((inc) => (
              <IncidentCard key={inc.id} incident={inc} muted />
            ))}
          </div>
        </section>
      )}
    </div>
  )
}

function ProofpackDownloadButton({
  incidentId,
  exportBlocked,
  exportBlockedReason,
}: {
  incidentId: string
  exportBlocked?: boolean
  exportBlockedReason?: string
}) {
  const [state, setState] = useState<'idle' | 'loading' | 'error'>('idle')
  const [errorMsg, setErrorMsg] = useState('')

  async function download() {
    if (exportBlocked) return
    setState('loading')
    setErrorMsg('')
    try {
      const resp = await fetch(`/api/v1/incidents/${encodeURIComponent(incidentId)}/proofpack?download=true`)
      if (!resp.ok) {
        const body = await resp.text().catch(() => '')
        if (resp.status === 401 || resp.status === 403) {
          setErrorMsg('Insufficient permissions for proofpack export.')
        } else if (resp.status === 404) {
          setErrorMsg('Incident not found.')
        } else {
          setErrorMsg(body || `HTTP ${resp.status}`)
        }
        setState('error')
        return
      }
      const blob = await resp.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filenameFromDisposition(
        resp.headers.get('content-disposition'),
        defaultProofpackFilename(incidentId),
      )
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
      setState('idle')
    } catch {
      setErrorMsg('Network error — MEL backend unreachable.')
      setState('error')
    }
  }

  return (
    <div className="flex flex-col gap-1.5">
      {exportBlocked && exportBlockedReason && (
        <p className="text-xs text-warning border border-warning/25 rounded-sm px-2 py-1.5 bg-warning/5" role="status">
          {exportBlockedReason}
        </p>
      )}
      <div className="flex flex-wrap items-center gap-2">
      <button
        type="button"
        onClick={() => void download()}
        disabled={state === 'loading' || exportBlocked}
        className="button-secondary text-xs min-h-[44px] sm:min-h-0 touch-manipulation"
        title={
          exportBlocked
            ? exportBlockedReason
            : 'Download incident evidence proofpack (JSON)'
        }
      >
        <Download className="h-3.5 w-3.5" />
        {state === 'loading' ? 'Assembling...' : 'Export proofpack'}
      </button>
      <span className="text-mel-xs text-muted-foreground/60">
        Snapshot at request-time. Review evidence_gaps. For continuity without proof, use handoff on the incident page.
      </span>
      {state === 'error' && errorMsg && (
        <span className="text-xs text-critical">{errorMsg}</span>
      )}
      </div>
    </div>
  )
}

function IncidentCard({
  incident: inc,
  muted = false,
  canMutate = false,
  canReadLinkedActions = true,
  workbenchWhyContext,
  proofpackExportBlocked,
  proofpackExportBlockedReason,
  workbenchSection,
  workbenchHighlight,
}: {
  incident: Incident
  muted?: boolean
  canMutate?: boolean
  canReadLinkedActions?: boolean
  workbenchWhyContext?: IncidentWorkQueueWhyContext
  proofpackExportBlocked?: boolean
  proofpackExportBlockedReason?: string
  workbenchSection?: 'needs' | 'backlog' | 'open'
  workbenchHighlight?: boolean
}) {
  void canMutate // reserved for future mutation controls
  const [expanded, setExpanded] = useState(!muted)
  const pending = inc.pending_actions?.filter(Boolean) ?? []
  const hasHandoffText = !!(inc.handoff_summary && inc.handoff_summary.trim())
  const owner = inc.owner_actor_id?.trim()
  const intel = inc.intelligence
  const hasIntel = !!intel
  const seenBefore = (intel?.signature_match_count ?? 0) > 1
  const hasSimilar = (intel?.similar_incidents?.length ?? 0) > 0
  const actionVis = resolvedIncidentActionVisibility(inc, { canReadLinkedActions })
  const memoryLine = intel?.action_outcome_memory ? outcomeMemoryScanLine(intel.action_outcome_memory) : null
  const memoryDecisionCue = !muted ? incidentMemoryDecisionCue(inc) : null
  const topoNum = !muted ? incidentTopologyFocusNodeNum(inc) : null
  const priorityWhy = !muted ? openIncidentShiftWhyLine(inc, workbenchWhyContext) : ''
  const replay = inc.replay_summary
  const replayTitle = replay?.summary || replaySemanticLabel(replay?.semantic)

  const severityVariant = inc.severity === 'critical' ? 'critical' : inc.severity === 'high' ? 'warning' : 'secondary'
  const stateVariant = inc.state === 'resolved' || inc.state === 'closed' ? 'success' : 'outline'

  const workbenchReturn =
    !muted && workbenchSection
      ? `/incidents?focus=${encodeURIComponent(inc.id)}&section=${encodeURIComponent(workbenchSection)}`
      : '/incidents'
  const incidentDetailReturnPath = `/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(workbenchReturn)}`

  return (
    <Card
      id={!muted ? `mel-workbench-row-${inc.id}` : undefined}
      className={clsx(
        muted && 'opacity-75',
        'transition-shadow hover:shadow-[0_20px_48px_-28px_hsl(var(--shell-shadow)/0.5)]',
        workbenchHighlight && 'ring-2 ring-info/40 border-info/30',
      )}
    >
      {/* Header stripe */}
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <AlertTriangle className={clsx('h-4 w-4 shrink-0', inc.severity === 'critical' ? 'text-critical' : inc.severity === 'high' ? 'text-warning' : 'text-muted-foreground')} />
              <CardTitle className="text-base">{inc.title || inc.id}</CardTitle>
            </div>
            <div className="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
              <span className="inline-flex items-center gap-1 font-mono">
                <Link2 className="h-3 w-3" />
                <Link to={`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(workbenchReturn)}`} className="hover:underline">
                  {inc.id.slice(0, 12)}
                </Link>
              </span>
              {inc.occurred_at && (
                <span className="inline-flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  {formatRelativeTime(inc.occurred_at)}
                </span>
              )}
              {owner && (
                <span className="inline-flex items-center gap-1">
                  <User className="h-3 w-3" />
                  {owner}
                </span>
              )}
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-1.5">
            {inc.state && <Badge variant={stateVariant as 'success' | 'outline'}>{inc.state}</Badge>}
            {inc.severity && <Badge variant={severityVariant as 'critical' | 'warning' | 'secondary'}>{inc.severity}</Badge>}
            {hasIntel && (
              <Badge variant={evidenceStrengthVariant(intel.evidence_strength)}>
                {intel.evidence_strength} evidence
              </Badge>
            )}
            {seenBefore && (
              <Badge variant="warning">
                seen {intel!.signature_match_count}x
              </Badge>
            )}
            {actionVis.kind === 'visibility_limited' && (
              <span title={actionVis.explanation}>
                <Badge variant="warning">{actionVis.shortLabel}</Badge>
              </span>
            )}
            {actionVis.kind === 'action_context_degraded' && (
              <span title={actionVis.explanation}>
                <Badge variant="warning">{actionVis.shortLabel}</Badge>
              </span>
            )}
            {actionVis.kind === 'no_linked_historical_signals' && (
              <span title={actionVis.explanation}>
                <Badge variant="secondary">{actionVis.shortLabel}</Badge>
              </span>
            )}
            {actionVis.kind === 'linked_observed' && actionVis.awaitingApproval > 0 && (
              <span title={actionVis.explanation}>
                <Badge variant="warning">{actionVis.awaitingApproval} approval wait</Badge>
              </span>
            )}
            {actionVis.kind === 'linked_observed' &&
              actionVis.linkedCount > 0 &&
              actionVis.awaitingApproval === 0 && (
                <span title={actionVis.explanation}>
                  <Badge variant="secondary">{actionVis.linkedCount} linked control</Badge>
                </span>
              )}
            {actionVis.kind === 'references_only' && (
              <span title={actionVis.explanation}>
                <Badge variant="outline">
                  {actionVis.pendingRefCount + actionVis.recentActionIdCount} action ref
                  {actionVis.pendingRefCount + actionVis.recentActionIdCount > 1 ? 's' : ''}
                </Badge>
              </span>
            )}
            {!muted &&
              triageBadgeLabels(inc.triage_signals?.codes).map((label) => (
                <span key={label} title="From incident.triage_signals — deterministic API, not hidden scoring">
                  <Badge variant="outline" className="text-mel-xs normal-case">
                    {label}
                  </Badge>
                </span>
              ))}
            {!muted && replay?.semantic && (
              <span title={replayTitle}>
                <Badge variant={replaySemanticBadgeVariant(replay.semantic)}>
                  {replaySemanticLabel(replay.semantic)}
                </Badge>
              </span>
            )}
            {pending.length > 0 && actionVis.kind !== 'references_only' && (
              <span title="Action IDs referenced on the incident record (verify against queue)">
                <Badge variant="outline">
                  {pending.length} ref ID{pending.length > 1 ? 's' : ''}
                </Badge>
              </span>
            )}
            <Link
              to={`/incidents/${inc.id}?return=${encodeURIComponent(workbenchReturn)}`}
              className="ml-1 inline-flex items-center gap-1 text-mel-sm font-semibold text-primary hover:underline min-h-[44px] sm:min-h-0 px-1 touch-manipulation"
              title="Open incident detail page"
            >
              <ArrowRight className="h-3 w-3" />
              Detail
            </Link>
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-4 pt-0">
        {!muted && (
          <IncidentRationaleSummary
            incident={inc}
            fallbackWhy={priorityWhy}
            className="border-border/40 bg-background/40"
          />
        )}

        {!muted && actionVis.explanation && actionVis.kind !== 'linked_observed' && (
          <div
            className="rounded-sm border border-border/40 bg-background/40 px-3 py-2 text-mel-sm text-muted-foreground"
            data-testid="incident-workbench-action-visibility"
            role="status"
          >
            <span className="font-semibold text-foreground">Control / action context: </span>
            {actionVis.explanation}
            {actionVis.suggestControlQueue && (
              <>
                {' '}
                <Link
                  to={`/control-actions?incident=${encodeURIComponent(inc.id)}`}
                  className="font-semibold text-primary hover:underline whitespace-nowrap"
                >
                  Open control queue →
                </Link>
              </>
            )}
          </div>
        )}

        {inc.summary && (
          <p className="text-sm leading-relaxed text-muted-foreground">{inc.summary}</p>
        )}

        {!muted && (
          <div className="flex flex-wrap gap-2" role="navigation" aria-label="Incident shortcuts">
            <Link
              to={`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(workbenchReturn)}#mel-investigation-path`}
              className="inline-flex items-center gap-1 rounded-sm border border-border/60 bg-card/50 px-2.5 py-2 text-mel-sm font-semibold text-primary hover:bg-muted/40 min-h-[44px] sm:min-h-0 touch-manipulation"
            >
              <Compass className="h-3 w-3 shrink-0" aria-hidden />
              Path
            </Link>
            <Link
              to={`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(workbenchReturn)}&replay=1`}
              className="inline-flex items-center gap-1 rounded-sm border border-border/60 bg-card/50 px-2.5 py-2 text-mel-sm font-semibold text-primary hover:bg-muted/40 min-h-[44px] sm:min-h-0 touch-manipulation"
            >
              <Activity className="h-3 w-3 shrink-0" aria-hidden />
              Replay
            </Link>
            <Link
              to={`/topology?incident=${encodeURIComponent(inc.id)}&filter=incident_focus${topoNum != null ? `&select=${topoNum}` : ''}&return=${encodeURIComponent(incidentDetailReturnPath)}`}
              className="inline-flex items-center gap-1 rounded-sm border border-border/60 bg-card/50 px-2.5 py-2 text-mel-sm font-semibold text-primary hover:bg-muted/40 min-h-[44px] sm:min-h-0 touch-manipulation"
            >
              <GitBranch className="h-3 w-3 shrink-0" aria-hidden />
              Topology{topoNum != null ? ` ${topoNum}` : ''}
            </Link>
            <Link
              to={`/planning?incident=${encodeURIComponent(inc.id)}&return=${encodeURIComponent(incidentDetailReturnPath)}`}
              className="inline-flex items-center gap-1 rounded-sm border border-border/60 bg-card/50 px-2.5 py-2 text-mel-sm font-semibold text-primary hover:bg-muted/40 min-h-[44px] sm:min-h-0 touch-manipulation"
            >
              <ClipboardList className="h-3 w-3 shrink-0" aria-hidden />
              Planning
            </Link>
            <Link
              to={`/control-actions?incident=${encodeURIComponent(inc.id)}&return=${encodeURIComponent(incidentDetailReturnPath)}`}
              className="inline-flex items-center gap-1 rounded-sm border border-border/60 bg-card/50 px-2.5 py-2 text-mel-sm font-semibold text-primary hover:bg-muted/40 min-h-[44px] sm:min-h-0 touch-manipulation"
            >
              <Zap className="h-3 w-3 shrink-0" aria-hidden />
              Controls
            </Link>
            <Link
              to={`${incidentDetailReturnPath}#shift-continuity-handoff`}
              className="inline-flex items-center gap-1 rounded-sm border border-border/60 bg-card/50 px-2.5 py-2 text-mel-sm font-semibold text-primary hover:bg-muted/40 min-h-[44px] sm:min-h-0 touch-manipulation"
            >
              <Download className="h-3 w-3 shrink-0" aria-hidden />
              Handoff / export
            </Link>
          </div>
        )}

        {memoryLine && !muted && (
          <p className="text-mel-sm text-muted-foreground border-l-2 border-border/60 pl-2.5" data-testid="incident-workbench-memory-scan">
            <span className="font-semibold text-foreground">Outcome memory (scan): </span>
            {memoryLine}
            <span className="text-muted-foreground/70"> — association only; open detail for caveats.</span>
          </p>
        )}

        {memoryDecisionCue && (
          <p
            className="text-mel-sm text-foreground border-l-2 border-warning/40 pl-2.5"
            data-testid="incident-workbench-memory-decision"
          >
            <span className="font-semibold">What history changes next: </span>
            {memoryDecisionCue}
          </p>
        )}

        {inc.reopened_from_incident_id && (
          <div className="rounded-sm border border-warning/25 bg-warning/5 px-3 py-2 text-xs text-muted-foreground">
            <span className="font-semibold text-foreground">Reopened</span>
            {inc.reopened_at && <span> · {formatRelativeTime(inc.reopened_at)}</span>}
            {' — '}
            <Link to={`/incidents/${encodeURIComponent(inc.reopened_from_incident_id)}`} className="font-mono text-primary hover:underline">
              prior {inc.reopened_from_incident_id.slice(0, 12)}…
            </Link>
          </div>
        )}

        {/* Quick intelligence snapshot — always visible */}
        {hasIntel && (
          <div className="space-y-2">
            <div className="flex flex-wrap gap-2">
              {intel.signature_label && (
                <Badge variant="outline">
                  <Activity className="h-3 w-3" />
                  {intel.signature_label}
                </Badge>
              )}
              {intel.wireless_context && (
                <Badge variant="outline">
                  {wirelessClassificationLabel(intel.wireless_context.classification)}
                </Badge>
              )}
              {hasSimilar && (
                <Badge variant="secondary">
                  {intel.similar_incidents!.length} similar prior
                </Badge>
              )}
            </div>
            {(intel.sparsity_markers?.length ?? 0) > 0 && (
              <div className="rounded-sm border border-border/50 bg-muted/20 px-3 py-2 text-xs">
                <p className="font-medium text-foreground">Intelligence limited by available evidence</p>
                <p className="mt-0.5 text-muted-foreground">
                  Sparse signals — treat similarity and outcome memory as weakly supported until more evidence arrives.
                </p>
                <ul className="mt-1.5 list-disc space-y-0.5 pl-4 text-muted-foreground">
                  {intel.sparsity_markers!.map((m) => (
                    <li key={m}>{sparsityMarkerLabel(m)}</li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        )}

        {/* Expand/collapse toggle */}
        <button
          type="button"
          onClick={() => setExpanded(!expanded)}
          className="flex items-center gap-1 text-xs font-semibold text-muted-foreground transition-colors hover:text-foreground"
        >
          {expanded ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
          {expanded ? 'Collapse detail' : 'Expand detail'}
        </button>

        {expanded && (
          <div className="space-y-4 animate-fade-in">
            {/* Handoff summary */}
            <DetailSection title="Handoff summary" icon={<FileText className="h-3.5 w-3.5" />}>
              <div className={clsx(
                'rounded-sm border px-3 py-2 text-sm',
                hasHandoffText ? 'border-border/60 bg-card/50 text-foreground' : 'border-dashed border-border/50 bg-muted/20 text-muted-foreground'
              )}>
                {hasHandoffText ? inc.handoff_summary : 'No handoff summary recorded.'}
              </div>
            </DetailSection>

            {/* Proofpack export */}
            <DetailSection title="Evidence proofpack" icon={<Download className="h-3.5 w-3.5" />}>
              <ProofpackDownloadButton
                incidentId={inc.id}
                exportBlocked={proofpackExportBlocked}
                exportBlockedReason={proofpackExportBlockedReason}
              />
              <a href={`/incidents/${encodeURIComponent(inc.id)}`} className="mt-2 inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline">
                Open full incident review <ArrowRight className="h-3 w-3" />
              </a>
            </DetailSection>

            {/* Referenced actions */}
            {pending.length > 0 && (
              <DetailSection title="Referenced action IDs" icon={<Zap className="h-3.5 w-3.5" />}>
                <div className="flex flex-wrap gap-2">
                  {pending.map((id) => (
                    <div
                      key={id}
                      className="flex items-center gap-2 rounded-sm border border-border/60 bg-card/50 px-2.5 py-1.5"
                    >
                      <code className="text-xs">{id.slice(0, 16)}...</code>
                      <button
                        type="button"
                        onClick={() => copyText(id)}
                        className="rounded p-0.5 text-muted-foreground hover:text-foreground"
                        title="Copy action ID"
                      >
                        <ClipboardCopy className="h-3 w-3" />
                      </button>
                    </div>
                  ))}
                </div>
                <Link to="/control-actions" className="mt-2 inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline">
                  View in control actions <ArrowRight className="h-3 w-3" />
                </Link>
              </DetailSection>
            )}

            {/* Intelligence deep dive */}
            {hasIntel && (
              <div className="space-y-3 rounded-md border border-border/60 bg-muted/10 p-4">
                <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                  <Eye className="h-3.5 w-3.5" />
                  Incident intelligence
                </div>

                {/* Similar incidents */}
                {hasSimilar && (
                  <DetailSection title="Similar prior incidents" icon={<Link2 className="h-3.5 w-3.5" />}>
                    <div className="space-y-1.5">
                      {intel.similar_incidents!.map((s) => (
                        <div key={s.incident_id} className="flex items-center gap-3 rounded-sm border border-border/50 bg-card/40 px-3 py-2 text-xs">
                          <span className="font-mono text-muted-foreground">{s.incident_id.slice(0, 12)}</span>
                          {s.title && <span className="flex-1 truncate text-foreground">{s.title}</span>}
                          {s.state && <Badge variant={s.state === 'resolved' ? 'success' : 'secondary'}>{s.state}</Badge>}
                          {s.occurred_at && <span className="text-muted-foreground/60">{formatRelativeTime(s.occurred_at)}</span>}
                        </div>
                      ))}
                    </div>
                  </DetailSection>
                )}

                {/* Wireless context */}
                {intel.wireless_context && (
                  <DetailSection title="Wireless context" icon={<Activity className="h-3.5 w-3.5" />}>
                    <div className="space-y-2 text-xs">
                      <div className="flex flex-wrap gap-1.5">
                        <Badge variant="outline">{wirelessClassificationLabel(intel.wireless_context.classification)}</Badge>
                        <Badge variant="secondary">confidence: {toWords(intel.wireless_context.confidence_posture)}</Badge>
                        <Badge variant="outline">evidence: {toWords(intel.wireless_context.evidence_posture)}</Badge>
                      </div>
                      {intel.wireless_context.summary && (
                        <p className="text-muted-foreground">{intel.wireless_context.summary}</p>
                      )}
                      {(intel.wireless_context.observed_domains?.length ?? 0) > 0 && (
                        <p className="text-muted-foreground">
                          Observed domains: {intel.wireless_context.observed_domains!.join(', ')}
                        </p>
                      )}
                      {(intel.wireless_context.reasons?.length ?? 0) > 0 && (
                        <ul className="space-y-1 pl-4">
                          {intel.wireless_context.reasons!.slice(0, 3).map((r) => (
                            <li key={r.code} className="list-disc text-muted-foreground">{r.statement}</li>
                          ))}
                        </ul>
                      )}
                      {(intel.wireless_context.evidence_gaps?.length ?? 0) > 0 && (
                        <EvidenceGapBanner gaps={intel.wireless_context.evidence_gaps!.slice(0, 3)} />
                      )}
                      {(intel.wireless_context.unsupported?.length ?? 0) > 0 && (
                        <div className="rounded-sm border border-border/60 bg-muted/20 px-3 py-2 text-muted-foreground">
                          <span className="font-medium text-foreground">Unsupported scope:</span>{' '}
                          {intel.wireless_context.unsupported!.map((u) => `${u.domain} ${u.scope}`).join(', ')}
                        </div>
                      )}
                    </div>
                  </DetailSection>
                )}

                {/* Investigate next */}
                {(intel.investigate_next?.length ?? 0) > 0 && (
                  <DetailSection title="Investigate next" icon={<HelpCircle className="h-3.5 w-3.5" />}>
                    <div className="space-y-1.5">
                      {intel.investigate_next!.slice(0, 3).map((g) => (
                        <div key={g.id} className="rounded-sm border border-border/50 bg-card/40 px-3 py-2 text-xs">
                          <p className="font-medium text-foreground">{g.title}</p>
                          <p className="mt-0.5 text-muted-foreground">{g.rationale}</p>
                        </div>
                      ))}
                    </div>
                  </DetailSection>
                )}

                {/* Action outcome memory */}
                {(intel.action_outcome_memory?.length ?? 0) > 0 && (
                  <DetailSection title="Historical action outcomes" icon={<Zap className="h-3.5 w-3.5" />}>
                    <p className="mb-2 text-mel-sm text-muted-foreground">
                      Historical observations from similar incidents. Association only — does not establish causality.
                    </p>
                    <div className="space-y-2">
                      {intel.action_outcome_memory!.map((m) => (
                        <div key={m.action_type} className="rounded-sm border border-border/50 bg-card/40 p-3 text-xs">
                          <div className="flex flex-wrap items-center gap-1.5">
                            <span className="font-medium text-foreground">{m.action_label || m.action_type}</span>
                            <Badge variant="outline">n={m.sample_size}</Badge>
                            <Badge variant={m.outcome_framing === 'improvement_observed' ? 'success' : m.outcome_framing === 'deterioration_observed' ? 'critical' : 'secondary'}>
                              {outcomeFramingLabel(m.outcome_framing)}
                            </Badge>
                            {m.sample_size < 3 && <Badge variant="warning">sparse</Badge>}
                          </div>
                          <div className="mt-1.5 flex flex-wrap gap-3 text-muted-foreground">
                            <span className="inline-flex items-center gap-1">
                              <CheckCircle2 className="h-3 w-3 text-success" /> {m.improvement_observed_count} improved
                            </span>
                            <span className="inline-flex items-center gap-1">
                              <XCircle className="h-3 w-3 text-critical" /> {m.deterioration_observed_count} deteriorated
                            </span>
                            <span className="inline-flex items-center gap-1">
                              <HelpCircle className="h-3 w-3" /> {m.inconclusive_count} inconclusive
                            </span>
                          </div>
                          {(m.caveats?.length ?? 0) > 0 && (
                            <EvidenceGapBanner gaps={m.caveats!} label="Caveat" />
                          )}
                        </div>
                      ))}
                    </div>
                  </DetailSection>
                )}

                {/* Action outcome trace */}
                {intel.action_outcome_trace && (
                  <DetailSection title="Snapshot traceability" icon={<Shield className="h-3.5 w-3.5" />}>
                    <div className="flex flex-wrap items-center gap-1.5 text-xs">
                      <Badge variant={snapshotCompletenessTone(intel.action_outcome_trace.completeness)}>
                        {toWords(intel.action_outcome_trace.completeness)}
                      </Badge>
                      <Badge variant="outline">persisted: {intel.action_outcome_trace.persisted_snapshot_count}</Badge>
                      {intel.action_outcome_trace.snapshot_write_failures > 0 && (
                        <Badge variant="warning">write failures: {intel.action_outcome_trace.snapshot_write_failures}</Badge>
                      )}
                    </div>
                    {intel.action_outcome_trace.snapshot_retrieval_error && (
                      <p className="mt-1.5 text-xs text-warning">
                        Retrieval error: {intel.action_outcome_trace.snapshot_retrieval_error}
                      </p>
                    )}
                  </DetailSection>
                )}

                {/* Degraded intelligence warning */}
                {intel.degraded && (
                  <div className="rounded-sm border border-warning/30 bg-warning/5 px-3 py-2.5 text-xs">
                    <div className="flex items-start gap-2">
                      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-warning" />
                      <div>
                        <p className="font-medium text-foreground">
                          Intelligence limited by available evidence
                        </p>
                        <p className="mt-0.5 text-muted-foreground">
                          Treat as investigative guidance, not causal proof.
                        </p>
                        {(intel.degraded_reasons?.length ?? 0) > 0 && (
                          <ul className="mt-1.5 space-y-0.5">
                            {intel.degraded_reasons!.map((reason) => (
                              <li key={reason} className="text-muted-foreground">
                                <code className="rounded bg-muted/60 px-1 py-0.5 text-mel-xs">{reason}</code>{' '}
                                {humanizeReasonCode(reason)}
                              </li>
                            ))}
                          </ul>
                        )}
                      </div>
                    </div>
                  </div>
                )}

                {intel.generated_at && (
                  <p className="text-mel-xs text-muted-foreground/50">
                    Intelligence generated {formatTimestamp(intel.generated_at)}
                  </p>
                )}
              </div>
            )}

            {/* Metadata row */}
            <div className="flex flex-wrap gap-x-6 gap-y-2 border-t border-border/40 pt-3 text-xs text-muted-foreground">
              <span>Created: {formatTimestamp(inc.occurred_at)}</span>
              <span>Updated: {formatTimestamp(inc.updated_at)}</span>
              {inc.resolved_at && <span>Resolved: {formatTimestamp(inc.resolved_at)}</span>}
              {inc.category && <span>Category: {inc.category}</span>}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function DetailSection({
  title,
  icon,
  children,
}: {
  title: string
  icon: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5 text-mel-sm font-semibold uppercase tracking-[0.16em] text-muted-foreground">
        {icon}
        {title}
      </div>
      {children}
    </div>
  )
}

function EvidenceGapBanner({ gaps, label = 'Evidence gap' }: { gaps: string[]; label?: string }) {
  return (
    <div className="rounded-sm border border-warning/25 bg-warning/5 px-3 py-2 text-xs text-muted-foreground">
      <span className="font-medium text-foreground">{label}:</span>{' '}
      {gaps.join(', ')}
    </div>
  )
}
