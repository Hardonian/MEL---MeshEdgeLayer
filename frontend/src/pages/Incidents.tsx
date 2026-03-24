import { useIncidents } from '@/hooks/useIncidents'
import { useOperatorContext } from '@/hooks/useOperatorContext'
import { PageHeader } from '@/components/ui/PageHeader'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatTimestamp, type ControlActionRecord, type Incident } from '@/types/api'
import { ClipboardCopy, RefreshCw } from 'lucide-react'

function isOpenIncident(inc: Incident): boolean {
  const s = (inc.state || '').toLowerCase()
  return s !== 'resolved' && s !== 'closed'
}

function copyText(text: string) {
  void navigator.clipboard.writeText(text)
}

export function Incidents() {
  const { data, loading, error, refresh } = useIncidents()
  const ctx = useOperatorContext()

  if (loading && !data) {
    return <Loading message="Loading incidents…" />
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
              className="rounded-lg bg-critical px-4 py-2 text-sm font-medium text-white hover:bg-critical/90"
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
  const canHandoff = ctx.trustUI?.incident_handoff_write === true

  return (
    <div className="space-y-6 p-4 md:p-8">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <PageHeader
          title="Incidents"
          description="Open incidents with durable handoff context. Linked control actions show the canonical incident_id FK from the database. Pending action IDs remain operator-supplied references from handoff when used."
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

      {!canHandoff && !ctx.loading && (
        <AlertCard
          variant="info"
          title="Read-only incident view"
          description="Your credentials do not include incident_handoff_write. You can list incidents but cannot record handoff via the API from this session."
        />
      )}

      {openIncidents.length === 0 ? (
        <EmptyState
          type="no-data"
          title="No open incidents"
          description={
            incidents.length === 0
              ? 'There are no incidents in the recent list. When transport or system incidents are raised, they appear here with owner and handoff fields when recorded.'
              : 'All recent incidents are resolved or closed. Expand history below if needed.'
          }
        />
      ) : (
        <div className="grid gap-4">
          {openIncidents.map((inc) => (
            <IncidentCard key={inc.id} incident={inc} />
          ))}
        </div>
      )}

      {incidents.length > openIncidents.length && (
        <section className="space-y-2">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
            Other recent incidents
          </h2>
          <div className="grid gap-3">
            {incidents
              .filter((i) => !isOpenIncident(i))
              .map((inc) => (
                <IncidentCard key={inc.id} incident={inc} muted />
              ))}
          </div>
        </section>
      )}
    </div>
  )
}

function IncidentCard({ incident: inc, muted = false }: { incident: Incident; muted?: boolean }) {
  const pending = inc.pending_actions?.filter(Boolean) ?? []
  const linked = inc.linked_control_actions ?? []
  const hasHandoffText = !!(inc.handoff_summary && inc.handoff_summary.trim())
  const owner = inc.owner_actor_id?.trim()

  return (
    <Card className={muted ? 'opacity-80' : ''}>
      <CardHeader className="pb-2">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div>
            <CardTitle className="text-lg font-semibold">{inc.title || inc.id}</CardTitle>
            <CardDescription className="font-mono text-xs">{inc.id}</CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            {inc.state && <Badge variant="outline">{inc.state}</Badge>}
            {inc.severity && <Badge variant="secondary">{inc.severity}</Badge>}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        {inc.summary && <p className="text-muted-foreground">{inc.summary}</p>}
        <dl className="grid gap-1 sm:grid-cols-2">
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Owner</dt>
            <dd className="font-mono text-xs">{owner || '—'}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Updated</dt>
            <dd>{formatTimestamp(inc.updated_at)}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Created</dt>
            <dd>{formatTimestamp(inc.occurred_at)}</dd>
          </div>
        </dl>
        <div>
          <div className="text-xs uppercase text-muted-foreground">Handoff summary</div>
          <div className="mt-1 rounded-md border border-border bg-muted/30 p-2 text-sm">
            {hasHandoffText ? inc.handoff_summary : 'No handoff summary recorded.'}
          </div>
        </div>
        <div>
          <div className="mb-1 text-xs uppercase text-muted-foreground">Linked control actions</div>
          {linked.length === 0 ? (
            <p className="text-muted-foreground">None linked via incident_id on control actions.</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {linked.map((a) => (
                <li
                  key={a.id}
                  className="rounded-md border border-border bg-background px-2 py-1.5 text-xs"
                >
                  <LinkedActionSummary action={a} />
                </li>
              ))}
            </ul>
          )}
        </div>
        <div>
          <div className="mb-1 text-xs uppercase text-muted-foreground">Pending action IDs</div>
          {pending.length === 0 ? (
            <p className="text-muted-foreground">None recorded for this incident.</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {pending.map((id) => (
                <li
                  key={id}
                  className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-border bg-background px-2 py-1.5"
                >
                  <code className="text-xs break-all">{id}</code>
                  <button
                    type="button"
                    onClick={() => copyText(id)}
                    className="inline-flex shrink-0 items-center gap-1 rounded border border-border px-2 py-0.5 text-xs hover:bg-muted"
                    title="Copy action id"
                  >
                    <ClipboardCopy className="h-3 w-3" />
                    Copy
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function LinkedActionSummary({ action: a }: { action: ControlActionRecord }) {
  return (
    <div className="flex flex-wrap items-baseline justify-between gap-2">
      <div>
        <span className="font-mono">{a.id}</span>
        <span className="text-muted-foreground"> · {a.action_type}</span>
      </div>
      <div className="flex flex-wrap gap-1">
        {a.lifecycle_state && (
          <Badge variant="outline" className="text-[10px]">
            {a.lifecycle_state}
          </Badge>
        )}
        {a.result && (
          <Badge variant="secondary" className="text-[10px]">
            {a.result}
          </Badge>
        )}
      </div>
    </div>
  )
}
