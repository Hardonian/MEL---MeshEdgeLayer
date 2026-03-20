import { useDeadLetters } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { AlertCard, InlineAlert } from '@/components/ui/AlertCard'
import { Loading, EmptyState } from '@/components/ui/EmptyState'
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
            <button
              onClick={refresh}
              className="rounded-lg bg-critical px-4 py-2 text-sm font-medium text-white hover:bg-critical/90"
            >
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
        title="Dead Letters"
        description="Messages that failed to be processed by transports. Dead letters are stored for inspection and debugging."
      />

      {/* Summary */}
      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard
          title="Total Dead Letters"
          value={deadLetters.length}
          description="Messages that failed processing"
          icon={<AlertTriangle className="h-5 w-5" />}
          variant={hasDeadLetters ? 'warning' : 'success'}
        />
        <StatCard
          title="Affected Transports"
          value={new Set(deadLetters.map(d => d.transport_name)).size}
          description="Transports with failures"
          icon={<Inbox className="h-5 w-5" />}
          variant="info"
        />
        <StatCard
          title="Most Recent"
          value={hasDeadLetters ? formatTimestamp(deadLetters[0]?.created_at).split(',')[0] : 'N/A'}
          description={hasDeadLetters ? 'First failure in queue' : 'No failures'}
          icon={<Clock className="h-5 w-5" />}
          variant="default"
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

      {/* Explanation */}
      <Card className="bg-muted/30">
        <CardHeader className="pb-3">
          <div className="flex items-center gap-2">
            <HelpCircle className="h-5 w-5 text-muted-foreground" />
            <CardTitle className="text-base">About Dead Letters</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="pt-0">
          <div className="prose prose-sm dark:prose-invert max-w-none">
            <p className="text-sm text-muted-foreground">
              Dead letters are messages that MEL was unable to process. This can happen due to:
            </p>
            <ul className="mt-2 list-disc pl-4 text-sm text-muted-foreground space-y-1">
              <li>Invalid message format or malformed payload</li>
              <li>Missing required fields or metadata</li>
              <li>Transport connection failures during processing</li>
              <li>Schema validation failures</li>
              <li>Storage or database errors</li>
            </ul>
            <p className="mt-2 text-sm text-muted-foreground">
              Dead letters are retained for debugging purposes. You can configure retention periods in Settings.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Dead Letters Table */}
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between">
            <CardTitle>Recent Dead Letters</CardTitle>
            <Badge variant="outline">{deadLetters.length} total</Badge>
          </div>
          <CardDescription>
            Messages that could not be processed and were stored for inspection
          </CardDescription>
        </CardHeader>
        <CardContent>
          {deadLetters.length === 0 ? (
            <EmptyState
              type="default"
              title="No dead letters"
              description="No persisted transport dead letters are currently stored. This means all messages are being processed successfully."
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
                    <span className="text-sm">{dl.reason}</span>
                  ),
                },
              ]}
              keyField="created_at"
              emptyMessage="No dead letters"
              emptyDescription="No failed messages to display."
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
