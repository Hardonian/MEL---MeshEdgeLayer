import { useNodes } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { NoNodesYet } from '@/components/ui/EmptyState'
import { formatRelativeTime, NodeInfo } from '@/types/api'
import { Radio, Clock, MapPin } from 'lucide-react'

export function Nodes() {
  const { data, loading, error, refresh } = useNodes()

  if (loading && !data) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Nodes"
          description="Mesh device inventory — all nodes observed by your transports."
        />
        <Card>
          <CardHeader>
            <CardTitle>Node Inventory</CardTitle>
            <CardDescription>All mesh nodes observed by your MEL instance</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable<NodeInfo>
              data={[]}
              columns={[
                { key: 'name', header: 'Node' },
                { key: 'id', header: 'ID' },
                { key: 'last_seen', header: 'Last Seen' },
                { key: 'gateway_id', header: 'Gateway' },
              ]}
              keyField="id"
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
              className="rounded-lg bg-critical px-4 py-2 text-sm font-medium text-white hover:bg-critical/90"
            >
              Retry
            </button>
          }
        />
      </div>
    )
  }

  const nodes = data || []

  return (
    <div className="space-y-6">
      <PageHeader
        title="Nodes"
        description="Mesh device inventory — all nodes observed by your transports."
      />

      {/* Stats */}
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
          value={nodes.filter(n => {
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
          value={new Set(nodes.filter(n => n.gateway_id).map(n => n.gateway_id)).size}
          description="Unique gateway nodes"
          icon={<MapPin className="h-5 w-5" />}
          variant="info"
        />
      </div>

      {/* Nodes Table */}
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Node Inventory</CardTitle>
              <CardDescription>
                All mesh nodes observed by your MEL instance
              </CardDescription>
            </div>
            <Badge variant="outline">{nodes.length} total</Badge>
          </div>
        </CardHeader>
        <CardContent>
          {nodes.length === 0 ? (
            <NoNodesYet />
          ) : (
            <DataTable<NodeInfo>
              data={nodes}
              columns={[
                {
                  key: 'name',
                  header: 'Node',
                  render: (node) => (
                    <div className="flex items-center gap-3">
                      <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-secondary">
                        <Radio className="h-4 w-4 text-secondary-foreground" />
                      </div>
                      <div>
                        <p className="font-medium">{node.long_name || 'Unknown Node'}</p>
                        <p className="text-xs text-muted-foreground">{node.short_name || '—'}</p>
                      </div>
                    </div>
                  ),
                },
                {
                  key: 'id',
                  header: 'ID',
                  render: (node) => (
                    <code className="rounded bg-muted px-2 py-1 text-xs font-mono">
                      {node.id}
                    </code>
                  ),
                },
                {
                  key: 'last_seen',
                  header: 'Last Seen',
                  render: (node) => (
                    <div className="flex items-center gap-2 text-sm">
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
                      {node.gateway_id ? (
                        <>
                          <MapPin className="h-3.5 w-3.5 text-muted-foreground" />
                          <code className="font-mono text-xs">{node.gateway_id}</code>
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
                        <span className="text-xs">{node.user.hw_model || 'Unknown hardware'}</span>
                      </div>
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )
                  ),
                },
              ]}
              keyField="id"
              emptyMessage="No nodes observed"
              emptyDescription="Nodes will appear here once mesh traffic is observed."
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
