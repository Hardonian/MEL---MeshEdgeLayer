import { useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

function withReturnParam(targetPath: string, returnPath: string): string {
  if (!returnPath.startsWith('/')) return targetPath
  const joiner = targetPath.includes('?') ? '&' : '?'
  return `${targetPath}${joiner}return=${encodeURIComponent(returnPath)}`
}
import { useControlActions } from '@/hooks/useControlActions'
import { useOperatorContext } from '@/hooks/useOperatorContext'
import { PageHeader } from '@/components/ui/PageHeader'
import { OperatorTruthRibbon } from '@/components/ui/OperatorTruthRibbon'
import { MelPanelInset, MelSegment, MelSegmentItem } from '@/components/ui/operator'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatTimestamp, formatRelativeTime, type ControlActionRecord } from '@/types/api'
import {
  RefreshCw,
  Zap,
  Clock,
  User,
  Shield,
  AlertTriangle,
  ChevronDown,
  ChevronUp,
  ArrowRight,
  Eye,
  CheckCircle2,
  XCircle,
} from 'lucide-react'
import { clsx } from 'clsx'
import { controlActionExecPhase } from '@/utils/controlActionPhase'

const LIFECYCLE_FILTERS = [
  { value: '', label: 'All' },
  { value: 'pending_approval', label: 'Awaiting approval' },
  { value: 'pending', label: 'Approved / queued' },
  { value: 'running', label: 'Executing' },
  { value: 'completed', label: 'Completed' },
]

export function ControlActions() {
  const [searchParams] = useSearchParams()
  const incidentFromUrl = (searchParams.get('incident') || '').trim()
  const returnParam = (searchParams.get('return') || '').trim()
  const [filter, setFilter] = useState('')
  const { data, loading, error, refresh } = useControlActions(filter)
  const ctx = useOperatorContext()

  const rows = useMemo(() => {
    const list = data ?? []
    if (!incidentFromUrl) return list
    return list.filter((a) => {
      const id = (a.incident_id || '').trim()
      return id === incidentFromUrl || id.startsWith(incidentFromUrl)
    })
  }, [data, incidentFromUrl])
  const canRead = ctx.trustUI?.read_actions === true || ctx.capabilities?.includes('read_actions')

  if (loading && !data) {
    return <Loading message="Loading control actions..." />
  }

  if (error && !data) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to load actions"
          description={error}
          action={<button type="button" onClick={() => void refresh()} className="button-danger">Retry</button>}
        />
      </div>
    )
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <PageHeader
          title="Control actions"
          description="Action lifecycle from queue to execution. Approval and execution are distinct operations."
        />
        <button
          type="button"
          onClick={() => { void refresh(); void ctx.refresh() }}
          className="button-secondary"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      <OperatorTruthRibbon summary="Submission, approval, dispatch, and execution are separate lifecycle states. The console reflects stored records and capabilities — it does not bypass governance on the server." />

      {ctx.error && (
        <AlertCard variant="warning" title="Operator context unavailable" description={ctx.error} />
      )}

      {!ctx.loading && !canRead && (
        <MelPanelInset tone="info" className="flex items-center gap-2 text-xs text-muted-foreground">
          <Eye className="h-3.5 w-3.5 text-signal-observed shrink-0" aria-hidden />
          Limited visibility. Your session may lack read_actions capability.
        </MelPanelInset>
      )}

      {incidentFromUrl && (
        <MelPanelInset className="flex flex-wrap items-center gap-2 text-sm">
          <span className="text-muted-foreground">Filtered to incident</span>
          <code className="font-mono text-xs bg-muted/40 px-2 py-0.5 rounded">{incidentFromUrl}</code>
          <Link
            to={returnParam.startsWith('/') ? `/control-actions?return=${encodeURIComponent(returnParam)}` : '/control-actions'}
            className="text-xs font-semibold text-primary hover:underline"
          >
            Clear filter
          </Link>
          {returnParam.startsWith('/') && (
            <Link to={returnParam} className="text-xs font-semibold text-primary hover:underline">
              ← Back
            </Link>
          )}
          <Link
            to={withReturnParam(`/incidents/${encodeURIComponent(incidentFromUrl)}`, returnParam)}
            className="text-xs font-semibold text-muted-foreground hover:text-foreground ml-auto"
          >
            Open incident →
          </Link>
        </MelPanelInset>
      )}

      <MelSegment label="Lifecycle" radiogroupLabel="Filter by lifecycle state">
        {LIFECYCLE_FILTERS.map((f) => (
          <MelSegmentItem
            key={f.value || 'all'}
            role="radio"
            aria-checked={filter === f.value}
            onClick={() => setFilter(f.value)}
            active={filter === f.value}
          >
            {f.label}
          </MelSegmentItem>
        ))}
      </MelSegment>

      {rows.length === 0 ? (
        <EmptyState
          type="no-data"
          title="No actions in this view"
          description="Try another lifecycle filter or confirm transports are generating control decisions."
        />
      ) : (
        <div className="space-y-3">
          {rows.map((a) => (
            <ActionCard key={a.id} action={a} incidentQuery={incidentFromUrl} />
          ))}
        </div>
      )}
    </div>
  )
}

function ActionCard({ action: a, incidentQuery }: { action: ControlActionRecord; incidentQuery?: string }) {
  const [expanded, setExpanded] = useState(false)
  const phase = controlActionExecPhase(a)
  const isHighBlast = a.high_blast_radius || a.approval_escalated_due_to_blast_radius

  return (
    <Card className={clsx(isHighBlast && 'border-warning/25')}>
      <CardHeader className="pb-2">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <Zap className={clsx('h-4 w-4 shrink-0', phase.variant === 'critical' ? 'text-critical' : phase.variant === 'warning' ? 'text-warning' : 'text-primary')} />
              <CardTitle className="text-base">{a.action_type}</CardTitle>
            </div>
            <div className="mt-1 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
              <span className="font-mono">{a.id.slice(0, 12)}</span>
              {a.created_at && (
                <span className="inline-flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  {formatRelativeTime(a.created_at)}
                </span>
              )}
              {(a.submitted_by || a.proposed_by) && (
                <span className="inline-flex items-center gap-1">
                  <User className="h-3 w-3" />
                  {a.submitted_by || a.proposed_by}
                </span>
              )}
            </div>
          </div>
          <div className="flex flex-wrap gap-1.5">
            <Badge variant={phase.variant}>{phase.label}</Badge>
            {a.result && a.result !== a.lifecycle_state && <Badge variant="secondary">{a.result}</Badge>}
            {isHighBlast && (
              <Badge variant="warning">
                <AlertTriangle className="h-3 w-3" />
                high blast radius
              </Badge>
            )}
            {a.sod_bypass && (
              <span title={a.sod_bypass_reason || 'SoD bypass invoked'}>
              <Badge variant="critical" className="cursor-help">
                <Shield className="h-3 w-3" />
                SoD bypass
              </Badge>
              </span>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-3 pt-0 text-sm">
        {a.reason && <p className="text-muted-foreground">{a.reason}</p>}

        {/* Key facts row */}
        <div className="flex flex-wrap gap-x-5 gap-y-1.5 text-xs text-muted-foreground">
          {a.transport_name && (
            <span>Transport: <span className="font-mono text-foreground">{a.transport_name}</span></span>
          )}
          {a.target_segment && (
            <span>Segment: <span className="font-mono text-foreground">{a.target_segment}</span></span>
          )}
          {a.target_node && (
            <span>Node: <span className="font-mono text-foreground">{a.target_node}</span></span>
          )}
          {a.blast_radius_class && (
            <span>Blast radius: <span className="text-foreground">{a.blast_radius_class}</span></span>
          )}
        </div>

        {/* Approval chain */}
        <div className="flex flex-wrap items-center gap-2">
          {a.approved_by ? (
            <span className="inline-flex items-center gap-1 text-xs text-success">
              <CheckCircle2 className="h-3 w-3" />
              Approved by {a.approved_by}
              {a.approved_at && <span className="text-muted-foreground/60">({formatRelativeTime(a.approved_at)})</span>}
            </span>
          ) : a.rejected_by ? (
            <span className="inline-flex items-center gap-1 text-xs text-critical">
              <XCircle className="h-3 w-3" />
              Rejected by {a.rejected_by}
            </span>
          ) : a.lifecycle_state === 'pending_approval' ? (
            <span className="inline-flex items-center gap-1 text-xs text-warning">
              <Clock className="h-3 w-3" />
              Awaiting approval
              {a.requires_separate_approver && <span className="text-muted-foreground">(separate approver required)</span>}
            </span>
          ) : null}
        </div>

        {/* Cross-link to incident */}
        {a.incident_id && (
          <div className="flex flex-wrap items-center gap-2">
            <Link
              to={`/incidents/${encodeURIComponent(a.incident_id)}`}
              className="inline-flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"
            >
              <AlertTriangle className="h-3 w-3" />
              Open incident: {a.incident_id.slice(0, 12)}
              <ArrowRight className="h-3 w-3" />
            </Link>
            {(!incidentQuery || incidentQuery !== a.incident_id) && (
              <Link
                to={`/control-actions?incident=${encodeURIComponent(a.incident_id)}`}
                className="inline-flex items-center gap-1 text-mel-sm font-medium text-muted-foreground hover:text-foreground"
              >
                Filter list to this incident
              </Link>
            )}
          </div>
        )}

        {/* Expand for full metadata */}
        <button
          type="button"
          onClick={() => setExpanded(!expanded)}
          className="flex items-center gap-1 text-xs font-semibold text-muted-foreground transition-colors hover:text-foreground"
        >
          {expanded ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
          {expanded ? 'Less detail' : 'Full detail'}
        </button>

        {expanded && (
          <MelPanelInset tone="default" className="animate-fade-in space-y-2 bg-muted/10 p-3">
            <dl className="grid gap-x-6 gap-y-1.5 text-xs sm:grid-cols-2 lg:grid-cols-3">
              <MetaRow label="Lifecycle" value={a.lifecycle_state} />
              <MetaRow label="Result" value={a.result} />
              <MetaRow label="Execution mode" value={a.execution_mode} />
              <MetaRow label="Approval model" value={(a.approval_mode || 'single_approver').replace(/_/g, ' ')} />
              {a.required_approvals != null && <MetaRow label="Required approvals" value={String(a.required_approvals)} />}
              {a.collected_approvals != null && <MetaRow label="Collected approvals" value={String(a.collected_approvals)} />}
              <MetaRow label="Created" value={formatTimestamp(a.created_at)} />
              <MetaRow label="Execution started" value={formatTimestamp(a.execution_started_at)} />
              <MetaRow label="Completed" value={formatTimestamp(a.completed_at)} />
              {a.execution_source && <MetaRow label="Execution source" value={a.execution_source} />}
              {a.approval_policy_source && <MetaRow label="Policy source" value={a.approval_policy_source} />}
              {a.approval_basis && a.approval_basis.length > 0 && (
                <div className="sm:col-span-2 lg:col-span-3">
                  <dt className="text-mel-sm uppercase tracking-[0.14em] text-muted-foreground">Approval basis</dt>
                  <dd className="mt-0.5 font-mono text-foreground">{a.approval_basis.join(', ')}</dd>
                </div>
              )}
            </dl>
            {a.outcome_detail && (
              <MelPanelInset tone="dense" className="text-xs text-muted-foreground">
                <span className="font-medium text-foreground">Outcome:</span> {a.outcome_detail}
              </MelPanelInset>
            )}
          </MelPanelInset>
        )}
      </CardContent>
    </Card>
  )
}

function MetaRow({ label, value }: { label: string; value?: string }) {
  return (
    <div>
      <dt className="text-mel-sm uppercase tracking-[0.14em] text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 text-foreground">{value || '\u2014'}</dd>
    </div>
  )
}
