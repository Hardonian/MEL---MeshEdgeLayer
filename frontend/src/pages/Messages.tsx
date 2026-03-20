import { useMessages } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { ErrorView, EmptyState, TableSkeleton } from '@/components/ui/StateViews'
import { formatTimestamp, Message } from '@/types/api'
import { MessageSquare, Clock, ArrowRight } from 'lucide-react'

export function Messages() {
  const { data, loading, error, refresh } = useMessages()

  if (loading && !data) {
    return (
      <div className="space-y-6">
        <div className="flex flex-col gap-2">
          <h1 className="text-3xl font-bold tracking-tight">Messages</h1>
          <p className="text-muted-foreground">Recent mesh message observations.</p>
        </div>
        <TableSkeleton rows={8} />
      </div>
    )
  }

  if (error && !data) {
    return <ErrorView message={error} onRetry={refresh} />
  }

  const messages = data || []

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight">Messages</h1>
        <p className="text-muted-foreground">
          Recent mesh message observations captured by your transports.
        </p>
      </div>

      {/* Messages Table */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Messages</CardTitle>
          <CardDescription>
            Last {messages.length} messages observed on the mesh
          </CardDescription>
        </CardHeader>
        <CardContent>
          {messages.length === 0 ? (
            <EmptyState
              icon={<MessageSquare className="h-10 w-10" />}
              title="No messages yet"
              description="No live message observations have been stored yet. This is expected when transports are idle or disconnected."
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b text-left">
                    <th className="pb-3 text-sm font-medium text-muted-foreground">Time</th>
                    <th className="pb-3 text-sm font-medium text-muted-foreground">Transport</th>
                    <th className="pb-3 text-sm font-medium text-muted-foreground">From</th>
                    <th className="pb-3 text-sm font-medium text-muted-foreground">To</th>
                    <th className="pb-3 text-sm font-medium text-muted-foreground">Port</th>
                    <th className="pb-3 text-sm font-medium text-muted-foreground">Payload</th>
                  </tr>
                </thead>
                <tbody>
                  {messages.map((msg, i) => (
                    <MessageRow key={i} message={msg} />
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

function MessageRow({ message }: { message: Message }) {
  return (
    <tr className="border-b last:border-0 hover:bg-muted/50">
      <td className="py-3">
        <div className="flex items-center gap-2 text-sm">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <span className="font-mono text-xs">{formatTimestamp(message.rx_time)}</span>
        </div>
      </td>
      <td className="py-3">
        <span className="text-sm">{message.transport_name}</span>
      </td>
      <td className="py-3">
        <code className="rounded bg-muted px-2 py-0.5 text-xs font-mono">
          {message.from_node}
        </code>
      </td>
      <td className="py-3">
        <div className="flex items-center gap-1">
          <ArrowRight className="h-3 w-3 text-muted-foreground" />
          <code className="rounded bg-muted px-2 py-0.5 text-xs font-mono">
            {message.to_node}
          </code>
        </div>
      </td>
      <td className="py-3">
        <span className="text-sm font-mono">{message.portnum}</span>
      </td>
      <td className="py-3">
        <span className="text-sm font-mono truncate max-w-xs block">
          {message.payload_text || '—'}
        </span>
      </td>
    </tr>
  )
}
