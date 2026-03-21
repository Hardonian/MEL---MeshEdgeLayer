import { useControlStatus, useControlHistory } from '@/hooks/useApi'
import { PageHeader } from '@/components/ui/PageHeader'
import { Loading } from '@/components/ui/StateViews'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { OperatorEmptyState } from '@/components/states/OperatorEmptyState'
import { safeDenialReason, formatOperatorTime } from '@/utils/apiResilience'
import { ShieldAlert, CheckCircle2, XCircle, Clock, Send } from 'lucide-react'

interface ControlAction {
  id: string;
  command: string;
  target_node_id?: string;
  transport_name?: string;
  result: string;
  denial_reason?: string;
  created_at?: string;
}

export function Control() {
  const { data: status, loading: statusLoading, error: statusError } = useControlStatus()
  const { data: history, loading: historyLoading, error: historyError } = useControlHistory()

  if ((statusLoading && !status) || (historyLoading && !history)) {
    return <Loading message="Loading control data..." />
  }

  if (statusError || historyError) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to load control data"
          description={statusError || historyError || 'Unknown error'}
        />
      </div>
    )
  }

  const actions: ControlAction[] = Array.isArray(history?.actions) ? history.actions : []

  return (
    <div className="space-y-6">
      <PageHeader
        title="Control Plane"
        description="Monitor and manage control actions sent to the mesh."
      />

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <CardTitle>Control Status</CardTitle>
            <Badge variant={status?.mode === 'active' ? 'success' : 'warning'}>
              {status?.mode || 'unknown'}
            </Badge>
          </div>
          <CardDescription>
            Current configuration state of the control plane.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center text-sm">
            <span className="font-medium mr-2">Operational Status:</span>
            <span className="text-muted-foreground">{status?.status || 'No status reported'}</span>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Action History</CardTitle>
          <CardDescription>Recent outbound commands and their outcomes.</CardDescription>
        </CardHeader>
        <CardContent>
          {actions.length === 0 ? (
            <OperatorEmptyState 
              title="No control actions" 
              description="No commands have been issued or recorded." 
            />
          ) : (
            <div className="space-y-4">
              {actions.map((action) => {
                const isExecuted = action.result === 'executed_successfully'
                const isBlocked = action.result === 'blocked' || action.result === 'denied'
                const isFailed = action.result === 'failed' || action.result === 'error'
                const isPending = action.result === 'requested' || action.result === 'pending'
                
                let icon = <Send className="h-4 w-4 text-muted-foreground" />
                let badgeVariant: 'success' | 'critical' | 'warning' | 'default' = 'default'
                let badgeText = action.result || 'Unknown'

                if (isExecuted) {
                  icon = <CheckCircle2 className="h-4 w-4 text-success" />
                  badgeVariant = 'success'
                  badgeText = 'Executed'
                } else if (isBlocked) {
                  icon = <ShieldAlert className="h-4 w-4 text-warning" />
                  badgeVariant = 'warning'
                  badgeText = 'Blocked'
                } else if (isFailed) {
                  icon = <XCircle className="h-4 w-4 text-critical" />
                  badgeVariant = 'critical'
                  badgeText = 'Failed'
                } else if (isPending) {
                  icon = <Clock className="h-4 w-4 text-info" />
                  badgeVariant = 'info' as any
                  badgeText = 'Pending'
                }

                return (
                  <div key={action.id} className="rounded-lg border p-4 hover:bg-muted/30 transition-colors">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        {icon}
                        <span className="font-semibold">{action.command || 'Unknown Command'}</span>
                        <code className="text-xs px-1.5 py-0.5 bg-muted rounded">
                          {action.id.slice(0, 8)}
                        </code>
                      </div>
                      <Badge variant={badgeVariant}>{badgeText}</Badge>
                    </div>
                    
                    <div className="grid grid-cols-2 gap-2 text-sm text-muted-foreground mt-3 bg-white dark:bg-black/10 p-3 rounded-md border border-border/50">
                      <div>
                        <span className="font-medium mr-1 text-foreground">Target:</span>
                        {action.target_node_id || 'Global/Broadcast'}
                      </div>
                      <div>
                        <span className="font-medium mr-1 text-foreground">Transport:</span>
                        {action.transport_name || 'System default'}
                      </div>
                      <div>
                        <span className="font-medium mr-1 text-foreground">Time:</span>
                        {formatOperatorTime(action.created_at)}
                      </div>
                      {isBlocked && (
                        <div>
                          <span className="font-medium mr-1 text-red-600">Reason:</span>
                          <span className="text-red-600">{safeDenialReason(action.denial_reason)}</span>
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
