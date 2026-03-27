import { useMessages } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { NoMessagesYet } from '@/components/ui/EmptyState'
import { formatTimestamp, Message } from '@/types/api'
import { TruncatedText } from '@/components/ui/TruncatedText'
import { MessageSquare, Clock, ArrowRight, Hash } from 'lucide-react'

export function Messages() {
  const { data, loading, error, refresh } = useMessages()

  if (loading && !data) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Messages"
          description="Recent mesh message observations captured by your transports."
        />
        <Card>
          <CardHeader>
            <CardTitle>Recent Messages</CardTitle>
            <CardDescription>Messages observed on the mesh</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable<Message>
              data={[]}
              columns={[
                { key: 'time', header: 'Time' },
                { key: 'transport', header: 'Transport' },
                { key: 'from', header: 'From' },
                { key: 'to', header: 'To' },
                { key: 'port', header: 'Port' },
                { key: 'payload', header: 'Payload' },
              ]}
              keyField="packet_id"
              isLoading={true}
            />
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error && !data) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to load messages"
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

  const messages = data || []

  // Get unique transports for stats
  const transports = [...new Set(messages.map(m => m.transport_name))]

  return (
    <div className="space-y-6">
      <PageHeader
        title="Messages"
        description="Recent mesh message observations captured by your transports."
      />

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard
          title="Total Messages"
          value={messages.length}
          description="Messages in current view"
          icon={<MessageSquare className="h-5 w-5" />}
          variant="default"
        />
        <StatCard
          title="Active Transports"
          value={transports.length}
          description="Transports with messages"
          icon={<Hash className="h-5 w-5" />}
          variant="info"
        />
        <StatCard
          title="Time Range"
          value={messages.length > 0 ? 'Recent window' : 'N/A'}
          description={
            messages.length > 0
              ? 'Observations in this list (not a live stream guarantee)'
              : 'No messages in this view yet'
          }
          icon={<Clock className="h-5 w-5" />}
          variant={messages.length > 0 ? 'success' : 'default'}
        />
      </div>

      {/* Messages Table */}
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Recent Messages</CardTitle>
              <CardDescription>
                {messages.length > 0 
                  ? `Last ${messages.length} messages observed on the mesh`
                  : 'Messages observed on the mesh'
                }
              </CardDescription>
            </div>
            <Badge variant="outline">{messages.length} messages</Badge>
          </div>
        </CardHeader>
        <CardContent>
          {messages.length === 0 ? (
            <NoMessagesYet />
          ) : (
            <DataTable<Message>
              data={messages}
              columns={[
                {
                  key: 'time',
                  header: 'Time',
                  render: (msg) => (
                    <div className="flex items-center gap-2">
                      <Clock className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                      <span className="font-mono text-xs whitespace-nowrap">
                        {formatTimestamp(msg.rx_time)}
                      </span>
                    </div>
                  ),
                },
                {
                  key: 'transport',
                  header: 'Transport',
                  render: (msg) => (
                    <Badge variant="outline" className="text-xs">
                      {msg.transport_name}
                    </Badge>
                  ),
                },
                {
                  key: 'from',
                  header: 'From',
                  render: (msg) => (
                    <code className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono">
                      {msg.from_node}
                    </code>
                  ),
                },
                {
                  key: 'to',
                  header: 'To',
                  render: (msg) => (
                    <div className="flex items-center gap-1">
                      <ArrowRight className="h-3 w-3 text-muted-foreground" />
                      <code className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono">
                        {msg.to_node}
                      </code>
                    </div>
                  ),
                },
                {
                  key: 'port',
                  header: 'Port',
                  render: (msg) => (
                    <span className="font-mono text-xs">{msg.portnum}</span>
                  ),
                },
                {
                  key: 'payload',
                  header: 'Payload',
                  render: (msg) =>
                    msg.payload_text ? (
                      <TruncatedText text={msg.payload_text} maxLen={56} className="block max-w-[min(20rem,100%)]" />
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    ),
                },
              ]}
              keyField="packet_id"
              emptyMessage="No messages yet"
              emptyDescription="Messages will appear here once mesh traffic is observed."
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
