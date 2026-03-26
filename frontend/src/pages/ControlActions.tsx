import { useMemo, useState } from 'react'
import { useControlActions } from '@/hooks/useControlActions'
import { useOperatorContext } from '@/hooks/useOperatorContext'
import { PageHeader } from '@/components/ui/PageHeader'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatTimestamp, type ControlActionRecord } from '@/types/api'
import { RefreshCw } from 'lucide-react'

const LIFECYCLE_FILTERS = [
  { value: '', label: 'All' },
  { value: 'pending_approval', label: 'Awaiting approval' },
  { value: 'pending', label: 'Approved / queued' },
  { value: 'running', label: 'Executing' },
  { value: 'completed', label: 'Completed' },
]

function execPhase(a: ControlActionRecord): string {
  const ls = (a.lifecycle_state || '').toLowerCase()
  const res = (a.result || '').toLowerCase()
  if (ls === 'pending_approval') return 'Awaiting approval'
  if (ls === 'pending' && res === 'approved') return 'Approved, not yet executed'
  if (ls === 'running') return 'Executing'
  if (ls === 'completed') {
    if (res === 'rejected') return 'Rejected'
    if (res.includes('failed')) return 'Failed'
    return 'Finished'
  }
  return a.lifecycle_state || '—'
}

export function ControlActions() {
  const [filter, setFilter] = useState('')
  const { data, loading, error, refresh } = useControlActions(filter)
  const ctx = useOperatorContext()

  const rows = useMemo(() => data ?? [], [data])
  const canRead = ctx.trustUI?.read_actions === true || ctx.capabilities?.includes('read_actions')

  if (loading && !data) {
    return <Loading message="Loading control actions…" />
  }

  if (error && !data) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to load actions"
          description={error}
          action={
            <button
              type="button"
              onClick={() => void refresh()}
              className="rounded-lg bg-critical px-4 py-2 text-sm font-medium text-white hover:bg-critical/90"
            >
              Retry
            </button>
          }
        />
      </div>
    )
  }

  return (
    <div className="space-y-6 p-4 md:p-8">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <PageHeader
          title="Control actions"
          description="Canonical action lifecycle from the backend. Approval and execution are distinct: pending after approve means queued for the executor, not done."
        />
        <button
          type="button"
          onClick={() => {
            void refresh()
            void ctx.refresh()
          }}
          className="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-sm font-medium hover:bg-muted"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      {ctx.error && (
        <AlertCard variant="warning" title="Operator context unavailable" description={ctx.error} />
      )}

      {!ctx.loading && !canRead && (
        <AlertCard
          variant="info"
          title="Limited visibility"
          description="Your session may lack read_actions; if this list is empty or requests fail, use an API key or role that includes read_actions or read_status."
        />
      )}

      <div className="flex flex-wrap items-center gap-2">
        <span className="text-xs uppercase text-muted-foreground">Lifecycle</span>
        {LIFECYCLE_FILTERS.map((f) => (
          <button
            key={f.value || 'all'}
            type="button"
            onClick={() => setFilter(f.value)}
            className={`rounded-full border px-3 py-1 text-xs font-medium transition-colors ${
              filter === f.value
                ? 'border-primary bg-primary/10 text-primary'
                : 'border-border text-muted-foreground hover:bg-muted'
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {rows.length === 0 ? (
        <EmptyState
          type="no-data"
          title="No actions in this view"
          description="Try another lifecycle filter or confirm transports are generating control decisions."
        />
      ) : (
        <div className="grid gap-4">
          {rows.map((a) => (
            <ActionCard key={a.id} action={a} />
          ))}
        </div>
      )}
    </div>
  )
}

function ActionCard({ action: a }: { action: ControlActionRecord }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div>
            <CardTitle className="text-base font-semibold">{a.action_type}</CardTitle>
            <CardDescription className="font-mono text-xs">{a.id}</CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            {a.lifecycle_state && <Badge variant="outline">{a.lifecycle_state}</Badge>}
            {a.result && <Badge variant="secondary">{a.result}</Badge>}
            {a.sod_bypass && (
              <span title={a.sod_bypass_reason || 'SoD bypass'}>
                <Badge variant="critical">SoD bypass</Badge>
              </span>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-2 text-sm">
        <p className="text-muted-foreground">{a.reason || '—'}</p>
        <dl className="grid gap-1 sm:grid-cols-2 lg:grid-cols-3">
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Phase</dt>
            <dd>{execPhase(a)}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Transport</dt>
            <dd className="font-mono text-xs">{a.transport_name || '—'}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Created</dt>
            <dd>{formatTimestamp(a.created_at)}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Submitted by</dt>
            <dd className="font-mono text-xs">{a.submitted_by || a.proposed_by || '—'}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Separate approver</dt>
            <dd>{a.requires_separate_approver ? 'Yes' : 'No'}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Approval model</dt>
            <dd className="text-xs">
              {(a.approval_mode || 'single_approver').replace(/_/g, ' ')}
              {a.required_approvals != null ? ` · required ${a.required_approvals}` : ''}
              {a.collected_approvals != null ? ` · collected ${a.collected_approvals}` : ''}
            </dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Blast radius / policy</dt>
            <dd className="text-xs">
              {a.blast_radius_class || '—'}
              {a.high_blast_radius ? ' · high (mesh/global class)' : ''}
              {a.approval_escalated_due_to_blast_radius ? ' · gated by high-blast config' : ''}
            </dd>
          </div>
          {a.approval_basis && a.approval_basis.length > 0 && (
            <div className="sm:col-span-2 lg:col-span-3">
              <dt className="text-xs uppercase text-muted-foreground">Approval basis (config)</dt>
              <dd className="font-mono text-xs">{a.approval_basis.join(', ')}</dd>
            </div>
          )}
          {a.execution_source && (
            <div>
              <dt className="text-xs uppercase text-muted-foreground">Last execution source</dt>
              <dd className="font-mono text-xs">{a.execution_source}</dd>
            </div>
          )}
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Approved</dt>
            <dd className="text-xs">
              {a.approved_by ? (
                <>
                  <span className="font-mono">{a.approved_by}</span>
                  {a.approved_at && <span className="text-muted-foreground"> · {formatTimestamp(a.approved_at)}</span>}
                </>
              ) : (
                '—'
              )}
            </dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Execution started</dt>
            <dd>{formatTimestamp(a.execution_started_at)}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Completed</dt>
            <dd>{formatTimestamp(a.completed_at)}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Incident</dt>
            <dd className="font-mono text-xs">{a.incident_id || '—'}</dd>
          </div>
        </dl>
        {a.outcome_detail && (
          <p className="rounded-md border border-border bg-muted/30 p-2 text-xs text-muted-foreground">
            {a.outcome_detail}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
