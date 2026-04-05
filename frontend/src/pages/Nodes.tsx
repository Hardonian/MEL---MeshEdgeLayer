import { useNodes } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { OperatorEmptyState } from '@/components/states/OperatorEmptyState'
import { StaleDataBanner } from '@/components/states/StaleDataBanner'
import { formatRelativeTime, NodeInfo } from '@/types/api'
import { Radio, Clock, MapPin } from 'lucide-react'

export function Nodes() {
  const { data, loading, error, refresh } = useNodes()

  if (loading && !data) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Nodes"
          subtitle="Mesh operations cockpit"
          description="Mesh device inventory. All nodes observed by your transports remain listed with the same evidence and timing semantics as the backend."
        />
        <Card className="overflow-hidden">
          <CardHeader className="border-b border-border/50">
            <CardTitle>Node Inventory</CardTitle>
            <CardDescription>All mesh nodes observed by your MEL instance.</CardDescription>
          </CardHeader>
          <CardContent className="pt-5">
            <DataTable<NodeInfo>
              data={[]}
              columns={[
                { key: 'name', header: 'Node' },
                { key: 'node_id', header: 'ID' },
                { key: 'last_seen', header: 'Last Seen' },
                { key: 'last_gateway_id', header: 'Gateway' },
              ]}
              keyField="node_id"
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
          title="Unable to load nodes"
          description={error}
          action={
            <button
              onClick={refresh}
              className="button-danger"
            >
              Retry
            </button>
          }
        />
      </div>
    )
  }

  const nodes = data || []

  const newestLastSeen = nodes.reduce((max, node) => {
    if (!node.last_seen) return max
    const t = new Date(node.last_seen).getTime()
    return t > max ? t : max
  }, 0)
  const staleTimestamp = newestLastSeen ? new Date(newestLastSeen).toISOString() : undefined

  return (
    <div className="space-y-6">
      <PageHeader
        title="Nodes"
        subtitle="Mesh operations cockpit"
        description="Mesh device inventory for this MEL instance. Rows stay grounded in observed mesh data rather than inferred topology state."
        action={<Badge variant="outline">{nodes.length} total</Badge>}
      />

      <StaleDataBanner lastSuccessfulIngest={staleTimestamp} componentName="Node Inventory" />

      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard
          title="Total Nodes"
          value={nodes.length}
          description="Devices observed on the mesh"
          icon={<Radio className="h-5 w-5" />}
          variant="default"
        />
        <StatCard
          title="Recently Active"
          value={nodes.filter((n) => {
            const lastSeen = n.last_seen ? new Date(n.last_seen) : null
            if (!lastSeen) return false
            const hourAgo = new Date(Date.now() - 60 * 60 * 1000)
            return lastSeen > hourAgo
          }).length}
          description="Active in the last hour"
          icon={<Clock className="h-5 w-5" />}
          variant="success"
        />
        <StatCard
          title="Known Gateways"
          value={new Set(nodes.filter((n) => n.last_gateway_id).map((n) => n.last_gateway_id)).size}
          description="Unique gateway nodes"
          icon={<MapPin className="h-5 w-5" />}
          variant="info"
        />
      </div>

      <Card className="overflow-hidden">
        <CardHeader className="border-b border-border/50 pb-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle>Node Inventory</CardTitle>
              <CardDescription className="mt-1">
                All mesh nodes observed by your MEL instance.
              </CardDescription>
            </div>
            <Badge variant="outline">{nodes.length} total</Badge>
          </div>
        </CardHeader>
        <CardContent className="pt-5">
          {nodes.length === 0 ? (
            <OperatorEmptyState title="No nodes yet" description="Nodes will appear here once mesh traffic is observed via connected transports." />
          ) : (
            <DataTable<NodeInfo>
              data={nodes}
              columns={[
                {
                  key: 'name',
                  header: 'Node',
                  render: (node) => (
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-md border border-border/70 bg-secondary text-secondary-foreground">
                        <Radio className="h-4 w-4" />
                      </div>
                      <div className="min-w-0">
                        <p className="truncate text-sm font-semibold text-foreground">{node.long_name || 'Unknown Node'}</p>
                        <p className="truncate text-xs uppercase tracking-[0.16em] text-muted-foreground">
                          {node.short_name || 'Unnamed'}
                        </p>
                      </div>
                    </div>
                  ),
                },
                {
                  key: 'id',
                  header: 'ID',
                  render: (node) => (
                    <code className="raw-block inline-flex px-2 py-1 text-xs font-mono text-foreground">
                      {node.node_id}
                    </code>
                  ),
                },
                {
                  key: 'last_seen',
                  header: 'Last Seen',
                  render: (node) => (
                    <div className="flex items-center gap-2 text-sm text-foreground">
                      <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                      {formatRelativeTime(node.last_seen)}
                    </div>
                  ),
                },
                {
                  key: 'gateway_id',
                  header: 'Gateway',
                  render: (node) => (
                    <div className="flex items-center gap-2">
                      {node.last_gateway_id ? (
                        <>
                          <MapPin className="h-3.5 w-3.5 text-muted-foreground" />
                          <code className="font-mono text-xs text-foreground">{node.last_gateway_id}</code>
                        </>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </div>
                  ),
                },
                {
                  key: 'user',
                  header: 'User Info',
                  render: (node) => (
                    node.user ? (
                      <div className="flex flex-col gap-0.5">
                        <span className="text-xs text-foreground">{node.user.hw_model || 'Unknown hardware'}</span>
                      </div>
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )
                  ),
                },
              ]}
              keyField="node_id"
              emptyMessage="No nodes observed"
              emptyDescription="Nodes will appear here once mesh traffic is observed."
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
