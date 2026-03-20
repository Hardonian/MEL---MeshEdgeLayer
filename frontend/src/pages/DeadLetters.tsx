import { useDeadLetters } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Loading, ErrorView, EmptyState } from '@/components/ui/StateViews'
import { formatTimestamp } from '@/types/api'
import { AlertTriangle, Inbox } from 'lucide-react'

export function DeadLetters() {
  const { data, loading, error, refresh } = useDeadLetters()

  if (loading && !data) {
    return <Loading message="Loading dead letters..." />
  }

  if (error && !data) {
    return <ErrorView message={error} onRetry={refresh} />
  }

  const deadLetters = data || []

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight">Dead Letters</h1>
        <p className="text-muted-foreground">
          Messages that failed to be processed by transports.
        </p>
      </div>

      {/* Summary */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-warning/10">
                <AlertTriangle className="h-6 w-6 text-warning" />
              </div>
              <div>
                <p className="text-2xl font-bold">{deadLetters.length}</p>
                <p className="text-sm text-muted-foreground">Total Dead Letters</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Dead Letters Table */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Dead Letters</CardTitle>
          <CardDescription>
            Messages that could not be processed and were stored for inspection
          </CardDescription>
        </CardHeader>
        <CardContent>
          {deadLetters.length === 0 ? (
            <EmptyState
              icon={<Inbox className="h-10 w-10" />}
              title="No dead letters"
              description="No persisted transport dead letters are currently stored. This means all messages are being processed successfully."
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b text-left">
                    <th className="pb-3 text-sm font-medium text-muted-foreground">Time</th>
                    <th className="pb-3 text-sm font-medium text-muted-foreground">Transport</th>
                    <th className="pb-3 text-sm font-medium text-muted-foreground">Type</th>
                    <th className="pb-3 text-sm font-medium text-muted-foreground">Topic</th>
                    <th className="pb-3 text-sm font-medium text-muted-foreground">Reason</th>
                  </tr>
                </thead>
                <tbody>
                  {deadLetters.map((dl, i) => (
                    <tr key={i} className="border-b last:border-0 hover:bg-muted/50">
                      <td className="py-3">
                        <span className="font-mono text-sm">{formatTimestamp(dl.created_at)}</span>
                      </td>
                      <td className="py-3">
                        <span className="text-sm">{dl.transport_name}</span>
                      </td>
                      <td className="py-3">
                        <Badge variant="outline">{dl.transport_type}</Badge>
                      </td>
                      <td className="py-3">
                        <code className="text-sm font-mono">{dl.topic || '—'}</code>
                      </td>
                      <td className="py-3">
                        <span className="text-sm">{dl.reason}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
