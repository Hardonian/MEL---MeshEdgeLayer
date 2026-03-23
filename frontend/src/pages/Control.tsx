import { useControlStatus, useControlHistory, useOperationalState } from '@/hooks/useApi'
import { PageHeader } from '@/components/ui/PageHeader'
import { Loading } from '@/components/ui/StateViews'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { OperatorEmptyState } from '@/components/states/OperatorEmptyState'
import { safeDenialReason, formatOperatorTime } from '@/utils/apiResilience'
import { ShieldAlert, CheckCircle2, XCircle, Clock, Send, ShieldCheck, Zap, Info, Lock } from 'lucide-react'

interface ControlAction {
  id: string;
  command?: string;
  action_type?: string;
  target_node_id?: string;
  target_transport?: string;
  transport_name?: string;
  result: string;
  denial_reason?: string;
  created_at?: string;
  outcome_detail?: string;
  advisory_only?: boolean;
  lifecycle_state?: string;
  execution_mode?: string;
  proposed_by?: string;
  approved_by?: string;
  evidence_bundle_id?: string;
}

interface RealityMatrixItem {
  action_type: string;
  actuator_exists: boolean;
  reversible: boolean;
  blast_radius_class: string;
  safe_for_guarded_auto: boolean;
  advisory_only: boolean;
  notes: string;
}

export function Control() {
  const { data: status, loading: statusLoading, error: statusError } = useControlStatus()
  const { data: history, loading: historyLoading, error: historyError } = useControlHistory()
  const { data: opState, loading: opLoading, error: opError } = useOperationalState()

  if ((statusLoading && !status) || (historyLoading && !history) || (opLoading && !opState)) {
    return <Loading message="Syncing with control plane..." />
  }

  if (statusError || historyError || opError) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Control Plane Desync"
          description={statusError || historyError || opError || 'Unexpected error communicating with control plane.'}
        />
      </div>
    )
  }

  const actions: ControlAction[] = Array.isArray(history?.actions) ? history.actions : []
  const realityMatrix: RealityMatrixItem[] = Array.isArray(status?.reality_matrix) ? status.reality_matrix : []
  const pendingApprovals: ControlAction[] = Array.isArray(opState?.pending_approvals) ? opState.pending_approvals : []

  return (
    <div className="space-y-6 pb-12">
      <PageHeader
        title="Control Plane"
        description="Truth-driven mesh remediation and guarded automation."
      />

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card className="md:col-span-2">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <CardTitle>Operational Mode</CardTitle>
                <Badge variant={status?.mode === 'guarded_auto' ? 'success' : status?.mode === 'advisory' ? 'warning' : 'default'}>
                  {status?.mode || 'disabled'}
                </Badge>
              </div>
              <ShieldCheck className={`h-5 w-5 ${status?.mode === 'guarded_auto' ? 'text-success' : 'text-muted-foreground'}`} />
            </div>
            <CardDescription>
              {status?.mode === 'guarded_auto' 
                ? 'MEL is monitoring mesh health and will automatically execute safe remediation actions.'
                : status?.mode === 'advisory'
                ? 'Control actions are evaluated and suggested, but never executed. Requires operator intervention.'
                : 'Control processes are dormant. Enable in mel.json to activate guardrails.'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="p-3 bg-muted/30 rounded-lg border border-border/50">
                  <div className="text-xs font-semibold text-muted-foreground uppercase mb-1">Queue Depth</div>
                  <div className="text-2xl font-bold font-mono">{status?.queue_depth || 0} / {status?.queue_capacity || 0}</div>
                </div>
                <div className="p-3 bg-muted/30 rounded-lg border border-border/50">
                  <div className="text-xs font-semibold text-muted-foreground uppercase mb-1">Active Actions</div>
                  <div className="text-2xl font-bold font-mono">{status?.active_actions || 0}</div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm border-b pb-2">Guarded Constraints</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4 pt-2">
            <div className="flex items-start gap-3 text-sm">
              <Lock className="h-4 w-4 mt-0.5 text-muted-foreground" />
              <div>
                <span className="font-medium block">Adherence</span>
                <span className="text-muted-foreground text-xs">{status?.policy_summary || 'Default policies enforced.'}</span>
              </div>
            </div>
            <div className="flex items-start gap-3 text-sm">
              <Zap className="h-4 w-4 mt-0.5 text-muted-foreground" />
              <div>
                <span className="font-medium block">Emergency Stop</span>
                <span className={status?.emergency_disable ? 'text-critical font-bold text-xs' : 'text-success text-xs'}>
                  {status?.emergency_disable ? 'ACTIVE (Automation Terminated)' : 'Inactive'}
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {pendingApprovals.length > 0 && (
        <Card className="border-warning/40">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-warning" />
              Pending approvals
            </CardTitle>
            <CardDescription>
              These actions require an operator approval before execution. Use{' '}
              <code className="text-xs bg-muted px-1 rounded">POST /api/v1/actions/&#123;id&#125;/approve</code>
              {' '}or <code className="text-xs bg-muted px-1 rounded">mel action approve</code> with the same config as the running daemon.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {pendingApprovals.map((pa) => (
              <div key={pa.id} className="rounded-lg border border-border/60 p-3 text-sm">
                <div className="flex flex-wrap items-center gap-2 mb-1">
                  <span className="font-mono text-xs font-semibold">{pa.action_type || 'action'}</span>
                  <code className="text-[10px] px-1.5 py-0.5 bg-muted rounded">{pa.id}</code>
                  <Badge variant="warning" className="text-[10px]">awaiting_review</Badge>
                </div>
                <div className="text-xs text-muted-foreground space-y-0.5">
                  <div><span className="text-foreground font-medium">Proposed by:</span> {pa.proposed_by || 'system'}</div>
                  <div><span className="text-foreground font-medium">Transport:</span> {pa.target_transport || pa.transport_name || '—'}</div>
                  {pa.evidence_bundle_id ? (
                    <div><span className="text-foreground font-medium">Evidence bundle:</span> {pa.evidence_bundle_id}</div>
                  ) : null}
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {realityMatrix.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Reality Matrix</CardTitle>
            <CardDescription>Supported actuators and their safety profiles.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-muted-foreground">
                    <th className="text-left font-medium py-2 px-4">Action</th>
                    <th className="text-left font-medium py-2 px-4">Actuator</th>
                    <th className="text-left font-medium py-2 px-4">Reversible</th>
                    <th className="text-left font-medium py-2 px-4">Blast Radius</th>
                    <th className="text-left font-medium py-2 px-4">Automation</th>
                  </tr>
                </thead>
                <tbody>
                  {realityMatrix.map((item) => (
                    <tr key={item.action_type} className="border-b border-border/50 hover:bg-muted/10 transition-colors">
                      <td className="py-3 px-4 font-mono text-xs">{item.action_type}</td>
                      <td className="py-3 px-4">
                        {item.actuator_exists ? (
                          <Badge variant="outline" className="text-success border-success/30 bg-success/5">Ready</Badge>
                        ) : (
                          <Badge variant="outline" className="text-muted-foreground bg-muted/10">Advisory Only</Badge>
                        )}
                      </td>
                      <td className="py-3 px-4 text-xs">{item.reversible ? 'Yes' : 'No'}</td>
                      <td className="py-3 px-4">
                        <span className="capitalize text-xs p-1 bg-muted rounded">{item.blast_radius_class}</span>
                      </td>
                      <td className="py-3 px-4">
                        {item.safe_for_guarded_auto ? (
                          <div className="flex items-center gap-1 text-success text-xs">
                            <CheckCircle2 className="h-3 w-3" /> Enabled
                          </div>
                        ) : (
                          <div className="flex items-center gap-1 text-muted-foreground text-xs">
                            <Lock className="h-3 w-3" /> Manual Only
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Action History</CardTitle>
          <CardDescription>Recent outbound commands and their outcomes.</CardDescription>
        </CardHeader>
        <CardContent>
          {actions.length === 0 ? (
            <OperatorEmptyState 
              title="No control history" 
              description="No automated or manual commands have been recorded in the database." 
            />
          ) : (
            <div className="space-y-4">
              {actions.map((action) => {
                const isExecuted = action.result === 'executed_successfully'
                const isBlocked = action.result === 'blocked' || action.result === 'denied' || action.result === 'denied_by_policy' || action.result === 'denied_by_cooldown'
                const isFailed = action.result === 'failed' || action.result === 'error' || action.result === 'failed_terminal'
                const isPending =
                  action.result === 'requested' ||
                  action.result === 'pending' ||
                  action.result === 'pending_approval' ||
                  action.lifecycle_state === 'pending_approval'
                
                let icon = <Send className="h-4 w-4 text-muted-foreground" />
                let badgeVariant: 'success' | 'critical' | 'warning' | 'default' = 'default'
                let badgeText = action.result?.replace(/_/g, ' ') || 'Unknown'

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
                  icon = <Clock className="h-4 w-4 text-muted-foreground" />
                  badgeVariant = 'secondary' as any
                  badgeText = 'Pending'
                }

                return (
                  <div key={action.id} className="rounded-lg border p-4 hover:bg-muted/30 transition-colors">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        {icon}
                        <span className="font-semibold">{action.action_type || action.command || 'Unknown'}</span>
                        <code className="text-[10px] px-1.5 py-0.5 bg-muted rounded text-muted-foreground">
                          {action.id.slice(0, 8)}
                        </code>
                        {action.advisory_only && (
                          <Badge variant="outline" className="text-[10px] h-4 font-normal">Advisory</Badge>
                        )}
                        {action.execution_mode === 'approval_required' && (
                          <Badge variant="outline" className="text-[10px] h-4 font-normal border-warning/40">approval_required</Badge>
                        )}
                      </div>
                      <Badge variant={badgeVariant} className="capitalize text-[10px]">{badgeText}</Badge>
                    </div>
                    
                    <div className="mt-3 bg-muted/20 dark:bg-black/20 p-3 rounded-md border border-border/50">
                       <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-y-2 gap-x-6 text-xs text-muted-foreground">
                        <div>
                          <span className="font-medium mr-1 text-foreground">Target:</span>
                          <span className="font-mono">{action.target_transport || action.transport_name || 'Global'}</span>
                        </div>
                        <div>
                          <span className="font-medium mr-1 text-foreground">Time:</span>
                          {formatOperatorTime(action.created_at)}
                        </div>
                        {(action.proposed_by || action.approved_by) && (
                          <div className="md:col-span-2">
                            <span className="font-medium mr-1 text-foreground">Attribution:</span>
                            <span className="text-muted-foreground">
                              proposed {action.proposed_by || '—'}
                              {action.approved_by ? ` · approved ${action.approved_by}` : ''}
                            </span>
                          </div>
                        )}
                        <div className="md:col-span-2 lg:col-span-1">
                          <span className="font-medium mr-1 text-foreground">Detail:</span>
                          <span className="italic">{action.outcome_detail || 'No outcome detail reported'}</span>
                        </div>
                      </div>
                      
                      {(isBlocked || action.denial_reason) && (
                        <div className="mt-2 pt-2 border-t border-border/50 flex items-start gap-2">
                          <Info className="h-3.5 w-3.5 mt-0.5 text-warning" />
                          <div className="text-xs">
                            <span className="font-semibold text-foreground mr-1">Denial Reason:</span>
                            <span className="text-muted-foreground">{safeDenialReason(action.denial_reason)}</span>
                          </div>
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
