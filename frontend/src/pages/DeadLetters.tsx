import { useDeadLetters } from '@/hooks/useApi'
import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { MelPageSection, MelPanel, MelPanelInset } from '@/components/ui/operator'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatTimestamp, DeadLetter } from '@/types/api'
import { AlertTriangle, Inbox, Clock, HelpCircle } from 'lucide-react'

export function DeadLetters() {
  const { data, loading, error, refresh } = useDeadLetters()

  if (loading && !data) {
    return <Loading message="Loading dead letters..." />
  }

  if (error && !data) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to load dead letters"
          description={error}
          action={
            <button type="button" onClick={refresh} className="button-danger">
              Retry
            </button>
          }
        />
      </div>
    )
  }

  const deadLetters = data || []
  const hasDeadLetters = deadLetters.length > 0

  return (
    <div className="space-y-6">
      <PageHeader
        title="Dead letters"
        description="Ingest failures persisted by this MEL instance for inspection. Presence here is transport evidence of a failed message, not a live queue guarantee."
      />

      {/* Summary */}
      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard
          title="Total Dead Letters"
          value={deadLetters.length}
          description="Messages that failed processing"
          icon={<AlertTriangle className="h-4 w-4" />}
          variant={hasDeadLetters ? 'warning' : 'success'}
          rhythm="console"
        />
        <StatCard
          title="Affected Transports"
          value={new Set(deadLetters.map(d => d.transport_name)).size}
          description="Transports with failures"
          icon={<Inbox className="h-4 w-4" />}
          variant="info"
          rhythm="console"
        />
        <StatCard
          title="Most Recent"
          value={hasDeadLetters ? formatTimestamp(deadLetters[0]?.created_at).split(',')[0] : 'N/A'}
          description={hasDeadLetters ? 'First failure in queue' : 'No failures'}
          icon={<Clock className="h-4 w-4" />}
          variant="default"
          rhythm="console"
        />
      </div>

      {/* Alert */}
      {hasDeadLetters && (
        <AlertCard
          variant="warning"
          title={`${deadLetters.length} dead letter${deadLetters.length > 1 ? 's' : ''} require attention`}
          description="These messages could not be processed. Review the reasons below and address the underlying issues."
        />
      )}

      <MelPageSection
        eyebrow="Reference"
        title="What dead letters mean here"
        description="Operator aid for failed ingest — bounded to what this instance stored."
      >
        <MelPanelInset tone="info" className="text-sm text-muted-foreground space-y-2">
          <p className="flex items-start gap-2">
            <HelpCircle className="h-4 w-4 shrink-0 mt-0.5 text-signal-observed" aria-hidden />
            <span>
              Typical causes include invalid or incomplete payloads, validation failures, transport disconnects during handling, or local storage errors. Treat the list as historical evidence, not a real-time failure rate.
            </span>
          </p>
          <p className="text-mel-xs">Retention is configurable under Settings on this instance.</p>
        </MelPanelInset>
      </MelPageSection>

      <MelPanel className="overflow-hidden">
        <CardHeader className="pb-4 px-4 pt-4">
          <div className="flex items-center justify-between">
            <CardTitle>Recent Dead Letters</CardTitle>
            <Badge variant="outline">{deadLetters.length} total</Badge>
          </div>
          <CardDescription>
            Messages that could not be processed and were stored for inspection
          </CardDescription>
        </CardHeader>
        <CardContent className="px-4 pb-4">
          {deadLetters.length === 0 ? (
            <EmptyState
              type="default"
              title="No dead letters"
              description="No dead-letter rows in the store for this view. That reflects persisted dead-letter records only — not proof that every ingest path always succeeds."
            />
          ) : (
            <DataTable<DeadLetter>
              data={deadLetters}
              columns={[
                {
                  key: 'time',
                  header: 'Time',
                  render: (dl) => (
                    <div className="flex items-center gap-2">
                      <Clock className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                      <span className="font-mono text-xs whitespace-nowrap">
                        {formatTimestamp(dl.created_at)}
                      </span>
                    </div>
                  ),
                },
                {
                  key: 'transport',
                  header: 'Transport',
                  render: (dl) => (
                    <span className="text-sm font-medium">{dl.transport_name}</span>
                  ),
                },
                {
                  key: 'type',
                  header: 'Type',
                  render: (dl) => (
                    <Badge variant="outline">{dl.transport_type}</Badge>
                  ),
                },
                {
                  key: 'topic',
                  header: 'Topic',
                  render: (dl) => (
                    <code className="text-xs font-mono">{dl.topic || '—'}</code>
                  ),
                },
                {
                  key: 'reason',
                  header: 'Reason',
                  render: (dl) => (
                    <span className="break-words text-sm">{dl.reason}</span>
                  ),
                },
              ]}
              keyField="created_at"
              emptyMessage="No dead letters in this slice"
              emptyDescription="Nothing to show for the current filter or window."
            />
          )}
        </CardContent>
      </MelPanel>
    </div>
  )
}
