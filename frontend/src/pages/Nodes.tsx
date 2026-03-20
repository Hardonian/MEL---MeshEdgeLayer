import { useNodes } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { ErrorView, EmptyState, TableSkeleton } from '@/components/ui/StateViews'
import { formatRelativeTime, NodeInfo } from '@/types/api'
import { Radio, Clock } from 'lucide-react'

export function Nodes() {
  const { data, loading, error, refresh } = useNodes()

  if (loading && !data) {
    return (
      <div className="space-y-6">
        <div className="flex flex-col gap-2">
          <h1 className="text-3xl font-bold tracking-tight">Nodes</h1>
          <p className="text-muted-foreground">Inventory of observed mesh devices.</p>
        </div>
        <TableSkeleton rows={8} />
      </div>
    )
  }

  if (error && !data) {
    return <ErrorView message={error} onRetry={refresh} />
  }

  const nodes = data || []

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight">Nodes</h1>
        <p className="text-muted-foreground">
          Mesh device inventory — all nodes observed by your transports.
        </p>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10">
                <Radio className="h-6 w-6 text-primary" />
              </div>
              <div>
                <p className="text-2xl font-bold">{nodes.length}</p>
                <p className="text-sm text-muted-foreground">Total Nodes</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Nodes Table */}
      <Card>
        <CardHeader>
          <CardTitle>Node Inventory</CardTitle>
          <CardDescription>
            All mesh nodes observed by your MEL instance
          </CardDescription>
        </CardHeader>
        <CardContent>
          {nodes.length === 0 ? (
            <EmptyState
              icon={<Radio className="h-10 w-10" />}
              title="No nodes observed yet"
              description="Node inventory is empty because no live mesh observations have been stored yet. This is expected when transports are idle or disconnected."
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b text-left">
                    <th className="pb-3 text-sm font-medium text-muted-foreground">Node</th>
                    <th className="pb-3 text-sm font-medium text-muted-foreground">ID</th>
                    <th className="pb-3 text-sm font-medium text-muted-foreground">Last Seen</th>
                    <th className="pb-3 text-sm font-medium text-muted-foreground">Gateway</th>
                  </tr>
                </thead>
                <tbody>
                  {nodes.map((node) => (
                    <NodeRow key={node.id} node={node} />
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

function NodeRow({ node }: { node: NodeInfo }) {
  return (
    <tr className="border-b last:border-0 hover:bg-muted/50">
      <td className="py-4">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-secondary">
            <Radio className="h-4 w-4 text-secondary-foreground" />
          </div>
          <div>
            <p className="font-medium">{node.long_name || 'Unknown'}</p>
            <p className="text-xs text-muted-foreground">{node.short_name || '—'}</p>
          </div>
        </div>
      </td>
      <td className="py-4">
        <code className="rounded bg-muted px-2 py-1 text-sm font-mono">
          {node.id}
        </code>
      </td>
      <td className="py-4">
        <div className="flex items-center gap-2 text-sm">
          <Clock className="h-4 w-4 text-muted-foreground" />
          {formatRelativeTime(node.last_seen)}
        </div>
      </td>
      <td className="py-4">
        <div className="flex items-center gap-2 text-sm">
          <Radio className="h-4 w-4 text-muted-foreground" />
          <code className="font-mono text-xs">{node.gateway_id || '—'}</code>
        </div>
      </td>
    </tr>
  )
}
