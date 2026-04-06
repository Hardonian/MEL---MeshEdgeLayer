import { useControlStatus, useControlHistory, useOperationalState } from '@/hooks/useApi'
import { useOperatorContext } from '@/hooks/useOperatorContext'
import { PageHeader } from '@/components/ui/PageHeader'
import { MelPanelInset, MelTruthBadge } from '@/components/ui/operator'
import { Loading } from '@/components/ui/StateViews'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { OperatorEmptyState } from '@/components/states/OperatorEmptyState'
import { safeDenialReason, formatOperatorTime } from '@/utils/apiResilience'
import type { MeshNodeControlAction } from '@/types/api'
import { ShieldAlert, CheckCircle2, XCircle, Clock, Send, ShieldCheck, Zap, Info, Lock } from 'lucide-react'

function str(v: unknown): string {
  return typeof v === 'string' ? v : ''
}

function automationModeTruthSemantic(mode: string | undefined): string {
  if (mode === 'frozen') return 'frozen'
  if (mode === 'maintenance') return 'degraded'
  if (mode === 'normal') return 'live'
  return 'unknown'
}

function incidentFromAction(a: MeshNodeControlAction): string {
  const ov = a.operator_view
  const lid = ov?.linked_incident_id
  if (typeof lid === 'string' && lid.trim()) return lid.trim()
  const d = a.details
  if (!d) return ''
  for (const k of ['incident_id', 'mel_incident_id', 'linked_incident_id']) {
    const x = d[k]
    if (typeof x === 'string' && x.trim()) return x.trim()
  }
  return ''
}

export function Control() {
  const { data: status, loading: statusLoading, error: statusError } = useControlStatus()
  const { data: history, loading: historyLoading, error: historyError } = useControlHistory()
  const { data: opState, loading: opLoading, error: opError } = useOperationalState()
  const { trustUI } = useOperatorContext()
  const canApprove = trustUI?.approve_control === true

  if ((statusLoading && !status) || (historyLoading && !history) || (opLoading && !opState)) {
    return <Loading message="Loading mesh action queue…" />
  }

  if (statusError || historyError || opError) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Mesh / node actions — API error"
          description={statusError || historyError || opError || 'Unexpected error communicating with the backend.'}
        />
      </div>
    )
  }

  const actions = history?.actions ?? []
  const realityMatrix = status?.reality_matrix ?? []
  const pendingApprovals = opState?.pending_approvals ?? []
  const queueMetrics = opState?.queue_metrics
  const executor = opState?.executor
  const queueDepth = queueMetrics?.approved_waiting_executor_count ?? null
  const queueCapacity = queueMetrics?.queued_lifecycle_pending_count ?? null
  const awaitingApproval = queueMetrics?.awaiting_approval_count ?? null
  const executorState = executor?.executor_activity
  const executorLastSeen = str(executor?.executor_last_heartbeat_at)
  const snapshotAt = str(opState?.snapshot_at)

  return (
    <div className="space-y-6 pb-12">
      <PageHeader
        title="Mesh / node actions"
        description="Guarded automation against transports, MQTT bridges, and node targets. Refresh the page to reload this view — it is not a live stream."
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
                ? 'MEL watches link / transport health and can run safe remediation on your behalf when policy allows.'
                : status?.mode === 'advisory'
                ? 'Mesh-side actions are evaluated and surfaced, but not executed automatically — operators drive changes.'
                : 'Automation is off. Enable guarded mode in mel.json to activate mesh-aware guardrails.'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <MelPanelInset className="p-3">
                  <div className="text-xs font-semibold text-muted-foreground uppercase mb-1">Executor queue depth</div>
                  <div className="text-2xl font-bold font-mono">{status?.queue_depth || 0} / {status?.queue_capacity || 0}</div>
                </MelPanelInset>
                <MelPanelInset className="p-3">
                  <div className="text-xs font-semibold text-muted-foreground uppercase mb-1">In-flight actions</div>
                  <div className="text-2xl font-bold font-mono">{status?.active_actions || 0}</div>
                </MelPanelInset>
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
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">Automation posture</CardTitle>
          </CardHeader>
          <CardContent>
            <MelTruthBadge semantic={automationModeTruthSemantic(opState?.automation_mode)}>
              {opState?.automation_mode || 'unknown'}
            </MelTruthBadge>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">Active freezes</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-mono font-semibold">{opState?.freeze_count ?? '—'}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">Approval backlog</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-mono font-semibold">{opState?.approval_backlog ?? pendingApprovals.length}</div>
            {opState?.approval_backlog !== undefined && opState.approval_backlog !== pendingApprovals.length && (
              <p className="text-mel-sm text-warning mt-1">
                Backlog snapshot differs from visible rows ({pendingApprovals.length}) due to API filtering/windowing.
              </p>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">Executor presence</CardTitle>
          </CardHeader>
          <CardContent>
            <div className={`text-sm font-medium ${executorState === 'inactive' ? 'text-warning' : ''}`}>
              {executorState === 'active' ? 'active' : executorState === 'inactive' ? 'inactive/stale' : 'unknown'}
            </div>
            {(queueDepth !== null || queueCapacity !== null) && (
              <p className="text-mel-sm text-muted-foreground mt-1">
                Approved waiting executor: {queueDepth ?? '—'} · Pending (non-approved): {queueCapacity ?? '—'}
              </p>
            )}
            {awaitingApproval !== null && (
              <p className="text-mel-sm text-muted-foreground mt-1">
                Awaiting second-operator approval: {awaitingApproval}
              </p>
            )}
            {(executorLastSeen || snapshotAt) && (
              <p className="text-mel-sm text-muted-foreground mt-1">
                {executorLastSeen ? `Executor heartbeat: ${formatOperatorTime(executorLastSeen)}` : ''}
                {executorLastSeen && snapshotAt ? ' · ' : ''}
                {snapshotAt ? `Snapshot: ${formatOperatorTime(snapshotAt)}` : ''}
              </p>
            )}
            {executor?.executor_presence_note ? (
              <p className="text-mel-sm text-muted-foreground mt-1">{executor.executor_presence_note}</p>
            ) : null}
          </CardContent>
        </Card>
      </div>

      {pendingApprovals.length > 0 && (
        <Card className="border-warning/40">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-warning" />
              Needs second-operator approval
            </CardTitle>
            <CardDescription>
              These mesh / node actions are held until a different operator approves them (separation of duties when a human proposed the action).
              {canApprove ? (
                <>
                  {' '}
                  Approve via{' '}
                  <code className="text-xs bg-muted px-1 rounded">POST /api/v1/actions/&#123;id&#125;/approve</code>
                  {' '}or <code className="text-xs bg-muted px-1 rounded">mel action approve</code> using the same config as <code className="text-xs bg-muted px-1 rounded">mel serve</code>.
                  {' '}Avoid <code className="text-xs bg-muted px-1 rounded">mel control approve</code> except documented break-glass.
                </>
              ) : (
                <>
                  {' '}
                  Your session does not include <code className="text-xs bg-muted px-1 rounded">approve_control_action</code>; you can read the queue but not approve with this API identity.
                </>
              )}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {pendingApprovals.map((pa) => {
              const ov = pa.operator_view
              const targetLine = str(ov?.target_summary) || [pa.target_node, pa.target_transport || pa.transport_name].filter(Boolean).join(' · ') || '—'
              const sod = ov?.sod_blocks_self === true
              return (
              <MelPanelInset key={pa.id} className="p-3 text-sm">
                <div className="flex flex-wrap items-center gap-2 mb-1">
                  <span className="font-mono text-xs font-semibold">{pa.action_type || 'mesh action'}</span>
                  <code className="text-mel-xs px-1.5 py-0.5 bg-muted rounded">{pa.id}</code>
                  <Badge variant="warning" className="text-mel-xs">{str(ov?.approval_status) || 'awaiting approver'}</Badge>
                </div>
                <div className="text-xs text-muted-foreground space-y-0.5">
                  <div><span className="text-foreground font-medium">Target summary:</span> {targetLine}</div>
                  <div><span className="text-foreground font-medium">Proposed by:</span> {pa.proposed_by || 'system'}</div>
                  {str(ov?.second_operator_note) ? (
                    <div className="text-warning">{str(ov?.second_operator_note)}</div>
                  ) : null}
                  {sod ? (
                    <div className="text-muted-foreground">Self-approval blocked for this row — use another operator id.</div>
                  ) : null}
                  {incidentFromAction(pa) ? (
                    <div><span className="text-foreground font-medium">Linked incident:</span> <code className="text-mel-xs bg-muted px-1 rounded">{incidentFromAction(pa)}</code></div>
                  ) : null}
                  {pa.evidence_bundle_id ? (
                    <div><span className="text-foreground font-medium">Evidence bundle:</span> {pa.evidence_bundle_id}</div>
                  ) : null}
                </div>
              </MelPanelInset>
            )})}
          </CardContent>
        </Card>
      )}

      {realityMatrix.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Action capability matrix</CardTitle>
            <CardDescription>What MEL can actually drive on-node or on-link vs advisory-only paths.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-muted-foreground">
                    <th className="text-left font-medium py-2 px-4">Node / link action</th>
                    <th className="text-left font-medium py-2 px-4">Actuator</th>
                    <th className="text-left font-medium py-2 px-4">Reversible</th>
                    <th className="text-left font-medium py-2 px-4">Impact scope</th>
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
          <CardTitle>Recent mesh / node actions</CardTitle>
          <CardDescription>Approval state and executor state are distinct: &quot;approved&quot; does not mean the change already ran on the target.</CardDescription>
        </CardHeader>
        <CardContent>
          {actions.length === 0 ? (
            <OperatorEmptyState 
              title="No mesh / node action history" 
              description="No automated or operator-driven actions are recorded in the database yet." 
            />
          ) : (
            <div className="space-y-4">
              {actions.map((action) => {
                const ov = action.operator_view
                const execLabel = str(ov?.execution_status)
                const isExecuted = action.result === 'executed_successfully' || execLabel.toLowerCase().includes('completed') || execLabel.toLowerCase().includes('finished')
                const isBlocked = action.result === 'blocked' || action.result === 'denied' || action.result === 'denied_by_policy' || action.result === 'denied_by_cooldown'
                const isFailed = action.result === 'failed' || action.result === 'error' || action.result === 'failed_terminal' || execLabel.toLowerCase() === 'failed'
                const isPendingApproval =
                  action.result === 'pending_approval' ||
                  action.lifecycle_state === 'pending_approval'
                const approvedWaiting =
                  action.result === 'approved' &&
                  action.lifecycle_state === 'pending' &&
                  !action.executed_at
                
                let icon = <Send className="h-4 w-4 text-muted-foreground" />
                let badgeVariant: 'success' | 'critical' | 'warning' | 'default' | 'secondary' = 'default'
                let badgeText = execLabel || action.result?.replace(/_/g, ' ') || 'Unknown'

                if (isExecuted) {
                  icon = <CheckCircle2 className="h-4 w-4 text-success" />
                  badgeVariant = 'success'
                  badgeText = execLabel || 'Completed on target'
                } else if (isBlocked) {
                  icon = <ShieldAlert className="h-4 w-4 text-warning" />
                  badgeVariant = 'warning'
                  badgeText = execLabel || 'Blocked'
                } else if (isFailed) {
                  icon = <XCircle className="h-4 w-4 text-critical" />
                  badgeVariant = 'critical'
                  badgeText = execLabel || 'Failed'
                } else if (isPendingApproval) {
                  icon = <Clock className="h-4 w-4 text-muted-foreground" />
                  badgeVariant = 'secondary'
                  badgeText = str(ov?.approval_status) || 'Needs approver'
                } else if (approvedWaiting) {
                  icon = <Clock className="h-4 w-4 text-warning" />
                  badgeVariant = 'warning'
                  badgeText = execLabel || 'Approved — waiting executor'
                } else if (
                  action.result === 'requested' ||
                  action.result === 'pending' ||
                  action.lifecycle_state === 'pending'
                ) {
                  icon = <Clock className="h-4 w-4 text-muted-foreground" />
                  badgeVariant = 'secondary'
                  badgeText = execLabel || 'Queued / running'
                }

                const targetLine =
                  str(ov?.target_summary) ||
                  [action.target_node, action.target_transport || action.transport_name].filter(Boolean).join(' · ') ||
                  '—'
                const breakGlass = ov?.break_glass_in_history === true
                const incId = incidentFromAction(action)

                return (
                  <div key={action.id} className="rounded-sm border p-4 hover:bg-muted/30 transition-colors">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        {icon}
                        <span className="font-semibold">{action.action_type || action.command || 'mesh action'}</span>
                        <code className="text-mel-xs px-1.5 py-0.5 bg-muted rounded text-muted-foreground">
                          {action.id.slice(0, 8)}
                        </code>
                        {action.advisory_only && (
                          <Badge variant="outline" className="text-mel-xs h-4 font-normal">Advisory</Badge>
                        )}
                        {action.execution_mode === 'approval_required' && (
                          <Badge variant="outline" className="text-mel-xs h-4 font-normal border-warning/40">second-operator gate</Badge>
                        )}
                        {breakGlass && (
                          <Badge variant="outline" className="text-mel-xs h-4 font-normal border-critical/40 text-critical">break-glass</Badge>
                        )}
                      </div>
                      <Badge variant={badgeVariant} className="capitalize text-mel-xs">{badgeText}</Badge>
                    </div>
                    
                    <div className="mt-3 bg-muted/20 dark:bg-black/20 p-3 rounded-md border border-border/50">
                       <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-y-2 gap-x-6 text-xs text-muted-foreground">
                        {ov ? (
                          <>
                            <div>
                              <span className="font-medium mr-1 text-foreground">Queue:</span>
                              {str(ov.queue_status)}
                            </div>
                            <div>
                              <span className="font-medium mr-1 text-foreground">Approval:</span>
                              {str(ov.approval_status)}
                            </div>
                            <div>
                              <span className="font-medium mr-1 text-foreground">Execution:</span>
                              {str(ov.execution_status)}
                            </div>
                          </>
                        ) : null}
                        <div>
                          <span className="font-medium mr-1 text-foreground">Target summary:</span>
                          <span>{targetLine}</span>
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
                        {incId ? (
                          <div className="md:col-span-2">
                            <span className="font-medium mr-1 text-foreground">Linked incident:</span>
                            <code className="text-mel-xs bg-muted px-1 rounded">{incId}</code>
                          </div>
                        ) : null}
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
