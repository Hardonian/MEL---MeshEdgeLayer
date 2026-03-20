
import { useControlStatus, useControlHistory } from '@/hooks/useApi'
import { PageHeader } from '@/components/ui/PageHeader'
import { Loading } from '@/components/ui/StateViews'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'

export function Control() {
  const { data: status, loading: statusLoading, error: statusError } = useControlStatus()
  const { data: history, loading: historyLoading, error: historyError } = useControlHistory()

  if ((statusLoading && !status) || (historyLoading && !history)) {
    return <Loading message="Loading control data..." />
  }

  if (statusError || historyError) {
    return <div>Error: {statusError || historyError}</div>
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Control Plane"
        description="Monitor and manage control actions."
      />

      <Card>
        <CardHeader>
          <CardTitle>Control Status</CardTitle>
          <CardDescription>
            Current status of the control plane.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p>Mode: {status?.mode}</p>
          <p>Status: {status?.status}</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Action History</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {history?.actions.map((action: any) => (
              <div key={action.id} className="rounded-lg border p-4">
                <div className="flex items-center justify-between">
                  <span className="font-semibold">{action.command}</span>
                  <Badge variant={action.result === 'executed_successfully' ? 'success' : 'destructive'}>
                    {action.result}
                  </Badge>
                </div>
                <div className="text-sm text-muted-foreground">
                  <p>Transport: {action.transport_name}</p>
                  <p>Target: {action.target_node_id}</p>
                  <p>Created: {new Date(action.created_at).toLocaleString()}</p>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
